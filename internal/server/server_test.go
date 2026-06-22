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
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tstangenberg/stratum/internal/plugin"
	eqfilter "github.com/tstangenberg/stratum/internal/plugin/filter/eq"
	simplepagination "github.com/tstangenberg/stratum/internal/plugin/pagination/simple"
	idscalar "github.com/tstangenberg/stratum/internal/plugin/scalar/id"
)

var h = Handler(NewStratumServer())

func assert501(t *testing.T, method, path string) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusNotImplemented {
		t.Fatalf("%s %s: expected 501, got %d", method, path, res.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("response body not valid JSON: %v", err)
	}
	if body["error"] != "not_implemented" {
		t.Fatalf("expected error=not_implemented, got %q", body["error"])
	}
}

func TestLiveness(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("response body not valid JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status=ok, got %q", body["status"])
	}
}
func TestInfo(t *testing.T)         { assert501(t, http.MethodGet, "/api/v1/info") }
func TestListSchemas(t *testing.T)  { assert501(t, http.MethodGet, "/api/v1/schemas") }
func TestDeleteSchema(t *testing.T) { assert501(t, http.MethodDelete, "/api/v1/schemas/foo") }
func TestGetSchema(t *testing.T)    { assert501(t, http.MethodGet, "/api/v1/schemas/foo") }
func TestUpsertSchema(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/foo", strings.NewReader(`{"schema":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("POST /api/v1/schemas/foo: expected 501, got %d", w.Code)
	}
}
func TestGetSchemaStatus(t *testing.T) { assert501(t, http.MethodGet, "/api/v1/schemas/foo/status") }

func TestBadRequestBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/foo", strings.NewReader(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad body, got %d", w.Code)
	}
}

func TestNotImplementedHandler_NonSentinelError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	notImplementedHandler(w, r, errors.New("something went wrong"))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// stubHealthPlugin is a test-only HealthPlugin.
type stubHealthPlugin struct {
	name    string
	status  string
	details map[string]any
}

func (s stubHealthPlugin) Name() string { return s.name }
func (s stubHealthPlugin) Check(_ context.Context) plugin.HealthStatus {
	return plugin.HealthStatus{Status: s.status, Details: s.details}
}

func doReadiness(t *testing.T, plugins ...plugin.HealthPlugin) *http.Response {
	t.Helper()
	srv := Handler(NewStratumServer(plugins...))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/ready", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	return w.Result()
}

func TestReadiness_NoPlugins(t *testing.T) {
	res := doReadiness(t)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body struct {
		Status     string         `json:"status"`
		Components map[string]any `json:"components"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("expected status=ok, got %q", body.Status)
	}
	if body.Components == nil {
		t.Fatalf("expected components to be present, got nil")
	}
}

func TestReadiness_AllOK(t *testing.T) {
	res := doReadiness(t,
		stubHealthPlugin{"database", plugin.StatusOK, nil},
		stubHealthPlugin{"cache", plugin.StatusOK, nil},
	)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body struct {
		Status     string                    `json:"status"`
		Components map[string]map[string]any `json:"components"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("expected status=ok, got %q", body.Status)
	}
	if len(body.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(body.Components))
	}
}

func TestReadiness_Degraded(t *testing.T) {
	res := doReadiness(t,
		stubHealthPlugin{"database", plugin.StatusOK, nil},
		stubHealthPlugin{"cache", plugin.StatusError, nil},
	)
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", res.StatusCode)
	}
	var body struct {
		Status     string                    `json:"status"`
		Components map[string]map[string]any `json:"components"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if body.Status != "degraded" {
		t.Fatalf("expected status=degraded, got %q", body.Status)
	}
}

func TestValidSchemaName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty", "", false},
		{"too long", "a" + strings.Repeat("b", 63), false},
		{"starts with digit", "1abc", false},
		{"starts with uppercase", "Abc", false},
		{"starts with underscore", "_abc", false},
		{"valid simple", "locations", true},
		{"valid with underscore", "my_schema", true},
		{"valid with digits", "schema1", true},
		{"exactly 63 chars", "a" + strings.Repeat("b", 62), true},
		{"contains uppercase", "MySchema", false},
		{"contains hyphen", "my-schema", false},
		{"single char", "a", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validSchemaName(tt.input)
			if got != tt.want {
				t.Errorf("validSchemaName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestWithDB_EnablesUpsert(t *testing.T) {
	// WithDB should set s.db so that UpsertSchema proceeds past the nil check.
	// Use a zero-value pool pointer (non-nil) to satisfy the nil guard without connecting.
	pool := new(pgxpool.Pool)
	srv := NewStratumServer().WithDB(pool)

	// Invalid schema name triggers 400 before any DB call — proves WithDB was set.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/INVALID_NAME",
		strings.NewReader(`{"sdl":"type Location { id: ID! }"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	Handler(srv).ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid name with DB set, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestUpsertSchema_InvalidName(t *testing.T) {
	pool := new(pgxpool.Pool)
	srv := NewStratumServer().WithDB(pool)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/INVALID",
		strings.NewReader(`{"sdl":"type Location { id: ID! }"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	Handler(srv).ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpsertSchema_InvalidSDL(t *testing.T) {
	pool := new(pgxpool.Pool)
	srv := NewStratumServer().WithDB(pool)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/locations",
		strings.NewReader(`{"sdl":"type { broken"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	Handler(srv).ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", w.Code)
	}
}

func TestServeGraphQL_NotFound(t *testing.T) {
	srv := NewStratumServer()
	req := httptest.NewRequest(http.MethodPost, "/graphql/nonexistent",
		strings.NewReader(`{"query":"{ location { list { id } } }"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	Handler(srv).ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// slowHealthPlugin is a test plugin that never returns.
type slowHealthPlugin struct{}

func (s slowHealthPlugin) Name() string { return "slow" }
func (s slowHealthPlugin) Check(ctx context.Context) plugin.HealthStatus {
	<-ctx.Done()
	return plugin.HealthStatus{Status: plugin.StatusError}
}

func TestReadiness_Timeout(t *testing.T) {
	srv := Handler(NewStratumServer(slowHealthPlugin{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/ready", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	res := w.Result()
	// Should return 503 (degraded) when timeout occurs
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", res.StatusCode)
	}
	var body struct {
		Status     string                    `json:"status"`
		Components map[string]map[string]any `json:"components"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if body.Status != "degraded" {
		t.Fatalf("expected status=degraded, got %q", body.Status)
	}
}

func TestReadiness_WithDetails(t *testing.T) {
	res := doReadiness(t,
		stubHealthPlugin{"database", plugin.StatusOK, map[string]any{"latency_ms": 5}},
		stubHealthPlugin{"cache", plugin.StatusOK, map[string]any{"latency_ms": 2}},
	)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body struct {
		Status     string                    `json:"status"`
		Components map[string]map[string]any `json:"components"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("expected status=ok, got %q", body.Status)
	}
	if len(body.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(body.Components))
	}
}

func TestWithQueryModifiers(t *testing.T) {
	p := simplepagination.New()
	srv := NewStratumServer().WithQueryModifiers(p)
	if len(srv.queryModifiers) != 1 {
		t.Fatalf("len(queryModifiers) = %d, want 1", len(srv.queryModifiers))
	}
	if srv.queryModifiers[0].Name() != "pagination" {
		t.Errorf("queryModifiers[0].Name() = %q, want %q", srv.queryModifiers[0].Name(), "pagination")
	}
}

func TestWithFilterPlugins(t *testing.T) {
	p := eqfilter.New("ID", idscalar.Plugin{}.GraphQLType())
	srv := NewStratumServer().WithFilterPlugins(p)
	if len(srv.filterPlugins) != 1 {
		t.Fatalf("len(filterPlugins) = %d, want 1", len(srv.filterPlugins))
	}
	if srv.filterPlugins[0].ScalarType() != "ID" {
		t.Errorf("filterPlugins[0].ScalarType() = %q, want %q", srv.filterPlugins[0].ScalarType(), "ID")
	}
}

// stubAuthPlugin is a test-only AuthPlugin.
type stubAuthPlugin struct {
	allowed bool
}

func (s stubAuthPlugin) Name() string { return "stub-auth" }
func (s stubAuthPlugin) Authenticate(r *http.Request) plugin.AuthResult {
	return plugin.AuthResult{Allowed: s.allowed}
}

func TestWithAuthPlugin(t *testing.T) {
	p := stubAuthPlugin{allowed: true}
	srv := NewStratumServer().WithAuthPlugin(p)
	if srv.authPlugin == nil {
		t.Fatal("authPlugin should be set")
	}
}

func TestAuth_RejectsWithout(t *testing.T) {
	srv := NewStratumServer().WithAuthPlugin(stubAuthPlugin{allowed: false})
	handler := Handler(srv)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schemas", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
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
}

func TestAuth_AllowsValid(t *testing.T) {
	srv := NewStratumServer().WithAuthPlugin(stubAuthPlugin{allowed: true})
	handler := Handler(srv)

	// Non-health endpoint with auth allowed — should pass through to handler
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schemas", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// 501 means the request passed auth and reached the handler
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", w.Code)
	}
}

func TestAuth_HealthExempt(t *testing.T) {
	srv := NewStratumServer().WithAuthPlugin(stubAuthPlugin{allowed: false})
	handler := Handler(srv)

	paths := []string{"/api/v1/health/live", "/api/v1/health/ready"}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code == http.StatusUnauthorized {
				t.Fatalf("health %s should be exempt from auth", path)
			}
		})
	}
}

func TestAuth_NilPluginSkipsAuth(t *testing.T) {
	srv := NewStratumServer()
	handler := Handler(srv)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with no auth plugin, got %d", w.Code)
	}
}
