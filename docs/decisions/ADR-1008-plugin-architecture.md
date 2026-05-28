# ADR-1008: Seven-type plugin architecture

**Status:** Accepted

## Context and Problem Statement

Stratum needs to be extensible without requiring changes to its core. Extensions must cover: data types, query operators, pagination, data hooks, schema change hooks, authentication, and health checks. The plugin model determines how extensions integrate, how they are registered, and how they are ordered.

## Considered Options

| Option | Pros | Cons |
|--------|------|------|
| **Seven distinct plugin types with dedicated interfaces** | Single responsibility per plugin, clean interfaces, independent installability, clear ordering | More interfaces to define and document |
| **Single generic plugin interface** | One interface to learn | Requires every plugin to declare which events it handles; cross-cutting concerns blur boundaries |
| **Hook-only (no plugin types for scalars/filters)** | Simpler for hooks | Scalar and filter logic would live in core or be unprincipled extensions |

## Decision Outcome

Chosen: **seven distinct plugin types**, each with a dedicated Go interface:

| Type | Interface | Responsibility |
|------|-----------|---------------|
| `ScalarPlugin` | `Name`, `ColumnType`, `GraphQLType` | Maps a GraphQL scalar to a PostgreSQL column type and a `graphql-go` scalar type; serialization is delegated to the `*graphql.Scalar` value returned by `GraphQLType` |
| `FilterPlugin` | `Name`, `ScalarType`, `Operators`, `ToSQL` | Adds filter operators (`eq`, `gte`, etc.) for a specific scalar type |
| `PaginationPlugin` | `Name`, `Arguments`, `ApplySQL`, `WrapResponse` | Adds pagination arguments (`offset`, cursor) to `list` queries |
| `DMLHookPlugin` | `Name`, `Directives`, `Events`, `Execute` | Runs before/after INSERT, UPDATE, SELECT |
| `DDLHookPlugin` | `Name`, `Directives`, `Events`, `Execute` | Runs before/after schema migrations |
| `AuthPlugin` | `Name`, `Authenticate` | Authenticates every request, returns `AuthContext` |
| `HealthPlugin` | `Name`, `Check` | Contributes a named health check to `GET /api/v1/health/ready` |

**Registration** uses Go's `init()` pattern — plugins register themselves via blank imports in `main.go`:

```go
import (
    _ "github.com/stratum/scalar-string"
    _ "github.com/stratum/api-key-auth"
    _ "github.com/stratum/database-health"
)
```

**Hook ordering** is configured in `stratum.yaml` with numeric priority keys (lower = earlier). DML and DDL hooks have separate numbering spaces. Health plugins have no ordering — all checks run concurrently and results are aggregated.

**HealthPlugin interface:**

```go
type HealthPlugin interface {
    Name() string
    Check(ctx context.Context) HealthStatus
}

type HealthStatus struct {
    Status  string         // "ok" | "error"
    Details map[string]any // optional extra info
}
```

The `/api/v1/health/ready` endpoint aggregates all registered health plugins. Adding a new dependency (S3, Redis, etc.) to Stratum requires only a new plugin — zero core changes.

**MVP bundle:** 13 plugins — 5 scalars, 5 eq-filters, `pagination-simple`, `api-key-auth`, `database-health` — are compiled into the default binary via blank imports.
