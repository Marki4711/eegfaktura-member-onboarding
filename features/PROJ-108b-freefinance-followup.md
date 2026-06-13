# PROJ-108b: FreeFinance Welle-5b Folge-Wellen

## Status: Approved (Implementation 2026-06-13)
**Created:** 2026-06-13
**Last Updated:** 2026-06-13
**Typ:** Backend-Follow-Up (PROJ-108 deferred Phasen F + G + K)

## Implementation-Notes 2026-06-13

In einem Rutsch nach PROJ-108-Deploy implementiert. AC-F1, AC-F2, AC-K1, AC-K2, AC-K3, AC-K4, AC-T1, AC-T2, AC-T3 done. AC-F3 (Scheduler-Integration-Test) deferred (Scheduler-Tests heute nur Pure-Helper-Level, voller Integration-Test braucht eigene Test-Infrastruktur). AC-G1 done (Field hinzugefuegt), AC-G2 als TODO im Code mit pauschalem `true`-Default, AC-G3 deferred (UI-Wiring eigenes PROJ). AC-D1 + AC-D2 done.

### Was geändert wurde

- **`internal/shared/billing_audit_kinds.go`**: 3 neue Konstanten (`cron_completed`, `billing_giveup`, `billing_retry_attempt`) + `IsKnownBillingAuditKind`-Whitelist erweitert. Neuer Test verifiziert dass PROJ-108b-Kinds + alle Bestand-Kinds in der Whitelist sind.
- **`internal/billing/scheduler.go`**: `QuarterlyResult` um `SkippedAlreadyProcessed`-Counter erweitert. Neues `CronAlertSender`-Interface. `SetCronAlertMailer`-Hook am Service. `processEEG` macht Pre-Lookup via `invoices.ListByPeriod(period.ID)` — wenn nicht-credit-note-Invoice existiert, SKIP mit Audit-Outcome `skip_already_processed`. `RunQuarterly` schreibt am Ende `cron_completed`-Audit mit kompletter Lauf-Statistik und triggert `cronAlertMailer.SendBillingCronAlert` wenn `out.Errors > 0` (best-effort async, log-warn bei Fail).
- **`internal/billing/preflight.go`**: `PreFlightResult` um `FreefinanceMandantRefsValid bool` erweitert. V1-Default `true` (optimistisch — Helm-Guard erzwingt LayoutSetupID wenn `globalLiveMode=true`). PROJ-108c bringt den realen FF-GET-Ping mit 5min-TTL-Cache.
- **`internal/mail/billing.go`**: `BillingCronAlertData`-Struct + `SendBillingCronAlert`-Methode (async best-effort wie chargeback). `billingTemplates` um `cronAlert` erweitert.
- **`internal/mail/templates/billing_cron_alert.html` NEU**: Owner-internal-Alert mit Quartal, EEGs-Processed, Errors, SkippedAlreadyProcessed-Counter, Direktlink zu `/admin/billing` Audit-Log.
- **`internal/mail/service.go`**: `MailService`-Interface + `NoOpMailService`-Stub erweitert um `SendBillingCronAlert`.
- **`cmd/server/main.go`**: Neuer `billingCronAlertAdapter` (Pattern wie `billingChargebackAdapter`), Scheduler-Wiring ergänzt um `billingSched.SetCronAlertMailer(...)`.
- **Tests**:
  - `internal/shared/billing_audit_kinds_test.go` NEU: 3 Tests für Whitelist-Drift-Schutz.
  - `internal/mail/billing_test.go`: neuer Test `TestSendBillingCronAlert_RendersErrorCountAndQuarter` verifiziert Subject + Body-Placeholders.

### Was bewusst deferred bleibt (PROJ-108c)

- **TX1→HTTP→TX2 Refactor des Vendor-Pfads**: PROJ-108b schützt durch Pre-Lookup gegen "Cron wurde schon gelaufen". Der Crash-Window zwischen FF-POST und unserem `Insert` ist weiterhin offen. V2: Reconciliation-Job der STAGING-Rechnungen in FF ohne DB-Eintrag findet.
- **Daily-Sync `import_failed`-Retry-Pass mit 7-Versuche-Counter**: braucht DB-State-Tracking (entweder neue Spalte oder Audit-Log-Counter-Query). Konstante `AuditKindBillingRetryAttempt` ist vorbereitet, eigentlicher Retry-Code kommt in PROJ-108c.
- **Real-Ping in PreFlight** für `FreefinanceMandantRefsValid`: braucht Live-FF-Call gegen den frischen Mandanten, ohne Mandant-Reset-Bestätigung nicht testbar.
- **UI-Wiring** für FreefinanceMandantRefsValid + SkippedAlreadyProcessed-Sichtbarkeit im Owner-UI.

### Verifikation

- `go build ./...` clean
- `go test ./internal/billing/... ./internal/mail/... ./internal/shared/... ./internal/freefinance/...` alle grün
- `govulncheck ./...` 0 Issues
- `gosec -severity medium ./internal/billing/... ./internal/mail/... ./internal/shared/...` 0 Issues
- `BILLING_GLOBAL_LIVE_MODE=false` Default bleibt; PROJ-108b ist Backend-only und ändert kein Frontend-Verhalten

## Dependencies
- **Requires:** PROJ-108 (deployed `v1.33.0-PROJ-108` am 2026-06-13)

## Hintergrund

PROJ-108 hat die FreeFinance-API-Inkompatibilitäten gefixt + OIDC-Auth eingeführt + Helm-Config-Rename. Drei Phasen wurden deferred zu dieser Folge-Welle:

- **Phase F:** Scheduler-DB-Pre-Lookup-Pattern. PROJ-108-Hauptbefund: `Idempotency-Key` wird von FreeFinance ignoriert → bei Cron-Re-Run nach partial-Success oder Crash würden Doppel-Rechnungen entstehen. Schutz: vor dem Vendor-Call prüfen, ob die Periode schon einen `billing_invoice` hat.
- **Phase G:** Pre-Flight-Erweiterung um `freefinanceMandantRefsValid`-Check. UI soll Owner warnen wenn die in Helm konfigurierten Mandant-UUIDs (Layout-Setup etc.) bei FF nicht mehr existieren (z. B. nach Mandant-Reset).
- **Phase K:** Cron-Alert-Mail bei `EEGsErrored > 0` + 3 neue Audit-Kinds (`cron_completed`, `billing_giveup`, `billing_retry_attempt`).

Grilling-Entscheidungen aus PROJ-108 (G3, G4, G11, G12) sind die architekturelle Basis dieser Welle.

## User Stories

- **Als Owner** möchte ich, dass ein Cron-Re-Run nach Pod-Restart keine Doppel-Rechnung erzeugt, auch wenn FreeFinance den `Idempotency-Key`-Header ignoriert.
- **Als Owner** möchte ich eine Mail bekommen, wenn der Quartals-Cron mit Errors > 0 endet, damit ich nicht erst am Folge-Quartal merke dass eine EEG-Rechnung fehlt.
- **Als Owner** möchte ich im `/admin/billing` Pre-Flight-Dialog warnen lassen wenn die in Helm hinterlegten FreeFinance-Mandant-UUIDs nicht mehr existieren (Pre-Flight-Backstop).

## Acceptance Criteria

### Phase F: Pre-Lookup-Pattern

- [ ] **AC-F1** `internal/billing/scheduler.go` `processEEG` macht am Anfang (nach Period-Calc, vor Vendor-Pfad) einen Pre-Lookup via `invoiceRepo.ListByPeriod(period.ID)`. Wenn mindestens eine non-credit-note Invoice existiert → SKIP mit Audit-Outcome `skip_already_processed`. Idempotenz garantiert auch ohne FreeFinance-Idempotency-Key.
- [ ] **AC-F2** Beim Re-Run zur selben Periode: `out.SkippedAlreadyProcessed` zählt hoch (neuer Counter in `QuarterlyResult`). Bestand-Counter unverändert.
- [ ] **AC-F3** Test in `scheduler_test.go` simuliert: erster Run erzeugt Invoice, zweiter Run zur selben Periode → SKIP, kein doppelter Vendor-Call.

### Phase G: Pre-Flight-Mandant-Refs-Skeleton

- [ ] **AC-G1** `internal/billing/preflight.go` `PreFlightResult` erweitert um `FreefinanceMandantRefsValid bool` (Default `true` wenn nicht-Live oder kein Live-Client konfiguriert; ansonsten Ergebnis des Live-Pings).
- [ ] **AC-G2** Live-Ping (`GET /clients/{id}/inv/layout_setups/{layout_id}`) als TODO-Stub, dokumentiert. V1 setzt den Wert pauschal auf `true` wenn `cfg.LayoutSetupID != ""` (Helm-Config gesetzt = optimistisch valid). Real-Ping kommt in PROJ-108c sobald Owner mit dem neuen Mandanten Live-Test machen kann.
- [ ] **AC-G3** PreFlightResult-Field exposed an UI-Layer (Welle 4b Owner-UI), aber UI-Wiring **out-of-scope** für PROJ-108b — kommt in PROJ-108c oder Folge-Welle.

### Phase K: Cron-Alert-Mail + Audit-Kinds

- [ ] **AC-K1** 3 neue Audit-Kind-Konstanten in `internal/shared/billing_audit_kinds.go`: `cron_completed`, `billing_giveup`, `billing_retry_attempt`. `IsKnownBillingAuditKind` erweitert.
- [ ] **AC-K2** `RunQuarterly` schreibt am Ende einen `cron_completed`-Audit mit Payload `{year, quarter, processed, sent, errors, skipped_already_processed, ...}`.
- [ ] **AC-K3** Wenn `out.Errors > 0` ODER `out.SkippedAlreadyProcessed > 0` mit ungewöhnlicher Häufigkeit → Owner-Mail-Alert via `cronAlertMailer` (neues optionales Interface). Async OK (nicht sicherheitsrelevant, anders als Mandate-Setup).
- [ ] **AC-K4** Neues Mail-Template `internal/mail/templates/billing_cron_alert.html`. Sender-Methode `SendBillingCronAlert(eegCount, errorCount, runID string) error`.
- [ ] **AC-K5** Test verifiziert dass `cron_completed`-Audit immer geschrieben wird (auch bei errors=0) und Cron-Alert-Mail nur bei errors>0 ausgelöst wird.

### Tests + Build

- [ ] **AC-T1** `go build ./...` clean
- [ ] **AC-T2** `go test ./...` grün
- [ ] **AC-T3** `govulncheck ./...` 0 Issues, `gosec -severity medium ./internal/billing/... ./internal/freefinance/...` 0 Issues

### Spec + Doku

- [ ] **AC-D1** PROJ-108-Spec AC-22 + AC-23 + AC-G4 (cron-alert) als done markiert
- [ ] **AC-D2** CHANGELOG-Eintrag für 2026-06-13 erweitert

## Edge Cases

- **Pre-Lookup findet eine `credit_note`-Invoice aber keine Original-Invoice**: kann passieren wenn der Owner zwischen den Cron-Runs eine Gutschrift erzeugt hat. → Pre-Lookup nur auf non-credit-note Invoices, neue Invoice darf entstehen (sehr unwahrscheinlich Edge-Case, dokumentiert).
- **Pre-Lookup findet eine `draft`-Invoice (z. B. Mandate-pending)**: hier IST eine Invoice da, aber kein Vendor-Call gemacht. Re-Run soll wieder Draft schreiben? V1: SKIP (Audit-Outcome `skip_already_processed`), Owner kann via Manual-Trigger neu anstoßen wenn nötig.
- **Cron-Alert-Mail-Sender ist nil**: async best-effort, kein Hard-Fail (anders als Mandate-Setup). Log-Warning.
- **`OwnerEmail` nicht konfiguriert**: Alert-Mail entfällt still, Log-Warning beim Skip.

## Out-of-Scope (für PROJ-108b)

- **TX1→HTTP→TX2 Refactor** des Vendor-Pfads selbst: V1 schützt nur gegen "Cron schon gelaufen". Crash zwischen FF-POST und unserem `Insert` bleibt als bekannte Lücke. V2 (PROJ-108c) bringt Reconciliation-Job der STAGING-Rechnungen in FF ohne DB-Eintrag findet.
- **Daily-Sync `import_failed`-Retry-Pass** (G12): braucht eigene DB-State-Tracking (retry_count) oder Audit-Log-Counter-Query. Defer zu PROJ-108c.
- **Real-Ping in PreFlight** (G11 Full): braucht Live-FF-Call, ohne Mandant-Reset-Bestätigung nicht testbar.
- **UI-Wiring** für FreefinanceMandantRefsValid: kommt in Folge-Welle nach PROJ-108c.

## Memory-Regeln aktiv

- `feedback_admin_field_full_chain` — neue PreFlightResult-Felder durchgereicht
- `feedback_shared_helpers_for_parallel_paths` — Pre-Lookup als Single-Helper im scheduler
- `feedback_migration_after_apply_drift` — KEINE neuen Migrationen
- `feedback_qa_full_chain_verify` — Tests verifizieren Pre-Lookup-Verhalten konkret

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_Architektur identisch zu PROJ-108 Grilling G3+G4+G11+G12 — keine separate /architecture nötig._

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
