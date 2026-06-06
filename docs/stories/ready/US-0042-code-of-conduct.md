---
id: US-0042
tags: [community, launch]
status: ready
depends_on: [US-0041]
---

# US-0042: Add CODE_OF_CONDUCT.md

**As a** community member  
**I want** clear community standards  
**So that** I know what behaviour is expected and how violations are handled

## Context

Standard Contributor Covenant v2.1. Added at OSS launch. Depends on US-0041 so CONTRIBUTING.md exists to be updated with a CoC link.

## Acceptance Criteria

- [ ] `CODE_OF_CONDUCT.md` exists at the repo root
- [ ] Uses Contributor Covenant v2.1
- [ ] Contact email for reporting violations is set to `thorben@stangenberg.net`
- [ ] Links to `CONTRIBUTING.md`
- [ ] `CONTRIBUTING.md` is updated to link to `CODE_OF_CONDUCT.md`

## E2E Tests

Manual verification: check file contents, verify email address, confirm cross-links resolve in both directions.
