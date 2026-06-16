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
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/tstangenberg/stratum/internal/plugin"
)

// Plugin provides the "eq" filter operator for a specific scalar type.
type Plugin struct {
	name       string
	scalarType string
	gqlType    graphql.Output
}

var _ plugin.FilterPlugin = (*Plugin)(nil)

// New creates an eq filter plugin for the given scalar type.
func New(scalarName string, gqlType graphql.Output) *Plugin {
	return &Plugin{
		name:       scalarName + "-eq-filter",
		scalarType: scalarName,
		gqlType:    gqlType,
	}
}

func (p *Plugin) Name() string       { return p.name }
func (p *Plugin) ScalarType() string { return p.scalarType }

func (p *Plugin) Operators(_ graphql.Output) graphql.InputObjectConfigFieldMap {
	return graphql.InputObjectConfigFieldMap{
		"eq": &graphql.InputObjectFieldConfig{Type: p.gqlType},
	}
}

func (p *Plugin) ToSQL(column string, operator string, value any, paramOffset int) (string, []any, error) {
	if operator != "eq" {
		return "", nil, fmt.Errorf("filter: %s: unsupported operator %q", p.name, operator)
	}
	return fmt.Sprintf("%s = $%d", column, paramOffset), []any{value}, nil
}
