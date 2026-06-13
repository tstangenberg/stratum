---
id: US-007
tags: [data, query, pagination]
status: done
---

# US-0007: List all records

**As a** developer  
**I want** to query a list of records of a type with pagination  
**So that** I can retrieve domain data in manageable pages

## Context

The `list` query is generated for every type. `limit` and `offset` are provided by the `pagination-simple` plugin from the MVP bundle. See ADR-1008 (plugin architecture) and US-0057 (configuration system).

## Acceptance Criteria

- [x] `query { <type> { list { ... } } }` returns all records up to the default limit (100)
- [x] `list(limit: N)` returns at most N records
- [x] `list(limit: N, offset: M)` skips the first M records
- [x] `limit` exceeding the hard maximum returns a GraphQL error
- [x] The hard maximum is read from `STRATUM_PLUGINS_PAGINATION_MAX_LIMIT` at startup via `pagination-simple` plugin (default: 1000)
- [x] The default page size is read from `STRATUM_PLUGINS_PAGINATION_DEFAULT_LIMIT` at startup via `pagination-simple` plugin (default: 100)
- [x] Empty table returns an empty array, not an error
- [x] Returns records in a stable order (insertion order by default)

## E2E Tests

- `e2e/query_test.go` — `TestListKantone`
