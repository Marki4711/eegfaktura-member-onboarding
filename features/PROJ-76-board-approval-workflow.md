# PROJ-76: Vorstands-Genehmigungs-Workflow für Beitrittserklärung

## Status: In Review (Backend + Frontend implementiert, Build/TSC grün, Tests grün)
**Created:** 2026-06-07
**Last Updated:** 2026-06-07

## Dependencies
- Erfordert: PROJ-46 / PROJ-53 (`activated`-Übergang als Mail-Trigger) — Deployed.
- Erfordert: PROJ-21 (bestehender Beitrittsbestätigungs-PDF-Renderer) — Deployed; dient als Vorlage für den neuen Beitrittserklärungs-Renderer.
- Optional: PROJ-32-Erweiterung (`eegContactPerson`) — falls in einer späteren Iteration der Vorstand-Name vorab eingedruckt werden soll. Im MVP nicht benötigt.
- **Supersedes:** PROJ-65 (Vorstands-Signaturblock im Beitrittsbestätigungs-PDF). PROJ-65 hätte nur den Signaturblock ergänzt; PROJ-76 deckt das ab und ersetzt zusätzlich das Mail-Routing.

## Hintergrund

Heute geht beim Status-Übergang `→ activated` die Beitrittsbestätigung als PDF automatisch an das Mitglied (mit EEG-Kopie). Manche EEGs wünschen einen anderen Workflow — ihre Statuten oder ihre Praxis sehen vor, dass der Vorstand den Beitritt **formal genehmigt** und das unterschriebene Dokument selbst ans Mitglied weiterleitet.

Owner-Feature-Request 2026-06-07: per-EEG-Toggle, der genau diesen Workflow umschaltet — bei aktivem Toggle entfällt die automatische Mitglieder-Mail komplett, stattdessen geht eine **Beitrittserklärung** mit Vorstands-Unterschriftslinie an den EEG-Kontakt. Vorstand unterschreibt manuell und leitet weiter.

PROJ-65 wäre nur eine PDF-Layout-Erweiterung gewesen (Signaturblock im normalen PDF, Mail-Flow bleibt). PROJ-76 ist die echte Workflow-Lösung — neues PDF + Mail-Routing-Wechsel.

## User Stories

- Als **EEG-Vorstand** möchte ich beim Mitgliederbeitritt eine vorgefertigte Beitrittserklärung erhalten, die ich nur noch unterschreiben und ans Mitglied weiterleiten muss — damit ich den statuten-konformen Beschluss-Workflow ohne manuelles Tippen einhalten kann.
- Als **EEG-Admin** möchte ich pro EEG umschalten können, ob die Mitgliederaktivierung im Auto-Modus (Mitglied bekommt die Bestätigung sofort) oder im Vorstands-Modus (Vorstand genehmigt und leitet weiter) abläuft.
- Als **EEG-Admin** möchte ich, wenn der Vorstand das Dokument verlegt, die Beitrittserklärung im Admin-UI herunterladen können, um sie selbst weiterzuleiten — ohne den Antrag zurücksetzen oder eine erneute Mail anstoßen zu müssen.
- Als **EEG-Admin** möchte ich im Antrags-Detail sehen, dass die Beitrittserklärung an den EEG-Kontakt verschickt wurde, damit ich weiß, dass der manuelle Weiterleitungsschritt noch aussteht.
- Als **Owner** möchte ich verhindern, dass die Beitrittserklärung mehrfach automatisch verschickt wird, ohne dem EEG-Admin den Resend-Pfad zu blockieren.

## Akzeptanzkriterien

### EEG-Setting

- [ ] Neue Spalte `board_approval_workflow_enabled BOOLEAN NOT NULL DEFAULT FALSE` in `registration_entrypoint`. Bestehende EEGs bleiben auf `FALSE` (heutiges Verhalten).
- [ ] Settings-UI: neuer Toggle mit dem Label „**Beitrittserklärung vom Vorstand genehmigen lassen, statt Beitrittsbestätigung automatisch zu versenden**" in `admin-eeg-settings-editor.tsx` unterhalb der bestehenden SEPA-Toggles. Mit Hint-Popover (Pattern aus `.claude/rules/frontend.md`) der das Modell erklärt: „Wenn aktiv, geht beim Wechsel auf ‚Aktiviert' eine Beitrittserklärung an den EEG-Kontakt statt einer Beitrittsbestätigung über unsere Plattform. Der Vorstand unterschreibt und leitet manuell weiter. Das Mitglied wird über die reguläre eegFaktura-Aktivierungs-Mail vom Core informiert."
- [ ] Toggle ist Teil des Configuration-Exports (PROJ-61): Schema-Feld als `*bool, omitempty` (Legacy-Backups ohne das Feld parsen weiter sauber, Default beim Import = FALSE), Exporter, Importer und Diff schreiben das Feld mit.
- [ ] Toggle ist Teil des PROJ-67-Awareness-Triggers (`isAdvancedEEGSettingsActive`) — beim Wechsel auf Standard-Modus zeigt der Banner an, dass dieser nicht-Default-Workflow aktiv ist.
- [ ] HTTP-Endpoint `PUT /api/admin/settings/eeg` akzeptiert das Feld; GET-Response liefert es mit.

### Neue DB-Spalte: separater Mehrfach-Versand-Schutz

- [ ] Neue Spalte `board_declaration_sent_at TIMESTAMPTZ NULL` auf `application`. Markiert, ob die Beitrittserklärung an den EEG-Kontakt verschickt wurde — eigene Spalte statt Recycling von `activation_notification_sent_at`, weil semantisch zwei unterschiedliche Mail-Events.
- [ ] Bei `board_approval_workflow_enabled=TRUE` wird `board_declaration_sent_at` gesetzt; `activation_notification_sent_at` bleibt NULL (wird nicht gemischt).
- [ ] Bei `board_approval_workflow_enabled=FALSE` (Auto-Modus) bleibt `board_declaration_sent_at` NULL; `activation_notification_sent_at` wird wie heute gesetzt.
- [ ] Migration: `ALTER TABLE application ADD COLUMN board_declaration_sent_at TIMESTAMPTZ NULL`. Non-blocking auf PostgreSQL 11+.

### PDF-Architektur: Single Renderer mit Variante

- [ ] **Kein eigener PDF-Renderer** — der bestehende `internal/pdf/approval_pdf.go` wird um einen Variant-Parameter erweitert: `GenerateApproval(data ApprovalPDFData, variant Variant)` mit `Variant ∈ {VariantBeitrittsbestätigung, VariantBeitrittserklärung}`. Begründung: ~90% des Renderers (Mitgliedsdaten, Zählpunkte, Zustimmungen, Genossenschaftsanteile, Netzbetreiber-Info-Seite) sind identisch — gemeinsamer Helper-Pfad vermeidet das parallel-struct-Drift-Pattern (Memory `feedback_shared_helpers_for_parallel_paths`).
- [ ] Variante steuert nur die zwei Unterschiede:
  - **Header-Titel:** „Beitrittsbestätigung" (Auto-Modus) vs. „Beitrittserklärung" (Vorstands-Modus).
  - **Vorstands-Signaturblock** am Ende der Hauptseite (nur bei `VariantBeitrittserklärung`, vor der ggf. folgenden Netzbetreiber-Info-Seite):
    - Überschrift „Genehmigung durch den Vorstand"
    - Eine Datum/Ort/Unterschrift-Linie, generisch beschriftet als „Vorstand"
    - Linien bleiben leer — kein Name, kein Datum vorgedruckt
- [ ] Dateiname je nach Variante: `Beitrittserklaerung-<Antragsnummer>.pdf` bzw. `Beitrittsbestaetigung-<Antragsnummer>.pdf`.
- [ ] EEG-Logo, Footer-Branding und Layout-Konventionen identisch zwischen beiden Varianten.
- [ ] **Generierung on-demand, kein Persistieren als BYTEA-Blob.** Sowohl bei der Mail beim Activate als auch beim Download wird das PDF aus den aktuellen Application-Daten neu generiert. Drift zwischen Mail-Versand und späterem Download ist gewollt, weil PROJ-70-Resync zwischenzeitlich Stammdaten ändern kann — der Download zeigt dann den aktuellen Stand.

### Mail-Routing beim `→ activated`

- [ ] Bei `board_approval_workflow_enabled=FALSE` (Default): **unverändertes Verhalten.** Beitrittsbestätigungs-PDF an Mitglied + EEG-Kopie wie heute. Idempotenz weiterhin via `activation_notification_sent_at`.
- [ ] Bei `board_approval_workflow_enabled=TRUE`:
  - **Keine** Mail über die Onboarding-Plattform an das Mitglied. Das Mitglied wird durch die reguläre eegFaktura-Core-Aktivierungs-Mail über den Status informiert (Core-Pfad, unabhängig von der Onboarding-Plattform — daher kein DSGVO-Lücken-Risiko).
  - **Eine** Mail an den `contact_email`-EEG-Kontakt mit der Beitrittserklärung als PDF-Anhang.
  - Subject (Erst-Aktivierung): „Beitrittserklärung zur Unterzeichnung – {Mitgliedsname} (Antrag {Antragsnummer})".
  - Begleittext:
    > Sehr geehrtes Vorstandsmitglied,
    >
    > {Mitgliedsname} ({EEG-Name}) wurde als Mitglied {Mitgliedsnummer} aufgenommen.
    >
    > Im Anhang finden Sie die Beitrittserklärung zur Unterzeichnung. Bitte unterzeichnen Sie das Dokument und leiten Sie es an das Mitglied weiter.
    >
    > Mit freundlichen Grüßen
    > Plattform-Onboarding
  - Idempotenz via `board_declaration_sent_at` (eigene Spalte, siehe oben).
- [ ] **Hart-Fail-Modus bei aktivem Vorstands-Workflow:** beide Pfade (regulärer Activate-Trigger UND `MarkActivatedSkipImport`) versenden die Mail sync hard-fail. Bei SMTP-Fehler oder fehlendem `contact_email` bricht der `→ activated`-Übergang ab; Antrag verbleibt im vorherigen Status. Konsistent mit Memory `feedback_mail_hard_fail`. Im Auto-Modus bleibt der bestehende best-effort-async-Pfad unverändert.
- [ ] **Bei Re-Aktivierung** (Activated → Reset → Activated, `board_declaration_sent_at` wurde durch ResetImport zurückgesetzt — siehe ResetImport-Sektion) markiert die zweite Mail explizit den Sachverhalt:
  - Subject: „Erneute Beitrittserklärung – {Mitgliedsname} (Antrag {Antragsnummer})".
  - Body-Block am Anfang: „Hinweis: Der Antrag wurde nach einer Korrektur erneut aktiviert. Bitte das neue Dokument unterschreiben und an das Mitglied weiterleiten; das vorherige Dokument ist nicht mehr gültig."
- [ ] Der `MarkActivatedSkipImport`-Pfad (PROJ-53 manuelle Skip-Import-Aktivierung) folgt **derselben Logik:** bei aktivem Toggle wird die Vorstands-Mail versandt (sync hard-fail), keine Member-Mail über die Plattform. `board_declaration_sent_at` wird gesetzt.

### Download-Pfad

- [ ] Antrags-Detail-Seite zeigt für aktivierte Anträge bei `board_approval_workflow_enabled=TRUE` einen Knopf „**Beitrittserklärung herunterladen**".
- [ ] Knopf ist nur sichtbar wenn:
  - Antrag im Status `activated` (vor diesem Status ist die Mitgliedsnummer noch nicht vergeben — PDF wäre unvollständig)
  - `board_approval_workflow_enabled=TRUE` zum Zeitpunkt der Anzeige
- [ ] Klick triggert eine On-Demand-Generierung des Beitrittserklärungs-PDFs (mit aktuellen Antragsdaten) und liefert sie als Download-Stream zurück. Dateiname `Beitrittserklaerung-<Antragsnummer>.pdf`, MIME `application/pdf`, `Content-Disposition: attachment`, `Content-Length` gesetzt — Browser darf nie als HTML interpretieren.
- [ ] Keine Mail wird ausgelöst — Admin kann das PDF lokal weiterverarbeiten (z.B. per Hand an Vorstand mailen oder ausdrucken).
- [ ] Endpoint: `GET /api/admin/applications/{id}/joining-declaration.pdf`. Tenant-Admin der zugehörigen RC oder Superuser. Standard-`checkTenantAccess`.
- [ ] **Audit-Log nur via `slog.Info`** mit `application_id, actor_subject, ip, user_agent` — kein `status_log`-Eintrag (Schema ist strict für Status-Übergänge: `from_status / to_status / changed_by_user_id / reason / created_at`; ein „Download-Eintrag" mit `from_status=activated, to_status=activated` wäre semantisch falsch). Forensik-Bedarf wird über Cluster-Logs gedeckt.
- [ ] Kein Throttle — niedriger Aufwand, Owner-Verantwortung.

### ResetImport-Verhalten

- [ ] Wenn ein Antrag per `POST /reset-import` von `activated` zurückgesetzt wird (unabhängig vom Toggle-Zustand):
  - `activation_notification_sent_at` wird auf NULL gesetzt (wie heute).
  - `board_declaration_sent_at` wird ebenfalls auf NULL gesetzt — beide Spalten werden synchron zurückgenommen, damit eine Re-Aktivierung den gewünschten Mail-Pfad wieder triggern kann (mit dem Re-Aktivierungs-Subject/Body, siehe oben).
- [ ] Konsistent mit dem heutigen Verhalten für die Mitglieder-Beitrittsbestätigung — keine Sonderregel.

### Sichtbarkeit im Antrags-Detail

- [ ] Wenn Antrag im Status `activated` und `board_approval_workflow_enabled=TRUE`:
  - Bestehende grüne „Aktiviert"-Status-Card bleibt unverändert.
  - **Zusätzlich darunter** ein blauer Info-Block (`bg-blue-50`-Stil): „**Beitrittserklärung an EEG-Kontakt versandt am {datum}.** Der Vorstand unterschreibt sie und leitet sie an das Mitglied weiter. Das Mitglied wird über die reguläre eegFaktura-Aktivierungs-Mail informiert."
  - Datum kommt aus `board_declaration_sent_at`.
- [ ] Im Auto-Modus (Toggle aus) bleibt die heutige Status-Anzeige unverändert (kein Hinweisblock).
- [ ] Block nutzt eine neutrale blaue Hinweisstil-Klasse (kein Warn-Banner — der Workflow ist gewollt, kein Problem).

### Tests

- [ ] **Go-Unit-Test:** Service-Layer wählt bei `board_approval_workflow_enabled=TRUE` den Beitrittserklärungs-Pfad UND skipt die Member-Mail UND setzt `board_declaration_sent_at`.
- [ ] **Go-Unit-Test:** Service-Layer wählt bei `board_approval_workflow_enabled=FALSE` den heutigen Beitrittsbestätigungs-Pfad UND sendet die Member-Mail UND setzt `activation_notification_sent_at` (Regression).
- [ ] **Go-Unit-Test:** `MarkActivatedSkipImport` folgt denselben Mail-Pfaden je nach Toggle-Zustand.
- [ ] **Go-Unit-Test:** Hart-Fail bei fehlendem `contact_email` im Vorstands-Modus — Status-Übergang bricht ab; Antrag verbleibt im vorherigen Status.
- [ ] **Go-Unit-Test:** Re-Aktivierung (Activated → Reset → Activated) im Vorstands-Modus versendet eine zweite Mail mit Re-Aktivierungs-Subject und Body-Hinweis.
- [ ] **PDF-Snapshot-Test:** PDF rendert mit `VariantBeitrittserklärung` inkl. Vorstands-Signaturblock; mit `VariantBeitrittsbestätigung` ohne Block (Regression).
- [ ] **Go-Unit-Test:** Download-Endpoint liefert das PDF nur bei `Status=activated` UND `board_approval_workflow_enabled=TRUE`; sonst 404/403 wie passend.
- [ ] **Go-Unit-Test:** Download-Endpoint prüft Tenant-Access — fremder Tenant bekommt 403.
- [ ] Bestehende Tests bleiben alle grün (`go test ./...`).

### Doku

- [ ] `docs/user-guide/06-admin-settings.md`: neuer Toggle „Vorstands-Genehmigungs-Workflow" mit Erklärung und kurzer Statuten-Hinweis-Sektion (ohne Rechtsberatung, ohne PROJ-Referenzen).
- [ ] `docs/user-guide/07-emails-and-pdfs.md`: Mail-Tabelle erweitert um die Variante „Wechsel auf Aktiviert mit aktivem Vorstands-Genehmigungs-Workflow" — Mitglied bekommt nichts, EEG-Kontakt bekommt die Beitrittserklärung.
- [ ] `docs/architecture.md` Mail-Flow-Tabelle: gleichermaßen ergänzt.
- [ ] `docs/api-spec.md`: Settings-Endpoint dokumentiert den neuen Toggle.
- [ ] `docs/domain-model.md`: `registration_entrypoint`-Sektion erwähnt `board_approval_workflow_enabled`; `application`-Sektion erwähnt `board_declaration_sent_at` und die Beziehung zum bestehenden `activation_notification_sent_at`.
- [ ] `CHANGELOG.md`: Eintrag unter `[Unreleased]`.

## Edge Cases

- **EEG hat keinen `contact_email` gepflegt + Vorstands-Modus aktiv:** Hart-Fail — der `→ activated`-Übergang bricht ab. Antrag bleibt im vorherigen Status (`ready_for_activation` oder `approved`), HTTP-Antwort nennt den Grund. Hintergrund: bei aktivem Vorstands-Modus ist die EEG-Mail kritisch (Plattform-Mail ans Mitglied entfällt, Plattform muss zumindest den Vorstand erreichen). Im Auto-Modus bleibt das heutige Skip-mit-Warn-Log-Verhalten unverändert.
- **Toggle wird während `→ activated`-Übergang umgeschaltet:** Race-Risiko, in der Praxis fast ausgeschlossen. Der Wert wird beim Mail-Build aus dem dann persistierten Stand gelesen — keine Snapshot-Logik notwendig.
- **Download nach Toggle-Wechsel:** Antrag wurde im Auto-Modus aktiviert (Member hatte Beitrittsbestätigung bekommen), danach wird der Toggle auf TRUE gestellt, dann Admin klickt Download. Verhalten: liefert die **Beitrittserklärung** (mit Vorstands-Signaturblock) — der vorherige Versand ans Mitglied bleibt unangetastet. Audit-Trail dokumentiert den Download separat vom ursprünglichen Mail-Versand.
- **Toggle wird nach Erstaktivierung deaktiviert:** Beitrittserklärung war an EEG-Kontakt raus. Toggle wird auf FALSE gesetzt. Download-Knopf im Antrags-Detail ist nicht mehr sichtbar (Bedingung erfüllt nicht mehr). Falls Admin doch herunterladen will: Toggle kurz aktivieren, herunterladen, dann ggf. zurücksetzen — pragmatisch.
- **Antrag wird zwei Mal aktiviert (ResetImport + Re-Aktivierung):** `activation_notification_sent_at` wird zurückgesetzt, zweite Aktivierung schickt Beitrittserklärung erneut an EEG-Kontakt. Konsistent.
- **EEG-Kopie (Auto-Modus):** im Auto-Modus geht heute die Beitrittsbestätigung an Mitglied + EEG-Kopie. Im Vorstands-Modus entfällt die Mitglieder-Mail komplett, die „EEG-Kopie"-Mail wird durch die Vorstands-Genehmigungs-Mail ersetzt — keine doppelten Mails an die EEG.
- **Konflikt mit `sepaMandateAtImport`:** Beim B2B / Mandat-bei-Import-Pfad geht heute beim Import (nicht beim Aktivierung) eine schlanke Mandat-Mail mit PDF an Mitglied + EEG-Kopie. Diese Pfade sind **unabhängig** vom Vorstands-Genehmigungs-Workflow — sie betreffen einen anderen Übergang. Im Vorstands-Modus laufen sie unverändert.
- **Antragsdetail wird vom Mitglied geprüft (PROJ-43 needs_info):** Vor `activated` ist der Toggle irrelevant. Verhalten nur beim `→ activated`-Übergang.
- **Member-Information bei aktivem Toggle:** das Mitglied erhält **keine** Mail über die Onboarding-Plattform, wird aber über die reguläre eegFaktura-Core-Aktivierungs-Mail informiert (Core-Pfad, läuft unabhängig). Damit bleibt das Mitglied über seinen Aktivierungs-Status auf dem Laufenden, ohne dass die Plattform eine eigene Hinweis-Mail versenden muss.

## Technische Anforderungen

- **Performance:** Im Auto-Modus bleibt der Mail-Versand im bestehenden best-effort-async-Pfad. Im Vorstands-Modus läuft die Mail sync hard-fail (Activate-Mail-Outage blockiert den Status-Wechsel — bewusste Owner-Entscheidung, weil Plattform-Mail ans Mitglied entfällt und damit die EEG-Mail das einzige Onboarding-Plattform-Signal ist). Download generiert das PDF on-demand (kein Cache) und liefert es direkt aus.
- **Sicherheit:** Download-Endpoint nur für Tenant-Admin der zugehörigen RC oder Superuser. Standard-`checkTenantAccess`-Pfad. `Content-Disposition: attachment`, MIME `application/pdf`, `Content-Length` gesetzt — Browser darf nie als HTML interpretieren.
- **Migration:** zwei Spalten in zwei Migrationen.
  - `registration_entrypoint.board_approval_workflow_enabled BOOLEAN NOT NULL DEFAULT FALSE`
  - `application.board_declaration_sent_at TIMESTAMPTZ NULL`
  - Beide ALTER ADD sind auf PostgreSQL 11+ non-blocking. Down-Migrationen droppen die Spalten.
- **Browser-Support:** UI-Elemente nutzen bestehende shadcn-Komponenten (Switch, Card, Button, Popover).
- **Memory-Regeln:**
  - `feedback_no_placeholders` — Toggle-Label ohne placeholder=.
  - `feedback_anonymized_examples` — Max Mustermann / Musterbetrieb in Doku-Beispielen.
  - `feedback_no_proj_refs_in_user_doc` — `docs/user-guide/**` muss PROJ-frei sein.
  - `feedback_mail_hard_fail` — irrelevant für PROJ-76, weil Download keine Mail mehr auslöst (Owner-Anpassung 2026-06-07).
  - `feedback_shared_helpers_for_parallel_paths` — Auto-Modus und Vorstands-Modus teilen sich denselben Mail-Service-Eintrittspunkt; das `if board_approval_workflow_enabled`-Branching lebt nur an einer Stelle.
- **Hint-Popover-Pattern:** wie in `.claude/rules/frontend.md`.

## Nicht im Scope

- **Digitale Signatur** (qualifizierte elektronische Signatur, Scan-Embed).
- **Upload des unterschriebenen Dokuments** im Admin-UI (Counter-Sign-Workflow) — Spätere Iteration falls gewünscht.
- **Hybrid-Versand:** Member-Mail UND Vorstands-Genehmigung gleichzeitig — Owner-Entscheidung „STATT".
- **Neuer Status `awaiting_board_signature`** — Owner-Entscheidung „Wechsel auf activated wie bisher".
- **Auto-Mahnung an Vorstand** bei nicht-Unterschrift.
- **Anwendung auf andere Mail-Übergänge** — nur `→ activated` ist betroffen.
- **Vorab-Druck des Vorstand-Namens** aus dem Core-Sync — Owner-Entscheidung „Linien leer lassen".
- **Mehrere Unterschriftenfelder** (Stellvertreter, Schriftführer) — Owner-Entscheidung „eine generische Linie 'Vorstand'".
- **Separates Vorstands-Mail-Feld** — Owner-Entscheidung „contact_email reicht".
- **Member-Hinweis-Mail über die Onboarding-Plattform** beim Wechsel auf aktiviert — Owner-Entscheidung „absoluter Verzicht". Die reguläre eegFaktura-Core-Aktivierungs-Mail (außerhalb dieses Codebases) informiert das Mitglied.
- **Datenschutz-Erklärung im Public-Form um Vorstands-Workflow-Hinweis erweitern** — entfällt, weil die Core-Mail das Mitglied informiert.

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)

**Erstellt:** 2026-06-07 — basierend auf der nach `/grill-me` reviewten Spec mit 17 verankerten Owner-Entscheidungen.

### Überblick

Das Feature ist ein **Mail-Routing-Wechsel** mit minimaler Schema-Erweiterung. Kein neues Datenmodell, kein neuer Status, kein neuer Renderer. Die Plattform lernt eine zweite Variante des bestehenden Aktivierungs-Prozesses, gesteuert durch genau einen EEG-Toggle. Alle bestehenden EEGs bleiben im heutigen Auto-Modus, weil der Toggle per Default auf FALSE steht. Migration ist nicht-destruktiv (zwei ALTER ADD).

### A) Komponenten-Baum: Backend-Änderungen

```
Activate-Pfad (heute existierend)
+-- admin_service.SendActivationNotification
|   +-- bestehender Auto-Modus (unverändert)
|   +-- NEU: Toggle-Branch
|       +-- liest registration_entrypoint.board_approval_workflow_enabled
|       +-- WENN aktiv → SendBoardApprovalRequest aufrufen (sync hard-fail)
|       +-- WENN nicht aktiv → wie heute (best-effort async)
|
+-- admin_service.MarkActivatedSkipImport (PROJ-53)
|   +-- gleicher Toggle-Branch wie oben (Spec-Kriterium B1)
|
+-- admin_service.ResetImport
    +-- erweitert: cleart sowohl activation_notification_sent_at
        als auch board_declaration_sent_at

Neue Bausteine
+-- mail/service.SendBoardApprovalRequest
|   +-- Subject + Body je nach Erst-/Re-Aktivierung
|   +-- contact_email-Pre-Check → Hart-Fail wenn leer
|   +-- Anhang: PDF in Variante Beitrittserklärung
|
+-- pdf/approval_pdf.GenerateApproval(data, variant)
|   +-- bestehende Render-Pipeline 1:1
|   +-- Header-Titel je nach Variante
|   +-- Vorstands-Signaturblock am Ende, nur bei Variante Beitrittserklärung
|
+-- application_repo.SetBoardDeclarationSentAt
|   +-- Setter für die neue Spalte (analog SetActivationNotificationSentAt)
|
+-- http/admin.DownloadJoiningDeclarationPDF
    +-- neuer HTTP-Endpoint, Sichtbarkeit Status=activated UND Toggle aktiv
    +-- Tenant-Check via bestehende checkTenantAccess
    +-- on-demand Render, kein Cache
    +-- slog.Info-Audit (kein status_log-Eintrag, Schema-konform)
```

**Kein neuer Service-Layer.** Alle Erweiterungen leben in bereits existierenden Service-Funktionen — die Toggle-Differenzierung ist eine 2-3-Zeilen-Abfrage in den zwei Eintrittspunkten (regulärer Activate + Skip-Import).

### B) Komponenten-Baum: Frontend-Änderungen

```
EEG-Einstellungen
+-- admin-eeg-settings-editor.tsx
|   +-- neuer Toggle „Beitrittserklärung vom Vorstand genehmigen lassen…"
|       +-- inkl. Hint-Popover (Pattern .claude/rules/frontend.md)
|       +-- State, Snapshot, Save-Payload erweitert
|       +-- nimmt am bestehenden Dirty-Tracking teil
|
+-- settings-mode.ts
|   +-- isAdvancedEEGSettingsActive-Liste um den neuen Toggle erweitert
|       (PROJ-67-Awareness-Trigger erkennt nicht-Default-Workflow)

Antrags-Detail
+-- admin-application-detail (oder vergleichbare Sicht)
|   +-- bei Status=activated UND Toggle aktiv:
|       +-- bestehende grüne „Aktiviert"-Status-Card bleibt unverändert
|       +-- DARUNTER ergänzt: blauer Info-Block (bg-blue-50)
|       |    „Beitrittserklärung an EEG-Kontakt versandt am {datum}.
|       |    Der Vorstand unterschreibt sie und leitet sie an das Mitglied
|       |    weiter. Das Mitglied wird über die reguläre eegFaktura-
|       |    Aktivierungs-Mail informiert."
|       +-- Download-Button „Beitrittserklärung herunterladen"

API-Library
+-- lib/api.ts
    +-- EEGSettings: neue boardApprovalWorkflowEnabled-Property
    +-- EEGSettingsSavePayload: gleiche Property
    +-- RegistrationConfig: kein Eintrag (Public-Form ist nicht betroffen)
    +-- neue Funktion downloadJoiningDeclarationPDF(applicationId, token)
        +-- ruft GET /api/admin/applications/{id}/joining-declaration.pdf
        +-- Browser-Download via Blob + Object-URL (analog AVV-Download PROJ-71)
```

### C) Datenmodell (plain language)

**Bestehende Tabelle `registration_entrypoint`** — eine neue Spalte:

| Feld | Typ | Default | Bedeutung |
|---|---|---|---|
| `board_approval_workflow_enabled` | Boolean | FALSE | Wenn TRUE: bei Wechsel auf „Aktiviert" geht die Beitrittserklärung an den EEG-Kontakt statt einer Beitrittsbestätigung über die Plattform an das Mitglied. |

**Bestehende Tabelle `application`** — eine neue Spalte:

| Feld | Typ | Default | Bedeutung |
|---|---|---|---|
| `board_declaration_sent_at` | Zeitstempel (nullable) | NULL | Wird gesetzt, wenn die Beitrittserklärung an den EEG-Kontakt geschickt wurde. Mehrfach-Versand-Schutz im Vorstands-Modus. Wird beim ResetImport-Pfad auf NULL zurückgesetzt. |

**Beziehung Workflow-Modus ↔ Antrag:** Pro Antrag entscheidet zum Zeitpunkt des „→ Aktiviert"-Übergangs der Toggle-Wert der zugehörigen EEG (`registration_entrypoint`). Wenn der Toggle nachträglich umgestellt wird, betrifft das **nur** nachfolgende Aktivierungen. Bereits aktivierte Anträge behalten ihren ursprünglichen Mail-Pfad.

**Beziehung der beiden Sent-At-Spalten:**

| Modus | activation_notification_sent_at | board_declaration_sent_at |
|---|---|---|
| Auto (Toggle FALSE) | gesetzt | bleibt NULL |
| Vorstand (Toggle TRUE) | bleibt NULL | gesetzt |
| ResetImport (beide) | NULL | NULL |

Eindeutige Trennung — keine semantische Mischung, keine Spalten-Mehrdeutigkeit.

### D) Datenfluss-Sequenzen

#### Sequenz 1: „→ Aktiviert" im Auto-Modus (heute, unverändert)

```
Activation-Check-Batch (oder manuelle Aktivierung)
   │
   ▼
Antrag wechselt auf Status „Aktiviert"
   │
   ▼
SendActivationNotification wird aufgerufen
   │
   ├── Toggle FALSE? → Auto-Pfad
   │      │
   │      ▼
   │   Beitrittsbestätigungs-PDF generieren (Variante: Beitrittsbestätigung)
   │      │
   │      ▼
   │   Mail an Mitglied + EEG-Kopie (best-effort async)
   │      │
   │      ▼
   │   activation_notification_sent_at gesetzt
```

#### Sequenz 2: „→ Aktiviert" im Vorstands-Modus (neu)

```
Activation-Check-Batch (oder manuelle Aktivierung)
   │
   ▼
Antrag wechselt auf Status „Aktiviert"
   │
   ▼
SendActivationNotification wird aufgerufen
   │
   ├── Toggle TRUE? → Vorstands-Pfad (SendBoardApprovalRequest)
   │      │
   │      ▼
   │   Pre-Check: contact_email gepflegt?
   │      │
   │      ├── NEIN → Hart-Fail: Status-Wechsel rollt zurück
   │      │           Antrag bleibt im vorherigen Status
   │      │           Admin sieht Fehlermeldung
   │      │
   │      └── JA → weiter
   │             │
   │             ▼
   │          Beitrittserklärungs-PDF generieren (Variante: Beitrittserklärung)
   │             │
   │             ▼
   │          Subject + Body je nach Erst-/Re-Aktivierung
   │          (board_declaration_sent_at vorher war NULL → Erst-Mail;
   │          vorher gesetzt → Re-Mail mit Re-Aktivierungs-Vermerk)
   │             │
   │             ▼
   │          Mail an EEG-Kontakt (sync hard-fail)
   │             │
   │             ├── SMTP-Fehler → Hart-Fail: Status-Wechsel rollt zurück
   │             │
   │             └── Erfolg → board_declaration_sent_at gesetzt
   │
   ▼
(Parallel, außerhalb dieses Codebases: eegFaktura-Core verschickt
seine reguläre Aktivierungs-Mail an das Mitglied — Member bleibt
über seinen Status informiert.)
```

#### Sequenz 3: Download-Button im Antrags-Detail

```
Admin öffnet aktivierten Antrag im Vorstands-Modus
   │
   ▼
Antrags-Detail-Sicht
   │
   ├── Status=activated UND Toggle aktiv → Download-Button sichtbar
   │
   ▼
Admin klickt „Beitrittserklärung herunterladen"
   │
   ▼
GET /api/admin/applications/{id}/joining-declaration.pdf
   │
   ▼
checkTenantAccess (Tenant-Admin der RC oder Superuser)
   │
   ├── kein Zugriff → 403
   │
   ▼
PDF on-demand generieren (Variante: Beitrittserklärung)
   │
   ▼
Stream als application/pdf, Content-Disposition: attachment
   │
   ▼
Browser triggert Download (Dateiname: Beitrittserklaerung-<Antragsnummer>.pdf)
   │
   ▼
slog.Info-Audit: application_id, actor_subject, ip, user_agent
(kein status_log-Eintrag, weil kein Status-Wechsel)
```

### E) Tech-Entscheidungen (Begründungen für PM)

| Entscheidung | Begründung |
|---|---|
| **Separate Spalte `board_declaration_sent_at`** statt Recycling des bestehenden `activation_notification_sent_at` | Zwei semantisch unterschiedliche Mail-Events. Mit Recycling wäre die Spalten-Bedeutung mehrdeutig — bei späteren Toggle-Wechseln oder Audit-Abfragen schwer zu erklären. Separate Spalten erlauben eine klare „im Auto-Modus passiert X, im Vorstands-Modus passiert Y"-Aussage. |
| **Single Renderer mit Variant-Parameter** statt separater PDF-Datei | ~90% des Renderer-Codes (Mitgliedsdaten, Zählpunkte, Zustimmungen, Genossenschaftsanteile, Netzbetreiber-Info-Seite) wären in einer zweiten Datei identisch. Memory `feedback_shared_helpers_for_parallel_paths` mahnt zu Recht vor dieser Form von Drift. Eine Variante steuert nur die zwei tatsächlichen Unterschiede (Header-Titel, Signaturblock am Ende). |
| **On-demand PDF** statt Persistierung als BYTEA-Blob | Die Anwendung hat keine DSGVO-Anker-Anforderung wie das AVV-PDF (PROJ-71), das einen unveränderlichen Akzept-Beleg darstellt. Hier ist das PDF eine deterministische Repräsentation der Antragsdaten. Wenn der Admin nach einem PROJ-70-Resync das Dokument neu herunterlädt, sieht er den aktuellen Stand — gewollt. Spart Spalte und Speicherplatz. |
| **Hart-Fail bei fehlendem `contact_email` im Vorstands-Modus** | Im Auto-Modus ist die Mitglieder-Mail die primäre Bestätigung — eine fehlende EEG-Kopie wird best-effort übersprungen. Im Vorstands-Modus ist die EEG-Mail der **einzige** Mail-Pfad der Plattform. Ohne EEG-Mail würde der Vorstand nichts wissen und das Mitglied (Plattform-seitig) auch nichts. Hart-Fail erzwingt die Datenpflege vor dem Übergang. |
| **`slog.Info` statt `status_log`** für den Download-Audit | `status_log` ist strikt für Status-Übergänge mit `from_status / to_status / reason`-Spalten. Ein „Download-Eintrag" mit `from_status=activated, to_status=activated` wäre semantisch falsch. Cluster-Logs decken den Forensik-Bedarf vollständig ab. |
| **Sync hard-fail im Vorstands-Modus** (Owner-Entscheidung D3) | Im Auto-Modus ist Mail-Versand best-effort — bei SMTP-Fehler bleibt der Status, eine spätere Resync-Aktion holt es nach. Im Vorstands-Modus ist die Mail kritisch genug, dass ein verlorener Versand den Aktivierungs-Prozess unterbricht. Trade-off: SMTP-Outages blockieren möglicherweise einen Activation-Check-Batch — bewusster Schutz vor stillem Datenverlust. |
| **Toggle als Advanced-Trigger (PROJ-67)** | Der Vorstands-Workflow ist eine nicht-Default-Konfiguration. EEGs, die ihn aktivieren, sollen im Standard-Settings-Modus den Awareness-Banner sehen — sonst übersieht ein neuer Admin den umgeschalteten Mail-Pfad. |
| **Member-Information via Core-Mail** (Owner-Klarstellung F1) | Plattform versendet **keine** Mitglieder-Mail im Vorstands-Modus. Mitglied wird durch die reguläre eegFaktura-Core-Aktivierungs-Mail informiert (außerhalb dieses Codebases). Damit entsteht keine DSGVO-Lücke und keine zusätzliche Public-Form-Erklärung notwendig. |

### F) Migrationspfad

| Schritt | Beschreibung | Risiko |
|---|---|---|
| 1. Migration N (registration_entrypoint) | Neue Spalte `board_approval_workflow_enabled` mit Default FALSE | Sehr gering — ALTER ADD mit Default ist auf PostgreSQL 11+ ein Metadaten-Update, kein Tabellen-Rewrite |
| 2. Migration N+1 (application) | Neue Spalte `board_declaration_sent_at` nullable | Sehr gering — analog non-blocking |
| 3. Backend-Deploy | Code prüft Toggle, neue Mail-Funktion + PDF-Variante | Kein Verhaltens-Wechsel für Bestands-EEGs (Toggle FALSE per Default) |
| 4. Frontend-Deploy | Neuer Toggle im Settings-UI, Download-Button + Hinweisblock im Antrags-Detail | Keine UI-Regression — neue UI-Elemente, keine geänderte Bestehende |
| 5. Konfiguration | EEGs, die den neuen Workflow wollen, setzen den Toggle | Ab dem Zeitpunkt wirkt der neue Pfad für nachfolgende Aktivierungen dieser EEG |

**Roll-back-Pfad:** Down-Migrationen droppen die zwei Spalten. Bestands-EEGs haben den Toggle bewusst aktiviert — wenn sie nach Rollback wieder im Auto-Modus landen, läuft das System einfach weiter (kein Datenverlust, kein Stuck-State).

### G) Risiken & Trade-offs

| Risiko | Auswirkung | Mitigation |
|---|---|---|
| **SMTP-Outage blockiert Activation-Check-Batch** | Eine längere Outage verhindert Aktivierungen vieler Anträge zugleich | Bewusster Owner-Trade-off (Entscheidung D3). Im Auto-Modus läuft der Batch unverändert weiter. |
| **Vorstand reagiert nie auf die Beitrittserklärung** | Mitglied erhält von der Plattform nichts — auch wenn der Antrag im System aktiviert ist | Die Plattform übernimmt diese Verantwortung bewusst nicht (Spec-Non-Goal). Owner-Entscheidung: Core-Mail informiert das Mitglied über den Status; alles weitere ist EEG-Verantwortung. |
| **Toggle wird nach laufender Aktivierung umgestellt** | Zwei Anträge im selben Tag bekommen unterschiedliche Mail-Pfade | Pro-Antrag-Snapshot beim Mail-Build (gelesener Wert zum Zeitpunkt des Übergangs). Keine UI-Warnung beim Toggle-Switch — Owner-Verantwortung. |
| **Re-Aktivierung sendet doppelte Mail an Vorstand** | Vorstand hat möglicherweise schon das erste Dokument unterschrieben und weitergeleitet | Re-Mail trägt klaren Subject + Body-Vermerk „Hinweis: Erneut aktiviert, vorheriges Dokument nicht mehr gültig" (Entscheidung D2). Sozial-organisatorisches Problem, technisch sauber kommuniziert. |
| **Download liefert PDF mit aktuelleren Daten als die Vorstands-Mail enthielt** | Vorstand und Admin sehen leicht unterschiedliche Inhalte, wenn ein Resync zwischendurch Stammdaten geändert hat | Entscheidung C2 (on-demand). Tester wissen, dass Resync das PDF-Bild beeinflusst. Im Audit-Log nachvollziehbar via PROJ-70-Logs. |

### H) Dependencies (Packages)

**Keine neuen Pakete** — alle Erweiterungen laufen auf dem bestehenden Stack:

- Backend: `database/sql`, `chi`-Router, `gofpdf` (PDF), `log/slog` (Audit), Standard-`net/smtp` über die bestehende Mail-Service-Abstraktion
- Frontend: `next/react`, `shadcn/ui` (Switch, Button, Popover, Card), `lucide-react` (Info-Icon), bestehende `next-auth`-Session-Anbindung
- Test: `testing` (Go), Snapshot-Pattern wie bei bestehenden PDF-Tests

### I) Open Points für die Implementation

Keine offenen Architektur-Branches. Backend und Frontend können starten — die Reihenfolge ist:

1. Migrationen (zwei Stück)
2. Backend: PDF-Variante + Mail-Service-Methode + Repository-Setter
3. Backend: Service-Layer-Branching in `SendActivationNotification` und `MarkActivatedSkipImport`
4. Backend: Download-HTTP-Endpoint
5. Backend: Tests
6. Frontend: Settings-Toggle + Snapshot + Hint-Popover
7. Frontend: `settings-mode.ts`-Trigger
8. Frontend: Antrags-Detail-Hinweisblock + Download-Button
9. Doku-Updates (sechs Dateien laut Spec)
10. Build + Tests + Commit + Push

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
