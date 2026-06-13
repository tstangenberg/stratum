---
id: US-006
tags: [data, mutation, relations]
status: done
---

# US-0006: Create a record with a relation

**As a** developer  
**I want** to create records that reference other records via N:1 relations  
**So that** I can persist connected domain data through the API

## Context

N:1 relations are expressed in the input as a foreign key ID field (e.g. `kantonId`). Stratum stores the FK in the database column derived from the field name (e.g. `kanton_id`). See ADR-1009 (FK naming convention). The MVP-1 example creates `Ortschaft` (references `Kanton`) and `PLZ` (references `Ortschaft`).

## Acceptance Criteria

- [x] `create` input accepts a relation field as an ID (e.g. `kantonId: "..."`)
- [x] Stratum stores the FK correctly in the database (`kanton_id` column)
- [x] The created record's relation is traversable in subsequent queries
- [x] Creating a record with a non-existent relation ID returns a GraphQL error
- [x] Creating `Ortschaft` with a valid `kantonId` persists correctly
- [x] Creating `PLZ` with a valid `ortschaftId` persists correctly

## E2E Tests

- `e2e/mutation_test.go` — `TestCreateOrtschaft`
- `e2e/mutation_test.go` — `TestCreatePLZ`
