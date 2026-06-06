---
id: US-0044
tags: [community, launch]
status: ready
depends_on: [US-0042]
---

# US-0044: Establish GitHub Releases changelog convention

**As a** user tracking Stratum updates  
**I want** structured release notes for every release  
**So that** I know what changed, what's fixed, and whether there are breaking changes

## Context

GitHub Releases is the changelog from day one. Depends on US-0042 (which is the last story to edit CONTRIBUTING.md before this one) to avoid concurrent edits to that file. The full CONTRIBUTING.md edit chain is: US-0041 → US-0042 → US-0044 → US-0049.

## Acceptance Criteria

- [ ] A release notes template is documented in `CONTRIBUTING.md` under a "Releases" section
- [ ] Template sections: What's New, Bug Fixes, Breaking Changes, Upgrade Notes
- [ ] Template includes a filled-in example entry so the format is unambiguous
- [ ] GitHub Releases tab is accessible on the repository (verify by visiting `github.com/tstangenberg/stratum/releases`)

## E2E Tests

Manual verification: confirm the template section is present in `CONTRIBUTING.md` with a concrete example entry, and that the Releases tab is reachable.
