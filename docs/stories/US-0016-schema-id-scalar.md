---
id: US-016
tags: [schema, scalar]
status: done
---

# US-0016: Schema with an explicit ID field

**As a** developer  
**I want** to define an explicit `ID` field in my schema  
**So that** I can supply my own identifiers instead of letting Stratum auto-generate them

## Context

`ID` is provided by the `scalar-id` plugin (see ADR-1007, ADR-1008). When `id: ID!` is present in the schema, the client may supply the ID in the mutation input. When omitted from input, Stratum generates one. Both cases must work. This story covers the explicit client-supplied case — auto-generation is covered implicitly by US-0004 through US-0015.

## Acceptance Criteria

- [x] Schema with an `ID!` field is accepted
- [x] PostgreSQL column type is `TEXT` and functions as the primary key
- [x] Client-supplied ID is stored and returned as-is
- [x] Duplicate ID on create returns a GraphQL error
- [x] If `id` is omitted from the mutation input, Stratum generates a unique ID

## E2E Tests

- `e2e/schema_test.go` — `TestSchemaIDScalar`
