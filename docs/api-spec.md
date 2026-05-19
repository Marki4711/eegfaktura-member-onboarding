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
Allowed values (12):
- `draft`
- `submitted`
- `email_confirmed` *(PROJ-31, only when the EEG opts in to e-mail confirmation)*
- `under_review`
- `needs_info`
- `approved`
- `rejected`
- `imported` *(transient — import service auto-routes immediately, see PROJ-46)*
- `import_failed`
- `awaiting_bank_confirmation` *(PROJ-46, b2b only, set automatically by import service)*
- `ready_for_activation` *(PROJ-46, set automatically by import service for non-b2b)*
- `activated` *(PROJ-46, strict end state)*

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
  "sepaMandateAtImport": false,
  "showCentralPolicy": true,
  "requireEmailConfirmation": false,
  "meteringPointPrefixConsumption": "AT00060010001",
  "meteringPointPrefixProduction": "AT00060010001",
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
  ],
  "cooperativeSharesEnabled": true,
  "cooperativeRequiredShares": 1,
  "cooperativeShareAmountCents": 10000
}
```

`fieldConfig` contains only explicitly configured fields. Missing fields fall back to system defaults (`hidden` for new fields, `optional` for `phone`, `birth_date`, `uid_number`, `bank_name`). Fields with admin state `admin_only` are returned as `"hidden"` — they are never shown to the member.

`introText` is `null` when no text is configured.

`sepaMandateEnabled` is `false` by default. When `true`, SEPA mandate checkboxes and PDF generation are activated.

`sepaMandateAtImport` (PROJ-48) is `false` by default. When `true`, the registration form shows an explanatory hint that the SEPA mandate will be sent later (at import time) with the Mitgliedsnummer printed as Mandatsreferenz, instead of being attached to the welcome mail. Only meaningful when `sepaMandateEnabled = true`.

`requireEmailConfirmation` (PROJ-31): when `true`, the registration success view shows the „Bitte E-Mail-Postfach prüfen"-Hinweis instead of the default „wird nun von unserem Team geprüft"-Text. Backend also gates the admin status transitions accordingly.

`meteringPointPrefixConsumption` / `meteringPointPrefixProduction` (PROJ-52): optional per-direction Zählpunkt-Prefix. `null` ⇒ no EEG-specific prefill (mask shows only „AT" as fixed). When set, the form prefills the Zählpunkt-Field on Richtung-Wechsel and auto-pads with leading zeros at onBlur to 33 characters total. Backend submit-validation enforces `HasPrefix` per direction (defense-in-depth). Format ist garantiert `^AT[0-9A-Z]{0,31}$` (DB CHECK + Service-Layer-Normalisierung).

`showCentralPolicy` controls whether the central operator privacy policy is included in `legalDocuments`. Defaults to `true`. When `false`, the central policy entry is omitted from the list even if env vars are set — intended for EEGs that configure their own privacy policy as a custom document.

`legalDocuments` contains the central privacy policy entry (`isCentralPolicy: true`) when `showCentralPolicy = true` and `CENTRAL_POLICY_URL` is set. EEG-specific documents precede it, ordered by `sortOrder`. The central policy is not stored in the database — it is configured via `CENTRAL_POLICY_TITLE` / `CENTRAL_POLICY_URL` env vars.

`cooperativeSharesEnabled` (PROJ-37): when `true`, the public form renders a "Genossenschaftsanteile" block. `cooperativeRequiredShares` (positive integer) is then the minimum the member must subscribe; `cooperativeShareAmountCents` (positive integer) is the price per share in cents. The total is computed client-side as `count × cooperativeShareAmountCents`. When `cooperativeSharesEnabled` is `false`, the two value fields are omitted from the response and the form skips the block.

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
  "titel": "Dr.",
  "titelNach": "BSc",
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
  "bankName": "Musterbank Linz",
  "sepaMandateAccepted": true,
  "meteringPoints": [
    {
      "meteringPoint": "AT0031000000000000000000990022105",
      "direction": "PRODUCTION",
      "participationFactor": 1.0,
      "transformer": "T1",
      "installationNumber": "12345",
      "installationName": "PV Dach",
      "addressStreet": "Werkstraße",
      "addressStreetNumber": "12",
      "addressZip": "4020",
      "addressCity": "Linz",
      "generationType": "pv",
      "batterySizeKwh": 10.5,
      "inverterManufacturer": "Fronius",
      "batteryControlAcceptable": true,
      "feedInForecast": 6000,
      "pvPowerKwp": 9.9,
      "feedInLimitPresent": true,
      "feedInLimitKw": 7.0
    },
    {
      "meteringPoint": "AT0031000000000000000000990022106",
      "direction": "CONSUMPTION",
      "participationFactor": 1.0,
      "consumptionPreviousYear": 4200,
      "consumptionForecast": 4000
    }
  ],
  "membershipStartDate": "2026-05-01",
  "personsInHousehold": 3,
  "heatPump": true,
  "electricVehicle": true,
  "electricVehicleCount": 1,
  "electricVehicleAnnualKm": 12000,
  "electricHotWater": null,
  "cooperativeSharesCount": 1,
  "networkOperatorAuthorization": true
}
```

All fields under `meteringPoints[].transformer/installationNumber/installationName/batterySizeKwh/inverterManufacturer/consumptionPreviousYear/consumptionForecast/feedInForecast/pvPowerKwp/feedInLimitPresent/feedInLimitKw` (PROJ-45 + PROJ-49) and the application-level household fields (`personsInHousehold`, `heatPump`, `electricVehicle`, `electricHotWater`, …) are optional by default. Whether they are required is determined by the EEG's `fieldConfig` (see 5.1). Fields not relevant to the current `memberType` are ignored.

`meteringPoints[].generationType` (PROJ-45) is required when `direction = PRODUCTION` (DB-CHECK enforces it); allowed values `pv`/`hydro`/`wind`/`biomass`. Server defaults to `pv` when missing on a PRODUCTION row. NULL is enforced for CONSUMPTION rows — service nullifies any submitted value.

`meteringPoints[].batterySizeKwh` and `meteringPoints[].inverterManufacturer` (PROJ-45) are only meaningful for PRODUCTION rows with `generationType = "pv"`. The service nulls them in all other cases.

`meteringPoints[].consumptionPreviousYear` and `meteringPoints[].consumptionForecast` (PROJ-49) are integer kWh values for CONSUMPTION rows. The service nulls them on PRODUCTION rows.

`meteringPoints[].feedInForecast` (PROJ-49) is the annual energy forecast in kWh/year for PRODUCTION rows (any `generationType`). The service nulls it on CONSUMPTION rows.

`meteringPoints[].pvPowerKwp`, `meteringPoints[].feedInLimitPresent`, and `meteringPoints[].feedInLimitKw` (PROJ-49) are only meaningful for PRODUCTION rows with `generationType = "pv"`. `feedInLimitPresent` is the member's "Einspeiselimit vorhanden?" toggle (some grid connections cap the feed-in power below the PV nameplate); `feedInLimitKw` is the maximum allowed feed-in in kW and is only stored when `feedInLimitPresent = true`. The service nulls each unfitting combination.

`meteringPoints[].batteryControlAcceptable` (PROJ-49 follow-up) is the member's "Speichersteuerung im Sinne der EEG vorstellbar?" answer. Only stored for PRODUCTION rows with `generationType = "pv"` AND when the member has provided at least one battery parameter (`batterySizeKwh` or `inverterManufacturer`). The service nulls the field in all other cases (including the unanswered "no battery" path).

`networkOperatorAuthorization` (PROJ-44) is the member's authorisation for the EEG to coordinate with the grid operator on their behalf. The configurable-field `network_operator_authorization` controls visibility — when the EEG sets it to `required`, the boolean must be `true` on submit (otherwise 400). When `hidden`, the server nulls/false-sets it. The auth timestamp `network_operator_authorization_at` is stamped automatically on the FALSE→TRUE flip.

`titelNach` (PROJ-39) is the optional academic title after the name (e.g. `BSc`, `MSc`). The existing `titel` field represents the title **before** the name. Both are independent.

`bankName` (PROJ-39) is the optional bank name. It used to be admin-only; with PROJ-39 the member can supply it directly on submit.

`meteringPoints[].addressStreet/addressStreetNumber/addressZip/addressCity` (PROJ-39): per-metering-point deviating address. All four fields are all-or-nothing — either all four omitted (the member's primary address is used) or all four supplied. Mixing yields HTTP 400.

`electricVehicleCount` and `electricVehicleAnnualKm` (PROJ-42): integer detail fields for the EV section. Both are only meaningful when `electricVehicle = true`. If `electricVehicle` is not actively `true` the server silently nulls both on save. When the EEG has configured `electric_vehicle_count` / `electric_vehicle_annual_km` as `required`, the required-check **only** fires when `electricVehicle = true` — applicants who answered EV=No are never asked for a count.

`cooperativeSharesCount` (PROJ-37) is required on the **submit** path (see 5.4) when the EEG has `cooperativeSharesEnabled = true` and must be `>= cooperativeRequiredShares`. On create it is optional — the public form populates it server-side at submit. The server silently ignores the value when the EEG has the feature disabled.

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
  "referenceNumber": "RC123456-2026-0001",
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
  "referenceNumber": "RC123456-2026-0001",
  "status": "draft",
  "updatedAt": "2026-04-18T12:30:00Z"
}
```

### Errors
- `400` validation error
- `404` application not found
- `409` status does not allow editing

---

## 5.3a Consent semantics (PROJ-36)

Two kinds of consent are recorded per application:

- **explicit** — member actively ticked a checkbox at submit. The frontend sends these in `consents[]` of the submit body. Stored with `consent_type='explicit'`.
- **informational** — non-required legal documents that were shown as info-links on the form. The frontend does NOT send these; the backend writes them at submit time from `legal_document` entries with `required=false`. Stored with `consent_type='informational'`.

Both types carry the same fields (`title`, `url`, `consentedAt`) and appear in `consents[]` of admin detail responses. The `consentType` field is the discriminator. Pre-PROJ-36 entries default to `explicit` via the DB column default.

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
- when the EEG has `cooperativeSharesEnabled = true` (PROJ-37): `cooperativeSharesCount` set and `>= cooperativeRequiredShares`

### Response 200
```json
{
  "id": "3f8c8c2d-....",
  "referenceNumber": "RC123456-2026-0001",
  "status": "submitted",
  "submittedAt": "2026-04-18T12:35:00Z"
}
```

### Effects
- `application.status = submitted`
- set `application.submitted_at`
- write entry in `status_log`
- when the EEG has `require_email_confirmation = true` (PROJ-31), a confirmation token is generated, hashed (SHA-256) into `email_confirmation_token_hash`, and the confirmation link is included in the welcome mail. The application stays at `submitted` until the member clicks the link (see 5.5).

### Errors
- `400` required fields missing
- `404` application not found
- `409` application already submitted or in a disallowed status

---

## 5.5 Confirm e-mail (PROJ-31)

### POST `/api/public/applications/confirm-email`

Consumes the single-use token sent in the welcome mail. On success, transitions the application from `submitted` → `email_confirmed` and unblocks admin review. The token travels in the request body (not the URL path) so it stays out of server access logs; the frontend page reads it from a URL fragment and posts it here.

### Request
```json
{ "token": "<32-byte url-safe random string>" }
```

### Response 200
```json
{
  "eegName": "Muster Energiegemeinschaft",
  "eegContactEmail": "kontakt@beispiel-eeg.at",
  "alreadyConfirmed": false
}
```

`alreadyConfirmed` is `true` when the same token is presented again after the first successful consumption — the page renders "Bereits bestätigt" instead of an error so the user doesn't worry about a broken link.

### Errors
- `400` token missing, malformed, expired (>30 days old), or no matching application
- All other failure modes are rendered as `400` with the generic German message "Der Bestätigungs-Link ist ungültig oder abgelaufen" — token enumeration is not possible from the response.

### Side effects on success
- `application.email_confirmed_at = NOW()` and `application.email_confirmation_used_at = NOW()` (first click only)
- `application.status` → `email_confirmed`
- entry in `status_log`

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
      "referenceNumber": "RC123456-2026-0001",
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
  "referenceNumber": "RC123456-2026-0001",
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
  ],
  "cooperativeSharesCount": 1,
  "cooperativeSharesEnabled": true,
  "cooperativeRequiredShares": 1,
  "cooperativeShareAmountCents": 10000
}
```

`consents` contains the immutable snapshots of legal document consents recorded at submission time. Empty array when no consents were submitted.

`cooperativeSharesCount` (PROJ-37) is the number of shares the member subscribed; null when the EEG hasn't enabled the feature or the application predates its activation. The three accompanying fields (`cooperativeSharesEnabled`, `cooperativeRequiredShares`, `cooperativeShareAmountCents`) mirror the **current** EEG-level settings and are joined in at detail-build time so the admin UI can compute `count × amountCents = total` without a parallel `/settings/eeg` round-trip. When the feature is disabled, `cooperativeSharesEnabled` is `false` and the two value fields are omitted.

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
  "iban": "AT611904300234573201",
  "accountHolder": "Max Muster",
  "bankName": "Musterbank Linz",
  "einzugsart": "core",
  "mandateReference": "RC123456-2026-0001",
  "mandateDate": "2026-05-17",
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
- `einzugsart` accepts `core` | `b2b` | `kein_sepa` (PROJ-48 — admin-controlled per application, no longer auto-derived from `memberType`)
- additional editable fields mirror the public submit body: `memberType`, `titel`, `titelNach`, `companyName`, `uidNumber`, `registerNumber`, plus the configurable energy/household fields and PROJ-37 `cooperativeSharesCount`

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
- `submitted -> rejected` *(PROJ-31, admin override for obvious junk before e-mail confirmation)*
- `email_confirmed -> under_review` *(PROJ-31)*
- `email_confirmed -> needs_info`
- `email_confirmed -> approved`
- `email_confirmed -> rejected`
- `under_review -> needs_info`
- `under_review -> approved`
- `under_review -> rejected`
- `needs_info -> submitted`
- `approved -> imported`
- `approved -> import_failed`
- `import_failed -> approved`
- `awaiting_bank_confirmation -> ready_for_activation` *(PROJ-46, admin manuell nach Bank-Bestätigung)*
- `awaiting_bank_confirmation -> under_review` *(PROJ-46, admin rückwärts)*
- `ready_for_activation -> activated` *(PROJ-46, admin manuell; auch via Batch-Endpoint, siehe 6.5.6)*
- `ready_for_activation -> under_review` *(PROJ-46, admin rückwärts)*

Reachable only via dedicated endpoints (NOT via this generic `/status` route):
- `submitted -> email_confirmed` — via member click on `POST /api/public/applications/confirm-email`
- `imported -> awaiting_bank_confirmation` / `imported -> ready_for_activation` — auto-transition by import service (Branch on `einzugsart`), see 6.5
- `imported|awaiting_bank_confirmation|ready_for_activation -> approved` — via `POST /api/admin/applications/{id}/reset-import` (PROJ-30 + PROJ-46, see 6.5.3). NOT possible from `activated` (strict end state).

When `registration_entrypoint.require_email_confirmation = TRUE` (PROJ-31), this endpoint rejects `submitted -> under_review|needs_info|approved` with HTTP 409 until the member has clicked the confirmation link. `submitted -> rejected` remains available as the admin's anti-spam override.

`activated` has **no transitions out** — deactivation must happen in the eegFaktura core directly (PROJ-46 Entscheidung A).

### Side effects
- on `approved`: set `approved_at`, set `reviewed_by_user_id`. **Since PROJ-46 Stage B no PDF/mail is generated here** — Beitrittsbestätigungs-PDF + Member/EEG mails are now generated/sent at import time (see 6.5), when the member number exists. The legacy `SendApprovalEmail` method and its template `application_approved_eeg.html` have been **removed** from the codebase.
- on `rejected`: set `rejected_at`, set `reviewed_by_user_id`, **PROJ-41:** synchroner Mail-Versand an Mitglied mit `reason` 1:1 im Body
- on `needs_info`: set `needs_info_reason`, **PROJ-43:** synchroner Mail-Versand an Mitglied mit `reason` 1:1 im Body
- on `ready_for_activation` (when coming from `awaiting_bank_confirmation`): set `bank_confirmed_at` *(PROJ-46)*
- on `activated`: set `activated_at`, async welcome mail an Mitglied *(PROJ-46)*
- always write entry in `status_log`

Die Mitglieder-Mails bei `rejected` und `needs_info` werden **synchron vor dem Commit** versendet (hard-fail). Schlägt der SMTP-Versand fehl, wird die Statusänderung zurückgerollt und der Aufruf antwortet mit HTTP 500 + Mail-Fehlermeldung — der Admin sieht das Problem direkt in der UI. Alle anderen Mails sind best-effort async.

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
  "status": "awaiting_bank_confirmation",
  "targetParticipantId": "4711",
  "memberTariffWarning": "core returned HTTP 404"
}
```

`status` reflects the final post-import status (PROJ-46): `awaiting_bank_confirmation` for `einzugsart=b2b`, otherwise `ready_for_activation`. `imported` is a transient intermediate state and is rarely seen — it remains only if the auto-followup transition fails (the admin can then reset via `/reset-import`).

`memberTariffWarning` (PROJ-27) is only present when the participant was created successfully but the follow-up call to set the member-level tariff failed. The application is still moved out of `approved` — meter tariffs are persisted; the admin needs to set the member tariff manually in the core.

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
- set `member_number`
- `status = imported` (transient), then **auto-transition** to either
  `awaiting_bank_confirmation` (b2b) or `ready_for_activation` (non-b2b)
  in a separate transaction (PROJ-46)
- write 1 or 2 entries in `status_log` (one for `→ imported`, one for the
  auto-followup)
- **PROJ-46 Stage B + PROJ-47**: best-effort async fan-out — generates the
  Beitrittsbestätigungs-PDF (mit Mitgliedsnummer), sends it to the member
  + EEG-Contact-Copy; for `einzugsart=b2b` adds a second attachment
  (Firmenlastschrift-Mandat-PDF mit Mandatsreferenz=Mitgliedsnummer)

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

## 6.5.1 Mark imported manually (PROJ-34)

### POST `/api/admin/applications/{id}/mark-imported-manually`

Recovery for the "stuck in-flight" scenario where the core created the participant but the onboarding bookkeeping failed. The admin reads the participant UUID + member-number from eegFaktura and submits them — the application then transitions from `approved` (with in-flight slot) to `imported`. Refused when the application is not in the stuck state (status≠approved, no in-flight slot, or in-flight younger than 2 minutes).

### Request body
```json
{
  "targetParticipantId": "0aeab3ff-4fcd-11f1-98e4-bed36ef4f0db",
  "memberNumber": "A006",
  "reason": "Manueller Recovery — DB-Unique-Verletzung"
}
```

`targetParticipantId` and `memberNumber` are mandatory. `reason` is optional (an audit-trail tag is added automatically).

### Response 200
Full `AdminApplicationDetailResponse` with the updated status.

### Errors
- `400` validation failed (missing UUID, empty memberNumber)
- `403` not authorized for this EEG
- `404` application not found
- `409` application is not in a stuck import state

---

## 6.5.2 Clear import lock (PROJ-34)

### POST `/api/admin/applications/{id}/clear-import-lock`

Releases the in-flight slot on a stuck application without changing its status. Allows the admin to retry the import, with the explicit risk of producing a duplicate participant in the core if the original attempt had already inserted there. The previous `target_participant_id` is preserved in the status_log entry.

### Request body
```json
{
  "reason": "Im Core kein Teilnehmer vorhanden — neu importieren"
}
```

`reason` is mandatory (min 5 chars).

### Response 200
Full `AdminApplicationDetailResponse` with cleared import bookkeeping (status remains `approved`).

### Errors
- `400` validation failed (reason missing/too short)
- `403` not authorized for this EEG
- `404` application not found
- `409` application is not in a stuck import state

---

## 6.5.3 Reset import (PROJ-30 + PROJ-46 extension)

### POST `/api/admin/applications/{id}/reset-import`

Transitions an application from a post-import status back to `approved` so
it can be re-imported after the eegFaktura admin deleted the participant
in the core. No call to the core — the admin verifies the deletion
manually.

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
- Application must be in status `imported`, `awaiting_bank_confirmation`,
  or `ready_for_activation` (PROJ-46 expansion). NOT `activated` —
  active members must be deactivated in the Core first (otherwise 409).
- The transitions `imported → approved`, `awaiting_bank_confirmation → approved`,
  `ready_for_activation → approved` are **only** reachable via this
  endpoint; the generic `POST /status` does not accept them.
- Tenant-Admin scope: must match the EEG of the application.

### Response 200
Returns the full `AdminApplicationDetail` after the reset (status now
`approved`, `targetParticipantId` + `memberNumber` cleared).

### Side effects
- `status = approved`
- `import_started_at = NULL`
- `import_finished_at = NULL`
- `imported_at = NULL`
- `target_participant_id = NULL`
- `import_error_message = NULL`
- `member_number = NULL` — assigned at import time (PROJ-27); cleared so the
  next re-import gets a fresh suggestion from the core's max+1 and doesn't
  show a stale assignment in the admin detail view
- `bank_confirmed_at = NULL`, `activated_at = NULL` — *(PROJ-46)* cleared
  so a re-import starts from a clean slate (b2b will need fresh bank-
  confirmation, activation will be re-evaluated)
- write `status_log` entry with `from=<current status>`, `to='approved'`,
  `reason = <user reason>\n[system] previous target_participant_id=<uuid>\n[system] previous member_number=<x>`
  (the old participant UUID and member number are archived in the log so
  the audit trail preserves them after the columns are cleared)

### Failure responses
- `400` reason missing / too short / too long
- `403` tenant mismatch
- `409` application not in a resetable status (`activated` rejects here)

---

## 6.5.6 Activation check (PROJ-46 Stage D)

### POST `/api/admin/applications/check-activation`

Admin-triggered batch check that asks the eegFaktura core which of our
`ready_for_activation` applications are now `ACTIVE` there, and transitions
matching rows to `activated`. Replaces the originally planned cron-polling
(user decision: admin-triggered keeps SOAP cost and surprise factor low).

### Request
No body. Tenant scope is derived from the admin's JWT (`tenant`-claim) —
superusers operate on all EEGs, tenant admins on their own RC numbers only.

### Response 200
```json
{
  "checked":   12,
  "activated":  3,
  "errors":   ["tenant RC0001: core returned HTTP 503"]
}
```

| Field | Meaning |
|---|---|
| `checked` | Number of applications inspected (in scope: `status = ready_for_activation` AND tenant in admin's allowed list) |
| `activated` | Number transitioned from `ready_for_activation` to `activated` |
| `errors` | Per-tenant or per-application errors (empty/omitted on full success) |

### Algorithm
1. List all `ready_for_activation` applications for the admin's tenants.
2. Group by `rc_number`.
3. Per tenant: call `GET /participant` (core) — bounded by 4 MiB / ~2000 participants per call.
4. Build an in-memory index `target_participant_id → core.status`.
5. For each candidate: evaluate the per-EEG `activation_mode`
   (PROJ-53). If the criterion is met, transition to `activated`
   via guarded `UpdateStatusAdminTx`, stamp `activated_at = NOW()`, write
   `status_log` entry with actor `system:activation-check`. The transition
   asynchronously triggers `SendActivationNotification` (full
   Beitrittsbestätigungs-Mail with PDF), idempotent via
   `activation_notification_sent_at` flag.
6. Best-effort — per-tenant errors don't abort the whole batch.

### Activation modes (PROJ-53)

The per-EEG column `registration_entrypoint.activation_mode` selects the
criterion. Default `participant_active` keeps the historical behaviour.

| Mode | Criterion |
|---|---|
| `participant_active` (Default) | Core participant's top-level `status == ACTIVE` |
| `any_meter_registration_started` | At least one of the participant's `meters[].processState` is in `{PENDING, APPROVED, ACTIVE}` — i.e. the network operator has at least acknowledged the EDA online-registration request |

EDA-state mapping (verified 2026-05-19 against a live tenant):
`ANFORDERUNG_ECON` keeps `processState = INVALID`,
`ANTWORT_ECON` → `PENDING`, `ZUSTIMMUNG_ECON` → `APPROVED`,
`ABSCHLUSS_ECON` → `ACTIVE`.

### POST `/api/admin/applications/{id}/mark-activated`

PROJ-53: manual `approved → activated` skip-import. The application is
moved directly to `activated` without calling the eegFaktura core. Use
only when the member already exists in the core (Faktura cannot delete
members) and was manually overwritten there with the onboarding data.
Triggers the same Beitrittsbestätigungs-Mail-with-PDF as the regular
activation path.

#### Auth
Bearer JWT, tenant must include the application's `rc_number` or
superuser flag.

#### Request body
```json
{
  "memberNumber": "0042"
}
```

| Field | Type | Required | Note |
|---|---|---|---|
| `memberNumber` | string | yes | Persisted to `application.member_number`. Used as `Mitgliedsnummer` on the Beitrittsbestätigung; must be unique within the EEG (collides ⇒ 409). |

#### Responses
| Status | Body | Meaning |
|---|---|---|
| `200 OK` | `AdminApplicationDetailResponse` | Transition succeeded; mail dispatch is async |
| `400 Bad Request` | `ErrorResponse` | `memberNumber` missing / empty |
| `403 Forbidden` | `ErrorResponse` | Wrong tenant |
| `404 Not Found` | `ErrorResponse` | Application doesn't exist |
| `409 Conflict` | `ErrorResponse` | Application is not in `approved` status, or `memberNumber` already used by another application in the same EEG |

### Failure responses
- `503` Core integration not configured (`CORE_BASE_URL` empty) or no admin
  bearer token present (dev mode without Keycloak)
- `500` Unexpected error during batch run (rare — per-tenant failures are
  collected in `errors`)

---

## 6.5.5 Reassign to a different EEG (PROJ-40)

### POST `/api/admin/applications/{id}/reassign-eeg`

Moves an application from its current EEG to a different EEG during admin review (e.g. member clicked the wrong RC link). Reference number is regenerated on the target EEG's per-year counter; the old `rc_number` + old `reference_number` are archived in the status_log audit trail.

### Request

```json
{
  "targetRcNumber": "RC123456",
  "reason": "Adresse liegt im Versorgungsgebiet der Ziel-EEG"
}
```

| Field | Required | Constraints |
|---|---|---|
| `targetRcNumber` | yes | 1–50 chars, normalized to uppercase |
| `reason` | yes | 5–500 chars |

### Rules

- Application status must be one of `submitted`, `email_confirmed`, `under_review`, `needs_info`. Anything else → 409.
- Admin must be authorized for **both** source RC and target RC (or be a superuser).
- Target RC must exist in `registration_entrypoint` and have `is_active = true`.
- Source = target → 409.
- Cooperative-shares, field-config, email-confirmation settings are **not** re-validated on reassign — admin uses `needs_info` if anything needs to be sorted out post-move.

### Response 200

Returns the full `AdminApplicationDetail` with `rcNumber` set to the new EEG and `referenceNumber` regenerated on the target counter.

### Side effects

- `rc_number = <new>`, `reference_number = <new ref>` via the target's PROJ-35 counter
- New `status_log` entry: `from_status == to_status` (status unchanged), reason = `<user reason>\n[system] previous rc_number=<old>\n[system] previous reference_number=<old>`
- **No** member-facing mail (V1)

### Failure responses

- `400` validation (reason too short, targetRcNumber missing)
- `403` admin not authorized for source or target
- `404` application or targetRcNumber not found
- `409` status not reassignable / source == target / target not active

---

## 6.5.4 Resend e-mail confirmation (PROJ-31)

### POST `/api/admin/applications/{id}/resend-email-confirmation`

Rotates the e-mail confirmation token (old token is invalidated) and resends the welcome mail to the member. Useful when the original mail was lost. Available only while the application is in status `submitted` with a still-pending confirmation; refused once the member has already confirmed.

### Request
No body.

### Response 200
```json
{ "ok": true }
```

### Rules
- application must be in `submitted` and not yet confirmed
- EEG must have `require_email_confirmation = true` (otherwise no token is needed)
- the new token's lifetime is reset to 30 days from now
- the previous token is invalidated immediately (single-use guarantee)
- a per-application throttle prevents resend abuse

### Errors
- `403` tenant mismatch
- `404` application not found
- `409` application already confirmed, or EEG does not require confirmation, or throttle hit

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

Allowed field names: `phone`, `birth_date`, `uid_number`, `bank_name`, `membership_start_date`, `persons_in_household`, `consumption_previous_year`, `consumption_forecast`, `feed_in_forecast`, `pv_power_kwp`, `inverter_power_kw`, `heat_pump`, `electric_vehicle`, `electric_vehicle_count`, `electric_vehicle_annual_km`, `electric_hot_water`, `network_operator_authorization` *(PROJ-44)*, `transformer`, `installation_number`, `installation_name`, `battery_size_kwh` *(PROJ-45)*, `inverter_manufacturer` *(PROJ-45)*

**Type-conditional visibility (PROJ-45):** the admin UI shows badges next to each conditional field — `[Verbraucher]` (only renders when the application has ≥1 CONSUMPTION metering point), `[Einspeisung]` (≥1 PRODUCTION), `[PV]` (additionally requires `generation_type='pv'` on the MP), `[+E-Auto]` (additionally requires `electric_vehicle=true`). Backend mirrors the gate: the required-check on these fields only fires when the matching MP-type / EV flag is present.

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

Returns the EEG settings — the eight Core-mastered fields (PROJ-32) plus the onboarding-only toggles.

### Response 200
```json
{
  "rcNumber": "RC123456",
  "eegId": "AT0040000000RC123456000000000000",
  "eegName": "Muster Energiegemeinschaft",
  "eegStreet": "Hauptstraße",
  "eegStreetNumber": "12",
  "eegZip": "4020",
  "eegCity": "Linz",
  "creditorId": "AT28ZZZ00000000000",
  "contactEmail": "kontakt@beispiel-eeg.at",
  "lastSyncedFromCoreAt": "2026-05-14T16:58:58.750289Z",
  "registrationActive": true,
  "sepaMandateEnabled": true,
  "useCompanySEPAMandate": false,
  "sepaMandateAtImport": false,
  "showCentralPolicy": true,
  "memberNumberStart": 1,
  "requireEmailConfirmation": false,
  "meteringPointPrefixConsumption": "AT00060010001",
  "meteringPointPrefixProduction": null,
  "activationMode": "participant_active",
  "cooperativeSharesEnabled": true,
  "cooperativeRequiredShares": 1,
  "cooperativeShareAmountCents": 10000
}
```

**Core-mastered fields** (PROJ-32, read-only — only modified via `/sync` below): `eegId`, `eegName`, `eegStreet`, `eegStreetNumber`, `eegZip`, `eegCity`, `creditorId`, `contactEmail`. `lastSyncedFromCoreAt` is `null` until the first successful sync.

`registrationActive` is `false` by default. `sepaMandateEnabled`, `useCompanySEPAMandate`, and `sepaMandateAtImport` (PROJ-48) default to `false`. `showCentralPolicy` defaults to `true`. `memberNumberStart` defaults to `1`. `requireEmailConfirmation` (PROJ-31) defaults to `false`.

`cooperativeSharesEnabled` (PROJ-37) defaults to `false`. When `true`, both `cooperativeRequiredShares` (positive integer, minimum mandatory shares per member) and `cooperativeShareAmountCents` (positive integer, price per share in cents) are returned. When `false`, those two value fields are omitted.

### Errors
- `400` missing `rc_number`
- `403` not authorized for this EEG

---

## 6.11 Save EEG settings

### PUT `/api/admin/settings/eeg?rc_number={rc_number}`

Writes the onboarding-only editable fields. The Core-mastered fields (`eegId`, `eegName`, address, `creditorId`, `contactEmail`) are **not** accepted in the request body — they are silently ignored (no 400) so a legacy client continues to work. To change those, use the sync endpoint (6.11b).

### Request body
```json
{
  "registrationActive": true,
  "sepaMandateEnabled": true,
  "useCompanySEPAMandate": false,
  "sepaMandateAtImport": false,
  "showCentralPolicy": true,
  "memberNumberStart": 1,
  "requireEmailConfirmation": false,
  "meteringPointPrefixesPresent": true,
  "meteringPointPrefixConsumption": "AT00060010001",
  "meteringPointPrefixProduction": null,
  "activationMode": "any_meter_registration_started",
  "cooperativeSharesEnabled": true,
  "cooperativeRequiredShares": 1,
  "cooperativeShareAmountCents": 10000
}
```

`activationMode` (PROJ-53) is optional in the request — `null`/omitted leaves the existing value unchanged (patch semantics). Allowed values: `participant_active` (default for new EEGs — Core-Teilnehmer-Status `ACTIVE` löst Activation aus) or `any_meter_registration_started` (mind. ein Zählpunkt mit `processState ∈ {PENDING, APPROVED, ACTIVE}`). Invalid values return `400`.

`registrationActive`: enables or disables the public registration form for this EEG. When `false`, `GET /api/public/registration/{rc_number}` returns `410 Gone`.

`showCentralPolicy`: when `false`, the central operator privacy policy is not shown in the public registration form. Intended for EEGs that have configured their own privacy policy as a custom document (see 6.16).

`useCompanySEPAMandate`: when `true`, the EEG opts in to the SEPA B2B (Firmenlastschrift) mandate variant. The mandate variant per application is **not** auto-derived from `memberType` — it is set by the admin via the application's `einzugsart` field (`core` | `b2b` | `kein_sepa`, default `core`). Only evaluated when `sepaMandateEnabled = true`. (PROJ-48 removed the previous auto-mapping `company|association → b2b`.)

`sepaMandateAtImport` (PROJ-48): when `true`, SEPA mandate PDFs are generated **at import time** (with the assigned Mitgliedsnummer printed as Mandatsreferenz) rather than at submit time (without reference). Use when the EEG runs a digital signature workflow on the mandate — a signed PDF cannot be modified afterwards, so the reference must be present before signing. When `false` (default), mandates are generated at submit time with a `<Mitgliedsnummer wird beim Import vergeben>` placeholder. Independent of `useCompanySEPAMandate`. Only evaluated when `sepaMandateEnabled = true`.

`memberNumberStart`: starting value for the per-EEG member number auto-increment counter. Defaults to `1` when not explicitly set.

`requireEmailConfirmation` (PROJ-31): when `true`, members must click the confirmation link in the welcome mail before the application becomes reviewable. While pending, the admin `/status` endpoint rejects `submitted → under_review|needs_info|approved` with 409.

`meteringPointPrefixesPresent` + `meteringPointPrefixConsumption` + `meteringPointPrefixProduction` (PROJ-52): **Patch-Semantik** — die zwei Prefix-Spalten werden nur dann persistiert, wenn `meteringPointPrefixesPresent: true` mitgeschickt wird. Sonst lässt der Handler die Spalten unberührt (sodass andere Editoren, die `saveEEGSettings` ohne Prefix-Felder aufrufen, keine bestehenden Werte clobbern). Beim Save: leerer String oder `null` ⇒ Backend cleart die Spalte, Wert ⇒ Backend normalisiert (Whitespace + Dots + Hyphens entfernen, uppercase) und validiert gegen `^AT[0-9A-Z]{0,31}$`. Bei Validierungsfehler 400 mit Field-Level-Message.

`cooperativeSharesEnabled` (PROJ-37): when `true`, the registration form renders a "Genossenschaftsanteile" block with a mandatory share count input; `cooperativeRequiredShares` (≥1) is the minimum, `cooperativeShareAmountCents` (>0) is the price per share. **Both must be present and positive when `cooperativeSharesEnabled=true`** — otherwise the request fails with a 400 carrying field-level error messages. When `false`, both value fields are server-side reset to `null` (cleanup). Config changes apply prospectively only — existing applications keep their stored count even if it now falls below the new minimum.

### Response
- `204 No Content`

### Errors
- `400` missing `rc_number` or invalid JSON
- `403` not authorized for this EEG

---

## 6.11a Compare EEG settings with core (PROJ-32)

### GET `/api/admin/settings/eeg/core-comparison?rc_number={rc_number}`

Fetches the current EEG master data from the eegFaktura core (forwarding the admin's bearer token) and diffs it against the locally stored values. Used by the settings page to render the drift banner. Memoised per RC for 30 s — repeated page-opens within that window share one core call.

### Response 200 (synchron)
```json
{
  "coreReachable": true,
  "inSync": true,
  "differingFields": [],
  "lastSyncedAt": "2026-05-14T16:58:58.750289Z"
}
```

### Response 200 (Drift)
```json
{
  "coreReachable": true,
  "inSync": false,
  "differingFields": [
    {"field": "eegName", "label": "EEG-Name", "localValue": "Alt", "coreValue": "Neu"}
  ],
  "lastSyncedAt": "2026-05-14T16:58:58.750289Z"
}
```

### Response 200 (Core nicht erreichbar)
```json
{
  "coreReachable": false,
  "coreUnreachableError": "core service timeout",
  "lastSyncedAt": "2026-05-14T16:58:58.750289Z"
}
```

Failure modes are returned as `200` so the UI can render the appropriate banner state without treating it as an error toast.

### Errors
- `400` missing `rc_number`
- `403` not authorized for this EEG
- `503` `CORE_BASE_URL` is not configured, or no bearer token in the request (admin session expired)

---

## 6.11b Sync EEG settings from core (PROJ-32)

### POST `/api/admin/settings/eeg/sync?rc_number={rc_number}`

Pulls the current EEG master data from the eegFaktura core and overwrites the eight synced fields on `registration_entrypoint`. Stamps `last_synced_from_core_at = NOW()`. Returns the same shape as `/core-comparison` so the frontend can re-render without an extra round-trip.

### Response 200
Same shape as `/core-comparison`. With `inSync: true` and the freshly stamped `lastSyncedAt`. PROJ-33: two additional fields cover the logo sync:
- `logoSyncedAt` — timestamp of the last successful logo fetch (NULL until the first one). Mirrors `registration_entrypoint.eeg_logo_synced_at`.
- `logoSyncWarning` — only set when the master-data sync succeeded but the follow-up logo fetch did not. Examples: "Logo überschreitet 256 KB — bitte in eegFaktura ein kleineres hinterlegen", "Logo-Format wird nicht unterstützt (nur PNG, JPEG, GIF)". The frontend renders this under the logo preview as an orange hint.

### Errors
- `400` missing `rc_number`
- `403` not authorized for this EEG
- `502` core returned an error on the master-data step (auth, schema mismatch, …) — message in `code: "core_unreachable"` body
- `503` `CORE_BASE_URL` is not configured, or no bearer token in the request

Logo-step failures do NOT produce a 502 — they become `logoSyncWarning` on a 200 response (best-effort semantics).

---

## 6.11c Get EEG logo (PROJ-33)

### GET `/api/admin/settings/eeg/logo?rc_number={rc_number}`

Returns the bytes of the EEG logo cached during the last successful sync, with the original `Content-Type` (PNG / JPEG / GIF). Intended for inline preview in the admin UI — the frontend fetches via JS (to supply the Bearer header) and renders the bytes through an Object URL.

### Response 200
- Body: raw image bytes
- `Content-Type`: `image/png` | `image/jpeg` | `image/gif`
- `Cache-Control: private, max-age=300`

### Errors
- `400` missing `rc_number`
- `403` not authorized for this EEG
- `404` no logo synced yet — the EEG either hasn't synced from the core, or the core has no logo configured. Body is `{"code":"not_found","message":"Noch kein Logo aus eegFaktura geladen"}`.

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

Generates and downloads the Beitrittsbestätigung (approval confirmation) as a PDF file for the given application. Available for applications in status `approved`, `imported`, `import_failed`, `awaiting_bank_confirmation`, `ready_for_activation`, or `activated`.

### Auth
Keycloak JWT. Tenant-admin access is checked against the application's RC number.

### Response
- `200 OK` — PDF file
  - `Content-Type: application/pdf`
  - `Content-Disposition: attachment; filename="beitrittsbestaetigung-{referenceNumber}.pdf"`
- `404 Not Found` — application not found
- `403 Forbidden` — tenant mismatch
- `409 Conflict` — application not in downloadable status

The PDF is identical to the one auto-attached to the member/EEG mails at import time (PROJ-46 Stage B). Up to PROJ-46 Stage B this PDF was emailed to the EEG on `→ approved`; that auto-send is gone and the PDF generation is now anchored to the import step (so the member number is available for the SEPA mandate reference). For B2B applicants the import mail additionally contains a separate Firmenlastschrift-Mandat-PDF with embedded Mandatsreferenz=Mitgliedsnummer (PROJ-47); that mandate PDF is currently **not** downloadable via this endpoint.

Contents:
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
