---
id: US-0050
tags: [community, launch]
status: done
---

# US-0050: Set up GitHub labels

**As a** first-time contributor  
**I want** to find issues labelled "good first issue"  
**So that** I can contribute without needing to understand the entire codebase

## Context

GitHub's default labels are generic. A curated label set helps contributors find entry points and helps maintainers triage issues efficiently. The `dependencies` label must be preserved — Dependabot applies it automatically to its PRs.

## Acceptance Criteria

- [x] The following labels exist on the repository with the specified colours:

  | Label | Colour | Purpose |
  |---|---|---|
  | `bug` | #d73a4a | Something is broken |
  | `enhancement` | #a2eeef | New feature or improvement |
  | `documentation` | #0075ca | Docs-only change |
  | `question` | #d876e3 | Open question or needs clarification |
  | `security` | #ee0701 | Vulnerability or security concern |
  | `performance` | #d4c5f9 | Not broken, but too slow |
  | `refactor` | #bfd4f2 | Internal cleanup, no user-visible change |
  | `ci` | #006b75 | GitHub Actions / pipeline issue |
  | `needs reproduction` | #fbca04 | Bug reported but not yet confirmed — waiting for a repro case |
  | `needs discussion` | #c5def5 | Design decision required before work can start |
  | `blocked` | #e99695 | Cannot progress — waiting on something external |
  | `plugin` | #e4e669 | Relates to the plugin system |
  | `breaking change` | #b60205 | Merging this will break existing behavior for users |
  | `dependencies` | #0075ca | Applied by Dependabot (do not remove) |
  | `good first issue` | #7057ff | Small, well-scoped, low risk — ideal entry point |
  | `help wanted` | #008672 | Maintainer won't pick this up soon; contributions welcome |
  | `wontfix` | #ffffff | Acknowledged but deliberately not addressed |
  | `duplicate` | #cfd3d7 | Same issue reported elsewhere — link to the original |
  | `invalid` | #e4e4e4 | Not a valid issue — wrong repo, spam, or not reproducible |

- [x] GitHub's default `good first issue`, `help wanted`, `bug`, `enhancement`, `documentation`, `duplicate`, `question`, `invalid`, and `wontfix` labels are updated in-place (edit colour and description) to match the curated values above — do not delete and recreate, to preserve any existing label references

## When to use

### Type — what kind of issue is it

| Label | Use when |
|---|---|
| `bug` | Something is broken or behaves incorrectly |
| `enhancement` | New feature or improvement to existing behavior |
| `documentation` | Docs-only — no code change |
| `question` | An open question needs answering before work can proceed |
| `security` | A vulnerability or security concern — users should be able to filter for these |
| `performance` | Nothing is broken, but it's too slow |
| `refactor` | Internal cleanup with no user-visible change; no spec to follow |
| `ci` | GitHub Actions or pipeline issue unrelated to dependencies |

### Status — where is this issue stuck

| Label | Use when |
|---|---|
| `needs reproduction` | A bug was reported but you cannot confirm it yet — ask for a repro case |
| `needs discussion` | The solution is unclear; a design decision is required before anyone codes |
| `blocked` | The issue cannot progress — waiting on another issue, a decision, or an external dependency |

### Scope — what area does it touch

| Label | Use when |
|---|---|
| `plugin` | The issue is specific to the plugin system |
| `breaking change` | Merging the fix or feature will break existing user behavior |
| `dependencies` | Applied automatically by Dependabot — do not add or remove manually |

### Contributor signals — help people find work

| Label | Use when |
|---|---|
| `good first issue` | The issue is small, well-scoped, and low-risk — ideal for a first contribution |
| `help wanted` | The maintainer won't pick this up soon; outside contributions are actively welcome |

### Resolution — how was the issue closed

| Label | Use when |
|---|---|
| `wontfix` | The issue is acknowledged but deliberately not addressed |
| `duplicate` | The same issue was reported elsewhere — link to the original before closing |
| `invalid` | The issue is not valid — wrong repo, spam, or cannot be reproduced after follow-up |

- [x] The "when to use" label guide is added to `CONTRIBUTING.md`

## E2E Tests

Manual verification: open the Labels page and confirm all curated labels are present with correct colours, and the removed defaults no longer appear.
