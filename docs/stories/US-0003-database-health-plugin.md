---
id: US-003
tags: [health, observability, database]
status: in-progress
---

# US-0003: Database health plugin

**As a** operator  
**I want** the readiness check to verify PostgreSQL connectivity  
**So that** I know immediately when Stratum cannot reach its database

## Context

`database-health` is a bundled `HealthPlugin` (see ADR-1008). It is included in the default binary and registers itself automatically. It performs a lightweight connectivity check — not a full query — to verify the database is reachable without adding unnecessary load.

## Acceptance Criteria

- [ ] Plugin is registered automatically in the default Stratum binary
- [ ] Check executes a lightweight PostgreSQL ping (e.g. `SELECT 1`)
- [ ] Returns `{ "status": "ok" }` when the database responds
- [ ] Returns `{ "status": "error", "details": { "error": "<message>" } }` when the database is unreachable
- [ ] Appears as `"database"` in the `components` map of `GET /api/v1/health/ready`
- [ ] Does not expose connection credentials or internal connection string in the error details

## E2E Tests

- `e2e/health_test.go` — `TestReadinessDegraded` (covered by US-0002)
