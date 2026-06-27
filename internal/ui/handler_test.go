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

package ui

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

type stubStatusProvider struct {
	liveness   string
	readiness  string
	components map[string]string
	plugins    []PluginInfo
}

func (s *stubStatusProvider) HealthStatus(_ context.Context) HealthResult {
	return HealthResult{
		Liveness:   s.liveness,
		Readiness:  s.readiness,
		Components: s.components,
	}
}

func (s *stubStatusProvider) Plugins() []PluginInfo {
	return s.plugins
}

func TestHandler_RedirectRoot(t *testing.T) {
	provider := &stubStatusProvider{
		liveness:  "ok",
		readiness: "ok",
	}
	h := NewHandler(provider)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Fatalf("expected 301, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/ui/status" {
		t.Fatalf("expected Location=/ui/status, got %q", loc)
	}
}

func TestHandler_StatusPage(t *testing.T) {
	provider := &stubStatusProvider{
		liveness:  "ok",
		readiness: "ok",
		components: map[string]string{
			"database": "ok",
		},
		plugins: []PluginInfo{
			{Name: "database", Type: "health"},
			{Name: "pagination-simple", Type: "query-modifier"},
		},
	}
	h := NewHandler(provider)

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("expected text/html, got %q", ct)
	}

	body := w.Body.String()

	tests := []struct {
		name    string
		contain string
	}{
		{"nav status link", "/ui/status"},
		{"nav schema link", "/ui/schema"},
		{"nav console link", "/ui/console"},
		{"liveness status", "liveness"},
		{"readiness status", "readiness"},
		{"plugin name", "database"},
		{"plugin type", "health"},
		{"api-key input", "api-key"},
		{"localStorage script", "localStorage"},
	}

	for _, tt := range tests {
		if !strings.Contains(body, tt.contain) {
			t.Errorf("%s: page missing %q", tt.name, tt.contain)
		}
	}
}

func TestHandler_NotFound(t *testing.T) {
	provider := &stubStatusProvider{
		liveness:  "ok",
		readiness: "ok",
	}
	h := NewHandler(provider)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestNewHandler_PanicsOnBrokenTemplates(t *testing.T) {
	provider := &stubStatusProvider{liveness: "ok", readiness: "ok"}

	// Verify NewHandler works with valid embedded templates
	h := NewHandler(provider)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}

	// Verify panic on broken template FS
	brokenFS := fstest.MapFS{
		"templates/layout.html": &fstest.MapFile{Data: []byte("{{.Invalid")},
		"templates/status.html": &fstest.MapFile{Data: []byte("")},
	}
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for broken template FS")
		}
	}()
	newHandlerFromFS(provider, brokenFS)
}

func TestHandler_StatusTemplateError(t *testing.T) {
	provider := &stubStatusProvider{
		liveness:  "ok",
		readiness: "ok",
	}
	// Create a template that will fail during execution
	broken := template.Must(template.New("layout.html").Parse("{{.NonExistent.Method}}"))
	h := newHandler(provider, broken)

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		// Header already written before error occurs during template execution;
		// the error text is appended to the response body.
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 200 or 500, got %d", w.Code)
		}
	}
}

func TestHandler_StaticAssets(t *testing.T) {
	provider := &stubStatusProvider{
		liveness:  "ok",
		readiness: "ok",
	}
	h := NewHandler(provider)

	tests := []struct {
		name     string
		path     string
		wantCode int
		wantCT   string
	}{
		{"css file", "/static/style.css", http.StatusOK, "text/css"},
		{"js file", "/static/htmx.min.js", http.StatusOK, "application/javascript"},
		{"not found", "/static/missing.txt", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Fatalf("expected %d, got %d", tt.wantCode, w.Code)
			}
			if tt.wantCT != "" {
				ct := w.Header().Get("Content-Type")
				if !strings.Contains(ct, tt.wantCT) {
					t.Fatalf("expected Content-Type %q, got %q", tt.wantCT, ct)
				}
			}
		})
	}
}
