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

package database_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/tstangenberg/stratum/internal/plugin"
	"github.com/tstangenberg/stratum/internal/plugin/database"
)

type mockPinger struct{ err error }

func (m *mockPinger) PingContext(_ context.Context) error { return m.err }

func TestDatabasePlugin_Name(t *testing.T) {
	p := database.New(&mockPinger{})
	if p.Name() != "database" {
		t.Errorf("Name() = %q, want %q", p.Name(), "database")
	}
}

func TestDatabasePlugin_CheckOK(t *testing.T) {
	p := database.New(&mockPinger{err: nil})
	status := p.Check(context.Background())
	if status.Status != plugin.StatusOK {
		t.Errorf("Check() status = %q, want %q", status.Status, plugin.StatusOK)
	}
	if status.Details != nil {
		t.Errorf("Check() details should be nil on success, got %v", status.Details)
	}
}

func TestDatabasePlugin_CheckError(t *testing.T) {
	p := database.New(&mockPinger{err: errors.New("connection refused")})
	status := p.Check(context.Background())
	if status.Status != plugin.StatusError {
		t.Errorf("Check() status = %q, want %q", status.Status, plugin.StatusError)
	}
	if _, ok := status.Details["error"]; !ok {
		t.Error("Check() details must contain 'error' key")
	}
}

func TestDatabasePlugin_CheckDoesNotExposeCredentials(t *testing.T) {
	dsn := "postgres://admin:supersecret@localhost:5432/mydb"
	p := database.New(&mockPinger{err: fmt.Errorf("failed to connect: %s", dsn)})
	status := p.Check(context.Background())
	msg, ok := status.Details["error"].(string)
	if !ok {
		t.Fatal("error detail should be a string")
	}
	if strings.Contains(msg, "supersecret") || strings.Contains(msg, "admin:") {
		t.Errorf("error detail must not expose credentials, got: %q", msg)
	}
}
