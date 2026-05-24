# PROJ-61: Konfigurations-Export & -Import pro EEG

## Status: Architected
**Created:** 2026-05-24
**Last Updated:** 2026-05-24 (nach /architecture — Tech Design Sektion ergänzt)

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

1. **EEG-Einstellungen** (12 Felder auf `registration_entrypoint`):
   - `intro_text`, `show_central_policy`, `require_email_confirmation`
   - SEPA: `sepa_mandate_enabled`, `use_company_sepa_mandate`, `sepa_mandate_at_import`
   - Zählpunkt-Prefixes: `metering_point_prefix_consumption`, `metering_point_prefix_production`
   - `activation_mode` (PROJ-53)
   - Genossenschaftsanteile (PROJ-37, 3 Felder): `cooperative_shares_enabled`,
     `cooperative_required_shares`, `cooperative_share_amount_cents`
2. **Mitgliedsfeld-Konfig** (`field_config`-Tabelle): alle Einträge der
   Quell-EEG (inkl. `participation_factor` — der ist ein field_config-
   Eintrag, kein registration_entrypoint-Feld)
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

- [ ] **AC-E1**: Es gibt eine dedizierte Sub-Seite
  `/admin/settings/import-export` (Sidebar-Eintrag in den
  Admin-Settings). Dort liegen alle Export-Buttons + der Import-
  Upload + (zukünftig erweiterbar: Audit-Liste).
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
- [ ] **AC-I2**: Das System validiert das File **strict**:
  - `schemaVersion` muss EXAKT `1` sein — V2/V0 wird mit „Member-
    Onboarding-Version inkompatibel" abgelehnt; keine Forward-
    Toleranz für unbekannte schemaVersion
  - JSON-Struktur ist syntaktisch korrekt
  - Jede Sektion entspricht dem erwarteten Sub-Typ-Schema
  - Bei Fehlern wird der Import abgelehnt mit einer konkreten
    Fehlermeldung
- [ ] **AC-I3**: Wenn das File mehrere Sektionen enthält, kann Admin
  per Checkbox auswählen, welche er importieren will. **Default: alle
  Sektionen UNAUSGEWÄHLT** — Admin muss aktiv pro Sektion ankreuzen.
  Friction-Schutz, weil es keinen Server-side-Rollback gibt.
- [ ] **AC-I4**: Vor dem Apply wird ein **Diff-Preview** angezeigt:
  pro ausgewählter Sektion eine Tabelle mit „aktueller Wert auf
  Ziel-EEG" → „neuer Wert aus Import". Bei Listen-Sub-Typen
  (legal_document, data_export_config, field_config) wird gezeigt:
  N Einträge auf Ziel-EEG aktuell → M Einträge nach Apply.
- [ ] **AC-I4b**: Sektionen, die im File **leer** sind (z. B. 0
  field_config-Einträge), werden als „lösche alle X bestehenden
  Einträge" mit ROTER WARNUNG dargestellt — Admin sieht: „47
  field_config-Einträge → 0 Einträge nach Apply" hervorgehoben.
  Bestätigt explizit den intended-Replace.
- [ ] **AC-I4c**: Bei den zwei netzbetreiber-spezifischen Feldern
  `metering_point_prefix_consumption` und `_production` wird im
  Diff zusätzlich ein Warn-Icon mit Tooltip „Netzbetreiber-
  spezifisch — prüfen ob auf Ziel-EEG gültig" angezeigt. Verhindert
  versehentliches Übertragen eines Prefixes aus dem Netzgebiet von
  EEG-A auf eine EEG-B in einem anderen Netzgebiet.
- [ ] **AC-I4d**: Bei den drei Cooperative-Shares-Feldern (geld-
  relevant) wird der Betrag-Wert in EUR formatiert angezeigt
  („€ 100,00" statt „10000 Cents") — Admin sieht klar, was er
  überschreibt.
- [ ] **AC-I5**: Admin bestätigt zweistufig: erst Diff sehen +
  Sektionen ankreuzen, dann modaler „Wirklich apply?"-Dialog mit
  „Apply" / „Abbrechen". Erst nach Bestätigung wird die Mutation
  ausgeführt.
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
  geänderter Einträge pro Sektion. Audit-Trail:
  **`slog.Info`-Eintrag** mit Feldern `event=config_import`,
  `rc_number`, `admin_user_id`, `sections=[...]`,
  `source_eeg=<rcNumber>` (aus Header). **Kein DB-Audit-Log** in V1
  (bewusste Owner-Entscheidung — Pre-State-Backup ist Admin-
  Verantwortung).
- [ ] **AC-I9**: Plugin-Registry-Drift: enthält ein Import einen
  `data_export_config` mit `plugin_type`, der auf der Ziel-Instanz
  nicht registriert ist, wird der Eintrag mit `is_obsolete = TRUE`
  angelegt (analog zum laufenden Sweep aus PROJ-60); kein
  Import-Failure. Im Diff-Preview deutlich als Warnung markiert.
- [ ] **AC-I10**: Field-Catalog-Drift: enthält ein
  `data_export_config` Column-Mappings auf Field-Keys, die im
  aktuellen Katalog nicht existieren (z. B. weil Field zwischen
  Quell- und Ziel-Deployment entfernt wurde), werden diese
  Column-Einträge beim Import **verworfen** mit Warn-Hinweis im
  Diff-Preview — kein Import-Failure.
- [ ] **AC-I11**: **Re-Sanitisierung am Eingang**: jedes importierte
  Feld läuft durch dieselbe Validation-/Sanitisierungs-Pipeline wie
  beim regulären Speichern via Admin-UI:
  - `intro_text` → bluemonday (XSS-Schutz)
  - `legal_document.url` → Format-Check: `https://`-Schema,
    keine `javascript:`/`data:`-Schemes, max 2 KB Länge
  - `field_config.name` → gegen `CONFIGURABLE_FIELDS`-Master-Katalog
  - `field_config.state` → ENUM-Check (`hidden`/`optional`/`required`/`admin_only`)
  - `metering_point_prefix_*` → DB-CHECK-Constraint-Format
    (`^AT[0-9A-Z]{0,31}$`)
  - `data_export_config.column_config` → Plugin's bestehender
    `ValidateConfig`
  - `activation_mode` → ENUM-Check
  - Cooperative-Shares → Constraint-Check (positive Werte,
    Required-Shares > 0 wenn Enabled)
  Garantie: ein Import kann **nichts** speichern, was nicht auch via
  UI ginge.
- [ ] **AC-I12**: **Per-Sektion-Item-Limits** verhindern Resource-
  Exhaustion:
  - `field_config`: max 100 Einträge
  - `legal_document`: max 50 Einträge
  - `data_export_config`: max 50 Einträge
  Überschreitung → 400 mit konkretem Limit-Hinweis, kein Apply.
- [ ] **AC-I13**: **Apply-Fehler-UX**: bei Apply-Fail (z. B.
  DB-Constraint, Sanitisierung-Reject) wird ein kategorisierter
  Fehler im Frontend angezeigt:
  „Apply fehlgeschlagen in Sektion `<name>`: `<human-readable Grund>`.
  Bitte File prüfen, Apply wurde komplett zurückgerollt." Roher
  DB-Error landet nur im Backend-slog, nicht im Response.
- [ ] **AC-I14**: **Concurrent-Lock**: pro `rc_number` läuft maximal
  ein Import gleichzeitig — durchgesetzt via
  `pg_advisory_xact_lock(hashtext(rc_number))` zu Beginn der Apply-
  Transaktion. Bei laufendem Import: zweiter Apply blockiert bis 10 s,
  dann 409 mit „EEG wird gerade konfiguriert, bitte später erneut".

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
- **File größer als sinnvolles Limit**: harte Grenze 1 MB → 413.
  Zusätzlich Per-Sektion-Item-Limits (siehe AC-I12) als zweite
  Verteidigungslinie.
- **File enthält bösartiges JSON** (z. B. `intro_text` mit
  `<script>`-Tag, `legal_document.url` mit `javascript:alert(1)`):
  Re-Sanitisierung (AC-I11) fängt es ab; bei Reject während Apply
  → Tx-Rollback + kategorisierter Fehler.
- **Doppel-Klick auf Apply-Button**: idempotent durch Submit-Spinner +
  Server-side Concurrency-Check (PROJ-60 hat das schon für
  config-update, Pattern wiederverwenden).
- **Import-File hat Sektion, die V1 nicht kennt** (z. B. zukünftiges
  `notification_settings`): Sektion wird im Diff als „nicht
  unterstützt — wird ignoriert" angezeigt, kein Import-Failure;
  Forward-Compat-Verhalten.
- **Quell-EEG hat 0 Einträge in einer Sektion** (z. B. keine
  Datenweiterleitungs-Configs): siehe AC-I4b — explizites „lösche
  alles" mit ROTER Warnung im Diff.
- **Cross-Schema-Version-Import**: V2 wird abgelehnt mit Hinweis
  „bitte aktuelles Member-Onboarding nutzen". V1-File auf V2-System:
  V2 muss `schemaVersion: 1`-Files konvertieren oder ablehnen — die
  V2-Spec wird das festlegen, V1 hat keine Forward-Compat-Garantie
  zu zukünftigen Versionen.
- **Concurrent Import**: zweiter Apply blockiert via Advisory-Lock
  (AC-I14); zweiter Admin sieht nach 10 s Timeout eine 409. Anderer
  Edit-Pfad (UI-Save eines einzelnen Settings parallel zum Import)
  ist nicht durch Lock geschützt — Last-Write-Wins, akzeptable
  Vereinfachung weil regulärer UI-Save atomarer ist.
- **Großer Diff-Preview-Output**: bei 50+ field_config-Einträgen wird
  die Diff-Tabelle lang. Pro Sub-Typ ein eigener Collapse-Block in der
  UI; default geöffnet bei Sektionen mit < 5 Änderungen, sonst
  collapsed.
- **Import in produktiv-aktive EEG**: keine Sperre. Field-Config-
  Änderung wirkt sofort auf laufende Public-Form-Sessions (Caching ist
  per-Request, kein Cache-Invalidate nötig). Data-Export-Config-
  Änderung beeinflusst nur künftige Jobs (laufende Jobs nutzen
  `config_snapshot`).
- **Rollback nach Apply**: V1 hat **keinen automatischen Pre-State-
  Backup**. Admin-Workflow für Rollback:
  1. VOR dem Import: bewusst Export der aktuellen Konfig der Ziel-EEG
     machen + lokal sichern
  2. Falls Apply rückgängig zu machen: gespeicherte Datei erneut
     importieren
  Dieser Workflow muss in der UI als Hinweis-Text sichtbar sein
  („Tipp: Sichere deine aktuelle Konfig vor dem Import, falls du
  zurückrollen willst").

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

## Grilling-Ergebnisse (2026-05-24)

20 Designentscheidungen in 5 Runden geklärt. Kompakt:

### Audit & Rollback (Minimal-Linie)

- **Audit**: nur `slog.Info`-Eintrag bei Apply (siehe AC-I8); keine
  DB-Persistenz, keine Audit-UI.
- **Pre-State-Backup**: NICHT automatisch. Admin-Verantwortung; UI
  zeigt Hinweis-Text.
- **Rollback**: kein dedizierter UI-Knopf. Workflow = vorher exportieren
  + bei Bedarf erneut importieren.
- **Konsequenz**: Apply ist faktisch irreversibel. Pre-Apply-UX
  (Diff + Confirm) trägt das Gewicht.

### Replace-Semantik

- Diff-Preview-Default: alle Sektionen **UNAUSGEWÄHLT** — Admin muss
  bewusst pro Sektion ankreuzen.
- Leere Sektion = explizites „lösche alles" mit ROTER Warnung.
- `data_export_config`-Replace: Soft-Delete der alten Einträge
  (PROJ-60-Pattern), neue Einträge mit neuen IDs.
- Confirm-UX: zweistufig (Datei → Diff → Bestätigungs-Modal → Apply).

### Security beim Import

- **Re-Sanitisierung**: alle Felder durchs Backend-Defense-Mesh
  (bluemonday für intro_text, URL-Format-Check für legal_document,
  ENUM-Check, Plugin-`ValidateConfig`, etc.) — siehe AC-I11.
- **Per-Sektion-Item-Limits**: 100 / 50 / 50 (siehe AC-I12).
- **Plugin-Type-Drift**: Unknown plugin_type → `is_obsolete=true`-
  Import mit Diff-Warnung.
- **Schema-Version-Strenge**: nur exakt `schemaVersion: 1`; V2/V0
  abgelehnt.
- **URL-Validation**: nur Format-Check, KEIN HEAD-Request (SSRF-Vektor
  vermeiden).

### UX / Edge Cases

- UI-Plazierung: eigene Sub-Seite `/admin/settings/import-export`.
- Concurrent-Lock: `pg_advisory_xact_lock(hashtext(rc_number))` mit
  10 s Timeout → 409.
- Zählpunkt-Prefix-Diff: zusätzliches Warn-Icon „Netzbetreiber-
  spezifisch".
- Apply-Fail-Frontend: kategorisierter Fehler + Sektion-Hinweis;
  roher DB-Error nur im Backend-slog.

### Field-Scope-Korrekturen

- **EEG-Einstellungen-Sub-Typ** ergänzt um drei Cooperative-Shares-
  Felder (PROJ-37).
- `participation_factor` aus EEG-Einstellungen entfernt — ist
  field_config-Eintrag, gehört in dessen Sub-Typ.
- **EEG-Stammdaten** (eeg_name, Adresse, eeg_id, contact_email,
  creditor_id), **Sequence-State** (member_number_start), **Identity**
  (rc_number), **Sync-State** (last_synced_*) und **Secrets**
  (external_api_key) bleiben außerhalb.
- **Export-Header** `exportedFrom: { rcNumber, eegName }` bleibt drin
  (kein Info-Leak — Tenant-Admin-sichtbar).

## Recommended Next Step

`/architecture` — Designentscheidungen sind durch, Tech-Design kann
die Datenmodelle, Endpoint-Signaturen, Service-Layer-Aufteilung,
Transaktions-Boundaries und das pg_advisory_xact_lock-Setup
ausarbeiten.

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)

**Stand:** 2026-05-24

### Leitidee: Assembly bestehender Bausteine

Das Feature führt **keine neue Tabelle, keine Migration und keine neuen
Drittabhängigkeiten** ein. Alle vier Sub-Typen haben bereits
Repositories mit den nötigen Save-Primitiven, alle Validation-Pipelines
existieren bereits an anderer Stelle. Das Tech-Design ist primär
**Orchestrierung**: ein Exporter, der vier Repos liest, ein Importer,
der vier Repos schreibt — beide eingerahmt durch JSON-Serialisierung,
Validation, Diff-Generation und Transaktions-Schutz.

### A) Komponenten-Struktur

#### Frontend

```
/admin/settings/import-export                   (neue Sub-Seite)
+-- Export-Card
|   +-- 4 Per-Sub-Typ-Buttons (Settings | Field-Config | Legal-Docs | Data-Export)
|   +-- 1 Bundle-Button (alle 4)
|   +-- Tipp-Box "Sichere deine Konfig vor jedem Import"
+-- Import-Card
|   +-- File-Drop-Zone (akzeptiert .json, max 1 MB, client-side-Vor-Validierung)
|   +-- Sektion-Auswahl (nach erfolgreichem Preview)
|       +-- Checkboxen, alle UNAUSGEWÄHLT default
|       +-- Pro Sektion: Diff-Tabelle alt → neu, Counts, Warn-Badges
|       +-- Spezial-Warnungen: ROTE Box bei "lösche alle X Einträge",
|           Warn-Icon bei Zählpunkt-Prefix, € bei Cooperative-Shares
|   +-- Apply-Button → Bestätigungs-Modal "Wirklich apply?" → Apply
+-- Erfolgs-/Fehler-Toast nach Abschluss
```

Neue React-Komponenten unter `src/components/config-import-export/`:

- `ExportButtons` — die fünf Export-Buttons + Download-Trigger
- `ImportDropzone` — File-Upload + Preview-Request
- `DiffPreviewPanel` — Sektion-Auswahl + Diff-Tabellen + Apply-Trigger
- `DiffTable` — generische Tabelle alt|neu mit Highlighting
- `ConfirmApplyDialog` — zweistufige Bestätigung

Wiederverwendet: `shadcn/ui` Dialog, Card, Checkbox, Button, Alert;
`sonner` für Toasts.

#### Backend

```
internal/configexport/                          (neues Paket)
+-- schema.go        — versionierte JSON-Strukturen pro Sub-Typ
+-- exporter.go      — assembliert Snapshot aus 4 Repos
+-- importer.go      — validate + diff + apply (Tx-orchestriert)
+-- diff.go          — generische Diff-Engine pro Sub-Typ
+-- limits.go        — Item-Limits-Konstanten (100/50/50)

internal/http/configexport.go                   (neuer Handler)
+-- 3 Endpoints (siehe API-Vertrag unten)
```

Bestehende Repos werden **read-only über Schnittstelle**
weiterverwendet:

- `RegistrationEntrypointRepository.GetByRCNumber` + `SaveEEGSettings`
- `FieldConfigRepository.Get` + `Save` (atomarer Map-Replace)
- `LegalDocumentRepository.GetByRCNumber` + `Create`/`Delete`/`Reorder`
- `dataexport.ConfigRepository.ListByRCNumber` + `Create` +
  `SoftDelete` + `MarkObsolete`

Keine Erweiterung bestehender Repos nötig — die vorhandenen
Save-Primitive decken alle vier Replace-Varianten ab.

### B) Datenmodell

**Persistenz: keine.** Es entsteht keine neue DB-Tabelle. Audit
geschieht ausschließlich via strukturiertem `slog`-Logging
(`event=config_import`, `rc_number`, `admin_user_id`, `sections`,
`source_eeg`, `applied_at`).

**Datei-Format: versioniertes JSON.**

Hülle:

```
{
  schemaVersion:  1                    // strict — V0/V2 abgelehnt
  exportedAt:     ISO-8601-Timestamp
  exportedFrom:   { rcNumber, eegName } // Wiedererkennungs-Header
  sections: {
    eegSettings?:        { ... 12 Felder ... }
    fieldConfig?:        [ { name, state, adminValue? }, ... ]
    legalDocuments?:     [ { title, url, required, sortOrder }, ... ]
    dataExportConfigs?:  [ { name, pluginType, configJSON }, ... ]
  }
}
```

Sub-Type-Schemas spiegeln **bewusst nur die DB-Spalten, die
exportiert werden** (siehe Scope V1). Identitäts-/Stammdaten-/Secret-/
Sequence-Felder fehlen schon im Schema — Scrubbing geschieht
deklarativ, nicht prozedural.

### C) API-Vertrag (3 Endpoints)

Alle drei Endpoints leben unter `/api/admin/config/*`, sind durch die
existierende Keycloak-Auth-Middleware geschützt, prüfen Tenant-
Zugriff via `checkTenantAccess(rcNumber)`.

| Endpoint | Methode | Zweck |
|---|---|---|
| `/api/admin/config/export` | `GET` | Liefert das JSON-File. Query-Parameter: `rcNumber`, `sections` (Komma-Liste oder `bundle`). Response-Header: `Content-Disposition` mit dem in AC-E5 spezifizierten Dateinamen. |
| `/api/admin/config/import/preview` | `POST` | Multipart-Upload des JSON-Files + Form-Feld `rcNumber`. Server: schema-validate, sanitize, limit-check, diff gegen aktuelle DB-Werte. Response: strukturierter Diff pro Sektion + Validierungs-/Warn-Hinweise. **Keine Mutation.** |
| `/api/admin/config/import/apply` | `POST` | Body: JSON-File-Inhalt + `sectionsToApply: ["eegSettings", ...]`. Server: re-validiert (zero-state), re-diff (für Audit), wendet in einer einzigen Transaktion + Advisory-Lock an. Response: Counts pro Sektion oder kategorisierter Fehler. |

**Bewusst gewählt: kein Preview-Token-State.** Der `apply`-Endpoint
re-validiert das File. Das kostet ~100 ms doppelte Validation, spart
aber ein In-Memory-Cache oder eine neue Tabelle für Preview-Sessions.
Sub-Sekunde, akzeptable Vereinfachung. Konsequenz: der Apply-Body
muss das vollständige File mitschicken (nicht nur eine Preview-ID).

### D) Transaktions-Modell

Alle vier Sektionen in einer einzigen DB-Transaktion:

1. Beginnen mit `pg_advisory_xact_lock(hashtext(rc_number))` —
   serialisiert konkurrierende Imports pro EEG. `lock_timeout` 10 s,
   sonst 409.
2. Pro ausgewählter Sektion: Aktuellen Stand lesen (für Re-Diff im
   Audit-Log) → Save-Primitive ausführen (siehe Mapping unten).
3. Commit. Bei Fehler in irgendeiner Sektion: Rollback der gesamten
   Transaktion → kategorisierter Fehler ans Frontend.

**Sektion-zu-Save-Mapping:**

| Sektion | Apply-Strategie |
|---|---|
| `eegSettings` | `RegistrationEntrypointRepository.SaveEEGSettings` — atomarer Update der 12 Spalten. Nicht im File enthaltene Felder bleiben unverändert (Forward-Compat-Garantie). |
| `fieldConfig` | `FieldConfigRepository.Save(rcNumber, fullMap)` — bestehender atomarer Replace (interne `DELETE + INSERT` in derselben Tx). |
| `legalDocuments` | Bestehende per-EEG `Delete-all` + `Create`-Loop in der Apply-Tx. Reorder folgt aus `sortOrder`-Feld. |
| `dataExportConfigs` | Pro existierender Eintrag: `SoftDelete`. Pro File-Eintrag: `Create` mit neuer ID. Plugin-Drift via `MarkObsolete` nach dem Insert-Loop. |

### E) Validation-Pipeline beim Import

Beim `preview` UND beim `apply` läuft jedes Feld durch dieselbe
Pipeline, die das reguläre UI-Save schon nutzt — Wiederverwendung
statt Parallelpfad:

1. **Schema-Check**: `schemaVersion == 1`, JSON-Struktur stimmt mit
   erwarteten Sub-Type-Schemas überein, Pflichtfelder vorhanden.
2. **Per-Sektion-Limit-Check**: `len(fieldConfig) ≤ 100`,
   `legalDocuments ≤ 50`, `dataExportConfigs ≤ 50`.
3. **Sanitize/Validate** pro Feld (siehe AC-I11 in Spec):
   - `intro_text` → `bluemonday`-Renderer
   - `legal_document.url` → Format-Check (HTTPS-only, Längen-Limit)
   - `field_config.name` → gegen `CONFIGURABLE_FIELDS`-Master-Katalog
   - `field_config.state` → ENUM-Check
   - `metering_point_prefix_*` → DB-CHECK-Constraint-Format
   - `activation_mode` → ENUM-Check
   - `cooperative_*` → Positiv-Konstraint
   - `data_export_config.column_config` → Plugin's `ValidateConfig`
4. **Drift-Check**:
   - `data_export_config.plugin_type` nicht registriert → `is_obsolete=true` markieren, **Warnung in Response**, kein Reject
   - `data_export_config.column_config` referenziert unbekannte
     Field-Keys → Spalten verwerfen, **Warnung**, kein Reject

### F) Diff-Generation

Pro Sektion separat. Output ist ein JSON-Strukturobjekt, das das
Frontend direkt rendert (keine HTML-Generation im Backend):

- **eegSettings**: 12 Einträge mit `field`, `oldValue`, `newValue`,
  optional `warningType` (z. B. `network_region_specific` für die
  beiden ZP-Prefix-Felder, `financial` für die Cooperative-Felder).
- **fieldConfig / legalDocuments / dataExportConfigs**: Pro Eintrag
  klassifiziert als `unchanged`, `modified`, `added`, `removed`.
  Plus Totals: `currentCount`, `afterCount`. Bei `afterCount == 0
  && currentCount > 0`: zusätzlich `wholeSectionDeletion: true` für
  die ROTE Warnung im Frontend.

### G) Tenant-Isolation + Permissions

- Export: `checkTenantAccess(rcNumber)` ODER Superuser. Liefert nur
  Daten der angefragten EEG.
- Import-Preview / -Apply: gleicher Check. Quell-EEG aus
  `exportedFrom`-Header wird **nicht** geprüft — Admin darf ein File
  importieren, das von einer anderen EEG (oder einem Kollegen)
  stammt; der Tenant-Check gilt nur für die **Ziel-EEG**.

### H) Fehlerbehandlung

- **Schema-Fehler** (kein JSON, falsche `schemaVersion`, fehlende
  Pflichtfelder) → 400 mit konkretem Hinweis, kein Apply.
- **Validation-Fehler** in einer Sektion → 400 mit Sektion + Feld;
  bei Multi-Sektion-Bundle wird die GANZE Apply abgelehnt (atomar).
- **Apply-Tx-Fehler** (z. B. DB-Constraint nach Sanitize) → 500 mit
  kategorisierter Meldung „Apply in Sektion <X> fehlgeschlagen,
  rollbacked"; roher DB-Error nur im `slog`.
- **Advisory-Lock-Timeout** → 409 „EEG wird gerade konfiguriert".
- **File zu groß** → 413 (durch existierende
  `MaxBodySize`-Middleware, hier auf 1 MB konfiguriert).

### I) Performance-Überlegung

- Export: 1 Query pro Sub-Typ-Repo, alle parallel-isierbar; bei
  realistischer Datenmenge (max ~200 Einträge gesamt) deutlich
  unter 500 ms.
- Preview: 1× Validate + 4× Repo-Read + 4× Diff = sub-second.
- Apply: 4× Repo-Schreibe in einem Tx + Advisory-Lock-Setup ≈
  300-800 ms im Median.

Item-Limits (100/50/50) garantieren, dass die O(n)-Operationen
nicht entartet werden.

### J) Was die Tech-Design-Entscheidung NICHT macht

- Keine neue DB-Tabelle für Audit (bewusste Owner-Entscheidung).
- Keine Migration nötig.
- Keine neuen Drittabhängigkeiten (kein neuer Validator-Library,
  alles via existierende Imports).
- Kein Cleanup-CronJob für Soft-Deleted `data_export_config` —
  bestehender PROJ-60-Cleanup-Job (siehe `cmd/cleanup`/CronJob in
  Helm) deckt die Akkumulation automatisch ab.
- Kein Preview-Token / kein Server-State zwischen Preview und Apply.

### K) Test-Strategie (Hinweise für /qa)

- **Backend-Unit**: Validation-Pipeline pro Sub-Typ (Happy + Reject),
  Diff-Engine (alle 4 Kategorien: unchanged/modified/added/removed),
  Schema-Version-Strenge, Limit-Checks.
- **Backend-Integration**: vollständiger Roundtrip Export → Preview
  → Apply gegen Test-DB; Concurrent-Apply mit zwei Goroutines (zweiter
  bekommt 409 nach 10 s); Plugin-Drift mit unbekanntem `pluginType`.
- **Frontend-Vitest**: Diff-Tabelle-Rendering, Sektion-Checkbox-
  Default-State, Bestätigungs-Dialog.
- **E2E (Playwright)**: Export-Bundle → File-Download in Browser →
  File-Upload → Preview-Render → Apply mit Section-Check → Toast.

### L) Dependencies

Keine neuen Pakete.

Frontend: bereits installiert — `shadcn/ui`, `sonner`, `next/dynamic`
(für Drop-Zone), File-Reader-API ist nativ.

Backend: bereits importiert — `bluemonday` für HTML-Sanitize,
`encoding/json`, `database/sql`, `pq` für Advisory-Lock.

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
