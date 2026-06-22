# ADR-1008: Seven-type plugin architecture

**Status:** Accepted

## Context and Problem Statement

Stratum needs to be extensible without requiring changes to its core. Extensions must cover: data types, query operators, list query augmentation, data hooks, schema change hooks, authentication, and health checks. The plugin model determines how extensions integrate, how they are registered, and how they are ordered.

## Considered Options

| Option | Pros | Cons |
|--------|------|------|
| **Seven distinct plugin types with dedicated interfaces** | Single responsibility per plugin, clean interfaces, independent installability, clear ordering | More interfaces to define and document |
| **Single generic plugin interface** | One interface to learn | Requires every plugin to declare which events it handles; cross-cutting concerns blur boundaries |
| **Hook-only (no plugin types for scalars/filters)** | Simpler for hooks | Scalar and filter logic would live in core or be unprincipled extensions |

## Decision Outcome

Chosen: **seven distinct plugin types**, each with a dedicated Go interface:

| Type | Interface | Responsibility | Status |
|------|-----------|---------------|--------|
| `ScalarPlugin` | `Name`, `ColumnType`, `GraphQLType` | Maps a GraphQL scalar to a PostgreSQL column type and a `graphql-go` scalar type | Implemented |
| `FilterPlugin` | `Name`, `ScalarType`, `Operators`, `ToSQL` | Adds filter operators (`eq`, `gte`, etc.) for a specific scalar type | Planned |
| `QueryModifier` | `Name`, `Arguments`, `ModifyQuery` | Augments list queries before execution — adds GraphQL arguments and appends SQL clauses | Implemented |
| `DMLHookPlugin` | `Name`, `Directives`, `Events`, `Execute` | Runs before/after INSERT, UPDATE, SELECT | Planned |
| `DDLHookPlugin` | `Name`, `Directives`, `Events`, `Execute` | Runs before/after schema migrations | Planned |
| `HTTPMiddleware` | `Name`, `Priority`, `Wrap` | Wraps HTTP requests for cross-cutting concerns (auth, rate-limiting, logging). Sorted by `Priority()` ascending — lower = outermost. Health endpoints always bypass the chain. | Implemented |
| `HealthPlugin` | `Name`, `Check` | Contributes a named health check to `GET /api/v1/health/ready` | Implemented |

## Plugin extension points

### ScalarPlugin — `internal/plugin/scalar/`

Maps a GraphQL scalar name to a PostgreSQL column type and a `graphql-go` output type. Registered in `NewStratumServer` as a `map[string]scalar.Plugin` keyed by scalar name. `BuildHandler` uses the map to resolve field column types and pass the correct `intType` to `QueryModifier.Arguments`.

### QueryModifier — `internal/plugin/`

Augments list queries before execution. Every registered modifier is applied in pipeline order:

1. `Arguments(intType)` — declares GraphQL arguments for the `list` field (return `nil` if none needed)
2. `ModifyQuery(query, params, args)` — appends SQL clauses and extends the parameter slice

```go
type QueryModifier interface {
    Name() string
    Arguments(intType graphql.Output) graphql.FieldConfigArgument
    ModifyQuery(query string, params []any, args map[string]any) (string, []any, error)
}
```

Registered in `NewStratumServer` as `[]plugin.QueryModifier`. Add a new modifier via `WithQueryModifiers(...)`. The default pipeline contains `pagination-simple`.

To add a new `QueryModifier` (e.g. a soft-delete filter):
1. Implement `plugin.QueryModifier` in a new package under `internal/plugin/`
2. Add it to the pipeline in `NewStratumServer` or pass it via `WithQueryModifiers`

### HTTPMiddleware — `internal/plugin/middleware.go`

Wraps HTTP requests for cross-cutting concerns. All registered middlewares are sorted by `Priority()` ascending and chained at startup. `/api/v1/health/live` and `/api/v1/health/ready` always bypass the chain.

```go
type HTTPMiddleware interface {
    Name() string
    Priority() int
    Wrap(next http.Handler) http.Handler
}
```

Registered via `WithMiddlewares(m ...plugin.HTTPMiddleware)`. Priority can be overridden in `stratum.yaml`:

```yaml
http-middleware:
  api-key-auth:
    priority: 100
```

→ env var `STRATUM_HTTP_MIDDLEWARE_API_KEY_AUTH_PRIORITY=100`. Each plugin reads its own env var and falls back to its compiled-in default.

### HealthPlugin — `internal/plugin/health.go`

Contributes a named component to `GET /api/v1/health/ready`. All checks run concurrently; the overall status is degraded if any check returns error.

```go
type HealthPlugin interface {
    Name() string
    Check(ctx context.Context) HealthStatus
}
```

Registered as variadic arguments to `NewStratumServer(plugins ...HealthPlugin)`.

## Registration

Plugins are wired via constructor injection in `NewStratumServer` and `server.Handler`. There is no global registry or `init()` side-effect pattern. The default binary in `cmd/stratum/main.go` composes the MVP bundle directly.

**MVP bundle:** 5 scalars (`String`, `ID`, `Int`, `Float`, `Boolean`), `pagination-simple`, `database-health`.

## Hook ordering

DML and DDL hooks (when implemented) will be ordered via numeric priority in `stratum.yaml` (lower = earlier). `HTTPMiddleware` ordering uses `Priority()` (not `stratum.yaml` numeric priority — each plugin reads its own env var which `stratum.yaml` sets). `QueryModifier` pipeline order is determined by slice position in `WithQueryModifiers`. Health plugins have no ordering — all checks run concurrently.
