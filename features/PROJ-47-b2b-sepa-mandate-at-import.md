# PROJ-47: B2B-SEPA-Firmenlastschrift-Mandat mit Mandatsreferenz beim Import

**Status:** In Review
**Created:** 2026-05-17

## Hintergrund

Das B2B-SEPA-Firmenlastschriftverfahren verlangt, dass die
**Mandatsreferenz** im Mandat-PDF stehen muss, bevor es bei der Bank
hinterlegt wird. Im Onboarding wird die Mandatsreferenz aus der
Mitgliedsnummer abgeleitet — die aber erst beim Import in den Core
vergeben wird (PROJ-46-Hintergrund).

Vor diesem Feature lieferte das Onboarding bereits beim Submit ein
Firmenlastschrift-PDF aus — aber **ohne Mandatsreferenz**
(`SEPAMandateData` hatte schlichtweg kein Feld dafür). PROJ-46 Stage B
verschob die Beitrittsbestätigung an den Import-Zeitpunkt, aber das
separate Firmenlastschrift-PDF wurde dort noch nicht generiert. Diese
Lücke wird hier geschlossen.

## Änderungen

### PDF-Generator (`internal/pdf/generator.go`)

`SEPAMandateData` bekommt ein neues optionales Feld:
- `MandateReference string` — Wenn leer: alter Platzhalter
  „Mandatsreferenz (wird von … ausgefüllt):" (für die Submit-Zeit-PDF
  unverändert). Wenn gesetzt: druckt **`Mandatsreferenz: <Wert>`**
  prominent im PDF.

Beide Renderer (`Generate` für Basislastschrift, `GenerateCompany` für
Firmenlastschrift) verwenden das gleiche Conditional.

### Mailer (`internal/mail/mailer.go`)

Neues `Attachment`-Struct + `Sender.SendWithAttachments`-Methode (mehrere
Anhänge in einem Send). Bestehende `SendWithAttachment`-Methode bleibt
und delegiert intern an die Multi-Variante.

### MailService (`internal/mail/service.go`)

`SendImportedNotification` bekommt einen zusätzlichen Parameter
`b2bMandatePDF []byte`. Wenn non-nil, wird er als zweiter Anhang
mitgeschickt (Dateiname `sepa-firmenlastschrift-mandat-<Mitgliedsnr>.pdf`).
Sowohl Member-Mail als auch EEG-Kopie erhalten beide Anhänge.

### Admin-Service (`internal/application/admin_service.go`)

`AdminApplicationService` bekommt eine neue Dependency
`sepaMandateGenerator pdf.SEPAMandateGenerator`. In
`SendPostImportNotification`:
1. Wenn `app.Einzugsart == "b2b"` UND `buildSEPAMandateData` liefert
   Daten (EEG-Adresse + CreditorID gepflegt)
2. setze `mandate.MandateReference = *app.MemberNumber`
3. setze `mandate.MemberName = *app.CompanyName` (B2B-Debtor ist die
   Firma, nicht die Kontaktperson)
4. lade Logo (analog Beitrittsbestätigung)
5. rufe `sepaMandateGenerator.GenerateCompany(mandate)` auf
6. übergebe das resultierende PDF als zweiten Anhang

Fehler beim B2B-PDF-Build blockieren die Hauptmail NICHT — Best-Effort
mit Log-Warnung.

### Mail-Template (`application_imported_member.html`)

Der bestehende b2b-Hinweis-Block wird um einen Hinweis erweitert:
„Anhang dieser E-Mail: die druckbare Firmenlastschrift-Mandats-PDF
(Dateiname „sepa-firmenlastschrift-mandat-<Mitgliedsnr>.pdf") mit
eingedruckter Mandatsreferenz — bitte unterschreiben und Ihrer Hausbank
vorlegen."

## Was passiert wann

| Trigger | Member-Mail Anhänge |
|---|---|
| `→ submitted` (unverändert) | Submit-Zeit-SEPA-PDF (mit Mandatsreferenz-Platzhalter) — wie bisher |
| `→ imported` (b2b auto-routing) | **Beitrittsbestätigung** (Mitgliedsnr.) + **Firmenlastschrift-Mandat mit Mandatsreferenz=Mitgliedsnr.** |
| `→ imported` (non-b2b) | Nur **Beitrittsbestätigung** (wie bisher) |
| `→ activated` (unverändert) | keine Anhänge |

## Out of Scope

- Update der Submit-Zeit-PDF mit Mandatsreferenz — die Mitgliedsnummer
  existiert dort noch nicht. Bleibt absichtlich beim Platzhalter.
- Member-Self-Service-Bestätigung der Bank-Pre-Notification — siehe
  PROJ-46 Entscheidung A (Admin-manuell bleibt).
- Versand des B2B-Mandats per Post — nur E-Mail-Anhang.

## Tests

- Build muss grün bleiben
- Smoke-Test: Antrag mit `einzugsart=b2b` + Member-Type „Unternehmen" →
  Import → Member-Mail muss zwei PDF-Anhänge enthalten, das zweite mit
  „Mandatsreferenz: <Mitgliedsnr.>" gedruckt
- Smoke-Test: Antrag mit `einzugsart=core` → Member-Mail hat genau
  einen Anhang (Beitrittsbestätigung)
- Smoke-Test: Antrag mit `einzugsart=b2b` aber EEG hat keine
  CreditorID gepflegt → kein 2. Anhang, Log-Info, Hauptmail geht raus
