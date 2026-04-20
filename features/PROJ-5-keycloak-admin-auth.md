# PROJ-5: Keycloak-gesicherte Admin-Oberfläche

## Status: Planned
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
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
