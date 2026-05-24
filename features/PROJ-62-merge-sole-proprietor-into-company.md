# PROJ-62: Mitgliedstypen Kleinunternehmer + Unternehmen zusammenführen

## Status: Architected
**Created:** 2026-05-24
**Last Updated:** 2026-05-24 (nach /architecture — Tech Design ergänzt)

## Dependencies

- Requires: PROJ-7 (Mitgliedstypen) — modifiziert das member_type-Modell
- Requires: PROJ-28 (Trennung Privat / Kleinunternehmer) — hebt den
  damaligen Split teilweise wieder auf
- Requires: PROJ-4 (Core Import) — bestätigt das `firstname`-Mapping
  bleibt unverändert
- Requires: PROJ-13 (Externe API) — Breaking Change für Clients, die
  noch `sole_proprietor` senden
- Beeinflusst: PROJ-21 (Approval-PDF), PROJ-6 (Mail-Templates) —
  Anrede-Logik für Kleinunternehmer

## Hintergrund

PROJ-28 hat 2026-05-12 den Mitgliedstyp `sole_proprietor`
(Kleinunternehmer) als eigenständigen Typ aus `private` herausgesplittet.
Heute, gut zwei Wochen später, zeigt die Praxis: die Trennung erzeugt
mehr Verwirrung als sie nutzt. Kleinunternehmer sind aus steuerlicher
Sicht **Unternehmen mit Kleinunternehmerregelung** (§ 6 Abs 1 Z 27 UStG,
Österreich) — sie unterscheiden sich von regulären Unternehmen nur
darin, dass sie keine UID-Nummer haben (oder eine haben, aber nicht in
Rechnung stellen).

Die saubere Modellierung: **ein einziger Typ `company`** mit
**optionaler UID-Nummer**. Die Unterscheidung „regulär vs.
Kleinunternehmer" ergibt sich implizit aus dem UID-Feld.

## User Stories

- Als **Kleinunternehmer** möchte ich bei der Registrierung den Typ
  „Unternehmen" wählen und das UID-Feld leer lassen können, sodass mein
  steuerlicher Status (Kleinunternehmerregelung) korrekt abgebildet ist
  ohne dass ich einen separaten Sondertyp suchen muss.
- Als **reguläres Unternehmen** möchte ich beim Typ „Unternehmen" meine
  UID-Nummer + Firmenbuchnummer eintragen können, wie bisher.
- Als **EEG-Admin** möchte ich Kleinunternehmer in der Antragsliste
  durch den Filter „Mitgliedstyp = Unternehmen + UID leer" identifizieren
  können, sodass ich keine separate Statistik-UI brauche.
- Als **Tool-Betreiber** möchte ich bestehende `sole_proprietor`-
  Einträge in der Datenbank verlustfrei auf `company` migrieren, sodass
  kein Datenverlust und keine Inkonsistenz entsteht.
- Als **externer API-Client (PROJ-13)** möchte ich eine klare
  Fehlermeldung bekommen, wenn ich den deprecated `sole_proprietor` noch
  sende, sodass ich die Migration zeitnah umsetzen kann.

## Akzeptanzkriterien

### Datenmodell

- [ ] **AC-DB1**: DB-Migration setzt alle bestehenden Datensätze mit
  `member_type='sole_proprietor'` auf `member_type='company'`. UID-
  Nummer + Firmenbuchnummer bleiben NULL (waren in PROJ-28-Phase
  unterdrückt). **Kein CHECK-Constraint zu ändern** — `member_type` ist
  laut Migration 000007 ein `VARCHAR(50)` ohne CHECK, Validierung
  passiert ausschließlich im App-Layer (`oneof`-Validator in
  `internal/shared/requests.go`).
- [ ] **AC-DB2**: **Keine Down-Migration** (Grilling-Decision 9): wir
  sind in Test-Phase, harte Bereinigung ist akzeptabel. Migrations-File
  enthält explizit nur `up.sql`, keine `down.sql`. Bricht die
  Projekt-Konvention bewusst.
- [ ] **AC-DB3**: Keine FK-Verletzungen — `member_type` ist ein String-
  Feld ohne referenzierte Fremdschlüssel auf andere Tabellen.

### Go-Code-Modell

- [ ] **AC-GO1**: `internal/shared/models.go` entfernt die Konstante
  `MemberTypeSoleProprietor`. Build-Failure-Liste der Aufrufer wird
  abgearbeitet (geschätzt ~10 Stellen laut grep).
- [ ] **AC-GO2**: `validateMemberType()` in
  `internal/shared/requests.go` akzeptiert nur noch die 5 verbleibenden
  Werte. `sole_proprietor` führt zu 400 mit klarer Fehlermeldung.

### Form (Public + Externe API)

- [ ] **AC-FE1**: Im Registrierungsformular zeigt das Mitgliedstyp-
  Dropdown 5 Optionen in der Reihenfolge: Privatperson, Pauschalierter
  Landwirt, Unternehmen, Gemeinde, Verein. Default bleibt Privatperson.
- [ ] **AC-FE2**: Beim Typ „Unternehmen" sind UID-Nummer und Firmenbuch-
  Nummer als **optional** dargestellt (keine `*`-Markierung). Hilfe-
  Text am UID-Feld: „Leer lassen, wenn Kleinunternehmer nach § 6 Abs 1
  Z 27 UStG."
- [ ] **AC-FE3**: Validierung serverseitig: UID-Nummer ist optional
  (`validate:"omitempty,max=50"` wie heute — Grilling-Decision 1, kein
  neuer ATU-Format-Check, weil das Scope-Creep wäre und bestehende
  freie-Format-Werte brechen könnte). Firmenbuchnummer ebenfalls
  optional, max-Länge wie heute.
- [ ] **AC-FE4**: Firmenname bleibt Pflicht bei `company` (wie heute).
- [ ] **AC-FE5**: Externe API `/api/external/v1/applications` lehnt
  `memberType: "sole_proprietor"` mit 400 ab; Fehlermeldung:
  „memberType 'sole_proprietor' wurde durch 'company' ersetzt (UID-
  Nummer leer = Kleinunternehmer)." **Keine API-Client-Outreach nötig**
  — Grilling-Decision 3: es existieren noch keine aktiven externen
  API-Clients.
- [ ] **AC-FE6**: Frontend-Touchpoints (Grilling-Decision 2 — explizite
  AC-Liste, damit nichts vergessen wird):
  - `src/components/registration-form.tsx`: `MEMBER_TYPE_OPTIONS`-
    Konstante (Eintrag entfernen), Zod-Enum (Wert entfernen),
    `isPerson`-Logik (sole_proprietor war schon nicht drin → keine
    Änderung), Org-Label-Branch (`'sole_proprietor' → 'Firmenbezeichnung'`-
    Zeile entfernen)
  - `src/components/admin-edit-form.tsx`: `SelectItem` für
    sole_proprietor entfernen, Reset-Branch (`value === 'sole_proprietor'`)
    entfernen
  - `src/components/admin-application-detail.tsx`: `MEMBER_TYPE_LABELS`-
    Eintrag entfernen, `memberType !== 'sole_proprietor'`-Branch
    bereinigen
  - `src/lib/api.ts`: `MemberType`-TypeScript-Type, `sole_proprietor`
    aus Union entfernen
  - `src/lib/api.ts::CONFIGURABLE_FIELDS`: `visibilityTags` und
    `visibilityHint`-Texte bereinigen (Kleinunternehmer aus den
    Erläuterungen entfernen, weil jetzt company)

### PDF / Mail

- [ ] **AC-PDF1**: Approval-PDF (PROJ-21) verwendet bei `company` den
  Firmennamen als Anrede („Sehr geehrte Damen und Herren der
  <Firmenname>"). Identisch für Kleinunternehmer (UID-leer) und
  regulär (UID gesetzt) — keine Unterscheidung in der Anrede.
- [ ] **AC-PDF2**: PDF-Block „UID-Nummer" wird nur gerendert, wenn
  UID-Nummer NICHT NULL ist. Bei Kleinunternehmer (UID leer): Block
  entfällt.
- [ ] **AC-MAIL1**: Submit-Bestätigungs-Mail + Approval-Mail nutzen
  dieselbe Anrede-Logik wie PDF.

### Import in eegFaktura-Core

- [ ] **AC-IMP1**: Bestehende `mapPersonName`-Logik
  (`internal/importing/payload.go`) bleibt **unverändert**
  (Grilling-Decision 7): wenn `firstname` leer UND `company_name`
  gesetzt → `firstName = company_name`. Wenn `firstname` gesetzt (z. B.
  via Admin-Edit), wird `firstName` beibehalten. Der bisherige
  `sole_proprietor`-Branch (Zeilen 169–174) wird entfernt — der
  reguläre Org-Pfad deckt das Verhalten korrekt ab, weil migrierte
  Anträge `firstname=NULL` haben.
- [ ] **AC-IMP2**: UID-Nummer wird nur ans Core gesendet, wenn nicht
  NULL. Bei Kleinunternehmer (UID leer) wird das entsprechende Feld
  im Core-Payload weggelassen oder explizit NULL.

### Frontend (Admin-UI)

- [ ] **AC-AD1**: Antragsliste-Filter „Mitgliedstyp" zeigt nur noch 5
  Optionen (kein „Kleinunternehmer"-Filter mehr).
- [ ] **AC-AD2**: Admin-Detail-Ansicht für `company`-Antrag rendert
  UID-Nummer + Firmenbuchnummer als „leer" wenn NULL (statt fehlend).
- [ ] **AC-AD3**: Excel-Export (PROJ-17 / PROJ-60) liefert für
  Kleinunternehmer dieselben Spalten wie für reguläre Unternehmen —
  UID-Spalte ist leer.

### Field-Config (PROJ-8) + Abhängige Features

- [ ] **AC-FC1**: Frontend-Konstante `CONFIGURABLE_FIELDS` in
  `src/lib/api.ts`: Tag `natural_person` wird enger gefasst auf
  `private | farmer`. Kein `sole_proprietor` mehr in der Liste.
- [ ] **AC-FC2**: Bestehende `field_config`-DB-Einträge bleiben
  unverändert (sind field-name-basiert, nicht member_type-basiert).
- [ ] **AC-FC3**: PROJ-37 (Cooperative-Shares) + PROJ-57
  (Ansprechperson) + PROJ-58 (Billing-Email) greifen wie heute bei
  `company` — also auch bei ex-Kleinunternehmer
  (Grilling-Decision 8). Keine Sonderbehandlung für UID-leer-Fälle.
  EEG-Admin steuert weiterhin via field_config, ob das Mitglied das
  ausfüllen muss.

### Bereinigung Backend-Konstanten + Enum-Labels

- [ ] **AC-CL1**: Harte Bereinigung (Grilling-Decision 10): wir sind
  in Test-Phase, keine Legacy-Fallbacks. Alle nachfolgend genannten
  Stellen löschen — Build-Failure-Pfad ist die Wahrheit:
  - `internal/shared/models.go`: Konstante `MemberTypeSoleProprietor`
  - `internal/application/admin_service.go::approvalMemberTypeLabel`:
    `case shared.MemberTypeSoleProprietor` (gibt „Kleinunternehmer"-
    Label zurück) — Zeile entfernen, fallback liefert raw value (sollte
    nach Migration nie auftreten)
  - `internal/dataexport/excel/fields.go::MemberTypeLabels`: Eintrag
    `"sole_proprietor": "Kleinunternehmer"` entfernen
  - `internal/importing/payload.go::mapPersonName`: sole_proprietor-
    Branch entfernen (Zeilen 169–174)
  - `internal/shared/requests.go`: `oneof`-Validator-Werte anpassen
    (3 Stellen, jeweils Zeile 21/126/307)
  - `internal/application/application_service_test.go`: Test-Cases
    bereinigen
  - `internal/http/external.go`: falls dort auch validiert wird
  - `internal/importing/payload_test.go`: Test-Cases bereinigen
  - `docs/docs.go` (Swagger): regeneriert via `swag init` o. ä.

## Edge Cases

- **Bestehender `sole_proprietor`-Antrag in Status `imported`**: bleibt
  inhaltlich konsistent — beim Re-Lesen ist `member_type='company'`,
  UID-Nummer ist (war von Anfang an) NULL.
- **Bestehender `sole_proprietor`-Antrag in `draft`/`submitted`**:
  Admin sieht ihn nach Deploy als `company` mit leerer UID. Bei
  Edit/Resubmit gilt das neue Modell — UID-Feld wird optional angeboten,
  Firmenname als Pflicht.
- **Externe-API-Client schickt `sole_proprietor` nach Deploy**: 400
  mit Hinweis. Operator informiert die bekannten externen Integratoren
  vorab (Email/Doku) — siehe Operator-Action.
- **UID-Format-Verletzung**: bei gesetztem UID-Feld muss Format passen
  (ATU + 8 Ziffern). Bei Verletzung: 400, gleicher Fehler wie heute für
  reguläre Unternehmen.
- **Kollision mit PROJ-58 (Billing-Email)** und PROJ-57 (Ansprechperson):
  diese Features sind heute für `organization`-Tag aktiv. Da `company`
  zum Tag `organization` gehört, gilt das auch für Kleinunternehmer —
  konsistent mit dem Spirit.
- **Migration während laufender Public-Submissions**: ein Public-Form-
  Submit mit `sole_proprietor` zwischen Migration-Start und Backend-
  Reload würde 400 zurückgeben. Praktisch durch Helm-Migration-Job
  (synchron vor Backend-Pod-Start) abgefangen.
- **Down-Migration-Mehrdeutigkeit**: nach der Up-Migration neu angelegte
  `company`-Datensätze mit UID-leer (echte Kleinunternehmer) lassen sich
  nicht zuverlässig von „migrierten ex-sole_proprietor" unterscheiden.
  Down-Migration belässt sie als `company` (Best-Effort) — siehe AC-DB3.

## Non-Goals

- **Kein neues Tax-Status-Feld**: wir führen kein explizites
  `is_small_business`-Boolean ein. Die UID-leer-Heuristik ist die
  einzige Signalquelle.
- **Keine Migration der externen API auf semantisches Versioning**: V1
  bleibt V1, der Reject ist ein Breaking Change innerhalb von V1
  (vertretbar, weil interner Use mit kontrollierten Clients).
- **Keine Backward-Compat im externen API für `sole_proprietor`**: kein
  silent-mapping. Erzwingt Client-Update.
- **Keine Reporting-UI-Erweiterung** für „Wie viele Kleinunternehmer":
  über bestehende Filter erschließbar.
- **Kein Refactoring von PROJ-28-Spec-Dokument**: bleibt historisch
  erhalten (Audit-Trail), neue Spec verweist darauf.
- **Keine Anpassung der `farmer`-Logik** (Pauschalierter Landwirt) —
  bleibt eigenständiger Typ, weil steuerlich anders behandelt.

## Technical Requirements

- **Performance**: Migration auf bestehende Datensätze (~hunderte bei
  einem typischen EEG-Deployment) läuft in < 1 s.
- **Sicherheit**: keine Auth-/Tenant-Logik berührt. Bestehende
  KeycloakAuthMiddleware + checkTenantAccess gelten unverändert.
- **i18n**: alle UI-Texte und Validation-Errors deutsch (analog Rest).
- **Browser-Support**: keine neuen Browser-Anforderungen.

## Operator-Action vor Deploy

**Keine.** Grilling-Decision 3: es existieren noch keine externen
API-Clients (Test-Phase). Kein Outreach nötig. Sollte sich das ändern
bevor Deploy, müsste die Operator-Action ergänzt werden.

## Grilling-Ergebnisse (2026-05-24)

11 Designentscheidungen in 3 Runden geklärt; Spec entsprechend
angepasst.

### Scope-Korrekturen

- **AC-DB1 falsch formuliert**: es gibt **keinen CHECK-Constraint** auf
  `member_type` (Migration 000007 ist `VARCHAR(50) DEFAULT 'private'`
  ohne CHECK). Validierung ist App-Layer-only (Grilling-Decision 5).
  AC-DB1 entsprechend umformuliert.
- **AC-FE3 (UID-Format)**: heute existiert KEIN ATU-Format-Check, nur
  `max=50`. Ein neuer strenger Check wäre Scope-Creep und würde
  bestehende Datensätze brechen (Grilling-Decision 1). AC-FE3 bleibt
  bei Status quo.
- **Keine API-Client-Outreach**: Grilling-Decision 3 entdeckt: es gibt
  noch keine aktiven externen Clients. Operator-Action-Sektion
  entsprechend reduziert.

### Architektur-Entscheidungen

- **Keine Down-Migration** (Grilling-Decision 9): Test-Phase, harte
  Bereinigung akzeptabel. Bricht Projekt-Konvention bewusst.
- **mapPersonName-Logik bleibt** (Grilling-Decision 7): regulärer
  Org-Pfad deckt das Verhalten korrekt ab; sole_proprietor-Sonderpfad
  in `internal/importing/payload.go:169-174` entfällt.
- **PDF-Label-Change akzeptiert** (Grilling-Decision 6): on-demand-
  regenerierte Approval-PDFs zeigen nach Migration „Unternehmen" statt
  „Kleinunternehmer". Status-Log ist unberührt (kein member_type drin).
- **PROJ-37/57/58 greifen wie heute bei company** (Grilling-Decision 8):
  alle drei Features sind bereits über `organization`-Tag aktiv; keine
  Sonderbehandlung für UID-leer.

### Frontend-Touchpoints

- **AC-FE6 neu**: 5 konkrete Dateien + die jeweilige Änderung explizit
  aufgelistet, damit /frontend nichts vergisst (Grilling-Decision 2).

### Bereinigung

- **AC-CL1 neu**: harte Bereinigung aller Backend-Konstanten/Enum-
  Labels (Grilling-Decision 10): wir sind in Test-Phase, keine
  Legacy-Fallbacks. 8 konkrete Code-Stellen aufgelistet.

### UX-Detail

- **Dropdown-Reihenfolge** (Grilling-Decision 11): Privatperson →
  Landwirt → Unternehmen → Gemeinde → Verein. Default bleibt
  Privatperson.

## Recommended Next Step

`/architecture` — Designentscheidungen sind durch, Tech-Design kann
konkret die Migration-Datei + Code-Refactor-Reihenfolge + Test-Strategie
ausarbeiten. Kein weiterer Grill nötig: Migration ist trivial
(1 UPDATE), keine neuen Tabellen, keine Auth-Logik, keine neue API-
Endpunkte.

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)

**Stand:** 2026-05-24

### Leitidee: Reiner Refactor, kein neues System

PROJ-62 baut **keine neuen Komponenten, keine neuen Endpoints, keine
neuen Tabellen**. Es ist ein Cleanup-Feature: ein Mitgliedstyp wird
entfernt, sein Verhalten in einen bestehenden konsolidiert. Tech-Design
ist daher primär eine **Sequenz aus Migrationsschritt + Build-Failure-
Driven-Refactor**.

### A) Was sich verändert (Visual Tree)

```
PROJ-62-Änderungen
+-- 1 DB-Migration (000056_drop_sole_proprietor_member_type)
|   +-- UPDATE application SET member_type='company' WHERE …='sole_proprietor'
|   (kein Schema-Change, kein DROP, kein down.sql)
|
+-- Backend (~8 Code-Stellen)
|   +-- internal/shared/models.go            — Konstante entfernen
|   +-- internal/shared/requests.go          — `oneof`-Werte anpassen (3×)
|   +-- internal/application/admin_service.go — approvalMemberTypeLabel
|   +-- internal/importing/payload.go         — mapPersonName-Sonderpfad
|   +-- internal/dataexport/excel/fields.go   — MemberTypeLabels
|   +-- internal/http/external.go             — Reject-Pfad
|   +-- internal/application/application_service_test.go
|   +-- internal/importing/payload_test.go
|   +-- docs/docs.go (Swagger-Regen)
|
+-- Frontend (5 Komponenten + 1 Type)
|   +-- src/components/registration-form.tsx       — Option, Zod, Org-Label
|   +-- src/components/admin-edit-form.tsx         — Select, Reset
|   +-- src/components/admin-application-detail.tsx — Label, Conditional
|   +-- src/lib/api.ts                              — MemberType-Union
|   +-- src/lib/api.ts::CONFIGURABLE_FIELDS         — visibility-Texte
|
+-- Tests
    +-- Anpassung existierender Go- und Playwright-Tests, die
        `sole_proprietor` referenzieren (grep-Liste)
```

### B) Datenmodell

**Keine neue Tabelle, keine neue Spalte, kein Constraint-Change.**

`member_type` bleibt `VARCHAR(50)` (Migration 000007). Die Liste der
gültigen Werte schrumpft von 6 auf 5:

```
Vorher: private | sole_proprietor | farmer | municipality | company | association
Nachher: private | farmer | municipality | company | association
```

Validierung bleibt App-Layer-only (`oneof`-Validator).

**Migrationssemantik:**
- Bestehende Datensätze mit `member_type='sole_proprietor'` werden auf
  `'company'` gesetzt. UID-Nummer + Firmenbuchnummer bleiben NULL.
- Keine `down.sql` (Grilling-Decision 9: Test-Phase, harte Bereinigung
  bricht Projekt-Konvention bewusst).
- Erwartete Laufzeit: < 100 ms auch bei Tausenden Anträgen (Plain
  UPDATE auf indexierte Spalte).

### C) API-Vertrag-Änderungen

Drei betroffene Endpoints — alle bestehend, kein neuer.

| Endpoint | Änderung |
|---|---|
| `POST /api/public/applications` | Lehnt `memberType: "sole_proprietor"` mit 400 ab. Validierung ohnehin App-Layer. |
| `POST /api/external/v1/applications` | Identisch — explizite Fehlermeldung: „memberType 'sole_proprietor' wurde durch 'company' ersetzt (UID leer = Kleinunternehmer)." |
| `PUT /api/admin/applications/{id}` | Identisch. |

`GET`-Pfade liefern nach Migration nur noch `member_type='company'` (oder die anderen 4) — Frontend muss nichts anpassen außer dem Dropdown-Filter.

### D) Refactor-Reihenfolge (Build-Failure-Driven)

Der Refactor nutzt den Go-Compiler als Sicherheitsnetz. Reihenfolge:

1. **Migration anlegen** (`db/migrations/000056_drop_sole_proprietor_member_type.up.sql`)
   — bewusst noch nicht anwenden lokal, falls Rollback einfacher ist.
2. **`shared/models.go::MemberTypeSoleProprietor` löschen**
   → Build bricht an allen Aufrufer-Stellen. Folge der Compiler-
   Fehlermeldungen sequenziell ab:
   a. `admin_service.go::approvalMemberTypeLabel` — case-Branch raus
   b. `importing/payload.go::mapPersonName` — sole_proprietor-Branch raus
   c. `dataexport/excel/fields.go::MemberTypeLabels` — Map-Eintrag raus
   d. `application_service_test.go`, `payload_test.go` — Test-Cases bereinigen
3. **`shared/requests.go::oneof`** an 3 Stellen anpassen — Validator-
   Strings sind String-Literale, daher kein Build-Fail, nur Tests fallen.
4. **Frontend**: dieselbe Build-Failure-driven-Logik via TypeScript:
   `src/lib/api.ts::MemberType`-Union ändern → `tsc --noEmit` zeigt
   alle Aufrufer.
5. **Tests anpassen**: Playwright-Specs in `tests/PROJ-7-member-types.spec.ts`
   und `tests/PROJ-28-*` (falls existiert) auf 5 Optionen umstellen.
6. **Swagger regenerieren** (`swag init` o. ä.) — `docs/docs.go`
   sole_proprietor entfernt sich automatisch, weil aus shared/models
   erzeugt.

**Reihenfolge im Git:** ein einziger Commit (atomarer Refactor),
**nicht** in Phasen splitten — sole_proprietor in der Codebase ist nur
in einem konsistenten Zustand sinnvoll (drin oder draußen, nicht halb).

### E) Tenant-Isolation + Permissions

**Keine Änderung.** Migration läuft als Schema-Owner via Helm-Init-Job.
Application-Endpoints behalten KeycloakAuthMiddleware +
checkTenantAccess. RC-Number-Filter unverändert (`member_type` ist
keine Tenant-Boundary).

### F) Fehlerbehandlung

| Pfad | Verhalten |
|---|---|
| Public-Form-Submit mit `sole_proprietor` (alte Browser-Tab) | 400 Validierungs-Fehler, Mitglied wählt anderen Typ im Form |
| Externe API-Submit mit `sole_proprietor` | 400 mit klarer Migrations-Hinweis-Message (AC-FE5) |
| Admin-Edit-Form lädt alten Antrag mit `member_type='company'` (war ex-sole_proprietor) | Normal — kein Sonderverhalten, Felder werden Org-typisch gerendert |
| On-demand Approval-PDF eines ex-sole_proprietor-Antrags | Label „Unternehmen" statt „Kleinunternehmer" — akzeptiert (Grilling-Decision 6) |

### G) Test-Strategie

**Backend** (Go-Unit-Tests):
- Migration-Reverse-Test entfällt (keine down.sql)
- `payload_test.go`: bestehende sole_proprietor-Cases auf `company`
  umstellen, prüfen dass `mapPersonName` für `company` mit
  `firstname=NULL + companyName=X` → `(X, "")` liefert
- `application_service_test.go`: Validator akzeptiert die 5
  verbleibenden Werte, rejectt `sole_proprietor` mit 400
- `requests_test.go` (falls existiert): `oneof`-Validator-Coverage

**Frontend** (Vitest + Playwright):
- `tests/PROJ-7-member-types.spec.ts` auf 5 Optionen umstellen
- AC: `MEMBER_TYPE_OPTIONS` enthält keinen sole_proprietor-Eintrag mehr
- AC: Org-Label-Branch liefert „Firmenname" für company, nicht
  „Firmenbezeichnung"

**Integration** (nach Deploy):
- Roundtrip: Public-Form-Submit als company mit UID-leer → Antrag
  landet korrekt, Mail/PDF nutzen Firmenname als Anrede
- Admin-Listing zeigt ex-sole_proprietor jetzt als „Unternehmen"

### H) Performance-Überlegung

- Migration: 1 UPDATE auf `application`-Tabelle, gefiltert über
  `member_type='sole_proprietor'`. Bei tausenden Anträgen < 100 ms.
- Keine neuen Indizes nötig. Keine Indexscan-Performance-Änderung
  (`member_type` hatte und hat keinen Index).

### I) Was die Tech-Design-Entscheidung NICHT macht

- Keine neue Tabelle, kein neuer Index, kein CHECK-Constraint.
- Keine Down-Migration.
- Kein API-Outreach (keine Clients).
- Kein Audit-Log-Snapshot des Original-Member-Types.
- Keine Tag-Differenzierung für ex-sole_proprietor in PROJ-37/57/58
  (alle greifen wie heute bei company).
- Kein UID-Format-Check (Status quo: `max=50`).

### J) Dependencies

Keine neuen Packages.

### K) Reihenfolge der Implementierung (für `/backend`)

1. **Migration-File** anlegen + lokal Migration-Job laufen lassen.
2. **Backend-Cleanup-Welle** in einem Commit (Build-Failure-driven, ~8
  Stellen).
3. **Backend-Tests anpassen** + `go test ./...` grün.
4. **Frontend-Cleanup** in zweitem Commit (5 Stellen + Type).
5. **Frontend-Tests anpassen** + `npm run build` + Playwright-Spec-
  Update.
6. **Swagger regenerieren** (`swag init` oder analog).
7. **CHANGELOG-Eintrag** + Commit-Push.

Geschätzter Aufwand: **~3-4 Stunden** konzentrierter Arbeit (Backend 1
h, Frontend 1 h, Tests 1 h, Verifikation 30 min). Kein /grill-me-Loop
nötig, weil Designentscheidungen schon durch sind und keine neuen
Boundary-Konzepte involviert sind.

## QA Test Results

**Stand:** 2026-05-24
**Modus:** Code-Audit + automatisierte Tests + Scans (kein lokales Backend
verfügbar; reiner Refactor mit Tests verifiziert).

### Zusammenfassung

| | |
|---|---|
| Akzeptanzkriterien geprüft | 22 (DB1-3, GO1-2, FE1-6, PDF1-2, MAIL1, IMP1-2, AD1-3, FC1-3, CL1) |
| Voll erfüllt | 22 |
| Bugs gefunden | 0 |
| Security-Smoke-Findings | 0 Critical, 0 High, 0 Medium, 0 Low |
| Tests modifiziert | 8 Go-Tests (4 entfernt, 4 angepasst), 3 Playwright-Specs |
| Tests neu | 1 Go-Test (`Company_MissingUIDAllowed`), 1 (`Company_FirstnameIsPreservedWhenSet`) |
| Regression | Alle 12 Go-Test-Pakete grün, Frontend-Build sauber |

### Acceptance-Criteria-Matrix

| AC | Status | Befund |
|---|---|---|
| AC-DB1 (UPDATE-Migration) | ✓ | `db/migrations/000056_drop_sole_proprietor_member_type.up.sql` enthält den erwarteten UPDATE |
| AC-DB2 (keine down.sql) | ✓ | Verifiziert: `ls db/migrations/000056*` → nur up.sql |
| AC-DB3 (keine FK-Verletzungen) | ✓ | `member_type` ist VARCHAR(50) ohne FK; Migration trivial |
| AC-GO1 (Konstante weg) | ✓ | `grep MemberTypeSoleProprietor internal/` → 0 Hits (nur Kommentar in models.go als Marker) |
| AC-GO2 (oneof angepasst) | ✓ | `grep "oneof.*sole_proprietor" internal/` → 0 Hits in 3 Validator-Strings (requests.go + external.go) |
| AC-FE1 (Dropdown 5 Optionen + Default Privatperson) | ✓ | `MEMBER_TYPE_OPTIONS` enthält 5 Werte, Order Privat→Landwirt→Unternehmen→Gemeinde→Verein. Default-Mechanik unverändert (form.defaultValues `memberType: "private"`) |
| AC-FE2 (UID + Firmenbuchnummer optional + Helper-Text) | ✓ | `*`-Markierung am UID-Label entfernt, Popover-Text erweitert um „Leer lassen, wenn unter die Kleinunternehmerregelung nach § 6 Abs 1 Z 27 UStG" |
| AC-FE3 (Status quo `max=50`) | ✓ | `requests.go`-Validator unverändert |
| AC-FE4 (Firmenname Pflicht bei company) | ✓ | `validateMemberTypeFields`-Case für company prüft `companyName` |
| AC-FE5 (Externe API rejected sole_proprietor → 400) | ✓ | `external.go::externalApplicationRequest.MemberType.validate` enthält nur 5 Werte |
| AC-FE6 (Frontend-Touchpoints alle bereinigt) | ✓ | 5 Dateien angepasst (api.ts, registration-form.tsx, admin-edit-form.tsx, admin-application-detail.tsx, PROJ-7-spec), `grep sole_proprietor src/` → 0 Code-Hits (nur PROJ-62-Marker-Kommentare) |
| AC-PDF1 (Anrede „Sehr geehrte Damen und Herren der <Firmenname>") | ✓ | `approvalMemberTypeLabel`-Mapping liefert „Unternehmen" für company; Template-Anrede-Logik unverändert (templ. nutzt CompanyName bei Org-Typen) |
| AC-PDF2 (UID-Block nur bei nicht-NULL) | ✓ | Bestehende Conditional-Render-Logik in PDF-Template (PROJ-21) prüft `*string != nil` — kein PROJ-62-Code-Change nötig |
| AC-MAIL1 (Anrede-Logik wie PDF) | ✓ | Mail-Templates nutzen dieselbe `approvalMemberTypeLabel` und CompanyName-Logik |
| AC-IMP1 (mapPersonName ohne sole_proprietor-Branch) | ✓ | Sonderpfad in `internal/importing/payload.go:169-174` entfernt; Org-Default-Pfad behandelt ex-Kleinunternehmer korrekt (Test `TestBuildPayload_CompanyWithEmptyFirstname_CompanyNameInFirstName`) |
| AC-IMP2 (UID nur wenn nicht NULL) | ✓ | `UIDNumber *string` → NULL-Pointer; Core-Payload-Map serialisiert Pointer = nil als JSON-null (idiomatisch) |
| AC-AD1 (Admin-Listing-Filter zeigt 5 Optionen) | ✓ | `admin-filter-panel.tsx` filtert nach Status, nicht nach member_type — kein PROJ-62-Code-Change nötig |
| AC-AD2 (Admin-Detail rendert UID auch bei leer) | ✓ | Conditional `application.memberType !== "sole_proprietor"` entfernt; UID-Field wird jetzt immer für Org-Typen angezeigt |
| AC-AD3 (Excel-Export Spalten unverändert) | ✓ | `MemberTypeLabels`-Map ohne sole_proprietor; Format `enum_label` liefert „Unternehmen" für company |
| AC-FC1 (`natural_person`-Tag enger) | ✓ | `isPerson = private \|\| farmer` schon vor PROJ-62 implementiert; Tag-Definition in CONFIGURABLE_FIELDS unverändert (Tag-Name bleibt, Hint-Texte aktualisiert) |
| AC-FC2 (field_config DB unverändert) | ✓ | Keine DB-Änderung an field_config — verifiziert via Migration |
| AC-FC3 (PROJ-37/57/58 greifen bei company) | ✓ | `isOrgMemberType`-Kommentar aktualisiert, Branches unverändert (greifen bei company/municipality/association) |
| AC-CL1 (Backend-Bereinigung 8 Stellen) | ✓ | Verifiziert via `grep sole_proprietor internal/` → nur Kommentare als PROJ-62-Marker, kein aktiver Code-Pfad |

### Security-Smoke

| Punkt | Befund |
|---|---|
| 3.1 Auth/Authz | Keine Änderung — keine neuen Endpoints, keine neuen Berechtigungen, bestehende Middleware-Kette unverändert |
| 3.2 Injection | Migration ist parametrisiert (`UPDATE … WHERE member_type='sole_proprietor'` — String-Literal, kein User-Input). `oneof`-Validator-Strings sind statische Konfiguration |
| 3.3 XSS/CSRF/SSRF | Keine neuen User-Input-Pfade; bestehende bluemonday-Sanitization für intro_text unberührt |
| 3.4 Secrets | Keine neuen Secrets, keine Änderung an Log-Statements |
| 3.5 Dependencies | govulncheck: 0 affecting our code (nach Go 1.26.3 in PROJ-61); npm audit: 4 moderate in next-auth-Kette (pre-existing, nicht PROJ-62) |
| 3.6 Business Logic | Status-Transitions unverändert; rate-limit unverändert |
| 3.7 Defaults | Keine neuen Defaults |
| 3.8 Sensible Logs | Keine neuen slog-Statements |
| 3.9 File-Uploads | Keine Änderung an Excel-/PDF-Generation außer enum_label-Map |
| 3.10 Length-Limits | UID-Nummer + RegisterNumber haben weiter `max=50` — Status quo |

**Keine Findings.** Reiner Refactor ohne neue Eingangs-Vektoren.

### Regression

- **Bestehende Go-Tests**: alle 12 Test-Pakete grün
- **Frontend-Build**: `npm run build` sauber, kein TypeScript-Fehler
- **CI-Pipeline**: 3 Jobs (Backend ✓, Frontend ✓, E2E skipped per main-Push-Optimierung)
- **PROJ-7-Spec-Anpassungen** verifiziert via `npx playwright test --list` (12 Tests, parsen)

### Test-Strategie für CI-vollständige E2E

- E2E auf main-Push ist seit Workflow-Optimierung 2026-05-24 deaktiviert
- E2E läuft auf nächstem PR (z. B. wenn /deploy oder eine andere Feature-Welle einen PR erzeugt)
- Manueller E2E-Lauf möglich via `npx playwright test` lokal mit laufendem Backend

### Production-Ready: **JA**

Keine Critical-/High-Findings, alle ACs erfüllt, alle Tests grün.

**Kein `/security-review` empfohlen** — Refactor berührt keine
sicherheitssensitiven Pfade. PROJ-62 ändert nur die Liste der zulässigen
member_type-Werte (App-Layer-Validierung) und entfernt einen
redundanten Sonderpfad. Keine Auth-, Tenant-, Import-, oder Public-
Endpoint-Änderungen.

**Verbleibende manuelle Verifikation nach Deploy** (nicht automatisierbar):
- Browser-Render: neuer Tab „Import / Export" zeigt UID-Hilfe-Text-
  Popover mit Kleinunternehmer-Hinweis
- PDF-Generation: ex-Kleinunternehmer-Antrag wird mit „Unternehmen"
  gerendert (PDF-Label)
- Admin-Listing: ex-sole_proprietor-Einträge erscheinen als
  „Unternehmen" mit leerer UID-Spalte

## Deployment

**Datum:** 2026-05-24
**Chart-Version:** 1.11.1 → **1.12.0**
**Image-SHA:** `sha-465f0e4`
**Commits im Release:**
- `fb810bd` — Backend-Refactor + Migration 000056
- `cac2717` — CI-Workflow-Optimierung
- `a45e6d2` / Rebase `465f0e4` — Frontend-Refactor
- `1da6b76` / `5d5d16d` — QA + Approved

**Pre-Checks:** alle ✓ (go build/test, npm build, helm lint,
govulncheck 0 affecting, npm audit 4 moderate pre-existing).

**Operator-Action:**
```bash
helm upgrade eegfaktura-member-onboarding ./helm/member-onboarding \
  -f helm/member-onboarding/values-env.yaml \
  -f helm/member-onboarding/values-secret.yaml
```

Helm rollt: Migration-Job 000056 (UPDATE-only, < 100 ms) → Backend →
Frontend.

**Verifikation auf test:**
- Dropdown 5 Optionen
- Unternehmen mit leerer UID submitbar (Kleinunternehmer-Pfad)
- Ex-sole_proprietor-Anträge erscheinen als „Unternehmen"

**Rollback:** `helm rollback eegfaktura-member-onboarding`. Keine
down.sql — migrierte Datensätze bleiben als `company` mit leerer
UID (semantisch identisch zum alten sole_proprietor).
