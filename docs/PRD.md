# Product Requirements Document
## eegfaktura Member Onboarding

## 1. Vision

`eegfaktura Member Onboarding` enables self-service registration of members for EEGs managed in eegFaktura.

Today, new members are created manually by an administrator in eegFaktura. The goal of this product is to reduce manual effort by allowing members to submit their own basic data through a web form.

Submitted data must first stay in a dedicated onboarding workflow. Only after admin review and explicit approval may the data be imported into the eegFaktura core.

## 2. Target Users

### Public user
A potential new EEG member who wants to register through a fixed registration link identified by the EEG's RC number.

### EEG admin
A user who reviews applications for a specific EEG and decides whether an application is complete and ready for import.

## 3. MVP Scope

Version 1 includes:

- fixed registration link per EEG, identified by the EEG's RC number
- RC number resolved locally through `member_onboarding.registration_entrypoint` — no direct reads from eegFaktura core tables
- public self-service form
- collection of member master data
- collection of multiple metering points
- admin review flow
- status handling
- controlled import into eegFaktura core
- dedicated onboarding persistence in schema `member_onboarding`

## 4. Core Features (Roadmap)

| Priority | Feature | Status | Spec |
|----------|---------|--------|------|
| P0 (MVP) | Public Registration | Approved | `features/PROJ-1-public-registration.md` |
| P0 (MVP) | Admin Review | Approved | `features/PROJ-2-admin-review.md` |
| P0 (MVP) | Admin Frontend UI | Planned | `features/PROJ-3-admin-frontend-ui.md` |
| P0 (MVP) | Core Import | Architected | `features/PROJ-4-core-import.md` |
| P1 | Keycloak-secured Admin Area | Planned | — |
| P2 | Email Notifications | Planned | `features/PROJ-6-email-notifications.md` |
| P2 | Externe Registrierungs-API | Planned | `features/PROJ-13-external-registration-api.md` |
| P2 | Cloudflare Turnstile Spam-Schutz | Planned | `features/PROJ-16-turnstile-spam-protection.md` |
| P2 | Stammdaten-Import aus eegFaktura-Excel | Planned | `features/PROJ-23-stammdaten-import.md` |

## 5. Success Metrics

The MVP is successful if:

- new members can create and submit an application themselves
- admins can review submitted applications
- admins can approve or reject applications
- approved applications can be imported into eegFaktura core
- no productive participant is created before explicit import
- the onboarding workflow remains simpler than the existing manual participant creation flow

## 6. Constraints

- The component must follow the same technology direction as eegFaktura.
- The backend must be implemented in Go.
- The frontend must follow the same web stack direction as `eegfaktura-web`.
- The onboarding data must use the existing PostgreSQL database with a separate schema `member_onboarding`.
- Admin authentication must use Keycloak.
- Import into eegFaktura core must happen through an internal service call only.
- Version 1 must stay deliberately smaller than the full participant model in eegFaktura.

## 7. Non-Goals

Version 1 does not include:

- email notifications on application submission — implemented in PROJ-6
- document uploads
- tariff selection
- role selection
- account or payment data
- tax data
- separate metering point addresses
- public account/login management
- direct writes into eegFaktura core tables
- bidirectional sync between onboarding and core

## 8. Core Business Rules

- one application contains exactly one member
- one application belongs to exactly one EEG
- the public registration entry point is identified by the EEG's RC number
- the RC number is resolved through `member_onboarding.registration_entrypoint`; the onboarding backend never reads EEG data directly from eegFaktura core tables
- one application can contain multiple metering points
- all metering points use the same address as the member in onboarding
- only approved applications may be imported
- imported participants are created in eegFaktura core, not directly in onboarding
- tariff, role, and similar business details are completed later in eegFaktura

## 9. Notes for Feature Work

Use this PRD as the high-level product context.

Detailed implementation work should be tracked via:
- `features/INDEX.md`
- individual feature specification files in `features/`
