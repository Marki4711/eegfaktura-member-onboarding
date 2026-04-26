---
paths:
  - "internal/**"
  - "cmd/**"
  - "src/**"
  - "helm/**"
  - "db/**"
  - ".github/**"
  - "Dockerfile*"
  - "next.config.*"
  - ".env*"
---

# Security Rules

This project uses Go backend + PostgreSQL + Keycloak + Next.js frontend + Helm/Kubernetes.
These rules replace any Supabase/Vercel/RLS assumptions from the starter kit template.

## Architecture Boundaries

- **Never write directly to eegFaktura core tables.** Import only via internal service call.
- **PostgreSQL access only through the Go backend.** No direct DB access from frontend.
- **Admin auth via Keycloak JWT.** No Supabase auth, no custom user tables.
- **Public registration endpoints have no auth by design** — rate-limit and validate all input.
- **External API (`/api/external/*`) uses API-key auth** (Bearer `moak_*`). Keys stored as bcrypt hash.
- **Schema:** all tables in `member_onboarding` schema only.

## Secrets Management

- NEVER commit secrets, API keys, tokens, or passwords to git
- Helm secrets (DB_PASSWORD, SMTP_PASSWORD, NEXTAUTH_SECRET, KEYCLOAK_CLIENT_SECRET, TURNSTILE_SECRET_KEY) must use Kubernetes Secret references — never plain values in values.yaml
- `NEXT_PUBLIC_*` env vars are baked into the browser bundle — only browser-safe, non-secret values allowed
- Document all required env vars in `.env.local.example` with placeholder values only
- Dev seed SQL may use example passwords in comments only — never real credentials

## Authorization

- Every admin endpoint must validate Keycloak JWT via middleware
- Tenant-admin scope: only RC numbers from the JWT `tenant` claim
- Cross-EEG access (tenant A reading tenant B's data) must be prevented — `checkTenantAccess` is the enforcer
- Superuser flag in JWT grants unrestricted access — changes to `IsSuperuser()` require human approval
- Status transitions must be validated server-side; the allowed transition map must not be bypassed

## Input Validation

- Validate ALL user input server-side in the Go handler or service layer
- Never trust frontend validation alone
- UUID path parameters must be parsed with `uuid.Parse` — reject malformed IDs with 400
- Registration code (`rc_number`) must be resolved through `registration_entrypoint` table only — never by reading eegFaktura core
- Sanitize HTML input with bluemonday before storage (intro text, admin notes)

## Public Endpoint Abuse Prevention

- Rate-limit `POST /api/public/applications` (currently 10 req/10 min per IP)
- Protect public registration with Cloudflare Turnstile when `TURNSTILE_SECRET_KEY` is set
- Reject registrations for inactive or unknown RC numbers with 404/410 — do not enumerate valid codes
- Changes to rate-limit configuration require human approval

## Personal Data & Privacy

- IBAN, email, phone, name must not appear in application logs or error responses
- Error responses must not contain stack traces, DB query details, or internal identifiers
- Excel export intentionally contains PII (admin-only, Keycloak-protected) — this is by design
- `slog.Info/Warn/Error` in handlers must log only request metadata (method, path, status, duration, request_id)

## API Response Models

- Admin list response must not expose fields unnecessary for the list view (no IBAN in list)
- Public endpoints must not expose application IDs of other registrants
- Error codes must be generic enough not to reveal internal state to unauthenticated callers

## Container & Infrastructure Security

- Containers must not run as root — add `securityContext.runAsNonRoot: true` and `runAsUser: 1000` in Helm templates
- Set `allowPrivilegeEscalation: false` in all container security contexts
- Set `readOnlyRootFilesystem: true` where possible (requires writable tmp mounts if needed)
- Base images must be kept up-to-date via weekly GitHub Actions rebuild + Dependabot
- Do not use `latest` tag for base images in Dockerfiles — pin to specific minor versions (e.g. `alpine:3.21`)

## Helm / Kubernetes

- Secrets must be declared as `secretKeyRef` in Helm templates — never hardcoded in values.yaml
- Ingress without TLS must not be used with real personal data
- `helm lint` and `kubeconform` must pass before merging Helm changes

## CI/CD & Supply Chain

- GitHub Actions workflows must pin action versions to **full commit SHA** — avoid `@latest` and mutable version tags (e.g. `@v4`); tags can be force-pushed to malicious commits (demonstrated by the 2026-03-19 trivy-action supply chain attack)
- Add the human-readable tag as a comment: `uses: actions/checkout@abc1234 # v6`
- Dependabot (`github-actions` ecosystem, weekly) keeps SHA pins up-to-date automatically — do not bypass it
- Do not add third-party actions without review
- Weekly security rebuilds are scheduled (`0 4 * * 1`) — they rebuild images with updated base layers
- Trivy scans run between local build and push — a compromised base image is caught before the image reaches the registry

## Code Review Triggers (require human approval before merge)

- Any changes to Keycloak JWT parsing or `IsSuperuser()` logic
- Any changes to tenant-isolation logic (`checkTenantAccess`, `RCNumbers` filter)
- Any changes to status transition rules
- Any changes to public registration endpoints
- Any new environment variables (must be documented in `.env.local.example`)
- Any changes to Helm secret handling
- Any changes to Dockerfiles or CI/CD workflows
- Any direct database queries outside the repository layer
- Any changes to import logic toward eegFaktura core

## Security Scanning

Run Snyk (if available via MCP or CLI) for:
- Dependency CVEs: on changes to `go.mod` / `package.json`
- SAST: on changes to `internal/`, `src/`
- IaC: on changes to `helm/`, `Dockerfile*`, `.github/`

govulncheck and `npm audit --audit-level=high` are lightweight alternatives when Snyk is not available.
False-positive suppressions must be documented with a reason — never suppressed silently.
