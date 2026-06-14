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

package pagination

import "github.com/graphql-go/graphql"

// Plugin adds pagination arguments to list queries and applies SQL pagination clauses.
type Plugin interface {
	// Name returns the plugin identifier.
	Name() string
	// Arguments returns the GraphQL argument config to add to each list field.
	// intType is the graphql-go output type for Int (sourced from the scalar plugin registry).
	Arguments(intType graphql.Output) graphql.FieldConfigArgument
	// ApplySQL appends pagination clauses to query, extends params, and returns the
	// modified query and params, or an error if args are invalid.
	// params contains any existing query parameters; ApplySQL appends its own starting
	// at the correct 1-based index.
	ApplySQL(query string, params []any, args map[string]any) (string, []any, error)
}
