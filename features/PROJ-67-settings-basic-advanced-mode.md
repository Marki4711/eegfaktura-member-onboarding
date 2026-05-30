# PROJ-67 — Basic-/Advanced-Modus für Einstellungen

**Status:** Planned
**Created:** 2026-05-30
**Owner:** TBD
**Source:** Owner-Direktive 2026-05-30 — Pilot-Rückmeldung „die Menge an Einstellmöglichkeiten überfordert kleine EEGs"

## Hintergrund

Die Einstellungsseite hat im Laufe der Zeit eine ansehnliche Anzahl an Toggles, Inputs und Konfigurations-Optionen angesammelt (Stammdaten & SEPA, Einleitungstext, Formular-Felder, Rechtsdokumente, Externe API, Datenweiterleitung, Import/Export). Für **kleine Vereine**, die im Wesentlichen nur „Stammdaten der Bewerber erfassen und in eegFaktura übertragen" wollen, ist die schiere Menge der Optionen **abschreckend** und führt zu Fehl-Konfigurationen oder Vermeidungsverhalten.

Owner-Direktive: Es soll eine **umschaltbare Ansicht** (Basic ↔ Erweitert) geben, die in der Basic-Variante nur die für den 80%-Use-Case nötigen Optionen zeigt und alle anderen ausblendet.

## User Stories

- **Als Admin einer kleinen EEG** möchte ich beim ersten Öffnen der Einstellungen nicht von Dutzenden Toggles erschlagen werden, sondern eine übersichtliche „Basic"-Ansicht mit den ~5 wichtigsten Entscheidungen sehen.
- **Als Admin einer Power-EEG** möchte ich mit einem Klick auf „Erweitert" alle Konfigurations-Optionen sehen, die ich heute schon kenne und nutze.
- **Als neue Admin** möchte ich nicht von Anfang an entscheiden müssen, welche Ansicht ich brauche — die Default-Ansicht sollte sinnvoll sein und mich nicht zur Power-User-Konfiguration zwingen.

## Was zählt zur „Basic"-Konfiguration?

Vorschlag, mit dem Owner zu validieren — bitte vor Implementierung gegenchecken:

### Basic — direkt im Settings-Tab sichtbar (alle EEGs brauchen das)

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

### Option A — Toggle „Basic / Erweitert" am Seitenkopf

```
Einstellungen RC123456          [ Basic | Erweitert ]
                                         ─ ─ ─
```

- Pro EEG persistiert (`registration_entrypoint.settings_view_mode`)
- Beim Wechsel werden „erweiterte" Sektionen ein-/ausgeblendet, **ohne** die hinterlegten Werte zu löschen
- Basic ist Default für neue EEGs; bestehende behalten ihr aktuelles Verhalten (Erweitert)
- **Vorteil:** klar, einfach, ein Klick
- **Nachteil:** zwei UI-Zustände müssen gepflegt werden; Erklärungs-Bedarf was wo lebt

### Option B — Progressive Disclosure mit „Erweiterte Einstellungen anzeigen"-Bereich pro Tab

```
Stammdaten & SEPA
─────────────────
[Basic-Sektionen sichtbar]

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
1. EEG bekommt beim Anlegen `settings_view_mode = 'basic'` als Default.
2. Bestehende EEGs migrieren auf `'advanced'` (= heutiges Verhalten — nichts ändert sich für sie).
3. Toggle oben rechts in den Einstellungen (sticky), wechselt zwischen den beiden Modi.
4. Im Basic-Modus sind „erweiterte" Sektionen nicht nur ausgeblendet, sondern auch mit einer kleinen Notiz versehen am unteren Tab-Ende: „Diese EEG nutzt die Basic-Ansicht. Für SEPA-B2B, Aktivierungs-Kriterium etc. → Modus auf Erweitert umstellen."

## Acceptance Criteria

### AC-1 — Persistenter Modus pro EEG
- Neue Spalte `registration_entrypoint.settings_view_mode` mit Werten `basic` / `advanced`, Default `basic`.
- Migration: bestehende EEGs bekommen `advanced` (rückwärts-kompatibel).
- Toggle am Seitenkopf der Settings-Page; Switch persistiert sofort und aktualisiert die Sichtbarkeit.

### AC-2 — Basic-Sichtbarkeit
- Im Basic-Modus sind nur die in dieser Spec genannten Basic-Sektionen sichtbar.
- Tabs „Import / Export" und ggf. „Datenweiterleitung" werden weiterhin angezeigt — die Datenweiterleitung ist Kern-Funktionalität auch für kleine EEGs (siehe Scope-Doku in `docs/user-guide/index.md`).
- Versteckte Sektionen verlieren **keine** hinterlegten Werte; sie sind nur nicht editierbar.

### AC-3 — Advanced-Sichtbarkeit
- Im Advanced-Modus ist alles wie heute sichtbar. Keine Regression bei Power-Usern.

### AC-4 — Mode-Wechsel mit ungespeicherten Änderungen
- Wenn der Admin den Modus wechselt während PROJ-66's dirty-State aktiv ist, greift derselbe Confirm-Dialog wie beim Tab-Wechsel.

### AC-5 — Doku-Spiegelung (Owner-Direktive 2026-05-30)
- `docs/user-guide/06-admin-settings.md` wird umstrukturiert: jede Sektion bekommt einen Marker, ob sie Basic oder Erweitert ist (z. B. via Header-Suffix „(Erweitert)" oder einer farbigen Box am Anfang der Sektion).
- Die Tabelle „Welche Toggle-Kombination ergibt was?" (SEPA) wird um eine Spalte „Basic / Erweitert" ergänzt.
- Im Doku-Kopf von 06 steht eine kurze Erklärung des Basic/Erweitert-Konzepts mit Verweis auf den Toggle.
- Ein neuer Doku-Abschnitt „Welcher Modus passt zu mir?" mit Faustregeln (kleine Vereine → Basic, Power-User mit B2B / SEPA-Sonderfällen → Erweitert).

## Non-Goals

- **Kein dritter Modus (z. B. „Custom").** Zwei reichen.
- **Keine Pro-Tab-Modi.** Der Modus gilt für die ganze Einstellungsseite.
- **Keine Berechnung der „best fit"-Mode-Empfehlung** anhand der bestehenden Konfiguration. Admin entscheidet bewusst.

## Offene Punkte (vor `/architecture`)

1. **Validierung der Basic-Sektionen-Liste:** ist die Liste oben deckungsgleich mit dem, was Pilot-EEGs tatsächlich nutzen? → mit 2-3 EEGs gegenchecken vor Implementierung.
2. **Datenweiterleitung als Basic-Tab?** Argument dafür: Excel-Export ist auch für kleine EEGs wertvoll (Buchhaltungs-Ablage, Datenschutz-AVV-Liste). Argument dagegen: die Plugin-Konfiguration ist komplex. → Vorschlag: Tab bleibt sichtbar, aber Default-Plugin-Konfig „Bewerber-Excel" wird beim ersten EEG-Anlegen vorangelegt.
3. **Migration für bestehende EEGs:** `advanced` (= heutiges Verhalten) oder optimistisch `basic`? Vorschlag: `advanced`, damit niemand überrascht wird.
4. **Mode-Indikator in der Doku:** wie auszeichnen — `(Erweitert)`-Header-Suffix, farbige Box, Icon? Sollte konsistent mit dem Settings-UI sein.
5. **Setup-Wizard (Option C)**: später als eigene Phase, oder gleich mitziehen? Aufwand-Schätzung entscheidet.

## Dependencies

- PROJ-66 (Tab-Switch-Schutz) — der Mode-Wechsel-Confirm-Dialog wiederverwendet die `UnsavedChangesDialog`-Komponente
- PROJ-61 (Config-Export) — das neue Feld `settings_view_mode` gehört in den Export/Import-Diff

## Risiken

- **Doku-Aufwand.** Owner-Direktive verlangt, dass die Klassifizierung sich in der Doku widerspiegelt. Das ist nicht-trivial — `06-admin-settings.md` ist heute ~300 Zeilen und müsste durchgearbeitet werden. → Aufwand mit einplanen.
- **„Versteckt aber persistiert"**-Falle. Wenn ein Admin im Basic-Modus eine erweiterte Option deaktiviert (durch Modus-Wechsel auf Advanced, Toggle umlegen, zurück auf Basic), wirkt das Setting weiterhin — sichtbar ist es aber nicht mehr. Sicherheits-/Audit-Implikation klären (z. B. SEPA-B2B aktiv, aber Admin sieht es nicht). → Vorschlag: Basic zeigt einen Hinweis-Banner „SEPA-B2B ist aktiv (nur in Erweitert sichtbar)".
- **Versuch in eine Richtung:** wenn der Toggle live geht und dann zurückgenommen werden müsste, ist die Migration nicht trivial. → konservativ rollouten (Test-Stage, eine Pilot-EEG, dann breit).
