---
id: US-0047
tags: [community, launch]
status: ready
depends_on: [US-0050]
---

# US-0047: Add GitHub issue templates

**As a** user who found a bug or wants to request a feature  
**I want** a structured form to fill in when opening an issue  
**So that** I provide enough context for the maintainer to act without back-and-forth

## Context

Issue templates are stored in `.github/ISSUE_TEMPLATE/`. Two templates: bug report and feature request. Label pre-assignment requires a `labels:` key in the YAML front matter of each template file. Blank issue suppression requires a separate `config.yml` in the same directory. Depends on US-0050 so the `bug` and `enhancement` labels exist before templates reference them.

## Acceptance Criteria

- [ ] `.github/ISSUE_TEMPLATE/bug_report.md` exists with YAML front matter (`name`, `about`, `labels: [bug]`) and fields: description, steps to reproduce, expected vs actual behaviour, Stratum version, environment
- [ ] `.github/ISSUE_TEMPLATE/feature_request.md` exists with YAML front matter (`name`, `about`, `labels: [enhancement]`) and fields: problem statement, proposed solution, alternatives considered
- [ ] `.github/ISSUE_TEMPLATE/config.yml` exists with `blank_issues_enabled: false`

## E2E Tests

Manual verification: open a new issue on the repo and confirm both templates appear as options, no blank issue option is shown, and issues filed via each template receive the correct label automatically.
