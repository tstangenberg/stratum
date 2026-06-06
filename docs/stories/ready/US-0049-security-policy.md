---
id: US-0049
tags: [community, launch]
status: ready
depends_on: [US-0044]
---

# US-0049: Add SECURITY.md

**As a** security researcher or user who found a vulnerability  
**I want** a clear process for reporting it privately  
**So that** it can be fixed before being disclosed publicly

## Context

GitHub automatically surfaces SECURITY.md as a "Security policy" on the repository's Security tab. Especially important for a data middleware tool that handles persistence and authentication. No README link needed — GitHub handles discoverability. This story also activates the SECURITY.md placeholder in CONTRIBUTING.md (written by US-0041). Depends on US-0044, the last story in the CONTRIBUTING.md edit chain (US-0041 → US-0042 → US-0044 → US-0049).

## Acceptance Criteria

- [ ] `SECURITY.md` exists at the repo root
- [ ] States the current support posture: only the latest release / main branch receives security fixes
- [ ] Provides a private reporting method (GitHub private vulnerability reporting or email to `thorben@stangenberg.net`)
- [ ] States the expected response time: acknowledgement within 48 hours, fix timeline communicated within 7 days
- [ ] GitHub private vulnerability reporting is enabled on the repository
- [ ] `CONTRIBUTING.md` placeholder "See SECURITY.md" is updated to a working link to `SECURITY.md`

## E2E Tests

Manual verification: confirm the Security tab shows the policy, the "Report a vulnerability" button is available, and the SECURITY.md link in CONTRIBUTING.md resolves correctly.
