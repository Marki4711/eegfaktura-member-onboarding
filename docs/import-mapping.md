# Import Mapping
## Member Onboarding -> eegFaktura Participant

## 1. Goal

An approved onboarding application is transformed on import into a participant payload that aligns with the existing eegFaktura participant structure.

Important:
- the onboarding model remains reduced
- the participant model may be more extensive
- fields not managed in onboarding are filled with empty values or defaults
- tariffs, roles, and account information are not managed in onboarding in V1

The existing participant structure contains, among others:
- `firstname`
- `lastname`
- `residentAddress`
- `billingAddress`
- `contact`
- `accountInfo`
- `businessRole`
- `role`
- `tariffId`
- `meters[]`

---

## 2. Source tables

- `member_onboarding.application`
- `member_onboarding.metering_point`

---

## 3. Field mapping table

| Source | Target | Required | Default / Rule | Comment |
|---|---|---:|---|---|
| `application.firstname` | `firstname` | yes | – | direct |
| `application.lastname` | `lastname` | yes | – | direct |
| `application.email` | `contact.email` | yes | – | direct |
| `application.phone` | `contact.phone` | no | `""` | direct |
| `application.resident_street` | `residentAddress.street` | yes | – | direct |
| `application.resident_street_number` | `residentAddress.streetNumber` | yes | – | direct |
| `application.resident_zip` | `residentAddress.zip` | yes | – | direct |
| `application.resident_city` | `residentAddress.city` | yes | – | direct |
| `application.resident_street` | `billingAddress.street` | yes | identical to resident address | V1 rule |
| `application.resident_street_number` | `billingAddress.streetNumber` | yes | identical to resident address | V1 rule |
| `application.resident_zip` | `billingAddress.zip` | yes | identical to resident address | V1 rule |
| `application.resident_city` | `billingAddress.city` | yes | identical to resident address | V1 rule |
| – | `residentAddress.type` | yes | `RESIDENCE` | set technically |
| – | `billingAddress.type` | yes | `BILLING` | set technically |
| `metering_point.metering_point` | `meters[].meteringPoint` | yes | – | per record |
| `metering_point.direction` | `meters[].direction` | yes | – | per record |
| `application.resident_street` | `meters[].street` | yes | member address | V1 rule |
| `application.resident_street_number` | `meters[].streetNumber` | yes | member address | V1 rule |
| `application.resident_zip` | `meters[].zip` | yes | member address | V1 rule |
| `application.resident_city` | `meters[].city` | yes | member address | V1 rule |
| `application.privacy_accepted` | `consents.privacyAccepted` | no | optional in adapter | only if core service uses it |
| `application.privacy_version` | `consents.privacyVersion` | no | optional in adapter | only if core service uses it |
| `application.privacy_accepted_at` | `consents.privacyAcceptedAt` | no | optional in adapter | only if core service uses it |
| `application.accuracy_confirmed` | `consents.accuracyConfirmed` | no | optional in adapter | only if core service uses it |
| `application.communication_consent` | `consents.communicationConsent` | no | optional in adapter | only if core service uses it |

---

## 4. Technical defaults

These fields are not managed in onboarding but are set technically during import.

| Target | Default / Rule | Comment |
|---|---|---|
| `id` | empty | core generates ID |
| `participantNumber` | empty or core-generated | to be confirmed with core |
| `participantSince` | current import timestamp | set technically |
| `status` | `NEW` | aligned with existing structure |
| `titleBefore` | `""` | not managed in V1 |
| `titleAfter` | `""` | not managed in V1 |
| `optionals.website` | `""` | not managed in V1 |
| `meters[].status` | `INIT` | technical default |
| `meters[].processState` | `NEW` | technical default |
| `meters[].participantId` | empty | set after core creation |
| `meters[].registeredSince` | current import timestamp | set technically |
| `meters[].partFact` | `100` | technical default, if required |
| `meters[].participantState` | `{}` or core default | to be confirmed with core |

---

## 5. Intentionally not managed in V1

These fields are not managed in Member Onboarding.

| Target field | Behavior in V1 |
|---|---|
| `accountInfo.*` | empty/default |
| `businessRole` | empty/default or core default |
| `role` | empty/default or core default |
| `taxNumber` | empty |
| `vatNumber` | empty |
| `tariffId` | empty |
| `meters[].tariff_id` | `null` |
| `meters[].gridOperatorName` | empty |
| `meters[].gridOperatorId` | empty |

These fields are added directly in eegFaktura after import or set by core default logic.

---

## 6. Example target payload for V1

```json
{
  "id": "",
  "participantNumber": "",
  "participantSince": "2026-04-18T12:00:00Z",
  "firstname": "Josef",
  "lastname": "Brandstätter",
  "status": "NEW",
  "titleBefore": "",
  "titleAfter": "",
  "residentAddress": {
    "street": "Musterstraße",
    "type": "RESIDENCE",
    "city": "Musterstadt",
    "streetNumber": "2",
    "zip": "1010"
  },
  "billingAddress": {
    "street": "Musterstraße",
    "type": "BILLING",
    "city": "Musterstadt",
    "streetNumber": "2",
    "zip": "1010"
  },
  "contact": {
    "email": "max@example.org",
    "phone": "0664/1234567"
  },
  "accountInfo": {
    "iban": "",
    "owner": "",
    "sepa": false,
    "bankName": "",
    "mandateReference": "",
    "sepaDirectDebit": "",
    "mandateDate": null
  },
  "businessRole": "",
  "role": "",
  "optionals": {
    "website": ""
  },
  "taxNumber": "",
  "vatNumber": "",
  "tariffId": "",
  "meters": [
    {
      "status": "INIT",
      "processState": "NEW",
      "participantId": "",
      "meteringPoint": "AT0031000000000000000000990022105",
      "direction": "CONSUMPTION",
      "registeredSince": "2026-04-18T12:00:00Z",
      "gridOperatorName": "",
      "gridOperatorId": "",
      "partFact": 100,
      "tariff_id": null,
      "street": "Musterstraße",
      "streetNumber": "2",
      "zip": "1010",
      "city": "Musterstadt",
      "participantState": {}
    }
  ]
}
```

This target payload is deliberately kept close to the visible participant structure.

---

## 7. Open points (resolved 2026-05-08/09 against deployed core)

The following were validated against the running eegFaktura core during the
PROJ-4 end-to-end test and from `eegfaktura/eegfaktura-backend` source.

| # | Question | Resolution |
|---|---|---|
| 1 | `participantNumber` generated by core? | Optional. We send our `member_number` (Mitgliedsnummer) as a string when present, otherwise empty — core does **not** auto-generate. |
| 2 | May `businessRole` and `role` be empty? | **No.** Empty `businessRole` makes the eegFaktura UI render the participant under the Privat tab even for companies (frontend uses `EEG_PRIVATE` / `EEG_BUSINESS` to switch views). See §8 below. |
| 3 | May `accountInfo` fields be empty? | Yes. `BankInfo.Iban` and `Owner` are `null.String` on the core model. |
| 4 | Are `meters[].gridOperatorName/Id` required? | **Not on the meter.** They live on the `Eeg` entity and are looked up server-side via the `tenant` HTTP header. Onboarding sends nothing. |
| 5 | Must `participantState` be provided? | No. The core's `RegisterParticipant` overwrites the participant `status` to `PENDING` regardless of input. We send `"NEW"` for symmetry; it is ignored. |
| 6 | Is `partFact` required? | **Yes — this is the Teilnahmefaktor in the eegFaktura UI.** We send `metering_point.participation_factor` (defaults to 100, configurable per metering point). |
| 7 | Is `processState` required on meter? | Yes, must be `"NEW"` on import. Confirmed from `eegfaktura-web/src/models/meteringpoint.model.ts` (`MeteringProcessStateType`). |

Additional findings:

- **Path prefix:** the deployed core API is mounted under `/api/*` (the `eegfaktura-web` `.env.production` confirms `VITE_API_SERVER_URL='/api'`). `CORE_BASE_URL` in our Helm values must include the `/api` suffix.
- **Auth:** the core's JWT middleware accepts any token signed by the realm `EEGFaktura`, regardless of `azp` (client_id). User-token forwarding from member-onboarding's NextAuth client works directly. The `tenant` HTTP header value must appear in the JWT's `Tenants` claim, otherwise 403.
- **Tenant claim format:** the core's middleware does a strict `json.Unmarshal` of `tenant` into `[]string`. If Keycloak emits `tenant` as a stringified-JSON-array (Claim JSON Type = `String`), the core returns **401** with an empty body. Set the `tenant` mapper's Claim JSON Type to **`JSON`** in the `eegfaktura-member-onboarding` client.

---

## 8. Member type → core role mapping

Onboarding `application.member_type` controls how the participant is sent to the core. The eegFaktura UI uses `businessRole` to switch between the Privat and Firma view.

| onboarding `member_type` | core `businessRole` | `firstName` | `lastName` |
|---|---|---|---|
| `private` | `EEG_PRIVATE` | `application.firstname` | `application.lastname` |
| `farmer` | `EEG_PRIVATE` | `application.firstname` | `application.lastname` |
| `company` | `EEG_BUSINESS` | `application.company_name` | empty (eegFaktura convention) |
| `association` | `EEG_BUSINESS` | `application.company_name` | empty |
| `municipality` | `EEG_BUSINESS` | `application.company_name` | empty |

`role` is always `EEG_USER`.

For non-natural-person types the company name is placed in `firstName` only because the core's `EegParticipant` schema has no separate company-name column and `firstname` is `NOT NULL`. If onboarding has collected a contact-person `firstname`/`lastname` even for company types, those take precedence and the company name is NOT injected.

---

## 9. Meter direction enum

Onboarding constants do not match the core's enum 1:1:

| onboarding `metering_point.direction` | core `meters[].direction` |
|---|---|
| `CONSUMPTION` | `CONSUMPTION` |
| `PRODUCTION` | **`GENERATION`** ← translated by `mapMeterDirection` |

Sending `PRODUCTION` directly leaves Einspeisungs-Zählpunkte with an unrecognised direction in the core (silently accepted but breaks billing).
