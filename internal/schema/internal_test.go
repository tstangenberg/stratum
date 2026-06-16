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
	"github.com/jackc/pgx/v5"
	"github.com/tstangenberg/stratum/internal/plugin"
	eqfilter "github.com/tstangenberg/stratum/internal/plugin/filter/eq"
	"github.com/tstangenberg/stratum/internal/plugin/scalar"
	idscalar "github.com/tstangenberg/stratum/internal/plugin/scalar/id"
	intscalar "github.com/tstangenberg/stratum/internal/plugin/scalar/int"
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

type stubRow struct {
	scanErr error
}

func (r *stubRow) Scan(_ ...any) error { return r.scanErr }

func TestScanGet_NoRows_ReturnsNil(t *testing.T) {
	row := &stubRow{scanErr: pgx.ErrNoRows}
	rec, err := scanGet(row, []string{"id", "name"}, "test_t", "missing-id")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if rec != nil {
		t.Fatalf("expected nil record, got %v", rec)
	}
}

func TestScanGet_ScanError(t *testing.T) {
	scanErr := errors.New("broken scan")
	row := &stubRow{scanErr: scanErr}
	_, err := scanGet(row, []string{"id", "name"}, "test_t", "some-id")
	if err == nil {
		t.Fatal("expected error from scanGet")
	}
	if !strings.Contains(err.Error(), "test_t") || !strings.Contains(err.Error(), "some-id") {
		t.Errorf("error = %q, want it to mention table and id", err)
	}
}

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

func TestIndexFilterPlugins(t *testing.T) {
	filters := []plugin.FilterPlugin{
		eqfilter.New("String", graphql.String),
		eqfilter.New("Int", graphql.Int),
	}
	idx := indexFilterPlugins(filters)
	if len(idx["String"]) != 1 {
		t.Errorf("expected 1 filter for String, got %d", len(idx["String"]))
	}
	if len(idx["Int"]) != 1 {
		t.Errorf("expected 1 filter for Int, got %d", len(idx["Int"]))
	}
	if len(idx["Float"]) != 0 {
		t.Errorf("expected 0 filters for Float, got %d", len(idx["Float"]))
	}
}

func TestBuildFilterInput_NoFilters(t *testing.T) {
	td := TypeDef{
		Name:   "Widget",
		Fields: []FieldDef{{Name: "name", Type: "String"}},
	}
	got := buildFilterInput(td, nil, map[string]scalar.Plugin{"String": stringscalar.Plugin{}})
	if got != nil {
		t.Error("expected nil when no filter plugins registered")
	}
}

func TestBuildFilterInput_SkipsRelations(t *testing.T) {
	td := TypeDef{
		Name: "Ortschaft",
		Fields: []FieldDef{
			{Name: "id", Type: "ID"},
			{Name: "kanton", Type: "Kanton", IsRelation: true},
		},
	}
	scalars := map[string]scalar.Plugin{"ID": idscalar.Plugin{}}
	filters := []plugin.FilterPlugin{eqfilter.New("ID", graphql.ID)}
	input := buildFilterInput(td, filters, scalars)
	if input == nil {
		t.Fatal("expected non-nil filter input")
	}
	fields := input.Fields()
	if _, ok := fields["kanton"]; ok {
		t.Error("relation field 'kanton' should not appear in filter input")
	}
	if _, ok := fields["id"]; !ok {
		t.Error("scalar field 'id' should appear in filter input")
	}
}

func TestBuildFilterInput_WithFilters(t *testing.T) {
	td := TypeDef{
		Name: "PLZ",
		Fields: []FieldDef{
			{Name: "id", Type: "ID"},
			{Name: "plz", Type: "Int"},
			{Name: "name", Type: "String"},
		},
	}
	scalars := map[string]scalar.Plugin{
		"ID":     idscalar.Plugin{},
		"Int":    intscalar.Plugin{},
		"String": stringscalar.Plugin{},
	}
	filters := []plugin.FilterPlugin{
		eqfilter.New("ID", graphql.ID),
		eqfilter.New("Int", scalars["Int"].GraphQLType()),
		eqfilter.New("String", graphql.String),
	}
	input := buildFilterInput(td, filters, scalars)
	if input == nil {
		t.Fatal("expected non-nil filter input")
	}
	fields := input.Fields()
	for _, name := range []string{"id", "plz", "name"} {
		if _, ok := fields[name]; !ok {
			t.Errorf("expected field %q in filter input", name)
		}
	}
}

func TestApplyFilters_NoFilter(t *testing.T) {
	fields := []FieldDef{{Name: "id", Type: "ID"}}
	idx := indexFilterPlugins(nil)
	clauses, params, err := applyFilters(map[string]any{}, fields, idx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clauses) != 0 {
		t.Errorf("expected 0 clauses, got %d", len(clauses))
	}
	if len(params) != 0 {
		t.Errorf("expected 0 params, got %d", len(params))
	}
}

func TestApplyFilters_EqFilter(t *testing.T) {
	fields := []FieldDef{
		{Name: "id", Type: "ID"},
		{Name: "plz", Type: "Int"},
		{Name: "name", Type: "String"},
	}
	filters := []plugin.FilterPlugin{
		eqfilter.New("Int", graphql.Int),
		eqfilter.New("String", graphql.String),
	}
	idx := indexFilterPlugins(filters)
	args := map[string]any{
		"filter": map[string]any{
			"plz": map[string]any{"eq": 8001},
		},
	}
	clauses, params, err := applyFilters(args, fields, idx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clauses) != 1 {
		t.Fatalf("expected 1 clause, got %d", len(clauses))
	}
	if clauses[0] != "plz = $1" {
		t.Errorf("clause = %q, want %q", clauses[0], "plz = $1")
	}
	if len(params) != 1 || params[0] != 8001 {
		t.Errorf("params = %v, want [8001]", params)
	}
}

func TestApplyFilters_MultipleFields(t *testing.T) {
	fields := []FieldDef{
		{Name: "plz", Type: "Int"},
		{Name: "name", Type: "String"},
	}
	filters := []plugin.FilterPlugin{
		eqfilter.New("Int", graphql.Int),
		eqfilter.New("String", graphql.String),
	}
	idx := indexFilterPlugins(filters)
	args := map[string]any{
		"filter": map[string]any{
			"plz":  map[string]any{"eq": 8001},
			"name": map[string]any{"eq": "Zürich"},
		},
	}
	clauses, params, err := applyFilters(args, fields, idx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clauses) != 2 {
		t.Fatalf("expected 2 clauses, got %d", len(clauses))
	}
	if len(params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(params))
	}
}

func TestApplyFilters_NilValue(t *testing.T) {
	fields := []FieldDef{{Name: "plz", Type: "Int"}}
	filters := []plugin.FilterPlugin{eqfilter.New("Int", graphql.Int)}
	idx := indexFilterPlugins(filters)
	args := map[string]any{
		"filter": map[string]any{
			"plz": map[string]any{"eq": nil},
		},
	}
	clauses, _, err := applyFilters(args, fields, idx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clauses) != 0 {
		t.Errorf("expected 0 clauses for nil value, got %d", len(clauses))
	}
}

func TestApplyFilters_WithExistingParams(t *testing.T) {
	fields := []FieldDef{{Name: "plz", Type: "Int"}}
	filters := []plugin.FilterPlugin{eqfilter.New("Int", graphql.Int)}
	idx := indexFilterPlugins(filters)
	args := map[string]any{
		"filter": map[string]any{
			"plz": map[string]any{"eq": 3000},
		},
	}
	existingParams := []any{"existing"}
	clauses, params, err := applyFilters(args, fields, idx, existingParams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clauses) != 1 {
		t.Fatalf("expected 1 clause, got %d", len(clauses))
	}
	if clauses[0] != "plz = $2" {
		t.Errorf("clause = %q, want %q", clauses[0], "plz = $2")
	}
	if len(params) != 2 || params[0] != "existing" || params[1] != 3000 {
		t.Errorf("params = %v, want [existing 3000]", params)
	}
}

func TestApplyFilters_ErrorFromPlugin(t *testing.T) {
	fields := []FieldDef{{Name: "plz", Type: "Int"}}
	filters := []plugin.FilterPlugin{eqfilter.New("Int", graphql.Int)}
	idx := indexFilterPlugins(filters)
	args := map[string]any{
		"filter": map[string]any{
			"plz": map[string]any{"gte": 100},
		},
	}
	_, _, err := applyFilters(args, fields, idx, nil)
	if err == nil {
		t.Fatal("expected error from unsupported operator")
	}
	if !strings.Contains(err.Error(), "plz.gte") {
		t.Errorf("error = %q, want it to mention plz.gte", err)
	}
}

func TestBuildFilterInput_EmptyOperators(t *testing.T) {
	td := TypeDef{
		Name:   "Widget",
		Fields: []FieldDef{{Name: "status", Type: "Unknown"}},
	}
	scalars := map[string]scalar.Plugin{"Unknown": stringscalar.Plugin{}}
	// No filter plugin for "Unknown" type → no filter input
	filters := []plugin.FilterPlugin{eqfilter.New("String", graphql.String)}
	got := buildFilterInput(td, filters, scalars)
	if got != nil {
		t.Error("expected nil filter input when field type has no matching filter plugin")
	}
}

type emptyOpsFilter struct{}

func (emptyOpsFilter) Name() string       { return "empty-ops" }
func (emptyOpsFilter) ScalarType() string { return "String" }
func (emptyOpsFilter) Operators(_ graphql.Output) graphql.InputObjectConfigFieldMap {
	return graphql.InputObjectConfigFieldMap{}
}
func (emptyOpsFilter) ToSQL(string, string, any, int) (string, []any, error) {
	return "", nil, nil
}

func TestBuildFilterInput_EmptyOperatorsFromPlugin(t *testing.T) {
	td := TypeDef{
		Name:   "Widget",
		Fields: []FieldDef{{Name: "name", Type: "String"}},
	}
	scalars := map[string]scalar.Plugin{"String": stringscalar.Plugin{}}
	filters := []plugin.FilterPlugin{emptyOpsFilter{}}
	got := buildFilterInput(td, filters, scalars)
	if got != nil {
		t.Error("expected nil filter input when plugin returns empty operators")
	}
}

func TestApplyFilters_SkipsRelationFields(t *testing.T) {
	fields := []FieldDef{
		{Name: "kanton", Type: "Kanton", IsRelation: true},
		{Name: "name", Type: "String"},
	}
	filters := []plugin.FilterPlugin{eqfilter.New("String", graphql.String)}
	idx := indexFilterPlugins(filters)
	args := map[string]any{
		"filter": map[string]any{
			"kanton": map[string]any{"eq": "ZH"},
		},
	}
	clauses, _, err := applyFilters(args, fields, idx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clauses) != 0 {
		t.Errorf("expected 0 clauses for relation field, got %d", len(clauses))
	}
}
