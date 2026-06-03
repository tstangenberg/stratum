---
id: US-013
tags: [schema, scalar]
status: in-review
---

# US-0013: Schema with an Int field

**As a** developer  
**I want** to define an `Int` field in my schema  
**So that** Stratum maps it to an INTEGER column and correctly serializes integer values

## Context

`Int` is provided by the `scalar-int` plugin in the MVP bundle (see ADR-1007, ADR-1008). This story verifies the scalar plugin contract: correct PostgreSQL column type, correct serialization from DB to GraphQL response, correct parsing of input values.

## Acceptance Criteria

- [x] Schema with an `Int!` field is accepted
- [x] PostgreSQL column type is `INTEGER`
- [x] Integer values can be written via mutation and read back correctly
- [x] Out-of-range values (exceeding 32-bit integer) return a GraphQL error

## E2E Tests

- `e2e/schema_test.go` — `TestSchemaIntScalar`
