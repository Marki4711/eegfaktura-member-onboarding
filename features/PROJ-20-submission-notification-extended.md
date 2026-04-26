# PROJ-20: Vollständige Antragsdaten in EEG-Einreichungsbenachrichtigung

## Status: In Progress
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

## Dependencies
- Requires: PROJ-6 (E-Mail-Benachrichtigungen) — erweitert das bestehende EEG-Benachrichtigungs-Template

## Hintergrund

Die bestehende EEG-Einreichungsbenachrichtigung (PROJ-6) enthält nur: Name, E-Mail-Adresse, Referenznummer und Zählpunkte. Der EEG-Betreiber erhält damit nicht alle relevanten Antragsdaten und muss sich im Admin-Bereich einloggen, um Vollständiges einzusehen. Die erweiterte E-Mail soll alle eingereichten Informationen enthalten, sodass der Betreiber den Antrag auch direkt aus der E-Mail heraus vollständig beurteilen kann.

## User Stories

- Als **EEG-Betreiber** möchte ich in der Einreichungs-E-Mail alle Antragsdaten des Mitglieds sehen (Adresse, IBAN, Telefon, Geburtsdatum usw.), damit ich den Antrag ohne Login in den Admin-Bereich beurteilen kann.
- Als **EEG-Betreiber** möchte ich die Mitgliedsart (Privatperson / Landwirt / Unternehmen) in der E-Mail erkennen können, damit ich sofort den richtigen Bearbeitungspfad einschlagen kann.
- Als **EEG-Betreiber** möchte ich alle angemeldeten Zählpunkte mit Richtung und Teilnahmefaktor sehen, damit ich die technische Planung sofort anstoßen kann.
- Als **EEG-Betreiber** möchte ich sehen, welchen konfigurierbaren Felder der Antragsteller ausgefüllt hat (z. B. Wärmepumpe vorhanden, Personenanzahl), damit ich ein vollständiges Bild des Haushalts habe.
- Als **Betreiber** möchte ich, dass ein Fehler beim Template-Rendering den Versand nicht blockiert, damit das Mitglied den Antrag trotzdem erfolgreich einreichen kann.

## Acceptance Criteria

### Antragstellerdaten
- [ ] Die E-Mail enthält alle Antragstellerdaten je nach Mitgliedstyp:
  - **Privatperson / Landwirt:** Vorname, Nachname, Geburtsdatum (falls ausgefüllt)
  - **Unternehmen:** Firmenname, UID-Nummer (falls ausgefüllt), Firmenbuchnummer (falls ausgefüllt)
- [ ] Kontaktdaten: E-Mail-Adresse, Telefon (falls ausgefüllt)
- [ ] Wohnadresse: Straße, Hausnummer, PLZ, Ort
- [ ] Mitgliedstyp (Privatperson / Landwirt / Unternehmen) ist deutlich erkennbar

### Bankverbindung
- [ ] IBAN (vollständig — kein Masking in der EEG-internen E-Mail)
- [ ] Kontoinhaber (falls verschieden vom Antragsteller)
- [ ] Art der SEPA-Ermächtigung: Basislastschrift / Firmenlastschrift / Per E-Mail

### Konfigurierbare Felder
- [ ] Alle konfigurierbaren Felder, die nicht auf `hidden` gesetzt sind und einen Wert haben, werden aufgelistet (z. B. Wärmepumpe: Ja, Personen im Haushalt: 3)
- [ ] Felder ohne Wert werden nicht aufgeführt

### Zählpunkte
- [ ] Bestehende Darstellung bleibt: Zählpunktnummer, Richtung (Verbrauch / Einspeisung), Teilnahmefaktor

### Zusatzinformationen
- [ ] Referenznummer
- [ ] Einreichungsdatum und -uhrzeit
- [ ] Link zur Admin-Detailansicht (direkter Link auf `/admin/applications/{id}`)
- [ ] RC-Nummer der EEG

### Fehlerverhalten
- [ ] Schlägt das Rendering fehl, wird der Fehler geloggt; der Einreichungs-Endpunkt gibt dennoch `200 OK` zurück
- [ ] Kein Absturz bei fehlenden optionalen Feldern (NULL-Werte werden stillschweigend ausgelassen)

### Template
- [ ] Das bestehende Template `application_submitted_eeg.html` wird ersetzt — kein neues Template
- [ ] Die E-Mail ist auf Deutsch
- [ ] Klar gegliedertes HTML-Layout (Abschnitte: Antragsteller, Bankverbindung, Zählpunkte, Zusätzliche Informationen)

## Edge Cases

- **IBAN fehlt:** Kann bei SEPA per E-Mail nicht vorkommen (kein Pflichtfeld in diesem Modus) → Abschnitt Bankverbindung wird ausgelassen wenn IBAN leer
- **Mitglied ist Unternehmen:** Vorname/Nachname werden nicht angezeigt; Firmenname steht an oberster Stelle
- **Kein einziges konfigurierbares Feld ausgefüllt:** Abschnitt „Zusätzliche Informationen" entfällt komplett
- **EEG hat keine `contact_email`:** Kein Versuch, keine Fehlermeldung (bleibt unverändert wie in PROJ-6)
- **Backend-URL nicht konfiguriert:** Link zur Admin-Detailansicht entfällt aus dem Template — kein Fehler

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)

### Betroffene Komponenten

Rein Backend — kein Frontend-Eingriff, keine neuen Endpunkte, keine DB-Migrationen.

```
internal/
  config/config.go                            ← erweitert: ADMIN_BASE_URL-Feld
  mail/
    service.go                                ← erweitert: eegTemplateData + Rendering-Logik
    templates/
      application_submitted_eeg.html          ← ersetzt: vollständige Antragsdaten
```

Keine neuen Pakete, keine neuen Abhängigkeiten.

### Datenmodell-Erweiterungen

Keine DB-Änderungen — alle benötigten Daten sind bereits in `shared.Application`, `shared.MeteringPoint` und `shared.RegistrationEntrypoint` vorhanden.

### Template-Variablen: `eegTemplateData`

Der bestehende `eegTemplateData`-Struct in `internal/mail/service.go` wird durch einen vollständigen Struct ersetzt:

```go
type eegTemplateData struct {
    // Identifikation
    ReferenceNumber string
    SubmittedAt     string   // formatiert als "02.01.2006 15:04"
    RCNumber        string

    // Mitgliedstyp
    MemberType      string   // "Privatperson" / "Landwirt" / "Unternehmen" / ...

    // Person (nur bei private / farmer)
    Firstname       string
    Lastname        string
    BirthDate       string   // formatiert als "02.01.2006" oder "" wenn nil

    // Unternehmen / Organisation (nur bei company / municipality / association)
    CompanyName     string
    UIDNumber       string
    RegisterNumber  string

    // Kontakt
    Email           string
    Phone           string   // "" wenn nil

    // Adresse
    ResidentStreet       string
    ResidentStreetNumber string
    ResidentZip          string
    ResidentCity         string

    // Bankverbindung
    IBAN            string   // "" wenn nil (Abschnitt wird im Template ausgelassen)
    AccountHolder   string   // "" wenn nil

    // Zählpunkte
    MeteringPoints  []shared.MeteringPoint

    // Konfigurierbare Felder (gefiltert: nur nicht-hidden, nicht leer)
    ConfigurableFields []ConfigurableFieldDisplay

    // Admin-Link (leer wenn ADMIN_BASE_URL nicht konfiguriert)
    AdminDetailURL  string
}

// ConfigurableFieldDisplay ist ein aufgelöster Name-Wert-Eintrag für das Template.
type ConfigurableFieldDisplay struct {
    Label string   // lesbarer Feldname auf Deutsch
    Value string   // formatierter Wert (bool → "Ja"/"Nein", int → Zahl, etc.)
}
```

### Optionale Felder: NULL-Behandlung im Template

Alle optionalen Felder werden in `service.go` vor der Template-Übergabe zu leeren Strings aufgelöst (kein Template-seitiges Dereferenzieren von Pointern). Fehlende optionale Werte ergeben `""`.

Das Template nutzt Go's `{{if .FieldName}}` um Abschnitte auszublenden:

```html
{{if .BirthDate}}
<tr><th>Geburtsdatum</th><td>{{.BirthDate}}</td></tr>
{{end}}

{{if .IBAN}}
<h3>Bankverbindung</h3>
...
{{end}}

{{if .ConfigurableFields}}
<h3>Zusätzliche Informationen</h3>
...
{{end}}
```

### Konfigurierbare Felder: Filterlogik

In `service.go` wird eine Hilfsfunktion `buildConfigurableFields(app, fieldConfig)` eingeführt:

```
for each known configurable field:
    entry := fieldConfig[fieldName]
    if entry.State == "hidden"  → skip
    if value is nil/zero        → skip
    append ConfigurableFieldDisplay{Label: "<Deutsch>", Value: "<formatiert>"}
```

Feldnamen-zu-Label-Mapping (statische Map in `service.go`):
- `persons_in_household` → „Personen im Haushalt"
- `consumption_previous_year` → „Verbrauch Vorjahr (kWh)"
- `consumption_forecast` → „Verbrauch Prognose (kWh)"
- `feed_in_forecast` → „Einspeisung Prognose (kWh)"
- `pv_power_kwp` → „PV-Leistung (kWp)"
- `heat_pump` → „Wärmepumpe vorhanden"
- `electric_vehicle` → „Elektrofahrzeug vorhanden"
- `electric_hot_water` → „Warmwasser elektrisch"
- `membership_start_date` → „Beitrittsdatum"

Da `SendSubmissionEmails` bereits `entrypoint *shared.RegistrationEntrypoint` enthält, wird der FieldConfig-Lookup ebenfalls im Service (nicht im Handler) durchgeführt. Der `SMTPMailService` erhält dafür Zugriff auf den `FieldConfigRepository` — oder alternativ wird der aufgelöste `fieldConfig`-Wert als zusätzlicher Parameter übergeben.

**Entscheidung:** Der aufgelöste `fieldConfig map[string]FieldConfigEntry` wird als neuer Parameter an `SendSubmissionEmails` übergeben. Das Interface ändert sich entsprechend:

```go
// Vorher
SendSubmissionEmails(app, meteringPoints, entrypoint, attachment []byte)

// Nachher
SendSubmissionEmails(app, meteringPoints, entrypoint, fieldConfig map[string]FieldConfigEntry, attachment []byte)
```

`FieldConfigEntry` wird aus dem `application`-Paket exportiert oder in `shared` verschoben (bestehende interne Typ-Definition wird wiederverwendet). Da `mail` nicht von `application` importieren darf, wird ein reduzierter `FieldState`-Typ in `shared` definiert:

```go
// internal/shared/models.go (Erweiterung)
type FieldConfigMap map[string]FieldConfigState

type FieldConfigState struct {
    State      string
    AdminValue *string
}
```

Alternativ (einfacher): `map[string]string` für `state` ohne `AdminValue` — ausreichend für das Template-Rendering.

**Endentscheidung (einfachstes Modell):** `fieldConfig map[string]string` (Feldname → State: `"visible"`, `"required"`, `"hidden"`, `"admin_only"`). Dieses Modell ist bereits in `RegistrationConfig.FieldConfig` als `map[string]string` vorhanden und kann direkt verwendet werden.

Der Aufrufer in `application_service.go` baut bereits `fieldConfig map[string]FieldConfigEntry`. Vor dem Goroutinen-Aufruf wird eine Funktion `toStateMap(fieldConfig map[string]FieldConfigEntry) map[string]string` aufgerufen, die nur den `State`-String extrahiert.

### Admin-Link: neue Env-Variable `ADMIN_BASE_URL`

```go
// internal/config/config.go — Erweiterung
type Config struct {
    ...
    AdminBaseURL string   // ADMIN_BASE_URL — optional; leer = kein Link
}
```

```bash
# .env.local.example
ADMIN_BASE_URL=https://admin.example.com
```

Link-Konstruktion in `service.go`:
```go
adminDetailURL := ""
if adminBaseURL != "" {
    adminDetailURL = adminBaseURL + "/admin/applications/" + app.ID.String()
}
```

`adminBaseURL` wird einmalig beim Start in `SMTPMailService` injiziert (neues Feld `adminBaseURL string`).

### Auslöse-Logik (unverändert zu PROJ-6)

```
POST /api/public/applications/{id}/submit
  → SubmitApplication()
      → Status-Übergang draft → submitted
      → fieldConfig laden (bereits vorhanden)
      → toStateMap(fieldConfig) aufrufen
      → go s.mailService.SendSubmissionEmails(app, meteringPoints, entrypoint, stateMap, attachment)
```

Der Goroutinen-Aufruf bleibt asynchron — kein Blockieren des Request-Pfads.

### Template-Struktur

`application_submitted_eeg.html` wird mit klar gegliederten Abschnitten neu geschrieben:

1. **Antragsteller** — Mitgliedstyp, Name/Firmenname, Geburtsdatum (wenn vorhanden), Kontakt, Adresse
2. **Bankverbindung** — IBAN, Kontoinhaber (nur wenn IBAN vorhanden)
3. **Zählpunkte** — Tabelle: Nummer, Richtung, Teilnahmefaktor
4. **Zusätzliche Informationen** — konfigurierbare Felder (nur wenn mindestens ein Wert vorhanden)
5. **Referenz** — Referenznummer, Einreichungsdatum, RC-Nummer, Admin-Link (wenn konfiguriert)

XSS-Schutz: `html/template` escapet automatisch — unverändertes Verhalten aus PROJ-6.

### Neue Abhängigkeiten

Keine neuen externen Abhängigkeiten.

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
