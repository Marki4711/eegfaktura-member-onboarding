# High-Level Architecture
## eegfaktura Member Onboarding

## 1. Goal and Context

`eegfaktura Member Onboarding` is a new component in the eegFaktura ecosystem that enables self-service registration of EEG members.

The component supplements the existing process in which new members are manually created by an administrator in eegFaktura. The goal is to allow members to register themselves through a web form. This data is not written directly into the productive master data of eegFaktura, but stored in a dedicated data store of the new component. Only after review by an admin does the deliberate transfer into the normal data store of eegFaktura take place.

## 2. Architecture Principles

- standalone component with its own repository: `eegfaktura-member-onboarding`
- frontend using the same stack as `eegfaktura-web`
- backend as a Go service
- same PostgreSQL database as eegFaktura, but dedicated schema `member_onboarding`
- Keycloak for the admin area
- import into the productive data store only through an internal service call to the eegFaktura core
- no direct write access from onboarding to core tables

## 3. Main Components

### 3.1 Public Web
Public user interface for new members.

Responsibilities:
- entry via a fixed registration link per EEG (identified by RC number)
- collection of member data
- collection of multiple metering points
- client-side validation
- submission of the application to the backend

Not responsible for:
- persistence
- status logic
- import
- direct communication with the core

### 3.2 Admin Web
Internal user interface for EEG administrators.

Responsibilities:
- display and filtering of incoming applications
- detail view of an application
- editing of master data
- setting status values
- maintaining internal notes
- triggering the import

Not responsible for:
- direct database access
- direct core write logic
- own authentication

### 3.3 Member Onboarding Backend
Central business logic of the component.

Technology:
- Go
- REST API
- PostgreSQL access
- Keycloak integration in the admin context

Responsibilities:
- Public API
- Admin API
- server-side validation
- status transitions
- reading and writing in schema `member_onboarding`
- resolving the RC number via `member_onboarding.registration_entrypoint` (no direct access to eegFaktura core tables)
- persistence of multiple metering points
- status history
- import mapping
- internal core call
- logging of the import result

### 3.4 Persistence
Persistence uses the same PostgreSQL database as eegFaktura, but in a dedicated schema:

- `member_onboarding.registration_entrypoint` — local mapping of RC numbers to EEGs; entry point for public registration
- `member_onboarding.application`
- `member_onboarding.metering_point`
- `member_onboarding.status_log`
- `member_onboarding.legal_document` — EEG-specific legal documents shown in the registration form
- `member_onboarding.document_consent` — immutable consent snapshots recorded at application submission

### 3.5 eegFaktura Core
The core remains the authoritative system for productive participant data.

Responsibilities:
- final business validation during import
- productive creation of the participant
- returning a target ID or error message
- authoritative source for EEG master data (Gemeinschafts-ID, name, address, creditor-ID, contact-email) — synced into the onboarding via PROJ-32 (see 3.5a)

### 3.5a Core integration — calls from the onboarding backend

The onboarding backend speaks to the core over HTTP/JSON and (since PROJ-32) GraphQL. The call surface is:

| Purpose | Endpoint | Used by |
|---|---|---|
| Create participant | `POST /api/participant` | PROJ-4 import |
| Assign member tariff (post-create) | `PUT /api/participant/v2/{id}` | PROJ-27 import |
| List tariffs | `GET /api/eeg/tariff` | PROJ-27 import dialog |
| List participants (member-number derivation) | `GET /api/participant` | PROJ-27 import dialog + duplicate check |
| EEG master data (GraphQL scalar `Eeg`) | `POST /api/query` with `{"query":"query { eeg }"}` | PROJ-32 stammdaten sync |
| Billing config (logo reference) | `GET /cash/api/billingConfigs/tenant/{rcNumber}` | PROJ-33 logo sync |
| Logo bytes | `GET /cash/api/billingConfigs/{id}/logoImage` | PROJ-33 logo sync |

**URL model.** `CORE_BASE_URL` is the **hostname only** (e.g. `https://eegfaktura.at`). Path prefixes are hardcoded per call site in `internal/coreclient/` because the deployed reverse-proxy multiplexes several services under one host (`/api/...` → eegFaktura-backend, `/cash/api/...` → eegfaktura-billing).

**Auth.** Every call forwards the **logged-in admin's Keycloak JWT verbatim** as `Authorization: Bearer ...`, with the EEG's RC number in the `tenant` header. No service account, no `client_credentials`. The core enforces tenant scoping via the JWT's `Tenants` claim. Rationale: audit trail attributes the change to the actual human, no extra Keycloak infra needed.

**EEG master data — single source of truth.** PROJ-32 mirrors eight values (Gemeinschafts-ID, name, four address fields, creditor-ID, contact-email) from the core into `registration_entrypoint`. PROJ-33 adds the EEG logo bytes (max 256 KB, PNG/JPEG/GIF) as a ninth synced asset, embedded top-right on the approval + SEPA mandate PDFs. The admin UI renders all of these read-only with a lock icon. PDF/Mail rendering reads from `registration_entrypoint` unchanged. Sync writes are triggered manually via the "Aus eegFaktura aktualisieren" button; the admin's JWT travels Browser → backend → Core. The logo step is **best-effort**: if it fails (oversize, unsupported MIME, billing service down) the master-data sync still succeeds and the UI shows a warning under the logo preview.

**Performance.**
- `&http.Client{Timeout: ...}` uses Go's `http.DefaultTransport` — keep-alive, connection pool, HTTP/2 automatic.
- Body caps via `io.LimitReader`: 64 KiB (participant create + GraphQL eeg + billingConfig), 256 KiB (eeg/tariff), 4 MiB (participant list), **256 KB (logo bytes — hard reject above)**.
- The drift-comparison endpoint (`GET /api/admin/settings/eeg/core-comparison`) memoises FetchEEGMasterData per RC for 30 s. Sync warms the cache with the just-fetched payload.

## 4. System Boundaries

Allowed connections:
- Public Web -> Member Onboarding Backend
- Admin Web -> Member Onboarding Backend
- Member Onboarding Backend -> Schema `member_onboarding`
- Member Onboarding Backend -> eegFaktura Core

Disallowed connections:
- Public Web -> eegFaktura Core
- Admin Web -> eegFaktura Core
- Frontend -> Database
- Member Onboarding -> direct core tables

## 5. Data Storage

The module uses a deliberately reduced relational model without JSON fields.

Tables:
- `member_onboarding.registration_entrypoint`
- `member_onboarding.application`
- `member_onboarding.metering_point`
- `member_onboarding.status_log`
- `member_onboarding.legal_document`
- `member_onboarding.document_consent`

Basic rules:
- one application contains exactly one member
- one application belongs to exactly one EEG
- the public entry point is identified by the EEG's RC number
- the RC number is resolved exclusively via `member_onboarding.registration_entrypoint`; no direct access to eegFaktura core tables
- one application can contain multiple metering points
- in onboarding, all metering points use the same address as the member
- differing metering point addresses are maintained later in eegFaktura
- tariffs, roles, and account information are not managed in onboarding

### Status values — three places, one source of truth

The set of allowed application statuses is enforced in **three independent
locations** that must be kept in sync. Forgetting any one of them ships a
production bug that's silent until a real status transition is attempted:

1. **`CLAUDE.md` → "Allowed status values"** — the canonical source. Update
   here first when adding or removing a status; the values list and the
   allowed-transition map are authoritative.
2. **`internal/shared` status constants and `AllowedTransitions` map** —
   server-side validation rejects any transition that's not listed here.
3. **`application_status_check` CHECK constraint on `member_onboarding.application`**
   — PostgreSQL rejects writes of unknown status strings with error 23514
   (`violates check constraint`). The constraint was introduced in
   `db/migrations/000001_initial_schema.up.sql`; every subsequent status
   change needs a new migration that `DROP`s and re-`ADD`s the constraint
   with the updated value set (see `000036_application_status_check_email_confirmed.up.sql`
   for the canonical pattern).

Tests that exercise transitions only against a Go-only fake store will pass
even when the DB constraint is stale — only an end-to-end click on a real
Postgres surfaces the mismatch. When introducing a new status, do one
manual end-to-end run on a staging cluster that has migrate-up applied,
before declaring the feature shippable.

## 6. Technology Decisions

### Frontend
Uses the same frontend/web stack as `eegfaktura-web`.

### Backend
Standalone Go service.

### Database
PostgreSQL, same DB system as eegFaktura, dedicated schema `member_onboarding`.

### Authentication
- Public Web: no login required
- Admin Web: existing Keycloak-based authentication

### API Style
REST with JSON.

### Deployment
Standalone build and standalone migrations in repository `eegfaktura-member-onboarding`.

### Time and Timezone

- PostgreSQL stores every timestamp as UTC (`timestamp with time zone`).
- API responses serialise timestamps in ISO 8601 / RFC 3339, always UTC.
- Every user-visible rendering (PDF, email, admin web) converts to **Europe/Vienna** with automatic CET/CEST handling.
- Backend helper: `internal/shared/timezone.go` (`DisplayLocation`, `FmtDateTime`, `FmtDate`); the package blank-imports `time/tzdata` so the IANA database is embedded into the binary (the Alpine base image does not ship `tzdata` by default).
- Frontend helper: `src/lib/datetime.ts` (`formatDateTime`, `formatDate`, `formatPlainDate`).
- Mail templates expose the same helpers via `template.Funcs` (`{{fmtDateTime …}}`).
- DATE columns (`birth_date`, `membership_start_date`) are timezone-unaware by design — they have no time component.

### Edge / Network Boundary

- **Body size limits** are enforced per route group via the `MaxBodySize` middleware: 256 KiB for `/api/public` and `/api/external`, 1 MiB for `/api/admin`. Decode errors surface as 400.
- **Trusted-proxy CIDRs** (`TRUSTED_PROXY_CIDRS` env var, default in Helm covers the typical K8s pod/service ranges): `X-Real-IP` / `X-Forwarded-For` headers are only honoured when the immediate peer (`r.RemoteAddr`) is inside a trusted CIDR. Otherwise `realIP()` falls back to `RemoteAddr` so a direct pod-callee cannot spoof the per-IP rate limit.
- **NetworkPolicies** (`networkPolicies.enabled` in Helm, default `true`): backend ← frontend + ingress controller, frontend ← ingress controller, postgres ← backend + migrate + seed only. The frontend cannot reach Postgres directly.

### Health Probes

- `GET /livez` (backend) and `GET /api/health` (frontend) return 200 unconditionally — used for kubelet livenessProbe so a transient DB outage cannot trigger a restart loop.
- `GET /readyz` (backend) pings the DB — used for readinessProbe so the pod is dropped from the Service endpoints while the DB is unavailable.
- `GET /health` remains for backwards compatibility (combined liveness+readiness with DB ping).

### Authentication Flow

- Admin frontend obtains a Keycloak JWT via NextAuth; backend validates via JWKS.
- `adminRequest` merges caller-supplied headers on top of the default headers so Authorization is never accidentally dropped.
- A 401 from the backend dispatches a global `auth:expired` window event; `SessionRefreshGuard` calls `signIn("keycloak")` so users hit a real login page instead of stale error banners.
- A sessionStorage-backed 30 s cooldown prevents an infinite redirect loop when a transient 401 (deploy in progress, new pod still loading JWKS) keeps recurring after the signIn roundtrip.

### Member Numbers

- Authoritative source is the eegFaktura core (`participantNumber` column, VARCHAR — values like `A005`, `M-12`, `123`).
- The onboarding does **not** assign numbers at submit time. `application.member_number` stays NULL until import succeeds.
- At import time the admin picks the number in the tariff dialog. The backend pre-fills the suggestion via `GET /api/admin/applications/{id}/next-member-number`, which groups existing core values by prefix + padding, picks the dominant pattern, and emits `<prefix><max+1>` zero-padded to the group's width.
- Pre-import duplicate check (`ImportService.MemberNumberTaken`) compares the chosen value against the core's participant list; surfaces 409 to the dialog.
- Partial UNIQUE index on `(rc_number, member_number) WHERE member_number IS NOT NULL` as defense-in-depth.

### Observability

- Prometheus `/metrics` on a separate port (env `METRICS_PORT`, default 9090). Never routed through the public ingress — only the in-cluster Prometheus pod can scrape it.
- Counters: `applications_submitted_total`, `imports_total{result}`, `mail_sent_total{kind,result}`, `rate_limit_hits_total`, `member_number_lookups_total{result}`. Plus `http_request_duration_seconds{method,status_class}` histogram. The bundled `go_*` and `process_*` collectors come for free.
- Helm: dedicated ClusterIP service with `prometheus.io/scrape` annotations (works with `kubernetes_sd_configs`). Optional `ServiceMonitor` (`metrics.serviceMonitor.enabled`) for prometheus-operator stacks.
- NetworkPolicy: backend pod allows ingress on :9090 from the configured Prometheus namespace (`networkPolicies.prometheusNamespace`, default `cattle-monitoring-system` for Rancher Monitoring).

### Database Performance Indexes

- `application(rc_number, status)` — list filtering by tenant + status (admin landing page)
- `status_log(application_id, created_at)`, `document_consent(application_id, consented_at)`, `metering_point(application_id, created_at)` — the three "list children, ordered by time" queries on every admin detail view
- `(rc_number, member_number) WHERE member_number IS NOT NULL` — partial UNIQUE for duplicate-detection
- Deep pagination is capped at `page = 10000` in the admin list handler so no OFFSET scan can run away.

### Mail Deliverability

- Transactional mails set `Reply-To` to a useful counterparty (EEG contact for member mails, applicant for EEG mails) so replies don't disappear into `noreply@`.
- `Auto-Submitted: auto-generated` header on every outgoing mail (RFC 3834) marks it as automated.
- `User-Agent` and `X-Mailer` are branded `eegFaktura Member Onboarding` (overrides the gomail library default).
- `Message-ID` uses the From-address domain (e.g. `<…@eegfaktura.at>`) instead of `os.Hostname()` (which in K8s is the random pod hash).
- Body structure: `multipart/mixed { multipart/alternative { text/plain, text/html }, application/pdf }` — the plain-text alternative is rendered from the HTML with table-aware formatting (label/value pairs).
- DNS authentication (DKIM `postal-TA3f2w._domainkey.eegfaktura.at`, SPF via `psrp.eegfaktura.at`, DMARC `p=reject` on `eegfaktura.at`) is already in place at the Postal/DNS layer.

## 7. Summary

`eegfaktura Member Onboarding` is implemented as a standalone component closely aligned with eegFaktura.

The architecture consists of:
- Public Web for self-registration
- Admin Web for review and import triggering
- Go backend as the business logic core
- PostgreSQL schema `member_onboarding`
- internal service call to the eegFaktura core for productive transfer
