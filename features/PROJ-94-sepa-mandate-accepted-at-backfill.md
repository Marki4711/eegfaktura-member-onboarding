# PROJ-94: SEPA-Mandat-Akzept-am Backfill für Bestand

## Status: Approved
**Created:** 2026-06-09
**Last Updated:** 2026-06-09
**Typ:** Bestand-Backfill-Migration
**Severity:** Low — kein operatives Risiko, aber Bestand-Audit-Trail-PDFs zeigen leeres Datum.

## Hintergrund

Owner-Direktive 2026-06-09 (Abend):

> „Was ich sonst noch hätte: Wäre es nicht sinnvoll, bei Verwendung von
> Audit-Trail ‚SEPA-Mandat akzeptiert am' durch den Zeitstempel automatisch
> auszufüllen?"

## Code-Stand vor PROJ-94

Der Submit-Pfad ([application_service.go:165-167](internal/application/application_service.go#L165-L167))
setzt `sepa_mandate_accepted_at = NOW()` wenn `SepaMandateAccepted=true`.

Der Admin-Edit-Pfad ([application_service.go:390-395](internal/application/application_service.go#L390-L395))
setzt `sepa_mandate_accepted_at = NOW()` wenn der Admin den Toggle auf TRUE
schaltet und das Feld vorher NULL war.

**Beide Pfade sind tight.** Neue Anträge bekommen den Zeitstempel automatisch.

## Lücke: Bestand-Anträge

Das Feld wurde mit Migration 000004 (`add_sepa_fields`) ohne Default-Wert
eingeführt — als `TIMESTAMP WITH TIME ZONE` (nullable). Bestand-Anträge aus
der Frühzeit haben das Feld NULL, obwohl `sepa_mandate_accepted=true`.
Audit-Trail-PDF + Admin-Detail zeigen dort leere Werte.

## Fix

Eine Migration backfilled die NULL-Werte mit `COALESCE(submitted_at, created_at)`.
Das ist der tatsächliche Akzept-Zeitpunkt aus Member-Sicht (zum Submit-Zeitpunkt
wurde der SEPA-Toggle angeklickt).

**Idempotent:** auf einem frisch importierten Cluster ohne NULL-Werte ist die
Migration wirkungslos.

## Acceptance Criteria

- [x] **AC-1** Migration `000075_backfill_sepa_mandate_accepted_at.up.sql`
  erstellt
- [x] **AC-2** Down-Migration ist NO-OP (Backfill nicht symmetrisch
  rückgängig, dokumentiert)
- [x] **AC-3** Doc-Kommentar verweist auf PROJ-94 + Owner-Direktive
- [x] **AC-4** CHANGELOG.md-Eintrag im selben Commit
- [x] **AC-5** Build clean, keine Code-Änderungen (Runtime ist bereits OK)

## Edge Cases

- **EC-1** Antrag mit `sepa_mandate_accepted=false` → nicht betroffen,
  AcceptedAt bleibt NULL (korrekt)
- **EC-2** Antrag mit `sepa_mandate_accepted=true` und AcceptedAt bereits
  gesetzt → WHERE-Klausel filtert das raus, keine Änderung
- **EC-3** Antrag mit `submitted_at IS NULL` (Status=draft) und
  `sepa_mandate_accepted=true` → COALESCE fällt auf `created_at`
- **EC-4** Keine Bestand-Anträge mit der Lücke vorhanden → Migration läuft
  null-Rows durch, idempotent

## Out of Scope

- Future-Proof: zusätzlich `NOT NULL`-Constraint auf das Feld setzen, wenn
  `sepa_mandate_accepted=true` → das wäre ein größerer Eingriff (CHECK +
  schemamigration), nicht im Scope des Backfill-Patches.
- Audit-Trail-Mode-spezifische Backfill-Logik (z. B. nur wenn EEG
  `SEPAMandateCoreAuditEnabled=true`) → über-engineered, Backfill ist
  auch für klassische Mandate harmlos.

## Risiken

- **R1**: COALESCE könnte einen falschen Zeitstempel setzen, wenn
  `submitted_at` und `created_at` beide vor dem eigentlichen Akzept-
  Zeitpunkt liegen. Praktisch unmöglich — der SEPA-Akzept ist ein
  Pflicht-Checkbox im Submit-Form, kann nicht NACH Submit erfolgen.
- **R2**: Bestand-Anträge, die im Admin-Edit nachträglich von
  `sepa_mandate_accepted=false → true` gehoben wurden ohne Code-Pfad,
  könnten verfälscht backfilled werden. Risiko klein, weil aktueller
  Admin-Edit-Code (`AdminUpdateApplication`) ohnehin auto-fillt.

---

## Deployment

**Deploy-Bookkeeping 2026-06-09 (Abend):**

- Hotfix-Cycle: direkter Commit, kein eigener /architecture-Pfad
- Helm-Bump folgt nach CI grün
- Tag: `v1.24.3-PROJ-94`

**Owner-Action:** im nächsten `helm upgrade` mit den anderen Bundle-
Hotfixes. Migration läuft pre-Backend-Update automatisch via Migration-Job.
Tester-Verifikation: ein Bestand-Antrag (Status `approved`+ älter) prüfen —
in der Admin-Detail-Ansicht sollte „SEPA-Mandat akzeptiert am" nach
Migration einen Zeitstempel zeigen, vorher leer.
