# PROJ-112: Datenweiterleitung – Export-Granularität „eine Zeile pro Zählpunkt"

## Status: Deployed
**Created:** 2026-06-14
**Last Updated:** 2026-06-14

## Deployment (2026-06-14)

- **Image:** `sha-7b5a6e3` (Backend + Frontend). CI Build & Test + Security Scan grün; Docker-Image gebaut.
- **Helm:** `helm/member-onboarding/values.yaml` images.{backend,frontend}.tag = `sha-7b5a6e3`.
- **Git-Tag:** `v1.41.0-PROJ-112`.
- **Keine DB-Migration**, kein neues Helm-Werk/Secret/Env — reiner Code-Rollout (Backend + Frontend).
- **Owner-Aktion (manuell):** `git pull` + `helm upgrade eegfaktura-member-onboarding ./helm/member-onboarding -f values-env.yaml -f values-secret.yaml`.
- **Bestand-Änderung (G2):** Mehrwert-Trennzeichen `", "` → `" | "` (auch bestehende Mitglieds-Modus-Configs, ZP-Listen-Zelle). In CHANGELOG + User-Guide dokumentiert.

## QA Test Results (2026-06-14)

**Verdikt: READY** — 0 Critical/High/Medium/Low. Kein `/security-review` nötig (keine Trigger: kein Auth/Tenant/Schema/Public/Import/Helm/CI/Secrets).

**Acceptance Criteria — alle 12 auf Code-Ebene verifiziert:**
- AC-1/2/3 (Modus-Umschalter): `excelConfig.RowMode` + `resolveRowMode` (leer→member), `omitempty` + `parseConfig`-Default halten Bestand-Configs im Mitglieds-Modus, UI-Select im Editor. ✓
- AC-4/5/6/8 (ZP-Modus): `buildRows` erzeugt 1 Zeile/ZP, Mitglied ohne ZP → 1 Leerzeile; `extractCell` zieht Mitglieds-Felder aus der Application (wiederholt) und ZP-Felder pro Zeile. Tests in `rows_test.go`. ✓
- AC-7 (voller ZP-Feldsatz): **Katalog-Konsistenz maschinell verifiziert** — 21 Backend-ZP-Keys (`AvailableMeteringPointFields`) == 21 Frontend-Keys (Kategorie „Zählpunkt"), `diff` leer. memberOnly-Set (5) identisch. Frontend-Unit-Test `data-export-fields.test.ts` pinnt das (feedback_parallel_math_shared_vector). ✓
- AC-9/10 (Aggregat-Sichtbarkeit): `memberOnlyFields` (4 Aggregate + meter_numbers) → `ValidateConfig` lehnt im ZP-Modus mit 400 ab (Defense-in-Depth); Frontend `isFieldVisibleInMode` blendet aus; meter_count in beiden Modi. ✓
- AC-11/12 (Format-Optionen + Spalten-Reihenfolge): unverändert. ✓

**Edge Cases:** EC-1 (Leerzeile), EC-3 (ZP-Sort nach Nummer), EC-5 (typabhängige NULL → leere Zelle via deref-Helper gegen typed-nil-Pointer-Falle), G2 (Pipe-Join), G4 (Sort Mitgliedsnummer→Name), G8 (Auto-Drop+Toast) durch `rows_test.go` + `data-export-fields.test.ts` abgedeckt.

**Security-Smoke:** CSV/Excel-Injection — `sanitiseSpreadsheetValue(extractCell(...))` in **beiden** Render-Loops (Zeile 46 XLSX, 91 CSV); Pipe-Join + ZP-Werte laufen durch den Schutz; nur der Zell-Anfang triggert Formel-Injektion, Mid-Cell-Werte nach `|` sind ungefährlich. Kein SQL (reine Formatierung). PII unverändert (admin-only + tenant-scoped via `LoadForExport`). `ValidateConfig` rowMode-Whitelist + Modus-Spalten-Zulässigkeit. `FilterUnknownFields` (PROJ-61-Import) kennt jetzt beide Kataloge — ZP-Spalten werden beim Config-Import nicht mehr verworfen. **0 Findings.**

**Scans:** `go build/vet/test ./...` grün · `npx tsc --noEmit` grün · `npx vitest run` 251 grün (245 + 6 neu) · `npm run build` grün · `gosec -severity medium ./internal/dataexport/...` 0 Findings · `govulncheck` 0 callable.

**Deferred:** E2E/Playwright (braucht Backend+DB) → manuell auf test-Env nach Deploy (Modus-Umschalter, ZP-Export-Zeilen, Auto-Drop-Toast, Pipe-Trennung).

**Regression:** Mitglieds-Modus unverändert (Bestand ohne rowMode = member). **Bewusste Bestand-Änderung (G2):** die ZP-Listen-Zelle (`meter_numbers`) nutzt jetzt `|` statt `, ` — in CHANGELOG/Doku vermerken. Beifang: PROJ-99-Aggregate + `mandate_reference` erstmals im Frontend-Picker sichtbar (geschlossene Bestand-Lücke).

## Implementation Notes (Backend — 2026-06-14)

Backend vollständig in `internal/dataexport/excel` (keine DB-Migration, kein neues Paket):
- **fields.go:** `MultiValueSeparator = " | "` (Pipe-Join, ersetzt Komma auch im Bestand-`meter_numbers`). Null-sichere Deref-Helper (`derefInt64/Float64/Bool`) gegen die typed-nil-Pointer-Falle. `memberDisplayName`-Helper (single source für member_name + Sortierung). Neuer **ZP-Feld-Katalog** `AvailableMeteringPointFields` (21 Felder, `Extract func(MeteringPoint)`) + `MeterDirectionLabels`/`GenerationTypeLabels`. `memberOnlyFields`-Set (4 Aggregate + meter_numbers) für die Modus-Sichtbarkeit.
- **plugin.go:** `RowMode`-Feld in `excelConfig` (`omitempty`), Konstanten `RowModeMember`/`RowModeMeteringPoint`, `resolveRowMode`. `ValidateConfig` prüft rowMode + Modus-Spalten-Zulässigkeit (Aggregat im ZP-Modus → 400). `FilterUnknownFields` (PROJ-61-Drift) kennt beide Kataloge.
- **renderer.go:** `exportRow{app, mp}` + `buildRows` (Flatten + Sort G4: Mitgliedsnummer numerisch → ohne Nummer ans Ende → Name; ZP-Modus innerhalb Mitglied nach ZP-Nummer; Mitglied ohne ZP → 1 Leerzeile). `extractCell` = ein gemeinsamer Pfad (Mitglieds-Feld vs ZP-Feld; ZP-Modus Einzelwert, Mitglieds-Modus Pipe-Join über alle ZP, Leere übersprungen). XLSX+CSV-Writer iterieren `[]exportRow`.
- **standard_configs.go:** `BuildPreviewTable` über `buildRows` (Vorschau respektiert Modus). Neue Vorlage „Zählpunkt-Export" (rowMode=metering_point).
- **Tests:** `rows_test.go` (Flatten beide Modi, Mitglied ohne ZP, ZP-Sort, Mitgliedsnummer+Name-Sort, Pipe-Join + NULL-Skip, Direction-enum_label, ValidateConfig rowMode + Aggregat-Reject + Katalog-Disjunktheit). Bestands-Tests angepasst (Pipe-Separator + Sortierung). `go build/vet/test ./...` grün.

Offen: **/frontend** (excel-editor Modus-Auswahl + modusabhängiger Picker + Vorschau + Auto-Drop-Hinweis G8; `src/lib/data-export-fields.ts` muss den ZP-Katalog + Modus-Sichtbarkeit spiegeln), dann /qa + /deploy.

## Dependencies
- Requires: PROJ-60 (Datenweiterleitung / Async-Plugin-Framework) — liefert das Excel-Plugin, den Config-Editor und das `ApplicationSnapshot`-Modell.
- Related: PROJ-99 (Zählpunkt-Aggregat-Felder) — die Summen-Felder; deren Sichtbarkeit ändert sich im neuen Modus.
- Related: PROJ-39/45/49 — liefern die per-Zählpunkt-Felder (abweichende Adresse, Erzeugungsform, Energie-Felder), die jetzt einzeln exportierbar werden.

## Problem / Begründung

Tester-Feedback 2026-06-14: „Ich kann in der Config der Datenweiterleitung keine Spalten auf ZP-Ebene anlegen: Verbrauch, PV-Größe usw."

Die Datenweiterleitung erzeugt heute **eine Zeile pro Mitglied** (flach). Zählpunkt-Daten gibt es nur als (a) kommagetrennte ZP-Nummern-Liste in einer Zelle und (b) **Summen** über alle Zählpunkte (PROJ-99). **Einzelwerte pro einzelnem Zählpunkt** sind nicht möglich, weil ein Mitglied N Zählpunkte hat, die nicht in eine Mitglieds-Zeile passen.

Gewählter Ansatz (Owner 2026-06-14, aus Tester-Feedback): ein **„eine Zeile pro Zählpunkt"-Modus**, bei dem die Mitglieds-Stammdaten je ZP-Zeile **wiederholt** werden. Damit wird jede Zählpunkt-Eigenschaft eine echte, einzeln wählbare Spalte.

## Owner-Entscheidungen (2026-06-14, via AskUserQuestion)

| # | Entscheidung | Wahl |
|---|---|---|
| D1 | **Modus** | **Umschalter pro Config.** Bestand „eine Zeile pro Mitglied" bleibt; neu „eine Zeile pro Zählpunkt". Beide Modi parallel; bestehende Configs unverändert im Mitglieds-Modus. |
| D2 | **Mitglied ohne Zählpunkt (ZP-Modus)** | **Eine Zeile mit leeren ZP-Spalten.** Kein Mitglied fällt still aus dem Export. |
| D3 | **ZP-Felder** | **Voller Satz**, der Admin wählt im Config-Editor, was er braucht. |
| D4 | **Aggregat-/Summen-Felder im ZP-Modus** | Die 4 PROJ-99-Aggregate (3 Summen + „Speicher vorhanden") **und** die ZP-Listen-Spalte werden im ZP-Modus **ausgeblendet** (durch Einzelwerte ersetzt). **„Anzahl Zählpunkte" bleibt** als Mitglieds-Kontext wählbar. *(Beim Grilling final bestätigen.)* |

### Neue wählbare Zählpunkt-Spalten (D3, voller Satz)

Aus `shared.MeteringPoint`:
- **ZP-Nummer** (`metering_point`)
- **Richtung** (`direction` — Verbraucher/Erzeuger)
- **Faktor %** (`participation_factor`)
- **Verbrauch Vorjahr** (`consumption_previous_year`)
- **Verbrauch Prognose** (`consumption_forecast`)
- **Einspeise-Prognose** (`feed_in_forecast`)
- **PV-Leistung kWp** (`pv_power_kwp`)
- **Einspeise-Limit vorhanden / Einspeise-Limit kW** (`feed_in_limit_present` / `feed_in_limit_kw`)
- **Erzeugungsform** (`generation_type`)
- **Speicher-Größe kWh** (`battery_size_kwh`)
- **Wechselrichter-Hersteller / -Leistung kW** (`inverter_manufacturer` / `inverter_power_kw`)
- **Speichersteuerung akzeptiert** (`battery_control_acceptable`)
- **Trafo / Anlagennummer / Anlagenname** (`transformer` / `installation_number` / `installation_name`)
- **Abweichende Adresse** je ZP (`address_street` / `address_street_number` / `address_zip` / `address_city`, PROJ-39)

## Grilling-Entscheidungen (Tech-Design-Stresstest 2026-06-14)

**Korrigiertes Modell (G1 — ersetzt die Modus-Rahmung oben):** Es geht nicht um „Mitglieds- vs. Zählpunkt-Daten", sondern um **zwei Darstellungen derselben Zählpunkt-Felder**, per Config wählbar (`rowMode`):

| | **Modus A — eine Zeile pro Zählpunkt** (`metering_point`) | **Modus B — eine Zeile pro Mitglied** (`member`, Default/Bestand) |
|---|---|---|
| Zeilen | 1 Zeile je Zählpunkt | 1 Zeile je Mitglied |
| Mitgliedsdaten | je ZP-Zeile **wiederholt** | einmal |
| ZP-Feld (z. B. Verbrauch) | **Einzelwert** des jeweiligen ZP | **alle Werte** des Mitglieds, **`|`-getrennt** in der Spalte (`3000 | 1500`) |

Dieselben ZP-Felder sind in **beiden** Modi wählbar; `rowMode` steuert nur **Zeilen-Granularität + Mehrwert-Darstellung**. Modus B ist der **bestehende** Mitglieds-Zeilen-Modus, **erweitert** um die per-ZP-Felder (heute gibt es dort nur ZP-Nummern-Liste + Summen).

- **G2 — Trennzeichen `|` durchgängig (inkl. Bestand):** Mehrwert-Zellen nutzen Pipe `" | "`. **Bewusste Bestand-Änderung:** das bestehende `FieldTypeMulti`-Join (ZP-Nummern-Liste `meter_numbers`) wird von `", "` auf `" | "` umgestellt — gilt auch für laufende Mitglieds-Modus-Configs. In der CSV-Variante kollidiert `|` nicht mit dem `;`-Spaltentrenner.
- **G3 — Summen-Felder behalten (präzisiert D4):** Die 4 PROJ-99-Aggregate bleiben **additiv** wählbar (in Modus B sinnvoll; in Modus A pro Zeile wiederholt → werden dort ausgeblendet). Kein Entfernen von Bestands-Feldern.
- **G4 — Sortierung:** Zeilen nach **Mitgliedsnummer** (numerisch); Mitglieder **ohne** Nummer ans Ende, dann alphabetisch nach Name. In Modus A innerhalb eines Mitglieds nach **ZP-Nummer** aufsteigend (deterministisch/vergleichbar über wiederholte Exporte).
- **G5 — Config-Schema:** neues Top-Level-Feld `rowMode` in `excelConfig` (`""`/`member` = Default/Bestand, `metering_point` = Modus A). `ValidateConfig` validiert `rowMode` + prüft, dass jede Spalte für den gewählten Modus zulässig ist. PROJ-61-Configexport/-import: `rowMode` läuft durch den JSON-Roundtrip sauber durch; der `DriftFilter` filtert nur Spalten-Field-Keys.
- **G6 — Field-Katalog:** **getrennter ZP-Feld-Katalog** mit `Extract func(MeteringPoint)` neben dem unveränderten Mitglieds-Katalog (`Extract func(ApplicationSnapshot)`). Jedes ZP-Feld **einmal** definiert (Einzel-ZP-Extract); der Renderer macht Single-Value (Modus A) bzw. Pipe-Join über alle ZP des Mitglieds (Modus B). Jedes Feld trägt eine **Modus-Sichtbarkeit** (Mitglieds-Felder: beide; ZP-Einzelfelder: beide; Aggregate/ZP-Liste: nur Modus B).
- **G7 — Renderer = ein Pfad:** Flatten-Stufe zu `[]exportRow{ app, mp *MeteringPoint }` (Modus B: 1 Row/App mp=nil; Modus A: 1 Row je (App, ZP), plus 1 Row mp=nil für Mitglieder ohne ZP). Die bestehenden XLSX/CSV-Writer iterieren `[]exportRow`; `extractAndFormat` bekommt zusätzlich den `mp`. **Live-Vorschau** (`PreviewSample`) respektiert `rowMode`. **Kein neuer DB-Zugriff** (MeteringPoints sind im Snapshot bereits geladen).
- **G8 — Modus-Wechsel:** Wird eine Config auf Modus A umgestellt und enthält dann ungültige Spalten (Aggregate/ZP-Liste), werden diese **automatisch entfernt + Hinweis** (analog PROJ-61-Drift-Filter); Speichern bleibt möglich.
- **G9 — Standardvorlage:** eine klonbare „Zählpunkt-Export"-Vorlage via `StandardConfigs` (Modus A, sinnvolle Default-Spalten: Mitgliedsnummer, Name, ZP-Nummer, Richtung, Verbrauch Vorjahr, PV-Leistung, Speicher-Größe). Exakte Spaltenliste in `/architecture`.

## User Stories

- Als **EEG-Admin** möchte ich pro Datenweiterleitungs-Config wählen können, ob der Export eine Zeile pro Mitglied oder eine Zeile pro Zählpunkt erzeugt, damit ich für Zielsysteme, die Zählpunkt-Daten brauchen, echte Einzelwerte liefern kann.
- Als **EEG-Admin** möchte ich im ZP-Modus einzelne Zählpunkt-Felder (Verbrauch, PV-Leistung, Speicher, Richtung …) als eigene Spalten wählen, damit jeder Zählpunkt vollständig und maschinenlesbar weitergegeben wird.
- Als **EEG-Admin** möchte ich, dass die Mitglieds-Stammdaten (Name, Adresse, IBAN, Mitgliedsnummer …) in jeder Zählpunkt-Zeile wiederholt werden, damit jede Zeile für sich vollständig ist und im Zielsystem zugeordnet werden kann.
- Als **EEG-Admin** möchte ich, dass auch Mitglieder ohne Zählpunkt mit einer Zeile (leere ZP-Spalten) erscheinen, damit der Export ein vollständiger Mitglieder-Abgleich bleibt.
- Als **EEG-Admin** mit bestehenden Configs möchte ich, dass sich an meinen Mitglieds-Zeilen-Exporten nichts ändert, damit laufende Weiterleitungen stabil bleiben.

## Acceptance Criteria

**Modus-Umschalter (D1)**
- [ ] AC-1: Eine Datenweiterleitungs-Config trägt einen Modus: „Zeile pro Mitglied" (Default/Bestand) oder „Zeile pro Zählpunkt".
- [ ] AC-2: Bestehende Configs behalten ohne Änderung den Mitglieds-Modus (Backward-Compat; keine Verhaltensänderung an laufenden Exporten).
- [ ] AC-3: Der Modus ist im Config-Editor sichtbar wähl- und umstellbar.

**ZP-Modus-Export (D2/D3)**
- [ ] AC-4: Im ZP-Modus erzeugt der Export **eine Zeile pro Zählpunkt** eines Mitglieds; ein Mitglied mit 3 Zählpunkten ergibt 3 Zeilen.
- [ ] AC-5: In jeder ZP-Zeile sind die gewählten **Mitglieds-Felder** (Stammdaten/Adresse/Bank/EEG/EEG-Stammdaten) mit dem identischen Mitglieds-Wert wiederholt.
- [ ] AC-6: Die gewählten **Zählpunkt-Spalten** tragen pro Zeile den Wert **des jeweiligen Zählpunkts**.
- [ ] AC-7: Der volle Satz der ZP-Felder (siehe oben) ist im Config-Editor wählbar.
- [ ] AC-8: Ein Mitglied **ohne** Zählpunkt erzeugt im ZP-Modus **eine** Zeile mit befüllten Mitglieds-Spalten und **leeren** ZP-Spalten.

**Aggregat-Sichtbarkeit (D4)**
- [ ] AC-9: Im ZP-Modus sind die PROJ-99-Aggregat-Felder (Jahresverbrauch-Summe, PV-Leistung-Summe, Speicher-Größe-Summe, Speicher vorhanden) und die ZP-Listen-Spalte **nicht** wählbar; „Anzahl Zählpunkte" bleibt wählbar.
- [ ] AC-10: Im Mitglieds-Modus bleiben alle bisherigen Felder inkl. Aggregate unverändert wählbar.

**Formatierung / Konsistenz**
- [ ] AC-11: Die bestehenden Format-Optionen pro Feldtyp (Datum, Zahl de/iso, Bool-Varianten, Enum-Wert/Label) gelten für die neuen ZP-Spalten genauso.
- [ ] AC-12: Die Spalten-Reihenfolge folgt der im Config-Editor festgelegten Reihenfolge (Mitglieds- und ZP-Spalten frei mischbar).

## Edge Cases

- **EC-1 (Mitglied ohne Zählpunkt):** genau **eine** Zeile, ZP-Spalten leer (AC-8). Keine doppelte/fehlende Zeile.
- **EC-2 (Modus-Wechsel einer bestehenden Config):** Umstellen Mitglied→ZP ändert nur die Ausgabe-Granularität; die gewählte Spaltenliste bleibt, im ZP-Modus nicht mehr verfügbare Felder (Aggregate/ZP-Liste) werden klar behandelt (ausgeblendet/markiert, kein stiller Datenfehler).
- **EC-3 (Sortierung der ZP-Zeilen):** Reihenfolge der Zählpunkt-Zeilen innerhalb eines Mitglieds ist **stabil/deterministisch** (z. B. nach ZP-Nummer), damit wiederholte Exporte vergleichbar sind.
- **EC-4 (Abweichende Adresse je ZP, PROJ-39):** die 4 Adress-Spalten sind all-or-nothing — entweder alle vier leer (ZP nutzt Mitgliederadresse) oder alle vier gesetzt. Export gibt die ZP-eigene Adresse aus, wenn vorhanden, sonst leer.
- **EC-5 (Typ-/Sichtbarkeitsregeln der Energie-Felder, PROJ-45/49):** Felder, die je nach Richtung/Erzeugungsform NULL sind (z. B. PV-Leistung bei Verbraucher-ZP), erscheinen als leere Zelle, nicht „0".
- **EC-6 (Mitglied mit vielen Zählpunkten):** Export bleibt korrekt und performant bei Mitgliedern mit vielen ZP (Zeilenzahl = Summe aller Zählpunkte).
- **EC-7 (leerer Export):** keine Mitglieder/keine Zählpunkte → leerer Export mit Header, kein Fehler.

## Technical Requirements
- Reine Export-/Plugin-Logik im Excel-Plugin (`internal/dataexport/excel`); Renderer iteriert im ZP-Modus über Zählpunkte statt nur über Applications.
- Keine DB-Schema-Änderung erwartet (alle ZP-Felder existieren bereits in `metering_point`).
- Tenant-Isolation unverändert (Export bleibt EEG-scoped wie heute).
- Keine Mitglieder-Identifikatoren über das im Mitglieds-Modus bereits Erlaubte hinaus (gleiche PII-Regeln; Export ist admin-only).

## Non-Goals
- Kein neues Plugin (nur Excel; CSV/weitere Plugins separat).
- Keine Änderung am Mitglieds-Modus (Bestand bleibt 1:1).
- Keine Änderung an der Kern-Übergabe an eegFaktura (separater Excel-Export PROJ-17 / Import-Pfad sind nicht betroffen).
- Keine neuen Zählpunkt-Felder im Datenmodell.

## Offene Punkte für /grill-me
- Config-Schema: Wie wird der Modus gespeichert (neues Feld in der `DataExportConfig.Config`-JSON des Excel-Plugins vs. eigenes Spaltenkonzept)? Auswirkung auf Configexport/-import (PROJ-61) + Bestand-Configs.
- Field-Katalog-Architektur: getrennte ZP-Field-Definitionen mit `Extract func(MeteringPoint)` neben den bestehenden `Extract func(ApplicationSnapshot)` — wie sauber im Katalog trennen (Mitglieds-Felder vs ZP-Felder vs Modus-abhängige Sichtbarkeit)?
- Renderer-Umbau: ein Code-Pfad mit Modus-Verzweigung vs. zwei Pfade; Vermeidung von Duplikat-Logik (Mitglieds-Felder werden in beiden Modi gleich extrahiert).
- D4 final: „Anzahl Zählpunkte" im ZP-Modus behalten — ja/nein. Verhalten beim Modus-Wechsel für bereits gewählte, dann ungültige Felder.
- Sortier-Schlüssel der ZP-Zeilen (ZP-Nummer vs. Anlage-Reihenfolge). → **im Grilling gelöst (G4)**.

---

## Tech Design (Solution Architect)

### A) Datenfluss & Komponenten

```
Admin: Config-Editor (Datenweiterleitung)
  ├── Darstellungs-Modus wählen:  ● eine Zeile pro Mitglied (Default)
  │                               ○ eine Zeile pro Zählpunkt
  ├── Spalten-Picker (zeigt modusabhängig verfügbare Felder)
  │     ├── Mitglieds-Felder (Stammdaten/Adresse/Bank/EEG)  ← beide Modi
  │     ├── Zählpunkt-Felder (Verbrauch, PV, Speicher, …)   ← beide Modi
  │     └── Summen / ZP-Liste                                ← nur Mitglieds-Modus
  └── Live-Vorschau (rendert im gewählten Modus)
                 │  speichert
                 ▼
   Config (Datenweiterleitung)  =  { Format, rowMode, Spalten[] }
                 │  Export auslösen → Job
                 ▼
   Excel-Plugin „Process"
     ├── 1. Flatten:  Mitglieder+Zählpunkte → Zeilen-Liste
     │        Modus B → 1 Zeile je Mitglied
     │        Modus A → 1 Zeile je Zählpunkt (+ 1 Leerzeile für Mitglied ohne ZP)
     ├── 2. Sortieren (Mitgliedsnummer → ohne Nummer ans Ende → Name; in A je ZP-Nr.)
     └── 3. Rendern (XLSX oder CSV, je Spalte Wert ziehen + formatieren)
                 ▼
        Download-Datei (XLSX / CSV)
```

Der **Flatten-Schritt** ist das Herzstück: Er übersetzt „Mitglieder mit ihren Zählpunkten" in eine flache Zeilenliste. Danach ist der Rest (Sortieren, Schreiben) für beide Modi **identisch** — die bestehenden Excel/CSV-Schreiber bleiben unangetastet.

### B) Datenmodell (Klartext)

**Keine Datenbank-Änderung.** Alle Zählpunkt-Daten existieren bereits und werden ohnehin schon mitgeladen. Es ändert sich nur die **Form der Export-Datei** und ein Feld in der Config:

- **`rowMode`** (neu, in der Config-JSON der Weiterleitung): „eine Zeile pro Mitglied" (Standard, = bisheriges Verhalten) oder „eine Zeile pro Zählpunkt". Fehlt das Feld (Bestand-Configs), gilt automatisch „pro Mitglied".
- **Zählpunkt-Felder** (neu wählbar, beide Modi): ZP-Nummer, Richtung, Faktor, Verbrauch Vorjahr/Prognose, Einspeise-Prognose/-Limit, PV-Leistung, Erzeugungsform, Speicher-Größe, Wechselrichter (Hersteller/Leistung), Speichersteuerung, Trafo/Anlagennummer/-name, abweichende Adresse.
- **Modus-Sichtbarkeit** je Feld: Mitglieds- und ZP-Felder sind in **beiden** Modi wählbar; Summen + ZP-Nummern-Liste nur im Mitglieds-Modus.
- **Mehrwert-Trennzeichen**: senkrechter Strich `" | "` — **einheitlich**, auch für die bestehende ZP-Nummern-Liste (war bisher Komma).

### C) Tech-Entscheidungen (warum so)

- **Ein Renderer-Pfad statt zwei** (Flatten-Stufe): Würden wir zwei getrennte Export-Pfade bauen, müssten Sortierung, Formatierung und die Schutz-Logik gegen Formel-Injektion doppelt gepflegt werden — teuer und fehleranfällig. Mit der Flatten-Stufe gibt es **eine** Zeilenliste und **einen** Schreiber; der Modus beeinflusst nur, wie die Zeilenliste entsteht.
- **Getrennter Zählpunkt-Feld-Katalog**: Die 30+ bestehenden Mitglieds-Felder bleiben **unverändert**. Zählpunkt-Felder werden **einmal** definiert (Wert eines einzelnen Zählpunkts); der Renderer entscheidet je Modus, ob er den Einzelwert nimmt (Modus A) oder die Werte aller Zählpunkte mit `|` verbindet (Modus B). So gibt es jede ZP-Eigenschaft nur an **einer** Stelle.
- **Kein Datenbank-/Schema-Change**: Zählpunkte sind im Export-Snapshot bereits geladen — das Feature ist reine Darstellungs-Logik. Das hält das Risiko niedrig (keine Migration, keine Daten-Rückwirkung).
- **`rowMode` als Config-Feld statt neuer Config-Typ**: Bestehende Weiterleitungen laufen unverändert weiter (fehlendes Feld = altes Verhalten), und der Konfig-Export/-Import (PROJ-61) trägt das Feld automatisch mit.
- **Pipe `|` als Trennzeichen**: kollidiert nicht mit dem Semikolon-Spaltentrenner der CSV-Variante und ist optisch eindeutig.
- **Auto-Entfernen ungültiger Spalten beim Modus-Wechsel**: Wer von Mitglieds- auf Zählpunkt-Modus wechselt, hat evtl. Summen-Spalten gewählt, die dort keinen Sinn ergeben. Statt das Speichern zu blockieren, werden sie automatisch entfernt + ein Hinweis gezeigt — gleiches Muster wie der bestehende Konfig-Import-Drift-Filter.

### D) Frontend-Auswirkung

Betroffen ist der **Datenweiterleitungs-Config-Editor** (`data-export/excel-editor.tsx`) plus der dortige gespiegelte Feld-Katalog (`src/lib/data-export-fields.ts`):
- Neue **Modus-Auswahl** (zwei Optionen) im Editor-Kopf.
- Der **Spalten-Picker** zeigt die Felder **modusabhängig** (Summen/ZP-Liste nur im Mitglieds-Modus; ZP-Einzelfelder in beiden).
- Die **Live-Vorschau** rendert im gewählten Modus (im Zählpunkt-Modus also mehrere Zeilen pro Beispiel-Mitglied).
- Beim Moduswechsel: **Hinweis**, welche Spalten automatisch entfernt wurden.
- Wie bei den HSL-/Validierungs-Fällen muss der **Frontend-Feld-Katalog mit dem Backend-Katalog konsistent** bleiben (gleiche Feld-Keys + Modus-Sichtbarkeit).

### E) Abhängigkeiten

**Keine neuen Pakete.** Genutzt werden die bestehenden Bausteine (Excel-Bibliothek `excelize`, das PROJ-60-Plugin-Framework, die PROJ-61-Konfig-Mechanik). Keine Migration, kein neuer Service, kein neues Helm-Werk.

### F) Empfohlene Umsetzungs-Reihenfolge

Das Feature ist **backend-lastig** (Flatten, Feld-Katalog, Renderer, Validierung). Empfehlung: **`/backend` zuerst** (Modus-Feld + ZP-Katalog + Flatten/Renderer + Standardvorlage + Validierung), dann **`/frontend`** (Modus-Auswahl + modusabhängiger Picker + Vorschau + Auto-Drop-Hinweis). So steht der maßgebliche Feld-Katalog backend-seitig fest, bevor das Frontend ihn spiegelt.
