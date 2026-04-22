# PROJ-8: Konfigurierbare Felder pro EEG

## Status: In Review
**Created:** 2026-04-21
**Last Updated:** 2026-04-22

## Dependencies
- Requires: PROJ-1 (Public Registration) — Felder werden im Registrierungsformular angezeigt
- Requires: PROJ-2 (Admin Review) — Admin verwaltet die Feldkonfiguration
- Requires: PROJ-5 (Keycloak-secured Admin Area) — Konfiguration nur für authentifizierte Admins

## User Stories

- Als EEG-Administrator möchte ich festlegen können, welche optionalen Felder im Registrierungsformular meiner EEG angezeigt werden, damit das Formular nur relevante Daten abfragt.
- Als EEG-Administrator möchte ich einzelne Felder als Pflichtfeld oder optional markieren können, damit die Datenqualität meinen Anforderungen entspricht.
- Als Mitglied möchte ich beim Öffnen des Registrierungslinks nur die für meine EEG relevanten Felder sehen, damit das Formular übersichtlich bleibt.
- Als Superuser möchte ich die Feldkonfiguration für alle EEGs einsehen und anpassen können, damit ich eine einheitliche Qualität sicherstellen kann.
- Als Entwickler möchte ich neue optionale Felder zentral definieren können, damit sie für alle EEGs aktivierbar sind ohne Codeänderungen.

## Acceptance Criteria

- [ ] Es existiert eine zentrale Liste konfigurierbarer Felder mit folgendem Inhalt:

  **Bestehende Felder (bereits im Formular, aber konfigurierbar):**
  | Feldname | Typ | Beschreibung |
  |---|---|---|
  | `phone` | Text | Telefonnummer |
  | `birth_date` | Datum | Geburtsdatum |
  | `uid_number` | Text | UID-Nummer (Unternehmen) |

  **Neue optionale Felder (Zählpunkt-Ebene):**
  | Feldname | Typ | Beschreibung |
  |---|---|---|
  | `transformer` | Text | Transformator |
  | `installation_number` | Text | Anlagen-Nr. |
  | `installation_name` | Text | Anlagenname |

  **Neue optionale Felder (Antrags-Ebene, Ja/Nein):**
  | Feldname | Typ | Beschreibung |
  |---|---|---|
  | `heat_pump` | Boolean | Wärmepumpe vorhanden |
  | `electric_vehicle` | Boolean | E-Auto vorhanden |
  | `electric_hot_water` | Boolean | Warmwasser elektrisch (Boiler) |

  **Neue optionale Felder (Antrags-Ebene, Zahl):**
  | Feldname | Typ | Beschreibung |
  |---|---|---|
  | `persons_in_household` | Ganzzahl | Anzahl Personen im Haushalt |
  | `consumption_previous_year` | Zahl (kWh) | Verbrauch Vorjahr |
  | `consumption_forecast` | Zahl (kWh) | Verbrauch Prognose |
  | `feed_in_forecast` | Zahl (kWh) | Einspeisung Prognose |
  | `pv_power_kwp` | Zahl (kWp) | PV-Leistung |

  **Neue optionale Felder (Antrags-Ebene, Datum):**
  | Feldname | Typ | Beschreibung |
  |---|---|---|
  | `membership_start_date` | Datum | Aktiv am (gewünschtes Beitrittsdatum zur EEG) |
- [ ] Jedes konfigurierbare Feld hat pro EEG drei Zustände: `hidden` (nicht angezeigt), `optional`, `required`
- [ ] Der `/api/public/registration/{rc_number}` Endpunkt liefert die Feldkonfiguration der EEG mit
- [ ] Das Registrierungsformular rendert Felder dynamisch entsprechend der Konfiguration
- [ ] Die Backend-Validierung prüft Pflichtfelder gemäß der EEG-Konfiguration (nicht statisch)
- [ ] Ein Admin kann die Feldkonfiguration seiner EEG(s) über die Admin-Oberfläche bearbeiten
- [ ] Änderungen an der Feldkonfiguration wirken sich sofort auf neue Registrierungsaufrufe aus
- [ ] Bereits eingereichte Anträge bleiben von Konfigurationsänderungen unberührt

## Edge Cases

- Was passiert, wenn ein Feld in einem bereits gespeicherten Antrag vorhanden ist, aber in der aktuellen Konfiguration auf `hidden` steht? → Antrag bleibt unverändert, Konfiguration gilt nur für neue Anträge.
- Was passiert, wenn die Feldkonfiguration einer EEG noch nicht angelegt wurde? → Fallback auf eine systemweite Standardkonfiguration.
- Was passiert, wenn ein Admin ein Pflichtfeld deaktiviert, das in alten Anträgen belegt ist? → Konfigurationsänderung wird gespeichert, historische Daten bleiben erhalten.
- Was passiert, wenn das IBAN-Feld deaktiviert wird, aber ein Antrag bereits eine IBAN enthält? → IBAN bleibt im Antrag gespeichert, wird aber im Formular nicht mehr abgefragt.
- Was passiert, wenn zwei Admins desselben Tenants gleichzeitig die Konfiguration ändern? → Last-write-wins, keine Konflikterkennung nötig.
- Zahlenfelder (kWh, kWp, Personen): Positive Ganzzahl; Validierung bei Einreichung gemäß Feldtyp.
- Boolean-Felder: Wird ein nicht ausgefülltes optionales Bool-Feld als `false` oder als `null` (keine Angabe) gespeichert? → Bei optionalen Feldern `null`, bei Pflichtfeld explizite Auswahl erzwingen.
- Zählpunkt-Felder vs. Antrags-Felder: Verbrauchs- und Einspeisewerte sowie PV-Leistung sind pro Zählpunkt relevant; Haushaltsdaten (Personen, Wärmepumpe, E-Auto, Boiler) gehören zum Antrag. Die Feldkonfiguration muss diese Zuordnung kennen.

## Technical Requirements

- Feldkonfiguration wird zusammen mit dem Registration-Entrypoint ausgeliefert (kein separater API-Aufruf)
- Backend-Validierung muss die Konfiguration aus der DB lesen, nicht aus statischen Struct-Tags
- Performance: Konfiguration wird pro Request aus der DB gelesen oder gecacht
- Die Liste der konfigurierbaren Felder ist im Code definiert (kein freies Hinzufügen über UI)

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)

### Feldkategorien

**Kategorie 1 — Pflichtfelder (fix, immer sichtbar und required):**
Name/Firmenname, E-Mail, Adresse, IBAN, SEPA-Mandat, Zählpunktnummer, Richtung, Teilnahmefaktor.
Diese Felder haben keinen konfigurierbaren State und erscheinen nicht in der Einstellungsseite.

**Kategorie 2 — Konfigurierbare Felder (state: `hidden` | `optional` | `required`):**
Alle in den Acceptance Criteria gelisteten Felder. Standard für neue Felder: `hidden`.

### Komponentenstruktur

```
Admin-Bereich
+-- Navigation (erweitert)
|   +-- Anträge (bestehend)
|   +-- Einstellungen (neu → /admin/settings)
+-- Einstellungsseite /admin/settings
    +-- EEG-Auswahl (nur Superuser mit mehreren EEGs)
    +-- Feldkonfigurations-Editor
        +-- Abschnitt: Antragsteller-Felder
        |   +-- Feldzeile: Label + Umschalter (hidden / optional / required)
        |   +-- (eine Zeile pro konfigurierbarem Feld)
        +-- Abschnitt: Zählpunkt-Felder
            +-- Feldzeile: Label + Umschalter

Registrierungsformular (bestehend, erweitert)
+-- Persönliche Daten
|   +-- Pflichtfelder (immer sichtbar)
|   +-- Konfigurierbare Antrags-Felder (je nach EEG-Konfiguration)
+-- Zählpunkte (je Block)
    +-- Pflichtfelder (immer sichtbar)
    +-- Konfigurierbare Zählpunkt-Felder (je nach EEG-Konfiguration)
```

### Datenhaltung

**Neue Tabelle `member_onboarding.field_config`:**
- `id` — UUID, Primärschlüssel
- `rc_number` — Text, Verweis auf `registration_entrypoint`
- `field_name` — Text (z.B. `heat_pump`, `transformer`)
- `state` — Text: `hidden` | `optional` | `required`
- `updated_at` — Zeitstempel
- Eindeutiger Index auf `(rc_number, field_name)`

Nur Abweichungen vom Standard werden gespeichert (sparse). Fehlt ein Eintrag, gilt: `hidden` für neue Felder, bisheriges Verhalten für bestehende Felder (`phone`, `birth_date`, `uid_number`).

**Neue Spalten in `member_onboarding.application`:**
- `membership_start_date` — Datum (nullable)
- `persons_in_household` — Ganzzahl (nullable)
- `consumption_previous_year`, `consumption_forecast`, `feed_in_forecast` — Ganzzahl kWh (nullable)
- `pv_power_kwp` — Dezimalzahl kWp (nullable)
- `heat_pump`, `electric_vehicle`, `electric_hot_water` — Boolean (nullable; `null` = keine Angabe)

**Neue Spalten in `member_onboarding.metering_point`:**
- `transformer`, `installation_number`, `installation_name` — Text (nullable)

**Felddefinition im Go-Code (zentrale Registry):**
Jedes konfigurierbare Feld ist im Code mit Name, Typ und Scope (`application` | `metering_point`) registriert. Neue Felder werden hier ergänzt — keine Schemaänderung an `field_config` nötig.

### API

**Bestehender Endpunkt erweitert:**
`GET /api/public/registration/{rc_number}` liefert zusätzlich:
```
"fieldConfig": {
  "phone": "optional",
  "heat_pump": "required",
  "transformer": "hidden",
  ...
}
```

**Neue Admin-Endpunkte:**
- `GET /api/admin/settings/fields?rc_number=RC123456` — Feldkonfiguration lesen
- `PUT /api/admin/settings/fields?rc_number=RC123456` — Feldkonfiguration speichern

**Bestehende Antrags-Endpunkte:**
`POST` und `PUT /api/public/applications/...` nehmen neue Felder entgegen. Die Pflichtfeld-Validierung liest den State aus der DB.

### Tech-Entscheidungen

| Entscheidung | Begründung |
|---|---|
| Pflichtfelder nicht konfigurierbar (Option B) | Kernfelder sind für den eegFaktura-Import zwingend erforderlich — Fehlkonfiguration würde den Prozess brechen |
| Sparse-Tabelle | Neue Felder sind automatisch `hidden` für alle EEGs ohne DB-Einträge anlegen zu müssen |
| Konfiguration im Registrierungs-Response gebündelt | Kein zweiter API-Aufruf im Frontend |
| Felddefinition im Code | Kontrollierte Einführung neuer Felder; keine freie Konfiguration undokumentierter Felder |
| Einstellungsseite `/admin/settings` | Trennung von Antragsverwaltung und EEG-Konfiguration; skaliert für PROJ-9 (Dokumente) u.a. |

### Neue Pakete
Keine — alle UI-Komponenten (Switch, Card, Tabs) sind bereits vorhanden.

## Implementation Notes (Frontend)

**Implemented 2026-04-22:**

- `src/lib/api.ts`: Added `FieldState`, `FieldConfig`, `ConfigurableField` types; `CONFIGURABLE_FIELDS` registry; `resolveFieldState` helper; `getFieldConfig`/`saveFieldConfig` admin API functions; new optional fields on `CreateApplicationRequest` and `MeteringPointRequest`.
- `src/components/registration-form.tsx`: Refactored to use `buildFormSchema(fieldConfig)` factory — all configurable fields are optional in the base schema; superRefine applies "required" checks based on EEG field config. Phone/birthDate/uidNumber now respect fieldConfig (default "optional" = shown). New "Weitere Angaben" card renders extra application-level fields when at least one is not "hidden". Boolean fields use a 3-way Select (Ja/Nein/Keine Angabe).
- `src/components/metering-point-fields.tsx`: Accepts `fieldConfig` prop; each metering point row conditionally shows transformer, installation_number, installation_name fields when not "hidden".
- `src/app/admin/settings/page.tsx`: New settings page at `/admin/settings` with EEG selector dropdown (for multi-EEG admins), loads field config via API, renders `AdminFieldConfigEditor`.
- `src/components/admin-field-config-editor.tsx`: New component — grouped field list (Antragsteller-Felder / Zählpunkt-Felder) with 3-way segmented control per field, save button with feedback.
- `src/app/admin/layout.tsx`: Added "Einstellungen" nav link.

**Backend not yet implemented.** Settings page shows an error if backend API returns 4xx/5xx. New registration form fields submit to the backend but will be ignored until PROJ-8 backend is deployed. All new fields default to "hidden" at the frontend level, so existing registration forms are unaffected until an admin explicitly enables fields.

## Implementation Notes (Backend)

**Implemented 2026-04-22:**

- `db/migrations/000011_configurable_fields.up.sql`: Adds 9 columns to `application`, 3 columns to `metering_point`, creates `member_onboarding.field_config` table.
- `internal/shared/models.go`: Added 9 new fields to `Application`, 3 to `MeteringPoint`. All nullable via pointer types.
- `internal/shared/requests.go`: Extended `CreateApplicationRequest`, `CreateMeteringPointRequest`, and `RegistrationConfig` (added `FieldConfig map[string]string`).
- `internal/application/field_config_repo.go`: New repository with `knownConfigurableFields` registry, `effectiveState()` helper, sparse `Get`/`Save` logic.
- `internal/application/application_repo.go`: All SQL updated for new columns (Create, GetByID, Update).
- `internal/application/metering_point_repo.go`: SQL updated for transformer/installation columns.
- `internal/application/registration_service.go`: Now accepts `fieldConfigRepo`, loads and includes `FieldConfig` in public response (fail-open).
- `internal/application/application_service.go`: `CreateApplication` + `SubmitApplication` load field config and call `validateConfigurableRequiredFields` + `validateConfigurableMeteringPointFields`. `validateMemberTypeFields` no longer hardcodes `birth_date` as required.
- `internal/application/admin_service.go`: `AdminApplicationService` gains `fieldConfigRepo`, `GetFieldConfig`, `SaveFieldConfig`.
- `internal/http/admin.go`: Added `GetFieldConfig` and `SaveFieldConfig` handlers with tenant-scope enforcement.
- `cmd/server/main.go`: `fieldConfigRepo` initialized, all constructors updated, routes `GET/PUT /api/admin/settings/fields` registered.
- `internal/application/application_service_test.go`: Updated stale `TestValidateMemberTypeFields_Private_MissingBirthDate` to reflect PROJ-8 design change.
- `internal/application/field_config_test.go`: 20 new unit tests covering `effectiveState` and both validation helpers.

## QA Test Results

**QA Date:** 2026-04-22
**Status:** In Review — 1 Medium bug found

### Automated Tests

| Suite | Result |
|---|---|
| `go test ./...` | ✅ All pass (20 new PROJ-8 unit tests + existing) |
| `npm test` (Vitest) | ⚠️ Pre-existing startup error (rolldown native binding on Windows) — unrelated to PROJ-8 |
| `npm run test:e2e` (chromium) | 37 passed, 1 failed (regression — see bug below) |
| `npm run test:e2e` (Mobile Safari) | WebKit binary not installed — pre-existing infra issue |

### Acceptance Criteria

| # | Criterion | Result |
|---|---|---|
| AC-1 | Central list of 15 configurable fields exists in code | ✅ Pass — `knownConfigurableFields` in `field_config_repo.go` |
| AC-2 | Each field has states: `hidden`, `optional`, `required` | ✅ Pass — enforced in DB constraint and `validFieldStates` map |
| AC-3 | `/api/public/registration/{rc_number}` includes `fieldConfig` | ✅ Pass — `RegistrationConfig.FieldConfig` returned |
| AC-4 | Registration form renders fields dynamically | ✅ Pass — `buildFormSchema(fieldConfig)` factory + `fs()` helper |
| AC-5 | Backend validates required fields per EEG config (not static) | ✅ Pass — `validateConfigurableRequiredFields` reads from DB config |
| AC-6 | Admin can edit field config via admin UI | ✅ Pass — `/admin/settings` page + `GET/PUT /api/admin/settings/fields` |
| AC-7 | Config changes affect new registrations immediately | ✅ Pass — config read per-request, no cache |
| AC-8 | Already submitted applications unaffected by config changes | ✅ Pass — config only applied at create/submit time |

### Edge Cases Tested

| Edge Case | Result |
|---|---|
| No field config in DB → fallback to `knownConfigurableFields` defaults | ✅ Pass |
| DB error loading field config → fail-open, registration continues | ✅ Pass (fmt.Printf warning + empty map fallback) |
| PUT body contains unknown field name | ✅ Pass — silently skipped, not persisted |
| PUT body contains invalid state value | ✅ Pass — silently skipped |
| Boolean optional field submitted as null | ✅ Pass — stored as NULL in DB |

### Security Audit

| Check | Result |
|---|---|
| `GET /api/admin/settings/fields` requires JWT | ✅ Protected by `KeycloakAuthMiddleware` |
| `PUT /api/admin/settings/fields` requires JWT | ✅ Protected |
| Tenant-admin can only read/write own RC numbers | ✅ `containsRC(claims.Tenant, rcNumber)` check |
| Superuser can access all RC numbers | ✅ `claims.IsSuperuser()` bypass |
| SQL injection in field name/state | ✅ Not possible — parameterized queries + allowlist validation |
| XSS via field config values | ✅ Values constrained to `hidden`/`optional`/`required` allowlist |
| Field config API exposes no sensitive data | ✅ Only field names and states returned |

### Bugs Found

**BUG-PROJ8-1 — Medium: uid_number no longer validated as required for company type on frontend (PROJ-7 AC-11 regression)**

- **Severity:** Medium
- **Affected test:** `PROJ-7-member-types.spec.ts:143` — AC-11: company type shows errors when uid and registerNumber missing
- **Steps to reproduce:** Open registration form as company type (`Unternehmen`), fill in company name, leave uid_number and register_number empty, submit. Expected: both error messages appear. Actual: only "Firmenbuchnummer ist erforderlich" appears; no "UID-Nummer ist erforderlich".
- **Root cause:** PROJ-8 made `uid_number` a configurable field with default state "optional". The frontend's `buildFormSchema` superRefine only marks it required when the EEG field config sets it to "required". The previous hardcoded check for company type was removed. The backend `validateMemberTypeFields` still requires uid_number for company type, so submissions are rejected server-side — but the client-side error is missing.
- **Impact:** Company-type registrants can advance through client-side validation without providing a UID number, then receive a confusing server-side error. No data integrity issue.
- **Fix needed:** Restore uid_number as structurally required for company type in the frontend's `superRefine`, independent of field config — OR remove the uid_number requirement from `validateMemberTypeFields` (backend) to be consistent with the configurable approach.

### New Tests Written

- `internal/application/field_config_test.go` — 20 Go unit tests:
  - `effectiveState`: explicit override, fallback to registered default, fallback to hidden, unknown field
  - `validateConfigurableRequiredFields`: all-optional config, phone/birthDate/membershipStartDate/pvPower/heatPump required/missing/present, multiple fields missing
  - `validateConfigurableMeteringPointFields`: transformer required/missing/present, installation_number, multiple points with second missing
- `tests/PROJ-8-configurable-fields.spec.ts` — 13 Playwright E2E tests (all pass):
  - Default visibility of phone, birthDate (optional = visible)
  - Default hiding of heat_pump, electric_vehicle, membership_start_date, persons_in_household
  - Default hiding of transformer, installation_number on metering point
  - Admin settings route redirects unauthenticated users
  - Registration form renders successfully with valid RC
  - Phone and birthDate do not trigger required errors by default

### Production-Ready Decision

**NOT READY** — 1 Medium bug (BUG-PROJ8-1) must be fixed. The regression breaks a documented PROJ-7 acceptance criterion and creates a misleading UX for company-type registrations.

## Deployment
_To be added by /deploy_
