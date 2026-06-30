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

Baut auf dem UI-Fundament aus US-0068 auf. Die Console schickt Queries an den bestehenden GraphQL-Endpunkt (`POST /graphql/{name}`). Das Ergebnis wird als formatiertes JSON im Browser angezeigt. Kein GraphiQL — eine einfache Textarea + Response-Anzeige reicht für den MVP.

Da der Endpunkt einen JSON-Body (`{"query": "..."}`) mit `Content-Type: application/json` erwartet, wird der Request clientseitig per `fetch()` abgesetzt (kein HTMX). Das Ergebnis wird per `JSON.stringify(data, null, 2)` ins DOM geschrieben.

## Acceptance Criteria

- [ ] `GET /ui/console` zeigt die Console-Seite
- [ ] Ein Dropdown oder Auswahlfeld zeigt alle vorhandenen Schemas (geladen von `GET /api/v1/schemas`); der Nutzer wählt das Ziel-Schema aus
- [ ] Eine Textarea erlaubt die Eingabe eines GraphQL-Queries
- [ ] Ein "Ausführen"-Button schickt den Query per `fetch()` als JSON-POST an `POST /graphql/{name}` mit dem gewählten Schema-Namen
- [ ] Das JSON-Ergebnis wird formatiert (pretty-printed) unterhalb angezeigt
- [ ] Fehler (GraphQL-Errors, HTTP-Fehler) werden inline angezeigt
- [ ] Der im Layout gespeicherte API-Key wird als `X-API-Key`-Header mitgeschickt

## E2E Tests

- `e2e/ui_test.go` — `TestUIGraphQLConsole`
