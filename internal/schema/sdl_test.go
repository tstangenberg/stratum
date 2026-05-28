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
