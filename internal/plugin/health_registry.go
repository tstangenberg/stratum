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

// HealthRegistry collects health-plugin factories and builds the plugin set.
// Each factory is called at build time so that env vars and config are read
// after the process has fully initialised.
type HealthRegistry struct {
	factories []func() HealthPlugin
}

// Register adds a factory to the registry. The factory returns nil to signal
// that the plugin is not configured and should be omitted.
func (r *HealthRegistry) Register(f func() HealthPlugin) {
	r.factories = append(r.factories, f)
}

// Build calls every factory, discards nil results, and returns the rest.
// Health plugins have no ordering — all checks run concurrently.
func (r *HealthRegistry) Build() []HealthPlugin {
	var ps []HealthPlugin
	for _, f := range r.factories {
		if p := f(); p != nil {
			ps = append(ps, p)
		}
	}
	return ps
}

var healthRegistry HealthRegistry

// RegisterHealthPlugin adds a factory to the health-plugin registry.
func RegisterHealthPlugin(f func() HealthPlugin) {
	healthRegistry.Register(f)
}

// BuildHealthPlugins calls the health-plugin registry and returns all configured plugins.
func BuildHealthPlugins() []HealthPlugin {
	return healthRegistry.Build()
}

// ResetHealthRegistryForTesting resets the health registry to empty and returns
// a restore function. Intended for use in tests only.
func ResetHealthRegistryForTesting() func() {
	original := healthRegistry
	healthRegistry = HealthRegistry{}
	return func() { healthRegistry = original }
}
