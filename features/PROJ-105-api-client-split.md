# PROJ-105: God-File-Refactor `src/lib/api.ts` (Phase 2 / Welle 1)

## Status: In Review (Welle 1A + 1B done 2026-06-13)

## Implementation-Notes Welle 1B (2026-06-13 Nachmittag)

Folgewelle nach 1A: 6 weitere End-of-File-Domänen extrahiert.

### Neue Module
- `reconciliation.ts` (PROJ-69 Reconciliation-Backstop, ~35 Zeilen)
- `resync.ts` (PROJ-70 Stammdaten-Resync + SEPA-Mandate-Resend, ~45 Zeilen)
- `settings.ts` (PROJ-32 EEG-Master-Data-Sync + `fetchEEGLogoBlob` + Intro-Text + PROJ-13 API-Key CRUD, ~105 Zeilen)
- `attachments.ts` (Excel-Export + Approval-PDF + PROJ-76 Joining-Declaration-PDF, ~60 Zeilen)
- `legal-docs.ts` (Legal-Documents CRUD + Reorder, ~50 Zeilen)
- `bulk.ts` (PROJ-25 Bulk-Actions, ~30 Zeilen)

### api.ts Stand nach Welle 1B
- **1295 Zeilen** (von 2442 in Welle 1A → −47 % gesamt)
- Re-exportiert jetzt 12 Domain-Module per Barrel (`public`, `data-export`, `configexport`, `billing`, `cockpit`, `reconciliation`, `resync`, `settings`, `attachments`, `legal-docs`, `bulk` + die Bestand-Helper aus `_internal`)
- Verbleibendes Bestand: Public-Form-Types (Response-Shapes lines 27-407), Admin-Application-Types (lines 408-737), Admin-Application-Functions (lines 738-1289) — Applications-CRUD/Status/Field-Config/Settings-CRUD/Recovery/Reset/Rollback/Reassign/Activation-Check/Tariffs/Settings-View-Mode

### Verifikation
- `tsc --noEmit` clean
- `vitest run` 238/238 grün
- `NEXT_PUBLIC_TEST_AUTH_MODE= npm run build` clean
- Backwards-Compat: alle Imports aus `@/lib/api` weiterhin valid

### Was bewusst deferred zu PROJ-105c (Welle 1C)
- **Welle 1C:** Applications-Domain extrahieren (~600 Zeilen) — komplexer wegen Cross-Deps zu Public-Form-Types
- **Welle 1C:** Form-Types + Admin-Types in `_form-types.ts` + `_admin-types.ts`
- **Welle 1C:** api.ts auf <50 Zeilen (Barrel-only) — AC-2 voll erfüllt
- **Welle 1C:** Co-located Tests pro Domain-Modul

---

Erste Welle des God-File-Refactors aus `project_priority_before_prod`. Nach PROJ-104-Deploy. Niedrigstes Risiko-Profil der drei God-Files — reines Type+Function-Modul ohne Runtime-State.

## Implementation-Notes 2026-06-13 (Welle 1A)

Pragmatic scope: 5 sauberste End-of-File-Domänen extrahiert. api.ts geht von **2442 → 1568 Zeilen (−874 / −36%)**. Spec-AC-2 ("<50 Zeilen") wird mit dieser Welle nicht erfüllt — die verbleibenden Domänen (admin-applications, settings, recovery, activation, reconciliation, resync, stammdaten-sync, board-approval, legal-documents, bulk-actions) brauchen Welle 1B (PROJ-105b) wegen höherer Cross-Dependency-Komplexität.

### Was geändert wurde

**Neue Module unter `src/lib/api/`:**
- `_internal.ts` (~145 Zeilen): `API_URL`, `adminAuthHeaders`, `ApiError`, `ApiResponseError`, `request`, `adminRequest` + sessionStorage-Cooldown-Helpers. Wird von allen Domain-Modulen + dem Bestand-api.ts importiert.
- `public.ts` (~50 Zeilen): `getRegistrationConfig`, `createApplication`, `submitApplication`. Public-Pfad ohne Auth. Types bleiben in api.ts (kommen in Welle 1B).
- `billing.ts` (~252 Zeilen): kompletter PROJ-104-Block — 9 Types + 12 Functions (Pricing-Plans, EEG-State, Pre-Flight, Live-Toggle, Edition, Trigger, Invoices, Credit-Note, Audit-Log, EEG-Own-Invoices).
- `cockpit.ts` (~50 Zeilen): PROJ-72-Block — 2 Types + 2 Functions.
- `data-export.ts` (~250 Zeilen): PROJ-60-Block — 11 Types + 12 Functions + `triggerBrowserDownload`-Helper.
- `configexport.ts` (~205 Zeilen): PROJ-61-Block — 13 Types + 3 Functions (Download/Preview/Apply). Importiert `FieldState` aus `../api` (Type-Re-Export bleibt in api.ts).

**`src/lib/api.ts` Anpassung:**
- Top: alle 6 neuen Module via `export *` re-exportiert. `API_URL`/`ApiResponseError`/`ApiError` zusätzlich explizit re-exportiert (für `import type`-Konsumenten).
- Helpers (`request`, `adminRequest`, `adminAuthHeaders`) werden jetzt aus `_internal.ts` importiert + von Bestand-Funktionen verwendet.
- DataExport-, ConfigExport-, Billing-, Cockpit-, Public-Blöcke ersatzlos gelöscht (Truncate ab Line 1565).
- Verbleibende Domänen weiter in api.ts: Admin-Applications-CRUD, Status-Transitions, Field-Config, Settings, Recovery, Reassign, Activation-Check, Tariffs, Reconciliation (PROJ-69), Stammdaten-Resync (PROJ-70), Master-Data-Sync (PROJ-32), Board-Approval-Download, Legal-Documents-Admin, Bulk-Actions.

### Backwards-Compat (AC-3)

KEINE Aufrufer-Side-Änderung. Spot-Checks: `BillingPricingPlan`/`BillingInvoice`/`CockpitEEG`/`DataExportJobResponse`/`ConfigExportFile` werden weiterhin aus `@/lib/api` importierbar dank Barrel-Re-Export. `npx tsc --noEmit` clean (verifiziert).

### AC-Erfüllungs-Map

| AC | Status | Anmerkung |
|---|---|---|
| AC-1 (~10-14 Module <300 Zeilen) | 🟡 Partial | 6/14 Module angelegt; alle <260 Zeilen |
| AC-2 (api.ts <50 Zeilen) | ⏳ Deferred zu PROJ-105b | api.ts 1568 Zeilen; verbleibende Domänen brauchen eigene Welle |
| AC-3 (kein Aufrufer-Change) | ✅ | Barrel-Re-Export, tsc clean |
| AC-4 (tsc --noEmit) | ✅ | Clean |
| AC-5 (npm run build) | ✅ | Production-Build clean (mit `NEXT_PUBLIC_TEST_AUTH_MODE=`) |
| AC-6 (vitest) | ✅ | 238/238 grün |
| AC-7 (Playwright E2E) | 🟡 | Lokaler Lauf out-of-scope; CI verifiziert post-push |
| AC-8 (Type-Imports gültig) | ✅ | Compiler-Check Pflicht, clean |
| AC-9 (Header-Kommentare) | ✅ | Jedes Modul mit PROJ-Referenz + Scope-Beschreibung |
| AC-10 (Co-located Tests) | ⏳ Deferred zu PROJ-105b | Bestand-Tests bleiben in `src/lib/api.test.ts` wenn vorhanden |
| AC-11 (git log --follow) | ✅ | Code als Block kopiert/eingefügt — Git-Rename-Detection greift bei Block-Match |

### Was bewusst deferred zu PROJ-105b

- **Welle 1B Domain-Split** für: applications, settings, recovery, activation, reconciliation, resync, stammdaten-sync, board-approval, legal-documents, bulk-actions, tariffs. Estimated: ~1 Tag.
- **api.ts auf <50 Zeilen** (Barrel-only). Folgt aus Welle 1B.
- **Co-located Tests** pro Domäne — Bestand-Test-Suite bleibt zentral bis Welle 1B.
- **`_types.ts`** für Cross-Domain-Types — erst wenn Cycle-Risk auftritt.

### Memory-Regeln aktiv

- `feedback_admin_field_full_chain` — irrelevant für PROJ-105 (kein Admin-Field)
- `feedback_no_proj_refs_in_user_doc` — irrelevant (Code-internal)
- `feedback_qa_full_chain_verify` — tsc + vitest + build alle clean

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
