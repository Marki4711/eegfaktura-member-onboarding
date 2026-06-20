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
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
