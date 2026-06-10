# PROJ-95: Tester-Bundle 2026-06-10 (drei Mail-/Edit-Fixes)

## Status: Deployed (2026-06-10)
**Created:** 2026-06-10
**Last Updated:** 2026-06-10
**Typ:** Hotfix-Bundle (Tester-Befund 2026-06-10)

## Tester-Befunde

Aus dem Tester-Chat 2026-06-10 morgens nach dem PROJ-86–94-Deploy.

### Befund A — Anlagenname verschwindet nach Admin-Edit

> „RC100387-2026-0009 hat die Namen leider wieder verloren."

PROJ-93-Diagnose-Log am test-Cluster:

```
application_id=45336e0e-... meter_index=0  equipment_name="" equipment_number="" transformer=""
application_id=45336e0e-... meter_index=1  equipment_name=""
application_id=39837b4c-... meter_index=0  equipment_name=""
```

`BuildPayload` mappt unschuldig — der Wert ist schon vor dem Payload-Build
leer. Onboarding-Drift, NICHT Core-Side-Bug.

**Root Cause:** `src/components/admin-edit-form.tsx` baut den Save-Payload
ohne `transformer`, `installationNumber`, `installationName` und ohne die
vier PROJ-39-Zaehlpunkt-Adressfelder. Backend
`metering_point_repo.CreateBulkTx` macht beim Update ein
`DELETE FROM metering_point WHERE application_id=X; INSERT mit den neuen
Werten`. Felder die im Payload fehlen → `req.* = undefined` → backend
`trimStringPtr(nil)=nil` → DB-INSERT setzt NULL → bei jedem Admin-Edit
verloren.

Memory [[feedback_shared_helpers_for_parallel_paths]] hatte die Drift-
Falle schon dokumentiert; `BuildMeteringPointFromRequest`-wipe-on-edit
war einer der vorherigen Beispiele.

### Befund B — Mail nutzt Mitgliedsnummer trotz manuell vergebener Mandatsreferenz

> „Ich habe eine Mandatsreferenz manuell vergeben — die steht in den
> PDFs und auch in Faktura, so wie sie sein soll. Aber die Mail
> ignoriert das und verwendet stattdessen die Mitgliedsnr."

**Root Cause:** `mandateAtImportData` (`internal/mail/service.go:847`) hat
nur `MemberNumber`, kein `MandateReference`-Feld. Die Templates
`application_imported_member.html` + `application_imported_eeg.html`
rendern hardcoded `{{.MemberNumber}}` als Mandatsreferenz. PDF + Core-
Payload nehmen korrekt den Admin-Override — die Mail-Text-Schicht
nicht.

### Befund C — „Hallo ," Leerzeichen vor Komma bei Firmen-SEPA

> „Das Leer zwischen Hallo und Komma müsste noch weg."

**Root Cause:** Templates `application_imported_member.html` +
`application_activated_member.html` rendern
`Hallo {{if .Firstname}}{{.Firstname}} {{end}}{{.Lastname}},`. Bei
Firmen-Mandat sind beide leer → `Hallo ,`. Die anderen drei Member-
Templates haben eine inkonsistente Variante mit `else → "zusammen"`.

## Implementation

### Backend

- **`internal/mail/service.go`** — neuer Template-Helper `greetingName`
  in der `templateFuncs`-Map. Trimt + Fallback auf „zusammen".
- **`internal/mail/service.go`** — `mandateAtImportData` erweitert um
  `MandateReference` + `IsCustomMandateReference`. `buildMandateAtImportData`
  berechnet beide:
  ```
  mandateRef = COALESCE(TrimSpace(app.MandateReference), memberNumber)
  custom     = mandateRef != memberNumber
  ```

### Frontend

- **`src/components/admin-edit-form.tsx`**:
  - `FormMeteringPoint` erweitert um `transformer`, `installationNumber`,
    `installationName` und die vier PROJ-39-Adress-Felder.
  - Initial-State-Map aus `application.meteringPoints` zieht die Felder durch.
  - Save-Payload (`.map((mp) => ({...}))`) gibt die Felder zurück
    (`mp.transformer?.trim() || undefined` etc.).

### Mail-Templates

- `application_submitted_member.html`
- `application_needs_info_member.html`
- `application_rejected_member.html`
- `application_imported_member.html`
- `application_activated_member.html`

→ alle auf `Hallo {{greetingName .Firstname .Lastname}},` unifiziert.

- `application_imported_member.html` — vier Mandatsreferenz-Stellen
  von `{{.MemberNumber}}` auf `{{.MandateReference}}{{if not .IsCustomMandateReference}} (entspricht deiner Mitgliedsnummer){{end}}`.
- `application_imported_eeg.html` — zwei Stellen analog mit Custom-Hint
  „(vom Admin manuell vergeben)".

PDF-Filenamen bleiben `sepa-mandat-{MemberNumber}.pdf` /
`sepa-firmenlastschrift-mandat-{MemberNumber}.pdf` — File-Naming-Konvention
ist member-number-stable, nicht mandate-reference-abhaengig.

## Acceptance Criteria

- [x] **AC-1** Admin-Edit-Save mit Anlagenname → `installation_name`
  wird in der DB nicht NULL.
- [x] **AC-2** Manuelle Mandatsreferenz im Admin-Edit → Mail-Text zeigt
  den Override-Wert, nicht die Mitgliedsnummer. „(entspricht …)"-Zusatz
  entfaellt.
- [x] **AC-3** Firmen-Mandat ohne Firstname/Lastname → Mail rendert
  `Hallo zusammen,` ohne Leerzeichen-Bug.
- [x] **AC-4** Bestand-Verhalten unveraendert: Mandatsreferenz NICHT
  manuell vergeben → Mail zeigt weiter Mitgliedsnummer + „(entspricht
  deiner Mitgliedsnummer)".
- [x] **AC-5** `go build ./...`, `go test ./...`, `npx tsc --noEmit`,
  `npx vitest run` alle clean.
- [x] **AC-6** CHANGELOG.md-Eintrag im selben Commit.

## Edge Cases

- **EC-1** Admin-Edit ohne Aenderungen an einem Zaehlpunkt: existing
  Anlagen-Felder werden 1:1 zurueckgeschickt, Backend re-inserted
  unveraendert.
- **EC-2** Admin loescht Anlagenname im Edit (leerer String): wird via
  `trim() || undefined` zu `undefined` → DB-INSERT NULL → korrekt.
- **EC-3** Mandatsreferenz exakt gleich der Mitgliedsnummer: `custom=false`
  → Mail rendert weiter den „(entspricht deiner Mitgliedsnummer)"-Hinweis.
  Genau so gewollt, weil semantisch identisch.
- **EC-4** Mandatsreferenz nur Whitespace: `TrimSpace` filtert vor dem
  Vergleich, kein false-positive.

## Out of Scope

- TODO-3 (Block-Reihenfolge in Bestaetigungs-Mail) — Tester-Screenshot
  steht aus.
- TODO-4 (Du/Sie-Konsistenz Firmen-SEPA) — Owner-Klaerung erforderlich.

## Deployment

**Deploy-Bookkeeping 2026-06-10 (morgens):**

- Hotfix-Cycle: direkter Commit, kein eigener /architecture-Pfad
- Code-Commit: `be167d3`
- Helm-Bump-Commit: `aa80bcf` (sha-be167d3)
- Tag: `v1.25.0-PROJ-95` gesetzt + gepusht (Minor — Mail-Template-Schema-
  Aenderung + Frontend-Field-Add)

**Owner-Action:** im naechsten `helm upgrade`. Tester-Verifikation:
1. Antrag mit Anlagenname submitten, Admin-Edit oeffnen + speichern
   ohne Aenderung am Zaehlpunkt → `installation_name` bleibt in der DB
   (kein Wipe). Erneuter Import zeigt `equipment_name` im
   PROJ-93-Diagnose-Log und in Faktura.
2. Antrag importieren mit manuell ueberschriebener Mandatsreferenz →
   Mail-Text + PDF + Faktura zeigen alle dieselbe Referenz.
3. Firmen-Antrag importieren → Mail-Anrede „Hallo Musterbetrieb GmbH,"
   bzw. „Hallo zusammen," (je nach gespeicherten Member-Daten),
   kein Leerzeichen-Komma-Bug.
