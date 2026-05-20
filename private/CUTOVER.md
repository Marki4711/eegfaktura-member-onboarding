# Cutover-Checkliste — PROJ-54 Repo-Split

**Voraussetzung:** PROJ-54-Spec gelesen, alle Entscheidungen sind klar
(siehe Abschnitt "Fixierte Entscheidungen" in `features/PROJ-54-*.md`).

**Geschätzte Dauer:** 1,5–2 Tage über 2–3 Tage verteilt.

---

## Schritt 1 — Privates Repo anlegen (½ Tag)

### 1.1 GitHub-Repo erstellen
- [ ] Repo `eegfaktura-member-onboarding-private` auf GitHub anlegen
  - Visibility: **Private**
  - Owner: gleicher Account
  - Keine Initialisierung mit README/.gitignore (wir spiegeln)

### 1.2 Vollständigen Mirror pushen
```powershell
# Aus einem temporären Verzeichnis:
cd c:\tmp
git clone --mirror https://github.com/Marki4711/eegfaktura-member-onboarding.git
cd eegfaktura-member-onboarding.git
git push --mirror https://github.com/Marki4711/eegfaktura-member-onboarding-private.git
```
- [ ] Verifizieren: Branches + Tags sind im privaten Repo sichtbar

### 1.3 Branch-Protection im privaten Repo
- [ ] `main` geschützt: Force-Push verboten, Status-Checks erforderlich
- [ ] Web-UI: Settings → Branches → Add rule

### 1.4 Secrets im privaten Repo setzen
- [ ] `SNYK_TOKEN` aus altem Repo kopieren
- [ ] `PUBLIC_REPO_TOKEN` neu anlegen (PAT mit `repo`-Scope, schreibt auf
      `eegfaktura-member-onboarding`) — wird vom Mirror-Workflow gebraucht
- [ ] Sonstige Secrets aus altem Repo nachziehen (Helm-Deploy, Smtp-Tests, …)

---

## Schritt 2 — Lokales Setup umstellen (¼ Tag)

### 2.1 Frischer Clone des privaten Repos
```powershell
cd c:\opt\repos
git clone https://github.com/Marki4711/eegfaktura-member-onboarding-private.git
```

### 2.2 Alten Workspace markieren
```powershell
cd c:\opt\repos
Rename-Item eegfaktura-member-onboarding eegfaktura-member-onboarding-archived
```
- [ ] Nicht löschen — als Read-only-Referenz für eine Woche behalten

### 2.3 VSCode / Tooling
- [ ] VSCode-Recents aktualisieren (neuer Pfad öffnen, alten Pin entfernen)
- [ ] Eventuell `c:\opt\repos\eegfaktura-member-onboarding-private` als
  bevorzugter Workspace setzen

### 2.4 Memory-Files aktualisieren
- [ ] `C:\Users\matth\.claude\projects\c--opt-repos-eegfaktura-member-onboarding\`
  → Projekt-Verzeichnis bleibt (an alten Pfad gebunden) ODER neues
  Projekt-Verzeichnis anlegen und Memories rüber kopieren
- [ ] Falls neues Verzeichnis: alle relativen Bash-/Tooling-Pfade auf
  neuen Workspace prüfen

---

## Schritt 3 — Mirror-Workflow + Hooks aktivieren (½ Tag)

Im **privaten Repo**, lokal:

### 3.1 Staging-Dateien an Zielort verschieben
```powershell
cd c:\opt\repos\eegfaktura-member-onboarding-private

# Workflow
New-Item -ItemType Directory -Force .github\scripts
Move-Item private\workflows\mirror-to-public.yml .github\workflows\

# Filter-Skripte
Move-Item private\scripts\apply-mirror-filter.sh .github\scripts\
Move-Item private\scripts\strip-private-frontmatter.sh .github\scripts\

# Whitelist
Move-Item private\mirror-whitelist.txt .github\

# Git-Hooks
New-Item -ItemType Directory -Force .githooks
Move-Item private\githooks\pre-commit .githooks\
Move-Item private\githooks\pre-push .githooks\

# Cleanup leere private/-Unterverzeichnisse
Remove-Item private\workflows -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item private\scripts -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item private\githooks -Recurse -Force -ErrorAction SilentlyContinue
```

### 3.2 Hooks aktivieren
```powershell
git config core.hooksPath .githooks
```
- [ ] Verifizieren: `git config core.hooksPath` zeigt `.githooks`

### 3.3 Initial-Commit privater Repo
```powershell
git add -A
git commit -m "chore(PROJ-54): Mirror-Workflow + Hooks an Zielort verschoben"
git push origin main
```
- [ ] Workflow läuft im privaten Repo, **bricht ab** beim ersten Push,
  weil `PUBLIC_REPO_TOKEN` ggf. noch nicht final konfiguriert ist —
  Logs prüfen, Token korrigieren

---

## Schritt 4 — Erster Mirror-Lauf (1 h)

### 4.1 Workflow-Lauf prüfen
- [ ] Im privaten Repo: Actions-Tab → Mirror-Workflow Status grün
- [ ] Im öffentlichen Repo: neuer Commit "Mirror sync from private @ <sha>"
  ist erschienen
- [ ] `private/`-Inhalte sind im öffentlichen Repo **nicht** zu sehen

### 4.2 Test mit `visibility: private` Frontmatter
- [ ] Eine Test-Datei `features/proj-99-test.md` anlegen mit Frontmatter
  ```yaml
  ---
  visibility: private
  ---
  ```
- [ ] Committen + pushen
- [ ] Verifizieren: Datei kommt **nicht** im öffentlichen Repo an
- [ ] Test-Datei wieder löschen

### 4.3 Test mit `private/`-Pfad
- [ ] `private/test-canary.md` anlegen mit beliebigem Inhalt
- [ ] Committen + pushen
- [ ] Verifizieren: Datei kommt **nicht** im öffentlichen Repo an
- [ ] Test-Datei wieder löschen

---

## Schritt 5 — Public-Repo nachjustieren (½ h)

### 5.1 README-Hinweis
- [ ] `README.md` im öffentlichen Repo um Hinweis-Block am Anfang ergänzen:
  > Dieses Repo ist ein gefilterter, öffentlicher Auszug eines
  > Member-Onboarding-Systems für österreichische EEGs. Aktive
  > Entwicklung und Betrieb finden im privaten Hauptrepo statt.
  > Issues + Pull Requests werden im Mirror nicht aktiv beobachtet.

### 5.2 Public-Repo-CI deaktivieren
- [ ] `.github/workflows/` im Mirror-Output:
  - Build-Workflow optional ein (als Public-Smoke-Build)
  - Snyk-Workflows aus (laufen nur im privaten Repo)
  - Helm-Deploy-Workflows aus (laufen nur im privaten Repo)
- [ ] Realisiert durch Whitelist-Filter, der nur einen reduzierten Satz
  von `.github/workflows/*.yml` ins Public lässt

### 5.3 Repo-Description anpassen
- [ ] GitHub-UI: Description auf "Public mirror of …" setzen
- [ ] Topic-Tags: "mirror", "read-only" ergänzen

---

## Schritt 6 — Doku-Sweep + Memory-Update (½ Tag)

### 6.1 CLAUDE.md (privater Repo)
- [ ] Hinweis ergänzen, dass `private/` und Frontmatter `visibility: private`
  nicht in den Mirror laufen
- [ ] Repository-Name aktualisieren: `eegfaktura-member-onboarding-private`

### 6.2 docs/operations.md (privater Repo)
- [ ] Mirror-Workflow-Beschreibung
- [ ] Recovery-Pfad bei Mirror-Bruch
- [ ] Hook-Aktivierung beschreiben

### 6.3 Memory-Files
- [ ] Neues Memory anlegen: `project_repo_split.md` mit:
  - Datum des Cutovers
  - Privater Repo: `eegfaktura-member-onboarding-private`
  - Public Repo: `eegfaktura-member-onboarding` (Mirror, read-only)
  - Workspace-Pfad: `c:\opt\repos\eegfaktura-member-onboarding-private`
  - Mirror-Workflow-Verhalten

### 6.4 PROJ-54 Status updaten
- [ ] `features/PROJ-54-*.md`: Status → Deployed, Deployed-Datum eintragen
- [ ] `features/INDEX.md`: Status → Deployed

---

## Schritt 7 — Saubere Übergabe (¼ Tag)

### 7.1 Erstes "echtes" privates Commit
- [ ] `private/pricing/.gitkeep` anlegen + Initial-Inhalt
  (z. B. die Pricing-Strategie aus dem Memory hierher migrieren)
- [ ] Committen, pushen, Mirror-Lauf verifizieren: kein Leak

### 7.2 Cleanup
- [ ] Falls `private/CUTOVER.md` (diese Datei) im Cutover-Repo-Stand
  hängenbleibt: entscheiden ob umbenennen `private/operations/cutover.md`
  als historisches Dokument oder löschen
- [ ] Alter Workspace `c:\opt\repos\eegfaktura-member-onboarding-archived`
  nach einer Woche löschen, wenn keine Probleme auftraten

### 7.3 Rollback bereithalten
Falls in den ersten Tagen Probleme auftreten:
- Mirror-Workflow im privaten Repo manuell deaktivieren
- Public-Repo auf Pre-Cutover-Commit zurücksetzen (`archive`-Tag setzen)
- Wieder direkt am alten Public-Workspace arbeiten, bis Problem analysiert
