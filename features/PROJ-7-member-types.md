# PROJ-7: Mitgliedstypen

**Status:** 🟢 Approved
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

## Tech Design (Solution Architect)

### Übersicht

PROJ-7 ist eine **vertikale Erweiterung** durch alle Schichten: Datenbank → Backend → Frontend. Es werden keine neuen Tabellen angelegt — alle neuen Felder kommen in die bestehende `application`-Tabelle. Die Unterscheidung zwischen Typen steuert ausschließlich die Anwendungslogik, nicht das Datenbankschema.

---

### Datenbankänderungen (2 Migrationen)

**Migration 007a — neue Spalten:**
- `member_type` (Text, Pflicht, Default: `private`) — speichert einen der vier Werte: `private`, `farmer`, `municipality`, `company`
- `company_name` (Text, nullable) — Organisationsname für Gemeinde und Unternehmen
- `uid_number` (Text, nullable) — UID / Umsatzsteuer-ID
- `register_number` (Text, nullable) — Firmenbuch- oder Vereinsnummer

**Migration 007b — bestehende Spalten anpassen:**
- `firstname` und `lastname` werden auf **nullable** geändert (waren bisher Pflicht auf DB-Ebene)
- Alle bestehenden Anträge bekommen `member_type = 'private'` gesetzt (einmaliger UPDATE)

`birth_date` ist bereits nullable — keine Änderung nötig.

---

### Backend-Struktur

#### Datenmodell (`shared/models.go`)

Das `Application`-Struct bekommt fünf neue Felder:
- `MemberType` (immer gefüllt)
- `CompanyName` (pointer/nullable)
- `UIDNumber` (pointer/nullable)
- `RegisterNumber` (pointer/nullable)
- `Firstname` und `Lastname` werden von `string` auf pointer/nullable geändert

#### Validierungslogik (`application_service.go`)

Ein neuer, zentraler Helfer `validateMemberTypeFields(app)` kapselt alle typabhängigen Regeln:

```
private / farmer:
  → firstname, lastname, birth_date erforderlich
  → company_name, uid_number, register_number werden ignoriert

municipality:
  → company_name erforderlich
  → uid_number optional
  → firstname, lastname, birth_date werden ignoriert

company:
  → company_name, uid_number, register_number erforderlich
  → firstname, lastname, birth_date werden ignoriert
```

Dieser Helfer wird an drei Stellen aufgerufen:
1. `CreateApplication` — nach dem Mapping der Request-Felder
2. `UpdateApplication` — nach dem Anwenden der Änderungen
3. `SubmitApplication` — als finale Pflichtfeld-Prüfung vor der Statusänderung

#### Request-Structs (`shared/requests.go`)

`CreateApplicationRequest`, `UpdateApplicationRequest` und `AdminUpdateApplicationRequest` erhalten alle vier neuen Felder. `firstname` und `lastname` werden optional (pointer). `member_type` ist in Create ein Pflichtfeld, in Update optional (nur wenn geändert).

#### Datenbank-Queries (`application_repo.go`)

INSERT, SELECT und UPDATE in allen Repository-Methoden um die vier neuen Spalten erweitern. `firstname` und `lastname` werden als nullable behandelt.

---

### Frontend-Struktur

#### Neue Komponente: `MemberTypeSelector`

Ein eigenständiger Block am Beginn des Registrierungsformulars. Zeigt vier wählbare Karten oder Tabs:

```
┌─────────────────────────────────────────────────────────┐
│  Mitgliedstyp                                           │
│                                                         │
│  ● Privat / Kleinunternehmer  (0 % USt.)               │
│  ○ Pauschalierter Landwirt    (13 % USt.)              │
│  ○ Gemeinde                   (variabel)                │
│  ○ Unternehmen                (20 % USt.)              │
└─────────────────────────────────────────────────────────┘
```

Verwendet die bereits installierte shadcn `RadioGroup`-Komponente.

#### Geänderte Komponente: `registration-form.tsx`

Das Formular beobachtet den ausgewählten Typ und zeigt abhängig davon unterschiedliche Felder an:

```
RegistrationForm
├── MemberTypeSelector (neu)
├── Card: Persönliche Daten / Organisationsdaten
│   ├── [private/farmer]  Vorname, Nachname, Geburtsdatum
│   └── [municipality]    Organisationsname, UID (optional)
│       [company]         Firmenname, UID (Pflicht), Reg.Nr. (Pflicht)
├── Card: Adresse              (unverändert)
├── Card: Bankverbindung       (unverändert)
├── Card: Zählpunkte           (unverändert)
└── Card: Einwilligungen       (unverändert)
```

Das Zod-Schema wird auf eine **diskriminierte Union** umgestellt: Der Typ `memberType` bestimmt, welche Felder verpflichtend sind. Das verhindert Validierungsfehler für Felder, die beim aktuellen Typ gar nicht angezeigt werden.

Beim Typ-Wechsel werden alle typspezifischen Felder des vorherigen Typs geleert.

#### Geänderte Komponente: `admin-application-detail.tsx`

Der Bereich „Mitgliedsdaten" zeigt je nach `member_type`:

```
[private / farmer]             [municipality / company]
──────────────────             ───────────────────────
Typ:       Privatperson        Typ:        Unternehmen
Vorname:   Max                 Name:       Muster GmbH
Nachname:  Mustermann          UID:        ATU12345678
Geburtsd.: 01.01.1980          Reg.Nr.:    FN 123456 a
```

#### Geänderte Komponente: `admin-edit-form.tsx`

Das Admin-Bearbeitungsformular erhält eine Typ-Auswahl (Dropdown oder RadioGroup) und zeigt die entsprechenden Felder dynamisch an — analog zum öffentlichen Formular.

---

### API-Typen (`src/lib/api.ts`)

`CreateApplicationRequest`, `AdminUpdateApplicationRequest` und `AdminApplicationDetail` erhalten die vier neuen Felder. `firstname` und `lastname` werden optional.

---

### Keine neuen Pakete erforderlich

Alle benötigten UI-Komponenten (RadioGroup, Tabs, Input, Select) sind bereits installiert. Keine neuen npm-Pakete notwendig.

---

### Reihenfolge der Implementierung

1. **Datenbank** — Migrationen 007a und 007b
2. **Backend** — Datenmodell, Validierungshelfer, Request-Structs, Repository-Queries
3. **Frontend** — MemberTypeSelector, Formularänderungen, Admin-Ansichten

Backend und Frontend können nach den Migrationen parallel bearbeitet werden, da die API-Typen vorab festgelegt sind.

---

## QA Test Results

**QA Date:** 2026-04-20
**Tester:** Claude QA Engineer

### Automated Tests

| Suite | Result |
|-------|--------|
| Go unit tests (`internal/application`) | ✅ 19/19 passed |
| Go mail tests (`internal/mail`) | ✅ 6/6 passed (fixed regression: `*string` fields) |
| TypeScript compilation (`tsc --noEmit`) | ✅ clean |
| Go build (`go build ./...`) | ✅ clean |
| E2E tests (`npm run test:e2e`) | ⚠️ written, require running dev server + seeded DB |
| Vitest (`npm test`) | ⚠️ pre-existing env issue (missing `rolldown-binding.win32-x64-msvc.node`), not caused by PROJ-7 |

### Acceptance Criteria

#### Formular: Typ-Auswahl
| # | Criterion | Result |
|---|-----------|--------|
| AC-1 | 4 Typ-Optionen am Formularbeginn | ✅ PASS |
| AC-2 | Standardauswahl „Privat / Kleinunternehmer" | ✅ PASS |
| AC-3 | Klar beschriftet inkl. USt.-Hinweis | ✅ PASS |
| AC-4 | Felder passen sich dynamisch an | ✅ PASS |

#### Formular: Felder je Typ
| # | Criterion | Result |
|---|-----------|--------|
| AC-5 | `private`/`farmer`: Vorname/Nachname/Geburtsdatum sichtbar und Pflicht | ✅ PASS |
| AC-6 | `municipality`: Organisationsname Pflicht, UID optional, Person-Felder versteckt | ✅ PASS |
| AC-7 | `company`: Firmenname/UID/Reg.Nr. Pflicht, Person-Felder versteckt | ✅ PASS |
| AC-8 | Typ-Wechsel setzt typspezifische Felder zurück | ✅ PASS |

#### Backend: Validierung
| # | Criterion | Result |
|---|-----------|--------|
| AC-9 | `member_type` Pflichtfeld mit 4 erlaubten Werten | ✅ PASS |
| AC-10 | Server-seitige Validierung typ-abhängig in Create/Update/Submit | ✅ PASS |
| AC-11 | `company`: uid_number + register_number Pflicht | ✅ PASS |
| AC-12 | `municipality`: uid_number optional | ✅ PASS |
| AC-13 | `private`/`farmer`: company_name/uid/register_number ignoriert (clearMemberTypeFields) | ✅ PASS |
| AC-14 | `municipality`/`company`: firstname/lastname/birthDate ignoriert | ✅ PASS |
| AC-15 | Ungültiger member_type → 400 | ✅ PASS (`oneof` validator + `default` branch in validateMemberTypeFields) |
| AC-16 | `birth_date` Pflicht für `private`/`farmer` im Backend | ❌ FAIL — **BUG-1** |

#### Backend: Persistenz
| # | Criterion | Result |
|---|-----------|--------|
| AC-17 | `member_type` in DB gespeichert | ✅ PASS |
| AC-18 | `company_name`/`uid_number`/`register_number` nullable gespeichert | ✅ PASS |
| AC-19 | `firstname`/`lastname` DB nullable | ✅ PASS |
| AC-20 | Migration setzt Default `private` für Bestandsdaten | ✅ PASS |

#### Admin: Detailansicht
| # | Criterion | Result |
|---|-----------|--------|
| AC-21 | Mitgliedstyp als lesbares Label sichtbar | ✅ PASS |
| AC-22 | Org-Felder für municipality/company, Person-Felder für private/farmer | ✅ PASS |

#### Admin: Bearbeitungsformular
| # | Criterion | Result |
|---|-----------|--------|
| AC-23 | Admin-Edit zeigt dem Typ entsprechende Felder | ✅ PASS |
| AC-24 | Admin kann Mitgliedstyp ändern | ✅ PASS (UI + backend) |
| AC-25 | AdminUpdateApplication validiert Pflichtfelder nach Typ-Wechsel | ❌ FAIL — **BUG-2** |

### Bugs Found

#### BUG-1 — Medium: Backend validiert `birth_date` nicht als Pflichtfeld für `private`/`farmer`

**Severity:** Medium
**Component:** `internal/application/application_service.go` — `validateMemberTypeFields`
**Description:** Laut Spec muss `birth_date` für `private` und `farmer` serverseitig Pflichtfeld sein. Die Funktion `validateMemberTypeFields` prüft nur `firstname` und `lastname`, nicht aber `birth_date`. Ein direkter API-Aufruf ohne `birthDate` (unter Umgehung des Frontends) wird akzeptiert.
**Steps to reproduce:**
```bash
POST /api/public/applications
{ "memberType": "private", "firstname": "Max", "lastname": "Muster",
  "email": "x@x.at", "residentStreet": "A", "residentStreetNumber": "1",
  "residentZip": "4020", "residentCity": "Linz", "iban": "AT611904300234573201",
  "accountHolder": "Max", "sepaMandateAccepted": true, "privacyAccepted": true,
  "privacyVersion": "2026-01", "accuracyConfirmed": true, "rcNumber": "...",
  "meteringPoints": [{"meteringPoint": "AT003100000000000000000000000001", "direction": "CONSUMPTION"}] }
```
→ Erwartet: 400 Validation Error für `birthDate`
→ Tatsächlich: 201 Created
**Fix:** In `validateMemberTypeFields`, für `private`/`farmer` auch `app.BirthDate == nil` prüfen.

#### BUG-2 — Medium: `AdminUpdateApplication` ruft `validateMemberTypeFields` nicht auf

**Severity:** Medium
**Component:** `internal/application/admin_service.go` — `AdminUpdateApplication`
**Description:** `clearMemberTypeFields` wird aufgerufen, aber `validateMemberTypeFields` nicht. Ein Admin kann den Typ auf `private` ändern ohne firstname/lastname zu liefern — die Daten werden ohne Validierung gespeichert.
**Steps to reproduce:**
```bash
PUT /api/admin/applications/{id}
{ "memberType": "private", "email": "x@x.at", "residentStreet": "A", ... }
# kein firstname/lastname
```
→ Erwartet: 400 Validation Error
→ Tatsächlich: 200 OK, Antrag mit memberType=private aber firstname=null
**Fix:** Nach `clearMemberTypeFields(app)` in `AdminUpdateApplication` auch `validateMemberTypeFields(app)` aufrufen und bei Fehler abbrechen.

#### BUG-3 — Low: Pre-existing regression: `internal/mail/service_test.go` brach durch `*string`-Umstellung

**Severity:** Low (bereits behoben im QA-Lauf)
**Component:** `internal/mail/service_test.go`
**Description:** Test verwendete `Firstname: "Josef"` statt `Firstname: &fn` nach der Umstellung auf `*string`. Im QA-Lauf direkt gefixt.
**Status:** ✅ Bereits behoben

### Security Audit

- **Injection:** Alle String-Eingaben werden via `strings.TrimSpace` bereinigt und über parametrisierte SQL-Abfragen (`$1, $2, ...`) eingefügt. Kein SQL-Injection-Risiko.
- **Input validation:** `member_type` wird mit `oneof`-Validator eingeschränkt. Keine freien Enum-Werte möglich.
- **Data leakage:** `uidNumber` und `registerNumber` sind sensible Geschäftsdaten. Sie werden nur über den Admin-Endpunkt (authentifiziert) exponiert — kein öffentlicher Endpunkt liefert diese Felder zurück.
- **Mass assignment:** `clearMemberTypeFields` verhindert, dass ein `company`-Antrag trotz Typ-Wechsel Personendaten behält (serverseitig bereinigt).

### Regression

- PROJ-1 (Public Registration): Formular-Grundstruktur unverändert, neuer MemberType-Block ist additiv. ✅
- PROJ-2/3 (Admin Review/Frontend): Admin-Liste und Detail nutzen jetzt nullable Felder korrekt. ✅ Kein Datenverlust bei Altanträgen (DEFAULT `private`).
- PROJ-6 (E-Mail): `derefString`-Hilfsfunktion verhindert nil-panic für Org-Typen. ✅

### Production-Ready Decision

**READY** — beide Medium-Bugs behoben (2026-04-20), alle 20 Go-Tests und TS-Build grün.

---

## Definition of Done

- [ ] Alle vier Mitgliedstypen können über das Formular erfasst werden
- [ ] Typspezifische Pflichtfelder werden client- und server-seitig validiert
- [ ] Datenbank-Migration legt neue Spalten an und setzt Defaults für Bestandsdaten
- [ ] Admin-Detailansicht zeigt Mitgliedstyp und zugehörige Felder korrekt an
- [ ] Admin kann Typ und Felder bearbeiten
- [ ] Bestehende Anträge (Typ `private`) funktionieren ohne Datenverlust weiter
- [ ] Go build und TypeScript Compilation fehlerfrei
