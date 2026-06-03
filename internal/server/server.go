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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tstangenberg/stratum/internal/api"
	"github.com/tstangenberg/stratum/internal/plugin"
	"github.com/tstangenberg/stratum/internal/plugin/scalar"
	idscalar "github.com/tstangenberg/stratum/internal/plugin/scalar/id"
	stringscalar "github.com/tstangenberg/stratum/internal/plugin/scalar/string"
	"github.com/tstangenberg/stratum/internal/schema"
)

var errNotImplemented = errors.New("not implemented")

// StratumServer is the main server struct.
type StratumServer struct {
	healthPlugins []plugin.HealthPlugin
	db            *pgxpool.Pool
	schemas       *schema.Store
	scalars       map[string]scalar.Plugin
}

// NewStratumServer creates a new StratumServer with the given health plugins.
func NewStratumServer(plugins ...plugin.HealthPlugin) *StratumServer {
	return &StratumServer{
		healthPlugins: plugins,
		schemas:       schema.NewStore(),
		scalars: map[string]scalar.Plugin{
			"String": stringscalar.Plugin{},
			"ID":     idscalar.Plugin{},
		},
	}
}

// WithDB sets the PostgreSQL connection pool and returns the server for chaining.
func (s *StratumServer) WithDB(db *pgxpool.Pool) *StratumServer {
	s.db = db
	return s
}

func (s *StratumServer) Liveness(_ context.Context, _ api.LivenessRequestObject) (api.LivenessResponseObject, error) {
	return api.Liveness200JSONResponse{Status: api.LivenessResponseStatusOk}, nil
}

func (s *StratumServer) Readiness(ctx context.Context, _ api.ReadinessRequestObject) (api.ReadinessResponseObject, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	type result struct {
		name   string
		status plugin.HealthStatus
	}

	ch := make(chan result, len(s.healthPlugins))
	var wg sync.WaitGroup
	for _, p := range s.healthPlugins {
		wg.Add(1)
		p := p
		go func() {
			defer wg.Done()
			ch <- result{name: p.Name(), status: p.Check(ctx)}
		}()
	}
	wg.Wait()
	close(ch)

	components := make(map[string]api.ComponentHealth, len(s.healthPlugins))
	overall := api.Ok
	for r := range ch {
		status := api.ComponentHealthStatusOk
		if r.status.Status != plugin.StatusOK {
			status = api.ComponentHealthStatusError
			overall = api.Degraded
		}
		var details *map[string]interface{}
		if r.status.Details != nil {
			d := make(map[string]interface{}, len(r.status.Details))
			for k, v := range r.status.Details {
				d[k] = v
			}
			details = &d
		}
		components[r.name] = api.ComponentHealth{Status: status, Details: details}
	}

	if overall == api.Degraded {
		return api.Readiness503JSONResponse{
			Status:     api.Degraded,
			Components: components,
		}, nil
	}
	return api.Readiness200JSONResponse{
		Status:     api.Ok,
		Components: components,
	}, nil
}

func (s *StratumServer) Info(_ context.Context, _ api.InfoRequestObject) (api.InfoResponseObject, error) {
	return nil, errNotImplemented
}

func (s *StratumServer) ListSchemas(_ context.Context, _ api.ListSchemasRequestObject) (api.ListSchemasResponseObject, error) {
	return nil, errNotImplemented
}

func (s *StratumServer) DeleteSchema(_ context.Context, _ api.DeleteSchemaRequestObject) (api.DeleteSchemaResponseObject, error) {
	return nil, errNotImplemented
}

func (s *StratumServer) GetSchema(_ context.Context, _ api.GetSchemaRequestObject) (api.GetSchemaResponseObject, error) {
	return nil, errNotImplemented
}

func (s *StratumServer) UpsertSchema(ctx context.Context, req api.UpsertSchemaRequestObject) (api.UpsertSchemaResponseObject, error) {
	if s.db == nil {
		return nil, errNotImplemented
	}

	name := req.Name
	if !validSchemaName(name) {
		return api.UpsertSchema400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{
				Error:   "bad_request",
				Message: "schema name must match [a-z][a-z0-9_]{0,62}",
			},
		}, nil
	}

	ps, err := schema.ParseSDL(req.Body.Sdl)
	if err != nil {
		return api.UpsertSchema422JSONResponse{
			ValidationErrorJSONResponse: api.ValidationErrorJSONResponse{
				Error:   "validation_failed",
				Message: err.Error(),
			},
		}, nil
	}

	for _, t := range ps.Types {
		if err := schema.CreateTable(ctx, s.db, name, t, s.scalars); err != nil {
			return nil, fmt.Errorf("upsert schema %q: %w", name, err)
		}
	}

	h, err := schema.BuildHandler(s.db, name, ps, s.scalars)
	if err != nil {
		return nil, fmt.Errorf("upsert schema %q: build handler: %w", name, err)
	}

	now := time.Now()
	endpoint := "/graphql/" + name
	s.schemas.Set(name, &schema.Schema{
		Name:      name,
		SDL:       req.Body.Sdl,
		Parsed:    ps,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
		Handler:   h,
	})

	return api.UpsertSchema200JSONResponse{
		Name:            name,
		Status:          api.Applied,
		Version:         1,
		UpdatedAt:       now,
		GraphqlEndpoint: &endpoint,
	}, nil
}

func (s *StratumServer) GetSchemaStatus(_ context.Context, _ api.GetSchemaStatusRequestObject) (api.GetSchemaStatusResponseObject, error) {
	return nil, errNotImplemented
}

// validSchemaName reports whether name is a safe PostgreSQL identifier prefix.
func validSchemaName(name string) bool {
	if len(name) == 0 || len(name) > 63 {
		return false
	}
	for i, r := range name {
		if i == 0 && !(r >= 'a' && r <= 'z') {
			return false
		}
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '_' {
			continue
		}
		return false
	}
	return true
}

// serveGraphQL handles POST /graphql/{name} requests.
func (s *StratumServer) serveGraphQL(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	sc, ok := s.schemas.Get(name)
	if !ok {
		http.NotFound(w, r)
		return
	}
	sc.Handler.ServeHTTP(w, r)
}

// notImplementedHandler writes a consistent 501 JSON body.
func notImplementedHandler(w http.ResponseWriter, _ *http.Request, err error) {
	if !errors.Is(err, errNotImplemented) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   "not_implemented",
		"message": "this endpoint is not yet implemented",
	})
}

// Handler returns an http.Handler for all Stratum routes:
//
//	/api/           → OpenAPI-generated REST endpoints
//	/graphql/{name} → dynamic GraphQL endpoint per schema
func Handler(srv *StratumServer) http.Handler {
	strict := api.NewStrictHandlerWithOptions(srv, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		},
		ResponseErrorHandlerFunc: notImplementedHandler,
	})
	mux := http.NewServeMux()
	mux.Handle("/api/", api.Handler(strict))
	mux.HandleFunc("POST /graphql/{name}", srv.serveGraphQL)
	return mux
}
