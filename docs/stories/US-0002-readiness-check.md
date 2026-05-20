---
id: US-002
tags: [health, observability]
status: open
---

# US-0002: Readiness check

**As a** operator  
**I want** Stratum to expose a readiness endpoint that reflects the health of all its dependencies  
**So that** load balancers and orchestrators can stop routing traffic when Stratum cannot serve requests

## Context

Readiness aggregates all registered `HealthPlugin` implementations (see ADR-1008). Each plugin contributes a named component status. The overall status is `ok` only if all components report `ok`. Any failing component degrades the overall status and causes the endpoint to return 503.

Adding a new dependency to Stratum (S3, Redis, etc.) only requires a new `HealthPlugin` — no changes to this endpoint.

## Acceptance Criteria

- [ ] `GET /api/v1/health/ready` returns HTTP 200 when all health plugins report `ok`
- [ ] `GET /api/v1/health/ready` returns HTTP 503 when any health plugin reports `error`
- [ ] Response includes `status` (top-level: `ok` | `degraded`) and `components` (one entry per registered health plugin)
- [ ] Response format:
  ```json
  {
    "status": "ok",
    "components": {
      "database": { "status": "ok" }
    }
  }
  ```
- [ ] All health plugin checks run concurrently
- [ ] A plugin check that exceeds 5s is treated as `error` (timeout)

## E2E Tests

- `e2e/health_test.go` — `TestReadinessOK`
- `e2e/health_test.go` — `TestReadinessDegraded` (DB unreachable → 503)
