// SPDX-License-Identifier: AGPL-3.0-or-later
package e2e

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tstangenberg/stratum/internal/server"
)

func TestLivenessOK(t *testing.T) {
	srv := server.NewStratumServer()
	handler := server.Handler(srv)

	// Warm up request to reduce first-call overhead
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Measure response time for the actual test
	start := time.Now()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	duration := time.Since(start)

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

	// Verify 5ms performance criterion
	if duration.Milliseconds() >= 5 {
		t.Fatalf("response time %v exceeds 5ms threshold", duration)
	}
}
