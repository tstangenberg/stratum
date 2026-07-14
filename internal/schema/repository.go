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

package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// PersistedSchema is the durable metadata stored for a live schema.
type PersistedSchema struct {
	Name      string    `json:"name"`
	SDL       string    `json:"sdl"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type rowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// Repository persists schema metadata in PostgreSQL.
type Repository struct {
	db rowQuerier
}

// NewRepository creates a PostgreSQL schema repository.
func NewRepository(db rowQuerier) *Repository {
	return &Repository{db: db}
}

// Upsert saves a schema, incrementing its durable version on re-upload.
func (r *Repository) Upsert(ctx context.Context, persisted PersistedSchema) (PersistedSchema, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO stratum_system.stratum_schemas (
			name, sdl, version, created_at, updated_at
		)
		VALUES ($1, $2, 1, $3, $4)
		ON CONFLICT (name) DO UPDATE
		SET sdl = EXCLUDED.sdl,
		    version = stratum_system.stratum_schemas.version + 1,
		    updated_at = EXCLUDED.updated_at
		RETURNING name, sdl, version, created_at, updated_at`,
		persisted.Name,
		persisted.SDL,
		persisted.CreatedAt,
		persisted.UpdatedAt,
	)

	var saved PersistedSchema
	if err := row.Scan(
		&saved.Name,
		&saved.SDL,
		&saved.Version,
		&saved.CreatedAt,
		&saved.UpdatedAt,
	); err != nil {
		return PersistedSchema{}, fmt.Errorf("schema repository: upsert %q: %w", persisted.Name, err)
	}
	return saved, nil
}

// All returns every persisted schema ordered by name.
func (r *Repository) All(ctx context.Context) ([]PersistedSchema, error) {
	row := r.db.QueryRow(ctx, `
		SELECT COALESCE(
			jsonb_agg(
				jsonb_build_object(
					'name', name,
					'sdl', sdl,
					'version', version,
					'created_at', created_at,
					'updated_at', updated_at
				)
				ORDER BY name
			),
			'[]'::jsonb
		)
		FROM stratum_system.stratum_schemas`)

	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, fmt.Errorf("schema repository: list: %w", err)
	}

	var schemas []PersistedSchema
	if err := json.Unmarshal(data, &schemas); err != nil {
		return nil, fmt.Errorf("schema repository: decode list: %w", err)
	}
	return schemas, nil
}
