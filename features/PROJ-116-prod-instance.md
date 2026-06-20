# PROJ-116: Prod-Instanz member-onboarding einrichten

## Status: In Progress
**Created:** 2026-06-20
**Last Updated:** 2026-06-20

## Ziel

Neben der bestehenden Test-Instanz eine dedizierte **Produktions-Instanz** von member-onboarding einrichten — für den Go-Live in der kostenlosen Phase (`globalLiveMode=false`). Die Test-Instanz bleibt unverändert (`-test`), es findet **keine Umbenennung** statt (Owner-Entscheidung 2026-06-20).

## Owner-Entscheidungen (2026-06-20)

- **Namespace:** `eegfaktura-member-onboarding` (ohne Suffix; Test behält `-test`).
- **Domain:** `member-onboarding.eegfaktura.at`.
- **Keycloak:** gleicher Client `eegfaktura-member-onboarding` (Realm `EEGFaktura`), nur Prod-Redirect-URIs ergänzen.
- **DB:** eigener Cluster-Postgres je Helm-Release → Prod bekommt eine eigene, leere DB (Chart deployt `postgres.yaml` pro Namespace). Nichts mit Test geteilt.
- **Kein Rename** der Test-Umgebung auf „pilot" (verworfen — Aufwand/Link-Risiko zu hoch für rein semantischen Gewinn).
- **Billing:** `globalLiveMode=false` während der kostenlosen Phase.

## Umfang

- **Kein Code / keine Migration / keine Chart-Änderung.** Der Helm-Chart ist bereits vollständig environment-parametrisiert (`namespace`, `ingress.host`, eigener Postgres, Keycloak/Core via `values-env.yaml`). Prod = zweite Helm-Release mit eigener `values-env.yaml` + `values-secret.yaml` (beide gitignored).
- **Mein Deliverable:** Einrichtungs-Runbook + Prod-`values-env`-Profil + Smoke-Test-Checkliste → `private/deploy/prod-instance-runbook.md` (visibility:private, weil Infra-Topologie).
- **Owner-Arbeit (kein Cluster-Apply durch Claude):** DNS, TLS, Keycloak-Redirect-URIs, frische Prod-Secrets, `helm install`.

## Offene Owner-Bestätigungen (im Runbook detailliert)

- Prod-`coreBaseUrl`: identisch zur Test-Instanz (echter Core) oder separater Prod-Core?
- Postal-Prod-Host/User/Absender (⚠ SMTP **Port 25**, 587 geblockt).
- Echte `customerOnboarding`-Owner-Stammdaten fürs AVV-PDF.

## Abgrenzung

- Free-Phase-Kommunikation (Banner „aktuell kostenlos", No-Charge-Gate, Trial-Entschärfung) ist **PROJ-115** — separat.
- FreeFinance-/Mollie-Live, `globalLiveMode`-Cutover: erst beim paid-Launch.

## Referenz

Runbook + vollständiges Prod-Values-Profil: `private/deploy/prod-instance-runbook.md`.
