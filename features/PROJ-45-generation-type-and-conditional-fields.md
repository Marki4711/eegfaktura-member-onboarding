# PROJ-45: Erzeugungsform + Batterie-Felder + typabhängige Sichtbarkeit

**Status:** Konzept (Diskussion offen)
**Created:** 2026-05-17

## Hintergrund

Drei zusammenhängende Anforderungen, die alle auf "EEG-Optimierung
durch bessere Vorab-Daten" zielen:

1. **Erzeugungsform** pro Erzeugungs-Zählpunkt erfassen
   (PV / Wasser / Wind / Biomasse), damit die EEG das Erzeugungsprofil
   einschätzen kann
2. **Batterie & Wechselrichter** pro Erzeugungs-Zählpunkt erfassen,
   damit vorhandene Speicher in die EEG-Optimierung einbezogen werden
   können
3. **Typabhängige Sichtbarkeit** der bestehenden Energie-Felder:
   Verbrauchsfelder nur wenn der Antrag mindestens einen CONSUMPTION-
   Zählpunkt hat, Erzeugungsfelder nur bei PRODUCTION-Zählpunkten

Punkt 3 zieht sich quer durch alle drei Bereiche — die neuen Felder
aus 1+2 sollen ebenfalls dieser Logik folgen ("nur anzeigen, wenn
PRODUCTION").

## 1. Erzeugungsform

### Datenmodell

Neue Spalte auf `metering_point`:

- `generation_type` VARCHAR(20) NULL — Werte: `pv` | `hydro` | `wind` |
  `biomass`. NULL bei CONSUMPTION-Zählpunkten (Service erzwingt das).
  Bei neuen PRODUCTION-Zählpunkten Default `pv` (häufigster Fall).

DB-Constraint: `CHECK (direction = 'CONSUMPTION' AND generation_type IS NULL)
OR (direction = 'PRODUCTION' AND generation_type IN ('pv','hydro','wind','biomass'))`
— bewusst harter Check, damit das Modell konsistent bleibt.

Bestandsdaten: alle PRODUCTION-Zählpunkte werden in der Migration auf
`'pv'` gesetzt (sicherer Default, häufigster Fall).

### UI

Im Public-Form pro Zählpunkt-Block:
- Wenn Richtung = PRODUCTION: zusätzliche Select-Box „Erzeugungsform"
  mit den vier Optionen, vorbelegt mit `pv`
- Wenn Richtung = CONSUMPTION: nichts anzeigen

Im Admin-Detail + Excel-Export + PDF: pro Erzeugungs-Zählpunkt
zusätzliche Spalte/Zeile mit dem Wert.

### Konfigurierbarkeit

**Bewusste Entscheidung:** `generation_type` ist **nicht**
PROJ-8-konfigurierbar — es ist ein Pflichtattribut jeder Erzeugungs-
Anlage, kein optionales EEG-spezifisches Feld. Default `pv` macht den
Aufwand für den Antragsteller minimal.

## 2. Batterie & Wechselrichter

### Datenmodell

Zwei neue Spalten auf `metering_point`:

- `battery_size_kwh` NUMERIC(7,2) NULL — Kapazität des Heimspeichers
  in kWh (z.B. `10.5`). Range bewusst großzügig (bis 99999.99).
- `inverter_manufacturer` VARCHAR(100) NULL — Freitext (z.B. „Fronius",
  „SMA", „Huawei"). Keine Auswahl-Liste — Hersteller-Landschaft ändert
  sich zu schnell, und Freitext liefert ausreichend Info zur EEG-
  Recherche.

Beide nur sinnvoll bei PRODUCTION-Zählpunkten (Service cleart bei
CONSUMPTION analog `clearEVDetailsIfDisabled`).

### Konfigurierbarkeit

PROJ-8-konfigurierbar (Default `hidden`):
- `battery_size_kwh` → Label „Größe Batterie (kWh)"
- `inverter_manufacturer` → Label „Hersteller Wechselrichter"

EEGs ohne Speicher-Programm können die Felder weglassen; EEGs mit
aktiver Batterie-Bewirtschaftung schalten sie auf `optional` oder
`required`.

### UI

Nur wenn (a) EEG-Konfig `!= hidden` UND (b) Zählpunkt-Richtung =
PRODUCTION. Sonst nicht rendern.

## 3. Typabhängige Sichtbarkeit (Querschnitt)

### Mapping Feld → benötigter Zählpunkt-Typ

| Bestehendes Feld | Benötigter Typ |
|---|---|
| `consumption_previous_year`     | CONSUMPTION |
| `consumption_forecast`          | CONSUMPTION |
| `heat_pump`                     | CONSUMPTION |
| `electric_vehicle`              | CONSUMPTION |
| `electric_vehicle_count`        | CONSUMPTION |
| `electric_vehicle_annual_km`    | CONSUMPTION |
| `electric_hot_water`            | CONSUMPTION |
| `persons_in_household`          | CONSUMPTION |
| `feed_in_forecast`              | PRODUCTION |
| `pv_power_kwp`                  | PRODUCTION |
| `generation_type` *(neu)*       | PRODUCTION (pro Zählpunkt) |
| `battery_size_kwh` *(neu)*      | PRODUCTION (pro Zählpunkt) |
| `inverter_manufacturer` *(neu)* | PRODUCTION (pro Zählpunkt) |

### Frontend-Verhalten

Im Public-Form:
- Live-Auswertung der eingegebenen Zählpunkte: hat der Antrag bereits
  mindestens einen CONSUMPTION/PRODUCTION-Zählpunkt?
- Application-Level-Felder werden zusätzlich zur EEG-Konfig nur dann
  gerendert, wenn der passende Typ vorhanden ist
- Zählpunkt-Level-Felder (`generation_type`, `battery_size_kwh`,
  `inverter_manufacturer`) werden direkt am Zählpunkt nach der
  Richtungs-Auswahl conditional gerendert

### Backend-Verhalten

- Service-Layer (analog `clearEVDetailsIfDisabled`): wenn kein
  passender Zählpunkt-Typ vorhanden ist, werden die zugehörigen
  Application-Level-Felder auf NULL gecleart
- `validateConfigurableRequiredFields`: required-Check für ein Feld
  feuert nur, wenn (a) EEG-Konfig = required UND (b) passender
  Zählpunkt-Typ vorhanden ist — analog zur PROJ-42-Sonderregel
  (EV-Detailfelder nur required wenn `electric_vehicle = true`)

### Edge-Case: Mitglied ändert Zählpunkt-Richtung nach Eingabe

Beispiel: Mitglied gibt 10.000 kWh Vorjahresverbrauch ein, ändert
dann den einzigen Zählpunkt von CONSUMPTION auf PRODUCTION → die
10.000 kWh werden serverseitig auf NULL gesetzt (clear-Pfad). Im
Frontend bleibt der Wert im React-State bis zum nächsten Re-Render —
beim Wechsel werden die Felder verborgen und somit ignoriert.

Bewusst keine Warnung im UI („Sie verlieren Daten!") — Komplexität
zu hoch für Edge-Case mit niedriger Eintrittswahrscheinlichkeit.

## Migration

`db/migrations/000040_generation_type_and_battery.up.sql`:
```sql
ALTER TABLE member_onboarding.metering_point
    ADD COLUMN generation_type        VARCHAR(20) NULL,
    ADD COLUMN battery_size_kwh       NUMERIC(7,2) NULL,
    ADD COLUMN inverter_manufacturer  VARCHAR(100) NULL;

UPDATE member_onboarding.metering_point
SET generation_type = 'pv'
WHERE direction = 'PRODUCTION';

ALTER TABLE member_onboarding.metering_point
    ADD CONSTRAINT metering_point_generation_type_check
    CHECK (
        (direction = 'CONSUMPTION' AND generation_type IS NULL)
        OR
        (direction = 'PRODUCTION' AND generation_type IN ('pv','hydro','wind','biomass'))
    );
```

## Field-Config-Registry erweitern

`knownConfigurableFields` (Backend) + `CONFIGURABLE_FIELDS` (Frontend):
- `battery_size_kwh` → defaultState `hidden`
- `inverter_manufacturer` → defaultState `hidden`

`generation_type` wird **nicht** in die Registry aufgenommen — es ist
fix sichtbar/required für jeden PRODUCTION-Zählpunkt.

## Export / Mail

- **Excel-Export:** drei neue Spalten am Ende der Zählpunkt-Zeile
  (`Erzeugungsform`, `Größe Batterie (kWh)`, `Hersteller WR`). Achtung:
  eegFaktura-Importer ignoriert unbekannte Spalten am Ende — daher
  kein Risiko für den Import-Prozess.
- **Approval-PDF:** pro Erzeugungs-Zählpunkt zusätzliche Zeile
  „Erzeugung: PV, Speicher 10 kWh (Fronius)"
- **Mail (Member + EEG):** in der Zählpunkt-Liste pro Production-MP
  eine Zusatzzeile (gleiches Format wie PDF)

## Out of Scope

- Per-Hersteller-Anbindung (Modell, Kapazität, etc.) — Freitext reicht
- Batterie-Leistung in kW (zusätzlich zur kWh-Kapazität) — falls
  später nötig, separate Spalte
- Validierung „Batterie nur bei PV" — auch Wind/Wasser können Speicher
  haben, kein Block
- Verschieben/Konvertieren bestehender Verbrauchs-Felder auf
  Zählpunkt-Ebene — würde die Migration aufblasen, YAGNI

## Tests

- Build muss grün bleiben
- Migration: Bestandsdaten — alle PRODUCTION-Zählpunkte erhalten `pv`
- Smoke-Test 1: Antrag nur CONSUMPTION → Erzeugungsfelder werden nicht
  angezeigt, weder Application-Level (pv_power_kwp) noch
  Zählpunkt-Level (generation_type)
- Smoke-Test 2: Antrag nur PRODUCTION → Verbrauchsfelder werden nicht
  angezeigt, generation_type ist Pflicht (Default `pv`)
- Smoke-Test 3: Antrag gemischt → beide Felder-Gruppen sichtbar
- Server-Validierung: required-Check für Verbrauchsfelder triggert
  nicht bei reinen PRODUCTION-Anträgen, auch wenn EEG-Konfig =
  required

## Offene Fragen vor Implementierung

1. **Generation_type-Pflicht bei Bestandsanträgen:** alle bestehenden
   PRODUCTION-Zählpunkte werden migrationsweise auf `pv` gesetzt.
   Falls EEGs das vorab korrigieren wollen, brauchen wir einen
   Admin-Edit-Pfad. → Vorschlag: per Admin-Detail-Edit-Form mit
   einem zusätzlichen Select pro Zählpunkt. Hinzufügen?
2. **Batterie-Felder Sichtbarkeitslogik:** nur bei `generation_type=pv`
   anzeigen, oder bei allen PRODUCTION-Typen? → Spec schlägt „alle
   PRODUCTION" vor (Wind-Anlagen können auch Speicher haben).
3. **Wechselrichter-Hersteller bei Wasser/Wind/Biomasse:** semantisch
   nicht immer passend (Wasserkraft hat keinen Wechselrichter im
   PV-Sinn). → Vorschlag: Feld trotzdem zeigen, Mitglied kann
   leer lassen.

## Empfohlene Aufteilung

Eine Implementierung in **einem** Feature-PR ist akzeptabel
(zusammenhängender Scope, dieselben Touch-Points). Alternative
Aufteilung wäre PROJ-45 = Erzeugungsform allein, PROJ-46 = Batterie,
PROJ-47 = typabhängige Sichtbarkeit — würde aber drei DB-Migrations
für ein zusammenhängendes Thema bedeuten. **Empfehlung: 1 PR.**
