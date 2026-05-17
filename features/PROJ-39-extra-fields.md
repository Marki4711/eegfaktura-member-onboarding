# PROJ-39: Titel-Nach + Bankname im Public-Form + abweichende Adresse je Zählpunkt

**Status:** In Review
**Created:** 2026-05-17

## Drei unabhängige Erweiterungen

### 1. „Titel Nach" am Antrag

Neues optionales Feld `titel_nach` neben dem bestehenden `titel` (= „Titel vor").
- DB: `application.titel_nach VARCHAR(50) NULL`
- Public-Form: Eingabefeld direkt nach „Titel" und vor „Vorname"
- Mail (Member + EEG): in der Personendaten-Sektion ausgegeben
- Approval-PDF: ergänzt im Personenblock
- Excel-Export: eigene Spalte
- Admin-Edit-Form: bearbeitbar

Bestehende Spalte `titel` wird **nicht umbenannt** (Risiko hoch, kein Mehrwert);
neue Spalte heißt explizit `titel_nach`, das vorhandene Feld ist implizit
„vor dem Namen".

### 2. „Bankname" auch im Public-Form

Das Feld `bank_name` existiert bereits in der DB und im Admin-Edit-Form, war
aber bisher **nicht** im öffentlichen Antragsformular. Anforderung: Mitglied
soll es gleich beim Einreichen erfassen können.

- Public-Form: neues Eingabefeld in der „Bankverbindung"-Sektion zwischen
  IBAN/Kontoinhaber und SEPA-Häkchen, optional
- API: `bankName` ergänzt in `CreateApplicationRequest`
- Bestehende Admin-Wege bleiben unverändert
- Keine DB-Migration nötig (Spalte existiert)

### 3. Abweichende Adresse pro Zählpunkt

**Bricht V1-Architektur-Entscheidung** (CLAUDE.md):

> *"In onboarding, all metering points use the same address as the member.
> Differing metering point addresses are maintained later in eegFaktura."*

Mit PROJ-39 wird das aufgehoben. Begründung: Mitglieder haben oft Zweitwohnsitze
oder mehrere Standorte (Hauptanlage + Nebengebäude), die jetzt schon korrekt
erfasst werden sollten, statt sie später in eegFaktura nachzupflegen.

#### Datenmodell

- `metering_point.address_street VARCHAR(255) NULL`
- `metering_point.address_street_number VARCHAR(50) NULL`
- `metering_point.address_zip VARCHAR(20) NULL`
- `metering_point.address_city VARCHAR(255) NULL`

**Semantik:** wenn ALLE vier NULL → die Adresse des Mitglieds gilt für diesen
Zählpunkt (impliziter Default, kein extra Flag in DB). Wenn ≥1 gesetzt →
alle vier sind Pflicht und beschreiben die abweichende Adresse.

#### UI

Pro Zählpunkt-Block:
- Checkbox „Abweichende Adresse für diesen Zählpunkt"
- Beim Aktivieren: 4 Felder werden eingeblendet (Straße, Hausnummer, PLZ, Ort)
- Beim Deaktivieren: Felder werden ausgeblendet UND die Werte zurückgesetzt
- Checkbox-State ist **rein UI**, wird nicht gespeichert — der Zustand
  ergibt sich beim Reload aus „sind die Adressfelder gefüllt?"

#### Validierung (Backend)

- Pro Zählpunkt: entweder alle vier Adressfelder leer ODER alle vier gesetzt
- Bei abweichender Adresse: gleiche Längen-Constraints wie für Mitgliedsadresse

#### Anzeige / Export

- Mail (EEG): Zählpunkt-Tabelle bekommt eine zusätzliche Zeile „Adresse:
  Straße Nr, PLZ Ort" falls abweichend
- Approval-PDF: Adresse pro Zählpunkt falls abweichend
- Admin-Detail: Adressfelder pro Zählpunkt-Card sichtbar
- Excel-Export: neue Spalten am Zählpunkt-Sheet (oder als Zusatzspalten am
  Hauptsheet, je nach bestehender Struktur)

## Migration

`db/migrations/000037_proj39_extra_fields.up.sql`:
```sql
ALTER TABLE member_onboarding.application
    ADD COLUMN titel_nach VARCHAR(50);

ALTER TABLE member_onboarding.metering_point
    ADD COLUMN address_street        VARCHAR(255),
    ADD COLUMN address_street_number VARCHAR(50),
    ADD COLUMN address_zip           VARCHAR(20),
    ADD COLUMN address_city          VARCHAR(255);
```

## Doku-Updates

- `CLAUDE.md` → „In onboarding, all metering points use the same address" entfernen
- `docs/architecture.md` → ebenso
- `docs/api-spec.md` → CreateApplicationRequest (titelNach, bankName,
  meteringPoints[].addressStreet/...), AdminApplicationDetail
- `docs/domain-model.md` → Tabellen + Geschäftsregel überarbeiten
- `docs/user-guide/02-member-registration.md` → neue Felder erklären
- `CHANGELOG.md` → Entry für PROJ-39

## Out of Scope

- Karten-Visualisierung der Zählpunkt-Adressen
- Adress-Autocomplete
- Geo-Code-Validierung
- Übernahme der abweichenden Adresse beim Core-Import (mapping-Frage —
  Core unterstützt das bereits, aber eigene Migration im Mapping nötig;
  siehe nachfolgende Spec)
