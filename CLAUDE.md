# CLAUDE.md — Stratum

## What this is

Schema-first data middleware. A user defines a domain model in GraphQL SDL → Stratum creates PostgreSQL tables and exposes a GraphQL API automatically. Self-hosted, open-source (AGPL v3).

## Documentation

- `docs/decisions/` — Architecture Decision Records (ADRs); read the relevant ADR before changing a technology or pattern
- `docs/stories/{open,ready,done,archive}/` — User stories with acceptance criteria and E2E test names; the active story defines what to build
- `docs/superpowers/plans/` — Implementation plans for stories; check for a plan matching the active story before writing any code (gitignored, local only)

## Tech stack

Go 1.26 · PostgreSQL · `vektah/gqlparser` (SDL parsing) · `graphql-go/graphql` (GraphQL execution) · `jackc/pgx/v5` · `oapi-codegen` (generated REST layer) · testcontainers-go (E2E tests)

## Stories and branches

- Stories live in `docs/stories/{open,ready,done,archive}/US-NNNN-<slug>.md`
- One branch per story: `story/US-NNNN-<slug>`
- Always work in a git worktree for the story branch — never edit files in the main checkout
- Never commit directly to `main` — always a PR
- No `Co-Authored-By` lines in commit messages

## Story lifecycle

`draft` → `ready` → `in-progress` → `in-review` → `done`

- Set status to `in-progress` when starting work on the branch
- Set status to `in-review` when opening a PR
- Set status to `done` when the PR is merged

## Before opening a PR

1. Verify every acceptance criterion in the story file is met
2. Check off each criterion (`- [ ]` → `- [x]`) in the story file and commit the update
3. Run the full test suite: `go test ./...`
4. Ensure 100% coverage for all internal packages: `go test ./internal/... -cover`

## Test strategy — Double Loop TDD

**Outer loop — always start here:**
1. Write a failing E2E test that covers the acceptance criterion end-to-end (real PostgreSQL, testcontainers)
2. Run it — confirm it fails for the right reason

**Inner loop — one unit at a time:**
3. For each unit needed to make the E2E pass:
   a. Write a failing unit test for the **happy path** — run it, confirm red
   b. Write the minimum production code to make it green — run, confirm green
   c. Write unit tests for error and edge cases one at a time — red, then green
   d. Refactor while tests stay green
4. Repeat step 3 until the E2E test passes

**Hard rules:**
- No production code without a failing test for it first
- E2E tests: real PostgreSQL only, no mocks
- Unit tests: mocks allowed for error/edge cases, not for happy path
- If a unit can't be tested without spinning up a real dependency, that is a design signal — extract an interface and use a test double in unit tests; reserve testcontainers for E2E and integration tests
- 100% coverage is the natural result of this process — do not bolt it on at the end

## Go rules

- Wrap errors: `fmt.Errorf("package: operation %q: %w", name, err)`
- No `panic` in library code
- Define interfaces at the point of use, keep them small (1–2 methods)
- Define the interface before writing the implementation — the interface is the design, not an afterthought
- If something is hard to test, it needs an interface; hard-to-test code is a design problem, not a test problem
- `context.Context` is always the first argument, never stored in structs
- No global variables — pass dependencies via constructors
- Table-driven tests, test files next to the code they test
- Stubs go directly on `StratumServer` — no `UnimplementedStrictServerInterface` embedding

## License

The pre-commit hook adds copyright and license. You don't have to add it.

## Key constraints

- No built-ins in core — scalars, filters, auth all come via plugins
- No N:M or 1:1 relations
- FK name = field name (`kanton_id`, not `kanton_type_id`)
- No mocks in E2E — real PostgreSQL only
- YAGNI — implement only what the active story requires
