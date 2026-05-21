---
id: US-0032
tags: [ci, dev-tooling, legal]
status: in-review
depends_on: [US-0033]
---

# US-0032: SPDX license header check with addlicense

**As a** contributor  
**I want** every Go source file to carry an SPDX license identifier  
**So that** the AGPL-3.0-or-later license is unambiguous when files are distributed in isolation

## Context

AGPL v3 recommends per-file license notices. `addlicense` (github.com/google/addlicense) adds and checks SPDX identifiers in a single command, integrates with Go tooling via `tools.go`, and supports ignore patterns for generated files.

The chosen identifier is `AGPL-3.0-or-later`. Generated files (`*.gen.go`) carry their own headers from the generator and are excluded.

## Acceptance Criteria

- [ ] `addlicense` is pinned in `tools.go` and appears in `go.mod`
- [ ] Every hand-written `.go` file contains `// SPDX-License-Identifier: AGPL-3.0-or-later` as the first line
- [ ] Generated files (`**/*.gen.go`) are excluded from the check
- [ ] CI runs `go run github.com/google/addlicense -check -l agpl -c "Thorben Stangenberg" -s=only -ignore "**/*.gen.go" ./cmd ./tools ./internal` and fails if any file is missing a header
- [ ] The failure message tells the contributor which files are missing headers and what command to run to fix them
- [ ] Running `go run github.com/google/addlicense -l agpl -c "Thorben Stangenberg" -s=only -ignore "**/*.gen.go" ./cmd ./tools ./internal` locally adds missing headers without modifying existing ones
- [ ] The scanned directories list (`./cmd ./tools ./internal`) is extended whenever a new top-level Go source directory is added

## E2E Tests

None — verified by removing a header from a `.go` file and confirming CI fails, then running the add command and confirming CI passes.
