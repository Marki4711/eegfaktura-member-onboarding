# PROJ-12: SEPA-Lastschriftmandat als PDF-Anhang im Willkommensmail

## Status: Planned
**Created:** 2026-04-23
**Last Updated:** 2026-04-23

## Dependencies
- Requires: PROJ-1 (Public Registration) — liefert Mitgliedsdaten (Name, Anschrift, IBAN) aus dem Antrag
- Requires: PROJ-6 (E-Mail-Benachrichtigungen) — bestehende E-Mail-Infrastruktur wird erweitert

## User Stories

- Als **neues Mitglied** möchte ich zusammen mit meiner Bestätigungs-E-Mail ein ausgefülltes SEPA-Lastschriftmandat als PDF erhalten, damit ich es ausdrucken, unterschreiben und an die EEG zurückschicken kann.
- Als **EEG-Administrator** möchte ich steuern können, ob dem Willkommensmail ein SEPA-Lastschriftmandat beigefügt wird oder nicht, damit ich diese Funktion nur aktiviere wenn meine EEG tatsächlich SEPA-Lastschriften einsetzt.
- Als **EEG-Administrator** möchte ich den Namen meiner Energiegemeinschaft, die Anschrift und die Creditor-ID im System hinterlegen können, damit diese Daten korrekt im SEPA-Mandat erscheinen.
- Als **EEG-Administrator** möchte ich diese EEG-Stammdaten in der Admin-Oberfläche bearbeiten können, damit ich sie bei Änderungen (z.B. neue Creditor-ID) aktuell halten kann.
- Als **Betreiber** möchte ich, dass ein Fehler bei der PDF-Generierung den E-Mail-Versand und die Einreichung nicht blockiert, damit das Mitglied seinen Antrag trotzdem erfolgreich abschicken kann.

## Acceptance Criteria

### EEG-Stammdaten im Admin-Backend
- [ ] Der Admin kann pro EEG folgende Felder hinterlegen und bearbeiten: **EEG-Name**, **Anschrift** (einzeiliger Freitext, z.B. „Laab 8, 1010 Musterstadt"), **Creditor-ID**
- [ ] Die Felder sind alle optional — fehlen sie, wird das PDF-Mandat nicht generiert (auch wenn `sepa_mandate_enabled = true`)
- [ ] Die Werte werden in `member_onboarding.registration_entrypoint` gespeichert (neue Felder)
- [ ] Änderungen sind sofort nach Speichern wirksam (kein Cache)

### Aktivierung/Deaktivierung pro EEG
- [ ] Im Admin-Backend gibt es einen Schalter (Toggle) „SEPA-Lastschriftmandat anhängen"
- [ ] Der Schalter ist pro EEG (RC-Nummer) steuerbar
- [ ] Ist der Schalter deaktiviert, wird kein PDF generiert und kein Anhang versendet — Standardwert ist **deaktiviert**
- [ ] Ist der Schalter aktiviert aber EEG-Name, Anschrift oder Creditor-ID fehlen, wird das PDF nicht generiert (kein Fehler für das Mitglied, aber ein Log-Eintrag)

### PDF-Inhalt (SEPA-Lastschriftmandat)
- [ ] Das PDF enthält alle Pflichtbestandteile eines SEPA-Lastschriftmandats:
  - Mandatsreferenz-Feld (mit Hinweis „wird von [EEG-Name] ausgefüllt")
  - Zahlungsempfänger: EEG-Name, Anschrift, Creditor-ID (aus DB)
  - Ermächtigungstext (standardisierter SEPA-Text auf Deutsch)
  - Zahlungsart: „wiederkehrend" vorausgewählt
  - Zahlungspflichtiger: Name des Mitglieds, Anschrift des Mitglieds, IBAN (aus Antrag)
  - Unterschriftsfeld (Datum/Ort + Unterschrift — leer, zum Ausfüllen)
  - BIC-Fußnote (gesetzlicher Hinweistext)
- [ ] Das PDF wird serverseitig generiert — das Mitglied erhält es fertig ausgefüllt mit seinen Daten
- [ ] Das PDF ist auf Deutsch

### E-Mail-Anhang
- [ ] Das PDF wird als Anhang zur bestehenden Bestätigungs-E-Mail an das Mitglied hinzugefügt (nicht als separate E-Mail)
- [ ] Dateiname: `sepa-lastschriftmandat.pdf`
- [ ] Wird nur bei Ersteinreichung (Status `draft → submitted`) angehängt — nicht bei Wiedereinreichung
- [ ] Die EEG-Benachrichtigungs-E-Mail erhält keinen PDF-Anhang

### Fehlerverhalten
- [ ] Schlägt die PDF-Generierung fehl, wird der Fehler geloggt und die E-Mail wird ohne Anhang versendet
- [ ] Die Einreichung wird nicht blockiert — weder durch PDF-Fehler noch durch E-Mail-Fehler

## Edge Cases

- **EEG-Name, Anschrift oder Creditor-ID fehlen:** Kein PDF wird generiert; E-Mail wird ohne Anhang versendet; Log-Eintrag: „SEPA PDF not generated — missing EEG fields"
- **SEPA-Mandat deaktiviert:** Kein PDF, kein Anhang, kein Fehler
- **PDF-Generierung schlägt fehl (z.B. Speicher, Library-Fehler):** Fehler wird geloggt; E-Mail wird ohne Anhang gesendet; Einreichung nicht blockiert
- **Mitglied hat keinen Account Holder / IBAN:** Kann nicht vorkommen — beide sind Pflichtfelder bei der Einreichung
- **Sehr langer EEG-Name oder Anschrift:** PDF-Layout muss mit langen Texten umgehen (Zeilenumbruch oder Kürzung mit Ellipsis)
- **Admin aktiviert Mandat ohne Creditor-ID gesetzt zu haben:** UI zeigt Warnung „Bitte alle EEG-Felder ausfüllen bevor Sie die Funktion aktivieren"

## Technical Requirements

- **PDF-Generierung:** Serverseitig in Go — bevorzugt `github.com/go-pdf/fpdf` (ehemals jung-kurt/gofpdf, aktiv maintained, keine CGO-Abhängigkeit)
- **Neue DB-Felder** in `member_onboarding.registration_entrypoint`:
  - `eeg_name TEXT NULL`
  - `eeg_address TEXT NULL`
  - `creditor_id VARCHAR(35) NULL`
  - `sepa_mandate_enabled BOOLEAN NOT NULL DEFAULT FALSE`
- **Backend:** PDF-Generierung als eigenes Package `internal/pdf/` mit Interface für Testbarkeit
- **E-Mail-Integration:** `internal/mail/service.go` wird erweitert — PDF-Bytes werden als Anhang übergeben
- **API:** Neue/erweiterte Admin-Endpunkte für EEG-Stammdaten (EEG-Name, Anschrift, Creditor-ID, SEPA-Toggle)
- **Frontend:** Neuer Abschnitt in der Admin-Settings-Seite für EEG-Stammdaten + SEPA-Toggle

---
<!-- Sections below are added by subsequent skills -->
