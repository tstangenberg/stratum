---
id: US-0033
tags: [ci, dev-tooling]
status: in-review
---

# US-0033: CI pipeline — build and test

**As a** contributor  
**I want** every push and PR to automatically build and run the test suite  
**So that** broken builds and failing tests are caught before they reach `main`

## Context

Foundational CI step. Creates the GitHub Actions workflow file that subsequent CI stories (lint, drift detection, license check) extend with additional jobs.

## Acceptance Criteria

- [ ] A GitHub Actions workflow at `.github/workflows/ci.yml` triggers on every push and PR targeting `main`
- [ ] The workflow pins the Go version to match `go.mod`
- [ ] `go build ./...` runs and fails the build on any compilation error
- [ ] `go vet ./...` runs and fails the build on any vet issue
- [ ] `go test -race ./...` runs and fails the build on any test failure
- [ ] Module dependencies are cached between runs (actions/cache on the Go module cache)
- [ ] The job name is `build` — subsequent CI stories add parallel jobs to the same workflow file

## E2E Tests

None — verified by pushing a commit that breaks the build and confirming CI fails, then fixing it and confirming CI passes.
