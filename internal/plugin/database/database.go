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

package database

import (
	"context"
	"regexp"

	"github.com/tstangenberg/stratum/internal/plugin"
)

// credentialsPattern matches the user:password portion of a DSN URL.
var credentialsPattern = regexp.MustCompile(`://[^:@/]+:[^@/]*@`)

// Pinger is satisfied by *pgxpool.Pool.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Plugin checks PostgreSQL connectivity.
type Plugin struct {
	db Pinger
}

// New returns a Plugin that uses db for connectivity checks.
func New(db Pinger) *Plugin {
	return &Plugin{db: db}
}

func (p *Plugin) Name() string { return "database" }

func (p *Plugin) Check(ctx context.Context) plugin.HealthStatus {
	if err := p.db.Ping(ctx); err != nil {
		msg := credentialsPattern.ReplaceAllString(err.Error(), "://<redacted>@")
		return plugin.HealthStatus{
			Status:  plugin.StatusError,
			Details: map[string]any{"error": msg},
		}
	}
	return plugin.HealthStatus{Status: plugin.StatusOK}
}
