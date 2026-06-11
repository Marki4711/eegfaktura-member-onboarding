# PROJ-103: Individuell anpassbare Brand-Farben (volles Theme pro EEG)

## Status: Deployed (2026-06-11)
**Created:** 2026-06-11
**Last Updated:** 2026-06-11
**Typ:** UX/Branding-Erweiterung (Tester-Wunsch nach PROJ-102-Deploy)

## Hintergrund

Tester-Rückmeldung 2026-06-11 — direkt nach dem Deploy von PROJ-102 (vier
vordefinierte Preset-Themes):

> „neuer feature request: die farben des erscheinungsbildes sollen
>  individuell anpassbar sein."

PROJ-102 hat heute Morgen vier Presets (`teal` / `leaf` / `sun` /
`slatey`) deployed. Tester sagen: das reicht nicht — sie wollen die
Brand-Farben ihrer EEG frei wählen.

Diskutiert wurden vier Lösungsrichtungen aus der ursprünglichen
PROJ-102-Discovery (Sektion „Mögliche Lösungsvarianten"):

- A.1/A.2/A.3 — Akzent-Farbe(n) on top vom Preset
- **B — volles Theme pro EEG** ← Owner-Entscheidung 2026-06-11
- D — eegFaktura-Core-Brand spiegeln
- E — Custom-CSS (verworfen, Security)

Owner hat sich nach Vergleich der vier Skalierungs-Stufen für **B** mit
**JSONB-Persistenz** (B.1) entschieden. Damit ist die Brand-
Anpassbarkeit vollumfänglich pro EEG — alle relevanten CSS-Variablen
sind editierbar, plus optional Font und Border-Radius.

## Architektur-Ausnahme von der „No-JSON"-Regel

CLAUDE.md / `.claude/rules/backend.md` verbieten JSON-Spalten im
Domain-Modell. Die Owner-Entscheidung für B.1 schafft eine bewusste
Einzel-Ausnahme:

- **Was:** `member_onboarding.registration_entrypoint.brand_theme JSONB
  NULL`
- **Warum erlaubt:** reine Präsentations-Konfiguration — Theme-Werte
  werden ausschließlich vom Public-Page-Rendering gelesen, nie für
  Joins, Filter, Reporting oder Audit verwendet. Bei jeder zusätzlichen
  Theme-Eigenschaft (Font, Schatten, Border-Radius, neue Farb-Tokens)
  wäre eine eigene Migration reine Bürokratie ohne Schutz-Nutzen.
- **Strikte Grenze:** die Ausnahme gilt ausschließlich für
  `brand_theme`. Alle anderen neuen Felder bleiben Spalten.
- **CLAUDE.md + backend.md sind angepasst** (Commit im Deploy-Commit).

## Scope

**Festgenagelt nach Owner-Direktive 2026-06-11:**

1. **Schema-Erweiterung:** `brand_theme JSONB NULL` auf
   `registration_entrypoint`. Migration 000077.
2. **Editierbare Felder im Theme-JSON** (vorläufig — wird in /grill-me
   verfeinert):
   - Farben: `primary`, `primaryFg`, `accent`, `accentFg`, `background`,
     `foreground`, `card`, `cardFg`, `border`, `ring` (HEX-Format
     `#RRGGBB`)
   - Optional: `fontFamily` (Whitelist gegen Google-Fonts-Subset oder
     System-Stack), `borderRadius` (numerisch, 0–1.5rem)
3. **Render-Schichtung** im Frontend (`brand-presets.ts` wird erweitert):
   - Ebene 1: `globals.css :root` (Default-Teal)
   - Ebene 2: Preset (`brand_preset` aus PROJ-102) überschreibt Ebene 1
   - Ebene 3: Custom-Theme (`brand_theme` aus PROJ-103) überschreibt
     Ebene 2 selektiv (nur gesetzte Felder)
4. **Admin-Editor** (`AdminBrandEditor` aus PROJ-102 wird erweitert):
   - Modus-Switch oben: „Preset wählen" (PROJ-102-Verhalten) /
     „Eigene Farben festlegen" (PROJ-103)
   - Im Custom-Modus: Color-Picker je Theme-Feld, plus
     `react-colorful` als Lib (leichtgewichtig, ~3 KB gz)
   - Live-Vorschau rechts neben den Pickers
   - **WCAG-Hard-Gate:** Save gesperrt bei Kontrast-Fail (mindestens
     AA-Ratio 4.5:1 für die drei kritischen Paare:
     primary/primaryFg, accent/accentFg, foreground/background)
   - Reset-Button „Zurück zum Preset"
5. **Configexport** — Owner-Direktive (revidiert 2026-06-11): **strikt
   bei Werten, tolerant bei unbekannten Keys**. Begründung: partielle
   NULL-Mappings bei ungültigen HEX-Werten erzeugen ein halb-gebrochenes
   Theme, das schlechter ist als ein klarer Reject. Ein Theme ist ein
   kohärentes Ganzes.
   - **Strikt:** ungültige HEX-Werte (z. B. `primary: "nicht-hex"`)
     → Import-Reject mit Field-Diagnose. Bundle muss korrigiert werden.
   - **Strikt:** JSON-Parse-Fail → Import-Reject mit klarem Fehler.
     Kein silent-NULL-Mapping.
   - **Strikt:** WCAG-Fail im Bundle (Theme ohne ausreichende
     Kontraste) → Import-Reject. Konsistent mit dem WCAG-Hard-Gate
     im Editor — wenn der Editor nicht savet, soll auch der Import
     nicht durchgehen.
   - **Tolerant:** unbekannte Keys (z. B. zukünftiges
     `shadowIntensity`) → gedroppt + per-Key-Warn-Log, Import läuft
     durch. Forward-Kompatibilität für künftige Onboarding-Versionen
     mit erweiterten Theme-Feldern.
6. **Mail-Templates + PDFs bleiben Out-of-Scope** (analog PROJ-102).
   Brand wirkt nur auf der Public-Web-Strecke.
7. **Geltungsbereich UI:** identisch PROJ-102 (`/register/<rc>` + Error-
   Pages innerhalb derselben PublicPageShell). `/confirm-email` bleibt
   außen vor (kennt RC nicht).

## Acceptance Criteria (Vorläufig, wird im Grilling verfeinert)

### Backend

- [ ] **AC-1** Migration 000077: `ADD COLUMN brand_theme JSONB NULL`.
- [ ] **AC-2** `RegistrationEntrypoint.BrandTheme *json.RawMessage` mit
  `db:"brand_theme"` + `json:"brandTheme,omitempty"`.
- [ ] **AC-3** Validator `IsValidBrandTheme(raw json.RawMessage) (ok
  bool, sanitized json.RawMessage)` — parsed JSON, validiert HEX-Format
  pro Farb-Feld, droppt unbekannte Keys, returns sanitized JSON für
  Persistierung. Tolerant gegenüber Teil-Sets (Admin kann nur 2 Farben
  setzen, andere bleiben weg).
- [ ] **AC-4** `SaveEEGSettings`-Handler nimmt `brandTheme` im Body
  entgegen (Patch-Semantik: nil = unverändert, leeres Objekt = NULL,
  valid → setzen, invalid HEX → 400 mit Field-Diagnose).
- [ ] **AC-5** Public-Response `getRegistrationConfig` erweitert um
  `brandTheme: object | null` (raw JSON durchreichen, nicht
  transformieren — Frontend rechnet HEX→HSL).
- [ ] **AC-6** Repo-Methode `SaveBrandTheme(rcNumber, *json.RawMessage)`
  analog zu `SaveBrandPreset`.

### Frontend — Public-Strecke

- [ ] **AC-7** `brand-presets.ts` erweitert um Override-Schicht
  `mergeCustomTheme(presetVars, customTheme) → finalVars`. Custom-
  Theme-Felder überschreiben selektiv, leere/fehlende Felder kommen
  vom Preset.
- [ ] **AC-8** HEX→HSL-Konvertierung beim SSR im
  `presetStyleBlock`-Pfad (kein Client-Bundle-Hit, nur Server-Render).
- [ ] **AC-9** `PublicPageShell` Props erweitert um `brandTheme`.
- [ ] **AC-10** Auto-Foreground-Berechnung wenn `primaryFg`/`accentFg`
  nicht gesetzt: Luminance-Check, weiß oder schwarz wird gewählt
  (defensive — Editor erzwingt im WCAG-Gate aber explizite Werte).

### Frontend — Admin-Editor

- [ ] **AC-11** `AdminBrandEditor` Modus-Switch zwischen Preset und
  Custom. Custom-Modus rendert Color-Picker-Grid + Live-Vorschau +
  WCAG-Panel.
- [ ] **AC-12** Color-Picker via `react-colorful` (~3 KB gz, Tree-
  Shaking-fähig). Kein größeres UI-Framework.
- [ ] **AC-13** Live-Vorschau: minimalistisches Render der echten
  Public-Page-Komponenten mit aktuellen Theme-Werten — kein iframe.
- [ ] **AC-14** WCAG-Kontrast-Check via npm-Package `wcag-contrast`
  (~1 KB). Drei kritische Paare:
  primary/primaryFg, accent/accentFg, foreground/background. Mindest-
  Ratio 4.5:1 (AA). Save-Button disabled bei Fail.
- [ ] **AC-15** Im PROJ-67-Awareness-Banner-Check
  (`isAdvancedEEGSettingsActive`): `brand_theme != null` ist non-default,
  triggert Banner (analog zu non-default Preset).
- [ ] **AC-16** Editor nur in PROJ-67-`advanced`-Modus sichtbar
  (Bestand-Bedingung aus PROJ-102, unverändert).

### Configexport (strikt bei Werten, tolerant bei unbekannten Keys)

- [ ] **AC-17** `EEGSettingsSection.BrandTheme *json.RawMessage`
  mit `omitempty`.
- [ ] **AC-18** Exporter reicht `ep.BrandTheme` 1:1 ins Bundle (kein
  Re-Format, kein Re-Validate beim Export).
- [ ] **AC-19** Importer ruft `validateBrandTheme(rcNumber, raw)` auf:
  - JSON-Parse-Fail → `ValidationError` → Import-Reject mit klarem
    Fehler („brand_theme: kein gültiges JSON")
  - Ungültige HEX-Werte → `ValidationError` mit Field-Diagnose →
    Import-Reject (z. B. „brand_theme.primary: `not-hex` ist kein
    gültiger HEX-Wert")
  - WCAG-Fail (Kontrast unter AA-Schwelle für primary/primaryFg,
    accent/accentFg, foreground/background) → `ValidationError` mit
    Kontrast-Werten → Import-Reject
  - Unbekannte Keys → gedroppt + per-Key-Warn-Log, Import läuft durch
    (Forward-Compat für künftige Onboarding-Versionen)
- [ ] **AC-20** Diff zeigt geänderte Theme-Keys einzeln (nicht den
  ganzen JSON-Blob als Diff-Block).

### Tests + Doku

- [ ] **AC-21** Backend-Tests: Sanitizer Round-Trip, Validator Edge-
  Cases, Repo-Save, Public-Response-Shape.
- [ ] **AC-22** Frontend-Tests: `mergeCustomTheme`, HEX→HSL,
  WCAG-Checker-Wrapper, AdminBrandEditor Render mit/ohne Custom-Theme.
- [ ] **AC-23** `go build ./...` clean, `npm run build` clean, alle
  Test-Suites grün.
- [ ] **AC-24** Doku: `docs/domain-model.md` (JSONB-Ausnahme +
  Theme-Felder-Liste), `docs/api-spec.md` (Public-Response + Admin-
  Patch), `docs/user-guide/06-admin-settings.md` (Custom-Theme-Modus,
  WCAG-Gate-Erläuterung, PROJ-frei, Muster-EEG-Beispiel),
  `docs/user-guide/changelog.md`, `CHANGELOG.md`.

## Edge Cases

- **EC-1** EEG ohne Brand-Theme → `brand_theme = NULL` → Page rendert
  mit Preset-Fallback (Bestand-Verhalten).
- **EC-2** EEG hat Preset + partielles Custom-Theme (nur 3 Farben
  überschrieben) → Render mischt Preset-Defaults mit Custom-
  Überschreibungen. Live-Vorschau zeigt das identisch.
- **EC-3** Admin lädt Configexport-Bundle aus älterer Onboarding-
  Version ohne `brandTheme` → Importer setzt nil → Page rendert mit
  Preset-Fallback. Vorwärtskompatibel.
- **EC-4** Admin lädt Bundle mit zukünftigen Theme-Keys (z. B. PROJ-104
  fügt `shadowIntensity` hinzu) → unbekannte Keys werden gedroppt +
  geloggt. Apply läuft durch.
- **EC-5** Admin pflegt Custom-Theme mit WCAG-Fail → Editor blockt Save.
  Beim Configexport-Import wird ein WCAG-failendes Theme **abgelehnt**
  (revidierte Owner-Direktive 2026-06-11) — konsistent zum Editor-Gate.
  Wenn ein Admin den UI-Gate umgehen will, muss er im Bundle den
  fehlerhaften Theme-Block manuell rausnehmen (= zurück zu Preset-
  Fallback).
- **EC-6** JSON-Payload-Größe: theoretisch unlimited, praktisch ~500 B
  pro Theme. Kein Concern.
- **EC-7** Konkurrierende Updates: Theme + Preset beide gleichzeitig
  → Auto-Save-Snapshot behandelt beide als atomare Felder, race-frei.
- **EC-8** Awareness-Banner: non-default Preset ODER non-NULL
  brand_theme → Banner. Beide gleichzeitig → ein Banner-Event, keine
  Doppelung.

## Out of Scope

- Mail-Templates + PDFs (bleibt Standard-Branding, analog PROJ-102).
- Admin-Bereich `/admin/*` (bleibt Teal).
- `/confirm-email` Page (kennt RC nicht, kann nicht gebrandet werden).
- Theme-Marktplatz / Sharing zwischen EEGs (kein Tester-Wunsch).
- Dark/Light-Mode-Switch (Public-Page ist Dark-only, bleibt so).
- Per-User-Theme-Override (sinnlos, Public-Page hat keine User-Session).
- Theme-Editor in der Public-Page (Mitglied sieht nur Result).

## Deployment

**Deploy-Bookkeeping 2026-06-11 (Abend):**

- Feature-Bump: zwei neue Spalten, JSONB-Pfad, Custom-Editor-Umbau, zwei neue npm-libs → Minor-Version-Wechsel
- Tag: `v1.30.0-PROJ-103`
- KEINE neuen ENV-Variablen
- KEINE Helm-Wert-Änderungen
- Migration 000077 läuft via migrate-Job automatisch vor Backend-Rollout, non-blocking (ALTER ADD COLUMN nullable JSONB + ADD COLUMN TEXT NOT NULL DEFAULT 'preset')
- npm dependency tree wächst um `react-colorful` + `wcag-contrast` (~4 KB gz Admin-only Bundle)

**Owner-Aktion nach CI-Build + Helm-Auto-Bump:**

```bash
helm upgrade eegfaktura-member-onboarding ./helm/member-onboarding \
  -f helm/member-onboarding/values-env.yaml \
  -f helm/member-onboarding/values-secret.yaml
```

**Tester-Verifikation nach Deploy:**

- Bestand-EEG ohne Brand-Settings → Public-Page rendert weiter Default-Teal (`brand_mode` defaultet auf `'preset'`)
- Admin „Alle Optionen" → Brand-Editor zeigt 2 Tabs
- Tab „Eigene Farben" → 8 Color-Picker + Font-Select + Live-Vorschau + WCAG-Panel
- Schwacher Kontrast (z. B. `#888888` / `#999999`) → WCAG-Panel zeigt Fail, Save blockiert
- Akzeptable Farben (z. B. `#1c3a28` / `#ffffff`) → Save geht durch, Public-Page rendert Custom-Theme
- Tab-Wechsel zurück auf Preset → `brand_mode='preset'`, Theme bleibt in DB persistiert
- Standard-Modus → Awareness-Banner schlägt bei aktivem Custom-Theme

## QA Test Results

**Tester:** QA Engineer (AI)
**Date:** 2026-06-11
**Scope:** Backend (Welle 1) + Frontend (Welle 2+3). Doku (Welle 4) ist im /deploy-Commit, out-of-scope für QA.

### Test-Suite-Stand (nach Fixes)

| Suite | Ergebnis |
|---|---|
| `go test ./...` | ✓ alle 12 Pakete grün |
| `go build ./...` | ✓ clean |
| `npx tsc --noEmit` | ✓ clean |
| `npx vitest run` | ✓ 156/156 grün (vorher 108 + 48 PROJ-103) |
| `NEXT_PUBLIC_TEST_AUTH_MODE= npm run build` | ✓ clean (5.5s) |
| `govulncheck ./...` | ✓ 0 callable (5 in transitiver Bestand, nicht callable) |
| `gosec -severity medium -confidence medium` | ✓ 0 Issues über 87 Files |
| `npm audit --audit-level=high` | ✓ 0 high (4 moderate aus Bestand pre-PROJ-103) |

### Acceptance-Criteria-Sweep (24 ACs)

| AC-Block | Status |
|---|---|
| AC-1 bis AC-6 Backend (Migration + Validator + Repo + Service + Handler + Response) | ✓ alle PASS |
| AC-7 bis AC-10 Public-Strecke (mergeCustomTheme, HEX→HSL, PublicPageShell, Auto-Foreground) | ✓ alle PASS |
| AC-11 bis AC-16 Admin-Editor (Tabs, Picker, Vorschau, WCAG, Awareness-Banner, advanced-only) | ✓ alle PASS (nach Fix-Welle, siehe unten) |
| AC-17 bis AC-20 Configexport (Schema, Exporter, Importer, Diff) | ✓ alle PASS |
| AC-21 Backend-Tests (~40 Cases inkl. gemeinsamer HSL-Vektor) | ✓ PASS |
| AC-22 Frontend-Tests (48 neue Cases) | ✓ PASS |
| AC-23 Build/Test clean | ✓ PASS |
| AC-24 Doku im Deploy-Commit | ⏳ DEFERRED |

**Resultat: 23/23 PASS + 1 deferred. Keine fehlgeschlagenen ACs.**

### Edge-Case-Sweep (8 ECs)

| EC | Status |
|---|---|
| EC-1 Bestand-EEG ohne Theme → DB-NULL → Default-Teal | ✓ PASS (Migration setzt brand_mode='preset' Default) |
| EC-2 Partial Theme → Preset-Defaults füllen Rest | ✓ PASS (`hexOrPreset`-Fallback in `mergeCustomTheme`) |
| EC-3 Pre-PROJ-103-Bundle | ✓ PASS (tolerant nil in Importer) |
| EC-4 Bundle mit zukünftigen Theme-Keys | ✓ PASS (gedroppt + Warn-Log) |
| EC-5 Editor-Gate vs Configexport (revidiert: beide strikt) | ✓ PASS |
| EC-6 JSON-Payload-Größe | ✓ PASS (~500 B) |
| EC-7 Konkurrente Updates Theme + Mode | ✓ PASS (atomar in einem PUT-Body) |
| EC-8 Awareness-Banner einheitlich | ✓ PASS (mit Info-5 zur leeren-Theme-Edge) |

### Security-Smoke

| Bereich | Status |
|---|---|
| Auth/Authz | ✓ Bestand-Pattern (Keycloak + checkTenantAccess) unverändert |
| Input-Validation | ✓ ValidateBrandTheme strikt; DB-CHECK Safety-Net |
| SQL-Injection | ✓ alle UPDATEs parameterisiert |
| XSS via brandTheme HEX | ✓ `hexToHsl` wirft bei invalid → Fallback auf Preset; nur valide HSL landen im Style-Block |
| dangerouslySetInnerHTML | ✓ zwei Bouncer-Schichten (Backend-Validator + Frontend-Bouncer); Doku-Kommentar erweitert |
| CSS-Injection via fontFamily | ✓ Whitelist-Filter `isAllowedFont` + hardgecodete `FONT_FAMILY_STACKS`-Map |
| PII in Logs | ✓ keine PII; nur rc_number + dropped key names im Warn-Log |
| Length-Limits | ✓ HEX max 7 Zeichen (Regex), Font max 9 Zeichen (Whitelist) |
| Payload-Größe | ✓ Worst-Case unverändert (~400 KB Logo dominiert, brand_theme < 1 KB) |

### Regression-Test

| Verwandtes Feature | Status |
|---|---|
| PROJ-102 Preset-Pfad (Tabs-Umbau) | ✓ Preset-Tab funktioniert identisch zu vorher |
| PROJ-84 Auto-Save (Snapshot um 2 Felder erweitert) | ✓ Cross-Field-Gates unverändert |
| PROJ-67 Awareness-Banner (einheitlicher Trigger) | ✓ Bestand-Cases (boardApproval, sepaB2B, cooperative, …) weiter funktional |
| PROJ-81 SEPA-Optional + Handler-Reihenfolge | ✓ Brand-Patch-Blöcke sind NACH der SEPA-Validation |
| PROJ-33 Logo-Sync | ✓ unverändert |
| PROJ-31 /confirm-email Page | ✓ unverändert (kennt keine RC) |

### Findings + Fix-Welle 1 (während dieser QA-Phase)

| # | Severity | Datei | Befund | Fix |
|---|---|---|---|---|
| 1 | Medium | `brand-custom-editor.tsx:200` | `useMemo` mit Side-Effect für `onValidationChange` — Anti-Pattern, fired während Render, kann unter strict-mode doppelt triggern | ✓ FIXED: `useMemo` → `useEffect` mit identischen Deps |
| 2 | Medium | `brand-custom-editor.tsx:315` | `setLocalHex(value)` während Render in `ColorField` überschreibt Teilstrings beim HEX-Tippen (z. B. `#ff00` → `#abcdef`) | ✓ FIXED: Sync nur bei externem Value-Change via `lastExternalValueRef` |
| 3 | Medium | `admin-eeg-settings-editor.tsx:412` | WCAG-Gate blockt **alle** EEG-Settings-Saves wenn Brand-Theme fail — auch nicht-brand-bezogene Änderungen (z. B. SEPA-Toggle) konnten nicht gespeichert werden, solange das Theme aus einer früheren Session in einem WCAG-Fail-Zustand persistiert war | ✓ FIXED: Gate prüft jetzt zusätzlich `brandTouched` (brandThemeJson oder brandMode != savedSnapshot) — blockt nur wenn Brand-Felder tatsächlich geändert wurden |
| 4 | Low | `brand-custom-editor.tsx` `ColorField` | Invalid-HEX in Input bleibt nach onBlur stehen (kein Reset auf Parent-Value) | DEFERRED — kosmetisch, User kann mit gültigem HEX überschreiben |
| 5 | Info | `settings-mode.ts` `isAdvancedEEGSettingsActive` | Leeres Theme `{v:1}` (= nur Schema-Tag, keine Custom-Felder) triggert Awareness-Banner via `Object.keys > 0` | Akzeptiert — sobald `brand_mode='custom'` explizit gewählt wurde, ist Banner-Trigger korrekt |

**Nach Fix-Welle 1: 0 Critical/High/Medium, 1 Low deferred, 1 Info akzeptiert.**

### Production-Ready-Entscheidung

**READY** — keine blockierenden Bugs nach Fix-Welle 1.

## Security Review

**Reviewer:** Security Engineer (AI)
**Date:** 2026-06-11
**Scope:** Migration 000077, neue Backend-Module (`hsl.go`, `brand_theme.go`), `RegistrationConfig`-Erweiterung, `SaveBrandTheme`/`SaveBrandMode`, `SaveEEGSettings`-Patch-Blöcke, Configexport-Pipeline-Erweiterung, neue Frontend-Module (`hsl.ts`, `brand-presets.ts` Custom-Theme-Layer, `brand-custom-editor.tsx`, `admin-brand-editor.tsx` Tabs-Umbau), zwei neue npm-Dependencies.

### Threat Model Summary

Drei Hauptangriffsflächen wurden untersucht: (1) der erweiterte
`dangerouslySetInnerHTML`-Pfad mit Theme-Daten — abgesichert durch
**zwei Bouncer-Schichten** (Backend-`ValidateBrandTheme` mit strikter
HEX-Regex + Font-Whitelist + Versions-Tag + WCAG-Gate; Frontend-
`hexOrPreset`/`isAllowedFont` der defensive auf Preset zurückfällt
selbst bei kompromittiertem Backend); (2) der JSONB-Persistenz-Pfad
mit Forward-Compat-Toleranz für unbekannte Keys (strikt-bei-Werten
verhindert silent-NULL-Korruption); (3) die zwei neuen npm-Libs
(`react-colorful` + `wcag-contrast`) — beide etablierte schmale
Utility-Pakete ohne bekannte CVEs.

### Findings

| Severity | Datei | Funktion | Risiko | Exploit-Szenario | Fix-Empfehlung | Confidence |
|---|---|---|---|---|---|---|
| Info | internal/shared/brand_theme.go vs src/lib/hsl.ts | WCAG-Math parallel-implementiert | Backend Go-WCAG + Frontend `wcag-contrast` Lib divergieren über die Zeit | Bei künftigem WCAG-3-Update oder Lib-Patch könnten Werte um ±0.1:1 abweichen → ein Theme das im Editor passt, wird vom Backend abgelehnt (oder umgekehrt) | Optional als Folge-PROJ: gemeinsamer Test-Vektor aus W3C-Referenz-Tabelle in beiden Test-Suiten, der bei Drift sofort meldet | High |
| Info | src/components/public-page-shell.tsx | dangerouslySetInnerHTML Pattern-Erweiterung | Wenn ein Entwickler später eine zweite Stelle einführt ohne Bouncer-Vertrag | Hypothetisch: jemand kopiert das Pattern für Mail-Templates oder PDF-Footers ohne ValidateBrandTheme-äquivalenten Filter | Optional: ESLint-Custom-Rule die `dangerouslySetInnerHTML` außerhalb von `public-page-shell.tsx` als Warn-Level markiert; CODEOWNERS für die Datei | Medium |
| Info | internal/configexport/importer.go | Theme-Apply ohne Audit-Trail | Theme-Wechsel via Configexport-Import hinterlässt keine Spur in status_log o. ä. | Forensik nach versehentlichem oder bösartigem Theme-Wechsel ist nur via DB-`updated_at` + manueller Auditierung möglich | Akzeptiert per Owner-Direktive — Theme ist reine Präsentations-Konfiguration, kein Audit-Bedarf in V1 | High |
| Info | tailwind.config.ts + globals.css | `--font-family` als globale CSS-Variable | Neue Variable wird von ALLEN Routen gelesen (admin + customer-onboarding + agb + …) | Wenn jemand später `font-sans` außerhalb von /register/<rc> erwartet aber das Custom-Theme den Stack ändert | Defensive: nur die PublicPageShell überschreibt die Variable im SSR-Block; alle anderen Routen lesen den Default aus globals.css. Funktional und nachvollziehbar. | High |
| Info | internal/shared/brand_theme.go ValidateBrandTheme | Empty/Whitespace-only String wird re-marshaled mit "primary":"" Feldern | Cosmetic: das sanitized JSON kann optionale Felder als leere Strings serialisieren | Speicher-Effizienz minimal beeinflusst (~50 Byte pro Theme), keine Sicherheitsrelevanz | Optional: `omitempty` ist gesetzt, also werden leere Strings nicht re-emitted — bereits korrekt | High |

**0 Critical / 0 High / 0 Medium / 0 Low / 5 Info.**

Alle 5 Info-Findings sind dokumentiert, eines davon (ESLint-Rule für
dangerouslySetInnerHTML) wäre ein nice-to-have für eine Folge-PROJ;
keiner blockiert den Deploy.

### Scan Results

| Scan | Ergebnis |
|---|---|
| `govulncheck ./...` | ✓ 0 callable; 5 transitive nicht-callable (Bestand) |
| `gosec -severity medium -confidence medium` | ✓ 0 Issues über 87 Files |
| `npm audit --audit-level=high` | ✓ 0 high (4 moderate Bestand pre-PROJ-103) |
| `trivy config helm/ --severity HIGH,CRITICAL` | ✓ 0 misconfigurations |
| Semgrep | ✗ nicht ausgeführt (lokal nicht installiert; CI-Job läuft separat) |

### Verdict: APPROVED

**Begründung:** 0 Critical/High/Medium/Low-Findings nach der QA-Fix-
Welle. Das zwei-Layer-Bouncer-Modell (Backend-Validator + Frontend-
Sanitizer) ist robust gegen CSS-Injection selbst bei einem
kompromittierten Backend. Die 5 Info-Findings sind alle akzeptierbar
oder beziehen sich auf Pattern-Etablierung für Folge-PROJs.

**Empfohlene Folge-PROJs (optional):**
- W3C-Referenz-Test-Vektor für WCAG-Math zwischen Go-Backend und
  JS-Frontend (Drift-Wache)
- ESLint-Custom-Rule die zusätzliche `dangerouslySetInnerHTML`-Stellen
  als Warn-Level markiert
- Diese sind keine Blocker — PROJ-103 ist deploy-fertig.

**Empfehlung:** `/security-review` als nächster Schritt, weil das Feature
mehrere security-relevante Bereiche berührt:
- Neue DB-Spalte mit CHECK-Constraint + neue JSONB-Spalte
- Public-Endpoint-Response um JSONB erweitert
- `dangerouslySetInnerHTML` Pattern-Erweiterung (themeStyleBlock statt presetStyleBlock)
- Zwei neue npm-Libs (`react-colorful`, `wcag-contrast`)

## Festgenagelte Owner-Entscheidungen (Grilling 2026-06-11)

| # | Punkt | Entscheidung |
|---|---|---|
| 1 | **Preset als Basis-Layer** | Pflicht. Admin wählt erst ein Preset, dann überschreibt Custom-Theme selektiv. Nie weißes Blatt. |
| 2 | **Render-Prio bei beiden gesetzt** | **Brand-Mode-Spalte (`brand_mode`) entscheidet** (Revision durch Q13). Wenn `brand_mode = 'custom'` UND `brand_theme != NULL` → rendert Custom-Theme. Sonst rendert Preset. |
| 3 | **Editierbare Felder** | 8 (`background`, `foreground`, `primary`, `primaryFg`, `accent`, `accentFg`, `card`, `cardFg`). 9 abgeleitet via deterministische HSL-Mathematik (`border` = card -10% L, `ring` = primary, `popover` = card -2% L, `popoverFg` = cardFg, `secondary` = accent -30% S, `secondaryFg` = accentFg, `muted` = background +5% L, `mutedFg` = foreground -30% L, `input` = card -5% L). |
| 4 | **WCAG-Level** | AA (4.5:1) als Hard-Gate. |
| 5 | **Font-Whitelist** | 4 System-Stacks: `sans-serif` (default Inter), `serif`, `monospace`, `system-ui`. Google Fonts als Folge-PROJ. |
| 6 | **Border-Radius** | **Nicht in V1.** Bleibt `globals.css` (0rem). Folge-PROJ wenn EEGs nachfragen. |
| 7 | **WCAG-Paare** | 3 Paare: `primary/primaryFg`, `accent/accentFg`, `foreground/background`. |
| 8 | **Header-Subtext-Farbe** | Bleibt `text-primary` (= automatisch durch Custom-Primary umgefärbt). Kein separates Theme-Feld. |
| 9 | **Modus-Switch-UI** | shadcn `Tabs` — zwei Tabs „Preset" und „Eigene Farben". |
| 10 | **Custom-Preset speichern** | V1 nein. Ein Custom-Theme pro EEG. Folge-PROJ wenn EEGs nachfragen. |
| 11 | **Auto-Foreground** | Backend pickt Schwarz/Weiß via Luminance-Check für Configexport-Imports ohne FG-Feld. Editor erzwingt im WCAG-Gate explizite Werte. |
| 12 | **PROJ-67-Awareness-Banner** | Einheitlicher Trigger: `isAdvancedEEGSettingsActive` true wenn non-default Preset **oder** `brand_mode = 'custom'` mit non-NULL theme. 1 Banner-Event, kein Doppel-Render. |
| 13 | **Tab-Wechsel-Verhalten** | Custom-Theme bleibt persistiert wenn Admin auf Preset-Tab zurückwechselt und speichert. Schaltet sich nur via `brand_mode = 'preset'` „inaktiv". Erlaubt Re-Wechsel ohne Re-Konfiguration. → **Erzwingt zusätzliche `brand_mode`-Spalte** (siehe Q2). |
| 14 | **HEX-Format** | Nur 6-stellig `#RRGGBB`. Keine 3-stellige Shorthand, kein Alpha-Kanal. |
| 15 | **Schema-Versioning** | Pflicht-Key `v: 1` im JSONB-Theme. Validator rejected Themes ohne `v`. Saubere Migrations-Tür für V2. |

## Daten-Modell-Konsequenz aus den Entscheidungen

Drei Spalten auf `registration_entrypoint`:

- **`brand_preset` (bestand PROJ-102)** — TEXT NULL CHECK IN ('teal','leaf','sun','slatey'). Welches Preset als Fundament.
- **`brand_theme` (PROJ-103 neu)** — JSONB NULL. Custom-Theme oder NULL wenn nicht konfiguriert.
- **`brand_mode` (PROJ-103 neu)** — TEXT NOT NULL DEFAULT 'preset' CHECK IN ('preset','custom'). Welcher Pfad rendert. Default 'preset' (Bestand-Verhalten wird nicht gebrochen).

Render-Entscheidungsbaum (Frontend `mergeCustomTheme`):

```
if brand_mode == 'custom' AND brand_theme != NULL:
    final_vars = preset_vars(brand_preset ?? 'teal')
                 |> overlay(brand_theme.colors)       # selektive Überschreibung
                 |> derive_secondary_9_vars()         # abgeleitete Felder rechnen
else:
    final_vars = preset_vars(brand_preset ?? 'teal')  # Bestand-PROJ-102-Verhalten
```

## Beispiel-JSONB-Theme

```json
{
  "v": 1,
  "primary":     "#3f8856",
  "primaryFg":   "#ffffff",
  "accent":      "#5ea571",
  "accentFg":    "#0e2015",
  "background":  "#1a2820",
  "foreground":  "#f1f8f3",
  "card":        "#22332a",
  "cardFg":      "#f1f8f3",
  "fontFamily":  "sans-serif"
}
```

Theme-Bundles ohne `v` oder mit `v != 1` werden vom Validator abgelehnt
(strikt). Unbekannte Top-Level-Keys (z. B. `shadowIntensity` aus einer
zukünftigen Version) werden gedroppt + geloggt (tolerant).

## Offene Klärungen (für /architecture)

Alle 15 Hauptpunkte sind oben in der Owner-Entscheidungen-Tabelle
festgenagelt. Für /architecture bleiben nur Implementierungs-Details:

- HEX→HSL-Konvertierung: Go-Helper für SSR-Backend + TS-Helper für
  Editor-Live-Vorschau. Parallel-Implementation (standard Math), beide
  müssen identische HSL-Tripel liefern für identischen Render.
- Konkrete Formeln für die 9 abgeleiteten CSS-Variablen (Lightness-/
  Saturation-Verschiebung).
- Validator-Signatur (`ValidateBrandTheme(raw) (sanitized, err)`)
  inklusive WCAG-Check.

## Tech Design (Solution Architect)

### A) Befunde aus den Vor-Implementierungs-Checks

1. **Bibliotheken `react-colorful` und `wcag-contrast` sind nicht installiert.**
   Beide kommen im Frontend-Wave via `npm install --save react-colorful
   wcag-contrast`. Zusammen ~4 KB gzipped, beide Tree-Shaking-fähig und
   Dependabot-bekannte Größen.
2. **`globals.css :root` hat heute keine `--font-family`-Variable.**
   Wir führen sie als 18. Tailwind-CSS-Variable ein, mit Default
   `sans-serif`. Das ist additiv — Bestand-Komponenten verwenden weiter
   die Tailwind-Default-`font-sans`-Klasse, die jetzt aus der Variable
   liest.
3. **`brand-presets.ts` hat eine klare named-export-API** (9 Exports inkl.
   `BRAND_PRESET_VARIABLES`, `presetStyleBlock`, `normalizeBrandPreset`).
   Der neue `mergeCustomTheme(presetVars, customTheme)`-Helper reiht sich
   als 10. Export ein. Bestehende Aufrufer von `presetStyleBlock` ändern
   sich nicht.
4. **`AdminBrandEditor` aus PROJ-102 ist stateless-controlled** (`{value,
   onChange, disabled}`). Auto-Save passiert im Parent
   (`AdminEEGSettingsEditor`) via PROJ-84-`useDebouncedAutoSave`-Hook.
   Der Umbau erweitert nur den Editor um zwei Props (`mode`, `theme`)
   und je einen `onChange`-Callback — die Auto-Save-Pipeline im Parent
   schluckt zwei zusätzliche Snapshot-Felder, unverändertes Pattern.

### B) Component-Tree

**Public-Strecke** (`/register/<rc>`) — strukturell wie PROJ-102, nur
mit zwei zusätzlichen Render-Eingaben:

```
PublicPageShell  (erweitert)
├── SSR-<style>-Injection
│   └── mergeCustomTheme(preset, mode, theme) → CSS-Variablen-Block
│       (statt nur presetStyleBlock(preset))
├── PublicHeader  (PROJ-102, unverändert)
├── main → RegistrationForm  (Bestand)
└── Footer  ("Powered by eegFaktura"-Switch unverändert)
```

**Admin-Editor** — komplett umgebaut von „Single-Select" auf „Tabs":

```
AdminBrandEditor  (umgebaut)
├── Tabs (shadcn)
│   ├── Tab "Preset"   ← bisheriges PROJ-102-Verhalten
│   │   └── Select + 4 Preview-Karten (Bestand)
│   └── Tab "Eigene Farben"   ← neu
│       └── BrandCustomEditor  (neue Komponente)
│           ├── Color-Picker-Grid (8 Felder)
│           │   └── ColorField × 8
│           │       ├── react-colorful Picker
│           │       ├── HEX-Input (synchron)
│           │       └── Label + Hilfetext
│           ├── Font-Select (4 System-Stacks)
│           ├── Live-Vorschau-Karte
│           │   └── nutzt denselben mergeCustomTheme-Helper
│           ├── WCAG-Kontrast-Panel
│           │   └── 3 Status-Anzeigen (primary/primaryFg,
│           │       accent/accentFg, foreground/background)
│           └── "Zurück zum Preset"-Hint  (= Hinweis auf Tab-Wechsel)
```

**Tab-Wechsel-Semantik:**
- Wechsel zwischen Tabs ändert nur den lokalen Editor-State und das
  Parent-Snapshot-Feld `brandMode`. `brand_theme` bleibt im Parent
  unangetastet (= persistiert).
- Auto-Save reagiert auf jeden Mode-Wechsel → DB-`brand_mode` wird
  sofort aktualisiert, Public-Page rendert sofort neu.
- WCAG-Gate ist nur im „Eigene Farben"-Tab aktiv. Im Preset-Tab gibt's
  keinen Gate (Presets sind durchgetestet).

### C) Datenmodell-Erweiterung

**Drei Spalten auf `registration_entrypoint`:**

- **`brand_preset`** *(Bestand PROJ-102)* — bleibt wie heute: Identifier
  eines der vier vordefinierten Presets oder NULL (= Default-Teal).
  Dient jetzt **immer** als Fundament — auch wenn der Custom-Modus
  aktiv ist, sind die 9 abgeleiteten Variablen und die nicht
  überschriebenen Felder Preset-Werte.
- **`brand_theme`** *(NEU)* — JSONB. Hält das Custom-Theme als
  strukturiertes Dokument mit Pflicht-Versions-Tag und 8 Color-Keys
  plus optional `fontFamily`. NULL solange der Admin im Custom-Modus
  noch nichts konfiguriert hat.
- **`brand_mode`** *(NEU)* — Text, nicht-NULL, Default `'preset'`,
  DB-Constraint auf `('preset','custom')`. Entscheidet welcher Pfad
  rendert. Default `'preset'` ist Bestand-PROJ-102-Verhalten — keine
  Bestand-EEG erfährt eine Verhaltens-Änderung durch die Migration.

**JSONB-Schema (`brand_theme`):**

Vorhanden müssen sein:
- `v: 1` — Schema-Versions-Tag (Pflicht)

Optional vorhanden (jeder Key strikt validiert):
- 8 Color-Keys — `primary`, `primaryFg`, `accent`, `accentFg`,
  `background`, `foreground`, `card`, `cardFg` — alle HEX-Format
  `#RRGGBB` (6-stellig, validiert per Regex)
- `fontFamily` — eine der 4 Whitelist-Strings: `sans-serif` / `serif`
  / `monospace` / `system-ui`

Unbekannte Top-Level-Keys werden gedroppt + per-Key-Warn-Log (Forward-
Compat). Ungültige Werte → Reject mit Field-Diagnose.

**Render-Entscheidungsbaum** (Frontend `mergeCustomTheme(presetVars,
mode, theme)`):

```
Schritt 1: Wenn mode = 'preset' ODER theme ist NULL:
           → finalVars = presetVars  (Bestand-PROJ-102-Pfad)
           → fertig.

Schritt 2: Wenn mode = 'custom' UND theme ist gesetzt:
           → Start mit presetVars als Fallback-Schicht
           → Für jedes Color-Feld im theme:
                Konvertiere HEX → HSL-Tripel
                Überschreibe entsprechende CSS-Variable
           → Wenn fontFamily im theme: überschreibe --font-family
           → Berechne die 9 abgeleiteten Variablen aus den
             8 expliziten (oder Preset-Fallback wenn Feld leer)
           → finalVars = das gemischte Variablen-Set
           → fertig.
```

**Konkrete Ableitungsformeln für die 9 Sekundär-Variablen:**

| Abgeleitete Variable | Formel | Begründung |
|---|---|---|
| `border` | `card` mit Lightness −10% | Subtile Trennlinie um Karten |
| `ring` | `primary` | Focus-Ring soll Brand-Farbe spiegeln |
| `popover` | `card` mit Lightness −2% | Popover hebt sich minimal von Karte ab |
| `popoverFg` | `cardFg` | Text in Popover wie in Karte |
| `secondary` | `accent` mit Saturation −30% | Sekundär-Akzent matter als Haupt-Akzent |
| `secondaryFg` | `accentFg` | Sekundär-Text wie Akzent-Text |
| `muted` | `background` mit Lightness +5% | Stiller Hintergrund-Block |
| `mutedFg` | `foreground` mit Lightness −30% | Gedämpfter Text |
| `input` | `card` mit Lightness −5% | Input-Hintergrund stiller als Karte |

Alle Formeln sind **deterministische HSL-Mathematik** — gleicher Input,
gleicher Output, ohne Zufall oder versteckte Heuristik.

### D) Tech-Entscheidungen mit Begründung

**1. HEX→HSL parallel Backend + Frontend (nicht single-source).**
Backend rechnet die Konvertierung beim SSR (für die Public-Page),
Frontend rechnet sie im Live-Vorschau-Editor (für den Admin). Single-
Source-of-Truth ginge nur über einen API-Roundtrip pro Picker-Bewegung
— inakzeptabel laggy. HEX→HSL ist standardisierte Math (kein Algorithm-
Drift möglich), wir verifizieren via gemeinsamem Test-Vektor.

**2. `brand_mode`-Spalte statt JSONB-internem `mode`-Feld.**
DB-Constraint auf `('preset','custom')` ist als Spalte trivial, in
JSONB nur per Application-Layer. Plus: spätere Queries „wie viele EEGs
nutzen Custom-Themes" sind dann ein einfacher SELECT statt JSON-Path-
Query. Plus: Mode-Wechsel ohne Theme-Daten-Touch — Auto-Save
serialisiert nicht den ganzen JSON-Blob nur weil der Mode flippt.

**3. Schema-Versions-Tag `v: 1` als Pflicht-Anker.**
Wenn V2 ein Theme-Feld umbenennt (z. B. `primary` → `primaryColor`)
oder ein neues Pflicht-Feld einführt, kann der Validator über `v`
unterscheiden statt heuristisch (anwesende/fehlende Keys) zu raten.
Mini-Overhead (5 Bytes pro Theme), große Migrations-Klarheit.

**4. Abgeleitete CSS-Variablen via deterministische HSL-Mathematik.**
Keine Heuristik, keine LLM-Bewertung, keine Tabellen — die Formeln
sind im Code-Kommentar dokumentiert und ein Test pinnt sie. Wenn EEGs
in V2 eine andere Sekundär-Strategie wollen, ist das ein einzelnes
Funktions-Update.

**5. shadcn `Tabs` statt ToggleGroup oder Select.**
Tabs trennen die beiden Modi visuell sauber. Beim Wechsel sieht der
Admin sofort den passenden Inhalts-Block (Preset-Karten vs.
Custom-Editor). ToggleGroup/Select würden die Inhalte überlagern oder
verstecken — schlechte Discoverability.

**6. Bibliothek `react-colorful` für Color-Picker.**
~3 KB gz, keine Abhängigkeiten, Tree-Shaking-fähig, etabliert,
TypeScript-tauglich. Alternative `react-color` ist ~30 KB — zu groß
für den UX-Nutzen. Eigener Color-Picker wäre 1–2 Tage Aufwand für
keinen Mehrwert.

**7. Bibliothek `wcag-contrast` für Kontrast-Check.**
~1 KB, pure-Funktion `hex(a, b) → number`, kein UI. Wir wrappen die
Funktion in unserem eigenen `wcagAA(a, b) → boolean`-Helper, damit
die AA-Schwelle eine einzige Stelle hat.

**8. Auto-Foreground-Picker im Backend (Defense-in-Depth).**
Configexport-Bundles aus älteren oder fremden Onboarding-Versionen
können theoretisch ein `primary` ohne `primaryFg` mitschicken. Der
Backend-Validator wirft die Bundle nicht weg, sondern füllt das fehlende
FG-Feld auto via Luminance (schwarz oder weiß je nach Helligkeit).
Editor-UI erzwingt im WCAG-Gate trotzdem explizite Werte — der Auto-
Picker ist nur die Defensive für Import-Pfade.

**9. JSONB statt mehrerer separater Spalten.**
Owner-Entscheidung mit CLAUDE.md-Ausnahme. Begründung in der Spec:
reine Präsentations-Konfiguration, kein Joins/Filter/Reporting-Use-Case.
Bei jeder neuen Theme-Eigenschaft eine Migration wäre Bürokratie.

**10. Public-Page-Bundle bleibt schlank.**
`react-colorful` und `wcag-contrast` werden **nur** im Admin-Bundle
genutzt. Public-Page bekommt vom Backend fertige HSL-Tripel im SSR-
Style-Block — kein Konvertierungs-Code im Browser. Bundle-Wachstum
der Public-Page = 0 KB.

### E) Implementierungs-Reihenfolge

**Welle 1 — Backend (Migration + Validator + Pipeline):**
1. Migration 000077 — `brand_theme` JSONB + `brand_mode` TEXT + Constraint
2. `RegistrationEntrypoint`-Modell um zwei Felder erweitern
3. Theme-Validator-Modul (`shared/brand_theme.go`):
   - HEX-Format-Check
   - Versions-Tag-Check
   - WCAG-Check (Pflicht-Paare aus Welle 1)
   - Auto-Foreground-Picker
   - Sanitizer (unbekannte Keys droppen)
4. HSL-Mathematik (`shared/hsl.go`): HEX↔HSL, Lightness/Saturation-
   Operationen, abgeleitete-Variablen-Berechnung
5. Repo um zwei Methoden erweitern (`SaveBrandTheme`,
   `SaveBrandMode`) — atomare Felder, getrennte Saves
6. `getRegistrationConfig`-Service reicht `brandMode` + `brandTheme`
   in die Public-Response durch (HEX direkt, keine Backend-
   Konvertierung — Frontend rechnet via Backend-bereitgestelltem
   Helper, gemeinsamer Test-Vektor)
7. Admin-Handler: SaveEEGSettings nimmt beide Felder im Body entgegen
8. Configexport: Schema + Exporter + Importer + Diff um beide Felder
   erweitern (strikt-bei-Werten / tolerant-bei-Keys)
9. Backend-Tests: Validator-Roundtrip, HSL-Mathematik-Vektor,
   Configexport-Sanitizer

**Welle 2 — Frontend Public-Strecke:**
1. `npm install react-colorful wcag-contrast`
2. `globals.css :root` um `--font-family: sans-serif` erweitern;
   `tailwind.config.ts` um `fontFamily.sans` aus der Variable lesen
3. `brand-presets.ts` erweitern:
   - `BrandTheme`-TS-Typ
   - `mergeCustomTheme(preset, mode, theme) → CSSVariables` Helper
   - `hexToHslTriple(hex) → "H S% L%"` Helper
   - Ableitungsformeln-Modul
4. `PublicPageShell` Props um `brandMode` + `brandTheme` erweitern,
   `mergeCustomTheme`-Aufruf statt `presetStyleBlock`
5. `lib/api.ts` Types erweitern (3 Stellen: RegistrationConfig,
   EEGSettings, EEGSettingsSavePayload)
6. Frontend-Tests: `mergeCustomTheme`-Snapshot pro Modus, HEX→HSL-
   Vektor-Test (sollte mit Backend-Vektor identisch sein)

**Welle 3 — Frontend Admin-Editor:**
1. Neue Komponente `BrandCustomEditor` (Color-Picker-Grid +
   Font-Select + Live-Vorschau + WCAG-Panel)
2. `AdminBrandEditor` umgebaut auf Tabs (Preset-Tab + Custom-Tab),
   beide Tabs steuern den Parent über separate `onChange`-Callbacks
3. `AdminEEGSettingsEditor` (Parent) erweitert: `brandTheme` und
   `brandMode` ins `Snapshot`-Type + reloadSettings + autoSave-Payload
   + discard-Reset
4. `settings-mode.ts` `isAdvancedEEGSettingsActive` triggert auch auf
   `brandMode === 'custom'`
5. Frontend-Tests: Tab-Switch ohne Save verliert nichts, WCAG-Gate
   blockt Save bei Fail, Live-Vorschau spiegelt aktuelle Picker-Werte

**Welle 4 — Doku + Deploy:**
- `docs/domain-model.md` — JSONB-Ausnahme + drei neue Spalten + Theme-
  Schema dokumentieren
- `docs/api-spec.md` — Public-Response + Admin-Patch um die beiden
  neuen Felder ergänzen
- `docs/user-guide/06-admin-settings.md` — Custom-Theme-Modus + WCAG-
  Gate-Erläuterung, PROJ-frei, Muster-EEG als Beispiel
- `docs/user-guide/changelog.md` + `CHANGELOG.md` im Deploy-Commit
- Tag `v1.30.0-PROJ-103` (Feature-Bump: zwei neue Spalten, neuer
  JSONB-Pfad, Admin-Editor-Umbau)
- Owner führt `helm upgrade` manuell aus

### F) Was nicht geändert wird

- Authentifizierung, Tenant-Isolation, Status-Modell — unverändert
- Public-Header bleibt funktional identisch (eegName + Logo aus
  PROJ-102, kein neues Brand-Element)
- `/confirm-email` Route bleibt außerhalb des Brand-Scopes
- Mail-Templates + PDFs — bewusste Out-of-Scope-Bestätigung
- Default-`brand_mode = 'preset'` bricht keine Bestand-EEG (alle
  rendern wie unter PROJ-102)
- PROJ-102 `presetStyleBlock` und `BRAND_PRESET_VARIABLES`-API
  bleiben — `mergeCustomTheme` ist ein zusätzlicher Layer obendrauf

### G) Dependencies

**Neu zu installieren:**
- `react-colorful` (~3 KB gz) — Color-Picker im Admin-Editor
- `wcag-contrast` (~1 KB) — Kontrast-Berechnungs-Helper im Admin-Editor

**Bestehende Bausteine wiederverwendet:**
- PROJ-102: Preset-Pipeline, `presetStyleBlock`, `BRAND_PRESET_VARIABLES`,
  `PublicPageShell`, `AdminBrandEditor`-Skelett, Settings-Mode-Check
- PROJ-84: `useDebouncedAutoSave`-Hook für Auto-Save von `brand_theme`
  und `brand_mode`
- PROJ-67: Settings-Sichtbarkeits-Modus + Awareness-Banner-Logik
- PROJ-81-Pattern: Configexport-Sanitizer + tolerant/strikt-Mix

### H) Risiken & Mitigationen

| Risiko | Mitigation |
|---|---|
| Bestand-EEGs erfahren nach Deploy plötzlich Custom-Modus | Migration setzt `brand_mode = 'preset'` als Default — Bestand-Verhalten 1:1 erhalten |
| HEX↔HSL-Rundungsdrift zwischen Go-Backend und TS-Frontend | Gemeinsamer Test-Vektor (50 HEX-Werte mit erwarteten HSL-Tripeln) für beide Helper |
| WCAG-Library liefert falsche Ratios | Wir wrappen `wcag-contrast` in einen eigenen `wcagAA`-Helper mit Snapshot-Test gegen bekannte Paare aus W3C-Doku |
| Color-Picker-Eingaben mit Lag bei vielen schnellen Bewegungen | `react-colorful` ist bewusst gewählt für Performance — Live-Vorschau debounced via 50 ms |
| Admin-Bundle wächst um ~4 KB gz | Akzeptabel; Code-Splitting hält Public-Page bei 0 KB Wachstum |
| Tab-Wechsel verwirft versehentlich Custom-Theme | Tab-Wechsel ändert nur lokalen Editor-State + brandMode; brand_theme bleibt im Parent-Snapshot persistiert. Test verifiziert: Tab nach Preset → Theme erhalten in DB |
| Configexport-Bundles mit zukünftigen Theme-Keys brechen Importer | Tolerant-bei-Keys-Regel droppt unbekannte Keys + loggt — Forward-Compat sichergestellt |
| WCAG-Gate ist zu streng, Admins werden frustriert | AA (nicht AAA) als Owner-festgenagelte Schwelle; UI zeigt klare Fix-Hints (z. B. „Akzent-Text dunkler wählen") |

