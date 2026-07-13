---
id: US-0063
tags: [config, infrastructure, dx]
status: done
---

# US-0063: Self-documenting configuration variables

**As a** developer or operator  
**I want** every configuration variable to be declared as a named constant with a description and default value in the code  
**So that** I can discover all available options and their defaults without reading multiple source files

## Context

The configuration system (ADR-1014, US-0057) blindly expands YAML keys to env vars — the core knows nothing about which keys exist. This is intentional, but it means config options are currently undiscoverable: magic strings scattered across `main.go` and plugin constructors, no defaults documented anywhere.

The fix is to declare each env var as an exported constant in the owning package, annotated with a structured doc comment. A small code generator (`cmd/configdocs`) walks all packages via `go/ast`, collects every `STRATUM_*` constant, and emits a markdown reference table. No central registry, no changes to ADR-1014.

## Acceptance Criteria

- [x] Every `STRATUM_*` env var used in the codebase is declared as an exported constant in the package that owns it (e.g. `config.EnvServerAddr`, `config.EnvDatabaseURL`)
- [x] Each constant has a doc comment following the convention: first line is a human-readable description; a `Default:` line gives the default value (or `none` if required)
- [x] No magic strings remain for `STRATUM_*` vars — all call sites reference the constant
- [x] `cmd/configdocs/main.go` generates `docs/configuration.md` — a markdown table with columns: Variable, Default, Description
- [x] `go generate ./...` (or an explicit `go run ./cmd/configdocs`) regenerates the doc; the command is documented in the generated file header
- [x] The generated `docs/configuration.md` is committed to the repo and kept up to date
- [x] `docs/decisions/ADR-1014-configuration-system.md` is updated with an addendum documenting the constant declaration convention and doc comment format

## E2E Tests

None required — this is a DX/documentation story with no runtime behaviour change. A CI lint step checking that `docs/configuration.md` is not stale (re-run generator, diff is empty) is sufficient and can be added as a follow-up.
