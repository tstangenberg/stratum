---
id: US-014
tags: [schema, scalar]
status: in-review
---

# US-0014: Schema with a Float field

**As a** developer  
**I want** to define a `Float` field in my schema  
**So that** Stratum maps it to a DOUBLE PRECISION column and correctly handles decimal values

## Context

`Float` is provided by the `scalar-float` plugin (see ADR-1007, ADR-1008). The MVP-1 example uses Float for WGS84 coordinates (`lat`, `lon`). This story verifies decimal precision is preserved through the full round-trip: write → PostgreSQL → GraphQL response.

## Acceptance Criteria

- [ ] Schema with a `Float!` field is accepted
- [ ] PostgreSQL column type is `DOUBLE PRECISION`
- [ ] Float values can be written via mutation and read back with decimal precision intact
- [ ] Integer literals (e.g. `1`) are accepted as Float input

## E2E Tests

- `e2e/schema_test.go` — `TestSchemaFloatScalar`
