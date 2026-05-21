# PROJ-56 — Netzbetreiber-Info-Seite im Beitrittsbestätigungs-PDF

**Status:** Planned
**Erstellt:** 2026-05-21
**Quelle:** Owner-Anforderung mit konkretem Layout-Bild
**Abhängigkeiten:** PROJ-21 (Beitrittsbestätigungs-PDF), PROJ-44 (Netzbetreiber-Vollmacht), PROJ-8 (konfigurierbare Felder)

---

## Ziel

Zusätzliche Seite im Beitrittsbestätigungs-PDF mit einer Zusammenfassung
der relevanten Daten für den Netzbetreiber: Kundennummer, Inventarnummer,
erteilte Vollmacht und Liste der Zählpunkte. Wird nur erzeugt, wenn das
Mitglied die Netzbetreiber-Vollmacht (PROJ-44) erteilt hat.

## Hintergrund

Die EEG-Verwaltung muss aktuell die Netzbetreiber-Kommunikation manuell
mit Daten aus mehreren Quellen anreichern (Kundennummer, Inventarnummer,
Vollmachtstext, Zählpunkt-Liste). Eine fertig generierte PDF-Seite
spart Verwaltungs-Aufwand und reduziert Fehlerquellen beim Abtippen.

## Scope

### Datenmodell

Zwei neue Spalten auf `member_onboarding.application`:

- `network_operator_customer_number` (TEXT, NULLABLE) — Kundennummer
  beim zuständigen Netzbetreiber
- `meter_inventory_number` (TEXT, NULLABLE) — Inventarnummer eines
  Zählers (per-Mitglied, ein Wert pro Antrag — auch wenn mehrere
  Zählpunkte)

Werte bleiben NULL, wenn die Vollmacht nicht erteilt wurde.

### Configurable-Fields-Integration (PROJ-8)

Zwei neue Einträge im `field_config`-Mechanismus:

- `network_operator_customer_number`
- `meter_inventory_number`

State pro Feld: `hidden | optional | required`, konfigurierbar pro EEG
über das Admin-Settings-UI. Default `hidden` (legacy-EEGs ohne aktive
Konfiguration sehen die Felder nicht).

Sichtbarkeits-Logik:
- Feld wird im Public-Formular **nur** angezeigt, wenn:
  1. `field_config` ≠ `hidden` UND
  2. Die `networkOperatorAuthorization`-Checkbox vom User aktiviert wurde
- Sobald die Vollmachts-Checkbox abgehakt wird, blenden sich die zwei
  Felder ein. Wird die Checkbox wieder deaktiviert, blenden sie sich
  aus und etwaige Eingaben werden verworfen (analog zu anderen
  konditionalen Bereichen).

### Frontend (registration-form.tsx)

- Zwei neue Inputs im Bereich der Netzbetreiber-Vollmacht (PROJ-44)
- Conditional render: `networkOperatorAuthorization === true &&
  fs(name) !== "hidden"`
- Required-Validierung: wenn `fs(name) === "required"` UND
  `networkOperatorAuthorization === true`, dann Pflicht.
  **Achtung:** Validierung muss `isPerson`/conditional-render-konsistent
  sein, sonst gleicher Submit-Hänger-Bug wie bei `birth_date` (siehe
  Commit 72d380b).

### Admin-UI

- Beide Werte im Admin-Edit-Form sichtbar und editierbar (auch nach
  Submit)
- Anzeige in der Anwendungs-Detailansicht

### PDF-Seite

Neue Seite im Beitrittsbestätigungs-PDF (PROJ-21), gerendert wenn:

```
application.network_operator_authorization === true
```

**Layout (laut Referenz-Bild):**

1. Überschrift: „Informationen für den Netzbetreiber"
2. Header-Zeile mit zwei Werten:
   - „Netzbetreiber Kundennummer: `<value>`"
   - „Inventarnummer eines Zählers: `<value>`"
   (Leerer Wert: Label ohne Wert; keine Sonderbehandlung)
3. Vollmachts-Block mit Checkbox-Symbol `[X]` + Vollmacht-Text wie in
   PROJ-44 definiert
4. „Vollmacht erteilt am `<application.submitted_at>`"
   - Format: `DD.MM.YYYY HH:MM` in lokaler Zeitzone (Europe/Vienna)
5. Tabelle der Zählpunkte:
   - Spalten: Zählpunktnummer | Adresse | Typ | TF
   - Quelle: `metering_point`-Datensätze des Antrags
   - Zählpunkt-Nr mit Gruppen-Spacing (wie auf Bild: "AT 003000 00000 …")
   - Adresse: zwei Zeilen (Straße + PLZ Stadt)
   - Typ: `CNSM` (Consumption) / `GNRT` (Generation) — kurzes Kürzel
     statt Vollform für Platzersparnis
   - TF: `participation_factor` in Prozent

### API

- Public `POST /api/public/applications`: neue optionale Felder im
  Request-Body, Persistierung wenn `network_operator_authorization=true`
- Admin `GET/PUT /api/admin/applications/{id}`: Felder in Response/
  Update-Payload

## Geklärte Entscheidungen

- **Erfassung:** Public-Formular, konditional sichtbar
- **Granularität:** Per-Mitglied (ein Wert pro Antrag), nicht per-Zählpunkt
- **PDF-Bedingung:** Nur wenn Vollmacht erteilt
- **Pflichtstatus:** Konfigurierbar pro EEG via `field_config`
- **Timestamp:** `application.submitted_at`

## Offene Fragen (vor Implementation kurz prüfen)

- Spalten-Position im PDF-Tabellen-Layout: ist das Bild final (Zählpunkt-
  Nr, Adresse, Typ, TF), oder soll z. B. die Energiemenge/PV-Leistung
  auch mit?
- Inventarnummer ist semantisch eines Zählers, aber per-Mitglied
  gespeichert — was, wenn das Mitglied mehrere Zähler hat? Bleibt es
  bei einer Sammel-Inventarnummer oder soll's später per-Zählpunkt
  werden? Aktuell folgen wir dem Bild = per-Mitglied.
- E-Mail-Versand: bekommt die EEG die Netzbetreiber-Info als separates
  Attachment oder ist sie Teil des bestehenden Approval-PDFs als
  letzte Seite?

## Implementierungs-Schritte

1. **DB-Migration**: zwei neue Spalten auf `application`-Tabelle
2. **Backend-Schema + Repository**: Felder im Go-Modell, Repo-CRUD,
   API-Endpoints (public POST + admin PATCH)
3. **Field-Config-Eintrag**: Default-State, Admin-UI für Konfiguration
4. **Frontend-Form**: Conditional render der zwei Felder + Validierung
5. **Admin-Edit-Form**: Felder sichtbar/editierbar
6. **PDF-Generator**: neue Seite konditional rendern
7. **Tests**: Unit-Tests für Schema-Validierung, Snapshot-Test PDF
8. **CHANGELOG + INDEX.md** aktualisieren

Geschätzter Aufwand: 2–3 Stunden in mehreren sauber geschnittenen Commits.
