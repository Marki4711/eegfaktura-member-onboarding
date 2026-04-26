# PROJ-21: Genehmigungs-Benachrichtigung mit Beitrittsbestätigung PDF

## Status: Deployed
**Created:** 2026-04-26
**Last Updated:** 2026-04-26

## Dependencies
- Requires: PROJ-6 (E-Mail-Benachrichtigungen) — nutzt bestehende SMTP-Infrastruktur
- Requires: PROJ-12 (SEPA-Lastschriftmandat PDF) — nutzt bestehende PDF-Generierungs-Infrastruktur (fpdf)
- Requires: PROJ-2 (Admin Review) — approved-Status-Übergang ist der Auslöser

## Hintergrund

Wenn ein Admin einen Antrag auf `approved` setzt, gibt es derzeit keine automatische Benachrichtigung an den EEG-Betreiber und kein Dokument, das den Beitritt des Mitglieds bestätigt. Die EEG benötigt ein revisionssicheres Dokument, das den Beitritt belegt, alle relevanten Daten des Mitglieds enthält, die erteilten Zustimmungen dokumentiert und den Bearbeitungsweg (Statusverlauf) nachvollziehbar macht. Dieses Dokument soll entweder ausgedruckt, oder digital abgelegt werden können. Ein Feld für die Mitgliedsnummer wird im Dokument freigelassen, damit es nach der Genehmigung handschriftlich oder per PDF-Reader ausgefüllt werden kann.

## User Stories

- Als **EEG-Betreiber** möchte ich automatisch eine E-Mail erhalten, wenn ein Antrag genehmigt wird, damit ich sofort über die neue Mitgliedschaft informiert bin.
- Als **EEG-Betreiber** möchte ich die Beitrittsbestätigung als PDF-Anhang erhalten, damit ich sie digital ablegen oder ausdrucken und unterschreiben kann.
- Als **EEG-Betreiber** möchte ich im PDF ein leeres Feld für die Mitgliedsnummer sehen, damit ich die EEG-interne Nummer nach der Aufnahme nachtragen kann.
- Als **EEG-Betreiber** möchte ich im PDF die erteilten Zustimmungen des Mitglieds (Datenschutz, Vereinsstatuten usw.) dokumentiert sehen, damit ich bei rechtlichen Fragen die Einwilligung nachweisen kann.
- Als **EEG-Betreiber** möchte ich den vollständigen Statusverlauf des Antrags im PDF sehen, damit der Bearbeitungsweg nachvollziehbar und revisionssicher dokumentiert ist.
- Als **Betreiber** möchte ich, dass ein Fehler bei der PDF-Generierung den Status-Übergang nicht blockiert, damit der Admin den Antrag trotzdem auf `approved` setzen kann.

## Acceptance Criteria

### Auslöser
- [ ] Die Benachrichtigung wird ausgelöst, wenn ein Antrag in den Status `approved` wechselt
- [ ] Auslöser ist der Admin-Statusübergang (beliebiger vorheriger Status → `approved`)
- [ ] Wird `approved` → `import_failed` → `approved` (Re-Approval), wird erneut eine E-Mail gesendet
- [ ] Hat die EEG keine `contact_email`, wird weder E-Mail noch PDF generiert (kein Fehler)

### E-Mail an EEG-Betreiber
- [ ] Empfänger: `contact_email` der zugehörigen EEG aus `registration_entrypoint`
- [ ] Betreff: „Mitgliedsantrag genehmigt – [Vorname Nachname / Firmenname] ([Referenznummer])"
- [ ] Inhalt: kurze Mitteilung dass der Antrag genehmigt wurde, Name des Mitglieds, Referenznummer, Hinweis auf den PDF-Anhang
- [ ] Die E-Mail ist auf Deutsch
- [ ] PDF als Anhang beigefügt (Dateiname: `beitrittsbestaetigung-[referenznummer].pdf`)

### PDF-Inhalt: Beitrittsbestätigung
Das PDF ist ein strukturiertes A4-Dokument mit folgendem Inhalt:

#### Kopfzeile
- [ ] Titel: „Beitrittsbestätigung"
- [ ] EEG-Name und RC-Nummer
- [ ] Ausstellungsdatum (Datum der Genehmigung)

#### Mitgliedsdaten
- [ ] Mitgliedstyp (Privatperson / Landwirt / Unternehmen)
- [ ] Name (Vorname + Nachname) oder Firmenname + UID/Firmenbuchnummer
- [ ] Geburtsdatum (falls vorhanden)
- [ ] Adresse (Straße, Hausnummer, PLZ, Ort)
- [ ] E-Mail-Adresse
- [ ] Telefon (falls vorhanden)

#### Bankverbindung
- [ ] IBAN
- [ ] Kontoinhaber (falls vorhanden)
- [ ] SEPA-Mandatsart (Basislastschrift / Firmenlastschrift / Per E-Mail)

#### Zählpunkte
- [ ] Tabelle: Zählpunktnummer, Richtung, Teilnahmefaktor

#### Erteilte Zustimmungen
- [ ] Liste aller Dokumente, denen zugestimmt wurde (Titel + URL)
- [ ] Datum der Zustimmung (= Einreichungsdatum)

#### Statusverlauf
- [ ] Tabelle: Status (von → nach), Zeitstempel, ggf. Kommentar aus Admin-Notiz

#### Mitgliedsnummer
- [ ] Sichtbares, beschriftetes Leerfeld: „Mitgliedsnummer: _________________________"
- [ ] Hinweis: „Wird von [EEG-Name] vergeben"

#### Konfigurierbare Felder (optional)
- [ ] Falls konfigurierbare Felder ausgefüllt sind (Wärmepumpe, Personenanzahl usw.): werden als zusätzlicher Abschnitt aufgeführt
- [ ] Leere Felder werden nicht aufgeführt

### Fehlerverhalten
- [ ] Schlägt die PDF-Generierung fehl, wird der Fehler geloggt; die E-Mail wird ohne PDF-Anhang gesendet (mit Hinweis „PDF konnte nicht generiert werden")
- [ ] Schlägt auch der E-Mail-Versand fehl, wird der Fehler geloggt; der Status-Übergang zu `approved` bleibt gültig
- [ ] Kein Absturz bei fehlenden optionalen Feldern (NULL-Werte werden stillschweigend ausgelassen)

### Template
- [ ] Neues E-Mail-Template: `internal/mail/templates/application_approved_eeg.html`
- [ ] PDF-Generator in `internal/pdf/` (eigene Datei, z. B. `approval_pdf.go`)
- [ ] Die E-Mail ist auf Deutsch

## Edge Cases

- **Antrag von Unternehmen:** Kein Vorname/Nachname im PDF/Mail; Firmenname + UID steht an erster Stelle
- **Keine `contact_email`:** Weder E-Mail noch PDF wird generiert; kein Fehler, kein Log-Warn (bereits in PROJ-6 definiertes Verhalten)
- **Statusverlauf ist leer / hat nur einen Eintrag:** Tabelle zeigt alle vorhandenen Einträge; ein einzelner Eintrag ist valide
- **Keine Zustimmungen gespeichert:** Abschnitt „Erteilte Zustimmungen" entfällt; kein Fehler
- **Re-Approval (approved → import_failed → approved):** Neue E-Mail wird gesendet; das PDF enthält den vollständigen Statusverlauf inkl. `import_failed`-Einträge
- **SMTP nicht konfiguriert (`SMTP_HOST` fehlt):** Kein Versuch, kein Fehler (NoOpMailService)
- **Konfigurierbare Felder alle leer:** Abschnitt entfällt komplett aus dem PDF

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)

### Betroffene Komponenten

Rein Backend — kein Frontend-Eingriff, keine neuen öffentlichen Endpunkte, keine DB-Migrationen.

```
internal/
  config/config.go                              ← erweitert: ADMIN_BASE_URL (bereits in PROJ-20 hinzugefügt, hier wiederverwendet)
  pdf/
    approval_pdf.go                             ← neu: ApprovalPDFGenerator Interface + FPDFApprovalGenerator
  mail/
    service.go                                  ← erweitert: MailService Interface + SendApprovalEmail-Methode
    templates/
      application_approved_eeg.html             ← neu: Genehmigungs-E-Mail-Template
  application/
    admin_service.go                            ← erweitert: ChangeStatus löst Approval-Mail aus
```

### Datenmodell-Erweiterungen

Keine DB-Änderungen — alle benötigten Daten existieren bereits:
- Antragsdaten: `shared.Application`
- Zählpunkte: `MeteringPointRepository.GetByApplicationID()`
- Statusverlauf: `StatusLogRepository.GetByApplicationID()`
- Zustimmungen: `DocumentConsentRepository.GetByApplicationID()`
- EEG-Kontaktdaten: `RegistrationEntrypointRepository.GetByRCNumber()`
- Konfigurierbare Felder: `FieldConfigRepository.Get()` (für PDF-Abschnitt)

### PDF-Generator: `internal/pdf/approval_pdf.go`

Neues Interface und Implementierung, analog zu `SEPAMandateGenerator` in `generator.go`:

```go
// ApprovalPDFData hält alle Daten für die Beitrittsbestätigung.
type ApprovalPDFData struct {
    // Kopfzeile
    EEGName         string
    RCNumber        string
    ApprovedAt      time.Time

    // Mitgliedsdaten
    MemberType      string   // "Privatperson" / "Landwirt" / "Unternehmen" / ...
    Firstname       string
    Lastname        string
    BirthDate       *time.Time
    CompanyName     string
    UIDNumber       string
    RegisterNumber  string
    Email           string
    Phone           string
    ResidentStreet       string
    ResidentStreetNumber string
    ResidentZip          string
    ResidentCity         string

    // Bankverbindung
    IBAN            string
    AccountHolder   string
    SepaMandateType string   // "Basislastschrift" / "Firmenlastschrift" (aus UseCompanySEPA + MemberType)

    // Zählpunkte
    MeteringPoints  []MeteringPointPDF

    // Zustimmungen
    Consents        []ConsentPDF

    // Statusverlauf
    StatusLog       []StatusLogPDF

    // Konfigurierbare Felder (nur befüllte, nicht-hidden)
    ConfigurableFields []ConfigurableFieldDisplay

    // Referenz
    ReferenceNumber string
}

type MeteringPointPDF struct {
    MeteringPoint       string
    Direction           string   // "Verbrauch" / "Einspeisung"
    ParticipationFactor int
}

type ConsentPDF struct {
    Title       string
    URL         string
    ConsentedAt time.Time
}

type StatusLogPDF struct {
    FromStatus string   // "" wenn nil (erster Eintrag)
    ToStatus   string
    Timestamp  time.Time
    Reason     string   // "" wenn nil
}

// ApprovalPDFGenerator erzeugt die Beitrittsbestätigung als PDF-Bytes.
type ApprovalPDFGenerator interface {
    GenerateApproval(data ApprovalPDFData) ([]byte, error)
}
```

**Implementierung** `FPDFApprovalGenerator` in `approval_pdf.go`:
- Bibliothek: `github.com/go-pdf/fpdf` (bereits im Projekt vorhanden, keine neue Abhängigkeit)
- Encoding: Windows-1252 via `charmap.Windows1252` (wie in `generator.go` etabliert)
- Layout: DIN A4, strukturierte Abschnitte mit fpdf-Tabellen und Trennlinien
- Abschnitte in dieser Reihenfolge: Kopfzeile → Mitgliedsdaten → Bankverbindung → Zählpunkte → Zustimmungen → Statusverlauf → Mitgliedsnummer-Leerfeld → optionale konfigurierbare Felder

**Fehlerverhalten:** `GenerateApproval` gibt `(nil, error)` zurück. Der Aufrufer loggt den Fehler und sendet die E-Mail ohne PDF-Anhang (inkl. Hinweistext im E-Mail-Body).

### E-Mail: `application_approved_eeg.html`

Neues Template in `internal/mail/templates/`. Template-Variablen:

```go
type approvedEEGTemplateData struct {
    MemberName      string   // "Vorname Nachname" oder Firmenname
    ReferenceNumber string
    EEGName         string
    PDFFailed       bool     // true wenn PDF-Generierung fehlschlug → Hinweistext im Body
}
```

Betreff (in `service.go` zusammengebaut):
```
"Mitgliedsantrag genehmigt – [MemberName] ([ReferenceNumber])"
```

Anhang-Dateiname: `beitrittsbestaetigung-[referenceNumber].pdf`

### MailService-Interface-Erweiterung

```go
// internal/mail/service.go

type MailService interface {
    SendSubmissionEmails(app, meteringPoints, entrypoint, fieldConfig, attachment)
    SendMemberConfirmation(app) error
    SendApprovalEmail(app *shared.Application, entrypoint *shared.RegistrationEntrypoint, pdfBytes []byte, pdfFailed bool) error
}
```

`NoOpMailService` implementiert die neue Methode als No-Op (gibt `nil` zurück).

`SMTPMailService.SendApprovalEmail`:
1. Bestimme `MemberName` (Vorname+Nachname oder Firmenname)
2. Prüfe `entrypoint.ContactEmail` — ist nil/leer: sofort `nil` zurückgeben, kein Log-Warn (bestehendes Verhalten aus PROJ-6)
3. Rendere `application_approved_eeg.html`
4. Sende E-Mail: mit PDF-Anhang wenn `pdfBytes != nil`, ohne Anhang wenn `pdfFailed = true`
5. Fehler beim Versand: loggen + zurückgeben (der Aufrufer ignoriert den Fehler und loggt seinerseits)

`approvalTpl` wird analog zu `memberTpl`/`eegTpl` einmalig bei `NewSMTPMailService` geparst und in `SMTPMailService` gehalten.

### Auslöse-Logik: `AdminApplicationService.ChangeStatus`

Der Auslöser sitzt in `admin_service.go` — konsistent mit PROJ-6, wo der Submit-Trigger in `application_service.go` sitzt. Der Admin-Service kennt bereits alle nötigen Repositories.

```
POST /api/admin/applications/{id}/status  { toStatus: "approved" }
  → ChangeStatus(id, "approved", reason, actorID)
      → Transition validieren (isAdminTransitionAllowed)
      → DB-Transaktion: UpdateStatusAdminTx + StatusLogEntry schreiben
      → Commit
      → toStatus == "approved"?
          JA →
              go func() {
                  // Daten laden (außerhalb der Transaktion, asynchron)
                  app      ← appRepo.GetByID(id)
                  mps      ← meteringRepo.GetByApplicationID(id)
                  statusLog ← statusLogRepo.GetByApplicationID(id)
                  consents  ← consentRepo.GetByApplicationID(id)
                  entrypoint ← entrypointRepo.GetByRCNumber(app.RCNumber)
                  fieldConfig ← fieldConfigRepo.Get(app.RCNumber)

                  // Fehler bei den Lookups: loggen, abbrechen
                  if err != nil { slog.Error(...); return }

                  // contact_email fehlt: stumm abbrechen (kein Fehler)
                  if entrypoint.ContactEmail == nil { return }

                  // PDF generieren
                  data := buildApprovalPDFData(app, mps, statusLog, consents, entrypoint, fieldConfig)
                  pdfBytes, pdfErr := s.approvalPDFGenerator.GenerateApproval(data)
                  pdfFailed := pdfErr != nil
                  if pdfFailed {
                      slog.Error("pdf: failed to generate approval PDF", "application_id", id, "error", pdfErr)
                  }

                  // E-Mail senden
                  if err := s.mailService.SendApprovalEmail(app, entrypoint, pdfBytes, pdfFailed); err != nil {
                      slog.Error("mail: failed to send approval email", "application_id", id, "error", err)
                  }
              }()
          NEIN → keine Aktion
      → ChangeStatusResponse zurückgeben
```

**Asynchron via Goroutine** — der HTTP-Handler bekommt sofort eine Antwort; E-Mail/PDF-Fehler blockieren den Status-Übergang nicht. Dies ist konsistent mit dem Versand-Muster aus PROJ-6 und PROJ-12.

### `AdminApplicationService` — neue Felder

```go
type AdminApplicationService struct {
    db                  *sql.DB
    appRepo             *ApplicationRepository
    meteringRepo        *MeteringPointRepository
    statusLogRepo       *StatusLogRepository
    fieldConfigRepo     *FieldConfigRepository
    entrypointRepo      *RegistrationEntrypointRepository
    consentRepo         *DocumentConsentRepository
    mailService         mail.MailService
    approvalPDFGenerator pdf.ApprovalPDFGenerator   // neu
}
```

`NewAdminApplicationService` erhält `approvalPDFGenerator pdf.ApprovalPDFGenerator` als neuen Parameter. In `cmd/server/main.go` wird `pdf.NewFPDFApprovalGenerator()` injiziert (neue Konstruktorfunktion in `approval_pdf.go`).

### `SepaMandateType`-Bestimmung im PDF

Der Typ wird aus bestehenden Feldern abgeleitet — kein neues DB-Feld:

```go
func resolveSepaMandateType(app *shared.Application, ep *shared.RegistrationEntrypoint) string {
    if !app.SepaMandateAccepted {
        return "Per E-Mail"
    }
    if ep.UseCompanySEPAMandate &&
        (app.MemberType == shared.MemberTypeCompany || app.MemberType == shared.MemberTypeAssociation) {
        return "Firmenlastschrift"
    }
    return "Basislastschrift"
}
```

### Wiring in `cmd/server/main.go`

```go
approvalPDFGen := pdf.NewFPDFApprovalGenerator()
adminService := application.NewAdminApplicationService(
    db, appRepo, meteringRepo, statusLogRepo, fieldConfigRepo,
    entrypointRepo, consentRepo, mailService, approvalPDFGen,
)
```

### Neue Abhängigkeiten

Keine neuen externen Abhängigkeiten. `github.com/go-pdf/fpdf` und `golang.org/x/text/encoding/charmap` sind bereits im Projekt vorhanden.

## QA Test Results

**QA Date:** 2026-04-26
**Result:** APPROVED — 1 bug fixed during QA (re-approval transition), no remaining critical/high bugs

### Acceptance Criteria Results

#### Auslöser
- [x] Benachrichtigung bei `approved`-Übergang: goroutine in `ChangeStatus()` nach `tx.Commit()` — PASS
- [x] Beliebiger vorheriger Status → `approved`: `adminTransitions` deckt submitted/under_review/needs_info → approved ab — PASS
- [x] **Re-Approval (`import_failed → approved`):** BUG gefunden und behoben — `StatusImportFailed: {StatusApproved}` wurde zu `adminTransitions` hinzugefügt; Test `TestFPDFApprovalGenerator_ReApprovalStatusLog` bestätigt PDF-Rendering mit import_failed-Einträgen — FIXED ✓
- [x] EEG ohne `contact_email`: goroutine bricht stumm ab — PASS

#### E-Mail an EEG-Betreiber
- [x] Empfänger: `*entrypoint.ContactEmail` — PASS
- [x] Betreff: „Mitgliedsantrag genehmigt – [Name] ([Referenz])": `TestSendApprovalEmail_SubjectContainsMemberNameAndRef` — PASS
- [x] Inhalt: Mitgliedname, Referenz, Anhang-Hinweis — PASS
- [x] PDF-Anhang: Dateiname `beitrittsbestaetigung-[referenznummer].pdf` — PASS (service.go L398)
- [x] E-Mail auf Deutsch: `TestApprovalTemplate_IsGerman` — PASS

#### PDF-Inhalt: Kopfzeile
- [x] Titel "Beitrittsbestätigung" — PASS
- [x] EEG-Name und RC-Nummer — PASS
- [x] Ausstellungsdatum: `data.ApprovedAt.Format("02.01.2006")` — PASS

#### PDF-Inhalt: Mitgliedsdaten
- [x] Mitgliedstyp — PASS
- [x] Name / Firmenname: `TestFPDFApprovalGenerator_CompanyMember` — PASS
- [x] Geburtsdatum (falls vorhanden): nil-Guard in `approval_pdf.go` — PASS
- [x] Adresse — PASS
- [x] E-Mail — PASS
- [x] Telefon (falls vorhanden) — PASS

#### PDF-Inhalt: Bankverbindung
- [x] IBAN (bedingt, nur wenn vorhanden): `if data.IBAN != ""` — PASS
- [x] Kontoinhaber (falls vorhanden) — PASS
- [x] SEPA-Mandatsart — PASS

#### PDF-Inhalt: Zählpunkte
- [x] Tabelle: Zählpunktnummer, Richtung, Teilnahmefaktor — PASS

#### PDF-Inhalt: Zustimmungen
- [x] Liste aller Dokumente + Zustimmungsdatum — PASS; leerer Abschnitt entfällt bei `len(data.Consents) == 0` — PASS

#### PDF-Inhalt: Statusverlauf
- [x] Tabelle: Von → Nach, Zeitstempel, Kommentar — PASS; `TestFPDFApprovalGenerator_LargeStatusLog` prüft Seitenumbruch — PASS

#### PDF-Inhalt: Mitgliedsnummer
- [x] Beschriftetes Leerfeld „Mitgliedsnummer: ___" — PASS
- [x] Hinweis „Wird von [EEG-Name] vergeben" — PASS

#### PDF-Inhalt: Konfigurierbare Felder
- [x] Optionaler Abschnitt nur bei befüllten Feldern: `TestFPDFApprovalGenerator_WithConfigurableFields` — PASS
- [x] Leere Felder nicht aufgeführt: `buildApprovalConfigurableFields()` — PASS

#### Fehlerverhalten
- [x] PDF-Generierung schlägt fehl → E-Mail ohne Anhang + Hinweis: `pdfFailed bool`-Parameter + `{{if .PDFFailed}}`-Block — PASS; `TestSendApprovalEmail_PDFFailedHintInBody` — PASS
- [x] E-Mail-Versand schlägt fehl → Fehler geloggt, Status-Übergang bleibt gültig (goroutine) — PASS
- [x] Kein Absturz bei NULL-Werten: `TestFPDFApprovalGenerator_EmptyOptionalFields` — PASS

#### Template
- [x] Neues Template `application_approved_eeg.html` — PASS
- [x] PDF-Generator in `internal/pdf/approval_pdf.go` — PASS
- [x] E-Mail auf Deutsch — PASS

### Bugs Found and Fixed

| Bug | Severity | Status |
|-----|----------|--------|
| `import_failed → approved` fehlte in `adminTransitions` — Re-Approval-Übergang wurde mit 409 Conflict abgelehnt | High | FIXED — `StatusImportFailed: {StatusApproved}` hinzugefügt |

### Security Smoke Test

- **Auth/Authz:** `ChangeStatus`-Handler prüft `checkTenantAccess` — korrekte Tenant-Isolation ✓
- **Injection:** Keine neuen SQL-Queries; PDF-Generator keine User-Inputs in Shell/Pfad ✓
- **XSS:** `html/template` auto-escaping; `TestApprovalTemplate_XSSEscaped` bestätigt ✓
- **Sensible Daten in PDF:** IBAN im PDF by-design (interne Admin-Dokumentation, Spec-Anforderung) ✓
- **PDF-Dateiname:** `beitrittsbestaetigung-[referenceNumber].pdf` — ReferenceNumber ist systemgeneriert, kein Path Traversal möglich ✓
- **Dependencies:** `govulncheck ./...` — No vulnerabilities ✓

### Tests Written

**Neue Unit Tests (`internal/pdf/approval_pdf_test.go`):**
- `TestFPDFApprovalGenerator_GeneratesValidPDF`
- `TestFPDFApprovalGenerator_OutputSizeReasonable`
- `TestFPDFApprovalGenerator_ContainsXRefTable`
- `TestFPDFApprovalGenerator_EmptyOptionalFields`
- `TestFPDFApprovalGenerator_CompanyMember`
- `TestFPDFApprovalGenerator_UmlautsEncoded`
- `TestFPDFApprovalGenerator_WithConfigurableFields`
- `TestFPDFApprovalGenerator_ReApprovalStatusLog`
- `TestFPDFApprovalGenerator_DifferentFromSEPA`
- `TestFPDFApprovalGenerator_LargeStatusLog`

**Neue Unit Tests (`internal/mail/service_test.go`):**
- `TestSendApprovalEmail_SendsToContactEmail`
- `TestSendApprovalEmail_SkipsWhenNoContactEmail`
- `TestSendApprovalEmail_SkipsWhenContactEmailEmpty`
- `TestSendApprovalEmail_SubjectContainsMemberNameAndRef`
- `TestSendApprovalEmail_PDFFailedHintInBody`
- `TestSendApprovalEmail_CompanyMember`
- `TestApprovalTemplate_IsGerman`
- `TestApprovalTemplate_XSSEscaped`

Alle Tests: `go test ./... -count=1` — **PASS**

## Deployment

**Deployed:** 2026-04-26
**Image SHA:** `sha-ab0b64e`
**Helm release:** `eegfaktura-member-onboarding`
**Hinweis:** Kein neues Pflicht-Env-Var; `ADMIN_BASE_URL` aus PROJ-20 wird wiederverwendet (optional)
