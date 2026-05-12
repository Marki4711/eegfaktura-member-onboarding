# PROJ-28: Trennung von „Privat" und „Kleinunternehmer"

## Status: Planned
**Created:** 2026-05-12
**Last Updated:** 2026-05-12 (Q1–Q5 resolved nach Empfehlungen)

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

## Resolved Decisions

Alle Open Questions wurden am 2026-05-12 nach den jeweiligen Empfehlungen entschieden. Die ACs oben spiegeln diesen Stand bereits wider.

### Q1: `businessRole` für Kleinunternehmer — **RESOLVED**

**Entscheidung:** `EEG_BUSINESS`.

`businessRole` steuert in eegFaktura nur die UI-Anzeige (Tab „Privat" vs. „Firma"), nicht die Steuerlogik (kommt aus dem Tarif). Da das Name-Mapping (`company_name` → `firstname`) identisch zu `company`/`municipality`/`association` ist, ist `EEG_BUSINESS` die konsistente Wahl — der Eintrag erscheint unter „Firma" mit dem Firmennamen im Vornamen-Feld.

### Q2: Migration bestehender `private`-Anträge — **RESOLVED**

**Entscheidung:** Keine automatische Migration.

Im aktuellen Schema gibt es kein verlässliches Diskriminierungsmerkmal zwischen Privatpersonen und Kleinunternehmern unter den Altdaten. Heuristiken produzieren False-Positives. Bestandsanträge bleiben `private`; Admins klassifizieren bei Bedarf manuell um.

### Q3: Geburtsdatum für Kleinunternehmer — **RESOLVED**

**Entscheidung:** Geburtsdatum entfällt für `kleinunternehmer`.

Die Userforderung ist eindeutig („nur Firmenname"). Falls eine EEG das Geburtsdatum doch benötigt, kann es über PROJ-8 (Konfigurierbare Felder pro EEG) optional aktiviert werden.

### Q4: E-Mail-Anrede für Kleinunternehmer — **RESOLVED**

**Entscheidung:** Neutrale Anrede „Sehr geehrte Damen und Herren" + Firmenname im Body, analog zu `company`/`association`/`municipality`.

### Q5: Sonderlogik in `mapPersonName` für Kleinunternehmer — **RESOLVED**

**Entscheidung:** Für `kleinunternehmer` immer `company_name` → `firstname`, niemals Override durch ein eventuell gefülltes `firstname`-Feld.

Das öffentliche Formular zeigt das Vorname-Feld für `kleinunternehmer` nicht an, daher kann es nicht legitim gefüllt sein. Eingehende `firstname`-Werte (z.B. über die externe API PROJ-13) werden ignoriert und in `mapPersonName` ausschließlich der `company_name` verwendet.

## Notes

- Spec ist klein im Umfang, aber berührt mehrere Schichten (DB-Constraint, Backend-Validation, Frontend-Form, Excel, PDF, E-Mail).
- Alle Open Questions sind entschieden — die Spec ist bereit für `/architecture`.
- Security-Review ist **nicht** erforderlich: keine neuen Endpoints, keine neuen Auth-Pfade, keine Schema-Änderung außer einem zusätzlich erlaubten Enum-Wert.

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
