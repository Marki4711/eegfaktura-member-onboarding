# PROJ-37: Genossenschaftsanteile (per-EEG-Konfiguration + Antragsfeld)

## Status: Planned
**Created:** 2026-05-15
**Last Updated:** 2026-05-15

## Dependencies
- Berührt: PROJ-1 (Public Registration — neues Formular-Feld), PROJ-2 (Admin Review — Anzeige im Antrags-Detail), PROJ-19 (Manuelle Aktivierung — Settings-Seite), PROJ-21 (Beitrittsbestätigung-PDF — neue Zeile im Daten-Block).
- Nicht berührt: Core-Import (PROJ-4) und Excel-Export (PROJ-17) — Genossenschaftsanteile bleiben **rein im Member-Onboarding**, eegFaktura hat keinen Platz dafür.

## Hintergrund

Je nach Rechtsform des Rechtsträgers einer EEG (Genossenschaft) muss bei der Registrierung erfasst werden, wieviele Genossenschaftsanteile das Mitglied zeichnet. Der Betrag pro Anteil ist eine satzungsmäßige Größe und wird vom EEG-Admin in den Einstellungen hinterlegt; das Mitglied gibt im Formular nur die Anzahl an. Der Gesamtbetrag ergibt sich aus `Anzahl × Betrag` und wird im Formular live, im Antrag und in der Beitrittsbestätigung als berechneter Wert ausgewiesen.

EEGs ohne Genossenschaftsstruktur (Verein, GmbH, …) sehen das Feld nicht — es ist per-EEG abschaltbar (Default: aus).

## User Stories

- Als **EEG-Admin einer Genossenschaft** möchte ich in den Einstellungen aktivieren, dass Mitglieder die Anzahl der gezeichneten Anteile angeben müssen, sowie den Betrag pro Anteil hinterlegen.
- Als **EEG-Admin einer Nicht-Genossenschaft** möchte ich, dass dieses Feld in meinem Registrierungsformular gar nicht erscheint — Default-Verhalten.
- Als **neues Mitglied einer Genossenschafts-EEG** möchte ich beim Ausfüllen die Anzahl der Anteile angeben und sofort sehen, was sich daraus an Gesamtsumme ergibt.
- Als **EEG-Admin** möchte ich im Antrags-Detail die Anteilsinfo sehen (Anzahl × Betrag = Gesamt) und gegebenenfalls korrigieren können, ohne das Mitglied über den `needs_info`-Flow zu schicken.
- Als **EEG-Admin** möchte ich, dass die Anteilsinfo in der **Beitrittsbestätigung** als eigene Zeile ausgewiesen ist — als Beleg für die Buchhaltung.

## Architekturentscheidungen

1. **Datenmodell minimal**:
   - Per-EEG-Konfiguration auf `registration_entrypoint`: `cooperative_shares_enabled BOOLEAN` + `cooperative_share_amount_cents BIGINT NULL`.
   - Per-Antrag-Wert auf `application`: `cooperative_shares_count INT NULL`.
   - **Der Gesamtbetrag wird NICHT gespeichert** — `count × amount_cents` ist eine reine Render-Berechnung. Doppelte Speicherung wäre eine Drift-Quelle (falls Admin den Betrag pro Anteil später ändert).

2. **Geld als Integer-Cents.** `BIGINT` in der DB, `int64` in Go, divide-by-100 nur in der Darstellung. Keine Float-Arithmetik, keine Rundungsprobleme. Konvention für künftige Geld-Spalten.

3. **Nicht in Excel-Export, nicht in Core-Payload.** Bewusste Entscheidung (User-Vorgabe 2026-05-15): eegFaktura hat keine Spalte für Genossenschaftsanteile, also fließt der Wert dort nirgends hin. Excel-Export-Format bleibt unverändert.

4. **Aktivierung erfordert Betrag.** Wenn das Toggle auf TRUE gesetzt wird, muss `cooperative_share_amount_cents > 0` sein. Server-seitige Validierung in `SaveEEGSettings`, sonst 400 mit klarer Fehlermeldung. Im UI wird das Eingabefeld zwingend mit-validiert.

5. **Pflichtfeld-Semantik im Formular (TBD — siehe Q1).** Default-Annahme: wenn EEG-seitig aktiviert, ist die Eingabe Pflicht und muss `≥ 1` sein. Falls Q1 anders entschieden wird, anpassbar.

6. **Admin-Korrektur erlaubt (TBD — siehe Q2).** Default-Annahme: der Admin kann den Wert über das bestehende Admin-Edit-Form ändern, ohne den Antrag in `needs_info` schicken zu müssen. Standard-Audit-Trail über `status_log` greift, da bei jeder Status-Änderung ein Eintrag entsteht — für reine Daten-Korrekturen gibt es aktuell keinen Audit-Trail (entspricht dem Verhalten bei anderen Admin-Edits).

## Synced Fields (Zusammenfassung)

| Wo | Spalte | Typ | Wer setzt |
|---|---|---|---|
| `registration_entrypoint` | `cooperative_shares_enabled` | BOOLEAN NOT NULL DEFAULT FALSE | EEG-Admin (Settings) |
| `registration_entrypoint` | `cooperative_share_amount_cents` | BIGINT NULL (NULL wenn disabled) | EEG-Admin (Settings) |
| `application` | `cooperative_shares_count` | INT NULL | Mitglied (Submit) bzw. Admin (Korrektur) |

## Acceptance Criteria

### Stage A: DB-Migration

- [ ] `db/migrations/000035_cooperative_shares.up.sql`:
  ```sql
  ALTER TABLE member_onboarding.registration_entrypoint
      ADD COLUMN cooperative_shares_enabled BOOLEAN NOT NULL DEFAULT FALSE,
      ADD COLUMN cooperative_share_amount_cents BIGINT NULL
      CHECK (cooperative_share_amount_cents IS NULL OR cooperative_share_amount_cents > 0);
  ALTER TABLE member_onboarding.application
      ADD COLUMN cooperative_shares_count INT NULL
      CHECK (cooperative_shares_count IS NULL OR cooperative_shares_count >= 0);
  ```
- [ ] `.down.sql` droppt die drei Spalten.
- [ ] Bestehende `SELECT *`-Queries — keine: die Repo-Methoden listen Spalten explizit.

### Stage B: Backend

- [ ] `shared.RegistrationEntrypoint` bekommt `CooperativeSharesEnabled bool` + `CooperativeShareAmountCents *int64`.
- [ ] `shared.Application` bekommt `CooperativeSharesCount *int`.
- [ ] `RegistrationEntrypointRepository.GetByRCNumber` + `SaveEEGSettings`:
  - Get liest die zwei neuen Spalten mit.
  - Save bekommt Parameter `cooperativeSharesEnabled bool, cooperativeShareAmountCents *int64`. Validierung: wenn enabled=true, dann amount muss `> 0` sein (sonst ValidationError mit Feld-Map `{"cooperativeShareAmountCents": "Betrag je Anteil ist erforderlich, wenn die Anteilsregistrierung aktiviert ist"}`).
- [ ] `ApplicationRepository` Create/Update: schreiben/lesen `cooperative_shares_count`.
- [ ] `ApplicationService.SubmitApplication`: wenn EEG `cooperative_shares_enabled=true`, dann `app.CooperativeSharesCount` Pflicht. Validierungsregel siehe Q1 (Default: `≥ 1`).
- [ ] **Nicht-Berührungspunkte**:
  - `internal/excel/generator.go`: keine Änderung.
  - `internal/importing/payload.go`: keine Änderung.

### Stage C: Public-Registration-Config-Endpoint

- [ ] `GET /api/public/registration/{rc_number}` liefert zusätzlich:
  ```json
  "cooperativeSharesEnabled": true,
  "cooperativeShareAmountCents": 5000
  ```
  Wenn enabled=false, Amount weglassen (omitempty).

### Stage D: Public-Form-Frontend

- [ ] Wenn `config.cooperativeSharesEnabled=true`, neuer Card-Block **„Genossenschaftsanteile"** zwischen Zählpunkten und Bankverbindung (Platzierung im Spec-Visual unten).
- [ ] Input „Anzahl Anteile *" (number, min nach Q1, max 1000 als Plausibilitäts-Cap).
- [ ] Read-only Anzeige „Betrag je Anteil: 50,00 €" (formatiert nach `de-AT` mit `Intl.NumberFormat`).
- [ ] Live-Computation „Gesamtbetrag: 150,00 €" — `useMemo` über `watch("cooperativeSharesCount") * amount / 100`.
- [ ] Zod-Schema kennt das neue Feld; bei deaktivierter EEG wird das Feld vom Schema komplett ignoriert.

### Stage E: Admin-Settings-UI

- [ ] In `admin-eeg-settings-editor.tsx`: neuer Abschnitt **„Genossenschaftsanteile"** mit:
  - Toggle „Genossenschaftsanteile erfassen"
  - Wenn aktiv: Input „Betrag je Anteil (€)" — Pflicht, mit `de-AT`-Locale, Decimal-Input. Frontend konvertiert in Cents vor dem Submit.
  - Hint: „Wird auf der Beitrittsbestätigung als Anzahl × Betrag = Gesamt ausgewiesen. Wird **nicht** an eegFaktura übermittelt — eine reine Onboarding-Erfassung."
- [ ] Validierung clientseitig: wenn Toggle aktiv, Betrag-Input nicht leer und `> 0,00 €`.

### Stage F: Admin-Antrags-Detail

- [ ] `AdminApplicationDetailResponse` enthält `cooperativeSharesCount`; das passende `cooperativeShareAmountCents` wird beim Detail-Load aus dem Entrypoint nachgeschlagen und in die Response gehängt (Schnittstelle: separat oder unter einem neuen Objekt `cooperativeShares: { count, amountCents, totalCents }`).
- [ ] `admin-application-detail.tsx`: konditionale Zeile in den Mitgliedsdaten oder eigene mini-Box „Genossenschaftsanteile: **3** × 50,00 € = **150,00 €**".
- [ ] Admin-Edit-Form (`admin-edit-form.tsx`): falls Q2 = ja, neues Eingabefeld zur Korrektur.

### Stage G: Beitrittsbestätigungs-PDF

- [ ] `pdf.ApprovalPDFData` bekommt drei Felder `CooperativeSharesCount *int`, `CooperativeShareAmountCents *int64`, plus eine konditionale Render-Funktion die die Zeile als
  > Genossenschaftsanteile: 3 × 50,00 € = 150,00 €
  
  in den Daten-Block einfügt (vor oder nach dem SEPA-Block — Position spec-final).
- [ ] Golden-Image-Test: PDF mit/ohne Anteile.

### Stage H: Doku

- [ ] `docs/domain-model.md`: drei neue Spalten beschreiben.
- [ ] `docs/api-spec.md`: GET /registration response + GET/PUT /settings/eeg + GET /applications/{id} um die neuen Felder erweitern.
- [ ] `docs/user-guide/06-admin-settings.md`: neuer Abschnitt „Genossenschaftsanteile" — Toggle erklären, Betrag-pro-Anteil-Eingabe, Hinweis dass die Info nicht in eegFaktura wandert.
- [ ] `CHANGELOG.md`: Eintrag.

## Open Questions

### Q1: Pflichtfeld-Semantik im Formular?
**Default-Annahme: Pflicht, min=1.** Mitglied muss mindestens einen Anteil zeichnen, sonst keine Mitgliedschaft (typisches Genossenschafts-Modell). Alternativen:
- Variante B: Pflicht, min=0 — Anteil ist optional, aber „kein Anteil" ist eine bewusste Eingabe.
- Variante C: Komplett optional, NULL erlaubt — leere Eingabe = nicht erhoben.

**TBD vom User.** Bis dahin implementieren wir Variante A.

### Q2: Soll der Admin den Wert nachträglich ändern können?
**Default-Annahme: ja.** Über das bestehende Admin-Edit-Form (`AdminUpdateApplication`). Standard-Audit gilt — eine eigene Sub-Audit-Spur ist Overkill für ein einzelnes Feld. Alternativen:
- Nur via `needs_info`-Flow (umständlicher).
- Eigener Audit-Eintrag bei Änderung (Overengineering für V1).

**TBD vom User.**

### Q3: Beispiel-Werte als Defaults?
Soll der Betrag-Input einen Standard-Vorschlag haben (z.B. „100,00 €")? Empfehlung: leer lassen, der EEG-Admin gibt aktiv den satzungsmäßigen Wert ein. Kein Pre-Fill.

### Q4: Untergrenze für Betrag je Anteil?
DB-Check ist `> 0`. Im UI kann man ein sinnvolles Mindestmaß setzen (z.B. `≥ 1,00 €` = 100 Cent), damit niemand versehentlich 1 Cent eingibt. Empfehlung: keine harte Untergrenze über `> 0` hinaus — Admin weiß was er tut.

### Q5: PDF-Position?
Empfehlung: zwischen dem Bankverbindungs-Block und dem Status-Verlauf, als eigene Mini-Sektion „Beitrag". Falls die EEG das Mandat aktiviert hat, könnte sich der Block dort anschließen, sonst eigenständig stehen.

## Out of Scope

- Genossenschaftsanteile als wiederkehrende Position im SEPA-Mandat.
- Anteils-Korrekturen mit eigener Audit-Spur (nutzt das Standard-Edit-Verhalten).
- Verkauf oder Rückzahlung von Anteilen (Lifecycle nach Genehmigung).
- Excel-Export mit eigenen Spalten — bewusst nicht, weil eegFaktura keinen Empfänger dafür hat.
- Übertragung in eegFaktura-Core via `POST /participant`.

## Realistische Implementationsdauer
~4–6 Stunden für alle 8 Stages, inkl. Doku. Größter Aufwand: Public-Form-Integration (Locale-aware Number-Input + Live-Computation + Schema-Conditional) und der PDF-Render.

## Pointer-Files

- Spec: `features/PROJ-37-cooperative-shares.md` (diese Datei)
- Verwandte Specs: PROJ-1, PROJ-8, PROJ-19, PROJ-21
- Backend-Entry-Points: `internal/application/application_service.go` (CreateApplication / SubmitApplication), `internal/application/registration_entrypoint_repo.go` (SaveEEGSettings)
- Frontend: `src/components/registration-form.tsx`, `src/components/admin-eeg-settings-editor.tsx`, `src/components/admin-application-detail.tsx`, `src/components/admin-edit-form.tsx`
- PDF: `internal/pdf/approval_pdf.go`
