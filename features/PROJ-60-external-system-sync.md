# PROJ-60 — Datenweiterleitung an externe Systeme nach Import

## Status: Planned (Idea — Erst-Konzept, noch ohne ausformulierte Requirements)
**Created:** 2026-05-23
**Last Updated:** 2026-05-23 (MVP-Pivot: Excel-Export statt Zoho)
**Quelle:** Owner-Anforderung

## Dependencies
- Requires: PROJ-4 (Core Import) — Daten-Export erfolgt nach erfolgreichem Core-Import
- Berührt: PROJ-17 (Excel-Export für eegFaktura-Import) — anderer Use-Case, gleiche `excelize`-Library wiederverwendbar
- Berührt: PROJ-46 (Post-Import-Stati) — Trigger / Filter könnte auf verschiedene Status ausgerichtet sein
- Berührt: PROJ-8 (konfigurierbare Felder pro EEG) — pro EEG aktivierbar
- Berührt: DSGVO-Auftragsverarbeitungs-Gefüge (bei Phase 2 ggü. externen Diensten)

---

## Idee in einem Satz

Nach erfolgreichem Import eines Mitglieds in den eegFaktura-Core
sollen Mitgliedsdaten in **externe Systeme** weitergeleitet werden,
damit die EEG das Mitglied dort parallel pflegen kann.

## Hintergrund

EEG-Vereine pflegen ihre Mitglieder teils nicht nur im
eegFaktura-Core, sondern parallel in einem eigenen System — z. B.
einem CRM (Zoho, HubSpot, Salesforce), Newsletter-Tool, Vereins-
Datenbank oder schlicht einer Excel-Datei für interne Auswertungen.

Aktuell muss der EEG-Admin diese parallele Pflege manuell machen:
- Mitglied im Onboarding-System genehmigen → Import in Core
- EEG-Admin muss die Daten **manuell** ins externe System übertragen

Das ist fehleranfällig, ineffizient und nicht skalierbar.

## Phasen-Strategie

| Phase | Use Case | Mechanik | Aufwand |
|---|---|---|---|
| **Phase 1 (MVP)** | **Excel-Export mit konfigurierbarem Spalten-Mapping** | Pull: Admin klickt „Export" oder Cron erzeugt periodisch | ~3–4 Tage |
| **Phase 2** | CRM-API-Integration (Zoho als erster Adapter) | Push: real-time bei Status-Übergang | ~5–7 Tage |
| **Phase 3 (optional)** | Weitere CRMs (HubSpot, Salesforce, Pipedrive) | gleicher Adapter-Pattern | je ~1–2 Tage pro Adapter |

Phase 1 zuerst, weil:
- **Niedrigere Komplexität** (kein OAuth, kein externer Service, keine Webhook-Reconciliation)
- **Universell anschlussfähig** — jedes CRM/Newsletter-Tool kann Excel importieren
- **DSGVO-einfacher** (Daten bleiben „im Haus" der EEG, kein Third-Party-Datentransfer)
- **Admin kann sofort experimentieren** mit Mapping-Konfiguration, ohne dass jemand einen Zoho-Vertrag haben muss
- **Liefert sofort Nutzen** für EEGs, die kein CRM nutzen (sondern z. B. einfache Datei-basierte Pflege)

Phase 2 (Zoho-Adapter) wird der nächste Schritt, sobald Phase 1
produktiv läuft und die ersten EEGs nach Real-Time-API-Sync fragen.

---

# Phase 1 (MVP): Excel-Export mit konfigurierbarem Spalten-Mapping

## Skizze

### Architektur

```
Admin-UI (EEG-Settings)
  ├─ Sektion „Daten-Export"
  ├─ Liste der gespeicherten Export-Templates
  ├─ Template anlegen/editieren:
  │    ├─ Name (z. B. „CRM-Monatsexport")
  │    ├─ Format: XLSX | CSV
  │    ├─ Spalten-Liste:
  │    │    ├─ Spalte 1: Header-Text + Onboarding-Feld + Format
  │    │    ├─ Spalte 2: …
  │    │    └─ + Hinzufügen
  │    ├─ Filter (optional):
  │    │    ├─ Status (imported / activated / …)
  │    │    ├─ Zeitraum (importiert seit Datum X)
  │    │    └─ Mitgliedstyp
  │    └─ Speichern
  └─ Aktionen pro Template:
       ├─ „Jetzt exportieren" → Download
       ├─ „E-Mail an mich senden"
       └─ „Periodisch versenden" (Cron, V2-Idee)

Backend
  ├─ Tabelle: export_template (pro EEG mehrere Templates)
  ├─ Service: ExportService.generate(templateId, eegId)
  │    → liest Mitglieder gemäß Filter
  │    → mappt Felder gemäß Spalten-Konfig
  │    → erzeugt XLSX/CSV via excelize (PROJ-17-Pattern)
  │    → Download-Stream oder Speicherung im DOC-Archiv
  └─ HTTP-Endpoint: GET /admin/export-templates/{id}/run → Datei-Download
```

### Konfigurations-Modell

```
export_template
  id              UUID PK
  rc_number       FK auf registration_entrypoint
  name            TEXT (z. B. „CRM-Monatsexport")
  format          ENUM (xlsx | csv)
  columns         JSONB (geordnete Liste von Spalten-Definitionen)
  filter          JSONB (Status-Filter, Zeitraum, Typ-Filter)
  schedule        TEXT NULL (Cron-Expression für V2-Auto-Versand)
  recipient_email TEXT NULL (für V2-Auto-Versand)
  created_at, updated_at
  
column-Schema in JSONB:
[
  {
    "header": "Vorname",
    "field": "firstname",
    "format": "string" | "date_dmy" | "boolean_jn" | "enum_label" | …
  },
  {
    "header": "Mitgliedsnummer",
    "field": "member_number",
    "format": "string"
  },
  …
]
```

### Verfügbare Felder zur Auswahl

Das Mapping-UI bietet eine Dropdown-Liste der exportierbaren
Onboarding-Felder. Vorschlag der Kategorien:

**Stammdaten:**
- Mitgliedstyp (`member_type`)
- Anrede (`titel`)
- Vorname (`firstname`)
- Nachname (`lastname`)
- Titel nach (`titel_nach`)
- Firmenname (`company_name`)
- UID-Nummer (`uid_number`)
- Firmenbuch-Nummer (`register_number`)
- Geburtsdatum (`birth_date`)

**Kontakt:**
- E-Mail (`email`)
- Telefon (`phone`)

**Adresse:**
- Straße + Hausnummer (`resident_street` + `_number`)
- PLZ (`resident_zip`)
- Ort (`resident_city`)

**Bank / Zahlung:**
- IBAN (`iban`) — **mit Confirm-Dialog beim Hinzufügen** wegen DSGVO-Sensibilität
- Kontoinhaber (`account_holder`)
- Einzugsart (`einzugsart`)

**EEG-spezifisch:**
- RC-Nummer (`rc_number`)
- Referenznummer (`reference_number`)
- Mitgliedsnummer (`member_number`)
- Beitrittsdatum (`membership_start_date`)
- Status (`status`)
- Importiert am (`imported_at`)
- Aktiviert am (`activated_at`)

**Zählpunkte** (multi-value — pro Mitglied evtl. mehrere):
- Anzahl Zählpunkte
- Zählpunkt-Nummer(n) (komma-getrennt oder separate Zeile pro Zählpunkt?)
- Direction (CONSUMPTION/PRODUCTION)
- Adresse pro Zählpunkt

**Konfigurierbare Felder** (PROJ-8, je EEG aktiv): nur die exportieren,
die die EEG auch erfasst.

### Wert-Transformationen (V1 minimal)

| Format-Tag | Wirkung | Beispiel |
|---|---|---|
| `string` | unverändert | „Max Mustermann" |
| `date_dmy` | Datum als DD.MM.YYYY | 23.05.2026 |
| `date_iso` | Datum als YYYY-MM-DD | 2026-05-23 |
| `boolean_jn` | `true`/`false` → „Ja"/„Nein" | „Ja" |
| `boolean_10` | `true`/`false` → 1/0 | 1 |
| `enum_label` | Member-Type-Wert → lesbares Label | „Privatperson" statt „private" |
| `number_de` | Zahlen mit Komma als Dezimal | „1.234,56" |

Wenig — V1 ist „basics decken". V2 könnte custom transformations bringen.

### Excel/CSV-Generierung

PROJ-17 hat bereits `internal/excel/generator.go` mit `excelize` —
gleiche Library für PROJ-60 wiederverwenden, aber mit dynamischen
Spalten basierend auf Template statt hardcoded.

CSV mit Standard-Konventionen:
- UTF-8 mit BOM (für Excel-Kompatibilität)
- Semikolon-Separator (DACH-Standard für deutsche Excel-Installationen)
- Quotes um Werte mit Sonderzeichen

### Trigger / Versand

**V1 (MVP):**
- **Manuell:** Admin klickt im UI „Jetzt exportieren" → Download
- **Auf Anfrage:** Admin klickt „E-Mail an mich" → eine Mail mit Datei-Anhang an die hinterlegte E-Mail

**V2 (später, falls Bedarf):**
- **Cron:** Schedule pro Template (z. B. „1. jeden Monats"), Datei wird automatisch generiert und an Recipient-E-Mail geschickt
- **API:** externer Service kann den Export per HTTP-Endpoint anfordern (Auth via Admin-API-Key)

### Speicherung der Exports

**Pro generierter Export wird** der Zeitpunkt + Datei optional im DOC-Archiv (FreeFinance via DOC-API, sobald Billing-Stack läuft — siehe PROJ-Pricing-Memo) oder lokal im Onboarding-Filesystem für Audit/Wiederholbarkeit abgelegt. V1: vermutlich nicht persistieren — Admin lädt herunter, kann jederzeit neu generieren.

## Acceptance Criteria (vorläufig)

### Template-Verwaltung
- [ ] EEG-Admin sieht im Admin-UI eine neue Sektion „Daten-Export" mit Liste der Templates
- [ ] Admin kann ein neues Template anlegen mit Name, Format, Spalten-Liste, Filter
- [ ] Admin kann ein bestehendes Template editieren oder löschen
- [ ] Templates sind pro EEG separat (kein Cross-EEG-Sharing)
- [ ] Pro EEG mehrere Templates möglich (z. B. „CRM-Monatsexport", „Adressliste für Newsletter", „Bank-Stamm­daten-Backup")

### Spalten-Mapping-UI
- [ ] Dropdown mit allen verfügbaren Feldern, gruppiert nach Kategorie
- [ ] Pro Spalte: Header-Text frei wählbar, Feld-Auswahl, Format-Auswahl
- [ ] Drag-and-Drop oder Auf/Ab-Buttons zum Sortieren der Spalten
- [ ] Live-Preview der ersten 5 Zeilen mit aktuellem Template

### Filter
- [ ] Status-Filter (Multi-Select: imported, activated, ready_for_activation, etc.)
- [ ] Zeitraum (`imported_at` zwischen X und Y)
- [ ] Mitgliedstyp-Filter (Multi-Select)
- [ ] Filter sind optional — leerer Filter = alle Mitglieder dieser EEG

### Export-Ausführung
- [ ] „Jetzt exportieren"-Button erzeugt Datei und triggert Download
- [ ] XLSX und CSV beide unterstützt
- [ ] Dateiname: `{eeg_name}-{template_name}-{YYYY-MM-DD}.{xlsx|csv}`
- [ ] „E-Mail an mich"-Button erzeugt Datei und versendet sie an die hinterlegte Admin-E-Mail

### Datenschutz
- [ ] Beim Hinzufügen sensitiver Felder (IBAN, Geburtsdatum) erscheint ein Hinweis: „Sie tragen Verantwortung dafür, wo und wie diese Daten weiterverarbeitet werden — beachten Sie DSGVO Art. 32 (Sicherheit der Verarbeitung)."
- [ ] Admin-Aktion (Template erstellt, exportiert) wird im `status_log` oder eigenem Audit-Log festgehalten
- [ ] Cross-Tenant-Schutz: EEG-Admin kann nur Daten ihrer eigenen EEG exportieren (via `checkTenantAccess`)

### Zählpunkte (Multi-Value)
- [ ] Im Mapping-UI Klärung, wie mit Multi-Value-Feldern umgegangen wird:
  - Option A: pro Zählpunkt **eine Spalte** mit komma-getrennten Werten („AT0031000…, AT0031000…")
  - Option B: pro Zählpunkt **eine Zeile** (Mitgliedsdaten wiederholt) — relational sauber, EEG-Admin muss sich für eines entscheiden
  - Vorschlag V1: Option A als Standard, Option B als Template-Toggle für fortgeschrittene Nutzer

## Edge Cases (vorläufig)

- Was passiert, wenn ein Mitglied zwischen Template-Erstellung und Export gelöscht wird? → Nur Mitglieder mit aktuellen Daten exportieren (LEFT JOIN ist falsch — INNER JOIN ist korrekt; oder: anzeigen „Mitglied X wurde gelöscht und nicht enthalten")
- Was, wenn ein Feld in einem alten Template entfernt wurde (z. B. konfigurierbares Feld wurde im EEG-Setting hidden gestellt)? → Spalte wird leer / mit „—" gefüllt; Template-Edit-UI zeigt Warnung
- Was, wenn der Export sehr groß wird (1.000+ Mitglieder bei großer EEG)? → Streaming-Generierung (kein In-Memory-Buffer), Limit der Filter-Filterung beachten
- Was passiert mit Sonderzeichen in Mitgliedsdaten (z. B. Umlaute, Semikolon im Namen)? → Korrektes Escaping in CSV, UTF-8-Encoding mit BOM
- Was, wenn das Template gerade vom Admin editiert wird und gleichzeitig ein zweiter Admin den Export auslöst? → Last-Write-Wins, kein Locking nötig (Templates sind selten und nicht hochfrequent geändert)
- Was passiert bei mehreren parallelen Export-Anfragen für dieselbe EEG? → unkritisch (read-only Operation), evtl. Rate-Limit als Schutz vor Missbrauch

## Technical Requirements (vorläufig)

- **DB-Migration** für `export_template`-Tabelle
- **excelize-Wiederverwendung** aus PROJ-17, aber als generischer Renderer mit Spalten-Definitionen statt hardcoded
- **Streaming-Output** für große Exports (HTTP-Chunked-Transfer)
- **Audit-Log** für Export-Aktionen (welcher Admin, welches Template, wann, wie viele Zeilen)
- **Mail-Versand** wiederverwendet aus PROJ-6 (Postal)
- **Front-End:** shadcn-Komponenten (Select, Input, Button, Table) — kein neues UI-Framework

---

# Phase 2 (später): CRM-API-Integration

## Skizze

Wenn Excel-Export-MVP läuft und die ersten EEGs nach Real-Time-Push
fragen: Erweiterung zur **Push-basierten Sync** mit Plugin-/Adapter-
Pattern.

### Architektur (Plugin-Pattern)

```
ExternalSystemAdapter (Interface)
  ├─ ExcelExportAdapter (existiert nach Phase 1)
  ├─ ZohoCRMAdapter (Phase 2, erster echter CRM-Adapter)
  └─ HubSpotAdapter, SalesforceAdapter, … (Phase 3+)

Jeder Adapter implementiert:
  - configure(credentials, mapping)
  - sync(member) → Push-Aktion
  - mode: pull (Excel) | push (CRM)
```

### Zoho CRM als erster Push-Adapter

- **Auth:** OAuth2 mit Refresh-Tokens, per EEG gespeichert (verschlüsselt)
- **Trigger:** asynchron bei Status-Übergang zu `imported` (oder konfigurierbar)
- **Mapping:** ähnliches Spalten-Konzept wie Excel-Template, aber mit Zoho-Ziel-Feldern (Standard: Contact-Modul, customizable)
- **Retry-Logik:** exponential backoff, max 3 Retries, Admin-Notification bei endgültigem Fehler
- **Idempotenz:** via `EEG_Onboarding_ID` als Custom-Field in Zoho (Suchschlüssel für Re-Sync)
- **Daten-Modell-Wiederverwendung:** das Excel-Mapping-Konzept aus Phase 1 lässt sich für Zoho-Field-Mapping wiederverwenden (Source-Feld → Target-Feld, mit Format-Transformation)

### Open Questions für Phase 2 (vor späterem `/requirements`-Lauf)

- **MVP-Umfang Phase 2:** nur Zoho oder gleich generisches Plugin-Framework?
- **Bidirektionalität:** CRM-Änderungen zurück ins Onboarding-System? Vermutlich Out-of-Scope auch in Phase 2.
- **DSGVO-Verantwortung:** EEG muss eigenen AVV mit Zoho haben. Wir sind als Onboarding-Anbieter „Datendurchleitung", nicht Auftragsverarbeiter für die Zoho-Daten — Hinweistext im Setup-Flow.
- **Field-Mapping**: Wiederverwendung des Excel-Template-Konzepts? Oder eigenes UI für CRM-Mapping?
- **Konflikt-Strategie**: Mitglied existiert im CRM schon (z. B. manuell angelegt) — skip / overwrite / merge / Admin-Entscheidung?

---

## Notes

- **Excel-Export ist universell anschlussfähig** und liefert sofort Nutzen, ohne dass eine EEG einen CRM-Vertrag haben muss
- **Phase 2 (Zoho) bleibt im Auge**, aber separater PROJ oder Erweiterung von PROJ-60 — Entscheidung wenn Phase 1 läuft
- **PROJ-17 (Excel-Export für eegFaktura-Import)** und PROJ-60 nutzen die gleiche Library, aber sind funktional getrennt: PROJ-17 ist ein **hardcoded Format** für genau einen Anwendungsfall (Core-Import), PROJ-60 ist **frei konfigurierbar** für beliebige externe Konsumenten
- **Wenn PROJ-55 (Self-Service-Portal) kommt:** kann der Excel-Export auch dort als „Mein Verein bekommt automatisch jeden Monat eine Mitglieder-Liste" auftauchen — Synergie-Effekt
- **Vermarktungs-Argument**: für EEGs, die Excel-basiert arbeiten (vermutlich die Mehrheit der kleineren Vereine), ist der Excel-Export bereits ein vollwertiges „CRM-Sync"-Feature ohne weitere Tool-Pflicht

## Nächster Schritt

Bei tatsächlicher Aufnahme der Spec: `/requirements`-Lauf mit dieser
Datei als Ausgangspunkt für **Phase 1**, um zu klären:
1. **Mapping-UI**: Drag-Drop vs. Up/Down-Buttons, Live-Preview-Tiefe
2. **Filter-Granularität**: welche Filter sind V1, welche V2
3. **Multi-Value-Handling für Zählpunkte**: Spalte mit Liste vs. Zeile pro Zählpunkt
4. **DSGVO-Hinweistexte**: exakter Wortlaut für IBAN-/Geburtsdatum-Warnung
5. **Persistierung von Exports**: V1 fly-by oder schon Archivierung im DOC-Archiv

Dann `/architecture` für die DB-Struktur + excelize-Generalisierung,
dann `/backend` + `/frontend` für die Implementierung.

Phase 2 (Zoho) wird separat angegangen, sobald Phase 1 in Produktion
und erste EEG-Bedarfe nach Real-Time-Sync auftreten.
