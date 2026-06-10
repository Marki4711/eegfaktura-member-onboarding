# PROJ-98: Anrede-Konditional je Mitgliedstyp

## Status: Deployed (2026-06-10)
**Created:** 2026-06-10
**Last Updated:** 2026-06-10
**Typ:** Mail-Wortlaut-Refactor (Owner-Korrektur)

## Hintergrund

PROJ-96 hatte die „überall Du"-Direktive 1:1 durchgezogen. Tester-
Befund kurz darauf:

> „die kürzlich durchgeführte anpassung des tons auf durchgängig du
> ging wohl zu weit. Bei den Gemeinden, Unternehmen und Vorstand soll
> eine förmlichere Anrede verwendet werden."

Owner-Klärung 2026-06-10 via AskUserQuestion:

| Bereich | Wahl |
|---|---|
| Mitglieder-Mails | Sie nur bei `company` + `municipality` (Vereine bleiben Du) |
| EEG-Customer-Onboarding-Mails | Zurück auf Sie (formaler B2B-Kontext) |
| PROJ-91 B2B-Vorbereitungs-Banner | Konditional je Mitgliedstyp |

Vereine (`association`) bleiben bewusst beim Du — Owner-Entscheidung.
Privat + Pauschalierter Landwirt sowieso Du.

## Implementation

### Backend

**`internal/mail/service.go`:**

- Neuer Template-Helper `anrede(useFormal, firstname, lastname)`:
  - `useFormal=true` → „Sehr geehrte Damen und Herren"
  - `useFormal=false` → „Hallo {Name}" (greetingName-Logik, mit
    Fallback „Hallo zusammen")
- Neue Funktion `memberUsesFormalAddress(memberType)`: single source
  of truth — `company`/`municipality` → true.
- `memberTemplateData`, `statusChangeTemplateData`, `mandateAtImportData`,
  `activationTemplateData` bekommen `UseFormal bool`.
- Alle vier Build-Funktionen (`buildMemberMailData`,
  `buildStatusChangeData`, `buildMandateAtImportData`,
  `buildActivationData`) setzen es via `memberUsesFormalAddress`.

**`internal/mail/b2b_notice.go`:**

- `RenderB2BPrepareNoticeBannerMember` neue Signatur
  `(prepare, useFormal bool)`. Sie-Variante:
  „Ihrer Hausbank … geben Sie der EEG kurz Bescheid".
  Du-Variante:
  „deiner Hausbank … gib der EEG kurz Bescheid".
- EEG-Banner (`RenderB2BPrepareNoticeBannerEEG`) bleibt unverändert —
  Empfänger ist immer die EEG (Vorstand/Ablage), nicht das Mitglied.

**`internal/mail/service.go` PROJ-96-Teilrevert:**

- `SendBoardApprovalRequest` Vorstands-Mail-Body wieder auf
  „Sehr geehrtes Vorstandsmitglied" + „Im Anhang finden Sie …
  unterzeichnen Sie … Mit freundlichen Grüßen".

**`internal/mail/customer_onboarding.go` PROJ-96-Teilrevert:**

- Welcome-Subject „Ihre EEG ist aktiviert", Reject-Subject „Ihre
  Anmeldung wurde abgelehnt".

### Templates

**5 Mitglieder-Templates** unifiziert auf
`{{anrede .UseFormal .Firstname .Lastname}},`:

- `application_submitted_member.html` — 14 Du-Forms (lang) inline
  als `{{if .UseFormal}}Sie-Block{{else}}Du-Block{{end}}` umgesetzt
- `application_imported_member.html` — 25 Du-Forms (längstes
  Template, 4 SEPA-Mandat-Varianten je SEPA-Typ + Audit-Mode);
  je Variante eigenes Sie/Du-Branching
- `application_activated_member.html` — 9 Du-Forms
- `application_needs_info_member.html` — 8 Du-Forms
- `application_rejected_member.html` — 5 Du-Forms

**EEG-Customer-Onboarding-Templates** (PROJ-96-Teilrevert):

- `customer_onboarding_welcome.html` — Title + Body wieder auf Sie,
  „Hallo" → „Guten Tag"
- `customer_onboarding_reject.html` — analog

### Tests

`b2b_notice_test.go`:

- `TestRenderB2BPrepareNoticeBannerMember` 4-Permutation
  (prepare × useFormal)
- Neuer Test `TestRenderB2BPrepareNoticeBannerMember_FormalDuSwitch`
  prüft, dass Du-Variante „deiner Hausbank" + „gib der EEG" enthält
  und Sie-Variante „Ihrer Hausbank" + „geben Sie der EEG".

## Acceptance Criteria

- [x] **AC-1** `memberUsesFormalAddress` als single source of truth,
  3 Mitgliedstypen → false (private, farmer, association),
  2 Mitgliedstypen → true (company, municipality).
- [x] **AC-2** Alle 5 Mitglieder-Templates rendern Anrede via
  `anrede`-Helper + Body je Sie/Du-Branch.
- [x] **AC-3** PROJ-91-Banner-Helper Member-Variante akzeptiert
  `useFormal` und schaltet Wortlaut um. EEG-Variante unverändert.
- [x] **AC-4** PROJ-96 für Vorstands-Mail + EEG-Customer-Onboarding-
  Mails teilrevertiert: Sie-Anrede zurück.
- [x] **AC-5** `go build ./...` clean, `go test ./...` clean.
- [x] **AC-6** CHANGELOG.md + INDEX.md-Eintrag im selben Commit.

## Edge Cases

- **EC-1** Bestand-Anträge mit company-MemberType: nächste Mail
  rendert Sie. Frühere bereits verschickte Mails bleiben unverändert
  (idempotente Senders).
- **EC-2** B2B-Vorbereitung bei association-Mitglied: Banner rendert
  Du-Variante. Korrekt, weil association nicht in der Sie-Gruppe.
- **EC-3** `greetingName`-Helper bleibt im Code als Fallback für
  ältere Templates / mögliche Re-Use — wird aber von keinem Template
  mehr referenziert.
- **EC-4** Firmen-Mandat ohne Firstname+Lastname: `anrede(true, "", "")`
  → „Sehr geehrte Damen und Herren". Anrede ist namensagnostisch.

## Out of Scope

- TODO-3 (Mail-Block-Reihenfolge in Bestätigungs-Mail) — Tester-
  Screenshot weiterhin pending.

## Deployment

**Deploy-Bookkeeping 2026-06-10 (nachmittags):**

- Mail-Wortlaut-Refactor; kein Schema-Change, keine Helm-Änderung.
- Code-Commit: `ac60017`
- Helm-Bump-Commit: `5955e9a` (sha-ac60017)
- Tag: `v1.26.0-PROJ-98` gesetzt + gepusht (Minor — Mail-Template-
  Struktur-Refactor + Banner-Helper-Signatur-Erweiterung).

**Tester-Verifikation:** ein company-Mitglied importieren →
Mandat-Mail mit „Sehr geehrte Damen und Herren," + Sie-Forms im
Body. Privatmitglied → „Hallo Max Mustermann," + Du-Forms wie
gewohnt. Vorstands-Mail bleibt unabhängig vom Mitgliedstyp auf Sie.
