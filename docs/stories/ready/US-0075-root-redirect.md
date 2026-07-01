---
id: US-0075
tags: [ui, dx]
status: ready
---

# US-0075: Weiterleitung von `/` auf `/ui`

**As a** Stratum-Nutzer  
**I want** dass ein Aufruf der Root-URL automatisch zur Web-Oberfläche weiterleitet  
**So that** ich nicht die genaue UI-URL kennen muss

## Context

Aktuell liefert `GET /` eine leere Antwort oder einen 404. Da `/ui` bereits auf `/ui/status` weiterleitet, reicht eine einzige zusätzliche Weiterleitung von `/` auf `/ui`.

## Acceptance Criteria

- [ ] `GET /` antwortet mit einem permanenten Redirect (301) auf `/ui`

## E2E Tests

- `e2e/server_test.go` — `TestRootRedirect`
