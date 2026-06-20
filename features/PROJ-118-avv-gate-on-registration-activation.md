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

### Kernidee
Die öffentliche Registrierung darf nur scharf werden, wenn die EEG den AVV
akzeptiert hat. „AVV akzeptiert" = es existiert eine **freigegebene, nicht
suspendierte Plattform-Buchung** (die erzwingt AGB+AVV+PDF-Beleg). Alle anderen
Wege zu „aktiv" werden geschlossen.

### A) Wo das Gate sitzt
```
Settings-Toggle „Registrierung aktiv" (EIN)
   └─ Gate-Prüfung: existiert freigegebene + aktive Buchung?  ── nein ──► 409 „booking_required"
                                                                            (Meldung: „Bitte zuerst
                                                                             Plattform buchen")
                              │ ja
                              ▼
                       is_active = true

Buchung → Auto-Akzept (PROJ-119)  ──►  setzt is_active=true selbst (sanktionierter Weg, kein Gate)
Öffentliche Registrierungs-Seite  ──►  prüft is_active schon (inaktiv → 404/410), UNVERÄNDERT
```

### B) Daten-Modell (Klartext) + Migration
Keine neue Tabelle. Eine **neue Migration** (die alte 000002 bleibt unangetastet —
Migrationen sind nach Apply unveränderlich) mit drei Teilen:
1. **Default-Umstellung:** Der Standardwert von „Registrierung aktiv" wird von
   *an* auf *aus* gesetzt. Wirkt nur auf **neu** angelegte EEG-Einträge —
   bestehende Werte bleiben unangetastet.
2. **Neue Ja/Nein-Eigenschaft „bestandsgeschützt aktiviert"** am EEG-Eintrag
   (Standard: nein).
3. **Einmaliger Bestandsschutz-Lauf:** Jede EEG, die *jetzt* aktiv ist, aber **keine**
   freigegebene Buchung hat, wird als „bestandsgeschützt aktiviert" markiert.

**Gate-Prädikat (zentral, der Kern-Stolperstein):** Aktivierung erlaubt, wenn
**(freigegebene Buchung existiert UND Vertrag nicht suspendiert)** ODER
**(„bestandsgeschützt aktiviert" = ja)**. Die reine „Vertrag aktiv?"-Prüfung genügt
NICHT, weil sie für eine *nie gebuchte* EEG fälschlich „aktiv" liefert
(Bestandsschutz-Netz) — deshalb muss zuerst die **freigegebene Buchung** nachgewiesen
werden.

### C) Tech-Entscheidungen (WHY)
1. **Bestandsschutz als eigene Ja/Nein-Eigenschaft (Variante ii)** — nicht über einen
   Eintrag im Vertrags-Ereignis-Log (Variante i) und nicht „ohne Marker" (Variante
   iii). WHY:
   - *Variante iii (kein Marker)* würde zwar bereits-aktive EEGs in Ruhe lassen, aber
     eine bestandsgeschützte EEG könnte sich nach einem Aus→Ein nicht mehr ohne
     Buchung reaktivieren — Owner-Wunsch („dürfen reaktivieren") verletzt.
   - *Variante i (Log-Ereignis)* vermischt die Registrierungs-Aktivierung mit der
     Vertrags-Semantik und müsste trotzdem zusätzlich geprüft werden — das Gate
     verlangt ohnehin den Buchungs-Nachweis.
   - *Variante ii (Eigenschaft am EEG-Eintrag)* ist genau dort, wo das Gate ohnehin
     liest, explizit und einfach abfragbar. Hinweis: `registration_entrypoint` hat
     bereits viele Spalten ([[project_todo_db_design_review]]) — eine bewusste,
     gut begründete Ergänzung.
2. **Owner-/Test-Ausnahme braucht KEINEN Sonderpfad.** Bestehende Test-/Owner-EEGs
   sind durch den Bestandsschutz-Lauf abgedeckt; **neue** Test-EEGs bucht der Owner
   einfach selbst — was dank **PROJ-119 Auto-Akzept** ein Ein-Klick-Schritt ist (sofort
   aktiv). Schöne Kopplung: Auto-Akzept macht die „Aktivierung ohne Buchung"-Ausnahme
   überflüssig.
3. **Öffentlicher Endpoint unverändert:** Er prüft schon `is_active`; das Gate sitzt
   eine Ebene davor (auf der Aktivierungs-Umschaltung), nicht im Public-Pfad.
4. **Reihenfolge der Migration:** erst neue Eigenschaft anlegen, dann den Bestands-Lauf
   (Daten füllen), dann ist der Default-Flip unabhängig — keine „Constraint vor
   Daten"-Falle ([[feedback_migration_constraint_before_data]]).

### D) Full-Chain (Umsetzungs-Stellen)
- **Migration** (Default-Flip + neue Eigenschaft + Bestands-Lauf) — **Schema-Change →
  Owner-Approval-Checkpoint**, Migration wird vor Apply gezeigt.
- EEG-Eintrag-Repository: neue Eigenschaft lesen; Helfer „darf diese EEG aktiviert
  werden?" (Buchungs-Nachweis + Vertrag-aktiv kombiniert — der Buchungs-Check kommt
  aus dem Customer-Onboarding-Bereich, wird als Prüf-Funktion in den Settings-Handler
  injiziert, analog zum bestehenden Vertrags-Checker).
- Settings-Speichern-Pfad: Gate vor der Aktivierungs-Umschaltung; klare Fehlermeldung
  + Fehlercode.
- Admin-Settings-Oberfläche: das Gate sichtbar machen — Aktivierungs-Schalter
  deaktiviert/mit Hinweis „erst Plattform buchen", statt erst beim Speichern zu
  scheitern.

### E) Dependencies
PROJ-71 (Buchung liefert den AVV-Nachweis), gekoppelt mit PROJ-119 (erzeugt den
„freigegeben"-Zustand). Keine neuen Pakete.

### Grilling-Findings (2026-06-20, codebasiert verifiziert)
1. **Gate-Scope bestätigt: nur der Settings-Aktivierungs-Pfad.** `is_active=true`
   wird im Code NUR an zwei Stellen geschrieben: `SaveIsActive` (Settings-Toggle —
   braucht das Gate) und `ApproveTx`/der Buchungs-Pfad (hat per Definition eine
   Buchung → KEIN Gate). DB-Default wird per Migration FALSE; manueller INSERT ist
   Owner-Sache. `SaveEEGSettings` setzt `is_active` NICHT (separater Pfad). → Das
   Gate sitzt an genau einer Stelle.
2. **Gate-Prädikat = `grandfathered ODER (freigegebene Buchung UND Vertrag aktiv)`.**
   „Freigegebene Buchung" = es gibt eine `approved`-Submission (`FindApprovedForRC`);
   „Vertrag aktiv" = `CheckContract.Active` (schließt suspendiert/terminiert aus).
   Eine suspendierte EEG hat zwar eine `approved`-Submission, ist aber nicht aktiv →
   Gate blockt korrekt (Reaktivierung nur durch Owner). Eine nie gebuchte EEG hat
   keine `approved`-Submission → Gate blockt, obwohl `CheckContract` allein „aktiv"
   liefern würde (Bestandsschutz-Netz) — genau der Grund, warum der Buchungs-Nachweis
   zuerst kommt.
3. **Cross-Package sauber:** ein **einziger injizierter Checker**
   `hasApprovedActiveBooking(rc) bool` (im `customeronboarding`-Paket gebaut:
   `FindApprovedForRC` + `CheckContract.Active`), analog zum bestehenden
   `customerContractChecker` in main.go. Der Settings-Handler liest
   `activation_grandfathered` aus dem schon geladenen Entrypoint und ODER-t:
   `grandfathered || hasApprovedActiveBooking`. Kein direkter Repo-Import im
   HTTP-Layer.
4. **Migration (neue Datei, 000002 unangetastet — [[feedback_migration_after_apply_drift]]):**
   Reihenfolge: (a) `ADD COLUMN activation_grandfathered BOOLEAN NOT NULL DEFAULT
   FALSE`; (b) Bestands-Lauf `UPDATE … SET activation_grandfathered=true WHERE
   is_active=true AND NOT EXISTS (approved submission für die rc)`; (c) `ALTER COLUMN
   is_active SET DEFAULT FALSE`. Kein CHECK → keine „Constraint-vor-Daten"-Falle.
   **Down-Migration:** Spalte droppen + Default wieder TRUE. Auf Prod trifft der
   Bestands-Lauf genau TE100200 (aktiv, keine Buchung) → grandfathered; bestehende
   `is_active`-Werte bleiben unangetastet. Dirty-Recovery siehe
   [[reference_migrate_dirty_flag_recovery]].
5. **Settings-UI-Gate (Full-Chain):** Die `GetEEGSettings`-Response bekommt ein
   berechnetes Feld **`canActivateRegistration`** (= `grandfathered ||
   hasApprovedActiveBooking`). Das Frontend deaktiviert den Aktivierungs-Schalter
   bzw. zeigt den Hinweis „erst Plattform buchen", wenn `!canActivateRegistration &&
   nicht bereits aktiv` — statt erst beim Speichern zu scheitern
   ([[feedback_admin_field_full_chain]]: Feld in der bestehenden Response, kein
   Extra-Call). Server-seitiges 409-Gate bleibt die harte Linie.

### AC-Schärfungen
- **AC-1** präzisiert: Gate sitzt ausschließlich am `SaveIsActive`-true-Zweig;
  Prädikat `grandfathered || (FindApprovedForRC && CheckContract.Active)`.
- **AC-6 (UI)** präzisiert: `canActivateRegistration`-Feld in `GetEEGSettings`.
- Neuer **AC:** Migration enthält Down-Pfad; Bestands-Lauf markiert nur
  aktuell-aktive-ohne-Buchung.

### Build/Deploy
Gemeinsam mit PROJ-119 (eine Migration nur hier, ein Tag `v1.45.0-PROJ-118-119`).
PROJ-118 allein wäre nicht kaputt (Aktivierung ginge dann nur über manuellen
Owner-Approve = mehr Reibung), aber gekoppelt ist es rund. **Schema-Change →
Owner-Approval-Checkpoint:** Migration wird im /backend-Schritt vor Apply gezeigt.
Nach Grilling: /backend.

## Implementation Notes (Backend+Frontend, 2026-06-20)
Umgesetzt (go build/vet/test + tsc + vitest 252 + build grün):
- **Migration 000091** (`avv_gate_activation_grandfathered`): up = ADD COLUMN
  `activation_grandfathered` → Bestands-Lauf (aktiv ohne approved-Buchung → TRUE) →
  `is_active` DEFAULT FALSE; down = DEFAULT TRUE + DROP COLUMN. **Vor Apply dem Owner
  gezeigt** (Schema-Checkpoint). Apply via helm-Migrate-Job beim Owner-`helm upgrade`.
- `shared/models.go` Feld `ActivationGrandfathered`; `registration_entrypoint_repo.go`
  GetByRCNumber SELECT+Scan ergänzt.
- `contract.go` **HasApprovedActiveBooking** (FindApprovedForRC-Existenz UND
  CheckContract.Active).
- `admin.go` **ActivationBookingCheckerFunc** + SetActivationBookingChecker +
  **activationAllowed**(ep) = `grandfathered || checker` (nil-Checker = Gate offen).
- `admin_settings_eeg.go`: **Gate** im SaveEEGSettings (RegistrationActive==true ohne
  Buchung+nicht-grandfathered → **409 `booking_required`**); **canActivateRegistration**
  in GetEEGSettings-Response.
- `main.go` injiziert den Checker (`customeronboarding.HasApprovedActiveBooking`).
- Frontend: `EEGSettings.canActivateRegistration`; Aktivierungs-Toggle **disabled** +
  Info-Popover „erst Plattform buchen", wenn `canActivate===false && nicht aktiv`.
- Test: `activationAllowed`-Wahrheitstabelle (DB-frei). DB-Pfade (Gate-409,
  HasApprovedActiveBooking, Migration) → QA/E2E nach Deploy.

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
