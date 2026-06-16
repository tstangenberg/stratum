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

package plugin

import "github.com/graphql-go/graphql"

// FilterPlugin adds filter operators for a specific scalar type.
// Each implementation provides one or more operators (e.g. "eq") and
// generates SQL WHERE clause fragments via ToSQL.
type FilterPlugin interface {
	// Name returns the plugin identifier (e.g. "int-eq-filter").
	Name() string
	// ScalarType returns the GraphQL scalar type name this plugin applies to (e.g. "Int").
	ScalarType() string
	// Operators returns the GraphQL input field configs keyed by operator name.
	Operators() graphql.InputObjectConfigFieldMap
	// ToSQL generates a SQL WHERE fragment for the given column, operator, and value.
	// paramOffset is the next available $N placeholder index.
	// Returns the clause (e.g. "col = $3"), the parameter values, and any error.
	ToSQL(column string, operator string, value any, paramOffset int) (string, []any, error)
}
