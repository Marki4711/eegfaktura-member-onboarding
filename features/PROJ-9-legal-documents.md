# PROJ-9: EEG-spezifische Rechtsdokumente mit granularer Zustimmung

## Status: In Progress
**Created:** 2026-04-21
**Last Updated:** 2026-04-25

## Dependencies
- Requires: PROJ-1 (Public Registration) — Zustimmung erfolgt im Registrierungsformular
- Requires: PROJ-2 (Admin Review) — Admin verwaltet die Dokumentenliste der EEG
- Requires: PROJ-5 (Keycloak-secured Admin Area) — Verwaltung nur für authentifizierte Admins

## User Stories

- Als EEG-Administrator möchte ich eine Liste von Rechtsdokumenten (AGB, Datenschutzerklärung, Statuten) für meine EEG hinterlegen können, damit Neumitglieder beim Beitritt gezielt zustimmen.
- Als EEG-Administrator möchte ich festlegen können, ob die Zustimmung zu einem Dokument verpflichtend oder freiwillig ist, damit ich rechtliche Anforderungen abbilden kann.
- Als EEG-Administrator möchte ich die Reihenfolge der angezeigten Dokumente steuern können, damit wichtige Dokumente zuerst erscheinen.
- Als Mitglied möchte ich für jedes Rechtsdokument eine eigene Checkbox sehen und den Link direkt öffnen können, damit ich weiß, womit ich zustimme.
- Als Mitglied möchte ich zusätzlich zur EEG-spezifischen Liste immer die zentrale Datenschutzerklärung des Tool-Betreibers sehen und zustimmen, damit der Betrieb des Tools transparent ist.
- Als Admin möchte ich im Antrag nachvollziehen können, welchen Dokumenten das Mitglied zugestimmt hat (Titel, URL, Zeitstempel), damit die Zustimmung nachweisbar ist.

## Acceptance Criteria

- [ ] Pro EEG kann eine geordnete Liste von Dokumenten angelegt werden (Titel, URL, Pflicht ja/nein, Reihenfolge)
- [ ] Die Dokumentenliste wird über den `/api/public/registration/{rc_number}` Endpunkt mitgeliefert
- [ ] Im Registrierungsformular wird pro EEG-Dokument eine eigene Checkbox mit verlinktem Titel angezeigt
- [ ] Pflichtdokumente blockieren das Absenden des Formulars wenn nicht angehakt
- [ ] Die zentrale Datenschutzerklärung des Tool-Betreibers wird immer angezeigt und ist immer Pflicht
- [ ] Beim Speichern des Antrags wird pro zugestimmtem Dokument gespeichert: Titel, URL, Zeitstempel
- [ ] Die gespeicherten Zustimmungen sind in der Admin-Detailansicht eines Antrags sichtbar
- [ ] Ein Admin kann Dokumente seiner EEG(s) hinzufügen, bearbeiten, löschen und sortieren
- [ ] Das Löschen eines Dokuments beeinflusst keine bereits gespeicherten Zustimmungen

## Edge Cases

- Was passiert, wenn eine EEG keine eigenen Dokumente hinterlegt hat? → Nur die zentrale Datenschutzerklärung wird angezeigt, Formular bleibt funktionsfähig.
- Was passiert, wenn ein Dokument-Link nicht erreichbar ist? → Das Formular zeigt den Link trotzdem an; die Erreichbarkeit wird nicht geprüft.
- **Dokumentenversionen:** Beim Einreichen wird ein unveränderlicher Snapshot aus Titel, URL und Zeitstempel gespeichert. Kein Hash — der URL-Snapshot mit Zeitstempel ist der Nachweis. Admins wird empfohlen, versionierte URLs zu verwenden (z.B. `/agb-v2.pdf`).
- Was passiert, wenn ein Dokument nach Einreichung eines Antrags geändert oder gelöscht wird? → Bereits gespeicherte Zustimmungen bleiben unverändert (Snapshot zum Zeitpunkt der Einreichung).
- Was passiert, wenn ein optionales Dokument nicht angehakt wird? → Antrag kann trotzdem eingereicht werden; keine Zustimmung wird für dieses Dokument gespeichert.
- Was passiert, wenn die URL eines Dokuments sehr lang ist? → URL wird vollständig gespeichert, im Formular aber nur der Titel verlinkt angezeigt.

## Technical Requirements

- Zustimmungen werden als unveränderlicher Snapshot gespeichert (Titel + URL zum Zeitpunkt der Einreichung)
- Die zentrale Datenschutzerklärung ist nicht in der Datenbank konfiguriert, sondern im Code/Konfiguration hinterlegt
- Reihenfolge der Dokumente ist explizit steuerbar (z.B. über ein `sort_order` Feld)
- Maximale Anzahl Dokumente pro EEG: reasonable limit (z.B. 10)

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)

### Übersicht

PROJ-9 ist ein Full-Stack-Feature: neues Datenbankschema, Backend-Erweiterungen (Admin-CRUD + öffentliche API + Consent-Speicherung) und Frontend-Erweiterungen (Admin-Verwaltungsseite + Registrierungsformular + Antragsdetail).

---

### Neue Datenbanktabellen (Migration 000018)

**`member_onboarding.legal_document`** — Dokumentenliste pro EEG

| Feld | Typ | Bedeutung |
|------|-----|-----------|
| `id` | UUID | Primärschlüssel |
| `rc_number` | TEXT | Fremdschlüssel → `registration_entrypoint(rc_number)`, ON DELETE CASCADE |
| `title` | TEXT | Angezeigter Titel im Formular |
| `url` | TEXT | Link zum Dokument |
| `required` | BOOLEAN | Pflichtfeld ja/nein |
| `sort_order` | INTEGER | Reihenfolge der Anzeige (aufsteigend) |
| `created_at` | TIMESTAMP | |
| `updated_at` | TIMESTAMP | |

Regeln:
- Max. 10 Dokumente pro EEG (in Anwendungscode erzwungen)
- `sort_order` ist eindeutig pro `rc_number`

**`member_onboarding.document_consent`** — Zustimmungs-Snapshot pro Antrag

| Feld | Typ | Bedeutung |
|------|-----|-----------|
| `id` | UUID | Primärschlüssel |
| `application_id` | UUID | Fremdschlüssel → `application(id)`, ON DELETE CASCADE |
| `title` | TEXT | Snapshot des Titels zum Einreichzeitpunkt |
| `url` | TEXT | Snapshot der URL zum Einreichzeitpunkt |
| `is_central_policy` | BOOLEAN | true = zentrale Datenschutzerklärung des Tool-Betreibers |
| `consented_at` | TIMESTAMP | Zeitpunkt der Zustimmung (= Einreichzeitpunkt des Antrags) |

Regeln:
- Unveränderlicher Snapshot — wird nie nachträglich geändert
- Kein Fremdschlüssel auf `legal_document` (Dokument kann nach Einreichung gelöscht werden)

---

### Komponenten-Struktur

**Admin-Seite — Einstellungen (EEG-Tab, neu: Reiter „Rechtsdokumente")**
```
AdminEEGSettingsPage
+-- [bestehend] EEGSettingsEditor (Stammdaten)
+-- [bestehend] FieldConfigEditor (Felder)
+-- [NEU] LegalDocumentsEditor
    +-- DokumentenListe
    |   +-- DokumentZeile (Titel, URL, Pflicht-Toggle, Reihenfolge-Pfeile, Löschen)
    |   +-- DokumentZeile ...
    +-- "Dokument hinzufügen"-Button
    +-- Hinweis auf zentrale Datenschutzerklärung (fix, nicht editierbar)
```

**Registrierungsformular (Erweiterung am Ende des Formulars)**
```
RegistrationForm
+-- [bestehend] Alle bisherigen Felder ...
+-- [NEU] RechtsdokumenteAbschnitt
    +-- Checkbox: [Pflicht] "Ich stimme den [AGB] zu"  ← Link öffnet Dokument
    +-- Checkbox: [Optional] "Ich stimme der [Datenschutz EEG] zu"
    +-- Checkbox: [Pflicht] "Ich stimme der [Datenschutzerklärung] zu"  ← zentrale Policy
+-- Absenden-Button (blockiert solange Pflicht-Checkboxen nicht angehakt)
```

**Admin-Antragsdetail (Erweiterung)**
```
AdminApplicationDetail
+-- [bestehend] Personendaten, Status, Zählpunkte ...
+-- [NEU] ZustimmungenAbschnitt
    +-- ZustimmungsZeile: Titel | URL | Zeitstempel | Hash (gekürzt)
    +-- ZustimmungsZeile ...
```

---

### API-Änderungen

**Öffentliche API — Erweiterung bestehender Endpunkte**

| Endpunkt | Änderung |
|----------|----------|
| `GET /api/public/registration/{rc_number}` | Antwort enthält neu: `legalDocuments: [{id, title, url, required, sortOrder}]` |
| `POST /api/public/applications` | Request-Body enthält neu: `consents: [{documentId?, title, url, isCentralPolicy}]`; beim Einreichen holt das Backend die Hashes serverseitig |

**Admin API — neue Endpunkte**

| Methode | Pfad | Beschreibung |
|---------|------|--------------|
| `GET` | `/api/admin/legal-documents?rc_number=` | Alle Dokumente einer EEG abrufen |
| `POST` | `/api/admin/legal-documents?rc_number=` | Neues Dokument anlegen |
| `PUT` | `/api/admin/legal-documents/{id}` | Dokument bearbeiten |
| `DELETE` | `/api/admin/legal-documents/{id}` | Dokument löschen |
| `PUT` | `/api/admin/legal-documents/reorder?rc_number=` | Reihenfolge aller Dokumente auf einmal aktualisieren |

`GET /api/admin/applications/{id}` wird um ein `consents`-Array erweitert.

---

### Backend-Pakete

Keine neuen Go-Pakete notwendig — die neue Logik passt in die bestehende Struktur:

| Datei | Inhalt |
|-------|--------|
| `internal/application/legal_document_repo.go` | CRUD für `legal_document`-Tabelle |
| `internal/application/document_consent_repo.go` | INSERT + SELECT für `document_consent` |
| `internal/http/admin.go` | 5 neue Handler-Methoden für Legal-Document-Admin-API |
| `internal/http/registration.go` | Erweiterung: Consents aus Request lesen, Hash berechnen, speichern |

---

### Zentrale Datenschutzerklärung

- Titel und URL werden als Umgebungsvariablen konfiguriert: `CENTRAL_POLICY_TITLE`, `CENTRAL_POLICY_URL`
- Sind sie nicht gesetzt, wird eine sinnvolle Default-URL verwendet
- Das Frontend erhält die zentrale Policy als festes letztes Element in `legalDocuments` (mit `isCentralPolicy: true`)
- Der Backend-Endpunkt `GET /api/public/registration/{rc_number}` fügt sie immer ans Ende der Liste an

---

### Tenant-Isolation

- Admins dürfen nur Dokumente ihrer eigenen EEG(s) verwalten (via `rc_number` aus Keycloak JWT)
- Die `legal_document`-Tabelle ist per `rc_number` isoliert — bestehende `checkTenantAccess`-Logik wird analog verwendet

---

### Technische Entscheidungen

| Entscheidung | Begründung |
|---|---|
| Separater `document_consent`-Snapshot statt FK auf `legal_document` | Löschen eines Dokuments darf bestehende Zustimmungen nicht beeinflussen |
| Kein Hash — nur Titel + URL + Zeitstempel | Kein outbound HTTP-Fetch → kein SSRF-Risiko, keine Latenz, keine Abhängigkeit von externen URLs; Admins nutzen versionierte URLs als Nachweis |
| Reorder-Endpunkt sendet komplette neue Reihenfolge | Einfacher als einzelne Patch-Calls; atomare DB-Transaktion |
| Keine eigene `legal`-Package | Passt in `internal/application/` — kein Grund für extra Paket bei dieser Größe |

---

### Migrations-Übersicht

| Migration | Inhalt |
|---|---|
| `000018_add_legal_documents.up.sql` | Tabellen `legal_document` + `document_consent` mit Constraints und Indexes |

## Implementation Notes

### Backend (2026-04-25)
- Migration `000018_add_legal_documents.up.sql` creates `legal_document` and `document_consent` tables
- `internal/application/legal_document_repo.go`: CRUD + reorder logic; `MaxLegalDocumentsPerEEG = 10`
- `internal/application/document_consent_repo.go`: bulk insert + fetch by application_id
- `internal/application/application_service.go`: `SubmitApplication(id, consents)` saves consent snapshots in transaction
- `internal/application/admin_service.go`: `GetApplicationDetail` fetches and maps consents to `DocumentConsentView`
- `internal/application/registration_service.go`: `GetRegistrationConfig` appends central policy from env vars
- `internal/http/admin.go`: 5 new handlers (List, Create, Update, Delete, Reorder) with tenant isolation
- `internal/config/config.go`: `CENTRAL_POLICY_TITLE` + `CENTRAL_POLICY_URL` env vars
- `cmd/server/main.go`: wired all new repos and handlers; routes `/api/admin/legal-documents/*`

### Frontend (2026-04-25)
- `src/lib/api.ts`: `LegalDocumentItem`, `ConsentInput`, `DocumentConsentView` types; updated `RegistrationConfig` and `AdminApplicationDetail`; CRUD functions; `submitApplication` accepts optional consents
- `src/components/admin-legal-documents-editor.tsx`: new component with list/add/edit/delete/reorder
- `src/components/registration-form.tsx`: dynamic checkboxes from `config.legalDocuments`; central policy URL linked; required docs validated; consents sent on submit
- `src/components/admin-application-detail.tsx`: consent snapshots shown in Einwilligungen card
- `src/app/admin/settings/page.tsx`: Rechtsdokumente section added

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
