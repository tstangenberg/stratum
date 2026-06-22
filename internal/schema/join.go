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
	"os"
	"strconv"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

// defaultMaxDepth is the maximum N:1 relation depth when no override is set.
const defaultMaxDepth = 5

// joinNode describes a single LEFT JOIN hop for an N:1 relation.
type joinNode struct {
	fieldName   string     // GraphQL field name (e.g. "ortschaft")
	alias       string     // SQL alias (e.g. "j1")
	table       string     // SQL table (e.g. "swiss_ortschaft")
	fkCol       string     // FK column on parent alias (e.g. "ortschaft_id")
	parentAlias string     // alias of the parent table (e.g. "t0")
	cols        []string   // columns to SELECT from this table
	nullable    bool       // true when the FK is nullable
	children    []joinNode // nested N:1 hops
}

// buildJoinNodes recursively builds LEFT JOIN nodes for all N:1 relations on td,
// up to maxDepth hops. seq is a counter for generating unique aliases.
func buildJoinNodes(td TypeDef, schemaName string, typeIndex map[string]TypeDef, parentAlias string, depth, maxDepth int, seq *int) []joinNode {
	if depth >= maxDepth {
		return nil
	}
	var nodes []joinNode
	for _, f := range td.Fields {
		if !f.IsRelation || f.IsList {
			continue
		}
		*seq++
		alias := fmt.Sprintf("j%d", *seq)
		relType := typeIndex[f.Type]
		node := joinNode{
			fieldName:   f.Name,
			alias:       alias,
			table:       tableName(schemaName, f.Type),
			fkCol:       fkColumnName(f.Name),
			parentAlias: parentAlias,
			cols:        columnNames(relType),
			nullable:    !f.NonNull,
			children:    buildJoinNodes(relType, schemaName, typeIndex, alias, depth+1, maxDepth, seq),
		}
		nodes = append(nodes, node)
	}
	return nodes
}

// joinSelectExprs returns the SELECT column expressions for all join nodes,
// using alias-qualified names with double-underscore separators for uniqueness.
// For example: j1.id AS "j1__id", j1.name AS "j1__name".
func joinSelectExprs(nodes []joinNode) []string {
	var exprs []string
	for _, n := range nodes {
		for _, col := range n.cols {
			exprs = append(exprs, fmt.Sprintf("%s.%s AS \"%s__%s\"", n.alias, col, n.alias, col))
		}
		exprs = append(exprs, joinSelectExprs(n.children)...)
	}
	return exprs
}

// joinClauses returns the LEFT JOIN SQL clauses for all join nodes.
func joinClauses(nodes []joinNode) []string {
	var clauses []string
	for _, n := range nodes {
		clauses = append(clauses, fmt.Sprintf("LEFT JOIN %s %s ON %s.id = %s.%s",
			n.table, n.alias, n.alias, n.parentAlias, n.fkCol))
		clauses = append(clauses, joinClauses(n.children)...)
	}
	return clauses
}

// joinAliasedColNames returns a flat list of the aliased column names (e.g. "j1__id")
// in the same order as joinSelectExprs.
func joinAliasedColNames(nodes []joinNode) []string {
	var names []string
	for _, n := range nodes {
		for _, col := range n.cols {
			names = append(names, fmt.Sprintf("%s__%s", n.alias, col))
		}
		names = append(names, joinAliasedColNames(n.children)...)
	}
	return names
}

// assembleJoinedRows converts flat scanned values into nested maps, populating
// relation fields from the joined columns. parentCols are the root table columns,
// joinCols are the aliased join column names, and nodes describe the join structure.
func assembleJoinedRows(vals []any, parentCols []string, joinCols []string, nodes []joinNode) map[string]any {
	row := make(map[string]any, len(parentCols))
	for i, name := range parentCols {
		row[name] = vals[i]
	}
	joinVals := vals[len(parentCols):]
	assembleNested(row, joinVals, nodes)
	return row
}

// assembleNested populates nested relation maps from flat joined values.
func assembleNested(parent map[string]any, joinVals []any, nodes []joinNode) {
	offset := 0
	for _, n := range nodes {
		colCount := totalJoinCols(n)
		// Use the "id" column as the null sentinel for absent LEFT JOIN rows.
		// Find it by name rather than assuming position 0, so field definition
		// order in the SDL does not affect correctness.
		idIdx := 0
		for i, c := range n.cols {
			if c == "id" {
				idIdx = i
				break
			}
		}
		idVal := joinVals[offset+idIdx]
		if idVal == nil {
			parent[n.fieldName] = nil
			offset += colCount
			continue
		}
		nested := make(map[string]any, len(n.cols))
		for i, col := range n.cols {
			nested[col] = joinVals[offset+i]
		}
		assembleNested(nested, joinVals[offset+len(n.cols):offset+colCount], n.children)
		parent[n.fieldName] = nested
		offset += colCount
	}
}

// totalJoinCols returns the total number of join columns for a node and all its children.
func totalJoinCols(n joinNode) int {
	count := len(n.cols)
	for _, c := range n.children {
		count += totalJoinCols(c)
	}
	return count
}

// MaxDepthFromEnv reads the max relation depth from the STRATUM_MAX_DEPTH environment variable.
func MaxDepthFromEnv() int {
	s := os.Getenv("STRATUM_MAX_DEPTH")
	if s == "" {
		return defaultMaxDepth
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return defaultMaxDepth
	}
	return v
}

// selectionRelationDepth returns the maximum field-selection nesting depth of a
// GraphQL query minus the three fixed Stratum wrapper levels
// (root operation → type namespace → list/get field).
//
// It parses the query with gqlparser so that string literals, named fragments,
// and aliases are all handled correctly. Raw brace counting is not used because
// it misreads } inside string argument values (depth bypass) and { inside
// string argument values (false rejection).
//
// Named fragments are resolved inline: a spread's depth equals the depth of
// the fragment body at the point of use. InlineFragment / FragmentSpread
// wrappers do not add a level themselves — only Field nodes do.
//
// Returns 0 for any query that cannot be parsed (graphql-go will surface the
// error independently).
func selectionRelationDepth(query string) int {
	doc, err := parser.ParseQuery(&ast.Source{Input: query})
	if err != nil || doc == nil {
		return 0
	}
	frags := make(map[string]ast.SelectionSet, len(doc.Fragments))
	for _, f := range doc.Fragments {
		frags[f.Name] = f.SelectionSet
	}
	max := 0
	for _, op := range doc.Operations {
		if d := selSetDepth(op.SelectionSet, frags); d > max {
			max = d
		}
	}
	if max < 3 {
		return 0
	}
	return max - 3
}

// selSetDepth returns the maximum depth of nested Field selections.
// Only *ast.Field nodes add a level; InlineFragment and FragmentSpread are
// expanded inline at the current level.
func selSetDepth(sel ast.SelectionSet, frags map[string]ast.SelectionSet) int {
	max := 0
	for _, s := range sel {
		var d int
		switch v := s.(type) {
		case *ast.Field:
			d = selSetDepth(v.SelectionSet, frags) + 1
		case *ast.InlineFragment:
			d = selSetDepth(v.SelectionSet, frags)
		case *ast.FragmentSpread:
			d = selSetDepth(frags[v.Name], frags)
		}
		if d > max {
			max = d
		}
	}
	return max
}

// qualifiedRootCols returns column expressions qualified with the root alias.
func qualifiedRootCols(rootAlias string, cols []string) []string {
	q := make([]string, len(cols))
	for i, c := range cols {
		q[i] = rootAlias + "." + c
	}
	return q
}

// buildListQueryWithJoins constructs a SELECT query with LEFT JOINs for N:1 relations.
func buildListQueryWithJoins(tbl string, rootCols []string, nodes []joinNode, childSubqueryExprs []string) string {
	rootAlias := "t0"
	selectExprs := qualifiedRootCols(rootAlias, rootCols)
	selectExprs = append(selectExprs, joinSelectExprs(nodes)...)
	selectExprs = append(selectExprs, childSubqueryExprs...)

	var sb strings.Builder
	sb.WriteString("SELECT ")
	sb.WriteString(strings.Join(selectExprs, ", "))
	sb.WriteString(" FROM ")
	sb.WriteString(tbl)
	sb.WriteString(" ")
	sb.WriteString(rootAlias)
	for _, clause := range joinClauses(nodes) {
		sb.WriteString(" ")
		sb.WriteString(clause)
	}
	return sb.String()
}
