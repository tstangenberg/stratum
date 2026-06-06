---
id: US-0038
tags: [dev-tooling, workflow, docs]
status: done
depends_on: [US-0026, US-0031]
---

# US-0038: Story folder structure

**As a** contributor  
**I want** story status to be visible from the filesystem  
**So that** I can see at a glance what is actionable, in progress, done, or archived without reading frontmatter

## Context

Story files currently live flat in `docs/stories/` with status tracked via a `status` field in frontmatter. This is invisible in the file tree and was the root cause of merge conflicts in the story status workflow (US-0026): the CI bot patching frontmatter on `main` conflicted with feature branches that inherited the same file.

The folder structure solves both problems: status is visible in the filesystem, and moving a file on `main` does not conflict with a branch that has the file unchanged in its original location.

The `status` field in frontmatter is kept as-is — it remains the machine-readable field used by tooling (US-0031 dependency promotion, US-0039 issue sync). The folder is the human-facing indicator; frontmatter is the structured data layer.

The automated CI triggers from US-0026 that patch `status` in frontmatter and commit to `main` during the PR lifecycle are removed. Status transitions become manual: a maintainer moves the file to the appropriate folder and updates the frontmatter field when the state changes.

The four folders:

| Folder | Meaning |
|---|---|
| `open/` | Draft, or `status: ready` but waiting on unmet dependencies |
| `ready/` | `status: ready` and all dependencies met — actionable now |
| `done/` | Merged and shipped |
| `archive/` | Rejected, won't-do, superseded, or indefinitely deferred |

`in-progress` and `in-review` states are not tracked in the filesystem — they are visible from the open PR itself.

A story moves from `open/` to `ready/` automatically: when a story is moved to `done/`, a script scans all stories in `open/` whose `depends_on` list is now fully satisfied (all referenced stories are in `done/`) and `git mv`s them to `ready/`. This replaces the frontmatter-patching behaviour from US-0031.

## Acceptance Criteria

- [x] `docs/stories/open/`, `docs/stories/ready/`, `docs/stories/done/`, and `docs/stories/archive/` directories exist
- [x] All existing story files are moved to the appropriate folder: `status: done` → `done/`; `status: ready` with all deps done → `ready/`; `status: ready` with unmet deps or draft → `open/`
- [x] The `status` field in frontmatter is preserved unchanged in all story files
- [x] The GitHub Actions workflow from US-0026 (PR lifecycle → frontmatter status patch) is removed
- [x] When a story is moved to `done/`, a script scans all stories in `open/` and `git mv`s those with `status: ready` and fully satisfied `depends_on` to `ready/`
- [x] The script commits to `main` with message `bot: US-NNNN unblocked → ready/`
- [x] Stories in `open/` with no `depends_on` are not moved by the script (draft — manual decision)
- [x] The frontmatter-patching workflow from US-0031 is removed; this script replaces it
- [x] Archive is a manual operation — no CI trigger moves files to `archive/`
- [x] CONTRIBUTING.md documents the folder convention and status values

## E2E Tests

None — verified by opening and merging a PR on a `story/US-NNNN-*` branch and confirming the story file moves from `ready/` to `done/` on `main`.
