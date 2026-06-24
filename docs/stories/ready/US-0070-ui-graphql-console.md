---
id: US-0070
tags: [ui, graphql, console]
status: ready
---

# US-0070: UI — GraphQL Console

**As a** Stratum-Nutzer  
**I want** GraphQL-Queries direkt in der Web-Oberfläche abschicken und das Ergebnis sehen  
**So that** ich die API interaktiv erkunden kann ohne externen Client

## Context

Baut auf dem UI-Fundament aus US-0068 auf. Die Console schickt Queries an den bestehenden GraphQL-Endpunkt. Das Ergebnis wird als formatiertes JSON im Browser angezeigt. Kein GraphiQL — eine einfache Textarea + Response-Anzeige reicht für den MVP.

## Acceptance Criteria

- [ ] `GET /ui/console` zeigt die Console-Seite
- [ ] Eine Textarea erlaubt die Eingabe eines GraphQL-Queries
- [ ] Ein "Ausführen"-Button schickt den Query per HTMX-POST an `/graphql`
- [ ] Das JSON-Ergebnis wird formatiert (pretty-printed) unterhalb angezeigt
- [ ] Fehler (GraphQL-Errors, HTTP-Fehler) werden inline angezeigt
- [ ] Der API-Key kann optional in einem Eingabefeld angegeben werden und wird als `Authorization`-Header mitgeschickt

## E2E Tests

- `e2e/ui_test.go` — `TestUIGraphQLConsole`
