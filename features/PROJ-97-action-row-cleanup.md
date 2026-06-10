# PROJ-97: Aktions-Reihe im Antrags-Detail aufgeräumt

## Status: Deployed (2026-06-10)
**Created:** 2026-06-10
**Last Updated:** 2026-06-10
**Typ:** UI-Hotfix (Tester-Befund)

## Tester-Befund

> „der Button für Beitrittserklärung herunterladen ist sehr deplaziert.
> weiters sind schon sehr viele button in der reihe. wie können wir
> das optimieren?"

Sieben Aktionen in der Detail-Ansicht waren als flache Button-Reihe
angeordnet, plus eine separate „Beitrittserklärung herunterladen"-
Aktion in einem blauen Vorstands-Workflow-Hinweisblock direkt
darunter.

## Owner-Direktive

3-Dropdown-Refactor + Hinweis-Card entfernen, Verhalten in der Doku
festhalten.

## Implementation

### Frontend (`src/components/admin-application-detail.tsx`)

**Gruppen:**

| Aktion | Wo | Status-Sichtbarkeit |
|--------|----|---------------------|
| Excel | Dropdown *Herunterladen* | DOWNLOAD_AVAILABLE_STATUSES |
| Beitrittsbestätigung | Dropdown *Herunterladen* | DOWNLOAD_AVAILABLE_STATUSES |
| Beitrittserklärung | Dropdown *Herunterladen* | `activated` + `boardApprovalWorkflowEnabled` |
| Bestätigungs-Mail / Bestätigungs-Link | Dropdown *Erneut senden* | immer (Label wechselt je nach E-Mail-Bestätigungs-Pending) |
| SEPA-Mandat | Dropdown *Erneut senden* | `activated` |
| Stammdaten abgleichen + Info-Popover | Top-Level | `activated` |
| Datenweiterleitung | Top-Level | immer |
| Löschen (destruktiv) | Top-Level | `draft` / `rejected` |
| Bearbeiten (primär) | Top-Level | review-stati + `import_failed` |

**Entfällt:** der blaue „Vorstands-Genehmigungs-Workflow aktiv"-
Hinweisblock samt Versanddatum-Anzeige und eigenständigem
„Beitrittserklärung herunterladen"-Button. Stattdessen ist die
Beitrittserklärung als Eintrag im Download-Dropdown verfügbar.

**Imports:**
- `DropdownMenu` + Sub-Komponenten aus `@/components/ui/dropdown-menu`
- `ChevronDown` aus `lucide-react`

**Inline-Feedback-Spans** (Erfolg/Fehler von Resend/Excel/PDF/
Joining-Declaration) bleiben unter den Aktions-Buttons; durch
`basis-full` klemmen sie in die nächste Flex-Zeile, statt zwischen
den Buttons zu hängen.

**Stammdaten-Resync-Popover-Text:** der bisherige Hinweis „klicke
‚SEPA-Mandat erneut senden'" wurde an den neuen Menü-Pfad angepasst:
„unter *Erneut senden → SEPA-Mandat* das Mandat neu versenden".

### Doku

- `docs/user-guide/04-admin-applications.md`: neue Aktions-Tabelle vor
  dem Mitgliedsdaten-Abschnitt.
- `docs/user-guide/06-admin-settings.md`: Beitrittserklärung-Download-
  Beschreibung umgeschrieben — kein blauer Hinweisblock mehr, Eintrag
  im Download-Dropdown.

## Acceptance Criteria

- [x] **AC-1** Drei Dropdowns + vier Top-Level-Buttons konsistent
  über die Status-Lifecycle.
- [x] **AC-2** Beitrittserklärung-Download nicht mehr in einer
  separaten Hinweis-Card.
- [x] **AC-3** Vorstands-Workflow-Hinweisblock entfernt — Verhalten
  in der Doku beschrieben.
- [x] **AC-4** Loading-States (Excel wird erstellt…, Mandat wird
  versandt…) als Dropdown-Item-Label sichtbar.
- [x] **AC-5** Disabled-State bei laufender Aktion + bei fehlenden
  Voraussetzungen (Stammdaten-Resync ohne `targetParticipantId`).
- [x] **AC-6** `npx tsc --noEmit` clean, `npm run build` clean,
  `npx vitest run` 88/88.

## Edge Cases

- **EC-1** Antrag ohne verfügbaren Download (z. B. Status `draft`):
  Download-Dropdown rendert nicht.
- **EC-2** Re-Send-Dropdown ist nie leer (Bestätigungs-Mail ist
  immer ein Eintrag).
- **EC-3** Stammdaten-Resync ohne `targetParticipantId`: Button
  disabled mit Title-Tooltip.
- **EC-4** Vorstands-Genehmigungs-Workflow ohne `activated`-Status:
  Beitrittserklärung-Eintrag bleibt versteckt.

## Out of Scope

- Du/Sie-Konsistenz-Refactor (Owner-Direktive 2026-06-10 mittags):
  separat als PROJ-98 (oder höher), nach Klärung der Mitgliedstyp-
  Konditional-Logik.

## Deployment

**Deploy-Bookkeeping 2026-06-10 (nachmittags):**

- UI-Hotfix-Cycle: direkter Commit, kein eigener /architecture-Pfad
- Code-Commits: `1a9336c` (Dropdown-Refactor) + `92c1263` (Plazierungs-Hotfix)
- Tag: `v1.25.2-PROJ-97` gesetzt + gepusht (auf `1a9336c`)

**Tester-Verifikation:** im Antrags-Detail die drei Dropdowns
checken; in einem `activated`-Antrag mit Vorstands-Workflow
verifizieren, dass die Beitrittserklärung als Dropdown-Eintrag
sichtbar ist und der frühere blaue Hinweisblock weg ist.
