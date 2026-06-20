# PROJ-118: AVV-Gate für Registrierungs-Aktivierung

## Status: Planned
**Created:** 2026-06-20
**Last Updated:** 2026-06-20

## Kontext & Motivation

Der Plattform-Betreiber ist Auftragsverarbeiter (Processor), die EEG ist
Verantwortliche (Controller). Sobald die öffentliche Member-Registrierung scharf
ist, sammelt die Plattform personenbezogene Mitgliederdaten im Auftrag der EEG.
DSGVO verlangt dafür einen wirksamen **Auftragsverarbeitungsvertrag (AVV)**.

**Befund (2026-06-20):** Die AVV-Zustimmung wird im „Plattform buchen"-Flow
(PROJ-71) zwar **erfasst** (Submit erzwingt `avvAccepted=true`, ein AVV-Akzept-PDF
wird als Beleg gespeichert, Owner-Approve setzt `is_active=true`), aber sie ist
**nicht als harte Sperre** auf die Registrierungs-Aktivierung verdrahtet. Es gibt
mehrere Wege zu `is_active=true`, und nur einer geht durch den AVV:

| Pfad zu `is_active=true` | AVV erzwungen? |
|---|---|
| Plattform buchen → Owner-Approve (`ApproveTx`) | ✅ ja |
| Settings-Toggle „Registrierung aktiv" (`SaveIsActive`, [admin_settings_eeg.go:282](../internal/http/admin_settings_eeg.go#L282)) | ❌ nein |
| DB-Default `is_active DEFAULT TRUE` (Migration 000002) / manueller INSERT | ❌ nein |

Der einzige Wächter auf den Settings-Endpoints (`enforceCustomerContract`)
blockt nur **explizit Suspendierte**: eine nie gebuchte EEG (kein Event im
`customer_onboarding_event_log`) liefert `CheckContract` → `Active: true`
([contract.go:44-48](../internal/customeronboarding/contract.go#L44-L48)) und
wird durchgelassen. Ergebnis: Die öffentliche Registrierung kann Mitglieder-PII
sammeln, **ohne dass je ein AVV akzeptiert wurde**.

**Günstiges Fenster:** Aktuell ist noch kein echter EEG auf Prod (nur die manuell
angelegte Test-EEG TE100200) — die Lücke lässt sich jetzt schließen, bevor
Bestandsdaten eine Migration teuer machen.

## Dependencies
- Requires: PROJ-71 (EEG-Customer-Onboarding) — liefert den AVV-erfassenden
  Buchungs-Flow + den Contract-Event-Log.
- Betrifft: PROJ-19 (Manuelle Aktivierung der Registrierung) — der Settings-Toggle,
  der gegated wird.
- Koordiniert mit: PROJ-119 (Auto-Akzept der Plattform-Buchung) — durch Auto-Akzept
  wird der `approved`-Zustand bereits mit dem Submit erreicht (kein manueller
  Owner-Approve mehr). Das Gate-Prädikat („freigegebene Submission existiert + nicht
  suspendiert") bleibt davon unberührt — es liest nur den `approved`-Zustand,
  unabhängig davon, wie er erreicht wurde. Empfehlung: gemeinsam bauen/deployen.

## User Stories
- Als **Plattform-Betreiber (Auftragsverarbeiter)** will ich, dass die öffentliche
  Member-Registrierung nur nach AVV-Zustimmung (über eine freigegebene Buchung)
  scharf geschaltet werden kann, damit keine Mitglieder-PII ohne gültigen AVV
  erhoben wird.
- Als **EEG-Admin** will ich eine klare Meldung, wenn ich die Registrierung ohne
  vorherige Buchung aktivieren will, damit ich weiß, dass ich zuerst „Plattform
  buchen" abschließen muss.
- Als **Plattform-Betreiber** will ich, dass bereits aktive Test-EEGs nach dem
  Rollout weiterlaufen, damit die Test-Zone nicht unterbrochen wird.
- Als **Plattform-Betreiber** will ich, dass keine `registration_entrypoint`-Zeile
  per Default still aktiv ist, damit manuell/Sync-erzeugte Zeilen das AVV-Gate
  nicht umgehen.

## Owner-Entscheidungen (2026-06-20)
1. **Gate-Definition:** Aktivierung nur erlaubt, wenn eine vom Owner **freigegebene
   Plattform-Buchung** existiert (Customer-Onboarding-Submission im Status
   `approved`, Contract **nicht** suspendiert). Diese erzwingt heute schon
   AGB+AVV+PDF-Beleg.
2. **Bestand:** Bereits aktive EEGs ohne Buchung werden **grandfathered** (bleiben
   aktiv, Legacy-Marker). Das Gate gilt nur für Aktivierungen ab Deploy.
3. **DB-Default:** `registration_entrypoint.is_active` wird auf **FALSE** geflippt,
   damit neue Zeilen nie still aktiv sind.

## Acceptance Criteria
- [ ] **AC-1 (Gate, server-seitig):** `RegistrationActive=true` über den
  Settings-Endpoint wird abgelehnt (HTTP 409 bzw. 423, Code z.B.
  `booking_required`), **außer** die EEG hat eine vom Owner freigegebene
  Customer-Onboarding-Buchung, die nicht suspendiert ist.
- [ ] **AC-2 (sanktionierter Pfad):** Der Buchung→Owner-Approve-Pfad
  (`ApproveTx`) setzt `is_active=true` unverändert — er ist der legitime
  Aktivierungsweg und wird nicht durchs Gate blockiert.
- [ ] **AC-3 (Deaktivieren bleibt frei):** `RegistrationActive=false`
  (Pausieren/Deaktivieren) ist unabhängig vom Buchungsstatus weiterhin erlaubt.
- [ ] **AC-4 (DB-Default):** Der Default von `registration_entrypoint.is_active`
  ist `FALSE`; neu eingefügte Zeilen (Auto-Sync, manueller INSERT) sind inaktiv,
  bis das Gate passiert wird. Bestehende Zeilen werden durch die Migration **nicht**
  deaktiviert (siehe AC-5).
- [ ] **AC-5 (Bestandsschutz):** EEGs, die zum Deploy-Zeitpunkt `is_active=true`
  sind, aber keine freigegebene Buchung haben, bleiben aktiv und erhalten einen
  Legacy-Marker; sie dürfen ohne neue Buchung deaktiviert und reaktiviert werden
  (Legacy-Status bleibt erhalten).
- [ ] **AC-6 (UI-Transparenz):** Die Settings-Oberfläche macht das Gate sichtbar —
  fehlt eine freigegebene Buchung, ist der Aktivierungs-Toggle deaktiviert oder
  zeigt einen Inline-Hinweis, der auf „Plattform buchen" verweist, statt erst beim
  Speichern zu scheitern.
- [ ] **AC-7 (klare Meldung):** Blockt das Gate, ist die Fehlermeldung in klarem
  Deutsch und benennt den nächsten Schritt („Bitte zuerst die Plattform buchen und
  auf die Freigabe warten.").
- [ ] **AC-8 (kein Bypass):** Das Gate wird server-seitig erzwungen; der
  Frontend-Hinweis ist rein informativ und kann es nicht umgehen.

## Edge Cases
- **EEG gebucht, aber suspendiert (Cool-Down):** Aktivierung blockiert (Contract
  nicht aktiv). Konsistent zum bestehenden Suspend-Guard für Mehrwert-Endpoints.
- **EEG nie gebucht, kein Event-Log:** muss blockiert werden — **obwohl**
  `CheckContract` hier `Active: true` als Bestandsschutz-Sicherheitsnetz liefert.
  → Das Gate-Prädikat muss auf eine **freigegebene Submission** (`FindApprovedForRC`
  + nicht suspendiert) prüfen, **nicht** auf `CheckContract.Active` allein. Dies ist
  der zentrale Implementierungs-Stolperstein.
- **Superuser/Betreiber-Test-EEGs:** Empfehlung — superuser-initiierte Aktivierung
  umgeht das Gate (protokolliert), damit der Betreiber Test-EEGs ohne vollständige
  Buchung hochziehen kann. Offen für /architecture-Bestätigung.
- **AVV-Versions-Bump nach Aktivierung:** Out of Scope V1 — aktive EEGs werden
  **nicht** automatisch re-gegated (siehe Non-Goals).
- **Grandfather-Marker vs. Contract-Event-Log:** /architecture entscheidet den
  Mechanismus (dedizierte Spalte/Flag vs. Event-Marker), damit der Legacy-Status
  die Customer-Onboarding-Contract-Semantik nicht verfälscht.
- **Race (Buchung wird zwischen Gate-Check und Aktivierung suspendiert):**
  Last-Write-Wins akzeptabel; der Suspend-Guard deckt die Mehrwert-Pfade ab.

## Non-Goals
- Re-Akzept / Re-Gating bei AVV-Versions-Bumps.
- Änderung des AVV-/AGB-Inhalts oder des Buchungs-Flows selbst.
- Deaktivieren bereits aktiver, grandfathered EEGs.
- Gating auf Antrags- oder Zählpunkt-Ebene (dieses Gate ist rein per-EEG-Aktivierung).

## Technical Requirements / Notes
- **Sicherheit & Approval:** Berührt Aktivierungs-/Auth-Logik **und** ein
  DB-Schema-Migration → `/security-review` Pflicht; Human-Approval-Checkpoint für
  Schema- + Auth-Änderung (CLAUDE.md).
- **Migration:** (a) Default von `registration_entrypoint.is_active` auf `FALSE`
  flippen; (b) einmaliger Grandfather-Pass für aktuell aktive EEGs ohne freigegebene
  Buchung. Bestehende Daten dürfen nicht deaktiviert werden.
- **Gate-Prädikat:** „freigegebene Buchung existiert UND nicht suspendiert" —
  **nicht** `CheckContract.Active` allein (siehe Edge Cases).
- Keine Änderung am öffentlichen Registrierungs-Endpoint nötig: der prüft bereits
  `is_active` (inaktive/unbekannte RC → 404/410). Das Gate sitzt eine Ebene davor
  (Aktivierungs-Transition).

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
