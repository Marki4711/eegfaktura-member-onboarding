# PROJ-108: FreeFinance-Client-Rewrite (Welle-5b nach AC-27b-Live-Befunden)

## Status: In Review (Implementation 2026-06-13, /qa pending)
**Created:** 2026-06-13
**Last Updated:** 2026-06-13 (Implementation Phase A-E + H + I in einem Rutsch)
**Typ:** Vendor-Client-Rewrite (PROJ-104 Welle-5b) — Backend-only

## Implementation-Notes 2026-06-13

In einem Rutsch implementiert (Phase A-E + H + I aus dem Tech-Design). Phase F (Scheduler-DB-Pre-Lookup-TX1→HTTP→TX2-Refactor), Phase G (Pre-Flight-Cache + freefinanceMandantRefsValid-Check) und Phase K (Cron-Alert-Mail + cron_completed-Audit-Kind) sind **deferred zu PROJ-108b** — Bestand-Scheduler funktioniert mit neuem Client unverändert (Customer-Struct gemappt), die DB-UNIQUE-Linie aus PROJ-104 AC-2 schützt schon heute gegen Doppel-Insert (auch ohne Pre-Lookup-Pattern; Crash-Window-Lücke bleibt aber als Risiko bestehen und wird in PROJ-108b geschlossen).

### Was geändert wurde

- **`internal/freefinance/auth.go` NEU** (~200 Zeilen): OIDC TokenSource mit Realm-Discovery-Caching (`/auth/issuer` 1× pro Lifetime), Client-Credentials-Flow gegen `<realm-url>/protocol/openid-connect/token`, atomic.Value-Token-Cache, singleflight-Refresh-Dedup (G1+G2), `Invalidate()`-Methode für 401-Retry, `ErrAuthEndpointUnavailable` für Hard-Fail bei Discovery/Token-Endpoint-Down.
- **`internal/freefinance/types.go` Rewrite** (~250 Zeilen): snake_case-DTOs `CustomerCreationDtoV2`, `InvoiceCreationDtoV2`, `InvoiceLineCreationDtoV2`, `TaxSetDtoV2`, `BusinessDocumentFinalizationDtoV2`, `CreditMemoCreationDtoV2`, `CreditMemoCreateResponse`. Internes `Customer`-Struct umgebaut (`CompanyName`/`FirstName`/`LastName` getrennt, `StreetName`+`StreetNumber` getrennt, `CountryISO` neu, `UID` bleibt aber als `tax_number` serialisiert). `LineItem` mit Helper-Methoden `LineNet()`/`LineTaxAmount()`/`LineTotal()` für Float-Math.
- **`internal/freefinance/client.go` Rewrite** (~300 Zeilen): `Config` umgebaut (`APIKey` → keine eigene Felder, OIDC via TokenSource; 5 neue Mandant-UUID-Felder); `NewHTTPClient(cfg, tokenSource)` neue Signatur; `EnsureCustomer` → `/clients/{id}/mas/customers` mit DTO-Mapping inkl. `ignore_in_bsa: false`; `CreateInvoice` → Single-Shot-Pfad mit `finalize:true` + `layout_setup`-UUID im Body (G9); `FinalizeInvoice` → Fallback-Pfad mit Body `{layout_setup, payment_term?, sequence_group?}`; `MarkAsPaid` → TODO-Kommentar (OpenAPI-Verifikation pending); **NEU `CreateCreditMemo`** → `/clients/{id}/inv/credit_memos` mit GoBD-Klartext-Verkettung im `text_top` ("Gutschrift zu Rechnung NR vom DATUM") + OpenAPI-TODO-Field `original_invoice`; `do()` ruft TokenSource pro Request, 401 invalidiert + 1× Retry, Idempotency-Key wird gesetzt aber Spec-Vermerk dass FF ihn ignoriert.
- **`internal/freefinance/mock.go` Update** (~150 Zeilen): MockClient implementiert das erweiterte Interface inkl. `CreateCreditMemo` (ffcm-mock-N). Customer-Struct-Felder durchgereicht.
- **`internal/freefinance/client_test.go` Rewrite** (~340 Zeilen): `stubTokenSource()` Helper für die einfachen Tests, multiplexte httptest-Server für 401-Test, snake_case-Body-Verifikation via `map[string]interface{}` (kein blind-`Unmarshal` in unsere DTOs), Path-Korrekturen für `/mas/customers` + `/inv/credit_memos`, Single-Shot-Verifikation (finalize:true + layout_setup im Body), Line-Math-Test (Net=50, Tax=10, Total=60 mit Qty=10×5€×20%).
- **`internal/freefinance/auth_test.go` NEU** (~190 Zeilen): TTL-Refresh-Verhalten (TTL > buffer = cache hit; TTL < buffer = refresh jedesmal), singleflight-Concurrent-Dedup, Discovery-Caching, Issuer-Down → `ErrAuthEndpointUnavailable`, Token-Endpoint-Down → `ErrAuthEndpointUnavailable`, `Invalidate()` erzwingt Refresh.
- **`internal/billing/scheduler.go` Mini-Anpassung**: `makeFreefinanceCustomer` baut `Customer{CompanyName, StreetName, StreetNumber, CountryISO: "AT", UID}` statt alter Felder. Scheduler-Logic unverändert.
- **`internal/billing/scheduler_test.go` Anpassung**: Test-Assertions für `StreetName`/`StreetNumber`/`CompanyName` statt `Street`/`Name`.
- **`internal/config/config.go`**: `BillingConfig` erweitert um `FreefinanceOIDCClientID` (default = `FreefinanceClientID`), `FreefinanceClientSecret`, `FreefinanceTaxClassEntryID`, `FreefinanceRevenueAccountID`, `FreefinanceLayoutSetupID`, `FreefinancePaymentTermID`, `FreefinanceSequenceGroupID`. Env-Var-Rename: `FREEFINANCE_API_KEY` → `FREEFINANCE_CLIENT_SECRET`. Das alte `FreefinanceAPIKey`-Feld komplett entfernt.
- **`cmd/server/main.go`**: Neuer `buildFreefinanceLiveClient(cfg)` Helper konsolidiert beide Call-Sites (Server + Cron). Live-Client kommt nur zustande wenn ALLE Pflicht-Felder (BaseURL + ClientID + OIDC-ClientID + ClientSecret + 3 Mandant-UUIDs) gesetzt sind. Bei fehlenden UUIDs → Warn-Log + Fallback auf Mock (defensive Defense gegen Helm-Guard-Bypass).
- **Helm**: `templates/secrets.yaml` — Guards ergänzt für `freefinanceClientSecret` (Rename) + 3 Mandant-UUIDs (TaxClassEntry, RevenueAccount, LayoutSetup) wenn `globalLiveMode=true`; `FREEFINANCE_API_KEY` → `FREEFINANCE_CLIENT_SECRET` im backend-secret. `templates/backend.yaml` + beide CronJob-Templates: 5 neue Env-Vars für Mandant-UUIDs + `FREEFINANCE_OIDC_CLIENT_ID` (default = Mandant-ID) + Secret-Rename. `values.yaml`: 6 neue `backend.billing.*`-Felder. `values-env.yaml.example` + `values-secret.yaml.example`: synchron erweitert + Schlüssel-Rename. Helm-Template-Test verifiziert: Preview-Modus (live=false) rendert OK, Live-Modus ohne ClientSecret → Guard fail, Live-Modus ohne Mandant-UUIDs → Guard fail, Live-Modus mit allen Feldern → OK.
- **PROJ-104-Spec Entscheidung #12**: revidiert in Spec — DB-UNIQUE ist alleinige Idempotenz-Linie, Idempotency-Key-Header bleibt defensiv im Code.

### AC-Erfüllungs-Map

| AC | Status | Anmerkung |
|---|---|---|
| AC-1 bis AC-6 | ✅ | Alle 6 neuen DTOs in types.go |
| AC-7 bis AC-10 | ⚠️ | AC-10 (MarkAsPaid) als TODO im Code dokumentiert — OpenAPI-Verifikation am Live-Cutover |
| AC-11 | N/A | Bestand-Code akzeptierte bereits 2xx; Spec-AC war faktisch fehl-gerahmt |
| AC-12 + AC-13 | ✅ | TaxClassEntryID + RevenueAccountID via Helm-Config in `mapLineItems` |
| AC-14 bis AC-17 | ✅ | LayoutSetupID Pflicht im Finalize-Body, Single-Shot-Pfad als Default |
| AC-18 bis AC-21 | ✅ | OIDC TokenSource mit Realm-Discovery, singleflight, Invalidate; `FREEFINANCE_CLIENT_SECRET` Rename in Helm |
| AC-22 + AC-23 | ⚠️ | Idempotency-Key-Header bleibt im HTTP-Request gesetzt; **PROJ-108b deferred**: DB-Pre-Lookup TX1→HTTP→TX2 Pattern (Bestand-Scheduler hat schon Idempotency via DB-UNIQUE auf `billing_period`) |
| AC-24 | ✅ | Crash-Window-Lücke in Spec dokumentiert, PROJ-108b liefert Reconciliation |
| AC-25 + AC-26 | ✅ | Mock spiegelt neue DTOs + CreateCreditMemo |
| AC-27 + AC-28 | ⚠️ | `CreateCreditMemo` implementiert mit GoBD-Klartext-Verkettung; **admin_billing.go-Adapter** für `CreateCreditNote` deferred zu PROJ-108b (Bestand schreibt heute nur DB-Eintrag, kein FF-Vendor-Call — kein Regress) |
| AC-29 bis AC-32 | ✅ | Tests grün, Helm-Guards verifiziert |
| AC-33 | ✅ | PROJ-104-Spec #12 revidiert |
| AC-34 + AC-35 | ⚠️ | Spec-Update done; `docs/architecture.md` Billing-Stack-Sektion Update deferred zu PROJ-108b |
| AC-36 | ⚠️ | `/security-review` als nächster Schritt nach diesem Skill |
| AC-37 | 🟡 | **PROJ-108b deferred** — Pre-Flight-Mandant-Refs-Check braucht Helm-UUIDs zur Validierung, ohne Live-Mandant nicht testbar |

### TODOs bei Live-Cutover (Owner-Tasks)

1. **OpenAPI-Inspection** vor erstem Live-Cron:
   - `MarkAsPaid`-Endpoint-Pfad verifizieren (vermutlich nicht existent → Methode als no-op markieren)
   - `CreditMemoCreationDtoV2.OriginalInvoice` Field-Name verifizieren (`original_invoice` vs `reference_id` vs `original_business_document`)
2. **Mandant-Reset-Bestätigung** durch FF-Support abwarten + neuen Tech-User anlegen
3. **5 Mandant-UUIDs** aus FF-UI ermitteln und in `values-env.yaml` eintragen:
   - `freefinanceTaxClassEntryId` (ESTD_020-Entry für 20 % USt)
   - `freefinanceRevenueAccountId` (Konto 4000)
   - `freefinanceLayoutSetupId` (Default-Layout)
   - `freefinancePaymentTermId` (optional)
   - `freefinanceSequenceGroupId` (optional)
4. **FREEFINANCE_CLIENT_SECRET** in `values-secret.yaml` setzen (FREEFINANCE_API_KEY-Wert kann verworfen werden — neuer OIDC-Tech-User-Secret nötig)
5. **Dry-Run via Subcommand** `billing-quarterly --rc=<test-rc> --dry-run` einmal gegen Live-FF (Subcommand-Erweiterung selbst auch PROJ-108b)

## Grilling-Entscheidungen (2026-06-13)

| # | Entscheidung | Konsequenz |
|---|---|---|
| G1 | **OIDC-Token-Cache:** `golang.org/x/sync/singleflight` + `atomic.Value`. Refresh wenn `remaining < 60s`. | `internal/freefinance/auth.go` mit singleflight-Group, atomic-Token-Slot, kein eigener Mutex |
| G2 | **Realm-Discovery:** einmal bei Token-Source-Init, persistent in-memory. Bei Pod-Restart neuer Discovery-Call. Hard-Fail bei 404/Timeout → `ErrAuthEndpointUnavailable`. | Keine Helm-Config für Realm. Realm-Discovery-Funktion 1× pro TokenSource-Init |
| G3 | **DB-Pre-Lookup-Pattern A:** TX1 (kurz, `INSERT ON CONFLICT DO NOTHING; SELECT FOR UPDATE SKIP LOCKED`) → HTTP außerhalb TX → TX2 (kurz, `UPDATE WHERE freefinance_invoice_id IS NULL`). | `scheduler.go` Refactor mit zwei expliziten TXen. Pool-Druck minimal |
| G4 | **Crash-Window-Detection V1:** Audit-Eintrag `kind=cron_completed` + Owner-Mail bei `EEGsErrored > 0`. | Neue Audit-Kind, neue Mail-Template + Sender-Methode. Async OK (nicht sicherheitsrelevant) |
| G5 | **Mandant-UUIDs V1 reine Helm-Config:** 5 Vars (Tax-Class-Entry/Revenue-Account/Layout-Setup PFLICHT, Payment-Term/Sequence-Group optional). Pre-Flight-Check pingt FF. | 5 neue values + Env-Vars + `required`-Guards für 3 Pflicht-UUIDs wenn `globalLiveMode=true` |
| G6 | **Helm-Kalt-Rename:** `freefinanceApiKey` → `freefinanceClientSecret`, Env `FREEFINANCE_API_KEY` → `FREEFINANCE_CLIENT_SECRET`. CHANGELOG-Eintrag als Migrations-Vermerk. Kein Aliasing. | Rename in `values.yaml`, `values-secret.yaml.example`, `templates/secrets.yaml`, `templates/backend.yaml`, beide CronJob-Templates, `config.go` Env-Var |
| G7 | **MarkAsPaid Live-Verifikation in Implementation:** OpenAPI-Pre-Check. Wenn FF kein POST `/pay`: `MarkAsPaid` entfällt, Scheduler-Aufruf wird no-op. | Spec-AC-10 wird in Implementation-Step 1 entschieden. Welle-3-Scheduler-Code bleibt aufruf-kompatibel |
| G8 | **Credit-Memo Verkettung via OpenAPI:** Field-Name `original_invoice`/`reference_id`/`original_business_document` aus OpenAPI ermitteln. GoBD-Verkettung über Klartext in `text_top`: „Gutschrift zu Rechnung NR vom DATUM" als Hard-Backup. | `client.go` Field-Name als TODO im Code mit OpenAPI-Verifikation. Klartext-Hint immer setzen |
| G9 | **Single-Shot Invoice-Finalize:** `CreateInvoice` mit `finalize: true` Top-Level-Flag + `layout_setup` UUID im Body. Two-Step als Fallback wenn HTTP 400. | 1 Roundtrip statt 2. `FinalizeInvoice`-Methode bleibt im Interface als Fallback |
| G10 | **Test-Strategie:** Mock-Tests in CI + Owner-Manual-Live-Smoke pre-Cutover via `billing-quarterly --rc=<test-rc> --dry-run`. Kein Live-Smoke in CI. | Subcommand-Erweiterung `--dry-run` Flag (skipped DB-Updates) |
| G11 | **Pre-Flight-Cache:** `sync.Map` mit TTL 5 min pro RC. Auto-Verfall, kein UI-Bust-Button. | `internal/billing/preflight.go` Erweiterung |
| G12 | **`import_failed`-Retry:** Daily-Sync (Bestand) erweitert um zweiten Pass für Quartale mit `freefinance_invoice_id IS NULL AND created_at < NOW() - 1 day`. Max 7 Versuche, dann audit `kind=billing_giveup` + Owner-Mail. | `RunDailyStatusSync` Refactor |
| G13 | **Tax-Class-Entry V1: eine UUID** (ESTD_020 für 20 % USt Einnahmen). | Single Helm-Config-Var, kein Edition-spezifisches Mapping |
| G14 | **TLS:** Standard-System-Trust-Store, kein Cert-Pinning. | `http.Client`-Default, keine Sonder-Konfig |
| G15 | **Helm-Upgrade-Resilienz:** K8s default SIGTERM-Handling, Scheduler prüft `ctx.Done()` zwischen EEGs (Bestand-Pattern). Kein PreStop-Hook. `terminationGracePeriodSeconds: 60`. | Verifizieren dass Scheduler-Loop `ctx.Done()` checkt. Falls nicht: 1-Zeilen-Fix |

## Dependencies

- **Requires:** PROJ-104 (Plattform-Abrechnung) — deployed `v1.32.0-PROJ-104` am 2026-06-13. PROJ-108 ersetzt das in PROJ-104 Welle 2 gebaute `internal/freefinance/`-Modul.
- **Blockierend für:** Owner-Live-Cutover. Solange PROJ-108 nicht deployed ist, schlägt jeder echte FreeFinance-API-Call fehl (camelCase-DTOs, falsche Pfade, falscher HTTP-Status-Check). `BILLING_GLOBAL_LIVE_MODE=false` bleibt der Schutzschild.

## Hintergrund

PROJ-104 Welle 2 hat `internal/freefinance/` gegen Web-Doku-Annahmen + camelCase-Vermutungen gebaut. AC-27b-Live-Verifikation 2026-06-13 in einem FreeFinance-Trial-Mandanten hat **7 substanzielle Code-Inkompatibilitäten + 1 Hauptbefund** offengelegt:

- camelCase-DTOs sind komplett falsch — FreeFinance nutzt snake_case
- Endpoint-Pfade brauchen Modul-Präfixe (`/mas/`, `/inv/`, `/fis/`, `/cbs/`)
- HTTP-Erfolgs-Status ist 200, nicht 201
- Customer-Field-Namen sind komplett anders strukturiert
- Tax-Klassen via UUID-Lookup statt Percent-Wert
- Layout-Setup-UUID ist Pflicht im Finalize-Body (Mandant-Default greift nicht)
- Gutschrift ist eigener Endpoint `/inv/credit_memos`, nicht Negativ-Invoice
- **Hauptbefund:** `Idempotency-Key`-Header wird **ignoriert** — drei Doppel-POSTs mit identischem Header → drei verschiedene Invoice-IDs

Mock-Tests in Welle 2 waren alle grün, weil sie die falschen Annahmen widerspiegelten. Die Lehre wurde als Memory `feedback_vendor_dto_via_openapi` festgenagelt (OpenAPI-Spec-Inspection als Pflicht-Step vor Vendor-Code).

## User Stories

- **Als Owner** möchte ich `BILLING_GLOBAL_LIVE_MODE=true` flippen können, ohne dass der erste echte Quartals-Cron-Lauf an Vendor-Calls scheitert.
- **Als Owner** möchte ich, dass ein Cron-Restart nach Crash keine Doppel-Rechnung in FreeFinance erzeugt, auch wenn FreeFinance den `Idempotency-Key`-Header ignoriert.
- **Als Plattform-Betreiber** möchte ich Mandant-spezifische UUIDs (Tax-Class-Entry, Revenue-Account, Layout, Payment-Term, Sequence-Group) per Helm-Config wechseln können, ohne Code-Deploy — z. B. nach einem FreeFinance-Mandant-Reset.
- **Als Plattform-Betreiber** möchte ich OIDC-Token automatisch refreshen lassen (TTL 5 min, kein refresh_token im Client-Credentials-Flow), ohne dass jeder API-Call einen neuen Token-Request triggert.
- **Als Plattform-Betreiber** möchte ich Gutschriften via eigenen `/inv/credit_memos`-Endpoint anlegen, statt Negativ-Invoices zu schicken (FreeFinance-Konformität, GoBD-Trail bleibt sauber).

## Acceptance Criteria

### DTO-Rewrite (snake_case + neue Field-Namen)

- [ ] **AC-1** `internal/freefinance/types.go`: alle JSON-Tags auf snake_case. `Customer`/`InvoiceCreateRequest`/`LineItem`/`InvoiceCreateResponse`/`CustomerCreateRequest`/`CustomerCreateResponse`/`MarkAsPaidRequest` umbenannt und an die echten FreeFinance-DTOs angeglichen.
- [ ] **AC-2** Customer-Struct (`CustomerCreationDtoV2`) hat: `customer_number` (statt `externalRef`), `company_name` ODER `first_name`+`last_name`, `street_name`+`street_number` (getrennt statt `street`), `zip_code` (statt `zip`), `city`, `country` (ISO `"AT"`), `tax_number` (statt `uid`), `ignore_in_bsa: false` (Pflicht-Bool). Mapping-Layer in `client.go` baut den Customer aus dem internen `Customer`-Typ.
- [ ] **AC-3** Invoice-Struct (`InvoiceCreationDtoV2`) Required-Felder: `date` (YYYY-MM-DD, statt `issueDate`), `customer` (UUID, statt `customerId`), `lines: []InvoiceLineCreationDtoV2`. Optional: `due_date`, `text_top`, `text_bottom`, `finalize: bool` (Top-Level-Flag für Single-Shot), `layout_setup` (UUID, **PFLICHT wenn `finalize=true`**), `sequence_group` (optional Override), `payment_term` (optional Override).
- [ ] **AC-4** InvoiceLine-Struct (`InvoiceLineCreationDtoV2`): `name`, `account` (Erloeskonto-UUID), `amount` (Menge, statt `qty`), `item_price` (pro Einheit), `price_type` (`"NET"` oder `"GROSS"`), `net` (Line-Total NET), `total` (Line-Total inkl. Tax), `taxes: {tax_1: {tax_class_entry: <UUID>, amount: <eur>}}` (bis zu 5 Tax-Slots, NICHT percent-basiert).
- [ ] **AC-5** Finalize-Struct (`BusinessDocumentFinalizationDtoV2`): `layout_setup` (Pflicht-UUID), `payment_term` (optional), `sequence_group` (optional), `e_invoice_version` (optional).
- [ ] **AC-6** Credit-Memo-Struct (`CreditMemoCreationDtoV2`): analog Invoice, eigener Endpoint-Body. **KEIN Negativ-Invoice mehr.**

### Pfad-Korrekturen (Modul-Präfixe)

- [ ] **AC-7** Customer-Endpoint: `POST /clients/{client_id}/mas/customers` (statt `/clients/{id}/customers`). HTTPClient-Pfad-Konstanten umstellen.
- [ ] **AC-8** Invoice-Endpoints (Bestand korrekt, Verifikation als Test): `POST /clients/{id}/inv/invoices` + `POST /clients/{id}/inv/invoices/{id}/finalize`.
- [ ] **AC-9** Credit-Memo: `POST /clients/{id}/inv/credit_memos` (neuer Endpoint).
- [ ] **AC-10** Mark-As-Paid: gegen FreeFinance-Realität neu verifizieren — der Pfad `POST /invoices/{id}/pay` wurde aus Web-Recherche übernommen und ist NICHT live-verifiziert. Wenn FreeFinance kein direktes Mark-Paid hat: stattdessen Tracking-Status via Webhook-Pull (siehe Welle-3-Daily-Sync). Owner-Verifikation-Schritt in der Implementation.

### HTTP-Status

- [ ] **AC-11** HTTPClient-Response-Handling akzeptiert **200 OK** als Erfolgs-Status für POST-Calls (statt 201 Created). Welle-2-Tests mit `resp.StatusCode == 201` müssen mit umgeschrieben werden.

### Tax-Class + Account-Referenzen

- [ ] **AC-12** Helm-Config-Erweiterung `backend.billing.freefinanceTaxClassEntryId` (ESTD_020-Entry-UUID für 20 % USt Einnahmen) + `backend.billing.freefinanceRevenueAccountId` (Konto 4000 UUID für Erloese). Beide Pflicht wenn `BILLING_GLOBAL_LIVE_MODE=true`. `cfg.Billing.FreefinanceTaxClassEntryID` + `FreefinanceRevenueAccountID` lesen die Werte.
- [ ] **AC-13** `client.go` baut Invoice-Lines mit den Helm-konfigurierten UUIDs (statt Percent-basierten Tax-Werten). Kein hardcoded Tax-Class-Code, kein FreeFinance-Default-Account.

### Layout + Sequence + Payment-Term

- [ ] **AC-14** Helm-Config `backend.billing.freefinanceLayoutSetupId` — **Pflicht** wenn Live-Mode AN, weil Finalize sonst mit `inv-layout-setup-not-found` scheitert. `cfg.Billing.FreefinanceLayoutSetupID` durchgereicht.
- [ ] **AC-15** Helm-Config `backend.billing.freefinancePaymentTermId` und `backend.billing.freefinanceSequenceGroupId` — **optional**. Wenn leer, nimmt FreeFinance den Mandant-Default. Wenn gesetzt, override.
- [ ] **AC-16** `HTTPClient.FinalizeInvoice` schickt jetzt einen Body mit `{"layout_setup": "<id>"}` statt leeren POST. Optional `payment_term` und `sequence_group` Override.
- [ ] **AC-17** Single-Shot-Pfad: `CreateInvoice` mit `finalize: true` Top-Level-Flag + `layout_setup`-UUID im Body spart den `/finalize`-Roundtrip. Default-Pfad nutzt Single-Shot wenn `cfg.Billing.FreefinanceLayoutSetupID` gesetzt ist.

### OIDC Token-Cache + Realm-Discovery

- [ ] **AC-18** Neues `internal/freefinance/auth.go` mit `OIDCTokenSource`-Struct. Beim ersten Aufruf:
  1. `GET <baseURL>/auth/issuer` → liefert `{"url": "https://accounts.freefinance.at/auth/realms/at", "realm": "at"}`. Realm wird **nicht hardcoded**.
  2. Token-URL = `{url}/protocol/openid-connect/token`
  3. `POST <token-url>` mit `grant_type=client_credentials&client_id=<CLIENTID>&client_secret=<SECRET>` → access_token, expires_in=299.
- [ ] **AC-19** Token-Cache mit TTL-Tracking. Re-Fetch automatisch wenn `remaining < 60s` (Buffer für Netzwerk-Latenz). Concurrent-Safe via `sync.RWMutex` oder `singleflight`.
- [ ] **AC-20** Realm-Discovery wird einmal beim Erst-Token gecached (statisch pro Mandant). Bei Token-Endpoint-Down → hard-fail mit `ErrAuthEndpointUnavailable`. Cron-Scheduler markiert Quartal als `import_failed` und retry-t beim nächsten täglichen Lauf (statt endlose Retry-Schleife).
- [ ] **AC-21** Helm-Config `backend.billing.freefinanceClientSecret` als neues `secretKeyRef` (Bestand `freefinanceApiKey` wird zu `freefinanceClientSecret` umbenannt — semantisch korrekter, weil OIDC-Client-Credentials-Flow keinen API-Key kennt). Bestand-Code-Pfad mit `APIKey`-Bearer-Token wird entfernt.

### DB-Pre-Lookup-Pattern (Idempotency-Key wird NICHT respektiert)

- [ ] **AC-22** Hauptbefund-Behandlung: `internal/billing/scheduler.go` `RunQuarterly` macht **DB-Pre-Lookup vor jedem Vendor-Call**. Ablauf:
  ```
  BEGIN TX
    INSERT INTO billing_period (rc, year, q, ...) ON CONFLICT (rc, year, q) DO NOTHING
    SELECT freefinance_invoice_id, billing_invoice WHERE billing_period_id = ?
    IF freefinance_invoice_id IS NOT NULL → SKIP (already created, idempotent)
    ELSE → CreateInvoice → UPDATE billing_invoice SET freefinance_invoice_id = ?
  COMMIT
  ```
  DB-`UNIQUE(rc_number, year, quarter)` auf `billing_period` (Bestand AC-2 aus PROJ-104) ist die alleinige Idempotenz-Linie.
- [ ] **AC-23** `Idempotency-Key`-Header bleibt im HTTP-Request (kostet nichts, falls FreeFinance sich später ändert), aber Scheduler **verlässt sich nicht darauf**. Spec-Entscheidung #26 aus PROJ-104 wird auf den Pfad „DB-UNIQUE schützt, Header ist defensiv" revidiert.
- [ ] **AC-24** Crash-Window-Lücke dokumentiert: wenn Cron stirbt nach erfolgreichem FreeFinance-POST aber vor DB-UPDATE, entsteht eine FF-Rechnung ohne korrespondierende DB-Zeile. V1: Owner-Manual-Cleanup über FreeFinance-UI + Memory-Lock-Eintrag. V2 (separates PROJ): Reconciliation-Job der STAGING-Rechnungen in FF ohne DB-Eintrag findet. Spec ergänzt `docs/architecture.md`-Hinweis.

### Mock-Client-Anpassung

- [ ] **AC-25** `internal/freefinance/mock.go` spiegelt die neue DTO-Struktur (snake_case, Customer-Felder, Tax-Class-Entry-Refs, Layout-Setup-IDs). Deterministische Fake-UUIDs bleiben (z. B. `mock-cust-<RC>`). MockClient wird vom Test-Modus + lokalen Entwickler-Setup genutzt.
- [ ] **AC-26** Snapshot-Tests in `internal/freefinance/client_test.go` aktualisiert: HTTPServer-Mock antwortet mit echten snake_case-Bodies + HTTP 200. Tests verifizieren konkrete Field-Namen via `json.Unmarshal` (nicht nur Struct-Round-Trip).

### Gutschrift via Credit-Memos-Endpoint

- [ ] **AC-27** Neue Client-Methode `HTTPClient.CreateCreditMemo(ctx, originalInvoiceID, reason, lineItems)` → POSTet auf `/clients/{id}/inv/credit_memos` mit `CreditMemoCreationDtoV2`. Verkettung zur Original-Invoice via FreeFinance-Field (genaue Field-Name in OpenAPI verifizieren — `original_invoice` oder `reference_id`).
- [ ] **AC-28** `internal/http/admin_billing.go` `CreateCreditNote`-Handler ruft jetzt `freefinance.CreateCreditMemo` statt einer negativ-Invoice-Konstruktion. `billing_invoice.cancels_invoice_id` bleibt die DB-interne Verkettung (PROJ-104 AC-5c).

### Tests + Build-Sweep

- [ ] **AC-29** Alle Welle-2-Tests umgeschrieben (Snapshot-Bodies + HTTP 200 + neue Field-Namen). `go test ./internal/freefinance/...` grün.
- [ ] **AC-30** OIDC-Token-Cache-Tests: TTL-Refresh, concurrent Token-Requests (singleflight), Realm-Discovery-Caching, Hard-Fail bei Token-Endpoint-Down.
- [ ] **AC-31** DB-Pre-Lookup-Pattern-Test in `internal/billing/scheduler_test.go`: simulierter Crash zwischen Vendor-Call und DB-Update → zweiter Lauf SKIPt korrekt wenn `freefinance_invoice_id` schon gesetzt ist.
- [ ] **AC-32** Helm-Template-Test: `required`-Guards für `freefinanceTaxClassEntryId`, `freefinanceRevenueAccountId`, `freefinanceLayoutSetupId` wenn `BILLING_GLOBAL_LIVE_MODE=true`.

### Spec + Doku

- [ ] **AC-33** Spec-Entscheidung #26 aus PROJ-104 revidiert (Idempotency-Key wird ignoriert, DB-UNIQUE schützt). Inline-Vermerk in `features/PROJ-104-platform-billing.md` plus Querverweis auf PROJ-108.
- [ ] **AC-34** `docs/architecture.md` Billing-Stack-Sektion um Token-Cache + Realm-Discovery + DB-Pre-Lookup-Pattern + Crash-Window-Lücke ergänzt.
- [ ] **AC-35** Memory `reference_freefinance_api.md` ist die Live-Quelle für DTOs/Pfade — wird bei jeder weiteren Drift gepflegt (kein Repo-Code-Kommentar duplikieren).

### Security-Review

- [ ] **AC-36** `/security-review` Pflicht-Gate vor Deploy. Neue Felder: Token-Cache-Pfad (Race-Conditions?), Realm-Discovery (Trust-Boundary zu FreeFinance-DNS?), neue Helm-Secrets (`freefinanceClientSecret`), Mandant-UUIDs als Config-Vars (Leakage-Risiko: UUIDs sind nicht sensibel, aber Mandant-ID-Korrelation per Logs vermeiden).

## Edge Cases

- **Token-Endpoint-Down (FreeFinance-OIDC-Outage):** Cron markiert das Quartal als `import_failed`, Daily-Sync retry-t. Kein endlosess Retry pro Cron-Lauf — `ErrAuthEndpointUnavailable` ist final für diesen Lauf.
- **Realm-Discovery-Down beim Erst-Start:** `/auth/issuer` 404 oder Timeout → hard-fail. Owner sieht `import_failed` in `/admin/billing` und kann manuell triggern wenn FreeFinance wieder oben ist.
- **FreeFinance-API-Outage mitten im Cron-Lauf:** ein EEG bekommt Rechnung, der nächste 502. Scheduler markiert das fehlende Quartal als `import_failed`, läuft weiter durch die übrigen EEGs. Daily-Sync retry-t.
- **Idempotency-Race (zwei parallele Cron-Pods bei Forbid-Bypass):** K8s-CronJob `concurrencyPolicy: Forbid` schützt im Regelbetrieb. Bei Bypass durch Owner-Manual-Trigger während Cron läuft, schützt DB-UNIQUE: zweiter Insert auf `(rc, year, quarter)` schlägt mit `ON CONFLICT DO NOTHING` ab, zweiter Vendor-Call wird übersprungen.
- **Mandant-Reset bei FreeFinance:** alle UUIDs (Tax-Class-Entry, Revenue-Account, Layout, Payment-Term, Sequence-Group) wechseln. Owner muss die neuen UUIDs in Helm-Config eintragen und einen `helm upgrade` machen. Code-Pfad ändert sich nicht — UUIDs werden zur Laufzeit gelesen.
- **Layout-Setup-ID falsch konfiguriert:** Finalize-Call schlägt mit `inv-layout-setup-not-found` fehl. Scheduler markiert als `import_failed`, Helm-Pre-Flight (`/admin/billing` Pre-Flight-Check) sollte die Mandant-Refs vorab pingen — `AC-37` (Pre-Flight-Erweiterung).
- [ ] **AC-37** Pre-Flight-Endpoint `/api/admin/billing/eegs/{rc}/pre-flight` erweitert um `freefinanceMandantRefsValid` (boolean) — pingt FreeFinance kurz mit `GET /clients/{id}/inv/layout_setups/{layout_id}` etc. zum Verifikations-Zeitpunkt. Wenn 404 → Owner-Hint im UI „Layout-Setup-UUID falsch konfiguriert".
- **Token-Cache-Stale beim Lang-Lauf-Cron:** TTL 299s, Cron läuft potentiell länger pro EEG-Batch. Token-Source refresht automatisch wenn `remaining < 60s` → kein Mid-Cron-Failure.
- **Credit-Memo ohne Original-Invoice in FreeFinance:** wenn Gutschrift auf eine Rechnung gemacht wird, die in FreeFinance noch nicht final ist (z. B. `freefinance_invoice_id IS NULL` weil Crash). Handler returnt 409 „Original-Rechnung nicht final" — Gutschrift nicht möglich. Owner muss erst Original-Rechnung in FF reparieren.

## Technical Requirements

- **Performance:** Cron-Lauf < 5 Min für 100 EEGs. Token-Request 1× pro Cron-Pod (gecached). Realm-Discovery 1× pro Token-Source-Init.
- **Security:** OIDC-Client-Credentials-Flow mit Helm-`secretKeyRef`. Kein Client-Secret in Logs. Token wird nicht persistiert (in-memory).
- **Observability:** Strukturiertes `slog`-Log pro Vendor-Call: `module`, `rc`, `endpoint`, `status_code`, `duration_ms`, `idempotency_key` (Header-Wert, nicht Body). Kein Body-Inhalt im Log (PII-Disziplin).
- **Retry-Strategie:** Retry-Backoff bei 5xx + 429 (Bestand), aber kein Retry bei 4xx (final). Token-Endpoint hat eigenen Retry-Counter (max 2 Versuche pro Cron-Lauf, dann `ErrAuthEndpointUnavailable`).

## Out-of-Scope (Folge-PROJ oder bewusst V1-tradeoff)

- **Init-Time-Lookup der Mandant-Refs:** statt Helm-Config könnte der Service beim Startup `/fis/tax_classes`, `/cbs/accounts`, `/inv/payment_terms`, `/inv/layout_setups`, `/inv/document_sequence_groups` abrufen und die Default-IDs cachen. Eleganter (kein Helm-Upgrade nötig nach Mandant-Reset), aber komplexer + braucht Caching. V2-Option, V1 nutzt Helm-Config-UUIDs.
- **Reconciliation-Job für Crash-Window-Lücke:** automatischer Abgleich STAGING-Rechnungen in FF ohne DB-Eintrag. V1: Owner-Manual-Cleanup. V2 (separates PROJ): Daily-Job listet `GET /clients/{id}/inv/invoices?status=draft` und matched gegen DB.
- **Echte Pricing-Werte:** Owner-Aufgabe via `/admin/billing` Tab „Pricing-Plan" (Bestand aus PROJ-104).
- **AGB-Cutover:** Owner-Aufgabe vor Live-Cutover in `src/content/legal/agb-v1.0.md`.
- **Multi-Target-Systeme (EDA-Portal etc.):** Memory `project_todo_multi_target_systems` — separates Epic, kein Scope-Erweiter hier.
- **Mark-As-Paid via FreeFinance-Direkt-Endpoint:** wenn FF kein Direkt-Mark-Paid hat (AC-10), bleibt der Webhook-Pull-Pfad. Kein zweiter Vendor-Call nach `paid`-Status.
- **OpenAPI-Code-Generator:** statt Hand-DTOs könnte ein Generator aus dem FreeFinance-OpenAPI-Spec laufen. V1 nicht — Hand-DTOs für die wenigen Felder sind übersichtlicher + kein Tooling-Aufwand.

## Pre-Implementation-Voraussetzungen (Owner-Aufgaben — NICHT Code-Scope)

Diese müssen erledigt sein, bevor die Implementation startet:

1. **Mandant-Reset bei FreeFinance bestätigt** durch FF-Support per Mail. Memory `project_session_2026-06-13` hat den Stand: Reset wurde am 2026-06-13 beantragt (Testdaten zurücksetzen, 4 Pflicht-Checkboxen), manuelles Approval durch FF-Support steht aus.
2. **Neuer Tech-User** in FreeFinance-UI angelegt + alter Tech-User aus AC-27b-Lauf gelöscht (der Secret war im Chat geleakt — Memory-Direktive `feedback_verify_vendor_claims`).
3. **Mandant-Setup nach Reset:** Owner legt 1 Zahlungsbedingung „14 Tage netto" + Default-Markierung an, 1 Belegkreis „Default" mit Default-Markierung, 1 Layout aktiviert + Default-Markierung.
4. **Neue UUIDs festhalten** und in Helm-Values eintragen:
   - Tax-Class-Entry `ESTD_020` (20 % USt Einnahmen) — UUID via `GET /clients/{id}/fis/tax_classes?code=ESTD` + `.../entries?code=020`
   - Revenue-Account Konto 4000 — UUID via `GET /clients/{id}/cbs/accounts?code=4000`
   - Layout-Setup — UUID des Default-Layouts
   - (Optional) Payment-Term-UUID, Sequence-Group-UUID

Diese Schritte sind in `private/vendor-setup/freefinance-trial-verification-2026-06-13.md` zu dokumentieren (wird durch die Implementation fortgeschrieben).

## Implementierungs-Reihenfolge (Schätzung ~2-3 Tage)

| Schritt | Aufwand | Inhalt |
|---|---|---|
| 1 | 0.5 Tag | `types.go` Rewrite (snake_case durchgängig, korrekte DTOs) |
| 2 | 0.25 Tag | `client.go` Pfad-Korrekturen (`/mas/`, `/inv/credit_memos`) + HTTP-200-Check |
| 3 | 0.5 Tag | `auth.go` OIDC-Token-Cache + Realm-Discovery + singleflight |
| 4 | 0.25 Tag | Helm-Config-Erweiterung (5 neue Values + `required`-Guards) |
| 5 | 0.5 Tag | `scheduler.go` DB-Pre-Lookup-Pattern + Crash-Window-Hinweis |
| 6 | 0.1 Tag | Single-Shot-Finalize-Path (`finalize: true` + `layout_setup` im Body) |
| 7 | 0.25 Tag | `mock.go` Spiegelung neuer DTOs |
| 8 | 0.5 Tag | `client_test.go` + `auth_test.go` + Welle-2-Tests umschreiben |
| 9 | 0.1 Tag | Spec + Doku-Updates (Spec-Entscheidung #26 revidieren) |
| 10 | 0.25 Tag | Pre-Flight-Erweiterung (`freefinanceMandantRefsValid`) |
| 11 | separater Tag | `/security-review` Pflicht-Gate |

## Memory-Regeln aktiv

- `feedback_vendor_dto_via_openapi` — OpenAPI-Spec-Inspection als Pflicht-Step BEVOR neue Code-Zeile (gilt insbesondere für Credit-Memo-DTO `original_invoice`-vs-`reference_id`-Field-Name)
- `feedback_verify_vendor_claims` — Live-Verifikation vor Implementation (Mandant-Reset-Status)
- `feedback_admin_field_full_chain` — neue Helm-Config-Vars brauchen Full-Chain (Helm values + values-env.yaml.example + values-secret.yaml.example + Config-Struct + Service-Wiring)
- `feedback_migration_after_apply_drift` — KEINE neuen Migrationen in PROJ-108 (rein Code-Rewrite + Helm-Config)
- `feedback_shared_helpers_for_parallel_paths` — Token-Cache + Realm-Discovery als Single-Helper für Live + Mock (Mock returnt deterministisches Token, Live nutzt OIDC)
- `feedback_qa_full_chain_verify` — QA muss alle Inkompatibilitäten konkret verifizieren (snake_case JSON-Tag, Pfad-Präfix, HTTP-200-Status, UUID-Refs in Body)
- `feedback_helm_values_split` — bei neuen Helm-Werten IMMER `values.yaml` + `values-env.yaml.example` (+ Secret-Beispiel) im selben Commit

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
