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

- [ ] Es existiert eine zentrale Liste konfigurierbarer Felder (z.B. `phone`, `uid_number`, `iban`, `birth_date`)
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

## Technical Requirements

- Feldkonfiguration wird zusammen mit dem Registration-Entrypoint ausgeliefert (kein separater API-Aufruf)
- Backend-Validierung muss die Konfiguration aus der DB lesen, nicht aus statischen Struct-Tags
- Performance: Konfiguration wird pro Request aus der DB gelesen oder gecacht
- Die Liste der konfigurierbaren Felder ist im Code definiert (kein freies Hinzufügen über UI)

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
