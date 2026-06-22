# ADR-1011: API key authentication for MVP

**Status:** Accepted

## Context and Problem Statement

Stratum's MVP is self-hosted. Every request to the GraphQL and REST endpoints must be authenticated to prevent unauthorized access. The auth mechanism must be simple to configure, require no user management infrastructure, and be replaceable by a plugin Post-MVP.

## Considered Options

| Option | Pros | Cons |
|--------|------|------|
| **API Key (from ENV)** | Zero infrastructure, single ENV variable, easy to rotate, widely understood | Not suitable for multi-user or per-user access control |
| **No auth** | No setup | Unsafe for any network-exposed deployment |
| **JWT** | Per-user claims, stateless, widely supported | Requires an identity provider; too much infrastructure for a self-hosted MVP |
| **OAuth2** | Industry standard for delegated auth | Same infrastructure concern as JWT; overkill for MVP |

## Decision Outcome

Chosen: **API key auth via `STRATUM_API_KEY` environment variable**, because:

- A self-hosted MVP user controls the deployment environment. A single shared API key, configured as an ENV variable, is the simplest possible secure configuration.
- No user management, no token issuance, no identity provider — the operator sets the key, the client sends it.
- Auth is implemented as the `api-key-auth` plugin (see ADR-1008). Stratum core has no auth logic. Replacing it with `jwt-auth` or `oauth2-auth` Post-MVP requires only swapping the plugin — no core changes.

**Request flow:** Auth plugin runs first, before any resolver or hook. If `Authenticate` returns `Allowed: false`, the server responds with 401 and no further processing occurs.

## Implementation note

`api-key-auth` was initially implemented against a dedicated `AuthPlugin` interface (`Authenticate(*http.Request) AuthResult`). It was subsequently migrated to the general `HTTPMiddleware` interface (`Priority() int`, `Wrap(http.Handler) http.Handler`) to enable a configurable middleware pipeline. The observable behaviour and security properties are unchanged.
