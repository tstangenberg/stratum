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
	return pool
}

func TestUpsertSchema_CreateTableError(t *testing.T) {
	pool := startServerPool(t)
	pool.Close() // closed pool forces CreateTable to fail
	srv := NewStratumServer().WithDB(pool)
	h := Handler(srv)

	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: `type Location { id: ID! name: String! }`})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/locations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when DB closed, got %d — %s", w.Code, w.Body.String())
	}
}

func TestUpsertSchema_Success(t *testing.T) {
	pool := startServerPool(t)
	srv := NewStratumServer().WithDB(pool)
	h := Handler(srv)

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
