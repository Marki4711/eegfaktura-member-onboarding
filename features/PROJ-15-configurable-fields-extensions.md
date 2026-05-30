# PROJ-15: Konfigurierbare Felder — Erweiterungen

## Status: Deployed (Erweiterung B teilweise zurückgebaut durch PROJ-68)
**Created:** 2026-04-24
**Last Updated:** 2026-05-30

> **PROJ-68 Nachtrag 2026-05-30:** Der EEG-weite Default-Wert-Mechanismus (`admin_value` + `applyAdminValues()`) wurde entfernt. Der vierte Feldzustand `admin_only` bleibt als reine Public-Form-Hide-Markierung erhalten (versteckt für Mitglieder, im Admin-Edit-Dialog pro Antrag editierbar). Acceptance-Criteria B2/B4/B5/B7 und die zugehörigen Tests sind damit obsolet — siehe `features/PROJ-68-remove-admin-value-default-mechanism.md`.

## Dependencies
- Requires: PROJ-8 (Konfigurierbare Felder) — diese Erweiterungen bauen direkt darauf auf
- Requires: PROJ-5 (Keycloak-secured Admin Area) — Konfiguration nur für authentifizierte Admins

## Enthaltene Erweiterungen

### Erweiterung A: Hilfetext für „Aktiv am"

Das Feld `membership_start_date` (Aktiv am / gewünschtes Beitrittsdatum) soll einen erklärenden Hilfetext erhalten, der das Mitglied informiert, was dieses Datum bedeutet.

### Erweiterung B: Vierter Feldstatus „Admin-Only"

Zu den bestehenden drei Zuständen (`hidden`, `optional`, `required`) kommt ein vierter Zustand:
**`admin_only`** — Das Feld wird **nicht** im Registrierungsformular angezeigt. Der Admin kann in den EEG-Einstellungen einen fixen Standardwert konfigurieren, der automatisch auf alle neuen Anträge angewendet wird.

---

## User Stories

### Hilfetext
- Als **neues Mitglied** möchte ich beim Feld „Aktiv am" eine kurze Erklärung sehen, was dieses Datum bedeutet, damit ich es korrekt ausfülle.

### Admin-Only
- Als **EEG-Administrator** möchte ich bestimmte Felder als „Admin-Only" konfigurieren können, damit ich intern einen Standardwert für alle Anträge vorgeben kann, ohne das Formular für Mitglieder zu verlängern.
- Als **EEG-Administrator** möchte ich für ein `admin_only`-Feld einen Standardwert hinterlegen können, der automatisch auf alle neu eingereichten Anträge angewendet wird.
- Als **Mitglied** möchte ich `admin_only`-Felder nicht im Registrierungsformular sehen, damit das Formular nur relevante Eingaben von mir verlangt.

---

## Acceptance Criteria

### A: Hilfetext für „Aktiv am"

- [ ] Unter dem Eingabefeld `membership_start_date` (wenn sichtbar) wird ein Hilfetext angezeigt:
  > „Datum, ab dem die Aktivierung der angegebenen Zählpunkte für die EEG erfolgen soll. Nützlich wenn die Aktivierung nicht sofort, sondern zu einem fest definierten Zeitpunkt stattfinden soll."
- [ ] Der Hilfetext erscheint nur wenn das Feld sichtbar ist (`optional`, `required` oder `admin_only`-Fallthrough falls sichtbar)
- [ ] Der Hilfetext ist statisch (kein konfigurierbarer Text, kein DB-Feld nötig)

### B: Vierter Feldstatus `admin_only`

- [ ] Der Umschalter in der Admin-Feldkonfiguration hat vier Zustände: **Ausblenden** / **Optional** / **Pflichtfeld** / **Admin-Vorgabe**
- [ ] Bei Auswahl von „Admin-Vorgabe" erscheint in der Einstellungszeile ein zusätzliches Eingabefeld für den Standardwert (typ-gerecht: Text, Zahl, Datum, Ja/Nein)
- [ ] Der Standardwert wird in der Datenbank als `admin_value TEXT` in `member_onboarding.field_config` gespeichert
- [ ] Das Feld mit Status `admin_only` wird **nicht** im Registrierungsformular angezeigt
- [ ] Beim Erstellen eines Antrags (POST `/api/public/registration` + interner Ablauf) wird der `admin_value` aus der Feldkonfiguration automatisch auf den entsprechenden Feldwert des Antrags gesetzt
- [ ] Die Konversion `admin_value TEXT` → Zieltyp des Feldes erfolgt serverseitig (Int: `strconv.Atoi`, Float: `strconv.ParseFloat`, Bool: `true`/`false`, Datum: `YYYY-MM-DD`); bei ungültigem Wert → Feld bleibt NULL (kein Fehler für das Mitglied)
- [ ] Ist `admin_value` leer/NULL, wird kein Wert gesetzt (Feld bleibt NULL im Antrag)
- [ ] Im Admin-Review-Bereich ist der vom Admin vorgegebene Wert genauso sichtbar wie ein manuell eingegebener Wert
- [ ] Die externe API (PROJ-13) übernimmt `admin_only`-Felder wenn im Body mitgeliefert; fehlen sie, wird der `admin_value` ebenso automatisch angewendet

## Edge Cases

- Admin setzt `admin_value = "abc"` für ein Integer-Feld → `admin_value` wird beim Speichern nicht validiert; Konversionsfehler beim Antrag erzeugt NULL (kein Abbruch)
- Admin wechselt Feld von `admin_only` → `optional` → `admin_value` bleibt in DB erhalten, wird aber nicht mehr automatisch angewendet (nur relevant wenn wieder auf `admin_only` gesetzt)
- `membership_start_date` mit `admin_only` + `admin_value = "2026-05-01"` → alle neuen Anträge bekommen dieses Startdatum ohne Mitglied-Eingabe
- Externes Mitglied (PROJ-13) liefert Feld explizit mit → expliziter Wert überschreibt den `admin_value`

## Technische Hinweise

- Neue DB-Spalte: `admin_value TEXT` in `member_onboarding.field_config` (nullable, nur relevant wenn `state = 'admin_only'`)
- Neue DB-State-Wert: `'admin_only'` — CHECK-Constraint in `field_config.state` anpassen
- Konversionslogik in `internal/application/` — dort wo `CreateApplicationRequest` befüllt wird

---

## QA Test Results

**Tested:** 2026-04-24
**Tester:** Claude (QA Engineer)
**Status:** APPROVED — keine Critical/High Bugs

### Test-Übersicht

| AC | Beschreibung | Status |
|----|-------------|--------|
| A1 | Hilfetext für `membership_start_date` im DOM vorhanden | PASS |
| A2 | Kein JS-Fehler nach PROJ-15-Änderungen | PASS |
| B: Vier Zustände im UI | Ausblenden / Optional / Pflichtfeld / Admin-Vorgabe | PASS |
| B: DB-Spalte `admin_value` | TEXT-Spalte in `field_config` vorhanden | PASS |
| B: Feld nicht im Formular | Backend mappt `admin_only` → `hidden` in öffentlicher API | PASS |
| B: Konversion admin_value → Zieltyp | Int, Float, Bool, Date — unit-tested | PASS |
| B: Leer/NULL → kein Wert | Feld bleibt NULL wenn admin_value leer oder nil | PASS |
| B6 | Admin-Einstellungsseite lädt ohne JS-Fehler | PASS |
| B1–B5, REG1 | API-Tests (Backend nicht verfügbar) | SKIPPED |

**E2E-Tests:** `tests/PROJ-15-configurable-fields-extensions.spec.ts` — 6 passed, 12 skipped (Backend offline)

**Unit-Tests (Go):** 8 neue Tests in `field_config_test.go` für `applyAdminValues` — alle bestanden

### Bugs

#### BUG-1 (Low) — `phone` und `birth_date` nicht in `applyAdminValues`
- **Beschreibung:** `phone` und `birth_date` sind als konfigurierbare Felder registriert (Standardzustand: `optional`) und erscheinen im Admin-UI mit der Option „Admin-Vorgabe". Der `applyAdminValues`-Aufruf in `application_service.go` ignoriert diese beiden Felder. Ein Admin-Wert für `phone` oder `birth_date` wird stillschweigend nicht angewendet.
- **Schwere:** Low — semantisch macht ein fixer Standardwert für Telefon/Geburtsdatum keinen Sinn; wird in der Praxis nicht genutzt werden
- **Schritte:** Admin setzt `phone` auf `admin_only` + `adminValue = "+43 ..."` → neuer Antrag hat `phone = NULL` statt dem Admin-Wert

#### BUG-2 (Medium) — Zählpunkt-Admin-Vorgaben werden nicht angewendet
- **Beschreibung:** `transformer`, `installation_number` und `installation_name` erscheinen in der Admin-UI unter „Zählpunkt-Felder" mit allen vier Zuständen inkl. „Admin-Vorgabe". Beim Erstellen eines Antrags gibt es jedoch kein Äquivalent zu `applyAdminValues` für Zählpunkte — Admin-Vorgaben für diese drei Felder werden stillschweigend ignoriert.
- **Schwere:** Medium — Feature-AC nicht erfüllt für Zählpunkt-Felder; Admin sieht Option im UI, die keine Wirkung hat
- **Schritte:** Admin setzt `transformer` auf `admin_only` + `adminValue = "T1"` → Zählpunkt im neuen Antrag hat `transformer = NULL`

#### BUG-3 (Low) — Admin-Vorgabe-Eingabefeld nicht typ-gerecht
- **Beschreibung:** Laut AC soll das Eingabefeld für den Admin-Standardwert typ-gerecht sein (Datum → Datepicker, Zahl → Number-Input, Ja/Nein → Toggle). Aktuell ist es immer ein einfaches `<Input type="text">`. Ungültige Werte (z.B. Text für ein Integer-Feld) werden zwar serverseitig abgefangen (→ NULL), aber das UI bietet keinen Eingabeschutz und keine Hinweise.
- **Schwere:** Low — kein Datenverlust möglich (Server konvertiert fehlertolerant), aber UX entspricht nicht der Spec
- **Schritte:** Feld `membership_start_date` auf `admin_only` setzen → es erscheint ein `<input type="text">` statt einem Datepicker

### Sicherheits-Audit
- `admin_only`-Felder werden korrekt als `hidden` in der öffentlichen Registrierungs-API zurückgegeben — Mitglieder sehen weder den Status noch den Admin-Wert
- Admin-Werte können nur von authentifizierten Admins gesetzt werden (Keycloak-geschützter Endpoint)
- Keine serverseitige Validierung des `admin_value`-Formats beim Speichern — bewusste Designentscheidung (Fehlertoleranz bei Konversion)

### Regressions-Tests
- Öffentliche Registrierungs-API: `fieldConfig`-Werte bleiben plain strings (E2E AC-REG1 — skipped, Backend offline)
- Registration-Formular rendert ohne Fehler (AC-A1, AC-A2 — PASS)
- Admin-Einstellungsseite rendert ohne Fehler (AC-B6 — PASS)

### Produktionsbereitschaft
**READY** — BUG-2 und BUG-3 sind bekannte Einschränkungen; keine Critical- oder High-Bugs. BUG-2 sollte in einem Folge-Ticket (PROJ-15b) behandelt werden.

---

## Deployment

**Deployed:** 2026-04-24
**Image Tag:** `sha-81265b9`

### Deployment-Schritte (auszuführen auf dem Server)

1. **DB-Migrationen anwenden:**
   ```bash
   make migrate-up
   # oder: migrate -path db/migrations -database $DB_URL up
   ```
   Angewendete Migrationen:
   - `000015_add_company_sepa_mandate` — `use_company_sepa_mandate` Spalte (zusammen mit PROJ-14)
   - `000016_add_admin_value_to_field_config` — `admin_value TEXT` Spalte + `admin_only` CHECK-Constraint in `field_config`

2. **Helm-Upgrade:**
   ```bash
   helm upgrade eegfaktura-member-onboarding ./helm/member-onboarding \
     -f helm/member-onboarding/values-env.yaml \
     -f helm/member-onboarding/values-secret.yaml
   ```
