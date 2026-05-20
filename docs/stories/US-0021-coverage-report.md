---
id: US-0021
tags: [ci, coverage, dev-tooling]
status: open
---

# US-0021: Test coverage report with Codecov

**As a** contributor  
**I want** test coverage to be measured and reported on every PR  
**So that** I can see which code is untested and track coverage trends over time

## Context

Go's built-in `-coverprofile` generates coverage data. Codecov consumes it, stores history, and posts a PR comment with coverage delta. The README gets a live badge. Codecov is free for public OSS repositories.

## Acceptance Criteria

- [ ] CI runs `go test -coverprofile=coverage.out ./...` on every push and PR
- [ ] Coverage report is uploaded to Codecov via the official GitHub Action
- [ ] Codecov posts a coverage summary comment on every PR
- [ ] README displays a Codecov coverage badge
- [ ] Coverage data is collected from unit tests only (E2E tests tracked separately)

## E2E Tests

None — this is a CI configuration story, verified by inspecting the GitHub Actions workflow and Codecov dashboard.
