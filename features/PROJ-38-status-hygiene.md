# PROJ-38: Status-Modell-Hygiene & Audit-Fixes

**Status:** Deployed
**Created:** 2026-05-16
**Predecessor:** PROJ-31 follow-up (Audit nach `application_status_check`-Regression)

## Hintergrund

Code-Audit am 2026-05-16 nach dem CHECK-Constraint-Fix (Migration 000036).
Drei echte Findings (von ursprünglich fünf — zwei waren False-Positives).

## Implementierte Fixes

### 1. `isKnownStatus` ergänzt `email_confirmed`

[internal/http/admin.go:2106](../internal/http/admin.go#L2106) — Whitelist
hatte 8 von 9 Status-Werten, `email_confirmed` fehlte. Der Wert wird vom
Admin-`/status`-Endpoint zwar weiterhin via `adminTransitions` als
Ziel-Status abgewiesen, aber der Helper soll der vollständigen
Status-Liste folgen. Defensiv, kein Verhalten-Change.

### 2. `UpdateStatusAdminTx` mit guarded `WHERE status = $expected_from`

[internal/application/application_repo.go:1059](../internal/application/application_repo.go#L1059) —
zusätzlicher Parameter `expectedFrom shared.ApplicationStatus`, plus
`AND status = $8` in der UPDATE-WHERE-Clause. Bei 0 betroffenen Rows
gibt die Methode `shared.ErrConflict` zurück → HTTP 409.

Damit ist der admin-seitige Status-Schreibpfad auf dem gleichen
Schutz-Niveau wie alle anderen `Mark*Tx`-Methoden im Repo. Wenn ein
zukünftiger Code-Pfad die `adminTransitions`-Map vergisst zu konsultieren
ODER zwischen `GetByID` und `Exec` ein paralleler Status-Wechsel
passiert, schlägt die UPDATE jetzt sauber fehl statt still durchzulaufen.

Einziger Caller (`ChangeStatus` in [admin_service.go:527](../internal/application/admin_service.go#L527))
übergibt `app.Status` als `expectedFrom`.

### 3. ResetImport: PROJ-31-Gate als „bewusst nicht" dokumentiert

[internal/application/admin_service.go:756](../internal/application/admin_service.go#L756) —
Audit hatte das Fehlen des `require_email_confirmation`-Gates als
„potentially concerning" markiert. Tatsächlich ist es korrekt: ein
Antrag in Status `imported` wurde bereits einmal akzeptiert. Re-Vetting
der E-Mail beim Reset würde nur ein Szenario betreffen, in dem die EEG
nachträglich die Bestätigung-Pflicht eingeschaltet hat — und das
betrifft sowieso alle historischen `approved`-Zeilen, also out of scope.
Doc-Block ergänzt, damit zukünftige Reviewer die Entscheidung kennen.

## Verworfene Findings (False-Positives)

- **Stale `app.Status` in EEG-Notification-Goroutine** (Audit-Finding 4)
  → Mail-Template referenziert `app.Status` gar nicht
  ([internal/mail/templates/](../internal/mail/templates/), grep ergibt 0),
  also kein beobachtbarer Effekt.
- **Bulk-Action lehnt unbekannte Action nicht ab** (Audit-Finding 5)
  → `BulkActionRequest.Action` hat bereits
  `validate:"required,oneof=approve reject under_review"` in
  [internal/shared/requests.go:380](../internal/shared/requests.go#L380);
  `h.validate.Struct(req)` greift vor dem Map-Lookup.

## Nicht in Scope

- **Submit-Mail-Retry** (Audit „medium"): Fire-and-Forget-Mail kann bei
  Pod-Restart verloren gehen. Retry-Infrastruktur ist eigener Umbau.
- **Auto-Reject-Doppel-Metrik** bei parallelen Pods: kein Daten-Risiko,
  nur Telemetrie-Drift. Akzeptiert.
- **Doku-Korrektur** in `docs/architecture.md`: bereits in Commit
  `3158eaa` (3-place-invariant zeigt jetzt auf `adminTransitions` in
  `admin_service.go`).

## Tests

- Build muss durchlaufen (`go build ./...`)
- Bestehende Unit-Tests müssen weiter laufen
- Manueller Smoke-Test nach Deploy:
  - Confirm-Email-Click auf `TE100200-2026-0002` → erfolgreich
  - Admin-Status-Change auf ungültigen Übergang → 409 mit sprechender Message
  - Admin-Status-Change auf parallel mutierten Antrag → 409 statt Stille
