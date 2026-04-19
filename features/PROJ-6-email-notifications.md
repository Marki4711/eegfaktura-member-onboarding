# PROJ-6: E-Mail-Benachrichtigungen bei Antragseinreichung

## Status: Planned
**Created:** 2026-04-19
**Last Updated:** 2026-04-19

## Dependencies
- Requires: PROJ-1 (Public Registration) — liefert den Einreichungs-Trigger (submit endpoint)
- Requires: Neues Feld `contact_email` in `member_onboarding.registration_entrypoint`

## Hintergrund

Beim aktuellen Stand erhält weder das einreichende Mitglied noch die zugehörige EEG eine Benachrichtigung, wenn ein Antrag eingereicht wird. Diese fehlende Rückmeldung war ein konkretes Feedback aus der Teamdiskussion.

E-Mails werden direkt per SMTP verschickt — kein externer Mail-Microservice, kein gRPC. Templates werden per Go `embed.FS` ins Binary eingebettet.

## User Stories

- Als **neues Mitglied** möchte ich nach dem Absenden meines Antrags eine Bestätigungs-E-Mail erhalten, damit ich weiß, dass mein Antrag eingegangen ist und ich auf Rückmeldung warten kann.
- Als **EEG-Administrator** möchte ich eine E-Mail erhalten, wenn ein neuer Antrag eingereicht wurde, damit ich zeitnah mit der Prüfung beginnen kann.
- Als **Betreiber** möchte ich, dass ein fehlgeschlagener E-Mail-Versand den Einreichungsprozess nicht blockiert, damit das Mitglied seinen Antrag trotzdem erfolgreich einreichen kann.
- Als **EEG-Administrator** möchte ich die Kontakt-E-Mail-Adresse meiner EEG im System hinterlegt haben, damit Benachrichtigungen an die richtige Adresse zugestellt werden.

## Acceptance Criteria

### E-Mail an das Mitglied (Bestätigung)
- [ ] Nach erfolgreichem Einreichen (Status `draft` → `submitted`) wird eine Bestätigungs-E-Mail an die E-Mail-Adresse des Mitglieds gesendet
- [ ] Die E-Mail enthält: Anrede mit Vorname + Nachname, Referenznummer des Antrags, Hinweis dass die EEG den Antrag prüfen wird
- [ ] Die E-Mail ist auf Deutsch

### E-Mail an die EEG (Benachrichtigung)
- [ ] Nach erfolgreichem Einreichen wird eine Benachrichtigungs-E-Mail an die `contact_email` der zugehörigen EEG gesendet
- [ ] Die E-Mail enthält: Name des Antragstellers (Vorname + Nachname), E-Mail-Adresse des Antragstellers, Referenznummer, Liste der angemeldeten Zählpunkte, Hinweis zur Bearbeitung im Admin-Bereich
- [ ] Hat die EEG keine `contact_email` hinterlegt, wird keine EEG-Benachrichtigung gesendet (kein Fehler)
- [ ] Die E-Mail ist auf Deutsch

### Fehlerverhalten
- [ ] Schlägt der SMTP-Versand fehl, wird der Fehler serverseitig geloggt
- [ ] Der Einreichungs-Endpunkt gibt trotzdem `200 OK` zurück — E-Mail-Fehler blockieren die Einreichung nicht
- [ ] Beide E-Mails (Mitglied + EEG) werden unabhängig voneinander versendet: Scheitert eine, wird die andere trotzdem versucht

### Datenbankänderung
- [ ] Tabelle `member_onboarding.registration_entrypoint` erhält ein neues Feld `contact_email VARCHAR(255) NULL`
- [ ] Das Feld ist optional — bestehende Einträge ohne `contact_email` bleiben gültig
- [ ] Die Änderung erfolgt über eine neue Migration

### Konfiguration
- [ ] SMTP-Verbindungsparameter werden ausschließlich über Umgebungsvariablen konfiguriert: `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`
- [ ] Ist `SMTP_HOST` nicht gesetzt, wird der E-Mail-Versand vollständig übersprungen (kein Fehler beim Start)

## Edge Cases

- **EEG hat keine contact_email:** Nur die Mitglieds-Bestätigung wird gesendet, keine EEG-Benachrichtigung, kein Fehler
- **Mitglied hat keine gültige E-Mail:** Kann nicht vorkommen — E-Mail ist Pflichtfeld bei der Einreichung (PROJ-1 validiert dies)
- **SMTP nicht erreichbar:** Fehler wird geloggt, Einreichung wird nicht blockiert
- **Antrag wird erneut eingereicht** (Status `needs_info` → `submitted`): E-Mails werden erneut versendet, da das Mitglied aktiv nachgebessert hat
- **SMTP_HOST nicht konfiguriert:** E-Mail-Versand wird stillschweigend übersprungen — sinnvoll für lokale Entwicklung ohne Mail-Server

## Technical Requirements

- **Versand:** Go `net/smtp` oder leichtgewichtige Bibliothek (z. B. `github.com/wneessen/go-mail`), direkt per SMTP
- **Templates:** Go `html/template` + `embed.FS`, eingebettet ins Binary — kein Volume-Mount erforderlich
- **Template-Dateien:**
  - `internal/mail/templates/application_submitted_member.html`
  - `internal/mail/templates/application_submitted_eeg.html`
- **Paketstruktur:** `internal/mail/` (mailer.go + service.go + templates/)
- **Kein** Mail-Microservice, kein gRPC, kein externer Dienst
- **Mandantenspezifische Templates:** nicht in V1

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
