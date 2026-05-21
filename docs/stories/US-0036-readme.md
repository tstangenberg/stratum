---
id: US-0036
tags: [docs, dev-tooling]
status: ready
depends_on: [US-0021]
---

# US-0036: Project README

**As a** developer discovering Stratum  
**I want** a clear README that explains what the project does and how to get started  
**So that** I can quickly evaluate whether it fits my needs and know how to run it locally

## Context

A minimal `README.md` with CI and Codecov badges was added in US-0021. This story expands it into a complete project README covering what Stratum is, how to build and run it, and how to contribute.

## Acceptance Criteria

- [ ] README opens with a one-paragraph project description
- [ ] CI and Codecov badges are present and link to the correct targets
- [ ] "Getting started" section covers: prerequisites, building (`go build ./...`), and running (`go run ./cmd/stratum`)
- [ ] "Contributing" section links to `CONTRIBUTING.md`
- [ ] "License" section states AGPL-3.0-or-later and links to `LICENSE`
- [ ] A `LICENSE` file containing the AGPL-3.0 full text is committed to the repo root

## E2E Tests

None — verified by reading the rendered README on GitHub and confirming all badges and links resolve correctly.
