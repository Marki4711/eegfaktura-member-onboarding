# PROJ-49: Energie-Felder pro Zählpunkt (Refactoring + Einspeiselimit)

**Status:** In Review
**Created:** 2026-05-17
**Last Updated:** 2026-05-17 (Backend + Frontend + Doku umgesetzt)

## Hintergrund

Bei der Implementierung von PROJ-45 (typabhängige Sichtbarkeit) wurde sichtbar, dass mehrere Energie-Felder konzeptionell **pro Zählpunkt** gehören, aktuell aber auf **Antrags-Ebene** liegen. Das führt zu unsauberer Datenstruktur:

- Hat ein Antrag mehrere Verbrauchs-Zählpunkte (z. B. Wohnung + Werkstatt), wird der eine `consumption_previous_year`-Wert beliebig zugeordnet.
- Hat ein Antrag mehrere Einspeise-Zählpunkte (z. B. PV + Wasserkraft), gilt das gleiche für `pv_power_kwp` und `feed_in_forecast`.
- Das Feld `pv_power_kwp` ist sogar an Erzeugungsform = `pv` gekoppelt — gehört also auf eine PRODUCTION-Zeile mit `generation_type='pv'`.

Zusätzlich fehlt das praxisrelevante Feld **Einspeiselimit**: manche Netzanschlüsse sind leistungstechnisch beschränkt (z. B. „nur 70 % der PV-Leistung einspeisbar"), das ist für die EEG-Planung wichtig zu wissen.

## Scope

### 1. Felder, die von `application` auf `metering_point` wandern

| Feld | Bisheriger Scope | Neuer Scope | Bedingung im Frontend |
|---|---|---|---|
| `consumption_previous_year` | Antrag (NULL) | Zählpunkt | nur CONSUMPTION |
| `consumption_forecast` | Antrag (NULL) | Zählpunkt | nur CONSUMPTION |
| `feed_in_forecast` | Antrag (NULL) | Zählpunkt | nur PRODUCTION |
| `pv_power_kwp` | Antrag (NULL) | Zählpunkt | nur PRODUCTION + `generation_type='pv'` |

Jeder Zählpunkt bekommt seine eigenen Werte — bei mehreren passenden Zählpunkten muss das Mitglied entsprechend mehrfach Eingaben machen.

### 2. Neues Feld: Einspeiselimit (PV)

Zwei neue Spalten auf `metering_point`:

- `feed_in_limit_present` BOOLEAN NULL — "Einspeiselimit vorhanden?" (Default-Eingabe: leer/Nein)
- `feed_in_limit_kw` NUMERIC(7,2) NULL — Wert in kW, nur bedeutsam wenn `feed_in_limit_present = TRUE`

Frontend: Auswahl „Einspeiselimit vorhanden? Ja/Nein". Bei Ja erscheint Eingabefeld kW. Bedingung: nur PRODUCTION + `generation_type='pv'`.

DB-Regel (Service-Layer, nicht CHECK):
- Bei `direction='CONSUMPTION'` oder `generation_type != 'pv'`: Service setzt beide Felder auf NULL
- Bei `feed_in_limit_present=FALSE/NULL`: Service setzt `feed_in_limit_kw` auf NULL

### 3. Bestandsdaten

**Entscheidung: Werte werden verworfen** (Option 2c, abgestimmt mit Owner 2026-05-17).

Begründung: Aktuell sind nur In-Review-Anträge betroffen, keine Deployed-Daten mit echten Mitgliedern. Migration ist deshalb riskoarm. Die zusätzliche Komplexität einer "greedy-auf-ersten-passenden-MP"-Migration lohnt nicht.

Migration:
1. Neue Spalten auf `metering_point` anlegen (alle NULL)
2. Alte Spalten auf `application` droppen (`consumption_previous_year`, `consumption_forecast`, `feed_in_forecast`, `pv_power_kwp`)
3. Alte `field_config`-Einträge mit den 4 Namen löschen (`DELETE FROM member_onboarding.field_config WHERE field_name IN (...)`)

### 4. Configurable-Fields (PROJ-8 + PROJ-45)

Entscheidung 4a: gleiche Feldnamen ohne Prefix, jetzt im MP-Scope. Begründung: bestehende MP-Felder (`transformer`, `battery_size_kwh`, `inverter_manufacturer`, `generation_type`) haben auch keinen Prefix — `field_config` ist ein flacher Namespace, Scope-Trennung erfolgt im Code über separate Validierungs-Funktionen.

Neue MP-Feldnamen in `field_config`:
- `consumption_previous_year` (CONSUMPTION-Badge)
- `consumption_forecast` (CONSUMPTION-Badge)
- `feed_in_forecast` (PRODUCTION-Badge)
- `pv_power_kwp` (PRODUCTION + PV-Badge)
- `feed_in_limit_kw` (PRODUCTION + PV-Badge) — `feed_in_limit_present` ist immer sichtbar wenn `pv_power_kwp` sichtbar; kein eigener Toggle

Default-State: `hidden` für alle 5. Jede EEG aktiviert bewusst, was sie braucht.

### 5. Aus dem Application-Scope entfernt

In der Admin-Settings-UI (PROJ-45 Badges-Block) erscheinen diese 4 Feldnamen nicht mehr als Application-Scope — sie wandern in die Sektion „Zählpunkt-Felder".

## API-Änderungen

### `POST /api/public/applications` Request

Felder `consumptionPreviousYear`, `consumptionForecast`, `feedInForecast`, `pvPowerKwp` werden aus dem Top-Level-Body **entfernt** und stattdessen pro `meteringPoints[]`-Eintrag akzeptiert:

```json
"meteringPoints": [
  {
    "meteringPoint": "AT00310...",
    "direction": "CONSUMPTION",
    "consumptionPreviousYear": 4200,
    "consumptionForecast": 4000
  },
  {
    "meteringPoint": "AT00310...",
    "direction": "PRODUCTION",
    "generationType": "pv",
    "feedInForecast": 6000,
    "pvPowerKwp": 9.9,
    "feedInLimitPresent": true,
    "feedInLimitKw": 7.0
  }
]
```

### `GET /api/admin/applications/{id}` Response

Felder wandern analog in das `meteringPoints[]`-Array. Top-Level enthält sie nicht mehr.

### `PUT /api/admin/applications/{id}` Body

Top-Level akzeptiert die 4 Felder nicht mehr (silently ignored — kein 400). MP-Replacement nimmt sie aus den MP-Einträgen entgegen.

## Frontend-Änderungen

- **`metering-point-fields.tsx`**: 5 neue Felder mit Sichtbarkeitsbedingungen (siehe Tabelle). Bei `generation_type` ≠ `pv` werden `pv_power_kwp`, `feed_in_limit_present`, `feed_in_limit_kw` ausgeblendet. `feed_in_limit_kw` nur sichtbar, wenn `feed_in_limit_present = true`.
- **`registration-form.tsx`**: Energie-Sektion auf Application-Ebene wird entrümpelt — die 4 Felder verschwinden, übrig bleiben dort nur `heat_pump`, `electric_vehicle` (+ EV-Details PROJ-42), `electric_hot_water`, `persons_in_household`, `membership_start_date`.
- **`admin-edit-form.tsx`**: Spiegelt die neuen MP-Felder.

## Mail-Templates + Approval-PDF

- `application_submitted_eeg.html` + `application_imported_eeg.html`: Energie-Details werden in der MP-Tabelle pro Zählpunkt ausgegeben, nicht mehr im Antrag-Block.
- Beitrittsbestätigungs-PDF: gleicher Umzug, pro Zählpunkt eine Zeile mit den passenden Werten.

## Import in eegFaktura Core

`docs/import-mapping.md` prüfen: bisher gingen `consumption_previous_year`, `feed_in_forecast`, etc. als Top-Level zum Core. Nach diesem Refactoring muss die Übergabe aggregiert oder pro MP erfolgen — Mapping wird in der Implementierung an den tatsächlichen Core-Endpunkt angepasst.

## Acceptance Criteria

1. Migration läuft auf einer leeren + auf der bestehenden Test-DB sauber durch (alte Spalten weg, neue Spalten da, field_config bereinigt).
2. Public-Form rendert die 5 Felder pro MP-Eintrag mit den richtigen Sichtbarkeitsbedingungen.
3. Admin sieht alle 5 Felder in der EEG-Einstellungen-Seite unter „Zählpunkt-Felder" mit den passenden Badges (Verbraucher / Einspeisung / PV).
4. Submit funktioniert mit gemischten MPs (1× CONSUMPTION + 1× PRODUCTION+PV mit Limit, 1× PRODUCTION+Wind ohne Limit).
5. Admin-Edit-Form kann die Werte pro MP ändern und speichern.
6. Approval-PDF zeigt die Werte pro Zählpunkt.
7. `go build ./...` + `go test ./...` grün, Frontend `npm run build` + `npm test` grün.

## Out of Scope

- Migration von Bestandsdaten (Werte werden verworfen — Entscheidung 2c).
- Aggregierte „Antragssumme" auf Application-Ebene (kann später hinzukommen, falls UI das braucht).

## Folge-Aktionen (nicht Teil dieses Commits)

Vier User-Guide-Screenshots sind nach PROJ-49 inhaltlich veraltet — neue Eingabefelder sind im Zählpunkt-Block sichtbar, der allgemeine Bereich der Doku zeigt sie nicht mehr. Beim nächsten Screenshot-Refresh (`npm run screenshots`) aktualisieren:

- `docs/user-guide/images/register-form-metering-points.png`
- `docs/user-guide/images/register-form-start.png`
- `docs/user-guide/images/admin-settings-fields.png`
- `docs/user-guide/images/admin-application-detail-1.png` / `…-2.png`
