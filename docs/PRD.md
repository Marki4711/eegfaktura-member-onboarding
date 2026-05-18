# Product Requirements Document
## eegfaktura Member Onboarding

## 1. Vision

`eegfaktura Member Onboarding` enables self-service registration of members for EEGs managed in eegFaktura.

Today, new members are created manually by an administrator in eegFaktura. The goal of this product is to reduce manual effort by allowing members to submit their own basic data through a web form.

Submitted data must first stay in a dedicated onboarding workflow. Only after admin review and explicit approval may the data be imported into the eegFaktura core.

## 2. Target Users

### Public user
A potential new EEG member who wants to register through a fixed registration link identified by the EEG's RC number.

### EEG admin
A user who reviews applications for a specific EEG and decides whether an application is complete and ready for import.

### Superuser
Cross-EEG operator with access to all RC numbers; used for cross-tenant operations (e.g. EEG-Umzuordnung, sammel-Aktivierungs-Check).

## 3. MVP Scope (shipped)

Version 1 — all delivered:

- fixed registration link per EEG, identified by the EEG's RC number
- RC number resolved locally through `member_onboarding.registration_entrypoint` — no direct reads from eegFaktura core tables
- public self-service form
- collection of member master data
- collection of multiple metering points (with per-MP deviating address — PROJ-39)
- admin review flow
- 12-status workflow including post-import lifecycle (PROJ-46)
- controlled import into eegFaktura core (direct API call — PROJ-4)
- dedicated onboarding persistence in schema `member_onboarding`

## 4. Feature Roadmap

Single source of truth for current implementation status is [`features/INDEX.md`](../features/INDEX.md). The table below mirrors that file at the time of writing (2026-05-17).

### Shipped to production

| ID | Feature | Notes |
|----|---------|-------|
| PROJ-1 | Public Registration | MVP foundation |
| PROJ-2 | Admin Review | MVP foundation |
| PROJ-3 | Admin Frontend UI | MVP foundation |
| PROJ-4 | Core Import | Direct API call to eegFaktura core |
| PROJ-5 | Keycloak-secured Admin Area | Auth foundation |
| PROJ-6 | E-Mail-Benachrichtigungen | Submit + EEG notification |
| PROJ-7 | Mitgliedstypen | Privat / Firma / Verein / Gemeinde / Landwirt |
| PROJ-8 | Konfigurierbare Felder pro EEG | hidden / optional / required / admin_only |
| PROJ-9 | EEG-spezifische Rechtsdokumente | Pflicht-Checkbox |
| PROJ-11 | Konfigurierbarer Einleitungstext | Pro EEG |
| PROJ-12 | SEPA-Lastschriftmandat PDF | Basislastschrift bei Submit |
| PROJ-13 | Externe Registrierungs-API | API-Key-Auth (`/api/external/*`) |
| PROJ-14 | SEPA-Firmenlastschriftmandat | Variante B2B |
| PROJ-15 | Konfigurierbare Felder Erweiterungen | mehr Feldtypen |
| PROJ-16 | Cloudflare Turnstile Spam-Schutz | Public-Form |
| PROJ-17 | Excel-Export für eegFaktura-Import | Fallback-Pfad |
| PROJ-18 | Datenschutzerklärung + Central Policy Toggle | Operator-weit konfigurierbar |
| PROJ-19 | Manuelle Aktivierung der Registrierung | Per-EEG-On/Off |
| PROJ-20 | Vollständige Antragsdaten in EEG-Mail | Volltext-Notification |
| PROJ-21 | Beitrittsbestätigung als PDF | Originally → approved, since PROJ-46 Stage B → imported |
| PROJ-24 | OpenAPI/Swagger Dokumentation | `/api/swagger` |
| PROJ-25 | Bulk-Aktionen im Admin | Multi-Select-Operationen |
| PROJ-31 | E-Mail-Adresse-Bestätigung (Anti-Abuse) | Per EEG opt-in |
| PROJ-32 | EEG-Stammdaten aus Core | GraphQL Sync, Phase 1 ohne Logo |
| PROJ-33 | EEG-Logo aus Core | Deployed 2026-05-18 |
| PROJ-34 | Robuste Import-Recovery | Orphan-Fallback + Pre-Check + Unstuck-GUI |
| PROJ-35 | Per-EEG-Referenznummern | Format `<RC>-<Jahr>-<NNNN>` |
| PROJ-36 | Info-Dokumente ohne Checkbox | Auto-informational consent |
| PROJ-37 | Genossenschaftsanteile | Per-EEG-Konfig + Submit-Validation |
| PROJ-38 | Status-Modell-Hygiene + Audit-Fixes | PROJ-31 Follow-up |
| PROJ-39 | Titel-Nach + Bankname + per-MP-Adresse | Erweiterte Stammdaten |
| PROJ-40 | EEG-Umzuordnung im Admin-Review | Tenant-Switch ohne Re-Submit |
| PROJ-41 | Status-Change-Mails (Ablehnung, hard-fail) | Sync-pre-commit SMTP |
| PROJ-42 | E-Fahrzeug-Detailerfassung | Anzahl + Jahres-km |
| PROJ-43 | Status-Change-Mails (Info-Anfrage, hard-fail) | gebündelt mit PROJ-41 |
| PROJ-44 | Netzbetreiber-Vollmacht | Per-EEG opt-in |
| PROJ-45 | Erzeugungsform + Batterie + typabh. Sichtbarkeit | PV/Wind/Hydro/Biomasse |
| PROJ-46 | Stati für Import-Nachbereitung (Stage A–D) | awaiting_bank → ready → activated |
| PROJ-47 | B2B-SEPA-Mandat mit Mandatsreferenz beim Import | Mitgliedsnummer als Mandatsreferenz |
| PROJ-48 | SEPA-Default-Core + konfigurierbares Mandat-Timing | Submit- vs. Import-Time-Mandat |
| PROJ-49 | Energie-Felder pro Zählpunkt + Einspeiselimit | Refactoring von app-level zu meter-level |
| PROJ-52 | Konfigurierbarer Zählpunkt-Prefix pro Richtung | Mask-Lock + Auto-Pad + 2-6-5-20-Format + Alphanumerik + SEPA-Mandat-Datum |

### Approved (wartet auf Deployment-Bündelung)

| ID | Feature |
|----|---------|
| PROJ-27 | Tarif-Auswahl beim Import |
| PROJ-28 | Trennung Privat / Kleinunternehmer |
| PROJ-29 | IBAN-Eingabe mit visueller Gruppierung |
| PROJ-30 | Reset eines importierten Antrags auf approved |

### On Hold

| ID | Feature | Grund |
|----|---------|-------|
| PROJ-10 | Admin Notifications | Vertagung — niedrige Dringlichkeit |
| PROJ-22 | Tailwind CSS v3 → v4 | Revertiert 2026-04-26 wegen Regressionen; Retry braucht Stabilisierung der v4-Ecosystem-Updates |
| PROJ-23 | Stammdaten-Import aus eegFaktura-Excel | Ersetzt durch PROJ-32 (GraphQL-Sync) |
| PROJ-26 | Eigener Mailserver pro EEG | Geparkt 2026-05-18 |
| PROJ-50 | Zugang Online-Portal Netzbetreiber + bedingte Anleitungs-Mail | Geparkt 2026-05-18 — mehrere offene Fragen |
| PROJ-51 | Anzeige offener Nutzungsgebühren im Admin-UI | Wartet auf Klärung des Abrechnungs- und Status-Pflege-Konzepts |

> **Next available feature ID:** PROJ-53 (siehe `features/INDEX.md`).

## 5. Success Metrics

The product is successful when:

- new members can create and submit an application themselves
- admins can review and decide on applications without leaving the admin web
- approved applications import correctly into eegFaktura core
- no productive participant is created before explicit import
- B2B-SEPA-Pre-Notification cycle (PROJ-46/47) closes ohne manuelle SQL-Eingriffe
- the onboarding workflow remains simpler than the manual participant creation flow it replaces

## 6. Constraints

- The component follows the same technology direction as eegFaktura.
- Backend implemented in Go, frontend in Next.js (same stack as `eegfaktura-web`).
- Onboarding data uses the existing PostgreSQL database in a separate schema `member_onboarding`.
- Admin authentication via Keycloak.
- Import into eegFaktura core happens only through an internal service call (no direct DB writes).
- All product timestamps stored in UTC, rendered Europe/Vienna at display layer.
- Public form rate-limited (10 req / 10 min / IP) + optional Cloudflare Turnstile.
- Frontend image runs on Node 22 LTS; monthly EOL-Check workflow guards against silent expiry of Node / Go / Postgres.

## 7. Non-Goals

Version 1 explicitly does NOT include:

- document management (file uploads from members)
- role selection by member (admin assigns in core)
- tax data collection
- public account/login management
- direct writes into eegFaktura core tables
- bidirectional sync between onboarding and core for member data (one-way for EEG master data only — PROJ-32)
- automatic activation polling — admin triggers the activation-check batch button (PROJ-46 Stage D, user decision)
- self-service B2B-bank-confirmation by member — admin sets the status after the member contacts them (PROJ-46 Entscheidung A)

### Topics that have moved INTO scope since the MVP

- **Tariff selection** at import time — added in PROJ-27 (per-application admin choice).
- **Account/payment data** for SEPA mandates — added incrementally (PROJ-12 basislastschrift, PROJ-14 firmenlastschrift, PROJ-47 mandatsreferenz at import).
- **EEG-Stammdaten-Sync from core** — added in PROJ-32 (one-way, read-only direction).

## 8. Core Business Rules

- one application contains exactly one member
- one application belongs to exactly one EEG
- the public registration entry point is identified by the EEG's RC number
- the RC number is resolved through `member_onboarding.registration_entrypoint`; the onboarding backend never reads EEG data directly from eegFaktura core tables
- one application can contain multiple metering points
- a metering point may inherit the member's primary address (default) or carry its own deviating address (PROJ-39)
- only `approved` applications may be imported
- after a successful import the application auto-routes via `imported` (transient) to either `awaiting_bank_confirmation` (b2b) or `ready_for_activation` (non-b2b) — PROJ-46
- `activated` is a strict end state — no transitions out, no reset; deactivation must happen in the eegFaktura core
- tariff, role, and similar business details are completed later in eegFaktura

## 9. Notes for Feature Work

Use this PRD as the high-level product context.

Detailed implementation work is tracked via:

- [`features/INDEX.md`](../features/INDEX.md) — single source of truth for feature status
- individual feature specification files in [`features/`](../features/)
- [`CLAUDE.md`](../CLAUDE.md) for the binding architecture decisions and workflow conventions
- [`docs/architecture.md`](architecture.md), [`docs/domain-model.md`](domain-model.md), [`docs/api-spec.md`](api-spec.md), [`docs/import-mapping.md`](import-mapping.md) for technical reference
