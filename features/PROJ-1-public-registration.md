# PROJ-1: Public Registration

## Status: Deployed
**Created:** 2026-04-18
**Last Updated:** 2026-04-25
**Deployed:** 2026-04-25

## Overview
Enable potential EEG members to register themselves through a public web interface. Members can access a registration link, fill out their personal and metering point information, and submit their application for admin review.

## User Story
As a potential new EEG member, I want to register myself through a web form so that I can submit my membership application without manual admin data entry.

## Scope
This feature covers the complete public registration flow for V1:

- Load registration entry point via fixed link per EEG
- Create new application with member master data and metering points
- Update application data before submission
- Submit application for admin review

The feature includes:
- Client-side form validation
- Server-side data validation and persistence
- Status tracking and logging
- Email confirmation (async)

## Non-Goals
- Admin review interface
- Keycloak authentication integration
- Import to eegFaktura core system
- Tariff or role management
- Document upload/handling
- Multi-language support
- Advanced form features (save drafts, file uploads)

## Acceptance Criteria

### Load Registration Entry Point
- [ ] When I access `/register/{rc_number}`, the system loads the registration configuration
- [ ] The RC number is resolved via `member_onboarding.registration_entrypoint` — no direct reads from eegFaktura core tables
- [ ] The page displays the EEG-specific title and registration form
- [ ] If the RC number is not found in `registration_entrypoint`, I see a 404 error page
- [ ] If `registration_entrypoint.is_active = false`, I see a 410 error page

### Create Application
- [ ] I can fill out the registration form with required personal information
- [ ] I can add one or more metering points with meter numbers and directions
- [ ] The form validates data client-side before submission
- [ ] The request carries `rcNumber`; the backend resolves `eeg_id` from `registration_entrypoint` and stores `rc_number` on the application
- [ ] Upon successful creation, I receive an application ID and reference number
- [ ] The application status is set to "draft"

### Update Application
- [ ] I can modify my application data while it's in "draft" status
- [ ] I can add, remove, or modify metering points
- [ ] All changes are validated before saving
- [ ] The application remains in "draft" status until submitted

### Submit Application
- [ ] I can submit my completed application
- [ ] The system validates all required data server-side
- [ ] Upon successful submission, status changes to "submitted"
- [ ] I receive a confirmation message with reference number
- [ ] An email confirmation is sent asynchronously
- [ ] The submission is logged in the status history

### Form Validation
- [ ] Required fields: firstname, lastname, email, resident address, at least one metering point
- [ ] Email format validation
- [ ] Phone number format validation (optional)
- [ ] Postal code validation for the country
- [ ] Metering point uniqueness within the application
- [ ] Privacy consent and accuracy confirmation required
- [ ] Maximum 10 metering points per application

### Data Persistence
- [ ] All application data is stored in the database
- [ ] Metering points are stored separately linked to the application
- [ ] Status changes are logged with timestamps
- [ ] Data integrity is maintained with proper constraints

## Edge Cases

### Network Issues
- If network fails during form submission, user data is preserved in the form
- Clear error messages guide users to retry
- No duplicate applications created on retry

### Invalid Data
- Server-side validation catches client-side bypass attempts
- Detailed error messages specify which fields need correction
- Form state preserved when validation fails

### Concurrent Updates
- If user has multiple tabs open, last save wins
- No data corruption from simultaneous operations

### Large Applications
- Applications with maximum metering points handled efficiently
- Form performance remains acceptable

### Email Delivery Issues
- Application submission succeeds even if email delivery fails
- Email retry logic handled asynchronously
- No user-facing errors for email failures

### Invalid Registration Links
- Clear error pages for unknown or inactive registration slugs
- No information leakage about valid slugs

## Dependencies
- None (this is the first feature)

## Tech Design (Solution Architect)

### Implementation Scope
Backend-only implementation for the first iteration. The public registration API will be built as a Go service with PostgreSQL persistence. Frontend development will follow in a subsequent iteration using the same stack as eegfaktura-web (Next.js).

### Component Responsibilities

**HTTP Handlers** (`internal/http/`):
- Parse incoming JSON requests
- Validate request structure and required fields
- Route requests to appropriate service methods
- Format JSON responses
- Handle HTTP status codes and error responses

**Application Services** (`internal/application/`):
- Contain business logic for registration operations
- Coordinate between repositories and external services
- Handle status transitions and validation rules
- Manage transaction boundaries
- Trigger asynchronous operations (email sending)

**Repositories** (`internal/application/`):
- Encapsulate database access patterns
- Execute SQL queries for CRUD operations
- Handle database constraints and errors
- Provide data mapping between database rows and Go structs

**Domain Models** (`internal/shared/`):
- Define Go structs for request/response payloads
- Represent database entities
- Include JSON tags for API serialization
- Define validation tags for input validation

### Handler/Service/Repository Structure

```
internal/
├── http/
│   ├── registration.go     # GET /api/public/registration/{rc_number}
│   └── application.go      # POST/PUT applications, POST submit
├── application/
│   ├── registration_service.go       # Resolves rc_number via registration_entrypoint_repo
│   ├── registration_entrypoint_repo.go  # DB ops for registration_entrypoint
│   ├── application_service.go        # Business logic for applications
│   ├── application_repo.go           # Database operations for applications
│   ├── metering_point_repo.go        # Database operations for metering points
│   └── status_log_repo.go            # Database operations for status logging
└── shared/
    ├── models.go          # Request/response/domain structs
    └── errors.go          # Custom error types
```

### Database Interactions

**member_onboarding.registration_entrypoint**:
- `SELECT` by `rc_number` to validate the entry point and resolve `eeg_id`
- Check `is_active` flag; return 410 if inactive, 404 if not found
- No writes from the public API — this table is managed by admins / deployment

**member_onboarding.application**:
- `INSERT` for application creation with generated UUID and reference number; stores `rc_number` and resolved `eeg_id`
- `SELECT` for loading applications by ID with status checks
- `UPDATE` for modifying application data (draft status only)
- `UPDATE` for status transitions (draft → submitted)
- Indexed on `id`, `reference_number`, `rc_number`, `status`

**member_onboarding.metering_point**:
- `INSERT` for adding metering points (bulk insert on create/update)
- `DELETE` followed by `INSERT` for replacing all metering points on update
- Foreign key constraint to `application.id`
- Unique constraint on `(application_id, metering_point)`
- Indexed on `application_id`

**member_onboarding.status_log**:
- `INSERT` for logging status changes with timestamps
- Foreign key constraint to `application.id`
- Tracks all status transitions with `old_status`, `new_status`, `changed_at`
- Used for audit trail and status history

### Validation Approach

**Request Validation**:
- Struct tags with `validate` package for field-level validation
- Custom validators for business rules (email format, phone format, postal codes)
- Metering point uniqueness within application
- Maximum 10 metering points per application
- Required field validation with custom error messages

**Business Rule Validation**:
- Status checks before allowing updates (only draft/needs_info)
- RC number validated against `member_onboarding.registration_entrypoint` — never against core tables
- Privacy consent and accuracy confirmation required
- Metering point format validation (existing eegFaktura standards)

**Database Constraint Validation**:
- Foreign key constraints for data integrity
- Unique constraints for metering points
- Check constraints for status values and directions

### Status Transition Handling

**Allowed Transitions**:
- `null` → `draft` (creation)
- `draft` → `draft` (updates allowed)
- `draft` → `submitted` (final submission)
- `needs_info` → `draft` (admin requests changes)

**Transition Logic**:
- Service methods check current status before allowing operations
- Status changes logged in `status_log` table with timestamps
- `submitted_at` timestamp set on submission
- Email notifications triggered on status changes

**Concurrency Protection**:
- Optimistic locking with `updated_at` timestamps
- Last-write-wins strategy for concurrent updates
- Status transition validation prevents invalid state changes

### Error Handling Approach

**HTTP Error Responses**:
- `400 Bad Request` for validation errors with detailed field messages
- `404 Not Found` for invalid application IDs or registration slugs
- `409 Conflict` for status conflicts or duplicate metering points
- `500 Internal Server Error` for unexpected database/system errors

**Custom Error Types**:
- `ValidationError` with per-field messages (`map[string]string`)
- Sentinel errors: `ErrNotFound`, `ErrGone`, `ErrConflict`, `ErrInternal`

**Error Response Format** (matches `docs/api-spec.md §7`):

Validation error:
```json
{
  "code": "validation_error",
  "message": "Validation failed",
  "fields": {
    "email": "Invalid email format"
  }
}
```

Not found:
```json
{
  "code": "not_found",
  "message": "Resource not found"
}
```

Gone (inactive registration):
```json
{
  "code": "gone",
  "message": "Registration is no longer active"
}
```

All error codes are lowercase. The response is a flat JSON object — there is no wrapping `"error"` key.

### Request/Response Model Boundaries

**Request Models** (input validation):
- `CreateApplicationRequest` - for POST /applications
- `UpdateApplicationRequest` - for PUT /applications/{id}
- `SubmitApplicationRequest` - for POST /applications/{id}/submit (empty body)

**Response Models** (output formatting):
- `RegistrationConfig` - for GET /registration/{slug}
- `ApplicationResponse` - for create/update operations
- `SubmitResponse` - for submit operations

**Domain Models** (internal business logic):
- `Application` - core business entity
- `MeteringPoint` - metering point entity
- `StatusLogEntry` - status change tracking

**Database Models** (SQL mapping):
- Direct mapping to table schemas
- JSON serialization for complex fields
- Timestamp handling with proper time zones

### Migration Dependency
Requires database schema `member_onboarding` and tables:
- `application` with all required columns and indexes
- `metering_point` with foreign key constraints
- `status_log` with foreign key constraints

Migration must be applied before deploying this feature.

### Local Development Considerations for PostgreSQL

**Database Setup**:
- Local PostgreSQL instance required
- Schema `member_onboarding` must be created
- Database connection via environment variables
- Migration scripts for initial schema setup

**Development Workflow**:
- Use `docker-compose` for local PostgreSQL instance
- Database migrations run on service startup
- Test database with sample data for development
- Connection pooling configured for development load

**Testing**:
- Unit tests for services and repositories
- Integration tests with test database
- API tests with HTTP client
- Database transaction rollback for test isolation

## Implementation Notes

### Backend Implementation Complete
The Go backend has been fully implemented with the following components:

**Database Schema**:
- Created `member_onboarding` schema with `application`, `metering_point`, and `status_log` tables
- Implemented database migration scripts for schema setup
- Added proper indexes and constraints for performance and data integrity

**Go Backend Structure**:
- `internal/shared/models.go` - Domain models, request/response structs, and error types
- `internal/config/config.go` - Environment-based configuration loading
- `internal/application/` - Repository and service layers with business logic
- `internal/http/` - HTTP handlers for all API endpoints
- `cmd/server/main.go` - Main server entry point with routing setup

**API Endpoints Implemented**:
- `GET /api/public/registration/{rc_number}` - Returns registration configuration (resolved via `registration_entrypoint`)
- `POST /api/public/applications` - Creates new application with metering points
- `PUT /api/public/applications/{id}` - Updates existing application (draft status only)
- `POST /api/public/applications/{id}/submit` - Submits application for review

**Key Features**:
- Complete request validation with detailed error messages
- Status transition logic with audit logging
- Database transaction management for data consistency
- Proper error handling and HTTP status codes
- Health check endpoint for monitoring

**Development Setup**:
- `go.mod` with all required dependencies
- `Makefile` for common development tasks
- `docker-compose.yml` for local PostgreSQL setup
- `.env.example` for environment configuration

### Frontend Implementation Complete

**Pages**:
- `/` — RC-number entry page (client component, redirects to `/register/[rc_number]`)
- `/register/[rc_number]` — Server component that resolves the registration config and renders the form; returns 404 for unknown RC numbers and an informational 410 state for inactive ones

**Components**:
- `src/components/registration-form.tsx` — Full registration form (Client Component); create + submit in one user action using `createApplication` then `submitApplication`; surfaces backend `fields` errors per-field via react-hook-form `setError`; shows success state with reference number
- `src/components/metering-point-fields.tsx` — Dynamic metering point group using `useFieldArray`; supports add/remove (1–10 points); CONSUMPTION/PRODUCTION select

**API client**:
- `src/lib/api.ts` — Typed fetch wrapper for `getRegistrationConfig`, `createApplication`, `submitApplication`; parses `ApiResponseError` with `{ code, message, fields }` matching the spec error format

**Tech choices**:
- react-hook-form + zod v4 for form state and client-side validation
- shadcn/ui: Form, Input, Checkbox, Select, Card, Alert, Badge, Button
- German-language UI to match target audience
- `NEXT_PUBLIC_API_URL` env var (default `http://localhost:8080`)
- Server component initial fetch avoids CORS on page load; client POST requests use the CORS middleware on the backend

**QA Fixes Applied (2026-04-19)**:
- `BirthDate` changed from `*time.Time` to `*string` in request structs; parsed to `time.Time` via `parseDateString` helper — fixes "Invalid JSON" on create
- `UpdateApplication` now wraps metering point replacement and application field update in a single shared transaction (`CreateBulkTx` + `UpdateTx`) — fixes partial update risk
- `CreateApplication` writes a `null → draft` status log entry inside the create transaction — completes the audit trail
- `go.sum` committed; `SERVER_PORT` corrected to `PORT` in `.env.example`; README updated to reflect that the frontend is implemented
- Server Component crash on `/register/[rc_number]` fixed by moving to `try/catch` + duck-type guard for `ApiResponseError`

**Deferred to future features**:
- Email notification on submission
- Admin interface for application review (PROJ-2)
- Per-EEG customizable registration title
- Rate limiting on public endpoints

## Affected API Endpoints
- `GET /api/public/registration/{rc_number}` - load registration config (resolved via `registration_entrypoint`)
- `POST /api/public/applications` - create new application
- `PUT /api/public/applications/{id}` - update existing application
- `POST /api/public/applications/{id}/submit` - submit application

## Definition of Done
- [x] Registration entry point loads correctly
- [x] Application creation works with all required fields
- [x] Application update functionality works
- [x] Application submission validates and changes status
- [x] All form validations work client and server-side
- [x] Data persistence works correctly
- [x] Status logging works for all operations
- [ ] Email confirmation is sent — deferred, async (future feature)
- [x] Error handling works for all edge cases
- [x] API endpoints return correct responses
- [x] Database constraints and indexes are in place