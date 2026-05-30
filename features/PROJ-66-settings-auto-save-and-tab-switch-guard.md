# PROJ-66 — Auto-Save für Settings-Editoren + Tab-Switch-Schutz

**Status:** Planned
**Created:** 2026-05-30
**Owner:** TBD
**Source:** Tester-Feedback 2026-05-30 — „auf manchen Settings-Seiten gibt es Speichern-Button, auf anderen nicht. Kann das automatisch gespeichert werden?"

## Hintergrund

Heutige UX im Settings-Tab-Container ist inkonsistent:

| Editor | Save-Pattern |
|---|---|
| EEG-Stammdaten & SEPA | expliziter Save-Button („Konfiguration speichern") — ~30 zusammenhängende Felder |
| Einleitungstext | expliziter Save-Button — Tiptap-Editor |
| Formular-Felder | expliziter Save-Button — ~30 atomare Toggle-Klicks |
| Rechtsdokumente | kein Sammel-Save — jede Aktion persistiert sofort (CRUD) |
| Externe API | kein Save-Konzept — Generate/Revoke sind Aktionen |
| Datenweiterleitung | eigene Save-Buttons in Plugin-Dialogen |

Rechtsdokumente und Externe API sind aktions-basiert — kein Save-Button ist hier UX-konform. Die drei Save-Button-Editoren haben aber zwei reale Probleme:

1. **Daten-Verlust-Risiko:** Admin schaltet Toggles um, wechselt den Tab, vergisst Save — Änderungen verloren ohne Warnung.
2. **Reibung bei häufigen Klicks:** Formular-Felder-Editor mit ~30 atomaren Toggles ist klassischer Fall für Auto-Save — der Save-Button am Ende fühlt sich aus heutiger UX-Sicht wie ein Anachronismus an.

## User Stories

- **Als EEG-Admin** möchte ich nicht riskieren, ungespeicherte Änderungen zu verlieren, wenn ich den Tab oder den Browser-Tab wechsle.
- **Als EEG-Admin** möchte ich beim Pflegen der Formular-Felder nicht extra „Speichern" klicken müssen — jeder Toggle-Klick soll direkt persistiert werden.
- **Als EEG-Admin** möchte ich beim Schreiben des Einleitungstexts ein Sicherheitsnetz haben, falls der Browser abstürzt — aber den finalen „Jetzt ist's fertig"-Klick selbst kontrollieren.

## Acceptance Criteria

### AC-1 — Tab-Switch-Schutz (Option C)
- Beim Versuch, den Settings-Tab zu wechseln, **während der aktuelle Tab ungespeicherte Änderungen hat**, erscheint ein Confirm-Dialog:
  - Titel: „Ungespeicherte Änderungen"
  - Beschreibung: nennt den verlassenen Tab und warnt, dass Änderungen verworfen werden
  - Buttons: „Hier bleiben" (Default, schließt Dialog) / „Verwerfen und wechseln"
- Bei `Verwerfen` wechselt der Tab; die ungespeicherten Werte im verlassenen Editor werden zurückgesetzt auf den zuletzt gespeicherten Stand
- Bei `Hier bleiben` schließt der Dialog, Tab bleibt aktiv
- Wirkt für die drei Save-Button-Editoren: EEG-Settings, Einleitungstext, Formular-Felder
- Wirkt **nicht** für die aktions-basierten Editoren (Rechtsdokumente, API, Datenweiterleitung, Import/Export) — die haben kein Konzept von „ungespeichert"

### AC-2 — Browser-Unload-Schutz (Option C)
- `beforeunload`-Event wird abgefangen, wenn irgendein Editor im Settings-Bereich `dirty=true` ist
- Browser zeigt seinen nativen „Diese Seite verlassen?"-Dialog
- Wirkt bei: Tab-Close, Refresh, externe Navigation

### AC-3 — EEG-Wechsel-Schutz (Option C)
- Wenn der Admin im EEG-Auswahl-Select eine andere EEG wählt **während ungespeicherte Änderungen vorliegen**, erscheint derselbe Confirm-Dialog
- Bei `Verwerfen` wechselt die EEG; bei `Hier bleiben` rollt das Select zurück

### AC-4 — Auto-Save für Formular-Felder (Option B Phase 1)
- Speichern-Button wird **entfernt**
- Jede Änderung (Toggle, Admin-Vorbefüllung-Input) triggert nach 500 ms Debounce einen API-Call
- Status-Indikator rechts oben in der Editor-Card zeigt den aktuellen Zustand:
  - `Speichert…` (während aktiv) — neutraler Text mit Spinner
  - `Gespeichert` (nach Erfolg, blendet nach 2s aus) — grünes Häkchen
  - `Fehler beim Speichern` (nach Fehler, bleibt sichtbar) — rotes X mit „Erneut versuchen"-Button
- Bei Fehler: Editor bleibt `dirty`, Tab-Switch-Schutz greift weiter

### AC-5 — Auto-Save-Backup für Einleitungstext (Option B Phase 2)
- Speichern-Button **bleibt** sichtbar — der Admin entscheidet bewusst, wann der Text „fertig" ist
- Zusätzlich: Tiptap-Inhalt wird nach 30 s Inaktivität automatisch gespeichert (`onUpdate`-Listener mit Debounce)
- Status-Text neben dem Save-Button: `Zuletzt automatisch gespeichert: HH:MM:SS` (nach erstem Auto-Save)
- Auto-Save führt zum gleichen API-Call wie der manuelle Save — Backend muss nicht verändert werden
- Bei Fehler: Auto-Save-Indikator zeigt Warnung („Auto-Speichern fehlgeschlagen — bitte manuell speichern")

### AC-6 — EEG-Settings unverändert
- Save-Button bleibt — die ~30 Felder formen eine semantische Einheit mit gegenseitigen Validierungen
- Editor meldet aber `dirty`-State an die Parent-Page, sodass AC-1/AC-2/AC-3 greifen

### AC-7 — Doku
- `docs/user-guide/06-admin-settings.md`: erklärt das neue UX-Muster (Auto-Save bei Formular-Felder, Confirm-Dialog bei Tab-Wechsel)
- `docs/user-guide/changelog.md`: Eintrag

## Non-Goals

- **Vollständiger Auto-Save überall (Option A).** Nicht für EEG-Settings, weil die Felder gegenseitig validieren und ein bewusster Sammel-Save Audit-Logging + Roll-Back-Szenarien sauber hält.
- **Versionierte Settings / Undo-History.** Nicht in V1.
- **Auto-Save-Konflikt-Auflösung** (zwei Admins editieren parallel). Last-write-wins wie heute.
- **Optimistic UI** mit lokaler Drift bei Server-Fehler. Bei Fehler bleibt der Editor `dirty` und der Admin retry'ed.

## Tech-Design-Skizze (vorläufig — nach `/architecture` konkretisieren)

### Komponenten

- `src/hooks/use-dirty-tracker.ts` — Custom Hook für `dirty` + `onDirtyChange`-Bridge
- `src/hooks/use-unsaved-changes-warning.ts` — `beforeunload`-Hook
- `src/hooks/use-debounced-auto-save.ts` — Debounce-Hook für AC-4 + AC-5
- `src/components/settings/unsaved-changes-dialog.tsx` — AlertDialog-Wrapper

### Änderungen

- `admin-field-config-editor.tsx`: Save-Button entfernen, `useDebouncedAutoSave` einbauen, Status-Indikator
- `admin-intro-text-editor.tsx`: Save-Button bleibt, `useDebouncedAutoSave(30000)` als Backup
- `admin-eeg-settings-editor.tsx`: `dirty`-State explizit, `onDirtyChange`-Prop, Save-Button bleibt
- `src/app/admin/settings/page.tsx`:
  - `Tabs` von uncontrolled zu controlled (`value` + `onValueChange`)
  - Dirty-State-Map `Record<string, boolean>` für die drei Editoren
  - Tab-Switch-Interception
  - EEG-Wechsel-Interception
  - `useUnsavedChangesWarning` einbinden

### Backend

Keine Änderungen. Bestehende Endpoints (`PUT /api/admin/eegs/{rc}/field-config`, `PUT /api/admin/eegs/{rc}/intro-text`, `PUT /api/admin/eegs/{rc}/settings`) bleiben unverändert. Auto-Save trifft sie nur häufiger.

**Rate-Limit-Check:** Field-Config-Editor sendet bei 500ms-Debounce maximal 2 Requests/Sekunde während aktiver Bedienung. Bei 30 Toggles in 30 Sekunden = max 30 Requests. Aktuelle Admin-Rate-Limits liegen weit darüber — kein Problem.

### Tests

- E2E: Tab-Switch mit ungespeicherten Änderungen zeigt Dialog, „Verwerfen" rollt zurück, „Hier bleiben" hält
- E2E: Field-Config-Editor — Toggle umlegen → 500ms warten → Refresh → Wert ist persistiert
- E2E: Intro-Text — Text tippen → 30s warten → Refresh → Inhalt ist persistiert
- Unit: `useDebouncedAutoSave` debounced korrekt, cancel'd bei unmount

## Offene Punkte

1. Soll der Tab-Switch-Confirm-Dialog auch greifen, wenn der Admin in der **Anwendungsliste** rumklickt (Sidebar-Navigation)? Aktueller Vorschlag: ja, über das `beforeunload`-Pattern oder Next-Router-Interception. Aber komplexer als Tab-Switch innerhalb der Settings-Seite. → V1: nur Tab-Switch + EEG-Wechsel + beforeunload. Sidebar-Navigation = V2.
2. Was passiert bei Auto-Save-Fehler im Field-Config-Editor? V1-Vorschlag: dirty bleibt true, Status-Indikator zeigt rot, Admin kann manuell retry'en über einen erscheinenden „Erneut speichern"-Button. Alternative: Toggle springt visuell zurück. → Tendenz zu Variante 1 (transparenter).
3. Wie lang soll der „Gespeichert"-Indikator stehen bleiben? V1: 2 Sekunden, dann ausblenden. Konsistenz mit anderen Auto-Save-UIs (Notion: ~1.5s, Linear: ~2s).

## Dependencies

- shadcn `AlertDialog` (bereits in Verwendung)
- Keine Backend-Dependencies

## Risiken

- **Tab-Switch-Race:** wenn Auto-Save mid-flight ist und der Admin den Tab wechselt, könnte die `dirty=false`-Markierung nach dem Switch eintreffen. → Auto-Save promise muss in-flight tracked werden; Tab-Switch wartet kurz oder geht trotzdem (Auto-Save läuft im Hintergrund weiter).
- **Tiptap onUpdate-Spam:** der Tiptap-Editor feuert `onUpdate` bei jedem Tastendruck. 30s-Debounce dämpft das, aber bei Auto-Save-Fehler sollte der Editor nicht jeden Tastendruck retry'en.
- **EEG-Wechsel mit `applyEpoch`-Remount:** der bestehende `key={...applyEpoch}`-Mechanismus remountet die Editoren komplett. Dirty-State muss bei Remount bei `false` starten.
