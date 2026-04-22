# Build Plan
## eegfaktura Member Onboarding

## Goal

This document describes the recommended implementation sequence for Claude Code.

## Phase 1: Repository Foundation

Goal:
- a runnable foundation for backend and documentation

Scope:
- set up repository structure
- create `docs/` folder
- create Go service skeleton
- configuration structure
- HTTP router
- health endpoint
- DB connection
- migrations folder

Definition of Done:
- service starts locally
- health endpoint responds
- DB connection is configurable
- project structure is documented

## Phase 2: Database Schema

Goal:
- technically create schema `member_onboarding` and all tables

Scope:
- migration `create schema member_onboarding`
- tables:
  - `member_onboarding.application`
  - `member_onboarding.metering_point`
  - `member_onboarding.status_log`
- constraints
- indexes
- define `updated_at` strategy

Definition of Done:
- migration runs successfully locally
- tables are present
- foreign keys and indexes are set

## Phase 3: Public API

Goal:
- make public registration technically available

Scope:
- `GET /api/public/registration/{rc_number}`
- `POST /api/public/applications`
- `PUT /api/public/applications/{id}`
- `POST /api/public/applications/{id}/submit`
- validation
- persistence in `application`, `metering_point`, `status_log`

Definition of Done:
- application can be created
- application can be updated
- application can be validated and submitted
- status history is written

## Phase 4: Admin API

Goal:
- review and editing by admins

Scope:
- `GET /api/admin/applications`
- `GET /api/admin/applications/{id}`
- `PUT /api/admin/applications/{id}`
- `POST /api/admin/applications/{id}/status`
- filtering and pagination
- EEG authorization check in the backend

Definition of Done:
- list works
- detail view works
- status transitions are validated and logged
- admin note is editable

## Phase 5: Import

Goal:
- import approved applications into eegFaktura

Scope:
- `POST /api/admin/applications/{id}/import`
- import mapping from onboarding to participant payload
- internal core client
- success/error handling
- update import status in `application`
- write `status_log`

Definition of Done:
- import is only allowed at status `approved`
- payload is correctly constructed
- success and failure are stored
- `target_participant_id` is set on success

## Phase 6: Auth and Hardening

Goal:
- production-ready security

Scope:
- Keycloak integration in the admin area
- role/EEG authorization check
- unify error handling
- logging
- basic tests
- complete API documentation

Definition of Done:
- admin endpoints are secured
- errors are consistent
- most important flows are tested

## Prompting Recommendation for Claude Code

Claude Code should always work in small packages.

Recommended sequence:
1. implement Phase 1
2. implement Phase 2
3. implement Phase 3
4. implement Phase 4
5. implement Phase 5
6. implement Phase 6

Recommended working style:
- always read relevant files in `docs/` first
- only implement one phase or one small sub-package at a time
- strictly follow architecture and domain rules
- do not introduce additional features without explicit approval
