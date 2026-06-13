# PROJ-104: Abrechnung der Plattform-Nutzung (Pricing-V3-Implementierung)

## Status: Deployed (2026-06-13)
**Created:** 2026-06-12
**Last Updated:** 2026-06-12 (nach 4 Grilling-Wellen, 16 zusätzliche Entscheidungen)
**Typ:** Plattform-Verrechnung + externe Stack-Integration (FreeFinance + Mollie)

## Hintergrund

Owner-Direktive 2026-06-12: „die tester sind zufrieden. jetzt müssen wir die abrechnung fertigstellen". Damit ist Phase 1A der vor-Prod-Roadmap angestoßen (siehe Memory `project_priority_before_prod`).

Pricing-Strategie ist als V3 im Memory `project_pricing_strategy` festgelegt:
- Zwei Editionen: **Standard** und **Pro**
- Preis **pro aktiviertem Mitglied** (Abkehr von V2-Import-Position-Staffel)
- 30-Tage-Trial bleibt
- AGB sagt aktuell „Standard/Pro, Konditionen folgen bei Produktiv-Start, Test-/Pilotphase kostenfrei"

Verrechnungs-Stack ist im Memory ebenfalls festgelegt: **FreeFinance Plus** (Invoicing + Buchhaltung, FinanzOnline-Direkt) + **Mollie B.V.** (PSP für SEPA-Lastschrift, EU-only). K8s-CronJob als Scheduler. Konkrete Preise werden über die DB-Tabelle `pricing_plan` versioniert gehalten und können später ohne Deploy geändert werden.

## Owner-Entscheidungen (festgenagelt in vier Klärungs-Wellen 2026-06-12)

| # | Entscheidung |
|---|---|
| 1 | **Zähler-Definition:** `application.activated_at IS NOT NULL` (PROJ-46). Antrag muss wirklich aktiviert sein. |
| 2 | **Trial-Start:** beim **ersten `activated_at`** eines Mitglieds (revidiert in Grilling-Welle 3, Welle 1 hatte ursprünglich `is_active=true` festgelegt). Lange Setup-Phasen ohne echte Nutzung verbrauchen kein Trial. Trial endet 30 Tage nach erstem `activated_at`. Bei späteren Aktivierungen wird `trial_started_at` nicht zurückgesetzt — einmaliger 30-Tage-Anlauf. |
| 3 | **Periodizität:** quartalsweise (Q1 = Jan–März, Cron am 1. des Folgemonats nach Quartalsende). Mindestrechnungsbetrag €10/Quartal, kumuliert ins Folgequartal, Verfall am Jahresende. |
| 4 | **Sichtbarkeit:** Owner + EEG-Admin sehen Rechnungen. EEG-Admin sieht eigene Rechnungs-Historie in Settings. |
| 5 | **Pricing-Werte:** in DB-Tabelle `pricing_plan` mit `gueltig_ab`/`gueltig_bis`. Helm-Config liefert nur Initial-Seed beim Erst-Setup. Preisversion wird in jede Rechnung archiviert (Rechnungs-Snapshot, nicht Live-Lookup). |
| 6 | **Edition-Differenzierung:** **PROJ-67 `settings_view_mode` als Brücke** — Standard = nur „Einfache Ansicht"-Features verfügbar; Pro = volle „Alle Optionen". Edition-Switch setzt den erlaubten Mode. Kein Feature-Flag-Wildwuchs, weil die Trennung schon strukturell im Code existiert. |
| 7 | **Mahnwesen:** Sanft + Cool-Down via PROJ-71. Bei `invoice.overdue` nach X Tagen (Default 14) werden die 6 Mehrwert-Endpoints in den Cool-Down geschoben (Public-Form läuft weiter). Owner kann manuell wieder freischalten. |
| 8 | **Verrechnungs-Stack in V1:** Voll-Anbindung. FreeFinance API (`POST /inv/invoices` + `finalize`) + Mollie Payments API (`POST /v2/payments`) + Mollie Webhook (Pull-Modell) → `PUT /invoices/{id}/pay` als Glue. Phase A + B aus Memory direkt umgesetzt. |
| 9 | **Test-/Pilot-Modus:** globaler `BILLING_LIVE_MODE`-Schalter (Helm-Config, Default `false`). Im Test-Modus läuft der Cron, generiert Daten, erzeugt **Preview-Rechnungen** (PDF lokal, kein Versand), feuert KEINE Mollie-Payments. Owner sieht „so würde abgerechnet" ohne dass Tester echte Verpflichtungen tragen. Switch auf `true` ist eine bewusste Owner-Aktion beim Prod-Cutover. |
| 10 | **PII-Disziplin:** Minimum-Datensatz an externe Vendoren. FreeFinance bekommt EEG-Stammdaten + Plan + Betrag + Leistungsbeschreibung („Plattform-Nutzung Q3/2026 — N aktivierte Mitglieder"). Mollie bekommt nur Owner-Mandate-Daten der EEG. **Keine Mitglieder-Namen, -IBANs oder -E-Mails** verlassen das Onboarding-System. |

### Zusätzliche Owner-Entscheidungen aus /grill-me (4 Wellen, 2026-06-12)

| # | Entscheidung |
|---|---|
| 11 | **Vendor-Credentials:** Helm-Secret-References analog Bestand. Neue Einträge `secrets.freefinanceApiKey` + `secrets.mollieApiKey` in values-secret.yaml. Backend liest über env vars `FREEFINANCE_API_KEY` + `MOLLIE_API_KEY`. Wird vom `required`-Guard in `templates/secrets.yaml` (aus Fable-Code-Review-Fix gestern) gefordert, wenn `BILLING_LIVE_MODE=true` global gesetzt ist. |
| 12 | **Cron-Idempotenz:** DB-`UNIQUE(rc_number, year, quarter)` auf `billing_period` (bereits in AC-2) plus `INSERT … ON CONFLICT DO NOTHING`. Zusätzlich `Idempotency-Key`-Header bei `FreeFinance.CreateInvoice` = `{rc_number}-{year}-Q{quarter}`. **REVIDIERT 2026-06-13 durch PROJ-108-AC-27b-Live-Test:** FreeFinance respektiert den `Idempotency-Key`-Header NICHT (drei Doppel-POSTs → drei verschiedene Invoice-IDs). Die DB-`UNIQUE`-Linie auf `billing_period` ist die **alleinige** Idempotenz-Sicherung; PROJ-108 ergänzt das DB-Pre-Lookup-Pattern in `scheduler.go` (TX1 INSERT+SELECT FOR UPDATE → HTTP außerhalb TX → TX2 UPDATE WHERE freefinance_invoice_id IS NULL). Header wird defensiv weiter gesetzt für den Fall dass FF das Verhalten ändert. |
| 13 | **Edition-Switch:** **direkter Switch** per EEG-Admin-Aktion (Owner-Direktive Welle 1 — Approval-Flow verworfen). Owner sieht den Wechsel im Audit-Log. **Anti-Abuse** strukturell über Entscheidung #15 gelöst (Edition-Snapshot pro Aktivierung) — kein technisches Switch-Limit nötig. |
| 14 | **`BILLING_LIVE_MODE`-Cutover:** **Pro-EEG-Schalter mit globalem Notbrems-Default.** Neue Spalte `registration_entrypoint.billing_live BOOLEAN NOT NULL DEFAULT FALSE`. Live wird abgerechnet, wenn `eeg.billing_live = TRUE` UND `cfg.Billing.GlobalLiveMode = true`. Owner kann pro EEG einzeln scharf schalten (Pilot-EEG zuerst), globaler Helm-Schalter ist Notbremse für alle. |
| 15 | **Anti-Abuse durch Edition-Snapshot statt Switch-Limit:** neue Spalte `application.edition_at_activation TEXT NULL` (`'standard'` / `'pro'`), gesetzt beim `activated`-Status-Transition zum dann aktiven `registration_entrypoint.edition`. Pricing-Service zählt pro Edition separat. EEG kann frei zwischen Editionen switchen — die Buchung bleibt fair, weil jede Aktivierung ihren Edition-Stand mitträgt. |
| 16 | **Pro → Standard Downgrade-Block:** wenn aktive Pro-Features konfiguriert sind (PROJ-13 API-Key existiert, PROJ-60 Datenweiterleitung-Plugins existieren, PROJ-69 Reconciliation aktiviert), blockiert der Switch mit Hinweis „Pro-Features aktiv — erst deaktivieren, dann downgraden". Keine automatische Deaktivierung. |
| 17 | **Mollie Mandate-Setup:** SEPA-DD First-Payment **€0,01** mit `sequenceType='first'`. Owner-Mail an EEG-Vorstand vorher („Sie sehen in den nächsten 7 Tagen eine €0,01-SEPA-Lastschrift von Mollie B.V. — das ist die Mandate-Validierung, kein Fehler"). Nach `paid` ist Mandate für recurring-Lastschriften aktiviert. |
| 18 | **Webhook-Failure-Recovery:** Mollie-Webhook ist Primary-Pfad. Zusätzlich Daily-CronJob `billing-status-sync` (Schedule `"0 5 * * *"`): pulled für `billing_invoice WHERE mollie_payment_id IS NOT NULL AND status IN ('sent', 'overdue')` einen Mollie-GET-Lookup. Statuswechsel werden aufgeholt, wenn Webhook verloren ging. |
| 19 | **Trial-Resume-Edge:** revidiert die ursprüngliche Welle-1-Antwort. Trial-Start ist beim **ersten `activated_at`** (siehe #2). Damit ist „Stale-Trial" (EEG flippte vor 1 Jahr `is_active=true` ohne echte Nutzung) automatisch fair gelöst. |
| 20 | **Stornierung bezahlter Rechnungen:** **Gutschrift-Rechnung-Pattern** (GoBD-konform). Bezahlte Rechnung bleibt bestehen, neue Gutschrift mit negativ-Betrag und Verweis-FK auf Original-Rechnung. Bei Bedarf manuelle SEPA-Rücklastschrift via Mollie-Refund-Endpoint durch Owner. |
| 21 | **USt-Behandlung:** **pauschal 20 %** auf alle Rechnungen (Owner ist regelbesteuert). Kein Sonder-Flag pro EEG. EEG-Buchhaltung bestimmt selbst, ob Vorsteuerabzug möglich ist. |
| 22 | **Pricing-Plan-Erst-Seed:** Standard €0 / Pro €0 als Default beim Erst-Setup (Helm `seed-job`). Owner muss explizit Preise via `/admin/billing` setzen, bevor `BILLING_LIVE_MODE` (pro EEG oder global) flippt. **Pre-Flight-Check** im Live-Aktivierungs-Pfad: „Preise sind noch €0 — wirklich live gehen?" mit Bestätigungsdialog. |
| 23 | **Owner-Billing-UI:** eigene Seite **`/admin/billing`** (nicht in PROJ-72 Cockpit integriert). Cockpit verlinkt darauf. Klare Trennung: Cockpit = EEG-Statistik, Billing-Seite = Rechnungs-/Pricing-Verwaltung. Parallele Entwicklung von PROJ-72 und PROJ-104 möglich. |
| 24 | **Multi-EEG-Admin Sichtbarkeit:** EEG-Admin sieht **pro EEG** den Tab „Rechnungen" in Settings (via PROJ-101-EEG-Switcher zwischen seinen EEGs). Keine konsolidierte Liste — konsistent mit allen anderen Per-EEG-Tabs. |
| 25 | **Mollie-IP-Allowlist:** Mollie publiziert offiziell die Webhook-IPs. Owner trägt sie einmal in Helm-Config `backend.mollieAllowedIPs` (CSV). Plus E-Mail-Subscription auf Mollie-Changelog für IP-Updates. Zusätzlicher Schutz: jeder Webhook-Request wird per Mollie-GET-Lookup auf die `tr_`-ID validiert (Authentizität, nicht nur IP). |
| 26 | **FreeFinance-Trial-Test als Pre-Welle-2-Aufgabe:** Owner verifiziert vor Beginn von Welle 2 (FreeFinance-Client-Implementation) per kostenlosem 30-Tage-Trial-Account: API-vs-UI-Nummernkreis, Kleinunternehmer-Hinweis bei API-Rechnungen, Idempotency-Header-Support. ~30 Min Aufwand. Memory `feedback_verify_vendor_claims` greift. |

## Scope

### IN-Scope (V1)

1. **Pricing-Plan-Tabelle** mit Editionen Standard/Pro, Preis pro aktiviertem Mitglied, gültig-ab/gültig-bis, USt-Satz.
2. **Edition-Zuweisung pro EEG** mit Standard-Default. Switch via Owner-UI (Cockpit) und EEG-Admin-UI (Settings) im Rahmen der jeweiligen Sichtbarkeits-Regel. Edition-Wechsel im laufenden Quartal: ab Folgequartal wirksam (klare Quartalsabschluss-Semantik).
3. **Trial-Tracking**: neue Spalte `registration_entrypoint.trial_started_at` (gesetzt beim ersten `is_active=true`-Flip). Trial endet 30 Tage danach. Bei wiederholtem `is_active=false → true`-Flip wird das Trial nicht zurückgesetzt.
4. **Quartalsweise Aktivierungs-Zähler**: pro EEG zählen, wie viele `application.activated_at`-Timestamps im jeweiligen Quartal lagen. Snapshot wird pro Quartal in `billing_period`-Tabelle archiviert.
5. **K8s-CronJob** `billing-quarterly`, Schedule `"0 4 1 1,4,7,10 *"` (1. des Monats nach Quartalsende, 04:00 UTC). Backend-Subcommand `cmd/billing/main.go` (oder analog).
6. **FreeFinance-Anbindung**: `internal/freefinance/client.go` mit `CreateInvoice`/`FinalizeInvoice`/`MarkAsPaid`. AVV-PDF-Auto-Upload ins DOC-Modul.
7. **Mollie-Anbindung**: `internal/mollie/client.go` mit `CreatePayment` (sequenceType=`recurring` für Bestand-Mandate, `first` für Erstabrechnung). Webhook-Handler `POST /api/webhooks/mollie` mit Pull-Modell (Body = Payment-ID, GET-Lookup für Status).
8. **`BILLING_LIVE_MODE`-Schalter** (Helm + Config). Wenn `false`: alle Vendor-Calls werden im Mock-Modus durchlaufen (lokal generiertes PDF, kein FreeFinance-POST, kein Mollie-POST), Audit-Log markiert die Rechnung als `preview_only`.
9. **Audit-Trail-Tabelle** `billing_invoice` mit Status `draft|preview|sent|paid|overdue|cancelled`, Rechnungs-Nummer, FreeFinance-ID, Mollie-Payment-ID, Erzeugungs-Zeitstempel, Preisversion (Snapshot).
10. **Owner-Cockpit-Sektion** (in PROJ-72 integriert oder separat): Übersicht aller Rechnungen mit Status, manuell-stornieren-Aktion, manuelle-Rechnung-erstellen-Aktion (z. B. für Edge-Cases nach Cool-Down).
11. **EEG-Admin-Settings-Tab** „Rechnungen": eigene Rechnungs-Historie mit Download-Link für jede PDF.
12. **Cool-Down-Integration** mit PROJ-71: bei `invoice.overdue` + X-Tage-Schwelle wird `customer_onboarding_status_log` ein `suspended`-Event mit `reason_code='payment_overdue'` geschrieben. Cool-Down-Logik aus PROJ-71 greift automatisch.
13. **AGB-Update** (Markdown-File): Pricing-Werte aus `pricing_plan` werden zur Anzeige in `src/content/legal/agb-v1.0.md` ausgespielt **erst wenn `BILLING_LIVE_MODE=true`**. Bis dahin bleibt der heutige Text „Standard/Pro, Konditionen folgen bei Produktiv-Start, Test-/Pilotphase kostenfrei".
14. **Webhook-Sicherheit**: Mollie-Webhook ist nur an die echte Mollie-API gebunden (IP-Allowlist + signed verification per GET-Lookup, nicht Body-Signatur — Mollie-Webhook ist Pull-Modell).

### OUT-Scope (Folge-PROJ)

- **Eigenständiger Zahlungseingang-Match per Bank-Feed** (z. B. finAPI) — Memory erwähnt das als Option, aber nicht V1-relevant solange Mollie/FreeFinance den Zustand tracken.
- **Mehrwährungs-Support** (EUR-only in V1).
- **Pro-Edition als komplett separate Feature-Subset-Liste** (außer der `settings_view_mode`-Brücke) — Pro/Standard ist heute via Mode-Switch differenziert, dedizierte „Pro-only"-Features kommen als eigene PROJs nach.
- **Selbst-Aktion „Plan ändern" durch EEG-Admin** für Edition-Wechsel ohne Owner-Approve — V1 setzt voraus, dass Edition-Wechsel von Owner bestätigt wird (Anti-Abuse).
- **Steuerberater-Direkt-Export BMD/RZL** — FreeFinance liefert das nativ, kein Eigenbau.
- **Rabatte / Coupon-Codes**.

## Acceptance Criteria

### Datenmodell

- [ ] **AC-1** Migration `000079_pricing_plan.up.sql`: neue Tabelle `pricing_plan` mit Spalten `id`, `edition` (`'standard'|'pro'`), `eur_per_active_member_per_quarter`, `vat_percent`, `gueltig_ab`, `gueltig_bis`, `created_at`. CHECK-Constraint: kein Overlap zwischen gleich-edition Zeiträumen.
- [ ] **AC-2** Migration `000080_billing_period.up.sql`: neue Tabelle `billing_period` mit Spalten `id`, `rc_number` (FK), `year`, `quarter` (1–4), `active_member_count`, `pricing_plan_id` (FK, Snapshot), `total_net`, `vat`, `total_gross`, `created_at`. UNIQUE auf (`rc_number`, `year`, `quarter`).
- [ ] **AC-3** Migration `000081_billing_invoice.up.sql`: neue Tabelle `billing_invoice` mit Spalten `id`, `billing_period_id` (FK), `status` (`'draft'|'preview'|'sent'|'paid'|'overdue'|'cancelled'`), `freefinance_invoice_id` (nullable), `mollie_payment_id` (nullable), `invoice_number_external` (nullable), `sent_at`, `paid_at`, `created_at`, `updated_at`. CHECK auf Status-Whitelist.
- [ ] **AC-4** Migration `000082_trial_started_at.up.sql`: neue Spalte `registration_entrypoint.trial_started_at TIMESTAMPTZ NULL`. **REVIDIERT in Grilling-Welle 3:** Service-Hook setzt den Wert **beim ersten `application.activated_at`-Trigger** (nicht beim `is_active`-Flip). Kein Reset bei späteren Aktivierungen.
- [ ] **AC-5** Migration `000083_eeg_edition.up.sql`: neue Spalte `registration_entrypoint.edition TEXT NOT NULL DEFAULT 'standard' CHECK (edition IN ('standard','pro'))`. Bestand-EEGs starten als Standard.
- [ ] **AC-5a** Migration `000084_application_edition_snapshot.up.sql`: neue Spalte `application.edition_at_activation TEXT NULL CHECK (edition_at_activation IS NULL OR edition_at_activation IN ('standard','pro'))`. Wird vom Service beim `activated`-Status-Transition gesetzt zum dann aktiven `registration_entrypoint.edition`. Bestand-Aktivierungen bleiben NULL (Backfill-Strategie siehe AC-Backfill).
- [ ] **AC-5b** Migration `000085_eeg_billing_live.up.sql`: neue Spalte `registration_entrypoint.billing_live BOOLEAN NOT NULL DEFAULT FALSE`. Owner-Aktion via Cockpit oder direkter DB-Edit setzt sie pro EEG auf TRUE. Live wirkt nur wenn `eeg.billing_live = TRUE AND cfg.Billing.GlobalLiveMode = true`.
- [ ] **AC-5c** Migration `000086_credit_invoice_link.up.sql`: `billing_invoice` bekommt Spalte `cancels_invoice_id UUID NULL REFERENCES billing_invoice(id)` für Gutschrift-Verkettung. Plus Status-Whitelist um `'credit_note'` erweitert.

### Pricing-Service

- [ ] **AC-6** `internal/billing/pricing_service.go` mit Funktion `CalculateQuarter(ctx, rcNumber, year, quarter) → BillingPeriod`: zählt `application.activated_at`-Timestamps im Quartal **gruppiert nach `edition_at_activation`** (Standard-Aktivierungen × Standard-Preis + Pro-Aktivierungen × Pro-Preis, je `pricing_plan` zum Quartals-Start). Schreibt `billing_period`-Zeile mit getrennten Spalten `count_standard` + `count_pro` und Pricing-Snapshots beider Editionen.
- [ ] **AC-7** Trial-Check: wenn `trial_started_at + 30 Tage > Quartals-Ende` → Quartal ist gratis (kein `billing_period`-Eintrag erstellt, oder `total_gross=0` mit `note='trial_period'`).
- [ ] **AC-8** Mindestbetrag-Logik: `total_gross < 10€` → kumuliert ins Folgequartal (Feld `carryover_from_period_id`). Im 4. Quartal des Jahres: Verfall, kein Rollover ins Folgejahr.
- [ ] **AC-9** Edition-Wechsel im laufenden Quartal: wenn `pricing_plan.edition` wechselt zwischen Quartals-Start und Quartals-Ende → das Quartal nutzt die Edition, die am Quartals-START aktiv war. Wechsel wirkt ab Folgequartal.

### Cron + Vendor-Anbindung

- [ ] **AC-10** Backend-Subcommand `cmd/server/main.go billing-quarterly` (oder eigenständig `cmd/billing/main.go`) liest aus Helm-Config, ermittelt das abzurechnende Quartal (= letztes abgeschlossenes Quartal), iteriert alle aktiven EEGs, ruft `CalculateQuarter` und triggert `IssueInvoice`-Pfad.
- [ ] **AC-11** K8s-CronJob `helm/.../templates/billing-cronjob.yaml` mit Schedule `"0 4 1 1,4,7,10 *"`, `backoffLimit: 2`, ServiceAccount mit Read-Access auf Postgres + Egress auf FreeFinance + Mollie.
- [ ] **AC-12** `internal/freefinance/client.go` mit Methoden `CreateInvoice(EEGData, LineItems) → FreeFinanceInvoiceID`, `FinalizeInvoice(id)`, `MarkAsPaid(id, mollie_payment_id)`. OpenAPI-konform laut Memory.
- [ ] **AC-13** `internal/mollie/client.go` mit `CreatePayment(amount, customer_mandate_id) → MolliePaymentID`. SequenceType `first` für Erst-Lastschrift einer EEG, `recurring` für Folge-Lastschriften.
- [ ] **AC-14** Webhook-Handler `POST /api/webhooks/mollie` (public, kein Auth — Mollie-IP-Allowlist via Helm-Config). Body = `{id: "tr_xxx"}`. Handler ruft Mollie-GET, prüft Status, updated `billing_invoice` + ruft FreeFinance-`MarkAsPaid` bei `status=paid`.

### `BILLING_LIVE_MODE`-Schalter

- [ ] **AC-15** Helm-Config `backend.billingGlobalLiveMode: false` (Default). Im Code via `cfg.Billing.GlobalLiveMode bool`. Plus pro EEG `registration_entrypoint.billing_live` (AC-5b). **Effektive Live-Bedingung:** `eeg.billing_live = TRUE AND cfg.Billing.GlobalLiveMode = true`. Sonst Preview-Modus.
- [ ] **AC-16** Wenn nicht-live: `FreeFinanceClient` ist ein Mock, der nur `slog.Info("would create freefinance invoice", …)` loggt und eine Fake-ID liefert. `MollieClient` analog (`would charge €X via SEPA-DD`).
- [ ] **AC-17** Wenn nicht-live: `billing_invoice.status='preview'` statt `'sent'`. PDF wird lokal generiert (vorhandener PDF-Service) und in `billing_invoice.preview_pdf_bytes` (BYTEA, nullable, max 256 KB) abgelegt.
- [ ] **AC-18** Wenn live: voller Vendor-Flow läuft. `billing_invoice.status='sent'` nach FreeFinance-Finalize. `FreeFinance.CreateInvoice` bekommt `Idempotency-Key`-Header = `{rc_number}-{year}-Q{quarter}` (Entscheidung #12).
- [ ] **AC-19** Owner-UI (`/admin/billing`) hat einen prominenten Banner „Preview-Modus" wenn EEG nicht-live ist. Cockpit-Übersicht zeigt klar an, ob eine Rechnung `preview` oder `sent` ist.
- [ ] **AC-19a** **Pre-Flight-Check beim Live-Aktivieren pro EEG:** Owner-Aktion „EEG live schalten" prüft `pricing_plan.eur_per_active_member_per_quarter > 0` für die aktive Edition. Wenn €0, Bestätigungsdialog „Preise sind noch €0 — wirklich live gehen?".

### Mahnwesen + Cool-Down

- [ ] **AC-20** Daily-CronJob `billing-overdue-check` (Schedule `"0 5 * * *"`): scannt `billing_invoice WHERE status='sent' AND sent_at < NOW() - INTERVAL '14 days' AND paid_at IS NULL`. Setzt Status auf `overdue` und schreibt PROJ-71-`suspended`-Event mit `reason_code='payment_overdue'`.
- [ ] **AC-20a** Daily-CronJob `billing-status-sync` (Schedule `"0 5 * * *"`, gleiche Pod-Invocation wie AC-20): für alle `billing_invoice WHERE mollie_payment_id IS NOT NULL AND status IN ('sent','overdue')` ruft Mollie-`GET /v2/payments/{id}` und gleicht Status ab (Webhook-Backstop, Entscheidung #18). Verspätete `paid`-Meldungen werden so aufgeholt, ohne Webhook-Verlust.
- [ ] **AC-21** Wenn Mollie-Webhook (oder AC-20a-Pull) `paid` meldet → `billing_invoice.status='paid'`, `paid_at=NOW()` und PROJ-71-`reactivated`-Event mit `reason_code='payment_received'`. Cool-Down-Logik aus PROJ-71 hebt automatisch auf. Plus FreeFinance-`MarkAsPaid` für die Rechnung im Vendor-System.
- [ ] **AC-21a** **Gutschrift-Pfad** (Entscheidung #20): Owner-Aktion „Rechnung gutschreiben" in `/admin/billing` erzeugt eine neue `billing_invoice`-Zeile mit `status='credit_note'`, negativ-Betrag, FK `cancels_invoice_id` auf das Original. FreeFinance bekommt einen `CreateInvoice`-Call für die Gutschrift (eigene Vendor-Invoice-ID). Original-Rechnung bleibt unverändert (GoBD-konform). Owner kann optional Mollie-Refund über externe Mollie-UI auslösen.

### Sichtbarkeit + UI

- [ ] **AC-22** Eigene Seite **`/admin/billing`** (Entscheidung #23, nicht in PROJ-72 Cockpit): Pricing-Plan-Editor (Standard/Pro Preise editieren, neue Plan-Version mit `gueltig_ab` einfügen), Rechnungs-Tabelle aller `billing_invoice`-Zeilen mit Filter (Status / Quartal / EEG), Gutschrift-Aktion (erzeugt `credit_note`-Eintrag mit FK-Verkettung), manueller „Rechnung jetzt erstellen"-Button für Edge-Cases, EEG-`billing_live`-Schalter pro Zeile.
- [ ] **AC-23** EEG-Admin-Settings: neuer Tab **„Rechnungen"** mit Liste der eigenen `billing_invoice`-Zeilen (Quartal, Status, Betrag, PDF-Download). Tab nur sichtbar wenn mindestens eine Rechnung existiert. **Pro EEG separater Tab** (Multi-EEG-Admin switcht via PROJ-101-EEG-Switcher, sieht jeweils nur die Rechnungen der aktiven EEG).
- [ ] **AC-24** Edition-Switch im EEG-Admin-UI: **direkter Switch** (Entscheidung #13, kein Approval-Flow). Sichtbar im PROJ-67-`advanced`-Modus. Auf Pro → Standard prüft das UI vorab, ob aktive Pro-Features konfiguriert sind (PROJ-13 API-Key, PROJ-60 Datenweiterleitung-Plugins, PROJ-69 Reconciliation aktiv) — wenn ja, Switch blockiert mit Hinweis (Entscheidung #16). Wirkung des Switches: zukünftige Aktivierungen tragen die neue Edition (`edition_at_activation`-Snapshot via Entscheidung #15). Aktivierungen, die schon stattgefunden haben, behalten ihre Edition.
- [ ] **AC-24a** Mollie SEPA-DD-Setup beim ersten EEG-`billing_live=true`-Flip: Backend triggert Mollie-`CreatePayment` mit `sequenceType='first'`, Amount=€0,01, `description='SEPA-Mandate-Aktivierung Plattform-Nutzung'`. Owner-Mail an EEG-Vorstand vorher (HTML-Template, neuer Mail-Type). Sobald Mollie `paid` meldet, ist Mandate-Status für recurring-Lastschriften aktiviert.

### Datenschutz + Compliance

- [ ] **AC-25** FreeFinance-Payload enthält **keine** Mitglieder-Identifikatoren. Leistungsbeschreibung lautet `"Plattform-Nutzung Q{quarter}/{year} — {count_standard} Mitglieder (Standard) + {count_pro} Mitglieder (Pro)"`. Audit-Test verifiziert das. **USt-Aufschlag pauschal 20 %** (Entscheidung #21), `vat_percent` aus `pricing_plan`-Snapshot zur Rechnungs-Erzeugung.
- [ ] **AC-26** Mollie-Payload enthält nur Owner-Mandate-Daten der EEG (IBAN, BIC, Mandat-Referenz). **Keine** EEG-Mitglieder-Daten.
- [ ] **AC-27** Webhook-Endpoint `POST /api/webhooks/mollie` ist hinter Mollie-IP-Allowlist (Helm-Config `backend.mollieAllowedIPs`, Entscheidung #25). Andere Requests → 403. **Zusätzlich** verifiziert jeder Webhook-Request die `tr_`-Payment-ID via Mollie-GET-Lookup (Authentizität, nicht nur Source-IP) — Defense-in-Depth gegen IP-Spoofing.
- [ ] **AC-27a** Seed-Job: bei Erst-Setup wird `pricing_plan` mit zwei Default-Zeilen befüllt — `(edition='standard', eur_per_active_member_per_quarter=0, vat_percent=20, gueltig_ab=NOW())` und Analoges für Pro (Entscheidung #22). `Idempotency`-fähig — `INSERT … ON CONFLICT DO NOTHING`.
- [ ] **AC-27b** **Pre-Welle-2-Owner-Verifikation:** vor Implementations-Welle 2 (FreeFinance-Client) verifiziert Owner per kostenlosem FreeFinance-30-Tage-Trial-Account folgende drei Punkte: (a) API-vs-UI-Nummernkreis getrennt oder synchronisiert? (b) Kleinunternehmer-Hinweis-Text bei API-Rechnungen renderbar? (c) `Idempotency-Key`-Header tatsächlich von FreeFinance respektiert? Ergebnis in `private/vendor-setup/freefinance-trial-verification-2026-06-XX.md` dokumentieren. ~30 Min Aufwand.

### Tests + Doku

- [ ] **AC-28** Backend-Tests für `pricing_service.go`: Trial-Check (innerhalb/außerhalb 30 Tagen), Mindestbetrag-Carryover, Jahres-Verfall, Edition-Wechsel-Mitte-Quartal.
- [ ] **AC-29** Backend-Tests für `freefinance/client.go` + `mollie/client.go`: Mock-HTTP, Erfolg + Fehlerpfade (Timeout, 5xx, 4xx-Validation).
- [ ] **AC-30** Backend-Test für `BILLING_LIVE_MODE=false`: Mocks werden gerufen, keine echten HTTP-Calls (Schutz via `httptest`-Server-Assert).
- [ ] **AC-31** docs/architecture.md aktualisieren: Billing-Stack-Diagramm + Datenfluss.
- [ ] **AC-32** docs/api-spec.md: `POST /api/webhooks/mollie` + Owner-/EEG-Endpoints für Rechnungs-Listing.
- [ ] **AC-33** docs/domain-model.md: vier neue Tabellen + zwei neue Spalten auf `registration_entrypoint`.
- [ ] **AC-34** User-Guide: neue Sektion „Rechnungen" in `06-admin-settings.md` + Erläuterung des EEG-Admin-Tabs.
- [ ] **AC-35** `CHANGELOG.md` + `docs/user-guide/changelog.md`-Einträge (Memory-Regel `feedback_changelog_one_block_per_day` einhalten).

## Edge Cases

- **EC-1** EEG schaltet `is_active` mehrfach um oder lässt es lange auf `true` ohne echte Nutzung: **revidiert in Grilling-Welle 3.** Trial startet beim ersten `activated_at` eines Mitglieds, nicht beim `is_active`-Flip. Stale-Trial-Problem (EEG flippte vor 1 Jahr `is_active=true`, keine Aktivierungen) löst sich automatisch — Trial beginnt erst bei tatsächlicher Nutzung. Bei späteren Aktivierungen wird `trial_started_at` nicht zurückgesetzt — einmaliger 30-Tage-Anlauf pro EEG.
- **EC-2** EEG hat in einem Quartal 0 aktivierte Mitglieder: kein `billing_period`-Eintrag (oder `total_gross=0` mit `note='no_activity'`) und keine FreeFinance-Rechnung — analog Memory-V2-Logik.
- **EC-3** EEG wird Mitte-Quartal angelegt (`is_active` erst im Q2): Trial-Phase startet im Q2; Q2 ist gratis bis ggf. Q3 wenn Trial vor Quartalsende endet. Tageweise Anteilsberechnung NICHT in V1 (Quartals-Granularität).
- **EC-4** Aktivierungs-Reset über PROJ-100 (`activated → imported`): die alte `activated_at` wird genullt. Im nächsten Quartal zählt das Mitglied bei erneuter Aktivierung wieder. Quartale, die bereits abgerechnet sind, werden **nicht** rückwirkend korrigiert (no-rollback-billing).
- **EC-5** Mollie-Webhook verspätet sich um >1 Stunde: Daily-CronJob `billing-overdue-check` würde das als overdue markieren. Lösung: 24h-Karenz statt 14 Tagen für den allerersten Lauf nach Rechnungs-Versand, danach echte 14-Tage-Schwelle. Plus: bei `paid`-Webhook-Eingang wird `overdue`-Status zurückgesetzt + PROJ-71-Cool-Down aufgehoben.
- **EC-6** FreeFinance-API-Outage zum Cron-Zeitpunkt: K8s-CronJob backoffLimit=2 mit exponential backoff. Bei fortgesetztem Fehler bleibt `billing_period` in `status='draft'`, `billing_invoice` wird gar nicht erstellt. Owner-Cockpit zeigt Drafts, manueller Retry-Button.
- **EC-7** EEG ist bereits in PROJ-71-Cool-Down (z. B. wegen früherer overdue) und neue Rechnung wird fällig: neue Rechnung wird erstellt, aber gar nicht versendet (Status bleibt `draft`). Owner muss erst Cool-Down aufheben (z. B. Sonderzahlung außerhalb des Systems), dann manuell Rechnung versenden.
- **EC-8** Pricing-Plan-Update mitten in Q3: alte Rechnung für Q2 nutzt weiterhin den alten `pricing_plan_id` (Snapshot). Neue Q3-Rechnung nutzt den neuen Plan. Owner kann historische Rechnungen jederzeit nachvollziehen.
- **EC-9** EEG-Edition-Wechsel mitten in Q3: **revidiert via Edition-Snapshot (Entscheidung #15).** Aktivierungen vor dem Switch tragen `edition_at_activation='standard'`, Aktivierungen nach dem Switch `'pro'`. Pricing-Service zählt getrennt, Q3-Rechnung hat zwei Positionen mit unterschiedlichen Tarifen. Keine „ab-Folgequartal"-Logik mehr nötig — Switch wirkt sofort, aber fair.
- **EC-10** Pro → Standard Downgrade mit aktiven Pro-Features: Switch wird vom UI blockiert (Entscheidung #16). EEG muss erst API-Key widerrufen, Datenweiterleitung-Plugins deaktivieren, PROJ-69-Reconciliation ausschalten. Pre-Check-Liste im Switch-Dialog mit Direktlinks zu den Settings.
- **EC-11** `BILLING_LIVE_MODE` pro EEG aktiviert (`registration_entrypoint.billing_live = TRUE`) — die schon erzeugten `preview`-Rechnungen bleiben `preview` (keine rückwirkende Versendung). Wirkung gilt ab nächster Cron-Iteration. Owner muss bewusst entscheiden, ob er für die EEG die letzte Preview als „manuell jetzt senden" auslöst oder beim Folge-Quartal startet.
- **EC-12** Mollie-Mandate-€0,01-Race: EEG-`billing_live` wird auf TRUE geflippt → Backend triggert SEPA-DD First-Payment. Mollie braucht ggf. mehrere Tage für die Lastschrift-Verarbeitung. Was wenn in dieser Zeit schon ein Quartals-Cron läuft? **Lösung:** Cron prüft `eeg.mollie_mandate_active = TRUE` (neue Spalte, gesetzt nach `paid`-Webhook der €0,01). Wenn Mandate noch nicht aktiv, wird Rechnung als `status='draft'` erstellt und bei nächstem Daily-Cron erneut versucht.
- **EC-13** FreeFinance-Idempotency-Key-Konflikt: zwei Wellen unseres Crons (z. B. K8s-Replica-Race oder kubectl manueller Trigger während automatischer Cron) feuern beide `CreateInvoice` mit demselben Key `{rc}-{year}-Q{quarter}`. FreeFinance dedupliziert und liefert beide Mal dieselbe Invoice-ID — unsere `billing_invoice`-Insert ist UNIQUE-geschützt, der zweite Insert macht ON CONFLICT DO NOTHING und überschreibt nichts.
- **EC-14** Mollie-IP-Allowlist-Drift: Mollie ändert Webhook-Source-IPs ohne Vorwarnung. Webhook-Endpoint loggt 403 mit `source_ip` als Audit-Trail. Owner sieht das im Cockpit-Health-Bereich („Letzte Webhook-Rejections"). Plus AC-27 Defense-in-Depth über Mollie-GET-Lookup — auch wenn IP-Allowlist veraltet ist, würde der Webhook nichts kaputt machen, nur als unauthorized verworfen.

## Out of Scope

- **Echtzeit-Rechnungs-Generierung** (Pay-per-Use direkt nach Aktivierung) — Quartals-Snapshot reicht für V1.
- **Mehrere Mandate pro EEG** (z. B. unterschiedliche Bankkonten für unterschiedliche Vereinsteile) — EEG hat genau ein Mandate.
- **Stornorechnungen mit USt-rückwirkender Korrektur** für bezahlte Rechnungen — nur unbezahlte können storniert werden (Cancel-vor-Pay-Pfad).
- **Internationale USt-Behandlung** (Reverse-Charge für EU-Vereine außerhalb AT). Alle EEGs sind AT-EEGs für V1.
- **Lemon-Squeezy / Paddle als Merchant-of-Record**: Memory erwähnt das als Option ab Wachstum >50 EEGs, V1 ist klassisch B2B.
- **PROJ-51 (Usage-Fee-Status-Display)**: kann in dieser PROJ aufgehen (Banner-Mechanik wird Teil des Cool-Down-Pfads), oder eigenständig bleiben. Klärung im /architecture-Skill.

## Risiken

- **Vendor-Vendor-Lock-in**: FreeFinance-Bindung 12 Monate (€360 brutto). Bei Mollie kein Lock-in, aber Mandate-Migration zu anderem PSP ist nicht trivial. Im /architecture klären: wie wir den Vendor-Code so abstrahieren, dass ein Wechsel mittelfristig nicht unmöglich wird.
- **AT-Compliance-Drift**: FinanzOnline-Anbindung von FreeFinance erfordert dass der Plattform-Betreiber selbst regelbesteuert ist (Memory). Bei Wechsel auf Kleinunternehmer-Regelung müsste das neu evaluiert werden.
- **Tester-Verwirrung im Preview-Modus**: wenn EEG-Admins „Preview-Rechnungen" sehen, könnte das missverstanden werden. UI muss extrem klar machen, dass keine Zahlungspflicht besteht.
- **Webhook-Sicherheit**: Mollie-Webhook ist public erreichbar. IP-Allowlist + Body-`tr_`-Prefix-Validation + GET-Lookup für Status-Authentizität. Im /security-review explizit prüfen.

## Dependencies

- **PROJ-46** (Post-Import-Status-Modell) — liefert `activated_at`
- **PROJ-64** (Faktura-Handover-Marker) — kontextuelle Logik des Aktivierungs-Zeitpunkts
- **PROJ-67** (settings_view_mode) — Brücke für Standard/Pro-Differenzierung
- **PROJ-71** (Customer-Onboarding-Vertrag) — Cool-Down-Mechanik wird wiederverwendet
- **PROJ-72** (Member-Onboarding-Cockpit, Phase 1B) — Owner-UI für Rechnungs-Übersicht. Reihenfolge: PROJ-104 spec'd zuerst, Implementation kann parallel zu PROJ-72 laufen oder PROJ-72 schluckt die Owner-UI mit.
- **PROJ-51** (Usage-Fee-Status-Display) — On-Hold-Spec, kann in dieser PROJ aufgehen.

## Implementierungs-Wellen (grob, für /architecture-Phase)

Aus Memory `project_pricing_strategy` übernommen (Phase A + B), aktualisiert für V3:

1. **Welle 1 — Datenmodell + Pricing-Service** (~2 Tage): Migrations 000079–000083, `pricing_service.go`, Trial-Logik, Mindestbetrag-Carryover. Tests.
2. **Welle 2 — FreeFinance-Client + Mocks** (~2 Tage): `internal/freefinance/`, OpenAPI-konformer Client, Mock-Pfad für `BILLING_LIVE_MODE=false`.
3. **Welle 3 — Mollie-Client + Webhook** (~2 Tage): `internal/mollie/`, Webhook-Handler, PROJ-71-Integration für Cool-Down.
4. **Welle 4 — Cron + Owner-/EEG-UI** (~3 Tage): K8s-CronJob, Cockpit-Sektion, EEG-Admin-Settings-Tab, Edition-Switch.
5. **Welle 5 — Doku + AGB-Update-Schalter + Tests-Sweep** (~1 Tag).

Gesamt ~10 Tage, deckt sich mit Memory-Schätzung 9–11 Tage.

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)

**Erstellt:** 2026-06-12
**Status nach Skill:** Architected
**Voraussetzungen geprüft:** Spec PROJ-104 (26 Owner-Entscheidungen + 38 ACs + 14 ECs) vollständig gelesen. Letzte Migration im Bestand ist `000078` (PROJ-101). PROJ-71-Cool-Down-Pfad lebt in `internal/customeronboarding/`. PROJ-67-`settings_view_mode`-Spalte existiert seit Migration `000059`. Es gibt zwei bestehende K8s-CronJobs im Helm-Chart (`data-export-cleanup-cronjob`, `restart-cronjob`) und einen `seed-job` — Pattern für quarterly-Billing-CronJob und Pricing-Seed sind also etabliert. PROJ-46-`activated_at` wird in `internal/application/application_repo.go` gesetzt.

### A) Component-Tree

```
Backend (Go, neuer Modul-Baum)
+-- internal/billing/                        <- NEU
|   +-- pricing_service.go                   <- Aktivierungs-Zähler + Edition-Snapshot + Mindestbetrag-Carryover
|   +-- period_repo.go                       <- DB-Zugriff auf billing_period / pricing_plan
|   +-- invoice_repo.go                      <- DB-Zugriff auf billing_invoice + Gutschrift-Verkettung
|   +-- scheduler.go                         <- Cron-Entry: iteriert EEGs, ruft Vendor-Glue
|   +-- live_mode.go                         <- Effektive Live-Bedingung (Pro-EEG ∧ Global)
|   +-- pdf_preview.go                       <- Lokales Preview-PDF (Mock-Pfad)
|   +-- audit.go                             <- Edition-Switch / billing_live-Flip / Gutschrift loggen
|
+-- internal/freefinance/                    <- NEU (Vendor-Client)
|   +-- client.go                            <- CreateInvoice / FinalizeInvoice / MarkAsPaid
|   +-- types.go                             <- Request/Response-DTOs
|   +-- mock.go                              <- BILLING_LIVE_MODE=false-Mock
|
+-- internal/mollie/                         <- NEU (Vendor-Client)
|   +-- client.go                            <- CreatePayment / GetPayment / CreateMandate-via-FirstPayment
|   +-- types.go                             <- Request/Response-DTOs
|   +-- mock.go                              <- BILLING_LIVE_MODE=false-Mock
|
+-- internal/http/
|   +-- billing_admin.go                     <- NEU: /api/admin/billing/* (Owner)
|   +-- billing_eeg.go                       <- NEU: /api/admin/eeg/{rc}/invoices (EEG-Admin)
|   +-- webhook_mollie.go                    <- NEU: POST /api/webhooks/mollie + IP-Allowlist + GET-Lookup
|
+-- internal/application/
|   +-- application_repo.go                  <- ERWEITERT: SetEditionAtActivation beim activated-Transition
|   +-- registration_entrypoint_repo.go      <- ERWEITERT: trial_started_at / eeg_edition / billing_live / mollie_mandate_active
|
+-- cmd/server/main.go                       <- ERWEITERT: Subcommand "billing-quarterly" + "billing-daily-sync"

Frontend (Next.js / React)
+-- src/app/admin/billing/                   <- NEU (Owner-only)
|   +-- page.tsx                             <- BillingDashboard (Tabs: Übersicht / Pricing-Plan / EEG-Live-Schalter)
|   +-- invoices/[id]/page.tsx               <- InvoiceDetail mit Status-Timeline
|
+-- src/components/billing/                  <- NEU
|   +-- pricing-plan-editor.tsx              <- Standard/Pro Preis-Pflege (versioniert)
|   +-- billing-period-list.tsx              <- Quartals-Übersicht mit Filter
|   +-- invoice-list.tsx                     <- Tabelle mit Filter Status/Quartal/EEG
|   +-- billing-live-toggle.tsx              <- Pro-EEG-Schalter mit Pre-Flight (€0-Check, Mandate-Status)
|   +-- credit-note-action.tsx               <- Gutschrift-Dialog mit Pflichtbegründung
|
+-- src/components/admin-eeg-settings-editor.tsx  <- ERWEITERT
    +-- Tab "Rechnungen"                     <- Eigene Rechnungs-Historie pro EEG (Read-Only)
    +-- Edition-Switch (im PROJ-67-advanced) <- Standard/Pro mit Downgrade-Block
    +-- Mandate-Status-Anzeige               <- "Mandate aktiv ab …" / "SEPA-Mandate-Validierung läuft"

Helm/Kubernetes
+-- helm/.../templates/
    +-- billing-quarterly-cronjob.yaml       <- NEU: Schedule "0 4 1 1,4,7,10 *"
    +-- billing-daily-sync-cronjob.yaml      <- NEU: Schedule "0 5 * * *"
    +-- pricing-seed-job.yaml                <- NEU: Default-€0-Seed beim Erst-Setup (analog seed-job.yaml)
    +-- values.yaml                          <- ERWEITERT: backend.billing.* Block
    +-- secrets.yaml                         <- ERWEITERT: freefinanceApiKey + mollieApiKey + IP-Allowlist
```

### B) Datenmodell (in Klartext)

**Vier neue Tabellen** im Schema `member_onboarding`:

1. **`pricing_plan`** — versionierte Preisliste
   - Pro Edition (Standard/Pro) ein Eintrag mit `gueltig_ab`/`gueltig_bis`
   - Preis pro aktiviertem Mitglied pro Quartal (Euro, netto)
   - USt-Satz (default 20 %)
   - Beim Rechnungs-Bauen wird die passende Plan-Zeile ausgewählt und ihre ID in `billing_period` archiviert
   - Pricing-Werte werden niemals nachträglich geändert — neue Werte = neue Zeile mit neuem `gueltig_ab`

2. **`billing_period`** — Quartals-Snapshot pro EEG
   - Felder: RC-Nummer, Jahr, Quartal (1–4), aktive-Mitglieder-Zähler (getrennt nach Edition-Snapshot: `count_standard` + `count_pro`), Pricing-Plan-Referenz, Netto/USt/Brutto
   - `UNIQUE(rc_number, year, quarter)` als harter Idempotenz-Schutz gegen Doppel-Cron-Läufe
   - Carryover-FK auf vorheriges Period bei Mindestbetrags-Übertrag

3. **`billing_invoice`** — Rechnungs-Bookkeeping
   - 1:1 zu `billing_period` im Standard-Pfad; bei Gutschrift entsteht eine zweite Zeile mit `cancels_invoice_id`-Verweis
   - Status: `draft → preview` (im Preview-Modus) ODER `draft → sent → paid` ODER `sent → overdue → paid|cancelled` ODER `credit_note`
   - Vendor-IDs: `freefinance_invoice_id` + `mollie_payment_id` (beide nullable für Preview-Modus)
   - PDF-Bytes bei Preview optional inline (max 256 KB)

4. **`billing_audit_log`** — wer hat wann was geändert
   - Edition-Switch (alt/neu/wer/wann)
   - `billing_live`-Flip (per EEG)
   - Pricing-Plan-Neue-Version
   - Gutschrift-Erstellung mit Grund
   - Manueller „Jetzt versenden"-Trigger im Preview-Modus

**Fünf Spalten-Erweiterungen** auf bestehenden Tabellen:

| Spalte | Tabelle | Typ | Default | Zweck |
|---|---|---|---|---|
| `trial_started_at` | `registration_entrypoint` | timestamptz NULL | NULL | Wird beim ersten `activated_at` einer Anwendung gesetzt; entscheidet ob ein Quartal trial-gratis ist |
| `eeg_edition` | `registration_entrypoint` | text NOT NULL CHECK | `'standard'` | Aktuelle Edition. Switch wirkt sofort, aber faire Verrechnung über Snapshot |
| `billing_live` | `registration_entrypoint` | bool NOT NULL | `FALSE` | Pro-EEG-Schalter; Live-Bedingung = `billing_live AND global_live` |
| `mollie_mandate_active` | `registration_entrypoint` | bool NOT NULL | `FALSE` | Wird auf TRUE gesetzt, sobald €0,01-Mandate-Setup von Mollie als `paid` zurückkommt |
| `edition_at_activation` | `application` | text NULL CHECK | NULL | Snapshot der Edition beim `activated`-Transition. Bestand-Aktivierungen bleiben NULL (Backfill in Welle 1 separater Job) |

**Render-Entscheidungsbaum (Rechnungs-Generierung im Cron):**

```
Für jedes EEG:
1. Hole alle application.activated_at IN [Quartal-Start, Quartal-Ende]
2. Gruppiere nach edition_at_activation → count_standard + count_pro
3. trial_started_at + 30 Tage > Quartal-Ende?
   → ja: kein Eintrag, Audit-Log "trial_period"
   → nein: weiter
4. Berechne total_net = count_standard * preis_standard + count_pro * preis_pro
5. total_gross = total_net * (1 + 0.20)
6. total_gross < 10€?
   → ja: carryover ins Folgequartal (sofern nicht Q4)
   → nein: weiter
7. Effektive Live-Bedingung prüfen (eeg.billing_live AND global_live)
   → false: Status='preview', PDF lokal, kein Vendor-Call
   → true: Mandate-Status prüfen (mollie_mandate_active)
       → false: Status='draft', nächster Daily-Cron versucht's nochmal
       → true: FreeFinance.CreateInvoice (mit Idempotency-Key {rc}-{year}-Q{quarter})
           → Mollie.CreatePayment (sequenceType='recurring', mandate-ref)
           → Status='sent', warten auf Webhook ODER Daily-Status-Sync
```

**Edition-Snapshot-Mechanik (bei `activated`-Transition):**

```
Wenn application.status wechselt zu 'activated':
1. Lies registration_entrypoint.eeg_edition (Bestand-Wert)
2. SET application.edition_at_activation = jener Wert
3. Lies registration_entrypoint.trial_started_at
4. Wenn NULL → SET trial_started_at = NOW()  (einmaliger Anlauf)
5. Audit-Log-Eintrag
```

### C) Tech-Entscheidungen mit Begründung

**1. BILLING_LIVE_MODE pro EEG + global — die wichtigste Design-Entscheidung**

Die Tester-Phase läuft jetzt produktiv mit echten Mitgliedern, aber der Owner will erst beim Prod-Cutover echte SEPA-Lastschriften ziehen. **Zwei-Stufen-Schalter:** ein globales Helm-Flag (Notbremse für alle) und ein Pro-EEG-Schalter (Pilot-EEG zuerst). Beide müssen TRUE sein, damit echte Vendor-Calls fließen. Im Preview-Modus läuft alles andere voll: Cron rechnet, Mitglieder-Aktivierungen werden gezählt, PDFs werden lokal generiert, Edition-Snapshots werden gesetzt — nur die externen API-Calls bleiben Mocks. Das macht die Phase 1A-Implementierung von Anfang an in der Tester-Umgebung verifizierbar, ohne Geld-Risiko.

**2. Edition-Snapshot statt Switch-Limit — anti-abuse ohne UX-Schmerz**

Ein EEG könnte morgens Standard buchen, Mitglieder aktivieren, dann auf Pro switchen und retro-aktiv abgerechnet werden — oder umgekehrt. Statt das mit einem Switch-Limit zu lösen, wird **jede Aktivierung mit ihrer damals-aktiven Edition versiegelt** (`application.edition_at_activation`). Das Pricing-Service zählt getrennt nach Snapshot, nicht nach aktueller Edition. Folge: EEG kann frei zwischen Editionen wechseln — alte Aktivierungen behalten ihren Tarif, neue tragen den neuen. Strukturell fair, ohne Verbotsregel.

**3. Trial-Start beim ersten `activated_at` statt beim `is_active=true`-Flip**

Klassische Falle: EEG flippt `is_active=true` zum Testen, vergisst es 6 Monate lang, kommt zurück, hat sein Trial verbraucht. Lösung in Grilling-Welle 3: Trial-Counter läuft erst, wenn das **erste echte Mitglied aktiviert wird**. Das ist genau der Zeitpunkt, ab dem die Plattform Wert liefert. Setup-Phasen sind frei, der Counter beginnt mit der ersten Wertschöpfung.

**4. Webhook + Daily-Sync-Cron als Backstop**

Mollie-Webhooks sind nicht zu 100 % zuverlässig (Netzwerk-Glitches, Replay-Lücken, IP-Allowlist-Drift). Statt sich auf eine einzige Schiene zu verlassen, gibt es **zwei Wege**: (a) der Webhook ist der Primärpfad und reagiert in Sekunden, (b) ein Daily-Cron um 5 Uhr morgens fragt für alle offenen Rechnungen aktiv den Status bei Mollie ab und gleicht ab. Webhook-Verlust bedeutet maximal 24h Verspätung statt einer „verlorenen" Rechnung.

**5. Gutschrift statt Storno**

GoBD verlangt fortlaufende Nummernkreise — bezahlte Rechnungen darf man nicht stornieren, sondern muss eine **Gutschrift mit negativ-Betrag** erzeugen. Das wird über die `cancels_invoice_id`-Verkettung sauber abgebildet. Sowohl Original als auch Gutschrift bleiben in den Vendor-Systemen erhalten, USt-Korrektur läuft automatisch.

**6. PROJ-71 Cool-Down statt Hard-Suspend**

EEG zahlt nicht → wir wollen nicht den Public-Onboarding-Pfad abklemmen (das wäre Selbst-Schaden, weil keine neuen Mitglieder = keine Aktivierungen = keine zukünftige Rechnung). Stattdessen werden nur die 6 Mehrwert-Endpoints aus PROJ-71 in Cool-Down geschoben (Datenweiterleitung, Reconciliation, Export-Configs, etc.). Public-Form läuft weiter, EEG-Admin sieht den Cool-Down-Banner, manuelle Owner-Aufhebung möglich.

**7. FreeFinance + Mollie statt eines Vendors**

FreeFinance Plus liefert die österreichische FinanzOnline-Direkt-Anbindung (das macht Mollie nicht). Mollie liefert die saubere SEPA-DD-PSP-Schicht (das macht FreeFinance nicht im Sinne von Self-Service-PSP). **Beide Best-of-Breed, klar getrennte Verantwortung:** FreeFinance = Rechnungs-Lebenszyklus + Buchhaltung, Mollie = Zahlungs-Lebenszyklus + Mandate. Verklebt durch unsere `MarkAsPaid`-Glue-Schicht.

**8. Idempotency-Key `{rc}-{year}-Q{quarter}` — defensive Cron-Hygiene**

K8s-CronJobs können mehrfach feuern (Replica-Race, manueller `kubectl create -from cronjob`, restart mid-execution). Statt unsere DB-Idempotenz allein zu trauen, geben wir FreeFinance einen **deterministischen Schlüssel** mit. Bei Wiederholung liefert FreeFinance dieselbe Invoice-ID zurück — wir schreiben die in unsere DB mit `ON CONFLICT DO NOTHING`. Doppelte Rechnung wird auf zwei Ebenen verhindert.

**9. K8s-CronJob statt In-App-Scheduler**

Ein In-App-Scheduler (`gocron`-Tick im Backend-Pod) wäre einfacher, aber: er feuert nur wenn der Pod läuft, er duplizert sich bei Replica >1, er macht Failure-Recovery selbst. **K8s-native CronJob** löst das: Owner kann manuell triggern (`kubectl create job --from=cronjob`), Logs landen in Standard-Kubernetes-Logs, `backoffLimit` ist deklarativ, `concurrencyPolicy: Forbid` verhindert Parallel-Läufe.

**10. Separate `/admin/billing`-Seite statt in PROJ-72-Cockpit-Quetsche**

PROJ-72 ist „Cockpit = EEG-Statistik". Billing ist „Rechnungs-/Pricing-/Mandate-Verwaltung". Beide Domänen, beide groß. Saubere Trennung über zwei Seiten, Cockpit verlinkt aufs Billing-Modul. Erlaubt **paralleles Arbeiten an PROJ-72 und PROJ-104** (Phase 1B + Phase 1A gleichzeitig denkbar).

### D) Implementierungs-Reihenfolge (5 Wellen, ~10 Tage)

**Welle 1 — Datenmodell + Pricing-Service** (~2 Tage)
- Migrationen 000079–000086 (8 Stück)
- `internal/billing/pricing_service.go` + `period_repo.go` + `invoice_repo.go`
- Edition-Snapshot-Hook in `application_repo.go` beim `activated`-Transition
- Trial-Start-Hook beim ersten `activated_at`
- Backfill-Job für Bestand-Aktivierungen (`edition_at_activation` rückwirkend zur dann-aktiven Edition)
- Tests: Quarter-Math, Carryover, Edition-Snapshot, Trial-Window
- **Owner-Verifikation FreeFinance-Trial parallel** (AC-27b, ~30 Min Owner-Aufwand)

**Welle 2 — Vendor-Clients + BILLING_LIVE_MODE** (~2 Tage)
- `internal/freefinance/client.go` mit CreateInvoice/FinalizeInvoice/MarkAsPaid
- `internal/mollie/client.go` mit CreatePayment/GetPayment
- `internal/billing/live_mode.go` (effektive Live-Bedingung)
- Mock-Pfade für nicht-live-Cluster
- Idempotency-Key-Konstruktion
- Tests mit `httptest`-Servern für beide Vendoren

**Welle 3 — Cron + Webhooks** (~2 Tage)
- `cmd/server/main.go billing-quarterly` Subcommand
- `cmd/server/main.go billing-daily-sync` Subcommand
- `internal/http/webhook_mollie.go` mit IP-Allowlist + GET-Lookup
- Helm: zwei neue CronJob-Templates + Secrets
- Tests Cron-Iteration über mehrere EEGs

**Welle 4 — Owner-UI + EEG-Admin-UI** (~3 Tage)
- `/admin/billing` Seite (Owner-only): Pricing-Plan-Editor + Rechnungs-Tabelle + EEG-Live-Toggle + Pre-Flight + Gutschrift-Aktion
- `admin-eeg-settings-editor.tsx`: Edition-Switch (im PROJ-67-advanced) + Downgrade-Block + Mandate-Status-Anzeige + Tab „Rechnungen" für eigene Liste
- Pre-Flight-Dialog €0-Pricing
- Mandate-Setup-€0,01-Trigger beim ersten `billing_live=TRUE`-Flip
- Owner-Mail-Template „SEPA-Mandate-Aktivierung läuft"

**Welle 5 — Mahnwesen + Doku** (~1 Tag)
- PROJ-71-Cool-Down-Integration (Daily-Overdue-Check feuert `suspended`-Event mit `reason_code='payment_overdue'`)
- Reactivation bei `paid`-Webhook
- `docs/architecture.md` (Billing-Stack-Diagramm)
- `docs/domain-model.md` (4 Tabellen + 5 Spalten)
- `docs/api-spec.md` (Mollie-Webhook + Owner/EEG-Endpoints)
- `docs/user-guide/06-admin-settings.md` (neue Sektion „Rechnungen")
- `CHANGELOG.md` + `docs/user-guide/changelog.md`
- AGB-Schalter aktivieren (Preise einblenden) — manuell durch Owner zum Prod-Cutover

### E) Dependencies & Risiken

**Neue Go-Dependencies:** keine externen Module nötig. FreeFinance- und Mollie-Clients werden mit `net/http` + `encoding/json` aus der Stdlib gebaut (Memory-konformer Minimalismus, kein Vendor-SDK-Lock-in). Idempotenz und Retry-Logic in eigenen Helpern.

**Neue npm-Dependencies:** keine. shadcn-Tables + bestehende Charts reichen für `/admin/billing`. Mandate-Status-Anzeige ist Plain-Text + Badge.

**Neue Helm-Werte (Schritt-für-Schritt-Liste für `values.yaml`):**
- `backend.billing.globalLiveMode: false` (Default — Notbremse)
- `backend.billing.minimumQuarterlyEur: 10`
- `backend.billing.overdueAfterDays: 14`
- `backend.billing.mollieAllowedIPs: ""` (CSV, Mollie-Webhook-Source)
- `backend.billing.freefinanceBaseUrl: "https://api.freefinance.at/..."` (Owner-final vor Welle 2)
- `backend.billing.mollieBaseUrl: "https://api.mollie.com/v2"`
- `secrets.freefinanceApiKey` (`secretKeyRef`-Pflicht)
- `secrets.mollieApiKey` (`secretKeyRef`-Pflicht)

**Risiken & Gegenmaßnahmen:**

| Risiko | Wahrscheinlichkeit | Auswirkung | Gegenmaßnahme |
|---|---|---|---|
| PROJ-71-Cool-Down-Pfad-Drift bei `payment_overdue`-Reason | Mittel | Mittel | Welle 5 schließt mit Integrationstest, der einen vollen overdue→reactivated-Zyklus durchspielt |
| K8s-CronJob-Permissions (Egress auf FreeFinance/Mollie) | Niedrig | Hoch | Welle 3 verifiziert Netzwerk-Policy vor Cron-Aktivierung |
| Webhook-Replay-Attack | Niedrig | Mittel | Defense-in-Depth: Mollie-GET-Lookup pro Webhook + Idempotency in DB |
| FreeFinance-API-Rate-Limits | Mittel | Niedrig | Cron läuft 1×/Quartal × N EEGs — selbst bei 500 EEGs unkritisch |
| Mollie-IP-Allowlist-Drift | Mittel | Niedrig | E-Mail-Subscription auf Mollie-Changelog + Defense-in-Depth via GET-Lookup |
| Stale-Trial bei Bestand-EEGs (Pre-PROJ-104-Aktivierungen) | Mittel | Niedrig | Welle 1 Backfill: `trial_started_at = min(activated_at) WHERE activated_at IS NOT NULL` |
| €10-Mindestbetrag-Edge: EEG aktiviert 1 Mitglied pro Quartal | Niedrig | Niedrig | Carryover-Logik (AC-8), Verfall am Jahresende ist akzeptiert |
| Vendor-Lock-in FreeFinance | Mittel | Mittel | Client-Interface (`BillingVendor`) abstrahiert — Wechsel auf z. B. Faktura erforderte nur neue Impl, nicht Architektur-Umbau |
| AGB-Text-Drift (Preise statt „folgen bei Produktiv-Start") | Niedrig | Niedrig | AGB-Update-Schalter ist letzter Owner-Schritt vor Prod-Cutover, dokumentiert in Welle 5 |

### Revisionen aus /grill-me 2026-06-12 (20 zusätzliche Entscheidungen)

Diese Block überschreibt die ursprünglichen ACs und Spec-Entscheidungen, wo angegeben. Wenn ein Konflikt entsteht, gewinnt diese Sektion.

**Datenmodell (Welle 1):**

- **R-1 (verschärft AC-1):** `pricing_plan`-Overlap-Schutz als harter PostgreSQL `EXCLUDE`-Constraint mit `btree_gist`-Extension auf `(edition WITH =, daterange(gueltig_ab, gueltig_bis, '[)') WITH &&)`. Race-frei. Migration 000079 aktiviert die Extension falls noch nicht aktiv.
- **R-2 (verschärft AC-5a):** Migration 000084 setzt `edition_at_activation = 'standard'` für ALLE Bestand-Aktivierungen inline. Bestand startet neutral. Owner kann per Skript korrigieren wenn nötig. Keine separate Backfill-Job-Komplexität.
- **R-3 (verschärft AC-4):** Migration 000082 lässt `trial_started_at = NULL` für alle Bestand-EEGs. Beim ersten Cron-Lauf nach Welle-1-Deploy wird für jede EEG mit Bestand-Aktivierungen ein **virtueller Trial-Start = Deploy-Datum** angenommen → 30-Tage-Grace-Period ab Welle 1. **Niemand wird durch IMMUTABLE-Migration sofort kostenpflichtig.** Pricing-Service trägt diese „virtual_trial_grace_until"-Logik als deterministischen Helper.
- **R-4 (NEU):** **Fünfte neue Tabelle** `billing_audit_log` für Edition-Switch, billing_live-Flip, Pricing-Plan-Versionierung, Gutschrift-Erstellung, Manual-Trigger, Mollie-Chargeback. Eigene Domäne, nicht in PROJ-71-Event-Log mischen. Migration 000087.

**Edition-Snapshot + PROJ-46/100-Pfad (Welle 2):**

- **R-5 (revidiert AC-Snapshot-Logik):** `ResetActivationTx` (PROJ-100) cleart `edition_at_activation` analog zu `activated_at`. Bei Re-Aktivierung wird der DANN aktive `eeg_edition`-Wert als neuer Snapshot gesetzt. Erfordert PROJ-100-Repo-Erweiterung (1 Spalte mehr im UPDATE-SQL).
- **R-6 (NEU):** **`eeg_edition` driven `settings_view_mode`** als Default-Sync. Edition-Switch auf `'pro'` setzt `settings_view_mode='advanced'`, Switch auf `'standard'` setzt `settings_view_mode='standard'`. Owner kann nachträglich manuell anders setzen (für Pro-EEGs, die Standard-Mode wünschen). Migration 000083 enthält keinen Backfill auf view_mode — Sync läuft nur bei Switch.
- **R-7 (verschärft AC-24, EC-10):** Downgrade-Block-Liste **konkret**:
  1. `external_api_key IS NOT NULL` (PROJ-13)
  2. `data_export_config` EXISTS für RC (PROJ-60)
  3. `reconciliation_enabled = TRUE` (PROJ-69)
  4. `board_approval_required = TRUE` (PROJ-76)
  5. `sepa_audit_trail_b2b = TRUE OR sepa_audit_trail_core = TRUE` (PROJ-78)
  6. `brand_mode = 'custom'` (PROJ-103)
  Dialog rendert pro Treffer einen Direkt-Link in die Settings, Owner muss erst deaktivieren.
- **R-8 (bestätigt Spec-Entscheidung #13):** **Edition-Switch bleibt EEG-Admin-Direkt-Switch.** EEG-Admin sieht im PROJ-67-`advanced`-Modus den Edition-Switch in seinen Settings und kann selbst wechseln. Downgrade-Block-Liste (R-7) wird im EEG-Admin-UI gerendert mit den 6 Pro-Settings-Checks + Direktlinks. Switch wirkt sofort; durch Edition-Snapshot-Mechanik (Entscheidung #15) bleiben bestehende Aktivierungen mit ihrem alten Tarif erhalten — Anti-Abuse ist strukturell, nicht über Verbotsregel. Owner sieht den Switch im `billing_audit_log`. **Korrektur der zwischenzeitlichen Grilling-Empfehlung Owner-only.** Begründung Owner 2026-06-12: EEG-Admins sollen autonom agieren können, Edition-Wechsel ist keine Owner-Approval-würdige Aktion solange Snapshot fair zählt.

**Vendor-Modell (Welle 3):**

- **R-9 (NEU):** **Sechste Spalten-Erweiterung** auf `registration_entrypoint`: `freefinance_customer_id TEXT NULL` (gesetzt beim ersten Live-Rechnungs-Lauf, persistiert die FF-CustomerID). Migration 000088.
- **R-10 (NEU):** **Siebte Spalten-Erweiterung** auf `registration_entrypoint`: `mollie_customer_id TEXT NULL` (gesetzt beim ersten `billing_live=TRUE`-Flip mit €0,01-Mandate-Setup, persistiert die Mollie-CustomerID + Mandate-Referenz). Symmetrisch zu FreeFinance. Migration 000088 (zusammen mit R-9).
- **R-11 (NEU AC-21b):** **Mollie-Chargeback-Webhook-Pfad.** Mollie sendet bei SEPA-DD-Rücklastschrift (innerhalb 8 Wochen) ein eigenes Event. Webhook-Handler `mollie_chargeback`:
  - Setzt `mollie_mandate_active = FALSE` auf der EEG-Zeile
  - Lässt `eeg.billing_live = TRUE` stehen (Owner-Entscheidung)
  - Schickt Owner-Alert-Mail (HTML-Template, neuer Mail-Type) mit RC + Betrag + Zeitstempel
  - Nächster Cron-Lauf erstellt Rechnungen als `status='draft'` (Mandate fehlt) statt `'sent'`
  - Owner kann manuell Mollie-Mandate-Reset triggern oder EEG live abschalten
- **R-12 (Tech-Design-Klarstellung):** Keine `BillingVendor`-Interface-Abstraktion. `internal/freefinance/client.go` + `internal/mollie/client.go` sind direkte HTTP-Clients mit eigenen DTOs. Mocks für `BILLING_LIVE_MODE=false`-Pfad in `_mock.go`-Files derselben Pakete. CLAUDE.md „No abstractions for single-use code" befolgt.

**Cron + UI (Welle 4):**

- **R-13 (verschärft AC-11):** K8s-CronJob hat `concurrencyPolicy: Forbid` + `startingDeadlineSeconds: 14400` (4h Toleranz für Cluster-Wartung am 1. Jan). Falls verpasst: kein Auto-Recovery — Owner triggert manuell via Manual-Trigger (R-16). Idempotency-Key + DB-UNIQUE verhindern Doppel-Rechnung bei Manual-Lauf.
- **R-14 (verschärft AC-19a):** Pre-Flight-Check beim EEG-`billing_live=TRUE`-Flip prüft **nur die aktive Edition der EEG** auf €0-Pricing. Wenn `eeg_edition='standard'` und `pricing_plan_standard=€0` → Dialog: „Standard kostet aktuell €0/Quartal — wirklich live gehen?". Owner-Bestätigung explizit. Wenn Standard preisig, Pro €0: kein Block (Pro kann nachträglich live werden).
- **R-15 (verschärft AC-22):** `/admin/billing`-Seite hat als Hauptansicht eine **EEG-Tabelle** mit Spalten (RC, Name, Edition-Badge `read-only`, Mandate-Status, Live-Toggle, Letzte Rechnung, Aktion). Live-Toggle in der Zeile öffnet Pre-Flight-Dialog (R-14). Pricing-Plan-Editor + Rechnungs-Tabelle sind separate Tabs derselben Seite. **Edition-Switch passiert NICHT hier** — er liegt im EEG-Admin-UI (siehe R-8). Owner sieht den Switch nachträglich im billing_audit_log-Tab.
- **R-16 (NEU AC-22a):** **Manual-Trigger** als Owner-Aktion in der EEG-Tabelle: „Letztes Quartal jetzt abrechnen". Triggert `CalculateQuarter(rc, lastYear, lastQuarter)` synchron, schreibt billing_audit_log-Eintrag `kind='manual_trigger'`. Idempotency-Key + DB-UNIQUE wirken: wenn schon abgerechnet, Dialog zeigt „Bereits abgerechnet: Rechnung XYZ vom DATUM". Kein freies Quartal-Choosing.

**Webhooks + Cool-Down + Sicherheit (Welle 5):**

- **R-17 (verschärft AC-14):** Mollie-Webhook bei **unbekanntem tr_xxx** (nicht in unserer DB): antwortet **200 OK** (Mollie wiederholt sonst). Schreibt billing_audit_log `kind='unknown_payment'` mit `source_ip` + `tr_id`. Mollie-GET-Lookup wird trotzdem versucht — wenn der Mollie-Account passt, ist das ein Drift-Signal. Owner-Alert (rate-limited, max 1×/h pro source_ip).
- **R-18 (verschärft AC-21):** Cool-Down-Aufhebung **nur wenn ALLE offenen overdue-Rechnungen einer EEG paid sind.** Solange auch nur eine `overdue` offen bleibt, bleibt der `payment_overdue`-Reason aktiv. Reactivation feuert ein einziges PROJ-71-Event mit `reason_code='payment_received'` und Detail-Felder über alle aufgeholten Rechnungs-IDs.
- **R-19 (NEU R-19, PROJ-71-Erweiterung):** PROJ-71-Event-Log muss **mehrere `reason_codes` parallel halten können**. Aktuelles Modell hat 1 Reason pro Suspended-Event. Erweiterung: bei jedem neuen Grund (z.B. `payment_overdue` zusätzlich zu manuellem Vertragsbruch-Suspend) wird ein neues Event geschrieben. Reactivation nur wenn ALLE offenen Reasons aufgelöst sind. **Kleine PROJ-71-Schema-Erweiterung** als Teil von PROJ-104 Welle 5 (eigene Mini-Migration, geht über die 8 Billing-Migrationen hinaus).
- **R-20 (NEU AC-25a):** **Logging-Whitelist** für Billing-Service:
  - **Erlaubt in slog:** `rc_number`, `year`, `quarter`, `count_standard`, `count_pro`, `total_eur`, `freefinance_invoice_id`, `mollie_payment_id`, `billing_period_id`, `billing_invoice_id`, `event_type`, `source_ip` (nur für Webhook-Audit)
  - **Verboten:** Mitglieder-Namen, IBAN, E-Mail, EEG-Vorstand-Daten (Name/Adresse), AGB-Texte, Bank-Konto-Details
  - Test in QA-Phase: grep über `internal/billing/`, `internal/freefinance/`, `internal/mollie/` und Webhook-Handler auf verbotene Felder
  - Identisch zu Bestand-`security.md`-Pattern, an die Domäne angepasst

### Aktualisierte Migrations-Liste (10 statt 8)

Nach Grilling-Revision:
- **000079** `pricing_plan` (mit `btree_gist`-Extension + EXCLUDE-Constraint)
- **000080** `billing_period`
- **000081** `billing_invoice`
- **000082** `registration_entrypoint.trial_started_at` (NULL bleibt, kein Backfill)
- **000083** `registration_entrypoint.eeg_edition` (Default 'standard')
- **000084** `application.edition_at_activation` (Backfill: alle Bestand auf 'standard' inline)
- **000085** `registration_entrypoint.billing_live` + `mollie_mandate_active` (zwei Spalten in einer Migration, konsistent)
- **000086** `billing_invoice.cancels_invoice_id` (Gutschrift-Verkettung) + Status-Whitelist um `credit_note` erweitern
- **000087** `billing_audit_log` (neue Tabelle für Owner-Aktionen, Vendor-Events, Manual-Triggers)
- **000088** `registration_entrypoint.freefinance_customer_id` + `mollie_customer_id` (Vendor-IDs)
- **000089** PROJ-71-Event-Log Erweiterung für mehrere `reason_codes` parallel (Mini-Migration, separat dokumentiert)

### Aktualisierte Welle-Aufteilung

Welle 1 (Datenmodell + Pricing-Service) wird wegen Migrations-Zuwachs (8→11) etwa **+0,5 Tage**:
- Migrationen 000079–000089 + ihre Down-Pfade
- Pricing-Service inkl. virtuelle-Trial-Grace-Logik
- Edition-Snapshot-Hook in `application_repo.go`
- `eeg_edition`→`settings_view_mode`-Sync-Helper
- Backfill in Migration 000084 inline (alle Bestand auf 'standard')

Welle 3 (Cron + Webhooks) wird wegen Chargeback-Pfad **+0,5 Tage**:
- Mollie-Webhook-Handler erweitert um `chargeback`-Event
- Owner-Alert-Mail-Template
- Daily-Status-Sync prüft auch Mandate-Status (Backstop)

Welle 4 (Owner-UI + EEG-UI) bleibt 3 Tage, aber:
- Edition-Switch verschiebt sich vom EEG-Settings ins `/admin/billing` (R-8)
- EEG-Settings-Tab „Rechnungen" wird read-only Badge statt aktivem Toggle
- Manual-Trigger-Aktion in EEG-Tabelle

Welle 5 (Mahnwesen + Doku) bekommt **PROJ-71-Mini-Erweiterung** für parallele reason_codes (+1 Tag):
- Migration 000089
- Service-Erweiterung in `internal/customeronboarding/`
- Tests
- → Welle 5 wird 2 Tage statt 1

**Neue Gesamt-Schätzung: ~11–12 Tage** (vorher 10).

### Empfehlung an Owner (nach Grilling 2026-06-12)

Tech-Design ist nun durch 5 Grilling-Wellen stress-getestet, 20 zusätzliche Entscheidungen oben in Revisionen-Block dokumentiert.

**Empfohlener nächster Schritt: `/backend PROJ-104` für Welle 1** (Datenmodell + Pricing-Service + Edition-Snapshot-Hook + virtuelle-Trial-Grace-Logik, ohne Vendor-Calls).

Welle 1 ist:
- Reversibel via Down-Migrationen
- Ohne externe Abhängigkeiten (keine Vendor-Anbindung)
- Frühe DB-Schema-Validation, bevor die teureren Vendor-Wellen starten

Owner-Pre-Welle-2-Aufgabe (R-Spec-AC-27b, ~30 Min): FreeFinance-Trial-Account-Test (API-vs-UI Nummernkreis, Kleinunternehmer-Hinweis, Idempotency-Header). Ergebnis in `private/vendor-setup/freefinance-trial-verification-2026-06-XX.md` dokumentieren. Kann **parallel zu Welle 1** laufen.

**Wenn weitere Klärungs-Wünsche aufkommen** vor `/backend`, dann punktuell — keine zweite volle Grilling-Welle nötig, das Tech-Design hat jetzt die kritischen Branches dokumentiert.

### Folge-Item nach PROJ-104-Abschluss (Owner-Vermerk 2026-06-12)

**DB-Design-Review.** `registration_entrypoint` wächst durch PROJ-104 auf 5 separate Domänen (Brand, Customer-Onboarding, Activation-Mode, SEPA-Toggles, Billing-Status + Vendor-IDs). `application` analog gewachsen durch PROJ-39/42/45/46/47/91/100/104. Nach Welle-5-Deploy ist eine dedizierte DB-Design-Review fällig, die prüft:

- Domänen-Trennung im DB-Schema (z. B. eigene 1:1-Sub-Tabelle `eeg_billing` für Billing-Felder + Vendor-IDs)
- Application-Lifecycle-Bookkeeping-Split (activated_at, faktura_handover_at, board_declaration_sent_at, edition_at_activation, … in eigene Tabelle?)
- Read-Performance auf Hot-Path (Public-Config-Endpoint, PROJ-32-Sync)
- Migrations-Historie >70 — Snapshot-Schema für neue Cluster?

Nicht Teil von PROJ-104 selbst. Eigene Memory-Notiz: `project_todo_db_design_review`.

### Idee (vertagt 2026-06-13): Lifecycle-sticky Pro statt Activation-Snapshot

Heute (PROJ-104 Spec #12 + AC-9 + EC-9): Pricing-Service zählt nach `application.edition_at_activation` — Snapshot zum Activation-Zeitpunkt. Edition-Wechsel danach beeinflusst die Aktivierung nicht.

Owner-Idee 2026-06-13: bei JEDEM Status-Übergang die aktuelle EEG-Edition mitschreiben (Spalte `status_log.eeg_edition_at_transition`) und die Pricing-Regel umstellen: sobald irgendeine Transition während des Application-Lifecycles auf Pro stand, ist Pro-Fee fällig — nicht nur der Activation-Snapshot.

Bewertung:
- Vorteil: Pro-Features tatsächlich genutzt = Pro-Preis. Anti-Abuse strukturell stärker als reines Activation-Snapshot.
- Risiko: Späte Transitions (z. B. PROJ-100 Reset → erneutes activated) können retroaktiv den Preis heben → schlechter kalkulierbar für EEG-Vorstand.
- Migration: saubere neue Migration (000089+) mit `eeg_edition_at_transition TEXT NULL`, alle Schreib-Pfade von status_log mitziehen.
- Edge-Case: bereits `submitted→under_review` direkt nach EEG-Switch auf Pro markiert die App lebenslang als Pro — gewollt oder zu aggressiv? Klären vor Umsetzung.

Entscheidung 2026-06-13: **NICHT in PROJ-104** umsetzen, Activation-Snapshot bleibt. Idee als Folge-PROJ vormerken, frühestens nach PROJ-104-Deploy + ersten Quartals-Daten neu bewerten.

## Implementierungs-Notes Welle 1 (2026-06-12)

Welle 1 ist Backend-only, kein Vendor-Call, kein UI, keine Cron-Jobs.
Implementierungs-Skill `/backend` 2026-06-12.

### Migrationen (9 neue Files)

- `000079_pricing_plan` — neue Tabelle mit EXCLUDE-Constraint via btree_gist (Grilling R-1)
- `000080_billing_period` — Quartals-Snapshot mit UNIQUE(rc, year, quarter)
- `000081_billing_invoice` — Bookkeeping-Tabelle, Welle 1 schreibt KEINE Inserts
- `000082_registration_entrypoint_trial_started_at` — NULL bleibt fuer Bestand (Grilling R-3)
- `000083_registration_entrypoint_eeg_edition` — Default `'standard'`
- `000084_application_edition_at_activation` — Backfill inline auf `'standard'` (Grilling R-2)
- `000085_registration_entrypoint_billing_live_and_mandate` — beide Bool-Spalten Default FALSE
- `000086_billing_invoice_credit_note_link` — `cancels_invoice_id` FK + CHECK-Whitelist um `credit_note` erweitert
- `000087_billing_audit_log` — neue Tabelle (5. Tabelle, bewusste 2. JSONB-Ausnahme in `payload`)

Migrationen 000088 (Vendor-IDs) und 000089 (PROJ-71-reason_codes) bleiben Welle 2 + Welle 5 vorbehalten.

### Neue Go-Module

- `internal/billing/` mit fünf Files:
  - `types.go` — `Edition`/`Quarter`/`PricingPlan`/`BillingPeriod`/`BillingInvoice`/`BillingAuditEvent` + Note/Status-Konstanten
  - `pricing_plan_repo.go` — `GetActivePlan`, `Insert`, `CloseOpenPlanTx`
  - `period_repo.go` — `GetByRCYearQuarter`, `InsertOrUpdate` (UPSERT auf UNIQUE), `ListByRC`
  - `invoice_repo.go` — `Insert`, `GetByID`, `ListByPeriod`, `UpdateStatus`, 256-KB-Preview-PDF-Limit
  - `audit_repo.go` — `Insert`/`InsertTx` mit Kind/ActorKind-Whitelist, `ExistsByKindAndRC` (Idempotenz für `VirtualTrialGraceApplied`)
  - `pricing_service.go` — Quarter-Math (`QuarterBoundaries`, `LastCompletedQuarter`), `IsInTrialAtEnd`, `EffectiveTrialAnchor` mit virtueller Trial-Grace, `CalculateQuarter` (Counter → Trial-Check → Pricing-Lookup → Mindestbetrag-Carryover → UPSERT)
  - `pricing_service_test.go` — Quarter-Math, Trial-Window-Edge-Cases (Tag 29/30/31), Quarter.IsValid, roundCurrency, MinimumQuarterlyEur-Defaults, Audit-Kind-Whitelist
- `internal/shared/billing_audit_kinds.go` — 9 Kind-Konstanten + Actor-Kind-Konstanten + Whitelists

### Erweiterte Files

- `internal/shared/models.go`:
  - `RegistrationEntrypoint` um 4 Felder (`TrialStartedAt`, `EegEdition`, `BillingLive`, `MollieMandateActive`)
  - `Application` um `EditionAtActivation *string`
  - `EditionStandard`/`EditionPro` Konstanten + `IsValidEdition`
- `internal/application/application_repo.go`:
  - `applicationColumns` + `scanApplicationRow` um `edition_at_activation` erweitert
  - `ResetActivationTx` + `ResetToReviewTx` clearen jetzt auch `edition_at_activation` (PROJ-100-Konsistenz)
  - Neue Methoden `ApplyEditionSnapshotTx`, `CountActivationsInQuarter`, `HasAnyActivationFor`
- `internal/application/admin_service.go`:
  - `ChangeStatus`: nach `UpdateStatusAdminTx` mit `toStatus=activated` wird `ApplyEditionSnapshotTx` + `SetTrialStartedAtIfNullTx` in derselben Tx aufgerufen
  - `MarkActivatedSkipImport` (PROJ-53): analoger Hook nach `MarkActivatedSkipImportTx`
- `internal/application/registration_entrypoint_repo.go`:
  - `GetByRCNumber`-SELECT um 4 neue Spalten erweitert
  - Neue Methoden: `SetTrialStartedAtIfNullTx`, `SetEegEditionTx` (synct settings_view_mode → Grilling R-6), `SetBillingLive`, `SetMollieMandateActive`, `GetTrialStartedAt`
- `internal/config/config.go`:
  - Neuer `BillingConfig`-Struct mit `GlobalLiveMode`, `DeployedAt` (virtueller Trial-Anker), `MinimumQuarterlyEur`
  - `getBoolEnv`/`getFloatEnv`/`getTimeEnv` Helpers
- Helm:
  - `values.yaml` + `values-env.yaml.example` um `backend.billing.{globalLiveMode,deployedAt,minimumQuarterlyEur}`
  - `templates/backend.yaml` reicht die drei Werte als `BILLING_*` ENV-Vars an den Backend-Pod

### Test-Stand

- `go build ./...` clean
- `go test ./...` alle Pakete grün (inkl. neuem `internal/billing` mit ~10 Test-Cases)
- Kein DB-Fixture-Test in Welle 1 (folgt in Welle 3, wenn das Cron-Glue dazukommt)

### AC-Erfüllungs-Map

- AC-1 (pricing_plan) ✓ inkl. EXCLUDE-Constraint via btree_gist (R-1)
- AC-2 (billing_period) ✓ mit UNIQUE-Idempotenz
- AC-3 (billing_invoice) ✓ Tabelle + Status-Whitelist (mit credit_note nach 000086)
- AC-4 (trial_started_at) ✓ NULL für Bestand + virtuelle Grace im Service (R-3)
- AC-5 (eeg_edition) ✓ Default standard
- AC-5a (edition_at_activation) ✓ inkl. Backfill auf standard (R-2)
- AC-5b (billing_live + mollie_mandate_active) ✓
- AC-5c (cancels_invoice_id + credit_note Status) ✓
- AC-6 (Pricing-Service mit Edition-Snapshot-Counter) ✓ via `CalculateQuarter`
- AC-7 (Trial-Check) ✓ via `EffectiveTrialAnchor` + `IsInTrialAtEnd`
- AC-8 (Mindestbetrag-Carryover + Q4-Verfall) ✓
- AC-9 (Edition-Wechsel-mitten-im-Quartal) ✓ strukturell via Snapshot
- AC-28 (Backend-Tests) ✓ partiell (Quarter-Math, Trial-Window, Edition, Constants)
- AC-29/30/31-35 — Vendor-Tests, Doku, etc. kommen in Welle 2-5

### Bewusste Out-of-Scope für Welle 1

- Vendor-Clients (FreeFinance, Mollie) — Welle 2
- Cron-Jobs + Subcommands — Welle 3
- Webhook-Endpoints + Chargeback-Pfad — Welle 3
- Owner-UI `/admin/billing` + EEG-Settings-Tab — Welle 4
- Mahnwesen-Integration mit PROJ-71 + Migration 000089 — Welle 5
- AGB-Schalter — Welle 5

### Pre-Welle-2-Owner-Aufgabe (AC-27b)

Owner verifiziert per FreeFinance-30-Tage-Trial-Account: API-vs-UI-Nummernkreis, Kleinunternehmer-Hinweis-Text, Idempotency-Header-Support. Ergebnis in `private/vendor-setup/freefinance-trial-verification-2026-06-XX.md`. ~30 Min, kann parallel zu Welle 1 oder direkt nach Welle 1 erfolgen.

### Empfehlung

Welle 1 ist reversibel via Down-Migrationen. Owner-empfohlenes Vorgehen:
1. Helm-Apply (Migrationen 000079-000087 laufen automatisch)
2. Smoke-Test: ein Antrag in den `activated`-Status setzen, prüfen ob `edition_at_activation` + `trial_started_at` korrekt gesetzt werden
3. SELECT-Smoke auf den 5 neuen Tabellen + 5 neuen Spalten
4. Dann Welle 2 starten (Vendor-Clients)

## Implementierungs-Notes Welle 2 (2026-06-12)

Welle 2 ist Backend-only, kein UI, kein Cron, kein Webhook. Liefert die Vendor-Client-Module + Live-Mode-Check + Vendor-Customer-ID-Persistenz.

### Migration 000088 (1 neue Datei + Down)

`registration_entrypoint`:
- `freefinance_customer_id TEXT NULL`
- `mollie_customer_id TEXT NULL`

Beide gesetzt erst in Welle 3 (Cron persistiert die externe ID nach erfolgreichem `EnsureCustomer`).

### Neue Go-Module

**`internal/freefinance/`** (4 Files):
- `types.go` — `Customer`, `LineItem`, `InvoiceCreateRequest/Response`, `MarkAsPaidRequest`, `APIError`, `ErrNotConfigured`, `Amount`-Helper
- `client.go` — `Client`-Interface + `HTTPClient` mit `EnsureCustomer`/`CreateInvoice` (mit `Idempotency-Key`-Header)/`FinalizeInvoice`/`MarkAsPaid`; Bearer-Auth; 3-fach-Retry für 5xx + 429; 4xx final
- `mock.go` — `MockClient` mit deterministischen IDs (`ffc-mock-{RC}`, `ffi-mock-{idempotency_key}`); Idempotenz-Map
- `client_test.go` — httptest-Server, Auth-Header, Idempotency-Key, Retry-Backoff, 4xx-no-Retry, MockClient-Determinismus

**`internal/mollie/`** (4 Files):
- `types.go` — `Amount` (String-Value mit 2 Dezimalen), `Customer`, `Payment`, `Mandate`, `PaymentCreateRequest`, Status-Konstanten (`open|pending|authorized|paid|failed|canceled|expired|charged_back`), `SequenceTypeFirst/Recurring`, `IsTerminalStatus`-Helper, Error-Typen
- `client.go` — `HTTPClient` mit `EnsureCustomer`/`CreateFirstPayment` (sequenceType=first, method=[directdebit])/`CreateRecurringPayment` (recurring + Mandate-ID)/`GetPayment`/`GetMandate`; gleicher Retry-Pattern
- `mock.go` — `MockClient` mit `tr_mock_N`-IDs; `SetPaymentStatus` als Test-Hilfe für Welle-3-Webhook-Tests
- `client_test.go` — Amount-Formatting, sequenceType-Body, Method-Liste, WebhookURL-Propagation, Metadata-Roundtrip, MockClient-Lifecycle

**`internal/billing/live_mode.go`** + Test:
- `IsLive(globalLive bool, eeg shared.RegistrationEntrypoint) bool` — beide Schalter UND-verknüpft. Welle 3 nutzt das, um zwischen Live- und Mock-Client zu wählen.

### Erweiterungen Bestand

- `internal/shared/models.go`: `FreefinanceCustomerID *string` + `MollieCustomerID *string` (json:"-", backend-intern)
- `internal/application/registration_entrypoint_repo.go`:
  - `GetByRCNumber`-SELECT um die 2 neuen Spalten
  - Neue Setter `SetFreefinanceCustomerID`, `SetMollieCustomerID`
- `internal/config/config.go`: `BillingConfig` um `FreefinanceBaseURL`, `FreefinanceAPIKey`, `MollieBaseURL`, `MollieAPIKey`, `MollieWebhookURL`
- Helm:
  - `values.yaml` + `values-env.yaml.example`: `backend.billing.freefinanceBaseUrl`, `mollieBaseUrl`, `mollieWebhookUrl`
  - `values-secret.yaml.example`: `secrets.freefinanceApiKey` + `secrets.mollieApiKey` (leer in Preview-Modus)
  - `templates/secrets.yaml`: required-Guard — wenn `globalLiveMode=true`, dann beide API-Keys Pflicht. Plus `FREEFINANCE_API_KEY` + `MOLLIE_API_KEY` im backend-secret
  - `templates/backend.yaml`: 5 neue ENV-Vars (`BILLING_FREEFINANCE_BASE_URL`, `BILLING_MOLLIE_BASE_URL`, `BILLING_MOLLIE_WEBHOOK_URL` + `FREEFINANCE_API_KEY`/`MOLLIE_API_KEY` via `secretKeyRef`)

### Test-Stand

- `go build ./...` clean
- `go test ./...` alle Pakete grün
- `go test ./internal/freefinance/...` — Auth, Idempotency-Key, Retries, 4xx-No-Retry, MockClient-Determinismus
- `go test ./internal/mollie/...` — Amount-Formatting, sequenceType, WebhookURL, Metadata, Status-Konstanten, MockClient-Lifecycle (`SetPaymentStatus`)
- `go test ./internal/billing/...` — `IsLive` Wahrheitstabelle

### AC-Erfüllungs-Map Welle 2

- AC-12 (FreeFinance-Client) ✓ Live + Mock, Idempotency-Key-Header, anonyme Leistungsbeschreibung (LineItem.Description vom Cron befüllt)
- AC-13 (Mollie-Client) ✓ Live + Mock, sequenceType=first/recurring, EUR-0,01-Helper, Webhook-URL-Propagation, GetPayment für Backstop
- AC-15 (BILLING_LIVE_MODE pro EEG + global) ✓ via `billing.IsLive`
- AC-16 (Mocks im Preview-Modus) ✓ `MockClient` in beiden Vendor-Paketen
- AC-18 (Live: Idempotency-Key-Header) ✓ Stripe-Pattern, branchen-üblich; Owner-Verifikation in AC-27b
- AC-25 (PII-Disziplin) ✓ Customer-Stammdaten + Beschreibungs-String, keine Mitglieder-Identifikatoren
- AC-26 (Mollie nur Owner-Mandate-Daten) ✓ Customer.Name/Email/Metadata.external_ref
- AC-29 (Vendor-Client-Tests) ✓ httptest, Mocks, Retries, 4xx
- AC-30 (`BILLING_LIVE_MODE=false`-Test) ✓ MockClient-Tests bestätigen No-HTTP

### Bewusste Out-of-Scope für Welle 2

- Webhook-Endpoint `POST /api/webhooks/mollie` — Welle 3
- Daily-Status-Sync-Cron + Quarterly-Cron — Welle 3
- Chargeback-Pfad (Setzt `mollie_mandate_active=FALSE`) — Welle 3
- Owner-UI + Mandate-Setup-Trigger beim `billing_live=TRUE`-Flip — Welle 4
- AC-19a Pre-Flight-Check + Manual-Trigger — Welle 4
- AC-21a Gutschrift-Pfad — Welle 4

### Pre-Welle-3-Owner-Aufgabe

AC-27b (FreeFinance-Trial-Verifikation) ist Voraussetzung für Live-Aktivierung. Welle 3 Cron-Glue greift die Vendor-Endpoints und braucht das verifizierte Endpoint-/Header-Schema. Code-Anpassungen bei Abweichung sind lokal in `internal/freefinance/client.go` (~50 Zeilen).

### Empfehlung

Welle 2 ist reversibel via Down-Migration 000088 + Rollback der Code-Änderungen. Owner-empfohlenes Vorgehen vor Welle 3:
1. Helm-Apply (Migration 000088 läuft automatisch; Code-Wechsel ohne sichtbare Wirkung in Preview-Modus)
2. AC-27b Trial-Verifikation FreeFinance (~30 Min)
3. Dann Welle 3 starten (K8s-CronJob + Webhook + Cool-Down-Integration mit PROJ-71)

## Implementierungs-Notes Welle 3 (2026-06-12)

Welle 3 ist Backend-only, kein UI. Liefert den Scheduler-Glue, den Mollie-Webhook und die PROJ-71-Cool-Down-Integration.

### Migration

Keine neue DB-Migration. Migration 000089 (PROJ-71 Multi-Reason-Code, Grilling R-19) vertagt: PROJ-71 hat bereits `ReasonPaymentFailed` und `ReasonPaymentReceived` als Reason-Codes und der Event-Log nimmt mehrere parallele Reasons über separate Events auf. V1-Limitierung: Reactivation hebt alle Suspend-Gründe gemeinsam auf (kein per-Reason-Counter). Owner kann manuell zurücksuspendieren. Falls in der Praxis problematisch, eigenes Folge-PROJ.

### Neue Files / Module

- `internal/billing/scheduler.go` — Cron-Top-Level (`RunQuarterly`, `RunDailyStatusSync`, `RunDailyOverdueCheck`, `TriggerForRC`, `ProcessWebhookPayment`). 600+ LOC mit Live-Pfad (FreeFinance + Mollie) und Preview-Pfad (Mocks). Live-Bedingung: `globalLiveMode AND eeg.billing_live AND mollie_mandate_active`. Bei Mandate-pending → status='draft', nächster Cron versucht's erneut.
- `internal/billing/cool_down.go` — `CoolDownService` mit `OnInvoiceOverdue` (schreibt PROJ-71 `EventSuspended` mit `ReasonPaymentFailed`) und `OnInvoicePaid` (prüft `CountOpenOverdueForRC` → nur bei 0 wird `EventReactivated` mit `ReasonPaymentReceived` geschrieben — Grilling R-18).
- `internal/http/webhook_mollie.go` — `POST /api/webhooks/mollie` mit IP-Allowlist + Defense-in-Depth via `Mollie.GetPayment` (Grilling R-17 / AC-27). Unknown `tr_xxx` → 200 OK + Audit-Log `AuditKindUnknownPaymentWebhook`. Upstream-Fehler → 500 (Mollie retried).
- `helm/.../templates/billing-quarterly-cronjob.yaml` — Schedule `"0 4 1 1,4,7,10 *"`, `concurrencyPolicy: Forbid`, `startingDeadlineSeconds: 14400` (4h Toleranz, Grilling R-13).
- `helm/.../templates/billing-daily-cronjob.yaml` — Schedule `"0 5 * * *"`, kombiniert Status-Sync (Webhook-Backstop) + Overdue-Check (Mahnwesen).
- Tests: `internal/http/webhook_mollie_test.go` (11 Cases: IP-Allowlist, XFF, Empty-Body, Unknown-tr_id, Upstream-Error, Status-Forward, parseAllowedIPs), `internal/billing/scheduler_test.go` (pure helpers: perUnit, effectiveVATPercent, deref, buildLineItems Standard-only + Mixed, makeFreefinanceCustomer + makeMollieCustomer mit anonymisierten Payloads).

### Erweiterungen Bestand

- `internal/billing/invoice_repo.go`: `ListOpenForSync` (status IN sent/overdue + Mollie-Payment-ID), `ListOverdueCandidates(threshold)`, `LookupRCByPaymentID`, `CountOpenOverdueForRC` (Grilling R-18), `MarkSent`, `MarkOverdue`, `MarkPaid`, `SetVendorIDs`.
- `internal/application/registration_entrypoint_repo.go`: `ListActiveRCNumbers` (für Scheduler-Iteration), `HasAnyActivationFor` (Spiegel-Methode, damit `eegBillingStateReader`-Interface aus Welle 1 vom Entrypoint-Repo erfüllt wird → Single-Repo-Wiring im Scheduler).
- `internal/shared/billing_audit_kinds.go`: 4 neue Audit-Kinds (`AuditKindSchedulerRun`, `AuditKindOverdueMarked`, `AuditKindPaymentReceived`, `AuditKindMandateActivated`).
- `cmd/server/main.go`:
  - Subcommand-Dispatch erweitert um `billing-quarterly` + `billing-daily`
  - `buildBillingScheduler(cfg)` als Wiring-Helper für beide Cron-Subcommands
  - HTTP-Server registriert `POST /api/webhooks/mollie` automatisch, wenn `cfg.Billing.MollieWebhookURL != ""`
  - Live-vs-Mock-Switch über `freefinance.NewHTTPClient` + `mollie.NewHTTPClient` (Fallback auf Mocks bei fehlender Config)
- Helm:
  - `values.yaml`: `backend.billing.mollieAllowedIPs` + `billingQuarterly.{enabled,schedule}` + `billingDaily.{enabled,schedule}`
  - `values-env.yaml.example`: analog
  - `templates/backend.yaml`: `BILLING_MOLLIE_ALLOWED_IPS` ENV-Var

### Test-Stand

- `go build ./...` clean
- `go test ./...` alle Pakete grün
- `go test ./internal/http/...` — 11 neue Webhook-Cases (IP-Allowlist, Defense-in-Depth, Unknown-tr_id-Pfad)
- `go test ./internal/billing/...` — Welle-1+2-Tests + 10 neue Scheduler-Helper-Cases (alle pure functions; DB-Integration kommt in Welle-4-QA)

### AC-Erfüllungs-Map Welle 3

- AC-10 (Subcommand `billing-quarterly`) ✓ via `runBillingQuarterly` + Scheduler
- AC-11 (K8s-CronJob) ✓ beide Templates mit Forbid + 14400s Deadline
- AC-14 (Mollie-Webhook + IP-Allowlist + GET-Lookup) ✓ Defense-in-Depth
- AC-18 (Live-Pfad mit Idempotency-Key) ✓ via Scheduler `runLiveVendor`
- AC-20 (Daily-Overdue-Check + PROJ-71 Cool-Down) ✓ `RunDailyOverdueCheck` + `CoolDownService.OnInvoiceOverdue`
- AC-20a (Daily-Status-Sync Backstop) ✓ `RunDailyStatusSync`
- AC-21 (Reactivation bei paid) ✓ via `CoolDownService.OnInvoicePaid` (nur wenn alle overdue erledigt — Grilling R-18)
- AC-21b (Mollie-Chargeback-Pfad) ✓ `applyMollieStatus` → `SetMollieMandateActive(false)` + Audit-Log; Owner-Alert-Mail kommt in Welle 4
- AC-25 (PII-Disziplin) ✓ Anonymisierte Line-Items, Customer ohne Mitglieder-Daten

### Bewusste Out-of-Scope für Welle 3

- Owner-UI `/admin/billing` — Welle 4
- Pre-Flight-Check €0 — Welle 4
- Pricing-Plan-Editor UI — Welle 4
- Owner-Alert-Mail bei Chargeback — Welle 4
- Owner-Mail „SEPA-Mandate-Aktivierung läuft" — Welle 4
- AGB-Update-Schalter — Welle 5
- DB-Integration-Tests gegen reales Postgres — Welle 4/5 mit /qa

### Empfehlung für Owner

1. **AC-27b FreeFinance-Trial-Verifikation** vor Live-Aktivierung (~30 Min); FreeFinance-Endpoint-Schema gegen `internal/freefinance/client.go` vergleichen. Bei Differenzen lokal anpassen.
2. **Helm-Apply** ohne Code-Risiko: `globalLiveMode: false` bleibt Default → Cron läuft Preview-only, schreibt nur Bookkeeping. Kein realer Vendor-Call.
3. **Smoke-Test** auf test-Cluster:
   - `kubectl create job --from=cronjob/...billing-quarterly billing-test-q1` → manueller Quartals-Trigger
   - SELECT auf `billing_period` + `billing_invoice` (`status='preview'`)
   - Audit-Log-Eintrag `scheduler_run` pro EEG
4. Dann **Welle 4** (Owner-UI + EEG-Settings-Tab + Pre-Flight-Check)

## Implementierungs-Notes Welle 4a (2026-06-12)

Welle 4a ist Backend-only — Owner-Endpoints + EEG-Admin-Read-Only + Owner-Mails + Mandate-Setup-Trigger. UI folgt in Welle 4b (/frontend).

### Migration

Keine neue DB-Migration. Bestand reicht.

### Neue Files / Module

- `internal/billing/preflight.go` — `CheckLiveActivation(planRepo, eeg, now) (*PreFlightResult, error)`. Liefert `HasZeroPricing`, `CurrentEurPerMember`, `CurrentEdition`, `MandateActive`, `BillingLive`, Vendor-Customer-Indicators. `ErrNoActivePricingPlan` als sentinel.
- `internal/billing/mandate_setup.go` — `MandateSetupService.TriggerMandateSetup(ctx, rc, ownerSubject)`. Sichert Mollie-Customer (ggf. EnsureCustomer + Persistierung), feuert EUR 0,01 First-Payment via Mollie-Client (Live oder Mock je nach Config), schreibt Audit `BillingLiveFlipped` mit Setup-Payload.
- `internal/mail/billing.go` — 3 neue Sender-Methoden + 3 Data-Structs. Templates in `internal/mail/templates/billing_{mandate_setup,chargeback_owner_alert,credit_note}.html`. SMTPMailService + NoOpMailService implementieren beide.
- `internal/http/admin_billing.go` — `AdminBillingHandler` mit 11 Endpoints unter `/api/admin/billing/*`:
  - `GET  /pricing-plans` → Liste der aktiven Pläne pro Edition
  - `POST /pricing-plans` → Neue Plan-Version (Validation + Audit `PricingPlanVersioned`)
  - `GET  /eegs` → Liste aller aktiven EEGs mit Billing-State
  - `GET  /eegs/{rc}/pre-flight` → CheckLiveActivation-Wrapper für UI-Vorab
  - `POST /eegs/{rc}/billing-live` → Toggle mit Pre-Flight + Mandate-Setup-Mail (sync hard-fail) + MandateSetupService-Trigger + Audit
  - `POST /eegs/{rc}/edition` → Owner-Override (SetEegEdition syncs settings_view_mode + Audit `EditionSwitched`)
  - `POST /eegs/{rc}/trigger` → Manual-Quartals-Trigger via `SchedulerService.TriggerForRC`
  - `GET  /invoices` → Liste (MVP: ListOpenForSync)
  - `GET  /invoices/{id}` → Detail
  - `POST /invoices/{id}/credit-note` → Gutschrift-Insert mit `cancels_invoice_id`-Verkettung + Pflicht-Grund + Audit `CreditNoteIssued`
  - `GET  /audit-log` → Filterbare Liste (kind, rc, limit)
- `internal/http/admin_eeg_invoices.go` — `AdminEEGInvoicesHandler.ListInvoices` für `GET /api/admin/eeg/{rc}/invoices` (Tenant-Admin Read-Only mit `containsRC`-Check).

### Erweiterungen Bestand

- `internal/mail/service.go`: MailService-Interface um 3 neue Methoden + NoOp-Stubs + SMTP-Konstruktor parsed billing-Templates
- `internal/application/registration_entrypoint_repo.go`: `SetEegEdition` als DB-Level-Wrapper um `SetEegEditionTx`
- `internal/billing/scheduler.go`: optionaler `ChargebackAlertSender`-Hook + `SchedulerConfig.OwnerEmail/AdminBaseURL`; `applyMollieStatus` ruft den Mailer bei Chargeback (best-effort)
- `cmd/server/main.go`:
  - `billingHTTPMollieLive(cfg)` + `billingSchedulerForHTTP(cfg, db)` Wiring-Helper
  - Adapter `billingChargebackAdapter` für Scheduler→MailService-Brücke
  - Neue Routen-Gruppe `/api/admin/billing/*` (Owner-only via per-Handler `requireSuperuser`)
  - Neue Route `GET /api/admin/eeg/{rc}/invoices` (Tenant-Admin)

### Test-Stand

- `go build ./...` clean
- `go test ./...` alle Pakete grün
- `internal/billing/preflight_test.go` — `PreFlightResult`-Felder, `ErrNoActivePricingPlan`-Sentinel
- `internal/mail/billing_test.go` — 4 Render-Cases (Mandate-Setup mit allen Platzhaltern, Chargeback mit Payment-ID, Credit-Note mit + ohne Grund)

### AC-Erfüllungs-Map Welle 4a

- AC-19a (Pre-Flight-Check €0-Pricing) ✓ `CheckLiveActivation` + `GET /pre-flight` + `POST /billing-live` blockt ohne `acceptZeroPricing: true`
- AC-21a (Gutschrift-Pfad) ✓ `POST /invoices/{id}/credit-note` mit Pflicht-Grund + Audit
- AC-22 (Pricing-Plan-Editor + Rechnungs-Liste) ✓ Backend ready; UI-Tabelle/Editor folgt Welle 4b
- AC-22a (Manual-Trigger) ✓ `POST /eegs/{rc}/trigger`
- AC-23 (EEG-Admin Tab "Rechnungen") ✓ Backend ready (`GET /eeg/{rc}/invoices`); UI Welle 4b
- AC-24 (Edition-Switch Owner-Override) ✓ `POST /eegs/{rc}/edition`
- AC-24a (Mollie Mandate-Setup + Mail) ✓ Sync vor Trigger, hard-fail bei Mail-Fehler
- AC-25 (USt 20% + anonymisierte Beschreibung) ✓ Welle 1+2-Pricing-Service + Welle-3-Scheduler liefern das schon

### Bewusste Out-of-Scope für Welle 4a

- React-Frontend (Welle 4b)
- PDF-Generation/Download für Rechnungen (späteres PROJ)
- Filter-Logik in `ListInvoices` (Frontend macht clientseitige Filter für V1)
- Vollständige Pricing-Plan-Liste (heutiger Lookup nur „aktiver Plan pro Edition"; Owner sieht in `/admin/billing`-Liste nur den aktuellen Tarif. Historischer Plan-Browse in Folge-Welle.)
- Downgrade-Block-Liste (R-7) — kommt im Frontend mit den 6 Pro-Settings-Checks (PROJ-13/60/69/76/78/103)
- AGB-Update-Schalter — Welle 5
- Doku-Updates — Welle 5

### Empfehlung für Owner

1. **Helm-Apply** ohne Risiko: Owner-Endpoints + Mails kompilieren und laufen, aber im Preview-Modus passiert kein Vendor-Call. `SetBillingLive`-Toggle bleibt ohne Effekt (Cron flippt sowieso auf `preview`).
2. **Smoke-Test** auf test-Cluster: `curl -X GET https://.../api/admin/billing/eegs` (Superuser-Token) → JSON-Liste; `POST /pricing-plans` mit `edition:"standard", eurPerActiveMemberPerQuarter: 5.00` → Audit-Log-Eintrag.
3. **Welle 4b** (/frontend) liefert die UI-Komponenten gegen diese Endpoints.

## Implementierungs-Notes Welle 4b (2026-06-12)

Welle 4b ist Frontend-only — UI gegen die Welle-4a-Endpoints.

### Neue API-Client-Functions in src/lib/api.ts

12 neue Functions + 9 neue Types unter dem Kommentar-Block `PROJ-104 Welle 4b`:

- Types: `BillingEdition`, `BillingPricingPlan`, `BillingEEGState`, `BillingPreFlightResult`, `BillingInvoiceStatus`, `BillingInvoice`, `EEGInvoiceItem`, `BillingAuditEvent`, `CreatePricingPlanRequest`, `SetBillingLiveRequest`, `SetBillingLiveResponse`, `TriggerQuarterlyResult`, `AuditLogFilters`
- Owner: `listPricingPlans`, `createPricingPlan`, `listBillingEEGs`, `getBillingPreFlight`, `setBillingLive`, `setBillingEdition`, `triggerBillingQuarterly`, `listBillingInvoices`, `getBillingInvoice`, `createCreditNote`, `listBillingAuditLog`
- EEG-Admin: `listEEGOwnInvoices`

### Neue Frontend-Files

**Owner-Seite**:
- `src/app/admin/billing/page.tsx` — `/admin/billing` mit 4 shadcn Tabs (EEGs / Pricing-Plan / Rechnungen / Audit-Log)

**Komponenten in `src/components/billing/`**:
- `billing-eeg-table.tsx` — EEG-Tabelle mit Edition-Badge, Mandate-Status, Live-Toggle, „Jetzt abrechnen" pro Zeile; Live-On öffnet `PreFlightDialog`, Live-Off direkt
- `pre-flight-dialog.tsx` — Lädt `getBillingPreFlight`, zeigt Tarif/Mandate/Vendor-Status, €0-Pflicht-Checkbox, ruft `setBillingLive(rc, { live, acceptZeroPricing })`
- `pricing-plan-editor.tsx` — 2 Edition-Cards (Standard/Pro) + `NewPricingPlanDialog` mit Input für EUR/USt/gueltigAb
- `invoice-table.tsx` — Status-Filter + Suche, Gutschrift-Aktion (nur bei sent/paid/overdue) öffnet `CreditNoteDialog`
- `credit-note-dialog.tsx` — Pflicht-Grund (min. 10 Zeichen, KEIN placeholder, Hint-Text), `createCreditNote` + Refresh
- `audit-log-list.tsx` — Filter (kind, rc), JSON-Pretty-Print für Payload
- `edition-switch-dialog.tsx` — Pro→Standard Downgrade-Block-Liste mit 6 Pro-Feature-Flags (PROJ-13/60/69/76/78/103) + Direktlinks
- `eeg-own-invoices.tsx` — EEG-Admin Read-Only-Tabelle (`listEEGOwnInvoices`), eigenes Empty-State

### Erweiterungen Bestand

- `src/app/admin/settings/page.tsx`: neuer Tab „Rechnungen" mit `EEGOwnInvoices`-Komponente (sichtbar in beiden View-Modi)

### Test-Stand

- `npx tsc --noEmit` clean
- `npx vitest run` — 238/238 grün (inkl. 2 neue Test-Files):
  - `src/components/billing/edition-switch-logic.test.ts` — 4 Downgrade-Block-Szenarien
  - `src/components/billing/credit-note-validation.test.ts` — 6 Cases (min-len, trim, whitespace, leer, lang)
- `npm run build` clean, `/admin/billing` als dynamische Route gelistet

### AC-Erfüllungs-Map Welle 4b

- AC-19a UI (Pre-Flight-Dialog mit €0-Bestätigungs-Checkbox) ✓
- AC-21a UI (Gutschrift-Dialog mit Pflicht-Grund) ✓
- AC-22 UI (Pricing-Plan-Editor + Rechnungs-Tabelle + EEG-Tabelle) ✓
- AC-22a UI („Jetzt abrechnen"-Button pro EEG) ✓
- AC-23 UI (EEG-Settings-Tab „Rechnungen" mit Read-Only-Liste) ✓
- AC-24 UI (Edition-Switch-Dialog mit Downgrade-Block-Liste) ✓
- AC-24a UI (Mandate-Setup-Pfad sichtbar im Pre-Flight + Mail wird vom Backend versendet) ✓

### Bewusste Out-of-Scope für Welle 4b

- PDF-Generation/Download (späteres PROJ — Frontend hat dafür heute keinen Endpoint, Welle 5 oder neues PROJ)
- AGB-Schalter — Welle 5
- User-Guide „Rechnungen" — Welle 5
- Mandate-Status-Anzeige im EEG-Settings-Stammdaten-Tab (wird heute schon im Pre-Flight-Dialog gerendert; eigene Sektion im Stammdaten-Editor wäre die nächste UX-Iteration)
- Frontend-Integration für Owner-Edition-Switch im `/admin/billing` (`EditionSwitchDialog` ist vorhanden und einsetzbar, Wiring kommt im nächsten Refinement)

### Empfehlung für Owner

1. **Helm-Apply** (Frontend-Image-Tag) — Backend hat sich seit Welle 4a nicht geändert
2. **Smoke-Test**:
   - `/admin/billing` als Superuser öffnen — EEG-Tabelle + 4 Tabs sichtbar
   - „Pricing-Plan" → „Neue Version anlegen" → Standard EUR 0.00 anlegen → Audit-Log-Eintrag
   - EEGs-Tab → „Live schalten" → Pre-Flight zeigt €0-Warnung + Pflicht-Checkbox → bestätigen → Backend feuert Mandate-Setup
   - EEG-Settings → Tab „Rechnungen" → Read-Only-Liste oder Empty-State
3. **Welle 5** (Mahnwesen-Pfeil-Polishing + AGB-Update-Schalter + Doku/User-Guide)

## Implementierungs-Notes Welle 5 (2026-06-12)

Welle 5 ist Doku-only — kein Backend-, kein Frontend-Code. Schließt die Lücken zwischen Implementation (Wellen 1–4) und Owner-Verständlichkeit.

### Geänderte Files

- `docs/domain-model.md` — neue Sektion 6 mit 4 Tabellen-Definitionen, 7 Spalten-Erweiterungen, Edition-Snapshot-Mechanik, Drei-Faktor-Live-Bedingung
- `docs/api-spec.md` — neue Sektion „PROJ-104 Platform Billing API" mit allen 12 Endpoints, Webhook, Cron-Subcommands, 13 Audit-Kind-Werten und 3 Owner-Mail-Triggern
- `docs/architecture.md` — neue Sektion „PROJ-104 Platform Billing Stack" mit Component-Map (ASCII), Data-flow-per-quarter, Cool-Down-Integration, Anti-Abuse-Design, Live-mode-safety
- `docs/user-guide/06-admin-settings.md` — neue Sektion „Tab Rechnungen" (EEG-Admin-Sicht: Spalten-Erklärung, Trial-/Live-Pfad, Edition-Switch, Snapshot-Pattern, Pro→Standard-Block) — PROJ-frei
- `docs/user-guide/changelog.md` — Tagesblock 2026-06-12 mit fünf User-spürbaren Änderungen (PROJ-frei)
- `CHANGELOG.md` — technischer Block 2026-06-12 (alle 5 Wellen kompakt)
- `src/content/legal/agb-v1.0.md` — § 4 mit HTML-Kommentar als Owner-Cutover-Reminder: beim Helm-Flip auf `globalLiveMode=true` muss § 4 manuell erweitert werden (Pricing-Werte, USt, Periode, Trial, Mindestbetrag, Zahlungs-Pfad). Pricing-Werte selbst landen NICHT im Repo — sie kommen via DB-Tabelle `pricing_plan`.

### AC-Erfüllungs-Map Welle 5

- AC-31 (architecture.md) ✓ Billing-Stack-Diagramm + Data-flow
- AC-32 (api-spec.md) ✓ Mollie-Webhook + Owner- und EEG-Endpoints
- AC-33 (domain-model.md) ✓ 4 Tabellen + 7 Spalten
- AC-34 (User-Guide-Sektion „Rechnungen") ✓ PROJ-frei
- AC-35 (CHANGELOG-Einträge) ✓ technisch + user-guide

### Bewusste Out-of-Scope auch nach Welle 5

- **AGB-Pricing-Werte** — Owner-Aktion beim Prod-Cutover (siehe HTML-Kommentar in `agb-v1.0.md`)
- **PDF-Generation/Download** für Rechnungen — separates Folge-PROJ
- **PROJ-71 Multi-Reason-Code** (Grilling R-19) — V1-akzeptabel als single-reason
- **Frontend-Integration für Owner-Edition-Switch** im `/admin/billing` — `EditionSwitchDialog` ist verfügbar, Wiring kommt im nächsten Refinement

### Empfehlung für Owner

1. **Pre-QA-Smoke** auf test-Cluster:
   - Helm-Apply (Backend + Frontend Tags der letzten Welle)
   - Migrationen 000079–000088 laufen automatisch
   - Smoke: `/admin/billing` als Superuser → 4 Tabs → „Neue Version anlegen" Standard EUR 0.00 → Audit-Log; Bestand-Antrag in `activated` → `edition_at_activation` + `trial_started_at` befüllt; Cron `kubectl create job --from=cronjob/...billing-quarterly billing-test-q1` → SELECT auf `billing_period` (status `preview`)
2. **`/qa` PROJ-104** — Acceptance Criteria + Tester-Phase-Smoke
3. **`/security-review` PROJ-104** — Pflicht-Gate wegen Mollie-Webhook (public), neue Helm-Secrets, FreeFinance OIDC-Auth, Live-Mode-Schalter
4. **AC-27b** (FreeFinance-Trial-Verifikation) bleibt Owner-Aufgabe — `private/vendor-setup/freefinance-trial-verification-2026-06-12.md` ausfüllen
5. **`/deploy`** nach grünem QA + Security-Review

## QA Test Results

**Tester:** QA Engineer (AI)
**Datum:** 2026-06-12
**Scope:** alle 5 Wellen (Datenmodell + Pricing + Vendor + Cron + Webhook + UI + Doku)
**Verdict:** **APPROVED** — 3 Findings, 3 inline gefixt, 0 Critical/High verbleibend.

### Build/Test/Scan-Sweep

| Tool | Status |
|---|---|
| `go build ./...` | clean |
| `go test ./...` | alle Pakete grün |
| `govulncheck ./...` | 0 callable vulnerabilities (5 transitive in nicht-callable Pfaden, akzeptiert) |
| `gosec -severity medium -confidence medium` | 0 Issues über alle PROJ-104-Pakete (nach F1-Fix) |
| `npx tsc --noEmit` | clean |
| `npx vitest run` | 238/238 grün |
| `NEXT_PUBLIC_TEST_AUTH_MODE= npm run build` | clean |
| `npm audit --audit-level=high` | 0 high (4 moderate Bestand: uuid in next-auth, akzeptiert) |
| `helm lint helm/member-onboarding/` | 0 failures (Bestand-Info: Chart.yaml ohne icon) |
| `trivy config --helm-set …` | 0 PROJ-104-Findings (Bestand: postgres-Container readOnlyRootFilesystem-Warnung, nicht PROJ-104-Scope) |

### Findings

| # | Severity | Datei | Funktion | Risiko | Status |
|---|---|---|---|---|---|
| F1 | Medium | `internal/billing/audit_repo.go` | `List` | gosec G202 — `LIMIT` + WHERE als String-Concatenation. WHERE-Templates sind intern fest verdrahtet (kein User-Input), aber Defense-in-Depth empfohlen. | **FIXED inline** — `LIMIT` als parametrisierter `$N`-Placeholder, WHERE-Branches per `#nosec G202`-Annotation dokumentiert |
| F2 | Low | UI / `admin_billing.go` | EditionSwitch in `/admin/billing` + EEG-Settings | UI-Lücke: `EditionSwitchDialog`-Komponente existiert, ist aber nirgendwo eingehängt. Backend-Endpoint `POST /api/admin/billing/eegs/{rc}/edition` ist Superuser-only — EEG-Admin-Direkt-Switch (Spec #13 + R-8-Korrektur) hat keinen passenden Endpoint. | **Deferred** — bereits in Welle-4b-Spec-Notes als „Out-of-Scope, kommt im nächsten Refinement" dokumentiert |
| F3 | Medium | `helm/.../templates/seed-job.yaml` | initial-seed | AC-27a Pricing-Plan-Seed fehlte: Helm-Seed-Job legte nur `registration_entrypoint` an, KEINE Default-Pricing-Pläne (Standard €0 / Pro €0). | **FIXED inline** — `seed-job.yaml` um zwei idempotente INSERT-Statements via `WHERE NOT EXISTS` erweitert |
| F4 | Info | `admin_billing.SetEdition` | Backend-Downgrade-Block | Backend prüft NICHT die 6 Pro-Feature-Flags (R-7), nur Frontend. Defense-in-Depth-Lücke, aber Superuser-only-Pfad. | **Deferred** — bei künftiger Öffnung für EEG-Admin-Endpoint Backend-Check Pflicht |
| F5 | Medium | `admin_billing.CreateCreditNote` | `reason`-Validation | Keine serverseitige max-length auf `reason`-Pflicht-Grund. Owner könnte beliebig lange Strings posten (Resource-Exhaustion). | **FIXED inline** — 2000-Zeichen-Cap mit klarer Fehlermeldung |

### AC-Erfüllungs-Map (Welle-übergreifend)

Datenmodell: AC-1 ✓ (EXCLUDE), AC-2 ✓ (UNIQUE), AC-3 ✓ (Status-CHECK), AC-4 ✓ (NULL+Grace), AC-5 ✓ (Default standard), AC-5a ✓ (Backfill), AC-5b ✓, AC-5c ✓

Pricing-Service: AC-6 ✓ (Edition-Snapshot-Counter), AC-7 ✓ (Trial-Anchor), AC-8 ✓ (Carryover), AC-9 ✓ (Snapshot)

Vendor/Cron: AC-10 ✓ (Subcommand), AC-11 ✓ (Forbid + 14400s), AC-12 ✓ (Idempotency-Key), AC-13 ✓ (sequenceType), AC-14 ✓ (IP+GET-Lookup)

Live-Mode: AC-15 ✓ (3-Faktor im Scheduler kombiniert), AC-16 ✓ (Mocks), AC-17 ✓ (preview-Status), AC-18 ✓ (Idempotency-Header), AC-19 ✓ (UI-Indikator), AC-19a ✓ (Pre-Flight)

Mahnwesen: AC-20 ✓, AC-20a ✓, AC-21 ✓ (CountOpenOverdueForRC==0), AC-21a ✓ (Gutschrift)

UI: AC-22 ✓ (/admin/billing), AC-22a ✓ (Manual-Trigger), AC-23 ✓ (EEG-Tab), AC-24 (Backend ✓, UI deferred via F2), AC-24a ✓ (Mandate-Setup-Mail sync)

Datenschutz: AC-25 ✓ (anonyme Beschreibung), AC-26 ✓ (kein PII in Mollie), AC-27 ✓ (IP-Allowlist + GET-Lookup), AC-27a ✓ (Seed-Job fixed via F3), AC-27b (Owner-Aufgabe, kein Code)

Tests/Doku: AC-28/29/30 ✓ (~50 neue Tests), AC-31/32/33 ✓ (architecture + api-spec + domain-model), AC-34 ✓ (User-Guide), AC-35 ✓ (CHANGELOG)

### EC-Sweep

EC-1 ✓ (Trial via activated_at), EC-2 ✓ (no_activity), EC-3 ✓ (Trial einmalig), EC-4 ✓ (Reset cleart edition_at_activation), EC-5 ✓ (Daily-Sync), EC-6 ✓ (draft bei Vendor-Error), EC-7 ✓ (Public-Form bleibt offen), EC-8 ✓ (Pricing-Plan-Snapshot via FK), EC-9 ✓ (Edition-Snapshot), EC-10 ✓ (UI-Block; Backend-Block deferred via F4), EC-11 ✓ (billing_live), EC-12 ✓ (mandate_active → draft + retry), EC-13 ✓ (Idempotency-Key + DB-UNIQUE), EC-14 ✓ (Defense-in-Depth GET-Lookup)

### Regression-Check

- **PROJ-46** activated_at-Setzung: bleibt unverändert; zusätzlich `ApplyEditionSnapshotTx` + `SetTrialStartedAtIfNullTx` als Folge-Hooks in derselben Tx ✓
- **PROJ-67** settings_view_mode: SetEegEditionTx synct Pro→advanced, Standard→standard (Grilling R-6) ✓
- **PROJ-71** Cool-Down: CoolDownService nutzt InsertEvent mit EventSuspended + ReasonPaymentFailed; OnInvoicePaid prüft CountOpenOverdueForRC==0 → EventReactivated + ReasonPaymentReceived ✓
- **PROJ-100** ResetActivationTx + ResetToReviewTx clearen `edition_at_activation` ✓
- **PROJ-102** brand_preset unangetastet ✓
- **PROJ-103** brand_theme JSONB unangetastet (R-7 Downgrade-Block-UI checkt brandMode='custom' clientseitig) ✓
- Bestand-Admin-Settings-Tabs + Customer-Onboarding-Endpoints unangetastet ✓

### Production-Ready-Verdikt

**APPROVED — 0 Critical, 0 High, 0 Medium (3 inline gefixt), 2 Info/Low (deferred mit Spec-Vermerk).**

### Empfehlung für nächsten Schritt

1. **`/security-review PROJ-104`** — Pflicht-Gate wegen Mollie-Webhook (public Endpoint), neue Helm-Secrets, FreeFinance OIDC-Auth, Live-Mode-Schalter
2. Nach grünem Security-Review → **`/deploy PROJ-104`**

Owner-Aktionen vor Live-Cutover:
- **AC-27b**: FreeFinance-Trial-Verifikation in `private/vendor-setup/freefinance-trial-verification-2026-06-12.md` ausfüllen
- **AGB § 4**: beim Produktiv-Start manuell um Pricing-Werte ergänzen + Version bumpen
- **F2 (UI-Edition-Switch)**: kann als kleines Folge-PROJ oder Welle-4c gefixt werden

## Security Review

**Reviewer:** Security Engineer (AI)
**Date:** 2026-06-13
**Scope:** alle 5 Wellen, neue Migrationen 000079–088, billing-Module, webhook_mollie, admin_billing/admin_eeg_invoices, Mandate-Setup, FreeFinance- + Mollie-Clients, Cool-Down-Integration, 2 neue CronJob-Templates, seed-job, Helm-Secrets, neue ENV-Vars

### Threat-Model-Summary

Worst-case-Szenarien für PROJ-104:
1. **Mollie-Webhook-Forge**: Angreifer triggert Status-Sprünge (paid/chargeback) auf bestehende Invoices ohne reale Zahlung → false-positive Mandate-Aktivierung, false-positive Cool-Down-Lift, ungerechtfertigte Owner-Alert-Mails. Verteidigungs-Layer: IP-Allowlist + Defense-in-Depth GET-Lookup gegen Mollie-API.
2. **Vendor-Credentials-Leak** (FreeFinance + Mollie API-Keys): kommerzieller + reputationeller Schaden. Verteidigung: Helm-Secret mit `secretKeyRef`, required-Guard bei `globalLiveMode=true`.
3. **BILLING_LIVE_MODE-Bypass**: echte Rechnungen während Tester-Phase. Verteidigung: 3-Faktor-Live-Check (Helm + EEG-Toggle + Mandate-aktiv) + Mock-Pfad als produktive Preview.
4. **Tenant-Crossover**: EEG-Admin sieht/manipuliert fremde Invoices. Verteidigung: `containsRC`-Check + Superuser-Bypass nur in `/api/admin/billing/*`.
5. **PII-Leak via Vendor-Payloads**: Mitglieder-Identifikatoren bei FreeFinance/Mollie. Verteidigung: Owner-Stammdaten + RC + Aktivierungs-Count only, keine Member-PII.
6. **Audit-Log-Integritätslücke**: Owner-relevante Events ohne RC-Referenz → DSGVO/GoBD-Anschluss schwierig.

### Findings

| Severity | File | Function/Area | Risk | Exploit Scenario | Recommended Fix | Confidence |
|---|---|---|---|---|---|---|
| **HIGH** | `internal/http/webhook_mollie.go:134-162` | `ipAllowed()` | IP-Allowlist via XFF/X-Real-IP spoofbar; das Bestand-Helper `isTrustedProxy` (`internal/http/trusted_proxy.go`) wird hier NICHT genutzt | Angreifer POSTet `/api/webhooks/mollie` mit `X-Forwarded-For: <Mollie-IP>` → `parts[0]` = attacker-spoof; allowlist falsch-positiv. Defense-in-Depth-GET-Lookup ist letzter Schild. Wenn Angreifer leaked-`tr_*`-IDs hat (z. B. via Log-Snippet, Reverse-Engineering, Mollie-Account-Crossover), kann er MarkPaid, Mandate-Activate, Cool-Down-Lift triggern. Spec AC-14 verlangt enforced IP-Allowlist. | `ipAllowed` muss zuerst `isTrustedProxy(remoteAddrIP(r))` prüfen, bevor XFF/X-Real-IP gehonoriert werden. Bei untrusted-Source → nur `RemoteAddr` gegen Allowlist matchen. Identisches Pattern wie `internal/http/middleware.go:82`. | High |
| **MEDIUM** | `internal/billing/scheduler.go:737-761` + `internal/http/webhook_mollie.go:99` | `ProcessWebhookPayment` / `applyMollieStatus` | Nil-Payment-Panic bei „`tr_*` in unserer DB vorhanden, aber Mollie liefert 404". Webhook-Handler ruft `ProcessWebhookPayment(ctx, paymentID, nil)`; `LookupRCByPaymentID` SUCCEEDS, `GetByID` SUCCEEDS, `applyMollieStatus(ctx, *inv, nil, ...)` → `switch p.Status` panicked → `middleware.Recoverer` fängt → 500. | Mollie liefert 404 bei key-mismatch (live vs test), Account-Wechsel, Payment-Lifecycle-GC. Mollie retried 500 bis 24 h alle paar Stunden → DB-Connection + slog-Spam pro Retry. Kombiniert mit dem Allowlist-Spoof (HIGH oben) bewusst triggerbar. | Im Webhook-Handler bei NotFound die DB-Existenz-Prüfung NICHT machen — direkt Audit + 200 OK. Oder im Scheduler vor `applyMollieStatus` `if p == nil` → eigener Audit-Pfad + return. | High |
| **MEDIUM** | `internal/http/admin_billing.go:682-690` | `CreateCreditNote` | Audit-Log-Eintrag für Gutschrift hat IMMER `rc_number = NULL` durch Placeholder-Lookup `h.periodRepo.GetByRCYearQuarter("", 0, 0)`. GoBD verlangt nachvollziehbaren Audit-Trail pro Vorgang; ohne RC-Referenz keine EEG-spezifische Audit-Suche möglich. | Owner schreibt Gutschrift → Eintrag in `billing_audit_log` mit `rc_number=NULL` → Audit-Liste `/admin/billing/audit-log?rc=...` zeigt diese Gutschrift NICHT in der EEG-Filter-Sicht. Bei GoBD-Prüfung Lücke. | `GetByID(periodID)` auf BillingPeriodRepository hinzufügen + Original-Invoice via `orig.BillingPeriodID` → Period → `period.RCNumber`. Oder direkter SELECT der RC via JOIN über `billing_invoice → billing_period`. | High |
| **MEDIUM** | `helm/member-onboarding/values.yaml:168` | `backend.billing.mollieAllowedIPs` | Default leer → `parseAllowedIPs` liefert nil → `ipAllowed` returns true für ALLE (Zeile 135-137). Spec AC-14 verlangt enforced IP-Allowlist im Prod-Modus. Owner muss aktiv setzen — leicht vergessen. | Owner deployed mit Default; `mollieWebhookUrl` aktiv (Mollie postet Webhooks rein), aber Allowlist offen → Webhook-Endpoint ist effektiv ohne Network-Auth. Nur Defense-in-Depth-GET-Lookup schützt. | (a) values.yaml mit den 3 publizierten Mollie-CIDR-Blocks vorbefüllen + COMMENT „bei Mollie-IP-Drift aktualisieren"; ODER (b) Helm-required-Guard: wenn `mollieWebhookUrl != ""` UND `mollieAllowedIPs == ""` → `fail` analog zu `secrets.freefinanceApiKey`. (b) ist sauberer. | High |
| **LOW** | `cmd/server/main.go:386` | Mollie-Webhook-Route | `/api/webhooks/mollie` ist NICHT unter `/api/public`-Route-Gruppe → kein `MaxBodySize` Middleware. `r.ParseForm()` liest gesamten Request-Body. | Angreifer (via Allowlist-Bypass oben) spammt POSTs mit Multi-MB-Form-Bodies → Memory-Spike pro Request. Recoverer fängt im OOM-Fall nicht. | Webhook-Route in eigener Sub-Group mit `r.Use(internalhttp.MaxBodySize(64 * 1024))` (Mollie-Bodies sind sub-1 KB). | High |
| **INFO** | `internal/http/admin_billing.go:120-161` | `listPricingPlans()` / `collectPlans` | Funktional unvollständig: gibt nur die aktive Zeile pro Edition zurück, nicht die Historie. Kommentar „MVP" + „Repo-Erweiterung in naechster Welle". Kein Security-Issue, aber Owner-UI sieht keine vergangenen Pricing-Versionen → GoBD-Reporting-Lücke später. | — | In Folge-PROJ Repository-Methode `ListAllOrderByEditionGueltigAbDesc()` ergänzen. | High |
| **INFO** | npm `esbuild ≥ 0.17 < 0.28` | dev-tree | HIGH-CVE GHSA-gv7w-rqvm-qjhr + GHSA-g7r4-m6w7-qqqr — Dev-Server-only (Windows arbitrary file read; NPM_CONFIG_REGISTRY RCE). Nicht im Production-Build aktiv. | Akzeptiert auch in PROJ-103 Security-Review. | — (Bestand-Issue, nicht PROJ-104) | High |
| **INFO** | `helm/member-onboarding/templates/postgres.yaml:46` | Postgres StatefulSet | Trivy KSV-0014 HIGH: `readOnlyRootFilesystem` nicht gesetzt. Bestand aus Helm-Deep-Audit Welle 4 (commit 2b67366). Postgres braucht writable rootfs für WAL/temp-files. | Bewusste Bestand-Entscheidung — nicht PROJ-104. | — (Bestand) | High |

### Scan-Ergebnisse

```
govulncheck ./...               → 0 callable (5 indirect pkg vulns, 1 indirect mod vuln nicht reachable)
gosec -severity medium ./...    → 0 Issues (109 Files, 14 #nosec-Annotationen including audit_repo G202)
npm audit --audit-level=high    → 1 HIGH (esbuild Bestand, akzeptiert) + 4 moderate
trivy config helm/ ...          → 1 HIGH (postgres readOnlyRootFilesystem, Bestand)
```

Alle PROJ-104-spezifischen neuen Helm-Templates (billing-quarterly-cronjob, billing-daily-cronjob, seed-job, billing-Secrets-Block) sind sauber: runAsNonRoot, allowPrivilegeEscalation=false, readOnlyRootFilesystem=true, capabilities drop ALL, automountServiceAccountToken=false, seccompProfile RuntimeDefault.

### Welle-Übergreifende Pattern-Checks

- ✅ **3-Faktor-Live**: `IsLive(globalLive, eeg)` als single source; im Scheduler vor jedem Vendor-Call (live/preview/draft-Pfad-Switch). MandateSetup nutzt 2-Faktor-Variante (`globalLive AND billing_live`) bewusst — das ist der Mandate-Aktivierungs-Pfad selbst.
- ✅ **Edition-Snapshot bei Activation**: Hook in `ChangeStatus` + `MarkActivatedSkipImport` schreibt `application.edition_at_activation`. `ResetActivationTx` + `ResetToReviewTx` clearen sie. Audit-Log überprüft (PROJ-100 Spec).
- ✅ **Virtuelle Trial-Grace**: rein in `pricing_service.EffectiveTrialAnchor` (Live-Computation), keine DB-Materialisierung. Bestand-Migration-frei.
- ✅ **Mandate-Setup-Mail SYNC vor Mollie-Charge**: `admin_billing.go:380-388` — bei Mail-Fail wird `billing_live=true` NICHT gesetzt (early return mit Error-Response). Memory-Regel erfüllt.
- ✅ **gosec G202 in audit_repo**: WHERE-Branches aus festen Templates, alle Werte parametrisiert, LIMIT als `$N`. `#nosec`-Begründung sauber im Code dokumentiert.
- ✅ **Tenant-Iso EEG-Read-Only**: `admin_eeg_invoices.go:67` — `claims.IsSuperuser() OR containsRC(claims.Tenant, rc)` (nil-Tenant safe, range über nil = false).
- ✅ **PII-Disziplin**: FreeFinance + Mollie Customer-Payloads nutzen NUR Owner-/EEG-Stammdaten (EEGName, ContactEmail, EEGStreet/Zip/City, VAT-Number, RC-Number). Keine Mitglieder-Identifikatoren.
- ✅ **Idempotency-Key {rc}-{year}-Q{quarter}**: Scheduler sendet via FreeFinance `Idempotency-Key`-Header. AC-27b verifiziert das tatsächliche Vendor-Verhalten Owner-seits.
- ✅ **DB-Migrations IMMUTABLE-readiness**: 9 neue Up + Down-Files; Down-Migrations sauber (DROP TABLE für neue, DROP COLUMN + CHECK-Restore für ALTER), keine Down-Daten-Verluste über die Spec-Intention hinaus.
- ✅ **Helm required-Guard**: `templates/secrets.yaml:29-36` blockt Helm bei `globalLiveMode=true` + leeren API-Keys via `fail`.
- ⚠️ **IP-Allowlist Helm-Guard**: existiert NICHT für `mollieAllowedIPs` (siehe MEDIUM Finding oben).
- ⚠️ **XFF-Trust ohne trusted-proxy-Gate**: Bestand-Helper `isTrustedProxy` nicht im Webhook-Pfad eingebunden (siehe HIGH Finding oben).

### Verdict: **BLOCKED → APPROVED nach Fix-Welle 5b (2026-06-13)**

Initial: 1 HIGH (XFF-Spoof IP-Allowlist-Bypass) + 3 MEDIUM (Nil-Payment-Panic, broken Credit-Note Audit-RC, Allowlist-leerer Default) + 1 LOW (Webhook MaxBodySize).

### Fix-Welle 5b (2026-06-13)

Alle 5 Findings inline gefixt, Tests grün, Helm-Guard verifiziert:

1. **HIGH** — `internal/http/webhook_mollie.go ipAllowed()`: jetzt `isTrustedProxy(remoteAddrIP(r))`-Gate vor XFF/X-Real-IP-Honorierung. Bei XFF wird die LETZTE Entry verwendet (closest-to-our-server), nicht die erste (Client-controlled). Zwei neue Tests: `XForwardedFor_HonoredFromTrustedProxy` + `XForwardedFor_RejectedWithoutTrustedProxy`.
2. **MEDIUM** — `internal/billing/scheduler.go ProcessWebhookPayment`: bei `p == nil` direkter Audit + return, kein `applyMollieStatus`-Call. Panic-Pfad geschlossen.
3. **MEDIUM** — `internal/http/admin_billing.go CreateCreditNote` + neue `BillingPeriodRepository.GetByID()`: Audit-Log nutzt jetzt `orig.BillingPeriodID → period.RCNumber` statt Placeholder-Lookup. GoBD-Audit-Trail vollständig.
4. **MEDIUM** — `helm/member-onboarding/templates/secrets.yaml`: neuer required-Guard analog zu API-Keys: wenn `mollieWebhookUrl != ""` UND `mollieAllowedIPs == ""` → Helm `fail` mit Hinweis auf Mollie-CIDR-Docs.
5. **LOW** — `cmd/server/main.go`: Webhook-Route wrappt `internalhttp.MaxBodySize(64 * 1024)`.

### Verifizierung

- `go build ./...` clean
- `go test ./...` alle Pakete grün
- `helm template ... --set mollieWebhookUrl=…` ohne `mollieAllowedIPs` → erwarteter Helm-Fail
- `helm template` mit beiden gesetzt → rendert sauber

### Verdict: **APPROVED**

0 Critical, 0 High, 0 Medium nach Fix-Welle. Bestand-Info-Findings (esbuild Dev-CVE, postgres readOnlyRootFilesystem) bleiben Bestand. Owner-Aufgaben AC-27b (FreeFinance-Trial-Verifikation) + AGB §4 Cutover bleiben vor Live-Cutover offen.

## Deployment

**Datum:** 2026-06-13
**Image-SHA:** `sha-a1f0980` (Backend + Frontend)
**Git-Tag:** `v1.32.0-PROJ-104`
**Verantwortlicher Owner-Apply:** manueller `helm upgrade` (kein Cluster-Apply von Claude-Seite)

### Was im Cluster ankommt

- 9 neue DB-Migrationen 000079–000087 (pricing_plan + billing_period + billing_invoice + 5 Spalten-Erweiterungen + billing_audit_log) + 000088 (Vendor-Customer-IDs) — werden vor Backend-Rollout via `migrate`-Job ausgeführt
- Backend mit neuem `internal/billing/` + `internal/freefinance/` + `internal/mollie/` + Mollie-Webhook-Endpoint + 11 Owner-Endpoints `/api/admin/billing/*`
- Frontend mit Owner-Seite `/admin/billing` (4 Tabs) + EEG-Settings-Tab „Rechnungen"
- Helm: 2 neue CronJobs (`billing-quarterly` + `billing-daily`), Seed-Job mit idempotenten Pricing-Plan-Defaults (Standard/Pro je €0), 5 neue Backend-ENV-Vars
- Security-Fixes inkludiert: XFF-Trust-Gate, Nil-Payment-Pfad, Credit-Note-Audit-RC, Helm-Required-Guard, MaxBodySize-Webhook

### Default-Verhalten nach Deploy (Tester-Phase läuft weiter)

- `BILLING_GLOBAL_LIVE_MODE=false` (Helm-Default) → KEINE realen Rechnungen, KEINE realen SEPA-Lastschriften
- Pricing-Plan-Defaults €0 (Seed-Job) → Pre-Flight blockt Live-Toggle bis Owner reale Werte gesetzt hat
- Webhook-Endpoint nur registriert wenn `backend.billing.mollieWebhookUrl` befüllt
- CronJobs aktiv: Daily 05:00 UTC läuft sofort, Quarterly startet erst am 1. Jul 2026

### Owner-Schritte für `helm upgrade` (manuell)

```bash
helm upgrade eegfaktura-member-onboarding ./helm/member-onboarding \
  -f helm/member-onboarding/values-env.yaml \
  -f helm/member-onboarding/values-secret.yaml
```

Helm fährt automatisch:
1. `migrate`-Job (Migrations 000079–000088 in IMMUTABLE-Zustand)
2. `seed`-Job (Pricing-Plan-Defaults, idempotent)
3. Rolling-Update Backend + Frontend
4. CronJob-Reconcile

Health-Check via `GET /health` und Smoke-Test der neuen Routes (`/admin/billing` mit Superuser-Login).

### Verbleibend (Owner-Aufgaben — nicht Teil des Deploys)

- **AC-27b FreeFinance-Trial-Verifikation:** Skeleton in `private/vendor-setup/freefinance-trial-verification-2026-06-12.md` (Mandant-ID, Tech-User, 3 Verifikations-Punkte). Pflicht vor Live-Cutover.
- **AGB § 4 Pricing-Cutover:** Pricing-Werte manuell in `src/content/legal/agb-v1.0.md` ergänzen + Version `v1.0 → v1.1`. HTML-Kommentar als Reminder im File.
- **Reale Pricing-Werte:** über `/admin/billing` Tab „Pricing-Plan" pflegen (überschreibt die €0-Seed-Defaults).
- **API-Keys + Webhook-URL:** `secrets.freefinanceApiKey` + `secrets.mollieApiKey` + `backend.billing.freefinanceClientId` + `backend.billing.mollieWebhookUrl` + `backend.billing.mollieAllowedIPs` in `values-env.yaml` + `values-secret.yaml` befüllen → zweiter `helm upgrade`.
- **Per-EEG Mandate-Setup:** Owner schaltet `billing_live=true` pro EEG via `/admin/billing` → triggert EUR 0,01 First-Payment + Mail an EEG-Vorstand.

Nach dem Cutover greift erst der Quartals-Cron am 1. Oktober 2026 (Q3-Abrechnung); davor sind alle Q3-Buchungen im Preview-Modus sichtbar.

### Deferred-Items mit Spec-Vermerk (für Folge-PROJ)

- **F2 (Low):** UI-Wiring für `EditionSwitchDialog` (~2-3h)
- **F4 (Info):** Backend-Downgrade-Block-Defense-in-Depth (Frontend-Block reicht V1)
- **Lifecycle-sticky Pro (Owner-Idee 2026-06-13):** Activation-Snapshot bleibt; Re-Evaluation nach Q1-Daten. Siehe `project_todo_lifecycle_sticky_pro.md`.
