# PROJ-17: Excel-Export für eegFaktura-Import

## Status: In Review
**Created:** 2026-04-25
**Last Updated:** 2026-04-25

## Kontext

PROJ-4 (direkter API-Import in eegFaktura) wird aus Security-Gründen zurückgestellt.
Als Alternative: der Admin generiert aus einem genehmigten Antrag ein Excel-File im
eegFaktura-Import-Format und importiert es manuell über die Excel-Import-Funktion in eegFaktura.

Das reduziert die Kopplung zwischen den Systemen: kein direkter API-Aufruf, kein
gemeinsamer Datenbankzugriff, keine Service-Account-Credentials.

**Referenz-Template:** `https://docs.eegfaktura.at/attachments/15`

## Dependencies
- Requires: PROJ-2 (Admin Review) — Antrag muss genehmigt sein
- Alternative to: PROJ-4 (Core Import via API)

## User Stories

- Als **EEG-Administrator** möchte ich für einen genehmigten Antrag ein Excel-File herunterladen, das ich direkt in eegFaktura importieren kann, damit ich das Mitglied ohne manuelles Eintippen anlegen kann.
- Als **EEG-Administrator** möchte ich, dass das Excel alle notwendigen Felder (Stammdaten, Zählpunkte, IBAN) korrekt vorausgefüllt hat, damit der Import in eegFaktura ohne Korrekturen funktioniert.
- Als **EEG-Administrator** möchte ich das Excel für jeden genehmigten Antrag mit einem Klick generieren können, ohne eine externe Anwendung zu öffnen.

## Acceptance Criteria

### Backend

- [ ] `GET /api/admin/applications/{id}/export/excel` gibt eine `.xlsx`-Datei zurück.
- [ ] Der Endpunkt ist über Keycloak JWT gesichert und prüft Tenant-Zugehörigkeit.
- [ ] Der Export ist nur für Anträge in Status `approved`, `imported` oder `import_failed` verfügbar — bei anderen Status: `409 Conflict`.
- [ ] Die generierte Datei enthält eine Header-Zeile mit den Spaltenbezeichnungen des eegFaktura-Templates.
- [ ] Die zweite Zeile enthält die Importer-Markierung `[### Leerzeile für Importer ###]`.
- [ ] Ab Zeile 3: eine Datenzeile pro Zählpunkt (Mitgliedsdaten werden pro Zeile wiederholt).
- [ ] Alle verfügbaren Felder aus dem Antrag werden korrekt in die entsprechenden Spalten gemappt (siehe Spalten-Mapping unten).
- [ ] Der Dateiname enthält die Referenznummer: `{referenceNumber}.xlsx` (z.B. `MO-2026-000001.xlsx`).
- [ ] Response-Header: `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` und `Content-Disposition: attachment; filename="{referenceNumber}.xlsx"`.

### Frontend

- [ ] Im Admin-Detail-View erscheint ein Button "Excel herunterladen" für Anträge in Status `approved`, `imported` oder `import_failed`.
- [ ] Ein Klick auf den Button löst den Download aus (Browser-Download-Dialog).
- [ ] Der Button ist nicht vorhanden für andere Status (draft, submitted, under_review, etc.).
- [ ] Bei Fehler wird eine Toast-Meldung angezeigt.

## Spalten-Mapping (eegFaktura-Import-Template)

| Spalte | Feldname | Quelle | Wert |
|--------|----------|--------|------|
| A | Netzbetreiber | registration_entrypoint | leer (V1) |
| B | Gemeinschafts-ID | registration_entrypoint.eeg_id | direkt |
| C | Ortsgebiet | — | `LOKAL` (Default) |
| D | PLZ | application.resident_zip | direkt |
| E | Ort | application.resident_city | direkt |
| F | Straße | application.resident_street | direkt |
| G | Hausnummer | application.resident_street_number | direkt |
| H | Stiege | — | leer |
| I | Stock | — | leer |
| J | Tür | — | leer |
| K | Adresszusatz | — | leer |
| L | Zählpunkt | metering_point.metering_point | direkt |
| M | Energierichtung | metering_point.direction | `CONSUMPTION`/`GENERATION` |
| N | EquipmentNr | metering_point.transformer | direkt |
| O | ObjektName | metering_point.installation_name | direkt |
| P | Überschusseinspeisung | — | leer |
| Q | Energiequelle | — | leer |
| R | Verteilungsmodell | — | leer |
| S | Zugeteilte Menge in Prozent | metering_point.participation_factor | als Zahl (z.B. 100) |
| T | TitelVor | — | leer |
| U | Name 1 | application.lastname / company_name | Nachname (privat) oder Firmenname |
| V | Name 2 | application.firstname | Vorname (privat), leer bei Unternehmen |
| W | TitelNach | — | leer |
| X | BusinessRole | application.member_type | `privat` oder `business` |
| Y | Mitglied seit | application.membership_start_date | im Format `D.M.YYYY` |
| Z | IBAN | application.iban | direkt |
| AA | Kontoinhaber | application.account_holder | direkt |
| AB | Bankname | — | leer |
| AC | Email | application.email | direkt |
| AD | TelefonNr | application.phone | direkt |
| AE | SteuerNr | — | leer |
| AF | UmsatzsteuerNr | application.uid_number | direkt |
| AG | MitgliedsNr | application.reference_number | direkt |
| AH | Zählpunktstatus | — | `ACTIVATED` (Default) |
| AI | registriert seit | application.created_at | im Format `D.M.YYYY` |
| AJ | Meter Codes | — | leer |

**BusinessRole-Mapping:**
- `private`, `farmer` → `privat`
- `company`, `association`, `municipality` → `business`

## Edge Cases

- **Antrag hat keinen Zählpunkt:** Darf nicht vorkommen (Pflichtfeld bei Einreichung), aber → Export-Fehler mit 422.
- **Antrag in falschem Status:** 409 Conflict zurückgeben.
- **Dateinamen-Sicherheit:** Referenznummer enthält nur alphanumerische Zeichen und `-` → kein Sanitizing nötig, trotzdem sicherstellen.
- **Mehrere Zählpunkte:** Jeder Zählpunkt ist eine eigene Zeile, alle Mitgliedsdaten werden wiederholt.
- **Fehlende optionale Felder** (phone, uid_number, etc.): leer lassen.

## Tech Design (Solution Architect)

### Backend
- Neue Methode `ExportApplicationExcel(id uuid.UUID)` in `AdminApplicationService`
- Liest Application + MeteringPoints + Entrypoint (für eeg_id)
- Verwendet `github.com/xuri/excelize/v2` zur Excel-Generierung
- Gibt `([]byte, string, error)` zurück (bytes, filename, error)
- Neuer Handler `ExportApplicationExcel` in `internal/http/admin.go`
- Route: `GET /api/admin/applications/{id}/export/excel`

### Frontend
- Neuer Button in der Admin-Detail-View (neben "Löschen" o.ä.)
- Fetch mit `responseType: blob`, dann dynamischer `<a download>`-Link
- Nur sichtbar bei Status `approved`, `imported`, `import_failed`

---
<!-- Sections below are added by subsequent skills -->

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
