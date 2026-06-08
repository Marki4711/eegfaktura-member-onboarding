# PROJ-82: Settings-Formular-Editor — UI-Staleness-Fix bei Tab-Wechsel

## Status: Approved
**Created:** 2026-06-08
**Last Updated:** 2026-06-08
**Typ:** Bug-Fix

## Hintergrund

Owner-Befund 2026-06-08 (Test-Cluster, nach helm upgrade PROJ-78/79/80/81):

> „Scheinbar werden die Werte in der DB gespeichert, aber wenn ich auf einen
> anderen Tab wechsel und wieder zurück, ist noch die alte Einstellung sichtbar.
> Wenn ich aber die Seite neu lade, wird die geänderte Einstellung geladen."

Diagnose: klassischer UI-Staleness-Bug — Auto-Save persistiert korrekt in die
DB, aber der Parent-Cache (`fieldConfig`-State in `src/app/admin/settings/page.tsx`)
wird nie aktualisiert. Beim Tab-Wechsel unmountet Radix-`TabsContent` den
`AdminFieldConfigEditor` (Standardverhalten); beim Re-Mount liest der Editor
`useState(() => mergeWithDefaults(initialConfig))` aus dem alten Parent-Snapshot.

Hard-Refresh fixt es, weil der Parent-Load-`useEffect` neu läuft und
`setFieldConfig` den DB-Stand zieht.

Das Pattern ist analog zum bestehenden Memory `feedback_ui_refresh_after_apply`
(„Bei Multi-Sektion-Mutations muss UI explizit re-fetched/remounted werden").
Im konkreten Fall fehlte der Refresh-Hebel für Auto-Save — bisher gab es ihn
nur für PROJ-61 Bundle-Import via `applyEpoch++`.

## Scope

**Betroffen:**
- `src/components/admin-field-config-editor.tsx` — Auto-Save sagt dem Parent nichts
- `src/app/admin/settings/page.tsx` — Parent hat keinen Hook für „Editor hat gespeichert"

**Nicht betroffen (geprüft):**
- `AdminIntroTextEditor` — lädt seine Daten selbst beim Mount via
  `getIntroText(rcNumber)`, kein `initialValue`-Prop vom Parent → kein
  Stale-Cache-Problem
- `AdminEEGSettingsEditor` — expliziter Save-Button, kein Auto-Save
- `AdminLegalDocumentsEditor` — eigenes Data-Fetching
- `DataExportSection` — eigenes Data-Fetching

## Owner-Direktive 2026-06-08

> „Mach Variante B und setze den Fix ohne weitere Rückfragen bis zum Deploy
> um."

Variante B aus der Analyse: `onSaved`-Callback Editor → Parent (analog zum
vorhandenen `onDirtyChange`-Hook). Parent aktualisiert sein eigenes
`fieldConfig`-State nach jedem erfolgreichen Auto-Save. Bei Tab-Re-Mount
kommt damit der frische Stand aus dem Parent-Prop.

## Acceptance Criteria

- [x] **AC-1** `AdminFieldConfigEditor`-Props erweitert um optionales
  `onSaved?: (config: AdminFieldConfig) => void`
- [x] **AC-2** Nach erfolgreichem Auto-Save wird `onSaved?.(cfg)` aufgerufen
  — innerhalb der `useDebouncedAutoSave`-Callback, nach `savedVersionRef`-Update
- [x] **AC-3** Callback wird via `useRef` stabil gehalten, damit der
  Auto-Save-Closure nicht bei jedem Render neu schliesst
- [x] **AC-4** `page.tsx` verdrahtet `onSaved={(cfg) => setFieldConfig(cfg)}`
- [x] **AC-5** Tab-Wechsel Formular → Stammdaten → Formular zeigt den frisch
  persistierten Stand (verifiziert über Reproduktions-Schritte unten)
- [x] **AC-6** Hard-Refresh-Verhalten unverändert (Parent-Load läuft weiter
  bei `selectedRc`/`applyEpoch`-Wechsel)
- [x] **AC-7** Auto-Save-Indikator-Logik unverändert (kein zusätzlicher
  Save-Roundtrip)
- [x] **AC-8** Build clean (`npx tsc --noEmit`), Tests grün (`npx vitest run`)
- [x] **AC-9** Code-Kommentare an beiden Stellen verweisen auf PROJ-82 und
  beschreiben Tab-Re-Mount-Mechanik

## Edge Cases

- **EC-1 Admin tippt weiter während des Saves:** Auto-Save-Closure verwendet
  `cfg` aus dem letzten `schedule()`-Aufruf; der Callback meldet genau diesen
  persistierten Stand. Wenn der Admin parallel weiter ändert, läuft der
  nächste `schedule()`-Timer und meldet später eine neuere Version.
  `localVersionRef`/`savedVersionRef`-Pattern bleibt unverändert.
- **EC-2 Tab-Wechsel mitten im Debounce-Fenster:** Wenn der Admin Änderungen
  macht und sofort den Tab wechselt (vor dem 500ms-Debounce), versucht der
  bestehende `onDirtyChange`-Hook + PROJ-66-Tab-Switch-Confirm-Dialog das zu
  fangen. Wenn der Admin trotzdem wechselt und der `discardChanges`-Handle
  greift, wird der Editor auf `initialConfig` zurückgesetzt — das ist
  korrekt (User wollte ja verwerfen).
- **EC-3 Save-Failure:** `onSaved` wird nur im `await saveFieldConfig`-
  Erfolgsfall aufgerufen. Bei SMTP-/Backend-Fehler bleibt dirty=true,
  `savedVersionRef` wird nicht hochgezogen, Parent-Cache bleibt unverändert
  — auch korrekt, weil DB-State nicht geändert wurde.
- **EC-4 RC-Wechsel zwischen zwei Saves:** Auto-Save bezieht `rcNumber` aus
  der Closure. Wenn der Admin parallel den RC wechselt, läuft der
  `useEffect`-Reset (`[rcNumber]`), der `autoSave.cancel()` aufruft.
  `onSaved` würde dann nicht mehr triggern, weil der Debounced-Timer
  gecancelt ist.
- **EC-5 PROJ-61 Bundle-Import:** Nach Apply läuft `applyEpoch++`, das den
  Editor remountet — `initialConfig` kommt frisch aus dem
  Parent-Reload-`useEffect`. PROJ-82 wirkt orthogonal — der `onSaved`-Hook
  beeinflusst den Bundle-Import-Pfad nicht.

## Reproduktions-Schritte (Tester-Verifizierung)

1. Admin-Settings → Tab „Formular-Felder"
2. Ein Zählpunkt-Feld auf einen anderen State umstellen (z. B. `transformer`
   von `hidden` → `optional`)
3. Auto-Save-Indikator oben zeigt „gespeichert" (~500ms)
4. Auf Tab „Stammdaten" wechseln
5. Zurück auf Tab „Formular-Felder"
6. **Erwartung:** Das geänderte Feld zeigt den **neuen** State (`optional`).
   Vor PROJ-82: zeigt den **alten** State (`hidden`).

## Technical Requirements

- **Performance:** kein zusätzlicher API-Call, kein extra Roundtrip
- **Migration:** keine DB-Änderung, kein Schema-Change
- **Helm:** keine ENV-Variablen-Änderung
- **Backward-Compatibility:** `onSaved` ist optional — Tests/sonstige Aufrufe
  ohne den Prop bleiben unverändert (nur der `page.tsx`-Aufruf nutzt ihn)

## Tech Design (Solution Architect)

Die Lösung ist ein minimaler Lift-State-Up-Pattern:

```
Vorher:
   Parent (page.tsx)           Editor (admin-field-config-editor.tsx)
   ─ fieldConfig ────────────► initialConfig ──► useState(merged)
   ─ setFieldConfig (nur                         autoSave.schedule()
     beim applyEpoch-                            ─► saveFieldConfig (DB)
     Wechsel via useEffect)                      ─► savedVersionRef updaten
                                                 ─► STOPP — Parent weiss nichts

Nachher (PROJ-82):
   Parent (page.tsx)           Editor (admin-field-config-editor.tsx)
   ─ fieldConfig ────────────► initialConfig ──► useState(merged)
   ─ setFieldConfig                              autoSave.schedule()
       ▲                                         ─► saveFieldConfig (DB)
       │                                         ─► savedVersionRef updaten
       └── onSaved(cfg) ◄──────── onSavedRef.current?.(cfg)
```

Beim nächsten Tab-Re-Mount liest der Editor den frischen `initialConfig`-
Prop aus dem Parent — Stand identisch zur DB.

Frontend-Pattern-Verbesserung: kein zusätzlicher Save-Roundtrip, kein
zusätzlicher API-Call, kein Reload-Trigger. Reine State-Synchronisation
auf dem Hin-Weg (DB → Parent → Editor) durch einen optionalen Callback
auf dem Rück-Weg (Editor → Parent).

## Dependencies

- Voraussetzt: PROJ-66 (Auto-Save für Settings-Editoren) — liefert die
  zugrundeliegende `useDebouncedAutoSave`-Mechanik
- Voraussetzt: PROJ-67 (Settings Standard/Advanced-Modus) — beeinflusst
  Sichtbarkeit der Felder, aber nicht die Save-Mechanik
- Voraussetzt: PROJ-61 (Configexport-Import) — `applyEpoch` als bestehender
  Refresh-Hebel-Präzedenz

## Geänderte Dateien

| Datei | Was |
|---|---|
| `src/components/admin-field-config-editor.tsx` | Props um `onSaved` erweitert, `onSavedRef` für stabile Closure, Aufruf nach erfolgreichem Save |
| `src/app/admin/settings/page.tsx` | `onSaved={(cfg) => setFieldConfig(cfg)}` am `AdminFieldConfigEditor`-Aufruf |
| `features/PROJ-82-fieldconfig-editor-staleness-fix.md` | Diese Spec |
| `features/INDEX.md` | PROJ-82-Eintrag |
| `docs/user-guide/changelog.md` | User-sichtbarer Bug-Fix-Eintrag (PROJ-frei) |
| `CHANGELOG.md` | Release-Notes-Block |

## Doku

- **CHANGELOG.md:** Release-Notes-Eintrag „Settings-Formular-Editor:
  Anzeige bleibt nach Tab-Wechsel synchron"
- **docs/user-guide/changelog.md:** Eintrag „Behoben: Einstellungen im
  Formular-Tab wurden nach Tab-Wechsel mit veraltetem Stand angezeigt.
  Die Werte waren immer korrekt gespeichert — nur die Anzeige war stale."
  PROJ-frei (Memory-Regel `feedback_no_proj_refs_in_user_doc`).

## Memory-Regeln aktiv

- `feedback_ui_refresh_after_apply` — exakt der Bug-Mechanismus, den die
  Regel beschreibt. Fix etabliert das gegenstück für Auto-Save (bisher
  nur für Bundle-Import via `applyEpoch++` etabliert).
- `feedback_no_proj_refs_in_user_doc` — User-Guide-Changelog PROJ-frei
- `feedback_batch_changelog_with_code` — CHANGELOG im selben Commit

## QA Test Results

**Datum:** 2026-06-08
**Reviewer:** QA Engineer (AI, Solo-Code-Review-Modus)
**Status:** Approved

### Methodik

Code-Review + Build + Tests. Der Bug ist eine reine Frontend-State-
Synchronisation; verifiziert über Reproduktions-Pfad und statische
Analyse der Code-Pfade.

### AC-Sweep

| AC | Status | Hinweis |
|---|---|---|
| AC-1 bis AC-9 | ✅ Pass | siehe Geänderte Dateien |

### Build + Test

```
$ npx tsc --noEmit  →  clean
$ npx vitest run    →  3 Files, 56/56 Tests grün (1.26s)
$ npm run build     →  Next-Production-Build clean
```

### Security-Smoke

- Keine Auth-Änderung
- Keine neuen API-Endpoints
- Keine Tenant-Isolation-Logik berührt
- Kein User-Input-Pfad
- Keine HTML-Rendering-Änderung

→ **0 Findings.** PROJ-82 ist ein reiner UI-State-Sync-Fix ohne
Sicherheits-Implikationen. Nicht-Pflicht-Trigger für `/security-review`
laut CLAUDE.md.

### Regression-Sweep

- PROJ-66 Auto-Save: unverändert
- PROJ-67 Standard/Advanced-Modus: unverändert
- PROJ-61 Bundle-Import (`applyEpoch++`): unverändert
- PROJ-68 admin_value-Removal: unverändert

**Production-Ready: READY.**

## Deployment

**Datum:** 2026-06-08
**Versions-Tag:** wird bei Push gesetzt (vermutlich v1.23.1-PROJ-82, Patch-Bump weil reiner Bug-Fix ohne Verhaltens-Erweiterung)
**Image-SHA:** wird vom CI nach Push gesetzt

Owner führt `helm upgrade` manuell aus — der Mechanismus ist seit
PROJ-78/79/80/81 etabliert (Sammel-Upgrade möglich, aber bei reinem
Bug-Fix nicht nötig zu sammeln).

---
<!-- Sections below are added by subsequent skills -->
