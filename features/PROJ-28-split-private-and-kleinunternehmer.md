# PROJ-28: Trennung von „Privat" und „Kleinunternehmer"

## Status: Planned
**Created:** 2026-05-12
**Last Updated:** 2026-05-12

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

### Neuer Mitgliedstyp `kleinunternehmer`
- [ ] In `shared/models.go` existiert die Konstante `MemberTypeKleinunternehmer` mit dem String-Wert `"kleinunternehmer"`
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
- [ ] Typ `kleinunternehmer`: **nur** das Feld Firmenname ist sichtbar und Pflicht. Vorname, Nachname, Geburtsdatum, UID und Firmenbuchnummer werden **nicht** angezeigt
- [ ] Wechsel zwischen `private` und `kleinunternehmer` setzt typspezifische Felder zurück (kein Daten-Carry-over)
- [ ] Die übrigen Typen (`farmer`, `municipality`, `company`) bleiben unverändert

### Backend: Validierung
- [ ] Für `kleinunternehmer`: `company_name` ist Pflicht; `firstname`, `lastname`, `birth_date`, `uid_number`, `register_number` werden ignoriert (falls übergeben)
- [ ] Für `private`: unverändert — `firstname`, `lastname`, `birth_date` Pflicht
- [ ] `validateMemberTypeFields` und `clearMemberTypeFields` werden um den neuen Zweig erweitert
- [ ] Der `kleinunternehmer`-Zweig wird in Create, Update **und** Submit konsistent geprüft (siehe PROJ-7 BUG-1/BUG-2)

### Backend: Import-Mapping in eegFaktura
- [ ] `mapPersonName` in `internal/importing/payload.go` behandelt `kleinunternehmer` analog zu `company`/`association`/`municipality`: `company_name` landet in `firstname`, `lastname` bleibt leer
- [ ] `isNaturalPerson(kleinunternehmer)` liefert `false` (damit das Name-Mapping greift)
- [ ] `mapBusinessRole(kleinunternehmer)` liefert **`EEG_BUSINESS`** (Anzeige unter Firma-Tab in eegFaktura — siehe Open Question Q1)

### Admin-UI: Detailansicht und Edit-Form
- [ ] `admin-application-detail.tsx` zeigt für `kleinunternehmer`:
  - Typ-Label: „Kleinunternehmer"
  - Datenblock: Firmenname (kein Vorname/Nachname/Geburtsdatum, keine UID, keine Reg.Nr.)
- [ ] `admin-edit-form.tsx` zeigt im Bearbeitungs-Modus dieselben Felder wie das öffentliche Formular für `kleinunternehmer`
- [ ] Filter/Sortierung in der Antragsliste unterstützt `kleinunternehmer` als eigenen Filterwert

### Excel-Export (PROJ-17) & PDF (PROJ-21)
- [ ] Excel-Export gibt `kleinunternehmer` mit lesbarem Label aus (analog zu `company` etc.)
- [ ] Approval-PDF zeigt für `kleinunternehmer` den Firmennamen analog zu Unternehmen
- [ ] E-Mail-Anrede bei `kleinunternehmer`: neutral mit Firmennamen, nicht „Sehr geehrte/r Vor- Nachname" (siehe Open Question Q4)

### Migration & Rückwärtskompatibilität
- [ ] Bestehende Anträge mit `member_type = private` bleiben unverändert auf `private` (sie sind Privatpersonen — der Kleinunternehmer-Anteil unter den Altdaten ist im Onboarding-System nicht unterscheidbar; Admin müsste manuell umklassifizieren bei Bedarf)
- [ ] Keine automatische Daten-Migration alter Anträge — siehe Open Question Q2
- [ ] Schema-Migration nur additiv: ein zusätzlich erlaubter Wert im `member_type`-CHECK/`oneof`, keine neuen Spalten

## Edge Cases

- Was passiert, wenn ein bestehender `private`-Antrag im Admin-Edit von `private` auf `kleinunternehmer` umgestellt wird? → Vorname/Nachname/Geburtsdatum werden serverseitig durch `clearMemberTypeFields` geleert; `company_name` muss gefüllt sein, sonst 400. Bestätigung im UI: Hinweis „Personendaten werden entfernt"
- Was passiert, wenn ein Kleinunternehmer-Antrag importiert wird, bevor der Admin einen Tarif (PROJ-27) zugewiesen hat? → wie bei `company` heute: Import läuft, Tarif bleibt leer und wird in eegFaktura nachgepflegt — keine Sonderbehandlung
- Was passiert, wenn ein Kleinunternehmer auch eine UID hat (z.B. weil er später umsatzsteuerpflichtig wird)? → er gehört dann zum Typ `company`, nicht `kleinunternehmer` — Admin klassifiziert um
- Was passiert bei der externen Registrierungs-API (PROJ-13)? → akzeptiert `member_type = kleinunternehmer` mit denselben Pflichtfeldern wie das öffentliche Formular

## Technical Requirements

- **Konsistenz:** Mapping-Logik zwischen Onboarding-`member_type` und eegFaktura-`businessRole` ist an einer Stelle (`internal/importing/payload.go`) — keine Duplikate
- **Tests:** `payload_test.go` bekommt einen Testfall für `kleinunternehmer` (BusinessRole + Name-Mapping). `application_service_test.go` bekommt Validierungs-Cases für create/update/submit
- **Rückwärtskompatibilität:** bestehende `private`-Anträge funktionieren ohne Datenverlust weiter; Excel/PDF/E-Mail rendern Altdaten unverändert
- **Beobachtbarkeit:** keine neuen Log-Felder erforderlich; das `member_type`-Feld ist bereits Bestandteil bestehender Logs

## Open Questions

### Q1: `businessRole` für Kleinunternehmer — `EEG_PRIVATE` oder `EEG_BUSINESS`?

Der Kleinunternehmer ist steuerlich ein Einzelunternehmer mit 0% USt (Kleinunternehmerregelung gem. § 6 Abs. 1 Z 27 UStG). Im eegFaktura-Core steuert `businessRole` nur die UI-Anzeige (Tab „Privat" vs. „Firma") — nicht die Steuerlogik (die kommt aus dem Tarif).

- (a) `EEG_PRIVATE` — der Kleinunternehmer ist steuerlich Privatperson, daher konsistent
- (b) `EEG_BUSINESS` — der Auftritt erfolgt mit Firmenname, daher landet er in eegFaktura unter dem Firma-Tab, was visuell und für den Admin intuitiver ist
- (c) konfigurierbar pro EEG (Overkill)

**Empfehlung:** (b). Begründung: das Name-Mapping (`company_name` → `firstname`) ist identisch zu `company`/`municipality`/`association`. Wenn wir `EEG_PRIVATE` setzen, würde eegFaktura den Eintrag unter „Privat" anzeigen, dort aber den Firmennamen im Vornamen-Feld zeigen — verwirrend. (b) ist konsistent.

### Q2: Migration bestehender `private`-Anträge

Im aktuellen Datenbestand wissen wir nicht, welche der `private`-Anträge eigentlich Kleinunternehmer sind. Die Information war im UI nicht differenzierbar.

- (a) Keine automatische Migration — Altdaten bleiben `private`, Admins klassifizieren manuell um, falls erforderlich
- (b) Backfill-Script, das nach Heuristiken (z.B. „company_name befüllt UND uid leer") umklassifiziert — riskant, weil `company_name` für `private` heute nicht existiert
- (c) Kein Backfill, aber One-time-Reminder per E-Mail an Admins der EEGs mit `private`-Anträgen

**Empfehlung:** (a). Es gibt im aktuellen Schema kein verlässliches Diskriminierungsmerkmal — jede Heuristik produziert False-Positives. Manuelle Korrektur durch Admins ist sauber.

### Q3: Geburtsdatum für Kleinunternehmer?

Beim heutigen `private`-Typ ist `birth_date` Pflicht. Ein Kleinunternehmer ist eigentlich auch eine natürliche Person.

- (a) Geburtsdatum entfällt — die Aussage des Users „nur Firmenname" ist wörtlich gemeint
- (b) Geburtsdatum bleibt Pflicht (steuerliche/Identifikations-Information)
- (c) Geburtsdatum wird optional

**Empfehlung:** (a). Die Userforderung ist eindeutig („nur Firmenname"). Falls eine EEG das Geburtsdatum doch benötigt, kann das über PROJ-8 (Konfigurierbare Felder pro EEG) zusätzlich aktiviert werden.

### Q4: E-Mail-Anrede für Kleinunternehmer

Heute werden Bestätigungs- und Approval-Mails an `private`-Mitglieder mit Vorname/Nachname personalisiert.

- (a) Neutrale Anrede „Sehr geehrte Damen und Herren" + Firmenname im Body (wie bei `company`)
- (b) Anrede „Sehr geehrter Kleinunternehmer {Firmenname}"
- (c) Anrede „Sehr geehrte/r {Firmenname}" — direkt mit Firmenname (kann grammatikalisch holpern bei „Sehr geehrter Maier IT")

**Empfehlung:** (a). Konsistent mit `company`/`association`/`municipality`.

### Q5: Sonderlogik in `mapPersonName` für Kleinunternehmer mit explizit gefülltem Vornamen?

Heute überschreibt `mapPersonName` für Nicht-natürliche-Personen den Vornamen mit `company_name`, **außer** wenn `firstname` bereits gefüllt ist (Convention für Kontaktpersonen einer Firma). Für Kleinunternehmer sollte es **keine** Kontaktperson geben — der Firmenname ist die Person.

- (a) Für `kleinunternehmer`: immer `company_name` → `firstname`, niemals Override durch `firstname`
- (b) Wie bei `company`: falls `firstname` aus alten Daten existiert, behalten (Kontaktperson)

**Empfehlung:** (a). Das Formular zeigt das Vorname-Feld für `kleinunternehmer` gar nicht an, also kann es nicht legitim gefüllt sein. Falls doch (per externer API), sollte es ignoriert/geleert werden, um Inkonsistenzen zu vermeiden.

## Notes

- Spec ist klein im Umfang, aber berührt mehrere Schichten (DB-Constraint, Backend-Validation, Frontend-Form, Excel, PDF, E-Mail). Eine `/grill-me`-Runde für Q1+Q3 lohnt sich, bevor Implementation startet.
- Security-Review ist **nicht** erforderlich: keine neuen Endpoints, keine neuen Auth-Pfade, keine Schema-Änderung außer einem zusätzlich erlaubten Enum-Wert.

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
