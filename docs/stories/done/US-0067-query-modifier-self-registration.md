---
id: US-0067
tags: [plugin, refactor]
status: done
---

# US-0067: QueryModifier self-registration

**As a** plugin author  
**I want** query modifiers to self-register via `init()`  
**So that** adding a query modifier requires only a blank import, not a call to `WithQueryModifiers`

## Context

`QueryModifier` plugins are currently wired via `WithQueryModifiers(modifiers ...plugin.QueryModifier)` on `StratumServer`. The `pagination-simple` modifier is constructed directly inside `NewStratumServer` as the default pipeline.

`HTTPMiddleware` already uses the self-registration pattern (ADR-1008): each plugin calls `plugin.RegisterMiddleware` in `init()` and the registry is collected at startup. This story applies the same pattern to `QueryModifier`.

Unlike `HTTPMiddleware`, `QueryModifier` pipeline order matters for query construction (e.g. pagination must come after filters). The registry should sort modifiers by priority, consistent with the `HTTPMiddleware` pattern.

## Acceptance Criteria

- [x] `plugin.RegisterQueryModifier(f func() plugin.QueryModifier)` and `plugin.BuildQueryModifiers() []plugin.QueryModifier` exist, backed by a typed registry that sorts by `Priority()`
- [x] `plugin.QueryModifier` gains a `Priority() int` method; `pagination-simple` returns a sensible default (e.g. 100)
- [x] `pagination-simple` self-registers via `init()`
- [x] `NewStratumServer` no longer hard-codes `simplepagination.New()` or accepts `WithQueryModifiers`; it calls `plugin.BuildQueryModifiers()` instead
- [x] The server package no longer imports `pagination/simple` directly
- [x] `cmd/stratum/main.go` activates `pagination-simple` via a blank import
- [x] All existing unit and E2E tests pass
- [x] 100% statement coverage on all `internal/` packages

## E2E Tests

Existing E2E tests exercise list queries with pagination. If they pass after the refactor, the query modifier wiring is correct. No new E2E tests are needed.
