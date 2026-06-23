---
id: US-0023
tags: [ci, dev-tooling]
status: done
---

# US-0023: Code statistics with scc

**As a** contributor  
**I want** lines of code and complexity statistics to be reported in CI  
**So that** I can track the size and complexity of the codebase over time

## Context

`scc` (Sloc Cloc Code) is a fast Go binary that reports lines of code, blank lines, comments, and estimated complexity per language and package. Running it in CI makes the output available in the workflow summary without requiring any external service.

## Acceptance Criteria

- [x] CI installs and runs `scc` on every push to `main`
- [x] Output is printed to the GitHub Actions workflow summary
- [x] Report breaks down stats by package (not just totals)
- [x] Test files are reported separately from production code
- [x] `scc` failure does not fail the CI build — it is informational only

## E2E Tests

None — verified by inspecting the GitHub Actions workflow summary.
