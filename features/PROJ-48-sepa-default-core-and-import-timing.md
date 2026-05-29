# PROJ-48: SEPA-Default-Core + konfigurierbares Mandat-Timing + B2B-Hinweis

**Status:** Deployed
**Created:** 2026-05-17

## Hintergrund

Aktuell wird die SEPA-Mandat-Variante (Basislastschrift vs.
Firmenlastschrift) beim Submit automatisch nach Mitgliedstyp gewählt:
EEG-Setting `useCompanySEPAMandate=true` PLUS Mitgliedstyp ∈
{company, association} → Firmenlastschrift. Sonst Basislastschrift.

Das ist suboptimal:
- B2B-Lastschrift ist bürokratischer und ist nicht für jedes Unternehmen
  überhaupt nötig (viele EEGs kommen mit Core-Lastschrift auch für
  Firmen aus).
- Die Mandats-Variante sollte eine **EEG-Admin-Entscheidung** sein, kein
  Automatismus aus dem Mitgliedstyp.

Plus zweites Thema: das Timing der Mandat-Übermittlung. Heute kommt das
Basis-Mandat beim Submit (also ohne Mitgliedsnummer als Mandatsreferenz);
das B2B-Mandat (PROJ-47) wird erst beim Import generiert (mit
Mandatsreferenz). Manche EEGs würden auch für Core-Lastschrift gerne den
Pfad „Mandat erst beim Import mit Mandatsreferenz" wählen — z.B. weil
die Bank bei einigen EEGs auch für Core-Mandate eine ausgefüllte
Mandatsreferenz verlangt.

## Änderungen

### 1. Default Einzugsart immer `core`

`application.einzugsart` startet immer mit `core` (passt schon). Die
**Auto-Logik**, die im Mail-Versand auf Basis von `memberType` +
`useCompanySEPAMandate` die Firmenlastschrift-Variante wählt, **entfällt
ersatzlos**. Stattdessen entscheidet `app.einzugsart` allein:

- `einzugsart=core` → Basis-Mandat-PDF (oder kein PDF, je nach Timing-
  Setting, siehe Punkt 3)
- `einzugsart=b2b` → Firmenlastschrift-PDF
- `einzugsart=kein_sepa` → kein PDF

Der EEG-Admin kann im Antrags-Edit-Form die Einzugsart von `core` auf
`b2b` umstellen (das ist heute schon möglich).

### 2. B2B-Hinweis bei Unternehmen + Gemeinden

Wenn die Submit-Mail an einen Member mit Mitgliedstyp ∈ {`company`,
`municipality`} geht, enthält sie einen kurzen Zusatzsatz (bewusst
schlank gehalten — keine Begründungsdetails, keine Aufzählung der
betroffenen Mitgliedstypen):

> Falls statt der Basislastschrift eine Firmenlastschrift (SEPA B2B)
> notwendig ist, meldet sich {{EEG-Name}} mit den notwendigen
> Unterlagen bei Ihnen.

`association` (Verein) bleibt **bewusst außerhalb** dieses Hinweises —
auf User-Wunsch nur Unternehmen + Gemeinden. Bei Bedarf erweiterbar.

### 3. EEG-Setting: Mandat-Timing

> **Architektur-Hintergrund (digitale Signatur):** ein digital
> signiertes PDF darf nach der Signatur **nicht mehr modifiziert
> werden** — jede Änderung bricht den kryptographischen Hash und damit
> die Signatur. Wenn das Onboarding-System ein Mandat ausliefert, das
> der Member digital signieren soll UND das Mandat soll die
> Mitgliedsnummer als Mandatsreferenz enthalten, dann muss das
> Mandat **erst zum Import-Zeitpunkt** erzeugt werden (vorher ist die
> Mitgliedsnummer noch nicht vergeben). Das neue Setting deckt genau
> diesen Workflow ab. Vollständige Behandlung der digitalen Signatur:
> siehe `docs/open-questions.md` OQ-6.

Neues Boolean-Setting `sepa_mandate_at_import` auf `registration_entrypoint`
(Default `FALSE` = heutiges Verhalten):

- **`sepa_mandate_at_import = FALSE`** (heutiger Default):
  - Basis-Mandat-PDF wird **beim Submit** generiert und angehängt
    (ohne Mandatsreferenz — Platzhalter „wird von EEG ausgefüllt")
  - B2B-Mandat (sofern `einzugsart=b2b`): erst beim Import mit
    eingedruckter Mandatsreferenz (PROJ-47, unverändert)
- **`sepa_mandate_at_import = TRUE`** (neu):
  - **Kein** Basis-Mandat-PDF beim Submit
  - Beim Import wird das Mandat-PDF (Basis ODER B2B, je nach
    `einzugsart`) generiert und an die Import-Mail angehängt — mit
    **eingedruckter Mandatsreferenz = Mitgliedsnummer**

Damit kann die EEG entscheiden:
- „Wir wollen das Mandat möglichst früh in der Hand" (Standard, FALSE)
- „Wir wollen ein vollständig ausgefülltes Mandat inkl. Referenz —
  das passt erst beim Import" (TRUE)

### 4. Mandatsreferenz-Rendering im Basis-Mandat

`pdf.SEPAMandateData.MandateReference` existiert bereits (seit PROJ-47).
Beide Renderer (`Generate` für Basislastschrift, `GenerateCompany` für
Firmenlastschrift) drucken den Wert inline statt des Platzhalters,
wenn das Feld gesetzt ist. Keine Änderung am PDF-Generator nötig.

## Datenmodell

Migration `db/migrations/000042_sepa_mandate_at_import.up.sql`:

```sql
ALTER TABLE member_onboarding.registration_entrypoint
    ADD COLUMN sepa_mandate_at_import BOOLEAN NOT NULL DEFAULT FALSE;
```

## EEG-Settings-Endpoint

`PUT /api/admin/settings/eeg` nimmt zusätzlich `sepaMandateAtImport: boolean`
entgegen. `GET /api/admin/settings/eeg` liefert es entsprechend.

## Frontend

Im Admin-Settings-Editor (`/admin/settings` → EEG-Einstellungen) neue
Switch-Option „SEPA-Mandat erst beim Import senden (mit Mitgliedsnummer
als Mandatsreferenz)" — Default-State spiegelt den DB-Wert.

## Mail-Pfade nach PROJ-48

| Setting | Einzugsart | Submit-Mail Mandat-PDF | Import-Mail Mandat-PDF |
|---|---|---|---|
| `at_import=false` | `core` | Basis (ohne Mandatsref) | — |
| `at_import=false` | `b2b` | Firmenlastschrift (ohne Mandatsref) | Firmenlastschrift mit Mandatsref *(PROJ-47)* |
| `at_import=false` | `kein_sepa` | — | — |
| `at_import=true` | `core` | — | **Basis mit Mandatsref *(neu)*** |
| `at_import=true` | `b2b` | — | Firmenlastschrift mit Mandatsref |
| `at_import=true` | `kein_sepa` | — | — |

Der **B2B-Hinweisblock** (Punkt 2) erscheint nur in der **Submit-Mail**,
nicht in der Import-Mail. Beim Import ist die Entscheidung schon
gefallen und das passende Mandat liegt bei.

## Migration / Backward Compatibility

- Bestandsanträge bleiben unverändert.
- Bestehende EEGs haben `sepa_mandate_at_import=FALSE` → heutiger Pfad.
- Bestehende Anträge mit `einzugsart=b2b`, die durch die alte Auto-Logik
  bei Submit Firmenlastschrift bekommen hätten: hier ändert sich
  nichts, weil die Submit-Mail bereits versendet ist.
- Neue Anträge: Submit setzt `einzugsart=core` (passt schon). Nur wenn
  ein Admin gleich zu Beginn `einzugsart=b2b` setzt UND der Antrag in
  `needs_info → submitted` re-submitted wird, würde dort die
  Firmenlastschrift kommen. Edge-Case, akzeptabel.

## Out of Scope

- Digitale Signatur des Mandats (siehe OQ-6 in `docs/open-questions.md`
  als separater Diskussions-Thread)
- Automatisches Umstellen `einzugsart` durch ein Wizard / KI / o.ä.
- Hinweistext pro EEG konfigurierbar (V1: fester Wortlaut im Code)
- `association` (Verein) als B2B-Hinweis-Empfänger (User-Entscheidung:
  nur company + municipality)

## Tests

- Build muss grün bleiben
- Smoke-Test 1: EEG mit `sepa_mandate_at_import=FALSE` (Default),
  Submit → Basis-Mandat-PDF in Submit-Mail (Platzhalter-Ref)
- Smoke-Test 2: Submit von einem Company-Mitglied → Basis-Mandat-PDF +
  B2B-Hinweis-Block in der Mail
- Smoke-Test 3: EEG mit `sepa_mandate_at_import=TRUE`, Submit → KEIN
  Mandat-PDF in Submit-Mail (aber sonst alle bisherigen Mail-Inhalte)
- Smoke-Test 4: Import desselben Antrags → Basis-Mandat-PDF mit
  Mandatsreferenz = Mitgliedsnummer in Import-Mail
- Smoke-Test 5: Admin setzt `einzugsart=b2b`, Import → Firmenlastschrift
  mit Mandatsreferenz (unabhängig vom Setting, PROJ-47-Pfad)

## Follow-up Bugfix 2026-05-28: Mandate-Werte erreichen den Core

Tester-Befund: bei `einzugsart=b2b` und bei `einzugsart=core` +
`sepa_mandate_at_import=true` landeten `accountInfo.mandateReference` und
`accountInfo.mandateDate` leer im eegFaktura-Core. Lokal waren die Werte
korrekt (PDF + DB), aber das Onboarding leitete sie erst NACH dem
`POST /participant` ab und schickte sie nie nach.

Fix in `internal/importing/import_service.go`:

- Neuer Gate-Helper `shouldDeriveMandateAtImport(app, ep)` zentralisiert die
  Bedingung (`einzugsart=b2b` ODER `core+SEPAMandateAtImport`, jeweils nur
  bei `SepaMandateAccepted=true`).
- `Import()` ruft die Ableitung jetzt vor `BuildPayload`. Werte werden via
  `SetMandateReferenceIfEmpty` + neuer `SetMandateDateIfEmpty` persistiert
  (idempotent, Admin-Override bleibt) und in das in-memory `app`-Objekt
  gespiegelt, sodass `BuildPayload` sie ins Payload aufnimmt.
- `SendPostImportNotification` nutzt jetzt durchgängig die IfEmpty-Variante
  — der Import-Pfad und der Post-Import-Mail-Pfad konkurrieren nicht mehr
  um denselben Spaltenwert.

Regression-Guards: fünf `TestShouldDeriveMandateAtImport_*` in
`internal/importing/payload_test.go`.

Submit-Pfad (`core + sepa_mandate_at_import=false`, PROJ-12) bleibt
bewusst unverändert: dort kommuniziert die Aktivierungs-Mail die Referenz
als Hinweis und das Mitglied trägt sie auf dem Papier-Mandat ein; der
Admin überträgt sie später manuell, wenn das signierte Mandat zurückkommt.
