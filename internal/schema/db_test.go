// Copyright (C) 2026 Thorben Stangenberg
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package schema_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/graphql-go/graphql"
	"github.com/tstangenberg/stratum/internal/plugin"
	eqfilter "github.com/tstangenberg/stratum/internal/plugin/filter/eq"
	"github.com/tstangenberg/stratum/internal/plugin/pagination/simple"
	"github.com/tstangenberg/stratum/internal/plugin/scalar"
	idscalar "github.com/tstangenberg/stratum/internal/plugin/scalar/id"
	intscalar "github.com/tstangenberg/stratum/internal/plugin/scalar/int"
	stringscalar "github.com/tstangenberg/stratum/internal/plugin/scalar/string"
	"github.com/tstangenberg/stratum/internal/schema"
)

func startPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	pgc, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = pgc.Terminate(ctx) })

	dsn, err := pgc.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func schemaScalars() map[string]scalar.Plugin {
	return map[string]scalar.Plugin{
		"String": stringscalar.Plugin{},
		"ID":     idscalar.Plugin{},
	}
}

func locationTypeDef() schema.TypeDef {
	return schema.TypeDef{
		Name: "Location",
		Fields: []schema.FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
			{Name: "name", Type: "String", NonNull: true},
		},
	}
}

func TestCreateTable_Success(t *testing.T) {
	pool := startPool(t)
	ctx := context.Background()

	err := schema.CreateTable(ctx, pool, "test", locationTypeDef(), schemaScalars())
	if err != nil {
		t.Fatalf("CreateTable: %v", err)
	}

	var tbl string
	err = pool.QueryRow(ctx,
		"SELECT tablename FROM pg_tables WHERE tablename = 'test_location'",
	).Scan(&tbl)
	if err != nil {
		t.Fatalf("table not found: %v", err)
	}
}

func TestCreateTable_UnknownScalar(t *testing.T) {
	td := schema.TypeDef{
		Name: "Widget",
		Fields: []schema.FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
			{Name: "price", Type: "Float", NonNull: true},
		},
	}
	// No DB needed — error occurs before any Exec call
	err := schema.CreateTable(context.Background(), nil, "test", td, map[string]scalar.Plugin{})
	if err == nil {
		t.Fatal("expected error for unknown scalar type")
	}
}

func TestCreateTable_NullableField(t *testing.T) {
	pool := startPool(t)
	ctx := context.Background()

	td := schema.TypeDef{
		Name: "Widget",
		Fields: []schema.FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
			{Name: "description", Type: "String", NonNull: false},
		},
	}
	if err := schema.CreateTable(ctx, pool, "test", td, schemaScalars()); err != nil {
		t.Fatalf("CreateTable with nullable field: %v", err)
	}
}

func TestCreateTable_Idempotent(t *testing.T) {
	pool := startPool(t)
	ctx := context.Background()

	td := locationTypeDef()
	scalars := schemaScalars()
	if err := schema.CreateTable(ctx, pool, "test", td, scalars); err != nil {
		t.Fatalf("first CreateTable: %v", err)
	}
	if err := schema.CreateTable(ctx, pool, "test", td, scalars); err != nil {
		t.Fatalf("second CreateTable (IF NOT EXISTS should be idempotent): %v", err)
	}
}

func TestCreateTable_DBError(t *testing.T) {
	pool := startPool(t)
	pool.Close()

	err := schema.CreateTable(context.Background(), pool, "test", locationTypeDef(), schemaScalars())
	if err == nil {
		t.Fatal("expected error when pool is closed")
	}
}

func TestGraphQLResolvers(t *testing.T) {
	pool := startPool(t)
	ctx := context.Background()

	td := locationTypeDef()
	scalars := schemaScalars()

	if err := schema.CreateTable(ctx, pool, "test", td, scalars); err != nil {
		t.Fatalf("CreateTable: %v", err)
	}

	ps := &schema.ParsedSchema{Types: []schema.TypeDef{td}}
	h, err := schema.BuildHandler(pool, "test", ps, scalars, []plugin.QueryModifier{simple.New()}, nil)
	if err != nil {
		t.Fatalf("BuildHandler: %v", err)
	}

	do := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/graphql/test", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}

	// list empty
	listW := do(`{"query":"{ location { list { id name } } }"}`)
	if listW.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d — %s", listW.Code, listW.Body.String())
	}

	// create
	createW := do(`{"query":"mutation { location { create(input: {name: \"Berlin\"}) { id name } } }"}`)
	if createW.Code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d — %s", createW.Code, createW.Body.String())
	}
	var createResult struct {
		Data struct {
			Location struct {
				Create struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"create"`
			} `json:"location"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(createW.Body).Decode(&createResult); err != nil {
		t.Fatalf("create decode: %v", err)
	}
	if len(createResult.Errors) > 0 {
		t.Fatalf("create errors: %v", createResult.Errors)
	}
	createdID := createResult.Data.Location.Create.ID
	if createdID == "" {
		t.Fatal("expected non-empty id")
	}

	// list non-empty (covers rows.Next() body in listRecords)
	listW2 := do(`{"query":"{ location { list { id name } } }"}`)
	if listW2.Code != http.StatusOK {
		t.Fatalf("list2: expected 200, got %d — %s", listW2.Code, listW2.Body.String())
	}
	var listResult struct {
		Data struct {
			Location struct {
				List []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"list"`
			} `json:"location"`
		} `json:"data"`
	}
	if err := json.NewDecoder(listW2.Body).Decode(&listResult); err != nil {
		t.Fatalf("list2 decode: %v", err)
	}
	if len(listResult.Data.Location.List) != 1 {
		t.Errorf("list2: expected 1 item, got %d", len(listResult.Data.Location.List))
	}

	// get existing record
	getW := do(`{"query":"{ location { get(id: \"` + createdID + `\") { id name } } }"}`)
	if getW.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d — %s", getW.Code, getW.Body.String())
	}

	// get non-existent record → returns null, no errors
	getNullW := do(`{"query":"{ location { get(id: \"00000000-0000-0000-0000-000000000000\") { id name } } }"}`)
	if getNullW.Code != http.StatusOK {
		t.Fatalf("get-nonexistent: expected 200, got %d — %s", getNullW.Code, getNullW.Body.String())
	}
	var getNullResult struct {
		Data   map[string]any             `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(getNullW.Body).Decode(&getNullResult); err != nil {
		t.Fatalf("get-nonexistent decode: %v", err)
	}
	if len(getNullResult.Errors) > 0 {
		t.Errorf("get-nonexistent: expected no errors, got %v", getNullResult.Errors)
	}
	locNS, ok := getNullResult.Data["location"].(map[string]any)
	if !ok {
		t.Fatal("get-nonexistent: expected location namespace in data")
	}
	if locNS["get"] != nil {
		t.Errorf("get-nonexistent: expected null, got %v", locNS["get"])
	}
}

func TestGraphQLResolvers_NullableField(t *testing.T) {
	pool := startPool(t)
	ctx := context.Background()

	td := schema.TypeDef{
		Name: "Widget",
		Fields: []schema.FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
			{Name: "description", Type: "String", NonNull: false},
		},
	}
	scalars := schemaScalars()

	if err := schema.CreateTable(ctx, pool, "test", td, scalars); err != nil {
		t.Fatalf("CreateTable: %v", err)
	}

	ps := &schema.ParsedSchema{Types: []schema.TypeDef{td}}
	h, err := schema.BuildHandler(pool, "test", ps, scalars, []plugin.QueryModifier{simple.New()}, nil)
	if err != nil {
		t.Fatalf("BuildHandler: %v", err)
	}

	// create without the optional 'description' field (covers createRecord !ok branch)
	req := httptest.NewRequest(http.MethodPost, "/graphql/test",
		strings.NewReader(`{"query":"mutation { widget { create(input: {}) { id } } }"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create-no-desc: expected 200, got %d — %s", w.Code, w.Body.String())
	}
}

func TestCreateTable_WithRelation(t *testing.T) {
	pool := startPool(t)
	ctx := context.Background()

	kantonTD := schema.TypeDef{
		Name: "Kanton",
		Fields: []schema.FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
			{Name: "name", Type: "String", NonNull: true},
		},
	}
	ortTD := schema.TypeDef{
		Name: "Ortschaft",
		Fields: []schema.FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
			{Name: "name", Type: "String", NonNull: true},
			{Name: "kanton", Type: "Kanton", NonNull: true, IsRelation: true},
		},
	}
	scalars := schemaScalars()

	if err := schema.CreateTable(ctx, pool, "test", kantonTD, scalars); err != nil {
		t.Fatalf("CreateTable Kanton: %v", err)
	}
	if err := schema.CreateTable(ctx, pool, "test", ortTD, scalars); err != nil {
		t.Fatalf("CreateTable Ortschaft: %v", err)
	}

	var colName string
	err := pool.QueryRow(ctx,
		`SELECT column_name FROM information_schema.columns
		 WHERE table_name = 'test_ortschaft' AND column_name = 'kanton_id'`).Scan(&colName)
	if err != nil {
		t.Fatalf("expected kanton_id column: %v", err)
	}
}

func TestCreateTable_NullableRelation(t *testing.T) {
	pool := startPool(t)
	ctx := context.Background()

	parentTD := schema.TypeDef{
		Name: "Parent",
		Fields: []schema.FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
		},
	}
	childTD := schema.TypeDef{
		Name: "Child",
		Fields: []schema.FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
			{Name: "parent", Type: "Parent", NonNull: false, IsRelation: true},
		},
	}
	scalars := schemaScalars()

	if err := schema.CreateTable(ctx, pool, "test", parentTD, scalars); err != nil {
		t.Fatalf("CreateTable Parent: %v", err)
	}
	if err := schema.CreateTable(ctx, pool, "test", childTD, scalars); err != nil {
		t.Fatalf("CreateTable Child: %v", err)
	}

	// Insert a child without a parent reference (NULL FK)
	_, err := pool.Exec(ctx, `INSERT INTO test_child (id) VALUES ('c1')`)
	if err != nil {
		t.Fatalf("insert child with null FK: %v", err)
	}
}

func TestGraphQLResolvers_Relation(t *testing.T) {
	pool := startPool(t)
	ctx := context.Background()

	kantonTD := schema.TypeDef{
		Name: "Kanton",
		Fields: []schema.FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
			{Name: "name", Type: "String", NonNull: true},
		},
	}
	ortTD := schema.TypeDef{
		Name: "Ortschaft",
		Fields: []schema.FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
			{Name: "name", Type: "String", NonNull: true},
			{Name: "kanton", Type: "Kanton", NonNull: true, IsRelation: true},
		},
	}
	scalars := schemaScalars()

	if err := schema.CreateTable(ctx, pool, "test", kantonTD, scalars); err != nil {
		t.Fatalf("CreateTable Kanton: %v", err)
	}
	if err := schema.CreateTable(ctx, pool, "test", ortTD, scalars); err != nil {
		t.Fatalf("CreateTable Ortschaft: %v", err)
	}

	ps := &schema.ParsedSchema{Types: []schema.TypeDef{kantonTD, ortTD}}
	h, err := schema.BuildHandler(pool, "test", ps, scalars, []plugin.QueryModifier{simple.New()}, nil)
	if err != nil {
		t.Fatalf("BuildHandler: %v", err)
	}

	do := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/graphql/test", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}

	// Create a Kanton
	createK := do(`{"query":"mutation { kanton { create(input: {name: \"Zürich\"}) { id name } } }"}`)
	var kRes struct {
		Data struct {
			Kanton struct {
				Create struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"create"`
			} `json:"kanton"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(createK.Body).Decode(&kRes); err != nil {
		t.Fatalf("kanton decode: %v", err)
	}
	if len(kRes.Errors) > 0 {
		t.Fatalf("kanton errors: %v", kRes.Errors)
	}
	kantonID := kRes.Data.Kanton.Create.ID

	// Create Ortschaft with relation
	createO := do(`{"query":"mutation { ortschaft { create(input: {name: \"Winterthur\", kantonId: \"` + kantonID + `\"}) { id name kanton { id name } } } }"}`)
	var oRes struct {
		Data struct {
			Ortschaft struct {
				Create struct {
					ID     string `json:"id"`
					Name   string `json:"name"`
					Kanton struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"kanton"`
				} `json:"create"`
			} `json:"ortschaft"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(createO.Body).Decode(&oRes); err != nil {
		t.Fatalf("ortschaft decode: %v", err)
	}
	if len(oRes.Errors) > 0 {
		t.Fatalf("ortschaft errors: %v", oRes.Errors)
	}
	if oRes.Data.Ortschaft.Create.Kanton.ID != kantonID {
		t.Errorf("kanton.id = %q, want %q", oRes.Data.Ortschaft.Create.Kanton.ID, kantonID)
	}

	// Get with relation traversal
	getO := do(`{"query":"{ ortschaft { get(id: \"` + oRes.Data.Ortschaft.Create.ID + `\") { id kanton { id name } } } }"}`)
	var gRes struct {
		Data struct {
			Ortschaft struct {
				Get struct {
					Kanton struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"kanton"`
				} `json:"get"`
			} `json:"ortschaft"`
		} `json:"data"`
	}
	if err := json.NewDecoder(getO.Body).Decode(&gRes); err != nil {
		t.Fatalf("get decode: %v", err)
	}
	if gRes.Data.Ortschaft.Get.Kanton.ID != kantonID {
		t.Errorf("get: kanton.id = %q, want %q", gRes.Data.Ortschaft.Get.Kanton.ID, kantonID)
	}

	// List with relation
	listO := do(`{"query":"{ ortschaft { list { id name kanton { id } } } }"}`)
	if listO.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", listO.Code)
	}

	// Non-existent FK returns error
	badFK := do(`{"query":"mutation { ortschaft { create(input: {name: \"Ghost\", kantonId: \"nonexistent\"}) { id } } }"}`)
	var badRes struct {
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(badFK.Body).Decode(&badRes); err != nil {
		t.Fatalf("bad fk decode: %v", err)
	}
	if len(badRes.Errors) == 0 {
		t.Fatal("expected error for non-existent FK")
	}
}

func TestGraphQLResolvers_NullableRelation(t *testing.T) {
	pool := startPool(t)
	ctx := context.Background()

	parentTD := schema.TypeDef{
		Name: "Author",
		Fields: []schema.FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
			{Name: "name", Type: "String", NonNull: true},
		},
	}
	childTD := schema.TypeDef{
		Name: "Book",
		Fields: []schema.FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
			{Name: "title", Type: "String", NonNull: true},
			{Name: "author", Type: "Author", NonNull: false, IsRelation: true},
		},
	}
	scalars := schemaScalars()

	if err := schema.CreateTable(ctx, pool, "test", parentTD, scalars); err != nil {
		t.Fatalf("CreateTable Author: %v", err)
	}
	if err := schema.CreateTable(ctx, pool, "test", childTD, scalars); err != nil {
		t.Fatalf("CreateTable Book: %v", err)
	}

	ps := &schema.ParsedSchema{Types: []schema.TypeDef{parentTD, childTD}}
	h, err := schema.BuildHandler(pool, "test", ps, scalars, []plugin.QueryModifier{simple.New()}, nil)
	if err != nil {
		t.Fatalf("BuildHandler: %v", err)
	}

	do := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/graphql/test", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}

	// Create Book without author (nullable FK, omit the field)
	createB := do(`{"query":"mutation { book { create(input: {title: \"1984\"}) { id title author { id } } } }"}`)
	var bRes struct {
		Data struct {
			Book struct {
				Create struct {
					ID     string `json:"id"`
					Title  string `json:"title"`
					Author *struct {
						ID string `json:"id"`
					} `json:"author"`
				} `json:"create"`
			} `json:"book"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(createB.Body).Decode(&bRes); err != nil {
		t.Fatalf("book decode: %v", err)
	}
	if len(bRes.Errors) > 0 {
		t.Fatalf("book errors: %v", bRes.Errors)
	}
	if bRes.Data.Book.Create.Author != nil && bRes.Data.Book.Create.Author.ID != "" {
		t.Error("expected nil/empty author for book created without authorId")
	}
}

func TestGraphQLResolvers_ClosedPool(t *testing.T) {
	pool := startPool(t)
	ctx := context.Background()

	td := locationTypeDef()
	scalars := schemaScalars()

	if err := schema.CreateTable(ctx, pool, "test", td, scalars); err != nil {
		t.Fatalf("CreateTable: %v", err)
	}

	ps := &schema.ParsedSchema{Types: []schema.TypeDef{td}}
	h, err := schema.BuildHandler(pool, "test", ps, scalars, []plugin.QueryModifier{simple.New()}, nil)
	if err != nil {
		t.Fatalf("BuildHandler: %v", err)
	}

	// Close pool so DB calls in resolvers fail
	pool.Close()

	doQuery := func(body string) {
		req := httptest.NewRequest(http.MethodPost, "/graphql/test", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		// Errors are returned as GraphQL errors (status 200 with errors field)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d — %s", w.Code, w.Body.String())
		}
	}

	doQuery(`{"query":"{ location { list { id name } } }"}`)
	doQuery(`{"query":"mutation { location { create(input: {name: \"x\"}) { id } } }"}`)
}

func TestGraphQLResolvers_ListPagination(t *testing.T) {
	pool := startPool(t)
	ctx := context.Background()

	td := locationTypeDef()
	scalars := map[string]scalar.Plugin{
		"String": stringscalar.Plugin{},
		"ID":     idscalar.Plugin{},
		"Int":    intscalar.Plugin{},
	}

	if err := schema.CreateTable(ctx, pool, "test", td, scalars); err != nil {
		t.Fatalf("CreateTable: %v", err)
	}

	t.Setenv("STRATUM_PLUGINS_PAGINATION_MAX_LIMIT", "5")
	ps := &schema.ParsedSchema{Types: []schema.TypeDef{td}}
	h, err := schema.BuildHandler(pool, "test", ps, scalars, []plugin.QueryModifier{simple.New()}, nil)
	if err != nil {
		t.Fatalf("BuildHandler: %v", err)
	}

	do := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/graphql/test", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}

	// Insert 3 records
	for _, name := range []string{"A", "B", "C"} {
		w := do(`{"query":"mutation { location { create(input: {name: \"` + name + `\"}) { id } } }"}`)
		if w.Code != http.StatusOK {
			t.Fatalf("create %s: %d — %s", name, w.Code, w.Body.String())
		}
	}

	type listResp struct {
		Data struct {
			Location struct {
				List []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"list"`
			} `json:"location"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}

	decodeList := func(w *httptest.ResponseRecorder) listResp {
		t.Helper()
		var r listResp
		if err := json.NewDecoder(w.Body).Decode(&r); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return r
	}

	t.Run("default_limit", func(t *testing.T) {
		r := decodeList(do(`{"query":"{ location { list { id name } } }"}`))
		if len(r.Errors) > 0 {
			t.Fatalf("errors: %v", r.Errors)
		}
		if len(r.Data.Location.List) != 3 {
			t.Errorf("expected 3, got %d", len(r.Data.Location.List))
		}
	})

	t.Run("limit_2", func(t *testing.T) {
		r := decodeList(do(`{"query":"{ location { list(limit: 2) { id name } } }"}`))
		if len(r.Errors) > 0 {
			t.Fatalf("errors: %v", r.Errors)
		}
		if len(r.Data.Location.List) != 2 {
			t.Errorf("expected 2, got %d", len(r.Data.Location.List))
		}
	})

	t.Run("limit_with_offset", func(t *testing.T) {
		r := decodeList(do(`{"query":"{ location { list(limit: 2, offset: 1) { id name } } }"}`))
		if len(r.Errors) > 0 {
			t.Fatalf("errors: %v", r.Errors)
		}
		if len(r.Data.Location.List) != 2 {
			t.Errorf("expected 2, got %d", len(r.Data.Location.List))
		}
	})

	t.Run("offset_beyond_data", func(t *testing.T) {
		r := decodeList(do(`{"query":"{ location { list(offset: 100) { id name } } }"}`))
		if len(r.Errors) > 0 {
			t.Fatalf("errors: %v", r.Errors)
		}
		if len(r.Data.Location.List) != 0 {
			t.Errorf("expected 0, got %d", len(r.Data.Location.List))
		}
	})

	t.Run("limit_exceeds_max", func(t *testing.T) {
		r := decodeList(do(`{"query":"{ location { list(limit: 10) { id name } } }"}`))
		if len(r.Errors) == 0 {
			t.Fatal("expected error for limit exceeding max")
		}
		if !strings.Contains(r.Errors[0].Message, "exceeds maximum") {
			t.Errorf("error = %q, want mention of exceeds maximum", r.Errors[0].Message)
		}
	})

	t.Run("negative_limit_clamped_to_zero", func(t *testing.T) {
		r := decodeList(do(`{"query":"{ location { list(limit: -1) { id name } } }"}`))
		if len(r.Errors) > 0 {
			t.Fatalf("errors: %v", r.Errors)
		}
		if len(r.Data.Location.List) != 0 {
			t.Errorf("expected 0, got %d", len(r.Data.Location.List))
		}
	})

	t.Run("negative_offset_clamped_to_zero", func(t *testing.T) {
		r := decodeList(do(`{"query":"{ location { list(offset: -5) { id name } } }"}`))
		if len(r.Errors) > 0 {
			t.Fatalf("errors: %v", r.Errors)
		}
		if len(r.Data.Location.List) != 3 {
			t.Errorf("expected 3, got %d", len(r.Data.Location.List))
		}
	})

	t.Run("stable_order", func(t *testing.T) {
		r1 := decodeList(do(`{"query":"{ location { list { id } } }"}`))
		r2 := decodeList(do(`{"query":"{ location { list { id } } }"}`))
		if len(r1.Data.Location.List) != len(r2.Data.Location.List) {
			t.Fatalf("lengths differ")
		}
		for i := range r1.Data.Location.List {
			if r1.Data.Location.List[i].ID != r2.Data.Location.List[i].ID {
				t.Fatalf("order differs at %d", i)
			}
		}
	})
}

type brokenFilter struct {
	gqlType graphql.Output
}

func (b brokenFilter) Name() string       { return "broken" }
func (b brokenFilter) ScalarType() string { return "Int" }
func (b brokenFilter) Operators() graphql.InputObjectConfigFieldMap {
	return graphql.InputObjectConfigFieldMap{
		"eq": &graphql.InputObjectFieldConfig{Type: b.gqlType},
	}
}
func (b brokenFilter) ToSQL(string, string, any, int) (string, []any, error) {
	return "", nil, errors.New("broken filter")
}

func TestFilterIntegration(t *testing.T) {
	pool := startPool(t)
	ctx := context.Background()

	td := schema.TypeDef{
		Name: "City",
		Fields: []schema.FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
			{Name: "name", Type: "String", NonNull: true},
			{Name: "pop", Type: "Int", NonNull: true},
		},
	}
	scalars := map[string]scalar.Plugin{
		"String": stringscalar.Plugin{},
		"ID":     idscalar.Plugin{},
		"Int":    intscalar.Plugin{},
	}

	if err := schema.CreateTable(ctx, pool, "test", td, scalars); err != nil {
		t.Fatalf("CreateTable: %v", err)
	}

	filters := []plugin.FilterPlugin{
		eqfilter.New("String", graphql.String),
		eqfilter.New("ID", graphql.ID),
		eqfilter.New("Int", scalars["Int"].GraphQLType()),
	}
	ps := &schema.ParsedSchema{Types: []schema.TypeDef{td}}
	h, err := schema.BuildHandler(pool, "test", ps, scalars, []plugin.QueryModifier{simple.New()}, filters)
	if err != nil {
		t.Fatalf("BuildHandler: %v", err)
	}

	do := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/graphql/test", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}

	// Create records
	do(`{"query":"mutation { city { create(input: {name: \"Zürich\", pop: 400000}) { id } } }"}`)
	do(`{"query":"mutation { city { create(input: {name: \"Bern\", pop: 130000}) { id } } }"}`)
	do(`{"query":"mutation { city { create(input: {name: \"Luzern\", pop: 80000}) { id } } }"}`)

	type listResp struct {
		Data struct {
			City struct {
				List []struct {
					Name string `json:"name"`
					Pop  int    `json:"pop"`
				} `json:"list"`
			} `json:"city"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}

	t.Run("filter_eq_string", func(t *testing.T) {
		w := do(`{"query":"{ city { list(filter: { name: { eq: \"Bern\" } }) { name pop } } }"}`)
		var resp listResp
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp.Errors) > 0 {
			t.Fatalf("errors: %v", resp.Errors)
		}
		if len(resp.Data.City.List) != 1 {
			t.Fatalf("expected 1 record, got %d", len(resp.Data.City.List))
		}
		if resp.Data.City.List[0].Name != "Bern" {
			t.Errorf("name = %q, want Bern", resp.Data.City.List[0].Name)
		}
	})

	t.Run("filter_eq_int", func(t *testing.T) {
		w := do(`{"query":"{ city { list(filter: { pop: { eq: 80000 } }) { name } } }"}`)
		var resp listResp
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp.Errors) > 0 {
			t.Fatalf("errors: %v", resp.Errors)
		}
		if len(resp.Data.City.List) != 1 {
			t.Fatalf("expected 1 record, got %d", len(resp.Data.City.List))
		}
		if resp.Data.City.List[0].Name != "Luzern" {
			t.Errorf("name = %q, want Luzern", resp.Data.City.List[0].Name)
		}
	})

	t.Run("filter_no_match", func(t *testing.T) {
		w := do(`{"query":"{ city { list(filter: { name: { eq: \"Basel\" } }) { name } } }"}`)
		var resp listResp
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp.Data.City.List) != 0 {
			t.Fatalf("expected 0 records, got %d", len(resp.Data.City.List))
		}
	})

	t.Run("no_filter_returns_all", func(t *testing.T) {
		w := do(`{"query":"{ city { list { name } } }"}`)
		var resp listResp
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp.Data.City.List) != 3 {
			t.Fatalf("expected 3 records, got %d", len(resp.Data.City.List))
		}
	})
}

func TestFilterIntegration_BrokenPlugin(t *testing.T) {
	pool := startPool(t)
	ctx := context.Background()

	td := schema.TypeDef{
		Name: "Item",
		Fields: []schema.FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
			{Name: "qty", Type: "Int", NonNull: true},
		},
	}
	scalars := map[string]scalar.Plugin{
		"ID":  idscalar.Plugin{},
		"Int": intscalar.Plugin{},
	}

	if err := schema.CreateTable(ctx, pool, "test", td, scalars); err != nil {
		t.Fatalf("CreateTable: %v", err)
	}

	filters := []plugin.FilterPlugin{brokenFilter{gqlType: scalars["Int"].GraphQLType()}}
	ps := &schema.ParsedSchema{Types: []schema.TypeDef{td}}
	h, err := schema.BuildHandler(pool, "test", ps, scalars, []plugin.QueryModifier{simple.New()}, filters)
	if err != nil {
		t.Fatalf("BuildHandler: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/graphql/test",
		strings.NewReader(`{"query":"{ item { list(filter: { qty: { eq: 1 } }) { id } } }"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var resp struct {
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Errors) == 0 {
		t.Fatal("expected GraphQL error from broken filter plugin")
	}
}
