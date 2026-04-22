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

## 7. Open points to confirm with eegFaktura Core

These points should be clarified before the final implementation:

1. Is `participantNumber` generated by the core?
2. May `businessRole` and `role` be empty?
3. May `accountInfo` fields be empty?
4. Are `meters[].gridOperatorName` and `meters[].gridOperatorId` required?
5. Must `participantState` be explicitly provided or does the core set it itself?
6. Is `partFact = 100` the correct technical default?
