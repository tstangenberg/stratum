---
id: US-0041
tags: [community, launch]
status: ready
depends_on: [US-0040]
---

# US-0041: Add CONTRIBUTING.md

**As a** potential contributor  
**I want** clear guidance on how to contribute to Stratum  
**So that** I can set up my dev environment and submit a PR without needing to ask basic questions

## Context

Added at OSS launch. Sets expectations before the community forms. Depends on US-0040 so the Discussions link is live when referenced.

## Acceptance Criteria

- [ ] `CONTRIBUTING.md` exists at the repo root
- [ ] Covers: dev environment setup, how to run tests, Double Loop TDD workflow, how to submit a PR
- [ ] Covers: plugin interface contracts and where to find them
- [ ] Links to GitHub Discussions for questions
- [ ] Includes a placeholder line "See CODE_OF_CONDUCT.md" (link activated by US-0042)
- [ ] Includes a placeholder line "See SECURITY.md for reporting vulnerabilities" (link activated by US-0049)

## E2E Tests

Manual verification: reviewer checks that all sections are present and links resolve.
