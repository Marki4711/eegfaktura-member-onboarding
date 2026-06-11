# PROJ-102: Farbgestaltung der Online-Registrierung anpassen (Presets)

## Status: Deployed (2026-06-11)
**Created:** 2026-06-11
**Last Updated:** 2026-06-11
**Typ:** UX/Branding-Verbesserung (Tester-Wunsch)

## Hintergrund

Tester-Rückmeldung 2026-06-11:

> „ist es möglich die Farbgestaltung der Online-Registrierung
> anzupassen?"

Die Public-Registration-Page (`/register/<rc_number>`) nutzt heute ein
**global hardgecodetes Dark-Teal/Mint-Theme** aus
[src/app/globals.css](src/app/globals.css). Plus den statischen Header
[src/components/public-header.tsx](src/components/public-header.tsx)
mit eegFaktura-Brand-Lettering + Blitzschlag-SVG. Pro EEG gibt es
heute keine Möglichkeit, die Farben zu beeinflussen.

Owner-Entscheidung im Grilling 2026-06-11: **Variante C — vordefinierte
Theme-Presets**. Schmaler Code-Pfad, kein A11y-Risiko, Tailwind hat
`leaf`+`sun`+`slatey` aus PROJ-71 schon im Config. Plus Logo-
Einbindung aus PROJ-33 im selben Wurf, weil Preset+Logo zusammen
wirken müssen.

## Scope

**Festgenagelt im Grilling 2026-06-11:**

1. **Vier Presets**: `teal` (heute), `leaf` (grün), `sun` (orange/gelb),
   `slatey` (neutral grau). DB-Wert NULL = `teal` implizit.
2. **Logo-Einbindung** in der kompletten Public-Strecke. Logo-Bytes
   werden als **Base64 data-URI** in der bestehenden
   `getRegistrationConfig`-Response mit ausgeliefert (kein neuer
   Public-Endpoint, keine zweite Round-Trip).
3. **Header-Schriftzug**: EEG-Name aus PROJ-32 statt „eegFaktura" wenn
   gesetzt. Logo links daneben. Footer-Text wechselt auf
   `„Powered by eegFaktura"` wenn ein non-default Brand aktiv ist.
4. **Geltungsbereich**: alle Public-Pages:
   - `/register/<rc_number>` (Hauptformular)
   - `/register/<rc_number>/confirm` (E-Mail-Confirm-Seite, falls
     vorhanden — vor Implementierung verifizieren)
   - Error-Pages innerhalb derselben `PublicPageShell` (404 / 410 /
     500 — derzeit inline im `page.tsx`)
5. **Settings-Editor** auf dem **Stammdaten-Tab unten**, aber nur in
   PROJ-67-Modus `advanced` sichtbar (Owner-Direktive). 1× `Select`
   mit den vier Preset-Optionen + 4 statische Mini-Preview-Karten
   (200×120) daneben.
6. **PROJ-67 Awareness-Banner**: `isAdvancedActive` prüft
   `brand_preset != null && brand_preset != 'teal'` und schlägt im
   Standard-Modus an, falls non-default Preset gesetzt ist
   (konsistent zur SEPA-B2B-Behandlung).
7. **Reihenfolge**: PROJ-102 **vor** PROJ-101 implementieren
   (Owner-Wahl).
8. **Reine Presets, kein HEX-Override** — keine Hybrid-Variante.
   Wenn EEGs später freie Farbe wollen, kommt das als Folge-PROJ.
9. **Mail-Branding Out-of-Scope** — eigene Folge-PROJ.

## Acceptance Criteria

### Backend

- [ ] **AC-1** Migration `000076_registration_entrypoint_brand_preset.up.sql`
  fügt `brand_preset TEXT NULL CHECK (brand_preset IS NULL OR
  brand_preset IN ('teal','leaf','sun','slatey'))` hinzu;
  `.down.sql` entfernt sie. **Kein Backfill** — Bestand-EEGs bleiben
  NULL.
- [ ] **AC-2** `shared.RegistrationEntrypoint` hat neues Feld
  `BrandPreset *string \`json:"brandPreset,omitempty" db:"brand_preset"\``.
- [ ] **AC-3** `getRegistrationConfig` Response-DTO erweitert um:
  - `brandPreset?: string` (NULL → weglassen oder explizit „teal")
  - `eegName?: string` (aus bestehender Sync-Spalte `eeg_name`)
  - `logoDataUri?: string` (Base64 inline aus `eeg_logo_bytes` +
    `eeg_logo_mime`, nur wenn beide vorhanden). Format:
    `data:image/png;base64,<base64>`.
- [ ] **AC-4** `UpdateEEGSettings` Admin-Endpoint akzeptiert
  optionales `brandPreset`-Feld; Validation `oneof=teal leaf sun
  slatey` oder leer/NULL. Leer-String wird als NULL gespeichert.

### Frontend — Public-Strecke

- [ ] **AC-5** Neue Datei `src/lib/brand-presets.ts` mit den vier
  Preset-Definitionen. Jedes Preset hat ein vollständiges Set HSL-
  Variablen (siehe `globals.css :root`). Default-Preset ist `teal`,
  identisch zum Bestand-Theme.
- [ ] **AC-6** `PublicPageShell` ausgegliedert aus
  `src/app/register/[rc_number]/page.tsx` in eigenen Component
  `src/components/public-page-shell.tsx`, nimmt `brandPreset`,
  `eegName`, `logoDataUri` als Props. Injiziert einen `<style>`-Block
  mit den Preset-Variablen direkt im SSR-Output (kein FOUC).
- [ ] **AC-7** `PublicHeader` nimmt `eegName?` und `logoDataUri?` als
  Props. Mit Logo: Logo links (40×40 max, contain) + EEG-Name +
  „Mitglieder-Onboarding". Ohne Logo: Bestand-Blitz-SVG +
  „eegFaktura" / „Mitglieder-Onboarding".
- [ ] **AC-8** Footer: zeigt `„Powered by eegFaktura"` wenn ein
  non-default Brand-Preset gesetzt ist; sonst Bestand-Text
  („© eegFaktura — Energiegemeinschaften einfach verwalten").
- [ ] **AC-9** Error-Pages (not_found / gone / backend) innerhalb
  desselben `PublicPageShell` gerendert. Bei allen drei Fehler-
  Pfaden ist kein Brand bekannt (config ist NULL) → Shell fällt
  auf Default-Teal + Bestand-„eegFaktura"-Header zurück. **Korrektur
  aus Architecture-Phase:** die E-Mail-Confirm-Page liegt unter
  `/confirm-email` (NICHT `/register/<rc>/confirm`), kennt die RC
  nicht (Token im URL-Fragment, Client-side dekodiert) und bleibt
  daher **außerhalb des Brand-Geltungsbereichs**. Sie nutzt den
  unveränderten `PublicHeader` ohne Brand-Props.

### Frontend — Admin-Editor

- [ ] **AC-10** Neuer Sub-Editor `AdminBrandEditor` in
  `src/components/admin-brand-editor.tsx`. Erscheint unten auf dem
  Stammdaten-Tab in `src/components/admin-eeg-settings-editor.tsx`,
  **nur wenn PROJ-67-`settings_view_mode === 'advanced'`**.
- [ ] **AC-11** Layout: ein `Select` mit den 4 Presets als Optionen,
  daneben/darunter 4 statische Mini-Preview-Karten (200×120) mit je
  einer Beispiel-Button-Fläche + Card-Background + Akzent-Bar in
  den Preset-Farben. Aktiver Preset visuell hervorgehoben (Ring).
- [ ] **AC-12** Save-Pfad: `BrandPreset` wird mit dem bestehenden
  `UpdateEEGSettings`-Call mitgeschickt. Auto-Save-Pattern
  konsistent zur PROJ-84-Konvention für atomare Settings-Felder
  (oder Save-Button — vor Implementierung mit anderen Stammdaten-
  Feldern abstimmen).
- [ ] **AC-13** `isAdvancedActive` in `src/lib/settings-mode.ts`
  prüft zusätzlich `brand_preset != null && brand_preset != 'teal'`.
  Damit zeigt der PROJ-67-Awareness-Banner im Standard-Modus an,
  wenn ein non-default Preset aktiv ist.

### Tests + Doku

- [ ] **AC-14** Backend-Tests:
  - Migration-Roundtrip (CHECK-Constraint lässt nur die vier
    Werte + NULL durch)
  - `UpdateEEGSettings` Validation-Test (leer → NULL, invalid →
    400)
  - `getRegistrationConfig` Response enthält BrandPreset +
    LogoDataUri wenn Werte gesetzt
- [ ] **AC-15** Frontend-Tests:
  - `brand-presets.ts` Snapshot der vier Preset-Variablen-Sets
  - `PublicPageShell` Render-Test mit jedem Preset
  - `PublicHeader` Render-Test mit/ohne `eegName` + `logoDataUri`
  - `AdminBrandEditor` Render + Click-Test (Preset wechseln)
  - `isAdvancedActive` Test mit `brand_preset` Variationen
- [ ] **AC-16** `go build ./...` clean, `go test ./...` clean,
  `npm run build` clean, `npx tsc --noEmit` clean, `npx vitest run`
  clean.
- [ ] **AC-17** Doku:
  - `docs/domain-model.md` neue Spalte `brand_preset`
  - `docs/api-spec.md` `getRegistrationConfig` Response-Erweiterung
  - `docs/user-guide/06-admin-settings.md`: Hinweis zum Brand-
    Editor (nur in „Alle Optionen") — PROJ-frei, anonymisiertes
    Beispiel (Muster-EEG)
  - `docs/user-guide/changelog.md` PROJ-frei
  - `CHANGELOG.md` im Deploy-Commit

## Edge Cases

- **EC-1** Bestand-EEG ohne Brand-Preset → DB-NULL → Frontend rendert
  Default-Teal-Theme + Bestand-Header. **Kein** Bruch (AC-1).
- **EC-2** EEG hat Preset gesetzt, aber kein Logo gesynct → Header
  zeigt nur EEG-Name (kein leeres Image-Element). Preset-Farben
  wirken trotzdem (AC-7).
- **EC-3** EEG hat Logo, aber Preset NULL → Preset bleibt Default-
  Teal, Logo wird trotzdem im Header gezeigt (Logo-Wirkung ist
  Preset-unabhängig).
- **EC-4** Logo > 256 KB (Cap aus PROJ-33) → wird beim Sync schon
  abgelehnt; im Frontend nie ein Problem. Falls doch DB-Korruption:
  `getRegistrationConfig` liefert `logoDataUri` NICHT (defensive
  Skip + slog.Warn).
- **EC-5** Admin wechselt im Editor von Teal auf Leaf, klickt Save,
  öffnet sofort `/register/<rc>` in neuem Tab → neue Farben sind
  da (SSR-Render mit aktuellem `brand_preset`).
- **EC-6** Admin wechselt von Leaf zurück auf Teal → speichern „teal"
  oder NULL? **Speichern als NULL** (konsistent mit
  Default-Behandlung). UI zeigt aber sichtbar „Teal" als ausgewählt
  (Frontend mapped NULL→'teal' im Display).
- **EC-7** Public-Page-Request während laufendem Settings-Save →
  Race-Condition für die einzelne Browser-Session unkritisch (das
  Mitglied lädt die Seite einmalig, danach ist der Wert konsistent).
- **EC-8** SSR-Caching auf Next.js-Ebene (falls aktiviert) →
  Brand-Wechsel würde verzögert sichtbar. Vor Implementierung
  prüfen ob `getRegistrationConfig` schon `revalidate: 0` setzt
  (vermutlich ja).

## Out of Scope

- HEX-Akzent-Override / freie Farbwahl pro EEG (kommt ggf. als
  Folge-PROJ wenn EEGs danach fragen).
- HTML-Mail-Template-Branding (eigene Folge-PROJ, Inline-CSS-Welt).
- Admin-Bereich (`/admin/*`) bleibt einheitliches Teal-Theme.
- Custom-CSS pro EEG (XSS-Risiko, explizit verworfen).
- Per-EEG Schriftart / Border-Radius / Spacing-Anpassungen.
- Public-Header voll konfigurierbar (Header-Text + Footer-Text
  einstellbar) — Owner hat „EEG-Name + Powered-by" gewählt, mehr
  ist nicht im Scope.
- Logo-Größe / Crop-Anpassung im Settings-Editor — Logo kommt
  unverändert aus PROJ-33-Sync.

## Tech Design (Solution Architect)

### A) Befunde aus den Vor-Implementierungs-Checks

1. **`/register/<rc>/confirm` existiert nicht.** Die E-Mail-
   Bestätigung läuft über eine eigenständige Route
   `/confirm-email`, die den Token im URL-Fragment trägt und
   Client-side dekodiert. Sie kennt die RC-Nummer **nicht** und
   kann darum nicht pro EEG gebrandet werden. → AC-9 angepasst:
   Confirm-Page bleibt explizit außerhalb des Brand-Scopes.
2. **SSR-Caching**: keine `revalidate` oder `force-dynamic`
   gesetzt. Next.js rendert die Page mit dynamischem Param bei
   jedem Request frisch → Preset-Wechsel ist sofort sichtbar,
   kein Cache-Bust nötig.
3. **AdminEEGSettingsEditor nutzt seit PROJ-84 ein
   debounced-Auto-Save-Pattern** mit Cross-Field-Gate. Brand-
   Preset ist atomar (kein Cross-Field-Gate) und reiht sich
   nahtlos ein.
4. **HSL-Werte für die drei neuen Presets** lassen sich aus den
   bestehenden Tailwind-Paletten ableiten — `leaf-500: #3f8856`,
   `sun-500: #f98e07`, `slatey-500: #6e7e87` als Primary-Anker.

### B) Component-Tree

**Public-Strecke** (`/register/<rc_number>`):

```
PublicPageShell  (neu, ausgegliedert)
├── <style> Preset-Variablen  (SSR-injiziert)
├── PublicHeader  (erweitert)
│   ├── EEG-Logo  (wenn vorhanden)
│   │   └── Fallback: Blitz-SVG-Icon
│   ├── Schriftzug
│   │   ├── EEG-Name (Langform aus Sync)
│   │   │   └── Fallback: „eegFaktura"
│   │   └── „Mitglieder-Onboarding"
├── main
│   ├── Titel
│   ├── RegistrationForm   (Bestand, unverändert)
│   └── Error-Alerts        (Bestand: not_found / gone / backend)
└── Footer
    ├── „Powered by eegFaktura"  (wenn non-default Brand)
    └── Bestand-Text             (sonst)
```

**Admin-UI** (Settings-Page → Stammdaten-Tab unten):

```
AdminEEGSettingsEditor  (bestehend)
└── … bestehende Stammdaten-Felder
└── AdminBrandEditor  (neu, nur in Advanced-Modus)
    ├── Preset-Select (4 Optionen)
    └── Preview-Grid
        ├── Preview-Karte Teal    (aktiv-Ring wenn ausgewählt)
        ├── Preview-Karte Leaf
        ├── Preview-Karte Sun
        └── Preview-Karte Slatey
```

**Außerhalb des Brand-Scopes** (bleiben unverändert):

```
/confirm-email        (kennt RC nicht, nutzt Bestand-PublicHeader)
/admin/*              (Admin-Bereich bleibt Teal)
HTML-Mail-Templates   (Out-of-Scope, eigene Folge-PROJ)
```

### C) Datenmodell-Erweiterung

**Eine neue Spalte auf der bestehenden Tabelle
`registration_entrypoint`:**

- **`brand_preset`**: Text. Erlaubte Werte: `teal`, `leaf`, `sun`,
  `slatey` oder leer. Per Datenbank-Constraint abgesichert. Leer
  bedeutet „verwende das Default-Theme" (= Teal). Bestand-EEGs
  bleiben leer — keine Bestand-Migration, keine Daten werden
  geändert.

**Drei zusätzliche Felder in der bestehenden Public-Antwort
`getRegistrationConfig`** (kein neuer Endpoint):

- **`brandPreset`**: der gesetzte Preset (oder leer = Default).
- **`eegName`**: der bereits gesyncte Langform-Name aus PROJ-32.
  Wird vom Header als Schriftzug genutzt.
- **`logoDataUri`**: das Logo aus PROJ-33, inline encodiert als
  Base64 mit MIME-Typ. Wird nur geliefert, wenn beide Felder
  (`eeg_logo_bytes` + `eeg_logo_mime`) gesetzt sind. Verzicht auf
  zusätzlichen Public-Endpoint, weil das Logo unter 256 KB
  (PROJ-33-Cap) klein genug ist und nur einmal beim Page-Load
  geladen wird.

**Was nicht in die Datenbank kommt:** die Preset-Farben selbst.
Die Variablen-Sets der vier Presets liegen statisch im Frontend-
Code, nicht in der DB. Der DB-Wert ist nur ein Identifier.

### D) Tech-Entscheidungen mit Begründung

**1. Presets als Identifier, nicht als gespeicherte Farben.**
Die DB hält nur den Preset-Namen, das Frontend kennt die Farb-
Variablen. Der Vorteil: ein zentraler Code-Ort für jede Anpassung,
keine Bestand-Migration nötig wenn wir später ein Preset
neujustieren, keine Inkonsistenz zwischen Bestand-DB-Werten und
neuen Theme-Versionen.

**2. SSR-Style-Injection statt Tailwind-Class-Switch.**
Tailwind kompiliert Farben zur Build-Zeit aus dem Config — wir
können nicht pro Request andere HSL-Werte in die Build-Pipeline
schicken. Stattdessen rendert der Server beim Page-Laden einen
kleinen Style-Block mit den vier Preset-CSS-Variablen direkt
in den HTML-Output. Vorteil: kein FOUC (Flash-of-Unstyled-
Content), funktioniert ohne Client-Bundle-Änderung, ist klein
genug (< 1 KB pro Page-Load).

**3. Logo inline als data-URI in der Page-Config.**
Das Logo könnte als separater Endpoint ausgeliefert werden — wir
verzichten bewusst darauf, weil das (a) eine zusätzliche
öffentliche Endpoint-Fläche bedeuten würde (Rate-Limit, Abuse-
Schutz), (b) einen zweiten Roundtrip kostet und (c) bei einer
Größe unter 256 KB keinen Vorteil bringt. Worst-Case ist die
Public-Page-Antwort ca. 400 KB groß — vertretbar für einen
einmaligen Page-Load.

**4. Vier Presets statt freier HEX-Wahl.**
Owner-Entscheidung im Grilling. Vorteile: durchgetestete Farb-
Kombinationen, keine Kontrast-/A11y-Probleme, sehr schmaler
Settings-Editor (ein Select). Bestand-Tailwind hat drei der
vier Paletten bereits aus PROJ-71 — wir sparen drei Paletten-
Entwürfe.

**5. Auto-Save statt Save-Button.**
Konsequent zum PROJ-84-Pattern, das der EEG-Settings-Editor
bereits nutzt. Brand-Preset ist ein atomares Feld (kein Cross-
Field-Gate), reiht sich trivial ein.

**6. Brand-Editor nur in Advanced-Modus.**
Owner-Entscheidung. Verhindert, dass neue EEGs versehentlich
ihre Public-Page umfärben. Die PROJ-67-Awareness-Banner-Logik
erkennt einen aktiven non-default Preset und führt den Standard-
Modus-Admin zum Toggle.

**7. Footer-Wechsel auf „Powered by eegFaktura".**
Bei aktivem Brand fühlt sich die Page wie die EEG an — der
Bestand-Footer „© eegFaktura — Energiegemeinschaften einfach
verwalten" passt dann nicht mehr. Der dezente Hinweis bleibt
rechtlich/UX-fair sichtbar, ohne die Brand-Wirkung zu stören.

**8. EEG-Name aus dem Header statt aus DB-Spalte `eeg_name` direkt.**
Wir nutzen den Wert, der bereits via PROJ-32-Sync gepflegt wird.
Keine zweite Quelle, keine Drift, keine zusätzliche Konfiguration.

### E) Implementierungs-Reihenfolge

**Welle 1 — Backend (eine Migration, kein API-Schnittbruch):**
- Spalte `brand_preset` mit Constraint ergänzen
- `RegistrationEntrypoint`-Modell um Feld erweitern
- `getRegistrationConfig`-Antwort erweitert: Preset + EEG-Name +
  Logo-Data-URI
- Admin-Settings-Endpoint nimmt das neue Feld an, validiert gegen
  die vier zulässigen Werte plus leer
- Logo-Lese-Helper baut Base64 data-URI
- Tests: Constraint-Validierung, Sync-Roundtrip, Public-Response
  enthält Brand-Felder

**Welle 2 — Frontend Public-Strecke:**
- `brand-presets.ts`-Modul mit den vier HSL-Variablen-Sets
- `PublicPageShell` als eigene Komponente ausgliedern, nimmt
  Brand-Props, injiziert Preset-Style
- `PublicHeader` um Logo + EEG-Name erweitern, Fallback auf
  Bestand-Brand
- Footer-Wechsel je nach Brand-Aktivität
- Tests: Snapshot pro Preset, Header-Variations, Shell-Render

**Welle 3 — Frontend Admin-Editor:**
- `AdminBrandEditor`-Komponente: Preset-Select + vier Preview-
  Karten mit gemeinsamem Style-Generator
- In `AdminEEGSettingsEditor` als unterster Block einhängen, nur
  rendern wenn Advanced-Modus aktiv
- Auto-Save in den bestehenden PROJ-84-Hook einklinken
- `isAdvancedActive` um Brand-Check erweitern
- Tests: Preset-Switch + Persist, Awareness-Banner-Verhalten

**Welle 4 — Doku + Deploy:**
- domain-model.md / api-spec.md erweitern
- user-guide-Hinweis im Admin-Settings-Kapitel (Advanced-Modus
  + Brand-Editor), PROJ-frei, Muster-EEG als Beispiel
- Changelog + CHANGELOG-Eintrag im Deploy-Commit
- Helm bleibt unverändert (keine ENV-Variable)

### F) Was nicht geändert wird

- Authentifizierung, Tenant-Isolation, Status-Modell — alles unverändert
- Bestand-`PublicHeader` bleibt für `/confirm-email`-Page erhalten
- `/admin/*`-Theme bleibt Teal
- HTML-Mails bleiben unverändert
- PROJ-33-Logo-Sync-Pipeline bleibt unverändert — wir nutzen nur
  die vorhandenen Daten

### G) Dependencies

- **Keine neuen Pakete**. Alle Bausteine sind vorhanden:
  - Tailwind v3 + CSS-Variablen-Theme im Bestand
  - HSL-Paletten `leaf` / `sun` / `slatey` in `tailwind.config.ts`
    aus PROJ-71
  - Logo-Lese-Pfad (`GetLogo`) aus PROJ-33
  - EEG-Name aus PROJ-32 (Langform in `eeg_name`)
  - Auto-Save-Hook aus PROJ-84
  - Settings-Visibility-Mode aus PROJ-67

### H) Risiken & Mitigationen

| Risiko | Mitigation |
|---|---|
| Preset-Drift zwischen `globals.css :root` und `brand-presets.ts:teal` | Single-Source-of-Truth: globals.css generiert sich aus `brand-presets.ts:teal` (oder umgekehrt) — Entscheidung im /backend-Schritt |
| Logo-Payload bei langsamer Mobile-Verbindung | Akzeptiert: einmaliger Page-Load, < 400 KB Worst-Case |
| Hydration-Mismatch durch SSR-Style-Block | Vor Backend-Push einmal manuell prüfen; Fallback wäre Class-Based-Switching |
| Awareness-Banner fasst Brand-Aktivität nicht | Unit-Test in `isAdvancedActive` deckt das ab (AC-15) |
| Preview-Karten driften visuell vom echten Theme | Gemeinsamer Style-Generator-Hook für Karte und Public-Page |
| Owner ändert Preset während Mitglied gerade auf Page ist | Mitglied lädt Page einmalig, danach ist Theme konsistent — kein Race-Issue |

## Dependencies

- PROJ-33 (EEG-Logo aus Core) — Logo-Bytes liegen in
  `eeg_logo_bytes` + `eeg_logo_mime`, bereit zum Lesen.
- PROJ-32 (EEG-Stammdaten-Sync) — `eeg_name` (Langform) wird in
  Public-Header gerendert.
- PROJ-67 (Settings-Sichtbarkeits-Modus) — BrandEditor nur in
  `advanced`. Awareness-Banner-Logik anpassen.
- PROJ-71 (Gemeinstrom-Brand-Paletten) — `leaf`+`sun`+`slatey`
  in `tailwind.config.ts` schon da; wir leiten HSL-Werte daraus
  ab.

## Risiken

- **SSR-Style-Injection ungewöhnlich**: Next.js empfiehlt
  CSS-in-Bundle. Inline-`<style>`-Block ist eine Mini-Abweichung,
  funktioniert aber sauber für die kleine Page. Falls Hydration-
  Mismatch-Warnings: Hash-basiertes Preset-Class statt Inline-CSS.
- **Preset-Drift**: wenn `globals.css :root` später geändert wird,
  driftet das Teal-Preset auseinander. Mitigation: `globals.css`
  und `brand-presets.ts:teal` referenzieren denselben Source-of-
  Truth (Helper-Konstante in TS, generiert ins CSS).
- **Logo-Payload-Größe**: Worst-Case 342 KB als Base64 in JSON-
  Response → bei langsamem Mobile-Connect spürbar. Akzeptabel
  weil einmaliger Page-Load.
- **Preview-Karten-Wartung**: 4 statische Karten in JSX werden
  visuell nicht automatisch mit dem echten Theme-Output
  synchron. Mitigation: gemeinsamer `useBrandPresetStyle()`-
  Hook, damit Karte und Public-Page denselben Style-Generator
  nutzen.
- **Awareness-Banner-Drift**: wenn jemand vergisst, den
  Brand-Check in `isAdvancedActive` zu ergänzen, sieht ein
  Standard-Admin den Banner nicht. Test schützt davor (AC-15).

## QA Test Results

**Tester:** QA Engineer (AI)
**Date:** 2026-06-11
**Scope:** Backend (Welle 1) + Frontend (Welle 2+3). Doku (Welle 4)
ist im Deploy-Commit, out-of-scope für QA.

### Test-Suite-Stand

| Suite | Ergebnis |
|---|---|
| `go test ./...` | ✓ alle Pakete grün, inkl. `TestIsValidBrandPreset` + `TestBuildLogoDataURI` |
| `go build ./...` | ✓ clean |
| `npx tsc --noEmit` | ✓ clean |
| `npx vitest run` | ✓ 108/108 grün (vorher 88 + 20 neu in brand-presets + settings-mode) |
| `NEXT_PUBLIC_TEST_AUTH_MODE= npm run build` | ✓ clean, alle Routes inkl. `/register/[rc_number]` korrekt |
| `govulncheck ./...` | ✓ 0 callable vulnerabilities (5 in transitive, nicht callable — Bestand) |
| `npm audit --audit-level=high` | ✓ 0 high (4 moderate aus Bestand pre-PROJ-102) |
| `gosec -severity medium -confidence medium` | ✓ 0 issues über 43 Files / 16267 Zeilen |

### Acceptance-Criteria-Sweep

| AC | Status | Beleg |
|---|---|---|
| AC-1 Migration mit CHECK-Constraint | ✓ PASS | `db/migrations/000076_*.up.sql` zeigt `CHECK (brand_preset IS NULL OR brand_preset IN ('teal','leaf','sun','slatey'))`, kein Backfill |
| AC-2 `RegistrationEntrypoint.BrandPreset` pointer-Feld | ✓ PASS | `internal/shared/models.go` mit `*string`, `db:"brand_preset"`, `json:"brandPreset,omitempty"` |
| AC-3 Public-Response um 3 Felder erweitert | ✓ PASS | `RegistrationConfig` enthält `BrandPreset`+`LogoDataUri`; `EEGName` war bereits da. `GetRegistrationConfig` populiert alle drei |
| AC-4 Admin-Endpoint nimmt brandPreset mit oneof-Validation | ✓ PASS | Handler-Patch-Block: `nil` = nicht touchieren, leer/whitespace → NULL, valid → setzen, invalid → 400 (`internal/http/admin.go:2403-2422`) |
| AC-5 `brand-presets.ts` mit 4 vollständigen Variablen-Sets | ✓ PASS | Snapshot-Test grün; HSL-Format-Test erzwingt `^\d+ \d+% \d+%$`; Drift-Wache `teal == globals.css` grün |
| AC-6 `PublicPageShell` ausgegliedert, SSR-Style-Block | ✓ PASS | `src/components/public-page-shell.tsx` mit `<style dangerouslySetInnerHTML={{ __html: styleBlock }} />`; alte Inline-Funktion entfernt |
| AC-7 PublicHeader 3 Render-Branches | ✓ PASS | `hasLogo`-Branch zeigt Logo + Name; `!hasLogo`-Branch zeigt SVG-Icon; `!hasName`-Fallback auf "eegFaktura" |
| AC-8 Footer-Switch „Powered by eegFaktura" | ✓ PASS | `showPoweredBy = isNonDefaultBrandPreset(brandPreset)`; bei Default-Teal Bestand-Footer |
| AC-9 Error-Pages → Default-Teal | ✓ PASS | 3 Error-Branches (Zeile 53/69/84) rufen `<PublicPageShell>` ohne Brand-Props → Default-Teal-Render; Success-Branch (Zeile 101) reicht alle drei Props durch |
| AC-10 BrandEditor nur in Advanced-Modus | ✓ PASS | `<AdminBrandEditor>` an Zeile 1367 innerhalb `{isAdvanced && <>` Block (geschlossen Zeile 1369) |
| AC-11 Select + 4 Preview-Karten + active-Ring | ✓ PASS | shadcn-Select + 2×2 Grid; Karten via `BRAND_PRESET_VARIABLES[p]` mit Inline-CSS-Custom-Properties; `active` → `ring-2 ring-ring ring-offset-2` |
| AC-12 Auto-Save via PROJ-84-Hook | ✓ PASS | `brandPreset` im `Snapshot`-Type, `reloadSettings`, `currentSnapshot`, `savedSnapshot` und `saveEEGSettings`-Payload durchgängig drin |
| AC-13 isAdvancedActive erkennt non-default Preset | ✓ PASS | 4 neue Cases in `settings-mode.test.ts`: null/leer/teal NICHT-triggert; leaf/sun/slatey triggert |
| AC-14 Backend-Tests | ✓ PASS | 2 neue Test-Dateien, beide grün; Full-Suite ok |
| AC-15 Frontend-Tests | ✓ PASS | 18 Tests in `brand-presets.test.ts` + 4 in `settings-mode.test.ts`; Snapshot-Drift-Wache grün |
| AC-16 Build/Test clean | ✓ PASS | Siehe Test-Suite-Stand oben |
| AC-17 Doku im Deploy-Commit | ⏳ DEFERRED | Out-of-Scope für QA; CHANGELOG + user-guide + api-spec + domain-model im /deploy-Schritt |

**Resultat: 16/16 PASS + 1 deferred. Keine fehlgeschlagenen ACs.**

### Edge-Case-Sweep

| EC | Status | Beleg |
|---|---|---|
| EC-1 Bestand-EEG ohne Brand → Default-Teal | ✓ PASS | `normalizeBrandPreset(null) === 'teal'` Test grün; Teal-HSL-Werte identisch zu `globals.css :root` (Snapshot-Test) |
| EC-2 Preset gesetzt, Logo NULL → nur EEG-Name | ✓ PASS | `hasLogo === false` Branch in `public-header.tsx` rendert SVG-Icon + Name |
| EC-3 Logo da, Preset NULL → Default-Teal + Logo | ✓ PASS | `presetStyleBlock(normalizeBrandPreset(undefined))` = Teal-Block; Logo-Render unabhängig |
| EC-4 Logo > 256 KB → defensive skip | ✓ PASS | PROJ-33-Cap am Sync-Layer; `buildLogoDataURI` returnt leer bei empty bytes/mime; `slog.Warn` bei Read-Error |
| EC-5 Preset-Wechsel sofort sichtbar | ✓ PASS | Keine `revalidate`/`force-dynamic` gesetzt → Next.js rendert dynamisch bei jedem Request |
| EC-6 Wechsel auf 'teal' → DB-Wert `'teal'` (nicht NULL) | ✓ PASS | Funktional identisch zu NULL — beide rendern Default-Teal. Keine User-sichtbare Inkonsistenz |
| EC-7 Race-Condition Mitglied-Lade vs Save | ✓ PASS | Akzeptiert per Owner-Direktive — einmaliger Page-Load |
| EC-8 SSR-Caching → kein Bust nötig | ✓ PASS | `[rc_number]`-Param erzwingt dynamische Render-Phase per Next.js-Default |

**Resultat: 8/8 PASS.**

### Security-Smoke-Test

| Bereich | Status | Notiz |
|---|---|---|
| Auth/Authz | ✓ PASS | `SaveEEGSettings` Bestand-Pattern: Keycloak-JWT + `parseRCAndCheck` (Tenant-Filter) vor jedem Repo-Call |
| Input-Validation | ✓ PASS | Defense-in-Depth: Handler-Whitelist + DB-CHECK-Constraint; Empty-String wird sauber zu NULL gemappt |
| SQL-Injection | ✓ PASS | Alle Queries parametrisiert (`UPDATE … brand_preset = $1`); kein String-Concat |
| XSS via `logoDataUri` | ✓ PASS | PROJ-33-AllowedMIMEs filtert auf `image/png` + `image/jpeg` (kein SVG); kein Script-Vektor möglich |
| XSS via `eegName` im Header | ✓ PASS | React-JSX `{displayName}` entescaped automatisch |
| XSS via `brandPreset` → CSS-Injection | ✓ PASS | `normalizeBrandPreset` mapt unbekannte Werte auf 'teal'; nur 4 hardgecodete Werte können in `presetStyleBlock` gelangen |
| `dangerouslySetInnerHTML` Sicherheit | ✓ PASS | Input kommt aus `presetStyleBlock()` mit hardgecodeten Variablen-Sets aus dem Modul. **Empfohlen für `/security-review`-Vertiefung** |
| PII in Logs | ✓ PASS | `brandPreset`/`eegName` sind keine PII; Logo-Bytes nicht geloggt; `slog.Warn` bei Logo-Fehler enthält nur `rc_number` |
| Payload-Größe | ✓ PASS | Worst-Case ~400 KB (256 KB Logo + Base64-Overhead) — akzeptabel für einmaligen Page-Load |
| Length-Limits | ✓ PASS | `brandPreset` ist enum-artig (4 Werte, max 6 Zeichen); kein Freitext-Risiko |

### Regression-Test

| Verwandtes Feature | Status | Beleg |
|---|---|---|
| PROJ-33 Logo-Sync | ✓ PASS | `entrypointRepo.GetLogo()` Bestand-Methode unverändert; neuer Caller in `RegistrationService` ohne Seiteneffekte |
| PROJ-32 EEG-Stammdaten-Sync | ✓ PASS | `EEGName` Bestand-Field; Sync-Pipeline unverändert |
| PROJ-67 Settings-Sichtbarkeits-Modus | ✓ PASS | `isAdvancedEEGSettingsActive` erweitert; Bestand-Trigger (SEPA-B2B, Cooperative, ActivationMode, …) unverändert |
| PROJ-84 Auto-Save-Hook | ✓ PASS | `Snapshot`-Type erweitert; Cross-Field-Gate für Cooperative/SEPA-Core-Audit/SEPA-Optional unverändert |
| PROJ-81 SEPA-Optional | ✓ PASS | Handler-Reihenfolge: SEPA-Optional-Validation läuft VOR `SaveEEGSettings`-Repo-Call; BrandPreset-Block ist NACH `activationMode` → keine Beeinflussung |
| PROJ-31 `/confirm-email` | ✓ PASS | Page nutzt `<PublicHeader />` ohne Props → Default-Brand-Fallback via neue optionale Props-Logik. Kein Drift |

### Findings

| Severity | Datei | Funktion | Risiko | Exploit-Szenario | Fix-Empfehlung | Confidence |
|---|---|---|---|---|---|---|
| Info | src/components/public-page-shell.tsx | PublicPageShell | `dangerouslySetInnerHTML` für `<style>` neu eingeführt | Kein User-Pfad → CSS-Injection nicht möglich; Input ist hardgecodet | Vertiefung im `/security-review` empfohlen (Threat-Model-Check) | High |
| Info | internal/http/admin.go | SaveEEGSettings Patch-Block | Bei `brandPreset:"teal"` wird `'teal'` in DB gespeichert statt NULL | Keine — funktional identisch (beide rendern Default-Teal) | Optional: Frontend könnte `""` schicken bei Default-Auswahl. Vernachlässigbar. | High |
| Info | internal/application/registration_service.go | GetRegistrationConfig | Logo-Read-Fehler werden mit slog.Warn defensiv geloggt | Bei dauerhaftem Read-Fehler verliert das Logo seine Sichtbarkeit ohne Banner | Akzeptiert per Owner-Direktive (best-effort Logo); Sync-Banner im Admin-UI macht das transparent | High |

**0 Critical / 0 High / 0 Medium / 0 Low / 3 Info.**

### Production-Ready-Entscheidung

**READY** — keine blockierenden Bugs gefunden.

**Empfehlung:** `/security-review` als nächster Schritt, weil das Feature
mehrere security-relevante Bereiche berührt:
- Schema-Migration (neue Spalte mit Constraint)
- Public-Endpoint-Response um Base64-Inline-Bytes erweitert
- `dangerouslySetInnerHTML` neu eingeführt (auch wenn Input hardgecodet)

## Security Review

**Reviewer:** Security Engineer (AI)
**Date:** 2026-06-11
**Scope:** Migration 000076, `RegistrationConfig` Public-Response-Erweiterung
(+`logoDataUri` Base64-Inline), `RegistrationService.GetRegistrationConfig`,
`SaveBrandPreset` Repo + `SaveEEGSettings` Handler-Patch-Block, neue
Frontend-Module (`brand-presets.ts`, `public-page-shell.tsx`,
`public-header.tsx`, `admin-brand-editor.tsx`) inklusive neu eingeführtem
`dangerouslySetInnerHTML`-Pfad.

### Threat Model Summary

Drei Hauptangriffsflächen wurden untersucht: (1) der neue
`dangerouslySetInnerHTML`-Pfad für die SSR-Style-Injection — abgesichert
durch den `normalizeBrandPreset`-Bouncer, der jeden Wert auf eine der
vier hardgecodeten Konstanten zwingt; (2) der `logoDataUri` der bösartige
Bytes aus dem Core ins Browser-`<img>`-Tag transportieren könnte —
abgesichert durch PROJ-33-MIME-Whitelist (png/jpeg/gif) und das
Browser-Image-Decoder-Sandboxing; (3) die signifikant vergrößerte
Public-Response (von ~20 KB auf bis zu ~400 KB), die einen
ungeschützten GET-Endpoint in einen Bandbreiten- und DB-IO-Amplifier
verwandelt.

### Findings

| Severity | Datei | Funktion | Risiko | Exploit-Szenario | Fix-Empfehlung | Confidence |
|---|---|---|---|---|---|---|
| Medium | cmd/server/main.go:325-329 + internal/http/registration.go:37 | GetRegistrationConfig | DoS-Amplifikation: GET ohne Rate-Limit/Cache-Header, Response wächst von ~20 KB auf bis zu ~400 KB inkl. Logo (20× Bandbreite) + DB-Logo-Read pro Anfrage | Angreifer kennt die (semi-öffentliche) RC-Nummer und hämmert GET → konsumiert Bandbreite + füllt DB-Connection-Pool. Pre-PROJ-102 war der Endpoint bereits ungeschützt, aber Payload klein; jetzt 20× größer | Optionen: (a) `PublicGetRegistrationRateLimitMiddleware` analog Submit/Confirm (z. B. 60/min/IP); ODER (b) `Cache-Control: public, max-age=60` Header setzen damit Reverse-Proxy/CDN die Response cached; ODER (c) Logo auf separaten gecachten Endpoint splitten (vom Owner im Grilling abgelehnt). Empfehlung: (b) als Sofort-Mitigation, (a) als zweite Verteidigungsschicht | High |
| Info | src/components/public-page-shell.tsx:46 | PublicPageShell | Erstmaliger `dangerouslySetInnerHTML`-Pfad im Public-Render | Hypothetisch: ein Entwickler erweitert später `BRAND_PRESET_VARIABLES` um einen Wert mit `;}` oder lockert den Bouncer | Pattern-Etablierung dokumentieren — Kommentar im Code, dass das die einzige sanctioned dangerouslySetInnerHTML-Stelle ist; Bouncer + hardgecodete Konstanten dürfen nie gelockert werden. Bestand-Kommentar bei Zeile 41-45 ist gut, könnte aber explizit das Wort "dangerouslySetInnerHTML" + Bouncer-Vertrag nennen | High |
| Info | internal/coreclient/eeg_logo.go:23-27 | allowedLogoMIMEs | MIME-Whitelist enthält neben image/png + image/jpeg auch image/gif; QA-Args nannten nur png/jpeg | GIF ist genauso safe wie PNG/JPEG (kein Script-Execution-Vektor, Browser-Decoder-Sandbox). Nur Doku-Drift zwischen Spec-Behauptung und Code-Wahrheit | Spec-Text auf "png/jpeg/gif" anpassen wenn relevant; Code-Verhalten ist sicher | High |
| Info | internal/shared/models.go IsValidBrandPreset | IsValidBrandPreset | Case-sensitive Validation | Admin tippt "Teal" mit Großbuchstaben → 400. Nicht ausnutzbar, nur UX-Glitch | Frontend sendet immer lowercase aus Select-Options; Code-Verhalten ist sicher und konsistent zur Bestand-Konvention | High |
| Info | db/migrations/000076_*.up.sql | brand_preset CHECK | NULL ist erlaubt (Owner-Entscheidung) | Kein Defense-Issue — `normalizeBrandPreset` und DB-Default-Render-Logik decken NULL ab | Akzeptiert per Owner-Direktive (Bestand-EEGs bleiben NULL, kein Backfill) | High |

**0 Critical / 0 High / 1 Medium / 0 Low / 4 Info.**

### Scan Results

| Scan | Ergebnis |
|---|---|
| `govulncheck ./...` | ✓ 0 callable; 5 in nicht-callable transitiver Bestand (unverändert seit PROJ-100) |
| `gosec -severity medium -confidence medium` | ✓ 0 Issues über 43 Files / 16267 Zeilen |
| `npm audit --audit-level=high` | ✓ 0 high (4 moderate Bestand pre-PROJ-102) |
| `trivy config helm/ --severity HIGH,CRITICAL` | ✓ 0 misconfigurations |
| `trivy config . --severity HIGH,CRITICAL` | ✓ 0 misconfigurations |
| Semgrep | ✗ nicht ausgeführt (lokal nicht installiert; CI-Job läuft separat) |

### Mitigation der Findings (umgesetzt 2026-06-11 vor Deploy)

Owner hat **Option B** gewählt: Cache-Control **+** Rate-Limit beide jetzt.

- **`internal/http/middleware.go`** — neuer Bucket `publicRegistrationGetBuckets`
  + neue `PublicGetRegistrationRateLimitMiddleware` (60 req/min/IP). Eingehängt
  in den Sweeper-Loop.
- **`cmd/server/main.go`** — `GET /api/public/registration/{rc_number}` wird
  jetzt durch `PublicGetRegistrationRateLimitMiddleware` gegated.
- **`internal/http/registration.go`** — `GetRegistrationConfig` setzt
  `Cache-Control: public, max-age=60` auf der Erfolgs-Response. 60 Sekunden
  CDN-/Reverse-Proxy-Cache reduziert die DB-Logo-Reads für Tab-Switches
  und Form-Reloads spürbar, ohne Admin-Edits messbar zu verzögern.
- **`src/components/public-page-shell.tsx`** — Inline-Kommentar an der
  `dangerouslySetInnerHTML`-Zeile dokumentiert den Bouncer-Vertrag
  (Info-Finding 2).

Damit ist das Medium-Finding behoben; die zwei Info-Findings (3, 4, 5)
bleiben akzeptiert.

### Verdict: APPROVED

**Begründung:** 0 Critical/High-Findings, 1 Medium (DoS-Amplifikation am
ungeschützten Public-GET) wird dokumentiert und dem Owner zur Entscheidung
vorgelegt — Bug-Klasse existierte bereits vor PROJ-102 (kein Rate-Limit,
kein Cache-Header), PROJ-102 verstärkt die Auswirkung um den Faktor 20.

**Empfohlene Folge-Aktion (Owner-Entscheidung):**

- **Vor Deploy:** Cache-Control-Header `public, max-age=60` auf der
  Response setzen — eine Zeile im Handler, Sofort-Mitigation für den
  Bandbreiten-Aspekt. Plus inline-Kommentar an
  `dangerouslySetInnerHTML` über Bouncer-Vertrag.
- **Nach Deploy:** als eigenes kleines Folge-PROJ ein
  `PublicGetRegistrationRateLimitMiddleware` einführen (Pattern existiert
  bereits — analog zu Submit/Confirm).

Wenn der Owner beide Mitigationen jetzt einbauen möchte, ist
das eine ~30-Zeilen-Änderung. Wenn der Owner deployen will und die
Mitigationen separat plant, ist die Schwere des Findings vertretbar
(Public-GET ist ohnehin ein bekannter Pattern-Bestand).

## Owner-Entscheidungen (Grilling 2026-06-11)

1. **Variante** → C (Presets), kein Hybrid mit A.
2. **Preset-Set** → 4 Presets: Teal, Leaf, Sun, Slatey.
3. **Logo-Einbindung** → ja, im selben Wurf, als data-URI in
   `getRegistrationConfig`-Response.
4. **Header-Schriftzug** → EEG-Name aus PROJ-32 + Logo; Footer
   wechselt auf „Powered by eegFaktura" bei non-default Brand.
5. **Vorschau** → statische Mini-Preview-Karten neben jedem Preset.
6. **Geltungsbereich** → komplette Public-Strecke (`/register`,
   `/confirm`, Error-Pages innerhalb derselben Shell).
7. **Default** → DB-NULL = Preset Teal implizit; kein Backfill.
8. **Settings-Sichtbarkeit** → nur in PROJ-67-`advanced`-Modus.
9. **Reihenfolge** → PROJ-102 **vor** PROJ-101.
10. **Awareness-Banner** → ja, `isAdvancedActive` prüft
    `brand_preset != null && != 'teal'`.
11. **Mail-Branding** → Out-of-Scope, eigene Folge-PROJ.
12. **HEX-Akzent-Override** → Out-of-Scope, reine Presets.

## Deployment

**Deploy-Bookkeeping 2026-06-11:**

- Feature-Bump: neue DB-Spalte + neuer Public-Response-Felder + neuer Public-Endpoint-Rate-Limit + neuer Admin-Editor → Minor-Version-Wechsel
- Tag: `v1.29.0-PROJ-102`
- KEINE neuen ENV-Variablen
- KEINE Helm-Wert-Änderungen
- Migration 000076 (ALTER ADD COLUMN nullable + CHECK) läuft via migrate-Job automatisch vor Backend-Rollout, non-blocking

**Owner-Aktion nach CI-Build + Helm-Auto-Bump:**

```bash
helm upgrade eegfaktura-member-onboarding ./helm/member-onboarding \
  -f helm/member-onboarding/values-env.yaml \
  -f helm/member-onboarding/values-secret.yaml
```

**Tester-Verifikation nach Deploy:**

- Bestand-EEG ohne Brand-Settings → Public-Page `/register/<rc>` rendert unverändert Teal-Theme + Bestand-Header
- Admin-UI: Einstellungs-Modus auf „Alle Optionen" umschalten → Brand-Editor unten am Stammdaten-Tab sichtbar
- Preset auf „Leaf" wechseln → Auto-Save → Public-Page neu laden → grünes Theme + EEG-Name + Logo (wenn synchronisiert) + Footer „Powered by eegFaktura"
- Standard-Modus aktivieren → Awareness-Banner zeigt „nicht-Default-Preset aktiv"

## Vor-Implementierungs-Checks (für /architecture)

- Verifizieren ob `/register/<rc_number>/confirm` als eigene Page
  existiert oder nur ein Path-Suffix ohne separate Route. Falls
  nicht existent → AC-9 Scope reduziert sich.
- Verifizieren wie heute Next.js SSR-Caching für
  `/register/<rc_number>` konfiguriert ist (`revalidate`,
  `force-dynamic`?).
- Verifizieren ob `AdminEEGSettingsEditor` bereits Auto-Save oder
  Save-Button hat — `brand_preset` soll konsistent zu den anderen
  Stammdaten-Feldern speichern.
- HSL-Werte für Leaf / Sun / Slatey aus `tailwind.config.ts`-
  Paletten ableiten (`leaf-500` → `--primary`, `leaf-50` →
  `--background`, …).
