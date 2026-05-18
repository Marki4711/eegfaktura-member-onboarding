# PROJ-33: EEG-Logo aus eegFaktura-Core (Phase 2 von PROJ-32)

## Status: Deployed
**Created:** 2026-05-14
**Last Updated:** 2026-05-14 (Stages A–F implemented; Q1 resolved via live `billingConfigs/tenant/TE100200` curl — `headerImageFileDataId` is the gate field, billing-config `id` is the URL parameter)

## Dependencies
- Parent: **PROJ-32** (EEG-Stammdaten-Sync) — diese Phase 2 hängt direkt am bestehenden Sync-Mechanismus und teilt die `registration_entrypoint`-Tabelle.
- Requires: PROJ-12 (SEPA-Mandat-PDF) und PROJ-21 (Beitrittsbestätigung-PDF) — das Logo wird in beiden gerendert.
- Berührt: [[eegFaktura Core API contract]] (Logo lebt im `eegfaktura-billing`-Service unter `/cash/api/...`).

## Hintergrund

PROJ-32 Phase 1 holt sieben Text-Stammdaten aus dem eegFaktura-Core in die Onboarding-DB. Das Logo der Energiegemeinschaft wird im Core ebenfalls hinterlegt (über die EEG-Verwaltung), aber bisher nicht ins Onboarding gespiegelt — die generierten PDFs (Beitrittsbestätigung, SEPA-Mandat) tragen daher keine EEG-Marke und wirken generisch.

**URL-Pfade (verifiziert 2026-05-14 via DevTools):**
- `GET /cash/api/billingConfigs/tenant/{rcNumber}` → JSON mit Billing-Config inkl. Logo-Referenz (genaues Feld bei Implementierung gegen Live-System verifizieren — Kandidaten: `logoImage`, `logoUrl`, oder eine ID, mit der man den zweiten Call macht)
- `GET /cash/api/billingConfigs/{id}/logoImage` → PNG-Bytes

Beide Endpoints liegen hinter dem `/cash/api/`-Prefix (Java/Spring `eegfaktura-billing`-Service) — **anderer Service als die GraphQL-`eeg`-Query** (Go `eegfaktura-backend` unter `/api/`). Die hostname-only `CORE_BASE_URL`-Architektur deckt beide ab (Pfad wird im coreclient hardcoded).

## User Stories

- Als **EEG-Admin** möchte ich, dass mein im Core hinterlegtes Logo automatisch auf der Beitrittsbestätigung und im SEPA-Mandat erscheint — ohne dass ich das Logo zusätzlich im Onboarding hochladen muss.
- Als **EEG-Admin** möchte ich auf der EEG-Einstellungen-Seite sehen, welches Logo aktuell für PDFs verwendet wird.
- Als **vfeeg-Betreiber** möchte ich, dass ein fehlendes oder beschädigtes Logo die PDF-Generation nicht hart fehlschlagen lässt — Mails sollen weiterhin rausgehen, auch wenn das Logo gerade nicht verfügbar ist.

## Architekturentscheidungen

1. **Single source of truth bleibt `registration_entrypoint`.** Wie bei den Text-Stammdaten: Sync schreibt die Logo-Bytes direkt in die DB; PDF-Rendering liest aus der DB. Kein Live-Fetch beim PDF-Render, kein separates Object-Storage.
2. **256 KB Cap auf die Logo-Bytes**, durchgesetzt via `io.LimitReader` beim Core-Fetch. Realistische Logo-Größen liegen bei 15–150 KB — der Cap fängt versehentlich hochgeladene Hi-Res-Fotos ab, bevor sie die DB aufblähen. Worst-Case 256 KB × 500 EEGs = ~128 MB, im DB-Volumen vernachlässigbar.
3. **MIME-Whitelist:** `image/png`, `image/jpeg`, `image/gif` — was `gofpdf.RegisterImageReader` akzeptiert. Andere Content-Types werden hart abgewiesen.
4. **Logo-Sync ist Teil des bestehenden Sync-Calls**, kein separater Button. Best-effort: wenn der Logo-Fetch fehlschlägt, schlägt der Sync nicht ganz fehl — Stammdaten werden trotzdem geschrieben, Response enthält einen Logo-Warnhinweis.
5. **Auth identisch zu Phase 1:** Admin-JWT-Forwarding + `tenant`-Header (siehe [[eegFaktura Core API contract]] „Auth model"). Kein Service-Account.
6. **Kein Server-Side-Resize in V1.** Wir speichern und embedden, was der Core liefert. Falls jemand am Cap scheitert: klare Fehlermeldung „Logo > 256 KB — bitte in eegFaktura ein kleineres hinterlegen". Resize wäre eine Phase 3, falls je nötig.

## Synced Fields (Phase 2)

| DB-Spalte | Quelle | Bemerkung |
|---|---|---|
| `eeg_logo_bytes` | `GET /cash/api/billingConfigs/{id}/logoImage` (Body) | BYTEA, NULL wenn nicht gesynct oder kein Logo im Core |
| `eeg_logo_mime` | Response `Content-Type` | TEXT (`image/png` etc.); NULL ⇒ kein Logo |
| `eeg_logo_synced_at` | `NOW()` beim erfolgreichen Sync | TIMESTAMPTZ; separat von `last_synced_from_core_at`, weil Logo-Sync best-effort und unabhängig fehlschlagen kann |

## Acceptance Criteria

### Stage A: DB Migration

- [ ] `db/migrations/000032_eeg_logo_sync.up.sql` fügt drei nullable Spalten an `registration_entrypoint`: `eeg_logo_bytes BYTEA`, `eeg_logo_mime TEXT`, `eeg_logo_synced_at TIMESTAMPTZ`
- [ ] `.down.sql` droppt die drei Spalten
- [ ] Bestehende `SELECT *`-Queries (falls vorhanden) sind durch explizite Spalten-Listen ersetzt, damit Standard-Reads die Logo-Bytes nicht ungewollt mitziehen

### Stage B: Core Client

- [ ] Neue Methode in `internal/coreclient/`: `FetchEEGLogo(ctx, bearerToken, tenant) (bytes []byte, mime string, err error)`
- [ ] Zwei-Schritt-Fetch:
  1. `GET {base}/cash/api/billingConfigs/tenant/{rcNumber}` → JSON parsen, Logo-Referenz extrahieren (genaues Response-Schema bei Implementierung gegen Live-System verifizieren — siehe Open Questions)
  2. `GET {base}/cash/api/billingConfigs/{id}/logoImage` → Bytes mit `io.LimitReader` 256 KB
- [ ] **Cap-Verletzung erkennen:** wenn der gelesene Body exakt 256 KB lang ist, wird das mit einem sprechenden Error (`ErrLogoTooLarge`) abgewiesen — nicht versucht, die Bytes als Bild zu interpretieren
- [ ] **MIME-Whitelist:** nur `image/png`, `image/jpeg`, `image/gif`; alles andere → `ErrLogoUnsupportedMime`
- [ ] **Spezial-Fall „kein Logo gesetzt":** 404 vom Billing-Service ⇒ `ErrLogoNotFound`, NICHT als Hard-Error nach oben weiterreichen (Sync soll trotzdem grün laufen)
- [ ] Unit-Tests für jeden Pfad (Success, TooLarge, UnsupportedMime, NotFound, HTTPError, Timeout)

### Stage C: Sync-Handler-Erweiterung

- [ ] `POST /api/admin/settings/eeg/sync` holt zusätzlich das Logo via `FetchEEGLogo` und schreibt `eeg_logo_bytes` + `eeg_logo_mime` + `eeg_logo_synced_at` mit
- [ ] **Logo-Fetch ist best-effort:** scheitert er, wird Stammdaten-Sync trotzdem committed; Response enthält ein neues optionales Feld `logoSyncWarning: string` mit der Fehlermeldung (z.B. „Logo zu groß: > 256 KB")
- [ ] Logo-Fetch wird auch übersprungen wenn Master-Data-Sync scheitert (kein halber Sync)
- [ ] Neuer Read-Endpoint: `GET /api/admin/settings/eeg/logo?rc_number={rc}` liefert das Logo aus der DB mit korrektem `Content-Type` und `Cache-Control: private, max-age=300` — damit das Frontend das Logo als `<img>` einbinden kann, ohne base64 in JSON zu schieben
- [ ] Endpoint ist Keycloak-protected und tenant-validiert (gleicher Mechanismus wie die anderen `/api/admin/settings/`-Endpoints)

### Stage D: PDF-Embedding

- [ ] In den beiden bestehenden PDF-Generatoren (`internal/pdf/approval.go` und `internal/pdf/sepa_mandate.go` — exakte Pfade bei Implementierung verifizieren) Logo top-right der ersten Seite einbetten
- [ ] Embed via `fpdf.RegisterImageReader(name, mimeType, bytes.NewReader(eegLogoBytes))` mit fester Höhe von **30 mm**, Breite proportional skaliert (max 50 mm, sonst proportional verkleinern)
- [ ] **Graceful Fallback:** wenn `eeg_logo_bytes` NULL ist → PDF rendert ohne Logo, kein Fehler
- [ ] **Fail-Safe:** wenn `RegisterImageReader` aus irgendeinem Grund failed (korruptes Bild, MIME-Mismatch) → Log-Warning, PDF rendert ohne Logo; PDF-Generation wirft niemals wegen Logo-Problemen ab
- [ ] Golden-Image-Tests für beide PDFs in zwei Varianten: mit Logo / ohne Logo

### Stage E: Admin-UI

- [ ] In der Stammdaten-Card (`src/components/admin-eeg-settings-editor.tsx`) zusätzlich zu den Text-Feldern eine kleine Logo-Vorschau (~80 px Breite, locked mit Schloss-Icon, identisches Styling wie die `SyncedField`s)
- [ ] Vorschau lädt via `GET /api/admin/settings/eeg/logo?rc_number=...` als `<img>`
- [ ] Wenn kein Logo gesynct → Placeholder „Noch kein Logo geladen" mit dezenter Border
- [ ] Falls `logoSyncWarning` in der Sync-Response gesetzt → Toast oder kleines orange-Banner unter der Logo-Vorschau („Logo konnte nicht aus eegFaktura geladen werden: <warning>")

### Stage F: Dokumentation

- [ ] `features/INDEX.md`: PROJ-33 als eigener Eintrag, Status flippt von Planned → ... → Deployed
- [ ] `docs/architecture.md` §3.5a: Logo-Spalten in der URL-Tabelle ergänzen (`/cash/api/billingConfigs/...`), Stammdaten-Sync-Beschreibung um „und Logo" erweitern
- [ ] `docs/api-spec.md`: §6.11b um `logoSyncWarning`-Feld in der Sync-Response ergänzen; neue §6.11c `GET /settings/eeg/logo`
- [ ] `docs/domain-model.md`: drei neue Spalten in der Core-mastered-Sektion
- [ ] `docs/user-guide/06-admin-settings.md`: Logo-Vorschau im Stammdaten-Block erwähnen, Hinweis „Logo wird aus eegFaktura übernommen — Änderung im Core, dann hier syncen"

## Open Questions

### Q1: Response-Schema des `billingConfigs/tenant/{rc}`-Endpoints?
Bei Implementierung gegen Live-System verifizieren. Mögliche Felder: `logoImageId`, `logoImage`, eine eingebettete URL, oder direkt der Logo-Body in einer base64-codierten Property. Ohne diese Info ist Stage B nicht final implementierbar — der User hat 2026-05-14 per Screenshot bestätigt, dass es einen `GET /cash/api/billingConfigs/{id}/logoImage`-Endpoint gibt, aber das Mapping vom `tenant/{rc}` zu der ID ist nicht dokumentiert.

**Empfohlene Vorgehensweise:** vor Stage B per DevTools/curl einen Beispiel-Request gegen `eegfaktura.at/cash/api/billingConfigs/tenant/TE100200` machen und die Response in einer Test-Fixture festhalten.

### Q2: Drift-Detection für das Logo?
Bei Text-Feldern (PROJ-32 Phase 1) zeigt die Settings-Page einen Drift-Banner („Stammdaten weichen ab"). Für das Logo wäre Drift-Detection nur via Byte-Vergleich machbar — teuer und nicht aussagekräftig (was sieht der Admin im Diff? Hex?).

**Empfehlung:** Logo bekommt **keinen Drift-Banner**. Stattdessen wird `eeg_logo_synced_at` als „Stand vom" angezeigt; ein Re-Sync ist immer ein expliziter Klick.

### Q3: Was passiert, wenn der Core ein neues Logo hat, der Admin aber nicht syncen kommt?
Erwartetes Verhalten: PDFs zeigen weiter das alte gecachte Logo, bis jemand „Aus eegFaktura aktualisieren" klickt. Konsistent zum Verhalten bei Text-Feldern.

### Q4: Cap auf den Sync-Endpoint selbst (Rate-Limit)?
Nicht in V1. Sync-Endpoint ist admin-only, kein Public Endpoint. Bei Bedarf später ergänzen.

## Out of Scope (für Phase 2)

- Server-Side-Resize des Logos
- Multiple Logo-Varianten (z.B. dark/light, square/wide)
- Drift-Detection auf Byte-Ebene
- Logo-Upload im Onboarding (Single Source of Truth bleibt der Core)
- Automatischer Refresh ohne Admin-Klick

## Pointer-Files

- Spec: `features/PROJ-33-eeg-logo-from-core.md` (diese Datei)
- Parent-Spec: `features/PROJ-32-eeg-master-data-from-core.md`
- Memory-Kontext: [[eegFaktura Core API contract]], [[eegfaktura-prod-url-prefixes]], [[proj32-phase1-handoff-2026-05-14]]
- Verwandte Code-Stellen (Phase 1, als Vorlage):
  - `internal/coreclient/eeg_master_data.go` — DTO + Fetch + Error-Sentinels
  - `internal/http/admin.go` — `SyncEEGSettingsFromCore` + Microcache
  - `internal/application/registration_entrypoint_repo.go` — `CoreMasterDataUpdate` + `SyncFromCore`
  - `src/components/admin-eeg-settings-editor.tsx` — `SyncedField` + Stammdaten-Card-Layout
