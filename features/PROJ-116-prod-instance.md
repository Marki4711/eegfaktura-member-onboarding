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

## Zonen-Konzept (Owner-Klärung 2026-06-20)

| Zone | member-onboarding | Faktura-Core-Anbindung | DB | Status |
|---|---|---|---|---|
| **Test** | bestehende `member-onboarding-test` läuft als **Test-Zone weiter** | aktuell **Prod-Core** angebunden (akzeptiert); sobald eine **Faktura-Test-Zone** verfügbar ist, diese für die Core-Operationen anbinden | eigene Test-DB | aktiv |
| **Pilot** | eigene Pilot-Zone | (Prod-Core) | **gemeinsame DB mit Prod** (Pilot = echte Daten, frühe Stufe) | **geplant, jetzt NICHT umgesetzt** |
| **Prod** | `member-onboarding` (diese PROJ-116) | Prod-Core | eigene Prod-DB | in Einrichtung |

Merksätze:
- Eine **Pilot-Zone teilt die DB mit Prod** (Pilot ist echte Nutzung, nur eine frühere Rollout-Stufe) — deshalb ist die heutige `-test` (eigene DB, Prod-Core) eher eine **Test-Zone** als eine Pilot-Zone. Genau so wird sie weiterbetrieben.
- **TODO (Owner):** sobald eine Faktura-**Test-Zone** existiert, die Test-Zone-`coreBaseUrl` von Prod-Core auf den Faktura-Test-Core umstellen.
- Die Pilot-Zone ist **bewusst vertagt** — kein Bau in PROJ-116. PROJ-116 liefert nur die Prod-Zone.

## Umfang

- **Kein Code / keine Migration / keine Chart-Änderung.** Der Helm-Chart ist bereits vollständig environment-parametrisiert (`namespace`, `ingress.host`, eigener Postgres, Keycloak/Core via `values-env.yaml`). Prod = zweite Helm-Release mit eigener `values-env.yaml` + `values-secret.yaml` (beide gitignored).
- **Mein Deliverable:** Einrichtungs-Runbook + Prod-`values-env`-Profil + Smoke-Test-Checkliste → `private/deploy/prod-instance-runbook.md` (visibility:private, weil Infra-Topologie).
- **Owner-Arbeit (kein Cluster-Apply durch Claude):** Keycloak-Redirect-URIs, frische Prod-Secrets, `helm install`. **DNS + TLS sind durch das vorhandene Wildcard `*.eegfaktura.at` bereits erledigt** (Owner-Bestätigung 2026-06-20).

## Offene Owner-Bestätigungen (im Runbook detailliert)

- ~~Prod-`coreBaseUrl`~~ ✅ **geklärt 2026-06-20**: identisch zur Test-Instanz (nur ein echter Core).
- Postal-Prod-Host/User/Absender (⚠ SMTP **Port 25**, 587 geblockt). Falls identisch zu Test → 1:1 übernehmen.
- Echte `customerOnboarding`-Owner-Stammdaten fürs AVV-PDF.

## Abgrenzung

- Free-Phase-Kommunikation (Banner „aktuell kostenlos", No-Charge-Gate, Trial-Entschärfung) ist **PROJ-115** — separat.
- FreeFinance-/Mollie-Live, `globalLiveMode`-Cutover: erst beim paid-Launch.

## Referenz

Runbook + vollständiges Prod-Values-Profil: `private/deploy/prod-instance-runbook.md`.
