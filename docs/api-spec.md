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
Allowed values (11):
- `draft`
- `submitted`
- `email_confirmed` *(PROJ-31, only when the EEG opts in to e-mail confirmation)*
- `under_review`
- `needs_info`
- `approved`
- `rejected`
- `imported` *(transient — import service auto-routes immediately, see PROJ-46)*
- `import_failed`
- `ready_for_activation` *(PROJ-46, set automatically by import service for all einzugsarten since PROJ-91)*
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
  "cooperativeShareAmountCents": 10000,
  "brandPreset": "leaf",
  "eegName": "Muster-EEG",
  "logoDataUri": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA…"
}
```

`fieldConfig` contains only explicitly configured fields. Missing fields fall back to system defaults (`hidden` for new fields, `optional` for `phone`, `birth_date`, `uid_number`, `bank_name`). Fields with admin state `admin_only` are returned as `"hidden"` — they are never shown to the member.

`introText` is `null` when no text is configured.

`sepaMandateAtImport` (PROJ-48) is `false` by default. When `true`, the registration form shows an explanatory hint that the SEPA mandate will be sent later (at import time) with the Mitgliedsnummer printed as Mandatsreferenz, instead of being attached to the welcome mail. PROJ-80 (2026-06-08) removed the `sepaMandateEnabled` toggle — the SEPA-Mandate-PDF is now always generated for SEPA members; the variant (audit-trail vs. signature field) is steered exclusively by `sepaMandateCoreAuditEnabled` resp. `sepaMandateB2BAuditEnabled`. Cross-Field-Coupling: when `sepaMandateCoreAuditEnabled = true`, this toggle is forced to `true` (the audit-trail PDF needs the Mitgliedsnummer as Mandatsreferenz to be complete).

`requireEmailConfirmation` (PROJ-31): when `true`, the registration success view shows the „Bitte E-Mail-Postfach prüfen"-Hinweis instead of the default „wird nun von unserem Team geprüft"-Text. Backend also gates the admin status transitions accordingly.

`meteringPointPrefixConsumption` / `meteringPointPrefixProduction` (PROJ-52): optional per-direction Zählpunkt-Prefix. `null` ⇒ no EEG-specific prefill (mask shows only „AT" as fixed). When set, the form prefills the Zählpunkt-Field on Richtung-Wechsel and auto-pads with leading zeros at onBlur to 33 characters total. Backend submit-validation enforces `HasPrefix` per direction (defense-in-depth). Format ist garantiert `^AT[0-9A-Z]{0,31}$` (DB CHECK + Service-Layer-Normalisierung).

`showCentralPolicy` controls whether the central operator privacy policy is included in `legalDocuments`. Defaults to `true`. When `false`, the central policy entry is omitted from the list even if env vars are set — intended for EEGs that configure their own privacy policy as a custom document.

`legalDocuments` contains the central privacy policy entry (`isCentralPolicy: true`) when `showCentralPolicy = true` and `CENTRAL_POLICY_URL` is set. EEG-specific documents precede it, ordered by `sortOrder`. The central policy is not stored in the database — it is configured via `CENTRAL_POLICY_TITLE` / `CENTRAL_POLICY_URL` env vars.

`cooperativeSharesEnabled` (PROJ-37): when `true`, the public form renders a "Genossenschaftsanteile" block. `cooperativeRequiredShares` (positive integer) is then the minimum the member must subscribe; `cooperativeShareAmountCents` (positive integer) is the price per share in cents. The total is computed client-side as `count × cooperativeShareAmountCents`. When `cooperativeSharesEnabled` is `false`, the two value fields are omitted from the response and the form skips the block.

`brandPreset` (PROJ-102): theme identifier for the public-page rendering. One of `teal` | `leaf` | `sun` | `slatey`, or absent/null = default theme (`teal`). The frontend maps the identifier to a hardcoded HSL-variable set and injects it as a `:root` style block at SSR time. Admins choose the preset in the Settings page (visible only in "Alle Optionen"-Modus).

`brandMode` (PROJ-103): one of `preset` | `custom`. Decides which render path the public page uses. `preset` (default — preserves PROJ-102 behaviour) renders the preset; `custom` AND `brandTheme != null` renders the custom theme on top of the preset (selective HEX override). Always present in the response (DB column is NOT NULL).

`brandTheme` (PROJ-103): only present when `brandMode = 'custom'` AND a custom theme is configured. JSON object with mandatory `v: 1` schema tag, optional 8 HEX color keys (`primary`, `primaryFg`, `accent`, `accentFg`, `background`, `foreground`, `card`, `cardFg` — all `#RRGGBB`) and optional `fontFamily` from a 4-entry whitelist (`sans-serif` / `serif` / `monospace` / `system-ui`). Backend validator (`ValidateBrandTheme`) is strict by value (WCAG-AA contrast hard-gate on the three mandatory pairs primary/primaryFg, accent/accentFg, foreground/background) and tolerant by unknown keys (dropped + warn-log; forward-compat to v2 schema). Frontend renders via HEX→HSL parallel helper (`src/lib/hsl.ts`, matched 1:1 to backend `internal/shared/hsl.go` via a shared test vector). Missing color fields fall back to the preset value; the 9 secondary CSS variables (`border`/`ring`/`popover` etc.) are deterministically derived from the 8 primary fields.

`eegName` (PROJ-32/-102) and `logoDataUri` (PROJ-33/-102): the long-form EEG name and the EEG logo as a Base64-inline data-URI. Both are populated when the corresponding Core-Sync has run. Logo is shipped inline (no second endpoint, no extra HTTP round-trip) and capped at 256 KB raw / ~342 KB Base64 by the PROJ-33 sync layer. Worst-case response size with logo is ~400 KB.

### Rate-Limit + Cache (PROJ-102)
The GET endpoint is rate-limited at **60 requests / minute / IP** (separate bucket from `POST /applications` and `POST /applications/confirm-email`) to mitigate bandwidth-amplification via repeated logo reads. Responses include `Cache-Control: public, max-age=60` so a reverse-proxy / CDN can serve cached responses for tab-switches and form-reload navigation without round-tripping to the backend.

### Errors
- `404` if `rc_number` is not found in `registration_entrypoint`
- `410` if `registration_entrypoint.is_active = false`
- `429` if the per-IP rate limit (60/min) is exceeded

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
  "networkOperatorAuthorization": true,
  "networkOperatorCustomerNumber": "K-998877",
  "meterInventoryNumber": "INV-12345",
  "hasContactPerson": true,
  "contactPersonName": "Erika Musterfrau",
  "contactPersonEmail": "erika@musterbetrieb.at",
  "contactPersonPhone": "0664/2345678",
  "hasBillingEmail": true,
  "billingEmail": "rechnung@musterbetrieb.at"
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

`networkOperatorCustomerNumber` and `meterInventoryNumber` (PROJ-56) are optional TEXT fields. They are only retained when (a) `networkOperatorAuthorization = true` AND (b) the matching `field_config` entry is not `hidden`. Otherwise the server nulls both. When the EEG sets either to `required`, a non-empty value is enforced — but only if the Vollmacht is also active (the field is conceptually gated behind the Vollmacht).

`hasContactPerson`, `contactPersonName`, `contactPersonEmail`, `contactPersonPhone` (PROJ-57) capture the optional Ansprechperson for organisation member types (`company`, `association`, `municipality`). The toggle is explicit so that "no, no contact person" and "yes, but the fields are empty" stay distinguishable on the wire. Service-side cleanup rules (`clearContactPersonIfDisabled`):
- `hasContactPerson=false` ⇒ all three TEXT fields nulled.
- `memberType` not in the org list ⇒ toggle false-set, all three nulled.
- All three subfields in `field_config` set to `hidden` (i.e. `contactPersonEnabled(fieldConfig)` is false) ⇒ toggle false-set, all three nulled.

Required-validation runs per-subfield: each name/email/phone is required only when `hasContactPerson=true` AND that subfield's `field_config` state is `required`. The e-mail format check also runs at `optional` whenever a value is supplied. There is no separate `contact_person` master switch in `field_config` (PROJ-57 v3 removed it) — the Public-Form checkbox „Ansprechperson angeben" appears automatically when at least one of the three subfields is not `hidden`.

`hasBillingEmail` and `billingEmail` (PROJ-58) capture an optional deviating billing e-mail for organisation member types. Same toggle semantics as Ansprechperson. Service nulls `billingEmail` and false-sets the toggle when `hasBillingEmail=false`, `memberType` is not an org type, or `field_config.billing_email = hidden`. Required-validation only fires when `hasBillingEmail=true` AND `field_config.billing_email = required`. E-mail format is validated whenever a non-empty value is supplied.

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
- `sepaMandateAccepted` — context-dependent (PROJ-81): must be `true` unless the EEG has activated the SEPA-choice option (`sepaOptionalEnabled = true`) **AND** the member's `memberType` is in `sepaOptionalMemberTypes`. In that case, `false` is accepted and the application is stored with `einzugsart = "kein_sepa"` (no mandate PDF). Bank fields (`iban`, `accountHolder`) remain mandatory regardless — eegFaktura-Core requires bank data for every member.

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
  "membershipStartDate": "2026-06-01",
  "personsInHousehold": 3,
  "heatPump": true,
  "electricVehicle": true,
  "electricVehicleCount": 1,
  "electricVehicleAnnualKm": 12000,
  "electricHotWater": false,
  "cooperativeSharesCount": 2,
  "networkOperatorAuthorization": true,
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
- additional editable fields mirror the public submit body: `memberType`, `titel`, `titelNach`, `companyName`, `uidNumber`, `registerNumber`
- **Zusatzangaben (admin-editable since 2026-05-28):** `membershipStartDate`, `personsInHousehold`, `heatPump`, `electricVehicle`, `electricVehicleCount`, `electricVehicleAnnualKm`, `electricHotWater`, `cooperativeSharesCount`, `networkOperatorAuthorization`. Pointer-Sentinel-Semantik: omitted ⇒ keine Änderung, explizit gesetzt ⇒ Wert übernommen. Das Admin-UI rendert ein Feld nur, wenn die EEG-Field-Config für diesen RC den State auf `optional`, `required` oder `admin_only` setzt; bei `hidden` wird das Feld weder angezeigt noch im Payload mitgesendet. Server-side rules:
  - `electricVehicleCount` and `electricVehicleAnnualKm` are nulled when `electricVehicle` is not `true` (`clearEVDetailsIfDisabled`)
  - `networkOperatorAuthorizationAt` is set only on the first `false`→`true` transition; a later `false` value does not clear the timestamp (audit-preserving)
- PROJ-56 / PROJ-57 / PROJ-58 fields are editable as in the public body: `networkOperatorCustomerNumber`, `meterInventoryNumber`, `hasContactPerson` + the three `contactPerson*` fields, `hasBillingEmail` + `billingEmail`. The same server-side cleanup rules apply (`clearContactPersonIfDisabled`, `clearBillingEmailIfDisabled`), so an admin edit that toggles a flag to `false` will null the dependent fields on save.

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
- `approved -> rejected` *(2026-05-29: nach Reset-Import landet der Antrag wieder in `approved`; der Admin kann ihn von dort ablehnen, falls das Mitglied gar nicht (mehr) importiert werden soll. Pflicht-Grund. `member_number` ist nach Reset-Import bereits NULL bzw. vor Import gar nicht gesetzt — keine Extra-Clearing-Logik.)*
- `import_failed -> approved`
- `ready_for_activation -> activated` *(PROJ-46, admin manuell; auch via Batch-Endpoint, siehe 6.5.6)*
- `ready_for_activation -> under_review` *(PROJ-46, admin rückwärts)*

Reachable only via dedicated endpoints (NOT via this generic `/status` route):
- `submitted -> email_confirmed` — via member click on `POST /api/public/applications/confirm-email`
- `imported -> ready_for_activation` — auto-transition by import service (since PROJ-91 for all einzugsarten; the previous b2b-branch via `awaiting_bank_confirmation` was removed), see 6.5
- `imported|ready_for_activation -> approved` — via `POST /api/admin/applications/{id}/reset-import` (PROJ-30 + PROJ-46, see 6.5.5).
- `activated -> imported` — via `POST /api/admin/applications/{id}/reset-activation` (PROJ-100, see 6.5.5b)
- `imported -> under_review` — via `POST /api/admin/applications/{id}/reset-to-review` (PROJ-100, see 6.5.5c)

When `registration_entrypoint.require_email_confirmation = TRUE` (PROJ-31), this endpoint rejects `submitted -> under_review|needs_info|approved` with HTTP 409 until the member has clicked the confirmation link. `submitted -> rejected` remains available as the admin's anti-spam override.

`activated` has **no transitions out** — deactivation must happen in the eegFaktura core directly (PROJ-46 Entscheidung A).

### Side effects
- on `approved`: set `approved_at`, set `reviewed_by_user_id`. **Since PROJ-46 Stage B no PDF/mail is generated here** — Beitrittsbestätigungs-PDF + Member/EEG mails are now generated/sent at import time (see 6.5), when the member number exists. The legacy `SendApprovalEmail` method and its template `application_approved_eeg.html` have been **removed** from the codebase.
- on `rejected`: set `rejected_at`, set `reviewed_by_user_id`, **PROJ-41:** synchroner Mail-Versand an Mitglied mit `reason` 1:1 im Body
- on `needs_info`: set `needs_info_reason`, **PROJ-43:** synchroner Mail-Versand an Mitglied mit `reason` 1:1 im Body
- *(PROJ-46/91: `bank_confirmed_at` trigger removed with PROJ-91; the column remains in the schema as historical evidence for migrated rows.)*
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
  "status": "ready_for_activation",
  "targetParticipantId": "4711",
  "memberTariffWarning": "core returned HTTP 404"
}
```

`status` reflects the final post-import status (PROJ-46): `ready_for_activation` for all einzugsarten since PROJ-91 (the previous b2b-branch via `awaiting_bank_confirmation` was removed; the workflow intent „prepare member for B2B" is now carried by the `prepare_b2b_documents` flag on the application). `imported` is a transient intermediate state and is rarely seen — it remains only if the auto-followup transition fails (the admin can then reset via `/reset-import`).

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
- **Bugfix 2026-05-28**: for the at-import mandate flows (`einzugsart=b2b` OR
  `einzugsart=core` AND `entrypoint.sepa_mandate_at_import=true`), the import
  now sets `mandate_reference = member_number` and `mandate_date = import_started_at`
  **before** the `POST /participant` call, so both values land in the Core's
  `accountInfo.mandateReference` / `accountInfo.mandateDate`. Idempotent: an
  admin-overridden `mandate_reference` (e.g. external customer number) is
  preserved.
- `status = imported` (transient), then **auto-transition** to `ready_for_activation`
  in a separate transaction (PROJ-46; since PROJ-91 the same path for all einzugsarten)
- write 1 or 2 entries in `status_log` (one for `→ imported`, one for the
  auto-followup)
- **PROJ-46 Stage B + PROJ-47**: best-effort async fan-out — generates the
  Beitrittsbestätigungs-PDF (mit Mitgliedsnummer), sends it to the member
  + EEG-Contact-Copy; for `einzugsart=b2b` OR `einzugsart=core` AND
  `prepare_b2b_documents=true` (PROJ-91) adds a second attachment
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

### GET `/api/admin/registration-entrypoints`

PROJ-101 (2026-06-11). Liefert ein schlankes Verzeichnis der EEGs, das
der Caller sehen darf — Tenant-Admin filtert auf den eigenen Tenant-Claim,
Superuser sieht alle. Wird vom Admin-Layout beim Mount einmal geladen
und über einen React-Context an die drei EEG-Auswahllisten (Settings-
Switcher, Antrags-Filter-Panel, Reassign-Dialog) sowie die Antragslisten-
Spalte „EEG" verteilt.

Bewusst PII-frei (kein IBAN, kein CreditorID, keine Adress-Felder).

Sortierung im Backend: alphabetisch nach `eegShortName` (NULL ans Ende),
Sekundär-Sort nach `rcNumber`.

#### Response 200
```json
{
  "entrypoints": [
    {
      "rcNumber": "RC0001",
      "eegShortName": "EEG-Test",
      "eegName": "Testenergiegemeinschaft EEG 1234"
    },
    {
      "rcNumber": "RC0002",
      "eegName": "Musterenergiegemeinschaft EEG"
    }
  ]
}
```

- `eegShortName` und `eegName` sind optional (`omitempty`); NULL-Werte
  in der DB werden weggelassen statt als `null` gesendet.
- Bei leerem Verzeichnis (Tenant ohne RCs, Fetch-Fehler) fallen die
  Listboxen auf reine RC-Darstellung zurück.

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
- Application must be in status `imported` or `ready_for_activation`
  (PROJ-46 expansion; the previous `awaiting_bank_confirmation` branch was
  removed by PROJ-91). NOT `activated` — admin must first call
  `POST /reset-activation` (PROJ-100) to roll the activation back, then
  this endpoint becomes available.
- The transitions `imported → approved`, `ready_for_activation → approved`
  are **only** reachable via this endpoint; the generic `POST /status`
  does not accept them.
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

## 6.5.5b Reset activation back to imported (PROJ-100)

### POST `/api/admin/applications/{id}/reset-activation`

Transitions an application from `activated` back to `imported`. First step
of the Owner-recovery chain for irrtümliche Aktivierungen — typical
trigger: a wrong manual `mark-activated` click or a `check-activation`
batch trigger that fired despite missing Core activity.

The Core is **NOT** contacted. The admin verifies the Core member status
separately. After this rollback the admin can either re-activate (via
`mark-activated` once the Core lage is correct) or drill further back
with `/reset-to-review` and then `rejected`.

### Request
```json
{
  "reason": "Aktivierung versehentlich angeklickt, Mitglied noch nicht im Core aktiv."
}
```

| Field | Required | Constraints |
|---|---|---|
| `reason` | yes | 10–500 chars (after trimming). Higher bar than `/reset-import` (min 5) because the rollback reverts a member-visible activation. |

### Rules
- Application must be in status `activated` (otherwise 409).
- Only reachable via this endpoint — generic `POST /status` does not accept
  the transition; the `adminTransitions` map intentionally has no entry for
  `activated`, and a drift-wache test enforces this.
- Tenant-Admin scope: must match the EEG of the application.

### Response 200
Returns the full `AdminApplicationDetail` after the reset (status now
`imported`).

### Side effects
- `status = imported`
- `activated_at = NULL`
- `activation_notification_sent_at = NULL` — so a fresh activation triggers
  the Beitrittsbestätigungs-Mail again
- `board_declaration_sent_at = NULL` — analog for the board-mode
- `member_number`, `target_participant_id`, `imported_at`, `mandate_reference`,
  `mandate_date`, `bank_confirmed_at` all **preserved** — the member is still
  considered imported in the Core
- write `status_log` entry with `from='activated'`, `to='imported'`,
  `reason='[reset-activation] <user reason>'` (system prefix makes the
  rollback visible in audit traces at a glance)

### Failure responses
- `400` reason missing / too short / too long
- `403` tenant mismatch
- `409` application not in `activated` status

---

## 6.5.5c Reset imported back to under_review (PROJ-100)

### POST `/api/admin/applications/{id}/reset-to-review`

Transitions an application from `imported` back to `under_review`. Second
step of the Owner-recovery chain after `/reset-activation` — typically used
when the admin decides the whole onboarding was wrong (data quality
insufficient, member retreated). From `under_review` the bestehender
Reject-Pfad reaches `rejected`.

Every import- and activation-bookkeeping field is cleared (13 columns
identical to `/reset-import`, only the target status differs). The Core
is **NOT** contacted; the admin handles Core cleanup separately.

### Request
```json
{
  "reason": "Mitglied hat den Beitritt nachträglich zurückgezogen; Antrag wird abgelehnt."
}
```

| Field | Required | Constraints |
|---|---|---|
| `reason` | yes | 10–500 chars (after trimming) |

### Rules
- Application must be in status `imported` (otherwise 409).
- Refuses with 409 while an import is in flight (`import_started_at` set,
  `import_finished_at` null) — identical guard to `/reset-import`.
- Only reachable via this endpoint — the `adminTransitions` map has no
  `imported` entry; drift-wache enforces.
- Tenant-Admin scope: must match the EEG of the application.

### Response 200
Returns the full `AdminApplicationDetail` after the reset (status now
`under_review`, `targetParticipantId` + `memberNumber` cleared).

### Side effects
Same 13 fields as `/reset-import` — `import_started_at`, `import_finished_at`,
`imported_at`, `target_participant_id`, `import_error_message`,
`member_number`, `bank_confirmed_at`, `activated_at`,
`activation_notification_sent_at`, `board_declaration_sent_at`,
`mandate_reference`, `mandate_date` — all set to NULL plus
`updated_at = NOW()`. Only difference: target status is `under_review`
instead of `approved`.

write `status_log` entry with `from='imported'`, `to='under_review'`,
`reason='[reset-to-review] <user reason>\n[system] previous target_participant_id=<uuid>\n[system] previous member_number=<x>'`
(prefix + suffixes preserve both the rollback signal and the archived
values for audit).

### Failure responses
- `400` reason missing / too short / too long
- `403` tenant mismatch
- `409` application not in `imported` status or import in flight

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

## 6.5.7 Resync application data from eegFaktura core (PROJ-70)

### POST `/api/admin/applications/{id}/resync-from-core`

Pullt den aktuellen Faktura-Teilnehmer-Datensatz für einen aktivierten Antrag und überschreibt in der Onboarding-DB Stammdaten, Adresse, Kontakt- und Bankverbindungsfelder, wo der Core einen normalisierten anderen Wert hält. Kein Status-Wechsel, kein Mandat-Reset — wenn IBAN oder Kontoinhaber sich ändern, muss der Admin separat `POST /resend-mandate` klicken.

### Headers
- `Authorization: Bearer <keycloak-token>` — Admin-Auth, Tenant-Check via `target_participant_id` der Application
- `X-Core-Authorization: Bearer <silent-sso-token>` — Token für den Faktura-Core-Call (bei `CORE_AUTH_MODE=exchange`)

### Request
No body.

### Response 200 — Diff erkannt
```json
{ "changed": ["firstname", "iban", "residentZip"] }
```

### Response 200 — Bereits synchron
```json
{ "changed": [] }
```

### Verglichene Felder (14)
- Stammdaten: `firstname`, `lastname`, `titel`, `titelNach`, `uidNumber`
- Adresse: `residentStreet`, `residentStreetNumber`, `residentZip`, `residentCity` (gelesen aus Faktura `residentAddress`, **nicht** `billingAddress`)
- Kontakt: `email` (case-insensitive Vergleich), `phone`
- Bank: `iban` (whitespace-strip + uppercase Vergleich), `bankName`, `accountHolder`

### NICHT abgeglichen (out-of-scope)
- `memberType`, `birthDate`, `membershipStartDate`, `registerNumber`, `companyName`
- Zählpunkte (`metering_point`)
- `cooperative_shares_count`
- Mandat-Felder (`mandate_reference`, `mandate_date`, `sepa_mandate_accepted`, `sepa_mandate_accepted_at`, `einzugsart`)
- `faktura_handover_at`, `activation_notification_sent_at`, `admin_note`

### Keep-bei-NULL
Wenn der Core für ein Feld NULL oder einen leeren String liefert, bleibt der Onboarding-Wert unverändert (kein Diff).

### Side-Effects bei Real-Change
- `application.updated_at = NOW()`
- Geänderte Spalten überschrieben mit getrimmter Core-Original-Form (IBAN/Email behalten ihre Faktura-Schreibweise, nicht die normalisierte Form)
- `status_log`-Eintrag mit `from_status = to_status = activated`, Reason-Text „Stammdaten aus eegFaktura abgeglichen (geänderte Felder: …)", changed_by_user_id = `resync:<keycloak-subject>`

### Errors
- `400` UUID malformed, oder `X-Core-Authorization`-Header fehlt im exchange-Mode
- `401` ohne Auth
- `403` Tenant-Mismatch (Antrag gehört einer fremden EEG)
- `404` Antrag nicht gefunden
- `409` Antrag nicht in Status `activated`, oder `target_participant_id` ist NULL (manuell aktiviert ohne Import)
- `502` Core-Call fehlgeschlagen, Mitglied nicht in Faktura gefunden (`code=core_member_not_found`), oder generischer Core-Fehler

---

## 6.5.8 Resend SEPA mandate mail (PROJ-70)

### POST `/api/admin/applications/{id}/resend-mandate`

Generiert ein neues SEPA-Mandat-PDF aus den aktuellen Onboarding-Werten (typischerweise nach einem Resync mit IBAN-/Kontoinhaber-Wechsel) und versendet es an das Mitglied. Hard-Fail bei SMTP-Error — der Admin sieht den Fehler im UI-Toast. Kein Status-Wechsel.

### Request
No body.

### Response 200
```json
{ "mailSent": true }
```

### Side-Effects
- Mail an `application.email` mit aktualisierter `IsRenewal=true`-Verzweigung in den Templates (Subject: „Aktualisiertes SEPA-Mandat – Mitgliedsnummer X")
- EEG-Kopie an `registration_entrypoint.contact_email`, sofern gesetzt
- `status_log`-Eintrag „SEPA-Mandat-Mail erneut versandt", changed_by_user_id = `mandate-renewal:<keycloak-subject>`

### Errors
- `400` UUID malformed
- `401` ohne Auth
- `403` Tenant-Mismatch
- `404` Antrag nicht gefunden
- `409` Antrag nicht in `activated`, oder `einzugsart = kein_sepa` (kein Mandat zu versenden)
- `500` Mail-Versand fehlgeschlagen (SMTP unreachable, Template-Render-Fehler, PDF-Generierung gescheitert)

### Hinweise
- **Kein Rate-Limit** (Owner-Entscheidung Simplification-Pass). Frontend-Doppel-Klick-Schutz greift via `disabled`-during-Request.
- Mandat-Mail-Template (`application_imported_member.html` + `application_imported_eeg.html`) verzweigt auf `IsRenewal` und zeigt „deine Bankverbindung wurde aktualisiert" statt der Import-Wording.

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
    "heat_pump": { "state": "required" },
    "transformer": { "state": "optional" },
    "persons_in_household": { "state": "admin_only" }
  }
}
```

Each field entry contains only `state`. PROJ-68 removed the EEG-wide default-value (`adminValue`) that used to live here — fields with `state = "admin_only"` are simply hidden from the public form and remain editable in the admin per-application edit dialog.

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
  "persons_in_household": { "state": "admin_only" }
}
```

Allowed field names: `phone`, `birth_date`, `uid_number`, `bank_name`, `membership_start_date`, `persons_in_household`, `consumption_previous_year`, `consumption_forecast`, `feed_in_forecast`, `pv_power_kwp`, `inverter_power_kw`, `heat_pump`, `electric_vehicle`, `electric_vehicle_count`, `electric_vehicle_annual_km`, `electric_hot_water`, `network_operator_authorization` *(PROJ-44)*, `transformer`, `installation_number`, `installation_name`, `battery_size_kwh` *(PROJ-45)*, `inverter_manufacturer` *(PROJ-45)*

**Type-conditional visibility (PROJ-45):** the admin UI shows badges next to each conditional field — `[Verbraucher]` (only renders when the application has ≥1 CONSUMPTION metering point), `[Einspeisung]` (≥1 PRODUCTION), `[PV]` (additionally requires `generation_type='pv'` on the MP), `[+E-Auto]` (additionally requires `electric_vehicle=true`). Backend mirrors the gate: the required-check on these fields only fires when the matching MP-type / EV flag is present.

Allowed states: `hidden`, `optional`, `required`, `admin_only`

When `state = "admin_only"`: the field is hidden from the public registration form, but remains visible and editable in the admin per-application edit dialog. PROJ-68 removed the previous EEG-wide default-value (`adminValue`); legacy bodies that still carry the field are silently accepted and the value is dropped.

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
  "sepaMandateAtImport": false,
  "showCentralPolicy": true,
  "memberNumberStart": 1,
  "requireEmailConfirmation": false,
  "meteringPointPrefixConsumption": "AT00060010001",
  "meteringPointPrefixProduction": null,
  "activationMode": "participant_active",
  "cooperativeSharesEnabled": true,
  "cooperativeRequiredShares": 1,
  "cooperativeShareAmountCents": 10000,
  "boardApprovalWorkflowEnabled": false,
  "joiningConfirmationToEEG": false,
  "sepaMandateCoreAuditEnabled": false,
  "sepaMandateB2BAuditEnabled": false,
  "sepaOptionalEnabled": false,
  "sepaOptionalMemberTypes": []
}
```

**Core-mastered fields** (PROJ-32, read-only — only modified via `/sync` below): `eegId`, `eegName`, `eegStreet`, `eegStreetNumber`, `eegZip`, `eegCity`, `creditorId`, `contactEmail`. `lastSyncedFromCoreAt` is `null` until the first successful sync.

`registrationActive` is `false` by default. `sepaMandateAtImport` (PROJ-48) defaults to `false`. (PROJ-73 removed the obsolete `useCompanySEPAMandate` EEG toggle; PROJ-80 removed the `sepaMandateEnabled` toggle. B2B/Core mandate selection now lives exclusively in the per-application `einzugsart` field; the variant of the always-generated PDF — audit-trail vs. signature field — is steered by `sepaMandateCoreAuditEnabled` / `sepaMandateB2BAuditEnabled`.) `showCentralPolicy` defaults to `true`. `memberNumberStart` defaults to `1`. `requireEmailConfirmation` (PROJ-31) defaults to `false`.

`cooperativeSharesEnabled` (PROJ-37) defaults to `false`. When `true`, both `cooperativeRequiredShares` (positive integer, minimum mandatory shares per member) and `cooperativeShareAmountCents` (positive integer, price per share in cents) are returned. When `false`, those two value fields are omitted.

`boardApprovalWorkflowEnabled` (PROJ-76) defaults to `false`. When `true`, the `→ activated` transition sends a **Beitrittserklärung** (with Vorstands-Signaturblock) to the EEG contact instead of an automatic Beitrittsbestätigung to the member. The board signs manually and forwards the document; the member is informed via the regular eegFaktura-Core activation mail. Status transition is sync hard-fail: missing `contact_email` or SMTP failure rolls back the activation.

`joiningConfirmationToEEG` (PROJ-114) defaults to `false`. When `true`, the `→ activated` transition sends the **Beitrittsbestätigung** (the PDF the member would normally receive) as a single forward-framed mail to the EEG `contact_email` (subject „Beitrittsbestätigung für … – bitte weiterleiten", Reply-To = member) instead of mailing the member; the member receives **nothing** and the usual separate EEG copy is suppressed. The board forwards the mail to the member, optionally adding a personal note. Independent of `boardApprovalWorkflowEnabled` — that toggle routes the *Beitrittserklärung* (a document to sign), this one the finished *Beitrittsbestätigung* (to forward); both can be on or off independently. Send-time fallback: if `contact_email` is empty when the mail is built, it falls back to the member and logs a warning (defense-in-depth behind the save-time validation below).

`sepaMandateCoreAuditEnabled` and `sepaMandateB2BAuditEnabled` (PROJ-78) both default to `false`. When `true` for the matching mandate type (`einzugsart=core` resp. `einzugsart=b2b`), the SEPA-mandate PDF renders the electronic audit-trail block (formfreie Willenserklärung gem. § 76 (3) EIWOG 2010 — Tenant, Zustimmungs-Zeitstempel, IP-Adresse) **in place of** the classic Datum/Unterschrift-Block. When `false`, the classic block is always rendered, even if the audit data is fully populated. The two toggles are independent: a single EEG can opt into the electronic variant for B2B (Geschäftsleute) while keeping CORE (Verbraucher) on the classic signature workflow, or vice versa. Audit fallback (PROJ-77): even with the toggle on, the renderer falls back to the classic block if any of the three audit data fields (`AuditTenant`, `sepa_mandate_accepted_at`, `sepa_mandate_accepted_ip`) is empty — relevant for legacy applications without IP capture.

`sepaOptionalEnabled` and `sepaOptionalMemberTypes` (PROJ-81) default to `false` / `[]`. When `sepaOptionalEnabled = true`, the SEPA-mandate online-consent checkbox is rendered **optional instead of mandatory** in the public registration form for the listed member types. A member who does not tick the checkbox is recorded with `einzugsart = "kein_sepa"` (no mandate PDF). **Bank fields (`IBAN`, `accountHolder`) remain mandatory in all cases** — eegFaktura-Core requires bank data for every member regardless of mandate status. Allowed list values: `private`, `farmer`, `association`, `municipality`. `company` is **never** valid in the list (B2B mandatory direct debit) — the backend returns `400` if a request body includes `company`. The configexport importer silently filters `company` and logs a warning. Cross-field validation: when `sepaOptionalEnabled = true` and `sepaOptionalMemberTypes` is empty, the save endpoint returns `400`.

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
  "cooperativeShareAmountCents": 10000,
  "boardApprovalWorkflowEnabled": false,
  "joiningConfirmationToEEG": false,
  "sepaMandateCoreAuditEnabled": false,
  "sepaMandateB2BAuditEnabled": false,
  "sepaOptionalEnabled": false,
  "sepaOptionalMemberTypes": ["private", "farmer"]
}
```

`joiningConfirmationToEEG` (PROJ-114): when `true`, the activation Beitrittsbestätigung is forwarded to the EEG `contact_email` instead of the member (see the GET section above for the full routing). **Cross-field validation:** if the body sets `joiningConfirmationToEEG = true` while the synchronised (Core-mastered) `contact_email` is empty, the save returns `400` with field `joiningConfirmationToEEG` and message „Bitte zuerst in eegFaktura eine Kontakt-E-Mail hinterlegen und die Stammdaten synchronisieren." (the value is validated against the stored Core value, not against the request body, because `contact_email` is read-only in onboarding).

`activationMode` (PROJ-53) is optional in the request — `null`/omitted leaves the existing value unchanged (patch semantics). Allowed values: `participant_active` (default for new EEGs — Core-Teilnehmer-Status `ACTIVE` löst Activation aus) or `any_meter_registration_started` (mind. ein Zählpunkt mit `processState ∈ {PENDING, APPROVED, ACTIVE}`). Invalid values return `400`.

`registrationActive`: enables or disables the public registration form for this EEG. When `false`, `GET /api/public/registration/{rc_number}` returns `410 Gone`.

`showCentralPolicy`: when `false`, the central operator privacy policy is not shown in the public registration form. Intended for EEGs that have configured their own privacy policy as a custom document (see 6.16).

**Mandate variant (Core vs B2B)** is selected per application via the `einzugsart` field (`core` | `b2b` | `kein_sepa`, default `core`). PROJ-73 (2026-06-06) removed the obsolete `useCompanySEPAMandate` EEG-global toggle, which had been a no-op since PROJ-48 replaced the auto-mapping `company|association → b2b` with the per-application `einzugsart` model. A legacy admin client that still sends `useCompanySEPAMandate` in the body is tolerated — the unknown field is ignored.

**PROJ-80 (2026-06-08)** entfernt den `sepaMandateEnabled`-Toggle. Seit PROJ-80 wird das SEPA-Mandat-PDF für jedes SEPA-Mitglied (`einzugsart != kein_sepa`) automatisch erzeugt; die Variante (Audit-Trail vs. klassisches Unterschriftenfeld) wird über `sepaMandateCoreAuditEnabled` und `sepaMandateB2BAuditEnabled` (PROJ-78) gesteuert. Die Online-Zustimmung-Checkbox im Public-Form ist Pflicht. Fehlt es an EEG-Stammdaten (z.B. `creditorId`), bricht der `POST /api/admin/applications/{id}/import` mit `409 Conflict` ab und nennt die fehlenden Felder.

`sepaMandateAtImport` (PROJ-48): when `true`, **Core**-mandates are generated **at import time** (with the assigned Mitgliedsnummer printed as Mandatsreferenz) rather than at submit time (without reference). Use when the EEG runs a digital signature workflow on the Core-mandate — a signed PDF cannot be modified afterwards, so the reference must be present before signing. When `false` (default), Core-mandates are generated at submit time with the application reference number as Mandatsreferenz. PROJ-80 Cross-Field-Coupling: wenn `sepaMandateCoreAuditEnabled = true`, ist `sepaMandateAtImport = true` zwingend (Backend-Validation 400 sonst). PROJ-74-Klarstellung: **wirkt nur auf Core-Mandate.** B2B-Mandate kommen unabhängig vom Toggle beim Import — die Mandatsreferenz wird erst dort vergeben.

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

## 6.11d Get/Save settings view mode (PROJ-67)

Per-EEG persistierter Sichtbarkeits-Modus der Admin-Settings-Page. UI-Pref-only, kein Backend-Enforcement (heute). Eigenes Endpoint-Paar getrennt von `/api/admin/settings/eeg`, damit der Page-Header-Toggle unabhängig vom dirty-State des Stammdaten-Editors arbeitet.

UI-Labels: `standard` → „Einfache Ansicht", `advanced` → „Alle Optionen". DB-Werte bleiben technisch `standard`/`advanced`.

### GET `/api/admin/settings/view-mode?rc_number={rc_number}`

### Response 200
```json
{
  "rcNumber": "RC123456",
  "viewMode": "advanced"
}
```

### PUT `/api/admin/settings/view-mode?rc_number={rc_number}`

### Request body
```json
{
  "viewMode": "standard"
}
```

`viewMode` ist Pflicht. Erlaubt: `"standard"` oder `"advanced"`. Case-sensitive — ungültige Werte → 400 mit `{ "viewMode": "ungültiger Wert (erlaubt: standard, advanced)" }`. DB-CHECK-Constraint als Safety-Net.

### Response 200
```json
{
  "rcNumber": "RC123456",
  "viewMode": "standard"
}
```

### Errors
- `400` missing `rc_number`, ungültige `viewMode`, oder malformed JSON
- `401` ohne Auth
- `403` nicht autorisiert für diese EEG
- `404` `rc_number` nicht in `registration_entrypoint`

### Migration-Defaults
- Bestehende EEGs (vor PROJ-67): `'advanced'` (rückwärts-kompatibel)
- Neu angelegte EEGs: `'standard'`

### Config-Export (PROJ-61) Integration
`settingsViewMode` ist in `EEGSettingsSection.settingsViewMode` (Pointer, additiv in v1 — kein SchemaVersion-Bump). Pre-PROJ-67-Bundles ohne das Feld werden beim Import als `'advanced'` interpretiert.

---

## 6.11e Run reconciliation (PROJ-69)

Login-getriggerter Abgleich der Onboarding-Anträge dieser EEG gegen die Faktura-Teilnehmerliste. Strict 2-Keys-Match (IBAN + E-Mail-Adresse exakt). Bei Treffer wird `application.faktura_handover_at` rückwirkend gesetzt und `member_number` befüllt, sofern noch NULL. Ein Status-Log-Eintrag „In eegFaktura erfasst (automatischer Abgleich)" wird ergänzt. Throttle: max. 1 Run pro EEG pro UTC-Tag (atomar über DB-UNIQUE).

### POST `/api/admin/reconciliation/run?rc_number={rc_number}`

### Headers
- `Authorization: Bearer <keycloak-token>` — Admin-Auth, Tenant-Check über RC-Number-Claim
- `X-Core-Authorization: Bearer <silent-sso-token>` — Token für den Faktura-Core-Call (bei `CORE_AUTH_MODE=exchange`)

### Response 200 — Normal-Run
```json
{
  "rcNumber": "RC123456",
  "runId": "8b3e…",
  "skipped": false,
  "matched": 3,
  "ambiguous": 0,
  "mnrConflicts": 0,
  "duplicates": 0,
  "alreadyHandedOver": 1,
  "errors": 0
}
```

### Response 200 — Throttled (heute schon gelaufen)
```json
{
  "rcNumber": "RC123456",
  "skipped": true,
  "skipReason": "throttled",
  "matched": 0, "ambiguous": 0, "mnrConflicts": 0,
  "duplicates": 0, "alreadyHandedOver": 0, "errors": 0
}
```

### Response 200 — Feature-Flag aus
```json
{ "rcNumber": "RC123456", "skipped": true, "skipReason": "disabled" }
```

### Counter-Semantik
- `matched` — Antrag bekam neuen `faktura_handover_at` + ggf. `member_number`
- `ambiguous` — ≥2 Core-Teilnehmer mit identer IBAN+E-Mail (skip + Detail-Log)
- `mnrConflicts` — Core-MNr ist in dieser EEG schon einem anderen Antrag zugewiesen → handover gesetzt, MNr nicht überschrieben
- `duplicates` — gleicher Core-Treffer matched mehrere Anträge → ältester gewinnt, jüngere bekommen Duplicate-Detail
- `alreadyHandedOver` — `faktura_handover_at` war bereits gesetzt (z. B. /import lief parallel) → kein Detail-Log
- `errors` — pro-Antrag Fehler (DB oder Conflict-Check), Run läuft weiter

### Errors
- `400` ohne `rc_number` — oder fehlender `X-Core-Authorization`-Header (Silent-SSO nicht bootstrapped)
- `401` ohne Auth
- `403` nicht autorisiert für diese EEG
- `502` Core-Call fehlgeschlagen — Run-Header wird mit `errors=1` finalisiert; Caller kann beim nächsten Login retry'n

### Feature-Flag
Backend-seitig durch `RECONCILIATION_ENABLED=true` aktiviert (Helm-Wert `backend.reconciliationEnabled`). Bei `false` antwortet der Endpoint mit `{skipped:true, skipReason:"disabled"}` — der Tenant-Check läuft TROTZDEM zuerst, damit der Feature-Status nicht über die Fehlerantwort an fremde Tenants leakt.

### Persistenz
Jeder Lauf erzeugt eine Zeile in `member_onboarding.reconciliation_run` (Header mit Countern) plus N Zeilen in `member_onboarding.reconciliation_match_detail` (positive Treffer + Konflikte; KEIN Detail-Eintrag für no-match oder already-handed-over). Siehe `docs/domain-model.md`.

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

Generates and downloads an xlsx file for the given application in eegFaktura import format. Only available for applications in status `approved`, `imported`, `import_failed`, `ready_for_activation`, or `activated`.

**Side effect (PROJ-64, 2026-05-29):** sets `application.faktura_handover_at = NOW()` if currently NULL. The xlsx matches the eegFaktura import template (36 columns A-AJ) and is therefore considered an off-platform handover for billing purposes. Subsequent downloads do not update the timestamp. Persist failure is logged but does not abort the download — a later `/import` or re-download would catch up.

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

Generates and downloads the Beitrittsbestätigung (approval confirmation) as a PDF file for the given application. Available for applications in status `approved`, `imported`, `import_failed`, `ready_for_activation`, or `activated`.

### Auth
Keycloak JWT. Tenant-admin access is checked against the application's RC number.

### Response
- `200 OK` — PDF file
  - `Content-Type: application/pdf`
  - `Content-Disposition: attachment; filename="beitrittsbestaetigung-{referenceNumber}.pdf"`
- `404 Not Found` — application not found
- `403 Forbidden` — tenant mismatch
- `409 Conflict` — application not in downloadable status

The PDF is identical to the one auto-attached to the member/EEG mails at import time (PROJ-46 Stage B). Up to PROJ-46 Stage B this PDF was emailed to the EEG on `→ approved`; that auto-send is gone and the PDF generation is now anchored to the import step (so the member number is available for the SEPA mandate reference). For B2B applicants (`einzugsart=b2b`) and for CORE-applicants with `prepare_b2b_documents=true` (PROJ-91) the import mail additionally contains a separate Firmenlastschrift-Mandat-PDF with embedded Mandatsreferenz=Mitgliedsnummer (PROJ-47); that mandate PDF is currently **not** downloadable via this endpoint.

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

## 6.18 Data export (PROJ-60)

Async plugin framework for forwarding member data to external systems. V1 ships the
Excel/CSV plugin; Phase 2 adds CRM plugins (Zoho, HubSpot, …) on the same framework.
All endpoints require Keycloak admin JWT + `rc_number` query parameter (except `/plugins`
which is global). Tenant isolation enforced via `parseRCAndCheck` + service-layer
re-validation.

### 6.18.1 List registered plugins

`GET /api/admin/data-export/plugins`

No `rc_number` required — the registry is global.

#### Response 200
```json
{
  "plugins": [
    {
      "type": "excel",
      "displayName": "Excel/CSV-Export",
      "standardConfigs": [
        { "name": "Newsletter-Adressliste", "config": { "format": "xlsx", "columns": [...] } },
        { "name": "CRM-Stammdaten",         "config": { "format": "xlsx", "columns": [...] } },
        { "name": "Buchhaltungs-Export",    "config": { "format": "xlsx", "columns": [...] } }
      ]
    }
  ]
}
```

### 6.18.2 List configurations

`GET /api/admin/data-export/configs?rc_number=AT00001`

Returns non-deleted configs for the EEG, ordered `plugin_type, name`.

#### Response 200
```json
{
  "configs": [
    {
      "id": "11111111-2222-3333-4444-555555555555",
      "rcNumber": "AT00001",
      "pluginType": "excel",
      "name": "Newsletter",
      "config": { "format": "xlsx", "columns": [...] },
      "isObsolete": false,
      "createdAt": "2026-05-23T12:00:00Z",
      "updatedAt": "2026-05-23T12:00:00Z"
    }
  ]
}
```

### 6.18.3 Create configuration

`POST /api/admin/data-export/configs?rc_number=AT00001`

```json
{
  "pluginType": "excel",
  "name": "Newsletter",
  "config": {
    "format": "xlsx",
    "columns": [
      { "header": "Vorname",  "field": "firstname", "format": "string" },
      { "header": "E-Mail",   "field": "email",     "format": "string" }
    ]
  }
}
```

**Excel `rowMode` (PROJ-112).** The excel config accepts an optional `rowMode`:
- `"member"` (default; also when the field is absent → backward-compatible with existing configs): one row per member. Metering-point fields render all of a member's values pipe-separated (`" | "`) in one cell.
- `"metering_point"`: one row per metering point, member data repeated per row, each metering-point field a single value. A member without metering points yields one row with empty metering-point columns.

The same metering-point fields are selectable in both modes (e.g. `metering_point`, `direction`, `consumption_previous_year`, `pv_power_kwp`, …). The aggregate fields (`*_sum`, `has_battery`) and `meter_numbers` are **member-mode only** and rejected with `400` if used while `rowMode = "metering_point"`. Rows are sorted by member number (members without a number sort last, then by name); in `metering_point` mode, a member's rows are ordered by metering-point number.

Server-side validation:
- `pluginType` must exist in the registry
- `name` must be unique per EEG across all plugin types (cross-plugin-type collision)
- `Plugin.ValidateConfig(config)` is called — for excel: `rowMode` must be empty/`member`/`metering_point`; at least 1, at most 50 columns, each with non-empty unique header, known field (member **or** metering-point catalogue), a format that matches the field's type, and (in `metering_point` mode) not a member-only field
- Per-EEG limit: max 20 non-deleted configurations (`DataExportMaxConfigsPerEEG`)

#### Response 201
Same shape as 6.18.2.

#### Errors
- `400 validation_error` — field-level errors under `fields` (`columns[i].header`, `columns[i].field`, `columns[i].format`)
- `403 forbidden` — tenant mismatch

### 6.18.4 Get configuration

`GET /api/admin/data-export/configs/{id}?rc_number=AT00001`

### 6.18.5 Update configuration

`PUT /api/admin/data-export/configs/{id}?rc_number=AT00001`

Same body as create. `pluginType` cannot be changed. Obsolete configs cannot be updated.

### 6.18.6 Delete configuration (soft)

`DELETE /api/admin/data-export/configs/{id}?rc_number=AT00001`

Sets `deleted_at = NOW()`. Active jobs referencing this config continue to run with the
frozen snapshot. Hard-delete happens only via the cleanup CronJob after 7 years.

#### Response 204
No body.

### 6.18.7 Live preview

`POST /api/admin/data-export/configs/preview?rc_number=AT00001`

```json
{
  "pluginType": "excel",
  "rcNumber": "AT00001",
  "config": { "format": "xlsx", "columns": [...] }
}
```

Runs `ValidateConfig` then renders the latest 5 post-imported members (`imported`,
`ready_for_activation`, `activated`) through the column mapping. Falls back to the plugin's synthetic sample when the EEG has no imported members
yet (`note` field populated).

#### Response 200
```json
{
  "headers": ["Vorname", "E-Mail"],
  "rows": [
    { "Vorname": "Max", "E-Mail": "max@example.com" }
  ],
  "note": ""
}
```

### 6.18.8 Trigger job

`POST /api/admin/data-export/jobs?rc_number=AT00001`

```json
{
  "configId": "11111111-2222-3333-4444-555555555555",
  "applicationIds": ["aaaa...", "bbbb..."]
}
```

Validation:
- 1 ≤ `applicationIds` ≤ 1000 (`DataExportMaxApplications`)
- All IDs must belong to `rc_number`
- Config must exist and not be obsolete
- Concurrency soft-limit: max 3 active jobs per EEG (overshoot 4-5 tolerated)

The job is created with `status='queued'`, the config payload is snapshotted, and a 202
is returned immediately. The in-app worker pool picks it up within 5 seconds.

#### Response 202
Job-shape (see 6.18.10).

#### Errors
- `400 validation_error` (limits, UUIDs)
- `403 forbidden`
- `404 not_found` — config-id unknown
- `409 conflict` — config is obsolete

### 6.18.9 Retry job

`POST /api/admin/data-export/jobs/{id}/retry?rc_number=AT00001`

Creates a NEW queued job with the same snapshot (config + application IDs). Original job
is untouched (audit trail preserved). Frontend modal re-subscribes its polling to the new
job-id via the `onRetried` callback.

#### Response 202
New job-shape.

### 6.18.10 Get job status

`GET /api/admin/data-export/jobs/{id}?rc_number=AT00001`

Frontend polls this every 2-5 seconds while the job is queued/running.

#### Response 200
```json
{
  "id": "...",
  "rcNumber": "AT00001",
  "configId": "...",
  "pluginType": "excel",
  "status": "running",
  "adminUserId": "f47ac10b-...",
  "processedCount": 47,
  "totalCount": 200,
  "resultSummary": { "downloaded": 47, "file_size": 12345 },
  "errorMessage": null,
  "retryCount": 0,
  "hasResult": false,
  "resultFileName": null,
  "resultFileSize": null,
  "createdAt": "2026-05-23T12:00:00Z",
  "startedAt": "2026-05-23T12:00:05Z",
  "finishedAt": null
}
```

`status` ∈ `queued`, `running`, `done`, `failed`, `expired`. `errorMessage` is
user-safe text (never contains stack traces or DB internals).

### 6.18.11 List jobs (BackOffice)

`GET /api/admin/data-export/jobs?rc_number=AT00001&status=failed&since=...&until=...&cursor=...&limit=50`

Filter: optional `status`, `since`, `until` (RFC3339). Pagination: cursor-based via
`created_at` of the last item; pass it back as `cursor`. Default limit 50, max 200.

#### Response 200
```json
{
  "jobs": [ { ... }, { ... } ],
  "failedLast7Days": 3,
  "nextCursor": "2026-05-23T11:55:00Z"
}
```

`failedLast7Days` powers the red Failed-Jobs-Badge in the BackOffice UI.

### 6.18.12 Download result file

`GET /api/admin/data-export/jobs/{id}/download?rc_number=AT00001`

Only available for `status='done'` jobs with a non-expired result. The filename built by
the worker follows `{rc_number}-{config_name}-{YYYY-MM-DD}.{xlsx|csv}` after stripping
path-traversal characters from the segments.

#### Response 200
Binary stream with `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
or `text/csv; charset=utf-8`, `Content-Disposition: attachment; filename="..."`,
`Content-Length: <bytes>`. CSV files include a UTF-8 BOM + semicolon separator (DACH-Excel
convention). All cell values whose first non-whitespace character is `=`, `+`, `-`, `@`,
TAB or CR are prefixed with `'` to defang CSV/Excel-injection.

#### Errors
- `404 not_found` — job-id unknown or result BLOB expired
- `409 conflict` — job not in `done` status

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
  "submitterIp": "203.0.113.42",
  "meteringPoints": [
    { "meteringPoint": "AT0010000000000000001000000000001", "direction": "CONSUMPTION", "participationFactor": 100 }
  ]
}
```

### `submitterIp` (PROJ-77, NEU 2026-06-07)

Optionaler Body-Param mit der **End-User-IP** zum Zeitpunkt der
SEPA-Mandats-Akzeptanz. Wird beim B2B-Firmenlastschrift-PDF als
Audit-Trail-Text gerendert (formfreie Willenserklärung gem. § 76 (3)
EIWOG 2010).

- Format: IPv4 (`192.0.2.42`) oder IPv6 (`2001:db8::42`)
- Validierung: bei ungültigem Format → `400 validation_error` mit
  Feld-Hinweis
- Fehlt der Param oder ist leer: das B2B-PDF fällt auf den klassischen
  Datum/Unterschrift-Block zurück (Backward-Compat).
- Hintergrund: bei Server-zu-Server-Integration ist `r.RemoteAddr` die
  IP des EEG-Integrators, nicht des Mitglieds. Nur der EEG-Integrator
  kennt die End-User-IP aus seinem ursprünglichen Browser-Request und
  muss sie explizit mitgeben.

### memberType values

Same as public API: `private` | `farmer` | `municipality` | `company` | `association`

> PROJ-62 (May 2026): `sole_proprietor` was removed. The
> Kleinunternehmer-Pfad is now `company` with an empty `uidNumber`.
> Requests submitting `sole_proprietor` are rejected with 400.

### Required fields

`memberType`, `email`, `residentStreet`, `residentStreetNumber`, `residentZip`, `residentCity`,
`iban`, `accountHolder`, `privacyAccepted: true`, `meteringPoints` (min 1).

`sepaMandateAccepted` — context-dependent (PROJ-81): must be `true` unless the EEG has activated the SEPA-choice option (`sepaOptionalEnabled = true`) **AND** the member's `memberType` is in `sepaOptionalMemberTypes`. In that case, `false` is accepted; the application is stored with `einzugsart = "kein_sepa"` (no mandate PDF). Bank fields (`iban`, `accountHolder`) remain mandatory regardless.

For `natural_person` types (`private`, `farmer`): `firstname` + `lastname` required.
For legal entity types (`municipality`, `company`, `association`): `companyName` required.
- `company`: `uidNumber` and `registerNumber` are both **optional**. Empty
  `uidNumber` signals the Kleinunternehmerregelung (§ 6 Abs 1 Z 27 UStG,
  0 % USt.); a populated `uidNumber` puts the application on the regular
  20 % USt. path.
- `association`: `registerNumber` is optional.
- `municipality`: no additional fields beyond `companyName`.

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

## 9. Customer Onboarding API *(PROJ-71)*

EEG-Customer-Onboarding: ein bereits per Keycloak authentifizierter EEG-Admin
bucht die SaaS-Plattform aus den Einstellungen heraus. Alle Endpoints liegen
unter `/api/admin/customer-onboarding/*` und sind JWT-protected. Submit ist
Tenant-scoped (RC aus Claims), Approve/Reject/List/Detail sind Superuser-only.

Status-Lifecycle (linear):
- `submitted` — EEG-Admin hat das Formular abgeschickt, wartet auf Owner-Freischaltung
- `approved` — Owner hat freigeschaltet, Vertrag aktiv
- `owner_rejected` — Owner hat vor Approve abgelehnt

Soft-Suspend eines aktiven Vertrags (Owner-Reject nach Approve) ist KEIN
Submission-Status-Wechsel — die Submission bleibt `approved`, der
Vertragsstatus lebt im `customer_onboarding_event_log`.

## 9.1 Submit customer onboarding

### POST `/api/admin/customer-onboarding/submit`

Keycloak-JWT-protected. Der `rcNumber` im Body muss in der `RCNumbers`-Claim
des Aufrufers enthalten sein (oder Superuser). Variante B 2026-06-06:
Stammdaten kommen LIVE aus `registration_entrypoint` (PROJ-32-Sync), nur
der RC + die zwei Akzept-Booleans werden uebertragen.

Backend-Ablauf:
1. Laed Stammdaten aus `registration_entrypoint` (Pflicht — sonst 404).
2. Generiert AVV-PDF SYNCHRON aus den Live-Stammdaten.
3. Persistiert Submission im Status `submitted` zusammen mit dem PDF-Blob.
4. Feuert asynchron Owner-Notification-Mail an `CUSTOMER_ONBOARDING_OWNER_EMAIL`.

### Request body

```json
{
  "rcNumber": "RC-12345",
  "agbAccepted": true,
  "avvAccepted": true
}
```

`agbAccepted` und `avvAccepted` müssen beide `true` sein. Die Versionen werden
serverseitig aus `AGBVersion` / `AVVVersion`-Konstanten gestempelt.

### Response 201

```json
{ "submissionId": "11111111-2222-3333-4444-555555555555" }
```

### Errors
- `400` — validation error
- `403` — Tenant-Mismatch (RC nicht in Claims) oder fehlende Auth
- `404` — kein `registration_entrypoint`-Stub fuer die RC (Admin-Auto-Sync nicht gelaufen)
- `409` — bereits eine `submitted`- oder `approved`-Submission für diese RC
- `500` — PDF-Generierung fehlgeschlagen (kein Submission-Insert)

## 9.2 List customer onboarding submissions (admin)

### GET `/api/admin/customer-onboarding/submissions`

Superuser-only. Optional query param `status` (komma-separierte Liste) filtert
auf einen oder mehrere Lifecycle-Werte. Default: alle 3 Statuswerte.

### Response 200

```json
[
  {
    "id": "uuid",
    "rcNumber": "RC-12345",
    "vereinsname": "Musterbetrieb GmbH",
    "boardName": "Max Mustermann",
    "boardEmail": "max.mustermann@example.org",
    "status": "submitted",
    "agbVersion": "1.0",
    "avvVersion": "1.0",
    "submittedAt": "2026-06-06T10:00:00Z",
    "approvedAt": null,
    "rejectedAt": null
  }
]
```

## 9.3 Get customer onboarding detail (admin)

### GET `/api/admin/customer-onboarding/submissions/{id}`

Superuser-only. Liefert die Submission inkl. aktuellem Vertragsstatus aus dem
Event-Log.

### Response 200

```json
{
  "submission": {
    "id": "uuid",
    "rcNumber": "RC-12345",
    "status": "approved",
    "...": "...",
    "submittedBySubject": "keycloak-sub-of-tenant-admin",
    "approvedAt": "2026-06-06T10:08:00Z",
    "approvedBySubject": "keycloak-sub-of-owner"
  },
  "contract": {
    "active": true,
    "latestEventType": "activated",
    "latestEventAt": "2026-06-06T10:08:00Z",
    "latestReasonCode": "owner_approve",
    "latestReasonText": "",
    "latestActorKind": "human",
    "latestActorSubject": "keycloak-sub-of-owner",
    "suspendedSince": null
  }
}
```

## 9.4 Approve customer onboarding submission (admin)

### POST `/api/admin/customer-onboarding/submissions/{id}/approve`

Superuser-only. Schaltet `submitted` → `approved` in einer Transaktion:
1. AVV-PDF wird generiert (best-effort, bei Fehler ohne Anhang weiter)
2. `ApproveTx` atomar mit Advisory-Lock auf `rc_number`:
   - `customer_onboarding_submission.status='approved'`, `approved_at`, `approved_by_subject`
   - Event `activated` in `customer_onboarding_event_log` (`reason_code=owner_approve`, `actor_kind=human`)
   - `registration_entrypoint.is_active=true` — wenn der Stub fehlt → 404, Tx rückgerollt
3. Welcome-Mail an den Vorstand mit AVV-PDF im Anhang. Fehler werden geloggt,
   blockieren den Approve aber NICHT (BUG-2 Fix 2026-06-06 — die frühere Variante
   sendete VOR dem Commit und konnte bei Tx-Fail einen „aktiviert"-benachrichtigten,
   nicht-aktiven Vorstand erzeugen). Bei Mail-Fehler kann der Owner das
   AVV-PDF separat über 9.6a `/avv-pdf` erneut herunterladen.

### Request body

```json
{}
```

### Response 200

```json
{ "result": "ok" }
```

### Errors
- `404` — Submission nicht gefunden, oder `registration_entrypoint`-Stub fehlt
- `409` — Submission nicht im Status `submitted`

## 9.5 Reject customer onboarding submission (admin)

### POST `/api/admin/customer-onboarding/submissions/{id}/reject`

Superuser-only. Behavior depends on current status:
- **Pre-Approve** (`submitted`): Status-Transition auf `owner_rejected`,
  Reason + Keycloak-Subject werden persistiert. Kein Event geschrieben.
- **Post-Approve** (`approved` mit aktivem Vertrag): atomar
  - Event `suspended` mit `reason_code='owner_decision'` ins Event-Log
  - `registration_entrypoint.is_active=false` (sperrt zusätzlich die
    Public-Member-Onboarding-Form, Soft-Suspend-Guards greifen)
  - Submission bleibt `approved` als historischer Beleg

Bei `notifyMember=true` wird eine Reject-Mail an den Vorstand mit Hard-Fail-
Semantik verschickt — Mail-Fehler bricht den Reject ab.

### Request body

```json
{ "reason": "Plausibilitätsprüfung negativ", "notifyMember": true }
```

### Response 200

```json
{ "result": "ok" }
```

### Errors
- `400` — Reason fehlt
- `404` — Submission nicht gefunden
- `409` — Submission im falschen Status oder Vertrag bereits suspendiert
- `500` — Reject-Mail-Fehler (Tx rückgerollt)

## 9.6 Get tenant onboarding status (admin)

### GET `/api/admin/customer-onboarding/status?rc_number=...`

Liefert die Status-Card-Daten für den `/admin/settings/`-Bereich. Tenant-Admin
sieht seine eigene erste RC aus den Claims; Superuser kann via Query-Param
`rc_number` jede beliebige RC abfragen.

### Response 200

Fünf mögliche States:

```json
{ "state": "none" }
```

```json
{
  "state": "submitted",
  "boardEmail": "max.mustermann@example.org",
  "boardName": "Max Mustermann",
  "submittedAt": "2026-06-06T10:00:00Z"
}
```

```json
{
  "state": "approved",
  "boardEmail": "...",
  "boardName": "...",
  "boardPhone": "+43 1 234567",
  "agbVersion": "1.0",
  "agbAcceptedAt": "2026-06-06T10:00:00Z",
  "avvVersion": "1.0",
  "avvAcceptedAt": "2026-06-06T10:00:00Z",
  "submittedAt": "...",
  "approvedAt": "...",
  "submissionId": "uuid"
}
```

```json
{
  "state": "suspended",
  "rejectedAt": "...",
  "rejectionReason": "Plausibilitätsprüfung negativ",
  "submissionId": "uuid"
}
```

```json
{
  "state": "owner_rejected",
  "rejectedAt": "...",
  "rejectionReason": "Plausibilitätsprüfung negativ"
}
```

## 9.6a Download AVV acceptance PDF *(BUG-1 Fix 2026-06-06)*

### GET `/api/admin/customer-onboarding/submissions/{id}/avv-pdf`

Liefert das AVV-Akzept-PDF einer `approved`-Submission. Wird beim Approve einmalig
als Welcome-Mail-Anhang versendet; dieser Endpoint erlaubt Self-Service-Re-Download.

Zugriff: Superuser ODER Tenant-Admin der RC der Submission. PDF wird deterministisch
aus den persistierten Submission-Daten regeneriert.

### Response 200

- Content-Type: `application/pdf`
- Content-Disposition: `attachment; filename="AVV-Akzept-<rcnumber>.pdf"`

### Errors
- `403` — Caller ist weder Superuser noch Tenant-Admin der RC
- `404` — Submission nicht gefunden
- `409` — Submission nicht im Status `approved` (kein PDF für `submitted`/`owner_rejected`)

## 9.7 Soft-Suspend guard *(PROJ-71)*

Sechs bestehende Admin-Endpoints sind Soft-Suspend-geschützt. Wenn der
Vertrag der adressierten EEG aktuell `suspended` ist, antworten sie mit `403`
und einer generischen Fehlermeldung:

- `POST /api/admin/applications/{id}/import`
- `GET  /api/admin/applications/export` (Excel)
- `GET  /api/admin/applications/{id}/approval-pdf`
- `POST /api/admin/reconciliation/run` (PROJ-69)
- `POST /api/admin/eeg/resync-from-core` (PROJ-32)
- `POST /api/admin/sepa-mandate/renewal`

Zusätzlich wird beim Soft-Suspend `registration_entrypoint.is_active=false`
gesetzt — damit sperrt auch die Public-Member-Onboarding-Form (`/onboarding`)
für die suspendierte RC.

Der Contract-Checker ist fail-open: bei DB-Lookup-Fehler wird der Endpoint
durchgelassen (Verfügbarkeit > Strenge), Fehler wird geloggt.

---

## PROJ-104 Platform Billing API (Welle 1+2+3+4)

Alle Owner-Endpoints unter `/api/admin/billing/*` sind **Superuser-only**
(per-Handler `requireSuperuser`-Check). EEG-Admin sieht eine eigene Read-
Only-Liste unter `/api/admin/eeg/{rc}/invoices` (Tenant-scoped via
`containsRC`).

### Owner-Endpoints

#### Pricing-Pläne

`GET /api/admin/billing/pricing-plans` → `{ plans: [BillingPricingPlan] }`

`POST /api/admin/billing/pricing-plans`
Body: `{ edition: "standard"|"pro", eurPerActiveMemberPerQuarter: number,
vatPercent: number, gueltigAb: "YYYY-MM-DD" }`
Response: `{ id: string }` + Audit-Log-Eintrag `pricing_plan_versioned`.

#### EEGs

`GET /api/admin/billing/eegs` → `{ eegs: [BillingEEGState] }` mit RC,
EEG-Name, Edition, BillingLive, MollieMandateActive, TrialStartedAt,
Vendor-Customer-IDs.

`GET /api/admin/billing/eegs/{rc}/pre-flight` → Pre-Flight-Check für
Owner-UI vor Live-Toggle. Liefert `HasZeroPricing`, `CurrentEurPerMember`,
`MandateActive`, `BillingLive`, Vendor-Customer-Status.

`POST /api/admin/billing/eegs/{rc}/billing-live`
Body: `{ live: boolean, acceptZeroPricing?: boolean }`
- Wenn `live=true` und `HasZeroPricing=true`: 400 ohne `acceptZeroPricing: true`
- Wenn `live=true`: **sync hard-fail** auf `SendBillingMandateSetup`-Mail
  an EEG-Vorstand (Memory `feedback_mail_hard_fail`). Anschließend
  `SetBillingLive` + Mandate-Setup-Trigger (Mollie EUR 0,01-First-Payment).
- Response: `{ billingLive, mollieMandateActive?, mandateSetupPaymentId? }`
- Audit-Log: `billing_live_flipped`

`POST /api/admin/billing/eegs/{rc}/edition`
Body: `{ edition: "standard"|"pro" }`
Owner-Override-Pfad. Synct `settings_view_mode` (Pro→advanced, Standard→standard).
Audit-Log: `edition_switched`.

`POST /api/admin/billing/eegs/{rc}/trigger`
Manual-Trigger für das letzte abgeschlossene Quartal (R-16). Idempotent
via `UNIQUE(rc_number, year, quarter)`. Audit-Log: `manual_trigger`.

#### Rechnungen

`GET /api/admin/billing/invoices` → `{ invoices: [BillingInvoice] }`
(MVP: `ListOpenForSync`)

`GET /api/admin/billing/invoices/{id}` → `BillingInvoice` Detail.

`POST /api/admin/billing/invoices/{id}/credit-note`
Body: `{ reason: string }` (Pflicht, min. 10 Zeichen UI-seitig, jeglicher
nicht-leere String backend-seitig)
- Original-Status muss `sent | paid | overdue` sein
- Schreibt neue Zeile mit `status='credit_note'` + `cancels_invoice_id=originalID`
- Audit-Log: `credit_note_issued`

#### Audit-Log

`GET /api/admin/billing/audit-log?kind=…&rc=…&limit=…`
→ `{ events: [BillingAuditEvent] }`

### EEG-Admin-Endpoint

`GET /api/admin/eeg/{rc}/invoices` → `{ invoices: [EEGInvoiceItem] }`
Tenant-scoped Read-Only (`containsRC`-Check). Liefert Quartal, Status,
Brutto, Versanddatum, Bezahltdatum, Rechnungsnummer.

### Webhook

`POST /api/webhooks/mollie` (public, IP-Allowlist + Defense-in-Depth)
- Body: form-encoded `id=tr_xxx` (Mollie-Convention)
- Handler ruft `mollie.GetPayment(id)` (Authentizitätscheck — Grilling R-17)
- Status `paid` → `billing_invoice.MarkPaid` + ggf. PROJ-71-Reactivation
- Status `failed | canceled | expired` → `status='cancelled'`
- Status `charged_back` → `mollie_mandate_active=false` + Owner-Alert-Mail
- Unknown `tr_xxx` → **200 OK** + Audit-Log `unknown_payment_webhook`
  (Mollie würde sonst retry-storm machen)
- Source-IP nicht in Allowlist → 403
- Aktivierung: nur wenn `cfg.Billing.MollieWebhookURL` gesetzt ist

### Cron-Subcommands

- `./server billing-quarterly` — letztes abgeschlossenes Quartal pro
  aktivem EEG abrechnen. K8s-CronJob Schedule `"0 4 1 1,4,7,10 *"`,
  `concurrencyPolicy: Forbid`, `startingDeadlineSeconds: 14400` (4h Toleranz).
- `./server billing-daily` — kombiniert Status-Sync (Mollie-GET für offene
  Rechnungen, Webhook-Backstop) und Overdue-Check (`sent_at` > 14 Tage +
  PROJ-71 Soft-Suspend via `ReasonPaymentFailed`). K8s-CronJob Schedule
  `"0 5 * * *"`.

### Audit-Log-Kinds

| Kind | Wann |
|---|---|
| `edition_switched` | EEG-Admin oder Owner ändert `eeg_edition` |
| `billing_live_flipped` | Owner toggelt `billing_live` (Welle 4a) |
| `pricing_plan_versioned` | Owner legt neue `pricing_plan`-Zeile an |
| `credit_note_issued` | Owner erzeugt Gutschrift |
| `manual_trigger` | Owner triggert Quartals-Abrechnung manuell |
| `trial_started` | Service setzt `trial_started_at` beim ersten `activated_at` |
| `virtual_trial_grace_applied` | Pricing-Service nutzt `cfg.DeployedAt` als Anker (Bestand-EEG ohne `trial_started_at`) |
| `unknown_payment_webhook` | Mollie-Webhook mit `tr_xxx` nicht in unserer DB |
| `mollie_chargeback` | Chargeback-Event → Mandate deaktiviert |
| `scheduler_run` | Quartals-Cron hat eine EEG bearbeitet (Outcome: sent/preview/draft/skip) |
| `overdue_marked` | Daily-Cron hat `sent` auf `overdue` geschoben + PROJ-71-Suspend ausgelöst |
| `payment_received` | Daily-Sync oder Webhook hat `paid` verarbeitet |
| `mandate_activated` | EUR 0,01-First-Payment ist `paid` → `mollie_mandate_active=true` |

### Owner-Mails (Welle 4a)

- `billing_mandate_setup` — sync vor Toggle-Flip, hard-fail wenn Mail
  scheitert → `billing_live` bleibt unverändert.
- `billing_chargeback_owner_alert` — best-effort async vom Webhook-Handler
  über `Scheduler.SetChargebackMailer`.
- `billing_credit_note` — best-effort async nach Gutschrift-Erzeugung.
