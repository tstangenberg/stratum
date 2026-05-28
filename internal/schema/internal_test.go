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

package schema

import (
	"regexp"
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/tstangenberg/stratum/internal/plugin/scalar"
	stringscalar "github.com/tstangenberg/stratum/internal/plugin/scalar/string"
)

func TestTableName(t *testing.T) {
	tests := []struct {
		schema, typ, want string
	}{
		{"locations", "Location", "locations_location"},
		{"my_schema", "Widget", "my_schema_widget"},
		{"s", "T", "s_t"},
	}
	for _, tt := range tests {
		got := tableName(tt.schema, tt.typ)
		if got != tt.want {
			t.Errorf("tableName(%q, %q) = %q, want %q", tt.schema, tt.typ, got, tt.want)
		}
	}
}

func TestFieldNames(t *testing.T) {
	td := TypeDef{
		Name: "Location",
		Fields: []FieldDef{
			{Name: "id"},
			{Name: "name"},
		},
	}
	got := fieldNames(td)
	if len(got) != 2 || got[0] != "id" || got[1] != "name" {
		t.Errorf("fieldNames() = %v, want [id name]", got)
	}
}

func TestNewID(t *testing.T) {
	uuidRe := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	for range 10 {
		id := newID()
		if !uuidRe.MatchString(id) {
			t.Errorf("newID() = %q, does not match UUID v4 pattern", id)
		}
	}
}

func TestScalarToGraphQL_IDField(t *testing.T) {
	f := FieldDef{Name: "id", Type: "ID", NonNull: true}
	out, err := scalarToGraphQL(f, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	nn, ok := out.(*graphql.NonNull)
	if !ok {
		t.Fatalf("expected NonNull, got %T", out)
	}
	if nn.OfType != graphql.ID {
		t.Errorf("OfType = %v, want graphql.ID", nn.OfType)
	}
}

func TestScalarToGraphQL_NullableString(t *testing.T) {
	f := FieldDef{Name: "description", Type: "String", NonNull: false}
	scalars := map[string]scalar.Plugin{"String": stringscalar.Plugin{}}
	out, err := scalarToGraphQL(f, scalars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != graphql.String {
		t.Errorf("got %v, want graphql.String", out)
	}
}

func TestScalarToGraphQL_UnknownScalar(t *testing.T) {
	f := FieldDef{Name: "price", Type: "Float", NonNull: true}
	_, err := scalarToGraphQL(f, map[string]scalar.Plugin{})
	if err == nil {
		t.Fatal("expected error for unknown scalar")
	}
}

func TestScalarToGraphQL_IDType(t *testing.T) {
	f := FieldDef{Name: "ref", Type: "ID", NonNull: false}
	out, err := scalarToGraphQL(f, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != graphql.ID {
		t.Errorf("got %v, want graphql.ID", out)
	}
}
