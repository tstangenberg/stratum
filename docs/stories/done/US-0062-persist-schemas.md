---
id: US-0062
tags: [schema, persistence, infrastructure]
status: done
---

# US-0062: Persist schemas to PostgreSQL and load on startup

**As a** Stratum operator  
**I want** uploaded schemas to survive application restarts  
**So that** I don't need to re-upload every schema each time the server restarts

## Context

The `schema.Store` is purely in-memory: every schema registered via `PUT /api/schemas/{name}` is lost when the process exits. The PostgreSQL tables for user data survive, but Stratum holds no record of the schemas that produced them, so `POST /graphql/{name}` returns 404 after a restart.

This story adds a `stratum_schemas` system table to persist the SDL and metadata, writes to it on every upsert, and loads all persisted schemas at startup so GraphQL endpoints are ready without any client action.

The `stratum_schemas` table lives in the `stratum_system` schema and is created via a Goose migration (see ADR-1016).

## Acceptance Criteria

- [x] A Goose migration creates the `stratum_system.stratum_schemas` table: `name TEXT PRIMARY KEY, sdl TEXT NOT NULL, version INT NOT NULL, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL`
- [x] `UpsertSchema` saves (or updates) the row in `stratum_schemas`
- [x] On startup, all rows from `stratum_schemas` are loaded; each SDL is parsed and its handler is built and registered in the store before the HTTP server begins accepting connections
- [x] A schema that fails to load at startup is logged and skipped — startup is not aborted
- [x] After a restart with the same database, `POST /graphql/{name}` responds correctly without any re-upload

## E2E Tests

- `e2e/schema_test.go` — `TestSchemaSurvivesRestart`
