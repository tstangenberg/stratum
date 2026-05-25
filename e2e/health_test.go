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
	"testing"
	"time"

	"github.com/tstangenberg/stratum/internal/plugin"
	"github.com/tstangenberg/stratum/internal/server"
)

// stubHealthPlugin is a test-only HealthPlugin.
type stubHealthPlugin struct {
	name   string
	status string
}

func (s stubHealthPlugin) Name() string { return s.name }
func (s stubHealthPlugin) Check(_ context.Context) plugin.HealthStatus {
	return plugin.HealthStatus{Status: s.status}
}

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

func TestReadinessOK(t *testing.T) {
	srv := server.NewStratumServer(
		stubHealthPlugin{"database", plugin.StatusOK},
		stubHealthPlugin{"cache", plugin.StatusOK},
	)
	handler := server.Handler(srv)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/ready", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var body struct {
		Status     string                    `json:"status"`
		Components map[string]map[string]any `json:"components"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("response body not valid JSON: %v", err)
	}

	if body.Status != "ok" {
		t.Fatalf("expected status=ok, got %q", body.Status)
	}

	if len(body.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(body.Components))
	}
}

func TestReadinessDegraded(t *testing.T) {
	srv := server.NewStratumServer(
		stubHealthPlugin{"database", plugin.StatusOK},
		stubHealthPlugin{"cache", plugin.StatusError},
	)
	handler := server.Handler(srv)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/ready", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", res.StatusCode)
	}

	var body struct {
		Status     string                    `json:"status"`
		Components map[string]map[string]any `json:"components"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("response body not valid JSON: %v", err)
	}

	if body.Status != "degraded" {
		t.Fatalf("expected status=degraded, got %q", body.Status)
	}
}
