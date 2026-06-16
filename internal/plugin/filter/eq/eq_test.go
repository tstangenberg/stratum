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

package eq

import (
	"testing"

	"github.com/graphql-go/graphql"
)

func TestPlugin_Name(t *testing.T) {
	p := New("Int", graphql.Int)
	if got := p.Name(); got != "Int-eq-filter" {
		t.Errorf("Name() = %q, want %q", got, "Int-eq-filter")
	}
}

func TestPlugin_ScalarType(t *testing.T) {
	p := New("String", graphql.String)
	if got := p.ScalarType(); got != "String" {
		t.Errorf("ScalarType() = %q, want %q", got, "String")
	}
}

func TestPlugin_Operators(t *testing.T) {
	p := New("Int", graphql.Int)
	ops := p.Operators()
	if _, ok := ops["eq"]; !ok {
		t.Fatal("expected 'eq' operator in field map")
	}
}

func TestPlugin_ToSQL(t *testing.T) {
	tests := []struct {
		name        string
		column      string
		operator    string
		value       any
		paramOffset int
		wantClause  string
		wantParams  []any
		wantErr     bool
	}{
		{
			name:        "eq_int",
			column:      "plz",
			operator:    "eq",
			value:       8001,
			paramOffset: 1,
			wantClause:  "plz = $1",
			wantParams:  []any{8001},
		},
		{
			name:        "eq_string",
			column:      "name",
			operator:    "eq",
			value:       "Zürich",
			paramOffset: 3,
			wantClause:  "name = $3",
			wantParams:  []any{"Zürich"},
		},
		{
			name:        "unsupported_operator",
			column:      "x",
			operator:    "gte",
			value:       1,
			paramOffset: 1,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New("Int", graphql.Int)
			clause, params, err := p.ToSQL(tt.column, tt.operator, tt.value, tt.paramOffset)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if clause != tt.wantClause {
				t.Errorf("clause = %q, want %q", clause, tt.wantClause)
			}
			if len(params) != len(tt.wantParams) {
				t.Fatalf("params len = %d, want %d", len(params), len(tt.wantParams))
			}
			for i, v := range params {
				if v != tt.wantParams[i] {
					t.Errorf("params[%d] = %v, want %v", i, v, tt.wantParams[i])
				}
			}
		})
	}
}
