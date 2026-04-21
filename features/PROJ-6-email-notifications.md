# PROJ-6: E-Mail-Benachrichtigungen bei Antragseinreichung

## Status: Deployed
**Created:** 2026-04-19
**Last Updated:** 2026-04-21

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

**Tested:** 2026-04-19
**Scope:** Pure backend — code review + Go unit tests (no browser testing; feature has no UI)
**Tester:** QA Engineer (AI)

### Acceptance Criteria Status

#### AC-1: Auslöser
- [x] E-Mails werden nur bei Ersteinreichung ausgelöst: `oldStatus == "draft"` check in `SubmitApplication()` (application_service.go:303)
- [x] Bei Wiedereinreichung (`needs_info → submitted`) werden keine E-Mails gesendet — `"needs_info"` does not match the draft condition

#### AC-2: E-Mail an das Mitglied (Bestätigung)
- [x] Wird ausgelöst wenn `fromStatus = draft` und `toStatus = submitted`
- [x] Empfänger: E-Mail-Adresse des Antragstellers (`app.Email`)
- [x] Enthält: Anrede mit Vorname + Nachname, Referenznummer — verified by unit tests
- [x] E-Mail ist auf Deutsch — verified by `TestMemberTemplate_IsGerman`

#### AC-3: E-Mail an die EEG (Benachrichtigung)
- [x] Wird ausgelöst wenn `fromStatus = draft` und `toStatus = submitted`
- [x] Benachrichtigungs-E-Mail an `contact_email` der EEG
- [x] Enthält: Name, E-Mail, Referenznummer, Zählpunkte — verified by unit tests
- [x] Hat die EEG keine `contact_email`, wird keine Benachrichtigung gesendet (kein Fehler) — checked at service.go:80
- [x] E-Mail ist auf Deutsch — verified by `TestEEGTemplate_IsGerman`
- [x] Beide E-Mails werden unabhängig versendet: Scheitert Mitglieds-E-Mail, wird EEG-E-Mail trotzdem versucht (if/else für member, dann unabhängiger EEG-Block)

#### AC-4: Fehlerverhalten
- [x] SMTP-Fehler wird serverseitig geloggt (`log.Printf` in service.go)
- [x] Einreichungs-Endpunkt gibt trotzdem `200 OK` zurück — goroutine via `go s.mailService.SendSubmissionEmails(...)`, Fehler werden nicht weitergegeben
- [x] Beide E-Mails werden unabhängig versucht

#### AC-5: Datenbankänderung
- [x] `contact_email VARCHAR(255) NULL` in `registration_entrypoint` — Migration 000003 korrekt
- [x] Feld ist optional, bestehende Einträge bleiben gültig
- [x] Änderung über neue Migration

#### AC-6: Konfiguration
- [x] SMTP-Parameter ausschließlich über Umgebungsvariablen: `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`
- [x] Ist `SMTP_HOST` nicht gesetzt, wird E-Mail-Versand übersprungen (`NoOpMailService`)

### Edge Cases Status

#### EC-1: EEG hat keine contact_email
- [x] Nur Mitglieds-Bestätigung gesendet, keine EEG-Benachrichtigung, kein Fehler

#### EC-2: SMTP nicht erreichbar
- [x] Fehler wird geloggt, Einreichung nicht blockiert (asynchrone Goroutine)

#### EC-3: Antrag wird erneut eingereicht (needs_info → submitted)
- [x] Keine E-Mails — `oldStatus == "draft"` ist nicht erfüllt

#### EC-4: SMTP_HOST nicht konfiguriert
- [x] `NoOpMailService` verwendet, kein Fehler beim Start

### Security Audit Results
- [x] `contact_email` nicht in öffentlicher API-Antwort exponiert (`RegistrationConfig` hat eigenes Struct ohne `ContactEmail`)
- [x] XSS-Schutz: `html/template` escapet automatisch — verified by `TestEEGTemplate_XSSEscaped` und `TestMemberTemplate_XSSEscaped`
- [x] SMTP-Credentials nur in Env-Variablen, nie geloggt (nur `SMTP_HOST` wird beim Start geloggt)
- [x] E-Mail-Adresse des Mitglieds durch PROJ-1 validiert (Pflichtfeld mit `validate:"required,email"`)
- [x] E-Mail-Header-Injection: Subject via `fmt.Sprintf` mit User-Daten — `go-mail` codiert Subject-Header korrekt (RFC 2047), kein Injektionsrisiko

### Bugs Found

#### ~~BUG-1: SMTPAuthPlain immer gesetzt, auch ohne Credentials~~ — BEHOBEN
- `WithSMTPAuth` wird nur gesetzt wenn `m.user != ""` (`internal/mail/mailer.go`)

#### ~~BUG-2: SMTP_FROM nicht beim Start validiert~~ — BEHOBEN
- `log.Fatalf` wenn `SMTP_HOST` gesetzt aber `SMTP_FROM` leer (`cmd/server/main.go`)

#### ~~BUG-3: Mailer nicht hinter Interface~~ — BEHOBEN
- `Sender`-Interface extrahiert; `SMTPMailService.sender Sender`; 4 neue Unit Tests mit `spySender`

#### ~~BUG-4: SMTP Env-Variablen fehlen in `.env.local.example`~~ — BEHOBEN
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM` dokumentiert

#### BUG-5: `fmt.Printf` statt `log.Printf` bei Entrypoint-Lookup-Fehler
- **Severity:** Low
- **Steps to Reproduce:** Entrypoint-Lookup schlägt fehl (z.B. RC-Nummer nicht mehr in DB)
- **Expected:** Strukturierter Log-Eintrag wie überall sonst
- **Actual:** `fmt.Printf` (kein Timestamp, kein Log-Level)
- **Ort:** `internal/application/application_service.go:306`
- **Priority:** Fix in next sprint

#### BUG-6: Dev-Seed ohne contact_email — EEG-Benachrichtigungspfad nicht testbar
- **Severity:** Low
- **Steps to Reproduce:** Lokale Entwicklung, `dev_seed.sql` ausführen, Antrag einreichen
- **Expected:** EEG-Benachrichtigungspfad testbar
- **Actual:** `contact_email IS NULL` → EEG-E-Mail wird nie versucht, kein Test möglich ohne manuelle DB-Änderung
- **Ort:** `db/seeds/dev_seed.sql`
- **Priority:** Nice to have

### Unit Tests
8 Go-Unit-Tests in `internal/mail/service_test.go`:
- `TestNoOpMailService_Noop` — Interface-Compliance, kein Panic
- `TestNewSMTPMailService_ParsesTemplates` — Template-Parsing erfolgreich
- `TestMemberTemplate_ContainsExpectedFields` — Pflichtfelder in Ausgabe vorhanden
- `TestMemberTemplate_IsGerman` — Sprache korrekt
- `TestEEGTemplate_ContainsExpectedFields` — Name, E-Mail, Referenz, Zählpunkte
- `TestEEGTemplate_IsGerman` — Sprache korrekt
- `TestEEGTemplate_XSSEscaped` — Script-Tag wird escaped
- `TestMemberTemplate_XSSEscaped` — Script-Tag wird escaped

Alle 8 Tests bestanden. `go vet` sauber.

### Summary
- **Acceptance Criteria:** 16/16 bestanden
- **Bugs Found:** 6 total (0 critical, 0 high, 3 medium, 3 low) — BUG-1 bis BUG-4 behoben
- **Security:** Pass (XSS-sicher, keine API-Datenlecks, Credentials sicher)
- **Production Ready:** YES — keine Critical/High Bugs verbleibend; BUG-5 + BUG-6 (Low) können folgen
- **Recommendation:** Deploy

## Deployment

**Deployed:** 2026-04-21 (vollständig inkl. SMTP-Konfiguration und Resend-Funktion)
**Image tag:** `latest` (built by GitHub Actions on push to main)

### Produktiv verifizierte SMTP-Konfiguration

Der Cluster hat keinen Zugriff auf Port 587 (von Firewall geblockt). **Port 25** verwenden:

```yaml
# values-env.yaml
backend:
  smtp:
    host: atvipostal.vfeeg.org
    port: "25"
    user: <credential-name-aus-postal>
    from: noreply@eegfaktura.at

# values-secret.yaml
secrets:
  smtpPassword: "<key-aus-postal-credentials>"
```

Postal verwendet den Credential-Namen als SMTP-Login und den Key als Passwort.

**Startup-Verifikation:** Nach `helm upgrade` muss der Backend-Log zeigen:
```json
{"level":"INFO","msg":"mail service enabled","smtp_host":"atvipostal.vfeeg.org"}
```

Ohne `SMTP_HOST` startet der Server mit `NoOpMailService` — keine E-Mails, kein Fehler.

### Zusätzliche Features (nach initialer Deployment-Doku ergänzt)

- **Resend-Funktion:** Admin-Detailansicht enthält Button "Bestätigung erneut senden" (`POST /api/admin/applications/{id}/resend-confirmation`). Sendet nur die Mitglieds-Bestätigungsmail, unabhängig vom aktuellen Antragsstatus.
- **EEG-Benachrichtigung:** Wird automatisch übersprungen wenn `contact_email` in `registration_entrypoint` NULL ist — kein manueller Eingriff nötig.
- **SMTP-Timeout:** 10 Sekunden (verhindert lange Request-Hänger bei falschem Host).

### Production Fixes (discovered during live testing 2026-04-21)

| # | Issue | Root Cause | Fix |
|---|-------|-----------|-----|
| P-1 | Port 587 geblockt → 15s Timeout + 500/502 | Cluster-Firewall blockiert Port 587 ausgehend | Port 25 verwenden |
| P-2 | `adminRequest` wirft Fehler bei 204 No Content | `res.json()` schlägt auf leerem Body fehl | 204-Check vor `res.json()` in `adminRequest` |
| P-3 | 15s Hänger bei falschem SMTP-Host | Kein Timeout im Mailer konfiguriert | `gomail.WithTimeout(10 * time.Second)` |
| P-4 | Health-Endpoint flutet Backend-Log | `/health` wurde wie normale Requests geloggt | `/health`-Requests im `SlogRequestLogger` überspringen |
