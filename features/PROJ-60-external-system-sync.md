# PROJ-60: Datenweiterleitung an externe Systeme nach Import

## Status: Planned
**Created:** 2026-05-23
**Last Updated:** 2026-05-23 (`/requirements` abgeschlossen)
**Quelle:** Owner-Anforderung

## Dependencies
- Requires: PROJ-4 (Core Import) — Export sinnvollerweise nach `imported`/`activated`
- Requires: PROJ-17 (Excel-Export für eegFaktura-Import) — `excelize`-Library wird wiederverwendet
- Requires: PROJ-25 (Bulk-Aktionen) — Export ist neue Bulk-Aktion in der Antragsliste
- Berührt: PROJ-46 (Post-Import-Stati) — exportierbar für jeden Status
- Berührt: PROJ-8 (konfigurierbare Felder pro EEG) — nur konfigurierte Felder werden als Spalten-Optionen angeboten

## Hintergrund

EEG-Vereine pflegen ihre Mitglieder teils nicht nur im
eegFaktura-Core, sondern parallel in einem eigenen System — z. B.
einem CRM (Zoho, HubSpot, Salesforce), Newsletter-Tool, Vereins-
Datenbank oder Excel-Datei für interne Auswertungen.

Aktuell muss der EEG-Admin diese parallele Pflege manuell machen:
Antrag im Onboarding-System öffnen → Stammdaten herauskopieren →
ins externe System einfügen. Das ist fehleranfällig, ineffizient
und nicht skalierbar.

**Phase 1 (dieses PROJ-60):** Excel/CSV-Export mit pro EEG
konfigurierbarem Spalten-Mapping, getriggert aus der Antragsliste
(Bulk-Aktion) oder Antrags-Detail (Single-Action).

**Phase 2 (späteres PROJ):** Push-basierte CRM-API-Integration
(Zoho/HubSpot/…) auf demselben Plugin-/Adapter-Framework. Skizze
in Sektion „Phase 2 Ausblick" unten.

## User Stories

- Als **EEG-Admin** möchte ich pro EEG mehrere Export-Templates speichern können (z. B. „Newsletter", „CRM-Stammdaten", „Bank-Backup"), sodass ich verschiedene Zielsysteme aus derselben Datenbasis bedienen kann.
- Als **EEG-Admin** möchte ich beim Erstellen eines Templates frei auswählen, welche Onboarding-Felder als Spalten erscheinen, welche Header-Texte sie haben und in welcher Reihenfolge, sodass das Excel exakt zum Ziel-System passt.
- Als **EEG-Admin** möchte ich beim Editieren eines Templates eine Vorschau mit Beispiel-Daten sehen, sodass ich das Mapping ohne ständigen Test-Download verifizieren kann.
- Als **EEG-Admin** möchte ich in der Antragsliste mehrere Anträge per Checkbox selektieren und als Bulk-Aktion in ein vorab gewähltes Template exportieren, sodass ich nicht jeden Antrag einzeln verarbeiten muss.
- Als **EEG-Admin** möchte ich in der Antrags-Detail-Ansicht einen Single-Export-Button haben, sodass ich einzelne Mitglieder ad-hoc weiterleiten kann.
- Als **EEG-Admin** möchte ich vorgefertigte Standard-Templates nutzen können (Newsletter, CRM, Buchhaltung), die ich bei Bedarf klonen und anpassen kann, sodass ich nicht von Grund auf alles selbst bauen muss.
- Als **vfeeg-Betreiber** möchte ich, dass die Bedienung dem etablierten Bulk-Action-Pattern (PROJ-25) folgt, sodass Admins kein neues Konzept lernen müssen.

## Acceptance Criteria

### Template-Verwaltung

- [ ] EEG-Admin sieht im Admin-UI eine neue Sektion „Export-Templates" unter EEG-Settings
- [ ] Admin kann ein neues Template anlegen mit: Name (1–100 Zeichen, pflicht), Format (XLSX oder CSV), Spalten-Liste (geordnet)
- [ ] Admin kann ein bestehendes Template editieren oder löschen
- [ ] Templates sind pro EEG separat (Tenant-isoliert)
- [ ] Pro EEG bis zu **10 Templates** möglich (Limit als Anti-Misuse, kann später erhöht werden)
- [ ] Eindeutige Namen pro EEG (Duplikat-Validierung)

### Standard-Templates (vordefiniert)

- [ ] Beim ersten Aufruf der Export-Templates-Sektion werden für die EEG drei vordefinierte Templates als Klon-Vorlagen angeboten:
  1. **„Newsletter-Adressliste"** — Vorname, Nachname (oder Firma), E-Mail, Anrede
  2. **„CRM-Stammdaten"** — Vorname, Nachname (oder Firma), E-Mail, Telefon, Adresse, Mitgliedsnummer, Beitrittsdatum
  3. **„Buchhaltungs-Export"** — Mitgliedsnummer, Vorname, Nachname (oder Firma), Rechnungsadresse, IBAN, UID-Nummer
- [ ] Admin klickt „Aus Vorlage erstellen" → neues Template wird mit den Default-Spalten initialisiert, Admin kann es anpassen
- [ ] Vordefinierte Vorlagen sind read-only (können nur als Basis dienen, nicht direkt editiert werden)

### Spalten-Mapping-UI

- [ ] Dropdown mit allen verfügbaren Onboarding-Feldern, gruppiert nach Kategorie:
  - **Stammdaten**: member_type, titel, firstname, lastname, titel_nach, company_name, uid_number, register_number, birth_date
  - **Kontakt**: email, phone
  - **Adresse**: resident_street (+ _number), resident_zip, resident_city
  - **Bank**: iban, account_holder, einzugsart
  - **EEG**: rc_number, reference_number, member_number, membership_start_date, status, imported_at, activated_at
  - **Zählpunkte**: meter_count, meter_numbers (komma-getrennt)
  - **Konfigurierbar (PROJ-8)**: nur die Felder, die für diese EEG aktiv sind (heat_pump, electric_vehicle, …)
- [ ] Pro Spalte editierbar: Header-Text (frei wählbar), Feld-Auswahl (Dropdown), Format (Dropdown der Format-Optionen für den jeweiligen Feld-Typ)
- [ ] Reihenfolge per Auf/Ab-Buttons änderbar (Drag-Drop kann später ergänzt werden)
- [ ] Mindestens 1 Spalte pro Template, maximal 50 Spalten

### Wert-Transformationen (Format-Optionen pro Feld-Typ)

- [ ] **Text-Felder**: nur Format „Text" (1:1)
- [ ] **Datum-Felder**: „DD.MM.YYYY" (Default), „YYYY-MM-DD" (ISO), „DD.MM.YYYY HH:MM" (für Timestamps)
- [ ] **Boolean-Felder**: „Ja/Nein" (Default), „true/false", „1/0", „Y/N"
- [ ] **Enum-Felder** (member_type, status, einzugsart): „Wert" (Roh-String) oder „Label" (lesbares Deutsch — z. B. „Privatperson" statt „private")
- [ ] **Zahl-Felder**: „DE-Format" (Komma als Dezimal, Punkt als Tausender), „ISO" (Punkt als Dezimal)

### Vorschau im Template-Editor

- [ ] Live-Preview zeigt die **letzten 5 importierten Mitglieder** dieser EEG mit dem aktuellen Mapping
- [ ] Preview aktualisiert sich bei jeder Spalten-Änderung in der UI
- [ ] Preview-Zeilen sind read-only (rein zur Visualisierung)
- [ ] Wenn die EEG keine importierten Mitglieder hat: Preview zeigt Beispiel-Daten („Max Mustermann", anonymisiert) mit Hinweis „Beispiel-Daten — sobald Sie Mitglieder importiert haben, sehen Sie hier echte Vorschau"

### Trigger: Bulk-Export aus Antragsliste

- [ ] In der Antragsliste (`admin-application-table.tsx`) ist Excel-Export eine neue Bulk-Aktion (analog zu „Genehmigen"/„Ablehnen" aus PROJ-25)
- [ ] Bei ≥1 selektiertem Antrag erscheint in der Aktionsleiste „Excel-Export" als zusätzliche Option
- [ ] Klick öffnet einen Dialog: „Welches Template verwenden?" mit Dropdown der vom EEG gespeicherten Templates + dem ausgewählten Format
- [ ] Bestätigung triggert Generierung + Download (XLSX oder CSV gemäß Template)
- [ ] **Anzahl der selektierten Anträge ist die Export-Menge** — kein zusätzlicher Filter, weil die Antragsliste bereits gefiltert ist (existierende Filter aus PROJ-3)
- [ ] Maximum 1.000 Anträge pro Bulk-Export (Performance-Cap, darüber Hinweis „bitte filtern oder mehrere Exports")

### Trigger: Single-Export aus Antrags-Detail

- [ ] In `admin-application-detail.tsx` gibt es im Aktionen-Menü einen Eintrag „Excel-Export"
- [ ] Klick öffnet denselben Template-Auswahl-Dialog
- [ ] Bestätigung erzeugt Datei mit **einer einzelnen Zeile** (dem aktuellen Antrag)
- [ ] Verfügbar für **jeden** Antragsstatus (auch draft, submitted) — Felder ohne Wert (z. B. `member_number` vor Import) bleiben leer

### Dateigenerierung + Download

- [ ] Dateiname: `{rc_number}-{template_name}-{YYYY-MM-DD}.{xlsx|csv}` — z. B. `RC456-CRM-Stammdaten-2026-05-23.xlsx`
- [ ] XLSX-Generierung via `excelize` (wie PROJ-17, aber dynamische Spalten)
- [ ] CSV-Generierung: UTF-8 mit BOM, Semikolon als Separator, Werte mit Sonderzeichen in Anführungszeichen
- [ ] Streaming-Download (kein In-Memory-Buffer-Build) für Exports >100 Zeilen
- [ ] Audit-Log-Eintrag in `status_log` oder neuer `export_log`-Tabelle: Admin-User, Template-ID, Antrags-Count, Zeitpunkt — **nicht** die Inhalte, nur Metadata

### Zählpunkte (Multi-Value)

- [ ] Spalte „Zählpunkte" enthält **komma-getrennte Liste** der Zählpunkt-Nummern des Mitglieds (z. B. „AT001234..., AT001235...")
- [ ] Spalte „Anzahl Zählpunkte" als optionale numerische Spalte verfügbar
- [ ] Detail-Felder pro Zählpunkt (Richtung, Adresse) sind in V1 **nicht exportierbar** — für solche Detail-Auswertungen muss die EEG die Datenbank-View oder einen Sonder-Export nutzen (V2-Erweiterung möglich, z. B. „Zeile pro Zählpunkt"-Toggle pro Template)

### DSGVO

- [ ] Beim Hinzufügen sensitiver Felder (IBAN, Geburtsdatum) erscheint ein Hinweis-Dialog: „Sie tragen die Verantwortung für die DSGVO-konforme Weiterverarbeitung dieser Daten im Zielsystem (Art. 32 — Sicherheit der Verarbeitung). Sind Sie sicher?"
- [ ] Audit-Log erfasst jede Export-Aktion: wer, wann, welches Template, wie viele Zeilen (kein Datenfeld-Inhalt)
- [ ] Cross-Tenant-Schutz: EEG-Admin kann nur Daten ihrer eigenen EEG exportieren (via `checkTenantAccess` analog zu allen anderen Admin-Endpoints)
- [ ] Persistierung: **fly-by**, kein Datei-Speichern im V1. Audit-Log enthält nur Metadata, nicht die Datei selbst (Wiederholbarkeit über erneute Generierung aus den aktuellen Mitgliedsdaten)

## Edge Cases

- **EEG-Admin selektiert Anträge gemischter Status** → alle gewählten werden exportiert. Felder, die nur bei bestimmten Status Werte haben (z. B. `member_number` ist erst ab `imported` gesetzt), bleiben für andere Status leer.
- **Template referenziert ein Feld, das die EEG nachträglich auf „hidden" gesetzt hat (PROJ-8)** → Spalte bleibt im Export leer. Template-Editor zeigt beim Bearbeiten Warnung „Feld X ist in den EEG-Einstellungen nicht aktiv".
- **Template wird gelöscht, während es gerade in einem Export-Dialog ausgewählt ist** → Dialog refresh-t Liste, Hinweis „Template nicht mehr verfügbar". Konsistenz über Optimistic-Lock nicht nötig (Templates ändern sich selten).
- **Mitglied wurde zwischen Selektion und Export gelöscht** → wird im Export übersprungen, Audit-Log notiert „X exportiert, Y übersprungen wegen zwischenzeitlicher Löschung".
- **Sonderzeichen in Mitgliedsdaten** (Semikolon, Anführungszeichen, Zeilenumbruch) → korrektes CSV-Escaping (Doppel-Anführungszeichen + Wert in Quotes). XLSX nativ unproblematisch.
- **Bulk-Export von >1.000 Anträgen** → frontendseitige Begrenzung mit Hinweis „bitte filtern oder mehrere Exports". Server-seitig zusätzlich erzwungen (Defense-in-Depth).
- **Admin von EEG A versucht via Frontend-Hack Anträge von EEG B zu exportieren** → Server `checkTenantAccess` blockt mit 403 (Standard-Tenant-Isolation).
- **Mitglied mit 0 Zählpunkten** (z. B. neuer draft-Antrag) → Spalte „Zählpunkte" bleibt leer, „Anzahl Zählpunkte" zeigt 0.
- **Excel-Datei wird beim Download durch User abgebrochen** → keine Folge auf Server-Seite (Streaming-Operation), Audit-Log enthält trotzdem den Start-Zeitpunkt.
- **Mehrere Admins exportieren parallel dasselbe Template** → unkritisch (read-only Operation), keine Locks nötig.

## Technical Requirements

- **DB-Migration** für `export_template`-Tabelle (rc_number FK, name UNIQUE per RC, format ENUM, columns JSONB, created_at, updated_at)
- **Optional: `export_log`-Tabelle** für Audit (export_id, rc_number, template_id NULL falls Template gelöscht, admin_user_id, applications_count, exported_at) — alternativ in `status_log` integriert
- **excelize-Wiederverwendung** aus `internal/excel/generator.go`, aber als generischer Renderer mit Column-Definitionen
- **Streaming-Output** via Go's `http.ResponseWriter` mit Chunked-Transfer für große Exports
- **Frontend**: shadcn-Komponenten (Dialog, Select, Input, Button, Table für Preview) — kein neues UI-Framework
- **Bulk-Action-Integration**: in `admin-application-table.tsx` als neue Option in der bestehenden Bulk-Action-Leiste
- **Single-Action-Integration**: in `admin-application-detail.tsx` als neuer Menüeintrag im Aktionen-Bereich
- **Tenant-Isolation**: alle Endpoints via `checkTenantAccess` geschützt
- **Auth**: `eeg_admin`-Rolle reicht (kein Superuser-Privileg nötig — EEGs verwalten ihre eigenen Templates)

## Resolved Decisions (aus `/requirements` 2026-05-23)

- **Q1 Zählpunkt-Multi-Value:** komma-getrennt in einer Spalte (Standard für V1). „Zeile pro Zählpunkt" als optionale V2-Erweiterung möglich.
- **Q2 Standard-Templates:** ja, **3 vordefinierte** (Newsletter, CRM-Stammdaten, Buchhaltung). Read-only als Klon-Basis.
- **Q3 Filter:** kein eigener Filter im Template — Selektion erfolgt **über die Antragsliste** (existierende PROJ-3-Filter + PROJ-25-Bulk-Action-Pattern) oder **Single aus Detail**. Template-Editor zeigt nur eine **Vorschau mit den letzten 5 importierten Mitgliedern** zum Mapping-Testen.
- **Q4 Persistierung:** fly-by im V1, Audit-Log nur mit Metadata. DOC-Archiv-Speicherung als V2-Option offen.

## Open Questions (für `/grill-me` zur Verschärfung)

Diese Punkte sollten im anschließenden `/grill-me`-Lauf stressgetestet werden:

- **Header-Sprache bei Standard-Templates**: nur Deutsch oder auch Englisch/mehrsprachig? (Newsletter-Tools sind oft englisch-konfiguriert.)
- **Maximum Anzahl Anträge pro Bulk-Export**: 1.000 als Erstwert — performance-getestet?
- **Audit-Log-Aufbewahrung**: indefinite oder mit Retention-Policy (z. B. 2 Jahre)?
- **Versionierung von Templates**: was passiert mit alten Audit-Log-Einträgen, wenn ein Template später geändert wird? Spalten-Konfig in Audit-Log snapshotten oder nur Template-ID referenzieren?
- **Standard-Template-Sprache bei EEG mit englischen Mitgliedern**: wird das Label-Format (z. B. „Privatperson") immer Deutsch sein, oder konfigurierbar?
- **Performance bei XLSX vs. CSV**: für >500 Zeilen ist XLSX deutlich langsamer (excelize-Overhead). Hinweis im UI?

## Notes

- **Phase 2 wird ein separates PROJ** (CRM-API-Integration), wenn Phase 1 produktiv läuft. Plugin-Architektur dieser Spec soll Phase-2-Erweiterung möglichst ohne Refactoring erlauben.
- **PROJ-25 (Bulk-Aktionen)** stellt das etablierte UI-Pattern bereit — Export ist semantisch nur eine weitere Bulk-Aktion (neben Genehmigen/Ablehnen/Zur-Prüfung).
- **Re-Use-Potenzial:** Das Mapping-Konzept (Source-Feld → Target-Feld mit Format-Transformation) wird in Phase 2 (Zoho/HubSpot-Adapter) nahezu 1:1 wiederverwendet.

---

# Phase 2 Ausblick: CRM-API-Integration

(Separates späteres PROJ, wenn Phase 1 produktiv läuft und EEG-Bedarf nach Real-Time-Push besteht.)

## Skizze

- **Plugin-/Adapter-Pattern**: `ExternalSystemAdapter`-Interface, erster Adapter `ZohoCRMAdapter`, später HubSpot/Salesforce/Pipedrive
- **Push-basiert**, asynchron bei Status-Übergang (z. B. `imported`)
- **OAuth2** pro EEG für Auth, Tokens verschlüsselt in DB
- **Mapping-Konzept aus Phase 1 wiederverwenden**: dieselbe Spalten-Definition, aber Target-Feld ist ein CRM-Field statt einer Excel-Spalte
- **Idempotenz** via `EEG_Onboarding_ID` als Custom-Field im CRM
- **DSGVO**: EEG braucht eigenen AVV mit CRM-Provider, Hinweistext im Setup-Flow

## Phase-2-spezifische Open Questions

- MVP-Umfang: nur Zoho oder gleich generisches Framework?
- Bidirektionalität: CRM-Änderungen zurück ins Onboarding-System?
- Konflikt-Strategie bei Duplikaten im CRM (skip / overwrite / merge)?
- Trigger: nur bei `imported` oder auch bei späteren Updates (Adresswechsel etc.)?

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
