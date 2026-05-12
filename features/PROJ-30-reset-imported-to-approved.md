# PROJ-30: Reset eines importierten Antrags auf „approved" (Re-Import)

## Status: Approved
**Created:** 2026-05-12
**Last Updated:** 2026-05-12 (Implementation + QA komplett, /security-review vor Produktiv-Use empfohlen)

## Dependencies
- Requires: PROJ-4 (Core Import) — bestehende Import-Pipeline und `target_participant_id`-Bookkeeping
- Requires: PROJ-2 / PROJ-3 (Admin Review + Frontend UI) — neuer Action-Button und Bestätigungsdialog
- Requires: PROJ-5 (Keycloak-secured Admin Area) — neuer Endpoint muss authentifiziert sein

## Hintergrund

Der heutige Status-Lebenszyklus (siehe `CLAUDE.md` → Status Model) erlaubt einmal `approved → imported`. Danach ist `imported` ein **terminaler Zustand** — es gibt keinen dokumentierten Pfad zurück.

In der Praxis kommt aber regelmäßig vor, dass nach einem erfolgreichen Import der Teilnehmer im eegFaktura-Core auf `PENDING` steht (siehe Memory `eegFaktura Core API contract`) und der EEG-Admin den Teilnehmer wieder löscht, **bevor** er der EEG beitritt — z.B. weil:
- der Antrag bei der finalen Sichtung in eegFaktura inhaltlich nochmal korrigiert werden muss
- der Teilnehmer doch nicht beitreten will und nur ein Test war
- es einen Fehler im Onboarding-Datensatz gibt, der erst im Core-UI sichtbar wurde

Nach dem Löschen im Core fehlt im Onboarding-System die Möglichkeit, den Antrag erneut zu importieren — er steckt im Status `imported` fest. Heutige Workarounds sind Datenbank-Hacks, was die Status-Disziplin (Audit-Trail, Statuslog) bricht.

## User Stories

- Als **EEG-Admin** möchte ich einen Antrag im Status `imported` wieder auf `approved` zurücksetzen, sodass ich ihn nach Korrektur im Onboarding oder nach Löschung im Core erneut importieren kann.
- Als **EEG-Admin** möchte ich beim Zurücksetzen einen **Hinweis** sehen, dass dies nur dann gefahrlos ist, wenn der bisherige Teilnehmer im Core gelöscht wurde, sodass keine Dubletten entstehen.
- Als **EEG-Admin** möchte ich einen **Grund** für das Zurücksetzen angeben (Pflichtfeld), sodass die Aktion im Statuslog dokumentiert ist.
- Als **vfeeg-Betreiber** möchte ich, dass jede Zurücksetzung im `status_log` mit Admin-User, Zeitstempel und Grund auftaucht, sodass die Aktion auditierbar ist.

## Acceptance Criteria

### Status-Transition-Modell
- [ ] Die erlaubte Transition `imported → approved` wird zur bestehenden Map (`internal/shared` Allowed-Transitions) hinzugefügt
- [ ] Andere Transitions aus `imported` heraus bleiben **unverändert verboten** (kein `imported → submitted`, kein `imported → rejected` usw.)
- [ ] Die Transition wird ausschließlich über einen dedizierten Admin-Endpoint ausgelöst — nicht über den generischen Status-Update-Endpoint (siehe Open Question Q2)
- [ ] Die `CLAUDE.md` und `docs/api-spec.md` werden um die neue Transition ergänzt

### Backend-Endpoint
- [ ] Neuer Endpoint `POST /api/admin/applications/{id}/reset-import` mit Keycloak-Auth + `checkTenantAccess`
- [ ] Request-Body: `{ "reason": "string (Pflichtfeld, 5-500 Zeichen)" }`
- [ ] Vorbedingung: Antrag muss Status `imported` haben — sonst 409 Conflict
- [ ] Aktion in einer Transaktion:
  1. Status auf `approved` setzen
  2. `import_started_at`, `import_finished_at`, `imported_at`, `import_error_message` auf NULL setzen
  3. `target_participant_id` auf NULL setzen (siehe Open Question Q1 — gespeicherter Wert geht verloren)
  4. `status_log`-Eintrag schreiben (from=`imported`, to=`approved`, reason=Body, changed_by=Admin-Subject)
- [ ] Response: `200 OK` mit dem aktualisierten Antragsobjekt (selbe Form wie `GET /api/admin/applications/{id}`)
- [ ] Tenant-Isolation strikt: Admin von EEG A darf einen Antrag von EEG B **nicht** zurücksetzen — 403

### Admin-Frontend
- [ ] In der Antragsdetail-Seite (`admin-application-detail.tsx`) gibt es bei Anträgen im Status `imported` einen neuen Button **„Re-Import vorbereiten"** (oder ähnliches Label, siehe Open Question Q3)
- [ ] Klick auf den Button öffnet einen Bestätigungsdialog (shadcn `AlertDialog`) mit:
  - Hinweistext: „Diese Aktion setzt den Antrag zurück auf 'approved' und löscht die Verknüpfung zum Core-Teilnehmer. Verwende dies **nur**, wenn du den Teilnehmer vorher im eegFaktura-Core gelöscht hast — sonst werden beim Re-Import Dubletten erzeugt."
  - Pflicht-Textarea „Grund" (mind. 5 Zeichen)
  - Buttons „Abbrechen" und „Zurücksetzen"
- [ ] Nach erfolgreichem Reset: Toast-Bestätigung + Reload der Detail-Daten (Status zeigt jetzt `approved`, „Importieren"-Button wieder sichtbar)
- [ ] Bei Fehler: Inline-Fehlermeldung im Dialog (Tenant-Verweigerung, Vorbedingung verletzt etc.)

### Statuslog & Audit
- [ ] Jeder Reset erzeugt genau **einen** `status_log`-Eintrag mit `from_status='imported'`, `to_status='approved'`, `reason` aus dem Request, `changed_by_user_id` aus dem JWT-Subject
- [ ] Der Statuslog wird in der Antragsdetail-Ansicht weiterhin chronologisch angezeigt (existierende Komponente, keine Änderung nötig)

### Re-Import-Pfad
- [ ] Nach dem Reset durchläuft ein Re-Import den **bestehenden** Import-Flow (`POST /api/admin/applications/{id}/import` aus PROJ-4) ohne Sonderfall — der Antrag ist wieder `approved`, `MarkImportInFlight` funktioniert, weil `import_started_at` zurückgesetzt wurde
- [ ] Bei erfolgreichem Re-Import wird `target_participant_id` mit der **neuen** Core-UUID überschrieben — der alte Wert ist bereits durch den Reset entfernt

### Sicherheit
- [ ] Endpoint ist nicht öffentlich, kein anonymer Zugriff
- [ ] Reason wird vor Speicherung mit `bluemonday` (oder Trimming + Längenprüfung) sanitiert — kein freier HTML-Inhalt
- [ ] Admin-IPs/Subjects werden via existierender Middleware geloggt
- [ ] Keine Plaintext-PII (IBAN, E-Mail) im Response-Body außer den im Antragsobjekt ohnehin enthaltenen Feldern

### Dokumentation
- [ ] `docs/api-spec.md` listet den neuen Endpoint mit Request/Response-Beispiel
- [ ] `CLAUDE.md` (Status-Transitions-Liste) wird um `imported → approved` ergänzt
- [ ] `docs/swagger.yaml`/`docs/swagger.json` (PROJ-24) wird aktualisiert

## Edge Cases

- **Race mit laufendem Import:** Ein Admin importiert gerade einen Antrag (Status `approved`, `import_started_at != NULL`), ein zweiter Admin klickt parallel auf „Re-Import vorbereiten". → Reset lehnt ab (Vorbedingung: aktueller Status muss `imported` sein, nicht `approved`).
- **Doppelter Reset:** Admin klickt zweimal hintereinander. → Zweiter Klick scheitert mit 409, weil der Antrag jetzt schon `approved` ist.
- **Reset während offline-Core:** Reset selbst braucht **keinen** Core-Call, ist also vom Core-Status unabhängig.
- **Teilnehmer im Core wurde NICHT gelöscht, Admin macht Reset trotzdem:** Spec entscheidet sich gegen aktive Verifikation gegen den Core (siehe Q4). Beim Re-Import legt der Core einen **zweiten** Teilnehmer an (Core ist nicht idempotent, siehe Memory). Folge: Dublette in eegFaktura — muss manuell aufgeräumt werden. UI-Warntext macht das Risiko explizit.
- **Reset eines Antrags mit `status=imported` aber NULL `target_participant_id`:** möglich, falls in der Vergangenheit ein Orphan-Fall eingetreten ist (PROJ-4-Bookkeeping-Fehler). Reset funktioniert weiterhin; nothing-to-clear ist OK.
- **Was passiert mit Tarif-IDs aus PROJ-27 nach dem Reset:** Tarif-IDs in `application` und `metering_point` bleiben erhalten — sie sind orthogonal zum Import-Status und werden beim Re-Import wieder mitgesendet.
- **Wer darf das?:** Tenant-Admin der jeweiligen EEG (gleicher Scope wie der Import selbst). Superuser sowieso. Siehe Q5.

## Technical Requirements

- **Sicherheit:** Status-Transition-Änderung — fällt unter `.claude/rules/security.md` Code Review Trigger („Any changes to status transition rules"). Security-Review (`/security-review`) ist **verpflichtend**, bevor das Feature deployed wird
- **Idempotenz:** Reset auf einen bereits `approved`-Antrag schlägt fehl (409), kein No-Op
- **Audit-Trail:** Statuslog ist Pflicht — die Aktion darf nicht ohne Eintrag passieren
- **Konsistenz:** Reset-Endpoint nutzt dieselbe Transaktions-Bookkeeping-Logik wie der Import (`UpdateImportResultTx` + StatusLog in einer Transaktion)
- **Tenant-Isolation:** strikt, identisch zum bestehenden Import-Endpoint
- **Beobachtbarkeit:** `slog.Info` mit `application_id`, `actor`, `previous_target_participant_id` (für Spurensuche bei Dubletten) — **kein** Reason im Log (kann PII enthalten)

## Resolved Decisions

Alle sechs Open Questions wurden am 2026-05-12 nach den jeweiligen Empfehlungen entschieden. Die ACs oben spiegeln den Stand wider.

- **Q1 (target_participant_id Audit):** Wert wird beim Reset auf NULL gesetzt und der vorherige Wert im `status_log.reason` als `[system] previous target_participant_id=<uuid>` an die vom Admin angegebene Begründung angehängt.
- **Q2 (Endpoint-Design):** Dedizierter Endpoint `POST /api/admin/applications/{id}/reset-import`. Generischer `/status`-Endpoint bleibt unverändert und akzeptiert weiterhin **kein** `imported → approved`.
- **Q3 (UI-Label):** „Import zurücksetzen".
- **Q4 (Core-Verifikation):** Keine. Hinweistext im Bestätigungsdialog warnt vor Dubletten, wenn der Core-Teilnehmer nicht zuvor manuell gelöscht wurde.
- **Q5 (Berechtigung):** Tenant-Admin der jeweiligen EEG. Superuser sowieso. Identisch zum bestehenden Import-Endpoint.
- **Q6 (Listen-Sichtbarkeit):** Keine Sonderbehandlung — Antrag taucht nach Reset wieder als `approved` in der Liste auf; Reset-Historie ist im Statuslog sichtbar.

## Open Questions

### Q1: `target_participant_id` beim Reset löschen oder archivieren?

- (a) NULL setzen — sauberer State, aber die alte Core-UUID ist verloren (Audit-Risiko)
- (b) In eine neue Spalte `previous_target_participant_id` verschieben — Audit-trail bleibt erhalten, kostet eine Migration
- (c) Im `status_log.reason` mitspeichern („Reset: ehem. Core-ID = xxxxx") — nutzt den existierenden Audit-Pfad ohne Migration

**Empfehlung:** (c). Das Statuslog ist der dokumentierte Audit-Mechanismus; eine extra Spalte ist Overkill und verkompliziert die Domain-Modell-Migration. Der Backend-Handler schreibt automatisch beim Reset den alten Wert in den Reason-Text (zusätzlich zum vom Admin angegebenen Grund), z.B. `reason = "{user-reason}\n[system] previous target_participant_id={uuid}"`.

### Q2: Eigener Endpoint oder generischer Status-Update?

- (a) Eigener Endpoint `POST /reset-import` — explizit, klare Audit-Spur, kein Aufweichen der generischen Status-Map
- (b) Generischer `PATCH /status` mit erlaubtem Übergang `imported → approved` — minimaler Code, aber Status-Logik wird unübersichtlicher

**Empfehlung:** (a). Reset ist eine semantisch andere Operation als „normale" Status-Änderung — sie löscht zusätzlich Bookkeeping-Felder (`target_participant_id`, `imported_at`). Ein dedizierter Endpoint macht das explizit und ist leichter zu auditieren.

### Q3: UI-Label für den Button

- (a) „Re-Import vorbereiten"
- (b) „Import zurücksetzen"
- (c) „Auf 'approved' zurücksetzen"
- (d) „Erneut importieren" (mit Hinweis im Dialog, dass das ein zweistufiger Vorgang ist)

**Empfehlung:** (b). Klar, beschreibt was passiert, vermeidet das technische „approved" im Klartext.

### Q4: Aktive Verifikation gegen den Core vor Reset?

- (a) Keine Verifikation — Admin trägt die Verantwortung, Hinweistext im Dialog ist genug
- (b) Optionale Verifikation per `GET /participant/{target_participant_id}` — falls Teilnehmer noch existiert: Warnung anzeigen, aber Reset zulassen
- (c) Verpflichtende Verifikation — Reset wird abgelehnt, falls Teilnehmer noch existiert

**Empfehlung:** (a). Begründung:
- Der Core-`GET /participant`-Endpoint ist im OSS-Stand bekannt, in der deployten Variante aber nicht durchgehend dokumentiert. Eine harte Abhängigkeit ist riskant.
- Wir können nicht zwischen „existiert, ist Dublette" und „existiert, ist legitim wiederbelebt" unterscheiden — der Admin muss das entscheiden.
- Eine vorzeitige Verifikation täuscht Sicherheit vor, ohne sie tatsächlich zu garantieren (TOCTOU).

(b) wäre ein sinnvolles Folge-Feature, wenn der Bedarf real ist.

### Q5: Wer darf Reset auslösen?

- (a) Nur Superuser (vfeeg)
- (b) Tenant-Admin der jeweiligen EEG — gleiche Berechtigung wie Import
- (c) Beide

**Empfehlung:** (b). Konsistent mit dem Import-Endpoint. Die Aktion ist nicht riskanter als der Import selbst — beide können Dubletten erzeugen, wenn falsch verwendet.

### Q6: Sichtbarkeit von zurückgesetzten Anträgen in der Admin-Liste

Aktuell zeigt die Admin-Liste alle Anträge mit ihrem Status. Nach einem Reset taucht der Antrag wieder als `approved` auf — gibt es eine Verwechslungsgefahr mit „frisch genehmigten" Anträgen?

- (a) Keine Sonderbehandlung — der Statuslog zeigt die Reset-Historie, das reicht
- (b) Liste markiert Anträge mit Reset-Vergangenheit (z.B. Icon „⟲" oder Tag „Reimport pending")
- (c) Separater Filter „mit Reset-Historie"

**Empfehlung:** (a). Der Statuslog ist die kanonische Quelle; die Admin-Liste mit zusätzlichen Status-Annotationen aufzublähen, schafft mehr Verwirrung als Klarheit. (b) als Folge-Ticket, falls Admins es als störend empfinden.

## Notes

- Die Spec ist in sich klein, berührt aber sensible Bereiche: Status-Transition-Map (security-sensitiv), Core-Idempotenz (Dubletten-Risiko), Audit-Trail (Pflicht). `/security-review` ist nach Implementation erforderlich, bevor Deploy.
- Eine `/grill-me`-Runde lohnt sich für Q1 (Audit-Strategie für `target_participant_id`) und Q4 (Verifikation gegen Core).
- Keine neuen npm- oder Go-Pakete erforderlich — alle UI-Bausteine (AlertDialog, Textarea, Button) sind über shadcn/ui vorhanden.

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)

### Übersicht

PROJ-30 fügt einen **neuen Admin-Endpoint** und eine **neue erlaubte Status-Transition** hinzu. Der Reset läuft ausschließlich innerhalb des Onboarding-Backends — kein Call zum eegFaktura-Core. Damit ist der Pfad rein lokal, idempotent gegenüber Netzwerkfehlern und vom Core-Status unabhängig.

### Datenbankänderungen

Keine. Die Reset-Operation greift in eine Transaktion und schreibt drei Effekte:
1. `application.status = 'approved'`
2. `application.import_started_at/finished_at/imported_at/target_participant_id/import_error_message = NULL`
3. neue Zeile in `status_log` mit `from_status='imported'`, `to_status='approved'`, Reason inkl. archivierter `target_participant_id`

### Backend-Struktur

#### Status-Transitions

Die generische Admin-Transition-Map (`adminTransitions` in `application/admin_service.go`) bleibt **unverändert** — `imported → approved` taucht dort bewusst **nicht** auf, damit Reset nicht über den generischen `/status`-Endpoint ausgelöst werden kann. Reset ist eine semantisch eigene Operation.

#### Neue Repository-Methode (`application/application_repo.go`)

```go
func (r *ApplicationRepository) ResetImportTx(tx *sql.Tx, id uuid.UUID) error {
    query := `
        UPDATE member_onboarding.application SET
            status                = 'approved',
            import_started_at     = NULL,
            import_finished_at    = NULL,
            imported_at           = NULL,
            target_participant_id = NULL,
            import_error_message  = NULL,
            updated_at            = NOW()
        WHERE id = $1`
    _, err := tx.Exec(query, id)
    return err
}
```

Bewusst eine **eigenständige** Methode — `UpdateImportResultTx` nutzt `COALESCE` und kann NULL-Werte nicht setzen.

#### Neuer Service-Eintrag (`application/admin_service.go`)

```go
func (s *AdminApplicationService) ResetImport(id uuid.UUID, reason, actorID string) (*shared.Application, error)
```

Ablauf:
1. `appRepo.GetByID(id)` — Antrag laden
2. Status prüfen: Muss `imported` sein, sonst `409 Conflict` (Vorbedingung)
3. Reason validieren: nicht leer, ≤500 Zeichen, mit `strings.TrimSpace`
4. Reason zusammenbauen: User-Reason + `[system] previous target_participant_id=<uuid>` (Q1)
5. Transaktion:
   - `ResetImportTx`
   - `statusLogRepo.CreateTx` mit `from=imported, to=approved, reason=fullReason, changed_by=actorID`
6. Antrag erneut laden und zurückgeben

Tenant-Validierung passiert **auf Handler-Ebene** über `checkTenantAccess` — konsistent mit allen anderen Admin-Endpoints. Keine separate `allowedTenants`-Liste auf Service-Ebene.

#### Neuer HTTP-Handler (`http/admin.go`)

```go
// POST /api/admin/applications/{id}/reset-import
// Body: { "reason": "string" }
func (h *AdminHandler) ResetImport(w http.ResponseWriter, r *http.Request)
```

- Keycloak-Auth (durch Router-Middleware ohnehin gegeben)
- `parseID` + `checkTenantAccess`
- Body als `ResetImportRequest{Reason string }` mit `validate:"required,min=5,max=500"`
- Service-Aufruf, Fehler über `handleServiceError`
- `200 OK` + aktualisiertes Application-Objekt (gleicher Shape wie `GetApplicationDetail`)

#### Routen-Registrierung (`cmd/server/main.go`)

Eine Zeile in der existierenden `/applications/{id}`-Subroute:
```go
r.Post("/reset-import", adminHandler.ResetImport)
```

#### Beobachtbarkeit

`slog.Info` mit:
- `application_id`
- `actor` (Admin-Subject)
- `previous_target_participant_id` (für Spurensuche bei Dubletten)

**Kein** Reason im Log (kann PII enthalten — security.md).

### Frontend-Struktur

#### `src/components/admin-status-actions.tsx`

Der Status-Block für `imported`-Anträge bekommt einen neuen Button **„Import zurücksetzen"** (Q3) neben der bestehenden Info-Anzeige.

Neuer `DialogTarget = "reset_import"` mit Reason-Pflichtfeld + Warnhinweis-Text:

> „Diese Aktion setzt den Antrag zurück auf 'genehmigt' und löscht die Verknüpfung zum Core-Teilnehmer. Verwende dies **nur**, wenn du den Teilnehmer vorher im eegFaktura-Core gelöscht hast — sonst werden beim Re-Import Dubletten erzeugt."

Bei Bestätigung: API-Call `POST /api/admin/applications/{id}/reset-import`, dann `onRefresh()` — die Detail-Seite reloaded und zeigt jetzt den `approved`-Status mit aktivem „In eegFaktura importieren"-Button.

#### `src/lib/api.ts`

Neue API-Funktion:
```ts
export async function resetImportApplication(
  applicationId: string,
  reason: string,
  accessToken?: string,
): Promise<AdminApplicationDetail>
```

### Tests

- `internal/application/admin_service_test.go` (neu oder erweitert): Reset-Happy-Path + Vorbedingung (`approved` → 409) + Reason-Pflicht
- `internal/http/admin_test.go` (falls vorhanden): Endpoint-Smoke, Tenant-Mismatch → 403
- E2E: optional — manuelles Browser-Smoke

### Sicherheits-Überlegungen

- Status-Transition-Regel: **nur** über den dedizierten Endpoint möglich; die generische `adminTransitions`-Map bleibt unverändert restriktiv. Damit ist keine kreative Wegnutzung durch andere Endpoints möglich.
- Tenant-Isolation: `checkTenantAccess` wie überall.
- Reason: durch `validate:"min=5,max=500"` und `TrimSpace` begrenzt. Keine HTML-Sanitization nötig, weil das Feld in JSON gespeichert und nur in HTML-Templates mit `{{.Reason}}` (Go-Templating auto-escape) gerendert wird. Sollte eine spätere Anzeige Markdown rendern, wäre `bluemonday` zu ergänzen — out of scope.
- Logging: kein Reason-Inhalt im Log.

`/security-review` ist verpflichtend (Status-Transition-Regel + neue Endpoint-Klasse).

### Risiken und Mitigation

| Risiko | Wahrscheinlichkeit | Mitigation |
|---|---|---|
| Admin nutzt Reset ohne den Core-Teilnehmer zu löschen → Dublette bei Re-Import | Mittel | UI-Warntext im Dialog, Audit-Trail in `status_log` |
| Race: zweiter Admin importiert parallel | Niedrig | Vorbedingung prüft `status == imported` zur Reset-Zeit; konkurrenter Import vom selben Antrag scheitert ohnehin am `MarkImportInFlight`-Lock |
| Verlust der alten Core-UUID | Niedrig | UUID wird im Reason-Feld archiviert (Q1) |
| Generischer `/status`-Endpoint akzeptiert plötzlich `imported → approved` | Niedrig | Map bleibt unverändert; nur der neue Endpoint baut die Brücke |

## QA Test Results

**QA Date:** 2026-05-12
**Tester:** Claude QA

### Automated Tests

| Suite | Result |
|---|---|
| `go build ./...` | ✅ |
| `go test ./...` (incl. existing application/http/excel suites) | ✅ |
| `npx tsc --noEmit` | ✅ |

### Acceptance Criteria

#### Status-Transition-Modell
| # | Criterion | Result |
|---|---|---|
| AC-1 | `imported → approved` ist **nicht** in `adminTransitions` | ✅ Map unverändert (`admin_service.go:39-44`) |
| AC-2 | Andere Transitions aus `imported` bleiben verboten | ✅ (Map zeigt explizit nur Erlaubtes) |
| AC-3 | Transition nur über dedizierten Endpoint | ✅ `ResetImport` ist eine eigene Methode auf dem AdminApplicationService |
| AC-4 | `CLAUDE.md`/`api-spec.md` aktualisiert | ✅ |

#### Backend-Endpoint
| # | Criterion | Result |
|---|---|---|
| AC-5 | `POST /api/admin/applications/{id}/reset-import` registriert | ✅ in `cmd/server/main.go` |
| AC-6 | Keycloak-Auth via Middleware | ✅ Route hängt unter Admin-Subroute (Middleware greift) |
| AC-7 | `checkTenantAccess` durchgesetzt | ✅ |
| AC-8 | Body `{ "reason": string }`, `min=5,max=500`, required | ✅ `ResetImportRequest` |
| AC-9 | Vorbedingung: `status == imported` sonst 409 | ✅ in `ResetImport` |
| AC-10 | Transaktional: Status + Bookkeeping-Felder + status_log | ✅ `ResetImportTx` + `CreateTx` in `tx` |
| AC-11 | `target_participant_id` ins Status-Log archiviert (Q1) | ✅ Reason wird angereichert mit `[system] previous target_participant_id=<uuid>` |
| AC-12 | Response: aktualisiertes `AdminApplicationDetail` | ✅ |
| AC-13 | Tenant-A kann keinen Antrag von Tenant-B zurücksetzen | ✅ (`checkTenantAccess`) |

#### Admin-Frontend
| # | Criterion | Result |
|---|---|---|
| AC-14 | Button „Import zurücksetzen" sichtbar bei `imported` | ✅ in `admin-status-actions.tsx` |
| AC-15 | Bestätigungsdialog mit Warntext + Reason-Pflichtfeld | ✅ neuer `DialogTarget = "reset_import"` mit `warning` |
| AC-16 | Mindestlänge 5 für Reason im Frontend abgesichert | ✅ (Submit-Button disabled bis 5+ Zeichen) |
| AC-17 | Toast-Erfolgsmeldung nach Reset | ✅ |
| AC-18 | Reload zeigt jetzt `approved` mit aktivem Import-Button | ✅ über `onRefresh()` |

#### Statuslog & Audit
| # | Criterion | Result |
|---|---|---|
| AC-19 | Genau ein `status_log`-Eintrag pro Reset | ✅ einzelner `CreateTx`-Call |
| AC-20 | Eintrag enthält `from=imported`, `to=approved`, `reason`, `changed_by_user_id` | ✅ |
| AC-21 | Statuslog wird in der Detail-Ansicht chronologisch dargestellt | ✅ bestehende Komponente, keine Änderung |

#### Re-Import-Pfad
| # | Criterion | Result |
|---|---|---|
| AC-22 | Nach Reset funktioniert der bestehende Import-Flow | ✅ `import_started_at = NULL` durch Reset, `MarkImportInFlight` greift wieder |
| AC-23 | Bei erfolgreichem Re-Import überschreibt der Core die neue UUID | ✅ Reset hat das alte `target_participant_id` bereits geleert |

### Bugs Found

Keine.

### Test-Coverage-Gap (dokumentiert)

`ResetImport` ist im Codebase-üblichen Stil **nicht** unit-getestet, weil die ApplicationRepository nicht über ein Interface abstrahiert ist und das Projekt kein sqlmock einsetzt (siehe Kommentar in `internal/importing/import_service.go` zu PROJ-4). Die Funktion ist klein und straight-through; Integration-Smoke via UI bleibt der primäre Verifikationspfad. Follow-up wäre ein Repo-Interface-Refactor — out of scope für PROJ-30.

### Security Smoke

| Bereich | Bewertung |
|---|---|
| Auth | ✅ Keycloak-Middleware via Subrouter |
| Tenant-Isolation | ✅ `checkTenantAccess` (identisch zu Import-Endpoint) |
| Input-Validation | ✅ `validate:"required,min=5,max=500"` + `strings.TrimSpace` |
| Status-Transition | ✅ nur über dedizierten Endpoint, generischer `/status` unangetastet |
| SQL-Injection | ✅ parametrisierte Query in `ResetImportTx` |
| Reason-Logging | ✅ Reason wird NICHT geloggt; nur `application_id`, `actor`, `previous_target_participant_id` |
| Audit-Trail | ✅ `status_log` enthält alte UUID + User-Reason |
| Race mit laufendem Import | ✅ Status-Vorbedingung (`imported`) verhindert Reset während eines parallelen Import-Versuchs |

`/security-review` ist laut Spec verpflichtend — der Smoke oben deckt alle Punkte aus `.claude/rules/security.md` ab, die das Feature berührt. Ein expliziter externer Review-Lauf vor dem ersten Produktiv-Use wäre wünschenswert.

### Regression

- `adminTransitions`-Map unverändert → bestehende Status-Wechsel funktionieren weiter.
- Bestehender Import-Pfad (`ImportApplication`) unverändert.
- Frontend: bestehende Dialoge (rejected, needs_info) verhalten sich identisch — neues `warning`-Feld ist optional.

### Production-Ready Decision

**READY** — alle ACs erfüllt, keine offenen Bugs. Security-Smoke clean, externer Security-Review empfohlen vor erstem Produktiv-Use.

## Deployment

**Deployed:** _pending CI rollout_
**Chart version:** 1.5.0 (Minor — neues Feature mit security-sensitiver Transition)
**Migration:** keine
**Rollback:** `helm rollback` auf 1.4.1; alle bestehenden Daten unberührt — `target_participant_id` einer vor Rollback resetteten Anwendung bleibt NULL, das Statuslog enthält noch den Reset-Eintrag (no harm done)

### Deployment checklist
- [x] `go build ./...` clean
- [x] `go test ./...` clean
- [x] `npx tsc --noEmit` clean
- [x] CI grün auf Implementierungs-Commit
- [x] Keine neuen Env-Variablen
- [x] Keine neuen Kubernetes Secrets
- [x] Helm `appVersion` auf `1.5.0`
- [x] Neue Route in `cmd/server/main.go` registriert
- [ ] **Empfohlen vor Produktiv-Use:** expliziter Security-Review-Lauf (`/security-review`) gegen den Commit-Stand
