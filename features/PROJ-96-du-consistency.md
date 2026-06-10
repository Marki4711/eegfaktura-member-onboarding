# PROJ-96: Du-Konsistenz in allen Mail-Pfaden

## Status: Approved
**Created:** 2026-06-10
**Last Updated:** 2026-06-10
**Typ:** Hotfix (Tester-Befund TODO-4)

## Hintergrund

Owner-Direktive 2026-06-10: „wir haben ja festgelegt, dass überall Du
verwendet wird." Tester-Befund TODO-4 aus dem Vorabend-Memo war kein
Owner-Klärungs-Fall, sondern ein konkreter Bug.

Tester (sinngemäß): „Beim Firmen-SEPA auf Sie zu wechseln, wirkt
komisch — kommt ja vorher die erste Mail per Du …"

## Befund (Code-Audit 2026-06-10 vor Fix)

Greppen über alle deutschen Sie-Formen (`Sie|Ihre|Ihr|Ihnen|Ihren|Ihres`):

- **`internal/mail/templates/application_*.html`**: alle 8 Mitglieder-
  Templates → **0 Treffer** (bereits sauber)
- **`internal/mail/b2b_notice.go:37`**: PROJ-91-B2B-Vorbereitungs-Banner
  Member-Variante → „bei **Ihrer** Hausbank … geben **Sie** der EEG kurz
  Bescheid" (das war die Tester-konkrete Stelle)
- **`internal/mail/customer_onboarding.go:70` + `:100`**: Subjects der
  EEG-Customer-Onboarding-Welcome- und Reject-Mail
- **`internal/mail/templates/customer_onboarding_welcome.html`**: „Guten
  Tag", „bei Ihnen", „können Sie", „erhalten Sie", „wenden Sie sich"
- **`internal/mail/templates/customer_onboarding_reject.html`**: „Guten
  Tag", „Ihre Anmeldung", „Ihnen", „Sie sich"
- **`internal/mail/service.go:1209-1229`**: Vorstands-Beitrittserklärungs-
  Mail (`SendBoardApprovalRequest`) — „Sehr geehrtes Vorstandsmitglied",
  „Im Anhang finden Sie", „unterzeichnen Sie", „Mit freundlichen Grüßen"

## Fixes

### Banner-Helper (`b2b_notice.go`)

- „Ihrer Hausbank" → „deiner Hausbank"
- „geben Sie der EEG kurz Bescheid" → „gib der EEG kurz Bescheid"

### EEG-Customer-Onboarding-Mails

`customer_onboarding.go`:
- Welcome-Subject: „Ihre EEG ist aktiviert" → „deine EEG ist aktiviert"
- Reject-Subject: „Ihre Anmeldung wurde abgelehnt" → „Deine Anmeldung
  wurde abgelehnt"

`customer_onboarding_welcome.html`:
- Title + Body: „Ihre EEG" → „deine EEG"
- „Guten Tag" → „Hallo"
- „meldet sich … bei Ihnen" → „bei dir"
- „können Sie unter Einstellungen pflegen" → „kannst du … pflegen"
- „erhalten Sie einen Link" → „erhältst du einen Link"
- „wenden Sie sich gerne" → „wende dich gerne"

`customer_onboarding_reject.html`:
- „Guten Tag" → „Hallo"
- „Ihre Anmeldung der EEG" → „Deine Anmeldung der EEG"
- „Es entstehen Ihnen keine Kosten" → „Es entstehen dir keine Kosten"
- „damit Sie die Situation klären können" → „damit du die Situation
  klären kannst"
- „Mitglieder-Submit auf Ihrer öffentlichen Seite" → „auf deiner
  öffentlichen Seite"
- „wenden Sie sich" → „wende dich"

### Vorstands-Beitrittserklärungs-Mail (`service.go:1209`)

- „Sehr geehrtes Vorstandsmitglied," → „Hallo,"
- „Im Anhang finden Sie die Beitrittserklärung … unterzeichnen Sie
  das Dokument und leiten Sie es an das Mitglied weiter." → „Im Anhang
  findest du die Beitrittserklärung … unterschreibe das Dokument und
  leite es an das Mitglied weiter."
- „Mit freundlichen Grüßen" → „Liebe Grüße"

## Acceptance Criteria

- [x] **AC-1** Grep über `internal/mail/` nach `\b(Sie|Ihre|Ihr|Ihnen|Ihren|Ihres)\b`:
  0 Treffer in Code + Templates
- [x] **AC-2** `go build ./...` clean
- [x] **AC-3** `go test ./...` clean (Mail-Tests verifizieren die
  Wortlaute nicht hartkodiert — Helper-Funktionen werden getestet,
  Templates rendern als Smoke-Test)
- [x] **AC-4** CHANGELOG.md-Eintrag im selben Commit

## Edge Cases

- **EC-1** EEG-Customer-Onboarding-Welcome ohne AVV-PDF: Pfadkommt
  durch, Du-Form unverändert.
- **EC-2** Vorstands-Mail im Re-Aktivierungs-Pfad (`isReActivation=true`):
  beide Zweige (regulär + re-Aktivierung) sind im Du.
- **EC-3** PROJ-91-Banner bei Toggle=false: leerer String, kein
  Sie/Du-Konflikt.

## Out of Scope

- TODO-3 (Block-Reihenfolge in der Bestätigungsmail) — Tester-
  Screenshot steht weiterhin aus.

## Deployment

**Deploy-Bookkeeping 2026-06-10 (vormittags):**

- Hotfix-Cycle: direkter Commit, kein eigener /architecture-Pfad
- Helm-Bump folgt nach CI grün
- Tag: `v1.25.1-PROJ-96`

**Tester-Verifikation:** B2B-Vorbereitungs-Mail an einen Test-Antrag,
EEG-Customer-Onboarding-Approval-Flow, Vorstands-Modus für
Beitrittserklärung — alle Du-Form, keine „Sehr geehrtes …"-Stelle mehr.
