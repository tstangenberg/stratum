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

// SPDX-License-Identifier: AGPL-3.0-or-later
package schema

import (
	"net/http"
	"time"
)

// ParsedSchema is the result of parsing a GraphQL SDL string.
type ParsedSchema struct {
	Types []TypeDef
}

// TypeDef represents a single GraphQL object type.
type TypeDef struct {
	Name   string
	Fields []FieldDef
}

// FieldDef represents a single field within an object type.
type FieldDef struct {
	Name    string
	Type    string // SDL scalar name: "String", "ID", "Int", etc.
	NonNull bool
}

// Schema is a stored, live schema with its metadata and active GraphQL handler.
type Schema struct {
	Name      string
	SDL       string
	Parsed    *ParsedSchema
	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
	Handler   http.Handler
}
