# High-Level Architecture
## eegfaktura Member Onboarding

## 1. Ziel und Einordnung

`eegfaktura Member Onboarding` ist eine neue Komponente im eegFaktura-Umfeld zur Selbstregistrierung von Mitgliedern einer EEG.

Die Komponente ergänzt den bestehenden Prozess, in dem neue Mitglieder bisher manuell durch einen Administrator in eegFaktura angelegt werden. Ziel ist es, dass sich Mitglieder künftig selbst über ein Webformular registrieren können. Diese Daten werden zunächst nicht direkt in den produktiven Stammdatenbestand von eegFaktura übernommen, sondern in einer eigenen Datenhaltung der neuen Komponente gespeichert. Erst nach Prüfung durch einen Admin erfolgt die bewusste Übernahme in den normalen Datenbestand von eegFaktura.

## 2. Architekturprinzipien

- eigenständige Komponente mit eigenem Repository: `eegfaktura-member-onboarding`
- Frontend im selben Stack wie `eegfaktura-web`
- Backend als Go-Service
- gleiche PostgreSQL-Datenbank wie eegFaktura, aber eigenes Schema `member_onboarding`
- Keycloak für den Admin-Bereich
- Import in den produktiven Datenbestand nur über einen internen Service-Aufruf an den eegFaktura-Core
- keine direkten Schreibzugriffe des Onboardings auf Core-Tabellen

## 3. Hauptkomponenten

### 3.1 Public Web
Öffentliche Benutzeroberfläche für neue Mitglieder.

Aufgaben:
- Einstieg über festen Registrierungslink pro EEG
- Erfassung von Mitgliedsdaten
- Erfassung mehrerer Zählpunkte
- clientseitige Validierung
- Absenden des Antrags an das Backend

Nicht verantwortlich für:
- Persistenz
- Statuslogik
- Import
- direkte Kommunikation mit dem Core

### 3.2 Admin Web
Interne Benutzeroberfläche für EEG-Administratoren.

Aufgaben:
- Anzeige und Filterung eingehender Anträge
- Detailansicht eines Antrags
- Bearbeitung der Stammdaten
- Setzen von Statuswerten
- Pflege interner Hinweise
- Auslösen des Imports

Nicht verantwortlich für:
- direkte Datenbankzugriffe
- direkte Core-Schreiblogik
- eigene Authentifizierung

### 3.3 Member Onboarding Backend
Zentrale Fachlogik der Komponente.

Technologie:
- Go
- REST-API
- PostgreSQL-Zugriff
- Keycloak-Anbindung im Admin-Kontext

Aufgaben:
- Public API
- Admin API
- serverseitige Validierung
- Statusübergänge
- Schreiben und Lesen im Schema `member_onboarding`
- Persistenz mehrerer Zählpunkte
- Statushistorie
- Import-Mapping
- interner Core-Aufruf
- Protokollierung des Importergebnisses

### 3.4 Persistenz
Die Persistenz erfolgt in derselben PostgreSQL-Datenbank wie eegFaktura, aber in einem eigenen dedizierten Schema:

- `member_onboarding.application`
- `member_onboarding.metering_point`
- `member_onboarding.status_log`

### 3.5 eegFaktura Core
Der Core bleibt das führende System für produktive Teilnehmerdaten.

Aufgaben:
- finale fachliche Validierung beim Import
- produktive Anlage des Teilnehmers
- Rückgabe von Ziel-ID oder Fehlermeldung

## 4. Systemgrenzen

Erlaubte Verbindungen:
- Public Web -> Member Onboarding Backend
- Admin Web -> Member Onboarding Backend
- Member Onboarding Backend -> Schema `member_onboarding`
- Member Onboarding Backend -> eegFaktura Core

Nicht erlaubte Verbindungen:
- Public Web -> eegFaktura Core
- Admin Web -> eegFaktura Core
- Frontend -> Datenbank
- Member Onboarding -> direkte Core-Tabellen

## 5. Datenhaltung

Das Modul verwendet ein bewusst reduziertes relationales Modell ohne JSON-Felder und ohne Dokumentenverwaltung.

Tabellen:
- `member_onboarding.application`
- `member_onboarding.metering_point`
- `member_onboarding.status_log`

Grundregeln:
- ein Antrag enthält genau ein Mitglied
- ein Antrag kann mehrere Zählpunkte enthalten
- im Onboarding haben alle Zählpunkte dieselbe Adresse wie das Mitglied
- abweichende Zählpunktadressen werden später in eegFaktura gepflegt
- Tarife, Rollen und Kontoinformationen werden nicht im Onboarding verwaltet

## 6. Technologische Festlegungen

### Frontend
Verwendet denselben Frontend-/Web-Stack wie `eegfaktura-web`.

### Backend
Eigenständiger Go-Service.

### Datenbank
PostgreSQL, gleiches DB-System wie eegFaktura, eigenes Schema `member_onboarding`.

### Authentifizierung
- Public Web: kein Login erforderlich
- Admin Web: bestehende Keycloak-basierte Authentifizierung

### API-Stil
REST mit JSON.

### Deployment
Eigenständiger Build und eigenständige Migrationen im Repository `eegfaktura-member-onboarding`.

## 7. Zusammenfassung

`eegfaktura Member Onboarding` wird als eigenständige, aber eng an eegFaktura angelehnte Komponente umgesetzt.

Die Architektur besteht aus:
- Public Web für die Selbsterfassung
- Admin Web für Review und Importauslösung
- Go-Backend als fachlicher Kern
- PostgreSQL-Schema `member_onboarding`
- internem Service-Aufruf an den eegFaktura-Core für die produktive Übernahme
