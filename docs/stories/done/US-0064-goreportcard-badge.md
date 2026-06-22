---
id: US-0064
tags: [docs]
status: done
---

# US-0064: Go Report Card badge in README

**As a** developer evaluating Stratum  
**I want** a Go Report Card badge in the README  
**So that** I can quickly see the project's code quality score without leaving GitHub

## Context

Go Report Card (goreportcard.com) analyses Go projects for formatting, linting, and vet issues and gives a public grade. Adding its badge to the README signals code quality at a glance, consistent with the existing CI and Codecov badges.

Badge URL: `https://goreportcard.com/badge/github.com/tstangenberg/stratum`  
Report URL: `https://goreportcard.com/report/github.com/tstangenberg/stratum`

## Acceptance Criteria

- [x] The Go Report Card badge is added to the badge row in `README.md`
- [x] The badge links to `https://goreportcard.com/report/github.com/tstangenberg/stratum`
- [x] The badge renders correctly on GitHub (verified by inspecting the rendered README)

## E2E Tests

None — verified by reading the rendered README on GitHub and confirming the badge resolves correctly.
