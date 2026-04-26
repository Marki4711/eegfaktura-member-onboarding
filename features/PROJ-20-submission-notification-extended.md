# PROJ-20: Vollständige Antragsdaten in EEG-Einreichungsbenachrichtigung

## Status: Planned
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

## Dependencies
- Requires: PROJ-6 (E-Mail-Benachrichtigungen) — erweitert das bestehende EEG-Benachrichtigungs-Template

## Hintergrund

Die bestehende EEG-Einreichungsbenachrichtigung (PROJ-6) enthält nur: Name, E-Mail-Adresse, Referenznummer und Zählpunkte. Der EEG-Betreiber erhält damit nicht alle relevanten Antragsdaten und muss sich im Admin-Bereich einloggen, um Vollständiges einzusehen. Die erweiterte E-Mail soll alle eingereichten Informationen enthalten, sodass der Betreiber den Antrag auch direkt aus der E-Mail heraus vollständig beurteilen kann.

## User Stories

- Als **EEG-Betreiber** möchte ich in der Einreichungs-E-Mail alle Antragsdaten des Mitglieds sehen (Adresse, IBAN, Telefon, Geburtsdatum usw.), damit ich den Antrag ohne Login in den Admin-Bereich beurteilen kann.
- Als **EEG-Betreiber** möchte ich die Mitgliedsart (Privatperson / Landwirt / Unternehmen) in der E-Mail erkennen können, damit ich sofort den richtigen Bearbeitungspfad einschlagen kann.
- Als **EEG-Betreiber** möchte ich alle angemeldeten Zählpunkte mit Richtung und Teilnahmefaktor sehen, damit ich die technische Planung sofort anstoßen kann.
- Als **EEG-Betreiber** möchte ich sehen, welchen konfigurierbaren Felder der Antragsteller ausgefüllt hat (z. B. Wärmepumpe vorhanden, Personenanzahl), damit ich ein vollständiges Bild des Haushalts habe.
- Als **Betreiber** möchte ich, dass ein Fehler beim Template-Rendering den Versand nicht blockiert, damit das Mitglied den Antrag trotzdem erfolgreich einreichen kann.

## Acceptance Criteria

### Antragstellerdaten
- [ ] Die E-Mail enthält alle Antragstellerdaten je nach Mitgliedstyp:
  - **Privatperson / Landwirt:** Vorname, Nachname, Geburtsdatum (falls ausgefüllt)
  - **Unternehmen:** Firmenname, UID-Nummer (falls ausgefüllt), Firmenbuchnummer (falls ausgefüllt)
- [ ] Kontaktdaten: E-Mail-Adresse, Telefon (falls ausgefüllt)
- [ ] Wohnadresse: Straße, Hausnummer, PLZ, Ort
- [ ] Mitgliedstyp (Privatperson / Landwirt / Unternehmen) ist deutlich erkennbar

### Bankverbindung
- [ ] IBAN (vollständig — kein Masking in der EEG-internen E-Mail)
- [ ] Kontoinhaber (falls verschieden vom Antragsteller)
- [ ] Art der SEPA-Ermächtigung: Basislastschrift / Firmenlastschrift / Per E-Mail

### Konfigurierbare Felder
- [ ] Alle konfigurierbaren Felder, die nicht auf `hidden` gesetzt sind und einen Wert haben, werden aufgelistet (z. B. Wärmepumpe: Ja, Personen im Haushalt: 3)
- [ ] Felder ohne Wert werden nicht aufgeführt

### Zählpunkte
- [ ] Bestehende Darstellung bleibt: Zählpunktnummer, Richtung (Verbrauch / Einspeisung), Teilnahmefaktor

### Zusatzinformationen
- [ ] Referenznummer
- [ ] Einreichungsdatum und -uhrzeit
- [ ] Link zur Admin-Detailansicht (direkter Link auf `/admin/applications/{id}`)
- [ ] RC-Nummer der EEG

### Fehlerverhalten
- [ ] Schlägt das Rendering fehl, wird der Fehler geloggt; der Einreichungs-Endpunkt gibt dennoch `200 OK` zurück
- [ ] Kein Absturz bei fehlenden optionalen Feldern (NULL-Werte werden stillschweigend ausgelassen)

### Template
- [ ] Das bestehende Template `application_submitted_eeg.html` wird ersetzt — kein neues Template
- [ ] Die E-Mail ist auf Deutsch
- [ ] Klar gegliedertes HTML-Layout (Abschnitte: Antragsteller, Bankverbindung, Zählpunkte, Zusätzliche Informationen)

## Edge Cases

- **IBAN fehlt:** Kann bei SEPA per E-Mail nicht vorkommen (kein Pflichtfeld in diesem Modus) → Abschnitt Bankverbindung wird ausgelassen wenn IBAN leer
- **Mitglied ist Unternehmen:** Vorname/Nachname werden nicht angezeigt; Firmenname steht an oberster Stelle
- **Kein einziges konfigurierbares Feld ausgefüllt:** Abschnitt „Zusätzliche Informationen" entfällt komplett
- **EEG hat keine `contact_email`:** Kein Versuch, keine Fehlermeldung (bleibt unverändert wie in PROJ-6)
- **Backend-URL nicht konfiguriert:** Link zur Admin-Detailansicht entfällt aus dem Template — kein Fehler

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
