---
id: US-0020
tags: [schema]
status: done
---

# US-0020: Schema with multiple independent types

**As a** developer  
**I want** to define multiple types in a single schema upload  
**So that** Stratum creates a separate table for each type and exposes queries for all of them

## Context

Each GraphQL type maps to its own PostgreSQL table. Multiple types in a single schema share one GraphQL endpoint (`/graphql/{name}`) and one namespace. This story covers types with no relations between them — relations are handled separately in US-0006, US-0010, US-0011.

## Acceptance Criteria

- [x] Schema with two or more types is accepted in a single `POST /schemas/{name}`
- [x] A separate PostgreSQL table is created for each type
- [x] The GraphQL endpoint exposes `get` and `list` queries for all types
- [x] Records for each type can be created and queried independently
- [x] Types do not interfere with each other (separate namespaces in the GraphQL schema)

## E2E Tests

- `e2e/schema_test.go` — `TestSchemaMultipleTypes`
