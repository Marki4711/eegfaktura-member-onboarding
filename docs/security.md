# Security Documentation
## eegfaktura Member Onboarding

## Overview

This document describes the security approach for the eegfaktura Member Onboarding service.
It covers automated scanning tools, the development security workflow, and the boundaries of
what automated scanning can and cannot guarantee.

---

## Security Tooling

### GitHub-native Security Features

The repository uses the following GitHub security features:

| Feature | Purpose |
|---------|---------|
| **Dependabot Alerts** | Known CVEs in npm and Go module dependencies |
| **Dependabot Security Updates** | Automatic PRs for security-relevant dependency updates |
| **Dependabot Version Updates** | Weekly PRs for dependency version updates (`.github/dependabot.yml`) |
| **Secret Scanning** | Detects accidentally committed tokens, API keys, passwords |
| **Push Protection** | Blocks pushes containing detected secrets |
| **Code Scanning / CodeQL** | Static analysis for security-relevant code patterns |

### Snyk (Ergänzender Scanner)

Snyk wird als ergänzender Scanner für folgende Bereiche eingesetzt:

| Scan-Typ | Ziel | Befehle |
|----------|------|---------|
| Open Source | Dependency-CVEs (npm + Go) | `snyk test` |
| Code (SAST) | First-party code, XSS, Injection | `snyk code test` |
| IaC | Helm-Templates, Kubernetes-YAML | `snyk iac test helm/` |
| Container | Docker-Images, Base-Layer-CVEs | `snyk container test` |

### Abdeckung der Finding-Klassen

| Finding-Klasse | Abgedeckt durch |
|---------------|-----------------|
| Dependency-CVEs (npm) | Dependabot, npm audit, Snyk OSS |
| Dependency-CVEs (Go) | Dependabot, govulncheck, Snyk OSS |
| Veraltete Base Images | Wöchentliche GitHub-Actions-Rebuilds, Snyk Container |
| Kompromittierte Base Images (Supply Chain) | Trivy-Scan zwischen Build und Push (exit-code 1 bei CRITICAL) |
| Kompromittierte GitHub Actions (Supply Chain) | SHA-Pinning aller Actions + Dependabot (github-actions, weekly) |
| Secrets im Code | GitHub Secret Scanning, Push Protection |
| XSS / Injection (SAST) | Snyk Code, CodeQL |
| Container läuft als root | Snyk IaC, manuelle Helm-Review |
| Docker-/IaC-Misconfigurations | Snyk IaC, helm lint, kubeconform |
| Auth/Authz-Fehler | Manuelles Code-Review, `/security-review` |
| Mandantentrennung | Manuelles Code-Review, `/security-review` |
| Business-Logik-Fehler | QA (`/qa`), manuelles Review |

**Wichtig:** Automatisierte Scanner können Auth-, Rollen/Rechte-, Business-Logik- und
Mandantentrennungs-Fehler nicht vollständig erkennen. Diese erfordern manuelles Review.

---

## Security-Workflow

### Für neue Features

```
Implementierung
    ↓
/qa
  - Acceptance Criteria
  - Regression Tests
  - E2E Tests
  - Security Smoke Test (offensichtliche Findings)
    ↓
/security-review  ← erforderlich bei sicherheitssensitiven Änderungen
  - Auth/Authz Review
  - Tenant-Isolations-Prüfung
  - Input Validation
  - Infrastructure Review
  - Scanner-Ergebnisse
  - Finale Freigabe
    ↓
/deploy
```

### Für Security-Findings (Dependabot / Snyk / Scanner)

1. Finding erkennen (Dependabot PR, Snyk-Alert, Scanner-Report)
2. Auswirkung einschätzen: Ist die vulnerable Code-Path im Produktionseinsatz?
3. Fix-Branch erstellen: `fix/security-<package>-<cve>`
4. Fix minimal-invasiv umsetzen (Patch-Version bevorzugen vor Major-Upgrade)
5. Tests, Lint, Build ausführen
6. Snyk/govulncheck erneut scannen
7. PR erstellen mit Referenz auf CVE
8. Review und Merge

### GitHub Actions SHA-Pinning

Alle Actions in `.github/workflows/` sind auf **vollständige Commit-SHAs** gepinnt, nicht auf Versions-Tags.

**Warum:** Version-Tags sind mutable — ein Angreifer kann einen Tag per force-push auf einen Malware-Commit umleiten (demonstriert durch den Trivy-supply-chain-Angriff vom 2026-03-19, der `aquasecurity/trivy-action` Tags 0.0.1–0.34.2 für ~12 Stunden kompromittierte). SHA-Pins sind content-addressed und können nicht umgeleitet werden.

**Format:**
```yaml
uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6
#                     ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ SHA  ^^^ Tag als Kommentar
```

**Wartung:** Dependabot (`github-actions` Ecosystem, wöchentlich) öffnet automatisch PRs wenn neue Versionen erscheinen. Der PR enthält den neuen SHA und aktualisiert den Tag-Kommentar. Dependabot-PRs für Actions nach Prüfung mergen.

**Neue Actions hinzufügen:**
1. SHA ermitteln: `gh api repos/<owner>/<repo>/git/ref/tags/<tag> --jq '.object.sha'`
2. Bei annotated tags einmal dereferenzieren: SHA auf den Commit-SHA prüfen (nicht Tag-Object-SHA)
3. Im Workflow eintragen: `uses: <owner>/<repo>@<sha> # <tag>`

### False Positives dokumentieren

Wenn ein Finding als False Positive eingestuft wird:
- Begründung in einem Kommentar im PR dokumentieren
- Snyk-Inline-Suppression oder `.snyk`-Datei mit Kommentar (kein stilles Ignorieren)
- Kein pauschales `snyk ignore --all`

---

## Sicherheitssensitive Bereiche

Folgende Bereiche erfordern bei Änderungen dediziertes `/security-review`:

- `internal/http/auth.go` — JWT-Parsing, `IsSuperuser()`
- `internal/http/admin.go` — Tenant-Zugriffskontrolle
- `internal/http/middleware.go` — Rate Limiting, Security Headers
- `internal/application/` — Status-Transitionen, Import-Logik
- `db/migrations/` — Schema-Änderungen
- `helm/` — Kubernetes-Deployment, Secrets
- `Dockerfile*` — Container-Images
- `.github/workflows/` — CI/CD-Pipelines
- `cmd/server/main.go` — Route-Registrierung, Auth-Middleware-Konfiguration

---

## Bekannte Einschränkungen (V1)

Die folgenden Punkte sind als known Issues dokumentiert und werden in nachfolgenden Versionen adressiert:

| Issue | Severity | Status |
|-------|----------|--------|
| next-auth/uuid Moderate-CVEs | Medium | Offen — Fix erfordert Breaking Change (next-auth Downgrade) |
| Go-Version in Dockerfile-Base-Image | Medium | Durch wöchentliche Rebuilds mitigiert |

---

## Lokale Snyk-Nutzung

### Installation und Authentifizierung

```bash
npm install -g snyk
snyk auth          # öffnet Browser für OAuth-Login
```

Token wird lokal gespeichert — **niemals ins Repository committen**.

### Dependency-Scan

```bash
# Go-Module
snyk test --all-projects

# npm
snyk test
```

### SAST (Code-Scan)

```bash
snyk code test
```

### IaC-Scan

Helm-Templates enthalten Go-Template-Syntax (`{{ }}`), die kein valides YAML ist.
Daher müssen die Templates zuerst gerendert werden:

```bash
helm template member-onboarding helm/member-onboarding \
  -f helm/member-onboarding/values.yaml \
  > /tmp/rendered-helm.yaml

snyk iac test /tmp/rendered-helm.yaml
```

Über Snyk MCP in Claude Code erfolgt dies automatisch beim `/security-review`.

### Snyk in Claude Code (MCP)

Snyk kann über das Model Context Protocol (MCP) in Claude Code eingebunden werden.
Die Konfiguration erfolgt lokal über den `claude mcp` Befehl — **nicht ins Repository committen**.

```bash
# Einmalige Einrichtung
npm install -g snyk
snyk auth                                              # Browser-Login
claude mcp add --scope user -t stdio Snyk -- npx -y snyk@latest mcp -t stdio
```

Danach Claude Code neu starten. Prüfen mit: `claude mcp list` → `Snyk: ✓ Connected`

**Hinweis:** `mcpServers` in `settings.json` ist kein unterstütztes Format — ausschließlich `claude mcp add` verwenden.

---

## Manuelle Schritte (einmalig einrichten)

- [x] Snyk-Token lokal via `snyk auth` konfiguriert
- [x] Snyk MCP lokal in Claude Code konfiguriert (`claude mcp add --scope user`)
- [x] Snyk Code (SAST) für Organisation `marki4711` aktiviert
- [x] Dependabot-Alerts aktiviert (`.github/dependabot.yml`)
- [x] GitHub Secret Scanning aktiviert
- [x] GitHub Push Protection aktiviert
- [x] GitHub Dependabot Security Updates aktiviert
- [x] CodeQL aktiviert
- [x] GitHub-Repository auf snyk.io importieren (für CI-Alerts im PR-Flow) — `snyk monitor` läuft auch in CI automatisch
- [x] Snyk-GitHub-Action in `.github/workflows/snyk.yml` ergänzt — SAST + SCA (Go + npm), `SNYK_TOKEN` als GitHub Secret gesetzt

## Dokumentierte False Positives

| Scanner | Regel | Datei | Begründung |
|---------|-------|-------|------------|
| CodeQL | `go/weak-sensitive-data-hashing` | `internal/http/apikey_middleware.go:80` | API-Keys sind hochentropische Zufalls-Token (`moak_` + 32 random chars). SHA-256 ist Standard für Token-Hashing (GitHub, Stripe). Bcrypt ist nur bei niedrig-entropischen Passwörtern relevant. Alert auf GitHub als False Positive dismissed. |
