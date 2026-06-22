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

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_YamlFileExpandsEnvVars(t *testing.T) {
	dir := t.TempDir()
	yaml := filepath.Join(dir, "stratum.yaml")
	content := "server:\n  addr: \":9090\"\ndatabase:\n  url: \"postgres://localhost/test\"\n"
	if err := os.WriteFile(yaml, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Clear any pre-existing values
	os.Unsetenv("STRATUM_SERVER_ADDR")
	os.Unsetenv("STRATUM_DATABASE_URL")
	t.Cleanup(func() {
		os.Unsetenv("STRATUM_SERVER_ADDR")
		os.Unsetenv("STRATUM_DATABASE_URL")
	})

	t.Setenv("STRATUM_CONFIG", yaml)

	if err := Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	tests := []struct {
		envVar string
		want   string
	}{
		{"STRATUM_SERVER_ADDR", ":9090"},
		{"STRATUM_DATABASE_URL", "postgres://localhost/test"},
	}
	for _, tt := range tests {
		got := os.Getenv(tt.envVar)
		if got != tt.want {
			t.Errorf("%s = %q, want %q", tt.envVar, got, tt.want)
		}
	}
}

func TestLoad_EnvVarNeverOverwritten(t *testing.T) {
	dir := t.TempDir()
	yaml := filepath.Join(dir, "stratum.yaml")
	content := "server:\n  addr: \":9090\"\n"
	if err := os.WriteFile(yaml, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("STRATUM_CONFIG", yaml)
	t.Setenv("STRATUM_SERVER_ADDR", ":7777")

	if err := Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	got := os.Getenv("STRATUM_SERVER_ADDR")
	if got != ":7777" {
		t.Errorf("STRATUM_SERVER_ADDR = %q, want %q (env var should win)", got, ":7777")
	}
}

func TestLoad_NoFileIsNotAnError(t *testing.T) {
	os.Unsetenv("STRATUM_CONFIG")
	t.Chdir(t.TempDir())

	if err := Load(); err != nil {
		t.Fatalf("Load() should not error when no config file exists: %v", err)
	}
}

func TestLoad_StratumConfigEnvVar(t *testing.T) {
	dir := t.TempDir()
	yaml := filepath.Join(dir, "custom.yaml")
	content := "server:\n  addr: \":3333\"\n"
	if err := os.WriteFile(yaml, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	os.Unsetenv("STRATUM_SERVER_ADDR")
	t.Cleanup(func() { os.Unsetenv("STRATUM_SERVER_ADDR") })
	t.Setenv("STRATUM_CONFIG", yaml)

	if err := Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	got := os.Getenv("STRATUM_SERVER_ADDR")
	if got != ":3333" {
		t.Errorf("STRATUM_SERVER_ADDR = %q, want %q", got, ":3333")
	}
}

func TestLoad_FallbackToStratumYamlInWorkingDir(t *testing.T) {
	dir := t.TempDir()
	yaml := filepath.Join(dir, "stratum.yaml")
	content := "server:\n  addr: \":4444\"\n"
	if err := os.WriteFile(yaml, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	os.Unsetenv("STRATUM_CONFIG")
	os.Unsetenv("STRATUM_SERVER_ADDR")
	t.Cleanup(func() { os.Unsetenv("STRATUM_SERVER_ADDR") })

	t.Chdir(dir)

	if err := Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	got := os.Getenv("STRATUM_SERVER_ADDR")
	if got != ":4444" {
		t.Errorf("STRATUM_SERVER_ADDR = %q, want %q", got, ":4444")
	}
}

func TestLoad_ListsAreCommaJoined(t *testing.T) {
	dir := t.TempDir()
	yaml := filepath.Join(dir, "stratum.yaml")
	content := "plugins:\n  auth:\n    api_keys:\n      - key1\n      - key2\n      - key3\n"
	if err := os.WriteFile(yaml, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	os.Unsetenv("STRATUM_PLUGINS_AUTH_API_KEYS")
	t.Cleanup(func() { os.Unsetenv("STRATUM_PLUGINS_AUTH_API_KEYS") })
	t.Setenv("STRATUM_CONFIG", yaml)

	if err := Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	got := os.Getenv("STRATUM_PLUGINS_AUTH_API_KEYS")
	if got != "key1,key2,key3" {
		t.Errorf("STRATUM_PLUGINS_AUTH_API_KEYS = %q, want %q", got, "key1,key2,key3")
	}
}

func TestLoad_DeepNesting(t *testing.T) {
	dir := t.TempDir()
	yaml := filepath.Join(dir, "stratum.yaml")
	content := "a:\n  b:\n    c: \"deep\"\n"
	if err := os.WriteFile(yaml, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	os.Unsetenv("STRATUM_A_B_C")
	t.Cleanup(func() { os.Unsetenv("STRATUM_A_B_C") })
	t.Setenv("STRATUM_CONFIG", yaml)

	if err := Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	got := os.Getenv("STRATUM_A_B_C")
	if got != "deep" {
		t.Errorf("STRATUM_A_B_C = %q, want %q", got, "deep")
	}
}

func TestLoad_StratumConfigPointsToNonexistentFile(t *testing.T) {
	t.Setenv("STRATUM_CONFIG", "/nonexistent/path/stratum.yaml")

	err := Load()
	if err == nil {
		t.Fatal("Load() should return error when STRATUM_CONFIG points to missing file")
	}
}

func TestLoad_InvalidYaml(t *testing.T) {
	dir := t.TempDir()
	yaml := filepath.Join(dir, "stratum.yaml")
	if err := os.WriteFile(yaml, []byte(":\n  :\n\t broken"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("STRATUM_CONFIG", yaml)

	err := Load()
	if err == nil {
		t.Fatal("Load() should return error for invalid YAML")
	}
}

func TestLoad_UnreadableFile(t *testing.T) {
	dir := t.TempDir()
	yaml := filepath.Join(dir, "stratum.yaml")
	if err := os.WriteFile(yaml, []byte("server:\n  addr: \":9090\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Make the file unreadable
	os.Chmod(yaml, 0o000)
	t.Cleanup(func() { os.Chmod(yaml, 0o644) })

	t.Setenv("STRATUM_CONFIG", yaml)

	err := Load()
	if err == nil {
		t.Fatal("Load() should return error for unreadable file")
	}
}

func TestLoad_HyphensInKeysConvertedToUnderscores(t *testing.T) {
	dir := t.TempDir()
	yaml := filepath.Join(dir, "stratum.yaml")
	content := "http-middleware:\n  api-key-auth:\n    priority: \"100\"\n"
	if err := os.WriteFile(yaml, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	os.Unsetenv("STRATUM_HTTP_MIDDLEWARE_API_KEY_AUTH_PRIORITY")
	t.Cleanup(func() { os.Unsetenv("STRATUM_HTTP_MIDDLEWARE_API_KEY_AUTH_PRIORITY") })
	t.Setenv("STRATUM_CONFIG", yaml)

	if err := Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	got := os.Getenv("STRATUM_HTTP_MIDDLEWARE_API_KEY_AUTH_PRIORITY")
	if got != "100" {
		t.Errorf("STRATUM_HTTP_MIDDLEWARE_API_KEY_AUTH_PRIORITY = %q, want %q", got, "100")
	}
}
