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

// SPDX-License-Identifier: AGPL-3.0-or-later
package schema_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tstangenberg/stratum/internal/plugin/scalar"
	stringscalar "github.com/tstangenberg/stratum/internal/plugin/scalar/string"
	"github.com/tstangenberg/stratum/internal/schema"
)

func stringScalars() map[string]scalar.Plugin {
	return map[string]scalar.Plugin{
		"String": stringscalar.Plugin{},
		"ID":     stringscalar.Plugin{},
	}
}

func locationSchema() *schema.ParsedSchema {
	return &schema.ParsedSchema{
		Types: []schema.TypeDef{
			{
				Name: "Location",
				Fields: []schema.FieldDef{
					{Name: "id", Type: "ID", NonNull: true},
					{Name: "name", Type: "String", NonNull: true},
				},
			},
		},
	}
}

func TestBuildHandler_Success(t *testing.T) {
	h, err := schema.BuildHandler(nil, "test", locationSchema(), stringScalars())
	if err != nil {
		t.Fatalf("BuildHandler: %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestBuildHandler_UnknownScalarInField(t *testing.T) {
	ps := &schema.ParsedSchema{
		Types: []schema.TypeDef{
			{Name: "Widget", Fields: []schema.FieldDef{
				{Name: "price", Type: "Float", NonNull: true},
			}},
		},
	}
	_, err := schema.BuildHandler(nil, "test", ps, map[string]scalar.Plugin{})
	if err == nil {
		t.Fatal("expected error for unknown scalar in field")
	}
}

func TestGQLHandler_BadJSON(t *testing.T) {
	h, err := schema.BuildHandler(nil, "test", locationSchema(), stringScalars())
	if err != nil {
		t.Fatalf("BuildHandler: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/graphql/test", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
