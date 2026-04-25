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
  "title": "Become a member",
  "active": true,
  "fieldConfig": {
    "phone": "optional",
    "birth_date": "optional",
    "heat_pump": "required",
    "transformer": "hidden"
  }
}
```

`fieldConfig` contains only explicitly configured fields. Missing fields fall back to system defaults (`hidden` for new fields, `optional` for `phone`, `birth_date`, `uid_number`). The frontend uses this to show/hide/require fields dynamically. Fields with admin state `admin_only` are returned as `"hidden"` here — they are never shown to the member.

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
empty

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
  ]
}
```

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
  "targetParticipantId": "4711"
}
```

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
  "sepaMandateEnabled": true,
  "useCompanySEPAMandate": false
}
```

All address/name fields are `null` when not yet configured. `sepaMandateEnabled` defaults to `false`. `useCompanySEPAMandate` defaults to `false`.

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
  "sepaMandateEnabled": true,
  "useCompanySEPAMandate": false
}
```

`useCompanySEPAMandate`: when `true`, members of type `company` or `association` receive the SEPA B2B mandate PDF instead of the standard CORE mandate. Only evaluated when `sepaMandateEnabled = true`.

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

## 6.16 Public registration config — introText field

`GET /api/public/registration/{rc_number}` includes `introText` in the response:

```json
{
  "rcNumber": "RC123456",
  "title": "Mitglied werden",
  "active": true,
  "fieldConfig": { ... },
  "introText": "<p>Willkommen!</p>"
}
```

`introText` is `null` when no text is configured. The frontend displays a default text in that case.

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

Same as public API: `private` | `farmer` | `municipality` | `company` | `association`

### Required fields

`memberType`, `email`, `residentStreet`, `residentStreetNumber`, `residentZip`, `residentCity`,
`iban`, `accountHolder`, `privacyAccepted: true`,
`sepaMandateAccepted: true`, `meteringPoints` (min 1).

For `natural_person` types (`private`, `farmer`): `firstname` + `lastname` required.
For legal entity types: `companyName` required.

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
