# PROJ-70 — Stammdaten-Resync von aktivierten Anträgen aus eegFaktura-Core

## Status: Planned
**Created:** 2026-06-01
**Last Updated:** 2026-06-01
**Source:** Tester-Feedback 2026-06-01 — aktivierte Anträge konnten weiterhin editiert werden (UX-Bug, jetzt via Commit `debc761` als Trivial-Fix gefixt: Button ausgeblendet). Owner-Direktive: statt klassisches Edit ein „Stammdaten aus eegFaktura abgleichen"-Knopf, der die Core-Werte für einen einzelnen Antrag nachzieht.

## Dependencies
- **PROJ-46** (SEPA-B2B + activation_mode) — liefert die Status `awaiting_bank_confirmation`, `ready_for_activation`, `activated` in die wir bei IBAN-/Kontoinhaber-Wechsel zurückkehren.
- **PROJ-69** (Reconciliation-Backstop) — semantisch verwandt, aber technisch eigenständig. PROJ-69 ist Batch-Pull der Faktura-Teilnehmerliste mit Strict-Match, schreibt nur `member_number` + `faktura_handover_at`. PROJ-70 ist on-demand Pull EINES bekannten Antrags via Core-ID, überschreibt vollständige Stammdaten inkl. Bank.
- **PROJ-50/53** (Activation-Notification + activated-Status) — der Resync verändert das bisherige strict-end-state-Verhalten von `activated`: bei IBAN-/Kontoinhaber-Wechsel ist ein Rückfall auf `awaiting_bank_confirmation` jetzt erlaubt.

## Hintergrund

Sobald ein Antrag den Status `activated` erreicht hat, gehört die Stammdaten-Pflege ins eegFaktura-Core. Das Onboarding-Tool hält weiter eine Kopie der Stammdaten in seiner DB (für Antrags-Detail-Ansicht, Excel-Export, Beitrittsbestätigungs-PDF). Diese Kopie veraltet, sobald im Core etwas geändert wird (Adresse, IBAN, Telefon). Folgen:

- Tenant-Admin sieht im Onboarding-Detail veraltete Werte und denkt fälschlich, das Mitglied hätte Adressänderungen noch nicht mitgeteilt
- Re-Export einer Beitrittsbestätigungs-PDF zeigt die alten Werte
- Bei Streit „warum steht im Onboarding noch die alte IBAN?"

**Lösung:** ein „Stammdaten aus eegFaktura abgleichen"-Knopf im Antrags-Detail (nur bei Status `activated` sichtbar). Klick → Backend pullt die aktuellen Core-Werte für genau diesen Antrag → überschreibt die Onboarding-DB → Toast zeigt welche Felder sich geändert haben. Bei IBAN- oder Kontoinhaber-Wechsel wird zusätzlich das bestehende SEPA-Mandat invalidiert und der Antrag fällt zurück auf `awaiting_bank_confirmation`, damit der Admin den bestehenden Bank-Confirmation-Flow auslösen kann.

## User Stories

1. **Als Tenant-Admin** möchte ich bei einem aktivierten Antrag mit einem Klick die aktuellen Core-Stammdaten in das Onboarding-Detail nachziehen können, **damit** die im Onboarding sichtbaren Werte nicht hinter dem Core veralten.
2. **Als Tenant-Admin** möchte ich nach dem Resync sehen, welche Felder sich geändert haben, **damit** ich die Änderung nachvollziehen und ggf. Folge-Aktionen (z. B. neue Beitrittsbestätigungs-PDF generieren) auslösen kann.
3. **Als Tenant-Admin** möchte ich, dass bei einer geänderten Bankverbindung (IBAN oder Kontoinhaber) automatisch das bestehende SEPA-Mandat invalidiert wird und der Antrag in den bestehenden Bank-Confirmation-Flow zurückfällt, **damit** ich nicht versehentlich mit einem ungültigen Mandat weiterabbuche.
4. **Als Tenant-Admin** möchte ich nach einem Bank-Daten-Resync die Mandat-Mail manuell auslösen können, **damit** ich entscheiden kann, ob ein neues Mandat wirklich nötig ist oder ob die Bankdaten-Änderung nur eine Korrektur im Core war.
5. **Als Mitglied** möchte ich, dass mir ein neues SEPA-Mandat-Formular automatisch nur zugeschickt wird, wenn der Admin das explizit auslöst, **damit** ich keine spam-ähnlichen Mails bei jedem Resync bekomme.

## Acceptance Criteria

### AC-1: Sichtbarkeit + Bedingung
- [ ] Der Knopf „Stammdaten aus eegFaktura abgleichen" erscheint **ausschließlich** im Antrags-Detail von Anträgen mit Status `activated`.
- [ ] In allen anderen Status (inkl. `ready_for_activation`, `awaiting_bank_confirmation`, `imported`) ist der Knopf nicht sichtbar.
- [ ] Wenn `application.target_participant_id` NULL ist (kein Core-Match), ist der Knopf disabled mit Hover-Hinweis „Antrag nicht mit eegFaktura-Mitgliedsnummer verknüpft".

### AC-2: Resync-Inhalt
- [ ] Folgende Felder werden bei Match aus dem Core übernommen:
  - **Stammdaten:** firstname, lastname, titel, titelNach, companyName, uidNumber
  - **Adresse:** street, streetNumber, zip, city, residentStreet, residentStreetNumber, residentZip, residentCity
  - **Kontakt:** email, phone
  - **Anteile:** cooperativeSharesCount
  - **Bank:** iban, bankName, accountHolder
- [ ] Felder die im Core leer/NULL sind, bleiben in der Onboarding-DB **unverändert** (Keep-Default).
- [ ] memberType, birthDate, membershipStartDate, registerNumber bleiben vom Resync unberührt (Identitäts-Kern).
- [ ] Zählpunkte (metering_point), Status-Felder, faktura_handover_at, member_number, admin_note bleiben vom Resync unberührt.

### AC-3: SEPA-Mandat-Invalidierung
- [ ] Wenn der Resync eine andere IBAN **oder** einen anderen Kontoinhaber im Core erkennt (Vergleich auf normalisierter IBAN + getrimmtes Kontoinhaber-Feld, beide case-insensitive), passiert zusätzlich:
  - `mandate_reference`, `mandate_date`, `sepa_mandate_accepted`, `sepa_mandate_accepted_at` werden geleert
  - `einzugsart` wird auf `kein_sepa` gesetzt
  - Status wechselt von `activated` auf `awaiting_bank_confirmation`
  - Status-Log-Eintrag: „Stammdaten aus eegFaktura abgeglichen — SEPA-Mandat invalidiert wegen IBAN-/Kontoinhaber-Wechsel"
- [ ] Wenn nur Stammdaten (nicht Bank) sich geändert haben, bleibt der Status auf `activated` und Mandat-Felder unverändert.

### AC-4: Audit + Status-Log
- [ ] Jeder Resync (auch wenn nichts geändert wurde) hinterlässt einen status_log-Eintrag.
- [ ] Reason-Text enthält die Liste der geänderten Feldnamen, z. B.: „Stammdaten aus eegFaktura abgeglichen (geändert: residentStreet, residentZip, phone)".
- [ ] Wenn nichts geändert wurde: Reason-Text „Stammdaten aus eegFaktura abgeglichen (bereits synchron)".
- [ ] changed_by_user_id im status_log enthält das Keycloak-Subject des auslösenden Admins (NICHT hardcoded „service").

### AC-5: UI-Feedback
- [ ] Vor dem Resync: kein Confirm-Dialog.
- [ ] Während des Resync: Button zeigt „Wird abgeglichen…" und ist disabled.
- [ ] Nach erfolgreichem Resync: Toast mit der Liste der geänderten Felder (deutsche Label-Namen, z. B. „Telefonnummer", „Wohnort PLZ") oder „Bereits synchron" wenn keine Änderung.
- [ ] Bei IBAN-/Kontoinhaber-Wechsel: Toast zusätzlich mit Hinweis „SEPA-Mandat wurde invalidiert — bitte ggf. neue Mandat-Mail auslösen".
- [ ] Antrags-Detail wird nach Resync neu geladen, damit die neuen Werte sichtbar werden (siehe Memory `feedback_ui_refresh_after_apply`).

### AC-6: Core-Fehler
- [ ] Core-API liefert 404 (Mitglied dort nicht mehr vorhanden) → 502-Response mit Body `{ "code": "core_member_not_found" }`, Toast „Mitglied in eegFaktura nicht gefunden — bitte mit Plattform-Admin klären".
- [ ] Core-API timeout / 5xx → 502-Response mit generischer Fehlermeldung, Toast „Abgleich mit eegFaktura fehlgeschlagen — bitte später erneut versuchen". Onboarding-DB bleibt unverändert (Hard-Fail, kein Partial-Update).
- [ ] Silent-SSO-Token fehlt (X-Core-Authorization Header leer) → 400 mit Hinweis „Core-Token nicht verfügbar — bitte Login erneuern".

### AC-7: Manuelle Mandat-Mail
- [ ] Nach einem IBAN-/Kontoinhaber-Wechsel-Resync ist der bestehende Resend-Confirmation-Knopf im Antrags-Detail (Status `awaiting_bank_confirmation`) der Trigger für die SEPA-Mandat-Mail. **Out-of-Scope für PROJ-70:** keine neue Mail-Logik, kein neuer Mail-Template, kein separater Knopf — der Admin nutzt den bestehenden Bank-Confirmation-Flow.
- [ ] Falls heute kein passender Mail-Trigger für `awaiting_bank_confirmation` existiert, **wird das im `/architecture`-Skill geklärt** und ggf. in eine separate Folge-PROJ-Nummer ausgelagert.

### AC-8: Auth + Tenant-Isolation
- [ ] Endpoint ist Keycloak-protected.
- [ ] Tenant-Admin kann den Resync nur für Anträge seiner eigenen EEGs auslösen (`checkTenantAccess` via rc_number).
- [ ] Superuser kann den Resync für jeden Antrag auslösen.

## Edge Cases

- **Mitglied im Core gelöscht (404):** Hard-Fail, Onboarding-DB unverändert. Toast informiert den Admin.
- **Resync-Idempotenz:** zweiter Klick direkt nach erfolgreichem Resync → Toast „Bereits synchron", kein zweiter status_log-Eintrag mit Feld-Liste (nur einer mit „bereits synchron")? **Entscheidung:** zweiter Eintrag wird trotzdem geschrieben (Audit-Trail vollständig), aber der Reason-Text ist „bereits synchron".
- **Resync während paralleler Admin-Aktion:** zwei Admins klicken gleichzeitig → erster gewinnt, zweiter bekommt die neuen Werte und „bereits synchron".
- **Antrag schon auf `awaiting_bank_confirmation` (z. B. via PROJ-46 schon zurückgefallen) + Admin klickt versehentlich nochmal):** Knopf ist nicht sichtbar (siehe AC-1) → kein versehentlicher Doppel-Resync.
- **Core liefert NULL für ein Feld, das im Onboarding gesetzt ist:** Onboarding-Wert bleibt erhalten (Keep). Status-Log zeigt dieses Feld nicht in der Änderungs-Liste.
- **Core liefert einen leeren String "" statt NULL:** behandeln wir als NULL (Keep). Sonst würden im Status-Log laufend Phantom-Änderungen auftauchen.
- **IBAN-Normalisierung beim Vergleich:** Whitespace + Case ignorieren (wie PROJ-69), sonst falsch-positive Mandat-Invalidierungen.
- **Kontoinhaber-Vergleich:** trim + case-insensitive. „Mustermann " ≠ „Mustermann" wäre absurd.
- **Resync vor erstem Login eines neuen Admins:** Silent-SSO-Token noch nicht im LocalStorage → 400. Admin loggt sich aus und wieder ein.
- **Resync triggert IBAN-Wechsel, Status fällt auf `awaiting_bank_confirmation`, Admin will doch nicht das Mandat erneuern:** Admin kann manuell via `POST /reset-import` oder die Status-Übergänge zurück auf `approved` und dann via Re-Import oder „Manuell aktivieren" wieder auf `activated`. Standard-Flow, kein PROJ-70-Special.
- **Multi-Tenant-Audit-Log-Forge-Versuch:** Frontend pickt manipulierte application_id einer fremden EEG → Backend checkt `checkTenantAccess` → 403, kein Resync.
- **DSGVO-Auskunft eines Mitglieds:** Mitglied fragt „welche Daten habt ihr von mir gespeichert?" — Antwort enthält die durch Resync letztgepflegten Werte. AVV §3 muss dokumentieren, dass die Daten aus dem Core gespiegelt werden.

## Technical Requirements

- **Performance:** Single-Application-Pull, < 3 Sekunden Server-Roundtrip (Core-Call + DB-Write + status_log).
- **Security:** Keycloak-Auth + Tenant-Check + Silent-SSO-Token (X-Core-Authorization). Kein PII in Logs (truncateForLog wie PROJ-69).
- **Browser Support:** Chrome, Firefox, Safari (Standard wie Rest der Admin-UI).
- **Idempotent:** mehrfacher Klick führt zu mehrfachen status_log-Einträgen, aber keine Daten-Korruption.
- **Audit-Trail:** status_log-only (kein eigenes data_resync_log).

## DSGVO / AVV

- **AVV-Update Pflicht** vor Aktivierung der Funktion auf Produktion:
  - § 3 Zwecke ergänzen um „Aktualisierung der im Onboarding-Tool gespeicherten Stammdaten aus dem eegFaktura-Kernsystem auf Admin-Anforderung"
  - § 4 Datenkategorien ergänzen um „Stammdaten-Snapshot (Name, Adresse, Kontakt, Bankverbindung) zum Zeitpunkt des Abgleichs"
  - Hinweis auf Quelle: Core-Daten werden via authentifiziertem Backend-zu-Core-Call gepullt, keine Drittstaaten-Übermittlung.
- **User-Guide-Update Pflicht** vor Aktivierung: Abschnitt „Stammdaten-Abgleich" mit Erklärung wann der Knopf erscheint, was er macht, was bei IBAN-Wechsel passiert.

## Out-of-Scope (explizit)

- **Push-Back** Onboarding → Core (Onboarding ist Read-Only-Konsument des Core nach Aktivierung)
- **Cron-Job oder Batch-Resync** für alle aktivierten Anträge einer EEG (nur on-demand pro Antrag)
- **Re-Send der Beitrittsbestätigungs-PDF** mit neuen Daten (separater Knopf, falls überhaupt gewünscht — eigene PROJ-Nummer)
- **Mitglieds-Self-Service** für eigenen Daten-Refresh (kein Member-facing UI)
- **Neue SEPA-Mandat-Mail-Logik** (existing `awaiting_bank_confirmation`-Mail-Trigger genutzt — falls keiner existiert, eigene Folge-PROJ-Nummer)
- **Eigene Tabelle `data_resync_log`** mit Vorher-Nachher-Snapshot (status_log reicht laut Owner-Entscheidung)
- **Feature-Flag** `RESYNC_ENABLED` (Funktion ist direkt aktiv nach Deploy)
- **Auto-Logout des Mitglieds** bei Daten-Änderung (Mitglied hat sowieso kein Login im Onboarding)

## Vorgelagerter Trivial-Fix

Commit `debc761` (2026-06-01) — der „Bearbeiten"-Knopf im Antrags-Detail wird nur noch in den Status angezeigt, die das Backend wirklich akzeptiert: `submitted`, `under_review`, `needs_info`, `approved`, `import_failed`. In `activated` (und allen anderen Post-Import-Status) ist er ausgeblendet. PROJ-70 ersetzt die fehlende Funktionalität für `activated` durch den Stammdaten-Resync-Knopf.

## Grill-Me Round (2026-06-01) — Owner-Entscheidungen festgenagelt

Konzept-Stresstest VOR /architecture. Code-Recon hat drei harte Findings gebracht:

1. **Faktura-Core hat kein `GET /participant/{id}`** — verifiziert gegen `c:/opt/repos/myeegfaktura/eegfaktura-backend/api/participantController.go:19-32`. Nur `GET /participant` (volle Liste, tenant-scoped), `PUT/DELETE/POST` für Mutationen. PROJ-70 muss die volle Liste pullen und client-seitig per `member_number` filtern. **Architektur muss klären:** Wiederverwendung `coreclient.ListParticipants` aus PROJ-69 + lokaler Filter, oder neuer dedizierter Code-Pfad.
2. **`activation_notification_sent_at`-Flag** ist explizit gegen Doppel-Versand der Beitrittsbestätigungs-Mail bei `reset-import + re-activate` gebaut (`admin_service.go:1047-1063`, `1103-1107`). Bleibt im IBAN-Wechsel-Loop gesetzt → keine zweite Mail.
3. **PROJ-69-Reconciliation filtert `WHERE faktura_handover_at IS NULL`** (`reconciliation_repo.go:185`). Aktivierte Anträge sind nicht im Reconciliation-Pool. Sauber, solange PROJ-70 `faktura_handover_at` beim IBAN-Loop nicht zurücksetzt.

### Owner-Entscheidungen (8 von 8)

| # | Branch | Entscheidung |
|---|---|---|
| 1 | Beitrittsbestätigungs-Mail bei Re-Aktivierung nach IBAN-Loop | **Nein, Flag schlägt zu.** activation_notification_sent_at bleibt gesetzt → PROJ-53-Idempotenz greift. Mitglied bekommt einmalig die Beitrittsbestätigung beim ersten activated-Übergang. Bei IBAN-Wechsel kommt nur die separate SEPA-Mandat-Mail (manuell durch Admin). |
| 2 | `faktura_handover_at` beim IBAN-Wechsel-Rollback | **Bleibt gesetzt.** Abrechnungs-Trigger ist nicht reversibel. Konsequenz: PROJ-69-Reconciliation pullt den Antrag NICHT erneut → saubere Trennung der zwei Systeme. |
| 3 | Bank-Diff-Pre-Confirm bei reinem Tippfehler-Verdacht | **Striktes Verhalten: jeder Bank-Diff invalidiert.** Lieber einmal zu viel ein neues Mandat anfordern als eins mit falschen Daten weiterabbuchen. Owner-Risiko bewusst akzeptiert. |
| 4 | status_log-Eintrag bei „bereits synchron" | **Nur bei Real-Change.** Kein status_log-Eintrag wenn der Resync keine Feld-Änderung findet. Toast zeigt „bereits synchron", Antrags-Verlauf bleibt sauber. |
| 5 | Partial-Sync-Disclosure (was NICHT abgeglichen wird) | **Info-Popover am Knopf.** Erklärt was abgeglichen wird (Stammdaten, Adresse, Kontakt, Bank) und was nicht (Zählpunkte, Identität-Kern). User-Guide-Eintrag erklärt es ausführlich. Toast nach Sync nur „X Felder aktualisiert". |
| 6 | Aktivierte Anträge OHNE `target_participant_id` (manuell via PROJ-53) | **Knopf disabled, Hover-Hinweis.** Konsistente UI, Tooltip: „Antrag wurde manuell aktiviert ohne Import — kein Faktura-Mitgliederlink, Abgleich nicht möglich". |
| 7 | Multi-Admin-Race | **Atomare UPDATE mit Vorher-Vergleich.** UPDATE … WHERE updated_at = $alter_wert. Zweiter Admin pickt die Werte des ersten → „bereits synchron". Idempotent, kein Lock-Mechanismus nötig. |
| 8 | DE-Field-Labels im Toast | **Frontend-Map.** Konstante mapping `residentStreet` → „Wohnort Straße", `iban` → „IBAN", etc. Backend liefert englische Feld-Namen, Frontend rendert. |

### Spec-Updates aus Grill-Me

**AC-3 (SEPA-Mandat-Invalidierung) wird ergänzt:**
- `faktura_handover_at` bleibt beim IBAN-Loop **unverändert** (Abrechnungs-Trigger nicht reversibel).
- `activation_notification_sent_at` bleibt beim IBAN-Loop **unverändert** (keine zweite Beitrittsbestätigung).

**AC-4 (Status-Log) wird ergänzt:**
- Wenn der Resync keine Feld-Änderung findet, wird **kein** status_log-Eintrag geschrieben (nur Toast „bereits synchron"). Bei Real-Change: ein Eintrag mit Feld-Namen-Liste.

**AC-1 (Sichtbarkeit) wird ergänzt:**
- Bei aktivierten Anträgen ohne `target_participant_id` (manuell via PROJ-53): Knopf sichtbar aber **disabled** mit Hover-Hinweis.
- Info-Popover am Knopf erklärt den partiellen Sync-Scope.

**AC-7 (Mandat-Mail) — Klärung:**
- Bestehender Resend-Confirmation-Flow im Status `awaiting_bank_confirmation` muss prüfen: existiert ein Mail-Trigger oder nicht? **Bleibt offen für /architecture-Code-Check.**

**AC-9 (neu): Concurrency**
- [ ] Resync verwendet atomares UPDATE mit Vorher-Vergleich gegen `application.updated_at`. Bei Race verliert der zweite Klick → Diff gegen die neu-geladene DB ergibt „bereits synchron".

## Offen für `/architecture`

1. **Code-Reuse mit PROJ-69:** Wiederverwendung von `coreclient.ListParticipants` + neuer Filter-Funktion, oder eigener coreclient-Pfad? Spec-Empfehlung: Wiederverwenden, da identische Auth-Kette + Tenant-Filter.
2. **`target_participant_id`-Speicherort:** Schema-Check (vermutlich `application.target_participant_id`, gesetzt beim erfolgreichen /import). Architektur klärt.
3. **Diff-Berechnung:** pro Feld plain compare oder normalisierte Vergleichsfunktion (Trim, lowercase für Email, IBAN-Normalisierung wie PROJ-69)? Insbesondere relevant für die Bank-Diff-Erkennung.
4. **Status-Map-Erweiterung:** `activated -> awaiting_bank_confirmation` in `shared/transitions.go` ergänzen. CLAUDE.md aktualisieren: „activated ist End-State" → „activated ist End-State außer bei PROJ-70-IBAN-Wechsel-Loop".
5. **Manuelle-Mandat-Mail-Trigger:** existiert ein Mail-Trigger für `awaiting_bank_confirmation`? Wenn nein, ist das Out-of-Scope-Risiko gross (Mitglied bekommt das neue Mandat-Formular nie zu sehen). Architektur muss konkret klären.
6. **Atomic-UPDATE-Implementation:** `WHERE updated_at = $expected` vs. `WHERE updated_at >= NOW() - 1s` vs. row-level Optimistic-Locking. Architektur entscheidet konkrete SQL-Form.
7. **AVV-Wortlaut + User-Guide-Wortlaut** konkret formulieren (Owner reviewt nach /architecture).

---

<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)

**Erstellt:** 2026-06-01, nach Spec + Grill-Me-Runde 1.

### Architektur-Linie

PROJ-70 ist ein **on-demand Pull-Sync von einem einzelnen Antrag**. Es entsteht **keine neue Tabelle**, keine neue Domain. Wir erweitern bestehende Strukturen (coreclient, status_log, status-transition-map) um die neuen Felder und einen neuen Pfad.

Der zentrale Mechanismus ist eine **Diff-Engine**, die zwischen dem aktuellen Onboarding-Zustand und dem aktuellen Core-Zustand pro Feld vergleicht. Das Ergebnis ist eine Liste der geänderten Felder, die sowohl die Persistierung als auch die UI-Toast-Anzeige steuert.

### A) Komponenten-Baum (Frontend)

```
Antrags-Detail-Seite (admin-application-detail.tsx, bestehend)
+-- Aktions-Button-Leiste (bestehend)
|   +-- [Bestehende Buttons: Export, Excel, Datenweiterleitung, …]
|   +-- NEU: "Stammdaten aus eegFaktura abgleichen"-Button
|   |       (nur in Status `activated`, disabled wenn target_participant_id NULL)
|   +-- NEU: Info-Popover am Resync-Button
|   |       (erklärt Sync-Scope: was abgeglichen wird, was nicht)
|   +-- NEU: "Neues SEPA-Mandat versenden"-Button
|           (nur in Status `awaiting_bank_confirmation`, post-Resync-Trigger)
+-- Stammdaten-Bereich (bestehend)
+-- Bank-Bereich (bestehend)
+-- Status-Verlauf (bestehend, nimmt status_log-Einträge auf)
+-- Toast-Bereich (bestehend Sonner-Toast)
    +-- NEU: Diff-Toast nach Resync mit DE-lokalisierten Feld-Namen
```

**Wichtig:** keine neue Seite, kein neues Modal, kein Confirm-Dialog. Alles passiert inline.

### B) Datenfluss (eine Resync-Aktion)

```
1. Admin klickt "Abgleichen"
   ↓
2. Frontend sendet POST mit Bearer-Token + X-Core-Authorization
   (Silent-SSO-Token aus LocalStorage, gleiches Pattern wie PROJ-69)
   ↓
3. Backend prüft: Keycloak-Auth → Tenant-Check (RC-Number) → Status=activated
   → target_participant_id non-null → 403/409 sonst
   ↓
4. Backend ruft Faktura-Core (volle Liste GET /participant, da kein /participant/{id}
   existiert — gleicher Constraint wie PROJ-69)
   ↓
5. Backend filtert die Liste auf target_participant_id und holt sich genau
   einen Datensatz. Wenn nicht gefunden: 502 mit code=core_member_not_found.
   ↓
6. Diff-Engine vergleicht 16 Felder zwischen Onboarding-DB und Core-Daten,
   pro Feld mit feldspezifischer Normalisierung:
   - IBAN: Whitespace-strip + uppercase (PROJ-69-Pattern wiederverwenden)
   - Email: lowercase + trim
   - sonstige Strings: trim
   - NULL-Handling: Core-NULL ⇒ kein Diff (Keep-Default)
   - Leerer String aus Core wird wie NULL behandelt
   ↓
7. Wenn KEINE Änderung: Backend antwortet mit leerer Diff-Liste,
   KEIN status_log-Eintrag. Frontend zeigt "bereits synchron"-Toast.
   ↓
8. Wenn Änderung: atomares UPDATE auf application-Tabelle mit
   Optimistic-Locking (WHERE updated_at = vorheriger-Wert). Wenn
   rowsAffected = 0 (Race mit anderem Admin): einmal neu lesen,
   neu diffen, neu schreiben. Beim zweiten Fail: Hard-Fail 409.
   ↓
9. Wenn iban ODER accountHolder im Diff: zusätzlich:
   - Mandat-Felder leeren (mandate_reference, mandate_date,
     sepa_mandate_accepted, sepa_mandate_accepted_at)
   - einzugsart auf "kein_sepa"
   - Status-Übergang `activated -> awaiting_bank_confirmation`
   - faktura_handover_at bleibt unverändert (Owner-Entscheidung Grill #2)
   - activation_notification_sent_at bleibt unverändert (Owner #1)
   ↓
10. status_log-Eintrag mit Reason-Text:
    "Stammdaten aus eegFaktura abgeglichen (geändert: <feld1>, <feld2>, …)"
    bzw. bei Bank-Diff zusätzlich:
    "Stammdaten abgeglichen + SEPA-Mandat invalidiert wegen IBAN-/Kontoinhaber-Wechsel"
    changed_by_user_id = "resync:<keycloak-subject>" (analog PROJ-69)
   ↓
11. Backend antwortet 200 mit Diff-Feld-Liste (englische Feld-Namen).
    Frontend rendert Diff-Toast mit DE-Labels via Frontend-Map.
    Antrags-Detail wird neu geladen (refetch-Pattern aus PROJ-61).
```

### C) Mandat-Resend-Pfad (zweiter neuer Endpoint, Owner-Entscheidung 2026-06-01)

```
1. Admin sieht Antrag in awaiting_bank_confirmation und klickt
   "Neues SEPA-Mandat versenden"
   ↓
2. Backend prüft Auth + Tenant + Status=awaiting_bank_confirmation
   ↓
3. Backend ruft die bestehende SendPostImportNotification-Mechanik
   (PROJ-53) auf:
   - PDF wird neu generiert (mit AKTUELLER IBAN + Kontoinhaber aus Onboarding-DB)
   - Mail an Mitglied via SMTP (hard-fail, kein async — Owner-Pattern für
     admin-getriggerte Mails, siehe Memory feedback_mail_hard_fail)
   ↓
4. status_log-Eintrag "SEPA-Mandat-Mail (neu) versandt"
   ↓
5. Frontend zeigt Toast "Mandat-Mail versandt", refetch Detail
```

Konsequenz: ab jetzt darf der Admin den bestehenden Status-Übergang
`awaiting_bank_confirmation -> ready_for_activation` manuell auslösen, sobald
das Mitglied das neue Mandat bestätigt hat. Der weitere Pfad zu `activated`
folgt der bestehenden PROJ-53-Logik (keine zweite Beitrittsbestätigungs-Mail
dank `activation_notification_sent_at`-Flag).

### D) Daten-Model-Änderungen

**Keine neuen Tabellen.** Folgende bestehende Strukturen werden ergänzt:

| Was | Bestehend | Änderung |
|---|---|---|
| `coreclient.CoreParticipantSummary` | hat ID, ParticipantNumber, Status, Meters, Contact.Email, AccountInfo.Iban (PROJ-69) | Erweiterung um Felder: FirstName, LastName, TitleBefore, TitleAfter, VatNumber, BillingAddress (Street, Number, Zip, City), ResidentAddress, Contact.Phone, BankAccount.Owner, BankAccount.BankName. Alle als Pointer + omitempty (tolerant gegen Schema-Drift, Pattern aus PROJ-69). |
| `internal/shared/transitions.go` | hat keinen ausgehenden Übergang aus `activated` | Neuer erlaubter Übergang: `activated → awaiting_bank_confirmation` (nur durch PROJ-70-Resync-Service erreichbar, nicht über generisches /status-Endpoint). |
| `CLAUDE.md` Status Model | „activated ist End-State — strictly no transitions out" | Update auf: „activated ist End-State außer beim PROJ-70 Resync-Loop, bei IBAN- oder Kontoinhaber-Wechsel kann der Antrag zurück auf `awaiting_bank_confirmation` fallen". |
| `application` Tabelle | bestehend | Keine Schema-Änderung. Resync schreibt in bestehende Spalten. |
| `status_log` Tabelle | bestehend | Keine Schema-Änderung. Resync erzeugt Einträge mit dem etablierten Format. |

### E) Diff-Engine — was wird wie verglichen

| Onboarding-Feld | Faktura-Feld | Normalisierung |
|---|---|---|
| firstname | FirstName | trim |
| lastname | LastName | trim |
| titel | TitleBefore | trim |
| titelNach | TitleAfter | trim |
| uidNumber | VatNumber | trim |
| residentStreet | BillingAddress.Street | trim |
| residentStreetNumber | BillingAddress.StreetNumber | trim |
| residentZip | BillingAddress.Zip | trim |
| residentCity | BillingAddress.City | trim |
| email | Contact.Email | lowercase + trim |
| phone | Contact.Phone | trim |
| iban | BankAccount.Iban | Whitespace-strip + uppercase (PROJ-69) |
| bankName | BankAccount.BankName | trim |
| accountHolder | BankAccount.Owner | trim |
| cooperativeSharesCount | (kein Mapping in Core) | NICHT abgleichen — bleibt in Onboarding-DB |
| residentAddress-Block | ResidentAddress (wenn ≠ Billing) | analog, falls Core das separat liefert |

**Wichtig:** memberType, birthDate, membershipStartDate, registerNumber, companyName, Zählpunkte, admin_note werden NICHT angefasst (Identitäts-Kern bzw. Out-of-Scope).

**NULL-Handling:** Core-NULL oder leerer String → kein Diff, Onboarding-Wert bleibt (Keep-Default).

### F) Concurrency-Strategie

**Optimistic-Locking auf `application.updated_at`:**

1. Backend liest Application + merkt sich `updated_at_old`.
2. Backend berechnet Diff.
3. Backend versucht UPDATE mit Bedingung `WHERE id = $1 AND updated_at = $updated_at_old`.
4. Wenn `rowsAffected = 0`: anderer Admin war schneller. Backend liest die Application **einmal** neu, diffiert gegen die neuen Werte, schreibt nochmal.
5. Wenn auch der zweite Versuch `rowsAffected = 0`: 409-Antwort „Race-Konflikt — bitte erneut versuchen".

Vorteil: keine zusätzliche Locking-Infrastruktur, idempotent über Pod-Grenzen hinweg.

### G) Status-Übergangs-Erweiterung

CLAUDE.md sowie `internal/shared/transitions.go` werden ergänzt:

- **Neu erlaubt:** `activated → awaiting_bank_confirmation` (nur via PROJ-70-Resync-Service, nicht via generisches `/status`-Endpoint — der generische Pfad lehnt diesen Übergang weiterhin ab).
- **Generic-`/status`-Pfad** bleibt strict (activated ist dort weiter End-State).

Diese Trennung schützt davor, dass ein Admin versehentlich aus dem Status-Dropdown den Antrag zurückwirft.

### H) Backend-Endpoints (zwei neue)

| Endpoint | Methode | Zweck |
|---|---|---|
| `/api/admin/applications/{id}/resync-from-core` | POST | Triggert Diff + Update + ggf. Mandat-Invalidierung |
| `/api/admin/applications/{id}/resend-mandate` | POST | Triggert neue SEPA-Mandat-Mail im Status `awaiting_bank_confirmation` |

Beide hinter dem bestehenden Keycloak-Middleware + `checkTenantAccess`.

### I) DSGVO / AVV

AVV-Update Pflicht vor Aktivierung der Funktion:

- **§ 3 Zwecke** — neuer Spiegel-Punkt: „Aktualisierung der im Onboarding-Tool gespeicherten Mitglieder-Stammdaten und -Bankdaten durch on-demand-Abruf aus dem eegFaktura-Kernsystem auf Anforderung der EEG-Administration."
- **§ 4 Datenkategorien** — Erweiterung der Liste um „Stammdaten-Snapshot (Vorname, Nachname, Titel, UID, Wohnort-Adresse, E-Mail, Telefon, Bankverbindung) zum Zeitpunkt des Resync".
- Hinweis: Quelle ist der bereits dokumentierte Faktura-Core, keine zusätzliche Drittstaaten-Übermittlung.

### J) User-Guide-Update Pflicht

Neuer Abschnitt im Admin-Handbuch:

- Wann erscheint der Resync-Button (Status `activated` + bestehender Core-Link)
- Was wird abgeglichen vs. was nicht (klare Bullet-Liste)
- Was passiert bei IBAN- oder Kontoinhaber-Wechsel (Mandat-Invalidierung + Status-Rückfall + Folge-Knopf)
- Hinweis: Mitgliedstyp, Geburtsdatum, Zählpunkte werden bewusst nicht gespiegelt

### K) Wiederverwendete Bausteine (keine neuen Dependencies)

- `coreclient.ListParticipants` aus PROJ-69 (gleiche Auth-Kette)
- `coreclient.CoreParticipantSummary` aus PROJ-69 (erweitert)
- `application_repo` für Lese-/Schreib-Operationen
- `status_log_repo` für Audit-Einträge (analog PROJ-69-Pattern)
- `SendPostImportNotification`-Mechanik aus PROJ-53 (für Resend-Mandate)
- shadcn `<Button>`, `<Popover>` (Info-Hint), Sonner Toast
- Frontend-Refetch-Pattern aus PROJ-61 (Detail-Reload)

**Keine neuen Packages, keine neue Library, kein neuer External-Dependency.**

### L) Risiken und Annahmen

| Risiko | Mitigation |
|---|---|
| Faktura-Core-Schema-Drift (z. B. Feld umbenannt) | Pointer + omitempty bei allen neuen Feldern. Bei NULL-Antwort wird Wert als „nicht relevant" behandelt (Keep-Default). |
| Manuell-aktivierte Anträge ohne target_participant_id | Button disabled mit Hover-Hinweis (Spec AC-1). |
| EEG mit >2000 Mitgliedern (PROJ-69-Pagination-Limit) | Wie PROJ-69 momentan. Falls relevant: Lookup über member_number-Filter im coreclient prüfen. Spec-Notiz: heute akzeptabel. |
| Tester ist verwirrt warum nicht alles synchronisiert wird | Info-Popover am Button + User-Guide-Abschnitt. |
| AVV nicht rechtzeitig aktualisiert | Hard-Gate: vor Produktions-Deploy AVV-Sign-off durch Owner. |

### M) Reihenfolge für die Implementierung

1. CoreParticipantSummary erweitern + coreclient-Test
2. Status-Map ergänzen + CLAUDE.md updaten
3. Diff-Engine als isolierte, gut testbare Service-Funktion (Mock-Repo)
4. Resync-Service + atomares UPDATE
5. Resend-Mandate-Service (Wiederverwendung PROJ-53)
6. Zwei neue HTTP-Endpoints + Route-Wiring
7. Frontend-Button + Info-Popover + Diff-Toast + DE-Label-Map
8. Frontend-Refetch nach Resync
9. AVV-Wortlaut + User-Guide-Abschnitt
10. E2E-Tests + QA + Security-Review

## Grill-Me Round 2 (2026-06-01) — Tech-Design-Stresstest

Nach /architecture-Skill durchgeführt. Code-Recon hat drei Findings ergänzt:

- **Adress-Mapping im Import** (`importing/payload.go:132`): Onboarding hat einen einzigen Adress-Block; beim /import wird er IDENTISCH in Faktura.BillingAddress UND Faktura.ResidentAddress geschrieben. Beide können sich post-Activation unabhängig im Core entwickeln.
- **target_participant_id-Lifecycle**: wird beim `reset-import` geleert (`application_repo.go:1092`). Manuell-aktivierte Anträge ohne Import haben es nie. Beide bereits durch AC-1 (Button disabled) abgedeckt.
- **Rate-Limit-Infrastruktur**: existiert nur für Public-Endpoints (`PublicSubmitRateLimitMiddleware`). Admin-Endpoints sind heute alle unlimitiert.

### Owner-Entscheidungen (10 von 10 Recommended)

| # | Branch | Entscheidung |
|---|---|---|
| 1 | Race-Edge bei IBAN-Wechsel (zweiter Admin klickt parallel) | **Re-read & Re-diff.** Zweiter Admin's UPDATE schlägt fehl, Backend liest Application neu (jetzt mit neuer IBAN + awaiting-Status), Diff ist leer → „bereits synchron"-Toast. Saubere Idempotenz, keine Doppel-Invalidierung. |
| 2 | Alte Mandat-Werte im status_log archivieren | **Nein, nur Feld-Namen.** Reason-Text z. B.: „SEPA-Mandat invalidiert wegen IBAN-/Kontoinhaber-Wechsel (alte IBAN ersetzt, Mandat-Felder geleert)". Status-Log ist tenant-sichtbar → PII (IBAN, Mandatsreferenz) widerspricht security.md. Audit-Recovery via DB-Backup. |
| 3 | BillingAddress vs ResidentAddress als Source | **ResidentAddress.** Onboarding-Feldnamen sind `residentStreet`/`residentZip` → semantisch passend. Bei Divergenz im Core wird nur ResidentAddress gespiegelt, BillingAddress ignoriert. |
| 4 | target_participant_id-Repair-Pfad bei Core-Migration | **Nur psql + User-Guide-Hinweis.** Extrem seltener Fall. Toast „Mitglied in eegFaktura nicht gefunden — bitte mit Plattform-Admin klären". Plattform-Admin updated via psql. Kein neuer UI-Pfad. |
| 5 | Status-Pfad-Schutz `activated → awaiting_bank_confirmation` | **Getrennte Transition-Maps.** Generische /status-Map bleibt strict (kein activated-out). Resync-Service hat eigene hartcodierte Transition-Liste. Unit-Test verifiziert die Trennung — Code-Review schaut auf das Status-Update-Statement, jeder Treffer außerhalb von `resync_service.go` ist ein Bug. |
| 6 | Schema-Drift-Detection | **Tolerant + Health-Check-Test.** Pointer + omitempty bleibt für Resilienz. Zusätzlich: dedizierter Test-Lauf (manuell oder im CI) gegen Faktura-Test-Endpoint, der die erwarteten Felder NICHT-NULL prüft → bei Drift schlägt der Test fehl + Hinweis. Resync selbst macht Hard-Fail nur bei NICHT-tolerierbaren Fehlern (Core-404, 5xx). |
| 7 | Mandat-Resend-Knopf auch in `activated` als „Kopie senden"? | **Nein, nur in `awaiting_bank_confirmation`.** Strikte Bindung an den Workflow. „Mandat-Kopie verloren"-Fälle gehen über andere Wege (Excel-Export, PDF-Generator). Verhindert Mail-Bursts und hält das Feature scope-eng. |
| 8 | UX-Reihenfolge: Diff-Toast vs Detail-Refetch | **Refetch zuerst, Toast sticky.** Sobald Backend antwortet: Detail-State sofort überschreiben. Toast erscheint persistent (8s oder manueller Dismiss) und zeigt die Diff-Liste. Admin sieht den neuen Zustand und im Toast was sich geändert hat. |
| 9 | cooperativeSharesCount-Mapping fehlt | **Out-of-Scope, kein Resync, kein UI-Pfad.** PROJ-70 ignoriert die Anteile explizit. Pflege via /reset-import + Edit-Form + erneuter Import bei Bedarf. Spec dokumentiert das als bewusste Einschränkung. |
| 10 | Rate-Limit für Mandat-Resend-Endpoint | **Soft-Throttle 3× pro Antrag pro 24h.** Backend prüft via `count(status_log WHERE reason LIKE 'SEPA-Mandat-Mail%' AND application_id=$ AND created_at > NOW()-INTERVAL '24h')`. Bei ≥3: 409 mit Hinweis. Schützt vor versehentlichem Mail-Burst ohne neue Middleware. |

### Spec-Updates aus Round 2

**AC-3 (SEPA-Mandat-Invalidierung) ergänzt:**
- Status-Übergang `activated → awaiting_bank_confirmation` läuft nur über die **Resync-Service-eigene** Transition-Map. Der generische `/status`-Endpoint bleibt strict (kein activated-out).
- status_log-Reason-Text enthält **NUR Feld-Namen**, keine alten Werte (PII-Schutz). Format z. B.: „SEPA-Mandat invalidiert wegen Wechsel der Felder: iban, accountHolder".

**AC-4 (Audit-Trail) ergänzt:**
- Keine PII (alte IBAN, Mandatsreferenz, etc.) im status_log-Reason-Text.

**AC-7 (Mandat-Resend) konkretisiert:**
- Resend-Button **nur in `awaiting_bank_confirmation`** sichtbar (nicht in `activated`).
- Soft-Throttle: max 3 Mandat-Mails pro Antrag pro 24h. Bei Limit: 409 mit konstruktiver Fehlermeldung.

**AC-9 (Concurrency) konkretisiert:**
- Race-Verhalten: bei `rowsAffected=0` → einmal neu lesen, neu diffen, neu schreiben. Wenn neuer Diff leer → „bereits synchron"-Toast (zweiter Admin pickt die Werte des ersten). Bei zweitem Race-Fail: 409 „Race-Konflikt".

**Neue AC-10: Adress-Mapping**
- [ ] Bei Divergenz zwischen Faktura BillingAddress und ResidentAddress wird **nur** ResidentAddress in Onboarding's residentStreet-Block gespiegelt. BillingAddress wird ignoriert.

**Neue AC-11: Drift-Detection**
- [ ] Dedizierter Schema-Audit-Test prüft, dass die 9 neuen Pflicht-Felder (FirstName, LastName, etc.) NICHT-NULL in einer Faktura-Test-Response erscheinen. Bei Drift: Test fehl + klarer Fehler. Resync selbst bleibt tolerant.

**Neue AC-12: Status-Pfad-Schutz**
- [ ] Unit-Test verifiziert, dass die Transition `activated → awaiting_bank_confirmation` über den generischen `/status`-Endpoint mit 409 abgewiesen wird.
- [ ] Resync-Service nutzt eine eigene hartcodierte Transition-Liste, nicht die generische Map.

### Updates der Risiken-Tabelle

| Risiko | Mitigation |
|---|---|
| Versehentlicher Status-Bypass via /status-Pfad | Getrennte Transition-Maps + Unit-Test (AC-12) |
| Schema-Drift unbemerkt | Schema-Audit-Test in CI (AC-11) |
| Admin missbraucht Mandat-Resend für Mail-Spam | Soft-Throttle 3×/24h via status_log-Count (AC-7) |
| Race zwischen zwei aktiven Admin-Sessions | Re-read & Re-diff (AC-9) — idempotent, zweiter Admin pickt den ersten |
| Faktura BillingAddress ≠ ResidentAddress | Eindeutige Mapping-Regel: ResidentAddress wins (AC-10) |
| target_participant_id ungültig wegen Core-Migration | psql-Pfad + User-Guide-Hinweis (Grill #4) |

### Verbleibende Annahmen für `/backend`

1. **Faktura-Adress-Liefer-Reihenfolge:** Bei Core-Antwort wird ResidentAddress gelesen, falls beide gefüllt sind. Falls ResidentAddress NULL/leer und BillingAddress gefüllt: weiter NULL behandeln (Keep-Default → keine Diff für residentStreet). **Backend testet diesen Fall explizit.**
2. **Mandat-Soft-Throttle-Counting:** Reason-Text-Prefix muss eindeutig sein, damit der COUNT-Filter sauber trifft (z. B. „SEPA-Mandat-Mail versandt" als fester Prefix).
3. **Schema-Audit-Test:** läuft als optionaler/manueller Test (nicht jede PR), Owner-Direktive ob CI-Integration oder Manual-Run nach Faktura-Updates.

## Simplification Pass (2026-06-01) — Owner-Direktive „kriegen wir das simpler hin"

Die zwei Grill-Runden haben Robustheit gegen seltene Edge-Cases eingebaut, die sich nicht durch konkrete Tester-Befunde rechtfertigen. Owner hat die Spec auf das Notwendige reduziert. Dieser Abschnitt ist **finale Direktive** und ersetzt widersprüchliche Stellen weiter oben.

### Gestrichen

- **Zweiter Endpoint `POST /resend-mandate`** — entfällt. Stattdessen sendet der **Resync-Service selbst** die SEPA-Mandat-Mail direkt nach der Mandat-Invalidierung, im selben Transaktions-Kontext. Ein Klick auf „Stammdaten abgleichen" mit Bank-Diff = ein Mandat-Mail-Versand.
- **Soft-Throttle 3×/24h** — entfällt. Mail geht nur dann raus, wenn beim Resync ein echter IBAN/Kontoinhaber-Wechsel erkannt wird. Wenn der Admin 10× klickt und nichts ändert sich, wird **null** Mail versandt. Kein expliziter Throttle nötig.
- **Schema-Drift-Audit-Test (AC-11)** — entfällt. Pointer + omitempty handhabt Drift bereits tolerant. Wenn Faktura ein Feld umbenennt, fallen die Tests in der Praxis spätestens beim ersten Tester-Resync auf — explizite Drift-Erkennung wäre Premature-Bauen.
- **Getrennte Transition-Maps + Status-Pfad-Schutz (AC-12)** — entfällt. `activated → awaiting_bank_confirmation` wird in der **bestehenden generischen Map** als erlaubter Übergang ergänzt. Risiko, dass jemand den Übergang versehentlich über das Status-Dropdown auslöst, ist akademisch — Statuswechsel ist sichtbar im status_log + reversibel.
- **Optimistic-Locking-Retry mit Re-read & Re-diff** — entfällt. Plain UPDATE mit Last-Write-Wins. Race zwischen zwei aktiven Admin-Sessions, die innerhalb von Sekunden dasselbe Mitglied resync'en, ist real selten und beide Versuche stehen im status_log.
- **Diff-Engine mit feldspezifischer Normalisierung** — vereinfacht. Trim für alle Strings, IBAN bekommt zusätzlich Whitespace-Strip + Uppercase (real-world nötig), Email zusätzlich Lowercase. Eine kurze Funktion, kein „Engine"-Begriff.
- **Mandat-Resend-Button im Frontend** — entfällt. Es gibt nur **einen** neuen Knopf: „Stammdaten aus eegFaktura abgleichen". Bei Bank-Diff wird die Mail automatisch mitversandt.

### Beibehalten — finale Implementierungs-Skelett

| Schicht | Was | Umfang |
|---|---|---|
| `coreclient.CoreParticipantSummary` | Erweiterung um 9 logische Felder (Pointer + omitempty): FirstName, LastName, TitleBefore, TitleAfter, VatNumber, Contact.Phone, AccountInfo.Owner, AccountInfo.BankName, ResidentAddress-Block | ~30 LOC |
| `internal/shared/transitions.go` | Eine Zeile ergänzen: `activated → awaiting_bank_confirmation` als erlaubter Übergang | ~1 LOC |
| `CLAUDE.md` | Status-Model-Beschreibung aktualisieren: „activated ist End-State außer bei PROJ-70-IBAN-Wechsel" | ~3 Zeilen |
| `internal/application/resync_service.go` (neu) | Pull → Trim-Vergleich → atomares UPDATE → ggf. Mandat-Felder leeren + Status-Wechsel + Mandat-Mail via bestehender `SendPostImportNotification`-Mechanik | ~150 LOC inkl. Tests |
| `internal/http/admin.go` | Neuer Handler `RunResync` | ~40 LOC |
| `cmd/server/main.go` | Eine Zeile Route-Wiring | ~1 LOC |
| `src/components/admin-application-detail.tsx` | Ein Button „Stammdaten aus eegFaktura abgleichen" + Info-Popover + Diff-Toast (Sonner sticky) + Refetch | ~80 LOC |
| `src/lib/api.ts` | Eine API-Funktion `runResyncFromCore` + Response-Type | ~15 LOC |

**Geschätzt ~320 LOC gesamt** (statt vorherigen ~600 LOC).

### Wirksame ACs nach Simplification

Folgende ACs aus der Original-Liste bleiben **wirksam**: AC-1 (Sichtbarkeit + Bedingung), AC-2 (Resync-Inhalt), AC-3 (SEPA-Mandat-Invalidierung — vereinfacht), AC-4 (Status-Log nur bei Real-Change), AC-5 (UI-Feedback), AC-6 (Core-Fehler), AC-8 (Auth + Tenant), AC-10 (Adress-Mapping ResidentAddress wins).

**Gestrichen:** AC-7 (Manuelle Mandat-Mail — der Versand passiert jetzt automatisch im Resync, kein separater Button), AC-9 (Concurrency-Retry-Loop), AC-11 (Drift-Audit-Test), AC-12 (Status-Pfad-Schutz-Unit-Test).

### Edge-Cases die wir bewusst akzeptieren

- **Multi-Admin-Race**: Last-Write-Wins. Beide status_log-Einträge sichtbar, Owner-Audit reicht.
- **Schema-Drift im Core**: Tolerant via Pointer+omitempty. Drift wird beim ersten realen Resync mit Tester-Mitteilung sichtbar.
- **Admin spammt den Knopf**: harmlos, weil bei „keine Änderung" weder DB-Update noch Mail-Versand erfolgt.
- **Admin invalidiert Mandat versehentlich**: status_log macht das nachvollziehbar. Wenn der IBAN-Wechsel-Trigger unerwünscht war, kann der Admin per Status-Übergang zurück auf `activated` oder via /reset-import + Re-Import die Sache geraderücken.

### Status-Map-Erweiterung im Detail

Eine einzige Zeile in der bestehenden allowedTransitions-Map: `StatusActivated: { StatusAwaitingBankConfirmation }`. Plus CLAUDE.md-Update. Keine getrennten Maps, kein Service-spezifischer Allowlist-Mechanismus.

## Final Simplification (2026-06-01) — Owner-Direktive „zwei unabhängige Knöpfe"

Beim Implementieren wurde klar: die Auto-Magie „IBAN-Wechsel triggert Mandat-Invalidierung + Status-Rückfall + Mail-Versand" ist immer noch overengineered. Owner-Direktive: **zwei unabhängige Knöpfe**, beide nur in `activated` sichtbar, keine Kopplung zwischen ihnen.

### Endgültiger Funktionsumfang

| Knopf | Was passiert |
|---|---|
| **„Stammdaten aus eegFaktura abgleichen"** | Pull aus Core → Trim-Vergleich → UPDATE der Application-Felder → status_log-Eintrag mit Diff-Liste. **Keine** Status-Änderung, **keine** Mandat-Logik. Last-Write-Wins. |
| **„SEPA-Mandat erneut senden"** | PDF mit der **aktuellen** IBAN + Kontoinhaber generieren → Mail an Mitglied versenden (hard-fail bei SMTP-Error) → status_log-Eintrag. **Keine** Status-Änderung. |

**Konsequenz:** der Admin entscheidet selbst. Sieht im Diff-Toast nach Resync „IBAN hat sich geändert" und klickt — wenn er das für nötig hält — explizit den zweiten Knopf. Das alte Mandat bleibt rechtlich gültig bis der Admin neu versendet. Diese Verantwortung liegt beim Admin, nicht bei der Auto-Magie.

### Was endgültig wegfällt

- Status-Map-Änderung (`activated` bleibt strict end-state)
- CLAUDE.md-Status-Model-Update
- IBAN/Kontoinhaber-Diff-Detection-Sonderfall
- Mandat-Felder-Reset-Logik
- einzugsart-Behandlung
- Mandat-Mail-Hard-Fail-Refactor von bestehendem `SendPostImportNotification`
- Status-Rückfall-Logik
- Verkopplung von Datensync und Mandat-Versand

### Implementierungs-Skelett (final)

| Schicht | Was | Umfang |
|---|---|---|
| `coreclient.CoreParticipantSummary` | Erweiterung um 9 logische Felder (bereits implementiert ✓) | ~30 LOC |
| `AdminApplicationService.ResyncFromCore` (neu) | Pull → Trim-Vergleich → UPDATE → status_log | ~80 LOC |
| `AdminApplicationService.SendMandateRenewalMail` (neu) | PDF gen + Mail-Versand (hard-fail) + status_log | ~70 LOC |
| `coreClient` als Constructor-Dep ergänzen | `NewAdminApplicationService` + main.go-Wiring | ~10 LOC |
| `internal/http/admin.go` | Zwei neue Handler `RunResyncFromCore`, `SendMandateRenewal` | ~80 LOC |
| `cmd/server/main.go` | Zwei Routen-Wirings | ~2 LOC |
| Frontend (kommt in /frontend) | Zwei Knöpfe, Toast, Refetch | ~100 LOC |

**Geschätzt ~280 LOC im Backend** (statt 320 + Mandat-Renewal-Komplexität). Frontend zusätzlich ~100 LOC.

### Wirksame ACs nach Final Simplification

**AC-1 (Sichtbarkeit) angepasst:** Beide Knöpfe nur in Status `activated`, nur wenn `target_participant_id` non-null. Info-Popover am Resync-Knopf erklärt was abgeglichen wird.

**AC-2 (Resync-Inhalt):** wie bisher, 14 Felder mit Trim-Vergleich. IBAN und AccountHolder sind reguläre Felder im Diff (keine Sonderbehandlung).

**AC-3 (SEPA-Mandat) komplett umformuliert:** Resync ändert nie das SEPA-Mandat. Ein separater „SEPA-Mandat erneut senden"-Knopf (in `activated` sichtbar) generiert ein neues PDF mit den aktuellen Onboarding-Werten (also den eben durch Resync aktualisierten) und versendet es an das Mitglied. Hard-Fail bei SMTP-Error. Kein Status-Wechsel.

**AC-4 (Status-Log) bleibt:** Eintrag nur bei Real-Change beim Resync. Mandat-Resend immer mit Eintrag „SEPA-Mandat-Mail versandt".

**AC-5 (UI-Feedback) bleibt** mit zwei Knöpfen statt einem.

**AC-6 (Core-Fehler) bleibt:** Hard-Fail-Toast.

**AC-8 (Auth + Tenant-Isolation) bleibt** für beide Endpoints.

**AC-10 (Adress-Mapping ResidentAddress) bleibt.**

**Gestrichen:** AC-7 (Mandat-Mail-Bedingung), AC-9 (Optimistic-Locking-Retry), AC-11 (Drift-Audit-Test), AC-12 (Status-Pfad-Schutz). Plus alle Bank-Diff-Konsequenzen aus AC-3.

### Edge-Cases die wir bewusst akzeptieren

- **Admin vergisst, nach IBAN-Wechsel die Mandat-Mail neu zu senden:** alte Abbuchung könnte fehlschlagen, Admin sieht das im Faktura-Core, korrigiert dort. Onboarding mischt sich nicht ein.
- **Admin spammt den Resend-Knopf:** Mitglied bekommt mehrere Mails. Akzeptiert — Admin ist verantwortlich.
- **Multi-Admin-Race auf Resync:** Last-Write-Wins, beide Versuche im status_log sichtbar.
- **Multi-Admin-Race auf Mandat-Resend:** zwei Mails gehen raus. Selten, akzeptiert.

## QA Test Results

**Datum:** 2026-06-01
**Reviewer:** QA Engineer (AI)
**Scope:** Backend Commits b2562b4 (resync-service + handlers) und Frontend dbeba49 (zwei Buttons + Toast + Refetch).

### Acceptance-Criteria-Validierung

| AC | Status | Anmerkung |
|---|---|---|
| AC-1 Sichtbarkeit (zwei Knöpfe nur in `activated`, Resync disabled wenn `targetParticipantId` NULL, Info-Popover) | ✅ Pass | Code-Review: Bedingung `application.status === "activated"` korrekt, `disabled` auf `!application.targetParticipantId` korrekt, `title`-Attribut liefert Hover-Tooltip. Info-Popover via shadcn Popover. **Live-Test in /qa-Manual erforderlich.** |
| AC-2 Resync-Inhalt (14 Felder, Core-NULL→Keep, IBAN/Email normalisiert, Out-of-Scope-Felder unverändert) | ✅ Pass | Code-Review: 14 Felder im Diff (5 Pflicht-Strings + 9 Pointer), Keep-bei-NULL durch `resyncStringTrim`/`resyncPtrStringTrim`, IBAN-Normalisierung durch `normalizeIBANForResync`, Email durch `lowerTrimCorePtr`. `UpdateAdminTx` ist Whitelist-basiert (siehe BUG-3 unten). |
| AC-3 kein Status-/Mandat-Wechsel im Resync | ✅ Pass | Code-Review: `toStatus: curStatus` (kein Wechsel), keine Manipulation von Mandat-Feldern oder `einzugsart` im Resync-Pfad. |
| AC-4 Status-Log (nur bei Real-Change, Feldnamen-Liste, Actor mit Subject) | ✅ Pass | Code-Review: `if len(changed) == 0 { return ... }` skippt Insert. Actor-Format `resync:<subject>` bzw. `mandate-renewal:<subject>` korrekt. Keine alten Werte im Reason-Text (PII-safe). |
| AC-5 UI-Feedback (Loading-Label, Bereits-synchron-Toast, Diff-Toast, Refetch-zuerst) | ✅ Pass | Code-Review: Button-Label `"Wird abgeglichen…"` bei `resyncing`, `toast.info("Stammdaten sind bereits synchron.")` bei leerer Diff-Liste, Sticky-Toast mit `duration: 8000` ms, `await fetchApplication()` **vor** Toast (Owner-Direktive Grill #8). |
| AC-6 Core-Fehler (400 ohne Token, 502 bei Core-404/5xx, keine PII in Logs) | ✅ Pass | Code-Review: 400 mit klarer Meldung bei leerem Core-Token, 502 mit `code=core_member_not_found` und 502 mit generischer Meldung bei sonstigen Errors, `truncateForLog` (300 chars + Newline-Strip) auf err.Error(). Onboarding-DB bleibt bei Error unverändert (Transaction nicht committed). |
| AC-8 Auth + Tenant-Isolation | ✅ Pass | Code-Review: Beide Handler nutzen `h.parseID(w, r)` + `h.checkTenantAccess(w, r, id)` vor dem Service-Call. Keycloak-Middleware ist über die `/api/admin`-Subroute aktiv. |
| AC-10 Adress-Mapping (ResidentAddress wird gelesen, BillingAddress ignoriert) | ✅ Pass | Code-Review: `addrPtr(core, addrStreet)` etc. lesen ausschließlich `core.ResidentAddress.*`, kein Zugriff auf BillingAddress. |

**8 von 8 ACs bestehen Code-Review.**

### Funktionale Findings

| ID | Severity | Datei | Beschreibung | Status |
|---|---|---|---|---|
| BUG-1 | **Medium** | `internal/mail/templates/application_imported_member.html` | Mail-Template-Content ist für den Resend-Mandat-Kontext irreführend: Text sagt „dein Antrag ist in die Bearbeitung übernommen worden" und „Die formale Beitrittsbestätigung folgt" — der Empfänger ist aber bereits aktiviertes Mitglied. Mitglied könnte verwirrt sein. | Open — Owner-Entscheidung: eigenes Template, Conditional im bestehenden Template, oder akzeptieren. |
| BUG-2 | **Low** | `internal/application/resync_service.go:319-329` `resyncPtrIBAN` | Bei IBAN-Diff wird die **normalisierte** Form (ohne Whitespace, uppercase) in die Onboarding-DB geschrieben, nicht der Original-Core-Wert. Format-Inkonsistenz: Eine vorher als „AT12 1904…" gespeicherte IBAN wird nach Resync zu „AT121904…". | Open — Auswirkung in der Praxis minimal (alle nachgelagerten Konsumenten — PDF-Generator, Excel-Export — normalisieren selbst). |
| BUG-3 | **Low** | `internal/application/resync_service.go:176` `UpdateAdminTx` | UpdateAdminTx schreibt ALLE Application-Spalten zurück, nicht nur die geänderten. Theoretisches Race-Risiko bei nebenläufigen Mutationen. **In der Praxis akzeptabel:** Bearbeiten-Button ist in `activated` ausgeblendet (Commit debc761), keine parallelen Edit-Pfade. | Akzeptiert — kein Fix nötig solange `activated` der einzige editierbare Status ohne Edit-Form bleibt. |
| BUG-4 | **Info** | `internal/application/resync_service.go:338-347` `lowerTrimCorePtr` | Email wird in lowercase + trimmed Form gespeichert, falls Core's Email-Casing abweicht. Akzeptable Normalisierung. | Akzeptiert. |

### Security-Smoke-Test

| Kategorie | Bewertung | Anmerkung |
|---|---|---|
| 3.1 Auth/Authz | ✅ Pass | Beide Endpoints hinter KeycloakAuthMiddleware. `checkTenantAccess` vor Service-Call. Keine Superuser-Logik. |
| 3.2 Injection | ✅ Pass | Repo-Methoden (UpdateAdminTx, status_log CreateTx) sind parameterisiert. Field-Namen im Reason-Text aus internen Konstanten, nicht User-Input. |
| 3.3 XSS/CSRF/SSRF | ✅ Pass | `coreclient.ListParticipants` ruft `cfg.Core.BaseURL` (Operator-kontrolliert). Frontend rendert nur camelCase-Feldnamen aus fester Liste. CSRF nicht relevant (Bearer-Auth, keine Cookies). |
| 3.4 Secrets/PII | ✅ Pass | Status-Log-Reason-Text enthält **Feldnamen**, keine alten Werte (security.md eingehalten). slog.Error nutzt `truncateForLog` (PROJ-69-Pattern). X-Core-Authorization-Token nicht in Response-Body. |
| 3.5 Dependencies | ✅ Pass | Keine neuen Packages. |
| 3.6 Business Logic | ⚠️ Info | Resync ändert Status nicht. Mandate-Resend hat **kein Rate-Limit** — Owner-Entscheidung in Simplification-Pass (Trust-the-Admin). Frontend-Doppel-Klick-Schutz via `disabled` während Request. |
| 3.7 Unsichere Defaults | ✅ Pass | coreClient ist nil-safe, klarer Fehler bei fehlender Config. Keine neuen Env-Vars/Secrets. |
| 3.8 Sensible Logs | ✅ Pass | `truncateForLog` auf Error-Pfade. Application-ID (UUID) im slog, keine PII. |
| 3.9 File-Uploads | n/a | Mandat-PDF wird server-seitig generiert und an Mail attached — kein User-Upload. |
| 3.10 Längen-Limits | ✅ Pass | Endpoints nehmen nur Path-Parameter `{id}` (UUID) und Header. Kein Request-Body. |

**Verdikt Security-Smoke-Test: APPROVED — keine kritischen oder High-Findings.**

### Tests-Status

- **Go-Unit-Tests:** 15 von 15 Paketen grün (`go test ./...`). Diff-Helper-Tests (resyncStringTrim, resyncPtrStringTrim, resyncPtrIBAN, normalizeIBANForResync, addrPtr, coreContact*-Helpers) + 2 PROJ-70-Decode-Tests im Coreclient.
- **TypeScript-Build:** clean (Frontend kompiliert ohne Fehler).
- **Playwright-Spec:** `tests/PROJ-70-activated-stammdaten-resync.spec.ts` neu — deckt Auth + Tenant-Isolation auf beide Endpoints + 400-bei-fehlendem-Token ab. UI-Tests warten auf einen aktivierten Test-Antrag im Seed (Manual-QA-Phase).
- **Service-Orchestration-Tests:** nicht als Unit-Test machbar ohne DB-Mock-Refactor (siehe Anmerkung in resync_service_test.go). E2E im Manual-QA-Pfad.

### Manuelle Smoke-Tests (Pflicht vor Deploy)

- [ ] Aktivierter Antrag im Admin-Detail anzeigen → beide Knöpfe sichtbar
- [ ] Antrag in einem anderen Status (z. B. submitted) → beide Knöpfe **nicht** sichtbar
- [ ] Info-Popover-Inhalt prüfen (was wird abgeglichen, was nicht, Mandat-Hinweis)
- [ ] Resync-Klick ohne Änderungen im Core → „Stammdaten sind bereits synchron"-Toast
- [ ] Resync-Klick mit echter Änderung → Diff-Toast (DE-Labels), Detail wird neu geladen
- [ ] Mandat-Resend-Klick → Success-Toast, status_log zeigt neuen Eintrag
- [ ] Mandat-Resend bei einzugsart=kein_sepa → 409 Conflict-Toast

### Empfehlung

**Production-Ready: JA, mit Vorbehalt.**

Code- + Security-Review sauber. BUG-1 (Mail-Template-Wording) ist **Medium** und sollte vor produktiver Aktivierung adressiert werden — Mitglieder bekommen sonst eine missverständliche Mail. BUG-2/3/4 sind Low/Info-Quality-Items, kein Deploy-Blocker.

**Vor Production-Deploy:**
1. BUG-1 entscheiden (eigenes Template, Conditional, oder akzeptieren mit User-Guide-Hinweis)
2. Manuelle Smoke-Tests durchführen
3. AVV-Update Pflicht (siehe Spec-Abschnitt „DSGVO / AVV")
4. User-Guide-Update Pflicht

**Empfehlung: `/security-review`** — neue Admin-Endpoints + Core-Token-Forwarding + PII-Pfade berühren Security-sensitive Bereiche per Skill-Definition. Status-Smoke war clean, aber Deep-Review ist Pflicht-Gate vor Deploy.

## Deployment
_To be added by /deploy_
