# Changelog

Alle nennenswerten Änderungen an diesem Projekt werden hier dokumentiert.
Format basiert auf [Keep a Changelog](https://keepachangelog.com/de/1.0.0/).

> Die Versionsnummern im CHANGELOG sind unabhängig von den Git-Tags vergeben,
> da die ursprünglichen Tags nicht konsistent nummeriert wurden.

---

## [Unreleased]

---

## [v1.9.0] - 2026-04-30

### Neu
- **Admin-GUI**: Button „Beitrittsbestätigung herunterladen" in der Antragsdetailansicht (`GET /api/admin/applications/{id}/approval-pdf`) für Status `approved`, `imported`, `import_failed`
- **Mitglieds-Bestätigungs-E-Mail**: Enthält jetzt alle eingegebenen Antragsdaten (Persönliche Daten, Adresse, Bankverbindung, Zählpunkte) und alle erteilten Zustimmungen

### Geändert
- **Beitrittsbestätigung PDF**: Mitgliedsnummer wird als erster Eintrag in MITGLIEDSDATEN angezeigt (kein leeres Leerfeld mehr)
- **Beitrittsbestätigung PDF**: Zustimmungen vollständig — Datenschutz (mit Version), Richtigkeit, SEPA (Checkbox oder „Per E-Mail übermittelt"), Dokumentzustimmungen mit Datum
- **Beitrittsbestätigung PDF**: Statusverlauf-Labels auf Deutsch (z. B. „Eingereicht" statt „submitted")
- **SEPA-Mandat**: Kontoinhaber-Feld wird ausschließlich aus `AccountHolder` befüllt — kein automatischer Fallback auf Vorname/Nachname mehr

### Infrastruktur
- Vitest-Konfiguration auf `.mts` umgestellt (behebt `ERR_REQUIRE_ESM`-Fehler bei `npm test`)
- Dokumentation aktualisiert: `docs/domain-model.md`, `docs/api-spec.md`, Feature-Specs PROJ-21 und PROJ-6

---

## [v1.8.0] - 2026-04-29

### Neu — PROJ-25: Bulk-Aktionen im Admin
- Mehrere Anträge gleichzeitig genehmigen, ablehnen oder zur Prüfung setzen
- Checkboxen pro Zeile + „Alle auswählen"-Checkbox mit indeterminate-State
- Aktionsleiste erscheint bei aktiver Auswahl mit Bestätigungsdialog
- Ergebnis-Zusammenfassung nach Ausführung (X erfolgreich, Y übersprungen)
- Backend: `POST /api/admin/applications/bulk-action` mit Tenant-Isolation; max. 200 Anträge pro Request; ungültige Transitionen werden übersprungen (kein Fehler)

### Neu — PROJ-24: OpenAPI/Swagger Dokumentation
- Interaktive Swagger UI unter `/swagger/` verfügbar
- Alle Admin- und Public-Endpunkte vollständig annotiert (Swaggo)
- Automatische Swagger-Generierung via `swag init` in CI

---

## [v1.7.0] - 2026-04-26

### Neu — PROJ-20: Vollständige Antragsdaten in EEG-Einreichungsbenachrichtigung
- EEG-Betreiber erhält bei jeder Neueinreichung alle Antragsdaten per E-Mail
- Felder: Mitgliedstyp, Name/Firma, Adresse, Kontakt, IBAN, SEPA-Ermächtigung, Zählpunkte, konfigurierbare Felder
- Konfigurierbare Felder werden nur angezeigt wenn nicht `hidden` und befüllt
- Optionaler Admin-Link zur Detailansicht (via `ADMIN_BASE_URL`-Umgebungsvariable)

### Neu — PROJ-21: Genehmigungs-Benachrichtigung mit Beitrittsbestätigung PDF
- Bei Status-Übergang → `approved` erhält die EEG automatisch eine E-Mail mit PDF-Anhang
- PDF „Beitrittsbestätigung" enthält: Mitgliedsdaten, Bankverbindung, Zählpunkte, Zustimmungen, Statusverlauf, konfigurierbare Felder
- PDF-Generierung schlägt fehl → E-Mail wird trotzdem gesendet (mit Hinweistext); Status-Übergang bleibt gültig
- Re-Approval (`import_failed → approved`) sendet erneut eine E-Mail

---

## [v1.6.0] - 2026-04-25

### Neu — PROJ-9: EEG-spezifische Rechtsdokumente
- Admin kann beliebige Rechtsdokumente pro EEG konfigurieren (Satzung, AGB usw.)
- Mitglied muss Pflichtdokumente vor Einreichung bestätigen
- Zustimmungen werden als unveränderliche Snapshots gespeichert (`document_consent`)
- Max. 10 Dokumente pro EEG; sortierbar per Drag-and-Drop

### Neu — PROJ-16: Cloudflare Turnstile Spam-Schutz
- Öffentliches Registrierungsformular mit Turnstile-CAPTCHA geschützt
- Aktivierung via `TURNSTILE_SECRET_KEY`-Umgebungsvariable (fehlt → deaktiviert)

### Neu — PROJ-17: Excel-Export für eegFaktura-Import
- Admin kann Antrag als `.xlsx`-Datei exportieren (`GET /api/admin/applications/{id}/export/excel`)
- Datei im eegFaktura-Importformat (36 Spalten, eine Zeile pro Zählpunkt)
- Nur für Status `approved`, `imported`, `import_failed`

### Neu — PROJ-18: Datenschutzerklärung & Central Policy Toggle
- Zentrale Datenschutzerklärung (Betreiber-Policy) über Umgebungsvariablen konfigurierbar (`CENTRAL_POLICY_TITLE`, `CENTRAL_POLICY_URL`)
- Pro EEG einstellbar, ob die zentrale Policy im Formular angezeigt wird (`showCentralPolicy`)
- EEGs mit eigener Datenschutzerklärung können die zentrale Policy ausblenden

### Neu — PROJ-19: Manuelle Aktivierung der Registrierung
- Neue EEGs sind standardmäßig inaktiv (`is_active = false`)
- Admin kann Registrierung pro EEG aktivieren/deaktivieren (Settings-Seite)
- Inaktive EEGs: öffentliches Formular liefert `410 Gone`

---

## [v1.5.0] - 2026-04-24

### Neu — PROJ-12: SEPA-Lastschriftmandat PDF
- Automatische Generierung eines SEPA-Lastschriftmandats als PDF-Anhang in der Mitglieds-Bestätigungs-E-Mail
- Aktivierung pro EEG via `sepaMandateEnabled`-Einstellung
- Unterstützt CORE- und B2B-Mandat
- Kann auch per E-Mail zugesandt werden (`sepa_mandate_enabled = false`): Hinweis im PDF und in der Bestätigungs-E-Mail

### Neu — PROJ-13: Externe Registrierungs-API
- `POST /api/external/v1/applications` — Anträge direkt aus externen Systemen einreichen
- API-Key-Authentifizierung (kein Keycloak); Key pro EEG generierbar/widerrufbar in den Admin-Settings
- Rate Limiting: 10 Requests / 60 Sekunden (Burst) + 200 Einreichungen / Tag (Quota)

### Neu — PROJ-14: SEPA-Firmenlastschriftmandat
- Für Mitglieder vom Typ `company` / `association` kann ein SEPA-B2B-Mandat statt des Standard-CORE-Mandats generiert werden
- Steuerbar über EEG-Einstellung `useCompanySEPAMandate`

### Neu — PROJ-15: Konfigurierbare Felder Erweiterungen
- Neuer Feld-Status `admin_only`: Feld ist im öffentlichen Formular verborgen, wird aber mit einem konfigurierten Admin-Standardwert automatisch befüllt
- Zählpunktfelder konfigurierbar: `transformer`, `installation_number`, `installation_name`

---

## [v1.4.0] - 2026-04-23

### Neu — PROJ-11: Konfigurierbarer Einleitungstext
- Admin kann pro EEG einen Einleitungstext für das Registrierungsformular hinterlegen (HTML, sanitisiert)
- Wird im öffentlichen Formular über dem Antragsformular angezeigt

---

## [v1.3.0] - 2026-04-22

### Neu — PROJ-8: Konfigurierbare Felder pro EEG
- Admin kann pro EEG konfigurieren, welche optionalen Felder im Registrierungsformular sichtbar, versteckt oder Pflicht sind
- Konfigurierbare Felder: `phone`, `birth_date`, `uid_number`, `membership_start_date`, `persons_in_household`, `consumption_previous_year`, `consumption_forecast`, `feed_in_forecast`, `pv_power_kwp`, `heat_pump`, `electric_vehicle`, `electric_hot_water`

---

## [v1.2.0] - 2026-04-21

### Neu — PROJ-6: E-Mail-Benachrichtigungen
- Mitglieds-Bestätigungs-E-Mail nach erfolgreicher Einreichung
- EEG-Benachrichtigungs-E-Mail an `contact_email` der EEG
- Asynchroner Versand (kein Blockieren der Einreichung bei SMTP-Fehler)
- Resend-Funktion im Admin: „Bestätigung erneut senden"
- Konfiguration via `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`

---

## [v1.1.0] - 2026-04-20

### Neu — PROJ-5: Keycloak-gesicherte Admin-Oberfläche
- Admin-Bereich erfordert Keycloak-Login (JWT Bearer Token)
- Tenant-Isolation: Admins sehen nur Anträge ihrer eigenen EEGs
- Superuser-Flag für EEG-übergreifenden Zugriff

### Neu — PROJ-7: Mitgliedstypen
- Unterstützung für fünf Mitgliedstypen: Privatperson, Landwirt, Gemeinde, Unternehmen, Verein
- Typenspezifische Felder (Firmenname, UID-Nummer, Firmenbuchnummer)
- Kompakte Select-UI im Registrierungsformular

---

## [v1.0.0] - 2026-04-19

### Neu — PROJ-1: Öffentliche Registrierung
- Öffentliches Registrierungsformular unter `/register/{rc_number}`
- Antragstellung mit Personendaten, Adresse, IBAN, Zählpunkten
- Mehrschrittiges Formular mit Validierung (Frontend + Backend)
- Antragsstatus: `draft` → `submitted`

### Neu — PROJ-2: Admin-Review
- Admin kann Anträge einsehen, bearbeiten und Status ändern
- Status-Workflow: `submitted → under_review → approved / rejected / needs_info`
- Admin-Notiz und Rückfrage-Grund pro Antrag

### Neu — PROJ-3: Admin-Frontend-UI
- Antragsübersicht mit Filter und Pagination
- Detailansicht mit vollständigen Antragsdaten
- Status-Aktionen direkt aus der Detailansicht
