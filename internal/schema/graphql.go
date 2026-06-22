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
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/graphql-go/graphql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tstangenberg/stratum/internal/plugin"
	"github.com/tstangenberg/stratum/internal/plugin/scalar"
)

// BuildHandler creates an HTTP handler that serves GraphQL for the given schema.
// It builds a dynamic graphql-go schema with Query (list, get) and Mutation (create) per type.
// Relation fields are resolved by loading the referenced record from the DB.
func BuildHandler(db *pgxpool.Pool, schemaName string, ps *ParsedSchema, scalars map[string]scalar.Plugin, modifiers []plugin.QueryModifier, filters []plugin.FilterPlugin, maxDepth int) (http.Handler, error) {
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
			if f.IsRelation && f.IsList {
				childObj := gqlObjects[f.Type]
				childTbl := tableName(schemaName, f.Type)
				childType := typeIndex[f.Type]
				childCols := columnNames(childType)
				fkCol := reverseFK(childType, t.Name)
				fieldName := f.Name
				obj.AddFieldConfig(fieldName, &graphql.Field{
					Type:    graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(childObj))),
					Resolve: resolveChildren(db, childTbl, childCols, fkCol, fieldName),
				})
				continue
			}
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
					Resolve: resolveRelation(db, relTbl, relCols, fkCol, fieldName),
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
	filtersByScalar := indexFilterPlugins(filters)

	for _, t := range ps.Types {
		obj := gqlObjects[t.Name]
		tbl := tableName(schemaName, t.Name)
		colNames := columnNames(t)

		seq := 0
		joinNodes := buildJoinNodes(t, schemaName, typeIndex, "t0", 0, maxDepth, &seq)
		joinColNames := joinAliasedColNames(joinNodes)

		inputFields := graphql.InputObjectConfigFieldMap{}
		for _, f := range t.Fields {
			if f.Name == "id" {
				inputFields["id"] = &graphql.InputObjectFieldConfig{Type: scalars["ID"].GraphQLType()}
				continue
			}
			if f.IsList {
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

		listArgMap, err := listArgs(modifiers, intType)
		if err != nil {
			return nil, err
		}

		filterInput := buildFilterInput(t, filtersByScalar, scalars)
		if filterInput != nil {
			listArgMap["filter"] = &graphql.ArgumentConfig{Type: filterInput}
		}

		typFields := t.Fields
		childSubqueries, err := buildChildSubqueries(t, schemaName, typeIndex, "t0")
		if err != nil {
			return nil, err
		}

		var childSubExprs []string
		for _, cs := range childSubqueries {
			childSubExprs = append(childSubExprs, cs.sql)
		}

		queryNS := graphql.NewObject(graphql.ObjectConfig{
			Name: t.Name + "Query",
			Fields: graphql.Fields{
				"list": &graphql.Field{
					Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(obj))),
					Args: listArgMap,
					Resolve: func(p graphql.ResolveParams) (any, error) {
						query := buildListQueryWithJoins(tbl, colNames, joinNodes, childSubExprs)
						var params []any
						whereClauses, params, err := applyFilters(p.Args, typFields, filtersByScalar, params)
						if err != nil {
							return nil, err
						}
						if len(whereClauses) > 0 {
							if len(joinNodes) > 0 {
								for i, c := range whereClauses {
									whereClauses[i] = "t0." + c
								}
							}
							query += " WHERE " + strings.Join(whereClauses, " AND ")
						}
						query += " ORDER BY t0.id"
						for _, mod := range modifiers {
							query, params, err = mod.ModifyQuery(query, params, p.Args)
							if err != nil {
								return nil, err
							}
						}
						var childFields []string
						for _, cs := range childSubqueries {
							childFields = append(childFields, cs.fieldName)
						}
						return listRecordsWithJoins(p.Context, db, query, params, colNames, joinColNames, joinNodes, childFields, tbl)
					},
				},
				"get": &graphql.Field{
					Type: obj,
					Args: graphql.FieldConfigArgument{
						"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					},
					Resolve: func(p graphql.ResolveParams) (any, error) {
						rec, err := getRecordWithJoins(p.Context, db, tbl, colNames, joinNodes, joinColNames, p.Args["id"].(string))
						if rec == nil {
							return nil, err
						}
						return rec, err
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
	return &gqlHandler{schema: gqlSchema, maxDepth: maxDepth}, nil
}

// resolveRelation returns a GraphQL resolver that loads the related record by FK.
// If the relation has already been pre-loaded via LEFT JOIN (present in the source
// map under fieldName), the pre-loaded value is returned without a DB round-trip.
func resolveRelation(db *pgxpool.Pool, relTbl string, relCols []string, fkCol string, fieldName string) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (any, error) {
		src, ok := p.Source.(map[string]any)
		if !ok {
			return nil, nil
		}
		if preloaded, exists := src[fieldName]; exists {
			return preloaded, nil
		}
		fkID, ok := src[fkCol].(string)
		if !ok || fkID == "" {
			return nil, nil
		}
		return getRecord(p.Context, db, relTbl, relCols, fkID)
	}
}

type gqlHandler struct {
	schema   graphql.Schema
	maxDepth int
}

func (h *gqlHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var params struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if depth := selectionRelationDepth(params.Query); depth > h.maxDepth {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{
				{"message": fmt.Sprintf("query depth %d exceeds maximum allowed depth %d", depth, h.maxDepth)},
			},
		})
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
// List relation fields are skipped — they have no DB column.
func columnNames(t TypeDef) []string {
	var cols []string
	for _, f := range t.Fields {
		if f.IsList {
			continue
		}
		if f.IsRelation {
			cols = append(cols, fkColumnName(f.Name))
		} else {
			cols = append(cols, f.Name)
		}
	}
	return cols
}

func listArgs(modifiers []plugin.QueryModifier, intType graphql.Output) (graphql.FieldConfigArgument, error) {
	args := graphql.FieldConfigArgument{}
	for _, mod := range modifiers {
		for k, v := range mod.Arguments(intType) {
			if _, exists := args[k]; exists {
				return nil, fmt.Errorf("graphql: query modifier %q declares argument %q already registered by a previous modifier", mod.Name(), k)
			}
			args[k] = v
		}
	}
	return args, nil
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

// rowScanner is the subset of pgx.Row used by scanGet.
type rowScanner interface {
	Scan(dest ...any) error
}

func getRecord(ctx context.Context, db *pgxpool.Pool, tbl string, cols []string, id string) (map[string]any, error) {
	row := db.QueryRow(ctx,
		fmt.Sprintf("SELECT %s FROM %s WHERE id = $1", strings.Join(cols, ", "), tbl),
		id,
	)
	return scanGet(row, cols, tbl, id)
}

func scanGet(row rowScanner, cols []string, tbl string, id string) (map[string]any, error) {
	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := row.Scan(ptrs...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get %s id=%s: %w", tbl, id, err)
	}
	rec := make(map[string]any, len(cols))
	for i, name := range cols {
		rec[name] = vals[i]
	}
	return rec, nil
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
		if f.IsList {
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
		if f.IsList {
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

// listRecordsWithJoins executes a query with LEFT JOINs and assembles nested relation maps.
func listRecordsWithJoins(ctx context.Context, db *pgxpool.Pool, query string, params []any, parentCols []string, joinCols []string, nodes []joinNode, childFields []string, tbl string) ([]map[string]any, error) {
	if len(nodes) == 0 && len(childFields) == 0 {
		return listRecords(ctx, db, query, params, parentCols, tbl)
	}
	if len(nodes) == 0 {
		return listRecordsWithChildren(ctx, db, query, params, parentCols, childFields, tbl)
	}
	rows, err := db.Query(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", tbl, err)
	}
	return scanListWithJoins(rows, parentCols, joinCols, nodes, childFields, tbl)
}

func scanListWithJoins(rows scannable, parentCols []string, joinCols []string, nodes []joinNode, childFields []string, tbl string) ([]map[string]any, error) {
	defer rows.Close()
	totalCols := len(parentCols) + len(joinCols) + len(childFields)
	var result []map[string]any
	for rows.Next() {
		vals := make([]any, totalCols)
		ptrs := make([]any, totalCols)
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("list %s: scan: %w", tbl, err)
		}
		row := assembleJoinedRows(vals[:len(parentCols)+len(joinCols)], parentCols, joinCols, nodes)
		childStart := len(parentCols) + len(joinCols)
		for i, name := range childFields {
			children, err := parseJSONChildren(vals[childStart+i])
			if err != nil {
				return nil, fmt.Errorf("list %s: parse children %q: %w", tbl, name, err)
			}
			row[name] = children
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if result == nil {
		result = []map[string]any{}
	}
	return result, nil
}

// getRecordWithJoins fetches a single record by ID with LEFT JOINs for N:1 relations.
func getRecordWithJoins(ctx context.Context, db *pgxpool.Pool, tbl string, parentCols []string, nodes []joinNode, joinCols []string, id string) (map[string]any, error) {
	if len(nodes) == 0 {
		return getRecord(ctx, db, tbl, parentCols, id)
	}
	rootAlias := "t0"
	selectExprs := qualifiedRootCols(rootAlias, parentCols)
	selectExprs = append(selectExprs, joinSelectExprs(nodes)...)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("SELECT %s FROM %s %s", strings.Join(selectExprs, ", "), tbl, rootAlias))
	for _, clause := range joinClauses(nodes) {
		sb.WriteString(" ")
		sb.WriteString(clause)
	}
	sb.WriteString(fmt.Sprintf(" WHERE %s.id = $1", rootAlias))
	row := db.QueryRow(ctx, sb.String(), id)
	totalCols := len(parentCols) + len(joinCols)
	vals := make([]any, totalCols)
	ptrs := make([]any, totalCols)
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := row.Scan(ptrs...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get %s id=%s: %w", tbl, id, err)
	}
	return assembleJoinedRows(vals, parentCols, joinCols, nodes), nil
}

// buildFilterInput creates the GraphQL filter input type for a domain type.
// For each scalar field, it creates a nested input object with operators from matching filter plugins.
// Returns nil if no filterable fields exist.
func buildFilterInput(t TypeDef, filtersByScalar map[string][]plugin.FilterPlugin, scalars map[string]scalar.Plugin) *graphql.InputObject {
	fields := graphql.InputObjectConfigFieldMap{}
	for _, f := range t.Fields {
		if f.IsRelation {
			continue
		}
		fps, ok := filtersByScalar[f.Type]
		if !ok {
			continue
		}
		operatorFields := graphql.InputObjectConfigFieldMap{}
		for _, fp := range fps {
			for k, v := range fp.Operators() {
				operatorFields[k] = v
			}
		}
		if len(operatorFields) == 0 {
			continue
		}
		fields[f.Name] = &graphql.InputObjectFieldConfig{
			Type: graphql.NewInputObject(graphql.InputObjectConfig{
				Name:   t.Name + "_" + f.Name + "_filter",
				Fields: operatorFields,
			}),
		}
	}
	if len(fields) == 0 {
		return nil
	}
	return graphql.NewInputObject(graphql.InputObjectConfig{
		Name:   t.Name + "Filter",
		Fields: fields,
	})
}

// indexFilterPlugins groups filter plugins by their scalar type for efficient lookup.
func indexFilterPlugins(filters []plugin.FilterPlugin) map[string][]plugin.FilterPlugin {
	m := make(map[string][]plugin.FilterPlugin, len(filters))
	for _, f := range filters {
		m[f.ScalarType()] = append(m[f.ScalarType()], f)
	}
	return m
}

// applyFilters extracts the filter argument from GraphQL args and generates SQL WHERE clauses.
func applyFilters(args map[string]any, fields []FieldDef, filtersByScalar map[string][]plugin.FilterPlugin, params []any) ([]string, []any, error) {
	filterArg, ok := args["filter"].(map[string]any)
	if !ok || filterArg == nil {
		return nil, params, nil
	}
	var clauses []string
	for _, f := range fields {
		if f.IsRelation {
			continue
		}
		fieldFilter, ok := filterArg[f.Name].(map[string]any)
		if !ok {
			continue
		}
		fps := filtersByScalar[f.Type]
		for operator, value := range fieldFilter {
			if value == nil {
				continue
			}
			for _, fp := range fps {
				clause, newParams, err := fp.ToSQL(f.Name, operator, value, len(params)+1)
				if err != nil {
					return nil, nil, fmt.Errorf("schema: apply filter %q.%q: %w", f.Name, operator, err)
				}
				clauses = append(clauses, clause)
				params = append(params, newParams...)
			}
		}
	}
	return clauses, params, nil
}

// reverseFK finds the FK column on childType that references parentTypeName.
func reverseFK(childType TypeDef, parentTypeName string) string {
	for _, f := range childType.Fields {
		if f.IsRelation && !f.IsList && f.Type == parentTypeName {
			return fkColumnName(f.Name)
		}
	}
	return ""
}

// childSubquery holds the SQL subquery expression and the field name for a 1:N list relation.
type childSubquery struct {
	fieldName string
	sql       string
}

// buildChildSubqueries builds correlated subqueries for each 1:N list relation on the type.
func buildChildSubqueries(t TypeDef, schemaName string, typeIndex map[string]TypeDef, parentRef string) ([]childSubquery, error) {
	var subs []childSubquery
	for i, f := range t.Fields {
		if !f.IsRelation || !f.IsList {
			continue
		}
		childType := typeIndex[f.Type]
		childTbl := tableName(schemaName, f.Type)
		childCols := columnNames(childType)
		fkCol := reverseFK(childType, t.Name)
		if fkCol == "" {
			return nil, fmt.Errorf("schema: list relation %q on %q: no reverse FK to %q", f.Name, t.Name, f.Type)
		}
		alias := fmt.Sprintf("_c%d", i)

		var kvParts []string
		for _, cc := range childCols {
			kvParts = append(kvParts, fmt.Sprintf("'%s', %s.%s", cc, alias, cc))
		}
		sub := fmt.Sprintf(
			"(SELECT COALESCE(json_agg(json_build_object(%s) ORDER BY %s.id), '[]'::json) FROM %s %s WHERE %s.%s = %s.id) AS %s",
			strings.Join(kvParts, ", "), alias, childTbl, alias, alias, fkCol, parentRef, f.Name,
		)
		subs = append(subs, childSubquery{fieldName: f.Name, sql: sub})
	}
	return subs, nil
}

// childQuerier is the interface used by resolveChildren to query child records.
type childQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// resolveChildren returns a resolver for a 1:N list relation field.
// If the children are already pre-loaded in the source map (from the list query's json_agg),
// they are returned directly. Otherwise, the children are fetched from the DB.
func resolveChildren(db childQuerier, childTbl string, childCols []string, fkCol string, fieldName string) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (any, error) {
		src, ok := p.Source.(map[string]any)
		if !ok {
			return []map[string]any{}, nil
		}
		if children, ok := src[fieldName]; ok {
			if children == nil {
				return []map[string]any{}, nil
			}
			return children, nil
		}
		parentID, ok := src["id"].(string)
		if !ok || parentID == "" {
			return []map[string]any{}, nil
		}
		return listChildRecords(p.Context, db, childTbl, childCols, fkCol, parentID)
	}
}

func listChildRecords(ctx context.Context, db childQuerier, childTbl string, childCols []string, fkCol string, parentID string) ([]map[string]any, error) {
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1 ORDER BY id",
		strings.Join(childCols, ", "), childTbl, fkCol)
	rows, err := db.Query(ctx, query, parentID)
	if err != nil {
		return nil, fmt.Errorf("list children %s: %w", childTbl, err)
	}
	return scanList(rows, childCols, childTbl)
}

// listRecordsWithChildren executes a query that includes json_agg subqueries for 1:N fields.
func listRecordsWithChildren(ctx context.Context, db *pgxpool.Pool, query string, params []any, parentCols []string, childFields []string, tbl string) ([]map[string]any, error) {
	rows, err := db.Query(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", tbl, err)
	}
	return scanListWithChildren(rows, parentCols, childFields, tbl)
}

func scanListWithChildren(rows scannable, parentCols []string, childFields []string, tbl string) ([]map[string]any, error) {
	defer rows.Close()
	totalCols := len(parentCols) + len(childFields)
	var result []map[string]any
	for rows.Next() {
		vals := make([]any, totalCols)
		ptrs := make([]any, totalCols)
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("list %s: scan: %w", tbl, err)
		}
		row := make(map[string]any, totalCols)
		for i, name := range parentCols {
			row[name] = vals[i]
		}
		for i, name := range childFields {
			children, err := parseJSONChildren(vals[len(parentCols)+i])
			if err != nil {
				return nil, fmt.Errorf("list %s: parse children %q: %w", tbl, name, err)
			}
			row[name] = children
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if result == nil {
		result = []map[string]any{}
	}
	return result, nil
}

// parseJSONChildren converts a json_agg result into a Go slice of maps.
// pgx may return the value as pre-parsed []any or as raw []byte/string.
func parseJSONChildren(raw any) ([]map[string]any, error) {
	switch v := raw.(type) {
	case []any:
		result := make([]map[string]any, 0, len(v))
		for _, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("schema: expected map in children array, got %T", item)
			}
			result = append(result, m)
		}
		return result, nil
	case []byte:
		var result []map[string]any
		if err := json.Unmarshal(v, &result); err != nil {
			return nil, fmt.Errorf("schema: unmarshal children: %w", err)
		}
		if result == nil {
			result = []map[string]any{}
		}
		return result, nil
	case string:
		var result []map[string]any
		if err := json.Unmarshal([]byte(v), &result); err != nil {
			return nil, fmt.Errorf("schema: unmarshal children: %w", err)
		}
		if result == nil {
			result = []map[string]any{}
		}
		return result, nil
	default:
		return nil, fmt.Errorf("schema: unexpected type %T for JSON children column", raw)
	}
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
