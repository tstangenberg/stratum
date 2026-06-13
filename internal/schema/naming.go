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
	"strings"
	"unicode"
)

// camelToSnake converts a camelCase string to snake_case.
// Consecutive uppercase runs are treated as a single word (e.g. "PLZ" → "plz").
func camelToSnake(s string) string {
	runes := []rune(s)
	var b strings.Builder
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 && !unicode.IsUpper(runes[i-1]) {
				b.WriteByte('_')
			}
			if i > 0 && unicode.IsUpper(runes[i-1]) && i+1 < len(runes) && !unicode.IsUpper(runes[i+1]) {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// fkColumnName returns the PostgreSQL column name for a relation field.
// Per ADR-1009: field name (camelCase → snake_case) + "_id".
func fkColumnName(fieldName string) string {
	return camelToSnake(fieldName) + "_id"
}

// fkInputName returns the GraphQL input field name for a relation.
// Convention: field name + "Id" (camelCase).
func fkInputName(fieldName string) string {
	return fieldName + "Id"
}
