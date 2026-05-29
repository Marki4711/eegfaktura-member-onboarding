# CLAUDE.md
## eegfaktura Member Onboarding

## Project Overview

This repository contains **eegfaktura Member Onboarding**.

The goal of this component is to support **self-service registration of EEG members**.  
New members should be able to enter their master data through a web form. These data must **not** be written directly into the productive eegFaktura participant data model. Instead, they are stored in a dedicated onboarding data model and reviewed by an admin before they are imported into the eegFaktura core.

This repository is based on the **AI Coding Starter Kit workflow**, but the technical implementation must follow the decisions documented in this repository, not the default template assumptions.

## Behavioral Guidelines

Behavioral guidelines to reduce common LLM coding mistakes. Merge with project-specific instructions as needed.

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

### 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them — don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

### 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

### 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it — don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

### 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

**These guidelines are working if:** fewer unnecessary changes in diffs, fewer rewrites due to overcomplication, and clarifying questions come before implementation rather than after mistakes.

## Repository

- Repository name (private main): `eegfaktura-member-onboarding-private`
- Public mirror: `eegfaktura-member-onboarding` (read-only, gefiltert via Mirror-Workflow)
- Beim Commit von business-sensiblem Material: unter `private/` ablegen ODER
  Frontmatter `visibility: private` in der Datei setzen — der Mirror-Filter
  schließt beides aus.

## Project Structure

Use and maintain a clear repository structure.

Expected structure:

- `docs/` project and technical documentation
- `features/` feature specifications and tracking
- `db/migrations/` SQL migrations
- `cmd/` application entrypoints
- `internal/` Go application code
- `.claude/rules/` Claude rules
- `.claude/skills/` Claude skills from the starter kit

Recommended Go structure:

- `cmd/server/` application entrypoint
- `internal/config/`
- `internal/http/`
- `internal/application/`
- `internal/meteringpoint/`
- `internal/statuslog/`
- `internal/importing/`
- `internal/coreclient/`
- `internal/shared/`
- `db/migrations/`

Do not invent a completely different structure without checking the existing repository first.

## Source of Truth

Always treat the repository documentation as the source of truth.

Read these files before making changes:

- `docs/architecture.md`
- `docs/domain-model.md`
- `docs/api-spec.md`
- `docs/import-mapping.md`
- `docs/PRD.md` if present
- `features/INDEX.md`
- the specific feature file you are implementing

## Binding Architecture Decisions

### Components

The solution consists of:

- Public Web
- Admin Web
- Member Onboarding Backend
- PostgreSQL schema `member_onboarding`
- internal import into the eegFaktura core

### System Boundaries

- Frontends talk only to the Member Onboarding backend
- only the backend accesses the database
- only the backend calls the eegFaktura core internally
- no direct writes to eegFaktura core tables
- no direct Public Web or Admin Web access to the core

### Technologies

- Frontend: the same web stack as `eegfaktura-web`
- Backend: **Go**
- Database: **PostgreSQL**
- Persistence: **same PostgreSQL database as eegFaktura, but separate schema `member_onboarding`**
- Admin authentication: **Keycloak**
- API style: **REST + JSON**

### Important Constraint

This repository uses the **starter kit workflow**, but it does **not** use Supabase as the backend architecture by default.

Do not assume:
- Supabase backend
- Supabase auth
- Supabase RLS
- Vercel deployment defaults
- template-specific database shortcuts

Use the documented architecture of this project instead.

## Domain Model

This module uses a deliberately reduced relational model without JSON fields.

Tables:

- `member_onboarding.registration_entrypoint` — maps EEG RC numbers to eeg_id; used as the local lookup table for public registration
- `member_onboarding.application`
- `member_onboarding.metering_point`
- `member_onboarding.status_log`

No JSON fields are used in the database.

## Important Business Rules

- One application contains exactly one member.
- One application belongs to exactly one EEG.
- One application is started through a public RC number link per EEG.
- The RC number is resolved through `member_onboarding.registration_entrypoint` — never by reading eegFaktura core tables directly.
- One application can contain multiple metering points.
- Each metering point may either inherit the member's primary address (default) or carry its own deviating address (PROJ-39). The four `address_*` columns on `metering_point` are all-or-nothing — either all four NULL or all four set; enforced server-side.
- Only applications in status `approved` may be imported.

## Status Model

Allowed status values:

- `draft`
- `submitted`
- `email_confirmed` *(PROJ-31, only reached when EEG opts in)*
- `under_review`
- `needs_info`
- `approved`
- `rejected`
- `imported`
- `import_failed`
- `awaiting_bank_confirmation` *(PROJ-46, only at `einzugsart=b2b`, set automatically by the import service)*
- `ready_for_activation` *(PROJ-46, set automatically by import service for non-b2b, or by admin after bank-confirmation)*
- `activated` *(PROJ-46, end state — strictly no transitions out)*

Allowed transitions:

- `draft -> submitted`
- `submitted -> under_review`
- `submitted -> email_confirmed` *(PROJ-31, only via member click on the confirmation link — `POST /api/public/applications/confirm-email`. Not exposed on the admin `/status` endpoint.)*
- `submitted -> rejected` *(PROJ-31, admin override for obvious junk before confirmation)*
- `email_confirmed -> under_review`
- `email_confirmed -> needs_info`
- `email_confirmed -> approved`
- `email_confirmed -> rejected`
- `under_review -> needs_info`
- `under_review -> approved`
- `under_review -> rejected`
- `needs_info -> submitted`
- `approved -> imported`
- `approved -> import_failed`
- `approved -> rejected` *(2026-05-29, Tester-Wunsch: nach `POST /reset-import` landet der Antrag wieder in `approved`; der Admin braucht dort die Ablehn-Option (Mitglied zurückgezogen, Daten-Qualität, …). Pflicht-Grund wie bei jedem Reject. `member_number` wurde durch `ResetImportTx` bereits genullt — keine Extra-Clearing-Logik.)*
- `approved -> activated` *(PROJ-53, admin manuell als Ausnahmefall — Mitglied existiert im Core bereits und wurde dort manuell mit den Onboarding-Daten überschrieben; überspringt den Import-Pfad. Mitgliedsnummer-Eingabe im Admin-UI erforderlich.)*
- `import_failed -> approved`
- `imported -> awaiting_bank_confirmation` *(PROJ-46, auto-transition by import service when `einzugsart=b2b`. Not exposed on `/status`.)*
- `imported -> ready_for_activation` *(PROJ-46, auto-transition by import service for non-b2b einzugsarten. Not exposed on `/status`.)*
- `awaiting_bank_confirmation -> ready_for_activation` *(PROJ-46, admin manuell after member confirms bank coordination)*
- `awaiting_bank_confirmation -> under_review` *(PROJ-46, admin rückwärts-Übergang)*
- `ready_for_activation -> activated` *(PROJ-46, admin manually OR via activation-check batch. Seit PROJ-53 entscheidet die per-EEG-Einstellung `activation_mode` über das Batch-Kriterium: `participant_active` (Default — Core-Teilnehmer-Status `ACTIVE`) oder `any_meter_registration_started` (min. ein Zählpunkt mit `processState ∈ PENDING/APPROVED/ACTIVE`). Beim Übergang wird die Beitrittsbestätigungs-Mail mit PDF versandt.)*
- `ready_for_activation -> under_review` *(PROJ-46, admin rückwärts-Übergang)*
- `imported -> approved` *(PROJ-30, only via dedicated `POST /reset-import` endpoint, never via generic `/status`)*
- `awaiting_bank_confirmation -> approved` *(PROJ-46, via `POST /reset-import`)*
- `ready_for_activation -> approved` *(PROJ-46, via `POST /reset-import`)*

When `registration_entrypoint.require_email_confirmation = TRUE` (PROJ-31), the
generic admin `/status` endpoint rejects `submitted -> under_review|needs_info|approved`
with 409 until the member has clicked the confirmation link. `submitted -> rejected`
remains available as the admin's anti-spam override.

## Explicit Non-Goals

These topics are **not part of version 1** and must not be introduced implicitly:

- document management
- tariff management
- role management
- account or payment information
- tax data
- direct core database writes
- custom admin user management
- Supabase-specific backend architecture
- Vercel assumptions as default deployment model
- extra features not documented in `docs/` or `features/`

## Database Rules

- schema name: `member_onboarding`
- table names:
  - `member_onboarding.registration_entrypoint`
  - `member_onboarding.application`
  - `member_onboarding.metering_point`
  - `member_onboarding.status_log`
- use `snake_case` for PostgreSQL tables and columns
- use UUID primary keys
- define foreign keys
- apply recommended indexes from `docs/domain-model.md`
- keep `updated_at` consistent
- do not use camelCase column names in PostgreSQL
- do not add undocumented tables

## Backend Rules

When writing backend code:

- use idiomatic Go
- separate handler, service, and repository layers
- keep SQL migrations as files in `db/migrations/`
- validate status transitions on the server side
- validate business rules on the server side, not only in the frontend
- encapsulate core import logic in dedicated packages such as `coreclient` and `importing`
- do not introduce hidden assumptions about core fields if they are not documented
- prefer small, explicit services over overly generic abstractions

## API Rules

- implement REST endpoints according to `docs/api-spec.md`
- use JSON request and response bodies
- use a consistent error model
- validate all input server-side
- do not add undocumented endpoints
- do not silently change request or response models

## Import Rules

Import into eegFaktura happens only through an **internal service call to the core**.

Important rules:

- never write directly to core tables
- follow `docs/import-mapping.md`
- keep onboarding deliberately reduced
- tariffs, roles, account information, tax data, and similar fields are not managed in V1
- missing or unclear core defaults must not be invented without explicit documentation

## Frontend Rules

- follow the same frontend technology direction as `eegfaktura-web`
- do not assume template-specific frontend structures unless they are actually present in this repository
- do not invent UI functionality outside the documented scope
- keep Public Web and Admin Web responsibilities separate

## Development Workflow

Use the starter kit workflow.

Recommended flow for new work:

```
/requirements   → clarify and specify the feature
/architecture   → design decisions
/frontend       → UI components
/backend        → API, database, services
/qa             → acceptance criteria, regression, E2E, security smoke test
/security-review → dedicated security gate (required for security-sensitive changes)
/deploy         → deployment bookkeeping and tag
```

Always work from a documented feature spec in `features/`.

**Claude-generated code must not be considered trusted until:**
- tests pass
- `/qa` passes
- security-sensitive changes received `/security-review`
- human approval checkpoints were satisfied

`/security-review` is required whenever changes touch:
Keycloak auth, tenant isolation, public endpoints, rate limiting, DB schema,
status transitions, import logic, Helm/Kubernetes, Dockerfiles, CI/CD, or secrets.

## Security Workflow mit Snyk

Dieses Projekt nutzt Snyk als ergänzenden Security-Scanner.

Bei sicherheitsrelevanten Änderungen führt Claude Code folgende Scans aus:

1. **Dependency-Änderungen** (`go.mod`, `package.json`): Snyk Open Source / `govulncheck` / `npm audit`
2. **App-Code-Änderungen** (`internal/`, `src/`): Snyk Code (SAST) wenn verfügbar
3. **Infrastruktur-Änderungen** (`helm/`, `Dockerfile*`, `.github/`): Snyk IaC / Container

Ablauf:
1. Scan vor der Änderung (Baseline)
2. Änderung implementieren
3. Tests, Lint, Build ausführen
4. Scan nach der Änderung
5. High- und Critical-Findings priorisieren und beheben
6. Findings nicht ohne dokumentierte Begründung ignorieren
7. Keine pauschalen Suppressions ohne explizite Begründung

Snyk ist über MCP in Claude Code einbindbar. Konfiguration lokal (nicht ins Repository committen):
```json
{
  "mcpServers": {
    "Snyk": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "snyk@latest", "mcp", "-t", "stdio"],
      "env": {}
    }
  }
}
```

Keine Secrets oder Snyk-Tokens ins Repository committen.
Details: `docs/security.md`

## Feature Tracking

All feature work must be tracked in:

- `features/INDEX.md`

Each feature must have exactly one spec file, for example:

- `features/PROJ-1-public-registration.md`

Before starting implementation work:

1. read `features/INDEX.md`
2. identify the current feature
3. read the corresponding feature file
4. only implement the documented scope

Update feature status when work is completed.

## Key Conventions

- Feature IDs use the format: `PROJ-1`, `PROJ-2`, ...
- One feature per spec file
- Keep features small and single-purpose
- Do not mix multiple major concerns in one implementation step
- Keep human approval checkpoints for:
  - schema changes
  - import changes
  - authentication changes
  - architectural changes

Recommended commit format:

- `feat(PROJ-X): short description`
- `fix(PROJ-X): short description`
- `refactor(PROJ-X): short description`

## Build and Test Commands

Use the repository's actual commands once they exist.

Typical examples may include:

- `make run`
- `make test`
- `make lint`
- `make migrate-up`

If commands are not yet implemented:
- do not invent production commands silently
- either add them explicitly to the repository
- or document clearly what is still missing

## Working Style

Work in small, clearly scoped steps.

Before generating code:

1. read the relevant docs
2. check `features/INDEX.md` for the current feature and its dependencies
3. read the relevant feature file in `features/`
4. implement only the requested scope
5. do not introduce extra features
6. keep changes aligned with the documented architecture

## Priority Order for Conflicts

If conflicts occur, use this priority order:

1. `CLAUDE.md`
2. feature file currently being implemented
3. `docs/api-spec.md`
4. `docs/domain-model.md`
5. `docs/import-mapping.md`
6. `docs/architecture.md`

If something is unclear, do not invent a new architecture.  
Stay close to the documented scope and make the smallest safe decision.