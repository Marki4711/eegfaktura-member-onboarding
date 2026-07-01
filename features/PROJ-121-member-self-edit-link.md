# PROJ-121 — Mitglieder-Bearbeitungslink (Self-Edit im needs_info-Flow)

**Status:** Deployed (v1.47.0-PROJ-121, sha-f905fa0)
**Owner-Anfrage:** 2026-07-01 (Tester/Nutzer: needs_info-Mail versprach einen
„ursprünglichen Antragslink", den es nicht gab)

## Deployment

- **Datum:** 2026-07-01
- **Tag:** `v1.47.0-PROJ-121`
- **Image:** `sha-f905fa0` (Backend + Frontend; values.yaml Auto-Bump c2975e9)
- **Migration:** keine (nur neue Query + Endpoint, kein Schema-Change) → kein
  Migrate-Job-Schema-Apply nötig, reiner Image-Wechsel
- **Security:** Review APPROVED (0 Critical/High)
- **Owner-Aktion:** `git pull` + `helm upgrade` je Zone (test + prod) mit
  `-f values-env.yaml -f values-secret.yaml`. Owner führt `helm upgrade` selbst
  aus (kein Cluster-Apply durch Claude).
- **Smoke:** Antrag auf „Info benötigt" setzen → Mail enthält
  „Antrag online bearbeiten"-Button; Link öffnet Formular vorbefüllt; Änderung +
  Absenden → `needs_info → submitted`; Link nach Genehmigung → „nicht mehr
  bearbeitbar".

## Problem

Die needs_info-Mail („Info anfordern") enthielt den Satz *„… kann [EEG] dir den
ursprünglichen Antragslink erneut zusenden."* — eine Funktion, die es **nicht
gab**: keine Admin-Aktion zum Senden eines Bearbeitungslinks, keine
Mitglieder-Bearbeitungsseite. Der öffentliche `/register/{rc}`-Link legt einen
**neuen** Antrag an (Duplikat), statt den bestehenden zu bearbeiten. Der Admin
konnte nur selbst über „Bearbeiten" korrigieren.

## Ziel

Das Mitglied bekommt in der needs_info-Mail einen **Bearbeitungslink** zu genau
seinem Antrag. Der Link öffnet das Registrierungsformular **vorbefüllt** mit den
bisherigen Angaben; nach dem Absenden geht der Antrag erneut in die Prüfung
(`needs_info → submitted`).

## User Stories

1. Als Mitglied möchte ich meinen Antrag nach einer Rückfrage selbst korrigieren,
   ohne alles neu einzugeben.
2. Als EEG-Betreiber möchte ich, dass die needs_info-Mail einen funktionierenden
   Bearbeitungslink enthält (kein leeres Versprechen).
3. Als Betreiber möchte ich, dass ein Bearbeitungslink nach Abschluss der Prüfung
   nicht mehr funktioniert (kein nachträgliches Ändern genehmigter Anträge).

## Akzeptanzkriterien

- **AC-1** Setzt der Admin einen Antrag auf `needs_info`, enthält die Mitglieder-
  Mail einen Button „Antrag online bearbeiten" mit einem Capability-Link
  (`/register/{rc}?edit={id}`). Ist `publicBaseURL` nicht konfiguriert, entfällt
  der Button (Fallback-Text). ✔
- **AC-2** Der Link öffnet das Formular mit allen bisherigen Angaben vorbefüllt
  (Mitgliedsdaten, Adresse, Bank, Zusatzangaben, Genossenschaftsanteile,
  Netzbetreiber-Vollmacht, Ansprechperson, Rechnungs-E-Mail, alle Zählpunkte
  inkl. PV/Batterie/Einspeiselimit). ✔
- **AC-3** Nach „Änderungen absenden" wird der Antrag aktualisiert (`PUT`) und
  wieder eingereicht (`needs_info → submitted`). Kein neuer Antrag, kein Duplikat. ✔
- **AC-4** Der Lade-Endpoint gibt einen Antrag **nur** in editierbarem Status
  (`draft`/`needs_info`) zurück; sonst generisches 404. Unbekannte/kaputte IDs
  ebenfalls 404/400 — die Capability-URL verrät nicht, ob eine ID existiert. ✔
- **AC-5** Der Lade-Endpoint liefert **keine** Admin-/Internal-Felder
  (adminNote, memberNumber, needsInfoReason, mandateReference, targetParticipantId,
  Import-Fehler, SEPA-IP, Status-Log). ✔
- **AC-6** Kein CAPTCHA im Edit-Modus (Capability = Antrags-UUID). ✔
- **AC-7** Der Edit-Round-Trip verliert keine Felder — insbesondere die
  Zusatzangaben (Beitrittsdatum, Personen, Wärmepumpe, E-Auto, Warmwasser) und
  die Genossenschaftsanteile (die im member-`UpdateTx` bislang gar nicht
  persistiert wurden — behoben). ✔

## Edge Cases

- **EC-1** RC-Mismatch: Gehört die geladene Antrags-ID nicht zur RC der Seite,
  wird der Edit-Modus verweigert (freundlicher Hinweis statt Leer-Formular). ✔
- **EC-2** Antrag inzwischen genehmigt/importiert/abgelehnt → 404 → Hinweis
  „Bearbeitungslink ungültig oder nicht mehr bearbeitbar". ✔
- **EC-3** `require_email_confirmation`: Re-Submit läuft über den bestehenden
  Submit-Pfad (unverändert). ✔

## Tech Design

**Design-Entscheidung (schlank, ohne Migration):** Capability = die **Antrags-
UUID** (schützt bereits `PUT`/`submit`). Kein separater Token → **keine Schema-
Migration**. Zugriff status-gegatet (`draft`/`needs_info`) → natürliche
Ablaufzeit. Token-Härtung (Ablauf/Widerruf) als spätere Option vermerkt.

- **Backend**
  - `GET /api/public/applications/{id}` (neu) → `PublicApplicationEditResponse`
    (form-shaped, nur eigene Daten, keine Admin-Felder). Status-Gate im Service
    (`GetApplicationForEdit`), Rate-Limit wie die Registration-Config.
  - `PUT`/`submit` existierten bereits für `draft`/`needs_info`.
    `UpdateApplicationRequest` + `UpdateApplication` um die Zusatzangaben-Felder
    erweitert (fehlten → stiller Datenverlust). `UpdateTx` um
    `cooperative_shares_count` ergänzt (Pre-Existing-Lücke: nur der Admin-Update
    persistierte es).
  - needs_info-Mail (`buildStatusChangeData` + Template) rendert den
    Bearbeitungs-Button; `admin_service` baut die URL aus `publicBaseURL`.
- **Frontend**
  - `getApplicationForEdit` + `updatePublicApplication` (public API-Client).
  - `mapApplicationToFormValues` (Inverse zu `buildCreatePayload`) +
    `buildUpdatePayload`. Round-trip-Test fixiert den Vertrag.
  - `RegistrationForm` mit `editApplication`-Prop: vorbefüllte defaultValues,
    Edit-Banner, `PUT`+`submit` statt `create`, kein Turnstile.
  - `/register/[rc_number]?edit=<id>` lädt den Antrag serverseitig, RC-Guard,
    Fehler-Alert.

## Verifikation

- go build/vet/test (voll) ✔, `npx tsc --noEmit` ✔, vitest (263, inkl. neuem
  Round-trip-Test) ✔, `npm run build` ✔.
- **E2E-Smoke** gegen lokalen Stack: GET 200 (needs_info) / 404 (approved,
  imported, unbekannt) / 400 (malformed); PII-Felder nicht in der Antwort;
  PUT persistiert alle Felder (inkl. membership_start_date + cooperative_shares);
  submit `needs_info → submitted`; GET danach 404 (re-locked). ✔

## Security Review

**Reviewer:** Security Engineer (AI) · **Datum:** 2026-07-01 · **Verdikt: APPROVED**

**Threat Model:** Neuer **unauthentifizierter** öffentlicher `GET
/api/public/applications/{id}` gibt die eigenen Antragsdaten eines Mitglieds
(inkl. IBAN) per **UUID-Capability** zurück. Worst Case bei Fehlern: PII-Leak
oder Lesen/Ändern von Anträgen außerhalb des Bearbeitungsfensters. Alle
Fokuspunkte am Code verifiziert.

### Verifiziert
- **PII-Exposure:** `PublicApplicationEditResponse` wird Feld-für-Feld gemappt
  (NICHT `shared.Application` verbatim). Enthält **keine** Admin-/Internal-Felder
  (adminNote, memberNumber, needsInfoReason, mandateReference,
  targetParticipantId, importError*, sepaMandateAcceptedIp, status_log) —
  per grep bestätigt. MeteringPoints = eigene ZP-Daten des Mitglieds. IBAN ist
  die **eigene** IBAN des Mitglieds (fürs Edit-Formular nötig) → akzeptiert.
- **Status-Gate/IDOR:** `GetApplicationForEdit` + `UpdateApplication` +
  `SubmitApplication` akzeptieren nur `draft`/`needs_info` → sonst
  `ErrNotFound`/`ErrConflict`. Re-Lock nach Submit E2E-bestätigt (GET→404).
- **Existence-Oracle:** non-editable UND unknown liefern beide generisches
  `{"code":"not_found","message":"Resource not found"}` (404). Kein Statuscode-
  Leak; Timing-Delta (PK-Hit vs. -Miss) vernachlässigbar bei 122-bit-UUID.
- **Injection:** `GetByID` / `GetByApplicationID` / `UpdateTx` ($43) alle
  parametrisiert. Kein neuer String-Concat-Query.
- **Rate-Limit:** GET unter `PublicGetRegistrationRateLimitMiddleware`,
  PUT/submit unter `PublicSubmitRateLimitMiddleware`. Mail hängt am
  Auth-geschützten Admin-needs_info, nicht am Public-GET.
- **Open-Redirect/SSRF:** `editURL` = `publicBaseURL` (Server-Config) + rcNumber
  (DB) + app.ID (UUID) — kein user-kontrollierter Anteil.
- **State-Corruption:** `UpdateApplication` ändert keinen Status; `UpdateTx` hat
  **genau einen** Aufrufer (member `UpdateApplication`, per grep bestätigt) →
  der `cooperative_shares_count`-Fix bricht keinen Admin-Pfad. Kein Schema-Change.

### Findings

| Severity | File | Area | Risk | Szenario | Fix | Confidence |
|----------|------|------|------|----------|-----|------------|
| Info | internal/http/application.go | GetApplicationForEdit | Capability-URL exponiert die **eigene** IBAN/PII des Mitglieds im GET | Wer den Link hat (Mitglied / Weiterleitung / Mail-Intercept), sieht die eigenen Antragsdaten | Akzeptiert: fürs Edit-Formular nötig; 122-bit-UUID; Link nur an Mitglieds-Mail; HTTPS; dieselbe UUID erlaubt bereits PUT (mächtiger). Optionale Härtung: Edit-Token mit Ablauf | High |
| Info | internal/lib (npm) | Dependencies | 2 High-CVEs im npm-Baum | Bestand — **kein** package.json-Change durch PROJ-121 | Separat via CI-Security-Scan (SARIF) triagen | High |

**Keine Critical/High-Findings durch PROJ-121 eingeführt.**

### Scans
- **govulncheck:** 0 callable (5 imported + 1 module, nicht aufgerufen).
- **gosec:** 64 Files, **0 Issues**.
- **npm audit:** 2 high / 0 critical — Bestand (kein package.json-Change), CI-SARIF.
- **Trivy IaC:** übersprungen — keine Helm-/Dockerfile-/CI-Änderung (begründet).

### Verdikt: **APPROVED**
Keine Critical/High durch das Feature. Info-Items dokumentiert + akzeptiert.
Deploy freigegeben (Owner macht `helm upgrade`; **keine Migration** → kein
Schema-Checkpoint).

## Offen

- Optionale Härtung: dedizierter Edit-Token mit Ablauf/Widerruf (Migration),
  falls die reine UUID-Capability später nicht ausreicht.
- 2 High-npm-CVEs (Bestand) separat über den CI-Security-Scan behandeln.
