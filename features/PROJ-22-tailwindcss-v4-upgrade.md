# PROJ-22: Tailwind CSS v3 → v4 Upgrade

## Status: Approved
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
- [x] `npm run dev` startet ohne Fehler
- [x] Visuelle Überprüfung: Registrierungsformular (Label-Abstände, Font, Card-Layout, Select-Dropdown)
- [x] Kein sichtbarer Layout-Bruch gegenüber v3 (akzeptierte Abweichung: Roboto auf Inputs statt System-Font, da v3 Font-Vererbung fehlerhaft war)

### Dokumentation
- [x] `docs/security.md` — unberührt

## Edge Cases

- **`@apply` mit entfernten Utilities:** v4 entfernt einige Utilities oder ändert deren Namen → Linting-Fehler müssen vor Build behoben werden
- **Custom Plugins:** Falls Drittanbieter-Plugins verwendet werden, müssen diese v4-kompatibel sein
- **Dark Mode:** Konfiguration ändert sich in v4 (CSS-Variable statt `class`-Toggle) → explizit testen
- **`tailwind-merge` / `clsx`:** Diese Libraries sind unabhängig von Tailwind und sollten kompatibel bleiben

## Tech Design (Solution Architect)
_To be added by /architecture_

## Implementation Notes

Visuelle Regressions die im Rahmen der v4-Migration behoben wurden:

- **Roboto Font**: `font-(--font-roboto)` Klasse auf body funktioniert in v4, Fallback-Regel in globals.css ergänzt
- **Card-Padding**: Tailwind v4 Cascade-Änderungen haben `p-6 pt-0` gebrochen → Inline-Styles auf CardHeader/CardContent/CardFooter
- **CardTitle-Höhe**: `tailwind-merge` v3 entfernte `leading-none` wenn kombiniert mit `text-*` → `leading-none` als letztes Argument in `cn()`, `tailwind-merge` auf v3.5 aktualisiert
- **FormItem-Abstände**: `space-y-2` in v4 verwendet `:where()` (Zero-Specificity) + `margin-block-end` (logische Eigenschaft); Tailwind v4 Preflight setzt `margin: 0` auf `*` in `@layer base` und überschreibt den physischen `margin-top`. Fix: `flex flex-col gap-2.5` — CSS Gap wird nicht durch Margin-Reset beeinflusst
- **Select-Dropdown**: `avoidCollisions={false}` + `side="bottom"` als Defaults in SelectContent gesetzt
- **Font-Smoothing**: Tailwind v4 Preflight setzt Form-Elemente auf Browser-Default-Rendering → explizit `-webkit-font-smoothing: antialiased` auf `input, textarea, button, select` in globals.css
- **Input Font-Weight**: `font-normal` explizit auf Input und SelectTrigger gesetzt
- **Font auf Inputs**: In v4 erben Inputs korrekt Roboto (v3 hatte Font-Vererbungs-Bug, Inputs verwendeten System-Font). Bewusst beibehalten.

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
