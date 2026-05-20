---
id: US-0026
tags: [ci, dev-tooling, workflow]
status: open
---

# US-0026: Automate story status from PR lifecycle

**As a** maintainer  
**I want** story status to update automatically as a PR moves through its lifecycle  
**So that** the story files always reflect actual progress without manual discipline

## Context

Story files live exclusively on `main` — feature branches contain only code. A GitHub Actions workflow triggers on `pull_request` events, extracts the story number from the branch name (`story/US-NNNN-*`), and commits a status update directly to `main`. This requires the bot account to be exempt from branch protection on `main` (one config line — same pattern used by dependabot and release-please).

Requires US-0025 (branch naming convention).

Transition map:

| PR event | New story status | Committed to |
|---|---|---|
| PR opened as draft | `in-progress` | `main` |
| PR marked ready for review | `in-review` | `main` |
| PR merged | `done` | `main` |
| PR closed without merge | `ready` | `main` |

## Acceptance Criteria

- [ ] Workflow triggers on `pull_request` events: `opened`, `ready_for_review`, `closed`
- [ ] Branch name is parsed to extract `US-NNNN` — branches not matching the convention are silently skipped
- [ ] The correct story file in `docs/stories/` is identified from the extracted story number on `main`
- [ ] The `status` field in the story frontmatter is updated and committed directly to `main`
- [ ] Commit message format: `chore: story US-NNNN → <status> [skip ci]`
- [ ] `[skip ci]` prevents a CI loop from the bot commit
- [ ] If no matching story file is found, the workflow logs a warning and exits without error
- [ ] Bot account is configured with a branch protection bypass for `main` (documented in repo settings)
- [ ] Feature branches never contain story files

## E2E Tests

None — verified by opening a draft PR on a `story/US-NNNN-*` branch and observing the story file update on `main`.
