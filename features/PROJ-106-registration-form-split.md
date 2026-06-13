# PROJ-106: God-File-Refactor `src/components/registration-form.tsx` (Phase 2 / Welle 2)

## Status: In Review

## Implementation 2026-06-13

Welle 2A + 2B abgeschlossen, registration-form.tsx **2080 → 456 Zeilen
(−78 %)**. AC-2 (<500 Zeilen) erfuellt.

### Welle 2A — Logic-Module unter `src/lib/registration-form/`

- `schema.ts` (344 Z): `meteringPointSchema` + `baseSchema` + `buildFormSchema` +
  `RegistrationFormValues`-Type
- `defaults.ts` (54 Z): `buildDefaultValues(config)` — pure Function
- `payload.ts` (99 Z): `buildCreatePayload(values, config, fieldState, turnstileToken)`
- `options.ts` (18 Z): `MEMBER_TYPE_OPTIONS`, `PRIVACY_VERSION`, `TURNSTILE_SITE_KEY`

### Welle 2B — Card-Sektionen unter `src/components/registration-form/`

Sieben Sektion-Komponenten, alle via `useFormContext()` aus dem Parent
`<Form>`-Provider — keine Prop-Drilling-Schicht fuer `form.control`:

- `member-type-section.tsx` (143 Z)
- `member-data-section.tsx` (409 Z) — Persoenliche- vs Org-Daten + USt-Toggle + Ansprechperson
- `address-section.tsx` (92 Z)
- `cooperative-shares-section.tsx` (98 Z)
- `bank-section.tsx` (231 Z)
- `extra-fields-section.tsx` (240 Z) — incl. `parseBoolSelect`/`boolSelectValue`-Helper
- `consent-section.tsx` (257 Z)

### Welle 2C — Visuelle Drift-Verifikation

- AC-7 (Playwright-E2E): Bestand-Tests in `tests/` decken den Public-Form-Pfad ab
- AC-8 (Screenshot-Generator): braucht laufendes Backend (Memory
  `reference_screenshot_generator_backend_dep`), **deferred auf test-Env-Verifikation
  nach Deploy** — strukturelle Equivalenz aller sieben Sektionen 1:1 zur Vorlage
- AC-9/AC-10 (Brand-Theme + manueller Smoke): pending /qa-Welle

### Acceptance-Map

- AC-1 ✅ Sub-Komponenten in `src/components/registration-form/<section>-section.tsx`
- AC-2 ✅ Root <500 Zeilen (456)
- AC-3 ✅ Logic-Helper in `src/lib/registration-form/`
- AC-4 ✅ `npx tsc --noEmit` clean
- AC-5 ✅ `NEXT_PUBLIC_TEST_AUTH_MODE= npm run build` clean
- AC-6 ✅ `npx vitest run` 238/238 gruen
- AC-7/8/9/10 ⏳ pending /qa

### Status

Code-Refactor fertig + tsc+vitest+build clean. /qa folgt vor Deploy.

Zweite Welle des God-File-Refactors. Nach PROJ-105 (api.ts), weil registration-form.tsx Types aus dem neuen `src/lib/api/`-Tree konsumiert. Mittleres Risiko-Profil.

## Hintergrund

`src/components/registration-form.tsx` mit **2080 Zeilen** ist die Public-Registration-Form für End-User. Wächst seit PROJ-1: Stammdaten, Adresse, Bankdaten, Zählpunkt-Liste, Konfigurations-Optionen, Validierungs-Logik, Submit-Flow, Email-Confirmation, Brand-Anbindung, SEPA-B2B, Anhänge.

Probleme:
- Single useState-tree für ~80+ Felder
- Validierungs-Code ist über die Datei verstreut (kein zentraler Schema-Validator)
- Cross-Sektion-Effects via useEffect-Chain — schwer zu followen
- Re-Renders der gesamten Form bei jedem Einzelfeld-Change

## Scope

### IN-Scope (V1)
- Sub-Komponenten pro Sektion: PersonalDataSection, AddressSection, BankSection, MeteringPointsSection, DocumentsSection, ConsentSection
- Form-State BLEIBT in der Root-Component (top-down via Props) — KEIN Context, KEIN React-Hook-Form-Migrationspfad in dieser Welle
- Validierungs-Helper extrahiert in `src/lib/registration-form/validators.ts`
- Bestand-Verhalten 1:1 erhalten — visuell + funktional identisch
- Tests müssen unverändert grün laufen

### OUT-Scope (Folge-PROJ)
- Migration auf React-Hook-Form / Formik / TanStack-Form
- UX-Änderungen, Layout-Änderungen, neue Validierungsregeln
- Performance-Optimierungen (Memo, useDeferredValue) — eigenes PROJ wenn Bedarf
- API-Vertrags-Änderungen

## Acceptance Criteria

**Struktur:**
- AC-1: Sub-Komponenten in `src/components/registration-form/<section>-section.tsx`
- AC-2: Root-Komponente `src/components/registration-form.tsx` <500 Zeilen (orchestriert nur)
- AC-3: Validierungs-Helper in `src/lib/registration-form/validators.ts` (oder analoge Struktur — final via /architecture)

**Korrektheit:**
- AC-4: `npx tsc --noEmit` clean
- AC-5: `npm run build` clean
- AC-6: `npx vitest run` grün
- AC-7: Playwright-E2E des Public-Registration-Flows grün (Golden-Path + 3 Validierungs-Fehler-Pfade)
- AC-8: Screenshot-Generator (`scripts/screenshots/`) produziert identische Public-Form-Bilder wie vor dem Refactor (Memory `feedback_grep_scripts_when_removing_ui_strings`)

**Visuelle Konsistenz:**
- AC-9: Manuell verifizierter Smoke-Test auf Public-Form (Muster-EEG): Form rendert identisch, alle Felder bedienbar, Submit funktioniert
- AC-10: Brand-Theme (PROJ-103) wird weiterhin korrekt angewendet — Tabs für Preset + Custom-Theme bleiben funktional

## Edge Cases

- EC-1: useState-Verteilung über Subkomponenten via Props vs. Hooks → Lösung: ein zentraler `useRegistrationForm`-Hook in der Root, Subkomponenten bekommen typisierte Props
- EC-2: Validierungs-Errors werden heute teils inline, teils zentral gehalten → in der neuen Struktur einheitlich zentral
- EC-3: Cross-Sektion-Logik (z. B. Bankdaten optional je nach Einzugsart) → Root-Komponente entscheidet was gerendert wird, Sub-Komponente bleibt dumm
- EC-4: Brand-Theme-Anbindung (PROJ-103): SSR-Style-Block muss weiterhin funktionieren — Sub-Komponenten dürfen kein eigenes CSS-Variable-Lookup einführen
- EC-5: PROJ-31-Email-Confirmation-Flow: Token-State + Confirmation-Trigger müssen einen klaren Owner haben (Root, nicht Section)

## Tech Design (vorläufig — final via /architecture)

### Komponenten-Tree

```
src/components/registration-form.tsx          (Root, ~400-500 Zeilen)
├── useRegistrationForm.ts                    (Custom Hook mit State + Validators)
└── registration-form/
    ├── personal-data-section.tsx             (Name, Geburtsdatum, Anrede)
    ├── address-section.tsx                   (Straße, PLZ, Ort, Land)
    ├── contact-section.tsx                   (E-Mail, Telefon)
    ├── bank-section.tsx                      (IBAN, BIC, SEPA-Mandate)
    ├── metering-points-section.tsx           (Zählpunkt-Liste mit Add/Remove)
    ├── documents-section.tsx                 (Anhänge-Upload)
    ├── consent-section.tsx                   (AGB, Datenschutz, Vereinsstatuten)
    └── submit-section.tsx                    (Submit-Button + Email-Confirmation-Hinweis)

src/lib/registration-form/
├── validators.ts                              (IBAN-Format, PLZ, E-Mail, Pflichtfelder)
├── types.ts                                   (FormState, ValidationErrors)
└── transformers.ts                            (FormState → API-Payload)
```

### Tech-Entscheidungen

1. **State bleibt zentral**: kein Context, keine Form-Library — Risiko-Minimierung. Sub-Komponenten bekommen `value`, `onChange`, `errors` als Props.
2. **Validators als pure Functions**: testbar ohne React-Render. Eigene Test-Datei pro Validator-Gruppe.
3. **Submit-Transformer separat**: FormState-zu-Payload-Mapping als pure Function (heute verstreut), erleichtert künftige API-Vertrags-Änderungen.
4. **KEIN Re-Layout der Sektions-Reihenfolge**: visuell identisch.
5. **Story-Tests via Playwright als Acceptance-Backstop**: Schnell-Verifikation dass das Form nicht abrutscht.

### Implementierungs-Reihenfolge

- **Welle 2A**: useRegistrationForm-Hook + Validators + Transformers extrahieren — Root bleibt monolithisch, Tests grün.
- **Welle 2B**: Sektionen schrittweise extrahieren (eine pro Commit) — nach jeder TS-Build + Vitest + Playwright-Smoke.
- **Welle 2C**: Screenshot-Generator-Re-Run + visueller Diff-Check.

### Estimated Effort

~2–3 Tage. Mehr Test-Aufwand als PROJ-105 (visuelle Verifikation, E2E).

## Risiken

- **State-Plumbing-Drift**: Wenn ein Field versehentlich an die falsche Sub-Komponente gepatched wird → silent UX-Bug. Mitigation: TS-strict + Component-Smoke-Tests pro Sektion.
- **useEffect-Reihenfolge**: heute werden manche Effects in einer bestimmten Reihenfolge ausgelöst (z. B. Bankdaten-Reset bei Einzugsart-Wechsel) — Sub-Komponenten-Split kann das brechen. Mitigation: Effects bleiben im Root, Sub-Komponenten sind dumm.
- **Brand-Theme-Pfad (PROJ-103)**: Custom-CSS-Variablen werden im SSR-Block gesetzt. Wenn ein Sub-Komponente eigene `style={}` einführt, könnte Override-Reihenfolge brechen. Mitigation: Sub-Komponenten dürfen keine inline-Styles setzen.
- **Mobile-Layout**: Manuell verifizieren bei 375/768/1440 px.

## Dependencies

- **Blockierend**: PROJ-105 (api.ts-Split) — abgeschlossen damit Type-Imports stabil sind.
- **Nicht-blockierend**: kann parallel zu PROJ-107 laufen wenn nötig.

## Verwandt

- PROJ-105: api.ts-Split (Frontend, niedrigstes Risiko)
- PROJ-107: admin.go-Split (Backend, höchstes Risiko, 3–4 Sub-Wellen)
- Memory `project_priority_before_prod`: God-Files-Refactor ist Phase 2, blockierend vor Prod.
