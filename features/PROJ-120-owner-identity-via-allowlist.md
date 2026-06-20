# PROJ-120: Owner-Identität via Allowlist — Superuser auf Datensicht reduzieren

## Status: Planned
**Created:** 2026-06-20
**Last Updated:** 2026-06-20

## Kontext & Motivation

Heute gewährt die **`superuser`-Rolle automatisch** Zugang zu allen
Owner-/Betreiber-Funktionen:

| Fläche | Frontend-Gate | Backend-Gate |
|---|---|---|
| **Cockpit** (`/admin/cockpit`) | `getCockpitMe` (Superuser **ODER** Allowlist) | `IsCockpitAllowed` — *Superuser ist immer erlaubt* + Allowlist |
| **Abrechnung** (`/admin/billing`) | `roles.includes("superuser")` | `requireSuperuser` (17 Endpoints) |
| **Plattform-Buchungen** (`/admin/customer-onboarding`) | `roles.includes("superuser")` | `requireSuperuser` (List/Detail/Approve/Reject) |

**Owner-Entscheidung 2026-06-20:** Owner-Identität und Superuser-Rolle entkoppeln.
- **Owner-Funktionen** (Cockpit, Abrechnung, Plattform-Buchungen) sind **nur** für
  die in der **Owner-Allowlist** konfigurierten Identitäten zugänglich.
- Die **`superuser`-Rolle** gewährt **nur** noch Daten-Sicht („sieht alle
  Anträge/EEGs", tenant-übergreifend) — **keinen** Zugang zu Owner-Funktionen.

Begründung: Ein Helfer mit Superuser-Rolle (für tenant-übergreifende Antrags-Sicht)
soll **nicht** automatisch Betreiber-Steuerung (Abrechnung, Buchungs-Freigabe,
Cockpit) erreichen. Owner = wer konfiguriert ist, nicht wer die Rolle hat.

## Dependencies
- Requires: PROJ-72 (Cockpit, `IsCockpitAllowed`, Allowlist `COCKPIT_ALLOWED_EMAILS`,
  Eligibility-Probe `/api/admin/owner-cockpit/me`).
- Betrifft: PROJ-104/PROJ-109 (Billing-Endpoints), PROJ-119 (Plattform-Buchungen-
  BackOffice + Nav-Link — dessen finales Gate gehört zu dieser Spec).

## User Stories
- Als **Plattform-Betreiber** will ich, dass Owner-Funktionen (Cockpit, Abrechnung,
  Plattform-Buchungen) nur der konfigurierten Owner-Allowlist offenstehen, damit ein
  Helfer mit Superuser-Rolle (nur für Daten-Sicht) keine Betreiber-Steuerung erreicht.
- Als **Betreiber** will ich, dass die Superuser-Rolle ausschließlich „alle Anträge
  EEG-übergreifend sehen" bedeutet, damit Rolle und Owner-Identität sauber getrennt sind.

## Acceptance Criteria
- [ ] **AC-1 (Entkopplung):** Cockpit, Abrechnung und Plattform-Buchungen
  (List/Detail/Approve/Reject) sind **nur** für Allowlist-Identitäten zugänglich.
  Die bloße `superuser`-Rolle **ohne** Allowlist-Eintrag gewährt **keinen** Zugang
  (Frontend-Nav ausgeblendet **und** Backend 403).
- [ ] **AC-2 (Superuser-Datensicht bleibt):** Die `superuser`-Rolle behält ihre
  Daten-Scope-Rechte unverändert: tenant-übergreifende Sicht aller Anträge
  (`/admin/applications` Liste/Detail) und jede bestehende Superuser-Daten-Scoping-
  Logik in Services — **nichts davon wird angefasst**.
- [ ] **AC-3 (neuer Owner-Check):** Ein neuer Check (z.B. `IsOwner(allowlist)` =
  Email-in-Allowlist, **ohne** Superuser-Auto-Grant) ersetzt `requireSuperuser` auf
  den Billing- + Customer-Onboarding-BackOffice-Endpoints und den Superuser-Zweig im
  Cockpit-Gate.
- [ ] **AC-4 (Eligibility-Probe):** Die Probe `/api/admin/owner-cockpit/me` liefert
  `eligible` = Email-in-Allowlist (kein Superuser-Auto-Grant). Alle drei
  Owner-Nav-Links (Cockpit, Abrechnung, Plattform-Buchungen) gaten auf diese Probe.
- [ ] **AC-5 (Billing-Nav):** `billing-nav-link` wechselt von
  `roles.includes("superuser")` auf die Owner-Eligibility-Probe.
- [ ] **AC-6 (kein Lockout — VERIFIKATION VOR DEPLOY, BLOCKING):** Die tatsächliche
  Login-Email des Betreibers muss in `OWNER_ALLOWED_EMAILS` (bzw. via Fallback in
  `cockpitAllowedEmails`) in **beiden** Zonen stehen.
  ⚠️ **Befund Backend-Phase 2026-06-20:** `values-env-prod.yaml` enthält
  `cockpitAllowedEmails: "office@gemeinstrom.at,eegfaktura@vfeeg.org"` — **NICHT**
  `matthiasm@vfeeg.at`. Wenn der Betreiber als `matthiasm@vfeeg.at` einloggt, sperrt
  ihn PROJ-120 aus allen Owner-Funktionen aus. Vor Deploy zwingend klären: entweder
  Login-Email in die Allowlist aufnehmen ODER mit einer bereits gelisteten Email
  einloggen. Der Betreiber bleibt Superuser (für `hasAdminAccess` + alle Anträge);
  die Allowlist-Mitgliedschaft ist die separate Owner-Bedingung.
- [ ] **AC-7 (Config):** `COCKPIT_ALLOWED_EMAILS` wird als „Owner-Allowlist"
  umgedeutet (optional Rename auf `OWNER_ALLOWED_EMAILS` — Entscheidung in
  /architecture; bei Rename Rückwärtskompatibilität ODER alle Zonen-values-env
  gleichzeitig anpassen, [[feedback_helm_values_split]]).
- [ ] **AC-8 (security, server-seitig):** Erzwingung auf **jedem** Owner-Endpoint
  server-seitig; Frontend-Ausblenden ist kosmetisch. Tenant-Admins bleiben
  vollständig von Owner-Funktionen ausgeschlossen (unverändert).
- [ ] **AC-9 (Config-Fallback):** Backend liest die Owner-Allowlist als
  `firstNonEmpty(OWNER_ALLOWED_EMAILS, COCKPIT_ALLOWED_EMAILS)`; Helm reicht in der
  Übergangsphase beide Env-Vars durch. Bestehende `values-env` laufen ohne Änderung
  weiter (kein Flag-Day, kein Lockout).
- [ ] **AC-10 (E2E-Test-Header):** `TestHeaderAuthMiddleware` + `adminAuthHeaders`
  unterstützen einen Email-Claim (`X-Test-Email`); die CI-Owner-Allowlist enthält die
  Test-Email, sodass Owner-Endpoint-E2E autorisierbar ist. Bestehende E2E-Specs (die
  keine Owner-Endpoints treffen) bleiben grün.
- [ ] **AC-11 (Test-Fixtures angepasst):** `admin_cockpit_test.go` spiegelt die neue
  Semantik (Superuser ohne Allowlist → NICHT berechtigt); Billing-/BackOffice-Handler-
  Tests autorisieren über einen allowlisted Email-Claim statt der bloßen Superuser-Rolle.

## Edge Cases
- **Reiner Superuser, nicht in Allowlist:** sieht alle Anträge; Cockpit/Abrechnung/
  Plattform-Buchungen ausgeblendet + 403. (gewünscht)
- **Allowlist-Email ohne Superuser-Rolle und ohne Tenant:** `hasAdminAccess` verlangt
  heute eine Rolle ODER einen Tenant → solch ein Nur-Owner würde am Admin-Layout zu
  `/unauthorized` umgeleitet. Da der Betreiber Superuser **bleibt**, tritt der Fall
  aktuell nicht ein. /architecture entscheidet, ob „Nur-Owner-via-Allowlist" künftig
  `hasAdminAccess` erfüllen soll (eigener Zweig) — vorerst Non-Goal.
- **Superuser-Referenzen, die Daten-Scope sind (NICHT Owner-Funktion):** z.B. der
  Customer-Onboarding-`Submit` (Superuser ODER Tenant — tenant-skopierte Buchung),
  Reconciliation, Antrags-Listen-Scoping. Diese bleiben **unverändert**. /architecture
  liefert die vollständige Klassifikation jeder `IsSuperuser()`/`requireSuperuser`/
  `roles.includes("superuser")`-Fundstelle als Daten-Scope (behalten) vs.
  Owner-Funktion (auf Owner-Check umstellen).
- **Cockpit-Tests (PROJ-72):** `TestIsCockpitAllowed_SuperuserAlwaysAllowed` u.ä.
  müssen an die neue Semantik (kein Auto-Grant) angepasst werden.

## Non-Goals
- Entfernen der Superuser-Rolle oder ihrer Daten-Scope-Rechte.
- Per-EEG-Owner-Rollen.
- Änderung des Tenant-Admin-Zugangs.
- „Nur-Owner-via-Allowlist erfüllt `hasAdminAccess`" (vorerst; Betreiber bleibt Superuser).

## Technical Requirements / Notes
- **Auth-Änderung** über 3 Backends + 3 Nav-Links + Probe → `/security-review`
  Pflicht; Human-Approval-Checkpoint (CLAUDE.md: Tenant-/Auth-Logik).
- **Audit-Pflicht:** jede `IsSuperuser()` / `requireSuperuser` /
  `roles.includes("superuser")`-Fundstelle klassifizieren (Daten-Scope vs.
  Owner-Funktion) — Ergebnis-Tabelle in /architecture, damit keine Daten-Scope-
  Stelle versehentlich auf den Owner-Check umgestellt wird (und umgekehrt).
- **Deploy-Voraussetzung:** Betreiber-Login-Email in der Allowlist je Zone — erfüllt
  (AC-6), vor Deploy gegenprüfen.

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)

### Kernidee
Heute ist „Superuser" automatisch „Owner" — die Rolle schaltet jede Betreiber-Fläche
frei. Wir trennen die beiden Begriffe sauber:
- **Owner-Identität** = Email in der **Owner-Allowlist** (heute `COCKPIT_ALLOWED_EMAILS`).
  Nur sie öffnet Owner-Funktionen.
- **Superuser-Rolle** = nur **Daten-Sicht** (alle Anträge/EEGs tenant-übergreifend
  sehen + verwalten). Gibt **keinen** Zugang mehr zu Owner-Funktionen.

Keine DB-Änderung. Reine Berechtigungs-Umverdrahtung + ein neuer „Owner-Check".

### A) Touch-Point-Baum (Full-Chain)
```
Owner-Allowlist (Helm-Config: OWNER_ALLOWED_EMAILS)
└── Backend: neuer "Owner-Check" (NUR Allowlist, kein Superuser-Auto-Grant)
    ├── Cockpit            (Liste + Eligibility-Probe)         → Owner-Check
    ├── Abrechnung         (17 Owner-Aktionen)                 → Owner-Check
    └── Plattform-Buchungen (Liste / Detail / Freigeben / Ablehnen) → Owner-Check
        │
        └── Eligibility-Probe /owner-cockpit/me  → "Owner ja/nein" (Allowlist-only)
            └── Frontend: 3 Nav-Links (Cockpit, Abrechnung, Plattform-Buchungen)
                rendern nur, wenn die Probe "ja" sagt
```

### B) Das Audit — jede Superuser-Stelle klassifiziert
**Kern-Deliverable.** Alle Fundstellen von `IsSuperuser()` / `requireSuperuser` /
`roles.includes("superuser")` wurden gesichtet und in zwei Klassen eingeteilt:

**→ Owner-Funktion (auf Owner-Check umstellen):**

| Bereich | Heutiges Gate | Stellen |
|---|---|---|
| **Cockpit** | `IsCockpitAllowed` (Superuser-ODER-Allowlist) | `admin_cockpit.go` GetMe-Probe + ListEEGs |
| **Abrechnung** | `requireSuperuser` | `admin_billing.go` — alle 17 Endpoints + der `requireSuperuser`-Helper |
| **Plattform-Buchungen** | `requireSuperuser` | `admin_customer_onboarding.go` — List / Detail / Approve / Reject + der `requireSuperuser`-Helper |
| **Nav-Links (Frontend)** | superuser-Rolle bzw. `getCockpitMe` | `cockpit-nav-link`, `billing-nav-link`, `customer-onboarding-nav-link` → alle auf die Owner-Probe |

**→ Daten-Sicht / Tenant-Scope (BLEIBT `IsSuperuser()` — NICHT anfassen):**

| Bereich | Warum bleibt |
|---|---|
| **Admin-Bereichs-Zugang** (`auth_middleware`: Superuser ODER Tenant) | Zugangstor, kein Owner-Feature |
| **Anträge** — Liste, Detail, Bulk-Delete, Reassign, Import (`admin_applications*`) | Genau das, was Superuser dürfen soll: alle Anträge EEG-übergreifend sehen/verwalten |
| **Tenant-Bypass-Lesepfade** — Legal-Docs, External-Keys, Daten-Export, Config-Export, EEG-eigene-Rechnungen (`admin_eeg_invoices`), Entrypoint-Sync-Skip | Superuser-Bypass des Tenant-Filters = Daten-Sicht, nicht Betreiber-Steuerung |
| **Customer-Onboarding** — Submit, AVV-PDF-Download, Tenant-Status | Tenant-skopiert (der EEG-Admin bucht/lädt seine eigene); Superuser nur als Tenant-Bypass |

**Wichtig:** `admin_customer_onboarding.go` enthält BEIDE Klassen — die 4 BackOffice-
Endpoints (Owner) wandern auf den Owner-Check, Submit/AVV-Download/Tenant-Status
(Tenant-Scope) bleiben unverändert. Saubere Trennung, da die BackOffice-Endpoints
den eigenen `requireSuperuser`-Helper nutzen.

### C) Daten- / Config-Modell
- **Keine** DB-Änderung, keine Migration.
- Einzige „Datenquelle" ist die **Owner-Allowlist** (kommaseparierte Email-Liste,
  Helm-konfiguriert, case-insensitiv, leer = niemand außer Dev-Modus).
- **Config-Rename-Entscheidung (bestätigt Owner 2026-06-20):** `COCKPIT_ALLOWED_EMAILS` →
  `OWNER_ALLOWED_EMAILS`. WHY: Der Wert steuert künftig **alle** Owner-Funktionen,
  nicht nur das Cockpit — der alte Name wäre irreführend. Migration ohne Flag-Day:
  das Backend liest zuerst `OWNER_ALLOWED_EMAILS`, fällt auf `COCKPIT_ALLOWED_EMAILS`
  zurück, falls nur der alte gesetzt ist; beide Zonen-`values-env` werden im selben
  Deploy umgestellt ([[feedback_helm_values_split]]). Alternative (geringere Churn):
  alten Namen behalten — dann aber Doku-Klarstellung, dass er Owner-weit gilt.

### D) Tech-Entscheidungen (WHY)
1. **Owner-Check = nur Allowlist** (kein Superuser-Zweig). WHY: erfüllt die Owner-
   Direktive „Rolle ≠ Owner". Ein Helfer mit Superuser-Rolle (für Antrags-Sicht)
   erreicht keine Betreiber-Steuerung.
2. **Probe-Semantik wird Allowlist-only.** Da alle 3 Nav-Links über dieselbe Probe
   gaten, werden sie automatisch konsistent. Der bisherige `authPath="superuser_role"`
   entfällt; übrig bleibt `owner_allowlist` / `none` (+ `dev_mode`).
3. **Bewusst NICHT umgestellt:** die ganze Daten-Sicht-Spalte oben. WHY: Superuser
   muss weiterhin alle Anträge sehen/verwalten und Tenant-Filter überbrücken — das
   ist der definierte Rest-Zweck der Rolle.
4. **Kein Lockout.** Der Betreiber bleibt Superuser (für Admin-Zugang + alle Anträge)
   UND ist in der Allowlist (`matthiasm@vfeeg.at`, test+prod, bestätigt) — er behält
   damit beide Welten.

### E) hasAdminAccess-Edge-Case (Non-Goal)
Eine reine Owner-Identität (nur in der Allowlist, ohne Superuser-Rolle und ohne
Tenant) würde am Admin-Layout-Tor (`Superuser ODER Tenant`) abgewiesen. Da der
Betreiber Superuser **bleibt**, tritt das nicht ein. „Nur-Owner-via-Allowlist ohne
Rolle" ist **Non-Goal** dieser Spec.

### F) Tests & Dev/Screenshot (für /grill-me + /backend)
- **PROJ-72-Cockpit-Tests** (`TestIsCockpitAllowed_SuperuserAlwaysAllowed` u.ä.)
  kehren ihre Erwartung um: Superuser **ohne** Allowlist ist künftig NICHT mehr
  berechtigt. Neue Tests fixieren „Allowlist ja / sonst nein".
- **Billing- + BackOffice-Handler-Tests**, die heute per `Roles=["superuser"]`
  autorisieren, müssen auf ein Allowlist-Setup (Email + Allowlist) umgestellt werden.
- **Dev-/Screenshot-Modus:** sicherstellen, dass die Owner-Flächen im lokalen
  Screenshot-Lauf renderbar bleiben (Test-Header/Dev setzt eine allowlisted Email
  ODER expliziter Dev-Bypass) — Detail für /backend.

### G) Entscheidungen + offene Punkte
**Bestätigt (Owner 2026-06-20):**
- Config-Rename `OWNER_ALLOWED_EMAILS` mit Rückwärts-Fallback auf
  `COCKPIT_ALLOWED_EMAILS`.
- Anträge-Schreibaktionen (Bulk-Delete, Reassign, Import) bleiben **Daten-Sicht**
  (Superuser) — Antrags-Verwaltung ist kein Betreiber-Feature; auch Tenant-Admins
  nutzen sie für den eigenen Tenant.

**Noch offen (für /grill-me + /backend):**
- Dev-/Screenshot-Zugang zu Owner-Flächen ohne echte Allowlist (Test-Header-Email
  vs. Dev-Bypass).

### H) Grilling-Findings (2026-06-20, codebasiert verifiziert)
1. **Audit exhaustiv bestätigt:** 84 `superuser`-Fundstellen über 19 Dateien gesichtet.
   Owner-Funktion **ausschließlich** in `admin_billing.go` (17), `admin_cockpit.go` (4)
   und den 4 BackOffice-Endpoints in `admin_customer_onboarding.go`. Alles andere ist
   Daten-Sicht / Tenant-Bypass / Infrastruktur. **Kein Owner-Endpoint übersehen, keine
   Daten-Sicht-Stelle fälschlich als Owner.**
2. **E2E/Test-Header (Lücke + Fix):** `TestHeaderAuthMiddleware` setzt Subject, aber
   **keine Email** → unter `IsOwner` würde ein `X-Test-Superuser`-Request Owner-
   Endpoints NICHT mehr autorisieren. ABER: **keine Bestand-E2E-Spec trifft Owner-
   Endpoints** (nur `tests/helpers/auth.ts` referenziert sie) → heute kein Bruch.
   Fix: `X-Test-Email`-Header + `email`-Option in `adminAuthHeaders` ergänzen; CI-
   Owner-Allowlist um diese Test-Email erweitern → künftige Owner-Endpoint-E2E möglich.
3. **Screenshots (kein Problem):** Der Generator besucht **keine** Owner-Seiten
   (Cockpit/Abrechnung/Plattform-Buchungen kommen in `generate-screenshots.ts` nicht
   vor). Einziger Effekt: Owner-Nav-Links verschwinden aus allgemeinen Admin-
   Screenshots (Bot `screenshot-bot@example.local` nicht allowlisted). **Empfehlung:
   so lassen** — Owner-Flächen sind betreiber-intern, gehören nicht in die EEG-Admin-
   User-Doku; aus Tenant-Admin-Sicht erscheinen die Owner-Links ohnehin korrekt nicht.
   (Falls Owner-Doku gewünscht: Bot-Email ins Screenshot-Backend-Allowlist.)
4. **Config-Rename-Rollout (zero-downtime):** Backend liest
   `firstNonEmpty(OWNER_ALLOWED_EMAILS, COCKPIT_ALLOWED_EMAILS)`. Helm reicht in der
   Übergangsphase BEIDE Env-Vars durch (OWNER aus neuem Value Default "", COCKPIT aus
   Bestand-Value). Bestehende `values-env` (`cockpitAllowedEmails=matthiasm@vfeeg.at`)
   laufen via Fallback weiter → **kein Lockout, kein Flag-Day**; Umstellung auf den
   neuen Namen später.
5. **Dev-Mode-Asymmetrie (Bestand, kein Regress):** Cockpit `ListEEGs` ist im No-Auth-
   Dev offen (`claims != nil &&`-Muster), Billing/BackOffice `requireSuperuser` →
   `claims==nil` → 403. Der neue `requireOwner` spiegelt `requireSuperuser`
   (`claims==nil`→403); Cockpit behält sein Muster; `GetMe` bleibt `dev_mode`→
   `eligible=true`. Diese Bestand-Asymmetrie wird NICHT angefasst (Scope-Disziplin).
6. **authPath/Audit-Log:** `superuser_role`-Pfad entfällt → `owner_allowlist` | `none`
   | `dev_mode`. Audit-Log behält `email_domain` + `authPath` (kein PII).
7. **Test-Fixtures (deterministisch, risikofrei):** `admin_cockpit_test.go`
   (`SuperuserAlwaysAllowed` etc.) Erwartung umkehren; Billing-/BackOffice-Handler-
   Tests von `Roles=[superuser]` auf allowlisted-Email-Claim umstellen.
8. **Eligibility-Probe bleibt frei abfragbar** (kein 403 — eigene Eligibility, kein
   Leak). 3 Nav-Links → 3× dieselbe Probe pro Render: minimal; optionale Dedupe via
   Shared-Hook/Context, **nicht blockierend**.

### Dependencies
Keine neuen Pakete. Wiederverwendung der PROJ-72-Allowlist-Infrastruktur (Email-Claim,
Allowlist-Parser, `/owner-cockpit/me`-Probe, Helm-Config-Muster).

### Handoff / Build-Reihenfolge
Empfehlung: erst `/grill-me` (Auth/Lockout/Klassifikation heikel), dann `/backend`
(Owner-Check + Gate-Umstellung + Config + Tests), dann `/frontend` (3 Nav-Links auf
die Owner-Probe), dann `/security-review` (Pflicht), dann `/deploy`.

## Implementation Notes (Backend, 2026-06-20)

**Umgesetzt (go build/vet/test ./... grün, helm lint grün, helm template verifiziert):**
- `auth_middleware.go`: `IsCockpitAllowed`→`IsOwner` (Allowlist-only, **kein** Superuser-
  Zweig), `CockpitAuthPath`→`OwnerAuthPath` (`owner_allowlist`|`none`). `IsSuperuser`
  unverändert. `TestHeaderAuthMiddleware` + `X-Test-Email`→`claims.Email`. `cors.go`:
  `X-Test-Email` in Allow-Headers.
- `config.go`: `CockpitConfig` entfernt → `Config.OwnerAllowedEmails []string` =
  `parseEmailAllowlist(getEnv("OWNER_ALLOWED_EMAILS", getEnv("COCKPIT_ALLOWED_EMAILS","")))`
  (Fallback, AC-9).
- Gates umgestellt (genau 3 Dateien): `admin_billing.go` (17× `requireSuperuser`→
  `requireOwner` + Feld `ownerAllowedEmails` + Ctor-Param), `admin_customer_onboarding.go`
  (4 BackOffice-Endpoints `requireOwner`; Submit/AVV/TenantStatus unverändert),
  `admin_cockpit.go` (`IsOwner`/`OwnerAuthPath`, Forbidden-Message, Header-Kommentar).
- `cmd/server/main.go`: `cfg.OwnerAllowedEmails` in alle 3 Konstruktoren; Route-Kommentar.
- Helm: `backend.yaml` reicht `OWNER_ALLOWED_EMAILS` (neu) + `COCKPIT_ALLOWED_EMAILS`
  (Fallback) durch; `values.yaml` + `values-env.yaml.example`: `ownerAllowedEmails` +
  Lockout-Warnung, `cockpitAllowedEmails` als Legacy-Fallback.
- Tests: `admin_cockpit_test.go` auf IsOwner/OwnerAuthPath-Semantik umgeschrieben
  (Superuser-ohne-Allowlist = NICHT Owner); `admin_billing_free_phase_guard_test.go`
  auf `ownerCtx`+Allowlist; `test_header_auth_test.go` X-Test-Email-Test; `tests/helpers/auth.ts`
  `email`-Option (AC-10/AC-11).
- **ALLE anderen `IsSuperuser`-Stellen unangetastet** (Daten-Sicht/Tenant-Bypass).

**⚠️ BLOCKING vor Deploy:** siehe AC-6 — Prod-Allowlist enthält die Betreiber-Login-
Email evtl. nicht. Owner-Verifikation nötig.

**Offen:** /frontend (3 Nav-Links auf Owner-Probe), dann /security-review (Auth-Change).

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
