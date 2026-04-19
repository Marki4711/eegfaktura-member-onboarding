# PROJ-3: Admin Frontend UI

## Status: In Progress
**Created:** 2026-04-19
**Last Updated:** 2026-04-19

## Overview

Build the admin web interface that EEG administrators use to review onboarding applications, inspect submitted member data, edit application records, and move applications through the review workflow.

This feature covers the frontend UI layer only. All admin business logic and API endpoints are implemented in PROJ-2 (Admin Review backend). No new backend code is introduced by this feature. Keycloak authentication is a separate feature (PROJ-5). Core Import trigger functionality is a separate feature (PROJ-4).

## Dependencies

- **Requires PROJ-2** (Admin Review) — all admin API endpoints (`GET /api/admin/applications`, `GET /api/admin/applications/{id}`, `PUT /api/admin/applications/{id}`, `POST /api/admin/applications/{id}/status`) must be implemented and running.
- **Blocks PROJ-5** (Keycloak Auth) — the admin frontend pages created here are unprotected until Keycloak is integrated. Until PROJ-5 is complete, admin pages must only be accessible from a protected network.

> **Security note:** Admin pages implemented in PROJ-3 carry no authentication. They must not be exposed publicly until PROJ-5 is complete.

---

## User Stories

1. As an EEG admin, I want to see a paginated, filterable list of all applications so that I can prioritize and work through the review queue efficiently.
2. As an EEG admin, I want to filter applications by status, name, email, or metering point number so that I can find specific applications quickly.
3. As an EEG admin, I want to view the full details of an application — including all member data, metering points, and status history — so that I can assess whether it is complete and correct.
4. As an EEG admin, I want to edit an applicant's member data and metering points so that I can correct errors or fill in missing information found during review.
5. As an EEG admin, I want to add or update an internal admin note on an application so that I can document context for my colleagues.
6. As an EEG admin, I want to change the status of an application using clearly labelled action buttons so that I can progress applications through the review workflow without leaving the UI.
7. As an EEG admin, I want status changes that require a reason (reject, request info) to prompt me for one so that my decision is always documented.
8. As an EEG admin, I want to see the full status history of an application so that I understand what decisions were made and when.
9. As an EEG admin, I want to see all metering points associated with an application so that I can verify the submitted connection data.

---

## Scope

### In Scope

**Application List Page** (`/admin/applications`)

- Paginated table of applications showing: reference number, firstname, lastname, email, status badge, submitted date, metering point count
- Filter panel: status (dropdown), lastname (text), email (text), metering point (text), submitted_from / submitted_to (date inputs)
- Page size selector and page navigation controls
- Empty state message for zero results
- Clicking a row navigates to the detail page
- Active filters reflected in URL query parameters (shareable, survives browser back)

**Application Detail Page** (`/admin/applications/[id]`)

- Full display of all application fields: name, birth date, email, phone, resident address, consent timestamps
- Metering points shown in a table: metering point number, direction
- Status log shown in reverse chronological order: from status, to status, reason, timestamp
- Current status displayed prominently as a coloured badge
- Admin note field shown; inline editable
- Edit button opening an edit form for member data and metering points
- Status action buttons displayed based on the current status and allowed transitions (see Status Actions below)
- Back link to the list page

**Status Actions**

The following actions are shown based on the application's current status. Only transitions supported by the admin status endpoint (PROJ-2) are included. Import-related transitions are out of scope.

| Current status | Available actions |
|---|---|
| `submitted` | Move to Under Review |
| `under_review` | Approve / Reject (reason required) / Request Info (reason required) |
| `needs_info` | Resubmit (moves back to submitted) |
| `approved` | No action — import trigger is PROJ-4 |
| `rejected` | No action — terminal in V1 |
| `imported` | Display only |
| `import_failed` | Display only — reset to approved is PROJ-4 |

**Edit Form**

- All member data fields editable: firstname, lastname, birth date, email, phone, all address fields
- Metering points: list of rows; each row has editable metering point number and direction; rows can be added and removed
- Admin note: text area, editable separately or within the same form submission
- Client-side validation for required fields and email format
- On save: calls `PUT /api/admin/applications/{id}`; displays success confirmation or error message
- Cancel discards all changes and returns to the read-only detail view

### Out of Scope

- Keycloak login, session management, or token handling (PROJ-5)
- Core Import trigger button or import status actions (PROJ-4)
- EEG-scoped access control or per-EEG filtering based on logged-in user (PROJ-5)
- Public registration pages (PROJ-1)
- Document or file handling
- Tariff, role, or account data management
- Email or push notifications to applicants
- Admin user management

---

## Acceptance Criteria

### List Page

- [ ] Page loads at `/admin/applications` and displays a table of applications fetched from `GET /api/admin/applications`
- [ ] Table columns: reference number, firstname, lastname, email, status, submitted date, metering point numbers (or count)
- [ ] Status is displayed as a visual badge with a distinct colour per status value
- [ ] Filter panel includes inputs for: status (dropdown), lastname (text), email (text), metering point (text), submitted_from (date), submitted_to (date)
- [ ] Applying filters sends the correct query parameters to the API and updates the table
- [ ] Pagination controls show current page and total results; page navigation works correctly
- [ ] Page size is configurable; default is 20
- [ ] Empty state message shown when the result set is empty
- [ ] Clicking a table row navigates to `/admin/applications/[id]`
- [ ] Active filters and current page are reflected in the URL so the page is shareable and survives browser back navigation

### Detail Page — Data Display

- [ ] Page loads at `/admin/applications/[id]` and displays all fields from `GET /api/admin/applications/{id}`
- [ ] All member data fields are shown: name, birth date, email, phone, resident street, street number, zip, city, country
- [ ] Consent section shows: privacy accepted, privacy version, privacy accepted at, accuracy confirmed, communication consent
- [ ] Reference number, EEG ID, RC number, and started / submitted / created timestamps are shown
- [ ] Current status is shown prominently as a coloured badge
- [ ] Admin note is shown; if null, a placeholder "No admin note" is displayed
- [ ] A "Back to list" link navigates back to `/admin/applications`
- [ ] 404 response from the API results in a "Application not found" message with a back link, not a crash

### Detail Page — Metering Points

- [ ] Metering points are shown in a table with columns: metering point number, direction
- [ ] If no metering points exist, an empty state message is shown

### Detail Page — Status Log

- [ ] Status log entries are shown in reverse chronological order (newest first)
- [ ] Each entry shows: from status, to status, reason (if present), and timestamp
- [ ] If no entries exist, an empty state message is shown

### Edit Form

- [ ] An "Edit" button opens the edit form; all editable member data fields are pre-filled with current values
- [ ] Metering points are shown as a list of rows; each row has editable fields for metering point number and direction
- [ ] Rows can be added (empty new row) and removed; at least one row must remain (client-side validation)
- [ ] Client-side validation prevents submission with an empty required field (firstname, lastname, email, street, zip, city, country, at least one metering point)
- [ ] Client-side validation shows an error for invalid email format
- [ ] Submitting the form calls `PUT /api/admin/applications/{id}` with all updated fields
- [ ] On success, the form closes and the detail page refreshes with the updated values
- [ ] On API error, the form stays open and an error message is displayed; no data is lost
- [ ] Admin note is editable as a text area; saving it updates `adminNote` via the same PUT request
- [ ] Cancelling the edit form discards all changes and returns to the read-only view without an API call

### Status Actions

- [ ] For `submitted`: a "Move to Under Review" button is shown; clicking it calls `POST .../status` with `{ "toStatus": "under_review" }`
- [ ] For `under_review`: "Approve", "Reject", and "Request Info" buttons are shown
- [ ] "Approve" calls `POST .../status` with `{ "toStatus": "approved" }` directly (no reason required)
- [ ] "Reject" opens a reason input; submit is blocked until reason is non-empty; calls `{ "toStatus": "rejected", "reason": "..." }`
- [ ] "Request Info" opens a reason input; submit is blocked until reason is non-empty; calls `{ "toStatus": "needs_info", "reason": "..." }`
- [ ] For `needs_info`: a "Resubmit" button is shown; clicking it calls `{ "toStatus": "submitted" }`
- [ ] For `approved`, `rejected`, `imported`, `import_failed`: no status action buttons are shown; a static note indicates the current state
- [ ] A successful status change updates the status badge and appends the new entry to the status log without a full page reload
- [ ] An API error on status change shows an error message inline; the displayed status does not change
- [ ] A 409 conflict response displays a message explaining the action is no longer valid and prompts the user to reload

---

## Edge Cases

### Application not found
- `GET /api/admin/applications/{id}` returns 404.
- Display a "Application not found" message with a back link to the list.
- No crash or blank page.

### API error on status change
- API returns 409, 422, or 500.
- Error message displayed inline below the action buttons.
- Displayed status unchanged; user can retry or reload.

### Concurrent status change
- Another admin changed the status between the user loading the page and taking an action.
- The 409 conflict response is shown with a message explaining the action is no longer allowed.
- User is prompted to reload the detail page to see the current status before retrying.

### Reason left empty for required transitions (reject, request info)
- The submit button for these actions is disabled until the reason field is non-empty.
- No API call is made without a reason.

### Empty metering points in edit form
- If the user removes all metering point rows, client-side validation shows an error ("at least one metering point is required") and blocks submission.
- No API call is made.

### Terminal status (rejected, imported)
- No status action buttons are shown.
- A static note is displayed indicating the application is in a terminal or pending-import state.

### Large status log
- Applications with many status log entries display all entries without truncation.
- The timeline scrolls within the page without breaking the layout.

### Long admin note
- Admin note textarea expands to show the full content.
- The display area wraps the note text without overflow.

### API unavailable on page load
- List page: shows an error message instead of the table; a retry button is offered.
- Detail page: shows an error message instead of the form; a retry button is offered.

---

## Affected Frontend Routes

| Route | Purpose |
|---|---|
| `/admin/applications` | Application list with filters and pagination |
| `/admin/applications/[id]` | Application detail, edit form, and status actions |

---

## Affected API Endpoints

All endpoints are implemented in PROJ-2 (Admin Review backend). No new backend endpoints are introduced by this feature.

| Method | Path | Used by |
|---|---|---|
| `GET` | `/api/admin/applications` | List page: fetch paginated, filtered application list |
| `GET` | `/api/admin/applications/{id}` | Detail page: fetch full application including metering points and status log |
| `PUT` | `/api/admin/applications/{id}` | Edit form: save member data, metering points, and admin note |
| `POST` | `/api/admin/applications/{id}/status` | Status action buttons: submit status transitions |

---

## Tech Design (Solution Architect)

### Implementation Scope

Frontend-only. No new backend endpoints are introduced. All data comes from the four existing PROJ-2 admin API endpoints. This feature aligns with Phase 4 of `docs/build-plan.md` (frontend layer addition).

No new shadcn/ui packages need to be installed — the following components are already available in `src/components/ui/` and cover every requirement: `Badge`, `Button`, `Table`, `Dialog`, `Input`, `Select`, `Textarea`, `Skeleton`, `Pagination`, `Card`, `Form`, `Separator`, `Sonner`.

---

### New Files

```
src/app/admin/
├── layout.tsx                         # AdminLayout — single auth hook point for PROJ-5
└── applications/
    ├── page.tsx                       # List page
    └── [id]/
        └── page.tsx                   # Detail page

src/components/
├── admin-filter-panel.tsx             # Filter form (status, text, date inputs)
├── admin-application-table.tsx        # Paginated table with row click navigation
├── admin-status-badge.tsx             # Colour-coded status badge
├── admin-application-detail.tsx       # Full detail composition
├── admin-metering-point-table.tsx     # Read-only metering points table
├── admin-status-log.tsx               # Status log timeline (reverse-chronological)
├── admin-status-actions.tsx           # Conditional action buttons + reason dialog
├── admin-note-editor.tsx              # Inline admin note display + edit
└── admin-edit-form.tsx                # Edit modal with member data + metering point rows

src/lib/api.ts                         # Extended with admin types and API functions
```

Existing files modified:

```
src/lib/api.ts     # + admin types (ApplicationListItem, AdminApplicationDetail,
                   #   AdminUpdateApplicationRequest, ChangeStatusRequest, etc.)
                   # + admin API functions (listApplications, getApplicationDetail,
                   #   updateApplication, changeApplicationStatus)
```

---

### Page Structure and Routes

```
src/app/admin/layout.tsx
  Wraps all /admin/* pages in a shared shell.
  In PROJ-3: renders children directly — no auth, no redirects.
  In PROJ-5: adds Keycloak session check and token injection here only.
  Contains a minimal navigation bar (link to list, product name).

src/app/admin/applications/page.tsx
  List page. Reads filter + page state from URL search params.
  Fetches GET /api/admin/applications with current params on mount and on param change.
  Delegates rendering to AdminFilterPanel and AdminApplicationTable.

src/app/admin/applications/[id]/page.tsx
  Detail page. Reads application ID from route param.
  Fetches GET /api/admin/applications/{id} on mount.
  Delegates rendering to AdminApplicationDetail.
```

---

### Component Tree

**List page:**

```
AdminLayout
└── ApplicationListPage
    ├── AdminFilterPanel
    │   ├── Select (status filter)
    │   ├── Input × 3  (lastname, email, metering_point)
    │   ├── Input × 2  (submitted_from, submitted_to — date type)
    │   └── Button (Apply) + Button (Clear)
    ├── AdminApplicationTable
    │   ├── Table
    │   │   └── ApplicationRow × n  (clickable → /admin/applications/[id])
    │   ├── Skeleton rows  (while loading)
    │   └── EmptyState message  (when items = [])
    └── Pagination  (page controls + page size selector)
```

**Detail page:**

```
AdminLayout
└── AdminApplicationDetail
    ├── DetailHeader
    │   ├── BackLink  (→ /admin/applications with preserved filters)
    │   ├── ReferenceNumber + AdminStatusBadge
    │   └── Button "Edit"  (opens AdminEditForm dialog)
    ├── MemberDataCard  (read-only: name, birth date, email, phone, address)
    ├── ConsentCard  (privacy accepted, version, confirmed at, accuracy, communication)
    ├── AdminMeteringPointTable  (metering point number + direction, read-only)
    ├── AdminStatusActions  (conditional per current status — see below)
    ├── AdminNoteEditor  (display + inline edit for admin_note)
    └── AdminStatusLog  (timeline, reverse-chronological)
```

**Edit form (dialog):**

```
AdminEditForm  (shadcn Dialog)
├── Form (react-hook-form backed)
│   ├── Member data fields (pre-filled from current application)
│   ├── MeteringPointRows  (dynamic list: add/remove rows, each with text + select)
│   └── AdminNote Textarea
└── FormActions
    ├── Button "Save"  (disabled while submitting)
    └── Button "Cancel"  (closes dialog, discards changes)
```

**Status actions panel:**

```
AdminStatusActions  (receives current status as prop)
├── [submitted]     → Button "Move to Under Review"
├── [under_review]  → Button "Approve"
│                   → Button "Reject"  (opens reason dialog)
│                   → Button "Request Info"  (opens reason dialog)
├── [needs_info]    → Button "Resubmit"
├── [approved]      → static note: "Awaiting import (PROJ-4)"   ← PROJ-4 placeholder slot
├── [rejected]      → static note: "Application rejected"
├── [imported]      → static note: "Imported successfully"
├── [import_failed] → static note: "Import failed (reset via PROJ-4)"  ← PROJ-4 placeholder slot
└── ReasonDialog  (shadcn Dialog — shared for reject and needs_info)
    ├── Textarea (reason, required)
    └── Button "Confirm" (disabled until reason non-empty) + Button "Cancel"
```

---

### Filter State Handling

All filter values and the current page number are stored exclusively in the URL search params — no React state outside of transient input values. This ensures:

- Filters survive browser back navigation
- URLs are shareable
- No state synchronisation bugs between URL and component state

**Read:** `useSearchParams()` reads current values on every render.

**Write:** Filter changes call `router.push()` (or `router.replace()` for page changes) with the new search params string. Changing any filter resets `page` to 1.

**Param names** match the API query params exactly: `status`, `lastname`, `email`, `metering_point`, `submitted_from`, `submitted_to`, `page`, `page_size`.

**Implementation note for the back link on the detail page:** The detail page back link should preserve filter state. This is achieved by reading `window.history.state` or passing the full list URL as a `returnTo` search param from the list page into the detail page link.

---

### API Integration

The existing `src/lib/api.ts` pattern — a typed `request<T>` helper that throws `ApiResponseError` on non-2xx — is extended with admin types and functions. No new pattern is introduced.

**New types added to `src/lib/api.ts`:**

| Type | Used by |
|---|---|
| `ApplicationStatus` | All admin pages — status badge, action logic |
| `ApplicationListItem` | List table rows |
| `ApplicationListResponse` | List page fetch result |
| `AdminApplicationDetail` | Detail page (includes `meteringPoints[]` and `statusLog[]`) |
| `MeteringPointDetail` | Detail meteringPoints array item |
| `StatusLogEntry` | Detail statusLog array item |
| `AdminUpdateApplicationRequest` | Edit form submit |
| `ChangeStatusRequest` | Status action submit |
| `ChangeStatusResponse` | Status action result |

**New functions added to `src/lib/api.ts`:**

| Function | Calls |
|---|---|
| `listApplications(params)` | `GET /api/admin/applications` |
| `getApplicationDetail(id)` | `GET /api/admin/applications/{id}` |
| `updateApplication(id, data)` | `PUT /api/admin/applications/{id}` |
| `changeApplicationStatus(id, req)` | `POST /api/admin/applications/{id}/status` |

All four functions are plain async functions that call `request<T>()`. They throw `ApiResponseError` on failure — the calling component catches and displays the error.

---

### Loading, Error, and Empty States

Every network call follows the same three-state pattern: loading → data | error.

**List page:**

| State | What is shown |
|---|---|
| Loading | Skeleton rows (5 rows × all columns) beneath the filter panel |
| Error | Error card with message + "Retry" button (re-triggers the fetch) |
| Empty (status 200, items = []) | Empty state: "Keine Anträge gefunden. Passen Sie die Filter an." |
| Data | Populated table + pagination |

**Detail page:**

| State | What is shown |
|---|---|
| Loading | Skeleton placeholders for each data section |
| 404 | "Antrag nicht gefunden" card with back link |
| Other error | Error card with message + "Retry" button |
| Data | Full detail layout |

**Status action / edit form:**

- The submit button shows a spinner and is disabled while the request is in flight.
- On API error: inline error message below the button; the dialog stays open; no data is lost.
- On success: `Sonner` toast notification (already wired in `layout.tsx`); dialog closes and the detail page data refreshes by re-fetching `GET /api/admin/applications/{id}`.

---

### Local Development Without Keycloak

In PROJ-3, the admin pages call the backend with no `Authorization` header. The PROJ-2 Go endpoints are unprotected, so they respond with full data.

`NEXT_PUBLIC_API_URL` (already used by the public registration) is the only environment variable needed. No new env vars are required.

**Local dev setup:**
1. Start the Go backend: `make run` (or `go run ./cmd/server`)
2. Start the Next.js dev server: `npm run dev`
3. Navigate to `http://localhost:3000/admin/applications`
4. The backend must have at least one `registration_entrypoint` row and some applications (from seed data or from the public registration form)

---

### Keycloak Compatibility (PROJ-5 Forward-Compatibility)

The design isolates the auth integration to a single point so PROJ-5 requires minimal changes to PROJ-3 code:

1. **`src/app/admin/layout.tsx`** — the only place that will add:
   - Keycloak session check (redirect to login if no valid token)
   - Token header injection (via a context provider or by patching the `request()` helper)

2. **`src/lib/api.ts`** — the `request()` helper will be extended in PROJ-5 to attach `Authorization: Bearer <token>`. All admin functions call `request()`, so the change propagates automatically.

3. **No auth logic is spread across individual page or component files** — each component receives only data props; none read or check session state directly.

---

### Core Import Compatibility (PROJ-4 Forward-Compatibility)

The detail page includes two placeholder areas that are empty in PROJ-3 and filled by PROJ-4:

1. **`AdminStatusActions`** — for `approved` and `import_failed` statuses, a static note is rendered today. In PROJ-4, an "Import" button and a "Reset to Approved" button are added to the same component by extending the status switch. The component's interface does not change.

2. **Import metadata display** — the `GET /api/admin/applications/{id}` response already returns `importedAt`, `targetParticipantId`, `importStartedAt`, `importFinishedAt`, `importErrorMessage`. These fields can be displayed as read-only metadata in the detail page in PROJ-3 even without the import action, so admins can see the current import state without any PROJ-4 work.

---

### Status Badge Colour Mapping

`AdminStatusBadge` renders a shadcn `Badge` with a colour variant determined by the status value:

| Status | Colour intent |
|---|---|
| `draft` | secondary (grey) |
| `submitted` | blue |
| `under_review` | yellow / warning |
| `needs_info` | orange |
| `approved` | green |
| `rejected` | destructive (red) |
| `imported` | green (muted) |
| `import_failed` | destructive (red) |

The mapping is a single lookup table in `admin-status-badge.tsx` — easy to update without touching other components.
