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
	"path/filepath"
	"strings"
	"testing"
)

func TestParseComment(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		wantDesc string
		wantDef  string
	}{
		{
			name:     "description and default",
			text:     "HTTP listen address.\nDefault: :8080",
			wantDesc: "HTTP listen address.",
			wantDef:  ":8080",
		},
		{
			name:     "multiline description",
			text:     "First line.\nSecond line.\nDefault: none",
			wantDesc: "First line. Second line.",
			wantDef:  "none",
		},
		{
			name:     "no default line",
			text:     "Just a description.",
			wantDesc: "Just a description.",
			wantDef:  "none",
		},
		{
			name:     "empty comment",
			text:     "",
			wantDesc: "",
			wantDef:  "none",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDesc, gotDef := parseComment(tt.text)
			if gotDesc != tt.wantDesc {
				t.Errorf("description: got %q, want %q", gotDesc, tt.wantDesc)
			}
			if gotDef != tt.wantDef {
				t.Errorf("default: got %q, want %q", gotDef, tt.wantDef)
			}
		})
	}
}

func TestCollect_singleConst(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "env.go"), []byte(`package foo

// HTTP listen address.
// Default: :8080
const EnvServerAddr = "STRATUM_SERVER_ADDR"

// Not a STRATUM var.
const Other = "OTHER"

const unexported = "STRATUM_IGNORED"
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	vars, err := collect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(vars) != 1 {
		t.Fatalf("got %d vars, want 1; vars: %+v", len(vars), vars)
	}
	v := vars[0]
	if v.Name != "STRATUM_SERVER_ADDR" {
		t.Errorf("name: got %q, want %q", v.Name, "STRATUM_SERVER_ADDR")
	}
	if v.Description != "HTTP listen address." {
		t.Errorf("description: got %q, want %q", v.Description, "HTTP listen address.")
	}
	if v.Default != ":8080" {
		t.Errorf("default: got %q, want %q", v.Default, ":8080")
	}
}

func TestCollect_groupedConst(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "env.go"), []byte(`package foo

const (
	// Default limit.
	// Default: 100
	EnvDefaultLimit = "STRATUM_PLUGINS_PAGINATION_DEFAULT_LIMIT"

	// Max limit.
	// Default: 1000
	EnvMaxLimit = "STRATUM_PLUGINS_PAGINATION_MAX_LIMIT"
)
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	vars, err := collect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(vars) != 2 {
		t.Fatalf("got %d vars, want 2; vars: %+v", len(vars), vars)
	}
}

func TestCollect_skipsTestFiles(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "env_test.go"), []byte(`package foo

const EnvTest = "STRATUM_TEST"
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	vars, err := collect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(vars) != 0 {
		t.Errorf("got %d vars from test file, want 0", len(vars))
	}
}

func TestCollect_groupDocFallback(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "env.go"), []byte(`package foo

// Group description.
// Default: group-default
const (
	EnvFoo = "STRATUM_FOO"
)
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	vars, err := collect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(vars) != 1 {
		t.Fatalf("got %d vars, want 1", len(vars))
	}
	if vars[0].Description != "Group description." {
		t.Errorf("description: got %q, want %q", vars[0].Description, "Group description.")
	}
	if vars[0].Default != "group-default" {
		t.Errorf("default: got %q, want %q", vars[0].Default, "group-default")
	}
}

func TestRun_happyPath(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "env.go"), []byte(`package foo

// Listen address.
// Default: :8080
const EnvAddr = "STRATUM_SERVER_ADDR"
`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out.md")
	if err := run(dir, out); err != nil {
		t.Fatalf("run: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "STRATUM_SERVER_ADDR") {
		t.Error("output missing STRATUM_SERVER_ADDR")
	}
}

func TestRun_collectError(t *testing.T) {
	if err := run("/nonexistent/root/path", t.TempDir()+"/out.md"); err == nil {
		t.Fatal("want error for nonexistent root, got nil")
	}
}

func TestRun_writeError(t *testing.T) {
	if err := run(t.TempDir(), "/nonexistent/dir/out.md"); err == nil {
		t.Fatal("want error for unwritable output path, got nil")
	}
}

func TestCollect_skipsVendorDir(t *testing.T) {
	dir := t.TempDir()
	vendorDir := filepath.Join(dir, "vendor", "pkg")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatal(err)
	}
	err := os.WriteFile(filepath.Join(vendorDir, "env.go"), []byte(`package pkg

const EnvVendored = "STRATUM_VENDORED"
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	vars, err := collect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(vars) != 0 {
		t.Errorf("got %d vars from vendor dir, want 0", len(vars))
	}
}
