# PROJ-83: Letzte EEG-Auswahl im Admin-Settings-UI persistieren

## Status: Approved
**Created:** 2026-06-08
**Last Updated:** 2026-06-08
**Typ:** UX-Polish

## Hintergrund

Owner-Feedback 2026-06-08:

> „Kann sich das System irgendwie merken, für welche EEG die Einstellungen
> zuletzt geöffnet waren? Er wird immer wieder auf die erste zurückgesprungen
> und ich möchte aber die Einstellungen der EEG sehen, die ich zuletzt offen
> hatte."

Heutiges Verhalten: bei jedem Aufruf von `/admin/settings` initialisiert
`page.tsx:217-221` `selectedRc` auf `rcNumbers[0]`. Bei 10+ EEGs ist das
jedes Mal ein Mehrklick im Listbox-Auswahlmenü.

## Owner-Direktive 2026-06-08

> „B" — Persistenz via `localStorage` (gegen die Alternativen URL-Query
> bzw. Backend-Preference-API).

Begründung: minimaler Aufwand, deckt den Pain-Point „auf einem Gerät
arbeiten" (95 % der Fälle). Geräte-Sync und Audit-Trail sind nicht
gefordert.

## Scope

### Betroffen
- `src/lib/last-used-rc.ts` (NEU) — Persistenz-Helper
- `src/lib/last-used-rc.test.ts` (NEU) — Verhaltens- und Sicherheitstests
- `src/app/admin/settings/page.tsx` — Initial-Load aus Storage, Write bei
  jedem `selectedRc`-Wechsel

### Nicht betroffen
- Backend (kein neuer Endpoint, keine DB-Spalte)
- Andere `/admin/*`-Pfade (Antragsliste, Cockpit-Vorgriff PROJ-72, etc.)
  — falls dort später dasselbe Pattern erwünscht ist, kann der Helper
  zentral wiederverwendet werden

## Acceptance Criteria

- [x] **AC-1** Persistenz-Helper `src/lib/last-used-rc.ts` mit drei
  Funktionen: `readLastUsedRc(allowedRcNumbers)`, `writeLastUsedRc(rc)`,
  `clearLastUsedRc()`
- [x] **AC-2** `readLastUsedRc` liefert nur dann den persistierten Wert,
  wenn er in den aktuell erlaubten RC-Nummern (`rcNumbers` aus dem JWT-
  Claim) enthalten ist
- [x] **AC-3** Wenn der persistierte Wert nicht mehr im Tenant-Scope ist,
  wird er aus dem Storage gelöscht (kein Stale-Hängenbleiben)
- [x] **AC-4** Bei `localStorage`-Fehler (Privat-Modus, Quota) wird der
  Fehler still geschluckt — der UI-Pfad fällt auf das heutige Default-
  Verhalten zurück (erste RC)
- [x] **AC-5** Der Storage-Key ist namespaced (`eegfaktura-onboarding:
  settings:lastRc`), damit kein Konflikt mit anderen Anwendungen oder
  zukünftigen Settings-Persistenzen entsteht
- [x] **AC-6** Im Storage wird nur der RC-String persistiert — kein Token,
  kein User-ID, keine PII. Verifiziert per Vitest-Test der Sicherheits-
  Eigenschaft
- [x] **AC-7** `page.tsx` ruft beim Initial-Load von `rcNumbers` zuerst
  `readLastUsedRc(rcNumbers)` auf und nutzt das Ergebnis; nur bei `null`
  fällt es auf `rcNumbers[0]` zurück
- [x] **AC-8** Jeder `setSelectedRc`-Aufruf (Initial, Listbox-Wechsel,
  EEG-Switch nach Dirty-Confirm) speichert via `writeLastUsedRc`
- [x] **AC-9** Tests + Build grün (Vitest 65/65, tsc clean, Next-Build
  clean)
- [x] **AC-10** User-Guide-Changelog + CHANGELOG.md aktualisiert (PROJ-frei
  im User-Guide laut Memory-Regel `feedback_no_proj_refs_in_user_doc`)

## Edge Cases

- **EC-1 Admin hat nur eine EEG:** `rcNumbers.length === 1`. Helper liest
  den Wert, sieht er ist gleich `rcNumbers[0]`, gibt ihn zurück. Verhalten
  identisch zum heutigen Default.
- **EC-2 Admin hat noch nie eine EEG geöffnet (frischer Browser):** Storage
  leer, Helper liefert `null`, Fallback auf `rcNumbers[0]`. Initial-Wert
  wird beim Initial-Set sofort persistiert (siehe AC-8) — nächster Aufruf
  hat schon Persistenz.
- **EC-3 Admin verliert eine EEG-Berechtigung:** Persistierte RC ist nicht
  mehr in `rcNumbers`. Helper verwirft den Wert + löscht ihn aus dem
  Storage; UI fällt auf `rcNumbers[0]`. Beim nächsten Wechsel auf eine
  andere EEG wird der neue Wert sauber persistiert. Keine 403-Schleife
  oder „komische Auswahl" möglich.
- **EC-4 localStorage in Privat-Modus / Quota erschöpft:** Try-Catch im
  Helper, Fehler still geschluckt. UI verhält sich exakt wie heute.
- **EC-5 Zwei Browser-Tabs auf `/admin/settings` parallel:** Tab A wählt
  RC123, Tab B wählt RC456. Beide schreiben in dasselbe localStorage —
  letzter Schreiber gewinnt. Wenn der Admin später neu lädt, sieht er die
  Auswahl des zuletzt aktiven Tabs. Akzeptiert: Tab-Sync ist nicht
  Persistenz-Ziel.
- **EC-6 Manipulierter Storage-Wert (z. B. via DevTools):** Helper prüft
  gegen `rcNumbers` — wenn der Wert nicht in der Whitelist ist, wird er
  verworfen. Kein Vertrauensanker auf den Storage-Wert; der eigentliche
  Tenant-Scope-Check passiert weiterhin im Backend.
- **EC-7 RC-Liste lädt verzögert:** `useEffect` ist gegated auf
  `rcNumbers.length > 0 && !selectedRc`. Solange `rcNumbers` leer ist,
  passiert nichts. Sobald die Liste eintrifft, läuft der Helper-Lookup.
  Kein Race möglich.
- **EC-8 PROJ-61 Bundle-Import erhöht `applyEpoch`:** orthogonal zur
  Persistenz, weil der `selectedRc`-State von `applyEpoch` nicht angetastet
  wird.

## Tech Design

```
Vorher:
   useEffect([rcNumbers, selectedRc])
   if (rcNumbers.length > 0 && !selectedRc)
     setSelectedRc(rcNumbers[0])

Nachher (PROJ-83):
   useEffect([rcNumbers, selectedRc])
   if (rcNumbers.length > 0 && !selectedRc)
     const lastUsed = readLastUsedRc(rcNumbers)
     setSelectedRc(lastUsed ?? rcNumbers[0])

   useEffect([selectedRc])  // NEU
   if (selectedRc) writeLastUsedRc(selectedRc)
```

Persistierungs-Effekt ist bewusst ein eigener `useEffect` statt inline an
jeder `setSelectedRc`-Call-Stelle:
- Initial-Default (`rcNumbers[0]`) wird auch automatisch persistiert
- Kein vergessenes `setSelectedRc` ohne Persistenz möglich (5 Call-Sites)
- Single Source of Truth für die Write-Logik

## Sicherheits-Bewertung

- **Auth/AuthZ:** Persistierter Wert ist nur Hint, kein Vertrauensanker.
  Tenant-Scope wird beim API-Aufruf vom Backend geprüft (etabliertes
  `checkTenantAccess`-Pattern).
- **Storage-Inhalt:** Nur der RC-String. Kein JWT, kein Session-Token,
  keine User-ID, keine PII. Verifiziert per Test.
- **XSS:** Storage-Wert wird nirgendwo als HTML/JS gerendert — er fließt
  ausschließlich in `setSelectedRc(string)` und von dort als Query-Param
  in API-Aufrufe (URL-encoded). Selbst bei manipuliertem Storage-Wert
  rejected das Backend RCs außerhalb des JWT-Claims.
- **Privacy:** localStorage ist gerätelokal. Owner-akzeptierter Trade-off
  (siehe Option-B-Direktive).

## Geänderte Dateien

| Datei | Status |
|---|---|
| `src/lib/last-used-rc.ts` | **NEW** — drei Helper |
| `src/lib/last-used-rc.test.ts` | **NEW** — 9 Tests inkl. Sicherheits-Anker |
| `src/app/admin/settings/page.tsx` | Modified — Import + zwei `useEffect` |
| `features/PROJ-83-last-used-eeg-persistence.md` | **NEW** — diese Spec |
| `features/INDEX.md` | Modified — Eintrag + Next-ID |
| `docs/user-guide/changelog.md` | Modified — PROJ-frei |
| `CHANGELOG.md` | Modified — Release-Notes-Eintrag |

## QA Test Results

**Datum:** 2026-06-08
**Reviewer:** QA Engineer (AI, Solo-Code-Review)
**Status:** Approved

### Test-Status

```
$ npx tsc --noEmit           →  clean
$ npx vitest run             →  4 files, 65/65 tests passing
$ NEXT_PUBLIC_TEST_AUTH_MODE= npm run build  →  Next-Production-Build clean
```

### Security-Smoke

- 0 Backend-Änderungen
- 0 Auth-Pfade berührt
- 0 neue API-Endpoints
- LocalStorage-Wert ist nicht-vertraulich (RC-String) und wird gegen
  Tenant-Scope validiert
- Browser-Storage-Inhalt durch Sicherheitstest verifiziert (nur 1 Key,
  nur RC-String)

→ **0 Findings.** Nicht-Pflicht-Trigger für `/security-review` laut
CLAUDE.md.

### Regression

- `setSelectedRc`-Aufrufer geprüft: 5 Call-Sites, jede führt durch den
  neuen `useEffect`-Persistenz-Pfad
- Bestehende Logik unverändert: Listbox-Wechsel mit Dirty-Confirm, EEG-
  Wechsel mit `setPendingAction`-Pfad
- `applyEpoch`-Mechanik unverändert

**Production-Ready: READY.**

## Deployment

**Datum:** 2026-06-08
**Versions-Tag:** v1.23.2-PROJ-83 (Patch-Bump, reiner UX-Polish ohne
Verhaltens-Erweiterung der Settings-Persistenz selbst)
**Image-SHA:** wird vom CI nach Push gesetzt

Owner führt `helm upgrade` manuell aus.

---
<!-- Sections below are added by subsequent skills -->
