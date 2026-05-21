// SPDX-License-Identifier: AGPL-3.0-or-later
package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

func TestReadiness(t *testing.T)    { assert501(t, http.MethodGet, "/api/v1/health/ready") }
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
