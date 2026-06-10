# PROJ-92: ResetImportTx clearet mandate_reference + mandate_date

## Status: Deployed (2026-06-09)
**Created:** 2026-06-09
**Last Updated:** 2026-06-09
**Typ:** Hotfix (Tester-Befund 2026-06-09 Abend)
**Severity:** High — SEPA-PDF zeigt nach Re-Import die alte Mitgliedsnummer

## Hintergrund

Tester-Befund 2026-06-09 (Abend):

> Manuel Wurzinger: „Er schreibt zwar in die e-mail eine neue Referenznummer
> aber im Sepa Formular ist die vom ersten Import. Das muss sich Matthias
> ansehen ich glaub weil das das Formular beim ersten Import erzeugt wurde
> wird das nicht geändert!"
>
> Manuel Wurzinger: „Zur Erklärung: ich hab jetzt versucht eine bereits
> importierten Antrag zurückzunehmen und mit neuer Mitgliedernummer erneut
> zu importieren"

## Root Cause

`ResetImportTx` ([application_repo.go:1150-1164](internal/application/application_repo.go#L1150-L1164))
cleart 9 Spalten beim Reset, aber **NICHT** `mandate_reference` und
`mandate_date`. Beim Re-Import läuft die Service-Layer-Logik
`SetMandateReferenceIfEmpty` ins Leere (Feld ist nicht leer),
die alte Referenz von der ersten Mitgliedsnummer bleibt im
SEPA-Mandat-PDF stehen.

**Folge:** Mail-Text zeigt die neue Mitgliedsnummer, das angehängte
SEPA-PDF aber die alte. Verwirrend für Mitglied + EEG-Vorstand;
operativ ein Compliance-Problem (Mandat referenziert eine
Mitgliedsnummer, die im Core gar nicht mehr existiert).

## Dependencies

- **Voraussetzt:** PROJ-30 (ResetImport-Endpoint), PROJ-47 (Mandate-
  Referenz=Mitgliedsnummer)
- **Berührt nicht:** PROJ-91 — PROJ-92 ist orthogonal zum
  Vorbereitungs-Toggle-Pfad

## Owner-Direktive 2026-06-09

> „Beide sofort fixen: PROJ-92 mandate_reference-Clear + PROJ-93
> Anlagenname-Mapping"

## Acceptance Criteria

- [x] **AC-1** `ResetImportTx`-UPDATE cleart zusätzlich
  `mandate_reference = NULL` + `mandate_date = NULL`
- [x] **AC-2** Doc-Kommentar an der Stelle dokumentiert den Tester-Befund
  + PROJ-92-Begründung
- [x] **AC-3** `go build ./...` + `go test ./...` clean
- [x] **AC-4** CHANGELOG.md-Eintrag im selben Commit (Memory
  `feedback_batch_changelog_with_code`)

## Edge Cases

- **EC-1** Reset auf Antrag der nie importiert war: `mandate_reference`
  + `mandate_date` waren schon NULL → UPDATE auf NULL ist idempotent.
- **EC-2** Reset auf Antrag der ohne SEPA-Mandat importiert wurde
  (`einzugsart=kein_sepa`): `mandate_reference` war schon NULL → siehe
  EC-1.
- **EC-3** Re-Import mit gleicher Mitgliedsnummer (Tester überschreibt
  Suggest manuell mit der alten): `SetMandateReferenceIfEmpty` schreibt
  die wieder, alles konsistent.
- **EC-4** Re-Import mit neuer Mitgliedsnummer (Tester-Befund):
  `mandate_reference` wird sauber neu gesetzt = neue Mitgliedsnummer,
  SEPA-PDF zeigt die neue Referenz. **Gefixt durch PROJ-92.**

## Tech Design (kurz)

Keine /architecture-Phase nötig — 2-Zeilen-Erweiterung in
`ResetImportTx`-SQL plus Doc-Kommentar.

---

## Deployment

**Deploy-Bookkeeping 2026-06-09 (Abend):**

- Hotfix-Cycle wie PROJ-86/87/88/89: direkter Commit, kein eigener
  /architecture-Pfad
- Code-Commit: `192d4ca`
- Helm-Bump-Commit: `28b3b88` (sha-192d4ca)
- Tag: `v1.24.1-PROJ-92` gesetzt + gepusht 2026-06-09 Abend

**Owner-Action:** der `helm upgrade` läuft mit PROJ-91 + PROJ-92
zusammen — PROJ-92 enthält keine Schema-Migration, nur Code-Change im
SQL-String von `ResetImportTx`. Verifikation durch Tester:
ResetImport → Re-Import mit neuer Nummer → SEPA-PDF zeigt die neue
Mitgliedsnummer als Mandatsreferenz.
