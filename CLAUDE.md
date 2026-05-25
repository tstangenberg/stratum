# CLAUDE.md — Stratum

## What this is

Schema-first data middleware. A user defines a domain model in GraphQL SDL → Stratum creates PostgreSQL tables and exposes a GraphQL API automatically. Self-hosted, open-source (AGPL v3).

## Design documents

Read these before touching the relevant code:

- `docs/plugin-system.md` — plugin types, interfaces, MVP bundle, init() registration
- `docs/rest-api.md` — REST endpoints, schema upload request/response format
- `docs/graphql-api.md` — nested namespace pattern, mutations, resolvers
- `docs/relations.md` — FK naming convention, N:1 / 1:N query resolution
- `docs/mvp/mvp-1.md` — Ortschaftenverzeichnis schema, fixture data, E2E test order
- `docs/superpowers/plans/` — implementation plans for active stories

## Tech stack

Go 1.26 · PostgreSQL · `vektah/gqlparser` (SDL parsing) · `graphql-go/graphql` (GraphQL execution) · `jackc/pgx/v5` · `oapi-codegen` (generated REST layer) · testcontainers-go (E2E tests)

## Stories and branches

- Stories live in `docs/stories/US-NNNN-<slug>.md`
- One branch per story: `story/US-NNNN-<slug>`
- Never commit directly to `main` — always a PR
- No `Co-Authored-By` lines in commit messages

## Test strategy — Double Loop TDD

Write a failing E2E test first. Drive implementation with failing unit tests. E2E tests run against real PostgreSQL (testcontainers) — no mocks.

## Go rules

- Wrap errors: `fmt.Errorf("package: operation %q: %w", name, err)`
- No `panic` in library code
- Define interfaces at the point of use, keep them small
- `context.Context` is always the first argument, never stored in structs
- No global variables — pass dependencies via constructors
- Table-driven tests, test files next to the code they test
- Stubs go directly on `StratumServer` — no `UnimplementedStrictServerInterface` embedding

## License

Every `.go` file must start with `// SPDX-License-Identifier: AGPL-3.0-or-later`. The `license` CI job enforces this via `addlicense -check`.

## Key constraints

- No built-ins in core — scalars, filters, auth all come via plugins
- No N:M or 1:1 relations — Post-MVP
- FK name = field name (`kanton_id`, not `kanton_type_id`)
- No mocks in E2E — real PostgreSQL only
- YAGNI — no code for Post-MVP features
