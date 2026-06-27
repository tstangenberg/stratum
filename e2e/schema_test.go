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
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/tstangenberg/stratum/internal/api"
	"github.com/tstangenberg/stratum/internal/server"
)

func TestUploadSchemaString(t *testing.T) {
	ctx := context.Background()

	pgc, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("stratum"),
		postgres.WithUsername("stratum"),
		postgres.WithPassword("stratum"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = pgc.Terminate(ctx) })

	dsn, err := pgc.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect pool: %v", err)
	}
	t.Cleanup(pool.Close)

	handler := server.Handler(server.NewStratumServer().WithDB(pool))

	// ── 1. Upload schema ────────────────────────────────────────────────────
	sdl := `type Location { id: ID! name: String! }`
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/locations",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("upload: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var uploadResp api.SchemaUploadResponse
	if err := json.NewDecoder(w.Body).Decode(&uploadResp); err != nil {
		t.Fatalf("upload: decode response: %v", err)
	}
	if uploadResp.Name != "locations" {
		t.Errorf("upload: name = %q, want %q", uploadResp.Name, "locations")
	}
	if uploadResp.Status != api.Applied {
		t.Errorf("upload: status = %q, want %q", uploadResp.Status, api.Applied)
	}
	if uploadResp.Version != 1 {
		t.Errorf("upload: version = %d, want 1", uploadResp.Version)
	}
	wantEndpoint := "/graphql/locations"
	if uploadResp.GraphqlEndpoint == nil || *uploadResp.GraphqlEndpoint != wantEndpoint {
		t.Errorf("upload: graphql_endpoint = %v, want %q", uploadResp.GraphqlEndpoint, wantEndpoint)
	}

	// ── 2. Create a record via GraphQL ──────────────────────────────────────
	gqlCreate := `{"query":"mutation { location { create(input: {name: \"Zürich\"}) { id name } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/locations",
		strings.NewReader(gqlCreate))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var createResult struct {
		Data struct {
			Location struct {
				Create struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"create"`
			} `json:"location"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&createResult); err != nil {
		t.Fatalf("create: decode: %v", err)
	}
	if len(createResult.Errors) > 0 {
		t.Fatalf("create: GraphQL errors: %v", createResult.Errors)
	}
	createdID := createResult.Data.Location.Create.ID
	if createdID == "" {
		t.Fatal("create: expected non-empty id")
	}
	if createResult.Data.Location.Create.Name != "Zürich" {
		t.Errorf("create: name = %q, want %q", createResult.Data.Location.Create.Name, "Zürich")
	}

	// ── 3. Read back via GraphQL get ────────────────────────────────────────
	gqlGet := fmt.Sprintf(`{"query":"{ location { get(id: \"%s\") { id name } } }"}`, createdID)
	req = httptest.NewRequest(http.MethodPost, "/graphql/locations",
		strings.NewReader(gqlGet))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var getResult struct {
		Data struct {
			Location struct {
				Get struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"get"`
			} `json:"location"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&getResult); err != nil {
		t.Fatalf("get: decode: %v", err)
	}
	if len(getResult.Errors) > 0 {
		t.Fatalf("get: GraphQL errors: %v", getResult.Errors)
	}
	if getResult.Data.Location.Get.ID != createdID {
		t.Errorf("get: id = %q, want %q", getResult.Data.Location.Get.ID, createdID)
	}
	if getResult.Data.Location.Get.Name != "Zürich" {
		t.Errorf("get: name = %q, want %q", getResult.Data.Location.Get.Name, "Zürich")
	}
}

func TestSchemaIDScalar(t *testing.T) {
	ctx := context.Background()

	pgc, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("stratum"),
		postgres.WithUsername("stratum"),
		postgres.WithPassword("stratum"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = pgc.Terminate(ctx) })

	dsn, err := pgc.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect pool: %v", err)
	}
	t.Cleanup(pool.Close)

	handler := server.Handler(server.NewStratumServer().WithDB(pool))

	// ── 1. Upload schema with ID! field ─────────────────────────────────────
	sdl := `type Thing { id: ID! name: String! }`
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/things",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("upload: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var uploadResp api.SchemaUploadResponse
	if err := json.NewDecoder(w.Body).Decode(&uploadResp); err != nil {
		t.Fatalf("upload: decode response: %v", err)
	}
	if uploadResp.Status != api.Applied {
		t.Errorf("upload: status = %q, want %q", uploadResp.Status, api.Applied)
	}

	// ── 2. Verify PostgreSQL column type is TEXT and primary key ─────────────
	var colType string
	err = pool.QueryRow(ctx,
		`SELECT data_type FROM information_schema.columns
		 WHERE table_name = 'things_thing' AND column_name = 'id'`).Scan(&colType)
	if err != nil {
		t.Fatalf("column type query: %v", err)
	}
	if colType != "text" {
		t.Errorf("column type = %q, want %q", colType, "text")
	}

	var constraintType string
	err = pool.QueryRow(ctx,
		`SELECT constraint_type FROM information_schema.table_constraints
		 WHERE table_name = 'things_thing' AND constraint_type = 'PRIMARY KEY'`).Scan(&constraintType)
	if err != nil {
		t.Fatalf("primary key query: %v", err)
	}
	if constraintType != "PRIMARY KEY" {
		t.Errorf("constraint_type = %q, want %q", constraintType, "PRIMARY KEY")
	}

	// ── 3. Client-supplied ID is stored and returned as-is ──────────────────
	gqlCreate := `{"query":"mutation { thing { create(input: {id: \"my-custom-id\", name: \"Widget\"}) { id name } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/things",
		strings.NewReader(gqlCreate))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("create-custom: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var createResult struct {
		Data struct {
			Thing struct {
				Create struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"create"`
			} `json:"thing"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&createResult); err != nil {
		t.Fatalf("create-custom: decode: %v", err)
	}
	if len(createResult.Errors) > 0 {
		t.Fatalf("create-custom: GraphQL errors: %v", createResult.Errors)
	}
	if createResult.Data.Thing.Create.ID != "my-custom-id" {
		t.Errorf("create-custom: id = %q, want %q", createResult.Data.Thing.Create.ID, "my-custom-id")
	}

	// Read it back
	gqlGet := `{"query":"{ thing { get(id: \"my-custom-id\") { id name } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/things",
		strings.NewReader(gqlGet))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get-custom: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var getResult struct {
		Data struct {
			Thing struct {
				Get struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"get"`
			} `json:"thing"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&getResult); err != nil {
		t.Fatalf("get-custom: decode: %v", err)
	}
	if len(getResult.Errors) > 0 {
		t.Fatalf("get-custom: GraphQL errors: %v", getResult.Errors)
	}
	if getResult.Data.Thing.Get.ID != "my-custom-id" {
		t.Errorf("get-custom: id = %q, want %q", getResult.Data.Thing.Get.ID, "my-custom-id")
	}
	if getResult.Data.Thing.Get.Name != "Widget" {
		t.Errorf("get-custom: name = %q, want %q", getResult.Data.Thing.Get.Name, "Widget")
	}

	// ── 4. Duplicate ID returns a GraphQL error ─────────────────────────────
	gqlDup := `{"query":"mutation { thing { create(input: {id: \"my-custom-id\", name: \"Duplicate\"}) { id } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/things",
		strings.NewReader(gqlDup))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("dup: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var dupResult struct {
		Data   any                        `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&dupResult); err != nil {
		t.Fatalf("dup: decode: %v", err)
	}
	if len(dupResult.Errors) == 0 {
		t.Fatal("dup: expected GraphQL errors for duplicate ID, got none")
	}

	// ── 5. Omitting id generates a unique ID ────────────────────────────────
	gqlAuto := `{"query":"mutation { thing { create(input: {name: \"AutoID\"}) { id name } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/things",
		strings.NewReader(gqlAuto))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("auto-id: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var autoResult struct {
		Data struct {
			Thing struct {
				Create struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"create"`
			} `json:"thing"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&autoResult); err != nil {
		t.Fatalf("auto-id: decode: %v", err)
	}
	if len(autoResult.Errors) > 0 {
		t.Fatalf("auto-id: GraphQL errors: %v", autoResult.Errors)
	}
	if autoResult.Data.Thing.Create.ID == "" {
		t.Fatal("auto-id: expected non-empty generated id")
	}
	if autoResult.Data.Thing.Create.ID == "my-custom-id" {
		t.Error("auto-id: generated id should differ from the custom one")
	}
	uuidRe := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !uuidRe.MatchString(autoResult.Data.Thing.Create.ID) {
		t.Errorf("auto-id: expected UUID v4 format, got %q", autoResult.Data.Thing.Create.ID)
	}
}

func TestSchemaFloatScalar(t *testing.T) {
	ctx := context.Background()

	pgc, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("stratum"),
		postgres.WithUsername("stratum"),
		postgres.WithPassword("stratum"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = pgc.Terminate(ctx) })

	dsn, err := pgc.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect pool: %v", err)
	}
	t.Cleanup(pool.Close)

	handler := server.Handler(server.NewStratumServer().WithDB(pool))

	// ── 1. Upload schema with Float! fields ─────────────────────────────────
	sdl := `type Coordinate { id: ID! lat: Float! lon: Float! }`
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/coords",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("upload: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var uploadResp api.SchemaUploadResponse
	if err := json.NewDecoder(w.Body).Decode(&uploadResp); err != nil {
		t.Fatalf("upload: decode response: %v", err)
	}
	if uploadResp.Status != api.Applied {
		t.Errorf("upload: status = %q, want %q", uploadResp.Status, api.Applied)
	}

	// ── 2. Verify PostgreSQL column type is DOUBLE PRECISION ────────────────
	var colType string
	err = pool.QueryRow(ctx,
		`SELECT data_type FROM information_schema.columns
		 WHERE table_name = 'coords_coordinate' AND column_name = 'lat'`).Scan(&colType)
	if err != nil {
		t.Fatalf("column type query: %v", err)
	}
	if colType != "double precision" {
		t.Errorf("column type = %q, want %q", colType, "double precision")
	}

	// ── 3. Create a record with decimal float values ────────────────────────
	// Exact float equality is safe here: 47.3769, 8.5417, 1.0, 2.0 are all
	// exactly representable in float64 and round-trip through DOUBLE PRECISION
	// → JSON → float64 without loss.
	gqlCreate := `{"query":"mutation { coordinate { create(input: {lat: 47.3769, lon: 8.5417}) { id lat lon } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/coords",
		strings.NewReader(gqlCreate))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var createResult struct {
		Data struct {
			Coordinate struct {
				Create struct {
					ID  string  `json:"id"`
					Lat float64 `json:"lat"`
					Lon float64 `json:"lon"`
				} `json:"create"`
			} `json:"coordinate"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&createResult); err != nil {
		t.Fatalf("create: decode: %v", err)
	}
	if len(createResult.Errors) > 0 {
		t.Fatalf("create: GraphQL errors: %v", createResult.Errors)
	}
	createdID := createResult.Data.Coordinate.Create.ID
	if createdID == "" {
		t.Fatal("create: expected non-empty id")
	}
	if createResult.Data.Coordinate.Create.Lat != 47.3769 {
		t.Errorf("create: lat = %v, want 47.3769", createResult.Data.Coordinate.Create.Lat)
	}
	if createResult.Data.Coordinate.Create.Lon != 8.5417 {
		t.Errorf("create: lon = %v, want 8.5417", createResult.Data.Coordinate.Create.Lon)
	}

	// ── 4. Read back via GraphQL get — decimal precision intact ──────────────
	gqlGet := fmt.Sprintf(`{"query":"{ coordinate { get(id: \"%s\") { id lat lon } } }"}`, createdID)
	req = httptest.NewRequest(http.MethodPost, "/graphql/coords",
		strings.NewReader(gqlGet))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var getResult struct {
		Data struct {
			Coordinate struct {
				Get struct {
					ID  string  `json:"id"`
					Lat float64 `json:"lat"`
					Lon float64 `json:"lon"`
				} `json:"get"`
			} `json:"coordinate"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&getResult); err != nil {
		t.Fatalf("get: decode: %v", err)
	}
	if len(getResult.Errors) > 0 {
		t.Fatalf("get: GraphQL errors: %v", getResult.Errors)
	}
	if getResult.Data.Coordinate.Get.Lat != 47.3769 {
		t.Errorf("get: lat = %v, want 47.3769", getResult.Data.Coordinate.Get.Lat)
	}
	if getResult.Data.Coordinate.Get.Lon != 8.5417 {
		t.Errorf("get: lon = %v, want 8.5417", getResult.Data.Coordinate.Get.Lon)
	}

	// ── 5. Integer literal accepted as Float input ──────────────────────────
	gqlIntLit := `{"query":"mutation { coordinate { create(input: {lat: 1, lon: 2}) { id lat lon } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/coords",
		strings.NewReader(gqlIntLit))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("int-literal: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var intLitResult struct {
		Data struct {
			Coordinate struct {
				Create struct {
					Lat float64 `json:"lat"`
					Lon float64 `json:"lon"`
				} `json:"create"`
			} `json:"coordinate"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&intLitResult); err != nil {
		t.Fatalf("int-literal: decode: %v", err)
	}
	if len(intLitResult.Errors) > 0 {
		t.Fatalf("int-literal: GraphQL errors: %v", intLitResult.Errors)
	}
	if intLitResult.Data.Coordinate.Create.Lat != 1.0 {
		t.Errorf("int-literal: lat = %v, want 1", intLitResult.Data.Coordinate.Create.Lat)
	}
	if intLitResult.Data.Coordinate.Create.Lon != 2.0 {
		t.Errorf("int-literal: lon = %v, want 2", intLitResult.Data.Coordinate.Create.Lon)
	}
}

func TestSchemaBooleanScalar(t *testing.T) {
	ctx := context.Background()

	pgc, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("stratum"),
		postgres.WithUsername("stratum"),
		postgres.WithPassword("stratum"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = pgc.Terminate(ctx) })

	dsn, err := pgc.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect pool: %v", err)
	}
	t.Cleanup(pool.Close)

	handler := server.Handler(server.NewStratumServer().WithDB(pool))

	// ── 1. Upload schema with Boolean! field ────────────────────────────────
	sdl := `type Record { id: ID! name: String! inAenderung: Boolean! }`
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/records",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("upload: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var uploadResp api.SchemaUploadResponse
	if err := json.NewDecoder(w.Body).Decode(&uploadResp); err != nil {
		t.Fatalf("upload: decode response: %v", err)
	}
	if uploadResp.Status != api.Applied {
		t.Errorf("upload: status = %q, want %q", uploadResp.Status, api.Applied)
	}

	// ── 2. Verify PostgreSQL column type is BOOLEAN ─────────────────────────
	var colType string
	err = pool.QueryRow(ctx,
		`SELECT data_type FROM information_schema.columns
		 WHERE table_name = 'records_record' AND column_name = 'inaenderung'`).Scan(&colType)
	if err != nil {
		t.Fatalf("column type query: %v", err)
	}
	if colType != "boolean" {
		t.Errorf("column type = %q, want %q", colType, "boolean")
	}

	// ── 3. Create with true, read back ──────────────────────────────────────
	gqlCreate := `{"query":"mutation { record { create(input: {name: \"Test\", inAenderung: true}) { id name inAenderung } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/records",
		strings.NewReader(gqlCreate))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("create-true: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var createResult struct {
		Data struct {
			Record struct {
				Create struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					InAenderung bool   `json:"inAenderung"`
				} `json:"create"`
			} `json:"record"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&createResult); err != nil {
		t.Fatalf("create-true: decode: %v", err)
	}
	if len(createResult.Errors) > 0 {
		t.Fatalf("create-true: GraphQL errors: %v", createResult.Errors)
	}
	trueID := createResult.Data.Record.Create.ID
	if trueID == "" {
		t.Fatal("create-true: expected non-empty id")
	}
	if !createResult.Data.Record.Create.InAenderung {
		t.Errorf("create-true: inAenderung = false, want true")
	}

	// Read back true record
	gqlGet := fmt.Sprintf(`{"query":"{ record { get(id: \"%s\") { id inAenderung } } }"}`, trueID)
	req = httptest.NewRequest(http.MethodPost, "/graphql/records",
		strings.NewReader(gqlGet))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get-true: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var getResult struct {
		Data struct {
			Record struct {
				Get struct {
					ID          string `json:"id"`
					InAenderung bool   `json:"inAenderung"`
				} `json:"get"`
			} `json:"record"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&getResult); err != nil {
		t.Fatalf("get-true: decode: %v", err)
	}
	if len(getResult.Errors) > 0 {
		t.Fatalf("get-true: GraphQL errors: %v", getResult.Errors)
	}
	if !getResult.Data.Record.Get.InAenderung {
		t.Errorf("get-true: inAenderung = false, want true")
	}

	// ── 4. Create with false, read back ─────────────────────────────────────
	gqlCreateFalse := `{"query":"mutation { record { create(input: {name: \"Test2\", inAenderung: false}) { id inAenderung } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/records",
		strings.NewReader(gqlCreateFalse))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("create-false: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var createFalseResult struct {
		Data struct {
			Record struct {
				Create struct {
					ID          string `json:"id"`
					InAenderung bool   `json:"inAenderung"`
				} `json:"create"`
			} `json:"record"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&createFalseResult); err != nil {
		t.Fatalf("create-false: decode: %v", err)
	}
	if len(createFalseResult.Errors) > 0 {
		t.Fatalf("create-false: GraphQL errors: %v", createFalseResult.Errors)
	}
	if createFalseResult.Data.Record.Create.InAenderung {
		t.Errorf("create-false: inAenderung = true, want false")
	}
	falseID := createFalseResult.Data.Record.Create.ID
	if falseID == "" {
		t.Fatal("create-false: expected non-empty id")
	}

	// Read back false record to confirm persistence
	gqlGetFalse := fmt.Sprintf(`{"query":"{ record { get(id: \"%s\") { id inAenderung } } }"}`, falseID)
	req = httptest.NewRequest(http.MethodPost, "/graphql/records",
		strings.NewReader(gqlGetFalse))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get-false: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var getFalseResult struct {
		Data struct {
			Record struct {
				Get struct {
					ID          string `json:"id"`
					InAenderung bool   `json:"inAenderung"`
				} `json:"get"`
			} `json:"record"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&getFalseResult); err != nil {
		t.Fatalf("get-false: decode: %v", err)
	}
	if len(getFalseResult.Errors) > 0 {
		t.Fatalf("get-false: GraphQL errors: %v", getFalseResult.Errors)
	}
	if getFalseResult.Data.Record.Get.InAenderung {
		t.Errorf("get-false: inAenderung = true, want false")
	}

	// ── 5. String "true" rejected as invalid Boolean input ──────────────────
	gqlStringBool := `{"query":"mutation { record { create(input: {name: \"Bad\", inAenderung: \"true\"}) { id } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/records",
		strings.NewReader(gqlStringBool))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("string-bool: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var stringBoolResult struct {
		Data   any                        `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&stringBoolResult); err != nil {
		t.Fatalf("string-bool: decode: %v", err)
	}
	if len(stringBoolResult.Errors) == 0 {
		t.Fatal("string-bool: expected GraphQL errors for string input, got none")
	}
}

func TestSchemaIntScalar(t *testing.T) {
	ctx := context.Background()

	pgc, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("stratum"),
		postgres.WithUsername("stratum"),
		postgres.WithPassword("stratum"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = pgc.Terminate(ctx) })

	dsn, err := pgc.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect pool: %v", err)
	}
	t.Cleanup(pool.Close)

	handler := server.Handler(server.NewStratumServer().WithDB(pool))

	// ── 1. Upload schema with Int! field ────────────────────────────────────
	sdl := `type Product { id: ID! name: String! quantity: Int! }`
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/products",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("upload: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var uploadResp api.SchemaUploadResponse
	if err := json.NewDecoder(w.Body).Decode(&uploadResp); err != nil {
		t.Fatalf("upload: decode response: %v", err)
	}
	if uploadResp.Status != api.Applied {
		t.Errorf("upload: status = %q, want %q", uploadResp.Status, api.Applied)
	}

	// ── 2. Verify PostgreSQL column type is INTEGER ─────────────────────────
	var colType string
	err = pool.QueryRow(ctx,
		`SELECT data_type FROM information_schema.columns
		 WHERE table_name = 'products_product' AND column_name = 'quantity'`).Scan(&colType)
	if err != nil {
		t.Fatalf("column type query: %v", err)
	}
	if colType != "integer" {
		t.Errorf("column type = %q, want %q", colType, "integer")
	}

	// ── 3. Create a record with an integer value ────────────────────────────
	gqlCreate := `{"query":"mutation { product { create(input: {name: \"Widget\", quantity: 42}) { id name quantity } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/products",
		strings.NewReader(gqlCreate))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var createResult struct {
		Data struct {
			Product struct {
				Create struct {
					ID       string `json:"id"`
					Name     string `json:"name"`
					Quantity int    `json:"quantity"`
				} `json:"create"`
			} `json:"product"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&createResult); err != nil {
		t.Fatalf("create: decode: %v", err)
	}
	if len(createResult.Errors) > 0 {
		t.Fatalf("create: GraphQL errors: %v", createResult.Errors)
	}
	createdID := createResult.Data.Product.Create.ID
	if createdID == "" {
		t.Fatal("create: expected non-empty id")
	}
	if createResult.Data.Product.Create.Quantity != 42 {
		t.Errorf("create: quantity = %d, want 42", createResult.Data.Product.Create.Quantity)
	}

	// ── 4. Read back via GraphQL get ────────────────────────────────────────
	gqlGet := fmt.Sprintf(`{"query":"{ product { get(id: \"%s\") { id name quantity } } }"}`, createdID)
	req = httptest.NewRequest(http.MethodPost, "/graphql/products",
		strings.NewReader(gqlGet))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var getResult struct {
		Data struct {
			Product struct {
				Get struct {
					ID       string `json:"id"`
					Name     string `json:"name"`
					Quantity int    `json:"quantity"`
				} `json:"get"`
			} `json:"product"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&getResult); err != nil {
		t.Fatalf("get: decode: %v", err)
	}
	if len(getResult.Errors) > 0 {
		t.Fatalf("get: GraphQL errors: %v", getResult.Errors)
	}
	if getResult.Data.Product.Get.Quantity != 42 {
		t.Errorf("get: quantity = %d, want 42", getResult.Data.Product.Get.Quantity)
	}

	// ── 5. Out-of-range value returns a GraphQL error ───────────────────────
	gqlOverflow := `{"query":"mutation { product { create(input: {name: \"Overflow\", quantity: 2147483648}) { id } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/products",
		strings.NewReader(gqlOverflow))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var overflowResult struct {
		Data   any                        `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&overflowResult); err != nil {
		t.Fatalf("overflow: decode: %v", err)
	}
	if len(overflowResult.Errors) == 0 {
		t.Error("overflow: expected GraphQL error for out-of-range Int, got none")
	}
}

func TestSchemaNullableFields(t *testing.T) {
	ctx := context.Background()

	pgc, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("stratum"),
		postgres.WithUsername("stratum"),
		postgres.WithPassword("stratum"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = pgc.Terminate(ctx) })

	dsn, err := pgc.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect pool: %v", err)
	}
	t.Cleanup(pool.Close)

	handler := server.Handler(server.NewStratumServer().WithDB(pool))

	// Schema: name is required (String!), description is nullable (String)
	sdl := `type Article { id: ID! name: String! description: String }`
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/articles",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("upload: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	// ── 1. Verify NOT NULL on required field ────────────────────────────────
	var nameNullable string
	err = pool.QueryRow(ctx,
		`SELECT is_nullable FROM information_schema.columns
		 WHERE table_name = 'articles_article' AND column_name = 'name'`).Scan(&nameNullable)
	if err != nil {
		t.Fatalf("name is_nullable query: %v", err)
	}
	if nameNullable != "NO" {
		t.Errorf("name is_nullable = %q, want %q (NOT NULL column)", nameNullable, "NO")
	}

	// ── 2. Verify nullable column for optional field ────────────────────────
	var descNullable string
	err = pool.QueryRow(ctx,
		`SELECT is_nullable FROM information_schema.columns
		 WHERE table_name = 'articles_article' AND column_name = 'description'`).Scan(&descNullable)
	if err != nil {
		t.Fatalf("description is_nullable query: %v", err)
	}
	if descNullable != "YES" {
		t.Errorf("description is_nullable = %q, want %q (nullable column)", descNullable, "YES")
	}

	// ── 3. Create a record without the nullable field — succeeds ────────────
	gqlCreate := `{"query":"mutation { article { create(input: {name: \"Go Basics\"}) { id name description } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/articles",
		strings.NewReader(gqlCreate))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("create-nullable: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var createResult struct {
		Data struct {
			Article struct {
				Create struct {
					ID          string  `json:"id"`
					Name        string  `json:"name"`
					Description *string `json:"description"`
				} `json:"create"`
			} `json:"article"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&createResult); err != nil {
		t.Fatalf("create-nullable: decode: %v", err)
	}
	if len(createResult.Errors) > 0 {
		t.Fatalf("create-nullable: GraphQL errors: %v", createResult.Errors)
	}
	createdID := createResult.Data.Article.Create.ID
	if createdID == "" {
		t.Fatal("create-nullable: expected non-empty id")
	}
	if createResult.Data.Article.Create.Name != "Go Basics" {
		t.Errorf("create-nullable: name = %q, want %q", createResult.Data.Article.Create.Name, "Go Basics")
	}
	if createResult.Data.Article.Create.Description != nil {
		t.Errorf("create-nullable: description = %v, want nil", *createResult.Data.Article.Create.Description)
	}

	// ── 4. Verify NULL stored in DB ─────────────────────────────────────────
	var dbDesc *string
	err = pool.QueryRow(ctx,
		`SELECT description FROM articles_article WHERE id = $1`, createdID).Scan(&dbDesc)
	if err != nil {
		t.Fatalf("query description: %v", err)
	}
	if dbDesc != nil {
		t.Errorf("db description = %q, want NULL", *dbDesc)
	}

	// ── 5. Querying a NULL field returns null in GraphQL response ────────────
	gqlGet := fmt.Sprintf(`{"query":"{ article { get(id: \"%s\") { id name description } } }"}`, createdID)
	req = httptest.NewRequest(http.MethodPost, "/graphql/articles",
		strings.NewReader(gqlGet))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get-nullable: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var getResult struct {
		Data struct {
			Article struct {
				Get struct {
					ID          string  `json:"id"`
					Name        string  `json:"name"`
					Description *string `json:"description"`
				} `json:"get"`
			} `json:"article"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&getResult); err != nil {
		t.Fatalf("get-nullable: decode: %v", err)
	}
	if len(getResult.Errors) > 0 {
		t.Fatalf("get-nullable: GraphQL errors: %v", getResult.Errors)
	}
	if getResult.Data.Article.Get.Description != nil {
		t.Errorf("get-nullable: description = %v, want null", *getResult.Data.Article.Get.Description)
	}

	// ── 6. Creating without a required field returns a GraphQL error ─────────
	gqlMissing := `{"query":"mutation { article { create(input: {description: \"no name\"}) { id } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/articles",
		strings.NewReader(gqlMissing))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("missing-required: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var missingResult struct {
		Data   any                        `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&missingResult); err != nil {
		t.Fatalf("missing-required: decode: %v", err)
	}
	if len(missingResult.Errors) == 0 {
		t.Fatal("missing-required: expected GraphQL error for missing required field, got none")
	}

	// ── 7. Creating with both fields set works and returns non-null ──────────
	gqlFull := `{"query":"mutation { article { create(input: {name: \"Full Article\", description: \"Has content\"}) { id name description } } }"}`
	req = httptest.NewRequest(http.MethodPost, "/graphql/articles",
		strings.NewReader(gqlFull))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("create-full: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var fullResult struct {
		Data struct {
			Article struct {
				Create struct {
					ID          string  `json:"id"`
					Name        string  `json:"name"`
					Description *string `json:"description"`
				} `json:"create"`
			} `json:"article"`
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(w.Body).Decode(&fullResult); err != nil {
		t.Fatalf("create-full: decode: %v", err)
	}
	if len(fullResult.Errors) > 0 {
		t.Fatalf("create-full: GraphQL errors: %v", fullResult.Errors)
	}
	if fullResult.Data.Article.Create.Name != "Full Article" {
		t.Errorf("create-full: name = %q, want %q", fullResult.Data.Article.Create.Name, "Full Article")
	}
	if fullResult.Data.Article.Create.Description == nil {
		t.Fatal("create-full: description is nil, want non-nil")
	}
	if *fullResult.Data.Article.Create.Description != "Has content" {
		t.Errorf("create-full: description = %q, want %q", *fullResult.Data.Article.Create.Description, "Has content")
	}
}

func TestSchemaValidationSyntaxError(t *testing.T) {
	ctx := context.Background()

	pgc, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("stratum"),
		postgres.WithUsername("stratum"),
		postgres.WithPassword("stratum"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = pgc.Terminate(ctx) })

	dsn, err := pgc.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect pool: %v", err)
	}
	t.Cleanup(pool.Close)

	handler := server.Handler(server.NewStratumServer().WithDB(pool))

	// ── 1. Upload invalid SDL with syntax error → 422 ──────────────────────
	sdl := `type { broken`
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/broken",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("syntax error: expected 422, got %d — body: %s", w.Code, w.Body.String())
	}

	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Details []struct {
			Line    *int    `json:"line"`
			Column  *int    `json:"column"`
			Message *string `json:"message"`
		} `json:"details"`
	}
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("syntax error: decode: %v", err)
	}
	if errResp.Error != "validation_failed" {
		t.Errorf("syntax error: error = %q, want %q", errResp.Error, "validation_failed")
	}
	if len(errResp.Details) == 0 {
		t.Fatal("syntax error: expected details with line and column, got none")
	}
	if errResp.Details[0].Line == nil {
		t.Error("syntax error: details[0].line is nil, want non-nil")
	}
	if errResp.Details[0].Column == nil {
		t.Error("syntax error: details[0].column is nil, want non-nil")
	}

	// ── 2. Empty SDL → 422 ─────────────────────────────────────────────────
	emptyBody, _ := json.Marshal(api.SchemaUploadRequest{Sdl: ""})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/schemas/broken",
		bytes.NewReader(emptyBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("empty SDL: expected 422, got %d — body: %s", w.Code, w.Body.String())
	}

	// ── 3. Database unchanged after rejected schema ────────────────────────
	var tableCount int
	err = pool.QueryRow(ctx,
		`SELECT count(*) FROM information_schema.tables
		 WHERE table_name LIKE 'broken_%'`).Scan(&tableCount)
	if err != nil {
		t.Fatalf("table count query: %v", err)
	}
	if tableCount != 0 {
		t.Errorf("expected 0 tables after rejected schema, got %d", tableCount)
	}

	// ── 4. Valid upload after failed upload succeeds ────────────────────────
	validSDL := `type Item { id: ID! name: String! }`
	validBody, _ := json.Marshal(api.SchemaUploadRequest{Sdl: validSDL})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/schemas/broken",
		bytes.NewReader(validBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("valid after failed: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestSchemaValidationUnknownDirective(t *testing.T) {
	ctx := context.Background()

	pgc, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("stratum"),
		postgres.WithUsername("stratum"),
		postgres.WithPassword("stratum"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = pgc.Terminate(ctx) })

	dsn, err := pgc.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect pool: %v", err)
	}
	t.Cleanup(pool.Close)

	handler := server.Handler(server.NewStratumServer().WithDB(pool))

	// ── 1. Upload SDL with unknown directive → 422 ─────────────────────────
	sdl := `type Location @unknown { id: ID! name: String! }`
	body, _ := json.Marshal(api.SchemaUploadRequest{Sdl: sdl})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schemas/directives",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("unknown directive: expected 422, got %d — body: %s", w.Code, w.Body.String())
	}

	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Details []struct {
			Line    *int    `json:"line"`
			Column  *int    `json:"column"`
			Message *string `json:"message"`
		} `json:"details"`
	}
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("unknown directive: decode: %v", err)
	}
	if errResp.Error != "validation_failed" {
		t.Errorf("unknown directive: error = %q, want %q", errResp.Error, "validation_failed")
	}
	if !strings.Contains(errResp.Message, "unknown") {
		t.Errorf("unknown directive: message = %q, expected it to mention the directive name", errResp.Message)
	}
}
