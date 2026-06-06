---
id: US-0048
tags: [community, launch]
status: ready
---

# US-0048: Add pull request template

**As a** contributor submitting a PR  
**I want** a checklist that reminds me what to include  
**So that** my PR is easier to review and less likely to be sent back for missing information

## Context

Stored at `.github/pull_request_template.md` (lowercase — GitHub's canonical name, case-sensitive on Linux runners). Shown automatically when a contributor opens a PR.

## Acceptance Criteria

- [ ] `.github/pull_request_template.md` exists (lowercase filename)
- [ ] Template includes: summary of change, type of change (bug fix / feature / docs / refactor), how to test, checklist (tests added, docs updated, ADR written if needed)

## E2E Tests

Manual verification: open a draft PR and confirm the template body is pre-populated.
