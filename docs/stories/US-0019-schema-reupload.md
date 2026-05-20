---
id: US-019
tags: [schema]
status: open
---

# US-0019: Schema re-upload — additive update

**As a** developer  
**I want** to re-upload a schema with an additional field  
**So that** Stratum adds the new column without touching existing data

## Context

Schema upload is an upsert. Re-uploading an existing schema with new fields triggers an Atlas migration that adds the missing columns. Existing rows have the new column set to `NULL` (or the column default). This story covers additive changes only — destructive changes (removing fields) are a separate concern handled Post-MVP via DDL hooks.

## Acceptance Criteria

- [ ] Re-uploading a schema with an added field returns HTTP 200 with `version` incremented
- [ ] The new column is added to the PostgreSQL table
- [ ] Existing records remain intact with the new column set to `NULL`
- [ ] New records can use the new field immediately after re-upload
- [ ] Re-uploading an identical schema (no changes) is idempotent — succeeds, version increments

## E2E Tests

- `e2e/schema_test.go` — `TestSchemaReuploadAddField`
