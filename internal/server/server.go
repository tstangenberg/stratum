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
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tstangenberg/stratum/internal/api"
	"github.com/tstangenberg/stratum/internal/plugin"
	eqfilter "github.com/tstangenberg/stratum/internal/plugin/filter/eq"
	"github.com/tstangenberg/stratum/internal/plugin/scalar"
	booleanscalar "github.com/tstangenberg/stratum/internal/plugin/scalar/boolean"
	floatscalar "github.com/tstangenberg/stratum/internal/plugin/scalar/float"
	idscalar "github.com/tstangenberg/stratum/internal/plugin/scalar/id"
	intscalar "github.com/tstangenberg/stratum/internal/plugin/scalar/int"
	stringscalar "github.com/tstangenberg/stratum/internal/plugin/scalar/string"
	"github.com/tstangenberg/stratum/internal/schema"
	"github.com/tstangenberg/stratum/internal/system"
	"github.com/tstangenberg/stratum/internal/ui"
)

var errNotImplemented = errors.New("not implemented")

type schemaRepository interface {
	Upsert(context.Context, schema.PersistedSchema) (schema.PersistedSchema, error)
	All(context.Context) ([]schema.PersistedSchema, error)
}

type schemaHandlerBuilder func(
	*pgxpool.Pool,
	string,
	*schema.ParsedSchema,
	map[string]scalar.Plugin,
	[]plugin.QueryModifier,
	[]plugin.FilterPlugin,
	int,
) (http.Handler, error)

type printfLogger interface {
	Printf(string, ...any)
}

// StratumServer is the main server struct.
type StratumServer struct {
	healthPlugins      []plugin.HealthPlugin
	middlewares        []plugin.HTTPMiddleware
	db                 *pgxpool.Pool
	schemas            *schema.Store
	scalars            map[string]scalar.Plugin
	queryModifiers     []plugin.QueryModifier
	filterPlugins      []plugin.FilterPlugin
	uiHandlerBuilder   func(ui.StatusProvider, ui.SchemaProvider) (*ui.Handler, error)
	createTable        func(ctx context.Context, db *pgxpool.Pool, schemaName string, t schema.TypeDef, scalars map[string]scalar.Plugin) error
	addColumns         func(ctx context.Context, db *pgxpool.Pool, schemaName string, t schema.TypeDef, scalars map[string]scalar.Plugin) error
	migrateSystem      func(context.Context, *pgxpool.Pool) error
	schemaRepository   schemaRepository
	buildSchemaHandler schemaHandlerBuilder
	logger             printfLogger
}

// NewStratumServer creates a new StratumServer. Health plugins and query
// modifiers are wired via their respective self-registration registries.
func NewStratumServer() *StratumServer {
	scalars := map[string]scalar.Plugin{
		"String":  stringscalar.Plugin{},
		"ID":      idscalar.Plugin{},
		"Int":     intscalar.Plugin{},
		"Float":   floatscalar.Plugin{},
		"Boolean": booleanscalar.Plugin{},
	}
	return &StratumServer{
		healthPlugins:  plugin.BuildHealthPlugins(),
		schemas:        schema.NewStore(),
		queryModifiers: plugin.BuildQueryModifiers(),
		scalars:        scalars,
		filterPlugins: []plugin.FilterPlugin{
			eqfilter.New("String", scalars["String"].GraphQLType()),
			eqfilter.New("ID", scalars["ID"].GraphQLType()),
			eqfilter.New("Int", scalars["Int"].GraphQLType()),
			eqfilter.New("Float", scalars["Float"].GraphQLType()),
			eqfilter.New("Boolean", scalars["Boolean"].GraphQLType()),
		},
		uiHandlerBuilder:   ui.NewHandler,
		createTable:        schema.CreateTable,
		addColumns:         schema.AddColumns,
		migrateSystem:      system.Migrate,
		buildSchemaHandler: schema.BuildHandler,
		logger:             log.Default(),
	}
}

// WithDB sets the PostgreSQL connection pool and returns the server for chaining.
func (s *StratumServer) WithDB(db *pgxpool.Pool) *StratumServer {
	s.db = db
	s.schemaRepository = schema.NewRepository(db)
	return s
}

// Initialize runs system migrations and restores persisted schemas.
func (s *StratumServer) Initialize(ctx context.Context) error {
	if s.db == nil {
		return nil
	}
	if err := s.migrateSystem(ctx, s.db); err != nil {
		return fmt.Errorf("initialize server: migrate system tables: %w", err)
	}

	persistedSchemas, err := s.schemaRepository.All(ctx)
	if err != nil {
		return fmt.Errorf("initialize server: load persisted schemas: %w", err)
	}
	for _, persisted := range persistedSchemas {
		parsed, err := schema.ParseSDL(persisted.SDL)
		if err != nil {
			s.logger.Printf("load persisted schema %q: parse SDL: %v", persisted.Name, err)
			continue
		}
		handler, err := s.buildSchemaHandler(
			s.db,
			persisted.Name,
			parsed,
			s.scalars,
			s.queryModifiers,
			s.filterPlugins,
			schema.MaxDepthFromEnv(),
		)
		if err != nil {
			s.logger.Printf("load persisted schema %q: build handler: %v", persisted.Name, err)
			continue
		}
		s.schemas.Set(persisted.Name, &schema.Schema{
			Name:      persisted.Name,
			SDL:       persisted.SDL,
			Parsed:    parsed,
			Version:   persisted.Version,
			CreatedAt: persisted.CreatedAt,
			UpdatedAt: persisted.UpdatedAt,
			Handler:   handler,
		})
	}
	return nil
}

// WithFilterPlugins replaces the entire filter plugin set and returns the server for chaining.
// The default set contains eq-filters for all MVP scalars; callers must include them explicitly if still needed.
func (s *StratumServer) WithFilterPlugins(plugins ...plugin.FilterPlugin) *StratumServer {
	s.filterPlugins = plugins
	return s
}

// WithMiddlewares appends HTTP middleware to the pipeline and returns the server for chaining.
// Middlewares are applied in ascending Priority() order at request time.
func (s *StratumServer) WithMiddlewares(m ...plugin.HTTPMiddleware) *StratumServer {
	s.middlewares = append(s.middlewares, m...)
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
	all := s.schemas.All()
	summaries := make([]api.SchemaSummary, len(all))
	for i, sc := range all {
		summaries[i] = api.SchemaSummary{
			Name:      sc.Name,
			Version:   sc.Version,
			UpdatedAt: sc.UpdatedAt,
		}
	}
	return api.ListSchemas200JSONResponse{Schemas: summaries}, nil
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
		resp := api.ValidationErrorJSONResponse{
			Error:   "validation_failed",
			Message: err.Error(),
		}
		var ve *schema.ValidationError
		if errors.As(err, &ve) {
			var details []api.ErrorDetail
			for _, d := range ve.Details {
				det := api.ErrorDetail{Message: strPtr(d.Message)}
				if d.Line != 0 {
					det.Line = intPtr(d.Line)
				}
				if d.Column != 0 {
					det.Column = intPtr(d.Column)
				}
				details = append(details, det)
			}
			if len(details) > 0 {
				resp.Details = &details
			}
		}
		return api.UpsertSchema422JSONResponse{ValidationErrorJSONResponse: resp}, nil
	}

	if req.Params.Preview != nil && *req.Params.Preview {
		return api.UpsertSchema200JSONResponse{
			Name:      name,
			Status:    api.Preview,
			Version:   1,
			UpdatedAt: time.Now(),
		}, nil
	}

	// CreateTable is IF NOT EXISTS, AddColumns uses IF NOT EXISTS per column.
	// Calling both unconditionally handles first upload, re-upload, and
	// re-upload after a server restart (empty in-memory store) identically.
	for _, t := range ps.Types {
		if err := s.createTable(ctx, s.db, name, t, s.scalars); err != nil {
			return nil, fmt.Errorf("upsert schema %q: %w", name, err)
		}
		if err := s.addColumns(ctx, s.db, name, t, s.scalars); err != nil {
			return nil, fmt.Errorf("upsert schema %q: %w", name, err)
		}
	}

	h, err := s.buildSchemaHandler(s.db, name, ps, s.scalars, s.queryModifiers, s.filterPlugins, schema.MaxDepthFromEnv())
	if err != nil {
		return nil, fmt.Errorf("upsert schema %q: build handler: %w", name, err)
	}

	now := time.Now()
	persisted, err := s.schemaRepository.Upsert(ctx, schema.PersistedSchema{
		Name:      name,
		SDL:       req.Body.Sdl,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("upsert schema %q: persist: %w", name, err)
	}

	endpoint := "/graphql/" + name
	newSchema := &schema.Schema{
		Name:      persisted.Name,
		SDL:       persisted.SDL,
		Parsed:    ps,
		Version:   persisted.Version,
		CreatedAt: persisted.CreatedAt,
		UpdatedAt: persisted.UpdatedAt,
		Handler:   h,
	}
	s.schemas.SetIfNewer(name, newSchema)

	return api.UpsertSchema200JSONResponse{
		Name:            name,
		Status:          api.Applied,
		Version:         newSchema.Version,
		UpdatedAt:       newSchema.UpdatedAt,
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

// HealthStatus returns the current liveness and readiness status for the UI.
func (s *StratumServer) HealthStatus(ctx context.Context) ui.HealthResult {
	result := ui.HealthResult{
		Liveness:   "ok",
		Readiness:  "ok",
		Components: make(map[string]string),
	}

	if len(s.healthPlugins) == 0 {
		return result
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	type checkResult struct {
		name   string
		status string
	}

	ch := make(chan checkResult, len(s.healthPlugins))
	var wg sync.WaitGroup
	for _, p := range s.healthPlugins {
		wg.Add(1)
		p := p
		go func() {
			defer wg.Done()
			st := p.Check(ctx)
			status := "ok"
			if st.Status != plugin.StatusOK {
				status = "error"
			}
			ch <- checkResult{name: p.Name(), status: status}
		}()
	}
	wg.Wait()
	close(ch)

	for r := range ch {
		result.Components[r.name] = r.status
		if r.status != "ok" {
			result.Readiness = "degraded"
		}
	}

	return result
}

// Schemas returns summary information about all registered schemas.
func (s *StratumServer) Schemas() []ui.SchemaInfo {
	all := s.schemas.All()
	infos := make([]ui.SchemaInfo, len(all))
	for i, sc := range all {
		infos[i] = ui.SchemaInfo{
			Name:    sc.Name,
			SDL:     sc.SDL,
			Version: sc.Version,
		}
	}
	return infos
}

// Plugins returns information about all registered plugins.
func (s *StratumServer) Plugins() []ui.PluginInfo {
	var plugins []ui.PluginInfo

	for _, p := range s.healthPlugins {
		plugins = append(plugins, ui.PluginInfo{Name: p.Name(), Type: "health"})
	}
	for _, m := range s.middlewares {
		plugins = append(plugins, ui.PluginInfo{Name: m.Name(), Type: "middleware"})
	}
	for _, qm := range s.queryModifiers {
		plugins = append(plugins, ui.PluginInfo{Name: qm.Name(), Type: "query-modifier"})
	}
	for _, fp := range s.filterPlugins {
		plugins = append(plugins, ui.PluginInfo{Name: fp.Name(), Type: "filter"})
	}

	return plugins
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

// Handler returns an http.Handler for all Stratum routes.
// Health endpoints bypass all middleware. Remaining requests pass through
// registered middlewares in ascending priority order before reaching the mux.
func Handler(srv *StratumServer) (http.Handler, error) {
	strict := api.NewStrictHandlerWithOptions(srv, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		},
		ResponseErrorHandlerFunc: notImplementedHandler,
	})
	mux := http.NewServeMux()
	mux.Handle("/api/", api.Handler(strict))
	mux.HandleFunc("POST /graphql/{name}", srv.serveGraphQL)
	uiHandler, err := srv.uiHandlerBuilder(srv, srv)
	if err != nil {
		return nil, err
	}
	mux.Handle("/ui/", http.StripPrefix("/ui", uiHandler))
	mux.HandleFunc("GET /ui", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/status", http.StatusMovedPermanently)
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui", http.StatusMovedPermanently)
	})
	return buildChain(srv.middlewares, mux), nil
}

func buildChain(middlewares []plugin.HTTPMiddleware, mux http.Handler) http.Handler {
	h := http.Handler(mux)
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i].Wrap(h)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || isHealthEndpoint(r.URL.Path) || isUIEndpoint(r.URL.Path) {
			mux.ServeHTTP(w, r)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func isHealthEndpoint(path string) bool {
	return path == "/api/v1/health/live" || path == "/api/v1/health/ready"
}

func isUIEndpoint(path string) bool {
	return path == "/ui" || strings.HasPrefix(path, "/ui/")
}

func intPtr(v int) *int       { return &v }
func strPtr(v string) *string { return &v }
