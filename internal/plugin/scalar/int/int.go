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

package intscalar

import (
	"math"
	"strconv"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/tstangenberg/stratum/internal/plugin/scalar"
)

var int32Scalar = graphql.NewScalar(graphql.ScalarConfig{
	Name: "Int",
	Description: "The `Int` scalar type represents non-fractional signed whole numeric " +
		"values. Int can represent values between -(2^31) and 2^31 - 1.",
	Serialize:  coerceInt32,
	ParseValue: coerceInt32,
	ParseLiteral: func(valueAST ast.Value) any {
		switch v := valueAST.(type) {
		case *ast.IntValue:
			n, err := strconv.ParseInt(v.Value, 10, 32)
			if err != nil {
				return nil
			}
			return int(n)
		}
		return nil
	},
})

func coerceInt32(value any) any {
	switch v := value.(type) {
	case int:
		if v < math.MinInt32 || v > math.MaxInt32 {
			return nil
		}
		return v
	case int32:
		return int(v)
	case int64:
		if v < math.MinInt32 || v > math.MaxInt32 {
			return nil
		}
		return int(v)
	case float64:
		if v != math.Trunc(v) || v < math.MinInt32 || v > math.MaxInt32 {
			return nil
		}
		return int(v)
	}
	return nil
}

// Plugin implements scalar.Plugin for the GraphQL Int type.
type Plugin struct{}

var _ scalar.Plugin = Plugin{}

func (Plugin) Name() string                 { return "Int" }
func (Plugin) ColumnType() string           { return "INTEGER" }
func (Plugin) GraphQLType() *graphql.Scalar { return int32Scalar }
