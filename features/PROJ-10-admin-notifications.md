# PROJ-10: Admin Notifications

## Status: Planned
**Created:** 2026-04-22
**Last Updated:** 2026-04-22

## Dependencies
- Requires: PROJ-5 (Keycloak-secured Admin Area) — notifications are user-bound and tied to the authenticated admin identity

## User Stories

- As an admin, I want to see a dialog with unread notifications after logging in, so that I am informed about relevant changes without having to search for them.
- As an admin, I want to dismiss a notification so that it does not appear again on subsequent logins.
- As an admin working on multiple EEGs, I want to receive notifications separately for each EEG, so that I can clearly see which EEG was affected.
- As a superuser, I want to receive notifications for all EEGs, so that I have a complete overview of system-wide events.
- As an admin sharing access to an EEG with colleagues, I want my own notification state to be independent of theirs, so that each admin manages their own inbox.

## Acceptance Criteria

- [ ] Unread notifications are fetched from the server after login and displayed as a dialog
- [ ] Each notification shows: event type, affected EEG (RC number + name), timestamp, and a short description
- [ ] The admin can dismiss individual notifications; dismissed notifications are stored server-side
- [ ] A dismissed notification does not reappear on subsequent logins or devices
- [ ] Notification state is per-user, not per-device (server-side persistence)
- [ ] If there are no unread notifications, no dialog is shown
- [ ] The system supports multiple notification types via a `type` field without requiring a schema change for new types
- [ ] First supported event type: **EEG activated** — triggered when `registration_entrypoint.is_active` is set to `true`; notifies all admins of that EEG
- [ ] Superusers receive EEG-activated notifications for all EEGs

## Edge Cases

- What if an admin logs in from two devices simultaneously? → Both devices fetch the same unread notifications; dismissing on one device dismisses server-side and the other device will see it as dismissed on next fetch.
- What if a notification is created for an EEG that the admin no longer has access to? → Notification is still shown (it was valid at creation time); no retroactive deletion.
- What if there are many unread notifications (e.g. 20+)? → Show all in a scrollable dialog, or paginate; do not silently drop any.
- What if the notification endpoint is unreachable at login? → Fail silently — the admin proceeds to the dashboard without a dialog; notifications will appear on next successful login.
- What if the same event fires multiple times in quick succession (e.g. is_active toggled off and on)? → Only create a new notification if one for the same event + EEG is not already unread for that user.

## Technical Requirements

- Notification type is stored as a string enum (e.g. `"eeg_activated"`) — new types are added in code without DB migration
- Payload for each notification type is stored as JSONB or structured columns (to be decided in architecture)
- API: `GET /api/admin/notifications` returns unread notifications for the authenticated user
- API: `POST /api/admin/notifications/{id}/dismiss` marks a notification as read
- Notifications are created server-side when the triggering event occurs (e.g. when is_active is set to true)

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
