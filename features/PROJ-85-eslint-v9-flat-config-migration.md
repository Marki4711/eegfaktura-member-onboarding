# PROJ-85: ESLint v9 Flat-Config-Migration

## Status: Planned
**Created:** 2026-06-08
**Last Updated:** 2026-06-08
**Typ:** Dev-Tooling
**Prio:** Niedrig (kein Production-Blocker)

## Hintergrund

Bei der PROJ-82-Arbeit fiel auf, dass `npm run lint` (= `next lint`)
fehlschlägt mit:

```
Invalid project directory provided, no such directory:
C:\opt\repos\eegfaktura-member-onboarding-private\lint
```

Direkter `npx eslint`-Aufruf zeigt die Ursache:

```
From ESLint v9.0.0, the default configuration file is now eslint.config.js.
If you are using a .eslintrc.* file, please follow the migration guide
to update your configuration file to the new format.
```

Im Repo liegen:
- `package.json` mit `"eslint": "^10"` (10 = pre-release, faktisch Flat-Config-only)
- Eine `.eslintrc.*`-Datei oder Next-eingebettete Legacy-Konfiguration

Beides ist inkompatibel. `next lint` ist zusätzlich seit einer Next-Major-Version
deprecated zugunsten direktem ESLint-Aufruf.

## Warum bisher kein Blocker

- CI-Workflow `CI Build & Test` läuft ohne Lint-Schritt (verifiziert
  2026-06-08 in den letzten 6 erfolgreichen Runs für PROJ-79/82/83)
- Lokal sind `npx tsc --noEmit` (Typecheck) und `npm run build`
  (Next-Production-Build) die echten Korrekt-heits-Gates und beide laufen
  clean durch
- Code-Review fängt die meisten Lint-Themen mit ab

## Owner-Direktive 2026-06-08

> „1) umsetzen als eigenes proj"

(Bezug: meine Liste mit ESLint als Item 1.)

## Scope

### Betroffen

- `package.json` — ESLint-Version-Strategie + Scripts klären
- `eslint.config.js` (NEU) oder `eslint.config.mjs` — Flat-Config
- Etwaige bestehende `.eslintrc.*`-Files entfernen
- `.github/workflows/*.yml` — optional Lint-Schritt im CI-Workflow ergänzen

### Nicht betroffen

- Anwendungs-Code (Lint-Regeln können punktuell `// eslint-disable-next-line`
  brauchen, aber kein logischer Eingriff)
- Backend (Go ist eigene Toolchain)

## Akzeptanzkriterien (Skizze)

- [ ] **AC-1** ESLint v9 Flat-Config in `eslint.config.js` (oder
  `eslint.config.mjs`) als Single source of truth
- [ ] **AC-2** Alle bisherigen `.eslintrc.*`-Dateien entfernt
- [ ] **AC-3** `npm run lint` führt `eslint .` aus, ohne `next lint`
  (Next-Lint ist seit Next 15 deprecated)
- [ ] **AC-4** Lint läuft clean über den bestehenden `src/`-Code (etwaige
  Findings werden entweder gefixt oder bewusst per `// eslint-disable-next-line`-
  Kommentar mit Begründung markiert)
- [ ] **AC-5** Wichtige Memory-Regeln als Lint-Pattern (wo möglich):
  - `feedback_no_placeholders`: Custom-Regel oder Override für
    `placeholder=`-Attribut auf Input/Textarea/Select
- [ ] **AC-6** CI-Workflow `CI Build & Test` führt `npm run lint` als
  Schritt vor `npm run build` aus
- [ ] **AC-7** Doku in `docs/development.md` (oder analog) aktualisiert

## Edge Cases

- **EC-1 Bestehende Regel-Verletzungen im Code:** Bei Migration werden
  möglicherweise mehrere Findings auftauchen. Entscheidung pro Finding:
  Fix in einem separaten Commit ODER `// eslint-disable-next-line` mit
  Begründung. Owner muss bei großem Cleanup-Bedarf eingebunden werden.
- **EC-2 `next lint`-Backwards-Compat:** Next 15 hat `next lint`
  deprecated und Next 16+ entfernt es. Migration auf direkten
  `eslint .` ist ohnehin der Weg.
- **EC-3 Editor-Integration (VS Code, IntelliJ):** Flat-Config wird von
  modernen ESLint-Plugins unterstützt. Sollte ohne weiteres greifen.
- **EC-4 Pre-Commit-Hook:** Aktuell vermutlich keiner. Falls einer kommt
  (`husky` o. ä.), muss er den neuen Lint-Command nutzen.

## Tech Design

ESLint v9 Flat-Config ist ein neues Format. Skizze:

```js
// eslint.config.js
import nextPlugin from "@next/eslint-plugin-next";
import tsParser from "@typescript-eslint/parser";

export default [
  {
    files: ["src/**/*.{ts,tsx}"],
    languageOptions: { parser: tsParser },
    plugins: { next: nextPlugin },
    rules: {
      ...nextPlugin.configs.recommended.rules,
      // Memory-Regel feedback_no_placeholders als Custom-Rule:
      "no-restricted-syntax": [
        "warn",
        {
          selector: "JSXAttribute[name.name='placeholder']",
          message: "Placeholder vermeiden — Hint-Popover stattdessen (siehe .claude/rules/frontend.md)",
        },
      ],
    },
  },
];
```

(Skizze — exakte Plugin-Liste, Parser-Optionen, Override-Strategien
gehören in /architecture-Phase.)

## Geschätzter Aufwand

2-4 h:
- Migration-Recherche (eslint v9 + Next 16 best practices): 30 Min
- Flat-Config schreiben + Plugins installieren: 1 h
- Bestehende Findings durchgehen + fixen oder dokumentieren: 1-2 h
- CI-Workflow-Schritt + Doku: 30 Min

## Workflow

`/grill-me` nicht zwingend (klar abgegrenzter Tooling-Wechsel).
`/architecture` reicht zur Plugin-Auswahl + Override-Strategie.

## Sicherheits-Bewertung

Reine Dev-Tooling-Änderung. Kein Production-Code-Pfad berührt.
`/security-review` nicht erforderlich.

## Geschätzte Risiken

- **Plugin-Inkompatibilität:** Next.js + ESLint-Plugin-Versionen müssen
  zusammenpassen. Lock-File-Konflikte möglich.
- **Verstecktes Lint-Debt:** Migration könnte 20-50 Findings im Code
  zeigen, die heute unerkannt sind. Bei größeren Mengen separater
  Cleanup-Commit empfohlen.

## Dependencies

- Optional voraussetzt: Next-Major-Update (falls aktuell auf Next 14)
- Berührt: keine PROJ-Specs direkt

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
