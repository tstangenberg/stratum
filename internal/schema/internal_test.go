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
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
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
	resolver := resolveRelation(nil, "tbl", []string{"id"}, "fk_id", "fk")
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
	resolver := resolveRelation(nil, "tbl", []string{"id"}, "fk_id", "fk")
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
	resolver := resolveRelation(nil, "tbl", []string{"id"}, "fk_id", "fk")
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
	got := buildFilterInput(td, indexFilterPlugins(nil))
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
	filters := []plugin.FilterPlugin{eqfilter.New("ID", graphql.ID)}
	input := buildFilterInput(td, indexFilterPlugins(filters))
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
	input := buildFilterInput(td, indexFilterPlugins(filters))
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
	clauses, params, err := applyFilters(map[string]any{}, fields, idx, nil, "")
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
	clauses, params, err := applyFilters(args, fields, idx, nil, "")
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
	clauses, params, err := applyFilters(args, fields, idx, nil, "")
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
	clauses, _, err := applyFilters(args, fields, idx, nil, "")
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
	clauses, params, err := applyFilters(args, fields, idx, existingParams, "")
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
	_, _, err := applyFilters(args, fields, idx, nil, "")
	if err == nil {
		t.Fatal("expected error from unsupported operator")
	}
	if !strings.Contains(err.Error(), `"plz"."gte"`) {
		t.Errorf("error = %q, want it to mention plz.gte", err)
	}
}

func TestBuildFilterInput_EmptyOperators(t *testing.T) {
	td := TypeDef{
		Name:   "Widget",
		Fields: []FieldDef{{Name: "status", Type: "Unknown"}},
	}
	// No filter plugin for "Unknown" type → no filter input
	filters := []plugin.FilterPlugin{eqfilter.New("String", graphql.String)}
	got := buildFilterInput(td, indexFilterPlugins(filters))
	if got != nil {
		t.Error("expected nil filter input when field type has no matching filter plugin")
	}
}

type emptyOpsFilter struct{}

func (emptyOpsFilter) Name() string       { return "empty-ops" }
func (emptyOpsFilter) ScalarType() string { return "String" }
func (emptyOpsFilter) Operators() graphql.InputObjectConfigFieldMap {
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
	filters := []plugin.FilterPlugin{emptyOpsFilter{}}
	got := buildFilterInput(td, indexFilterPlugins(filters))
	if got != nil {
		t.Error("expected nil filter input when plugin returns empty operators")
	}
}

func TestReverseFK_Found(t *testing.T) {
	child := TypeDef{
		Name: "Ortschaft",
		Fields: []FieldDef{
			{Name: "id", Type: "ID"},
			{Name: "name", Type: "String"},
			{Name: "kanton", Type: "Kanton", IsRelation: true},
		},
	}
	got := reverseFK(child, "Kanton")
	if got != "kanton_id" {
		t.Errorf("reverseFK() = %q, want %q", got, "kanton_id")
	}
}

func TestReverseFK_NotFound(t *testing.T) {
	child := TypeDef{
		Name: "Ortschaft",
		Fields: []FieldDef{
			{Name: "id", Type: "ID"},
			{Name: "name", Type: "String"},
		},
	}
	got := reverseFK(child, "Kanton")
	if got != "" {
		t.Errorf("reverseFK() = %q, want empty", got)
	}
}

func TestReverseFK_SkipsListRelation(t *testing.T) {
	child := TypeDef{
		Name: "Ortschaft",
		Fields: []FieldDef{
			{Name: "id", Type: "ID"},
			{Name: "kantone", Type: "Kanton", IsRelation: true, IsList: true},
		},
	}
	got := reverseFK(child, "Kanton")
	if got != "" {
		t.Errorf("reverseFK() = %q, want empty (should skip list)", got)
	}
}

func TestColumnNames_SkipsListRelation(t *testing.T) {
	td := TypeDef{
		Name: "Kanton",
		Fields: []FieldDef{
			{Name: "id", Type: "ID"},
			{Name: "kuerzel", Type: "String"},
			{Name: "ortschaften", Type: "Ortschaft", IsRelation: true, IsList: true},
		},
	}
	got := columnNames(td)
	want := []string{"id", "kuerzel"}
	if len(got) != len(want) {
		t.Fatalf("columnNames() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("columnNames()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBuildChildSubqueries_NoListRelations(t *testing.T) {
	td := TypeDef{
		Name: "Location",
		Fields: []FieldDef{
			{Name: "id", Type: "ID"},
			{Name: "name", Type: "String"},
		},
	}
	subs, err := buildChildSubqueries(td, "test", map[string]TypeDef{"Location": td}, "test_location")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 0 {
		t.Errorf("expected 0 subqueries, got %d", len(subs))
	}
}

func TestBuildChildSubqueries_WithListRelation(t *testing.T) {
	kantonTD := TypeDef{
		Name: "Kanton",
		Fields: []FieldDef{
			{Name: "id", Type: "ID"},
			{Name: "kuerzel", Type: "String"},
			{Name: "ortschaften", Type: "Ortschaft", IsRelation: true, IsList: true},
		},
	}
	ortTD := TypeDef{
		Name: "Ortschaft",
		Fields: []FieldDef{
			{Name: "id", Type: "ID"},
			{Name: "name", Type: "String"},
			{Name: "kanton", Type: "Kanton", IsRelation: true},
		},
	}
	typeIndex := map[string]TypeDef{"Kanton": kantonTD, "Ortschaft": ortTD}
	subs, err := buildChildSubqueries(kantonTD, "swiss", typeIndex, "t0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("expected 1 subquery, got %d", len(subs))
	}
	if subs[0].fieldName != "ortschaften" {
		t.Errorf("fieldName = %q, want %q", subs[0].fieldName, "ortschaften")
	}
	if !strings.Contains(subs[0].sql, "json_agg") {
		t.Error("subquery should contain json_agg")
	}
	if !strings.Contains(subs[0].sql, "kanton_id") {
		t.Error("subquery should reference kanton_id FK")
	}
	if !strings.Contains(subs[0].sql, "t0.id") {
		t.Error("subquery should reference parent alias")
	}
}

func TestBuildChildSubqueries_MissingReverseFK(t *testing.T) {
	parentTD := TypeDef{
		Name: "Kanton",
		Fields: []FieldDef{
			{Name: "id", Type: "ID"},
			{Name: "ortschaften", Type: "Ortschaft", IsRelation: true, IsList: true},
		},
	}
	childTD := TypeDef{
		Name: "Ortschaft",
		Fields: []FieldDef{
			{Name: "id", Type: "ID"},
			{Name: "name", Type: "String"},
		},
	}
	typeIndex := map[string]TypeDef{"Kanton": parentTD, "Ortschaft": childTD}
	_, err := buildChildSubqueries(parentTD, "test", typeIndex, "test_kanton")
	if err == nil {
		t.Fatal("expected error for missing reverse FK")
	}
	if !strings.Contains(err.Error(), "no reverse FK") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "no reverse FK")
	}
}

func TestResolveChildren_PreLoaded(t *testing.T) {
	preloaded := []map[string]any{{"id": "1", "name": "Zürich"}}
	resolver := resolveChildren(nil, "tbl", nil, "", "ortschaften")
	p := graphql.ResolveParams{
		Source: map[string]any{"ortschaften": preloaded},
	}
	got, err := resolver(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, ok := got.([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any, got %T", got)
	}
	if len(result) != 1 || result[0]["name"] != "Zürich" {
		t.Errorf("got %v, want [{name:Zürich}]", result)
	}
}

func TestResolveChildren_PreLoadedNil(t *testing.T) {
	resolver := resolveChildren(nil, "tbl", nil, "", "ortschaften")
	p := graphql.ResolveParams{
		Source: map[string]any{"ortschaften": nil},
	}
	got, err := resolver(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, ok := got.([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any, got %T", got)
	}
	if len(result) != 0 {
		t.Errorf("expected empty, got %d items", len(result))
	}
}

func TestResolveChildren_NonMapSource(t *testing.T) {
	resolver := resolveChildren(nil, "tbl", nil, "", "ortschaften")
	p := graphql.ResolveParams{Source: "not-a-map"}
	got, err := resolver(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, ok := got.([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any, got %T", got)
	}
	if len(result) != 0 {
		t.Errorf("expected empty, got %d items", len(result))
	}
}

func TestResolveChildren_MissingParentID(t *testing.T) {
	resolver := resolveChildren(nil, "tbl", nil, "", "children")
	p := graphql.ResolveParams{
		Source: map[string]any{"id": 42}, // not a string
	}
	got, err := resolver(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, ok := got.([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any, got %T", got)
	}
	if len(result) != 0 {
		t.Errorf("expected empty, got %d items", len(result))
	}
}

func TestParseJSONChildren_SliceAny(t *testing.T) {
	input := []any{
		map[string]any{"id": "1", "name": "Zürich"},
		map[string]any{"id": "2", "name": "Bern"},
	}
	got, err := parseJSONChildren(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	if got[0]["name"] != "Zürich" || got[1]["name"] != "Bern" {
		t.Errorf("got %v", got)
	}
}

func TestParseJSONChildren_EmptySlice(t *testing.T) {
	got, err := parseJSONChildren([]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %d", len(got))
	}
}

func TestParseJSONChildren_Bytes(t *testing.T) {
	data := []byte(`[{"id":"1","name":"Zürich"}]`)
	got, err := parseJSONChildren(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0]["name"] != "Zürich" {
		t.Errorf("got %v", got)
	}
}

func TestParseJSONChildren_String(t *testing.T) {
	got, err := parseJSONChildren(`[{"id":"1"}]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 item, got %d", len(got))
	}
}

func TestParseJSONChildren_EmptyJSON(t *testing.T) {
	got, err := parseJSONChildren([]byte(`[]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %d", len(got))
	}
}

func TestParseJSONChildren_UnexpectedType(t *testing.T) {
	_, err := parseJSONChildren(42)
	if err == nil {
		t.Fatal("expected error for unexpected type")
	}
	if !strings.Contains(err.Error(), "unexpected type") {
		t.Errorf("error = %q, want it to mention unexpected type", err)
	}
}

func TestParseJSONChildren_NonMapInArray(t *testing.T) {
	input := []any{"not-a-map"}
	_, err := parseJSONChildren(input)
	if err == nil {
		t.Fatal("expected error for non-map item in array")
	}
	if !strings.Contains(err.Error(), "expected map") {
		t.Errorf("error = %q, want it to mention expected map", err)
	}
}

func TestParseJSONChildren_InvalidJSON(t *testing.T) {
	_, err := parseJSONChildren([]byte(`not-json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseJSONChildren_BytesNull(t *testing.T) {
	got, err := parseJSONChildren([]byte(`null`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %d", len(got))
	}
}

func TestParseJSONChildren_StringInvalidJSON(t *testing.T) {
	_, err := parseJSONChildren("not-json")
	if err == nil {
		t.Fatal("expected error for invalid JSON string")
	}
}

func TestParseJSONChildren_StringNull(t *testing.T) {
	got, err := parseJSONChildren("null")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %d", len(got))
	}
}

func TestScanListWithChildren_ScanError(t *testing.T) {
	scanErr := errors.New("broken scan")
	rows := &stubRows{nextN: 1, scanErr: scanErr}
	_, err := scanListWithChildren(rows, []string{"id"}, []string{"children"}, "test_t")
	if err == nil {
		t.Fatal("expected error from scanListWithChildren")
	}
	if !strings.Contains(err.Error(), "scan") {
		t.Errorf("error = %q, want it to mention scan", err)
	}
}

func TestScanListWithChildren_Empty(t *testing.T) {
	rows := &stubRows{nextN: 0}
	result, err := scanListWithChildren(rows, []string{"id"}, []string{"children"}, "test_t")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty, got %d", len(result))
	}
}

type stubRowsWithErr struct {
	stubRows
	rowErr error
}

func (r *stubRowsWithErr) Err() error { return r.rowErr }

func TestScanListWithChildren_RowsErr(t *testing.T) {
	rowErr := errors.New("rows iteration error")
	rows := &stubRowsWithErr{stubRows: stubRows{nextN: 0}, rowErr: rowErr}
	_, err := scanListWithChildren(rows, []string{"id"}, []string{"children"}, "test_t")
	if err == nil {
		t.Fatal("expected error from rows.Err()")
	}
	if !errors.Is(err, rowErr) {
		t.Errorf("error = %v, want %v", err, rowErr)
	}
}

type scanWithValsRows struct {
	nextN  int
	called int
	vals   [][]any
	closed bool
}

func (r *scanWithValsRows) Close()     { r.closed = true }
func (r *scanWithValsRows) Err() error { return nil }
func (r *scanWithValsRows) Next() bool { r.called++; return r.called <= r.nextN }
func (r *scanWithValsRows) Scan(dest ...any) error {
	row := r.vals[r.called-1]
	for i, v := range row {
		ptr := dest[i].(*any)
		*ptr = v
	}
	return nil
}

func TestScanListWithChildren_HappyPath(t *testing.T) {
	rows := &scanWithValsRows{
		nextN: 2,
		vals: [][]any{
			{"id1", "ZH", []any{map[string]any{"id": "o1", "name": "Zürich"}}},
			{"id2", "BE", []any{}},
		},
	}
	result, err := scanListWithChildren(rows, []string{"id", "kuerzel"}, []string{"ortschaften"}, "test_t")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result))
	}
	if result[0]["kuerzel"] != "ZH" {
		t.Errorf("row 0 kuerzel = %v, want ZH", result[0]["kuerzel"])
	}
	ort, ok := result[0]["ortschaften"].([]map[string]any)
	if !ok {
		t.Fatalf("ortschaften: expected []map[string]any, got %T", result[0]["ortschaften"])
	}
	if len(ort) != 1 || ort[0]["name"] != "Zürich" {
		t.Errorf("ortschaften = %v, want [{name:Zürich}]", ort)
	}
	beOrt, ok := result[1]["ortschaften"].([]map[string]any)
	if !ok {
		t.Fatalf("BE ortschaften: expected []map[string]any, got %T", result[1]["ortschaften"])
	}
	if len(beOrt) != 0 {
		t.Errorf("BE ortschaften: expected empty, got %d", len(beOrt))
	}
}

func TestScanListWithChildren_ParseError(t *testing.T) {
	rows := &scanWithValsRows{
		nextN: 1,
		vals:  [][]any{{"id1", 42}}, // 42 is not parseable as JSON children
	}
	_, err := scanListWithChildren(rows, []string{"id"}, []string{"children"}, "test_t")
	if err == nil {
		t.Fatal("expected error for unparseable children")
	}
	if !strings.Contains(err.Error(), "parse children") {
		t.Errorf("error = %q, want it to mention parse children", err)
	}
}

type stubQuerier struct {
	rows scannable
	err  error
}

func (q *stubQuerier) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	if q.err != nil {
		return nil, q.err
	}
	return q.rows.(pgx.Rows), q.err
}

func TestResolveChildren_FallbackToDB_Error(t *testing.T) {
	dbErr := errors.New("db query error")
	mock := &stubQuerier{err: dbErr}
	resolver := resolveChildren(mock, "test_ortschaft", []string{"id", "name"}, "kanton_id", "ortschaften")
	p := graphql.ResolveParams{
		Source:  map[string]any{"id": "parent-1"},
		Context: context.Background(),
	}
	_, err := resolver(p)
	if err == nil {
		t.Fatal("expected error from DB query")
	}
	if !strings.Contains(err.Error(), "list children") {
		t.Errorf("error = %q, want it to mention list children", err)
	}
}

func TestApplyFilters_WithTableAlias(t *testing.T) {
	fields := []FieldDef{{Name: "name", Type: "String"}}
	filters := []plugin.FilterPlugin{eqfilter.New("String", graphql.String)}
	idx := indexFilterPlugins(filters)
	args := map[string]any{
		"filter": map[string]any{
			"name": map[string]any{"eq": "Zürich"},
		},
	}
	clauses, _, err := applyFilters(args, fields, idx, nil, "t0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clauses) != 1 {
		t.Fatalf("expected 1 clause, got %d", len(clauses))
	}
	if clauses[0] != "t0.name = $1" {
		t.Errorf("clause = %q, want %q", clauses[0], "t0.name = $1")
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
	clauses, _, err := applyFilters(args, fields, idx, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clauses) != 0 {
		t.Errorf("expected 0 clauses for relation field, got %d", len(clauses))
	}
}

func TestAssembleNested_IDNotFirstColumn(t *testing.T) {
	// A type where id is NOT the first field — verifies the null sentinel uses id, not position 0.
	widget := TypeDef{
		Name: "Widget",
		Fields: []FieldDef{
			{Name: "label", Type: "String", NonNull: false}, // nullable, comes first
			{Name: "id", Type: "ID", NonNull: true},
		},
	}
	parent := TypeDef{
		Name: "Parent",
		Fields: []FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
			{Name: "widget", Type: "Widget", IsRelation: true, NonNull: true},
		},
	}
	idx := map[string]TypeDef{"Widget": widget, "Parent": parent}
	seq := 0
	nodes := buildJoinNodes(parent, "test", idx, "t0", 0, 5, &seq)

	parentCols := columnNames(parent) // ["id", "widget_id"]

	// Simulate a real widget row where label IS null but id is non-null ("w1").
	// The old code would use joinVals[0] (label=nil) as sentinel and discard the row.
	// The fixed code uses the "id" column explicitly and sees id="w1", so keeps the row.
	vals := []any{
		"p1", // parent id
		"w1", // widget_id (FK)
		nil,  // j1__label (nullable — this widget has no label)
		"w1", // j1__id
	}

	row := assembleJoinedRows(vals, parentCols, nodes)
	widgetVal, ok := row["widget"]
	if !ok {
		t.Fatal("expected widget key to be present")
	}
	if widgetVal == nil {
		t.Fatal("expected widget to be non-nil (label=nil is not a missing row), but got nil")
	}
	widgetMap, ok := widgetVal.(map[string]any)
	if !ok {
		t.Fatalf("expected widget to be map[string]any, got %T", widgetVal)
	}
	if widgetMap["id"] != "w1" {
		t.Errorf("widget.id = %v, want w1", widgetMap["id"])
	}
}

// ── Join plan tests ─────────────────────────────────────────────────────────

func plzOrtschaftKantonTypes() (TypeDef, TypeDef, TypeDef, map[string]TypeDef) {
	kanton := TypeDef{
		Name: "Kanton",
		Fields: []FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
			{Name: "kuerzel", Type: "String", NonNull: true},
		},
	}
	ortschaft := TypeDef{
		Name: "Ortschaft",
		Fields: []FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
			{Name: "name", Type: "String", NonNull: true},
			{Name: "kanton", Type: "Kanton", IsRelation: true, NonNull: true},
		},
	}
	plz := TypeDef{
		Name: "PLZ",
		Fields: []FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
			{Name: "plz", Type: "Int", NonNull: true},
			{Name: "ortschaft", Type: "Ortschaft", IsRelation: true, NonNull: true},
		},
	}
	idx := map[string]TypeDef{"Kanton": kanton, "Ortschaft": ortschaft, "PLZ": plz}
	return plz, ortschaft, kanton, idx
}

func TestBuildJoinNodes_TwoHopChain(t *testing.T) {
	plz, _, _, idx := plzOrtschaftKantonTypes()
	seq := 0
	nodes := buildJoinNodes(plz, "swiss", idx, "t0", 0, 5, &seq)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 top-level join node, got %d", len(nodes))
	}
	ort := nodes[0]
	if ort.fieldName != "ortschaft" {
		t.Errorf("fieldName = %q, want ortschaft", ort.fieldName)
	}
	if ort.alias != "j1" {
		t.Errorf("alias = %q, want j1", ort.alias)
	}
	if ort.table != "swiss_ortschaft" {
		t.Errorf("table = %q, want swiss_ortschaft", ort.table)
	}
	if ort.fkCol != "ortschaft_id" {
		t.Errorf("fkCol = %q, want ortschaft_id", ort.fkCol)
	}
	if ort.parentAlias != "t0" {
		t.Errorf("parentAlias = %q, want t0", ort.parentAlias)
	}
	if len(ort.children) != 1 {
		t.Fatalf("expected 1 child join node, got %d", len(ort.children))
	}
	kan := ort.children[0]
	if kan.fieldName != "kanton" {
		t.Errorf("child fieldName = %q, want kanton", kan.fieldName)
	}
	if kan.alias != "j2" {
		t.Errorf("child alias = %q, want j2", kan.alias)
	}
	if kan.table != "swiss_kanton" {
		t.Errorf("child table = %q, want swiss_kanton", kan.table)
	}
	if kan.fkCol != "kanton_id" {
		t.Errorf("child fkCol = %q, want kanton_id", kan.fkCol)
	}
	if kan.parentAlias != "j1" {
		t.Errorf("child parentAlias = %q, want j1", kan.parentAlias)
	}
}

func TestBuildJoinNodes_MaxDepthZero(t *testing.T) {
	plz, _, _, idx := plzOrtschaftKantonTypes()
	seq := 0
	nodes := buildJoinNodes(plz, "swiss", idx, "t0", 0, 0, &seq)
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes at maxDepth=0, got %d", len(nodes))
	}
}

func TestBuildJoinNodes_MaxDepthOne(t *testing.T) {
	plz, _, _, idx := plzOrtschaftKantonTypes()
	seq := 0
	nodes := buildJoinNodes(plz, "swiss", idx, "t0", 0, 1, &seq)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node at maxDepth=1, got %d", len(nodes))
	}
	if len(nodes[0].children) != 0 {
		t.Errorf("expected 0 children at maxDepth=1, got %d", len(nodes[0].children))
	}
}

func TestBuildJoinNodes_SkipsListRelation(t *testing.T) {
	td := TypeDef{
		Name: "Kanton",
		Fields: []FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
			{Name: "ortschaften", Type: "Ortschaft", IsRelation: true, IsList: true},
		},
	}
	idx := map[string]TypeDef{"Kanton": td}
	seq := 0
	nodes := buildJoinNodes(td, "swiss", idx, "t0", 0, 5, &seq)
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes for list relation, got %d", len(nodes))
	}
}

func TestBuildJoinNodes_NullableRelation(t *testing.T) {
	td := TypeDef{
		Name: "PLZ",
		Fields: []FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
			{Name: "ortschaft", Type: "Ortschaft", IsRelation: true, NonNull: false},
		},
	}
	ort := TypeDef{
		Name: "Ortschaft",
		Fields: []FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
			{Name: "name", Type: "String", NonNull: true},
		},
	}
	idx := map[string]TypeDef{"PLZ": td, "Ortschaft": ort}
	seq := 0
	nodes := buildJoinNodes(td, "swiss", idx, "t0", 0, 5, &seq)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if !nodes[0].nullable {
		t.Error("expected nullable=true for non-required relation")
	}
}

func TestJoinSelectExprs_TwoHop(t *testing.T) {
	plz, _, _, idx := plzOrtschaftKantonTypes()
	seq := 0
	nodes := buildJoinNodes(plz, "swiss", idx, "t0", 0, 5, &seq)
	exprs := joinSelectExprs(nodes)
	// Ortschaft has: id, name, kanton_id → 3 exprs
	// Kanton has: id, kuerzel → 2 exprs
	if len(exprs) != 5 {
		t.Fatalf("expected 5 select exprs, got %d: %v", len(exprs), exprs)
	}
	if exprs[0] != `j1.id AS "j1__id"` {
		t.Errorf("exprs[0] = %q, want j1.id AS \"j1__id\"", exprs[0])
	}
}

func TestJoinClauses_TwoHop(t *testing.T) {
	plz, _, _, idx := plzOrtschaftKantonTypes()
	seq := 0
	nodes := buildJoinNodes(plz, "swiss", idx, "t0", 0, 5, &seq)
	clauses := joinClauses(nodes)
	if len(clauses) != 2 {
		t.Fatalf("expected 2 join clauses, got %d", len(clauses))
	}
	if !strings.Contains(clauses[0], "LEFT JOIN swiss_ortschaft j1") {
		t.Errorf("clause[0] = %q, want LEFT JOIN swiss_ortschaft j1", clauses[0])
	}
	if !strings.Contains(clauses[0], "j1.id = t0.ortschaft_id") {
		t.Errorf("clause[0] = %q, want ON j1.id = t0.ortschaft_id", clauses[0])
	}
	if !strings.Contains(clauses[1], "LEFT JOIN swiss_kanton j2") {
		t.Errorf("clause[1] = %q, want LEFT JOIN swiss_kanton j2", clauses[1])
	}
	if !strings.Contains(clauses[1], "j2.id = j1.kanton_id") {
		t.Errorf("clause[1] = %q, want ON j2.id = j1.kanton_id", clauses[1])
	}
}

func TestTotalJoinCols(t *testing.T) {
	plz, _, _, idx := plzOrtschaftKantonTypes()
	seq := 0
	nodes := buildJoinNodes(plz, "swiss", idx, "t0", 0, 5, &seq)
	got := totalJoinCols(nodes[0])
	// ortschaft: id, name, kanton_id (3) + kanton: id, kuerzel (2) = 5
	if got != 5 {
		t.Errorf("totalJoinCols = %d, want 5", got)
	}
}

func TestAssembleJoinedRows_TwoHop(t *testing.T) {
	plz, _, _, idx := plzOrtschaftKantonTypes()
	seq := 0
	nodes := buildJoinNodes(plz, "swiss", idx, "t0", 0, 5, &seq)
	parentCols := []string{"id", "plz", "ortschaft_id"}

	vals := []any{
		"plz-1", 8001, "ort-1", // parent cols
		"ort-1", "Zürich", "kan-1", "kan-1", "ZH", // join cols
	}
	row := assembleJoinedRows(vals, parentCols, nodes)
	if row["id"] != "plz-1" {
		t.Errorf("id = %v, want plz-1", row["id"])
	}
	ort, ok := row["ortschaft"].(map[string]any)
	if !ok {
		t.Fatalf("ortschaft: expected map, got %T", row["ortschaft"])
	}
	if ort["name"] != "Zürich" {
		t.Errorf("ortschaft.name = %v, want Zürich", ort["name"])
	}
	kan, ok := ort["kanton"].(map[string]any)
	if !ok {
		t.Fatalf("kanton: expected map, got %T", ort["kanton"])
	}
	if kan["kuerzel"] != "ZH" {
		t.Errorf("kanton.kuerzel = %v, want ZH", kan["kuerzel"])
	}
}

func TestAssembleJoinedRows_NullIntermediate(t *testing.T) {
	plzTD := TypeDef{
		Name: "PLZ",
		Fields: []FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
			{Name: "plz", Type: "Int", NonNull: true},
			{Name: "ortschaft", Type: "Ortschaft", IsRelation: true, NonNull: false},
		},
	}
	ortTD := TypeDef{
		Name: "Ortschaft",
		Fields: []FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
			{Name: "name", Type: "String", NonNull: true},
			{Name: "kanton", Type: "Kanton", IsRelation: true, NonNull: true},
		},
	}
	kantonTD := TypeDef{
		Name: "Kanton",
		Fields: []FieldDef{
			{Name: "id", Type: "ID", NonNull: true},
			{Name: "kuerzel", Type: "String", NonNull: true},
		},
	}
	idx := map[string]TypeDef{"PLZ": plzTD, "Ortschaft": ortTD, "Kanton": kantonTD}
	seq := 0
	nodes := buildJoinNodes(plzTD, "swiss", idx, "t0", 0, 5, &seq)
	parentCols := []string{"id", "plz", "ortschaft_id"}

	// All join columns are nil (no ortschaft)
	vals := []any{
		"plz-1", 9999, nil,
		nil, nil, nil, nil, nil,
	}
	row := assembleJoinedRows(vals, parentCols, nodes)
	if row["ortschaft"] != nil {
		t.Errorf("ortschaft = %v, want nil", row["ortschaft"])
	}
}

func TestSelectionRelationDepth(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  int
	}{
		{"no relation", `{ plz { list { plz } } }`, 0},
		{"one hop", `{ plz { list { plz ortschaft { name } } } }`, 1},
		{"two hops", `{ plz { list { plz ortschaft { name kanton { kuerzel } } } } }`, 2},
		{"six hops", `{ g { list { f { e { d { c { b { a { name } } } } } } } } }`, 6},
		// Named fragment: relation chain inside fragment must count correctly (bypass fix)
		{"named fragment two hops", `{ plz { list { ...frag } } } fragment frag on PLZ { ortschaft { name kanton { kuerzel } } }`, 2},
		// String literal } deflates raw brace count, making depth appear shallower than it is (bypass fix)
		{"string literal } does not deflate depth", `{ entity { list(filter: { code: { eq: "}" } }) { rel { subrel { x } } } } }`, 2},
		// String literal { inflates raw brace count, causing false rejection (false-positive fix)
		{"string literal { does not inflate depth", `{ plz { list(filter: { code: { eq: "{" } }) { plz } } } }`, 0},
		// Inline fragment: must not add a level itself, but fields inside still count (depth-bypass fix)
		{"inline fragment two hops", `{ plz { list { ... on PLZ { ortschaft { name kanton { kuerzel } } } } } }`, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectionRelationDepth(tt.query)
			if got != tt.want {
				t.Errorf("selectionRelationDepth(%q) = %d, want %d", tt.query, got, tt.want)
			}
		})
	}
}

func TestMaxDepthFromEnv(t *testing.T) {
	// Default
	t.Setenv("STRATUM_MAX_DEPTH", "")
	if got := MaxDepthFromEnv(); got != 5 {
		t.Errorf("default MaxDepthFromEnv() = %d, want 5", got)
	}

	// Custom
	t.Setenv("STRATUM_MAX_DEPTH", "3")
	if got := MaxDepthFromEnv(); got != 3 {
		t.Errorf("MaxDepthFromEnv() = %d, want 3", got)
	}

	// Invalid → default
	t.Setenv("STRATUM_MAX_DEPTH", "invalid")
	if got := MaxDepthFromEnv(); got != 5 {
		t.Errorf("MaxDepthFromEnv(invalid) = %d, want 5", got)
	}

	// Zero → default
	t.Setenv("STRATUM_MAX_DEPTH", "0")
	if got := MaxDepthFromEnv(); got != 5 {
		t.Errorf("MaxDepthFromEnv(0) = %d, want 5", got)
	}
}

func TestBuildListQueryWithJoins(t *testing.T) {
	plz, _, _, idx := plzOrtschaftKantonTypes()
	seq := 0
	nodes := buildJoinNodes(plz, "swiss", idx, "t0", 0, 5, &seq)
	rootCols := columnNames(plz)
	query := buildListQueryWithJoins("swiss_plz", rootCols, nodes, nil)

	if !strings.Contains(query, "FROM swiss_plz t0") {
		t.Errorf("query missing FROM clause: %s", query)
	}
	if !strings.Contains(query, "LEFT JOIN swiss_ortschaft j1 ON j1.id = t0.ortschaft_id") {
		t.Errorf("query missing ortschaft join: %s", query)
	}
	if !strings.Contains(query, "LEFT JOIN swiss_kanton j2 ON j2.id = j1.kanton_id") {
		t.Errorf("query missing kanton join: %s", query)
	}
	if !strings.Contains(query, "t0.id") {
		t.Errorf("query missing qualified root columns: %s", query)
	}
}

func TestBuildListQueryWithJoins_WithChildSubqueries(t *testing.T) {
	plz, _, _, idx := plzOrtschaftKantonTypes()
	seq := 0
	nodes := buildJoinNodes(plz, "swiss", idx, "t0", 0, 5, &seq)
	rootCols := columnNames(plz)
	childExprs := []string{
		"(SELECT COALESCE(json_agg(json_build_object('id', _c2.id) ORDER BY _c2.id), '[]'::json) FROM swiss_child _c2 WHERE _c2.plz_id = t0.id) AS children",
	}
	query := buildListQueryWithJoins("swiss_plz", rootCols, nodes, childExprs)

	if !strings.Contains(query, "FROM swiss_plz t0") {
		t.Errorf("query missing FROM clause: %s", query)
	}
	if !strings.Contains(query, "LEFT JOIN swiss_ortschaft j1") {
		t.Errorf("query missing ortschaft join: %s", query)
	}
	if !strings.Contains(query, "t0.id) AS children") {
		t.Errorf("query missing child subquery expression: %s", query)
	}
}

func TestResolveRelation_PreLoaded(t *testing.T) {
	preloaded := map[string]any{"id": "1", "name": "Zürich"}
	resolver := resolveRelation(nil, "tbl", nil, "ortschaft_id", "ortschaft")
	p := graphql.ResolveParams{
		Source: map[string]any{
			"ortschaft_id": "1",
			"ortschaft":    preloaded,
		},
	}
	got, err := resolver(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", got)
	}
	if m["name"] != "Zürich" {
		t.Errorf("name = %v, want Zürich", m["name"])
	}
}

func TestResolveRelation_PreLoadedNil(t *testing.T) {
	resolver := resolveRelation(nil, "tbl", nil, "ortschaft_id", "ortschaft")
	p := graphql.ResolveParams{
		Source: map[string]any{
			"ortschaft_id": nil,
			"ortschaft":    nil,
		},
	}
	got, err := resolver(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

// ── scanListWithJoins tests ─────────────────────────────────────────────────

func simpleJoinNodes() []joinNode {
	return []joinNode{{
		fieldName:   "rel",
		alias:       "j1",
		table:       "t_rel",
		fkCol:       "rel_id",
		parentAlias: "t0",
		cols:        []string{"id", "name"},
	}}
}

func TestScanListWithJoins_ScanError(t *testing.T) {
	scanErr := errors.New("broken scan")
	rows := &stubRows{nextN: 1, scanErr: scanErr}
	_, err := scanListWithJoins(rows, []string{"id"}, []string{"j1__id", "j1__name"}, simpleJoinNodes(), nil, "test_t")
	if err == nil {
		t.Fatal("expected error from scanListWithJoins")
	}
	if !strings.Contains(err.Error(), "scan") {
		t.Errorf("error = %q, want it to mention scan", err)
	}
}

func TestScanListWithJoins_RowsErr(t *testing.T) {
	rowErr := errors.New("rows iteration error")
	rows := &stubRowsWithErr{stubRows: stubRows{nextN: 0}, rowErr: rowErr}
	_, err := scanListWithJoins(rows, []string{"id"}, []string{"j1__id", "j1__name"}, simpleJoinNodes(), nil, "test_t")
	if err == nil {
		t.Fatal("expected error from rows.Err()")
	}
	if !errors.Is(err, rowErr) {
		t.Errorf("error = %v, want %v", err, rowErr)
	}
}

func TestScanListWithJoins_Empty(t *testing.T) {
	rows := &stubRows{nextN: 0}
	result, err := scanListWithJoins(rows, []string{"id"}, []string{"j1__id", "j1__name"}, simpleJoinNodes(), nil, "test_t")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty, got %d", len(result))
	}
}

func TestScanListWithJoins_WithChildren(t *testing.T) {
	nodes := simpleJoinNodes()
	rows := &scanWithValsRows{
		nextN: 1,
		vals: [][]any{
			{"id1", "rel1", "RelName", []any{map[string]any{"id": "c1"}}},
		},
	}
	result, err := scanListWithJoins(rows, []string{"id"}, []string{"j1__id", "j1__name"}, nodes, []string{"children"}, "test_t")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result))
	}
	ch, ok := result[0]["children"].([]map[string]any)
	if !ok {
		t.Fatalf("children: expected []map[string]any, got %T", result[0]["children"])
	}
	if len(ch) != 1 {
		t.Errorf("expected 1 child, got %d", len(ch))
	}
}

func TestScanListWithJoins_ChildrenParseError(t *testing.T) {
	nodes := simpleJoinNodes()
	rows := &scanWithValsRows{
		nextN: 1,
		vals: [][]any{
			{"id1", "rel1", "RelName", 42}, // 42 is not parseable
		},
	}
	_, err := scanListWithJoins(rows, []string{"id"}, []string{"j1__id", "j1__name"}, nodes, []string{"children"}, "test_t")
	if err == nil {
		t.Fatal("expected error for unparseable children")
	}
	if !strings.Contains(err.Error(), "parse children") {
		t.Errorf("error = %q, want it to mention parse children", err)
	}
}

// ── ServeHTTP max_depth test ────────────────────────────────────────────────

func TestGQLHandler_MaxDepthExceeded(t *testing.T) {
	ps := &ParsedSchema{
		Types: []TypeDef{{
			Name: "Location",
			Fields: []FieldDef{
				{Name: "id", Type: "ID", NonNull: true},
				{Name: "name", Type: "String", NonNull: true},
			},
		}},
	}
	scalars := map[string]scalar.Plugin{
		"ID":     idscalar.Plugin{},
		"String": stringscalar.Plugin{},
	}
	h, err := BuildHandler(nil, "test", ps, scalars, nil, nil, 1)
	if err != nil {
		t.Fatalf("BuildHandler: %v", err)
	}
	// depth 2 exceeds maxDepth=1: { location { list { name deep { deeper { x } } } } } = depth 2
	body := `{"query":"{ location { list { name deep { deeper { x } } } } }"}`
	req := httptest.NewRequest("POST", "/graphql/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	errs, ok := resp["errors"]
	if !ok {
		t.Fatal("expected errors in response")
	}
	errList, ok := errs.([]any)
	if !ok || len(errList) == 0 {
		t.Fatal("expected non-empty errors list")
	}
	msg := errList[0].(map[string]any)["message"].(string)
	if !strings.Contains(msg, "exceeds maximum") {
		t.Errorf("error message = %q, want it to mention exceeds maximum", msg)
	}
}

func TestSelectionRelationDepth_ShallowQuery(t *testing.T) {
	if got := selectionRelationDepth("{}"); got != 0 {
		t.Errorf("selectionRelationDepth({}) = %d, want 0", got)
	}
	if got := selectionRelationDepth("{ a }"); got != 0 {
		t.Errorf("selectionRelationDepth({ a }) = %d, want 0", got)
	}
}
