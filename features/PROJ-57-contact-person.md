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

## v2 — Feiner steuerbare Pflichtigkeit (Update 2026-05-21)

Erweiterung der Pflicht-Logik: Email und Telefon werden seit v2 pro EEG
einzeln auf `hidden | optional | required` gestellt werden können.

**Status:** Implementiert.

**Änderungen:**

- Zwei neue field_config-Einträge `contact_person_email` und
  `contact_person_phone`, beide Default `required` (= identisches
  Verhalten zu V1 für alle Bestand-EEGs).
- `clearContactPersonIfDisabled` cleart das jeweilige Detail-Feld
  serverseitig, wenn der State `hidden` ist.
- Required-Validierung gegated auf `hasContactPerson && state == "required"`.
- Frontend-Render konditional pro Feld; Pflicht-Marker dynamisch.
- Email-Format-Check läuft auch bei `optional`, falls Wert eingegeben.
- Admin-Edit-Form bleibt unverändert — Admin sieht alle drei Felder.

**Name bleibt fix Pflicht** wenn Toggle aktiv: ohne Name ist eine
Ansprechperson semantisch sinnlos, daher kein eigener field_config.

## v3 — Master-Switch entfällt, alle drei Felder einzeln steuerbar (2026-05-21 abends)

Vereinfachung des Modells: der separate `contact_person`-Master-Switch
entfällt. Stattdessen werden alle drei Felder (Name, Email, Telefon)
einzeln per field_config konfigurierbar. Die Sichtbarkeit der Checkbox
im Public-Formular wird aus den drei Sub-Feldern abgeleitet.

**Status:** Implementiert.

**Änderungen:**

- `contact_person` als field_config-Eintrag entfernt
- Neuer Eintrag `contact_person_name` (Default hidden, analog zu
  Email + Telefon)
- Default aller drei Subfelder ist jetzt `hidden` — Feature aus, bis
  EEG aktiv konfiguriert (vorher: Email + Phone Default required,
  Master-Switch Default hidden — Verhalten identisch, weil Master-
  Switch hidden alles ausblendete)
- Backend-Helper `contactPersonEnabled(fieldConfig)`: liefert true,
  wenn mindestens eines der drei != hidden
- `clearContactPersonIfDisabled`: cleart bei „alle drei hidden" ODER
  nicht-Org-Mitgliedstyp
- Required-Validierung für jedes Feld nur bei state=required (Name
  also nicht mehr automatisch Pflicht — wenn EEG Name auf optional
  stellt, kann er leer bleiben)
- Frontend: Sichtbarkeit der gesamten Sektion + jedes einzelnen Felds
  konditional auf nicht-hidden; Pflicht-Marker dynamisch

**Migration:** EEGs, die zuvor `contact_person=optional` gesetzt hatten,
müssen die drei Subfelder neu konfigurieren. Da das Feature heute
frisch eingeführt wurde, ist der Migrations-Aufwand minimal.
