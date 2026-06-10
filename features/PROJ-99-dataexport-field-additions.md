# PROJ-99: Datenweiterleitung — fehlende Felder im Field-Picker

## Status: Deployed (2026-06-10)
**Created:** 2026-06-10
**Last Updated:** 2026-06-10
**Typ:** Feature-Add (Tester-Befund Dani 2026-06-10)

## Hintergrund

Tester-Befund Dani Strasser 2026-06-10 17:09:

> „Was auch nicht zur Auswahl steht, sind Mandatsreferenz (ja, wir
> sind zwanghaft und sammeln alle Infos an einem Ort), Jahresverbrauch,
> PV-Leistung, Speicher ja/nein/Größe – das ganze zusätzlich erfragte
> Gedöns halt."

Code-Audit bestätigt: vier Application-/Metering-Point-Felder fehlen
im Excel-Export-Field-Picker. Alle Daten sind in der DB vorhanden +
in den Repo-Layern korrekt geladen — nur die `fields.go`-Registry hat
die Einträge nicht.

## Scope

Vier neue Field-Definitionen in
`internal/dataexport/excel/fields.go`:

### Application-Level (1 Feld, direkter Extract)

| Field-Key | Label | Type | Quelle |
|---|---|---|---|
| `mandate_reference` | „Mandatsreferenz" | text | `application.mandate_reference` |

### Metering-Point-Aggregate (3 Felder + 1 Bonus)

Application-1:N-Metering-Points → Aggregation nötig. Owner-Direktive
**SUM** über alle Zählpunkte (passt zu „Gesamtjahresverbrauch",
„Gesamt-PV-Leistung", „Gesamt-Speichergröße"). NULL-Werte werden
beim Aggregieren übersprungen, leeres Ergebnis bleibt leer.

| Field-Key | Label | Type | Aggregation |
|---|---|---|---|
| `consumption_previous_year_sum` | „Jahresverbrauch (Summe)" | number | `SUM(consumption_previous_year)` |
| `pv_power_kwp_sum` | „PV-Leistung (Summe, kWp)" | number | `SUM(pv_power_kwp)` |
| `battery_size_kwh_sum` | „Speicher-Größe (Summe, kWh)" | number | `SUM(battery_size_kwh)` |
| `has_battery` | „Speicher vorhanden" | bool | `ANY(battery_size_kwh > 0)` |

Kategorie für alle vier: **Zählpunkte (aggregiert)** — separat von
der bestehenden „Zählpunkte"-Gruppe, damit Admin den Aggregat-Modus
visuell unterscheidet.

## Acceptance Criteria

- [x] **AC-1** Field-Definitionen registriert; Field-Picker in der UI
  zeigt sie unter neuen / bestehenden Kategorien.
- [x] **AC-2** SUM-Aggregation überspringt NULL-Werte; wenn alle
  Zählpunkte das Feld NULL haben → leere Excel-Zelle, nicht „0".
- [x] **AC-3** `has_battery` rendert „Ja" wenn mind. ein Zählpunkt
  `battery_size_kwh > 0` hat, sonst „Nein".
- [x] **AC-4** Bestand-Configs der Admins bleiben gültig — keine
  Migration nötig.
- [x] **AC-5** `go build ./...` clean, `go test ./...` clean,
  Field-Registry-Tests greifen.
- [x] **AC-6** CHANGELOG.md + INDEX.md-Eintrag im selben Commit.

## Edge Cases

- **EC-1** Antrag ohne Zählpunkte: alle SUMs → NULL → leere Excel-
  Zelle; `has_battery` → „Nein".
- **EC-2** Antrag mit Mix CONSUMPTION/PRODUCTION-Zählpunkten: SUM
  geht über alle Zählpunkte mit Nicht-NULL-Wert. Das gewünschte
  Verhalten ist „Gesamtsumme egal welche Direction" — der Admin
  weiß, was er aggregiert.
- **EC-3** `mandate_reference` NULL: leere Excel-Zelle. Wenn Admin
  manuell überschrieben hat (PROJ-95-Pfad), zeigt Excel den Override.
- **EC-4** Bestand-Antrag vor PROJ-49 (Energie-Felder eingeführt):
  alle MP-Werte NULL → SUM bleibt leer. By-design.

## Out of Scope

- Metering-Point-Pro-Zeile-Export (eine Zeile je Zählpunkt statt
  Aggregation): eigene Spec, würde das Spalten-Modell sprengen.
- Berechnete Felder (z. B. „Produktion/Verbrauch-Verhältnis"):
  Folge-Spec.
- Befunde A, B, D, E aus Dani's Liste — separate Items in
  `project_tester_feedback_2026-06-10_dani_dataexport`.

## Tech Design

Keine Schema-Änderung, keine Migration. Reines Code-Add in
`fields.go` + Test-Erweiterung in `fields_test.go`.

Helper-Funktionen für SUM-Aggregation neu in `fields.go`:

- `sumInt64(metering []shared.MeteringPoint, pick func(mp) *int64) interface{}`
- `sumFloat64(metering []shared.MeteringPoint, pick func(mp) *float64) interface{}`

Beide returnen `nil` wenn alle Picks NULL liefern, sonst die Summe.
NULL-Handling kompatibel zum bestehenden `Extract`-Pattern, wo
Excel-Renderer leere Strings für `nil` rendert.

## Deployment

**Deploy-Bookkeeping 2026-06-10 (abends):**

- Feature-Add; kein Schema-Change, keine Helm-Änderung.
- Code-Commit: `997d93f`
- Helm-Bump-Commit: `d6c94f0` (sha-997d93f)
- Tag: `v1.26.1-PROJ-99` gesetzt + gepusht.

**Tester-Verifikation:** Dani konfiguriert eine Spalte „Jahresverbrauch
(Summe)", speichert, exportiert. In der Excel-Spalte steht der
SUM-Wert aller Zählpunkte des Antrags.
