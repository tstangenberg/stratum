# ADR-1003: Use GraphQL SDL as the schema definition language

**Status:** Accepted

## Context and Problem Statement

Users define their domain models by uploading a schema to Stratum. The schema language must be expressive enough to describe types, relations, and plugin-specific extensions, while being familiar and well-tooled.

## Considered Options

| Option | Pros | Cons |
|--------|------|------|
| **GraphQL SDL** | Same language as the query interface, self-documenting, Directives for extensions, widely understood, tooling (linters, formatters) exists | Requires parsing library |
| **JSON Schema** | Widely known, good tooling | Verbose, no Directive equivalent, different from query language |
| **Protobuf** | Compact, code-gen ecosystem | Overkill for this use case, different from query interface |
| **Custom DSL** | Full control | Must be built, documented, and tooled from scratch |

## Decision Outcome

Chosen: **GraphQL SDL**, because:

- There is no media break: users define their domain model in the same language they use to query it. Learning one language is enough.
- GraphQL Directives (`@constraint`, `@versioned`, `@protect`) are the natural extension point for plugins — no new syntax to invent.
- Existing SDL parsers (`vektah/gqlparser`) handle validation, syntax errors, and introspection out of the box.
- SDL is self-documenting — field types and nullability are explicit, and descriptions can be added inline.
