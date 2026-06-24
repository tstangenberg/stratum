---
id: US-0072
tags: [ci, dev-tooling]
status: open
depends_on: [US-0033]
---

# US-0072: golangci-lint — statische Analyse

**As a** contributor  
**I want** golangci-lint to run on every push and PR, and locally via `make lint`  
**So that** common bugs and code smells are caught automatically before they reach `main`

## Context

`go vet` (US-0033) deckt grundlegende Verdächtigkeiten ab. golangci-lint ergänzt dies mit einem Standard-Set an Linter-Regeln: unbehandelte Fehler, nutzlose Zuweisungen, tote Symbole und tiefere statische Analyse.

Das Makefile wird als kanonischer Einstiegspunkt für lokale Entwickler-Tasks eingeführt. Die Konvention ist in `CONTRIBUTING.md` dokumentiert.

## Acceptance Criteria

- [ ] `.golangci.yml` im Repo-Root aktiviert das Standard-Set: `errcheck`, `gosimple`, `govet`, `ineffassign`, `staticcheck`, `unused`
- [ ] Neuer Job `lint` in `.github/workflows/ci.yml` nutzt `golangci-lint-action@v6` mit gepinnter Version
- [ ] Der `lint`-Job läuft parallel zu `build` (kein `needs:`)
- [ ] `Makefile` im Repo-Root enthält ein `lint`-Target
- [ ] `make lint` prüft ob `golangci-lint` installiert ist; fehlt es, erscheint eine klare Fehlermeldung mit Installationsbefehl und Exit-Code ≠ 0
- [ ] `CONTRIBUTING.md` enthält einen Abschnitt "Local Development" mit `make lint` und Installationshinweisen (brew / install-script)
- [ ] Alle bestehenden Dateien passieren den Linter ohne Findings

## E2E Tests

None — verified by introducing a deliberate lint violation (e.g. unhandled error) and confirming CI fails, then fixing it and confirming CI passes.
