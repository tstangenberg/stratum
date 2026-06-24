---
id: US-0069
tags: [ui, schema]
status: ready
---

# US-0069: UI — Schema-Verwaltung

**As a** Stratum-Nutzer  
**I want** mein GraphQL-Schema über die Web-Oberfläche hochladen und einsehen  
**So that** ich Stratum ohne API-Client ausprobieren kann

## Context

Baut auf dem UI-Fundament aus US-0068 auf. Schema-Upload nutzt den bestehenden REST-Endpunkt. Die Seite liefert HTML-Fragmente via HTMX zurück — kein JSON, kein separater API-Call aus dem Browser.

## Acceptance Criteria

- [ ] `GET /ui/schema` zeigt die Schema-Seite
- [ ] Die Seite zeigt das aktuell geladene Schema (SDL) in einem Read-only-Textbereich — kein API-Key erforderlich
- [ ] Ein Textarea erlaubt das Eingeben eines neuen SDL-Strings
- [ ] Ein "Hochladen"-Button schickt das Schema per HTMX-POST an den bestehenden Upload-Endpunkt
- [ ] Der im Layout gespeicherte API-Key wird beim Upload als `Authorization`-Header mitgeschickt
- [ ] Ist kein API-Key gesetzt, wird vor dem Upload ein Hinweis angezeigt
- [ ] Erfolg und Fehler werden inline auf der Seite angezeigt (kein Page-Reload)
- [ ] Ist kein Schema geladen, wird ein entsprechender Hinweis angezeigt

## E2E Tests

- `e2e/ui_test.go` — `TestUISchemaUpload`
