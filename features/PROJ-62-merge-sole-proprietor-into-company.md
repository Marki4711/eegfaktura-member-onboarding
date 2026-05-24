# PROJ-62: Mitgliedstypen Kleinunternehmer + Unternehmen zusammenführen

## Status: Planned
**Created:** 2026-05-24
**Last Updated:** 2026-05-24

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

- [ ] **AC-DB1**: DB-Migration entfernt `sole_proprietor` aus dem
  `member_type`-CHECK-Constraint und behält nur:
  `private | farmer | municipality | company | association` (5 statt 6).
- [ ] **AC-DB2**: DB-Migration setzt alle bestehenden Datensätze mit
  `member_type='sole_proprietor'` auf `member_type='company'`. UID-
  Nummer + Firmenbuchnummer bleiben NULL (waren in PROJ-28-Phase
  unterdrückt).
- [ ] **AC-DB3**: Down-Migration ist umsetzbar (für rollback): fügt
  `sole_proprietor` wieder zum CHECK-Constraint dazu und setzt die in
  Schritt AC-DB2 migrierten Datensätze zurück. **Caveat dokumentieren**:
  Datensätze, die nach AC-DB2 als `company` mit UID-leer angelegt wurden,
  würden in der Down-Migration als `company` bleiben (kein zuverlässiger
  Reverse-Mapping möglich) — Down ist daher Best-Effort.
- [ ] **AC-DB4**: Keine FK-Verletzungen — `member_type` ist ein String-
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
  Dropdown 5 Optionen: Privatperson, Pauschalierter Landwirt, Gemeinde,
  Unternehmen, Verein. Default bleibt Privatperson.
- [ ] **AC-FE2**: Beim Typ „Unternehmen" sind UID-Nummer und Firmenbuch-
  Nummer als **optional** dargestellt (keine `*`-Markierung). Hilfe-
  Text am UID-Feld: „Leer lassen, wenn Kleinunternehmer nach § 6 Abs 1
  Z 27 UStG."
- [ ] **AC-FE3**: Validierung serverseitig: bei `member_type=company`
  ist UID-Nummer optional; falls gesetzt, muss sie das österreichische
  Format (`ATU` + 8 Ziffern) erfüllen. Firmenbuchnummer ebenfalls
  optional; Format wie heute (max-Länge).
- [ ] **AC-FE4**: Firmenname bleibt Pflicht bei `company` (wie heute).
- [ ] **AC-FE5**: Externe API `/api/external/v1/applications` lehnt
  `memberType: "sole_proprietor"` mit 400 ab; Fehlermeldung:
  „memberType 'sole_proprietor' wurde durch 'company' ersetzt (UID-
  Nummer leer = Kleinunternehmer)."

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

- [ ] **AC-IMP1**: Für `company` (egal ob UID gesetzt oder leer) wird
  beim Import via `internal/importing/payload.go` weiterhin
  `firstname = company_name` gemappt. Konsistent zu PROJ-28-Verhalten.
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

### Field-Config (PROJ-8)

- [ ] **AC-FC1**: Frontend-Konstante `CONFIGURABLE_FIELDS` in
  `src/lib/api.ts`: Tag `natural_person` wird enger gefasst auf
  `private | farmer`. Kein `sole_proprietor` mehr in der Liste.
- [ ] **AC-FC2**: Bestehende `field_config`-DB-Einträge bleiben
  unverändert (sind field-name-basiert, nicht member_type-basiert).

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

- **Bestehende externe API-Clients informieren**: Mail/Doku-Update an
  alle bekannten Integratoren, die `/api/external/v1/applications`
  nutzen. Hinweis: ab Release-Datum X führt `memberType: "sole_proprietor"`
  zu 400. Clients müssen vor diesem Datum auf `"company"` umstellen.
  Bekannte Integratoren laut Memory: prüfen via `external_api_key`-
  Tabelle, welche EEGs einen aktiven API-Key haben.

## Recommended Next Step

`/grill-me` — die Spec berührt ein Domain-Konzept (Mitgliedstypen),
eine DB-Migration mit Daten-Mutation, Backward-Compat einer externen API
und mehrere abhängige Features (PDF, Mail, Import, Reporting, Field-
Config). Klassischer Grill-Kandidat: Annahmen über UID-Validierung,
Field-Config-Tag-Implikationen, Down-Migration-Strategie sollten
stressgetestet werden, bevor `/architecture` startet.

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
