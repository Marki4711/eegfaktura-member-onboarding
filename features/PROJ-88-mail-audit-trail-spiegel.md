# PROJ-88: Mail-Templates auf Audit-Trail-Variante umstellen (PROJ-78-Lücke)

## Status: Approved
**Created:** 2026-06-08
**Last Updated:** 2026-06-08
**Typ:** Bug-Fix (PROJ-78-Lücke, High Severity)

## Hintergrund

Tester-Feedback 2026-06-08:

> Obwohl die Option Audit-Trail für die B2B SEPA-Lastschrift aktiv war,
> wurde im Email um Unterschrift der SEPA-Lastschrift gebeten. Bitte
> überprüfen.
>
> Text war:
> „Anhang dieser E-Mail: die druckbare Firmenlastschrift-Mandats-PDF
> (Dateiname „sepa-firmenlastschrift-mandat-404.pdf") mit eingedruckter
> Mandatsreferenz — bitte unterschreiben und deiner Hausbank vorlegen."

Owner-Direktive 2026-06-08:
> „A ja" — Wortlaut-Vorschläge angenommen, direkt umsetzen wie PROJ-86/87.

## Diagnose

PROJ-78 (Elektronisches SEPA-Mandat, Toggle) hat das **PDF**-Rendering
auf den Audit-Trail-Block umgestellt (keine Unterschrift-Linie mehr).
Aber die **Mail-Templates** wurden nicht mitgezogen — sie sagen weiter
„bitte unterschreiben". Gleicher Bug auf CORE-Audit- und B2B-Audit-Pfad.

Konkret betroffen:
- `internal/mail/templates/application_imported_member.html` — Mitglied
  bekommt Aufforderung zur Unterschrift trotz Audit-PDF
- `internal/mail/templates/application_imported_eeg.html` — EEG bekommt
  Aufforderung „warte auf unterschriebenes Mandat" trotz Audit-PDF

Pfad heute: `mandateAtImportData`-Struct hat `IsB2B bool`, aber keine
Audit-bezogenen Felder. Templates haben `{{if .IsB2B}}…{{else}}…{{end}}`
ohne Audit-Differenzierung.

## Owner-Direktive 2026-06-08

> „A" — Wortlaut-Vorschläge approbiert, Direkt-Hotfix.

Approbierte Wortlaut-Tabelle:

| Pfad | Mail-Text |
|---|---|
| **CORE Klassik** (Audit OFF) | „unterschreibe und sende zurück" + ID-Austria-Anleitung — **unverändert** |
| **CORE Audit** (Audit ON) | „dein SEPA-Mandat ist mit elektronisch dokumentierter Zustimmung beigefügt. Keine weitere Aktion nötig." |
| **B2B Klassik** (Audit OFF) | „unterschreiben + Hausbank vorlegen" — **unverändert** |
| **B2B Audit** (Audit ON) | „dein B2B-Mandat ist mit elektronisch dokumentierter Zustimmung beigefügt. Hausbank-Pre-Notification bleibt B2B-Rulebook-Pflicht, aber keine Unterschrift mehr nötig." |

## Acceptance Criteria

### Daten-Struct

- [x] **AC-1** `mandateAtImportData` um zwei neue Felder erweitert:
  `CoreAuditActive bool`, `B2BAuditActive bool`
- [x] **AC-2** `buildMandateAtImportData` setzt beide Felder aus
  `ep.SEPAMandateCoreAuditEnabled` / `ep.SEPAMandateB2BAuditEnabled`

### Member-Template

- [x] **AC-3** 4-Branch-Logik: Klassik-CORE / Audit-CORE / Klassik-B2B /
  Audit-B2B
- [x] **AC-4** Audit-Varianten haben **keine** Unterschriftsaufforderung
  und **keine** Rücksende-Aufforderung
- [x] **AC-5** B2B-Audit-Variante behält den **Hausbank-Pre-Notification**-
  Hinweis (das ist SEPA-Rulebook, unabhängig vom Mandat-Modus)
- [x] **AC-6** Owner-approbierte Wortlaute exakt übernommen

### EEG-Template

- [x] **AC-7** 4-Branch-Logik analog zum Member-Template
- [x] **AC-8** CORE-Audit-Variante sagt „Ablage-Kopie geht separat" und
  „kann Lastschriften sofort einziehen" — keine „warte auf unter-
  schriebenes Mandat" mehr
- [x] **AC-9** B2B-Audit-Variante sagt „Mitglied hat elektronisch
  zugestimmt" + „Bank-Rückmeldung abwarten" (Pre-Notification-Workflow
  bleibt erhalten)

### Tests

- [x] **AC-10** Vier neue Integration-Tests in
  `internal/mail/service_test.go`:
  - `TestSendMandateAtImport_CoreKlassik_AsksForSignature`
  - `TestSendMandateAtImport_CoreAudit_DoesNotAskForSignature`
  - `TestSendMandateAtImport_B2BKlassik_AsksForSignatureAndBank`
  - `TestSendMandateAtImport_B2BAudit_NoSignatureButKeepBankPreNotification`
- [x] **AC-11** Tests verifizieren Wortlaut-Schlüssel via konstantem
  String-Anker (vermeidet Drift bei zukünftigen Wortlaut-Anpassungen
  durch Owner) — siehe `wortlaut*`-Konstanten
- [x] **AC-12** B2B-Audit-Test verifiziert explizit, dass
  Pre-Notification-Hinweis trotzdem erhalten bleibt — das ist die
  SEPA-Rulebook-Pflicht, die einen Edge-Case-Bug auslösen würde, wenn
  sie versehentlich entfernt würde

### Build + Doku

- [x] **AC-13** `go build ./...` clean
- [x] **AC-14** `go test ./...` 14 Pakete grün, neue Tests in `mail`
- [x] **AC-15** CHANGELOG.md + User-Guide-Changelog ergänzt
- [x] **AC-16** Spec dokumentiert die 4 Branches und die Owner-
  Wortlaut-Approval

## Edge Cases

- **EC-1 PROJ-80 EEG-Kopie bei Audit-Modus:** PROJ-80 hat die Logik
  eingeführt, dass die EEG bei Audit-Variante eine Ablage-Kopie der
  Member-Mail bekommt (siehe `service.go:980-1003`). PROJ-88 ändert die
  EEG-Kopie-Wortlaute mit, damit der Text zur Action passt.
- **EC-2 PROJ-70 Renewal-Pfad:** Renewal-Mails benutzen dieselbe
  Template-Datei. Wenn die EEG den Audit-Toggle umstellt und dann ein
  Renewal-Mail rausgeht, sieht der neue Empfänger die Audit-Variante —
  das ist konsistent zur tatsächlichen PDF-Variante.
- **EC-3 Owner stellt Audit-Toggle inkonsistent ein:** z.B. CORE-Audit
  ON, B2B-Audit OFF. Wenn ein b2b-Antrag importiert wird, rendert das
  Template die B2B-Klassik-Variante (nicht die Audit-Variante). Korrekt
  — der B2B-Audit-Toggle ist die Wahrheit, nicht der CORE-Audit-Toggle.
- **EC-4 Mitglied verwirrt durch Audit-Wortlaut:** der neue Wortlaut
  „elektronisch dokumentiert (formfreie Willenserklärung gem. § 76 (3)
  EIWOG)" ist juristisch korrekt aber für Laien unverständlich. Owner
  hat das in der Approval bewusst akzeptiert — der Verweis auf den
  Paragraphen ist Rechtsschutz für die EEG (und im PDF dasselbe Wording).

## Tech Design

```
Vorher:
  IsB2B?
    ja → "unterschreiben + Hausbank vorlegen"
    nein → "unterschreibe und sende zurück"

Nachher (PROJ-88):
  IsB2B?
    ja:
      B2BAuditActive?
        ja → "elektronisch dokumentiert" + "Pre-Notification an Hausbank"
        nein → "unterschreiben + Hausbank vorlegen" (Klassik, unverändert)
    nein:
      CoreAuditActive?
        ja → "elektronisch dokumentiert, keine weitere Aktion nötig"
        nein → "unterschreibe und sende zurück" (Klassik, unverändert)
```

Frontend: **kein Eingriff**.
Helm: **kein Eingriff**.
DB: **kein Eingriff**.
Migration: **keine**.

## Sicherheits-Bewertung

- Pure Mail-Wortlaut-Änderung
- Keine neuen API-Endpoints
- Keine Auth-Änderung
- Keine PII-Behandlung neu
- Defense-in-Depth bleibt: PDF-Audit-Block (PROJ-78) ist die Wahrheit;
  Mail-Text spiegelt jetzt die PDF-Realität (vorher war es inkonsistent)

→ `/security-review` nicht erforderlich.

## Geänderte Dateien

| Datei | Status |
|---|---|
| `internal/mail/service.go` | Modified — 2 neue Struct-Felder + buildMandateAtImportData |
| `internal/mail/templates/application_imported_member.html` | Modified — 4-Branch-Logik |
| `internal/mail/templates/application_imported_eeg.html` | Modified — 4-Branch-Logik |
| `internal/mail/service_test.go` | Modified — 4 neue Integration-Tests |
| `features/PROJ-88-mail-audit-trail-spiegel.md` | **New** — diese Spec |
| `features/INDEX.md` | Modified — Eintrag |
| `docs/user-guide/changelog.md` | Modified — PROJ-frei |
| `CHANGELOG.md` | Modified — Release-Notes-Block |

## QA Test Results

**Datum:** 2026-06-08
**Reviewer:** QA Engineer (AI)
**Status:** Approved

```
$ go test -v -run "TestSendMandateAtImport" ./internal/mail/...
PASS  TestSendMandateAtImport_CoreKlassik_AsksForSignature
PASS  TestSendMandateAtImport_CoreAudit_DoesNotAskForSignature
PASS  TestSendMandateAtImport_B2BKlassik_AsksForSignatureAndBank
PASS  TestSendMandateAtImport_B2BAudit_NoSignatureButKeepBankPreNotification
ok    internal/mail  0.144s

$ go test ./...
14 Pakete grün
```

**Production-Ready: READY.**

## Deployment

**Datum:** 2026-06-08
**Versions-Tag:** wird beim Push gesetzt (vermutlich v1.23.6-PROJ-88)
**Image-SHA:** wird vom CI nach Push gesetzt
**Status:** wartet auf `helm upgrade`

Owner führt `helm upgrade` manuell aus. Bündelt sich mit PROJ-86 +
PROJ-87 — ein Apply bringt alle drei live.

---
<!-- Sections below are added by subsequent skills -->
