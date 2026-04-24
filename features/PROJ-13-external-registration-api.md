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
- [ ] Der angezeigte Key hat das Format: `moak_<32 zufällige alphanumerische Zeichen>` — die RC-Nummer ist **nicht** Teil des Keys (kein Information Leak falls ein Key versehentlich in Logs oder E-Mails landet)
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
- **Integrationsanforderung**: Der API-Key darf **niemals im Browser-Frontend des Betreibers** verwendet werden. Der Aufruf von `POST /api/external/v1/applications` muss server-seitig erfolgen (PHP, Node.js, .NET, etc.). Der API-Key wird als Umgebungsvariable auf dem Server des Betreibers gespeichert und verlässt diesen nicht. Betreiber, die ein reines Browser-Formular ohne eigenen Server betreiben möchten, müssen das Session-Token-Verfahren verwenden (siehe Tech Design).

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)

### Betroffene Komponenten

Beide Seiten — Backend (neuer Endpunkt + Middleware + DB-Tabelle) und Frontend (neuer Settings-Abschnitt).

```
Admin Settings Page (bestehend)
└── Einleitungstext-Editor (bestehend)
└── EEG-Stammdaten & SEPA-Mandat (bestehend)
└── [NEU] AdminApiKeyEditor
│   ├── Status-Anzeige: aktiv / kein Key vorhanden + Datum der letzten Generierung
│   ├── Button „API-Key generieren"
│   │   └── Bestätigungs-Dialog (einmalige Anzeige)
│   │       ├── Key-Text (kopierbar, Monospace)
│   │       ├── Hinweis „Dieser Key wird nicht mehr angezeigt"
│   │       └── Button „Schließen"
│   └── Button „Key widerrufen" (nur sichtbar wenn ein aktiver Key existiert)
└── Formular-Felder-Editor (bestehend)

Backend
├── db/migrations/
│   ├── 000014_add_external_api_key.up.sql    ← neu: Tabelle external_api_key
│   └── 000014_add_external_api_key.down.sql
├── internal/application/
│   └── external_api_key_repo.go              ← neu: CRUD für external_api_key
├── internal/http/
│   ├── external.go                           ← neu: POST /api/external/v1/applications
│   ├── apikey_middleware.go                  ← neu: API-Key-Authentifizierung + Rate Limit
│   └── admin.go                              ← erweitert: 3 neue Key-Verwaltungs-Endpunkte
└── cmd/server/main.go                        ← erweitert: neue Route-Gruppe /api/external
```

### Datenmodell-Erweiterung

Neue Tabelle `member_onboarding.external_api_key`:

| Feld | Typ | Bedeutung |
|------|-----|-----------|
| `id` | UUID | Primärschlüssel |
| `rc_number` | TEXT UNIQUE | Fremdschlüssel zur EEG — pro EEG max. 1 Zeile |
| `key_hash` | VARCHAR(64) | SHA-256-Hash des Klartext-Keys (Hex), nie der Key selbst |
| `revoked_at` | TIMESTAMPTZ NULL | NULL = aktiv; gesetzt = widerrufen |
| `last_generated_at` | TIMESTAMPTZ | Zeitpunkt der letzten Key-Generierung |
| `created_at` | TIMESTAMPTZ | Zeitpunkt der ersten Generierung |

Ein neuer Key überschreibt `key_hash` und setzt `revoked_at` auf NULL. Es gibt immer maximal eine Zeile pro EEG (UPSERT).

### API-Änderungen

**Neue Route-Gruppe** `/api/external` — eigene API-Key-Middleware (kein Keycloak):

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `POST` | `/api/external/v1/applications` | Externe Einreichung — Key im `Authorization: Bearer`-Header |

**Neue Admin-Endpunkte** (in bestehender `/api/admin/settings`-Gruppe, Keycloak-gesichert):

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET` | `/api/admin/settings/api-key?rc_number=...` | Status: aktiv/inaktiv + `last_generated_at` |
| `POST` | `/api/admin/settings/api-key?rc_number=...` | Key generieren — gibt Klartext einmalig zurück |
| `DELETE` | `/api/admin/settings/api-key?rc_number=...` | Aktiven Key widerrufen |

### Einreichungsfluss (extern)

```
POST /api/external/v1/applications
  1. API-Key-Middleware: Key aus Authorization-Header extrahieren
     → SHA-256 hashen → in external_api_key suchen
     → nicht gefunden oder revoked_at gesetzt → 401
     → Rate-Limit-Check (10 req/min pro Key) → 429 bei Überschreitung
     → RC-Nummer in Request-Kontext setzen
  2. Handler: alle Felder aus Body validieren (identische Regeln wie Formular)
     → Fehler → 422 mit Fehlerliste
  3. ApplicationService.CreateApplication() → Antrag im Status "draft"
  4. ApplicationService.SubmitApplication() → direkt zu "submitted"
     → Bestätigungsmail + SEPA-PDF (falls aktiv) → async
  5. Response: 201 Created mit id + referenceNumber
```

**Wiederverwendung**: Schritte 3 + 4 rufen exakt dieselben Service-Methoden auf wie das Standardformular. Kein duplizierter Validierungs- oder Einreichungscode.

### API-Key-Middleware

Folgt dem Muster der bestehenden `KeycloakAuthMiddleware` in `internal/http/auth_middleware.go`:
- Liest `Authorization: Bearer <key>` aus dem Header
- Berechnet SHA-256 des Keys
- Schlägt in `external_api_key` nach — verifiziert, dass `revoked_at IS NULL`
- Prüft In-Memory-Rate-Limit (sliding window, 10 req/60s pro Key-Hash)
- Legt RC-Nummer im Request-Kontext ab (analog zu Keycloak-Claims)

### Tech-Entscheidungen

**Kein `rc_number` im Key-Format** — `moak_<32chars>` ohne RC-Nummer. Die Zuordnung Key→EEG erfolgt ausschließlich über den DB-Lookup. Würde ein Key versehentlich in einem Log, einer E-Mail oder einem Git-Commit landen, ist die betroffene EEG nicht sofort identifizierbar.

**SHA-256 statt bcrypt** — API-Keys sind 32 zufällige alphanumerische Zeichen. Bei dieser Länge und Zufälligkeit ist ein Wörterbuchangriff nicht praktikabel. SHA-256 ist bei jedem Request in Mikrosekunden berechenbar; bcrypt würde 100–300 ms kosten und die API bei hoher Last ausbremsen.

**In-Memory Rate Limiting** — Kein Redis, keine externe Abhängigkeit. Bei einem einzelnen Pod für den MVP ist das ausreichend. Bei mehreren Pods wird das Limit pro Pod geprüft (effektiv N×10 req/min global) — akzeptabler Kompromiss für V1.

**Einmaliger Key im Dialog** — Der Klartext-Key verlässt das Backend genau einmal (bei Generierung). Das Frontend zeigt ihn in einem modalen Dialog mit Kopier-Button. Sobald der Dialog geschlossen wird, ist der Key unwiederbringlich — nur ein neuer Key kann generiert werden. Dieses Muster ist aus GitHub/Stripe/Supabase bekannt und gut verstanden.

**Kein separates `source`-Feld auf `application`** — Externe und formularbasierte Anträge sind im Admin-Backend identisch behandelt. Es gibt keinen Filter oder Hinweis auf die Herkunft. Das vereinfacht den Admin-Workflow und entspricht der User Story.

### Neue Pakete

Keine neuen externen Abhängigkeiten — SHA-256 und sync.Map sind in der Go-Standardbibliothek enthalten.

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
