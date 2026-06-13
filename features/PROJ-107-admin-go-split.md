# PROJ-107: God-File-Refactor `internal/http/admin.go` (Phase 2 / Welle 3)

## Status: In Progress (Sub-Welle 107a abgeschlossen)

## Implementation 107a 2026-06-13 — Solo-Cluster

Erste Sub-Welle des God-File-Refactors. Vier isolierte Handler-Cluster mit
geringstem Risiko-Profil (kein gemeinsamer Subrouter-State, keine
Cross-Domain-Service-Calls).

**admin.go: 3316 → 2771 Zeilen (−16 %).**

### Extrahierte Files

| File | Methoden | LOC | Tenant-Iso |
|---|---|---|---|
| `admin_external_keys.go` | GetAPIKeyStatus, GenerateAPIKey, RevokeAPIKey | 127 | 3× `containsRC` |
| `admin_legal_documents.go` | List/Create/Update/Delete/Reorder | 229 | 5× `containsRC` (3 RC-basiert, 2 ID-Lookup) |
| `admin_attachments.go` | ExportApplicationExcel, DownloadApprovalPDF, DownloadJoiningDeclarationPDF | 197 | 3× `checkTenantAccess` |
| `admin_members.go` | SuggestNextMemberNumber | 73 | 1× `checkTenantAccess` |

**12/12 Handler-Tenant-Checks unveraendert.**

### AC-4: `tenant.go` Konsolidierung

- `internal/http/tenant.go` (29 Z): Single-Source-Helper `containsRC` mit
  Doku-Header zum nil-Slice-Verhalten (Memory `feedback_tenant_filter_nil_vs_empty`)
- `internal/http/tenant_test.go`: 8 Test-Vektoren incl. nil-Slice-Bypass-Sicherung,
  Case-Sensitivity, Prefix-Match-Verbot, Empty-String-Edge-Cases

### Verifikation 107a

- `go build ./...` clean
- `go test ./...` alle Pakete gruen
- `govulncheck ./...` 0 callable
- `gosec -severity medium ./internal/http/...` 0 Issues
- Tenant-Iso-Audit: alle 12 verschobenen Handler haben den `containsRC` bzw
  `checkTenantAccess`-Call vor jedem Repo-Aufruf

### Pending Sub-Wellen

- **107b** (Settings-Cluster): settings + settings_save_tx — eigenes /security-review
- **107c** (Applications-Cluster): applications + applications_recovery + attachments —
  Hochsensible Status-Transitions, eigenes /security-review
- **107d** (Audit-Cluster + admin.go-Slimming + Konsistenz-Sweep): finales /security-review


Dritte und schwerste Welle des God-File-Refactors. **3316 Zeilen + 77 exportierte Funktionen.** Enthält Tenant-Iso-Logik, Auth-Middleware-Aufrufe, viele Subrouter-Pfade. Risiko: HÖCHSTE.

**WICHTIG:** Diese Welle bekommt eigene `/grill-me`-Session VOR Implementierung + `/security-review` PRO Sub-Welle. Kein Big-Bang.

## Hintergrund

`internal/http/admin.go` hat sich von einem klaren Admin-Handler zu einem Catch-All gewachsen: Application-CRUD, Tenant-Settings, Customer-Onboarding (PROJ-71), Datenweiterleitung (PROJ-60), Reconciliation (PROJ-69), Audit (PROJ-78), External-API-Keys (PROJ-13), Brand-Editor (PROJ-103), Legal-Documents, Excel-Export, Member-Number-Sync (PROJ-27), Activation-Marker (PROJ-46), PROJ-100-Recovery-Pfade.

Probleme:
- Tenant-Iso-Checks (`containsRC`, `checkTenantAccess`) sind über die Datei verstreut. Audit eines neuen Endpoints erfordert das Lesen der Gesamt-Datei.
- Test-Coverage ist NICHT 100%: ein versehentlich ausgelassener Tenant-Check beim Refactor wird vom Compiler nicht gefangen.
- Subtle Middleware-Kompositionen via chi.Router-Subrouter — Wechselwirkungen unklar wenn man Pfade verschiebt.
- 3300+ Zeilen sprengen IDE-Outline-Sicht; Maintainer-Frustration.

## Scope

### IN-Scope (V1)
- Split nach Domäne in `internal/http/admin/`-Unterverzeichnis ODER nach Verb (CRUD pro Entity) — final via /grill-me + /architecture
- Tenant-Iso-Helper konsolidiert in `internal/http/tenant.go` (Single-Source — gehört ggf. zusätzlich Refactor `containsRC` → `claims.HasRC(rc)`-Method)
- Route-Wiring in `cmd/server/main.go` bleibt zentral
- Bestand-Tests müssen unverändert grün laufen
- KEINE API-Vertrags-Änderung

### OUT-Scope (Folge-PROJ)
- Auth-Middleware-Refactor (eigenes PROJ wenn nötig)
- Status-Transition-Map-Refactor (gehört in eigenes PROJ)
- DB-Schema-Änderungen
- Performance-Optimierungen (z. B. N+1-Queries fixen)

## Acceptance Criteria

**Struktur:**
- AC-1: `internal/http/admin/` Domain-Files (~10–14 Module, je <500 Zeilen)
- AC-2: `internal/http/admin.go` ist entweder leer (entfernt) ODER nur noch ein dünner Re-Export-Layer / Constructor-Hub
- AC-3: Jeder neue File hat Header-Kommentar mit Domain + Tenant-Iso-Pflichten
- AC-4: `containsRC`-Helper in `internal/http/tenant.go` mit Tests (Memory `feedback_tenant_filter_nil_vs_empty` als Test-Vektor)

**Korrektheit:**
- AC-5: `go build ./...` clean
- AC-6: `go test ./...` alle Tests grün — identisches Test-Set wie vor dem Split
- AC-7: `govulncheck ./...` 0 callable
- AC-8: `gosec -severity medium ./internal/...` 0 Issues
- AC-9: `npm run test:e2e` (Playwright Admin-Flows) grün — Settings + Apps + Billing + Datenexport-Tabs alle erreichbar

**Sicherheit — Pflicht-Verifikation pro Sub-Welle:**
- AC-10: Jeder Handler, der einen RC liest, ruft `claims.IsSuperuser() || containsRC(claims.Tenant, rc)` VOR Service-Layer-Aufruf — verifiziert via grep + manuelles Review
- AC-11: Kein Handler darf RC-Daten zurückgeben ohne Tenant-Check (auch nicht READ-only-GET)
- AC-12: Mutations (POST/PUT/DELETE) — selbe Pflicht
- AC-13: /security-review PRO Sub-Welle (nicht pro PROJ — 3–4 separate Reviews)

## Edge Cases

- EC-1: Bestand-Subrouter mit gemeinsamer Middleware-Chain (z. B. `r.Route("/admin/customer-onboarding", ...) {r.Use(...)}`) — beim Split müssen die Middleware unverändert pro Sub-File angewandt werden
- EC-2: Cross-Domain-Function-Calls (z. B. AdminHandler ruft customer-onboarding-Service direkt) — Service-Wiring muss erhalten bleiben
- EC-3: Audit-Log-Schreibungen sind heute teils im Handler, teils im Service — beim Split konsolidieren oder dokumentieren
- EC-4: Test-Setup: viele Tests bauen einen `AdminHandler` mit allen Dependencies — Test-Helper-Refactor wahrscheinlich nötig
- EC-5: PROJ-100-Recovery-Pfade (Reset-Activation, Reset-To-Review) — hochsensible Status-Transitions, dürfen NICHT versehentlich Tenant-Check verlieren

## Tech Design (vorläufig — final via /grill-me + /architecture)

### Domain-Split-Vorschlag

```
internal/http/admin/
├── handler.go                      (AdminHandler-Struct + Constructor)
├── applications.go                 (Application-CRUD, status-transitions, list, get)
├── applications_recovery.go        (PROJ-100 reset-activation, reset-to-review, reset-import)
├── settings.go                     (EEG-Settings inkl. Brand, View-Mode)
├── settings_save_tx.go             (SaveAllEEGSettingsTx-Caller)
├── attachments.go                  (PROJ-30 PDF-Uploads)
├── members.go                      (PROJ-27 Member-Number-Sync)
├── external_keys.go                (PROJ-13 API-Key-Mgmt)
├── legal_documents.go              (Legal-Docs-CRUD)
├── audit.go                        (PROJ-78 Audit-Log-Reads)
└── excel_export.go                 (Excel-Export)

internal/http/
├── admin.go                        (entfernt ODER Slim-Constructor-Hub)
└── tenant.go                       (containsRC + Tenant-Iso-Helper + Tests)
```

### Tech-Entscheidungen

1. **AdminHandler-Struct bleibt monolithisch**: 77 Methoden auf einem Receiver-Type, aber über mehrere Files verteilt (Go erlaubt das problemlos). Constructor-Wiring in `handler.go`, Methods in Domain-Files.
2. **Tenant-Helper auf claims-Method**: `claims.HasRC(rc)` statt `containsRC(claims.Tenant, rc)` — explizit nil-safe (Memory `feedback_tenant_filter_nil_vs_empty`).
3. **Sub-Wellen statt Big-Bang**: 3–4 Sub-Wellen (PROJ-107a, PROJ-107b, PROJ-107c, PROJ-107d), je Domain-Cluster. Pro Sub-Welle eigene `/security-review`.
4. **KEIN Auth-Middleware-Refactor**: bleibt 1:1 wie heute, sonst Risiko-Explosion.
5. **Test-Migration parallel**: pro Domain-Split die Tests mitziehen (`admin_applications_test.go` etc.).

### Sub-Wellen-Vorschlag

- **107a (kleinste Risiken, Solo-Cluster)**: external_keys, legal_documents, excel_export, members → 1 Tag, eigener Security-Review.
- **107b (Settings-Cluster)**: settings + settings_save_tx — sensitives Tenant-Pfad-Bündel → 1.5 Tage + Security-Review.
- **107c (Applications-Cluster)**: applications + applications_recovery + attachments — Hochsensible Status-Transitions → 2 Tage + Security-Review.
- **107d (Audit-Cluster + Cleanup)**: audit + admin.go-Slimming + Konsistenz-Sweep → 0.5 Tag + final Security-Review.

### Estimated Effort

~4–5 Tage Brutto inklusive 4 Security-Review-Slots. Höchster Aufwand der drei God-Files.

## Risiken

- **HIGH: Tenant-Iso-Bypass durch Split-Drift**: ein Handler verliert beim Verschieben den `containsRC`-Aufruf → fremde EEGs sichtbar/manipulierbar. Mitigation: pro Sub-Welle grep-basierter Audit + manuelles Review aller verschobenen Handler.
- **MEDIUM: Subrouter-Middleware-Drift**: chi.Router-Subrouter-Komposition wird durch File-Split kompliziert. Mitigation: Route-Wiring bleibt 1:1 in main.go, Sub-Files exportieren NUR Handler-Methods.
- **MEDIUM: Test-Coverage-Lücken werden sichtbar**: Refactor deckt Bestand-Lücken auf. Mitigation: Erwartet — vor Sub-Welle Coverage-Baseline messen, neue Tests sind Bonus aber kein Blocker.
- **MEDIUM: git-blame-Verlust**: 3300 Zeilen in Splits = viel Bug-Archäologie-Schaden. Mitigation: `git mv` mit minimalen Edits, dann separate Cleanup-Commits.
- **LOW: Compile-Time-Performance**: Multi-File-Package kompiliert minimal langsamer als Single-File. Vermutlich nicht messbar.

## Dependencies

- **Blockierend**: PROJ-104-Deploy abgeschlossen.
- **Blockierend für Prod-Gang**: PROJ-107 muss VOR Phase 3 (Prod) abgeschlossen sein — Memory `project_priority_before_prod`.
- **Nicht-blockierend**: kann parallel zu PROJ-106 laufen wenn Test-Coverage es zulässt.

## Verwandt

- PROJ-105: api.ts-Split (Frontend, niedrigstes Risiko)
- PROJ-106: registration-form.tsx-Split (Frontend, mittleres Risiko)
- Memory `project_priority_before_prod`: God-Files-Refactor ist Phase 2, **blockierend vor Prod**
- Memory `feedback_tenant_filter_nil_vs_empty`: nil-Slice-Falle bei containsRC
- Memory `feedback_webhook_xff_trust_gate`: Pattern wie Bestand-Helper aus Codebase wiederverwendet werden
