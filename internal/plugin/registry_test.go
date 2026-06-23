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
	"net/http"
	"testing"
)

type stubMiddleware struct {
	name     string
	priority int
}

func (s *stubMiddleware) Name() string                        { return s.name }
func (s *stubMiddleware) Priority() int                       { return s.priority }
func (s *stubMiddleware) Wrap(next http.Handler) http.Handler { return next }

func middlewareNames(ms []HTTPMiddleware) []string {
	names := make([]string, len(ms))
	for i, m := range ms {
		names[i] = m.Name()
	}
	return names
}

func resetMiddlewareRegistry(t *testing.T) {
	t.Helper()
	original := middlewareRegistry
	t.Cleanup(func() { middlewareRegistry = original })
	middlewareRegistry = Registry{}
}

func TestRegisterMiddlewareAndBuild(t *testing.T) {
	resetMiddlewareRegistry(t)
	RegisterMiddleware(func() HTTPMiddleware { return &stubMiddleware{name: "x", priority: 0} })
	ms := BuildMiddlewares()
	if len(ms) != 1 || ms[0].Name() != "x" {
		t.Fatalf("BuildMiddlewares() = %v, want [x]", middlewareNames(ms))
	}
}

func TestRegistry_BuildEmpty(t *testing.T) {
	r := Registry{}
	if ms := r.Build(); len(ms) != 0 {
		t.Fatalf("Build() on empty registry = %v, want empty", ms)
	}
}

func TestRegistry_BuildSkipsNilFactory(t *testing.T) {
	r := Registry{}
	r.Register(func() HTTPMiddleware { return nil })
	if ms := r.Build(); len(ms) != 0 {
		t.Fatalf("Build() with nil factory result = %v, want empty", ms)
	}
}

func TestRegistry_BuildReturnsSinglePlugin(t *testing.T) {
	r := Registry{}
	r.Register(func() HTTPMiddleware { return &stubMiddleware{name: "a", priority: 0} })
	ms := r.Build()
	if len(ms) != 1 || ms[0].Name() != "a" {
		t.Fatalf("Build() = %v, want [a]", middlewareNames(ms))
	}
}

func TestRegistry_BuildSortsByPriority(t *testing.T) {
	r := Registry{}
	r.Register(func() HTTPMiddleware { return &stubMiddleware{name: "third", priority: 300} })
	r.Register(func() HTTPMiddleware { return &stubMiddleware{name: "first", priority: 100} })
	r.Register(func() HTTPMiddleware { return &stubMiddleware{name: "second", priority: 200} })
	ms := r.Build()
	names := middlewareNames(ms)
	if len(ms) != 3 || names[0] != "first" || names[1] != "second" || names[2] != "third" {
		t.Fatalf("Build() order = %v, want [first second third]", names)
	}
}

func TestRegistry_BuildMixedNilAndNonNil(t *testing.T) {
	r := Registry{}
	r.Register(func() HTTPMiddleware { return nil })
	r.Register(func() HTTPMiddleware { return &stubMiddleware{name: "b", priority: 200} })
	r.Register(func() HTTPMiddleware { return nil })
	r.Register(func() HTTPMiddleware { return &stubMiddleware{name: "a", priority: 100} })
	ms := r.Build()
	names := middlewareNames(ms)
	if len(ms) != 2 || names[0] != "a" || names[1] != "b" {
		t.Fatalf("Build() = %v, want [a b]", names)
	}
}
