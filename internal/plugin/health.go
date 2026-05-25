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
package plugin

import "context"

const (
	StatusOK    = "ok"
	StatusError = "error"
)

// HealthStatus is returned by a HealthPlugin check.
type HealthStatus struct {
	Status  string         // StatusOK | StatusError
	Details map[string]any // optional, must not contain credentials
}

// HealthPlugin contributes a named component to GET /api/v1/health/ready.
type HealthPlugin interface {
	Name() string
	Check(ctx context.Context) HealthStatus
}
