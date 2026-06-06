---
id: US-0040
tags: [community, launch]
status: ready
---

# US-0040: Enable GitHub Discussions with categories

**As a** potential user or contributor  
**I want** a structured place to ask questions and share ideas  
**So that** I can get help and participate in the Stratum community without needing a separate account

## Context

GitHub Discussions is enabled on the Stratum repo at OSS launch. It is the primary community hub. No additional infrastructure needed — users are already on GitHub.

## Acceptance Criteria

- [ ] GitHub Discussions is enabled on the Stratum repository
- [ ] The following categories are configured:

  | Category | Type |
  |---|---|
  | Announcements | Announcements (maintainer-only) |
  | Q&A | Q&A |
  | Plugin Development | General |
  | Show & Tell | General |
  | Ideas | General |

## E2E Tests

Manual verification: confirm in GitHub repo Settings → General (Features section) → Discussions is enabled, and check each category exists under the Discussions tab.
