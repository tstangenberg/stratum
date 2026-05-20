---
id: US-0028
tags: [ci, api, dev-tooling]
status: blocked
depends_on: [US-0033]
---

# US-0028: OpenAPI spec validation with Redocly

**As a** contributor  
**I want** the OpenAPI spec to be validated on every PR  
**So that** invalid or inconsistent spec changes are caught before they reach `main`

## Context

`api/openapi.yaml` is the contract for Stratum's REST API (see ADR-1013). Redocly CLI lints OpenAPI 3.0 specs — it catches structural errors, missing required fields, broken `$ref` references, and style violations. Running it in CI ensures the spec is always valid and the contract stays trustworthy.

A `redocly.yaml` config at the repo root controls which rules are enforced.

## Acceptance Criteria

- [ ] CI runs `redocly lint api/openapi.yaml` on every push and PR
- [ ] The build fails if the spec contains any errors
- [ ] A `redocly.yaml` config is committed to the repo root with the chosen ruleset
- [ ] The following rules are enforced as errors: no unused components, no broken `$ref`s, all operations have an `operationId`, all responses define a schema
- [ ] Warnings are reported but do not fail the build
- [ ] The lint step runs before the `oapi-codegen` generation step (US-0029)

## E2E Tests

None — verified by introducing a deliberate spec error and confirming CI fails.
