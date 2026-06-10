# PROJ-90: customer_onboarding_submission Schema-Drift-Aufholung

## Status: In Progress
**Created:** 2026-06-09
**Last Updated:** 2026-06-09
**Typ:** Hotfix (Schema-Drift seit 2026-06-06)
**Severity:** High — blockiert die komplette PROJ-71-Plattform-Selbst-Buchung auf test

## Hintergrund

Tester-Befund 2026-06-09:

> „Ich wollte gerade die ‚Buchung' über Vertrag & Onboarding-Status durchführen
> und habe die beiden Zustimmungen akzeptiert. Funktioniert nicht mit dem
> Fehler ‚an internal error occurred'"

Pod-Log auf test (2026-06-09 05:33):

```
ERROR customer-onboarding: submit failed
error: create submission: insert submission: pq: column "avv_pdf" of
       relation "customer_onboarding_submission" does not exist
       at position 5:4 (42703)
```

### Root Cause: Migration nach Apply geändert

Migration `000062_create_customer_onboarding_submission.up.sql` wurde am
2026-06-06 (Commit `2b1d96f`, PROJ-71-Variante-B-Refactor) inhaltlich
komplett umgeschrieben:

- **Vorher (Variante A):** Stammdaten-Spalten (`legal_form`, `vereinsname`,
  `uid_number`, `billing_*`, `board_*`), kein PDF-Blob, Legal-Form-Check-Constraint
- **Nachher (Variante B):** Stammdaten in `registration_entrypoint` (per PROJ-32
  Core-Sync), Submission speichert nur Akzept-Audit + `avv_pdf BYTEA NOT NULL`

Auf test-Cluster stand `schema_migrations.version=62` bereits vom 2026-06-05 —
**vor dem Refactor**. Die neue Definition wird also nie ausgeführt. Tabelle hat
noch Variante-A-Struktur. Service-Code erwartet aber `avv_pdf`-Spalte (siehe
[internal/customeronboarding/repository.go:77-92](internal/customeronboarding/repository.go#L77-L92)).

Diagnose-Trail:
1. Tester-Befund 2026-06-09 morgens
2. Pod-Log per `kubectl logs` → konkrete pq-Fehlermeldung
3. `git log -- db/migrations/000062_...up.sql` zeigt 4 Commits — eindeutiger
   „Migration nach Apply"-Drift

### Warum erst jetzt aufgefallen?

- PROJ-71-Variante-B-Refactor 2026-06-06 lief auf test-Cluster nicht durch,
  weil `schema_migrations.version=62` bereits schon gesetzt war
- PROJ-71 Mega-Session ging fokussiert in Code-Reviews + Security-Fixes;
  Buchung wurde nicht direkt getestet (Owner ist superuser-only, Tester
  greifen den Pfad zuerst an)
- Drei Tage Stillstand zwischen Refactor und Tester-Klick

## Dependencies

- **Voraussetzt:** PROJ-71-Variante-B-Refactor (Commit `2b1d96f`, 2026-06-06)
- **Blockiert:** komplette PROJ-71-Buchung auf jedem Cluster, wo Migration
  000062 mit Variante-A-Inhalt gelaufen ist

## Owner-Direktiven 2026-06-09

| # | Entscheidung |
|---|---|
| 1 | Bestand auf test: `DELETE FROM` ok (Buchung lief seit 2026-06-06 nicht durch — keine Variante-A-Daten zu erhalten) |
| 2 | Workflow: direkt Migration + Spec + Commit + helm upgrade (Hotfix-Stil wie PROJ-86/87/88/89) |
| 3 | Prod-Schema-Check nach Apply auf test (Owner verifiziert mit `\d` ob prod denselben Drift hat) |

## Acceptance Criteria

- [ ] **AC-1** Migration `000073_customer_onboarding_submission_schema_realign.up.sql`
  + `.down.sql` existieren und sind syntaktisch valide
- [ ] **AC-2** Up-Migration:
  1. `DELETE FROM member_onboarding.customer_onboarding_submission` (Bestand droppen)
  2. `ADD COLUMN IF NOT EXISTS avv_pdf BYTEA NOT NULL`
  3. `DROP COLUMN IF EXISTS` für alle 11 Variante-A-Stammdaten-Spalten
  4. `DROP CONSTRAINT IF EXISTS cos_legal_form_valid`
- [ ] **AC-3** Down-Migration stellt Variante-A-Spalten + Constraint wieder her
  (Symmetrie-Anker, ohne Bestands-Rekonstruktion — geht nicht aus avv_pdf-Blob)
- [ ] **AC-4** Doc-Kommentar in der Up-Migration dokumentiert das
  „Migration nach Apply"-Drift-Pattern + Datum + Commit-SHA, damit zukünftige
  Reviewer den Kontext finden (verweist auf Memory `feedback_migration_after_apply_drift`)
- [ ] **AC-5** Nach Apply auf test:
  `\d member_onboarding.customer_onboarding_submission` zeigt `avv_pdf bytea NOT NULL`
  und kein `legal_form`/`vereinsname`/`billing_*`/`board_*`
- [ ] **AC-6** Tester-Verification: Plattform-Buchung über „Vertrag & Onboarding-Status"
  läuft mit beiden akzeptierten Zustimmungen erfolgreich durch (Status 201 statt 500)
- [ ] **AC-7** Owner-Aktion: Prod-Schema mit `\d` prüfen ob derselbe Drift vorliegt;
  falls ja, dort ebenfalls helm upgrade fahren

## Edge Cases

- **EC-1** Bestehende `owner_rejected`-Submissions auf test (falls vorhanden):
  werden vom `DELETE FROM` mit erwischt. Begründung: kein audit-relevanter Bestand,
  Variante-A-Phase war Test-Zeitraum.
- **EC-2** Multi-Apply (Idempotenz): `ADD COLUMN IF NOT EXISTS` + `DROP COLUMN IF EXISTS`
  laufen auch bei mehrfacher Ausführung sauber durch.
- **EC-3** Down-Up-Down-Zyklus: Down-Migration kann den Variante-A-Bestand nicht
  rekonstruieren (Stammdaten-Spalten leer); akzeptiert als bewusste Asymmetrie.
- **EC-4** Prod hat noch alte Codebase + alte Tabelle: kein Drift, kein Handlungsbedarf,
  solange Variante-B-Code dort nicht ausgerollt ist.

## Implementation Plan

1. **DONE:** Migration `000073_customer_onboarding_submission_schema_realign.up.sql` +
   `.down.sql` auf Disk
2. INDEX-Eintrag PROJ-90 + Next Available auf PROJ-91
3. CHANGELOG.md-Eintrag im selben Commit (Memory `feedback_batch_changelog_with_code`)
4. Commit + Push: `fix(PROJ-90): Realign customer_onboarding_submission schema after Variante-B-Refactor-Drift`
5. CI watchen (Build + Push Docker Images)
6. Helm-Auto-Bump-Commit via `git pull --rebase` abholen
7. Git-Tag `v1.23.8-PROJ-90` setzen + pushen
8. Owner-Action: `helm upgrade` auf test (Migration-Job läuft)
9. Owner-Action: Prod-Schema-Check, ggf. helm upgrade auch dort
10. Tester-Verification der Buchung

## Lessons Learned

Siehe Memory `feedback_migration_after_apply_drift`: jede Migration ist nach
ihrem ersten Apply auf irgendeinem Cluster IMMUTABLE. Inhaltliche Änderungen
kommen als neue Aufhol-Migration mit nächster Nummer — niemals durch
Re-Editieren der bestehenden Datei. Drei Tage Bug-Window wegen Verstoß
gegen diese Regel.
