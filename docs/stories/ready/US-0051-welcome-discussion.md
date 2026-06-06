---
id: US-0051
tags: [community, launch]
status: ready
depends_on: [US-0040, US-0041]
---

# US-0051: Create pinned welcome Discussion

**As a** visitor discovering Stratum's community  
**I want** to see a welcoming introduction when I open Discussions  
**So that** I immediately understand what Stratum is and how to get involved

## Context

A pinned Announcements post created when Discussions goes live. Makes an otherwise empty Discussions feel intentional. Done manually by the maintainer, not via automation.

## Acceptance Criteria

- [ ] A post exists in the Announcements category titled "Welcome to the Stratum community"
- [ ] Post covers: what Stratum is (one paragraph), how to get help (Q&A category), how to contribute (link to CONTRIBUTING.md), how to share what you've built (Show & Tell category)
- [ ] Post is pinned to the top of Discussions
- [ ] Post is written by the maintainer account (not a bot)

## E2E Tests

Manual verification: open Discussions as an unauthenticated user and confirm the welcome post is pinned and visible.
