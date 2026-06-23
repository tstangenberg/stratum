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

package apikey

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/tstangenberg/stratum/internal/plugin"
)

func init() {
	plugin.RegisterMiddleware(func() plugin.HTTPMiddleware {
		if p := FromEnv(); p != nil {
			return p
		}
		return nil
	})
}

// Plugin authenticates requests by comparing the X-API-Key header
// against a pre-shared key using constant-time comparison.
type Plugin struct {
	key string
}

// New creates an api-key-auth plugin that validates the X-API-Key header.
func New(key string) *Plugin {
	return &Plugin{key: key}
}

// FromEnv creates a plugin from the STRATUM_API_KEY environment variable.
// Returns nil when the variable is not set, which leaves auth disabled.
func FromEnv() *Plugin {
	key := os.Getenv("STRATUM_API_KEY")
	if key == "" {
		return nil
	}
	return &Plugin{key: key}
}

func (p *Plugin) Name() string { return "api-key-auth" }

// Priority returns the middleware position in the chain. Override via
// STRATUM_HTTP_MIDDLEWARE_API_KEY_AUTH_PRIORITY in stratum.yaml.
func (p *Plugin) Priority() int {
	if s := os.Getenv("STRATUM_HTTP_MIDDLEWARE_API_KEY_AUTH_PRIORITY"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 0 {
			return v
		}
	}
	return 100
}

func (p *Plugin) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("X-API-Key")
		if subtle.ConstantTimeCompare([]byte(got), []byte(p.key)) != 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":   "unauthorized",
				"message": "valid API key required",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Compile-time check that *Plugin satisfies plugin.HTTPMiddleware.
var _ plugin.HTTPMiddleware = (*Plugin)(nil)
