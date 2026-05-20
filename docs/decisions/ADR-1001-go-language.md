# ADR-1001: Use Go as the implementation language

**Status:** Accepted

## Context and Problem Statement

Stratum is a self-hosted binary that users run alongside their own PostgreSQL instance. The language choice shapes deployment complexity, ecosystem fit, and long-term maintainability.

## Considered Options

| Option | Pros | Cons |
|--------|------|------|
| **Go** | Single static binary, fast compilation, excellent stdlib, strong PostgreSQL ecosystem, simple cross-compilation | Verbose error handling, less expressive generics than Rust |
| **Rust** | Memory safety, top performance, growing ecosystem | Steep learning curve, slower compilation, smaller PostgreSQL ecosystem |
| **TypeScript / Node.js** | Large ecosystem, fast iteration | Runtime dependency (Node), weaker typing at runtime, harder to package as single binary |

## Decision Outcome

Chosen: **Go 1.22+**, because:

- Self-hosting is a first-class requirement — a single static binary with no runtime dependency is the best possible UX for that use case.
- Go 1.22+ ships method-based routing and path parameters in `net/http` stdlib, removing the need for a web framework.
- The PostgreSQL ecosystem in Go (pgx, Atlas) is mature and well-maintained.
- `graphql-go/graphql` — the only Go library that supports fully dynamic runtime schema construction — is written in Go, making in-process embedding straightforward.
