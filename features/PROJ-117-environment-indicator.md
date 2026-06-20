# PROJ-117: Umgebungs-Indikator (Nicht-Prod-Kennzeichnung)

## Status: In Progress
**Created:** 2026-06-20
**Last Updated:** 2026-06-20

## Überblick / Kontext

Optische, dauerhafte Kennzeichnung, wenn der Nutzer **nicht in der Prod-Umgebung** ist. Owner-Idee 2026-06-20.

**Sicherheitsrelevanz (Kernmotivation):** Die Test-Zone (`member-onboarding-test`) ist aktuell an den **PRODUKTIV-Faktura-Core** angebunden (Zonen-Konzept, PROJ-116). Ein Admin, der nicht merkt, dass er in der Test-Umgebung ist, kann über **Import/Sync reale Mitglieder in den Prod-Core schreiben**. Ein deutlicher Umgebungs-Indikator beugt der Verwechslung vor.

Reiner **Indikator** — keine funktionale Sperre. Getrennt von PROJ-115 (Free-Banner im Rechnungen-Tab) und PROJ-116 (Prod-Instanz); alle koexistieren.

## Owner-Entscheidungen (2026-06-20)

- **Scope:** Banner im **Admin-Bereich UND im Public-Registrierungsformular**.
- **Verhalten:** **dauerhaft**, nicht wegklickbar (Safety-Indikator).
- **Inhalt:** **Label + Warn-/Hinweissatz** (nicht nur Label).
- **Generisch:** ein konfigurierbarer Label-Wert pro Deployment (Test heute, Pilot später); Prod = leer = kein Banner.
- **WICHTIG — Wording-Korrektur (Owner):** Der Hinweistext darf **nicht pauschal „keine Produktivdaten"** behaupten. Weil die Test-Zone am **Prod-Core** hängt, wirken Core-Operationen (Import/Sync) auf echte Daten. Deshalb ist auch der **Hinweistext konfigurierbar**, damit jede Umgebung die tatsächliche Lage akkurat beschreibt.

## User Stories

1. Als **EEG-Admin** möchte ich auf den ersten Blick sehen, in welcher Umgebung ich bin, damit ich nicht versehentlich in der Test-Umgebung arbeite und dabei reale Core-Aktionen auslöse.
2. Als **Plattform-Betreiber** möchte ich pro Deployment einen Umgebungs-Label + einen akkuraten Warnhinweis setzen können, damit der Banner die tatsächliche Risiko-Lage (z. B. „Test-Onboarding, aber Prod-Core") korrekt wiedergibt.
3. Als **Tester/Mitglied** am öffentlichen Test-Registrierungslink möchte ich erkennen, dass dies keine echte/produktive Seite ist.
4. Als **Plattform-Betreiber** möchte ich in Prod **keinen** Banner sehen — die Abwesenheit eines Labels reicht als „dies ist Prod".

## Acceptance Criteria

- **AC-1:** Ist ein Umgebungs-Label konfiguriert (nicht leer), zeigt die App einen **globalen, dauerhaften, nicht schließbaren** Banner oben — im **Admin-Bereich** und im **Public-Registrierungsformular**.
- **AC-2:** Ist **kein** Label konfiguriert (Prod), erscheint **kein** Banner und kein Layout-Shift.
- **AC-3:** Der Banner zeigt den konfigurierten **Label** (z. B. „TEST") **und** einen **konfigurierten Hinweistext**. Beide sind pro Deployment setzbar.
- **AC-4:** Der Hinweistext ist **nicht** auf „keine Produktivdaten" fixiert. Für die aktuelle Test-Zone (Prod-Core) beschreibt der konfigurierte Default sinngemäß: *Onboarding-Daten sind Test, aber Core-Operationen (Import/Sync) treffen das echte Faktura-System.*
- **AC-5:** Gestaltung auffällig, aber nicht-alarmierend (z. B. amber/orange), klar abgesetzt vom Inhalt und vom PROJ-115-Free-Banner; **responsive** (ab 375px lesbar), überlagert keine Bedienelemente.
- **AC-6:** Generisch über Umgebungen — Sichtbarkeit + Texte allein über Config-Werte, **kein Code-Change pro Umgebung**.

## Edge Cases

- **EC-1:** Label gesetzt, Hinweistext leer → Banner zeigt nur den Label (Hinweistext optional).
- **EC-2:** Sehr langer Label/Hinweistext → Banner bricht sauber um, bleibt lesbar, verdeckt keine Navigation.
- **EC-3:** Admin **und** Public gleichzeitig erreichbar → Banner an beiden Stellen konsistent.
- **EC-4:** Free-Banner (PROJ-115) **und** Umgebungs-Banner gleichzeitig (Rechnungen-Tab einer Test-EEG) → beide lesbar, keine Überlappung (Umgebungs-Banner global oben, Free-Banner im Tab-Inhalt).
- **EC-5:** Prod (kein Label) → garantiert kein Banner.
- **EC-6:** Admin- vs. Public-Publikum: der Admin-Hinweis (Core-Risiko) ist Fachsprache; fürs Public-Form ggf. eine einfachere Formulierung sinnvoll → siehe offener Punkt.

## Non-Goals

- Keine **funktionale** Sperre (z. B. Import in Test blockieren) — nur optisch.
- Keine umgebungsabhängige Logik außer der Anzeige.
- Keine Auth-/Tenant-/Import-Änderung.
- **Keine DB-Migration** — reine Config-Werte, kein DB-Feld.

## Dependencies

- Profitiert vom **Zonen-Konzept** (PROJ-116), aber **unabhängig** baubar/deploybar.
- Unabhängig von PROJ-115.

## Offene Punkte für /architecture (bzw. /grill-me)

- **Runtime-Config-Mechanismus (Haupt-Branch):** `NEXT_PUBLIC_*`-Build-Time-Vars gehen **nicht**, weil dasselbe Docker-Image in Test **und** Prod läuft (nur Helm-Values unterscheiden sich). Label + Hinweistext müssen **Runtime**-Config sein — z. B. Backend exponiert sie (bestehende Public-Config-/Health-Route erweitern oder neue schlanke Route) und Frontend liest sie; alternativ Next.js server-side Runtime-Env im Layout. Zu klären in /architecture.
- **Admin- vs. Public-Wording (EC-6):** ein gemeinsamer Hinweistext für beide, oder zwei Werte (Admin = Core-Risiko-Hinweis, Public = einfacher „Testseite")? Empfehlung: ein Label + getrennter Admin-/Public-Hinweistext, falls günstig; sonst ein neutraler gemeinsamer Text.
- **Helm:** neue Werte (z. B. `environmentLabel`, `environmentNotice[Admin|Public]`) → `values.yaml` + `values-env.yaml.example` im selben Commit (feedback_helm_values_split).

## Tech Design (Solution Architect)

### Entscheidung Runtime-Config-Mechanismus

Erkundung der Codebase: Das **Admin-Layout** (und das Root-Layout) sind **Server-Komponenten**, die heute schon Laufzeit-Umgebungswerte direkt lesen (z. B. den Core-Auth-Modus, den Keycloak-Aussteller) und an die Oberfläche durchreichen. **Genau dieser Weg** trägt auch den Umgebungs-Label:

- **Gewählt:** Der Umgebungs-Label + Hinweistext kommen als **Laufzeit-Umgebungswerte** (gesetzt pro Deployment über Helm) und werden **serverseitig beim Rendern** gelesen. **Kein** Backend-Endpoint, **kein** Datenbankzugriff, **kein** `NEXT_PUBLIC_*` (das würde den Wert ins Image einbacken — dasselbe Image läuft in Test und Prod, ginge also nicht).
- **Wichtige Feinheit:** Manche reinen Info-Seiten (Datenschutz, AGB, AVV) werden **statisch vorab** erzeugt — dort würde ein Umgebungswert zur Bau-Zeit eingefroren. Deshalb sitzt der Banner **nicht** im Root-Layout, sondern in den **dynamisch (pro Aufruf) gerenderten** Oberflächen: dem **Admin-Bereich** und dem **öffentlichen Registrierungsformular**. Beide werden ohnehin bei jedem Aufruf frisch gerendert → der Laufzeitwert greift korrekt. Die wenigen statischen Info-Seiten tragen den Banner nicht — unkritisch (statischer Rechtstext, kein Risiko-Pfad).

### A) Komponenten-Struktur

```
Admin-Bereich (dynamisch)
+-- [NEU] Umgebungs-Banner  (nur wenn Label gesetzt)  ← Hauptrisiko-Pfad (Import → Prod-Core)
+-- bestehende Kopfzeile + Navigation + Inhalt

Öffentliches Registrierungsformular (dynamisch)
+-- [NEU] Umgebungs-Banner  (nur wenn Label gesetzt)
+-- bestehende Public-Seite (Branding-Shell + Formular)

Statische Info-Seiten (Datenschutz/AGB/AVV)  → bewusst KEIN Banner
Prod (kein Label gesetzt)                     → KEIN Banner, überall
```

Eine **einzige Banner-Komponente**, an **zwei** dynamischen Stellen eingehängt.

### B) Daten-Modell (Klartext)

- **Keine Datenbank, keine Migration, kein Tabellenfeld.**
- Zwei **Konfigurationswerte pro Deployment** (Laufzeit-Umgebungswerte, via Helm):
  - **Umgebungs-Label** (z. B. „TEST", „PILOT"). Steuert die Sichtbarkeit: gesetzt → Banner an; leer (Prod) → Banner aus.
  - **Hinweistext** — frei konfigurierbar, damit er die *tatsächliche* Lage beschreibt. Für die heutige Test-Zone (Prod-Core!): sinngemäß „Onboarding-Daten sind Test, aber Import/Core-Aktionen treffen das echte Faktura-System" — **nicht** „keine Produktivdaten".

### C) Frontend-Auswirkung

- Eine neue Banner-Komponente (bestehende UI-Bausteine, auffällig amber, **dauerhaft/nicht schließbar**, responsive ab 375px), klar abgesetzt vom Inhalt **und** vom PROJ-115-Free-Banner (anderer Ort, anderer Zweck).
- Eingehängt im Admin-Layout und an der öffentlichen Registrierungs-Oberfläche. Sichtbar nur, wenn der Label-Wert gesetzt ist.
- **Keine** neuen Pakete, **keine** Formularfelder (keine Placeholder-Thematik).

### D) Admin-/Public-Wording (EC-6)

- **Gewählt:** **ein gemeinsamer, konfigurierter Hinweistext** für beide Oberflächen (ein Wert). Begründung: ein einziger, ehrlich formulierter Satz ist für beide Zielgruppen korrekt (Admin: Import wirkt real; Public: eingegebene Daten können ins echte Core-System gelangen) — das vermeidet zwei Pflege-Stellen und Widersprüche.
- Falls später doch getrennte Texte gewünscht sind, ist ein zweiter Wert (Admin/Public) eine kleine additive Erweiterung — **jetzt nicht** umgesetzt.

### E) Tech-Entscheidungen (WHY)

- **Laufzeit-Umgebungswert statt `NEXT_PUBLIC_`:** dasselbe Docker-Image läuft in Test **und** Prod; ein eingebackener Wert könnte nicht unterscheiden. Serverseitiges Lesen zur Laufzeit löst das — und nutzt ein im Code **bereits etabliertes** Muster.
- **Config statt DB-Feld:** „in welcher Umgebung läuft die Instanz" ist eine Deployment-Eigenschaft, kein fachliches Datum — gehört in die Helm-Konfiguration, nicht in die Datenbank.
- **Konfigurierbarer Hinweistext statt fixem „keine Produktivdaten":** weil die Test-Zone am **Prod-Core** hängt, wäre eine pauschale „keine echten Daten"-Aussage **gefährlich falsch** (ein Import schreibt reale Mitglieder). Der Text muss pro Deployment die echte Lage beschreiben.
- **Banner in dynamischen Oberflächen statt Root-Layout:** verhindert, dass statisch vorerzeugte Seiten den Umgebungswert zur Bau-Zeit einfrieren.

### F) Betroffene Stellen (Full-Chain)

- **Helm:** zwei neue Werte (Label + Hinweistext) in `values.yaml` + `values-env.yaml.example`, plus Durchreichen als Laufzeit-Umgebungswerte in das **Frontend-Deployment** (Frontend-Template). (feedback_helm_values_split — beides im selben Commit.)
- **Frontend:** neue Banner-Komponente · Einhängen im Admin-Layout · Einhängen an der öffentlichen Registrierungs-Oberfläche · Lesen der zwei Laufzeitwerte serverseitig.
- **Kein** Go-Backend-Code, **keine** DB-Migration, **keine** neuen Endpunkte, **keine** Auth-/Tenant-/Import-Änderung.

### G) Dependencies

Keine neuen Pakete. Bestehende UI-Bausteine + das vorhandene Server-seitige-Env-Lese-Muster.

### H) Empfohlene Umsetzungs-Reihenfolge

Reines **`/frontend`** (Banner-Komponente + zwei Layout-Einhängungen + Lesen der Werte) **zusammen mit der Helm-Verdrahtung** (zwei Werte + Frontend-Template). **Kein** `/backend` (Go) nötig. Danach `/qa` → `/deploy` (Owner setzt die Werte pro Umgebung in `values-env.yaml`).

## Implementation Notes (Frontend + Helm, 2026-06-20)

Umgesetzt (kein Go-Backend, keine DB). `tsc` + `vitest` (252) + `npm run build` + `helm lint` grün.

- **Banner-Komponente:** [environment-banner.tsx](../src/components/environment-banner.tsx) — server-kompatible Anzeige (keine Client-Logik), amber-Leiste mit `AlertTriangle`-Icon, fettem Label + Hinweistext, responsive (flex-wrap), dauerhaft. **Rendert `null`, wenn Label leer** (Prod).
- **Admin:** [admin/layout.tsx](../src/app/admin/layout.tsx) — Banner ganz oben im Wrapper, liest `process.env.ENVIRONMENT_LABEL`/`ENVIRONMENT_NOTICE` server-seitig (gleiches Muster wie CORE_AUTH_MODE).
- **Public:** [public-page-shell.tsx](../src/components/public-page-shell.tsx) — Banner oben in der Shell. Shell wird **nur** von der dynamischen Register-Route genutzt (ƒ) → deckt alle 4 Register-Pfade (Erfolg + 3 Fehler) ab, ohne statische Seiten zur Build-Zeit einzubacken.
- **Helm (feedback_helm_values_split, ein Commit):** `values.yaml` (`frontend.environmentLabel`/`environmentNotice` Default `""`) + `values-env.yaml.example` (Test-Zone-Beispiel mit Prod-Core-Hinweis) + `frontend.yaml` (Env-Vars `ENVIRONMENT_LABEL`/`ENVIRONMENT_NOTICE` ins Deployment).
- Build bestätigt: `/register/[rc_number]` bleibt `ƒ` (dynamisch) → Laufzeit-Env greift.

## Nächster Schritt

Frontend + Helm abgeschlossen (2026-06-20). → `/qa` (Banner sichtbar bei gesetztem Label / unsichtbar bei leer; helm lint; responsive) → `/deploy` (ggf. zusammen mit PROJ-115). Owner setzt die Werte pro Umgebung in `values-env.yaml`.
