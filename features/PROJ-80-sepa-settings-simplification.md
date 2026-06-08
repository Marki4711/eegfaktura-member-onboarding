# PROJ-80: SEPA-Settings-Vereinfachung (sepaMandateEnabled entfernen)

## Status: Approved
**Created:** 2026-06-08
**Last Updated:** 2026-06-08 (Backend + Frontend implementiert, alle Tests grün, Doku komplett, QA APPROVED, Security-Review APPROVED)

## Dependencies

- Erfordert: PROJ-48 (SEPA-Mandat erst beim Import) — Deployed, Timing-Toggle bleibt, Label wird umbenannt
- Erfordert: PROJ-74 (B2B-Mandat-Gate-Fix) — Deployed, `sepaMandateEnabled` hat seit PROJ-74 keinen B2B-Bezug mehr
- Erfordert: PROJ-75 (SEPA-Einwilligungs-Checkbox in Bankverbindungs-Card) — Deployed, Online-Zustimmung-Checkbox wird hier von „conditional" auf „immer Pflicht" umgestellt
- Erfordert: PROJ-78 (Toggle Elektronisches SEPA-Mandat) — Deployed, CORE-Audit-Toggle bleibt unverändert
- Supersedes (teilweise): PROJ-73-Pattern (Legacy-Kompat für entfernte Toggles), siehe `LegacyUseCompanySEPAMandate` als Vorlage

## Hintergrund

Der Toggle „SEPA-Mandat als Datei dem Antragsteller übermitteln, statt
die Zustimmung per Checkbox einzuholen" (`sepaMandateEnabled` in DB +
API) hat seine semantische Klarheit verloren:

- Mit PROJ-74 wurde `sepaMandateEnabled` für B2B-Mandate entkoppelt
  (B2B-PDFs kommen unabhängig vom Toggle).
- Mit PROJ-77 + PROJ-78 (Audit-Trail) sind die Verhaltensmuster
  „Online-Zustimmung" und „PDF mit elektronischer Signatur-Dokumentation"
  konvergiert — beide signalisieren elektronische Zustimmung, der
  Unterschied liegt nur noch im PDF-Anhang.

## Owner-Direktive 2026-06-07/08

Im neuen Modus ist nicht mehr die zentrale Frage, ob ein PDF erzeugt
wird. **Es wird immer ein PDF erzeugt** — die zwei nachgelagerten
Toggles entscheiden über **Variante** (Audit-Block vs. Unterschriften-
Feld) und **Versand-Zeitpunkt** (Submit vs. Import).

### Drei neue Konstanten

- Online-Zustimmung-Checkbox im Mitgliederformular ist **immer Pflicht**
  (auch bei aktiviertem Audit-Toggle — der Audit-Block dokumentiert die
  Zustimmung, die Checkbox ist die Zustimmung selbst).
- SEPA-Mandat-PDF wird für `einzugsart=core` **immer erzeugt** (analog
  zu B2B seit PROJ-74).
- Bankverbindungs-Block im Formular bleibt unverändert: konditional auf
  `einzugsart != kein_sepa` (heutiges Verhalten, kein Eingriff).

### Zwei verbleibende per-EEG-Toggles

| Toggle | An | Aus |
|---|---|---|
| **CORE-Audit-Toggle** (`sepa_mandate_core_audit_enabled`) | PDF enthält Audit-Trail-Block. Mitglied muss nichts mehr tun. | PDF enthält klassisches Unterschriftenfeld. Mitglied unterschreibt und sendet zurück. |
| **Timing-Toggle** (`sepa_mandate_at_import`) | PDF kommt erst beim Import — Mandatsreferenz = Mitgliedsnummer. | PDF kommt beim Submit (Bestätigungs-Mail) — Mandatsreferenz = Antrags-Referenznummer. |

**Label-Wechsel im UI:**
- alt: „SEPA-Mandat erst beim Import senden (mit Mitgliedsnummer als Mandatsreferenz)"
- neu: **„SEPA-Mandat erst beim Import senden (Mandatsreferenz = Mitgliedsnummer)"**
- Owner-Anmerkung 2026-06-08: Label muss klassik- und audit-neutral
  formuliert sein, weil der Timing-Toggle in beiden Varianten Sinn hat.

### Verhaltens-Matrix (3 gültige Kombinationen, 1 verboten)

Owner-Klarstellung 2026-06-08: **CORE-Audit aktiv erzwingt Timing aktiv**.
Begründung: beim Audit-Pfad hat das Mitglied keine Aktion mehr — würde
das PDF beim Submit raus, käme es ohne Mandatsreferenz (Mitgliedsnummer
existiert noch nicht). Das wäre ein „unvollständiges Mandat" beim
Mitglied. Ein Mandat ohne Mandatsreferenz hat keine eindeutige Zuordnung
zu einer späteren Lastschrift — rechtlich und buchhalterisch wertlos.

| CORE-Audit | Timing | Verhalten |
|---|---|---|
| aus | aus | PDF beim Submit, mit Unterschriftenfeld, Mandatsreferenz = Antragsnummer. Mitglied unterschreibt + sendet zurück. |
| aus | an | PDF erst beim Import, mit Unterschriftenfeld, Mandatsreferenz = Mitgliedsnummer. Mitglied unterschreibt + sendet zurück. (= heutiger Digital-Signatur-Workflow) |
| **an** | **aus** | **VERBOTEN — wird auto-korrigiert auf Timing=an** (siehe AC-3a Cross-Field-Validation) |
| an | an | PDF erst beim Import, mit Audit-Trail-Block, Mandatsreferenz = Mitgliedsnummer. Mitglied muss nichts mehr tun. |

## User Stories

- Als **EEG-Admin** möchte ich keinen verwirrenden Toggle mehr sehen,
  der nach PROJ-74 ohnehin nur noch teilweise gewirkt hat — die SEPA-
  Sektion soll nur die zwei wirklich relevanten Entscheidungen
  (Audit/Klassik, Submit/Import) übrig lassen.
- Als **Mitglied** möchte ich im Formular klar erkennen, dass meine
  Online-Zustimmung verbindlich ist — die Pflicht-Checkbox in der
  Bankverbindungs-Card erscheint immer (nicht nur bei bestimmten
  EEG-Konfigurationen).
- Als **EEG-Vorstand mit konservativem Compliance-Profil** möchte ich
  weiter ein vom Mitglied unterschriebenes PDF zurückbekommen — dafür
  lasse ich beide Toggles aus (Default-Stand nach Migration für die
  meisten Bestands-EEGs).
- Als **EEG-Vorstand mit modernem Onboarding-Setup** möchte ich keine
  unterschriebenen PDFs mehr zurückerhalten müssen — dafür aktiviere ich
  den CORE-Audit-Toggle.

## Akzeptanzkriterien

### AC-1: Migration mit Backfill
- Neue Migration `000071_drop_sepa_mandate_enabled.{up,down}.sql`:
  - **Backfill VOR DROP:**
    ```sql
    UPDATE member_onboarding.registration_entrypoint
    SET sepa_mandate_core_audit_enabled = TRUE,
        sepa_mandate_at_import = TRUE
    WHERE sepa_mandate_enabled = FALSE;
    ```
    EEGs, die heute „nur Online-Zustimmung" haben (`sepaMandateEnabled=false`),
    bekommen **beide** neuen Toggles aktiviert: Audit-Pfad (damit Mitglied
    nichts unterschreiben muss) UND Import-Timing (damit das Mandat vollständig
    ist — Mandatsreferenz = Mitgliedsnummer). Siehe AC-3a Cross-Field-Validation.
  - **Drop:** `ALTER TABLE member_onboarding.registration_entrypoint DROP COLUMN sepa_mandate_enabled;`
- Down-Migration: `ADD COLUMN sepa_mandate_enabled BOOLEAN NOT NULL DEFAULT TRUE;` + Rück-Backfill `UPDATE … SET sepa_mandate_enabled = FALSE WHERE sepa_mandate_core_audit_enabled = TRUE;` (Best-Effort-Restore, nicht perfekt invers — Timing-Toggle bleibt im Down-Pfad unverändert, da Spalte schon vor PROJ-80 existierte).

### AC-2: Backend-Struct + Repo
- `shared.RegistrationEntrypoint.SEPAMandateEnabled` entfernt
- `registration_entrypoint_repo.go`:
  - SELECT-Spalte entfernt
  - Scan-Variable entfernt
  - `SaveEEGSettings`-Signatur: `sepaMandateEnabled bool`-Parameter entfernt, UPDATE-SET-Klausel entfernt
- `registration_entrypoint_repo_tx.go`:
  - `EEGSettingsForImport.SEPAMandateEnabled` entfernt
  - `SaveAllEEGSettingsTx` UPDATE-SET-Klausel entfernt

### AC-3a: Cross-Field-Validation (CORE-Audit ⇒ Timing)

Owner-Klarstellung 2026-06-08: CORE-Audit aktiv ohne Timing ist
unzulässig (würde unvollständiges Mandat beim Submit liefern). Drei
Schichten erzwingen das:

- **Migration-Backfill** (AC-1): EEGs mit heute `sepaMandateEnabled=FALSE`
  bekommen **beide** Spalten auf TRUE gesetzt (`sepa_mandate_core_audit_enabled=TRUE` UND `sepa_mandate_at_import=TRUE`)
- **Backend-Validation** im `SaveEEGSettings`-Handler: wenn Body
  `sepaMandateCoreAuditEnabled=TRUE` und `sepaMandateAtImport=FALSE`,
  antwortet 400 mit Feld-Hinweis `sepaMandateAtImport: "Bei aktivem CORE-Audit-Pfad muss das Mandat erst beim Import gesendet werden."`. Alternative-Design (geprüft, verworfen): Silent-Auto-Set
  `sepa_mandate_at_import=TRUE` im Backend ohne Fehler. Verworfen, weil
  das Frontend dann state-divergent zur DB ist (Toggle steht „aus" obwohl
  DB „an" hat) — entweder Reload nötig oder verwirrend.
- **Frontend-Coupling** im Settings-Editor: bei Aktivieren des CORE-
  Audit-Toggles wird der Timing-Toggle automatisch mit auf TRUE gesetzt
  UND `disabled`. Disable bleibt, solange CORE-Audit aktiv ist. Bei
  Deaktivieren des CORE-Audit-Toggles wird Timing-Toggle wieder
  freigegeben (bleibt aber bei TRUE — Admin kann es manuell auf FALSE
  zurückstellen, wenn er weg vom Import-Pfad will).
- **Tooltip-Erklärung** am disabled Timing-Toggle: „Bei aktivem CORE-
  Audit-Pfad ist Import-Timing zwingend, weil das Mandat-PDF erst mit
  der Mitgliedsnummer als Mandatsreferenz vollständig ist."

### AC-3: Service-Layer
- `buildSEPAMandateData` CORE-Pfad:
  - alt: `if !ep.SEPAMandateEnabled || len(MissingMandateFields(ep)) > 0 { return nil }`
  - neu: `if len(MissingMandateFields(ep)) > 0 { return nil }` — kein Toggle-Check mehr, PDF wird immer erzeugt, wenn die Stammdaten reichen.
- Service-Layer-Test `TestBuildSEPAMandateData_CoreWithoutSepaMandateEnabledReturnsNil` entfällt; neuer Test `TestBuildSEPAMandateData_CoreAlwaysReturnsMandateWhenFieldsComplete` ergänzt.

### AC-4: HTTP-Handler
- `GetEEGSettings`-Response-Map: `sepaMandateEnabled`-Key entfernt
- `SaveEEGSettings`-Body-Struct: `SEPAMandateEnabled bool`-Feld entfernt, Save-Aufruf-Parameter entfernt

### AC-5: Configexport (Legacy-Kompat-Pattern)
- `schema.go`: `SEPAMandateEnabled bool` → **entfernt** UND ersetzt durch
  Legacy-Pointer `LegacySEPAMandateEnabled *bool \`json:"sepaMandateEnabled,omitempty"\``
  (Pattern analog `LegacyUseCompanySEPAMandate` aus PROJ-73). Importer
  ignoriert den Wert, Exporter setzt ihn nicht mehr — Pre-PROJ-80-Bundles
  parsen weiterhin mit dem strikten `DisallowUnknownFields`-Decoder.
- `exporter.go`: SEPAMandateEnabled-Zuweisung entfernt
- `importer.go`: SEPAMandateEnabled-Zuweisung entfernt; bei nicht-nil `LegacySEPAMandateEnabled` im Bundle wird `slog.Info` mit `event="legacy_field_ignored"`, `field="sepaMandateEnabled"`, `rc_number`, `value` geloggt (Audit-Trail für zukünftige Bundle-Migrationen)
- `diff.go`: `cmp("sepaMandateEnabled", …)` entfernt

### AC-6: Settings-Editor-Frontend
- `useState`, Snapshot-Type, Save-Payload, reloadSettings-Hydration,
  currentSnapshot, discardChanges für `sepaMandateEnabled` entfernt
- UI-Block (Switch + Label + Popover für „SEPA-Mandat als Datei dem
  Antragsteller übermitteln …") komplett entfernt
- Warn-Banner für fehlende EEG-Stammdaten bleibt — wird umgebaut auf
  unkonditionalen Default-Text (keine `sepaMandateEnabled`-Verzweigung
  mehr). Beide Pfade des Banners zeigen heute denselben Bug-Modus an
  („fehlende Pflichtfelder", siehe registration-form-Schema), nur die
  Wortwahl variiert. Neuer Wortlaut: konservativ, vereint beide Fälle.
- Conditional-Block `{isAdvanced && sepaMandateEnabled && (…)}` für
  Timing-Toggle + CORE-Audit-Toggle umstellen auf `{isAdvanced && (…)}`
  — beide Sub-Toggles waren bisher hinter `sepaMandateEnabled` versteckt,
  jetzt sind sie auf Top-Level der SEPA-Sektion. CORE-Audit-Toggle und
  Timing-Toggle stehen damit auf derselben Hierarchie-Ebene wie der B2B-
  Audit-Toggle (alle drei sind unabhängig).

### AC-7a: Kurz-Erklärungen unter den SEPA-Toggles (Konsistenz mit anderen Settings)

Owner-Anmerkung 2026-06-08 (mit Screenshot-Beleg): die anderen
Einstellungen im Editor (Vorstands-Genehmigungs-Workflow,
E-Mail-Adresse bestätigen, Aktivierungs-Kriterium, Zählpunkt-Prefixes)
folgen einem konsistenten Pattern — Label + Info-Icon (Popover für
Detail) **plus** eine **kurze Erklärung als Text direkt unter dem
Label** in `text-xs text-muted-foreground`. Die drei SEPA-Toggles
(CORE-Audit, B2B-Audit, Timing) zeigen heute nur das Info-Icon ohne
Kurz-Erklärung.

Für PROJ-80 alle drei SEPA-Toggles auf das konsistente Pattern bringen:

- **CORE-Audit-Toggle** (PROJ-78, hier UI-Layout-Update):
  > Aktiv: das Mandat-PDF enthält einen Audit-Trail mit Zustimmungs-
  > Zeitpunkt + IP — das Mitglied muss nichts zurücksenden, das Mandat
  > geht erst beim Import mit der Mitgliedsnummer als Mandatsreferenz
  > raus. Aus (Standard): das PDF hat ein Unterschriftenfeld, das
  > Mitglied unterschreibt und sendet zurück.

- **B2B-Audit-Toggle** (PROJ-78, hier UI-Layout-Update):
  > Aktiv: das Firmenlastschrift-PDF enthält einen Audit-Trail mit
  > Zustimmungs-Zeitpunkt + IP — das Mitglied muss nichts zurücksenden.
  > Aus (Standard): das PDF hat ein Unterschriftenfeld, das Mitglied
  > unterschreibt und sendet zurück. Wirkt unabhängig vom CORE-Toggle,
  > weil die Rechtsbewertung für Firmenlastschriften anders ausfallen
  > kann.

- **Timing-Toggle** (PROJ-48, hier umbenannt + UI-Layout-Update):
  > Aktiv: das Mandat-PDF kommt erst beim Import in eegFaktura mit der
  > Mitgliedsnummer als Mandatsreferenz. Aus (Standard): das PDF kommt
  > sofort bei der Bestätigungs-Mail mit der Antragsnummer als
  > Mandatsreferenz. Notwendig, wenn das Mandat digital signiert wird
  > (signiertes PDF darf nicht mehr verändert werden).

Pattern-Vorlage siehe `requireEmailConfirmation`-Toggle im selben
Editor — `<div className="space-y-1"><div className="flex items-center
gap-1"><Label/><Popover/></div><p className="text-xs
text-muted-foreground">Kurz-Erklärung</p></div>`.

### AC-7b: Timing-Toggle umbenannt
- Label-Wechsel im Settings-Editor:
  - alt: „SEPA-Mandat erst beim Import senden (mit Mitgliedsnummer als Mandatsreferenz)"
  - neu: **„SEPA-Mandat erst beim Import senden (Mandatsreferenz = Mitgliedsnummer)"**
- DB-Spaltenname (`sepa_mandate_at_import`) + Go-Feld (`SEPAMandateAtImport`)
  + JSON-Feld (`sepaMandateAtImport`) bleiben unverändert — reine
  UI-Bezeichnungsänderung.
- Hint-Popover ggf. anpassen: Audit-Pfad ist auch betroffen
  (PDF kommt erst beim Import, Audit-Block referenziert ursprünglichen
  Submit-Zeitpunkt + IP).

### AC-8: Mitglieder-Formular
- `registration-form.tsx`:
  - Validation in `buildFormSchema`: SEPA-Mandat-Akzeptanz (`sepaMandateAccepted`) **immer Pflicht** (alt: `!sepaMandateEnabled && !data.sepaMandateAccepted`); `sepaMandateEnabled`-Parameter entfällt
  - Hinweis-Absatz „Das SEPA-Lastschriftmandat erhältst du nach der Freigabe deines Antrags …" bei aktivem Timing-Toggle: alt `sepaMandateEnabled && sepaMandateAtImport`, neu nur `sepaMandateAtImport`
  - Online-Zustimmung-Checkbox im Bankverbindungs-Block: **immer
    sichtbar** (alt: `!sepaMandateEnabled`). PROJ-75-Kommentar aktualisieren
    (alte Begründung „nur bei Online-Zustimmungs-Lösung" entfällt)
  - `config.sepaMandateEnabled` aus `useState`/Props entfernt
- `RegistrationConfig`-Typ in `api.ts`: `sepaMandateEnabled?` entfernt

### AC-9: Tests
- Backend-Tests anpassen:
  - `TestBuildSEPAMandateData_CoreWithoutSepaMandateEnabledReturnsNil` entfällt
  - Neuer Test: CORE-Pfad liefert non-nil bei vollständigen Stammdaten, unabhängig von einem (nicht mehr existierenden) Toggle
  - `baseEntrypoint(true)`/`baseEntrypoint(false)`-Helper anpassen (Parameter entfällt)
- Frontend-Tests:
  - `settings-mode.test.ts` `defaultEegSettings` ohne `sepaMandateEnabled`
  - keine neuen Tests nötig — `isAdvancedEEGSettingsActive` reagiert nicht auf den entfernten Toggle (war nie Advanced-Trigger)
- Form-Test (falls vorhanden): SEPA-Mandat-Akzeptanz immer Pflicht-Check

### AC-10: EEG-Kopie des SEPA-Mandat-PDF bei Audit-Trail-Variante

Owner-Ergänzung 2026-06-08: Bei aktivem `sepa_mandate_core_audit_enabled`
hat das Mitglied keine Rücksende-Pflicht — das ausgefüllte Dokument
existiert nur als E-Mail-Anhang an das Mitglied. Die EEG braucht aber
eine eigene Ablage-Kopie (Aufbewahrungspflicht, Audit-Trail-Beleg).

**Pfad (eindeutig wegen Coupling AC-3a):** CORE-Audit aktiv ⇒ Timing
aktiv ⇒ PDF kommt über `SendMandateAtImportNotification`
([internal/mail/service.go:925](internal/mail/service.go#L925)). Submit-
Pfad kommt im Audit-Modus nie zum Zug.

**Verhalten:**
- Wenn CORE-Audit-Toggle TRUE → unmittelbar nach erfolgreicher Mitglieds-
  Mail wird **eine separate Mail** an `registration_entrypoint.contact_email`
  versendet mit derselben PDF-Kopie im Anhang.
- Wenn CORE-Audit-Toggle FALSE → keine zusätzliche EEG-Mail (Mitglied
  sendet das unterschriebene Original zurück → EEG hat eine Original-
  Kopie aus dem normalen Rücksende-Workflow).

**Mail-Format:**
- **Subject:** `Ablage-Kopie: SEPA-Mandat — {Mitgliedsname}, Antrag {Referenznummer}`
- **Body:**
  ```
  Anbei eine Kopie des SEPA-Lastschriftmandats für {Mitgliedsname}
  (Antrag {Referenznummer}). Das Mitglied hat dem Mandat am {Datum,
  formatiert DD.MM.YYYY} elektronisch zugestimmt (siehe Audit-Trail-
  Block im PDF). Diese Mail dient ausschließlich Ihrer eigenen Ablage.
  ```
  + Standard-Signatur (analog zu anderen System-Mails)
- **Empfänger:** `registration_entrypoint.contact_email` (To — nicht BCC)
- **Anhang:** identische PDF-Bytes wie an das Mitglied (kein erneutes
  Rendern, damit Datum/Audit-Block-Inhalt 100% gleich sind)

**Fehler-Verhalten:**
- **Fehlender `contact_email`** → Silent-Skip mit `slog.Warn` (Log-
  Felder: `application_id`, `rc_number`, `reason="contact_email_missing"`).
  Mitglieds-Mail bleibt unberührt.
- **SMTP-Outage** → Best-effort. Mitglieds-Mail ist bereits raus (sie ist
  im heutigen `SendMandateAtImportNotification` sync hart-fail; sie geht
  als erste durch und entscheidet über Import-Erfolg). EEG-Kopie wird
  separat versendet, danach. Schlägt sie fehl → `slog.Warn`
  (`reason="smtp_outage_eeg_copy"`), Import-Status bleibt unverändert.
- Konsequenz: der Mitglieds-Pfad ist nie durch EEG-Kopie blockiert,
  Memory `feedback_mail_hard_fail` greift hier bewusst NICHT — die EEG-
  Ablage-Kopie ist eine Convenience, kein admin-getriebenes
  Bestätigungs-Event.

**PROJ-76-Interaktion:** Wenn ein EEG sowohl PROJ-76 (Vorstands-
Genehmigungs-Workflow) als auch PROJ-80-Audit aktiv hat, bekommt die
EEG-Adresse **zwei** separate Mails:
1. Vorstandsmail mit Beitrittserklärungs-PDF (PROJ-76,
   `SendBoardApprovalRequest`)
2. Ablage-Kopie SEPA-Mandat (PROJ-80, neuer Pfad)

Keine Verschmelzung — saubere Trennung der Mail-Zwecke.

### AC-10b: Bankverbindungs-Label „Kontoinhaber:in" → „Kontowortlaut" (Tester-Bitte)

Owner-Ergänzung 2026-06-08 (Tester-Bitte): Bei gemeinsamen Haushaltskonten
mit zwei Kontoinhabern haben Mitglieder bisher nur den eigenen Namen
eingetragen, statt den exakten Kontowortlaut (inkl. zweiter Person).
Ergebnis: SEPA-Mandat-Name passt nicht zum Konto-Wortlaut, Lastschrift
wird abgelehnt.

- **Label-Wechsel** in `registration-form.tsx`
  ([Zeile 1488](src/components/registration-form.tsx#L1488)):
  - alt: „Kontoinhaber:in *"
  - neu: **„Kontowortlaut *"**
- **Validation-Fehlertext** in `buildFormSchema`
  ([Zeile 164](src/components/registration-form.tsx#L164)):
  - alt: „Kontoinhaber:in ist erforderlich"
  - neu: **„Kontowortlaut ist erforderlich"**
- **Hint-Popover** am Label (neu, Pattern aus
  `.claude/rules/frontend.md`):
  > Bitte den exakten Wortlaut aus dem Konto-/Bankauszug eintragen — bei
  > gemeinsamen Konten beide Namen. Der Kontowortlaut muss mit dem Konto
  > übereinstimmen, sonst lehnt die Bank die SEPA-Lastschrift ab.
- **DB-Feldname bleibt** `account_holder` / Go-Feld `AccountHolder` /
  JSON-Feld `accountHolder` — reine UI-Bezeichnungsänderung, keine
  Schema-Migration.
- **PDF-Render-Pfade** unverändert: das SEPA-Mandat-PDF nutzt weiterhin
  `accountHolder` als Name im „Zahlungspflichtiger Name"-Feld (siehe
  `buildSEPAMandateData` Zeile 1641-1644). Das Mitglied trägt jetzt
  konsequenter den ganzen Kontowortlaut ein → PDF zeigt automatisch
  beide Namen.

### AC-11: Doku
- `docs/api-spec.md`: `sepaMandateEnabled` aus GET/PUT-Body entfernen,
  Beschreibung im Settings-Endpoint anpassen; Cross-Field-Validation
  (CORE-Audit ⇒ Timing) dokumentieren
- `docs/domain-model.md`: `sepa_mandate_enabled`-Zeile entfernen, neue
  Erläuterung der Drei-Permutationen-Matrix (3 gültige Kombinationen,
  1 verboten) inkl. Cross-Field-Coupling
- `docs/user-guide/06-admin-settings.md`: Sektion „SEPA-Lastschriftmandat"
  komplett umschreiben (PROJ-frei):
  - knapper Einleitungsabsatz: „Das System erzeugt für jedes SEPA-
    Mitglied automatisch ein Mandat-PDF. Die zwei Toggles unten
    entscheiden, wie das PDF aussieht und wann es verschickt wird."
  - „Was bleibt jetzt immer gleich" — drei-Punkte-Aufzählung:
    PDF-Erzeugung, Online-Zustimmung-Checkbox, Bankverbindungspflicht
    (bei `einzugsart != kein_sepa`)
  - **Drei-Permutationen-Tabelle** (Markdown), Format wie heutige
    Vier-Permutationen-Tabelle aber kompakter: CORE-Audit × Timing →
    Verhalten/PDF-Versand-Zeitpunkt/Mandatsreferenz/Mitglied-Aktion/
    EEG-Kopie. Die verbotene Kombination explizit erwähnen mit Hinweis
    auf Auto-Coupling.
- `docs/user-guide/changelog.md`: User-facing Eintrag mit folgendem
  Wortlaut (Owner-bestätigt 2026-06-08):
  > **SEPA-Einstellungen vereinfacht**
  >
  > Der Schalter „SEPA-Mandat als Datei dem Antragsteller übermitteln"
  > entfällt. Das System erzeugt jetzt für jedes Mitglied automatisch
  > ein SEPA-Mandat-PDF. Zwei Schalter steuern weiterhin die Variante:
  >
  > - „Im CORE-Mandat den elektronischen Audit-Trail nutzen" entscheidet,
  >   ob das PDF einen Audit-Trail-Block enthält (kein Rücksenden nötig)
  >   oder ein klassisches Unterschriftenfeld.
  > - „SEPA-Mandat erst beim Import senden" entscheidet, ob das PDF
  >   sofort mit der Bestätigungs-Mail rausgeht (Mandatsreferenz =
  >   Antragsnummer) oder erst beim Import (Mandatsreferenz =
  >   Mitgliedsnummer).
  >
  > Wenn der Audit-Trail aktiv ist, wird das Mandat automatisch erst
  > beim Import gesendet, weil das PDF beim Submit-Zeitpunkt noch keine
  > Mitgliedsnummer hätte und damit unvollständig wäre.
  >
  > Wenn der Audit-Trail aktiv ist, bekommt zusätzlich die EEG-Kontakt-
  > adresse eine Ablage-Kopie des Mandats — das Mitglied muss nichts
  > zurücksenden, daher hat die EEG sonst kein Beleg-Exemplar.
  >
  > EEGs, die bisher mit reiner Online-Zustimmung gearbeitet haben
  > (ohne PDF-Versand), bekommen automatisch den Audit-Trail-Modus und
  > das Import-Timing aktiviert. Damit bleibt das Mitglieder-Erlebnis
  > gleich (keine Unterschrift nötig), und die EEG hat ein vollständig
  > dokumentiertes Mandat.
- `CHANGELOG.md`: voller technischer Eintrag mit Backend/Frontend/
  Migration/Tests/Doku-Details

## Edge Cases

1. **Bestands-EEGs mit `sepaMandateEnabled=true`** (= heutiges Default-
   Verhalten: PDF mit Unterschriftenfeld bei Submit): bekommen beide
   neuen Toggles auf FALSE durch die Default-Klauseln (PROJ-78), also
   identisches Verhalten wie heute. Kein Backfill nötig.
2. **Bestands-EEGs mit `sepaMandateEnabled=false`** (= heute: nur
   Online-Zustimmung, kein PDF): bekommen via Migration-Backfill
   `sepa_mandate_core_audit_enabled = TRUE`, damit ihr Mitglied weiter
   nichts unterschreiben muss. Verhaltenswechsel: heute wurde kein PDF
   geschickt; ab PROJ-80 wird ein PDF mit Audit-Trail-Block geschickt.
   Owner-Direktive: Bestand-Kontinuität durch (b) — siehe Pre-Spec-
   Klärung 2026-06-08.
3. **`einzugsart=kein_sepa`-Anträge**: unverändert. `buildSEPAMandateData`
   default-Branch returnt weiterhin nil. Kein PDF. Bankverbindungs-Block
   im Formular wird konditional ausgeblendet (heutiges Verhalten).
4. **B2B-Anträge**: unverändert. PROJ-74 hat `sepaMandateEnabled` für
   B2B schon entkoppelt. B2B-PDF kommt immer beim Import, B2B-Audit-
   Toggle (PROJ-78) entscheidet PDF-Variante.
5. **Stammdaten-Mangel**: wenn `MissingMandateFields(ep)`-Liste nicht
   leer ist, returnt der Build-Helper nil → kein PDF, kein Anhang.
   Online-Zustimmung-Checkbox bleibt Pflicht — Mitglied stimmt
   weiterhin elektronisch zu, EEG muss Stammdaten nachpflegen, dann
   greift Resend (PROJ-70).
6. **Configexport-Round-Trip**: alte Bundles (Pre-PROJ-80) tragen
   `sepaMandateEnabled` im JSON. `LegacySEPAMandateEnabled *bool, omitempty`
   parst das, Importer ignoriert den Wert. Owner-Hinweis im Diff-Preview:
   das Feld wird nicht mehr angezeigt (Drop aus der Diff-Liste).
7. **Resend-Pfad (PROJ-70)**: nutzt `buildSEPAMandateData` → übernimmt
   aktuelles Verhalten. Wenn EEG vor Cleanup `sepaMandateEnabled=true`
   hatte und ein Antrag damals submitiert wurde, sieht der Antrag heute
   ein PDF mit Unterschrift. Nach Cleanup + Resend rendert das System
   ebenfalls PDF mit Unterschrift (beide neue Toggles default FALSE).
   Identisches Verhalten.

## Non-Goals

- KEIN Eingriff in den B2B-Pfad — `sepaMandateB2BAuditEnabled` (PROJ-78)
  bleibt, B2B-PDF-Generierung bleibt unverändert.
- KEIN Eingriff in `einzugsart=kein_sepa` — kein PDF, kein Bankverbindungs-
  Block, kein Online-Zustimmung-Checkbox (heutiges Verhalten).
- KEIN Eingriff in Bankverbindungs-Block-Logik im Formular — bleibt
  konditional auf `einzugsart != kein_sepa`.
- KEIN Eingriff in Mandatsreferenz-Manuell-Override (Admin-Edit-Form-
  Feld bleibt).
- KEINE Migration der DB-Spalte `sepa_mandate_at_import` — bleibt
  unverändert, nur Label im UI wird angepasst.
- KEINE rückwirkende Re-Generierung versendeter PDFs.

## Memory-Regeln

- `feedback_admin_field_full_chain` — 6-Layer-Removal (DB-Migration,
  Struct, Repo SELECT+UPDATE+Tx, Service-Layer, HTTP-Handler GET+PUT,
  Frontend api.ts + Editor + State + Save-Payload + reloadSettings +
  discardChanges + Form-Komponente)
- `feedback_no_proj_refs_in_user_doc` — User-Guide-Rewrite PROJ-frei
- `feedback_batch_changelog_with_code` — Doku im selben Commit wie Code
- `feedback_shared_helpers_for_parallel_paths` — keine neuen
  Parallel-Pfade entstehen; im Gegenteil, die Verzweigung
  `if !sepaMandateEnabled …` wird abgebaut

## Workflow

`/requirements` → `/grill-me` (Default — Schema-Drop + Migration-Backfill
+ Default-Verhaltensaenderung sind alle Trigger) → `/backend` (Standard-
Reihenfolge: Migration → shared/Models → Repo → Service → HTTP → Configexport
→ Tests; Doku im selben Commit) → `/frontend` (Settings-Editor + Form +
Tests) → `/qa` → `/security-review` (DB-Schema-Change Pflicht-Trigger) →
`/deploy` (`v1.21.0-PROJ-80` Minor-Bump wegen Schema-Drop + Default-
Verhaltensaenderung).
