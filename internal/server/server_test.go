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
	restore := plugin.ResetHealthRegistryForTesting()
	t.Cleanup(restore)
	for _, p := range plugins {
		plugin.RegisterHealthPlugin(func() plugin.HealthPlugin { return p })
	}
	srv := Handler(NewStratumServer())
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

func TestUpsertSchema_InvalidSDL_DetailsPopulated(t *testing.T) {
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

	var body struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Details []struct {
			Line    *int    `json:"line,omitempty"`
			Column  *int    `json:"column,omitempty"`
			Message *string `json:"message,omitempty"`
		} `json:"details"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("response body not valid JSON: %v", err)
	}
	if body.Error != "validation_failed" {
		t.Errorf("error = %q, want %q", body.Error, "validation_failed")
	}
	if len(body.Details) == 0 {
		t.Fatal("expected at least one detail in 422 response")
	}
	d := body.Details[0]
	if d.Line == nil || *d.Line == 0 {
		t.Error("expected non-zero line in first detail")
	}
	if d.Column == nil || *d.Column == 0 {
		t.Error("expected non-zero column in first detail")
	}
	if d.Message == nil || *d.Message == "" {
		t.Error("expected non-empty message in first detail")
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
	restore := plugin.ResetHealthRegistryForTesting()
	t.Cleanup(restore)
	plugin.RegisterHealthPlugin(func() plugin.HealthPlugin { return slowHealthPlugin{} })
	srv := Handler(NewStratumServer())
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

// stubHTTPMiddleware is a test-only HTTPMiddleware.
type stubHTTPMiddleware struct {
	name     string
	priority int
	allowed  bool
}

func (s stubHTTPMiddleware) Name() string  { return s.name }
func (s stubHTTPMiddleware) Priority() int { return s.priority }
func (s stubHTTPMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.allowed {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// recordingMiddleware records invocation order for chain-ordering tests.
type recordingMiddleware struct {
	name     string
	priority int
	order    *[]string
}

func (r *recordingMiddleware) Name() string  { return r.name }
func (r *recordingMiddleware) Priority() int { return r.priority }
func (r *recordingMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		*r.order = append(*r.order, r.name)
		next.ServeHTTP(w, req)
	})
}

func TestWithMiddlewares(t *testing.T) {
	m := stubHTTPMiddleware{name: "stub", priority: 100, allowed: true}
	srv := NewStratumServer().WithMiddlewares(m)
	if len(srv.middlewares) != 1 {
		t.Fatalf("expected 1 middleware, got %d", len(srv.middlewares))
	}
	if srv.middlewares[0].Name() != "stub" {
		t.Errorf("Name() = %q, want %q", srv.middlewares[0].Name(), "stub")
	}
}

func TestBuildChain_AppliesInGivenOrder(t *testing.T) {
	var order []string
	// Passed in deliberately non-priority order; buildChain must preserve it.
	middlewares := []plugin.HTTPMiddleware{
		&recordingMiddleware{name: "third", priority: 300, order: &order},
		&recordingMiddleware{name: "first", priority: 100, order: &order},
		&recordingMiddleware{name: "second", priority: 200, order: &order},
	}

	muxCalled := false
	mux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { muxCalled = true })

	h := buildChain(middlewares, mux)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schemas", nil)
	h.ServeHTTP(httptest.NewRecorder(), req)

	if len(order) != 3 || order[0] != "third" || order[1] != "first" || order[2] != "second" {
		t.Errorf("call order = %v, want [third first second]", order)
	}
	if !muxCalled {
		t.Error("expected mux to be called")
	}
}

func TestMiddleware_Rejects(t *testing.T) {
	srv := NewStratumServer().WithMiddlewares(stubHTTPMiddleware{name: "stub", priority: 100, allowed: false})
	handler := Handler(srv)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schemas", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestMiddleware_Allows(t *testing.T) {
	srv := NewStratumServer().WithMiddlewares(stubHTTPMiddleware{name: "stub", priority: 100, allowed: true})
	handler := Handler(srv)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schemas", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// 501 means the request passed middleware and reached the handler
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", w.Code)
	}
}

func TestMiddleware_HealthExempt(t *testing.T) {
	srv := NewStratumServer().WithMiddlewares(stubHTTPMiddleware{name: "stub", priority: 100, allowed: false})
	handler := Handler(srv)

	t.Run("/api/v1/health/live", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("health/live should return 200, got %d", w.Code)
		}
	})

	t.Run("/api/v1/health/ready", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health/ready", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
			t.Fatalf("health/ready should return 200 or 503, got %d", w.Code)
		}
	})
}

func TestNoMiddlewareSkipsAuth(t *testing.T) {
	srv := NewStratumServer()
	handler := Handler(srv)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with no middleware, got %d", w.Code)
	}
}
