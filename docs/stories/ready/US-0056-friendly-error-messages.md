---
id: US-0056
tags: [data, mutation, errors]
status: ready
---

# US-0056: Friendly error messages for constraint violations

**As a** developer consuming the Stratum GraphQL API  
**I want** to receive clear, user-friendly error messages when a constraint violation occurs  
**So that** I can handle errors in my client without being exposed to internal database details

## Context

Currently, when a FK or other constraint violation occurs during a mutation, the raw `pgx` error is wrapped and forwarded as-is to the GraphQL consumer. This leaks internal table names (e.g. `swiss_kanton`) and PostgreSQL-specific error codes. For a self-hosted tool this is acceptable short-term, but the API surface should return structured, friendly errors instead.

Identified in a review comment on PR-06.

## Acceptance Criteria

- [ ] A FK violation (e.g. referencing a non-existent relation ID) returns a GraphQL error with a human-readable message (e.g. `"referenced kanton does not exist"`) instead of a raw PostgreSQL error
- [ ] No internal table names or PostgreSQL error codes are exposed in GraphQL error messages
- [ ] Other constraint violations (e.g. unique, not-null) return similarly friendly messages
- [ ] Unexpected database errors (not constraint violations) are still wrapped and returned as a generic error without internal details

## E2E Tests

- `e2e/mutation_test.go` — `TestCreateRecordFKViolationError`
- `e2e/mutation_test.go` — `TestCreateRecordFriendlyConstraintError`
