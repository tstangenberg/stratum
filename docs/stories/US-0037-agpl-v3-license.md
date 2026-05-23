---
id: US-0037
tags: [legal, docs]
status: in-progress
---

# US-0037: AGPL v3 License File

## Context

The repository uses AGPL-3.0-or-later declared via SPDX headers in every Go source file, but no `LICENSE` file exists at the repo root. Without it, GitHub cannot auto-detect the license, legal clarity is missing for contributors and users, and tools that scan for license files will flag the repo.

## Acceptance Criteria

- [ ] A `LICENSE` file exists at the repo root containing the full AGPL-3.0 text
- [ ] Copyright notice reads: `Copyright (C) 2026 Thorben Stangenberg`
- [ ] GitHub license detection correctly identifies the repo as AGPL-3.0

## E2E Tests

None — verified by GitHub UI and `gh repo view --json licenseInfo`.
