# PROJ-40: EEG-Umzuordnung eines Antrags im Review

**Status:** Deployed
**Created:** 2026-05-17
**Security-sensitive:** ja — `/security-review` empfohlen vor Prod-Deploy

## Anforderung

> „Es soll möglich sein, einen Beitrittswerber im Zuge der Überprüfung
> einer anderen EEG zuzuordnen."

Mitglied registriert sich über RC-Link der EEG A. Im Review stellt der
Admin fest: der Antrag gehört eigentlich zur EEG B (z.B. weil das
Mitglied den falschen Link geklickt hat oder der Adress-Bereich
woanders hingehört). Bisher müsste das Mitglied neu einreichen — mit
PROJ-40 kann der Admin den Antrag direkt umzuordnen.

## Design-Entscheidungen (Default-Wahl, im Code änderbar)

### Q1 — Wer darf umzuordnen?

**Entscheidung:** Admin muss für **beide** RCs (Quelle + Ziel)
autorisiert sein. Superuser darf alles.

Begründung: ein Single-Tenant-Admin könnte sonst Anträge in fremde EEGs
„schieben". Da der Admin aber oft mehrere EEGs verwaltet (insb. wenn
mehrere benachbarte EEGs vom selben Operator betreut werden), ist
ungebunden auf eigene Tenants akzeptabel.

### Q2 — Welche Status sind umzuordbar?

Erlaubt: `submitted`, `email_confirmed`, `under_review`, `needs_info`.

Verboten: `draft` (nicht sichtbar im Admin), `approved`, `imported`,
`import_failed`, `rejected` (Workflow zu weit fortgeschritten).

### Q3 — Referenznummer behalten oder neu vergeben?

**Entscheidung:** neu vergeben über den per-EEG/per-Jahr-Counter (PROJ-35)
der **neuen** EEG. Die alte Ref wird im status_log-Reason archiviert.

Begründung: die Ref ist member-facing (in Mails, im Confirmation-Screen).
Eine Ref `RCALT-2026-0007` für einen Antrag, der jetzt zu `RCNEU` gehört,
ist verwirrend.

### Q4 — Status / Reason / Audit

- Status bleibt unverändert (kein Status-Wechsel, nur RC-Wechsel)
- Reason ist **Pflicht** (min. 5 Zeichen), Pattern wie reset-import
- status_log-Entry: `from_status = to_status` (gleicher Status), Reason
  enthält die User-Begründung + `[system] previous rc_number=<old>` +
  `[system] previous reference_number=<old>`

### Q5 — Mail an Mitglied?

**Entscheidung:** keine Mail in V1.

Begründung: das Mitglied erfährt die Umzuordnung implizit über die
nächste needs_info-/Approval-/Rejection-Mail, die ja die neue EEG als
Absender hat. Wenn explizite Mail gewünscht: separater Spec PROJ-X.

### Q6 — Cooperative Shares / Field Config / Email-Confirmation-Setting

**Entscheidung:** keine Re-Validierung bei Reassignment. Der bestehende
Datenstand bleibt erhalten:
- `cooperative_shares_count` bleibt wie eingegeben (auch wenn die neue
  EEG andere Pflichtanteile hat)
- Konfigurierbare Felder bleiben wie eingegeben
- `email_confirmed_at` bleibt wie es ist — kein Neu-Triggern der
  Bestätigungs-Mail

Begründung: Reassignment ist ein Admin-Workflow-Tool, kein Re-Submit.
Bei harten Konflikten soll der Admin needs_info nutzen, um nachzufragen.

## Backend

### Endpoint

```
POST /api/admin/applications/{id}/reassign-eeg
Body: { "targetRcNumber": "RC123456", "reason": "Adresse liegt im Versorgungsgebiet der EEG B" }
```

Responses:
- `200 OK` → AdminApplicationDetail (mit neuer rcNumber + neuer referenceNumber)
- `400` Validierung (Reason zu kurz, targetRcNumber leer)
- `403` Admin ist für source oder target nicht autorisiert
- `404` Antrag oder targetRcNumber existieren nicht
- `409` Status nicht umzuordbar / Quelle == Ziel / target ist nicht is_active

### Service

`AdminApplicationService.ReassignEEG(id, targetRcNumber, reason, actorID, allowedRCNumbers)`
- Lädt App, prüft Status
- Lädt source + target Entrypoints, prüft is_active
- Tenant-Check: actor muss für `app.RCNumber` UND `targetRcNumber` autorisiert sein (oder Superuser; `allowedRCNumbers == nil`)
- Generiert neue Referenznummer via bestehenden Counter für target RC
- TX: `UPDATE application SET rc_number=$, reference_number=$ WHERE id=$ AND rc_number=$old AND status IN (...)`
  + StatusLog-Entry
- Commit

### Repo

`UpdateRCNumberTx(tx, id, expectedFromRC, newRC, newRefNumber)` — guarded
UPDATE mit `WHERE id=$1 AND rc_number=$2 AND status IN (...)`. Bei 0
Rows: `shared.ErrConflict`.

## Frontend

- Neuer Button **„EEG umzuordnen"** im Statusaktionen-Block — sichtbar
  nur wenn (a) der Status reassignable ist UND (b) der Admin Zugriff auf
  ≥ 2 EEGs hat (Single-EEG-Admin braucht die Funktion nicht)
- Dialog: Dropdown der verfügbaren Target-RCs (alle aus der
  `tenant`-Claim des Admins, ohne die aktuelle), Reason-Textarea,
  Hinweis „Beim Umzuordnen wird eine neue Referenznummer vergeben."
- POST gegen den neuen Endpoint, on success → reload

## Out of Scope

- Bulk-Reassign (mehrere Anträge gleichzeitig)
- Member-Notification (siehe Q5)
- Re-Trigger der Bestätigungs-Mail bei Wechsel auf eine EEG mit
  require_email_confirmation = true (siehe Q6)
- Cross-Tenant-Reassignment für Admins, die nur Source ODER Target
  verwalten — Superuser-Route

## Tests

- Build muss grün bleiben
- Smoke-Test nach Deploy:
  - Antrag in EEG A einreichen, Admin (mit Zugriff auf A+B) klickt
    „Umzuordnen → EEG B"
  - In der Detail-Ansicht erscheint die neue RC + neue Referenz
  - Im status_log steht der old-RC + old-ref archiviert
  - Antrag erscheint in der Liste unter EEG B, nicht mehr unter A

## Security-Review-Punkte

- `checkTenantAccess` für source + target — beide müssen bestehen
- Source = Target → 409 (kein Bypass)
- Target muss `is_active = true` sein
- UpdateRCNumberTx hat `WHERE id=$ AND rc_number=$expected AND status IN (...)`
  als zweite Verteidigungslinie
