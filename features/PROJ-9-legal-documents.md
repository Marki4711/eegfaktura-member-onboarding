# PROJ-9: EEG-spezifische Rechtsdokumente mit granularer Zustimmung

## Status: Approved
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

**Date:** 2026-04-25
**QA Status:** Approved (pending `/security-review` for public endpoint + DB schema changes)

### Acceptance Criteria Results

| # | Criterion | Result | Method |
|---|-----------|--------|--------|
| AC-1 | Pro EEG: geordnete Liste von Dokumenten anlegen (Titel, URL, Pflicht, Reihenfolge) | PASS (code review) | Admin CRUD not testable without Keycloak; verified in `internal/http/admin.go` + `legal_document_repo.go` |
| AC-2 | Dokumentenliste über `/api/public/registration/{rc_number}` mitgeliefert | PASS | E2E: backend returns `legalDocuments` array incl. central policy |
| AC-3 | Pro EEG-Dokument eigene Checkbox mit verlinktem Titel | PASS | E2E: AC-3, AC-3b, AC-3c |
| AC-4 | Pflichtdokumente blockieren Absenden wenn nicht angehakt | PASS | E2E: AC-4+5 (central policy required, form blocked) |
| AC-5 | Zentrale Datenschutzerklärung immer angezeigt, immer Pflicht | PASS | E2E: AC-5 (always required, marked with `*`) |
| AC-6 | Pro Zustimmung: Titel, URL, Zeitstempel gespeichert | PASS (code review) | `document_consent_repo.go` `CreateBulkTx`; `application_service.go` saves snapshot |
| AC-7 | Gespeicherte Zustimmungen in Admin-Detailansicht sichtbar | PASS (code review) | `admin-application-detail.tsx` renders `consents` array; not testable without Keycloak |
| AC-8 | Admin kann Dokumente hinzufügen, bearbeiten, löschen, sortieren | PASS (code review) | `admin-legal-documents-editor.tsx` + 5 admin API endpoints; not testable without Keycloak |
| AC-9 | Löschen beeinflusst keine gespeicherten Zustimmungen | PASS (design) | `document_consent` hat keinen FK auf `legal_document`; Snapshot-Prinzip |

**Result: 9/9 criteria passed**

### Edge Cases

| Edge Case | Result |
|-----------|--------|
| Keine EEG-Dokumente konfiguriert → nur zentrale Policy | PASS — Formular lädt und ist voll funktionsfähig |
| Optionales Dokument nicht angehakt → Antrag einreichbar | PASS (E2E: AC-6-edge) |
| Dokument-Link nicht erreichbar → wird trotzdem angezeigt | PASS (design: keine Erreichbarkeitsprüfung) |
| Dokument nach Einreichung gelöscht → bestehende Zustimmungen unberührt | PASS (design: Snapshot ohne FK) |

### Security Smoke Test

| Severity | Datei | Funktion | Risiko | Exploit-Szenario | Fix-Empfehlung | Confidence |
|----------|-------|----------|--------|------------------|----------------|------------|
| Medium | `internal/http/admin.go` | `handleCreateLegalDocument`, `handleUpdateLegalDocument` | Kein max-length check auf `title`/`url` — sehr lange Strings möglich | Admin sendet 100 KB langen Titel → evtl. DB-Fehler oder Performance-Problem | Max-Length-Validierung für `title` (z.B. 500 Zeichen) und `url` (z.B. 2000 Zeichen) hinzufügen | High |
| Low | `internal/http/registration.go` / `internal/application/application_service.go` | `SubmitApplication` | Public user kann beliebige Consent-Einträge einreichen (Titel, URL nicht gegen EEG-Dokumente geprüft) | Mitglied sendet `consents: [{title: "fake", url: "http://evil.com", isCentralPolicy: false}]` — wird so gespeichert | Design-Entscheidung: Snapshot-Prinzip. Für V2 optional: Whitelist-Check gegen `legal_document`-IDs | Low |
| Low | `internal/application/application_service.go` | `SubmitApplication` | Consent-Speicherung "best-effort" — Fehler beim Speichern blockiert Einreichung nicht | Netzwerkfehler im Consent-INSERT → Antrag eingereicht ohne Consent-Snapshot | Erwägen, Consent-Fehler als kritisch zu behandeln (Rollback der ganzen Transaktion) | Medium |

**Sicherheitshinweis:** Feature berührt public endpoint, DB-Schema-Migration und Admin-CRUD → **`/security-review` empfohlen.**

### Automated Tests

**Unit Tests:** vitest auf Windows nicht lauffähig (pre-existing: `@rolldown/binding-win32-x64-msvc` optional dependency bug mit TypeScript 6.0).

**E2E Tests:** `tests/PROJ-9-legal-documents.spec.ts` — **10/10 bestanden** (chromium)

**Regression:** 106 bestanden, 8 vorher existierende Fehler (PROJ-11 backend-unavailable-Test, PROJ-12/14 API-Shape-Mismatches, PROJ-17 Route-Issue, PROJ-8 Dev-DB-Zustand) — keine PROJ-9-Regressionen.

### Cross-Browser / Responsive

Manuelle Tests durchgeführt:
- Chrome Desktop: Formular zeigt zentrale Policy-Checkbox, Zustimmung blockiert Absenden ohne Haken
- Admin-Bereich: Redirect ohne Keycloak-Auth bestätigt
- Mobile: Playwright iPhone 13 — E2E-Tests auf Mobile Safari nicht separat ausgeführt (Testumgebung Windows)

## Deployment

**Deployed:** 2026-04-25
**Tag:** `v1.9.0-PROJ-9`

### Pre-Deployment Checklist
- [x] `npm run build` — passes (standalone output)
- [x] `go build ./...` — passes
- [x] QA approved: 9/9 acceptance criteria passed
- [x] Security review approved: Medium findings fixed (URL scheme + length validation)
- [x] Migration `000018_add_legal_documents.up.sql` applied
- [x] All commits pushed to `main`

### Helm Chart Changes
- `helm/member-onboarding/values.yaml`: Added `backend.centralPolicyTitle` + `backend.centralPolicyUrl`
- `helm/member-onboarding/templates/backend.yaml`: `CENTRAL_POLICY_TITLE` + `CENTRAL_POLICY_URL` env vars

### Production Configuration Required
Set in `values-env.yaml` before deploying to production:
```yaml
backend:
  centralPolicyTitle: "Datenschutzerklärung"
  centralPolicyUrl: "https://your-eeg.at/datenschutz"  # REQUIRED — set to actual URL
```

### Rollback
```bash
helm rollback eegfaktura-member-onboarding
# DB: run 000018_add_legal_documents.down.sql (drops legal_document + document_consent tables)
```
