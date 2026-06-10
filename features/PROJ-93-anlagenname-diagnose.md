# PROJ-93: Anlagenname kommt nicht in Faktura an — Diagnose-Log

## Status: Deployed (2026-06-09)
**Created:** 2026-06-09
**Last Updated:** 2026-06-09
**Typ:** Diagnose-Hotfix (Tester-Befund 2026-06-09)
**Severity:** Medium — Daten gehen nicht verloren (DB + Mail haben den Wert), nur die Übernahme in Faktura-Core ist fraglich.

## Hintergrund

Tester-Befund 2026-06-09 (Dani Strasser):

> „Bei der Firmenanmeldung hab ich auch wieder den Anlagennamen versucht
> — findet sich auch in ‚Deine Beitrittserklärung wurde eingereicht
> (RC100387-2026-0007)', kommt aber nicht in der Faktura an."

Beispiel-Antrag: `RC100387-2026-0007`.

## Bisherige Verifikation

Vollständiger Audit des Code-Pfads ergab: alle Layer sind **korrekt verdrahtet**:

| Layer | Befund |
|---|---|
| `src/components/registration-form.tsx:707-709` | Frontend submitted `installationName` korrekt |
| `internal/shared/requests.go:94` | Request-DTO hat `InstallationName *string` mit `validate:"omitempty,max=100"` |
| `internal/application/application_service.go:1767` | Service mappt `req.InstallationName → MeteringPoint.InstallationName` via `trimStringPtr` |
| `internal/application/metering_point_repo.go:33-67` | INSERT/SELECT enthalten `installation_name` |
| `internal/application/metering_point_repo.go:105-107` | scanMeteringPointRow füllt `point.InstallationName` |
| `internal/importing/payload.go:160-162` | Mapping `mp.InstallationName → meter.EquipmentName` |
| `internal/importing/payload.go:90` | JSON-Tag: `EquipmentName string json:"equipmentName,omitempty"` |
| `c:/opt/repos/myeegfaktura/eegfaktura-backend/model/participant.go:83` | Core-Side: `EquipmentName null.String json:"equipmentName,omitempty" db:"equipmentName"` |
| `c:/opt/repos/myeegfaktura/eegfaktura-backend/database/meteringPointDao.go:48` | Goqu-INSERT verwendet `db`-Tag → Spalte `equipmentName` |
| `ba27a6a` (Commit 2026-06-05) | „installation_name war bereits korrekt" (vorheriger Audit) |

**Es ist nichts offensichtlich kaputt.** Die letzten Änderungen am Pfad
(`PROJ-79`, `PROJ-91`) berühren das Meter-Mapping nicht.

## Hypothesen für die Lücke

1. **Faktura-Core deployed Version ist hinter dem Source-Stand**
   → Wire-Format kommt korrekt an, aber gespeichert wird falsch
2. **Tester schaut an der falschen Stelle im Faktura-UI** (Anlagenname
   vs. Anlagen-Nr.)
3. **Trim → Empty-String → omitempty-Drop** wenn nur Whitespace getippt
   wurde — heute sehr unwahrscheinlich, weil Mail den Wert zeigt
4. **EEG-Field-Config setzt `installation_name=admin_only`** und der
   Public-Form hat den Wert über einen alten Browser-Cache eingereicht
   → Service hätte ihn aber sowieso übernommen, kein Wipe

## Diagnose-Schritt (dieser PROJ)

Statt blind „etwas zu fixen", was die Symptomatik nicht erklärt:
ein strukturiertes `slog.Info` direkt vor dem `CreateParticipant`-Call.
Das Log surfact die tatsächlichen Equipment-Felder pro Meter im Payload:

- `application_id`
- `meter_index`
- `metering_point`
- `equipment_name`
- `equipment_number`
- `transformer`

**Kein PII** — nur Anlagen-Metadaten. Log läuft unabhängig vom Erfolg
des Imports.

## Acceptance Criteria

- [x] **AC-1** `slog.Info` Log-Line vor `CreateParticipant` aufgenommen
- [x] **AC-2** Doc-Kommentar verweist auf PROJ-93 + Tester-Befund
- [x] **AC-3** `go build ./...` clean
- [x] **AC-4** CHANGELOG.md-Eintrag im selben Commit

## Edge Cases

- **EC-1** Antrag ohne Meter → kein Log (Import schlägt vorher fehl)
- **EC-2** Anlagenname leer/nicht gesetzt → Log zeigt
  `equipment_name=""` (zur Diagnose hilfreich)

## Reproduktion + nächster Schritt

1. **Owner deployt** den nächsten Bundle (PROJ-86 bis PROJ-93)
2. **Tester re-importiert** den Antrag (ggf. via Reset-Import → Re-Import)
3. **Backend-Pod-Logs filtern** auf
   `import: meter equipment fields in core payload`
4. **Vergleich** zwischen Log-Wert + Faktura-Core-Anzeige
   - Log zeigt korrekten Wert + Faktura zeigt nichts → Core-Side-Bug
     (Faktura-Core fixen)
   - Log zeigt leer → noch ein Layer-Drift im Onboarding (in PROJ-95
     adressieren)

## Out of Scope

- Faktura-Core-Side-Fix (in eegfaktura-backend separat lösen, falls Log
  das bestätigt)
- Field-Config-Bereinigung (eigene PROJ falls relevant)

---

## Deployment

**Deploy-Bookkeeping 2026-06-09 (Abend):**

- Diagnose-Hotfix wie PROJ-86/87/88/89: direkter Commit, kein eigener
  /architecture-Pfad
- Code-Commit: `55b6142`
- Helm-Bump-Commit: `cf756c0` (sha-55b6142)
- Tag: `v1.24.2-PROJ-93` gesetzt + gepusht 2026-06-09 Abend

**Owner-Action:** im nächsten `helm upgrade` mit den anderen Bundle-
Hotfixes. Tester-Verifikation nach Deploy: Antrag re-importieren,
Log-Output aus Backend-Pod prüfen.
