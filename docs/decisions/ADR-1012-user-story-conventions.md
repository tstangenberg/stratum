# ADR-1012: User story conventions

**Status:** Accepted

## Context and Problem Statement

Stratum uses user stories to define features and acceptance criteria. Stories need a consistent format, naming scheme, and metadata so they can be discovered, filtered, and linked to E2E tests by contributors and maintainers alike.

## Considered Options

### Naming

| Option | Example | Pros | Cons |
|--------|---------|------|------|
| **Flat sequential** | `US-0001` | Simple, no category decisions | No grouping at a glance |
| Category prefix | `US-H01`, `US-D01` | Groups by domain | Edge cases — stories often span categories |
| Milestone prefix | `US-MVP1-001` | Clear scope | Verbose; prefix becomes noise within a milestone |

### Grouping

| Option | Pros | Cons |
|--------|------|------|
| **Frontmatter tags** | Flexible, multi-tag, filterable | Requires tooling awareness |
| Subdirectories | Filesystem-native | Forces single category; reorganisation is noisy |
| Flat, no grouping | Simplest | Unmanageable at scale |

## Decision Outcome

**Naming:** flat sequential `US-NNNN` (zero-padded to 4 digits). Simple, unambiguous, no upfront category taxonomy required. 4 digits supports up to 9,999 stories.

**Grouping:** frontmatter tags. A story may carry multiple tags. No fixed tag taxonomy — use whatever is descriptive (`health`, `schema`, `data`, `auth`, `observability`, etc.).

**Location:** `docs/stories/US-NNNN-<slug>.md`

**Format:** classic user story with acceptance criteria and linked E2E tests.

---

## Story Template

```markdown
---
id: US-NNNN
tags: [tag1, tag2]
status: draft
depends_on: []   # optional — omit if no dependencies
---

# US-NNNN: <Title>

**As a** <role>
**I want** <action>
**So that** <benefit>

## Context

<Optional: background, constraints, design references>

## Acceptance Criteria

- [ ] ...
- [ ] ...

## E2E Tests

- `e2e/<file>_test.go` — `<TestFunctionName>`
```

---

## Frontmatter Fields

| Field | Values | Required |
|-------|--------|----------|
| `id` | `US-NNNN` | Yes |
| `tags` | array of strings | Yes |
| `status` | see below | Yes |
| `depends_on` | array of story IDs, e.g. `[US-0028, US-0029]` | No |

### Story Status

| Status | Meaning | Set by |
|--------|---------|--------|
| `draft` | Acceptance criteria not finalized — not ready to pick up | Author |
| `blocked` | AC is complete but one or more dependencies are not yet `done` | Author / Automated (US-0031) |
| `ready` | Refined, clear AC, all dependencies done — safe to pick up | Author / Automated (US-0031) |
| `in-progress` | Being implemented — draft PR open | Automated (US-0026) |
| `in-review` | PR open and ready for review | Automated (US-0026) |
| `done` | Merged, E2E tests pass | Automated (US-0026) |
| `cancelled` | Will not be implemented | Author |

A story with `depends_on` starts as `blocked`. The CI bot (US-0031) promotes it to `ready` automatically once every listed dependency reaches `done`.

## Branch Naming Convention

Branches implementing a story must follow: `story/US-NNNN-<slug>`

Example: `story/US-0021-coverage-report`

Branches not tied to a story (`fix/`, `chore/`, etc.) are free-form. See US-0025 and US-0026 for the enforcement and automation details.

**Story files live only on `main`.** Feature branches contain only code. All status transitions are committed directly to `main` by the bot account, which holds a branch protection bypass for this purpose.

## Relationship to E2E Tests

Each story lists the E2E test(s) that verify it. Every E2E test maps to exactly one story. A story may be verified by more than one E2E test.
