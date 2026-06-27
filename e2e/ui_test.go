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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tstangenberg/stratum/internal/plugin"
	"github.com/tstangenberg/stratum/internal/server"
)

func TestUIStatusPage(t *testing.T) {
	restore := plugin.ResetHealthRegistryForTesting()
	t.Cleanup(restore)
	plugin.RegisterHealthPlugin(func() plugin.HealthPlugin {
		return stubHealthPlugin{"database", plugin.StatusOK}
	})

	srv := server.NewStratumServer()
	handler := server.Handler(srv)

	t.Run("GET /ui redirects to /ui/status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ui", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusMovedPermanently {
			t.Fatalf("expected 301, got %d", w.Code)
		}
		loc := w.Header().Get("Location")
		if loc != "/ui/status" {
			t.Fatalf("expected Location=/ui/status, got %q", loc)
		}
	})

	t.Run("GET /ui/status returns HTML with health and plugins", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ui/status", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		ct := w.Header().Get("Content-Type")
		if !strings.Contains(ct, "text/html") {
			t.Fatalf("expected Content-Type text/html, got %q", ct)
		}

		body := w.Body.String()

		// Sidebar navigation
		if !strings.Contains(body, "/ui/status") {
			t.Fatal("page missing link to /ui/status")
		}
		if !strings.Contains(body, "/ui/schema") {
			t.Fatal("page missing link to /ui/schema")
		}
		if !strings.Contains(body, "/ui/console") {
			t.Fatal("page missing link to /ui/console")
		}

		// API-Key input
		if !strings.Contains(body, "api-key") || !strings.Contains(body, "localStorage") {
			t.Fatal("page missing API-Key input with localStorage support")
		}

		// Health status
		if !strings.Contains(body, "liveness") || !strings.Contains(body, "ok") {
			t.Fatal("page missing liveness health status")
		}
		if !strings.Contains(body, "readiness") {
			t.Fatal("page missing readiness health status")
		}

		// Plugin list with name and type
		if !strings.Contains(body, "database") {
			t.Fatal("page missing plugin name 'database'")
		}
		if !strings.Contains(body, "health") {
			t.Fatal("page missing plugin type 'health'")
		}
	})

	t.Run("unknown path returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ui/nonexistent", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("static assets are served", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ui/static/htmx.min.js", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 for htmx.min.js, got %d", w.Code)
		}
	})
}
