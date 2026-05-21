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

// UnimplementedStrictServerInterface returns 501 for every operation.
// Embed it in StratumServer and override methods as they are implemented.
type UnimplementedStrictServerInterface struct{}

func (UnimplementedStrictServerInterface) Liveness(_ context.Context, _ api.LivenessRequestObject) (api.LivenessResponseObject, error) {
	return nil, errNotImplemented
}

func (UnimplementedStrictServerInterface) Readiness(_ context.Context, _ api.ReadinessRequestObject) (api.ReadinessResponseObject, error) {
	return nil, errNotImplemented
}

func (UnimplementedStrictServerInterface) Info(_ context.Context, _ api.InfoRequestObject) (api.InfoResponseObject, error) {
	return nil, errNotImplemented
}

func (UnimplementedStrictServerInterface) ListSchemas(_ context.Context, _ api.ListSchemasRequestObject) (api.ListSchemasResponseObject, error) {
	return nil, errNotImplemented
}

func (UnimplementedStrictServerInterface) DeleteSchema(_ context.Context, _ api.DeleteSchemaRequestObject) (api.DeleteSchemaResponseObject, error) {
	return nil, errNotImplemented
}

func (UnimplementedStrictServerInterface) GetSchema(_ context.Context, _ api.GetSchemaRequestObject) (api.GetSchemaResponseObject, error) {
	return nil, errNotImplemented
}

func (UnimplementedStrictServerInterface) UpsertSchema(_ context.Context, _ api.UpsertSchemaRequestObject) (api.UpsertSchemaResponseObject, error) {
	return nil, errNotImplemented
}

func (UnimplementedStrictServerInterface) GetSchemaStatus(_ context.Context, _ api.GetSchemaStatusRequestObject) (api.GetSchemaStatusResponseObject, error) {
	return nil, errNotImplemented
}

// StratumServer is the main server struct. Embed UnimplementedStrictServerInterface
// to get 501 responses for unimplemented endpoints, then override methods one by one.
type StratumServer struct {
	UnimplementedStrictServerInterface
}

func (s *StratumServer) Liveness(_ context.Context, _ api.LivenessRequestObject) (api.LivenessResponseObject, error) {
	return api.Liveness200JSONResponse{Status: api.LivenessResponseStatusOk}, nil
}

// NewStratumServer creates a new StratumServer.
func NewStratumServer() *StratumServer {
	return &StratumServer{}
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
		RequestErrorHandlerFunc:  func(w http.ResponseWriter, r *http.Request, err error) { http.Error(w, err.Error(), http.StatusBadRequest) },
		ResponseErrorHandlerFunc: notImplementedHandler,
	})
	return api.Handler(strict)
}
