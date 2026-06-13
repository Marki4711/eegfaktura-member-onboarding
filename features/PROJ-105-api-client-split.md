# PROJ-105: God-File-Refactor `src/lib/api.ts` (Phase 2 / Welle 1)

## Status: Planned

Erste Welle des God-File-Refactors aus `project_priority_before_prod`. Nach PROJ-104-Deploy. Niedrigstes Risiko-Profil der drei God-Files — reines Type+Function-Modul ohne Runtime-State.

## Hintergrund

`src/lib/api.ts` ist mit **2402 Zeilen + 164 exportierten Symbolen** der zweitgrößte God-File. Wachstum-Treiber: jede PROJ hat hier Types + Fetch-Functions reingelegt (PROJ-13 ApiKey, PROJ-22 Excel, PROJ-31 EmailConfirm, PROJ-46 Activation, PROJ-60 DataExport, PROJ-67 Settings-View-Mode, PROJ-71 Customer-Onboarding, PROJ-78–84 Reconciliation/Audit/B2B, PROJ-103 BrandTheme, PROJ-104 Billing).

Probleme:
- Maintainer-Onboarding: neue Devs müssen 2400 Zeilen scannen, um die nächste Function-Signature zu finden.
- Type-Drift-Risiko: Bei manchen Domänen liegen Types lokal in Components zusätzlich (Duplikat-Pattern).
- Test-Granularität: `api.test.ts` ist eine Mega-Datei; isolierte Tests pro Domäne erleichtern Regression-Diagnose.
- IDE-Performance: TypeScript-Server lädt bei jedem Tippen die gesamte Datei.

## Scope

### IN-Scope (V1)
- Split nach Domäne in `src/lib/api/`-Unterverzeichnis mit ~10–14 Modulen (siehe Tech-Design unten)
- Barrel-Export aus `src/lib/api.ts` für Backwards-Compat — KEINE Aufrufer-Side-Änderungen
- Tests pro Domäne in `src/lib/api/<domain>/<domain>.test.ts` (co-located)
- Type-Re-Exports zentral, damit `import type { Application } from '@/lib/api'` weiter funktioniert
- TS-Build + Vitest + Playwright müssen unverändert grün laufen
- Keine API-Vertrags-Änderung — reines Code-Reorg

### OUT-Scope (Folge-PROJ)
- Backend-Refactor `internal/http/admin.go` → PROJ-107
- Component-Refactor `registration-form.tsx` → PROJ-106
- API-Vertrags-Änderungen, Endpoint-Umbenennungen, Response-Shape-Refactors
- Migration auf SWR/TanStack-Query (eigenes PROJ wenn überhaupt)

## Acceptance Criteria

**Struktur:**
- AC-1: `src/lib/api/` existiert mit Domänen-Modulen (~10–14 Files, je <300 Zeilen)
- AC-2: `src/lib/api.ts` ist nur noch Barrel-Re-Export (<50 Zeilen)
- AC-3: KEINE Aufrufer-Side-Änderung in `src/app/**` oder `src/components/**` — bestehende `import { foo } from '@/lib/api'` funktioniert unverändert

**Korrektheit:**
- AC-4: `npx tsc --noEmit` clean
- AC-5: `npm run build` (production) clean
- AC-6: `npx vitest run` alle Tests grün — identisches Test-Set wie vor dem Split (keine Test-Coverage-Reduktion)
- AC-7: `npm run test:e2e` (Playwright) Smoke-Set grün — Public-Form + Admin-Settings + Billing-Page erreichen ihre Daten
- AC-8: Bestehende Type-Imports (z. B. `import type { EEGSettings, Application, BillingInvoice } from '@/lib/api'`) bleiben gültig — Compiler-Check Pflicht

**Qualität:**
- AC-9: Jedes Domänen-Modul hat einen Header-Kommentar mit der enthaltenen Domäne (PROJ-Referenzen entfernt — Code-internal, kein User-Doc, aber für Maintainer hilfreich)
- AC-10: Co-located `<domain>.test.ts` pro Domäne, falls Tests existieren (sonst leer mit `describe.skip` als Anker für nächste Welle)
- AC-11: `git log --follow` funktioniert für die migrierten Symbole (über `git mv`-Splits unterstützt)

## Edge Cases

- EC-1: Zyklische Imports zwischen Domänen-Modulen (z. B. Settings braucht Application-Type, Application braucht Settings-Type) → Lösung: gemeinsame Types in `src/lib/api/_types.ts`
- EC-2: Bestand-Tests in `src/lib/api.test.ts` müssen pro Domäne aufgeteilt werden — Test-Setup (Mocks, Vitest-Setup-Hooks) ggf. duplizieren oder in `src/lib/api/_test-helpers.ts` ziehen
- EC-3: API-Client-Functions mit cross-domain-Calls (z. B. `submitOnboarding` ruft `setEdition` intern) — direkt-Import statt Barrel um Zyklus zu vermeiden
- EC-4: Falls eine Function NICHT eindeutig zu einer Domäne gehört (z. B. generischer `fetchWithAuth`-Helper) → bleibt in `src/lib/api/_internal.ts` als shared utility
- EC-5: Build-Output-Größe sollte sich nicht messbar ändern — Tree-Shaking ist bereits aktiv, der Split sollte nur die Source-Reorg sein

## Tech Design (vorläufig — final via /architecture)

### Domänen-Split-Vorschlag

```
src/lib/api/
├── _internal.ts         (fetchWithAuth, getAuthHeader, baseUrl-resolver)
├── _types.ts            (cross-domain types: ApiError, PaginatedResponse)
├── applications.ts      (CRUD von applications, status-transitions, attachments)
├── public.ts            (public registration GET/POST)
├── settings.ts          (EEG-Settings inkl. brand_*, view_mode, Stammdaten)
├── customer-onboarding.ts (PROJ-71 contract + event-log)
├── data-export.ts       (PROJ-60 Datenweiterleitung)
├── reconciliation.ts    (PROJ-69 Audit/Reconciliation)
├── billing.ts           (PROJ-104 Pricing-Plan, EEG-Billing-State, Invoices, Audit)
├── auth.ts              (Session, Token-Refresh, Logout)
├── attachments.ts       (PROJ-30 PDF-Uploads)
├── audit.ts             (cross-cutting Audit-Log-Reads PROJ-78)
├── external.ts          (External-API-Key-Management PROJ-13)
└── system.ts            (Health-Check, Version-Info)

src/lib/api.ts            (Barrel: re-export *)
```

### Tech-Entscheidungen

1. **Barrel-Export für Backwards-Compat**: kein Big-Bang-Refactor, ~120+ Aufrufer-Files bleiben unangetastet.
2. **Co-located Tests**: `src/lib/api/billing.test.ts` neben `src/lib/api/billing.ts` — etabliertes Pattern aus PROJ-103.
3. **`_internal.ts` für shared helpers**: kein eigenes Modul-Namespace; Unterscheidung via `_`-Prefix.
4. **Type-Konsolidierung**: Cross-domain Types in `_types.ts`, domain-spezifische bleiben im Domain-File.
5. **Git-mv für History**: `git mv` (oder Equivalent: separate add/delete im selben Commit) damit `git log --follow` funktioniert.

### Implementierungs-Reihenfolge

- **Welle 1A**: Skeleton `src/lib/api/` mit allen leeren Modulen + `_internal.ts` + Barrel — Compiler grün.
- **Welle 1B**: Migration domäne-by-domäne (1 Commit pro Domäne) — nach jeder Migration TS-Build + Vitest grün. Reihenfolge: kleinste zuerst (system, external, auth) → mittel (settings, applications, public) → grösste (billing).
- **Welle 1C**: Tests aufteilen, dann alte `api.test.ts` entfernen.
- **Welle 1D**: Doku (`docs/architecture.md` Frontend-Abschnitt erweitern; KEIN User-Guide-Update nötig — interne Strukturänderung).

### Estimated Effort

~1–1.5 Tage. Wenig Risiko, viel Mechanik.

## Risiken

- **Type-Drift**: wenn Domänen-Modul einen Type lokal redeklariert statt re-importiert → silent Bug. Mitigation: TS strict + Compiler-Check pro Welle.
- **Vitest-Setup-Drift**: wenn Test-Mocks unterschiedlich zwischen alter und neuer Datei laufen → False-Positives. Mitigation: Tests 1:1 mitkopieren, dann alte Datei löschen.
- **Cyclic-Import-Spirale**: Domain-A importiert von Domain-B importiert von Domain-A → Lösung: Types in `_types.ts` ziehen.
- **Git-Blame-Verlust**: `git mv` + Edit im selben Commit kann Blame brechen. Mitigation: erst `git mv` ohne Edits, dann Cleanup-Commit.

## Dependencies

- **Blockierend**: PROJ-104-Deploy abgeschlossen (sonst Merge-Konflikt-Risiko mit Welle 4b-File `src/lib/api.ts`).
- **Nicht-blockierend, aber sinnvoll**: PROJ-106 (registration-form.tsx) sollte NACH PROJ-105 laufen, weil registration-form.tsx Types aus api.ts importiert.

## Verwandt

- PROJ-106: registration-form.tsx-Split (Frontend, mittleres Risiko)
- PROJ-107: admin.go-Split (Backend, höchstes Risiko, 3–4 Sub-Wellen)
- Memory `project_priority_before_prod`: God-Files-Refactor ist Phase 2, blockierend vor Prod.
