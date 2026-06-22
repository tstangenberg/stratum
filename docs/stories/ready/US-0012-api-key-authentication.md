---
id: US-012
tags: [auth]
status: in-progress
---

# US-0012: API key authentication

**As a** operator  
**I want** all API endpoints to require a valid API key  
**So that** unauthorized access to the GraphQL and REST APIs is blocked

## Context

Authentication is handled by the `api-key-auth` plugin from the MVP bundle. It reads `STRATUM_API_KEY` from the environment and validates the `X-API-Key` header on every request. The auth plugin runs before any resolver, hook, or schema logic. See ADR-1011 (API key auth) and ADR-1008 (plugin architecture).

## Acceptance Criteria

- [ ] Requests with a valid `X-API-Key` header are processed normally
- [ ] Requests without an `X-API-Key` header return HTTP 401
- [ ] Requests with an incorrect `X-API-Key` value return HTTP 401
- [ ] HTTP 401 responses do not reveal whether the key exists or is wrong
- [ ] Auth applies to all endpoints: GraphQL (`/graphql/{name}`) and REST (`/schemas/...`)
- [ ] `GET /api/v1/health/live` and `GET /api/v1/health/ready` are exempt from auth (public health checks)

## E2E Tests

- `e2e/auth_test.go` — `TestAuthMissingKey`
- `e2e/auth_test.go` — `TestAuthInvalidKey`
