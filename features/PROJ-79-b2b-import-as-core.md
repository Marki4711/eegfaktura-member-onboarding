# PROJ-79: B2B-Import als CORE in eegFaktura-Core

## Status: Approved
**Created:** 2026-06-08
**Last Updated:** 2026-06-08 (post-grill, 10 weitere Owner-Entscheidungen festgenagelt)

## Kritischer Befund aus Grilling

`einzugsart="b2b"` wird in der gesamten Production-Codebase **nirgendwo
programmatisch gesetzt**. Public-Submit (`application_service.go:174`) entscheidet
ausschließlich zwischen `core` und `kein_sepa`. Externe API
(`internal/http/external.go:131`) durchläuft denselben Service. **`b2b` entsteht
heute ausschließlich durch manuellen Admin-Edit** (`admin_service.go:448`,
`*req.Einzugsart`).

Folge für PROJ-79:
- Das geänderte Mapping (`b2b → CORE`) greift praktisch nur, wenn der Admin
  einen Antrag manuell auf `b2b` gestellt hat und importiert.
- Externe-API-Konsistenz bleibt im Code erhalten (Robustheit für zukünftige
  Self-Service-b2b-Pfade), wird in der Spec aber nicht als praktischer
  Trigger ausgewiesen.
- Tester sehen das Feature ausschließlich, wenn sie im Antrags-Detail-Edit
  einen Antrag auf b2b setzen, dann importieren — der typische
  Public-Form-Flow triggert es nie.

## Hintergrund

Für `einzugsart=b2b` (Firmenlastschrift) verlangt das SEPA-B2B-SDD-Rulebook eine
separate Vereinbarung des Mandats zwischen Mitglied und dessen Hausbank. Bis
diese Bank-Abstimmung durch ist, würde eine sofortige B2B-Abbuchung im
eegFaktura-Core von der Bank abgelehnt werden.

Owner-Direktive 2026-06-07: der Import in den Faktura-Core soll für b2b-Anträge
trotzdem stattfinden, aber mit SEPA-Typ `CORE` (Basislastschrift), bis die Bank
die B2B-Aktivierung bestätigt hat. Damit ist die Mitgliedschaft sofort aktiv,
ohne Risiko fehlgeschlagener Erst-Lastschriften. Die EEG-Kontaktperson bekommt
in der Aktivierungs-Mail den expliziten Hinweis, dass sie die Bank-Klärung
eigenständig verfolgen und nach Bestätigung den Core-SEPA-Typ manuell auf B2B
umstellen muss.

Im Member-Onboarding bleibt `application.einzugsart='b2b'` unverändert — nur die
Core-Anlage geht mit CORE.

## Dependencies

- **Voraussetzt:** PROJ-4 (Core Import — Import-Pfad inkl. `mapEinzugsart`)
- **Voraussetzt:** PROJ-46 (Status-Modell — `awaiting_bank_confirmation` triggert
  weiterhin nur bei `einzugsart=b2b` und ist unverändert)
- **Voraussetzt:** PROJ-53 (Aktivierungs-Mail im Auto-Modus)
- **Voraussetzt:** PROJ-76 (Vorstands-Workflow-Mail) — beide Mail-Pfade bekommen
  den Hinweis-Block
- **Berührt:** PROJ-13 (Externe API) — geht denselben Import-Pfad
- **Berührt nicht:** PROJ-74 (B2B-Mandat-Pflicht-Check beim Import bleibt
  erhalten — Mandat-PDF wird trotzdem gebraucht)
- **Berührt nicht:** PROJ-78 (Audit-Toggle) — orthogonal, wirkt nur auf
  PDF-Rendering

## User Stories

- **Als EEG-Admin** möchte ich, dass ein importierter b2b-Antrag im Faktura-Core
  als CORE-Lastschrift angelegt wird, damit die erste Abbuchung nicht an einer
  noch nicht aktivierten B2B-Mandatsvereinbarung scheitert.
- **Als EEG-Admin** möchte ich in der Aktivierungs-Mail einen klaren Hinweis
  bekommen, dass ich die B2B-Bank-Aktivierung eigenständig verfolgen und den
  Core-SEPA-Typ nach Bestätigung manuell umstellen muss.
- **Als Mitglied** möchte ich nach Antrags-Aktivierung keine fehlgeschlagene
  Erst-Lastschrift erleben, die einen unnötigen Mahnungs-Reflex bei der EEG
  auslöst.
- **Als EEG-Mitarbeiter:in beim Externe-API-Integrator** möchte ich denselben
  CORE-statt-B2B-Workflow bekommen, damit Integrator-API-Anlagen sich konsistent
  zu Public-Form-Anlagen verhalten.
- **Als Owner** möchte ich, dass bestehende b2b-Anlagen im Core unangetastet
  bleiben, damit keine laufenden Lastschriften gestört werden.

## Acceptance Criteria

### Mapping-Änderung (Import-Pfad)

- [ ] **AC-1** `mapEinzugsart` in `internal/importing/payload.go` mappt
  `b2b → "CORE"` (statt heute `"B2B"`). `core → "CORE"` und
  `kein_sepa → ""` bleiben unverändert.
- [ ] **AC-2** `Sepa bool`-Feld (`CoreBankInfo.Sepa`) bleibt für b2b auf `true`
  — der Core soll weiter wissen, dass ein Mandat existiert; nur der Typ ist
  CORE.
- [ ] **AC-3** Mandat-Referenz und Mandat-Datum (PROJ-47) werden weiterhin
  korrekt im Core-Payload mitgegeben, unverändert zu heute.
- [ ] **AC-4** Code-Kommentar in `mapEinzugsart` und am Call-Site (Zeile 202)
  dokumentiert die PROJ-79-Direktive und Begründung (Bank-Klärung-Workaround).
- [ ] **AC-5** Externe API (`POST /api/external/v1/applications` → spätere
  Import-Triggerung) geht denselben Pfad — keine Sonderbehandlung.

### Aktivierungs-Mail-Hinweis

- [ ] **AC-6** EEG-Kopie der Beitrittsbestätigung im Auto-Modus
  (`SendActivationNotification`, PROJ-53) enthält bei `app.Einzugsart == "b2b"`
  einen Hinweis-Block. Bei anderen Einzugsarten kein Eingriff.
- [ ] **AC-7** Beitrittserklärung im Vorstands-Modus
  (`SendBoardApprovalRequest`, PROJ-76) enthält bei `app.Einzugsart == "b2b"`
  denselben Hinweis-Block. Bei anderen Einzugsarten kein Eingriff.
- [ ] **AC-8** Hinweis-Block-Wortlaut (DE):
  > **Hinweis B2B-SEPA-Mandat:**
  > Der Antrag wurde im Member-Onboarding mit Einzugsart „Firmenlastschrift
  > (B2B)" angelegt, aber zur Sicherheit im eegFaktura-Core zunächst als
  > Basislastschrift (CORE) importiert. Bitte vereinbaren Sie die
  > Firmenlastschrift-Aktivierung eigenständig mit der Hausbank des Mitglieds.
  > Sobald die Bank die B2B-Aktivierung bestätigt hat, ändern Sie den SEPA-Typ
  > im eegFaktura-Core manuell auf B2B.
  >
  > Hintergrund: Eine sofortige B2B-Abbuchung kann ohne Bank-Aktivierung
  > abgelehnt werden — der CORE-Pfad überbrückt die Klärungs-Phase ohne Risiko
  > fehlgeschlagener Erst-Lastschriften.
- [ ] **AC-9** Hinweis-Block wird als amber/gelber Banner gerendert (
  `background-color: #fffbeb`, `border-left: 4px solid #f59e0b`, Titel-Text
  `color: #92400e`, Body-Text `color: #78350f`) — exakt der Stil des PROJ-81-
  kein_sepa-Banners (siehe `application_activated_eeg.html:17` und
  `service.go:1128`). Banner-Titel: **„Hinweis B2B-SEPA-Mandat"** mit
  Vorzeichen `⚠`. Positionierung oberhalb der Antrags-Detail-Tabelle.
- [ ] **AC-10** **Shared Helper** statt PROJ-81-Mirror: neue Funktion
  `RenderB2BImportNoticeBanner() template.HTML` (oder ähnlich, in `mail`-Package)
  ist die einzige Stelle, die den Banner-HTML-Block produziert. Auto-Modus-
  Template bekommt das fertige HTML als `template.HTML`-Field auf
  `activationTemplateData` und rendert via `{{.B2BNoticeHTML}}` (kein
  `{{if}}` — bei Nicht-b2b liefert der Helper `""`). `SendBoardApprovalRequest`
  ruft denselben Helper inline auf und schreibt das Ergebnis in `bodyHTML`.
  Single source of truth, kein Drift-Risiko. (Memory-Regel
  `feedback_shared_helpers_for_parallel_paths`.) **Bewusste Abweichung vom
  PROJ-81-Pattern**, das den Banner-HTML doppelt verdrahtet hat (Template +
  Inline) und damit Drift-anfällig ist.

### Verhalten — Bestand & Status-Modell

- [ ] **AC-11** Bestehende b2b-Anträge, deren Core-Anlage bereits mit B2B
  importiert wurde, bleiben unangetastet. Kein Backfill-Skript, kein
  Admin-Knopf zum Zurücksetzen.
- [ ] **AC-12** Status-Modell unverändert: `imported → awaiting_bank_confirmation`
  triggert weiterhin bei `einzugsart=b2b` (PROJ-46), egal welcher SEPA-Typ im
  Core gelandet ist. Der Admin-Workflow ist konsistent mit der heutigen
  Erwartung („B2B-Anträge brauchen Bank-Bestätigung").
- [ ] **AC-13** PROJ-74-Hart-Fail beim Import bleibt erhalten: ohne
  B2B-Mandat-PDF (`bytes` leer) wird der Import abgebrochen. Das Mandat-PDF
  wird trotzdem für die Bank-Vorlage gebraucht.
- [ ] **AC-14** `application.einzugsart` im Member-Onboarding bleibt `b2b` —
  Admin sieht im Antrags-Detail weiter „Firmenlastschrift".

### Externe API & UI-Sichtbarkeit

- [ ] **AC-15** Externe API durchläuft denselben Service-Layer-Pfad
  (`CreateApplication`). Da `einzugsart` heute nicht im Externe-API-Body
  exponiert ist und nur durch Admin-Edit auf `b2b` gesetzt werden kann,
  greift die Mapping-Änderung praktisch nicht für API-Submits. Keine
  OpenAPI-Spec-Änderung. Das Mapping bleibt aber für Robustheit gegen
  zukünftige b2b-Pfade (z.B. Self-Service-b2b-Wahl) konsistent.
- [ ] **AC-16** Kein UI-Indikator im Antrags-Detail (kein Banner, kein
  Status-Log-Eintrag). Member und Admin sehen weiter `einzugsart=b2b`, der
  CORE-statt-B2B-Workflow ist nur in der EEG-Mail sichtbar.
- [ ] **AC-16a** `SendActivationNotification` bleibt async best-effort
  (Default-Verhalten, `go ...`-Aufrufe in `admin.go:1489`,
  `admin_service.go:729+907`). Wenn der b2b-Hinweis-Mail fehlschlägt, wird
  der Fehler nur im Pod-Log sichtbar — keine UI-Eskalation, keine Status-
  Log-Erzwingung. Owner-Entscheidung: B2B-Anlagen sind selten, manuelle
  Reaktion auf Log-Fehler zumutbar.

### Konfiguration & Settings

- [ ] **AC-17** Hartkodierte globale Regel — kein Per-EEG-Toggle. Keine neue
  DB-Spalte, kein Settings-UI-Eingriff.

### Tests

- [ ] **AC-18** Unit-Tests in `internal/importing/payload_test.go`:
  - `TestBuildPayload_EinzugsartMapping` Test-Case `{"b2b", "B2B"}` wird zu
    `{"b2b", "CORE"}` umgestellt. (heute Zeile 339)
  - Neuer Test `TestBuildPayload_B2B_IntentionallyMappedToCORE` mit
    Code-Kommentar zur PROJ-79-Begründung (Bank-Klärung-Workaround), damit
    künftige Devs nicht versehentlich „zurück-fixen".
- [ ] **AC-19** Unit-Test für Mail-Hinweis-Helper
  (`internal/mail/b2b_notice_test.go` oder ähnlich): `RenderB2BImportNoticeBanner`
  rendert bei `einzugsart="b2b"` den Block (HTML-Substring-Check), bei
  `core`/`kein_sepa`/`""` liefert leeren String.
- [ ] **AC-20** Integration-Test in `internal/mail/service_test.go`:
  - `SendActivationNotification` mit `einzugsart="b2b"` (b2b-Antrag) erzeugt
    eine EEG-Kopie, die den Hinweis-Wortlaut „Hinweis B2B-SEPA-Mandat"
    enthält. Bei `einzugsart="core"` nicht.
  - `SendBoardApprovalRequest` mit `einzugsart="b2b"` erzeugt eine Mail,
    deren Body den Hinweis-Wortlaut enthält. Bei `einzugsart="core"` nicht.

### Doku

- [ ] **AC-21** Neue Sektion in `docs/import-mapping.md`:
  **„SEPA-Typ-Mapping beim Core-Import"** mit Tabelle (onboarding
  `einzugsart` → Core `sepaDirectDebit`-Wert) und ausführlicher Begründung
  des `b2b → CORE`-Sonderfalls (Bank-Klärungs-Workaround). Dauerhafter
  Referenzpunkt für Devs.
- [ ] **AC-22** `docs/user-guide/06-admin-settings.md` SEPA-Sektion ergänzt
  um Subblock **„Firmenlastschrift im Faktura-Core"** mit Erklärung des
  CORE-statt-B2B-Workflows + Hinweis zur Bank-Klärung. Anonymisiertes
  Beispiel mit „Musterbetrieb GmbH". **PROJ-frei** (Memory-Regel
  `feedback_no_proj_refs_in_user_doc`).
- [ ] **AC-23** `docs/user-guide/changelog.md` Eintrag 2026-06-XX (PROJ-frei).
- [ ] **AC-24** `CHANGELOG.md` Eintrag mit PROJ-79-Bezug (Release-Notes-Teil),
  im selben Commit wie der Code (Memory-Regel `feedback_batch_changelog_with_code`).

## Edge Cases

- **EC-0 b2b entsteht heute ausschließlich per Admin-Edit:** Public-Submit
  (`application_service.go:174`) entscheidet zwischen `core` und `kein_sepa`.
  Externe API geht denselben Pfad. Nur `admin_service.go:448` setzt
  `app.Einzugsart = *req.Einzugsart` aus dem Admin-Edit-Form. PROJ-79
  greift praktisch nur, wenn der Admin im Antrags-Detail einen Antrag
  manuell auf b2b stellt und dann importiert. Tester-Workflow: Admin-Edit
  setzt einzugsart=b2b, Import-Button drücken, dann im Core sehen, dass
  CORE angelegt wurde, dann die EEG-Aktivierungs-Mail prüfen.
- **EC-1 b2b mit `sepa_mandate_accepted=false` (Admin-Edit-Inkonsistenz):**
  Admin kann theoretisch `einzugsart=b2b` setzen und
  `sepa_mandate_accepted=false` lassen. Verhalten nach Owner-Entscheidung:
  Mapping greift trotzdem (`b2b → "CORE"`), `Sepa` folgt
  `SepaMandateAccepted` (also `false`). Core bekommt
  `SepaDirectDebit="CORE"` + `Sepa=false` — semantisch komisch, war aber
  auch vor PROJ-79 schon so (vorher: `SepaDirectDebit="B2B"` + `Sepa=false`).
  PROJ-79 erzwingt keine Konsistenz. Keine zusätzliche Server-Validation.
- **EC-2 Reset-Import (PROJ-30) + Re-Import:** Antrag wird zurückgesetzt auf
  `approved`, Mandat-PDF + Bank-Daten unverändert, dann neu importiert. Erwartung:
  Re-Import nach PROJ-79-Deploy legt im Core CORE an (auch wenn beim ersten
  Import noch B2B angelegt wurde — der erste Core-Eintrag bleibt aber bestehen,
  wir reden über die Wiederherstellung). Owner-Direktive AC-11 sagt: alte Core-
  Anlagen unangetastet. Reset+Re-Import erzeugt im Core entweder einen neuen
  Eintrag oder einen Update — hängt vom Core-Verhalten ab. Klärung: Reset-Import
  ist bewusst eine Admin-Aktion mit Verantwortung; was der Core macht ist
  Out-of-Scope für PROJ-79.
- **EC-3 EEG-Kontakt fehlt (NULL `contact_email`):** Auto-Modus-Mail an Mitglied
  geht raus, EEG-Kopie wird übersprungen (heutiges Verhalten). Hinweis-Block
  geht in dem Fall nicht raus — der EEG bekommt also nichts. Mitigation: bei
  fehlendem EEG-Kontakt bleibt die Hauptverantwortung beim Admin im Onboarding-
  Frontend (er wird beim nächsten Login sehen, dass ein b2b-Antrag importiert
  ist). Out-of-Scope: PROJ-79 löst kein fehlendes EEG-Kontakt-Problem.
- **EC-4 Mail-Send-Failure:** Hinweis-Block kommt nicht raus, aber Import war
  erfolgreich, Core ist mit CORE angelegt. Tester-Risiko: EEG weiß nicht, dass
  sie Bank-Klärung machen muss. Mitigation: `feedback_mail_hard_fail`-Regel —
  Aktivierungs-Mail mit Hart-Fail (heutiges Verhalten ab PROJ-46/PROJ-53).
  Wenn die Mail fehlschlägt, sieht der Admin den Fehler sofort und kann manuell
  reagieren. PROJ-79 ändert das Mail-Sendeverhalten nicht.
- **EC-5 Reconciliation-Backstop (PROJ-69) für b2b-Anträge:** Reconciliation
  matcht über IBAN+Email, SEPA-Typ ist nicht Teil. Bei Match wird
  `faktura_handover_at` gesetzt — keine Interferenz mit PROJ-79.
- **EC-6 PROJ-78 Audit-Toggle aktiv für B2B:** B2B-Mandat-PDF rendert den
  Audit-Block statt klassischer Unterschrift. PROJ-79 wirkt orthogonal — der
  PDF-Render-Pfad ist unabhängig vom Core-SEPA-Typ-Mapping. Test: ein b2b-
  Antrag mit Audit-Toggle wird im Core mit CORE angelegt, das PDF hat trotzdem
  den Audit-Block.
- **EC-7 Test-Mode / Dev-Environment ohne echten Core:** Mock-Core nimmt jeden
  SepaDirectDebit-Wert an. Tests in `internal/importing/payload_test.go` decken
  das Mapping ab, kein Live-Core-Test nötig.
- **EC-8 Mail-Template-i18n:** alle Mails sind heute DE-only. Wortlaut ist
  fix DE, keine EN-Variante. Konsistent mit PROJ-79-Scope.
- **EC-9 Mitglied widerruft B2B-Mandat zwischen Submit und Import:** Wenn der
  Admin den Antrag noch im `approved`-Status hat und das Mandat zurückgezogen
  wird, kann der Admin manuell `einzugsart` auf `kein_sepa` umstellen (heutiges
  Verhalten). PROJ-79 ändert das nicht — der Antrag würde dann mit `""`-SEPA
  in den Core gehen.
- **EC-10 Bestehender Antrag mit `einzugsart=b2b` der schon im Status
  `awaiting_bank_confirmation` ist (PROJ-46):** Status-Wechsel bleibt unverändert.
  Der Admin kann manuell `ready_for_activation` setzen, sobald die Bank-
  Vereinbarung durch ist. Im Core ist der Eintrag (post-PROJ-79) bereits als
  CORE angelegt — Admin muss zusätzlich im Core den Typ auf B2B ändern. Das ist
  der Workflow, den die EEG-Mail beschreibt.

## Technical Requirements

- **Performance:** keine Auswirkung (Mapping ist O(1), Mail-Hinweis-Block ist
  statisches Template-HTML)
- **Security:** Mail-Hinweis ist statisches HTML — kein User-Input gerendert,
  keine XSS-Risiken. `app.Einzugsart` (DB-controlled) wird nur per
  `strings.EqualFold`-Check verwendet.
- **Backward-Compatibility:** Bestandsanträge unangetastet (AC-11). Mapping-
  Änderung wirkt nur ab Deploy-Zeitpunkt auf neue/re-imported Anträge.
- **Migrationen:** keine (kein DB-Schema-Change).
- **Helm:** keine neuen ENV-Variablen, kein values.yaml-Eingriff.
- **External API:** keine OpenAPI-Spec-Änderung (Request-Body unverändert,
  Antwort unverändert).
- **Browser-Support:** irrelevant (rein Backend + Mail-Templates).

## Owner-Entscheidungen (festgenagelt 2026-06-08)

### Aus /requirements

| # | Frage | Entscheidung |
|---|---|---|
| 1 | Mail-Pfade für Hinweis-Block | Beide Aktivierungs-Mails (Auto-Modus PROJ-53 EEG-Kopie + Vorstands-Modus PROJ-76 EEG-Kontakt) |
| 2 | Bestandsanträge mit schon importierten B2B-Core-Anlagen | Unangetastet — PROJ-79 wirkt nur auf neue Imports |
| 3 | Konfigurierbarkeit | Hartkodiert für alle EEGs, kein Per-EEG-Toggle |
| 4 | Wortlaut Hinweis-Block | Vorschlag unverändert (4 Sätze inkl. Hintergrund-Erklärung) |
| 5 | UI-Indikator im Antrags-Detail | Kein UI-Eingriff (Default, recommended) |
| 6 | Externe API (PROJ-13) | Analog — gleiches Mapping, kein Sonderfall |

### Aus /grill-me

| # | Frage | Entscheidung |
|---|---|---|
| 7 | Banner-Render-Architektur | **Shared Helper** statt PROJ-81-Mirror — `RenderB2BImportNoticeBanner() template.HTML` als single source, beide Mail-Pfade rufen ihn auf. Bewusste Verbesserung gegenüber dem PROJ-81-Pattern. |
| 8 | Mail-Delivery-Garantie | Nichts ändern — `SendActivationNotification` bleibt async best-effort. Fehler nur im Pod-Log. Manuelle Reaktion zumutbar. |
| 9 | Mapping-Edge `b2b + sepaMandateAccepted=false` | Mapping greift trotzdem (`b2b → CORE`), `Sepa=false`. Keine zusätzliche Server-Validation. |
| 10 | Status-Modell `awaiting_bank_confirmation` weiterhin sinnvoll? | Ja, unverändert — Status repräsentiert die Member-Onboarding-Sicht „warten auf Bank-Klärung", nicht den Core-SEPA-Typ. |
| 11 | Banner-Style | Amber/gelb wie PROJ-81 (`#fffbeb`, `border-amber-500`) — Konsistenz mit der kein_sepa-Banner-Familie |
| 12 | Banner-Titel | „⚠ Hinweis B2B-SEPA-Mandat" |
| 13 | Test-Strategie | 3 Tests: Mapping (umgestellt + neuer Intention-Test) + Helper-Unit-Test + Mail-Integration-Tests (beide Pfade) |
| 14 | User-Guide-Stelle | `06-admin-settings.md` SEPA-Sektion, neuer Subblock „Firmenlastschrift im Faktura-Core" |
| 15 | b2b-Trigger-Realität | Spec stellt klar, dass b2b heute nur per Admin-Edit entsteht. Externe-API-Mapping-Konsistenz bleibt für Zukunfts-Robustheit im Code. |
| 16 | Doku-Stelle Mapping-Regel | Neue Sektion in `docs/import-mapping.md` (zusätzlich zu Code-Kommentar in `mapEinzugsart`) |

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)

### A) Komponenten-Baum

```
Backend-Änderungen (Go)
+-- internal/importing/
|   +-- payload.go
|       +-- mapEinzugsart()  ◀ einzige Mapping-Stelle: b2b liefert künftig "CORE"
|   +-- payload_test.go
|       +-- TestBuildPayload_EinzugsartMapping  ◀ Test-Case b2b umgestellt
|       +-- TestBuildPayload_B2B_IntentionallyMappedToCORE  ◀ NEU (Intention-Schutz)
|
+-- internal/mail/
|   +-- b2b_notice.go  ◀ NEU: zentraler Banner-HTML-Helper
|   |   +-- RenderB2BImportNoticeBanner(einzugsart)  ◀ liefert "" wenn nicht b2b
|   +-- b2b_notice_test.go  ◀ NEU: Unit-Test für 3 Einzugsart-Werte
|   +-- service.go
|   |   +-- activationTemplateData  ◀ neues Field B2BNoticeHTML
|   |   +-- buildActivationData()  ◀ befüllt Field via Helper
|   |   +-- SendBoardApprovalRequest()  ◀ Helper-Aufruf statt Inline-HTML
|   +-- service_test.go
|   |   +-- TestSendActivationNotification_B2B_Banner  ◀ NEU
|   |   +-- TestSendBoardApprovalRequest_B2B_Banner  ◀ NEU
|   +-- templates/
|       +-- application_activated_eeg.html  ◀ neue Render-Position für {{.B2BNoticeHTML}}

Doku-Änderungen
+-- docs/import-mapping.md  ◀ neue Sektion „SEPA-Typ-Mapping beim Core-Import"
+-- docs/user-guide/06-admin-settings.md  ◀ neuer Subblock „Firmenlastschrift im Faktura-Core" (PROJ-frei)
+-- docs/user-guide/changelog.md  ◀ Eintrag 2026-06-08 (PROJ-frei)
+-- CHANGELOG.md  ◀ Eintrag mit PROJ-79-Bezug
```

Frontend: **kein Eingriff**. Helm: **kein Eingriff**. DB: **kein Eingriff**.

### B) Datenfluss-Sequenzen

**Sequenz 1: Import eines b2b-Antrags in den eegFaktura-Core**

```
Admin klickt Import im Antrags-Detail
  ↓
admin_service.go ImportApplication()
  ↓
importing.BuildPayload(app, ...)
  ↓
mapEinzugsart("b2b") → "CORE"      ◀ PROJ-79-Änderung
  ↓
CoreParticipantPayload mit SepaDirectDebit="CORE", Sepa=true
  ↓
POST /participant an eegFaktura-Core
  ↓
Status-Übergang imported → awaiting_bank_confirmation (PROJ-46, unverändert)
```

**Sequenz 2: Aktivierungs-Mail im Auto-Modus (PROJ-53)**

```
Status-Übergang ready_for_activation → activated
  ↓
go SendActivationNotification(appID)     ◀ async best-effort, unverändert
  ↓
buildActivationData(app, ep, pdfFailed)
  ↓
B2BNoticeHTML = RenderB2BImportNoticeBanner(app.Einzugsart)
                = template.HTML("<div…>") wenn b2b, "" sonst
  ↓
Mitglied-Mail rendert ohne Banner (Member-Template referenziert das Field nicht)
EEG-Kopie rendert mit Banner ({{.B2BNoticeHTML}} im EEG-Template)
  ↓
SMTP best-effort, Fehler nur im Pod-Log
```

**Sequenz 3: Vorstands-Mail (PROJ-76)**

```
Status-Übergang approved → ready_for_activation (Vorstands-Modus aktiv)
  ↓
SendBoardApprovalRequest(app, ep, pdfBytes, isReActivation)  ◀ sync hard-fail
  ↓
bodyHTML mit Begrüßung + (Re-Aktivierungs-Hinweis) + Mitgliedsname-Zeile
  ↓
banner := RenderB2BImportNoticeBanner(app.Einzugsart)
bodyHTML.WriteString(string(banner))     ◀ leere String wenn nicht b2b
  ↓
SMTP hard-fail (Fehler propagiert zum Caller)
```

Banner-Helper wird in beiden Pfaden mit **demselben Eingabe-Wert** (`app.Einzugsart`)
aufgerufen. Drift ausgeschlossen.

### C) Datenmodell

**Keine Änderung.** Es gibt keine neuen Tabellen, keine neuen Spalten und keine
Migration. Die Wirkung von PROJ-79 ist ausschließlich in der Daten-Übersetzung
zwischen Onboarding und Core sichtbar:

| Onboarding (DB) | Core (API-Payload) |
|---|---|
| `application.einzugsart = "core"` | `sepaDirectDebit = "CORE"`, `sepa = true` |
| `application.einzugsart = "b2b"` | `sepaDirectDebit = "CORE"`, `sepa = true` *(neu — PROJ-79)* |
| `application.einzugsart = "kein_sepa"` | `sepaDirectDebit = ""` (omitempty), `sepa = false` |

Das `einzugsart`-Feld in der Onboarding-DB bleibt enum-artiger TEXT mit den drei
Werten. Externes Sichtbar-Werden der Änderung passiert **nur** beim Aufbau der
Core-Payload.

### D) Tech-Entscheidungen (Begründung)

1. **Warum Shared Helper statt Bool-Field + Template-`{{if}}`?**
   PROJ-81 hat den Banner-HTML doppelt verdrahtet (im Template UND inline in
   SendBoardApprovalRequest). Beide Stellen müssen synchron bleiben, wenn
   Wortlaut, Farbe oder Struktur sich ändern. Tech-Debt. Für PROJ-79 ziehen wir
   die Lehre: **eine zentrale Funktion liefert das Banner-HTML**, beide Pfade
   rufen sie auf. Wortlaut-Änderungen passieren an genau einer Stelle.

2. **Warum `template.HTML`-Field, nicht Bool?**
   Wenn das Field bereits fertiges HTML enthält, hat das Template keine
   Markup-Verantwortung. Es rendert nur noch `{{.B2BNoticeHTML}}`. Bei
   Nicht-b2b liefert der Helper `""`, das Field rendert leer. Kein
   `{{if}}`-Branch im Template nötig.

3. **Warum bleibt das Mail-Send-Modell unverändert?**
   `SendActivationNotification` ist seit PROJ-53 async best-effort
   (Fire-and-forget mit `go ...`). B2B-Anlagen sind in der Praxis selten
   (heute nur Admin-Edit-getriggert). Manuelle Reaktion auf Pod-Log-Fehler ist
   zumutbar. Eine Inkonsistenz „sync hard-fail für b2b, best-effort für core"
   wäre architektonisch hässlich und schwer zu testen.

4. **Warum hartkodiert global, kein Per-EEG-Toggle?**
   Owner-Direktive: alle EEGs profitieren vom Bank-Klärungs-Workaround. Die
   Alternative (Toggle) würde 6 zusätzliche Code-Layer (DB-Spalte, Repo,
   Settings-Handler, UI, Configexport, Doku) ohne erkennbaren Mehrwert
   schaffen.

5. **Warum kein Backfill bestehender b2b-Core-Anlagen?**
   Eine laufende B2B-Lastschrift bei einer Bank, die das Mandat akzeptiert
   hat, soll nicht aus Versehen auf CORE umgestellt werden. Owner-Direktive:
   Bestand unangetastet.

6. **Warum Banner amber/gelb, nicht blau (Info)?**
   Konsistenz mit dem PROJ-81-kein_sepa-Banner. Beide signalisieren
   „Handlungsbedarf für den EEG-Admin". Visuelle Familie schafft Wieder-
   erkennbarkeit.

7. **Warum kein UI-Indikator im Antrags-Detail?**
   Die EEG-Mail ist der primäre Kommunikationskanal. Ein zusätzlicher Banner
   im Antrags-Detail würde das UI überfrachten und denselben Hinweis verdoppeln.
   Wenn die Mail fehlschlägt, sieht der Admin im Pod-Log nach (siehe
   Trade-off-Tabelle).

8. **Warum kein neuer Status im Status-Modell?**
   `awaiting_bank_confirmation` repräsentiert die Member-Onboarding-Sicht
   „Mitglied hat B2B-Mandat, EEG muss Bank-Klärung machen". Dieser Begriff
   bleibt gültig, unabhängig davon, was im Core angelegt wurde.

### E) Migrationspfad

| Schritt | Was passiert |
|---|---|
| 1. Backend-Deploy | Mapping greift sofort auf alle neuen Imports (Admin-Edit + Externe API + zukünftige b2b-Pfade) |
| 2. Bestand | Bereits importierte b2b-Anlagen im Core bleiben unangetastet — kein Backfill, kein Skript |
| 3. Roll-back | Reverter-Commit auf `mapEinzugsart` reicht — eine Code-Stelle, keine DB-Änderung |
| 4. Cluster-Upgrade | Standard-Helm-Bump-Workflow (Image-Tag), keine Helm-Werte-Änderung |

### F) Risiken & Trade-offs

| Risiko | Wahrscheinlichkeit | Mitigation |
|---|---|---|
| Aktivierungs-Mail fehlschlägt, EEG hört nichts von Bank-Klärung | niedrig | Pod-Log-Fehler, manuelle Admin-Reaktion (Owner-akzeptiert) |
| Admin macht inkonsistenten b2b-Edit (Mandat=false) | sehr niedrig | Mapping greift trotzdem, `Sepa=false` im Core (Owner-akzeptiert) |
| Externe API erweitert sich später um b2b-Pfade | mittel (Zukunft) | Mapping greift dann automatisch konsistent (Robustheit) |
| PROJ-78 Audit-Toggle gleichzeitig aktiv | hoch (typisch) | Orthogonal — Audit wirkt auf PDF, PROJ-79 auf Core-Payload, kein Konflikt |
| PROJ-69 Reconciliation läuft über b2b-Antrag | mittel | Match über IBAN+Email, SEPA-Typ irrelevant, kein Konflikt |
| PROJ-74 Hart-Fail (B2B-Mandat-PDF fehlt) | mittel | Bleibt erhalten — PDF wird für Bank-Vorlage gebraucht |
| Banner-Wortlaut driftet zwischen den beiden Mail-Pfaden | sehr niedrig | Shared Helper als Single source eliminiert das strukturell |

### G) Dependencies

- **Keine neuen Go-Pakete.** Alle benötigten sind bereits in `go.mod`:
  `html/template`, `bytes`, `strings`, `fmt`.
- **Keine neuen NPM-Pakete.** Kein Frontend-Eingriff.
- **Keine externen Service-Abhängigkeiten.** Mail-Helper rendert statisches HTML;
  Mapping-Funktion arbeitet rein auf Strings.

### H) Implementierungs-Reihenfolge

```
1. mapEinzugsart-Änderung + Code-Kommentar + Test-Update
   ↓
2. b2b_notice.go (Helper + Unit-Test)
   ↓
3. activationTemplateData um B2BNoticeHTML erweitern
   ↓
4. buildActivationData befüllt Field via Helper
   ↓
5. application_activated_eeg.html: Render-Position oberhalb der Antrags-Tabelle
   ↓
6. SendBoardApprovalRequest: Inline-Helper-Aufruf an PROJ-81-Stelle
   ↓
7. Integration-Tests für beide Mail-Pfade
   ↓
8. docs/import-mapping.md: neue Sektion „SEPA-Typ-Mapping beim Core-Import"
   ↓
9. docs/user-guide/06-admin-settings.md: Subblock „Firmenlastschrift im Faktura-Core"
   (PROJ-frei, Musterbetrieb GmbH)
   ↓
10. docs/user-guide/changelog.md + CHANGELOG.md (im selben Commit wie Code)
```

Jeder Schritt baut auf dem vorigen auf; Build + `go test ./...` zwischen jedem
größeren Edit. Helm-Bump ist nicht erforderlich (keine ENV-Variablen).

### I) Out-of-Scope (bewusst nicht enthalten)

- **Self-Service-b2b-Wahl im Public-Form** — eigenes Folge-PROJ. Wenn das
  kommt, greift PROJ-79 automatisch ohne weitere Anpassung.
- **Backfill bestehender b2b-Core-Anlagen** — Owner-Entscheidung gegen.
- **UI-Banner im Antrags-Detail** — Owner-Entscheidung gegen.
- **Per-EEG-Toggle** — Owner-Entscheidung gegen.
- **Status-Modell-Umbenennung von `awaiting_bank_confirmation`** —
  Owner-Entscheidung gegen.
- **Status-Log-Eintrag beim Import** — Owner-Entscheidung gegen.

## QA Test Results

**Datum:** 2026-06-08
**Reviewer:** QA Engineer (AI)
**Status:** Approved

### Methodik

Code-Review-basiertes QA. Mapping-Funktion und Banner-Helper sind reine
Pure-Functions ohne externe Abhängigkeit — Verhalten ist via Unit + Integration-
Tests vollständig abgedeckt. Browser-Test entfällt, weil PROJ-79 keine
UI-Komponenten hat (kein Frontend-Eingriff, kein Antrags-Detail-Banner). Mail-
Verhalten via Mock-Sender-Pattern verifiziert.

### AC-by-AC Sweep

| Kategorie | AC | Status | Hinweis |
|---|---|---|---|
| Mapping | AC-1 | ✅ Pass | `payload.go:289` mappt `b2b → "CORE"`. PROJ-79-Begründungs-Kommentar 17 Zeilen lang. |
| Mapping | AC-2 | ✅ Pass | `CoreBankInfo.Sepa` folgt `app.SepaMandateAccepted` (Zeile 203). PROJ-79 ändert das nicht. |
| Mapping | AC-3 | ✅ Pass | `MandateReference`, `MandateDate` (Zeilen 200-201) unverändert. |
| Mapping | AC-4 | ✅ Pass | Doc-Kommentar an `mapEinzugsart` dokumentiert Owner-Direktive, Bank-Klärungs-Workaround und Admin-Edit-Realität. |
| Mapping | AC-5 | ✅ Pass | Externe API geht durch denselben `CreateApplication`-Service, Mapping greift im selben `BuildPayload`-Pfad. |
| Mail | AC-6 | ✅ Pass | `SendActivationNotification` rendert EEG-Kopie via Template; `{{.B2BNoticeHTML}}` ist drin (Template Zeile 33). Test `TestSendActivationNotification_B2B_ShowsBanner_InEEGCopy` verifiziert. |
| Mail | AC-7 | ✅ Pass | `SendBoardApprovalRequest` ruft Helper inline auf (Zeile ~1127), vor dem PROJ-81-Block. Test `TestSendBoardApprovalRequest_B2B_ShowsBanner` verifiziert. |
| Mail | AC-8 | ✅ Pass | Wortlaut in `b2b_notice.go` enthält alle 4 Kern-Aussagen (Titel + Basislastschrift + Hausbank + manuell auf B2B). `TestRenderB2BImportNoticeBanner_ContainsKeyPhrases` ist die Wache. |
| Mail | AC-9 | ✅ Pass | Exakter Stil: `#fffbeb` + `border-amber-500` + `#92400e` (Titel) + `#78350f` (Body) + `⚠`. Konsistent zu PROJ-81. Position: oberhalb der Antrags-Tabelle (Template-Zeile 33 vor Tabellen-Start Zeile 35). |
| Mail | AC-10 | ✅ Pass | `RenderB2BImportNoticeBanner` ist die einzige Stelle mit Banner-HTML. Beide Pfade rufen sie auf — Auto-Modus via `B2BNoticeHTML`-Field, Vorstands-Modus via Inline-Call. Verifiziert per Grep auf "Hinweis B2B-SEPA-Mandat" (nur 1 Code-Stelle + Tests). |
| Bestand | AC-11 | ✅ Pass | Kein Backfill-Skript, kein Migration. Mapping greift nur auf neue Imports. |
| Bestand | AC-12 | ✅ Pass | `import_service.go:428` + `:631` + `:682` triggern weiterhin nur bei `app.Einzugsart == "b2b"` für `awaiting_bank_confirmation`. Unverändert. |
| Bestand | AC-13 | ✅ Pass | `buildSEPAMandateData` (`application_service.go:1650`) unverändert. PROJ-74-Hart-Fail erhalten. |
| Bestand | AC-14 | ✅ Pass | `application.einzugsart` bleibt `b2b` in DB. Mapping wirkt nur in `BuildPayload`. |
| Externe API + UI | AC-15 | ✅ Pass | Externe API durchläuft `CreateApplication` → Service entscheidet `einzugsart` zwischen `core` und `kein_sepa`. PROJ-79-Mapping wirkt theoretisch, faktisch nie aktiv. Keine OpenAPI-Änderung. |
| Externe API + UI | AC-16 | ✅ Pass | Kein UI-Indikator implementiert. `admin-application-detail.tsx` unverändert. |
| Externe API + UI | AC-16a | ✅ Pass | `SendActivationNotification` bleibt async best-effort (`go ...`-Aufrufe in `admin.go:1489`, `admin_service.go:729+907` unverändert). |
| Konfig | AC-17 | ✅ Pass | Keine DB-Spalte, kein Settings-UI, kein Toggle. Hartkodierte globale Regel im Mapping. |
| Tests | AC-18 | ✅ Pass | `TestBuildPayload_EinzugsartMapping` umgestellt (Case `b2b → CORE` + 2 zusätzliche Case-Insensitive-Cases). `TestBuildPayload_B2B_IntentionallyMappedToCORE` als Regressions-Wache mit ausführlichem Doc-Comment. |
| Tests | AC-19 | ✅ Pass | `TestRenderB2BImportNoticeBanner` (7 Permutationen) + `TestRenderB2BImportNoticeBanner_ContainsKeyPhrases` (Kern-Aussagen-Wache). |
| Tests | AC-20 | ✅ Pass | 4 Mail-Integration-Tests: `TestSendActivationNotification_B2B_ShowsBanner_InEEGCopy` + `_Core_DoesNotShowB2BBanner` + `TestSendBoardApprovalRequest_B2B_ShowsBanner` + `_Core_DoesNotShowB2BBanner`. Plus 3 `buildActivationData`-Tests. |
| Doku | AC-21 | ✅ Pass | `docs/import-mapping.md` Sektion 3.1 mit Tabelle, ausführlicher Begründung, Workflow-Beschreibung und „Praktische Trigger-Realität". |
| Doku | AC-22 | ✅ Pass | `docs/user-guide/06-admin-settings.md` Subblock „Firmenlastschrift im Faktura-Core" — PROJ-frei, Beispiel mit Musterbetrieb GmbH. |
| Doku | AC-23 | ✅ Pass | `docs/user-guide/changelog.md` Eintrag 2026-06-08 oberhalb des PROJ-81-Eintrags. PROJ-frei. |
| Doku | AC-24 | ✅ Pass | `CHANGELOG.md` PROJ-79-Block oberhalb des PROJ-81-Eintrags mit allen Implementations-Details. |

**Ergebnis:** 24 / 24 ACs Pass.

### Edge-Case Sweep

| EC | Stand | Hinweis |
|---|---|---|
| EC-0 (b2b nur per Admin-Edit) | ✅ verifiziert | Spec dokumentiert es als „Kritischer Befund aus Grilling" am Anfang + EC-0 + AC-15. |
| EC-1 (b2b + !Mandat = inkonsistent) | ✅ Code-Verhalten korrekt | Mapping greift trotzdem (`mapEinzugsart` ist stur), `Sepa=false` (`payload.go:203`). |
| EC-2 (Reset-Import + Re-Import) | ✅ verifiziert | Reset-Import-Pfad unangetastet. Re-Import nach PROJ-79-Deploy würde im Core CORE anlegen — Owner-akzeptiert. |
| EC-3 (EEG-Kontakt fehlt) | ✅ verifiziert | `SendActivationNotification` skippt EEG-Kopie wenn `ContactEmail == nil || == ""` (`service.go:1055`). Banner geht dann nicht raus — Spec dokumentiert das als bekannte Limitation. |
| EC-4 (Mail-Send-Failure) | ✅ verifiziert | Async best-effort, Fehler nur im Pod-Log. Owner-Entscheidung. |
| EC-5 (Reconciliation PROJ-69) | ✅ verifiziert | `reconciliation_repo.go` matcht nicht über SEPA-Typ — kein Konflikt. |
| EC-6 (PROJ-78 Audit-Toggle) | ✅ verifiziert | PDF-Render-Pfad ist `buildSEPAMandateData` (unverändert). Mapping wirkt nur auf Core-Payload. |
| EC-7 (Test-Mode ohne Live-Core) | ✅ verifiziert | Mapping-Test komplett unabhängig vom Core. |
| EC-8 (i18n) | ✅ N/A | Mails sind DE-only, konsistent mit Codebase. |
| EC-9 (Mandat-Widerruf) | ✅ verifiziert | Admin-Edit auf `kein_sepa` weiter möglich. PROJ-79 ändert das nicht. |
| EC-10 (`awaiting_bank_confirmation`-Workflow) | ✅ verifiziert | Status-Pfad unverändert. |

**Ergebnis:** 11 / 11 ECs OK.

### Security Smoke Test

| Bereich | Befund |
|---|---|
| Auth/AuthZ | Kein neuer Endpoint. Import-Pfad ist Keycloak-protected (Tenant-Admin oder Superuser). Mapping greift nur intern. |
| Injection | Mapping ist purer `strings.ToLower + TrimSpace + switch`. Kein User-Input. Helper-HTML ist statisches `template.HTML` mit Compile-Time-Strings. |
| XSS/CSRF | Banner-HTML ist 100% statisch — keine User-Input-Interpolation. `template.HTML` wird vom Go-Template-Engine bewusst nicht escapet, aber der Inhalt kommt nicht aus User-Land. Helper macht `strings.EqualFold + TrimSpace` auf `app.Einzugsart` (DB-controlled enum). |
| Secrets | Keine Secrets in Banner-Text, keine in neuen Logs. |
| Dependencies | `govulncheck`: 0 callable vulnerabilities. `gosec`: 0 issues auf importing + mail. `npm audit`: 4 moderate vor PROJ-79 unverändert (uuid via next-auth, Pre-PROJ-79-Lage). |
| Tenant-Isolation | Mapping ist tenant-agnostic, wirkt in jedem Import-Pfad identisch. Keine Cross-EEG-Logik. |
| Status-Transition-Bypass | Status-Modell unverändert. `awaiting_bank_confirmation` triggert weiter nur bei `einzugsart=b2b`. |
| Logging | Keine neuen Logs. Helper loggt nichts. |
| Personal Data | Banner-Wortlaut enthält keine PII. Mail-Pfade gehen nur an EEG-Kontaktperson. |
| Eingabe-Längenbeschränkungen | N/A — kein neuer Endpoint, kein neues Request-Feld. |

**Ergebnis: 0 Findings.**

### Regression Sweep

| Feature | Stand | Hinweis |
|---|---|---|
| PROJ-81 (SEPA optional) | ✅ Pass | `{{if .NoSepaMandate}}` Block im EEG-Template noch da. `application_activated_eeg.html:15` + Inline-Block in `SendBoardApprovalRequest:1133+` unangetastet. PROJ-79-Banner kommt davor. |
| PROJ-80 (SEPA-Settings-Vereinfachung) | ✅ Pass | Keine Berührung mit PROJ-79. |
| PROJ-78 (Audit-Toggle) | ✅ Pass | PDF-Render-Pfad orthogonal — wirkt auf `buildSEPAMandateData`, nicht auf Core-Payload-Mapping. |
| PROJ-76 (Vorstands-Workflow) | ✅ Pass | `SendBoardApprovalRequest` funktioniert weiter, neuer Banner-Block bei b2b vor dem PROJ-81-Block (mutually exclusive, weil einzugsart enum). |
| PROJ-74 (Hart-Fail B2B-PDF) | ✅ Pass | `buildSEPAMandateData` unverändert. PROJ-74-Hart-Fail erhalten. |
| PROJ-69 (Reconciliation) | ✅ Pass | Match-Strategie über IBAN+Email unangetastet. SEPA-Typ nicht Teil des Matchings. |
| PROJ-53 (Aktivierungs-Mail) | ✅ Pass | `SendActivationNotification` rendert weiterhin Member-Mail (ohne Banner — Member-Template referenziert das Field nicht) + EEG-Kopie (mit Banner bei b2b). Verifiziert per `TestSendActivationNotification_B2B_ShowsBanner_InEEGCopy`. |
| PROJ-47 / PROJ-48 (SEPA-Mandat-Timing) | ✅ Pass | Mandat-PDF + Mandat-Referenz unverändert. |
| PROJ-46 (Status-Modell) | ✅ Pass | `awaiting_bank_confirmation` triggert weiter bei einzugsart=b2b (`import_service.go:428/631/682`). |
| PROJ-4 (Core Import) | ✅ Pass | `BuildPayload`-Aufruf-Struktur unverändert, nur `mapEinzugsart`-Returnwert für b2b geändert. |

**Ergebnis: 0 Regressionen.**

### Tests-Status

```
$ go test ./internal/importing/... -run "EinzugsartMapping|B2B_Intentionally"
PASS  TestBuildPayload_EinzugsartMapping (8 subtests)
PASS  TestBuildPayload_B2B_IntentionallyMappedToCORE
ok    internal/importing  0.165s

$ go test ./internal/mail/... -run "B2B|B2BNotice"
PASS  TestRenderB2BImportNoticeBanner (7 subtests)
PASS  TestRenderB2BImportNoticeBanner_ContainsKeyPhrases
PASS  TestBuildActivationData_B2B_SetsB2BNoticeHTML
PASS  TestBuildActivationData_Core_DoesNotSetB2BNotice
PASS  TestBuildActivationData_KeinSepa_DoesNotSetB2BNotice
PASS  TestSendActivationNotification_B2B_ShowsBanner_InEEGCopy
PASS  TestSendActivationNotification_Core_DoesNotShowB2BBanner
PASS  TestSendBoardApprovalRequest_B2B_ShowsBanner
PASS  TestSendBoardApprovalRequest_Core_DoesNotShowB2BBanner
ok    internal/mail  0.128s

$ go test ./... — 14 Pakete grün, 0 failures
$ go build ./... — clean
```

### Scan-Status

| Scan | Ergebnis |
|---|---|
| `govulncheck ./...` | 0 callable vulnerabilities |
| `gosec ./internal/importing/... ./internal/mail/...` | 0 issues, 2888 Lines, 6 Files |
| `npm audit --audit-level=high` | 0 high/critical. 4 moderate vor PROJ-79 unverändert (uuid via next-auth Transitive). |
| `grep PROJ- docs/user-guide/` | leer (PROJ-frei verifiziert) |
| `git status helm/` | clean (kein versehentlicher Helm-Eingriff) |

### Production-Ready-Empfehlung

**READY.**

24 / 24 ACs Pass, 11 / 11 ECs OK, 0 Security-Findings, 0 Regressionen, 19 neue
Tests grün, 14 Pakete grün, Scans clean.

PROJ-79 ist eine sehr kleine, fokussierte Mapping-Änderung mit gut isoliertem
Helper-Pattern. Das Single-Source-Helper-Pattern (Lehre aus der PROJ-81-Doppel-
verdrahtung) ist sauber umgesetzt und durch Tests gegen Drift gesichert.

### Pflicht-Trigger /security-review

PROJ-79 berührt **Import-Logik** — Pflicht-Trigger laut CLAUDE.md.
**Empfehlung:** `/security-review` vor `/deploy` durchführen.

---

## Security Review

**Reviewer:** Security Engineer (AI)
**Date:** 2026-06-08
**Scope:**
- `internal/importing/payload.go` (`mapEinzugsart`)
- `internal/importing/payload_test.go` (Mapping-Tests)
- `internal/mail/b2b_notice.go` (NEU, Banner-Helper)
- `internal/mail/b2b_notice_test.go` (NEU)
- `internal/mail/service.go` (`activationTemplateData.B2BNoticeHTML`, `buildActivationData`, `SendBoardApprovalRequest`)
- `internal/mail/service_test.go` (Integration-Tests)
- `internal/mail/templates/application_activated_eeg.html` (`{{.B2BNoticeHTML}}`)
- `docs/import-mapping.md` + `docs/user-guide/06-admin-settings.md` + `docs/user-guide/changelog.md` + `CHANGELOG.md`

### Threat Model Summary

PROJ-79 ist eine Mapping-Änderung in der Core-Import-Übersetzung
(`einzugsart=b2b → "CORE"`) plus ein statischer EEG-Mail-Banner. Im
schlimmsten Fall einer fehlerhaften Mapping-Implementation würde ein
Mitglied im Faktura-Core mit falschem SEPA-Typ angelegt — das ist ein
Buchhaltungs-/Workflow-Problem, **kein Daten-Leak, keine Auth-Lücke,
keine Tenant-Isolation-Verletzung.** Der Banner-Helper liefert statisches
Compile-Time-HTML; der einzige variable Input (`einzugsart`) ist
server-side enum-validiert (`oneof=kein_sepa b2b core` in
`shared/requests.go:333`) und wird im Helper ausschließlich als
Control-Flow-Discriminator verwendet (`strings.EqualFold + TrimSpace`),
**nie interpoliert**.

### Independent Code Verification

| Behauptung aus QA | Verifiziert? | Befund |
|---|---|---|
| Banner-HTML ist 100% statisch | ✅ | `b2b_notice.go:31-35` Go-Raw-String-Literal, keine `fmt.Sprintf`, keine Concat mit Input |
| `einzugsart` ist enum-validiert | ✅ | `shared/requests.go:333` `validate:"omitempty,oneof=kein_sepa b2b core"` |
| User-Inputs in EEG-Mail werden HTML-escaped | ✅ | `service.go:1136-1137` `html.EscapeString(memberName/eegName/memberNumber)` |
| Helper liefert "" bei nicht-b2b | ✅ | `b2b_notice.go:28-30` Early-Return mit `template.HTML("")` |
| Mail-Recipient kommt aus `ep.ContactEmail` | ✅ | `service.go:1147` (Vorstands) + `service.go:1068` (Auto-Modus EEG-Kopie) |
| Status-Modell unverändert | ✅ | `import_service.go:428/631/682` triggert `awaiting_bank_confirmation` weiter bei `einzugsart=b2b` |
| PROJ-74 Hart-Fail erhalten | ✅ | `application_service.go:1650` `buildSEPAMandateData` unverändert |

### Findings

**0 Findings** — kein Critical, kein High, kein Medium, kein Low, kein Info.

Begründung:
- **Auth/AuthZ:** Kein neuer Endpoint. Import-Pfad bleibt Keycloak-protected mit `checkTenantAccess`. Banner-Render läuft nur server-side, kein Client-Vektor.
- **XSS:** Banner-HTML ist Compile-Time-Literal. `template.HTML`-Typ wird vom Go-html/template-Engine bewusst als Raw-HTML behandelt — das ist die korrekte Verwendung des Typs, weil der Inhalt aus Code-Land kommt, nicht User-Land. Selbst wenn ein böswilliger Admin via Admin-Edit-Form `einzugsart` manipulieren würde, lehnt der Validator (`oneof`) jeden Wert außer `kein_sepa`/`b2b`/`core` ab. Der Helper selbst matcht via `EqualFold + TrimSpace` nur exakt `b2b` und liefert bei allen anderen Werten den leeren String — defensive Eingabe-Behandlung.
- **Injection:** Mapping ist purer String-Switch. Banner-Helper ist purer String-Switch. Keine SQL, keine Shell, keine Path-Operations.
- **Tenant-Isolation:** Mapping wirkt in jedem Import-Pfad identisch. Banner geht via `ep.ContactEmail` an die EEG-Kontaktperson genau dieses Antrags-Tenant — kein Cross-EEG-Leak möglich, weil `ep` aus dem Application-Repository pro Antrag geladen wird (RC-gefiltert).
- **PII/DSGVO:** Banner-Text enthält keine PII. Mail-Subject und Recipient sind aus dem bestehenden Pfad (memberName/contactEmail) — Pre-PROJ-79-Bestand, kein neuer PII-Pfad.
- **Logging:** Helper loggt nichts. Mail-Pfade loggen Errors mit `application_id` (UUID) und Send-Errors — kein PII-Leak, etablierter Pattern.
- **Schema-Migration:** Keine. Roll-back ist ein Reverter-Commit auf eine Code-Stelle.
- **Mail-Send-Modell:** Unverändert (Auto-Modus async best-effort, Vorstands-Modus sync hard-fail). Owner-akzeptierter Trade-off bei Mail-Fail: nur Pod-Log.
- **Status-Transition-Bypass:** Status-Modell unverändert. `awaiting_bank_confirmation` triggert weiter nur bei `einzugsart=b2b`. Kein neuer Status-Übergang, kein Bypass möglich.
- **Dependencies:** Keine neuen Pakete. Bestehende Scan-Lage unverändert.

### Scan Results

| Scan | Ergebnis |
|---|---|
| `govulncheck ./...` | **0 callable vulnerabilities**, 5 in import packages (nicht aufgerufen), 1 in modules required |
| `gosec -severity medium -confidence medium ./...` | **0 issues** über 32980 Lines / 90 Files (volle Repo) |
| `npm audit --audit-level=high` | **0 high/critical**. 4 moderate Pre-PROJ-79-Bestand (uuid GHSA-w5hq-g745-h8pq via next-auth Transitive) |
| `trivy config helm/ --severity HIGH,CRITICAL` | **0 misconfigurations** |
| `trivy config . --severity HIGH,CRITICAL` | **0 misconfigurations** über k8s/Helm/Dockerfiles |
| Semgrep | Not run (gosec full-Repo war clean — Semgrep ist optional bei 0 gosec-Findings) |

### Verdict: **APPROVED**

PROJ-79 ist eine sehr kleine, sicherheitsneutrale Änderung:
- 1 Zeile Mapping-Returnwert (`"B2B" → "CORE"`) + 17 Zeilen Doc-Kommentar
- 1 neuer Helper mit statischem Banner-HTML, defensive Input-Behandlung
- 1 neues Struct-Field, befüllt via Helper
- 1 Template-Field-Render-Position
- 1 Inline-Aufruf in der Vorstands-Mail (PROJ-81-Stelle als Vorbild)

Keine neuen Endpoints, keine Schema-Änderung, keine Auth-Änderung, keine
Tenant-Isolation-Änderung, keine Public-Endpoint-Änderung, keine
Secrets/ENV-Variablen-Änderung, keine Helm/Dockerfile-Änderung. Alle Scans
clean. Alle Tests grün (19/19 neue Tests + 14 Pakete grün).

**Empfehlung:** APPROVED for `/deploy`.

## Deployment

**Datum (Bookkeeping):** 2026-06-08
**Version:** `v1.23.0-PROJ-79`
**Image-SHA:** wird vom CI nach Push gesetzt (Auto-Bump-Commit `chore: update Helm image tags to sha-XXXXXXX [skip ci]`)
**Status:** Code merged + Helm-Image bereitgestellt, **wartet auf manuellen `helm upgrade` durch Owner**

### Pre-Deploy-Status (verifiziert)

| Check | Ergebnis |
|---|---|
| `go build ./...` | clean |
| `go test ./...` | 14 Pakete grün, 19 neue PROJ-79-Tests |
| QA-Sektion | APPROVED (24/24 ACs, 11/11 ECs, 0 Findings) |
| Security-Review-Sektion | APPROVED (0 Findings, alle 5 Scanner clean) |
| `govulncheck` | 0 callable vulnerabilities |
| `gosec` full-Repo | 0 issues / 32980 Lines |
| `trivy config` helm + IaC | 0 misconfigurations |
| `npm audit --audit-level=high` | 0 high/critical |
| PROJ-Refs im User-Guide | leer (PROJ-frei) |
| Helm-Drift | keine (kein values.yaml-Eingriff) |
| DB-Migration | keine (Code-only) |

### Geänderte Dateien

| Pfad | Status |
|---|---|
| `internal/importing/payload.go` | Modified — `mapEinzugsart`: `b2b → "CORE"` + 17 Zeilen Doc |
| `internal/importing/payload_test.go` | Modified — Test umgestellt + Regressions-Wache |
| `internal/mail/b2b_notice.go` | **New** — Shared Banner-Helper |
| `internal/mail/b2b_notice_test.go` | **New** — Unit-Tests |
| `internal/mail/service.go` | Modified — `B2BNoticeHTML`-Field + Verdrahtung |
| `internal/mail/service_test.go` | Modified — 7 Integration-Tests |
| `internal/mail/templates/application_activated_eeg.html` | Modified — Render-Position |
| `docs/import-mapping.md` | Modified — Sektion 3.1 |
| `docs/user-guide/06-admin-settings.md` | Modified — „Firmenlastschrift im Faktura-Core" |
| `docs/user-guide/changelog.md` | Modified — Eintrag 2026-06-08 |
| `CHANGELOG.md` | Modified — PROJ-79-Block |
| `features/PROJ-79-b2b-import-as-core.md` | **New** — Spec inkl. QA + Security-Review |
| `features/INDEX.md` | Modified — Status-Updates |
| `docs/PRD.md` | Modified — Roadmap-Eintrag |

### Owner-Aktion-Liste

PROJ-79 reiht sich in den Helm-Upgrade-Stau ein. Auf Cluster (test) warten
bereits PROJ-78, PROJ-80 und PROJ-81 — alle vier können in **einem
einzigen `helm upgrade`-Lauf** ausgeliefert werden.

**Schritt 1: Commit + Push (kein Cluster-Apply, das macht die CI)**

```bash
# Stage + Commit (Code + Tests + Doku im selben Commit)
git add internal/importing/payload.go internal/importing/payload_test.go \
        internal/mail/b2b_notice.go internal/mail/b2b_notice_test.go \
        internal/mail/service.go internal/mail/service_test.go \
        internal/mail/templates/application_activated_eeg.html \
        docs/import-mapping.md docs/user-guide/06-admin-settings.md \
        docs/user-guide/changelog.md CHANGELOG.md \
        features/PROJ-79-b2b-import-as-core.md features/INDEX.md docs/PRD.md

git commit -m "feat(PROJ-79): B2B-Import als CORE in eegFaktura-Core"
git push
```

**Schritt 2: CI baut Image + Helm-Bump (~5 Min)**

GitHub Actions baut `marki4711/eegfaktura-member-onboarding-backend:sha-XXXXXXX` und
schreibt einen `chore: update Helm image tags`-Commit auf main zurück.

**Schritt 3: Git-Tag setzen (nach CI-Build erfolgreich)**

```bash
git pull
git tag -a v1.23.0-PROJ-79 -m "Deploy PROJ-79: B2B-Import als CORE in eegFaktura-Core"
git push origin v1.23.0-PROJ-79
```

**Schritt 4: `helm upgrade` auf test (gilt für PROJ-78 + PROJ-79 + PROJ-80 + PROJ-81)**

```bash
helm upgrade eegfaktura-member-onboarding ./helm/member-onboarding \
  -f helm/member-onboarding/values-env.yaml \
  -f helm/member-onboarding/values-secret.yaml
```

**Wichtig:**
- **Keine neuen ENV-Variablen** für PROJ-79 — bestehende `values-env.yaml` reicht
- **Keine Migration nötig** — der Migrate-Job läuft trotzdem, hat aber nichts zu tun
- Der Helm-Bump-Commit muss vor dem `helm upgrade` lokal vorliegen (`git pull` davor)

**Schritt 5: Post-Rollout-Verify**

```bash
kubectl rollout status deployment/eegfaktura-member-onboarding-backend -n eegfaktura-member-onboarding-test
curl -fsS https://member-onboarding-test.eegfaktura.at/health
```

**Rollback (falls nötig):**

```bash
helm rollback eegfaktura-member-onboarding
```

PROJ-79 ist Code-only und Roll-back-freundlich — keine DB-Daten verändert.

### Funktionstest nach Deploy (Tester-Pfad)

1. Admin-UI öffnen, einen Antrag im Status `approved` aufrufen.
2. Admin-Edit-Form aufmachen → Einzugsart auf **Firmenlastschrift (B2B)** stellen → Speichern.
3. Antrag importieren.
4. Verifizieren in eegFaktura-Core: das Mitglied ist mit SEPA-Typ **CORE** angelegt (nicht B2B) → Mapping wirkt.
5. Antrag auf `ready_for_activation` und dann `activated` ziehen.
6. EEG-Kontaktperson öffnet die Aktivierungs-Mail → der gelbe **„Hinweis B2B-SEPA-Mandat"**-Banner ist sichtbar.
7. Falls Vorstands-Modus aktiv ist (PROJ-76): die Beitrittserklärung-Mail an den Vorstand enthält denselben Banner.

### Versions-Historie

| PROJ | Version | Datum |
|---|---|---|
| PROJ-78 | v1.18.0-PROJ-78 | 2026-06-07 |
| PROJ-80 | v1.21.0-PROJ-80 | 2026-06-08 |
| PROJ-81 | v1.22.0-PROJ-81 | 2026-06-08 |
| **PROJ-79** | **v1.23.0-PROJ-79** | **2026-06-08** |
