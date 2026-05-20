# ADR-1006: Use net/http stdlib — no web framework

**Status:** Accepted

## Context and Problem Statement

Stratum exposes a REST API for schema management and a GraphQL endpoint for data access. A routing solution is needed. The choice between stdlib and a third-party framework affects the binary size, dependency surface, and upgrade path.

## Considered Options

| Option | Pros | Cons |
|--------|------|------|
| **net/http stdlib (Go 1.22+)** | Zero dependencies, stable API, ships with Go, method routing + path params built in | Less ergonomic middleware chaining than frameworks |
| **Chi** | Lightweight, idiomatic, good middleware ecosystem | External dependency; the reason to use it (routing) was resolved in Go 1.22 |
| **Echo / Gin / Fiber** | Feature-rich, popular | Heavier dependency, more opinions than Stratum needs |

## Decision Outcome

Chosen: **net/http stdlib**, because:

- Go 1.22 introduced `Method` and path-parameter routing directly in `ServeMux`. The primary reason to reach for Chi — routing — no longer applies.
- No external dependencies means a smaller binary, no transitive version conflicts, and no framework upgrade decisions to make.
- Stratum's REST surface is small (5 schema endpoints + 2 system endpoints). The ergonomic advantage of a framework does not justify the dependency.
