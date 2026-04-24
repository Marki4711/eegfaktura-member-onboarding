# PROJ-13: Externe Registrierungs-API

## Status: Planned
**Created:** 2026-04-24
**Last Updated:** 2026-04-24

## Dependencies
- Requires: PROJ-1 (Public Registration) — gleiches Datenmodell, gleiche Einreichungslogik
- Requires: PROJ-6 (E-Mail-Benachrichtigungen) — Bestätigungsmail wird wiederverwendet
- Extends: PROJ-12 (SEPA-Lastschriftmandat PDF) — PDF-Anhang gilt auch für externe Einreichungen

## User Stories

- Als **EEG-Betreiber** möchte ich Mitgliedsdaten über eine REST-API einreichen können, damit ich in meiner eigenen Website ein eigenes Registrierungsformular betreiben kann ohne das eingebaute Formular zu verwenden.
- Als **EEG-Betreiber** möchte ich im Admin-Backend einen API-Key generieren und widerrufen können, damit ich den Zugang zur externen API selbst verwalten kann.
- Als **EEG-Administrator** möchte ich extern eingereichte Anträge genauso in der Admin-Oberfläche sehen und bearbeiten können wie Anträge über das Standardformular, damit es keinen Unterschied im Prüfungsworkflow gibt.
- Als **neues Mitglied** möchte ich auch bei einer externen Einreichung eine Bestätigungsmail erhalten, damit ich den Eingang meines Antrags bestätigt bekomme.
- Als **Betreiber** möchte ich, dass ein falscher oder fehlender API-Key klar mit HTTP 401 abgelehnt wird, damit ich Integrationsfehler schnell erkenne.

## Acceptance Criteria

### API-Key-Verwaltung (Admin-Backend)

- [ ] Im Admin-Backend gibt es pro EEG einen Abschnitt „Externe API" mit den Optionen: Key generieren / Key widerrufen
- [ ] Ein Klick auf „API-Key generieren" erzeugt einen neuen Key und zeigt ihn **einmalig** im Klartext an (danach nicht mehr lesbar)
- [ ] Der angezeigte Key hat das Format: `moak_<rc_number>_<32 zufällige alphanumerische Zeichen>` (leicht identifizierbar)
- [ ] In der Datenbank wird nur der Hash des Keys gespeichert (SHA-256), niemals der Klartext
- [ ] Pro EEG existiert maximal ein aktiver API-Key — wird ein neuer generiert, wird der alte automatisch invalidiert
- [ ] Der Admin kann den aktiven Key jederzeit widerrufen (kein neuer Key wird dabei erzeugt)
- [ ] Der Admin sieht, ob ein API-Key aktiv ist, und wann er zuletzt generiert wurde — aber nie den Key selbst
- [ ] Änderungen (generieren / widerrufen) sind sofort wirksam (kein Cache)

### Externe Einreichungs-Endpunkt

- [ ] `POST /api/external/v1/applications` akzeptiert einen vollständigen Mitgliedsantrag inkl. Zählpunkten
- [ ] Authentifizierung erfolgt über `Authorization: Bearer <api-key>` im HTTP-Header
- [ ] Der API-Key bestimmt die EEG (RC-Nummer) — es wird keine RC-Nummer im Body benötigt
- [ ] Bei gültigem Key und validen Daten: Antrag wird angelegt und **direkt in `submitted` Status** überführt
- [ ] Response: `201 Created` mit `{ "id": "...", "referenceNumber": "..." }`
- [ ] Bei ungültigem oder widerrufenem Key: `401 Unauthorized`
- [ ] Bei fehlenden oder ungültigen Feldern: `422 Unprocessable Entity` mit Fehlerliste (identische Validierungsregeln wie Standardformular)
- [ ] Bei inaktivem Entrypoint (`is_active = false`): `410 Gone`

### Pflichtfelder im Request-Body

- [ ] `memberType` — Pflicht (`natural_person` | `legal_entity`)
- [ ] `firstname` + `lastname` — Pflicht bei `natural_person`
- [ ] `companyName` — Pflicht bei `legal_entity`
- [ ] `email` — Pflicht (valides E-Mail-Format)
- [ ] `residentStreet`, `residentStreetNumber`, `residentZip`, `residentCity` — alle Pflicht
- [ ] `residentCountry` — Pflicht (ISO 3166-1 alpha-2, z.B. `AT`)
- [ ] `iban` — Pflicht (valides IBAN-Format)
- [ ] `accountHolder` — Pflicht
- [ ] `privacyAccepted: true` — Pflicht (der Aufrufer bestätigt, dass das Mitglied zugestimmt hat)
- [ ] `sepaMandateAccepted: true` — Pflicht
- [ ] `meteringPoints` — mindestens ein Eintrag, jeder mit `meteringPoint` (Zählpunktbezeichnung) und `direction` (`CONSUMPTION` | `PRODUCTION` | `GENERATION`)
- [ ] Konfigurierbare Felder (`birthDate`, `phone`, etc.) — Pflicht/Optional gemäß der aktiven `field_config` der EEG (identisch wie Standardformular)

### Verhalten nach Einreichung

- [ ] Bestätigungsmail wird an das Mitglied versendet — identisches Verhalten wie beim Standardformular
- [ ] SEPA-Lastschriftmandat PDF wird angehängt wenn für die EEG aktiviert und alle EEG-Felder ausgefüllt
- [ ] EEG-Benachrichtigungsmail wird versendet wenn `contact_email` konfiguriert ist
- [ ] Status-Log-Eintrag wird geschrieben (transition `draft → submitted`)
- [ ] Der Antrag erscheint im Admin-Backend ohne besonderen Hinweis auf die Einreichungsart

### Rate Limiting

- [ ] Maximal 10 Einreichungen pro Minute pro API-Key — bei Überschreitung: `429 Too Many Requests`

## Edge Cases

- **API-Key gehört zu inaktiver EEG** (`is_active = false`): `410 Gone` — kein Hinweis auf den Key-Status
- **Neuer Key wird generiert während bestehende Requests laufen**: Laufende Requests mit dem alten Key werden noch abgeschlossen (keine Mid-Request-Invalidierung)
- **`privacyAccepted: false` oder `sepaMandateAccepted: false`**: `422` — Feld-Fehler, kein Antrag wird angelegt
- **Doppelte Einreichung** (gleiche E-Mail + RC in kurzer Zeit): Kein technisches Duplikat-Blocking — verhält sich wie beim Standardformular (zwei separate Anträge möglich)
- **Konfigurierbare Pflichtfelder fehlen**: `422` mit denselben Fehlermeldungen wie das Standardformular
- **Sehr langer Mitgliedsname / Anschrift**: Gleiche Längenbeschränkungen wie im Standardformular
- **Widerruf während Admin den Key gerade anzeigt** (Race Condition): Kein Problem — Key wird nach Anzeige nicht gespeichert

## Technical Requirements

- **Neuer Endpunkt**: `POST /api/external/v1/applications` — eigene Route-Gruppe `/api/external`, kein Keycloak-Middleware (eigene API-Key-Middleware)
- **Neue DB-Tabelle**: `member_onboarding.external_api_key`
  - `id`, `rc_number` (FK), `key_hash` (VARCHAR(64), SHA-256 hex), `created_at`, `revoked_at` (nullable), `last_generated_at`
- **Key-Hashing**: SHA-256 des Klartext-Keys (kein bcrypt — Performance bei jedem Request)
- **Admin-Endpunkte**: 
  - `POST /api/admin/settings/api-key?rc_number=...` — generiert neuen Key, gibt Klartext zurück (einmalig)
  - `DELETE /api/admin/settings/api-key?rc_number=...` — widerruft aktiven Key
  - `GET /api/admin/settings/api-key?rc_number=...` — liefert Status (aktiv/inaktiv + `last_generated_at`)
- **Rate Limiting**: In-Memory-Counter pro Key (Token-Bucket oder sliding window, 10 req/min)
- **Einreichungslogik**: Wiederverwendung von `ApplicationService.SubmitApplication` — kein duplizierter Code
- **Frontend**: Neuer Abschnitt „Externe API" in der Admin-Settings-Seite

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
