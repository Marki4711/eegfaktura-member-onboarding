# Changelog

Alle nennenswerten Änderungen an diesem Projekt werden hier dokumentiert.
Format basiert auf [Keep a Changelog](https://keepachangelog.com/de/1.0.0/).

> Die Versionsnummern im CHANGELOG sind unabhängig von den Git-Tags vergeben,
> da die ursprünglichen Tags nicht konsistent nummeriert wurden.

---

## [Unreleased]

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
