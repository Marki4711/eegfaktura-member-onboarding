# PROJ-60 — Datenweiterleitung an externe Systeme nach Import (CRM-Sync)

## Status: Planned (Idea — Erst-Konzept, noch ohne ausformulierte Requirements)
**Created:** 2026-05-23
**Last Updated:** 2026-05-23
**Quelle:** Owner-Anforderung
**Erster genannter Anbieter:** Zoho CRM (https://www.zoho.com/de/crm/developer/api.html)

## Dependencies
- Requires: PROJ-4 (Core Import) — Daten-Push erfolgt nach erfolgreichem Core-Import
- Berührt: PROJ-46 (Post-Import-Stati) — Trigger könnte auf verschiedene Status-Übergänge ausgerichtet sein (imported / activated / …)
- Berührt: PROJ-8 (konfigurierbare Felder pro EEG) — pro EEG aktivierbar
- Berührt: DSGVO-Auftragsverarbeitungs-Gefüge zwischen EEG, Onboarding-Anbieter und externem System

---

## Idee in einem Satz

Nach erfolgreichem Import eines Mitglieds in den eegFaktura-Core werden
ausgewählte Mitgliedsdaten an ein **externes System** (zunächst Zoho
CRM, später potenziell weitere) weitergeleitet, damit die EEG das
Mitglied dort parallel in ihrer eigenen CRM-Welt pflegen kann.

## Hintergrund

EEG-Vereine pflegen ihre Mitglieder teils nicht nur im
eegFaktura-Core, sondern parallel in einem eigenen CRM-System (Zoho,
HubSpot, Salesforce, etc.) — z. B. für Sales/Outreach,
Spenden-Tracking, Newsletter-Versand oder Veranstaltungseinladungen.

Aktuell muss der EEG-Admin diese parallele Pflege manuell machen:
- Mitglied im Onboarding-System genehmigen → import in Core
- Mitglied erscheint im Core
- EEG-Admin muss die Daten **manuell** ins CRM übertragen

Das ist:
- fehleranfällig (Tippfehler, vergessene Felder)
- ineffizient (Doppelpflege)
- nicht skalierbar (bei größeren EEGs prohibitive Mehrarbeit)

Eine automatische Weiterleitung würde diesen Aufwand eliminieren.

## Skizze des Konzepts (erster Wurf)

### Architektur

```
Onboarding-App
  ├─ Mitglied wird importiert (PROJ-4)
  │    Status: approved → imported
  │
  └─ External-Sync-Worker
       ├─ Liest EEG-Konfiguration
       │    (Welche externen Systeme sind aktiv?)
       │
       ├─ Pro aktivem System:
       │    ├─ Mapping anwenden (Onboarding-Felder → CRM-Felder)
       │    ├─ Authenticate (OAuth2 für Zoho etc.)
       │    ├─ POST /contacts oder /leads
       │    ├─ Externe ID in eigener Mapping-Tabelle speichern
       │    └─ Retry-Logik bei Fehlern
       │
       └─ Status in DB: ext_sync_status_log
```

### Plugin-/Adapter-Pattern

Statt nur Zoho fest zu verdrahten:
- Generisches **ExternalSystemAdapter**-Interface im Backend
- Erste Implementierung: **ZohoCRMAdapter**
- Später erweiterbar mit z. B. HubSpotAdapter, SalesforceAdapter, PipedriveAdapter
- Pro EEG konfigurierbar welches System(e) genutzt wird

### Per-EEG-Konfiguration

Neue Tabelle oder erweiterung der `registration_entrypoint`-Tabelle:

```
external_system_config
  rc_number        FK auf registration_entrypoint
  provider         ENUM (zoho_crm, hubspot, salesforce, …)
  enabled          BOOLEAN
  credentials      ENCRYPTED JSONB (OAuth tokens etc.)
  field_mapping    JSONB (Onboarding-Feld → CRM-Feld)
  trigger_status   ARRAY (welche Status-Übergänge feuern den Sync)
  created_at, updated_at
```

### Mapping Onboarding → Zoho

Standardmapping (Vorschlag, anpassbar pro EEG):

| Onboarding-Feld | Zoho CRM-Feld (Contact-Modul) |
|---|---|
| `firstname` + `lastname` (für `private`/`farmer`) ODER `company_name` (für Org-Typen) | `Last_Name` (Pflicht in Zoho) + `First_Name` |
| `email` | `Email` (Pflicht in Zoho) |
| `phone` | `Phone` |
| `resident_street` + `_number` | `Mailing_Street` |
| `resident_zip` | `Mailing_Zip` |
| `resident_city` | `Mailing_City` |
| `member_type` | Custom Field `Member_Type` |
| `iban` | Custom Field `IBAN` (verschlüsselt) — **DSGVO-Frage:** zoho-seitige Verschlüsselung? |
| `member_number` (nach Import) | Custom Field `EEG_Member_Number` |
| `uid_number` | Custom Field `UID_Number` |
| Onboarding-Antrags-ID | Custom Field `EEG_Onboarding_ID` (Cross-Referenz) |

Zoho CRM Modul: voraussichtlich **Contacts** (Einzelpersonen) — oder ggf. **Accounts** für Org-Typen mit Personen als Subordinate. Klärung im /requirements-Lauf.

### Trigger-Punkt

Standardmäßig: **bei Status-Übergang zu `imported`** (= erfolgreich im Core gelandet).

Optional konfigurierbar pro EEG:
- `imported` (Default — Mitglied ist im Core, parallele CRM-Anlage)
- `activated` (Mitglied ist endgültig aktiv — alternative wenn EEG nur „echte" Mitglieder im CRM will)
- Auch bei späteren Updates (Adressänderung etc.) sync triggern? → bidirektionale Komplexität, eher V2

## Open Questions (zu klären in `/requirements`-Lauf)

### Funktionalität
- **MVP vs. Full**: erstes Release nur Zoho einbahnig, oder gleich Plugin-Architektur vorbereiten?
- **Trigger**: nur bei `imported`, oder auch `activated`, oder beide?
- **Updates**: Re-Sync bei Adressänderung / E-Mail-Wechsel / Member-Number-Vergabe — nötig oder Out-of-Scope?
- **Bidirektional?** CRM-Änderungen zurück ins Onboarding-System — wahrscheinlich Out-of-Scope V1
- **Welches Zoho-Modul** (Contacts / Leads / Accounts)? Default Contacts vermutlich, aber EEG-konfigurierbar wünschenswert?
- **Field-Mapping**: Standard-Mapping vs. customizable pro EEG vs. UI für Admin zum selbst konfigurieren?
- **Bulk-Sync für Bestandsdaten**: bei Erstkonfiguration durch EEG — sollen alle bestehenden importierten Mitglieder retroaktiv in Zoho gepusht werden?

### Auth + Credentials
- **OAuth2-Setup**: Zoho CRM nutzt OAuth2 mit Refresh-Tokens. Per EEG eigene Tokens, gespeichert verschlüsselt.
- **Token-Renewal**: Wer kümmert sich um Refresh-Token-Rotation? Bei Ablauf: Admin-Notification?
- **Multi-Tenant in Zoho**: Verwendet die EEG ihr eigenes Zoho-Konto, oder gibt es Anbieter mit zentraler Zoho-Org für mehrere EEGs?

### Datenschutz / DSGVO
- **Verantwortlichkeit**: Wenn wir Daten an Zoho weiterleiten — sind wir dann (zusätzliche) Auftragsverarbeiter für die EEG bzgl. der Zoho-Weiterleitung, oder ist das technisch nur eine „Datendurchleitung"?
- **AVV-Verantwortlichkeit**: Die EEG muss einen separaten AVV mit Zoho haben. Müssen wir sicherstellen, dass das vorhanden ist, oder reicht eine Hinweispflicht in unserer Doku?
- **Datenminimierung**: nur Felder syncen, die für die CRM-Pflege wirklich nötig sind. IBAN z. B. eher nicht (Zoho ist kein Banking-System).
- **Drittlandtransfer**: Zoho hat EU-Datacenter (zoho.eu) — pro EEG konfigurierbar das EU-Datacenter erzwingen, sonst US-Transfer-Problematik.
- **Recht zu Widerruf**: wenn ein Mitglied seine Daten löschen lässt — muss der Sync rückwärts laufen (auch in Zoho löschen)?

### Robustheit
- **Retry-Strategie**: bei Zoho-API-Fehler (rate limit, timeout, 5xx) — exponential backoff? Max-Retries?
- **Idempotenz**: wenn dasselbe Mitglied zweimal versucht wird zu syncen — Duplikate vermeiden über `EEG_Onboarding_ID` als Suchschlüssel.
- **Rate-Limits**: Zoho CRM hat Rate-Limits (z. B. 100/min für Free-Tier, höher für Paid). Bei großen Bulk-Sync-Operationen drosseln.
- **Fehler-Sichtbarkeit**: Admin sieht im UI, welche Mitglieder erfolgreich gesynct wurden und welche fehlgeschlagen.

### Konfiguration im Admin-UI
- Eigene Section unter EEG-Settings („Externe Systeme") nur für `eeg_admin` der jeweiligen EEG (nicht für andere EEGs sichtbar)
- OAuth-Flow im UI: „Mit Zoho verbinden" → Popup mit Zoho-Login → Token kommt zurück → in DB verschlüsselt gespeichert
- Field-Mapping als JSON-Editor oder UI-Form?

## Acceptance Criteria (vorläufig — wird im /requirements verfeinert)

- [ ] Pro EEG aktivierbar/deaktivierbar im Admin-UI
- [ ] Zoho-OAuth-Flow funktioniert (initiale Verbindung + Token-Refresh)
- [ ] Bei `imported`-Status-Übergang wird Mitglied automatisch in Zoho angelegt
- [ ] Standard-Field-Mapping deckt die Pflichtfelder ab (Last_Name, Email + EEG_Onboarding_ID)
- [ ] Bei Fehler: Retry mit Backoff, nach 3 Fehlversuchen Admin-Notification
- [ ] Zoho-Contact-ID wird in Onboarding-DB gespeichert (für Re-Sync / Update / Audit)
- [ ] Admin-UI zeigt pro Mitglied den Sync-Status (synced / pending / failed)
- [ ] Bei Mitglieds-Update (Adresse etc.) wahlweise Re-Sync (V1 oder V2-Frage)
- [ ] DSGVO: Hinweistext beim EEG-Admin im Setup-Flow, dass eigener AVV mit Zoho nötig ist
- [ ] EU-Datacenter (zoho.eu) wird verwendet bei Pflichtkonfiguration

## Edge Cases (vorläufig)

- Was passiert, wenn Zoho-API down ist während eines Imports? → Sync erfolgt **asynchron** (eigener Worker / Queue), nicht synchron im Import-Pfad — sonst blockiert Zoho-Downtime den Core-Import.
- Was passiert, wenn dasselbe Mitglied zweimal importiert wird (Re-Import via PROJ-30 + Re-Sync)? → Update statt Create (Lookup via `EEG_Onboarding_ID`).
- Was, wenn das Mitglied im CRM bereits manuell angelegt wurde? → Konflikt-Strategie (skip / overwrite / merge / Admin-Entscheidung)?
- Was, wenn die EEG ihren Zoho-Vertrag kündigt? → Sync deaktivierbar im UI, Bestandsdaten in Zoho bleiben.
- Was passiert mit gelöschten Mitgliedern (`rejected`-Anträge)? → Standardmäßig kein Sync (rejected wird nicht importiert).
- Was passiert beim Status-Übergang `imported` → `approved` via PROJ-30? → Soll auch im CRM rückgängig gemacht werden? Vermutlich nein (CRM hat seine eigene Lifecycle-Logik).

## Technical Requirements (vorläufig)

- **Asynchroner Worker** statt synchron im Import-Pfad — vermeidet Blocking + bessere Retry-Möglichkeit
- **Encrypted-at-Rest** für OAuth-Tokens in der DB (analog Keycloak-Secrets-Pattern)
- **Audit-Log** in `ext_sync_status_log` für jede Sync-Operation (Mitglieds-ID, Provider, Status, Timestamp, Fehler-Message)
- **Rate-Limiting-Awareness** im Adapter-Code (Zoho-Rate-Limit-Headers respektieren)
- **Mock-Adapter für Tests** (Test-Mode ohne echte Zoho-Calls)

## Notes

- **Zoho ist nur der erste Anbieter** — Architektur soll Erweiterung erleichtern, ohne dass jedes neue System eine Core-Code-Änderung benötigt. Plugin-Pattern ist deshalb wichtig.
- **Vendor-Lock-in vermeiden**: pro EEG-Konfiguration soll wechselbar sein (Migration zwischen CRMs ohne Code-Änderung an unserer Seite).
- **Verhältnis zu PROJ-55 (Self-Service-Portal)**: Wenn das Portal kommt, wird der CRM-Sync zur Grundlage für „Mitglied pflegt seine eigenen Daten im Portal, Änderungen werden auch ins CRM gesynct" — bidirektionaler Aspekt rückt dann näher.
- **Vermarktungs-Argument**: für EEGs, die sowieso schon Zoho/HubSpot nutzen, ist das ein starkes Argument für Member-Onboarding („Sie müssen Daten nicht mehr doppelt pflegen").

## Nächster Schritt

Bei tatsächlicher Aufnahme der Spec: `/requirements`-Lauf mit dieser
Datei als Ausgangspunkt, um folgende Punkte zu schärfen:
1. **MVP-Scope**: nur Zoho oder gleich Plugin-Pattern?
2. **Trigger + Update-Strategie**: einmalig bei Import vs. laufende Re-Syncs?
3. **Field-Mapping**: hardcoded Standard vs. konfigurierbar pro EEG?
4. **Konflikt-Strategie** bei Duplikaten im Ziel-System?
5. **DSGVO-Verantwortung**: wie genau wird die AVV-Beziehung dokumentiert?

Dann `/architecture` für die technische Plugin-/Adapter-Struktur, dann
`/backend` + `/frontend` für die Implementierung.
