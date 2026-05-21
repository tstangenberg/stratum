// SPDX-License-Identifier: AGPL-3.0-or-later
package plugin

import "context"

// HealthStatus is returned by a HealthPlugin check.
type HealthStatus struct {
	Status  string         // "ok" | "error"
	Details map[string]any // optional, must not contain credentials
}

// HealthPlugin contributes a named component to GET /api/v1/health/ready.
type HealthPlugin interface {
	Name() string
	Check(ctx context.Context) HealthStatus
}
