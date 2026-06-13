---
id: US-0061
tags: [jobs, import, data]
status: open
---

# US-0061: Import JobPlugin

**As a** developer  
**I want** to import records into a type from a JSON file  
**So that** I can seed environments or migrate data into Stratum

## Context

Second `JobPlugin` implementation. Triggered via `POST /api/jobs` with `{"operation": "import", "schema": "...", "type": "...", "data": [...]}`. The server streams the body to storage before returning 202 (supports both `application/json` and `Content-Encoding: gzip`). The worker reads from storage, decompresses, parses the JSON array, and INSERTs records in batches. Duplicate ID errors are recorded per record — processing continues. See `docs/superpowers/specs/2026-06-13-data-import-export-design.md`.

Depends on US-0058 (job system) and US-0059 (storage plugin).

## Acceptance Criteria

- [ ] `ImportPlugin` registered for `operation: "import"` in `internal/plugin/job/import`
- [ ] Server streams request body to `StoragePlugin.Write` before returning 202
- [ ] Both `application/json` and `Content-Encoding: gzip` request bodies are accepted
- [ ] Worker reads from storage, decompresses if needed, parses JSON array
- [ ] Records are INSERTed in batches
- [ ] Duplicate ID: error recorded in `summary.errors`, processing continues
- [ ] Completed job has `status: "done"` and `summary: {"processed": N, "failed": M, "errors": [...]}`
- [ ] Unknown schema or type returns `400` when the job is created via `POST /api/v1/jobs`

## E2E Tests

- `e2e/import_test.go` — `TestImportKantone`
- `e2e/import_test.go` — `TestImportGzip`
- `e2e/import_test.go` — `TestImportDuplicateId`
- `e2e/import_test.go` — `TestImportUnknownType`
