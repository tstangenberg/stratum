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
		RequestErrorHandlerFunc:  func(w http.ResponseWriter, r *http.Request, err error) { http.Error(w, err.Error(), http.StatusBadRequest) },
		ResponseErrorHandlerFunc: notImplementedHandler,
	})
	return api.Handler(strict)
}
