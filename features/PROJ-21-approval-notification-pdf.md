# PROJ-21: Genehmigungs-Benachrichtigung mit Beitrittsbestätigung PDF

## Status: Planned
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

## Dependencies
- Requires: PROJ-6 (E-Mail-Benachrichtigungen) — nutzt bestehende SMTP-Infrastruktur
- Requires: PROJ-12 (SEPA-Lastschriftmandat PDF) — nutzt bestehende PDF-Generierungs-Infrastruktur (fpdf)
- Requires: PROJ-2 (Admin Review) — approved-Status-Übergang ist der Auslöser

## Hintergrund

Wenn ein Admin einen Antrag auf `approved` setzt, gibt es derzeit keine automatische Benachrichtigung an den EEG-Betreiber und kein Dokument, das den Beitritt des Mitglieds bestätigt. Die EEG benötigt ein revisionssicheres Dokument, das den Beitritt belegt, alle relevanten Daten des Mitglieds enthält, die erteilten Zustimmungen dokumentiert und den Bearbeitungsweg (Statusverlauf) nachvollziehbar macht. Dieses Dokument soll entweder ausgedruckt, oder digital abgelegt werden können. Ein Feld für die Mitgliedsnummer wird im Dokument freigelassen, damit es nach der Genehmigung handschriftlich oder per PDF-Reader ausgefüllt werden kann.

## User Stories

- Als **EEG-Betreiber** möchte ich automatisch eine E-Mail erhalten, wenn ein Antrag genehmigt wird, damit ich sofort über die neue Mitgliedschaft informiert bin.
- Als **EEG-Betreiber** möchte ich die Beitrittsbestätigung als PDF-Anhang erhalten, damit ich sie digital ablegen oder ausdrucken und unterschreiben kann.
- Als **EEG-Betreiber** möchte ich im PDF ein leeres Feld für die Mitgliedsnummer sehen, damit ich die EEG-interne Nummer nach der Aufnahme nachtragen kann.
- Als **EEG-Betreiber** möchte ich im PDF die erteilten Zustimmungen des Mitglieds (Datenschutz, Vereinsstatuten usw.) dokumentiert sehen, damit ich bei rechtlichen Fragen die Einwilligung nachweisen kann.
- Als **EEG-Betreiber** möchte ich den vollständigen Statusverlauf des Antrags im PDF sehen, damit der Bearbeitungsweg nachvollziehbar und revisionssicher dokumentiert ist.
- Als **Betreiber** möchte ich, dass ein Fehler bei der PDF-Generierung den Status-Übergang nicht blockiert, damit der Admin den Antrag trotzdem auf `approved` setzen kann.

## Acceptance Criteria

### Auslöser
- [ ] Die Benachrichtigung wird ausgelöst, wenn ein Antrag in den Status `approved` wechselt
- [ ] Auslöser ist der Admin-Statusübergang (beliebiger vorheriger Status → `approved`)
- [ ] Wird `approved` → `import_failed` → `approved` (Re-Approval), wird erneut eine E-Mail gesendet
- [ ] Hat die EEG keine `contact_email`, wird weder E-Mail noch PDF generiert (kein Fehler)

### E-Mail an EEG-Betreiber
- [ ] Empfänger: `contact_email` der zugehörigen EEG aus `registration_entrypoint`
- [ ] Betreff: „Mitgliedsantrag genehmigt – [Vorname Nachname / Firmenname] ([Referenznummer])"
- [ ] Inhalt: kurze Mitteilung dass der Antrag genehmigt wurde, Name des Mitglieds, Referenznummer, Hinweis auf den PDF-Anhang
- [ ] Die E-Mail ist auf Deutsch
- [ ] PDF als Anhang beigefügt (Dateiname: `beitrittsbestaetigung-[referenznummer].pdf`)

### PDF-Inhalt: Beitrittsbestätigung
Das PDF ist ein strukturiertes A4-Dokument mit folgendem Inhalt:

#### Kopfzeile
- [ ] Titel: „Beitrittsbestätigung"
- [ ] EEG-Name und RC-Nummer
- [ ] Ausstellungsdatum (Datum der Genehmigung)

#### Mitgliedsdaten
- [ ] Mitgliedstyp (Privatperson / Landwirt / Unternehmen)
- [ ] Name (Vorname + Nachname) oder Firmenname + UID/Firmenbuchnummer
- [ ] Geburtsdatum (falls vorhanden)
- [ ] Adresse (Straße, Hausnummer, PLZ, Ort)
- [ ] E-Mail-Adresse
- [ ] Telefon (falls vorhanden)

#### Bankverbindung
- [ ] IBAN
- [ ] Kontoinhaber (falls vorhanden)
- [ ] SEPA-Mandatsart (Basislastschrift / Firmenlastschrift / Per E-Mail)

#### Zählpunkte
- [ ] Tabelle: Zählpunktnummer, Richtung, Teilnahmefaktor

#### Erteilte Zustimmungen
- [ ] Liste aller Dokumente, denen zugestimmt wurde (Titel + URL)
- [ ] Datum der Zustimmung (= Einreichungsdatum)

#### Statusverlauf
- [ ] Tabelle: Status (von → nach), Zeitstempel, ggf. Kommentar aus Admin-Notiz

#### Mitgliedsnummer
- [ ] Sichtbares, beschriftetes Leerfeld: „Mitgliedsnummer: _________________________"
- [ ] Hinweis: „Wird von [EEG-Name] vergeben"

#### Konfigurierbare Felder (optional)
- [ ] Falls konfigurierbare Felder ausgefüllt sind (Wärmepumpe, Personenanzahl usw.): werden als zusätzlicher Abschnitt aufgeführt
- [ ] Leere Felder werden nicht aufgeführt

### Fehlerverhalten
- [ ] Schlägt die PDF-Generierung fehl, wird der Fehler geloggt; die E-Mail wird ohne PDF-Anhang gesendet (mit Hinweis „PDF konnte nicht generiert werden")
- [ ] Schlägt auch der E-Mail-Versand fehl, wird der Fehler geloggt; der Status-Übergang zu `approved` bleibt gültig
- [ ] Kein Absturz bei fehlenden optionalen Feldern (NULL-Werte werden stillschweigend ausgelassen)

### Template
- [ ] Neues E-Mail-Template: `internal/mail/templates/application_approved_eeg.html`
- [ ] PDF-Generator in `internal/pdf/` (eigene Datei, z. B. `approval_pdf.go`)
- [ ] Die E-Mail ist auf Deutsch

## Edge Cases

- **Antrag von Unternehmen:** Kein Vorname/Nachname im PDF/Mail; Firmenname + UID steht an erster Stelle
- **Keine `contact_email`:** Weder E-Mail noch PDF wird generiert; kein Fehler, kein Log-Warn (bereits in PROJ-6 definiertes Verhalten)
- **Statusverlauf ist leer / hat nur einen Eintrag:** Tabelle zeigt alle vorhandenen Einträge; ein einzelner Eintrag ist valide
- **Keine Zustimmungen gespeichert:** Abschnitt „Erteilte Zustimmungen" entfällt; kein Fehler
- **Re-Approval (approved → import_failed → approved):** Neue E-Mail wird gesendet; das PDF enthält den vollständigen Statusverlauf inkl. `import_failed`-Einträge
- **SMTP nicht konfiguriert (`SMTP_HOST` fehlt):** Kein Versuch, kein Fehler (NoOpMailService)
- **Konfigurierbare Felder alle leer:** Abschnitt entfällt komplett aus dem PDF

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
