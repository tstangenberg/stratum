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
	"errors"
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

type stubSchemaProvider struct {
	schemas []SchemaInfo
}

func (s *stubSchemaProvider) Schemas() []SchemaInfo {
	return s.schemas
}

var defaultSchemaStub = &stubSchemaProvider{}

func TestHandler_RedirectRoot(t *testing.T) {
	provider := &stubStatusProvider{
		liveness:  "ok",
		readiness: "ok",
	}
	h, err := NewHandler(provider, defaultSchemaStub)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

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
	h, err := NewHandler(provider, defaultSchemaStub)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

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
	h, err := NewHandler(provider, defaultSchemaStub)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestNewHandlerFromFS_ReturnsErrorOnBrokenTemplates(t *testing.T) {
	provider := &stubStatusProvider{liveness: "ok", readiness: "ok"}

	h, err := NewHandler(provider, defaultSchemaStub)
	if err != nil {
		t.Fatalf("NewHandler: unexpected error: %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil handler")
	}

	brokenFS := fstest.MapFS{
		"templates/layout.html": &fstest.MapFile{Data: []byte("{{.Invalid")},
		"templates/status.html": &fstest.MapFile{Data: []byte("")},
		"templates/schema.html": &fstest.MapFile{Data: []byte("")},
	}
	_, err = newHandlerFromFS(provider, defaultSchemaStub, brokenFS)
	if err == nil {
		t.Fatal("expected error for broken template FS")
	}
}

func TestHandler_StatusTemplateError(t *testing.T) {
	provider := &stubStatusProvider{liveness: "ok", readiness: "ok"}
	broken := template.Must(template.New("layout.html").Funcs(template.FuncMap{
		"fail": func() (string, error) { return "", errors.New("forced template failure") },
	}).Parse(`{{fail}}`))
	h := newHandler(provider, defaultSchemaStub, broken)

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when template execution fails, got %d", w.Code)
	}
}

func TestHandler_SchemaPage(t *testing.T) {
	status := &stubStatusProvider{liveness: "ok", readiness: "ok"}

	t.Run("empty state", func(t *testing.T) {
		schemas := &stubSchemaProvider{}
		h, err := NewHandler(status, schemas)
		if err != nil {
			t.Fatalf("NewHandler: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/schema", nil)
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
		if !strings.Contains(body, "Kein Schema vorhanden") {
			t.Error("empty hint missing")
		}
	})

	t.Run("with schemas", func(t *testing.T) {
		schemas := &stubSchemaProvider{schemas: []SchemaInfo{
			{Name: "locations", SDL: "type Location { id: ID! }", Version: 1},
			{Name: "tasks", SDL: "type Task { id: ID! }", Version: 3},
		}}
		h, err := NewHandler(status, schemas)
		if err != nil {
			t.Fatalf("NewHandler: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/schema", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		body := w.Body.String()
		for _, want := range []string{"locations", "tasks", "schema-name", "Hochladen", "Formatieren"} {
			if !strings.Contains(body, want) {
				t.Errorf("page missing %q", want)
			}
		}
	})

	t.Run("includes codemirror assets", func(t *testing.T) {
		schemas := &stubSchemaProvider{}
		h, err := NewHandler(status, schemas)
		if err != nil {
			t.Fatalf("NewHandler: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/schema", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		body := w.Body.String()
		for _, want := range []string{"codemirror.js", "codemirror.css", "graphql-web.js", "schema.js"} {
			if !strings.Contains(body, want) {
				t.Errorf("page missing asset reference %q", want)
			}
		}
	})
}

func TestHandler_SchemaTemplateError(t *testing.T) {
	status := &stubStatusProvider{liveness: "ok", readiness: "ok"}
	schemas := &stubSchemaProvider{}
	broken := template.Must(template.New("layout.html").Funcs(template.FuncMap{
		"fail": func() (string, error) { return "", errors.New("forced template failure") },
	}).Parse(`{{fail}}`))
	h := newHandler(status, schemas, broken)

	req := httptest.NewRequest(http.MethodGet, "/schema", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when template execution fails, got %d", w.Code)
	}
}

func TestHandler_SchemaListFragment(t *testing.T) {
	status := &stubStatusProvider{liveness: "ok", readiness: "ok"}

	t.Run("empty state", func(t *testing.T) {
		h, err := NewHandler(status, &stubSchemaProvider{})
		if err != nil {
			t.Fatalf("NewHandler: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/schema/list", nil)
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
		if !strings.Contains(body, "Kein Schema vorhanden") {
			t.Error("empty hint missing")
		}
		if strings.Contains(body, "<html") {
			t.Error("fragment must not contain full layout")
		}
	})

	t.Run("with schemas", func(t *testing.T) {
		schemas := &stubSchemaProvider{schemas: []SchemaInfo{
			{Name: "locations", SDL: "type Location { id: ID! }", Version: 1},
			{Name: "tasks", SDL: "type Task { id: ID! }", Version: 3},
		}}
		h, err := NewHandler(status, schemas)
		if err != nil {
			t.Fatalf("NewHandler: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/schema/list", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		body := w.Body.String()
		for _, want := range []string{"locations", "tasks"} {
			if !strings.Contains(body, want) {
				t.Errorf("fragment missing %q", want)
			}
		}
		if strings.Contains(body, "<html") {
			t.Error("fragment must not contain full layout")
		}
	})
}

func TestHandler_SchemaListTemplateError(t *testing.T) {
	status := &stubStatusProvider{liveness: "ok", readiness: "ok"}
	schemas := &stubSchemaProvider{}
	broken := template.Must(template.New("layout.html").Funcs(template.FuncMap{
		"fail": func() (string, error) { return "", errors.New("forced template failure") },
	}).Parse(`{{fail}}`))
	h := newHandler(status, schemas, broken)

	req := httptest.NewRequest(http.MethodGet, "/schema/list", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when template execution fails, got %d", w.Code)
	}
}

func TestHandler_StaticAssets(t *testing.T) {
	provider := &stubStatusProvider{
		liveness:  "ok",
		readiness: "ok",
	}
	h, err := NewHandler(provider, defaultSchemaStub)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

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
