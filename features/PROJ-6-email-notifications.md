# PROJ-6: E-Mail-Benachrichtigungen bei Antragseinreichung

## Status: Planned
**Created:** 2026-04-19
**Last Updated:** 2026-04-19 (Auslöser auf Ersteinreichung eingeschränkt)

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

### Auslöser
- [ ] E-Mails werden **nur bei der Ersteinreichung** ausgelöst: wenn der Antrag vom Status `draft` in `submitted` wechselt
- [ ] Bei Wiedereinreichung (`needs_info` → `submitted`) werden **keine** E-Mails gesendet — das Mitglied korrigiert einen bestehenden Antrag und erhält bereits eine Bildschirm-Rückmeldung

### E-Mail an das Mitglied (Bestätigung)
- [ ] Wird ausgelöst wenn `fromStatus = draft` und `toStatus = submitted`
- [ ] Empfänger: E-Mail-Adresse des Antragstellers
- [ ] Die E-Mail enthält: Anrede mit Vorname + Nachname, Referenznummer des Antrags, Hinweis dass die EEG den Antrag prüfen wird
- [ ] Die E-Mail ist auf Deutsch

### E-Mail an die EEG (Benachrichtigung)
- [ ] Wird ausgelöst wenn `fromStatus = draft` und `toStatus = submitted`
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
- **Antrag wird erneut eingereicht** (Status `needs_info` → `submitted`): Keine E-Mails — der Übergang `fromStatus = needs_info` ist kein Ersteinreichungs-Ereignis. Das Mitglied erhält die Bestätigung bereits auf dem Bildschirm.
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

### Betroffene Komponenten

Rein Backend — keine Frontend-Änderungen.

```
internal/
  config/config.go                            ← erweitert: SMTPConfig
  shared/models.go                            ← erweitert: ContactEmail in RegistrationEntrypoint
  application/
    registration_entrypoint_repo.go           ← erweitert: contact_email aus DB lesen
    application_service.go                    ← erweitert: MailService injizieren + aufrufen
  mail/                                       ← neu
    mailer.go                                 ← SMTP-Verbindung und Versand
    service.go                                ← Template-Rendering, Entscheidungslogik
    templates/
      application_submitted_member.html       ← neu
      application_submitted_eeg.html          ← neu
db/migrations/
  000003_add_contact_email_…up.sql            ← neu
  000003_add_contact_email_…down.sql          ← neu
```

### Datenmodell-Erweiterung

`registration_entrypoint` erhält ein neues optionales Feld:

| Feld | Typ | Pflicht |
|------|-----|---------|
| `contact_email` | VARCHAR(255) | nein (NULL erlaubt) |

Bestehende Einträge ohne Wert bleiben unverändert gültig.

### Auslöse-Logik

Der Auslöser sitzt in `SubmitApplication()` in `application_service.go`. Die Methode kennt bereits `oldStatus`. Der MailService wird nur aufgerufen wenn `oldStatus == "draft"`:

```
POST /api/public/applications/{id}/submit
  → SubmitApplication()
      → Status-Übergang wird durchgeführt
      → oldStatus == "draft"?
          JA  → MailService.SendSubmissionEmails(application, meteringPoints, entrypoint)
                    → Bestätigung an application.Email (immer)
                    → Benachrichtigung an entrypoint.ContactEmail (nur wenn gesetzt)
                    → Fehler: loggen, nicht an den Aufrufer weitergeben
          NEIN → keine E-Mail (Wiedereinreichung nach needs_info)
      → Response wie bisher
```

### Entkopplung über Interface

`MailService` wird als Interface definiert und in `ApplicationService` injiziert. Für lokale Entwicklung ohne SMTP-Konfiguration wird eine No-Op-Implementierung verwendet — kein Absturz, keine Fehlermeldung.

### Konfiguration

Fünf neue Umgebungsvariablen in `config.go`:

| Variable | Bedeutung | Pflicht |
|----------|-----------|---------|
| `SMTP_HOST` | SMTP-Server-Adresse | Ja — fehlt: Versand deaktiviert |
| `SMTP_PORT` | Port (Standard: 587) | Nein |
| `SMTP_USER` | Login-Benutzername | Nein |
| `SMTP_PASSWORD` | Login-Passwort | Nein |
| `SMTP_FROM` | Absenderadresse | Ja (wenn SMTP_HOST gesetzt) |

### Templates

HTML-Templates werden per Go `embed.FS` direkt ins Binary eingebettet — kein Volume-Mount, keine externen Dateien.

| Template | Empfänger | Template-Variablen |
|----------|-----------|-------------------|
| `application_submitted_member.html` | Antragsteller | Firstname, Lastname, ReferenceNumber |
| `application_submitted_eeg.html` | EEG | Firstname, Lastname, Email, ReferenceNumber, MeteringPoints |

### Neue Abhängigkeit

`github.com/wneessen/go-mail` — leichtgewichtige Go-Bibliothek für SMTP mit HTML-Mail-Unterstützung. Kein gRPC, kein Microservice, keine weitere externe Abhängigkeit.

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
