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

1. **Clone and Install**
   ```bash
   git clone <repository-url>
   cd eegfaktura-member-onboarding
   npm install
   npx playwright install chromium  # For E2E tests
   ```

2. **Database Setup**
   - Ensure PostgreSQL is running
   - Schema `member_onboarding` should exist
   - Run migrations from `db/migrations/`

3. **Environment Configuration**
   - Copy `.env.local.example` to `.env.local`
   - Configure database connection, Keycloak settings

4. **Start Development**
   ```bash
   npm run dev  # Frontend on localhost:3000
   # Backend: follow Go service setup in docs/
   ```

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

Frontend deploys to Vercel. Backend Go service deployment follows eegFaktura infrastructure standards.

See `docs/production/` for production setup guides including error tracking, security headers, and performance optimization.
