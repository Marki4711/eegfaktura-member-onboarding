# eegfaktura Member Onboarding

> Self-service registration system for EEG members with admin review workflow.

This repository contains the **eegfaktura Member Onboarding** component, enabling potential EEG members to register themselves through a web form. Submitted data is stored in a dedicated onboarding schema and reviewed by admins before import into the eegFaktura core system.

## Overview

- **Purpose**: Self-service registration for EEG members
- **Workflow**: Public registration → Admin review → Import to eegFaktura core
- **Architecture**: Next.js frontend + Go backend service
- **Database**: PostgreSQL (existing), schema `member_onboarding`
- **Authentication**: Keycloak for admin access
- **Integration**: Internal service calls to eegFaktura core

## Features

### MVP Scope
- Public self-service registration form
- Collection of member master data and metering points
- Admin review and approval workflow
- Status tracking and controlled import
- Keycloak-secured admin area

### Out of Scope
- Document management
- Tariff or role management in onboarding
- Direct writes to eegFaktura core tables

## Tech Stack

| Component | Technology | Details |
|-----------|------------|---------|
| **Frontend** | Next.js 16 | React, TypeScript, Tailwind CSS, shadcn/ui |
| **Backend** | Go | REST API service |
| **Database** | PostgreSQL | Schema `member_onboarding` |
| **Auth** | Keycloak | Admin authentication |
| **Deployment** | Vercel | Frontend hosting |

## Project Structure

```
eegfaktura-member-onboarding/
├── docs/                          # Project documentation
│   ├── PRD.md                     # Product Requirements Document
│   ├── architecture.md            # Technical architecture
│   └── build-plan.md              # Implementation phases
├── features/                      # Feature specifications
│   ├── INDEX.md                   # Feature tracking
│   └── PROJ-X-*.md                # Individual feature specs
├── src/                           # Frontend source code
│   ├── app/                       # Next.js App Router pages
│   ├── components/                # React components
│   └── lib/                       # Utilities and configurations
├── db/migrations/                 # Database migrations (Go backend)
├── cmd/                           # Go application entrypoints
├── internal/                      # Go application code
└── .claude/                       # AI development workflow
```

## Development Workflow

This project uses the AI Coding Starter Kit workflow with specialized skills:

1. **Requirements** (`/requirements`) - Define features with user stories and acceptance criteria
2. **Architecture** (`/architecture`) - Design technical approach (PM-friendly)
3. **Frontend** (`/frontend`) - Build UI components with Next.js and shadcn/ui
4. **Backend** (`/backend`) - Implement Go APIs and database schemas
5. **QA** (`/qa`) - Test features against criteria + security audit
6. **Deploy** (`/deploy`) - Deploy to production with checks

### Getting Started

#### Backend (Go service)

All commands must be run from the repository root.

**1. Start PostgreSQL**

```bash
docker compose up -d
```

**2. Run database migrations**

```bash
# bash / Make
make migrate-up

# PowerShell (no Make)
$env:DATABASE_URL = "postgres://postgres:password@localhost:5432/member_onboarding?sslmode=disable"
go run ./cmd/migrate -direction up

# Roll back one step
go run ./cmd/migrate -direction down
```

The migration runner is a plain Go program (`cmd/migrate/main.go`) that uses the
golang-migrate library directly — no external `migrate` binary is needed.
Override the default DATABASE_URL if your PostgreSQL uses different credentials:

```bash
make migrate-up DATABASE_URL=postgres://user:pass@host:5432/dbname?sslmode=disable
```

**3. Seed local dev data**

After migrations, insert a test registration entry point so the happy path is
testable without any admin UI:

```powershell
# PowerShell
$env:PGPASSWORD = "password"
psql -h localhost -U postgres -d member_onboarding -f db/seeds/dev_seed.sql
```

```bash
# bash / Make
PGPASSWORD=password psql -h localhost -U postgres -d member_onboarding -f db/seeds/dev_seed.sql
```

This inserts one active entry: **RC number `RC123456`**, EEG ID `00000000-0000-0000-0000-000000000001`.

**4. Build and run the Go server**

```bash
# Copy and adjust environment
copy .env.example .env.local     # Windows
# cp .env.example .env.local     # bash

go run ./cmd/server              # run directly
# or
go build -o bin/server.exe ./cmd/server && ./bin/server.exe
```

The server reads configuration from environment variables (see `.env.example`).
Default port is `8080`. Health check: `GET http://localhost:8080/health`.

**One-shot local setup**

```bash
make dev-setup   # docker compose up -d + migrate-up
make run         # go run ./cmd/server
# then seed: PGPASSWORD=password psql ... -f db/seeds/dev_seed.sql
```

**5. Test the public registration flow**

```powershell
# 1. Look up the registration entry point
Invoke-RestMethod -Uri http://localhost:8080/api/public/registration/RC123456

# 2. Create an application
$body = @{
    rcNumber             = "RC123456"
    firstname            = "Josef"
    lastname             = "Brandstatter"
    email                = "josef@example.org"
    residentStreet       = "Flurweg"
    residentStreetNumber = "2"
    residentZip          = "4331"
    residentCity         = "Naarn"
    residentCountry      = "AT"
    privacyAccepted      = $true
    privacyVersion       = "2026-01"
    accuracyConfirmed    = $true
    communicationConsent = $false
    meteringPoints       = @(@{ meteringPoint = "AT0031000000000000000000990022105"; direction = "CONSUMPTION" })
} | ConvertTo-Json -Depth 5

Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/public/applications `
    -ContentType "application/json" -Body $body
```

Expected responses:
- `GET /api/public/registration/RC123456` → 200 with `rcNumber`, `eegId`, `active: true`
- `POST /api/public/applications` → 201 with `id`, `referenceNumber`, `status: "draft"`
- Unknown RC number → 404
- Inactive RC number → 410

---

#### Frontend (Next.js)

1. **Clone and Install**
   ```bash
   git clone <repository-url>
   cd eegfaktura-member-onboarding
   npm install
   npx playwright install chromium  # For E2E tests
   ```

2. **Environment Configuration**
   - Copy `.env.local.example` to `.env.local`
   - The only required variable is `NEXT_PUBLIC_API_URL` (defaults to `http://localhost:8080`)

3. **Start Development**
   ```bash
   npm run dev  # Frontend on localhost:3000
   ```

   Open `http://localhost:3000` in the browser. Enter RC number `RC123456` (seeded by `dev_seed.sql`) to access the registration form.

### Feature Development

Features are tracked in `features/INDEX.md`. To add a new feature:

```
/requirements I want to add email notifications for application status changes
```

The skill will create a feature spec and update tracking. Then proceed with architecture and implementation.

## Build & Test

```bash
npm run dev          # Development server
npm run build        # Production build
npm run lint         # ESLint
npm run test         # Unit tests (Vitest)
npm run test:e2e     # E2E tests (Playwright)
npm run test:all     # All test suites
```

## Deployment

### Container Images

Both services are published to Docker Hub on every push to `main`:

| Image | Docker Hub |
|-------|-----------|
| Backend | `marki4711/eegfaktura-member-onboarding-backend` |
| Frontend | `marki4711/eegfaktura-member-onboarding-frontend` |

**Tags:** `latest` (default branch) and short git SHA (e.g. `abc1234`).

#### GitHub Secrets and Variables Required

| Name | Type | Purpose |
|------|------|---------|
| `DOCKERHUB_USERNAME` | Secret | Docker Hub login |
| `DOCKERHUB_TOKEN` | Secret | Docker Hub access token |
| `NEXT_PUBLIC_API_URL` | Repository variable | Backend URL baked into the frontend image at build time |

Set these under **Settings → Secrets and variables → Actions** in the GitHub repository.

#### Building images locally

```bash
# Backend
docker build -f Dockerfile.backend -t marki4711/eegfaktura-member-onboarding-backend .

# Frontend (set API URL for the target environment)
docker build -f Dockerfile.frontend \
  --build-arg NEXT_PUBLIC_API_URL=https://api.example.com \
  -t marki4711/eegfaktura-member-onboarding-frontend .
```

The frontend image requires `NEXT_PUBLIC_API_URL` at **build time** because Next.js bakes `NEXT_PUBLIC_*` variables into the static bundle. Pass the correct URL for each target environment.

Backend Go service deployment follows eegFaktura infrastructure (Kubernetes) standards.

### Kubernetes Test Environment

The test installation is managed with **Helm**. The chart lives in [`helm/member-onboarding/`](helm/member-onboarding/).

**Hostname:** `member-onboarding-test.eegfaktura.at`  
Ingress routes `/api` → Go backend, `/` → Next.js frontend.

The backend image ships two binaries (`/app/server`, `/app/migrate`) and the migration files (`/app/db/migrations`). No Go installation is needed on the target machine.

#### Chart overview

| Template | What it creates |
|----------|----------------|
| `namespace.yaml` | Namespace `eegfaktura-member-onboarding-test` |
| `secrets.yaml` | `postgres-secret`, `backend-secret` (passwords + SMTP) |
| `postgres.yaml` | PostgreSQL 16 StatefulSet + PVC + Service |
| `migrate-job.yaml` | Helm hook (`post-install`, `pre-upgrade`) — runs migrations automatically |
| `seed-job.yaml` | Helm hook (`post-install`, once) — inserts RC123456 test data |
| `backend.yaml` | Go backend Deployment + Service |
| `frontend.yaml` | Next.js frontend Deployment + Service |
| `ingress.yaml` | Two Ingress objects (web + api) |

#### First-time setup

**1. Secrets file**

```bash
cp helm/member-onboarding/values-secret.yaml.example helm/member-onboarding/values-secret.yaml
# Edit values-secret.yaml — set postgresPassword and dbPassword to the same strong password
```

`values-secret.yaml` is listed in `.helmignore` and never committed.

**2. Install**

```bash
helm install eegfaktura-member-onboarding ./helm/member-onboarding \
  --create-namespace \
  -f helm/member-onboarding/values-secret.yaml
```

Helm will:
1. Deploy namespace, secrets, postgres, backend, frontend, ingress
2. Run the migration Job (waits for postgres, then applies all migrations)
3. Run the seed Job (inserts RC123456 test data, idempotent)

#### Upgrade (new image or config change)

```bash
helm upgrade eegfaktura-member-onboarding ./helm/member-onboarding \
  -f helm/member-onboarding/values-secret.yaml
```

On upgrade, the migration Job runs automatically before the backend is updated (`pre-upgrade` hook).

#### SMTP aktivieren

In `values-secret.yaml` ergänzen:

```yaml
backend:
  smtp:
    host: smtp.example.com
    port: "587"
    user: noreply@example.com
    from: noreply@example.com
secrets:
  smtpPassword: "FILL_IN"
```

Dann `helm upgrade` ausführen.

#### Rollback

```bash
helm rollback eegfaktura-member-onboarding   # rollt auf die vorige Revision zurück
```

#### Uninstall

```bash
helm uninstall eegfaktura-member-onboarding
# PVC wird nicht automatisch gelöscht — Postgres-Daten bleiben erhalten
kubectl delete pvc -n eegfaktura-member-onboarding-test --all
```

#### DNS

Point `member-onboarding-test.eegfaktura.at` to the cluster's nginx ingress controller IP/load balancer before installing the chart.

The test ingress runs without TLS. Access the installation at `http://member-onboarding-test.eegfaktura.at`.

#### Note on NEXT_PUBLIC_API_URL

The frontend image bakes `NEXT_PUBLIC_API_URL` at build time. The `:latest` image is built with this variable unset (empty), so the browser makes relative `/api/...` requests — which the ingress routes correctly to the backend. No env var override is needed in the pod spec.

If you need to rebuild with a specific API URL, set `vars.NEXT_PUBLIC_API_URL` in the GitHub repository variables and re-trigger the workflow.
