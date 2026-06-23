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
	"context"
	"testing"
)

type stubHealthPlugin struct {
	name string
}

func (s *stubHealthPlugin) Name() string { return s.name }
func (s *stubHealthPlugin) Check(_ context.Context) HealthStatus {
	return HealthStatus{Status: StatusOK}
}

func healthPluginNames(ps []HealthPlugin) []string {
	names := make([]string, len(ps))
	for i, p := range ps {
		names[i] = p.Name()
	}
	return names
}

func resetHealthRegistry(t *testing.T) {
	t.Helper()
	original := healthRegistry
	t.Cleanup(func() { healthRegistry = original })
	healthRegistry = HealthRegistry{}
}

func TestRegisterHealthPluginAndBuild(t *testing.T) {
	resetHealthRegistry(t)
	RegisterHealthPlugin(func() HealthPlugin { return &stubHealthPlugin{name: "db"} })
	ps := BuildHealthPlugins()
	if len(ps) != 1 || ps[0].Name() != "db" {
		t.Fatalf("BuildHealthPlugins() = %v, want [db]", healthPluginNames(ps))
	}
}

func TestHealthRegistry_BuildEmpty(t *testing.T) {
	r := HealthRegistry{}
	if ps := r.Build(); len(ps) != 0 {
		t.Fatalf("Build() on empty registry = %v, want empty", ps)
	}
}

func TestHealthRegistry_BuildSkipsNilFactory(t *testing.T) {
	r := HealthRegistry{}
	r.Register(func() HealthPlugin { return nil })
	if ps := r.Build(); len(ps) != 0 {
		t.Fatalf("Build() with nil factory result = %v, want empty", ps)
	}
}

func TestHealthRegistry_BuildReturnsSinglePlugin(t *testing.T) {
	r := HealthRegistry{}
	r.Register(func() HealthPlugin { return &stubHealthPlugin{name: "a"} })
	ps := r.Build()
	if len(ps) != 1 || ps[0].Name() != "a" {
		t.Fatalf("Build() = %v, want [a]", healthPluginNames(ps))
	}
}

func TestHealthRegistry_BuildMultiplePlugins(t *testing.T) {
	r := HealthRegistry{}
	r.Register(func() HealthPlugin { return &stubHealthPlugin{name: "a"} })
	r.Register(func() HealthPlugin { return &stubHealthPlugin{name: "b"} })
	ps := r.Build()
	if len(ps) != 2 {
		t.Fatalf("Build() = %v, want [a b]", healthPluginNames(ps))
	}
}

func TestHealthRegistry_BuildMixedNilAndNonNil(t *testing.T) {
	r := HealthRegistry{}
	r.Register(func() HealthPlugin { return nil })
	r.Register(func() HealthPlugin { return &stubHealthPlugin{name: "b"} })
	r.Register(func() HealthPlugin { return nil })
	r.Register(func() HealthPlugin { return &stubHealthPlugin{name: "a"} })
	ps := r.Build()
	if len(ps) != 2 {
		t.Fatalf("Build() = %v, want [b a]", healthPluginNames(ps))
	}
	if ps[0].Name() != "b" || ps[1].Name() != "a" {
		t.Fatalf("Build() order = %v, want [b a]", healthPluginNames(ps))
	}
}

func TestResetHealthRegistryForTesting(t *testing.T) {
	resetHealthRegistry(t)
	RegisterHealthPlugin(func() HealthPlugin { return &stubHealthPlugin{name: "x"} })
	restore := ResetHealthRegistryForTesting()
	ps := BuildHealthPlugins()
	if len(ps) != 0 {
		t.Fatalf("after reset, Build() = %v, want empty", healthPluginNames(ps))
	}
	restore()
	ps = BuildHealthPlugins()
	if len(ps) != 1 || ps[0].Name() != "x" {
		t.Fatalf("after restore, Build() = %v, want [x]", healthPluginNames(ps))
	}
}
