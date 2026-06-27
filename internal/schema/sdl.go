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
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func isBuiltinType(name string) bool {
	switch name {
	case "String", "Int", "Float", "Boolean", "ID",
		"Query", "Mutation", "Subscription":
		return true
	}
	return false
}

// ParseSDL parses a GraphQL SDL string and extracts user-defined object types.
// Fields whose type refers to another object type in the same schema are marked
// as relations (IsRelation = true). The returned types are topologically sorted
// so that referenced types appear before the types that reference them.
// Returns an error if the SDL is invalid or defines no object types.
func ParseSDL(sdl string) (*ParsedSchema, error) {
	if strings.TrimSpace(sdl) == "" {
		return nil, &ValidationError{Msg: "schema: sdl is empty"}
	}
	src := &ast.Source{Name: "user", Input: sdl}
	gqlSchema, err := gqlparser.LoadSchema(src)
	if err != nil {
		return nil, toValidationError(err)
	}

	userTypes := make(map[string]bool)
	for name, def := range gqlSchema.Types {
		if def.Kind != ast.Object || strings.HasPrefix(name, "__") || isBuiltinType(name) {
			continue
		}
		userTypes[name] = true
	}

	byName := make(map[string]TypeDef, len(userTypes))
	for name, def := range gqlSchema.Types {
		if !userTypes[name] {
			continue
		}
		td := TypeDef{Name: name}
		for _, f := range def.Fields {
			var fd FieldDef
			fd.Name = f.Name
			if f.Type.Elem != nil {
				fd.Type = f.Type.Elem.NamedType
				fd.NonNull = f.Type.Elem.NonNull
				fd.IsList = true
			} else {
				fd.Type = f.Type.NamedType
				fd.NonNull = f.Type.NonNull
			}
			if userTypes[fd.Type] {
				fd.IsRelation = true
			}
			td.Fields = append(td.Fields, fd)
		}
		byName[name] = td
	}

	if len(byName) == 0 {
		return nil, &ValidationError{Msg: "schema: sdl defines no object types"}
	}

	sorted, err := topoSort(byName)
	if err != nil {
		return nil, &ValidationError{Msg: err.Error(), cause: err}
	}
	return &ParsedSchema{Types: sorted}, nil
}

// topoSort returns types ordered so that referenced types come before the
// types that reference them. Returns an error on circular references.
func topoSort(byName map[string]TypeDef) ([]TypeDef, error) {
	const (
		unvisited = 0
		visiting  = 1
		visited   = 2
	)
	state := make(map[string]int, len(byName))
	var order []TypeDef

	var visit func(name string) error
	visit = func(name string) error {
		switch state[name] {
		case visited:
			return nil
		case visiting:
			return fmt.Errorf("schema: circular relation involving %q", name)
		}
		state[name] = visiting
		td := byName[name]
		for _, f := range td.Fields {
			if f.IsRelation && !f.IsList {
				if err := visit(f.Type); err != nil {
					return err
				}
			}
		}
		state[name] = visited
		order = append(order, td)
		return nil
	}

	names := sortedKeys(byName)
	for _, name := range names {
		if err := visit(name); err != nil {
			return nil, err
		}
	}
	return order, nil
}

// sortedKeys returns the keys of a map sorted lexicographically for deterministic output.
func sortedKeys(m map[string]TypeDef) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func toValidationError(err error) *ValidationError {
	if list, ok := err.(gqlerror.List); ok {
		var details []ValidationDetail
		for _, e := range list {
			for _, loc := range e.Locations {
				details = append(details, ValidationDetail{
					Line:    loc.Line,
					Column:  loc.Column,
					Message: e.Message,
				})
			}
			if len(e.Locations) == 0 {
				details = append(details, ValidationDetail{Message: e.Message})
			}
		}
		return &ValidationError{Msg: "schema: parse sdl", Details: details, cause: err}
	}
	var gqlErr *gqlerror.Error
	if errors.As(err, &gqlErr) {
		var details []ValidationDetail
		for _, loc := range gqlErr.Locations {
			details = append(details, ValidationDetail{
				Line:    loc.Line,
				Column:  loc.Column,
				Message: gqlErr.Message,
			})
		}
		return &ValidationError{Msg: "schema: parse sdl", Details: details, cause: gqlErr}
	}
	return &ValidationError{Msg: "schema: parse sdl: " + err.Error(), cause: err}
}
