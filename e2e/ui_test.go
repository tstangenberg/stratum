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

func startUITestServer(t *testing.T) (http.Handler, *pgxpool.Pool) {
	t.Helper()
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

	handler := mustServerHandler(t, server.NewStratumServer().WithDB(pool))
	return handler, pool
}

func uploadSchema(t *testing.T, handler http.Handler, name, sdl string) {
	t.Helper()
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/"+name,
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upload %q: expected 200, got %d — body: %s", name, w.Code, w.Body.String())
	}
}

func TestUISchemaList(t *testing.T) {
	handler, _ := startUITestServer(t)

	t.Run("empty state shows hint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ui/schema", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		ct := w.Header().Get("Content-Type")
		if !strings.Contains(ct, "text/html") {
			t.Fatalf("expected text/html, got %q", ct)
		}
		body := w.Body.String()
		if !strings.Contains(body, "Kein Schema vorhanden") {
			t.Error("empty state hint missing")
		}
	})

	t.Run("lists uploaded schema", func(t *testing.T) {
		uploadSchema(t, handler, "locations", `type Location { id: ID! name: String! }`)

		req := httptest.NewRequest(http.MethodGet, "/ui/schema", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		body := w.Body.String()
		if !strings.Contains(body, "locations") {
			t.Error("schema name 'locations' missing from list")
		}
		if !strings.Contains(body, "1") {
			t.Error("schema version missing from list")
		}
	})

	t.Run("page contains schema UI elements", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ui/schema", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		body := w.Body.String()
		for _, want := range []string{
			"schema-name",
			"Hochladen",
			"Formatieren",
		} {
			if !strings.Contains(body, want) {
				t.Errorf("page missing %q", want)
			}
		}
	})
}

func TestUISchemaUpload(t *testing.T) {
	handler, _ := startUITestServer(t)

	uploadSchema(t, handler, "tasks", `type Task { id: ID! title: String! }`)

	req := httptest.NewRequest(http.MethodGet, "/ui/schema", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "tasks") {
		t.Error("uploaded schema 'tasks' not visible on schema page")
	}
}

func TestUISchemaLint(t *testing.T) {
	handler, _ := startUITestServer(t)

	invalidSDL := `type Broken { id: ID! name: }`
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: invalidSDL})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/test?preview=true",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d — body: %s", w.Code, w.Body.String())
	}

	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Details *[]struct {
			Line    *int    `json:"line,omitempty"`
			Column  *int    `json:"column,omitempty"`
			Message *string `json:"message,omitempty"`
		} `json:"details,omitempty"`
	}
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if errResp.Error != "validation_failed" {
		t.Errorf("error = %q, want %q", errResp.Error, "validation_failed")
	}
	if errResp.Details == nil || len(*errResp.Details) == 0 {
		t.Fatal("expected validation details with line/column info")
	}
	detail := (*errResp.Details)[0]
	if detail.Line == nil {
		t.Error("expected line number in validation detail")
	}
	if detail.Message == nil {
		t.Error("expected message in validation detail")
	}
}

func TestUIGraphQLConsole(t *testing.T) {
	handler, _ := startUITestServer(t)

	t.Run("console page renders", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ui/console", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		ct := w.Header().Get("Content-Type")
		if !strings.Contains(ct, "text/html") {
			t.Fatalf("expected text/html, got %q", ct)
		}

		body := w.Body.String()
		for _, want := range []string{
			"Console",
			"schema-select",
			"query-input",
			"btn-execute",
			"result-output",
			"console.js",
		} {
			if !strings.Contains(body, want) {
				t.Errorf("page missing %q", want)
			}
		}
	})

	t.Run("schema dropdown populated after upload", func(t *testing.T) {
		uploadSchema(t, handler, "widgets", `type Widget { id: ID! name: String! }`)

		req := httptest.NewRequest(http.MethodGet, "/ui/console", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		body := w.Body.String()
		if !strings.Contains(body, "widgets") {
			t.Error("schema 'widgets' missing from console dropdown")
		}
	})

	t.Run("list schemas API returns uploaded schema", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/schemas", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
		}
		ct := w.Header().Get("Content-Type")
		if !strings.Contains(ct, "application/json") {
			t.Fatalf("expected application/json, got %q", ct)
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
		found := false
		for _, s := range resp.Schemas {
			if s.Name == "widgets" {
				found = true
				break
			}
		}
		if !found {
			t.Error("schema 'widgets' not in list response")
		}
	})

	t.Run("graphql query execution", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"query": `{ widget { list { id name } } }`,
		})
		req := httptest.NewRequest(http.MethodPost, "/graphql/widgets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
		}
		ct := w.Header().Get("Content-Type")
		if !strings.Contains(ct, "application/json") {
			t.Fatalf("expected application/json, got %q", ct)
		}

		var gqlResp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&gqlResp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if _, ok := gqlResp["data"]; !ok {
			t.Error("expected 'data' key in GraphQL response")
		}
	})
}
