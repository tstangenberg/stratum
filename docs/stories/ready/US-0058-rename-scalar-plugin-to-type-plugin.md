---
id: US-0058
tags: [refactor, plugin, types]
status: ready
---

# US-0058: Rename ScalarPlugin to TypePlugin

**As a** plugin author  
**I want** the type mapping plugin to be named `TypePlugin`  
**So that** the terminology reflects what the plugin actually does rather than leaking GraphQL vocabulary into the domain model

## Context

`ScalarPlugin` maps a named type to a PostgreSQL column type and a `graphql-go` output type. The name "scalar" is borrowed from GraphQL (leaf primitive types), but from Stratum's perspective the plugin defines a *type mapping* — it is not inherently GraphQL-specific. `TypePlugin` is more accurate as a domain name and keeps the plugin vocabulary consistent with the broader architecture (see ADR-1008).

## Acceptance Criteria

- [ ] `internal/plugin/scalar/` package and all files are renamed to `internal/plugin/types/`
- [ ] The `scalar.Plugin` interface is renamed to `types.Plugin` (or `plugin.TypePlugin` if kept in the root plugin package)
- [ ] All five built-in implementations (`String`, `ID`, `Int`, `Float`, `Boolean`) are updated
- [ ] `internal/server/server.go` and `internal/schema/graphql.go` references are updated
- [ ] ADR-1008 is updated to reflect the new name
- [ ] All tests pass and coverage is maintained

## E2E Tests

None — this is a pure rename with no behaviour change; covered by the existing test suite passing.
