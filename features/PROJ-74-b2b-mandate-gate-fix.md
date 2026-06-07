# PROJ-74: B2B-Mandat-Gate-Fix + UI-Klarstellung für SEPA-Mandat-Toggles

## Status: Planned
**Created:** 2026-06-06
**Last Updated:** 2026-06-06

## Dependencies
- Erfordert: PROJ-48 (`einzugsart` als per-Antrag-Mandat-Variante) — Deployed
- Erfordert: PROJ-73 (Cleanup `use_company_sepa_mandate`) — In Progress, im selben Deploy-Zyklus
- Verändert: `buildSEPAMandateData` (internal/application/application_service.go:1606) — zentrale Gate-Funktion für SEPA-Mandat-PDF-Generierung

## Hintergrund

Beim Code-Trace eines Tester-Befunds („trotz `SEPAMandateEnabled=false` kommt ein Mandat-PDF") wurde sichtbar, dass die heutige Gate-Logik **alle** PDF-Generierungspfade sperrt, sobald `SEPAMandateEnabled=false`. Das ist nur für Core-Mandate korrekt — für B2B (SEPA-Firmenlastschrift) ist es falsch:

- **SEPA-Rulebook:** B2B-Lastschrift erfordert zwingend ein unterschriebenes Mandat-PDF. Eine reine Online-Checkbox ersetzt das Mandat nicht.
- **Owner-Use-Case:** Eine EEG, die für Privatmember mit der Checkbox-Lösung arbeitet (`SEPAMandateEnabled=false`), soll trotzdem für B2B-Anträge ein Mandat-PDF beim Import bekommen.
- **Heute (Bug):** EEG mit `SEPAMandateEnabled=false` + Admin stellt Antrag auf `einzugsart=b2b` → `buildSEPAMandateData()` returnt nil → Post-Import-Mail loggt „SEPA mandate skipped — EEG configuration incomplete" und sendet die Mail ohne Anhang. Das Mitglied bekommt keine Mandat-Vorlage und der EEG-Vorstand kann mangels Unterschrift rechtswidrig fakturieren.

Zusätzlich sind die zwei Settings-UI-Toggles („SEPA-Mandat von der EEG bereitstellen" und „SEPA-Mandat erst beim Import senden") irreführend — sie sagen nicht, dass beide ausschließlich für Core-Mandate gelten. B2B-Mandate werden im Code-Pfad immer beim Import generiert ([admin_service.go:964](internal/application/admin_service.go#L964): `wantsB2B := einzugsart == "b2b"`, unabhängig vom Timing-Toggle), aber die UI suggeriert anderes.

## User Stories

- Als **EEG-Vorstand** mit Privatmember-Schwerpunkt möchte ich die Checkbox-Lösung verwenden (kein PDF-Mandat im Default), aber gleichzeitig sicher sein, dass für einzelne B2B-Mitglieder trotzdem ein rechtskonformes Mandat-PDF beim Import erzeugt wird.
- Als **Admin** möchte ich beim Setzen von `einzugsart=b2b` im Bearbeiten-Dialog wissen, dass das B2B-Mandat-PDF unabhängig von der EEG-Setting beim Import generiert wird.
- Als **Admin** möchte ich an den zwei Settings-Toggles auf einen Blick verstehen, dass sie nur Core-Mandate betreffen — damit ich nicht annehme, dass „SEPA-Mandat erst beim Import senden" auch B2B-Versand steuert.
- Als **Owner** möchte ich verhindern, dass ein B2B-Antrag importiert wird, ohne dass das vorgeschriebene Mandat-PDF erzeugt werden kann (DSGVO + SEPA-Compliance).
- Als **Admin** möchte ich proaktiv gewarnt werden, falls EEG-Stammdaten unvollständig sind und ich gleichzeitig planen könnte, B2B-Anträge zu bearbeiten — bevor ich auf den Import-Bug stoße.

## Akzeptanzkriterien

### Backend Gate-Logik

- [ ] `buildSEPAMandateData(app, ep)` differenziert nach `app.Einzugsart`:
  - `einzugsart="b2b"` → der EEG-Toggle `ep.SEPAMandateEnabled` wird **ignoriert**; geprüft wird ausschließlich `missingMandateFields(ep)`. Wenn Stammdaten vollständig: PDF-Daten zurück. Wenn unvollständig: nil zurück.
  - `einzugsart="core"` → heutiges Verhalten unverändert: nil wenn `!ep.SEPAMandateEnabled` oder Stammdaten fehlen.
  - `einzugsart="kein_sepa"` → bleibt nil (kein Mandat).
- [ ] `missingMandateFields(ep)` liefert die Liste **unabhängig** von `ep.SEPAMandateEnabled`. Die heutige Frühe-Rückgabe `if !ep.SEPAMandateEnabled { return nil }` entfällt. Begründung: Liste ist eine reine Stammdaten-Prüfung; Aufrufer entscheidet kontextabhängig, ob die Liste relevant ist.
- [ ] Submit-Pfad ([application_service.go:744](internal/application/application_service.go#L744)) unverändert — `einzugsart` ist beim Submit hartkodiert `"core"`, also greift kein Fix.
- [ ] Post-Import-Pfad ([admin_service.go:1037-1039](internal/application/admin_service.go#L1037)) generiert B2B-PDF auch bei `SEPAMandateEnabled=false`, sofern Stammdaten vollständig sind.
- [ ] Resync-Pfad ([resync_service.go:281-283](internal/application/resync_service.go#L281)) generiert B2B-PDF auch bei `SEPAMandateEnabled=false`, sofern Stammdaten vollständig sind.

### Hart-Fail bei B2B + fehlenden Stammdaten

- [ ] Wenn `einzugsart=b2b` UND `missingMandateFields(ep)` nicht leer ist UND der Antrag den Status-Wechsel zu `imported` ansteht: der Status-Wechsel bricht ab, der Antrag verbleibt im vorherigen Status. Der Fehler wird dem auslösenden Admin sichtbar zurückgegeben (HTTP-Antwort mit menschenlesbarer Begründung + Auflistung der fehlenden Felder).
- [ ] Begründung: B2B-Lastschrift ohne unterschriebenes Mandat ist nach SEPA-Rulebook rechtswidrig. Ein "Skip mit Warn-Log" wie bei Core ist hier unzulässig — der Owner darf nicht im Nachhinein feststellen, dass ein importierter B2B-Antrag kein Mandat-PDF erhielt.
- [ ] Bei Core (`einzugsart=core`) + fehlenden Stammdaten: bleibt beim heutigen Verhalten (Warn-Log + Skip, kein Hart-Fail), weil eine Online-Zustimmung als Fallback existiert.
- [ ] Resync-Pfad: gleiche Hart-Fail-Logik wie Post-Import.
- [ ] Hart-Fail-Fehlermeldung enthält die genauen fehlenden Felder (`eeg_name`, `eeg_street`, `creditor_id`, …) — der Admin muss sofort sehen, was zu pflegen ist.

### Settings-UI: Hint-Popover an den Toggles

- [ ] Am Toggle „SEPA-Mandat von der EEG bereitstellen": Info-Icon neben dem Label, Click öffnet Popover mit Text:
  > „Steuert nur Privat-Mitglieder (Core-Lastschrift). Wenn aktiv: Member bekommen ein vorausgefülltes Lastschriftmandat-PDF an die Eingangsbestätigung angehängt. Wenn inaktiv: reine Online-Zustimmung per Checkbox.
  >
  > Hinweis: B2B-Mandate (Firmenlastschrift) werden in jedem Fall als PDF beim Import generiert — das schreibt das SEPA-Regelwerk vor und lässt sich nicht deaktivieren."
- [ ] Am Toggle „SEPA-Mandat erst beim Import senden": Info-Icon neben dem Label, Click öffnet Popover mit Text:
  > „Steuert nur das Versand-Timing für Privat-Anträge (Core-Lastschrift). Standard (aus): das Core-Mandat geht als PDF-Anhang mit der Eingangsbestätigung, der Member trägt die Mandatsreferenz händisch ein. Aktiv: das Core-Mandat kommt erst beim Import mit eingedruckter Mandatsreferenz (= Mitgliedsnummer).
  >
  > B2B-Mandate kommen unabhängig vom diesem Toggle beim Import — die Mandatsreferenz wird erst dort vergeben."
- [ ] Pattern aus [.claude/rules/frontend.md](.claude/rules/frontend.md) Hint/Popover-Sektion. Touch-tauglich. Keine `placeholder=`-Attribute.

### Warn-Banner: proaktive Stammdaten-Warnung

- [ ] Der bestehende Warn-Banner „SEPA-Mandate werden derzeit nicht generiert" (heute nur bei `sepaMandateEnabled=true` aktiv, [admin-eeg-settings-editor.tsx:595](src/components/admin-eeg-settings-editor.tsx#L595)) erscheint künftig auch bei `sepaMandateEnabled=false`, sofern `missingMandateFields()` nicht leer ist.
- [ ] Bei `sepaMandateEnabled=false`-Variante wird der Text angepasst, z.B.:
  > „⚠ Falls Sie B2B-Anträge bearbeiten, können keine Mandate generiert werden
  >
  > Folgende Pflichtfelder aus den EEG-Stammdaten fehlen: **{missing}**. Bitte in eegFaktura ergänzen und oben auf „Aus eegFaktura aktualisieren" klicken. Für reine Privat-Anträge (Core-Lastschrift) ist die Online-Zustimmung weiterhin möglich, aber B2B-Anträge werden beim Import abgewiesen."
- [ ] Banner-Logik bleibt clientseitig (kein Backend-Roundtrip): identische `missing`-Liste wie heute, nur die Anzeige-Bedingung wird auf `missing.length > 0 && (sepaMandateEnabled || true)` reduziert (= immer wenn Felder fehlen).

### Tests

- [ ] **Go: `buildSEPAMandateData`** — zwei neue Unit-Tests:
  - `TestBuildSEPAMandateData_B2B_Bypasses_SEPAMandateEnabled`: `ep.SEPAMandateEnabled=false`, alle Stammdaten OK, `app.Einzugsart="b2b"` → returns `*pdf.SEPAMandateData` non-nil.
  - `TestBuildSEPAMandateData_Core_StillRequires_SEPAMandateEnabled`: `ep.SEPAMandateEnabled=false`, alle Stammdaten OK, `app.Einzugsart="core"` → returns nil.
- [ ] **Go: Hart-Fail im Post-Import-Pfad** — Test, der bei `einzugsart=b2b` + fehlender Creditor-ID den Status-Wechsel auf `imported` mit Fehler abbricht. Antrags-Status bleibt unverändert.
- [ ] Bestehende Tests müssen alle weiter grün laufen (`go test ./...`).

### Doku

- [ ] [docs/domain-model.md](docs/domain-model.md): bei `sepa_mandate_enabled` ergänzen: „**Wirkt nur auf Core-Mandate.** B2B-Mandate (`einzugsart=b2b`) werden in jedem Fall als PDF beim Import generiert, weil das SEPA-Regelwerk eine unterschriebene Mandatsvorlage verlangt."
- [ ] [docs/architecture.md](docs/architecture.md) Mail-Tabelle: B2B-Branch ergänzen, der den Hart-Fail-Fall dokumentiert.
- [ ] [docs/api-spec.md](docs/api-spec.md) Settings-Endpoint: bei `sepaMandateEnabled` und `sepaMandateAtImport` jeweils klarstellen, dass beide nur Core-Mandate betreffen.
- [ ] [CHANGELOG.md](CHANGELOG.md) Eintrag unter `[Unreleased]` — beschreibt Gate-Fix + UI-Klarstellung + Hart-Fail-Verhalten.

## Edge Cases

- **EEG ohne Creditor-ID + ausschließlich Privat-Anträge:** Warn-Banner zeigt sich, ist aber nur Hinweis. Privat-Anträge laufen weiter (Online-Zustimmung). Owner kann den Banner entweder durch Creditor-ID-Pflege auflösen oder ignorieren, falls B2B nie auftreten wird.
- **Admin ändert `einzugsart=core → b2b` BEVOR Import:** Beim späteren Import greift der Hart-Fail, falls Stammdaten fehlen. Status-Wechsel wird abgewiesen, der Admin muss Stammdaten pflegen oder die Änderung rückgängig machen.
- **Admin ändert `einzugsart=b2b → core` BEVOR Import:** Kein Hart-Fail, weil Core-Pfad bei `SEPAMandateEnabled=false` weiterhin still überspringt (Online-Zustimmung gilt).
- **Resync-Pfad (PROJ-70):** ein Admin klickt „SEPA-Mandat erneut senden" auf einem aktivierten B2B-Antrag, die EEG hat Creditor-ID inzwischen gelöscht (sehr unwahrscheinlich, aber denkbar). Hart-Fail mit derselben Fehlermeldung wie im Post-Import. Bestehender Mandats-Versand wird nicht überschrieben.
- **Mandat-PDF erfolgreich generiert, aber Mail-Versand schlägt fehl:** Hart-Fail nur auf PDF-Generierungs-Stufe. Mail-Versand ist bewusst best-effort (sonst blockieren SMTP-Probleme den Import). Mandate, deren Mail nicht ankam, können per Resync nachgeschickt werden.
- **Concurrent Admin-Edits:** zwei Admins editieren denselben Antrag parallel — einer setzt `b2b`, der andere `core`. Der gespeicherte Wert gewinnt; Hart-Fail-Logik bezieht sich auf den persistierten Wert beim Status-Wechsel, nicht auf einen ungespeicherten UI-Zustand.

## Technische Anforderungen

- **Performance:** keine zusätzlichen DB-Roundtrips — `buildSEPAMandateData` ist bereits memory-only.
- **Sicherheit / Compliance:** Hart-Fail im B2B-Pfad verhindert rechtswidrigen Import. Audit-Log enthält Antrags-ID + RC + fehlende Felder.
- **Logging:** Hart-Fail-Ablehnungen sind `slog.Warn` mit Application-ID, RC, `missingFields`. Kein PII (kein Member-Name, keine IBAN).
- **Browser-Support:** Hint-Popover ist Click-basiert (shadcn Popover) — Touch-tauglich.
- **Memory-Regeln:**
  - `feedback_no_placeholders`: kein `placeholder=` auf Inputs.
  - `feedback_mail_hard_fail`: passt zur Owner-Direktive (admin-getriebene Operationen scheitern sichtbar).
  - `feedback_shared_helpers_for_parallel_paths`: nur **eine** Quelle für die B2B-Gate-Logik (`buildSEPAMandateData`). Keine parallele Struct-Logik in admin_service oder resync_service.
- **Hint-Popover-Pattern:** wie in [.claude/rules/frontend.md](.claude/rules/frontend.md) referenziert (Popover statt Tooltip, Info-Icon, max-w-60).

## Nicht im Scope

- **Pro-Antrag-Override „dieser B2B-Antrag bekommt KEIN PDF":** rechtlich nicht zulässig, daher kein Feature.
- **Mischbetrieb-EEGs mit getrennten Privat/Firma-PDF-Pfaden in derselben EEG:** automatisch durch das `einzugsart`-Modell gelöst.
- **Settings-Mode-Awareness-Trigger erweitern:** `sepaMandateAtImport` bleibt einziger SEPA-Advanced-Trigger. Die zusätzlich klarere Hint-Anzeige ist kein Awareness-Trigger.
- **B2B-Mandat per Submit-Mail versenden:** bewusst ausgeschlossen — die Mandatsreferenz (= Mitgliedsnummer) wird erst beim Import vergeben.
- **Migration für bestehende EEGs:** keine — Datenmodell unverändert, reiner Code-Fix.

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_Skipped — Owner-Direktive 2026-06-06: Direkt-Implementation. Scope ist mechanisch (Gate-Funktion-Erweiterung + UI-Texte), Daten- und API-Modell unverändert._

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
