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
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tstangenberg/stratum/internal/plugin/scalar"
)

// CreateTable creates a PostgreSQL table for the given TypeDef.
// Table name convention: {schemaName}_{typeName_lowercase}.
// The "id" field is always TEXT PRIMARY KEY regardless of its SDL scalar.
// Relation fields produce a FK column: {snake_field_name}_id TEXT [NOT NULL]
// REFERENCES {schemaName}_{referencedType}(id).
func CreateTable(ctx context.Context, db *pgxpool.Pool, schemaName string, t TypeDef, scalars map[string]scalar.Plugin) error {
	tblName := tableName(schemaName, t.Name)
	cols := []string{"id TEXT PRIMARY KEY"}
	for _, f := range t.Fields {
		if f.Name == "id" || f.IsList {
			continue
		}
		def, err := buildColDef(f, t.Name, schemaName, scalars)
		if err != nil {
			return err
		}
		cols = append(cols, def)
	}
	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", tblName, strings.Join(cols, ", "))
	if _, err := db.Exec(ctx, sql); err != nil {
		return fmt.Errorf("migrate: create table %q: %w", tblName, err)
	}
	return nil
}

// AddColumns issues a single ALTER TABLE ADD COLUMN IF NOT EXISTS for every
// non-id, non-list field in t. Using IF NOT EXISTS makes the call idempotent:
// existing columns are silently skipped, so the same schema can be applied
// repeatedly (e.g. after a server restart) without error.
func AddColumns(ctx context.Context, db *pgxpool.Pool, schemaName string, t TypeDef, scalars map[string]scalar.Plugin) error {
	var clauses []string
	for _, f := range t.Fields {
		if f.Name == "id" || f.IsList {
			continue
		}
		def, err := buildColDef(f, t.Name, schemaName, scalars)
		if err != nil {
			return err
		}
		clauses = append(clauses, "ADD COLUMN IF NOT EXISTS "+def)
	}
	if len(clauses) == 0 {
		return nil
	}
	tblName := tableName(schemaName, t.Name)
	sql := fmt.Sprintf("ALTER TABLE %s %s", tblName, strings.Join(clauses, ", "))
	if _, err := db.Exec(ctx, sql); err != nil {
		return fmt.Errorf("migrate: add columns to %q: %w", tblName, err)
	}
	return nil
}

// buildColDef returns the SQL column definition fragment for a single field.
// This is shared by CreateTable and AddColumns to keep their column handling
// identical (same NOT NULL, same FK format).
func buildColDef(f FieldDef, typeName, schemaName string, scalars map[string]scalar.Plugin) (string, error) {
	null := ""
	if f.NonNull {
		null = " NOT NULL"
	}
	if f.IsRelation {
		col := fkColumnName(f.Name)
		refTbl := tableName(schemaName, f.Type)
		return fmt.Sprintf("%s TEXT%s REFERENCES %s(id)", col, null, refTbl), nil
	}
	p, ok := scalars[f.Type]
	if !ok {
		return "", fmt.Errorf("migrate: unknown scalar %q for field %q.%q", f.Type, typeName, f.Name)
	}
	return fmt.Sprintf("%s %s%s", f.Name, p.ColumnType(), null), nil
}

// tableName returns the PostgreSQL table name for a schema + type combination.
func tableName(schemaName, typeName string) string {
	return schemaName + "_" + strings.ToLower(typeName)
}
