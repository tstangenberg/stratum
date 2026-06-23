---
id: US-0065
tags: [plugin, refactor]
status: open
---

# US-0065: HealthPlugin self-registration

**As a** plugin author  
**I want** health plugins to self-register via `init()`  
**So that** adding a health plugin requires only a blank import, not a change to `NewStratumServer` or `main.go`

## Context

`HealthPlugin` is currently passed as variadic arguments to `NewStratumServer(plugins ...plugin.HealthPlugin)`. The database health plugin is constructed and wired by hand in `cmd/stratum/main.go`.

`HTTPMiddleware` already uses the self-registration pattern (ADR-1008): each plugin calls `plugin.RegisterMiddleware` in `init()` and the registry is collected at startup via `plugin.BuildMiddlewares()`. This story applies the same pattern to `HealthPlugin`.

## Acceptance Criteria

- [ ] `plugin.RegisterHealthPlugin(f func() plugin.HealthPlugin)` and `plugin.BuildHealthPlugins() []plugin.HealthPlugin` exist, backed by a `plugin.Registry`-based health registry
- [ ] `internal/plugin/database` self-registers via `init()` — returning `nil` from the factory when `STRATUM_DATABASE_URL` is not set
- [ ] `NewStratumServer` no longer accepts `HealthPlugin` as a variadic argument; health plugins are wired internally via `plugin.BuildHealthPlugins()`
- [ ] `cmd/stratum/main.go` activates the database health plugin via a blank import — no manual construction
- [ ] All existing unit and E2E tests pass
- [ ] 100% statement coverage on all `internal/` packages

## E2E Tests

Existing E2E tests exercise the `/api/v1/health/ready` endpoint. If they pass after the refactor, the health plugin wiring is correct. No new E2E tests are needed.
