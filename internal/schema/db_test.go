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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/tstangenberg/stratum/internal/plugin/scalar"
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
		"ID":     stringscalar.Plugin{},
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
	h, err := schema.BuildHandler(pool, "test", ps, scalars)
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

	// get non-existent record (covers getRecord error path)
	getErrW := do(`{"query":"{ location { get(id: \"00000000-0000-0000-0000-000000000000\") { id name } } }"}`)
	if getErrW.Code != http.StatusOK {
		t.Fatalf("get-nonexistent: expected 200, got %d — %s", getErrW.Code, getErrW.Body.String())
	}
	var getErrResult struct {
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(getErrW.Body).Decode(&getErrResult); err != nil {
		t.Fatalf("get-nonexistent decode: %v", err)
	}
	if len(getErrResult.Errors) == 0 {
		t.Error("get-nonexistent: expected GraphQL errors for missing record")
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
	h, err := schema.BuildHandler(pool, "test", ps, scalars)
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

func TestGraphQLResolvers_ClosedPool(t *testing.T) {
	pool := startPool(t)
	ctx := context.Background()

	td := locationTypeDef()
	scalars := schemaScalars()

	if err := schema.CreateTable(ctx, pool, "test", td, scalars); err != nil {
		t.Fatalf("CreateTable: %v", err)
	}

	ps := &schema.ParsedSchema{Types: []schema.TypeDef{td}}
	h, err := schema.BuildHandler(pool, "test", ps, scalars)
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
