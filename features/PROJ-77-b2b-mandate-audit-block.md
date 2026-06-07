# PROJ-77: B2B-Mandat-Audit-Block (Firmenlastschrift-PDF)

## Status: In Review (Backend implementiert, Tests grün, Doku aktualisiert)
**Created:** 2026-06-07
**Last Updated:** 2026-06-07

## Dependencies

- Erfordert: PROJ-14 (Firmenlastschrift-Mandat-PDF-Generierung) — Deployed, wird hier modifiziert
- Erfordert: PROJ-31 (E-Mail-Bestätigung) — Deployed, beeinflusst den Wortlaut des Audit-Texts
- Erfordert: PROJ-32 (EEG-Stammdaten-Sync) — Deployed, liefert `eeg_name`
- Erfordert: PROJ-48 (per-Antrag `einzugsart`) — Deployed, Voraussetzung für „nur B2B"-Scope

## Hintergrund

Owner-Feature-Request 2026-06-07: Im SEPA-Firmenlastschrift-Mandat-PDF
(`einzugsart=b2b`) soll der heutige Datum/Unterschrift-Block durch einen
Audit-Trail-Text ersetzt werden, der die elektronisch erteilte Zustimmung
rechtssicher dokumentiert.

**Rechtlicher Kontext:** § 76 (3) EIWOG 2010 anerkennt formfreie
Willenserklärungen für Elektrizitätsverträge. Mit dem Audit-Trail wird die
elektronische SEPA-Zustimmung als rechtskonforme Willenserklärung
dokumentiert — ohne dass das Mitglied physisch unterschreiben muss.

**Audit-Text (Owner-Wortlaut):**

> Der Kunde hat der **{tenant}** nach Verifizierung seiner E-Mail-Adresse
> am **{datum}** **{uhrzeit}** von der IP-Adresse **{ip}** auf
> elektronischem Weg (formfreie Willenserklärung gem. § 76 (3) EIWOG 2010)
> seine Zustimmung zum Vertrag im obigen Sinne sowie für das
> SEPA-Lastschriftmandat erteilt.

**Platzhalter-Quellen:**

| Platzhalter | Datenquelle |
|---|---|
| `{tenant}` | `registration_entrypoint.eeg_name` (PROJ-32-Sync) |
| `{datum}` `{uhrzeit}` | `application.sepa_mandate_accepted_at` (vorhanden), formatiert als `DD.MM.YYYY HH:MM` |
| `{ip}` | `application.sepa_mandate_accepted_ip` (NEU — Migration 000069) |

## User Stories

- Als **Owner** möchte ich, dass das B2B-Mandat-PDF die elektronische
  Zustimmung des Mitglieds rechtssicher dokumentiert, damit die EEG bei
  einem Klärungsfall (z.B. Rückbuchung, Bank-Anfrage) den Audit-Trail
  vorlegen kann.
- Als **EEG-Vorstand** möchte ich keine physisch unterschriebene B2B-Mandat-
  Vorlage mehr brauchen, sondern die elektronische Zustimmung als
  vollwertigen Nachweis nutzen können.
- Als **Mitglied** möchte ich erkennen, dass mein Klick auf die SEPA-
  Akzept-Checkbox + meine E-Mail-Bestätigung als formfreie
  Willenserklärung gilt — ohne dass ich nachträglich ein PDF unterzeichnen
  und zurücksenden muss.
- Als **EEG-Integrator** über die externe API möchte ich die IP des End-
  Users als optionalen Body-Param mitgeben können, damit der Audit-Trail
  auch bei API-Submits vollständig wird.
- Als **Owner** möchte ich, dass Bestandsanträge ohne IP-Erfassung
  weiterhin den klassischen Unterschriftsblock bekommen — Backward-Compat
  ohne Datenverlust.

## Akzeptanzkriterien

### Datenmodell

- [ ] Neue Spalte `application.sepa_mandate_accepted_ip INET NULL` via
  Migration 000069. Type `INET` für PostgreSQL-Native-Validierung von
  IPv4/IPv6. Default NULL — Bestandsanträge bleiben unangetastet.
- [ ] **Kein Index** (Grilling D1) — bei <50.000 Anträgen kein
  Performance-Thema. Spätere Audit-Queries sind selten und akzeptieren
  Sequenz-Scan. GiST/BTREE bei Bedarf später nachziehen.
- [ ] **Bleibt NULL-fähig** (Grilling D2) — kein späterer NOT-NULL-Pfad
  geplant. Bestandsanträge dürfen NULL bleiben.

### Persistenz-Semantik

- [ ] **Erst-Submit gewinnt** (Grilling A1): Bei mehrfachem Submit aus
  dem `needs_info`-Korrekturpfad bleiben `sepa_mandate_accepted_at` UND
  `_ip` unverändert. Die SEPA-Zustimmung ist die historische — Daten-
  Korrektur ändert sie nicht. Die Repository-Schicht prüft, ob das
  Feld bereits gesetzt ist, und überschreibt es **nicht**.
- [ ] **ResetImport lässt die Audit-Spalten unangetastet** (Grilling A2):
  `activated → approved` cleart die `sepa_mandate_accepted_*`-Felder
  **nicht**. Die ursprüngliche Zustimmung bleibt als Audit-Anker
  erhalten; eine spätere Re-Aktivierung greift auf denselben Audit-Trail.
- [ ] **PROJ-70 Stammdaten-Resync verwendet die ursprüngliche Member-IP**
  (Grilling A3): Wenn der Admin „Stammdaten aus eegFaktura abgleichen"
  klickt und das B2B-Mandat-PDF neu generiert wird, rendert der Audit-
  Block mit der ursprünglichen `sepa_mandate_accepted_ip`. Begründung:
  die Zustimmung ist nicht neu — der Admin korrigiert nur Stammdaten.
  Eine Admin-IP würde zwei semantische Events vermischen.

### Public-Submit-Pfad

- [ ] `POST /api/public/applications` erfasst die End-User-IP via
  bestehendem `realIP`-Middleware-Helper (`internal/http/middleware.go`)
  und schreibt sie auf `application.sepa_mandate_accepted_ip`.
- [ ] Persistierungs-Zeitpunkt: beim Initial-Submit, synchron mit dem
  `sepa_mandate_accepted_at`-Stempel.
- [ ] Wenn der `realIP`-Helper keine IP liefert (z.B. fehlgeschlagene
  Trusted-Proxy-Konfig): Spalte bleibt NULL — kein Fehler im Submit-Pfad.

### Externe API (`/api/external/v1/applications`)

- [ ] Neuer optionaler Request-Body-Param `submitterIp` (string, IPv4/IPv6).
  EEG-Integrator kann die End-User-IP aus seinem ursprünglichen Browser-
  Request mitliefern.
- [ ] Validierung: wenn `submitterIp` gesetzt, muss das Format gültige
  IPv4 oder IPv6 sein. Bei Fehler → `400 validation_error` mit Feld-
  Hinweis.
- [ ] Wenn `submitterIp` fehlt oder ungültig ist: Spalte bleibt NULL.
  Begründung: die IP des EEG-Servers ist für SEPA-Audit-Trail
  irreführend (das ist nicht die Zustimmungs-IP des Mitglieds).

### B2B-PDF-Renderer (`einzugsart=b2b`)

- [ ] In `internal/pdf/generator.go` `GenerateCompany`-Methode (B2B-PDF):
  der heutige Unterschriftsblock (Zeilen 378-395, „Ort, Datum,
  Unterschrift"-Linie + Box) wird durch den **Audit-Block** ersetzt,
  **wenn** alle folgenden Bedingungen erfüllt sind:
  - `application.sepa_mandate_accepted_at` ist nicht NULL
  - `application.sepa_mandate_accepted_ip` ist nicht NULL
  - `eeg_name` ist gesetzt
- [ ] Andernfalls bleibt der heutige Unterschriftsblock erhalten (Fallback
  für Bestandsanträge und Externe-API-Submits ohne `submitterIp`).
- [ ] Audit-Block-Wortlaut (Standard, EEG mit `require_email_confirmation=true`):
  > Der Kunde hat der **{tenant}** nach **Verifizierung** seiner E-Mail-
  > Adresse am {datum} {uhrzeit} von der IP-Adresse {ip} auf elektronischem
  > Weg (formfreie Willenserklärung gem. § 76 (3) EIWOG 2010) seine
  > Zustimmung zum Vertrag im obigen Sinne sowie für das SEPA-Lastschrift-
  > mandat erteilt.
- [ ] Audit-Block-Wortlaut bei EEG mit `require_email_confirmation=false`:
  > Der Kunde hat der **{tenant}** nach **Eingabe** seiner E-Mail-Adresse
  > am {datum} {uhrzeit} von der IP-Adresse {ip} auf elektronischem Weg
  > (formfreie Willenserklärung gem. § 76 (3) EIWOG 2010) seine Zustimmung
  > zum Vertrag im obigen Sinne sowie für das SEPA-Lastschriftmandat
  > erteilt.
- [ ] Datumsformat: `DD.MM.YYYY HH:MM` (z.B. `21.05.2026 11:50`),
  Europe/Vienna-Timezone — konsistent mit anderen Datumsausgaben im PDF.
- [ ] Visuelle Gestaltung des Audit-Blocks (Grilling B1+B2):
  - **Kopfzeile** (fettgedruckt, 9pt): „**Elektronisch erteiltes Mandat
    (gem. § 76 (3) EIWOG 2010)**" — macht die Rechtsgrundlage explizit
    und unterscheidet visuell von der heutigen handschriftlichen
    Unterschriftslinie.
  - Audit-Text (8pt regulär) darunter, gerendert mit `MultiCell` und
    automatischem Zeilenumbruch — die Box-Höhe passt sich dynamisch an
    die Text-Länge an.
  - Eingerahmt in derselben Box-Position wie der heutige Unterschriftsblock,
    aber höhenflexibel (~25-35mm statt fixe 15mm). PDF-Layout-Test
    verifiziert, dass keine Inhalte abgeschnitten werden.
- [ ] **IPv6-Darstellung** (Grilling B3): Adressen werden komprimiert
  ausgegeben (`2001:db8:85a3::8a2e:370:7334` statt der voll-
  expandierten Form). Go-Standard-Library `net.IP.String()` macht das
  automatisch. Im PDF-Renderer kein zusätzlicher Code nötig.

### Core-Mandat-PDF (`einzugsart=core`)

- [ ] **Unverändert.** `GenerateBasis` rendert weiterhin den klassischen
  Datum/Unterschrift-Block. Owner-Direktive: Audit-Block ist B2B-spezifisch.

### Tests

- [ ] **PDF-Snapshot:** B2B-PDF mit allen drei Platzhaltern befüllt
  (Tenant „Musterstadt EEG", Datum `21.05.2026 11:50`, IP `192.0.2.42`)
  enthält den vollständigen Audit-Text. Header „SEPA-Firmenlastschrift-
  Mandat" bleibt; Box-Layout bleibt.
- [ ] **PDF-Snapshot:** B2B-PDF mit fehlender IP (NULL) fällt auf den
  heutigen Unterschriftsblock zurück. Box-Layout identisch zum
  Pre-PROJ-77-Verhalten.
- [ ] **PDF-Snapshot:** B2B-PDF mit `require_email_confirmation=false`
  rendert Wortlaut „nach Eingabe" statt „nach Verifizierung".
- [ ] **Regression:** Core-Mandat-PDF (`GenerateBasis`) ist unverändert
  — gleicher Snapshot wie pre-PROJ-77.
- [ ] **Submit-Test:** Public-Submit speichert IP aus `realIP`-Middleware.
- [ ] **Externe-API-Test:** Submit mit `submitterIp` speichert das Feld;
  Submit ohne `submitterIp` lässt NULL.
- [ ] **Validierung:** Externe API rejects malformed `submitterIp`
  (z.B. `"foo.bar"`, `"999.999.999.999"`) mit 400.
- [ ] **Regression A1:** Mehrfach-Submit-Test — IP der ersten Submission
  bleibt erhalten, Re-Submission überschreibt nicht.
- [ ] **Regression A2:** ResetImport-Test — `sepa_mandate_accepted_at`
  und `_ip` bleiben gesetzt.
- [ ] **Regression A3:** PROJ-70-Resync-Test — PDF wird mit ursprünglicher
  Member-IP gerendert, nicht mit Admin-Trigger-IP.
- [ ] **PDF-Layout-Test:** Audit-Block mit IPv6-Adresse + langem EEG-Namen
  bricht sauber um, schneidet nichts ab.

### Doku & Aussen-Kommunikation

- [ ] `docs/domain-model.md`: neue Spalte `sepa_mandate_accepted_ip`
  beschreiben mit Hinweis auf PROJ-77 + § 76 (3) EIWOG.
- [ ] `docs/api-spec.md` Sektion 8 (External API): neuer optionaler
  `submitterIp`-Body-Param dokumentieren **mit explizitem „NEU 2026-06-XX:
  Audit-Block für B2B-Mandate"-Hinweis** (Grilling C1). Bestands-
  Integratoren werden nicht aktiv benachrichtigt — die Doku-Aktualisierung
  ist der Kommunikationsweg.
- [ ] `docs/user-guide/06-admin-settings.md` Sektion „Externe API" /
  „Nutzung der API": Ein-Zeilen-Hinweis ergänzen, dass das `submitterIp`-
  Feld neu im Request-Body akzeptiert wird (Grilling C2). Im Settings-UI
  selbst keine Änderung.
- [ ] `docs/architecture.md`: knapper Verweis auf den PROJ-77-Audit-
  Trail-Mechanismus, falls eine Übersicht der Audit-Felder existiert.
- [ ] `src/app/datenschutz/page.tsx`: vorhandene IP-Erwähnung um expliziten
  Hinweis ergänzen (Grilling E2): „Bei elektronischer Erteilung eines
  SEPA-Lastschriftmandats speichern wir die IP-Adresse zum Zeitpunkt
  Ihrer Zustimmung als Audit-Trail gemäß § 76 (3) EIWOG 2010."
- [ ] `CHANGELOG.md`: Eintrag unter `[Unreleased]`.
- [ ] `docs/user-guide/changelog.md`: 2026-06-XX-Eintrag (in User-Sprache,
  PROJ-frei).

### Audit/DSGVO

- [ ] **IP im Excel-Export** (Grilling E1): IP-Spalte wird über das
  PROJ-15-FieldConfig-Pattern als optionales Export-Feld konfigurierbar.
  Default `hidden` — Owner-Entscheidung pro EEG, ob die IP im Export
  enthalten ist. Kein automatisches Mitliefern (DSGVO-Default
  „minimaler Export-Datenpunkt").

## Edge Cases

- **Bestandsantrag (vor PROJ-77 erstellt) wird re-aktiviert** und das
  B2B-PDF wird neu generiert (PROJ-70 Resync): IP ist NULL → heutiger
  Unterschriftsblock. Korrekt — die elektronische Zustimmung wurde nie
  erfasst.
- **IPv6-Adresse**: INET akzeptiert sie nativ. Audit-Text rendert sie als
  einzelne Zeile (PDF-Renderer muss den langen String umbrechen können
  oder kleinere Schrift wählen — Layout-Test nötig).
- **IP-Adressen-Format mit Port** (z.B. aus `r.RemoteAddr`): Port wird
  abgeschnitten, bevor der Wert in `sepa_mandate_accepted_ip` geschrieben
  wird. `realIP`-Helper macht das schon, muss nur durchgängig genutzt
  werden.
- **X-Forwarded-For-Chain mit mehreren Hops** (Reverse-Proxy + CDN):
  `realIP`-Middleware berücksichtigt nur die erste IP der trusted
  Proxy-Chain. Spalte enthält genau eine IP, kein Komma-getrennter String.
- **Datum vor Zeitzonen-Wechsel** (z.B. Wechsel CET/CEST am 30.03.):
  `Europe/Vienna` automatisch korrekt durch `shared.FmtDateTime`-Helper
  (PROJ-27).
- **Externe API mit gültigem `submitterIp` aber Server-IP des EEG würde
  dem widersprechen**: kein Konflikt — `submitterIp` aus Body gewinnt,
  Server-IP wird ignoriert.
- **Mitglied widerruft Zustimmung** (Status-Wechsel `rejected`): Audit-
  Trail bleibt in der DB erhalten (Beweismittel-Anker). Spalte wird nicht
  gecleart.
- **Mehrfach-Submit nach `needs_info`-Korrekturpfad:** IP der **ersten**
  Submission bleibt erhalten. Die Re-Submission ist Daten-Korrektur, nicht
  neue Zustimmung. (Owner-Entscheidung A1.)
- **`activated → approved`-Reset:** beide Audit-Spalten bleiben gesetzt;
  Re-Aktivierung greift auf denselben Audit-Trail. (A2.)
- **PROJ-70 Stammdaten-Resync auf aktiviertem Antrag:** B2B-Mandat-PDF
  wird mit der ursprünglichen Member-IP neu generiert — Admin-Aktion
  überschreibt nicht. (A3.)

## Technische Anforderungen

- **Migration:** Eine `ALTER TABLE` mit `ADD COLUMN`. Non-blocking auf
  PostgreSQL 11+.
- **Submit-Pfad-Eingriff minimal:** `realIP`-Helper ist bereits da, muss
  nur an einer Stelle aufgerufen + gespeichert werden.
- **PDF-Renderer-Eingriff:** ein If-Branch in `GenerateCompany`. Single
  Helper-Funktion für Audit-Block-Render (kein Duplikat).
- **Performance:** unverändert. Kein zusätzlicher DB-Roundtrip; IP wird
  im selben INSERT geschrieben wie der bestehende Submit.
- **Sicherheit:** `submitterIp` muss validiert werden (RFC 791/2460),
  sonst SQL-Injection-Schutz via `INET`-Spalte ist verlassen.
- **Memory-Regeln:**
  - `feedback_anonymized_examples` — Test-Snapshots: Max Mustermann,
    Beispiel-IP `192.0.2.42` (RFC 5737 documentation range)
  - `feedback_no_proj_refs_in_user_doc` — `docs/user-guide/changelog.md`
    PROJ-frei
  - `feedback_shared_helpers_for_parallel_paths` — wenn Core-Mandat
    irgendwann auch den Audit-Block bekommt, gemeinsamen Helper bauen.

## Nicht im Scope

- **Core-Mandat-PDF** (`einzugsart=core`) bleibt unverändert. Audit-Block
  ist B2B-spezifisch.
- **Pro-EEG-Toggle** zum Aktivieren/Deaktivieren des Audit-Blocks —
  nicht gewünscht, der Block ist Standard bei allen B2B-Mandaten mit
  vollständigen Daten.
- **IP-Anonymisierung** für DSGVO (z.B. letztes Octett auf 0 setzen).
  Owner-Direktive: vollständige IP ist Audit-Pflicht, nicht Tracking-
  Datenpunkt.
- **Audit-Block für andere PDFs** (Beitrittsbestätigung, AVV-PDF) —
  separate Specs falls gewünscht.
- **Frontend-Änderungen**: keine. Reine Backend-Modifikation.
- **GDPR-Lösch-/Auskunfts-Sonderbehandlung** für die IP — folgt der
  bestehenden DSGVO-Pipeline (PROJ-25/PROJ-26 Datenweiterleitung).

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)

**Erstellt:** 2026-06-07 — basierend auf der nach `/grill-me` reviewten Spec mit 13 verankerten Owner-Entscheidungen (A1–F1).

### Überblick

Das Feature ist eine **PDF-Layout-Anpassung mit Datenmodell-Erweiterung**. Eine neue nullable Spalte fängt die IP der elektronischen SEPA-Zustimmung; der B2B-PDF-Renderer entscheidet je nach Daten-Vollständigkeit zwischen dem heutigen Unterschriftsblock und dem neuen Audit-Block. Bestandsanträge bleiben rückwärtskompatibel: ohne IP greift weiterhin der klassische Block. Reine Backend-Änderung, kein Frontend-Anteil.

### A) Komponenten-Baum: Backend-Änderungen

```
Datenmodell (neu)
+-- Migration 000069
|   +-- application.sepa_mandate_accepted_ip INET NULL
|   +-- kein Index (Audit-Queries sind selten)
|
+-- shared.Application
    +-- neues Feld SepaMandateAcceptedIP *net.IP (oder *string)

Submit-Pfade (zwei Eintrittspunkte)
+-- Public-Submit POST /api/public/applications
|   +-- bestehende realIP-Middleware liest die End-User-IP
|   +-- Service-Layer reicht die IP an die Repository-Schicht durch
|   +-- Repository: Insert in application inkl. IP
|       +-- Idempotenz beim Re-Submit (needs_info-Pfad): IP wird NICHT
|           überschrieben, wenn bereits gesetzt (Erst-Submit gewinnt)
|
+-- Externe API POST /api/external/v1/applications
    +-- neuer optionaler Body-Param submitterIp
    +-- Format-Validierung gegen IPv4/IPv6 (Standard-Library net.ParseIP)
    +-- Bei Validierungsfehler → 400 mit Feld-Hinweis
    +-- Fehlt der Param: IP bleibt NULL (Audit-Block-Fallback greift später)

PDF-Renderer (internal/pdf/generator.go GenerateCompany)
+-- bestehender Renderer wird um einen Variant-Switch erweitert
|   +-- Audit-Block gerendert, wenn:
|       - sepa_mandate_accepted_at gesetzt
|       - sepa_mandate_accepted_ip gesetzt
|       - eeg_name gesetzt
|   +-- Sonst: heutiger Unterschriftsblock (Backward-Compat)
|
+-- Audit-Block-Layout
|   +-- Kopfzeile: „Elektronisch erteiltes Mandat (gem. § 76 (3) EIWOG 2010)"
|   +-- Audit-Text mit drei Platzhaltern (Tenant, Datum/Uhrzeit, IP)
|   +-- Wortlaut-Variante je nach require_email_confirmation
|       (Verifizierung vs Eingabe)
|   +-- Layout: MultiCell-Auto-Umbruch, 8pt, höhenflexible Box
|
+-- IPv6-Darstellung
    +-- Go-Standard net.IP.String() liefert komprimierte Form

Service-Schichten (Anpassungen statt Neubau)
+-- application_service.SubmitApplication
|   +-- nimmt IP zusätzlich entgegen, reicht an Repo durch
|
+-- resync_service.SendMandateRenewalMail (PROJ-70)
|   +-- B2B-Mandat-PDF-Generierung läuft mit der ursprünglichen Member-IP
|       aus der Application — kein Überschreiben mit Admin-IP
|
+-- ResetImport-Pfad
    +-- sepa_mandate_accepted_* bleibt unangetastet (im Gegensatz zu
        activation_notification_sent_at / board_declaration_sent_at,
        die PROJ-76 cleart). Begründung: SEPA-Zustimmung ist Audit-Anker.

Excel-Export (PROJ-15 FieldConfig)
+-- neues Field-Config-Entry „sepa_mandate_accepted_ip"
    +-- Default visibility: hidden
    +-- EEG-Admin kann pro EEG aktivieren

Datenschutz-Doku
+-- src/app/datenschutz/page.tsx
    +-- bestehender IP-Hinweis ergänzt um EIWOG-Referenz
```

**Kein neuer Endpoint, kein neues Service-Modul.** Alle Änderungen sind Erweiterungen bestehender Pfade — minimale Refactor-Belastung.

### B) Datenmodell (plain language)

**Bestehende Tabelle `application`** — eine neue Spalte:

| Feld | Typ | Default | Bedeutung |
|---|---|---|---|
| `sepa_mandate_accepted_ip` | IP-Adresse (nullable) | NULL | IP-Adresse zum Zeitpunkt der SEPA-Mandats-Akzeptanz. Bei Bestandsanträgen NULL — der PDF-Renderer fällt dann auf den klassischen Unterschriftsblock zurück. Wird beim ResetImport nicht zurückgesetzt (Audit-Anker). |

**Beziehung:** Pro Antrag genau eine IP, geschrieben beim Initial-Submit. Korrektur-Submissions (`needs_info → submitted`) überschreiben den Wert nicht — der ursprüngliche Zustimmungs-Zeitpunkt bleibt erhalten.

**Zusammenspiel mit bestehenden Audit-Spalten:**

| Spalte | Bedeutung | Wann gesetzt |
|---|---|---|
| `sepa_mandate_accepted` | Hat das Mitglied die Checkbox angeklickt? | Beim Submit |
| `sepa_mandate_accepted_at` | Wann hat es geklickt? | Beim Submit |
| `sepa_mandate_accepted_ip` *(neu)* | Von welcher IP? | Beim Submit |

Alle drei zusammen ergeben den vollständigen Audit-Trail nach § 76 (3) EIWOG.

### C) Datenfluss-Sequenzen

#### Sequenz 1: Public-Submit aus dem Mitglieder-Browser

```
Browser (Mitglied klickt „Einreichen")
   │
   ▼
HTTPS-Request an POST /api/public/applications
   │
   ▼
realIP-Middleware ermittelt die End-User-IP
   │
   ├── X-Forwarded-For von Trusted-Proxy? → erste IP der Chain
   ├── sonst → r.RemoteAddr (Port abgeschnitten)
   ▼
Service-Layer „SubmitApplication" erhält die IP als zusätzlichen Parameter
   │
   ▼
Repository: INSERT mit IP in sepa_mandate_accepted_ip
(zusammen mit sepa_mandate_accepted_at, in derselben Transaktion)
   │
   ▼
Response 201 Created
```

#### Sequenz 2: Externe API von einem EEG-Backend

```
EEG-Integrator-Backend (kennt die End-User-IP)
   │
   ▼
HTTPS-Request an POST /api/external/v1/applications
mit Body: {... antragsdaten ..., "submitterIp": "203.0.113.42"}
   │
   ▼
Handler validiert submitterIp via net.ParseIP
   │
   ├── ungültig → 400 mit Feld-Hinweis
   ├── fehlt → IP bleibt NULL (Audit-Block-Fallback greift)
   ▼
Service-Layer „SubmitExternalApplication" erhält die IP
   │
   ▼
Repository: INSERT inkl. IP
   │
   ▼
Response 201 mit Antrags-ID + Referenz-Nummer
```

#### Sequenz 3: B2B-Mandat-PDF-Generierung (Mail-Pfad + Resync-Pfad)

```
B2B-Mandat-PDF muss generiert werden
(Submit-Pfad ODER PROJ-46 Post-Import ODER PROJ-70 Resync)
   │
   ▼
PDF-Renderer GenerateCompany(data) wird aufgerufen
mit Application-Daten + EEG-Daten
   │
   ▼
Check: Sind alle drei Audit-Felder vorhanden?
   │
   ├── ja → AUDIT-BLOCK
   │     │
   │     ▼
   │   Wortlaut-Variante je nach EEG-require_email_confirmation
   │   („nach Verifizierung" vs. „nach Eingabe")
   │     │
   │     ▼
   │   Kopfzeile + Audit-Text mit Tenant/Datum/Uhrzeit/IP
   │     │
   │     ▼
   │   PDF-Bytes mit Audit-Block am Ende
   │
   └── nein → UNTERSCHRIFTSBLOCK (heutiges Verhalten)
         │
         ▼
       Datum/Ort/Unterschrift-Linie + Box wie bisher
         │
         ▼
       PDF-Bytes mit klassischem Block
```

### D) Tech-Entscheidungen (Begründungen für PM)

| Entscheidung | Begründung |
|---|---|
| **INET statt TEXT** | PostgreSQL validiert IPv4/IPv6 nativ und akzeptiert nichts Ungültiges. Schutz gegen verseuchte Daten ohne Code-Validierung im Backend. |
| **Kein Index auf der neuen Spalte** | Audit-Queries („wie viele Anträge von dieser IP?") sind selten und akzeptieren Sequenz-Scan. Index ohne erkennbaren Nutzen kostet INSERT-Performance. |
| **NULL-fähig statt NOT NULL** | Bestandsanträge wurden vor PROJ-77 erstellt und haben keine IP. NOT NULL würde sie kaputt-migrieren. Der PDF-Renderer fällt für diese Anträge auf den klassischen Unterschriftsblock zurück — saubere Backward-Compat. |
| **Erst-Submit gewinnt** (A1) | Die SEPA-Zustimmung ist historisches Event. Korrektur-Submissions über den `needs_info`-Pfad sind Daten-Korrektur, keine neue Zustimmung. Beweisrechtlich darf der ursprüngliche Zeitpunkt nicht verloren gehen. |
| **ResetImport behält den Audit-Trail** (A2) | Ein zurückgesetzter Import macht die Zustimmung nicht ungültig — das Mitglied hat akzeptiert. Re-Aktivierungen greifen auf denselben Audit-Anker. |
| **PROJ-70-Resync verwendet Original-IP** (A3) | Beim Stammdaten-Resync ändert der Admin Daten, nicht die Zustimmung. Eine Admin-IP im Audit-Block würde zwei semantische Events vermischen und Rechtsklarheit zerstören. |
| **Kopfzeile „Elektronisch erteiltes Mandat (gem. § 76 (3) EIWOG 2010)"** (B2) | Macht die Rechtsgrundlage für Empfänger (Bank, Vorstand) explizit. Vermeidet Verwechslung mit handschriftlicher Unterschriftsstelle. |
| **MultiCell mit Auto-Umbruch** (B1) | Der Audit-Text ist ~250 Zeichen lang — fix-breite Linie würde abschneiden. MultiCell ist Standard-fpdf-Operation, kein Sonderaufwand. |
| **Optionaler `submitterIp` statt Server-IP** (E im Pre-Grilling) | Bei der externen API redet ein EEG-Backend mit unserem Server (Server-zu-Server). Was wir technisch sehen, ist die IP des EEG-Backends — nicht des Mitglieds. Der EEG-Integrator muss die End-User-IP aus seinem ursprünglichen Browser-Request mitgeben. |
| **Excel-Export-Sichtbarkeit via FieldConfig** (E1) | DSGVO-Default „minimaler Export" — Owner entscheidet pro EEG, ob die IP exportiert wird. Konsistent mit dem bestehenden PROJ-15-Konfigurations-Pattern. |
| **Datenschutz-Erklärung um EIWOG-Hinweis ergänzen** (E2) | DSGVO Art. 13/14 verlangt Transparenz über die Verarbeitungs-Rechtsgrundlage. Der bestehende IP-Hinweis im Datenschutz-Text wird um den expliziten EIWOG-Zweck ergänzt. |

### E) Migrationspfad

| Schritt | Beschreibung | Risiko |
|---|---|---|
| 1. Migration 000069 | Neue Spalte `application.sepa_mandate_accepted_ip INET NULL` | Sehr gering — `ALTER ADD COLUMN` mit NULL-Default ist auf PostgreSQL 11+ ein Metadaten-Update, kein Tabellen-Rewrite |
| 2. Backend-Deploy | Alter Code rendert weiter den Unterschriftsblock; neue Spalte existiert, aber niemand schreibt rein. | Kein Verhaltens-Wechsel |
| 3. Submit-Pfad-Deploy (Public + Externe API) | Neue Submissions schreiben die IP. Bestandsanträge bleiben NULL. | Kein Brake — der PDF-Renderer akzeptiert NULL |
| 4. PDF-Renderer-Deploy | Variant-Switch wird aktiv: neue Submissions bekommen den Audit-Block, alte (NULL) den klassischen | Sichtbar im PDF, kein Datenrisiko |
| 5. FieldConfig + Datenschutz-Doku | Excel-Feld-Registrierung + DSGVO-Hinweis im Public-Form | Reine UI- + Doku-Änderung |

**Roll-back-Pfad:** Down-Migration droppt die Spalte. Alle Submissions seit Deploy verlieren ihren Audit-Trail; B2B-PDFs fallen automatisch auf den Unterschriftsblock zurück. Keine Daten-Inkonsistenz, weil die Spalte nirgendwo referenziert wird.

**Reihenfolge ist nicht atomar verzahnt** — Backend kann inkrementell deployt werden:
- Migration alleine ist harmlos.
- Submit-Pfad ohne PDF-Renderer-Update: IPs werden gesammelt, aber im PDF noch unsichtbar (sammelt sich für später).
- PDF-Renderer-Update ohne Submit-Pfad: wirkt sich auf NULL-Bestände nicht aus, neue Submissions hätten dann den klassischen Block. Suboptimal aber nicht kaputt.

### F) Risiken & Trade-offs

| Risiko | Auswirkung | Mitigation |
|---|---|---|
| **PDF-Layout bei IPv6 + langem EEG-Namen** | Audit-Text könnte über die Box-Grenzen hinaus laufen | Snapshot-Test mit beiden Worst-Case-Werten (IPv6-Adresse 39 Zeichen + EEG-Name 60+ Zeichen). Bei Bedarf Schriftgröße auf 7pt oder Box dehnen. |
| **Externe API-Integratoren ohne `submitterIp`** | Bestands-Integratoren senden den Param nicht → Bestandsanträge bekommen klassischen Block; sieht aus wie Hybrid-Bestand | Doku-Update in `api-spec.md` mit „NEU 2026-06-XX"-Hinweis. Aktive Benachrichtigung gibt's nicht (keine Integrator-Mailing-Liste). Integratoren sehen den Unterschied im PDF und können nachfragen. |
| **DSGVO Art. 15 Auskunftspflicht** | IP-Adresse muss bei Auskunft mitgeteilt werden — bisher kein automatisierter Pfad | Excel-Export-FieldConfig erlaubt EEG-Admin, die Spalte im Antrags-Export sichtbar zu machen. Auf Anfrage von Hand mitexportierbar. Eigener „IP-Auskunfts-Endpoint" wäre Overkill. |
| **Trusted-Proxy-Konfig falsch / fehlend** | `realIP` liefert `RemoteAddr` direkt → bei Reverse-Proxy ist das die Proxy-IP, nicht die Mitglied-IP. Audit-Block wäre dann irreführend | Operations-Doku verifizieren, dass `TRUSTED_PROXY_CIDRS` korrekt gesetzt ist. Bei falscher Konfig: Audit-Block zeigt unbrauchbare IP — Owner muss nachpflegen. |
| **PDF-Layout-Test im Snapshot** | Snapshot-Tests vergleichen PDF-Bytes — Layout-Änderungen ohne semantische Änderung würden alle Tests brechen | Snapshot-Tests vorsichtig verfassen — nur die Audit-spezifischen Snapshots, nicht den ganzen Renderer. Bei Layout-Iterationen Snapshots gezielt neu aufnehmen. |
| **Bestandsantrag nach PROJ-77 erneut SEPA-Mandat-PDF (PROJ-70-Resync)** | Audit-Felder waren beim Submit nicht gesetzt → bleiben NULL → Resync-PDF zeigt klassischen Block | Gewolltes Verhalten. Tester muss verstehen: nur Submissions nach Deploy haben den Audit-Block. |

### G) Dependencies (Packages)

**Keine neuen Pakete.** Alle Erweiterungen auf bestehendem Stack:

- `database/sql` mit `net.IP`-Mapping über `lib/pq` (oder `pgx`) — bereits in Verwendung
- `net` aus Go-Standard-Library für `ParseIP`-Validierung
- `gofpdf` für PDF-Rendering — bereits in Verwendung; `MultiCell` ist Standard-API
- Bestehende `realIP`-Middleware ([internal/http/middleware.go](internal/http/middleware.go))

### H) Implementations-Reihenfolge

Keine offenen Architektur-Branches. Implementation wie immer in dieser Reihenfolge:

1. Migration 000069 (Spalte hinzufügen)
2. `shared.Application` um das Feld erweitern
3. Repository: SELECT-Spalte + INSERT/UPDATE-Pfad
4. Service-Layer Public-Submit: IP an Repo durchreichen, Idempotenz-Check
5. Service-Layer Externe API: `submitterIp` aus Body, Validierung
6. PDF-Renderer `GenerateCompany`: Variant-Switch + Audit-Block-Render
7. Resync-Service: kein Eingriff nötig (verwendet bestehende Daten)
8. Excel-Export FieldConfig-Entry
9. Datenschutz-Erklärung (`src/app/datenschutz/page.tsx`)
10. Tests (Migration-Snapshot + PDF-Snapshot + Submit-Tests + Externe-API-Tests + Regression A1/A2/A3)
11. Doku (api-spec.md + user-guide/06 + CHANGELOG + user-guide/changelog)
12. Build + go test + Commit + Push

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
