---
id: US-0066
tags: [plugin, refactor]
status: open
---

# US-0066: ScalarPlugin self-registration

**As a** plugin author  
**I want** scalar plugins to self-register via `init()`  
**So that** adding a scalar type requires only a blank import, not a change to `NewStratumServer`

## Context

Scalar plugins are currently constructed and hard-coded in `NewStratumServer` as a `map[string]scalar.Plugin` keyed by scalar name. The five MVP scalars (`String`, `ID`, `Int`, `Float`, `Boolean`) are all explicitly imported and instantiated inside the server package.

The `eq` filter plugins also depend on scalars — they are constructed immediately after, using `scalars["String"].GraphQLType()` etc. This coupling will need to be considered during the refactor.

`HTTPMiddleware` already uses the self-registration pattern (ADR-1008): each plugin calls `plugin.RegisterMiddleware` in `init()` and the registry is collected at startup. This story applies the same pattern to `ScalarPlugin`.

## Acceptance Criteria

- [ ] `plugin.RegisterScalar(f func() scalar.Plugin)` and `plugin.BuildScalars() map[string]scalar.Plugin` exist, backed by a typed registry
- [ ] Each scalar plugin (`scalar/string`, `scalar/id`, `scalar/int`, `scalar/float`, `scalar/boolean`) self-registers via `init()`
- [ ] `NewStratumServer` no longer hard-codes scalar imports or constructs the scalars map internally; it calls `plugin.BuildScalars()` instead
- [ ] The server package no longer imports individual scalar packages
- [ ] `cmd/stratum/main.go` activates scalar plugins via blank imports
- [ ] All existing unit and E2E tests pass
- [ ] 100% statement coverage on all `internal/` packages

## E2E Tests

Existing E2E tests exercise scalar types end-to-end. If they pass after the refactor, the scalar wiring is correct. No new E2E tests are needed.
