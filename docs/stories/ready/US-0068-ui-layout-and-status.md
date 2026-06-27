---
id: US-0068
tags: [ui, status, plugins]
status: ready
---

# US-0068: UI — Layout und Status-Seite

**As a** Stratum-Nutzer  
**I want** eine eingebettete Web-Oberfläche unter `/ui` aufrufen  
**So that** ich den Status und die aktiven Plugins auf einen Blick sehe, ohne die API direkt nutzen zu müssen

## Context

Das GUI wird vom Stratum-Server selbst ausgeliefert — kein separates Deployment. Alle Assets (Templates, CSS, HTMX) sind via `embed.FS` ins Binary eingebettet. Die UI besteht aus einem gemeinsamen Layout mit Sidebar-Navigation und drei Seiten: Status (diese Story), Schema (US-0069), Console (US-0070).

Technologie: Go `html/template`, HTMX (kein Build-Step), einfaches CSS.

Struktur:
```
internal/ui/
  handler.go
  embed.go
  templates/
    layout.html
    status.html
  static/
    htmx.min.js
    style.css
```

**Routing:** Der `UIHandler` wird am `StratumServer` als eigener Route-Block unter `/ui` montiert (`http.StripPrefix("/ui", uiHandler)`). Alle Unterseiten (`/ui/status`, `/ui/schema`, `/ui/console`) werden innerhalb des Handlers geroutet.

**localStorage und Sicherheit:** Der API-Key wird in `localStorage` gespeichert. Das ist für eine self-hosted Admin-UI ein bewusster Trade-off — einfache Handhabung hat Vorrang vor XSS-Härtung. Wer XSS-Angriffe auf die Admin-UI fürchtet, hat größere Probleme. Kein Handlungsbedarf.

## Acceptance Criteria

- [ ] `GET /ui` leitet auf `GET /ui/status` weiter (301)
- [ ] `GET /ui/status` liefert die Status-Seite
- [ ] Unbekannte Pfade unter `/ui/*` liefern HTTP 404
- [ ] Die Seite zeigt Sidebar-Navigation mit Links zu Status, Schema, Console
- [ ] Das Layout enthält ein API-Key-Eingabefeld (Sidebar oder Header), das für alle authentifizierten Operationen genutzt wird
- [ ] Der eingegebene API-Key wird im Browser gespeichert (localStorage) und bei Seitennavigation wiederhergestellt — manuelle Verifikation, kein E2E-Test
- [ ] Die Status-Seite zeigt den aktuellen Health-Status (liveness + readiness) — kein API-Key erforderlich
- [ ] Die Status-Seite listet alle registrierten Plugins mit Name und Typ — kein API-Key erforderlich
- [ ] Alle Assets sind via `embed.FS` ins Binary eingebettet
- [ ] Der `UIHandler` registriert sich am `StratumServer` ohne neue externe Dependencies
- [ ] HTMX wird als eingebettete statische Datei ausgeliefert (kein CDN)

## E2E Tests

- `e2e/ui_test.go` — `TestUIStatusPage`: ruft `/ui/status` auf und prüft Health-Status und Plugin-Liste im gerenderten HTML
