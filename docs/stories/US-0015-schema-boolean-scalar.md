---
id: US-015
tags: [schema, scalar]
status: in-review
---

# US-0015: Schema with a Boolean field

**As a** developer  
**I want** to define a `Boolean` field in my schema  
**So that** Stratum maps it to a BOOLEAN column and correctly handles true/false values

## Context

`Boolean` is provided by the `scalar-boolean` plugin (see ADR-1007, ADR-1008). The MVP-1 example uses Boolean for `inAenderung` (record under active change). This story verifies true/false round-trip and correct column type.

## Acceptance Criteria

- [ ] Schema with a `Boolean!` field is accepted
- [ ] PostgreSQL column type is `BOOLEAN`
- [ ] `true` and `false` can be written via mutation and read back correctly
- [ ] String values (`"true"`) are rejected as invalid input

## E2E Tests

- `e2e/schema_test.go` — `TestSchemaBooleanScalar`
