# PROJ-60: Datenweiterleitung an externe Systeme — Plugin-Framework

## Status: Architected
**Created:** 2026-05-23
**Last Updated:** 2026-05-23 (`/architecture` abgeschlossen, Tech Design ergänzt)
**Quelle:** Owner-Anforderung

## Dependencies
- Requires: PROJ-4 (Core Import) — Daten-Weiterleitung sinnvoll nach `imported`/`activated`, technisch für jeden Status möglich
- Requires: PROJ-17 (Excel-Export für eegFaktura-Import) — `excelize`-Library wird vom Excel-Plugin wiederverwendet
- Requires: PROJ-25 (Bulk-Aktionen) — Datenweiterleitung ist neue Bulk-Aktion in der Antragsliste
- Berührt: PROJ-46 (Post-Import-Stati) — exportierbar/sync-bar für jeden Status
- Berührt: PROJ-8 (konfigurierbare Felder pro EEG) — nur konfigurierte Felder werden als Mapping-Optionen angeboten
- Berührt: Bestehendes K8s-CronJob-Pattern (z. B. `restart-cronjob.yaml`) — wiederverwendet für Job-Cleanup

## Hintergrund

EEG-Vereine pflegen ihre Mitglieder teils nicht nur im
eegFaktura-Core, sondern parallel in einem eigenen System — z. B.
einem CRM (Zoho, HubSpot, Salesforce), Newsletter-Tool, Vereins-
Datenbank oder Excel-Datei für interne Auswertungen.

Aktuell muss der EEG-Admin diese parallele Pflege manuell machen.
Das ist fehleranfällig, ineffizient und nicht skalierbar.

## Architektur-Vision

Statt einer fest verdrahteten „Excel-Export"-Funktion bauen wir ein
**generisches Datenweiterleitungs-Framework** mit Plugin-Architektur:

- **Framework** definiert das Plugin-Interface, die asynchrone
  Job-Mechanik, die Auswahl-UI, das Audit-Log und das Failure-Handling
- **Plugins** implementieren je einen konkreten Ziel-Kanal:
  - **Phase 1 (dieses PROJ-60):** Excel/CSV-Export-Plugin
  - **Phase 2 (späteres PROJ):** Zoho-CRM-Plugin
  - **Phase 3+:** HubSpot, Salesforce, Pipedrive, Mailchimp, …

Phase 2 wird auf „Plugin schreiben + UI-Komponenten dazu" reduziert
(~3–4 Tage statt ~5–7), weil Framework, DB-Struktur, Audit-Log,
Trigger-UI, Async-Job-Queue, Recovery-Mechanik und Tenant-Isolation
bereits stehen.

### Async-Framework als Foundation

**Wichtige Architektur-Entscheidung** (`/grill-me` 2026-05-23): das
Framework ist **von Anfang an asynchron**. Begründung:

- Excel-Plugin generiert kleine Dateien (typisch <100 KB für 500 Anträge), könnte synchron laufen
- Push-Plugins (Phase 2) machen API-Calls mit Rate-Limiting → bei 1.000 Anträgen 5–10 Minuten → HTTP-Timeout-Garantie
- Sync-V1 + Async-V2-Refactor würde alle V1-Patterns (Endpoint-Form, UI-Dialog, Audit-Schema) brechen

→ Async-Pipeline auch für Excel: User-Klick erzeugt Job in Queue, Worker
verarbeitet, UI pollt Status, bei Done erscheint Download-Link (Excel)
oder Sync-Status (Push).

Overhead bei kleinen Excel-Exports: ~1 Sek zusätzliche Wartezeit
(Job-Pickup + UI-Polling-Intervall). Akzeptabel.

## Plugin-Interface (Konzept)

| Methode | Zweck |
|---|---|
| `Type()` | Plugin-Typ-Identifikator (z. B. `"excel"`, `"zoho_crm"`) |
| `DisplayName()` | Anzeige-Name für Admin-UI |
| `ValidateConfig(config)` | Prüft Plugin-spezifische Konfiguration auf Plausibilität |
| `Process(ctx, configSnapshot, applications) → Result` | Hauptlogik. **Erhält Snapshot der Config**, nicht Live-Reference, damit gleichzeitige Config-Änderungen den Job nicht brechen |
| `StandardConfigs()` | Liefert vordefinierte Standard-Konfigurationen (z. B. drei Excel-Templates) als read-only Klon-Basis |

**Result-Typen:**
- `DownloadResult`: enthält Datei-Bytes + MIME-Type + Dateiname → wird in DB als BLOB gespeichert, User lädt via UI-Link
- `SyncResult`: enthält Status-Map pro Antrag (created / updated / failed) → Audit-Log + UI-Anzeige

## DB-Schema

```
data_export_config
  - id UUID PK
  - rc_number TEXT FK → registration_entrypoint
  - plugin_type TEXT (z. B. "excel", "zoho_crm")
  - name TEXT (UNIQUE per rc_number, cross-plugin-type)
  - config JSONB (plugin-spezifisch)
  - is_obsolete BOOLEAN DEFAULT FALSE (wenn plugin_type aus Backend entfernt wurde)
  - created_at, updated_at

data_export_job (Job-Queue + State)
  - id UUID PK
  - rc_number TEXT FK
  - config_id UUID FK NULL (NULL wenn Config inzwischen gelöscht)
  - config_snapshot JSONB (Snapshot der Config zum Job-Trigger-Zeitpunkt)
  - plugin_type TEXT (Snapshot, damit Plugin-Removal nicht das Audit zerstört)
  - application_ids UUID[] (Snapshot der ausgewählten Antrags-IDs)
  - status TEXT CHECK (status IN ('queued','running','done','failed','expired'))
  - admin_user_id TEXT
  - created_at, started_at, finished_at TIMESTAMPTZ
  - result_summary JSONB (z. B. {"downloaded": 47} oder {"synced": 45, "failed": 2})
  - error_message TEXT NULL
  - retry_count INTEGER DEFAULT 0
  - INDEX (status, created_at) für effizientes Queue-Polling
  - INDEX (rc_number, status) für „aktive Jobs pro EEG"-Check (Concurrency-Limit)

data_export_result (Datei-Storage für DownloadResults)
  - job_id UUID PK FK → data_export_job
  - file_name TEXT
  - mime_type TEXT
  - file_bytes BYTEA
  - file_size INTEGER (für UI-Anzeige + Stats)
  - downloaded_at TIMESTAMPTZ NULL (für Audit, optional)
  - expires_at TIMESTAMPTZ (created_at + 24h, Cleanup-Cron räumt auf)
```

**Audit-Trail-Strategie:**
- `data_export_job` ist langlebig (persistiert über TTL hinaus)
- Bei Cleanup wird **nicht** die Job-Zeile gelöscht, sondern nur die `data_export_result`-Zeile + Job-Status auf `expired` gesetzt
- Damit bleibt der vollständige Audit-Trail (Snapshot der Config, IDs, Result-Summary) langfristig erhalten

## Acceptance Criteria

### Framework-Komponenten

- [ ] DB-Migrationen für `data_export_config`, `data_export_job`, `data_export_result`
- [ ] Backend-Plugin-Registry (Build-Time / Compile-In via Go-Imports, Pattern wie `sql.Driver`-Registry)
- [ ] Pro EEG max. **20 aktive Konfigurationen** (über alle Plugin-Typen)
- [ ] Eindeutige Namen pro EEG (cross-plugin-type-validation)
- [ ] Pro EEG max. **3 parallele Jobs** (Status `queued` oder `running`) — überzählige werden auf `queued` gesetzt und warten

### Job-Queue + Worker

- [ ] Job-Erzeugung beim Bulk-Action-Trigger: Job-Row mit `status='queued'`, Config wird zum Snapshot kopiert
- [ ] **In-App-Goroutine-Pool** im Backend-Pod als Worker (kein separater Worker-Pod)
- [ ] Worker pollt periodisch (z. B. alle 5 Sek) die Queue mit `SELECT ... FOR UPDATE SKIP LOCKED LIMIT 1 WHERE status='queued'`
- [ ] Multi-Replica-safe via Row-Locking
- [ ] Worker setzt `status='running'`, `started_at=NOW()` beim Pickup
- [ ] Bei erfolgreicher Verarbeitung: `status='done'`, `result_summary` gefüllt, bei DownloadResult auch `data_export_result`-Zeile angelegt
- [ ] Bei Fehler: `status='failed'`, `error_message` gefüllt

### Job-Recovery (Zombie-Cleanup)

- [ ] **K8s-CronJob `data-export-cleanup`** (alle 10 Min)
  - findet Jobs mit `status='running'` und `started_at < NOW() - 1h`
  - setzt sie auf `status='failed'` mit `error_message='zombie cleanup — worker did not finish'` und `retry_count++`
  - Admin sieht im BackOffice die Failed Jobs und kann manuell re-triggern (Mechanik: einfach Bulk-Action wiederholen)
- [ ] Derselbe Cron räumt `data_export_result`-Zeilen mit `expires_at < NOW()` auf (Hard-Delete der BLOB-Zeile, Job-Zeile bleibt mit `status='expired'`)
- [ ] Backend-Subcommand `data-export-cleanup` (analog zum `billing-quarterly`-Pattern aus dem PSP-Stack) — gleicher Helm-CronJob-Pattern

### Concurrency-Limit + Idempotenz

- [ ] Vor Job-Erzeugung Check: aktuelle Anzahl `queued`+`running`-Jobs der EEG ≤ 3 → sonst wird der neue Job trotzdem als `queued` erzeugt, aber Worker verarbeitet die Queue in FIFO-Reihenfolge
- [ ] **Keine** automatische Idempotenz-Deduplikation — zwei identische Bulk-Aktionen erzeugen zwei separate Jobs
- [ ] Konsequenz für Push-Plugins (Phase 2): müssen Plugin-intern Idempotenz handhaben (z. B. Lookup im CRM via `EEG_Onboarding_ID` vor Push)

### Admin-UI: Konfigurations-Verwaltung

- [ ] Neue Sektion „Datenweiterleitung" unter EEG-Settings
- [ ] Liste aller Konfigurationen, gruppiert nach Plugin-Typ
- [ ] Pro Plugin-Typ ein „Neue Konfiguration anlegen"-Button (Plugin-Auswahl per Dropdown — V1 nur Excel sichtbar)
- [ ] Plugin-spezifischer Editor (für Excel: Spalten-Mapping mit Live-Preview)
- [ ] Konfiguration mit `is_obsolete=true` (Plugin nicht mehr im System) wird ausgegraut mit Hinweis „Plugin nicht mehr verfügbar — nur Löschen möglich"
- [ ] Konfiguration editierbar oder löschbar (Löschen erlaubt auch wenn aktive Jobs darauf referenzieren — `config_id` wird NULL, `config_snapshot` bleibt)

### Trigger: Bulk-Datenweiterleitung aus Antragsliste

- [ ] In `admin-application-table.tsx` neue Bulk-Aktion „Datenweiterleitung" (analog PROJ-25)
- [ ] Bei ≥1 selektiertem Antrag erscheint die Aktion in der Aktionsleiste
- [ ] Klick öffnet Dialog mit **einstufiger Liste** aller konfigurierten Plugin-Konfigurationen für diese EEG, z. B.:
  - „📊 Excel: CRM-Stammdaten" → Datei-Download
  - „📊 Excel: Newsletter" → Datei-Download
  - *(später)* „☁️ Zoho: Hauptverbindung" → Push an externes System
- [ ] Hinter jedem Eintrag steht Plugin-Typ-Icon + Result-Typ-Hinweis
- [ ] Bestätigung erzeugt Job, Dialog wechselt zu Polling-Modus
- [ ] Maximum **1.000 Anträge pro Bulk-Aktion** (Frontend-Begrenzung + Backend-Defense-in-Depth)

### Trigger: Single-Datenweiterleitung aus Antrags-Detail

- [ ] In `admin-application-detail.tsx` ein Menüeintrag „Datenweiterleitung" mit derselben einstufigen Liste
- [ ] Verfügbar für jeden Antragsstatus
- [ ] Funktioniert via gleichem Job-Mechanismus (Job mit 1 Antrags-ID)

### UI-Polling + Notification

- [ ] Nach Job-Trigger zeigt UI einen Toast/Modal mit Live-Status: „queued" → „running (verarbeitet 50 von 500)" → „done"
- [ ] Polling-Intervall: 2 Sek bei kleinen Jobs (<100 Anträge), 5 Sek bei größeren
- [ ] Bei `done` + `DownloadResult`: Download-Link erscheint im Toast/Modal, klick-bar
- [ ] Bei `done` + `SyncResult` (Phase 2): Status-Report (z. B. „45 synchronisiert, 2 fehlgeschlagen"), Link zum Detail-Audit-Log
- [ ] Bei `failed`: Fehler-Meldung mit „Retry"-Button (erzeugt neuen Job mit denselben Anträgen + Config)
- [ ] Wenn User den Tab schließt während Job läuft: Job läuft im Backend weiter (kein Cancel)
- [ ] **Failure-Mail** an EEG-Admin (Adresse aus EEG-Einstellungen oder Keycloak-Profil) bei `status='failed'` — enthält Job-ID, Plugin-Typ, Fehler-Kurzbeschreibung, Link zum BackOffice

### BackOffice-Übersicht

- [ ] Tab „Datenweiterleitungs-Jobs" in EEG-Settings: Liste der letzten 100 Jobs mit Status, Zeitpunkt, Konfiguration, Anzahl Anträge, Result-Summary
- [ ] **Badge mit Anzahl Failed Jobs der letzten 7 Tage** prominent oben (Admin sieht Probleme auf einen Blick)
- [ ] Filter nach Status (Failed/Done/Running/Expired)
- [ ] Pro Job Aktionen: „Erneut ausführen" (erzeugt neuen Job mit Snapshot-Config + Snapshot-Anträgen), „Datei herunterladen" (falls noch nicht expired)

### DSGVO

- [ ] Beim Hinzufügen sensitiver Felder (IBAN, Geburtsdatum) im Excel-Mapping erscheint Warnhinweis: „Sie tragen die Verantwortung für die DSGVO-konforme Weiterverarbeitung im Zielsystem (Art. 32)."
- [ ] Bei Push-Plugins (Phase 2): zusätzlicher Setup-Hinweis „Stellen Sie sicher, dass ein eigener AVV mit [Vendor] besteht."
- [ ] Cross-Tenant-Schutz: `checkTenantAccess` für alle Endpoints — Admin sieht nur Konfigurationen + Jobs der eigenen EEG
- [ ] Persistierung: Datei-BLOB 24h-TTL, danach automatischer Hard-Delete via Cleanup-Cron. Job-Zeile (mit Config-Snapshot + Result-Summary) bleibt unbegrenzt für Audit
- [ ] **Mitglied-Widerruf wirkt erst auf zukünftige Exports** — bereits heruntergeladene Dateien sind in der Verantwortung des Admins (analog zu jeder Daten-Übergabe an Dritte)

---

## Phase 1: Excel-Plugin (erster konkreter Adapter)

### Excel-Plugin-Konfiguration (JSONB-Struktur)

```json
{
  "format": "xlsx" | "csv",
  "columns": [
    { "header": "Vorname", "field": "firstname", "format": "string" },
    { "header": "Zählpunkte", "field": "meter_numbers", "format": "comma_separated" }
  ]
}
```

### Standard-Konfigurationen für Excel-Plugin

- [ ] Drei vordefinierte read-only Vorlagen als Klon-Basis:
  1. **„Newsletter-Adressliste"** — Vorname, Nachname (oder Firma), E-Mail, Anrede
  2. **„CRM-Stammdaten"** — Vorname, Nachname (oder Firma), E-Mail, Telefon, Adresse, Mitgliedsnummer, Beitrittsdatum
  3. **„Buchhaltungs-Export"** — Mitgliedsnummer, Vorname, Nachname (oder Firma), Rechnungsadresse, IBAN, UID-Nummer

### Excel-Plugin-Spalten-Mapping-UI

- [ ] Dropdown mit allen verfügbaren Onboarding-Feldern, gruppiert nach Kategorie:
  - **Stammdaten** / **Kontakt** / **Adresse** / **Bank** / **EEG** / **Zählpunkte** / **Konfigurierbar (PROJ-8)**
- [ ] Pro Spalte: Header (frei wählbar), Feld-Auswahl, Format
- [ ] Reihenfolge per Auf/Ab-Buttons (Drag-Drop als V2)
- [ ] Mindestens 1, maximal 50 Spalten
- [ ] **Field-Hiding-Behandlung (PROJ-8):** Konfigurationen werden nicht automatisch angepasst, wenn Felder via field_config auf `hidden` gesetzt werden. Editor zeigt Warn-Badge „Feld X ist in den EEG-Einstellungen nicht aktiv — Spalte bleibt leer beim Export". Admin entscheidet manuell, ob er die Spalte entfernt.

### Excel-Plugin-Wert-Transformationen

- **Text**: 1:1
- **Datum**: „DD.MM.YYYY" (Default), „YYYY-MM-DD", „DD.MM.YYYY HH:MM"
- **Boolean**: „Ja/Nein" (Default), „true/false", „1/0", „Y/N"
- **Enum**: „Wert" (Roh) oder „Label" (lesbares Deutsch)
- **Zahl**: „DE-Format", „ISO"
- **Multi-Value (Zählpunkte)**: „comma_separated"

### Excel-Plugin-Live-Preview

- [ ] Live-Preview zeigt **letzte 5 importierten Mitglieder** dieser EEG
- [ ] Aktualisiert sich bei jeder Spalten-Änderung
- [ ] Bei keiner Mitglieder-Datenlage: anonymisierte Beispiel-Daten mit Hinweis

### Excel-Plugin-Datei-Generierung

- [ ] Dateiname: `{rc_number}-{config_name}-{YYYY-MM-DD}.{xlsx|csv}`
- [ ] XLSX via `excelize` (PROJ-17-Pattern, dynamische Spalten)
- [ ] CSV: UTF-8 mit BOM, Semikolon, Quotes bei Sonderzeichen
- [ ] Result wird als DownloadResult zurückgegeben → in `data_export_result` BLOB gespeichert

### Excel-Plugin-Zählpunkte

- [ ] Spalte „Zählpunkte" mit Format `comma_separated` → komma-getrennte Liste
- [ ] „Anzahl Zählpunkte" als separate numerische Spalte
- [ ] Detail-Felder pro Zählpunkt nicht in V1 — „Zeile pro Zählpunkt" als V2-Option

---

## Edge Cases

- **Admin selektiert Anträge mit gemischtem Status** → alle werden verarbeitet, Felder die nur bei bestimmten Status Werte haben bleiben leer
- **PROJ-8 setzt Feld auf hidden, alte Config referenziert es** → Spalte bleibt leer im Export, Editor zeigt Warnung
- **Config wird gelöscht während Job läuft** → Job läuft zu Ende mit `config_snapshot` (Live-Reference auf Config nicht nötig). Audit-Log zeigt `config_id=NULL`, aber Snapshot vollständig
- **Plugin wird in zukünftigem Release entfernt** → Configs bekommen `is_obsolete=true`, im UI ausgegraut, im Bulk-Action-Dialog ausgeblendet, aber Audit-Log + alte Jobs bleiben lesbar (`plugin_type` ist im Job-Snapshot)
- **Mitglied wird zwischen Selektion und Job-Pickup gelöscht** → Worker überspringt mit `result_summary.skipped++`
- **Worker-Pod restartet während Job läuft** → Cleanup-Cron erkennt Zombie nach 1h, setzt auf `failed` mit Retry-Möglichkeit
- **Datei-Download-Link wird nach 24h aufgerufen** → 404 „Datei abgelaufen, bitte Job erneut ausführen"
- **Mail-Versand bei Failure schlägt fehl** (SMTP down) → Failure-Notification-Mail wird in Mail-Service-Retry-Queue gepuffert. Admin sieht den Job im BackOffice-Badge auf jeden Fall
- **Admin triggert dieselbe Bulk-Action zweimal kurz hintereinander** → zwei separate Jobs, beide werden verarbeitet. Bei Excel harmlos, bei Push-Plugins muss Plugin-Idempotenz greifen (Custom-Field-Lookup)
- **Sonderzeichen in Mitgliedsdaten** → CSV-Quoting + XLSX-Native-Handling
- **Cross-Tenant-Hack via config_id-Manipulation** → 403 via checkTenantAccess
- **Config mit invalide Werten (z. B. Spalte ohne Feld)** → `ValidateConfig()` blockt beim Speichern
- **>1.000 Anträge in Bulk-Aktion selektiert** → Frontend zeigt Hinweis „bitte filtern oder mehrere Exports", Backend-Defense-in-Depth bricht ab mit 400
- **Mehrere Admins exportieren parallel** → kein Lock-Problem (Excel ist read-only), Concurrency-Limit (3 pro EEG) verhindert Misbrauch
- **EEG hat 4 Jobs gleichzeitig laufen wollen** → 4. Job bleibt `queued` bis Slot frei (FIFO)

## Technical Requirements

- **DB-Migrationen**: `data_export_config`, `data_export_job`, `data_export_result`
- **Backend-Plugin-Registry** als Map `plugin_type` → Plugin-Instanz, beim Backend-Startup initialisiert via Go-Import-Side-Effects (Pattern wie `sql.Driver`)
- **Worker-Goroutine-Pool** im Backend-Pod (configurable Pool-Size, Default 3), pollt Queue mit `FOR UPDATE SKIP LOCKED`
- **Excel-Plugin** unter `internal/dataexport/plugins/excel/`
- **Cleanup-Cron** als K8s-CronJob analog `restart-cronjob.yaml`, Backend-Subcommand `data-export-cleanup`
- **HTTP-Endpoints**:
  - `GET /api/admin/data-export/configs` — Liste der EEG-Configs
  - `POST /api/admin/data-export/configs` — neue Config (plugin-specific validation)
  - `PUT /api/admin/data-export/configs/{id}` — Config-Update
  - `DELETE /api/admin/data-export/configs/{id}` — Config-Löschen
  - `POST /api/admin/data-export/configs/{id}/preview` — Live-Preview-Endpoint (Excel-spezifisch)
  - `POST /api/admin/data-export/jobs` — neuer Job (Body: config_id + application_ids)
  - `GET /api/admin/data-export/jobs/{id}` — Job-Status (für Polling)
  - `GET /api/admin/data-export/jobs/{id}/download` — DownloadResult abrufen (nur bei `status=done` + Result vorhanden + nicht expired)
  - `GET /api/admin/data-export/jobs` — Liste der letzten N Jobs (BackOffice-Übersicht)
  - `POST /api/admin/data-export/jobs/{id}/retry` — Job erneut ausführen (erzeugt neuen Job mit Snapshot)
- **Frontend**:
  - Bulk-Action in `admin-application-table.tsx` (analog PROJ-25)
  - Single-Action in `admin-application-detail.tsx`
  - Config-Sektion + Editor in `admin-eeg-settings-editor.tsx` (oder eigene Sub-Seite)
  - Plugin-spezifische React-Komponenten in `src/components/data-export/plugins/{plugin_type}/`
  - Polling-Hook für Job-Status (z. B. `useJobPolling(jobId)`)
- **`excelize`-Wiederverwendung** aus PROJ-17 mit dynamischen Spalten
- **Mail-Service** wiederverwendet aus PROJ-6 für Failure-Notification
- **Tenant-Isolation** auf allen Endpoints via `checkTenantAccess`
- **Auth**: `eeg_admin`-Rolle reicht (kein Superuser-Privileg nötig)

## Resolved Decisions (aus `/requirements` 2026-05-23 + `/grill-me` 2026-05-23)

### Architektur
- **Plugin-Framework statt fest verdrahteter Excel-Logik** — Excel ist erster Plugin, Phase 2 (Zoho/HubSpot/…) sind weitere Plugins im selben Framework
- **Async-Framework mit Job-Queue von Anfang an** — auch Excel läuft async (1 Sek Overhead akzeptiert), spart V2-Refactor
- **Plugin-Registry Build-Time / Compile-In** (Go-Import-Pattern wie `sql.Driver`) — keine Runtime-Discovery
- **Worker als In-App-Goroutine-Pool** im Backend-Pod mit `FOR UPDATE SKIP LOCKED` für Multi-Replica-Safety
- **Job-Recovery via Cleanup-Cron** (K8s-CronJob, alle 10 Min) — Zombie-Jobs nach 1h auf `failed`, Datei-BLOBs nach 24h-TTL hard-gelöscht
- **Config-Snapshot at Job-Creation** — Job-Record enthält JSONB-Snapshot der Config zum Trigger-Zeitpunkt, Live-Config-Änderungen brechen laufende Jobs nicht

### UX
- **Einstufige Liste im Bulk-Action-Dialog** („Excel: Newsletter", später „Zoho: Hauptverbindung") statt zweistufiger Plugin-Auswahl
- **Frontend-Pluggability:** pro Plugin eigene React-Komponenten, fest verdrahtet (kein JSON-Schema-Dynamic-Forms)
- **UI-Polling + Toast für Job-Status** + Download-Link bei Done (statt synchroner Datei-Stream)
- **Failure-Notification:** UI-Badge im BackOffice (Failed-Jobs-Counter) + Mail an EEG-Admin bei jedem Fail
- **Concurrency-Limit:** max 3 parallele Jobs pro EEG (Anti-Misuse, kein Workflow-Block)
- **Keine Idempotenz-Deduplikation** — zwei identische Bulk-Actions erzeugen zwei separate Jobs (Admin kann bewusst neu starten; Push-Plugins müssen Plugin-intern dedup-pen)

### Excel-Plugin
- **Zählpunkte komma-getrennt** in einer Spalte (V1), „Zeile pro Zählpunkt" als V2-Option
- **3 vordefinierte Standard-Vorlagen** (Newsletter, CRM-Stammdaten, Buchhaltung) als read-only Klon-Basis
- **Field-Hiding-Behandlung:** Konfigurationen werden nicht auto-geändert wenn PROJ-8 ein Feld auf hidden setzt — Spalte bleibt leer beim Export, Editor zeigt Warnung
- **Plugin-Lifecycle:** Configs bleiben in der DB, ausgegraut im UI mit Hinweis, ausgeblendet im Bulk-Action-Dialog — Audit-Trail bleibt erhalten

### Datenmanagement
- **Datei-Storage in DB-BLOB** (`data_export_result`-Tabelle) mit 24h-TTL — Multi-Replica-safe, kein externer Storage nötig
- **Audit-Trail langlebig:** Job-Zeile (mit Config-Snapshot, Application-IDs-Snapshot, Result-Summary) wird nie gelöscht, nur Datei-BLOB nach 24h
- **Cross-Tenant-Schutz:** `checkTenantAccess` auf allen Endpoints
- **DSGVO-Mitglied-Widerruf:** wirkt nur auf zukünftige Exports — bereits heruntergeladene Dateien sind in Admin-Verantwortung

### Filter
- **Kein eigener Filter im Framework** — Selektion über Antragsliste (PROJ-3-Filter + PROJ-25-Bulk-Action) oder Single aus Detail

## Open Questions (für `/architecture` oder spätere Klärung)

Diese sind Detail-/Operations-Fragen, kein Architektur-Risiko mehr:

- **Header-Sprache bei Standard-Vorlagen**: DE only oder mehrsprachig? Empfehlung: DE only für V1 (Owner-Kontext), Mehrsprachigkeit nur wenn echte Anforderung kommt
- **Performance XLSX vs. CSV** bei 500+ Zeilen — Benchmarks im `/architecture` machen, ggf. Hinweis im UI „XLSX ist langsamer bei >500 Zeilen"
- **Audit-Log-Retention-Policy**: indefinite oder Limit (z. B. 5 Jahre analog § 132 BAO)? Empfehlung: indefinite im V1, Cleanup-Job kann später nachgezogen werden
- **Standard-Vorlagen-Label-Sprache** bei englisch-konfigurierten EEGs — Out-of-Scope V1
- **1.000-Anträge-Limit pro Bulk-Aktion** performance-getestet — im `/architecture` Benchmarks machen
- **Multi-Plugin-Cascade** („erst Excel, dann Zoho" als verkettete Aktion) — V2-Idee, Out-of-Scope V1
- **Cron für regelmäßige Plugin-Runs** (z. B. „jede Woche an Mailchimp pushen") — Plugin-Architektur ist kompatibel, aber UI/Trigger separat zu designen

---

# Phase 2 Ausblick: weitere Plugins

(Separate spätere PROJs, jeweils nur Plugin-Implementierung + UI-Komponenten — kein Framework-Eingriff mehr.)

## Zoho CRM-Plugin (vermutlich erstes Push-Plugin)

- **Auth:** OAuth2 mit Refresh-Tokens, pro EEG eine Konfiguration mit verschlüsselten Credentials
- **Konfigurations-UI:** OAuth-Setup + Field-Mapping (Source-Onboarding-Feld → Target-Zoho-Feld)
- **Process-Implementierung:** Push pro Antrag, Idempotenz via Custom-Field `EEG_Onboarding_ID`, Retry-Logik mit Backoff
- **Result-Typ:** `SyncResult` mit Status pro Antrag (created / updated / failed)
- **Webhook-Handler:** falls Zoho bidirektional sync senden soll (out-of-scope V1), neuer HTTP-Endpoint `/api/webhooks/data-export/zoho` mit HMAC-Validation

## Weitere mögliche Plugins (Reihenfolge je nach Bedarf)

- **HubSpot CRM** (gleiches Schema wie Zoho)
- **Salesforce** (komplexer wegen Custom-Objects)
- **Pipedrive**
- **Mailchimp** (Listen-Mitglieder hinzufügen)
- **Webhook-Generic** (HTTP-POST an beliebige URL mit JSON-Payload — für EEGs mit Eigenbau-Systemen)

Jeder Plugin benötigt:
- Backend-Implementierung des Plugin-Interfaces (~1–2 Tage)
- Frontend-Konfigurations-UI (~1–2 Tage)
- Tests + Mock-Adapter (~½ Tag)

Phasen-Reihenfolge wird über Markt-Nachfrage entschieden.

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)

### Übersicht

PROJ-60 baut ein **generisches Async-Plugin-Framework** für Datenweiterleitungen mit dem Excel/CSV-Plugin als erster Implementierung. Die Architektur ist von Anfang an darauf ausgelegt, weitere Plugins (Zoho-CRM, HubSpot, etc. in Phase 2+) ohne Framework-Refactor hinzufügen zu können.

**Drei Kern-Bestandteile:**
1. **Plugin-Framework** — Registry, Interface, Job-Queue, Worker, Cleanup-Cron
2. **Excel-Plugin** — erste konkrete Implementierung mit Spalten-Mapping, Live-Preview, XLSX/CSV-Generierung
3. **Admin-UI** — Konfigurations-Verwaltung, Bulk-/Single-Trigger, Job-Polling, BackOffice-Übersicht

Drei neue DB-Tabellen, ein neuer K8s-CronJob, neue Routen im Frontend. Keine Änderung an bestehenden Datenstrukturen.

### Komponenten-Struktur (Überblick)

```
Admin-UI (EEG-Settings)
+-- Sektion „Datenweiterleitung" (neu)
|   +-- Konfigurations-Liste (gruppiert nach Plugin-Typ)
|   |   +-- „Excel-Exports"
|   |   |   +-- Konfig „Newsletter"     [bearbeiten] [löschen]
|   |   |   +-- Konfig „CRM-Stammdaten" [bearbeiten] [löschen]
|   |   |   +-- + Neue Konfiguration anlegen
|   |   +-- (später) „Zoho CRM"
|   +-- Konfigurations-Editor (Plugin-spezifisch)
|   |   +-- Excel-Editor:
|   |   |   +-- Name, Format (XLSX/CSV)
|   |   |   +-- Spalten-Liste mit Sortierung
|   |   |   |   +-- Spalte: Header, Feld-Dropdown, Format-Dropdown
|   |   |   |   +-- + Hinzufügen / Up/Down/Entfernen pro Spalte
|   |   |   +-- Live-Preview (letzte 5 importierte Mitglieder)
|   |   +-- (später) Zoho-Editor: OAuth-Setup, Field-Mapping
|   +-- Standard-Vorlagen-Bereich
|   |   +-- „Newsletter-Adressliste" → [Aus Vorlage erstellen]
|   |   +-- „CRM-Stammdaten"          → [Aus Vorlage erstellen]
|   |   +-- „Buchhaltungs-Export"     → [Aus Vorlage erstellen]
|   +-- Sub-Tab „Jobs"
|       +-- Failed-Jobs-Badge (rot, prominent)
|       +-- Liste der letzten 100 Jobs (Status, Zeit, Konfig, Anzahl)
|       +-- Pro Job: [Erneut ausführen] / [Datei herunterladen falls verfügbar]

Antragsliste (admin-application-table.tsx)
+-- Bestehende Bulk-Action-Leiste (PROJ-25)
    +-- „Genehmigen" / „Ablehnen" / „Zur Prüfung"
    +-- NEU: „Datenweiterleitung" → Dialog mit einstufiger Liste

Antrags-Detail (admin-application-detail.tsx)
+-- Bestehende Aktionen
    +-- NEU: „Datenweiterleitung" im Menü → gleicher Dialog

Job-Status-Toast (overlay, global)
+-- Polling alle 2-5 Sek
+-- Status: queued → running (Progress) → done | failed
+-- Bei done + DownloadResult: [Datei herunterladen]
+-- Bei done + SyncResult: Status-Report
+-- Bei failed: Fehler + [Retry]
```

### Datenmodell (in Klartext)

**Drei neue Tabellen** im `member_onboarding`-Schema:

**1. `data_export_config` — Plugin-Konfigurationen pro EEG**

Jede Konfiguration ist eine Instanz eines Plugins mit einem Namen, z. B. „Excel-Plugin als ‚Newsletter‘". Hat:

- Eindeutige ID + Referenz zur EEG (über `rc_number`)
- Plugin-Typ (z. B. `"excel"`, später `"zoho_crm"`)
- Name (eindeutig pro EEG, cross-plugin-type)
- Plugin-spezifische Konfiguration als flexibles JSON-Feld (für Excel: Spalten-Liste + Format)
- Flag „obsolet" für den Fall, dass der Plugin-Typ später aus dem Backend entfernt wird (Audit-Trail bleibt)
- Standard-Timestamps

**2. `data_export_job` — Job-Queue und -State, langlebig**

Jede ausgelöste Bulk- oder Single-Aktion erzeugt einen Job-Datensatz. Hält:

- Job-ID + EEG-Referenz
- Referenz auf die ursprüngliche Konfiguration (NULL wenn inzwischen gelöscht — Audit bleibt trotzdem lesbar)
- **Snapshot der Konfiguration** zum Trigger-Zeitpunkt (JSON) — laufende Jobs sind immun gegen Config-Edits
- Plugin-Typ als Snapshot (damit Plugin-Removal Audit nicht zerstört)
- Liste der selektierten Antrags-IDs (Snapshot)
- Status: `queued` / `running` / `done` / `failed` / `expired`
- Ausführender Admin
- Zeitstempel (created, started, finished)
- Result-Summary (JSON, z. B. `{"downloaded": 47}` oder `{"synced": 45, "failed": 2}`)
- Fehler-Meldung bei Failure
- Retry-Counter

Bleibt **unbegrenzt** erhalten für Audit. Nur Datei-BLOBs werden nach TTL gelöscht.

**3. `data_export_result` — Datei-Storage mit TTL**

Pro Job-Result eine BLOB-Zeile:

- Job-ID (Primary + Foreign Key)
- Dateiname, MIME-Type, Bytes, Größe
- Download-Zeitpunkt (optional, für spätere Stats)
- `expires_at` = `created_at + 24h` — Cleanup-Cron räumt auf

Nur für `DownloadResult`-Plugins (Excel). `SyncResult`-Plugins (Phase 2) haben keine Datei, brauchen diese Tabelle nicht.

### Async-Workflow (typischer Excel-Export-Lauf)

```
[Admin]                    [Backend HTTP]        [DB]          [Worker-Goroutine]
   |                           |                   |                  |
   | selektiert 200 Anträge    |                   |                  |
   | → Bulk-Action "Daten-     |                   |                  |
   |   weiterleitung"          |                   |                  |
   | → wählt "Excel: News-     |                   |                  |
   |   letter"                 |                   |                  |
   | → bestätigt               |                   |                  |
   |--------------------------→|                   |                  |
   |                           | POST /jobs        |                  |
   |                           |------------------→|                  |
   |                           | (Config + IDs als |                  |
   |                           |  Snapshot in Job- |                  |
   |                           |  Zeile schreiben) |                  |
   |                           |←------------------|                  |
   |←--------------------------| 202 Accepted +    |                  |
   | Toast erscheint:          |   job_id          |                  |
   | "Job queued"              |                   |                  |
   |                           |                   |←-- pollt alle    |
   |                           |                   |   5 Sek          |
   |                           |                   |    (SELECT ...   |
   |                           |                   |    FOR UPDATE    |
   |                           |                   |    SKIP LOCKED)  |
   |                           |                   |←-- nimmt Job an  |
   |                           |                   |   status=running |
   |                           |                   |                  |
   |--- pollt alle 2 Sek -----→|                   |                  |
   |                           | GET /jobs/{id}    |                  |
   |                           |------------------→|                  |
   |                           |←------------------|                  |
   |←-- "running, 50 of 200"---|                   |   verarbeitet:   |
   |                           |                   |   - Snapshot     |
   |                           |                   |     auspacken    |
   |                           |                   |   - Mitglieder   |
   |                           |                   |     laden        |
   |                           |                   |   - excelize-    |
   |                           |                   |     Renderer     |
   |                           |                   |     pro Zeile    |
   |                           |                   |   - BLOB in      |
   |                           |                   |     data_export_ |
   |                           |                   |     result       |
   |                           |                   |   - status=done  |
   |--- weiter Polling -------→|                   |                  |
   |                           | GET /jobs/{id}    |                  |
   |                           |------------------→|                  |
   |                           |←-- status=done    |                  |
   |←-- "Fertig! Download..."--|   + result_url    |                  |
   |                           |                   |                  |
   | klickt Download           |                   |                  |
   |--------------------------→|                   |                  |
   |                           | GET /jobs/{id}/   |                  |
   |                           |     download      |                  |
   |                           |------------------→|                  |
   |                           |←-- BLOB-Stream    |                  |
   |←-- Datei lädt herunter----|                   |                  |
```

### Backend-Komponenten

| Datei / Package | Inhalt |
|---|---|
| `db/migrations/00006X_data_export_*.up.sql` (3 Migrationen) | DB-Schema: config, job, result |
| `internal/shared/models.go` | Structs für Config, Job, Result, Plugin-Result-Typen (DownloadResult, SyncResult) |
| `internal/shared/requests.go` | Request/Response-Typen für API |
| `internal/dataexport/plugins/registry.go` | Plugin-Registry (Map, init über Imports) |
| `internal/dataexport/plugins/plugin.go` | Plugin-Interface |
| `internal/dataexport/plugins/excel/` | Excel-Plugin-Implementierung |
| `internal/dataexport/plugins/excel/plugin.go` | Plugin-Type, Validate, Process, StandardConfigs |
| `internal/dataexport/plugins/excel/renderer.go` | Spalten-Mapping + excelize-Wrapper (Wiederverwendung aus PROJ-17) |
| `internal/dataexport/service/config_service.go` | CRUD für Configs |
| `internal/dataexport/service/job_service.go` | Job-Erzeugung, Status-Abfrage, Retry-Logik |
| `internal/dataexport/worker/worker.go` | Goroutine-Pool, Job-Pickup mit FOR UPDATE SKIP LOCKED |
| `internal/dataexport/cleanup/cleanup.go` | Cleanup-Logik für Zombie-Jobs + abgelaufene BLOBs |
| `cmd/server/main.go` | Backend-Subcommand-Wiring: `data-export-cleanup` |
| `internal/http/admin.go` | Neue HTTP-Endpoints (Configs CRUD, Job-Trigger, Polling, Download, Retry) |

### Frontend-Komponenten

| Komponente | Inhalt |
|---|---|
| `src/components/data-export/configs-list.tsx` | Liste aller Konfigurationen pro EEG, gruppiert nach Plugin-Typ |
| `src/components/data-export/jobs-list.tsx` | BackOffice-Übersicht der letzten Jobs mit Filter + Failed-Badge |
| `src/components/data-export/job-status-toast.tsx` | Globaler Polling-Toast mit Live-Status + Download-Link |
| `src/components/data-export/bulk-action-dialog.tsx` | Einstufige Plugin-Konfig-Liste, wird aus Antragsliste + Antrag-Detail aufgerufen |
| `src/components/data-export/plugins/excel/editor.tsx` | Excel-spezifischer Editor: Spalten-Mapping, Live-Preview |
| `src/components/data-export/plugins/excel/preview-table.tsx` | Read-only Tabelle der letzten 5 Mitglieder mit aktuellem Mapping |
| `src/components/data-export/plugins/excel/standard-templates.tsx` | „Aus Vorlage erstellen"-Auswahl der 3 Standard-Templates |
| `src/components/admin-application-table.tsx` | Erweiterung: neue Bulk-Action „Datenweiterleitung" |
| `src/components/admin-application-detail.tsx` | Erweiterung: neue Single-Action „Datenweiterleitung" |
| `src/components/admin-eeg-settings-editor.tsx` | Erweiterung: neue Tab/Sektion „Datenweiterleitung" |
| `src/lib/api.ts` | Neue API-Funktionen für die Daten-Export-Endpoints |
| `src/lib/types.ts` | TypeScript-Typen für Config, Job, Result, PluginType |

### K8s-/Helm-Komponenten

| Datei | Inhalt |
|---|---|
| `helm/member-onboarding/templates/data-export-cleanup-cronjob.yaml` | Neuer K8s-CronJob (Schedule alle 10 Min) mit Subcommand `data-export-cleanup`, analog `restart-cronjob.yaml` |
| `helm/member-onboarding/values.yaml` | `dataExport.cleanup.schedule` + Defaults |

### Plugin-Pattern (Klassen-Diagramm)

```
┌────────────────────────────────────────────────────────────────┐
│                    Plugin-Registry                              │
│  Map[string → Plugin]   (initialisiert via Go-Imports)         │
│  - Register(p Plugin)                                           │
│  - Get(pluginType string) → Plugin                              │
│  - List() → []Plugin                                            │
└────────────────────────────────────────────────────────────────┘
                          ▲
                          │ implements
                          │
┌────────────────────────────────────────────────────────────────┐
│                    Plugin (Interface)                          │
│  - Type() string                                                │
│  - DisplayName() string                                         │
│  - ValidateConfig(config JSON) error                            │
│  - Process(ctx, configSnapshot, apps) → Result                  │
│  - StandardConfigs() []ConfigTemplate                           │
└────────────────────────────────────────────────────────────────┘
                          ▲
                ┌─────────┴─────────────┬──────────────┐
                │                       │              │
        ┌───────┴────────┐    ┌─────────┴────────┐   ...
        │  ExcelPlugin   │    │   ZohoPlugin     │
        │  (Phase 1)     │    │   (Phase 2)      │
        └────────────────┘    └──────────────────┘

Result (Interface)
├── DownloadResult    {Bytes, FileName, MimeType}
└── SyncResult        {PerAppStatus map[uuid] → string}
```

### Reihenfolge der Implementierung

1. **DB-Migrationen** — die drei Tabellen (config, job, result) anlegen
2. **Backend-Models + Requests** — Structs für API-Vertrag
3. **Plugin-Framework Foundation** — Registry, Interface, Result-Typen
4. **Excel-Plugin** — Renderer, ValidateConfig, Process, StandardConfigs
5. **Job-Service** — Erzeugung, Status-Abfrage, Concurrency-Check
6. **Worker** — Goroutine-Pool mit FOR UPDATE SKIP LOCKED + Pickup-Logik
7. **Cleanup** — Logik für Zombies + TTL-BLOBs, K8s-CronJob + Backend-Subcommand
8. **HTTP-Endpoints** — Routes + Handler in admin.go
9. **Frontend-Typen + API-Wrapper**
10. **Frontend-Komponenten** — Configs-List, Editor, Bulk-Dialog, Job-Toast, BackOffice-View
11. **Bulk-Action-Integration** in admin-application-table.tsx
12. **Single-Action-Integration** in admin-application-detail.tsx
13. **Tests** — Unit für Plugin-Validate/Process, Integration für Job-Worker + Cleanup, E2E für Bulk-Trigger + Download
14. **Doku** — `docs/api-spec.md` + `docs/domain-model.md` + Swagger-Regeneration

Backend-Schritte 1–8 sind sequenziell. Frontend-Schritte 9–12 parallelisierbar mit Backend-Tests/Doku.

### Tech-Entscheidungen (begründet)

| Entscheidung | Begründung |
|---|---|
| **Plugin-Framework von Anfang an** statt Excel-only-Refactor später | Gleicher Gesamt-Aufwand (~8–11 Tage). Sauberer + niedrigere Hürde für Phase 2 (Zoho-Plugin ~3–4 Tage). |
| **Async-Framework auch für Excel** | Push-Plugins (Phase 2) brauchen async (Rate-Limit-Throttling = 5–10 Min Verarbeitung). Sync-V1 + Async-V2-Refactor würde alle Patterns brechen. Excel-Overhead von ~1 Sek akzeptabel. |
| **Build-Time-Plugin-Registry** statt Runtime-Discovery | Pattern wie `sql.Driver` etabliert in Go. Keine Plugin-Loader-Risiken. Passt zu Single-Operator-Setup. |
| **In-App-Goroutine-Pool** statt separater Worker-Pod | Skaliert mit Backend-Replicas. Kein zweiter Deployment-Pfad. `FOR UPDATE SKIP LOCKED` macht Multi-Replica-safe. Resource-Limits per Pod begrenzen Risiko. |
| **K8s-CronJob-Cleanup** (Zombies + BLOB-TTL) | Pattern `restart-cronjob.yaml` existiert bereits. Einfacher als Heartbeat-Pattern, ausreichend robust. |
| **Config-Snapshot at Job-Creation** | Laufende Jobs sind immun gegen Config-Edits. Audit-Trail bleibt vollständig auch nach Config-Löschung. |
| **DB-BLOB für Datei-Storage** statt PVC oder S3 | Multi-Replica-safe ohne externe Infrastruktur. Excel-Dateien <100 KB → Storage trivial. Bei späterem Wachstum (Mio-Dateien): Migration zu S3 möglich, aber unwahrscheinlich. |
| **UI-Polling alle 2–5 Sek** statt WebSocket / Server-Sent-Events | Stateless HTTP, einfach im Backend (kein Connection-State), einfach im Frontend. Latency völlig akzeptabel bei Job-Dauern <2 Min. |
| **Einstufige Plugin-Config-Liste im Dialog** | Schnellste UX. „Excel: Newsletter" ist eindeutiger Bezug, kein Plugin-zuerst-dann-Config-Klick nötig. |
| **Pro Plugin eigene Frontend-Komponenten** (statt JSON-Schema-Dynamic-Forms) | Excel-Spalten-Editor + Live-Preview hat komplexe UX, die JSON-Schema nicht gut darstellt. Bei jedem neuen Plugin lieber spezifische React-Komponenten als generische Form-Generierung. |
| **Max 3 parallele Jobs pro EEG** | Anti-Misuse ohne Workflow-Block. FIFO-Queue darüber. |
| **Keine Idempotenz-Deduplikation** | Admin kann bewusst Re-Trigger machen. Push-Plugins müssen Plugin-intern Idempotenz handhaben (z. B. Custom-Field-Lookup im Ziel-System). |
| **Cleanup-Cron alle 10 Min** | Balance zwischen Recovery-Latency und Cluster-Last. Zombie-Threshold 1h reicht für typische Jobs. |

### Externe Abhängigkeiten

**Keine neuen npm- oder Go-Pakete erforderlich:**
- **`excelize`** — bereits aus PROJ-17 (Excel-Export für eegFaktura)
- **`pq` / `database/sql`** — bereits etabliert für PostgreSQL
- **`shadcn/ui` Komponenten** — alle nötigen (Dialog, Select, Input, Button, Table, Toast, Badge) bereits installiert
- **Mail-Service** — Postal-Integration aus PROJ-6 wiederverwendet für Failure-Notifications

### Test-Strategie

| Test-Schicht | Cases |
|---|---|
| **Unit (Plugin)** | Excel-Plugin: ValidateConfig akzeptiert valide / lehnt invalide Configs ab; Process generiert XLSX/CSV mit korrekten Spalten + Werten; Format-Transformationen für alle Typen; StandardConfigs liefert 3 Vorlagen |
| **Unit (Service)** | Config-CRUD: Tenant-Isolation, Name-Eindeutigkeit, Max-20-Limit; Job-Service: Concurrency-Limit-Check (3 pro EEG), Snapshot wird korrekt erzeugt |
| **Integration (Worker)** | FOR UPDATE SKIP LOCKED in Multi-Worker-Szenario keine Doppel-Pickups; Worker setzt Status korrekt von queued→running→done/failed; Result wird als BLOB persistiert |
| **Integration (Cleanup)** | Zombie-Jobs nach 1h auf failed mit Retry-Counter; BLOBs nach 24h hard-deleted; Job-Zeile bleibt mit status=expired |
| **Integration (API)** | Bulk-Trigger mit 200 Anträgen erzeugt Job mit Snapshot; Polling-Endpoint liefert aktuellen Status; Download-Endpoint streamt BLOB; Tenant-Hack-Versuch via config_id → 403; > 1000 Anträge → 400 |
| **E2E (Playwright)** | EEG-Admin erstellt Config aus Standard-Vorlage „Newsletter"; Live-Preview zeigt Mitglieder; Bulk-Action aus Antragsliste, Toast zeigt Done, Datei lädt herunter; Spalten-Mapping ändern + neu exportieren; Job-Failure simuliert (z. B. Plugin-Validate-Fehler), BackOffice-Badge erscheint, Mail wird versendet (Mail-Mock prüft) |

### Risiken & Mitigationen

| Risiko | Wahrscheinlichkeit | Mitigation |
|---|---|---|
| Worker-Goroutine blockiert HTTP-Performance bei großen Jobs | Mittel | Resource-Limits per Pod + max-concurrent-jobs-Cap (3 parallel) + große Jobs sind read-only DB-Operationen, kein Lock-Stress |
| Zombie-Jobs nach Pod-Crash bleiben „running" forever | Niedrig | Cleanup-Cron alle 10 Min mit 1h-Threshold setzt sie auf failed mit retry_count++ |
| Multi-Replica: zwei Worker greifen denselben Job | Sehr niedrig | `FOR UPDATE SKIP LOCKED` ist atomare Postgres-Operation, garantiert Single-Pickup |
| User schließt Browser-Tab während Job läuft | Garantiert | Job läuft im Backend weiter, Result wird im BLOB gespeichert (24h-TTL), Failure-Mail informiert Admin auch wenn UI nicht aktiv |
| Datei-BLOB-Storage wächst unkontrolliert | Niedrig | 24h-TTL + Cleanup-Cron. Bei 100 Jobs/Woche × 100 KB = 10 MB/Woche initial — vernachlässigbar. Monitoring via DB-Stats |
| Plugin-Removal bricht Audit-Trail | Niedrig | Plugin-Typ wird im Job-Snapshot persistiert. UI markiert Configs als obsolet, schließt sie aus aktiven Dialogen aus, lässt aber Read-Access. |
| Config-Schema-Migration bei Plugin-Update (V2: Excel-Plugin bekommt neue Felder) | Niedrig | ValidateConfig pro Plugin-Version. Bei Breaking Changes: Migration-Script konvertiert alte Config-JSONB-Werte in neue Form, ähnliches Pattern wie field_config-Migrationen |
| Excel-Plugin mit >1000 Spalten (Misuse) | Sehr niedrig | Backend-Limit max 50 Spalten pro Config |
| Cleanup-Cron läuft länger als 10 Min (Job-Backlog) | Sehr niedrig | concurrencyPolicy: Forbid im CronJob verhindert Überschneidungen; bei wirklich extremer Last: Cleanup-Frequenz reduzieren oder Batch-Größe limitieren |
| Failure-Mail an Admin geht verloren (SMTP down) | Niedrig | BackOffice-Badge ist primärer Notification-Kanal. Mail ist Bonus. Admin sieht Failed-Jobs auf jeden Fall beim nächsten BackOffice-Besuch |

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
