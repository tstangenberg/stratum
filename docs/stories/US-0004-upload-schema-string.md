---
id: US-004
tags: [schema, scalar]
status: open
---

# US-0004: Upload a schema with a String field

**As a** developer  
**I want** to upload a minimal GraphQL SDL schema containing a single String field  
**So that** Stratum creates the persistence layer and exposes a working GraphQL endpoint

## Context

This is the foundational story — it proves the full pipeline works end-to-end: REST upload → SDL parsing → Atlas migration → GraphQL endpoint live. Every other story builds on this. The schema is deliberately minimal: one type, one String field, one required ID.

## Acceptance Criteria

- [ ] `POST /schemas/{name}` with a valid SDL returns HTTP 200
- [ ] Response includes `name`, `status: "applied"`, `version: 1`, and `graphql_endpoint`
- [ ] A PostgreSQL table is created with a `TEXT` column for the String field
- [ ] `GET /graphql/{name}` responds to a query immediately after upload
- [ ] A record can be created and read back with the String value intact

## E2E Tests

- `e2e/schema_test.go` — `TestUploadSchemaString`
