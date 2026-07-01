# PROJ-121 — Mitglieder-Bearbeitungslink (Self-Edit im needs_info-Flow)

**Status:** In Review
**Owner-Anfrage:** 2026-07-01 (Tester/Nutzer: needs_info-Mail versprach einen
„ursprünglichen Antragslink", den es nicht gab)

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

## Security

Berührt **öffentlichen Endpoint** (PII per Capability) + **Status-Transition**
→ `/security-review` **Pflicht** vor Deploy. Kein Schema-Change (keine Migration).

## Offen

- `/security-review` (Public-Endpoint + Status-Transition).
- Optionale Härtung: dedizierter Edit-Token mit Ablauf/Widerruf (Migration),
  falls die reine UUID-Capability später nicht ausreicht.
