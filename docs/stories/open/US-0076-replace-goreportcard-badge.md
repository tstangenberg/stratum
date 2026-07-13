---
id: US-0076
tags: [ci, dev-tooling]
status: ready
depends_on: [US-0072]
---

# US-0076: Replace Go Report Card Badge

**As a** maintainer  
**I want** to replace the defunct Go Report Card badge in the README  
**So that** the README displays only working, meaningful quality indicators

## Context

goreportcard.com has been sunset and is no longer operational. The current `README.md` contains a Go Report Card badge that links to a dead service. It should be removed or replaced with an equivalent alternative.

Candidate replacements that are alive and meaningful for a Go project:

- **golangci-lint badge** — `golangci-lint` is already wired into CI (US-0072); a passing badge can be derived from the GitHub Actions workflow status for the lint job (same pattern as the existing CI badge)
- **pkg.go.dev badge** — official Go package index; renders a simple "reference docs" badge; zero setup required

Since golangci-lint runs in CI (US-0072), a lint-specific workflow status badge is the closest functional equivalent: it signals code quality at a glance without relying on a third-party SaaS.

## Acceptance Criteria

- [ ] The Go Report Card badge (`goreportcard.com`) is removed from `README.md`
- [ ] A golangci-lint workflow status badge is added to `README.md` (linking to the lint workflow run, same style as the existing CI badge)
- [ ] All remaining badges in `README.md` point to reachable URLs

## E2E Tests

None — verified by inspecting `README.md` and confirming the goreportcard link is gone and the new badge URL resolves.
