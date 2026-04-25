---
id: PROJ-19
title: Manuelle Aktivierung der Registrierung
status: Deployed
created: 2026-04-25
---

## Summary

EEGs werden beim ersten Admin-Login nicht mehr automatisch für die öffentliche Registrierung aktiviert. Stattdessen startet eine neue EEG als inaktiv (`is_active=false`). Der Admin kann die Registrierung gezielt über den Toggle "Mitgliederregistrierung aktiv" in den EEG-Einstellungen aktivieren.

## User Story

Als EEG-Administrator möchte ich die öffentliche Registrierung meiner EEG gezielt aktivieren und deaktivieren können, damit keine ungewollten Registrierungen eingehen, bevor alles konfiguriert ist.

## Acceptance Criteria

- Neue EEGs starten mit `is_active=false` (kein Auto-Aktivieren bei Login).
- Bestehende EEGs mit `is_active=true` bleiben weiterhin aktiv.
- `GET /api/admin/settings/eeg` liefert `registrationActive` im Response.
- `PUT /api/admin/settings/eeg` mit `registrationActive: true/false` speichert den Wert.
- Der Admin-Bereich zeigt oben in den EEG-Einstellungen einen Toggle "Mitgliederregistrierung aktiv".
- Wenn `registrationActive=false`, erhalten Besucher des Registrierungslinks `410 Gone` (bestehende Logik).

## Implementation Notes

- `UpsertForRCNumbers` inseriert jetzt mit `is_active=FALSE` statt `TRUE`.
- Neue Repo-Methode `SaveIsActive(rcNumber, active)` — analog zu `SaveShowCentralPolicy`.
- `SaveEEGSettings` Handler nimmt `registrationActive *bool` entgegen (optionales Partial-Update).
- Frontend-Toggle ist Teil des EEG-Einstellungen-Formulars (Save-Button), nicht auto-save.
- Keine DB-Migration nötig — `is_active` Spalte existiert bereits.
