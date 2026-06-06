---
id: US-008
tags: [data, query]
status: ready
---

# US-0008: Get a single record by ID

**As a** developer  
**I want** to retrieve a single record by its ID via GraphQL  
**So that** I can look up specific domain entities efficiently

## Context

The `get` query is generated for every type. It performs a primary key lookup — the most common read pattern. The MVP-1 example retrieves a single `Ortschaft` by ID.

## Acceptance Criteria

- [ ] `query { <type> { get(id: "...") { ... } } }` returns the record with the given ID
- [ ] Returns `null` for an unknown ID (not an error)
- [ ] Returns all requested scalar fields correctly typed

## E2E Tests

- `e2e/query_test.go` — `TestGetOrtschaft`
