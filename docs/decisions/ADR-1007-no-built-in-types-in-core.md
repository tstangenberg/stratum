# ADR-1007: No built-in types in core — all scalars via plugins

**Status:** Accepted

## Context and Problem Statement

GraphQL defines five built-in scalar types: `String`, `Int`, `Float`, `Boolean`, `ID`. The question is whether Stratum's core should handle these natively, or whether they should be implemented as plugins like any other scalar.

## Considered Options

| Option | Pros | Cons |
|--------|------|------|
| **No built-ins — all scalars via plugins (including String, Int, etc.)** | Maximum consistency, clean plugin architecture, core has no type knowledge | Slightly more bootstrapping; the MVP bundle must always include the 5 core scalars |
| **Built-in scalars in core, plugins only for extensions** | Less setup for the common case | Special-cased logic in core; built-in and plugin scalars follow different code paths; harder to override |
| **Built-ins in core, but overridable by plugins** | Flexible | Two code paths for the same concept; unclear precedence |

## Decision Outcome

Chosen: **no built-ins — all scalars via plugins**, because:

- The core has zero type knowledge. Every scalar — including `String` and `Int` — passes through the same plugin interface. There is no special-cased path.
- Plugins can replace or extend any scalar without fighting core assumptions.
- The MVP bundle ships 5 scalar plugins (`scalar-string`, `scalar-int`, `scalar-float`, `scalar-boolean`, `scalar-id`) and 5 eq-filter plugins. Installing Stratum without these plugins produces a server that rejects all schemas — which is intentional and honest about what the core is.
- Post-MVP scalars (`DateTime`, `JSON`, `Money`) follow the exact same interface as the core-bundled ones. There is no privilege difference.
