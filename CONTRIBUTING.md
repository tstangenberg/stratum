# Contributing to Stratum

## Branch protection

`main` is protected. All changes must arrive via a pull request and pass CI (`build` and `license` checks) before merging.

## Story branches

Branches implementing a story must follow: `story/US-NNNN-<slug>`

Example: `story/US-0033-ci-build-and-test`

## Story folder convention

Story files live in `docs/stories/` under one of four subfolders:

| Folder | Meaning |
|---|---|
| `open/` | Draft, or `status: ready` but waiting on unmet dependencies |
| `ready/` | `status: ready` and all dependencies met — actionable now |
| `done/` | Merged and shipped |
| `archive/` | Rejected, won't-do, superseded, or indefinitely deferred |

`in-progress` and `in-review` states are **not** tracked in the filesystem — they are visible from the open PR itself.

### Status values (frontmatter)

The `status` field in frontmatter remains the machine-readable source of truth:

`draft` → `ready` → `in-progress` → `in-review` → `done`

### Moving stories

Status transitions are **manual**: a maintainer moves the file to the appropriate folder and updates the frontmatter field.

When a story is moved to `done/`, a CI workflow (`story-promote.yml`) automatically scans `open/` for stories with `status: ready` whose `depends_on` list is fully satisfied (all referenced stories are in `done/`). Those stories are `git mv`'d to `ready/` and committed to `main`.

Stories in `open/` with **no** `depends_on` field are never promoted automatically — they require a manual decision.

Archive is always a manual operation — no CI trigger moves files to `archive/`.

## Bot token (BOT_TOKEN)

The story-promote workflow pushes directly to `main` using a fine-grained personal access token stored as the `BOT_TOKEN` repository secret. The token is owned by the repository owner, who is exempt from branch protection (`enforce_admins: false`).

**To rotate or set up the token:**

1. Go to GitHub → Settings → Developer settings → Personal access tokens → Fine-grained tokens
2. Create a new token:
   - Resource owner: `tstangenberg`
   - Repository access: Only `tstangenberg/stratum`
   - Permissions → Repository permissions → Contents: **Read and write**
3. Add the token as a repository secret named `BOT_TOKEN`:
   `gh secret set BOT_TOKEN --repo tstangenberg/stratum`

## Pre-commit hooks

This project uses pre-commit to run checks locally before committing. The hooks check for:

- Copyright and SPDX license headers
- Go formatting (`gofmt`)
- Go static analysis (`go vet`)
- OpenAPI spec validation

**Installation:**

```bash
brew install license-eye  # or: go install github.com/apache/skywalking-eyes/cmd/license-eye@latest
brew install pre-commit
pre-commit install
```

**Run manually on all files:**

```bash
pre-commit run --all-files
```

**Auto-fix copyright headers locally:**

```bash
license-eye header fix
```

## License headers

Every hand-written `.go` file must carry both copyright and SPDX headers:

```
Copyright 2026 Thorben Stangenberg
SPDX-License-Identifier: AGPL-3.0-or-later
```
