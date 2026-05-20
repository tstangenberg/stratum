# Contributing to Stratum

## Branch protection

`main` is protected. All changes must arrive via a pull request and pass CI (`build` and `license` checks) before merging.

## Story branches

Branches implementing a story must follow: `story/US-NNNN-<slug>`

Example: `story/US-0033-ci-build-and-test`

The CI bot updates the story status automatically from the PR lifecycle:

| PR event | Story status |
|---|---|
| Draft PR opened | `in-progress` |
| PR marked ready for review | `in-review` |
| PR merged | `done` |
| PR closed without merge | `ready` |

## Bot token (BOT_TOKEN)

The story-status workflow commits directly to `main` (bypassing the PR requirement) using a fine-grained personal access token stored as the `BOT_TOKEN` repository secret. The token is owned by the repository owner, who is exempt from branch protection (`enforce_admins: false`).

**To rotate or set up the token:**

1. Go to GitHub → Settings → Developer settings → Personal access tokens → Fine-grained tokens
2. Create a new token:
   - Resource owner: `tstangenberg`
   - Repository access: Only `tstangenberg/stratum`
   - Permissions → Repository permissions → Contents: **Read and write**
3. Add the token as a repository secret named `BOT_TOKEN`:
   `gh secret set BOT_TOKEN --repo tstangenberg/stratum`

## License headers

Every hand-written `.go` file must carry an SPDX header:

```
// SPDX-License-Identifier: AGPL-3.0-or-later
```

To add missing headers locally:

```
go run github.com/google/addlicense -l agpl -c "Thorben Stangenberg" -s=only -ignore "**/*.gen.go" ./cmd ./tools ./internal
```
