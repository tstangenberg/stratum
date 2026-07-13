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

import "sync"

// Store is a thread-safe in-memory registry of live schemas.
type Store struct {
	mu      sync.RWMutex
	schemas map[string]*Schema
}

// NewStore creates an empty Store.
func NewStore() *Store {
	return &Store{schemas: make(map[string]*Schema)}
}

// Set stores or replaces a schema by name.
func (s *Store) Set(name string, schema *Schema) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.schemas[name] = schema
}

// SetIfNewer stores schema unless the current value has a higher version.
func (s *Store) SetIfNewer(name string, schema *Schema) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.schemas[name]; ok && existing.Version > schema.Version {
		return
	}
	s.schemas[name] = schema
}

// Upsert stores sch under name, atomically assigning its Version and CreatedAt
// under the write lock. If a schema with the same name already exists, Version
// is incremented from the existing value and CreatedAt is preserved; otherwise
// Version is set to 1. This prevents concurrent re-uploads from losing a
// version increment (TOCTOU).
func (s *Store) Upsert(name string, sch *Schema) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.schemas[name]; ok {
		sch.Version = existing.Version + 1
		sch.CreatedAt = existing.CreatedAt
	} else {
		sch.Version = 1
	}
	s.schemas[name] = sch
}

// Get retrieves a schema by name. Returns (nil, false) if not found.
func (s *Store) Get(name string) (*Schema, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sc, ok := s.schemas[name]
	return sc, ok
}

// All returns a snapshot of every stored schema.
func (s *Store) All() []*Schema {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Schema, 0, len(s.schemas))
	for _, sc := range s.schemas {
		out = append(out, sc)
	}
	return out
}
