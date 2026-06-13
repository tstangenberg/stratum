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

package e2e

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "stratum")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/stratum")
	cmd.Dir = filepath.Join("..")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build binary: %v\n%s", err, out)
	}
	return bin
}

func TestConfigYamlBindsAddr(t *testing.T) {
	bin := buildBinary(t)
	port := freePort(t)
	addr := fmt.Sprintf(":%d", port)

	// Write a stratum.yaml with server.addr
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "stratum.yaml")
	yamlContent := fmt.Sprintf("server:\n  addr: %q\n", addr)
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin)
	cmd.Env = append(os.Environ(), "STRATUM_CONFIG="+yamlPath)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start binary: %v", err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })

	// Poll until the server is ready or timeout.
	target := fmt.Sprintf("http://127.0.0.1:%d/api/v1/health/live", port)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(target)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return // success
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("server did not start on %s within timeout", addr)
}

func TestConfigEnvVarOverridesYaml(t *testing.T) {
	bin := buildBinary(t)
	yamlPort := freePort(t)
	envPort := freePort(t)

	// Write a stratum.yaml with one address
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "stratum.yaml")
	yamlContent := fmt.Sprintf("server:\n  addr: %q\n", fmt.Sprintf(":%d", yamlPort))
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Override with env var — env var should win
	envAddr := fmt.Sprintf(":%d", envPort)
	cmd := exec.CommandContext(ctx, bin)
	cmd.Env = append(os.Environ(),
		"STRATUM_CONFIG="+yamlPath,
		"STRATUM_SERVER_ADDR="+envAddr,
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start binary: %v", err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })

	// The server should listen on envPort, NOT yamlPort
	target := fmt.Sprintf("http://127.0.0.1:%d/api/v1/health/live", envPort)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(target)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return // success — env var won
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("server did not start on env-overridden address %s within timeout", envAddr)
}
