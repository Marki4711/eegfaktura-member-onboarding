# PROJ-41 & PROJ-43: Status-Change-Mails an Mitglied (Ablehnung + Info-Anfrage)

**Status:** In Review
**Created:** 2026-05-17

## Hintergrund

Bisher erfährt der Beitrittswerber nichts, wenn der EEG-Admin den Antrag
ablehnt oder Rückfragen stellt. Die Begründung landet nur im internen
`status_log`. Folge: Mitglied weiß nicht, dass der Antrag stillsteht.

PROJ-41 und PROJ-43 sind strukturell identisch (gleicher Mail-Trigger,
gleicher Admin-UI-Hinweis), darum als gemeinsame Spec.

## Anforderungen

### PROJ-41 — Ablehnungs-Mail

- Bei Statuswechsel **`* → rejected`** wird automatisch eine E-Mail an
  `application.email` versendet
- Inhalt: Antragsnummer, EEG-Name, **die vom Admin eingegebene Begründung**
  (1:1 in den Mail-Body), Reply-To = EEG-Kontakt-E-Mail
- Admin-UI: im Begründungs-Dialog für „Ablehnen" ein Hinweis-Block:
  > „Die hier eingegebene Begründung wird per E-Mail an den Beitrittswerber
  > übermittelt."

### PROJ-43 — Info-Anfrage-Mail

- Bei Statuswechsel **`* → needs_info`** wird automatisch eine E-Mail an
  `application.email` versendet
- Inhalt: Antragsnummer, EEG-Name, **die vom Admin eingegebene Rückfrage**
  (1:1 in den Mail-Body), Reply-To = EEG-Kontakt-E-Mail
- Admin-UI: im Begründungs-Dialog für „Info benötigt" ein Hinweis-Block:
  > „Der hier eingegebene Text wird per E-Mail an den Beitrittswerber
  > übermittelt."

## Implementation

### Backend

- **Mail-Templates** (neu, in `internal/mail/templates/`)
  - `application_rejected_member.html`
  - `application_needs_info_member.html`
- **MailService-Interface** (`internal/mail/service.go`) bekommt zwei neue
  Methoden:
  - `SendRejectedNotification(app, entrypoint, reason)`
  - `SendNeedsInfoNotification(app, entrypoint, reason)`
- **`NoOpMailService`** implementiert beide als No-Op (für Dev ohne SMTP)
- **`AdminApplicationService.ChangeStatus`** triggert nach erfolgreichem
  Commit asynchron die passende Mail — gleiches Goroutine-Pattern wie
  beim Approval-Mail (`acquireMailSem`/`releaseMailSem`, Best-Effort,
  Fehler werden geloggt aber blocken die Statusänderung nicht)
- **Metric**: `mail_sent_total` mit neuen Labels `member_rejection` und
  `member_needs_info` (success/failed)

### Frontend

- **`admin-status-actions.tsx`**: im `Dialog` für `dialogTarget = "rejected"`
  oder `"needs_info"` ein Hinweis-Block über dem Textarea — kontextsensitiv
  formuliert ("Ablehnungs-Begründung" / "Rückfrage").

### Vorbedingungen

- `application.email` muss gesetzt sein (ist im Pflicht-Set, sollte immer
  der Fall sein)
- Wenn SMTP nicht konfiguriert ist → NoOpMailService greift, kein Fehler
- Wenn EEG keine `contact_email` hat → Reply-To bleibt leer
  (`transactionalOpts("")` ist tolerant)

## Out of Scope

- Mehrfach-Round-Trip für `needs_info → submitted → needs_info`-Schleifen
  (jedes Mal löst eine neue Mail aus; das ist gewollt)
- Member-Antwort-Workflow innerhalb der App (für Antworten nutzt das
  Mitglied einfach Reply-To zur EEG)
- Bulk-Action-Variante: BulkChangeStatus geht erst mal **ohne** Mail-Trigger
  (kann später nachgezogen werden, wenn Bedarf da ist)

## Tests

- Build muss grün bleiben, `go test ./...`
- Manuelle Smoke-Tests nach Deploy:
  - Antrag ablehnen mit Test-Begründung → Mail kommt an, Begründung 1:1
    im Body
  - Antrag auf needs_info mit Test-Rückfrage → Mail kommt an
  - Im Admin-Dialog: Hinweis-Block sichtbar bei Ablehnen + Info-Anfrage,
    nicht beim Bestätigen (approved) oder „In Prüfung nehmen"
