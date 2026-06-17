---
id: US-0059
tags: [jobs, storage, infrastructure]
status: open
---

# US-0059: StoragePlugin interface and PostgreSQL implementation

**As a** Stratum operator  
**I want** job results stored durably  
**So that** export results survive worker restarts and are available for download

## Context

Job results (export output, import input) need a pluggable storage backend. This story defines the `StoragePlugin` interface and ships the MVP PostgreSQL implementation with gzip compression. See `docs/decisions/ADR-1015-job-system.md`.

Depends on US-0058 (job system).

## Acceptance Criteria

- [ ] `StoragePlugin` interface defined in `internal/plugin/storage` with `Write`, `Read`, `Delete`
- [ ] `stratum_job_results` table exists in `stratum_system` via a Goose migration (see ADR-1016) with `job_id` FK to `stratum_jobs`
- [ ] PostgreSQL implementation compresses on `Write` (gzip) and decompresses on `Read`
- [ ] `result_ref` in the job row is set to the job ID for the PostgreSQL backend
- [ ] Storage backend is wired into `StratumServer` at startup
- [ ] `GET /api/v1/jobs/{id}/result` streams the result via `StoragePlugin.Read` with `Content-Type: application/gzip` and correct `Content-Disposition` filename

## E2E Tests

- `e2e/jobs_test.go` — `TestExportResultDownload` (validated as part of US-0060)
