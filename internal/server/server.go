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

	"github.com/tstangenberg/stratum/internal/api"
)

var errNotImplemented = errors.New("not implemented")

type StratumServer struct{}

func NewStratumServer() *StratumServer {
	return &StratumServer{}
}

func (s *StratumServer) Liveness(_ context.Context, _ api.LivenessRequestObject) (api.LivenessResponseObject, error) {
	return api.Liveness200JSONResponse{Status: api.LivenessResponseStatusOk}, nil
}

func (s *StratumServer) Readiness(_ context.Context, _ api.ReadinessRequestObject) (api.ReadinessResponseObject, error) {
	return nil, errNotImplemented
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

func Handler(srv *StratumServer) http.Handler {
	strict := api.NewStrictHandlerWithOptions(srv, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		},
		ResponseErrorHandlerFunc: notImplementedHandler,
	})
	return api.Handler(strict)
}
