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
| Container laufen als root (kein `securityContext`) | High | Behoben — `runAsNonRoot: true`, `runAsUser: 1000` in Helm-Templates |
| `allowPrivilegeEscalation` nicht gesetzt | Medium | Behoben — `allowPrivilegeEscalation: false` in allen Containers |
| `readOnlyRootFilesystem` nicht gesetzt (Backend) | Medium | Behoben — `readOnlyRootFilesystem: true` für Backend + Migrate |
| Next.js CVEs (npm audit) | Medium | Offen — `npm audit fix` ausstehend |
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

```bash
snyk iac test helm/
snyk iac test Dockerfile.backend
snyk iac test Dockerfile.frontend
```

### Snyk in Claude Code (MCP)

Snyk kann über das Model Context Protocol (MCP) in Claude Code eingebunden werden.
Die Konfiguration erfolgt lokal — **nicht ins Repository committen**.

Beispiel für die lokale Claude-Code-Konfiguration (`~/.claude/settings.json` oder projektlokal):

```json
{
  "mcpServers": {
    "Snyk": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "snyk@latest", "mcp", "-t", "stdio"],
      "env": {}
    }
  }
}
```

Der Snyk-Token muss in der lokalen Umgebung (`SNYK_TOKEN`) oder via `snyk auth` konfiguriert sein.

---

## Manuelle Schritte (einmalig einrichten)

- [ ] Snyk-Konto erstellen und GitHub-Repository importieren (snyk.io)
- [ ] Snyk-Token lokal via `snyk auth` konfigurieren
- [ ] Snyk MCP lokal in Claude Code konfigurieren (s. o.)
- [ ] GitHub Security Features im Repository aktivieren (Settings → Security)
- [ ] Dependabot-Alerts aktivieren
- [ ] Secret Scanning und Push Protection aktivieren
- [ ] Optional: Snyk-GitHub-Action in `.github/workflows/` ergänzen (empfohlen für CI-Gate)
