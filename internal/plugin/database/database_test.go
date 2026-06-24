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

func TestFromEnv_ReturnsNilWhenUnset(t *testing.T) {
	t.Setenv("STRATUM_DATABASE_URL", "")
	if p := database.FromEnv(); p != nil {
		t.Fatal("FromEnv() should return nil when STRATUM_DATABASE_URL is empty")
	}
}

func TestFromEnv_ReturnsNilOnInvalidDSN(t *testing.T) {
	t.Setenv("STRATUM_DATABASE_URL", "not-a-valid-dsn://:::broken")
	if p := database.FromEnv(); p != nil {
		t.Fatal("FromEnv() should return nil on invalid DSN")
	}
}

type mockPinger struct{ err error }

func (m *mockPinger) Ping(_ context.Context) error { return m.err }

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

func TestFromEnv_ReturnsPluginOnValidDSN(t *testing.T) {
	t.Setenv("STRATUM_DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	t.Cleanup(func() { database.ClosePool() })
	p := database.FromEnv()
	if p == nil {
		t.Fatal("FromEnv() should return non-nil for a valid DSN")
	}
}

func TestFactory_ReturnsNilWhenUnset(t *testing.T) {
	t.Setenv("STRATUM_DATABASE_URL", "")
	if p := database.Factory(); p != nil {
		t.Fatal("Factory() should return nil when STRATUM_DATABASE_URL is unset")
	}
}

func TestFactory_ReturnsPluginWhenSet(t *testing.T) {
	t.Setenv("STRATUM_DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	t.Cleanup(func() { database.ClosePool() })
	p := database.Factory()
	if p == nil {
		t.Fatal("Factory() should return non-nil when STRATUM_DATABASE_URL is set")
	}
	if p.Name() != "database" {
		t.Fatalf("Factory().Name() = %q, want %q", p.Name(), "database")
	}
}

func TestInit_RegistersFactory(t *testing.T) {
	restore := plugin.ResetHealthRegistryForTesting()
	t.Cleanup(restore)
	plugin.RegisterHealthPlugin(database.Factory)
	t.Setenv("STRATUM_DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	t.Cleanup(func() { database.ClosePool() })
	ps := plugin.BuildHealthPlugins()
	found := false
	for _, p := range ps {
		if p.Name() == "database" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected database plugin when factory registered and DSN set")
	}
}

func TestPool_ReturnsNilBeforeFromEnv(t *testing.T) {
	database.ClosePool()
	t.Cleanup(func() { database.ClosePool() })
	if p := database.Pool(); p != nil {
		t.Fatal("Pool() should return nil before FromEnv is called")
	}
}

func TestPool_ReturnsPoolAfterFromEnv(t *testing.T) {
	t.Cleanup(func() { database.ClosePool() })
	t.Setenv("STRATUM_DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	database.FromEnv()
	if p := database.Pool(); p == nil {
		t.Fatal("Pool() should return non-nil after FromEnv with valid DSN")
	}
}

func TestClosePool_ClosesAndNilsPool(t *testing.T) {
	t.Setenv("STRATUM_DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	database.FromEnv()
	database.ClosePool()
	if p := database.Pool(); p != nil {
		t.Fatal("Pool() should return nil after ClosePool")
	}
}

func TestClosePool_NoopWhenNil(t *testing.T) {
	database.ClosePool()
	database.ClosePool()
}
