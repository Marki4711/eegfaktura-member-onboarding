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

### Ergänzende Scanner (Solo-Dev-Stack seit 2026-06-06)

Snyk wurde am 2026-06-06 abgelöst — der Mindesttarif von 5 Seats passt nicht
zum aktuellen Solo-Setup. Die vier Snyk-Module sind durch kostenlose Tools
ersetzt; der Rückstieg ist mit der Backup-Konfig unter
`private/snyk-restore/` jederzeit möglich.

| Scan-Typ | Ziel | Tool | Wo |
|---|---|---|---|
| **SAST (Multi-Lang)** | First-party code: XSS, Injection, Secrets-im-Code | **Semgrep CE** | `security-scan.yml` |
| **SAST (Go-spezifisch)** | Go-Patterns: schwache Crypto, unsichere File-Modes, fehlende Error-Checks | **gosec** | `security-scan.yml` |
| **Open Source (Go)** | CVEs in Go-Dependencies + Stdlib-Reachability | **govulncheck** + Dependabot (gomod) | `ci.yml` + `dependabot.yml` |
| **Open Source (npm)** | CVEs in npm-Dependencies | **npm audit** + Dependabot (npm) | `ci.yml` + `dependabot.yml` |
| **IaC** | Helm-Templates, Dockerfiles, K8s-Manifeste | **Trivy config-scan** | `security-scan.yml` |
| **Container** | Base-Image-Layer-CVEs vor Push in die Registry | **Trivy image-scan** | `docker-publish.yml` (existiert bereits) |

**Workflow-Trigger** (`security-scan.yml`):
- Push auf `main` — SAST-Analogie zum alten Snyk-Code-Push-Trigger
- Pull-Request — Vorab-Sicht auf Findings, kein Build-Fail
- Cron Sonntags 04:00 UTC — Drift-Erkennung bei unverändertem Code
- `workflow_dispatch` — manuelles On-Demand-Run

**Findings landen in der GitHub Security-Tab** über SARIF-Upload —
zentralisierte Triage statt verstreute Tool-spezifische Reports.

**Rückstieg auf Snyk** (falls Team auf >5 Seats wächst oder spezifische
Snyk-Findings vermisst werden): siehe `private/snyk-restore/README.md`.

### Abdeckung der Finding-Klassen

| Finding-Klasse | Abgedeckt durch |
|---------------|-----------------|
| Dependency-CVEs (npm) | Dependabot, npm audit |
| Dependency-CVEs (Go) | Dependabot, govulncheck |
| Veraltete Base Images | Wöchentliche GitHub-Actions-Rebuilds, Trivy image-scan |
| Kompromittierte Base Images (Supply Chain) | Trivy-Scan zwischen Build und Push (exit-code 1 bei CRITICAL) |
| Kompromittierte GitHub Actions (Supply Chain) | SHA-Pinning aller Actions + Dependabot (github-actions, weekly) |
| Secrets im Code | GitHub Secret Scanning, Push Protection, Semgrep `p/secrets` |
| XSS / Injection (SAST) | Semgrep, gosec, CodeQL |
| Container läuft als root | Trivy IaC, manuelle Helm-Review |
| Docker-/IaC-Misconfigurations | Trivy IaC, helm lint, kubeconform |
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

### Für Security-Findings (Dependabot / Scanner)

1. Finding erkennen (Dependabot PR, GitHub Security-Tab, Scanner-Report)
2. Auswirkung einschätzen: Ist der vulnerable Code-Path im Produktionseinsatz?
3. Fix-Branch erstellen: `fix/security-<package>-<cve>`
4. Fix minimal-invasiv umsetzen (Patch-Version bevorzugen vor Major-Upgrade)
5. Tests, Lint, Build ausführen
6. govulncheck / npm audit / Semgrep erneut scannen
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
- Tool-spezifische Suppression mit Begründungs-Kommentar im Code (Semgrep: `// nosemgrep: <rule>`, gosec: `// #nosec G<code>`)
- Kein pauschales Ignorieren ganzer Regeln/Verzeichnisse ohne dokumentierte Begründung

---

## Sicherheitssensitive Bereiche

Folgende Bereiche erfordern bei Änderungen dediziertes `/security-review`:

- `internal/http/auth.go` — JWT-Parsing, `IsSuperuser()`
- `internal/http/admin.go` + `internal/http/admin_*.go` + `internal/http/tenant.go` — Tenant-Zugriffskontrolle. Seit PROJ-107 (2026-06-13) ist `admin.go` als reiner Constructor-Hub auf 281 Zeilen reduziert; die 48 Public-Handler leben in 17 Domain-Files (`admin_external_keys.go`, `admin_legal_documents.go`, `admin_attachments.go`, `admin_members.go`, `admin_settings_*.go`, `admin_applications*.go`, `admin_reconciliation.go`, `admin_entrypoints.go`, `admin_helpers.go`). `containsRC` ist Single-Source in `tenant.go` mit `tenant_test.go` (8 Vektoren inkl. nil-Slice-Bypass-Sicherung). Cross-Cutting (`parseRCAndCheck`, `checkTenantAccess`, `enforceCustomerContract*`) bleibt in `admin.go`.
- `internal/http/middleware.go` — Rate Limiting, Security Headers
- `internal/application/` — Status-Transitionen, Import-Logik, Post-Import-Übergänge (PROJ-46), Reset-Import-Erweiterung, SEPA-Mandat-Timing-Branch (PROJ-48), Zählpunkt-Prefix-Match-Validation beim Submit (PROJ-52: `validateMeteringPointPrefixMatch` — pro-Richtung HasPrefix-Check gegen `registration_entrypoint.metering_point_prefix_*`; defense-in-depth zur Frontend-Mask), manueller `approved → activated`-Skip (PROJ-53: `MarkActivatedSkipImport` mit Status-Vor­bedingungs-Check + Mitgliedsnummer-Pflicht + Local-Uniqueness-Check)
- `internal/application/email_confirmation.go` — Token-Erzeugung und -Hashing (PROJ-31)
- `internal/importing/` — Core-Calls (POST /participant, GET /participant für Activation-Check inkl. PROJ-53 Modus-Switch A/B), Auto-Branch nach Import
- `internal/mail/` — Mail-Templates für PROJ-46/PROJ-53 (`application_imported_*` jetzt Mandat-only, `application_activated_*` jetzt volle Beitrittsbestätigung; neues `application_activated_eeg.html` für EEG-Kopie), PROJ-47 (B2B-Mandat-Anhang) und PROJ-48 (CORE-Mandat-Anhang am Import-Zeitpunkt + B2B-Hinweis im Submit-Mail). Neue `SendActivationNotification` ist idempotent via `application.activation_notification_sent_at`
- `db/migrations/` — Schema-Änderungen, insbes. CHECK-Constraint-Erweiterung bei neuen Status-Werten (zuletzt 000048 für `activation_mode`-Enum)
- `helm/` — Kubernetes-Deployment, Secrets
- `Dockerfile*` — Container-Images
- `.github/workflows/` — CI/CD-Pipelines (inkl. `eol-check.yml` für proaktive Runtime-EOL-Warnung)
- `cmd/server/main.go` — Route-Registrierung, Auth-Middleware-Konfiguration, neue Endpoints (`/check-activation`, `/mark-activated`)

**Runtime-Hygiene:**
- Frontend-Image läuft auf **Node 22 LTS** (Node 20 ist seit 2026-04-30 EOL; Bump vom 2026-05-17). Bei nächstem Runtime-Wechsel den `cycle`-Eintrag in `.github/workflows/eol-check.yml` und den `@types/node`-Ignore-Filter in `.github/dependabot.yml` nachziehen.
- Monatlicher EOL-Check-Workflow (`.github/workflows/eol-check.yml`) fragt endoflife.date für Node / Go / PostgreSQL und öffnet ein GitHub-Issue, sobald eine Komponente innerhalb von 60 Tagen EOL erreicht.
- Trivy-Image-Scan im Docker-Publish-Workflow deckt die Backend- und Frontend-Images vor dem Push in die Registry ab (`exit-code 1` bei CRITICAL); Trivy IaC im `security-scan.yml` scannt die Dockerfile-Konfiguration selbst (HIGH+CRITICAL).

---

## E-Mail-Bestätigung (PROJ-31)

Anti-Abuse-Mechanismus gegen Junk-Anträge mit fremder E-Mail-Adresse. Aktivierbar per EEG-Setting `require_email_confirmation`.

**Token-Handling:**
- 32 Byte Zufall, base64url-codiert (≥256 Bit Entropie); Plaintext nur in der ausgehenden Mail
- DB speichert ausschließlich den SHA-256-Hash (`application.email_confirmation_token_hash`)
- Lieferung im URL-Fragment (`/confirm-email#<token>`) — Browser sendet Fragmente nie an Server, kein Server-Access-Log enthält den Token
- Frontend strippt den Token nach Lesen aus der Adresszeile (`replaceState`) und postet ihn ins Backend
- `Referrer-Policy: no-referrer` auf der Confirm-Seite blockt jegliches Token-Leak via Referer
- Lebensdauer 30 Tage, single-use (idempotenter Re-Click zeigt „bereits bestätigt")
- Token-Rotation bei Resend (alter Token wird sofort invalidiert)
- Generic 400 für „ungültig oder abgelaufen", damit Angreifer „existiert nicht" nicht von „abgelaufen" unterscheiden können

**Auto-Reject-Hintergrundjob:**
- Läuft alle 6 Stunden in jedem Backend-Pod (`internal/application/auto_reject.go`)
- Überträgt abgelaufene `submitted`-Anträge auf `rejected` mit System-Reason
- Verhindert dauerhaft „hängende" Anträge, die nie bestätigt wurden
- Idempotent über `WHERE status=$expected`-Guard — Daten-safe bei parallelen Pods (kosmetisch zählt nur die Telemetrie doppelt)

**Rate Limiting:**
- `POST /api/public/applications` ist auf 10 Requests / 10 Minuten pro IP begrenzt
- `POST /api/public/applications/confirm-email` hat seit 2026-05-19 **einen eigenen, deutlich großzügigeren Bucket** (30 Requests / 1 Minute pro IP, separater Bucket-Map). Begründung: das 32-Byte-Token macht Brute-Force astronomisch — der Limit ist reine Defence-in-Depth — und der frühere geteilte Bucket mit `/applications` löste „Zu viele Einreichungen"-Fehler bei Testern/Tester-Setups aus. Eigene Fehlermeldung „Zu viele Bestätigungsversuche".
- Admin-`resend-email-confirmation` hat eine per-application-Throttle, damit ein Admin nicht in Schleife Mails versendet

---

## Bekannte Einschränkungen (V1)

Die folgenden Punkte sind als known Issues dokumentiert und werden in nachfolgenden Versionen adressiert:

| Issue | Severity | Status |
|-------|----------|--------|
| next-auth/uuid Moderate-CVEs | Medium | Offen — Fix erfordert Breaking Change (next-auth Downgrade) |
| Go-Version in Dockerfile-Base-Image | Medium | Durch wöchentliche Rebuilds mitigiert |

---

## Lokale Scanner-Nutzung

### Go-Stdlib + Module CVE-Scan

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

### npm-Dependency-Scan

```bash
npm audit --audit-level=high
```

### SAST (Go-spezifisch) — gosec

```bash
go install github.com/securego/gosec/v2/cmd/gosec@latest
gosec -severity medium -confidence medium ./...
```

### SAST (Multi-Language) — Semgrep

```bash
# CLI-Install (einmalig)
python -m pip install semgrep   # ODER: brew install semgrep

# Voller Scan mit den Regel-Paketen aus security-scan.yml
semgrep scan \
  --config=p/security-audit \
  --config=p/owasp-top-ten \
  --config=p/golang \
  --config=p/typescript \
  --config=p/react \
  --config=p/secrets \
  --severity=WARNING --severity=ERROR \
  --metrics=off
```

### IaC-Scan — Trivy

```bash
# Trivy installieren (einmalig)
brew install trivy   # oder Paket-Manager der Wahl

# Helm-Charts scannen
trivy config helm/ --severity HIGH,CRITICAL

# Dockerfiles + Repo-weite IaC
trivy config . --severity HIGH,CRITICAL
```

---

## Manuelle Schritte (einmalig einrichten)

- [x] Dependabot-Alerts aktiviert (`.github/dependabot.yml` — `npm` + `gomod` + `github-actions`)
- [x] GitHub Secret Scanning aktiviert
- [x] GitHub Push Protection aktiviert
- [x] GitHub Dependabot Security Updates aktiviert
- [x] CodeQL aktiviert
- [x] gosec + Semgrep + Trivy in `.github/workflows/security-scan.yml` (SARIF-Upload in GitHub Security-Tab)
- [x] Trivy-Image-Scan in `.github/workflows/docker-publish.yml` (Build-Block bei CRITICAL)

## Lizenz-Compliance

Die Anwendung ist proprietäre, kommerzielle Software. Lizenzhinweise:

- `LICENSE` im Repo-Root enthält die proprietäre Lizenz-Erklärung des Copyright-Inhabers.
- `THIRD_PARTY_NOTICES.md` listet alle direkten Go- und Node-Abhängigkeiten mit ihren OSS-Lizenzen, inklusive Quellenangaben.
- Geprüft wurde am 2026-05-14: keine GPL-, AGPL- oder SSPL-kontaminierten Abhängigkeiten in der Produktions-Lieferkette. Sharp/libvips (LGPL-3.0-or-later) ist als dynamisch geladenes Native-Addon des `sharp`-npm-Pakets enthalten; der LGPL-§6-Source-Offer ist in `THIRD_PARTY_NOTICES.md` dokumentiert.
- Bei neuen Abhängigkeiten (`go get` / `npm install`) muss die Lizenz vor dem Merge geprüft werden. Empfohlene Tools: `go-licenses report ./...` für Go, `license-checker --production --excludePrivatePackages` für npm.
- Pauschale Suppressions oder das Aufnehmen GPL-/AGPL-/SSPL-Pakete erfordern menschliche Freigabe.

## Dokumentierte False Positives

| Scanner | Regel | Datei | Begründung |
|---------|-------|-------|------------|
| CodeQL | `go/weak-sensitive-data-hashing` | `internal/http/apikey_middleware.go:80` | API-Keys sind hochentropische Zufalls-Token (`moak_` + 32 random chars). SHA-256 ist Standard für Token-Hashing (GitHub, Stripe). Bcrypt ist nur bei niedrig-entropischen Passwörtern relevant. Alert auf GitHub als False Positive dismissed. |
