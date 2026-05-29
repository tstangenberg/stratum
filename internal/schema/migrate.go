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
func CreateTable(ctx context.Context, db *pgxpool.Pool, schemaName string, t TypeDef, scalars map[string]scalar.Plugin) error {
	tblName := tableName(schemaName, t.Name)
	cols := []string{"id TEXT PRIMARY KEY"}
	for _, f := range t.Fields {
		if f.Name == "id" {
			continue
		}
		p, ok := scalars[f.Type]
		if !ok {
			return fmt.Errorf("migrate: unknown scalar %q for field %q.%q", f.Type, t.Name, f.Name)
		}
		null := ""
		if f.NonNull {
			null = " NOT NULL"
		}
		cols = append(cols, fmt.Sprintf("%s %s%s", f.Name, p.ColumnType(), null))
	}
	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", tblName, strings.Join(cols, ", "))
	if _, err := db.Exec(ctx, sql); err != nil {
		return fmt.Errorf("migrate: create table %q: %w", tblName, err)
	}
	return nil
}

// tableName returns the PostgreSQL table name for a schema + type combination.
func tableName(schemaName, typeName string) string {
	return schemaName + "_" + strings.ToLower(typeName)
}
