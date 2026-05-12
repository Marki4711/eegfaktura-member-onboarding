# PROJ-28: Trennung von „Privat" und „Kleinunternehmer"

## Status: Approved
**Created:** 2026-05-12
**Last Updated:** 2026-05-12 (Implementation + QA complete, BUG-1 fixed)

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

**QA Date:** 2026-05-12
**Tester:** Claude QA

### Automated Tests

| Suite | Result |
|---|---|
| `go test ./...` | ✅ alle Pakete grün |
| `go build ./...` | ✅ |
| `npx tsc --noEmit` | ✅ |
| `application_service_test.go` (4 neue sole_proprietor-Cases) | ✅ |
| `payload_test.go` (3 neue sole_proprietor-Cases inkl. Q5-Regression) | ✅ |
| GitHub Actions CI (Backend + Frontend) auf `b1da1fc` | ✅ success |

### Acceptance Criteria

#### Neuer Mitgliedstyp `sole_proprietor`
| # | Criterion | Result |
|---|---|---|
| AC-1 | Konstante `MemberTypeSoleProprietor` mit Wert `"sole_proprietor"` | ✅ |
| AC-2 | `oneof`-Validator akzeptiert neuen Wert (4 Stellen) | ✅ |
| AC-3 | `MemberTypePrivate` bedeutet nur noch Privatperson | ✅ |

#### Formular: Typ-Auswahl
| # | Criterion | Result |
|---|---|---|
| AC-4 | Public-Form zeigt neue Option „Kleinunternehmer" | ✅ |
| AC-5 | Altes Label „Privat / Kleinunternehmer" → „Privatperson" | ✅ |
| AC-6 | Default „Privatperson" beibehalten | ✅ |

#### Formular: Felder je Typ
| # | Criterion | Result |
|---|---|---|
| AC-7 | `sole_proprietor` zeigt nur Firmenname (kein Vorname/Nachname/Geburtsdatum/UID/Reg.Nr.) | ✅ |
| AC-8 | Typ-Wechsel löscht typspezifische Felder (`onMemberTypeChange`) | ✅ |
| AC-9 | Übrige Typen unverändert | ✅ |

#### Backend: Validierung & Bereinigung
| # | Criterion | Result |
|---|---|---|
| AC-10 | `validateMemberTypeFields(sole_proprietor)` verlangt `company_name` | ✅ |
| AC-11 | `clearMemberTypeFields(sole_proprietor)` leert Personenfelder + UID + Reg.Nr., behält CompanyName | ✅ |
| AC-12 | Validierung in Create, Update, Submit und AdminUpdate aktiv | ✅ (bestehende Aufrufstellen) |

#### Backend: Import-Mapping
| # | Criterion | Result |
|---|---|---|
| AC-13 | `mapBusinessRole(sole_proprietor) == "EEG_BUSINESS"` | ✅ |
| AC-14 | `mapPersonName(sole_proprietor)` setzt CompanyName in firstName, lastName leer | ✅ |
| AC-15 | Q5: incoming `firstname` für sole_proprietor wird ignoriert (nicht überschrieben) | ✅ (Test `TestBuildPayload_SoleProprietor_IncomingFirstnameIsIgnored`) |
| AC-16 | Regression: `company` mit Kontaktperson behält weiterhin firstname | ✅ (Test `TestBuildPayload_NonPrivateWithContactPerson`) |

#### Admin-UI
| # | Criterion | Result |
|---|---|---|
| AC-17 | `MEMBER_TYPE_LABELS["sole_proprietor"] == "Kleinunternehmer"` | ✅ |
| AC-18 | Admin-Detail zeigt für sole_proprietor nur Firmenname, kein UID/Reg.Nr.-Block | ✅ |
| AC-19 | Admin-Edit-Form spiegelt Public-Form-Felder (UID nur für `company`/`municipality`/`association`) | ✅ |
| AC-20 | Antragsliste: sole_proprietor erscheint mit Firmenname in der Namensspalte | ✅ (existierender Fallback-Branch) |

#### Output-Renderer
| # | Criterion | Result |
|---|---|---|
| AC-21 | PDF-Renderer (`approval_pdf.go`) zeigt Mitgliedstyp „Kleinunternehmer", Firmenname-Zeile, kein Name-Block | ✅ (Template-Conditionals greifen) |
| AC-22 | Excel-Export: `mapBusinessRole(sole_proprietor) == "business"` (Spalte X) | ✅ (Default-Branch) |
| AC-23 | E-Mail-Anrede neutral für sole_proprietor (kein „Sehr geehrte/r ,") | ❌ → **BUG-1** (gefixt während QA) |

#### Migration & Rückwärtskompatibilität
| # | Criterion | Result |
|---|---|---|
| AC-24 | Keine DB-Migration, nur Anwendungs-Level | ✅ |
| AC-25 | Bestehende `private`-Anträge bleiben funktional | ✅ |
| AC-26 | Excel-/PDF-/Mail-Renderer für Bestandsanträge unverändert | ✅ |

### Bugs Found

#### BUG-1 — Medium: Mail-Anrede „Sehr geehrte/r ," für alle nicht-natürlichen Personen

**Severity:** Medium (UX, kein Datenverlust)
**Component:** `internal/mail/templates/application_submitted_member.html`
**Description:** Das Member-Submission-Template rendert hart `Sehr geehrte/r {{.Firstname}} {{.Lastname}},`. Für Mitgliedstypen ohne Personennamen (sole_proprietor — neu durch PROJ-28; auch bestehend für company/association/municipality) sind diese Felder leer, sodass die Anrede zu „Sehr geehrte/r ," wird.
**Steps to reproduce:**
1. Antrag mit `memberType = sole_proprietor` und gefülltem `companyName` einreichen
2. Eingehende Member-Mail prüfen → Anrede ist beschädigt
**Fix:** Template-Conditional eingeführt:
```html
{{if .Firstname}}
  <p>Sehr geehrte/r {{.Firstname}} {{.Lastname}},</p>
{{else}}
  <p>Sehr geehrte Damen und Herren,</p>
{{end}}
```
Dadurch fällt sole_proprietor (und implizit alle Org-Typen) auf die neutrale Anrede zurück — entspricht Q4 der Spec.
**Status:** ✅ Behoben.

### Security Smoke

| Bereich | Risiko | Bewertung |
|---|---|---|
| Neue Auth-Pfade | Keine | ✓ |
| Status-Transitions | Unverändert | ✓ |
| Tenant-Isolation | Nicht berührt | ✓ |
| Input-Validierung | `oneof` strikt; CompanyName über bestehende `min/max`-Tags begrenzt | ✓ |
| SQL-Injection | Keine neuen Queries | ✓ |
| PII-Logging | Keine neuen Logs | ✓ |
| Mass Assignment | `clearMemberTypeFields` verhindert leftover-Daten bei Typ-Wechsel | ✓ |

→ Kein `/security-review` erforderlich (entspricht Spec-Notiz).

### Regression

- Bestehende 5 Mitgliedstypen: keine Verhaltensänderung in Validatoren oder Mapper.
- `company` mit Kontaktperson: explizit per Test abgesichert (Regression-Test ist Bestandsschutz).
- Excel-/PDF-Renderer: Default-Branches greifen unverändert für Bestandstypen.
- Mail-Template-Fix verbessert auch UX für `company`/`association`/`municipality` (vorher latentes Anrede-Problem).

### Production-Ready Decision

**READY** — BUG-1 behoben, alle ACs erfüllt, CI auf `b1da1fc` grün. Status kann nach Deploy auf `Approved` bzw. `Deployed` wechseln.

## Deployment

**Deployed:** _pending CI rollout_
**Chart version:** 1.4.0 / appVersion 1.4.0
**Migration:** none — additive feature, no DB schema change
**Rollback:** `helm rollback` to chart `1.3.0`; no migration needs to be reverted

### Deployment checklist
- [x] `go build ./...` clean
- [x] `go test ./...` clean
- [x] `npx tsc --noEmit` clean
- [x] CI Build & Test green on `b1da1fc` + `25a1c87`
- [x] QA approved, BUG-1 fixed
- [x] No new environment variables required
- [x] No new Kubernetes Secrets required
- [x] Helm chart `appVersion` bumped to `1.4.0`
- [x] Image tag auto-bumps via existing Helm-tag CI step (no manual action)
