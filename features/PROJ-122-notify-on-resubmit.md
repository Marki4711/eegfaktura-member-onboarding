# PROJ-122 — Benachrichtigung beim Wieder-Einreichen nach Rückfrage

**Status:** In Review
**Owner-Anfrage:** 2026-07-01 (Follow-up zu PROJ-121: „bekommt die EEG eine Info,
wenn der Antrag aktualisiert wurde?")

## Problem

Mit PROJ-121 kann ein Mitglied seinen Antrag nach einer Rückfrage
(`needs_info`) selbst korrigieren und erneut einreichen (`needs_info →
submitted`). Bisher wurde die **Verständigungskette nur beim Erst-Einreichen**
(`draft → submitted`) ausgelöst — beim Wieder-Einreichen ging **keine** Mail
raus: weder die Übersicht ans Mitglied noch eine Info an die EEG. Der Rückweg
in die Prüfung war stumm; die EEG hätte die Aktualisierung nur durch Blick in
die Antragsliste bemerkt.

## Ziel

Beim Wieder-Einreichen nach Rückfrage läuft **dieselbe Verständigungskette wie
beim Erstantrag**:
- Das **Mitglied** bekommt erneut die Bestätigungs-Mail mit der **Übersicht der
  übermittelten Werte** (inkl. aktualisiertem SEPA-Mandat-PDF, falls zutreffend).
- Die **EEG** wird über die Aktualisierung **informiert** (dieselbe EEG-
  Benachrichtigung wie beim Erstantrag).

## Akzeptanzkriterien

- **AC-1** `needs_info → submitted` löst `SendSubmissionEmails` aus (Mitglied-
  Bestätigung + EEG-Benachrichtigung) — verifiziert per Backend-Log
  (`mail: sending submission emails`). ✔
- **AC-2** Beim Wieder-Einreichen wird **kein** neuer E-Mail-Bestätigungs-Token
  erzeugt (das Mitglied ist bereits bestätigt) → die EEG-Info geht **sofort**
  raus, nicht deferred (`confirmation_pending:false`). ✔
- **AC-3** Der Submit-Zähler (`ApplicationsSubmittedTotal`) zählt weiterhin nur
  den echten Erst-Submit (`draft → submitted`), nicht das Wieder-Einreichen. ✔
- **AC-4** Erst-Einreichen (`draft → submitted`) bleibt unverändert (inkl.
  E-Mail-Bestätigungs-Flow bei `require_email_confirmation`). ✔

## Edge Cases

- **EC-1** `require_email_confirmation=TRUE`: Um überhaupt in `needs_info` zu
  landen, muss das Mitglied bereits bestätigt haben (Bestand-Regel: `submitted →
  under_review/needs_info` ist bis zur Bestätigung gesperrt). Daher ist beim
  Wieder-Einreichen keine erneute Bestätigung nötig — Token-Minting ist auf
  `draft` gegated. ✔
- **EC-2** EEG ohne `contact_email`: EEG-Benachrichtigung wird übersprungen
  (`skipping EEG notification (no contact_email)`) — wie beim Erstantrag. ✔
- **EC-3** SMTP nicht erreichbar: best-effort (Goroutine), der Re-Submit selbst
  schlägt NICHT fehl (member-getriggert — darf nicht an SMTP hängen). ✔

## Tech Design

Minimal-invasiv in `ApplicationService.SubmitApplication`:
1. E-Mail-Bestätigungs-Token nur bei `oldStatus == draft` minten (vorher: immer
   bei `require_email_confirmation`). So bleibt `emailConfirmationURL` beim
   Wieder-Einreichen leer → `SendSubmissionEmails` deferred die EEG-Info nicht.
2. Der bestehende Mail-Block feuert jetzt bei `draft || needs_info` (vorher nur
   `draft`); der Submit-Zähler bleibt auf `draft` beschränkt.

Kein neues Template, keine neue Mail-Methode, kein Schema-Change, kein neuer
Endpoint — reine Wiederverwendung der Erst-Submit-Kette.

## Verifikation

- go build/vet + `go test ./internal/application/ ./internal/mail/` grün.
- **E2E-Smoke** (lokaler Stack, SMTP bewusst unerreichbar): Re-Submit eines
  `needs_info`-Antrags → Log `mail: sending submission emails
  confirmation_pending:false` + Dispatch von Mitglied-Bestätigung UND
  EEG-Benachrichtigung (Send-Fehler nur wegen Dummy-SMTP — Pfad bewiesen).

## Security

Kein neuer Endpoint, keine PII-Exposure-Änderung, kein Schema-/Auth-/Tenant-
Change. Mail-Amplification vernachlässigbar: Re-Submit ist rate-limitiert
(`PublicSubmitRateLimitMiddleware`, 10/10min/IP) und **self-limiting** — nach
dem Submit ist der Status `submitted` (nicht mehr `needs_info`), ein erneutes
Einreichen ist erst möglich, nachdem die EEG wieder „Info anfordern" klickt.
→ Kein dediziertes `/security-review` nötig (nur Benachrichtigung auf einer
bestehenden Transition; die Transition-Regeln selbst bleiben unverändert).

## Dependencies

- Baut auf PROJ-121 (Mitglieder-Bearbeitungslink) auf; deployt gemeinsam.
