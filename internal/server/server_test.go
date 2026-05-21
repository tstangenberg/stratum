// SPDX-License-Identifier: AGPL-3.0-or-later
package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tstangenberg/stratum/internal/server"
)

func TestUnimplementedEndpointReturns501(t *testing.T) {
	h := server.Handler(server.NewStratumServer())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schemas", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", res.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("response body not valid JSON: %v", err)
	}
	if body["error"] != "not_implemented" {
		t.Fatalf("expected error=not_implemented, got %q", body["error"])
	}
}
