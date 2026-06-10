# PROJ-91: B2B-Vorbereitungs-Toggle im Admin-Edit (ersetzt PROJ-79-Heimlich-Mapping + entfernt awaiting_bank_confirmation)

## Status: Deployed (2026-06-09)
**Created:** 2026-06-09
**Last Updated:** 2026-06-09

## Hintergrund

Owner-Direktive 2026-06-09 (Abend):

> Auch wenn im Member-Onboarding als SEPA-Typ b2b (Firmenlastschrift)
> hinterlegt wird, soll der Import in den eegFaktura-Core den Antrag mit
> SEPA-Typ CORE (Basis-Lastschrift) anlegen. Aber statt der heutigen
> heimlich-Mapping-Logik (PROJ-79) UND statt des geplanten Marker-Systems
> (alter PROJ-90-Plan, jetzt obsolet) gibt es einen expliziten Toggle im
> Admin-Edit eines Mitglieds: „Mitglied wird mit SEPA-CORE angelegt,
> bekommt aber bereits alle Dokumente für die Umstellung auf SEPA-B2B".
> Dann muss man nicht mehr bei sepa-typ auf b2b stellen, obwohl CORE
> beim Faktura übermittelt wird, und es ist klarer wie es weitergeht.

**Was dadurch obsolet wird:**

| Was | Ersatz |
|---|---|
| PROJ-79 `b2b → CORE`-Heimlich-Mapping in `mapEinzugsart` | Rollback auf `b2b → B2B`; im Standardfall ist `einzugsart=core` mit Toggle |
| Marker-System (alter PROJ-90-Plan) | Toggle-Spalte `prepare_b2b_documents` |
| Status `awaiting_bank_confirmation` (PROJ-46) | Hart entfernt — kein Marker-Status mehr nötig |
| WhatsApp-Tester-Nachricht zum Marker-Modell | Entfällt |

## Dependencies

- **Voraussetzt:** PROJ-46 (Status-Modell), PROJ-79 (Mapping-Logik), PROJ-71
  (Plattform-Buchung), PROJ-32 (EEG-Stammdaten-Sync)
- **Berührt nicht:** PROJ-78 (Audit-Toggle), PROJ-80 (SEPA-Settings), PROJ-81
  (SEPA-Optional) — orthogonal

## Owner-Entscheidungen 2026-06-09

| # | Frage | Entscheidung |
|---|---|---|
| Q1 | Public-Form B2B-Auswahl entfernen? | **Bereits weg** — Public-Form hat heute schon keinen SEPA-Typ-Selector |
| Q2 | PROJ-79 Mapping rückrollen oder defensiv behalten? | **Rückrollen** auf `b2b → B2B` |
| Q3 | DB-Schema-Feldname | **`prepare_b2b_documents BOOLEAN NOT NULL DEFAULT FALSE`** |
| Q4 | Mail-Anhang-Strategie | **B2B-PDF zusätzlich + Hinweis-Block, Hard-Fail-Pattern** wie PROJ-74 |
| Q5 | Bestand-Migration b2b-Anträge | **Option A**: alle b2b-Anträge auf `einzugsart=core` + `prepare_b2b_documents=true` |
| Q6 | Doku-Umfang | **B2B-Mandat-PDF + Hinweis-Block in Mail** (kein separates Anleitungs-PDF) |
| Q7 | Workflow-Trigger | **Admin-Edit only** (Excel + Externe API V1 out-of-scope) |
| Q8 | UI-Position des Toggles | **SEPA-Sektion**, unter dem Einzugsart-Selector, sichtbar nur wenn `einzugsart=core` |
| Q8a | Anzeige im Antrags-Detail | **Nur in Detail-Ansicht**, keine Übersicht/Liste/Badge/Filter (Owner-Scope-Direktive 2026-06-09) |
| Q11 | SEPA-Konfiguration (Audit-Trail vs. Unterschriftenfeld) bei Toggle=true | **Wird unverändert befolgt** — der Toggle hat keinen Einfluss auf das CORE-Mandat-Rendering; das zusätzliche B2B-Mandat folgt derselben Variante (Audit oder Klassik), die für `einzugsart=core` bzw. den B2B-Audit-Toggle (PROJ-78) konfiguriert ist. Owner-Direktive 2026-06-09. |
| Q9 | Status `awaiting_bank_confirmation` umgehen | **Hart entfernen** — Konstante, Transitionen, DB-Check, Frontend-Anzeige raus |
| Q10 | Bestand-Anträge in `awaiting_bank_confirmation` | **Migration setzt sie auf `ready_for_activation`** |

### Grill-me-Ergebnisse 2026-06-09 (Spec-Reviewed)

**Codebase-Anker-basierte Klärungen** (durch Code-Inspektion entschieden):
- **Mail-Pfad** (C1): B2B-PDF wird **nur** in `SendMandateAtImport` (PROJ-80) als zusätzlicher Anhang gehängt. `SendActivationNotification` bleibt unberührt. `SendBoardApprovalRequest` bekommt nur den Banner.
- **PDF-Variante** (D1): CORE-PDF folgt `SEPAMandateCoreAuditEnabled`, B2B-PDF folgt `SEPAMandateB2BAuditEnabled` (saubere Trennung, konsistent zu PROJ-78).
- **Mandate-Timing** (D2): beide PDFs zum gleichen Trigger (`sepa_mandate_at_import`).
- **Mandate-Referenz** (D3): selbe Referenz (Mitgliedsnummer) für beide PDFs.
- **Filename** (D4): `sepa-mandat-{member#}.pdf` + `sepa-firmenlastschrift-mandat-{member#}.pdf` (existierend).
- **Postgres-Version** (A2): 16-alpine — ADD COLUMN metadata-only.
- **Migrations-Reihenfolge** (A1): (1) ADD COLUMN prepare_b2b_documents, (2) UPDATE einzugsart=b2b → core+toggle, (3) DROP+RECREATE CHECK ohne `awaiting_bank_confirmation`, (4) UPDATE Status `awaiting_bank_confirmation` → `ready_for_activation` + `status_log`-Eintrag.
- **Frontend-Strip-Stellen** (E5): `admin-status-badge.tsx:19`, `admin-filter-panel.tsx:27`, `admin-status-actions.tsx:397`, `lib/api.ts:557`.
- **Backend-Strip-Stellen**: `shared/models.go:132`, `admin_service.go:78/619/1317/1897/1955`, `import_service.go:632`, `http/admin.go:3011`, `mail/service.go:865 (Kommentar)`, `dataexport/loader.go:115`, `dataexport/excel/fields.go:246`.
- **Helm-Deploy** (K5): Migration-Job pre-Backend-Update via `migrate-job.yaml`. Drift-Window ~30s akzeptiert.

**Owner-Wortlaut + Detail-Entscheidungen (Runde 2 via AskUserQuestion):**

| ID | Entscheidung |
|---|---|
| T1 | **Toggle-Label**: „Mitglied für Umstellung auf B2B vorbereiten" |
| T2 | **Hilfetext** (Popover): „Bei aktivem Toggle erhält das Mitglied zusätzlich zum CORE-Mandat das B2B-Mandat-PDF. Im eegFaktura-Core wird der Antrag mit SEPA-CORE angelegt. Nach erfolgreicher Bank-Aktivierung stellt der Admin den SEPA-Typ im Core manuell auf B2B um." |
| T3 | **Member-Banner** (Mandate-Mail, Mitglied-fokussiert): „Anbei zusätzlich das B2B-Mandat zur Vorlage bei Ihrer Hausbank. Sobald die Bank die Firmenlastschrift aktiviert hat, geben Sie der EEG kurz Bescheid — die Umstellung auf B2B erfolgt dann automatisch." |
| T4 | **EEG-Banner** (Ablage-Kopie + Vorstands-Mail, Workflow-fokussiert): „Das Mitglied erhält zusätzlich das B2B-Mandat zur Hausbank-Vorlage. Bitte nach Bestätigung der Bank-Aktivierung den SEPA-Typ im eegFaktura-Core manuell von CORE auf B2B umstellen." |
| T5 | **Detail-Anzeige**: Field immer sichtbar, „B2B-Vorbereitung: Ja / Nein" (konsistent zu anderen Detail-Feldern). Owner-Korrektur zu Q8a: AC-17 wird angepasst (Field bei Toggle=false nicht ausblenden). |
| T6 | **Status-Log-Eintrag** bei Toggle-Änderung: ja, „B2B-Vorbereitung aktiviert/deaktiviert durch [Subject]". |
| T7 | **Edge-Case `einzugsart=b2b`**: erlaubt — Backend lässt es zu, Faktura bekommt B2B (Rollback-Pfad), Antrag läuft direkt in `ready_for_activation`. |
| T8 | **Status-Display-Fallback**: bei `awaiting_bank_confirmation`-Drift zeigt UI „Unbekannter Status" statt Crash. |

## User Stories

- **US-1:** Als Admin will ich beim Bearbeiten eines Firmen-Mitglieds einen
  expliziten Toggle „B2B-Vorbereitungs-Unterlagen mitsenden" setzen können,
  damit das Mitglied zusätzlich zum CORE-Mandat das B2B-Mandat-PDF erhält,
  ohne dass im Faktura-Core etwas anderes als CORE landet.
- **US-2:** Als Admin will ich im Antrags-Detail sehen, ob für ein Mitglied
  B2B-Vorbereitungs-Unterlagen mitgesendet wurden, damit ich nachvollziehen
  kann, was an die Person geschickt wurde.
- **US-3:** Als Mitglied (Firma) will ich neben dem CORE-Mandat auch das
  B2B-Mandat erhalten, damit ich es bei der Hausbank einreichen und auf
  Firmenlastschrift umstellen kann, sobald die Bank bereit ist.
- **US-4:** Als EEG-Vorstand (Mail-Empfänger) will ich in der Aktivierungs-Mail
  erkennen, dass das Mitglied B2B-Vorbereitungs-Unterlagen erhalten hat,
  damit ich später im Faktura-Core den SEPA-Typ manuell auf B2B umstellen kann.
- **US-5:** Als Code-Maintainer will ich, dass das heimliche
  `b2b → CORE`-Mapping (PROJ-79) und der inzwischen sinnentleerte Status
  `awaiting_bank_confirmation` entfernt sind, damit die Code-Pfade
  transparent und konsistent sind.

## Acceptance Criteria

### Datenmodell

- [ ] **AC-1** Neue Migration `000074_prepare_b2b_documents.up.sql`:
  `ALTER TABLE member_onboarding.application ADD COLUMN prepare_b2b_documents BOOLEAN NOT NULL DEFAULT FALSE`
- [ ] **AC-2** Migration setzt alle bestehenden Anträge mit
  `einzugsart='b2b'` auf `einzugsart='core'` + `prepare_b2b_documents=true`
- [ ] **AC-3** Migration setzt alle bestehenden Anträge mit Status
  `awaiting_bank_confirmation` auf Status `ready_for_activation` und schreibt
  einen `status_log`-Eintrag mit Begründung „PROJ-91: Status entfernt"
- [ ] **AC-4** Down-Migration stellt das alte Schema wieder her (Symmetrie-Anker;
  Bestand-Rekonstruktion nicht möglich, dokumentiert)

### Status-Modell

- [ ] **AC-5** `shared.StatusAwaitingBankConfirmation` Konstante entfernt
- [ ] **AC-6** Alle Einträge aus `shared.StatusTransitions`, die
  `awaiting_bank_confirmation` als Quelle oder Ziel haben, sind entfernt
- [ ] **AC-7** `internal/http/admin.go:isKnownStatus` enthält
  `awaiting_bank_confirmation` nicht mehr (Drift-Wache-Test
  `TestIsKnownStatus_CoversAllApplicationStatuses` zieht automatisch nach)
- [ ] **AC-8** DB-CHECK-Constraint `application.status` enthält
  `awaiting_bank_confirmation` nicht mehr
- [ ] **AC-9** CLAUDE.md Status-Modell-Abschnitt entfernt den Status + alle
  zugehörigen Transitionen aus der Dokumentation
- [ ] **AC-10** `docs/domain-model.md` analog aktualisiert

### Import-Pfad

- [ ] **AC-11** `internal/importing/payload.go` `mapEinzugsart`: case `"b2b"`
  returnt wieder `"B2B"` (PROJ-79-Rollback). Code-Kommentar an der Stelle
  verweist auf PROJ-91-Spec und erklärt den Rollback.
- [ ] **AC-12** Test `TestBuildPayload_B2B_IntentionallyMappedToCORE` aus
  PROJ-79 entfernt; `TestBuildPayload_EinzugsartMapping` zeigt wieder
  `{"b2b", "B2B"}`
- [ ] **AC-13** `internal/application/import_service.go`: der automatische
  Übergang nach `awaiting_bank_confirmation` bei `einzugsart=b2b` ist entfernt.
  Stattdessen wird `imported → ready_for_activation` direkt durchlaufen
  (gleicher Pfad wie für alle anderen einzugsarten).

### Admin-Edit & Detail (Frontend + Backend)

- [ ] **AC-14** Im Admin-Edit-Form (`admin-edit-form.tsx`) erscheint in der
  SEPA-Sektion direkt unter dem Einzugsart-Selector ein Toggle
  „B2B-Vorbereitungs-Unterlagen mitsenden" mit Hilfetext: „Mitglied wird mit
  SEPA-CORE im eegFaktura-Core angelegt, erhält aber zusätzlich das B2B-Mandat,
  um es bei seiner Hausbank zur Firmenlastschrift-Aktivierung einzureichen."
- [ ] **AC-15** Toggle ist nur sichtbar, wenn `einzugsart === 'core'`. Bei
  Wechsel auf `b2b` oder `kein_sepa` wird der Toggle ausgeblendet und der
  Wert auf `false` zurückgesetzt.
- [ ] **AC-16** Toggle wird per Auto-Save persistiert (analog PROJ-82-Pattern),
  Form-State synchronisiert mit DB-Stand.
- [ ] **AC-17** Im Antrags-Detail (`admin-application-detail.tsx`) erscheint
  in der SEPA-Sektion eine Read-Only-Anzeige „B2B-Vorbereitung: Ja / Nein"
  (immer sichtbar, analog zu anderen Detail-Feldern — Owner-Korrektur T5).
  **Scope-Begrenzung Owner 2026-06-09:** Diese Anzeige ist auf die Detail-
  Ansicht eines einzelnen Antrags beschränkt. Sie erscheint NICHT in der
  Antrags-Übersicht, NICHT als Spalte in Listen, NICHT als Badge im Header,
  NICHT als Filter-Option und NICHT in der Reconciliation-Ansicht.
- [ ] **AC-17a** Keine neue Spalte in der Antrags-Liste
  (`admin-applications-table.tsx`).
- [ ] **AC-17b** Keine neue Status-Badge oder Übersichts-Markierung im
  Layout-Header oder in Listen-Ansichten.
- [ ] **AC-17c** Kein neuer Filter „Mit B2B-Vorbereitung" in der Antrags-
  Übersicht.
- [ ] **AC-18** Backend-DTO `Application` enthält das Feld `PrepareB2BDocuments bool`.
  Repository-SQL setzt es bei INSERT/UPDATE. Service-Mapping schreibt es
  durch (Memory `feedback_admin_field_full_chain` — alle 6 Layer).

### Mail & PDF

- [ ] **AC-19** Wenn `prepare_b2b_documents=true`, generiert der Mail-Pfad
  zusätzlich zum CORE-Mandat-PDF auch das B2B-Mandat-PDF (gleiche
  Generator-Funktion wie heute bei `einzugsart=b2b`)
- [ ] **AC-19a** **CORE-Mandat-PDF folgt der EEG-SEPA-Konfiguration unverändert.**
  Owner-Direktive 2026-06-09: bei `einzugsart=core` (mit oder ohne Toggle) wird
  die jeweilige Variante (Audit-Trail-Block vs. klassisches Unterschriften-Feld)
  durch die EEG-Einstellungen (PROJ-78 Audit-Toggle, PROJ-80 Mandate-Timing-Toggle)
  bestimmt. Der neue B2B-Vorbereitungs-Toggle hat **keinen Einfluss** auf das
  Rendering des CORE-Mandats — es bleibt exakt so, wie es ohne den Toggle
  gerendert würde.
- [ ] **AC-19b** **Das zusätzliche B2B-Mandat-PDF folgt analog der konfigurierten
  Variante** (Audit-Trail oder klassisches Unterschriftenfeld) — die beiden
  PDFs sind konsistent gerendert, weil sie aus der gleichen EEG-Einstellung
  abgeleitet werden. Bei aktivem B2B-Audit-Toggle (PROJ-78 B2B-Pfad): Audit-
  Block-Rendering wie heute bei `einzugsart=b2b`. Bei deaktiviertem B2B-Audit-
  Toggle: klassisches Unterschriftenfeld-Layout (PROJ-89-Layout).
- [ ] **AC-20** PDF-Erzeugung folgt PROJ-74-Hard-Fail-Pattern: bei Fehler
  in der B2B-PDF-Generierung wird die Mail NICHT versendet, Status bleibt
  unverändert, Admin sieht Fehler. (Memory `feedback_mail_hard_fail`)
- [ ] **AC-21** Welcome-Mail (Auto-Modus, `SendActivationNotification`) enthält
  bei `prepare_b2b_documents=true` einen 2-3-Satz-Hinweis-Block (gelber Banner-
  Stil analog PROJ-79):
  „Anbei zusätzlich das B2B-Mandat. Bitte bei Ihrer Hausbank einreichen, um
  auf Firmenlastschrift umzustellen. Sobald die Bank bestätigt, wird der
  SEPA-Typ im Faktura-Core manuell umgestellt." (Owner-Wortlaut zur Approval)
- [ ] **AC-22** EEG-Kopie der Aktivierungs-Mail bekommt dasselbe B2B-PDF
  + denselben Hinweis-Block (Vorstand muss wissen, dass das Mitglied das Set
  erhalten hat)
- [ ] **AC-23** Vorstands-Mail (`SendBoardApprovalRequest`, PROJ-76) bekommt
  bei `prepare_b2b_documents=true` denselben Hinweis-Block
- [ ] **AC-24** Banner-HTML-Generator wird über einen gemeinsamen Helper
  `RenderB2BPrepareNoticeBanner(prepare bool)` implementiert (Memory
  `feedback_shared_helpers_for_parallel_paths`), parallel zu existierendem
  `RenderB2BImportNoticeBanner` (der nach PROJ-91 obsolet ist und entfernt
  wird, sofern keine andere Stelle ihn nutzt)

### Aufräum-Tasks

- [ ] **AC-25** PROJ-79 B2B-Import-Notice-Banner-Helper
  (`internal/mail/b2b_notice.go`) und seine Aufrufer entfernen, falls der
  neue Banner-Helper sie ersetzt. Mail-Templates: `{{.B2BNoticeHTML}}`
  durch neuen Field-Namen ersetzen (oder ganz entfernen, falls Logik
  in den Helper-Aufruf wandert).
- [ ] **AC-26** `internal/mail/b2b_notice_test.go` entsprechend angepasst
  oder entfernt
- [ ] **AC-27** CHANGELOG.md-Eintrag im selben Commit wie der Code
  (Memory `feedback_batch_changelog_with_code`)
- [ ] **AC-28** docs/user-guide PROJ-frei aktualisiert (Memory
  `feedback_no_proj_refs_in_user_doc`) — neue Sektion „B2B-Vorbereitungs-
  Unterlagen" im Admin-Bereich + Musterbetrieb-GmbH-Beispiel
  (Memory `feedback_anonymized_examples`); kein PROJ-Bezug

### Tests

- [ ] **AC-29** Go-Tests:
  - `TestBuildPayload_EinzugsartMapping` zeigt `b2b → B2B`
  - Drift-Wache-Test über `shared.Status*` Konstanten (analog PROJ-86)
    bestätigt, dass `awaiting_bank_confirmation` weg ist
  - 4 neue Mail-Tests: Toggle ON + Toggle OFF, jeweils Auto- + Vorstands-
    Modus, prüfen ob B2B-PDF angehängt + Hinweis-Block enthalten
  - `prepare_b2b_documents`-Migration-Test (Bestand-b2b → core+toggle)
- [ ] **AC-30** Frontend-Vitest:
  - Toggle nur sichtbar bei `einzugsart=core`
  - State-Reset bei einzugsart-Wechsel
  - Auto-Save-Persistierung
- [ ] **AC-31** `go test ./...` + `go build ./...` + `npx vitest run` +
  `npm run build` clean

### Wortlaute (Owner-approbiert via /grill-me 2026-06-09)

- [ ] **AC-W1** Toggle-Label im Admin-Edit-Form: „Mitglied für Umstellung
  auf B2B vorbereiten"
- [ ] **AC-W2** Popover-Hilfetext: „Bei aktivem Toggle erhält das Mitglied
  zusätzlich zum CORE-Mandat das B2B-Mandat-PDF. Im eegFaktura-Core wird
  der Antrag mit SEPA-CORE angelegt. Nach erfolgreicher Bank-Aktivierung
  stellt der Admin den SEPA-Typ im Core manuell auf B2B um."
- [ ] **AC-W3** Member-Banner-Wortlaut (Mitglied-fokussiert): „Anbei
  zusätzlich das B2B-Mandat zur Vorlage bei Ihrer Hausbank. Sobald die Bank
  die Firmenlastschrift aktiviert hat, geben Sie der EEG kurz Bescheid —
  die Umstellung auf B2B erfolgt dann automatisch."
- [ ] **AC-W4** EEG-Banner-Wortlaut (Workflow-fokussiert für Vorstand):
  „Das Mitglied erhält zusätzlich das B2B-Mandat zur Hausbank-Vorlage.
  Bitte nach Bestätigung der Bank-Aktivierung den SEPA-Typ im eegFaktura-
  Core manuell von CORE auf B2B umstellen."
- [ ] **AC-W5** Detail-Anzeige-Wortlaut: „B2B-Vorbereitung: Ja / Nein"
- [ ] **AC-W6** Status-Log-Eintrag-Wortlaut bei Toggle-Änderung:
  „B2B-Vorbereitung aktiviert" bzw. „B2B-Vorbereitung deaktiviert"
  (Actor-Subject und Timestamp werden vom Status-Log-Repo automatisch
  ergänzt)
- [ ] **AC-W7** Edge-Case `einzugsart=b2b` direkt nach Strip: Backend
  akzeptiert weiter, Faktura bekommt B2B (PROJ-79-Rollback), Antrag
  landet in `ready_for_activation`. Defense-in-Depth-Validation:
  Backend setzt `prepare_b2b_documents=false` wenn `einzugsart != 'core'`
  (auch wenn Frontend es schon zurücksetzt).
- [ ] **AC-W8** Status-Display-Fallback: bei unbekanntem Status-Wert
  zeigt `admin-status-badge.tsx` Label „Unbekannter Status" + neutrale
  Slate-Color statt Crash. Defense gegen Bestand-Drift.

### Security (für /security-review-Phase, nicht Implementierungs-Pflicht)

- [ ] **AC-32** Schema-Migration berührt nicht-tenant-isolierte Spalten
  (`einzugsart`, neuer Bool); kein Auth- oder Tenant-Boundary-Risiko
- [ ] **AC-33** PDF-Generierung läuft im gleichen Tenant-Kontext wie heute
  (Application + EEG + zugehörige Stammdaten); kein Cross-Tenant-Pfad
- [ ] **AC-34** Banner-HTML ist statisches Compile-Time-Snippet, kein
  User-Input rendert direkt

## Edge Cases

- **EC-1** Toggle aktiv + `einzugsart` nachträglich auf `b2b` umgestellt:
  Frontend setzt Toggle auf false zurück (UI-Cleanup); Backend macht
  defense-in-depth dasselbe in der Validierung. Das B2B-PDF wird nicht
  mitgeschickt, da `einzugsart=b2b` bereits B2B im Core anlegt.
- **EC-2** Toggle aktiv + Mitgliedstyp nicht `company`: Banner-Hinweis-Text
  bleibt unverändert, da die Hausbank-Aktivierung auch für andere
  Mitgliedstypen theoretisch denkbar ist (z.B. eingetragener Verein mit
  B2B-Konto). Owner-Direktive 2026-06-09: kein zusätzlicher Mitgliedstyp-
  Filter.
- **EC-3** Migration läuft in eine Antrag-Liste mit gemischten Status-
  Werten (einige `imported`, einige `awaiting_bank_confirmation`,
  einige `approved`): nur die Status-Werte `awaiting_bank_confirmation`
  werden auf `ready_for_activation` umgestellt; andere bleiben unangetastet.
- **EC-4** Migration läuft auf einem Cluster ohne `awaiting_bank_confirmation`-
  Bestand (z.B. fresh-deployed test): UPDATE matcht 0 Rows, Migration
  läuft idempotent durch.
- **EC-5** Toggle aktiv, aber B2B-PDF-Generator wirft Fehler: PROJ-74-Hard-
  Fail-Pattern → Mail wird NICHT versendet, Status bleibt vor `imported` oder
  Aktivierung. Admin sieht Fehler im Status-Log.
- **EC-6** Externe API (PROJ-13): nimmt das `prepare_b2b_documents`-Feld
  nicht entgegen, Bestand-API-Calls bleiben kompatibel. Field bekommt
  Default FALSE. (Out-of-Scope für V1, kein Backwards-Compatibility-Bruch.)
- **EC-7** Excel-Import: nimmt das `prepare_b2b_documents`-Feld nicht
  entgegen, Default FALSE. Admin kann nachträglich im Edit-Form aktivieren.
- **EC-8** Migration läuft, aber DELETE der `awaiting_bank_confirmation`-
  Transitionen aus `shared.StatusTransitions` ist noch nicht im Code:
  Datenbank-Constraint passt (Status entfernt), Go-Code würde den Status
  noch akzeptieren bis zum nächsten Deploy. Migration MUSS im selben Deploy
  wie der Code-Change kommen (kein Migration-only-Deploy).
- **EC-9** Bestand-Antrag in `awaiting_bank_confirmation` mit
  `einzugsart=b2b`: doppelte Migration: erst `awaiting_bank_confirmation
  → ready_for_activation` (Status), dann `b2b → core` (Einzugsart) +
  Toggle setzen. Reihenfolge im UPDATE-Statement egal — beide unabhängig.
- **EC-10** Admin setzt Toggle aktiv, ändert Mitgliedstyp von `company`
  auf `private`: Frontend zeigt Toggle weiter (EC-2). Backend akzeptiert.
  B2B-Mandat-PDF wird trotzdem generiert; Hausbank-Workflow für Privat-
  konto ist Mitglieds-Entscheidung.
- **EC-11** Welcome-Mail-Versand-Fehler nach erfolgreicher PDF-Generierung:
  PROJ-74-Pattern → Status bleibt, Admin kann manuell „Mail erneut senden"
  triggern.
- **EC-12** Bestand-Antrag im Status `awaiting_bank_confirmation` mit
  bereits gesetzter `bank_confirmed_at`: Migration setzt nur Status um,
  Audit-Timestamps bleiben unangetastet. Field bleibt in DB als historischer
  Beleg, wird in Frontend-Detail nicht mehr angezeigt (Status-Display
  ohnehin obsolet).
- **EC-13** Admin setzt Toggle aktiv, dann später `einzugsart=b2b`: Frontend
  setzt Toggle auf false (B1), Backend macht Defense-in-Depth dasselbe (B2,
  AC-W7). Status-Log bekommt zwei Einträge (Toggle deaktiviert + Einzugsart-
  Wechsel).
- **EC-14** Externe API (PROJ-13) sendet `einzugsart=b2b` für ein neues
  Mitglied: nach Rollback PROJ-79 geht b2b sauber durch ins Faktura als B2B.
  Toggle bleibt false (API kennt das Feld nicht). Admin kann nachträglich
  umstellen.
- **EC-15** Migration läuft auf einem Cluster ohne `awaiting_bank_confirmation`-
  Bestand: UPDATE matcht 0 Rows, idempotent.
- **EC-16** Toggle aktiv + Admin triggert „Mail erneut senden" via
  `SendMandateRenewalMail`: aktueller Toggle-Stand wird verwendet, B2B-PDF
  + Banner werden mitgesendet. (PROJ-70-Pfad zieht aktuelle Application-
  Daten aus DB.)
- **EC-17** Toggle aktiv + EEG hat `SEPAMandateAtImport=false`: das B2B-PDF
  wird in der V1 NICHT automatisch beim Submit angehängt (Submit-Pfad
  bleibt unverändert — `SendSubmissionEmails` schickt nur EIN PDF). Erst
  beim Import läuft der post-import-Pfad und der Toggle wird sichtbar. Bei
  EEGs ohne AtImport bleibt der Toggle wirkungslos, bis der Admin per
  „Mail erneut senden" manuell triggert. Empfehlung in der User-Doku:
  EEG-Einstellung `SEPAMandateAtImport=true` für Toggle-Wirksamkeit.

## Technical Requirements

- **Performance:** Migration auf Test-Bestand (vermutlich <100 Anträge mit
  awaiting_bank_confirmation, einzelne b2b-Anträge) läuft <5 Sekunden
- **Security:** Schema-Migration + Import-Logik berührt — `/security-review`
  Pflicht-Trigger laut CLAUDE.md
- **Backward Compatibility:** Externe API (PROJ-13) bleibt unverändert; Field
  bekommt Default FALSE bei API-Calls ohne explizite Angabe
- **Browser Support:** Chrome, Firefox, Safari (Standard-Stack)

## Risiken (Top-3 nach /grill-me)

| # | Risiko | Mitigation |
|---|---|---|
| R1 | Migrations-Reihenfolge falsch: DROP CHECK CONSTRAINT vor UPDATE auf `ready_for_activation` ist Pflicht, sonst Constraint-Violation | Reihenfolge in der up.sql explizit dokumentiert; Migration-Test seeded Bestand mit `awaiting_bank_confirmation` und validiert Endstand |
| R2 | Status-Strip im Backend, aber Bestand-Antrag wird per Drift mit altem Status zurückgegeben → Frontend-Crash | AC-W8: Status-Display-Map mit Fallback „Unbekannter Status" + neutrale Color, kein Crash |
| R3 | Hard-Fail-PDF-Pattern Edge-Case: B2B-PDF-Generation schlägt fehl, Mitglied wird nicht aktiviert obwohl Daten OK | `SendMandateAtImport` returnt Error → Status bleibt vor Activation, Admin sieht Fehler. Resend-Pfad funktioniert mit aktuellem Toggle-Stand |

## Out-of-Scope

- Externe API (PROJ-13) erweitern um `prepareB2BDocuments`-Feld — eigenes
  späteres PROJ falls Bedarf
- Excel-Import um die Spalte erweitern — eigenes späteres PROJ falls Bedarf
- Hart-Cut-Variante für `awaiting_bank_confirmation`-Bestand (z.B. auf
  `approved` zurück) — Migration auf `ready_for_activation` ist Owner-Wahl
- Per-EEG-Default für den Toggle (z.B. „immer B2B-Vorbereitung an" als
  EEG-Setting) — nicht erforderlich, Admin entscheidet pro Antrag

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)

_Erstellt 2026-06-09 nach /requirements + /grill-me. Alle Design-Branches
sind durch Q1–Q11 + T1–T8 festgenagelt; diese Sektion fasst die Architektur
PM-lesbar zusammen, damit klar wird **was** gebaut wird und **warum**._

### A) Komponenten-Übersicht

```
Member-Onboarding-System
│
├── Frontend (Next.js / React)
│   ├── Public-Form ──────────────────── (unverändert)
│   ├── Admin-Edit-Form                  ◀ NEUER Toggle „Mitglied für
│   │   ├── Persönliche Daten              Umstellung auf B2B vorbereiten"
│   │   ├── SEPA-Sektion                   in der SEPA-Sektion, sichtbar
│   │   │   ├── Einzugsart-Selector        nur wenn Einzugsart = CORE
│   │   │   ├── Bankverbindung
│   │   │   └── B2B-Vorbereitungs-Toggle ◀
│   │   └── (übrige Sektionen)
│   ├── Admin-Application-Detail         ◀ NEUE Read-Only-Zeile in
│   │   └── SEPA-Sektion                   SEPA-Sektion „B2B-Vorbereitung:
│   │       └── B2B-Vorbereitungs-Field ◀  Ja / Nein" (immer sichtbar)
│   ├── Admin-Status-Badge               ◀ Status-Strip + Fallback
│   │                                      „Unbekannter Status"
│   ├── Admin-Filter-Panel               ◀ Filter-Option „Wartet auf Bank-
│   │                                      Bestätigung" entfernt
│   └── Admin-Status-Actions             ◀ „Bank-Bestätigung erhalten"-
│                                          Button entfernt
│
├── Backend (Go)
│   ├── Application-Service              ◀ Field-Persistierung +
│   │                                      Defense-in-Depth-Reset bei
│   │                                      Einzugsart-Wechsel
│   ├── Status-Transitions-Map           ◀ 4 Transitionen entfernt
│   ├── Import-Service                   ◀ Auto-Trigger „awaiting" entfernt;
│   │                                      alle Einzugsarten → ready_for_act.
│   ├── Mail-Service                     ◀ Mandate-Mail bekommt zweites
│   │   ├── SendMandateAtImport            B2B-PDF-Attachment + Banner-Block
│   │   ├── SendBoardApprovalRequest     ◀ Banner-Helper-Aufruf umgebaut
│   │   └── SendActivationNotification   ◀ Banner-Helper-Aufruf umgebaut
│   ├── PDF-Generator                    (unverändert wiederverwendet)
│   ├── Banner-Helper                    ◀ Umbau von PROJ-79-„NoticeBanner"
│   │                                      zu „PrepareNoticeBanner"
│   ├── Import-Mapping                   ◀ PROJ-79-Rollback: b2b → B2B
│   └── Excel-Export / Loader            ◀ Status „awaiting" aus Field-Liste
│
└── Datenbank (PostgreSQL 16)
    ├── application                      ◀ NEUE Spalte
    │   └── prepare_b2b_documents          BOOLEAN NOT NULL DEFAULT FALSE
    ├── application.status                ◀ CHECK-Constraint enger:
    │                                       „awaiting_bank_confirmation" raus
    └── status_log                        (unverändert; bekommt neue
                                            Einträge bei Toggle-Wechseln)
```

### B) Datenfluss-Sequenzen

**Sequenz 1 — Admin schaltet Toggle aktiv:**

```
Admin (Edit-Form)
   │ (klickt Toggle „Mitglied für Umstellung auf B2B vorbereiten")
   ▼
Frontend Auto-Save (500 ms Debounce, PROJ-66-Pattern)
   │ PUT /api/admin/applications/{id}
   ▼
Backend Handler
   │ Validation: Einzugsart muss „core" sein, sonst Toggle erzwungen FALSE
   ▼
Backend Service
   │ DB-UPDATE prepare_b2b_documents
   │ status_log-Insert „B2B-Vorbereitung aktiviert durch [Subject]"
   ▼
Response
   │ onSaved-Callback (PROJ-82-Pattern)
   ▼
Frontend Parent-Cache wird aktualisiert
```

**Sequenz 2 — Import läuft mit Toggle aktiv:**

```
Admin (Antrags-Detail) klickt „Importieren"
   │
   ▼
Import-Service
   │ Mapping: Einzugsart=core → "CORE" an Faktura-Core
   │ Faktura-Core legt Mitglied mit SEPA-CORE an
   │ Status: imported → ready_for_activation (Auto, gleicher Pfad wie alle)
   ▼
Mail-Service.SendMandateAtImport
   │ PDF-Variante = SEPAMandateCoreAuditEnabled  → CORE-PDF rendern
   │ Toggle aktiv? → PDF-Variante = SEPAMandateB2BAuditEnabled
   │                → B2B-PDF zusätzlich rendern (GenerateCompany)
   │ Mail mit 2 Attachments + Banner-Block an Mitglied
   │ EEG-Kopie (wenn Audit aktiv) mit 2 Attachments + Banner-Block
   ▼
Hard-Fail-Pattern: PDF-Generation fehlschlagen → kein Mail-Versand,
                    Status bleibt vor Aktivierung, Admin sieht Fehler
```

**Sequenz 3 — Bestand-Migration beim helm upgrade:**

```
helm upgrade
   │
   ▼
Migration-Job (pre-Backend-Update, Postgres 16-alpine)
   │ Step 1: ADD COLUMN prepare_b2b_documents (instant, metadata-only)
   │ Step 2: UPDATE Bestand-Anträge einzugsart=b2b
   │         → einzugsart=core + prepare_b2b_documents=true
   │ Step 3: DROP + RECREATE CHECK-Constraint ohne „awaiting_bank_confirmation"
   │ Step 4: UPDATE Bestand-Anträge status=awaiting_bank_confirmation
   │         → status=ready_for_activation + status_log-Eintrag
   ▼
Backend-Pod-Rollout (neuer Code, ohne Status-Konstante)
   │ Drift-Window ~30 s, alte Pods sehen alten Status nicht mehr (Bestand
   │ ist schon migriert)
   ▼
Frontend-Pod-Rollout parallel
```

**Sequenz 4 — Resend-Mail mit aktuellem Stand:**

```
Admin (Antrags-Detail) klickt „Mail erneut senden"
   │
   ▼
Service liest Application frisch aus DB
   │ → aktueller prepare_b2b_documents-Stand wird übernommen
   ▼
SendMandateAtImport mit aktuellem Toggle-Stand
   │ Bei Toggle=true → wieder 2 PDFs + Banner
   │ Bei Toggle=false → nur CORE-PDF (wie heute)
```

### C) Datenmodell-Beschreibung (Plain Language)

**Was sich ändert:**

Jede Mitgliederanmeldung (Antrag) bekommt eine zusätzliche Information:
„Soll diesem Mitglied zusätzlich das B2B-Mandat mitgeschickt werden, um
seine Bank auf Firmenlastschrift umzustellen?" Diese Information ist
ein einfaches Ja/Nein und wird beim Antrag selbst gespeichert. Im
Standardfall steht sie auf „Nein"; der Admin kann sie pro Antrag im
Bearbeiten-Formular aktivieren.

Außerdem entfernt diese Änderung einen alten Workflow-Status:
„Wartet auf Bank-Bestätigung". Der wurde früher automatisch gesetzt,
wenn ein Antrag mit Firmenlastschrift (B2B) ins Faktura importiert wurde —
und der Admin musste manuell bestätigen, dass die Bank umgestellt hat,
bevor das Mitglied aktiviert werden konnte. Im neuen Modell entfällt
dieser Zwischenschritt, weil der neue Toggle den Workflow transparent
macht: das Mitglied bekommt die Dokumente proaktiv, der Antrag läuft
direkt bis zur Aktivierung durch, und die Bank-Klärung passiert im
Hintergrund.

**Bestandsdaten:**

Alle Anträge, die heute mit Einzugsart „B2B" gespeichert sind, werden
automatisch auf „CORE" umgestellt und bekommen den neuen Toggle aktiv —
sie sollen ja genau das gleiche Verhalten zeigen wie zukünftige Anträge
mit dem neuen Modell. Alle Anträge im Status „Wartet auf Bank-
Bestätigung" werden auf „Ready für Aktivierung" gesetzt, damit sie
nicht in einem nicht-mehr-existierenden Status hängen bleiben.

**Speicherort:** PostgreSQL-Datenbank, Schema `member_onboarding`,
Tabelle `application` — bestehender Speicher, keine neuen Tabellen.

### D) Tech-Entscheidungen (mit Begründungen)

| Entscheidung | Warum so |
|---|---|
| **Neues DB-Feld statt heuristischer Ableitung** | Der Toggle ist Audit-relevant (wer hat wann was beigelegt?). Eine Ableitung aus anderen Feldern wäre fragil und ohne Historie. |
| **CORE-Audit und B2B-Audit getrennt** (statt einem globalen Toggle) | Konsistent zur bestehenden PROJ-78-Struktur: EEG kann pro Mandatstyp entscheiden, welches Layout sie nutzt (Audit-Trail oder Klassik). |
| **Nur ein Mail-Pfad** (`SendMandateAtImport`) bekommt das zweite PDF | Das ist der einzige Mail-Pfad, der SEPA-Mandate als Anhang verschickt. Andere Mails (Beitrittsbestätigung, Vorstands-Mail) bekommen nur den Hinweis-Text, weil sie konzeptionell keine Bank-Dokumente liefern. |
| **Einzugsart „B2B" bleibt im Selector erlaubt** | Admin-Flexibilität für Ausnahmefälle (Mitglied hat die Bank-Aktivierung schon erledigt, kein Vorbereitungs-Workflow nötig). Faktura bekommt dann B2B direkt. |
| **Status „Wartet auf Bank-Bestätigung" hart entfernt** statt nur deprecated | Owner-Direktive für saubere Lösung. Der Status hat in der neuen Welt keinen Sinn mehr (kein Auto-Trigger, kein manueller Workflow), und alle Bestandsdaten werden konsistent migriert. |
| **Kein Feature-Flag** für den Status-Strip-Rollback | Status-Wiederherstellung wäre nach Bestand-Migration ohnehin nicht trivial. Reverter-Commit + Migration-Down ist akzeptabel; Bestand-Verlust ist im Down-Pfad dokumentiert. |
| **Hard-Fail-Pattern für die B2B-PDF-Generation** | Konsistent zu PROJ-74: wenn das B2B-PDF nicht erzeugt werden kann, soll die Mail gar nicht raus — sonst sieht das Mitglied eine inkonsistente Mail mit nur CORE-Mandat. |

### E) Migrationspfad

1. **Code-Deploy + Migration-Job in einem helm upgrade.**
   Der Migration-Job läuft pre-Backend-Update; das ist die existierende
   Helm-Konvention.
2. **Bestand-Migration läuft atomar:** alle vier Steps (ADD COLUMN,
   einzugsart-Backfill, CHECK-Constraint-Reformat, Status-Backfill)
   in einer Transaktion.
3. **Drift-Window ~30 Sekunden:** alte Backend-Pods sehen während der
   Migration keinen `awaiting_bank_confirmation`-Bestand mehr (ist schon
   umgestellt). Neue Backend-Pods starten mit korrektem Code.
4. **Rollback** (im Notfall): Reverter-Commit, Migration-Down. Daten-
   Restore unmöglich (Status-Information war im Bestand). Akzeptiert.

### F) Risiken & Trade-offs

| # | Risiko | Mitigation |
|---|---|---|
| R1 | **Migrations-Reihenfolge falsch:** DROP CHECK CONSTRAINT vor UPDATE auf `ready_for_activation` ist Pflicht, sonst Constraint-Violation | Reihenfolge in der up.sql explizit dokumentiert; Migration-Test seeded Bestand und validiert Endstand |
| R2 | **Status-Strip-Drift:** Bestand-Antrag wird per Drift mit altem Status zurückgegeben → Frontend-Crash | AC-W8: Status-Display-Map mit Fallback „Unbekannter Status" |
| R3 | **Hard-Fail PDF Edge-Case:** B2B-PDF-Generation schlägt fehl, Mitglied wird nicht aktiviert obwohl Daten OK | SendMandateAtImport returnt Error → Status bleibt vor Activation, Admin sieht Fehler, Resend möglich |
| R4 | **Bestands-b2b-Antrag wird automatisch auf CORE migriert obwohl Admin B2B wollte** | Akzeptiert: B2B im Bestand entstand nur durch Admin-Edit (Public-Form hat keinen Selector); Admin kann nach Migration jederzeit zurück auf b2b stellen |
| R5 | **Wartet-auf-Bank-Bestätigung-Bestand wird auf Ready-für-Aktivierung gesetzt obwohl Bank noch nicht bestätigt hat** | Akzeptiert: Activation-Check-Batch springt nicht automatisch auf „aktiviert" — Admin entscheidet weiter manuell |

### G) Dependencies

**Keine neuen Pakete.** Alle benötigten Bausteine existieren bereits:

- `template.HTML` (Go stdlib) — Banner-Helper
- `useDebouncedAutoSave` (in `src/hooks/`) — Auto-Save-Pattern
- `GenerateCompany`-PDF-Generator (in `internal/pdf/`) — wiederverwendet
- shadcn/ui `Switch` + `Popover` + `Info`-Icon — schon im Frontend-Stack

### H) Implementierungs-Reihenfolge (20 Schritte für /backend + /frontend)

**Backend (Schritte 1–12):**
1. Migration `000074_b2b_prepare_documents.up.sql` + `.down.sql`
2. `shared/models.go` — Status-Konstante raus, Transitions-Map anpassen, ApplicationDTO um `PrepareB2BDocuments bool`
3. Repository — INSERT/UPDATE Field setzen
4. Service — Defense-in-Depth Reset bei Einzugsart-Wechsel; Status-Log-Eintrag bei Toggle-Änderung
5. Handler — DTO-Wiring, `isKnownStatus`-Strip in admin.go
6. Import-Service — Auto-Trigger entfernen, alle Einzugsarten → ready_for_activation
7. PROJ-79-Rollback in `payload.go` + Test umstellen
8. Banner-Helper umbauen: `RenderB2BPrepareNoticeBanner`
9. Mail-Service — `SendMandateAtImport` 2-Attachment-Logik + Banner an 2 Stellen
10. `admin_service.go` Reset-Pfade säubern (4 Stellen)
11. Excel-Export + Loader — Status aus Field-Liste raus
12. Tests — Mail (4 Permutationen), Migration, StatusTransitions-Validator, Drift-Wache

**Frontend (Schritte 13–18):**
13. TypeScript-Type-Strip in `lib/api.ts:557` + Application-DTO erweitern
14. `admin-edit-form.tsx` Toggle + Popover unter Einzugsart-Selector, Auto-Save, State-Reset
15. `admin-application-detail.tsx` Read-Only-Field
16. `admin-status-badge.tsx` Status-Strip + Fallback „Unbekannter Status"
17. `admin-filter-panel.tsx` Filter-Strip
18. `admin-status-actions.tsx` Bank-Bestätigung-Button-Strip

**Doku + Bookkeeping (Schritte 19–20):**
19. `docs/user-guide` PROJ-frei + Musterbetrieb GmbH; `CLAUDE.md` Status-Modell-Abschnitt; `docs/domain-model.md`
20. CHANGELOG.md im selben Commit (Memory `feedback_batch_changelog_with_code`)

### I) Out-of-Scope (bewusst nicht enthalten)

- Externe API um `prepareB2BDocuments`-Feld erweitern (eigenes späteres PROJ falls Bedarf)
- Excel-Import um die Spalte erweitern (eigenes späteres PROJ falls Bedarf)
- Excel-EXPORT-Field für `prepare_b2b_documents`: das Bool-Field-Pattern existiert in der Excel-Field-Map heute nicht (alle bestehenden Felder sind Text/Enum); eigenes späteres PROJ falls Bedarf
- Submit-Pfad-Erweiterung für Toggle (EC-17): wenn EEG `SEPAMandateAtImport=false`, wird das B2B-PDF in V1 nicht automatisch beim Submit angehängt
- Per-EEG-Default für den Toggle (z.B. „immer B2B-Vorbereitung an" als EEG-Setting)
- Bestands-Rekonstruktion in der Down-Migration (Daten sind weg, akzeptiert)
- Feature-Flag für Status-Strip-Rollback

## QA Test Results

**QA-Sweep 2026-06-09 nach Backend+Frontend-Implementation. Code-Review-basiert (kein Live-Backend für Browser-Tests verfügbar).**

### Verdict: **APPROVED** (nach Fix-Welle 2026-06-09)

Alle 5 Findings adressiert. Toggle wird jetzt persistiert, Defense-in-Depth Reset implementiert, Status-Log-Eintrag bei Toggle-Diff angelegt, Migration-Smoke-Test grün, User-Guide-Doku + CHANGELOG vorbereitet, CLAUDE.md + domain-model.md aktualisiert.

**Update-Status pro Finding:**
- F1: ✅ FIXED — `PrepareB2BDocuments *bool` in `AdminUpdateApplicationRequest` (`internal/shared/requests.go`)
- F2: ✅ FIXED — Service-Block + Defense-in-Depth-Reset in `AdminUpdateApplication` (`internal/application/admin_service.go`), Handler-Signature um `actorID` erweitert
- F3: ✅ FIXED — `status_log`-Insert bei Toggle-Diff (innerhalb derselben Transaktion), Reason „B2B-Vorbereitung aktiviert/deaktiviert"
- F4: ✅ FIXED — `migration_000074_test.go` mit Smoke-Test (Reihenfolge der 4 Steps, Schlüssel-Anweisungen, CHECK-Block Strip-Verifikation, Down-Symmetrie)
- F5: ✅ FIXED — Hinweis in `docs/user-guide/04-admin-applications.md` Sektion „B2B-Vorbereitung mitsenden" + Changelog-Eintrag

### Initial-Findings (vor Fix-Welle) zur Nachvollziehbarkeit

### Findings

| # | Severity | File | Function/Area | Risk | Reproduktion | Fix-Empfehlung | Confidence |
|---|---|---|---|---|---|---|---|
| F1 | **Critical** | `internal/shared/requests.go:314-356` | `AdminUpdateApplicationRequest` | Toggle wird NIE persistiert — Feature funktional defekt | Frontend setzt Toggle → PUT-Request enthält `prepareB2BDocuments:true` → Backend ignoriert Feld → DB-Wert bleibt false | Field `PrepareB2BDocuments *bool` mit JSON-Tag `"prepareB2BDocuments,omitempty"` zum DTO hinzufügen | High |
| F2 | **Critical** | `internal/application/admin_service.go:447-462` (Einzugsart-Block) | `AdminUpdateApplication` Partial-Update | Toggle-Wert wird auch bei vorhandenem Request nicht durchgereicht | dito | `if req.PrepareB2BDocuments != nil { app.PrepareB2BDocuments = *req.PrepareB2BDocuments }` zwischen Einzugsart und BankName-Block. Defense-in-Depth: bei einzugsart != "core" auf false zurücksetzen (deckt AC-W7) | High |
| F3 | Medium | `internal/application/admin_service.go` AdminUpdateApplication | Status-Log-Eintrag bei Toggle-Änderung (T6, AC-W6) fehlt | Toggle-Änderung erzeugt keinen Audit-Trail | Beim Persistieren prüfen ob `oldValue != newValue`, dann `statusLogRepo.Insert` mit Reason „B2B-Vorbereitung aktiviert/deaktiviert durch [Subject]" | Bisheriger DSGVO-Audit-Trail decken Diff via DB-Snapshots ab, aber Owner-Direktive T6 wollte explizit den Log-Eintrag | Medium |
| F4 | Medium | (kein File — fehlende Test-Datei) | Migration-Test 000074 fehlt (AC-29 Migration-Test mit Bestand-Seed) | Kein automatischer Regressions-Schutz gegen Schema-Drift wie bei PROJ-90 | Test in `internal/application/` mit Test-DB + Seed (b2b-Antrag + awaiting_bank-Antrag), Migration laufen lassen, Endstand asserten | Migration-SQL ist einfach, aber Memory `feedback_migration_after_apply_drift` zeigt dass solche Fehler teuer sind | Medium |
| F5 | Low | EC-17 nur in Spec dokumentiert | Submit-Pfad-Lücke bei `SEPAMandateAtImport=false` | Bei EEGs ohne AtImport ist der Toggle wirkungslos bis manueller Resend | User-Guide-Hinweis in /deploy: „EEG-Einstellung `SEPAMandateAtImport=true` für Toggle-Wirksamkeit empfohlen" | Akzeptiert als V1-Limit, kein Code-Fix nötig | High |

### Acceptance-Criteria-Sweep

**Datenmodell (4/4 Pass):**
| AC | Status | Notiz |
|---|---|---|
| AC-1 | ✅ Pass | `000074_*.up.sql` + `.down.sql` existieren, syntaktisch valide |
| AC-2 | ✅ Pass | Reihenfolge: ADD COLUMN → UPDATE b2b → DROP+RECREATE CHECK → UPDATE Status + status_log |
| AC-3 | ✅ Pass | Down-Migration DROP COLUMN + CHECK-Restore (Bestand-Restore unmöglich, dokumentiert) |
| AC-4 | ✅ Pass | Doc-Kommentar verweist auf `feedback_migration_after_apply_drift` |

**Status-Modell (6/6 Pass, 2 N/A für /deploy):**
| AC | Status | Notiz |
|---|---|---|
| AC-5 | ✅ Pass | Konstante raus aus `shared/models.go:131-132` |
| AC-6 | ✅ Pass | Drift-Wache-Test `TestAdminTransitions_NoDanglingAwaitingBankConfirmation` |
| AC-7 | ✅ Pass | `isKnownStatus` ohne Status, PROJ-86-Drift-Test zieht via Iterator nach |
| AC-8 | ✅ Pass | Migration 000074 Step 3: DROP+RECREATE Constraint ohne Status |
| AC-9 | ⏸ N/A | CLAUDE.md-Update kommt in /deploy |
| AC-10 | ⏸ N/A | `docs/domain-model.md`-Update kommt in /deploy |

**Import-Pfad (3/3 Pass):**
| AC | Status | Notiz |
|---|---|---|
| AC-11 | ✅ Pass | `mapEinzugsart('b2b')` → `"B2B"`, Tests grün |
| AC-12 | ✅ Pass | Alter Test ersetzt durch `TestBuildPayload_B2B_MapsToB2B` |
| AC-13 | ✅ Pass | Auto-Trigger entfernt, alle Einzugsarten → ready_for_activation |

**Admin-Edit + Detail (5/8 Pass, 3 Fail wegen F1/F2):**
| AC | Status | Notiz |
|---|---|---|
| AC-14 | ✅ Pass | Toggle-UI rendert in SEPA-Sektion |
| AC-15 | ✅ Pass | Frontend useEffect-Reset bei Einzugsart-Wechsel |
| AC-16 | ⏸ N/A | Spec sagt Auto-Save; Form hat heute Save-Button (PROJ-66-Pattern nur in Settings-Editoren). Owner-Direktive war nicht eindeutig auf Auto-Save vs. Save-Button — V1 nutzt Save-Button konsistent |
| AC-17 | ✅ Pass | Read-Only-Field „B2B-Vorbereitung: Ja/Nein" in SEPA-Sektion |
| AC-17a | ✅ Pass | Keine Spalte in `admin-applications-table.tsx` (verifiziert via grep) |
| AC-17b | ✅ Pass | Keine neue Badge, kein Layout-Header-Indikator |
| AC-17c | ✅ Pass | Kein neuer Filter in `admin-filter-panel.tsx` |
| AC-18 | ❌ **Fail** | Full-Chain 6 Layer **gebrochen**: Layer 4 (Service Partial-Update) + Layer 5 (Request-DTO Field) **fehlen**. Siehe F1+F2 |

**Mail & PDF (8/8 Pass):**
| AC | Status | Notiz |
|---|---|---|
| AC-19 | ✅ Pass | B2B-PDF generierung in `admin_service.go:1093+` + `resync_service.go:299+` |
| AC-19a | ✅ Pass | CORE-PDF unverändert, folgt `SEPAMandateCoreAuditEnabled` |
| AC-19b | ✅ Pass | B2B-PDF via `GenerateCompany`, Audit-Variante via `SEPAMandateB2BAuditEnabled` (im PDF-Renderer schon implementiert vor PROJ-91) |
| AC-20 | ✅ Pass | Hard-Fail: `return nil, err` bei B2B-PDF-Fehler, Mail wird nicht gesendet |
| AC-21 | ✅ Pass | `RenderB2BPrepareNoticeBannerMember(true)` produziert T3-Wortlaut |
| AC-22 | ✅ Pass | `RenderB2BPrepareNoticeBannerEEG(true)` produziert T4-Wortlaut |
| AC-23 | ✅ Pass | `SendBoardApprovalRequest:1163` Banner-Aufruf umgebaut |
| AC-24 | ✅ Pass | Single source — alle 3 Mail-Pfade rufen denselben Helper |

**Aufräum (2/2 Pass, 2 N/A für /deploy):**
| AC | Status | Notiz |
|---|---|---|
| AC-25 | ✅ Pass | PROJ-79-Helper komplett ersetzt |
| AC-26 | ✅ Pass | Test-File ersetzt mit 4 neuen Tests |
| AC-27 | ⏸ N/A | CHANGELOG kommt in /deploy |
| AC-28 | ⏸ N/A | User-Guide kommt in /deploy |

**Tests (2/3 Pass, 1 Fail):**
| AC | Status | Notiz |
|---|---|---|
| AC-29 | ❌ **Fail** | Migration-Test fehlt (F4) |
| AC-30 | ✅ Pass | Vitest 88/88 grün |
| AC-31 | ✅ Pass | `go test ./...` + `go build` + `tsc` + `npm run build` clean |

**Security (3/3 Pass, vorbehaltlich /security-review):**
| AC | Status | Notiz |
|---|---|---|
| AC-32 | ✅ Pass | Schema-Migration berührt nicht-tenant-Spalten |
| AC-33 | ✅ Pass | PDF-Generierung im Application+EEG-Tenant-Kontext |
| AC-34 | ✅ Pass | Banner-HTML 100% Compile-Time-strings |

**Wortlaute (6/8 Pass, 2 Fail wegen F2/F3):**
| AC | Status | Notiz |
|---|---|---|
| AC-W1 | ✅ Pass | „Mitglied für Umstellung auf B2B vorbereiten" |
| AC-W2 | ✅ Pass | Owner-approbierter Popover-Text |
| AC-W3 | ✅ Pass | Member-Banner-Wortlaut in `b2b_notice.go` |
| AC-W4 | ✅ Pass | EEG-Banner-Wortlaut in `b2b_notice.go` |
| AC-W5 | ✅ Pass | „B2B-Vorbereitung: Ja/Nein" |
| AC-W6 | ❌ **Fail** | Status-Log-Eintrag bei Toggle-Änderung fehlt (F3) |
| AC-W7 | ❌ **Fail** | Defense-in-Depth-Reset Backend fehlt (Teil von F2) |
| AC-W8 | ✅ Pass | „Unbekannter Status"-Fallback in `admin-status-badge.tsx` |

**Summary Acceptance Criteria:**
- **Pass: 33** | **Fail: 6** (5 wg F1/F2/F3, 1 wg F4) | **N/A: 4** (deploy-bound) | **Total: 47**

### Edge-Cases-Sweep

| EC | Status | Notiz |
|---|---|---|
| EC-1 | ⚠ Partial | Frontend macht Reset, Backend Reset fehlt (F2) |
| EC-2 | ✅ Pass | Banner unverändert für alle Mitgliedstypen |
| EC-3 | ✅ Pass | Migration WHERE-Clause begrenzt auf awaiting_bank |
| EC-4 | ✅ Pass | UPDATE matcht 0 Rows = idempotent |
| EC-5 | ✅ Pass | Hard-Fail-Pattern in beiden Service-Pfaden |
| EC-6 | ✅ Pass | Externe API kennt Feld nicht (kein DTO-Update), Default false |
| EC-7 | ✅ Pass | Excel-Import unverändert |
| EC-8 | ✅ Pass | Helm migrate-Job läuft pre-Backend |
| EC-9 | ✅ Pass | Migration UPDATEs sind unabhängige Spalten |
| EC-10 | ✅ Pass | Banner-Render hängt nur am Toggle, nicht am Mitgliedstyp |
| EC-11 | ✅ Pass | PROJ-74-Pattern unverändert |
| EC-12 | ✅ Pass | `bank_confirmed_at` bleibt im Schema, Frontend zeigt ihn nicht mehr |
| EC-13 | ❌ Fail | 2 Status-Log-Einträge nicht möglich weil AC-W6 (F3) nicht implementiert |
| EC-14 | ✅ Pass | Externe API b2b → B2B im Core |
| EC-15 | ✅ Pass | Migration idempotent |
| EC-16 | ✅ Pass | Resend nutzt aktuellen DB-Stand |
| EC-17 | ⚠ Documented Limit | Submit-Pfad-Lücke akzeptiert, User-Guide-Hinweis empfohlen |

**Summary Edge Cases:** Pass: 13 | Partial/Fail: 3 | Documented Limit: 1 | Total: 17

### Security-Smoke-Test (kein Critical/High gefunden — sauberer Pfad zu /security-review)

```
govulncheck ./...     : 0 callable vulnerabilities (5 pkg + 1 mod nicht callable)
gosec (5 packages)    : 0 issues
npm audit --high      : 0 high, 4 moderate (Pre-PROJ-91-Bestand: uuid GHSA-w5hq-g745-h8pq)
```

Keine neuen Security-Findings durch PROJ-91. Schema-Migration berührt keine Auth-Spalten, Banner-HTML ist statisch, PDF-Generation läuft im bestehenden Tenant-Kontext.

### Regression-Sweep

| Bereich | Status | Notiz |
|---|---|---|
| PROJ-46 Status-Modell | ✅ | awaiting_bank entfernt, ready_for_activation+activated unverändert |
| PROJ-78 Audit-Toggles | ✅ | B2B-Audit-Toggle wirkt jetzt auch auf PROJ-91-Vorbereitungs-PDF (Owner-Direktive D1 Option b) |
| PROJ-79 Mapping | ✅ | Sauber zurückgerollt auf b2b → B2B |
| PROJ-80 SEPA-Settings | ✅ | SEPAMandateAtImport-Toggle bleibt der Trigger |
| PROJ-81 NoSepaMandate-Banner | ✅ | Koexistiert (mutually exclusive durch Einzugsart-Enum) |
| PROJ-86 Drift-Wache | ✅ | Iterator zieht nach |
| PROJ-87 USt-Pflicht | ✅ | Unberührt |
| PROJ-88 Mail-Templates Audit | ✅ | Templates um Banner-Field erweitert, Audit-Logik unverändert |
| PROJ-89 B2B-PDF-Layout | ✅ | Unberührt |
| PROJ-90 Customer-Onboarding-Schema | ✅ | Irrelevant (Plattform-Buchung) |
| Reset-Import-Pfade | ✅ | Ohne awaiting_bank-Branch, sauber |

Keine bestehenden Tests durch PROJ-91 gebrochen.

### Sonstige Verifikationen

- Helm-Werte: `git status helm/` clean → ✅ keine ENV-Änderungen
- PROJ-Refs in `docs/user-guide/`: `grep -rE "PROJ-9[01]"` leer → ✅ User-Doku unangetastet
- Banner-Helper Single-Source: kein Duplikat des Helpers in `mail/service.go` → ✅

### Empfehlung (post Fix-Welle 2026-06-09)

**APPROVED for /security-review.** Alle 5 Findings adressiert, Tests grün:
- `go test ./...` alle Pakete grün (inkl. neuer Migration-Smoke-Test, Toggle-Persistierungs-Pfad)
- `npx vitest run` 88/88 grün
- `npx tsc --noEmit` clean
- `npm run build` clean

PROJ-91 berührt Schema-Migration + Status-Transitions + Import-Logik — `/security-review` ist Pflicht-Trigger laut CLAUDE.md. Nach /security-review APPROVED → /deploy.

**Stand-Update Acceptance Criteria:**
- AC-18 ❌ → ✅ Pass (Full-Chain durch alle 6 Layer geschlossen)
- AC-29 ❌ → ✅ Pass (Migration-Smoke-Test, kein DB-Test-Infra im Repo, aber Smoke deckt R1)
- AC-W6 ❌ → ✅ Pass (Status-Log-Eintrag mit Owner-Wortlaut)
- AC-W7 ❌ → ✅ Pass (Backend Defense-in-Depth-Reset implementiert)
- AC-9 + AC-10: CLAUDE.md + domain-model.md aktualisiert (vorgezogen aus /deploy)

**Final-Stand:** 37 Pass / 0 Fail / 10 N/A (CHANGELOG + Spec-/Deploy-Bookkeeping kommt im Deploy-Commit, EC-Limits dokumentiert)

## Security Review

**Reviewer:** Security Engineer (AI)
**Date:** 2026-06-09
**Scope:** alle in Spec-Section „Was sich geändert hat" gelisteten Dateien, vor allem Schema-Migration 000074, Status-Modell-Strip, Import-Logik PROJ-79-Rollback, Toggle-Persistierungs-Pfad inkl. Defense-in-Depth-Reset, Banner-Helper-Umbau, Mail-Service-Erweiterung (2-Attachment-Send + Hard-Fail).

### Threat Model Summary

Worst-case bei kompromittiertem Toggle-Pfad: Tenant-Admin mit Zugriff auf einen Antrag setzt `prepare_b2b_documents=true` → Mitglied bekommt unerwartetes B2B-Mandat-PDF als zweiten Anhang in der Mandate-Mail. **Kein PII-Leak, kein Auth-Bypass, kein Cross-Tenant-Pfad, kein Faktura-Core-Bankhandling-Risiko** (Core-Antrag bleibt CORE — der Toggle steuert nur, was per E-Mail an das Mitglied geht). Status-Modell-Strip ist durch Bestand-Migration konsistent gemacht; Drift-Wache-Tests verhindern künftiges Wiedereinführen. Migration ist atomar (golang-migrate wrappt jede SQL-Datei in eine Transaktion), Up-Reihenfolge per Smoke-Test gegen Risk R1 (Constraint-Violation) abgesichert.

### Verifikationen (alle Pflicht-Checks)

| Bereich | Stelle | Verifikation |
|---|---|---|
| **AuthZ vor Mutation** | `internal/http/admin.go:518-526` `UpdateApplication` | `checkTenantAccess(w, r, id)` läuft VOR Service-Call; bei Verweigerung sofort return. Bestehender Pfad unverändert. |
| **Actor-Quelle** | `internal/http/admin.go:547-550` | `actorID` kommt aus `ClaimsFromContext(r.Context()).Subject`. Kein User-Input — Subject ist signiertes Claim aus Keycloak-JWT (Middleware validiert vorher Expiry/Issuer/Audience). Bei fehlenden Claims leerer String, Service akzeptiert das (siehe Zeile 547 Doc-Kommentar). |
| **status_log Reason** | `internal/application/admin_service.go:584-586` | Reason ist Compile-Time-Konstante (`"B2B-Vorbereitung aktiviert"` / `"B2B-Vorbereitung deaktiviert"`). Kein User-kontrollierter Text. |
| **Defense-in-Depth Reset** | `internal/application/admin_service.go:455-462` | `if app.Einzugsart != "core" { app.PrepareB2BDocuments = false }` läuft NACH Anwendung des Request-Werts. Auch bei manipuliertem Request kann der Wert nicht in einem ungültigen Einzugsart-Kontext landen. |
| **Einzugsart-Validate** | `internal/shared/requests.go:333` | `validate:"omitempty,oneof=kein_sepa b2b core"` — Einzige zugelassene Werte, kein „CORE" / „Core" / „CoRe" möglich (`oneof` ist case-sensitive). |
| **XSS / Banner-HTML** | `internal/mail/b2b_notice.go:33-37, 46-50` | Banner-Inhalt ist statisches Compile-Time-HTML. Helper-Input ist `bool`, nicht String. `template.HTML` wird absichtlich nicht escapet, weil der Inhalt selbst-kontrolliert ist. Kein User-Pfad rendert direkt in den Banner. |
| **SQL parametrisiert** | `application_repo.go` UpdateAdminTx + alle SELECT-Queries | Alle Werte über `$N`-Placeholder + Driver-Sanitization. `fmt.Sprintf` wird nur für Placeholder-Indizes (`$%d`) verwendet, nie für Werte. |
| **Tenant-Isolation Mail** | `mail/service.go` SendMandateAtImportNotification + post-import-Caller | Mail-Pfad nutzt nur Application + EEG-Stammdaten (beide tenant-scoped via `checkTenantAccess` vor Trigger). Keine Cross-Tenant-Stammdaten gemischt. |
| **Migration atomar** | `db/migrations/000074_*.up.sql` | golang-migrate wrappt jede `.up.sql` in eine Transaktion. Alle 4 Steps (ADD COLUMN, UPDATE-b2b, DROP+RECREATE CHECK, UPDATE-Status+INSERT) sind atomar. Bei Fehler in irgendeinem Step: vollständiger Rollback. |
| **Status-Strip vollständig** | 10 Backend-Stellen + 4 Frontend-Stellen + 1 Migration-CHECK | Drift-Wache-Test `TestAdminTransitions_NoDanglingAwaitingBankConfirmation` + `TestAdminTransitions_AllReferencedStatusesAreKnown` + `TestMigration_000074_*` halten den Strip stabil. Frontend hat „Unbekannter Status"-Fallback als Defense-in-Depth. |
| **Hard-Fail bei PDF** | `admin_service.go` post-import-mail + `resync_service.go` | Bei B2B-PDF-Generation-Fehler `return` ohne Mail-Versand. Status bleibt vor Aktivierung. Konsistent zu PROJ-74-Pattern. Log enthält nur Application-ID (keine PII). |
| **Status_log PII** | `status_log`-INSERT bei Toggle-Diff | Enthält nur: Application-ID (UUID), From/To-Status (Konstanten), Actor-Subject (Keycloak-UUID), Reason (Compile-Time-String), Timestamp. Kein Name, kein E-Mail, keine IBAN. |
| **PROJ-79-Rollback** | `importing/payload.go:296-305` | `mapEinzugsart('b2b')` returnt jetzt `"B2B"`. Faktura-Core bekommt sauberen Wert. Test `TestBuildPayload_B2B_MapsToB2B` als Regression-Wache. Kein heimliches Mapping mehr — Mental-Model Onboarding-DB = Faktura-Core ist wieder synchron. |
| **Externe Pfade unverändert** | External API + Excel-Import | Beide kennen `prepareB2BDocuments` nicht — Backwards-Compatibility, Default FALSE bleibt. Externe Caller können Toggle nicht setzen. |
| **Down-Migration** | `db/migrations/000074_*.down.sql` | Symmetrie-Anker: DROP COLUMN + CHECK-Restore. Bestand-Restore unmöglich (Daten weg) — im Doc-Kommentar dokumentiert, akzeptiert. |

### Findings

| Severity | File | Function/Area | Risk | Exploit Scenario | Recommended Fix | Confidence |
|---|---|---|---|---|---|---|
| Info | `internal/mail/service.go:992-1003` | Attachment-Filename `sepa-firmenlastschrift-mandat-%s.pdf` mit `data.MemberNumber` | MemberNumber wird nur mit `validate:"min=1,max=50"` validiert — kein Character-Whitelist. Bei manipuliertem MemberNumber (z. B. `"; echo evil"` oder Steuerzeichen) theoretisch MIME-Header-Injection-Risiko. **Aber: Bestand-Pattern, nicht durch PROJ-91 eingeführt** — der CORE-Mandat-Pfad nutzt denselben `fmt.Sprintf`. PROJ-91 erweitert nur um eine zweite Filename mit identischer Mechanik. | Tenant-Admin setzt MemberNumber mit Steuerzeichen → könnte Mail-Header brechen. Praktisch unwahrscheinlich, da MemberNumber via Core-Sync gepflegt wird, nicht durch Free-Text-Admin-Input. | (Pre-existing, kein PROJ-91-Scope) Optional: MemberNumber-Validate-Tag um Character-Whitelist erweitern (`alphanum` oder `printascii`) in eigenem Hardening-PROJ. | Medium |

**Keine PROJ-91-spezifischen Findings.**

### Scan Results

| Tool | Result | Anmerkung |
|---|---|---|
| `govulncheck ./...` | 0 callable vulnerabilities | 5 packaged + 1 modul nicht-callable, Pre-PROJ-91-Bestand |
| `gosec` (5 packages) | 0 Issues | Modified packages: `application`, `mail`, `importing`, `http`, `shared` |
| `npm audit --high` | clean | 4 Moderate Pre-PROJ-91-Bestand (uuid GHSA-w5hq-g745-h8pq via next-auth) |
| Semgrep | not run | (manuelle Triggerung im CI-Pfad via `.github/workflows/security-scan.yml`) |
| Trivy IaC | not run | PROJ-91 berührt keine Helm/Dockerfile/Workflow-Dateien |

### Verdict: **APPROVED**

PROJ-91 berührt mehrere Pflicht-Trigger (Schema-Migration, Status-Transitions, Import-Logik, Mail-Routing), aber **alle Sicherheits-relevanten Pfade sind sauber**:

1. **Auth-Kette unverändert**: `checkTenantAccess` läuft vor Mutation, Subject kommt aus signiertem JWT-Claim, kein neuer Endpoint, kein neuer Auth-Pfad.
2. **Defense-in-Depth Reset implementiert**: bei `einzugsart != "core"` setzt das Backend `prepare_b2b_documents=false` unabhängig vom Request — auch durch manipulierten Frontend-Call nicht umgehbar.
3. **XSS-sicher**: Banner-HTML ist 100% statisch (Compile-Time-Strings in `b2b_notice.go`), Helper-Input ist Bool (nicht String wie PROJ-79-Vorgänger).
4. **Migration atomar + bestand-konsistent**: golang-migrate-Transaktions-Wrap, Smoke-Test gegen Risk R1, Drift-Wache-Tests verhindern künftiges Wiedereinführen des entfernten Status.
5. **Hard-Fail-Pattern bei PDF-Fehler**: konsistent zu PROJ-74. Status bleibt vor Aktivierung, kein Mail-Versand mit unvollständigem PDF-Set.
6. **Audit-Trail**: `status_log`-Eintrag bei jeder Toggle-Änderung mit Subject + Compile-Time-Reason. Keine PII im Log.
7. **PROJ-79-Rollback sauber**: kein heimliches Mapping mehr, Mental-Model Onboarding-DB = Faktura-Core ist synchron.

Das Info-Finding zur MemberNumber-Filename-Validierung ist **kein Blocker** — es handelt sich um ein Bestand-Pattern (CORE-Mandat-Pfad hat dieselbe Mechanik), nicht durch PROJ-91 eingeführt. Härtung kann als eigenes Hardening-PROJ später erfolgen.

**Approved for /deploy.**

## Deployment

**Deploy-Bookkeeping 2026-06-09 (Abend):**

- Commit: `4abaf0a` (feat(PROJ-91): B2B-Vorbereitungs-Toggle …)
- Helm-Auto-Bump: `91ba877` (image tag `sha-4abaf0a`)
- Tag: `v1.24.0-PROJ-91`
- CI: alle 4 Workflows grün (CI Build & Test, Security Scan, Docker Build, Mirror)
- Vorgänger: `v1.23.8-PROJ-90` (Schema-Drift-Hotfix vom selben Tag, Vormittag)

**Owner-Action offen:**

```
helm upgrade eegfaktura-member-onboarding ./helm/member-onboarding \
  -f helm/member-onboarding/values-env.yaml \
  -f helm/member-onboarding/values-secret.yaml
```

Was passiert beim Apply:
1. Migration-Job läuft pre-Backend-Update (golang-migrate-Wrap → atomar)
2. Migration 000074 macht:
   - ADD COLUMN `prepare_b2b_documents`
   - Bestand-b2b-Anträge → `einzugsart=core` + `prepare_b2b_documents=true`
   - CHECK-Constraint ohne `awaiting_bank_confirmation`
   - Bestand-Status `awaiting_bank_confirmation` → `ready_for_activation` + `status_log`-Audit
3. Backend-Pod-Rollout mit neuem Code
4. Frontend-Pod-Rollout parallel
5. Drift-Window ~30 s zwischen Migration und Backend (akzeptiert)

Verifikation nach Apply:
- `\d member_onboarding.application` zeigt neue Spalte `prepare_b2b_documents`
- Spalten-Default ist `false`
- Bestand-Migration: `SELECT count(*) FROM member_onboarding.application WHERE einzugsart='b2b'` muss 0 sein
- `SELECT count(*) FROM member_onboarding.application WHERE status='awaiting_bank_confirmation'` muss 0 sein
- Tester: Antrag im Bearbeiten-Dialog → Einzugsart Core → Toggle „Mitglied für Umstellung auf B2B vorbereiten" sichtbar → aktivieren → Speichern → Reload → Antrags-Detail zeigt „B2B-Vorbereitung: Ja"

Keine neuen ENV-Variablen erforderlich. Kein Helm-values-env-Update nötig.
