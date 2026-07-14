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

package schema_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/tstangenberg/stratum/internal/schema"
	"github.com/tstangenberg/stratum/internal/system"
)

func TestRepositoryUpsertAndAll(t *testing.T) {
	ctx := context.Background()
	pool := startPool(t)
	if err := system.Migrate(ctx, pool); err != nil {
		t.Fatalf("system.Migrate: %v", err)
	}
	repository := schema.NewRepository(pool)

	createdAt := time.Date(2026, time.July, 13, 18, 0, 0, 0, time.UTC)
	first, err := repository.Upsert(ctx, schema.PersistedSchema{
		Name:      "devices",
		SDL:       `type Device { id: ID! serial: String! }`,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	})
	if err != nil {
		t.Fatalf("Upsert new schema: %v", err)
	}
	if first.Version != 1 {
		t.Errorf("new version = %d, want 1", first.Version)
	}
	if !first.CreatedAt.Equal(createdAt) {
		t.Errorf("new created_at = %v, want %v", first.CreatedAt, createdAt)
	}
	if !first.UpdatedAt.Equal(createdAt) {
		t.Errorf("new updated_at = %v, want %v", first.UpdatedAt, createdAt)
	}

	updatedAt := createdAt.Add(time.Hour)
	second, err := repository.Upsert(ctx, schema.PersistedSchema{
		Name:      "devices",
		SDL:       `type Device { id: ID! serial: String! firmware: String }`,
		CreatedAt: updatedAt,
		UpdatedAt: updatedAt,
	})
	if err != nil {
		t.Fatalf("Upsert existing schema: %v", err)
	}
	if second.Version != 2 {
		t.Errorf("updated version = %d, want 2", second.Version)
	}
	if !second.CreatedAt.Equal(createdAt) {
		t.Errorf("updated created_at = %v, want preserved %v", second.CreatedAt, createdAt)
	}
	if !second.UpdatedAt.Equal(updatedAt) {
		t.Errorf("updated updated_at = %v, want %v", second.UpdatedAt, updatedAt)
	}

	all, err := repository.All(ctx)
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("All returned %d schemas, want 1", len(all))
	}
	if all[0] != second {
		t.Errorf("All[0] = %+v, want %+v", all[0], second)
	}
}

type errorRow struct {
	err error
}

func (r errorRow) Scan(...any) error {
	return r.err
}

type rowQuerierFunc func(context.Context, string, ...any) pgx.Row

func (f rowQuerierFunc) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return f(ctx, sql, args...)
}

func TestRepositoryErrors(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("query failure")
	repository := schema.NewRepository(rowQuerierFunc(
		func(context.Context, string, ...any) pgx.Row {
			return errorRow{err: wantErr}
		},
	))

	if _, err := repository.Upsert(ctx, schema.PersistedSchema{Name: "devices"}); !errors.Is(err, wantErr) {
		t.Errorf("Upsert error = %v, want %v", err, wantErr)
	}
	if _, err := repository.All(ctx); !errors.Is(err, wantErr) {
		t.Errorf("All error = %v, want %v", err, wantErr)
	}
}

type jsonRow struct {
	data []byte
}

func (r jsonRow) Scan(dest ...any) error {
	target := dest[0].(*[]byte)
	*target = r.data
	return nil
}

type jsonQuerier struct {
	row jsonRow
}

func (q jsonQuerier) QueryRow(context.Context, string, ...any) pgx.Row {
	return q.row
}

func TestRepositoryAllInvalidJSON(t *testing.T) {
	repository := schema.NewRepository(jsonQuerier{row: jsonRow{data: []byte(`{`)}})

	if _, err := repository.All(context.Background()); err == nil {
		t.Fatal("All error = nil, want JSON decode error")
	}
}
