---
id: US-011
tags: [data, query, relations]
status: open
---

# US-0011: Traverse a N:1 relation chain

**As a** developer  
**I want** to traverse multiple N:1 hops in a single GraphQL query  
**So that** I can navigate the full object graph without multiple round trips

## Context

Each N:1 hop adds a `LEFT JOIN` to the SQL query. Multiple hops chain LEFT JOINs. A safety limit of `max_depth: 5` (configurable in `stratum.yaml`) prevents unbounded traversal. The MVP-1 example traverses `PLZ → Ortschaft → Kanton` — a 2-hop chain. See the Relations design doc.

## Acceptance Criteria

- [ ] `query { plz { list { plz ortschaft { name kanton { kuerzel } } } } }` returns each PLZ with its Ortschaft and that Ortschaft's Kanton
- [ ] The full chain is resolved in a single SQL query using LEFT JOINs
- [ ] A missing intermediate relation (nullable N:1) returns `null` for that field, not an error
- [ ] Queries exceeding `max_depth` return a GraphQL error

## E2E Tests

- `e2e/query_test.go` — `TestTraversePLZOrtschaft`
