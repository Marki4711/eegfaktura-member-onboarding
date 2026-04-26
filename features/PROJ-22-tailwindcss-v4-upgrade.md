# PROJ-22: Tailwind CSS v3 → v4 Upgrade

## Status: Planned
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

## Dependencies
- Requires: stabiles Zeitfenster ohne parallele Frontend-Features

## Hintergrund

Das Projekt verwendet derzeit Tailwind CSS v3 (`^3.4.1`). Tailwind CSS v4 ist ein Breaking-Major-Release mit grundlegend geänderter Konfiguration: Die JavaScript-basierte `tailwind.config.js` wird durch eine CSS-native Konfiguration (`@import "tailwindcss"` + CSS-Variablen) ersetzt. Die meisten Utility-Klassen bleiben kompatibel, aber die Konfiguration und Build-Integration müssen migriert werden.

**Warum jetzt?** Der Dependabot-PR für v4 wurde bewusst geschlossen (kein Auto-Merge), da ein Major-Upgrade ein separates Feature-Ticket und manuelle Migration erfordert.

## User Stories

- Als **Entwickler** möchte ich Tailwind CSS v4 verwenden, damit ich von den Performance-Verbesserungen und der CSS-nativen Konfiguration profitiere.
- Als **Entwickler** möchte ich, dass alle bestehenden UI-Komponenten nach dem Upgrade unverändert aussehen, damit kein visueller Regressionsschaden entsteht.

## Acceptance Criteria

### Migration
- [ ] `tailwindcss` in `package.json` auf v4 aktualisiert
- [ ] `tailwind.config.js` / `tailwind.config.ts` entfernt oder auf das neue v4-Format migriert
- [ ] CSS-Entrypoint (`globals.css` o.ä.) auf `@import "tailwindcss"` umgestellt
- [ ] PostCSS-Konfiguration (`postcss.config.js`) aktualisiert (v4 ändert den Plugin-Namen)
- [ ] Alle `@apply`-Direktiven geprüft und ggf. angepasst (v4 schränkt `@apply` ein)
- [ ] Alle benutzerdefinierten Theme-Werte (Farben, Abstände etc.) in CSS-Variablen übertragen

### Qualitätssicherung
- [ ] `npm run build` schlägt nicht fehl
- [ ] `npm run dev` startet ohne Fehler
- [ ] Visuelle Überprüfung der wichtigsten Seiten: Registrierungsformular, Admin-Listenansicht, Admin-Detailansicht
- [ ] Kein sichtbarer Layout-Bruch oder Farbabweichung gegenüber v3

### Dokumentation
- [ ] `docs/security.md` — offene CVE-Einträge für next-auth/uuid (waren mit Tailwind-Version verlinkt) bleiben unberührt

## Edge Cases

- **`@apply` mit entfernten Utilities:** v4 entfernt einige Utilities oder ändert deren Namen → Linting-Fehler müssen vor Build behoben werden
- **Custom Plugins:** Falls Drittanbieter-Plugins verwendet werden, müssen diese v4-kompatibel sein
- **Dark Mode:** Konfiguration ändert sich in v4 (CSS-Variable statt `class`-Toggle) → explizit testen
- **`tailwind-merge` / `clsx`:** Diese Libraries sind unabhängig von Tailwind und sollten kompatibel bleiben

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
