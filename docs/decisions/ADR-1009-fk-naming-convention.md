# ADR-1009: FK column name is derived from the field name, not the type name

**Status:** Accepted

## Context and Problem Statement

When Stratum maps a many-to-one relation to PostgreSQL, it must create a foreign key column. The column name must be derivable from the SDL without ambiguity — even when a single type references another type more than once.

## Considered Options

| Option | Example SDL | Generated FK column | Ambiguous with two fields of same type? |
|--------|-------------|--------------------|-----------------------------------------|
| **Field name + `_id`** | `billingAddress: Address!` | `billing_address_id` | No — field names are unique per type |
| **Type name + `_id`** | `billingAddress: Address!` | `address_id` | Yes — second field also produces `address_id` |
| **Explicit `@relation(field:)` directive** | `@relation(field: "billing_address_id")` | user-defined | No ambiguity, but verbose and requires directive for every relation |

## Decision Outcome

Chosen: **field name + `_id`** (`{camelCaseField}_id` → `{snake_case_field}_id`), because:

- Field names are unique within a type by GraphQL spec, so the FK column name is always unambiguous.
- No directive is needed for the common case — the convention is derivable from the SDL alone.
- Multiple relations to the same type work naturally:

```graphql
type Order {
  billingAddress:  Address!   # → billing_address_id
  shippingAddress: Address!   # → shipping_address_id
}
```

A `@relation(field:)` directive is planned for Post-MVP edge cases (e.g., explicitly controlling the FK name for compatibility with an existing database), but is not needed for the MVP.
