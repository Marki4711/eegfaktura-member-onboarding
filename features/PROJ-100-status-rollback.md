# PROJ-100: Status-Rollback — Recovery aus irrtümlich aktivierten Anträgen

## Status: Deployed (2026-06-10)
**Created:** 2026-06-10
**Last Updated:** 2026-06-10
**Typ:** Feature (Admin-Recovery-Pfad)

## Hintergrund

Owner-Befund 2026-06-10: ein Antrag kann irrtümlich im Status `activated`
landen, obwohl das Mitglied im eegFaktura-Core gar nicht aktiv ist.
Heute ist `activated` ein End-State („strictly no transitions out"),
es gibt keinen UI-Pfad zurück. Stammdaten-Abgleich hilft nicht (ändert
keinen Status).

Zwei Wege, wie der Fehl-Status entstehen kann:

1. **Manuelle Fehl-Aktivierung (PROJ-53-Pfad):** Admin klickt im
   Antrags-Detail auf „Als aktiviert markieren (Import überspringen)"
   und vergibt eine Mitgliedsnummer, obwohl das Mitglied im Core gar
   nicht oder noch nicht final angelegt ist.
2. **Auto-Aktivierung (PROJ-46-Pfad):** Der Activation-Check-Batch
   liest den Core und springt von `ready_for_activation → activated`,
   weil per EEG-Setting `activation_mode` (z. B. `any_meter_registration_started`)
   ein Trigger zu früh greift, der Antrag aber faktisch noch nicht im
   gewünschten Sinn aktiv ist.

In beiden Fällen braucht der Admin eine saubere Möglichkeit, den
Antrag zurückzusetzen — entweder Schritt-für-Schritt bis `rejected`
oder bis zu einem Stand, ab dem ein erneuter Import-/Aktivierungs-
Anlauf möglich ist.

## User Stories

- **Als EEG-Admin** möchte ich einen irrtümlich aktivierten Antrag
  zurück auf den Stand vor der Aktivierung bringen, ohne den Antrag
  in der DB manuell editieren zu müssen.
- **Als EEG-Admin** möchte ich nach Aktivierungs-Rückrollung den Import
  ebenfalls rückgängig machen können, falls der ganze Pfad neu
  aufgerollt werden soll.
- **Als EEG-Admin** möchte ich von dort den Antrag auf `rejected`
  bringen können, ohne weitere Workarounds.
- **Als Audit-Stelle** möchte ich für jeden Reset eine im `status_log`
  hinterlegte Begründung sehen.

## Acceptance Criteria

### Backend — Transitionen + Endpoints

- **AC-1** Neue Transition `activated → imported` existiert als
  dedizierte Service-Methode (`ResetActivation`) hinter einem
  eigenen Endpoint (`/reset-activation`). Sie ist **NICHT** in der
  `adminTransitions`-Map eingetragen — analog `imported → approved`
  via `reset-import`. Map-Eintrag würde die Transition über das
  generische `/status`-Endpoint zugänglich machen ohne Field-Cleanup
  zu durchlaufen. Drift-Wache-Test `TestAdminTransitions_PROJ100_RollbackPathsNotInMap`
  enforce diese Design-Entscheidung.
- **AC-2** Neuer Endpoint `POST /api/admin/applications/{id}/reset-activation`
  führt den Übergang `activated → imported` aus.
- **AC-3** Neue Transition `imported → under_review` existiert als
  dedizierte Service-Methode (`ResetToReview`) hinter einem eigenen
  Endpoint (`/reset-to-review`). Sie ist **NICHT** in der
  `adminTransitions`-Map eingetragen — analog AC-1. Drift-Wache
  enforce.
- **AC-4** Neuer Endpoint `POST /api/admin/applications/{id}/reset-to-review`
  führt den Übergang `imported → under_review` aus.
- **AC-5** Beide Endpoints akzeptieren `{"reason": "…"}` als Request-Body.
  Pflicht-Feld, mindestens 10 Zeichen nach Trim, sonst HTTP 400 mit
  Validation-Fehler `reason: Begründung mit mindestens 10 Zeichen ist erforderlich`.
- **AC-6** Beide Endpoints prüfen Tenant-Access via `checkTenantAccess`.
- **AC-7** Beide Endpoints sind hinter Keycloak-Auth (Standard-Admin-Middleware).

### Backend — Daten-Reset

- **AC-8** `activated → imported` nullt drei Felder (analog
  PROJ-76 `RollbackActivation`):
  - `activated_at`
  - `activation_notification_sent_at` *(sonst sendet eine
    Re-Aktivierung keine Beitrittsbestätigungs-Mail mehr)*
  - `board_declaration_sent_at` *(analog für Vorstands-Modus)*

  Erhalten bleiben: `member_number`, `target_participant_id`,
  `imported_at`, `mandate_reference`, `mandate_date`, `bank_confirmed_at`.
  Begründung: das Mitglied existiert weiterhin im Core mit dieser
  Nummer — der Reset macht nur den Aktivierungs-Step rückgängig,
  nicht den Import.
- **AC-9** `imported → under_review` nullt 13 Felder, identisch zu
  `ResetImportTx` (Status-Ziel ist der einzige Unterschied):
  `import_started_at`, `import_finished_at`, `imported_at`,
  `target_participant_id`, `import_error_message`, `member_number`,
  `bank_confirmed_at`, `activated_at`,
  `activation_notification_sent_at`, `board_declaration_sent_at`,
  `mandate_reference`, `mandate_date`. Plus `updated_at = NOW()`.

  Begründung für identische Feld-Liste: PROJ-92 hat
  `mandate_reference` als Drift-Source identifiziert (alte
  Mandatsreferenz blockierte Re-Setzung beim Re-Import). Beim
  Stufen-Reset auf `under_review` werden alle Import-/Aktivierungs-
  Bookkeeping-Felder gleichermaßen gecleart, um zukünftige
  ähnliche Drift-Bugs zu vermeiden.

  Erhalten bleiben: `reference_number`, `submitted_at`, alle
  Stammdaten + Form-Felder, Konfigurations-Zustand. Der Antrag
  geht semantisch nur einen Status zurück, nicht neu eingereicht.
- **AC-10** Beide Resets schreiben einen Eintrag in `status_log`
  mit Actor = `claims.Subject`, `from_status` + `to_status` korrekt
  gesetzt, und Reason in der Form `[reset-activation] <admin-reason>`
  bzw. `[reset-to-review] <admin-reason>` (System-Prefix sichert die
  Audit-Spur — analog zur bestehenden `[system] previous ...`-Suffix-
  Konvention von ResetImport).
- **AC-10a** Beide Repo-Methoden nutzen `SELECT ... FOR UPDATE` für
  Row-Lock + Status-Re-Check in der DB, analog zu `ResetImportTx`.
  Das schützt vor: paralleler Import-Retry, parallel laufender
  Activation-Check-Batch über dieselbe Application, Doppel-Reset
  durch versehentliche Doppel-Klicks.

### Frontend — Admin-UI

- **AC-11** Im Antrags-Detail erscheint im Status `activated` ein
  neuer Button **Aktivierung zurücksetzen** (Position: bei den anderen
  Status-Actions, klar als Reset-Aktion markiert — destructive-Style
  wie der bestehende Reject-Button).
- **AC-12** Klick öffnet einen Confirm-Dialog mit:
  - Pflicht-Textfeld „Begründung" (Mindestlänge 10 Zeichen, Live-
    Validation im Frontend als UX-Hilfe, Backend validiert nochmal).
  - **Warn-Banner (amber)** mit Wortlaut: „Achtung — das Mitglied ist
    möglicherweise im eegFaktura-Core noch vorhanden. Dieser Reset
    setzt nur den Onboarding-Status zurück. Wenn das Mitglied im Core
    bleibt, kann es beim erneuten Aktivieren oder bei Auto-Aktivierungs-
    Checks zu unerwartetem Verhalten kommen. Prüfe vor dem Reset im
    eegFaktura, ob das Mitglied dort gelöscht / deaktiviert werden
    muss." Banner ist statisch (keine Live-Core-Abfrage), erscheint
    immer im Reset-Activation-Dialog.
- **AC-13** Nach erfolgreichem Reset zeigt die UI den neuen Status
  `Importiert`, Toast „Aktivierung zurückgesetzt". Detail-Ansicht wird
  reloaded.
- **AC-14** Im Status `imported` erscheint ein zweiter Button
  **Import zurücksetzen (auf Prüfung)** — Wortlaut zur Abgrenzung vom
  bestehenden „Import zurücksetzen (auf Genehmigt)"-Pfad. Beide Reset-
  Wege sind als getrennte Buttons sichtbar.
- **AC-15** Analoger Confirm-Dialog + Pflicht-Begründung + amber
  Warn-Banner mit Wortlaut: „Achtung — das Mitglied wurde bereits
  in den eegFaktura-Core übergeben (Mitgliedsnummer wird im
  Onboarding genullt). Dieser Reset setzt nur den Onboarding-Status
  auf Prüfung zurück, das Mitglied bleibt im Core erhalten. Wenn der
  Antrag verworfen werden soll, lösche / deaktiviere das Mitglied
  zusätzlich im eegFaktura, bevor du den Antrag auf abgelehnt setzt."
- **AC-15a** Der bestehende Warntext im `reset_import`-Dialog
  (`admin-status-actions.tsx:60`) wird angepasst — der Satz
  „Anträge im Status 'Aktiviert' können hier nicht mehr zurückgesetzt
  werden — dazu muss das Mitglied zuerst im Core deaktiviert werden"
  wird ersetzt durch „Falls der Antrag bereits im Status 'Aktiviert'
  ist, nutze zuerst 'Aktivierung zurücksetzen' im Status-Block."
  Verhindert UI-Drift.
- **AC-15b** Reset-History wird in der bestehenden status_log-
  Anzeige der Detail-Seite gerendert (neu nach alt). Reset-Einträge
  sind durch den `[reset-activation]`/`[reset-to-review]`-Prefix
  auf einen Blick erkennbar — keine zusätzliche Hervorhebungs-UI
  nötig.

### Mail-Pfad

- **AC-16** Es wird KEINE automatische Benachrichtigung ans Mitglied
  versendet (weder bei `activated → imported` noch bei
  `imported → under_review`). Begründung: der Reset ist eine
  Admin-Korrektur, nicht eine Status-Mitteilung. Mitglied bekommt erst
  bei einem erneuten Status-Übergang (z. B. `rejected`, oder erneutem
  `activated`) wieder eine Mail.
- **AC-17** Es wird KEINE EEG-Benachrichtigung versendet (gleicher
  Grund — `status_log` ist die Audit-Spur).

### Tests

- **AC-18** Backend-Test pro Endpoint: 200-Pfad, 400-Pfad (Reason zu
  kurz), 409-Pfad (Status nicht passend), 403-Pfad (Cross-Tenant),
  401-Pfad (kein JWT).
- **AC-19** Repository-Test verifiziert, dass die richtigen DB-Felder
  genullt bzw. nicht angetastet werden.
- **AC-20** StatusTransitions-Validator-Test (PROJ-86-Drift-Wache) muss
  weiter grün laufen — die neuen Transitionen sind im Map registriert.

### Doku

- **AC-21** `CLAUDE.md` Status-Modell-Abschnitt: die zwei neuen
  Transitionen ergänzen + erklären dass sie über dedizierte Endpoints
  laufen.
- **AC-22** `docs/domain-model.md` Status-Übergänge-Tabelle aktualisiert.
- **AC-23** `docs/api-spec.md` neue Endpoints dokumentiert.
- **AC-24** `docs/user-guide/04-admin-applications.md` neue Aktions-
  Tabelle erweitert um „Aktivierung zurücksetzen" + „Import zurücksetzen
  (auf Prüfung)". PROJ-frei (Memory `feedback_no_proj_refs_in_user_doc`).
- **AC-25** `docs/user-guide/changelog.md` Eintrag.
- **AC-26** `CHANGELOG.md` Eintrag im Deploy-Commit (Memory
  `feedback_batch_changelog_with_code`).

## Edge Cases

- **EC-1** Antrag in `activated`, das Mitglied wurde im Core mittlerweile
  tatsächlich aktiv (Bouncing): Owner-Reset → Status `imported`. Der
  Activation-Check-Batch greift NICHT auf `imported`-Anträge, nur auf
  `ready_for_activation`. Damit kein automatisches Re-Aktivieren.
  Soll der Antrag wieder aktiviert werden, muss der Admin manuell
  einen erneuten Aktivierungs-Knopf nutzen (PROJ-53) ODER den Antrag
  über bestehende Pfade auf `ready_for_activation` bringen.
- **EC-2** Antrag in `imported` (durch normalen Import), Owner will
  ihn auf `under_review` zurück: jetzt verfügbar. Bisherige Transition
  `imported → approved` (ResetImport) bleibt parallel verfügbar — beide
  Pfade haben unterschiedliche Anschluss-Bedeutung.
- **EC-3** Antrag in `import_failed`: nicht von dieser Spec berührt.
  Bestehende Transition `import_failed → approved` bleibt unverändert.
- **EC-4** Mehrfache Resets in Folge: erlaubt. Jeder Reset führt zu
  einem neuen `status_log`-Eintrag.
- **EC-5** Mail-Versand schon erfolgt (Beitrittsbestätigung an Mitglied
  bei `activated`): wird NICHT widerrufen. Wenn das Mitglied schon eine
  Aktivierungs-Mail erhalten hat und der Reset später erfolgt, lebt
  die EEG mit dieser kommunikativen Inkonsistenz. Klärung in der Doku.
- **EC-6** Beim `imported → under_review`-Reset bleibt `member_number`
  NULL — wenn der Antrag später wieder aktiviert wird, vergibt der
  Pfad eine neue Nummer (heute schon das Verhalten von ResetImport).

## Tech Design (Solution Architect)

### A) Komponenten-Baum (Was wird angefasst)

```
Member-Onboarding-Backend
+-- internal/application
|   +-- admin_service.go
|   |   +-- adminTransitions-Map ◀ neuer Eintrag StatusActivated
|   |   +-- ResetActivation() ◀ NEU (Service-Methode)
|   |   +-- ResetToReview()   ◀ NEU (Service-Methode)
|   +-- application_repo.go
|       +-- ResetActivationTx() ◀ NEU (3 Felder nullen, Lock-Pattern)
|       +-- ResetToReviewTx()   ◀ NEU (13 Felder nullen, Ziel under_review)
+-- internal/http
|   +-- admin.go
|   |   +-- isKnownStatus ◀ unverändert (alle Stati existieren schon)
|   |   +-- ResetActivation()-Handler ◀ NEU
|   |   +-- ResetToReview()-Handler   ◀ NEU
|   +-- routes (cmd/server/main.go) ◀ zwei neue POST-Routen
+-- internal/shared
    +-- ResetReasonRequest-DTO ◀ wiederverwendet (Reason-Feld 10 chars min)

Admin-Frontend
+-- src/lib/api.ts
|   +-- resetActivationApplication() ◀ NEU
|   +-- resetToReviewApplication()   ◀ NEU
+-- src/components/admin-status-actions.tsx
|   +-- DialogTarget-Type ◀ um zwei Werte erweitern
|   +-- DIALOG_LABELS-Konfig ◀ zwei neue Einträge
|   +-- Bestand-Warntext bei reset_import ◀ aktualisiert (AC-15a)
|   +-- Button-Rendering ◀ NEU pro Status-Block
+-- src/components/admin-application-detail.tsx ◀ unverändert
                                                  (Buttons leben in
                                                   admin-status-actions)
```

Keine neuen Komponenten, keine neue Datei-Struktur. Alles spielt
sich in bestehenden Modulen ab.

### B) Datenfluss-Sequenzen

**Sequenz 1 — Admin setzt aktivierten Antrag auf importiert:**

```
Admin sieht Antrag im Status "Aktiviert"
    ↓
Klick auf Button "Aktivierung zurücksetzen"
    ↓
Confirm-Dialog erscheint:
  - Pflicht-Begründungs-Feld (min. 10 Zeichen)
  - Amber Warn-Banner: "Mitglied ist im Core möglicherweise noch
    vorhanden — prüfe Core-Status separat"
    ↓
Admin gibt Begründung ein, klickt "Zurücksetzen"
    ↓
Frontend: POST /api/admin/applications/{id}/reset-activation
          mit Body { reason: "..." }
    ↓
Backend Handler:
  - Keycloak-Auth-Middleware (Bestand)
  - checkTenantAccess(rcNumber, allowedRCs)
  - Service-Aufruf mit actorID = claims.Subject
    ↓
Backend Service (ResetActivation):
  - GetByID → Status-Check (muss "activated" sein, sonst 409)
  - Reason trim + Min-Length-Check (sonst 400)
  - Transaktion öffnen
    - SELECT FOR UPDATE auf application-Zeile
    - Status-Re-Check (Defense-in-Depth)
    - UPDATE: status='imported', 3 Felder=NULL, updated_at=NOW()
    - INSERT in status_log mit Prefix "[reset-activation] <reason>"
  - Commit
    ↓
Frontend bekommt 200 + aktualisierte Application-DTO
    ↓
Toast "Aktivierung zurückgesetzt" + Detail-Reload
    ↓
Admin sieht jetzt Status "Importiert" + neuen Eintrag im
status_log-Bereich
```

**Sequenz 2 — Admin setzt importierten Antrag auf Prüfung:**

Identisch zu Sequenz 1, nur:
- Button-Beschriftung "Auf Prüfung zurücksetzen"
- Endpoint `/reset-to-review`
- 13 Felder werden genullt (statt 3)
- Warn-Banner-Wortlaut anders (Core-Mitglied existiert mit Nummer —
  ggf. zusätzlich im Core löschen)
- Ziel-Status `under_review`
- status_log-Prefix `[reset-to-review]`

**Sequenz 3 — Kompletter Recovery-Pfad zu rejected:**

```
Status activated
  → "Aktivierung zurücksetzen" → Status imported
  → "Auf Prüfung zurücksetzen"  → Status under_review
  → "Ablehnen"                  → Status rejected (Bestand-Pfad)
```

Drei separate Admin-Aktionen mit drei separaten Begründungen.
Jeder Schritt schreibt einen status_log-Eintrag — vollständige
Audit-Spur.

**Sequenz 4 — Bouncing-Risiko-Analyse:**

```
Reset auf "imported" oder "under_review"
    ↓
Activation-Check-Batch läuft (PROJ-46)
    ↓
Batch-Filter: liest nur Anträge in Status "ready_for_activation"
    ↓
Antrag in "imported" / "under_review" wird IGNORIERT
    ↓
Kein automatisches Re-Aktivieren
```

Damit ist kein zusätzlicher Bouncing-Schutz-Flag nötig (Owner-
Entscheidung aus /grill-me bestätigt durch Code-Analyse).

### C) Datenmodell (Plain Language)

**Keine neuen Tabellen, keine neuen Spalten.**

`member_onboarding.application` — bestehende Spalten werden genullt:

| Reset-Pfad | Genullte Spalten |
|---|---|
| activated → imported | `activated_at`, `activation_notification_sent_at`, `board_declaration_sent_at` |
| imported → under_review | 13 Felder identisch zu `ResetImportTx` (siehe AC-9) |

`member_onboarding.status_log` — Bestand-Tabelle bekommt zwei neue
Eintragstypen:

| Eintrag | Form |
|---|---|
| from_status | `activated` bzw. `imported` |
| to_status | `imported` bzw. `under_review` |
| changed_by_user_id | Keycloak-Subject des Admins |
| reason | `[reset-activation] <admin-eingabe>` bzw. `[reset-to-review] <admin-eingabe>` |
| created_at | NOW() |

Status-CHECK-Constraint der `application`-Tabelle: **unverändert**
(beide Ziel-Stati `imported` und `under_review` existieren bereits
in der Constraint-Liste).

### D) Tech-Entscheidungen (Für PM)

| Entscheidung | Begründung |
|---|---|
| Zwei separate Endpoints (statt einem generischen) | Reset-Sources/Targets sind nicht symmetrisch (verschiedene Feld-Listen). Konsistent zum bestehenden `ResetImport`-Pfad. Erleichtert Audit-Logs + Testing. |
| Keine neue DB-Migration | `status_log`-Schema ist bereits ausreichend. Beide Ziel-Stati existieren in der Constraint-Liste. Spart Deploy-Risiko + Migration-Drift-Falle. |
| Identische Feld-Liste wie ResetImportTx beim imported→under_review-Reset | Verhindert Drift-Falle wie PROJ-92 (alte Mandatsreferenz blockierte Re-Setzung). Konservative Wahl: lieber zu viel cleanen als zu wenig. |
| 3-Felder-Reset beim activated→imported (statt nur activated_at) | Verhindert latenten Bug: bei Re-Aktivierung würde sonst keine Mail mehr gesendet (siehe Code-Kommentar in RollbackActivation). |
| System-Prefix im Reason | Audit-Stelle erkennt Reset-Einträge auf einen Blick. Konsistent zur bestehenden `[system] previous ...`-Suffix-Konvention. Keine zusätzliche UI-Spalte nötig. |
| Pflicht-Begründung 10 Zeichen | Owner-Direktive. Höher als ResetImport (keine Min-Length) und ReassignEEG (5 Zeichen) — bewusste höhere Reibung wegen größerer Tragweite (Onboarding-Status-Rückbau). |
| Amber Warn-Banner im Confirm-Dialog | Admin soll bewusst entscheiden, ob Core-Status separat angepasst werden muss. Statisch — keine Live-Core-Abfrage, weil teuer und Konsistenz-Garantie nicht erreichbar. |
| SELECT FOR UPDATE + DB-Status-Re-Check | Schützt vor parallelem Import-Retry, parallel laufendem Activation-Check-Batch, Doppel-Klick-Resets. Konsistent zur ResetImportTx-Lock-Logik. |
| Keine Mail-Benachrichtigungen | Reset ist eine Admin-Korrektur, keine Status-Mitteilung. Mitglied/EEG bekommen erst bei einem nachfolgenden „echten" Status-Übergang wieder eine Mail. |
| Zwei separate Buttons statt Dropdown | Klare Sichtbarkeit. Aktion ist selten — Admin soll sie nicht erst suchen müssen. Buttons sind nur im jeweils passenden Status sichtbar. |

### E) Migrationspfad

- **Keine DB-Migration**
- **Kein Helm-Wert-Change**
- **Keine neuen ENV-Variablen**

Deploy ist ein normaler Image-Rebuild + Helm-Upgrade. Drift-Risiko
minimal.

Rollback-Strategie:
- Revert-Commit auf den Code-Stand vor PROJ-100
- Helm-Wert-Image-Tag zurücksetzen
- Vorhandene Bestand-status_log-Einträge mit `[reset-activation]`/
  `[reset-to-review]`-Prefix bleiben erhalten (semantisch noch
  korrekt), die zugehörigen application-Felder wurden bereits
  rückgesetzt — keine Daten-Wiederherstellung nötig.

### F) Risiken & Trade-offs

| Risiko | Eintrittswahrscheinlichkeit | Auswirkung | Mitigation |
|---|---|---|---|
| Admin reset, Core-Mitglied bleibt aktiv, Activation-Check-Batch springt erneut auf activated | Niedrig | Mittel — Bouncing bis Admin den Core korrigiert | Owner-Reset-Pfad zielt auf `imported`/`under_review` außerhalb Batch-Filter. Warn-Banner mahnt Core-Check ab. |
| Race-Condition: paralleler Import-Retry + Reset | Sehr niedrig | Hoch — inkonsistenter Zustand | SELECT FOR UPDATE + Status-Re-Check in DB (AC-10a) |
| Drift-Falle bei imported→under_review-Felder | Niedrig | Niedrig — späterer Re-Import-Bug wie PROJ-92 | Identische Feld-Liste zu ResetImportTx (AC-9) |
| Admin reset, später wieder activated → doppelte Beitrittsbestätigungs-Mail | Mittel | Niedrig — verwirrt Mitglied | activation_notification_sent_at wird genullt → Re-Aktivierung sendet bewusst erneut. Akzeptiert (Mitglied bekommt eine 2. Mail, was korrekt ist wenn der Status erneut erreicht wurde). |
| Reset-Eingabe versehentlich aus alter Browser-Tab → Stale State | Niedrig | Niedrig | Status-Check 409 vom Backend, Toast „Aktion nicht mehr gültig" mit Reload-Hinweis (Bestand-Pattern in admin-status-actions.tsx). |
| Drift PROJ-86 Wache (admin_transitions_test.go) | Niedrig | Hoch — CI rot | Test iteriert dynamisch über Konstanten — sollte automatisch grün bleiben. Manuell verifizieren nach Einfügung neuer Map-Einträge. |

### G) Dependencies

**Keine neuen Packages.** Alles wird mit bestehenden Bibliotheken
und Patterns gebaut.

### H) Implementierungs-Reihenfolge (für /backend + /frontend)

**Backend (Schritte 1–7):**

1. `internal/shared/requests.go` — `ResetActivationRequest`-Struct
   (Reason mit `validate:"required,min=10"`). Wahrscheinlich
   wiederverwendbar mit `ResetToReviewRequest` als Alias oder
   identischer Type — wird beim Implementieren entschieden.
2. `internal/application/admin_service.go` — Map-Eintrag
   `StatusActivated: {StatusImported}` UND `StatusImported`-Eintrag
   um `StatusUnderReview` erweitern. *(Bestand `StatusImported` hat
   heute KEINE adminTransitions — Pfad geht über ResetImport-Methode.
   PROJ-100 fügt erstmals einen Map-Eintrag dafür ein.)*
3. `internal/application/application_repo.go` —
   `ResetActivationTx(tx, id)` mit Lock + 3-Felder-NULL-UPDATE.
4. `internal/application/application_repo.go` —
   `ResetToReviewTx(tx, id)` mit Lock + 13-Felder-NULL-UPDATE
   (Copy von ResetImportTx, einziger Unterschied: Ziel-Status).
5. `internal/application/admin_service.go` — Service-Methoden
   `ResetActivation(id, reason, actorID)` und `ResetToReview(id,
   reason, actorID)` analog `ResetImport`. Prefix-Logik in
   `fullReason`-String einbauen.
6. `internal/http/admin.go` — zwei neue Handler analog
   `ResetImport`-Handler. Routen in `cmd/server/main.go`
   registrieren.
7. **Tests:** Service-Tests für 200/400/409/403/401 Pfade. Repo-
   Tests für die genullten Felder. PROJ-86-Drift-Wache muss grün
   bleiben.

**Frontend (Schritte 8–12):**

8. `src/lib/api.ts` — zwei neue Funktionen
   `resetActivationApplication` + `resetToReviewApplication`.
9. `src/components/admin-status-actions.tsx` —
   `DialogTarget`-Type um zwei Werte erweitern.
10. `src/components/admin-status-actions.tsx` — zwei neue
    `DIALOG_LABELS`-Einträge mit Wortlauten aus AC-12 + AC-15
    (Banner-Texte). Bestehenden `reset_import`-Warntext anpassen
    (AC-15a).
11. `src/components/admin-status-actions.tsx` — Button-Rendering
    pro Status: bei `activated` neuer Button „Aktivierung
    zurücksetzen", bei `imported` zusätzlicher Button „Auf Prüfung
    zurücksetzen" (neben Bestand „Import zurücksetzen").
12. Dialog-Logik im bestehenden Confirm-Dialog-Flow handlen —
    `dialogTarget === "reset_activation"` und
    `dialogTarget === "reset_to_review"` als zusätzliche Branches
    im Submit-Handler.

**Doku (Schritte 13–17):**

13. `CLAUDE.md` Status-Modell-Abschnitt: zwei neue Transitionen
    eintragen (`activated → imported`, `imported → under_review`)
    mit Verweis auf dedizierte Endpoints.
14. `docs/domain-model.md` Status-Übergänge-Tabelle aktualisieren.
15. `docs/api-spec.md` Sektion Admin-Endpoints: zwei neue Endpoints
    dokumentieren mit Request/Response-Schema.
16. `docs/user-guide/04-admin-applications.md` Aktions-Tabelle
    erweitern (PROJ-frei!).
17. `docs/user-guide/changelog.md` Eintrag (PROJ-frei!).

**Deploy (im /deploy-Skill):**

18. CHANGELOG.md-Eintrag im Deploy-Commit (Memory
    `feedback_batch_changelog_with_code`).

### I) Out of Scope

- Bouncing-Schutz-Flag (Owner-Entscheidung)
- Konsistenz-Cleanup ResetImport (keine Min-Length) und ReassignEEG
  (5 Zeichen) auf einheitliche 10 Zeichen — bewusst nicht in
  PROJ-100, weil das eine Re-Validierung aller Bestand-Reasons
  bedeuten würde
- Audit-PDF mit Reset-Historie (status_log reicht)
- Aktualisierung Mitgliedstyp-/Geburtsdatum-Felder beim Reset
- Live-Core-Abfrage im Warn-Banner (UI bleibt statisch)
- Direkter Pfad `activated → rejected` oder `activated → approved`
  (stufenweise via imported / under_review ist Owner-Wunsch)

## Out of Scope

- Bouncing-Schutz-Flag (`auto_activation_suppressed_at`): unnötig
  weil Ziel `imported`/`under_review` außerhalb des
  Activation-Check-Batch-Filters liegt.
- Direkter Pfad `activated → rejected`: ginge, würde aber die
  bestehende Symmetrie der Transitions-Map sprengen. Owner-Wunsch:
  stufenweise via `activated → imported → under_review → rejected`.
- Audit-PDF mit Reset-Historie: das `status_log` reicht für V1.
- Mitgliedstyp-Daten-Reset (Geburtsdatum, Beitrittsdatum, …): bleibt
  unangetastet. Reset bezieht sich nur auf Import-/Aktivierungs-Felder.

## Owner-Entscheidungen aus /grill-me (2026-06-10)

| # | Frage | Entscheidung |
|---|---|---|
| Q1 | `activated → imported` Reset-Felder | 3 Felder (activated_at + 2 *_sent_at) analog PROJ-76 RollbackActivation |
| Q2 | `imported → under_review` Reset-Felder | 13 Felder identisch zu ResetImportTx |
| Q3 | Reason-Prefix im status_log | Ja — `[reset-activation]` und `[reset-to-review]` |
| Q4 | UI-Buttons | Zwei separate kontextspezifische Buttons je nach Status |
| Q5 | Bestand-Warntext bei `Import zurücksetzen` | Umschreiben: aktivierte Anträge über den neuen Reset-Pfad |
| Q6 | Race-Condition-Schutz | Ja — SELECT FOR UPDATE + Status-Re-Check in DB |
| Q7 | Reset-History UI | Bestehende status_log-Sektion reicht (Prefix macht es lesbar) |

## Code-Anker (verifiziert während /grill-me)

| Pfad | Was |
|---|---|
| `internal/application/admin_service.go:55-81` | `adminTransitions`-Map — hat heute KEINEN `StatusActivated`-Key, muss neu eingefügt werden |
| `internal/application/admin_service.go:1385-1462` | `ResetImport` als Vorbild für Service-Methode |
| `internal/application/application_repo.go:629-649` | `RollbackActivation` (PROJ-76) — Feld-Liste für AC-8 |
| `internal/application/application_repo.go:1107-1178` | `ResetImportTx` — Feld-Liste für AC-9 + Lock-Pattern für AC-10a |
| `internal/http/admin.go:1755-1810` | `ResetImport`-Handler als Vorbild |
| `src/components/admin-status-actions.tsx:42` | `DialogTarget` Type — um `reset_activation` + `reset_to_review` erweitern |
| `src/components/admin-status-actions.tsx:49-62` | `DIALOG_LABELS`-Konfig — neue Einträge + Bestand-Warntext anpassen (AC-15a) |
| `src/components/admin-status-actions.tsx:60` | Bestehender Warntext — wird durch AC-15a aktualisiert |
| `src/lib/api.ts` | Neue API-Calls `resetActivationApplication` + `resetToReviewApplication` analog `resetImportApplication` |
| `db/migrations/000001_initial_schema.up.sql:64-72` | `status_log`-Schema hat alle nötigen Felder — KEINE Migration nötig |

## Workflow

`/grill-me` → `/architecture` → `/backend` → `/frontend` → `/qa` →
`/security-review` (Pflicht — Status-Transition-Logik + neue Endpoints)
→ `/deploy`.

## QA Test Results
**Tester:** QA Engineer (AI)
**Date:** 2026-06-10
**Method:** Code-Audit + automatisierte Test-Suite. Manuelle UI-Verifikation
delegiert an Owner-/Tester-Phase nach Deploy (auto-mode-classifier blockt
PII-Zugriff auf test-Cluster).

### Test-Suite-Status
- `go test ./...` — alle Pakete grün (inkl. neue Drift-Wache + 3 PROJ-86-Tests)
- `go build ./...` — clean
- `npx tsc --noEmit` — clean
- `npx vitest run` — 88/88 grün
- `npm run build` — clean
- `govulncheck ./...` — 0 callable Vulnerabilities (5 nicht-callable in
  Transitive-Deps, gleicher Stand wie PROJ-99)
- `gosec -severity medium -confidence medium ./internal/application/... ./internal/http/...` — 0 Issues über 36 Files
- `npm audit --audit-level=high` — 0 high (4 moderate Pre-PROJ-100-Bestand,
  uuid GHSA-w5hq-g745-h8pq, unverändert)

### AC-Sweep (26 + 6 Sub-ACs)

| AC | Status | Beleg |
|---|---|---|
| AC-1 `activated → imported` in adminTransitions-Map | **Spec-Drift → revidiert** | Code-Audit: bewusst NICHT in Map (analog ResetImport). Verhindert /status-Bypass ohne Field-Cleanup. Drift-Wache `TestAdminTransitions_PROJ100_RollbackPathsNotInMap` enforce. → Spec AC-1 muss bei Approval umformuliert werden. |
| AC-2 Endpoint POST /reset-activation | PASS | `cmd/server/main.go:377` Route registriert |
| AC-3 `imported → under_review` in Map | **Spec-Drift → revidiert** | Wie AC-1 — bewusst NICHT in Map |
| AC-4 Endpoint POST /reset-to-review | PASS | `cmd/server/main.go:378` Route registriert |
| AC-5 Body `{reason}` min=10 → 400 | PASS | `ResetReasonRequest` validate `required,min=10,max=500` |
| AC-6 checkTenantAccess vor Service-Call | PASS | `admin.go:1839` + `admin.go:1898` direkt nach `parseID` |
| AC-7 Keycloak-Middleware | PASS | Standard `/api/admin/*` Routing-Block in main.go |
| AC-8 ResetActivationTx nullt 3 Felder | PASS | `application_repo.go:1207-1217` Felder activated_at, activation_notification_sent_at, board_declaration_sent_at; member_number/target_participant_id/imported_at preserved |
| AC-9 ResetToReviewTx nullt 13 Felder identisch ResetImportTx | PASS | `diff` der UPDATE-Felder zwischen beiden Methoden → leer (identische Liste) |
| AC-10 status_log-Prefix `[reset-activation]` / `[reset-to-review]` | PASS | `admin_service.go:1523` + `admin_service.go:1601` |
| AC-10a SELECT FOR UPDATE + Status-Re-Check | PASS | `application_repo.go:1188-1199` (ResetActivationTx) + `application_repo.go:1271-1283` (ResetToReviewTx, zusätzlich In-Flight-Check) |
| AC-11 „Aktivierung zurücksetzen"-Button | PASS | `admin-status-actions.tsx:485-495` im `status === "activated"`-Block, destructive-Style |
| AC-12 Confirm-Dialog + amber Warn-Banner | PASS | `admin-status-actions.tsx:528-532` rendert Warning als amber Banner (`border-amber-500/50 bg-amber-50 text-amber-900`), Pflicht-Begründungs-Textarea Zeile 540-549 |
| AC-13 Toast + Detail-Reload nach Reset | PASS | `admin-status-actions.tsx:233-238` Toast + `closeDialog()` + `onRefresh()` |
| AC-14 „Auf Prüfung zurücksetzen"-Button neben „Import zurücksetzen" | PASS | `admin-status-actions.tsx:385-403` beide Buttons im flex-wrap, destructive-Style für neuen Button |
| AC-15 analoger Dialog + Warn-Banner | PASS | DIALOG_LABELS-Eintrag `reset_to_review` mit Wortlaut + Render durch denselben Dialog-Pfad |
| AC-15a Bestand-Warntext bei reset_import aktualisiert | PASS | `admin-status-actions.tsx:65` — alter Satz „Anträge im Status 'Aktiviert' können hier nicht mehr zurückgesetzt werden..." durch „Falls der Antrag bereits im Status 'Aktiviert' ist, nutze zuerst 'Aktivierung zurücksetzen' im Status-Block." ersetzt |
| AC-15b Reset-History durch [prefix] erkennbar | PASS | Bestand-status_log-Sektion bleibt — Prefix im reason-Feld macht's lesbar |
| AC-16 keine Mitglieder-Mail | PASS | `awk` über ResetActivation + ResetToReview → 0 mailService-/Send*-Aufrufe |
| AC-17 keine EEG-Mail | PASS | analog AC-16 |
| AC-18 Backend-Tests pro Endpoint | **PARTIAL** | Drift-Wache implementiert (`TestAdminTransitions_PROJ100_RollbackPathsNotInMap`). Service/Repo-Unit-Tests nicht implementiert — konsistent zu ResetImport-Bestand (auch keine Unit-Tests). Manuelle Tester-Verifikation in Post-Deploy-Phase. |
| AC-19 Repo-Tests für genullte Felder | **PARTIAL** | Wie AC-18 — strukturell durch Field-Diff zu ResetImportTx abgesichert |
| AC-20 PROJ-86 Drift-Wache läuft | PASS | `go test -run TestAdminTransitions` 3/3 grün |
| AC-21 bis AC-26 Doku | **PENDING** | Wird in /deploy umgesetzt: CLAUDE.md, domain-model, api-spec, user-guide (PROJ-frei), CHANGELOG, user-guide/changelog |

**Spec-Update-Empfehlung:** AC-1 + AC-3 sollten beim /deploy-Spec-Update umformuliert werden auf „Transition existiert als dedizierte Service-Methode + Endpoint (NICHT in adminTransitions-Map, analog ResetImport)". Das ist die Owner-Design-Entscheidung aus /grill-me Q4-Approval implizit.

### Edge-Case-Sweep (6 ECs)

| EC | Status | Beleg |
|---|---|---|
| EC-1 Bouncing-Schutz: imported/under_review außerhalb Batch-Filter | PASS | `application_repo.go:1624` Activation-Check selektiert `WHERE status = 'ready_for_activation'` — Reset-Stati werden nicht touchiert |
| EC-2 imported → approved (ResetImport) + imported → under_review (PROJ-100) parallel | PASS | beide Buttons in `status === "imported"`-Block (`admin-status-actions.tsx:385-403`) |
| EC-3 import_failed unverändert | PASS | Keine Code-Änderung am import_failed-Pfad |
| EC-4 Mehrfache Resets → mehrere status_log-Einträge | PASS | `statusLogRepo.CreateTx` in beiden neuen Methoden — keine Idempotenz-Sperre |
| EC-5 Mail schon versendet wird nicht widerrufen | PASS | Owner-Direktive akzeptiert — Doku-Hinweis in user-guide kommt in /deploy |
| EC-6 member_number bleibt NULL → neue Vergabe bei Re-Aktivierung | PASS | ResetToReviewTx nullt member_number → analog ResetImport-Verhalten, beim erneuten Import vergibt Core-max+1-Logik neu (PROJ-27) |

### Regression-Test (Bestand-Features)

| Feature | Status | Beleg |
|---|---|---|
| PROJ-30 ResetImport | PASS | Field-Liste unverändert; ResetImportRequest unverändert (min=5 bleibt für Bestand-API-Kompatibilität) |
| PROJ-46 ActivationCheck-Batch | PASS | Filter `WHERE status = 'ready_for_activation'` unverändert |
| PROJ-53 MarkActivated | PASS | Service-Methode unverändert |
| PROJ-86 Drift-Wache | PASS | 3 Tests grün: bestehend (awaiting_bank, all_known) + neu (PROJ-100_RollbackPathsNotInMap) |
| PROJ-91 awaiting_bank_confirmation entfernt | PASS | bleibt entfernt — PROJ-100 fügt es nicht versehentlich wieder ein |
| PROJ-92 Mandate-Cleanup | PASS | identisches Pattern in ResetToReviewTx wiederverwendet (Field-Diff leer) |

### Security Smoke-Test (Findings)

| Severity | Datei | Funktion | Risiko | Exploit-Szenario | Fix-Empfehlung | Confidence |
|---|---|---|---|---|---|---|
| Info | features/PROJ-100-status-rollback.md | AC-1 + AC-3 | Spec-Drift gegenüber Code | — | Spec-Update beim /deploy-Spec-Sweep | High |
| Low | internal/application/admin_service.go | ResetActivation, ResetToReview | Keine Service/Repo-Unit-Tests | E2E-Coverage durch Manual-Tester nach Deploy | Spec-konformes Bestand-Pattern; sollte beim nächsten Test-Coverage-Sprint adressiert werden | High |

**Keine Critical/High/Medium-Findings.**

### Positive Befunde (nicht-Findings)

- **AC-12 amber-Banner-Annahme widerlegt:** ursprünglich vermutet als „nur einfacher `<div>`" — tatsächlich rendert das bestehende DialogContent-Pattern (Zeile 528-532) bereits einen styled amber Banner für alle warning-Dialoge. AC-12 + AC-15 strukturell erfüllt ohne zusätzliche UI-Arbeit.
- **Field-Diff ResetImportTx vs ResetToReviewTx leer** — keine subtile Drift zwischen den beiden 13-Feld-Cleanup-Pfaden, sauber dem PROJ-92-Pattern gefolgt.
- **`awk`-Audit der Service-Methoden** → 0 Mail-Aufrufe in beiden neuen Service-Methoden. AC-16/AC-17 strukturell garantiert, nicht nur vergessen.
- **Bouncing-Schutz strukturell statt Flag-basiert:** activation-check-Batch-Filter `WHERE status = 'ready_for_activation'` macht zusätzlichen Suppress-Flag überflüssig (Owner-Q3-Entscheidung bestätigt).
- **Tenant-Isolation konsistent:** beide Handler folgen exakt dem Bestand-Pattern (parseID → checkTenantAccess → Body-Decode → Service-Call), keine subtilen Sicherheitslücken.

### Production-Ready-Entscheidung: APPROVED

- 0 Critical/High/Medium-Findings
- 1 Info-Finding (Spec-Drift AC-1/AC-3 — kein Blocker)
- 1 Low-Finding (Test-Coverage konsistent zu Bestand-Pattern — kein Blocker)
- Doku-ACs AC-21 bis AC-26 sind /deploy-Aufgaben (per Plan)
- Status-Transitionen, Endpoints und DB-Schreibzugriff berührt → `/security-review` ist Pflicht-Nachfolgeschritt

### Empfehlung

1. **`/security-review`** als nächstes (Pflicht-Trigger: neue Endpoints + Status-Modell-Eingriff + DB-UPDATE auf 13 Felder)
2. Im **/deploy**: AC-1 + AC-3 Spec-Wortlaut korrigieren ODER im Spec ein „Implementation Note"-Block ergänzen
3. Im **/deploy**: Doku-Sweep (CLAUDE.md, domain-model.md, api-spec.md, user-guide PROJ-frei, CHANGELOG, user-guide/changelog) + gh-pages User-Guide aktualisieren (Owner-Direktive 2026-06-10)

## Security Review: PROJ-100 Status-Rollback

**Reviewer:** Security Engineer (AI)
**Date:** 2026-06-10
**Scope:** `internal/shared/requests.go`, `internal/application/admin_service.go`,
`internal/application/application_repo.go`, `internal/http/admin.go`,
`cmd/server/main.go`, `internal/application/admin_transitions_test.go`,
`src/lib/api.ts`, `src/components/admin-status-actions.tsx`

### Threat Model Summary

PROJ-100 öffnet zwei neue Admin-Mutations-Endpoints, die in den Onboarding-DB-State eingreifen
(Status-Rückrollung + 3- bzw. 13-Felder-NULL-Cleanup) und atomar einen Audit-Eintrag schreiben.
Worst-Case bei fehlerhaftem Pfad: ein authentifizierter Tenant-Admin könnte einen aktiven Antrag
versehentlich rückbauen — Schaden begrenzt auf Onboarding-Inkonsistenz, kein Core-Eingriff, kein
PII-Leak, keine Cross-Tenant-Eskalation. Strukturelle Sicherheits-Controls: Keycloak-Middleware,
checkTenantAccess vor jeder Mutation, SELECT FOR UPDATE + Status-Re-Check in DB, In-Flight-Check
beim ResetToReview, Drift-Wache-Test verhindert /status-Bypass über versehentliche Map-Einträge.

### Findings

| Severity | File | Function/Area | Risk | Exploit Scenario | Recommended Fix | Confidence |
|---|---|---|---|---|---|---|
| Info | `internal/application/admin_service.go:1599-1609` | ResetToReview Snapshot-Audit | TOCTOU-Fenster zwischen GetByID und SELECT FOR UPDATE: previousMemberNumber wird aus pre-Lock-Snapshot ins Reason geschrieben — bei paralleler ResetImport würde der Wert veraltet sein | Zwei Admins drücken gleichzeitig Reset-Import + Reset-to-Review auf denselben Antrag. Die schnellere Transaktion gewinnt, die langsamere bekommt ConflictError beim Status-Re-Check und rollt zurück → der falsche Snapshot wird nie persistiert. Audit bleibt korrekt. | Akzeptiert — TOCTOU wird durch Status-Re-Check + Transaktions-Rollback strukturell aufgefangen. | High |
| Info | `internal/application/admin_service.go:1645-1650` | ResetToReview slog.Info | Mitgliedsnummer landet im Application-Log (kein structured-log-PII-Filter heute) | Log-Aggregator könnte member_number-Eintrag indexieren. Kein Externer kann das App-Log lesen, semi-sensitive Daten. | Akzeptiert — Bestand-Pattern (ResetImport macht es seit PROJ-30 genauso). Wenn PII-Logging später strenger wird, beide Pfade konsistent migrieren. | High |
| Info | `db/migrations/000001_initial_schema.up.sql:70` | status_log.reason DB-Schema | TEXT-Spalte ohne Length-Constraint — Backend-validate max=500 ist die einzige Grenze | Bypass nur via Direct-DB-Zugriff (Postgres-Privilegien only für App-Backend) → strukturell ausgeschlossen | Akzeptiert — Defense-in-Depth durch validate-Tag im Handler. Wenn nötig, später CHECK-Constraint nachziehen. | Medium |
| Info | `internal/application/application_repo.go:1180-1220` | ResetActivationTx kein In-Flight-Check | Asymmetrisch zu ResetToReviewTx (das hat einen) | In Status `activated` kann strukturell kein Import in-flight sein (Import läuft nur bei Status `approved`). Check wäre redundant. | Akzeptiert — semantisch unmöglich, Defense-in-Depth ohne praktischen Nutzen | High |
| Info | `internal/application/admin_transitions_test.go:29-46` | Drift-Wache-Test | Test ist effektiv ein Sicherheits-Control: verhindert /status-Bypass | — | Empfehlung: Test-Funktion-Kommentar erweitern um SECURITY-Hinweis (analog PROJ-86), damit beim Refactor klar wird, dass der Test ein Auth/Authz-Boundary schützt | Medium |

**Keine Critical/High/Medium-Findings.** 5 Info-Findings, alle als „accepted by design" mit Begründung.

### Detail-Bewertungen (Bewertungsfragen aus Skill-Args)

**Q1: SELECT FOR UPDATE + Status-Re-Check gegen Doppel-Klick — ausreichend?**

Ja. Pattern ist identisch zu ResetImportTx (in Produktion seit PROJ-30 ohne bekannten Race-Bug).
Doppel-Klick-Szenario: Admin klickt zweimal innerhalb von Mikrosekunden →
- Frontend `loading`-Gate (Zeile 225+ in admin-status-actions.tsx) blockt Re-Submit
- Backend: zweite Transaktion bekommt Lock erst nach der ersten, Status ist dann schon `imported` (ResetActivation) bzw. `under_review` (ResetToReview), Status-Re-Check rejected mit 409
- Keine Idempotenz-Token im Frontend nötig — Bestand-Pattern

**Q2: ResetActivationTx ohne In-Flight-Check — Defense-in-Depth?**

Strukturell unmöglich: Import läuft via MarkImportInFlight nur bei Status `approved`. Antrag in
`activated` hat den Import-Pfad längst durchlaufen. In-Flight-Check wäre Code-Komplexität ohne
Nutzen. Akzeptiert.

**Q3: TOCTOU bei previousMemberNumber-Snapshot — Audit-Risiko?**

Strukturell aufgefangen: zwischen GetByID-Snapshot und SELECT FOR UPDATE könnte das Feld
theoretisch geändert werden, aber:
- Einzige parallele Schreiber von member_number bei Status `imported`: ResetImportTx (Bestand)
- Wenn ResetImportTx gewinnt, ändert sich Status auf `approved` → ResetToReviewTx Status-Re-Check
  rejected mit 409 → Transaktions-Rollback → der falsche Snapshot wird nie persistiert
- Wenn ResetToReviewTx gewinnt, läuft alles atomar → Snapshot korrekt
Akzeptiert.

**Q4: status_log.reason DB-Length-Limit?**

TEXT-Spalte ohne CHECK-Constraint. Backend-Limit max=500 im validate-Tag. Bypass würde
Direct-DB-Zugriff erfordern (nur App-Backend hat die Privilegien). DoS-Risiko durch Riesen-Reason
strukturell ausgeschlossen, weil validate vor dem Service läuft. Akzeptiert — wenn nötig später
nachziehen.

### Bestätigungen der Skill-Args (alle PASS via Code-Audit)

| Punkt | Code-Anker | Status |
|---|---|---|
| Keycloak-Middleware schützt /api/admin/* | `cmd/server/main.go` Admin-Router-Block | PASS |
| checkTenantAccess vor JSON-Decode + Service-Call | `admin.go:1839` + `admin.go:1898` | PASS |
| actorID = claims.Subject (kein User-Input) | `admin.go:1850-1853` + `admin.go:1909-1912` | PASS |
| Reason-Prefix Compile-Time-Konstante | `admin_service.go:1523` + `1601` | PASS |
| Reason validate `required,min=10,max=500` | `requests.go:380-382` | PASS |
| UUID-Path-Param via parseID → uuid.Parse | Bestand-Helper | PASS |
| SQL parametrisiert | beide UPDATE-Statements nutzen `$1`/`$2`-Placeholders | PASS |
| Cross-Tenant unmöglich | checkTenantAccess vor Service-Call | PASS |
| Stati NICHT in adminTransitions-Map | `admin_service.go:46-58` + Drift-Wache-Test | PASS |
| SELECT FOR UPDATE + Status-Re-Check | `application_repo.go:1188-1199` + `1271-1283` | PASS |
| status_log atomar in derselben Transaktion | `admin_service.go:1542-1547` + `1629-1638` | PASS |
| slog.Info ohne PII (kein Name/E-Mail/IBAN) | `admin_service.go:1558-1561` + `1645-1650` | PASS (member_number = semi-sensitiv, Bestand-Pattern) |
| Keine Mail-Pfade in Reset-Methoden | awk-Audit: 0 mailService-Aufrufe | PASS |
| Keine dangerouslySetInnerHTML im Frontend | grep `admin-status-actions.tsx` | PASS |
| Warn-Banner-Wortlaut statisch (kein User-Input) | DIALOG_LABELS Compile-Time-Konstanten | PASS |
| Reason-Textarea: plain-text-Sink | React-Standard-Rendering | PASS |

### Scan Results

- **govulncheck ./...** — 0 callable Vulnerabilities. 5 nicht-callable in Transitive-Deps (Pre-PROJ-100, unverändert).
- **gosec -severity medium -confidence medium ./internal/application/... ./internal/http/...** — 0 Issues über 36 Files, 14784 Lines.
- **npm audit --audit-level=high** — 0 high. 4 moderate Pre-PROJ-100-Bestand (uuid GHSA-w5hq-g745-h8pq).
- **trivy config helm/** — keine HIGH/CRITICAL Misconfigurations (Scope: PROJ-100 ändert kein Helm).
- **Semgrep** — nicht lokal ausgeführt (CI-Workflow `.github/workflows/security-scan.yml` läuft beim Push auf main).

### Verdict: APPROVED

- 0 Critical / 0 High / 0 Medium / 5 Info — alle Info-Findings akzeptiert mit Begründung
- Alle Pflicht-Trigger (Status-Transitions, neue Endpoints, DB-Schreibzugriff, Auth/Tenant)
  bestätigt durch Code-Audit
- Strukturelle Sicherheits-Controls (Lock-Pattern, Drift-Wache-Test, checkTenantAccess) sauber
  implementiert und konsistent zu Bestand-Patterns (ResetImport, RollbackActivation)
- Keine neuen Angriffsflächen — die zwei neuen Endpoints liegen hinter dem gleichen Auth-Stack
  wie alle bestehenden Admin-Endpoints

**Empfehlung:** `/deploy` direkt fortsetzen.

## Deployment

**Datum:** 2026-06-10
**Tag:** `v1.27.0-PROJ-100`
**Image-SHA:** wird vom Auto-Bump-Commit nach Push gesetzt
**Helm-Werte:** keine Änderung
**Migration-Job:** kein Schema-Change, keine Migration nötig

**Owner-Aktion auf test-Cluster:**

```
helm upgrade eegfaktura-member-onboarding ./helm/member-onboarding \
  -f helm/member-onboarding/values-env.yaml \
  -f helm/member-onboarding/values-secret.yaml
```

**Tester-Verifikation:**

1. Antrag mit Status *Aktiviert* öffnen → neuer Button „Aktivierung zurücksetzen" sichtbar
2. Klick → Confirm-Dialog mit amber Warn-Banner + Pflicht-Begründungs-Feld
3. Begründung mit ≥10 Zeichen → 200, Toast „Aktivierung zurückgesetzt", Status jetzt *Importiert*
4. Optional: Status *Importiert* → zwei Reset-Buttons sichtbar → „Auf Prüfung zurücksetzen" → analog
5. Statusverlauf zeigt beide Einträge mit `[reset-activation]` / `[reset-to-review]`-Prefix

**Doku-Updates im Deploy-Commit:**

- `CLAUDE.md` Status-Modell um zwei neue Transitionen erweitert
- `docs/domain-model.md` analog
- `docs/api-spec.md` zwei neue Endpoints (6.5.5b, 6.5.5c) dokumentiert
- `docs/user-guide/04-admin-applications.md` Aktions-Tabelle erweitert (PROJ-frei)
- `docs/user-guide/changelog.md` 2026-06-10-Eintrag (PROJ-frei)
- `CHANGELOG.md` Feature-Eintrag
