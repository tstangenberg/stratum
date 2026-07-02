---
id: US-019
tags: [schema]
status: done
---

# US-0019: Schema re-upload — additive update

**As a** developer  
**I want** to re-upload a schema with an additional field  
**So that** Stratum adds the new column without touching existing data

## Context

Schema upload is an upsert. Re-uploading an existing schema with new fields triggers an Atlas migration that adds the missing columns. Existing rows have the new column set to `NULL` (or the column default). This story covers additive changes only — destructive changes (removing fields) are a separate concern handled Post-MVP via DDL hooks.

## Acceptance Criteria

- [x] Re-uploading a schema with an added field returns HTTP 200 with `version` incremented
- [x] The new column is added to the PostgreSQL table
- [x] Existing records remain intact with the new column set to `NULL`
- [x] New records can use the new field immediately after re-upload
- [x] Re-uploading an identical schema (no changes) is idempotent — succeeds, version increments

## E2E Tests

- `e2e/schema_test.go` — `TestSchemaReuploadAddField`
