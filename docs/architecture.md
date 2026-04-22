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

### 3.5 eegFaktura Core
The core remains the authoritative system for productive participant data.

Responsibilities:
- final business validation during import
- productive creation of the participant
- returning a target ID or error message

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

The module uses a deliberately reduced relational model without JSON fields and without document management.

Tables:
- `member_onboarding.registration_entrypoint`
- `member_onboarding.application`
- `member_onboarding.metering_point`
- `member_onboarding.status_log`

Basic rules:
- one application contains exactly one member
- one application belongs to exactly one EEG
- the public entry point is identified by the EEG's RC number
- the RC number is resolved exclusively via `member_onboarding.registration_entrypoint`; no direct access to eegFaktura core tables
- one application can contain multiple metering points
- in onboarding, all metering points use the same address as the member
- differing metering point addresses are maintained later in eegFaktura
- tariffs, roles, and account information are not managed in onboarding

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

## 7. Summary

`eegfaktura Member Onboarding` is implemented as a standalone component closely aligned with eegFaktura.

The architecture consists of:
- Public Web for self-registration
- Admin Web for review and import triggering
- Go backend as the business logic core
- PostgreSQL schema `member_onboarding`
- internal service call to the eegFaktura core for productive transfer
