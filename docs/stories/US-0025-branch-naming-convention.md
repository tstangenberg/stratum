---
id: US-0025
tags: [ci, dev-tooling, workflow]
status: open
---

# US-0025: Branch naming convention for stories

**As a** contributor  
**I want** a clear branch naming convention tied to user stories  
**So that** CI can identify which story a PR implements and automate status updates

## Context

The branch name is the most reliable signal available to GitHub Actions — it's set at PR creation and doesn't change. The convention `story/US-NNNN-<slug>` makes the story number machine-readable without requiring contributors to follow a commit message or PR title format. This story is a prerequisite for US-0026 (story status automation).

## Acceptance Criteria

- [ ] Convention documented in `CONTRIBUTING.md`: branches implementing a story must follow `story/US-NNNN-<slug>` (e.g. `story/US-0021-coverage-report`)
- [ ] A CI check runs on every PR and warns (but does not fail) if the branch name does not match the convention
- [ ] PRs not tied to a story (e.g. `fix/typo`, `chore/deps`) are explicitly allowed — the check only warns on branches starting with `story/` that don't match the pattern
- [ ] The convention is referenced in `docs/decisions/ADR-1012-user-story-conventions.md`

## E2E Tests

None — verified by inspecting the CI check output on a correctly and incorrectly named branch.
