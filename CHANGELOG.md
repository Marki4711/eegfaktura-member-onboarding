# Changelog

Alle nennenswerten Änderungen an diesem Projekt werden hier dokumentiert.
Format basiert auf [Keep a Changelog](https://keepachangelog.com/de/1.0.0/).

> Die Versionsnummern im CHANGELOG sind unabhängig von den Git-Tags vergeben,
> da die ursprünglichen Tags nicht konsistent nummeriert wurden.

---

## [Unreleased]

### Fix — PROJ-87: USt-Pflicht-Status in der Antrags-Detail-Ansicht sichtbar *(2026-06-08)*

Tester-Feedback 2026-06-08: in der Antrags-Detail-Ansicht
(`admin-application-detail.tsx`) war der USt-Pflicht-Status eines
Antragstellers (Kleinunternehmer ja/nein) nicht direkt erkennbar.
Der Admin musste „leere UID-Nummer ⇒ Kleinunternehmer" im Kopf
nachvollziehen — in der Bearbeiten-Maske dagegen zeigte eine
Checkbox den Wert direkt.

Fix: neuer `<Field>`-Eintrag „USt-pflichtig" mit Wert „Ja" oder
„Nein (Kleinunternehmerregelung)" für Mitgliedstyp `company` und
`association`. Position: zwischen UID-Nummer und Firmenbuch-/
Vereinsnummer.

Ableitungs-Logik identisch zum Edit-Form
(`admin-edit-form.tsx:100-102`):
`!!(application.uidNumber && application.uidNumber.trim() !== "")`
Beide Pfade leiten aus demselben DB-Feld ab (PROJ-63-Direktive:
`vatLiable` ist kein DB-Feld, sondern reines UI-Gate).

5-Zeilen-Frontend-Change. Kein API/DB/Helm-Eingriff.

Spec: `features/PROJ-87-vatliable-status-in-admin-detail.md`.

### Fix — PROJ-86: isKnownStatus-Whitelist PROJ-46-Drift-Hotfix *(2026-06-08)*

`isKnownStatus()` in `internal/http/admin.go` listete nur 9 von 12
Werten der `shared.ApplicationStatus`-Enumeration. Die drei
PROJ-46-Status (`awaiting_bank_confirmation`, `ready_for_activation`,
`activated`) fehlten seit ihrer Einführung 2026-05-17. Konsequenz:
jeder direkte `POST /api/admin/applications/{id}/status` mit einem
dieser Werte wurde vom Backend mit `400 Validation failed:
toStatus = unrecognised status value` abgelehnt. Das Frontend zeigte
nur die generische Top-Level-Message, nicht den Field-Error — daher
unbemerkt geblieben.

Owner-Befund 2026-06-08: nach `helm upgrade` von PROJ-79/82/83/84
zeigte ein b2b-Antrag im Status „Warte auf Bank-Bestätigung" beim
Klick auf „Bank-Bestätigung erhalten" das rote „Validation failed".
Auto-Modus (Activation-Check-Batch) und Vorstands-Workflow (PROJ-76)
gehen am Endpoint vorbei — dadurch ist der Bug Wochen lang
unauffällig geblieben.

Fix: switch-Liste in `isKnownStatus` um die 3 fehlenden Status
erweitert. Doc-Kommentar dokumentiert die Drift-Geschichte.

Drift-Wache: neuer Go-Test `TestIsKnownStatus_CoversAllApplicationStatuses`
iteriert über alle `shared.Status*`-Konstanten und prüft, dass jeder
Wert von `isKnownStatus` akzeptiert wird. Eine
`adminUnreachableStatuses`-Map ist vorbereitet für künftige
internal-only-Status mit Begründungs-Eintrag. Plus
`TestIsKnownStatus_RejectsUnknownStrings` als Strict-Mode-Anker.

Pure Backend-Änderung: keine API-Vertragsänderung, keine
Migration, keine Helm-Werte-Änderung.

Sekundär-Befund (außerhalb PROJ-86-Scope): `admin-status-actions.tsx`
zeigt bei Nicht-Conflict-Fehlern nur die generische Top-Level-
`err.message` statt die Field-Errors. Bekannt, separater Fix wenn
Owner es priorisiert.

Spec: `features/PROJ-86-isknownstatus-proj46-drift-hotfix.md`.

### Feature — PROJ-84: EEG-Stammdaten-Editor auf Auto-Save mit Cross-Field-Gate *(2026-06-08)*

Der `AdminEEGSettingsEditor` (Tab „Stammdaten & SEPA" unter
`/admin/settings`) hatte als einziger Settings-Editor noch einen
expliziten „Konfiguration speichern"-Button — weil drei
Cross-Field-Validierungen (PROJ-37 Genossenschaftsanteile,
PROJ-80 SEPA-CORE-Audit-Coupling, PROJ-81 SEPA-Wahl-Whitelist)
serverseitig greifen, die einen naiven Auto-Save mit Halb-Zuständen
torpediert hätten.

Owner-Beobachtung 2026-06-08: „Es ist unintuitiv, dass bei den
Formular-Feldern automatisch gespeichert wird, bei den Stammdaten aber
nicht."

Fix: die drei Backend-Regeln werden client-seitig in
`src/lib/eeg-settings-validation.ts` gespiegelt (`validateCooperativeShares`,
`validateSEPACoreAuditCoupling`, `validateSEPAOptionalWhitelist`, plus
Aggregat `validateEEGSettingsForAutoSave`). Vor jedem
`autoSave.schedule()` läuft der Aggregat-Validator; bei einem
unvollständigen Bündel wird der Schedule übersprungen und ein amber
Hint-Banner direkt unter dem zugehörigen Master-Toggle gerendert.

Owner-approbierter Wortlaut: „Änderungen werden gespeichert, sobald die
folgenden Pflichtfelder ausgefüllt sind:" + bullet-Liste der konkret
fehlenden Felder.

Owner-Entscheidungen aus /grill-me 2026-06-08:
- Toggle-OFF blendet Sub-Felder vollständig aus und resetet den State
  (Validation wird dadurch automatisch grün, Backend cleart die
  DB-Spalten beim nächsten Save)
- Kein Feature-Flag — Roll-back via Git-Revert
- Last-Write-Wins (kein Optimistic-Lock)
- `onSaved`-Callback Editor → Parent verhindert Stale-Cache analog
  PROJ-82

Backend-Validation bleibt unverändert als Defense-in-Depth. Drift-Schutz
via Vitest-Permutationstest (23 Test-Cases in
`src/lib/eeg-settings-validation.test.ts`); bei Backend-Regel-Änderung
ohne Frontend-Update bleibt CI rot und Deploy blockiert.

Pure Frontend-Änderung: keine API/DB/Migration/Helm-Änderung.

Spec: `features/PROJ-84-eeg-settings-auto-save.md`.

### Feature — PROJ-83: Letzte EEG-Auswahl im Admin-Settings persistieren *(2026-06-08)*

`/admin/settings` initialisierte `selectedRc` bisher immer auf
`rcNumbers[0]`. Bei 10+ EEGs war das jedes Mal ein Mehrklick im
Listbox-Auswahlmenü.

Fix via `localStorage`-Persistenz (Owner-Direktive — Variante B aus
der Analyse). Neuer Helper `src/lib/last-used-rc.ts` mit `readLastUsedRc`/
`writeLastUsedRc`/`clearLastUsedRc`. Sicherheits-Eigenschaften:
- Storage-Inhalt ist nur die RC-Nummer als String — kein Token, kein
  PII, kein JSON-Wrapper mit Extras (verifiziert per Vitest-Test)
- Tenant-Scope-Validation beim Lesen: wenn die persistierte RC nicht
  mehr in den aktuellen `rcNumbers` (JWT-Claim) enthalten ist, wird
  der Wert still verworfen und aus dem Storage entfernt — kein Risiko
  einer 403-Schleife oder „komischen Auswahl"
- LocalStorage-Fehler (Privat-Modus, Quota) werden geschluckt, der
  UI-Pfad fällt auf das heutige Default-Verhalten

Namespaced Storage-Key `eegfaktura-onboarding:settings:lastRc` —
kein Konflikt mit anderen Anwendungen oder zukünftigen
Settings-Persistenzen.

Tests: 9 neue Vitest-Cases (Verhalten + Sicherheits-Anker). Build clean.

Spec: `features/PROJ-83-last-used-eeg-persistence.md`.

### Fix — PROJ-82: Settings-Formular-Editor — UI-Staleness bei Tab-Wechsel *(2026-06-08)*

`AdminFieldConfigEditor` persistierte Konfigurationsänderungen korrekt in
der DB, aber der Parent-State `fieldConfig` in
`src/app/admin/settings/page.tsx` wurde nicht aktualisiert. Bei Tab-Wechsel
auf einen anderen Settings-Tab und zurück unmountet Radix-`TabsContent`
den inaktiven Editor; beim Re-Mount initialisierte
`useState(() => mergeWithDefaults(initialConfig))` aus dem alten
Parent-Snapshot. Folge: Tab-Wechsel zurück zeigt den alten Stand, obwohl
in der DB der neue Wert steht. Hard-Refresh fixte es, weil der
Parent-Load-`useEffect` neu lief.

Fix: neuer `onSaved`-Callback auf den Editor-Props. Nach erfolgreichem
Auto-Save meldet der Editor den persistierten Stand zurück an den
Parent, der sein `fieldConfig`-State synchronisiert. Beim nächsten
Tab-Re-Mount kommt der frische Stand. Kein zusätzlicher API-Call,
keine Backend-Änderung, keine Migration.

Owner-Direktive 2026-06-08 (Variante B aus der Analyse).

Spec: `features/PROJ-82-fieldconfig-editor-staleness-fix.md`.

### Feature — PROJ-79: B2B-Import als CORE in eegFaktura-Core *(2026-06-08)*

`mapEinzugsart` in `internal/importing/payload.go` mappt jetzt
`einzugsart=b2b` auf `"CORE"` (statt `"B2B"`) im Core-API-Payload.
Hintergrund: das SEPA-B2B-SDD-Rulebook verlangt eine separate
Bank-Mandatsvereinbarung, die typischerweise Tage bis Wochen dauert.
Bis dahin würde eine sofortige B2B-Abbuchung von der Hausbank des
Mitglieds abgelehnt — der CORE-Pfad überbrückt diese Klärungs-Phase
risikolos. `application.einzugsart` in der Onboarding-DB bleibt
unverändert auf `b2b`; das Status-Modell läuft weiter über
`awaiting_bank_confirmation` (PROJ-46).

Beide Aktivierungs-Mail-Pfade (Auto-Modus aus PROJ-53 und
Vorstands-Modus aus PROJ-76) rendern bei b2b-Anträgen einen gelben
Hinweis-Banner mit der Aufforderung an die EEG-Kontaktperson, die
B2B-Aktivierung mit der Hausbank des Mitglieds zu klären und nach
Bestätigung den Core-SEPA-Typ manuell auf B2B umzustellen. Banner-HTML
lebt zentral in `internal/mail/b2b_notice.go`
(`RenderB2BImportNoticeBanner`) — Single source of truth für beide
Pfade, bewusste Verbesserung gegenüber der PROJ-81-Doppelverdrahtung
des kein_sepa-Banners.

Hartkodierte globale Regel, kein Per-EEG-Toggle, keine DB-Migration,
kein Helm-Eingriff. Bestandsanträge im Core bleiben unangetastet —
laufende B2B-Lastschriften mit aktivem Mandat dürfen nicht gestört
werden. Owner-Direktive 2026-06-08.

In der heutigen Production-Codebase entsteht `einzugsart=b2b`
ausschließlich durch manuellen Admin-Edit; Public-Submit und Externe
API erzeugen nur `core` oder `kein_sepa`. Das Mapping bleibt aber
konsistent für zukünftige b2b-Pfade.

Tests:
`TestBuildPayload_EinzugsartMapping` umgestellt + neuer
`TestBuildPayload_B2B_IntentionallyMappedToCORE`-Regressions-Wache;
Helper-Unit-Tests (`TestRenderB2BImportNoticeBanner` mit 7 Permutationen
+ `TestRenderB2BImportNoticeBanner_ContainsKeyPhrases`);
Mail-Integration-Tests für beide Pfade
(`TestSendActivationNotification_B2B_ShowsBanner_InEEGCopy`,
`TestSendBoardApprovalRequest_B2B_ShowsBanner` + Negativ-Tests für
einzugsart=core).

Doku-Updates: neue Sektion `3.1 SEPA-Typ-Mapping beim Core-Import` in
`docs/import-mapping.md`; neuer Subblock „Firmenlastschrift im
Faktura-Core" in `docs/user-guide/06-admin-settings.md` (PROJ-frei,
anonymisiertes Beispiel mit Musterbetrieb GmbH); Eintrag in
`docs/user-guide/changelog.md`.

Spec: `features/PROJ-79-b2b-import-as-core.md` (16 Owner-Entscheidungen
aus /requirements + /grill-me).

### Feature — PROJ-81: SEPA-Einwilligung optional pro Mitgliedstyp *(2026-06-08)*

Per-EEG-Toggle „SEPA-Feld für ausgewählte Mitgliedstypen auf optional setzen" mit konfigurierbarer
Mitgliedstyp-Whitelist. Wenn aktiv, ist die SEPA-Einwilligungs-Checkbox
im Public-Form für die gelisteten Mitgliedstypen optional. Bei nicht
angekreuzter Checkbox wird `einzugsart=kein_sepa` gesetzt, kein
Mandat-PDF erzeugt. Bankdaten bleiben IMMER Pflicht (eegFaktura-Core
verlangt sie für jedes Mitglied — Owner-Direktive 2026-06-08).

`company` darf nie in der Whitelist sein (B2B-Pflicht-Lastschrift) —
Backend rejected, Configexport-Importer filtert und loggt.

**Datenmodell:**
- Migration `000072` ergänzt zwei Spalten auf
  `member_onboarding.registration_entrypoint`:
  - `sepa_optional_enabled BOOLEAN NOT NULL DEFAULT FALSE`
  - `sepa_optional_member_types TEXT[] NOT NULL DEFAULT '{}'`

**Backend:**
- Zentraler Helper `shared.IsSEPAOptional(ep, memberType)` als Wahrheit
  für Public-Submit, externe API und Admin-Service.
  `IsValidSEPAOptionalMemberType()` als Whitelist-Check (private/farmer/
  association/municipality — company nie).
- `CreateApplicationRequest.SepaMandateAccepted` ohne `required`-Tag.
  IBAN/AccountHolder bleiben `required` (Owner-Direktive: Bankdaten
  immer Pflicht).
- `application_service.CreateApplication`: Defense-in-Depth-Check und
  server-side `einzugsart`-Entscheidung (`kein_sepa` wenn Checkbox aus
  + Toggle/Mitgliedstyp passt).
- `internal/http/external.go`: harter SEPA-Check entfernt, durchgereicht
  an Service-Helper.
- `internal/http/admin.go`: GET/PUT-Body um beide Felder erweitert,
  Cross-Field-Validation (Toggle on ⇒ Liste nicht leer; `company` in
  Liste → 400; ungültige Mitgliedstypen → 400).
- `RegistrationEntrypointRepository.SaveEEGSettings` + `SaveAllEEGSettingsTx`
  + `EEGSettingsForImport` um beide Felder erweitert.
- Configexport: Schema/Exporter/Importer/Diff um beide Felder erweitert.
  Importer filtert `company` defensiv aus der Liste und loggt via
  `slog.Warn`.

**Mail-Hinweis bei kein_sepa (alle drei EEG-Mail-Pfade):**
- `application_submitted_eeg.html`, `application_activated_eeg.html`,
  PROJ-76-Beitrittserklärung-Mail bekommen einen gelben Hinweis-Banner
  bei `app.einzugsart=="kein_sepa"`: „Kein SEPA-Lastschriftmandat
  erteilt — die Abrechnung muss über einen alternativen Zahlungsweg
  direkt mit dem Mitglied vereinbart werden."
- Single source of truth über `NoSepaMandate bool` auf den Mail-Daten-
  Structs (`eegSubmissionData`, `activationTemplateData`) und inline
  im `SendBoardApprovalRequest`-Body.
- `application_imported_eeg.html` braucht keinen Eingriff (Mail geht
  bei `kein_sepa` nicht raus).

**Frontend:**
- `EEGSettings`, `EEGSettingsSavePayload`, `RegistrationConfig` und
  `ConfigEEGSettings` um beide Felder erweitert.
- Neuer Helper `src/lib/sepa-optional.ts` (`isSepaOptional()` +
  `isValidSepaOptionalMemberType()`) als TS-Pendant des Go-Helpers,
  inklusive Mitgliedstyp-Labels und -Reihenfolge.
- `src/components/admin-eeg-settings-editor.tsx`: Master-Toggle +
  4er-Checkbox-Liste (eingerückt, sichtbar nur bei aktivem Toggle).
  Pre-Save-Validation für „Toggle on + leere Liste" (UX-Komfort,
  Backend ist die Wahrheit).
- `src/components/registration-form.tsx`: `buildFormSchema` bekommt
  Sepa-Optional-Subset; Sternchen am Checkbox-Label + zod-`required`-
  Check konditional. Live-Re-Evaluation bei `memberType`-Wechsel über
  `form.watch`. Bei optionaler Variante zusätzlich Inline-Hint unter
  der Checkbox.
- `src/components/admin-application-detail.tsx`: blauer Info-Banner
  über der Bankverbindungs-Karte bei `einzugsart=kein_sepa`.
- `src/lib/settings-mode.ts`: `sepaOptionalEnabled=true` löst
  Advanced-Mode aus (analog PROJ-76/78 — Toggle bleibt nach Reload
  sichtbar).
- `src/components/admin-legal-documents-editor.tsx`: vollständiger
  Snapshot beim Toggle-Save um die beiden neuen Felder ergänzt
  (verhindert versehentliches Reset auf Default).

**Beifang-Fixes (im selben Commit):**
- `internal/dataexport/excel/fields.go`: `EinzugsartLabels` korrigiert
  — vor PROJ-81 kannte die Map nur `basis` und `b2b`, während die DB
  seit PROJ-23 `core`/`b2b`/`kein_sepa` speichert. Map hat jetzt alle
  drei Werte mit Labels. PROJ-81 macht den Bug sichtbarer (mehr
  `kein_sepa`-Anträge im Public-Form).
- `internal/application/application_service.go`: veralteter Block-
  Kommentar in `buildSEPAMandateData` aktualisiert (PROJ-80 hatte
  `SEPAMandateEnabled` entfernt, der Kommentar zog nicht nach).

**Tests:**
- `internal/shared/sepa_optional_test.go`: 8 Permutationen für den
  Go-Helper + Whitelist-Check.
- `internal/configexport/sanitize_sepa_optional_test.go`: 6 Cases
  inkl. company-Filter.
- `internal/dataexport/excel/fields_test.go`: fixiert das gefixte
  `EinzugsartLabels`-Mapping.
- `internal/mail/service_test.go`: zwei neue Tests für `NoSepaMandate`-
  Flag-Mapping (kein_sepa vs. core).
- `src/lib/sepa-optional.test.ts` (Vitest): Spiegel der Go-Helper-
  Permutationen, dient als Drift-Schutz zwischen Backend und Frontend.
- `src/lib/settings-mode.test.ts`: Advanced-Mode-Trigger-Test für
  `sepaOptionalEnabled=true`.

**Doku:**
- `docs/api-spec.md`: GET/PUT `/api/admin/settings/eeg` um die zwei
  Felder erweitert; Public-Submit- und Externe-API-Validation-Regeln
  ergänzt.
- `docs/domain-model.md`: zwei neue Spalten auf
  `registration_entrypoint` dokumentiert.
- `docs/user-guide/06-admin-settings.md`: neuer Abschnitt „SEPA-Wahl
  im Formular zulassen" mit Beispiel (Max Mustermann), Hinweis auf
  abweichenden Abrechnungsweg, Bankdaten-Pflicht.
- `docs/user-guide/changelog.md`: Eintrag 2026-06-08 (PROJ-frei).

### Feature — PROJ-80: SEPA-Settings-Vereinfachung *(2026-06-08)*

Der EEG-Toggle „SEPA-Mandat als Datei dem Antragsteller übermitteln"
(`sepa_mandate_enabled`) ist entfernt. Das System erzeugt jetzt für jedes
SEPA-Mitglied (`einzugsart != kein_sepa`) automatisch ein Mandat-PDF; die
Variante (Audit-Trail vs. Unterschriftenfeld) wird durch die zwei
nachgelagerten Toggles aus PROJ-78 gesteuert.

**Neue Konstanten** (vorher konfigurierbar, jetzt Pflicht):
- Online-Zustimmung-Checkbox im Mitgliederformular ist immer Pflicht.
- SEPA-Mandat-PDF wird für jedes SEPA-Mitglied automatisch erzeugt
  (sofern die EEG-Stammdaten reichen).
- Bankverbindungs-Block im Public-Form bleibt konditional auf
  `einzugsart != kein_sepa` (unverändert).

**Cross-Field-Coupling (CORE-Audit ⇒ Timing):** wenn
`sepa_mandate_core_audit_enabled = TRUE`, ist `sepa_mandate_at_import = TRUE`
zwingend. Bei Audit-Pfad hat das Mitglied keine Aktion mehr — das PDF darf
beim Submit-Zeitpunkt nicht raus, weil noch keine Mitgliedsnummer als
Mandatsreferenz vorhanden ist (Mandat wäre unvollständig). Drei Schichten
erzwingen das: Migration-Backfill, Backend-Validation (400 bei
Unstimmigkeit), Frontend-UI (Timing-Toggle wird auto-aktiviert + disabled,
wenn Audit aktiv).

**Verhaltens-Matrix (3 gültige Kombinationen):**

| CORE-Audit | Timing | Verhalten |
|---|---|---|
| aus | aus | PDF beim Submit, mit Unterschriftenfeld, Mandatsreferenz bleibt leer (Platzhalter „wird von der EEG ausgefüllt"). Mitglied unterschreibt + sendet zurück; EEG trägt die Mandatsreferenz später nach. |
| aus | an | PDF erst beim Import, mit Unterschriftenfeld, Mandatsreferenz = Mitgliedsnummer. Mitglied unterschreibt + sendet zurück. |
| an | an | PDF erst beim Import, mit Audit-Trail-Block, Mandatsreferenz = Mitgliedsnummer. Mitglied muss nichts mehr tun. |

**EEG-Kopie des SEPA-Mandat-PDF bei Audit-Pfad:** bei aktivem CORE-Audit-
Toggle (resp. B2B-Audit-Toggle) bekommt die EEG-Kontaktadresse eine
separate Ablage-Kopie der Mandat-PDF — Mitglied muss nichts zurücksenden,
EEG hätte sonst keinen Beleg. Subject: „Ablage-Kopie: SEPA-Mandat — {Name},
Antrag {Referenznummer}". Best-effort (Member-Mail ist sync hart-fail,
EEG-Kopie ist nachgelagert und blockiert den Import nicht). Bei
Klassik-Pfad keine EEG-Kopie (Mitglied sendet Original zurück).

**Backend:**
- Migration 000071: Backfill `UPDATE … SET sepa_mandate_core_audit_enabled = TRUE, sepa_mandate_at_import = TRUE WHERE sepa_mandate_enabled = FALSE`, dann `ALTER TABLE … DROP COLUMN sepa_mandate_enabled`
- `shared.RegistrationEntrypoint.SEPAMandateEnabled` entfernt
- `registration_entrypoint_repo.go` SELECT + `SaveEEGSettings`-Signatur ohne den Parameter
- `registration_entrypoint_repo_tx.go` `EEGSettingsForImport.SEPAMandateEnabled` raus
- `application_service.go` `buildSEPAMandateData` CORE-Pfad ohne Toggle-Check (nur noch Stammdaten-Check)
- `mail/service.go`:
  - `ResolveSepaMandateType` ohne den Toggle (nur noch `kein_sepa`/`!Accepted`/`b2b`/`core`-Vier-Fall-Logik)
  - `memberTemplateData.SEPAMandateEnabled` raus, Template `application_submitted_member.html` vereinfacht
  - `SendMandateAtImportNotification`: EEG-Kopie nur bei aktivem Audit-Toggle der jeweiligen einzugsart; Subject „Ablage-Kopie"; best-effort
  - `buildActivationData` Hint-Logik ohne den Toggle
- `http/admin.go`:
  - GET-Response-Map + PUT-Body-Struct ohne `sepaMandateEnabled`
  - Cross-Field-Validation: `sepaMandateCoreAuditEnabled && !sepaMandateAtImport` → 400 mit Feld-Hinweis
- `pdf/approval_pdf.go` `SEPAMandateEnabled` raus, Zustimmungs-Liste vereinfacht
- `configexport`: Schema-Feld `LegacySEPAMandateEnabled *bool, omitempty` (Pattern wie PROJ-73 `LegacyUseCompanySEPAMandate`); Exporter setzt nichts; Importer ignoriert und loggt `slog.Info`; Diff zeigt das Feld nicht mehr
- `resync_service.go` `!entrypoint.SEPAMandateEnabled`-Guard entfällt
- `registration_service.go` + `shared.RegistrationConfig` ohne `SEPAMandateEnabled`-Public-Field

**Frontend:**
- `EEGSettings.sepaMandateEnabled` + `EEGSettingsSavePayload.sepaMandateEnabled` + `RegistrationConfig.sepaMandateEnabled` raus
- `admin-eeg-settings-editor.tsx`:
  - Toggle-Block + State + Snapshot + Save-Payload + reload/discard ohne `sepaMandateEnabled`
  - Warn-Banner vereinfacht (kein Toggle-abhängiger Wortlaut mehr)
  - CORE-Audit-Toggle + Timing-Toggle aus dem `{isAdvanced && sepaMandateEnabled && …}`-Block in ein eigenständiges `{isAdvanced && …}` herausgezogen
  - Cross-Field-Coupling im UI: CORE-Audit aktivieren → Timing-Toggle wird auto-gesetzt + disabled, mit Erklärungs-Popover
  - Timing-Toggle umbenannt: „SEPA-Mandat erst beim Import senden (Mandatsreferenz = Mitgliedsnummer)"
  - Kurz-Erklärungen unter allen drei SEPA-Toggles (Konsistenz mit anderen Settings — Tester-Bitte 2026-06-08)
- `admin-legal-documents-editor.tsx` Save-Payload-Anpassung
- `registration-form.tsx`:
  - `buildFormSchema`-Signatur ohne den Parameter; Online-Zustimmung-Validation immer aktiv
  - Hinweis-Absatz bei Timing-Toggle ohne `sepaMandateEnabled`-Vorbedingung
  - Online-Zustimmung-Checkbox immer sichtbar (vorher conditional auf `!sepaMandateEnabled`)
  - **Label-Wechsel** (Tester-Bitte 2026-06-08): „Kontoinhaber:in *" → **„Kontowortlaut *"** mit neuem Hint-Popover (Erklärung gemeinsame Haushaltskonten). Hintergrund: SEPA-Lastschriften wurden abgelehnt, weil bei gemeinsamen Konten nur ein Name eingetragen war
- `settings-mode.test.ts` ohne `sepaMandateEnabled`-Property

**Doku:**
- `docs/api-spec.md` GET/PUT-Beispiele + Beschreibung ohne `sepaMandateEnabled`, mit Coupling-Erklärung
- `docs/domain-model.md` `sepa_mandate_enabled` als „entfernt durch PROJ-80" markiert mit Backfill-Beschreibung
- `docs/user-guide/06-admin-settings.md` SEPA-Sektion umgeschrieben (Drei-Permutationen-Matrix statt Vier; neue Konstanten; PROJ-frei)
- `docs/user-guide/changelog.md` 2026-06-08-Eintrag

**Tests:**
- `internal/application/application_service_test.go`:
  - `baseEntrypoint`-Helper ohne Parameter
  - `TestBuildSEPAMandateData_ReturnsNilWhenDisabled` entfällt
  - `TestBuildSEPAMandateData_Core_StillRequiresSEPAMandateEnabled` ersetzt durch `TestBuildSEPAMandateData_PROJ80_CoreAlwaysReturnsMandate` (positiver Test)
  - alle `baseEntrypoint(true|false)`-Aufrufe auf `baseEntrypoint()` umgestellt
- `internal/mail/service_test.go`:
  - `ResolveSepaMandateType`-Tests ohne `SEPAMandateEnabled`; `_OnlineConsentOnly`-Test entfällt
  - `BuildActivationData`-Tests umgebaut; `_OnlineConsent_KeinMandateHint` entfällt

### Feature — PROJ-78: Toggle „Elektronisches SEPA-Mandat" (B2B + CORE separat) *(2026-06-07)*

Zwei unabhängige Per-EEG-Schalter steuern, ob das SEPA-Mandat-PDF den
elektronischen Audit-Trail-Block (§ 76 (3) EIWOG 2010) oder den klassischen
Datum/Unterschrift-Block rendert — getrennt für Basislastschrift (CORE)
und Firmenlastschrift (B2B). **Default beide FALSE** (klassische Variante),
weil die Rechtsbewertung der elektronischen Willenserklärung derzeit
geklärt wird; bis dahin soll kein EEG unbemerkt in die fragliche Variante
laufen. Die heute mittag mit PROJ-77 deployte B2B-Audit-Variante ist
damit bis zum aktiven Opt-in eines EEG faktisch stillgelegt.

**Toggles getrennt**, weil die Rechtsbewertung für Geschäftsleute (B2B)
anders ausfallen kann als für Verbraucher (CORE). Ein EEG könnte z. B.
B2B-Audit zulassen, weil hier ohnehin höhere Sorgfaltsannahme gilt,
während CORE weiterhin physisch unterschrieben werden muss.

**Backend:**
- Migration 000070: zwei neue Spalten `registration_entrypoint.sepa_mandate_core_audit_enabled BOOLEAN NOT NULL DEFAULT FALSE` und `…b2b_audit_enabled BOOLEAN NOT NULL DEFAULT FALSE` (eine kombinierte Migration, beide Spalten zusammen)
- Audit-Block-Render-Logik aus `pdf.GenerateCompany` in einen geteilten Helper `renderSEPAAuditBlock` extrahiert; klassischer Unterschriftsblock bleibt inline pro Generator (B2B + CORE haben unterschiedliche Layouts)
- CORE-SEPA-PDF (`pdf.Generate`) erhält den Audit-Block-Pfad neu — selbe Wortlaut-Variante wie B2B, mit derselben Gate-Bedingung `Toggle && AuditDaten vollständig`
- `buildSEPAMandateData` wählt anhand `app.Einzugsart` den passenden Toggle aus dem EEG und übergibt das Ergebnis als `SEPAMandateData.ElectronicMandateEnabled` an den Renderer
- Renderer-Check (Short-Circuit): `if data.ElectronicMandateEnabled && AuditTenant != "" && !AcceptedAt.IsZero() && AuditIP != ""`
- `SaveEEGSettings`-Signatur um beide Booleans erweitert; `PUT /api/admin/settings/eeg` akzeptiert die neuen Felder; `GET /api/admin/settings/eeg` liefert sie zurück
- Configexport-Schema erweitert (`*bool, omitempty`); Importer setzt nil → FALSE (konservativer Default für Pre-PROJ-78-Bundles); Diff zeigt beide Felder als separate Zeilen
- Resend (PROJ-70) und Activate-Mail-Trigger (PROJ-46/53) übernehmen automatisch den aktuellen Toggle-Stand der EEG — Toggle ist EEG-Policy, kein Antrags-Snapshot

**Frontend:**
- Zwei neue Toggles im Admin-Settings-Editor, jeweils im **Advanced-Modus** sichtbar:
  - **CORE-Audit-Toggle** im bestehenden `{isAdvanced && sepaMandateEnabled && (…)}`-Block neben „SEPA-Mandat erst beim Import senden" — nur sinnvoll wenn CORE-PDF überhaupt erzeugt wird
  - **B2B-Audit-Toggle** in einem **separaten** `{isAdvanced && (…)}`-Block direkt darunter — **NICHT** an `sepaMandateEnabled` gekoppelt (B2B-Mandate-PDFs werden unabhängig vom CORE-Toggle erzeugt, PROJ-74)
- Hint-Popover je Toggle erklärt § 76 (3) EIWOG-Hintergrund und die Wahlfreiheit pro Mandat-Typ
- `isAdvancedEEGSettingsActive` triggert auf jeden TRUE-Wert eines der beiden Toggles
- TypeScript-Typen: `EEGSettings.sepaMandateCoreAuditEnabled?`, `EEGSettings.sepaMandateB2BAuditEnabled?`, `EEGSettingsSavePayload.sepaMandateCoreAuditEnabled` (required), `…B2BAuditEnabled` (required)

**Tests:**
- PDF-Snapshot-Tests (`internal/pdf/generator_test.go`):
  - PROJ-77 B2B-Tests um `ElectronicMandateEnabled=true` ergänzt (Regression-Schutz)
  - NEU `Generate_AuditBlock_RenderedWhenAllFieldsSet` (CORE-Pendant)
  - NEU `Generate_AuditBlock_FallbackOnMissingIP` (CORE-Pendant)
  - NEU `Generate_AuditBlock_VerifyVerbToggle` (CORE-Pendant)
  - NEU `Generate_AuditBlock_IPv6` (CORE-Pendant)
  - NEU `GenerateCompany_ToggleFalse_FallsBackToClassic` (B2B-Toggle-Übersteuerung)
  - UMGEBAUT `Generate_ToggleFalse_IgnoresAuditFields` (ersetzt PROJ-77 `Generate_Core_IgnoresAuditFields`)
- Service-Layer-Tabellen-Test `TestBuildSEPAMandateData_PROJ78_ToggleMapping` mit 6 Permutationen `{core,b2b} × CoreToggle × B2BToggle`
- Frontend-Unit-Tests in `settings-mode.test.ts` für CORE-Toggle, B2B-Toggle und beide-Toggles-gleichzeitig

**Owner-Entscheidungen festgehalten:**
- Default beide FALSE für ALLE EEGs (Test-Betrieb erlaubt abrupten Cut; keine Backward-Compat-Sonderlogik für PROJ-77-Bestands-Audits)
- Spalten-Namenskonvention `sepa_mandate_*_audit_enabled` (innerhalb der `sepa_mandate_*`-Gruppe; nicht `electronic_*`)
- Service-Layer wählt pro `einzugsart`; Renderer bleibt mandatstyp-agnostisch
- Audit-Render-Helper geteilt, klassischer Block inline pro Generator (Layout-Unterschiede CORE/B2B)
- Service-Layer-Test als Tabelle, PDF-Tests als individuelle Funktionen mit klaren Failure-Lokationen

### Feature — PROJ-77: B2B-Mandat-Audit-Block (§ 76 (3) EIWOG 2010) *(2026-06-07)*

Im SEPA-Firmenlastschrift-Mandat-PDF (`einzugsart=b2b`) ersetzt ein
Audit-Trail-Text den klassischen Datum/Unterschrift-Block. Der Text
dokumentiert die elektronische SEPA-Zustimmung als formfreie
Willenserklärung gemäß § 76 (3) EIWOG 2010 — das Mitglied muss nicht
mehr physisch unterschreiben.

**Audit-Text** mit drei Platzhaltern:
- EEG-Name (`registration_entrypoint.eeg_name`)
- Zustimmungs-Zeitpunkt (`application.sepa_mandate_accepted_at`)
- IP-Adresse (`application.sepa_mandate_accepted_ip`, neu)

**Wortlaut-Variante** je nach `require_email_confirmation` der EEG:
„nach Verifizierung" (Bestätigungs-Link aktiv) vs. „nach Eingabe"
(ohne Bestätigung).

**Backend:**
- Migration 000069: neue Spalte `application.sepa_mandate_accepted_ip INET NULL`
- Public-Submit erfasst IP via bestehender `realIP`-Middleware
- Externe API erweitert um optionalen `submitterIp`-Body-Param
  (Server-zu-Server-Pattern: EEG-Integrator gibt End-User-IP mit)
- Validierung via `net.ParseIP`; ungültig → 400 mit Feld-Hinweis
- B2B-PDF-Renderer (`GenerateCompany`): Variant-Switch mit Kopfzeile
  „Elektronisch erteiltes Mandat (gem. § 76 (3) EIWOG 2010)" und
  `MultiCell`-Auto-Umbruch für IPv6-Adressen
- `UpdateTx` schützt die IP via `COALESCE` (Erst-Submit gewinnt)
- ResetImport behält den Audit-Trail (kein Cleanup)
- PROJ-70-Resync rendert das PDF mit der ursprünglichen Member-IP

**Excel-Export:** neues Field `sepa_mandate_accepted_ip` registriert;
Default per EEG-FieldConfig-Pattern hidden (DSGVO-minimal).

**Datenschutz-Erklärung** (`/datenschutz`) um expliziten EIWOG-Hinweis
ergänzt.

**Backward-Compat:** Bestandsanträge ohne IP fallen auf den klassischen
Datum/Unterschrift-Block zurück. Core-Mandat-PDF (`einzugsart=core`)
bleibt vollständig unverändert — Audit-Block ist B2B-spezifisch.

**Tests:** 5 neue PDF-Snapshot-Tests (Audit gerendert, IPv6, langer
Tenant-Name, Wortlaut-Wechsel, Fallback, Core-Regression). Alle 12
Pakete grün.

### Feature — PROJ-76: Vorstands-Genehmigungs-Workflow für Beitrittserklärung *(2026-06-07)*

Per-EEG-Toggle, der den Aktivierungs-Mail-Pfad umstellt. Hintergrund:
manche EEGs wollen die Beitrittsbestätigung nicht automatisch ans Mitglied
versenden, sondern durch den Vorstand formell genehmigen lassen
(Statuten-Anforderung Aufnahmebeschluss, Bedarf an unterschriebener
Bestätigung).

**Bei aktivem Toggle:**
- Beim Wechsel auf „Aktiviert" entfällt die Member-Beitrittsbestätigungs-
  Mail über die Plattform komplett.
- Stattdessen geht eine **Beitrittserklärung** (mit Vorstands-
  Signaturblock am Ende) an den EEG-Kontakt. Vorstand unterschreibt
  manuell und leitet ans Mitglied weiter.
- Das Mitglied wird über die reguläre eegFaktura-Core-Aktivierungs-Mail
  über seinen Status informiert (Core-Pfad, läuft unabhängig).
- **Sync hard-fail vor Commit**: bei fehlendem `contact_email` oder
  SMTP-Outage rollt der Status-Wechsel zurück, der Antrag bleibt im
  vorherigen Status. Admin sieht aussagekräftige Fehlermeldung.

**Download-Knopf im Antrags-Detail:** ein blauer Info-Block mit
Versand-Datum + „Beitrittserklärung herunterladen" macht das PDF
jederzeit on-demand verfügbar. Nützlich, wenn der Vorstand das
Dokument verlegt hat.

**Technisches:**
- Single PDF-Renderer mit Variant-Parameter (kein Code-Duplikat).
- Separate Spalte `application.board_declaration_sent_at` neben
  `activation_notification_sent_at` — saubere semantische Trennung der
  Mail-Events; bei ResetImport werden beide Spalten synchron auf NULL
  gesetzt.
- Bug-Fix als Beifang: `activation_notification_sent_at` war im
  ResetImport-Pfad bisher **nicht** zurückgesetzt — Re-Aktivierungen im
  Auto-Modus hätten keine Mail mehr versandt. Mit PROJ-76 mitgefixed.
- Toggle ist Teil des PROJ-67-Awareness-Triggers (Erweitert-Modus).
- Configexport (PROJ-61) toleriert Pre-PROJ-76-Bundles (Schema-Feld
  ist `*bool, omitempty`).
- Migrations 000067 + 000068 (zwei `ALTER ADD COLUMN`, beide
  nicht-blocking auf PostgreSQL).

PROJ-65 (Vorstands-Signaturblock im bestehenden PDF, Planned) wird
durch PROJ-76 vollständig abgedeckt und ist auf **Superseded** gesetzt.

### UX — PROJ-75: SEPA-Einwilligungs-Checkbox in der Bankverbindungs-Card *(2026-06-06)*

Tester-Wunsch 2026-06-06: Die SEPA-Einwilligungs-Checkbox im öffentlichen
Anmeldeformular saß bisher im allgemeinen Einwilligungsblock — weit weg
von den Konto-Eingabefeldern. Neuer Platz: direkt unter den
Eingabefeldern IBAN/Kontoinhaber:in/Bankname in der Bankverbindungs-Card.

Außerdem zeigt der neue Text den konkreten EEG-Namen und die Creditor-ID,
die aus dem Public-Registration-Config-Payload kommen (Backend liefert
`eegName` und `creditorId` neu im
`GET /api/public/registration/{rc_number}`-Endpoint).

Fallback-Verhalten: wenn EEG noch keinen PROJ-32-Sync gemacht hat,
greift ein generischer Text ohne Namen; die Creditor-Zeile wird
ausgeblendet, wenn die ID leer ist. Sichtbarkeits-Bedingung der Checkbox
bleibt unverändert (`sepaMandateEnabled=false` = Online-Zustimmungs-
Lösung).

### Fix — PROJ-74: B2B-Mandate trotz `SEPAMandateEnabled=false` + Hart-Fail-Schutz *(2026-06-06)*

Aufgedeckt durch Tester-Befund 2026-06-06: bei `SEPAMandateEnabled=false`
sperrte `buildSEPAMandateData` ALLE PDF-Generierungspfade — auch B2B.
Das war für Core korrekt (Online-Zustimmung als Fallback), für B2B aber
ein rechtswidriger Zustand: SEPA-Regelwerk erlaubt für die Firmenlastschrift
keine reine Online-Zustimmung.

Gate-Logik in `buildSEPAMandateData`:
- `einzugsart="b2b"` → Toggle wird ignoriert, nur Stammdaten-Vollständigkeit zählt.
- `einzugsart="core"` → heutiges Verhalten unverändert (Toggle muss aktiv sein).
- `einzugsart="kein_sepa"` oder leer → kein Mandat.

`MissingMandateFields` ist seit dem Fix ein reiner Daten-Check (frühere
Frühe-Rückgabe bei `!SEPAMandateEnabled` entfällt) und ist als exportierte
Helper-Funktion auch im `importing`-Package verfügbar.

**Hart-Fail im Pre-Import-Check** (Status-Wechsel `approved → imported`):
fehlen für einen B2B-Antrag EEG-Stammdaten, lehnt
`POST /api/admin/applications/{id}/import` den Status-Wechsel mit
`409 Conflict` ab und nennt die fehlenden Felder. Der Antrag bleibt im
vorherigen Status, bis die Stammdaten gepflegt sind. Für Core-Anträge
bleibt es bei Skip+Warn (Online-Zustimmung als Fallback). Der Resync-Pfad
(PROJ-70) erhält dieselbe Hart-Fail-Logik mit präziserer Fehlermeldung.

**Settings-UI-Klarstellung:**
- Hint-Popover am Toggle „SEPA-Mandat von der EEG bereitstellen": klärt,
  dass der Schalter nur Core-Mandate (Privat) steuert und B2B-Mandate
  unabhängig davon erzeugt werden.
- Hint-Popover am Toggle „SEPA-Mandat erst beim Import senden": klärt,
  dass das Versand-Timing nur Core-Anträge betrifft.
- Warn-Banner „Stammdaten unvollständig" erscheint künftig auch bei
  `SEPAMandateEnabled=false`, sofern Pflichtfelder fehlen — mit
  Konjunktiv-Text „Falls Sie B2B-Anträge bearbeiten…".

### Cleanup — PROJ-73: Verwaisten EEG-Toggle `use_company_sepa_mandate` entfernt *(2026-06-06)*

PROJ-14 hatte den Toggle eingeführt, um pro EEG zu entscheiden, ob
Unternehmen/Vereine das B2B-Mandat-PDF statt des Core-Mandats erhalten.
PROJ-48 ersetzte diese Auto-Mapping-Logik durch das per-Antrag-
`einzugsart`-Modell — der EEG-Toggle blieb seither funktionslos im
Settings-UI und verwirrte Admins, die ihn umlegten und beobachteten,
dass nichts passierte.

**Migration 000066** droppt `registration_entrypoint.use_company_sepa_mandate`.
Bestehende `=true`-Werte gehen verloren — verlustfrei, weil der Toggle
keine Domain-Wirkung hatte. Der Schalter ist aus dem Settings-UI und
allen Backend-/Frontend-Schichten entfernt. Backup-Imports aus
PROJ-61 (Config-Export-Bundles) tolerieren das alte Feld weiterhin —
es wird beim Decoder ignoriert. `PROJ-14`-Spec auf **Superseded**
markiert mit Verweis auf das aktuelle `einzugsart`-Modell.

### Feature — PROJ-64: Faktura-Handover-Billing-Trigger *(2026-05-29)*

Schließt die Lücke zwischen dem geplanten Quartals-Verrechnungs-Modell
(zählt neu an eegFaktura übergebene Anträge) und dem Excel-Export, der
1:1 das eegFaktura-Import-Template liefert. Ein Admin, der das xlsx
nutzt statt unseren `POST /import`-Endpoint, wäre bisher nicht
verrechnet worden.

Neue Spalte `application.faktura_handover_at TIMESTAMPTZ` (Migration
`000057_add_faktura_handover_at`). Wird vom ersten der beiden Wege
gesetzt — erfolgreicher Core-Import ODER Download des Faktura-Format-
Excel — und ist idempotent (`SetFakturaHandoverAtIfEmpty`). Spätere
Downloads oder Re-Imports lassen den Wert unverändert; jeder Antrag
zählt im Billing maximal einmal. Backfill in der Migration: bestehende
`imported_at IS NOT NULL`-Anträge bekommen denselben Timestamp, damit
Pre-PROJ-64-Imports nicht versehentlich als neu billbar gewertet
werden. Partial-Index `idx_application_faktura_handover_at` optimiert
die geplante Quartals-Billing-Query.

Code-Pfade:

- `internal/importing/import_service.go::Import()` setzt
  `faktura_handover_at = importFinishedAt` nach erfolgreichem Core-POST.
  Best-effort: ein Persist-Fehler bricht den Import nicht ab — ein
  späterer Trigger holt den Wert nach.
- `internal/application/admin_service.go::ExportApplicationExcel()` setzt
  `faktura_handover_at = NOW()` nach erfolgreicher xlsx-Generierung,
  bevor die Datei an den Caller geht.

`imported_at` bleibt für die Status-Logik (Reset-Erkennung, Audit,
Mail-Templating) unverändert; es spiegelt nur noch den Onboarding-
Workflow, nicht mehr den Billing-Trigger. Die geplante Quartals-Cron
schwenkt von `imported_at IS NOT NULL` auf
`faktura_handover_at IS NOT NULL`.

Frontend (`admin-application-detail.tsx`):

- Marker-Zeile im Detail-Header („An eegFaktura übergeben am …, für
  die Verrechnung berücksichtigt"), sichtbar wenn `fakturaHandoverAt`
  gesetzt ist.
- Confirmation-AlertDialog vor dem ERSTEN Excel-Download. Klärt den
  Admin auf, dass der Download als Übergabe vermerkt wird und
  verweist auf die Datenweiterleitung (PROJ-60) als nicht-billbare
  Alternative für reine Backup-/Audit-Use-Cases. Bei wiederholten
  Downloads (`fakturaHandoverAt != null`) wird der Dialog
  übersprungen.

Specs: `features/PROJ-64-faktura-handover-billing-trigger.md`.

### Feature — `approved → rejected` Transition *(2026-05-29)*

Tester-Wunsch: „Ein Mitglied das genehmigt wurde kann ich ja nicht löschen!
Kann man es einbauen das wenn ein Import zurückgesetzt wurde die Option
Ablehnen angeboten wird?" — Genau. Vorher hatte `adminTransitions` gar keinen
`approved`-Key auf der Ausgangsseite; nach einem `POST /reset-import` saß der
Antrag in `approved` ohne Möglichkeit, ihn final abzulehnen.

Neu in `internal/application/admin_service.go::adminTransitions`:
`approved → rejected` mit Pflicht-Grund (greift den vorhandenen
`requiresReason`-Hook ab). `member_number` muss nicht gesondert geleert
werden — `ResetImportTx` nullt sie schon beim Reset, und vor dem ersten
Import ist sie ohnehin NULL.

Frontend: `admin-status-actions.tsx` zeigt im `approved`-Block jetzt einen
destructive „Ablehnen"-Button neben „In eegFaktura importieren" und
„Manuell aktivieren …". Klick öffnet den bestehenden Rejection-Dialog mit
Pflicht-Grund.

Vier Regression-Guards in `application_service_test.go::TestAdminTransition_*`
decken `approved → rejected` (erlaubt), `approved → approved` (verboten,
keine versehentliche Self-Transition), `approved → imported` (bleibt dem
dedizierten Import-Endpoint vorbehalten) und `rejected → *` (terminal) ab.

### Fix — Import-Pfad: Mandatsreferenz + Mandatsdatum fehlten im Core *(2026-05-28)*

Tester-Befund: „Sepa Daten muss man aber in EEGFaktura Händisch nachtragen. Is
aber so gedacht oder?" Antwort: nein, war ein Bug. Bei den at-import-Mandat-
Pfaden (`einzugsart=b2b` und `einzugsart=core` mit `sepa_mandate_at_import=true`)
hat das Onboarding die Mandatsreferenz aus der Mitgliedsnummer und das Datum
aus dem Importzeitpunkt zwar korrekt für das lokale SEPA-Mandat-PDF abgeleitet,
aber **erst nach** dem `POST /participant`-Call. Im Core landeten leere Werte;
der Admin musste sie händisch in eegFaktura nachtragen.

Reihenfolge vorher:

1. `Import()` baut Payload aus DB-Stand → mandate_reference + mandate_date sind NULL.
2. Core erhält das Payload mit leerem `accountInfo.mandateReference` / `mandateDate`.
3. Async-Goroutine `SendPostImportNotification` setzt **danach** die Werte lokal,
   baut das PDF, schickt die Mail. Keine Re-Sync in den Core.

Fix:

- Neuer Helper `shouldDeriveMandateAtImport(app, ep)` in
  `internal/importing/import_service.go` zentralisiert das Gate. Trigger ist
  `(einzugsart=b2b OR core+SEPAMandateAtImport)` und `SepaMandateAccepted=true`.
- `Import()` ruft das Gate jetzt VOR `BuildPayload`. Bei Treffer werden
  mandate_reference (= MemberNumber) und mandate_date (= `importStartedAt`)
  via `SetMandateReferenceIfEmpty` und neuem `SetMandateDateIfEmpty`
  persistiert und in den in-memory `app` gespiegelt — `BuildPayload` reicht sie
  durch in `accountInfo`.
- `SetMandateDateIfEmpty` ist neu; bisheriges `SetMandateDate` setzte unbedingt
  `NOW()` und hätte den Import-Wert sofort wieder überschrieben.
  `SendPostImportNotification` nutzt jetzt die IfEmpty-Variante → idempotent.
- Submit-Pfad (`core + sepa_mandate_at_import=false`, PROJ-12) bleibt unverändert:
  Der Member bekommt die Referenz erst beim activated-Übergang via
  Hinweis-Block in der Beitrittsbestätigungs-Mail kommuniziert; persistiert
  wird sie erst, wenn der signierte Papier-Mandat zurückkommt und der Admin
  sie selbst einträgt. Das ist by-design.

Bestandsanträge, die VOR diesem Fix importiert wurden: die Werte sind in der
Onboarding-DB, fehlen aber im Core. Korrektur entweder via `reset-import +
re-import` (überschreibt den Core-Datensatz vollständig), oder durch
manuelles Eintragen im eegFaktura-Frontend.

Fünf Regression-Guards in `payload_test.go::TestShouldDeriveMandateAtImport_*`
decken alle Trigger-Kombinationen ab.

### Fix — Admin-Edit: „Zusatzangaben"-Karte fehlte komplett *(2026-05-28)*

Tester-Befund: „beim editieren in Admin kann man Zusatzdaten — Beitritts-
Datum als Admin nicht bearbeiten?" Korrekt — und nicht nur das
Beitrittsdatum: die gesamte Zusatzangaben-Karte (membership_start_date,
persons_in_household, heat_pump, electric_vehicle inkl. Count/Km,
electric_hot_water, cooperative_shares_count, network_operator_authorization)
war in der Admin-Detail-Ansicht sichtbar, aber im Edit-Formular gar nicht
vorhanden. Backend-DTO (`AdminUpdateApplicationRequest`) hatte die Felder
ebenfalls nicht; `UpdateAdminTx` schrieb sie nicht in die DB. Drei Schichten
hatten denselben blinden Fleck.

Follow-up am gleichen Tag: Erstversion zeigte ALLE Felder unabhängig von
der EEG-Field-Config. Tester hat zurecht reklamiert, dass eine EEG mit
`membership_start_date = hidden` das Feld im Edit-Dialog trotzdem sah —
inkonsistent zum Public-Form. Folgekorrektur: AdminEditForm zieht jetzt
`getRegistrationConfig(rcNumber)` und rendert ein Feld nur, wenn dessen
state ≠ `hidden` ist. `admin_only` zählt als sichtbar (das ist genau der
Sinn dieses States: nur im Admin editierbar). Hidden-Felder werden im
Payload nicht mitgesendet — Bestandswerte bleiben erhalten.

Fix:

- **Backend** (`internal/shared/requests.go`,
  `internal/application/admin_service.go`,
  `internal/application/application_repo.go`): Felder zum DTO ergänzt;
  Pointer-Sentinel-Semantik (omittet = unverändert, explizit = setzen).
  `AdminUpdateApplication` ruft jetzt `clearEVDetailsIfDisabled(app)` wie
  der Public-Update-Pfad, sodass E-Auto-Count/Km serverseitig genullt
  werden, wenn der Toggle aus ist. `network_operator_authorization_at`
  wird wie im Public-Pfad nur beim First-Grant-Übergang gesetzt
  (false→true), nicht beim Wieder-Entfernen — Audit-Spur bleibt.
  `UpdateAdminTx` schreibt jetzt 11 zusätzliche Spalten.

- **Frontend** (`src/lib/api.ts`, `src/components/admin-edit-form.tsx`):
  TS-Typen ergänzt; neue „Zusatzangaben"-Section zwischen Adresse und
  Zählpunkte mit Date-Input, Number-Inputs und Checkboxes. E-Auto-Sub-
  Felder werden nur gerendert, wenn der Toggle aktiv ist. Popover-Hinweis
  zur Audit-Semantik der Netzbetreiber-Vollmacht.

Bekannter pre-existing Gap (nicht in diesem Fix gelöst):
`cooperative_shares_count` wird auch vom Public-Update-Pfad
(`UpdateApplication`/`UpdateTx`) nicht geschrieben — nur Create + (jetzt)
Admin-Update setzen ihn. Wenn ein Member im `needs_info`-Status seine
Anteilszahl ändern soll, müsste das auch dort nachgezogen werden.

### Fix — Aktivierungs-Mail: Mandatsreferenz-Hinweis fälschlich bei Online-Zustimmung *(2026-05-28)*

Tester-Befund: Nach Status „aktiv" bekam ein Mitglied die Beitritts-
bestätigungs-Mail mit dem Hinweis-Block „SEPA-Lastschriftmandat —
Mandatsreferenz: 395. Bitte ergänze diese auf dem Mandatsformular, das
wir dir bei der Eingangsbestätigung zugeschickt haben." — obwohl die
EEG keinerlei Papier-Mandat ausstellt (`sepa_mandate_enabled = false`)
und das Mitglied online zugestimmt hatte. Es gibt also gar kein
Mandatsformular, auf dem etwas zu ergänzen wäre.

Root cause in `internal/mail/service.go::buildActivationData`: die
Hint-Gate vergaß den `SEPAMandateEnabled`-Check und triggerte für
jede Submit-Pfad-Mandatsannahme im `einzugsart=core`-Fall.

Fix: zusätzliches Gate `ep.SEPAMandateEnabled` ergänzt. Der Hinweis
erscheint nun nur noch, wenn (a) Mandat erteilt, (b) die EEG ein
Papier-Mandat-PDF ausstellt, (c) es per Submit-Pfad ohne Referenz
versendet wurde, (d) jetzt eine Mitgliedsnummer als Referenz vergeben
ist. Drei neue Regression-Guards in `service_test.go` decken die
Online-Consent-, Papier-Mandat- und AtImport-Variante ab.

### Fix — PROJ-61 Bundle-Import: UI-Refresh nach Apply *(2026-05-27)*

Tester-Befund: ein Bundle-Import schrieb 30 fieldConfig-Einträge
sauber in die DB (Backend-Log bestätigt: `input=30 inserted=30`),
aber die Settings-Page zeigte weiter die alten Werte —
„jungfräuliche Formularfelder", obwohl die DB den neuen Stand hatte.

Root cause war rein UI-seitig: `settings/page.tsx` lud die FieldConfig
nur einmal beim EEG-Select (`useEffect [selectedRc, accessToken]`).
Nach einem Apply gab es keinen Refresh-Trigger; offene Tabs hielten
ihren bereits geladenen Pre-Apply-Snapshot fest. Dass der Tester
Einleitungstext + Rechtsdokumente als „übernommen" sah, lag
vermutlich daran, dass er diese Tabs zum ersten Mal nach dem Apply
öffnete — die Fetches passierten beim Tab-Mount mit den neuen Daten.

Fix in `src/app/admin/settings/page.tsx`:
- Neuer `applyEpoch`-Counter (init 0), nach Apply um 1 erhöht.
- `useEffect`-Dependency für fieldConfig erweitert um `applyEpoch`
  → Re-Fetch ohne Tab-Wechsel.
- Jede Tab-Inhalt-Komponente (`AdminEEGSettingsEditor`,
  `AdminIntroTextEditor`, `AdminFieldConfigEditor`,
  `AdminLegalDocumentsEditor`, `DataExportSection`) bekommt
  `key={'<tag>-${applyEpoch}'}` → React remounted nach Apply, jeder
  Sub-Editor lädt seine Daten frisch.
- `ConfigImportExportSection` erweitert um optionalen `onApplied`-
  Callback, von `handleApplied` nach Toast aufgerufen.

Backend-Diagnose-Logging aus `6c9fea2` bleibt drin — hat hier den
entscheidenden Hinweis gegeben („persisted: 30") und ist auch für
zukünftige Drift-Befunde gegen `knownConfigurableFields` nützlich.

### Fix — Tester-Feedback-Bundle 2026-05-27

Vier Befunde aus dem nächsten Test-Lauf, alle wieder zurück auf das
Anti-Pattern „parallele Code-Pfade die in Sync bleiben sollten"
(Memory `feedback_shared_helpers_for_parallel_paths`).

**1. „Verbrauch Prognose" verschwand nach Aktivierung im Admin-Tool.**
Tatsächliche Ursache: jede Admin-Edit-Speicherung wischte stillschweigend
**neun** Meter-Felder. `AdminUpdateApplication` baute ein Meter-Struct-
Literal ohne PROJ-45-Felder (GenerationType, BatterySizeKwh,
InverterManufacturer, InverterPowerKw) UND ohne PROJ-49-Energie-Felder
(Verbrauch Vorjahr/Prognose, Einspeisung Prognose, PV-Leistung,
Einspeiselimit + Wert, Speichersteuerung-akzeptabel). Da `CreateBulkTx`
die Meter komplett ersetzt, war ein einziger Save genug. Dieselbe
Lücke im public `UpdateApplication` (etwas kleinere Felder-Schnittmenge).

**2. „Verbrauch Prognose" auch im Beitrittsbestätigungs-PDF weg.**
Gleiche Ursache wie 1 — der Wipe war in der DB. Nach Re-Edit erscheinen
die Werte wieder.

**3. Beitrittsbestätigungs-Mail wurde nach Aktivierung nicht versendet.**
Wenn die Aktivierung über den **Activation-Check-Batch** (PROJ-46 Stage D,
EEG-Mode `participant_active` oder `any_meter_registration_started`)
lief, transitionierte `ImportService.markActivated` nur den DB-Status
plus Status-Log — **rief `SendActivationNotification` nie auf**. Nur
der Admin-Klick-Pfad in `admin_service.go:643` machte den Send.

  Fix: `CheckActivations` liefert jetzt `ActivatedIDs` (internal-only,
  JSON-`-`). Der HTTP-Handler dispatcht `SendActivationNotification`
  per ID nach dem Batch-Lauf. Idempotent auf
  `activation_notification_sent_at`, also keine Doppel-Sends bei
  wiederholten Batches.

**4. PDF: „SEPA-Ermächtigung: Per E-Mail" obwohl online zugestimmt.**
`approvalSepaMandateType` und `resolveSepaMandateType` (zwei identische
Funktionen — das Anti-Pattern wieder) mappten `!SEPAMandateEnabled` auf
„Per E-Mail" ohne Rücksicht auf `SepaMandateAccepted`. Aber: wenn die
EEG kein Onboarding-PDF-Mandat anbietet UND das Mitglied die Online-
Checkbox angekreuzt hat, sollte „Online-Zustimmung erteilt" stehen.

  Fix: beide Duplikate ersetzt durch `mail.ResolveSepaMandateType` (eine
  Funktion, exported). Logik in der korrekten Reihenfolge:
  - `einzugsart=kein_sepa` → „Kein SEPA"
  - `!SepaMandateAccepted` → „Per E-Mail" (ausstehend)
  - `!SEPAMandateEnabled && Accepted` → „Online-Zustimmung erteilt" (NEU)
  - `Enabled && Accepted && b2b` → „Firmenlastschrift"
  - sonst → „Basislastschrift"

**Nachhaltige Prävention (zentraler Helper):** neuer
`BuildMeteringPointFromRequest(req, normalized, now)` in
`internal/application/application_service.go` ist jetzt die einzige
Stelle, an der ein `shared.MeteringPoint` aus dem Request gebaut wird.
Alle drei Pfade (CreateApplication, UpdateApplication,
AdminUpdateApplication) routen dadurch. Neue persistierte Meter-Felder
müssen genau einmal ergänzt werden; Drift ist strukturell unmöglich.
Analog `mail.ResolveSepaMandateType` für die SEPA-Variante.

Tests: 5 × `TestResolveSepaMandateType_*` decken jede Wire-Variante;
2 × `TestBuildMeteringPointFromRequest_*` sweepen alle persistierten
Felder (Regression-Guard) und prüfen den Participation-Factor-Default.

### Fix — Tester-Feedback-Bundle 2026-05-26

Drei eng verwandte Befunde aus dem Test-Feedback, alle ausgelöst von
parallelen Code-Pfaden, die dieselbe Datenstruktur bauen sollten und
mit der Zeit auseinanderdrifteten.

**1. Admin-Detail zeigte konfigurierbare Felder nicht.**
Beitrittsdatum, Personen im Haushalt, Wärmepumpe, E-Auto (+ Anzahl /
Jahres-km), Warmwasser elektrisch — die Mitglieds-Submit-Mail rendert
sie, das Admin-Detail-UI nicht. Backend liefert sie auch in der
Detail-Response (`shared.Application` mit json-Tags), aber das
TypeScript-`AdminApplicationDetail`-Interface listete sie nicht. Neue
„Zusatzangaben"-Karte rendert nur die Felder mit Werten (so bleibt
sie für EEGs ohne diese Konfiguration unsichtbar).

**2. Resend-Bestätigungsmail rendert „nur den Namen".**
`SendMemberConfirmation` (Admin-Button „Bestätigung erneut senden")
befüllte 12 von ~25 `memberTemplateData`-Feldern — die initiale
`SendSubmissionEmails` befüllt alle 25. Der Drift war über mehrere
PROJ-Iterationen entstanden.

  Fix: `buildMemberMailData(app, mps, ep, hasAttach, consents, url)`
  als Single-Source-of-Truth; beide Pfade routen dadurch. Neue Felder
  am Template müssen jetzt nur an einer Stelle ergänzt werden;
  zukünftiger Drift ist strukturell unmöglich.

  Signatur-Erweiterung: `SendMemberConfirmation(app, meteringPoints,
  entrypoint, consents)`. Das SEPA-PDF wird beim Resend bewusst NICHT
  neu generiert (würde eine neue Mandatsreferenz vergeben, was wir
  post-submit nie wollen). `ResendMemberConfirmation` in
  `admin_service.go` lädt MeteringPoints + Consents zusätzlich.

**3. Beitrittsbestätigung-PDF nicht mehr herunterladbar nach
Aktivierung.** Selber Befund wie Commit `3aa3444` (gleicher Tag) —
PROJ-46-Post-Import-States waren nicht in der Status-Allow-Map.
Bereits gefixt, wartet auf Helm-Upgrade.

**Erkenntnis (Memory `feedback_shared_helpers_for_parallel_paths.md`):**
Wenn zwei Code-Pfade dieselbe Datenstruktur bauen sollen, IMMER über
einen gemeinsamen Helper. Drei Bugs an drei Tagen (Generation-Label,
Resend-Mail-Felder, PDF-Status-Map) — alle aus dem gleichen Anti-
Pattern „zwei Struct-Literale die in Sync bleiben sollten".

### Fix — Stammdaten-Sync: EEG-Name aus `description` statt `name` *(2026-05-25)*

Im Admin-Bereich „Stammdaten" zeigte das Feld **EEG-Name** den kurzen
internen Handle aus dem Core (z.B. `EEG-TEST`) statt der
beschreibenden Bezeichnung (`Testenergiegemeinschaft EEG 1234`).
Ursache: der GraphQL-Sync hat `eeg.name` gelesen — dieses Feld ist
aber im Core ein technischer Handle ≙ `rcNumber`. Die Klar-Bezeichnung
liegt in `eeg.description`.

- `internal/coreclient/eeg_master_data.go` — `EEGMasterData.Name` →
  `Description` (DTO-Feld umbenannt, JSON-Tag `description`).
- `internal/http/admin.go` — beide Konsumenten umgezogen:
  - `SyncFromCore` persistiert `core.Description` in
    `registration_entrypoint.eeg_name`.
  - `buildEEGSettingsComparison` (Synchron-Banner / Diff) vergleicht
    ebenfalls gegen `Description`.
- `docs/domain-model.md` — Quell-Feld auf `eeg.description` aktualisiert.

Bestandsdaten in `eeg_name` werden beim nächsten Klick auf „Aus
eegFaktura aktualisieren" automatisch korrigiert.

### Fix — Core-Import: Netzbetreiber pro Zählpunkt *(2026-05-25)*

Im `POST /participant`-Payload an den eegFaktura-Core fehlten die
Netzbetreiber-Felder pro Zählpunkt komplett. Im Core-UI hatten die
Zählpunkte des importierten Mitglieds keine Operator-Zuordnung; das
nachgelagerte EDA-Routing hatte keine Adressierung.

Auflösung folgt dem E-Control-Standard: `gridOperatorId` sind die
ersten 8 Zeichen der Zählpunktnummer (`AT` + 6-stelliger Code, z.B.
`AT003000` = Netz Oberösterreich). `gridOperatorName` wird im
Import-Moment gegen den neuen `GET /api/eeg/gridoperators`-Lookup
des Cores aufgelöst. **Jeder Zählpunkt wird unabhängig aufgelöst** —
zwingend für BEGs (Bürgerenergie­gemeinschaften), deren Zählpunkte
über mehrere Netzgebiete verteilt sein können.

Weder ID noch Name werden lokal persistiert; beide werden pro Import
neu abgeleitet. Best-effort-Lookup: wenn `/eeg/gridoperators` nicht
erreichbar ist, läuft der Import mit Id-only weiter (Name bleibt
leer), statt den Import abzubrechen.

Touchpoints:
- `internal/importing/payload.go` — `GridOperatorID/Name`-Felder +
  `deriveGridOperatorID`-Helper auf `CoreMeteringPoint`.
- `internal/coreclient/core_client.go` — `ListGridOperators` auf
  `CoreClient`-Interface + `HTTPCoreClient`-Implementierung gegen
  `GET /api/eeg/gridoperators`.
- `internal/importing/import_service.go` — Operator-Map einmal pro
  Import vor `BuildPayload` ziehen.
- `internal/importing/payload_test.go` — neue Tests für
  GridOperator-Ableitung über mehrere Netzgebiete (inkl.
  malformed-Meter) und nil-Map-Fallback.

Feature-Idee notiert (Backlog `docs/FEATURE-IDEAS.md`): bei regionalen
EEGs prüfen, ob die Operator-ID des Zählpunkts überhaupt im Netzgebiet
der EEG liegt — verhindert Fehl-Anmeldungen. BEGs bleiben ausgenommen.

#### Sackgasse: Meter-Tarif-Casing — snake_case war richtig

Parallel wurde der JSON-Tag `tariff_id` (snake_case) auf
`CoreMeteringPoint.TariffID` kurzzeitig auf camelCase `tariffId`
umgestellt, weil der lokale Core-Mirror (`model.MeteringPoint`)
camelCase deklariert. Der deployte Core nimmt aber tatsächlich
snake_case auf Meter-Ebene (verifiziert via Prod-Payload des Owners
nachträglich) — die Änderung hat den Meter-Tarif beim Import gedroppt
und wurde noch am selben Tag revertiert.

Lesson learnt: der lokale Core-Mirror weicht vom deployten Core ab
(siehe Memory `project_myeegfaktura_source.md`). Bei Wire-Feldern
zählt eine echte Prod-Payload mehr als die Mirror-Struct-Tags.

### PROJ-63 — Follow-up: Firmenbuchnummer-/UID-Label-Alignment *(2026-05-24)*

Owner-Beobachtung im Test-Deploy: die Firmenbuchnummer-Zelle saß ein
paar Pixel über der UID-Zelle. Ursache war der `flex items-center gap-1`-
Wrapper um das UID-Label (für den Info-Icon-Popover) — die Nachbar-
Zelle ohne Wrapper hatte minimal andere Vertikal-Metriken.

Fix: Firmenbuchnummer-Label in dieselbe Flex-Struktur gewickelt (ohne
Icon-Slot), damit beide Grid-Cells strukturgleich rendern und pixelgenau
alignen. Spec PROJ-63 um „Open Follow-ups" (Screenshots nachziehen)
ergänzt.

### PROJ-63 — USt-Pflicht-Checkbox bei Unternehmen + Verein *(2026-05-24)*

Frontend-only Refactor als saubere Lösung für den PROJ-62-Follow-up:
„leere UID = Kleinunternehmer" ist mehrdeutig, weil auch Firmen mit
UID Kleinunternehmer sein können. Statt einer DB-Spalte gating eine
UI-Checkbox das UID-Eingabefeld:

- **`src/components/registration-form.tsx`**: Checkbox „Das Unternehmen
  / Der Verein ist umsatzsteuerpflichtig (Regelbesteuerung)" für
  `memberType ∈ {company, association}`. Default unchecked
  (Kleinunternehmer). UID-Feld nur sichtbar wenn Checkbox aktiv;
  dann Pflicht (Zod-superRefine). `onMemberTypeChange` resettet
  Toggle bei Wechsel auf Nicht-Org-Typ oder auf `municipality`.
- **`src/components/admin-edit-form.tsx`**: Spiegel-Verhalten via
  lokales `useState`, initialisiert aus `application.uidNumber` (truthy
  ⇒ Toggle an). Backwards-kompatibel zu Bestandsanträgen.
- **Gemeinde bewusst ausgeschlossen**: dort wird USt pro Zählpunkt
  (Hoheitsbereich vs. BgA, PROJ-59) differenziert; ein pauschaler
  Toggle wäre irreführend.
- **Kein DB-Touchpoint, kein Backend-Touchpoint, keine Migration.**
  Status „Kleinunternehmer" bleibt implizit aus `uid_number IS NULL`
  ableitbar wie heute. Externe API-Aufrufer unverändert.
- **`tests/PROJ-7-member-types.spec.ts`** AC-13..AC-15: Default-Hidden-
  Zustand, Toggle reveals + macht UID pflicht, Untick cleart UID.

`tsc --noEmit` und `go build ./...` grün.

### PROJ-62 — Follow-up: USt-Hints aus Mitgliedstyp-Dropdown entfernt *(2026-05-24)*

Owner-Feedback: USt-Sätze in der Auswahlbox sind irreführend, da
Kleinunternehmerregelung und UID-Vorhandensein orthogonal sind (auch
Firmen mit UID können Kleinunternehmer sein). Die tatsächliche USt-
Einordnung ergibt sich aus den Folgefeldern und wird in der Abrechnung
geklärt.

- **`src/components/registration-form.tsx`**: `MEMBER_TYPE_OPTIONS`
  liefert nur noch `value` + `label` (kein `hint` mehr). `SelectItem`-
  Render zeigt nur das Label, ohne den `(… % USt.)`-Suffix.
- **`tests/PROJ-7-member-types.spec.ts`** AC-3: Assertion auf „13 %
  USt." / „20 % USt." im Listbox entfernt — stattdessen Prüfung, dass
  alle 5 erwarteten Mitgliedstyp-Labels als Options sichtbar sind.
- **`docs/user-guide/02-member-registration.md`**: USt.-Hinweis-Spalte
  aus der Mitgliedstyp-Tabelle entfernt, erklärender Satz angepasst.
- **`docs/user-guide/changelog.md`**: Eintrag „USt.-Hinweise im Dropdown
  vereinheitlicht" auf „… entfernt" reformuliert.

### PROJ-62 — Frontend: sole_proprietor entfernt *(2026-05-24)*

Build-Failure-driven Refactor analog zum Backend, gesteuert über
`tsc --noEmit`. 5 Frontend-Touchpoints aus AC-FE6:

- **`src/lib/api.ts`**: `MemberType`-Union von 6 auf 5 Werte reduziert
  (sole_proprietor raus). `CONFIGURABLE_FIELDS`-Hint für
  `persons_in_household` aktualisiert (Kleinunternehmer aus der
  Organizations-Liste entfernt).
- **`src/components/registration-form.tsx`**:
  - `MEMBER_TYPE_OPTIONS`-Konstante: sole_proprietor-Eintrag raus,
    Reihenfolge auf Privat → Landwirt → Unternehmen → Gemeinde → Verein.
    USt-Hint für Unternehmen auf „0 % oder 20 % USt." erweitert.
  - Zod-Enum auf 5 Werte reduziert.
  - Org-Label-Branch: sole_proprietor → „Firmenbezeichnung" entfernt;
    company nutzt Default „Firmenname".
  - UID-Pflicht-Validierung bei company entfernt — UID + Firmenbuch-
    nummer sind beide optional.
  - UID-Form-Label: `*`-Markierung entfernt.
  - UID-Hilfe-Text-Popover erweitert um „Leer lassen, wenn unter die
    Kleinunternehmerregelung nach § 6 Abs 1 Z 27 UStG (umsatzsteuer-
    befreit)".
- **`src/components/admin-edit-form.tsx`**: `SelectItem` für
  sole_proprietor entfernt, Reset-Branch in `onMemberTypeChange` zur
  Org-Default-Logik konsolidiert.
- **`src/components/admin-application-detail.tsx`**:
  - `MEMBER_TYPE_LABELS`-Eintrag entfernt + Reihenfolge umgestellt.
  - UID-Field-Conditional entfernt (wird jetzt für alle Org-Typen
    angezeigt, auch wenn leer = Kleinunternehmer).
- **Tests** (`tests/PROJ-7-member-types.spec.ts`):
  - AC-2-Label „Privatperson / Kleinunternehmer" → „Privatperson"
  - AC-3-USt-Hint-Erwartung: „0 % USt." einzelner Check entfernt
    (jetzt im kombinierten „0 % oder 20 % USt."-Hint enthalten)
  - AC-11 invertiert: company mit leerer UID + leerer Firmenbuch-
    nummer muss jetzt **erfolgreich** submitten (Kleinunternehmer-Pfad)

`npm run build` läuft sauber durch. `tsc --noEmit` ohne Fehler.

### PROJ-62 — Backend: Mitgliedstypen Kleinunternehmer + Unternehmen zusammenführen *(2026-05-24)*

`sole_proprietor` (PROJ-28) wird mit `company` verschmolzen. UID-Nummer
wird optional — leer impliziert Kleinunternehmerregelung
(§ 6 Abs 1 Z 27 UStG). Frontend folgt im nächsten Commit.

- **Migration `000056_drop_sole_proprietor_member_type.up.sql`**:
  einfaches `UPDATE application SET member_type='company' WHERE
  member_type='sole_proprietor'`. Kein CHECK-Constraint (es gab keinen),
  keine down.sql (Test-Phase, harte Bereinigung).
- **`internal/shared/models.go`**: Konstante `MemberTypeSoleProprietor`
  entfernt.
- **`internal/shared/requests.go`**: `oneof`-Validator an 3 Stellen auf
  5 Werte reduziert (private/farmer/municipality/company/association).
- **`internal/http/external.go`**: Externe API rejected
  `memberType="sole_proprietor"` jetzt mit 400 statt es zu akzeptieren.
- **`internal/application/application_service.go`**:
  - `clearTypeIrrelevantFields`: sole_proprietor-Branch entfernt
    (Org-Branch deckt das Verhalten ab)
  - `validateMemberTypeFields`: sole_proprietor-Case entfernt; bei
    `company` ist UID-Nummer jetzt **optional** (Kleinunternehmer-Pfad)
  - `isOrgMemberType`-Kommentar aktualisiert
- **`internal/application/admin_service.go::approvalMemberTypeLabel`**:
  Kleinunternehmer-Label-Eintrag entfernt.
- **`internal/importing/payload.go::mapPersonName`**: sole_proprietor-
  Sonderpfad entfernt. Org-Default-Pfad behandelt ex-Kleinunternehmer
  korrekt (firstname leer + companyName gesetzt → firstName=companyName).
  Wenn ein company-Antrag explizit firstname gesetzt hat (z. B. via
  Admin-Edit), wird der **beibehalten** statt überschrieben.
- **`internal/dataexport/excel/fields.go::MemberTypeLabels`**:
  `sole_proprietor`-Eintrag entfernt.
- **Tests**:
  - 4 PROJ-28-spezifische sole_proprietor-Tests entfernt (Verhalten
    existiert nicht mehr)
  - `Company_MissingUID`-Test umgedreht: erwartet jetzt KEINEN Fehler
    bei leerer UID (`Company_MissingUIDAllowed`)
  - 2 payload-Tests umgeschrieben: ex-sole_proprietor läuft als
    `company`, plus neuer Test: firstname-Beibehaltung bei company
  - `TestBuildPayload_BusinessRoleAndRole`-Tabelle: sole_proprietor-
    Zeile entfernt (BusinessRole für company bleibt EEG_BUSINESS)
- **Swagger** regeneriert via `swag init` — sole_proprietor aus
  `docs/{docs.go,swagger.json,swagger.yaml}` entfernt.

Alle Go-Tests grün. Frontend-Refactor (5 Komponenten + 1 Type) folgt
im nächsten Commit via /frontend-Skill.

### PROJ-61 — Security-Review-Findings gefixt *(2026-05-24)*

Fünf Findings aus dem /security-review umgesetzt; PROJ-61 ist jetzt
deploy-ready ohne offene Sub-Tickets aus dem Review.

- **Finding #1 (Medium)**: Längen-Limits für `legal_document.title`
  (500 Zeichen) und `data_export_config.name` (200 Zeichen) als
  Konstanten in `internal/configexport/limits.go` + Check in
  `validateAndSanitize`.
- **Finding #2 (Medium)**: Error-Log-Hygiene in
  `internal/http/configexport.go`. Raw DB-Fehler werden mit
  `error_class`-Kategorie + `.Error()`-String geloggt — verhindert
  Schema-/Query-Leaks in Pod-Logs.
- **Finding #3 (Low)**: Defense-in-Depth-Limit auf
  `?sections=`-Query-Param: max 20 Items (`MaxSectionsQueryItems`).
- **Finding #4 (Info)**: Audit-Log-Key `source_eeg` →
  `claimed_source_eeg`. Markiert klar, dass der Wert User-controlled
  ist; echte verifizierte Ziel-EEG bleibt in `rc_number`.
- **Finding #5 (Info)**: Go-Toolchain `go.mod` `1.26.2` → `1.26.3`.
  Schließt 6 Stdlib-CVEs (XSS in html/template, HTTP/2-Loop,
  quadratic net/mail, NUL-Byte net/Dial). `govulncheck` nach Bump:
  0 Vulnerabilities im eigenen Code.

3 neue Unit-Tests (LegalDocumentTitleTooLong/AtLimitOK,
DataExportConfigNameTooLong).

### PROJ-61 — Bug-Fixes nach QA-Run *(2026-05-24)*

Drei Bugs aus dem /qa-Run gefixt; PROJ-61 ist jetzt Production-Ready.

- **Bug #1 (High, AC-I10)**: Field-Catalog-Drift blockte mit 400 statt
  zu verwerfen + warnen wie spezifiziert. Fix:
  - Neues optionales Capability-Interface `dataexport.DriftFilter` in
    `internal/dataexport/plugin.go`. Plugins, die Config-References auf
    katalog-globale Field-Sets enthalten, implementieren es.
  - Excel-Plugin implementiert `FilterUnknownFields`: filtert
    `columns[].field`-Einträge, die nicht in `AvailableFields` stehen,
    gibt verworfene Field-Keys für die Diff-Warning zurück.
  - `internal/configexport/importer.go::validateAndSanitize` ruft per
    Type-Assertion `DriftFilter` auf, bevor `plugin.ValidateConfig`
    läuft. Gefilterte Config wird in das File-Struct zurückgeschrieben
    (Apply speichert die gefilterte Variante). Warning erscheint im
    Diff-Preview.
- **Bug #2 (Low)**: `intro_text` ohne Längenlimit. Fix:
  - Neue Konstante `MaxIntroTextLength = 50 KB` in `limits.go`.
  - `validateAndSanitize` prüft die Länge nach Sanitisierung; bei
    Überschreitung 400 mit klarer Fehlermeldung. Carry-over im
    UI-Save-Pfad bleibt — eigenes Sub-Ticket zum Konsolidieren.
- **Bug #3 (Low, AC-I14)**: Lock-Error-Message zu generisch. Fix:
  - Neue Helper-Funktion `isLockTimeoutErr` erkennt SQLSTATE `55P03`
    via SQLState()-Interface + Fallback-String-Match.
  - Nur bei echtem lock_timeout → 409 mit „EEG wird gerade konfiguriert";
    andere Lock-Erwerbs-Fehler → 500 mit generischer Meldung. Verhindert
    irreführende UX bei DB-Connection-Problemen.

**Architektur-Verbesserung als Nebeneffekt**: `validateAndSanitize`
gibt jetzt `(warnings []string, err error)` zurück statt nur `error`.
Die alte `collectDriftWarnings`-Funktion ist obsolet (Compat-Pfad
bleibt für externe Aufrufer).

Tests: 8 neue Unit-Tests in `importer_validate_test.go`:
- intro_text-Längenlimit (Reject + Just-Under-OK)
- Excel-DriftFilter (drops + warns + Config-Re-Marshal)
- ValidateConfig rejected weiterhin invalid format
- Plugin-Type-Drift-Warning via validateAndSanitize-Return
- isLockTimeoutErr (nil, String-Match, SQLState 55P03, other-SQLState)

Side-Effect-Import des Excel-Plugins im Test-File, damit Plugin-Registry
für DriftFilter-Tests gefüllt ist.

### PROJ-61 — Konfigurations-Export & -Import pro EEG (Frontend) *(2026-05-24)*

5 React-Komponenten unter `src/components/config-import-export/` +
neuer Tab „Import / Export" in `/admin/settings`. Frontend nutzt die
3 Backend-Endpoints unter `/api/admin/config/*`.

- **`ExportButtons`**: 4 Per-Sub-Typ-Buttons + 1 Komplett-Bundle-Button.
  Pro Klick: `GET /export?sections=...` → JSON-Blob → Browser-Download
  via dynamisches `<a download>`-Element.
- **`ImportDropzone`**: File-Drop-Zone + File-Picker mit
  Client-Side-Validation (`.json`-Extension + ≤ 1 MB). Lädt das File
  zweimal — einmal als FormData für `POST /import/preview`, einmal als
  parsed JSON für späteren Apply-Body (Stateless-Apply).
- **`DiffTable`** (vier Spezialisierungen):
  - `EEGSettingsDiffTable` — 12 Felder; ZP-Prefix mit Network-Icon-
    Tooltip „Netzbetreiber-spezifisch", Cooperative-Shares mit
    Euro-Icon und Cents-zu-EUR-Formatierung
  - `FieldConfigDiffTable` — Name + alt/neu State + Change-Badge
  - `DataExportConfigDiffTable` — Name + Plugin + Change-Badge
  - `LegalDocumentsDiffPanel` — zwei-Spalten-Layout
    „Werden entfernt | Werden hinzugefügt" (kein Match-Key, weil title
    nicht UNIQUE)
  - `WholeSectionDeletionWarning` — rote Box für AC-I4b
- **`DiffPreviewPanel`**: Sektion-Checkboxes (Default: UNAUSGEWÄHLT),
  pro Sektion eine Diff-Card, Apply-Button mit Section-Count-Label,
  Warnings-Block für Drift-Hinweise (Unknown plugin_type).
- **`ConfirmApplyDialog`**: AlertDialog mit Sektion-Liste +
  Irreversibilitäts-Hinweis. Apply läuft erst nach explizitem Klick.
- **`ConfigImportExportSection`** (Top-Level): Tipp-Box „Vor Import
  absichern" + Export-Card + Import-Card mit Drop-Zone-oder-Diff-State.

API-Layer in `src/lib/api.ts` erweitert um:
- TypeScript-Interfaces, die das Backend-Schema spiegeln
  (`ConfigExportFile`, `ConfigDiff`, `ConfigApplySummary`, …)
- `downloadConfigExport`, `previewConfigImport`, `applyConfigImport`
- `triggerBrowserDownload`-Helper (für File-Blob → Browser-Download)

Tab-Integration: 7. Tab „Import / Export" in `/admin/settings/page.tsx`
(Sub-Seiten-Idee aus Tech-Design durch Tab ersetzt — bewusst, weil das
das etablierte UX-Pattern ist und keine parallele Route nötig wird).

`npm run build` läuft sauber durch.

### PROJ-61 — Konfigurations-Export & -Import pro EEG (Backend) *(2026-05-24)*

Neues Feature: Tenant-Admin kann die Konfig einer EEG als versionierte
JSON-Datei exportieren und auf eine andere EEG (für die er auch
Admin-Rechte hat) importieren. Vier Sub-Typen: EEG-Einstellungen,
Field-Config, Legal-Documents, Data-Export-Configs. Replace-Semantik
mit Diff-Preview und Cross-Section-Atomarität.

**Refactoring (Schritt 1):**
- Neues Paket `internal/sanitize/` extrahiert die bluemonday-Policy
  + URL-Format-Validator + ENUM-Checks aus `internal/http/admin.go`.
  Sowohl HTTP-Handler-Pfad als auch der neue Import-Pfad nutzen
  denselben Code (kein Drift-Risiko).
- 16 Unit-Tests für sanitize-Paket.

**Repo-Erweiterungen (Schritt 2):**
- 7 Tx-Variant-Methoden für Cross-Section-Atomarität:
  - `RegistrationEntrypointRepository.SaveAllEEGSettingsTx`
    (konsolidiert: 12 Felder in 1 UPDATE statt 6 separater Save-Aufrufe)
  - `FieldConfigRepository.SaveTx`
  - `LegalDocumentRepository.DeleteByRCNumberTx` + `CreateTx`
  - `dataexport.ConfigRepository.CreateTx` + `SoftDeleteByRCNumberTx`
    + `MarkObsoleteTx`
- Bestehende non-Tx-Methoden bleiben unverändert für UI-Pfade.

**Neues Paket `internal/configexport/`:**
- `schema.go` — versionierte JSON-Strukturen, SchemaVersion=1 strict
- `limits.go` — Per-Sektion-Item-Limits (100/50/50), MaxFileSize 1 MB
- `exporter.go` — assembliert Snapshot aus 4 Repos
- `diff.go` — Diff-Engine: per-Name-Match für field_config +
  data_export_configs, Komplett-Replace-Diff für legal_documents
  (kein UNIQUE auf title), per-Feld-Diff mit Warning-Types für
  EEG-Settings (network_region_specific, financial)
- `importer.go` — Validate + Sanitize + Diff + Apply in Tx mit
  `pg_advisory_xact_lock(hashtext(rc_number))` (10 s Timeout → 409),
  Stateless-Apply (Re-Validation statt Preview-Token), Drift-Warnings
  für unknown plugin_type
- 25 Unit-Tests (Parse, Diff-Engine, Validator-Pipeline)

**HTTP-Handler `internal/http/configexport.go`:**
- `GET /api/admin/config/export?rcNumber=...&sections=...` — JSON-
  Download mit Content-Disposition-Filename
- `POST /api/admin/config/import/preview` — Multipart-Upload,
  Response: strukturierter Diff
- `POST /api/admin/config/import/apply` — JSON-Body mit
  `sectionsToApply`, Response: ApplySummary mit Counts pro Sektion
- Tenant-Auth via existierende KeycloakAuthMiddleware +
  per-Handler-rcNumber-Check
- 1 MB File-Size-Limit (auch via MaxBodySize-Middleware)
- UTF-8-BOM beim Parse stripped (manche Editoren fügen es hinzu)
- Kategorisierte Fehler-Responses (ValidationError mit section + field)

**Bewusst NICHT implementiert:**
- Keine neue DB-Tabelle, keine Migration (Owner-Entscheidung:
  Minimal-Audit nur via slog, kein DB-Audit-Log)
- Keine Pre-State-Auto-Backup (Admin-Verantwortung)
- Keine Preview-Token-State (Apply re-validiert komplett, stateless)
- Keine HEAD-Request-URL-Validation (SSRF-Vermeidung)
- Keine Frontend-Komponenten — kommen im nächsten Schritt

**Bleibt für /qa und /frontend:**
- Integration-Tests gegen Test-DB (Apply-Pfad mit Tx + Advisory-Lock)
- Frontend-UI unter `/admin/settings/import-export`
- E2E-Roundtrip-Tests (Export → Upload → Preview → Apply)

### PROJ-60 — EEG-Stammdaten als exportierbare Spalten *(2026-05-24)*

Eigentümer-Anforderung: Mitglieder-Backup-Liste außerhalb des Systems
braucht EEG-Stammdaten (Name, Adresse, Creditor-ID, …) als Spalten —
diese leben auf `registration_entrypoint` und waren bisher in PROJ-60
nicht exportierbar.

- `ApplicationSnapshot` (`internal/dataexport/plugin.go`) bekommt
  `Entrypoint *shared.RegistrationEntrypoint`-Feld; Loader lädt die
  Entrypoint-Zeile einmal pro Job (1 RC = 1 Entrypoint) und teilt den
  Pointer auf alle Snapshots — keine N-Roundtrips.
- `AppLoader`-Konstruktor erwartet jetzt zusätzlich
  `*RegistrationEntrypointRepository` (Aufruf in `cmd/server/main.go`
  angepasst).
- Neue Field-Kategorie **„EEG-Stammdaten"** mit 8 Spalten in
  `internal/dataexport/excel/fields.go` + Mirror in
  `src/lib/data-export-fields.ts`:
  - `eeg_name`, `eeg_street`, `eeg_street_number`, `eeg_zip`, `eeg_city`
  - `eeg_id` (Core-Referenz)
  - `eeg_creditor_id` (SEPA-Gläubiger-ID)
  - `eeg_contact_email`
- Helper `entrypointStr()` fängt nil-Entrypoint sauber ab — Plugin-
  Vertrag bleibt defensive auch ohne Loader-Hilfe nutzbar.
- 3 neue Go-Unit-Tests in `internal/dataexport/excel/plugin_test.go`
  (Happy-Path, nil-Entrypoint, NULL-Optionalfelder).

### Welle 11 — Severity-Drift + Tot-Code in metrics *(2026-05-24)*

Sub-Tickets **3d + 3e** aus AUDIT-TODO. Reine Cleanup-Welle.

- `internal/metrics/metrics.go`: `statusClassFromString()` (toter Helper
  mit `var _ = ...`-Suppressor) gelöscht; ungenutzten `strconv`-Import
  mit entfernt.
- 3 `slog.Error` → `slog.Warn` umgestellt, wo der Caller noch
  Kontext-/Recovery-Möglichkeit hat:
  - `internal/dataexport/worker.go:135` (Pickup-DB-Fehler, retried im
    nächsten Tick)
  - `internal/mail/service.go:603` (EEG-Template-Render-Fail, Member-Mail
    läuft separat weiter)
  - `internal/application/admin_service.go:993` (PDF-Gen-Fail wird per
    Flag an SendActivationNotification gereicht, Mail geht ohne Attachment)
- Konvention etabliert: `slog.Error` nur für Pfade ohne weiteren
  Caller-Kontext.

§4b (composite-Index) bewusst nicht angefasst — Audit-Eintrag sagt
selbst „nicht ohne EXPLAIN-Daten"; verschärft §4c (Write-Amplification
auf der 14-Index-Tabelle). Wandert in §4a-Folge (Operator-Action).

### Welle 10 — E2E-Auth-Fixture (Header-basierte Test-Claims) *(2026-05-24)*

Sub-Ticket **5h** aus AUDIT-TODO. Schaltet authenticated-Pfade in
CI-Tests frei, ohne dass Keycloak in CI laufen muss.

- **Backend** (`internal/http/auth_middleware.go`): neue Middleware
  `TestHeaderAuthMiddleware()` liest synthetische Claims aus
  Request-Headern:
  - `X-Test-Tenant: RC123,RC456` → Tenant-Admin
  - `X-Test-Superuser: true` → Superuser-Realm-Rolle
  - `X-Test-Subject: <id>` → optionaler Subject
  - Beide leer → 401 (Tests können auth-required asserten)
  - Nur ein Tenant ohne Superuser → 403 (genau wie produktiv-Middleware)
- **`cmd/server/main.go`**: aktiviert die Test-Middleware wenn
  `TEST_AUTH_MODE=headers`, ersetzt dann `KeycloakAuthMiddleware`.
  Sicherheitsguard: `log.Fatalf` wenn `ENVIRONMENT=production` mit
  diesem Flag — die `X-Test-*`-Header sind triviale Forgery.
  `slog.Warn` zum Startup, damit der Modus im Audit-Log sichtbar ist.
- **Tests** (`internal/http/test_header_auth_test.go`): 4 Go-Unit-Tests
  decken alle Modi ab (ohne Header → 401, Tenant-Header → 200, Superuser
  → 200, Custom-Subject).
- **Frontend** (`tests/helpers/auth.ts`): `adminAuthHeaders()`,
  `tenantAdminHeaders()`, `superuserHeaders()`-Conveniences. Smoke-Spec
  `tests/helpers-auth.spec.ts` mit 3 Tests gegen den neuen Mode.
- **CI** (`.github/workflows/ci.yml`): `TEST_AUTH_MODE: headers`
  env-Var im `e2e`-Job; aktiviert damit die Test-Middleware.
- **PROJ-17** AC-BE1 und AC-BE5: `test.skip(CI)` entfernt — die Tests
  prüfen jetzt korrekt 401 ohne Header.

### Welle 9 — Playwright in CI + `skipIfBackendDown`-Konsolidierung *(2026-05-24)*

Sub-Ticket **5a + 5i** aus AUDIT-TODO (Audit-Marathon-Restschuld).

- `.github/workflows/ci.yml`: neuer `e2e`-Job mit Postgres-17-Service,
  `migrate -direction=up`, `dev_seed.sql`, Backend (Go) + Frontend
  (Next.js production-build) als Background-Prozesse mit `/health`- bzw.
  `/`-Polling, Playwright-Browser-Cache und Report-Artifact-Upload bei
  Failure.
- PR-CI läuft Chromium-only über neue `PLAYWRIGHT_BROWSERS=chromium`
  ENV-Variable (124 statt 496 Tests); Multi-Browser-Matrix (Firefox +
  WebKit + Mobile Safari) bleibt lokal Default und wandert in einen
  zukünftigen nightly-Workflow (eigenes Sub-Ticket).
- `playwright.config.ts`: `webServer` deaktiviert wenn `process.env.CI`,
  weil der Workflow Backend + Frontend selbst startet; Reporter in CI
  zusätzlich `list` (Stream-Output).
- Acht duplizierte `skipIfBackendDown`-Helper in den Spec-Dateien
  (PROJ-11 bis -17, PROJ-25) durch konsolidierten Import aus
  `tests/helpers/backend.ts::ensureBackendUp` ersetzt. Akzeptiert sowohl
  `Page` als auch `APIRequestContext`. In CI (`process.env.CI === 'true'`)
  hart-fail statt skip — verhindert grüne Test-Runs bei totem Backend.
- 12 latent-brittle Tests in 6 Spec-Files mit
  `test.skip(process.env.CI === "true", "AUDIT-TODO §5b/5h: …")`
  getaggt. Failure-Modi:
  - **§5b (Seed-Inadequacy)**: PROJ-7/8/9/11/12/14 — Tests setzen
    reichere Settings/Configs voraus, die der minimal-seed
    (`RC123456 / is_active=TRUE`) nicht liefert. UI rendert nicht
    wie erwartet (z.B. Combobox "Mitgliedstyp" fehlt).
  - **§5h (Auth-Fixture)**: PROJ-17 (AC-BE1/BE5) — erwarten 401,
    bekommen in CI 200, weil `KEYCLOAK_JWKS_URL` leer ist.
  Lokal mit echtem Backend laufen die Tests weiterhin.
- Verbliebene Sub-Tickets: 5b–5f (fehlende E2E-Specs +
  Seed-Erweiterung), 5g (MailHog), 5h (Auth-Fixture / Test-Token),
  5j (`networkidle` → `waitForResponse`),
  Nightly-Multi-Browser-Workflow.

### PROJ-60 — Datenweiterleitung an externe Systeme (async Plugin-Framework + Excel/CSV-Plugin) *(2026-05-23)*

Komplett neues asynchrones Framework für die Weitergabe importierter
Mitglieder an externe Systeme. V1 ships das Excel/CSV-Export-Plugin;
Phase 2 (Zoho, HubSpot, …) baut ohne Framework-Eingriff auf.

**DB-Schema (Migration 000052):**
- `data_export_config` — Plugin-Konfigurationen pro EEG, Soft-Delete via `deleted_at`, UNIQUE auf `(rc_number, name)` WHERE non-deleted
- `data_export_job` — Async-Job-Queue + langlebiger Audit-Trail, mit `config_snapshot` (immune gegen Config-Edits zur Laufzeit) und 4 spezialisierten Partial-Indizes (Pickup, Concurrency-Check, BackOffice-Liste, Zombie-Scan)
- `data_export_result` — Datei-BLOBs mit 24 h TTL, FK CASCADE auf Job

**Backend:**
- Plugin-Registry mit Side-Effect-Import (`sql.Driver`-Pattern) — neue Plugins via einem Import in `cmd/server/main.go`
- In-App-Worker-Pool (3 Goroutines, 5 s Polling) mit `SELECT ... FOR UPDATE SKIP LOCKED` — multi-replica-safe
- Worker-Shutdown vor HTTP-Shutdown (`Worker.Stop(ctx)` mit 60 s Budget) — keine Zombie-Jobs mehr bei Rollouts; Helm-Template `terminationGracePeriodSeconds: 120`
- K8s-CronJob `data-export-cleanup` (`*/10 * * * *`): Zombie-Recovery + BLOB-TTL + DSGVO-Hard-Delete nach 7 J
- 12 neue Admin-Endpoints unter `/api/admin/data-export/*` (Plugins-Liste, Configs CRUD, Preview, Jobs CRUD inkl. Listing, Download, Retry)
- DSGVO: `slog.Info classification=sensitive-export` bei IBAN/Geburtsdatum-Exports; CSV/Excel-Injection-Defense für Werte mit Prefix `=+-@\t\r` (auch nach Leading-Whitespace/NBSP/BOM)
- Filename-Schema `{rc_number}-{config_name}-{YYYY-MM-DD}.{xlsx|csv}` mit Path-Traversal-Sanitization
- FailureMailer-Adapter sendet Plain-Text-Mail an `registration_entrypoint.contact_email` mit Job-Details + BackOffice-Link
- Batch-Loader (`GetByIDs` / `GetByApplicationIDs`) — N+1 eliminiert für 1000-Apps-Bulks
- 30 Unit-Tests im Excel-Plugin (ValidateConfig, formatValue, Renderer, Process, sanitiseSpreadsheetValue inkl. Whitespace-Bypass-Edge-Cases)

**Frontend:**
- Settings-Page jetzt mit shadcn Tabs (6 Sektionen statt langer Liste): Stammdaten | Einleitungstext | Formular-Felder | Rechtsdokumente | Externe API | Datenweiterleitung
- Excel-Editor mit Spalten-Mapping (Header/Feld/Format/Up-Down-Remove), Live-Preview (debounced, skipped bei unvollständigen Spalten), 3 Standard-Vorlagen (Newsletter, CRM-Stammdaten, Buchhaltung), DSGVO-Popover bei IBAN/Geburtsdatum, alphabetische Sortierung pro Kategorie im Feld-Dropdown
- Trigger-Dialog (einstufige Plugin-Konfig-Liste), Bulk-Action in Antragsliste mit Cross-EEG-Schutz, Single-Action im Antrags-Detail
- Polling-Modal (2 s/5 s) mit Progress-Bar, Download bei Done, Retry bei Failed (Retry-Polling re-subscribed via `onRetried`-Callback)
- BackOffice-Jobs-Tab mit Failed-Badge (7 Tage), Status-Filter, Cursor-Pagination
- Aussagekräftige Fehlermeldungen via `formatValidationError` (Backend-`fields`-Map wird ausgepackt, Pfade wie `columns[1].header` prettifiziert zu „Spalte 2 → Spaltenkopf")

**Bugfix-Welle parallel zu PROJ-60:**
- `persons_in_household` ist konzeptuell nur für `private` und `farmer` sinnvoll → Backend `clearMemberTypeFields` cleart bei Org-Typen, Required-Check zusätzlich auf `isNaturalPerson` gegated, Public-Form rendert das Feld nicht mehr für Org-Typen, Admin-Field-Config-Editor zeigt zwei Badges (`consumption` + `natural_person`)
- Verein-Submit funktioniert wieder bei EEGs, die `persons_in_household` als Pflichtfeld konfiguriert hatten
- `jobs-list` functional setState verhindert Race bei „Mehr laden" + Filter-Wechsel
- Placeholder-Verstoß im Bulk-Reject-Dialog entfernt (Label trägt jetzt den Hinweis)

**Audit-Welle 2 (Re-Audit-Findings, 2026-05-23):**
- Worker-Shutdown ohne TriggerJob-Race: neuer `JobService.MarkShuttingDown()` (atomic.Bool) wird in `main.go` vor `workerCancel()` gerufen; TriggerJob/Retry returnen 409 während Drain — keine Zombie-Jobs mehr durch hastige Admins
- `LoadForExport` hard-failt jetzt wenn alle Apps via Tenant-Filter rausfliegen (vorher: silent leerer Export wurde als „done" markiert)
- ListJobs N+1 eliminiert: neue Repo-Methode `GetMetadataByJobIDs` reduziert ein 1+N-Listing-Query-Pattern auf 1+1
- Retry-Modal: `onRetried`-Prop ist jetzt required (TypeScript-enforced) — Tech-Debt-Trap (silent polling-freeze nach Retry) geschlossen
- K8s-Hardening auf den zwei PROJ-60-Templates: `automountServiceAccountToken: false` + `seccompProfile: RuntimeDefault` (lateral movement bei Container-Compromise blockiert)
- 12 Swag-Annotationen für `internal/http/dataexport.go` (vorher: `swag init` skippte PROJ-60 silent)
- Doku-Hygiene: domain-model.md Section-Nummern §3.6/3.7/3.8 → §3.9/3.10/3.11 (Kollision mit `document_consent`/`external_api_key`/`reference_number_counter` aufgelöst); CHANGELOG/TODO „11 Endpoints" → „12"

**Audit-Welle 3 (Re-Re-Audit-Folge, 2026-05-24):**
- `CountFailedSince`-Fehler im Jobs-Listing wird jetzt geloggt (vorher silent geswallowed → Badge zeigte stille 0)
- `LoadForExport`-Fehlermessage entfernt die RC-Nummer aus dem User-Error (defensiver gegen Cross-Tenant-Info-Leak)
- TODO-docs-sync.md-Drift gefixt (alte §3.6/3.7/3.8-Referenz)

**Helm-Deep-Audit-Welle 4 (2026-05-24):**
- **Produktiver Bug behoben**: `data-export-cleanup`-CronJob (PROJ-60) war nicht in der Postgres-NetworkPolicy-Allowlist → wäre bei strikten CNIs (Calico/Cilium) bei jedem Run gescheitert. Vierter `podSelector`-Eintrag ergänzt.
- Konsistente Härtung über alle 8 Pod-Workloads: `seccompProfile: RuntimeDefault`, `automountServiceAccountToken: false` (wo möglich), `readOnlyRootFilesystem: true`, drop-ALL-capabilities, `allowPrivilegeEscalation: false` (vorher nur backend + data-export-cleanup gehärtet)
- Postgres: livenessProbe (höheres `failureThreshold` als readiness), `terminationGracePeriodSeconds: 60` für sauberen smart-shutdown, cpu-Limit
- Frontend: startupProbe für Next.js Cold-Start (failureThreshold 30 × 2 s), tmp-emptyDir für readOnlyRootFilesystem
- Ingress: `ssl-redirect` + HSTS (180 d) + X-Content-Type-Options + Referrer-Policy + X-Frame-Options + proxy-body-size 10 MB
- Namespace: PSA `restricted` enforced + audit + warn (defensive: zukünftige nicht-konforme Workloads werden vom API-Server rejected)
- seed-job: SQL-Injection-Vektor geschlossen — `values.seed.*` werden jetzt als Env-Vars in psql gesetzt + `\set` + `:'name'`-Safe-Quoting statt Template-Inline-Interpolation
- restart-cronjob: `startingDeadlineSeconds: 300`, Pod- und Container-Security-Context
- `_helpers.tpl`: `app.kubernetes.io/version`-Label auf `.Chart.AppVersion` statt Backend-Image-Tag (war für Postgres/Frontend/CronJob irreführend)
- **Resource-Requests minimiert** (Owner-Entscheidung): backend 50→10 m, postgres 100→25 m, frontend 100→25 m, jobs 10→5 m. Cluster-Sizing-Kosten gering halten solange wir noch nicht produktiv sind. Limits bleiben großzügig für Peak.

Bewusst aufgeschoben in `docs/AUDIT-TODO.md` 5a–5f: TLS-Block (cert-manager), SealedSecrets-Migration, Egress-NetPol, HA + PDB, HPA, Postgres-Backup-Doku.

**Observability-Audit-Welle 5 (2026-05-24):**
- **CRITICAL — PII-Leak in Logs gefixt**: `internal/mail/service.go` loggte `app.Email` + `entrypoint.ContactEmail` voll an 5 Stellen (Verstoß gegen `.claude/rules/security.md`: „IBAN, email, phone, name must not appear in application logs"). Neue `emailDomain()`-Helper-Funktion gibt nur den `@suffix` zurück; alle 5 Stellen umgestellt auf Log-Key `to_domain`.
- Neues Paket `internal/logfields/` zentralisiert slog-Field-Keys (`RCNumber`, `JobID`, `ApplicationID`, `Classification`, `AdminUserID`, …) plus fixiertes `classification`-Vokabular (`pii-read`, `pii-export`, `sensitive-export`). Verhindert Drift (`"rc"` vs `"rc_number"`, `"user_id"` vs `"admin_user_id"`); neue Code-Stellen sollen importieren statt Strings tippen.
- DSGVO-Audit-Trail-Marker auf zwei weitere PII-Pfade ausgeweitet: `GetApplicationDetail` (`classification=pii-read`) und `ExportApplicationExcel` (`classification=pii-export`). Pendant zu PROJ-60 `sensitive-export`. Log-Shipper können auf `classification=`-Vokabular filtern und an die Compliance-Archivierung routen.
- `internal/dataexport/worker.go` Sensitive-Export-Marker nutzt jetzt die `logfields`-Konstanten statt Literals.

Bewusst aufgeschoben in `docs/AUDIT-TODO.md` 3a–3e (eigene PROJs):
- 3a: 10 neue Prometheus-Metrics (`coreclient_request_duration`, `dataexport_jobs_total`, `_job_duration`, `_queue_depth`, `_workers_busy`, `_blob_bytes_total`, `_cleanup_runs_total`, `turnstile_verifications_total`, `applications_submitted_by_type_total`, `smtp_send_duration_seconds`)
- 3b: OpenTelemetry-Tracing-Bootstrap mit 4 Stufen (CoreClient, DataExport-Pipeline, Trace-Log-Correlation, K8s-Collector)
- 3c: Logger-Context-Middleware (`slog.With("request_id", ...)` im ctx, Helper `log.FromCtx`)
- 3d: Tot-Code `metrics/metrics.go:statusClassFromString` aufräumen
- 3e: Severity-Drift bereinigen (3 Stellen `slog.Error` → `slog.Warn` für transiente/Caller-Kontext-Pfade)

**Data-Model-Slimming-Audit-Welle 8 (2026-05-24, Migrationen 000054 + 000055):**
- **DROP `application.reviewed_by_user_id`** (Migration 000054) — echtes Tot-Datum, war via COALESCE in `UpdateStatusAdminTx` gesetzt + ins JSON serialisiert, aber nirgends im Code konsumiert. Audit-Quelle für „wer hat Status geändert" ist `status_log.changed_by_user_id`. Begleitend: `UpdateStatusAdminTx`-Signatur entfernt den `reviewedByUserID`-Parameter; 3 Caller in `admin_service.go` + `importing/import_service.go` angepasst (system-actor landet weiterhin in `status_log`).
- **DROP `application.email_confirmation_used_at`** (Migration 000055) — 100 % redundant zu `email_confirmed_at` (`MarkEmailConfirmedTx` setzte beide auf denselben NOW(); Idempotenz-Check `application_service.go:825` funktional identisch, wurde auf `EmailConfirmedAt != nil` umgestellt). Down-Migration backfillt aus `email_confirmed_at`.
- Doku-Patches in `docs/domain-model.md` für zwei bewusste Trade-offs: §3.10 `application_ids UUID[]` als bewusste Ausnahme zur „no JSON columns"-Regel (Snapshot-Charakter) + §3.9 `is_obsolete` als bewusst materialisiertes Cache-Boolean (Registry runtime-only).

**Bestätigt OK, nicht angefasst** (waren TODO-Verdacht, alle legitim): `accuracy_confirmed`, `privacy_version`, `has_contact_person`/`has_billing_email`, `processed_count`/`total_count`/`retry_count` als INT, `field_config` als sparse-table, alle PROJ-46-Lifecycle-Timestamps.

**E2E-Test-Coverage-Audit-Welle 7 (2026-05-24):**
- Browser-Matrix erweitert in `playwright.config.ts`: Desktop-Firefox + Desktop-WebKit (Safari-Engine) ergänzt; vorher nur Chromium + Mobile-Safari.
- Neue Helper `tests/helpers/test-data.ts` mit `uniqueEmail()`/`uniqueRef()`/`TEST_RC_NUMBER`. Verhindert Akkumulations-Flakes durch fixed-string-Collisions (`test@example.at` etc.) und nutzt `@e2e.local` (RFC 6761-reserviert, kann nicht resolven).
- API-Vertrag-Drift in `tests/PROJ-12-sepa-mandate-pdf.spec.ts:156` gefixt: Backend liefert `active` (per `shared.RegistrationConfig`), Spec hatte `isActive` → `toHaveProperty` lief silent grün gegen nicht-existente Property.

Coverage-Score nach Audit: 11 von ~50 Deployed/Approved-PROJs haben eine Spec; 4 davon mit Voll-Coverage (PROJ-7/8/9/11/15), Rest sind Smoke/Auth-Wand-Tests. Top-5-Lücken: PROJ-1 Happy-Path, PROJ-31 Email-Confirmation, PROJ-46/53 Post-Import-Stati, PROJ-60 Data-Export, PROJ-2 Status-Transition-Matrix.

Bewusst aufgeschoben in `docs/AUDIT-TODO.md` 5a–5j (eigene PROJs):
- 5a **CRITICAL** — Playwright in CI aktivieren (eigener Job mit Postgres-Service-Container + globalSetup); ohne den verrotten Specs ungemerkt
- 5b–5f: die fünf priorisierten fehlenden Top-Specs
- 5g: MailHog/Mock-SMTP für Mail-Assertions
- 5h: Auth-Fixture (Test-Token / NODE_ENV=test-Bypass)
- 5i: `skipIfBackendDown` → hart-fail in CI
- 5j: `waitForLoadState("networkidle")` → `waitForResponse(...)` an 10 Stellen

**DB-Performance-Audit-Welle 6 (2026-05-24, Migration 000053):**
- **HIGH**: fehlender Index auf `external_api_key.key_hash` → jeder externe API-Call (Bearer `moak_*`) machte Seq-Scan. Neuer Partial-Index `WHERE revoked_at IS NULL` (widerrufene Keys werden ohnehin 401 abgewiesen).
- **LOW-Cleanup**: zwei redundante Plain-B-Tree-Indizes gedroppt — `idx_application_reference_number` und `idx_registration_entrypoint_rc_number` waren Duplikate von UNIQUE-Constraints (Postgres legt für UNIQUE automatisch einen Index an). Spart Write-Amplification.
- `docs/domain-model.md` §3.7 ergänzt um den neuen Partial-Index.

Bewusst aufgeschoben in `docs/AUDIT-TODO.md` 4a–4c:
- 4a: EXPLAIN-ANALYZE gegen Prod-DB für 6 Hot-Path-Queries (Operator-Action) + `pg_stat_user_indexes`-Auswertung nach 30 Tagen Prod-Laufzeit (DROP-Index-Kandidaten finden)
- 4b: `idx_application_submitted_at` ggf. durch composite `(rc_number, submitted_at DESC)` ersetzen, falls EXPLAIN das nahelegt
- 4c: Write-Amplification auf `application` (14+ Indizes) im Auge behalten

### PROJ-57 v3 — Ansprechperson ohne Master-Switch, drei Felder einzeln steuerbar *(2026-05-21)*

Vereinfachung des Konfigurations-Modells: der separate
`contact_person`-Master-Switch entfällt. Stattdessen werden alle drei
Felder (`contact_person_name`, `contact_person_email`,
`contact_person_phone`) einzeln per field_config konfigurierbar
(hidden/optional/required). Die Ansprechperson-Checkbox im Public-
Formular erscheint automatisch, sobald mindestens eines der drei
Felder nicht hidden ist.

- field_config: `contact_person` entfernt; neuer Eintrag
  `contact_person_name`; Defaults aller drei Felder = hidden
  (Feature aus, bis EEG aktiv konfiguriert)
- Backend: neuer Helper `contactPersonEnabled(fieldConfig)`,
  `clearContactPersonIfDisabled` cleart bei allen-drei-hidden,
  Required-Validierung pro Feld nur bei state=required
- Public-Formular: Checkbox-Sichtbarkeit aus den drei Sub-Feldern
  abgeleitet; Name-Feld auch konditional renderbar; Pflicht-Marker
  dynamisch
- Admin-Field-Config-Editor: zeigt jetzt drei Org-Typen-Einträge
  statt vier (Master-Switch + 2). Die Reihenfolge folgt der
  natürlichen Form-Reihenfolge (Name → Email → Telefon).

Hinweis für bestehende Konfiguration: alte EEGs mit `contact_person`-
Eintrag in der DB werden vom System ignoriert. Sie müssen die drei
Subfelder neu konfigurieren, um das Feature wieder zu aktivieren.

### PROJ-58 — Abweichende Rechnungs-E-Mail für Org-Mitgliedstypen *(2026-05-21)*

Bei Unternehmen, Vereinen und Gemeinden kann jetzt eine separate
E-Mail-Adresse für den Rechnungsversand angegeben werden. Per
Checkbox in der Bankverbindungs-Sektion aktivierbar.

- Zwei neue Spalten auf `application` (Migration 000051):
  `has_billing_email` (BOOL) + `billing_email` (TEXT)
- field_config-Eintrag `billing_email` (Default `hidden`,
  per-EEG konfigurierbar)
- Public-Form: Checkbox + Input in Bankverbindungs-Card, nur bei
  Org-Mitgliedstypen UND field_config != hidden. Required bei
  aktivem Toggle + Email-Format-Check.
- Admin-Detail-View + Admin-Edit-Form: Toggle + Email editierbar
  für Org-Mitgliedstypen
- Beitritts-PDF: zusätzliche Zeile „Rechnungs-E-Mail:" in der
  Bankverbindungs-Sektion, wenn gesetzt
- Server-Side-Cleanup: `clearBillingEmailIfDisabled` cleart die
  Felder auf NULL bei Toggle-off, nicht-Org-Mitgliedstyp oder
  field_config=hidden

Vorbereitung für das künftige eigene Rechnungsmodul. Versand-Logik
folgt mit dem Billing-Modul, kein automatischer Mail-Versand jetzt.

### PROJ-57 v2 — feiner steuerbare Ansprechperson-Pflichtigkeit *(2026-05-21)*

Erweiterung der Ansprechperson-Logik aus PROJ-57: Email und Telefon
können seit dieser Version pro EEG einzeln auf `hidden | optional |
required` gestellt werden. Name bleibt fix Pflicht wenn Toggle aktiv
(ohne Name keine sinnvolle Ansprechperson).

- Zwei neue field_config-Einträge: `contact_person_email` und
  `contact_person_phone`, beide Default `required` (= bisheriges
  Verhalten unverändert für bereits konfigurierte EEGs)
- Im Admin-Field-Config-Editor mit „Org-Typen"-Badge sichtbar
- Public-Form rendert das jeweilige Feld nur, wenn nicht hidden,
  und passt Pflicht-Marker (*) dynamisch an
- Server-Cleanup in `clearContactPersonIfDisabled` setzt das Detail-
  Feld auf NULL, wenn der EEG-State `hidden` ist — Schutz vor
  forged Clients
- Email-Format wird auch bei `optional` geprüft, falls Wert da
- Admin-Edit-Form sieht weiterhin alle drei Felder durchgehend
  (Admin-Korrektur-Pfad nicht eingeschränkt; Backend cleart bei hidden)

### PROJ-57 — Ansprechperson für Org-Mitgliedstypen *(2026-05-21)*

Optionale Ansprechperson für Unternehmen, Vereine und Gemeinden. Toggle-
Checkbox aktiviert drei zusätzliche Felder (Name, E-Mail, Telefon), die
in PDF, Submission-Mail und Admin-UI durchlaufen.

Eckdaten:

- **Vier neue Spalten** auf `application` (Migration 000050):
  `has_contact_person` (BOOL), plus `contact_person_name/email/phone` (TEXT).
- **field_config-Eintrag** `contact_person` (Default hidden, per-EEG
  konfigurierbar). Single-Switch für den ganzen Block; Mitgliedstyp-
  Filterung im Code (nur company/association/municipality).
- **Public-Formular**: Checkbox unter UID/Vereinsnummer. Wenn aktiv:
  Name + E-Mail + Telefon (alle drei Pflicht). Required-Validierung
  gegated auf Toggle-aktiv (verhindert Submit-Hänger-Bug-Pattern).
- **Admin-UI**: Detail-View zeigt Ansprechperson-Block wenn gesetzt;
  Edit-Form erlaubt Toggle umschalten und Werte editieren (sichtbar
  nur bei Org-Mitgliedstypen).
- **Beitritts-PDF**: neuer Block „Ansprechperson" zwischen
  Mitgliedsdaten und Bankverbindung, gerendert wenn Toggle aktiv.
- **EEG-Submission-Mail** (PROJ-20): neuer Block in
  `application_submitted_eeg.html` zwischen Adresse und Bankverbindung.
- **Server-Side-Cleanup**: `clearContactPersonIfDisabled` cleart die
  drei Felder auf NULL, wenn Toggle aus oder Mitgliedstyp nicht in der
  Org-Liste — schützt gegen forged Clients.
- **Excel-Export** (PROJ-17) wurde bewusst NICHT erweitert.

### PROJ-56 — Netzbetreiber-Info-Seite im Beitrittsbestätigungs-PDF *(2026-05-21)*

Zusätzliche PDF-Seite mit allen Daten, die die EEG-Verwaltung für die
Netzbetreiber-Korrespondenz braucht. Wird konditional gerendert: nur
wenn das Mitglied die Netzbetreiber-Vollmacht aktiv erteilt hat
(PROJ-44).

Eckdaten:

- **Zwei neue per-Mitglied-Felder** auf `application`:
  `network_operator_customer_number` und `meter_inventory_number`
  (Migration 000049). Beide TEXT NULL.
- **Conditional Rendering** im Public-Formular: erscheinen direkt
  unter der Vollmachts-Checkbox, sobald sie aktiviert wird. Jedes Feld
  einzeln per `field_config` ein-/ausblendbar; Required-Status pro EEG
  konfigurierbar.
- **Admin-UI**: zwei Felder im Detail-View (Anzeige) und im Edit-Form
  (editierbar) — beide nur sichtbar wenn Vollmacht aktiv.
- **PDF-Seite** (`approval_pdf.go`) mit:
  - Überschrift "Informationen für den Netzbetreiber"
  - Kundennummer + Inventarnummer
  - [X]-Box mit Volltext der Vollmacht + Timestamp
    ("Vollmacht erteilt am `<submitted_at>`")
  - Tabelle aller Zählpunkte (Nr / Adresse zwei-zeilig / Typ CNSM-GNRT / TF)
  - 33-stellige AT-Zählpunkt-Nummern werden in 5 Gruppen (2-6-5-10-10)
    gruppiert dargestellt für bessere Lesbarkeit.
- **Validierung**: Required-Check der zwei Felder läuft nur wenn die
  Vollmacht aktiv ist — sonst Submit-Hänger-Falle wie beim Geburtsdatum
  vermieden (vgl. Commit `72d380b`).
- **Server-Side-Cleanup**: `clearNetworkAuthIfHidden` setzt die zwei
  Felder auf NULL, wenn die Vollmacht nicht (mehr) erteilt ist oder
  die EEG die Felder versteckt hat.

### Bug-Fixes 2026-05-21

- **Beitrittsbestätigungs-PDF**: Netzbetreiber-Vollmacht wurde sowohl
  in „Erteilte Zustimmungen" als auch in „Weitere Angaben" gerendert.
  Der Duplikat-Eintrag in „Weitere Angaben" wurde entfernt; der voll-
  formulierte Block in „Erteilte Zustimmungen" bleibt.
- **Beitrittsbestätigungs-PDF**: Format der Zustimmungs-Zeile geändert
  von `- Statuten — Zugestimmt am …` auf `- Statuten zugestimmt am …`
  (Gedankenstrich entfernt, klein geschrieben).
- **Public-Formular**: Hinweis „SEPA-Mandat erhältst du per E-Mail …"
  wandert aus der Einwilligungs-Box in die Bankverbindung-Box —
  kontextnah am IBAN-Feld statt versehentlich wie eine weitere
  Einwilligung wirkend.
- **Public-Formular**: Submit-Hänger bei Mitgliedstyp `sole_proprietor`,
  `company`, `municipality`, `association` behoben — Geburtsdatum-
  Validierung lief unbedingt, obwohl das Feld nur für isPerson-Typen
  gerendert wird. Selbe Falle für consumption-only-Felder
  (`persons_in_household`, `heat_pump`, …) zusätzlich gefixt.

### PROJ-54 — Repo-Split: privates Hauptrepo + öffentlicher Mirror *(2026-05-20)*

Aktive Entwicklung läuft ab sofort im privaten Repo
`Marki4711/eegfaktura-member-onboarding-private`; der öffentliche Repo
`Marki4711/eegfaktura-member-onboarding` wird via GitHub-Action-Mirror
auf jeden Push automatisch aktualisiert.

Eckdaten:

- **Whitelist** (`.github/mirror-whitelist.txt`): definiert was im
  Public-Mirror erscheint. `private/` und alle `.github/`-Inhalte sind
  ausgeschlossen.
- **Frontmatter-Filter**: einzelne Markdown-Dateien mit YAML-Frontmatter
  `visibility: private` werden zusätzlich aus dem Mirror entfernt.
- **CI/CD-Verteilung**: Snyk, EOL-Check, Docker-Publish, Dependabot, CI
  Build & Test laufen nur im privaten Repo. Public hat keine Actions.
- **Git-Hooks** (`.githooks/pre-commit`, `pre-push`): defensive Schicht,
  blockt direkten Push aufs Public-Repo + warnt bei `private/`-Pfaden.
- **Smoke-Build** (Go + Node) auf dem gefilterten Output: schlägt fehl,
  bricht Mirror ab (kein Public-Push).
- **Mirror-Lag**: ~80–90 s pro Push.

Sensible Bereiche (Pricing, Verträge, DPIA, Pen-Test-Reports,
Anbieter-Setups, eigenes Rechnungsmodul) landen ab sofort unter
`private/` und werden nicht öffentlich gespiegelt.

### Optionales UID-Feld für Verein im Public-Form *(2026-05-20)*

Mitgliedstyp `association` zeigt im öffentlichen Registrierungsformular jetzt
zusätzlich zur (Pflicht-) Vereinsnummer ein **optionales UID-Nummer-Feld** —
analog zur bereits vorhandenen Umsetzung bei `municipality` (Gemeinde).

Backend, Admin-Edit-Form, Mail/PDF/Excel und der Core-Payload-Mapper kannten
das Feld bereits für `association` (kein Nullen in `clearMemberTypeFields`,
kein Required-Check); reines Frontend-Rendering-Gap geschlossen
(`src/components/registration-form.tsx`).

### Teilnahmefaktor pro EEG konfigurierbar *(2026-05-19)*

Das Feld `participation_factor` (Teilnahmefaktor in %) ist jetzt über die
PROJ-8-Field-Config pro EEG ein-/ausblendbar:

- Neu in `knownConfigurableFields` (Backend) + `CONFIGURABLE_FIELDS.meteringPoint`
  (Frontend) mit Default `optional` — heutiges Verhalten bleibt erhalten.
- Bei `hidden` oder `admin_only` rendert das Public-Formular kein Eingabefeld;
  der Wert wird serverseitig automatisch auf **100 %** defaulted
  (`defaultParticipationFactor` in `application_service.go`).
- Bei `optional` oder `required` ist das Feld sichtbar und mit 100 % vorbelegt —
  das Mitglied kann ändern oder den Default beibehalten.
- Validate-Tag von `required,min=1,max=100` auf `min=0,max=100` gelockert,
  damit das Frontend bei `hidden` einen 0er-Submit machen kann (Service
  mappt 0 → 100).
- **PDF, Mail und Excel-Export zeigen den Teilnahmefaktor in allen Modi
  unverändert** — der Toggle steuert nur die Public-Form-Sichtbarkeit, nicht
  die Render-Pfade. Der Core-Import (`partFact` = Mitglied-Wert) bleibt
  unverändert.

Docs: `docs/user-guide/06-admin-settings.md` Abschnitt „Spezielle
konfigurierbare Felder" um den neuen Toggle ergänzt.

### PROJ-53 — Aktivierungs-Modus pro EEG + Beitrittsbestätigung erst bei `activated` + manueller `approved → activated`-Skip *(2026-05-19)*

Drei zusammenhängende Änderungen am Activation-/Mail-Lifecycle:

**1. Beitrittsbestätigung wandert von `imported` nach `activated`**
- `SendImportedNotification` (volle Beitrittsbestätigung + PDF + optional Mandat) entfällt.
- Neue Funktion `SendMandateAtImportNotification` (schlank, nur Mandat-Anlage)
  läuft beim Wechsel auf `imported` — und auch nur dann, wenn überhaupt ein
  Mandat zu versenden ist (b2b oder `sepa_mandate_at_import=true`).
- Neue Funktion `SendActivationNotification` (volle Beitrittsbestätigung mit
  PDF an Member + EEG-Contact) läuft beim Wechsel auf `activated`.
- Templates: `application_imported_*.html` umgeschrieben auf "Anlage Mandat —
  Beitrittsbestätigung folgt"; `application_activated_member.html` enthält
  jetzt die volle Beitrittsbestätigung; neues `application_activated_eeg.html`.
- Alte kurze `SendActivatedNotification`-Welcome-Mail entfällt (war Funktion
  mit identischem Auslöser, aber dünnerem Inhalt — wird durch die volle
  Beitrittsbestätigungs-Mail abgelöst).
- **Idempotenz:** neue Spalte `application.activation_notification_sent_at`
  speichert den Sendetag. Wird beim erfolgreichen Versand gesetzt; mehrfache
  Aktivierungen schicken nicht doppelt.
- **Hartes Cut-off für Bestandsanträge:** Migration 047 setzt das Flag
  retrospektiv für alle Anträge in `imported/ready_for_activation/
  awaiting_bank_confirmation/activated` auf `updated_at`. So bekommen
  Mitglieder, die schon eine "alte" Beitrittsbestätigung beim Import erhalten
  haben, beim Übergang auf activated keine zweite.

**2. Aktivierungs-Kriterium pro EEG konfigurierbar**
- Neue Spalte `registration_entrypoint.activation_mode` (Default
  `participant_active`, alternativ `any_meter_registration_started`).
  Migration 048 inkl. DB-CHECK.
- `CoreParticipantSummary` um `Meters []CoreMeterSummary{MeteringPoint, Status, ProcessState}`
  erweitert — die nötigen Felder lieferte das deployed Core-Endpoint
  `GET /api/participant` schon, wurden bisher nur weggeworfen
  (verifiziert am 2026-05-19 gegen RC101294).
- `ImportService.CheckActivations` evaluiert pro EEG den `activation_mode`:
  - `participant_active`: heutige Logik — `participant.status == ACTIVE`
  - `any_meter_registration_started`: min. ein Zählpunkt mit
    `processState ∈ {PENDING, APPROVED, ACTIVE}` (Netzbetreiber hat
    EDA-Online-Registrierung mindestens bestätigt)
- Admin-Settings-Editor: Radio-Block "Aktivierungs-Kriterium" mit Erklärtexten
  zu beiden Varianten.
- API: `GET/PUT /api/admin/settings/eeg` um `activationMode` erweitert
  (Patch-Semantik, Enum-Validation).
- Default ist rückwärtskompatibel — kein Bestands-EEG kippt ungewollt um.

**3. Manueller `approved → activated`-Skip-Import (Ausnahmefall)**
- Neue Transition `approved → activated` (zusätzlich zu `approved → imported`
  und `approved → import_failed`). NICHT über generisches `/status` zugänglich
  — nur über dedizierten Endpoint.
- Use-case: Mitglied existiert im eegFaktura-Core bereits (Faktura erlaubt
  kein Löschen) und wurde dort manuell mit den Onboarding-Daten
  überschrieben. Der Onboarding-Antrag muss trotzdem zu `activated` kommen.
- Neuer Endpoint `POST /api/admin/applications/{id}/mark-activated` mit
  Pflicht-Body `{"memberNumber": "..."}`. Validiert: Status muss `approved`
  sein, Mitgliedsnummer muss frei sein (kein Konflikt in der EEG).
- Triggert dieselbe `SendActivationNotification` wie der reguläre Pfad
  (Flag-Check verhindert doppelten Versand).
- Admin-UI: Button "Manuell aktivieren …" auf der Detailansicht einer
  `approved`-Anwendung, öffnet Dialog mit Pflichtfeld Mitgliedsnummer und
  deutlichem Warnhinweis "Nur verwenden wenn Core-Member bereits manuell
  überschrieben".

**Tests:** neue Unit-Tests `TestShouldActivate` (11 Cases: A/B-Modus,
Edge-Cases inkl. Fallback bei unbekanntem Mode-Wert) und
`TestIsValidActivationMode` (Enum-Validator als Source-of-Truth-Gate
zwischen HTTP-Layer und DB-CHECK).

**Docs:** `docs/architecture-diagram.md` (State-Diagramm + Legende),
`CLAUDE.md` (Transitionsliste), `docs/domain-model.md` (neue Spalten),
`docs/api-spec.md` (neues Endpoint, Activation-Modus-Tabelle, EDA-Mapping,
`activationMode` in EEG-Settings-Beispielen).

### Docs — Audit aller `docs/` und `docs/user-guide/` *(2026-05-18)*

Vollständiger Durchgang aller Top-Level-Dokumente und der User-Guide nach
heute deployed Features. Befunde und Fixes:

**User-Guide:**
- `04-admin-applications.md` + `05-admin-status.md`: 5× „In Prüfung" /
  „Zur Prüfung" / „In Prüfung nehmen" → „In Bearbeitung" / „In Bearbeitung
  nehmen" / „Zurück in Bearbeitung" (Status-Filter, Button-Labels,
  Section-Titel). PDF und Feature-Specs bewusst nicht angefasst.
- `06-admin-settings.md`: Neuer Abschnitt **„Zählpunkt-Prefixes (PROJ-52)"**
  mit Beschreibung von Verbraucher-/Einspeisungs-Prefix, Format-Regeln,
  Live-Vorschau, Auto-Pad und Backend-Match-Validation. `bank_name` in der
  Liste „Spezielle konfigurierbare Felder" ergänzt.
- `02-member-registration.md`: Member-Type-Tabelle um `Kleinunternehmer`
  ergänzt + USt.-Hinweis-Spalte. Schritt 5 (Zählpunkte) um neues Layout
  (Richtung+Faktor zuerst, Zählpunkt full-width darunter), Mask-Lock und
  Auto-Pad-Verhalten erweitert. Schritt 7 ergänzt um die heutige
  PROJ-31-Success-Variante („Bitte E-Mail-Postfach prüfen").
- `05-admin-status.md`: Hinweis zur Mail-Footer-Änderung (mailto-Link statt
  Postadresse) und zum vorbefüllten SEPA-Mandat-Datum ergänzt.

**Top-Level-Docs:**
- `PRD.md`: 17 Features (PROJ-33 bis PROJ-49 ohne PROJ-43-Duplikat) +
  PROJ-52 von „In Review" / „Planned" → „Shipped to production".
  PROJ-26 + PROJ-50 in den „On Hold"-Block verschoben.
- `security.md`: `validateMeteringPointPrefixMatch` (PROJ-52) zu den
  security-sensitive Bereichen unter `internal/application/` ergänzt.
- `api-spec.md`, `domain-model.md`, `architecture.md`, `import-mapping.md`,
  `operations.md`, `open-questions.md`, `keycloak-setup.md`: keine
  Anpassungen nötig — wurden bei den jeweiligen Feature-Commits mitgepflegt.

**Mail-Templates + PDF-Generatoren:**
- Audit bestätigt: alle `{{.Field}}`-Referenzen matchen die Go-Structs,
  Footer-Texte zeigen `EEGContactEmail` als mailto-Link, Zählpunkte werden
  in der 2-6-5-20-Gruppierung gerendert, SEPA-MandateDate wird in beiden
  PDF-Varianten oberhalb der Unterschriftslinie vorbefüllt. Keine Fixes
  erforderlich.

**Screenshots in `docs/user-guide/images/`:**
- Folgende Screenshots zeigen veraltete UI-Texte und sollten bei nächster
  Gelegenheit neu aufgenommen werden (manuell, kein Headless-Setup im
  CI): `admin-filter-panel.png` („In Prüfung"), `admin-status-actions.png`
  („In Prüfung nehmen" / „Zurück in Prüfung"), `admin-application-detail-1.png`
  („zur Prüfung bereit"), `admin-settings-eeg.png` (fehlender Zählpunkt-
  Prefix-Block), `register-form-metering-points.png` (neues Layout +
  Prefix-Lock).

### Reviews — Code-Review + Security-Review *(2026-05-18)*

Nach dem Docs-Audit zusätzlich:

- **Code-Review**: Cross-Check aller Mail-Templates, PDF-Generatoren und
  HTTP-Handler gegen api-spec.md, domain-model.md und die heute deployed
  Features. Vier parallele Explore-Agents (Mail+PDF, API, User-Guide,
  Top-Level-Docs) — alle Mail-Felder konsistent, kein undokumentierter
  Endpoint, keine veralteten Surface-Definitionen. Einziger Hinweis:
  `docs/docs.go` (Swagger-UI-Generierung) ist seit PROJ-28 nicht regeneriert
  — `api-spec.md` ist Source of Truth und aktuell, Swagger-UI hinkt
  nach. Vor nächstem Release `swag init -g cmd/server/main.go` ausführen
  (nicht-blockierend, optional).
- **Security-Review**: PROJ-52 Prefix-Match-Validation greift als
  defense-in-depth zusätzlich zur Frontend-Mask, DB-CHECK-Constraint
  (`^AT[0-9A-Z]{0,31}$`) schließt den letzten Layer. Normalisierung
  (Whitespace + Dots + Hyphens, uppercase) wird vor Validierung
  ausgeführt — kein Bypass via Eingabe-Tricks. Keine Auth-Boundaries
  geändert, keine neuen öffentlichen Endpoints, kein Geheimnis im Code.
  `app.MandateDate` ist eine reine Tagesinformation (keine PII-Eskalation).
  Bestehende Snyk-Scans + govulncheck weiter grün.

### Geändert — Zählpunkt-Mask auf offizielle Gruppierung 2-6-5-20 *(2026-05-17)*

Recherche zur E-Control / MeteringCode-Spec ergab, dass die offizielle
vierteilige Struktur der Zählpunktbezeichnung in Österreich
`AT | Netzbetreibernummer (6) | PLZ (5) | Zählpunktnummer (20)` lautet.
Die bisherige UI-Mask `2-6-5-12-8` war willkürlich.

Mask im Mitgliederformular auf die offizielle Aufteilung umgestellt
(`AT 000000 00000 [20 Stellen]`). Visuelle Änderung, keine Auswirkung
auf Validierung oder gespeicherte Daten (33 Stellen unverändert).

Vorbereitung für PROJ-52 (konfigurierbarer Prefix pro Richtung + Auto-Pad
+ alphanumerischer letzter Block — Spec angelegt, Implementierung folgt).

### Geändert — Speichersteuerung-Frage + Batterie-Gruppierung (PROJ-49 follow-up) *(2026-05-17)*

Neue Mitglied-Frage „Speichersteuerung im Sinne der EEG vorstellbar?" auf
PV-Erzeuger-Zählpunkten. Gleichzeitig UI-Refactoring: die bisher einzeln
sichtbaren Speicher-Felder werden hinter einer Master-Checkbox gruppiert.

**Datenmodell (Migration 000044):**
- `metering_point.battery_control_acceptable` BOOLEAN NULL — Mitglied-Antwort
  Ja/Nein. Service-Layer cleart das Feld, wenn kein PV-Zählpunkt oder wenn
  das Mitglied keine Batterie-Parameter angegeben hat.

**Sichtbarkeitsregeln:**
- Nur bei `direction='PRODUCTION'` + `generation_type='pv'`
- Nur wenn `battery_size_kwh` ODER `inverter_manufacturer` befüllt ist
- PROJ-8-konfigurierbar via field_config (`battery_control_acceptable`,
  Default `hidden`)

**API:**
- `meteringPoints[].batteryControlAcceptable` in Public-, Admin-,
  Externe-API (Request + Response).
- Required-Validierung greift nur, wenn das Mitglied tatsächlich Batterie-
  Daten gesetzt hat — sonst entfällt die Frage komplett.

**Frontend (UX-Verbesserung):**
- Neuer `BatteryBlock` in `metering-point-fields.tsx` mit Master-Checkbox
  „Batteriespeicher vorhanden". Nach Aktivieren erscheinen drei gruppierte
  Felder darunter: Größe Batterie (kWh), Hersteller Wechselrichter,
  Speichersteuerung im Sinne der EEG vorstellbar?.
- Deaktivieren der Master-Checkbox cleart alle drei Felder.
- Beim Reload wird der Toggle-Zustand aus dem Vorhandensein eines der drei
  Werte abgeleitet (Pattern aus `DeviatingAddressBlock`).
- `GenerationBlock` schlanker: Batterie-Felder dort entfernt, jetzt nur
  noch generation_type + PV-Leistung + Einspeise-Forecast + Einspeiselimit.

**Mail-Templates:** `FormatGenerationLine` rendert die Antwort wenn gesetzt
als zusätzliches Segment, z. B. `…, Speichersteuerung im Sinne der EEG: Ja`.

### Geändert — Energie-Felder pro Zählpunkt (PROJ-49) *(2026-05-17)*

Refactoring: 4 Energie-Felder wandern von `application` auf `metering_point`,
1 neues Feld kommt dazu.

**Datenmodell (Migration 000043):**
- `metering_point` bekommt 6 neue Spalten:
  `consumption_previous_year`, `consumption_forecast`, `feed_in_forecast`,
  `pv_power_kwp`, `feed_in_limit_present`, `feed_in_limit_kw`.
- `application` verliert 4 Spalten:
  `consumption_previous_year`, `consumption_forecast`, `feed_in_forecast`, `pv_power_kwp`.
- Bestandswerte werden verworfen (Entscheidung Owner 2026-05-17: nur In-Review-Anträge betroffen).
- Alte `field_config`-Einträge mit den 4 Namen werden gelöscht — EEGs reaktivieren bewusst.

**Sichtbarkeit (Service-Layer enforced):**
- `consumption_*` nur bei `direction='CONSUMPTION'`.
- `feed_in_forecast` nur bei `direction='PRODUCTION'` (alle Erzeugungsformen).
- `pv_power_kwp` / `feed_in_limit_*` nur bei `direction='PRODUCTION'` + `generation_type='pv'`.
- `feed_in_limit_kw` nur wenn `feed_in_limit_present=TRUE`.

**Neues Feld:** Einspeiselimit (Bool „vorhanden" + optional kW-Wert).
Manche Netzanschlüsse sind leistungstechnisch beschränkt (z. B. „nur 70 % der PV einspeisbar"); EEG braucht diese Info für die Planung.

**API:**
- `POST /api/public/applications` Request: 4 Felder wandern von Top-Level in `meteringPoints[]`-Einträge; 2 neue `feedInLimitPresent` / `feedInLimitKw`.
- `GET /api/admin/applications/{id}` Response: gleiche Bewegung in der Antwort.
- `PUT /api/admin/applications/{id}` Body: 4 Top-Level-Felder werden ignoriert.
- Externe API (`POST /api/external/v1/applications`) analog.

**Mail-Templates + PDF:**
- `FormatGenerationLine` rendert pro Zählpunkt jetzt:
  - CONSUMPTION: `Verbrauch Vorjahr X kWh, Prognose Y kWh`
  - PRODUCTION + pv: `PV 9,9 kWp, Prognose 6000 kWh/J, Speicher 10,5 kWh (Fronius), Einspeiselimit 7,0 kW`
- Die 4 Application-Level-Felder erscheinen nicht mehr im „Zusätzliche Informationen"-Block.

**Frontend:**
- `metering-point-fields.tsx`: 6 neue Felder mit Sichtbarkeitsbedingungen + neuer `ConsumptionDetailsBlock`.
- `registration-form.tsx`: 4 Application-Level-Felder + zugehörige Defaults/Validation/Payload entfernt; per-MP-Payload um die neuen Felder erweitert.
- `admin-eeg-settings-editor.tsx`: Felder wandern automatisch in die „Zählpunkt-Felder"-Sektion (via `CONFIGURABLE_FIELDS.meteringPoint`).

### Geändert — Register-Dialog + Admin-Settings: Audit-Fixes *(2026-05-17)*

Vollständiger Inhalts-Audit analog zum Mail-Template-Audit. 51 Findings
in drei Wellen abgearbeitet.

**Welle A (kritisch):**
- `admin-legal-documents-editor.tsx` `handlePolicyToggle` Payload-Fix:
  vollständiger Settings-Snapshot wird mitgesendet — vorher fehlten
  `sepaMandateAtImport` (PROJ-48), `cooperativeSharesEnabled`,
  `cooperativeRequiredShares`, `cooperativeShareAmountCents` (PROJ-37),
  d.h. der Datenschutz-Toggle überschrieb diese Settings stillschweigend
  mit Defaults. **Echter Datenverlust-Pfad behoben.**
- `admin-eeg-settings-editor.tsx` B2B-Label-Korrektur: „für Unternehmen
  und **Gemeinden**" (vorher „Unternehmen und Vereine" — fachlich falsch
  nach PROJ-48; Vereine bekommen kein B2B-Auto-Mandat).
- `admin-eeg-settings-editor.tsx` SEPA-Haupt-Toggle umformuliert auf
  „SEPA-Mandat von der EEG bereitstellen" (vorher: „dem Willkommensmail
  anhängen", was nach PROJ-48 nicht mehr stimmt). Beim
  at-import-Sub-Toggle ausführlicher Hilfetext.
- `registration-form.tsx`: Neuer Hinweistext bei
  `sepaMandateEnabled=true` UND `sepaMandateAtImport=true` — Member
  weiß nun, dass das Mandat erst mit der Beitrittsbestätigung kommt.
  `RegistrationConfig` (Backend + Frontend-Type) um
  `sepaMandateAtImport` erweitert.
- Veraltete Texte „Die zentrale Datenschutzerklärung wird über
  Servereinstellungen konfiguriert" in `app/admin/settings/page.tsx`
  und `admin-legal-documents-editor.tsx` korrigiert — der per-EEG
  Toggle existiert seit PROJ-18.
- Superuser-URL-Hinweis (`/admin/settings?rc=…`) in `settings/page.tsx`
  ersetzt — Code las den URL-Param nie aus, die Anleitung funktionierte
  nicht.

**Welle B (Konsistenz):**
- **`NETWORK_OPERATOR_AUTH_TEXT`** als Konstante in `src/lib/api.ts`
  extrahiert. Public-Form rendert nun aus der Konstante (Single Source
  of Truth, Spec/UI-Drift verhindert).
- `SyncedField` jetzt mit echtem disabled `<Input>` statt visuell
  ähnlichem `<div>` — A11y-Fix für Screen Reader.
- Genossenschaftsanteile-Sichtbarkeits-Bug: Block wird jetzt gerendert,
  sobald `cooperativeSharesEnabled=true` (auch wenn `amountCents=null`).
  Vorher wurde der ganze Block stillschweigend ausgeblendet — Member
  scheiterte beim Submit am Backend-400.
- Du→Sie im Admin-Editor (zwei Stellen in
  `admin-eeg-settings-editor.tsx` mit „Klicke" → „Klicken Sie").
- „Bestätigungs-Mail" → „Eingangsbestätigung" (PROJ-31-Erläuterung).
- Mitglieds­typ-Label „Gemeinde / öffentl. Körperschaft" → ausgeschrieben.
- `orgLabel` für Kleinunternehmer ergänzt („Firmenbezeichnung" statt
  fallback „Firmenname").
- `Aktiv am (Beitrittsdatum)`-Hilfetext: `<p>` → Popover (Frontend-
  Regel-Compliance, Pattern aus `.claude/rules/frontend.md`).

**Welle C (Kosmetik):**
- Unicode-Pfeile `▲`/`▼`/`▴`/`▾` → lucide `ChevronUp`/`ChevronDown`
  in `admin-legal-documents-editor.tsx` + `admin-eeg-settings-editor.tsx`.
- „+ Dokument hinzufügen" → lucide `PlusCircle`-Icon-Pattern.
- Placeholder `"Richtung"` aus `metering-point-fields.tsx` entfernt
  (wurde nie sichtbar, Wert ist immer initial gesetzt).
- `z.B.` → `z. B.` Typografie-Fix in `admin-api-key-editor.tsx`.
- Doppelter Kommentar zur Metering-Points-Karte in `registration-form.tsx`
  entfernt.

Backend + Tests grün.

### Geändert — Mail-Templates: Audit-Fixes + Orphan-Cleanup *(2026-05-17)*

Vollständiger Inhalts-Audit aller 8 Mail-Templates + Behebung der
gefundenen Inkonsistenzen.

**Welle 1 (kritisch):**
- `application_imported_member.html` um PROJ-48-Pfad erweitert: neuer
  `HasMandateAttachment`-Flag in `importedTemplateData` triggert einen
  zusätzlichen Block „Ihr SEPA-Lastschriftmandat" mit Signatur-Anleitung
  (Ausdruck oder ID-Austria-App) — wird gerendert, wenn beim Import ein
  Basis-Mandat angehängt wurde (PROJ-48-`sepa_mandate_at_import=true`-
  Pfad für `einzugsart=core`). B2B-Block (PROJ-47) bleibt parallel.
- `application_imported_eeg.html` analog: zusätzlicher Hinweis-Block,
  wenn das Mitglied das Basis-Mandat mit ausgefüllter Mandatsreferenz
  bekommen hat — Admin weiß, dass auf unterschriebene Rücksendung
  gewartet werden muss, bevor Lastschriften eingezogen werden.
- `approvalSepaMandateType` und `resolveSepaMandateType` an PROJ-48
  angepasst: SEPA-Variante richtet sich jetzt allein nach
  `app.einzugsart` (Auto-Logik via Mitgliedstyp + useCompanySEPAMandate
  entfernt — entsprach nicht mehr dem neuen Default-Core-Workflow).
- `application_approved_eeg.html` **gelöscht** (Orphan seit PROJ-46
  Stage B). `SendApprovalEmail`-Method aus MailService-Interface,
  NoOpMailService und SMTPMailService entfernt. Zugehörige Tests
  (`TestSendApprovalEmail*`, `TestApprovalTemplate*`) entfernt.
  Auch der `approvalTpl`-Field und `approvedEEGTemplateData`-Typ weg.

**Welle 2 (Inhalte + Konsistenz):**
- `application_submitted_member.html`:
  - SEPA-Tabellenzeile vereinfacht (vorher mit verschachtelter
    SEPAMandateEnabled/Accepted-Logik): zeigt jetzt klare drei
    Varianten — „Mandat als PDF-Anhang", „Mandat wird mit
    Beitrittsbestätigung übermittelt" (PROJ-48-Pfad), oder
    „Online-Zustimmung erteilt"
  - Redundanter Schluss-Text entfernt (war doppelt mit
    Confirmation-Box am Anfang)
- `application_submitted_eeg.html`: zusätzliche Zeile
  „E-Mail bestätigt am: …" wenn PROJ-31 aktiv ist — macht den
  Zeitversatz zwischen Submit und EEG-Mail-Versand transparent.
  `EmailConfirmedAt`-Feld in `eegTemplateData` ergänzt.
- `application_needs_info_member.html`: Anleitung erweitert um den
  Hinweis, dass die EEG den ursprünglichen Antragslink erneut zusenden
  kann, wenn das Mitglied Angaben direkt im Form korrigieren möchte
  (vorher nur „E-Mail antworten").
- `application_activated_member.html`: realistischere Formulierung —
  „formal aktiv, tatsächliche Teilnahme startet sobald der
  Netzbetreiber freigeschaltet hat" (vorher überoptimistisch
  „ab sofort am Sharing teil"). Plus erste-Abrechnungs-Hinweis.
- **Konsistente Signaturen quer durch alle Member-Templates**:
  „Ihr Team von {EEG-Name}" mit Fallback „Ihre Energiegemeinschaft"
  (vorher mal „Ihr eegFaktura-Team"). Der eegFaktura-Brand bleibt
  nur im Footer als Erzeuger-Hinweis.
- Alle Member-Templates beginnen einheitlich mit
  „Sehr geehrte/r {Vorname} {Nachname}".

### Neu — PROJ-48: SEPA-Default-Core + konfigurierbares Mandat-Timing + B2B-Hinweis *(2026-05-17)*

Drei zusammenhängende Änderungen am SEPA-Workflow:

1. **Default-Einzugsart immer `core`.** Die Auto-Logik „Firmenlastschrift
   bei Mitgliedstyp company/association mit useCompanySEPAMandate=true"
   im Submit-Pfad **entfällt ersatzlos**. Submit-Mail enthält jetzt
   immer das Basis-Mandat (oder kein PDF, je nach Setting 3). Admin
   kann die Einzugsart per Antrags-Edit weiterhin auf `b2b` umstellen.
2. **B2B-Hinweis-Block in der Submit-Mail** bei Mitgliedstyp
   `company` und `municipality` — kurzer Satz: „Falls statt der
   Basislastschrift eine Firmenlastschrift (SEPA B2B) notwendig ist,
   meldet sich {EEG-Name} mit den notwendigen Unterlagen bei Ihnen."
   (Verein bewusst ausgenommen — User-Wunsch.)
3. **Neues EEG-Setting `sepa_mandate_at_import`** (Default `FALSE` =
   heutiges Verhalten). Bei `TRUE` wird das SEPA-Mandat-PDF NICHT
   beim Submit, sondern erst beim Import mit eingedruckter
   Mandatsreferenz = Mitgliedsnummer versendet — auch für `core`
   (bislang nur PROJ-47-Pfad für `b2b`).

Architektur-Hintergrund: PROJ-48-Setting löst den Konflikt „digital
signiertes Dokument darf nicht mehr modifiziert werden". Wenn die EEG
digitale Mandate verwendet und Mandatsreferenz im Dokument verlangt
wird, ist der at-import-Pfad der einzige saubere Weg. Volltext zur
Digital-Signatur-Diskussion: `docs/open-questions.md` OQ-6 (neu).

- Migration `000042_sepa_mandate_at_import`
- `RegistrationEntrypoint.SEPAMandateAtImport` + Repo + Settings-Endpoint
- Submit-Mail-Logik: kein PDF bei `sepa_mandate_at_import=true`,
  ansonsten immer Basis-Variante (Firmenlastschrift-Auto-Wahl entfernt)
- Import-Mail-Logik: zusätzlicher Basis-Mandat-Anhang bei
  `einzugsart=core` + `sepa_mandate_at_import=true` (PROJ-47-B2B-Pfad
  unverändert)
- Mail-Template (`application_submitted_member.html`): B2B-Hinweis-
  Block conditional auf neuem `ShowB2BHint`-Flag
- Frontend Admin-Settings-Editor: neuer Switch „SEPA-Mandat erst beim
  Import senden" inkl. Tooltip-Hinweis auf Digital-Signatur-Use-Case
- OQ-6 in `docs/open-questions.md` ergänzt: vollständige Behandlung der
  Architektur-Implikationen einer digitalen Mandat-Signatur

### Neu — PROJ-47: B2B-SEPA-Firmenlastschrift-Mandat mit Mandatsreferenz beim Import *(2026-05-17)*

Schließt die in PROJ-46 erkannte Lücke: ein B2B-Antragsteller bekam
zwar bei Submission ein Firmenlastschrift-PDF, aber ohne die später
vergebene Mitgliedsnummer als Mandatsreferenz — die B2B-Bank verlangt
diese aber ausdrücklich. Mit PROJ-47 wird beim Import ein **zweites
Firmenlastschrift-Mandat-PDF mit eingedruckter Mandatsreferenz =
Mitgliedsnummer** generiert und an die Member-Mail (+ EEG-Kopie)
angehängt, das der Member ausdrucken und an seine Hausbank
weiterreichen kann.

- `pdf.SEPAMandateData`: neues optionales Feld `MandateReference`.
  Beide PDF-Renderer (Generate / GenerateCompany) drucken den Wert
  inline statt des Platzhalters „wird von … ausgefüllt".
- `mail.Sender` erweitert um `Attachment`-Struct +
  `SendWithAttachments(...)` für Multi-Anhang-Versand. Bestehende
  Single-Attachment-API bleibt und delegiert intern.
- `SendImportedNotification` nimmt zusätzlich `b2bMandatePDF []byte`;
  bei non-empty wird das B2B-Mandat als zweiter PDF-Anhang verschickt
  (Dateiname `sepa-firmenlastschrift-mandat-<Mitgliedsnr>.pdf`).
- `AdminApplicationService` bekommt
  `sepaMandateGenerator pdf.SEPAMandateGenerator` als Dependency.
  Beim Post-Import-Notification wird bei `einzugsart=b2b` der B2B-
  Mandat-Generator aufgerufen, Debtor-Name aus CompanyName,
  Mandatsreferenz aus MemberNumber, Logo aus EEG-Cache.
- Mail-Template `application_imported_member.html` ergänzt um Hinweis
  auf den zweiten PDF-Anhang im b2b-Block.
- Best-Effort bei B2B-PDF-Fehlern (Log + ohne 2. Anhang weiter); die
  Hauptmail mit Beitrittsbestätigung geht in jedem Fall raus.

### Neu — PROJ-46 Stage D: Activation-Check via Core *(2026-05-17)*

Admin-getriggerter Batch-Check ersetzt das ursprünglich geplante Cron-
Polling (User-Entscheidung B). Button „Aktivierung im Core prüfen" in
der Antrags-Übersicht (`/admin/applications`) ruft einen neuen Endpoint
auf, der alle `ready_for_activation`-Anträge der eigenen Tenants gegen
den eegFaktura-Core abgleicht und ACTIVE-Mitglieder automatisch auf
`activated` setzt.

- Neuer Endpoint `POST /api/admin/applications/check-activation` (kein
  ID, batch). Tenant-Scope kommt aus den JWT-Claims (Superuser ohne
  Filter, sonst nur eigene RCs).
- `coreclient.CoreParticipantSummary` um `Status string` erweitert
  (Werte: `NEW`, `PENDING`, `ACTIVE`).
- `ImportService.CheckActivations`: gruppiert Kandidaten per Tenant,
  ruft Core `GET /participant` einmal pro Tenant, mappt per
  `target_participant_id`, transitioniert bei `Status == "ACTIVE"`
  via guarded UpdateStatusAdminTx + Status-Log-Eintrag mit Actor
  `system:activation-check`.
- `ApplicationRepository.ListReadyForActivation(allowedRCNumbers)`
  liefert die minimalen Felder für den Cross-Reference.
- Frontend-Button in `applications-page-content.tsx`: zeigt Toast mit
  Ergebnis (`X von Y auf Aktiviert gesetzt`), refresht danach die Liste.
  Bei 0 Treffern oder Fehlern entsprechende Info/Warning-Toasts.

### Neu — PROJ-46 Stage C: Admin-UI für Post-Import-Stati *(2026-05-17)*

- `ApplicationStatus`-Typ um drei neue Werte erweitert
  (`awaiting_bank_confirmation`, `ready_for_activation`, `activated`)
- `AdminStatusBadge`: neue Farben — Amber für „Warte auf Bank-Bestätigung",
  Cyan für „Bereit zur Aktivierung", tiefes Smaragd für „Aktiviert"
- `admin-filter-panel`: die drei neuen Stati erscheinen als Filter-Option
- `admin-status-actions`: drei neue Block-Layouts:
  - `awaiting_bank_confirmation`: prominente Amber-Hinweisbox „Warte auf
    Bank-Bestätigung" + Buttons „Bank-Bestätigung erhalten", „Zurück in
    Prüfung", „Import zurücksetzen"
  - `ready_for_activation`: Buttons „Als aktiv markieren" (grün),
    „Zurück in Prüfung", „Import zurücksetzen"
  - `activated`: rein informativer Text — keine weiteren Aktionen
    (strikter Endzustand)
- Reset-Import-Dialog-Warning erweitert: erwähnt jetzt explizit, dass
  `activated`-Anträge nicht resetbar sind und dass Mitgliedsnummer +
  Bank-Bestätigung mitgelöscht werden

### Neu — PROJ-46 Stage B: PDF-Timing + Member-Mails nach Import + Aktivierung *(2026-05-17)*

PDF-Generierung wandert von `→ approved` zum Import-Zeitpunkt (wenn die
Mitgliedsnummer steht — Voraussetzung für die B2B-SEPA-Mandatsreferenz):

- Drei neue Mail-Templates: `application_imported_member.html`,
  `application_imported_eeg.html`, `application_activated_member.html`
- Neue MailService-Methoden `SendImportedNotification` (Member + EEG)
  und `SendActivatedNotification` (Member only). NoOpMailService und
  Interface entsprechend erweitert.
- `SendImportedNotification` schickt PDF-Anhang an Member und Kopie an
  EEG-Contact. Beide Templates zeigen bei `einzugsart=b2b` einen
  Zusatz-Hinweis: Member bekommt die Anleitung zur Hausbank-Pre-
  Notification, EEG sieht den Hinweis „Auf Bank-Bestätigung warten —
  bitte auf ready_for_activation weiterschalten".
- Neue Service-Methode `AdminApplicationService.SendPostImportNotification(appID)`
  bündelt die heavy-Loads (App, MPs, Status-Log, Consents, Entrypoint,
  FieldConfig, Logo) + PDF-Build + Mail-Send. Aus dem HTTP-Import-
  Handler nach `importService.Import()`-Erfolg in Goroutine aufgerufen
  (best-effort, blockiert nicht die HTTP-Response).
- `→ approved`-Trigger im `ChangeStatus` entfernt — die alte
  Approval-Mail an EEG (ohne Mitgliedsnummer im B2B-Mandat) entfällt
  komplett, ersetzt durch den Import-Trigger.
- `→ activated`-Trigger ergänzt: schickt Welcome-Mail an Member über
  `SendActivatedNotification` in Goroutine.
- `SendApprovalEmail` bleibt auf dem MailService-Interface (für Test-
  Kompatibilität), wird aus Produktiv-Code aber nicht mehr aufgerufen
  (Deprecation-Kommentar gesetzt). `application_approved_eeg.html`
  bleibt vorerst im Repo (kein aktiver Send-Pfad mehr).
- Prometheus-Counter neu: `eeg_imported`, `member_imported`,
  `member_activated` (success/failed-Labels wie bei bestehenden Mails).

### Neu — PROJ-46 Stage A: Stati für Import-Nachbereitung *(2026-05-17)*

Erste Stage: DB + Backend-Übergänge + Reset-Erweiterung. Mails (Stage B),
Admin-UI (Stage C) und Activation-Check-Button (Stage D) folgen separat.

- Migration `000041_post_import_statuses`: drei neue Status-Werte
  (`awaiting_bank_confirmation`, `ready_for_activation`, `activated`),
  CHECK-Constraint erweitert, zwei neue Audit-Timestamps
  (`bank_confirmed_at`, `activated_at`)
- Import-Service: nach erfolgreichem `→ imported` läuft automatisch
  ein Branch — `einzugsart=b2b` ⇒ `awaiting_bank_confirmation`,
  sonst direkt `ready_for_activation`. Status `imported` existiert
  nur Millisekunden als Landing-Zone für die Import-Bookkeeping.
- `adminTransitions`-Map: neue Übergänge für die zwei mittleren
  Stati (manuelle Weiterschaltung + Rückwärts auf `under_review`).
  `activated` ist strikter Endzustand, keine Transitions hinaus.
- `UpdateStatusAdminTx`: COALESCE-Pattern um `bank_confirmed_at`
  und `activated_at` erweitert; Service stempelt die Timestamps
  beim jeweiligen Übergang.
- Reset-Import (PROJ-30) erweitert: Reset ist jetzt auch aus
  `awaiting_bank_confirmation` und `ready_for_activation` möglich
  (zurück auf `approved`). Aus `activated` **nicht** — strikter
  Endzustand, Deaktivierung muss im Core erfolgen. Reset cleart
  zusätzlich `bank_confirmed_at` + `activated_at` für sauberen Retry.
- CLAUDE.md Status-Sektion aktualisiert (3 neue Stati + 7 neue
  Transition-Einträge dokumentiert).

### Neu — PROJ-45: Erzeugungsform + Batterie + typabhängige Sichtbarkeit *(2026-05-17)*

Drei zusammenhängende Erweiterungen rund um Erzeugungs-Zählpunkte:

1. **Erzeugungsform pro PRODUCTION-Zählpunkt** — neues Pflichtfeld
   `generation_type` mit den Werten `pv`/`hydro`/`wind`/`biomass`,
   Default `pv`. Bestandsdaten werden migrationsweise auf `pv` gesetzt.
   DB-CHECK erzwingt: CONSUMPTION ⇒ NULL, PRODUCTION ⇒ einer der vier Werte.
2. **Batterie + Wechselrichter pro PV-Zählpunkt** — zwei neue PROJ-8-
   konfigurierbare Felder `battery_size_kwh` (NUMERIC) und
   `inverter_manufacturer` (Freitext). Default `hidden`; werden nur
   gerendert wenn EEG-Konfig aktiv UND `generation_type='pv'`.
3. **Typabhängige Sichtbarkeit der App-Level-Energie-Felder** —
   Verbrauchs-Felder (Wärmepumpe, E-Auto, Verbrauch …) erscheinen nur
   wenn der Antrag mindestens einen CONSUMPTION-Zählpunkt hat;
   Erzeugungsfelder (PV-Leistung, Einspeisung Prognose) nur bei
   PRODUCTION-Zählpunkten. Frontend rendert live; Backend cleart die
   Felder beim Speichern (`clearAppFieldsByMpTypes`) und gated den
   required-Check entsprechend.

Migration `000040_generation_type_and_battery` läuft als Pre-Upgrade-
Job automatisch beim nächsten Deploy.

- Service-Layer-Normalisierung (`normalizeMeteringPointGeneration`):
  CONSUMPTION ⇒ generation_type/battery/inverter NULL; PRODUCTION ohne
  expliziten Typ ⇒ `pv`; non-pv ⇒ battery/inverter NULL — Schutz gegen
  forged Clients und konsistente Persistenz
- Admin-Edit-Form (`admin-edit-form.tsx`): Erzeugungsform-Select pro
  PRODUCTION-Zählpunkt + Batterie/Hersteller-Inputs bei PV
- Admin-Detail-Tabelle: neue Spalte „Erzeugung" mit kompakter
  Darstellung „PV, Speicher 10,5 kWh (Fronius)"
- Approval-PDF: zusätzliche Zeile pro PRODUCTION-Zählpunkt
  („Erzeugung: PV, Speicher 10 kWh (Fronius)")
- Mail (Member + EEG): Erzeugungs-Zeile in der Zählpunkt-Tabelle
- Excel-Export: drei neue Spalten am Ende der Zeile
  (`Erzeugungsform`, `Größe Batterie (kWh)`, `Hersteller WR`) —
  eegFaktura-Importer ignoriert unbekannte Spalten, kein Import-Risiko
- `validateConfigurableRequiredFields` neue Signatur mit `mps`-Parameter
  für typabhängiges Gating; Unit-Tests passen `nil` (kein Gating).

### Neu — PROJ-44: Netzbetreiber-Vollmacht (per-EEG konfigurierbar) *(2026-05-17)*

Manche Netzbetreiber (z.B. Netz OÖ) verlangen eine separate Vollmacht
des Mitglieds, damit die EEG in dessen Namen mit dem Netzbetreiber
verhandeln darf. Die Vollmacht ist nicht Teil der EEG-Mitgliedschafts­
zustimmung und nicht bei jeder EEG nötig — daher als neues
konfigurierbares Feld (PROJ-8-Pattern, Default `hidden`).

- Migration `000039_network_operator_authorization`: zwei Spalten auf
  `application` — `network_operator_authorization BOOLEAN NOT NULL DEFAULT FALSE`
  + `network_operator_authorization_at TIMESTAMPTZ NULL`
- Neues konfigurierbares Feld `network_operator_authorization` —
  EEGs mit Anforderung setzen es auf `required`, Bestands-EEGs bleiben
  auf `hidden` (kein Sichtbarkeitswechsel ohne Admin-Aktion)
- Verbindlicher Wortlaut der Vollmacht im Frontend (Checkbox-Label),
  Wortlaut versioniert über Code-Commit (keine DB-Versionierung — YAGNI)
- Service-Layer: `_at` wird automatisch auf `NOW()` gesetzt, wenn das
  Flag von FALSE auf TRUE wechselt; `clearNetworkAuthIfHidden` schützt
  vor forged Clients, die das Flag für EEGs mit `hidden`-Config setzen
- Approval-PDF + Member-/EEG-Mail: rendern „Netzbetreiber-Vollmacht
  erteilt: Ja" über bestehenden `buildConfigurableFields`-Pfad;
  FALSE wird unterdrückt (Default für Bestandsanträge)
- Admin-Detail: zeigt Vollmacht + Erteilungs-Timestamp, wenn erteilt
- Excel-Export: bewusst **nicht** befüllt — eegFaktura-Importer­spalten­
  struktur kennt das Feld nicht, Audit-Trail liegt in DB + PDF + Mail

### Geändert — Node-Runtime auf Node 22 LTS gebumpt + automatischer EOL-Check *(2026-05-17)*

Node 20 ist seit 30. April 2026 End-of-Life — keine neuen Security-Patches
mehr. Aktualisiert auf Node 22 LTS (Support bis April 2027), minimaler
Versions-Sprung mit geringstem Regressions-Risiko.

- `Dockerfile.frontend`: 3× `node:20-alpine` → `node:22-alpine`
- `.github/workflows/ci.yml` + `snyk.yml`: `node-version: '20'` → `'22'`
- `package.json`: `@types/node ^20` → `^22` (npm install regeneriert das Lock)
- `dependabot.yml`: Filter für `@types/node` Major-Bumps bleibt aktiv —
  bei nächstem Runtime-Sprung manuell nachziehen
- Neuer Workflow `.github/workflows/eol-check.yml`: läuft monatlich
  (`cron: '0 6 1 * *'`), fragt endoflife.date für **Node**, **Go**,
  **PostgreSQL** und öffnet GitHub-Issues sobald eine Komponente
  innerhalb von 60 Tagen EOL erreicht oder bereits EOL ist. De-dupliziert
  via offene `eol-check`-Issues, sodass kein monatliches Spamming
- Nach jedem Upgrade muss der `cycle`-Eintrag im EOL-Workflow auf die
  neue Major-Version nachgezogen werden (siehe Inline-Kommentar)

### Neu — PROJ-40: EEG-Umzuordnung eines Antrags im Review *(2026-05-17)*

Wenn ein Mitglied über den falschen RC-Link der EEG A registriert hat,
aber eigentlich zur EEG B gehört, kann der Admin den Antrag direkt
umordnen — ohne Re-Submit durch das Mitglied.

- Neuer Endpoint `POST /api/admin/applications/{id}/reassign-eeg`
- **Tenant-Check beidseitig:** Admin muss für Quelle UND Ziel autorisiert
  sein (oder Superuser); sonst 403
- **Reassignable nur in aktiver Review-Phase:** `submitted`,
  `email_confirmed`, `under_review`, `needs_info`. Anything else → 409
- **Neue Referenznummer** wird über den per-EEG-Counter (PROJ-35) der
  Ziel-EEG vergeben, damit die Member-facing-ID zur neuen EEG passt
- **Audit-Trail:** status_log-Entry mit Status unverändert + Reason +
  `[system] previous rc_number=...` + `[system] previous reference_number=...`
- **Repo-Guard** (defense-in-depth): `UpdateRCNumberTx` validiert
  `WHERE id=$ AND rc_number=$expected AND status IN (...)` — bei 0 Rows
  ErrConflict
- **Frontend**: Button „EEG umzuordnen" im Statusaktionen-Block, sichtbar
  nur wenn der Admin ≥ 2 EEGs verwaltet. Dialog mit Dropdown der Ziel-EEGs +
  Begründung + Hinweis-Block auf die neue Referenznummer
- **Out-of-Scope (V1):** Bulk-Reassign, Member-Mail, Re-Validierung von
  Cooperative-Shares / Field-Config / Email-Confirmation-Setting

### Neu — PROJ-42: E-Fahrzeug-Detailerfassung *(2026-05-17)*

Das bestehende `electric_vehicle`-Ja/Nein wird ergänzt um zwei optionale
Detail-Felder, die für die EEG-Lastprofil-Optimierung relevant sind:

- `electric_vehicle_count` (INT) — Anzahl der E-Fahrzeuge im Haushalt
- `electric_vehicle_annual_km` (INT) — geschätzte Gesamt-Jahreskilometer

Beide Felder folgen dem PROJ-8-Configurable-Fields-Pattern: pro EEG
einstellbar (default `hidden`). Im Public-Form werden sie **nur**
angezeigt, wenn (a) die EEG sie aktiviert hat UND (b) der Bewerber
„Ja" beim E-Auto angekreuzt hat. Service-Layer cleart beide Werte
serverseitig auf NULL falls `electric_vehicle != true` (kein DB-CHECK,
sondern Service-Gate `clearEVDetailsIfDisabled`).

Mail (Member + EEG), Approval-PDF, Excel-Export, Admin-Detail werden
über die bestehende Configurable-Fields-Pipeline automatisch versorgt
— sobald die Felder konfiguriert sind, erscheinen sie im
„Zusätzliche Informationen"-Block.

Migration: `db/migrations/000038_ev_details.up.sql`.

### Geändert — PROJ-41 + PROJ-43: Mail-Versand jetzt hard-fail *(2026-05-17)*

Der initiale Best-Effort-Goroutine-Versand wurde umgestellt auf:

- **Synchron + pre-commit**: rejected/needs_info-Mail wird gerendert und
  versendet, BEVOR `tx.Commit()` läuft. Bei Fehler greift `defer tx.Rollback()`
- **Hard-fail**: Mail-Fehler → Statuswechsel wird NICHT persistiert + API
  antwortet 500 mit Fehlermeldung → Admin sieht das Problem sofort im Dialog
  („Mail konnte nicht versendet werden"), kein stilles Scheitern im Log
- Approval-Mail bleibt vorerst best-effort (PDF-generation macht Sync teurer)
- Submission-Mails bleiben unverändert (public-facing, würde Antrags-Submit
  blocken)

### Neu — PROJ-41 + PROJ-43: Status-Change-Mails an Mitglied *(2026-05-17)*

Bisher erfuhr der Beitrittswerber nichts, wenn der EEG-Admin den Antrag
ablehnte oder Rückfragen stellte — der Antrag stand einfach still. Jetzt
löst jeder Wechsel auf `rejected` (PROJ-41) bzw. `needs_info` (PROJ-43)
automatisch eine E-Mail an `application.email` aus.

- Zwei neue Mail-Templates (`application_rejected_member.html`,
  `application_needs_info_member.html`) übernehmen die vom Admin
  eingegebene Begründung/Rückfrage **1:1** in den Mail-Body
- Reply-To = EEG-Kontakt-E-Mail, damit Antworten direkt an die EEG gehen
- Admin-Dialog zeigt einen blauen Hinweis-Block: „Der hier eingegebene
  Text wird per E-Mail an den Beitrittswerber übermittelt"
- Best-Effort + async: scheitert der Versand, wird der Statuswechsel nicht
  zurückgerollt — Fehler landet im Log + Prometheus-Metric
  `mail_sent_total{kind="member_rejection|member_needs_info"}`
- Out of scope: BulkChangeStatus löst (vorerst) keine Mails aus

### Neu — PROJ-39: Titel-Nach + Bankname im Public-Form + abweichende Adresse je Zählpunkt *(2026-05-17)*

Drei unabhängige Erweiterungen am öffentlichen Antragsformular.

- **„Titel nach"** als zusätzliches optionales Personenfeld (z.B. BSc, MSc, MBA). Bestehende `titel`-Spalte bleibt erhalten und repräsentiert implizit „Titel vor". Migration 000037 fügt `application.titel_nach` hinzu. Sichtbar in Mail, PDF und Excel-Export
- **„Bankname"** ist jetzt direkt vom Mitglied eingebbar (war bisher admin-only). Spalte `application.bank_name` existierte schon, nur neue Frontend- und API-Pfade
- **Abweichende Adresse je Zählpunkt** (Bricht V1-Architekturentscheidung!): Migration 000037 fügt 4 Adress-Spalten auf `metering_point` hinzu. UI zeigt eine Checkbox „Abweichende Adresse" pro Zählpunkt; bei Aktivierung werden Straße, Hausnummer, PLZ, Ort eingeblendet. Checkbox-State wird **nicht** persistiert — der Zustand ergibt sich beim Reload daraus, ob die vier Adressfelder gefüllt sind. Server enforciert die All-or-Nothing-Regel (entweder alle vier leer oder alle vier gesetzt)
- Mail (Member + EEG), Approval-PDF, Excel-Export, Admin-Detail-View berücksichtigen alle drei neuen Felder
- CLAUDE.md + docs/architecture.md aktualisiert: alte „all metering points use the same address as the member"-Klausel entfernt

### Behoben — Reset-Import: Mitgliedsnummer wird gelöscht *(2026-05-17)*

Beim Zurücksetzen eines Imports (`imported → approved`) blieb bisher die
Mitgliedsnummer am Antrag stehen, obwohl die zugehörige Participant-Zeile
in eegFaktura nicht mehr existiert. Resultat: stale Anzeige im Admin-Detail
+ Konflikt-Vorschlag beim nächsten Import-Versuch (selbe Mitgliedsnummer
würde wieder vorgeschlagen).

- **Backend** (`ResetImportTx`): zusätzlich `member_number = NULL`
- **Audit-Trail** (`AdminApplicationService.ResetImport`): die vorherige
  Mitgliedsnummer wird wie schon zuvor die `target_participant_id` an die
  Begründung angehängt (`[system] previous member_number=<x>`), damit sie
  nach dem Reset im Statusverlauf nachvollziehbar bleibt
- **Doku**: `docs/api-spec.md` 6.5.3 ergänzt um die zusätzliche Spalte +
  erweiterten Log-Reason

### Behoben — PROJ-31 Constraint-Lücke + Helm-Fix *(2026-05-16)*

- **DB**: Migration 000036 ergänzt `email_confirmed` im `application_status_check`-CHECK-Constraint. Davor lief jeder `confirm-email`-POST in einen Postgres-23514-Fehler → HTTP 500 „An internal error occurred". Ursache: PROJ-31 hatte die Status-Konstante + Transition-Map gepflegt, die DB-Constraint aber nie angepasst (Tests liefen gegen Go-Fake-Store, nicht gegen echtes Postgres)
- **Helm**: Backend-Deployment bekommt `PUBLIC_BASE_URL` aus `frontend.nextauthUrl` (single source of truth für die öffentliche App-URL). Vorher war die Env-Var im Chart gar nicht definiert → der PROJ-31-Confirm-Link wurde nie generiert (silent fallback auf Legacy-Flow ohne Bestätigungs-Block in der Mail)
- **Doku**: `docs/architecture.md` dokumentiert das Status-Set als 3-place-Invariant (Code-Konstanten + adminTransitions-Map + DB-CHECK-Constraint)

### Neu — PROJ-38: Status-Modell-Hygiene & Audit-Fixes *(2026-05-16)*

Code-Audit nach der PROJ-31-Constraint-Regression. Drei Findings umgesetzt, zwei als False-Positive verworfen.

- **`UpdateStatusAdminTx`** mit guarded `WHERE status = $expected_from` — bei 0 betroffenen Rows kommt `ErrConflict` (HTTP 409). Damit ist der admin-seitige Status-Schreibpfad auf dem gleichen Schutz-Niveau wie alle anderen `Mark*Tx`-Methoden. Vergisst ein Caller die Transition-Map oder mutiert ein paralleler Prozess parallel den Status, schlägt die UPDATE jetzt sauber fehl statt still durchzulaufen
- **`isKnownStatus`** deckt jetzt alle 9 Status-Werte ab (`email_confirmed` fehlte). Defensiv — der `adminTransitions`-Layer hatte die Konsequenz bereits korrekt abgefangen
- **`ResetImport`** dokumentiert, warum der PROJ-31-Confirmation-Gate hier intentional fehlt (Antrag bereits einmal vetted via `approved → imported`)
- Out of scope für separate Specs: Submit-Mail-Retry, Auto-Reject-Doppel-Metrik bei parallelen Pods

### Neu — PROJ-37: Genossenschaftsanteile *(2026-05-15)*

EEG-Admins können pro EEG aktivieren, ob Mitglieder bei der Registrierung Genossenschaftsanteile zeichnen müssen. Die Pflichtanzahl und der Wert je Anteil sind per EEG konfigurierbar; das Formular zeigt eine Live-Berechnung des Gesamtbetrags, die Beitrittsbestätigung weist die Anteile als eigene Sektion aus.

- **DB**: Migration 000035 fügt `registration_entrypoint.cooperative_shares_enabled` + `cooperative_required_shares` + `cooperative_share_amount_cents` und `application.cooperative_shares_count` hinzu (Integer-Cents für Geld, keine Float-Drift)
- **Admin-Settings**: neuer Abschnitt „Genossenschaftsanteile" mit Toggle + zwei conditional sichtbaren Inputs (Pflichtanteile + €-Wert). Validierung: enabled=true ⇒ beide Werte Pflicht, positiv
- **Public-Form**: konditioneller Block „Genossenschaftsanteile" zwischen Zählpunkten und Bankverbindung mit Hinweistext „Pflichtanteil je Standort: N", Eingabe (min=N, prefilled=N), Live-Berechnung Wert × Anzahl = Gesamtbetrag
- **Submit-Validierung**: `count >= required_shares` (Pflicht wenn EEG enabled). Konfig-Änderungen wirken **prospektiv** — bestehende Anträge bleiben unverändert
- **Admin-Detail**: eigene Mini-Box „Genossenschaftsanteile: N × X € = N·X €" mit Orange-Hinweis falls Bestand unter aktuellem Pflichtmaß
- **Beitrittsbestätigungs-PDF**: neue Sektion „GENOSSENSCHAFTSANTEILE" mit Anzahl × Wert = Gesamtbetrag
- **Nicht in Excel-Export, nicht in Core-Payload** — rein im Onboarding (eegFaktura hat keine Spalte dafür)
- **Bekannte V1-Lücke**: Admin-Edit-Form kennt das Feld noch nicht. Korrektur über needs_info-Flow möglich; Direkt-Edit folgt in V1.1 falls häufig benötigt

### Neu — PROJ-36: Optionale Rechtsdokumente als Info-Dokumente *(2026-05-15)*

Beta-Feedback: optionale Checkboxen waren verwirrend (Mitglieder wussten nicht, ob ihr fehlendes Häkchen rechtlich relevant ist). Der Toggle pro Rechtsdokument ist jetzt binär — **Pflicht-Zustimmung** oder **Nur zur Information**.

- **DB**: Migration 000034 fügt `document_consent.consent_type` (`explicit` | `informational`) hinzu, Default `explicit` für Bestandsdaten
- **Public-Form**: Pflicht-Dokumente bleiben Checkboxen; Info-Dokumente landen in einem eigenen „Zur Information"-Block mit Top-Border-Separator unterhalb aller Pflicht-Häkchen
- **Backend** (`ApplicationService.SubmitApplication`): schreibt automatisch `informational`-Consent-Einträge für jedes nicht-required `legal_document` der EEG — Audit-Trail bleibt vollständig auch ohne Häkchen
- **Admin-Detail + Beitrittsbestätigungs-PDF**: zwei separate Blöcke „Zugestimmte Dokumente · Zugestimmt am …" / „Zur Kenntnis genommene Dokumente · Kenntnis genommen am …"
- **Admin-Settings**: Toggle-Label kontextsensitiv („Mitglied muss zustimmen" / „Nur zur Information"), erklärender Hilfetext darunter, Listen-Badge sagt „Pflicht-Zustimmung" bzw. „Nur zur Information"

### Neu — PROJ-35: Per-EEG-Referenznummern *(2026-05-14)*

Antrags-Referenznummer im Format **`<RC>-<Jahr>-<NNNN>`** (z.B. `RC105720-2026-0001`) statt der bisherigen globalen `MO-YYYY-NNNNNN`-Sequenz. Counter resettet pro EEG und pro Jahr.

- Migration 000033 mit Counter-Tabelle `reference_number_counter (rc_number, year, last_value)`; atomare Increment via `INSERT … ON CONFLICT DO UPDATE … RETURNING`
- Bestehende Anträge behalten ihre alten Refs (Links in bereits verschickten Mails bleiben gültig)
- 4-stelliger Counter reicht für 9 999 Anträge/EEG/Jahr — Overflow gibt sprechenden Fehler statt Format zu erweitern

### Neu — PROJ-34: Robuste Import-Recovery *(2026-05-14)*

Behebt den „stuck-in-flight"-Fehlerklasse, die heute im Test-Cluster sichtbar wurde (Antrag bleibt nach DB-UNIQUE-Verletzung dauerhaft in `approved + in-flight`-Zustand).

- **Orphan-Fallback**: Wenn das Bookkeeping nach erfolgreichem Core-Insert fehlschlägt (UNIQUE-Index aus Migration 28 etc.), wechselt der Antrag in einer zweiten Transaktion auf `import_failed` mit `target_participant_id` und sprechender Fehlermeldung. Der bestehende Reset-Import-Flow (PROJ-30) wird damit zur Recovery-Route.
- **Lokaler Pre-Check** vor dem Core-Aufruf: `MemberNumberUsedLocally` blockiert duplikate `member_number` im selben EEG mit 409, bevor irgendetwas an den Core geht — kein Orphan-Teilnehmer mehr aus diesem Fehlerpfad
- **Stuck-Detection**: `AdminApplicationDetailResponse.importStuck` (server-side berechnet: `approved + import_started_at > 2 min + finished_at NULL`)
- **Zwei neue Admin-Endpoints**:
  - `POST /api/admin/applications/{id}/mark-imported-manually` — Admin gibt Core-UUID + Mitgliedsnummer ein, sauberer Übergang nach `imported`
  - `POST /api/admin/applications/{id}/clear-import-lock` — Lock raus, Status bleibt `approved` (Duplikatsrisiko, sprechender Warntext)
- **Admin-UI**: oranger Banner über der Statusaktions-Card mit zwei Recovery-Buttons inkl. Bestätigungsdialogen

### Neu — PROJ-33: EEG-Logo aus Core *(2026-05-14, Phase 2 von PROJ-32)*

EEG-Logo aus eegfaktura-billing-Service ziehen und in die Beitrittsbestätigung + SEPA-Mandat einbetten.

- **Endpoints**: `GET /cash/api/billingConfigs/tenant/{rc}` → `headerImageFileDataId` als Indikator, dann `GET /cash/api/billingConfigs/{billingConfigId}/logoImage` → Bytes
- **DB**: Migration 000032 mit `eeg_logo_bytes BYTEA`, `eeg_logo_mime TEXT`, `eeg_logo_synced_at TIMESTAMPTZ`
- **Caps**: 256 KB Hard-Limit via `io.LimitReader`, MIME-Whitelist `image/png|jpeg|gif` (gofpdf-kompatibel)
- **Best-effort**: Logo-Fetch-Fehler bricht den Stammdaten-Sync nicht ab; `logoSyncWarning` landet in der Response (Frontend rendert orangen Hinweis unter der Logo-Vorschau)
- **PDFs**: `embedLogoTopRight` rendert 30 mm hoch top-right, max 50 mm breit; korrupt-Bild oder fpdf-Fehler werden geloggt und übersprungen, PDF rendert weiter ohne Logo
- **Admin-UI**: Logo-Vorschau als 9tes Synced-Field in der Stammdaten-Card; Object-URL über `fetchEEGLogoBlob` (Bearer-Header), Cache-Bust via `eegLogoSyncedAt`-Timestamp
- **Neuer Endpoint**: `GET /api/admin/settings/eeg/logo?rc_number=…` liefert die Bytes mit korrektem `Content-Type` + 5-Min-Private-Cache

### Neu — PROJ-32: EEG-Stammdaten-Sync aus Core *(2026-05-14)*

Acht EEG-Stammdaten-Felder (Gemeinschafts-ID, Name, vier Adressfelder, Creditor-ID, Kontakt-E-Mail) werden direkt aus eegFaktura gespiegelt und sind im Onboarding **schreibgeschützt**.

- **GraphQL-Endpoint**: `POST {base}/api/query` mit `query { eeg }` (scalar `Eeg` — kein Selection-Set, returnt vollständiges JSON)
- **DB**: Migration 000031 mit `last_synced_from_core_at`; bestehende Stammdaten-Spalten werden vom Sync überschrieben
- **Architektur**: Single source of truth = `registration_entrypoint`; Auth = User-Context-Bearer-Forwarding (kein Service-Account); Microcache 30s auf `CompareEEGSettingsWithCore`
- **URL-Modell**: `CORE_BASE_URL` ist jetzt nur der Hostname (z.B. `https://eegfaktura.at`); Pfad-Prefixe (`/api/...`, `/cash/api/...`) sind im coreclient hardcoded — der frühere `CORE_GRAPHQL_URL`-env-var ist weg
- **UI**: Drift-Banner (grün/orange/grau) mit per-Feld-Diff; „Aus eegFaktura aktualisieren"-Button verwendet das Admin-JWT
- **Performance-Fix nebenbei**: `ListParticipants`-Body-Cap von 1 MiB auf 4 MiB hochgezogen (verhindert silent Truncation bei großen EEGs)

### Neu — PROJ-31: E-Mail-Adresse-Bestätigung (Anti-Abuse) *(2026-05-14)*

Pro EEG aktivierbar: Mitglieder müssen den Link in der Bestätigungs-Mail klicken, bevor der Antrag in den Admin-Review-Zustand wechselt.

- **Status-Modell**: neuer `email_confirmed`-Zustand zwischen `submitted` und `under_review`
- **DB**: Migration 000030 mit `email_confirmation_token_hash` (SHA-256), `email_confirmation_token_expires_at`, `email_confirmed_at`, `email_confirmation_used_at`, `registration_entrypoint.require_email_confirmation`
- **Security**: Token im URL-Fragment (`#token`) statt im Pfad → bleibt aus Server-Logs raus; Referrer-Policy `no-referrer`; idempotente Re-Clicks („Bereits bestätigt"-Seite statt 400)
- **Resend-Endpoint** für die Admin-Detail-Page; **30-Tage-Auto-Reject** via Background-Job
- **Admin-Guards**: `/status`-Endpoint refuses `submitted → under_review|needs_info|approved` mit 409 solange die Bestätigung aussteht — `submitted → rejected` bleibt als Anti-Spam-Override verfügbar

### Geändert — sonstige UX/Stabilität *(2026-05-15)*

- **B2B-Toggle-Label**: „Firmenlastschrift (B2B) für Unternehmen und **Vereine** verwenden" (zuvor „Verbände" — die Antrags-Auswahl kennt nur `Verein`)
- **Admin-Conflict-Messages**: Server-spezifische 409-Meldungen werden statt eines generischen „Aktion nicht mehr gültig"-Texts angezeigt (z.B. „E-Mail-Adresse des Bewerbers ist noch nicht bestätigt …")
- **Core-HTTP-400-Hint**: Opake `core returned HTTP 400: {}` wird auf eine handlungsorientierte Meldung übersetzt („Wahrscheinlichste Ursache: einer der Zählpunkte ist im Core bereits einem aktiven Teilnehmer zugeordnet")
- **Health-Probe-Spam**: K8s-Liveness/Readiness-Pings (`/livez`, `/readyz`) werden nicht mehr im Request-Log aufgezeichnet (Metric-Histogramm bekommt sie weiterhin)
- **CI**: `update-helm`-Job führt Retry-with-Rebase aus, behebt Race wenn manuelle Pushes mit dem Auto-Tag-Bump kollidieren

### Neu — PROJ-27: Tarif-Auswahl beim Import

Beim Klick auf „Importieren" öffnet sich ein Dialog, in dem Admin Tarif für Mitglied und je Zählpunkt wählt. Tarife werden zum Klick-Zeitpunkt live aus dem Core gelesen (`GET /eeg/tariff`), keine Persistierung im Onboarding.

- **Backend**: `coreclient.ListTariffs` + neuer Admin-Endpoint `GET /api/admin/tariffs?rcNumber=…`
- **Import-Flow**: Mitgliedstarif via `PUT /participant/v2/{id}` nach `POST /participant` (Core `EegParticipantBase.TariffId` ist `goqu:skipinsert`), Meter-Tarife direkt im `POST`-Body
- **Frontend**: `import-tariff-dialog.tsx` ersetzt den `confirm()`-Dialog
- Failure-Mode: schlägt das nachgelagerte Mitglieds-Tarif-Update fehl, wird Warnung in der Response zurückgegeben (Import gilt aber als erfolgreich)

### Neu — PROJ-28: Trennung Privat / Kleinunternehmer

Eigener `member_type` `sole_proprietor` (Kleinunternehmer). Privatperson zeigt Vor-/Nachname, Kleinunternehmer nur Firmenname (wird intern als `firstname` im Core eingestellt, weil dort NOT NULL).

- **Backend**: neue Konstante `MemberTypeSoleProprietor`, oneof-Validatoren erweitert (4 Stellen), Member-Type-Validation passt UID/Register-Felder an
- **Frontend**: zusätzlicher SelectItem; Admin-Edit-Form blendet UID/Register je nach Typ ein/aus
- **Salutation-Fix als Side-Effect**: leere `Sehr geehrte/r ,` für alle Org-Typen behoben (`application_submitted_member.html` mit `{{if .Firstname}}…{{else}}Sehr geehrte Damen und Herren{{end}}`)

### Neu — PROJ-29: IBAN-Eingabe mit visueller Gruppierung

IBAN-Feld nutzt `MaskedInput` (react-imask) mit Block-Gruppierung pro 4 Stellen.

- **Initiale Umsetzung**: feste Mask `aa00 0000 …` (AT/DE/ES/BE/LU/SI — alle Länder mit reinen Ziffern im BBAN)
- **Erweiterung (2026-05-13)**: **dynamische landesabhängige Mask** — `src/lib/iban-mask.ts` generiert pro Land aus `ibantools.countrySpecs.bban_regexp` die exakte Mask-Struktur (Ziffern vs. Buchstaben vs. alphanumerisch). ~80 IBAN-Länder werden ohne Mapping-Wartung unterstützt
- **Validierungs-Bugfix**: `zod`-Transform strippt jetzt `[^A-Z0-9]` (vorher nur `\s`), entfernt iMask-Platzhalter `_` aus dem submitted value bevor `isValidIBAN` prüft

### Neu — PROJ-30: Reset eines importierten Antrags auf `approved`

Wenn ein importiertes Mitglied im Core wieder gelöscht wird, kann der Admin den Antrag jetzt auf `approved` zurücksetzen, um ihn erneut zu importieren.

- **Endpoint**: `POST /api/admin/applications/{id}/reset-import` (Body: `{ "reason": "…" }`)
- **Repo**: `ResetImportTx` setzt `import_*`-Felder + `target_participant_id` zurück auf NULL; alte Participant-ID wird im `status_log.reason` archiviert
- **Status-Modell**: Die Transition `imported → approved` ist bewusst **nicht** im generischen `adminTransitions`-Map — sie geht ausschließlich über den dedizierten Endpoint (Security-relevant, siehe CLAUDE.md)
- **Frontend**: „Import zurücksetzen"-Button mit Bestätigungsdialog inkl. Hinweis auf vorherige Core-Löschung

### Neu — Approval-PDF: Einheitliche Zustimmungs-Timestamps

PDF-Bereich „ERTEILTE ZUSTIMMUNGEN" zeigt jetzt überall Datum **und** Uhrzeit:

- Datenschutz: `privacy_accepted_at`
- Richtigkeit der Angaben: `submitted_at` (Validierung erfolgt im Submit-Moment, keine eigene Spalte nötig — keine Migration)
- SEPA-Mandat: `sepa_mandate_accepted_at` (Format `am DD.MM.YYYY HH:MM`)
- Dokument-Zustimmungen: erweitert um Uhrzeit

### Geändert

- **PDF + Mail: SEPA-Mandat-Beschriftung korrigiert (zuvor invertiert).** Bei `SEPAMandateEnabled=true` (Admin-Setting „SEPA-Lastschriftmandat dem Willkommensmail anhängen") zeigt PDF und Member-Mail jetzt **„Per E-Mail übermittelt"**, bei `false` **„Erteilt"**. Vorher andersrum gelabelt.
- **Zählpunkt-Feld**: schmalere Darstellung am Desktop (Default-Sans + `tabular-nums` + `tracking-tighter` + `px-2`), damit die 37-stellige Mask in einer Zeile passt. Mobile-Optik bleibt identisch.
- **Zählpunkt-Label**: Info-Popover beim Label erklärt was die Zählpunktnummer ist und wo sie zu finden ist (Stromrechnung / Kundenportal).

### Behoben — Zeitzone: alle sichtbaren Timestamps jetzt Europe/Vienna

PostgreSQL speichert UTC; vorher rendete PDF / Mail / Admin-Web jeweils unterschiedlich (UTC vs. Browser-Zone). Vereinheitlicht auf Europe/Vienna mit CET/CEST-Umstellung:

- **Backend**: neuer Helper `internal/shared/timezone.go` (`DisplayLocation`, `FmtDateTime`, `FmtDate`). PDF und Mail-Service nutzen ihn; Mail-Templates über `template.Funcs` (`{{fmtDateTime …}}`)
- **Frontend**: neuer Helper `src/lib/datetime.ts` (`formatDateTime`, `formatDate`, `formatPlainDate` — alle mit `timeZone: "Europe/Vienna"`). Ersetzt 4 inline-Implementierungen in `admin-application-detail`, `admin-application-table`, `admin-api-key-editor`, `admin-status-log`
- **DATE-Felder** (`birth_date`, `membership_start_date`) bleiben TZ-unaware, da ohne Zeitkomponente

### Strenge Zählpunktnummer-Validierung

Frontend (Zod) und Backend (Regex + struct tag `len=33,startswith=AT`) lehnen Eingaben außerhalb von `^AT[0-9]{31}$` ab. Eingabe wird automatisch ge-uppercased und whitespace-bereinigt.

### Sonstiges

- Favicon hinzugefügt (`src/app/icon.svg`, Next.js App-Router Auto-Detect)
- Mobile-Optik: Zählpunkt-Input nutzt `text-xs font-mono tracking-tight` auf engen Viewports

### Neu — Click-to-Sort in der Admin-Liste

Spalten-Header der Antrags-Tabelle sind klickbar und sortieren server-seitig:

- Frontend: Pfeil-Icon (↕ inaktiv · ↑ ASC · ↓ DESC) je Spalte; Default `submittedAt DESC`. Status in URL-Params `?sort=…&order=…` persistiert, Filter-Reset bewahrt die Sortierung.
- Backend: `sort` + `order` Query-Parameter auf `GET /api/admin/applications`. Strict-Whitelist im Repo (`allowedSortColumns`) — kein SQL-Injection-Risiko. Name-Sort nutzt `COALESCE(NULLIF(firstname+lastname), company_name)`, damit Privat- und Firmen-Einträge in einer alphabetischen Reihenfolge erscheinen.

### Behoben — Architektur-Review-Sweep (Chart 1.6.16 → 1.7.7)

Bündel kleinerer und kritischer Verbesserungen, motiviert durch einen umfassenden Architektur-Review vor dem Ramp-up auf mehr User:

#### Datenintegrität / silent-data-loss

- **AdminNoteEditor schickte einen vollen `PUT /applications/{id}` mit nur dem Notiz-Feld** — Backend macht für `meteringPoints` einen REPLACE, sodass jedes Notiz-Speichern auf einem Firmen-/Vereins-Antrag die `participationFactor`-Werte aller Zählpunkte auf `0` zurücksetzte. Neuer dedizierter Endpoint `PATCH /api/admin/applications/{id}/admin-note` schreibt nur die `admin_note`-Spalte; Frontend nutzt `setAdminNote()` aus dem API-Client.
- **Duplicate-Draft-Falle**: `createApplication` → `submitApplication`-Flow ohne ID-Cache produzierte bei Submit-Fehler + Retry einen zweiten Draft. App-ID + Form-Values-Snapshot werden jetzt in `useRef` gespeichert; Retry ohne Edits überspringt `create`. 404-Response invalidiert den Cache.
- **Superuser-Bulk-Delete löschte 0 Anträge**: Der Handler ließ `rcNumbers` für Superuser leer, das Repo machte daraus einen Early-Return mit 0 Löschungen. Eigene `DeleteAllDrafts()` ohne Scope für Superuser, alte `DeleteDraftsByRCNumbers()` weiterhin für Tenant-Admins. Log-Line zeigt `superuser=true/false`.
- **Frontend `adminRequest` überschrieb Authorization**: Bei Aufrufen, die eigene `headers: {...}` mitgaben, wurde der Bearer-Token verschluckt → 401 `duration_ms=0`. Headers werden jetzt explizit gemerged statt gespreaded.

#### Security-Härtung

- **Body-Size-Limits per Route-Gruppe** via neuer `MaxBodySize`-Middleware: 256 KiB für `/api/public` und `/api/external`, 1 MiB für `/api/admin`. Schließt unbounded-Body-DoS-Surface.
- **Trusted-Proxy-CIDR für `realIP()`**: Header `X-Real-IP` / `X-Forwarded-For` werden nur akzeptiert, wenn `r.RemoteAddr` aus den konfigurierten CIDRs kommt (env `TRUSTED_PROXY_CIDRS`, default in Helm: typische K8s-Pod/Service-CIDRs). Verhindert Spoofing des per-IP-Rate-Limits.
- **NetworkPolicies** (opt-in via `networkPolicies.enabled`, default true): `backend ← frontend + ingress`, `frontend ← ingress`, `postgres ← backend + migrate + seed` (NICHT Frontend). Defense-in-Depth gegen kompromittierte NPM-Transitives im Frontend-Pod.
- **Status-Transition `imported → approved`** bereits in PROJ-30 ausschließlich über dedizierten Endpoint (`POST /reset-import`) erreichbar, nie über die generische `/status`-Route.

#### Resilience

- **Health-Probes gesplittet**: Backend bekommt `/livez` (always 200, kein DB-Touch) und `/readyz` (DB-Ping). Frontend bekommt `/api/health` (always 200, kein Backend-Call). Helm-Probes umgestellt — DB-Blip kann nicht mehr per `livenessProbe` einen Restart-Loop auslösen, Backend-Outage kaskadiert nicht in Frontend-NotReady.
- **AbortController** in Admin-Web-Fetches (Liste, Detail, Tariff-Dialog): `useEffect`-Cleanup mit `AbortController`, `signal` durch `adminRequest`. Race-Condition bei schneller Navigation / Tariff-Dialog-EEG-Wechsel beseitigt.
- **Zentrales 401-Handling**: `adminRequest` emittiert `auth:expired`-Event auf 401; `SessionRefreshGuard` triggert `signIn("keycloak")`. User landen auf Keycloak-Login statt rote Error-Banner bei abgelaufenen JWTs.
- **`tzdata` in Go-Binary**: `_ "time/tzdata"` Blank-Import in `internal/shared/timezone.go`. `time.LoadLocation("Europe/Vienna")` funktionierte im Alpine-Container nicht, weil Alpine standardmäßig kein `tzdata`-Paket hat → Helper fiel still auf UTC zurück trotz aller PDF/Mail/Frontend-TZ-Migration. ~450 KB Binary-Overhead.

#### Operations

- **Velero-Pre-Backup-Hook am Postgres-StatefulSet** (`pre.hook.backup.velero.io/command: psql -c CHECKPOINT;`) — Cluster-Velero macht jetzt konsistente CSI-Snapshots statt Crash-Recovery-Restore.
- **`docs/operations.md`** als App-spezifisches Runbook: Backup-Scope + RPO/RTO, Restore-Verfahren (Namespace-only, PVC-only, Full-Cluster), 7-Punkte Post-Restore-Checklist, 4 Incident-Szenarien (Core-Outage, SMTP-Down, Lastspitze, Velero-Alert), Deployment + Rollback, bekannte Einschränkungen.
- **Slim `checkTenantAccess`**: Neue `GetRCNumberByID`-Query statt voller `GetApplicationDetail` (sparte ~4 Round-Trips pro Admin-Click).

#### Mail / Spam-Deliverability

Bestätigte Analyse einer realen Production-Mail: DKIM=pass (`postal-TA3f2w._domainkey.eegfaktura.at`), SPF=pass (via `psrp.eegfaktura.at`-Subdomain-Delegation), DMARC=pass. **Authentication ist bereits korrekt** — keine DNS-Änderungen erforderlich. Content-/Header-seitige Optimierungen:

- **From-Header mit Display-Name**: `"eegFaktura Mitglieder-Onboarding" <noreply@eegfaktura.at>` via neuer Env `SMTP_FROM_NAME` und `msg.FromFormat()`. Legitimitäts-Signal für Inbox-Provider.
- **Reply-To pro Mail-Typ**: Member-Bestätigung → EEG-Contact-Email; EEG-Notification + Approval → Antragsteller-Email. Replies auf `noreply@` haben damit ein sinnvolles Ziel.
- **`Auto-Submitted: auto-generated`** (RFC 3834) auf allen Mails. Transaktional-Indikator für Gmail; bricht Out-of-Office-Loops.
- **`User-Agent` + `X-Mailer`** via `SetUserAgent()` beide auf `"eegFaktura Member Onboarding"` (statt gomail-Default `go-mail v0.7.2 // github…`, der manche Filter triggert).
- **`Message-ID`**: `<random-hex>@eegfaktura.at` statt `<…@member-onboarding-test-backend-9df68fbc9-wlsq4>` (Pod-Hostname).
- **Plain-Text-Alternative verbessert**: `htmlToText` rendert Tabellen als `Label: Wert`, Links als `text (url)`, strippt `<head>`/`<style>`/`<script>` vor Tag-Entfernung. Schließt die HTML-vs-Plain-Divergenz, die klassische Spam-Filter flaggen.
- **Identification-Footer** in allen 3 Templates: Grund der Mail, Sender-Identifikation, Hinweis dass Reply-Path funktioniert.

#### Tests + Doku

- `internal/mail/mailer_test.go` neu: 4 Tests gegen Multipart-Struktur, Headers, User-Agent-Branding, Message-ID-Domain.
- `docs/architecture.md` ergänzt um Time/Timezone-Konvention und (siehe oben) Resilience-Bausteine.

### Neu — Mitgliedsnummer wird beim Import vergeben (statt beim Submit)

Die Mitgliedsnummer ist im Core-System die Quelle der Wahrheit. Das Onboarding kennt erst zum Import-Zeitpunkt den aktuellen höchsten Wert. Die Pflege im Onboarding (`registration_entrypoint.member_number_start` + Auto-Assign in `AssignMemberNumberTx`) wird durch eine Live-Abfrage am Core ersetzt.

- **Neuer Endpoint** `GET /api/admin/applications/{id}/next-member-number` — ruft Core `GET /participant`, ermittelt nächste freie Nummer
- **Pattern-aware Vorschlag**: Algorithmus erkennt dominantes Muster (Präfix + Padding). `A001, A002, A005` → Vorschlag `A006`. `M-12, M-13` → `M-14`. Reine Ziffern: `1, 2, 3` → `4`. Padding wächst (`01, 99` → `100`). Bei gemischten Mustern gewinnt die Gruppe mit den meisten Einträgen.
- **String-typed**: Migration 000027 promoted `application.member_number` von `INT` auf `TEXT`, weil Core `participantNumber` `VARCHAR` ist. Models, Repo, Payload, PDF, Excel, Frontend-Types durchgängig string.
- **Pre-Import-Duplikat-Check** im Backend: vor `POST /participant` wird die gewählte Nummer gegen die Core-Teilnehmerliste verglichen; bei Doppelvergabe 409.
- **Tariff-Dialog erweitert** um „Mitgliedsnummer"-Input (Pflichtfeld, max 50 Zeichen, mit Vorschlag-Prefill).
- **AdminEditForm**: Mitgliedsnummer-Feld entfernt.
- **AdminEEGSettingsEditor**: „Mitgliedsnummer Startwert"-Feld entfernt; Spalte `registration_entrypoint.member_number_start` bleibt im Schema (unbenutzt).
- **`AssignMemberNumberTx`** Call beim Submit ist raus. `application.member_number` ist von Submit bis Import `NULL`; das Approval-PDF rendert die Spalte erst nach erfolgreichem Import.

### Neu — Click-to-Sort, Auth-Loop-Cooldown, Import-Robustheit

#### Click-to-Sort auf der Admin-Liste
- Server-seitige Sortierung mit strict Allowlist (`allowedSortColumns`); URL-persistierte `sort`/`order`-Parameter; Pfeil-Icons (↕/↑/↓) im Header
- „Name"-Sortierung nutzt `COALESCE(NULLIF(TRIM(CONCAT_WS(' ', firstname, lastname)), ''), company_name)` — Privatpersonen und Firmen mischen alphabetisch korrekt

#### Auth-Loop nach Deploy
- 401 → `signIn("keycloak")` → Keycloak-Roundtrip → 401 (neuer Pod noch nicht ready) → Loop. Behoben mit sessionStorage-basiertem 30s-Cooldown der die Page-Navigation überlebt. Zweite 401 innerhalb des Cooldowns triggert keinen erneuten Redirect; Banner „Anmeldung erforderlich, aber automatische Weiterleitung wurde unterdrückt".

#### Import-Robustheit-Bündel
- **Import-Context detachen**: nach `MarkImportInFlight` läuft der Core-Call auf `context.WithTimeout(context.Background(), 2*time.Minute)`. Browser-Close oder Network-Drop unterbricht den Core-Call nicht mehr → keine Orphan-Participants im Core + Duplikat bei Retry.
- **ResetImportTx mit `SELECT ... FOR UPDATE`**: explizite Row-Lock + Pre-Check `(import_started_at NOT NULL AND import_finished_at IS NULL)`. Reset während laufenden Imports = 409 statt Race.
- **Migration 000028**: partial UNIQUE Index `(rc_number, member_number) WHERE NOT NULL` als Defense-in-Depth gegen Doppelvergabe.

### Neu — Observability: Prometheus /metrics

Counter (Namespace `eegfaktura_mo`): `applications_submitted_total`, `imports_total{result}`, `mail_sent_total{kind,result}`, `rate_limit_hits_total`, `member_number_lookups_total{result}`, `http_request_duration_seconds{method,status_class}`. Bundled `go_*` + `process_*`.

- **Separater HTTP-Server auf :9090** (env `METRICS_PORT`, default `9090`), bewusst NICHT durch den Public-Ingress geroutet
- **Helm**: dedizierter ClusterIP-Service (`backend-metrics`) mit `prometheus.io/scrape`-Annotationen; optional `ServiceMonitor` (`metrics.serviceMonitor.enabled`) für prometheus-operator-Stacks; NetworkPolicy erlaubt Ingress aus `networkPolicies.prometheusNamespace` (Default `cattle-monitoring-system` für Rancher)
- **Counter-Overhead vernachlässigbar** (Nanosekunden pro `Inc()`); deaktivierbar via `metrics.enabled: false`

### Performance — Quickwins-Bündel

- **Migration 000029**: composite indexes `(application_id, created_at)` auf `status_log`, `document_consent`, `metering_point`. Admin-Detail-View liest jetzt ohne heap-fetch + sort.
- **Deep-Pagination-Cap**: `page > 10_000` wird gedeckelt — kein OFFSET-Scan über Millionen Zeilen durch Buggy-Clients.
- **„Alle Entwürfe löschen"-Dialog respektiert `rc_number`-Filter**: Count + Delete-Call führen den aktiven Filter mit. Multi-EEG-Admin kann nicht mehr versehentlich über alle EEGs hinweg löschen.

### External-API Scope-Review (Befund)

Audit: `/api/external/*` exponiert ausschließlich `POST /v1/applications` mit API-Key-Auth. Keine Liste/Detail-Endpoints, keine RC-Number-Enumeration, keine Admin-Operations. **Scope ist bereits minimal**, keine Cleanup-Arbeit notwendig.

---

## [v1.10.0] - 2026-05-09

### Neu — PROJ-4: Core Import

Synchroner Import genehmigter Anträge in das eegFaktura-Core-System.

- **Backend**: `POST /api/admin/applications/{id}/import` ruft den Core-Endpoint `POST /participant` auf. Bearer-Token des angemeldeten Admins wird durchgereicht, `tenant`-HTTP-Header wird auf die RC-Nummer der Application gesetzt.
- **Architektur**: neue Pakete `internal/coreclient` (HTTP-Wrapper) und `internal/importing` (Orchestrierung + Payload-Mapping)
- **Concurrency-Sperre**: `MarkImportInFlight` verhindert Duplikate im (nicht-idempotenten) Core durch parallele Klicks
- **Defense-in-Depth**: Service-Level-Tenant-Check zusätzlich zum Handler-Check
- **Frontend**: Status-Aktionen-Box zeigt „In eegFaktura importieren" für `approved`-Anträge, „Import erneut versuchen" + Error-Banner für `import_failed`, sowie die Participant-ID nach erfolgreichem Import
- **Konfig**: `CORE_BASE_URL` (mit `/api`-Suffix) und `CORE_TIMEOUT_SECONDS` als neue Env-Vars; via Helm-Values `backend.coreBaseUrl` durchgereicht

### Erkenntnisse aus dem Live-Rollout

- **Keycloak Tenant-Mapper**: muss `Claim JSON Type: JSON` haben (nicht `String`), sonst lehnt der Core mit 401 leerem Body ab
- **businessRole** muss gesetzt werden (`EEG_PRIVATE` / `EEG_BUSINESS`), sonst Privat-Tab im UI auch für Firmen
- **firstname** der Core-Tabelle ist NOT NULL — für Firmen/Vereine/Gemeinden wird der Organisationsname dort eingestellt
- **Meter-Direction**: Onboarding `PRODUCTION` → Core `GENERATION`

Details siehe `features/PROJ-4-core-import.md` und `docs/import-mapping.md` §7–§9.

### Geändert

- `coreclient`: UTF-8-sichere Truncation, erkennt zusätzlich `context.Canceled` und `net.Error.Timeout()`, klare Sentinel-Errors
- `ImportService`: Bookkeeping-Failure nach Core-Erfolg loggt Participant-ID + surface in Result (Operator kann manuell aufräumen)
- Handler nutzt `errors.Is`/`errors.As` für robuste Error-Routing über Wrapping hinweg

### Infrastruktur

- Helm-Chart erweitert um `backend.coreBaseUrl` und `backend.coreTimeoutSeconds`
- `values-env.yaml.example` dokumentiert beide Werte mit Beispiel inkl. `/api`-Suffix

---

## [v1.9.0] - 2026-04-30

### Neu
- **Admin-GUI**: Button „Beitrittsbestätigung herunterladen" in der Antragsdetailansicht (`GET /api/admin/applications/{id}/approval-pdf`) für Status `approved`, `imported`, `import_failed`
- **Mitglieds-Bestätigungs-E-Mail**: Enthält jetzt alle eingegebenen Antragsdaten (Persönliche Daten, Adresse, Bankverbindung, Zählpunkte) und alle erteilten Zustimmungen

### Geändert
- **Beitrittsbestätigung PDF**: Mitgliedsnummer wird als erster Eintrag in MITGLIEDSDATEN angezeigt (kein leeres Leerfeld mehr)
- **Beitrittsbestätigung PDF**: Zustimmungen vollständig — Datenschutz (mit Version), Richtigkeit, SEPA (Checkbox oder „Per E-Mail übermittelt"), Dokumentzustimmungen mit Datum
- **Beitrittsbestätigung PDF**: Statusverlauf-Labels auf Deutsch (z. B. „Eingereicht" statt „submitted")
- **SEPA-Mandat**: Kontoinhaber-Feld wird ausschließlich aus `AccountHolder` befüllt — kein automatischer Fallback auf Vorname/Nachname mehr

### Infrastruktur
- Vitest-Konfiguration auf `.mts` umgestellt (behebt `ERR_REQUIRE_ESM`-Fehler bei `npm test`)
- Dokumentation aktualisiert: `docs/domain-model.md`, `docs/api-spec.md`, Feature-Specs PROJ-21 und PROJ-6

---

## [v1.8.0] - 2026-04-29

### Neu — PROJ-25: Bulk-Aktionen im Admin
- Mehrere Anträge gleichzeitig genehmigen, ablehnen oder zur Prüfung setzen
- Checkboxen pro Zeile + „Alle auswählen"-Checkbox mit indeterminate-State
- Aktionsleiste erscheint bei aktiver Auswahl mit Bestätigungsdialog
- Ergebnis-Zusammenfassung nach Ausführung (X erfolgreich, Y übersprungen)
- Backend: `POST /api/admin/applications/bulk-action` mit Tenant-Isolation; max. 200 Anträge pro Request; ungültige Transitionen werden übersprungen (kein Fehler)

### Neu — PROJ-24: OpenAPI/Swagger Dokumentation
- Interaktive Swagger UI unter `/swagger/` verfügbar
- Alle Admin- und Public-Endpunkte vollständig annotiert (Swaggo)
- Automatische Swagger-Generierung via `swag init` in CI

---

## [v1.7.0] - 2026-04-26

### Neu — PROJ-20: Vollständige Antragsdaten in EEG-Einreichungsbenachrichtigung
- EEG-Betreiber erhält bei jeder Neueinreichung alle Antragsdaten per E-Mail
- Felder: Mitgliedstyp, Name/Firma, Adresse, Kontakt, IBAN, SEPA-Ermächtigung, Zählpunkte, konfigurierbare Felder
- Konfigurierbare Felder werden nur angezeigt wenn nicht `hidden` und befüllt
- Optionaler Admin-Link zur Detailansicht (via `ADMIN_BASE_URL`-Umgebungsvariable)

### Neu — PROJ-21: Genehmigungs-Benachrichtigung mit Beitrittsbestätigung PDF
- Bei Status-Übergang → `approved` erhält die EEG automatisch eine E-Mail mit PDF-Anhang
- PDF „Beitrittsbestätigung" enthält: Mitgliedsdaten, Bankverbindung, Zählpunkte, Zustimmungen, Statusverlauf, konfigurierbare Felder
- PDF-Generierung schlägt fehl → E-Mail wird trotzdem gesendet (mit Hinweistext); Status-Übergang bleibt gültig
- Re-Approval (`import_failed → approved`) sendet erneut eine E-Mail

---

## [v1.6.0] - 2026-04-25

### Neu — PROJ-9: EEG-spezifische Rechtsdokumente
- Admin kann beliebige Rechtsdokumente pro EEG konfigurieren (Satzung, AGB usw.)
- Mitglied muss Pflichtdokumente vor Einreichung bestätigen
- Zustimmungen werden als unveränderliche Snapshots gespeichert (`document_consent`)
- Max. 10 Dokumente pro EEG; sortierbar per Drag-and-Drop

### Neu — PROJ-16: Cloudflare Turnstile Spam-Schutz
- Öffentliches Registrierungsformular mit Turnstile-CAPTCHA geschützt
- Aktivierung via `TURNSTILE_SECRET_KEY`-Umgebungsvariable (fehlt → deaktiviert)

### Neu — PROJ-17: Excel-Export für eegFaktura-Import
- Admin kann Antrag als `.xlsx`-Datei exportieren (`GET /api/admin/applications/{id}/export/excel`)
- Datei im eegFaktura-Importformat (36 Spalten, eine Zeile pro Zählpunkt)
- Nur für Status `approved`, `imported`, `import_failed`

### Neu — PROJ-18: Datenschutzerklärung & Central Policy Toggle
- Zentrale Datenschutzerklärung (Betreiber-Policy) über Umgebungsvariablen konfigurierbar (`CENTRAL_POLICY_TITLE`, `CENTRAL_POLICY_URL`)
- Pro EEG einstellbar, ob die zentrale Policy im Formular angezeigt wird (`showCentralPolicy`)
- EEGs mit eigener Datenschutzerklärung können die zentrale Policy ausblenden

### Neu — PROJ-19: Manuelle Aktivierung der Registrierung
- Neue EEGs sind standardmäßig inaktiv (`is_active = false`)
- Admin kann Registrierung pro EEG aktivieren/deaktivieren (Settings-Seite)
- Inaktive EEGs: öffentliches Formular liefert `410 Gone`

---

## [v1.5.0] - 2026-04-24

### Neu — PROJ-12: SEPA-Lastschriftmandat PDF
- Automatische Generierung eines SEPA-Lastschriftmandats als PDF-Anhang in der Mitglieds-Bestätigungs-E-Mail
- Aktivierung pro EEG via `sepaMandateEnabled`-Einstellung
- Unterstützt CORE- und B2B-Mandat
- Kann auch per E-Mail zugesandt werden (`sepa_mandate_enabled = false`): Hinweis im PDF und in der Bestätigungs-E-Mail

### Neu — PROJ-13: Externe Registrierungs-API
- `POST /api/external/v1/applications` — Anträge direkt aus externen Systemen einreichen
- API-Key-Authentifizierung (kein Keycloak); Key pro EEG generierbar/widerrufbar in den Admin-Settings
- Rate Limiting: 10 Requests / 60 Sekunden (Burst) + 200 Einreichungen / Tag (Quota)

### Neu — PROJ-14: SEPA-Firmenlastschriftmandat
- Für Mitglieder vom Typ `company` / `association` kann ein SEPA-B2B-Mandat statt des Standard-CORE-Mandats generiert werden
- Steuerbar über EEG-Einstellung `useCompanySEPAMandate`

### Neu — PROJ-15: Konfigurierbare Felder Erweiterungen
- Neuer Feld-Status `admin_only`: Feld ist im öffentlichen Formular verborgen, wird aber mit einem konfigurierten Admin-Standardwert automatisch befüllt
- Zählpunktfelder konfigurierbar: `transformer`, `installation_number`, `installation_name`

---

## [v1.4.0] - 2026-04-23

### Neu — PROJ-11: Konfigurierbarer Einleitungstext
- Admin kann pro EEG einen Einleitungstext für das Registrierungsformular hinterlegen (HTML, sanitisiert)
- Wird im öffentlichen Formular über dem Antragsformular angezeigt

---

## [v1.3.0] - 2026-04-22

### Neu — PROJ-8: Konfigurierbare Felder pro EEG
- Admin kann pro EEG konfigurieren, welche optionalen Felder im Registrierungsformular sichtbar, versteckt oder Pflicht sind
- Konfigurierbare Felder: `phone`, `birth_date`, `uid_number`, `membership_start_date`, `persons_in_household`, `consumption_previous_year`, `consumption_forecast`, `feed_in_forecast`, `pv_power_kwp`, `heat_pump`, `electric_vehicle`, `electric_hot_water`

---

## [v1.2.0] - 2026-04-21

### Neu — PROJ-6: E-Mail-Benachrichtigungen
- Mitglieds-Bestätigungs-E-Mail nach erfolgreicher Einreichung
- EEG-Benachrichtigungs-E-Mail an `contact_email` der EEG
- Asynchroner Versand (kein Blockieren der Einreichung bei SMTP-Fehler)
- Resend-Funktion im Admin: „Bestätigung erneut senden"
- Konfiguration via `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`

---

## [v1.1.0] - 2026-04-20

### Neu — PROJ-5: Keycloak-gesicherte Admin-Oberfläche
- Admin-Bereich erfordert Keycloak-Login (JWT Bearer Token)
- Tenant-Isolation: Admins sehen nur Anträge ihrer eigenen EEGs
- Superuser-Flag für EEG-übergreifenden Zugriff

### Neu — PROJ-7: Mitgliedstypen
- Unterstützung für fünf Mitgliedstypen: Privatperson, Landwirt, Gemeinde, Unternehmen, Verein
- Typenspezifische Felder (Firmenname, UID-Nummer, Firmenbuchnummer)
- Kompakte Select-UI im Registrierungsformular

---

## [v1.0.0] - 2026-04-19

### Neu — PROJ-1: Öffentliche Registrierung
- Öffentliches Registrierungsformular unter `/register/{rc_number}`
- Antragstellung mit Personendaten, Adresse, IBAN, Zählpunkten
- Mehrschrittiges Formular mit Validierung (Frontend + Backend)
- Antragsstatus: `draft` → `submitted`

### Neu — PROJ-2: Admin-Review
- Admin kann Anträge einsehen, bearbeiten und Status ändern
- Status-Workflow: `submitted → under_review → approved / rejected / needs_info`
- Admin-Notiz und Rückfrage-Grund pro Antrag

### Neu — PROJ-3: Admin-Frontend-UI
- Antragsübersicht mit Filter und Pagination
- Detailansicht mit vollständigen Antragsdaten
- Status-Aktionen direkt aus der Detailansicht
