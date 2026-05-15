# PROJ-37: Genossenschaftsanteile (per-EEG-Konfiguration + Antragsfeld)

## Status: Architected
**Created:** 2026-05-15
**Last Updated:** 2026-05-15 (Konzept verfeinert: konfigurierbare Pflichtanteile statt fest 1; Form-Min und -Default = konfigurierter Pflichtwert; Konfig-Änderungen wirken nur prospektiv.)

## Dependencies
- Berührt: PROJ-1 (Public Registration — neues Formular-Feld), PROJ-2 (Admin Review — Anzeige im Antrags-Detail), PROJ-19 (Manuelle Aktivierung — Settings-Seite), PROJ-21 (Beitrittsbestätigung-PDF — neue Sektion im Datenblock).
- Nicht berührt: Core-Import (PROJ-4) und Excel-Export (PROJ-17) — Genossenschaftsanteile bleiben **rein im Member-Onboarding**, eegFaktura hat keinen Platz dafür.

## Hintergrund

EEGs mit Genossenschafts-Rechtsform verlangen von neuen Mitgliedern beim Beitritt die Zeichnung mindestens eines Pflichtanteils. Die genaue Anzahl der Pflichtanteile (z.B. 1, 3, oder 5) ist satzungs­spezifisch und variiert pro EEG. Der Wert eines einzelnen Anteils ist ebenfalls satzungs­spezifisch (z.B. €75, €100). Mitglieder dürfen freiwillig **mehr** als den Pflichtwert zeichnen — die Zeichnungs-Untergrenze ist allerdings hart, ohne Pflichtanteil keine Mitgliedschaft.

EEGs ohne Genossenschaftsstruktur (Verein, GmbH, Gemeinde, …) blenden das Feld vollständig aus.

## User Stories

- Als **EEG-Admin einer Genossenschaft** möchte ich in den Einstellungen drei Werte konfigurieren: (a) ob Anteile erfasst werden, (b) wieviele Pflichtanteile mindestens gezeichnet werden müssen, (c) welcher Euro-Wert ein Anteil hat.
- Als **EEG-Admin einer Nicht-Genossenschafts-EEG** sehe ich dieses Feature im Default-State (Toggle aus) und werde im Formular nicht damit konfrontiert.
- Als **neues Mitglied einer Genossenschafts-EEG** möchte ich beim Ausfüllen sehen, wieviele Pflichtanteile ich zeichnen muss, kann freiwillig mehr eingeben, und sehe live wieviel Geld das insgesamt ergibt.
- Als **EEG-Admin** möchte ich im Antrags-Detail die gezeichneten Anteile sehen (Anzahl × Anteilswert = Gesamtbetrag) und gegebenenfalls korrigieren können, ohne das Mitglied über den `needs_info`-Flow zu schicken.
- Als **EEG-Admin** möchte ich, dass die Anteils-Information in der **Beitrittsbestätigung** als eigene Zeile ausgewiesen ist — als Buchhaltungs-Beleg.

## Architekturentscheidungen

1. **Drei Einstellungs-Felder auf `registration_entrypoint`:**
   - `cooperative_shares_enabled BOOLEAN NOT NULL DEFAULT FALSE` — Master-Toggle.
   - `cooperative_required_shares INT NULL` — Pflichtanteil-Mindestmaß (NULL wenn disabled, sonst `≥ 1`).
   - `cooperative_share_amount_cents BIGINT NULL` — Preis pro Anteil in Cent (NULL wenn disabled, sonst `> 0`).

2. **Per-Antrag-Wert auf `application`:** `cooperative_shares_count INT NULL` — Anzahl, die das Mitglied gezeichnet hat.

3. **Geld als Integer-Cents** (`BIGINT`). Keine Float-Drift, keine Rundungsprobleme. Konvention für künftige Geld-Spalten.

4. **Conditional Settings-UI:** die zwei Wert-Felder (Pflichtanteile + Anteilswert) sind in der Admin-Settings-Card nur sichtbar, wenn der Toggle aktiv ist. Analog zum bestehenden SEPA-Pattern (Firmenlastschrift erscheint erst bei aktivem SEPA-Toggle).

5. **Konfig-Konsistenz beim Save:** `SaveEEGSettings` validiert, dass `enabled=TRUE ⇒ required_shares ≥ 1 ∧ share_amount_cents > 0`. Sonst 400 mit Feldhinweisen. DB-CHECK ergänzt: Werte selbst dürfen nur positiv sein.

6. **Form-Min und -Default = konfigurierter Pflichtwert.** Beim ersten Render: `count = required_shares`. Min-Validierung: `count ≥ required_shares`. Mitglied kann nach oben überschreiben, nicht darunter.

7. **Konfig-Änderungen wirken nur prospektiv.** Wenn der EEG-Admin `required_shares` später ändert (z.B. von 1 auf 3), bleiben **bestehende Anträge unberührt** — auch wenn deren `count` dadurch unter dem neuen Min liegt. Submit-Validierung greift nur bei neuen Submits und beim Admin-Edit-Save (siehe AC).

8. **Gesamtbetrag wird NICHT gespeichert.** `count × share_amount_cents` ist eine reine Render-Berechnung. Speicherung wäre eine Drift-Quelle.

9. **Nicht in Excel-Export, nicht im Core-Payload.** Bewusste Entscheidung: eegFaktura hat keine Spalte für Genossenschaftsanteile, also fließt der Wert dort nirgends hin. Bleibt rein im Onboarding.

10. **PDF: in der Beitrittsbestätigung, nicht im SEPA-Mandat.** Beitrittsbestätigung dient als Buchhaltungs­beleg; SEPA-Mandat regelt den wiederkehrenden Beitragseinzug und ist semantisch getrennt.

## Synced Fields (Zusammenfassung)

| Wo | Spalte | Typ | Wer setzt | Gültigkeit |
|---|---|---|---|---|
| `registration_entrypoint` | `cooperative_shares_enabled` | BOOLEAN NOT NULL | EEG-Admin (Settings) | jederzeit änderbar, wirkt prospektiv |
| `registration_entrypoint` | `cooperative_required_shares` | INT NULL, CHECK `> 0` wenn nicht NULL | EEG-Admin (Settings) | Pflicht wenn enabled=TRUE |
| `registration_entrypoint` | `cooperative_share_amount_cents` | BIGINT NULL, CHECK `> 0` wenn nicht NULL | EEG-Admin (Settings) | Pflicht wenn enabled=TRUE |
| `application` | `cooperative_shares_count` | INT NULL, CHECK `> 0` wenn nicht NULL | Mitglied (Submit) bzw. Admin (Korrektur) | Pflicht im Submit wenn EEG enabled |

## Acceptance Criteria

### Stage A: DB-Migration

- [ ] `db/migrations/000035_cooperative_shares.up.sql`:
  ```sql
  ALTER TABLE member_onboarding.registration_entrypoint
      ADD COLUMN cooperative_shares_enabled BOOLEAN NOT NULL DEFAULT FALSE,
      ADD COLUMN cooperative_required_shares INT NULL
          CHECK (cooperative_required_shares IS NULL OR cooperative_required_shares > 0),
      ADD COLUMN cooperative_share_amount_cents BIGINT NULL
          CHECK (cooperative_share_amount_cents IS NULL OR cooperative_share_amount_cents > 0);
  ALTER TABLE member_onboarding.application
      ADD COLUMN cooperative_shares_count INT NULL
          CHECK (cooperative_shares_count IS NULL OR cooperative_shares_count > 0);
  ```
- [ ] `.down.sql` droppt die vier Spalten.

### Stage B: Backend — Models + Repo

- [ ] `shared.RegistrationEntrypoint` bekommt `CooperativeSharesEnabled bool`, `CooperativeRequiredShares *int`, `CooperativeShareAmountCents *int64`.
- [ ] `shared.Application` bekommt `CooperativeSharesCount *int`.
- [ ] `RegistrationEntrypointRepository.GetByRCNumber` liest die drei neuen Spalten mit.
- [ ] `RegistrationEntrypointRepository.SaveEEGSettings` Signatur wird um die drei neuen Parameter erweitert. Validierung im Service-Layer (`AdminService.SaveEEGSettings`):
  - `enabled=true && (required_shares==nil || amount==nil)` → ValidationError mit Feld-Map
  - `required_shares <= 0` → ValidationError
  - `amount <= 0` → ValidationError
  - `enabled=false` → required_shares + amount werden auf NULL gesetzt (cleanup; bisheriger Wert wird verworfen)
- [ ] `ApplicationRepository` Create/Update/Get: schreiben/lesen `cooperative_shares_count`.

### Stage C: Backend — Submit-Validierung

- [ ] `ApplicationService.SubmitApplication`: wenn `entrypoint.CooperativeSharesEnabled`:
  - `app.CooperativeSharesCount == nil` → ValidationError „Anzahl der Genossenschaftsanteile ist erforderlich"
  - `*app.CooperativeSharesCount < *entrypoint.CooperativeRequiredShares` → ValidationError „Mindestens {required} Pflichtanteil(e) müssen gezeichnet werden"
- [ ] `AdminApplicationService.AdminUpdateApplication`: dieselbe Validierung auf Admin-Edit, falls EEG enabled. **Ausnahme**: wenn der EEG-Admin den Wert auf NULL setzen will, ist das nur erlaubt wenn das EEG inzwischen `enabled=false` ist (Bestandsschutz).
- [ ] Keine Reload-Validierung bei Konfig-Änderung: ändert ein Admin später `required_shares`, werden Bestandsanträge nicht erneut geprüft.

### Stage D: Public-Registration-Config-Endpoint

- [ ] `GET /api/public/registration/{rc_number}` liefert zusätzlich (nur wenn enabled):
  ```json
  "cooperativeSharesEnabled": true,
  "cooperativeRequiredShares": 1,
  "cooperativeShareAmountCents": 10000
  ```
  Bei `enabled=false`: nur `cooperativeSharesEnabled: false`, andere zwei Felder weggelassen.

### Stage E: Public-Form-Frontend

- [ ] Card-Block **„Genossenschaftsanteile"** zwischen Zählpunkten und Bankverbindung, bedingt gerendert auf `config.cooperativeSharesEnabled`.
- [ ] Read-only Hint: „Pflichtanteil je Standort: **{required}** Anteil(e)" — Wert aus Config.
- [ ] Eingabe „Anzahl Anteile gesamt *": Number-Input, `min={required}`, Default-Wert = `required`. Live-Validierung im Zod-Schema (`refine` mit Zugriff auf `cooperativeRequiredShares` aus dem Config-Kontext).
- [ ] Live-Berechnung (HTML-Block, kein Input):
  ```
  Genossenschaftsanteilswert:   100,00 €
  Gezeichnete Anteile:          × 3
  ─────────────────────────────
  Gesamtbetrag:                 300,00 €
  ```
  Format via `Intl.NumberFormat("de-AT", { style: "currency", currency: "EUR" })`.
- [ ] Bei `cooperativeSharesEnabled=false`: Field-Block komplett ausgeblendet, Schema kennt das Feld nicht.

### Stage F: Admin-Settings-UI

- [ ] In `admin-eeg-settings-editor.tsx`: neuer Abschnitt **„Genossenschaftsanteile"** mit:
  - Toggle „Genossenschaftsanteile erfassen"
  - **Conditional, nur sichtbar wenn Toggle = aktiv:**
    - Number-Input „Pflichtanteile je Standort *" (min=1)
    - Decimal-Input „Anteilswert (€) *" (>0, mit `de-AT`-Format; Frontend wandelt in Cents)
  - Hint-Text: „Diese Werte werden auf der Beitrittsbestätigung als Anzahl × Anteilswert = Gesamtbetrag ausgewiesen. Werden **nicht** an eegFaktura übermittelt — reine Onboarding-Erfassung."
- [ ] Frontend-Validierung: bei aktivem Toggle Pflicht beide Werte; Schema lehnt Save sonst clientseitig ab.
- [ ] Wenn Admin den Toggle auf AUS setzt: Felder werden ausgeblendet; beim nächsten Save werden Backend-Werte auf NULL gesetzt (cleanup).

### Stage G: Admin-Antrags-Detail

- [ ] `AdminApplicationDetailResponse` enthält `cooperativeShares: { count, requiredShares, amountCents, totalCents } | null`. Backend joint zur Render-Zeit `application.count` mit dem EEG-Entrypoint und berechnet Total.
- [ ] `admin-application-detail.tsx`: konditionale Mini-Box „Genossenschaftsanteile: **3** × 100,00 € = **300,00 €". Wenn Bestand unter aktuellem Pflichtmaß: zusätzlich orangener Hinweis „Liegt unter aktuell konfiguriertem Pflichtmaß von {currentRequired}" — nur informativ, kein Block.
- [ ] Admin-Edit-Form (`admin-edit-form.tsx`): wenn EEG aktiv, Eingabefeld für `cooperativeSharesCount` mit Min-Validierung und Live-Total.

### Stage H: Beitrittsbestätigungs-PDF

- [ ] `pdf.ApprovalPDFData` bekommt drei Felder: `CooperativeSharesCount *int`, `CooperativeShareAmountCents *int64` (Snapshot vom Submit-Zeitpunkt) — der `RequiredShares`-Snapshot wird nicht im PDF benötigt, nur Count + Amount.
- [ ] Konditionale Sektion vor dem Status-Verlauf:
  ```
  GENOSSENSCHAFTSANTEILE
  Anzahl gezeichneter Anteile:  3
  Wert je Anteil:               100,00 €
  Gesamtbetrag:                 300,00 €
  ```
- [ ] Golden-Image-Test: PDF mit/ohne Anteile.
- [ ] **Snapshot-Verhalten:** beim Submit wird `cooperativeSharesCount` im Antrag gespeichert; der `share_amount_cents` ist KEIN Snapshot, sondern wird beim PDF-Render aus dem aktuellen `registration_entrypoint` gelesen. Bei späterer Änderung des Anteilswerts ändert sich also retroaktiv auch der dargestellte Wert in der Beitrittsbestätigung. *(Diskussionswürdig: könnte für eine echte Buchhaltung problematisch sein. Falls nötig: zusätzliche Spalte `application.cooperative_share_amount_cents_snapshot`. Für V1 verzichten wir darauf — EEG-Admins ändern den Anteilswert in der Praxis kaum nachträglich.)*

### Stage I: Doku

- [ ] `docs/domain-model.md`: vier neue Spalten beschreiben + Hinweis zur prospektiven Konfig-Wirkung.
- [ ] `docs/api-spec.md`: GET `/registration` + GET/PUT `/settings/eeg` + GET `/applications/{id}` um die neuen Felder erweitern.
- [ ] `docs/user-guide/06-admin-settings.md`: neuer Abschnitt „Genossenschaftsanteile" — Toggle, Pflichtmaß, Anteilswert; Hinweis dass die Info nicht in eegFaktura wandert und Änderungen nicht rückwirken.
- [ ] `CHANGELOG.md`: Eintrag.

## Geklärte Fragen

| Q | Antwort |
|---|---|
| Pflicht vs. optional im Formular | **Pflicht, min = `required_shares` aus Config.** Wer Feature an macht, muss `required_shares ≥ 1` setzen — `enabled=true ∧ required_shares=0` ist nicht zulässig. |
| Admin nachträglich editierbar | **Ja**, via Admin-Edit-Form mit identischer Min-Validierung gegen den aktuell konfigurierten Wert. |
| Konfig-Änderung rückwirkend? | **Nein**, prospektiv. Bestand wird nur informativ markiert wenn unter neuem Min. |
| Excel-Export | **Nicht enthalten.** |
| Core-Payload | **Nicht enthalten.** |
| Geld-Storage | **Integer-Cents (BIGINT).** |
| PDF | **Beitrittsbestätigung ja, SEPA-Mandat nein.** |
| Anteilswert-Snapshot pro Antrag | **Nein für V1** — PDF liest aktuellen Konfig-Wert. Bei Bedarf später nachrüstbar mit zusätzlicher Snapshot-Spalte. |

## Out of Scope

- Pflichtanteile pro Zählpunkt skaliert (z.B. 1 Anteil je Zählpunkt). Aktuell flach: 1 Mitglied = `required_shares` Anteile.
- Anteils-Verkauf / -Rückzahlung / Lifecycle nach Genehmigung.
- Genossenschaftsanteile als Position im SEPA-Mandat.
- Audit-Spur bei Konfig-Änderung der drei Settings (Standard-Edit-Save reicht).

## Realistische Implementations­dauer

~5–7 Stunden für alle 9 Stages inkl. Doku. Größter Aufwand: Public-Form-Integration (Locale-Number-Input + Live-Computation + Conditional Schema) und der PDF-Render.

## Pointer-Files

- Spec: `features/PROJ-37-cooperative-shares.md` (diese Datei)
- Verwandte Specs: PROJ-1, PROJ-8, PROJ-19, PROJ-21
- Backend-Entry-Points: `internal/application/application_service.go` (CreateApplication / SubmitApplication), `internal/application/registration_entrypoint_repo.go` (SaveEEGSettings)
- Frontend: `src/components/registration-form.tsx`, `src/components/admin-eeg-settings-editor.tsx`, `src/components/admin-application-detail.tsx`, `src/components/admin-edit-form.tsx`
- PDF: `internal/pdf/approval_pdf.go`
