# PROJ-42: E-Fahrzeug-Detailerfassung (Anzahl + Jahres-km)

**Status:** Deployed
**Created:** 2026-05-17

## Hintergrund

Das bestehende Feld `electric_vehicle` ist ein boolescher „ja/nein". Für
die EEG-Lastprofil-Optimierung sind aber konkrete Mengen interessant:
Anzahl der E-Fahrzeuge und gefahrene Jahres-Kilometer (=> Schätzung des
Ladestrom-Bedarfs).

## Datenmodell

Zwei neue optionale Felder auf `application`:

- `electric_vehicle_count` INT NULL — Anzahl der E-Fahrzeuge
- `electric_vehicle_annual_km` INT NULL — geschätzte Jahres-Kilometer
  (Gesamtsumme aller E-Fahrzeuge des Haushalts)

Beide sind **nur relevant** wenn `electric_vehicle = TRUE`. Wenn der Wert
des Mitglieds für E-Auto auf „nein" steht, werden die beiden Felder
serverseitig ignoriert (auf NULL gesetzt).

Migration: `db/migrations/000038_ev_details.up.sql`.

## Feldkonfiguration (PROJ-8-Pattern)

Wie die bestehenden Energie-Felder pro EEG einstellbar:

- `electric_vehicle_count` — `defaultState: "hidden"`, Label „Anzahl
  E-Fahrzeuge"
- `electric_vehicle_annual_km` — `defaultState: "hidden"`, Label
  „Jahres-Kilometer (E-Fahrzeuge)"

EEGs, die diese Felder nicht sammeln möchten, lassen sie auf `hidden`.

## UI-Verhalten

Im Public-Form:
- Beide Felder werden **nur** gerendert, wenn (a) die EEG sie als
  `optional` oder `required` konfiguriert hat UND (b) das Mitglied
  `electric_vehicle = true` angekreuzt hat
- Bei Deaktivierung des E-Auto-Häkchens werden beide Werte gecleart
- Beide Inputs sind `type="number"`, min=1 für count, min=0 für km

## Anzeige / Export

- Mail (Member + EEG): die zwei Werte erscheinen im „Zusätzliche
  Informationen"-Block — via `buildConfigurableFields` automatisch,
  sobald die Felder konfiguriert sind
- Approval-PDF: dito
- Excel-Export: dito (configurable-field-Pfad)
- Admin-Detail: dito

## Out of Scope

- Per-Vehicle-Daten (Marke/Modell/Akku-kWh) — bewusst nicht V1
- Differenzierung Benzin/Hybrid/etc. — out
- Aggregierte EEG-Reports — wird ggf. separater Spec

## Tests

- Build muss grün bleiben
- Smoke-Test: EEG mit den beiden Feldern auf `optional` konfigurieren,
  Antrag mit E-Auto + Anzahl + km einreichen, dann Mail + PDF + Excel
  prüfen
