# PROJ-54 — Aufteilung in öffentliches Schaufenster + privates Hauptrepo

**Status:** Deployed
**Erstellt:** 2026-05-20
**Cutover:** 2026-05-20
**Deployed:** 2026-05-20
**Abhängigkeiten:** keine

---

## Ziel

Das Projekt zieht aus einem einzelnen öffentlichen Repo in ein **zweistufiges
Setup** um:

- **Privates Hauptrepo** — Wahrheitsquelle, enthält alles (Code, Geschäftslogik,
  Verträge, Pricing, Pen-Test-Reports, DPIA, AVV-Templates, Anbieter-Setup-Doku,
  Billing-Module, Mahn-Logik).
- **Öffentliches Schaufenster-Repo** — gefilterter Read-only-Mirror eines
  definierten Whitelist-Sets (Kern-Code, generische Docs, Migrations, CI ohne
  Secret-Pfade). Behält die Open-Source-Sichtbarkeit, schließt aber sensible
  Bereiche aus.

Vor dem Prod-Cutover muss das Setup stehen, damit die in den kommenden
Wochen ohnehin entstehenden sensiblen Artefakte (Pen-Test-Report, DPIA,
Zahlungsdienstleister-Setup, Rechnungsmodul, Vertragstexte) sicher gelagert
werden, ohne dass sie nachträglich aus Git-Historie entfernt werden müssen.

## Hintergrund

Aktueller Zustand: ein öffentlicher Repo
`github.com/gemeinstrom/eegfaktura-member-onboarding`. Bisher wurde die
Sensibilitätsgrenze über Disziplin (`MEMORY.md`-Regel "Pricing/Verträge nicht
in docs/CHANGELOG/PROJ-Specs") eingehalten. Das skaliert nicht mehr, sobald:

1. **Zahlungsdienstleister angebunden wird** — Webhook-Endpoints,
   Anbieter-spezifische Setup-Schritte, ggf. API-Keys in Setup-Dokumentation
2. **Eigenes Rechnungsmodul gebaut wird** (laut Pricing-Strategie) —
   Vertrags-/Tarif-/Kunden-Stammdaten, Mahn-Logik, Rechnungstemplates mit
   Anbieter-Daten
3. **Pen-Test-Report eingeholt wird** (vor Prod-Cutover, siehe Audit-Plan) —
   Findings + Fix-Trail soll nicht öffentlich nachlesbar sein
4. **DSGVO-DPIA + AVV-Templates** entstehen — reale Anbieter-Namen,
   juristische Texte, ggf. Vertrags-IDs
5. **Audit-Logs / Incident-Postmortems** anfallen — sollen referenzierbar,
   aber nicht öffentlich sein

Diese Artefakte sind in den kommenden 4–8 Wochen fällig. Wenn der Split
nachher passiert, müssen Inhalte aus Public-Git-Historie entfernt werden
(`git filter-repo`), was Risiko und Aufwand erzeugt. **Jetzt** ist der
kostengünstigste Zeitpunkt.

## Optionen-Vergleich (Vorab-Analyse)

| Option | Beschreibung | Pro | Contra |
|---|---|---|---|
| **A: Submodule** | Public-Repo bleibt, Sensibles in eigenes privates Repo + Submodul | Open-Source-Historie erhalten | Submodule sind brittle, leichter Slip möglich |
| **B: Public Mirror** | Privater Repo = Wahrheit; Public ist gefilterter Mirror via CI | Klare Trennung, schwer kaputt zu kriegen, audit-friendly | Mehr CI-Arbeit, Public hinkt einen Push-Zyklus hinterher |
| **C: Komplett privat** | Bestehender Repo wird auf privat umgestellt | Einfachste Lösung | Verliert Open-Source-Story; Snyk-Free-Tier-Druck (haben wir aber schon entschärft via Push/Wochen-Split) |
| **D: Status quo + Disziplin** | Kein Split, nur strengere Doku-Regeln + Pre-Commit-Hooks | Null Migration | Single Slip → permanent in Public-Historie, irreversibel |

**Entscheidung: Option B** (Public Mirror). Begründung: schwer kaputt zu
kriegen, klar definierbares Whitelist-Set, Open-Source-Geschichte bleibt
sichtbar, Mirror-Lag von wenigen Minuten ist akzeptabel.

### Fixierte Entscheidungen (2026-05-20)

- **Repo-Name (privat):** `eegfaktura-member-onboarding-private`
- **Historien-Strategie:** (a) Public-Historie bleibt unverändert; ab Cutover
  werden nur gefilterte Commits angehängt
- **Marker-Strategie:** beides — `private/`-Verzeichnis für ganze Bereiche +
  Frontmatter `visibility: private` für einzelne gemischte Dateien
- **Security-Scans (Snyk/Trivy/Dependabot):** nur im privaten Repo
- **Pre-Commit-Hook:** ja, im privaten Repo bauen (defensive Schicht zusätzlich
  zum Mirror-Filter)
- **Lokaler Workspace-Pfad:** Umzug auf
  `c:\opt\repos\eegfaktura-member-onboarding-private`
- **Filter-Fehler-Verhalten:** Mirror-Workflow bricht ab, kein Push; CI alarmiert
- **Bestehende Specs:** bleiben unverändert public (keine retrospektive
  Markierung)
- **Zeitpunkt:** sofort (diese Woche), vor Anbindung Zahlungsdienstleister
  und vor Pen-Test-Beauftragung

## Scope

### Repo-Topologie

1. **Neues privates Repo anlegen**, z. B.
   `github.com/gemeinstrom/eegfaktura-member-onboarding-private` (Name in offenen
   Fragen).
2. **Vollständige Spiegelung** des aktuellen Public-Repos (`git clone --mirror`)
   ins neue private Repo — inklusive Historie + Branches + Tags.
3. **Public-Repo bleibt bestehen**, wird aber zum Mirror-Ziel. Beim ersten
   Cutover wird `main` durch einen frischen, gefilterten Stand ersetzt
   (Historie wird gerade gehalten oder behalten — Entscheidung in offenen
   Fragen).

### Whitelist-Definition (was bleibt im Public)

Strict Whitelist, nichts läuft "von allein" rüber. Default: privat. Default
für **public** ist nur das, was hier explizit aufgelistet ist:

```
cmd/
internal/
  ├── application/        (Kern-Domain — public)
  ├── coreclient/         (Core-Integration — public)
  ├── http/               (Handler — public)
  ├── importing/          (Import-Logik — public)
  ├── mail/               (Mailing — public, Templates ohne Anbieter-Daten)
  ├── meteringpoint/      (Domain — public)
  ├── pdf/                (PDF-Generator — public)
  ├── shared/             (gemeinsam — public)
  └── statuslog/          (public)
src/                     (Frontend — public)
db/migrations/           (Schema — public)
docs/
  ├── architecture.md
  ├── api-spec.md
  ├── domain-model.md
  ├── import-mapping.md
  ├── operations.md
  ├── security.md         (ohne Anbieter-Namen / Vertrags-Details)
  ├── PRD.md
  └── user-guide/         (öffentliche Doku)
features/                (PROJ-Specs — public, sofern nicht business-sensitive)
.github/workflows/       (CI ohne Secret-Pfade — Workflows ja, Secrets im GH-UI)
helm/                    (Helm-Charts — public, Werte über values.yaml-Defaults)
README.md
CLAUDE.md
.claude/rules/
.claude/skills/
package.json, package-lock.json, go.mod, go.sum, tsconfig.json, …
.dockerignore, .gitignore, .editorconfig, …
LICENSE
```

### Blacklist (bleibt nur im privaten Repo)

```
private/
  ├── billing/            (Rechnungsmodul, sobald gebaut)
  ├── contracts/          (Vertrags-Templates AVV, EEG-Verträge)
  ├── dpia/               (DSGVO-Folgenabschätzung + VVT)
  ├── pricing/            (Tarif-Modelle, Kalkulationen)
  ├── pentest/            (Pen-Test-Reports + Fix-Trail)
  ├── postmortems/        (Incident-Berichte)
  ├── runbooks/           (Operationelle Runbooks mit Anbieter-Daten)
  └── vendor-setup/       (PSP-Setup-Anleitungen mit echten Daten)
features/PROJ-XX-*.md    (business-sensitive Specs — Markierung via Frontmatter)
docs/internal/           (interne Doku-Sammelplatz)
```

Sensible Spec-Markierung: Frontmatter-Feld `visibility: private` in der
`.md`-Datei. Der Mirror-Workflow filtert diese gezielt raus.

### Mirror-Workflow

GitHub Action im privaten Repo, getriggert auf `push` zu `main`:

1. **Whitelist-Filter anwenden** — `git filter-repo` mit Pfad-Whitelist auf
   einen temporären Branch.
2. **Frontmatter-Filter** — Skript scannt alle `.md` im Filter-Output, entfernt
   Dateien mit `visibility: private` im Frontmatter.
3. **Smoke-Check** — Public-Build (`make build` / `npm run build`) muss auf
   dem gefilterten Output durchlaufen, sonst Abbruch.
4. **Push** auf `main` des Public-Repos (Public-Branch ist Mirror-Output,
   nicht Wahrheits-Quelle). Force-Push nur wenn Historien-Linearität bricht.

**Filter-Fehler-Verhalten:** Wenn Schritt 1, 2 oder 3 fehlschlägt, **bricht
der Workflow ab und der Push wird nicht durchgeführt**. Public bleibt einen
Mirror-Zyklus hinter dem privaten Repo zurück. GitHub-Action-Notification
geht per E-Mail raus. Lieber Mirror-Lag als sensibler Leak.

### Pre-Commit-Hook (defensive Schicht)

Zusätzlich zum Mirror-Filter läuft im privaten Repo ein Pre-Commit-Hook,
der absichert, dass:

- Auf einem Branch namens `mirror/*` oder `public-*` **keine** Pfade aus dem
  Blacklist-Set committet werden können
- Auf `main` kein `private/`-Pfad an einem `git push --force origin public-*`
  hängenbleibt (Pre-Push-Hook ergänzt)

Implementierung: `.githooks/pre-commit` + `.githooks/pre-push`, aktiviert
per `core.hooksPath = .githooks` Repo-Config. Hooks sind versioniert.

### Secrets / CI

- **GitHub-Secrets bleiben pro Repo getrennt.** Beide Repos brauchen eigene
  `SNYK_TOKEN`, `KEYCLOAK_*`, etc. Sekretär-Verteilung wird einmalig pro Repo
  angelegt.
- **CI-Workflows laufen primär im privaten Repo** (Build + Test + Deploy +
  Snyk + Trivy). Public-Repo läuft optional einen reduzierten Smoke-Build
  zur Vitrine.
- **Helm-Deployment-Pipeline** zieht aus dem privaten Repo (dort liegen
  Production-Werte). Public-Repo wird nicht für Deploys verwendet.
- **Dependabot** läuft primär privat; Public-Repo bekommt einen Mirror der
  Updates automatisch beim nächsten Push.

### Doku-Updates

- `README.md` (public): Hinweis "Dies ist der öffentliche Auszug eines
  Member-Onboarding-Systems für österreichische EEGs. Aktive Entwicklung
  und Betrieb finden im privaten Repo statt."
- `CLAUDE.md` (beide): Hinweis auf Whitelist-Regel, dass `private/` und
  Frontmatter `visibility: private` nicht in den Mirror laufen.
- `private/CUTOVER.md` (private-only): Mirror-Workflow-Dokumentation +
  Cutover- und Recovery-Schritte. (Ursprünglich war `docs/operations.md`
  vorgesehen, die bleibt aber bewusst public als Cluster-Runbook ohne
  Mirror-Bezug.)
- `.claude/rules/general.md`: Hinweis ergänzen, dass Pricing/Verträge nur
  unter `private/` landen dürfen.

### Migration / Cutover-Plan

1. **Vorbereitung (½ Tag)**
   - Privates Repo `eegfaktura-member-onboarding-private` anlegen
   - Vollständigen Mirror des Public-Repos darauf pushen (`git clone --mirror`
     + `git push --mirror`)
   - Branch-Protection auf `main` im privaten Repo einrichten
   - Repo-Secrets im privaten Repo neu setzen (SNYK_TOKEN, KEYCLOAK_*,
     SMTP_*, Helm-Deploy-Tokens etc.)
2. **Whitelist-/Mirror-Workflow bauen + testen (½ Tag)**
   - `mirror-whitelist.txt` (Pfad-Liste) anlegen
   - GitHub Action `mirror-to-public.yml` schreiben (Filter + Frontmatter-
     Strip + Smoke-Build + Push)
   - Pre-Commit-/Pre-Push-Hooks in `.githooks/` ablegen + `core.hooksPath`
     dokumentieren
   - Auf einem `cutover/test`-Branch im privaten Repo den Filter laufen
     lassen, Output prüfen
3. **Cutover (1 h)**
   - Letzten direkten Push ins Public-Repo durchführen
   - Public-Repo: Beschreibung anpassen, "this is a mirror"-Hinweis im README
   - Privates Repo: erster echter Mirror-Push, prüfen dass Public-Inhalte
     unverändert sind
   - Public-Repo: alte CI-Workflows deaktivieren oder reduzieren (nur Smoke-
     Build, keine Snyk/Deploy)
4. **Lokales Setup umstellen (½ Tag)**
   - Neuer Clone unter `c:\opt\repos\eegfaktura-member-onboarding-private`
   - Alter Pfad `c:\opt\repos\eegfaktura-member-onboarding` bleibt als
     Read-only-Referenz oder wird archiviert (`-archived`-Suffix)
   - VSCode-Workspace-Recents aktualisieren
   - Hook-Pfad aktivieren: `git config core.hooksPath .githooks`
   - Memory-Files aktualisieren: alle Bash-/Tooling-Pfade auf neuen Pfad
     ziehen (siehe Akzeptanzkriterien)
5. **Erstes "echtes" Commit im privaten Repo** mit `private/`-Pfad
   (z. B. `private/pricing/.gitkeep`) — prüfen, dass:
   - Commit lokal sauber durchläuft
   - Mirror-Workflow läuft, Push auf Public, `private/` taucht **nicht** auf
6. **Doku-Sweep**: README/CLAUDE/operations anpassen, `private/README.md`
   anlegen mit Übersicht was unter `private/` lebt

Geschätzter Gesamtaufwand: **1,5–2 Tage** über 2–3 Tage verteilt.

### Rollback-Plan

Falls der Mirror-Workflow Probleme macht oder Sensibles "lecken" sollte:

- **Sofort-Maßnahme:** Public-Repo auf `archive`-Branch von vor Cutover
  zurücksetzen, Mirror-Workflow deaktivieren.
- **Worst case:** Public-Repo komplett auf "archived" setzen, alle weitere
  Entwicklung nur privat.

## Akzeptanzkriterien

- [ ] Privates Repo `eegfaktura-member-onboarding-private` existiert, enthält
  vollständige Historie + alle Branches + Tags des aktuellen Public-Repos
- [ ] Public-Repo bleibt erreichbar unter bestehender URL, Historie unverändert
- [ ] Mirror-Workflow läuft auf jedem `main`-Push im privaten Repo, dauert
  < 3 min bis zum aktualisierten Public-Stand
- [ ] Whitelist-Filter ist deklarativ in `mirror-whitelist.txt` und nicht im
  Workflow-Skript hardcoded
- [ ] Frontmatter-Marker `visibility: private` in einer `.md`-Datei
  unterdrückt die Datei zuverlässig im Mirror — verifiziert per Test-Datei
- [ ] Smoke-Build auf dem Mirror-Output (`make build`, `npm run build`)
  schlägt nicht fehl; bei Fehlschlag bricht der Mirror-Push ab
- [ ] Pre-Commit-Hook + Pre-Push-Hook im privaten Repo aktiv, verhindern
  versehentliche `private/`-Pfade auf Mirror-Branches
- [ ] CI-Workflows im privaten Repo decken Build + Test + Snyk + Deploy ab
- [ ] Snyk / Trivy / Dependabot laufen **nur** im privaten Repo
- [ ] Helm-Deploy-Pipeline läuft aus privatem Repo
- [ ] README.md im Public erklärt klar, dass das Repo ein Mirror ist
- [ ] `private/`-Verzeichnis existiert im privaten Repo, ist `.gitignore`-frei,
  taucht aber nicht im Public-Mirror auf — verifiziert per Test-Commit
- [ ] Dokumentation des Mirror-Workflows liegt unter `private/CUTOVER.md`
  (nur im privaten Repo) + Recovery-Pfad
- [ ] Lokaler Workspace umgezogen auf
  `c:\opt\repos\eegfaktura-member-onboarding-private`; alter Pfad als
  `-archived` markiert oder gelöscht
- [ ] Memory-Files aktualisiert (Bash-/Tooling-Pfade verweisen auf neuen
  Workspace-Pfad)

## Implementierungs-Notizen

**Cutover am 2026-05-20 abgeschlossen.**

Shipped:
- Privates Repo `gemeinstrom/eegfaktura-member-onboarding-private` (visibility private),
  Public-Repo `gemeinstrom/eegfaktura-member-onboarding` als gefilterter Mirror.
- Mirror-Infrastruktur: `.github/workflows/mirror-to-public.yml`,
  `.github/mirror-whitelist.txt`, `.github/scripts/apply-mirror-filter.sh`,
  `.github/scripts/strip-private-frontmatter.sh`.
- Defensive Hooks: `.githooks/pre-commit`, `.githooks/pre-push`, aktiviert per
  `core.hooksPath = .githooks`.
- Lokaler Workspace umgezogen nach `c:\opt\repos\eegfaktura-member-onboarding-private`,
  alter Pfad als `eegfaktura-member-onboarding-archived` behalten.

Abweichungen vom Spec:
- **Mirror-Doku in `private/CUTOVER.md` statt `docs/operations.md`.**
  `operations.md` bleibt bewusst public — sie dokumentiert den Cluster-Runbook
  (Velero/Ceph/Wasabi, Postgres-Backup, Restore) und hat keinen Bezug zum
  Mirror-Workflow. Mirror-Cutover-/Recovery-Schritte leben unter `private/CUTOVER.md`.
- **Public-Mirror hat keinerlei `.github/workflows/`** (kein Smoke-Build, kein
  CodeQL via Workflow). Begründung: PAT-Scope minimal halten (kein `workflow`-
  Permission nötig), keine doppelten Findings, keine Dependabot-Loops gegen
  einen Mirror, der beim nächsten Sync überschrieben würde. CI/Snyk/EOL-Check/
  Docker-Publish/Dependabot laufen ausschließlich privat.

Stolperfallen während Cutover (für Retrospektive):
- `set -e` + `((counter++))` im Filter-Skript bricht ab, sobald der Counter 0
  ist — Fix in 04ccc85.
- Erste Whitelist-Iteration zog `ci.yml` mit ins Public; korrigiert in 6c06c3d /
  08ba8f5.
