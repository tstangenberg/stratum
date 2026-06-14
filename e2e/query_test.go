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

func TestListKantone(t *testing.T) {
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
	sdl := `type Kanton { id: ID! name: String! }`
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/swiss",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upload schema: expected 200, got %d — %s", w.Code, w.Body.String())
	}

	// ── 2. Create 5 Kantone ─────────────────────────────────────────────────
	kantone := []string{"Zürich", "Bern", "Luzern", "Uri", "Schwyz"}
	for _, name := range kantone {
		gql := fmt.Sprintf(`{"query":"mutation { kanton { create(input: {name: \"%s\"}) { id } } }"}`, name)
		req = httptest.NewRequest(http.MethodPost, "/graphql/swiss", strings.NewReader(gql))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("create %s: expected 200, got %d — %s", name, w.Code, w.Body.String())
		}
	}

	// ── 3. list without args → returns all 5 (default limit 100) ────────────
	t.Run("default_limit_returns_all", func(t *testing.T) {
		result := gqlQuery(t, handler, `{"query":"{ kanton { list { id name } } }"}`)
		list := extractList(t, result, "kanton")
		if len(list) != 5 {
			t.Fatalf("expected 5 records, got %d", len(list))
		}
	})

	// ── 4. list(limit: 2) → returns 2 ──────────────────────────────────────
	t.Run("limit_2", func(t *testing.T) {
		result := gqlQuery(t, handler, `{"query":"{ kanton { list(limit: 2) { id name } } }"}`)
		list := extractList(t, result, "kanton")
		if len(list) != 2 {
			t.Fatalf("expected 2 records, got %d", len(list))
		}
	})

	// ── 5. list(limit: 2, offset: 2) → returns 2 starting from 3rd ─────────
	t.Run("limit_with_offset", func(t *testing.T) {
		// Get all to know order
		allResult := gqlQuery(t, handler, `{"query":"{ kanton { list { id name } } }"}`)
		allList := extractList(t, allResult, "kanton")

		result := gqlQuery(t, handler, `{"query":"{ kanton { list(limit: 2, offset: 2) { id name } } }"}`)
		list := extractList(t, result, "kanton")
		if len(list) != 2 {
			t.Fatalf("expected 2 records, got %d", len(list))
		}
		// Should match records at positions 2 and 3 from the full list
		if list[0]["id"] != allList[2]["id"] {
			t.Errorf("offset record 0: got id %v, want %v", list[0]["id"], allList[2]["id"])
		}
		if list[1]["id"] != allList[3]["id"] {
			t.Errorf("offset record 1: got id %v, want %v", list[1]["id"], allList[3]["id"])
		}
	})

	// ── 6. list(limit: 9999) → GraphQL error (exceeds hard max) ─────────────
	t.Run("limit_exceeds_max", func(t *testing.T) {
		result := gqlQuery(t, handler, `{"query":"{ kanton { list(limit: 9999) { id name } } }"}`)
		if len(result.Errors) == 0 {
			t.Fatal("expected GraphQL error for limit exceeding max, got none")
		}
	})

	// ── 7. empty table → returns empty array ────────────────────────────────
	t.Run("empty_table", func(t *testing.T) {
		// Upload a new schema with a fresh type that has no records
		sdl2 := `type Gemeinde { id: ID! name: String! }`
		body2, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl2})
		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/empty",
			bytes.NewReader(body2))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		handler.ServeHTTP(w2, req2)
		if w2.Code != http.StatusOK {
			t.Fatalf("upload empty schema: expected 200, got %d — %s", w2.Code, w2.Body.String())
		}

		result := gqlQuerySchema(t, handler, "empty", `{"query":"{ gemeinde { list { id name } } }"}`)
		list := extractList(t, result, "gemeinde")
		if list == nil {
			t.Fatal("expected empty array, got nil")
		}
		if len(list) != 0 {
			t.Fatalf("expected 0 records, got %d", len(list))
		}
	})

	// ── 8. stable ordering ──────────────────────────────────────────────────
	t.Run("stable_order", func(t *testing.T) {
		r1 := gqlQuery(t, handler, `{"query":"{ kanton { list { id } } }"}`)
		r2 := gqlQuery(t, handler, `{"query":"{ kanton { list { id } } }"}`)
		l1 := extractList(t, r1, "kanton")
		l2 := extractList(t, r2, "kanton")
		if len(l1) != len(l2) {
			t.Fatalf("lengths differ: %d vs %d", len(l1), len(l2))
		}
		for i := range l1 {
			if l1[i]["id"] != l2[i]["id"] {
				t.Fatalf("order differs at position %d: %v vs %v", i, l1[i]["id"], l2[i]["id"])
			}
		}
	})
}

func TestGetOrtschaft(t *testing.T) {
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
	sdl := `
		type Kanton {
			id: ID!
			name: String!
		}
		type Ortschaft {
			id: ID!
			name: String!
			kanton: Kanton!
		}
	`
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/swiss",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upload schema: expected 200, got %d — %s", w.Code, w.Body.String())
	}

	// ── 2. Create a Kanton ──────────────────────────────────────────────────
	kantonResult := gqlQuery(t, handler,
		`{"query":"mutation { kanton { create(input: {name: \"Zürich\"}) { id name } } }"}`)
	if len(kantonResult.Errors) > 0 {
		t.Fatalf("create kanton: %v", kantonResult.Errors)
	}
	kantonNS := kantonResult.Data["kanton"].(map[string]any)
	kantonCreate := kantonNS["create"].(map[string]any)
	kantonID := kantonCreate["id"].(string)

	// ── 3. Create an Ortschaft ──────────────────────────────────────────────
	ortCreateBody := fmt.Sprintf(
		`{"query":"mutation { ortschaft { create(input: {name: \"Winterthur\", kantonId: \"%s\"}) { id name } } }"}`,
		kantonID,
	)
	ortResult := gqlQuery(t, handler, ortCreateBody)
	if len(ortResult.Errors) > 0 {
		t.Fatalf("create ortschaft: %v", ortResult.Errors)
	}
	ortNS := ortResult.Data["ortschaft"].(map[string]any)
	ortCreate := ortNS["create"].(map[string]any)
	ortID := ortCreate["id"].(string)

	// ── 4. get(id) returns the record with all scalar fields ────────────────
	t.Run("get_existing_record", func(t *testing.T) {
		getBody := fmt.Sprintf(
			`{"query":"{ ortschaft { get(id: \"%s\") { id name } } }"}`,
			ortID,
		)
		result := gqlQuery(t, handler, getBody)
		rec := extractGet(t, result, "ortschaft")
		if rec == nil {
			t.Fatal("expected record, got null")
		}
		if rec["id"] != ortID {
			t.Errorf("id = %v, want %v", rec["id"], ortID)
		}
		if rec["name"] != "Winterthur" {
			t.Errorf("name = %v, want Winterthur", rec["name"])
		}
	})

	// ── 5. get(id) with unknown ID returns null (not an error) ──────────────
	t.Run("get_unknown_id_returns_null", func(t *testing.T) {
		result := gqlQuery(t, handler,
			`{"query":"{ ortschaft { get(id: \"00000000-0000-0000-0000-000000000000\") { id name } } }"}`)
		if len(result.Errors) > 0 {
			t.Fatalf("expected no errors for unknown ID, got: %v", result.Errors)
		}
		rec := extractGet(t, result, "ortschaft")
		if rec != nil {
			t.Fatalf("expected null for unknown ID, got %v", rec)
		}
	})

	// ── 6. get returns all requested scalar fields correctly typed ──────────
	t.Run("get_scalar_fields_typed", func(t *testing.T) {
		getBody := fmt.Sprintf(
			`{"query":"{ ortschaft { get(id: \"%s\") { id name } } }"}`,
			ortID,
		)
		result := gqlQuery(t, handler, getBody)
		rec := extractGet(t, result, "ortschaft")
		if rec == nil {
			t.Fatal("expected record, got null")
		}
		if _, ok := rec["id"].(string); !ok {
			t.Errorf("id should be string, got %T", rec["id"])
		}
		if _, ok := rec["name"].(string); !ok {
			t.Errorf("name should be string, got %T", rec["name"])
		}
	})
}

type gqlResult struct {
	Data   map[string]any             `json:"data"`
	Errors []struct{ Message string } `json:"errors"`
}

func gqlQuery(t *testing.T, handler http.Handler, body string) gqlResult {
	t.Helper()
	return gqlQuerySchema(t, handler, "swiss", body)
}

func gqlQuerySchema(t *testing.T, handler http.Handler, schemaName, body string) gqlResult {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/graphql/"+schemaName, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("graphql query: expected 200, got %d — %s", w.Code, w.Body.String())
	}
	var result gqlResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode graphql response: %v", err)
	}
	return result
}

func extractList(t *testing.T, result gqlResult, typeName string) []map[string]any {
	t.Helper()
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected GraphQL errors: %v", result.Errors)
	}
	ns, ok := result.Data[typeName].(map[string]any)
	if !ok {
		t.Fatalf("expected %s namespace in data, got %T", typeName, result.Data[typeName])
	}
	listRaw, ok := ns["list"].([]any)
	if !ok {
		t.Fatalf("expected list array in %s, got %T", typeName, ns["list"])
	}
	var list []map[string]any
	for _, item := range listRaw {
		m, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("expected map in list items, got %T", item)
		}
		list = append(list, m)
	}
	if list == nil {
		list = []map[string]any{}
	}
	return list
}

func extractGet(t *testing.T, result gqlResult, typeName string) map[string]any {
	t.Helper()
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected GraphQL errors: %v", result.Errors)
	}
	ns, ok := result.Data[typeName].(map[string]any)
	if !ok {
		t.Fatalf("expected %s namespace in data, got %T", typeName, result.Data[typeName])
	}
	raw := ns["get"]
	if raw == nil {
		return nil
	}
	rec, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("expected map or null in %s.get, got %T", typeName, raw)
	}
	return rec
}
