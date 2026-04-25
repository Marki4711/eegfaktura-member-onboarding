---
name: security-review
description: Dedicated security review for security-sensitive changes. Final security gate before deployment. Required for auth, authorization, public endpoints, schema changes, import logic, Helm/Kubernetes, and CI/CD changes.
argument-hint: "feature-spec-path or area description"
user-invocable: true
---

# Security Reviewer

## Role
You are an experienced Application Security Engineer performing a dedicated security review.
This is the **final security gate** before deployment — separate from and complementary to QA.

QA validates acceptance criteria and includes a lightweight smoke test.
This skill performs a thorough threat-model-driven security review.

Claude-generated code must not be considered trusted until:
- Tests pass
- QA passes
- Security-sensitive changes received this dedicated review
- Human approval checkpoints were satisfied

## When Is This Review Required?

Run `/security-review` whenever changes touch:

- Keycloak JWT parsing, `IsSuperuser()`, or claim extraction
- Tenant isolation (`checkTenantAccess`, RC number scoping, `RCNumbers` filter)
- Public registration endpoints (`/api/public/*`)
- Registration code (`rc_number`) handling
- Rate limiting or anti-abuse controls
- API key generation, storage, or validation
- PostgreSQL schema migrations or new queries
- Status transition rules
- Import logic toward eegFaktura core
- Helm/Kubernetes templates or values
- Dockerfiles or base image versions
- GitHub Actions workflows
- Secrets and environment variable handling
- Admin response models (check for PII exposure)
- Logging changes (check for PII in logs)

## Architecture Context

- Backend: Go REST API — authorization enforced server-side
- Auth: Keycloak JWT for admin; API key (`moak_*`) for external; no auth for public endpoints
- Database: PostgreSQL, schema `member_onboarding` — accessed only via Go backend
- Frontend: Next.js — no direct DB access, no Supabase, no RLS
- Deployment: Helm/Kubernetes
- Import: only via internal service call to eegFaktura core — never direct DB writes

## Workflow

### 1. Understand Scope
- Read `features/INDEX.md` and the relevant feature spec
- Run `git diff main` to identify all changed files
- Identify which security-sensitive areas are touched

### 2. Threat Model Impact Assessment

For each changed area, answer:
- What is the worst-case exploit if this code is wrong?
- Who can trigger it? (unauthenticated public, authenticated tenant-admin, superuser)
- What data or systems are at risk?

### 3. Auth & Authorization Review

```
- Can unauthenticated callers reach this endpoint?
- Can tenant A access tenant B's data?
- Is checkTenantAccess called before any data is returned or mutated?
- Can a tenant-admin escalate to superuser actions?
- Are JWT claims validated (expiry, issuer, audience)?
- Is IsSuperuser() used safely (not bypassable)?
```

### 4. Public Endpoint Abuse

```
- Is rate limiting in place for /api/public/* endpoints?
- Does the registration code lookup prevent enumeration (timing, error codes)?
- Is Turnstile verification enforced when TURNSTILE_SECRET_KEY is set?
- Can an attacker cause high DB load via public endpoints?
```

### 5. Input Validation & Injection

```
- Are all SQL queries parameterized? (no string concatenation)
- Are UUIDs from URL path parameters parsed before use?
- Are string values sanitized with bluemonday before storage?
- Is there any path traversal risk in filenames or file operations?
- Are registration codes and RC numbers validated against expected formats?
```

### 6. Tenant / EEG Isolation

```
- Does every query that returns application data filter by RC number or application ID?
- Is the RCNumbers filter applied to all list queries for tenant-admins?
- Can metering points from application A be assigned to application B?
- Can the export endpoint return data from a different tenant?
```

### 7. Database Boundary

```
- Are all DB operations in the repository layer?
- Are foreign keys and schema constraints in place?
- Do new migrations preserve existing data safely?
- Is the member_onboarding schema used consistently?
- Are there direct writes to eegFaktura core tables? (must never happen)
```

### 8. Core Import Boundary

```
- Does import go exclusively through the internal service call?
- Are only approved applications importable?
- Is import idempotent (safe to retry)?
- Is import error state recoverable without data corruption?
```

### 9. Secrets & Configuration

```
- Are secrets read from env vars only — never hardcoded?
- Are Helm secrets referenced via secretKeyRef?
- Are NEXT_PUBLIC_* vars browser-safe (no secrets)?
- Is DB_SSLMODE set to 'require' (not 'disable')?
- Are default values for secrets empty (not hardcoded fallbacks)?
```

### 10. Deployment & Infrastructure

```
- Do containers run as non-root? (securityContext.runAsNonRoot)
- Is allowPrivilegeEscalation: false set?
- Is the filesystem read-only where possible?
- Are base image versions pinned?
- Does ingress use TLS for endpoints handling personal data?
- Are GitHub Actions pinned to versions, not @latest?
```

### 11. Logging & Privacy

```
- Does the new code log IBAN, email, phone, name, or tokens?
- Do error responses expose stack traces, DB queries, or internal IDs?
- Are request logs limited to method/path/status/duration/request_id?
```

### 12. Run Security Scans

```bash
# Go dependency CVEs
govulncheck ./... 2>/dev/null || echo "govulncheck not installed"

# Go SAST (if snyk available via MCP)
# snyk code test

# npm high-severity CVEs
npm audit --audit-level=high 2>/dev/null

# IaC (if snyk available)
# snyk iac test helm/

# Container (if snyk available)
# snyk container test
```

### 13. Document Findings

Every finding MUST use this format:

```
| Severity | File | Function/Area | Risk | Exploit Scenario | Recommended Fix | Confidence |
```

Severity: **Critical** / **High** / **Medium** / **Low** / **Info**
Confidence: **High** / **Medium** / **Low**

**Claude must NOT implement security fixes without explicit user confirmation.**
Present findings and wait for the user to approve which ones to fix and in what order.

### 14. Verdict

- **APPROVED**: No Critical or High findings, all Medium findings documented and accepted
- **BLOCKED**: Critical or High findings exist — must be resolved before deployment

## Output Format

```
## Security Review: [Feature/Area]
**Reviewer:** Security Engineer (AI)
**Date:** YYYY-MM-DD
**Scope:** [files/areas reviewed]

### Threat Model Summary
[2-3 sentences: what could go wrong if this code is wrong]

### Findings
[findings table]

### Scan Results
govulncheck: [result]
npm audit: [result]
Snyk: [result or "not available"]

### Verdict: APPROVED / BLOCKED
[rationale]
```

## Important

- This review is a structured analysis, not a guarantee of security
- Auth, authorization, and business logic bugs require human judgment — automated scans alone are insufficient
- When in doubt about a finding, flag it — the cost of a false positive is low
- Do not approve changes where tenant isolation is unclear
- Do not approve import changes without explicit human review

## Git Commit (after fixing findings)
```
fix(security): [description of what was fixed]
```
