---
id: US-001
tags: [health, observability]
status: in-progress
---

# US-0001: Liveness check

**As a** operator  
**I want** Stratum to expose a liveness endpoint  
**So that** my process manager or container runtime can detect when the process has stopped responding and restart it

## Context

Liveness answers one question: is the process alive? It does not check external dependencies — a failing database does not make the process "dead". Restarting the process would not fix a database outage, so liveness must never fail due to external state.

## Acceptance Criteria

- [ ] `GET /api/v1/health/live` returns HTTP 200 while the server is running
- [ ] Response body: `{"status": "ok"}`
- [ ] No external checks are performed (no database ping, no file system check)
- [ ] Responds in under 5ms under normal load

## E2E Tests

- `e2e/health_test.go` — `TestLivenessOK`
