# PROJ-46: Stati für Import-Nachbereitung (B2B-Bank-Bestätigung + Aktivierung)

**Status:** Konzept
**Created:** 2026-05-17

## Hintergrund

Das aktuelle Status-Modell endet bei `imported` — sobald der Antrag im
eegFaktura-Core angelegt ist, gilt die Onboarding-Arbeit als erledigt.
In der Praxis fehlen aber zwei Nachbereitungs-Schritte:

1. **B2B-SEPA-Bank-Bestätigung:** B2B-SEPA-Mandate verlangen als
   Mandatsreferenz die Mitgliedsnummer (die erst beim Import vergeben
   wird) UND eine Pre-Notification durch das Mitglied an seine
   Hausbank. Der Onboarding-Prozess muss auf die Member-Bestätigung
   warten, bevor das Mandat scharf geschaltet werden darf. Bei
   nicht-B2B-Einzugsarten (`core`, `kein_sepa`) entfällt dieser Schritt.

2. **Aktivierung in der EEG:** Auch nach erfolgreichem Import + Bank-
   Bestätigung muss das Mitglied final in der EEG aktiviert werden
   (im eegFaktura-Core). Bisher unsichtbar — soll explizit als Status
   geführt werden, mit manueller Weiterschaltung UND Automatik (Polling
   des Core-Aktivierungsstatus).

Zusätzlich: das aktuelle Approval-PDF wird bei `→ approved` ohne
Mitgliedsnummer generiert. Das B2B-SEPA-Mandat darin ist dadurch
unvollständig. PDF-Generierung muss nach hinten auf `→ imported`
(wenn die Mitgliedsnummer steht).

## Status-Modell-Erweiterung

Drei neue Status-Werte:

- `awaiting_bank_confirmation` — nur bei `einzugsart=b2b`. Wartet auf
  Admin-Bestätigung, dass das Mitglied seine Bank über das B2B-Mandat
  informiert hat.
- `ready_for_activation` — Mitglied kann in der EEG aufgenommen werden.
  Endzustand vor der Core-Aktivierung.
- `activated` — Mitglied ist in der EEG aktiv. Endzustand des
  Onboarding-Prozesses.

## Neuer Flow

```
approved
  ↓ (Import erfolgreich + Mitgliedsnummer vergeben)
imported
  ↓ (Auto-Branch beim Service nach Import-Erfolg)
  ├─ einzugsart=b2b →  awaiting_bank_confirmation
  │                      ↓ (Admin manuell, Member meldet sich beim Admin)
  │                    ready_for_activation
  │
  └─ sonst →  ready_for_activation (Auto-Skip)
                ↓ (Admin manuell ODER Polling-Automatik)
              activated
```

### Erlaubte Übergänge

Neu hinzu (über bestehende hinaus):
- `imported → awaiting_bank_confirmation` *(Auto bei `einzugsart=b2b`)*
- `imported → ready_for_activation` *(Auto bei nicht-b2b)*
- `awaiting_bank_confirmation → ready_for_activation` *(manuell Admin)*
- `ready_for_activation → activated` *(manuell Admin ODER Polling-Job)*

Rückwärts (User-Wunsch 5):
- `awaiting_bank_confirmation → under_review` *(Admin-Override)*
- `ready_for_activation → under_review` *(Admin-Override)*

Reset-Import-Erweiterung (User-Wunsch 6, PROJ-30-Erweiterung):
- `imported → approved` *(bestehend)*
- `awaiting_bank_confirmation → approved` *(neu, läuft über `/reset-import`)*
- `ready_for_activation → approved` *(neu)*
- `activated → approved` *(neu — siehe offene Frage A)*

Alle Resets löschen `member_number` + schreiben Audit-Trail-Eintrag in
`status_log` (bestehende PROJ-30-Mechanik wird wiederverwendet).

### Status-Reihenfolge im DB-CHECK-Constraint

```sql
status IN (
  'draft','submitted','email_confirmed','under_review','needs_info',
  'approved','rejected','imported','import_failed',
  'awaiting_bank_confirmation','ready_for_activation','activated'
)
```

## Datenmodell-Erweiterung

Neue Spalten auf `application`:
- `bank_confirmed_at` TIMESTAMPTZ NULL — wann der Admin die Bank-
  Bestätigung gesetzt hat (Audit-Trail). NULL solange noch nicht
  bestätigt; bleibt gesetzt, auch wenn Status weiterläuft.
- `activated_at` TIMESTAMPTZ NULL — wann das Mitglied im Core aktiv
  wurde (manuell vom Admin oder automatisch vom Polling-Job).

Keine neue Tabelle nötig — alle Übergänge laufen über den bestehenden
`status_log`.

Migration: `db/migrations/000041_post_import_statuses.up.sql`.

## PDF-Timing-Umstellung (User-Wunsch 2 + 3)

**Bisher:** Bei `→ approved` wird die Beitrittsbestätigungs-PDF (mit
optionalem SEPA-Mandat) gebaut und an EEG-Contact geschickt — ohne
Mitgliedsnummer.

**Neu:**
- Bei `→ imported` wird die PDF gebaut (Mitgliedsnummer vorhanden) und
  an das Mitglied geschickt; EEG-Contact bekommt eine separate Kopie
  (Reply-To = Member, damit die EEG auf Bestätigungs-Rückfragen direkt
  antworten kann)
- Bei `einzugsart=b2b` enthält die Mail explizit den Hinweis, dass das
  Mitglied seine Bank über das B2B-Mandat informieren muss + dass die
  EEG dafür eine Rückmeldung erwartet
- Die bestehenden zwei SEPA-Mandat-Varianten (Text-Inline in der
  Beitrittsbestätigung + separates PDF für SEPA-pflichtige EEGs)
  bleiben unverändert — nur das **Timing** verschiebt sich. (User-
  Wunsch 3: „es gibt jetzt schon die Text- und PDF-Variante. diese
  soll es weiterhin geben.")

**Bisheriger `approved`-Mail-Pfad:** entfällt komplett. Approval ist
jetzt ein reiner Zwischenstatus „freigegeben für Import", ohne externe
Kommunikation.

## Member-Mail-Trigger

| Übergang | Mail an Member | Inhalt |
|---|---|---|
| `→ imported` | ✅ neu | Beitrittsbestätigungs-PDF + (bei b2b) B2B-SEPA-Mandat-Hinweis |
| `→ awaiting_bank_confirmation` | ❌ (Status implizit via imported-Mail kommuniziert) | — |
| `→ ready_for_activation` | ❌ | rein interner Zwischenstatus |
| `→ activated` | ✅ neu | „Willkommen, Sie sind nun aktiv in der EEG" |

EEG-Mail-Trigger (zusätzlich):
| `→ imported` | Kopie der Beitrittsbestätigung an `contact_email` |
| `→ awaiting_bank_confirmation` | Hinweis „Member wartet auf Bank-Bestätigung — bitte bei Rückmeldung Status weiterschalten" |
| `→ activated` | optional Kopie — Admin sieht den Übergang sowieso im UI |

## Polling-Automatik für `→ activated` (User-Wunsch 4)

Ein periodischer Job (Cron, z.B. stündlich) iteriert über alle Anträge
in Status `ready_for_activation` und fragt eegFaktura-Core, ob das
Mitglied dort aktiv ist. Bei „aktiv" → automatischer Übergang auf
`activated`, `activated_at = NOW()`, Status-Log-Eintrag mit
`actor=system`.

**Benötigt Core-API-Endpoint** zum Aktivitäts-Check. Falls noch nicht
vorhanden → separates Vor-Ticket. Bis dahin: nur manuelle Aktivierung
durch den Admin, Polling-Job als Folge-Implementierung.

Konfigurierbar pro EEG via neuem Boolean
`registration_entrypoint.auto_activation_enabled` (Default `false`).
Admin kann den Automatismus einschalten, sobald die Core-Anbindung
verifiziert ist.

## Admin-UI-Erweiterung

- **Status-Aktions-Buttons** in `admin-status-actions.tsx`: neue
  Buttons für `→ awaiting_bank_confirmation` (Auto-trigger nicht
  manuell anwählbar), `→ ready_for_activation`, `→ activated`.
- **Status-Badge-Farben** für die drei neuen Stati definieren.
- **Reset-Import-Dialog** erweitern: Quellstatus kann jetzt auch
  `awaiting_bank_confirmation` / `ready_for_activation` / `activated`
  sein.
- **Detail-Page:** wenn Status `awaiting_bank_confirmation`, prominente
  Hinweisbox „Auf Member-Rückmeldung warten".

## Offene Fragen

**A) Reset aus `activated`:** User-Antwort 6 erwähnt nur die zwei
neuen mittleren Stati. `activated → approved` wäre möglich, aber
heißt: ein aktives Mitglied wird wieder zum Bewerber. Ich würde es
**erlauben**, aber im UI eine Extra-Warnung anzeigen
(„Mitglied ist aktiv — Reset entfernt es aus der EEG"). Bestätigen?

**B) Polling-Intervall + Core-API:** Stündlich? Täglich? Und gibt es
schon einen Core-Endpoint zum Aktivitäts-Check, oder muss der erst
gebaut werden?

**C) Auto-Übergang `imported → awaiting_bank_confirmation` bzw.
`ready_for_activation`:** soll der Übergang im selben DB-Commit wie
der `→ imported`-Übergang passieren, oder als zweiter Commit direkt
danach? Sauberer: zweistufig im selben Service-Aufruf (`imported`
wird nie länger als ein paar Millisekunden bestehen). Bestätigen?

## Out of Scope V1

- Self-Service-Bestätigung durch Member per Mail-Link (Variante 1a) —
  bewusst nicht V1 (User-Wunsch 1: Variante b = Admin manuell)
- Member-Notifikation bei `→ ready_for_activation` (rein intern)
- Konfigurierbarer Wortlaut der Bank-Hinweis-Mail pro EEG
- Multi-Stage-Polling (z.B. nicht aktiv nach N Tagen → Eskalations-Mail)

## Tests

- Migration: Bestandsdaten — bestehende `imported`-Anträge bleiben
  unverändert (keine Auto-Migration auf neue Stati, der Admin entscheidet
  manuell wie weiter)
- Smoke-Test 1: B2B-SEPA-Antrag → Import → automatisch
  `awaiting_bank_confirmation` → Admin setzt auf `ready_for_activation`
  → Admin setzt auf `activated`
- Smoke-Test 2: Nicht-B2B-Antrag → Import → automatisch
  `ready_for_activation` → Admin setzt auf `activated`
- Smoke-Test 3: Reset-Import aus jedem der neuen Stati → zurück auf
  `approved`, Mitgliedsnummer gelöscht, Status-Log-Eintrag mit Reason
- Smoke-Test 4: Rückwärts-Übergang `awaiting_bank_confirmation →
  under_review`

## Empfohlene Implementierungs-Stages

Da der Scope groß ist:

1. **Stage A — DB + Backend + Übergänge** (ohne Mails, ohne Polling):
   Migration, Status-Enums, Service-Logik, Reset-Import-Erweiterung.
2. **Stage B — PDF-Timing + Member/EEG-Mails:** Verschiebung der
   PDF-Generierung, neue Mail-Templates für Bank-Hinweis und
   Aktivierungs-Bestätigung.
3. **Stage C — Admin-UI:** Status-Buttons, Detail-Page-Banner,
   Reset-Dialog.
4. **Stage D — Polling-Automatik:** Cron-Job + Core-API-Endpoint
   (vorausgesetzt der Endpoint existiert oder wird hier mitgebaut).

Stages A–C sind in einem Sprint machbar; D hängt am Core.
