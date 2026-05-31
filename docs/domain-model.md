# Domain Model
## eegfaktura Member Onboarding

## 1. Goal

The data model for `eegfaktura Member Onboarding` is deliberately kept simple and uses as few tables as possible.

It supports:
- self-registration of new members
- multiple metering points per application
- admin review and approval
- traceable status history
- later import into eegFaktura

Not part of the model:
- tariffs
- role management
- JSON fields

## 2. Schema

All tables reside in the PostgreSQL schema:

- `member_onboarding`

## 3. Tables

### 3.1 `member_onboarding.registration_entrypoint`

Local mapping table between RC number and EEG.

Purpose:
- resolving the publicly used RC number to the internal EEG identifier
- no direct access to eegFaktura core tables at runtime
- controls whether a registration is active

Fields:
- `id`
- `rc_number`
- `is_active` — boolean, default false; controls whether the public registration form is active; must be explicitly enabled by the admin via the settings page
- `intro_text` — nullable, sanitized HTML string for the public registration form
- `sepa_mandate_enabled` — boolean, default false; controls whether SEPA mandate PDF is attached to welcome email
- `use_company_sepa_mandate` — boolean, default false; opts the EEG in to the SEPA B2B (Firmenlastschrift) mandate variant. The per-application mandate type lives in `application.einzugsart` (`core` | `b2b` | `kein_sepa`) and is set by the admin, not auto-derived from `member_type` (PROJ-48 removed the previous auto-mapping). Only evaluated when `sepa_mandate_enabled = true`.
- `sepa_mandate_at_import` *(PROJ-48)* — boolean, default false; when true, SEPA mandate PDFs are generated at import time (with assigned `member_number` printed as Mandatsreferenz) instead of submit time. Use for digital-signature workflows where the signed PDF cannot be modified afterwards.
- `show_central_policy` — boolean, default true; when false, the central operator privacy policy is not shown in the public registration form (for EEGs that configure their own policy as a legal document)
- `member_number_start` — INT NOT NULL DEFAULT 1; starting value for the per-EEG member number auto-increment counter; the first member number assigned for this EEG will be this value
- `require_email_confirmation` — boolean, default false (PROJ-31); when true, members must click the link in the confirmation mail before the application becomes reviewable; admin `/status` endpoint rejects `submitted → under_review|needs_info|approved` with 409 until confirmed
- `metering_point_prefix_consumption` *(PROJ-52)* — VARCHAR(33) NULL; pro-EEG konfigurierbarer Zählpunkt-Prefix für CONSUMPTION-Anschlüsse. DB-CHECK `^AT[0-9A-Z]{0,31}$`. Service-Layer normalisiert vor dem Save (Whitespace + Dots + Hyphens entfernen, uppercase). NULL ⇒ heutiges Verhalten (nur „AT" ist fix). Submit-Validation prüft Match (`HasPrefix`) für jeden CONSUMPTION-Zählpunkt.
- `metering_point_prefix_production` *(PROJ-52)* — analog für PRODUCTION-Anschlüsse. Fällt zurück auf reines AT-Pattern wenn nur eine Richtung konfiguriert ist (Fallback-Regel 2a).
- `activation_mode` *(PROJ-53)* — VARCHAR(40) NOT NULL DEFAULT `participant_active`. Steuert das Kriterium des Activation-Check-Batchs für den Wechsel `ready_for_activation → activated`. Werte: `participant_active` (Core-Teilnehmer-Status `ACTIVE` — Default, rückwärtskompatibel) oder `any_meter_registration_started` (min. ein Zählpunkt im Core mit `processState ∈ {PENDING, APPROVED, ACTIVE}` — d.h. der Netzbetreiber hat auf die EDA-Meldung geantwortet). DB-CHECK auf das Enum.
- `settings_view_mode` *(PROJ-67)* — VARCHAR(40) NOT NULL. Per-EEG Sichtbarkeits-Modus der Admin-Settings-Page. Werte: `standard` (Einfache Ansicht — Default für neu angelegte EEGs, reduzierte Sicht) oder `advanced` (Alle Optionen — Default für vor PROJ-67 angelegte EEGs via Migration 000059, volle Sicht). DB-CHECK auf das Enum. Heute reine UI-Pref, kein Backend-Enforcement; spätere Lizenz-PROJ kann den Wert aus einer Subscription-Tabelle synchronisieren.
- `cooperative_shares_enabled` *(PROJ-37)* — boolean, default false; aktiviert die Genossenschaftsanteile-Erfassung im Mitgliederformular. Wenn TRUE, müssen die beiden folgenden Felder gesetzt sein.
- `cooperative_required_shares` *(PROJ-37)* — INT NULL, CHECK `> 0`; Pflichtanteils-Mindestmaß pro Mitglied. NULL wenn Feature deaktiviert.
- `cooperative_share_amount_cents` *(PROJ-37)* — BIGINT NULL, CHECK `> 0`; Preis pro Anteil in Cent. NULL wenn Feature deaktiviert. Speicherung als Integer-Cents vermeidet Float-Drift.
- `created_at`
- `updated_at`

**Core-mastered fields (PROJ-32 — synced from eegFaktura, read-only in the admin UI):**
- `eeg_id` — Gemeinschafts-ID; used as the Excel-Export Spalte B value and for the eegFaktura import. Source: GraphQL `eeg.communityId`.
- `eeg_name` — official name of the energy community. Source: `eeg.description` (descriptive long name, e.g. "Testenergiegemeinschaft EEG 1234"). Note: the core's `eeg.name` is a short internal handle equivalent to `rcNumber` (e.g. "EEG-TEST") and is not used here.
- `eeg_street`, `eeg_street_number`, `eeg_zip`, `eeg_city` — EEG address. Source: `eeg.address.{street, streetNumber, zip, city}`.
- `creditor_id` — SEPA creditor ID (max 35 chars). Source: `eeg.accountInfo.creditorId`.
- `contact_email` — EEG notification recipient (admin-Benachrichtigung bei neuem Antrag). Source: `eeg.contact.email`.
- `last_synced_from_core_at` — nullable TIMESTAMPTZ; stamped on every successful master-data sync; NULL until the first sync after PROJ-32 deploy.
- `eeg_logo_bytes` *(PROJ-33)* — nullable BYTEA, max 256 KB; PNG/JPEG/GIF bytes of the EEG logo pulled from `eegfaktura-billing` (`/cash/api/billingConfigs/{id}/logoImage`). Embedded top-right in approval + SEPA mandate PDFs.
- `eeg_logo_mime` *(PROJ-33)* — nullable TEXT; one of `image/png`, `image/jpeg`, `image/gif`. NULL ⇒ no logo.
- `eeg_logo_synced_at` *(PROJ-33)* — nullable TIMESTAMPTZ; stamped on every successful logo sync. Separate from `last_synced_from_core_at` because the logo sync is best-effort: master-data can sync successfully while the logo step skips or fails.

These ten values are written exclusively by the sync endpoint (`POST /api/admin/settings/eeg/sync`) which forwards the admin's Keycloak JWT to the eegFaktura core. The legacy `PUT /api/admin/settings/eeg` no longer accepts them in the request body. See `features/PROJ-32-eeg-master-data-from-core.md` and `features/PROJ-33-eeg-logo-from-core.md`.

Rules:
- `rc_number` is unique
- only entries with `is_active = true` allow a registration
- maintenance is performed by admins or through deployment configuration

---

### 3.1a `member_onboarding.document_consent` (PROJ-36 note)

Audit-Snapshot pro Antrag + Rechtsdokument. Spalten:
- `id`, `application_id`, `title`, `url`, `is_central_policy`, `consented_at` (unverändert seit PROJ-9)
- `consent_type` (PROJ-36) — `explicit` wenn das Mitglied aktiv eine Checkbox geklickt hat, `informational` wenn das Dokument als Info-Link angezeigt wurde (kein Häkchen, Kenntnisnahme implizit durch Antrags-Submit). Default `explicit` für Bestandseinträge.

Eindeutigkeitsregeln: keine — eine Application kann mehrere Consents für unterschiedliche Dokumente haben.

---

### 3.1b `member_onboarding.field_config`

Per-EEG configuration of optional form fields. Only explicitly configured values are stored (sparse table); missing entries fall back to system defaults.

Fields:
- `id`
- `rc_number` — references `registration_entrypoint(rc_number)`, ON DELETE CASCADE
- `field_name` — name of the configurable field (e.g. `heat_pump`, `transformer`)
- `state` — `hidden` | `optional` | `required` | `admin_only`
  - `admin_only` means: hidden from public registration form, but visible and editable in the admin per-application edit dialog. PROJ-68 (migration 000058) dropped the `admin_value` default-value column that used to live here.
- `updated_at`

Rules:
- `(rc_number, field_name)` is unique
- `field_name` must be one of the centrally registered configurable fields (enforced in application code)
- `state` is constrained to `hidden`, `optional`, `required`, `admin_only` (DB CHECK constraint)
- missing entries default to `hidden` for new fields; `optional` for `phone`, `birth_date`, `uid_number`, `bank_name`, `participation_factor`
- field_config entries used since PROJ-56/57/58 (alle Default `hidden`): `network_operator_customer_number`, `meter_inventory_number`, `contact_person_name`, `contact_person_email`, `contact_person_phone`, `billing_email`. Es gibt keinen Master-Switch `contact_person` mehr (PROJ-57 v3 hat ihn entfernt — die Public-Form-Checkbox „Ansprechperson angeben" erscheint automatisch, sobald mindestens eines der drei Subfelder ≠ `hidden` ist).
- `admin_only` fields are returned as `hidden` in the public registration config — members never see them

---

### 3.2 `member_onboarding.application`

Central main table for an onboarding application.

Contains:
- identification
- EEG assignment (via `rc_number`)
- status
- person data
- contact data
- address data
- consents
- SEPA / bank data
- admin note
- import status

Fields:
- `id`
- `reference_number` — Format **`<RC>-<Jahr>-<NNNN>`** seit PROJ-35 (z.B. `RC105720-2026-0001`), 4-stelliger Counter pro EEG und Jahr. Anträge die vor PROJ-35 erstellt wurden behalten ihr altes Format `MO-YYYY-NNNNNN`. Eindeutigkeit über `application.reference_number`-UNIQUE-Constraint garantiert.
- `rc_number`
- `status`
- `started_at`
- `submitted_at`
- `approved_at`
- `rejected_at`
- `imported_at`
- `member_type`
- `titel` — nullable VARCHAR(50), optional title prefix (e.g. "Mag.", "Dr."); im UI als „Titel vor" gelabelt
- `titel_nach` *(Migration 000037)* — nullable VARCHAR(50), optional title suffix (e.g. "BSc", "MSc", "MBA"); im UI als „Titel nach"
- `firstname`
- `lastname`
- `birth_date`
- `company_name`
- `uid_number`
- `register_number`
- `email`
- `phone`
- `resident_street`
- `resident_street_number`
- `resident_zip`
- `resident_city`
- `privacy_accepted`
- `privacy_version`
- `privacy_accepted_at`
- `accuracy_confirmed`
- `iban`
- `account_holder`
- `sepa_mandate_accepted`
- `sepa_mandate_accepted_at`
- `einzugsart` — VARCHAR(20) NOT NULL DEFAULT 'core'; SEPA mandate type: `core` = Basislastschrift, `b2b` = Firmenlastschrift, `kein_sepa` = kein SEPA-Mandat (admin trägt Zahlungsdaten manuell im Core nach)
- `bank_name` — nullable; bank name used in SEPA mandate
- `mandate_reference` — nullable; SEPA mandate reference number
- `mandate_date` — nullable DATE; date of SEPA mandate signature
- `reviewed_by_user_id`
- `admin_note`
- `needs_info_reason`
- `target_participant_id`
- `import_started_at`
- `import_finished_at`
- `import_error_message`
- `created_at`
- `updated_at`
- `membership_start_date` *(nullable, configurable)*
- `persons_in_household` *(nullable integer, configurable)*
- `heat_pump` *(nullable boolean, configurable)*
- `electric_vehicle` *(nullable boolean, configurable)*
- `electric_hot_water` *(nullable boolean, configurable)*
- `member_number` — nullable TEXT (since migration 000027); assigned at import time, chosen by the admin in the import dialog (pre-filled with the next free value derived from the core's existing participantNumber pattern, alphanumeric supported, e.g. "A006"). Shown as first data field in the approval PDF.
- `email_confirmation_token_hash` — nullable BYTEA; SHA-256 of the single-use confirmation token (PROJ-31). NULL means no token has been issued. Cleared on confirmation (kept after consumption so a second click can return "already confirmed").
- `email_confirmation_token_expires_at` — nullable TIMESTAMPTZ; token validity window (30 days).
- `bank_confirmed_at` *(PROJ-46)* — nullable TIMESTAMPTZ; stamped when admin transitions `awaiting_bank_confirmation → ready_for_activation` after the member confirms hausbank pre-notification. NULL on the non-b2b auto-skip path.
- `activated_at` *(PROJ-46)* — nullable TIMESTAMPTZ; stamped when admin manually activates OR the activation-check batch finds the member ACTIVE in Core. PROJ-53 extends the trigger: also stamped by the manual `approved → activated` skip-import path (`POST /mark-activated`).
- `activation_notification_sent_at` *(PROJ-53)* — nullable TIMESTAMPTZ; set when the Beitrittsbestätigungs-Mail with PDF was successfully delivered. Guards against double-send when an application transitions in/out of `activated` multiple times. Migration 047 retro-fits the flag for applications that were already in `imported/ready_for_activation/awaiting_bank_confirmation/activated` at deploy time (hard cut-off: no duplicate mail to existing members).
- `network_operator_authorization` *(PROJ-44)* — BOOLEAN NOT NULL DEFAULT FALSE; member-granted authorisation for the EEG to coordinate with the grid operator on their behalf. Per-EEG via `field_config` (default `hidden`).
- `network_operator_authorization_at` *(PROJ-44)* — nullable TIMESTAMPTZ; audit timestamp set on FALSE→TRUE transition.
- `network_operator_customer_number` *(PROJ-56, Migration 000049)* — nullable TEXT; Kundennummer beim Netzbetreiber. Im Public-Formular nur sichtbar, wenn (a) `field_config` ≠ `hidden` UND (b) das Mitglied die Vollmacht-Checkbox aktiv setzt. Service-Layer cleart auf NULL, wenn `network_operator_authorization=FALSE` oder `field_config=hidden`.
- `meter_inventory_number` *(PROJ-56, Migration 000049)* — nullable TEXT; Inventarnummer des Zählers. Gleiche Sichtbarkeits- und Cleanup-Regel wie `network_operator_customer_number`.
- `has_contact_person` *(PROJ-57, Migration 000050)* — BOOLEAN NOT NULL DEFAULT FALSE; expliziter Toggle, damit „leer + nein" und „leer + ja" semantisch unterscheidbar bleiben. Wird im Public-Formular nur bei Org-Mitgliedstypen (`company`, `association`, `municipality`) als Checkbox „Ansprechperson angeben" gerendert; die Checkbox erscheint automatisch, sobald mindestens eines der drei Subfelder im `field_config` ≠ `hidden` ist (kein eigener Master-Switch im `field_config`, siehe PROJ-57 v3).
- `contact_person_name` *(PROJ-57, Migration 000050)* — nullable TEXT; Ansprechperson-Name. Per-EEG via `field_config` (Default `hidden`). Service-Layer cleart auf NULL, wenn `has_contact_person=FALSE`, Mitgliedstyp nicht in der Org-Liste, oder alle drei contact_person-Felder im `field_config` auf `hidden` stehen.
- `contact_person_email` *(PROJ-57, Migration 000050)* — nullable TEXT; gleiche Bedingungen wie `contact_person_name`. E-Mail-Format-Check läuft auch bei `field_config=optional`, falls Wert eingegeben.
- `contact_person_phone` *(PROJ-57, Migration 000050)* — nullable TEXT; gleiche Bedingungen wie `contact_person_name`.
- `has_billing_email` *(PROJ-58, Migration 000051)* — BOOLEAN NOT NULL DEFAULT FALSE; expliziter Toggle für „abweichende Rechnungs-E-Mail". Nur bei Org-Mitgliedstypen, in der Bankverbindungs-Section gerendert.
- `billing_email` *(PROJ-58, Migration 000051)* — nullable TEXT; Rechnungs-E-Mail. Per-EEG via `field_config` (Default `hidden`). Service-Layer cleart auf NULL, wenn `has_billing_email=FALSE`, Mitgliedstyp nicht in der Org-Liste oder `field_config=hidden`. E-Mail-Format-Check bei nicht-leerem Wert.
- `email_confirmed_at` — nullable TIMESTAMPTZ; set when the member clicked the link.
- `email_confirmation_used_at` — nullable TIMESTAMPTZ; first-click timestamp (separate from `email_confirmed_at` to detect re-clicks).
- `cooperative_shares_count` *(PROJ-37)* — INT NULL, CHECK `> 0`; Anzahl der vom Mitglied gezeichneten Genossenschaftsanteile. NULL bei EEGs ohne aktiviertes Anteils-Feature; sonst Submit-validiert `>= registration_entrypoint.cooperative_required_shares`. Gesamtbetrag wird nicht gespeichert — `count × amount` ist Render-Berechnung.

### 3.3 `member_onboarding.metering_point`

**PROJ-45-Spalten** (Erzeugungsform + Batterie pro Zählpunkt):
- `generation_type` VARCHAR(20) NULL — `pv` | `hydro` | `wind` | `biomass`. NULL bei CONSUMPTION, Pflicht (CHECK) bei PRODUCTION. Default `pv` für neue Production-Zählpunkte; Bestandsdaten werden migrationsweise auf `pv` gesetzt.
- `battery_size_kwh` NUMERIC(7,2) NULL — Kapazität des Heimspeichers in kWh. Nur sinnvoll wenn `generation_type='pv'` (Service-Layer cleart sonst); PROJ-8-konfigurierbar (Default `hidden`).
- `inverter_manufacturer` VARCHAR(100) NULL — Freitext-Hersteller (Fronius/SMA/Huawei …). Gleiche Bedingungen wie `battery_size_kwh`.

**PROJ-49-Spalten** (Energie-Felder pro Zählpunkt — Migration 000043 hat sie von der `application`-Tabelle hierher verschoben, Bestandswerte verworfen):
- `consumption_previous_year` BIGINT NULL — Verbrauch Vorjahr in kWh. Nur sinnvoll bei `direction='CONSUMPTION'` (Service-Layer cleart sonst); PROJ-8-konfigurierbar (Default `hidden`).
- `consumption_forecast` BIGINT NULL — Verbrauch Prognose in kWh. Gleiche Bedingungen wie `consumption_previous_year`.
- `feed_in_forecast` BIGINT NULL — Einspeisung Prognose in kWh/Jahr. Nur bei `direction='PRODUCTION'` (alle Erzeugungsformen); Service-Layer cleart sonst.
- `pv_power_kwp` NUMERIC(7,2) NULL — installierte PV-Leistung in kWp. Nur bei `direction='PRODUCTION'` mit `generation_type='pv'`; Service-Layer cleart sonst.
- `feed_in_limit_present` BOOLEAN NULL — „Einspeiselimit vorhanden?" (manche Netzanschlüsse sind leistungstechnisch beschränkt). Nur bei `direction='PRODUCTION'` mit `generation_type='pv'`; Service-Layer cleart sonst.
- `feed_in_limit_kw` NUMERIC(7,2) NULL — maximaler Einspeisewert in kW. Nur gefüllt wenn `feed_in_limit_present = TRUE`; Service-Layer cleart sonst.
- `battery_control_acceptable` BOOLEAN NULL *(Migration 000044)* — Mitglied-Antwort auf „Speichersteuerung im Sinne der EEG vorstellbar?". Nur sinnvoll bei `direction='PRODUCTION'` + `generation_type='pv'` UND das Mitglied hat Batterie-Parameter (`battery_size_kwh` oder `inverter_manufacturer`) angegeben. Service-Layer cleart sonst.

**PROJ-45-Constraint:**
```sql
CHECK (
    (direction = 'CONSUMPTION' AND generation_type IS NULL)
    OR
    (direction = 'PRODUCTION' AND generation_type IN ('pv','hydro','wind','biomass'))
)
```



PROJ-39: vier optionale `address_*`-Spalten erfassen eine abweichende
Standortadresse je Zählpunkt. Wenn alle vier NULL sind, gilt die
Adresse des Mitglieds (`application.resident_*`); wenn mindestens eine
gesetzt ist, müssen alle vier gesetzt sein (All-or-Nothing-Regel im
Service-Layer, nicht via DB-Constraint — damit zukünftige Datenmigrationen
ohne Constraint-Tricks auskommen).

Felder:
- `address_street` — VARCHAR(255), optional
- `address_street_number` — VARCHAR(50), optional
- `address_zip` — VARCHAR(20), optional
- `address_city` — VARCHAR(255), optional

Bricht die ursprüngliche V1-Architekturentscheidung „all metering points
use the same address as the member" aus älteren Versionen dieser Doku.

Stores the metering points of an application.

Fields:
- `id`
- `application_id`
- `metering_point`
- `direction`
- `participation_factor`
- `transformer` *(nullable, configurable via PROJ-8)*
- `installation_number` *(nullable, configurable)*
- `installation_name` *(nullable, configurable)*
- `address_street` / `address_street_number` / `address_zip` / `address_city` *(PROJ-39, all-or-nothing)*
- `generation_type` *(PROJ-45, Pflicht bei PRODUCTION via CHECK)*
- `battery_size_kwh` *(PROJ-45, nullable, configurable, nur PV)*
- `inverter_manufacturer` *(PROJ-45, nullable, configurable, nur PV)*
- `inverter_power_kw` *(Migration 000046, nullable NUMERIC kW, configurable, nur PRODUCTION + PV — Nennleistung des PV-Wechselrichters; Service-Layer cleart das Feld in allen anderen Fällen)*
- `consumption_previous_year` *(PROJ-49, nullable BIGINT kWh, configurable, nur CONSUMPTION)*
- `consumption_forecast` *(PROJ-49, nullable BIGINT kWh, configurable, nur CONSUMPTION)*
- `feed_in_forecast` *(PROJ-49, nullable BIGINT kWh/Jahr, configurable, nur PRODUCTION)*
- `pv_power_kwp` *(PROJ-49, nullable NUMERIC kWp, configurable, nur PRODUCTION + PV)*
- `feed_in_limit_present` *(PROJ-49, nullable boolean, nur PRODUCTION + PV)*
- `feed_in_limit_kw` *(PROJ-49, nullable NUMERIC kW, nur wenn feed_in_limit_present=TRUE)*
- `battery_control_acceptable` *(PROJ-49 follow-up, nullable boolean, nur PRODUCTION + PV + vorhandener Batterie-Parameter)*
- `created_at`
- `updated_at`

Rules:
- one application can have multiple metering points
- `metering_point` is unique within an application
- a metering point may inherit the member's primary address (default) or carry its own deviating address (PROJ-39, see Section 3.3 above). The four `address_*` columns are all-or-nothing — either all four NULL or all four set; enforced server-side
- `generation_type` is NULL for CONSUMPTION and Pflicht für PRODUCTION (DB-CHECK); `battery_size_kwh` + `inverter_manufacturer` werden vom Service auf NULL gesetzt wenn nicht-PV

### 3.4 `member_onboarding.status_log`

Records status changes of an application.

Fields:
- `id`
- `application_id`
- `from_status`
- `to_status`
- `changed_by_user_id`
- `reason`
- `created_at`

### 3.5 `member_onboarding.legal_document`

Per-EEG list of legal documents shown in the public registration form.

Fields:
- `id`
- `rc_number` — references `registration_entrypoint(rc_number)`, ON DELETE CASCADE
- `title` — displayed link text in the form (max 500 chars)
- `url` — URL to the document (max 2048 chars, http/https only)
- `required` — boolean; when `true`, unchecked box blocks form submission
- `sort_order` — integer; ascending display order
- `created_at`
- `updated_at`

Rules:
- max 10 documents per EEG (enforced in application code)
- the central operator privacy policy is NOT stored here — it is configured via env vars (`CENTRAL_POLICY_TITLE`, `CENTRAL_POLICY_URL`) and appended by the backend to every public config response

---

### 3.6 `member_onboarding.document_consent`

Immutable consent snapshots stored at application submission time.

Fields:
- `id`
- `application_id` — references `application(id)`, ON DELETE CASCADE
- `title` — snapshot of document title at submission time
- `url` — snapshot of document URL at submission time
- `is_central_policy` — boolean; `true` for the operator's central privacy policy
- `consented_at` — timestamp of consent (= application submission time)

Rules:
- no foreign key to `legal_document` — deleting a document never affects stored consents
- records are never updated after creation
- an application may have zero consent entries if submitted without consent data

---

### 3.7 `member_onboarding.external_api_key`

Stores the hashed API key for external integrations (see `POST /api/external/v1/applications`). At most one active key exists per EEG.

Fields:
- `id`
- `rc_number` — UNIQUE, references `registration_entrypoint(rc_number)`, ON DELETE CASCADE
- `key_hash` — VARCHAR(64); bcrypt hash of the API key; the plaintext key is never stored
- `revoked_at` — nullable TIMESTAMPTZ; set when the key is revoked; `NULL` means active
- `last_generated_at` — TIMESTAMPTZ; timestamp of the last key generation
- `daily_count` — INT NOT NULL DEFAULT 0; number of submissions today (quota enforcement)
- `quota_date` — nullable DATE; date window for `daily_count` (resets at UTC midnight)
- `created_at`

Indexes:
- UNIQUE `(rc_number)` (implicit via UNIQUE-Constraint)
- `(key_hash) WHERE revoked_at IS NULL` *(Migration 000053)* — Partial-Index für den Lookup-Hot-Path bei jedem externen API-Call

Rules:
- At most one key record per EEG (UNIQUE on `rc_number`)
- The plaintext API key is returned only once at generation time and never stored
- `revoked_at IS NOT NULL` means the key is revoked; all external requests with this key receive `401`
- Revoking does not delete the row; generating a new key replaces the hash in the existing row
- Burst rate limit (10 requests / 60 seconds) is enforced in-memory per pod; daily quota (200 submissions / day) is DB-backed via `daily_count` + `quota_date`

### 3.8 `member_onboarding.reference_number_counter` *(PROJ-35)*

Per-EEG, per-year counter for the new reference-number format `<RC>-<Jahr>-<NNNN>`.

Fields:
- `rc_number` — VARCHAR, FK to `registration_entrypoint(rc_number)`
- `year` — INT
- `last_value` — INT NOT NULL DEFAULT 0; last assigned counter value
- PRIMARY KEY `(rc_number, year)`

Rules:
- Atomically incremented via `INSERT … ON CONFLICT DO UPDATE … RETURNING last_value + 1`
- Per-EEG isolation: parallel submits across EEGs never block each other
- Per-year reset: counter starts at `0001` each calendar year
- Legacy applications created before PROJ-35 keep their `MO-YYYY-NNNNNN` reference numbers (uniqueness across both formats is guaranteed by the column-level UNIQUE on `application.reference_number`)

### 3.9 `data_export_config` (PROJ-60)

> **Hinweis zu `is_obsolete`** (Audit-Welle 8, 2026-05-24): Das Boolean wird
> per `MarkObsolete`-UPDATE beim Backend-Start gesetzt, wenn `plugin_type`
> aus der Code-Registry verschwunden ist. Es ist bewusst materialisiert
> (kein VIEW), weil die Registry ein Code-Runtime-Wert ist, der zur DB-
> Migrationszeit nicht bekannt wäre.

Plugin-specific configurations for the asynchronous data-export framework. Each row is one
named instance of a registered plugin (e.g. one Excel/CSV column-mapping) scoped to one EEG.

- `id` — UUID PRIMARY KEY (default `gen_random_uuid()`)
- `rc_number` — TEXT NOT NULL, FK → `registration_entrypoint(rc_number)` ON DELETE CASCADE
- `plugin_type` — TEXT NOT NULL (stable identifier of a registered plugin, e.g. `"excel"`)
- `name` — TEXT NOT NULL (unique per EEG across all plugin types — UNIQUE INDEX with `deleted_at IS NULL` predicate)
- `config` — JSONB NOT NULL DEFAULT `'{}'` (plugin-specific payload, validated by `Plugin.ValidateConfig` on every write)
- `is_obsolete` — BOOLEAN NOT NULL DEFAULT FALSE (set by `MarkObsoletePluginsOnStartup` when the registry no longer carries `plugin_type`)
- `deleted_at` — TIMESTAMPTZ NULL (soft-delete marker)
- `created_at`, `updated_at` — TIMESTAMPTZ NOT NULL DEFAULT NOW()

Indexes:
- UNIQUE `(rc_number, name)` WHERE `deleted_at IS NULL`
- `(rc_number, plugin_type)` WHERE `deleted_at IS NULL`

Rules:
- Soft-delete: deletion sets `deleted_at`; reads filter `deleted_at IS NULL`. Hard-delete only via the cleanup CronJob after 7 years (DSGVO § 132 BAO).
- Tenant isolation: enforced server-side via JWT `tenant` claim + `parseRCAndCheck` on every endpoint plus a service-layer `cfg.RCNumber == rcNumber` cross-check.

### 3.10 `data_export_job` (PROJ-60)

Async job queue and long-lived audit trail for data-export runs. Never deleted (cleanup
only blanks the BLOB and flips the status to `expired`).

- `id` — UUID PRIMARY KEY (default `gen_random_uuid()`)
- `rc_number` — TEXT NOT NULL, FK → `registration_entrypoint(rc_number)` ON DELETE CASCADE
- `config_id` — UUID NULL, FK → `data_export_config(id)` ON DELETE SET NULL (allows config deletion without losing the audit trail)
- `config_snapshot` — JSONB NOT NULL (frozen copy of `data_export_config.config` at trigger time — running jobs are immune to subsequent config edits)
- `plugin_type` — TEXT NOT NULL (snapshot of `data_export_config.plugin_type` so plugin removal doesn't break the audit)
- `application_ids` — UUID[] NOT NULL (snapshot of the application IDs selected by the admin; deletion of an application produces a dangling reference, which the loader silently skips). **Note (Audit-Welle 8, 2026-05-24):** Array-Spalten widersprechen dem Geist der Repo-Regel „no JSON columns". Hier bewusste Ausnahme — der Snapshot-Charakter (immutability nach Job-Trigger) lässt eine Junction-Table `data_export_job_application` unattraktiv erscheinen (Backfill + Concurrency-Sicherheit beim INSERT). Falls jemals Cross-Job-Statistiken über Anträge nötig werden, ist die Normalisierung der bessere Pfad.
- `status` — TEXT NOT NULL CHECK IN (`queued`, `running`, `done`, `failed`, `expired`)
- `admin_user_id` — TEXT NOT NULL (Keycloak `sub` of the admin who triggered the job)
- `processed_count` — INTEGER NOT NULL DEFAULT 0
- `total_count` — INTEGER NOT NULL
- `result_summary` — JSONB NULL (e.g. `{"downloaded": 47, "file_size": 12345}`)
- `error_message` — TEXT NULL (user-safe text — internal `%v` details only go to slog)
- `retry_count` — INTEGER NOT NULL DEFAULT 0 (incremented by zombie-recovery)
- `created_at`, `started_at`, `finished_at` — TIMESTAMPTZ

Indexes:
- `(status, created_at)` WHERE `status = 'queued'` — Worker queue-poll
- `(rc_number, status)` WHERE `status IN ('queued', 'running')` — concurrency-limit check
- `(rc_number, created_at DESC)` — BackOffice job-list
- `(started_at)` WHERE `status = 'running'` — zombie-recovery scan

Rules:
- Pickup: `SELECT ... FOR UPDATE SKIP LOCKED LIMIT 1` — multi-replica-safe.
- Concurrency soft-limit: max 3 queued+running per EEG (race-tolerant — bursts up to 4-5 are intentionally accepted).
- Sensitive-export audit: when the snapshot contains `iban` or `birth_date`, the worker emits `slog.Info classification=sensitive-export` with `admin_user_id` for DSGVO compliance.

### 3.11 `data_export_result` (PROJ-60)

File BLOBs with 24-hour TTL, written by download-style plugins (Excel/CSV in V1). Push-style
plugins do not populate this table.

- `job_id` — UUID PRIMARY KEY, FK → `data_export_job(id)` ON DELETE CASCADE
- `file_name` — TEXT NOT NULL (built by the worker as `{rc_number}-{config_name}-{YYYY-MM-DD}.{ext}` with path-traversal characters stripped)
- `mime_type` — TEXT NOT NULL
- `file_bytes` — BYTEA NOT NULL
- `file_size` — INTEGER NOT NULL
- `expires_at` — TIMESTAMPTZ NOT NULL (`created_at + 24h`)
- `downloaded_at` — TIMESTAMPTZ NULL (best-effort, bumped on first download)
- `created_at` — TIMESTAMPTZ NOT NULL DEFAULT NOW()

Indexes:
- `(expires_at)` — TTL cleanup scan

Rules:
- TTL cleanup: the `data-export-cleanup` CronJob (`*/10 * * * *`) deletes rows where `expires_at < NOW()` and flips the corresponding job to `status='expired'`.
- BLOB-Auth: download endpoint re-validates tenant via the job's `rc_number`.
- Spreadsheet-injection defense: cell values starting with `=`, `+`, `-`, `@`, TAB, or CR (after stripping leading SPACE/NBSP/BOM) are prefixed with `'` by `excel/renderer.sanitiseSpreadsheetValue`.

### 3.12 `member_onboarding.reconciliation_run` *(PROJ-69)*

Header row per reconciliation run (one EEG, one trigger). The throttle
guarantee — at most one run per EEG per UTC day — is enforced via a UNIQUE
index, not by application code, so multiple browser tabs or pods can race
the INSERT and the database picks exactly one winner.

- `id` — UUID PRIMARY KEY DEFAULT `gen_random_uuid()`
- `rc_number` — VARCHAR NOT NULL, FK → `registration_entrypoint(rc_number)` ON DELETE CASCADE
- `started_at` — TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `finished_at` — TIMESTAMPTZ NULL (NULL while running; stale-recovery >1h marks via `error_detail='stale-run-recovered'`)
- `triggered_by` — VARCHAR NOT NULL — trigger kind (today only `'login'`; reserved: `'on_demand'`)
- `triggered_by_user` — VARCHAR NOT NULL — Keycloak subject of the admin whose login triggered the run
- `total_apps_checked` — INT NOT NULL DEFAULT 0
- `matched_count` — INT NOT NULL DEFAULT 0
- `ambiguous_count` — INT NOT NULL DEFAULT 0
- `conflict_count` — INT NOT NULL DEFAULT 0 (sum of `mnr_conflict` + `duplicate_application`)
- `already_handed_over_count` — INT NOT NULL DEFAULT 0 (PROJ-64 race: `faktura_handover_at` was non-NULL already)
- `error_count` — INT NOT NULL DEFAULT 0
- `error_detail` — TEXT NULL (only `'stale-run-recovered'` in normal operation)

Indexes:
- UNIQUE `(rc_number, ((started_at AT TIME ZONE 'UTC')::date))` — throttle source-of-truth
- `(rc_number)`
- `(started_at)`

Rules:
- Lifecycle managed by `internal/application/reconciliation_repo.go`: `AcquireRunLock` → service iterates apps → `FinalizeRun`.
- Stale-Recovery: at `AcquireRunLock` time, any run for this EEG that's still `finished_at IS NULL` and started >1h ago is force-closed with `error_detail='stale-run-recovered'`. Lets the next run claim the day's slot after a pod crash.

### 3.13 `member_onboarding.reconciliation_match_detail` *(PROJ-69)*

Per-application result row. Only **positive** outcomes are stored (matched,
ambiguous, mnr_conflict, duplicate_application, error) — no row for no-match
or already-handed-over to avoid log-bloat.

- `id` — UUID PRIMARY KEY DEFAULT `gen_random_uuid()`
- `run_id` — UUID NOT NULL, FK → `reconciliation_run(id)` ON DELETE CASCADE
- `application_id` — UUID NOT NULL, FK → `application(id)` ON DELETE CASCADE
- `core_member_number` — TEXT NULL (Faktura participantNumber; NULL when match was made but core had no MNr)
- `result` — VARCHAR NOT NULL CHECK IN (`'matched'`, `'ambiguous'`, `'mnr_conflict'`, `'duplicate_application'`, `'error'`)
- `error_detail` — TEXT NULL (only when `result='error'`; truncated to ≤500 chars by the service)
- `created_at` — TIMESTAMPTZ NOT NULL DEFAULT NOW()

Indexes:
- `(run_id)`
- `(application_id)`

Rules:
- Audit-trail only — no admin-UI surface in V1 (psql-inspection only). EEG-decommissioning cascades the run + details away with the parent `registration_entrypoint` row; intentional accept (no separate retention table).
- Service writes via the repo's `InsertMatchDetail`; mock-friendly via the `ReconciliationServiceRepo` interface.

---

## 4. Status Model

Allowed status values (12):
- `draft`
- `submitted`
- `email_confirmed` *(PROJ-31, only reached when the EEG opts in to e-mail confirmation)*
- `under_review`
- `needs_info`
- `approved`
- `rejected`
- `imported` *(transient — Import-Service auto-routes immediately, see PROJ-46)*
- `import_failed`
- `awaiting_bank_confirmation` *(PROJ-46, only at `einzugsart=b2b`, set automatically by import service)*
- `ready_for_activation` *(PROJ-46, set automatically by import service for non-b2b, or by admin after bank confirmation)*
- `activated` *(PROJ-46, strict end state — no transitions out, no reset)*

Allowed transitions:
- `draft -> submitted`
- `submitted -> under_review`
- `submitted -> email_confirmed` *(PROJ-31, only via member click on `POST /api/public/applications/confirm-email`. Not exposed on the admin `/status` endpoint.)*
- `submitted -> rejected` *(PROJ-31, admin anti-spam override before confirmation)*
- `email_confirmed -> under_review`
- `email_confirmed -> needs_info`
- `email_confirmed -> approved`
- `email_confirmed -> rejected`
- `under_review -> needs_info`
- `under_review -> approved`
- `under_review -> rejected`
- `needs_info -> submitted`
- `approved -> imported`
- `approved -> import_failed`
- `approved -> activated` *(PROJ-53, admin manuell via `POST /api/admin/applications/{id}/mark-activated` — Ausnahmefall wenn Mitglied im Core bereits existiert und manuell überschrieben wurde; Mitgliedsnummer-Pflichteingabe; Import-Pfad wird übersprungen)*
- `import_failed -> approved`
- `imported -> awaiting_bank_confirmation` *(PROJ-46, auto by import service when `einzugsart=b2b`. Not exposed on `/status`.)*
- `imported -> ready_for_activation` *(PROJ-46, auto by import service for non-b2b. Not exposed on `/status`.)*
- `awaiting_bank_confirmation -> ready_for_activation` *(PROJ-46, admin manuell nach Bank-Bestätigung)*
- `awaiting_bank_confirmation -> under_review` *(PROJ-46, admin rückwärts)*
- `ready_for_activation -> activated` *(PROJ-46, admin manuell ODER Batch-Button `POST /api/admin/applications/check-activation`)*
- `ready_for_activation -> under_review` *(PROJ-46, admin rückwärts)*
- `imported -> approved` *(PROJ-30, only via `POST /reset-import`, never via generic `/status`)*
- `awaiting_bank_confirmation -> approved` *(PROJ-46, via `POST /reset-import`)*
- `ready_for_activation -> approved` *(PROJ-46, via `POST /reset-import`)*

When `registration_entrypoint.require_email_confirmation = TRUE` (PROJ-31), the generic admin `/status` endpoint rejects `submitted -> under_review|needs_info|approved` with 409 until the member has clicked the confirmation link. `submitted -> rejected` remains available as the admin's anti-spam override.

The set of allowed status values is enforced in **three places** (Go constants in `internal/shared/models.go`, `adminTransitions` map in `internal/application/admin_service.go`, and the `application_status_check` CHECK constraint — see migration `000041_post_import_statuses.up.sql` for the latest DROP-and-re-ADD pattern). All three must be updated when introducing a new status.

## 5. Business Rules

- One application contains exactly one member.
- One application belongs to exactly one EEG.
- An application is started via the EEG's RC number.
- The RC number is resolved via `member_onboarding.registration_entrypoint`; no direct access to eegFaktura core tables.
- The field `rc_number` in `application` stores the RC number through which the application was started.
- If `registration_entrypoint.is_active = false`, the registration is rejected (HTTP 410).
- One application can contain multiple metering points.
- A metering point may inherit the member's primary address (default) or carry its own deviating address (PROJ-39 — see Section 3.3 above). All four `address_*` columns are either NULL together or all set together; the all-or-nothing rule is enforced server-side.
- Tariffs, roles, and account information are only maintained after import into eegFaktura.
- Only applications in status `approved` may be imported.
