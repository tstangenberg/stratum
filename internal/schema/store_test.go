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
package schema_test

import (
	"testing"
	"time"

	"github.com/tstangenberg/stratum/internal/schema"
)

func TestStore_SetAndGet(t *testing.T) {
	store := schema.NewStore()
	s := &schema.Schema{
		Name:      "locations",
		SDL:       `type Location { id: ID! name: String! }`,
		Version:   1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store.Set("locations", s)

	got, ok := store.Get("locations")
	if !ok {
		t.Fatal("expected to find schema 'locations'")
	}
	if got.Name != "locations" {
		t.Errorf("name = %q, want %q", got.Name, "locations")
	}
}

func TestStore_GetMissing(t *testing.T) {
	store := schema.NewStore()
	_, ok := store.Get("nonexistent")
	if ok {
		t.Fatal("expected not found for missing schema")
	}
}
