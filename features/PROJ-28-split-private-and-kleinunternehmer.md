# PROJ-28: Trennung von „Privat" und „Kleinunternehmer"

## Status: In Progress
**Created:** 2026-05-12
**Last Updated:** 2026-05-12 (Implementation started)

## Dependencies
- Requires: PROJ-7 (Mitgliedstypen) — erweitert dessen Mitgliedstypen-Modell
- Requires: PROJ-4 (Core Import) — passt das Name-Mapping in `internal/importing/payload.go` an
- Requires: PROJ-3 (Admin Frontend UI) — Edit-Form und Detailansicht

## Hintergrund

Aktuell werden **Privatperson** und **Kleinunternehmer** unter dem Mitgliedstyp `private` zusammengefasst (siehe PROJ-7 — Optionslabel „Privat / Kleinunternehmer"). Beide nutzen dieselben Eingabefelder (Vorname + Nachname + Geburtsdatum) und werden im eegFaktura-Core unter `businessRole = EEG_PRIVATE` angelegt.

Das ist für Kleinunternehmer falsch: ein Kleinunternehmer tritt **mit Firmennamen** auf, hat aber **keine UID und keine Firmenbuchnummer** (sonst wäre er ein normales `company`-Mitglied). Im eegFaktura-Core soll der Firmenname dann nach der bestehenden Convention im Feld `firstname` landen — wie bei `company`/`association`/`municipality`.

## User Stories

- Als **Kleinunternehmer** möchte ich bei der Registrierung einen eigenen Mitgliedstyp wählen, der nur den Firmennamen abfragt, sodass meine Anmeldung meiner tatsächlichen Auftrittsform entspricht.
- Als **Privatperson** möchte ich klar von Kleinunternehmern unterschieden werden, sodass mein Mitgliedstyp eindeutig „Privatperson" ist und nicht mit Unternehmern vermischt wird.
- Als **EEG-Admin** möchte ich Kleinunternehmer im Admin-UI als eigenen Typ erkennen, sodass ich z.B. für Berichte oder Kommunikation gezielt nach ihnen filtern kann.
- Als **EEG-Admin** möchte ich, dass beim Import in eegFaktura für Kleinunternehmer der Firmenname im Feld `firstname` landet (wie bei Unternehmen/Gemeinden), sodass die Anzeige in eegFaktura konsistent ist.

## Acceptance Criteria

### Neuer Mitgliedstyp `sole_proprietor`
- [ ] In `shared/models.go` existiert die Konstante `MemberTypeSoleProprietor` mit dem String-Wert `"sole_proprietor"`
- [ ] `oneof`-Validator und `member_type`-Migration akzeptieren den neuen Wert
- [ ] `MemberTypePrivate` bleibt erhalten und bedeutet ab sofort **ausschließlich** Privatperson (kein Kleinunternehmer)

### Formular: Typ-Auswahl
- [ ] Die Typ-Auswahl im Registrierungsformular zeigt **fünf** Optionen statt bisher vier:
  1. Privatperson (0 % USt.)
  2. Kleinunternehmer (0 % USt.)
  3. Pauschalierter Landwirt (13 % USt.)
  4. Gemeinde
  5. Unternehmen / Verein (20 % USt.)
- [ ] Das alte Label „Privat / Kleinunternehmer" wird durch „Privatperson" ersetzt
- [ ] Default-Auswahl bleibt „Privatperson"

### Formular: Felder je Typ
- [ ] Typ `private`: Felder Vorname, Nachname, Geburtsdatum sind sichtbar und Pflicht (wie bisher)
- [ ] Typ `sole_proprietor`: **nur** das Feld Firmenname ist sichtbar und Pflicht. Vorname, Nachname, Geburtsdatum, UID und Firmenbuchnummer werden **nicht** angezeigt
- [ ] Wechsel zwischen `private` und `sole_proprietor` setzt typspezifische Felder zurück (kein Daten-Carry-over)
- [ ] Die übrigen Typen (`farmer`, `municipality`, `company`) bleiben unverändert

### Backend: Validierung
- [ ] Für `sole_proprietor`: `company_name` ist Pflicht; `firstname`, `lastname`, `birth_date`, `uid_number`, `register_number` werden ignoriert (falls übergeben)
- [ ] Für `private`: unverändert — `firstname`, `lastname`, `birth_date` Pflicht
- [ ] `validateMemberTypeFields` und `clearMemberTypeFields` werden um den neuen Zweig erweitert
- [ ] Der `sole_proprietor`-Zweig wird in Create, Update **und** Submit konsistent geprüft (siehe PROJ-7 BUG-1/BUG-2)

### Backend: Import-Mapping in eegFaktura
- [ ] `mapPersonName` in `internal/importing/payload.go` behandelt `sole_proprietor` analog zu `company`/`association`/`municipality`: `company_name` landet in `firstname`, `lastname` bleibt leer
- [ ] `isNaturalPerson(sole_proprietor)` liefert `false` (damit das Name-Mapping greift)
- [ ] `mapBusinessRole(sole_proprietor)` liefert **`EEG_BUSINESS`** (Anzeige unter Firma-Tab in eegFaktura — siehe Open Question Q1)

### Admin-UI: Detailansicht und Edit-Form
- [ ] `admin-application-detail.tsx` zeigt für `sole_proprietor`:
  - Typ-Label: „Kleinunternehmer"
  - Datenblock: Firmenname (kein Vorname/Nachname/Geburtsdatum, keine UID, keine Reg.Nr.)
- [ ] `admin-edit-form.tsx` zeigt im Bearbeitungs-Modus dieselben Felder wie das öffentliche Formular für `sole_proprietor`
- [ ] Filter/Sortierung in der Antragsliste unterstützt `sole_proprietor` als eigenen Filterwert

### Excel-Export (PROJ-17) & PDF (PROJ-21)
- [ ] Excel-Export gibt `sole_proprietor` mit lesbarem Label aus (analog zu `company` etc.)
- [ ] Approval-PDF zeigt für `sole_proprietor` den Firmennamen analog zu Unternehmen
- [ ] E-Mail-Anrede bei `sole_proprietor`: neutral mit Firmennamen, nicht „Sehr geehrte/r Vor- Nachname" (siehe Open Question Q4)

### Migration & Rückwärtskompatibilität
- [ ] Bestehende Anträge mit `member_type = private` bleiben unverändert auf `private` (sie sind Privatpersonen — der Kleinunternehmer-Anteil unter den Altdaten ist im Onboarding-System nicht unterscheidbar; Admin müsste manuell umklassifizieren bei Bedarf)
- [ ] Keine automatische Daten-Migration alter Anträge — siehe Open Question Q2
- [ ] Schema-Migration nur additiv: ein zusätzlich erlaubter Wert im `member_type`-CHECK/`oneof`, keine neuen Spalten

## Edge Cases

- Was passiert, wenn ein bestehender `private`-Antrag im Admin-Edit von `private` auf `sole_proprietor` umgestellt wird? → Vorname/Nachname/Geburtsdatum werden serverseitig durch `clearMemberTypeFields` geleert; `company_name` muss gefüllt sein, sonst 400. Bestätigung im UI: Hinweis „Personendaten werden entfernt"
- Was passiert, wenn ein Kleinunternehmer-Antrag importiert wird, bevor der Admin einen Tarif (PROJ-27) zugewiesen hat? → wie bei `company` heute: Import läuft, Tarif bleibt leer und wird in eegFaktura nachgepflegt — keine Sonderbehandlung
- Was passiert, wenn ein Kleinunternehmer auch eine UID hat (z.B. weil er später umsatzsteuerpflichtig wird)? → er gehört dann zum Typ `company`, nicht `sole_proprietor` — Admin klassifiziert um
- Was passiert bei der externen Registrierungs-API (PROJ-13)? → akzeptiert `member_type = sole_proprietor` mit denselben Pflichtfeldern wie das öffentliche Formular

## Technical Requirements

- **Konsistenz:** Mapping-Logik zwischen Onboarding-`member_type` und eegFaktura-`businessRole` ist an einer Stelle (`internal/importing/payload.go`) — keine Duplikate
- **Tests:** `payload_test.go` bekommt einen Testfall für `sole_proprietor` (BusinessRole + Name-Mapping). `application_service_test.go` bekommt Validierungs-Cases für create/update/submit
- **Rückwärtskompatibilität:** bestehende `private`-Anträge funktionieren ohne Datenverlust weiter; Excel/PDF/E-Mail rendern Altdaten unverändert
- **Beobachtbarkeit:** keine neuen Log-Felder erforderlich; das `member_type`-Feld ist bereits Bestandteil bestehender Logs

## Resolved Decisions

Alle Open Questions wurden am 2026-05-12 nach den jeweiligen Empfehlungen entschieden. Die ACs oben spiegeln diesen Stand bereits wider.

### Q1: `businessRole` für Kleinunternehmer — **RESOLVED**

**Entscheidung:** `EEG_BUSINESS`.

`businessRole` steuert in eegFaktura nur die UI-Anzeige (Tab „Privat" vs. „Firma"), nicht die Steuerlogik (kommt aus dem Tarif). Da das Name-Mapping (`company_name` → `firstname`) identisch zu `company`/`municipality`/`association` ist, ist `EEG_BUSINESS` die konsistente Wahl — der Eintrag erscheint unter „Firma" mit dem Firmennamen im Vornamen-Feld.

### Q2: Migration bestehender `private`-Anträge — **RESOLVED**

**Entscheidung:** Keine automatische Migration.

Im aktuellen Schema gibt es kein verlässliches Diskriminierungsmerkmal zwischen Privatpersonen und Kleinunternehmern unter den Altdaten. Heuristiken produzieren False-Positives. Bestandsanträge bleiben `private`; Admins klassifizieren bei Bedarf manuell um.

### Q3: Geburtsdatum für Kleinunternehmer — **RESOLVED**

**Entscheidung:** Geburtsdatum entfällt für `sole_proprietor`.

Die Userforderung ist eindeutig („nur Firmenname"). Falls eine EEG das Geburtsdatum doch benötigt, kann es über PROJ-8 (Konfigurierbare Felder pro EEG) optional aktiviert werden.

### Q4: E-Mail-Anrede für Kleinunternehmer — **RESOLVED**

**Entscheidung:** Neutrale Anrede „Sehr geehrte Damen und Herren" + Firmenname im Body, analog zu `company`/`association`/`municipality`.

### Q5: Sonderlogik in `mapPersonName` für Kleinunternehmer — **RESOLVED**

**Entscheidung:** Für `sole_proprietor` immer `company_name` → `firstname`, niemals Override durch ein eventuell gefülltes `firstname`-Feld.

Das öffentliche Formular zeigt das Vorname-Feld für `sole_proprietor` nicht an, daher kann es nicht legitim gefüllt sein. Eingehende `firstname`-Werte (z.B. über die externe API PROJ-13) werden ignoriert und in `mapPersonName` ausschließlich der `company_name` verwendet.

## Notes

- Spec ist klein im Umfang, aber berührt mehrere Schichten (DB-Constraint, Backend-Validation, Frontend-Form, Excel, PDF, E-Mail).
- Alle Open Questions sind entschieden — die Spec ist bereit für `/architecture`.
- Security-Review ist **nicht** erforderlich: keine neuen Endpoints, keine neuen Auth-Pfade, keine Schema-Änderung außer einem zusätzlich erlaubten Enum-Wert.

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)

### Übersicht

PROJ-28 ist eine **fokussierte, additive Erweiterung** des Mitgliedstyp-Modells aus PROJ-7. Es wird **kein** neues Datenbank-Schema benötigt — `member_type` ist eine TEXT-Spalte mit Default `private` und ohne CHECK-Constraint; die Werte-Einschränkung passiert auf Anwendungsebene (`oneof`-Tag im Go-Validator, diskriminierte Zod-Union im Frontend). Die Änderung besteht aus drei Strängen: Validatoren erweitern, Import-Mapping erweitern, Formulare und Output-Renderer erweitern.

### Datenbankänderungen

Keine. Bestehende `private`-Anträge bleiben unverändert (Q2). Falls in Zukunft ein CHECK-Constraint nachgezogen werden sollte (separates Hardening-Ticket), gehört das nicht zu PROJ-28.

### Backend-Struktur

#### Datenmodell (`internal/shared/models.go`)
- Neue Konstante: `MemberTypeSoleProprietor MemberType = "sole_proprietor"`
- Keine Struct-Änderung — `CompanyName` ist bereits `*string`

#### Request-Validatoren (`internal/shared/requests.go`)
- `oneof`-Tag auf `member_type` in `CreateApplicationRequest`, `UpdateApplicationRequest`, `AdminUpdateApplicationRequest` (und ggf. der externen API aus PROJ-13) um `sole_proprietor` erweitern

#### Validierungs- und Bereinigungslogik (`internal/application/application_service.go`)

Zwei bestehende Helfer aus PROJ-7 bekommen je einen neuen `case`:

- `validateMemberTypeFields(app)` → für `sole_proprietor`:
  - `company_name` Pflicht
  - `firstname`, `lastname`, `birth_date`, `uid_number`, `register_number` werden NICHT geprüft
- `clearMemberTypeFields(app)` → für `sole_proprietor`:
  - leert `firstname`, `lastname`, `birth_date`, `uid_number`, `register_number`
  - behält `company_name`

Aufrufstellen (`CreateApplication`, `UpdateApplication`, `SubmitApplication`, `AdminUpdateApplication`) sind bereits PROJ-7-konform verdrahtet — keine neuen Aufrufstellen.

#### Import-Mapping (`internal/importing/payload.go`)

Drei Funktionen brauchen den neuen Typ:

- `isNaturalPerson(sole_proprietor)` → `false` — damit das Company-Mapping in `mapPersonName` greift
- `mapBusinessRole(sole_proprietor)` → `EEG_BUSINESS` — fällt automatisch über die `isNaturalPerson`-Negation an, keine neue Branch nötig
- `mapPersonName(sole_proprietor)` → **expliziter Special-Case vor der bestehenden Logik**, weil Q5 Override durch ein bereits gefülltes `firstname` ausschließt:
  ```go
  if app.MemberType == shared.MemberTypeSoleProprietor {
      return derefString(app.CompanyName), ""
  }
  // bestehende company/association/municipality-Logik unverändert
  ```

Damit bleibt die bestehende Kontaktpersonen-Convention für `company`/`association`/`municipality` (Vorname behalten, falls gesetzt) intakt — nur für `sole_proprietor` wird sie ignoriert.

#### E-Mail-Service (`internal/mail/service.go`)
- Anrede-Funktion behandelt `sole_proprietor` analog zu `company`/`association`/`municipality`: neutrale Anrede + Firmenname (Q4)
- Templates `application_submitted_member.html`, `application_submitted_eeg.html`, Approval-Mail prüfen den Typ über dieselbe Helper-Funktion

#### Excel-Export (`internal/excel/generator.go`)
- Label-Map: `sole_proprietor` → `"Kleinunternehmer"` für die Spalte „Mitgliedstyp"
- Firmenname-Spalte erhält den Wert wie bei `company`

#### Approval-PDF (`internal/pdf/approval_pdf.go`)
- Renderer-Logik prüft `isNaturalPerson`; `sole_proprietor` rendert den Firmennamen-Block analog zu `company`

### Frontend-Struktur

#### TypeScript-Typen (`src/lib/api.ts`)
- `MemberType` Union erweitern: `"private" | "sole_proprietor" | "farmer" | "municipality" | "company" | "association"`
- Reihenfolge in der Union spiegelt die UI-Reihenfolge der Optionen wider

#### Registrierungsformular (`src/components/registration-form.tsx`)
- **Zod-Schema:** diskriminierte Union um den `sole_proprietor`-Zweig erweitern
  - Erforderlich: `companyName` (gleiche Regel wie bei `company`)
  - Nicht geprüft: `firstname`, `lastname`, `birthDate`, `uidNumber`, `registerNumber`
- **MemberTypeSelector:** fünfte RadioCard zwischen „Privatperson" und „Pauschalierter Landwirt"
- **Label-Refactor:** alte Option „Privat / Kleinunternehmer" → „Privatperson". Neue Option: „Kleinunternehmer (0 % USt.)"
- **Reset-Logik beim Typ-Wechsel:** existierende Helper-Funktion erkennt `sole_proprietor` als Ziel und leert Personenfelder; beim Wechsel weg von `sole_proprietor` wird `companyName` geleert (gleiches Pattern wie bei `company`)
- **Conditional Rendering:** für `sole_proprietor` wird nur das Firmenname-Eingabefeld angezeigt — keine UID-, keine Reg.Nr.-, keine Person-Felder

#### Admin-Detail-Ansicht (`src/components/admin-application-detail.tsx`)
- Daten-Block bei `sole_proprietor`:
  - Typ-Label: „Kleinunternehmer"
  - nur Firmenname
  - kein Vorname/Nachname/Geburtsdatum, keine UID, keine Reg.Nr.
- Filter/Tab-Logik (falls vorhanden) erkennt `sole_proprietor` als eigenen Filterwert

#### Admin-Edit-Form (`src/components/admin-edit-form.tsx`)
- Spiegel des Public-Forms: gleiche fünf Optionen, dieselbe Conditional-Field-Logik
- Existierende `private`-Anträge erscheinen weiterhin als „Privatperson" (Q2: keine Auto-Migration)

### Keine neuen Pakete erforderlich

Alle UI-Bausteine (RadioGroup, Input, Card) und Backend-Bibliotheken sind vorhanden. Keine zusätzliche npm- oder Go-Abhängigkeit.

### Test-Strategie

Bestehende Test-Module werden um den neuen Typ erweitert — kein neues Test-File:

- `internal/application/application_service_test.go`
  - `Create/Update/Submit` mit `memberType=sole_proprietor` + `companyName` → erfolgreich
  - `Create` mit `memberType=sole_proprietor` ohne `companyName` → 400
  - `Update` von `sole_proprietor` → `private` ohne `firstname` → 400 (Pflichtfeld-Wechsel)
  - `clearMemberTypeFields` leert Personenfelder bei Typ `sole_proprietor`
- `internal/importing/payload_test.go`
  - `mapBusinessRole(sole_proprietor)` → `EEG_BUSINESS`
  - `mapPersonName(sole_proprietor)` mit `companyName="A"`, `firstname=null` → `("A", "")`
  - **Spezial-Case Q5:** `mapPersonName(sole_proprietor)` mit `companyName="A"`, `firstname="B"` → `("A", "")` (firstname wird ignoriert)
  - Regressionscheck: `mapPersonName(company)` mit Kontaktperson bleibt unverändert
- `internal/excel/generator_test.go`
  - Label-Output enthält `"Kleinunternehmer"` für `sole_proprietor`-Antrag
- `tests/PROJ-7-member-types.spec.ts` (Playwright)
  - Neuer E2E-Smoketest: Public-Registration mit Typ Kleinunternehmer und Firmenname

### Reihenfolge der Implementierung

1. **Backend-Validierung & Models** — Konstante, `oneof`-Tags, `validate-/clearMemberTypeFields`-Cases. Foundation; alles Weitere baut darauf
2. **Import-Mapping** — `isNaturalPerson` + `mapPersonName`-Special-Case in `internal/importing/payload.go`
3. **Frontend-Types & Zod-Schema** — Union erweitern, diskriminierter Zweig
4. **UI** — `MemberTypeSelector` (5. Karte), Conditional-Felder, Admin-Detail, Admin-Edit
5. **Output-Renderer** — E-Mail-Anrede, Excel-Label, PDF-Renderer
6. **Tests** — Unit + E2E
7. **Docs** — `docs/import-mapping.md` §8 (Member type → core role mapping), `docs/api-spec.md`, `docs/domain-model.md`

Backend-Schritte 1–2 und Frontend-Schritte 3–4 sind parallelisierbar, sobald die API-Form (TypeScript-Union) klar ist.

### Risiken und Mitigation

| Risiko | Wahrscheinlichkeit | Mitigation |
|---|---|---|
| Ein Output-Renderer (E-Mail/Excel/PDF) übersieht den neuen Typ und rendert leer/falsch | Mittel | Default-/Fallback-Branch in jedem Renderer (gibt zumindest `company_name` aus, kein leerer String); je Renderer ein Test-Case |
| `mapPersonName`-Special-Case bricht bestehende `company`-Convention | Niedrig | Regressionstest für `company`-Kontaktperson bleibt erhalten; neuer Test für `sole_proprietor` exklusiv |
| Zod-Discriminated-Union vergisst einen Pfad → Frontend akzeptiert Inkonsistenzen | Niedrig | Backend-Validierung fängt es ab; Code-Review prüft alle fünf Zweige |
| Altdaten-`private` wird als „Privatperson" angezeigt, obwohl es ein Kleinunternehmer ist | Niedrig | Spec-resolved (Q2): manuelle Umklassifizierung durch Admin |
| Externe API (PROJ-13) sendet `firstname` für `sole_proprietor` und erwartet, dass es übernommen wird | Niedrig | Q5 explizit dokumentiert: `firstname` wird für `sole_proprietor` ignoriert; OpenAPI-Doc (PROJ-24) entsprechend ergänzen |

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
