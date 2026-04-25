---
id: PROJ-18
title: Datenschutzerklärung & Central Policy Toggle
status: Deployed
created: 2026-04-25
---

## Summary

Two related improvements to legal document handling:

1. A static `/datenschutz` page was generated and hosted within the app so that EEGs without their own privacy policy URL have a sensible fallback link.
2. A per-EEG toggle in the Admin → Rechtsdokumente section allows admins to hide the central operator privacy policy from the public registration form — for EEGs that have configured their own privacy policy as a custom document.

## User Stories

- As a member, I can click a "Datenschutzerklärung" link in the registration form and reach a DSGVO-compliant privacy policy page.
- As an EEG admin, I can deactivate the default privacy policy in the registration form when my EEG provides its own.

## Acceptance Criteria

- `GET /datenschutz` serves a static, DSGVO-compliant German privacy policy page (no auth required).
- The privacy policy link in the registration form points to `/datenschutz` when no `CENTRAL_POLICY_URL` env var is set.
- When `CENTRAL_POLICY_URL` is set, the link uses that URL.
- The admin Rechtsdokumente section shows a toggle "Standard-Datenschutzerklärung im Registrierungsformular anzeigen" (default: on).
- When toggled off, the central policy entry is removed from the public registration config response.
- The toggle state is persisted in `registration_entrypoint.show_central_policy` (migration 000019).
- When `showCentralPolicy = false`, the `privacyAccepted` checkbox is hidden and pre-set to `true` so submission still works.

## Implementation Notes

- New DB column: `registration_entrypoint.show_central_policy BOOLEAN NOT NULL DEFAULT true` (migration 000019).
- Backend `GetRegistrationConfig` only appends the central policy when `ep.ShowCentralPolicy && centralPolicy.URL != ""`.
- `GET /api/admin/settings/eeg` and `PUT` both include `showCentralPolicy`; the PUT uses `*bool` to allow partial updates.
- `src/app/datenschutz/page.tsx` — static Next.js server component, no auth, uses `PublicHeader`.
- `admin-legal-documents-editor.tsx` — toggle below the document list with optimistic update + rollback.
