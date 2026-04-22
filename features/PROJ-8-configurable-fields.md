# PROJ-8: Konfigurierbare Felder pro EEG

## Status: Planned
**Created:** 2026-04-21
**Last Updated:** 2026-04-21

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

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
