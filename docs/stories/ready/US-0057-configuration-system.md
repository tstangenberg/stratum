---
id: US-0057
tags: [config, infrastructure]
status: ready
---

# US-0057: Configuration system

**As an** operator deploying Stratum  
**I want** to supply configuration via a YAML file or environment variables  
**So that** I can configure both the server and plugins without rebuilding the binary

## Context

Stratum currently reads two hard-coded env vars (`STRATUM_ADDR`, `DATABASE_URL`). As the plugin set grows — especially with stateful plugins like `api-key-auth` — a proper configuration system is needed. The design is specified in `docs/superpowers/specs/2026-06-13-configuration-design.md` and the decision is recorded in `docs/decisions/ADR-1014-configuration-system.md`.

The core reads a YAML file and blindly expands all leaf values to env vars using a deterministic naming rule (`STRATUM_` prefix + uppercase path). Env vars already set in the environment always win. Plugins read their own env vars directly — the core knows nothing about plugin config.

## Acceptance Criteria

- [ ] `internal/config.Load()` is called at the top of `main()` before any other initialization
- [ ] `Load()` resolves the config file in order: `STRATUM_CONFIG` env var → `./stratum.yaml` → no file (not an error)
- [ ] Every YAML leaf value is expanded to an env var: path segments joined with `_`, uppercased, prefixed with `STRATUM_`
- [ ] `Load()` never overwrites an env var that is already set in the process environment
- [ ] YAML lists are comma-joined: `[a, b]` → `"a,b"`
- [ ] `STRATUM_ADDR` is renamed to `STRATUM_SERVER_ADDR` in `main.go`
- [ ] `DATABASE_URL` is renamed to `STRATUM_DATABASE_URL` in `main.go`
- [ ] A `stratum.yaml` example is added to the repo root (not committed as the default — as a `stratum.yaml.example`)
- [ ] `gopkg.in/yaml.v3` is added to `go.mod`

## E2E Tests

- `e2e/config_test.go` — `TestConfigYamlBindsAddr`: server started with a `stratum.yaml` containing `server.addr`; confirms it listens on the configured address
- `e2e/config_test.go` — `TestConfigEnvVarOverridesYaml`: server started with both a YAML file and a conflicting env var; confirms the env var wins
