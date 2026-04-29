# Backend Implementation Checklist

## Core Checklist
- [ ] Checked existing tables/APIs via git before creating new ones
- [ ] SQL migration files created (`db/migrations/00000X_description.up.sql` + `.down.sql`)
- [ ] Migration tested: `make migrate-up` then `make migrate-down` then `make migrate-up` again
- [ ] Indexes created on performance-critical columns
- [ ] Foreign keys set with appropriate ON DELETE behavior
- [ ] All planned API endpoints implemented in `internal/http/`
- [ ] Swagger annotations added to every new handler function (`@Summary`, `@Tags`, `@Param`, `@Success`, `@Failure`, `@Router`, `@Security` where applicable)
- [ ] `swag init --dir cmd/server,internal/http,internal/shared --output docs --parseDependency --parseInternal` ausgeführt und `docs/` committed
- [ ] Keycloak auth middleware applied on all admin endpoints
- [ ] Tenant isolation enforced (`checkTenantAccess` / `RCNumbers`)
- [ ] Input validation in handler or service layer (not only frontend)
- [ ] Meaningful error responses with correct HTTP status codes
- [ ] No TypeScript errors in frontend API types (`src/lib/api.ts`)
- [ ] All endpoints tested manually (curl or Invoke-RestMethod)
- [ ] No hardcoded secrets in source code
- [ ] `docs/api-spec.md` and `docs/domain-model.md` updated

## Verification (run before marking complete)
- [ ] `go build ./...` passes without errors
- [ ] `go test ./...` passes
- [ ] All acceptance criteria from feature spec addressed in API
- [ ] `features/INDEX.md` status updated to "In Progress"
- [ ] Code committed to git

## Performance Checklist
- [ ] Indexes on all frequently filtered / sorted columns
- [ ] No N+1 queries (use joins or batch queries)
- [ ] List endpoints use pagination
- [ ] DB connection pool configured (SetMaxOpenConns / SetMaxIdleConns)
- [ ] Rate limiting on public-facing APIs
