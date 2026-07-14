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

package server

import (
	"bytes"
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

	"github.com/tstangenberg/stratum/internal/api"
	"github.com/tstangenberg/stratum/internal/system"
)

func startServerPool(t *testing.T) *pgxpool.Pool {
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
	if err := system.Migrate(ctx, pool); err != nil {
		t.Fatalf("system.Migrate: %v", err)
	}
	return pool
}

func TestUpsertSchema_CreateTableError(t *testing.T) {
	pool := startServerPool(t)
	pool.Close() // closed pool forces CreateTable to fail
	srv := NewStratumServer().WithDB(pool)
	h := mustHandler(srv)

	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: `type Location { id: ID! name: String! }`})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/locations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when DB closed, got %d — %s", w.Code, w.Body.String())
	}
}

func TestUpsertSchema_BuildHandlerError(t *testing.T) {
	pool := startServerPool(t)
	srv := NewStratumServer().WithDB(pool)
	h := mustHandler(srv)

	sdl := `type Foo { id: ID! name: String! } type FooQuery { id: ID! name: String! }`
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/locations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when BuildHandler fails, got %d — %s", w.Code, w.Body.String())
	}
}

func TestListSchemas_WithSchemas(t *testing.T) {
	pool := startServerPool(t)
	srv := NewStratumServer().WithDB(pool)
	h := mustHandler(srv)

	sdl := `type Location { id: ID! name: String! }`
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/locations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upload: expected 200, got %d — %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/schemas", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", w.Code)
	}

	var resp struct {
		Schemas []struct {
			Name    string `json:"name"`
			Version int    `json:"version"`
		} `json:"schemas"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(resp.Schemas))
	}
	if resp.Schemas[0].Name != "locations" {
		t.Errorf("name = %q, want %q", resp.Schemas[0].Name, "locations")
	}
	if resp.Schemas[0].Version != 1 {
		t.Errorf("version = %d, want 1", resp.Schemas[0].Version)
	}
}

func TestUpsertSchema_Success(t *testing.T) {
	pool := startServerPool(t)
	srv := NewStratumServer().WithDB(pool)
	h := mustHandler(srv)

	sdl := `type Location { id: ID! name: String! }`
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/locations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — %s", w.Code, w.Body.String())
	}

	var resp api.SchemaUploadResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Name != "locations" {
		t.Errorf("name = %q, want %q", resp.Name, "locations")
	}
	if resp.Status != api.Applied {
		t.Errorf("status = %q, want applied", resp.Status)
	}
	if resp.Version != 1 {
		t.Errorf("version = %d, want 1", resp.Version)
	}
	wantEndpoint := "/graphql/locations"
	if resp.GraphqlEndpoint == nil || *resp.GraphqlEndpoint != wantEndpoint {
		t.Errorf("graphql_endpoint = %v, want %q", resp.GraphqlEndpoint, wantEndpoint)
	}

	// Also exercise serveGraphQL — creates a record and gets it back
	gqlCreate := `{"query":"mutation { location { create(input: {name: \"Munich\"}) { id name } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/locations", strings.NewReader(gqlCreate))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("graphql create: expected 200, got %d — %s", w.Code, w.Body.String())
	}
}

func TestUpsertSchema_Reupload(t *testing.T) {
	pool := startServerPool(t)
	srv := NewStratumServer().WithDB(pool)
	h := mustHandler(srv)

	// First upload
	sdl1 := `type Location { id: ID! name: String! }`
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl1})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/locations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upload v1: expected 200, got %d — %s", w.Code, w.Body.String())
	}

	// Re-upload with added field
	sdl2 := `type Location { id: ID! name: String! description: String }`
	body, _ = json.Marshal(api.SchemaUploadRequest{Sdl: sdl2})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/schemas/locations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upload v2: expected 200, got %d — %s", w.Code, w.Body.String())
	}

	var resp api.SchemaUploadResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode v2: %v", err)
	}
	if resp.Version != 2 {
		t.Errorf("version = %d, want 2", resp.Version)
	}
	if resp.Status != api.Applied {
		t.Errorf("status = %q, want applied", resp.Status)
	}

	// Idempotent re-upload
	body, _ = json.Marshal(api.SchemaUploadRequest{Sdl: sdl2})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/schemas/locations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upload v3: expected 200, got %d — %s", w.Code, w.Body.String())
	}

	var resp3 api.SchemaUploadResponse
	if err := json.NewDecoder(w.Body).Decode(&resp3); err != nil {
		t.Fatalf("decode v3: %v", err)
	}
	if resp3.Version != 3 {
		t.Errorf("version = %d, want 3", resp3.Version)
	}

	// GraphQL still works with the new field
	gqlCreate := `{"query":"mutation { location { create(input: {name: \"Berlin\", description: \"Capital\"}) { id name description } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/locations", strings.NewReader(gqlCreate))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("graphql create: expected 200, got %d — %s", w.Code, w.Body.String())
	}
}

func TestUpsertSchema_ReuploadAddColumnsError(t *testing.T) {
	pool := startServerPool(t)
	srv := NewStratumServer().WithDB(pool)
	h := mustHandler(srv)

	// First upload succeeds
	sdl1 := `type Item { id: ID! name: String! }`
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl1})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/items", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upload v1: expected 200, got %d — %s", w.Code, w.Body.String())
	}

	// Close pool to force AddColumns to fail on re-upload
	pool.Close()

	sdl2 := `type Item { id: ID! name: String! price: Int }`
	body, _ = json.Marshal(api.SchemaUploadRequest{Sdl: sdl2})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/schemas/items", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when AddColumns fails, got %d — %s", w.Code, w.Body.String())
	}
}

func TestUpsertSchema_ReuploadNewType(t *testing.T) {
	pool := startServerPool(t)
	srv := NewStratumServer().WithDB(pool)
	h := mustHandler(srv)

	// First upload with one type
	sdl1 := `type Author { id: ID! name: String! }`
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl1})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/library", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upload v1: expected 200, got %d — %s", w.Code, w.Body.String())
	}

	// Re-upload adding a new type
	sdl2 := `type Author { id: ID! name: String! }
type Book { id: ID! title: String! }`
	body, _ = json.Marshal(api.SchemaUploadRequest{Sdl: sdl2})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/schemas/library", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upload v2: expected 200, got %d — %s", w.Code, w.Body.String())
	}

	var resp api.SchemaUploadResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode v2: %v", err)
	}
	if resp.Version != 2 {
		t.Errorf("version = %d, want 2", resp.Version)
	}
}

func TestUpsertSchema_ReuploadNewTypeError(t *testing.T) {
	pool := startServerPool(t)
	srv := NewStratumServer().WithDB(pool)
	h := mustHandler(srv)

	// First upload
	sdl1 := `type Widget { id: ID! name: String! }`
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl1})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/widgets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upload v1: expected 200, got %d — %s", w.Code, w.Body.String())
	}

	// Close pool then re-upload adding a new type → CreateTable fails
	pool.Close()

	sdl2 := `type Widget { id: ID! name: String! }
type Gadget { id: ID! label: String! }`
	body, _ = json.Marshal(api.SchemaUploadRequest{Sdl: sdl2})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/schemas/widgets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when CreateTable fails on re-upload, got %d — %s", w.Code, w.Body.String())
	}
}

func TestUpsertSchema_ReuploadAfterRestart(t *testing.T) {
	pool := startServerPool(t)

	// "First server": upload v1 with one field.
	srv1 := NewStratumServer().WithDB(pool)
	h1 := mustHandler(srv1)

	sdl1 := `type Device { id: ID! serial: String! }`
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl1})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h1.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upload v1: expected 200, got %d — %s", w.Code, w.Body.String())
	}

	// "Restart": new server with an empty in-memory store — same DB.
	srv2 := NewStratumServer().WithDB(pool)
	h2 := mustHandler(srv2)

	// Re-upload v2 with a new field. The table already exists in the DB but
	// the store is empty, so isReupload would have been false in the old code.
	sdl2 := `type Device { id: ID! serial: String! firmware: String }`
	body, _ = json.Marshal(api.SchemaUploadRequest{Sdl: sdl2})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/schemas/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h2.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upload v2 after restart: expected 200, got %d — %s", w.Code, w.Body.String())
	}

	// The firmware column must exist in the DB.
	var colExists bool
	err := pool.QueryRow(context.Background(),
		`SELECT EXISTS(SELECT 1 FROM information_schema.columns
		 WHERE table_name = 'devices_device' AND column_name = 'firmware')`).Scan(&colExists)
	if err != nil {
		t.Fatalf("column check: %v", err)
	}
	if !colExists {
		t.Fatal("firmware column not found — AddColumns was not called after restart")
	}

	// GraphQL must work with the new field.
	gqlCreate := `{"query":"mutation { device { create(input: {serial: \"SN-001\", firmware: \"2.0\"}) { id serial firmware } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/devices", strings.NewReader(gqlCreate))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h2.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("graphql create after restart: expected 200, got %d — %s", w.Code, w.Body.String())
	}
}
