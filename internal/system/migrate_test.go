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

package system

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type migrationRunnerFunc func(context.Context) error

func (f migrationRunnerFunc) Up(ctx context.Context) error {
	return f(ctx)
}

func startMigrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	pgc, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = pgc.Terminate(ctx) })

	dsn, err := pgc.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestMigrateCreatesSchemaTable(t *testing.T) {
	ctx := context.Background()
	pool := startMigrationPool(t)

	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate idempotently: %v", err)
	}

	rows, err := pool.Query(ctx, `
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_schema = 'stratum_system'
		  AND table_name = 'stratum_schemas'
		ORDER BY ordinal_position`)
	if err != nil {
		t.Fatalf("query columns: %v", err)
	}
	defer rows.Close()

	type column struct {
		name       string
		dataType   string
		isNullable string
	}
	var got []column
	for rows.Next() {
		var col column
		if err := rows.Scan(&col.name, &col.dataType, &col.isNullable); err != nil {
			t.Fatalf("scan column: %v", err)
		}
		got = append(got, col)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate columns: %v", err)
	}

	want := []column{
		{name: "name", dataType: "text", isNullable: "NO"},
		{name: "sdl", dataType: "text", isNullable: "NO"},
		{name: "version", dataType: "integer", isNullable: "NO"},
		{name: "created_at", dataType: "timestamp with time zone", isNullable: "NO"},
		{name: "updated_at", dataType: "timestamp with time zone", isNullable: "NO"},
	}
	if len(got) != len(want) {
		t.Fatalf("columns = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("column %d = %+v, want %+v", i, got[i], want[i])
		}
	}

	var versionTableExists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'stratum_system'
			  AND table_name = 'goose_db_version'
		)`).Scan(&versionTableExists); err != nil {
		t.Fatalf("query version table: %v", err)
	}
	if !versionTableExists {
		t.Fatal("stratum_system.goose_db_version does not exist")
	}
}

func TestMigrateErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("bootstrap schema", func(t *testing.T) {
		pool := startMigrationPool(t)
		pool.Close()

		if err := Migrate(ctx, pool); err == nil {
			t.Fatal("Migrate error = nil, want bootstrap error")
		}
	})

	t.Run("create provider", func(t *testing.T) {
		pool := startMigrationPool(t)
		wantErr := errors.New("provider failure")

		err := migrate(ctx, pool, func(*sql.DB) (migrationRunner, error) {
			return nil, wantErr
		})
		if !errors.Is(err, wantErr) {
			t.Fatalf("migrate error = %v, want %v", err, wantErr)
		}
	})

	t.Run("apply migrations", func(t *testing.T) {
		pool := startMigrationPool(t)
		wantErr := errors.New("up failure")

		err := migrate(ctx, pool, func(*sql.DB) (migrationRunner, error) {
			return migrationRunnerFunc(func(context.Context) error {
				return wantErr
			}), nil
		})
		if !errors.Is(err, wantErr) {
			t.Fatalf("migrate error = %v, want %v", err, wantErr)
		}
	})
}

type failingSubFS struct {
	err error
}

func (f failingSubFS) Open(string) (fs.File, error) {
	return nil, fs.ErrNotExist
}

func (f failingSubFS) Sub(string) (fs.FS, error) {
	return nil, f.err
}

func TestNewMigrationRunnerErrors(t *testing.T) {
	t.Run("open migrations", func(t *testing.T) {
		wantErr := errors.New("sub failure")

		if _, err := newMigrationRunnerFromFS(new(sql.DB), failingSubFS{err: wantErr}); !errors.Is(err, wantErr) {
			t.Fatalf("newMigrationRunnerFromFS error = %v, want %v", err, wantErr)
		}
	})

	t.Run("configure provider", func(t *testing.T) {
		files := fstest.MapFS{
			"migrations/README.md": {Data: []byte("no migrations")},
		}

		if _, err := newMigrationRunnerFromFS(new(sql.DB), files); err == nil {
			t.Fatal("newMigrationRunnerFromFS error = nil, want no-migrations error")
		}
	})
}
