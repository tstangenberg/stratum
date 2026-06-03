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

package intscalar_test

import (
	"math"
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/tstangenberg/stratum/internal/plugin/scalar"
	intscalar "github.com/tstangenberg/stratum/internal/plugin/scalar/int"
)

func TestIntPlugin(t *testing.T) {
	p := intscalar.Plugin{}

	t.Run("Name", func(t *testing.T) {
		if p.Name() != "Int" {
			t.Errorf("Name() = %q, want %q", p.Name(), "Int")
		}
	})

	t.Run("ColumnType", func(t *testing.T) {
		if p.ColumnType() != "INTEGER" {
			t.Errorf("ColumnType() = %q, want %q", p.ColumnType(), "INTEGER")
		}
	})

	t.Run("GraphQLType", func(t *testing.T) {
		gqlType := p.GraphQLType()
		if gqlType == nil {
			t.Fatal("GraphQLType() returned nil")
		}
		if gqlType.Name() != "Int" {
			t.Errorf("GraphQLType().Name() = %q, want %q", gqlType.Name(), "Int")
		}
	})

	t.Run("implements scalar.Plugin", func(t *testing.T) {
		var _ scalar.Plugin = intscalar.Plugin{}
	})
}

func TestIntScalar_Serialize(t *testing.T) {
	p := intscalar.Plugin{}
	serialize := p.GraphQLType().Serialize

	tests := []struct {
		name  string
		input any
		want  any
	}{
		{"int in range", 42, 42},
		{"int zero", 0, 0},
		{"int negative", -1, -1},
		{"int32 max", int(math.MaxInt32), int(math.MaxInt32)},
		{"int32 min", int(math.MinInt32), int(math.MinInt32)},
		{"int above max on 64-bit", int(math.MaxInt32 + 1), nil},
		{"int32 value", int32(100), 100},
		{"int64 in range", int64(42), 42},
		{"int overflow", int64(math.MaxInt32 + 1), nil},
		{"int underflow", int64(math.MinInt32 - 1), nil},
		{"float64 in range", float64(7), 7},
		{"float64 fractional", float64(3.14), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serialize(tt.input)
			if got != tt.want {
				t.Errorf("Serialize(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIntScalar_ParseLiteral(t *testing.T) {
	p := intscalar.Plugin{}
	parseLiteral := p.GraphQLType().ParseLiteral

	t.Run("valid int literal", func(t *testing.T) {
		v := parseLiteral(&ast.IntValue{Value: "42"})
		if v != 42 {
			t.Errorf("ParseLiteral(42) = %v, want 42", v)
		}
	})

	t.Run("overflow literal", func(t *testing.T) {
		v := parseLiteral(&ast.IntValue{Value: "2147483648"})
		if v != nil {
			t.Errorf("ParseLiteral(2147483648) = %v, want nil", v)
		}
	})

	t.Run("underflow literal", func(t *testing.T) {
		v := parseLiteral(&ast.IntValue{Value: "-2147483649"})
		if v != nil {
			t.Errorf("ParseLiteral(-2147483649) = %v, want nil", v)
		}
	})

	t.Run("non-int literal", func(t *testing.T) {
		v := parseLiteral(&ast.StringValue{Value: "hello"})
		if v != nil {
			t.Errorf("ParseLiteral(string) = %v, want nil", v)
		}
	})

	t.Run("max int32 literal", func(t *testing.T) {
		v := parseLiteral(&ast.IntValue{Value: "2147483647"})
		if v != int(math.MaxInt32) {
			t.Errorf("ParseLiteral(2147483647) = %v, want %d", v, math.MaxInt32)
		}
	})

	t.Run("min int32 literal", func(t *testing.T) {
		v := parseLiteral(&ast.IntValue{Value: "-2147483648"})
		if v != int(math.MinInt32) {
			t.Errorf("ParseLiteral(-2147483648) = %v, want %d", v, math.MinInt32)
		}
	})
}

func TestIntScalar_ParseValue(t *testing.T) {
	p := intscalar.Plugin{}
	parseValue := p.GraphQLType().ParseValue

	tests := []struct {
		name  string
		input any
		want  any
	}{
		{"int in range", 42, 42},
		{"int overflow", int64(math.MaxInt32 + 1), nil},
		{"float64 in range", float64(42), 42},
		{"string", "hello", nil}, // no explicit string case; falls through to default: return nil
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseValue(tt.input)
			if got != tt.want {
				t.Errorf("ParseValue(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIntScalar_IsStrictInt32(t *testing.T) {
	p := intscalar.Plugin{}
	gqlType := p.GraphQLType()
	if gqlType == graphql.Int {
		t.Fatal("GraphQLType() must not return the built-in graphql.Int")
	}
}
