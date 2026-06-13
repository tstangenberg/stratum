---
id: US-0053
tags: [community, launch]
status: ready
---

# US-0053: Configure branch protection rules for main

**As a** maintainer accepting community contributions  
**I want** branch protection rules on main  
**So that** no code is merged without review and passing CI, protecting the project from accidental or low-quality changes

## Context

Without branch protection, community contributors (and maintainers) can push directly to main or merge PRs without review. This is acceptable during solo development but must be in place before the OSS launch invites outside contributions.

## Acceptance Criteria

- [ ] Branch protection rule exists for `main`
- [ ] "Require a pull request before merging" is enabled
- [ ] "Require approvals" is enabled (minimum 1 approval)
- [ ] "Dismiss stale pull request approvals when new commits are pushed" is enabled
- [ ] "Require status checks to pass before merging" is enabled with all CI checks listed
- [ ] "Require branches to be up to date before merging" is enabled
- [ ] "Do not allow bypassing the above settings" is enabled (applies to admins too)

## E2E Tests

Manual verification: attempt to push directly to main and confirm it is rejected; open a draft PR and confirm status checks are required before merge is enabled.
