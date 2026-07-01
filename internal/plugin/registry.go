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

// Registry collects middleware factories and builds a priority-sorted pipeline.
// Each factory is called at build time so that env vars and config are read
// after the process has fully initialised.
type Registry struct {
	factories []func() HTTPMiddleware
}

// Register adds a factory to the registry. The factory returns nil to signal
// that the plugin is not configured and should be omitted from the pipeline.
func (r *Registry) Register(f func() HTTPMiddleware) {
	r.factories = append(r.factories, f)
}

// Build calls every factory, discards nil results, and returns the rest sorted
// by ascending Priority() — lower values run first (outermost in the chain).
func (r *Registry) Build() []HTTPMiddleware {
	var ms []HTTPMiddleware
	for _, f := range r.factories {
		if m := f(); m != nil {
			ms = append(ms, m)
		}
	}
	sort.SliceStable(ms, func(i, j int) bool {
		return ms[i].Priority() < ms[j].Priority()
	})
	return ms
}

var middlewareRegistry Registry

// RegisterMiddleware adds a factory to the middleware registry.
func RegisterMiddleware(f func() HTTPMiddleware) {
	middlewareRegistry.Register(f)
}

// BuildMiddlewares calls the middleware registry and returns the sorted pipeline.
func BuildMiddlewares() []HTTPMiddleware {
	return middlewareRegistry.Build()
}
