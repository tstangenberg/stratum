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
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	ensureDockerHost()
	os.Exit(m.Run())
}

// ensureDockerHost reads the active Docker context to find the socket path and
// sets DOCKER_HOST when it is not already present. This allows testcontainers-go
// to locate non-standard Docker providers such as OrbStack without manual setup.
func ensureDockerHost() {
	if os.Getenv("DOCKER_HOST") != "" {
		return
	}
	out, err := exec.Command("docker", "context", "inspect", "--format",
		`{{(index .Endpoints "docker").Host}}`).Output()
	if err != nil {
		return
	}
	if host := strings.TrimSpace(string(out)); host != "" {
		os.Setenv("DOCKER_HOST", host)
	}
}
