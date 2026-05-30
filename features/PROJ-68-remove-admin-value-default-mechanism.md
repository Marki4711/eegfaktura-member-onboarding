# PROJ-68 — admin_value-Default-Mechanismus entfernen

**Status:** In Progress
**Created:** 2026-05-30
**Owner:** TBD
**Source:** Tester-Feedback 2026-05-30 — „die Option Admin-Vorgabe sollte die Eigenschaft vor dem Nutzer verstecken, aber nur im Admin-Edit-Dialog editierbar machen. Das Wert-Eingabefeld in den Settings ist nicht notwendig."

## Hintergrund

PROJ-15B (April 2026) führte den vierten Feldzustand `admin_only` ein. Die Spec sah dabei zwei kombinierte Effekte vor:

1. Feld wird im Public-Form **ausgeblendet**
2. Admin hinterlegt in den EEG-Settings einen **Default-Wert** (`field_config.admin_value`), der serverseitig beim Submit auf alle neuen Anträge gesetzt wird (`applyAdminValues()` in `application_service.go`)

Owner-Entscheidung 2026-05-30: Der Default-Wert-Mechanismus wird **nicht genutzt** und macht die UI unnötig verwirrend. Der erwünschte Sinn des `admin_only`-States ist nur Punkt 1 (versteckt im Public-Form, sichtbar/editierbar im Admin-Edit-Dialog pro Antrag).

## Was bleibt — was geht

### Bleibt

- Feldzustand `admin_only` als gültiger Wert in `field_config.state`
- Im Public-Form: Felder mit `admin_only` werden **nicht** angezeigt (`hidden`-äquivalent für Mitgliedssicht)
- Im Admin-Edit-Form: Felder mit `admin_only` werden **angezeigt + sind editierbar** (heutiges Verhalten via `fsApp(name) !== "hidden"`)
- Externe API (`/api/external/applications`): kann `admin_only`-Felder im Body mitliefern — der explizite Wert gewinnt heute schon

### Geht

- Spalte `field_config.admin_value`
- Go-Feld `application.FieldConfigEntry.AdminValue`
- TS-Feld `AdminFieldConfigEntry.adminValue`
- Funktion `applyAdminValues()` + ihr Aufruf in `SubmitApplication`
- UI-Input-Zeile unter dem Toggle bei `admin_only`-Auswahl
- 9 Unit-Tests `TestApplyAdminValues_*` in `field_config_test.go`
- E2E-Test-Setup für `adminValue` in `PROJ-15-configurable-fields-extensions.spec.ts` (AC-B2, AC-B4)
- Config-Export-/Import-Feld `adminValue` (PROJ-61): wird beim Import aus alten Bundles still ignoriert

## Acceptance Criteria

### AC-1 — DB-Schema
- Neue Migration `000058_drop_admin_value_from_field_config.up.sql`: `ALTER TABLE … DROP COLUMN admin_value`
- CHECK-Constraint behält alle vier State-Werte (`hidden`, `optional`, `required`, `admin_only`)
- Down-Migration re-added die Spalte (Werte bleiben leer — kein Rollback der Daten möglich, der Wert ist weg)

### AC-2 — Backend
- `FieldConfigEntry.AdminValue` aus Go-Struct entfernen
- `applyAdminValues()` + Aufruf in `SubmitApplication` entfernen
- Alle SQL-Queries auf `admin_value` entfernen (`field_config_repo.go` + `_tx.go`)
- Admin-Handler-DTO (GET + PUT `/api/admin/settings/fields`) verliert `adminValue`
- ConfigExport-Schema verliert `AdminValue`; Importer ignoriert das Feld aus alten Bundles (kein Fehler, ein Log-Warn)

### AC-3 — Frontend
- TS-Typen `AdminFieldConfigEntry` + `FieldConfigImportEntry` verlieren `adminValue`
- `admin-field-config-editor.tsx`: die `{entry.state === "admin_only" && <Input>}`-Block wird entfernt
- Status-Auswahl bleibt 4-stellig (Ausblenden / Optional / Pflichtfeld / Admin-Vorgabe)

### AC-4 — Tests
- `internal/application/field_config_test.go`: alle 9 `TestApplyAdminValues_*`-Tests + Helper `baseAppWithAllOptional` (falls nur dort genutzt) entfernen
- `tests/PROJ-15-configurable-fields-extensions.spec.ts`: AC-B2 + AC-B4 entfernen oder ohne `adminValue` formulieren

### AC-5 — Doku
- `docs/domain-model.md`: `admin_value` aus der `field_config`-Beschreibung entfernen
- `docs/api-spec.md`: GET + PUT `/api/admin/settings/fields` Body ohne `adminValue`
- `docs/user-guide/06-admin-settings.md`: Beschreibung des `admin_only`-States neu formulieren („vor Mitglied versteckt, nur im Admin-Edit-Dialog pro Antrag editierbar — kein EEG-weiter Default")
- `docs/user-guide/changelog.md`: Eintrag
- `features/PROJ-15`: Nachtrag-Notiz im Spec, dass B2 (Default-Wert-Mechanismus) per PROJ-68 zurückgebaut wurde

## Non-Goals

- **Den `admin_only`-State entfernen.** Nur den Default-Wert-Mechanismus.
- **Migration der historischen `admin_value`-Daten** (Backfill auf Anträge). Wenn eine EEG das Feature heute nutzt, sind die bestehenden Anträge bereits mit dem Wert befüllt; neue Anträge brauchen Admin-Edit.
- **Externe API neu definieren.** `/api/external/*` akzeptiert wie heute Felder im Body — `admin_only` ändert daran nichts.

## Migration-Strategy

```sql
-- 000058_drop_admin_value_from_field_config.up.sql
ALTER TABLE member_onboarding.field_config
  DROP COLUMN admin_value;
```

CHECK-Constraint bleibt unverändert. Bestehende Zeilen mit `state = 'admin_only'` behalten ihren State; nur der `admin_value`-Wert geht verloren. Daten-Verlust ist akzeptiert (Owner-Entscheidung — Feature wird nicht genutzt).

## Risiken

- **EEGs, die das Feature aktiv nutzen:** verlieren ihre Default-Werte ersatzlos. Aktuell unbestätigt; vor Deploy einmal `SELECT rc_number, field_name, admin_value FROM member_onboarding.field_config WHERE admin_value IS NOT NULL` gegen Production prüfen. Bei Treffern: Admins vor Deploy informieren.
- **Config-Bundles, die `adminValue` enthalten:** Importer muss das Feld beim Lesen toleranter werden (silent ignore + log warn) statt Schema-Fehler zu werfen.
- **Audit-Trail:** `field_config`-Änderungen werden heute nicht versioniert; die Spalten-Drop ist endgültig.
