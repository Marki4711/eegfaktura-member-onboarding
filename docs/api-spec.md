# API Specification
## eegfaktura Member Onboarding

## 1. Scope

Diese API spezifiziert die Schnittstellen für:

- Public Registration API
- Admin API
- internen Import-Flow Richtung eegFaktura Core

Nicht Teil dieser API:
- direkte Core-APIs
- Keycloak-Konfiguration
- Tarif-/Rollenpflege
- Dokumenten-Uploads

## 2. General Rules

- Format: JSON
- API style: REST
- UTF-8
- Timestamps: ISO-8601 / RFC3339
- DB schema: `member_onboarding`
- Tabellen:
  - `member_onboarding.registration_entrypoint`
  - `member_onboarding.application`
  - `member_onboarding.metering_point`
  - `member_onboarding.status_log`

## 3. Authentication

### Public API
Kein Login erforderlich.

### Admin API
Authentifizierung über bestehenden eegFaktura-/Keycloak-Mechanismus.
Die Fachlogik prüft zusätzlich die EEG-Berechtigung im Backend.

---

## 4. Domain Types

### Status
Erlaubte Werte:
- `draft`
- `submitted`
- `under_review`
- `needs_info`
- `approved`
- `rejected`
- `imported`
- `import_failed`

### Meter Direction
Erlaubte Werte:
- `CONSUMPTION`
- `PRODUCTION`

---

## 5. Public API

## 5.1 Load registration entry point

### GET `/api/public/registration/{rc_number}`

Lädt die Grundkonfiguration für einen festen Registrierungslink anhand der RC-Nummer der EEG.

Die RC-Nummer wird gegen `member_onboarding.registration_entrypoint` geprüft.
Es erfolgt kein direkter Zugriff auf eegFaktura-Core-Tabellen.

### Path params
- `rc_number: string` — RC-Nummer der EEG

### Response 200
```json
{
  "rcNumber": "RC123456",
  "eegId": "9f3d5f0d-....",
  "title": "Mitglied werden",
  "active": true
}
```

### Errors
- `404` wenn `rc_number` in `registration_entrypoint` nicht gefunden
- `410` wenn `registration_entrypoint.is_active = false`

---

## 5.2 Create application

### POST `/api/public/applications`

Legt einen neuen Antrag an.

### Request
```json
{
  "rcNumber": "RC123456",
  "firstname": "Josef",
  "lastname": "Brandstätter",
  "birthDate": "1962-06-06",
  "email": "max@example.org",
  "phone": "0664/1234567",
  "residentStreet": "Musterstraße",
  "residentStreetNumber": "2",
  "residentZip": "1010",
  "residentCity": "Musterstadt",
  "residentCountry": "AT",
  "privacyAccepted": true,
  "privacyVersion": "2026-01",
  "accuracyConfirmed": true,
  "iban": "AT123456789012345678",
  "accountHolder": "Josef Brandstätter",
  "sepaMandateAccepted": true,
  "meteringPoints": [
    {
      "meteringPoint": "AT0031000000000000000000990022105",
      "direction": "CONSUMPTION"
    }
  ]
}
```

### Rules
- `rcNumber` Pflicht
- `firstname` Pflicht
- `lastname` Pflicht
- `email` Pflicht
- `residentStreet` Pflicht
- `residentStreetNumber` Pflicht
- `residentZip` Pflicht
- `residentCity` Pflicht
- `residentCountry` Pflicht
- mindestens ein `meteringPoint`
- `meteringPoint` innerhalb des Requests eindeutig
- `direction` muss `CONSUMPTION` oder `PRODUCTION` sein
- `privacyAccepted` muss `true` sein
- `accuracyConfirmed` muss `true` sein
- `privacyVersion` Pflicht, wenn `privacyAccepted = true`
- `iban` Pflicht (15–34 Zeichen, Leerzeichen werden normalisiert)
- `accountHolder` Pflicht
- `sepaMandateAccepted` muss `true` sein

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
- `400` Validierungsfehler
- `404` unbekannte `rcNumber`
- `410` Registrierung deaktiviert (`is_active = false`)
- `409` doppelte Zählpunktnummer im selben Request

---

## 5.3 Update application

### PUT `/api/public/applications/{id}`

Aktualisiert einen bestehenden Antrag im Status `draft` oder `needs_info`.

### Path params
- `id: uuid`

### Request
Gleiches Modell wie Create.

### Rules
- nur erlaubt bei `draft` oder `needs_info`
- vorhandene Zählpunkte werden vollständig durch den Request ersetzt

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
- `400` Validierungsfehler
- `404` Antrag nicht gefunden
- `409` Status erlaubt keine Bearbeitung

---

## 5.4 Submit application

### POST `/api/public/applications/{id}/submit`

Sendet den Antrag final ab.

### Path params
- `id: uuid`

### Request
leer

### Rules
Vor Submit müssen gesetzt sein:
- `firstname`
- `lastname`
- `email`
- `residentStreet`
- `residentStreetNumber`
- `residentZip`
- `residentCity`
- `residentCountry`
- mindestens ein Zählpunkt
- `privacyAccepted = true`
- `privacyVersion` gesetzt
- `privacyAcceptedAt` wird serverseitig gesetzt
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
- `application.submitted_at` setzen
- Eintrag in `status_log`

### Errors
- `400` Pflichtdaten fehlen
- `404` Antrag nicht gefunden
- `409` Antrag bereits submitted oder in nicht erlaubtem Status

---

## 6. Admin API

## 6.1 List applications

### GET `/api/admin/applications`

Liefert die Admin-Liste.

### Query params
- `status`
- `eeg_id`
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
      "eegId": "9f3d5f0d-....",
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
- nur Anträge der EEGs, für die der Benutzer berechtigt ist

---

## 6.2 Get application detail

### GET `/api/admin/applications/{id}`

### Response 200
```json
{
  "id": "3f8c8c2d-....",
  "referenceNumber": "MO-2026-000001",
  "eegId": "9f3d5f0d-....",
  "rcNumber": "RC123456",
  "status": "submitted",
  "firstname": "Josef",
  "lastname": "Brandstätter",
  "birthDate": "1962-06-06",
  "email": "max@example.org",
  "phone": "0664/1234567",
  "residentStreet": "Musterstraße",
  "residentStreetNumber": "2",
  "residentZip": "1010",
  "residentCity": "Musterstadt",
  "residentCountry": "AT",
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
- `404` nicht gefunden
- `403` keine Berechtigung für EEG

---

## 6.3 Update application as admin

### PUT `/api/admin/applications/{id}`

### Request
```json
{
  "firstname": "Josef",
  "lastname": "Brandstätter",
  "birthDate": "1962-06-06",
  "email": "max@example.org",
  "phone": "0664/1234567",
  "residentStreet": "Musterstraße",
  "residentStreetNumber": "2",
  "residentZip": "1010",
  "residentCity": "Musterstadt",
  "residentCountry": "AT",
  "adminNote": "Telefonnummer geprüft",
  "meteringPoints": [
    {
      "meteringPoint": "AT0031000000000000000000990022105",
      "direction": "CONSUMPTION"
    }
  ]
}
```

### Rules
- bearbeitbar in `submitted`, `under_review`, `needs_info`, `approved`, `import_failed`
- Zählpunkte werden vollständig ersetzt

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
  "reason": "Antrag vollständig geprüft"
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
- bei `approved`: `approved_at` setzen, `reviewed_by_user_id` setzen
- bei `rejected`: `rejected_at` setzen, `reviewed_by_user_id` setzen
- bei `needs_info`: `needs_info_reason` setzen
- immer Eintrag in `status_log`

### Response 200
```json
{
  "id": "3f8c8c2d-....",
  "status": "approved"
}
```

### Errors
- `400` ungültiger Zielstatus
- `403` keine Berechtigung
- `409` unzulässiger Statusübergang

---

## 6.5 Import application

### POST `/api/admin/applications/{id}/import`

### Rules
- nur Status `approved`
- nur berechtigte Admins
- Import läuft synchron für V1

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
- `import_started_at` setzen
- `import_finished_at` setzen
- `imported_at` setzen
- `target_participant_id` setzen
- `status = imported`
- `status_log` schreiben

### Side effects on failure
- `import_started_at` setzen
- `import_finished_at` setzen
- `import_error_message` setzen
- `status = import_failed`
- `status_log` schreiben

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
