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
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/tstangenberg/stratum/internal/api"
	"github.com/tstangenberg/stratum/internal/server"
)

func TestUploadSchemaString(t *testing.T) {
	ctx := context.Background()

	pgc, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("stratum"),
		postgres.WithUsername("stratum"),
		postgres.WithPassword("stratum"),
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
		t.Fatalf("connect pool: %v", err)
	}
	t.Cleanup(pool.Close)

	handler := server.Handler(server.NewStratumServer().WithDB(pool))

	// ── 1. Upload schema ────────────────────────────────────────────────────
	sdl := `type Location { id: ID! name: String! }`
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/locations",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("upload: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var uploadResp api.SchemaUploadResponse
	if err := json.NewDecoder(w.Body).Decode(&uploadResp); err != nil {
		t.Fatalf("upload: decode response: %v", err)
	}
	if uploadResp.Name != "locations" {
		t.Errorf("upload: name = %q, want %q", uploadResp.Name, "locations")
	}
	if uploadResp.Status != api.Applied {
		t.Errorf("upload: status = %q, want %q", uploadResp.Status, api.Applied)
	}
	if uploadResp.Version != 1 {
		t.Errorf("upload: version = %d, want 1", uploadResp.Version)
	}
	wantEndpoint := "/graphql/locations"
	if uploadResp.GraphqlEndpoint == nil || *uploadResp.GraphqlEndpoint != wantEndpoint {
		t.Errorf("upload: graphql_endpoint = %v, want %q", uploadResp.GraphqlEndpoint, wantEndpoint)
	}

	// ── 2. Create a record via GraphQL ──────────────────────────────────────
	gqlCreate := `{"query":"mutation { location { create(input: {name: \"Zürich\"}) { id name } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/locations",
		strings.NewReader(gqlCreate))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d — body: %s", w.Code, w.Body.String())
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
	if err := json.NewDecoder(w.Body).Decode(&createResult); err != nil {
		t.Fatalf("create: decode: %v", err)
	}
	if len(createResult.Errors) > 0 {
		t.Fatalf("create: GraphQL errors: %v", createResult.Errors)
	}
	createdID := createResult.Data.Location.Create.ID
	if createdID == "" {
		t.Fatal("create: expected non-empty id")
	}
	if createResult.Data.Location.Create.Name != "Zürich" {
		t.Errorf("create: name = %q, want %q", createResult.Data.Location.Create.Name, "Zürich")
	}

	// ── 3. Read back via GraphQL get ────────────────────────────────────────
	gqlGet := fmt.Sprintf(`{"query":"{ location { get(id: \"%s\") { id name } } }"}`, createdID)
	req = httptest.NewRequest(http.MethodPost, "/graphql/locations",
		strings.NewReader(gqlGet))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var getResult struct {
		Data struct {
			Location struct {
				Get struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"get"`
			} `json:"location"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&getResult); err != nil {
		t.Fatalf("get: decode: %v", err)
	}
	if len(getResult.Errors) > 0 {
		t.Fatalf("get: GraphQL errors: %v", getResult.Errors)
	}
	if getResult.Data.Location.Get.ID != createdID {
		t.Errorf("get: id = %q, want %q", getResult.Data.Location.Get.ID, createdID)
	}
	if getResult.Data.Location.Get.Name != "Zürich" {
		t.Errorf("get: name = %q, want %q", getResult.Data.Location.Get.Name, "Zürich")
	}
}

func TestSchemaIDScalar(t *testing.T) {
	ctx := context.Background()

	pgc, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("stratum"),
		postgres.WithUsername("stratum"),
		postgres.WithPassword("stratum"),
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
		t.Fatalf("connect pool: %v", err)
	}
	t.Cleanup(pool.Close)

	handler := server.Handler(server.NewStratumServer().WithDB(pool))

	// ── 1. Upload schema with ID! field ─────────────────────────────────────
	sdl := `type Thing { id: ID! name: String! }`
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/things",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("upload: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var uploadResp api.SchemaUploadResponse
	if err := json.NewDecoder(w.Body).Decode(&uploadResp); err != nil {
		t.Fatalf("upload: decode response: %v", err)
	}
	if uploadResp.Status != api.Applied {
		t.Errorf("upload: status = %q, want %q", uploadResp.Status, api.Applied)
	}

	// ── 2. Verify PostgreSQL column type is TEXT and primary key ─────────────
	var colType string
	err = pool.QueryRow(ctx,
		`SELECT data_type FROM information_schema.columns
		 WHERE table_name = 'things_thing' AND column_name = 'id'`).Scan(&colType)
	if err != nil {
		t.Fatalf("column type query: %v", err)
	}
	if colType != "text" {
		t.Errorf("column type = %q, want %q", colType, "text")
	}

	var constraintType string
	err = pool.QueryRow(ctx,
		`SELECT constraint_type FROM information_schema.table_constraints
		 WHERE table_name = 'things_thing' AND constraint_type = 'PRIMARY KEY'`).Scan(&constraintType)
	if err != nil {
		t.Fatalf("primary key query: %v", err)
	}
	if constraintType != "PRIMARY KEY" {
		t.Errorf("constraint_type = %q, want %q", constraintType, "PRIMARY KEY")
	}

	// ── 3. Client-supplied ID is stored and returned as-is ──────────────────
	gqlCreate := `{"query":"mutation { thing { create(input: {id: \"my-custom-id\", name: \"Widget\"}) { id name } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/things",
		strings.NewReader(gqlCreate))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("create-custom: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var createResult struct {
		Data struct {
			Thing struct {
				Create struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"create"`
			} `json:"thing"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&createResult); err != nil {
		t.Fatalf("create-custom: decode: %v", err)
	}
	if len(createResult.Errors) > 0 {
		t.Fatalf("create-custom: GraphQL errors: %v", createResult.Errors)
	}
	if createResult.Data.Thing.Create.ID != "my-custom-id" {
		t.Errorf("create-custom: id = %q, want %q", createResult.Data.Thing.Create.ID, "my-custom-id")
	}

	// Read it back
	gqlGet := `{"query":"{ thing { get(id: \"my-custom-id\") { id name } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/things",
		strings.NewReader(gqlGet))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get-custom: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var getResult struct {
		Data struct {
			Thing struct {
				Get struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"get"`
			} `json:"thing"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&getResult); err != nil {
		t.Fatalf("get-custom: decode: %v", err)
	}
	if len(getResult.Errors) > 0 {
		t.Fatalf("get-custom: GraphQL errors: %v", getResult.Errors)
	}
	if getResult.Data.Thing.Get.ID != "my-custom-id" {
		t.Errorf("get-custom: id = %q, want %q", getResult.Data.Thing.Get.ID, "my-custom-id")
	}
	if getResult.Data.Thing.Get.Name != "Widget" {
		t.Errorf("get-custom: name = %q, want %q", getResult.Data.Thing.Get.Name, "Widget")
	}

	// ── 4. Duplicate ID returns a GraphQL error ─────────────────────────────
	gqlDup := `{"query":"mutation { thing { create(input: {id: \"my-custom-id\", name: \"Duplicate\"}) { id } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/things",
		strings.NewReader(gqlDup))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("dup: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var dupResult struct {
		Data   any                        `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&dupResult); err != nil {
		t.Fatalf("dup: decode: %v", err)
	}
	if len(dupResult.Errors) == 0 {
		t.Fatal("dup: expected GraphQL errors for duplicate ID, got none")
	}

	// ── 5. Omitting id generates a unique ID ────────────────────────────────
	gqlAuto := `{"query":"mutation { thing { create(input: {name: \"AutoID\"}) { id name } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/things",
		strings.NewReader(gqlAuto))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("auto-id: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var autoResult struct {
		Data struct {
			Thing struct {
				Create struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"create"`
			} `json:"thing"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&autoResult); err != nil {
		t.Fatalf("auto-id: decode: %v", err)
	}
	if len(autoResult.Errors) > 0 {
		t.Fatalf("auto-id: GraphQL errors: %v", autoResult.Errors)
	}
	if autoResult.Data.Thing.Create.ID == "" {
		t.Fatal("auto-id: expected non-empty generated id")
	}
	if autoResult.Data.Thing.Create.ID == "my-custom-id" {
		t.Error("auto-id: generated id should differ from the custom one")
	}
}
