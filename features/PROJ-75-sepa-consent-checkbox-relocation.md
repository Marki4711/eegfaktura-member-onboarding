# PROJ-75: SEPA-Einwilligungs-Checkbox in Bankverbindungs-Card verschoben

## Status: In Progress
**Created:** 2026-06-06
**Last Updated:** 2026-06-06

## Dependencies
- Erfordert: PROJ-32 (EEG-Stammdaten-Sync) — `eegName` + `creditorId` müssen
  aus dem Core gesynct sein, damit der Checkbox-Text vollständig gerendert
  werden kann. Fallback-Text greift solange.

## Hintergrund

Tester-Wunsch 2026-06-06: Die SEPA-Einwilligungs-Checkbox im öffentlichen
Anmeldeformular saß bisher im allgemeinen Einwilligungsblock — neben
„Richtigkeit meiner Angaben" und der Netzbetreiber-Vollmacht — und
forderte den Member auf, dem Lastschriftmandat zuzustimmen. Räumlich war
sie damit weit von den Konto-Eingabefeldern entfernt, was die direkte
Beziehung zwischen Bestätigung und Daten verschleierte.

Außerdem war der bisherige Text generisch („… der Energiegemeinschaft ein
SEPA-Lastschriftmandat …") und wies nicht auf den konkreten Empfänger
oder die Creditor-ID hin. Owner-Wunsch: Klarstellung mit EEG-Name und
Creditor-ID am direkten Eingabe-Ort.

## User Stories

- Als **Mitglied** möchte ich die SEPA-Einwilligungs-Checkbox direkt unter
  meinen Konto-Eingabefeldern sehen, damit der Kontext klar ist.
- Als **Mitglied** möchte ich im Checkbox-Text den Namen der konkreten EEG
  und ihre Creditor-ID sehen, damit ich weiß, wem ich die Einzugs-
  Ermächtigung erteile.
- Als **EEG-Vorstand** möchte ich, dass meine Mitglieder unmissverständlich
  erkennen, wer der Lastschrift-Berechtigte ist — auch zur DSGVO-konformen
  Information.

## Akzeptanzkriterien

- [x] Die SEPA-Einwilligungs-Checkbox erscheint nicht mehr im allgemeinen
  Einwilligungsblock (oberhalb von „Netzbetreiber-Vollmacht").
- [x] Die Checkbox erscheint stattdessen in der Bankverbindungs-Card
  (`Card` mit `CardTitle="Bankverbindung"`), **direkt unter** den
  Eingabefeldern IBAN/Kontoinhaber:in/Bankname.
- [x] Sichtbarkeit unverändert: nur bei `sepaMandateEnabled=false` (= EEG
  arbeitet mit Online-Zustimmung statt PDF-Mandat). Bei
  `sepaMandateEnabled=true` ist die Checkbox unsichtbar — das
  PDF-Mandat dokumentiert die Akzeptanz, `sepaMandateAccepted` wird im
  Form-State automatisch auf `true` gesetzt.
- [x] **Neuer Text:**
  > „Hiermit bestätige ich die Richtigkeit der angegebenen
  > Kontoinformationen und ermächtige **\<Name der EEG\>** zum
  > Bankeinzug im Rahmen der Leistungsabrechnung. \*"
  >
  > Darunter, kleiner: „Creditor ID: **\<Creditor-ID der EEG\>**"
- [x] Beide Variablen kommen aus dem Public-Registration-Config-Payload
  (`RegistrationConfig.eegName`, `RegistrationConfig.creditorId`).
- [x] Backend liefert die zwei Felder neu im Public-Endpoint
  `GET /api/public/registration/{rc_number}` — Nullable, weil vor dem
  ersten PROJ-32-Sync nicht zwingend gepflegt.
- [x] Fallback-Verhalten:
  - Wenn `eegName` leer/null: Text rendert mit „… ermächtige die
    Energiegemeinschaft zum Bankeinzug …" (generisch).
  - Wenn `creditorId` leer/null: die „Creditor ID: …"-Zeile wird
    ausgeblendet, statt eine leere Zeile zu zeigen.
- [x] Der Wert von `sepaMandateAccepted` wird unverändert im Submit-Payload
  übergeben — Backend bleibt unverändert (außer dem neuen Config-Feld).

## Edge Cases

- **EEG ohne PROJ-32-Sync (eegName + creditorId beide leer):** Fallback-
  Text rendert ohne Namen + ohne Creditor-ID. Funktional gleichwertig
  zum alten Text, aber visuell sauber.
- **EEG mit eegName, ohne creditorId:** Name wird im Hauptlabel
  eingesetzt; Creditor-Zeile entfällt. Kein „undefined".
- **EEG mit creditorId, ohne eegName:** Fallback im Hauptlabel; Creditor-
  Zeile rendert.
- **`sepaMandateEnabled=true`:** Checkbox bleibt unsichtbar, der Member
  bekommt das PDF-Mandat per Eingangsbestätigung. `sepaMandateAccepted`
  ist im Form-State implizit `true`.
- **`sepaMandateAtImport=true`:** Der bestehende Hinweistext am Anfang der
  Bankverbindungs-Card („Das SEPA-Lastschriftmandat erhältst du nach der
  Freigabe deines Antrags per E-Mail …") bleibt sichtbar und steht
  weiterhin oberhalb der Eingabefelder. Sichtbarkeits-Bedingung bleibt
  `sepaMandateEnabled && sepaMandateAtImport`.
- **Member ändert Konto-Daten und scrollt nicht zurück:** die Checkbox
  ist jetzt direkt unter den Feldern — die Wahrscheinlichkeit, dass der
  Member sie übersieht, sinkt deutlich.

## Technische Anforderungen

- **Frontend-only** (mit minimaler Backend-Erweiterung um die zwei
  Config-Felder).
- **Keine Schema-Änderung** in der DB — `eegName` und `creditorId` werden
  bereits in `registration_entrypoint` gespeichert (PROJ-32).
- **Keine API-Vertrags-Änderung** für Submit — `sepaMandateAccepted`
  bleibt im Payload.
- **Memory-Regeln:**
  - `feedback_no_placeholders`: keine Placeholder-Attribute.
  - `feedback_anonymized_examples`: in Doku Max Mustermann / Musterbetrieb.
  - `feedback_no_proj_refs_in_user_doc`: User-Guide PROJ-frei.

## Nicht im Scope

- Pre-Filling von `accountHolder` aus Member-Name — eigene Diskussion.
- Anzeige der Creditor-ID im PDF-Mandat (das hat sie bereits per PROJ-12).
- Erweiterung des Texts um Sondervereinbarungen (Vorabankündigung,
  Rückbuchungsfrist) — Standard-SEPA-Wortlaut, nicht relevant.
- Visuelles Redesign der Bankverbindungs-Card.

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_Skipped — Owner-Direktive 2026-06-06: reine GUI-Anpassung mit minimaler
Backend-Erweiterung (zwei Config-Felder). Kein neues Datenmodell._

## QA Test Results
_Optional — manuelle Browser-Verifikation reicht. Tests sind unverändert
(Checkbox-Verhalten + Submit-Payload sind die gleichen)._

## Deployment
_Im nächsten regulären Helm-Upgrade-Zyklus. Keine Migration nötig._
