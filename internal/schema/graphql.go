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
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/graphql-go/graphql"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tstangenberg/stratum/internal/plugin/pagination"
	"github.com/tstangenberg/stratum/internal/plugin/scalar"
)

// BuildHandler creates an HTTP handler that serves GraphQL for the given schema.
// It builds a dynamic graphql-go schema with Query (list, get) and Mutation (create) per type.
// Relation fields are resolved by loading the referenced record from the DB.
func BuildHandler(db *pgxpool.Pool, schemaName string, ps *ParsedSchema, scalars map[string]scalar.Plugin, pag pagination.Plugin) (http.Handler, error) {
	intType := graphql.Int
	if s, ok := scalars["Int"]; ok {
		intType = s.GraphQLType()
	}
	typeIndex := make(map[string]TypeDef, len(ps.Types))
	for _, t := range ps.Types {
		typeIndex[t.Name] = t
	}

	gqlObjects := make(map[string]*graphql.Object, len(ps.Types))
	for _, t := range ps.Types {
		gqlObjects[t.Name] = graphql.NewObject(graphql.ObjectConfig{
			Name:   t.Name,
			Fields: graphql.Fields{},
		})
	}

	for _, t := range ps.Types {
		obj := gqlObjects[t.Name]
		for _, f := range t.Fields {
			if f.IsRelation {
				relObj := gqlObjects[f.Type]
				relTbl := tableName(schemaName, f.Type)
				relCols := columnNames(typeIndex[f.Type])
				fkCol := fkColumnName(f.Name)
				fieldName := f.Name
				var relType graphql.Output = relObj
				if f.NonNull {
					relType = graphql.NewNonNull(relObj)
				}
				obj.AddFieldConfig(fieldName, &graphql.Field{
					Type:    relType,
					Resolve: resolveRelation(db, relTbl, relCols, fkCol),
				})
				continue
			}
			ft, err := scalarToGraphQL(f, scalars)
			if err != nil {
				return nil, err
			}
			obj.AddFieldConfig(f.Name, &graphql.Field{Type: ft})
		}
	}

	queryFields := graphql.Fields{}
	mutationFields := graphql.Fields{}

	for _, t := range ps.Types {
		obj := gqlObjects[t.Name]
		tbl := tableName(schemaName, t.Name)
		colNames := columnNames(t)

		inputFields := graphql.InputObjectConfigFieldMap{}
		for _, f := range t.Fields {
			if f.Name == "id" {
				inputFields["id"] = &graphql.InputObjectFieldConfig{Type: scalars["ID"].GraphQLType()}
				continue
			}
			if f.IsRelation {
				inName := fkInputName(f.Name)
				var ft graphql.Input = graphql.ID
				if f.NonNull {
					ft = graphql.NewNonNull(graphql.ID)
				}
				inputFields[inName] = &graphql.InputObjectFieldConfig{Type: ft}
				continue
			}
			ft, _ := scalarToGraphQL(f, scalars)
			inputFields[f.Name] = &graphql.InputObjectFieldConfig{Type: ft}
		}
		inputObj := graphql.NewInputObject(graphql.InputObjectConfig{
			Name:   "Create" + t.Name + "Input",
			Fields: inputFields,
		})

		queryNS := graphql.NewObject(graphql.ObjectConfig{
			Name: t.Name + "Query",
			Fields: graphql.Fields{
				"list": &graphql.Field{
					Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(obj))),
					Args: pag.Arguments(intType),
					Resolve: func(p graphql.ResolveParams) (any, error) {
						base := fmt.Sprintf("SELECT %s FROM %s ORDER BY id", strings.Join(colNames, ", "), tbl)
						query, params, err := pag.ModifyQuery(base, nil, p.Args)
						if err != nil {
							return nil, err
						}
						return listRecords(p.Context, db, query, params, colNames, tbl)
					},
				},
				"get": &graphql.Field{
					Type: obj,
					Args: graphql.FieldConfigArgument{
						"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					},
					Resolve: func(p graphql.ResolveParams) (any, error) {
						return getRecord(p.Context, db, tbl, colNames, p.Args["id"].(string))
					},
				},
			},
		})

		mutationNS := graphql.NewObject(graphql.ObjectConfig{
			Name: t.Name + "Mutation",
			Fields: graphql.Fields{
				"create": &graphql.Field{
					Type: graphql.NewNonNull(obj),
					Args: graphql.FieldConfigArgument{
						"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(inputObj)},
					},
					Resolve: func(p graphql.ResolveParams) (any, error) {
						input := p.Args["input"].(map[string]any)
						return createRecord(p.Context, db, tbl, t.Fields, input)
					},
				},
			},
		})

		ns := strings.ToLower(t.Name)
		empty := func(p graphql.ResolveParams) (any, error) { return map[string]any{}, nil }
		queryFields[ns] = &graphql.Field{
			Type:    graphql.NewNonNull(queryNS),
			Resolve: empty,
		}
		mutationFields[ns] = &graphql.Field{
			Type:    graphql.NewNonNull(mutationNS),
			Resolve: empty,
		}
	}

	gqlSchema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name:   "Query",
			Fields: queryFields,
		}),
		Mutation: graphql.NewObject(graphql.ObjectConfig{
			Name:   "Mutation",
			Fields: mutationFields,
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("graphql: build schema: %w", err)
	}
	return &gqlHandler{schema: gqlSchema}, nil
}

// resolveRelation returns a GraphQL resolver that loads the related record by FK.
func resolveRelation(db *pgxpool.Pool, relTbl string, relCols []string, fkCol string) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (any, error) {
		src, ok := p.Source.(map[string]any)
		if !ok {
			return nil, nil
		}
		fkID, ok := src[fkCol].(string)
		if !ok || fkID == "" {
			return nil, nil
		}
		return getRecord(p.Context, db, relTbl, relCols, fkID)
	}
}

type gqlHandler struct{ schema graphql.Schema }

func (h *gqlHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var params struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	result := graphql.Do(graphql.Params{
		Schema:         h.schema,
		RequestString:  params.Query,
		VariableValues: params.Variables,
		Context:        r.Context(),
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// scalarToGraphQL maps a FieldDef to its graphql-go output type via the scalar plugin registry.
func scalarToGraphQL(f FieldDef, scalars map[string]scalar.Plugin) (graphql.Output, error) {
	p, ok := scalars[f.Type]
	if !ok {
		return nil, fmt.Errorf("graphql: unknown scalar %q for field %q", f.Type, f.Name)
	}
	var base graphql.Output = p.GraphQLType()
	if f.NonNull {
		return graphql.NewNonNull(base), nil
	}
	return base, nil
}

// columnNames returns the actual PostgreSQL column names for a type's fields.
// Relation fields are mapped to their FK column name (e.g. kanton → kanton_id).
func columnNames(t TypeDef) []string {
	var cols []string
	for _, f := range t.Fields {
		if f.IsRelation {
			cols = append(cols, fkColumnName(f.Name))
		} else {
			cols = append(cols, f.Name)
		}
	}
	return cols
}

// scannable is the subset of pgx.Rows used by scanList.
type scannable interface {
	Close()
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func listRecords(ctx context.Context, db *pgxpool.Pool, query string, params []any, cols []string, tbl string) ([]map[string]any, error) {
	rows, err := db.Query(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", tbl, err)
	}
	return scanList(rows, cols, tbl)
}

func scanList(rows scannable, cols []string, tbl string) ([]map[string]any, error) {
	defer rows.Close()
	result := []map[string]any{}
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("list %s: scan: %w", tbl, err)
		}
		row := make(map[string]any, len(cols))
		for i, name := range cols {
			row[name] = vals[i]
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func getRecord(ctx context.Context, db *pgxpool.Pool, tbl string, cols []string, id string) (map[string]any, error) {
	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	err := db.QueryRow(ctx,
		fmt.Sprintf("SELECT %s FROM %s WHERE id = $1", strings.Join(cols, ", "), tbl),
		id,
	).Scan(ptrs...)
	if err != nil {
		return nil, fmt.Errorf("get %s id=%s: %w", tbl, id, err)
	}
	row := make(map[string]any, len(cols))
	for i, name := range cols {
		row[name] = vals[i]
	}
	return row, nil
}

func createRecord(ctx context.Context, db *pgxpool.Pool, tbl string, fields []FieldDef, input map[string]any) (map[string]any, error) {
	// graphql.ID coerces all inputs to string at the GraphQL layer, so a
	// string type assertion is the only case we need to handle here.
	id, ok := input["id"].(string)
	if !ok || id == "" {
		id = newID()
	}
	cols := []string{"id"}
	args := []any{id}
	placeholders := []string{"$1"}
	ph := 2
	for _, f := range fields {
		if f.Name == "id" {
			continue
		}
		if f.IsRelation {
			inName := fkInputName(f.Name)
			val, ok := input[inName]
			if !ok {
				continue
			}
			cols = append(cols, fkColumnName(f.Name))
			args = append(args, val)
			placeholders = append(placeholders, fmt.Sprintf("$%d", ph))
			ph++
			continue
		}
		val, ok := input[f.Name]
		if !ok {
			continue
		}
		cols = append(cols, f.Name)
		args = append(args, val)
		placeholders = append(placeholders, fmt.Sprintf("$%d", ph))
		ph++
	}
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tbl, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
	if _, err := db.Exec(ctx, sql, args...); err != nil {
		return nil, fmt.Errorf("create %s: %w", tbl, err)
	}
	row := map[string]any{"id": id}
	for _, f := range fields {
		if f.Name == "id" {
			continue
		}
		if f.IsRelation {
			inName := fkInputName(f.Name)
			if val, ok := input[inName]; ok {
				row[fkColumnName(f.Name)] = val
			}
			continue
		}
		if val, ok := input[f.Name]; ok {
			row[f.Name] = val
		}
	}
	return row, nil
}

// newID generates a random UUID v4 string without external dependencies.
func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
