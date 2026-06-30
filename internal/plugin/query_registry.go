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

import "sort"

// QueryModifierRegistry collects query-modifier factories and builds a
// priority-sorted pipeline. Each factory is called at build time so that env
// vars and config are read after the process has fully initialised.
type QueryModifierRegistry struct {
	factories []func() QueryModifier
}

// Register adds a factory to the registry. The factory returns nil to signal
// that the plugin is not configured and should be omitted from the pipeline.
func (r *QueryModifierRegistry) Register(f func() QueryModifier) {
	r.factories = append(r.factories, f)
}

// Build calls every factory, discards nil results, and returns the rest sorted
// by ascending Priority() — lower values run first.
func (r *QueryModifierRegistry) Build() []QueryModifier {
	var ms []QueryModifier
	for _, f := range r.factories {
		if m := f(); m != nil {
			ms = append(ms, m)
		}
	}
	sort.Slice(ms, func(i, j int) bool {
		return ms[i].Priority() < ms[j].Priority()
	})
	return ms
}

var queryModifierRegistry QueryModifierRegistry

// RegisterQueryModifier adds a factory to the query-modifier registry.
func RegisterQueryModifier(f func() QueryModifier) {
	queryModifierRegistry.Register(f)
}

// BuildQueryModifiers calls the query-modifier registry and returns the sorted pipeline.
func BuildQueryModifiers() []QueryModifier {
	return queryModifierRegistry.Build()
}
