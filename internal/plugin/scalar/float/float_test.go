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

package floatscalar_test

import (
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/tstangenberg/stratum/internal/plugin/scalar"
	floatscalar "github.com/tstangenberg/stratum/internal/plugin/scalar/float"
)

func TestFloatPlugin(t *testing.T) {
	p := floatscalar.Plugin{}

	t.Run("Name", func(t *testing.T) {
		if p.Name() != "Float" {
			t.Errorf("Name() = %q, want %q", p.Name(), "Float")
		}
	})

	t.Run("ColumnType", func(t *testing.T) {
		if p.ColumnType() != "DOUBLE PRECISION" {
			t.Errorf("ColumnType() = %q, want %q", p.ColumnType(), "DOUBLE PRECISION")
		}
	})

	t.Run("GraphQLType", func(t *testing.T) {
		gqlType := p.GraphQLType()
		if gqlType != graphql.Float {
			t.Errorf("GraphQLType() = %v, want graphql.Float", gqlType)
		}
	})

	t.Run("implements scalar.Plugin", func(t *testing.T) {
		var _ scalar.Plugin = floatscalar.Plugin{}
	})
}
