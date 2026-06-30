---
id: US-0073
tags: [ci, dev-tooling]
status: ready
depends_on: [US-0033]
---

# US-0073: SonarCloud — Code-Quality-Dashboard

**As a** maintainer  
**I want** SonarCloud to analyse every push and PR and display results on a dashboard  
**So that** code quality, code smells, duplications and security hotspots are continuously visible

## Context

Codecov (US-0021/US-0022) deckt Test-Coverage ab. SonarCloud ergänzt dies mit einer breiteren Qualitätssicht: Bugs, Code Smells, Duplications und Security Hotspots auf einem zentralen Dashboard. Kein Quality Gate — SonarCloud ist rein informativ und blockiert keine PRs.

Coverage wird als Artifact zwischen den Jobs weitergegeben, damit SonarCloud dieselbe `coverage.out` nutzt wie Codecov und keine Tests doppelt laufen.

## Acceptance Criteria

- [ ] `sonar-project.properties` im Repo-Root mit `sonar.projectKey`, `sonar.organization`, Go-Source-Pfaden und `sonar.go.coverage.reportPaths=coverage.out`
- [ ] Der `build`-Job lädt `coverage.out` als GitHub Actions Artifact hoch (nach dem Codecov-Upload)
- [ ] Neuer Job `sonarcloud` in `.github/workflows/ci.yml` mit `needs: [build]`
- [ ] `sonarcloud`-Job lädt `coverage.out` als Artifact herunter und führt `SonarSource/sonarqube-scan-action` aus (gepinnte Version)
- [ ] `SONAR_TOKEN` ist als Secret im GitHub-Repo hinterlegt
- [ ] Der Job läuft mit `continue-on-error: true` — ein SonarCloud-Ausfall blockiert keine PRs
- [ ] SonarCloud-Badge (Quality Gate Status) ist in `README.md` eingebunden

## E2E Tests

None — verified by pushing a commit and confirming the SonarCloud dashboard shows analysis results and the badge appears in the README.
