# ADR-1016: Use Goose for system table migrations

**Status:** Accepted

## Context and Problem Statement

Stratum owns a set of internal PostgreSQL tables (`stratum_jobs`, `stratum_schemas`, etc.) that live in the `stratum_system` schema. These tables are fixed by Stratum itself — not derived from user SDL — and evolve only when Stratum ships a new release (e.g., adding a column, creating a new table). They need a versioned migration mechanism so that production databases are upgraded safely across releases.

ADR-1005 chose Atlas for user table migrations, where the schema is computed at runtime from a user-supplied SDL diff. That model does not apply here: system table schemas are known ahead of time, change only at release boundaries, and benefit from a clear, auditable migration history rather than runtime diffing.

The current approach — ad-hoc `CREATE TABLE IF NOT EXISTS` statements in Go — does not handle `ALTER TABLE`, has no migration history, and gives no visibility into what schema version a production instance is running.

## Considered Options

| Option | Pros | Cons |
|--------|------|------|
| **Goose** (`github.com/pressly/goose/v3`) | File-based versioned SQL; supports Go migrations; embeds via `embed.FS`; clean programmatic API; active project | Another dependency |
| **golang-migrate** | Battle-tested, widely used, similar feature set | API is slightly more cumbersome; Go migration functions less ergonomic |
| **Atlas** (extend ADR-1005) | Single tool for all migrations | Diff-based model is a poor fit for fixed system schemas; conflates two different migration concerns |
| **`CREATE TABLE IF NOT EXISTS`** | No dependency | Cannot handle `ALTER TABLE`; no migration history; no visibility into schema version |

## Decision Outcome

Chosen: **Goose** (`github.com/pressly/goose/v3`).

**Scope:** system tables only — everything in the `stratum_system` PostgreSQL schema. User tables remain under Atlas as per ADR-1005.

**Migration files** live in `internal/system/migrations/` as numbered SQL files and are embedded in the binary via `embed.FS`:

```
internal/system/migrations/
  00001_create_stratum_schemas.sql
  00002_create_stratum_jobs.sql
  ...
```

**Migration history** is tracked in `stratum_system.goose_db_version` (Goose's default table, scoped to the system schema).

**Execution:** migrations run automatically at startup, before the HTTP server begins accepting connections. A failed migration aborts startup.

**Go migrations** (`.go` files alongside SQL files) are available for cases where a migration requires application logic, but SQL-only is the default.

**Consequences:**

- System table schema history is versioned, auditable, and visible in the migration table
- `ALTER TABLE` across releases is handled correctly — no manual intervention required
- The binary is self-contained: migration files are embedded, no external tool or file path needed at runtime
- Atlas and Goose coexist cleanly — they manage disjoint sets of tables with no overlap
