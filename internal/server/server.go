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
	"net/http"
	"sync"
	"time"

	"github.com/tstangenberg/stratum/internal/api"
	"github.com/tstangenberg/stratum/internal/plugin"
)

var errNotImplemented = errors.New("not implemented")

// UnimplementedStrictServerInterface returns 501 for every operation.
// Embed it in StratumServer and override methods as they are implemented.
type UnimplementedStrictServerInterface struct{}

// StratumServer is the main server struct. Embed UnimplementedStrictServerInterface
// to get 501 responses for unimplemented endpoints, then override methods one by one.
type StratumServer struct {
	UnimplementedStrictServerInterface
	healthPlugins []plugin.HealthPlugin
}

// NewStratumServer creates a new StratumServer with the given health plugins.
func NewStratumServer(plugins ...plugin.HealthPlugin) *StratumServer {
	return &StratumServer{healthPlugins: plugins}
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

func (s *StratumServer) UpsertSchema(_ context.Context, _ api.UpsertSchemaRequestObject) (api.UpsertSchemaResponseObject, error) {
	return nil, errNotImplemented
}

func (s *StratumServer) GetSchemaStatus(_ context.Context, _ api.GetSchemaStatusRequestObject) (api.GetSchemaStatusResponseObject, error) {
	return nil, errNotImplemented
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

// Handler returns a net/http handler for the Stratum API.
func Handler(srv *StratumServer) http.Handler {
	strict := api.NewStrictHandlerWithOptions(srv, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		},
		ResponseErrorHandlerFunc: notImplementedHandler,
	})
	return api.Handler(strict)
}
