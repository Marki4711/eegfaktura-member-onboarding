# PROJ-34: Robuste Import-Recovery (Orphan-Fallback + Pre-Check + Admin-Unstuck)

## Status: In Review
**Created:** 2026-05-14
**Last Updated:** 2026-05-14 (Stages A–F implementiert; konkreter Fehler aus dem Test-Cluster-Log analysiert und behoben)

## Dependencies
- Berührt: PROJ-4 (Core-Import), PROJ-27 (Tarif-Auswahl beim Import), PROJ-30 (Reset eines importierten Antrags auf approved)
- Berührt: Migration 000028 (`uniq_application_rc_member_number`-Partial-Index, der den aktuell beobachteten Fehler triggert)

## Hintergrund

Am 2026-05-14 trat ein konkreter Fehler im Test-Cluster auf, der drei zusammenhängende Schwächen im Import-Pfad sichtbar gemacht hat:

**Beobachteter Fehlerfall (Application aff918e3-37db-4181-a7be-2fbd96855f40):**

```
2026-05-14T19:42:29 ERROR import: bookkeeping failed after successful core insert
target_participant_id: 0aeab3ff-4fcd-11f1-98e4-bed36ef4f0db
db_error: pq: duplicate key value violates unique constraint "uniq_application_rc_member_number"
```

Ablauf:
1. `MarkImportInFlight` setzt `import_started_at` — Lock genommen.
2. `MemberNumberTaken` prüft die member_number gegen die **Core-Teilnehmerliste** → grün (keine Kollision dort).
3. `coreClient.CreateParticipant` → Core legt Teilnehmer mit der Nummer erfolgreich an (UUID `0aeab3ff-…`).
4. `UpdateImportResultTx` will in der Onboarding-DB schreiben → Partial-UNIQUE-Index `(rc_number, member_number) WHERE member_number IS NOT NULL` blockiert, weil **eine andere lokale Application denselben Wert** hat.
5. Transaction rollback. `import_started_at` bleibt gesetzt, `import_finished_at` bleibt NULL → Application steckt für immer in „approved + in-flight".
6. Folge-Klicks auf „Importieren" → 409 „another import is already in progress for this application".

Daraus ergeben sich drei Schwächen, die zusammen den Stuck-State produzieren:

**S1: Orphan-Fallback ist stumm.** Wenn der DB-Bookkeeping-Step nach erfolgreichem Core-Insert fehlschlägt, wird zwar geloggt, aber **die Application-Row wird in keinen sauberen Endzustand überführt**. Der in-flight-Slot bleibt für immer reserviert; der Admin sieht in der UI keine Spur des Problems, sondern nur das immer-409-Verhalten.

**S2: Pre-Check ignoriert die lokale DB.** `MemberNumberTaken` prüft nur den Core. Wenn aus historischen Gründen (alte Migration, halbe Recovery, Bug) eine lokale Application bereits eine `member_number` für diese RC hat, schlägt der Core-Insert trotzdem durch (Core ist nicht unique-constrained auf participantNumber), aber der lokale UPDATE scheitert am UNIQUE-Index. Das ist genau der Eskalationsweg für jeden Edge-Case in der DB.

**S3: Kein Admin-Recovery-Pfad.** Aktuell muss der Admin den Operator (vfeeg) anrufen, der per SQL den Lock räumt. Es gibt keinen UI-Knopf für „dieser Antrag steckt seit X Minuten — manuell reparieren".

## User Stories

- Als **EEG-Admin** möchte ich bei einem stecken gebliebenen Antrag **selbst** den korrekten Endzustand wählen können (als importiert markieren mit Core-UUID, oder Import-Lock räumen für Retry), ohne den Operator zu involvieren.
- Als **EEG-Admin** möchte ich vor dem Import gewarnt werden, wenn eine bereits existierende Application im gleichen EEG dieselbe member_number trägt — vor dem Core-Aufruf, nicht danach.
- Als **vfeeg-Betreiber** möchte ich, dass ein fehlgeschlagener Bookkeeping-Step die Application in einen sauberen `import_failed`-Zustand überführt, damit der bestehende Reset-Import-Flow (PROJ-30) als zweite Verteidigungslinie greift.

## Architekturentscheidungen

1. **Stuck-Detection im Backend:** eine Application ist „stuck" wenn `status='approved' AND import_started_at IS NOT NULL AND import_finished_at IS NULL` UND `import_started_at < NOW() - 2 minutes`. Das 2-min-Fenster matcht das Core-Call-Timeout — alles darüber ist garantiert nicht mehr in Flight, sondern abgebrochen.

2. **Orphan-Fallback wechselt Status:** wenn `UpdateImportResultTx` nach erfolgreichem Core-Insert fehlschlägt, wird in einer **separaten Transaktion** der Antrag auf `import_failed` gesetzt mit Fehlermeldung „Core hat Teilnehmer {ID} angelegt, lokale Verknüpfung fehlgeschlagen: {db_error}". `target_participant_id` und `import_finished_at` werden mit gesetzt, damit der Admin via PROJ-30 die volle Verknüpfung später nachholen kann. Eine fehlgeschlagene Fallback-Transaktion wird zusätzlich geloggt — der Lock bleibt dann doch hängen, aber das ist ein zweites Failure-Layer (DB komplett down).

3. **Lokaler Pre-Check ergänzt Core-Check:** vor `CreateParticipant` zusätzlich `SELECT 1 FROM application WHERE rc_number=$1 AND member_number=$2 AND id != $3 AND member_number IS NOT NULL LIMIT 1`. Wenn Treffer → 409 Conflict bevor irgendwas an den Core geschickt wird. Das verhindert genau das beobachtete Szenario.

4. **Unstuck-UI im Admin:** auf der Application-Detail-Seite erscheint ein orange-Banner wenn der Server-Status den stuck-Zustand meldet. Banner bietet zwei Aktionen:
   - **„Als importiert markieren"** — öffnet einen Dialog für die Core-Teilnehmer-UUID (Admin schaut die im Core nach), schreibt status=imported + target_participant_id + member_number + imported_at, beendet den Lock sauber.
   - **„Import-Lock räumen (Retry)"** — Bestätigungsdialog mit deutlichem Hinweis „dies kann zu einem Duplikat im Core führen, falls der vorige Import dort erfolgreich war". Setzt `import_started_at = NULL`, `import_finished_at = NULL`, status bleibt `approved`. Dann ist „Importieren" wieder klickbar.
5. **Stuck-Flag im Detail-Response:** das bestehende `GET /api/admin/applications/{id}` bekommt ein neues Feld `importStuck: boolean` plus `importStartedAt`-Echo, sodass die Frontend-Banner-Logik nicht nochmal raten muss.

## Acceptance Criteria

### Stage A: Orphan-Fallback in Import-Service
- [ ] `internal/importing/import_service.go`: wenn `s.persistResult()` nach erfolgreichem `CreateParticipant` failed, separat eine **zweite** Transaktion starten die schreibt:
  - `status = 'import_failed'`
  - `import_started_at` unverändert (für Audit)
  - `import_finished_at = NOW()`
  - `target_participant_id = <core-UUID aus dem erfolgreichen Aufruf>`
  - `member_number = NULL` (klar, dass die geplante Nummer nicht angekommen ist) — oder die geplante Nummer? Diskussion: NULL ist sicherer, weil wir nicht wissen ob der Core sie als participantNumber bekommen hat. Default: NULL setzen.
  - `import_error_message = "Core hat Teilnehmer {participantID} angelegt, lokale Verknüpfung fehlgeschlagen: {sanitizedErr}"`
  - Plus `status_log`-Eintrag mit `from='approved'`, `to='import_failed'`, `reason=<message>`.
- [ ] Fehlschlag dieser zweiten Transaktion wird **nochmals geloggt** (kritisch — operator-aufmerksamkeitswürdig); der Lock bleibt dann hängen, aber das ist ein DB-Total-Ausfall-Szenario.
- [ ] Unit-Test: orphan-Pfad führt zu `status=import_failed` und Eintrag im status_log.

### Stage B: Lokaler Pre-Check
- [ ] Neuer Repo-Methode `application_repo.go`: `MemberNumberUsedLocally(rcNumber string, memberNumber string, excludingID uuid.UUID) (bool, error)` — SELECT mit `LIMIT 1`.
- [ ] `import_service.go` `Import`: zwischen `MemberNumberTaken` (Core) und `MarkImportInFlight` zusätzlicher Check via `MemberNumberUsedLocally`. Bei Treffer: Conflict-Error mit klarem Text „Die Mitgliedsnummer {N} ist im Onboarding bereits einem anderen Antrag ({referenceNumber}) zugeordnet."
- [ ] Test: parallel-Application mit gleicher member_number → 409, der Core wird nicht aufgerufen.

### Stage C: Stuck-Detection im Detail-Response
- [ ] `shared.AdminApplicationDetail` (oder das passende Detail-DTO) bekommt zwei neue Felder:
  - `ImportStuck bool `json:"importStuck"`` — true wenn die Definition aus Architektur-Punkt 1 erfüllt ist (approved + in-flight > 2 Min).
  - `ImportStartedAt *time.Time `json:"importStartedAt,omitempty"`` — echo für die Anzeige im Banner.
- [ ] Backend-Handler berechnet `importStuck` aus den DB-Feldern und der aktuellen Zeit.
- [ ] Frontend `src/lib/api.ts` und das Application-Detail-Type spiegeln die neuen Felder.

### Stage D: Unstuck-Endpoints
- [ ] Neuer Endpoint `POST /api/admin/applications/{id}/mark-imported-manually`
  - Body: `{ "targetParticipantId": "<uuid>", "memberNumber": "<string>" }` (beide Pflicht)
  - Validierung: Application muss aktuell im stuck-State sein (`approved + in-flight > 2 Min`); sonst 409.
  - Tx: setzt status=imported, target_participant_id, member_number, imported_at, import_finished_at, import_error_message=NULL; status_log-Eintrag mit reason='Manuell als importiert markiert (Orphan-Recovery)'.
- [ ] Neuer Endpoint `POST /api/admin/applications/{id}/clear-import-lock`
  - Body: `{ "reason": "<text>" }`
  - Validierung: gleicher stuck-State-Check.
  - Tx: setzt import_started_at=NULL, import_finished_at=NULL, status bleibt approved; status_log-Eintrag mit dem reason und from=to=approved (für den Audit-Trail).
- [ ] Beide Endpoints: Keycloak-protected, Tenant-Check via existing helper.

### Stage E: Admin-UI Unstuck-Banner
- [ ] Application-Detail-Seite: wenn `importStuck === true`, zeigt einen orange Alert oberhalb der Status-Actions:
  > „⚠️ Import-Vorgang steckt fest. Letzter Versuch: {importStartedAt}. Bitte wählen Sie eine der folgenden Aktionen, um den Antrag zu reparieren."
  - Button **„Als importiert markieren"** → öffnet Dialog mit zwei Pflichtfeldern (Core-UUID + Mitgliedsnummer), POST `/mark-imported-manually`.
  - Button **„Import-Lock räumen"** → Bestätigungsdialog mit explizitem Warntext über mögliche Core-Duplikate, POST `/clear-import-lock`.
- [ ] Nach erfolgreicher Aktion: Detail-Page reload + grüner Toast.

### Stage F: Dokumentation
- [ ] `docs/api-spec.md`: neue Sektionen für die zwei Endpoints + Hinweis im Import-Endpoint, dass ein 409 jetzt auch aus dem lokalen Pre-Check kommen kann.
- [ ] `docs/operations.md`: Runbook-Sektion „Hängengebliebener Import (Orphan-State)" mit der SQL-Diagnose und Hinweis dass die UI jetzt den Standard-Recovery-Pfad bietet.
- [ ] `docs/user-guide/05-admin-status.md`: erwähnt das Stuck-Banner und die zwei Wahlmöglichkeiten.

## Out of Scope (für PROJ-34)

- **Hintergrund-Janitor:** ein periodischer Worker, der stuck-Anträge nach X Minuten automatisch auf `import_failed` setzt. Wäre eine vollständige Selbstheilung, aber für unsere Volumenebene (≤ 50 EEGs, wenige Imports/Tag) reicht der manuelle Pfad. Backlog-Eintrag.
- **Vollständige Core-Side-Cleanup:** wenn der Admin „Import-Lock räumen" wählt und der Core hat den Teilnehmer bereits → der Orphan-Teilnehmer bleibt im Core. Manueller Aufruf des EEG-Admins in eegFaktura nötig. Wir loggen aber `target_participant_id` falls bekannt.
- **Memory-Adressraum:** keine Änderungen am Import-Code-Pfad selbst (z.B. komplette Async-Variante). Wir reparieren nur den Fail-Modus.

## Pointer-Files

- Spec: `features/PROJ-34-import-recovery.md` (diese Datei)
- Verwandte Code-Stellen:
  - `internal/importing/import_service.go` — Import + persistResult
  - `internal/application/application_repo.go` — MarkImportInFlight, UpdateImportResultTx, ResetImportTx (PROJ-30)
  - `internal/http/admin.go` — Import-Handler + neue Endpoints
  - `src/app/(admin)/admin/applications/[id]/page.tsx` (oder ähnlich) — Detail-Seite
- Related memories: [[eegFaktura Core API contract]]
