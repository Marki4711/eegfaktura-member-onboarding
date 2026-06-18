# PROJ-114: Beitrittsbestätigung an die EEG statt an das Mitglied (Vorstand leitet weiter)

## Status: In Progress
**Created:** 2026-06-18
**Last Updated:** 2026-06-18

## Implementation Notes (Backend — 2026-06-18)

Full-Chain umgesetzt (G8), kein neues Paket:
- **Migration 000090** (additiv): `joining_confirmation_to_eeg BOOLEAN NOT NULL DEFAULT FALSE` auf `registration_entrypoint`.
- **shared.RegistrationEntrypoint**: Feld `JoiningConfirmationToEEG`.
- **registration_entrypoint_repo.go**: `GetByRCNumber` SELECT+Scan (einziger Full-Scan-Pfad); `SaveEEGSettings` (Admin-Pfad) Param + UPDATE.
- **registration_entrypoint_repo_tx.go**: `EEGSettingsForImport`-Feld + `SaveAllEEGSettingsTx` UPDATE (Configexport-Import-Pfad).
- **admin_settings_eeg.go**: GET-Map + Save-Body + **D2-Validierung** (lädt `ep`, prüft synchronisierten `contact_email`; Toggle=true + leer → 400 mit Hinweis auf eegFaktura-Sync) + Save-Call-Arg.
- **configexport** (G7): schema/exporter/importer/diff um den Toggle erweitert (Pointer-Muster wie board_approval; nil = Pre-PROJ-114 ⇒ FALSE).
- **Mail (G2/G3/G6)**: `SendActivationNotification`-Verzweigung — ON+contact_email → nur Forward-EEG-Mail (neues Template `application_activated_eeg_forward.html`, Subject „… – bitte weiterleiten", PDF angehängt, Reply-To Mitglied), Mitglied nichts; ON+leer → Fallback ans Mitglied + slog.Warn; OFF → Bestand (Member-Mail + EEG-Kopie). pdfFailed-Branch im Forward-Template mit „Erneut senden"-Hinweis.
- **Tests**: Mail-Routing (3 Pfade ON/ON-leer/OFF), Configexport-Roundtrip. `go build/vet/test ./...` grün, gosec 0 neu (4 Bestand-Findings in fremden Dateien).
- D2-Handler-Validierung: kein DB-loses HTTP-Harness vorhanden → Verifikation auf Code-Ebene in /qa.

## Implementation Notes (Frontend — 2026-06-18)

- **applications.ts**: `joiningConfirmationToEEG` in `EEGSettings` (optional) + `EEGSettingsSavePayload` (required).
- **admin-eeg-settings-editor.tsx**: Toggle direkt unter dem PROJ-76-Vorstands-Schalter (Advanced-Sektion), mit Hinweis-Popover (kein placeholder). **Client-Sperre (D2)**: `disabled` solange keine `contactEmail` UND aktuell OFF (ein bereits gesetztes ON bleibt ausschaltbar — Defense gegen Config-Import-Zustand) + amber-Hinweis „in eegFaktura hinterlegen + synchronisieren". Voll in den PROJ-84-Auto-Save-Pfad eingereiht (State + Snapshot-Type + Load + currentSnapshot + autoSave-Payload + onSaved-Payload + discard).
- **admin-legal-documents-editor.tsx**: Full-Snapshot-Save um das Feld ergänzt (sonst hätte der Policy-Toggle-Save den Wert auf false zurückgesetzt — Full-Replace-Disziplin).
- tsc + vitest + `npm run build` grün.

Offen: **/qa** + **/deploy** (Migration 000090 → Schema-Change-Deploy).

## Dependencies
- Requires: PROJ-53/46 (Beitrittsbestätigungs-Mail mit PDF beim `activated`-Übergang, `SendActivationNotification`).
- Related (getrennt!): PROJ-76 (Vorstands-Genehmigungs-Workflow — routet die **Beitrittserklärung**, ein anderes Dokument, an die EEG). PROJ-114 betrifft die **Beitrittsbestätigung** (nach Aktivierung). Eigener, unabhängiger Schalter.
- Related: PROJ-84 (Auto-Save EEG-Settings-Editor), PROJ-98 (Anrede-Konditional), PROJ-32 (contact_email-Herkunft).

## Problem / Begründung
Tester-Feedback 2026-06-18: Der Vorstand möchte die Beitrittsbestätigung **selbst** an das neue Mitglied schicken — mit persönlicher Grußnachricht / Zusatzinfos. Heute geht die Beitrittsbestätigung (PDF + Mail) beim Aktivierungs-Übergang direkt ans Mitglied (`app.Email`), plus eine separate EEG-Kopie/Notification an `contact_email`. Gewünscht: eine **per-EEG-Option**, die die Beitrittsbestätigung **an die EEG umleitet** und die Mitglieds-Mail unterdrückt — der Vorstand leitet sie dann eigenhändig weiter.

## Owner-Entscheidungen (2026-06-18)

| # | Entscheidung | Wahl |
|---|---|---|
| D1 | **Mitglied im EEG-Modus** | Bekommt **gar keine** Beitrittsbestätigungs-Mail vom System (1A). Der Vorstand schickt alles. |
| D2 | **Kontakt-E-Mail NULL** | Schalter ist **nicht aktivierbar** ohne gesetzte `contact_email` — serverseitige Validierung blockt das Speichern (2B). |
| D3 | **Reichweite** | **Nur** die Beitrittsbestätigung (Aktivierungs-Mail). Status-Change-, SEPA-Mandat- und sonstige Mitglieds-Mails bleiben unverändert ans Mitglied (3A). |
| D4 | **EEG-Mail im Aktiv-Fall** | Die EEG bekommt **eine** Mail = das **Beitrittsbestätigungs-PDF + „bitte weiterleiten"-Vorspann**; sie ersetzt im Aktiv-Fall die bisherige separate EEG-Kopie. |
| D5 | **Default** | **OFF** — Bestand unverändert (Mitglied bekommt die Bestätigung direkt). |
| D6 | **Vorspann-Wortlaut** (Owner-approbiert, anpassbar) | „Anbei die Beitrittsbestätigung für {Mitglied}. Bitte leite sie an das Mitglied weiter — du kannst eine persönliche Nachricht ergänzen." |

## Grilling-Entscheidungen (Tech-Design-Stresstest 2026-06-18)

**Schlüssel-Befunde aus dem Code:**
- **`contact_email` ist Core-gemastert** (PROJ-32-Sync, in `SaveEEGSettings` bewusst NICHT akzeptiert — admin_settings_eeg.go:160-163). Der EEG-Admin kann sie **nicht im Onboarding** setzen; bei leerer Adresse muss er sie in eegFaktura hinterlegen + Stammdaten synchronisieren.
- **Die EEG-Kopie der Aktivierungs-Mail bekommt heute schon das PDF angehängt** (service.go:1238-1255). Der Vorstand hat das Beitrittsbestätigungs-PDF also bereits → das Feature ist schlank: Mitglieds-Mail unterdrücken + EEG-Mail als „bitte weiterleiten" umrahmen.

- **G1 — Spalte:** `joining_confirmation_to_eeg BOOLEAN NOT NULL DEFAULT FALSE` auf `registration_entrypoint` (Migration, Bestand unkritisch).
- **G2 — Mail-Routing:** Eine Verzweigung in `SendActivationNotification`. **ON:** eine Forward-EEG-Mail (eigenes Template, klarer „bitte weiterleiten"-Wortlaut + Subject „Beitrittsbestätigung für {Mitglied} – bitte weiterleiten") + PDF an `contact_email`; **keine** Mitglieds-Mail; **keine** zusätzliche alte EEG-Kopie. **OFF:** Bestand (Mitglieds-Mail + EEG-Kopie). Reply-To der Forward-Mail = Mitglied (`app.Email`, wie heute) → Vorstand kann direkt ans Mitglied antworten.
- **G3 — NULL-Fallback zur Sendezeit (EC-1):** ON, aber `contact_email` leer (Race/Re-Sync nach Aktivieren des Toggles) → Fallback an das Mitglied (Bestand-Mail) + Warn-Log. Defense-in-Depth zur D2-Validierung.
- **G4 — D2-Validierung + UX:** Server prüft beim Save den **synchronisierten** `ep.ContactEmail` (Handler hat `ep` geladen, Pattern wie PROJ-37/80/81). Toggle=true + leere contact_email → 400, Feld-Message: „Bitte zuerst in eegFaktura eine Kontakt-E-Mail hinterlegen und die Stammdaten synchronisieren." Frontend gated den Toggle gleichlautend (kein stiller Save).
- **G5 — Toggle-Platzierung:** nahe dem PROJ-76-Vorstands-Workflow-Schalter (thematisch „Vorstand"), mit Hinweis-Popover (kein placeholder).
- **G6 — `pdfFailed` im ON-Modus:** Die EEG bekommt trotzdem eine Mail mit Hinweis „Beitrittsbestätigung konnte nicht erzeugt werden — bitte im Admin unter ‚Erneut senden → Beitrittsbestätigung' erneut auslösen" (Pfad existiert, PROJ-97). Mitglied weiterhin nichts.
- **G7 — Config-Export/-Import (PROJ-61):** Neuer Toggle wird ins Konfig-Bundle aufgenommen (configexport-Schema + Exporter + Importer), konsistent mit `board_approval_workflow_enabled` — sonst wird er beim EEG-zu-EEG-Konfig-Import still nicht übertragen.
- **G8 — Full-Chain-Stellen** (feedback_admin_field_full_chain): (1) Migration; (2) `shared.RegistrationEntrypoint`-Feld + Scan; (3) `admin_settings_eeg.go` GET + Save-Body + D2-Validierung; (4) `registration_entrypoint_repo` `SaveAllEEGSettingsTx` (Import-Pfad) + Admin-Save-SQL; (5) configexport schema/exporter/importer (G7); (6) `SendActivationNotification`-Verzweigung + neues Forward-Template; (7) Frontend: TS-Type (`EEGSettings`/Save-Payload), Toggle im `admin-eeg-settings-editor`, Payload, Client-Gate.

## User Stories
- Als **EEG-Vorstand** möchte ich einstellen, dass die Beitrittsbestätigung an die EEG statt ans Mitglied geht, damit ich sie mit einer persönlichen Begrüßung selbst ans Mitglied weiterleiten kann.
- Als **EEG-Vorstand** möchte ich im Aktiv-Fall eine Mail mit dem fertigen Beitrittsbestätigungs-PDF und einem klaren „bitte weiterleiten"-Hinweis bekommen, damit ich sofort weiterleiten kann.
- Als **Mitglied** einer EEG mit aktiviertem Schalter bekomme ich die Bestätigung **vom Vorstand** (nicht doppelt vom System).
- Als **EEG-Admin** möchte ich den Schalter nur aktivieren können, wenn eine Kontakt-E-Mail hinterlegt ist, damit keine Bestätigung ins Leere läuft.
- Als **Betreiber** möchte ich, dass bestehende EEGs unverändert weiterlaufen (Default OFF).

## Acceptance Criteria

**Einstellung**
- [ ] AC-1: Neue per-EEG-Boolean-Einstellung „Beitrittsbestätigung an die EEG senden (Vorstand leitet weiter)" auf `registration_entrypoint`, Default FALSE.
- [ ] AC-2: Im EEG-Settings-Editor sicht- und umschaltbar (Auto-Save-Pfad PROJ-84), mit erklärendem Hinweis.
- [ ] AC-3 (D2): Aktivieren des Schalters ist **serverseitig blockiert**, wenn `contact_email` leer/NULL ist (400 mit klarer Feldmeldung); das Frontend gated den Toggle entsprechend (kein stiller Save).

**Mail-Routing bei Aktivierung (Schalter ON)**
- [ ] AC-4 (D1): Beim `activated`-Übergang bekommt das **Mitglied keine** Beitrittsbestätigungs-Mail.
- [ ] AC-5 (D4): Die **EEG** (`contact_email`) bekommt **eine** Mail = Beitrittsbestätigungs-**PDF** + Forward-Vorspann (D6). Diese ersetzt im Aktiv-Fall die bisherige separate EEG-Kopie (keine zwei EEG-Mails).
- [ ] AC-6: Das **PDF selbst ist unverändert** (gleiche Beitrittsbestätigung).

**Schalter OFF (Bestand)**
- [ ] AC-7: Unverändertes Verhalten — Mitglied bekommt Beitrittsbestätigung+PDF, EEG bekommt die bestehende Kopie/Notification.

**Abgrenzung**
- [ ] AC-8 (D3): Nur die Aktivierungs-/Beitrittsbestätigungs-Mail ist betroffen; alle anderen Mitglieds-Mails (Status-Change, SEPA-Mandat, B2B-Hinweis …) gehen unverändert ans Mitglied.
- [ ] AC-9: Unabhängig von PROJ-76 — beide Schalter können gleichzeitig aktiv sein und betreffen verschiedene Dokumente/Mails ohne Konflikt.

## Edge Cases
- **EC-1 (contact_email zur Sendezeit doch leer):** Obwohl AC-3 das Aktivieren ohne contact_email blockt, kann die Adresse später (z. B. Core-Re-Sync) leer werden. Zum Aktivierungs-Zeitpunkt: wenn Schalter ON aber contact_email leer → **Fallback ans Mitglied + Warn-Log** (niemand geht still leer aus).
- **EC-2 (PDF-Generierung fehlgeschlagen, `pdfFailed`):** Bestehendes Verhalten bleibt — die Mail wird mit PDF-Fehler-Hinweis versandt; im EEG-Modus geht sie an die EEG, das Mitglied bekommt nichts.
- **EC-3 (Re-Aktivierung, PROJ-100 Reset → erneut activated):** Schalter gilt erneut; Routing identisch.
- **EC-4 (PROJ-76 zusätzlich aktiv):** Beitrittserklärung-Routing (PROJ-76) + Beitrittsbestätigung-Routing (PROJ-114) gleichzeitig — kein Konflikt, verschiedene Mails.
- **EC-5 (Anrede-Konditional PROJ-98):** Der Forward-Vorspann an die EEG nutzt eine neutrale Anrede an den Vorstand; das (unveränderte) PDF behält seine Mitglieds-Anrede.

## Technical Requirements
- DB-Schema: neue Spalte auf `member_onboarding.registration_entrypoint` (Migration + Human-Approval-Checkpoint).
- Full-Chain für den Toggle (feedback_admin_field_full_chain): TS-Type, Form-Toggle, Save-Payload, Go-DTO, Service-Mapping, Repo-SQL.
- Mail-Routing-Änderung lokal in `SendActivationNotification` (Empfänger + Vorspann); kein neuer Mail-Typ, gleiches PDF.
- Validierung server- **und** clientseitig (Defense-in-Depth) für die contact_email-Vorbedingung.
- Keine Status-Transition-Änderung.

## Non-Goals
- Keine Änderung an PROJ-76 (Beitrittserklärung-Routing).
- Kein In-App-„Weiterleiten-mit-Nachricht"-Editor — der Vorstand leitet extern (eigener Mail-Client) weiter.
- Keine Änderung am PDF-Inhalt.
- Kein Umleiten anderer Mitglieds-Mails (nur Beitrittsbestätigung).

## Tech Design (Solution Architect)

### A) Datenfluss & Komponenten

```
EEG-Settings-Editor (Vorstands-Bereich)
  └── Toggle „Beitrittsbestätigung an die EEG senden (Vorstand leitet weiter)"
        │  Auto-Save (PROJ-84)
        ▼
   Speichern mit Vorbedingungs-Prüfung:
     Toggle EIN + keine Kontakt-E-Mail hinterlegt?
        → Speichern abgelehnt, Hinweis: „in eegFaktura hinterlegen + synchronisieren"
        │ sonst: gespeichert
        ▼
   EEG-Einstellung (registration_entrypoint): joining_confirmation_to_eeg = ja/nein
        │
        │  … später, beim Übergang eines Antrags auf „Aktiviert" …
        ▼
   Versand der Beitrittsbestätigung liest die Einstellung:
     ┌─ AUS (Standard) → wie bisher: Mitglied bekommt Bestätigung + PDF,
     │                   EEG bekommt ihre Kopie
     └─ EIN            → Mitglied bekommt NICHTS;
                         EEG bekommt EINE Mail = „bitte weiterleiten" + PDF
                         (Kontakt-E-Mail leer? → Notfall-Rückfall ans Mitglied)
```

Die Einstellung ist eine **reine Weiche beim Mail-Versand** — sie ändert nichts am Status-Ablauf, am PDF-Inhalt oder an anderen Mails.

### B) Datenmodell (Klartext)

- **Eine neue Ja/Nein-Einstellung pro EEG**: „Beitrittsbestätigung an die EEG statt ans Mitglied". Standard **Aus** → bestehende EEGs verhalten sich unverändert.
- **Vorbedingung**: Die Einstellung lässt sich nur einschalten, wenn die EEG eine **Kontakt-E-Mail** hat. Diese Adresse wird **aus eegFaktura übernommen** (nicht im Onboarding editierbar) — fehlt sie, muss die EEG sie dort hinterlegen und die Stammdaten synchronisieren.
- Keine weiteren Felder, keine neuen Tabellen.

### C) Tech-Entscheidungen (warum so)

- **Eigene „Weiterleiten"-Mail statt nur ein Schalter im alten Text:** Der Vorstand braucht einen klaren Hinweis „bitte ans Mitglied weiterleiten" und einen passenden Betreff. Ein eigener, kurzer Mail-Text ist verständlicher als die bisherige „Antrag aktiviert"-Notification umzudeuten. Das PDF ist ohnehin schon angehängt.
- **Vorbedingung Kontakt-E-Mail (Einschalt-Sperre):** Ohne Kontakt-Adresse ginge die Bestätigung ins Leere und das Mitglied bekäme gar nichts. Die Sperre verhindert genau diese stille Lücke — und weil die Adresse aus eegFaktura kommt, lenkt die Meldung den Admin an die richtige Stelle.
- **Notfall-Rückfall ans Mitglied (trotz Sperre):** Die Kontakt-Adresse kann nach dem Einschalten durch eine erneute Synchronisierung wieder leer werden. Damit in dem seltenen Fall niemand komplett leer ausgeht, fällt der Versand dann auf das Mitglied zurück (mit Warn-Eintrag im Log). Gürtel **und** Hosenträger.
- **Nur die Beitrittsbestätigung:** Bewusst eng — Status-Mails, SEPA-Mandat usw. bleiben unverändert, um Verwirrung zu vermeiden.
- **In den Konfig-Export aufnehmen:** Wie der bestehende Vorstands-Workflow-Schalter wandert die neue Einstellung in das EEG-Konfig-Bundle, damit sie beim Übertragen einer Konfiguration von EEG zu EEG nicht still verloren geht.

### D) Frontend-Auswirkung

Betroffen ist der **EEG-Settings-Editor** (`admin-eeg-settings-editor.tsx`):
- Neuer **Toggle** nahe dem Vorstands-Genehmigungs-Schalter, mit **Hinweis-Popover** (was passiert, dass der Vorstand selbst weiterleitet).
- **Gesperrt**, wenn keine Kontakt-E-Mail hinterlegt ist — mit Erklärung, dass sie aus eegFaktura kommt und synchronisiert werden muss (kein stiller Save; Backend lehnt zusätzlich ab).
- Reiht sich in den bestehenden **Auto-Save**-Pfad ein.

### E) Abhängigkeiten

**Keine neuen Pakete.** Genutzt werden die bestehenden Bausteine (Mail-Service + Templates, EEG-Settings-Editor, Konfig-Export-Mechanik, Migrations-Job). Eine additive DB-Migration (eine Spalte), kein neues Helm-Werk/Secret/Env.

### F) Empfohlene Umsetzungs-Reihenfolge

**`/backend` zuerst** (Migration + Mail-Routing-Weiche + Forward-Template + Einschalt-Validierung + Konfig-Export-Aufnahme), dann **`/frontend`** (Toggle + Hinweis + Client-Sperre). So steht das Verhalten serverseitig fest, bevor das UI es spiegelt. Danach `/qa` (Mail-Routing-Pfade beide Modi + Vorbedingung + Rückfall) → `/deploy` (mit Migration).

## Offene Punkte für /grill-me
- **Herkunft von `contact_email`:** Core-gemastert (PROJ-32-Sync) oder im Onboarding editierbar? Bestimmt, wie die D2-Validierung greift (gegen synchronisierten Wert vs. eingebbaren Wert) + die UX-Meldung.
- Genaue Stelle der bestehenden „EEG-Kopie" in `SendActivationNotification` und wie sie im Aktiv-Fall durch die kombinierte Mail ersetzt wird (eine Render-/Send-Verzweigung statt Duplikat).
- Platzierung des Toggles im Editor (eigener „Benachrichtigungen"-Bereich vs. nahe dem Vorstands-Workflow-Toggle) + Wortlaut Label/Hinweis.
- Reply-To/From im EEG-Modus (heute Reply-To = EEG-Kontakt; passt der Vorspann dazu).
