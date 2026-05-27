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
package stringscalar_test

import (
	"testing"

	"github.com/graphql-go/graphql"
	stringscalar "github.com/tstangenberg/stratum/internal/plugin/scalar/string"
)

func TestStringPlugin(t *testing.T) {
	p := stringscalar.Plugin{}
	if p.Name() != "String" {
		t.Errorf("Name() = %q, want %q", p.Name(), "String")
	}
	if p.ColumnType() != "TEXT" {
		t.Errorf("ColumnType() = %q, want %q", p.ColumnType(), "TEXT")
	}
	if p.GraphQLType() != graphql.String {
		t.Errorf("GraphQLType() = %v, want graphql.String", p.GraphQLType())
	}
}
