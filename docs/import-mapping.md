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
| `metering_point.installation_name` | `meters[].equipmentName` | no | omitempty when null/empty | Bezeichnung des Zählpunkts (siehe Public-Form-Popover „Hauptanlage, Nebengebäude, …"). Wird auf der Faktura-Rechnung ausgewiesen. Regression-Test: `TestBuildPayload_InstallationNameMapsToEquipmentName` |
| `metering_point.installation_number` | `meters[].equipmentNumber` | no | omitempty when null/empty | Anlagen-Nr. |
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

## 3.1 SEPA-Typ-Mapping beim Core-Import (PROJ-79 → PROJ-91)

`application.einzugsart` wird beim Aufbau des Core-Payloads in
`accountInfo.sepaDirectDebit` (Core-Enum, uppercase) übersetzt. Die
Mapping-Funktion ist `mapEinzugsart` in `internal/importing/payload.go`.

| Onboarding `application.einzugsart` | Core `accountInfo.sepaDirectDebit` | Core `accountInfo.sepa` |
|---|---|---|
| `core` (Basislastschrift) | `"CORE"` | `true` |
| `b2b` (Firmenlastschrift) | `"B2B"` | `true` |
| `kein_sepa` | `""` (omitempty) | `false` |
| sonst (leer, unbekannt) | `""` (omitempty) | `false` |

### Geschichte: PROJ-79 → PROJ-91

PROJ-79 (2026-06-08) hatte `b2b` zunächst **bewusst auf `CORE`** gemappt
— um die Bank-Klärungs-Phase mit dem CORE-Schutz zu überbrücken. PROJ-91
(2026-06-09) hat diesen Heimlich-Mapping zurückgerollt und durch einen
expliziten Admin-Toggle ersetzt:

- **`einzugsart=b2b`** geht jetzt wieder direkt mit SEPA-B2B in den Core
  (wie vor PROJ-79). Erste Lastschrift kann nur ausgeführt werden, wenn
  die Hausbank des Mitglieds das B2B-Mandat registriert hat — kein
  Rückbuchungsrisiko nach erfolgreicher Lastschrift.
- **`einzugsart=core` + `prepare_b2b_documents=true`** ist der neue
  Vorbereitungs-Pfad (PROJ-91): Antrag wird mit SEPA-CORE im Core
  angelegt → Lastschriften starten sofort, das Mitglied bekommt aber
  schon das B2B-Mandat-PDF zum Unterschreiben und Vorlegen bei seiner
  Bank. Nach Bank-Bestätigung stellt der Admin den SEPA-Typ im Core
  manuell auf B2B um.

### Workflow-Hinweis in der Mail (PROJ-91)

Wenn `prepare_b2b_documents=true` ist, geht in der Mandat-Mail an
Mitglied + EEG-Kontaktperson ein gelber **Hinweis-Banner** mit:

> **B2B-Vorbereitung:** Das Konto ist im eegFaktura-Core als SEPA-CORE
> angelegt. Sobald die Hausbank des Mitglieds das B2B-Mandat bestätigt
> hat, stellen Sie den SEPA-Typ im Core manuell auf B2B um.

Der Banner-HTML lebt zentral in
`internal/mail/b2b_notice.go` — Single source of truth für die zwei
Mail-Pfade (Member-Variante + EEG-Variante).

### Bestand-Migration

Migration 000074 (PROJ-91) hat alle Bestand-Anträge mit `einzugsart=b2b`
auf `einzugsart=core` + `prepare_b2b_documents=true` umgestellt — sie
verhalten sich seither wie der neue Vorbereitungs-Pfad. Im Faktura-Core
schon importierte B2B-Anlagen wurden nicht angefasst; laufende
B2B-Lastschriften mit aktivem Bank-Mandat bleiben unberührt.

### Praktische Trigger-Realität

`einzugsart="b2b"` entsteht heute in der Production-Codebase
**ausschließlich durch manuellen Admin-Edit** im Antrags-Detail
(`internal/application/admin_service.go`). Public-Submit und Externe
API erzeugen nur `core` oder `kein_sepa`. Der Vorbereitungs-Toggle
`prepare_b2b_documents` ist ebenfalls Admin-only und nur bei
`einzugsart=core` sichtbar.

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
| `meters[].partFact` | `100` | Teilnahmefaktor in % from `metering_point.participation_factor`; this is the value the eegFaktura UI's "Teilnehmer Faktor" input shows |
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

Onboarding-only fields (intentionally **not** sent to core):
- `application.cooperative_shares_count` *(PROJ-37)* — cooperative shares are bookkeeping inside the EEG and have no representation in the core's participant model.
- `application.network_operator_authorization` + `network_operator_authorization_at` *(PROJ-44)* — Vollmacht für die EEG, mit dem Netzbetreiber zu agieren. Keine Core-Repräsentation; lokal in `application` als Audit-Trail aufbewahrt.
- `application.activated_at` *(PROJ-46)* — Audit-Timestamp für den Onboarding-Status `activated`. Nicht im Core. *(`bank_confirmed_at` wurde mit PROJ-91 deprecated; neuer Code schreibt das Feld nicht mehr, die Spalte bleibt als historischer Beleg für migrierte Bestand-Anträge.)*
- `metering_point.generation_type` / `battery_size_kwh` / `inverter_manufacturer` *(PROJ-45)* — EEG-Optimierungs-Metadaten. Werden lokal gespeichert (Excel-Export für eegFaktura-Importer am Spalten-Ende ergänzt), gehen aber nicht über die JSON-`POST /participant`-API mit (Core kennt diese Felder noch nicht).

### Activation-Check (PROJ-46 Stage D + PROJ-53, reverse-read)

`POST /api/admin/applications/check-activation` ruft pro Tenant `GET /participant` im Core auf. Das Auslöse-Kriterium pro EEG ist konfigurierbar via `registration_entrypoint.activation_mode` (PROJ-53):

- **`participant_active`** (Default): liest `participant.status` (`NEW` / `PENDING` / `ACTIVE`); transitioniert bei `ACTIVE` auf `activated`.
- **`any_meter_registration_started`** (PROJ-53): liest zusätzlich `participant.meters[].processState`; transitioniert sobald mindestens ein Zählpunkt `processState ∈ {PENDING, APPROVED, ACTIVE}` hat (Netzbetreiber hat die EDA-Online-Registrierung mindestens bestätigt).

Keine Schreib-Aktion gegen den Core, nur ein Lese-Pfad zum Status-Sync. Bei erfolgreichem Wechsel auf `activated` läuft asynchron `SendActivationNotification` (volle Beitrittsbestätigungs-Mail mit PDF), idempotent via `application.activation_notification_sent_at`.

Der direkte Skip `approved → activated` (PROJ-53, `POST /api/admin/applications/{id}/mark-activated`) ist nicht Teil dieses Lesepfads — er ist eine reine Onboarding-Operation für den Ausnahmefall „Core-Member bereits manuell überschrieben".

### Reverse integration: EEG master data sync (PROJ-32 / PROJ-33)

The opposite direction — sync **from** the core **to** the onboarding tool — is documented separately. Fields synced into `registration_entrypoint` (EEG name, address, contact, IBAN, CreditorID, logo bytes) read the core via the GraphQL `query { eeg }` scalar in the **user context** (bearer-forwarding, no service account). These fields become read-only in the onboarding admin UI; only the sync endpoint can change them. Single source of truth lives in the core.

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
| 6 | Is the Teilnahmefaktor required? | **Yes — JSON field is `partFact`** in the participant payload. We send `metering_point.participation_factor` (defaults to 100). The deployed core stores it and the eegFaktura UI's "Teilnehmer Faktor in %" reads it back via the GET response. NB: the GET response also exposes a separate `allocationFactor` field (DB column `base.meteringpoint.allocation_factor`) — that's the clearing/distribution factor, not the UI input field, and we do not populate it on import. |
| 7 | Is `processState` required on meter? | Yes, must be `"NEW"` on import. Confirmed from `eegfaktura-web/src/models/meteringpoint.model.ts` (`MeteringProcessStateType`). |

Additional findings:

- **Path prefix:** the deployed core API is mounted under `/api/*` (the `eegfaktura-web` `.env.production` confirms `VITE_API_SERVER_URL='/api'`). `CORE_BASE_URL` in our Helm values must include the `/api` suffix.
- **Auth:** the core's JWT middleware accepts any token signed by the realm `EEGFaktura`, regardless of `azp` (client_id). User-token forwarding from member-onboarding's NextAuth client works directly. The `tenant` HTTP header value must appear in the JWT's `Tenants` claim, otherwise 403.
- **Tenant claim format:** the core's middleware does a strict `json.Unmarshal` of `tenant` into `[]string`. If Keycloak emits `tenant` as a stringified-JSON-array (Claim JSON Type = `String`), the core returns **401** with an empty body. Set the `tenant` mapper's Claim JSON Type to **`JSON`** in the `eegfaktura-member-onboarding` client.
- **SEPA mandate at import (2026-05-28):** when `einzugsart=b2b` OR (`einzugsart=core` AND `entrypoint.sepa_mandate_at_import=true`), the import path now derives `accountInfo.mandateReference = memberNumber` and `accountInfo.mandateDate = importStartedAt` **before** the Core POST, so the values land in eegFaktura. Gated by `shouldDeriveMandateAtImport` in `internal/importing/import_service.go`; idempotent via the `SetMandate*IfEmpty` repo methods so an admin-overridden mandate reference is preserved. The submit-time flow (`einzugsart=core` + `sepa_mandate_at_import=false`, PROJ-12) intentionally leaves both fields empty at import — the reference is communicated to the member via the activation mail's hint block and persisted only when the admin enters it after the signed paper mandate comes back.

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

For non-natural-person types the company name is placed in `firstName` only because the core's `EegParticipant` schema has no separate company-name column and `firstname` is `NOT NULL`. For `company`/`association`/`municipality`, if onboarding has collected a contact-person `firstname`/`lastname`, those take precedence and the company name is NOT injected.

> PROJ-62 (May 2026): `sole_proprietor` was merged into `company`.
> Ex-Kleinunternehmer applications now arrive as `company` with an empty
> `uidNumber`. The mapping is identical to other company applications —
> the absent UID is preserved and the core distinguishes the tax mode
> from the UID presence.

---

## 9. Meter direction enum

Onboarding constants do not match the core's enum 1:1:

| onboarding `metering_point.direction` | core `meters[].direction` |
|---|---|
| `CONSUMPTION` | `CONSUMPTION` |
| `PRODUCTION` | **`GENERATION`** ← translated by `mapMeterDirection` |

Sending `PRODUCTION` directly leaves Einspeisungs-Zählpunkte with an unrecognised direction in the core (silently accepted but breaks billing).

---

## 10. Tariff handling (PROJ-27)

Tariffs are not persisted in the onboarding DB. The admin picks them at
import time in a popup; the selection is sent in the body of
`POST /api/admin/applications/{id}/import`:

```json
{
  "tariffId": "<uuid>",
  "meterTariffs": {
    "AT0030000000000000000000012345678": "<uuid>"
  }
}
```

Mapping into the core:

| Field | Where it lands in the core | Mechanism |
|---|---|---|
| `meterTariffs[<mp>]` | `meters[].tariff_id` in `POST /participant` body | direct, snake_case JSON tag |
| `tariffId` (member) | `base.participant."tariffId"` column | follow-up `PUT /participant/v2/{id}` with `{"path": "tariffId", "value": "<uuid>"}` |

The follow-up call is necessary because the core's
`EegParticipantBase.TariffId` is `goqu:"skipinsert"` — sending `tariffId` in
the `POST /participant` body is silently ignored.

A failure of the follow-up call sets `ImportResult.MemberTariffWarning` but
does NOT roll back the import (meter tariffs are already persisted; the
admin can re-assign the member tariff in eegFaktura).

Tariff types (from the core's `base.tariff.type` column):

| Type | Verwendung im Onboarding |
|---|---|
| `EEG` | Mitglieds-Tarif (Application-Level) |
| `VZP` | Verbraucher-Zählpunkt (`direction = CONSUMPTION`) |
| `EZP` | Einspeise-Zählpunkt (`direction = GENERATION`) |
| `AKONTO` | Wird im Onboarding nicht angeboten — Prepaid-Konto-Tarif |

Inactive tariffs (`inactiveSince != null`) are filtered out by the frontend.
