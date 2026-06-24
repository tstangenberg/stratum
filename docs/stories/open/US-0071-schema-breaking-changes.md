---
id: US-0071
tags: [schema, migration]
status: draft
depends_on: [US-0019]
---

# US-0071: Schema breaking changes — non-null addition, field removal, type change

**As a** developer  
**I want** Stratum to handle breaking schema changes safely  
**So that** I can evolve my schema without silently corrupting data or breaking existing clients

## Context

US-0019 covers additive nullable changes, which are safe to auto-migrate. Three remaining migration types require explicit policy decisions before Stratum can act on them:

| Change | Backward compat | Forward compat | Problem |
|--------|----------------|----------------|---------|
| Add non-null field | yes | **no** | Existing rows have NULL — violates the constraint |
| Remove field | **no** | yes | Existing API clients and queries that reference the field break |
| Change field type | **no** | **no** | Existing data may not be representable in the new type; existing clients break |

Each type requires a different migration strategy and possibly a different SDL annotation (e.g. `@default`, `@deprecated`) or an explicit opt-in from the developer.

Decisions to make before implementation:
- Add non-null field: require `@default(value: "...")` directive, or reject outright, or allow only when the table is empty.
- Remove field: reject immediately, soft-delete via `@deprecated` first, or allow with an explicit `@drop` opt-in.
- Change type: reject in all cases, or allow a defined set of safe casts (e.g. `Int` → `Float`).

## Acceptance Criteria

- [ ] Re-uploading a schema that adds a non-null field without a default is rejected with HTTP 422 and `error: "breaking_change"` identifying the field
- [ ] Re-uploading a schema that removes a field is rejected with HTTP 422 and `error: "breaking_change"` identifying the field
- [ ] Re-uploading a schema that changes a field's type is rejected with HTTP 422 and `error: "breaking_change"` identifying the field and the type change
- [ ] A rejected breaking change leaves the database and stored schema unchanged
- [ ] (stretch) A non-null addition with `@default(value: "...")` is accepted and existing rows are backfilled with the default

## E2E Tests

- `e2e/schema_test.go` — `TestSchemaReuploadAddNonNullFieldRejected`
- `e2e/schema_test.go` — `TestSchemaReuploadRemoveFieldRejected`
- `e2e/schema_test.go` — `TestSchemaReuploadChangeTypeRejected`
