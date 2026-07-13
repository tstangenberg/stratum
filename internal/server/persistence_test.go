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

package server

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tstangenberg/stratum/internal/plugin"
	"github.com/tstangenberg/stratum/internal/plugin/scalar"
	"github.com/tstangenberg/stratum/internal/schema"
)

type schemaRepositoryStub struct {
	upsert func(context.Context, schema.PersistedSchema) (schema.PersistedSchema, error)
	all    func(context.Context) ([]schema.PersistedSchema, error)
}

func (s schemaRepositoryStub) Upsert(
	ctx context.Context,
	persisted schema.PersistedSchema,
) (schema.PersistedSchema, error) {
	return s.upsert(ctx, persisted)
}

func (s schemaRepositoryStub) All(ctx context.Context) ([]schema.PersistedSchema, error) {
	return s.all(ctx)
}

func TestInitializeWithoutDatabase(t *testing.T) {
	if err := NewStratumServer().Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize without database: %v", err)
	}
}

func TestInitializeErrors(t *testing.T) {
	wantErr := errors.New("injected failure")
	tests := []struct {
		name  string
		setup func(*StratumServer)
	}{
		{
			name: "migration",
			setup: func(srv *StratumServer) {
				srv.migrateSystem = func(context.Context, *pgxpool.Pool) error {
					return wantErr
				}
			},
		},
		{
			name: "list persisted schemas",
			setup: func(srv *StratumServer) {
				srv.migrateSystem = func(context.Context, *pgxpool.Pool) error {
					return nil
				}
				srv.schemaRepository = schemaRepositoryStub{
					all: func(context.Context) ([]schema.PersistedSchema, error) {
						return nil, wantErr
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewStratumServer().WithDB(new(pgxpool.Pool))
			tt.setup(srv)

			if err := srv.Initialize(context.Background()); !errors.Is(err, wantErr) {
				t.Fatalf("Initialize error = %v, want %v", err, wantErr)
			}
		})
	}
}

func TestInitializeLogsAndSkipsInvalidSchema(t *testing.T) {
	createdAt := time.Date(2026, time.July, 13, 18, 0, 0, 0, time.UTC)
	var logs bytes.Buffer
	srv := NewStratumServer().WithDB(new(pgxpool.Pool))
	srv.logger = log.New(&logs, "", 0)
	srv.migrateSystem = func(context.Context, *pgxpool.Pool) error {
		return nil
	}
	srv.schemaRepository = schemaRepositoryStub{
		all: func(context.Context) ([]schema.PersistedSchema, error) {
			return []schema.PersistedSchema{
				{
					Name:      "broken",
					SDL:       `type { broken`,
					Version:   1,
					CreatedAt: createdAt,
					UpdatedAt: createdAt,
				},
				{
					Name:      "devices",
					SDL:       `type Device { id: ID! serial: String! }`,
					Version:   3,
					CreatedAt: createdAt,
					UpdatedAt: createdAt.Add(time.Hour),
				},
			}, nil
		},
	}
	srv.buildSchemaHandler = func(
		*pgxpool.Pool,
		string,
		*schema.ParsedSchema,
		map[string]scalar.Plugin,
		[]plugin.QueryModifier,
		[]plugin.FilterPlugin,
		int,
	) (http.Handler, error) {
		return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), nil
	}

	if err := srv.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if _, ok := srv.schemas.Get("broken"); ok {
		t.Error("broken schema was loaded")
	}
	loaded, ok := srv.schemas.Get("devices")
	if !ok {
		t.Fatal("devices schema was not loaded")
	}
	if loaded.Version != 3 {
		t.Errorf("loaded version = %d, want 3", loaded.Version)
	}
	if !loaded.CreatedAt.Equal(createdAt) {
		t.Errorf("loaded created_at = %v, want %v", loaded.CreatedAt, createdAt)
	}
	if !strings.Contains(logs.String(), `load persisted schema "broken"`) {
		t.Errorf("logs = %q, want broken schema load error", logs.String())
	}
}

func TestInitializeLogsAndSkipsHandlerError(t *testing.T) {
	wantErr := errors.New("handler failure")
	var logs bytes.Buffer
	srv := NewStratumServer().WithDB(new(pgxpool.Pool))
	srv.logger = log.New(&logs, "", 0)
	srv.migrateSystem = func(context.Context, *pgxpool.Pool) error {
		return nil
	}
	srv.schemaRepository = schemaRepositoryStub{
		all: func(context.Context) ([]schema.PersistedSchema, error) {
			return []schema.PersistedSchema{{
				Name: "devices",
				SDL:  `type Device { id: ID! serial: String! }`,
			}}, nil
		},
	}
	srv.buildSchemaHandler = func(
		*pgxpool.Pool,
		string,
		*schema.ParsedSchema,
		map[string]scalar.Plugin,
		[]plugin.QueryModifier,
		[]plugin.FilterPlugin,
		int,
	) (http.Handler, error) {
		return nil, wantErr
	}

	if err := srv.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if _, ok := srv.schemas.Get("devices"); ok {
		t.Error("schema with handler error was loaded")
	}
	if !strings.Contains(logs.String(), `load persisted schema "devices"`) {
		t.Errorf("logs = %q, want handler load error", logs.String())
	}
}

func TestUpsertSchemaPersistError(t *testing.T) {
	wantErr := errors.New("persist failure")
	srv := NewStratumServer().WithDB(new(pgxpool.Pool))
	srv.createTable = func(
		context.Context,
		*pgxpool.Pool,
		string,
		schema.TypeDef,
		map[string]scalar.Plugin,
	) error {
		return nil
	}
	srv.addColumns = func(
		context.Context,
		*pgxpool.Pool,
		string,
		schema.TypeDef,
		map[string]scalar.Plugin,
	) error {
		return nil
	}
	srv.buildSchemaHandler = func(
		*pgxpool.Pool,
		string,
		*schema.ParsedSchema,
		map[string]scalar.Plugin,
		[]plugin.QueryModifier,
		[]plugin.FilterPlugin,
		int,
	) (http.Handler, error) {
		return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), nil
	}
	srv.schemaRepository = schemaRepositoryStub{
		upsert: func(
			context.Context,
			schema.PersistedSchema,
		) (schema.PersistedSchema, error) {
			return schema.PersistedSchema{}, wantErr
		},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/schemas/devices",
		strings.NewReader(`{"sdl":"type Device { id: ID! serial: String! }"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mustHandler(srv).ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 — body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), wantErr.Error()) {
		t.Errorf("body = %q, want persist error", w.Body.String())
	}
}
