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
	"testing"

	"github.com/tstangenberg/stratum/internal/schema"
)

func findField(fields []schema.FieldDef, name string) (schema.FieldDef, bool) {
	for _, f := range fields {
		if f.Name == name {
			return f, true
		}
	}
	return schema.FieldDef{}, false
}

func TestParseSDL_SingleStringField(t *testing.T) {
	sdl := `type Location { id: ID! name: String! }`
	ps, err := schema.ParseSDL(sdl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ps.Types) != 1 {
		t.Fatalf("expected 1 type, got %d", len(ps.Types))
	}
	loc := ps.Types[0]
	if loc.Name != "Location" {
		t.Errorf("type name = %q, want %q", loc.Name, "Location")
	}
	if len(loc.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(loc.Fields))
	}

	id, ok := findField(loc.Fields, "id")
	if !ok {
		t.Fatal("field 'id' not found")
	}
	if id.Type != "ID" {
		t.Errorf("field id.Type = %q, want %q", id.Type, "ID")
	}
	if !id.NonNull {
		t.Error("field id.NonNull = false, want true")
	}

	name, ok := findField(loc.Fields, "name")
	if !ok {
		t.Fatal("field 'name' not found")
	}
	if name.Type != "String" {
		t.Errorf("field name.Type = %q, want %q", name.Type, "String")
	}
	if !name.NonNull {
		t.Error("field name.NonNull = false, want true")
	}
}

func TestParseSDL_EmptySDL(t *testing.T) {
	_, err := schema.ParseSDL("")
	if err == nil {
		t.Fatal("expected error for empty SDL")
	}
}

func TestParseSDL_InvalidSDL(t *testing.T) {
	_, err := schema.ParseSDL(`type { broken`)
	if err == nil {
		t.Fatal("expected error for invalid SDL")
	}
}

func TestParseSDL_NoObjectTypes(t *testing.T) {
	// SDL valid but only defines the built-in Query type — filtered out, yielding no user types
	_, err := schema.ParseSDL(`type Query { id: ID }`)
	if err == nil {
		t.Fatal("expected error when SDL has no non-builtin object types")
	}
}

func TestParseSDL_RelationField(t *testing.T) {
	sdl := `
		type Kanton { id: ID! name: String! }
		type Ortschaft { id: ID! name: String! kanton: Kanton! }
	`
	ps, err := schema.ParseSDL(sdl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ps.Types) != 2 {
		t.Fatalf("expected 2 types, got %d", len(ps.Types))
	}

	var ort schema.TypeDef
	for _, td := range ps.Types {
		if td.Name == "Ortschaft" {
			ort = td
			break
		}
	}
	if ort.Name == "" {
		t.Fatal("type Ortschaft not found")
	}

	kf, ok := findField(ort.Fields, "kanton")
	if !ok {
		t.Fatal("field 'kanton' not found in Ortschaft")
	}
	if !kf.IsRelation {
		t.Error("field kanton.IsRelation = false, want true")
	}
	if kf.Type != "Kanton" {
		t.Errorf("field kanton.Type = %q, want %q", kf.Type, "Kanton")
	}
	if !kf.NonNull {
		t.Error("field kanton.NonNull = false, want true")
	}
}

func TestParseSDL_TopologicalOrder(t *testing.T) {
	sdl := `
		type PLZ { id: ID! code: String! ortschaft: Ortschaft! }
		type Kanton { id: ID! name: String! }
		type Ortschaft { id: ID! name: String! kanton: Kanton! }
	`
	ps, err := schema.ParseSDL(sdl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ps.Types) != 3 {
		t.Fatalf("expected 3 types, got %d", len(ps.Types))
	}

	// Kanton must come before Ortschaft, Ortschaft before PLZ
	indexOf := func(name string) int {
		for i, td := range ps.Types {
			if td.Name == name {
				return i
			}
		}
		return -1
	}
	if indexOf("Kanton") > indexOf("Ortschaft") {
		t.Error("Kanton should appear before Ortschaft")
	}
	if indexOf("Ortschaft") > indexOf("PLZ") {
		t.Error("Ortschaft should appear before PLZ")
	}
}

func TestParseSDL_CircularRelation(t *testing.T) {
	sdl := `
		type A { id: ID! b: B! }
		type B { id: ID! a: A! }
	`
	_, err := schema.ParseSDL(sdl)
	if err == nil {
		t.Fatal("expected error for circular relation")
	}
}

func TestParseSDL_ListRelation(t *testing.T) {
	sdl := `
		type Kanton { id: ID! kuerzel: String! ortschaften: [Ortschaft!] }
		type Ortschaft { id: ID! name: String! kanton: Kanton! }
	`
	ps, err := schema.ParseSDL(sdl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ps.Types) != 2 {
		t.Fatalf("expected 2 types, got %d", len(ps.Types))
	}

	var kanton schema.TypeDef
	for _, td := range ps.Types {
		if td.Name == "Kanton" {
			kanton = td
			break
		}
	}
	if kanton.Name == "" {
		t.Fatal("type Kanton not found")
	}

	ortF, ok := findField(kanton.Fields, "ortschaften")
	if !ok {
		t.Fatal("field 'ortschaften' not found in Kanton")
	}
	if !ortF.IsRelation {
		t.Error("field ortschaften.IsRelation = false, want true")
	}
	if !ortF.IsList {
		t.Error("field ortschaften.IsList = false, want true")
	}
	if ortF.Type != "Ortschaft" {
		t.Errorf("field ortschaften.Type = %q, want %q", ortF.Type, "Ortschaft")
	}
}

func TestParseSDL_ListRelationNotCircular(t *testing.T) {
	sdl := `
		type Kanton { id: ID! ortschaften: [Ortschaft!] }
		type Ortschaft { id: ID! kanton: Kanton! }
	`
	_, err := schema.ParseSDL(sdl)
	if err != nil {
		t.Fatalf("list+N:1 should not be circular, got: %v", err)
	}
}

func TestParseSDL_ScalarFieldNotRelation(t *testing.T) {
	sdl := `type Location { id: ID! name: String! }`
	ps, err := schema.ParseSDL(sdl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, f := range ps.Types[0].Fields {
		if f.IsRelation {
			t.Errorf("field %q.IsRelation = true, want false", f.Name)
		}
	}
}
