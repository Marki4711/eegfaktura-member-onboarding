# PROJ-25: Bulk-Aktionen im Admin

## Status: Planned
**Created:** 2026-04-29
**Last Updated:** 2026-04-29

## Dependencies
- PROJ-2 (Admin Review — Status-Transitionen)
- PROJ-3 (Admin Frontend UI)
- PROJ-5 (Keycloak Auth)

## User Stories
- Als EEG-Admin möchte ich mehrere Anträge gleichzeitig genehmigen können, damit ich bei vielen Beitritten nicht jeden Antrag einzeln öffnen muss.
- Als EEG-Admin möchte ich mehrere Anträge gleichzeitig ablehnen können, damit ich Sammelablehnungen effizient durchführen kann.
- Als EEG-Admin möchte ich alle sichtbaren Anträge auf einmal auswählen können, damit ich nicht jeden Antrag einzeln anklicken muss.
- Als EEG-Admin möchte ich eine Vorschau sehen welche Anträge von der Bulk-Aktion betroffen sind, bevor ich bestätige, damit ich keine falschen Anträge versehentlich verarbeite.

## Acceptance Criteria
- [ ] In der Antragsliste kann jeder Antrag per Checkbox ausgewählt werden
- [ ] Eine "Alle auswählen"-Checkbox selektiert alle aktuell sichtbaren (gefilterten) Anträge
- [ ] Bei mindestens einem ausgewählten Antrag erscheint eine Aktionsleiste mit verfügbaren Bulk-Aktionen
- [ ] Verfügbare Bulk-Aktionen: "Genehmigen" (→ approved), "Ablehnen" (→ rejected), "Zur Prüfung" (→ under_review)
- [ ] Nur Aktionen die für den aktuellen Status der ausgewählten Anträge zulässig sind, werden angeboten (ungültige Transitionen werden übersprungen, nicht als Fehler behandelt)
- [ ] Vor der Ausführung erscheint ein Bestätigungsdialog: "X Anträge genehmigen?"
- [ ] Nach der Ausführung zeigt eine Zusammenfassung: X erfolgreich, Y übersprungen (ungültige Transition)
- [ ] Der Backend-Endpunkt validiert jede Status-Transition serverseitig (kein Bypass durch direkte API-Aufrufe)
- [ ] Tenant-Isolation: Admins können nur Anträge ihrer eigenen EEGs in Bulk-Aktionen einschließen

## Edge Cases
- Gemischte Status in der Auswahl (z.B. draft + submitted): Nur Anträge mit gültiger Transition werden verarbeitet, der Rest wird übersprungen
- Gleichzeitige Bearbeitung: Wenn ein anderer Admin einen Antrag in der Zwischenzeit geändert hat, wird dieser Antrag übersprungen (nicht als Fehler, sondern als "übersprungen" gemeldet)
- Leere Auswahl: Aktionsleiste wird ausgeblendet
- Sehr große Auswahl (>100 Anträge): Backend verarbeitet alle, Frontend zeigt Ladeindikator

## Technical Requirements
- Neuer Backend-Endpunkt: `POST /api/admin/applications/bulk-action`
- Request: `{ action: "approve"|"reject"|"under_review", ids: [uuid, ...] }`
- Response: `{ succeeded: [uuid, ...], skipped: [uuid, ...] }`
- Maximale Batch-Größe: 200 Anträge pro Request
- Keycloak-Auth + Tenant-Check für jeden Antrag in der Liste
- Frontend: Checkbox-State in lokalem React-State (kein Server-State nötig)

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
