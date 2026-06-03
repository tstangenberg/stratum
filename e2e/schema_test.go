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

func TestSchemaBooleanScalar(t *testing.T) {
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

	// ── 1. Upload schema with Boolean! field ────────────────────────────────
	sdl := `type Record { id: ID! name: String! inAenderung: Boolean! }`
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/records",
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

	// ── 2. Verify PostgreSQL column type is BOOLEAN ─────────────────────────
	var colType string
	err = pool.QueryRow(ctx,
		`SELECT data_type FROM information_schema.columns
		 WHERE table_name = 'records_record' AND column_name = 'inaenderung'`).Scan(&colType)
	if err != nil {
		t.Fatalf("column type query: %v", err)
	}
	if colType != "boolean" {
		t.Errorf("column type = %q, want %q", colType, "boolean")
	}

	// ── 3. Create with true, read back ──────────────────────────────────────
	gqlCreate := `{"query":"mutation { record { create(input: {name: \"Test\", inAenderung: true}) { id name inAenderung } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/records",
		strings.NewReader(gqlCreate))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("create-true: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var createResult struct {
		Data struct {
			Record struct {
				Create struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					InAenderung bool   `json:"inAenderung"`
				} `json:"create"`
			} `json:"record"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&createResult); err != nil {
		t.Fatalf("create-true: decode: %v", err)
	}
	if len(createResult.Errors) > 0 {
		t.Fatalf("create-true: GraphQL errors: %v", createResult.Errors)
	}
	trueID := createResult.Data.Record.Create.ID
	if trueID == "" {
		t.Fatal("create-true: expected non-empty id")
	}
	if !createResult.Data.Record.Create.InAenderung {
		t.Errorf("create-true: inAenderung = false, want true")
	}

	// Read back true record
	gqlGet := fmt.Sprintf(`{"query":"{ record { get(id: \"%s\") { id inAenderung } } }"}`, trueID)
	req = httptest.NewRequest(http.MethodPost, "/graphql/records",
		strings.NewReader(gqlGet))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get-true: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var getResult struct {
		Data struct {
			Record struct {
				Get struct {
					ID          string `json:"id"`
					InAenderung bool   `json:"inAenderung"`
				} `json:"get"`
			} `json:"record"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&getResult); err != nil {
		t.Fatalf("get-true: decode: %v", err)
	}
	if len(getResult.Errors) > 0 {
		t.Fatalf("get-true: GraphQL errors: %v", getResult.Errors)
	}
	if !getResult.Data.Record.Get.InAenderung {
		t.Errorf("get-true: inAenderung = false, want true")
	}

	// ── 4. Create with false, read back ─────────────────────────────────────
	gqlCreateFalse := `{"query":"mutation { record { create(input: {name: \"Test2\", inAenderung: false}) { id inAenderung } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/records",
		strings.NewReader(gqlCreateFalse))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("create-false: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var createFalseResult struct {
		Data struct {
			Record struct {
				Create struct {
					ID          string `json:"id"`
					InAenderung bool   `json:"inAenderung"`
				} `json:"create"`
			} `json:"record"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&createFalseResult); err != nil {
		t.Fatalf("create-false: decode: %v", err)
	}
	if len(createFalseResult.Errors) > 0 {
		t.Fatalf("create-false: GraphQL errors: %v", createFalseResult.Errors)
	}
	if createFalseResult.Data.Record.Create.InAenderung {
		t.Errorf("create-false: inAenderung = true, want false")
	}

	// ── 5. String "true" rejected as invalid Boolean input ──────────────────
	gqlStringBool := `{"query":"mutation { record { create(input: {name: \"Bad\", inAenderung: \"true\"}) { id } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/records",
		strings.NewReader(gqlStringBool))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("string-bool: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var stringBoolResult struct {
		Data   any                        `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&stringBoolResult); err != nil {
		t.Fatalf("string-bool: decode: %v", err)
	}
	if len(stringBoolResult.Errors) == 0 {
		t.Fatal("string-bool: expected GraphQL errors for string input, got none")
	}
}
