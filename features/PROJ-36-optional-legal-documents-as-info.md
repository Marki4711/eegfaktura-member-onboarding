# PROJ-36: Optionale Rechtsdokumente als Info-Dokumente

## Status: In Review
**Created:** 2026-05-14
**Last Updated:** 2026-05-15 (Stages 1–6 implementiert: Migration 34, Backend-Submit-Pfad mit auto-informational, Public-Form zweispaltig, Admin-Settings-Labels, Admin-Detail separate Anzeige, PDF separate Blöcke, Doku.)

## Dependencies
- Berührt: PROJ-9 (EEG-spezifische Rechtsdokumente)
- Berührt: PROJ-18 (Datenschutzerklärung & Central Policy Toggle)
- Berührt: PROJ-1 (Public Registration — Consent-Anzeige im Formular)

## Hintergrund

Aktuell gibt es bei den EEG-Rechtsdokumenten zwei Modi: **„Zustimmung erforderlich"** (Pflicht-Häkchen, Submit-Blocker bis angehakt) und **„nicht erforderlich"** (Häkchen sichtbar, kann aber unangehakt eingereicht werden).

Feedback aus dem Beta-Test 2026-05-14:
> „Es macht wenig Sinn, bei optionalen Rechtsdokumenten eine Möglichkeit zum Anhaken anzubieten. Wenn die Bestätigung nicht verpflichtend ist, ist das Dokument eigentlich nur zur Info und müsste entsprechend auch anders dargestellt und protokolliert werden."

Die aktuelle UX vermittelt dem Mitglied, dass der Haken bedeutsam sei — er wird aber rechtlich nicht zwingend verlangt. Das ist verwirrend, lässt im Worst Case Mitglieder denken sie hätten etwas „abgewählt", was sie eigentlich nicht ablehnen konnten.

## User Stories

- Als **EEG-Admin** möchte ich bei einem Rechtsdokument entscheiden können, ob es **rein informativ** (Link wird angezeigt, kein Häkchen) oder **zustimmungspflichtig** (Häkchen, Submit-Blocker) ist — nichts dazwischen.
- Als **Mitglied** möchte ich klar sehen, welche Dokumente ich aktiv akzeptiere und welche ich nur zur Kenntnis nehme.
- Als **vfeeg-Betreiber** möchte ich im Audit-Log unterscheiden können zwischen „Mitglied hat zugestimmt" (mit Zeitstempel) und „Mitglied wurde informiert" (kein expliziter Klick nötig).

## Vorgeschlagene Änderungen

### Datenmodell

`legal_document.requires_consent` wird klar zweipolig: TRUE oder FALSE.
- TRUE: bisheriges Verhalten — Pflicht-Häkchen, Submit-Blocker.
- FALSE: **neues Verhalten** — kein Häkchen im Formular, nur Link-Anzeige. **Nicht** mehr „optional anhakbar".

`document_consent` (Audit-Tabelle): bei `requires_consent=FALSE`-Dokumenten wird trotzdem ein Eintrag geschrieben — aber mit Marker, dass es eine **Info-Kenntnisnahme** war, kein aktiver Klick. Vorschlag: neue Spalte `consent_type TEXT NOT NULL DEFAULT 'explicit'` mit Werten `'explicit' | 'informational'`.

### Public-Registration-Frontend
Im Formular: zwei separate Sektionen:
- **„Erforderliche Zustimmungen"** — Pflicht-Häkchen, wie bisher.
- **„Zur Information"** — Liste mit Link + Hinweistext „Die folgenden Dokumente werden Ihnen zur Information bereitgestellt. Mit Absenden des Antrags bestätigen Sie, diese zur Kenntnis genommen zu haben."

### Admin-UI
Im Settings → Rechtsdokumente: das aktuelle „Zustimmung erforderlich"-Toggle bleibt, **aber:**
- Label klären: „Mitglied muss zustimmen" (TRUE) vs „Nur zur Information" (FALSE).
- Hinweistext direkt darunter, der das Verhalten erklärt.

### Antrags-Detail (Admin-View)
Die Consents-Liste am Antrag zeigt:
- Bei `consent_type='explicit'`: bisheriger Eintrag mit Zeitstempel.
- Bei `consent_type='informational'`: andere Darstellung — z.B. „Zur Kenntnis genommen am …" statt „Zugestimmt am …".

### PDF (Beitrittsbestätigung)
Im Consents-Block: separate Auflistung für explicit vs informational. Rechtssicherheit: bei einem Streitfall kann der EEG-Admin unterscheiden was das Mitglied aktiv akzeptiert hat.

## Acceptance Criteria (skizziert)

- [ ] Migration: `consent_type` an `document_consent` (TEXT NOT NULL DEFAULT 'explicit', CHECK in `'explicit', 'informational'`).
- [ ] Backend `application_service.SubmitApplication`: schreibt Consent-Einträge für ALLE konfigurierten Dokumente mit dem korrekten Typ.
- [ ] Frontend Public-Form: zwei getrennte Sektionen, kein Häkchen für info-only-Dokumente.
- [ ] Admin-UI: Labelling klären, Toggle-Verhalten unverändert (Speicher-Flag bleibt `requires_consent`).
- [ ] Antrags-Detail-PDF und Admin-View: separate Anzeige.
- [ ] Migration der Bestandsdaten: alle bestehenden Consents bekommen `consent_type='explicit'` (war ja die einzige Variante). Bei Bestands-Dokumenten mit `requires_consent=FALSE` werden keine neuen Consent-Einträge zurückgeschrieben, aber für die nächste Submit-Welle wird das neue Verhalten greifen.
- [ ] Docs: api-spec, domain-model, user-guide.

## Open Questions

### Q1: Was passiert mit bestehenden Antrags-Consents, die bei optionalen Dokumenten ein leeres Häkchen hatten?
Bisher wurde nichts geschrieben wenn nicht angehakt. Mit PROJ-36 würden ALLE Antrags-Submits einen Eintrag pro Dokument bekommen (`explicit` bei Pflicht, `informational` bei Info-only). Empfehlung: keine rückwirkende Datenmigration.

### Q2: Soll ein dritter Modus eingeführt werden (z.B. „optional", explizit anhakbar aber nicht Pflicht)?
Aus User-Feedback klar **nein** — das ist genau der verwirrende Zustand, den wir abschaffen.

### Q3: Pflicht-Anzeige eines Info-Dokuments?
Sollte das Mitglied einen Info-Link anklicken **müssen** (read-confirmation)? Vorschlag: nein — der Hinweistext bei Submit reicht.

## Out of Scope
- Verträge / digital signierbare Dokumente — separates Feature falls je gewünscht.
- Versionierung der Dokumente — bereits durch Consent-Snapshot abgedeckt.

## Pointer-Files

- Spec: `features/PROJ-36-optional-legal-documents-as-info.md`
- Related: `features/PROJ-9-legal-documents.md`, `features/PROJ-18-datenschutz-policy-toggle.md`
- Schema: `member_onboarding.legal_document`, `member_onboarding.document_consent`
