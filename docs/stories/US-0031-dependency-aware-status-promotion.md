---
id: US-0031
tags: [ci, dev-tooling]
status: blocked
depends_on: [US-0026, US-0027]
---

# US-0031: Dependency-aware story status promotion

**As a** contributor  
**I want** stories to automatically move from `blocked` to `ready` when all their dependencies are done  
**So that** I don't have to manually monitor dependency chains

## Context

Stories may declare dependencies via `depends_on` in frontmatter (see ADR-1012). A story with unmet dependencies has status `blocked` — it should not be picked up. When the CI bot marks a story as `done` (US-0026), it should check whether any `blocked` stories were waiting on it and promote them to `ready` if all their dependencies are now satisfied.

This extends the existing story status bot (US-0026). The bot already commits directly to `main` using the branch protection bypass from US-0027.

## Acceptance Criteria

- [ ] When the bot sets a story to `done`, it scans all other stories for a `depends_on` field that includes the just-completed story ID
- [ ] For each such story, the bot checks whether every ID listed in `depends_on` has status `done`
- [ ] If all dependencies are `done`, the bot sets the story's status to `ready` and commits to `main`
- [ ] A story with no `depends_on` field is unaffected
- [ ] The commit message identifies the promotion: `bot: US-NNNN unblocked → ready (deps: US-XXXX, US-YYYY)`
- [ ] If only some dependencies are satisfied the story remains `blocked`

## E2E Tests

None — verified by marking a dependency story `done` and confirming the dependent story transitions to `ready` in the next CI run.
