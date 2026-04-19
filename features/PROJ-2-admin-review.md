# PROJ-2: Admin Review

## Status: Architected
**Created:** 2026-04-19
**Last Updated:** 2026-04-19

## Overview

Enable EEG administrators to review submitted onboarding applications, update applicant data, move applications through the review workflow, and document their decisions — all through a dedicated admin interface.

This feature covers only the admin review workflow. Import into eegFaktura core is a separate feature (PROJ-3). Keycloak authentication is a separate feature (PROJ-4).

## Dependencies

- **Requires PROJ-1** (Public Registration) — applications must exist in the database before admin review is possible.
- **Blocks PROJ-4** (Keycloak Auth) — the admin endpoints created here will be secured once Keycloak integration is added. Until then, the endpoints are unprotected and must not be exposed publicly.
- **Blocks PROJ-3** (Core Import) — the `approved` status transition produced here is the prerequisite for triggering import.

> **Security note:** Admin endpoints implemented in PROJ-2 carry no authentication. They must be placed behind a network boundary (e.g. only accessible from a private network or via a reverse proxy with IP allowlist) until PROJ-4 is complete. EEG-scoped access control (`reviewed_by_user_id`, per-EEG filtering) is also deferred to PROJ-4.

---

## User Stories

1. As an EEG admin, I want to see a paginated, filterable list of all applications so that I can prioritize and work through them efficiently.
2. As an EEG admin, I want to view the full details of an application — including all personal data, metering points, and status history — so that I can assess whether it is complete and correct.
3. As an EEG admin, I want to update an applicant's data so that I can correct errors or fill in missing information found during review.
4. As an EEG admin, I want to move an application from `submitted` to `under_review` so that I signal I have started processing it.
5. As an EEG admin, I want to approve an application so that it becomes eligible for import into eegFaktura.
6. As an EEG admin, I want to reject an application with a reason so that the applicant can be informed and the decision is documented.
7. As an EEG admin, I want to mark an application as needing more information and record what is missing so that the applicant can correct and re-submit it.
8. As an EEG admin, I want to add or update an internal note on an application so that I can document context for my colleagues.

---

## Scope

### In Scope

- List applications with filters and pagination (`GET /api/admin/applications`)
- Application detail view including metering points and status log (`GET /api/admin/applications/{id}`)
- Admin update of application master data and metering points (`PUT /api/admin/applications/{id}`)
- Status transitions: submitted → under_review, under_review → needs_info / approved / rejected (`POST /api/admin/applications/{id}/status`)
- Admin note (`admin_note`) creation and update
- `needs_info_reason` recorded on needs_info transition
- Status log entry written for every transition
- Backend API only — admin frontend UI is a future feature

### Out of Scope

- Keycloak authentication or EEG-scoped access control (PROJ-4)
- Import into eegFaktura core (PROJ-3)
- Email or push notifications to applicants
- Document or file handling
- Public registration endpoints (PROJ-1)
- Tariff, role, or account data
- Direct reads from eegFaktura core tables
- Admin frontend UI (follow-up feature after PROJ-2 backend)

---

## Acceptance Criteria

### List Applications

- [ ] `GET /api/admin/applications` returns 200 with a paginated array of application summaries
- [ ] Each summary includes: `id`, `referenceNumber`, `eegId`, `rcNumber`, `status`, `firstname`, `lastname`, `email`, `submittedAt`, and a list of metering point numbers
- [ ] Supports filtering by: `status`, `eeg_id`, `reference_number`, `lastname`, `email`, `metering_point`, `submitted_from`, `submitted_to`
- [ ] Supports pagination via `page` and `page_size` query parameters
- [ ] Results are ordered by `submitted_at` descending by default
- [ ] Returns 200 with an empty `items` array when no results match the filter
- [ ] Returns `page`, `pageSize`, and `total` alongside `items`
- [ ] `page_size` is clamped to a maximum of 100

### Application Detail

- [ ] `GET /api/admin/applications/{id}` returns 200 with the full application record
- [ ] Response includes all fields from `member_onboarding.application`
- [ ] Response includes the full list of metering points with `id`, `meteringPoint`, and `direction`
- [ ] Response includes the full status log as an array ordered by `created_at` ascending
- [ ] Each status log entry includes: `fromStatus`, `toStatus`, `changedByUserId`, `reason`, `createdAt`
- [ ] Returns 404 if the application ID does not exist

### Admin Update

- [ ] `PUT /api/admin/applications/{id}` updates the application's member data and/or metering points
- [ ] Updatable fields: `firstname`, `lastname`, `birthDate`, `email`, `phone`, `residentStreet`, `residentStreetNumber`, `residentZip`, `residentCity`, `residentCountry`, `adminNote`, and `meteringPoints`
- [ ] Metering points are fully replaced on update (same behaviour as public update)
- [ ] Update is allowed only in statuses: `submitted`, `under_review`, `needs_info`, `approved`, `import_failed`
- [ ] Returns 409 if the application is in `draft`, `rejected`, or `imported` status
- [ ] Returns 404 if the application ID does not exist
- [ ] Returns 400 for validation errors (email format, metering point duplicates, etc.)
- [ ] `updated_at` is refreshed on every successful update

### Status Transitions

- [ ] `POST /api/admin/applications/{id}/status` accepts `{ "toStatus": "...", "reason": "..." }`
- [ ] Allowed admin-initiated transitions:
  - `submitted → under_review`
  - `under_review → needs_info` (requires non-empty `reason`)
  - `under_review → approved`
  - `under_review → rejected` (requires non-empty `reason`)
  - `needs_info → submitted`
- [ ] Returns 400 if `toStatus` is not a recognised status value
- [ ] Returns 409 if the transition is not allowed from the application's current status
- [ ] Returns 409 if `under_review → needs_info` or `under_review → rejected` is requested without a `reason`
- [ ] On `approved`: sets `approved_at` timestamp; `reviewed_by_user_id` written as `null` (populated once Keycloak auth is wired in PROJ-4)
- [ ] On `rejected`: sets `rejected_at` timestamp; `reviewed_by_user_id` written as `null`
- [ ] On `needs_info`: writes `reason` to `needs_info_reason` on the application
- [ ] Every successful transition writes an entry to `member_onboarding.status_log` with `from_status`, `to_status`, `reason`, and `created_at`
- [ ] Response returns `{ "id": "...", "status": "..." }` on success

### Admin Note

- [ ] `adminNote` can be set or updated via `PUT /api/admin/applications/{id}` at any allowed status
- [ ] `adminNote` is returned in the detail view
- [ ] Setting `adminNote` to `null` or empty string clears the field

---

## Edge Cases

### Application not found
- All endpoints return 404 with `{ "code": "not_found", "message": "..." }`.

### Invalid status transition
- Returns 409 with `{ "code": "conflict", "message": "status transition is not allowed" }`.
- The current and requested status must be included in the error message to aid debugging.

### Missing reason on needs_info / rejected
- Returns 400 with `{ "code": "validation_error", "fields": { "reason": "reason is required for this transition" } }`.

### Update while in disallowed status
- `PUT /api/admin/applications/{id}` on a `draft`, `rejected`, or `imported` application returns 409.
- Draft applications are owned by the public user; admin should not modify them before submission.

### Concurrent status updates
- Last-write-wins. No optimistic locking for V1. Acceptable given low-concurrency admin workflows.

### Empty application list
- `GET /api/admin/applications` with filters that match nothing returns `{ "items": [], "page": 1, "pageSize": 20, "total": 0 }`.

### Large page_size
- `page_size` above 100 is clamped to 100. No error is returned — the clamped value is reflected in the response `pageSize` field.

### needs_info → submitted transition
- This transition is also triggered when the public user re-submits after a `needs_info` status (handled in PROJ-1 submit flow). The admin-facing status endpoint should also allow it to let an admin manually re-open an application.

### reviewed_by_user_id before Keycloak
- Written as `null` for all transitions in PROJ-2. The field is present in the response but empty. PROJ-4 will populate it.

### Metering points on admin update
- If `meteringPoints` is provided: full replacement (delete all, insert new), same as public update.
- If `meteringPoints` is omitted: existing metering points are unchanged.
- If `meteringPoints` is an empty array: returns 400 — at least one metering point is required.

---

## Affected Tables

| Table | Operations |
|---|---|
| `member_onboarding.application` | SELECT (list + detail), UPDATE (data fields, status timestamps, admin_note, needs_info_reason) |
| `member_onboarding.metering_point` | SELECT (as part of detail), DELETE + INSERT (on admin update) |
| `member_onboarding.status_log` | SELECT (as part of detail), INSERT (on every status transition) |

No new tables are required. No schema migrations are expected beyond those already applied by PROJ-1.

---

## Affected API Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/admin/applications` | List applications with filters + pagination |
| `GET` | `/api/admin/applications/{id}` | Application detail with metering points + status log |
| `PUT` | `/api/admin/applications/{id}` | Update application data and/or metering points |
| `POST` | `/api/admin/applications/{id}/status` | Trigger a status transition |

All endpoints are specified in `docs/api-spec.md` §6. Response shapes are defined there and must not diverge.

---

## Tech Design (Solution Architect)

### Implementation Scope

Backend-only. No database migrations are required — all columns needed by this feature (`admin_note`, `needs_info_reason`, `reviewed_by_user_id`, `approved_at`, `rejected_at`) already exist in the schema created by PROJ-1 migrations. New routes are added to the existing Go service.

---

### Component Responsibilities

The same three-layer separation established in PROJ-1 applies unchanged:

**HTTP Handlers** (`internal/http/admin.go` — new file)
- Parse and validate incoming requests
- Route to the appropriate service method
- Map service errors to HTTP status codes using the existing `httpStatusFor` / `writeError` helpers
- Write JSON responses

**Application Service** (`internal/application/admin_service.go` — new file)
- Contain all business logic for admin operations
- Enforce allowed status transitions and reason requirements
- Coordinate between repositories
- Manage transaction boundaries for multi-table writes
- Accept an optional `actorID string` parameter on status transitions (empty in PROJ-2, populated from Keycloak token in PROJ-4 — see Keycloak Compatibility section)

**Repositories** — existing files extended, no new files required
- `application_repo.go`: add `List` (filtered + paginated query) and `UpdateAdmin` (update fields accessible only to admins, including `admin_note`, `needs_info_reason`, and status timestamps)
- `metering_point_repo.go`: existing `GetByApplicationID` and `CreateBulkTx` are sufficient
- `status_log_repo.go`: existing `CreateTx` is sufficient; add `GetByApplicationID` if not already returning from existing method (it already exists)

---

### New Files

```
internal/
├── http/
│   └── admin.go                        # AdminHandler (4 handler methods)
└── application/
    └── admin_service.go                # AdminApplicationService
```

Two existing files receive new methods:

```
internal/application/application_repo.go    # + List, UpdateAdmin, UpdateStatusAdmin
internal/shared/requests.go                 # + AdminUpdateApplicationRequest, ChangeStatusRequest
internal/shared/models.go (or requests.go)  # + response types for admin endpoints
cmd/server/main.go                          # + admin route registration + auth middleware placeholder
```

---

### Request and Response Models

**New request models** (`internal/shared/requests.go`):

`AdminUpdateApplicationRequest` — partial update, all fields optional; same shape as the public `UpdateApplicationRequest` but adds `AdminNote *string`. Metering point handling is identical to the public update.

`ChangeStatusRequest` — carries `ToStatus string` and `Reason string`. Both fields are present in every request; `Reason` is only validated as required for specific target statuses (`needs_info`, `rejected`).

**New response models** (`internal/shared/` — a new `admin_models.go` or appended to `requests.go`):

`ApplicationListItem` — summary row used in the list response: `id`, `referenceNumber`, `eegId`, `rcNumber`, `status`, `firstname`, `lastname`, `email`, `submittedAt`, and `meteringPoints []string` (metering point number strings only, not full objects).

`ApplicationListResponse` — wraps the list: `items []ApplicationListItem`, `page int`, `pageSize int`, `total int`.

`AdminApplicationDetailResponse` — full application record (all columns from `member_onboarding.application`) plus `meteringPoints []MeteringPoint` and `statusLog []StatusLogEntry`. Reuses the existing `MeteringPoint` and `StatusLogEntry` domain structs.

`ChangeStatusResponse` — minimal: `id uuid.UUID`, `status string`.

---

### Handler / Service / Repository Structure

```
GET  /api/admin/applications
  → AdminHandler.ListApplications
      → AdminApplicationService.ListApplications(filters, page, pageSize)
          → ApplicationRepository.List(filters, page, pageSize)  [returns rows + total count]
          → MeteringPointRepository.GetMeteringPointNumbersByApplicationIDs(ids)
          ← ApplicationListResponse

GET  /api/admin/applications/{id}
  → AdminHandler.GetApplicationDetail
      → AdminApplicationService.GetApplicationDetail(id)
          → ApplicationRepository.GetByID(id)      [already exists]
          → MeteringPointRepository.GetByApplicationID(id)  [already exists]
          → StatusLogRepository.GetByApplicationID(id)      [already exists]
          ← AdminApplicationDetailResponse

PUT  /api/admin/applications/{id}
  → AdminHandler.UpdateApplication
      → AdminApplicationService.AdminUpdateApplication(id, req)
          → ApplicationRepository.GetByID(id)      [status check]
          → MeteringPointRepository.ValidateUniqueMeteringPoints(points)
          → db.Begin()
              → ApplicationRepository.UpdateAdmin(tx, app)
              → MeteringPointRepository.CreateBulkTx(tx, id, points)  [if provided]
          → tx.Commit()
          ← ApplicationResponse

POST /api/admin/applications/{id}/status
  → AdminHandler.ChangeStatus
      → AdminApplicationService.ChangeStatus(id, toStatus, reason, actorID)
          → ApplicationRepository.GetByID(id)      [current status]
          → validate transition is allowed
          → validate reason present if required
          → db.Begin()
              → ApplicationRepository.UpdateStatusAdmin(tx, id, toStatus, timestamps, reason, actorID)
              → StatusLogRepository.CreateTx(tx, entry)
          → tx.Commit()
          ← ChangeStatusResponse
```

---

### List Query and Filtering

The list query is a single SQL SELECT against `member_onboarding.application`. Filters are applied as optional WHERE conditions — only the parameters actually present in the request are included in the query. The implementation builds the WHERE clause and argument list dynamically, using numbered placeholders (`$1`, `$2`, …) safe against injection.

**Supported filters and their column mappings:**

| Query param | Column / approach |
|---|---|
| `status` | `application.status = $n` |
| `eeg_id` | `application.eeg_id = $n` |
| `reference_number` | `application.reference_number = $n` (exact) |
| `lastname` | `application.lastname ILIKE $n` (case-insensitive prefix or contains match) |
| `email` | `application.email ILIKE $n` |
| `metering_point` | `EXISTS (SELECT 1 FROM member_onboarding.metering_point mp WHERE mp.application_id = application.id AND mp.metering_point = $n)` |
| `submitted_from` | `application.submitted_at >= $n` |
| `submitted_to` | `application.submitted_at <= $n` |

Default ordering: `submitted_at DESC NULLS LAST` (draft applications with no `submitted_at` sort to the end).

**Pagination:** `LIMIT` and `OFFSET` derived from `page` and `page_size`. `page_size` is clamped to 100 in the service before reaching the repository. A second COUNT query with the same WHERE clause (but no LIMIT/OFFSET) produces the `total` field.

**Metering point summaries in the list:** After fetching a page of applications, a second query fetches all metering point numbers for the returned application IDs in one round-trip. The service joins these into each `ApplicationListItem`.

---

### Detail View Data Assembly

Three sequential repository calls, assembled by the service:

1. `ApplicationRepository.GetByID(id)` — the full application row. Returns `ErrNotFound` if missing; the handler maps this to 404.
2. `MeteringPointRepository.GetByApplicationID(id)` — all metering points for the application.
3. `StatusLogRepository.GetByApplicationID(id)` — all status log entries, ordered by `created_at ASC`.

The service assembles these three results into a single `AdminApplicationDetailResponse`. No additional database queries are needed.

---

### Admin Update Flow

1. Fetch the current application via `GetByID`. Return 404 if not found.
2. Check that the current status is one of: `submitted`, `under_review`, `needs_info`, `approved`, `import_failed`. Return 409 otherwise.
3. Apply the partial-update patch: only fields present in the request overwrite the current application values. Fields absent from the request remain unchanged — including `adminNote` (omitting it does not clear it; sending `null` or `""` clears it explicitly).
4. Validate the patched values (email format, metering point uniqueness, etc.). Return 400 on failure.
5. Open a database transaction:
   - `ApplicationRepository.UpdateAdmin(tx, app)` — writes all updatable fields including `admin_note`.
   - `MeteringPointRepository.CreateBulkTx(tx, id, points)` — only if `meteringPoints` was present in the request. Uses the same DELETE-then-INSERT approach as the public update.
6. Commit.
7. Return the standard `ApplicationResponse` (id, referenceNumber, status, createdAt, updatedAt).

A new `UpdateAdmin` repo method is required rather than reusing the public `UpdateTx` because it must also write `admin_note`, which the public API cannot set. All other field coverage is identical.

---

### Status Transition Handling

**Allowed transition table** (enforced in the service, not the handler):

| From | To | Reason required |
|---|---|---|
| `submitted` | `under_review` | No |
| `under_review` | `needs_info` | Yes |
| `under_review` | `approved` | No |
| `under_review` | `rejected` | Yes |
| `needs_info` | `submitted` | No |

Any other combination returns 409 conflict. An unrecognised `toStatus` value returns 400 validation error.

**Side effects per target status:**

| Target | Field written |
|---|---|
| `under_review` | nothing extra |
| `needs_info` | `needs_info_reason = reason` |
| `approved` | `approved_at = now()`, `reviewed_by_user_id = actorID` |
| `rejected` | `rejected_at = now()`, `reviewed_by_user_id = actorID` |
| `submitted` (from needs_info) | `submitted_at = now()` |

These field writes are encapsulated in `ApplicationRepository.UpdateStatusAdmin(tx, ...)` — a single UPDATE that sets the correct columns for the given target status.

The entire transition — application UPDATE + status_log INSERT — is wrapped in one database transaction using the existing `CreateTx` method on the status log repository.

---

### Status Log Behaviour

Every call to the `ChangeStatus` service method writes one row to `member_onboarding.status_log`, regardless of which transition occurred. The entry records:

- `application_id` — the application being transitioned
- `from_status` — the status before the transition (read from the application before the UPDATE)
- `to_status` — the requested target status
- `changed_by_user_id` — the `actorID` parameter (null in PROJ-2; Keycloak user ID in PROJ-4)
- `reason` — the `reason` field from the request body (may be empty for transitions that do not require it)
- `created_at` — server-set timestamp

The write is inside the same transaction as the application UPDATE, so a failed commit leaves no orphaned log entries.

---

### Error Handling

PROJ-2 reuses the established flat error model from PROJ-1 (`{ code, message, fields? }`) and the same `httpStatusFor` / `writeError` / `writeJSON` helpers in `internal/http/common.go`.

| Situation | HTTP | Code |
|---|---|---|
| Application not found | 404 | `not_found` |
| Disallowed status for update | 409 | `conflict` |
| Invalid status transition | 409 | `conflict` |
| Missing required reason | 400 | `validation_error` |
| Field validation failure | 400 | `validation_error` |
| Database or unexpected error | 500 | `internal_error` |

The `handleServiceError` helper in the `ApplicationHandler` is the model; the new `AdminHandler` applies the same pattern.

---

### Route Registration

A new `/api/admin` route group is added in `cmd/server/main.go`. It is structured to accept a future auth middleware as a single insertion point:

```
r.Route("/api/admin", func(r chi.Router) {
    // PROJ-4: insert Keycloak middleware here — one line change
    r.Route("/applications", func(r chi.Router) {
        r.Get("/", adminHandler.ListApplications)
        r.Route("/{id}", func(r chi.Router) {
            r.Get("/", adminHandler.GetApplicationDetail)
            r.Put("/", adminHandler.UpdateApplication)
            r.Post("/status", adminHandler.ChangeStatus)
        })
    })
})
```

The CORS middleware already registered globally covers admin routes without additional configuration.

---

### Keycloak Compatibility

PROJ-2 is designed so that adding Keycloak authentication in PROJ-4 requires no service or repository changes — only additions at the handler and router layers:

1. **Auth middleware:** A single `r.Use(keycloakMiddleware)` line inside the `/api/admin` route group activates token validation for all admin endpoints.
2. **Actor ID propagation:** The `ChangeStatus` service method already accepts an `actorID string` parameter. In PROJ-2 the handler passes an empty string. In PROJ-4 the handler extracts the user ID from the validated JWT context and passes it — no service signature change.
3. **EEG scoping:** The list endpoint already accepts an `eeg_id` filter parameter. In PROJ-2 it is optional and user-supplied. In PROJ-4 the handler reads the EEG IDs the authenticated user is permitted to access from the JWT claims and either injects them as a mandatory filter or validates the user-supplied value — no repository interface change.

---

### Migration Dependency

No new migrations. All columns used by PROJ-2 already exist:

| Column | Table | Added in |
|---|---|---|
| `admin_note` | `application` | migration 000001 |
| `needs_info_reason` | `application` | migration 000001 |
| `reviewed_by_user_id` | `application` | migration 000001 |
| `approved_at` | `application` | migration 000001 |
| `rejected_at` | `application` | migration 000001 |
| `submitted_at` | `application` | migration 000001 |
| `from_status` | `status_log` | migration 000001 |
| `changed_by_user_id` | `status_log` | migration 000001 |
| `reason` | `status_log` | migration 000001 |

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
