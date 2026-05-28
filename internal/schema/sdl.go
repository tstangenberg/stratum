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

package schema

import (
	"fmt"
	"strings"

	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

// builtinTypes lists GraphQL built-in type names to skip during parsing.
var builtinTypes = map[string]bool{
	"String": true, "Int": true, "Float": true, "Boolean": true, "ID": true,
	"Query": true, "Mutation": true, "Subscription": true,
}

// ParseSDL parses a GraphQL SDL string and extracts user-defined object types.
// Returns an error if the SDL is invalid or defines no object types.
func ParseSDL(sdl string) (*ParsedSchema, error) {
	if strings.TrimSpace(sdl) == "" {
		return nil, fmt.Errorf("schema: sdl is empty")
	}
	src := &ast.Source{Name: "user", Input: sdl}
	gqlSchema, err := gqlparser.LoadSchema(src)
	if err != nil {
		return nil, fmt.Errorf("schema: parse sdl: %w", err)
	}

	var types []TypeDef
	for name, def := range gqlSchema.Types {
		if def.Kind != ast.Object {
			continue
		}
		if strings.HasPrefix(name, "__") {
			continue
		}
		if builtinTypes[name] {
			continue
		}

		td := TypeDef{Name: name}
		for _, f := range def.Fields {
			td.Fields = append(td.Fields, FieldDef{
				Name:    f.Name,
				Type:    f.Type.NamedType,
				NonNull: f.Type.NonNull,
			})
		}
		types = append(types, td)
	}

	if len(types) == 0 {
		return nil, fmt.Errorf("schema: sdl defines no object types")
	}
	return &ParsedSchema{Types: types}, nil
}
