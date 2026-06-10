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

func TestCreateOrtschaft(t *testing.T) {
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

	// ── 1. Upload schema with Kanton and Ortschaft ──────────────────────────
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
		t.Fatalf("upload: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	// ── 2. Create a Kanton ──────────────────────────────────────────────────
	gqlCreateKanton := `{"query":"mutation { kanton { create(input: {name: \"Zürich\"}) { id name } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/swiss",
		strings.NewReader(gqlCreateKanton))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("create kanton: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var kantonResult struct {
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
	if err := json.NewDecoder(w.Body).Decode(&kantonResult); err != nil {
		t.Fatalf("create kanton: decode: %v", err)
	}
	if len(kantonResult.Errors) > 0 {
		t.Fatalf("create kanton: GraphQL errors: %v", kantonResult.Errors)
	}
	kantonID := kantonResult.Data.Kanton.Create.ID
	if kantonID == "" {
		t.Fatal("create kanton: expected non-empty id")
	}

	// ── 3. Create an Ortschaft referencing the Kanton ───────────────────────
	gqlCreateOrt := fmt.Sprintf(
		`{"query":"mutation { ortschaft { create(input: {name: \"Winterthur\", kantonId: \"%s\"}) { id name kanton { id name } } } }"}`,
		kantonID,
	)
	req = httptest.NewRequest(http.MethodPost, "/graphql/swiss",
		strings.NewReader(gqlCreateOrt))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("create ortschaft: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var ortResult struct {
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
	if err := json.NewDecoder(w.Body).Decode(&ortResult); err != nil {
		t.Fatalf("create ortschaft: decode: %v", err)
	}
	if len(ortResult.Errors) > 0 {
		t.Fatalf("create ortschaft: GraphQL errors: %v", ortResult.Errors)
	}
	ortID := ortResult.Data.Ortschaft.Create.ID
	if ortID == "" {
		t.Fatal("create ortschaft: expected non-empty id")
	}
	if ortResult.Data.Ortschaft.Create.Name != "Winterthur" {
		t.Errorf("create ortschaft: name = %q, want %q", ortResult.Data.Ortschaft.Create.Name, "Winterthur")
	}

	// ── 4. Verify FK stored correctly in DB ─────────────────────────────────
	var dbKantonID string
	err = pool.QueryRow(ctx,
		`SELECT kanton_id FROM swiss_ortschaft WHERE id = $1`, ortID).Scan(&dbKantonID)
	if err != nil {
		t.Fatalf("query kanton_id: %v", err)
	}
	if dbKantonID != kantonID {
		t.Errorf("db kanton_id = %q, want %q", dbKantonID, kantonID)
	}

	// ── 5. Relation is traversable in subsequent queries ────────────────────
	gqlGet := fmt.Sprintf(
		`{"query":"{ ortschaft { get(id: \"%s\") { id name kanton { id name } } } }"}`,
		ortID,
	)
	req = httptest.NewRequest(http.MethodPost, "/graphql/swiss",
		strings.NewReader(gqlGet))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get ortschaft: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var getResult struct {
		Data struct {
			Ortschaft struct {
				Get struct {
					ID     string `json:"id"`
					Name   string `json:"name"`
					Kanton struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"kanton"`
				} `json:"get"`
			} `json:"ortschaft"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&getResult); err != nil {
		t.Fatalf("get ortschaft: decode: %v", err)
	}
	if len(getResult.Errors) > 0 {
		t.Fatalf("get ortschaft: GraphQL errors: %v", getResult.Errors)
	}
	if getResult.Data.Ortschaft.Get.Kanton.ID != kantonID {
		t.Errorf("get: kanton.id = %q, want %q", getResult.Data.Ortschaft.Get.Kanton.ID, kantonID)
	}
	if getResult.Data.Ortschaft.Get.Kanton.Name != "Zürich" {
		t.Errorf("get: kanton.name = %q, want %q", getResult.Data.Ortschaft.Get.Kanton.Name, "Zürich")
	}

	// ── 6. Non-existent relation ID returns a GraphQL error ─────────────────
	gqlBadFK := `{"query":"mutation { ortschaft { create(input: {name: \"Ghost\", kantonId: \"nonexistent-id\"}) { id } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/swiss",
		strings.NewReader(gqlBadFK))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("bad fk: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var badFKResult struct {
		Data   any                        `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&badFKResult); err != nil {
		t.Fatalf("bad fk: decode: %v", err)
	}
	if len(badFKResult.Errors) == 0 {
		t.Fatal("bad fk: expected GraphQL errors for non-existent relation ID, got none")
	}
}

func TestCreatePLZ(t *testing.T) {
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

	// ── 1. Upload schema with Kanton, Ortschaft, PLZ ────────────────────────
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
		type PLZ {
			id: ID!
			code: String!
			ortschaft: Ortschaft!
		}
	`
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/geo",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("upload: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	// ── 2. Create Kanton → Ortschaft → PLZ chain ────────────────────────────
	gqlKanton := `{"query":"mutation { kanton { create(input: {name: \"Bern\"}) { id } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/geo",
		strings.NewReader(gqlKanton))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var kRes struct {
		Data struct {
			Kanton struct {
				Create struct {
					ID string `json:"id"`
				} `json:"create"`
			} `json:"kanton"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&kRes); err != nil {
		t.Fatalf("kanton decode: %v", err)
	}
	if len(kRes.Errors) > 0 {
		t.Fatalf("kanton errors: %v", kRes.Errors)
	}
	kantonID := kRes.Data.Kanton.Create.ID

	gqlOrt := fmt.Sprintf(
		`{"query":"mutation { ortschaft { create(input: {name: \"Bern\", kantonId: \"%s\"}) { id } } }"}`,
		kantonID,
	)
	req = httptest.NewRequest(http.MethodPost, "/graphql/geo",
		strings.NewReader(gqlOrt))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var oRes struct {
		Data struct {
			Ortschaft struct {
				Create struct {
					ID string `json:"id"`
				} `json:"create"`
			} `json:"ortschaft"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&oRes); err != nil {
		t.Fatalf("ortschaft decode: %v", err)
	}
	if len(oRes.Errors) > 0 {
		t.Fatalf("ortschaft errors: %v", oRes.Errors)
	}
	ortID := oRes.Data.Ortschaft.Create.ID

	// ── 3. Create PLZ referencing Ortschaft ─────────────────────────────────
	gqlPLZ := fmt.Sprintf(
		`{"query":"mutation { plz { create(input: {code: \"3000\", ortschaftId: \"%s\"}) { id code ortschaft { id name kanton { id name } } } } }"}`,
		ortID,
	)
	req = httptest.NewRequest(http.MethodPost, "/graphql/geo",
		strings.NewReader(gqlPLZ))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("create plz: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var plzResult struct {
		Data struct {
			PLZ struct {
				Create struct {
					ID        string `json:"id"`
					Code      string `json:"code"`
					Ortschaft struct {
						ID     string `json:"id"`
						Name   string `json:"name"`
						Kanton struct {
							ID   string `json:"id"`
							Name string `json:"name"`
						} `json:"kanton"`
					} `json:"ortschaft"`
				} `json:"create"`
			} `json:"plz"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&plzResult); err != nil {
		t.Fatalf("plz decode: %v", err)
	}
	if len(plzResult.Errors) > 0 {
		t.Fatalf("plz errors: %v", plzResult.Errors)
	}
	if plzResult.Data.PLZ.Create.Code != "3000" {
		t.Errorf("plz code = %q, want %q", plzResult.Data.PLZ.Create.Code, "3000")
	}

	// ── 4. Verify FK stored correctly in DB ─────────────────────────────────
	var dbOrtID string
	err = pool.QueryRow(ctx,
		`SELECT ortschaft_id FROM geo_plz WHERE id = $1`,
		plzResult.Data.PLZ.Create.ID).Scan(&dbOrtID)
	if err != nil {
		t.Fatalf("query ortschaft_id: %v", err)
	}
	if dbOrtID != ortID {
		t.Errorf("db ortschaft_id = %q, want %q", dbOrtID, ortID)
	}

	// ── 5. Deep relation traversal: PLZ → Ortschaft → Kanton ────────────────
	if plzResult.Data.PLZ.Create.Ortschaft.ID != ortID {
		t.Errorf("plz.ortschaft.id = %q, want %q", plzResult.Data.PLZ.Create.Ortschaft.ID, ortID)
	}
	if plzResult.Data.PLZ.Create.Ortschaft.Name != "Bern" {
		t.Errorf("plz.ortschaft.name = %q, want %q", plzResult.Data.PLZ.Create.Ortschaft.Name, "Bern")
	}
	if plzResult.Data.PLZ.Create.Ortschaft.Kanton.ID != kantonID {
		t.Errorf("plz.ortschaft.kanton.id = %q, want %q", plzResult.Data.PLZ.Create.Ortschaft.Kanton.ID, kantonID)
	}
	if plzResult.Data.PLZ.Create.Ortschaft.Kanton.Name != "Bern" {
		t.Errorf("plz.ortschaft.kanton.name = %q, want %q", plzResult.Data.PLZ.Create.Ortschaft.Kanton.Name, "Bern")
	}
}
