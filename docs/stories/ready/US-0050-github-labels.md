---
id: US-0050
tags: [community, launch]
status: ready
---

# US-0050: Set up GitHub labels

**As a** first-time contributor  
**I want** to find issues labelled "good first issue"  
**So that** I can contribute without needing to understand the entire codebase

## Context

GitHub's default labels are generic. A curated label set helps contributors find entry points and helps maintainers triage issues efficiently. The `dependencies` label must be preserved — Dependabot applies it automatically to its PRs.

## Acceptance Criteria

- [ ] The following labels exist on the repository with the specified colours:

  | Label | Colour | Purpose |
  |---|---|---|
  | `good first issue` | #7057ff | Entry point for new contributors |
  | `help wanted` | #008672 | Maintainer is actively seeking contributors |
  | `bug` | #d73a4a | Something is broken |
  | `enhancement` | #a2eeef | New feature or improvement |
  | `documentation` | #0075ca | Docs-only change |
  | `plugin` | #e4e669 | Relates to the plugin system |
  | `breaking change` | #b60205 | Introduces a breaking change |
  | `dependencies` | #0075ca | Applied by Dependabot (do not remove) |

- [ ] The following default GitHub labels are deleted: `duplicate`, `invalid`, `question`, `wontfix`
- [ ] GitHub's default `good first issue` and `help wanted` labels are updated in-place (edit colour and description) to match the curated values above — do not delete and recreate, to preserve any existing label references

## E2E Tests

Manual verification: open the Labels page and confirm all curated labels are present with correct colours, and the removed defaults no longer appear.
