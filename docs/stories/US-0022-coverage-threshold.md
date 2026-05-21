---
id: US-0022
tags: [ci, coverage, dev-tooling]
status: open
---

# US-0022: Coverage threshold gate

**As a** maintainer  
**I want** CI to fail when unit test coverage drops below a defined threshold  
**So that** coverage cannot silently regress as the codebase grows

## Context

A coverage floor prevents the common pattern where new code ships without tests and coverage slowly erodes. The threshold is configured in `codecov.yml` at the repo root. The initial threshold should be set conservatively based on actual coverage when this story is implemented — it can be raised over time.

## Acceptance Criteria

- [x] `codecov.yml` defines a minimum coverage threshold for the project
- [x] Codecov marks the PR check as failed when coverage drops below the threshold
- [x] The threshold applies to the `patch` (changed lines) as well as the `project` (overall)
- [x] The threshold value is documented in `codecov.yml` with a comment explaining the rationale

## E2E Tests

None — verified by inspecting the Codecov PR check on a PR that reduces coverage.
