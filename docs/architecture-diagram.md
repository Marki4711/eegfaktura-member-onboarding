# Architecture Diagram: eegfaktura-member-onboarding

## System Context: Shared Infrastructure

```mermaid
graph TB
    subgraph eegFaktura["eegFaktura (existing system)"]
        EF_WEB["eegfaktura-web\n(Admin-Frontend)"]
        EF_BACKEND["eegfaktura\n(Backend / REST API)"]
        EF_DB[("PostgreSQL\nSchema: public\n(productive data)")]
        EF_WEB -->|REST| EF_BACKEND
        EF_BACKEND -->|read/write| EF_DB
    end

    subgraph MemberOnboarding["eegfaktura-member-onboarding (this repo)"]
        MOB_PUB["Public Web\n(Beitrittsformular)"]
        MOB_ADM["Admin Web\n(Antragsverwaltung)"]
        MOB_BE["Member Onboarding\nBackend (Go)"]
        MOB_DB[("PostgreSQL\nSchema: member_onboarding")]
        MOB_PUB -->|REST| MOB_BE
        MOB_ADM -->|REST| MOB_BE
        MOB_BE -->|read/write| MOB_DB
    end

    subgraph SharedInfra["Shared Infrastructure"]
        KC["Keycloak\n(Identity Provider)"]
        POSTAL["Postal\n(SMTP / E-Mail)"]
    end

    CF["Cloudflare\nTurnstile\n(CAPTCHA)"]

    MOB_ADM -->|"Login / Token\nverification"| KC
    EF_WEB -->|"Login / Token\nverification"| KC
    MOB_BE -->|"JWT validation\n(Admin API)"| KC

    MOB_BE -->|"Confirmation\nemail"| POSTAL
    EF_BACKEND -->|"System emails"| POSTAL

    MOB_BE -->|"CAPTCHA\nverification\n(public form)"| CF

    MOB_BE -->|"Import approved\napplication\n(internal call)"| EF_BACKEND
```

---

## Current Integration Points and Risks

### 1. Keycloak (Auth)

| Aspekt | Detail |
|---|---|
| **Nutzung** | Admin-Login für `eegfaktura-web` und `eegfaktura-member-onboarding` |
| **Teilen** | Beide Systeme teilen dieselbe Keycloak-Instanz und denselben Realm |
| **Risiko** | Fällt Keycloak aus, ist der Admin-Bereich **beider** Systeme nicht erreichbar. Das öffentliche Beitrittsformular ist davon **nicht** betroffen (kein Login erforderlich). |
| **Risiko** | Keycloak-Konfigurationsänderungen (Realm-Einstellungen, Client-Config, Token-Lebensdauer) können beide Systeme gleichzeitig beeinflussen. |
| **Risiko** | Mandantentrennung erfolgt über Keycloak-Gruppen/Rollen; Fehlkonfigurationen könnten Admins aus verschiedenen EEGs auf fremde Daten zugreifen lassen. |
| **Maßnahme** | Monitoring: Keycloak-Health-Endpunkt überwachen. Konfigurationsänderungen vor Produktionseinsatz in Staging testen. |

### 2. Postal (SMTP / E-Mail)

| Aspekt | Detail |
|---|---|
| **Nutzung** | Bestätigungsemail an Mitglied + Benachrichtigungsemail an EEG bei neuem Antrag |
| **Teilen** | Beide Systeme verwenden dieselbe Postal-Instanz auf Port 25 (Port 587 gesperrt) |
| **Risiko** | Postal-Ausfall blockiert Bestätigungsemails. Der Antrag wird **dennoch** gespeichert – die E-Mail-Zustellung wird nicht für die Persistenz benötigt. |
| **Risiko** | Postal-Authentifizierungsprobleme (API-Key, Domain-Verifizierung) können die E-Mail-Zustellung ohne Hinweis unterbrechen. |
| **Risiko** | Shared Infrastructure: E-Mail-Konfigurationsfehler in einem System können die Zustellung auch für das andere System beeinflussen. |
| **Maßnahme** | E-Mail-Fehler werden geloggt (kein Silent Fail). SMTP-Verbindungsfehler sind im Backend-Log sichtbar. |

---

## Integration mit eegFaktura Core (live)

Beide Integrationspfade — direkter API-Aufruf und Excel-Export — sind
implementiert und in Produktion. Der direkte API-Pfad ist der primäre
Onboarding-Flow; der Excel-Export bleibt als Fallback (z.B. bei
Core-Wartung, oder für Anwender, die lieber prüfen-vor-Import).

### A) Direkter API-Aufruf (Standard-Pfad, live)

```mermaid
graph LR
    MOB_BE["Member Onboarding\nBackend"] -->|"POST /api/participant\n(Bearer Token, tenant-header)"| EF_BACKEND["eegfaktura\nBackend"]
    MOB_BE -->|"GET /api/participant\n(Activation-Check, PROJ-46 Stage D)"| EF_BACKEND
    MOB_BE -->|"GraphQL eeg-master-data\n(PROJ-32 Sync)"| EF_BACKEND
    EF_BACKEND -->|write/read| EF_DB[("eegFaktura DB\nSchema: public")]
```

| Aspekt | Detail |
|---|---|
| **Status** | **Live seit PROJ-4** (Onboarding → Core), erweitert in PROJ-27 (Tariff-Selection), PROJ-30 (Reset-Import), PROJ-32 (EEG-Stammdaten-Sync, GraphQL), PROJ-33 (Logo-Sync), PROJ-34 (Stuck-Recovery), PROJ-46 (Post-Import-Stati + GET /participant Activation-Check). |
| **Auth** | Admin-Bearer-Token wird per `Authorization`-Header weitergereicht (User-Context, kein Service-Account). |
| **Boundary** | Backend ist die einzige Komponente, die den Core erreicht. Frontends und externe API rufen nie direkt den Core auf. |
| **Resilience** | Retry-Logik auf HTTP-Level; PROJ-34-Stuck-Recovery für orphan-Participants; Reset-Import für rollback nach Core-seitiger Korrektur. |

### B) Excel-Export (Fallback, live)

```mermaid
graph LR
    MOB_BE["Member Onboarding\nBackend"] -->|"Export: XLSX-Datei\nper Download"| ADMIN["EEG-Admin"]
    ADMIN -->|"Import: Upload\nin eegFaktura"| EF_WEB["eegfaktura-web\n(Excel-Import)"]
```

| Aspekt | Detail |
|---|---|
| **Status** | **Live seit PROJ-17.** Endpoint `GET /api/admin/applications/{id}/export/excel`. |
| **Wann verwenden** | Wenn der Core temporär nicht erreichbar ist, oder wenn der Admin den Import zunächst prüfen will (Excel ist editierbar). |
| **Nachteil** | Manueller Schritt im EEG-Faktura-Frontend; PROJ-45-Spalten (Erzeugungsform, Batterie, Wechselrichter) am Spalten-Ende werden vom alten Importer ggf. ignoriert. |

---

## Status-Übergänge (vollständig)

Visuelle Darstellung aller 12 Status-Werte mit allen erlaubten Übergängen.
Renderbar direkt in GitHub und IDEs (Mermaid `stateDiagram-v2`). Pendant
zum kanonischen Modell in `CLAUDE.md` und `docs/domain-model.md`.

```mermaid
stateDiagram-v2
    direction LR

    [*] --> draft

    draft --> submitted: Mitglied submit

    state submitted_choice <<choice>>
    submitted --> submitted_choice
    submitted_choice --> email_confirmed: Mitglied klickt Link\n(PROJ-31)
    submitted_choice --> under_review: Admin
    submitted_choice --> rejected: Admin (Anti-Spam)

    email_confirmed --> under_review: Admin
    email_confirmed --> needs_info: Admin
    email_confirmed --> approved: Admin
    email_confirmed --> rejected: Admin

    under_review --> needs_info: Admin (hard-fail mail)
    under_review --> approved: Admin
    under_review --> rejected: Admin (hard-fail mail)

    needs_info --> submitted: Mitglied editiert + re-submit

    approved --> imported: Import erfolgreich
    approved --> import_failed: Import-Fehler

    import_failed --> approved: Admin retry

    state imported_branch <<choice>>
    imported --> imported_branch: auto (Service)
    imported_branch --> awaiting_bank_confirmation: einzugsart=b2b
    imported_branch --> ready_for_activation: sonst

    awaiting_bank_confirmation --> ready_for_activation: Admin\n(Bank-Bestätigung erhalten)
    awaiting_bank_confirmation --> under_review: Admin (rückwärts)

    ready_for_activation --> activated: Admin manuell\nODER Batch-Check
    ready_for_activation --> under_review: Admin (rückwärts)

    activated --> [*]

    %% Reset-Pfade via POST /reset-import
    imported --> approved: Reset-Import
    awaiting_bank_confirmation --> approved: Reset-Import
    ready_for_activation --> approved: Reset-Import

    note right of activated
      Strikter Endzustand
      kein Reset
      Deaktivierung im Core
    end note

    note right of imported
      Transient (ms)
      Auto-Branch sofort danach
    end note
```

### Legende

| Element | Bedeutung |
|---|---|
| **Auto-Übergang** | Vom Import-Service automatisch ausgeführt — `imported → awaiting_bank_confirmation` und `imported → ready_for_activation` |
| **Admin-Aktion** | Manuell im Admin-Web ausgelöst — alle Übergänge mit „Admin"-Label |
| **Mitglied-Aktion** | Vom Mitglied ausgelöst — `draft → submitted`, `submitted → email_confirmed` (Klick), `needs_info → submitted` (Re-Submit) |
| **hard-fail mail** | Synchron-blockierender Mail-Versand (PROJ-41 + PROJ-43) — schlägt SMTP fehl, wird der Statuswechsel rückgängig gemacht |
| **Reset-Import** | Dedizierte `POST /reset-import`-Route (PROJ-30 + PROJ-46 expansion); NICHT erreichbar aus `activated` |
| **Batch-Check** | Optional via `POST /api/admin/applications/check-activation` (PROJ-46 Stage D); fragt Core ob Mitglied ACTIVE und setzt automatisch |

---

## Gesamtübersicht: Systemgrenzen und Verantwortlichkeiten

```mermaid
graph TB
    subgraph Public["Öffentlich (kein Login)"]
        MEMBER["Neues Mitglied\n(Browser)"]
    end

    subgraph AdminArea["Admin-Bereich (Keycloak-Login)"]
        EEG_ADMIN["EEG-Administrator\n(Browser)"]
    end

    subgraph MO["eegfaktura-member-onboarding"]
        PUB_FORM["Beitrittsformular\n(Next.js)"]
        ADM_UI["Admin-Oberfläche\n(Next.js)"]
        BE["Backend\n(Go REST API)"]
        DB[("member_onboarding\nSchema")]
    end

    subgraph EF["eegFaktura"]
        EF_WEB["eegfaktura-web"]
        EF_BE["eegfaktura Backend"]
        EF_DB[("public\nSchema")]
    end

    KC["Keycloak"]
    POSTAL["Postal SMTP"]
    CF["Cloudflare Turnstile"]

    MEMBER --> PUB_FORM
    PUB_FORM -->|"CAPTCHA-Token"| CF
    PUB_FORM -->|"POST /api/public/applications"| BE
    BE -->|"Verify token"| CF

    EEG_ADMIN --> ADM_UI
    EEG_ADMIN --> EF_WEB
    ADM_UI -->|"Bearer Token"| BE
    EF_WEB --> EF_BE

    ADM_UI & EF_WEB -->|"Auth"| KC
    BE -->|"JWT validation"| KC

    BE --> DB
    BE -->|"E-Mail"| POSTAL
    EF_BE -->|"E-Mail"| POSTAL

    BE -.->|"geplant:\nImport-API-Call"| EF_BE
    EF_BE --> EF_DB
```

> **Legende:**
> - Durchgezogene Pfeile: bereits implementiert
> - Gestrichelte Pfeile: geplant, noch nicht implementiert
