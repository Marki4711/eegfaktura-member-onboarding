# PROJ-86: isKnownStatus-Whitelist PROJ-46-Drift-Hotfix

## Status: Approved
**Created:** 2026-06-08
**Last Updated:** 2026-06-08
**Typ:** Bug-Fix (Tag-1-Drift seit 2026-05-17)
**Severity:** High — blockiert manuelle Admin-Status-Transitionen auf alle 3 PROJ-46-Status

## Hintergrund

Owner-Befund 2026-06-08 (Test-Cluster, kurz nach `helm upgrade` für PROJ-82/83/84):

> Antrag im Status „Warte auf Bank-Bestätigung" — Klick auf
> „Bank-Bestätigung erhalten" liefert rotes „Validation failed" unter den
> Statusaktionen-Buttons.

Diagnose: `isKnownStatus()` in [internal/http/admin.go:2996](internal/http/admin.go#L2996)
listet nur 9 von 12 ApplicationStatus-Werten. Die drei PROJ-46-Status
fehlen seit der PROJ-46-Einführung 2026-05-17:

- `awaiting_bank_confirmation`
- `ready_for_activation`
- `activated`

Folge: jeder direkte `POST /api/admin/applications/{id}/status` mit
einem dieser Werte wird vom Backend mit
`400 Validation failed: toStatus = unrecognised status value` abgelehnt.

Das Frontend zeigt nur die generische Top-Level-Message („Validation
failed") und unterdrückt den Field-Error — daher unbemerkt geblieben.

### Warum erst jetzt aufgefallen?

- **Auto-Modus (Activation-Check-Batch)** geht am `ChangeStatus`-Endpoint
  vorbei — direkter Service-Aufruf
- **Vorstands-Workflow (PROJ-76)** geht ebenfalls am Endpoint vorbei
- **PROJ-79-Deploy 2026-06-08** brachte den Owner erstmals in die
  Situation, einen b2b-Antrag manuell von `awaiting_bank_confirmation`
  auf `ready_for_activation` umzustellen

### Sekundär-Befund (nicht in PROJ-86-Scope)

Das Frontend in
[admin-status-actions.tsx:114-117](src/components/admin-status-actions.tsx#L114-L117)
zeigt bei Nicht-Conflict-Fehlern nur `err.message` — nicht die
Field-Errors. Dadurch sind 400er mit detaillierten Field-Errors für den
Admin nicht aussagekräftig. Eigene Spec wenn Owner das fixen will (vermutlich PROJ-87).

## Dependencies

- **Voraussetzt:** PROJ-46 (Status-Modell mit den 3 neuen Status)
- **Berührt nicht:** PROJ-79/82/83/84 — alle live, aber dieser Bug
  existiert seit PROJ-46-Deploy 2026-05-17

## Owner-Direktive 2026-06-08

> „da passt was nicht"

Direkter Hotfix-Auftrag. Keine /grill-me-Phase, kein /architecture —
trivialer 3-Zeilen-Switch-Fix.

## Acceptance Criteria

- [x] **AC-1** `isKnownStatus()` listet alle 12 in
  `shared.ApplicationStatus` definierten Werte
- [x] **AC-2** Doc-Kommentar an der Stelle dokumentiert die PROJ-86-
  Direktive + Tag-1-Bug-Charakteristik (damit das nächste Mal ein
  Status hinzukommt, der Reviewer den Kontext findet)
- [x] **AC-3** Regressions-Test `TestIsKnownStatus_CoversAllApplicationStatuses`
  iteriert über die `shared.Status*`-Konstanten und prüft, dass jeder
  Wert von `isKnownStatus` akzeptiert wird. Test schlägt fehl, wenn ein
  künftiger neuer Status nicht in die Whitelist aufgenommen wurde.
- [x] **AC-4** Regressions-Test `TestIsKnownStatus_RejectsUnknownStrings`
  verifiziert, dass die Funktion case-sensitive bleibt und keine
  Trim/Whitespace-Toleranz hat (Backend-Strict-Mode).
- [x] **AC-5** `go test ./...` + `go build ./...` clean
- [x] **AC-6** Tests-Pattern mit „Spec-Drift" / „Drift-Wache"-Kommentaren
  konsistent zum bereits etablierten Drift-Test-Pattern (PROJ-81
  SEPA-Optional, PROJ-84 EEG-Settings-Validation)

## Edge Cases

- **EC-1 Künftiger internal-only-Status:** der Test hat eine
  `adminUnreachableStatuses`-Map vorbereitet, in die solche Status mit
  Begründung eingetragen werden können, damit der Test sie bewusst
  ausschließt. Heute leer.
- **EC-2 Status-Liste in `shared.ApplicationStatus` ändert sich:** Test
  führt die Liste hartkodiert mit — Reviewer muss bei jeder
  Status-Erweiterung beide Stellen pflegen. Eine reflection-basierte
  Lösung wäre möglich, aber für 12 Werte Overkill.
- **EC-3 Status-Übergangs-Validität:** `isKnownStatus` prüft nur, ob der
  String ein gültiger Enum-Wert ist. Die eigentliche Status-Transition-
  Erlaubnis (z. B. `imported → activated` ist nicht erlaubt) prüft
  weiterhin der Service-Layer mit dem etablierten Pattern.

## Tech Design

3-Zeilen-Switch-Erweiterung. Kein Architecture-Eingriff:

```
isKnownStatus()
  switch shared.ApplicationStatus(s) {
  case StatusDraft, StatusSubmitted, StatusEmailConfirmed,
       StatusUnderReview, StatusNeedsInfo, StatusApproved,
       StatusRejected, StatusImported, StatusImportFailed,
       // PROJ-86 (2026-06-08): Tag-1-Bug-Fix
       StatusAwaitingBankConfirmation, StatusReadyForActivation,
       StatusActivated:
    return true
  }
```

Drift-Schutz via Vitest-äquivalentem Go-Test, der über alle
`shared.Status*`-Konstanten iteriert.

## QA Test Results

**Datum:** 2026-06-08
**Reviewer:** QA Engineer (AI, Solo-Code-Review)
**Status:** Approved

```
$ go test -v -run "IsKnownStatus" ./internal/http/...
PASS  TestIsKnownStatus_CoversAllApplicationStatuses (alle 12 Status grün)
PASS  TestIsKnownStatus_RejectsUnknownStrings (5 Negativ-Cases)
ok    internal/http  0.198s

$ go test ./...
14 Pakete grün
ok    internal/http  0.202s
```

### Security-Smoke

- Status-Transition-Logik (Pflicht-Trigger laut CLAUDE.md): der Fix
  öffnet 3 zusätzliche Status-Werte als „bekannt", **erweitert aber
  nicht die erlaubten Transitions**. Die Status-Transition-Erlaubnis
  prüft weiterhin der Service-Layer.
- Bestand-Risiko: Statuswerte sind bereits seit PROJ-46 in der DB. Der
  Bug hat nur verhindert, dass der Admin über das `/status`-Endpoint
  dorthin wechseln kann. Andere Pfade (Auto-Modus, Vorstands-Workflow,
  Import-Pfad) haben diese Status seit Wochen korrekt gesetzt.
- Kein neuer Endpoint, keine Auth-Änderung, keine Schema-Änderung.
- Defense-in-Depth: Service-Layer-Status-Transitions-Map bleibt
  unverändert als Wahrheits-Anker.

**0 Findings.** `/security-review` nicht erforderlich — die
Status-Transition-Logik selbst ist unverändert; PROJ-86 öffnet nur die
String-Erkennung, nicht die Erlaubnis-Regeln.

## Deployment

**Datum:** 2026-06-08
**Versions-Tag:** `v1.23.4-PROJ-86` (Patch-Bump, reiner Bug-Fix)
**Image-SHA:** wird vom CI nach Push gesetzt
**Status:** wartet auf `helm upgrade` durch Owner

Owner führt `helm upgrade` manuell aus. Da gerade erst der PROJ-82/83/84-
Upgrade lief, ist ein zweiter kurzer Apply nötig.

---
<!-- Sections below are added by subsequent skills -->
