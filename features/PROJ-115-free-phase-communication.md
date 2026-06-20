# PROJ-115: Free-Phase-Kommunikation & No-Charge-Gate

## Status: In Progress
**Created:** 2026-06-20
**Last Updated:** 2026-06-20

## Überblick / Kontext

Strategie-Änderung Owner 2026-06-20: member-onboarding geht **zunächst kostenlos** live. Den EEGs wird kommuniziert, dass eine **kostenpflichtige Nutzung geplant** ist und **rechtzeitig vorab angekündigt** wird.

Technische Grundlage (verifiziert): Der Abrechnungs-Apparat ist bereits schlafend. `IsLive = globalLiveMode && eeg.billing_live` ([internal/billing/live_mode.go](../internal/billing/live_mode.go)); beide stehen auf `false` → Mock-Clients, keine FreeFinance/Mollie-Calls, Rechnungen `status='preview'`. Der Quartals-Cron legt im Preview-Modus nur Preview-Rechnungen an und sendet **keine** EEG-facing Mail; die einzigen automatischen Mails sind Owner-facing (Cron-Fehler-Alert, „Lauf bereit zur Freigabe" — letzterer feuert im Preview-Modus gar nicht, da `InvoicesPendingApproval` 0 bleibt). Die zwei EEG-facing Billing-Mails (`MandateSetup`, `CreditNote`) sind manuelle Admin-Aktionen.

Dieses Feature macht die Free-Phase **sichtbar** und **abgesichert** — es schaltet **keine** echte Abrechnung scharf.

## User Stories

1. Als **EEG-Vorstand** möchte ich im Rechnungen-Tab klar sehen, dass die Nutzung aktuell kostenlos ist und eine spätere Kostenpflicht vorab angekündigt wird, damit ich keine unerwartete Rechnung befürchte.
2. Als **EEG-Vorstand** möchte ich die im Tab gezeigten Beträge eindeutig als unverbindliche Vorschau erkennen, damit ich sie nicht für fällige Forderungen halte.
3. Als **EEG-Vorstand** möchte ich beim Trial-Hinweis nicht den Eindruck „danach automatisch kostenpflichtig" bekommen, solange die kostenlose Phase gilt.
4. Als **Plattform-Betreiber** möchte ich sicher sein, dass in der kostenlosen Phase keine zahlungs- oder mahnbezogene Mail an eine EEG rausgeht — auch nicht versehentlich durch einen manuellen Klick.

## Acceptance Criteria

### Free-Banner (Rechnungen-Tab)
- **AC-1:** Im Rechnungen-Tab erscheint ein gut sichtbarer Hinweis-Banner sinngemäß: „Aktuell kostenlos. Eine kostenpflichtige Nutzung ist geplant und wird rechtzeitig vorab angekündigt." (Exakter Wortlaut Owner-approbiert in /architecture.)
- **AC-2:** Preview-Rechnungen tragen ein klares **„Vorschau"-Badge** (Statuslabel `preview`: „Preview" → „Vorschau"), gekeyt auf `status==='preview'` **pro Rechnung** (nicht auf das globale Banner-Flag) — eine spätere echte `sent`/`paid`-Rechnung wird nie fälschlich als Vorschau markiert.
- **AC-3:** Banner erscheint nur im **Free-Zustand** der EEG. Quelle: der Tenant-Endpoint `GET /api/admin/eeg/{rc}/invoices` liefert ein berechnetes `billingLive` (= `IsLive(globalLiveMode, eeg)`); Banner zeigt wenn `billingLive=false`. Deckt Mischzustand (EC-1) + paid-Cutover (EC-3) automatisch ab, ohne Redeploy. `globalLiveMode` wird **nicht** roh ans Frontend geleakt.
- **AC-4:** Der Hinweis erscheint **nur** im Rechnungen-Tab (keine weiteren Stellen — Owner-Entscheidung 2026-06-20).

### Trial-Marker — OUT OF SCOPE (Grilling 2026-06-20)
Der Trial-Marker wird dem **EEG-Vorstand gar nicht** angezeigt — `trialStartedAt`/`trial_period` lebt nur im Owner-Cockpit ([billing-eeg-table.tsx](../src/components/billing/billing-eeg-table.tsx)) + dem Period-Note. Im Tenant-Rechnungen-Tab gibt es nichts umzutexten. Die ursprünglichen AC-5/AC-6 **entfallen**; die Free-Botschaft tragen vollständig der Banner (AC-1) + das „Vorschau"-Label (AC-2). Owner-Cockpit-Trial-Wording bleibt unverändert (Owner kennt die Mechanik).

### No-Charge-Gate
- **AC-5:** Ein Quartals- **und** Daily-Cron-Lauf bei `globalLiveMode=false` sendet **keine** EEG-facing Zahlungs-/Mahn-Mail (nur Preview-Rechnungen; Owner-facing Alerts erlaubt). Per QA mit echtem Cron-Lauf verifiziert. (Befund: es existiert ohnehin **kein** Dunning-Mail an EEGs — Rechnung macht FreeFinance, Einzug Mollie, beide hinter `IsLive`/Approval.)
- **AC-6:** Bei `globalLiveMode=false` blockiert ein Guard die **zwei** EEG-mail-auslösenden Owner-Aktionen am Handler-Eingang: `SetBillingLive` (true-Zweig — enthält `SendBillingMandateSetup` + Mollie-Mandate-Trigger, [admin_billing.go:381](../internal/http/admin_billing.go#L381)) und `CreateCreditNote` (`SendBillingCreditNote`). Antwort **409 Conflict** + Meldung „In der kostenlosen Phase nicht verfügbar — der globale Live-Modus ist deaktiviert." Kein Mailversand, kein Vendor-Call. `SetBillingLive(false)` (Off-Toggle) bleibt erlaubt.

## Edge Cases

- **EC-1 (Mischzustand):** EEG mit `billing_live=true`, aber `globalLiveMode=false` → gilt als nicht-live (`IsLive` UND-Logik) → Banner zeigt, Guard greift.
- **EC-2 (Owner-Footgun):** Owner klickt MandateSetup/CreditNote im Free-Modus → 4xx mit Hinweis, kein Mailversand, kein Vendor-Call, sauberes Audit-„skip".
- **EC-3 (paid-Cutover):** Owner setzt später `globalLiveMode=true` → Banner verschwindet automatisch, Trial-Wording regulär, Guard hebt sich auf — kein Daten-/Migrations-Eingriff.
- **EC-4 (Vorschau-Beträge):** Preview-Rechnung mit Betrag 0 / carryover unter Mindestbetrag → Vorschau-Label trotzdem korrekt, kein „fällig"-Eindruck.
- **EC-5 (Multi-Tenant):** Banner/Label sind pro EEG-Sicht (tenant-scoped) am jeweiligen Preview-Zustand orientiert.

## Non-Goals

- Prod-Instanz-Einrichtung → **PROJ-116** (separat).
- AVV/DPIA, AGB-Klausel „free→paid", Load-Test → Owner-Tracks ohne Code.
- FreeFinance-/Mollie-Live, `globalLiveMode`-Cutover, Edition-Auto-Ableitung (PROJ-110) / Kosten-Transparenz (PROJ-111) → erst paid-Launch.
- **Keine** echte Abrechnung scharf schalten.

## Dependencies

- Requires: **PROJ-104** (Billing-Datenmodell, `globalLiveMode`/`IsLive`, Rechnungen-Tab).
- Requires: **PROJ-109** (Freigabe-Gate / Cron-Staging — Quelle der Preview-Rechnungen).
- Verwandt: **PROJ-116** (Prod-Instanz läuft mit `globalLiveMode=false` → Banner dort aktiv).

## Grilling-Entscheidungen (2026-06-20)

Code-Erkundung + Owner-Entscheidungen — fließen direkt in die ACs oben:

1. **Banner-Zustandsquelle:** Tenant-Endpoint `GET /api/admin/eeg/{rc}/invoices` um berechnetes `billingLive` (= `IsLive(globalLiveMode, eeg)`) erweitern; Banner zeigt wenn `false`. Per-Rechnung-`status='preview'` allein versagt im Leer-Fall (neue Free-EEG ohne Rechnungen). → AC-3. Backend-Detail: der Tenant-Invoices-Handler braucht `globalLiveMode` (aus `cfg`) injiziert, um `IsLive` zu berechnen.
2. **Trial-Marker out of scope:** wird dem EEG-Vorstand nicht angezeigt → kein Tenant-Retext (AC-5/6 alt gestrichen).
3. **No-Charge-Guard:** zwei Handler-Eingänge (`SetBillingLive` true-Zweig + `CreateCreditNote`), `globalLiveMode=false` → **409**. Nicht `IsLive`-basiert, sondern **global** `globalLiveMode` (das ist der Free-Phasen-Schalter; per-EEG `billing_live` ist hier irrelevant, weil `SetBillingLive` es ja gerade setzt). → AC-6.
4. **Label-Trennung:** Banner ← effektives `billingLive`-Flag; „Vorschau"-Badge ← `status==='preview'` pro Rechnung. Zwei getrennte Signale, kein Mislabeling. → AC-2/AC-3.

### Owner-approbierte Wortlaute
- **Banner:** „Die Nutzung ist derzeit **kostenlos**. Eine kostenpflichtige Nutzung ist geplant und wird **rechtzeitig vorab angekündigt**. Die unten gezeigten Beträge sind eine **unverbindliche Vorschau** und werden aktuell nicht in Rechnung gestellt."
- **Badge** `preview`: „Preview" → **„Vorschau"**
- **Guard-Meldung (409):** „In der kostenlosen Phase nicht verfügbar — der globale Live-Modus ist deaktiviert."

### Betroffene Stellen (Full-Chain für /architecture)
- Backend: Tenant-Invoices-Handler (`billingLive` in Response + `globalLiveMode` injizieren), `SetBillingLive` + `CreateCreditNote` (409-Guard), `AdminBillingHandler` braucht `globalLiveMode`.
- Frontend: `src/lib/api/billing.ts` (`EEGInvoiceItem`-Response um `billingLive` bzw. Wrapper), `src/components/billing/eeg-own-invoices.tsx` (Banner + Badge-Relabel „Preview"→„Vorschau").
- Keine DB-Migration.

## Tech Design (Solution Architect)

### A) Datenfluss & Komponenten

Es gibt zwei voneinander unabhängige Wirkungen — eine **Anzeige-Wirkung** (Banner + Badge) und eine **Schutz-Wirkung** (Guard).

**Anzeige (was der EEG-Vorstand sieht):**

```
EEG-Settings → Tab „Rechnungen"
  └── Rechnungen-Liste (bestehende Komponente)
       ├── [NEU] Free-Banner   ── sichtbar, wenn die EEG nicht live abgerechnet wird
       └── Rechnungs-Tabelle
            └── Status-Spalte: Badge „Vorschau" (statt „Preview") für Vorschau-Zeilen
```

Der Banner braucht eine verlässliche Antwort auf „ist diese EEG gerade in der kostenlosen Phase?". Diese Antwort berechnet das Backend zentral und schickt sie als **ein einziges Ja/Nein-Feld** mit der Rechnungsliste mit. Die Regel dahinter ist die bestehende „echt-live"-Regel: live ist eine EEG nur, wenn der globale Live-Schalter **und** der EEG-Schalter beide an sind. Solange einer von beiden aus ist (in der kostenlosen Phase ist der globale aus), gilt: kostenlos → Banner an.

Wichtig: Das Vorschau-Badge in der Tabelle hängt **nicht** an diesem globalen Feld, sondern am Status der **einzelnen Rechnung**. So bekommt eine später einmal echt versandte Rechnung nie fälschlich ein „Vorschau"-Etikett, selbst wenn die EEG zwischenzeitlich wieder kostenlos wäre.

**Schutz (was der Plattform-Betreiber nicht versehentlich auslösen kann):**

```
Owner-Aktion „EEG live schalten"   ─┐
Owner-Aktion „Gutschrift erstellen" ─┤── Türsteher: globaler Live-Schalter aus?
                                     │     → ja: Aktion abgelehnt (409), keine Mail
                                     │     → nein: Aktion läuft wie bisher
```

Das sind die **einzigen zwei** Stellen, an denen das System eine Mail an eine EEG schicken könnte, die nach „Bezahlen/Mandat" aussieht. Beide sind manuelle Owner-Aktionen. Ein vorgeschalteter Türsteher lehnt sie ab, solange die kostenlose Phase global gilt — mit einer klaren Meldung. Das „EEG wieder still-schalten" bleibt jederzeit erlaubt.

Der automatische Quartals-/Tageslauf ist **schon heute** sicher: in der kostenlosen Phase legt er nur Vorschau-Rechnungen an und verschickt keine EEG-Mail. Das wird in der QA mit einem echten Laufdurchlauf nachgewiesen — kein Code nötig.

### B) Daten-Modell (Klartext)

- **Keine neue Tabelle, keine neue Spalte, keine Migration.**
- Neu ist nur **ein berechnetes Antwort-Feld** „kostenlos ja/nein" in der Rechnungsliste der EEG — es wird bei jedem Abruf frisch aus den zwei bestehenden Schaltern errechnet und nirgends gespeichert.
- Dazu zwei **Text-Änderungen** in der Oberfläche (Banner-Text, „Vorschau" statt „Preview") und eine **Fehlermeldung** für den Türsteher.

### C) Frontend-Auswirkung

- Bestehende Komponente „eigene Rechnungen" bekommt **oben einen Hinweis-Banner** (auffällig, aber nicht alarmierend — Info-Stil), der nur erscheint, wenn das neue „kostenlos"-Feld gesetzt ist. Er erscheint auch dann, wenn noch gar keine Rechnung existiert.
- Das Status-Etikett für Vorschau-Rechnungen wird auf **deutsch „Vorschau"** umbenannt.
- Verwendet bestehende Bausteine (Hinweis-/Badge-Komponenten); **keine neuen Pakete**, keine Formularfelder (also keine Placeholder-Thematik).

### D) Tech-Entscheidungen (WHY)

- **Berechnetes „kostenlos"-Feld statt globalen Schalter roh ausliefern:** Der globale Live-Schalter ist eine Betreiber-interne Konfiguration. Würde man ihn roh ans Frontend geben, müsste die Oberfläche die „echt-live"-Regel selbst nachbauen — fehleranfällig und doppelt gepflegt. Ein vom Backend fertig berechnetes Ja/Nein ist die einzige Wahrheit und deckt automatisch den Mischzustand (EC-1) und den späteren Umstieg auf kostenpflichtig (EC-3) ab, ohne erneutes Ausrollen.
- **Türsteher global statt pro EEG:** Die kostenlose Phase ist eine **globale** Aussage des Betreibers. Eine einzelne EEG „live" zu schalten, während global alles kostenlos ist, ist widersprüchlich und hätte als einzige reale Folge eine Mandats-Mail an die EEG — genau das, was wir verhindern wollen. Deshalb greift der Türsteher am globalen Schalter.
- **Badge getrennt vom Banner-Feld:** Banner = „ist die EEG gerade kostenlos?" (globaler Zustand). Badge = „ist diese eine Rechnung eine Vorschau?" (Zustand der einzelnen Rechnung). Zwei verschiedene Fragen → zwei verschiedene Quellen, damit nichts falsch etikettiert wird.

### E) Betroffene Stellen (Full-Chain)

- **Backend:** Tenant-Rechnungsliste (berechnetes „kostenlos"-Feld in der Antwort; der globale Live-Schalter muss dem zuständigen Handler bekannt gemacht werden) · Owner-Aktionen „EEG live schalten" und „Gutschrift erstellen" (Türsteher mit 409) · der Owner-Billing-Handler muss den globalen Live-Schalter kennen.
- **Frontend:** API-Typ der Rechnungsliste (neues Feld) · Komponente „eigene Rechnungen" (Banner + „Vorschau"-Umbenennung).
- **Keine** DB-Migration, **keine** neuen Pakete, **keine** neuen Endpunkte.

### F) Dependencies

Keine neuen Pakete. Genutzt werden die bestehende Rechnungs-Komponente, die bestehende „echt-live"-Regel und die bestehende Owner-Billing-Handler-Schicht.

### G) Empfohlene Umsetzungs-Reihenfolge

**`/backend` zuerst** (berechnetes „kostenlos"-Feld in der Rechnungsliste + 409-Türsteher an den zwei Owner-Aktionen), dann **`/frontend`** (Banner + „Vorschau"-Badge). So steht der Vertrag (Antwort-Feld + Guard) fest, bevor die Oberfläche ihn spiegelt. Danach `/qa` (inkl. No-Charge-Cron-Nachweis) → `/deploy`.

## Implementation Notes (Backend, 2026-06-20)

Backend umgesetzt (kein DB-Change). Tests grün (`go build/vet/test ./...`).

- **billingLive in Tenant-Response (AC-3):** [admin_eeg_invoices.go](../internal/http/admin_eeg_invoices.go) — `AdminEEGInvoicesHandler` um `entrypointRepo` + `globalLiveMode` erweitert; `GET /api/admin/eeg/{rc}/invoices` liefert jetzt `{ invoices, billingLive }`, wobei `billingLive = billing.IsLive(globalLiveMode, eeg)`. Bei eeg-Lookup-Fehler defensiv `false` (= Free-Phase, Banner zeigt). `globalLiveMode` wird **nicht** roh ausgeliefert. Verdrahtung in [main.go](../cmd/server/main.go) (`cfg.Billing.GlobalLiveMode` + `entrypointRepo`).
- **409-Guard (AC-6):** [admin_billing.go](../internal/http/admin_billing.go) — `SetBillingLive` (vor dem eeg-Load: `req.Live && !GlobalLiveMode` → 409, Off-Toggle bleibt) und `CreateCreditNote` (direkt nach Superuser-Check → 409). Beide: `code=free_phase_active`, Meldung „In der kostenlosen Phase nicht verfügbar — der globale Live-Modus ist deaktiviert." Kein Mail-/Vendor-Zugriff vor dem Reject.
- **Tests:** [admin_billing_free_phase_guard_test.go](../internal/http/admin_billing_free_phase_guard_test.go) — beide Guards → 409 + Code (DB-los, nil-Repos). `IsLive`-Wahrheitstabelle schon in `live_mode_test.go`. Bestehende Billing-/HTTP-Tests grün.
- **Frontend-Vertrag für /frontend:** `listEEGOwnInvoices` liefert künftig zusätzlich `billingLive: boolean`. Banner zeigt wenn `!billingLive`; Badge `preview` „Preview"→„Vorschau" gekeyt auf `status`.

## Nächster Schritt

Backend abgeschlossen (2026-06-20). → `/frontend` (Banner + Badge-Relabel, liest `billingLive`) → `/qa` (mit No-Charge-Cron-Test) → `/deploy`.
