# PROJ-113: Datenweiterleitung – Haushalts-/Verbrauchs-Felder ergänzen

## Status: Deployed
**Created:** 2026-06-18
**Last Updated:** 2026-06-18

## Dependencies
- Requires: PROJ-60 (Datenweiterleitung) — Feld-Katalog. Verwandt: PROJ-99 (gleiche Klasse: fehlende Export-Felder nachziehen), PROJ-42 (E-Fahrzeug-Detailerfassung), PROJ-8/15 (konfigurierbare Felder).

## Problem
Tester-Befund 2026-06-18: „E-Auto + Wärmepumpe fehlen bei Weiterleitung für Excel." Die Haushalts-/Verbrauchs-Felder existieren am Antrag (Public-Form, konfigurierbar), waren aber **komplett nicht** im Datenweiterleitungs-Feld-Katalog (`AvailableFields` / `EXCEL_FIELD_CATALOG`) — also im Spalten-Picker nicht wählbar. Betroffen ist die ganze Gruppe (nicht nur die 2 genannten), darum alle auf einmal geschlossen.

## Acceptance Criteria
- [x] AC-1: Folgende 6 Application-Felder sind im Export-Feld-Katalog (Backend + Frontend) als Spalten wählbar, Kategorie **„Haushalt"**:
  - `persons_in_household` (Personen im Haushalt, Zahl)
  - `heat_pump` (Wärmepumpe, Ja/Nein)
  - `electric_vehicle` (E-Auto, Ja/Nein)
  - `electric_vehicle_count` (E-Auto: Anzahl, Zahl)
  - `electric_vehicle_annual_km` (E-Auto: Jahres-km, Zahl)
  - `electric_hot_water` (Elektrische Warmwasserbereitung, Ja/Nein)
- [x] AC-2: NULL-Werte (Mitglied hat das Feld nicht befüllt / EEG hat es auf `hidden`) erscheinen als **leere Zelle**, nicht als „<nil>" oder „0" (deref-Helper gegen die typed-nil-Pointer-Falle — `derefInt`/`derefBool`).
- [x] AC-3: Felder sind in **beiden** rowMode-Modi (member + metering_point) verfügbar (Mitglieds-Felder, kein memberOnly).
- [x] AC-4: Backend- und Frontend-Katalog sind konsistent (gleiche Keys), per Test gepinnt.

## Edge Cases
- EC-1: EEG hat das Feld auf `hidden` → Wert NULL → leere Zelle (kein Default-„0"/„Nein").
- EC-2: `electric_vehicle_count`/`_annual_km` sind nur befüllt, wenn `electric_vehicle = true` (Service-Layer cleart sonst auf NULL) → im Export leer bei nicht-E-Auto-Mitgliedern.

## Tech / Non-Goals
- Reine Katalog-Erweiterung in `internal/dataexport/excel/fields.go` + `src/lib/data-export-fields.ts`. Keine DB-Migration, kein neues Paket, keine Logik-Änderung. Felder existieren bereits am `shared.Application`-Model.
- Kein Security-Review nötig (kein Auth/Tenant/Schema/Public/Import/Helm-Trigger). Export bleibt admin-only + tenant-scoped.

## QA / Deployment
- `go build/vet/test ./...` grün (neuer Test `TestPROJ113_HouseholdFieldsRegistered` inkl. NULL→leer). `npx tsc` + `npx vitest run` grün (Frontend-Katalog-Test um Haushalts-Felder erweitert). `npm run build` grün.
- Deployed 2026-06-18, `v1.41.1-PROJ-113`. Keine DB-Migration. Owner: `helm upgrade`.
