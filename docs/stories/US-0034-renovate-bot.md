---
id: US-0034
tags: [ci, dev-tooling, dependencies]
status: in-progress
---

# US-0034: Automated dependency updates with Renovate

**As a** maintainer  
**I want** dependency updates proposed automatically as PRs  
**So that** Go modules and GitHub Actions stay current without manual tracking

## Context

Renovate scans `go.mod` and workflow files on a schedule and opens PRs for outdated dependencies. It is more configurable than Dependabot: updates can be grouped, scheduled, and automerged by severity.

Configuration lives in `renovate.json` at the repo root. Renovate is enabled by installing the GitHub App on the repository.

## Acceptance Criteria

- [ ] `renovate.json` is committed to the repo root
- [ ] Go module dependencies are updated weekly, grouped into a single PR
- [ ] GitHub Actions versions are updated weekly, grouped into a single PR
- [ ] Patch-level updates are automerged if CI passes; minor and major require manual review
- [ ] The Renovate GitHub App is installed on `tstangenberg/stratum`
- [ ] A test PR appears within the first scheduled run confirming Renovate is active

## E2E Tests

None — verified by confirming a Renovate PR or onboarding issue appears on the repository.
