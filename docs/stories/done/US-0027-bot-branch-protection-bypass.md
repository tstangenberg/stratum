---
id: US-0027
tags: [ci, dev-tooling, workflow]
status: done
---

# US-0027: Bot account branch protection bypass for main

**As a** maintainer  
**I want** the automation bot to commit story status updates directly to `main`  
**So that** story files are always current without requiring a PR for every status transition

## Context

Story files live exclusively on `main`. The bot account (GitHub Actions bot or a dedicated service account) needs a branch protection bypass to commit directly. This is a standard pattern used by dependabot, release-please, and similar tools. Without this, the story status automation (US-0026) cannot function. Requires a one-time configuration in the GitHub repository settings.

## Acceptance Criteria

- [ ] A dedicated bot account or GitHub Actions token is configured for the repository
- [ ] The bot account is added to the branch protection bypass list for `main` in GitHub repository settings
- [ ] The bot can push commits to `main` with the message format `chore: story US-NNNN → <status> [skip ci]`
- [ ] CI does not trigger on bot commits (`[skip ci]` is respected by GitHub Actions)
- [ ] No other account or workflow gains unintended bypass access as a result of this configuration
- [ ] The bypass configuration is documented in `CONTRIBUTING.md` so future maintainers understand why it exists

## E2E Tests

None — verified by confirming a bot commit lands on `main` without triggering CI and without requiring a PR.
