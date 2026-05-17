# PROJ-4: Core Import

## Status: Deployed
**Created:** 2026-04-19
**Last Updated:** 2026-05-09

## Deployment

- **Released:** 2026-05-09 with image `sha-8285fc7` (and follow-up Opus-review fixes in subsequent images)
- **First successful end-to-end import:** `MO-2026-778412` (Gemeinde St. Nikolaus, RC `TE100200`) → core participant `b83c41ba-4b61-11f1-a4a2-fe4879f36266`
- **Helm config required:** `backend.coreBaseUrl` must include the `/api` path prefix (see `docs/import-mapping.md` §7)
- **Keycloak config required:** `tenant` mapper on the `eegfaktura-member-onboarding` client must use `Claim JSON Type: JSON` (not `String`); a `Group Membership` mapper writing into `access_groups` is recommended for parity with the eegFaktura web client

## Lessons learned during V1 rollout (2026-05-08/09)

The end-to-end test surfaced four issues not covered by the original spec, all now fixed:

1. **`tenant` claim format mismatch.** Keycloak's default User-Attribute mapper serialised `tenant` as a stringified JSON array (`"[\"TE100200\"]"`); the core's `json.Unmarshal` into `[]string` failed and returned 401 with an empty body. Fix was Keycloak-side: change Claim JSON Type to `JSON`.
2. **`businessRole` was empty.** The eegFaktura frontend uses `businessRole` (`EEG_PRIVATE` / `EEG_BUSINESS`) to switch the Privat/Firma view. Imported company members displayed as Privat. Fixed by mapping member_type → businessRole in the payload adapter.
3. **`firstname` NOT NULL violation for company types.** Onboarding does not collect firstname/lastname for `company` / `municipality` / `association` member types — the core's participant table requires `firstname`. Fixed by placing the organisation name in `firstName` and leaving `lastName` empty.
4. **Meter direction enum mismatch.** Onboarding's `PRODUCTION` is the core's `GENERATION`. Fixed by an explicit translation in `mapMeterDirection`.

See `docs/import-mapping.md` §7–§9 for the validated contract.

## Overview

Enable EEG administrators to import an approved onboarding application into the eegFaktura core system, creating a productive participant record with all associated metering points. The import is triggered manually by an admin, executed synchronously by the backend, and its outcome is recorded in the onboarding database.

This feature covers the full V1 import flow: trigger, payload assembly, core service call, success and failure handling, and audit persistence. It does not cover Keycloak authentication (PROJ-5) or the admin frontend UI (PROJ-3).

## Dependencies

- **Requires PROJ-1** (Public Registration) — applications must exist in the onboarding database.
- **Requires PROJ-2** (Admin Review) — the import endpoint is only reachable after an admin has approved an application (`status = approved`). PROJ-2 provides the `approved` status transition.
- **Blocks PROJ-5** (Keycloak Auth) — the import endpoint is unprotected in PROJ-4 and must be secured in PROJ-5.
- **Depends on eegFaktura Core** — requires a reachable internal core service endpoint and agreement on the participant payload contract (see open questions).

> **Security note:** The import endpoint carries no authentication in PROJ-4. It must be placed behind the same network boundary as the other admin endpoints until PROJ-5 is complete.

---

## User Stories

1. As an EEG admin, I want to trigger the import of an approved application so that the applicant is created as a productive participant in eegFaktura without any manual data entry.
2. As an EEG admin, I want to see that the import succeeded so that I know the participant has been created and can complete their setup in eegFaktura.
3. As an EEG admin, I want to see a clear error message when the import fails so that I understand what went wrong and can take corrective action.
4. As an EEG admin, I want to retry an import that previously failed so that temporary core-side issues do not permanently block the onboarding flow.
5. As an EEG admin, I want the full import history to be recorded in the application's status log so that I have an audit trail of all import attempts.

---

## Scope

### In Scope

- `POST /api/admin/applications/{id}/import` — trigger import of an approved application
- Pre-import validation: application must be in `approved` status
- Payload assembly from onboarding data according to `docs/import-mapping.md`
- Internal HTTP call to eegFaktura core service to create the participant
- Success handling: status → `imported`, timestamps set, `target_participant_id` persisted
- Failure handling: status → `import_failed`, error message persisted, timestamps set
- Status log entry written for every import attempt (success and failure)
- Retry path: `import_failed → approved` transition available via the admin status endpoint (PROJ-2), then re-trigger import
- Backend only — no admin frontend import button in this feature

### Out of Scope

- Keycloak authentication (PROJ-5)
- Admin frontend import trigger UI (follow-up to PROJ-3 Admin Frontend UI)
- Email or push notifications for import events
- Automatic or scheduled import (V1 is always admin-triggered and synchronous)
- Import of applications not in `approved` status
- Partial import (all metering points are included or the entire import fails)
- Rollback of a successful import in the core
- Bidirectional sync between onboarding and core after import
- Tariff, role, or account data (not managed in onboarding)
- Document handling
- Direct writes to any eegFaktura core tables

---

## Acceptance Criteria

### Trigger Import

- [ ] `POST /api/admin/applications/{id}/import` triggers the import of the specified application
- [ ] Returns 404 if the application ID does not exist
- [ ] Returns 409 if the application is not in `approved` status, with message indicating current status
- [ ] Only one import attempt runs per request — the endpoint is not idempotent; calling it again on an `imported` application returns 409

### Payload Assembly

- [ ] The participant payload is assembled from the application's fields according to `docs/import-mapping.md`
- [ ] All required fields (`firstname`, `lastname`, `email`, `residentAddress.*`, `billingAddress.*`) are present in the payload
- [ ] `billingAddress` is identical to `residentAddress` for V1 (no separate billing address in onboarding)
- [ ] All metering points for the application are included in the payload as `meters[]`
- [ ] Each meter entry includes `meteringPoint`, `direction`, and the member's resident address fields
- [ ] Technical defaults are applied for fields not managed in onboarding: `status = NEW`, `meters[].status = INIT`, `meters[].processState = NEW`, `partFact = 100`, `participantSince = now()`
- [ ] Consent fields (`privacyAccepted`, `privacyVersion`, `privacyAcceptedAt`, `accuracyConfirmed`, `communicationConsent`) are included in the payload if the core service accepts them
- [ ] Fields not managed in V1 (`accountInfo.*`, `businessRole`, `role`, `tariffId`, etc.) are sent as empty strings or omitted per core service contract

### Success Handling

- [ ] On successful core response, application status transitions to `imported`
- [ ] `imported_at` is set to the time of the successful import
- [ ] `import_started_at` is set when the import attempt begins
- [ ] `import_finished_at` is set when the core response is received (success or failure)
- [ ] `target_participant_id` is set to the participant ID returned by the core service
- [ ] A status log entry is written with `from_status = approved`, `to_status = imported`, and `created_at = import time`
- [ ] Response body: `{ "success": true, "applicationId": "...", "status": "imported", "targetParticipantId": "..." }`

### Failure Handling

- [ ] On failed core response (error status, timeout, or invalid response), application status transitions to `import_failed`
- [ ] `import_error_message` stores the error detail from the core or an internal error description
- [ ] `import_started_at` and `import_finished_at` are set even on failure
- [ ] A status log entry is written with `from_status = approved`, `to_status = import_failed`
- [ ] Application data is unchanged — no rollback required as the onboarding record is not modified beyond status and timestamps
- [ ] Response body signals failure: `{ "success": false, "applicationId": "...", "status": "import_failed", "message": "..." }`
- [ ] The HTTP status code on import failure is 409, 422, or 500 depending on the error type

### Retry Path

- [ ] An admin can reset an `import_failed` application back to `approved` using `POST /api/admin/applications/{id}/status` with `{ "toStatus": "approved" }` (admin status endpoint, PROJ-2)
- [ ] After resetting to `approved`, the admin can trigger the import again via the import endpoint
- [ ] Each retry attempt creates a new status log entry pair (`approved → import_failed` or `approved → imported`)
- [ ] Previous `import_error_message` is overwritten by the new attempt's outcome
- [ ] Previous `import_started_at` and `import_finished_at` are overwritten by the new attempt

### Status Log

- [ ] Every import attempt — success or failure — writes exactly one entry to `member_onboarding.status_log`
- [ ] The entry records `from_status`, `to_status`, and `created_at`
- [ ] The status log write is inside the same database transaction as the application status update
- [ ] A failed DB write on status log does not leave a partially-imported application — the transaction rolls back and the application remains in `approved`

---

## Edge Cases

### Application not in approved status
- Returns 409: `{ "code": "conflict", "message": "only applications in approved status can be imported" }`.
- Includes the current status in the message.

### Application already imported
- `status = imported`: returns 409. Import is not repeatable once successful.
- `status = import_failed`: returns 409. Admin must reset to `approved` first via the status endpoint, then retry.

### import_failed → approved transition
- The `import_failed → approved` transition must be available via the admin status endpoint (PROJ-2).
- Note: this transition was inadvertently excluded when PROJ-2 removed import-related transitions from `adminTransitions` (M4 fix). It must be added back because it is not an import action — it is an admin reset to allow retry. This is a tracked gap to be resolved in PROJ-4.

### Core service unavailable
- The core service is unreachable or times out.
- Outcome: import fails with `import_failed`, `import_error_message` stores timeout description.
- The onboarding application remains unchanged except for the status transition and timestamps.
- The admin can retry after the core becomes available again.

### Core service returns an error response
- The core returns a 4xx or 5xx response.
- Outcome: `import_failed`, `import_error_message` stores the HTTP status code and response body (truncated if too long).
- If the core returns a participant ID despite an error status — the response is treated as failed to avoid partial state.

### Core service creates the participant but response is lost
- Network failure after the core has already created the participant.
- V1 accepts last-write-wins: the import will be retried if the admin resets to `approved`. The core service must be idempotent on re-create, or the admin must resolve duplicates in the core manually.
- This is a known limitation of synchronous V1 import without two-phase commit.

### Missing metering points
- If the application has no metering points at the time of import, the import fails with a validation error before calling the core (400 bad request, not 409).

### Large payload
- Applications with up to 10 metering points must be handled correctly. The full metering point list is always fetched before building the payload.

### Concurrent import attempts
- Two requests targeting the same application simultaneously are serialized through `ApplicationRepository.MarkImportInFlight`. Before calling the core, the import service performs a conditional UPDATE that matches `status='approved'` AND no other attempt is in-flight (`import_started_at IS NULL OR import_finished_at IS NOT NULL`). The winning request persists `import_started_at` and clears `import_finished_at`, marking the slot busy. The losing request's UPDATE matches zero rows and the service returns 409 ("another import is already in progress for this application"). The marker is cleared by `persistResult` writing `import_finished_at` together with the final status.

---

## Affected Tables

| Table | Operations |
|---|---|
| `member_onboarding.application` | SELECT (status check + payload assembly), UPDATE (status, timestamps, `target_participant_id`, `import_error_message`) |
| `member_onboarding.metering_point` | SELECT (included in payload assembly) |
| `member_onboarding.status_log` | INSERT (import result, inside transaction with application UPDATE) |

No schema migrations are expected — all columns used by this feature (`import_started_at`, `import_finished_at`, `imported_at`, `target_participant_id`, `import_error_message`) already exist from migration 000001.

---

## Affected API Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/admin/applications/{id}/import` | Trigger import of an approved application |

The endpoint is specified in `docs/api-spec.md §6.5`.

### Response shapes

**Success (200):**
```json
{
  "success": true,
  "applicationId": "3f8c8c2d-....",
  "status": "imported",
  "targetParticipantId": "4711"
}
```

**Failure (409 / 422 / 500):**
```json
{
  "success": false,
  "applicationId": "3f8c8c2d-....",
  "status": "import_failed",
  "message": "participant import failed: ..."
}
```

---

## Core API Contract (geklärt 2026-05-08 aus eegfaktura-backend-Quellcode)

Die folgenden Punkte wurden aus dem öffentlichen Source-Code von [`eegfaktura/eegfaktura-backend`](https://github.com/eegfaktura/eegfaktura-backend) abgeleitet. Frage 6 bleibt offen, blockiert die Implementierung aber nicht.

| # | Frage | Antwort | Quelle |
|---|---|---|---|
| 7 | HTTP-Methode + Pfad? | `POST /participant` (Statusantwort 201 Created) | `api/participantController.go` |
| 8 | Auth-Mechanismus? | Keycloak-JWT (`Authorization: Bearer ...`) **plus** HTTP-Header `tenant: <RC-Number>`. Der Tenant-Wert muss in der JWT-Claim `Tenants []string` enthalten sein, sonst 403 | `api/middleware/jwt.go` |
| 10 | `targetParticipantId` in Response? | Response-Body ist das vollständige `EegParticipant`-Objekt; ID liegt im Feld `id` (UUID, Top-Level) | `api/participantController.go` |
| 1 | `participantNumber` vom Core erzeugt? | Nein. Optional (`null.String`) — Onboarding kann unsere Mitgliedsnummer mitgeben oder leer lassen | `model/participant.go` |
| 2 | `businessRole`/`role` leer? | Ja. Beide `string`, leere Strings akzeptiert | `model/participant.go` |
| 3 | `accountInfo` leer? | Ja. `BankInfo.Iban` + `Owner` beide `null.String`, optional | `model/participant.go` |
| 4 | `meters[].gridOperatorName/Id` Pflicht? | Nein — diese Felder existieren auf Meter-Ebene gar nicht. Sie liegen auf der `Eeg`-Entität (`OperatorName`, `GridOperator`) und werden vom Core via `tenant`-Header aufgelöst | `model/Eeg.go`, `model/participant.go` |
| 5 | `participantState` mitschicken? | Wird vom Core überschrieben. `RegisterParticipant` setzt `status = PENDING` immer fest. Was Onboarding sendet ist egal | `database/participantDao.go::RegisterParticipant` |
| 9 | Core idempotent? | **Nein.** Jeder `POST /participant` legt eine neue UUID an (`uuid.NewUUID()`) — kein Upsert. Diese V1-Limitation für Retries bleibt bestehen | `database/participantDao.go::RegisterParticipant` |
| 6 | `partFact = 100` korrekter Default? | **Offen.** Feld existiert weder auf `EegParticipant` noch auf `MeteringPoint` im aktuellen Modell. Wird nicht im Payload mitgeschickt; falls der Core es zwingend braucht, ergibt sich ein 400, der gezielt nachgerüstet werden kann | — |

### Konsequenzen für die Implementierung

1. **`tenant`-HTTP-Header ist Pflicht.** Der `internal/coreclient/` setzt bei jedem Call den `tenant`-Header auf den RC der EEG.
2. **JWT-Forwarding der Admin-Session.** Das eingehende Admin-JWT (PROJ-5/Keycloak) wird unverändert an den Core durchgereicht, weil der Core dieselbe Realm-Konfiguration nutzt. Kein eigener Service-Account in V1.
3. **Erwarteter Status nach Core-Anlage = `PENDING`**, nicht `ACTIVE`. Der finale Activation-Schritt (`POST /participant/{id}/confirm`) bleibt im eegFaktura-Admin und ist nicht Teil dieser Feature-Spec.
4. **Erfolgs-Status 201, nicht 200.** Die Antwort wird parsen und `id` als `target_participant_id` speichern.

---

## Notes on import_failed → approved

The PROJ-2 admin status endpoint removed all import-related transitions from `adminTransitions` as part of the M4 QA fix. That fix correctly removed `approved → imported` and `approved → import_failed` (which belong exclusively to the import endpoint).

However, `import_failed → approved` is **not** an import action — it is an admin manual reset to enable retry. It must be added back to the PROJ-2 admin status endpoint `adminTransitions`. This change is small (one line in `admin_service.go`) and must be implemented as part of PROJ-4, or as a targeted fix to PROJ-2 before PROJ-4 goes live.

This is the only code change that touches PROJ-2 scope within the PROJ-4 feature.

---

## Tech Design (Solution Architect)

### Implementation Scope

Backend-only.

No database migrations are required — all columns used by this feature (`import_started_at`, `import_finished_at`, `imported_at`, `target_participant_id`, `import_error_message`) already exist in the schema from migration 000001.

Two new Go packages are introduced, as recommended by `CLAUDE.md`:

- `internal/coreclient/` — HTTP wrapper for the eegFaktura core participant creation API
- `internal/importing/` — import orchestration service and payload mapping adapter

One new handler method is added to the existing `AdminHandler` in `internal/http/admin.go`.

One small change is made to `internal/application/admin_service.go` to restore the `import_failed → approved` transition (see section below).

---

### New and Modified Files

**New files:**

```
internal/
├── coreclient/
│   └── core_client.go      # HTTP wrapper for the eegFaktura core participant API
└── importing/
    ├── import_service.go   # Import orchestration: validate → map → call → persist
    └── payload.go          # CoreParticipantPayload struct and mapping adapter
```

**Existing files modified:**

```
internal/http/admin.go                  # + ImportApplication handler method
internal/application/admin_service.go  # restore import_failed → approved transition
internal/config/config.go              # + CoreBaseURL and CoreTimeoutSeconds fields
cmd/server/main.go                     # wire CoreClient, ImportService, import route
```

---

### Component Responsibilities

**HTTP Handler** (`internal/http/admin.go` — new method on existing `AdminHandler`)

- Parse the application ID from the URL path
- Delegate to `ImportService.Import(id)`
- Map success and failure outcomes to the correct HTTP status codes and response shapes
- The handler does not contain any import logic or payload building — it only translates between HTTP and service layer

**Import Service** (`internal/importing/import_service.go`)

- Enforce the pre-import preconditions (status check, metering point presence)
- Record `import_started_at` before calling the core
- Coordinate the payload adapter and the core client
- Persist the import outcome (success or failure) inside a single database transaction
- Return a structured result that the handler maps to the HTTP response

**Payload Adapter** (`internal/importing/payload.go`)

- Convert `shared.Application` + `[]shared.MeteringPoint` into a `CoreParticipantPayload` struct
- Apply all V1 mapping rules from `docs/import-mapping.md`:
  - `billingAddress` = `residentAddress` (V1 rule — no separate billing address in onboarding)
  - each meter's address fields = member's resident address (V1 rule)
  - `residentAddress.type = "RESIDENCE"`, `billingAddress.type = "BILLING"`
  - technical defaults: `status = "NEW"`, `meters[].status = "INIT"`, `meters[].processState = "NEW"`, `partFact = 100`, `participantSince = import timestamp`
- Consent fields included conditionally (see open questions)
- Fields not managed in V1 (`accountInfo`, `businessRole`, `role`, `tariffId`, etc.) sent as empty strings or omitted per `docs/import-mapping.md §5`

**Core Client** (`internal/coreclient/core_client.go`)

- Wraps the internal HTTP call to the eegFaktura core service
- Configurable base URL and timeout via `internal/config/`
- Exposes a single method: send a participant payload and return either the `targetParticipantId` string or a structured error
- Defined behind an interface so it can be replaced by a test stub without changing any import service code
- Uses the standard `net/http` client — no third-party HTTP library

**Repositories** — no new repository methods needed beyond what PROJ-2 already added, except one new method:

- `ApplicationRepository`: add `UpdateImportResultTx(tx, id, fields)` — updates status, import timestamps, `target_participant_id`, and `import_error_message` atomically. Uses the same COALESCE pattern as `UpdateStatusAdminTx` so only the relevant columns for the current outcome are overwritten.
- `StatusLogRepository.CreateTx` — already exists; used unchanged.

---

### Request Flow

```
POST /api/admin/applications/{id}/import
  → AdminHandler.ImportApplication
      → ImportService.Import(id)
          → ApplicationRepository.GetByID(id)           [fetch current application]
          → MeteringPointRepository.GetByApplicationID(id) [fetch metering points]
          → validate: status == approved                 [→ 409 if not]
          → validate: len(meteringPoints) > 0            [→ 400 if none]
          → record import_started_at = now()
          → PayloadAdapter.Build(application, meteringPoints, importStartedAt)
          → CoreClient.CreateParticipant(payload)        [OUTSIDE the DB transaction]
          →  on success:
               record import_finished_at = now()
               db.Begin()
                 ApplicationRepository.UpdateImportResultTx(tx,
                   status=imported, imported_at, import_started_at,
                   import_finished_at, target_participant_id)
                 StatusLogRepository.CreateTx(tx, approved→imported)
               tx.Commit()
               ← {success: true, applicationId, status: "imported", targetParticipantId}
          →  on failure:
               record import_finished_at = now()
               normalize error message
               db.Begin()
                 ApplicationRepository.UpdateImportResultTx(tx,
                   status=import_failed, import_started_at,
                   import_finished_at, import_error_message)
                 StatusLogRepository.CreateTx(tx, approved→import_failed)
               tx.Commit()
               ← {success: false, applicationId, status: "import_failed", message}
```

---

### Transaction Boundaries

The eegFaktura core HTTP call happens **outside** the database transaction. This is intentional:

- Holding a DB transaction open across a network call would unnecessarily pin a connection and risk timeouts.
- The DB transaction wraps only the two writes: application UPDATE + status_log INSERT.
- Both writes succeed or both are rolled back — the application never reaches a partially-written state.

**Known V1 limitation:** If the core creates the participant successfully but the subsequent DB transaction fails, the onboarding record stays in `approved` and the core has a participant with no matching `target_participant_id` in onboarding. If the admin retries and the core is not idempotent, a duplicate participant may be created in the core. This is accepted as a V1 limitation (see edge case in spec). See open question #9.

---

### Status Transitions Related to Import

The import endpoint owns these transitions and writes them directly — it does not call the admin `ChangeStatus` service method:

| From | To | Trigger |
|---|---|---|
| `approved` | `imported` | Successful core response |
| `approved` | `import_failed` | Failed or timeout core response |

The admin status endpoint (`ChangeStatus` in `admin_service.go`) is responsible for the retry reset:

| From | To | Trigger |
|---|---|---|
| `import_failed` | `approved` | Admin manual reset via `POST /api/admin/applications/{id}/status` |

No other import-related transitions exist at the service layer.

---

### Restoring import_failed → approved

The M4 QA fix during PROJ-2 correctly removed `approved → imported` and `approved → import_failed` from `adminTransitions` in `admin_service.go` (those belong exclusively to the import endpoint).

However, `import_failed → approved` is a manual admin reset — not an import action — and was inadvertently removed in the same fix. As part of PROJ-4, one entry is added back to `adminTransitions`:

- `StatusImportFailed → [StatusApproved]`

No reason is required for this transition. No `reviewed_by_user_id` or timestamp side effects are needed — the transition simply unlocks retry. A status_log entry is written as for every admin transition.

This is the only change to `admin_service.go` in PROJ-4.

---

### Error Normalization

The import service classifies errors from the core client into three categories and stores a human-readable description in `import_error_message`:

| Situation | Stored message |
|---|---|
| Network timeout | `"core service timeout after Ns"` |
| HTTP error response | `"core returned HTTP {status}: {truncated body}"` |
| Response parse failure | `"could not parse core response: {detail}"` |

The message is truncated to 1000 characters before storage (the `import_error_message` column is TEXT, but the practical limit prevents runaway payloads).

The HTTP response code returned to the caller:
- `409` if the application is not in `approved` status (pre-import conflict)
- `400` if the application has no metering points (pre-import validation)
- `500` if the core call fails (import attempt failed)

---

### target_participant_id Storage

On a successful core response, the import service parses the participant ID from the response body and stores it in `application.target_participant_id` (type TEXT in the DB, `*string` in Go).

The exact field path in the core response body is **open question #10** and must be confirmed before implementation. The core client is responsible for extracting this value from the response and returning it to the import service as a plain string.

---

### Idempotency Expectations

The import endpoint is **not** idempotent from the onboarding side:

- Calling it on an `approved` application starts one import attempt.
- Calling it again on the same `approved` application (concurrent or sequential) returns 409 once the first attempt has updated the status.
- Once an application reaches `imported`, calling the endpoint again returns 409.
- To retry after `import_failed`, the admin must reset to `approved` first via the status endpoint, then call import again.

Whether the **core service** is idempotent is **open question #9**. If the core is not idempotent, retries after network failure may create duplicate participants — this is a known V1 limitation that the admin must resolve manually in the core.

---

### Configuration

Two new fields are added to `internal/config/config.go` and the corresponding environment variables:

| Config field | Env var | Purpose |
|---|---|---|
| `CoreBaseURL` | `CORE_BASE_URL` | Base URL of the eegFaktura core internal API |
| `CoreTimeoutSeconds` | `CORE_TIMEOUT_SECONDS` | HTTP timeout for the core call (default: 30s) |

Both fields are added to `.env.local.example`. If `CORE_BASE_URL` is not set, the import service fails fast at startup (fail-loud configuration — no silent no-ops).

---

### Local Development and Testability

**Without a live core service:** The core client is defined behind a Go interface. In tests, a simple in-memory stub implementing that interface replaces the real HTTP client. No network calls are needed to test the import service logic.

**With a live core service for integration testing:** `CORE_BASE_URL` can be pointed at a local mock HTTP server (e.g., `httptest.NewServer` in a Go integration test, or a simple HTTP stub binary). The failure path — network error, 5xx — is testable without any core involvement by pointing `CORE_BASE_URL` at an unreachable host.

**Suggested test coverage:**

| Scenario | What to verify |
|---|---|
| Application not found | 404 response |
| Application not in `approved` status | 409 with current status in message |
| No metering points | 400 response |
| Core returns success | status → imported, target_participant_id set, status_log written |
| Core returns error | status → import_failed, error_message stored, status_log written |
| Core times out | status → import_failed, timeout message stored |
| DB transaction fails after core success | application remains in approved (no partial write) |
| import_failed → approved reset | status transitions back, retry possible |
