---
id: US-0044
tags: [community, launch]
status: ready
depends_on: [US-0041]
---

# US-0044: Establish GitHub Releases changelog convention

**As a** user tracking Stratum updates  
**I want** structured release notes for every release  
**So that** I know what changed, what's fixed, and whether there are breaking changes

## Context

GitHub Releases is the changelog from day one. Depends on US-0041 so the template can be documented in CONTRIBUTING.md alongside the rest of the contributor guide.

## Acceptance Criteria

- [ ] A release notes template is documented in `CONTRIBUTING.md`
- [ ] Template sections: What's New, Bug Fixes, Breaking Changes, Upgrade Notes

## E2E Tests

Manual verification: confirm the template section is present in `CONTRIBUTING.md` and clearly describes the expected format.
