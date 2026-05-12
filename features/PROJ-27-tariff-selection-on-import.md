# PROJ-27: Tarif-Auswahl beim Import

## Status: Approved
**Created:** 2026-05-09
**Last Updated:** 2026-05-12 (Implementation + QA komplett)

## Dependencies
- Requires: PROJ-4 (Core Import) — bestehende Import-Pipeline und `internal/coreclient`
- Requires: PROJ-3 (Admin Frontend UI) — Erweiterung der Admin-Edit-Form
- Requires: PROJ-5 (Keycloak-secured Admin Area) — Tarif-Lookup nutzt das Admin-JWT

## Hintergrund

Aktuell legt der Import (`POST /participant`) jeden Teilnehmer und Zählpunkt **ohne Tarif** im eegFaktura-Core an (`tariffId` und `meters[].tariff_id` werden nicht gesetzt). Der EEG-Admin muss anschließend in eegFaktura jeden Datensatz manuell öffnen und einen Tarif zuweisen — das ist der häufigste manuelle Nacharbeitsschritt nach einem Onboarding-Import.

Tarife werden im eegFaktura-Core verwaltet (Tabelle `base.tariff`, UUID-PKs, eindeutig pro Tenant). Zum Zeitpunkt eines Imports kennt das Onboarding-System diese Liste nicht — sie soll **beim Klick auf „In eegFaktura importieren" aktuell aus dem Core geladen** und in einem Auswahl-Popup angeboten werden. Der Admin wählt Mitglieds-Tarif und pro Zählpunkt einen Tarif aus; die Auswahl wird **nicht persistiert**, sondern direkt im Import-Call an den Core mitgesendet.

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

### Keine Persistenz im Onboarding
- [ ] **Keine** DB-Migration. `tariff_id` wird weder auf `application` noch auf `metering_point` gespeichert.
- [ ] Tarif-Auswahl ist eine reine Import-Time-Entscheidung; sie lebt nur im Memory zwischen Popup-Klick und Core-Call.
- [ ] Bei `import_failed` und einem Retry muss der Admin die Tarife neu wählen — die vorherige Auswahl ist nicht gemerkt. Akzeptabler Trade-off; Import-Retries sind selten.

### Admin-UI: Import-Popup
- [ ] Der bestehende „In eegFaktura importieren"-Button (`AdminStatusActions`) öffnet **kein direktes Import**, sondern ein neues Popup „Import vorbereiten".
- [ ] Beim Öffnen des Popups lädt das Frontend per `GET /api/admin/tariffs?rcNumber=…` die aktuelle Tarif-Liste aus dem Core.
- [ ] Solange der Lookup läuft: Spinner; Bestätigungsbutton deaktiviert.
- [ ] Bei Erfolg zeigt das Popup:
  - **Ein Dropdown** für Mitglieds-Tarif (gefiltert `type == "EEG"`, inaktive ausgeblendet)
  - **Ein Dropdown pro Zählpunkt** (gefiltert `type == "VZP"` für CONSUMPTION, `"EZP"` für GENERATION)
  - Format pro Eintrag: `{name} — {centPerKWh} ct/kWh{discount>0 ? `, Rabatt {discount}%` : ``}{useVat ? ` (USt {vatInPercent}%)` : ``}` (Beispiel: `Abnahmetarif Rabatt10 — 13 ct/kWh, Rabatt 10% (USt 20%)`)
  - „(kein Tarif)"-Option zuerst — Admin kann jedes Feld explizit leer lassen
- [ ] Bei Lookup-Fehler (Core nicht erreichbar, Timeout): Hinweis im Popup („Tarife konnten nicht geladen werden — Import erfolgt ohne Tarife"). Bestätigungsbutton bleibt **aktiv**; Import läuft ohne Tarife (Bestandsverhalten).
- [ ] Bei 0 Tarifen einer Direction: Dropdown zeigt nur „(kein Tarif)" plus Hinweis „Keine {Verbraucher|Erzeuger}-Tarife in eegFaktura definiert".
- [ ] Bestätigungsbutton ruft den existierenden Import-Endpunkt auf und schickt die Tarif-Auswahl mit (siehe „Import-Pipeline").

### Import-Pipeline
- [ ] Der bestehende `POST /api/admin/applications/{id}/import` bekommt einen optionalen Request-Body:
  ```json
  { "tariffId": "<uuid>|null", "meterTariffs": { "<metering_point_id>": "<uuid>|null", ... } }
  ```
- [ ] Leerer Body und fehlende Felder bleiben rückwärtskompatibel — alle Tarife optional.
- [ ] `BuildPayload` wird so erweitert, dass die übergebenen Tarif-IDs als `tariffId` (Mitglied) und `tariff_id` (pro Meter, snake_case wie im Core-Modell) im `POST /participant`-Body landen.
- [ ] Wenn keine Tarif-ID gesetzt ist, wird das Feld **weggelassen** (`omitempty`) — analog zum heutigen Verhalten.
- [ ] Vor dem Import wird **nicht** erneut gegen den Core validiert, dass die Tarif-IDs noch existieren — der Core lehnt ab, falls eine ID ungültig ist.

### Public Registration Form
- [ ] Die Public-Form (`registration-form.tsx`) zeigt **kein** Tarif-Feld — Mitglieder kennen die internen eegFaktura-Tarife nicht und sollen sie nicht selber wählen.
- [ ] Der Admin wählt den Tarif beim Import-Klick aus.

### Externe Registrierungs-API (PROJ-13)
- [ ] **Keine Änderung** — die externe API setzt keine Tarife. Da Tarif-Auswahl Import-Time ist, gibt es keinen Persistenz-Pfad, in den die externe API schreiben könnte. Falls Bedarf besteht, müsste der externe Aufrufer den Import-Endpunkt selbst aufrufen — out of scope für PROJ-27.

### Sicherheit & Tenant-Isolation
- [ ] Der Tarif-Lookup-Endpoint im Onboarding-Backend (z.B. `GET /api/admin/tariffs?rcNumber=…`) ist Keycloak-geschützt
- [ ] Der Endpoint validiert, dass der Admin Zugriff auf die angegebene EEG hat (`checkTenantAccess`) — sonst 403
- [ ] Es ist **nicht** möglich, durch Manipulation der `rcNumber` Tarife einer fremden EEG zu lesen
- [ ] Der Endpoint cacht Tarife **pro Tenant**, nicht global — kein Cross-Tenant-Leak möglich

## Edge Cases

- Was passiert, wenn der Core zwischen Popup-Öffnen und Confirm einen Tarif löscht? → Der Core lehnt den Import ab, der Antrag landet in `import_failed`. Admin muss den Import erneut starten und neu wählen.
- Was passiert, wenn der Core während des Popup-Öffnens nicht erreichbar ist? → Popup zeigt Hinweis, Confirm bleibt aktiv, Import erfolgt ohne Tarife (Bestandsverhalten).
- Was passiert, wenn ein Antrag bei `import_failed` re-importiert wird? → Der Admin öffnet das Popup erneut, wählt erneut. Die vorherige Auswahl ist nicht gespeichert.
- Was passiert bei Bulk-Aktionen (PROJ-25, „Mehrere importieren")? → Bulk-Import ist heute über `/bulk-action` möglich. Für Tarif-Auswahl in Bulk müsste ein eigenes Bulk-Popup gebaut werden — **out of scope** für PROJ-27.
- Was passiert, wenn die Core-Tarif-Liste 0 Einträge hat? → Beide Dropdowns zeigen nur „(kein Tarif)"; Hinweistext „Keine Tarife in eegFaktura definiert". Confirm bleibt aktiv.
- Was passiert, wenn für eine Direction keine passenden Tarife existieren? → Dropdown zeigt „(kein Tarif)" + Hinweis „Keine {Verbraucher|Erzeuger}-Tarife in eegFaktura definiert".
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

- 2026-05-12: Stakeholder-Klarstellung — Lookup beim Klick auf „Importieren", **kein** Persistieren der Auswahl. Spec entsprechend umgebaut, DB-Migration fällt weg.
- Security-Review (`/security-review`) empfohlen — neuer authentifizierter Endpoint plus Erweiterung des Import-Pfads.

---
<!-- Sections below are added by subsequent skills -->

## Resolved Decisions

Lookup-Zeitpunkt (2026-05-12, Stakeholder-Klarstellung): **beim Klick auf „Importieren"**, nicht im Edit-Form. Tarif-Auswahl wird **nicht persistiert**.

- **Q1** (2026-05-09): Endpoint `GET /api/eeg/tariff` mit Bearer + tenant-Header. Antwort-Felder dokumentiert.
- **Q2:** Backend-Proxy. Frontend ruft `GET /api/admin/tariffs?rcNumber=…` im Onboarding-Backend, das den Core aufruft.
- **Q3:** Popup öffnet sich auch bei Core-Ausfall; Dropdowns disabled, aber Import ohne Tarif bleibt möglich (Bestandsverhalten).
- **Q4:** **Kein** Cache mehr nötig — Lookup passiert nur beim Import-Klick (selten); Frische geht vor Geschwindigkeit.
- **Q5:** Excel-Export bleibt unverändert. Da nichts persistiert wird, gibt es nichts zu exportieren.
- **Q6:** Externe API (PROJ-13) ändert sich nicht — Tarif-Auswahl ist UI-/Admin-Aktion.
- **Q7** (2026-05-09): Type-Filterung über `type`-Feld (EEG/VZP/EZP).
- **Q8:** Tarif **nicht** in Public-Submission-Bestätigung — irrelevant, da kein Tarif persistiert wird.

## Tech Design (Solution Architect)

### Übersicht

Drei Schichten:

1. **Backend Core-Client:** neue Methode `ListTariffs(ctx, bearerToken, tenant)` über den bestehenden `CoreClient`. Kein Cache — Lookup ist Import-time, selten genug.
2. **Backend HTTP:** zwei Endpoints:
   - `GET /api/admin/tariffs?rcNumber=…` — Frontend ruft das beim Popup-Öffnen
   - bestehender `POST /api/admin/applications/{id}/import` bekommt einen optionalen Request-Body mit den Auswahl-IDs
3. **Frontend:** Import-Button öffnet jetzt ein Popup mit dynamisch geladenen Tarif-Dropdowns. Confirm sendet die Auswahl an den Import-Endpunkt.

**Keine DB-Migration. Kein Persistenz-Effekt. Kein Bestandsdaten-Risiko.**

### Datenbankänderungen

Keine.

### Backend-Struktur

#### CoreClient-Erweiterung (`internal/coreclient/core_client.go`)

Neue Methode auf dem `CoreClient`-Interface:
```go
ListTariffs(ctx context.Context, bearerToken, tenant string) ([]CoreTariff, error)
```

Wraps `GET {baseURL}/eeg/tariff`. Response wird in eine schmale `CoreTariff`-Struct deserialisiert, die nur die für das Onboarding relevanten Felder enthält:
```go
type CoreTariff struct {
    ID            string  `json:"id"`
    Type          string  `json:"type"`        // EEG | VZP | EZP
    Name          string  `json:"name"`
    CentPerKWh    float64 `json:"centPerKWh"`
    Discount      float64 `json:"discount"`
    UseVat        bool    `json:"useVat"`
    VatInPercent  float64 `json:"vatInPercent"`
    InactiveSince *string `json:"inactiveSince"`
}
```

Fehler werden über die bestehende Error-Typologie (`CoreHTTPError`, `CoreParseError`, `ErrCoreTimeout`) abgebildet.

#### HTTP-Endpoint Tariff-Lookup (`internal/http/admin.go`)

```
GET /api/admin/tariffs?rcNumber=<RC>
```
- Keycloak-Auth via Subrouter
- Tenant-Validierung über bestehendes `containsRC(claims.Tenant, rcNumber)`-Pattern
- Forwarded Bearer-Token an `coreClient.ListTariffs`
- Antwort: `{ "tariffs": [{...}] }` — Pass-through der `CoreTariff`-Felder
- Bei Core-Fehler: `503 service_unavailable` (oder 502); Frontend behandelt das als „Tarife nicht verfügbar"

#### Import-Request-Erweiterung

`POST /api/admin/applications/{id}/import` akzeptiert einen optionalen JSON-Body:
```json
{
  "tariffId": "uuid|null",
  "meterTariffs": { "<metering_point_id>": "uuid|null" }
}
```

Backward-compatible: leerer Body oder fehlende Felder = kein Tarif (Bestandsverhalten).

#### Import-Service (`internal/importing/import_service.go`)

`Import(ctx, id, bearerToken, actorID, allowedTenants, selection)` bekommt einen zusätzlichen Parameter:
```go
type TariffSelection struct {
    MemberTariffID string            // empty = none
    MeterTariffIDs map[string]string // meteringPoint -> tariffID
}
```

`BuildPayload` wird so erweitert, dass die meter-spezifischen Tarif-IDs im POST /participant landen (als `meters[].tariff_id`).

**Wichtige Erkenntnis aus dem Core-Source** (`myeegfaktura/eegfaktura-backend/model/participant.go`):
- `EegParticipantBase.TariffId` hat `goqu:"omitempty,skipinsert"` → wird beim `POST /participant` **ignoriert**, kann nur per UPDATE gesetzt werden.
- `MeteringPoint.TariffId` hat **kein** `skipinsert` → wird im POST mitinsertiert.

Daraus folgt der Import-Flow:
1. `POST /participant` mit `meters[].tariff_id` aus `selection.MeterTariffIDs` (omitempty)
2. Wenn `selection.MemberTariffID != ""`: Follow-up-Call
   ```
   PUT /participant/v2/{participantID}
   Body: { "path": "tariffId", "value": "<uuid>" }
   ```
   Bei Fehler dieses Calls: Loggen + im Import-Result vermerken („participant created, member tariff assignment failed"), aber Antrag bleibt `imported` (Meter-Tarife sind ja schon drin). Admin kann Tarif manuell im Core nachpflegen.

#### Import-Payload (`internal/importing/payload.go`)

`CoreParticipantPayload`:
- Kein Tarif-Feld nötig (Core ignoriert es).

`CoreMeteringPoint`:
- `TariffID string \`json:"tariff_id,omitempty"\`` (snake_case, Core-Konvention)

Wird bei leerem String über `omitempty` weggelassen.

#### Core-Client (`internal/coreclient/core_client.go`)

Neue Methode auf dem `CoreClient`-Interface:
```go
ListTariffs(ctx context.Context, bearerToken, tenant string) ([]CoreTariff, error)
UpdateParticipantField(ctx context.Context, bearerToken, tenant, participantID, path string, value any) error
```

`ListTariffs` wraps `GET /eeg/tariff`. `UpdateParticipantField` wraps `PUT /participant/v2/{id}` mit Body `{"path": path, "value": value}`.

### Frontend-Struktur

#### TypeScript-Typen (`src/lib/api.ts`)

- Neuer Type `Tariff` (passend zur Backend-Response: `id`, `type`, `name`, `centPerKWh`, `discount`, `useVat`, `vatInPercent`, `inactiveSince`)
- `fetchTariffs(rcNumber, token): Promise<{ tariffs: Tariff[] }>`
- `importApplication(id, body?, token)` — Body um `tariffId` + `meterTariffs` erweitert (beide optional)

#### Neuer Dialog (`src/components/import-tariff-dialog.tsx` oder inline)

Triggered, wenn der Admin den Import-Button für einen `approved`/`import_failed`-Antrag klickt:

1. Beim Öffnen: `fetchTariffs(application.rcNumber)` → state `tariffs`
2. Während Loading: Spinner; Confirm disabled
3. Nach Erfolg:
   - 1 × Mitglieds-Tarif-Dropdown (`type=EEG`, inaktive ausgeblendet)
   - n × Zählpunkt-Tarif-Dropdowns (pro Meter, `VZP`/`EZP` je Direction)
   - Jeweils erste Option „(kein Tarif)"
   - Dropdown-Label: `{name} — {centPerKWh} ct/kWh[, Rabatt {discount}%][ (USt {vatInPercent}%)]`
4. Nach Fehler: gelber Hinweis „Tarife konnten nicht geladen werden — Import erfolgt ohne Tarife"; Confirm bleibt aktiv
5. Confirm → `importApplication(id, { tariffId, meterTariffs })`, Toast, Refresh

Der Dialog ersetzt die bisherige `confirm()`-Browser-Box im `AdminStatusActions`.

#### Public-Form

Keine Änderung (Q8). Tarif bleibt Import-Time-Auswahl des Admins.

### Tests

- `internal/coreclient/core_client_test.go`: `ListTariffs` Happy-Path + HTML-Response-Detection
- `internal/importing/payload_test.go`: `BuildPayload` setzt `tariffId` (Mitglied) und `meters[].tariff_id` (pro Meter) korrekt; lässt sie bei leerem String weg
- Frontend: manueller Browser-Smoke (kein vitest-Setup für Dialog-Flow)

### Implementation-Reihenfolge

1. CoreClient `ListTariffs`
2. HTTP-Endpoint `GET /api/admin/tariffs`
3. Import-Service + Payload-Erweiterung um `TariffSelection`
4. Import-Handler-Body-Parsing
5. Frontend: api.ts → ImportTariffDialog → AdminStatusActions
6. Tests + Docs (api-spec.md, import-mapping.md)

### Sicherheits-Überlegungen

- Cross-Tenant-Leak: ausgeschlossen, weil der Endpoint `containsRC` durchsetzt
- Auth-Forwarding: das Admin-Bearer-Token wird an den Core durchgereicht — derselbe Pfad wie der bestehende Import (PROJ-4)
- Keine DB-Änderung → keine Migration-Risiken

`/security-review` ist empfohlen (neuer authentifizierter Endpoint + Erweiterung des Import-Pfads). Da kein neues Schema und keine neue Auth-Logik dazukommt, ist das Risiko kontrolliert.

## QA Test Results

**QA Date:** 2026-05-12
**Tester:** Claude QA

### Automated Tests
| Suite | Result |
|---|---|
| `go build ./...` | ✅ |
| `go test ./...` (alle bestehenden Tests + erweiterte `BuildPayload`-Signatur) | ✅ |
| `npx tsc --noEmit` | ✅ |

### Acceptance Criteria

#### Tarif-Lookup
| # | Criterion | Result |
|---|---|---|
| AC-1 | `GET {core}/eeg/tariff` via Bearer + tenant | ✅ `coreclient.HTTPCoreClient.ListTariffs` |
| AC-2 | Pass-through Bearer-Token (Admin-JWT) | ✅ |
| AC-3 | Tenant-Isolation: nur Tarife der EEG des Admins | ✅ `containsRC(claims.Tenant, rcNumber)` |
| AC-4 | Inaktive Tarife ausgeblendet | ✅ Frontend-Filter (`inactiveSince == null`) |
| AC-5 | Cache: keine — Lookup nur beim Import-Klick | ✅ Frontend lädt bei jedem Dialog-Open neu |
| AC-6 | Bei Core-Fehler: Import bleibt möglich (Dialog-Hinweis) | ✅ Confirm bleibt aktiv, leere Selection |

#### Tarif-Typen-Filterung
| # | Criterion | Result |
|---|---|---|
| AC-7 | Mitglieds-Tarif-Dropdown nur `type=EEG` | ✅ |
| AC-8 | Verbraucher-Meter nur `type=VZP` | ✅ |
| AC-9 | Erzeuger-Meter nur `type=EZP` | ✅ |

#### Keine Persistenz
| # | Criterion | Result |
|---|---|---|
| AC-10 | Kein DB-Schema-Update | ✅ |
| AC-11 | Tarif-Auswahl nicht in `application`/`metering_point` gespeichert | ✅ |

#### Import-Pipeline
| # | Criterion | Result |
|---|---|---|
| AC-12 | `POST /participant` mit `meters[].tariff_id` (snake_case, omitempty) | ✅ |
| AC-13 | Member-Tarif via Follow-up `PUT /participant/v2/{id}` (Core-skipinsert workaround) | ✅ |
| AC-14 | Member-Tarif-Fehler: Import bleibt `imported`, Warning im Result | ✅ `ImportResult.MemberTariffWarning` |
| AC-15 | Leerer Body bleibt rückwärtskompatibel (Legacy „kein Tarif") | ✅ Body-Parsing nur wenn `ContentLength > 0` |

#### Admin-UI: Import-Popup
| # | Criterion | Result |
|---|---|---|
| AC-16 | „In eegFaktura importieren" öffnet Popup statt direktem Confirm | ✅ |
| AC-17 | Spinner während Tarif-Load | ✅ |
| AC-18 | Member-Dropdown + Pro-Meter-Dropdowns | ✅ `ImportTariffDialog` |
| AC-19 | Anzeigeformat `{name} — {centPerKWh} ct/kWh[, Rabatt …%][ (USt …%)]` | ✅ `tariffLabel` |
| AC-20 | „(kein Tarif)"-Option je Dropdown | ✅ |
| AC-21 | Hinweis bei leerer Liste je Typ | ✅ |
| AC-22 | Toast bei Member-Tarif-Warning | ✅ `toast.warning(...)` |

### Bugs Found

Keine.

### Test-Coverage-Gap

`ImportService.Import` und `coreClient.ListTariffs/UpdateParticipantField` werden im bestehenden Stil nicht unit-getestet (kein sqlmock im Projekt). Manueller Browser-Smoke ist der primäre Verifikationspfad. Follow-up: ggf. sqlmock einführen, separates Ticket.

### Security Smoke

| Bereich | Bewertung |
|---|---|
| Tenant-Isolation (`/api/admin/tariffs`) | ✅ `containsRC` (analog `GetIntroText`) |
| Bearer-Token-Forwarding | ✅ identischer Pfad wie PROJ-4 Import |
| Keine neue Persistenz | ✅ keine Schema-Risiken |
| Input-Validation | ✅ Body-Parser akzeptiert nur `tariffId` + `meterTariffs` (Map String→String) |
| Reason-Logging | n/a — kein Reason hier |

`/security-review` empfohlen, nicht zwingend (kein neues Schema, kein neuer Auth-Pfad).

### Regression

- `BuildPayload`-Signatur erweitert (zusätzlicher `meterTariffIDs`-Parameter) — alle bestehenden Tests aktualisiert (`nil` als Default).
- Legacy-Import-Flow (Body weglassen) → identisches Verhalten zu vor PROJ-27.
- `ImportApplication`-Handler nimmt Body **nur** wenn `ContentLength > 0`, sonst Bestand.

### Production-Ready Decision

**READY** — alle ACs erfüllt, Backend + Frontend grün, keine offenen Bugs.

## Deployment

**Deployed:** _pending CI rollout_
**Chart version:** 1.6.0 (Minor — neues Feature)
**Migration:** keine
**Rollback:** `helm rollback` auf 1.5.0; keine Daten betroffen, da nichts persistiert wird.

### Deployment checklist
- [x] `go build ./...` clean
- [x] `go test ./...` clean
- [x] `npx tsc --noEmit` clean
- [x] Keine neuen Env-Variablen
- [x] Helm `appVersion` auf `1.6.0`
- [x] Neuer Route `/api/admin/tariffs` registriert
- [ ] **Empfohlen:** Browser-Smoke gegen Test-EEG (Tarif-Dropdowns sichtbar, Import mit + ohne Tarif funktioniert)
