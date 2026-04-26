# PROJ-22: Tailwind CSS v3 → v4 Upgrade

## Status: In Review
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
- [x] `tailwindcss` in `package.json` auf v4 aktualisiert (v4.2.4)
- [x] `tailwind.config.ts` entfernt — Theme in `@theme`-Block in `globals.css` übertragen
- [x] CSS-Entrypoint `globals.css` auf `@import 'tailwindcss'` umgestellt
- [x] PostCSS-Konfiguration auf `@tailwindcss/postcss` aktualisiert, `autoprefixer` entfernt (in v4 eingebaut)
- [x] Alle `@apply`-Direktiven geprüft — keine vorhanden
- [x] Alle Theme-Werte (Farben, borderRadius, Animationen) in `@theme`-Block in globals.css übertragen
- [x] Dark-Mode-Konfiguration migriert: `darkMode: ["class"]` → `@custom-variant dark (&:is(.dark *))`
- [x] 28 Komponenten-Dateien automatisch migriert (u.a. `focus-visible:outline-none` → `focus-visible:outline-hidden`)
- [x] Bug in `pagination.tsx` behoben: `outline-solid` (ungültige Variante nach Migration) → `outline`

### Qualitätssicherung
- [x] `npm run build` erfolgreich
- [ ] `npm run dev` startet ohne Fehler — manuell zu prüfen
- [ ] Visuelle Überprüfung: Registrierungsformular, Admin-Listenansicht, Admin-Detailansicht
- [ ] Kein sichtbarer Layout-Bruch oder Farbabweichung gegenüber v3

### Dokumentation
- [x] `docs/security.md` — unberührt

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
