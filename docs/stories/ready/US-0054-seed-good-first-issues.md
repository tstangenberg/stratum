---
id: US-0054
tags: [community, launch]
status: ready
depends_on: [US-0047, US-0050]
---

# US-0054: Seed "good first issue" issues

**As a** first-time contributor  
**I want** to find concrete, well-scoped tasks I can tackle immediately  
**So that** I can make a meaningful contribution without needing deep knowledge of the codebase

## Context

The `good first issue` label (US-0050) and issue templates (US-0047) are only useful if real issues carry them. Without seeded issues, first-time contributors land on an empty label and have nowhere to start. Depends on US-0047 and US-0050 so templates and labels exist before issues are created.

## Acceptance Criteria

- [ ] At least 5 issues exist with the `good first issue` label
- [ ] Each issue has a clear problem statement, acceptance criteria, and a pointer to the relevant code area
- [ ] Issues are scoped to tasks completable without understanding the full codebase (docs improvements, new scalar plugin, new filter plugin, test coverage, example schema)
- [ ] Each issue uses the appropriate issue template
- [ ] Issues are assigned to the "Ideas" or "Backlog" column on the public roadmap (US-0043)

## E2E Tests

Manual verification: open the Issues tab filtered by `good first issue` and confirm at least 5 well-described issues appear.
