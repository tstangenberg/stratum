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

// QueryModifier augments a SQL list query before execution.
// Implementations may declare GraphQL arguments via Arguments (return nil if none)
// and append clauses to the SQL query via ModifyQuery.
type QueryModifier interface {
	// Name returns the plugin identifier.
	Name() string
	// Arguments returns GraphQL argument configs to add to each list field.
	// intType is the graphql-go Int type sourced from the scalar registry.
	// Return nil if this modifier requires no client-supplied arguments.
	Arguments(intType graphql.Output) graphql.FieldConfigArgument
	// ModifyQuery appends clauses to query, extends params, and returns the
	// modified versions, or an error if args are invalid.
	// params contains any existing query parameters; ModifyQuery appends its own
	// starting at the correct 1-based index.
	ModifyQuery(query string, params []any, args map[string]any) (string, []any, error)
}
