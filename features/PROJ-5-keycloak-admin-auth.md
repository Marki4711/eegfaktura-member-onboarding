# PROJ-5: Keycloak-gesicherte Admin-Oberfläche

## Status: Architected
**Created:** 2026-04-19
**Last Updated:** 2026-04-20

## Dependencies
- Requires: PROJ-2 (Admin Review) — Admin-API muss existieren, bevor sie abgesichert wird
- Requires: PROJ-3 (Admin Frontend UI) — Admin-Oberfläche muss existieren

## User Stories

- Als Tenant-Admin möchte ich mich mit meinem Keycloak-Account einloggen, damit ich die Anträge meiner EEGs verwalten kann.
- Als Tenant-Admin möchte ich nur die Anträge meiner zugewiesenen EEGs sehen, damit keine unbefugten Datenzugriffe möglich sind.
- Als Superuser möchte ich alle Anträge aller EEGs sehen, damit ich systemweite Verwaltungsaufgaben erledigen kann.
- Als unauthentifizierter Benutzer möchte ich beim Aufruf der Admin-Oberfläche automatisch zum Keycloak-Login weitergeleitet werden.
- Als Tenant-Admin möchte ich nach dem Login sichergehen, dass für meine EEGs ein Eintrag in der Datenbank existiert, damit die Registrierungslinks funktionieren.

## Acceptance Criteria

### Authentifizierung
- [ ] Der Admin-Bereich (`/admin`) ist ohne gültigen Keycloak-Token nicht zugänglich
- [ ] Nicht eingeloggte Benutzer werden automatisch zum Keycloak-Login-Screen weitergeleitet
- [ ] Nach erfolgreichem Login werden Benutzer zurück zur Admin-Oberfläche geleitet
- [ ] Ein Logout-Button beendet die Session und leitet zum Keycloak-Logout weiter

### Autorisierung — Tenant-Admin
- [ ] Ein Benutzer mit nicht-leerem `tenant`-Attribut im JWT gilt als Tenant-Admin
- [ ] Tenant-Admins sehen ausschließlich Anträge von EEGs, deren RC-Nummern in ihrem `tenant`-Array stehen
- [ ] Die Filterliste der Admin-Oberfläche ist auf die eigenen EEGs eingeschränkt
- [ ] Direktzugriff auf einen Antrag einer fremden EEG via URL liefert HTTP 403

### Autorisierung — Superuser
- [ ] Ein Benutzer mit der Realm Role `superuser` sieht Anträge aller EEGs ohne Einschränkung
- [ ] Superuser haben kein `tenant`-Attribut (oder es wird ignoriert)

### Kein Zugriff
- [ ] Benutzer ohne `superuser`-Rolle und ohne `tenant`-Attribut erhalten HTTP 403
- [ ] Die Admin-Oberfläche zeigt eine verständliche Fehlermeldung bei 403

### Sync-Logik (Tenant-Admin)
- [ ] Nach dem Login eines Tenant-Admins wird für jede RC-Nummer in seinem `tenant`-Array geprüft, ob ein Eintrag in `registration_entrypoint` existiert
- [ ] Fehlende Einträge werden per `INSERT ... ON CONFLICT DO NOTHING` automatisch angelegt
- [ ] Die Sync-Logik läuft einmalig pro Session, nicht bei jedem Request
- [ ] Für Superuser wird keine Sync-Logik ausgeführt
- [ ] Bestehende Einträge werden nicht gelöscht, wenn eine RC-Nummer aus dem `tenant`-Attribut entfernt wird

### Token-Struktur
- [ ] Das `tenant`-Attribut ist als Multivalued User Attribute via Client Scope Mapper im Access Token enthalten
- [ ] Die App liest `realm_access.roles` für die Superuser-Prüfung
- [ ] Die App liest `tenant` (String-Array) für die Tenant-Admin-Prüfung

## Edge Cases

- **Leeres `tenant`-Array:** Benutzer hat das Attribut, aber es ist leer → kein Zugriff (wie kein Attribut), HTTP 403
- **Token abgelaufen:** Refresh-Token wird verwendet; falls auch abgelaufen → Redirect zum Login
- **Keycloak nicht erreichbar:** Fehlermeldung statt stummem Fail; kein Zugriff auf Admin-Bereich
- **RC-Nummer im `tenant`-Attribut existiert nicht in eegFaktura:** Eintrag in `registration_entrypoint` wird trotzdem angelegt (eeg_id aus Keycloak ist die einzige Quelle); der Antrag läuft dann ins Leere bis die EEG in eegFaktura angelegt ist
- **Superuser hat zusätzlich ein `tenant`-Attribut:** `superuser`-Rolle hat Vorrang — alle EEGs werden angezeigt
- **Gleichzeitige Sessions:** Sync läuft pro Session unabhängig; doppelte Inserts werden durch `ON CONFLICT DO NOTHING` abgefangen
- **Tenant-Admin wird zum Superuser befördert:** Beim nächsten Login greift die neue Rolle; keine manuelle Aktion nötig

## Technical Requirements

- **Keycloak-Realm:** `EEGFaktura`
- **Keycloak-Client:** `eegfaktura-member-onboarding`
- **Valid Redirect URI / Web Origin:** wird pro Deployment konfiguriert (frei wählbare Domain)
- **Token-Claim `tenant`:** Multivalued User Attribute, via Client Scope Mapper in den Access Token gemappt
- **Token-Claim `realm_access.roles`:** Standard Keycloak JWT-Struktur
- **Backend-Middleware:** Jeder Admin-API-Request wird serverseitig gegen das JWT validiert (kein reines Frontend-Guarding)
- **Session-Sync:** Einmalig nach Token-Ausstellung, nicht bei jedem API-Call

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)

### Überblick

PROJ-5 fügt dem bestehenden Admin-Bereich zwei Dinge hinzu:
1. **Authentifizierung** — der Zugang ist nur mit einem gültigen Keycloak-Token möglich
2. **Autorisierung** — was ein eingeloggter Benutzer sehen darf, hängt von seinem Benutzertyp ab

Es gibt keine neuen Seiten. Die bestehenden Admin-Seiten (`/admin/applications`, `/admin/applications/[id]`) bleiben unverändert — sie werden lediglich abgesichert.

### Komponenten-Struktur

```
src/app/admin/
+-- layout.tsx              ← NEU: Keycloak-Session prüfen; kein Token → Login-Redirect
|                              Logout-Button in der Header-Leiste
+-- applications/
|   +-- page.tsx            ← unverändert (Absicherung erfolgt im layout)
|   +-- [id]/page.tsx       ← unverändert

src/lib/
+-- auth.ts                 ← NEU: Token lesen, Rolle/Tenant prüfen, Hilfsfunktionen
+-- keycloak.ts             ← NEU: Keycloak-Client-Konfiguration

src/app/admin/
+-- unauthorized/page.tsx   ← NEU: 403-Seite für eingeloggte Benutzer ohne Berechtigung
```

**Go-Backend (bestehend, wird erweitert):**
```
internal/http/
+-- middleware.go           ← NEU: JWT-Validierung für alle /api/admin/* Routen
+-- admin.go                ← erweitert: tenant-Filter aus dem Token anwenden
internal/application/
+-- registration_entrypoint_repo.go  ← erweitert: UpsertForTenants (Sync-Logik)
```

### Datenfluss: Login

```
Browser                  Next.js (SSR)           Keycloak
  |                           |                      |
  |-- GET /admin/applications→|                      |
  |                           |-- kein Token?        |
  |                           |-- Redirect --------->|
  |                           |                      |-- Login-Formular
  |<----------------------------------------- Code--|
  |-- GET /admin/applications→|                      |
  |   ?code=...               |-- Token tauschen --->|
  |                           |<--- Access Token ----|
  |                           |-- Sync-Logik (falls Tenant-Admin)
  |<-- Admin-Oberfläche ------|
```

### Datenfluss: API-Request

```
Browser             Next.js API-Route        Go-Backend
  |                       |                      |
  |-- GET /api/admin/... →|                      |
  |   (mit Session-Cookie)|-- Bearer Token ----->|
  |                       |                      |-- JWT prüfen
  |                       |                      |-- Rolle/Tenant extrahieren
  |                       |                      |-- Filter anwenden
  |<-- gefilterte Daten --|<----- Response -------|
```

### Autorisierungslogik (vereinfacht)

| Benutzertyp | Erkennungsmerkmal | Sichtbarkeit |
|---|---|---|
| Superuser | `realm_access.roles` enthält `superuser` | alle EEGs |
| Tenant-Admin | `tenant`-Array nicht leer | nur eigene RC-Nummern |
| Kein Zugriff | weder noch | HTTP 403 |

### Sync-Logik bei Login (Tenant-Admin)

Nach dem ersten gültigen Token-Tausch prüft die App für jeden Eintrag im `tenant`-Array des Tokens, ob ein Datensatz in `registration_entrypoint` existiert. Fehlende Einträge werden automatisch angelegt. Das passiert einmalig pro Session — nicht bei jedem Seitenaufruf.

### Tech-Entscheidungen

**NextAuth.js mit Keycloak-Provider**
Das Standard-Paket für Next.js-Authentifizierung. Übernimmt den OAuth2-Flow (Login, Token-Tausch, Refresh, Logout) und stellt die Session serverseitig zur Verfügung. Kein eigener Auth-Code nötig.

**JWT-Validierung im Go-Backend (Middleware)**
Die Admin-API-Endpunkte prüfen jeden Request serverseitig. Das Frontend-Guarding allein ist nicht ausreichend — ein direkter API-Aufruf ohne Frontend würde sonst unkontrolliert durchkommen.

**Tenant-Filter im Backend, nicht im Frontend**
Die RC-Nummern-Einschränkung wird serverseitig in der SQL-Query angewendet. Das Frontend zeigt nur, was das Backend zurückgibt — kein clientseitiges Ausblenden von Daten.

**`ON DELETE RESTRICT` auf dem FK `application.rc_number`**
Bereits umgesetzt (Migration 000009). Stellt sicher, dass RC-Nummern in `registration_entrypoint` nicht gelöscht werden können, solange Anträge darauf verweisen.

### Neue Abhängigkeiten (npm)

| Paket | Zweck |
|---|---|
| `next-auth` | OAuth2/OIDC-Flow mit Keycloak, Session-Management |
| `jose` | JWT-Signaturprüfung im Go-Backend (Go-seitig: `golang-jwt/jwt`) |

**Go-seitig:**
| Paket | Zweck |
|---|---|
| `golang-jwt/jwt/v5` | JWT-Parsing und -Validierung |
| `MicahParks/keyfunc` | Automatisches Laden der Keycloak JWKS-Keys |

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
