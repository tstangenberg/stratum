---
id: US-0060
tags: [jobs, export, data]
status: open
---

# US-0060: Export JobPlugin

**As a** developer  
**I want** to export all records of a type as a JSON file  
**So that** I can migrate data between environments or provide data portability

## Context

First `JobPlugin` implementation. Triggered via `POST /api/jobs` with `{"operation": "export", "schema": "...", "type": "..."}`. Records are SELECTed in batches, streamed as a gzip-compressed JSON array to the `StoragePlugin`, and available for download when the job completes. Single type only — full schema export is post-MVP. See `docs/superpowers/specs/2026-06-13-data-import-export-design.md`.

Depends on US-0058 (job system) and US-0059 (storage plugin).

## Acceptance Criteria

- [ ] `ExportPlugin` registered for `operation: "export"` in `internal/plugin/job/export`
- [ ] Records are SELECTed in batches (batch size: `STRATUM_PLUGINS_EXPORT_BATCH_SIZE`, default: 1000)
- [ ] Output is a valid JSON array of records streamed to `StoragePlugin.Write`
- [ ] Unknown schema or type returns `400` when the job is created
- [ ] Completed job has `status: "done"` and `summary: {"exported": N}`
- [ ] `GET /api/v1/jobs/{id}/result` returns a gzip-compressed JSON file named `export-{type}-{id}.json.gz`

## E2E Tests

- `e2e/export_test.go` — `TestExportKantone`
- `e2e/export_test.go` — `TestExportResultDownload`
- `e2e/export_test.go` — `TestExportUnknownType`
