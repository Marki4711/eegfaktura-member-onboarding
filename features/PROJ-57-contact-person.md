# PROJ-57 — Ansprechperson für Org-Mitgliedstypen

**Status:** In Review
**Erstellt:** 2026-05-21
**Implementiert:** 2026-05-21
**Quelle:** Owner-Anforderung
**Abhängigkeiten:** PROJ-8 (konfigurierbare Felder), PROJ-20 (Submission-Mail), PROJ-21 (Beitritts-PDF)

---

## Ziel

Bei Mitgliedstypen **Unternehmen, Verein und Gemeinde** soll eine optionale
Ansprechperson angegeben werden können. Wenn die Checkbox aktiv ist,
werden drei zusätzliche Felder eingeblendet: Name, E-Mail, Telefon.

## Hintergrund

Bei Organisationen unterscheiden sich Vertragspartner (= das Org-Konto)
und Ansprechperson (= konkreter Mensch). Bisher gibt's nur ein E-Mail- und
Telefon-Feld, das die EEG-Verwaltung primär für die Korrespondenz nutzt.
Wenn die Org eine bestimmte Person als Ansprechpartner benennen will,
muss das aktuell im Admin-Notiz-Feld hinterlegt werden — unstrukturiert.

## Geklärte Entscheidungen

- **Mitgliedstypen mit Ansprechperson-Option:** `company`, `association`,
  `municipality`. Nicht `private`, `farmer`, `sole_proprietor`.
- **Konfigurierbarkeit:** per-EEG via field_config-Eintrag `contact_person`.
  Default `hidden`.
- **Sichtbarkeit außerhalb des Formulars:**
  - Beitrittsbestätigungs-PDF: eigener Block, wenn Ansprechperson gesetzt
  - EEG-Submission-Mail (PROJ-20): Block in der Mail an die EEG
  - Admin-Detail-Anzeige + Admin-Edit-Form: standard
- **Excel-Export** (PROJ-17): NICHT erweitert (explizit ausgeschlossen)

## Datenmodell

Vier neue Spalten auf `application`:

- `has_contact_person` (BOOLEAN NOT NULL DEFAULT FALSE) — Toggle-Flag
- `contact_person_name` (TEXT NULL)
- `contact_person_email` (TEXT NULL)
- `contact_person_phone` (TEXT NULL)

Der Toggle ist explizit, damit „leer + ja ich will eine Ansprechperson"
und „leer + nein, keine Ansprechperson" semantisch unterscheidbar bleiben.
Beim Submit wird die explizite Wahl gespeichert; Service-Layer cleart
die drei TEXT-Felder auf NULL, wenn `has_contact_person = false`.

## field_config

Single-Switch `contact_person`:

- `hidden`: Feature nicht verfügbar, Checkbox nicht sichtbar (Default)
- `optional`: Checkbox sichtbar, User kann opt-in
- `required`: aktuell wie `optional` behandelt — eine erzwungene
  Ansprechperson ist semantisch unscharf, kein klarer Use-Case in V1

## Frontend-Layout

- Im Org-Daten-Block, **nach** UID/Vereinsnummer, **vor** den allgemeinen
  Kontaktfeldern (Email, Telefon)
- Checkbox „Ansprechperson angeben" — sichtbar nur bei Org-Mitgliedstypen
  UND field_config != hidden
- Wenn aktiv: 3 Inputs in einer responsiven Grid-Zeile (Name full-width
  oder als erste; Email + Telefon als 2-Spalten-Reihe darunter)
- Required-Validierung der 3 Felder gegated auf `hasContactPerson === true`
  (verhindert Submit-Hänger-Bug-Pattern, siehe Geburtsdatum-Bug-Fix)

## PDF

Eigener Block „Ansprechperson" im PDF (nur wenn `has_contact_person`),
mit Name / E-Mail / Telefon als dataRow-Items in einer sectionHeader-
Sektion.

## Mail

In der EEG-Submission-Mail (PROJ-20) ein neuer Sub-Block unter den
Stammdaten der Org, wenn Ansprechperson gesetzt.

## API

- `POST /api/public/applications`: vier neue optionale Felder im Body
- `PATCH /api/admin/applications/{id}`: dito (Admin-Edit)
- Backend cleart die drei TEXT-Felder serverseitig, wenn
  `hasContactPerson=false` oder Mitgliedstyp nicht in der Org-Liste.

## Validierung

- `hasContactPerson=true` ⇒ Name, Email, Phone alle drei required
- Email-Format-Check (Standard `validate:"email"`)
- Phone als max-50 String
- Bei nicht-Org-Mitgliedstypen werden die Felder serverseitig
  unterdrückt (analog zum bestehenden `clearMemberTypeFields`)

## Implementierungs-Schritte

1. DB-Migration 000050
2. Backend (models, requests, repo, service, field_config)
3. Frontend (CONFIGURABLE_FIELDS, schema, defaults, render, submit)
4. Admin-Detail + Admin-Edit
5. PDF-Generator (neuer Block)
6. Submission-Mail (PROJ-20-Template)
7. CHANGELOG + INDEX-Status

## Out of Scope (V1)

- Mehrere Ansprechpersonen
- Excel-Export (explizit ausgeschlossen)
- Rolle/Funktion der Ansprechperson (z. B. „Obmann", „Geschäftsführer")
- Bestätigungs-E-Mail an die Ansprechperson selbst
