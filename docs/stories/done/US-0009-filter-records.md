---
id: US-009
tags: [data, query, filter]
status: done
---

# US-0009: Filter records by field value

**As a** developer  
**I want** to filter records by field value in GraphQL queries  
**So that** I can retrieve only the records that match a condition without fetching all data

## Context

Filtering is operator-based. The `eq` operator is provided for all MVP scalar types by the bundled filter plugins (`string-eq-filter`, `int-eq-filter`, etc.). Filters are passed as input objects on the `list` query. See ADR-1008 (plugin architecture). The MVP-1 example filters `PLZ` by the `plz` integer field.

## Acceptance Criteria

- [x] `list(filter: { <field>: { eq: <value> } })` returns only matching records
- [x] `eq` filter works for all MVP scalar types: `String`, `Int`, `Float`, `Boolean`, `ID`
- [x] Filter with no matches returns an empty array
- [x] Filter combined with `limit`/`offset` works correctly
- [x] `plz { list(filter: { plz: { eq: 8001 } }) { plz ortschaft { name } } }` returns the correct record

## E2E Tests

- `e2e/query_test.go` — `TestFilterPLZ`
