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

package plugin

import (
	"testing"

	"github.com/graphql-go/graphql"
)

type stubQueryModifier struct {
	name     string
	priority int
}

func (s *stubQueryModifier) Name() string  { return s.name }
func (s *stubQueryModifier) Priority() int { return s.priority }
func (s *stubQueryModifier) Arguments(_ graphql.Output) graphql.FieldConfigArgument {
	return nil
}
func (s *stubQueryModifier) ModifyQuery(q string, p []any, _ map[string]any) (string, []any, error) {
	return q, p, nil
}

func queryModifierNames(ms []QueryModifier) []string {
	names := make([]string, len(ms))
	for i, m := range ms {
		names[i] = m.Name()
	}
	return names
}

func resetQueryModifierRegistry(t *testing.T) {
	t.Helper()
	original := queryModifierRegistry
	t.Cleanup(func() { queryModifierRegistry = original })
	queryModifierRegistry = QueryModifierRegistry{}
}

func TestRegisterQueryModifierAndBuild(t *testing.T) {
	resetQueryModifierRegistry(t)
	RegisterQueryModifier(func() QueryModifier { return &stubQueryModifier{name: "x", priority: 0} })
	ms := BuildQueryModifiers()
	if len(ms) != 1 || ms[0].Name() != "x" {
		t.Fatalf("BuildQueryModifiers() = %v, want [x]", queryModifierNames(ms))
	}
}

func TestQueryModifierRegistry_BuildEmpty(t *testing.T) {
	r := QueryModifierRegistry{}
	if ms := r.Build(); len(ms) != 0 {
		t.Fatalf("Build() on empty registry = %v, want empty", ms)
	}
}

func TestQueryModifierRegistry_BuildSkipsNilFactory(t *testing.T) {
	r := QueryModifierRegistry{}
	r.Register(func() QueryModifier { return nil })
	if ms := r.Build(); len(ms) != 0 {
		t.Fatalf("Build() with nil factory result = %v, want empty", ms)
	}
}

func TestQueryModifierRegistry_BuildReturnsSinglePlugin(t *testing.T) {
	r := QueryModifierRegistry{}
	r.Register(func() QueryModifier { return &stubQueryModifier{name: "a", priority: 0} })
	ms := r.Build()
	if len(ms) != 1 || ms[0].Name() != "a" {
		t.Fatalf("Build() = %v, want [a]", queryModifierNames(ms))
	}
}

func TestQueryModifierRegistry_BuildSortsByPriority(t *testing.T) {
	r := QueryModifierRegistry{}
	r.Register(func() QueryModifier { return &stubQueryModifier{name: "third", priority: 300} })
	r.Register(func() QueryModifier { return &stubQueryModifier{name: "first", priority: 100} })
	r.Register(func() QueryModifier { return &stubQueryModifier{name: "second", priority: 200} })
	ms := r.Build()
	names := queryModifierNames(ms)
	if len(ms) != 3 || names[0] != "first" || names[1] != "second" || names[2] != "third" {
		t.Fatalf("Build() order = %v, want [first second third]", names)
	}
}

func TestQueryModifierRegistry_BuildMixedNilAndNonNil(t *testing.T) {
	r := QueryModifierRegistry{}
	r.Register(func() QueryModifier { return nil })
	r.Register(func() QueryModifier { return &stubQueryModifier{name: "b", priority: 200} })
	r.Register(func() QueryModifier { return nil })
	r.Register(func() QueryModifier { return &stubQueryModifier{name: "a", priority: 100} })
	ms := r.Build()
	names := queryModifierNames(ms)
	if len(ms) != 2 || names[0] != "a" || names[1] != "b" {
		t.Fatalf("Build() = %v, want [a b]", names)
	}
}
