---
id: US-017
tags: [schema]
status: ready
---

# US-0017: Schema with nullable fields

**As a** developer  
**I want** to define optional (nullable) fields in my schema  
**So that** I can model domain data where some attributes are not always present

## Context

In GraphQL SDL, `field: String!` is required (non-null) and `field: String` is nullable. Stratum maps required fields to `NOT NULL` columns and nullable fields to nullable columns. Nullable fields may be omitted from mutation input without error.

## Acceptance Criteria

- [ ] `field: String!` produces a `NOT NULL` column in PostgreSQL
- [ ] `field: String` (no `!`) produces a nullable column in PostgreSQL
- [ ] Creating a record without providing a nullable field succeeds — the field is stored as `NULL`
- [ ] Creating a record without providing a required field returns a GraphQL error
- [ ] Querying a `NULL` field returns `null` in the GraphQL response

## E2E Tests

- `e2e/schema_test.go` — `TestSchemaNullableFields`
