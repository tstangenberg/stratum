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
| Story file added to `open/` | Create GitHub Issue; write issue number into story frontmatter |
| Story file moved to `done/` | Close GitHub Issue |
| Story file moved to `archive/` | Close GitHub Issue with label `won't fix` |
| PR opened for `story/US-NNNN-*` | Add label `in progress` to linked issue |
| PR merged | Issue closed automatically via `Closes #N` in PR description |

### Issue body: story link

Every GitHub Issue — whether created by the sync workflow or by the community — contains a `Story` field in a structured footer:

```
**Story:** [US-0038](link to story file in repo)
```

If no story file exists yet (community-created issue, not yet triaged), the field is present but empty:

```
**Story:** —
```

When a maintainer creates a story file and links it via `issue: <N>`, the sync workflow updates the issue footer with the story link.

### Community-created issues

External contributors open a GitHub Issue → maintainer triages → if accepted, maintainer creates a story file in `open/` with `issue: <N>` → the sync workflow updates the existing issue with the structured story content (title, ACs), applies the `story` label, and fills in the `Story` link in the footer.

Fully automated issue → story file creation is out of scope: community issues lack story structure and require human acceptance.

## Acceptance Criteria

- [ ] A GitHub Actions workflow triggers when a story file is added to `open/` (push to `main`): creates a GitHub Issue with the story title and body (Context + ACs rendered as Markdown), writes the issue number back into the story frontmatter, and includes a `Story` footer with a link to the story file
- [ ] Community-created issues have a `Story: —` footer added by an issue template; when a story file is linked via `issue: <N>`, the workflow updates the footer with the story link
- [ ] When a story file is moved to `done/`, the linked issue is closed with reason `completed`
- [ ] When a story file is moved to `archive/`, the linked issue is closed with label `won't fix`
- [ ] When a PR is opened for a `story/US-NNNN-*` branch, the linked issue (looked up via `issue` field in story frontmatter on `main`) receives the label `in progress`
- [ ] Stories without an `issue` field are silently skipped by all triggers
- [ ] The triage flow is documented in CONTRIBUTING.md: how to link an existing community issue to a new story file
- [ ] The `issue` field is added to the story file template

## E2E Tests

None — verified manually by adding a story to `open/` and confirming the GitHub Issue appears with correct content and labels.
