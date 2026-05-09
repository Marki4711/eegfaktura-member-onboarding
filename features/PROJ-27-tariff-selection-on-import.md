# PROJ-27: Tarif-Auswahl beim Import

## Status: Planned
**Created:** 2026-05-09
**Last Updated:** 2026-05-09 (Q1 + Q7 resolved nach Verifikation gegen deployten Core)

## Dependencies
- Requires: PROJ-4 (Core Import) — bestehende Import-Pipeline und `internal/coreclient`
- Requires: PROJ-3 (Admin Frontend UI) — Erweiterung der Admin-Edit-Form
- Requires: PROJ-5 (Keycloak-secured Admin Area) — Tarif-Lookup nutzt das Admin-JWT

## Hintergrund

Aktuell legt der Import (`POST /participant`) jeden Teilnehmer und Zählpunkt **ohne Tarif** im eegFaktura-Core an (`tariffId` und `meters[].tariff_id` werden nicht gesetzt). Der EEG-Admin muss anschließend in eegFaktura jeden Datensatz manuell öffnen und einen Tarif zuweisen — das ist der häufigste manuelle Nacharbeitsschritt nach einem Onboarding-Import.

Tarife werden im eegFaktura-Core verwaltet (Tabelle `base.tariff`, UUID-PKs, eindeutig pro Tenant). Zum Zeitpunkt eines Imports kennt das Onboarding-System diese Liste nicht — sie soll dynamisch aus dem Core geladen und im Admin-UI als Auswahl angeboten werden, sodass der Admin **vor** dem Import den richtigen Tarif zuweisen kann.

## User Stories

- Als **EEG-Admin** möchte ich beim Bearbeiten eines Antrags vor dem Import einen Tarif für das Mitglied auswählen können, sodass der Teilnehmer mit korrektem Tarif im eegFaktura-Core landet.
- Als **EEG-Admin** möchte ich pro Zählpunkt einen eigenen Tarif auswählen können (z.B. Verbraucher- vs. Erzeuger-Tarif), sodass jede Zähleranlage mit dem richtigen Tarif importiert wird.
- Als **EEG-Admin** möchte ich, dass die Auswahlliste der Tarife immer den **aktuellen Stand** aus eegFaktura widerspiegelt, sodass neue Tarife sofort verfügbar sind, ohne dass das Onboarding-System konfiguriert oder neu deployed werden muss.
- Als **EEG-Admin** möchte ich, dass ich auch keinen Tarif auswählen kann (Feld leer lassen), sodass der bisherige Workflow (Tarif manuell in eegFaktura nachpflegen) weiterhin funktioniert — Tarif-Auswahl ist optional.
- Als **EEG-Admin** möchte ich eine klare Fehlermeldung sehen, falls die Tarif-Liste nicht aus dem Core geladen werden kann, sodass ich entscheiden kann, ob ich ohne Tarif importiere oder den Import verschiebe.

## Acceptance Criteria

### Tarif-Lookup aus eegFaktura
- [ ] Backend ruft die verfügbaren Tarife über `GET {coreBaseUrl}/eeg/tariff` aus dem eegFaktura-Core ab
- [ ] Der Lookup nutzt **dasselbe Bearer-Token und denselben `tenant`-Header** wie der bestehende Import (PROJ-4) — keine zusätzlichen Credentials
- [ ] Tarife werden **pro EEG (Tenant)** geladen — Tarife einer EEG dürfen niemals einer anderen EEG angeboten werden
- [ ] Tarife mit `inactiveSince != null` werden im Onboarding-UI **nicht** angeboten (nur in der Anzeige bestehender, bereits zugewiesener IDs als "Tarif inaktiv (Name)" sichtbar)
- [ ] Cache-Strategie: kurzlebiger In-Memory-Cache pro Tenant (z.B. 60 s) ist erlaubt, längere Cachezeiten sind nicht zulässig (siehe Open Question Q4)
- [ ] Bei Core-Fehler (Timeout, 5xx, nicht erreichbar) liefert das Onboarding-Backend einen klaren Fehler und das UI zeigt den Zustand "Tarife konnten nicht geladen werden"
- [ ] Bei Core-Fehler wird der Import **nicht blockiert** — der Admin kann ohne Tarif importieren (siehe Open Question Q3)

### Tarif-Typen und Zuordnung

Das Core-`tariff`-Objekt hat ein Feld `type` mit den Werten **`EEG`**, **`VZP`** oder **`EZP`**. Daraus ergibt sich die Filterung im UI:

| Auswahlfeld im Onboarding | Erlaubte `type`-Werte | Hintergrund |
|---|---|---|
| Mitglieds-Tarif (Application-Level) | `EEG` | Mitgliedsbeitrag/Participant-Fee |
| Zählpunkt-Tarif (`direction = CONSUMPTION`) | `VZP` | Verbraucher-Zählpunkt |
| Zählpunkt-Tarif (`direction = GENERATION`) | `EZP` | Einspeise-Zählpunkt |

- [ ] Das Mitglieds-Tarif-Dropdown zeigt nur Tarife mit `type == "EEG"`
- [ ] Das Zählpunkt-Tarif-Dropdown zeigt pro Zähler nur Tarife mit dem zur Direction passenden `type` (`VZP` für CONSUMPTION, `EZP` für GENERATION)
- [ ] Wechselt der Admin die Direction eines Zählpunkts in der Edit-Form, wird der zugewiesene Tarif **zurückgesetzt** (sonst wäre die Zuordnung type-inkonsistent) — UI-Hinweis zeigt das an

### Persistenz im Onboarding
- [ ] `member_onboarding.application` bekommt eine neue Spalte `tariff_id UUID NULL` (Member-Tarif)
- [ ] `member_onboarding.metering_point` bekommt eine neue Spalte `tariff_id UUID NULL` (Zählpunkt-Tarif)
- [ ] Beide Spalten sind `NULL`-fähig — Tarif-Auswahl ist optional
- [ ] Es gibt **keine** Foreign-Key-Constraint auf eine Tarif-Tabelle (Tarife sind im eegFaktura-Core, nicht im Onboarding-Schema)
- [ ] Tarif-IDs werden **nur** als UUID gespeichert; Anzeigename des Tarifs wird **nicht** persistiert (immer dynamisch aus Core nachgeladen, um Drifts zu vermeiden)
- [ ] Migration ist additiv (`ADD COLUMN`) — bestehende Anträge bleiben mit `tariff_id = NULL` lauffähig

### Admin-UI (Edit-Form)
- [ ] In der bestehenden Admin-Edit-Form (`admin-edit-form.tsx`) gibt es einen neuen Abschnitt "Tarif" mit einem Dropdown für den Mitglieds-Tarif
- [ ] In der Zählpunkt-Tabelle gibt es eine neue Spalte "Tarif" mit einem Dropdown pro Zeile
- [ ] Beide Dropdowns zeigen die aus dem Core geladenen Tarife im Format `{name} — {centPerKWh} ct/kWh{discount > 0 ? `, Rabatt {discount}%` : ``}{useVat ? ` (USt {vatInPercent}%)` : ``}` (Beispiel: `Abnahmetarif Rabatt10 — 13 ct/kWh, Rabatt 10% (USt 20%)`)
- [ ] Beide Dropdowns haben eine "(kein Tarif)"-Option, mit der der Admin die Auswahl explizit leer lassen kann
- [ ] Die Tarif-Liste wird beim **Öffnen der Edit-Form** geladen, nicht beim Import-Klick — der Admin sieht die Tarife sofort
- [ ] Gibt es einen Ladefehler, wird ein Hinweis angezeigt ("Tarife nicht verfügbar — Antrag wird ohne Tarif importiert"); der Save/Import-Button bleibt aktiv
- [ ] Wenn ein Antrag bereits eine Tarif-ID gespeichert hat, die in der Core-Liste **nicht mehr existiert** (z.B. Tarif wurde im Core gelöscht), zeigt das Dropdown den Wert als "Unbekannter Tarif (ID …)" an und der Admin muss eine neue Auswahl treffen oder explizit "(kein Tarif)" wählen, bevor er importieren kann

### Import-Pipeline
- [ ] Beim Import wird `application.tariff_id` als `tariffId` im Participant-Payload gesendet
- [ ] Beim Import wird pro Zählpunkt `metering_point.tariff_id` als `tariff_id` im Meter-Payload gesendet
- [ ] Wenn keine Tarif-ID gesetzt ist, wird das Feld **weggelassen** (nicht als `null` oder `""` gesendet) — analog zum heutigen Verhalten
- [ ] Vor dem Import wird **nicht** erneut gegen den Core validiert, dass die Tarif-IDs noch existieren — der Core lehnt ab, falls eine ID ungültig ist; die Antwort des Cores wird wie bisher als Import-Fehler gespeichert
- [ ] Excel-Export (PROJ-17) übernimmt die Tarif-IDs, sodass auch der Excel-Pfad konsistent bleibt — **Open Question Q5**

### Public Registration Form
- [ ] Die Public-Form (`registration-form.tsx`) zeigt **kein** Tarif-Feld — Mitglieder kennen die internen eegFaktura-Tarife nicht und sollen sie nicht selber wählen
- [ ] Der Admin pflegt den Tarif beim Review/Approval

### Externe Registrierungs-API (PROJ-13)
- [ ] Die externe API darf optional eine Tarif-ID pro Mitglied und pro Zählpunkt mitliefern
- [ ] Wird eine Tarif-ID mitgeliefert, validiert das Backend nicht gegen den Core (Konsistenz mit Onboarding-Edit-Pfad) — Validierung passiert erst beim Import
- [ ] **Open Question Q6:** soll die externe API überhaupt Tarif-IDs erlauben?

### Sicherheit & Tenant-Isolation
- [ ] Der Tarif-Lookup-Endpoint im Onboarding-Backend (z.B. `GET /api/admin/tariffs?rcNumber=…`) ist Keycloak-geschützt
- [ ] Der Endpoint validiert, dass der Admin Zugriff auf die angegebene EEG hat (`checkTenantAccess`) — sonst 403
- [ ] Es ist **nicht** möglich, durch Manipulation der `rcNumber` Tarife einer fremden EEG zu lesen
- [ ] Der Endpoint cacht Tarife **pro Tenant**, nicht global — kein Cross-Tenant-Leak möglich

## Edge Cases

- Was passiert, wenn der Core während der Edit-Session einen Tarif löscht, den der Admin gerade ausgewählt hat? → Beim nächsten Save/Import: Core lehnt ab, Antrag bleibt in `approved` mit Fehlermeldung; Admin sieht "Unbekannter Tarif" beim erneuten Öffnen der Form
- Was passiert, wenn der Admin einen Tarif auswählt, dann den Core ausfällt, dann erneut speichert? → Save speichert die UUID weiterhin (kein Lookup beim Save nötig); erst der Import schlägt fehl, falls der Core den Tarif nicht akzeptiert
- Was passiert, wenn ein Antrag importiert wird, dann re-importiert (bei `import_failed`)? → Tarif-IDs werden unverändert mitgesendet, Admin kann sie vor dem Re-Import bei Bedarf anpassen
- Was passiert bei Bulk-Aktionen (PROJ-25, „Mehrere genehmigen + importieren")? → Falls Bulk-Approval einen Bulk-Import triggert, müssen die einzelnen Anträge bereits gespeicherte Tarif-IDs nutzen; eine Bulk-Tarif-Auswahl ist **out of scope** für PROJ-27 (Folge-Feature)
- Was passiert, wenn die Core-Tarif-Liste 0 Einträge hat (keine Tarife konfiguriert)? → Dropdown zeigt nur "(kein Tarif)"; Hinweis-Text "Keine Tarife in eegFaktura definiert"
- Was passiert, wenn für eine Direction keine passenden Tarife existieren (z.B. nur `VZP`-Tarife, aber ein Zählpunkt ist `GENERATION`)? → Zählpunkt-Dropdown zeigt nur "(kein Tarif)" mit Hinweis "Keine Erzeuger-Tarife (EZP) in eegFaktura definiert"
- Was passiert, wenn ein zuvor gewählter Tarif zwischenzeitlich `inactiveSince` gesetzt bekommen hat? → Anzeige als "Tarif inaktiv: {name}"; der Admin muss eine neue Auswahl treffen oder explizit "(kein Tarif)" wählen, sonst blockiert der Import
- Was passiert, wenn der Core neue Tarif-Felder einführt (z.B. preisinfo, gültigAb/Bis), die wir nicht kennen? → Onboarding zeigt nur Name und ID; zusätzliche Felder werden ignoriert (Forward-Compatibility)
- Was passiert, wenn zwei Admins parallel die Tarif-Liste laden und einer einen Tarif während der Anzeige des anderen anpasst? → Beide sehen ihre jeweiligen Stände; spätestens beim Save wird der dann aktuelle Wert gespeichert (last-write-wins, wie heute)

## Technical Requirements

- **Performance:** Tarif-Lookup darf das Öffnen der Edit-Form nicht spürbar verzögern (Lookup parallel zum Form-Render erlaubt; Skeleton/Spinner für Dropdown akzeptabel)
- **Sicherheit:** Tenant-Isolation strikt; keine Tarif-IDs einer EEG dürfen einer anderen EEG sichtbar werden
- **Konsistenz:** Tarif-Auswahl bleibt optional — kein Bruch für EEGs, die Tarife weiterhin manuell in eegFaktura zuweisen
- **Rückwärtskompatibilität:** Bestehende Anträge ohne Tarif-IDs bleiben importierbar; keine Daten-Migration nötig
- **Beobachtbarkeit:** Tarif-Lookup-Fehler werden mit `slog.Warn` geloggt (Tenant, Status-Code, abgefragte URL — kein Token)

## Open Questions / Options zu evaluieren

### Q1: Welcher Core-Endpoint liefert die Tarife? — **RESOLVED 2026-05-09**

**Endpoint:** `GET {coreBaseUrl}/eeg/tariff` (unter `/api/eeg/tariff` über den Ingress)
**Auth:** Bearer-Token + `tenant`-Header, identisch zu `POST /participant`
**Response:** JSON-Array von Tarif-Objekten. Felder, die wir im Onboarding nutzen:

| Feld | Typ | Verwendung |
|---|---|---|
| `id` | UUID | wird in `application.tariff_id` / `metering_point.tariff_id` gespeichert |
| `version` | int | nicht persistiert; informativ |
| `type` | `"EEG" \| "VZP" \| "EZP"` | Filterung pro Auswahlfeld (siehe AC oben) |
| `name` | string | Dropdown-Label |
| `centPerKWh` | number | Dropdown-Label |
| `discount` | number | Dropdown-Label, falls > 0 |
| `useVat` / `vatInPercent` | bool / number | Dropdown-Label, falls `useVat == true` |
| `inactiveSince` | timestamp \| null | Filtert inaktive Tarife in Auswahllisten aus |

Alle anderen Felder (`participantFee`, `baseFee`, `meteringPointFee`, `freeKWh`, `billingPeriod`, `vatSupplementaryText`, `useMeteringPointFee`, `accountNetAmount`, `accountGrossAmount`, `businessNr`, `meteringPointVat`, `createdAt`) werden ignoriert — sie gehören zur Tarif-Detailpflege im eegFaktura-Core und sind für die Auswahl irrelevant.

Beispiel-Response: siehe `docs/import-mapping.md` §10 (wird im Architecture-Schritt ergänzt).

### Q2: Lookup-Strategie — Backend-Proxy oder Frontend direkt?

- (a) Frontend ruft direkt den Core auf (mit Bearer-Token aus NextAuth)
- (b) Frontend ruft Onboarding-Backend, das den Core proxiet

**Empfehlung:** (b) — konsistent mit `CLAUDE.md` ("Frontends talk only to the Member Onboarding backend; only the backend accesses the database; only the backend calls the eegFaktura core internally"). Außerdem ist Caching, Logging und Tenant-Validierung im Backend einfacher.

### Q3: Was tun, wenn der Core beim Edit-Form-Öffnen nicht erreichbar ist?

- (a) Edit-Form gar nicht öffnen, harten Fehler anzeigen
- (b) Edit-Form öffnen, Tarif-Dropdown deaktivieren, andere Felder editierbar
- (c) Edit-Form öffnen, Dropdown als Freitext-Eingabe (UUID) anbieten

**Empfehlung:** (b). Der Admin soll andere Antragsdaten weiterhin editieren können; Tarif-Auswahl ist eine Komfortfunktion, kein Pflichtschritt. Ein Hinweis ("Tarife aktuell nicht verfügbar") macht das Verhalten klar.

### Q4: Cache-Lebensdauer für Tarif-Liste

- (a) Kein Cache — jeder Edit-Form-Öffnen-Klick ruft den Core
- (b) Kurzlebiger Cache (30–60 s pro Tenant)
- (c) Längerlebiger Cache mit Invalidierung bei Import-Fehler oder manuellem Refresh-Button
- (d) Cache pro Browser-Session (per-User-Storage)

**Empfehlung:** (b). Ausreichend, um wiederholtes Öffnen kurz hintereinander effizient zu machen, ohne dass neue Tarife im Core lange unsichtbar bleiben.

### Q5: Excel-Export (PROJ-17) — Tarife mitexportieren?

Der Excel-Export wird genutzt, wenn der Direkt-Import (PROJ-4) nicht verfügbar ist (z.B. EEG ohne Core-Konnektivität).

- (a) Excel-Export erweitern um Spalten "Mitglieds-Tarif (ID)" und "Zählpunkt-Tarif (ID)"
- (b) Tarif-IDs werden im Export weggelassen, EEG pflegt sie manuell nach
- (c) Tarif-Spalten enthalten Tarif-Namen statt UUIDs

**Empfehlung:** (a) — UUIDs sind eindeutig und vom Excel-Import in eegFaktura akzeptiert; Namen sind nicht eindeutig (Tarif-Umbenennungen) und benötigen Lookup. UUIDs sind unleserlich, aber Excel-Import hat das ohnehin nicht als UI-Schicht.

### Q6: Externe Registrierungs-API (PROJ-13) — Tarif-IDs erlauben?

- (a) Ja, optional, ohne Validierung gegen Core
- (b) Ja, optional, mit Validierung gegen Core (kostet zusätzlichen Lookup pro Registrierung)
- (c) Nein, externe API setzt nie Tarife — Admin pflegt sie immer manuell

**Empfehlung:** (a). Die externe API ist für Integrationen mit EEG-eigenen Tools gedacht (siehe Memory-Note „External API scope review needed"); diese Tools können die Core-Tarife kennen und mitliefern. Validierung beim Eingang würde den Admin-Pfad inkonsistent machen (auch dort wird nicht validiert).

### Q7: Tarif-Filterung nach Direction? — **RESOLVED 2026-05-09**

Der Core liefert pro Tarif ein Feld `type` mit den Werten `EEG`, `VZP` oder `EZP`. Damit ist die Filterung deterministisch:
- Mitglieds-Tarif-Dropdown → nur `type == "EEG"`
- Zählpunkt-Tarif-Dropdown bei CONSUMPTION → nur `type == "VZP"`
- Zählpunkt-Tarif-Dropdown bei GENERATION → nur `type == "EZP"`

Implementiert in den ACs oben.

### Q8: Anzeige im Public-Frontend nach Submission?

- (a) Mitglied sieht in der Submission-Bestätigung den zugewiesenen Tarif
- (b) Tarif bleibt rein admin-intern bis zum Import

**Empfehlung:** (b). Der Tarif kann zwischen Submission und Approval noch geändert werden; eine vorzeitige Anzeige schafft falsche Erwartungen. Die offizielle Information bekommt das Mitglied über die Approval-Mail (PROJ-21), die ohnehin nach dem Import-Schritt kommt.

## Notes

- Spec sollte vor `/architecture` durch `/grill-me` laufen, insbesondere wegen Q1 (Core-API-Verträglichkeit) und Q2/Q3 (UI-Verhalten bei Core-Ausfall).
- Security-Review (`/security-review`) ist erforderlich, da: (a) neue authentifizierte Endpoint-Klasse, (b) Tenant-Isolation für Tarif-Listen, (c) DB-Schema-Änderung an `application` und `metering_point`.
- Migration nummeriert sich an die bestehenden `db/migrations/0000XX_*.sql` an.

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
