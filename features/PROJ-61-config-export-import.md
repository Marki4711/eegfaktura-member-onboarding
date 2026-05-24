# PROJ-61: Konfigurations-Export & -Import pro EEG

## Status: Planned
**Created:** 2026-05-24
**Last Updated:** 2026-05-24

## Dependencies
- Requires: PROJ-5 (Keycloak-Admin-Auth) — Tenant-Isolation für Zugriffsprüfung
- Requires: PROJ-8 (Konfigurierbare Felder) — Sub-Typ `field_config` ist Teil des Exports
- Requires: PROJ-9 + PROJ-36 (Rechtsdokumente) — Sub-Typ `legal_document` ist Teil des Exports
- Requires: PROJ-32 (EEG-Stammdaten-Sync) — implizit, weil `eeg_id`/`eeg_name` als Quell-EEG-Info im Export-Header stehen
- Requires: PROJ-60 (Datenweiterleitung) — Sub-Typ `data_export_config` ist Teil des Exports

## Hintergrund

Es kommt regelmäßig vor, dass eine einzelne Person mehrere EEGs im
Member-Onboarding-System verwaltet. Aktuell muss jede Konfiguration —
Feld-Sichtbarkeit, Rechtsdokumente, Datenweiterleitungs-Mappings,
EEG-Settings — manuell pro EEG nachgepflegt werden. Das ist:

- **Fehleranfällig**: vergessene Felder driften zwischen EEGs auseinander
- **Aufwändig**: bei n EEGs n-mal dieselbe Klick-Sequenz
- **Schwer zu auditieren**: keine Möglichkeit zu sagen „EEG-X und EEG-Y haben
  dieselbe Grund-Konfig", weil kein Diff-Tool existiert

Ziel: Admin kann seine fertige Konfig von Quell-EEG als JSON-File
exportieren und auf eine oder mehrere Ziel-EEGs identisch oder als
Ausgangspunkt für individuelle Anpassungen anwenden.

## User Stories

- Als Tenant-Admin, der zwei EEGs (A und B) verwaltet, will ich die
  fertige Konfig von EEG-A als Datei exportieren und auf EEG-B
  identisch anwenden, damit beide EEGs dasselbe Member-Onboarding-
  Verhalten zeigen.
- Als Tenant-Admin will ich beim Import vor dem tatsächlichen Apply
  einen **Diff sehen** (alt → neu), damit ich versehentliche
  Überschreibungen verhindern kann.
- Als Tenant-Admin will ich aus einem Bundle-File nur **einzelne
  Sektionen importieren** (z. B. nur Datenweiterleitung, nicht die
  Feld-Konfig), damit ich punktuell EEGs aneinander angleichen kann
  ohne andere Settings zu überschreiben.
- Als Tenant-Admin will ich **nur einen Sub-Typ** exportieren können
  (z. B. nur die Datenweiterleitungs-Configs), damit ich nicht jedes
  Mal alle 4 Sektionen herunterladen muss.
- Als Tenant-Admin will ich beim Import sehen, woher das File stammt
  (Quell-EEG-Name + Export-Datum), damit ich nachvollziehen kann,
  welche Konfig ich gerade anwende.

## Scope V1

**Vier exportierbare Sub-Typen:**

1. **EEG-Einstellungen** (Felder auf `registration_entrypoint`):
   - `intro_text`, `show_central_policy`, `require_email_confirmation`
   - SEPA: `sepa_mandate_enabled`, `use_company_sepa_mandate`, `sepa_mandate_at_import`
   - Zählpunkt-Prefixes: `metering_point_prefix_consumption`, `metering_point_prefix_production`
   - `activation_mode` (PROJ-53)
   - `participation_factor` (PROJ-37)
2. **Mitgliedsfeld-Konfig** (`field_config`-Tabelle): alle Einträge der Quell-EEG
3. **Rechtsdokumente** (`legal_document`-Tabelle): alle Einträge der Quell-EEG
4. **Datenweiterleitung** (`data_export_config`-Tabelle): alle nicht-deleted Einträge der Quell-EEG

**Bewusst exkludiert:**

- EEG-Stammdaten (`eeg_name`, `eeg_street`, `eeg_zip`, `eeg_city`,
  `eeg_id`) — kommen aus Core-Sync (PROJ-32), nicht editierbar
- Secrets: `external_api_key` — werden NIE exportiert (Prinzip „secrets
  bleiben pro EEG einzigartig")
- EEG-Kontakt: `contact_email`, `creditor_id` — EEG-spezifisch, kein
  generischer Konfig-Wert
- Sequence-State: `member_number_start` — exportierten Wert auf
  Ziel-EEG anzuwenden könnte zu Member-Number-Kollisionen führen
- Identifikation: `rc_number` — die ist die EEG selbst, nicht Konfig
- Legal-Document-URLs, die EEG-spezifisch sind: bleiben als Strings im
  Export drin (Admin sieht im Diff, was er anpassen muss); kein
  automatisches Scrubbing — würde mehr Verwirrung stiften, weil die
  URL-Konvention nicht maschinell erkennbar ist (`eeg-x.at/agb` vs
  `cloud-doc/12345`)

## Akzeptanzkriterien

### Export

- [ ] **AC-E1**: Im EEG-Admin-Bereich gibt es einen Button-Block
  „Konfig-Export". Beim Klick wird ein JSON-File heruntergeladen.
- [ ] **AC-E2**: Admin kann pro Sub-Typ (4 Buttons: „EEG-Einstellungen",
  „Feld-Konfig", „Rechtsdokumente", „Datenweiterleitung") einzeln
  exportieren ODER einen 5. Button „Komplett-Bundle" nutzen, der alle
  4 Sektionen kombiniert.
- [ ] **AC-E3**: Das exportierte JSON enthält einen Header:
  ```json
  {
    "schemaVersion": 1,
    "exportedAt": "2026-05-24T12:34:56Z",
    "exportedFrom": { "rcNumber": "RC...", "eegName": "Meine EEG" },
    "sections": { ... }
  }
  ```
- [ ] **AC-E4**: Die exportierten Daten enthalten KEINE EEG-spezifischen
  Felder (rc_number, eeg_id, contact_email, creditor_id,
  member_number_start, eeg_name/address). Bei Sub-Typen mit
  Composite-Keys (`legal_document.id`, `data_export_config.id`,
  `field_config.id`) werden die DB-IDs weggelassen — beim Import werden
  neue IDs generiert.
- [ ] **AC-E5**: Der Dateiname folgt dem Pattern
  `member-onboarding-config_<rcNumber>_<sub-typ-oder-bundle>_<YYYY-MM-DD>.json`
  damit Admin mehrere Files im Filesystem unterscheiden kann.
- [ ] **AC-E6**: Export ist read-only — keine DB-Mutation, kein
  status_log-Eintrag (verändert nichts).

### Import

- [ ] **AC-I1**: Im EEG-Admin-Bereich gibt es einen Button „Konfig-
  Import". Admin wählt eine JSON-Datei aus dem lokalen Filesystem.
- [ ] **AC-I2**: Das System validiert das File:
  - `schemaVersion` ist bekannt (V1 akzeptiert nur `schemaVersion: 1`)
  - JSON-Struktur ist syntaktisch korrekt
  - Jede Sektion entspricht dem erwarteten Sub-Typ-Schema
  - Bei Fehlern wird der Import abgelehnt mit einer konkreten
    Fehlermeldung
- [ ] **AC-I3**: Wenn das File mehrere Sektionen enthält, kann Admin
  per Checkbox auswählen, welche er importieren will (alle als Default
  vorausgewählt).
- [ ] **AC-I4**: Vor dem Apply wird ein **Diff-Preview** angezeigt:
  pro ausgewählter Sektion eine Tabelle mit „aktueller Wert auf
  Ziel-EEG" → „neuer Wert aus Import". Bei Listen-Sub-Typen
  (legal_document, data_export_config, field_config) wird gezeigt:
  N Einträge auf Ziel-EEG aktuell → M Einträge nach Apply.
- [ ] **AC-I5**: Admin bestätigt mit „Apply" — erst dann wird die
  Mutation ausgeführt. Alternativ „Abbrechen" verwirft.
- [ ] **AC-I6**: Beim Apply mit Multi-Section: alle ausgewählten
  Sektionen werden in **einer DB-Transaktion** angewendet — bei
  Fehler in einer Sektion wird die gesamte Änderung zurückgerollt.
- [ ] **AC-I7**: Replace-Semantik pro Sektion:
  - **EEG-Einstellungen**: Felder im Import werden auf Ziel-EEG
    überschrieben; nicht im Import enthaltene Felder bleiben
    unverändert (Forward-Compat falls V1.0 noch Setting X nicht
    kennt, das V1.1 hinzufügt).
  - **Field-Config**: bestehende `field_config`-Einträge der Ziel-EEG
    werden komplett gelöscht und durch Import-Einträge ersetzt.
  - **Legal-Documents**: bestehende `legal_document`-Einträge der
    Ziel-EEG werden komplett gelöscht und durch Import-Einträge
    ersetzt (neue IDs).
  - **Data-Export-Configs**: bestehende `data_export_config`-Einträge
    der Ziel-EEG werden auf `deleted_at = NOW()` gesetzt (Soft-Delete
    aus PROJ-60), neue Einträge aus dem Import werden mit neuen IDs
    angelegt. Bereits abgeschlossene Jobs bleiben durch
    `config_snapshot` weiterhin auditierbar.
- [ ] **AC-I8**: Nach dem Apply: Bestätigungs-Meldung mit Anzahl
  geänderter Einträge pro Sektion + Audit-Log-Eintrag in
  `status_log` (NICHT pro Application — separates Audit-Konzept; siehe
  Tech-Design).
- [ ] **AC-I9**: Plugin-Registry-Drift: enthält ein Import einen
  `data_export_config` mit `plugin_type`, der auf der Ziel-Instanz
  nicht registriert ist, wird der Eintrag mit `is_obsolete = TRUE`
  angelegt (analog zum laufenden Sweep aus PROJ-60); kein
  Import-Failure.
- [ ] **AC-I10**: Field-Catalog-Drift: enthält ein
  `data_export_config` Column-Mappings auf Field-Keys, die im
  aktuellen Katalog nicht existieren (z. B. weil Field zwischen
  Quell- und Ziel-Deployment entfernt wurde), werden diese
  Column-Einträge beim Import **verworfen** mit Warn-Hinweis im
  Diff-Preview — kein Import-Failure.

### Permissions

- [ ] **AC-P1**: Tenant-Admin sieht die Export/Import-UI auf seiner
  zugewiesenen EEG. Superuser sieht sie auf allen EEGs.
- [ ] **AC-P2**: Beim Import wird verifiziert, dass der Admin Zugriff
  auf die Ziel-EEG hat (`checkTenantAccess`). Die Quell-EEG aus dem
  Export-Header ist informational — kein Tenant-Check auf
  Quell-Seite (Admin könnte ein File von jemand anderem importieren,
  das ist OK).

## Edge Cases

- **File falsch formatiert**: kein JSON, kein Header, falsches
  `schemaVersion` → 400 mit konkreter Fehlermeldung, kein Apply.
- **File größer als sinnvolles Limit**: harte Grenze 1 MB (Configs sind
  klein, alles darüber ist Angriffsvektor) → 413.
- **Doppel-Klick auf Apply-Button**: idempotent durch Submit-Spinner +
  Server-side Concurrency-Check (PROJ-60 hat das schon für
  config-update, Pattern wiederverwenden).
- **Import-File hat Sektion, die V1 nicht kennt** (z. B. zukünftiges
  `notification_settings`): Sektion wird im Diff als „nicht
  unterstützt — wird ignoriert" angezeigt, kein Import-Failure;
  Forward-Compat-Verhalten.
- **Quell-EEG hat 0 Einträge in einer Sektion** (z. B. keine
  Datenweiterleitungs-Configs): Export enthält leere Liste; Import
  führt Replace mit leerer Liste aus → löscht alle Einträge der
  Ziel-EEG in dieser Sektion. **Im Diff klar als „X Einträge → 0
  Einträge" anzeigen**, damit Admin nicht versehentlich Daten verliert.
- **Cross-Schema-Version-Import**: V2 wird abgelehnt mit Hinweis
  „bitte aktuelles Member-Onboarding nutzen". V1-File auf V2-System
  wird akzeptiert (Forward-Compat ist Server-Pflicht).
- **Concurrent Edit**: Admin importiert, parallel ändert anderer Admin
  ein Setting auf der Ziel-EEG. Replace-Semantik gewinnt (Last-Write-
  Wins). Bewusste Vereinfachung; kein Optimistic-Locking in V1.
- **Großer Diff-Preview-Output**: bei 50+ field_config-Einträgen wird
  die Diff-Tabelle lang. Pro Sub-Typ ein eigener Collapse-Block in der
  UI; default geöffnet bei Sektionen mit < 5 Änderungen, sonst
  collapsed.
- **Import in produktiv-aktive EEG**: keine Sperre. Field-Config-
  Änderung wirkt sofort auf laufende Public-Form-Sessions (Caching ist
  per-Request, kein Cache-Invalidate nötig). Data-Export-Config-
  Änderung beeinflusst nur künftige Jobs (laufende Jobs nutzen
  `config_snapshot`).

## Non-Goals (explizit nicht in V1)

- **Diff zwischen zwei live EEGs (ohne File-Roundtrip)**: nice-to-have,
  aber File-basiert reicht für den Use-Case. Spätere PROJ.
- **Bulk-Apply auf mehrere Ziel-EEGs gleichzeitig**: V1 ist
  per-EEG-Import. Wer 5 EEGs angleichen will, importiert 5-mal.
  Sinnvolle V2-Erweiterung.
- **Versionierte Templates**: kein „Template-Repository", aus dem
  Admin auswählt. Files leben im Filesystem des Admins.
- **Schedule / Auto-Sync**: keine automatischen Cron-Imports.
- **Export von Audit-Daten** (status_log, application-history) — ist
  kein Konfig-Export, sondern Daten-Migration.
- **Export von Mitgliedsdaten** — fällt in PROJ-60-Scope, hat eigenen
  Workflow.
- **Cross-Instanz-Import** (von einem anderen member-onboarding-
  Deployment): das File-Schema ist nicht garantiert kompatibel
  zwischen Major-Versionen; wir dokumentieren „Import nur aus
  gleicher Major-Version".

## Technical Requirements

- **Response-Time**: Export < 1 s, Diff-Preview-Generation < 1 s, Apply
  < 2 s (alle Sub-Typen).
- **Security**: Tenant-Isolation strikt; secrets (API-Keys) NIEMALS
  exportiert; File-Upload-Größenlimit 1 MB; JSON-Schema-Validation
  serverseitig.
- **Auditability**: jeder Apply schreibt einen Eintrag in ein neues
  Audit-Konzept (Tech-Design entscheidet: eigene Tabelle vs Erweiterung
  status_log).
- **i18n**: UI-Texte deutsch (analog zum restlichen Admin-Bereich).
- **Browser-Support**: Chrome, Firefox, Safari (analog Rest des Admin-
  Bereichs).

## Offene Fragen für `/grill-me`

Folgende Punkte sind im Spec bewusst „so" entschieden, sollten aber
gegen Edge Cases gegrillt werden:

1. **Cross-EEG-Import-Audit**: brauchen wir ein eigenes
   `config_import_log` oder reicht ein generischer Admin-Event-Log?
2. **field_config-Replace-Semantik**: „komplett ersetzen" könnte Felder
   auf hidden setzen, die der Admin bewusst gesetzt hat. Reicht der
   Diff-Preview als Schutz?
3. **data_export_config Soft-Delete vs Hard-Replace**: PROJ-60 nutzt
   Soft-Delete für Audit. Beim Import löschen wir Alt-Configs soft.
   Aber: wenn der Admin 3-mal hin- und herimportiert, sammelt sich
   Müll in der Soft-Delete-Liste. Cleanup-Strategie?
4. **Plugin-Registry-Drift in Praxis**: ist `is_obsolete=true`-Import
   wirklich der richtige Default oder sollte Admin warnen + manuell
   bestätigen?
5. **Zählpunkt-Prefix-Übernahme**: bei zwei EEGs in unterschiedlichen
   Netzgebieten könnten die Prefixes unterschiedlich sein. Sollte das
   Field gesondert markiert sein („network-region-specific")?
6. **Schema-Version-Migration**: wenn V2 Settings hinzufügt, lädt V1-
   File V2 OK (Forward-Compat) — aber wenn V2 Settings UMBENENNT,
   wird's brüchig. Default-Mapping-Tabelle nötig?
7. **legal_document mit kaputter URL**: was wenn Admin EEG-A-URL
   einfach unverändert auf EEG-B importiert? Reicht der Diff oder
   sollte das Backend per HEAD-Request prüfen?

## Recommended Next Step

`/grill-me` (Default per requirements-Skill) — die Spec berührt
mehrere DB-Tabellen + Migration-/Forward-Compat-Logik + Audit. Findings
fließen zurück in diese Datei, bevor `/architecture` startet.

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
