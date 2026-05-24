# Changelog

Alle nennenswerten Änderungen an diesem Projekt werden hier dokumentiert.
Format basiert auf [Keep a Changelog](https://keepachangelog.com/de/1.0.0/).

> Die Versionsnummern im CHANGELOG sind unabhängig von den Git-Tags vergeben,
> da die ursprünglichen Tags nicht konsistent nummeriert wurden.

---

## [Unreleased]

### Welle 11 — Severity-Drift + Tot-Code in metrics *(2026-05-24)*

Sub-Tickets **3d + 3e** aus AUDIT-TODO. Reine Cleanup-Welle.

- `internal/metrics/metrics.go`: `statusClassFromString()` (toter Helper
  mit `var _ = ...`-Suppressor) gelöscht; ungenutzten `strconv`-Import
  mit entfernt.
- 3 `slog.Error` → `slog.Warn` umgestellt, wo der Caller noch
  Kontext-/Recovery-Möglichkeit hat:
  - `internal/dataexport/worker.go:135` (Pickup-DB-Fehler, retried im
    nächsten Tick)
  - `internal/mail/service.go:603` (EEG-Template-Render-Fail, Member-Mail
    läuft separat weiter)
  - `internal/application/admin_service.go:993` (PDF-Gen-Fail wird per
    Flag an SendActivationNotification gereicht, Mail geht ohne Attachment)
- Konvention etabliert: `slog.Error` nur für Pfade ohne weiteren
  Caller-Kontext.

§4b (composite-Index) bewusst nicht angefasst — Audit-Eintrag sagt
selbst „nicht ohne EXPLAIN-Daten"; verschärft §4c (Write-Amplification
auf der 14-Index-Tabelle). Wandert in §4a-Folge (Operator-Action).

### Welle 10 — E2E-Auth-Fixture (Header-basierte Test-Claims) *(2026-05-24)*

Sub-Ticket **5h** aus AUDIT-TODO. Schaltet authenticated-Pfade in
CI-Tests frei, ohne dass Keycloak in CI laufen muss.

- **Backend** (`internal/http/auth_middleware.go`): neue Middleware
  `TestHeaderAuthMiddleware()` liest synthetische Claims aus
  Request-Headern:
  - `X-Test-Tenant: RC123,RC456` → Tenant-Admin
  - `X-Test-Superuser: true` → Superuser-Realm-Rolle
  - `X-Test-Subject: <id>` → optionaler Subject
  - Beide leer → 401 (Tests können auth-required asserten)
  - Nur ein Tenant ohne Superuser → 403 (genau wie produktiv-Middleware)
- **`cmd/server/main.go`**: aktiviert die Test-Middleware wenn
  `TEST_AUTH_MODE=headers`, ersetzt dann `KeycloakAuthMiddleware`.
  Sicherheitsguard: `log.Fatalf` wenn `ENVIRONMENT=production` mit
  diesem Flag — die `X-Test-*`-Header sind triviale Forgery.
  `slog.Warn` zum Startup, damit der Modus im Audit-Log sichtbar ist.
- **Tests** (`internal/http/test_header_auth_test.go`): 4 Go-Unit-Tests
  decken alle Modi ab (ohne Header → 401, Tenant-Header → 200, Superuser
  → 200, Custom-Subject).
- **Frontend** (`tests/helpers/auth.ts`): `adminAuthHeaders()`,
  `tenantAdminHeaders()`, `superuserHeaders()`-Conveniences. Smoke-Spec
  `tests/helpers-auth.spec.ts` mit 3 Tests gegen den neuen Mode.
- **CI** (`.github/workflows/ci.yml`): `TEST_AUTH_MODE: headers`
  env-Var im `e2e`-Job; aktiviert damit die Test-Middleware.
- **PROJ-17** AC-BE1 und AC-BE5: `test.skip(CI)` entfernt — die Tests
  prüfen jetzt korrekt 401 ohne Header.

### Welle 9 — Playwright in CI + `skipIfBackendDown`-Konsolidierung *(2026-05-24)*

Sub-Ticket **5a + 5i** aus AUDIT-TODO (Audit-Marathon-Restschuld).

- `.github/workflows/ci.yml`: neuer `e2e`-Job mit Postgres-17-Service,
  `migrate -direction=up`, `dev_seed.sql`, Backend (Go) + Frontend
  (Next.js production-build) als Background-Prozesse mit `/health`- bzw.
  `/`-Polling, Playwright-Browser-Cache und Report-Artifact-Upload bei
  Failure.
- PR-CI läuft Chromium-only über neue `PLAYWRIGHT_BROWSERS=chromium`
  ENV-Variable (124 statt 496 Tests); Multi-Browser-Matrix (Firefox +
  WebKit + Mobile Safari) bleibt lokal Default und wandert in einen
  zukünftigen nightly-Workflow (eigenes Sub-Ticket).
- `playwright.config.ts`: `webServer` deaktiviert wenn `process.env.CI`,
  weil der Workflow Backend + Frontend selbst startet; Reporter in CI
  zusätzlich `list` (Stream-Output).
- Acht duplizierte `skipIfBackendDown`-Helper in den Spec-Dateien
  (PROJ-11 bis -17, PROJ-25) durch konsolidierten Import aus
  `tests/helpers/backend.ts::ensureBackendUp` ersetzt. Akzeptiert sowohl
  `Page` als auch `APIRequestContext`. In CI (`process.env.CI === 'true'`)
  hart-fail statt skip — verhindert grüne Test-Runs bei totem Backend.
- 12 latent-brittle Tests in 6 Spec-Files mit
  `test.skip(process.env.CI === "true", "AUDIT-TODO §5b/5h: …")`
  getaggt. Failure-Modi:
  - **§5b (Seed-Inadequacy)**: PROJ-7/8/9/11/12/14 — Tests setzen
    reichere Settings/Configs voraus, die der minimal-seed
    (`RC123456 / is_active=TRUE`) nicht liefert. UI rendert nicht
    wie erwartet (z.B. Combobox "Mitgliedstyp" fehlt).
  - **§5h (Auth-Fixture)**: PROJ-17 (AC-BE1/BE5) — erwarten 401,
    bekommen in CI 200, weil `KEYCLOAK_JWKS_URL` leer ist.
  Lokal mit echtem Backend laufen die Tests weiterhin.
- Verbliebene Sub-Tickets: 5b–5f (fehlende E2E-Specs +
  Seed-Erweiterung), 5g (MailHog), 5h (Auth-Fixture / Test-Token),
  5j (`networkidle` → `waitForResponse`),
  Nightly-Multi-Browser-Workflow.

### PROJ-60 — Datenweiterleitung an externe Systeme (async Plugin-Framework + Excel/CSV-Plugin) *(2026-05-23)*

Komplett neues asynchrones Framework für die Weitergabe importierter
Mitglieder an externe Systeme. V1 ships das Excel/CSV-Export-Plugin;
Phase 2 (Zoho, HubSpot, …) baut ohne Framework-Eingriff auf.

**DB-Schema (Migration 000052):**
- `data_export_config` — Plugin-Konfigurationen pro EEG, Soft-Delete via `deleted_at`, UNIQUE auf `(rc_number, name)` WHERE non-deleted
- `data_export_job` — Async-Job-Queue + langlebiger Audit-Trail, mit `config_snapshot` (immune gegen Config-Edits zur Laufzeit) und 4 spezialisierten Partial-Indizes (Pickup, Concurrency-Check, BackOffice-Liste, Zombie-Scan)
- `data_export_result` — Datei-BLOBs mit 24 h TTL, FK CASCADE auf Job

**Backend:**
- Plugin-Registry mit Side-Effect-Import (`sql.Driver`-Pattern) — neue Plugins via einem Import in `cmd/server/main.go`
- In-App-Worker-Pool (3 Goroutines, 5 s Polling) mit `SELECT ... FOR UPDATE SKIP LOCKED` — multi-replica-safe
- Worker-Shutdown vor HTTP-Shutdown (`Worker.Stop(ctx)` mit 60 s Budget) — keine Zombie-Jobs mehr bei Rollouts; Helm-Template `terminationGracePeriodSeconds: 120`
- K8s-CronJob `data-export-cleanup` (`*/10 * * * *`): Zombie-Recovery + BLOB-TTL + DSGVO-Hard-Delete nach 7 J
- 12 neue Admin-Endpoints unter `/api/admin/data-export/*` (Plugins-Liste, Configs CRUD, Preview, Jobs CRUD inkl. Listing, Download, Retry)
- DSGVO: `slog.Info classification=sensitive-export` bei IBAN/Geburtsdatum-Exports; CSV/Excel-Injection-Defense für Werte mit Prefix `=+-@\t\r` (auch nach Leading-Whitespace/NBSP/BOM)
- Filename-Schema `{rc_number}-{config_name}-{YYYY-MM-DD}.{xlsx|csv}` mit Path-Traversal-Sanitization
- FailureMailer-Adapter sendet Plain-Text-Mail an `registration_entrypoint.contact_email` mit Job-Details + BackOffice-Link
- Batch-Loader (`GetByIDs` / `GetByApplicationIDs`) — N+1 eliminiert für 1000-Apps-Bulks
- 30 Unit-Tests im Excel-Plugin (ValidateConfig, formatValue, Renderer, Process, sanitiseSpreadsheetValue inkl. Whitespace-Bypass-Edge-Cases)

**Frontend:**
- Settings-Page jetzt mit shadcn Tabs (6 Sektionen statt langer Liste): Stammdaten | Einleitungstext | Formular-Felder | Rechtsdokumente | Externe API | Datenweiterleitung
- Excel-Editor mit Spalten-Mapping (Header/Feld/Format/Up-Down-Remove), Live-Preview (debounced, skipped bei unvollständigen Spalten), 3 Standard-Vorlagen (Newsletter, CRM-Stammdaten, Buchhaltung), DSGVO-Popover bei IBAN/Geburtsdatum, alphabetische Sortierung pro Kategorie im Feld-Dropdown
- Trigger-Dialog (einstufige Plugin-Konfig-Liste), Bulk-Action in Antragsliste mit Cross-EEG-Schutz, Single-Action im Antrags-Detail
- Polling-Modal (2 s/5 s) mit Progress-Bar, Download bei Done, Retry bei Failed (Retry-Polling re-subscribed via `onRetried`-Callback)
- BackOffice-Jobs-Tab mit Failed-Badge (7 Tage), Status-Filter, Cursor-Pagination
- Aussagekräftige Fehlermeldungen via `formatValidationError` (Backend-`fields`-Map wird ausgepackt, Pfade wie `columns[1].header` prettifiziert zu „Spalte 2 → Spaltenkopf")

**Bugfix-Welle parallel zu PROJ-60:**
- `persons_in_household` ist konzeptuell nur für `private` und `farmer` sinnvoll → Backend `clearMemberTypeFields` cleart bei Org-Typen, Required-Check zusätzlich auf `isNaturalPerson` gegated, Public-Form rendert das Feld nicht mehr für Org-Typen, Admin-Field-Config-Editor zeigt zwei Badges (`consumption` + `natural_person`)
- Verein-Submit funktioniert wieder bei EEGs, die `persons_in_household` als Pflichtfeld konfiguriert hatten
- `jobs-list` functional setState verhindert Race bei „Mehr laden" + Filter-Wechsel
- Placeholder-Verstoß im Bulk-Reject-Dialog entfernt (Label trägt jetzt den Hinweis)

**Audit-Welle 2 (Re-Audit-Findings, 2026-05-23):**
- Worker-Shutdown ohne TriggerJob-Race: neuer `JobService.MarkShuttingDown()` (atomic.Bool) wird in `main.go` vor `workerCancel()` gerufen; TriggerJob/Retry returnen 409 während Drain — keine Zombie-Jobs mehr durch hastige Admins
- `LoadForExport` hard-failt jetzt wenn alle Apps via Tenant-Filter rausfliegen (vorher: silent leerer Export wurde als „done" markiert)
- ListJobs N+1 eliminiert: neue Repo-Methode `GetMetadataByJobIDs` reduziert ein 1+N-Listing-Query-Pattern auf 1+1
- Retry-Modal: `onRetried`-Prop ist jetzt required (TypeScript-enforced) — Tech-Debt-Trap (silent polling-freeze nach Retry) geschlossen
- K8s-Hardening auf den zwei PROJ-60-Templates: `automountServiceAccountToken: false` + `seccompProfile: RuntimeDefault` (lateral movement bei Container-Compromise blockiert)
- 12 Swag-Annotationen für `internal/http/dataexport.go` (vorher: `swag init` skippte PROJ-60 silent)
- Doku-Hygiene: domain-model.md Section-Nummern §3.6/3.7/3.8 → §3.9/3.10/3.11 (Kollision mit `document_consent`/`external_api_key`/`reference_number_counter` aufgelöst); CHANGELOG/TODO „11 Endpoints" → „12"

**Audit-Welle 3 (Re-Re-Audit-Folge, 2026-05-24):**
- `CountFailedSince`-Fehler im Jobs-Listing wird jetzt geloggt (vorher silent geswallowed → Badge zeigte stille 0)
- `LoadForExport`-Fehlermessage entfernt die RC-Nummer aus dem User-Error (defensiver gegen Cross-Tenant-Info-Leak)
- TODO-docs-sync.md-Drift gefixt (alte §3.6/3.7/3.8-Referenz)

**Helm-Deep-Audit-Welle 4 (2026-05-24):**
- **Produktiver Bug behoben**: `data-export-cleanup`-CronJob (PROJ-60) war nicht in der Postgres-NetworkPolicy-Allowlist → wäre bei strikten CNIs (Calico/Cilium) bei jedem Run gescheitert. Vierter `podSelector`-Eintrag ergänzt.
- Konsistente Härtung über alle 8 Pod-Workloads: `seccompProfile: RuntimeDefault`, `automountServiceAccountToken: false` (wo möglich), `readOnlyRootFilesystem: true`, drop-ALL-capabilities, `allowPrivilegeEscalation: false` (vorher nur backend + data-export-cleanup gehärtet)
- Postgres: livenessProbe (höheres `failureThreshold` als readiness), `terminationGracePeriodSeconds: 60` für sauberen smart-shutdown, cpu-Limit
- Frontend: startupProbe für Next.js Cold-Start (failureThreshold 30 × 2 s), tmp-emptyDir für readOnlyRootFilesystem
- Ingress: `ssl-redirect` + HSTS (180 d) + X-Content-Type-Options + Referrer-Policy + X-Frame-Options + proxy-body-size 10 MB
- Namespace: PSA `restricted` enforced + audit + warn (defensive: zukünftige nicht-konforme Workloads werden vom API-Server rejected)
- seed-job: SQL-Injection-Vektor geschlossen — `values.seed.*` werden jetzt als Env-Vars in psql gesetzt + `\set` + `:'name'`-Safe-Quoting statt Template-Inline-Interpolation
- restart-cronjob: `startingDeadlineSeconds: 300`, Pod- und Container-Security-Context
- `_helpers.tpl`: `app.kubernetes.io/version`-Label auf `.Chart.AppVersion` statt Backend-Image-Tag (war für Postgres/Frontend/CronJob irreführend)
- **Resource-Requests minimiert** (Owner-Entscheidung): backend 50→10 m, postgres 100→25 m, frontend 100→25 m, jobs 10→5 m. Cluster-Sizing-Kosten gering halten solange wir noch nicht produktiv sind. Limits bleiben großzügig für Peak.

Bewusst aufgeschoben in `docs/AUDIT-TODO.md` 5a–5f: TLS-Block (cert-manager), SealedSecrets-Migration, Egress-NetPol, HA + PDB, HPA, Postgres-Backup-Doku.

**Observability-Audit-Welle 5 (2026-05-24):**
- **CRITICAL — PII-Leak in Logs gefixt**: `internal/mail/service.go` loggte `app.Email` + `entrypoint.ContactEmail` voll an 5 Stellen (Verstoß gegen `.claude/rules/security.md`: „IBAN, email, phone, name must not appear in application logs"). Neue `emailDomain()`-Helper-Funktion gibt nur den `@suffix` zurück; alle 5 Stellen umgestellt auf Log-Key `to_domain`.
- Neues Paket `internal/logfields/` zentralisiert slog-Field-Keys (`RCNumber`, `JobID`, `ApplicationID`, `Classification`, `AdminUserID`, …) plus fixiertes `classification`-Vokabular (`pii-read`, `pii-export`, `sensitive-export`). Verhindert Drift (`"rc"` vs `"rc_number"`, `"user_id"` vs `"admin_user_id"`); neue Code-Stellen sollen importieren statt Strings tippen.
- DSGVO-Audit-Trail-Marker auf zwei weitere PII-Pfade ausgeweitet: `GetApplicationDetail` (`classification=pii-read`) und `ExportApplicationExcel` (`classification=pii-export`). Pendant zu PROJ-60 `sensitive-export`. Log-Shipper können auf `classification=`-Vokabular filtern und an die Compliance-Archivierung routen.
- `internal/dataexport/worker.go` Sensitive-Export-Marker nutzt jetzt die `logfields`-Konstanten statt Literals.

Bewusst aufgeschoben in `docs/AUDIT-TODO.md` 3a–3e (eigene PROJs):
- 3a: 10 neue Prometheus-Metrics (`coreclient_request_duration`, `dataexport_jobs_total`, `_job_duration`, `_queue_depth`, `_workers_busy`, `_blob_bytes_total`, `_cleanup_runs_total`, `turnstile_verifications_total`, `applications_submitted_by_type_total`, `smtp_send_duration_seconds`)
- 3b: OpenTelemetry-Tracing-Bootstrap mit 4 Stufen (CoreClient, DataExport-Pipeline, Trace-Log-Correlation, K8s-Collector)
- 3c: Logger-Context-Middleware (`slog.With("request_id", ...)` im ctx, Helper `log.FromCtx`)
- 3d: Tot-Code `metrics/metrics.go:statusClassFromString` aufräumen
- 3e: Severity-Drift bereinigen (3 Stellen `slog.Error` → `slog.Warn` für transiente/Caller-Kontext-Pfade)

**Data-Model-Slimming-Audit-Welle 8 (2026-05-24, Migrationen 000054 + 000055):**
- **DROP `application.reviewed_by_user_id`** (Migration 000054) — echtes Tot-Datum, war via COALESCE in `UpdateStatusAdminTx` gesetzt + ins JSON serialisiert, aber nirgends im Code konsumiert. Audit-Quelle für „wer hat Status geändert" ist `status_log.changed_by_user_id`. Begleitend: `UpdateStatusAdminTx`-Signatur entfernt den `reviewedByUserID`-Parameter; 3 Caller in `admin_service.go` + `importing/import_service.go` angepasst (system-actor landet weiterhin in `status_log`).
- **DROP `application.email_confirmation_used_at`** (Migration 000055) — 100 % redundant zu `email_confirmed_at` (`MarkEmailConfirmedTx` setzte beide auf denselben NOW(); Idempotenz-Check `application_service.go:825` funktional identisch, wurde auf `EmailConfirmedAt != nil` umgestellt). Down-Migration backfillt aus `email_confirmed_at`.
- Doku-Patches in `docs/domain-model.md` für zwei bewusste Trade-offs: §3.10 `application_ids UUID[]` als bewusste Ausnahme zur „no JSON columns"-Regel (Snapshot-Charakter) + §3.9 `is_obsolete` als bewusst materialisiertes Cache-Boolean (Registry runtime-only).

**Bestätigt OK, nicht angefasst** (waren TODO-Verdacht, alle legitim): `accuracy_confirmed`, `privacy_version`, `has_contact_person`/`has_billing_email`, `processed_count`/`total_count`/`retry_count` als INT, `field_config` als sparse-table, alle PROJ-46-Lifecycle-Timestamps.

**E2E-Test-Coverage-Audit-Welle 7 (2026-05-24):**
- Browser-Matrix erweitert in `playwright.config.ts`: Desktop-Firefox + Desktop-WebKit (Safari-Engine) ergänzt; vorher nur Chromium + Mobile-Safari.
- Neue Helper `tests/helpers/test-data.ts` mit `uniqueEmail()`/`uniqueRef()`/`TEST_RC_NUMBER`. Verhindert Akkumulations-Flakes durch fixed-string-Collisions (`test@example.at` etc.) und nutzt `@e2e.local` (RFC 6761-reserviert, kann nicht resolven).
- API-Vertrag-Drift in `tests/PROJ-12-sepa-mandate-pdf.spec.ts:156` gefixt: Backend liefert `active` (per `shared.RegistrationConfig`), Spec hatte `isActive` → `toHaveProperty` lief silent grün gegen nicht-existente Property.

Coverage-Score nach Audit: 11 von ~50 Deployed/Approved-PROJs haben eine Spec; 4 davon mit Voll-Coverage (PROJ-7/8/9/11/15), Rest sind Smoke/Auth-Wand-Tests. Top-5-Lücken: PROJ-1 Happy-Path, PROJ-31 Email-Confirmation, PROJ-46/53 Post-Import-Stati, PROJ-60 Data-Export, PROJ-2 Status-Transition-Matrix.

Bewusst aufgeschoben in `docs/AUDIT-TODO.md` 5a–5j (eigene PROJs):
- 5a **CRITICAL** — Playwright in CI aktivieren (eigener Job mit Postgres-Service-Container + globalSetup); ohne den verrotten Specs ungemerkt
- 5b–5f: die fünf priorisierten fehlenden Top-Specs
- 5g: MailHog/Mock-SMTP für Mail-Assertions
- 5h: Auth-Fixture (Test-Token / NODE_ENV=test-Bypass)
- 5i: `skipIfBackendDown` → hart-fail in CI
- 5j: `waitForLoadState("networkidle")` → `waitForResponse(...)` an 10 Stellen

**DB-Performance-Audit-Welle 6 (2026-05-24, Migration 000053):**
- **HIGH**: fehlender Index auf `external_api_key.key_hash` → jeder externe API-Call (Bearer `moak_*`) machte Seq-Scan. Neuer Partial-Index `WHERE revoked_at IS NULL` (widerrufene Keys werden ohnehin 401 abgewiesen).
- **LOW-Cleanup**: zwei redundante Plain-B-Tree-Indizes gedroppt — `idx_application_reference_number` und `idx_registration_entrypoint_rc_number` waren Duplikate von UNIQUE-Constraints (Postgres legt für UNIQUE automatisch einen Index an). Spart Write-Amplification.
- `docs/domain-model.md` §3.7 ergänzt um den neuen Partial-Index.

Bewusst aufgeschoben in `docs/AUDIT-TODO.md` 4a–4c:
- 4a: EXPLAIN-ANALYZE gegen Prod-DB für 6 Hot-Path-Queries (Operator-Action) + `pg_stat_user_indexes`-Auswertung nach 30 Tagen Prod-Laufzeit (DROP-Index-Kandidaten finden)
- 4b: `idx_application_submitted_at` ggf. durch composite `(rc_number, submitted_at DESC)` ersetzen, falls EXPLAIN das nahelegt
- 4c: Write-Amplification auf `application` (14+ Indizes) im Auge behalten

### PROJ-57 v3 — Ansprechperson ohne Master-Switch, drei Felder einzeln steuerbar *(2026-05-21)*

Vereinfachung des Konfigurations-Modells: der separate
`contact_person`-Master-Switch entfällt. Stattdessen werden alle drei
Felder (`contact_person_name`, `contact_person_email`,
`contact_person_phone`) einzeln per field_config konfigurierbar
(hidden/optional/required). Die Ansprechperson-Checkbox im Public-
Formular erscheint automatisch, sobald mindestens eines der drei
Felder nicht hidden ist.

- field_config: `contact_person` entfernt; neuer Eintrag
  `contact_person_name`; Defaults aller drei Felder = hidden
  (Feature aus, bis EEG aktiv konfiguriert)
- Backend: neuer Helper `contactPersonEnabled(fieldConfig)`,
  `clearContactPersonIfDisabled` cleart bei allen-drei-hidden,
  Required-Validierung pro Feld nur bei state=required
- Public-Formular: Checkbox-Sichtbarkeit aus den drei Sub-Feldern
  abgeleitet; Name-Feld auch konditional renderbar; Pflicht-Marker
  dynamisch
- Admin-Field-Config-Editor: zeigt jetzt drei Org-Typen-Einträge
  statt vier (Master-Switch + 2). Die Reihenfolge folgt der
  natürlichen Form-Reihenfolge (Name → Email → Telefon).

Hinweis für bestehende Konfiguration: alte EEGs mit `contact_person`-
Eintrag in der DB werden vom System ignoriert. Sie müssen die drei
Subfelder neu konfigurieren, um das Feature wieder zu aktivieren.

### PROJ-58 — Abweichende Rechnungs-E-Mail für Org-Mitgliedstypen *(2026-05-21)*

Bei Unternehmen, Vereinen und Gemeinden kann jetzt eine separate
E-Mail-Adresse für den Rechnungsversand angegeben werden. Per
Checkbox in der Bankverbindungs-Sektion aktivierbar.

- Zwei neue Spalten auf `application` (Migration 000051):
  `has_billing_email` (BOOL) + `billing_email` (TEXT)
- field_config-Eintrag `billing_email` (Default `hidden`,
  per-EEG konfigurierbar)
- Public-Form: Checkbox + Input in Bankverbindungs-Card, nur bei
  Org-Mitgliedstypen UND field_config != hidden. Required bei
  aktivem Toggle + Email-Format-Check.
- Admin-Detail-View + Admin-Edit-Form: Toggle + Email editierbar
  für Org-Mitgliedstypen
- Beitritts-PDF: zusätzliche Zeile „Rechnungs-E-Mail:" in der
  Bankverbindungs-Sektion, wenn gesetzt
- Server-Side-Cleanup: `clearBillingEmailIfDisabled` cleart die
  Felder auf NULL bei Toggle-off, nicht-Org-Mitgliedstyp oder
  field_config=hidden

Vorbereitung für das künftige eigene Rechnungsmodul. Versand-Logik
folgt mit dem Billing-Modul, kein automatischer Mail-Versand jetzt.

### PROJ-57 v2 — feiner steuerbare Ansprechperson-Pflichtigkeit *(2026-05-21)*

Erweiterung der Ansprechperson-Logik aus PROJ-57: Email und Telefon
können seit dieser Version pro EEG einzeln auf `hidden | optional |
required` gestellt werden. Name bleibt fix Pflicht wenn Toggle aktiv
(ohne Name keine sinnvolle Ansprechperson).

- Zwei neue field_config-Einträge: `contact_person_email` und
  `contact_person_phone`, beide Default `required` (= bisheriges
  Verhalten unverändert für bereits konfigurierte EEGs)
- Im Admin-Field-Config-Editor mit „Org-Typen"-Badge sichtbar
- Public-Form rendert das jeweilige Feld nur, wenn nicht hidden,
  und passt Pflicht-Marker (*) dynamisch an
- Server-Cleanup in `clearContactPersonIfDisabled` setzt das Detail-
  Feld auf NULL, wenn der EEG-State `hidden` ist — Schutz vor
  forged Clients
- Email-Format wird auch bei `optional` geprüft, falls Wert da
- Admin-Edit-Form sieht weiterhin alle drei Felder durchgehend
  (Admin-Korrektur-Pfad nicht eingeschränkt; Backend cleart bei hidden)

### PROJ-57 — Ansprechperson für Org-Mitgliedstypen *(2026-05-21)*

Optionale Ansprechperson für Unternehmen, Vereine und Gemeinden. Toggle-
Checkbox aktiviert drei zusätzliche Felder (Name, E-Mail, Telefon), die
in PDF, Submission-Mail und Admin-UI durchlaufen.

Eckdaten:

- **Vier neue Spalten** auf `application` (Migration 000050):
  `has_contact_person` (BOOL), plus `contact_person_name/email/phone` (TEXT).
- **field_config-Eintrag** `contact_person` (Default hidden, per-EEG
  konfigurierbar). Single-Switch für den ganzen Block; Mitgliedstyp-
  Filterung im Code (nur company/association/municipality).
- **Public-Formular**: Checkbox unter UID/Vereinsnummer. Wenn aktiv:
  Name + E-Mail + Telefon (alle drei Pflicht). Required-Validierung
  gegated auf Toggle-aktiv (verhindert Submit-Hänger-Bug-Pattern).
- **Admin-UI**: Detail-View zeigt Ansprechperson-Block wenn gesetzt;
  Edit-Form erlaubt Toggle umschalten und Werte editieren (sichtbar
  nur bei Org-Mitgliedstypen).
- **Beitritts-PDF**: neuer Block „Ansprechperson" zwischen
  Mitgliedsdaten und Bankverbindung, gerendert wenn Toggle aktiv.
- **EEG-Submission-Mail** (PROJ-20): neuer Block in
  `application_submitted_eeg.html` zwischen Adresse und Bankverbindung.
- **Server-Side-Cleanup**: `clearContactPersonIfDisabled` cleart die
  drei Felder auf NULL, wenn Toggle aus oder Mitgliedstyp nicht in der
  Org-Liste — schützt gegen forged Clients.
- **Excel-Export** (PROJ-17) wurde bewusst NICHT erweitert.

### PROJ-56 — Netzbetreiber-Info-Seite im Beitrittsbestätigungs-PDF *(2026-05-21)*

Zusätzliche PDF-Seite mit allen Daten, die die EEG-Verwaltung für die
Netzbetreiber-Korrespondenz braucht. Wird konditional gerendert: nur
wenn das Mitglied die Netzbetreiber-Vollmacht aktiv erteilt hat
(PROJ-44).

Eckdaten:

- **Zwei neue per-Mitglied-Felder** auf `application`:
  `network_operator_customer_number` und `meter_inventory_number`
  (Migration 000049). Beide TEXT NULL.
- **Conditional Rendering** im Public-Formular: erscheinen direkt
  unter der Vollmachts-Checkbox, sobald sie aktiviert wird. Jedes Feld
  einzeln per `field_config` ein-/ausblendbar; Required-Status pro EEG
  konfigurierbar.
- **Admin-UI**: zwei Felder im Detail-View (Anzeige) und im Edit-Form
  (editierbar) — beide nur sichtbar wenn Vollmacht aktiv.
- **PDF-Seite** (`approval_pdf.go`) mit:
  - Überschrift "Informationen für den Netzbetreiber"
  - Kundennummer + Inventarnummer
  - [X]-Box mit Volltext der Vollmacht + Timestamp
    ("Vollmacht erteilt am `<submitted_at>`")
  - Tabelle aller Zählpunkte (Nr / Adresse zwei-zeilig / Typ CNSM-GNRT / TF)
  - 33-stellige AT-Zählpunkt-Nummern werden in 5 Gruppen (2-6-5-10-10)
    gruppiert dargestellt für bessere Lesbarkeit.
- **Validierung**: Required-Check der zwei Felder läuft nur wenn die
  Vollmacht aktiv ist — sonst Submit-Hänger-Falle wie beim Geburtsdatum
  vermieden (vgl. Commit `72d380b`).
- **Server-Side-Cleanup**: `clearNetworkAuthIfHidden` setzt die zwei
  Felder auf NULL, wenn die Vollmacht nicht (mehr) erteilt ist oder
  die EEG die Felder versteckt hat.

### Bug-Fixes 2026-05-21

- **Beitrittsbestätigungs-PDF**: Netzbetreiber-Vollmacht wurde sowohl
  in „Erteilte Zustimmungen" als auch in „Weitere Angaben" gerendert.
  Der Duplikat-Eintrag in „Weitere Angaben" wurde entfernt; der voll-
  formulierte Block in „Erteilte Zustimmungen" bleibt.
- **Beitrittsbestätigungs-PDF**: Format der Zustimmungs-Zeile geändert
  von `- Statuten — Zugestimmt am …` auf `- Statuten zugestimmt am …`
  (Gedankenstrich entfernt, klein geschrieben).
- **Public-Formular**: Hinweis „SEPA-Mandat erhältst du per E-Mail …"
  wandert aus der Einwilligungs-Box in die Bankverbindung-Box —
  kontextnah am IBAN-Feld statt versehentlich wie eine weitere
  Einwilligung wirkend.
- **Public-Formular**: Submit-Hänger bei Mitgliedstyp `sole_proprietor`,
  `company`, `municipality`, `association` behoben — Geburtsdatum-
  Validierung lief unbedingt, obwohl das Feld nur für isPerson-Typen
  gerendert wird. Selbe Falle für consumption-only-Felder
  (`persons_in_household`, `heat_pump`, …) zusätzlich gefixt.

### PROJ-54 — Repo-Split: privates Hauptrepo + öffentlicher Mirror *(2026-05-20)*

Aktive Entwicklung läuft ab sofort im privaten Repo
`Marki4711/eegfaktura-member-onboarding-private`; der öffentliche Repo
`Marki4711/eegfaktura-member-onboarding` wird via GitHub-Action-Mirror
auf jeden Push automatisch aktualisiert.

Eckdaten:

- **Whitelist** (`.github/mirror-whitelist.txt`): definiert was im
  Public-Mirror erscheint. `private/` und alle `.github/`-Inhalte sind
  ausgeschlossen.
- **Frontmatter-Filter**: einzelne Markdown-Dateien mit YAML-Frontmatter
  `visibility: private` werden zusätzlich aus dem Mirror entfernt.
- **CI/CD-Verteilung**: Snyk, EOL-Check, Docker-Publish, Dependabot, CI
  Build & Test laufen nur im privaten Repo. Public hat keine Actions.
- **Git-Hooks** (`.githooks/pre-commit`, `pre-push`): defensive Schicht,
  blockt direkten Push aufs Public-Repo + warnt bei `private/`-Pfaden.
- **Smoke-Build** (Go + Node) auf dem gefilterten Output: schlägt fehl,
  bricht Mirror ab (kein Public-Push).
- **Mirror-Lag**: ~80–90 s pro Push.

Sensible Bereiche (Pricing, Verträge, DPIA, Pen-Test-Reports,
Anbieter-Setups, eigenes Rechnungsmodul) landen ab sofort unter
`private/` und werden nicht öffentlich gespiegelt.

### Optionales UID-Feld für Verein im Public-Form *(2026-05-20)*

Mitgliedstyp `association` zeigt im öffentlichen Registrierungsformular jetzt
zusätzlich zur (Pflicht-) Vereinsnummer ein **optionales UID-Nummer-Feld** —
analog zur bereits vorhandenen Umsetzung bei `municipality` (Gemeinde).

Backend, Admin-Edit-Form, Mail/PDF/Excel und der Core-Payload-Mapper kannten
das Feld bereits für `association` (kein Nullen in `clearMemberTypeFields`,
kein Required-Check); reines Frontend-Rendering-Gap geschlossen
(`src/components/registration-form.tsx`).

### Teilnahmefaktor pro EEG konfigurierbar *(2026-05-19)*

Das Feld `participation_factor` (Teilnahmefaktor in %) ist jetzt über die
PROJ-8-Field-Config pro EEG ein-/ausblendbar:

- Neu in `knownConfigurableFields` (Backend) + `CONFIGURABLE_FIELDS.meteringPoint`
  (Frontend) mit Default `optional` — heutiges Verhalten bleibt erhalten.
- Bei `hidden` oder `admin_only` rendert das Public-Formular kein Eingabefeld;
  der Wert wird serverseitig automatisch auf **100 %** defaulted
  (`defaultParticipationFactor` in `application_service.go`).
- Bei `optional` oder `required` ist das Feld sichtbar und mit 100 % vorbelegt —
  das Mitglied kann ändern oder den Default beibehalten.
- Validate-Tag von `required,min=1,max=100` auf `min=0,max=100` gelockert,
  damit das Frontend bei `hidden` einen 0er-Submit machen kann (Service
  mappt 0 → 100).
- **PDF, Mail und Excel-Export zeigen den Teilnahmefaktor in allen Modi
  unverändert** — der Toggle steuert nur die Public-Form-Sichtbarkeit, nicht
  die Render-Pfade. Der Core-Import (`partFact` = Mitglied-Wert) bleibt
  unverändert.

Docs: `docs/user-guide/06-admin-settings.md` Abschnitt „Spezielle
konfigurierbare Felder" um den neuen Toggle ergänzt.

### PROJ-53 — Aktivierungs-Modus pro EEG + Beitrittsbestätigung erst bei `activated` + manueller `approved → activated`-Skip *(2026-05-19)*

Drei zusammenhängende Änderungen am Activation-/Mail-Lifecycle:

**1. Beitrittsbestätigung wandert von `imported` nach `activated`**
- `SendImportedNotification` (volle Beitrittsbestätigung + PDF + optional Mandat) entfällt.
- Neue Funktion `SendMandateAtImportNotification` (schlank, nur Mandat-Anlage)
  läuft beim Wechsel auf `imported` — und auch nur dann, wenn überhaupt ein
  Mandat zu versenden ist (b2b oder `sepa_mandate_at_import=true`).
- Neue Funktion `SendActivationNotification` (volle Beitrittsbestätigung mit
  PDF an Member + EEG-Contact) läuft beim Wechsel auf `activated`.
- Templates: `application_imported_*.html` umgeschrieben auf "Anlage Mandat —
  Beitrittsbestätigung folgt"; `application_activated_member.html` enthält
  jetzt die volle Beitrittsbestätigung; neues `application_activated_eeg.html`.
- Alte kurze `SendActivatedNotification`-Welcome-Mail entfällt (war Funktion
  mit identischem Auslöser, aber dünnerem Inhalt — wird durch die volle
  Beitrittsbestätigungs-Mail abgelöst).
- **Idempotenz:** neue Spalte `application.activation_notification_sent_at`
  speichert den Sendetag. Wird beim erfolgreichen Versand gesetzt; mehrfache
  Aktivierungen schicken nicht doppelt.
- **Hartes Cut-off für Bestandsanträge:** Migration 047 setzt das Flag
  retrospektiv für alle Anträge in `imported/ready_for_activation/
  awaiting_bank_confirmation/activated` auf `updated_at`. So bekommen
  Mitglieder, die schon eine "alte" Beitrittsbestätigung beim Import erhalten
  haben, beim Übergang auf activated keine zweite.

**2. Aktivierungs-Kriterium pro EEG konfigurierbar**
- Neue Spalte `registration_entrypoint.activation_mode` (Default
  `participant_active`, alternativ `any_meter_registration_started`).
  Migration 048 inkl. DB-CHECK.
- `CoreParticipantSummary` um `Meters []CoreMeterSummary{MeteringPoint, Status, ProcessState}`
  erweitert — die nötigen Felder lieferte das deployed Core-Endpoint
  `GET /api/participant` schon, wurden bisher nur weggeworfen
  (verifiziert am 2026-05-19 gegen RC101294).
- `ImportService.CheckActivations` evaluiert pro EEG den `activation_mode`:
  - `participant_active`: heutige Logik — `participant.status == ACTIVE`
  - `any_meter_registration_started`: min. ein Zählpunkt mit
    `processState ∈ {PENDING, APPROVED, ACTIVE}` (Netzbetreiber hat
    EDA-Online-Registrierung mindestens bestätigt)
- Admin-Settings-Editor: Radio-Block "Aktivierungs-Kriterium" mit Erklärtexten
  zu beiden Varianten.
- API: `GET/PUT /api/admin/settings/eeg` um `activationMode` erweitert
  (Patch-Semantik, Enum-Validation).
- Default ist rückwärtskompatibel — kein Bestands-EEG kippt ungewollt um.

**3. Manueller `approved → activated`-Skip-Import (Ausnahmefall)**
- Neue Transition `approved → activated` (zusätzlich zu `approved → imported`
  und `approved → import_failed`). NICHT über generisches `/status` zugänglich
  — nur über dedizierten Endpoint.
- Use-case: Mitglied existiert im eegFaktura-Core bereits (Faktura erlaubt
  kein Löschen) und wurde dort manuell mit den Onboarding-Daten
  überschrieben. Der Onboarding-Antrag muss trotzdem zu `activated` kommen.
- Neuer Endpoint `POST /api/admin/applications/{id}/mark-activated` mit
  Pflicht-Body `{"memberNumber": "..."}`. Validiert: Status muss `approved`
  sein, Mitgliedsnummer muss frei sein (kein Konflikt in der EEG).
- Triggert dieselbe `SendActivationNotification` wie der reguläre Pfad
  (Flag-Check verhindert doppelten Versand).
- Admin-UI: Button "Manuell aktivieren …" auf der Detailansicht einer
  `approved`-Anwendung, öffnet Dialog mit Pflichtfeld Mitgliedsnummer und
  deutlichem Warnhinweis "Nur verwenden wenn Core-Member bereits manuell
  überschrieben".

**Tests:** neue Unit-Tests `TestShouldActivate` (11 Cases: A/B-Modus,
Edge-Cases inkl. Fallback bei unbekanntem Mode-Wert) und
`TestIsValidActivationMode` (Enum-Validator als Source-of-Truth-Gate
zwischen HTTP-Layer und DB-CHECK).

**Docs:** `docs/architecture-diagram.md` (State-Diagramm + Legende),
`CLAUDE.md` (Transitionsliste), `docs/domain-model.md` (neue Spalten),
`docs/api-spec.md` (neues Endpoint, Activation-Modus-Tabelle, EDA-Mapping,
`activationMode` in EEG-Settings-Beispielen).

### Docs — Audit aller `docs/` und `docs/user-guide/` *(2026-05-18)*

Vollständiger Durchgang aller Top-Level-Dokumente und der User-Guide nach
heute deployed Features. Befunde und Fixes:

**User-Guide:**
- `04-admin-applications.md` + `05-admin-status.md`: 5× „In Prüfung" /
  „Zur Prüfung" / „In Prüfung nehmen" → „In Bearbeitung" / „In Bearbeitung
  nehmen" / „Zurück in Bearbeitung" (Status-Filter, Button-Labels,
  Section-Titel). PDF und Feature-Specs bewusst nicht angefasst.
- `06-admin-settings.md`: Neuer Abschnitt **„Zählpunkt-Prefixes (PROJ-52)"**
  mit Beschreibung von Verbraucher-/Einspeisungs-Prefix, Format-Regeln,
  Live-Vorschau, Auto-Pad und Backend-Match-Validation. `bank_name` in der
  Liste „Spezielle konfigurierbare Felder" ergänzt.
- `02-member-registration.md`: Member-Type-Tabelle um `Kleinunternehmer`
  ergänzt + USt.-Hinweis-Spalte. Schritt 5 (Zählpunkte) um neues Layout
  (Richtung+Faktor zuerst, Zählpunkt full-width darunter), Mask-Lock und
  Auto-Pad-Verhalten erweitert. Schritt 7 ergänzt um die heutige
  PROJ-31-Success-Variante („Bitte E-Mail-Postfach prüfen").
- `05-admin-status.md`: Hinweis zur Mail-Footer-Änderung (mailto-Link statt
  Postadresse) und zum vorbefüllten SEPA-Mandat-Datum ergänzt.

**Top-Level-Docs:**
- `PRD.md`: 17 Features (PROJ-33 bis PROJ-49 ohne PROJ-43-Duplikat) +
  PROJ-52 von „In Review" / „Planned" → „Shipped to production".
  PROJ-26 + PROJ-50 in den „On Hold"-Block verschoben.
- `security.md`: `validateMeteringPointPrefixMatch` (PROJ-52) zu den
  security-sensitive Bereichen unter `internal/application/` ergänzt.
- `api-spec.md`, `domain-model.md`, `architecture.md`, `import-mapping.md`,
  `operations.md`, `open-questions.md`, `keycloak-setup.md`: keine
  Anpassungen nötig — wurden bei den jeweiligen Feature-Commits mitgepflegt.

**Mail-Templates + PDF-Generatoren:**
- Audit bestätigt: alle `{{.Field}}`-Referenzen matchen die Go-Structs,
  Footer-Texte zeigen `EEGContactEmail` als mailto-Link, Zählpunkte werden
  in der 2-6-5-20-Gruppierung gerendert, SEPA-MandateDate wird in beiden
  PDF-Varianten oberhalb der Unterschriftslinie vorbefüllt. Keine Fixes
  erforderlich.

**Screenshots in `docs/user-guide/images/`:**
- Folgende Screenshots zeigen veraltete UI-Texte und sollten bei nächster
  Gelegenheit neu aufgenommen werden (manuell, kein Headless-Setup im
  CI): `admin-filter-panel.png` („In Prüfung"), `admin-status-actions.png`
  („In Prüfung nehmen" / „Zurück in Prüfung"), `admin-application-detail-1.png`
  („zur Prüfung bereit"), `admin-settings-eeg.png` (fehlender Zählpunkt-
  Prefix-Block), `register-form-metering-points.png` (neues Layout +
  Prefix-Lock).

### Reviews — Code-Review + Security-Review *(2026-05-18)*

Nach dem Docs-Audit zusätzlich:

- **Code-Review**: Cross-Check aller Mail-Templates, PDF-Generatoren und
  HTTP-Handler gegen api-spec.md, domain-model.md und die heute deployed
  Features. Vier parallele Explore-Agents (Mail+PDF, API, User-Guide,
  Top-Level-Docs) — alle Mail-Felder konsistent, kein undokumentierter
  Endpoint, keine veralteten Surface-Definitionen. Einziger Hinweis:
  `docs/docs.go` (Swagger-UI-Generierung) ist seit PROJ-28 nicht regeneriert
  — `api-spec.md` ist Source of Truth und aktuell, Swagger-UI hinkt
  nach. Vor nächstem Release `swag init -g cmd/server/main.go` ausführen
  (nicht-blockierend, optional).
- **Security-Review**: PROJ-52 Prefix-Match-Validation greift als
  defense-in-depth zusätzlich zur Frontend-Mask, DB-CHECK-Constraint
  (`^AT[0-9A-Z]{0,31}$`) schließt den letzten Layer. Normalisierung
  (Whitespace + Dots + Hyphens, uppercase) wird vor Validierung
  ausgeführt — kein Bypass via Eingabe-Tricks. Keine Auth-Boundaries
  geändert, keine neuen öffentlichen Endpoints, kein Geheimnis im Code.
  `app.MandateDate` ist eine reine Tagesinformation (keine PII-Eskalation).
  Bestehende Snyk-Scans + govulncheck weiter grün.

### Geändert — Zählpunkt-Mask auf offizielle Gruppierung 2-6-5-20 *(2026-05-17)*

Recherche zur E-Control / MeteringCode-Spec ergab, dass die offizielle
vierteilige Struktur der Zählpunktbezeichnung in Österreich
`AT | Netzbetreibernummer (6) | PLZ (5) | Zählpunktnummer (20)` lautet.
Die bisherige UI-Mask `2-6-5-12-8` war willkürlich.

Mask im Mitgliederformular auf die offizielle Aufteilung umgestellt
(`AT 000000 00000 [20 Stellen]`). Visuelle Änderung, keine Auswirkung
auf Validierung oder gespeicherte Daten (33 Stellen unverändert).

Vorbereitung für PROJ-52 (konfigurierbarer Prefix pro Richtung + Auto-Pad
+ alphanumerischer letzter Block — Spec angelegt, Implementierung folgt).

### Geändert — Speichersteuerung-Frage + Batterie-Gruppierung (PROJ-49 follow-up) *(2026-05-17)*

Neue Mitglied-Frage „Speichersteuerung im Sinne der EEG vorstellbar?" auf
PV-Erzeuger-Zählpunkten. Gleichzeitig UI-Refactoring: die bisher einzeln
sichtbaren Speicher-Felder werden hinter einer Master-Checkbox gruppiert.

**Datenmodell (Migration 000044):**
- `metering_point.battery_control_acceptable` BOOLEAN NULL — Mitglied-Antwort
  Ja/Nein. Service-Layer cleart das Feld, wenn kein PV-Zählpunkt oder wenn
  das Mitglied keine Batterie-Parameter angegeben hat.

**Sichtbarkeitsregeln:**
- Nur bei `direction='PRODUCTION'` + `generation_type='pv'`
- Nur wenn `battery_size_kwh` ODER `inverter_manufacturer` befüllt ist
- PROJ-8-konfigurierbar via field_config (`battery_control_acceptable`,
  Default `hidden`)

**API:**
- `meteringPoints[].batteryControlAcceptable` in Public-, Admin-,
  Externe-API (Request + Response).
- Required-Validierung greift nur, wenn das Mitglied tatsächlich Batterie-
  Daten gesetzt hat — sonst entfällt die Frage komplett.

**Frontend (UX-Verbesserung):**
- Neuer `BatteryBlock` in `metering-point-fields.tsx` mit Master-Checkbox
  „Batteriespeicher vorhanden". Nach Aktivieren erscheinen drei gruppierte
  Felder darunter: Größe Batterie (kWh), Hersteller Wechselrichter,
  Speichersteuerung im Sinne der EEG vorstellbar?.
- Deaktivieren der Master-Checkbox cleart alle drei Felder.
- Beim Reload wird der Toggle-Zustand aus dem Vorhandensein eines der drei
  Werte abgeleitet (Pattern aus `DeviatingAddressBlock`).
- `GenerationBlock` schlanker: Batterie-Felder dort entfernt, jetzt nur
  noch generation_type + PV-Leistung + Einspeise-Forecast + Einspeiselimit.

**Mail-Templates:** `FormatGenerationLine` rendert die Antwort wenn gesetzt
als zusätzliches Segment, z. B. `…, Speichersteuerung im Sinne der EEG: Ja`.

### Geändert — Energie-Felder pro Zählpunkt (PROJ-49) *(2026-05-17)*

Refactoring: 4 Energie-Felder wandern von `application` auf `metering_point`,
1 neues Feld kommt dazu.

**Datenmodell (Migration 000043):**
- `metering_point` bekommt 6 neue Spalten:
  `consumption_previous_year`, `consumption_forecast`, `feed_in_forecast`,
  `pv_power_kwp`, `feed_in_limit_present`, `feed_in_limit_kw`.
- `application` verliert 4 Spalten:
  `consumption_previous_year`, `consumption_forecast`, `feed_in_forecast`, `pv_power_kwp`.
- Bestandswerte werden verworfen (Entscheidung Owner 2026-05-17: nur In-Review-Anträge betroffen).
- Alte `field_config`-Einträge mit den 4 Namen werden gelöscht — EEGs reaktivieren bewusst.

**Sichtbarkeit (Service-Layer enforced):**
- `consumption_*` nur bei `direction='CONSUMPTION'`.
- `feed_in_forecast` nur bei `direction='PRODUCTION'` (alle Erzeugungsformen).
- `pv_power_kwp` / `feed_in_limit_*` nur bei `direction='PRODUCTION'` + `generation_type='pv'`.
- `feed_in_limit_kw` nur wenn `feed_in_limit_present=TRUE`.

**Neues Feld:** Einspeiselimit (Bool „vorhanden" + optional kW-Wert).
Manche Netzanschlüsse sind leistungstechnisch beschränkt (z. B. „nur 70 % der PV einspeisbar"); EEG braucht diese Info für die Planung.

**API:**
- `POST /api/public/applications` Request: 4 Felder wandern von Top-Level in `meteringPoints[]`-Einträge; 2 neue `feedInLimitPresent` / `feedInLimitKw`.
- `GET /api/admin/applications/{id}` Response: gleiche Bewegung in der Antwort.
- `PUT /api/admin/applications/{id}` Body: 4 Top-Level-Felder werden ignoriert.
- Externe API (`POST /api/external/v1/applications`) analog.

**Mail-Templates + PDF:**
- `FormatGenerationLine` rendert pro Zählpunkt jetzt:
  - CONSUMPTION: `Verbrauch Vorjahr X kWh, Prognose Y kWh`
  - PRODUCTION + pv: `PV 9,9 kWp, Prognose 6000 kWh/J, Speicher 10,5 kWh (Fronius), Einspeiselimit 7,0 kW`
- Die 4 Application-Level-Felder erscheinen nicht mehr im „Zusätzliche Informationen"-Block.

**Frontend:**
- `metering-point-fields.tsx`: 6 neue Felder mit Sichtbarkeitsbedingungen + neuer `ConsumptionDetailsBlock`.
- `registration-form.tsx`: 4 Application-Level-Felder + zugehörige Defaults/Validation/Payload entfernt; per-MP-Payload um die neuen Felder erweitert.
- `admin-eeg-settings-editor.tsx`: Felder wandern automatisch in die „Zählpunkt-Felder"-Sektion (via `CONFIGURABLE_FIELDS.meteringPoint`).

### Geändert — Register-Dialog + Admin-Settings: Audit-Fixes *(2026-05-17)*

Vollständiger Inhalts-Audit analog zum Mail-Template-Audit. 51 Findings
in drei Wellen abgearbeitet.

**Welle A (kritisch):**
- `admin-legal-documents-editor.tsx` `handlePolicyToggle` Payload-Fix:
  vollständiger Settings-Snapshot wird mitgesendet — vorher fehlten
  `sepaMandateAtImport` (PROJ-48), `cooperativeSharesEnabled`,
  `cooperativeRequiredShares`, `cooperativeShareAmountCents` (PROJ-37),
  d.h. der Datenschutz-Toggle überschrieb diese Settings stillschweigend
  mit Defaults. **Echter Datenverlust-Pfad behoben.**
- `admin-eeg-settings-editor.tsx` B2B-Label-Korrektur: „für Unternehmen
  und **Gemeinden**" (vorher „Unternehmen und Vereine" — fachlich falsch
  nach PROJ-48; Vereine bekommen kein B2B-Auto-Mandat).
- `admin-eeg-settings-editor.tsx` SEPA-Haupt-Toggle umformuliert auf
  „SEPA-Mandat von der EEG bereitstellen" (vorher: „dem Willkommensmail
  anhängen", was nach PROJ-48 nicht mehr stimmt). Beim
  at-import-Sub-Toggle ausführlicher Hilfetext.
- `registration-form.tsx`: Neuer Hinweistext bei
  `sepaMandateEnabled=true` UND `sepaMandateAtImport=true` — Member
  weiß nun, dass das Mandat erst mit der Beitrittsbestätigung kommt.
  `RegistrationConfig` (Backend + Frontend-Type) um
  `sepaMandateAtImport` erweitert.
- Veraltete Texte „Die zentrale Datenschutzerklärung wird über
  Servereinstellungen konfiguriert" in `app/admin/settings/page.tsx`
  und `admin-legal-documents-editor.tsx` korrigiert — der per-EEG
  Toggle existiert seit PROJ-18.
- Superuser-URL-Hinweis (`/admin/settings?rc=…`) in `settings/page.tsx`
  ersetzt — Code las den URL-Param nie aus, die Anleitung funktionierte
  nicht.

**Welle B (Konsistenz):**
- **`NETWORK_OPERATOR_AUTH_TEXT`** als Konstante in `src/lib/api.ts`
  extrahiert. Public-Form rendert nun aus der Konstante (Single Source
  of Truth, Spec/UI-Drift verhindert).
- `SyncedField` jetzt mit echtem disabled `<Input>` statt visuell
  ähnlichem `<div>` — A11y-Fix für Screen Reader.
- Genossenschaftsanteile-Sichtbarkeits-Bug: Block wird jetzt gerendert,
  sobald `cooperativeSharesEnabled=true` (auch wenn `amountCents=null`).
  Vorher wurde der ganze Block stillschweigend ausgeblendet — Member
  scheiterte beim Submit am Backend-400.
- Du→Sie im Admin-Editor (zwei Stellen in
  `admin-eeg-settings-editor.tsx` mit „Klicke" → „Klicken Sie").
- „Bestätigungs-Mail" → „Eingangsbestätigung" (PROJ-31-Erläuterung).
- Mitglieds­typ-Label „Gemeinde / öffentl. Körperschaft" → ausgeschrieben.
- `orgLabel` für Kleinunternehmer ergänzt („Firmenbezeichnung" statt
  fallback „Firmenname").
- `Aktiv am (Beitrittsdatum)`-Hilfetext: `<p>` → Popover (Frontend-
  Regel-Compliance, Pattern aus `.claude/rules/frontend.md`).

**Welle C (Kosmetik):**
- Unicode-Pfeile `▲`/`▼`/`▴`/`▾` → lucide `ChevronUp`/`ChevronDown`
  in `admin-legal-documents-editor.tsx` + `admin-eeg-settings-editor.tsx`.
- „+ Dokument hinzufügen" → lucide `PlusCircle`-Icon-Pattern.
- Placeholder `"Richtung"` aus `metering-point-fields.tsx` entfernt
  (wurde nie sichtbar, Wert ist immer initial gesetzt).
- `z.B.` → `z. B.` Typografie-Fix in `admin-api-key-editor.tsx`.
- Doppelter Kommentar zur Metering-Points-Karte in `registration-form.tsx`
  entfernt.

Backend + Tests grün.

### Geändert — Mail-Templates: Audit-Fixes + Orphan-Cleanup *(2026-05-17)*

Vollständiger Inhalts-Audit aller 8 Mail-Templates + Behebung der
gefundenen Inkonsistenzen.

**Welle 1 (kritisch):**
- `application_imported_member.html` um PROJ-48-Pfad erweitert: neuer
  `HasMandateAttachment`-Flag in `importedTemplateData` triggert einen
  zusätzlichen Block „Ihr SEPA-Lastschriftmandat" mit Signatur-Anleitung
  (Ausdruck oder ID-Austria-App) — wird gerendert, wenn beim Import ein
  Basis-Mandat angehängt wurde (PROJ-48-`sepa_mandate_at_import=true`-
  Pfad für `einzugsart=core`). B2B-Block (PROJ-47) bleibt parallel.
- `application_imported_eeg.html` analog: zusätzlicher Hinweis-Block,
  wenn das Mitglied das Basis-Mandat mit ausgefüllter Mandatsreferenz
  bekommen hat — Admin weiß, dass auf unterschriebene Rücksendung
  gewartet werden muss, bevor Lastschriften eingezogen werden.
- `approvalSepaMandateType` und `resolveSepaMandateType` an PROJ-48
  angepasst: SEPA-Variante richtet sich jetzt allein nach
  `app.einzugsart` (Auto-Logik via Mitgliedstyp + useCompanySEPAMandate
  entfernt — entsprach nicht mehr dem neuen Default-Core-Workflow).
- `application_approved_eeg.html` **gelöscht** (Orphan seit PROJ-46
  Stage B). `SendApprovalEmail`-Method aus MailService-Interface,
  NoOpMailService und SMTPMailService entfernt. Zugehörige Tests
  (`TestSendApprovalEmail*`, `TestApprovalTemplate*`) entfernt.
  Auch der `approvalTpl`-Field und `approvedEEGTemplateData`-Typ weg.

**Welle 2 (Inhalte + Konsistenz):**
- `application_submitted_member.html`:
  - SEPA-Tabellenzeile vereinfacht (vorher mit verschachtelter
    SEPAMandateEnabled/Accepted-Logik): zeigt jetzt klare drei
    Varianten — „Mandat als PDF-Anhang", „Mandat wird mit
    Beitrittsbestätigung übermittelt" (PROJ-48-Pfad), oder
    „Online-Zustimmung erteilt"
  - Redundanter Schluss-Text entfernt (war doppelt mit
    Confirmation-Box am Anfang)
- `application_submitted_eeg.html`: zusätzliche Zeile
  „E-Mail bestätigt am: …" wenn PROJ-31 aktiv ist — macht den
  Zeitversatz zwischen Submit und EEG-Mail-Versand transparent.
  `EmailConfirmedAt`-Feld in `eegTemplateData` ergänzt.
- `application_needs_info_member.html`: Anleitung erweitert um den
  Hinweis, dass die EEG den ursprünglichen Antragslink erneut zusenden
  kann, wenn das Mitglied Angaben direkt im Form korrigieren möchte
  (vorher nur „E-Mail antworten").
- `application_activated_member.html`: realistischere Formulierung —
  „formal aktiv, tatsächliche Teilnahme startet sobald der
  Netzbetreiber freigeschaltet hat" (vorher überoptimistisch
  „ab sofort am Sharing teil"). Plus erste-Abrechnungs-Hinweis.
- **Konsistente Signaturen quer durch alle Member-Templates**:
  „Ihr Team von {EEG-Name}" mit Fallback „Ihre Energiegemeinschaft"
  (vorher mal „Ihr eegFaktura-Team"). Der eegFaktura-Brand bleibt
  nur im Footer als Erzeuger-Hinweis.
- Alle Member-Templates beginnen einheitlich mit
  „Sehr geehrte/r {Vorname} {Nachname}".

### Neu — PROJ-48: SEPA-Default-Core + konfigurierbares Mandat-Timing + B2B-Hinweis *(2026-05-17)*

Drei zusammenhängende Änderungen am SEPA-Workflow:

1. **Default-Einzugsart immer `core`.** Die Auto-Logik „Firmenlastschrift
   bei Mitgliedstyp company/association mit useCompanySEPAMandate=true"
   im Submit-Pfad **entfällt ersatzlos**. Submit-Mail enthält jetzt
   immer das Basis-Mandat (oder kein PDF, je nach Setting 3). Admin
   kann die Einzugsart per Antrags-Edit weiterhin auf `b2b` umstellen.
2. **B2B-Hinweis-Block in der Submit-Mail** bei Mitgliedstyp
   `company` und `municipality` — kurzer Satz: „Falls statt der
   Basislastschrift eine Firmenlastschrift (SEPA B2B) notwendig ist,
   meldet sich {EEG-Name} mit den notwendigen Unterlagen bei Ihnen."
   (Verein bewusst ausgenommen — User-Wunsch.)
3. **Neues EEG-Setting `sepa_mandate_at_import`** (Default `FALSE` =
   heutiges Verhalten). Bei `TRUE` wird das SEPA-Mandat-PDF NICHT
   beim Submit, sondern erst beim Import mit eingedruckter
   Mandatsreferenz = Mitgliedsnummer versendet — auch für `core`
   (bislang nur PROJ-47-Pfad für `b2b`).

Architektur-Hintergrund: PROJ-48-Setting löst den Konflikt „digital
signiertes Dokument darf nicht mehr modifiziert werden". Wenn die EEG
digitale Mandate verwendet und Mandatsreferenz im Dokument verlangt
wird, ist der at-import-Pfad der einzige saubere Weg. Volltext zur
Digital-Signatur-Diskussion: `docs/open-questions.md` OQ-6 (neu).

- Migration `000042_sepa_mandate_at_import`
- `RegistrationEntrypoint.SEPAMandateAtImport` + Repo + Settings-Endpoint
- Submit-Mail-Logik: kein PDF bei `sepa_mandate_at_import=true`,
  ansonsten immer Basis-Variante (Firmenlastschrift-Auto-Wahl entfernt)
- Import-Mail-Logik: zusätzlicher Basis-Mandat-Anhang bei
  `einzugsart=core` + `sepa_mandate_at_import=true` (PROJ-47-B2B-Pfad
  unverändert)
- Mail-Template (`application_submitted_member.html`): B2B-Hinweis-
  Block conditional auf neuem `ShowB2BHint`-Flag
- Frontend Admin-Settings-Editor: neuer Switch „SEPA-Mandat erst beim
  Import senden" inkl. Tooltip-Hinweis auf Digital-Signatur-Use-Case
- OQ-6 in `docs/open-questions.md` ergänzt: vollständige Behandlung der
  Architektur-Implikationen einer digitalen Mandat-Signatur

### Neu — PROJ-47: B2B-SEPA-Firmenlastschrift-Mandat mit Mandatsreferenz beim Import *(2026-05-17)*

Schließt die in PROJ-46 erkannte Lücke: ein B2B-Antragsteller bekam
zwar bei Submission ein Firmenlastschrift-PDF, aber ohne die später
vergebene Mitgliedsnummer als Mandatsreferenz — die B2B-Bank verlangt
diese aber ausdrücklich. Mit PROJ-47 wird beim Import ein **zweites
Firmenlastschrift-Mandat-PDF mit eingedruckter Mandatsreferenz =
Mitgliedsnummer** generiert und an die Member-Mail (+ EEG-Kopie)
angehängt, das der Member ausdrucken und an seine Hausbank
weiterreichen kann.

- `pdf.SEPAMandateData`: neues optionales Feld `MandateReference`.
  Beide PDF-Renderer (Generate / GenerateCompany) drucken den Wert
  inline statt des Platzhalters „wird von … ausgefüllt".
- `mail.Sender` erweitert um `Attachment`-Struct +
  `SendWithAttachments(...)` für Multi-Anhang-Versand. Bestehende
  Single-Attachment-API bleibt und delegiert intern.
- `SendImportedNotification` nimmt zusätzlich `b2bMandatePDF []byte`;
  bei non-empty wird das B2B-Mandat als zweiter PDF-Anhang verschickt
  (Dateiname `sepa-firmenlastschrift-mandat-<Mitgliedsnr>.pdf`).
- `AdminApplicationService` bekommt
  `sepaMandateGenerator pdf.SEPAMandateGenerator` als Dependency.
  Beim Post-Import-Notification wird bei `einzugsart=b2b` der B2B-
  Mandat-Generator aufgerufen, Debtor-Name aus CompanyName,
  Mandatsreferenz aus MemberNumber, Logo aus EEG-Cache.
- Mail-Template `application_imported_member.html` ergänzt um Hinweis
  auf den zweiten PDF-Anhang im b2b-Block.
- Best-Effort bei B2B-PDF-Fehlern (Log + ohne 2. Anhang weiter); die
  Hauptmail mit Beitrittsbestätigung geht in jedem Fall raus.

### Neu — PROJ-46 Stage D: Activation-Check via Core *(2026-05-17)*

Admin-getriggerter Batch-Check ersetzt das ursprünglich geplante Cron-
Polling (User-Entscheidung B). Button „Aktivierung im Core prüfen" in
der Antrags-Übersicht (`/admin/applications`) ruft einen neuen Endpoint
auf, der alle `ready_for_activation`-Anträge der eigenen Tenants gegen
den eegFaktura-Core abgleicht und ACTIVE-Mitglieder automatisch auf
`activated` setzt.

- Neuer Endpoint `POST /api/admin/applications/check-activation` (kein
  ID, batch). Tenant-Scope kommt aus den JWT-Claims (Superuser ohne
  Filter, sonst nur eigene RCs).
- `coreclient.CoreParticipantSummary` um `Status string` erweitert
  (Werte: `NEW`, `PENDING`, `ACTIVE`).
- `ImportService.CheckActivations`: gruppiert Kandidaten per Tenant,
  ruft Core `GET /participant` einmal pro Tenant, mappt per
  `target_participant_id`, transitioniert bei `Status == "ACTIVE"`
  via guarded UpdateStatusAdminTx + Status-Log-Eintrag mit Actor
  `system:activation-check`.
- `ApplicationRepository.ListReadyForActivation(allowedRCNumbers)`
  liefert die minimalen Felder für den Cross-Reference.
- Frontend-Button in `applications-page-content.tsx`: zeigt Toast mit
  Ergebnis (`X von Y auf Aktiviert gesetzt`), refresht danach die Liste.
  Bei 0 Treffern oder Fehlern entsprechende Info/Warning-Toasts.

### Neu — PROJ-46 Stage C: Admin-UI für Post-Import-Stati *(2026-05-17)*

- `ApplicationStatus`-Typ um drei neue Werte erweitert
  (`awaiting_bank_confirmation`, `ready_for_activation`, `activated`)
- `AdminStatusBadge`: neue Farben — Amber für „Warte auf Bank-Bestätigung",
  Cyan für „Bereit zur Aktivierung", tiefes Smaragd für „Aktiviert"
- `admin-filter-panel`: die drei neuen Stati erscheinen als Filter-Option
- `admin-status-actions`: drei neue Block-Layouts:
  - `awaiting_bank_confirmation`: prominente Amber-Hinweisbox „Warte auf
    Bank-Bestätigung" + Buttons „Bank-Bestätigung erhalten", „Zurück in
    Prüfung", „Import zurücksetzen"
  - `ready_for_activation`: Buttons „Als aktiv markieren" (grün),
    „Zurück in Prüfung", „Import zurücksetzen"
  - `activated`: rein informativer Text — keine weiteren Aktionen
    (strikter Endzustand)
- Reset-Import-Dialog-Warning erweitert: erwähnt jetzt explizit, dass
  `activated`-Anträge nicht resetbar sind und dass Mitgliedsnummer +
  Bank-Bestätigung mitgelöscht werden

### Neu — PROJ-46 Stage B: PDF-Timing + Member-Mails nach Import + Aktivierung *(2026-05-17)*

PDF-Generierung wandert von `→ approved` zum Import-Zeitpunkt (wenn die
Mitgliedsnummer steht — Voraussetzung für die B2B-SEPA-Mandatsreferenz):

- Drei neue Mail-Templates: `application_imported_member.html`,
  `application_imported_eeg.html`, `application_activated_member.html`
- Neue MailService-Methoden `SendImportedNotification` (Member + EEG)
  und `SendActivatedNotification` (Member only). NoOpMailService und
  Interface entsprechend erweitert.
- `SendImportedNotification` schickt PDF-Anhang an Member und Kopie an
  EEG-Contact. Beide Templates zeigen bei `einzugsart=b2b` einen
  Zusatz-Hinweis: Member bekommt die Anleitung zur Hausbank-Pre-
  Notification, EEG sieht den Hinweis „Auf Bank-Bestätigung warten —
  bitte auf ready_for_activation weiterschalten".
- Neue Service-Methode `AdminApplicationService.SendPostImportNotification(appID)`
  bündelt die heavy-Loads (App, MPs, Status-Log, Consents, Entrypoint,
  FieldConfig, Logo) + PDF-Build + Mail-Send. Aus dem HTTP-Import-
  Handler nach `importService.Import()`-Erfolg in Goroutine aufgerufen
  (best-effort, blockiert nicht die HTTP-Response).
- `→ approved`-Trigger im `ChangeStatus` entfernt — die alte
  Approval-Mail an EEG (ohne Mitgliedsnummer im B2B-Mandat) entfällt
  komplett, ersetzt durch den Import-Trigger.
- `→ activated`-Trigger ergänzt: schickt Welcome-Mail an Member über
  `SendActivatedNotification` in Goroutine.
- `SendApprovalEmail` bleibt auf dem MailService-Interface (für Test-
  Kompatibilität), wird aus Produktiv-Code aber nicht mehr aufgerufen
  (Deprecation-Kommentar gesetzt). `application_approved_eeg.html`
  bleibt vorerst im Repo (kein aktiver Send-Pfad mehr).
- Prometheus-Counter neu: `eeg_imported`, `member_imported`,
  `member_activated` (success/failed-Labels wie bei bestehenden Mails).

### Neu — PROJ-46 Stage A: Stati für Import-Nachbereitung *(2026-05-17)*

Erste Stage: DB + Backend-Übergänge + Reset-Erweiterung. Mails (Stage B),
Admin-UI (Stage C) und Activation-Check-Button (Stage D) folgen separat.

- Migration `000041_post_import_statuses`: drei neue Status-Werte
  (`awaiting_bank_confirmation`, `ready_for_activation`, `activated`),
  CHECK-Constraint erweitert, zwei neue Audit-Timestamps
  (`bank_confirmed_at`, `activated_at`)
- Import-Service: nach erfolgreichem `→ imported` läuft automatisch
  ein Branch — `einzugsart=b2b` ⇒ `awaiting_bank_confirmation`,
  sonst direkt `ready_for_activation`. Status `imported` existiert
  nur Millisekunden als Landing-Zone für die Import-Bookkeeping.
- `adminTransitions`-Map: neue Übergänge für die zwei mittleren
  Stati (manuelle Weiterschaltung + Rückwärts auf `under_review`).
  `activated` ist strikter Endzustand, keine Transitions hinaus.
- `UpdateStatusAdminTx`: COALESCE-Pattern um `bank_confirmed_at`
  und `activated_at` erweitert; Service stempelt die Timestamps
  beim jeweiligen Übergang.
- Reset-Import (PROJ-30) erweitert: Reset ist jetzt auch aus
  `awaiting_bank_confirmation` und `ready_for_activation` möglich
  (zurück auf `approved`). Aus `activated` **nicht** — strikter
  Endzustand, Deaktivierung muss im Core erfolgen. Reset cleart
  zusätzlich `bank_confirmed_at` + `activated_at` für sauberen Retry.
- CLAUDE.md Status-Sektion aktualisiert (3 neue Stati + 7 neue
  Transition-Einträge dokumentiert).

### Neu — PROJ-45: Erzeugungsform + Batterie + typabhängige Sichtbarkeit *(2026-05-17)*

Drei zusammenhängende Erweiterungen rund um Erzeugungs-Zählpunkte:

1. **Erzeugungsform pro PRODUCTION-Zählpunkt** — neues Pflichtfeld
   `generation_type` mit den Werten `pv`/`hydro`/`wind`/`biomass`,
   Default `pv`. Bestandsdaten werden migrationsweise auf `pv` gesetzt.
   DB-CHECK erzwingt: CONSUMPTION ⇒ NULL, PRODUCTION ⇒ einer der vier Werte.
2. **Batterie + Wechselrichter pro PV-Zählpunkt** — zwei neue PROJ-8-
   konfigurierbare Felder `battery_size_kwh` (NUMERIC) und
   `inverter_manufacturer` (Freitext). Default `hidden`; werden nur
   gerendert wenn EEG-Konfig aktiv UND `generation_type='pv'`.
3. **Typabhängige Sichtbarkeit der App-Level-Energie-Felder** —
   Verbrauchs-Felder (Wärmepumpe, E-Auto, Verbrauch …) erscheinen nur
   wenn der Antrag mindestens einen CONSUMPTION-Zählpunkt hat;
   Erzeugungsfelder (PV-Leistung, Einspeisung Prognose) nur bei
   PRODUCTION-Zählpunkten. Frontend rendert live; Backend cleart die
   Felder beim Speichern (`clearAppFieldsByMpTypes`) und gated den
   required-Check entsprechend.

Migration `000040_generation_type_and_battery` läuft als Pre-Upgrade-
Job automatisch beim nächsten Deploy.

- Service-Layer-Normalisierung (`normalizeMeteringPointGeneration`):
  CONSUMPTION ⇒ generation_type/battery/inverter NULL; PRODUCTION ohne
  expliziten Typ ⇒ `pv`; non-pv ⇒ battery/inverter NULL — Schutz gegen
  forged Clients und konsistente Persistenz
- Admin-Edit-Form (`admin-edit-form.tsx`): Erzeugungsform-Select pro
  PRODUCTION-Zählpunkt + Batterie/Hersteller-Inputs bei PV
- Admin-Detail-Tabelle: neue Spalte „Erzeugung" mit kompakter
  Darstellung „PV, Speicher 10,5 kWh (Fronius)"
- Approval-PDF: zusätzliche Zeile pro PRODUCTION-Zählpunkt
  („Erzeugung: PV, Speicher 10 kWh (Fronius)")
- Mail (Member + EEG): Erzeugungs-Zeile in der Zählpunkt-Tabelle
- Excel-Export: drei neue Spalten am Ende der Zeile
  (`Erzeugungsform`, `Größe Batterie (kWh)`, `Hersteller WR`) —
  eegFaktura-Importer ignoriert unbekannte Spalten, kein Import-Risiko
- `validateConfigurableRequiredFields` neue Signatur mit `mps`-Parameter
  für typabhängiges Gating; Unit-Tests passen `nil` (kein Gating).

### Neu — PROJ-44: Netzbetreiber-Vollmacht (per-EEG konfigurierbar) *(2026-05-17)*

Manche Netzbetreiber (z.B. Netz OÖ) verlangen eine separate Vollmacht
des Mitglieds, damit die EEG in dessen Namen mit dem Netzbetreiber
verhandeln darf. Die Vollmacht ist nicht Teil der EEG-Mitgliedschafts­
zustimmung und nicht bei jeder EEG nötig — daher als neues
konfigurierbares Feld (PROJ-8-Pattern, Default `hidden`).

- Migration `000039_network_operator_authorization`: zwei Spalten auf
  `application` — `network_operator_authorization BOOLEAN NOT NULL DEFAULT FALSE`
  + `network_operator_authorization_at TIMESTAMPTZ NULL`
- Neues konfigurierbares Feld `network_operator_authorization` —
  EEGs mit Anforderung setzen es auf `required`, Bestands-EEGs bleiben
  auf `hidden` (kein Sichtbarkeitswechsel ohne Admin-Aktion)
- Verbindlicher Wortlaut der Vollmacht im Frontend (Checkbox-Label),
  Wortlaut versioniert über Code-Commit (keine DB-Versionierung — YAGNI)
- Service-Layer: `_at` wird automatisch auf `NOW()` gesetzt, wenn das
  Flag von FALSE auf TRUE wechselt; `clearNetworkAuthIfHidden` schützt
  vor forged Clients, die das Flag für EEGs mit `hidden`-Config setzen
- Approval-PDF + Member-/EEG-Mail: rendern „Netzbetreiber-Vollmacht
  erteilt: Ja" über bestehenden `buildConfigurableFields`-Pfad;
  FALSE wird unterdrückt (Default für Bestandsanträge)
- Admin-Detail: zeigt Vollmacht + Erteilungs-Timestamp, wenn erteilt
- Excel-Export: bewusst **nicht** befüllt — eegFaktura-Importer­spalten­
  struktur kennt das Feld nicht, Audit-Trail liegt in DB + PDF + Mail

### Geändert — Node-Runtime auf Node 22 LTS gebumpt + automatischer EOL-Check *(2026-05-17)*

Node 20 ist seit 30. April 2026 End-of-Life — keine neuen Security-Patches
mehr. Aktualisiert auf Node 22 LTS (Support bis April 2027), minimaler
Versions-Sprung mit geringstem Regressions-Risiko.

- `Dockerfile.frontend`: 3× `node:20-alpine` → `node:22-alpine`
- `.github/workflows/ci.yml` + `snyk.yml`: `node-version: '20'` → `'22'`
- `package.json`: `@types/node ^20` → `^22` (npm install regeneriert das Lock)
- `dependabot.yml`: Filter für `@types/node` Major-Bumps bleibt aktiv —
  bei nächstem Runtime-Sprung manuell nachziehen
- Neuer Workflow `.github/workflows/eol-check.yml`: läuft monatlich
  (`cron: '0 6 1 * *'`), fragt endoflife.date für **Node**, **Go**,
  **PostgreSQL** und öffnet GitHub-Issues sobald eine Komponente
  innerhalb von 60 Tagen EOL erreicht oder bereits EOL ist. De-dupliziert
  via offene `eol-check`-Issues, sodass kein monatliches Spamming
- Nach jedem Upgrade muss der `cycle`-Eintrag im EOL-Workflow auf die
  neue Major-Version nachgezogen werden (siehe Inline-Kommentar)

### Neu — PROJ-40: EEG-Umzuordnung eines Antrags im Review *(2026-05-17)*

Wenn ein Mitglied über den falschen RC-Link der EEG A registriert hat,
aber eigentlich zur EEG B gehört, kann der Admin den Antrag direkt
umordnen — ohne Re-Submit durch das Mitglied.

- Neuer Endpoint `POST /api/admin/applications/{id}/reassign-eeg`
- **Tenant-Check beidseitig:** Admin muss für Quelle UND Ziel autorisiert
  sein (oder Superuser); sonst 403
- **Reassignable nur in aktiver Review-Phase:** `submitted`,
  `email_confirmed`, `under_review`, `needs_info`. Anything else → 409
- **Neue Referenznummer** wird über den per-EEG-Counter (PROJ-35) der
  Ziel-EEG vergeben, damit die Member-facing-ID zur neuen EEG passt
- **Audit-Trail:** status_log-Entry mit Status unverändert + Reason +
  `[system] previous rc_number=...` + `[system] previous reference_number=...`
- **Repo-Guard** (defense-in-depth): `UpdateRCNumberTx` validiert
  `WHERE id=$ AND rc_number=$expected AND status IN (...)` — bei 0 Rows
  ErrConflict
- **Frontend**: Button „EEG umzuordnen" im Statusaktionen-Block, sichtbar
  nur wenn der Admin ≥ 2 EEGs verwaltet. Dialog mit Dropdown der Ziel-EEGs +
  Begründung + Hinweis-Block auf die neue Referenznummer
- **Out-of-Scope (V1):** Bulk-Reassign, Member-Mail, Re-Validierung von
  Cooperative-Shares / Field-Config / Email-Confirmation-Setting

### Neu — PROJ-42: E-Fahrzeug-Detailerfassung *(2026-05-17)*

Das bestehende `electric_vehicle`-Ja/Nein wird ergänzt um zwei optionale
Detail-Felder, die für die EEG-Lastprofil-Optimierung relevant sind:

- `electric_vehicle_count` (INT) — Anzahl der E-Fahrzeuge im Haushalt
- `electric_vehicle_annual_km` (INT) — geschätzte Gesamt-Jahreskilometer

Beide Felder folgen dem PROJ-8-Configurable-Fields-Pattern: pro EEG
einstellbar (default `hidden`). Im Public-Form werden sie **nur**
angezeigt, wenn (a) die EEG sie aktiviert hat UND (b) der Bewerber
„Ja" beim E-Auto angekreuzt hat. Service-Layer cleart beide Werte
serverseitig auf NULL falls `electric_vehicle != true` (kein DB-CHECK,
sondern Service-Gate `clearEVDetailsIfDisabled`).

Mail (Member + EEG), Approval-PDF, Excel-Export, Admin-Detail werden
über die bestehende Configurable-Fields-Pipeline automatisch versorgt
— sobald die Felder konfiguriert sind, erscheinen sie im
„Zusätzliche Informationen"-Block.

Migration: `db/migrations/000038_ev_details.up.sql`.

### Geändert — PROJ-41 + PROJ-43: Mail-Versand jetzt hard-fail *(2026-05-17)*

Der initiale Best-Effort-Goroutine-Versand wurde umgestellt auf:

- **Synchron + pre-commit**: rejected/needs_info-Mail wird gerendert und
  versendet, BEVOR `tx.Commit()` läuft. Bei Fehler greift `defer tx.Rollback()`
- **Hard-fail**: Mail-Fehler → Statuswechsel wird NICHT persistiert + API
  antwortet 500 mit Fehlermeldung → Admin sieht das Problem sofort im Dialog
  („Mail konnte nicht versendet werden"), kein stilles Scheitern im Log
- Approval-Mail bleibt vorerst best-effort (PDF-generation macht Sync teurer)
- Submission-Mails bleiben unverändert (public-facing, würde Antrags-Submit
  blocken)

### Neu — PROJ-41 + PROJ-43: Status-Change-Mails an Mitglied *(2026-05-17)*

Bisher erfuhr der Beitrittswerber nichts, wenn der EEG-Admin den Antrag
ablehnte oder Rückfragen stellte — der Antrag stand einfach still. Jetzt
löst jeder Wechsel auf `rejected` (PROJ-41) bzw. `needs_info` (PROJ-43)
automatisch eine E-Mail an `application.email` aus.

- Zwei neue Mail-Templates (`application_rejected_member.html`,
  `application_needs_info_member.html`) übernehmen die vom Admin
  eingegebene Begründung/Rückfrage **1:1** in den Mail-Body
- Reply-To = EEG-Kontakt-E-Mail, damit Antworten direkt an die EEG gehen
- Admin-Dialog zeigt einen blauen Hinweis-Block: „Der hier eingegebene
  Text wird per E-Mail an den Beitrittswerber übermittelt"
- Best-Effort + async: scheitert der Versand, wird der Statuswechsel nicht
  zurückgerollt — Fehler landet im Log + Prometheus-Metric
  `mail_sent_total{kind="member_rejection|member_needs_info"}`
- Out of scope: BulkChangeStatus löst (vorerst) keine Mails aus

### Neu — PROJ-39: Titel-Nach + Bankname im Public-Form + abweichende Adresse je Zählpunkt *(2026-05-17)*

Drei unabhängige Erweiterungen am öffentlichen Antragsformular.

- **„Titel nach"** als zusätzliches optionales Personenfeld (z.B. BSc, MSc, MBA). Bestehende `titel`-Spalte bleibt erhalten und repräsentiert implizit „Titel vor". Migration 000037 fügt `application.titel_nach` hinzu. Sichtbar in Mail, PDF und Excel-Export
- **„Bankname"** ist jetzt direkt vom Mitglied eingebbar (war bisher admin-only). Spalte `application.bank_name` existierte schon, nur neue Frontend- und API-Pfade
- **Abweichende Adresse je Zählpunkt** (Bricht V1-Architekturentscheidung!): Migration 000037 fügt 4 Adress-Spalten auf `metering_point` hinzu. UI zeigt eine Checkbox „Abweichende Adresse" pro Zählpunkt; bei Aktivierung werden Straße, Hausnummer, PLZ, Ort eingeblendet. Checkbox-State wird **nicht** persistiert — der Zustand ergibt sich beim Reload daraus, ob die vier Adressfelder gefüllt sind. Server enforciert die All-or-Nothing-Regel (entweder alle vier leer oder alle vier gesetzt)
- Mail (Member + EEG), Approval-PDF, Excel-Export, Admin-Detail-View berücksichtigen alle drei neuen Felder
- CLAUDE.md + docs/architecture.md aktualisiert: alte „all metering points use the same address as the member"-Klausel entfernt

### Behoben — Reset-Import: Mitgliedsnummer wird gelöscht *(2026-05-17)*

Beim Zurücksetzen eines Imports (`imported → approved`) blieb bisher die
Mitgliedsnummer am Antrag stehen, obwohl die zugehörige Participant-Zeile
in eegFaktura nicht mehr existiert. Resultat: stale Anzeige im Admin-Detail
+ Konflikt-Vorschlag beim nächsten Import-Versuch (selbe Mitgliedsnummer
würde wieder vorgeschlagen).

- **Backend** (`ResetImportTx`): zusätzlich `member_number = NULL`
- **Audit-Trail** (`AdminApplicationService.ResetImport`): die vorherige
  Mitgliedsnummer wird wie schon zuvor die `target_participant_id` an die
  Begründung angehängt (`[system] previous member_number=<x>`), damit sie
  nach dem Reset im Statusverlauf nachvollziehbar bleibt
- **Doku**: `docs/api-spec.md` 6.5.3 ergänzt um die zusätzliche Spalte +
  erweiterten Log-Reason

### Behoben — PROJ-31 Constraint-Lücke + Helm-Fix *(2026-05-16)*

- **DB**: Migration 000036 ergänzt `email_confirmed` im `application_status_check`-CHECK-Constraint. Davor lief jeder `confirm-email`-POST in einen Postgres-23514-Fehler → HTTP 500 „An internal error occurred". Ursache: PROJ-31 hatte die Status-Konstante + Transition-Map gepflegt, die DB-Constraint aber nie angepasst (Tests liefen gegen Go-Fake-Store, nicht gegen echtes Postgres)
- **Helm**: Backend-Deployment bekommt `PUBLIC_BASE_URL` aus `frontend.nextauthUrl` (single source of truth für die öffentliche App-URL). Vorher war die Env-Var im Chart gar nicht definiert → der PROJ-31-Confirm-Link wurde nie generiert (silent fallback auf Legacy-Flow ohne Bestätigungs-Block in der Mail)
- **Doku**: `docs/architecture.md` dokumentiert das Status-Set als 3-place-Invariant (Code-Konstanten + adminTransitions-Map + DB-CHECK-Constraint)

### Neu — PROJ-38: Status-Modell-Hygiene & Audit-Fixes *(2026-05-16)*

Code-Audit nach der PROJ-31-Constraint-Regression. Drei Findings umgesetzt, zwei als False-Positive verworfen.

- **`UpdateStatusAdminTx`** mit guarded `WHERE status = $expected_from` — bei 0 betroffenen Rows kommt `ErrConflict` (HTTP 409). Damit ist der admin-seitige Status-Schreibpfad auf dem gleichen Schutz-Niveau wie alle anderen `Mark*Tx`-Methoden. Vergisst ein Caller die Transition-Map oder mutiert ein paralleler Prozess parallel den Status, schlägt die UPDATE jetzt sauber fehl statt still durchzulaufen
- **`isKnownStatus`** deckt jetzt alle 9 Status-Werte ab (`email_confirmed` fehlte). Defensiv — der `adminTransitions`-Layer hatte die Konsequenz bereits korrekt abgefangen
- **`ResetImport`** dokumentiert, warum der PROJ-31-Confirmation-Gate hier intentional fehlt (Antrag bereits einmal vetted via `approved → imported`)
- Out of scope für separate Specs: Submit-Mail-Retry, Auto-Reject-Doppel-Metrik bei parallelen Pods

### Neu — PROJ-37: Genossenschaftsanteile *(2026-05-15)*

EEG-Admins können pro EEG aktivieren, ob Mitglieder bei der Registrierung Genossenschaftsanteile zeichnen müssen. Die Pflichtanzahl und der Wert je Anteil sind per EEG konfigurierbar; das Formular zeigt eine Live-Berechnung des Gesamtbetrags, die Beitrittsbestätigung weist die Anteile als eigene Sektion aus.

- **DB**: Migration 000035 fügt `registration_entrypoint.cooperative_shares_enabled` + `cooperative_required_shares` + `cooperative_share_amount_cents` und `application.cooperative_shares_count` hinzu (Integer-Cents für Geld, keine Float-Drift)
- **Admin-Settings**: neuer Abschnitt „Genossenschaftsanteile" mit Toggle + zwei conditional sichtbaren Inputs (Pflichtanteile + €-Wert). Validierung: enabled=true ⇒ beide Werte Pflicht, positiv
- **Public-Form**: konditioneller Block „Genossenschaftsanteile" zwischen Zählpunkten und Bankverbindung mit Hinweistext „Pflichtanteil je Standort: N", Eingabe (min=N, prefilled=N), Live-Berechnung Wert × Anzahl = Gesamtbetrag
- **Submit-Validierung**: `count >= required_shares` (Pflicht wenn EEG enabled). Konfig-Änderungen wirken **prospektiv** — bestehende Anträge bleiben unverändert
- **Admin-Detail**: eigene Mini-Box „Genossenschaftsanteile: N × X € = N·X €" mit Orange-Hinweis falls Bestand unter aktuellem Pflichtmaß
- **Beitrittsbestätigungs-PDF**: neue Sektion „GENOSSENSCHAFTSANTEILE" mit Anzahl × Wert = Gesamtbetrag
- **Nicht in Excel-Export, nicht in Core-Payload** — rein im Onboarding (eegFaktura hat keine Spalte dafür)
- **Bekannte V1-Lücke**: Admin-Edit-Form kennt das Feld noch nicht. Korrektur über needs_info-Flow möglich; Direkt-Edit folgt in V1.1 falls häufig benötigt

### Neu — PROJ-36: Optionale Rechtsdokumente als Info-Dokumente *(2026-05-15)*

Beta-Feedback: optionale Checkboxen waren verwirrend (Mitglieder wussten nicht, ob ihr fehlendes Häkchen rechtlich relevant ist). Der Toggle pro Rechtsdokument ist jetzt binär — **Pflicht-Zustimmung** oder **Nur zur Information**.

- **DB**: Migration 000034 fügt `document_consent.consent_type` (`explicit` | `informational`) hinzu, Default `explicit` für Bestandsdaten
- **Public-Form**: Pflicht-Dokumente bleiben Checkboxen; Info-Dokumente landen in einem eigenen „Zur Information"-Block mit Top-Border-Separator unterhalb aller Pflicht-Häkchen
- **Backend** (`ApplicationService.SubmitApplication`): schreibt automatisch `informational`-Consent-Einträge für jedes nicht-required `legal_document` der EEG — Audit-Trail bleibt vollständig auch ohne Häkchen
- **Admin-Detail + Beitrittsbestätigungs-PDF**: zwei separate Blöcke „Zugestimmte Dokumente · Zugestimmt am …" / „Zur Kenntnis genommene Dokumente · Kenntnis genommen am …"
- **Admin-Settings**: Toggle-Label kontextsensitiv („Mitglied muss zustimmen" / „Nur zur Information"), erklärender Hilfetext darunter, Listen-Badge sagt „Pflicht-Zustimmung" bzw. „Nur zur Information"

### Neu — PROJ-35: Per-EEG-Referenznummern *(2026-05-14)*

Antrags-Referenznummer im Format **`<RC>-<Jahr>-<NNNN>`** (z.B. `RC105720-2026-0001`) statt der bisherigen globalen `MO-YYYY-NNNNNN`-Sequenz. Counter resettet pro EEG und pro Jahr.

- Migration 000033 mit Counter-Tabelle `reference_number_counter (rc_number, year, last_value)`; atomare Increment via `INSERT … ON CONFLICT DO UPDATE … RETURNING`
- Bestehende Anträge behalten ihre alten Refs (Links in bereits verschickten Mails bleiben gültig)
- 4-stelliger Counter reicht für 9 999 Anträge/EEG/Jahr — Overflow gibt sprechenden Fehler statt Format zu erweitern

### Neu — PROJ-34: Robuste Import-Recovery *(2026-05-14)*

Behebt den „stuck-in-flight"-Fehlerklasse, die heute im Test-Cluster sichtbar wurde (Antrag bleibt nach DB-UNIQUE-Verletzung dauerhaft in `approved + in-flight`-Zustand).

- **Orphan-Fallback**: Wenn das Bookkeeping nach erfolgreichem Core-Insert fehlschlägt (UNIQUE-Index aus Migration 28 etc.), wechselt der Antrag in einer zweiten Transaktion auf `import_failed` mit `target_participant_id` und sprechender Fehlermeldung. Der bestehende Reset-Import-Flow (PROJ-30) wird damit zur Recovery-Route.
- **Lokaler Pre-Check** vor dem Core-Aufruf: `MemberNumberUsedLocally` blockiert duplikate `member_number` im selben EEG mit 409, bevor irgendetwas an den Core geht — kein Orphan-Teilnehmer mehr aus diesem Fehlerpfad
- **Stuck-Detection**: `AdminApplicationDetailResponse.importStuck` (server-side berechnet: `approved + import_started_at > 2 min + finished_at NULL`)
- **Zwei neue Admin-Endpoints**:
  - `POST /api/admin/applications/{id}/mark-imported-manually` — Admin gibt Core-UUID + Mitgliedsnummer ein, sauberer Übergang nach `imported`
  - `POST /api/admin/applications/{id}/clear-import-lock` — Lock raus, Status bleibt `approved` (Duplikatsrisiko, sprechender Warntext)
- **Admin-UI**: oranger Banner über der Statusaktions-Card mit zwei Recovery-Buttons inkl. Bestätigungsdialogen

### Neu — PROJ-33: EEG-Logo aus Core *(2026-05-14, Phase 2 von PROJ-32)*

EEG-Logo aus eegfaktura-billing-Service ziehen und in die Beitrittsbestätigung + SEPA-Mandat einbetten.

- **Endpoints**: `GET /cash/api/billingConfigs/tenant/{rc}` → `headerImageFileDataId` als Indikator, dann `GET /cash/api/billingConfigs/{billingConfigId}/logoImage` → Bytes
- **DB**: Migration 000032 mit `eeg_logo_bytes BYTEA`, `eeg_logo_mime TEXT`, `eeg_logo_synced_at TIMESTAMPTZ`
- **Caps**: 256 KB Hard-Limit via `io.LimitReader`, MIME-Whitelist `image/png|jpeg|gif` (gofpdf-kompatibel)
- **Best-effort**: Logo-Fetch-Fehler bricht den Stammdaten-Sync nicht ab; `logoSyncWarning` landet in der Response (Frontend rendert orangen Hinweis unter der Logo-Vorschau)
- **PDFs**: `embedLogoTopRight` rendert 30 mm hoch top-right, max 50 mm breit; korrupt-Bild oder fpdf-Fehler werden geloggt und übersprungen, PDF rendert weiter ohne Logo
- **Admin-UI**: Logo-Vorschau als 9tes Synced-Field in der Stammdaten-Card; Object-URL über `fetchEEGLogoBlob` (Bearer-Header), Cache-Bust via `eegLogoSyncedAt`-Timestamp
- **Neuer Endpoint**: `GET /api/admin/settings/eeg/logo?rc_number=…` liefert die Bytes mit korrektem `Content-Type` + 5-Min-Private-Cache

### Neu — PROJ-32: EEG-Stammdaten-Sync aus Core *(2026-05-14)*

Acht EEG-Stammdaten-Felder (Gemeinschafts-ID, Name, vier Adressfelder, Creditor-ID, Kontakt-E-Mail) werden direkt aus eegFaktura gespiegelt und sind im Onboarding **schreibgeschützt**.

- **GraphQL-Endpoint**: `POST {base}/api/query` mit `query { eeg }` (scalar `Eeg` — kein Selection-Set, returnt vollständiges JSON)
- **DB**: Migration 000031 mit `last_synced_from_core_at`; bestehende Stammdaten-Spalten werden vom Sync überschrieben
- **Architektur**: Single source of truth = `registration_entrypoint`; Auth = User-Context-Bearer-Forwarding (kein Service-Account); Microcache 30s auf `CompareEEGSettingsWithCore`
- **URL-Modell**: `CORE_BASE_URL` ist jetzt nur der Hostname (z.B. `https://eegfaktura.at`); Pfad-Prefixe (`/api/...`, `/cash/api/...`) sind im coreclient hardcoded — der frühere `CORE_GRAPHQL_URL`-env-var ist weg
- **UI**: Drift-Banner (grün/orange/grau) mit per-Feld-Diff; „Aus eegFaktura aktualisieren"-Button verwendet das Admin-JWT
- **Performance-Fix nebenbei**: `ListParticipants`-Body-Cap von 1 MiB auf 4 MiB hochgezogen (verhindert silent Truncation bei großen EEGs)

### Neu — PROJ-31: E-Mail-Adresse-Bestätigung (Anti-Abuse) *(2026-05-14)*

Pro EEG aktivierbar: Mitglieder müssen den Link in der Bestätigungs-Mail klicken, bevor der Antrag in den Admin-Review-Zustand wechselt.

- **Status-Modell**: neuer `email_confirmed`-Zustand zwischen `submitted` und `under_review`
- **DB**: Migration 000030 mit `email_confirmation_token_hash` (SHA-256), `email_confirmation_token_expires_at`, `email_confirmed_at`, `email_confirmation_used_at`, `registration_entrypoint.require_email_confirmation`
- **Security**: Token im URL-Fragment (`#token`) statt im Pfad → bleibt aus Server-Logs raus; Referrer-Policy `no-referrer`; idempotente Re-Clicks („Bereits bestätigt"-Seite statt 400)
- **Resend-Endpoint** für die Admin-Detail-Page; **30-Tage-Auto-Reject** via Background-Job
- **Admin-Guards**: `/status`-Endpoint refuses `submitted → under_review|needs_info|approved` mit 409 solange die Bestätigung aussteht — `submitted → rejected` bleibt als Anti-Spam-Override verfügbar

### Geändert — sonstige UX/Stabilität *(2026-05-15)*

- **B2B-Toggle-Label**: „Firmenlastschrift (B2B) für Unternehmen und **Vereine** verwenden" (zuvor „Verbände" — die Antrags-Auswahl kennt nur `Verein`)
- **Admin-Conflict-Messages**: Server-spezifische 409-Meldungen werden statt eines generischen „Aktion nicht mehr gültig"-Texts angezeigt (z.B. „E-Mail-Adresse des Bewerbers ist noch nicht bestätigt …")
- **Core-HTTP-400-Hint**: Opake `core returned HTTP 400: {}` wird auf eine handlungsorientierte Meldung übersetzt („Wahrscheinlichste Ursache: einer der Zählpunkte ist im Core bereits einem aktiven Teilnehmer zugeordnet")
- **Health-Probe-Spam**: K8s-Liveness/Readiness-Pings (`/livez`, `/readyz`) werden nicht mehr im Request-Log aufgezeichnet (Metric-Histogramm bekommt sie weiterhin)
- **CI**: `update-helm`-Job führt Retry-with-Rebase aus, behebt Race wenn manuelle Pushes mit dem Auto-Tag-Bump kollidieren

### Neu — PROJ-27: Tarif-Auswahl beim Import

Beim Klick auf „Importieren" öffnet sich ein Dialog, in dem Admin Tarif für Mitglied und je Zählpunkt wählt. Tarife werden zum Klick-Zeitpunkt live aus dem Core gelesen (`GET /eeg/tariff`), keine Persistierung im Onboarding.

- **Backend**: `coreclient.ListTariffs` + neuer Admin-Endpoint `GET /api/admin/tariffs?rcNumber=…`
- **Import-Flow**: Mitgliedstarif via `PUT /participant/v2/{id}` nach `POST /participant` (Core `EegParticipantBase.TariffId` ist `goqu:skipinsert`), Meter-Tarife direkt im `POST`-Body
- **Frontend**: `import-tariff-dialog.tsx` ersetzt den `confirm()`-Dialog
- Failure-Mode: schlägt das nachgelagerte Mitglieds-Tarif-Update fehl, wird Warnung in der Response zurückgegeben (Import gilt aber als erfolgreich)

### Neu — PROJ-28: Trennung Privat / Kleinunternehmer

Eigener `member_type` `sole_proprietor` (Kleinunternehmer). Privatperson zeigt Vor-/Nachname, Kleinunternehmer nur Firmenname (wird intern als `firstname` im Core eingestellt, weil dort NOT NULL).

- **Backend**: neue Konstante `MemberTypeSoleProprietor`, oneof-Validatoren erweitert (4 Stellen), Member-Type-Validation passt UID/Register-Felder an
- **Frontend**: zusätzlicher SelectItem; Admin-Edit-Form blendet UID/Register je nach Typ ein/aus
- **Salutation-Fix als Side-Effect**: leere `Sehr geehrte/r ,` für alle Org-Typen behoben (`application_submitted_member.html` mit `{{if .Firstname}}…{{else}}Sehr geehrte Damen und Herren{{end}}`)

### Neu — PROJ-29: IBAN-Eingabe mit visueller Gruppierung

IBAN-Feld nutzt `MaskedInput` (react-imask) mit Block-Gruppierung pro 4 Stellen.

- **Initiale Umsetzung**: feste Mask `aa00 0000 …` (AT/DE/ES/BE/LU/SI — alle Länder mit reinen Ziffern im BBAN)
- **Erweiterung (2026-05-13)**: **dynamische landesabhängige Mask** — `src/lib/iban-mask.ts` generiert pro Land aus `ibantools.countrySpecs.bban_regexp` die exakte Mask-Struktur (Ziffern vs. Buchstaben vs. alphanumerisch). ~80 IBAN-Länder werden ohne Mapping-Wartung unterstützt
- **Validierungs-Bugfix**: `zod`-Transform strippt jetzt `[^A-Z0-9]` (vorher nur `\s`), entfernt iMask-Platzhalter `_` aus dem submitted value bevor `isValidIBAN` prüft

### Neu — PROJ-30: Reset eines importierten Antrags auf `approved`

Wenn ein importiertes Mitglied im Core wieder gelöscht wird, kann der Admin den Antrag jetzt auf `approved` zurücksetzen, um ihn erneut zu importieren.

- **Endpoint**: `POST /api/admin/applications/{id}/reset-import` (Body: `{ "reason": "…" }`)
- **Repo**: `ResetImportTx` setzt `import_*`-Felder + `target_participant_id` zurück auf NULL; alte Participant-ID wird im `status_log.reason` archiviert
- **Status-Modell**: Die Transition `imported → approved` ist bewusst **nicht** im generischen `adminTransitions`-Map — sie geht ausschließlich über den dedizierten Endpoint (Security-relevant, siehe CLAUDE.md)
- **Frontend**: „Import zurücksetzen"-Button mit Bestätigungsdialog inkl. Hinweis auf vorherige Core-Löschung

### Neu — Approval-PDF: Einheitliche Zustimmungs-Timestamps

PDF-Bereich „ERTEILTE ZUSTIMMUNGEN" zeigt jetzt überall Datum **und** Uhrzeit:

- Datenschutz: `privacy_accepted_at`
- Richtigkeit der Angaben: `submitted_at` (Validierung erfolgt im Submit-Moment, keine eigene Spalte nötig — keine Migration)
- SEPA-Mandat: `sepa_mandate_accepted_at` (Format `am DD.MM.YYYY HH:MM`)
- Dokument-Zustimmungen: erweitert um Uhrzeit

### Geändert

- **PDF + Mail: SEPA-Mandat-Beschriftung korrigiert (zuvor invertiert).** Bei `SEPAMandateEnabled=true` (Admin-Setting „SEPA-Lastschriftmandat dem Willkommensmail anhängen") zeigt PDF und Member-Mail jetzt **„Per E-Mail übermittelt"**, bei `false` **„Erteilt"**. Vorher andersrum gelabelt.
- **Zählpunkt-Feld**: schmalere Darstellung am Desktop (Default-Sans + `tabular-nums` + `tracking-tighter` + `px-2`), damit die 37-stellige Mask in einer Zeile passt. Mobile-Optik bleibt identisch.
- **Zählpunkt-Label**: Info-Popover beim Label erklärt was die Zählpunktnummer ist und wo sie zu finden ist (Stromrechnung / Kundenportal).

### Behoben — Zeitzone: alle sichtbaren Timestamps jetzt Europe/Vienna

PostgreSQL speichert UTC; vorher rendete PDF / Mail / Admin-Web jeweils unterschiedlich (UTC vs. Browser-Zone). Vereinheitlicht auf Europe/Vienna mit CET/CEST-Umstellung:

- **Backend**: neuer Helper `internal/shared/timezone.go` (`DisplayLocation`, `FmtDateTime`, `FmtDate`). PDF und Mail-Service nutzen ihn; Mail-Templates über `template.Funcs` (`{{fmtDateTime …}}`)
- **Frontend**: neuer Helper `src/lib/datetime.ts` (`formatDateTime`, `formatDate`, `formatPlainDate` — alle mit `timeZone: "Europe/Vienna"`). Ersetzt 4 inline-Implementierungen in `admin-application-detail`, `admin-application-table`, `admin-api-key-editor`, `admin-status-log`
- **DATE-Felder** (`birth_date`, `membership_start_date`) bleiben TZ-unaware, da ohne Zeitkomponente

### Strenge Zählpunktnummer-Validierung

Frontend (Zod) und Backend (Regex + struct tag `len=33,startswith=AT`) lehnen Eingaben außerhalb von `^AT[0-9]{31}$` ab. Eingabe wird automatisch ge-uppercased und whitespace-bereinigt.

### Sonstiges

- Favicon hinzugefügt (`src/app/icon.svg`, Next.js App-Router Auto-Detect)
- Mobile-Optik: Zählpunkt-Input nutzt `text-xs font-mono tracking-tight` auf engen Viewports

### Neu — Click-to-Sort in der Admin-Liste

Spalten-Header der Antrags-Tabelle sind klickbar und sortieren server-seitig:

- Frontend: Pfeil-Icon (↕ inaktiv · ↑ ASC · ↓ DESC) je Spalte; Default `submittedAt DESC`. Status in URL-Params `?sort=…&order=…` persistiert, Filter-Reset bewahrt die Sortierung.
- Backend: `sort` + `order` Query-Parameter auf `GET /api/admin/applications`. Strict-Whitelist im Repo (`allowedSortColumns`) — kein SQL-Injection-Risiko. Name-Sort nutzt `COALESCE(NULLIF(firstname+lastname), company_name)`, damit Privat- und Firmen-Einträge in einer alphabetischen Reihenfolge erscheinen.

### Behoben — Architektur-Review-Sweep (Chart 1.6.16 → 1.7.7)

Bündel kleinerer und kritischer Verbesserungen, motiviert durch einen umfassenden Architektur-Review vor dem Ramp-up auf mehr User:

#### Datenintegrität / silent-data-loss

- **AdminNoteEditor schickte einen vollen `PUT /applications/{id}` mit nur dem Notiz-Feld** — Backend macht für `meteringPoints` einen REPLACE, sodass jedes Notiz-Speichern auf einem Firmen-/Vereins-Antrag die `participationFactor`-Werte aller Zählpunkte auf `0` zurücksetzte. Neuer dedizierter Endpoint `PATCH /api/admin/applications/{id}/admin-note` schreibt nur die `admin_note`-Spalte; Frontend nutzt `setAdminNote()` aus dem API-Client.
- **Duplicate-Draft-Falle**: `createApplication` → `submitApplication`-Flow ohne ID-Cache produzierte bei Submit-Fehler + Retry einen zweiten Draft. App-ID + Form-Values-Snapshot werden jetzt in `useRef` gespeichert; Retry ohne Edits überspringt `create`. 404-Response invalidiert den Cache.
- **Superuser-Bulk-Delete löschte 0 Anträge**: Der Handler ließ `rcNumbers` für Superuser leer, das Repo machte daraus einen Early-Return mit 0 Löschungen. Eigene `DeleteAllDrafts()` ohne Scope für Superuser, alte `DeleteDraftsByRCNumbers()` weiterhin für Tenant-Admins. Log-Line zeigt `superuser=true/false`.
- **Frontend `adminRequest` überschrieb Authorization**: Bei Aufrufen, die eigene `headers: {...}` mitgaben, wurde der Bearer-Token verschluckt → 401 `duration_ms=0`. Headers werden jetzt explizit gemerged statt gespreaded.

#### Security-Härtung

- **Body-Size-Limits per Route-Gruppe** via neuer `MaxBodySize`-Middleware: 256 KiB für `/api/public` und `/api/external`, 1 MiB für `/api/admin`. Schließt unbounded-Body-DoS-Surface.
- **Trusted-Proxy-CIDR für `realIP()`**: Header `X-Real-IP` / `X-Forwarded-For` werden nur akzeptiert, wenn `r.RemoteAddr` aus den konfigurierten CIDRs kommt (env `TRUSTED_PROXY_CIDRS`, default in Helm: typische K8s-Pod/Service-CIDRs). Verhindert Spoofing des per-IP-Rate-Limits.
- **NetworkPolicies** (opt-in via `networkPolicies.enabled`, default true): `backend ← frontend + ingress`, `frontend ← ingress`, `postgres ← backend + migrate + seed` (NICHT Frontend). Defense-in-Depth gegen kompromittierte NPM-Transitives im Frontend-Pod.
- **Status-Transition `imported → approved`** bereits in PROJ-30 ausschließlich über dedizierten Endpoint (`POST /reset-import`) erreichbar, nie über die generische `/status`-Route.

#### Resilience

- **Health-Probes gesplittet**: Backend bekommt `/livez` (always 200, kein DB-Touch) und `/readyz` (DB-Ping). Frontend bekommt `/api/health` (always 200, kein Backend-Call). Helm-Probes umgestellt — DB-Blip kann nicht mehr per `livenessProbe` einen Restart-Loop auslösen, Backend-Outage kaskadiert nicht in Frontend-NotReady.
- **AbortController** in Admin-Web-Fetches (Liste, Detail, Tariff-Dialog): `useEffect`-Cleanup mit `AbortController`, `signal` durch `adminRequest`. Race-Condition bei schneller Navigation / Tariff-Dialog-EEG-Wechsel beseitigt.
- **Zentrales 401-Handling**: `adminRequest` emittiert `auth:expired`-Event auf 401; `SessionRefreshGuard` triggert `signIn("keycloak")`. User landen auf Keycloak-Login statt rote Error-Banner bei abgelaufenen JWTs.
- **`tzdata` in Go-Binary**: `_ "time/tzdata"` Blank-Import in `internal/shared/timezone.go`. `time.LoadLocation("Europe/Vienna")` funktionierte im Alpine-Container nicht, weil Alpine standardmäßig kein `tzdata`-Paket hat → Helper fiel still auf UTC zurück trotz aller PDF/Mail/Frontend-TZ-Migration. ~450 KB Binary-Overhead.

#### Operations

- **Velero-Pre-Backup-Hook am Postgres-StatefulSet** (`pre.hook.backup.velero.io/command: psql -c CHECKPOINT;`) — Cluster-Velero macht jetzt konsistente CSI-Snapshots statt Crash-Recovery-Restore.
- **`docs/operations.md`** als App-spezifisches Runbook: Backup-Scope + RPO/RTO, Restore-Verfahren (Namespace-only, PVC-only, Full-Cluster), 7-Punkte Post-Restore-Checklist, 4 Incident-Szenarien (Core-Outage, SMTP-Down, Lastspitze, Velero-Alert), Deployment + Rollback, bekannte Einschränkungen.
- **Slim `checkTenantAccess`**: Neue `GetRCNumberByID`-Query statt voller `GetApplicationDetail` (sparte ~4 Round-Trips pro Admin-Click).

#### Mail / Spam-Deliverability

Bestätigte Analyse einer realen Production-Mail: DKIM=pass (`postal-TA3f2w._domainkey.eegfaktura.at`), SPF=pass (via `psrp.eegfaktura.at`-Subdomain-Delegation), DMARC=pass. **Authentication ist bereits korrekt** — keine DNS-Änderungen erforderlich. Content-/Header-seitige Optimierungen:

- **From-Header mit Display-Name**: `"eegFaktura Mitglieder-Onboarding" <noreply@eegfaktura.at>` via neuer Env `SMTP_FROM_NAME` und `msg.FromFormat()`. Legitimitäts-Signal für Inbox-Provider.
- **Reply-To pro Mail-Typ**: Member-Bestätigung → EEG-Contact-Email; EEG-Notification + Approval → Antragsteller-Email. Replies auf `noreply@` haben damit ein sinnvolles Ziel.
- **`Auto-Submitted: auto-generated`** (RFC 3834) auf allen Mails. Transaktional-Indikator für Gmail; bricht Out-of-Office-Loops.
- **`User-Agent` + `X-Mailer`** via `SetUserAgent()` beide auf `"eegFaktura Member Onboarding"` (statt gomail-Default `go-mail v0.7.2 // github…`, der manche Filter triggert).
- **`Message-ID`**: `<random-hex>@eegfaktura.at` statt `<…@member-onboarding-test-backend-9df68fbc9-wlsq4>` (Pod-Hostname).
- **Plain-Text-Alternative verbessert**: `htmlToText` rendert Tabellen als `Label: Wert`, Links als `text (url)`, strippt `<head>`/`<style>`/`<script>` vor Tag-Entfernung. Schließt die HTML-vs-Plain-Divergenz, die klassische Spam-Filter flaggen.
- **Identification-Footer** in allen 3 Templates: Grund der Mail, Sender-Identifikation, Hinweis dass Reply-Path funktioniert.

#### Tests + Doku

- `internal/mail/mailer_test.go` neu: 4 Tests gegen Multipart-Struktur, Headers, User-Agent-Branding, Message-ID-Domain.
- `docs/architecture.md` ergänzt um Time/Timezone-Konvention und (siehe oben) Resilience-Bausteine.

### Neu — Mitgliedsnummer wird beim Import vergeben (statt beim Submit)

Die Mitgliedsnummer ist im Core-System die Quelle der Wahrheit. Das Onboarding kennt erst zum Import-Zeitpunkt den aktuellen höchsten Wert. Die Pflege im Onboarding (`registration_entrypoint.member_number_start` + Auto-Assign in `AssignMemberNumberTx`) wird durch eine Live-Abfrage am Core ersetzt.

- **Neuer Endpoint** `GET /api/admin/applications/{id}/next-member-number` — ruft Core `GET /participant`, ermittelt nächste freie Nummer
- **Pattern-aware Vorschlag**: Algorithmus erkennt dominantes Muster (Präfix + Padding). `A001, A002, A005` → Vorschlag `A006`. `M-12, M-13` → `M-14`. Reine Ziffern: `1, 2, 3` → `4`. Padding wächst (`01, 99` → `100`). Bei gemischten Mustern gewinnt die Gruppe mit den meisten Einträgen.
- **String-typed**: Migration 000027 promoted `application.member_number` von `INT` auf `TEXT`, weil Core `participantNumber` `VARCHAR` ist. Models, Repo, Payload, PDF, Excel, Frontend-Types durchgängig string.
- **Pre-Import-Duplikat-Check** im Backend: vor `POST /participant` wird die gewählte Nummer gegen die Core-Teilnehmerliste verglichen; bei Doppelvergabe 409.
- **Tariff-Dialog erweitert** um „Mitgliedsnummer"-Input (Pflichtfeld, max 50 Zeichen, mit Vorschlag-Prefill).
- **AdminEditForm**: Mitgliedsnummer-Feld entfernt.
- **AdminEEGSettingsEditor**: „Mitgliedsnummer Startwert"-Feld entfernt; Spalte `registration_entrypoint.member_number_start` bleibt im Schema (unbenutzt).
- **`AssignMemberNumberTx`** Call beim Submit ist raus. `application.member_number` ist von Submit bis Import `NULL`; das Approval-PDF rendert die Spalte erst nach erfolgreichem Import.

### Neu — Click-to-Sort, Auth-Loop-Cooldown, Import-Robustheit

#### Click-to-Sort auf der Admin-Liste
- Server-seitige Sortierung mit strict Allowlist (`allowedSortColumns`); URL-persistierte `sort`/`order`-Parameter; Pfeil-Icons (↕/↑/↓) im Header
- „Name"-Sortierung nutzt `COALESCE(NULLIF(TRIM(CONCAT_WS(' ', firstname, lastname)), ''), company_name)` — Privatpersonen und Firmen mischen alphabetisch korrekt

#### Auth-Loop nach Deploy
- 401 → `signIn("keycloak")` → Keycloak-Roundtrip → 401 (neuer Pod noch nicht ready) → Loop. Behoben mit sessionStorage-basiertem 30s-Cooldown der die Page-Navigation überlebt. Zweite 401 innerhalb des Cooldowns triggert keinen erneuten Redirect; Banner „Anmeldung erforderlich, aber automatische Weiterleitung wurde unterdrückt".

#### Import-Robustheit-Bündel
- **Import-Context detachen**: nach `MarkImportInFlight` läuft der Core-Call auf `context.WithTimeout(context.Background(), 2*time.Minute)`. Browser-Close oder Network-Drop unterbricht den Core-Call nicht mehr → keine Orphan-Participants im Core + Duplikat bei Retry.
- **ResetImportTx mit `SELECT ... FOR UPDATE`**: explizite Row-Lock + Pre-Check `(import_started_at NOT NULL AND import_finished_at IS NULL)`. Reset während laufenden Imports = 409 statt Race.
- **Migration 000028**: partial UNIQUE Index `(rc_number, member_number) WHERE NOT NULL` als Defense-in-Depth gegen Doppelvergabe.

### Neu — Observability: Prometheus /metrics

Counter (Namespace `eegfaktura_mo`): `applications_submitted_total`, `imports_total{result}`, `mail_sent_total{kind,result}`, `rate_limit_hits_total`, `member_number_lookups_total{result}`, `http_request_duration_seconds{method,status_class}`. Bundled `go_*` + `process_*`.

- **Separater HTTP-Server auf :9090** (env `METRICS_PORT`, default `9090`), bewusst NICHT durch den Public-Ingress geroutet
- **Helm**: dedizierter ClusterIP-Service (`backend-metrics`) mit `prometheus.io/scrape`-Annotationen; optional `ServiceMonitor` (`metrics.serviceMonitor.enabled`) für prometheus-operator-Stacks; NetworkPolicy erlaubt Ingress aus `networkPolicies.prometheusNamespace` (Default `cattle-monitoring-system` für Rancher)
- **Counter-Overhead vernachlässigbar** (Nanosekunden pro `Inc()`); deaktivierbar via `metrics.enabled: false`

### Performance — Quickwins-Bündel

- **Migration 000029**: composite indexes `(application_id, created_at)` auf `status_log`, `document_consent`, `metering_point`. Admin-Detail-View liest jetzt ohne heap-fetch + sort.
- **Deep-Pagination-Cap**: `page > 10_000` wird gedeckelt — kein OFFSET-Scan über Millionen Zeilen durch Buggy-Clients.
- **„Alle Entwürfe löschen"-Dialog respektiert `rc_number`-Filter**: Count + Delete-Call führen den aktiven Filter mit. Multi-EEG-Admin kann nicht mehr versehentlich über alle EEGs hinweg löschen.

### External-API Scope-Review (Befund)

Audit: `/api/external/*` exponiert ausschließlich `POST /v1/applications` mit API-Key-Auth. Keine Liste/Detail-Endpoints, keine RC-Number-Enumeration, keine Admin-Operations. **Scope ist bereits minimal**, keine Cleanup-Arbeit notwendig.

---

## [v1.10.0] - 2026-05-09

### Neu — PROJ-4: Core Import

Synchroner Import genehmigter Anträge in das eegFaktura-Core-System.

- **Backend**: `POST /api/admin/applications/{id}/import` ruft den Core-Endpoint `POST /participant` auf. Bearer-Token des angemeldeten Admins wird durchgereicht, `tenant`-HTTP-Header wird auf die RC-Nummer der Application gesetzt.
- **Architektur**: neue Pakete `internal/coreclient` (HTTP-Wrapper) und `internal/importing` (Orchestrierung + Payload-Mapping)
- **Concurrency-Sperre**: `MarkImportInFlight` verhindert Duplikate im (nicht-idempotenten) Core durch parallele Klicks
- **Defense-in-Depth**: Service-Level-Tenant-Check zusätzlich zum Handler-Check
- **Frontend**: Status-Aktionen-Box zeigt „In eegFaktura importieren" für `approved`-Anträge, „Import erneut versuchen" + Error-Banner für `import_failed`, sowie die Participant-ID nach erfolgreichem Import
- **Konfig**: `CORE_BASE_URL` (mit `/api`-Suffix) und `CORE_TIMEOUT_SECONDS` als neue Env-Vars; via Helm-Values `backend.coreBaseUrl` durchgereicht

### Erkenntnisse aus dem Live-Rollout

- **Keycloak Tenant-Mapper**: muss `Claim JSON Type: JSON` haben (nicht `String`), sonst lehnt der Core mit 401 leerem Body ab
- **businessRole** muss gesetzt werden (`EEG_PRIVATE` / `EEG_BUSINESS`), sonst Privat-Tab im UI auch für Firmen
- **firstname** der Core-Tabelle ist NOT NULL — für Firmen/Vereine/Gemeinden wird der Organisationsname dort eingestellt
- **Meter-Direction**: Onboarding `PRODUCTION` → Core `GENERATION`

Details siehe `features/PROJ-4-core-import.md` und `docs/import-mapping.md` §7–§9.

### Geändert

- `coreclient`: UTF-8-sichere Truncation, erkennt zusätzlich `context.Canceled` und `net.Error.Timeout()`, klare Sentinel-Errors
- `ImportService`: Bookkeeping-Failure nach Core-Erfolg loggt Participant-ID + surface in Result (Operator kann manuell aufräumen)
- Handler nutzt `errors.Is`/`errors.As` für robuste Error-Routing über Wrapping hinweg

### Infrastruktur

- Helm-Chart erweitert um `backend.coreBaseUrl` und `backend.coreTimeoutSeconds`
- `values-env.yaml.example` dokumentiert beide Werte mit Beispiel inkl. `/api`-Suffix

---

## [v1.9.0] - 2026-04-30

### Neu
- **Admin-GUI**: Button „Beitrittsbestätigung herunterladen" in der Antragsdetailansicht (`GET /api/admin/applications/{id}/approval-pdf`) für Status `approved`, `imported`, `import_failed`
- **Mitglieds-Bestätigungs-E-Mail**: Enthält jetzt alle eingegebenen Antragsdaten (Persönliche Daten, Adresse, Bankverbindung, Zählpunkte) und alle erteilten Zustimmungen

### Geändert
- **Beitrittsbestätigung PDF**: Mitgliedsnummer wird als erster Eintrag in MITGLIEDSDATEN angezeigt (kein leeres Leerfeld mehr)
- **Beitrittsbestätigung PDF**: Zustimmungen vollständig — Datenschutz (mit Version), Richtigkeit, SEPA (Checkbox oder „Per E-Mail übermittelt"), Dokumentzustimmungen mit Datum
- **Beitrittsbestätigung PDF**: Statusverlauf-Labels auf Deutsch (z. B. „Eingereicht" statt „submitted")
- **SEPA-Mandat**: Kontoinhaber-Feld wird ausschließlich aus `AccountHolder` befüllt — kein automatischer Fallback auf Vorname/Nachname mehr

### Infrastruktur
- Vitest-Konfiguration auf `.mts` umgestellt (behebt `ERR_REQUIRE_ESM`-Fehler bei `npm test`)
- Dokumentation aktualisiert: `docs/domain-model.md`, `docs/api-spec.md`, Feature-Specs PROJ-21 und PROJ-6

---

## [v1.8.0] - 2026-04-29

### Neu — PROJ-25: Bulk-Aktionen im Admin
- Mehrere Anträge gleichzeitig genehmigen, ablehnen oder zur Prüfung setzen
- Checkboxen pro Zeile + „Alle auswählen"-Checkbox mit indeterminate-State
- Aktionsleiste erscheint bei aktiver Auswahl mit Bestätigungsdialog
- Ergebnis-Zusammenfassung nach Ausführung (X erfolgreich, Y übersprungen)
- Backend: `POST /api/admin/applications/bulk-action` mit Tenant-Isolation; max. 200 Anträge pro Request; ungültige Transitionen werden übersprungen (kein Fehler)

### Neu — PROJ-24: OpenAPI/Swagger Dokumentation
- Interaktive Swagger UI unter `/swagger/` verfügbar
- Alle Admin- und Public-Endpunkte vollständig annotiert (Swaggo)
- Automatische Swagger-Generierung via `swag init` in CI

---

## [v1.7.0] - 2026-04-26

### Neu — PROJ-20: Vollständige Antragsdaten in EEG-Einreichungsbenachrichtigung
- EEG-Betreiber erhält bei jeder Neueinreichung alle Antragsdaten per E-Mail
- Felder: Mitgliedstyp, Name/Firma, Adresse, Kontakt, IBAN, SEPA-Ermächtigung, Zählpunkte, konfigurierbare Felder
- Konfigurierbare Felder werden nur angezeigt wenn nicht `hidden` und befüllt
- Optionaler Admin-Link zur Detailansicht (via `ADMIN_BASE_URL`-Umgebungsvariable)

### Neu — PROJ-21: Genehmigungs-Benachrichtigung mit Beitrittsbestätigung PDF
- Bei Status-Übergang → `approved` erhält die EEG automatisch eine E-Mail mit PDF-Anhang
- PDF „Beitrittsbestätigung" enthält: Mitgliedsdaten, Bankverbindung, Zählpunkte, Zustimmungen, Statusverlauf, konfigurierbare Felder
- PDF-Generierung schlägt fehl → E-Mail wird trotzdem gesendet (mit Hinweistext); Status-Übergang bleibt gültig
- Re-Approval (`import_failed → approved`) sendet erneut eine E-Mail

---

## [v1.6.0] - 2026-04-25

### Neu — PROJ-9: EEG-spezifische Rechtsdokumente
- Admin kann beliebige Rechtsdokumente pro EEG konfigurieren (Satzung, AGB usw.)
- Mitglied muss Pflichtdokumente vor Einreichung bestätigen
- Zustimmungen werden als unveränderliche Snapshots gespeichert (`document_consent`)
- Max. 10 Dokumente pro EEG; sortierbar per Drag-and-Drop

### Neu — PROJ-16: Cloudflare Turnstile Spam-Schutz
- Öffentliches Registrierungsformular mit Turnstile-CAPTCHA geschützt
- Aktivierung via `TURNSTILE_SECRET_KEY`-Umgebungsvariable (fehlt → deaktiviert)

### Neu — PROJ-17: Excel-Export für eegFaktura-Import
- Admin kann Antrag als `.xlsx`-Datei exportieren (`GET /api/admin/applications/{id}/export/excel`)
- Datei im eegFaktura-Importformat (36 Spalten, eine Zeile pro Zählpunkt)
- Nur für Status `approved`, `imported`, `import_failed`

### Neu — PROJ-18: Datenschutzerklärung & Central Policy Toggle
- Zentrale Datenschutzerklärung (Betreiber-Policy) über Umgebungsvariablen konfigurierbar (`CENTRAL_POLICY_TITLE`, `CENTRAL_POLICY_URL`)
- Pro EEG einstellbar, ob die zentrale Policy im Formular angezeigt wird (`showCentralPolicy`)
- EEGs mit eigener Datenschutzerklärung können die zentrale Policy ausblenden

### Neu — PROJ-19: Manuelle Aktivierung der Registrierung
- Neue EEGs sind standardmäßig inaktiv (`is_active = false`)
- Admin kann Registrierung pro EEG aktivieren/deaktivieren (Settings-Seite)
- Inaktive EEGs: öffentliches Formular liefert `410 Gone`

---

## [v1.5.0] - 2026-04-24

### Neu — PROJ-12: SEPA-Lastschriftmandat PDF
- Automatische Generierung eines SEPA-Lastschriftmandats als PDF-Anhang in der Mitglieds-Bestätigungs-E-Mail
- Aktivierung pro EEG via `sepaMandateEnabled`-Einstellung
- Unterstützt CORE- und B2B-Mandat
- Kann auch per E-Mail zugesandt werden (`sepa_mandate_enabled = false`): Hinweis im PDF und in der Bestätigungs-E-Mail

### Neu — PROJ-13: Externe Registrierungs-API
- `POST /api/external/v1/applications` — Anträge direkt aus externen Systemen einreichen
- API-Key-Authentifizierung (kein Keycloak); Key pro EEG generierbar/widerrufbar in den Admin-Settings
- Rate Limiting: 10 Requests / 60 Sekunden (Burst) + 200 Einreichungen / Tag (Quota)

### Neu — PROJ-14: SEPA-Firmenlastschriftmandat
- Für Mitglieder vom Typ `company` / `association` kann ein SEPA-B2B-Mandat statt des Standard-CORE-Mandats generiert werden
- Steuerbar über EEG-Einstellung `useCompanySEPAMandate`

### Neu — PROJ-15: Konfigurierbare Felder Erweiterungen
- Neuer Feld-Status `admin_only`: Feld ist im öffentlichen Formular verborgen, wird aber mit einem konfigurierten Admin-Standardwert automatisch befüllt
- Zählpunktfelder konfigurierbar: `transformer`, `installation_number`, `installation_name`

---

## [v1.4.0] - 2026-04-23

### Neu — PROJ-11: Konfigurierbarer Einleitungstext
- Admin kann pro EEG einen Einleitungstext für das Registrierungsformular hinterlegen (HTML, sanitisiert)
- Wird im öffentlichen Formular über dem Antragsformular angezeigt

---

## [v1.3.0] - 2026-04-22

### Neu — PROJ-8: Konfigurierbare Felder pro EEG
- Admin kann pro EEG konfigurieren, welche optionalen Felder im Registrierungsformular sichtbar, versteckt oder Pflicht sind
- Konfigurierbare Felder: `phone`, `birth_date`, `uid_number`, `membership_start_date`, `persons_in_household`, `consumption_previous_year`, `consumption_forecast`, `feed_in_forecast`, `pv_power_kwp`, `heat_pump`, `electric_vehicle`, `electric_hot_water`

---

## [v1.2.0] - 2026-04-21

### Neu — PROJ-6: E-Mail-Benachrichtigungen
- Mitglieds-Bestätigungs-E-Mail nach erfolgreicher Einreichung
- EEG-Benachrichtigungs-E-Mail an `contact_email` der EEG
- Asynchroner Versand (kein Blockieren der Einreichung bei SMTP-Fehler)
- Resend-Funktion im Admin: „Bestätigung erneut senden"
- Konfiguration via `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`

---

## [v1.1.0] - 2026-04-20

### Neu — PROJ-5: Keycloak-gesicherte Admin-Oberfläche
- Admin-Bereich erfordert Keycloak-Login (JWT Bearer Token)
- Tenant-Isolation: Admins sehen nur Anträge ihrer eigenen EEGs
- Superuser-Flag für EEG-übergreifenden Zugriff

### Neu — PROJ-7: Mitgliedstypen
- Unterstützung für fünf Mitgliedstypen: Privatperson, Landwirt, Gemeinde, Unternehmen, Verein
- Typenspezifische Felder (Firmenname, UID-Nummer, Firmenbuchnummer)
- Kompakte Select-UI im Registrierungsformular

---

## [v1.0.0] - 2026-04-19

### Neu — PROJ-1: Öffentliche Registrierung
- Öffentliches Registrierungsformular unter `/register/{rc_number}`
- Antragstellung mit Personendaten, Adresse, IBAN, Zählpunkten
- Mehrschrittiges Formular mit Validierung (Frontend + Backend)
- Antragsstatus: `draft` → `submitted`

### Neu — PROJ-2: Admin-Review
- Admin kann Anträge einsehen, bearbeiten und Status ändern
- Status-Workflow: `submitted → under_review → approved / rejected / needs_info`
- Admin-Notiz und Rückfrage-Grund pro Antrag

### Neu — PROJ-3: Admin-Frontend-UI
- Antragsübersicht mit Filter und Pagination
- Detailansicht mit vollständigen Antragsdaten
- Status-Aktionen direkt aus der Detailansicht
