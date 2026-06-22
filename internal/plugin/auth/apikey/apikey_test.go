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

package apikey

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestName(t *testing.T) {
	p := New("secret")
	if p.Name() != "api-key-auth" {
		t.Fatalf("Name() = %q, want %q", p.Name(), "api-key-auth")
	}
}

func TestDefaultPriority(t *testing.T) {
	p := New("secret")
	if p.Priority() != 100 {
		t.Fatalf("Priority() = %d, want 100", p.Priority())
	}
}

func TestPriorityFromEnv(t *testing.T) {
	t.Setenv("STRATUM_HTTP_MIDDLEWARE_API_KEY_AUTH_PRIORITY", "42")
	p := New("secret")
	if p.Priority() != 42 {
		t.Fatalf("Priority() = %d, want 42", p.Priority())
	}
}

func TestPriorityInvalidEnvFallsBack(t *testing.T) {
	t.Setenv("STRATUM_HTTP_MIDDLEWARE_API_KEY_AUTH_PRIORITY", "not-a-number")
	p := New("secret")
	if p.Priority() != 100 {
		t.Fatalf("Priority() = %d, want 100 (default)", p.Priority())
	}
}

func TestWrap(t *testing.T) {
	const key = "my-secret-key"
	p := New(key)

	tests := []struct {
		name       string
		header     string
		wantStatus int
	}{
		{"valid key", key, http.StatusOK},
		{"missing key", "", http.StatusUnauthorized},
		{"wrong key", "wrong", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			handler := p.Wrap(next)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("X-API-Key", tt.header)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestWrap_UnauthorizedBody(t *testing.T) {
	p := New("secret")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := p.Wrap(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("body not valid JSON: %v", err)
	}
	if body["error"] != "unauthorized" {
		t.Errorf("error = %q, want %q", body["error"], "unauthorized")
	}
	if body["message"] != "valid API key required" {
		t.Errorf("message = %q, want %q", body["message"], "valid API key required")
	}
}
