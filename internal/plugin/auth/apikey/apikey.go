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
	"net/http"

	"github.com/tstangenberg/stratum/internal/plugin"
)

// Plugin authenticates requests by comparing the X-API-Key header
// against a pre-shared key using constant-time comparison.
type Plugin struct {
	key string
}

// New creates an api-key-auth plugin that validates the X-API-Key header.
func New(key string) *Plugin {
	return &Plugin{key: key}
}

func (p *Plugin) Name() string { return "api-key-auth" }

func (p *Plugin) Authenticate(r *http.Request) plugin.AuthResult {
	got := r.Header.Get("X-API-Key")
	if subtle.ConstantTimeCompare([]byte(got), []byte(p.key)) == 1 {
		return plugin.AuthResult{Allowed: true}
	}
	return plugin.AuthResult{Allowed: false}
}
