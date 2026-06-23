---
id: US-0055
tags: [community, launch]
status: done
---

# US-0055: Configure Dependabot

**As a** user evaluating Stratum for production use  
**I want** to see that dependencies are kept up to date  
**So that** I can trust the project is actively maintained and won't accumulate security debt

## Context

Dependabot opens automated PRs when dependencies have new versions or known vulnerabilities. It requires a `.github/dependabot.yml` config file. For a Go project it covers Go modules; optionally also GitHub Actions.

## Acceptance Criteria

- [x] `.github/dependabot.yml` exists
- [x] Go module updates are configured (`package-ecosystem: gomod`, `directory: /`, weekly schedule)
- [x] GitHub Actions updates are configured (`package-ecosystem: github-actions`, `directory: /`, weekly schedule)
- [x] The `dependencies` label is assigned to Dependabot PRs (uses the label preserved in US-0050)
- [ ] A Dependabot PR appears within one week of the config being merged (confirms the config is valid)

## E2E Tests

Manual verification: confirm `.github/dependabot.yml` is present and valid; check the Insights → Dependency graph → Dependabot tab shows the config is active.
