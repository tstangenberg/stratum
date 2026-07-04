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

func TestStore_Upsert_NewSchema(t *testing.T) {
	store := schema.NewStore()
	sch := &schema.Schema{Name: "locations", SDL: `type Location { id: ID! }`}
	store.Upsert("locations", sch)

	got, ok := store.Get("locations")
	if !ok {
		t.Fatal("expected to find schema after Upsert")
	}
	if got.Version != 1 {
		t.Errorf("Version = %d, want 1 for new schema", got.Version)
	}
}

func TestStore_Upsert_ExistingSchema(t *testing.T) {
	store := schema.NewStore()
	first := &schema.Schema{Name: "locations", SDL: `type Location { id: ID! }`}
	store.Upsert("locations", first)

	created := first.CreatedAt
	second := &schema.Schema{Name: "locations", SDL: `type Location { id: ID! name: String! }`}
	store.Upsert("locations", second)

	got, ok := store.Get("locations")
	if !ok {
		t.Fatal("expected to find schema after second Upsert")
	}
	if got.Version != 2 {
		t.Errorf("Version = %d, want 2 after re-upload", got.Version)
	}
	if !got.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt changed on re-upload: got %v, want %v", got.CreatedAt, created)
	}
}

func TestStore_All(t *testing.T) {
	store := schema.NewStore()

	all := store.All()
	if len(all) != 0 {
		t.Fatalf("expected empty store, got %d entries", len(all))
	}

	now := time.Now()
	store.Set("alpha", &schema.Schema{Name: "alpha", Version: 1, CreatedAt: now, UpdatedAt: now})
	store.Set("beta", &schema.Schema{Name: "beta", Version: 2, CreatedAt: now, UpdatedAt: now})

	all = store.All()
	if len(all) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(all))
	}

	names := map[string]bool{}
	for _, s := range all {
		names[s.Name] = true
	}
	if !names["alpha"] || !names["beta"] {
		t.Errorf("expected alpha and beta, got %v", names)
	}
}
