# ADR-1002: Use PostgreSQL as the storage backend

**Status:** Accepted

## Context and Problem Statement

Stratum maps user-defined GraphQL types to persistent storage. The storage backend determines what SQL features are available for query resolution and how migrations are handled.

## Considered Options

| Option | Pros | Cons |
|--------|------|------|
| **PostgreSQL** | Production-ready, native column types, `json_agg` for 1:N relations, Atlas support, row-level security, rich extension ecosystem | Requires a running PostgreSQL instance |
| **SQLite** | Zero external dependency, easy local dev | Limited concurrency, no `json_agg`, unsuitable for multi-user production |
| **MySQL / MariaDB** | Wide adoption | Weaker JSON support, less capable Atlas integration, divergent SQL dialect |

## Decision Outcome

Chosen: **PostgreSQL**, because:

- Native column types (TEXT, INTEGER, DOUBLE PRECISION, BOOLEAN, TIMESTAMPTZ) map directly to GraphQL scalars — no JSON marshaling overhead.
- `json_agg` + `GROUP BY` resolves 1:N relations in a single SQL query without N+1 problems and without deduplication logic in Go.
- Atlas (`ariga.io/atlas`) has first-class PostgreSQL support with diff-based migrations.
- Row-level security and schema namespacing are available for Post-MVP multi-tenancy plugins without requiring a different database.
