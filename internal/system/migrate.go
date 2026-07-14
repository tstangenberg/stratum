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
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type migrationRunner interface {
	Up(context.Context) error
}

type gooseRunner struct {
	provider *goose.Provider
}

func (r gooseRunner) Up(ctx context.Context) error {
	_, err := r.provider.Up(ctx)
	return err
}

// Migrate applies all pending Stratum system-table migrations.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	return migrate(ctx, pool, newMigrationRunner)
}

func migrate(
	ctx context.Context,
	pool *pgxpool.Pool,
	newRunner func(*sql.DB) (migrationRunner, error),
) error {
	if _, err := pool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS stratum_system`); err != nil {
		return fmt.Errorf("system: create schema: %w", err)
	}

	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	runner, err := newRunner(db)
	if err != nil {
		return fmt.Errorf("system: create migration runner: %w", err)
	}
	if err := runner.Up(ctx); err != nil {
		return fmt.Errorf("system: apply migrations: %w", err)
	}
	return nil
}

func newMigrationRunner(db *sql.DB) (migrationRunner, error) {
	return newMigrationRunnerFromFS(db, migrationFiles)
}

func newMigrationRunnerFromFS(db *sql.DB, files fs.FS) (migrationRunner, error) {
	migrations, err := fs.Sub(files, "migrations")
	if err != nil {
		return nil, fmt.Errorf("system: open migrations: %w", err)
	}
	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		db,
		migrations,
		goose.WithTableName("stratum_system.goose_db_version"),
		goose.WithLogger(log.New(io.Discard, "", 0)),
	)
	if err != nil {
		return nil, fmt.Errorf("system: configure migrations: %w", err)
	}
	return gooseRunner{provider: provider}, nil
}
