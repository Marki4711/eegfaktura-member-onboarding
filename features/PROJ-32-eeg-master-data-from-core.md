# PROJ-32: EEG-Stammdaten aus eegFaktura-Core beziehen (inkl. Logo)

## Status: Planned
**Created:** 2026-05-14
**Last Updated:** 2026-05-14

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

- [ ] Neuer Methode `(c *HTTPCoreClient) GetEEG(ctx, bearerToken, tenant) (*CoreEEG, error)` analog zu `GetTariffs`. Trifft `GET <CORE_BASE_URL>/eeg` mit Bearer-Auth und `tenant`-Header.
- [ ] Neuer DTO-Typ `CoreEEG` in `internal/coreclient/` mit den Feldern aus dem Core-Response (Name, Adresse-Komponenten, Creditor-ID, Contact-E-Mail, Logo, …).
- [ ] Logo-Repräsentation: zur Implementierungszeit gegen die deployte API verifizieren (siehe Q1). Wahrscheinliche Optionen:
  - (a) `logoUrl string` → absoluter Pfad in den Filestore-Service
  - (b) `logo string` → base64-encoded Bytes inkl. MIME-Typ-Prefix
  - (c) Logo lebt in eigenem Endpoint `GET /eeg/logo`
- [ ] Fehlerbehandlung: Core 404 → `shared.ErrNotFound`, alle anderen Fehler werden gewrappt und gemeldet.

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

## Open Questions

### Q1: Wie repräsentiert die deployte Core-API das EEG-Logo?
- (a) Als `logoUrl` zu einem internen Filestore-Service?
- (b) Als base64-encoded Bytes im JSON-Response von `GET /eeg`?
- (c) Über einen separaten Endpoint `GET /eeg/logo`?

**Empfehlung:** Vor der Implementierung **einmalig** gegen die deployte API testen (z.B. `curl` mit gültigem Bearer-Token). Spec macht keinen blinden Vorgriff — die drei Varianten sind im Code abstrahiert hinter dem `CoreEEG`-DTO + `EmbedEEGLogo`-Funktion, sodass jede Variante mit ~10 LOC Anpassung passt.

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

### Q8: ⚠️ Auth-Knoten — wer hat ein Bearer-Token beim PDF-Render?

Das ist die größte offene Frage. Heute funktioniert es so:
- `POST /participant` (Import) braucht Bearer aus dem Admin-JWT → ist verfügbar, weil ein Admin im Browser triggert.
- PDF-Render beim **Submit** (Member ohne Auth) → kein Admin-JWT verfügbar.

Für `GET /eeg` braucht der Core einen Bearer + tenant. Möglichkeiten:
- (a) Dedizierter **Service-Account** im Core (eigene Credentials für Onboarding-Backend). Onboarding hält die Credentials in einer K8s-Secret. ✓ Sauberster Weg.
- (b) Onboarding bekommt einen Bypass via spezielles `X-Internal-Auth`-Header. Erfordert Core-Änderung.
- (c) `GET /eeg` wird unauthentifiziert verfügbar gemacht. Weichgespült, weil EEG-Stammdaten nicht hochsensibel sind. Erfordert Core-Änderung.

**Empfehlung:** (a). Pure Onboarding-Änderung möglich, sobald der vfeeg-Betreiber im Core einen Service-Account anlegt. Eintragen in `.env.local.example` als `CORE_SERVICE_ACCOUNT_TOKEN` (Long-lived Bearer).

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
