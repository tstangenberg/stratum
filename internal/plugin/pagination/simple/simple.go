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

package simple

import (
	"fmt"
	"os"
	"strconv"

	"github.com/graphql-go/graphql"
)

const (
	defaultLimit = 100
	defaultMax   = 1000
)

// Plugin provides simple offset-based pagination via limit and offset GraphQL arguments.
// Configure via:
//
//	STRATUM_PLUGINS_PAGINATION_DEFAULT_LIMIT — default records per page (default: 100)
//	STRATUM_PLUGINS_PAGINATION_MAX_LIMIT     — hard maximum records per page (default: 1000)
type Plugin struct {
	defaultLimit int
	maxLimit     int
}

// New creates a Plugin reading its configuration from environment variables.
func New() *Plugin {
	p := &Plugin{defaultLimit: defaultLimit, maxLimit: defaultMax}
	if s := os.Getenv("STRATUM_PLUGINS_PAGINATION_DEFAULT_LIMIT"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			p.defaultLimit = n
		}
	}
	if s := os.Getenv("STRATUM_PLUGINS_PAGINATION_MAX_LIMIT"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			p.maxLimit = n
		}
	}
	return p
}

func (p *Plugin) Name() string { return "pagination" }

// Arguments returns the limit and offset GraphQL argument configs.
func (p *Plugin) Arguments(intType graphql.Output) graphql.FieldConfigArgument {
	return graphql.FieldConfigArgument{
		"limit":  &graphql.ArgumentConfig{Type: intType},
		"offset": &graphql.ArgumentConfig{Type: intType},
	}
}

// ApplySQL appends LIMIT/OFFSET clauses to query using parameterised placeholders.
func (p *Plugin) ApplySQL(query string, params []any, args map[string]any) (string, []any, error) {
	limit := p.defaultLimit
	if limit > p.maxLimit {
		limit = p.maxLimit
	}
	if v, ok := args["limit"].(int); ok {
		if v > p.maxLimit {
			return "", nil, fmt.Errorf("limit %d exceeds maximum %d", v, p.maxLimit)
		}
		if v < 0 {
			v = 0
		}
		limit = v
	}
	offset := 0
	if v, ok := args["offset"].(int); ok && v > 0 {
		offset = v
	}
	n := len(params)
	return fmt.Sprintf("%s LIMIT $%d OFFSET $%d", query, n+1, n+2), append(params, limit, offset), nil
}
