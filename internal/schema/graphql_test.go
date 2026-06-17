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

package schema_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gql "github.com/graphql-go/graphql"
	"github.com/tstangenberg/stratum/internal/plugin"
	"github.com/tstangenberg/stratum/internal/plugin/pagination/simple"
	"github.com/tstangenberg/stratum/internal/plugin/scalar"
	idscalar "github.com/tstangenberg/stratum/internal/plugin/scalar/id"
	stringscalar "github.com/tstangenberg/stratum/internal/plugin/scalar/string"
	"github.com/tstangenberg/stratum/internal/schema"
)

func stringScalars() map[string]scalar.Plugin {
	return map[string]scalar.Plugin{
		"String": stringscalar.Plugin{},
		"ID":     idscalar.Plugin{},
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

type stubModifier struct{ argKey string }

func (s stubModifier) Name() string { return "stub" }
func (s stubModifier) Arguments(intType gql.Output) gql.FieldConfigArgument {
	if s.argKey == "" {
		return nil
	}
	return gql.FieldConfigArgument{s.argKey: &gql.ArgumentConfig{Type: intType}}
}
func (s stubModifier) ModifyQuery(q string, p []any, _ map[string]any) (string, []any, error) {
	return q, p, nil
}

func TestBuildHandler_DuplicateModifierArg_ReturnsError(t *testing.T) {
	_, err := schema.BuildHandler(nil, "test", locationSchema(), stringScalars(), []plugin.QueryModifier{
		stubModifier{argKey: "limit"},
		stubModifier{argKey: "limit"},
	}, nil)
	if err == nil {
		t.Fatal("expected error for duplicate modifier argument")
	}
}

func TestBuildHandler_Success(t *testing.T) {
	h, err := schema.BuildHandler(nil, "test", locationSchema(), stringScalars(), []plugin.QueryModifier{simple.New()}, nil)
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
	_, err := schema.BuildHandler(nil, "test", ps, map[string]scalar.Plugin{}, []plugin.QueryModifier{simple.New()}, nil)
	if err == nil {
		t.Fatal("expected error for unknown scalar in field")
	}
}

func TestBuildHandler_SchemaConstructionError(t *testing.T) {
	ps := &schema.ParsedSchema{
		Types: []schema.TypeDef{
			{Name: "Query", Fields: []schema.FieldDef{
				{Name: "id", Type: "ID", NonNull: true},
				{Name: "name", Type: "String", NonNull: true},
			}},
		},
	}
	_, err := schema.BuildHandler(nil, "test", ps, stringScalars(), []plugin.QueryModifier{simple.New()}, nil)
	if err == nil {
		t.Fatal("expected error when type name collides with root Query")
	}
}

func TestBuildHandler_ListRelation_MissingReverseFK(t *testing.T) {
	ps := &schema.ParsedSchema{
		Types: []schema.TypeDef{
			{Name: "Kanton", Fields: []schema.FieldDef{
				{Name: "id", Type: "ID"},
				{Name: "ortschaften", Type: "Ortschaft", IsRelation: true, IsList: true},
			}},
			{Name: "Ortschaft", Fields: []schema.FieldDef{
				{Name: "id", Type: "ID"},
				{Name: "name", Type: "String"},
			}},
		},
	}
	_, err := schema.BuildHandler(nil, "test", ps, stringScalars(), []plugin.QueryModifier{simple.New()}, nil)
	if err == nil {
		t.Fatal("expected error for missing reverse FK")
	}
	if !strings.Contains(err.Error(), "no reverse FK") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "no reverse FK")
	}
}

func TestGQLHandler_BadJSON(t *testing.T) {
	h, err := schema.BuildHandler(nil, "test", locationSchema(), stringScalars(), []plugin.QueryModifier{simple.New()}, nil)
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
