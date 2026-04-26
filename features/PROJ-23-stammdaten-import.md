# PROJ-23: Stammdaten-Import aus eegFaktura-Excel-Export

## Status: Planned
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

## Dependencies
- Requires: PROJ-3 (Admin Frontend UI) — Import-Button in bestehender EEG-Einstellungsseite
- Requires: PROJ-8 (Konfigurierbare Felder pro EEG) — `registration_entrypoint` als zentrale EEG-Tabelle

## Hintergrund

EEG-Administratoren müssen die Stammdaten ihrer EEG im Member-Onboarding-Tool manuell pflegen (Name, Adresse, Bankverbindung usw.). Diese Daten existieren bereits in eegFaktura und können dort als Excel-Datei exportiert werden. Ohne Import-Funktion müssen Admins Änderungen doppelt pflegen — einmal in eegFaktura, einmal im Onboarding-Tool. Das Feature ermöglicht es, den Stammdaten-Export aus eegFaktura direkt hochzuladen, damit das System die relevanten Felder automatisch übernimmt.

## User Stories

- Als **EEG-Admin** möchte ich die Excel-Stammdatei aus eegFaktura in den Einstellungen hochladen können, damit ich Namen, Adresse und Bankverbindung der EEG nicht händisch eingeben muss.
- Als **EEG-Admin** möchte ich eine klare Erfolgsmeldung nach dem Import sehen, damit ich weiss, welche Felder aktualisiert wurden.
- Als **EEG-Admin** möchte ich eine verständliche Fehlermeldung erhalten, wenn die Excel-Datei nicht zur meiner EEG passt, damit ich nicht versehentlich falsche Daten importiere.
- Als **Superuser** möchte ich Stammdaten für jede beliebige EEG importieren können, damit ich bei Bedarf zentral unterstützen kann.
- Als **System** möchte ich sicherstellen, dass ein Admin nur Daten für seine eigene EEG importieren kann, damit Tenant-Isolation gewahrt bleibt.

## Acceptance Criteria

### Datei-Upload

- [ ] In der bestehenden EEG-Einstellungsseite (`/admin/settings`) gibt es einen neuen Abschnitt „Stammdaten-Import"
- [ ] Ein Datei-Upload-Button akzeptiert ausschließlich `.xlsx`-Dateien (MIME-Typ `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`)
- [ ] Maximale Dateigröße: 10 MB
- [ ] Andere Dateiformate (.xls, .csv, .pdf) werden mit einer verständlichen Fehlermeldung abgelehnt

### RC-Nummern-Erkennung

- [ ] Das System liest den **Sheet-Namen** der Excel-Datei aus, der der RC-Nummer der EEG entspricht (z. B. `RC12345678`)
- [ ] Existiert kein Sheet dessen Name einer bekannten RC-Nummer im System entspricht, wird ein Fehler angezeigt: „Kein passendes Tabellenblatt gefunden. Erwartet: RC-Nummer als Blattname."
- [ ] Für einen Tenant-Admin wird zusätzlich geprüft, dass die erkannte RC-Nummer zu seiner eigenen EEG gehört — andernfalls wird der Import mit Fehler 403 abgelehnt

### Importierte Felder

Die folgenden Felder werden aus dem passenden Sheet ausgelesen und in `registration_entrypoint` gespeichert:

| Feld in DB | Feldbezeichnung |
|---|---|
| `eeg_name` | EEG-Name |
| `eeg_id` | Gemeinschafts-ID |
| `contact_email` | E-Mail-Adresse der EEG |
| `eeg_street` | Straße |
| `eeg_street_number` | Hausnummer |
| `eeg_zip` | PLZ |
| `eeg_city` | Wohnort / Ort |
| `eeg_iban` | IBAN der EEG (neu) |
| `eeg_account_holder` | Kontoinhaber der EEG (neu) |

> **Hinweis:** Die exakten Zellen/Spalten im eegFaktura-Excel-Export werden im Tech Design (Architecture) definiert, sobald eine Beispiel-Exportdatei vorliegt.

- [ ] Alle importierten Felder werden vollständig überschrieben (auch wenn der bestehende Wert manuell gesetzt war)
- [ ] Felder, die im Excel leer sind, werden als leerer String / NULL gespeichert
- [ ] Felder, die im Tech Design nicht gemappt sind, werden ignoriert (kein Fehler)

### Neue Datenbankfelder

- [ ] Neue Spalten `eeg_iban` (TEXT, nullable) und `eeg_account_holder` (TEXT, nullable) in `member_onboarding.registration_entrypoint`
- [ ] Migration als `db/migrations/`-Datei

### Feedback an den Admin

- [ ] Erfolgreicher Import: Meldung mit Liste der aktualisierten Felder und deren neuen Werten
- [ ] Fehlgeschlagener Import: verständliche Fehlermeldung (falsches Format, RC nicht gefunden, kein Zugriff)
- [ ] Während des Uploads: Ladezustand sichtbar (Spinner oder Disabled-Button)

### Sicherheit & Zugriffskontrolle

- [ ] Nur authentifizierte Admins (Keycloak) können den Import-Endpunkt aufrufen
- [ ] Tenant-Admins können ausschließlich ihre eigene EEG importieren (Tenant-Isolation)
- [ ] Superuser können jede EEG importieren
- [ ] Die hochgeladene Datei wird nur im Arbeitsspeicher verarbeitet — keine persistente Speicherung der Excel-Datei auf dem Server

## Edge Cases

- **Mehrere Sheets in der Excel-Datei:** Das System sucht nur nach dem Sheet mit der RC-Nummer der eigenen EEG; weitere Sheets werden ignoriert
- **Sheet-Name enthält Leerzeichen/Sonderzeichen:** Der Sheet-Name wird nach Trim mit der RC-Nummer verglichen
- **Excel aus einer anderen eegFaktura-Installation:** RC-Nummer wird nicht gefunden → Fehlermeldung
- **Tenant-Admin lädt Excel einer fremden EEG hoch:** RC-Nummer gefunden, aber nicht zugänglich → 403, Fehlermeldung „Diese EEG gehört nicht zu Ihrem Account"
- **Excel-Datei ist beschädigt / kein gültiges XLSX:** Parser-Fehler → verständliche Fehlermeldung, kein Absturz
- **Feld `eeg_iban` enthält ungültiges IBAN-Format:** Import wird trotzdem durchgeführt (keine Validierung der IBAN auf Bankebene) — Admins sind für Korrektheit verantwortlich
- **Import-Endpunkt unter Last (großes Excel):** Maximale Dateigröße 10 MB verhindert Ressourcenerschöpfung

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
