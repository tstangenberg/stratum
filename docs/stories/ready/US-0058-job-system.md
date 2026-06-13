---
id: US-0058
tags: [jobs, infrastructure]
status: ready
---

# US-0058: Generic async job system

**As a** Stratum operator  
**I want** long-running operations to run asynchronously  
**So that** HTTP requests complete immediately and large operations don't time out

## Context

Stratum needs a generic async job system to support import, export, and future long-running operations. Jobs are stored in PostgreSQL, claimed by a worker pool using `SELECT ... FOR UPDATE SKIP LOCKED`, and dispatched to registered `JobPlugin` implementations. See `docs/decisions/ADR-1015-job-system.md` and `docs/superpowers/specs/2026-06-13-data-import-export-design.md`.

## Acceptance Criteria

- [ ] `stratum_jobs` table exists in the system schema via Atlas migration
- [ ] `JobPlugin` interface is defined in `internal/plugin/job`
- [ ] Worker pool starts with `StratumServer` and shuts down gracefully on context cancellation
- [ ] Pool size is configurable via `STRATUM_SERVER_WORKER_POOL_SIZE` (default: 4)
- [ ] Workers claim jobs atomically using `SELECT ... FOR UPDATE SKIP LOCKED`
- [ ] Job status transitions: `pending` → `running` → `done` | `failed`
- [ ] `POST /api/v1/jobs` creates a job and returns `202 { job_id, status: "pending" }`
- [ ] `GET /api/v1/jobs/{id}` returns current job status, timestamps, error, and summary
- [ ] `DELETE /api/v1/jobs/{id}` deletes the job row and its stored result
- [ ] Unknown `operation` returns a `400` error

## E2E Tests

- `e2e/jobs_test.go` — `TestCreateJobUnknownOperation`
- `e2e/jobs_test.go` — `TestJobLifecycle`
