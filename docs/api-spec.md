# API Specification
## eegfaktura Member Onboarding

## 1. Scope

This API specifies the interfaces for:

- Public Registration API
- Admin API
- internal import flow toward eegFaktura Core

Not part of this API:
- direct core APIs
- Keycloak configuration
- tariff/role management
- document uploads

## 2. General Rules

- Format: JSON
- API style: REST
- UTF-8
- Timestamps: ISO-8601 / RFC3339
- DB schema: `member_onboarding`
- Tables:
  - `member_onboarding.registration_entrypoint`
  - `member_onboarding.application`
  - `member_onboarding.metering_point`
  - `member_onboarding.status_log`
  - `member_onboarding.legal_document`
  - `member_onboarding.document_consent`

## 3. Authentication

### Public API (`/api/public/*`)
No authentication required. The public registration form is intentionally open.

### External API (`/api/external/*`)
API-key authentication — no Keycloak required. Each API key is bound to exactly one EEG.

```
Authorization: Bearer moak_<32-char-random-key>
```

Keys are generated and revoked in the Admin Settings page (see section 6.12–6.14).
The key value is shown only once at generation time and cannot be retrieved again.

### Admin API (`/api/admin/*`)
Authentication via the existing eegFaktura/Keycloak mechanism (JWT Bearer token).
The business logic additionally validates the EEG authorization in the backend.

---

## 4. Domain Types

### Status
Allowed values:
- `draft`
- `submitted`
- `under_review`
- `needs_info`
- `approved`
- `rejected`
- `imported`
- `import_failed`

### Meter Direction
Allowed values:
- `CONSUMPTION`
- `PRODUCTION`

---

## 5. Public API

## 5.1 Load registration entry point

### GET `/api/public/registration/{rc_number}`

Loads the basic configuration for a fixed registration link based on the EEG's RC number.

The RC number is validated against `member_onboarding.registration_entrypoint`.
No direct access to eegFaktura core tables takes place.

### Path params
- `rc_number: string` — RC number of the EEG

### Response 200
```json
{
  "rcNumber": "RC123456",
  "title": "Mitglied werden",
  "active": true,
  "fieldConfig": {
    "phone": "optional",
    "birth_date": "optional",
    "heat_pump": "required",
    "transformer": "hidden"
  },
  "introText": "<p>Willkommen!</p>",
  "sepaMandateEnabled": true,
  "showCentralPolicy": true,
  "legalDocuments": [
    {
      "id": "3f8c8c2d-...",
      "title": "Satzung der Energiegemeinschaft",
      "url": "https://example.at/satzung.pdf",
      "required": true,
      "sortOrder": 0,
      "isCentralPolicy": false
    },
    {
      "id": "00000000-0000-0000-0000-000000000000",
      "title": "Datenschutzerklärung",
      "url": "https://example.at/datenschutz",
      "required": true,
      "sortOrder": 9999,
      "isCentralPolicy": true
    }
  ]
}
```

`fieldConfig` contains only explicitly configured fields. Missing fields fall back to system defaults (`hidden` for new fields, `optional` for `phone`, `birth_date`, `uid_number`). Fields with admin state `admin_only` are returned as `"hidden"` — they are never shown to the member.

`introText` is `null` when no text is configured.

`sepaMandateEnabled` is `false` by default. When `true`, SEPA mandate checkboxes and PDF generation are activated.

`showCentralPolicy` controls whether the central operator privacy policy is included in `legalDocuments`. Defaults to `true`. When `false`, the central policy entry is omitted from the list even if env vars are set — intended for EEGs that configure their own privacy policy as a custom document.

`legalDocuments` contains the central privacy policy entry (`isCentralPolicy: true`) when `showCentralPolicy = true` and `CENTRAL_POLICY_URL` is set. EEG-specific documents precede it, ordered by `sortOrder`. The central policy is not stored in the database — it is configured via `CENTRAL_POLICY_TITLE` / `CENTRAL_POLICY_URL` env vars.

### Errors
- `404` if `rc_number` is not found in `registration_entrypoint`
- `410` if `registration_entrypoint.is_active = false`

---

## 5.2 Create application

### POST `/api/public/applications`

Creates a new application.

### Request
```json
{
  "rcNumber": "RC123456",
  "firstname": "Max",
  "lastname": "Muster",
  "birthDate": "1985-06-15",
  "email": "max.muster@example.at",
  "phone": "0664/1234567",
  "residentStreet": "Musterstraße",
  "residentStreetNumber": "2",
  "residentZip": "4020",
  "residentCity": "Linz",
  "privacyAccepted": true,
  "privacyVersion": "2026-01",
  "accuracyConfirmed": true,
  "iban": "AT611904300234573201",
  "accountHolder": "Max Muster",
  "sepaMandateAccepted": true,
  "meteringPoints": [
    {
      "meteringPoint": "AT0031000000000000000000990022105",
      "direction": "CONSUMPTION",
      "participationFactor": 1.0,
      "transformer": "T1",
      "installationNumber": "12345",
      "installationName": "PV Dach"
    }
  ],
  "membershipStartDate": "2026-05-01",
  "personsInHousehold": 3,
  "consumptionPreviousYear": 4200,
  "consumptionForecast": 4000,
  "feedInForecast": 6000,
  "pvPowerKwp": 9.9,
  "heatPump": true,
  "electricVehicle": false,
  "electricHotWater": null
}
```

All fields under `meteringPoints[].transformer/installationNumber/installationName` and the application-level energy/household fields are optional by default. Whether they are required is determined by the EEG's `fieldConfig` (see 5.1). Fields not relevant to the current `memberType` are ignored.

### Rules
- `rcNumber` required
- `firstname` required
- `lastname` required
- `email` required
- `residentStreet` required
- `residentStreetNumber` required
- `residentZip` required
- `residentCity` required
- at least one `meteringPoint`
- `meteringPoint` must be unique within the request
- `direction` must be `CONSUMPTION` or `PRODUCTION`
- `privacyAccepted` must be `true`
- `accuracyConfirmed` must be `true`
- `privacyVersion` required when `privacyAccepted = true`
- `iban` required (15–34 characters, whitespace is normalized)
- `accountHolder` required
- `sepaMandateAccepted` must be `true`

### Response 201
```json
{
  "id": "3f8c8c2d-....",
  "referenceNumber": "MO-2026-000001",
  "status": "draft",
  "createdAt": "2026-04-18T12:00:00Z",
  "updatedAt": "2026-04-18T12:00:00Z"
}
```

### Errors
- `400` validation error
- `404` unknown `rcNumber`
- `410` registration disabled (`is_active = false`)
- `409` duplicate metering point number in the same request

---

## 5.3 Update application

### PUT `/api/public/applications/{id}`

Updates an existing application in status `draft` or `needs_info`.

### Path params
- `id: uuid`

### Request
Same model as Create.

### Rules
- only allowed in status `draft` or `needs_info`
- existing metering points are fully replaced by the request

### Response 200
```json
{
  "id": "3f8c8c2d-....",
  "referenceNumber": "MO-2026-000001",
  "status": "draft",
  "updatedAt": "2026-04-18T12:30:00Z"
}
```

### Errors
- `400` validation error
- `404` application not found
- `409` status does not allow editing

---

## 5.4 Submit application

### POST `/api/public/applications/{id}/submit`

Submits the application.

### Path params
- `id: uuid`

### Request
Optional body with consent snapshots:

```json
{
  "consents": [
    {
      "title": "Satzung der Energiegemeinschaft",
      "url": "https://example.at/satzung.pdf",
      "isCentralPolicy": false
    },
    {
      "title": "Datenschutzerklärung",
      "url": "https://example.at/datenschutz",
      "isCentralPolicy": true
    }
  ]
}
```

`consents` is optional. Each entry is a snapshot of the document title and URL at the time of submission. If not provided, no consent entries are stored. The backend does not validate consent entries against configured `legal_document` records — the frontend is responsible for sending the correct entries.

### Rules
Before submit, the following must be set:
- `firstname`
- `lastname`
- `email`
- `residentStreet`
- `residentStreetNumber`
- `residentZip`
- `residentCity`
- at least one metering point
- `privacyAccepted = true`
- `privacyVersion` set
- `privacyAcceptedAt` is set server-side
- `accuracyConfirmed = true`

### Response 200
```json
{
  "id": "3f8c8c2d-....",
  "referenceNumber": "MO-2026-000001",
  "status": "submitted",
  "submittedAt": "2026-04-18T12:35:00Z"
}
```

### Effects
- `application.status = submitted`
- set `application.submitted_at`
- write entry in `status_log`

### Errors
- `400` required fields missing
- `404` application not found
- `409` application already submitted or in a disallowed status

---

## 6. Admin API

## 6.1 List applications

### GET `/api/admin/applications`

Returns the admin list.

### Query params
- `status`
- `rc_number`
- `reference_number`
- `lastname`
- `email`
- `metering_point`
- `submitted_from`
- `submitted_to`
- `page`
- `page_size`

### Response 200
```json
{
  "items": [
    {
      "id": "3f8c8c2d-....",
      "referenceNumber": "MO-2026-000001",
      "rcNumber": "RC123456",
      "status": "submitted",
      "firstname": "Josef",
      "lastname": "Brandstätter",
      "email": "max@example.org",
      "submittedAt": "2026-04-18T12:35:00Z",
      "meteringPoints": [
        "AT0031000000000000000000990022105"
      ]
    }
  ],
  "page": 1,
  "pageSize": 20,
  "total": 1
}
```

### Rules
- only applications for the EEGs the user is authorized for

---

## 6.2 Get application detail

### GET `/api/admin/applications/{id}`

### Response 200
```json
{
  "id": "3f8c8c2d-....",
  "referenceNumber": "MO-2026-000001",
  "rcNumber": "RC123456",
  "status": "submitted",
  "firstname": "Max",
  "lastname": "Muster",
  "birthDate": "1985-06-15",
  "email": "max.muster@example.at",
  "phone": "0664/1234567",
  "residentStreet": "Musterstraße",
  "residentStreetNumber": "2",
  "residentZip": "4020",
  "residentCity": "Linz",
  "privacyAccepted": true,
  "privacyVersion": "2026-01",
  "privacyAcceptedAt": "2026-04-18T12:35:00Z",
  "accuracyConfirmed": true,
  "communicationConsent": false,
  "adminNote": null,
  "needsInfoReason": null,
  "meteringPoints": [
    {
      "id": "1a....",
      "meteringPoint": "AT0031000000000000000000990022105",
      "direction": "CONSUMPTION"
    }
  ],
  "statusLog": [
    {
      "fromStatus": "draft",
      "toStatus": "submitted",
      "changedByUserId": null,
      "reason": "submitted by public user",
      "createdAt": "2026-04-18T12:35:00Z"
    }
  ],
  "consents": [
    {
      "id": "1a2b3c...",
      "title": "Satzung der Energiegemeinschaft",
      "url": "https://example.at/satzung.pdf",
      "isCentralPolicy": false,
      "consentedAt": "2026-04-18T12:35:00Z"
    },
    {
      "id": "2b3c4d...",
      "title": "Datenschutzerklärung",
      "url": "https://example.at/datenschutz",
      "isCentralPolicy": true,
      "consentedAt": "2026-04-18T12:35:00Z"
    }
  ]
}
```

`consents` contains the immutable snapshots of legal document consents recorded at submission time. Empty array when no consents were submitted.

### Errors
- `404` not found
- `403` not authorized for EEG

---

## 6.3 Update application as admin

### PUT `/api/admin/applications/{id}`

### Request
```json
{
  "firstname": "Max",
  "lastname": "Muster",
  "birthDate": "1985-06-15",
  "email": "max.muster@example.at",
  "phone": "0664/1234567",
  "residentStreet": "Musterstraße",
  "residentStreetNumber": "2",
  "residentZip": "4020",
  "residentCity": "Linz",
  "adminNote": "Telefonnummer verifiziert",
  "meteringPoints": [
    {
      "meteringPoint": "AT0031000000000000000000990022105",
      "direction": "CONSUMPTION"
    }
  ]
}
```

### Rules
- editable in `submitted`, `under_review`, `needs_info`, `approved`, `import_failed`
- metering points are fully replaced

### Response 200
```json
{
  "id": "3f8c8c2d-....",
  "updatedAt": "2026-04-18T13:00:00Z"
}
```

---

## 6.4 Change status

### POST `/api/admin/applications/{id}/status`

### Request
```json
{
  "toStatus": "approved",
  "reason": "Application fully reviewed"
}
```

### Allowed transitions
- `submitted -> under_review`
- `under_review -> needs_info`
- `under_review -> approved`
- `under_review -> rejected`
- `needs_info -> submitted`
- `approved -> imported`
- `approved -> import_failed`
- `import_failed -> approved`

### Side effects
- on `approved`: set `approved_at`, set `reviewed_by_user_id`
- on `rejected`: set `rejected_at`, set `reviewed_by_user_id`
- on `needs_info`: set `needs_info_reason`
- always write entry in `status_log`

### Response 200
```json
{
  "id": "3f8c8c2d-....",
  "status": "approved"
}
```

### Errors
- `400` invalid target status
- `403` not authorized
- `409` disallowed status transition

---

## 6.5 Import application

### POST `/api/admin/applications/{id}/import`

### Request body (optional, PROJ-27)

```json
{
  "tariffId": "uuid-of-EEG-tariff-or-empty",
  "meterTariffs": {
    "AT0030000000000000000000012345678": "uuid-of-VZP-or-EZP-tariff"
  }
}
```

The body is fully optional — an empty body or omitted JSON keeps the legacy
"no tariffs" behaviour (the participant is created in the core with no
tariff assignment, the admin pflegt es manuell im Core nach).

- `tariffId`: UUID of an `EEG`-type tariff (Mitgliedsbeitrag). Set on the
  participant via a follow-up `PUT /participant/v2/{participantId}` call after
  the participant is created, because the core's `EegParticipantBase.TariffId`
  is `goqu:"skipinsert"` and is ignored on participant insert.
- `meterTariffs`: map of `meteringPoint` → tariff UUID. Goes into the
  `meters[].tariff_id` field of the `POST /participant` body directly.

### Rules
- only status `approved`
- only authorized admins
- import runs synchronously in V1

### Response 200
```json
{
  "success": true,
  "applicationId": "3f8c8c2d-....",
  "status": "imported",
  "targetParticipantId": "4711",
  "memberTariffWarning": "core returned HTTP 404"
}
```

`memberTariffWarning` (PROJ-27) is only present when the participant was
created successfully but the follow-up call to set the member-level tariff
failed. The application is still moved to `imported` — meter tariffs are
persisted; the admin needs to set the member tariff manually in the core.

### Failure response 409 / 422 / 500
```json
{
  "success": false,
  "applicationId": "3f8c8c2d-....",
  "status": "import_failed",
  "message": "participant import failed"
}
```

### Side effects on success
- set `import_started_at`
- set `import_finished_at`
- set `imported_at`
- set `target_participant_id`
- `status = imported`
- write `status_log`

### Side effects on failure
- set `import_started_at`
- set `import_finished_at`
- set `import_error_message`
- `status = import_failed`
- write `status_log`

---

## 6.5.05 Tariff lookup (PROJ-27)

### GET `/api/admin/tariffs?rcNumber={rcNumber}`

Proxies the eegFaktura core's `GET /eeg/tariff` for the import-time tariff
selection dialog. Tenant-Admin scope: the `rcNumber` must be in the admin's
JWT `Tenants` claim (or the admin is a superuser).

### Response 200
```json
{
  "tariffs": [
    {
      "id": "dfd00405-9a42-11ee-ad15-22b3d9edaadd",
      "type": "EZP",
      "name": "Einspeisetarif Landwirt",
      "centPerKWh": 11,
      "discount": 0,
      "useVat": true,
      "vatInPercent": 13,
      "inactiveSince": null
    }
  ]
}
```

The frontend filters tariffs by `type` (`EEG` for the member dropdown,
`VZP`/`EZP` for the meter dropdowns) and hides entries with `inactiveSince`
set. The full upstream payload contains more pricing fields (`participantFee`,
`baseFee`, `freeKWh`, `meteringPointFee`, ...); only the subset above is
exposed to the frontend.

### Failure responses
- `400` rcNumber missing
- `403` tenant mismatch
- `503` core unavailable — the frontend then offers an "Import ohne Tarife"
  fallback (the import still runs, no tariff assignments).

---

## 6.5.1 Reset import (PROJ-30)

### POST `/api/admin/applications/{id}/reset-import`

Transitions an application from `imported` back to `approved` so it can be
re-imported after the eegFaktura admin deleted the participant in the core.
No call to the core — the admin verifies the deletion manually.

### Request
```json
{
  "reason": "Mitglied versehentlich importiert, Daten in der Faktura gelöscht."
}
```

| Field | Required | Constraints |
|---|---|---|
| `reason` | yes | 5–500 chars (after trimming) |

### Rules
- Application must be in status `imported` (otherwise 409).
- The transition `imported → approved` is **only** reachable via this
  endpoint; the generic `POST /status` does not accept it.
- Tenant-Admin scope: must match the EEG of the application.

### Response 200
Returns the full `AdminApplicationDetail` after the reset (status now
`approved`, `targetParticipantId` cleared).

### Side effects
- `status = approved`
- `import_started_at = NULL`
- `import_finished_at = NULL`
- `imported_at = NULL`
- `target_participant_id = NULL`
- `import_error_message = NULL`
- write `status_log` entry with `from='imported'`, `to='approved'`,
  `reason = <user reason>\n[system] previous target_participant_id=<uuid>`
  (the old participant UUID is archived in the log so the audit trail
  preserves it after the column is cleared)

### Failure responses
- `400` reason missing / too short / too long
- `403` tenant mismatch
- `409` application not in `imported` status

---

## 6.6 Get field config

### GET `/api/admin/settings/fields?rc_number={rc_number}`

Returns the stored field configuration for an EEG. Only explicitly saved overrides are returned; the frontend applies defaults for missing fields.

### Query params
- `rc_number` — required

### Response 200
```json
{
  "rcNumber": "RC123456",
  "fieldConfig": {
    "heat_pump": { "state": "required", "adminValue": null },
    "transformer": { "state": "optional", "adminValue": null },
    "persons_in_household": { "state": "admin_only", "adminValue": "3" }
  }
}
```

Each field entry contains `state` and optionally `adminValue`. `adminValue` is only relevant when `state = "admin_only"` and is automatically applied to new applications.

### Errors
- `400` missing `rc_number`
- `403` not authorized for this EEG

---

## 6.7 Save field config

### PUT `/api/admin/settings/fields?rc_number={rc_number}`

Replaces the field configuration for an EEG atomically. Unknown field names and invalid states are silently skipped.

### Query params
- `rc_number` — required

### Request body
```json
{
  "phone": { "state": "required" },
  "birth_date": { "state": "optional" },
  "heat_pump": { "state": "required" },
  "transformer": { "state": "hidden" },
  "persons_in_household": { "state": "admin_only", "adminValue": "3" }
}
```

Allowed field names: `phone`, `birth_date`, `uid_number`, `membership_start_date`, `persons_in_household`, `consumption_previous_year`, `consumption_forecast`, `feed_in_forecast`, `pv_power_kwp`, `heat_pump`, `electric_vehicle`, `electric_hot_water`, `transformer`, `installation_number`, `installation_name`

Allowed states: `hidden`, `optional`, `required`, `admin_only`

When `state = "admin_only"`: the field is hidden from the public registration form; `adminValue` is automatically written to new applications (server-side type conversion: int via `Sscanf %d`, float via `%f`, bool via `"true"/"false"`, date via `YYYY-MM-DD`). Invalid values result in NULL (no error).

### Response
- `204 No Content` on success

### Errors
- `400` missing `rc_number` or invalid JSON
- `403` not authorized for this EEG

---

## 6.8 Get intro text

### GET `/api/admin/settings/intro-text?rc_number={rc_number}`

Returns the stored intro text for an EEG. Returns `null` when no text is configured.

### Query params
- `rc_number` — required

### Response 200
```json
{
  "rcNumber": "RC123456",
  "introText": "<p>Willkommen bei unserer Energiegemeinschaft!</p>"
}
```

`introText` is `null` when no text has been saved yet.

### Errors
- `400` missing `rc_number`
- `403` not authorized for this EEG
- `404` RC number not found

---

## 6.9 Save intro text

### PUT `/api/admin/settings/intro-text?rc_number={rc_number}`

Saves the intro text for an EEG. The text is sanitized server-side (only `p`, `br`, `strong`, `b`, `em`, `i`, `ul`, `ol`, `li`, `a[href]` are allowed). Send `null` or empty string to clear the text.

### Query params
- `rc_number` — required

### Request body
```json
{
  "introText": "<p>Willkommen! Bitte füllen Sie das Formular aus.</p>"
}
```

Send `{ "introText": null }` to clear the text (public form will show default text).

### Response
- `204 No Content` on success

### Errors
- `400` missing `rc_number` or invalid JSON
- `403` not authorized for this EEG
- `404` RC number not found

---

## 6.10 Get EEG settings

### GET `/api/admin/settings/eeg?rc_number={rc_number}`

Returns the EEG master data used for SEPA mandate PDF generation.

### Response 200
```json
{
  "rcNumber": "RC123456",
  "eegName": "Muster Energiegemeinschaft",
  "eegStreet": "Hauptstraße",
  "eegStreetNumber": "12",
  "eegZip": "4020",
  "eegCity": "Linz",
  "creditorId": "AT28ZZZ00000000000",
  "registrationActive": true,
  "sepaMandateEnabled": true,
  "useCompanySEPAMandate": false,
  "showCentralPolicy": true,
  "memberNumberStart": 1
}
```

All address/name fields are `null` when not yet configured. `registrationActive` is `false` by default — new EEGs are inactive until explicitly activated by an admin. `sepaMandateEnabled` defaults to `false`. `useCompanySEPAMandate` defaults to `false`. `showCentralPolicy` defaults to `true`. `memberNumberStart` defaults to `1` — the first member number assigned for this EEG will be this value.

### Errors
- `400` missing `rc_number`
- `403` not authorized for this EEG

---

## 6.11 Save EEG settings

### PUT `/api/admin/settings/eeg?rc_number={rc_number}`

### Request body
```json
{
  "eegName": "Muster Energiegemeinschaft",
  "eegStreet": "Hauptstraße",
  "eegStreetNumber": "12",
  "eegZip": "4020",
  "eegCity": "Linz",
  "creditorId": "AT28ZZZ00000000000",
  "registrationActive": true,
  "sepaMandateEnabled": true,
  "useCompanySEPAMandate": false,
  "showCentralPolicy": true,
  "memberNumberStart": 1
}
```

`registrationActive`: enables or disables the public registration form for this EEG. When `false`, `GET /api/public/registration/{rc_number}` returns `410 Gone`.

`showCentralPolicy`: when `false`, the central operator privacy policy is not shown in the public registration form. Intended for EEGs that have configured their own privacy policy as a custom document (see 6.16).

`useCompanySEPAMandate`: when `true`, members of type `company` or `association` receive the SEPA B2B mandate PDF instead of the standard CORE mandate. Only evaluated when `sepaMandateEnabled = true`.

`memberNumberStart`: starting value for the per-EEG member number auto-increment counter. The first member number assigned for this EEG will be this value. Defaults to `1` when not explicitly set.

### Response
- `204 No Content`

### Errors
- `400` missing `rc_number` or invalid JSON
- `403` not authorized for this EEG

---

## 6.12 Get API key status

### GET `/api/admin/settings/api-key?rc_number={rc_number}`

Returns whether an external API key exists for this EEG.

### Response 200
```json
{
  "active": true,
  "lastGeneratedAt": "2026-04-24T10:00:00Z"
}
```

`active: false` means no key exists or it has been revoked. The key value itself is never returned after initial generation.

---

## 6.13 Generate API key

### POST `/api/admin/settings/api-key?rc_number={rc_number}`

Generates a new API key (invalidates any existing key).

### Response 200
```json
{
  "apiKey": "moak_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
}
```

The key is shown exactly once. Store it securely — it cannot be retrieved again.

---

## 6.14 Revoke API key

### DELETE `/api/admin/settings/api-key?rc_number={rc_number}`

Revokes the API key. External integrations using this key will receive `401` immediately.

### Response
- `204 No Content`

---

## 6.15 Export application as Excel

### GET `/api/admin/applications/{id}/export/excel`

Generates and downloads an xlsx file for the given application in eegFaktura import format. Only available for applications in status `approved`, `imported`, or `import_failed`.

### Auth
Keycloak JWT. Tenant-admin access is checked against the application's RC number.

### Response
- `200 OK` — xlsx file
  - `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
  - `Content-Disposition: attachment; filename="{referenceNumber}.xlsx"`
- `404 Not Found` — application not found
- `403 Forbidden` — tenant mismatch
- `409 Conflict` — application not in exportable status
- `422 Unprocessable Entity` — application has no metering points

The file contains:
- Row 1: column headers (36 columns, A–AJ per eegFaktura import template)
- Row 2: importer marker `[### Leerzeile für Importer ###]`
- Rows 3+: one data row per metering point (member data repeated per row)

---

## 6.16 Download approval PDF

### GET `/api/admin/applications/{id}/approval-pdf`

Generates and downloads the Beitrittsbestätigung (approval confirmation) as a PDF file for the given application. Only available for applications in status `approved`, `imported`, or `import_failed`.

### Auth
Keycloak JWT. Tenant-admin access is checked against the application's RC number.

### Response
- `200 OK` — PDF file
  - `Content-Type: application/pdf`
  - `Content-Disposition: attachment; filename="beitrittsbestaetigung-{referenceNumber}.pdf"`
- `404 Not Found` — application not found
- `403 Forbidden` — tenant mismatch
- `409 Conflict` — application not in downloadable status

The PDF contains the same data as the approval PDF automatically emailed to the EEG on status change to `approved`:
- Header: title "Beitrittsbestätigung", EEG name, RC number, approval date, reference number
- Mitgliedsdaten: member number (if assigned), member type, name/company, birth date, address, email, phone
- Bankverbindung: IBAN, account holder, SEPA mandate type (Basislastschrift / Firmenlastschrift / Per E-Mail)
- Zählpunkte: table with metering point number, direction, participation factor
- Erteilte Zustimmungen: privacy acceptance (with version), accuracy confirmation, SEPA (checkbox or per-email note), document consents with dates
- Statusverlauf: table with status transitions (from → to) in German labels, timestamps, comments
- Weitere Angaben: configurable fields (if any are filled in)

---

## 6.17 Legal documents — Admin CRUD

Manages the list of EEG-specific legal documents shown in the public registration form.

---

### GET `/api/admin/legal-documents?rc_number={rc_number}`

Returns all legal documents for an EEG, ordered by `sortOrder`.

### Response 200
```json
[
  {
    "id": "3f8c8c2d-...",
    "rcNumber": "RC123456",
    "title": "Satzung der Energiegemeinschaft",
    "url": "https://example.at/satzung.pdf",
    "required": true,
    "sortOrder": 0,
    "createdAt": "2026-04-25T10:00:00Z",
    "updatedAt": "2026-04-25T10:00:00Z"
  }
]
```

Empty array when no documents are configured.

### Errors
- `400` missing `rc_number`
- `403` not authorized for this EEG

---

### POST `/api/admin/legal-documents?rc_number={rc_number}`

Creates a new legal document. Maximum 10 documents per EEG.

### Request
```json
{
  "title": "Satzung der Energiegemeinschaft",
  "url": "https://example.at/satzung.pdf",
  "required": true
}
```

Validation: `title` required (max 500 chars), `url` required (max 2048 chars, must use `http`/`https` scheme).

### Response 201
The created document object (same shape as list item).

### Errors
- `400` validation error
- `403` not authorized
- `409` document limit (10) reached

---

### PUT `/api/admin/legal-documents/{id}`

Updates title, url, and required flag of an existing document.

### Request
```json
{
  "title": "Satzung (aktualisiert)",
  "url": "https://example.at/satzung-v2.pdf",
  "required": true
}
```

Same validation as create.

### Response
- `204 No Content`

### Errors
- `400` validation error
- `403` not authorized for the document's EEG
- `404` document not found

---

### DELETE `/api/admin/legal-documents/{id}`

Deletes a legal document. Existing consent snapshots in `document_consent` are not affected (no foreign key).

### Response
- `204 No Content`

### Errors
- `403` not authorized for the document's EEG
- `404` document not found

---

### PUT `/api/admin/legal-documents/reorder?rc_number={rc_number}`

Replaces the sort order for all documents of an EEG atomically. Send all document IDs in the desired order.

### Request
```json
{
  "ids": ["3f8c8c2d-...", "7a1b2c3d-..."]
}
```

All IDs must be valid UUIDs. IDs not belonging to the given `rc_number` are silently ignored.

### Response
- `204 No Content`

### Errors
- `400` missing `rc_number` or invalid UUID
- `403` not authorized

---

## 7. Error model

### Validation error
```json
{
  "code": "validation_error",
  "message": "validation failed",
  "fields": {
    "email": "must be a valid email address"
  }
}
```

### Forbidden
```json
{
  "code": "forbidden",
  "message": "user is not allowed to access this EEG"
}
```

### Not found
```json
{
  "code": "not_found",
  "message": "application not found"
}
```

### Conflict
```json
{
  "code": "conflict",
  "message": "status transition is not allowed"
}
```

### Unprocessable Entity
```json
{
  "code": "unprocessable_entity",
  "message": "application has no metering points"
}
```

## 8. External API

### Authentication

All endpoints under `/api/external` use API-key authentication — no Keycloak required.

```
Authorization: Bearer moak_<32-char-random-key>
```

The key is generated in the Admin Settings page and must be kept server-side only.

### Rate limits

- **Burst**: 10 requests / 60 seconds per key (in-memory, per pod)
- **Daily quota**: 200 submissions / day per key (UTC midnight reset, DB-backed)
- Exceeded: `429 Too Many Requests` with `Retry-After` header

### Error codes specific to external API

| HTTP | code | Meaning |
|------|------|---------|
| 401 | `unauthorized` | Missing, invalid, or revoked API key |
| 410 | `gone` | EEG is inactive |
| 422 | `validation_error` | Invalid or missing fields |
| 429 | `rate_limit_exceeded` | Burst limit exceeded |
| 429 | `quota_exceeded` | Daily quota exhausted |

## 8.1 Submit external application

### POST `/api/external/v1/applications`

Submit a member application from an external integration (e.g. operator's own website form).
The API key determines the EEG — no `rcNumber` in the body.

### Request

```json
{
  "memberType": "private",
  "firstname": "Josef",
  "lastname": "Muster",
  "email": "max.mustermann@example.org",
  "residentStreet": "Testgasse",
  "residentStreetNumber": "5",
  "residentZip": "8010",
  "residentCity": "Graz",
  "residentCountry": "AT",
  "iban": "AT61190430023457320",
  "accountHolder": "Josef Muster",
  "privacyAccepted": true,
  "sepaMandateAccepted": true,
  "meteringPoints": [
    { "meteringPoint": "AT0010000000000000001000000000001", "direction": "CONSUMPTION", "participationFactor": 100 }
  ]
}
```

### memberType values

Same as public API: `private` | `sole_proprietor` | `farmer` | `municipality` | `company` | `association`

### Required fields

`memberType`, `email`, `residentStreet`, `residentStreetNumber`, `residentZip`, `residentCity`,
`iban`, `accountHolder`, `privacyAccepted: true`,
`sepaMandateAccepted: true`, `meteringPoints` (min 1).

For `natural_person` types (`private`, `farmer`): `firstname` + `lastname` required.
For legal entity types (`municipality`, `company`, `association`, `sole_proprietor`): `companyName` required.
- `sole_proprietor` (PROJ-28, Kleinunternehmer): only `companyName` required; `firstname`, `lastname`, `birth_date`, `uid_number`, `register_number` are ignored if present.
- `company`: additionally `uidNumber` + `registerNumber` required.
- `association`: additionally `registerNumber` required.

Configurable fields follow the EEG's active `field_config` — identical rules to the public form.

### Response 201

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "referenceNumber": "REF-2026-0042"
}
```

### Effects

- Application created directly in `submitted` status (draft → submitted in one step)
- Confirmation email sent to the member
- SEPA mandate PDF attached if enabled for the EEG
- EEG notification email sent if `contact_email` is configured

## 8.2 Get API key status

### GET `/api/admin/settings/api-key?rc_number=...`

Requires Keycloak authentication (admin area).

### Response 200

```json
{
  "active": true,
  "lastGeneratedAt": "2026-04-24T10:30:00Z"
}
```

## 8.3 Generate API key

### POST `/api/admin/settings/api-key?rc_number=...`

Generates a new key. Any existing active key is immediately invalidated. The plaintext key
is returned **once only** — it is not stored and cannot be retrieved again.

### Response 201

```json
{
  "apiKey": "moak_Xy7kR2..."
}
```

## 8.4 Revoke API key

### DELETE `/api/admin/settings/api-key?rc_number=...`

Revokes the active key immediately. No new key is created. All integrations using this key
will receive `401` from this point onwards.

### Response 204

No body.
