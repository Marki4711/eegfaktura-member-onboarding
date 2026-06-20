# PROJ-119: Auto-Akzept der Plattform-Buchung

## Status: Planned
**Created:** 2026-06-20
**Last Updated:** 2026-06-20

## Kontext & Motivation

Heute ist die Plattform-Buchung (PROJ-71) zweistufig: Der EEG-Admin klickt
„Plattform buchen" → bestätigt AGB+AVV → Status `submitted` → **der Owner muss
manuell im BackOffice `/admin/customer-onboarding/{id}` auf „Approve" klicken** →
erst dann `approved`, Event `activated`, `is_active=true`, Welcome-Mail.

**Probleme (Owner 2026-06-20):**
1. Der Approve-Schritt erzeugt manuellen Aufwand für einen Solo-Betreiber.
2. Das Freigabe-BackOffice hat **keinen Nav-Link** — es ist nur über die direkte
   URL erreichbar (Nav zeigt nur Anträge/Einstellungen/Cockpit/Abrechnung). Das
   Cockpit (PROJ-72) ist read-only und hat bewusst keine Approve-Action. Dadurch
   wirkt es, als gäbe es die Freigabe-Funktion gar nicht.

**Owner-Entscheidung 2026-06-20:** Buchungen werden **immer automatisch akzeptiert**
(kein Toggle). Begründung: Der AVV wird bereits beim Submit erfasst (AGB+AVV-Häkchen
`eq=true` + AVV-Akzept-PDF als Beleg); der Owner-Approve war nur ein Geschäfts-Review,
auf das der Owner verzichten will. Falsch-/Junk-Buchungen werden nachgelagert per
Suspend behandelt (BackOffice bleibt dafür erhalten).

## Dependencies
- Requires: PROJ-71 (EEG-Customer-Onboarding) — der Buchungs-/Approve-Flow, der
  geändert wird.
- Koordiniert mit: PROJ-118 (AVV-Gate für Registrierungs-Aktivierung) — PROJ-118
  liest den `approved`-Zustand der Buchung; Auto-Akzept erzeugt diesen Zustand
  bereits beim Submit. Empfehlung: gemeinsam bauen/deployen.

## User Stories
- Als **EEG-Admin** will ich, dass meine Plattform-Buchung sofort nach Bestätigung
  von AGB+AVV wirksam wird, damit ich ohne Wartezeit auf eine manuelle Freigabe mit
  dem Mitglieder-Onboarding starten kann.
- Als **Plattform-Betreiber** will ich, dass Buchungen automatisch aktiviert werden,
  damit ich nicht jede einzelne manuell freigeben muss.
- Als **Plattform-Betreiber** will ich das Buchungs-BackOffice über die Navigation
  erreichen, damit ich Buchungen einsehen und bei Bedarf suspendieren kann.

## Acceptance Criteria
- [ ] **AC-1 (Auto-Akzept):** Ein erfolgreicher Buchungs-Submit (AGB+AVV akzeptiert)
  führt die Aktivierung **atomar** als Teil des Submits aus: Status `approved`,
  Event `activated`, `is_active=true` — ohne separaten Owner-Approve-Schritt.
- [ ] **AC-2 (Welcome-Mail):** Die Welcome-Mail mit AVV-PDF wird wie bisher beim
  Aktivieren versendet (jetzt direkt im Auto-Akzept-Pfad).
- [ ] **AC-3 (Owner-Info):** Die bestehende Owner-Benachrichtigung beim Submit wird
  zu einer reinen FYI umformuliert („EEG X hat gebucht und wurde automatisch
  aktiviert"), statt eine Handlungsaufforderung zur Freigabe zu sein.
- [ ] **AC-4 (kein hängender Zwischenstatus):** Es bleibt keine `submitted`-Buchung
  in Warteposition zurück — der Pfad geht direkt auf `approved` (bzw. durchläuft
  `submitted` nur transient innerhalb derselben Transaktion).
- [ ] **AC-5 (Suspend bleibt):** Das BackOffice behält die Reject/Suspend-Funktion
  (Post-Approve-Suspend → `is_active=false`), damit der Owner eine fälschlich
  aktivierte EEG nachträglich sperren kann.
- [ ] **AC-6 (Nav-Link):** Das Buchungs-BackOffice (`/admin/customer-onboarding`)
  ist über die Admin-Navigation erreichbar (nicht nur per direkter URL).
- [ ] **AC-7 (Doppel-Buchungs-Schutz):** Der bestehende Schutz gegen
  Doppel-Einreichung (`HasActiveSubmissionFor`) bleibt wirksam.
- [ ] **AC-8 (AVV nicht umgangen, security):** Auto-Akzept umgeht die AVV-Erfassung
  nicht — AGB+AVV-`eq=true`-Validierung und PDF-Erzeugung bleiben Pflicht; schlägt
  die PDF-Erzeugung fehl, scheitert der gesamte Submit (und damit die Aktivierung)
  — keine Aktivierung ohne AVV-Beleg.

## Edge Cases
- **Submit scheitert mittendrin (PDF-Fehler):** keine Aktivierung, kein
  Teil-Zustand (atomar) — wie heute.
- **Zuvor suspendierte/terminierte EEG bucht erneut:** Auto-Akzept darf eine
  **bestehende Suspendierung nicht still aufheben**. Empfehlung: bei aktiver
  Suspendierung/Terminierung KEIN Auto-Reaktivieren — manuelle Owner-Reaktivierung
  nötig. Offen für /architecture-Bestätigung.
- **Owner will eine bestimmte EEG vorab prüfen:** Mit Always-On-Auto-Akzept gibt es
  kein Vorab-Gate; der Owner kann nur nachgelagert suspendieren. Per Owner-Entscheidung
  akzeptiert.
- **Cockpit-Nutzer ohne Superuser (Email-Allowlist, PROJ-72):** Der neue Nav-Link
  zum BackOffice respektiert dieselbe Owner-/Sichtbarkeitslogik wie das BackOffice
  selbst (nicht jedem Tenant-Admin zeigen).

## Non-Goals
- Konfigurierbarer Auto-Akzept-Toggle (per EEG/global) — Owner wählte Always-On.
- Entfernen der BackOffice-Liste (bleibt für Suspend/Übersicht).
- Änderung des AVV-/AGB-Inhalts oder der Akzept-Erfassung.

## Technical Requirements / Notes
- **Sicherheit & Approval:** Berührt Status-Transitions (Customer-Onboarding) +
  Mail-Timing → `/security-review` Pflicht (Status-Transition-Trigger laut CLAUDE.md).
- Voraussichtlich **keine DB-Schema-Änderung** — der bestehende Approve-Pfad
  (`ApproveTx`) wird im Submit-Handler inline aufgerufen statt durch einen separaten
  Owner-Klick. Mit /architecture verifizieren.
- Build-/Deploy-Reihenfolge mit PROJ-118 koordinieren (gemeinsamer Go-Live sinnvoll).

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)

### Kernidee
Der zweistufige Buchungs-Ablauf wird einstufig: Die Bestätigung von AGB+AVV durch
den EEG-Admin aktiviert die EEG sofort — der separate Owner-Freigabe-Klick entfällt.

### A) Ablauf vorher → nachher
```
VORHER:  EEG-Admin: „Plattform buchen" (AGB+AVV)
         → Status „submitted"  → wartet auf Owner
         → Owner klickt „Approve" im BackOffice
         → Status „aktiviert", Registrierung scharf, Welcome-Mail

NACHHER: EEG-Admin: „Plattform buchen" (AGB+AVV)
         → in EINEM Schritt: aktiviert, Registrierung scharf, Welcome-Mail
         → Owner bekommt nur noch eine FYI-Mail („gebucht + auto-aktiviert")
```

### B) Daten-Modell (Klartext)
- **Keine DB-Migration.** Es ändert sich nur die *Reihenfolge* im Buchungs-Ablauf,
  nicht das Datenmodell. Der bestehende Status `submitted` wird nur noch transient
  innerhalb desselben Vorgangs durchlaufen (oder übersprungen) und bleibt nie als
  „Warteposten" liegen.
- Der bestehende **Doppel-Buchungs-Schutz** (eine EEG kann nicht zweimal buchen,
  solange eine eingereichte/aktive Buchung existiert) bleibt unverändert.

### C) Tech-Entscheidungen (WHY)
1. **Aktivierung über die bestehende Freigabe-Logik wiederverwenden** (kein
   Duplikat): Derselbe atomare Schritt, der heute beim Owner-Approve läuft
   (Status → aktiviert, `is_active=true`, Welcome-Mail mit AVV-PDF), wird direkt im
   Buchungs-Vorgang ausgelöst. WHY: ein einziger Code-Pfad für „aktivieren" — keine
   zwei Pfade, die auseinanderdriften.
2. **Suspendierte EEG kann nicht heimlich reaktiviert werden** — und zwar **ohne
   neue Logik**: Der bestehende Doppel-Buchungs-Schutz blockiert eine erneute
   Buchung, solange die (auch suspendierte) Buchung im System ist. Eine reaktivierte
   EEG bleibt also eine bewusste Owner-Handlung. WHY: kein Risiko, dass Auto-Akzept
   eine Sperre umgeht.
3. **Owner-Benachrichtigung wird FYI**: Inhalt von „bitte prüfen/freigeben" zu
   „EEG X hat gebucht und ist aktiv" umformuliert. Reject/Suspend im BackOffice
   bleibt für den Nachhinein-Fall.
4. **AVV bleibt Pflicht**: AGB+AVV-Häkchen + die synchrone AVV-PDF-Erzeugung bleiben
   Voraussetzung; schlägt die PDF-Erzeugung fehl, scheitert der ganze Vorgang (keine
   Aktivierung ohne AVV-Beleg).

### D) Full-Chain (Umsetzungs-Stellen)
- Buchungs-Service (Submit) ruft die Aktivierungs-Logik inline auf (statt nur
  „submitted" zu schreiben).
- Owner-Benachrichtigungs-Mail: Text → FYI.
- Welcome-Mail: unverändert, feuert jetzt im Buchungs-Schritt.
- BackOffice-Liste/Detail: kein neuer Wartezustand mehr; Suspend/Reject bleibt.
  Nav-Link „Plattform-Buchungen" ist bereits live (committet).

### E) Dependencies
PROJ-71 (Buchungs-/Freigabe-Lifecycle). Keine neuen Pakete, keine Migration.

### Grilling-Findings (2026-06-20, codebasiert verifiziert)
1. **Eine Transaktion, kein „submitted"-Waise.** `ApproveTx` verlangt heute eine
   bereits existierende `submitted`-Zeile (`UPDATE … WHERE status='submitted'`) und
   öffnet eine eigene Tx. Für Auto-Akzept NICHT „CreateSubmission, dann ApproveTx"
   (zwei Tx → Crash dazwischen = hängende `submitted`-Buchung). Stattdessen **eine
   Transaktion** „buchen+aktivieren": Submission direkt als `approved` anlegen +
   Event `activated` + `is_active=true`, alles atomar (advisory-lock wie ApproveTx
   gegen Suspend-Race). `InsertEvent` ist bereits Tx-fähig (`sqlExec`-Param) →
   wiederverwendbar; `CreateSubmission` wird Tx-fähig gemacht oder im neuen
   Tx-Pfad inline (shared helper, [[feedback_shared_helpers_for_parallel_paths]]).
2. **Welcome-Mail bleibt best-effort, NACH Commit** (DB zuerst — Bestand-BUG-2-Fix).
   Der AVV-PDF-Beleg + DB-Commit sind die harte Wahrheit; die Aktivierung darf nicht
   an der Mail-Zustellbarkeit scheitern. Bewusst KEIN Hard-Fail hier (Abweichung von
   [[feedback_mail_hard_fail]] begründet: member-getriggerter Buchungs-Flow, Beleg =
   PDF). Owner-Notification: async, best-effort, Text → FYI.
3. **Event-Herkunft:** der `activated`-Event im Auto-Akzept trägt `reason_code`
   `auto_accept` (statt `owner_approve`), `actor_kind=human`, `actor_subject` = der
   submittende EEG-Admin. So bleibt der Audit-Trail ehrlich (kein fiktiver Owner).
4. **`submitted` wird toter, aber geduldeter Zweig.** Der neue Pfad erzeugt nie mehr
   eine wartende `submitted`-Buchung. BackOffice-„Approve" + `BadgeKindSubmitted` +
   Cockpit-`latest_cos` bleiben **unverändert** (defensive Reserve für etwaige
   Vor-Deploy-Strays — der Owner kann eine stray `submitted` weiterhin manuell
   freigeben). Kein Status-Wert entfernen, kein Cleanup nötig.
5. **Suspendierte Re-Buchung bleibt blockiert** (verifiziert): `HasActiveSubmissionFor`
   = `status IN (submitted,approved)`; Suspend behält `approved` → erneuter Submit →
   `ErrAlreadyActive`. Auto-Akzept kann eine Sperre NICHT umgehen. Null neue Logik.

### AC-Schärfungen
- **AC-1** präzisiert: Buchen+Aktivieren in **einer** DB-Transaktion (kein
  `submitted`-Zwischenzustand persistiert).
- **AC-3** (Event): `activated` mit `reason_code=auto_accept`, `actor=human/submitter`.
- **AC-2** (Welcome-Mail): best-effort nach Commit (kein Hard-Fail).

### Build/Deploy
Gemeinsam mit PROJ-118 deployen (eine Migration in PROJ-118, ein Tag, z. B.
`v1.45.0-PROJ-118-119`) — PROJ-118s Gate liest den `approved`-Zustand, den
Auto-Akzept direkt erzeugt. Nach Grilling: /backend.

## Implementation Notes (Backend+Frontend, 2026-06-20)
Umgesetzt (go build/vet/test + tsc + vitest 252 + build grün), **keine Migration**:
- `repository.go` **SubmitAndActivateTx**: eine Tx — advisory-lock → INSERT submission
  direkt `approved` → InsertEvent `activated`/`auto_accept` → `is_active=true`; 23505 →
  `ErrAlreadyActive`. Kein `submitted`-Waise.
- `service.go` **Submit** ruft jetzt SubmitAndActivateTx (Status direkt approved,
  approved_at/by gesetzt) + **spawnAutoAcceptMails** (Welcome mit AVV-PDF + Owner-FYI,
  beide best-effort NACH Commit). Altes spawnSubmitBackground ersetzt.
- `contract.go` Konstante **ReasonAutoAccept** `auto_accept`.
- Mail: Owner-Notification-Template + Betreff → FYI „gebucht & automatisch aktiviert".
- ApproveTx + BackOffice-Approve BLEIBEN (Reserve für Vor-Deploy-`submitted`-Strays).

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
