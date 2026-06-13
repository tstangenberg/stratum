---
id: US-007
tags: [data, query, pagination]
status: ready
---

# US-0007: List all records

**As a** developer  
**I want** to query a list of records of a type with pagination  
**So that** I can retrieve domain data in manageable pages

## Context

The `list` query is generated for every type. `limit` is provided by the core (default 100, hard max configurable via `STRATUM_SERVER_LIST_MAX_LIMIT`, default 1000). `offset` is provided by the `pagination-simple` plugin from the MVP bundle. See ADR-1008 (plugin architecture) and US-0057 (configuration system).

## Acceptance Criteria

- [ ] `query { <type> { list { ... } } }` returns all records up to the default limit (100)
- [ ] `list(limit: N)` returns at most N records
- [ ] `list(limit: N, offset: M)` skips the first M records
- [ ] `limit` exceeding the hard maximum returns a GraphQL error
- [ ] The hard maximum is read from `STRATUM_SERVER_LIST_MAX_LIMIT` at startup (default: 1000)
- [ ] Empty table returns an empty array, not an error
- [ ] Returns records in a stable order (insertion order by default)

## E2E Tests

- `e2e/query_test.go` — `TestListKantone`
