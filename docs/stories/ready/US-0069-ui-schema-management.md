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

Baut auf dem UI-Fundament aus US-0068 auf. Schema-Upload nutzt den bestehenden REST-Endpunkt (`POST /api/v1/schemas/{name}`). Die Seite liefert HTML-Fragmente via HTMX zurück — kein JSON, kein separater API-Call aus dem Browser.

Mehrere benannte Schemas können gleichzeitig existieren. Die Seite besteht aus zwei Bereichen: einer Schema-Liste (links) und einer kombinierten Detail-/Editieransicht (rechts).

### Editor

Der Editor verwendet CodeMirror 6 mit GraphQL-Syntax-Highlighting, eingebunden als statisches Asset analog zu htmx. Zusätzlich:

- **Formatter**: Ein "Formatieren"-Button ruft clientseitig `graphql.print(graphql.parse(sdl))` auf und ersetzt den Editor-Inhalt mit dem formatierten SDL. Das `graphql`-Paket wird als statisches Browser-Bundle eingebunden.
- **Linter**: Ein "Prüfen"-Button schickt das aktuelle SDL per HTMX-POST an `POST /api/v1/schemas/{name}?preview=true`. Validierungsfehler aus der `422`-Antwort (mit Zeile und Spalte) werden als CodeMirror-Lint-Marker inline im Editor angezeigt. Keine clientseitige Validierungslogik — die Servervalidierung ist autoritativ.

Da CodeMirror seinen Inhalt nicht in ein natives `<textarea>` zurückschreibt, liest ein kleines JS-Snippet den Editor-Inhalt vor dem HTMX-POST aus und befüllt ein verstecktes Input-Feld.

## Acceptance Criteria

- [ ] `GET /ui/schema` zeigt die Schema-Seite
- [ ] Die Seite listet alle vorhandenen Schemas (`GET /api/v1/schemas`) mit Name und Version — kein API-Key erforderlich
- [ ] Ist noch kein Schema vorhanden, wird ein entsprechender Hinweis angezeigt
- [ ] Ein Klick auf einen Schema-Eintrag füllt die Detailansicht: Name-Feld und Editor werden mit den bestehenden Werten befüllt
- [ ] Die Detailansicht enthält ein Textfeld für den Schema-Namen (URL-sicherer Bezeichner, Pattern `^[a-z][a-z0-9_-]*$`)
- [ ] Die Detailansicht enthält einen CodeMirror-6-Editor mit GraphQL-Syntax-Highlighting für das SDL
- [ ] Ein "Formatieren"-Button formatiert das SDL im Editor clientseitig via `graphql.print(parse(sdl))`; bei Syntaxfehler wird eine Fehlermeldung angezeigt ohne den Inhalt zu verändern
- [ ] Ein "Prüfen"-Button sendet das SDL an `POST /api/v1/schemas/{name}?preview=true`; Validierungsfehler werden als Lint-Marker im Editor angezeigt
- [ ] Ein "Hochladen"-Button schickt Name und SDL per HTMX-POST an `POST /api/v1/schemas/{name}`
- [ ] Der im Layout gespeicherte API-Key wird beim Upload und beim Prüfen als `X-API-Key`-Header mitgeschickt
- [ ] Ist kein API-Key gesetzt, wird vor dem Upload ein Hinweis angezeigt
- [ ] Erfolg und Fehler werden inline auf der Seite angezeigt (kein Page-Reload)
- [ ] Nach erfolgreichem Upload aktualisiert sich die Schema-Liste automatisch

## E2E Tests

- `e2e/ui_test.go` — `TestUISchemaUpload`
- `e2e/ui_test.go` — `TestUISchemaList`
- `e2e/ui_test.go` — `TestUISchemaLint`
