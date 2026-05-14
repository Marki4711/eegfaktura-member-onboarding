# PROJ-32: EEG-Stammdaten aus eegFaktura-Core beziehen (inkl. Logo)

## Status: In Review
**Created:** 2026-05-14
**Last Updated:** 2026-05-14 (Phase 1 implementiert + URL-Architektur final: single Hostname-only env var. Phase 2 / Logo-Embed steht als Folge-Spec aus.)

> **Spec-Anmerkung:** Während der Implementierung wurde die ursprüngliche
> Architektur (Live-Resolver mit In-Memory-Cache + Service-Account) zu einem
> einfacheren Modell vereinfacht. Statt eines Caches überträgt der **Sync
> die Core-Daten direkt in `registration_entrypoint`** — die DB ist der
> single-source-of-truth-Speicher für Render-Pfade. Auth nutzt das **Admin-
> JWT** des klickenden Admins (Token-Forwarding wie bei `POST /participant`)
> statt eines Service-Accounts. Damit sind die Original-Stages A–G und die
> Open Questions Q2/Q4/Q6/Q7/Q8 hinfällig; die effektive Umsetzung ist in
> den drei Stages A–C der commit history dokumentiert. Detailtext oben ist
> aus Doku-Gründen erhalten geblieben, aber nicht die gelieferte Implementation.

> **URL-Architektur (final 2026-05-14):** `CORE_BASE_URL` ist nur der
> **Hostname** (z.B. `https://eegfaktura.at`). Die Pfad-Präfixe sind im
> coreclient pro Aufruf hardcoded, weil der produktive Reverse-Proxy
> mehrere Services unter demselben Host multiplext:
> - `{base}/api/participant`, `{base}/api/eeg/tariff`, `{base}/api/query` — eegFaktura-backend
> - `{base}/cash/api/billingConfigs/...` — eegfaktura-billing (Phase 2 Logo)
>
> Ein separates `CORE_GRAPHQL_URL` env existiert NICHT mehr. Deployments
> müssen `CORE_BASE_URL` von `…/cash/api` auf den reinen Hostname umstellen.

> **Auth-Modell — User-Context-Bearer-Forwarding (locked 2026-05-14):**
> Alle Core-Aufrufe (Stammdaten-Sync, Import, Tarif-Lookup) leiten das
> **Bearer-JWT des angemeldeten Admins** unverändert an den Core weiter,
> zusammen mit dem `tenant`-Header (RC-Nummer). **Kein Service-Account,
> kein client_credentials-Flow, kein zusätzlicher Keycloak-Client.**
>
> Die ursprüngliche Spec (Stages A–G) sah einen Service-Account mit
> tenant-Claim-Mapper vor; das ist bewusst verworfen worden:
> - Audit-Trail im Core attribuiert die Änderung an den realen Admin
>   statt an einen anonymen Onboarding-Service-Account.
> - Keine zusätzliche Infrastruktur (Client + Secret-Rotation + Mapper).
> - Der Admin ist im Settings-UI ohnehin schon authentifiziert; sein
>   Tenant-Claim deckt genau die RCs ab, für die er syncen darf.
>
> **Nicht ohne explizite Entscheidung umstellen.** Mögliche Gründe für
> einen späteren Umbau: Hintergrund-Jobs ohne Admin-Klick, oder
> Rate-Limit-Isolation zwischen Onboarding- und Human-Traffic.

## Dependencies
- Requires: PROJ-4 (Core Import) — wieder­verwendet die bestehende `coreclient`-Infrastruktur (Bearer-Auth, tenant-Header, Timeouts).
- Requires: PROJ-12 / PROJ-21 (SEPA-PDF, Beitrittsbestätigung-PDF) — die EEG-Stammdaten werden dort gerendert; ein Logo-Embed kommt hier hinzu.
- Berührt: PROJ-19 (Manuelle Aktivierung / EEG-Einstellungen) — die Admin-UI für EEG-Stammdaten wird umgebaut.

## Hintergrund

Aktuell pflegt der EEG-Admin EEG-Stammdaten (Name, Adresse, Creditor-ID, Kontakt-E-Mail) **manuell** im Onboarding über die Einstellungen-Seite. Diese Felder duplizieren Daten, die im eegFaktura-Core bereits gepflegt werden, und führen regelmäßig zu Drift:

- EEG-Adressen ändern sich im Core, Onboarding-PDFs zeigen weiter die alte Adresse.
- Creditor-IDs werden vom EEG-Admin abgetippt — Tippfehler sind nicht selten.
- Logos können im Core hinterlegt werden, das Onboarding kennt sie nicht und kann sie deshalb auch nicht in den PDFs verwenden.

Recherche-Ergebnis 2026-05-14:
- Der Core exponiert bereits `GET /eeg` (Bearer + tenant-Header, derselbe Auth-Mode wie `POST /participant`). Im OSS-Modell `Eeg.go` enthält die Response: Name, EegAddress (street/zip/city), AccountInfo (creditorId, iban, bankName, …), Contact (email, phone), Optionals (website).
- Die produktiv-deployte Version hat zusätzlich ein **Logo-Feld** in der EEG-Verwaltung (vom User per Screenshot bestätigt; das OSS-Modell ist veraltet). Die genaue Datendarstellung (URL? base64? Filestore-ID?) muss zur Implementierungs­zeit gegen die echte API verifiziert werden.
- Das Onboarding nutzt heute aus dem Core nur `POST /participant`, `GET /eeg/tariff`, `PUT /participant/v2/{id}`, `GET /participant`. Ein Aufruf von `GET /eeg` existiert nicht.

## User Stories

- Als **EEG-Admin** möchte ich EEG-Stammdaten **nicht doppelt** in eegFaktura-Core und Onboarding pflegen — Änderungen im Core sollen sich automatisch auf Onboarding-PDFs und Mails auswirken.
- Als **EEG-Admin** möchte ich, dass mein im Core hinterlegtes **Logo** auf der Beitrittsbestätigung und im SEPA-Mandat-PDF erscheint.
- Als **EEG-Admin** möchte ich auf der EEG-Einstellungen-Seite weiterhin sehen können, welche Stammdaten verwendet werden — aber bei der Quelle "Core" sollen die Felder **schreibgeschützt** sein.
- Als **vfeeg-Betreiber** möchte ich, dass der Onboarding-Service auch dann funktioniert, wenn der Core kurzzeitig nicht erreichbar ist — ein Cache mit klarer TTL und Fallback-Verhalten ist Pflicht.
- Als **vfeeg-Betreiber** möchte ich keine 1:1-Live-Abfrage pro PDF — der Core soll nicht mit Onboarding-Last beaufschlagt werden.

## Acceptance Criteria

### Core-Client-Erweiterung

- [ ] Neuer Service-Token-Fetcher `coreclient.ServiceAccountTokenSource` (caches access token, refreshes before expiry via Keycloak `client_credentials`).
- [ ] Neue Methode `(c *HTTPCoreClient) FetchEEGMasterData(ctx, rc) (*CoreEEG, error)` mit GraphQL-POST gegen `CORE_GRAPHQL_URL`. Query:
  ```graphql
  query { eeg { id name description rcNumber
                address { street streetNumber zip city }
                contact { phone email }
                accountInfo { iban owner bankName creditorId bic sepa }
                optionals { website } } }
  ```
- [ ] Neuer DTO-Typ `CoreEEG` in `internal/coreclient/` mit den Feldern oben (alle nullable).
- [ ] Neue Methode `(c *BillingClient) FetchEEGLogo(ctx, rc) (bytes []byte, mimeType string, err error)` — zweistufig:
  1. `GET <BILLING_BASE_URL>/billingConfigs/tenant/{rc}` → BillingConfigDTO mit `id` und `headerImageFileDataId`
  2. Falls `headerImageFileDataId != nil`: `GET <BILLING_BASE_URL>/billingConfigs/{id}/logoImage` → Bytes
- [ ] Fehlerbehandlung: 404 → `shared.ErrNotFound`, GraphQL-`errors` Array → wrap mit klarem Message, alle anderen Fehler → wrap.

### Cache-Schicht

- [ ] Onboarding hält die EEG-Stammdaten in einem **In-Memory-Cache** mit TTL.
- [ ] Default-TTL: 15 Minuten (siehe Q2 für Begründung). Konfigurierbar via env `CORE_EEG_CACHE_TTL` in Sekunden.
- [ ] Cache-Key: RC-Nummer (case-insensitive).
- [ ] Cache wird Lazy gefüllt: erster Aufruf nach Programmstart oder Ablauf trifft den Core.
- [ ] Bei Core-Fehler greift ein **Stale-While-Error**-Mechanismus: der letzte erfolgreich gecachte Wert wird als Fallback zurückgegeben (mit klarem Log-Hinweis). Erst wenn auch der Stale-Wert leer ist, gibt es einen harten Fehler — der dann von der lokalen `registration_entrypoint`-Tabelle abgefangen wird (siehe Q3).
- [ ] Prometheus-Counter `eeg_master_data_fetch_total{result}` mit Labels `cache_hit | cache_miss_success | cache_miss_failed | stale_fallback`.

### Quelle-Auflösung (Resolver)

- [ ] Neues Service-Konzept `EEGMasterDataResolver`, eine zentrale Stelle, die aus RC-Nummer die "effektiven" Stammdaten ermittelt.
- [ ] Reihenfolge:
  1. Core abfragen (via Cache)
  2. Bei Misserfolg: lokale `registration_entrypoint`-Werte als Fallback
  3. Bei beidem leer: leere Strings rendern (PDF zeigt keine Adresse, keine Creditor-ID; SEPA-Mandat-Sektion wird ausgespart)
- [ ] Alle bestehenden Render-Pfade (SEPA-PDF, Beitrittsbestätigung-PDF, Mail-Templates EEG-Block) gehen ab jetzt durch den Resolver — kein direkter Zugriff mehr auf `registration_entrypoint.eeg_name` etc. aus Render-Code.
- [ ] Snapshot-Verhalten: bestehende Anwendungen, die bereits abgeschickt sind, **erben nicht** rückwirkend die Core-Daten in archivierten PDFs — sondern bei Neu-Rendern (Resend, Re-Download) gilt der dann aktuelle Core-Stand. Begründung: Snapshotting würde eine neue DB-Spalte pro Stammdaten-Feld erfordern; das ist mehr Wert als das Risiko. (siehe Q4)

### Logo-Embedding in PDFs

- [ ] Neue Datei `internal/pdf/logo.go` mit `EmbedEEGLogo(pdf *gofpdf.Fpdf, logoData []byte, position …)`. Wraps `fpdf.RegisterImageReader` + `fpdf.Image`.
- [ ] PDF-Layout: Logo oben rechts in der Kopfzeile, max. Höhe ~24 mm (Q5). Fällt das Logo weg (kein Logo im Core), bleibt der Header-Bereich frei.
- [ ] Unterstützte Formate: PNG, JPEG (über `RegisterImageReader` auto-detect). SVG wird **nicht** unterstützt (fpdf kann nicht); bei SVG-Logo → Fallback zu textbasierter Kopfzeile.
- [ ] Bei Logo-Loading-Fehler: PDF wird ohne Logo gerendert; Warnung im Log.

### Admin-UI

- [ ] EEG-Einstellungen-Seite zeigt die Stammdaten-Felder weiterhin, aber **read-only** mit Hinweis: "Wird automatisch aus eegFaktura bezogen. Änderungen direkt im eegFaktura-Core vornehmen."
- [ ] Ein "Aktualisieren"-Button forciert einen Cache-Refresh (Cache invalidieren + neu laden). Sinnvoll wenn der Admin gerade im Core eine Änderung gemacht hat und sie sofort sehen will, ohne 15 Min auf den TTL zu warten.
- [ ] Logo wird klein gerendert (z.B. 80×80 px) zur Sicht-Kontrolle.
- [ ] Wenn der Core den EEG-Endpoint nicht ausliefert (`CORE_BASE_URL` leer oder 404): Felder fallen auf die alte manuelle Pflege zurück, mit Hinweis "eegFaktura-Core nicht verfügbar — manuelle Pflege aktiv". Felder sind dann wieder editierbar (siehe Q6).

### Migration / Rückwärts-Kompatibilität

- [ ] Bestehende Werte in `registration_entrypoint.eeg_name/eeg_street/...` bleiben **unangetastet** — sie dienen ab jetzt als Fallback.
- [ ] Keine Datenmigration. Keine Schema-Migration nötig.
- [ ] Beim ersten Cache-Refill nach Deploy: Core-Antwort überschreibt die UI-Anzeige; manuelle Werte bleiben als Fallback liegen.
- [ ] Wenn ein EEG-Admin später die manuelle Pflege wieder will, kann er den Toggle (Q7) zurückdrehen.

### Konfiguration

- [ ] Neue Env-Var `CORE_EEG_CACHE_TTL` (default `900` = 15 min).
- [ ] Bestehende `CORE_BASE_URL` reicht — kein zusätzlicher Endpoint nötig.
- [ ] Kein dedizierter Read-only-Toggle nötig solange Q7=(a). Wenn Q7=(b) gewählt: neuer Eintrag in `registration_entrypoint` (`eeg_master_data_source` ENUM `core|manual` mit Default `core`).

### Dokumentation

- [ ] `docs/api-spec.md`: neuer Core-Call `GET /eeg` dokumentieren (Backend-extern, nicht Onboarding-extern).
- [ ] `docs/architecture.md`: neuer Abschnitt "EEG-Stammdaten" — Resolver-Diagramm und Cache-TTL.
- [ ] `docs/user-guide/06-admin-settings.md`: Stammdaten-Sektion umschreiben (read-only, "Aktualisieren"-Button erklären).

### Observability

- `eeg_master_data_fetch_total{result}` siehe oben.
- `slog.Info` bei jedem Core-Fetch: `rc_number`, `latency_ms`, `result`.
- Bei Stale-Fallback: `slog.Warn` mit Stale-Age-in-Sekunden.

## Edge Cases

- **Core-Endpoint liefert weniger Felder als erwartet** (z.B. kein Logo): Cache-Eintrag wird trotzdem gespeichert; fehlende Felder als leer behandelt. Resolver fällt für fehlende Felder auf die manuelle DB-Tabelle zurück, wenn vorhanden.
- **Core-Endpoint kennt diese RC-Nummer nicht** (404): Resolver fällt komplett auf die DB-Tabelle zurück; Admin-UI zeigt "Stammdaten in eegFaktura nicht gefunden — manuelle Pflege wird verwendet".
- **Core-Endpoint gibt ein anderes Schema zurück als erwartet** (Feldumbenennung in einer Core-Version): JSON-Decode-Fehler. Resolver fällt auf DB-Tabelle zurück + lauter Warn-Log.
- **Logo ist riesig** (z.B. 5 MB PNG): PDF-Embedding-Zeit steigt linear; bei >2 MB würde das Mail-Attachment auffällig groß. **Spec-Entscheidung:** Cache-Layer verkleinert Logos automatisch auf max. 600 px Längsseite via `image/jpeg` re-encode. Bei SVG: ablehnen.
- **Logo ist eine externe URL** statt embedded bytes: Cache holt die URL einmal ab und cached die Bytes (nicht die URL) — sonst hängt unsere PDF-Generierung an einer externen IPv4/TLS-Konnektivität.
- **Race zwischen Admin-Aktualisierung und parallel laufendem PDF-Render**: Renderer holt die Daten aus dem Resolver; wenn die im Cache veraltet sind (15 min), kann es zu Mischzuständen kommen. Spec-Entscheidung: hinnehmbar, der Effekt ist max. 15 min Verzögerung.
- **Multi-Replica-Deployment** (S6 aus dem Memo, derzeit parked): jeder Replica hat seinen eigenen In-Memory-Cache. TTL-Differenz zwischen Replicas → ein Replica kann 15 min lang noch alte Daten ausliefern. Akzeptabel.

## Technical Requirements

- **Performance:** Core-Fetch maximal alle 15 min pro RC-Nummer pro Replica. PDF-Generierung darf nie auf einen Core-Call warten, der >2 s dauert; bei Timeout → Stale-Fallback.
- **Sicherheit:** der Core-Call nutzt denselben Bearer-Token wie der `POST /participant`-Call. **Wichtig:** dieser Token wird heute aus dem Admin-JWT extrahiert. Beim PDF-Render (z.B. Bestätigungs-Mail beim Submit) gibt es **kein Admin-JWT**. → siehe Q8 (kritische Frage zur Auth).
- **Privacy:** Core-Stammdaten enthalten kontaktinfos (E-Mail, Telefon des EEG). Werden sie in Logs gedruckt? Antwort: nein, nur die rc_number wird geloggt.
- **Rückwärts-Kompatibilität:** Tenants ohne Core-Anbindung (`CORE_BASE_URL` leer) sollen weiterhin funktionieren — Resolver liefert dann immer aus der DB-Tabelle.

## Resolved Decisions (2026-05-14)

Source-Recherche im `myeegfaktura`-Mono-Repo hat alle Open Questions außer Q2/Q5/Q6/Q7 (Defaults) eindeutig beantwortet:

- **Stammdaten:** GraphQL-Endpoint im `eegfaktura-backend`-Service. Route: `POST /query` (server.go:179). Auth: User-JWT-OIDC via `GQLProtect()`. Query liefert das komplette `Eeg`-Modell mit allen für PDFs/Mails relevanten Feldern.
- **Logo + BillingConfig:** Eigener Service `eegfaktura-billing` (Java/Spring Boot, separates Repo `c:\opt\repos\myeegfaktura\eegfaktura-billing`). Endpoints:
  - `GET /api/billingConfigs/tenant/{tenantId}` → BillingConfigDTO mit `id`, `headerImageFileDataId`, `footerImageFileDataId`, +Invoice-Templating-Felder (für uns irrelevant). 1:1-Mapping RC ↔ BillingConfig.
  - `GET /api/billingConfigs/{id}/logoImage` → liefert PNG/JPG-Bytes direkt (Content-Type aus DB-MimeType-Spalte). Bytes liegen als `@Lob byte[]` in eegfaktura-billing's eigener Postgres-DB — **kein** zusätzlicher Filestore-Roundtrip.
  - Auth: `TenantContext.validateTenant()` prüft `tenant`-Claim im JWT gegen Request-PathVariable. Service-Account-Token mit `tenant`-Claim akzeptiert (via Custom-Mapper im Keycloak-Client).
- **Reverse-Proxy-Pfad:** Im Live-System (`eegfaktura.at`) wird beiden Services ein `/cash`-Prefix vorgeschaltet. Aus Onboarding-Sicht:
  - `CORE_GRAPHQL_URL = https://eegfaktura.at/cash/api/query`
  - `BILLING_BASE_URL = https://eegfaktura.at/cash/api`

## Open Questions

### Q1: Wie repräsentiert die deployte Core-API das EEG-Logo? — geklärt

**Live-API-Inspektion 2026-05-14 (Bearer-authentifiziert gegen `eegfaktura.at`):**

Die Core-API hat **zwei** relevante Pfade:

**a) Stammdaten: GraphQL** (Endpoint heißt `query`, Pfad vermutlich `/cash/api/query`).

Request:
```graphql
query { eeg { id name description ... address { ... } contact { ... } accountInfo { ... } } }
```

Response (Auszug aus dem Live-System):
```json
{ "data": { "eeg": {
    "id": "TE100200",
    "rcNumber": "TE100200",
    "name": "EEG-TEST",
    "address": { "street": "Sonnenplatz", "streetNumber": "4", "zip": "9720", "city": "Strahlhausen" },
    "contact": { "phone": "00436641234567", "email": "test-eeg@gmx.at" },
    "accountInfo": { "iban": "AT613456789012345678", "owner": "EEG-TEST", "bankName": "Testbank",
                     "creditorId": null, "bic": null, "sepa": false }
}}}
```

Wichtig:
- `eeg.id === rcNumber` → kein zusätzlicher RC-Lookup nötig
- Stammdaten kommen vom GraphQL-Endpoint, nicht aus einer REST-Route
- `creditorId` ist hier `null` (Testdaten) — das passiert real und der Resolver muss damit umgehen

**b) Logo: REST über `BillingConfig`** (`/cash/api/billingConfigs/...`).

Vorgang ist zweistufig:

1. `GET /cash/api/billingConfigs?tenantId={rc}` (oder Filterung über tenant-Header) → liefert die `BillingConfig` für diese EEG, enthält u.a.:
   ```json
   { "id": "269c2abf-bc33-40f6-827e-386987c42b16",
     "tenantId": "TE100200",
     "headerImageFileDataId": "f291f0de-abb4-4a0d-b3ac-4006a40c5c35",
     ...weitere Felder, irrelevant für uns... }
   ```

2. `GET /cash/api/billingConfigs/{id}/logoImage` → liefert die PNG-Bytes direkt:
   ```
   200 OK
   Content-Type: image/png
   Content-Length: 132285
   ```

Implementations-Plan:
- `coreclient.FetchEEGMasterData(ctx, bearer, tenant)` → GraphQL-POST
- `coreclient.FetchEEGLogo(ctx, bearer, tenant)` → zweistufig:
  - listet Billing-Configs, nimmt die mit passender `tenantId`, extrahiert `id`
  - holt `/logoImage`-Bytes
- Beide Methoden cachen lokal (Stammdaten als Struct, Logo als Bytes mit MIME-Typ)
- Resolver-Pfad: Core-Wert pro Feld; bei null/leer → lokaler DB-Fallback; Logo → falls Fehlschlag, PDF rendert ohne Logo

**Offener Restpunkt:** wie genau lautet die GraphQL-URL (`/cash/api/query` oder `/query` oder `/graphql`?). Beim ersten Implementations-Commit prüfen wir das per `curl` gegen das Live-System.

### Q2: Cache-TTL — 15 Minuten oder anders?

- Argument für kurz (5 min): Änderungen propagieren schnell.
- Argument für lang (1 h): minimale Last auf Core, weniger Network-IO.

**Empfehlung:** 15 Minuten als Mittelwert. Plus expliziter "Aktualisieren"-Button im Admin-UI, der den Cache für eine RC-Nummer sofort invalidiert. So muss der Admin nicht warten, wenn er gerade im Core editiert hat.

### Q3: Was tun, wenn Core **und** lokale DB beide leer sind?

- (a) PDF mit leeren EEG-Feldern rendern (heutiges Verhalten)
- (b) Submit ablehnen
- (c) Admin-Warnung im Onboarding, aber Submit erlauben

**Empfehlung:** (a). Das ist das heutige Verhalten und wird vom Member nicht bemerkt. Stört nur den Doku-Beleg; das ist ein Admin-Problem nicht ein Member-Problem.

### Q4: Snapshot bei Submit, oder immer Live-Stand?

- (a) Live (= Resolver beim Render): einfach, aber alte PDFs zeigen neue Adresse wenn der EEG umgezogen ist.
- (b) Snapshot (= eigene Spalten in `application` für EEG-Daten): immutable Beleg, aber dicke Migration und doppelter Storage.
- (c) Hybrid: aktuelle Mails/PDFs leben/live, einmal generierte und versendete PDFs liegen als Blob im Filestore (heute nicht der Fall — PDFs werden nach Versand verworfen).

**Empfehlung:** (a). Die einzige Sektion, wo Snapshotting wirklich gebraucht würde, wäre das SEPA-Mandat (juristisch ist der Stand zum Zeitpunkt der Unterschrift relevant). Das ist eine Folge-Spec.

### Q5: Logo-Position und -Größe im PDF

- Oben rechts, max. 24 × 24 mm, proportional skaliert?
- Oder zentral oben, breiter?

**Empfehlung:** Oben rechts, 24 mm Höhe, proportional. Lässt den linken Bereich für den "Beitrittsbestätigung" / "SEPA-Lastschriftmandat"-Titel frei. Wenn der EEG-Admin das Format ändern will, ist das eine Folge-Spec.

### Q6: Im Core-down-Fall: Stammdaten **read-only** halten oder editierbar?

- (a) Read-only: konsistent mit dem "Core ist Source of Truth"-Versprechen, aber Admin kann nichts tun.
- (b) Editierbar: Admin kann manuell pflegen, sobald Core wieder da ist überschreibt der Cache.

**Empfehlung:** (b). Wenn der Core 24 h offline ist, soll der Admin handeln können. Sobald der Core wieder da ist, gewinnt er — kein Datenverlust.

### Q7: Per-EEG-Opt-out aus dem Core-Lookup?

- (a) Nein — alle EEGs nutzen den Core-Resolver, manuelle Werte sind nur Fallback.
- (b) Ja — neues Setting `eeg_master_data_source` (`core` | `manual`) pro EEG.

**Empfehlung:** (a). Die User-Frage war explizit "braucht es überhaupt eine manuelle Wartungsmöglichkeit?". Antwort: nein, der Fallback bei Core-Ausfall ist genug. Spart Code, spart UI-Komplexität.

### Q8: Auth — geklärt: Service-Account via Keycloak client_credentials

**Recherche 2026-05-14:**
- Keycloak ist bereits Auth-Provider für Core+Billing — und Keycloak unterstützt `grant_type=client_credentials` out-of-the-box.
- `eegfaktura-backend` (`keycloak.go:48,50,66`) zeigt Bestandscode für client_credentials-Flow, das Konzept ist also im Stack vertraut.
- Sowohl `GQLProtect()` (backend) als auch `TenantContext.validateTenant()` (billing) konsumieren standard-OIDC-Tokens mit `tenant`-Claim. Das Token muss nur korrekt geclaimt sein, woher es kommt (User-Login oder client_credentials) ist egal.

**Lösung — keine Code-Änderungen am Core/Billing nötig:**
1. **Im Keycloak einrichten** (einmalige Aktion durch vfeeg-Betreiber):
   - Neuen Client `eegfaktura-onboarding-service` mit `Service Accounts Enabled`
   - Custom-Mapper: hardcoded oder per-Client-Attribut konfigurierter `tenant`-Claim mit allen RC-Nummern, die das Onboarding bedienen darf
   - Rollen-Mapping: passende Rolle, die `GQLProtect`/`TenantContext` akzeptieren
2. **Im Onboarding:**
   - Neue Env-Vars `CORE_OAUTH_CLIENT_ID` + `CORE_OAUTH_CLIENT_SECRET` + `CORE_OAUTH_TOKEN_URL`
   - Token-Cache mit Refresh: Token holen, gültig bis Ablauf, dann refreshen
   - Bei jedem Core/Billing-Call: Bearer aus dem Cache + `tenant`-Header

**Damit ist die "PDF-Render ohne Admin-JWT"-Sorge gelöst** — das Onboarding hat **immer** ein gültiges Token zur Hand, unabhängig davon ob ein Admin gerade aktiv ist.

Vorteil gegenüber der "passive cache, befüllt nur durch Admin-Action"-Variante: vollständige Funktionalität auch bei Erst-Submit nach Reboot, ohne dass jemand vorher die Admin-UI angefasst haben muss.

**Sicherheits-Implikationen:**
- Service-Account-Secret muss in K8s-Secret (nicht in `values.yaml`).
- Tenant-Mapper begrenzt den Blast-Radius: das Token sieht nur die expliziten RC-Nummern.
- Token-Lebensdauer: 1 h Default Keycloak, refresh implementiert.

## Notes

- Dieses Feature schließt **Backlog-Item #1** aus dem 2026-05-14-Memo (EEG-Stammdaten aus Core).
- Logo-Embedding im PDF schließt das Folge-Backlog-Item "Logo aus Core in PDF verwenden".
- Realistische Implementations­dauer:
  - **Phase 1 (Text-Stammdaten, ohne Logo):** 4–6 Stunden — Core-Client, Cache, Resolver, Renderer-Anbindung, Admin-UI Read-only.
  - **Phase 2 (Logo):** 2–4 Stunden, nachdem Q1 verifiziert ist.
- Keine neuen npm/Go-Pakete erforderlich. `image/jpeg` ist Standard Library.
- Q8 ist der **Blocker** — ohne Service-Account-Token im Onboarding-Backend kann die Feature nicht laufen. Daher Empfehlung: vor Implementierungs­start im Core einen Service-Account-Login einrichten.

---
<!-- Sections below are added by subsequent skills -->
