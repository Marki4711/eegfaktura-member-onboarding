# PROJ-15: Konfigurierbare Felder — Erweiterungen

## Status: Planned
**Created:** 2026-04-24
**Last Updated:** 2026-04-24

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
