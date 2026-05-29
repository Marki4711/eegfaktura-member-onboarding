# PROJ-64: Faktura-Handover-Billing-Trigger

**Status:** Deployed
**Created:** 2026-05-29

## Hintergrund

Die geplante Quartals-Verrechnung der SaaS-Nutzung soll auf der Anzahl der
neu an eegFaktura übergebenen Anträge basieren (siehe Pricing-Modell V2
2026-05-25, intern dokumentiert). Bisher zählte als "Übergabe" nur ein
erfolgreicher `POST /api/admin/applications/{id}/import` —
operationalisiert über `application.imported_at IS NOT NULL`.

Lücke: der Endpoint `GET /api/admin/applications/{id}/export/excel`
liefert genau dieselben Antragsdaten im eegFaktura-Import-Template
(36 Spalten A-AJ, siehe `internal/excel/generator.go`). Ein Admin kann
den Antrag direkt im eegFaktura-Frontend über Datei-Upload importieren
und entzieht sich damit der Verrechnung. Tester hat die Lücke
2026-05-29 gemeldet.

## Ziel

Den Billing-Trigger so um den Excel-Bypass erweitern, dass jede
Erst-Übergabe an eegFaktura — egal über welchen Weg — gezählt wird.

## Design-Entscheidung

Neue Spalte `application.faktura_handover_at TIMESTAMPTZ`. Wird vom
ersten der beiden Wege gesetzt:

1. Erfolgreicher `POST /import` (Core-Push)
2. Erster Download des Faktura-Format-Excel via `GET /export/excel`

Idempotent: spätere Downloads/Re-Imports lassen den Wert unverändert
(via `SetFakturaHandoverAtIfEmpty`). Damit zählt jeder Antrag maximal
einmal — egal wie oft der Admin das Excel zieht oder den Import
zurücksetzt + neu importiert.

Geplante Quartals-Cron schwenkt von `imported_at IS NOT NULL` auf
`faktura_handover_at IS NOT NULL`. `imported_at` bleibt für die
Status-Logik (Audit, Reset-Import-Erkennung, Mail-Templating).

## Verworfene Alternativen

- **Billing auf `approved`-Status umstellen** — semantisch anders
  (Kunde zahlt für den Genehmigungs-Akt statt für die Übergabe). Würde
  mit dem heute gerade hinzugefügten `approved → rejected`-Pfad zu
  Refund-Logik führen. Komplexer ohne klaren Vorteil.
- **Excel-Export aus dem Faktura-Format entfernen** — würde laufende
  Admin-Workflows brechen, die das xlsx als Backup-/Audit-Format nutzen.
  Datenweiterleitung (PROJ-60) bleibt als nicht-verrechnungsrelevanter
  Pfad für reine Backup-Use-Cases bestehen.

## Datenmodell

```sql
ALTER TABLE member_onboarding.application
    ADD COLUMN faktura_handover_at TIMESTAMPTZ NULL;

CREATE INDEX idx_application_faktura_handover_at
    ON member_onboarding.application (faktura_handover_at, rc_number)
 WHERE faktura_handover_at IS NOT NULL;
```

Partial-Index optimiert die Quartals-Billing-Query (Anzahl Anträge mit
handover_at im Quartal, gruppiert nach `rc_number`). NULL-Zeilen
bleiben außerhalb des Index.

Migration 000057 enthält den Backfill:

```sql
UPDATE member_onboarding.application
   SET faktura_handover_at = imported_at
 WHERE imported_at IS NOT NULL;
```

Pre-PROJ-64-Imports werden damit nicht versehentlich als neu billbar
gewertet.

## API-Surface

- `AdminApplicationDetail.fakturaHandoverAt` (JSON: `fakturaHandoverAt`,
  Typ `string | null` im Frontend). Read-only — nicht via Edit/Update
  setzbar.
- Keine neuen Endpoints. Beide Trigger-Pfade existieren bereits
  (`/import` und `/export/excel`); sie setzen den Wert serverseitig
  als Seiteneffekt.

## Frontend

`admin-application-detail.tsx`:

- Marker-Zeile unter dem Antragsteller-Namen im Header:
  „An eegFaktura übergeben am <Datum>" — sichtbar wenn
  `fakturaHandoverAt != null`.
- Confirmation-Dialog beim ERSTEN Excel-Download (Faktura-Format).
  Klärt den Admin auf, dass der Download verrechnungsrelevant ist und
  verweist auf die Datenweiterleitung (PROJ-60) als nicht-verrechnete
  Alternative für Backup-Use-Cases. Bei wiederholten Downloads
  (`fakturaHandoverAt != null`) bleibt der Dialog aus — die Übergabe
  ist bereits vermerkt.

## Akzeptanz-Kriterien

- [x] AC-1: Migration `000057_add_faktura_handover_at` läuft idempotent
  durch (up + down + Backfill aus `imported_at`).
- [x] AC-2: Erfolgreicher `POST /import` setzt `faktura_handover_at`,
  wenn die Spalte NULL ist. Re-Import nach Reset ändert den Wert nicht.
- [x] AC-3: Erster `GET /export/excel` setzt `faktura_handover_at`.
  Spätere Downloads ändern den Wert nicht.
- [x] AC-4: `application.fakturaHandoverAt` ist im
  Admin-Detail-Endpoint enthalten.
- [x] AC-5: Frontend zeigt Marker-Zeile bei nicht-NULL handover_at und
  Confirmation-Dialog beim ersten Excel-Download.
- [x] AC-6: Backend-Tests decken die Idempotenz von
  `SetFakturaHandoverAtIfEmpty` ab.

## Smoke-Tests

- Antrag importieren → `fakturaHandoverAt` im Detail-Endpoint gesetzt
  auf Import-Timestamp.
- Antrag zuerst per Excel ziehen → handover_at = jetzt; danach
  importieren → handover_at unverändert (= Excel-Zeitpunkt).
- Antrag importieren, dann „Import zurücksetzen" → handover_at bleibt
  gesetzt. Erneuter Import → keine doppelte Zählung im Billing.

## Out of Scope

- Implementierung der Quartals-Billing-Cron selbst. Dieser PROJ
  liefert nur den Trigger; der Cron + Free-Tier-Snapshot wandert in
  einen späteren PROJ (siehe interne Pricing-Memo).
- Backfill der `core_active_members_at_handover`-Spalte. Diese wird
  vom Billing-PROJ eingeführt, sobald die Core-Mitgliederzahl pro
  EEG aus dem Faktura-Backend lesbar ist.
