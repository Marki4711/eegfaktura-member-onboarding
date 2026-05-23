# PROJ-60: Datenweiterleitung an externe Systeme — Plugin-Framework

## Status: Planned
**Created:** 2026-05-23
**Last Updated:** 2026-05-23 (`/requirements` abgeschlossen, Pivot auf Plugin-Framework)
**Quelle:** Owner-Anforderung

## Dependencies
- Requires: PROJ-4 (Core Import) — Daten-Weiterleitung sinnvoll nach `imported`/`activated`, aber technisch für jeden Status möglich
- Requires: PROJ-17 (Excel-Export für eegFaktura-Import) — `excelize`-Library wird vom Excel-Plugin wiederverwendet
- Requires: PROJ-25 (Bulk-Aktionen) — Datenweiterleitung ist neue Bulk-Aktion in der Antragsliste
- Berührt: PROJ-46 (Post-Import-Stati) — exportierbar/sync-bar für jeden Status
- Berührt: PROJ-8 (konfigurierbare Felder pro EEG) — nur konfigurierte Felder werden als Mapping-Optionen angeboten

## Hintergrund

EEG-Vereine pflegen ihre Mitglieder teils nicht nur im
eegFaktura-Core, sondern parallel in einem eigenen System — z. B.
einem CRM (Zoho, HubSpot, Salesforce), Newsletter-Tool, Vereins-
Datenbank oder Excel-Datei für interne Auswertungen.

Aktuell muss der EEG-Admin diese parallele Pflege manuell machen:
Antrag öffnen → Stammdaten herauskopieren → ins externe System
einfügen. Das ist fehleranfällig, ineffizient und nicht skalierbar.

## Architektur-Vision: Plugin-Framework

Statt einer fest verdrahteten „Excel-Export"-Funktion bauen wir ein
**generisches Datenweiterleitungs-Framework** mit Plugin-Architektur:

- **Framework** definiert das Interface, die Trigger-Mechanik (Bulk
  + Single), die Auswahl-UI und das Audit-Log
- **Plugins** implementieren je einen konkreten Ziel-Kanal:
  - **Phase 1 (dieses PROJ-60):** Excel/CSV-Export-Plugin
  - **Phase 2 (späteres PROJ):** Zoho-CRM-Plugin
  - **Phase 3+:** HubSpot, Salesforce, Pipedrive, Mailchimp, …

Vorteil: Phase 2 wird vom kompletten PROJ auf „Plugin schreiben + UI-
Komponenten dazu" reduziert (~3–4 Tage statt ~5–7), weil Framework,
DB-Struktur, Audit-Log, Trigger-UI und Tenant-Isolation schon stehen.

**Aufwand-Vergleich:**
- Excel-only V1 ohne Framework: ~3–4 Tage. Phase 2 später ~5–7 Tage Refactor + Zoho-Code = **~8–11 Tage total**
- Framework + Excel als erster Plugin V1: ~5–7 Tage. Phase 2 später nur ~3–4 Tage = **~8–11 Tage total**

→ Gleicher Gesamt-Aufwand, aber sauberere Architektur und niedrigere
Hürde für jeden weiteren Plugin.

## User Stories

- Als **EEG-Admin** möchte ich pro EEG mehrere Datenweiterleitungs-Konfigurationen anlegen (z. B. „Excel: Newsletter", „Excel: CRM-Stammdaten", später „Zoho: Hauptverbindung"), sodass ich verschiedene Zielsysteme aus derselben Datenbasis bedienen kann.
- Als **EEG-Admin** möchte ich für jeden Plugin-Typ die spezifischen Konfigurations-Optionen pflegen können (für Excel: Spalten-Mapping; später für Zoho: OAuth-Verbindung + Field-Mapping), sodass jedes Zielsystem optimal angesprochen wird.
- Als **EEG-Admin** möchte ich in der Antragsliste mehrere Anträge selektieren und „Datenweiterleitung" als Bulk-Aktion auswählen, dann in einer einstufigen Liste meine konfigurierte Aktion wählen (z. B. „Excel: Newsletter"), sodass der Workflow konsistent zu anderen Bulk-Aktionen ist.
- Als **EEG-Admin** möchte ich in der Antrags-Detail-Ansicht einen Single-Datenweiterleitungs-Button mit derselben Liste haben, sodass ich einzelne Mitglieder ad-hoc weiterleiten kann.
- Als **EEG-Admin** möchte ich vorgefertigte Standard-Konfigurationen je Plugin-Typ nutzen können (z. B. drei Excel-Standard-Templates), die ich klonen und anpassen kann, sodass ich nicht von Grund auf alles selbst bauen muss.
- Als **vfeeg-Betreiber** möchte ich, dass das Framework Plugin-Erweiterungen ermöglicht, ohne das Kern-Onboarding-System anfassen zu müssen.

## Plugin-Interface (Konzept, kein Code)

Jeder Plugin implementiert eine einheitliche Schnittstelle:

| Methode | Zweck |
|---|---|
| `Type()` | Plugin-Typ-Identifikator (z. B. `"excel"`, `"zoho_crm"`) |
| `DisplayName()` | Anzeige-Name für Admin-UI (z. B. „Excel-Export", „Zoho CRM") |
| `ValidateConfig(config)` | Prüft die Plugin-spezifische Konfiguration auf Plausibilität |
| `Process(config, applications) → Result` | Hauptlogik: erzeugt Datei (Download-Result) oder pusht Daten (Sync-Result) |
| `PreviewSchema()` | Liefert Beschreibung der Konfigurations-Felder (für Standard-Vorlagen-Discovery) |

**Result-Typen:**
- **DownloadResult**: enthält Datei-Stream + MIME-Type + Dateiname → Browser-Download
- **SyncResult**: enthält Status pro Antrag (success / failed / skipped) → Audit-Log + UI-Feedback

Das Framework dispatch-t je nach Result-Typ unterschiedlich (Datei-Stream
zum Browser vs. Status-Anzeige nach Push-Operation).

## Acceptance Criteria

### Framework-Komponenten

- [ ] DB-Tabelle `data_export_config` mit Spalten: `id`, `rc_number`, `plugin_type` (TEXT), `name` (UNIQUE pro `rc_number`), `config` (JSONB plugin-spezifisch), `created_at`, `updated_at`
- [ ] Backend-Registry für registrierte Plugins (Map `plugin_type` → Plugin-Instanz)
- [ ] Pro EEG bis zu **20 Konfigurationen total** über alle Plugin-Typen (Anti-Misuse-Limit, kann später erhöht werden)
- [ ] Eindeutige Namen pro EEG (Cross-Plugin-Type-Validation, damit „Newsletter" nicht doppelt verwendet wird)
- [ ] Audit-Log-Tabelle `data_export_log`: `id`, `rc_number`, `config_id` (nullable falls Konfiguration gelöscht), `plugin_type`, `applications_count`, `admin_user_id`, `executed_at`, `result_summary` (JSONB — Status-Counts) — **kein** Inhalts-Speicher der Daten
- [ ] Tenant-Isolation: `eeg_admin` sieht/nutzt nur Konfigurationen ihrer eigenen EEG

### Admin-UI: Konfigurations-Verwaltung

- [ ] Neue Sektion „Datenweiterleitung" unter EEG-Settings
- [ ] Liste aller konfigurierten Plugin-Konfigurationen, gruppiert nach Plugin-Typ (z. B. „Excel-Exports" mit allen Excel-Konfigs darunter)
- [ ] Pro Plugin-Typ ein „Neue Konfiguration anlegen"-Button
- [ ] Plugin-Auswahl (im V1 nur Excel sichtbar; bei Phase 2 erscheint Zoho als weitere Option ohne Frontend-Refactor des Frameworks)
- [ ] Plugin-spezifischer Editor für die jeweilige Konfiguration (für Excel: Spalten-Mapping-Editor mit Live-Preview)
- [ ] Konfiguration kann editiert oder gelöscht werden

### Trigger: Bulk-Datenweiterleitung aus Antragsliste

- [ ] In der Antragsliste (`admin-application-table.tsx`) ist „Datenweiterleitung" eine neue Bulk-Aktion (analog zu Genehmigen/Ablehnen aus PROJ-25)
- [ ] Bei ≥1 selektiertem Antrag erscheint in der Aktionsleiste „Datenweiterleitung" als zusätzliche Option
- [ ] Klick öffnet einen Dialog mit **einstufiger Liste** aller konfigurierten Plugin-Konfigurationen für diese EEG, z. B.:
  - „📊 Excel: CRM-Stammdaten"
  - „📊 Excel: Newsletter"
  - „📊 Excel: Buchhaltung"
  - *(später)* „☁️ Zoho: Hauptverbindung"
- [ ] Hinter jedem Eintrag steht ein Icon, das den Plugin-Typ anzeigt + Hinweis auf den Result-Typ („Datei-Download" / „Push an externes System")
- [ ] Bestätigung dispatch-t an den Plugin → Result wird verarbeitet (Download oder Push-Status-Anzeige)
- [ ] Anzahl der selektierten Anträge ist die Verarbeitungs-Menge — kein zusätzlicher Filter, weil die Antragsliste bereits Filter hat (existierende PROJ-3-Filter)
- [ ] Maximum 1.000 Anträge pro Bulk-Aktion (Plugin-übergreifend, kann pro Plugin enger gesetzt werden)

### Trigger: Single-Datenweiterleitung aus Antrags-Detail

- [ ] In `admin-application-detail.tsx` ein Menüeintrag „Datenweiterleitung" mit derselben einstufigen Liste
- [ ] Verfügbar für jeden Antragsstatus
- [ ] Push-Plugins (Phase 2) zeigen Status nach erfolgtem Sync; Excel-Plugin triggert direkten Download

### Audit-Log

- [ ] Jede Datenweiterleitungs-Ausführung schreibt einen `data_export_log`-Eintrag
- [ ] Eintrag enthält: Plugin-Typ, Konfigurations-ID (oder NULL falls inzwischen gelöscht), Anzahl Anträge, ausführender Admin, Zeitpunkt, Result-Summary (z. B. `{ "downloaded": 47 }` oder `{ "synced": 45, "failed": 2 }`)
- [ ] Kein Datenfeld-Inhalt im Log (DSGVO + Storage-Hygiene)
- [ ] BackOffice-View: Liste der letzten 100 Datenweiterleitungs-Aktionen pro EEG (für Audit und Debugging)

### DSGVO

- [ ] Beim Hinzufügen sensitiver Felder in einem Excel-Mapping (IBAN, Geburtsdatum) erscheint Warnhinweis: „Sie tragen die Verantwortung für die DSGVO-konforme Weiterverarbeitung im Zielsystem (Art. 32)."
- [ ] Bei Push-Plugins (Phase 2): zusätzlicher Hinweis im Setup-Flow: „Stellen Sie sicher, dass Sie einen separaten AVV mit dem Ziel-Anbieter ([Zoho/HubSpot/…]) abgeschlossen haben."
- [ ] Cross-Tenant-Schutz: `checkTenantAccess` für alle Endpoints
- [ ] Persistierung: **fly-by** im V1, nur Audit-Log-Metadata. Datei-Inhalt nicht gespeichert (Wiederholbarkeit durch erneute Generierung aus aktuellen Daten)

---

## Phase 1: Excel-Plugin (erster konkreter Adapter)

Die folgenden Acceptance Criteria gelten **spezifisch für den
Excel-Plugin**, der als erste konkrete Implementierung des Plugin-
Frameworks ausgeliefert wird.

### Excel-Plugin-Konfiguration (JSONB-Struktur)

```json
{
  "format": "xlsx" | "csv",
  "columns": [
    {
      "header": "Vorname",
      "field": "firstname",
      "format": "string"
    },
    {
      "header": "Zählpunkte",
      "field": "meter_numbers",
      "format": "comma_separated"
    }
  ]
}
```

### Standard-Konfigurationen für Excel-Plugin

- [ ] Drei vordefinierte read-only Vorlagen werden im UI angeboten:
  1. **„Newsletter-Adressliste"** — Vorname, Nachname (oder Firma), E-Mail, Anrede
  2. **„CRM-Stammdaten"** — Vorname, Nachname (oder Firma), E-Mail, Telefon, Adresse, Mitgliedsnummer, Beitrittsdatum
  3. **„Buchhaltungs-Export"** — Mitgliedsnummer, Vorname, Nachname (oder Firma), Rechnungsadresse, IBAN, UID-Nummer
- [ ] Admin klickt „Aus Vorlage erstellen" → neue editierbare Konfiguration wird mit Default-Spalten initialisiert

### Excel-Plugin-Spalten-Mapping-UI

- [ ] Dropdown mit allen verfügbaren Onboarding-Feldern, gruppiert nach Kategorie:
  - **Stammdaten**: member_type, titel, firstname, lastname, titel_nach, company_name, uid_number, register_number, birth_date
  - **Kontakt**: email, phone
  - **Adresse**: resident_street (+ _number), resident_zip, resident_city
  - **Bank**: iban, account_holder, einzugsart
  - **EEG**: rc_number, reference_number, member_number, membership_start_date, status, imported_at, activated_at
  - **Zählpunkte**: meter_count, meter_numbers (komma-getrennt)
  - **Konfigurierbar (PROJ-8)**: nur Felder, die für die EEG aktiv sind
- [ ] Pro Spalte editierbar: Header-Text (frei wählbar), Feld-Auswahl, Format
- [ ] Reihenfolge per Auf/Ab-Buttons änderbar (Drag-Drop als V2)
- [ ] Mindestens 1, maximal 50 Spalten

### Excel-Plugin-Wert-Transformationen

- **Text**: 1:1
- **Datum**: „DD.MM.YYYY" (Default), „YYYY-MM-DD" (ISO), „DD.MM.YYYY HH:MM"
- **Boolean**: „Ja/Nein" (Default), „true/false", „1/0", „Y/N"
- **Enum** (member_type, status, einzugsart): „Wert" (Roh) oder „Label" (lesbares Deutsch)
- **Zahl**: „DE-Format" (Komma-Dezimal), „ISO" (Punkt-Dezimal)
- **Multi-Value** (Zählpunkte): „comma_separated" — kommas­ementhalten alle Werte

### Excel-Plugin-Live-Preview im Editor

- [ ] Live-Preview zeigt die **letzten 5 importierten Mitglieder** dieser EEG mit aktueller Spalten-Konfiguration
- [ ] Aktualisiert sich bei jeder Spalten-Änderung
- [ ] Read-only, rein visualisierend
- [ ] Falls EEG keine importierten Mitglieder hat: Beispiel-Daten (anonymisiert) mit Hinweis

### Excel-Plugin-Datei-Generierung

- [ ] Dateiname: `{rc_number}-{config_name}-{YYYY-MM-DD}.{xlsx|csv}` — z. B. `RC456-Newsletter-2026-05-23.xlsx`
- [ ] XLSX via `excelize` (PROJ-17-Pattern, aber mit dynamischen Spalten)
- [ ] CSV: UTF-8 mit BOM, Semikolon als Separator (DACH-Standard), Werte mit Sonderzeichen in Anführungszeichen
- [ ] Streaming-Output für >100 Zeilen

### Excel-Plugin-Zählpunkte (Multi-Value)

- [ ] Spalte „Zählpunkte" mit Format `comma_separated` enthält komma-getrennte Liste der Zählpunkt-Nummern
- [ ] „Anzahl Zählpunkte" als separate numerische Spalte verfügbar
- [ ] Detail-Felder pro Zählpunkt (Richtung, Adresse) **nicht** in V1 exportierbar — V2-Option „Zeile pro Zählpunkt" möglich

---

## Edge Cases

- **Admin selektiert Anträge mit gemischtem Status** → Plugin verarbeitet alle. Felder, die nur bei bestimmten Status Werte haben (z. B. `member_number` ab `imported`), bleiben für andere leer.
- **Konfiguration referenziert ein Feld, das die EEG nachträglich auf „hidden" gesetzt hat (PROJ-8)** → Spalte bleibt leer. Editor zeigt Warnung beim Bearbeiten.
- **Konfiguration wird gelöscht, während sie gerade in einem Bulk-Action-Dialog ausgewählt ist** → Dialog refresht Liste, Hinweis „Konfiguration nicht mehr verfügbar".
- **Mitglied wurde zwischen Selektion und Verarbeitung gelöscht** → wird übersprungen, Audit-Log notiert.
- **Sonderzeichen in Mitgliedsdaten** → korrektes Escaping (CSV-Quotes, XLSX nativ).
- **Bulk-Aktion >1.000 Anträge** → Frontend-Begrenzung mit Hinweis, Backend-Defense-in-Depth.
- **Cross-Tenant-Hack** → 403 via `checkTenantAccess`.
- **Mitglied ohne Zählpunkte** → „Zählpunkte"-Spalte leer, „Anzahl Zählpunkte" = 0.
- **Datei-Download wird abgebrochen** → kein Server-Side-Cleanup nötig (Streaming).
- **Mehrere Admins exportieren parallel** → unkritisch (read-only).
- **Plugin-Konfiguration mit invaliden Werten** (z. B. Spalte ohne Feld) → `ValidateConfig` blockiert beim Speichern.
- **Konfiguration für Plugin-Typ, der aus dem System entfernt wurde** (zukünftig theoretisch möglich) → Konfiguration wird in der Liste ausgegraut mit Hinweis „Plugin nicht mehr verfügbar".

## Technical Requirements

- **DB-Migrationen**:
  - `data_export_config` (Konfigurationen pro EEG)
  - `data_export_log` (Audit-Trail der Ausführungen)
- **Backend-Plugin-Registry** als Map `plugin_type` → Plugin-Instanz, initialisiert beim Startup
- **Excel-Plugin-Implementierung** unter `internal/dataexport/plugins/excel/` als erste konkrete Implementierung
- **Generischer HTTP-Endpoint**: `POST /api/admin/data-export/run` mit `config_id` + `application_ids` → Plugin-Dispatch
- **Plugin-spezifische Konfigurations-Endpoints**: `GET/POST/PUT/DELETE /api/admin/data-export/configs/{plugin_type}/{id}` mit plugin-spezifischer Validation
- **Frontend**:
  - Generische Bulk-Action-Komponente in `admin-application-table.tsx`
  - Plugin-spezifische Editor-Komponenten in `src/components/data-export/plugins/{plugin_type}/`
  - Wiederverwendung von shadcn-Komponenten (Dialog, Select, Input, Table)
- **`excelize`-Wiederverwendung** aus PROJ-17, aber als generischer Renderer mit Spalten-Definitionen
- **Streaming-Output** via `http.ResponseWriter` Chunked-Transfer
- **Tenant-Isolation** auf allen Endpoints

## Resolved Decisions (aus `/requirements` 2026-05-23)

- **Architektur-Pivot:** Plugin-Framework statt fest verdrahteter Excel-Logik. Excel ist erster Plugin, Phase 2 (Zoho/CRM) ist „weiterer Plugin im gleichen Framework", kein PROJ-Refactor.
- **Dialog-Struktur:** einstufige Liste aller konfigurierten Plugin-Konfigurationen (z. B. „Excel: Newsletter", später „Zoho: Hauptverbindung"). Schnellster Workflow.
- **Frontend-Pluggability:** pro Plugin eigene UI-Komponenten, fest verdrahtet im Code. Keine JSON-Schema-basierte dynamische Form-Generierung (overengineered für V1).
- **Zählpunkt-Multi-Value:** komma-getrennt in einer Spalte (Excel-Plugin V1). „Zeile pro Zählpunkt" als V2-Option.
- **Standard-Konfigurationen:** 3 vordefinierte read-only Excel-Vorlagen (Newsletter, CRM-Stammdaten, Buchhaltung) als Klon-Basis.
- **Filter:** kein eigener Filter im Framework — Selektion über Antragsliste (PROJ-3-Filter + PROJ-25-Bulk-Action) oder Single aus Detail.
- **Persistierung:** fly-by, Audit-Log nur mit Metadata.

## Open Questions (für `/grill-me`)

- **Plugin-Registry-Erweiterung:** wie wird ein neuer Plugin im Backend registriert — Build-Zeit-Compile-In oder Runtime-Discovery? Empfehlung: Build-Time im V1 (einfacher), Runtime-Discovery erst wenn echte 3rd-Party-Plugin-Anforderung kommt.
- **Header-Sprache** bei Standard-Vorlagen: DE only oder mehrsprachig?
- **Performance** bei XLSX vs. CSV ab 500+ Zeilen — `excelize`-Overhead?
- **Audit-Log-Retention-Policy** — indefinite oder 2-Jahres-Limit?
- **Konfigurations-Versionierung** im Audit-Log: Snapshot der Config zum Ausführungszeitpunkt oder nur Reference auf aktuelle Config?
- **Standard-Vorlagen-Label-Sprache** bei englisch-konfigurierten EEGs (z. B. internationalen Genossenschaften)?
- **1.000-Anträge-Limit** pro Bulk-Aktion performance-getestet — oder pro Plugin-Typ unterschiedlich (Excel verträgt mehr als Push-Plugins mit Rate-Limits)?
- **Multi-Plugin-Cascade:** „erst Excel, dann Zoho" als verkettete Aktion — V2-Idee oder Out-of-Scope?

## Notes

- **PROJ-25 (Bulk-Aktionen)** ist das etablierte UI-Pattern — Datenweiterleitung wird konsistent integriert (Checkbox-Select + Bulk-Action-Bar + Bestätigungs-Dialog)
- **Plugin-Framework als Architektur-Investment**: gleicher Gesamt-Aufwand wie „Excel-only + späterer Refactor", aber strukturell sauberer und niedrigere Hürde für jeden weiteren Plugin
- **PROJ-17 (Excel-Export für Core-Import)** bleibt unverändert — anderer Use-Case (hardcoded Format für eegFaktura), wird nicht durch das neue Framework abgelöst

---

# Phase 2 Ausblick: weitere Plugins

(Separate spätere PROJs, jeweils nur Plugin-Implementierung + UI-Komponenten — kein Framework-Eingriff mehr.)

## Zoho CRM-Plugin (vermutlich erstes Push-Plugin)

- **Auth:** OAuth2 mit Refresh-Tokens, pro EEG eine Konfiguration mit verschlüsselten Credentials
- **Konfigurations-UI:** OAuth-Setup + Field-Mapping (Source-Onboarding-Feld → Target-Zoho-Feld)
- **Process-Implementierung:** Push pro Antrag, Idempotenz via Custom-Field `EEG_Onboarding_ID`, Retry-Logik mit Backoff
- **Result-Typ:** `SyncResult` mit Status pro Antrag (created / updated / failed)

## Weitere mögliche Plugins (Reihenfolge je nach Bedarf)

- **HubSpot CRM** (gleiches Schema wie Zoho, anderer API-Provider)
- **Salesforce** (ähnlich, komplexer wegen Custom-Objects)
- **Pipedrive**
- **Mailchimp** (Listen-Mitglieder hinzufügen für Newsletter-Versand)
- **Webhook-Generic** (HTTP-POST an beliebige URL mit JSON-Payload — für EEGs mit Eigenbau-Systemen)

Jeder dieser Plugins benötigt:
- Backend-Implementierung des Plugin-Interfaces (~1–2 Tage)
- Frontend-Konfigurations-UI (~1–2 Tage)
- Optional: Trigger-Tests + Mock-Adapter für Tests (~½ Tag)

Phasen-Reihenfolge wird über Markt-Nachfrage entschieden (welche EEGs welches CRM nutzen).

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
