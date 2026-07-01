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

	"github.com/tstangenberg/stratum/internal/plugin/auth/apikey"
	"github.com/tstangenberg/stratum/internal/server"
)

const testAPIKey = "test-secret-key-42"

func startAuthTestServer(t *testing.T) http.Handler {
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

	return mustServerHandler(t, server.NewStratumServer().WithDB(pool).WithMiddlewares(apikey.New(testAPIKey)))
}

func TestAuthMissingKey(t *testing.T) {
	handler := startAuthTestServer(t)

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/v1/schemas/locations", `{"sdl":"type Location { id: ID! }"}`},
		{http.MethodGet, "/api/v1/schemas", ""},
		{http.MethodPost, "/graphql/locations", `{"query":"{ location { list { id } } }"}`},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			var req *http.Request
			if ep.body != "" {
				req = httptest.NewRequest(ep.method, ep.path, strings.NewReader(ep.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(ep.method, ep.path, nil)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d — body: %s", w.Code, w.Body.String())
			}

			var body map[string]string
			if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
				t.Fatalf("response body not valid JSON: %v", err)
			}
			if body["error"] != "unauthorized" {
				t.Errorf("error = %q, want %q", body["error"], "unauthorized")
			}
			if body["message"] != "valid API key required" {
				t.Errorf("message = %q, want %q", body["message"], "valid API key required")
			}
		})
	}

	t.Run("exempt /api/v1/health/live", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("health/live should return 200, got %d", w.Code)
		}
	})

	t.Run("exempt /api/v1/health/ready", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health/ready", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
			t.Fatalf("health/ready should return 200 or 503, got %d", w.Code)
		}
	})
}

func TestAuthInvalidKey(t *testing.T) {
	handler := startAuthTestServer(t)

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/v1/schemas/locations", `{"sdl":"type Location { id: ID! }"}`},
		{http.MethodGet, "/api/v1/schemas", ""},
		{http.MethodPost, "/graphql/locations", `{"query":"{ location { list { id } } }"}`},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			var req *http.Request
			if ep.body != "" {
				req = httptest.NewRequest(ep.method, ep.path, strings.NewReader(ep.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(ep.method, ep.path, nil)
			}
			req.Header.Set("X-API-Key", "wrong-key")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d — body: %s", w.Code, w.Body.String())
			}

			var body map[string]string
			if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
				t.Fatalf("response body not valid JSON: %v", err)
			}
			if body["error"] != "unauthorized" {
				t.Errorf("error = %q, want %q", body["error"], "unauthorized")
			}
			if body["message"] != "valid API key required" {
				t.Errorf("message = %q, want %q", body["message"], "valid API key required")
			}
		})
	}

	// Use GET /api/v1/schemas (non-exempt, returns 200 = passed auth and reached handler)
	t.Run("valid key passes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/schemas", nil)
		req.Header.Set("X-API-Key", testAPIKey)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 (passed auth), got %d — body: %s", w.Code, w.Body.String())
		}
	})
}
