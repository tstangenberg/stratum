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
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestName(t *testing.T) {
	p := New("secret")
	if p.Name() != "api-key-auth" {
		t.Fatalf("Name() = %q, want %q", p.Name(), "api-key-auth")
	}
}

func TestAuthenticate(t *testing.T) {
	const key = "my-secret-key"
	p := New(key)

	tests := []struct {
		name    string
		header  string
		allowed bool
	}{
		{"valid key", key, true},
		{"missing key", "", false},
		{"wrong key", "wrong", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("X-API-Key", tt.header)
			}
			result := p.Authenticate(req)
			if result.Allowed != tt.allowed {
				t.Fatalf("Authenticate() = %v, want %v", result.Allowed, tt.allowed)
			}
		})
	}
}
