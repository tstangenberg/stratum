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
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
)

// StatusProvider supplies data for the status page.
type StatusProvider interface {
	HealthStatus(ctx context.Context) HealthResult
	Plugins() []PluginInfo
}

// HealthResult holds the aggregated health check results.
type HealthResult struct {
	Liveness   string
	Readiness  string
	Components map[string]string
}

// PluginInfo describes a registered plugin.
type PluginInfo struct {
	Name string
	Type string
}

// Handler serves the embedded web UI.
type Handler struct {
	provider StatusProvider
	mux      *http.ServeMux
	tmpl     *template.Template
}

// NewHandler creates a UI handler backed by the given status provider.
func NewHandler(provider StatusProvider) (*Handler, error) {
	return newHandlerFromFS(provider, templates)
}

func newHandlerFromFS(provider StatusProvider, tmplFS fs.FS) (*Handler, error) {
	tmpl, err := template.ParseFS(tmplFS, "templates/layout.html", "templates/status.html")
	if err != nil {
		return nil, fmt.Errorf("ui: parse templates: %w", err)
	}
	return newHandler(provider, tmpl), nil
}

func newHandler(provider StatusProvider, tmpl *template.Template) *Handler {
	h := &Handler{provider: provider, tmpl: tmpl}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /status", h.handleStatus)
	mux.HandleFunc("GET /static/", h.handleStatic)
	mux.HandleFunc("GET /{$}", h.handleRoot)
	h.mux = mux

	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if the request matches any registered route
	_, pattern := h.mux.Handler(r)
	if pattern == "" {
		http.NotFound(w, r)
		return
	}
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) handleRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/ui/status", http.StatusMovedPermanently)
}

func (h *Handler) handleStatus(w http.ResponseWriter, r *http.Request) {
	health := h.provider.HealthStatus(r.Context())
	plugins := h.provider.Plugins()

	data := struct {
		Health  HealthResult
		Plugins []PluginInfo
	}{
		Health:  health,
		Plugins: plugins,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		http.Error(w, fmt.Sprintf("ui: render template: %v", err), http.StatusInternalServerError)
	}
}

func (h *Handler) handleStatic(w http.ResponseWriter, r *http.Request) {
	// Strip the leading /static/ to get the file path within the static dir
	path := strings.TrimPrefix(r.URL.Path, "/static/")
	data, err := static.ReadFile("static/" + path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	switch {
	case strings.HasSuffix(path, ".js"):
		w.Header().Set("Content-Type", "application/javascript")
	case strings.HasSuffix(path, ".css"):
		w.Header().Set("Content-Type", "text/css")
	}
	_, _ = w.Write(data)
}
