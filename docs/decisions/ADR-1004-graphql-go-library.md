# ADR-1004: Use graphql-go/graphql for runtime schema generation

**Status:** Accepted

## Context and Problem Statement

Stratum must accept a user-supplied GraphQL SDL schema at runtime and immediately expose a fully functional GraphQL endpoint — without restarting the server or recompiling anything. The GraphQL library must support building and replacing schemas programmatically at runtime.

## Considered Options

| Option | Stars | Last release | Pros | Cons |
|--------|-------|-------------|------|------|
| **graphql-go/graphql** | 10,155 | Apr 2023 (API stable) | Programmatic runtime schema construction, in-process, no code-gen | Last release is old; slower than code-gen alternatives |
| **99designs/gqlgen** | 10,721 | Active | Fast, type-safe, widely used | Code-generation from a fixed SDL — cannot build schemas dynamically at runtime |
| **graph-gophers/graphql-go** | 4,752 | Active | Reflection-based, active maintenance | Requires Go structs at compile time — not suitable for fully dynamic schemas |
| **vektah/gqlparser + custom resolver layer** | 558 | Active | Maximum control, no library constraints | Significant implementation effort for MVP; parser only, no execution engine |

## Decision Outcome

Chosen: **graphql-go/graphql**, because it is the only Go library that supports fully dynamic, programmatic schema construction at runtime:

```go
fields := graphql.Fields{}
for _, field := range schema.Fields {
    fields[field.Name] = &graphql.Field{
        Type:    mapScalar(field.Type),
        Resolve: genericResolver(field),
    }
}
obj := graphql.NewObject(graphql.ObjectConfig{Name: schema.Name, Fields: fields})
```

`gqlgen` and `graph-gophers/graphql-go` both require knowing the schema at compile time. Stratum's core value proposition is that the schema arrives at runtime — eliminating both alternatives.

**Mitigation for library age:** The resolver layer is encapsulated behind an internal interface. If `graphql-go/graphql` becomes unmaintained or a better alternative emerges Post-MVP, the resolver layer can be swapped without touching plugin or schema logic.
