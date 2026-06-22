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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tstangenberg/stratum/internal/plugin/auth/apikey"
	"github.com/tstangenberg/stratum/internal/server"
)

const testAPIKey = "test-secret-key-42"

func TestAuthMissingKey(t *testing.T) {
	srv := server.NewStratumServer().WithMiddlewares(apikey.New(testAPIKey))
	handler := server.Handler(srv)

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

	// Health endpoints must be exempt
	healthPaths := []string{"/api/v1/health/live", "/api/v1/health/ready"}
	for _, path := range healthPaths {
		t.Run("exempt "+path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code == http.StatusUnauthorized {
				t.Fatalf("health endpoint %s should be exempt from auth, got 401", path)
			}
		})
	}
}

func TestAuthInvalidKey(t *testing.T) {
	srv := server.NewStratumServer().WithMiddlewares(apikey.New(testAPIKey))
	handler := server.Handler(srv)

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

	// Valid key should pass through (use a non-DB endpoint to avoid 501)
	t.Run("valid key passes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
		req.Header.Set("X-API-Key", testAPIKey)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 with valid key, got %d", w.Code)
		}
	})
}
