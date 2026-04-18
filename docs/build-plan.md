# Build Plan
## eegfaktura Member Onboarding

## Ziel

Dieses Dokument beschreibt die empfohlene Reihenfolge der technischen Umsetzung für Claude Code.

## Phase 1: Repository-Grundgerüst

Ziel:
- lauffähiges Grundgerüst für Backend und Dokumentation

Umfang:
- Repository-Struktur anlegen
- `docs/`-Ordner anlegen
- Go-Service-Grundgerüst erstellen
- Konfigurationsstruktur
- HTTP-Router
- Health-Endpoint
- DB-Verbindung
- Migrationsordner

Definition of Done:
- Service startet lokal
- Health-Endpoint antwortet
- DB-Verbindung ist konfigurierbar
- Projektstruktur ist dokumentiert

## Phase 2: Datenbankschema

Ziel:
- Schema `member_onboarding` und drei Tabellen technisch anlegen

Umfang:
- Migration `create schema member_onboarding`
- Tabellen:
  - `member_onboarding.application`
  - `member_onboarding.metering_point`
  - `member_onboarding.status_log`
- Constraints
- Indizes
- `updated_at`-Strategie festlegen

Definition of Done:
- Migration läuft lokal erfolgreich
- Tabellen sind vorhanden
- Foreign Keys und Indizes sind gesetzt

## Phase 3: Public API

Ziel:
- öffentliche Registrierung technisch verfügbar machen

Umfang:
- `GET /api/public/registration/{registration_slug}`
- `POST /api/public/applications`
- `PUT /api/public/applications/{id}`
- `POST /api/public/applications/{id}/submit`
- Validierung
- Persistenz in `application`, `metering_point`, `status_log`

Definition of Done:
- Antrag kann angelegt werden
- Antrag kann geändert werden
- Antrag kann validiert und submitted werden
- Statushistorie wird geschrieben

## Phase 4: Admin API

Ziel:
- Review und Bearbeitung durch Admins

Umfang:
- `GET /api/admin/applications`
- `GET /api/admin/applications/{id}`
- `PUT /api/admin/applications/{id}`
- `POST /api/admin/applications/{id}/status`
- Filter und Pagination
- EEG-Berechtigungsprüfung im Backend

Definition of Done:
- Liste funktioniert
- Detailansicht funktioniert
- Statusübergänge werden geprüft und protokolliert
- Admin-Notiz ist bearbeitbar

## Phase 5: Import

Ziel:
- freigegebene Anträge in eegFaktura importieren

Umfang:
- `POST /api/admin/applications/{id}/import`
- Import-Mapping von Onboarding nach Participant-Payload
- interner Core-Client
- Erfolg-/Fehlerbehandlung
- Importstatus in `application` aktualisieren
- `status_log` schreiben

Definition of Done:
- Import ist nur bei `approved` erlaubt
- Payload wird korrekt aufgebaut
- Erfolg und Fehler werden gespeichert
- `target_participant_id` wird bei Erfolg gesetzt

## Phase 6: Auth und Härtung

Ziel:
- produktionsnahe Absicherung

Umfang:
- Keycloak-Anbindung im Admin-Bereich
- Rollen-/EEG-Prüfung
- Fehlerhandling vereinheitlichen
- Logging
- Basis-Tests
- API-Dokumentation vervollständigen

Definition of Done:
- Admin-Endpunkte sind abgesichert
- Fehler sind konsistent
- wichtigste Flows sind getestet

## Prompting-Empfehlung für Claude Code

Claude Code sollte immer in kleinen Paketen arbeiten.

Empfohlene Reihenfolge:
1. Phase 1 implementieren
2. Phase 2 implementieren
3. Phase 3 implementieren
4. Phase 4 implementieren
5. Phase 5 implementieren
6. Phase 6 implementieren

Empfohlener Arbeitsstil:
- immer zuerst relevante Dateien in `docs/` lesen
- nur eine Phase oder ein kleines Teilpaket gleichzeitig umsetzen
- Architektur- und Domain-Regeln strikt einhalten
- keine zusätzlichen Features ohne explizite Freigabe einführen
