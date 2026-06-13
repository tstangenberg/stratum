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
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load resolves and reads the configuration file, expanding every YAML leaf
// value to an environment variable using the naming rule: path segments joined
// with "_", uppercased, prefixed with "STRATUM_". Environment variables already
// set are never overwritten.
//
// File resolution order: STRATUM_CONFIG env var → ./stratum.yaml → no file (not
// an error). If STRATUM_CONFIG is set but the file does not exist, Load returns
// an error.
func Load() error {
	path, err := resolveConfigPath()
	if err != nil {
		return err
	}
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("config: read %q: %w", path, err)
	}

	var root map[string]any
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("config: parse %q: %w", path, err)
	}

	expand(root, "STRATUM")
	return nil
}

func resolveConfigPath() (string, error) {
	if p := os.Getenv("STRATUM_CONFIG"); p != "" {
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("config: file %q not found: %w", p, err)
		}
		return p, nil
	}
	if _, err := os.Stat("stratum.yaml"); err == nil {
		return "stratum.yaml", nil
	}
	return "", nil
}

func expand(node map[string]any, prefix string) {
	for key, val := range node {
		envKey := prefix + "_" + strings.ToUpper(key)
		switch v := val.(type) {
		case map[string]any:
			expand(v, envKey)
		case []any:
			setIfAbsent(envKey, joinList(v))
		default:
			setIfAbsent(envKey, fmt.Sprintf("%v", v))
		}
	}
}

func setIfAbsent(key, value string) {
	if _, exists := os.LookupEnv(key); exists {
		return
	}
	os.Setenv(key, value)
}

func joinList(items []any) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%v", item))
	}
	return strings.Join(parts, ",")
}
