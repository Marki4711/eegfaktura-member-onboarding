# PROJ-2: Admin Review

## Status: Planned
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
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
