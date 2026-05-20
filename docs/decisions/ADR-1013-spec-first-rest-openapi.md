# ADR-1013: Spec-first REST development with OpenAPI and oapi-codegen

**Status:** Accepted

## Context and Problem Statement

Stratum exposes a REST API for schema management and system operations. Without a machine-readable contract, the API is defined only by implementation — drift between intent and code is invisible until a consumer breaks. A spec-first approach inverts this: the spec is the contract, and the implementation must conform to it.

## Considered Options

| Option | Pros | Cons |
|--------|------|------|
| **Spec-first with oapi-codegen** | Contract is explicit before code exists, Go interfaces generated from spec, CI validates spec, client SDK generation possible | Extra step in workflow; generated code must be committed or regenerated in CI |
| **Code-first (annotations)** | No separate spec file | Spec is derived from implementation — drift is still possible, spec lags changes |
| **Spec as documentation only** | No tooling coupling | No enforcement; spec drifts from implementation silently |

## Decision Outcome

Chosen: **spec-first with OpenAPI 3.0 and oapi-codegen**, because:

- The spec exists before any implementation — it is the design artifact, not a byproduct.
- `oapi-codegen` generates a Go `ServerInterface` from the spec. The compiler enforces that every endpoint in the spec is implemented. Unimplemented endpoints are compile errors.
- `redocly lint` runs in CI on every PR — an invalid spec fails the build before any Go code is touched.
- `net/http` server generation is supported natively in oapi-codegen v2 — no framework dependency introduced.

**Single document for MVP:** One `api/openapi.yaml` covers all REST endpoints, organised with tags (`schema`, `system`). GraphQL endpoints (`/graphql/{name}`) are not included — they are self-documenting via introspection.

**Workflow:**
1. Write or update `api/openapi.yaml`
2. Run `oapi-codegen` to regenerate `internal/api/api.gen.go`
3. Implement or update the `ServerInterface` in Go
4. CI validates the spec with `redocly lint`

**oapi-codegen config:** `api/oapi-codegen.yaml`

**Generated output:** `internal/api/api.gen.go` — committed to the repository, not generated at build time.
