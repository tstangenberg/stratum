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
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/tstangenberg/stratum/internal/plugin/scalar"
	idscalar "github.com/tstangenberg/stratum/internal/plugin/scalar/id"
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
	out, err := scalarToGraphQL(f, map[string]scalar.Plugin{"ID": idscalar.Plugin{}})
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
	out, err := scalarToGraphQL(f, map[string]scalar.Plugin{"ID": idscalar.Plugin{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != graphql.ID {
		t.Errorf("got %v, want graphql.ID", out)
	}
}

type stubRows struct {
	nextN   int
	called  int
	scanErr error
	closed  bool
}

func (r *stubRows) Close()              { r.closed = true }
func (r *stubRows) Err() error          { return nil }
func (r *stubRows) Next() bool          { r.called++; return r.called <= r.nextN }
func (r *stubRows) Scan(_ ...any) error { return r.scanErr }

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"kanton", "kanton"},
		{"billingAddress", "billing_address"},
		{"shippingAddress", "shipping_address"},
		{"ortschaft", "ortschaft"},
		{"myFieldName", "my_field_name"},
		{"id", "id"},
		{"PLZ", "plz"},
		{"XMLParser", "xml_parser"},
		{"ABCDef", "abc_def"},
	}
	for _, tt := range tests {
		got := camelToSnake(tt.input)
		if got != tt.want {
			t.Errorf("camelToSnake(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFKColumnName(t *testing.T) {
	tests := []struct {
		fieldName, want string
	}{
		{"kanton", "kanton_id"},
		{"billingAddress", "billing_address_id"},
		{"ortschaft", "ortschaft_id"},
	}
	for _, tt := range tests {
		got := fkColumnName(tt.fieldName)
		if got != tt.want {
			t.Errorf("fkColumnName(%q) = %q, want %q", tt.fieldName, got, tt.want)
		}
	}
}

func TestFKInputName(t *testing.T) {
	tests := []struct {
		fieldName, want string
	}{
		{"kanton", "kantonId"},
		{"billingAddress", "billingAddressId"},
		{"ortschaft", "ortschaftId"},
	}
	for _, tt := range tests {
		got := fkInputName(tt.fieldName)
		if got != tt.want {
			t.Errorf("fkInputName(%q) = %q, want %q", tt.fieldName, got, tt.want)
		}
	}
}

func TestColumnNames(t *testing.T) {
	td := TypeDef{
		Name: "Ortschaft",
		Fields: []FieldDef{
			{Name: "id", Type: "ID"},
			{Name: "name", Type: "String"},
			{Name: "kanton", Type: "Kanton", IsRelation: true},
		},
	}
	got := columnNames(td)
	want := []string{"id", "name", "kanton_id"}
	if len(got) != len(want) {
		t.Fatalf("columnNames() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("columnNames()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestTopoSort_CircularReference(t *testing.T) {
	byName := map[string]TypeDef{
		"A": {Name: "A", Fields: []FieldDef{{Name: "b", Type: "B", IsRelation: true}}},
		"B": {Name: "B", Fields: []FieldDef{{Name: "a", Type: "A", IsRelation: true}}},
	}
	_, err := topoSort(byName)
	if err == nil {
		t.Fatal("expected error for circular reference")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("error = %q, want it to mention circular", err)
	}
}

func TestResolveRelation_NonMapSource(t *testing.T) {
	resolver := resolveRelation(nil, "tbl", []string{"id"}, "fk_id")
	p := graphql.ResolveParams{Source: "not-a-map"}
	got, err := resolver(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestResolveRelation_MissingFK(t *testing.T) {
	resolver := resolveRelation(nil, "tbl", []string{"id"}, "fk_id")
	p := graphql.ResolveParams{Source: map[string]any{"id": "x"}}
	got, err := resolver(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestResolveRelation_EmptyFK(t *testing.T) {
	resolver := resolveRelation(nil, "tbl", []string{"id"}, "fk_id")
	p := graphql.ResolveParams{Source: map[string]any{"fk_id": ""}}
	got, err := resolver(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestScanList_ScanError(t *testing.T) {
	scanErr := errors.New("broken scan")
	rows := &stubRows{nextN: 1, scanErr: scanErr}
	_, err := scanList(rows, []string{"id", "name"}, "test_t")
	if err == nil {
		t.Fatal("expected error from scanList")
	}
	if !strings.Contains(err.Error(), "scan") {
		t.Errorf("error = %q, want it to mention scan", err)
	}
	if !rows.closed {
		t.Error("expected rows to be closed")
	}
}
