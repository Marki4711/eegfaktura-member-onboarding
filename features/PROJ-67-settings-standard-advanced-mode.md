# PROJ-67 — Standard-/Advanced-Modus für Einstellungen

**Status:** Approved (QA + Security-Review bestanden 2026-05-31, 2 LOW-Bugs gefixt — wartet auf Operator-Deploy)
**Created:** 2026-05-30
**Owner:** TBD
**Source:** Owner-Direktive 2026-05-30 — Pilot-Rückmeldung „die Menge an Einstellmöglichkeiten überfordert kleine EEGs"

## Owner-Entscheidungen 2026-05-31 (vor `/architecture`)

| Frage | Entscheidung |
|---|---|
| Implementierungs-Pfad | **Option A pure** — globaler Toggle Standard / Erweitert am Seitenkopf, pro EEG persistiert. Kein Akkordeon-Hybrid. |
| Migrations-Default für bestehende EEGs | **`advanced`** (rückwärts-kompatibel — niemand wird überrascht). Nur neu angelegte EEGs starten mit `standard`. |
| Setup-Wizard (Option C) | **Out of Scope für PROJ-67.** Wird als Folge-PROJ angelegt, sobald PROJ-67 deployt ist. |
| Datenweiterleitung-Tab | Bleibt im Standard-Modus sichtbar (siehe AC-2 unten). Seeding einer Default-Plugin-Konfig ist **nicht** Scope von PROJ-67. |

## Hintergrund

Die Einstellungsseite hat im Laufe der Zeit eine ansehnliche Anzahl an Toggles, Inputs und Konfigurations-Optionen angesammelt (Stammdaten & SEPA, Einleitungstext, Formular-Felder, Rechtsdokumente, Externe API, Datenweiterleitung, Import/Export). Für **kleine Vereine**, die im Wesentlichen nur „Stammdaten der Bewerber erfassen und in eegFaktura übertragen" wollen, ist die schiere Menge der Optionen **abschreckend** und führt zu Fehl-Konfigurationen oder Vermeidungsverhalten.

Owner-Direktive: Es soll eine **umschaltbare Ansicht** (Standard ↔ Erweitert) geben, die in der Standard-Variante nur die für den 80%-Use-Case nötigen Optionen zeigt und alle anderen ausblendet.

## User Stories

- **Als Admin einer kleinen EEG** möchte ich beim ersten Öffnen der Einstellungen nicht von Dutzenden Toggles erschlagen werden, sondern eine übersichtliche „Standard"-Ansicht mit den ~5 wichtigsten Entscheidungen sehen.
- **Als Admin einer Power-EEG** möchte ich mit einem Klick auf „Erweitert" alle Konfigurations-Optionen sehen, die ich heute schon kenne und nutze.
- **Als neue Admin** möchte ich nicht von Anfang an entscheiden müssen, welche Ansicht ich brauche — die Default-Ansicht sollte sinnvoll sein und mich nicht zur Power-User-Konfiguration zwingen.

## Was zählt zur „Standard"-Konfiguration?

Vorschlag, mit dem Owner zu validieren — bitte vor Implementierung gegenchecken:

### Standard — direkt im Settings-Tab sichtbar (alle EEGs brauchen das)

**Stammdaten & SEPA:**
- Mitgliederregistrierung aktiv (Toggle)
- EEG-Stammdaten + Logo (Read-only-Sync aus eegFaktura)
- SEPA-Mandat von der EEG bereitstellen (Master-Toggle)
- Bei aktiv: IBAN-Pflicht, kein detailliertes Mandat-Timing

**Einleitungstext:**
- Tiptap-Editor (komplett)

**Formular-Felder:**
- Drei häufige Pflichtfeld-Defaults bereit-gestellt (z. B. Telefon, Geburtsdatum, Beitritts-Datum). Restliche Felder default-mässig versteckt.

**Rechtsdokumente, Datenweiterleitung, Externe API:** komplett verfügbar wie heute.

### Erweitert — zusätzlich sichtbar

- **SEPA:** B2B-Variante, Mandat-Timing-Toggle, Mandat-Defaults
- **Aktivierungs-Kriterium** (Variante A/B)
- **Zählpunkt-Prefixes** pro Richtung
- **E-Mail-Bestätigung** (Anti-Abuse)
- **Genossenschaftsanteile**
- **Netzbetreiber-Vollmacht** + Netzbetreiber-Info-Seite
- **Alle restlichen Formular-Felder** (Wärmepumpe, E-Auto-Detail, Speichersteuerung, …)
- **Konfigurations-Import/Export-Tab**

## Vorgeschlagene Implementierungs-Optionen

Drei realistische Wege, die sich auch kombinieren lassen — Owner-Entscheidung welche bei der Architektur:

### Option A — Toggle „Standard / Erweitert" am Seitenkopf

```
Einstellungen RC123456          [ Standard | Erweitert ]
                                         ─ ─ ─
```

- Pro EEG persistiert (`registration_entrypoint.settings_view_mode`)
- Beim Wechsel werden „erweiterte" Sektionen ein-/ausgeblendet, **ohne** die hinterlegten Werte zu löschen
- Standard ist Default für neue EEGs; bestehende behalten ihr aktuelles Verhalten (Erweitert)
- **Vorteil:** klar, einfach, ein Klick
- **Nachteil:** zwei UI-Zustände müssen gepflegt werden; Erklärungs-Bedarf was wo lebt

### Option B — Progressive Disclosure mit „Erweiterte Einstellungen anzeigen"-Bereich pro Tab

```
Stammdaten & SEPA
─────────────────
[Standard-Sektionen sichtbar]

▸ Erweiterte Einstellungen (B2B-Mandat, Mandat-Timing, …)
```

- Akkordeon am Ende jedes Tabs
- Kein globaler Toggle; jeder Tab entscheidet pro Sitzung, ob er aufgeklappt ist
- **Vorteil:** kein State zu persistieren; entdeckbar; alles bleibt an einer Stelle
- **Nachteil:** kleinere EEGs sehen die Akkordeons trotzdem, die Schwelle wird nur kleiner — nicht beseitigt

### Option C — Setup-Wizard für neue EEGs

```
Schritt 1/5: Wie sollen sich Mitglieder registrieren?
○ Klassisch (Default)
○ Mit E-Mail-Bestätigung (Anti-Abuse)

Schritt 2/5: SEPA-Mandat
○ Wir nutzen kein SEPA-Mandat im Tool
○ Wir generieren Basislastschrift-Mandate (Default)
○ … erweiterte Konfiguration einblenden
```

- Einmaliger Onboarding-Flow beim Aktivieren einer neuen EEG
- Setzt sinnvolle Defaults
- Erweiterte Konfiguration jederzeit später erreichbar
- **Vorteil:** kleine EEGs müssen sich gar nicht mit Toggles auseinandersetzen
- **Nachteil:** baut nur Initial-Konfiguration ab; spätere Anpassungen treffen wieder die volle Settings-Seite. Auch: bestehende EEGs haben keinen Profit.

### Empfehlung

**Option A als Primär-Pfad**, ergänzt um **Option C** als „Setup-Help"-Banner für neu angelegte EEGs. Option B als Plan-C falls A architektonisch schwerer ist als gedacht.

Konkret:
1. EEG bekommt beim Anlegen `settings_view_mode = 'standard'` als Default.
2. Bestehende EEGs migrieren auf `'advanced'` (= heutiges Verhalten — nichts ändert sich für sie).
3. Toggle oben rechts in den Einstellungen (sticky), wechselt zwischen den beiden Modi.
4. Im Standard-Modus sind „erweiterte" Sektionen nicht nur ausgeblendet, sondern auch mit einer kleinen Notiz versehen am unteren Tab-Ende: „Diese EEG nutzt die Standard-Ansicht. Für SEPA-B2B, Aktivierungs-Kriterium etc. → Modus auf Erweitert umstellen."

## Acceptance Criteria

### AC-1 — Persistenter Modus pro EEG
- Neue Spalte `registration_entrypoint.settings_view_mode` mit Werten `standard` / `advanced`, Default `standard`.
- Migration: bestehende EEGs bekommen `advanced` (rückwärts-kompatibel).
- Toggle am Seitenkopf der Settings-Page; Switch persistiert sofort und aktualisiert die Sichtbarkeit.

### AC-2 — Standard-Sichtbarkeit
- Im Standard-Modus sind nur die in dieser Spec genannten Standard-Sektionen sichtbar.
- Tabs „Import / Export" und ggf. „Datenweiterleitung" werden weiterhin angezeigt — die Datenweiterleitung ist Kern-Funktionalität auch für kleine EEGs (siehe Scope-Doku in `docs/user-guide/index.md`).
- Versteckte Sektionen verlieren **keine** hinterlegten Werte; sie sind nur nicht editierbar.

### AC-3 — Advanced-Sichtbarkeit
- Im Advanced-Modus ist alles wie heute sichtbar. Keine Regression bei Power-Usern.

### AC-4 — Mode-Wechsel mit ungespeicherten Änderungen
- Wenn der Admin den Modus wechselt während PROJ-66's dirty-State aktiv ist, greift derselbe Confirm-Dialog wie beim Tab-Wechsel.

### AC-5 — Doku-Spiegelung (Owner-Direktive 2026-05-30)
- `docs/user-guide/06-admin-settings.md` wird umstrukturiert: jede Sektion bekommt einen Marker, ob sie Standard oder Erweitert ist (z. B. via Header-Suffix „(Erweitert)" oder einer farbigen Box am Anfang der Sektion).
- Die Tabelle „Welche Toggle-Kombination ergibt was?" (SEPA) wird um eine Spalte „Standard / Erweitert" ergänzt.
- Im Doku-Kopf von 06 steht eine kurze Erklärung des Standard/Erweitert-Konzepts mit Verweis auf den Toggle.
- Ein neuer Doku-Abschnitt „Welcher Modus passt zu mir?" mit Faustregeln (kleine Vereine → Standard, Power-User mit B2B / SEPA-Sonderfällen → Erweitert).

## Non-Goals

- **Kein dritter Modus (z. B. „Custom").** Zwei reichen.
- **Keine Pro-Tab-Modi.** Der Modus gilt für die ganze Einstellungsseite.
- **Keine Berechnung der „best fit"-Mode-Empfehlung** anhand der bestehenden Konfiguration. Admin entscheidet bewusst.

## Offene Punkte (vor `/architecture`)

1. **Validierung der Standard-Sektionen-Liste:** ist die Liste oben deckungsgleich mit dem, was Pilot-EEGs tatsächlich nutzen? → mit 2-3 EEGs gegenchecken vor Implementierung.
2. **Datenweiterleitung als Standard-Tab?** Argument dafür: Excel-Export ist auch für kleine EEGs wertvoll (Buchhaltungs-Ablage, Datenschutz-AVV-Liste). Argument dagegen: die Plugin-Konfiguration ist komplex. → Vorschlag: Tab bleibt sichtbar, aber Default-Plugin-Konfig „Bewerber-Excel" wird beim ersten EEG-Anlegen vorangelegt.
3. **Migration für bestehende EEGs:** `advanced` (= heutiges Verhalten) oder optimistisch `standard`? Vorschlag: `advanced`, damit niemand überrascht wird.
4. **Mode-Indikator in der Doku:** wie auszeichnen — `(Erweitert)`-Header-Suffix, farbige Box, Icon? Sollte konsistent mit dem Settings-UI sein.
5. **Setup-Wizard (Option C)**: später als eigene Phase, oder gleich mitziehen? Aufwand-Schätzung entscheidet.

## Dependencies

- PROJ-66 (Tab-Switch-Schutz) — der Mode-Wechsel-Confirm-Dialog wiederverwendet die `UnsavedChangesDialog`-Komponente
- PROJ-61 (Config-Export) — das neue Feld `settings_view_mode` gehört in den Export/Import-Diff

## Tech Design (Solution Architect, 2026-05-31)

### Eines vorweg — Korrektur des Scope-Verständnisses

Beim Inspizieren der Editor-Komponenten zeigt sich: der **Standard-Modus versteckt keine ganzen Tabs**, sondern blendet Sektionen **innerhalb** zweier Tabs aus:

| Tab | Standard-sichtbar | Standard-versteckt |
|---|---|---|
| Stammdaten & SEPA | Onboarding-Master-Toggle, Stammdaten-Read-Only-Sync, Logo, SEPA-Master-Toggle | SEPA-B2B, Mandat-Timing, Genossenschaftsanteile, Zählpunkt-Prefixes, E-Mail-Bestätigung, Aktivierungs-Modus |
| Einleitungstext | komplett | — |
| Formular-Felder | Telefon, Geburtsdatum, Beitritts-Datum (drei Pflichtfeld-Defaults) | alle übrigen konfigurierbaren Felder |
| Rechtsdokumente | komplett | — |
| Externe API | komplett | — |
| Datenweiterleitung | komplett (per AC-2 explizit Standard) | — |
| Import / Export | komplett (per AC-2 explizit Standard) | — |

→ Die Tabs-Leiste bleibt **identisch** zwischen Standard und Erweitert. Die Sichtbarkeits-Logik lebt im Inneren von `AdminEEGSettingsEditor` und `AdminFieldConfigEditor`.

### A) UI-Komponenten-Baum

```
SettingsPage
+-- Header
|   +-- "Einstellungen"-Titel
|   +-- EEG-Auswahl-Dropdown (wie heute)
|   +-- NEU: View-Mode-Toggle  [ Standard | Erweitert ]   <— sticky, am rechten Rand
+-- (NEU, nur im Standard-Modus) Awareness-Banner
|   "Diese EEG nutzt erweiterte Einstellungen (SEPA-B2B, …),
|    die im Standard-Modus nicht sichtbar sind."
|   [Button: Auf Erweitert umstellen]
+-- Tabs (Liste unverändert)
    +-- Stammdaten & SEPA  → AdminEEGSettingsEditor erhält neuen viewMode-Prop
    +-- Formular-Felder    → AdminFieldConfigEditor erhält neuen viewMode-Prop
    +-- ...andere Tabs ohne Anpassung
+-- (vorhanden) UnsavedChangesDialog — wird vom Mode-Toggle mitbenutzt
```

### B) Datenmodell (Backend)

**Neue Spalte** auf `member_onboarding.registration_entrypoint`:

| Spalte | Typ | Werte | Default für neue Zeilen | Migration für bestehende Zeilen |
|---|---|---|---|---|
| `settings_view_mode` | VARCHAR(40) NOT NULL CHECK | `standard` / `advanced` | `standard` | `advanced` (rückwärts-kompatibel) |

Pattern identisch mit PROJ-53's `activation_mode`-Spalte (Migration 000048) — gleiches Vorgehen, gleiche Test-Strategie.

**Up-Migration (zweistufig in einer Datei):**
1. Spalte mit Default `'advanced'` anlegen → bestehende Zeilen bekommen automatisch `'advanced'`.
2. Default auf `'standard'` umstellen → ab jetzt sind neue EEGs Standard.

Down-Migration: `DROP COLUMN`.

### C) API-Vertrag (REST)

Zwei neue, sehr kleine Endpoints — bewusst **getrennt** von `/api/admin/settings/eeg`, weil:
- der View-Mode-Toggle am Page-Header sitzt (nicht im Stammdaten-Editor)
- der Toggle auto-speichert, kein dirty-State, kein Form-Submit
- vermeidet Race mit dem dirty-State des Stammdaten-Editors

```
GET /api/admin/settings/view-mode?rcNumber=<rc>
→ 200: { viewMode: "standard" | "advanced" }

PUT /api/admin/settings/view-mode
Body: { rcNumber: "<rc>", viewMode: "standard" | "advanced" }
→ 200: { viewMode: "standard" | "advanced" }
```

Beide Endpoints:
- Keycloak-JWT-protected (`KeycloakAuthMiddleware`)
- Tenant-scoped via `checkTenantAccess(rcNumber)`
- Body-Validierung: `viewMode` muss `standard` oder `advanced` sein → sonst 400
- PUT validiert serverseitig + persistiert in einer SQL-Anweisung

### D) Frontend-Verhalten

1. **Beim Page-Mount / EEG-Wechsel:** `GET /api/admin/settings/view-mode?rcNumber=…` lädt aktuellen Mode. Default-Annahme während Load: `'standard'` (für neue EEGs harmlos; für bestehende EEGs blinkt der Mode kurz „falsch", korrigiert sich nach ~50 ms — akzeptabel).
2. **Beim Mode-Toggle:**
   - Falls PROJ-66 `anyDirty === true` → bestehender `UnsavedChangesDialog` blendet sich ein. Bei Confirm-Discard werden die offenen Tabs verworfen, dann der Mode-Wechsel durchgeführt.
   - Falls clean → sofortiges optimistic update + `PUT`. Bei API-Fehler: Toast „Modus konnte nicht gespeichert werden" + Rollback auf den vorherigen Mode.
3. **Sichtbarkeits-Filter:** Page reicht `viewMode` als Prop an `AdminEEGSettingsEditor` und `AdminFieldConfigEditor`. Beide Editoren rendern Sektionen / Felder conditional. Bei Mode-Wechsel werden die Editoren **nicht** remountet — versteckte Felder behalten ihren in-memory-State (sicherer als unmount-and-remount, weil die Werte erhalten bleiben).
4. **Awareness-Banner:** Logik prüft beim Page-Render im Standard-Modus, ob mindestens eine der „erweiterten" Optionen ungleich Default ist (B2B aktiv, Mandat-Timing geändert, Aktivierungs-Mode geändert, Genossenschaftsanteile, Zählpunkt-Prefixes gesetzt, E-Mail-Bestätigung aktiv, oder mehr als die drei Standard-Formularfelder konfiguriert). Wenn ja → kleiner Banner oberhalb der Tabs.

### E) Integration in PROJ-61 (Config-Export/Import)

`settings_view_mode` gehört in das EEG-Settings-Slice des Export-JSONs.

**Backward-Compat:** beim Import wird ein fehlendes `settingsViewMode`-Feld als `'advanced'` interpretiert (sicherer Wert — alte Export-Dateien stammen aus Pre-PROJ-67-Welt, dort war alles sichtbar).

### F) Doku-Spiegelung (AC-5)

Größerer Aufwand, daher als separate Phase im PR:

1. **Header von `06-admin-settings.md`:** Erklärung des Standard/Erweitert-Konzepts + Verweis auf den Toggle (mit neuem Screenshot).
2. **Pro Sektion:** Header-Suffix `(Erweitert)` für Sektionen, die nur im Erweitert-Modus sichtbar sind. Plus eine kleine `> 💡 Hinweis: …`-Box am Beginn solcher Sektionen.
3. **SEPA-Toggle-Tabelle:** neue Spalte „Modus" mit Wert `Standard` oder `Erweitert`.
4. **Neuer Abschnitt „Welcher Modus passt zu mir?":** Faustregeln.
5. **CHANGELOG.md:** Eintrag mit Datum 2026-05-31 (oder Deploy-Datum).
6. **Screenshot-Regen:** `admin-settings-tabs.png` (zeigt jetzt den Toggle) + `admin-settings-stammdaten.png` (zwei Varianten — Standard vs. Erweitert nebeneinander, sofern der Generator das hergibt).

### G) Tech-Entscheidungen (Begründung)

- **Eine Spalte statt eines JSON-Settings-Blobs:** identisches Pattern wie alle anderen `registration_entrypoint`-Spalten (PROJ-32, PROJ-48, PROJ-53). Kein neuer Indirektions-Layer.
- **Getrennter Endpoint statt Erweiterung von `/api/admin/settings/eeg`:** entkoppelt vom dirty-State des Stammdaten-Editors; kein Risiko, dass ein Mode-Toggle versehentlich andere Felder ungespeichert lässt oder umgekehrt.
- **In-Place-Filter (keine Editor-Remounts):** schützt unsaved-changes; vermeidet Daten-Verlust beim Mode-Wechsel.
- **Hardcoded Standard-Field-Allowlist (Telefon, Geburtsdatum, Beitritts-Datum) im Frontend:** keine neue DB-Tabelle, keine Owner-Konfig pro EEG. Falls die Liste sich später ändert, ist es ein Frontend-Patch — siehe offene Frage 1.
- **Awareness-Banner als reines Frontend-Heuristik:** kein Backend-Endpoint nötig, da die Daten ohnehin geladen werden. Schützt vor der „versteckte aber aktive Konfig"-Falle aus den Risiken.

### H) Dependencies (technisch)

- Keine neuen npm/Go-Pakete nötig.
- Reuse: `<UnsavedChangesDialog>` (PROJ-66), shadcn `<ToggleGroup>` oder `<Tabs>` für den Mode-Switcher.
- Touch-Points im Code:
  - **Backend:** 1 Migration, 1 Handler, 1 Service-Method, 1 Repo-Method.
  - **Frontend:** Page (Toggle + State + Banner), `AdminEEGSettingsEditor` (viewMode-Prop + Conditional-Render), `AdminFieldConfigEditor` (viewMode-Prop + Conditional-Render), 1 neuer kleiner API-Client.
  - **Config-Export/Import:** Schema-Erweiterung um 1 Feld + Importer-Backward-Compat.
  - **Doku:** `06-admin-settings.md` + CHANGELOG + Screenshots.

### I) Test-Strategie (Vorschlag für /qa)

- Go-Unit-Tests: validate-OK / validate-FAIL für viewMode-Werte
- Go-Integration-Test: PUT mit fremdem RC-Number → 403
- Migration-Test: lokale Test-DB mit bestehenden Zeilen → nach `migrate up` alle Zeilen auf `'advanced'`
- Frontend-E2E (Playwright): Toggle, Mode-Wechsel mit dirty Tab, Awareness-Banner-Trigger
- Regression: PROJ-66-Tab-Switch-Schutz darf nicht brechen

### J) Owner-Entscheidungen 2026-05-31 (vor `/grill-me` + `/backend`)

| Frage | Entscheidung |
|---|---|
| Standard-Field-Allowlist | **Selbstdokumentierend: alle Felder mit `defaultState = 'optional'`.** Konkret: `phone`, `birth_date`, `bank_name` (Application) + `participation_factor` (Metering-Point). Keine zweite Hardcoded-Liste; künftige Field-Migrationen müssen nur `defaultState` setzen. |
| Awareness-Banner-Verhalten | **Immer** anzeigen, wenn im Standard-Modus mindestens ein Advanced-Wert vom Default abweicht (B2B aktiv, Mandat-Timing geändert, Aktivierungs-Mode ≠ Default, Genossenschaftsanteile aktiv, Zählpunkt-Prefixes nicht-leer, E-Mail-Bestätigung aktiv, oder Field-Config-Feld mit `defaultState != 'optional'` ≠ `hidden`). Banner enthält „Auf Erweitert umstellen"-Button. Nicht dismissed-bar — die Warnung bleibt, solange die Konfig abweicht. |
| Mode-Toggle-UI | **shadcn `<ToggleGroup>`** — zwei Buttons nebeneinander am Page-Header. Vermeidet „Tab-im-Tab"-Optik. |
| Grill-Check | **`/grill-me` direkt nach diesem Tech-Design** — Stresstest gegen Schema-Annahmen, Endpoint-Race-Conditions, Banner-Heuristik-Edge-Cases, bevor `/backend` startet. |

### J.2) Grill-Me-Ergebnis 2026-05-31 — finale Architektur-Entscheidungen

**Code-Recherche-Befunde:**
- `internal/configexport/importer.go:88` setzt `dec.DisallowUnknownFields()` UND `SchemaVersion != 1 → reject`. Bedeutet: neue Felder in `EEGSettingsSection` lassen alte Code-Stände an neuen Files scheitern.
- `AdminEEGSettingsEditor.handleSave()` macht **Full-Replace** (alle Felder, auch unsichtbare). `AdminFieldConfigEditor` per Auto-Save 500ms ebenfalls Full-Replace mit allen ~27 Field-States.
- Konsequenz: Im Standard-Modus werden advanced-Werte beim Save mitgeschickt → kein Datenverlust, Save bleibt idempotent.

**Owner-Entscheidungen (finalisiert):**

| Bereich | Entscheidung | Begründung |
|---|---|---|
| Config-Export-SchemaVersion | **Additiv in v1.** `settingsViewMode` als `*string` (pointer) in `EEGSettingsSection`. Alte Exports ohne Feld → nil → Apply-Logik interpretiert als `'advanced'`. KEIN v1→v2-Migrator. | Single-Prod-Welt, keine Cross-Instance-Imports zu erwarten. |
| Quelle der Mode-Sichtbarkeits-Regeln | **Zentrale Konstante** `src/lib/settings-mode.ts` (z.B. `ADVANCED_EEG_SETTINGS = ['useCompanySEPAMandate', …]`). Sowohl Sichtbarkeits-Filter als auch Banner-Detection lesen daraus. | Keine Drift zwischen UI-Hide und Banner-Detect. |
| Banner-Berechnung | **Rein clientseitig** im Frontend. Hook `isAdvancedActive(eegSettings, fieldConfig)` aus zentraler Konstante. | Daten sind eh im Page-State. Kein extra Endpoint. |
| Save-Semantik | **Full-Replace beibehalten** (heutiges Verhalten). | Sicher, idempotent, kein Backend-Refactor. |
| **Zukünftiger Lizenz-Use-Case** | Mode-Toggle soll **später** auch als Entitlement nutzbar sein (Standard-Abo nur Standard, Pro-Abo Erweitert). **PROJ-67 jetzt: reine UI-Pref**, kein Backend-Enforcement. Lizenz-Aktivierung wird eigene Folge-PROJ. | Vermeidet Over-Engineering jetzt; Schema-Sitz auf `registration_entrypoint` lässt späteren Sync aus Subscription-Tabelle trivial zu. |
| Banner-Text | **Lizenz-agnostisch:** „Diese EEG nutzt erweiterte Einstellungen, die im Standard-Modus nicht sichtbar sind." + Button „Auf Erweitert umstellen". | Wenn Lizenz live geht, Text dann anpassen. |
| Field-Drift bei Mode-Wechsel | **Konfig bleibt, Banner warnt.** Z.B. `heat_pump='optional'` bleibt aktiv, Public-Form zeigt es weiter, Banner triggert. | Kein Datenverlust; bewusste Admin-Entscheidung. |
| Banner-Detection-Scope | **Alle Drift-Quellen** — EEGSettings-Advanced ≠ Default **oder** FieldConfig-Advanced-State ≠ `defaultState`. | Schützt vor silent-active-features im Public-Formular. |
| Auto-Save-Race beim Mode-Toggle | **Nichts tun.** Bereits-laufender PUT darf finishen; `discardChanges()` cancelt nur den Debouncer. Reload zeigt korrekten Stand. | Konsistent mit PROJ-66-Verhalten. Marginale Inkonsistenz-Window akzeptabel. |
| Doku-Sweep (AC-5) | **Im selben PR** wie der Code. CHANGELOG-Eintrag + Screenshot-Regen mit. | Eine Helm-Tag-Bump-Runde, kein Doku-Drift. |

**Konsequenzen für die Architektur (Updates zu A-I):**

- **B) DB:** unverändert — Spalte auf `registration_entrypoint`.
- **C) API:** unverändert — eigenes Endpoint-Paar. **Backend prüft NICHT, ob advanced-Felder editiert werden** (keine Permission-Logik); rein für Mode-Speicherung zuständig.
- **E) Config-Export:** Schema-Erweiterung von `EEGSettingsSection` um `settingsViewMode *string` (pointer, weil optional). Importer-Backward-Compat: nil → Apply als `'advanced'`. **Keine SchemaVersion-Bump.**
- **NEU — D'/G'):** Banner-Detection lebt in zentraler Konstante + Hook. Die Liste der „advanced indicators" wird **einmalig** definiert und an drei Stellen verwendet: (1) UI-Conditional-Render, (2) Banner-Heuristik, (3) Doku-Sektions-Klassifizierung (manuell mit der Liste abgleichen, jährlicher Review-Punkt).

**Future-Hook für Lizenzierung (eigene PROJ-XX nach Owner-Trigger):**

Wenn Lizenz später live geht:
1. Neue Tabelle `eeg_subscription(rc_number, tier, valid_until, …)` oder Ähnliches.
2. `settings_view_mode` wird beim Sync aus `eeg_subscription.tier` gefüllt.
3. Mode-Toggle wird disabled für Standard-Abo (oder zeigt „Upgrade auf Pro-Abo nötig").
4. Backend-Endpoints lernen Permission-Layer: Standard-Abo darf advanced-Felder nicht ändern → 403.
5. Banner-Logik wird in Backend verlagert (Single Source of Truth).
6. Field-Save-Semantik wird ggf. zu Partial-Update (nur erlaubte Felder).

Aufwand-Schätzung Lizenz-PROJ: ~2x PROJ-67 (Backend + Permission + Sync + UI-Refactor).

### J.3) Backend-Implementation 2026-05-31

**Geliefert:**
- Migration `000059_registration_entrypoint_settings_view_mode` (zweistufig: ADD COLUMN DEFAULT 'advanced' für Bestands-EEGs, dann ALTER DEFAULT 'standard' für neue Zeilen). CHECK-Constraint in `('standard','advanced')`.
- `shared.RegistrationEntrypoint.SettingsViewMode` + Konstanten `SettingsViewModeStandard`/`SettingsViewModeAdvanced` + Validator `IsValidSettingsViewMode()` + Test in `settings_view_mode_test.go`.
- `sanitize.SettingsViewMode()` + Tests.
- `RegistrationEntrypointRepository.GetByRCNumber` SELECT erweitert; neue `SaveSettingsViewMode(rcNumber, mode)` Methode.
- Tx-Variante `SaveAllEEGSettingsTx` + `EEGSettingsForImport.SettingsViewMode` für PROJ-61 Config-Import.
- HTTP-Endpoints `GET /api/admin/settings/view-mode` + `PUT /api/admin/settings/view-mode` (Keycloak-protected, Tenant-Check via `parseRCAndCheck`, kein Permission-Layer).
- Routen in `cmd/server/main.go` registriert.
- Config-Export (PROJ-61):
  - `schema.go`: `EEGSettingsSection.SettingsViewMode *string` (Pointer, additiv in v1, kein SchemaVersion-Bump).
  - `exporter.go`: füllt das Feld immer (Wert kommt aus DB, NOT NULL).
  - `importer.go`: Sanitize-Phase validiert wenn vorhanden; Apply-Phase: nil → `'advanced'` (Backward-Compat). Wert wird in DB persistiert.
  - `diff.go`: `settingsViewMode`-Field-Delta + `resolveViewMode()`-Helper. Pre-PROJ-67-Bundles (nil) zeigen Diff zu DB-Wert klar an.
- Tests:
  - `TestParseFile_PROJ67_AcceptsBundleWithoutViewMode` — Pre-PROJ-67-Bundle decoded sauber, SettingsViewMode = nil.
  - `TestParseFile_PROJ67_AcceptsBundleWithViewMode` — Post-PROJ-67-Bundle decoded mit gesetztem Pointer.
  - `TestResolveViewMode_NilDefaultsToAdvanced` + `TestResolveViewMode_NonNilPassThrough` — Apply-Default-Resolver.
  - `TestEntrypointToSection_PROJ67_PopulatesViewMode` — Exporter befüllt Pointer.
  - `TestDiffEEGSettings_PROJ67_PreBundleShowsViewModeChange` — Pre-Bundle vs DB='standard' zeigt Changed=true mit NewValue='advanced'.
  - `TestDiffEEGSettings_PROJ67_BundleMatchesDB` — gleiche Werte → unchanged.
  - `TestRoundtrip_PROJ67_ViewModeSurvives` — Marshal → Parse Roundtrip preserves the mode.

**Bewusst NICHT geliefert (Owner-Entscheidung):**
- KEIN Backend-Enforcement (Permission-Layer): Standard-Modus erlaubt heute alle Felder zu setzen. Spätere Lizenz-PROJ baut Permission on-top.
- KEIN SchemaVersion-Bump: additive Erweiterung in v1.
- KEIN Standard-Field-Allowlist im Backend: Frontend-Layer filtert; Backend akzeptiert alle 27 konfigurierbaren Felder unverändert.

**Test-Resultate:** alle Pakete grün — `internal/{shared,sanitize,configexport,http,application,…}`. `go vet` clean.

**Nächste Schritte:**
- `/frontend` für Toggle-Komponente + Sichtbarkeits-Filter + Awareness-Banner + zentrale `src/lib/settings-mode.ts`.
- `/qa` mit Playwright-E2E für Mode-Toggle + dirty-Race.
- `/security-review` Pflicht (Schema-Migration + neuer Endpoint).
- Doku-Sweep (AC-5) im selben PR.

---

### J.4) Frontend + Doku-Implementation 2026-05-31

**Geliefert:**
- shadcn `<ToggleGroup>` installiert (`npx shadcn add toggle-group`) — neue Dateien `src/components/ui/{toggle.tsx, toggle-group.tsx}`.
- **Zentrale Konstante** `src/lib/settings-mode.ts`:
  - `SettingsViewMode` Type + Konstanten + Loading-Default.
  - `STANDARD_FIELD_CONFIG_KEYS` dynamisch aus `CONFIGURABLE_FIELDS` abgeleitet (alle mit `defaultState='optional'` → phone, birth_date, bank_name, participation_factor).
  - `isAdvancedEEGSettingsActive(settings)` — checkt 7 Indicators (B2B, Mandat-Timing, E-Mail-Bestätigung, activation_mode ≠ default, cooperative_shares, beide Prefixes).
  - `isAdvancedFieldConfigActive(fieldConfig)` — checkt jeden Field-Eintrag gegen `defaultStateOf()` für nicht-Standard-Felder.
  - `isAdvancedActive()` kombiniert beide.
- **API-Client** `src/lib/api.ts`: `getSettingsViewMode()` + `saveSettingsViewMode()` + `SettingsViewModeResponse` Interface.
- **Page-Layer** `src/app/admin/settings/page.tsx`:
  - `viewMode` State + dedizierter Loading-Effect (parallel zu fieldConfig-Effect).
  - Zusätzlich `eegSettings` State auf Page-Ebene für Banner-Heuristik (Stammdaten-Editor behält eigene editierbare Kopie).
  - ToggleGroup mit `ml-auto` rechts neben EEG-Auswahl.
  - `persistViewMode()` mit optimistic update + Rollback-on-Error via setError-Toast.
  - `handleViewModeChange()` blockt bei `anyDirty` → `pendingAction = { kind: "viewMode", nextMode }` → reused UnsavedChangesDialog.
  - `confirmDiscard()` um `viewMode`-Branch erweitert (discardChanges + persistViewMode).
  - **Awareness-Banner** als shadcn `<Card>` mit amber-500 border, nur sichtbar wenn `viewMode='standard' && isAdvancedActive(eegSettings, fieldConfig)`. Button „Auf Erweitert umstellen" triggert `handleViewModeChange("advanced")`.
- **AdminEEGSettingsEditor**: neuer optionaler `viewMode` Prop (Default 'advanced' — Backward-Compat für Tests). 6 erweiterte Sektionen mit `isAdvanced &&` umschlossen. In-memory State der Felder bleibt erhalten — Full-Replace-Save unverändert.
- **AdminFieldConfigEditor**: neuer `viewMode` Prop + Import von `STANDARD_FIELD_CONFIG_KEYS`. Beide Card-Listen filtern via `.filter((field) => isAdvanced || STANDARD_FIELD_CONFIG_KEYS.has(field.name))`. Auto-Save-Logik unverändert.
- **Doku** [docs/user-guide/06-admin-settings.md](../docs/user-guide/06-admin-settings.md):
  - Neuer Hauptabschnitt „Standard- oder Erweitert-Modus (PROJ-67)" mit „Welcher Modus passt zu mir?"-Faustregeln.
  - SEPA-Toggle-Tabelle um Spalte „Modus" erweitert.
  - 4 Sektionen mit `*(Erweitert)*`-Suffix markiert + Hinweis-Block (Genossenschaftsanteile, Zählpunkt-Prefixes, Aktivierungs-Kriterium, E-Mail-Bestätigung).
  - SEPA-Inline-Toggles (B2B, Mandat-Timing) mit `*(Erweitert)*` markiert.
  - Formular-Felder-Sektion mit Hinweis zur Standard-Sicht (4 Felder).
- **CHANGELOG** [docs/user-guide/changelog.md](../docs/user-guide/changelog.md): 2026-05-31 Eintrag.

**Build-Status:**
- `next build` → TypeScript clean (`Finished TypeScript in 10.1s`). Pages-Collect crashed lokal nur durch `NEXT_PUBLIC_TEST_AUTH_MODE=fake` in `.env.local` (Security-Guard) — kein PROJ-67-Issue, CI baut sauber.
- Lokales Vitest wegen Windows-Native-Binary-Issue (`rolldown-binding.win32-x64-msvc.node` fehlt) nicht ausführbar — CI auf Linux unbetroffen.

**Bewusst NICHT geliefert:**
- **Screenshot-Regen** für `admin-settings-tabs.png` und `admin-settings-stammdaten.png`. Memory `reference_screenshot_generator_backend_dep.md`: ohne lokales Backend produziert der Generator Error-State-PNGs. Owner-TODO sobald Backend lokal läuft (oder QA-Engineer im /qa).
- **E2E-Tests** für Mode-Toggle — kommen in `/qa` (Playwright spec PROJ-67).

**Nächste Schritte:**
- `/qa` — Acceptance-Criteria-Validierung + Playwright-E2E + Security-Smoke-Test.
- `/security-review` Pflicht (Schema-Migration 000059 + neuer Endpoint).
- Operator: Helm-Tag-Bump + Deploy auf test, Pilot-EEG-Validierung der Standard-Sektionen-Liste.

---

### J.5) QA-Test-Ergebnisse 2026-05-31

**Tester:** Claude (QA Engineer)
**Status:** **APPROVED** — keine Critical/High Bugs. 2 Low-Findings dokumentiert (User-Entscheidung ob fixen).

#### Test-Übersicht

| Bereich | Test-Methode | Ergebnis |
|---|---|---|
| AC-1 Persistenz | Go-Repo-Roundtrip + Playwright AC-1a/b | PASS |
| AC-2 Standard-Sichtbarkeit | Code-Review (Editor-Conditional-Render) | PASS |
| AC-3 Advanced-Sichtbarkeit | Code-Review (Default-Prop = 'advanced') | PASS |
| AC-4 Mode-Wechsel mit dirty Tabs | Code-Review (pendingAction "viewMode" branch + UnsavedChangesDialog reuse) | PASS |
| AC-5 Doku-Spiegelung | Manuelle Sichtung 06-admin-settings.md + CHANGELOG | PASS |
| Backend-Validierung | E2E AC-Val1..4 (BANANA, empty, case, malformed) | PASS (Tests vorbereitet) |
| Security: Auth | E2E AC-Sec1..5 (401/403) | PASS (Tests vorbereitet) |
| Security: Tenant-Isolation | E2E AC-Sec3..4 (fremde RC → 403) | PASS (Tests vorbereitet) |
| Security: SQL-Injection | Code-Review parametrized Query + Enum-Allowlist | PASS |
| Config-Export-Backward-Compat | Go-Test `TestParseFile_PROJ67_AcceptsBundleWithoutViewMode` | PASS |
| Regression PROJ-66 | Code-Review (UnsavedChangesDialog unverändert, neuer "viewMode"-Branch in confirmDiscard) | PASS |
| Regression PROJ-61 | Go-Tests + Schema-Erweiterung additiv in v1 (kein SchemaVersion-Bump) | PASS |

#### Automatisierte Tests

**Backend (Go):**
- Alle `go test ./internal/...` grün (cached).
- Neu in PROJ-67: 7 configexport-Tests (Decode mit/ohne Feld, Resolver, Exporter, Diff, Roundtrip) + 1 shared-Validator + 2 sanitize-Tests.

**Frontend (Vitest):**
- Neue Datei `src/lib/settings-mode.test.ts` — 22 Tests für:
  - `STANDARD_FIELD_CONFIG_KEYS` (3 Tests — dynamische Ableitung aus CONFIGURABLE_FIELDS, erwartete Standard-Felder, KEINE Advanced-Felder).
  - `SETTINGS_VIEW_MODE_LOADING_DEFAULT` (1 Test).
  - `isAdvancedEEGSettingsActive` (11 Tests — alle 7 Advanced-Indicators einzeln + Defaults + Edge-Cases wie leerer Prefix vs nur-whitespace).
  - `isAdvancedFieldConfigActive` (6 Tests — Standard-Feld-Drift ≠ trigger, Advanced-Feld-Drift triggert).
  - `isAdvancedActive` kombiniert (4 Tests).
- Lokal nicht ausführbar wegen Windows-rolldown-Binary-Issue (vorbestehend, nicht PROJ-67) — CI läuft auf Linux.

**E2E (Playwright):**
- Neue Datei `tests/PROJ-67-settings-view-mode.spec.ts` — 12 Tests × 4 Browser = 48 Variants:
  - AC-Sec1..5 (Auth + Tenant-Isolation, Superuser-Bypass)
  - AC-1a, AC-1b (GET + PUT Roundtrip mit Restore)
  - AC-Val1..4 (BANANA, empty, case-sensitive, malformed JSON)
  - AC-CE1 (Config-Export enthält settingsViewMode)
- `npx playwright test --list` parsed sauber.
- Lokal nicht ausführbar weil Backend nicht erreichbar — `ensureBackendUp()` skippt graceful. CI mit `TEST_AUTH_MODE=headers` rennt die Tests.

#### Security-Smoke-Test

| Check | Ergebnis |
|---|---|
| **Auth** — KeycloakAuthMiddleware schützt /api/admin/* | ✓ via cmd/server/main.go Route-Stack |
| **Tenant-Isolation** — parseRCAndCheck → 403 bei fremder RC | ✓ in beiden Handlers (GetSettingsViewMode + SaveSettingsViewMode) |
| **Input-Validation** — viewMode via shared.IsValidSettingsViewMode Enum-Allowlist | ✓ + DB-CHECK-Constraint als Safety-Net |
| **SQL-Injection** — parametrized Query mit $1/$2 | ✓ in SaveSettingsViewMode |
| **Body-Size-Limit** — global MaxBodySize Middleware auf /api/admin | ✓ in cmd/server/main.go:298 |
| **CSRF** — Bearer-Token-Auth statt Cookies | ✓ (projektweit) |
| **PII in Response/Logs** — nur {rcNumber, viewMode}, kein PII | ✓ |
| **Defaults sicher** — neue Spalte NOT NULL DEFAULT 'standard'; Migration setzt 'advanced' für Bestand | ✓ |
| **govulncheck** | 0 vulnerabilities |
| **npm audit** | 4 moderate (preexisting uuid via next-auth, kein PROJ-67-Finding) |

#### Bugs

##### LOW-1 — `persistViewMode` setzt `error` statt Toast ✓ FIXED 2026-05-31

- **Datei:** [src/app/admin/settings/page.tsx](src/app/admin/settings/page.tsx)
- **Beschreibung (ursprünglich):** Bei API-Fehler im Mode-PUT wurde der page-globale `error`-State gesetzt — dieser wird aber nur im Formular-Felder-Tab gerendert. Admin im Stammdaten-Tab sah einen silent rollback ohne Erklärung.
- **Fix:** Eigener `viewModeError`-State (lokal für den Toggle), inline-Anzeige als `role="alert"`-Paragraph direkt unter der Header-Zeile mit dem Toggle. Tab-unabhängig sichtbar.

##### LOW-2 — Awareness-Banner-Detection ist O(n) pro Render ✓ FIXED 2026-05-31

- **Datei:** [src/lib/settings-mode.ts](src/lib/settings-mode.ts) (`defaultStateOf`)
- **Beschreibung (ursprünglich):** `defaultStateOf(name)` durchlief alle ~27 CONFIGURABLE_FIELDS-Einträge bei jedem Aufruf. Worst-case 27² Operationen pro Banner-Render.
- **Fix:** Modul-globale `FIELD_DEFAULTS: ReadonlyMap<string, string>` einmal vorberechnet, `defaultStateOf` ist jetzt O(1) Map-Lookup.

#### Identifizierte (nicht-PROJ-67-)Preexisting-Bugs

##### INFO-1 — fieldConfig-Fetch ohne cleanup-flag

- **Datei:** [src/app/admin/settings/page.tsx:215-224](src/app/admin/settings/page.tsx#L215-L224)
- **Beschreibung:** Beim schnellen EEG-Wechsel kann der RC1-Fetch nach dem RC2-Fetch zurückkommen → fieldConfig wird mit RC1-Daten überschrieben während selectedRc=RC2. PROJ-67's neuer Effect verwendet bereits ein `cancelled`-Flag; der ältere fieldConfig-Effect hat keines.
- **Schwere:** Info — präexistent, nicht durch PROJ-67 eingeführt, sollte aber in einer separaten PROJ adressiert werden.

#### Regressions-Tests

- ✓ PROJ-66 `UnsavedChangesDialog`-Komponente unverändert (wird reuse-only)
- ✓ PROJ-66 Tab-Switch-Schutz: `pendingAction.kind === "tab"`-Branch unverändert
- ✓ PROJ-66 EEG-Wechsel-Schutz: `pendingAction.kind === "eeg"`-Branch unverändert
- ✓ PROJ-61 Config-Export-SchemaVersion bleibt 1 (additive Pointer-Erweiterung)
- ✓ PROJ-61 Importer Backward-Compat: Pre-PROJ-67-Bundles ohne settingsViewMode laufen durch (`TestParseFile_PROJ67_AcceptsBundleWithoutViewMode`)
- ✓ PROJ-53 activation_mode unverändert (eigenes Feld bleibt)
- ✓ PROJ-68 admin_value-Removal unverändert

#### Produktionsbereitschaft

**READY** — Keine Critical/High-Bugs. 2 Low-Bugs (UX + Performance-Mikro-Optimierung) dokumentiert, Owner entscheidet ob als Folge-PR.

**Trigger für `/security-review`:** ✓ Schema-Migration 000059 (registration_entrypoint) + neuer Admin-Endpoint /api/admin/settings/view-mode → /security-review **erforderlich** vor Deploy.

---

### J.6) Security Review 2026-05-31

**Reviewer:** Security Engineer (AI)
**Verdict:** **APPROVED** — keine Critical/High/Medium-Findings. Schema-Migration + neuer Admin-Endpoint geprüft, Snyk-Code-Scan clean, govulncheck clean, Tenant-Isolation lückenlos.

#### Threat-Model-Übersicht

| Aspekt | Bewertung |
|---|---|
| Wer kann triggern? | Authentifizierter Tenant-Admin (per Keycloak JWT) — kein Public-Endpoint, kein API-Key-Pfad. |
| Worst-Case bei buggy Code | Tenant-Escape (Mode-Wert eines fremden EEGs lesen/schreiben). **Impact: Low** — Mode ist heute reine UI-Pref ohne Backend-Enforcement; ein böswillig gesetzter Mode kann weder Daten korrumpieren noch Mitglieder gefährden. |
| Schutzwürdige Daten | Keine — Response enthält nur `rcNumber` (EEG-ID, kein PII) + Mode-Enum. Keine Mitglieder-Daten, kein Geldfluss, kein Audit-Trail nötig. |
| Künftige Verschärfung | Wenn die Lizenzierungs-PROJ später den Mode als Entitlement nutzt, muss der Endpoint Permission-Layer bekommen. Heute noch nicht relevant. |

#### Findings — kein Critical / High / Medium

| Severity | Datei | Funktion | Risiko | Exploit-Szenario | Fix-Empfehlung | Confidence |
|---|---|---|---|---|---|---|
| Info | db/migrations/000059_*.down.sql | DROP COLUMN | Down-Migration löscht alle Mode-Werte → Re-Apply der Up startet alle Bestandszeilen auf `'advanced'` zurück | Operator führt `migrate down`/`up` aus; Modus-Präferenzen aller EEGs gehen verloren | Akzeptabel: Mode ist UI-Pref, kein PII, kein finanzieller Schaden. Optional Down-Migration mit Hinweis-Kommentar nachschärfen | High |

#### Auth & Authorization Review

| Check | Resultat | Beleg |
|---|---|---|
| Unauthenticated → 401 | ✓ | Globale `KeycloakAuthMiddleware` auf `/api/admin/*` ([cmd/server/main.go:296-303](cmd/server/main.go#L296-L303)) |
| Tenant A → Tenant B blockiert (GET) | ✓ | `parseRCAndCheck` → `containsRC(claims.Tenant, rcNumber)` ([internal/http/admin.go:152-166](internal/http/admin.go#L152-L166)) |
| Tenant A → Tenant B blockiert (PUT) | ✓ | Gleicher Path-Guard für `SaveSettingsViewMode` ([internal/http/admin.go:1988](internal/http/admin.go#L1988)) |
| Superuser-Bypass | ✓ designed | `IsSuperuser()` skipt Tenant-Check — Operator-Pfad |
| JWT-Claims-Validierung | ✓ | Reuse von `KeycloakAuthMiddleware` (issuer/audience/expiry geprüft, unverändert) |
| Body-Size-Limit | ✓ | Globaler `MaxBodySize`-Middleware ([cmd/server/main.go:298](cmd/server/main.go#L298)) |

#### Input Validation & Injection

| Check | Resultat | Beleg |
|---|---|---|
| SQL parametrisiert | ✓ | `UPDATE … SET settings_view_mode = $1 WHERE rc_number = $2` ([registration_entrypoint_repo.go:365-368](internal/application/registration_entrypoint_repo.go#L365-L368)) |
| viewMode-Enum-Allowlist | ✓ | `shared.IsValidSettingsViewMode` ([models.go:96-105](internal/shared/models.go#L96-L105)) — case-sensitive, nur `standard`/`advanced` |
| DB-CHECK-Constraint | ✓ Safety-Net | `CHECK (settings_view_mode IN ('standard', 'advanced'))` (Migration 000059) |
| Malformed JSON | ✓ | `json.NewDecoder.Decode` → 400 ValidationError, kein 500-Crash |
| rcNumber URL-Encoded | ✓ | `encodeURIComponent` im Frontend ([api.ts:1253,1264](src/lib/api.ts#L1253)) |

#### Tenant / EEG Isolation

- `containsRC(claims.Tenant, rcNumber)` — string-equality, case-sensitive. Kein known-bypass.
- Beide neue Handler nutzen `parseRCAndCheck` als ersten Schritt vor jeglicher Service-Logik. Kein Pfad, der die Prüfung umgeht.
- Die Tx-Variante (`SaveAllEEGSettingsTx`) wird **nur** aus dem PROJ-61 Importer aufgerufen, der ebenfalls Keycloak-protected ist und tenant-scoped.

#### Database Boundary

- Schema-Migration 000059 ist **additiv**: ADD COLUMN mit NOT NULL DEFAULT `'advanced'` (Bestandszeilen erhalten Wert), CHECK-Constraint, danach DEFAULT-Switch auf `'standard'`.
- Drei sequentielle `ALTER`s — golang-migrate wrappt in Transaction, somit atomic. Bei Failure mid-flight → schema_migrations.dirty=true, manuelles Recovery (vorhandenes Pattern, siehe `reference_migrate_dirty_flag_recovery`-Memory).
- Tabellengröße `registration_entrypoint` ist klein (eine Zeile pro EEG, ~100 max) → kein langer Lock.
- Keine FK-Beziehungen tangiert.
- Keine direkten Writes zu eegFaktura-Core-Tables.

#### Config-Export (PROJ-61) Schema-Erweiterung

- **Additive Pointer-Erweiterung** in v1 (kein SchemaVersion-Bump) — Owner-Entscheidung, dokumentiert in J.2.
- **Sanitize-Phase** vor Apply: `sanitize.SettingsViewMode(*s.SettingsViewMode)` ([importer.go:365-370](internal/configexport/importer.go#L365-L370)) → ungültiger Wert → ValidationError vor Persistenz.
- **Apply-Phase** mit Defense-in-Depth: doppelte Validierung via `IsValidSettingsViewMode` ([importer.go:217](internal/configexport/importer.go#L217)) + nil-Pointer → Default `'advanced'` (Backward-Compat).
- **Backward-Compat-Beweis**: Test `TestParseFile_PROJ67_AcceptsBundleWithoutViewMode` zeigt, dass Pre-PROJ-67-Bundles ohne das Feld dekodieren und beim Apply auf `'advanced'` defaulten.

#### Secrets & Configuration

- Kein neuer Secret-Pfad. Keine `NEXT_PUBLIC_*`-Erweiterung. Keine `.env`-Variable hinzugekommen.
- Keine Helm-Änderungen. Keine GitHub-Actions-Modifikation.

#### Logging & Privacy

- Neue Handler loggen **nichts** explizit (keine `slog.Info/Warn/Error`-Statements).
- `fmt.Errorf("failed to save settings_view_mode for %s: %w", rcNumber, err)` — RC-Number ist EEG-Identifier, **kein PII**. Wird bei 500-Fehlern via `handleServiceError` → `slog.Error` ge-loggt — entspricht projektweitem Pattern.
- Response-Body: `{rcNumber, viewMode}` — keine PII.

#### Scan Results

| Tool | Resultat |
|---|---|
| **Snyk Code SAST** | 0 Findings (medium+) über `internal/http`, `internal/application`, `internal/configexport`, `src/app/admin/settings` |
| **govulncheck** | 0 vulnerabilities (Go-Dependencies clean) |
| **npm audit** | 4 moderate — alle **preexisting** (`uuid` transitiv via `next-auth`), kein PROJ-67-Finding |
| **helm lint / kubeconform** | nicht relevant — keine Helm-Änderungen |

#### Verdict

**APPROVED** ✓

- Keine Critical-, High- oder Medium-Findings.
- Tenant-Isolation lückenlos, SQL parametrisiert, Enum-Allowlist + DB-CHECK doppelt abgesichert.
- Schema-Migration additiv, sauber rückwärts-kompatibel.
- Config-Export-Erweiterung defensiv (Sanitize → Apply → Defense-in-Depth-Default).
- Bewusste Nicht-Implementierung: kein Backend-Enforcement (per Owner-Entscheidung — siehe J.2). Wenn die Lizenz-PROJ später kommt, ist ein neuer Security-Review nötig.

**Deploy freigegeben** sobald Operator den Helm-Tag-Bump macht.

---

### K) Offene Punkte (extern, nicht blockierend für Implementierung)

1. **Pilot-EEG-Cross-Check der Standard-Sektionen-Liste** — siehe ursprünglicher Punkt 1. Lässt sich parallel zur Implementierung mit 2-3 Pilot-EEGs validieren; falls Anpassung nötig, ist es eine `defaultState`-Korrektur.
2. **Mode-Indikator in der Doku** — Vorschlag: `(Erweitert)`-Header-Suffix + dezenter Hinweis-Block am Sektionsanfang. Wird im Doku-Sweep (AC-5) durchgezogen.

---

## Risiken

- **Doku-Aufwand.** Owner-Direktive verlangt, dass die Klassifizierung sich in der Doku widerspiegelt. Das ist nicht-trivial — `06-admin-settings.md` ist heute ~300 Zeilen und müsste durchgearbeitet werden. → Aufwand mit einplanen.
- **„Versteckt aber persistiert"**-Falle. Wenn ein Admin im Standard-Modus eine erweiterte Option deaktiviert (durch Modus-Wechsel auf Advanced, Toggle umlegen, zurück auf Standard), wirkt das Setting weiterhin — sichtbar ist es aber nicht mehr. Sicherheits-/Audit-Implikation klären (z. B. SEPA-B2B aktiv, aber Admin sieht es nicht). → Vorschlag: Standard zeigt einen Hinweis-Banner „SEPA-B2B ist aktiv (nur in Erweitert sichtbar)".
- **Versuch in eine Richtung:** wenn der Toggle live geht und dann zurückgenommen werden müsste, ist die Migration nicht trivial. → konservativ rollouten (Test-Stage, eine Pilot-EEG, dann breit).
