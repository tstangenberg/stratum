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

package main

import (
	"os"
	"testing"
)

func TestResolveAddr_DefaultsTo8080(t *testing.T) {
	os.Unsetenv("STRATUM_SERVER_ADDR")
	got := resolveAddr()
	if got != ":8080" {
		t.Errorf("resolveAddr() = %q, want %q", got, ":8080")
	}
}

func TestResolveAddr_UsesEnvVar(t *testing.T) {
	os.Setenv("STRATUM_SERVER_ADDR", ":9090")
	defer os.Unsetenv("STRATUM_SERVER_ADDR")
	got := resolveAddr()
	if got != ":9090" {
		t.Errorf("resolveAddr() = %q, want %q", got, ":9090")
	}
}

func TestResolveMaxListLimit_Default(t *testing.T) {
	os.Unsetenv("STRATUM_SERVER_LIST_MAX_LIMIT")
	got := resolveMaxListLimit()
	if got != 0 {
		t.Errorf("resolveMaxListLimit() = %d, want 0", got)
	}
}

func TestResolveMaxListLimit_FromEnv(t *testing.T) {
	t.Setenv("STRATUM_SERVER_LIST_MAX_LIMIT", "500")
	got := resolveMaxListLimit()
	if got != 500 {
		t.Errorf("resolveMaxListLimit() = %d, want 500", got)
	}
}
