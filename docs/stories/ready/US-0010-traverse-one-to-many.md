---
id: US-010
tags: [data, query, relations]
status: in-progress
---

# US-0010: Traverse a 1:N relation

**As a** developer  
**I want** to traverse 1:N relations in a single GraphQL query  
**So that** I can fetch a parent record together with all its children without a separate request

## Context

1:N traversal is resolved using PostgreSQL `json_agg` + `GROUP BY` in a single SQL query — no N+1 problem, no deduplication in Go. Depth is limited to 1 level of list nesting in the MVP (lists within lists are Post-MVP). The MVP-1 example traverses `Kanton → Ortschaften`. See the Relations design doc.

## Acceptance Criteria

- [ ] `query { kanton { list { kuerzel ortschaften { name } } } }` returns each Kanton with its nested Ortschaften
- [ ] The nested list is resolved in a single SQL query (no N+1)
- [ ] A Kanton with no Ortschaften returns an empty array for `ortschaften`, not an error
- [ ] `limit`/`offset` on the parent list works correctly; children are not paginated in MVP

## E2E Tests

- `e2e/query_test.go` — `TestTraverseKantonOrtschaft`
