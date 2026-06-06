---
id: US-018
tags: [schema]
status: ready
---

# US-0018: Schema validation — invalid SDL rejected

**As a** developer  
**I want** Stratum to reject invalid SDL with a clear error message  
**So that** I get immediate, actionable feedback when my schema has a mistake

## Context

Validation runs before any migration. A rejected schema never touches the database. Error responses follow the standard Stratum error format with `line` and `column` details where applicable (see REST API design).

## Acceptance Criteria

- [ ] SDL with a syntax error returns HTTP 422 with `error: "validation_failed"` and `details` containing `line` and `column`
- [ ] SDL referencing an unknown directive returns HTTP 422 with a descriptive message identifying the directive name
- [ ] Empty SDL body returns HTTP 422
- [ ] A valid schema that was previously rejected leaves the database unchanged
- [ ] A valid schema upload after a failed upload succeeds normally

## E2E Tests

- `e2e/schema_test.go` — `TestSchemaValidationSyntaxError`
- `e2e/schema_test.go` — `TestSchemaValidationUnknownDirective`
