---
id: US-0039
tags: [dev-tooling, workflow, community]
status: ready
depends_on: [US-0038]
---

# US-0039: GitHub Issues sync

**As a** maintainer  
**I want** story files to be mirrored as GitHub Issues  
**So that** the community can see what is planned, comment, and submit new requests

## Context

Story files are the internal source of truth (structured ACs, context, technical detail). GitHub Issues are the public-facing representation — discoverable, commentable, linkable from PRs. A trigger system keeps the two in sync automatically.

Story files are authoritative for content. GitHub Issues are authoritative for community interaction (comments, reactions, additional requests). The sync is one-way for content (file → issue) and manual for triage (issue → file).

An `issue` field in story frontmatter links the two:

```yaml
---
id: US-0038
issue: 42
---
```

### Trigger map

| Event | Action |
|---|---|
| Story file added to `open/` or `ready/` without `issue` field | Create GitHub Issue; write issue number into story frontmatter |
| Story file moved to `done/` | Close GitHub Issue |
| Story file moved to `archive/` | Close GitHub Issue with label `won't fix` |
| Branch `story/US-NNNN-*` created | Add label `in-progress` to linked issue |
| PR opened for `story/US-NNNN-*` | Replace label `in-progress` with `in-review` on linked issue |
| PR merged | Issue closed automatically via `Closes #N` in PR description |

### Issue body: story link and checksum

Every GitHub Issue created by the sync workflow contains a structured footer:

```
**Story:** [US-0038](https://github.com/.../docs/stories/ready/US-0038-story-folder-structure.md)
**Checksum:** `a3f8c2d1`
```

Community-created issues (no story file yet) get a footer with empty fields via issue template:

```
**Story:** —
**Checksum:** —
```

### Bidirectional content sync via checksum

A checksum of the story content is stored in two places: the story frontmatter (`checksum: a3f8c2d1`) and the issue footer. When either side changes, the checksums diverge and the sync workflow updates the other side.

| Direction | Trigger | Action |
|---|---|---|
| Story → Issue | Push to `main` | Recompute checksum; if it differs from issue footer → update issue body + footer checksum |
| Issue → Story | `issues: edited` event | Extract checksum from issue footer; if it differs from story frontmatter → update story file + frontmatter checksum, commit to `main` |

The checksum prevents sync loops: after a sync both sides have the same checksum, so the next trigger fires but takes no action.

### Community-created issues

External contributors open a GitHub Issue → maintainer triages → if accepted, maintainer creates a story file in `open/` with `issue: <N>` → the sync workflow updates the existing issue with the structured story content (title, ACs), applies the `story` label, and fills in the `Story` link in the footer.

Fully automated issue → story file creation is out of scope: community issues lack story structure and require human acceptance.

## Acceptance Criteria

- [ ] A GitHub Actions workflow triggers on every push to `main`: any story file in `open/` or `ready/` without an `issue` field gets a GitHub Issue created (title + Context + ACs as Markdown body + structured footer with story link and checksum); the issue number and checksum are written back into the story frontmatter
- [ ] A one-time backfill script creates issues for all existing stories in `open/` and `ready/` that have no `issue` field
- [ ] On push to `main`, for stories that already have an `issue` field: recompute checksum; if it differs from the checksum in the issue footer → update issue body and footer checksum
- [ ] On `issues: edited` event: extract checksum from issue footer; if it differs from the story frontmatter checksum → update story file content and frontmatter checksum, commit to `main` with message `sync: US-NNNN story updated from issue`
- [ ] Checksum is computed over the story body (excluding frontmatter) so frontmatter-only changes (e.g. `issue`, `checksum` fields) do not trigger content sync
- [ ] Community-created issues have a `Story: — / Checksum: —` footer added by issue template; when linked via `issue: <N>`, the workflow fills in the story link and checksum
- [ ] When a story file is moved to `done/`, the linked issue is closed with reason `completed`
- [ ] When a story file is moved to `archive/`, the linked issue is closed with label `won't fix`
- [ ] When a branch `story/US-NNNN-*` is created, the linked issue receives the label `in-progress`
- [ ] When a PR is opened for a `story/US-NNNN-*` branch, the label `in-progress` is replaced with `in-review` on the linked issue
- [ ] Stories in `done/` or `archive/` without an `issue` field are silently skipped
- [ ] The triage flow is documented in CONTRIBUTING.md: how to link an existing community issue to a new story file
- [ ] The `issue` field is added to the story file template

## E2E Tests

None — verified manually by adding a story to `open/` and confirming the GitHub Issue appears with correct content and labels.
