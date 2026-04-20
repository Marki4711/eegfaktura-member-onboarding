# PROJ-7: Mitgliedstypen

**Status:** 🔵 Planned
**Created:** 2026-04-20
**Last Updated:** 2026-04-20

## Überblick

EEG-Mitglieder sind nicht ausschließlich Privatpersonen. Es gibt vier steuerlich relevante Mitgliedstypen mit unterschiedlichen Pflichtfeldern. Das Registrierungsformular und der gesamte Backend-Stack müssen alle vier Typen korrekt erfassen und validieren.

## User Stories

**US-1:** Als Kleinunternehmer möchte ich mich als Privatperson/Kleinunternehmer registrieren, damit meine Anmeldung meiner tatsächlichen steuerlichen Situation entspricht.

**US-2:** Als pauschalierter Landwirt möchte ich den korrekten Mitgliedstyp auswählen, damit die EEG meine steuerlichen Besonderheiten (13 % USt.) kennt.

**US-3:** Als Vertreter einer Gemeinde möchte ich eine Gemeinde als Mitglied anmelden und optional eine UID-Nummer angeben, falls die Gemeinde umsatzsteuerpflichtig ist.

**US-4:** Als Vertreter eines Unternehmens möchte ich eine Firma oder einen Verein anmelden und dabei Firmennamen, UID-Nummer und Firmenbuch-/Vereinsnummer angeben, damit alle steuerlich relevanten Daten vollständig erfasst sind.

**US-5:** Als Admin möchte ich den Mitgliedstyp und alle zugehörigen Felder in der Detailansicht sehen, damit ich den Antrag korrekt prüfen und verarbeiten kann.

## Scope

Vier Mitgliedstypen mit typspezifischen Pflichtfeldern:

| Typ | Kurzname | USt. | Namensfelder | Geburtsdatum | UID | Reg.Nr. |
|-----|----------|------|--------------|--------------|-----|---------|
| Privat / Kleinunternehmer | `private` | 0 % | Vorname + Nachname | Pflicht | — | — |
| Pauschalierter Landwirt | `farmer` | 13 % | Vorname + Nachname | Pflicht | — | — |
| Gemeinde | `municipality` | variabel | Organisationsname | — | optional | — |
| Unternehmen | `company` | 20 % | Firmenname | — | Pflicht | Pflicht |

**Neue Felder (für alle Typen, je nach Typ ausgefüllt oder leer):**
- `member_type` – Mitgliedstyp (immer Pflicht)
- `company_name` – Organisationsname (Pflicht für `municipality` und `company`)
- `uid_number` – UID-Nummer / Umsatzsteuer-ID (Pflicht für `company`, optional für `municipality`)
- `register_number` – Firmenbuch- oder Vereinsnummer (Pflicht für `company`)

**Geänderte Felder:**
- `firstname`, `lastname` – werden nullable: Pflicht bei `private` und `farmer`, nicht vorhanden bei `municipality` und `company`
- `birth_date` – bleibt nullable: Pflicht bei `private` und `farmer`, nicht vorhanden bei `municipality` und `company`

## Non-Goals

- Steuerliche Berechnung oder Fakturierung
- Unterschiedliche Tarifzuordnung je Typ
- Validierung der UID gegen externe Dienste (MIAS etc.)
- Separate Ansprechpersonen-Felder für Organisationen

## Acceptance Criteria

### Formular: Typ-Auswahl

- [ ] Am Beginn des Formulars gibt es eine Typ-Auswahl mit vier Optionen: Privat/Kleinunternehmer, Pauschalierter Landwirt, Gemeinde, Unternehmen
- [ ] Die Standardauswahl ist „Privat / Kleinunternehmer"
- [ ] Die Typ-Auswahl ist sichtbar und klar beschriftet (inkl. USt.-Hinweis)
- [ ] Nach Auswahl des Typs passen sich die Formularfelder sofort dynamisch an

### Formular: Felder je Typ

- [ ] Typ `private` und `farmer`: Felder Vorname, Nachname, Geburtsdatum sind sichtbar und Pflichtfelder; Organisationsname, UID, Reg.Nr. werden nicht angezeigt
- [ ] Typ `municipality`: Feld Organisationsname ist sichtbar und Pflichtfeld; UID ist sichtbar und optional; Vorname/Nachname/Geburtsdatum/Reg.Nr. werden nicht angezeigt
- [ ] Typ `company`: Felder Firmenname, UID-Nummer und Firmenbuch-/Vereinsnummer sind sichtbar und Pflichtfelder; Vorname/Nachname/Geburtsdatum werden nicht angezeigt
- [ ] Ein Wechsel des Typs setzt die typspezifischen Felder zurück (kein Daten-Carry-over zwischen Person- und Organisationsfeldern)

### Backend: Validierung

- [ ] `member_type` ist immer ein Pflichtfeld mit einem der vier erlaubten Werte
- [ ] Server-seitige Validierung prüft für jeden Typ die korrekten Pflichtfelder (kein Verlass auf Client-Validierung allein)
- [ ] Für `company`: `uid_number` und `register_number` müssen nicht leer sein
- [ ] Für `municipality`: `uid_number` ist optional
- [ ] Für `private` und `farmer`: `firstname`, `lastname` und `birth_date` müssen vorhanden sein; `company_name`, `uid_number` und `register_number` werden ignoriert (falls übergeben)
- [ ] Für `municipality` und `company`: `company_name` muss vorhanden sein; `firstname`, `lastname` und `birth_date` werden ignoriert
- [ ] Ungültige `member_type`-Werte werden mit `400 Validation Error` abgelehnt

### Backend: Persistenz

- [ ] `member_type` wird in der Datenbank gespeichert
- [ ] `company_name`, `uid_number`, `register_number` werden gespeichert (nullable für nicht zutreffende Typen)
- [ ] `firstname` und `lastname` sind in der Datenbank nullable (Pflicht nur über Anwendungslogik für Person-Typen)
- [ ] Bestehende Anträge ohne `member_type` erhalten per Migration den Defaultwert `private`

### Admin: Detailansicht

- [ ] Der Mitgliedstyp ist in der Admin-Detailansicht sichtbar (als lesbares Label, nicht als technischer Key)
- [ ] Bei Organisationen wird Organisationsname, UID (falls vorhanden) und Reg.Nr. (falls vorhanden) angezeigt
- [ ] Bei Personen wird Vorname, Nachname, Geburtsdatum angezeigt (wie bisher)

### Admin: Bearbeitungsformular

- [ ] Das Admin-Bearbeitungsformular zeigt die dem Typ entsprechenden Felder an
- [ ] Der Admin kann den Mitgliedstyp eines Antrags ändern (mit entsprechender Anpassung der Pflichtfelder)

## Edge Cases

### Typ-Wechsel im Formular
- Wechselt der User von „Privatperson" auf „Unternehmen", werden Vorname/Nachname/Geburtsdatum geleert und ausgeblendet
- Wechselt er zurück, sind die Felder leer (kein automatisches Wiederherstellen)
- Validierung läuft immer gegen den aktuell ausgewählten Typ

### Fehlende Pflichtfelder je nach Typ
- Reicht ein `company`-Antrag ohne `uid_number` ein, bekommt er einen klaren Validierungsfehler mit Feldangabe
- Reicht ein `private`-Antrag ohne `firstname` ein, bekommt er einen klaren Validierungsfehler

### UID-Format
- Format-Validierung der UID (für AT: `ATU` + 8 Ziffern) wünschenswert, aber nicht zwingend für V1
- Falls keine Format-Validierung: leere oder offensichtlich ungültige Werte werden durch Pflichtfeld-Check abgefangen

### Bestehende Anträge (Datenmigration)
- Alle Anträge, die vor PROJ-7 erstellt wurden, bekommen `member_type = 'private'`
- `company_name`, `uid_number`, `register_number` bleiben NULL für Altanträge
- `firstname` und `lastname` sind bei Altanträgen immer gefüllt (kein Migration-Problem)

### Submit-Validierung
- `SubmitApplication` prüft die Pflichtfelder ebenfalls typ-abhängig (nicht nur CreateApplication)
- Ein Draft, der vor PROJ-7 erstellt und jetzt gesubmittet wird, ist immer `private` → bestehende Pflichtfelder gelten

## Betroffene Komponenten

**Datenbank:**
- Migration: `firstname`, `lastname` auf nullable ändern
- Migration: neue Spalten `member_type`, `company_name`, `uid_number`, `register_number`
- Migration: Default `member_type = 'private'` für Bestandsdaten

**Backend (`internal/`):**
- `shared/models.go`: Application-Struct um neue Felder erweitern; `firstname`/`lastname` auf `*string` ändern
- `shared/requests.go`: CreateApplicationRequest, UpdateApplicationRequest, AdminUpdateApplicationRequest
- `application/application_service.go`: typ-abhängige Validierung in Create, Update, Submit
- `application/application_repo.go`: SQL-Queries anpassen (INSERT, SELECT, UPDATE)
- `application/admin_service.go`: AdminUpdate typ-abhängige Felder

**Frontend (`src/`):**
- `src/lib/api.ts`: Neue Felder in Request- und Response-Typen
- `src/components/registration-form.tsx`: Typ-Auswahl, dynamische Felder, Zod-Schema
- `src/components/admin-application-detail.tsx`: Typ-Label und neue Felder in der Detailansicht
- `src/components/admin-edit-form.tsx`: Typ-abhängige Felder im Bearbeitungsformular

## Abhängigkeiten

- Setzt PROJ-1 (Public Registration) voraus — erweitert dessen Datenmodell und Formular
- Setzt PROJ-2/PROJ-3 (Admin Review + Frontend) voraus — erweitert Admin-Ansichten

## Definition of Done

- [ ] Alle vier Mitgliedstypen können über das Formular erfasst werden
- [ ] Typspezifische Pflichtfelder werden client- und server-seitig validiert
- [ ] Datenbank-Migration legt neue Spalten an und setzt Defaults für Bestandsdaten
- [ ] Admin-Detailansicht zeigt Mitgliedstyp und zugehörige Felder korrekt an
- [ ] Admin kann Typ und Felder bearbeiten
- [ ] Bestehende Anträge (Typ `private`) funktionieren ohne Datenverlust weiter
- [ ] Go build und TypeScript Compilation fehlerfrei
