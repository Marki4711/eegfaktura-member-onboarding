# PROJ-73: Verwaisten EEG-Toggle `use_company_sepa_mandate` entfernen

## Status: In Progress
**Created:** 2026-06-06
**Last Updated:** 2026-06-06

## Dependencies
- Erfordert: PROJ-48 (`einzugsart` als per-Antrag-Mandat-Variante) — fertig
- Abhängig: keine
- Supersedes: PROJ-14 (Auto-Mapping `company|association → B2B`, durch PROJ-48 inhaltlich ersetzt)

## Hintergrund

PROJ-14 (2026-04-24) führte den EEG-globalen Toggle
`registration_entrypoint.use_company_sepa_mandate` ein, um zu entscheiden,
ob Unternehmen/Vereine das B2B-Mandat statt des Core-Mandats erhalten.

PROJ-48 (2026-05-15) ersetzte diese Logik durch das per-Antrag-
`einzugsart`-Modell:
- `einzugsart=core` → Core-Mandat
- `einzugsart=b2b` → B2B-Mandat
- `einzugsart=kein_sepa` → kein Mandat

Der EEG-Toggle blieb seither **funktionslos**: keine PDF-Auswahl-Logik
liest ihn, keine Mail-Logik checkt ihn. Im Settings-UI steht er aber
weiter als Switch, was Admins verwirrt (Tester-Befund 2026-06-06:
„der Schalter macht nichts").

Dieser Cleanup entfernt den Toggle ersatzlos.

## User Stories

- Als **EEG-Administrator** möchte ich keine UI-Toggles sehen, die nichts
  bewirken, damit ich nicht ratlos vor einem Schalter stehe, der meine
  Aktion ignoriert.
- Als **Owner** möchte ich toten Domain-Code entfernen, damit künftige
  Wartung nicht durch funktionslose Pfade abgelenkt wird.

## Akzeptanzkriterien

- [x] **Migration 000066** droppt `registration_entrypoint.use_company_sepa_mandate`. Down-Migration setzt die Spalte mit Default `FALSE` wieder ein (Werte gehen nicht zurück — akzeptabel, da der Toggle keine Domain-Wirkung hatte).
- [x] **Backend**: Feld aus `shared.RegistrationEntrypoint` entfernt. `RegistrationEntrypointRepository.GetByRCNumber` SELECT + Scan ohne das Feld. `SaveEEGSettings`-Signatur reduziert um den entsprechenden Parameter. `EEGSettingsForImport` (Tx-Struct) ohne das Feld. `SaveAllEEGSettingsTx`-UPDATE-Liste reduziert.
- [x] **HTTP-Settings-Handler**: GET-Response gibt das Feld nicht mehr aus, PUT-Body lehnt es stillschweigend ab (`json.Decode` ignoriert unbekannte Felder).
- [x] **Configexport-Schicht**: Schema-Feld als Legacy-Alias (`LegacyUseCompanySEPAMandate *bool, omitempty`) erhalten, damit ältere Backup-JSONs mit `DisallowUnknownFields` weiter decodieren. Exporter schreibt es nicht mehr, Importer ignoriert es, Diff-Ausgabe ignoriert es.
- [x] **Frontend**: Feld aus drei Interfaces in `src/lib/api.ts` entfernt. Settings-Mode-Advanced-Trigger in `src/lib/settings-mode.ts` ohne `useCompanySEPAMandate`-Bedingung — die übrigen sechs Trigger reichen aus. UI-Toggle aus `admin-eeg-settings-editor.tsx` entfernt inklusive State, Snapshot, Discard-Pfad, Save-Payload. Cleanup-Aufruf `setUseCompanySEPAMandate(false)` auf `sepaMandateEnabled=false` ist mit dem Toggle weggefallen.
- [x] **Tests**: Vitest-Spec `settings-mode.test.ts` ohne den `useCompanySEPAMandate`-Trigger-Test, Default-EEGSettings-Helper ohne das Feld. Go-Test `application_service_test.go` ohne das Feld. Pre-/Post-PROJ-67-Bundle-Tests in `configexport_test.go` belassen das alte Feld in der JSON-Payload (Rückwärtskompat-Beweis). E2E-Spec `tests/PROJ-14-company-sepa-mandate.spec.ts` entfernt (kein Pendant mehr in der UI).
- [x] **Doku**: `docs/domain-model.md` Eintrag entfernt + Verweis auf das `einzugsart`-Modell ergänzt. `docs/architecture.md` Mail-Tabelle ohne den `useCompanySEPAMandate`-Hinweis. `docs/api-spec.md` Request-/Response-Beispiele ohne das Feld + Erklärung des Legacy-Tolerated-Verhaltens. `docs/open-questions.md` OQ-2 mit PROJ-73-Vermerk. `CHANGELOG.md` Eintrag unter `[Unreleased]`.
- [x] **Spec-Pflege**: `PROJ-14` auf Status `Superseded` mit Begründungs-Hinweis. `PROJ-73` (diese Spec) angelegt. `features/INDEX.md` + `docs/PRD.md`-Roadmap aktualisiert.
- [ ] **Build + Tests grün**: `go build ./...`, `go test ./...`, `npx tsc --noEmit` ohne Fehler.
- [ ] **CI-Pipeline grün**: nach Push muss der Build-Workflow durchlaufen.

## Edge Cases

- **EEGs mit aktuell `use_company_sepa_mandate=true`**: Der Wert geht beim DROP verloren. Verlustfrei, da der Toggle keine Domain-Wirkung hatte. Owner kommunziert das im CHANGELOG.
- **Alte PROJ-61-Backup-JSONs**: enthalten den Schlüssel `useCompanySEPAMandate` noch. Der strict Decoder (`DisallowUnknownFields` in `internal/configexport/importer.go`) würde ohne Schema-Anker einen Parse-Fehler werfen. Lösung: Schema-Feld bleibt als Legacy-Anker (`*bool, omitempty`), Wert wird beim Import verworfen.
- **Settings-Mode-Advanced-Trigger**: `useCompanySEPAMandate=true` war einer von sieben Triggern, die den „Erweitert"-Banner auslösen. Nach Entfernung bleiben sechs Trigger (PROJ-31, PROJ-48, PROJ-37, PROJ-52 (×2), PROJ-53). EEGs, die heute den Banner nur wegen dieses Toggles zeigen, werden ihn nach Cleanup nicht mehr zeigen — unkritisch, da der Toggle nichts steuert.
- **Frontend-Legacy-Clients**: ältere Admin-UIs senden eventuell noch das Feld im PUT-Body. `json.Decode` im Backend toleriert unbekannte Felder (kein 400).

## Nicht im Scope

- Anpassung der B2B-Mandat-Gate-Logik bei `SEPAMandateEnabled=false` — siehe PROJ-74.
- Klarstellung der UI-Texte beim SEPA-Mandat-Toggle und Mandat-Timing-Toggle — siehe PROJ-74.

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_Direkt-Implementierung 2026-06-06 ohne separates Tech-Design-Skill — der Scope ist mechanisch (DB-Spalte droppen + alle Code-Stellen entfernen) und braucht kein Architekturpapier._

## QA Test Results
_Skipped — reiner Cleanup ohne neue Domain-Logik. Bestehende Test-Suite (Go + Vitest + tsc) decken das Verhalten ab._

## Deployment
_Eingegliedert in den nächsten regulären Helm-Upgrade-Zyklus. Migration 000066 läuft beim Migrate-Job automatisch mit._
