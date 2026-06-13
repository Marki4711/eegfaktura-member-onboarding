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

Single source of truth for current implementation status is [`features/INDEX.md`](../features/INDEX.md). The table below mirrors that file at the time of writing (2026-05-19).

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
| PROJ-21 | Beitrittsbestätigung als PDF | Originally → approved, PROJ-46 Stage B → imported, PROJ-53 → activated |
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
| PROJ-46 | Stati für Import-Nachbereitung (Stage A–D) | ready_for_activation → activated *(PROJ-91 hat den Zwischenstop `awaiting_bank_confirmation` entfernt)* |
| PROJ-47 | B2B-SEPA-Mandat mit Mandatsreferenz beim Import | Mitgliedsnummer als Mandatsreferenz |
| PROJ-48 | SEPA-Default-Core + konfigurierbares Mandat-Timing | Submit- vs. Import-Time-Mandat |
| PROJ-49 | Energie-Felder pro Zählpunkt + Einspeiselimit | Refactoring von app-level zu meter-level |
| PROJ-52 | Konfigurierbarer Zählpunkt-Prefix pro Richtung | Mask-Lock + Auto-Pad + 2-6-5-20-Format + Alphanumerik + SEPA-Mandat-Datum |
| PROJ-53 | Aktivierungs-Modus pro EEG + Beitrittsbestätigung erst bei `activated` + manueller `approved → activated`-Skip | Modus A/B konfigurierbar, Mail-Verschiebung, Skip-Endpoint |
| PROJ-54 | Public/Private Repo-Split | Mirror-Workflow, `private/`-Pfad + `visibility: private`-Frontmatter |
| PROJ-56 | Netzbetreiber-Info-Seite im Beitrittsbestätigungs-PDF | Zwei optionale Felder (Kundennummer, Inventarnummer), nur sichtbar bei aktiver Vollmacht |
| PROJ-57 | Ansprechperson für Org-Mitgliedstypen | v3-Design: drei Subfelder einzeln per field_config, Checkbox erscheint automatisch — kein Master-Switch |
| PROJ-58 | Abweichende Rechnungs-E-Mail | Toggle + E-Mail in der Bankverbindungs-Section, nur bei Org-Mitgliedstypen |
| PROJ-59 | BgA / Hoheitsbereich-Vermerk im Anlagennamen | Reiner Hilfetext-Vermerk bei Gemeinden — kein Schema, keine Validierung; Admin liest beim Tarif-Setzen |
| PROJ-60 | Datenweiterleitung an externe Systeme | Async-Plugin-Framework mit Job-Queue + In-App-Worker; Excel/CSV-Plugin als erste Implementierung. Phase 2 = weitere Plugins (CRM, Zoho/HubSpot, …) |
| PROJ-61 | Konfigurations-Export & -Import pro EEG | Vier Sub-Typen (EEG-Settings, Field-Config, Legal-Documents, Data-Export-Configs) als versionierte JSON-Datei + Diff-Preview |
| PROJ-62 | Mitgliedstypen Kleinunternehmer + Unternehmen zusammenführen | `sole_proprietor` entfällt; `company` mit optionaler UID = Kleinunternehmerregelung |
| PROJ-63 | USt-Pflicht-Checkbox bei Unternehmen + Verein | UI-Gate für UID-Eingabe, kein DB-Feld |
| PROJ-64 | Faktura-Handover-Billing-Trigger | `application.faktura_handover_at` deckt /import UND /export/excel — Excel-Bypass für Verrechnung geschlossen |

### Approved (wartet auf Deployment-Bündelung)

| ID | Feature |
|----|---------|
| PROJ-27 | Tarif-Auswahl beim Import |
| PROJ-28 | Trennung Privat / Kleinunternehmer |
| PROJ-29 | IBAN-Eingabe mit visueller Gruppierung |
| PROJ-30 | Reset eines importierten Antrags auf approved |

### In Progress

| ID | Feature | Stand |
|----|---------|-------|
| PROJ-66 | Settings-Auto-Save + Tab-Switch-Schutz | Implementiert auf main 2026-05-30; wartet auf Deployment-Bündelung |

### Planned

| ID | Feature |
|----|---------|
| ~~PROJ-65~~ | ~~Vorstands-Signaturblock im Beitrittsbestätigungs-PDF~~ — Superseded durch PROJ-76 |
| PROJ-67 | Basic-/Advanced-Modus für Einstellungen — reduzierte Sicht für kleine EEGs |
| PROJ-69 | Reconciliation-basierter Billing-Backstop — Free-Rider-Detection via periodischem Core-Match (IBAN+E-Mail), setzt `faktura_handover_at` rückwirkend |
| PROJ-70 | Stammdaten-Resync für aktivierte Anträge — On-Demand-Pull von Core-Werten pro Antrag, bei IBAN-/Kontoinhaber-Wechsel SEPA-Mandat-Invalidierung + Rückfall auf `ready_for_activation` *(PROJ-91 hat `awaiting_bank_confirmation` entfernt; Resync-Rückfall geht jetzt auf den verbleibenden Vor-Aktivierungs-Status)* |
| PROJ-71 | EEG-Customer-Onboarding-Formular + AVV-PDF + Auto-Antwort-Mail — Self-Service-Anmeldung für zahlende EEG-Kunden (Phase A der Customer-Onboarding-Pipeline) |
| PROJ-72 | Member-Onboarding-Cockpit — Owner-EEG-Übersicht aller EEGs mit Live-KPIs (Aktiv-Badge, Customer-Onboarding-State, Anträge-Pipeline) und Direkt-Links zu Anträgen & Einstellungen — **Phase 1B vor Prod** |
| PROJ-104 | Abrechnung der Plattform-Nutzung (Pricing-V3) — Standard/Pro über `settings_view_mode` differenziert, Preis pro aktiviertem Mitglied, quartalsweise, FreeFinance + Mollie, K8s-CronJob, globaler `BILLING_LIVE_MODE`-Schalter (Test-Phase ohne Zahlungspflicht) — **Phase 1A vor Prod** |
| PROJ-73 | Cleanup: verwaisten EEG-Toggle `use_company_sepa_mandate` entfernt — Domain-Logik seit PROJ-48 funktionslos; Settings-UI aufgeräumt, Migration 000066 |
| PROJ-74 | B2B-Mandat-Gate-Fix — `buildSEPAMandateData` lässt B2B-Anträge auch bei `SEPAMandateEnabled=false` durch (SEPA-Rulebook), Hart-Fail bei fehlenden Stammdaten, UI-Klarstellung an den SEPA-Toggles |
| PROJ-75 | SEPA-Einwilligungs-Checkbox in Bankverbindungs-Card verschoben, mit EEG-spezifischem Text + Creditor-ID |
| PROJ-76 | Vorstands-Genehmigungs-Workflow für Beitrittserklärung — per-EEG-Toggle, eigenes PDF mit Vorstands-Signaturblock, Mail-Routing-Wechsel und On-Demand-Download im Admin-UI; supersedes PROJ-65 |
| PROJ-77 | B2B-Mandat-Audit-Block — elektronische SEPA-Zustimmung wird als formfreie Willenserklärung (§ 76 (3) EIWOG) im Firmenlastschrift-PDF dokumentiert (Tenant, Zustimmungs-Zeitstempel, IP); ersetzt den klassischen Unterschriftsblock für Anträge mit IP-Erfassung |
| PROJ-78 | Toggle „Elektronisches SEPA-Mandat" (B2B + CORE separat) — zwei unabhängige Per-EEG-Schalter steuern Audit-Trail-Variante vs. klassischer Unterschriftsblock pro Mandat-Typ; CORE-Audit neu, B2B-Audit (PROJ-77) hinter Toggle; Default beide FALSE (klassisch) bis Rechtsklärung |
| PROJ-79 | B2B-Import als CORE in eegFaktura-Core — bei `einzugsart=b2b` wird im Faktura-Core trotzdem SEPA-Typ CORE angelegt, um die Bank-Klärungsphase der B2B-Mandatsvereinbarung ohne Risiko fehlgeschlagener Erst-Lastschriften zu überbrücken; Aktivierungs-Mail an EEG-Kontakt mit Hinweis zur eigenständigen Bank-Klärung und manuellen Umstellung auf B2B im Core nach Bestätigung; hartkodierte globale Regel |
| PROJ-80 | SEPA-Settings-Vereinfachung — `sepaMandateEnabled`-Toggle entfernt; Online-Zustimmung + CORE-PDF werden Konstanten; nur noch CORE-Audit + Timing als Konfig; Timing-Label umbenannt; Migration-Backfill für Bestands-EEGs; EEG-Mail-Kopie bei Audit-Variante (Ablage-Pflicht); Konsistenz-Anpassung Kurz-Erklärungen unter allen SEPA-Toggles |
| PROJ-81 | SEPA-Einwilligung optional pro Mitgliedstyp — Per-EEG-Master-Toggle + konfigurierbare Mitgliedstyp-Liste (private/association/municipality, `company` zwangsweise ausgenommen). SEPA-Einwilligungs-Checkbox wird im Public-Form für gewählte Mitgliedstypen optional; bei Nicht-Anhaken werden Bankdaten ebenfalls optional und `einzugsart` auf `kein_sepa` gesetzt; Backend-Defense-in-Depth-Validation |

### On Hold

| ID | Feature | Grund |
|----|---------|-------|
| PROJ-10 | Admin Notifications | Vertagung — niedrige Dringlichkeit |
| PROJ-22 | Tailwind CSS v3 → v4 | Revertiert 2026-04-26 wegen Regressionen; Retry braucht Stabilisierung der v4-Ecosystem-Updates |
| PROJ-23 | Stammdaten-Import aus eegFaktura-Excel | Ersetzt durch PROJ-32 (GraphQL-Sync) |
| PROJ-26 | Eigener Mailserver pro EEG | Geparkt 2026-05-18 |
| PROJ-50 | Zugang Online-Portal Netzbetreiber + bedingte Anleitungs-Mail | Geparkt 2026-05-18 — mehrere offene Fragen |
| PROJ-51 | Anzeige offener Nutzungsgebühren im Admin-UI | Wartet auf Klärung des Abrechnungs- und Status-Pflege-Konzepts |
| PROJ-55 | Nachmelden von Zählpunkten anhand der Mitgliedsnummer | Wartet auf Self-Service-Portal-Direction (Owner-Entscheidung 2026-05-23) |

> **Next available feature ID:** PROJ-82 (siehe `features/INDEX.md`).

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

### Scope boundary (2026-05-30 owner direction, repeated tester question)

Member-Onboarding is a **data-capture, review, and handover pipeline for membership applicants** — explicitly not:

- **Not a member management system.** Once an application is handed off (typically via Core import or plugin-based data forwarding), the persistent member record lives in the target system. Address, bank, tariff, and contract changes for existing members happen there, not here.
- **Not a long-term data store.** Application records remain available for audit and reset, but are intentionally not the source of truth. Long-term storage belongs in the target system.
- **Not a reporting or analytics tool.** No dashboards, no BI module, no member queries by filter sets. Reporting belongs in the system that received the data.

This bounded scope is deliberate. Feature requests pointing toward member management, reporting, or persistent member-side data live outside this product and should be redirected — the target system covers them already.

Pilot framing of the concrete value: **„a simple form mask that gets new members cleanly into eegFaktura while avoiding the typical error sources of manual entry."** Structured intake + one-click import + integrated communication + audit trail to handover.

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
- after a successful import the application auto-routes via `imported` (transient) directly to `ready_for_activation` for all einzugsarten — PROJ-46 + PROJ-91
- `activated` is a strict end state — no transitions out, no reset; deactivation must happen in the eegFaktura core
- tariff, role, and similar business details are completed later in eegFaktura

## 9. Notes for Feature Work

Use this PRD as the high-level product context.

Detailed implementation work is tracked via:

- [`features/INDEX.md`](../features/INDEX.md) — single source of truth for feature status
- individual feature specification files in [`features/`](../features/)
- [`CLAUDE.md`](../CLAUDE.md) for the binding architecture decisions and workflow conventions
- [`docs/architecture.md`](architecture.md), [`docs/domain-model.md`](domain-model.md), [`docs/api-spec.md`](api-spec.md), [`docs/import-mapping.md`](import-mapping.md) for technical reference
