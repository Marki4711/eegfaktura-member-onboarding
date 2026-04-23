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
- documents
- tariffs
- role management
- separate metering point addresses
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
- `is_active`
- `contact_email` — nullable, EEG notification email address
- `intro_text` — nullable, sanitized HTML string for the public registration form
- `eeg_name` — nullable, official name of the energy community
- `eeg_street` — nullable, street of the EEG address
- `eeg_street_number` — nullable, house number of the EEG address
- `eeg_zip` — nullable, postal code of the EEG address
- `eeg_city` — nullable, city of the EEG address
- `creditor_id` — nullable, SEPA creditor ID (max 35 chars)
- `sepa_mandate_enabled` — boolean, default false; controls whether SEPA mandate PDF is attached to welcome email
- `created_at`
- `updated_at`

Rules:
- `rc_number` is unique
- only entries with `is_active = true` allow a registration
- maintenance is performed by admins or through deployment configuration

---

### 3.1a `member_onboarding.field_config`

Per-EEG configuration of optional form fields. Only explicitly configured values are stored (sparse table); missing entries fall back to system defaults.

Fields:
- `id`
- `rc_number` — references `registration_entrypoint(rc_number)`, ON DELETE CASCADE
- `field_name` — name of the configurable field (e.g. `heat_pump`, `transformer`)
- `state` — `hidden` | `optional` | `required`
- `updated_at`

Rules:
- `(rc_number, field_name)` is unique
- `field_name` must be one of the centrally registered configurable fields (enforced in application code)
- `state` is constrained to `hidden`, `optional`, `required` (DB CHECK constraint)
- missing entries default to `hidden` for new fields; `optional` for `phone`, `birth_date`, `uid_number`

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
- admin note
- import status

Fields:
- `id`
- `reference_number`
- `rc_number`
- `status`
- `started_at`
- `submitted_at`
- `approved_at`
- `rejected_at`
- `imported_at`
- `firstname`
- `lastname`
- `birth_date`
- `company_name`
- `uid_number`
- `register_number`
- `member_type`
- `email`
- `phone`
- `resident_street`
- `resident_street_number`
- `resident_zip`
- `resident_city`
- `resident_country`
- `privacy_accepted`
- `privacy_version`
- `privacy_accepted_at`
- `accuracy_confirmed`
- `iban`
- `account_holder`
- `sepa_mandate_accepted`
- `sepa_mandate_accepted_at`
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
- `consumption_previous_year` *(nullable integer kWh, configurable)*
- `consumption_forecast` *(nullable integer kWh, configurable)*
- `feed_in_forecast` *(nullable integer kWh, configurable)*
- `pv_power_kwp` *(nullable decimal kWp, configurable)*
- `heat_pump` *(nullable boolean, configurable)*
- `electric_vehicle` *(nullable boolean, configurable)*
- `electric_hot_water` *(nullable boolean, configurable)*

### 3.3 `member_onboarding.metering_point`

Stores the metering points of an application.

Fields:
- `id`
- `application_id`
- `metering_point`
- `direction`
- `participation_factor`
- `transformer` *(nullable, configurable)*
- `installation_number` *(nullable, configurable)*
- `installation_name` *(nullable, configurable)*
- `created_at`
- `updated_at`

Rules:
- one application can have multiple metering points
- `metering_point` is unique within an application
- all metering points use the same address as the member in onboarding

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

## 4. Status Model

Allowed status values:
- `draft`
- `submitted`
- `under_review`
- `needs_info`
- `approved`
- `rejected`
- `imported`
- `import_failed`

Allowed transitions:
- `draft -> submitted`
- `submitted -> under_review`
- `under_review -> needs_info`
- `under_review -> approved`
- `under_review -> rejected`
- `needs_info -> submitted`
- `approved -> imported`
- `approved -> import_failed`
- `import_failed -> approved`

## 5. Business Rules

- One application contains exactly one member.
- One application belongs to exactly one EEG.
- An application is started via the EEG's RC number.
- The RC number is resolved via `member_onboarding.registration_entrypoint`; no direct access to eegFaktura core tables.
- The field `rc_number` in `application` stores the RC number through which the application was started.
- If `registration_entrypoint.is_active = false`, the registration is rejected (HTTP 410).
- One application can contain multiple metering points.
- All metering points use the same address as the member in onboarding.
- Tariffs, roles, and account information are only maintained after import into eegFaktura.
- Only applications in status `approved` may be imported.
