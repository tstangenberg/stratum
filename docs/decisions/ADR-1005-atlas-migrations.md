# ADR-1005: Use Atlas for user table migrations

**Status:** Accepted

> **Scope:** this ADR covers user tables only — tables created from user-supplied SDL (e.g. `myschema_City`). System tables (`stratum_system.*`) are managed by Goose; see ADR-1016.

## Context and Problem Statement

When a user uploads or updates a schema, Stratum must automatically create or alter PostgreSQL tables to match. The migration engine must be embeddable in Go (no external CLI dependency at runtime), support diff-based migration (compute what needs to change, not a list of hand-written migration files), and be reliable in production.

## Considered Options

| Option | Pros | Cons |
|--------|------|------|
| **Atlas** (`ariga.io/atlas`) | Diff-first, Go-native embeddable, no external tool, active development, PostgreSQL first-class | Relatively new compared to golang-migrate |
| **golang-migrate** | Battle-tested, widely used | File-based migration files only — Stratum would need to generate SQL diffs itself |
| **Custom SQL generation** | Full control | Significant implementation effort; error-prone edge cases (nullable changes, type changes) |

## Decision Outcome

Chosen: **Atlas** (`ariga.io/atlas`), because:

- Diff-first: Atlas computes the difference between the desired schema state and the current database state and generates the SQL to reconcile them. This is exactly what Stratum needs when a user updates their SDL.
- Go-native embeddable: Atlas runs in-process as a Go library. No shell exec, no external binary to install.
- No migration file management: Stratum does not maintain a migration file history — Atlas computes the migration on demand from the current schema state.
