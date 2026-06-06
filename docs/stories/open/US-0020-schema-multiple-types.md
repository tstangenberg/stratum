---
id: US-020
tags: [schema]
status: open
---

# US-0020: Schema with multiple independent types

**As a** developer  
**I want** to define multiple types in a single schema upload  
**So that** Stratum creates a separate table for each type and exposes queries for all of them

## Context

Each GraphQL type maps to its own PostgreSQL table. Multiple types in a single schema share one GraphQL endpoint (`/graphql/{name}`) and one namespace. This story covers types with no relations between them — relations are handled separately in US-0006, US-0010, US-0011.

## Acceptance Criteria

- [ ] Schema with two or more types is accepted in a single `POST /schemas/{name}`
- [ ] A separate PostgreSQL table is created for each type
- [ ] The GraphQL endpoint exposes `get` and `list` queries for all types
- [ ] Records for each type can be created and queried independently
- [ ] Types do not interfere with each other (separate namespaces in the GraphQL schema)

## E2E Tests

- `e2e/schema_test.go` — `TestSchemaMultipleTypes`
