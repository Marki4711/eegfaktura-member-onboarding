# Import Mapping
## Member Onboarding -> eegFaktura Participant

## 1. Goal

Ein freigegebener Onboarding-Antrag wird beim Import in ein Participant-Payload transformiert, das sich an der bestehenden eegFaktura-Teilnehmerstruktur orientiert.

Wichtig:
- das Onboarding-Modell bleibt reduziert
- das Participant-Modell darf umfangreicher sein
- nicht gepflegte Felder werden leer oder mit Defaultwerten befüllt
- Tarife, Rollen und Kontoinformationen werden in V1 nicht im Onboarding gepflegt

Die bestehende Participant-Struktur enthält u. a.:
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

| Source | Target | Pflicht | Default / Regel | Kommentar |
|---|---|---:|---|---|
| `application.firstname` | `firstname` | ja | – | direkt |
| `application.lastname` | `lastname` | ja | – | direkt |
| `application.email` | `contact.email` | ja | – | direkt |
| `application.phone` | `contact.phone` | nein | `""` | direkt |
| `application.resident_street` | `residentAddress.street` | ja | – | direkt |
| `application.resident_street_number` | `residentAddress.streetNumber` | ja | – | direkt |
| `application.resident_zip` | `residentAddress.zip` | ja | – | direkt |
| `application.resident_city` | `residentAddress.city` | ja | – | direkt |
| `application.resident_street` | `billingAddress.street` | ja | identisch zu Wohnadresse | V1-Regel |
| `application.resident_street_number` | `billingAddress.streetNumber` | ja | identisch zu Wohnadresse | V1-Regel |
| `application.resident_zip` | `billingAddress.zip` | ja | identisch zu Wohnadresse | V1-Regel |
| `application.resident_city` | `billingAddress.city` | ja | identisch zu Wohnadresse | V1-Regel |
| – | `residentAddress.type` | ja | `RESIDENCE` | technisch gesetzt |
| – | `billingAddress.type` | ja | `BILLING` | technisch gesetzt |
| `metering_point.metering_point` | `meters[].meteringPoint` | ja | – | pro Datensatz |
| `metering_point.direction` | `meters[].direction` | ja | – | pro Datensatz |
| `application.resident_street` | `meters[].street` | ja | Mitgliedsadresse | V1-Regel |
| `application.resident_street_number` | `meters[].streetNumber` | ja | Mitgliedsadresse | V1-Regel |
| `application.resident_zip` | `meters[].zip` | ja | Mitgliedsadresse | V1-Regel |
| `application.resident_city` | `meters[].city` | ja | Mitgliedsadresse | V1-Regel |
| `application.privacy_accepted` | `consents.privacyAccepted` | nein | optional im Adapter | nur falls Core-Service es nutzt |
| `application.privacy_version` | `consents.privacyVersion` | nein | optional im Adapter | nur falls Core-Service es nutzt |
| `application.privacy_accepted_at` | `consents.privacyAcceptedAt` | nein | optional im Adapter | nur falls Core-Service es nutzt |
| `application.accuracy_confirmed` | `consents.accuracyConfirmed` | nein | optional im Adapter | nur falls Core-Service es nutzt |
| `application.communication_consent` | `consents.communicationConsent` | nein | optional im Adapter | nur falls Core-Service es nutzt |

---

## 4. Technical defaults

Diese Felder werden nicht im Onboarding gepflegt, aber beim Import technisch gesetzt.

| Target | Default / Regel | Kommentar |
|---|---|---|
| `id` | leer | Core erzeugt ID |
| `participantNumber` | leer oder Core-generiert | noch final mit Core klären |
| `participantSince` | aktueller Importzeitpunkt | technisch gesetzt |
| `status` | `NEW` | passend zur bestehenden Struktur |
| `titleBefore` | `""` | V1 nicht gepflegt |
| `titleAfter` | `""` | V1 nicht gepflegt |
| `optionals.website` | `""` | V1 nicht gepflegt |
| `meters[].status` | `INIT` | technischer Default |
| `meters[].processState` | `NEW` | technischer Default |
| `meters[].participantId` | leer | wird nach Core-Anlage gesetzt |
| `meters[].registeredSince` | aktueller Importzeitpunkt | technisch gesetzt |
| `meters[].partFact` | `100` | technischer Default, falls erforderlich |
| `meters[].participantState` | `{}` oder Core-Default | final mit Core klären |

---

## 5. Intentionally not managed in V1

Diese Felder werden im Member Onboarding nicht gepflegt.

| Target field | Verhalten in V1 |
|---|---|
| `accountInfo.*` | leer/default |
| `businessRole` | leer/default oder Core-Default |
| `role` | leer/default oder Core-Default |
| `taxNumber` | leer |
| `vatNumber` | leer |
| `tariffId` | leer |
| `meters[].tariff_id` | `null` |
| `meters[].gridOperatorName` | leer |
| `meters[].gridOperatorId` | leer |

Diese Felder werden nach dem Import direkt in eegFaktura ergänzt oder durch Core-Defaultlogik gesetzt.

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
    "street": "Flurweg",
    "type": "RESIDENCE",
    "city": "Naarn",
    "streetNumber": "2",
    "zip": "4331"
  },
  "billingAddress": {
    "street": "Flurweg",
    "type": "BILLING",
    "city": "Naarn",
    "streetNumber": "2",
    "zip": "4331"
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
      "street": "Flurweg",
      "streetNumber": "2",
      "zip": "4331",
      "city": "Naarn",
      "participantState": {}
    }
  ]
}
```

Dieses Zielpayload ist bewusst nah an der sichtbaren participant-Struktur gehalten.

---

## 7. Open points to confirm with eegFaktura Core

Diese Punkte sollten vor der finalen Implementierung geklärt werden:

1. Wird `participantNumber` vom Core generiert?
2. Dürfen `businessRole` und `role` leer bleiben?
3. Dürfen `accountInfo`-Felder leer bleiben?
4. Sind `meters[].gridOperatorName` und `meters[].gridOperatorId` Pflicht?
5. Muss `participantState` explizit geliefert werden oder setzt der Core das selbst?
6. Ist `partFact = 100` der richtige technische Default?
