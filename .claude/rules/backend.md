---
paths:
  - "internal/**"
  - "cmd/**"
  - "db/migrations/**"
---

# Backend Development Rules

## Stack
Go backend + PostgreSQL (`member_onboarding` schema) — no Supabase, no RLS, no ORM.

## Database
- All tables in `member_onboarding` schema
- Use snake_case column names, UUID primary keys, foreign keys with ON DELETE CASCADE
- Add indexes on columns used in WHERE, ORDER BY, and JOIN (see `docs/domain-model.md`)
- Never use JSON columns — use proper relational columns. **Eine Ausnahme:** `registration_entrypoint.brand_theme JSONB` (PROJ-103) hält die individuelle Theme-Konfiguration der Public-Page (reine Präsentations-Konfiguration, kein Domain-Datum). Gilt nur für dieses eine Feld.
- Every schema change goes in a numbered migration: `db/migrations/00000X_description.up.sql` + `.down.sql`

## Go patterns
- Separate handler, service, and repository layers
- Validate all user input in the handler or service layer — never trust the frontend
- Parse UUID path params with `uuid.Parse` — reject malformed IDs with 400
- Use `database/sql` directly with parameterized queries — no ORM
- Status transitions validated server-side against the allowed map in `shared/`

## Auth
- Admin endpoints protected by `KeycloakAuthMiddleware` — never skip it
- Tenant isolation enforced via `checkTenantAccess` / `RCNumbers` from JWT claims
- Superuser check (`IsSuperuser()`) grants unrestricted access — changes require human approval

## Security
- Never hardcode secrets — environment variables only
- Sanitize HTML input with bluemonday before storage
- Error responses must not leak stack traces, DB details, or internal IDs
