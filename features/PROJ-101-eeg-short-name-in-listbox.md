# PROJ-101: EEG-Kurzform in den Auswahllisten

## Status: Architected (eingefroren nach Grilling 2026-06-11)
**Created:** 2026-06-11
**Last Updated:** 2026-06-11
**Typ:** UX-Verbesserung (Tester-Wunsch) + Schema-Erweiterung (PROJ-32-Folge)

## Hintergrund

Tester-Rückmeldung 2026-06-11:

> „in den Einstellungen wird in der Listbox zur Auswahl der EEG aktuell
> nur die RC-Nummer angezeigt. Es wäre verständlicher, wenn dort die
> Kurzform des EEG-Namens stünde."

Im eegFaktura-Core gibt es zwei Bezeichnungen pro EEG:

- **Langform** (`description`, z. B. „Testenergiegemeinschaft EEG 1234")
  — wird seit PROJ-32 in `registration_entrypoint.eeg_name` synchronisiert.
- **Kurzform** (`name`, z. B. „EEG-Test") — eine **eigenständige**
  kürzere Bezeichnung, die der EEG-Verwalter im Core pflegt. Bisheriger
  Code-Kommentar im Onboarding (`internal/coreclient/eeg_master_data.go`)
  ging fälschlich davon aus, sie sei mit `rcNumber` identisch — das ist
  nicht der Fall.

Die Kurzform ist kürzer und für den Admin im Onboarding leichter
wiederzuerkennen als die opake RC-Nummer (z. B. `RC0001`).

**Historie:** In der frühen Onboarding-Version (vor 2026-05-25) wurde
`name` (Kurzform) als „EEG-Name" verwendet. Mit der 2026-05-25-Iteration
ist das Mapping auf `description` (Langform) gewechselt, weil die
Langform im AVV-PDF + SEPA-PDF besser lesbar ist. PROJ-101 holt die
Kurzform nicht als „EEG-Name"-Ersatz zurück, sondern als zusätzliche
Spalte `eeg_short_name` — Langform bleibt für PDF/Mail, Kurzform
kommt für die Admin-UI-Auswahllisten dazu. Memory
`project_eegfaktura_core_api` ist entsprechend korrigiert.

**Owner-Direktive 2026-06-11 (Grilling):** die Kurzform soll als
zentraler Wert verfügbar sein, sodass in einer späteren PROJ
„diverse Berichten und Listen" (Excel-Export, Antrags-Listen,
ggf. Mails) potenziell von RC-Nummer auf Kurzform umgestellt werden
können. PROJ-101 beschränkt sich aber auf die drei Admin-UI-
Auswahllisten — der Bericht-/Listen-Wechsel ist Out-of-Scope und
kommt als Folge-PROJ, sobald der Owner ihn priorisiert.

## Scope

**Festgenagelt im Grilling 2026-06-11:**

1. **Schema-Erweiterung:** neue Spalte `eeg_short_name TEXT NULL` auf
   `member_onboarding.registration_entrypoint`. Migration `000075_*`.
2. **Core-Sync (PROJ-32-Erweiterung):** Feld `name` aus dem GraphQL-
   Response `data.eeg.name` ins DTO `EEGMasterData` aufnehmen. Beim
   Sync wird `name` mit `strings.TrimSpace` normalisiert; leerer
   String oder whitespace-only wird als `NULL` gespeichert. Der
   irreführende Code-Kommentar in `eeg_master_data.go` wird korrigiert.
3. **Neuer Backend-Endpoint** `GET /api/admin/registration-entrypoints`:
   liefert pro Tenant ein Array `[{rcNumber, eegShortName, eegName}]`.
   Tenant-Admin sieht nur RCs aus `session.tenant`, Superuser sieht
   alle. Liest aus `member_onboarding.registration_entrypoint`.
   Keine PII (kein IBAN, kein CreditorID).
4. **Frontend:** zentraler Fetch + React-Context, der den Endpoint
   beim AdminLayout-Mount einmal lädt. Drei Listboxen konsumieren
   den Context:
   - **Settings-EEG-Switcher** (`src/app/admin/settings/page.tsx:314-323`)
   - **Antrags-Filter-Panel** (`src/components/admin-filter-panel.tsx:135-145`)
   - **Reassign-Dialog Ziel-EEG** (`src/components/admin-status-actions.tsx:639-648`)
5. **Display-Format:** `Kurzform • RC-Nummer` einzeilig. Beispiel:
   `EEG-Test • RC0001`. Bei NULL-Kurzform nur die RC-Nummer (kein
   Bullet, kein Hinweis-Text). Helper `formatEegLabel(rcNumber, shortName)`.
6. **Sortierung:** alphabetisch nach Kurzform (case-insensitive,
   `localeCompare("de")`); EEGs ohne Kurzform landen ans Ende, dort
   alphabetisch nach RC-Nummer. Im Filter-Panel bleibt der
   `<SelectItem value="all">Alle EEGs</SelectItem>` an erster Stelle
   (vor der sortierten EEG-Liste).
7. **Sync-Strategie:** rein manuell über den bestehenden PROJ-32-
   Sync-Knopf in den Settings. Kein Auto-Sync, kein Hintergrund-Sync.
8. **`last-used-rc` bleibt RC-basiert** (`src/lib/last-used-rc.ts`):
   der gespeicherte Wert ist weiter die RC-Nummer. Kurzform ist nur
   eine Display-Schicht, kein Lookup-Key.

## Acceptance Criteria

### Backend

- [ ] **AC-1** Migration `000075_registration_entrypoint_short_name.up.sql`
  fügt `eeg_short_name TEXT NULL` hinzu; `.down.sql` entfernt sie.
- [ ] **AC-2** `shared.RegistrationEntrypoint` hat neues Feld
  `EEGShortName *string \`json:"eegShortName,omitempty" db:"eeg_short_name"\``.
- [ ] **AC-3** `coreclient.EEGMasterData` parst `name` aus der GraphQL-
  Response in ein neues Feld `Name *string`. Code-Kommentar (Zeile
  25–29) korrigiert: `name` ist eine eigenständige Kurzform, nicht
  identisch mit `rcNumber`.
- [ ] **AC-4** PROJ-32-Sync-Service schreibt `EEGMasterData.Name` in
  `registration_entrypoint.eeg_short_name`. `strings.TrimSpace`
  + Empty-Check → NULL. Verhalten konsistent zur 2026-06-06-
  PROJ-32-Erweiterung (Legal/VatNumber/ContactPerson/ContactPhone).
- [ ] **AC-5** Repo-Methode `ListEntrypointsForTenant(rcNumbers []string)`
  liefert Slice `[]EntrypointSummary{rcNumber, eegShortName, eegName}`.
  Filter per WHERE-IN, Sortierung im SQL (`ORDER BY eeg_short_name
  NULLS LAST, rc_number`).
- [ ] **AC-6** Handler `GET /api/admin/registration-entrypoints`:
  - Keycloak-JWT-Pflicht (bestehende Middleware)
  - Tenant-Admin: filter auf `session.tenant`-RCs
  - Superuser: liefert alle RCs aus `registration_entrypoint`
  - Response-Schema: `{entrypoints: [{rcNumber, eegShortName?, eegName?}]}`
  - Keine PII (kein IBAN, kein CreditorID, keine Adress-Felder)

### Frontend

- [ ] **AC-7** Neuer Helper `formatEegLabel(rcNumber: string, shortName?:
  string): string` in `src/lib/eeg-label.ts`: liefert
  `${shortName} • ${rcNumber}` wenn `shortName` truthy und nicht-
  whitespace, sonst nur `rcNumber`. Plus Unit-Tests für die vier
  Cases (shortName gesetzt / leer / whitespace / undefined).
- [ ] **AC-8** Neuer `EegDirectoryProvider`-React-Context in
  `src/components/eeg-directory-context.tsx`: fetcht beim Mount
  `GET /api/admin/registration-entrypoints`, cached den Response,
  exposed Hook `useEegDirectory()` → `{rcNumber, eegShortName,
  eegName}[]` plus `formatLabel(rcNumber)`-Convenience.
- [ ] **AC-9** AdminLayout mountet den `EegDirectoryProvider`
  einmalig oberhalb der Routes.
- [ ] **AC-10** **Settings-EEG-Switcher**
  (`src/app/admin/settings/page.tsx:314-323`) zeigt
  `formatEegLabel(rc, shortName)` als SelectItem-Children;
  Sortierung gemäß useEegDirectory-Reihenfolge.
- [ ] **AC-11** **Antrags-Filter-Panel**
  (`src/components/admin-filter-panel.tsx:135-145`) analog;
  `<SelectItem value="all">Alle EEGs</SelectItem>` bleibt als
  erster Eintrag vor den sortierten EEGs.
- [ ] **AC-12** **Reassign-Dialog Ziel-EEG**
  (`src/components/admin-status-actions.tsx:639-648`) analog;
  `availableTargetRcs` wird mit `useEegDirectory` gemappt.
- [ ] **AC-13** Bei NULL-Kurzform für alle EEGs (z. B. vor erstem
  Sync) fallen alle drei Listboxen auf reine RC-Darstellung zurück
  — kein Bruch.

### Tests + Doku

- [ ] **AC-14** Backend-Tests: Sync-Roundtrip-Test (`name` aus Core
  → DB-Spalte), Empty-String-Test (Core liefert "" → NULL), Handler-
  Test (Tenant-Filter + Superuser-Bypass + leere Liste).
- [ ] **AC-15** Frontend-Tests: `formatEegLabel`-Helper-Tests,
  Sortier-Test (Mix mit NULL), Render-Tests für die drei Listboxen
  mit Mock-Directory.
- [ ] **AC-16** `go build ./...` clean, `go test ./...` clean,
  `npm run build` clean, `npx tsc --noEmit` clean, `npx vitest run`
  clean.
- [ ] **AC-17** Doku:
  - `docs/domain-model.md` neue Spalte dokumentieren
  - `docs/api-spec.md` neuer Endpoint
  - `docs/user-guide/06-admin-settings.md`: Hinweis dass nach dem
    Sync die Kurzform in den EEG-Auswahllisten verfügbar wird
    (PROJ-frei, anonymisiertes Beispiel)
  - `docs/user-guide/changelog.md`-Eintrag (PROJ-frei)
  - `CHANGELOG.md` im Deploy-Commit

## Edge Cases

- **EC-1** Bestand-EEG hat noch nie gesynct → `eeg_short_name` NULL
  → Listbox zeigt nur RC-Nummer (AC-13).
- **EC-2** Core liefert `name = ""` oder whitespace-only → behandeln
  als NULL → Fallback auf RC-Nummer (AC-4).
- **EC-3** Zwei EEGs im Tenant haben dieselbe Kurzform (Tester pflegt
  schlampig) → Display-Format `Kurzform • RC-Nummer` disambiguiert
  visuell. Sortierung legt sie nebeneinander, dann nach RC-Sekundär-
  Schlüssel.
- **EC-4** Admin ändert Kurzform im Core nach Onboarding-Setup →
  manueller Sync via PROJ-32-Knopf in Einstellungen aktualisiert sie.
  Bis dahin zeigt Onboarding die alte Kurzform.
- **EC-5** EegDirectory-Endpoint kommt mit 401/403/500 zurück →
  `useEegDirectory` rendert die Listboxen ohne Kurzform (Fallback auf
  reine RC-Darstellung wie AC-13). Inline-Fehler-Toast einmalig.
- **EC-6** Superuser ohne `session.tenant` → Endpoint liefert die
  vollständige Liste aller RCs aus `registration_entrypoint`.

## Out of Scope

- **Wechsel von RC-Nummer auf Kurzform in Berichten und Listen**
  (Antrags-Liste, Excel-Export, Mail-Templates): Owner-Direktive
  „denkbar" — eigene Folge-PROJ.
- Editierbares Kurzform-Feld im Onboarding-Admin-UI (Kurzform bleibt
  Read-Only-Mirror aus dem Core, analog `eeg_name`).
- Kurzform im PDF (AVV, SEPA, Beitrittsbestätigung): heute steht dort
  die Langform — kein Tester-Wunsch zur Änderung.
- Kurzform im Excel-Export Datenweiterleitung: separate Spec falls
  Tester das nachfragt.
- Auto-Sync / Hintergrund-Sync der Stammdaten.
- Hinweis-Banner „Kurzform noch nicht synchronisiert" — Owner hat
  explizit nur den manuellen Sync gewählt, ohne Banner-Nudge.

## Tech Design (Grobskizze für /architecture)

### Backend

```
db/migrations/000075_registration_entrypoint_short_name.up.sql
ALTER TABLE member_onboarding.registration_entrypoint
    ADD COLUMN eeg_short_name TEXT NULL;
```

- `internal/coreclient/eeg_master_data.go`:
  - `EEGMasterData.Name *string` neues Feld
  - Code-Kommentar 25–29 korrigieren

- `internal/application/registration_entrypoint_repo.go`:
  - `ListEntrypointsForTenant(ctx, rcNumbers []string) ([]EntrypointSummary, error)`
  - Plus Superuser-Variante ohne Filter

- `internal/application/registration_entrypoint_repo_tx.go`:
  - Sync-UPDATE um `eeg_short_name` erweitern, `strings.TrimSpace`
    + Empty-Check

- `internal/http/admin.go`:
  - `ListRegistrationEntrypoints` Handler
  - Response-DTO `RegistrationEntrypointSummaryResponse`

- `cmd/server/main.go`:
  - `r.Get("/registration-entrypoints", adminHandler.ListRegistrationEntrypoints)`

### Frontend

- `src/lib/api.ts`: `listRegistrationEntrypoints(token)` →
  `Promise<EegDirectoryEntry[]>`
- `src/lib/eeg-label.ts`: `formatEegLabel` Helper + Tests
- `src/components/eeg-directory-context.tsx`: Provider + Hook
- `src/app/admin/layout.tsx`: Mount Provider
- 3 SelectItem-Renderings ändern:
  - `settings/page.tsx`, `admin-filter-panel.tsx`,
    `admin-status-actions.tsx`

### Dependencies

- PROJ-32 (EEG-Stammdaten aus Core) — die Sync-Infrastruktur existiert.
- Kein neuer Core-API-Endpoint nötig — `name` ist bereits im
  GraphQL-Response enthalten, wir parsen ihn heute nur nicht.

## Risiken

- Falls der Core `name` für viele Bestand-EEGs leer hält, fällt der
  UX-Gewinn klein aus, bis EEG-Verwalter sie pflegen. Owner-
  Hinweis an die EEGs erwägen (Out-of-Scope für die Spec).
- Cache-Drift wenn der Admin die Kurzform im Core ändert: Onboarding
  zeigt veralteten Wert bis zum nächsten manuellen Sync — by design,
  konsistent mit allen anderen PROJ-32-Feldern.
- AdminLayout-Mount-Fetch ist ein zusätzlicher Backend-Call pro
  Session-Start. Bei 20 Tenant-EEGs ist die Payload klein (<5 KB),
  Last-Profil unkritisch.

## Owner-Entscheidungen (Grilling 2026-06-11)

1. **Geltungsbereich** → alle drei EEG-Listboxen (Settings + Filter +
   Reassign).
2. **Display-Format** → `Kurzform • RC-Nummer`.
3. **Lookup-API** → neuer Endpoint
   `GET /api/admin/registration-entrypoints`.
4. **Endpoint-Schnitt** → tenant-gefiltert, drei Felder
   `{rcNumber, eegShortName, eegName}`, kein PII.
5. **Sortierung** → alphabetisch nach Kurzform
   (`localeCompare("de")`), NULL ans Ende.
6. **NULL-Fallback** → nur RC-Nummer anzeigen, kein extra Hinweis.
7. **Empty-String-Mapping** → wie NULL behandeln
   (`strings.TrimSpace` + Empty-Check beim Sync).
8. **Sync-Strategie** → rein manuell über den bestehenden PROJ-32-
   Sync-Knopf.
9. **last-used-rc** → bleibt RC-basiert; `Alle EEGs`-Eintrag bleibt
   erster Eintrag im Filter-Panel.
10. **Berichte/Listen-Wechsel** → Out-of-Scope; eigene Folge-PROJ.
