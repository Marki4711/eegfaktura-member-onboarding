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

## Geplante zukünftige Integration: Import in eegFaktura Core

### Option A: Direkter API-Aufruf (intern)

```mermaid
graph LR
    MOB_BE["Member Onboarding\nBackend"] -->|"POST /api/participants\n(Bearer Token)"| EF_BACKEND["eegfaktura\nBackend"]
    EF_BACKEND -->|write| EF_DB[("eegFaktura DB\nSchema: public")]
```

| Aspekt | Detail |
|---|---|
| **Vorteil** | Vollständig automatisierbar, kein manueller Schritt |
| **Risiko** | **Enge Kopplung**: Änderungen an der eegFaktura-API erfordern Anpassungen im Onboarding-Backend. |
| **Risiko** | eegFaktura-API muss eine stabile, versionierte Schnittstelle für den internen Gebrauch bereitstellen. |
| **Risiko** | Netzwerkfehler oder API-Fehler können den Import-Schritt blockieren; Retry-Logik ist erforderlich. |
| **Risiko** | Auth-Token für den Service-to-Service-Call muss sicher verwaltet werden (Keycloak Service Account oder API-Key). |
| **Aktueller Status** | In der Architektur vorgesehen (`internal/coreclient`, `internal/importing`), aber **noch nicht implementiert**. |

### Option B: Excel-Export / Datei-Import (entkoppelt)

```mermaid
graph LR
    MOB_BE["Member Onboarding\nBackend"] -->|"Export: XLSX-Datei\nper Download"| ADMIN["EEG-Admin"]
    ADMIN -->|"Import: Upload\nin eegFaktura"| EF_WEB["eegfaktura-web\n(Excel-Import)"]
```

| Aspekt | Detail |
|---|---|
| **Vorteil** | **Vollständig entkoppelt**: Kein direkter API-Aufruf zwischen den Systemen. Änderungen an eegFaktura beeinflussen das Onboarding nicht. |
| **Vorteil** | Kein Netzwerkfehlerrisiko zwischen den Backends. |
| **Vorteil** | Admin behält volle Kontrolle und kann vor dem Import prüfen. |
| **Nachteil** | Manueller Schritt: Admin muss Export herunterladen und in eegFaktura importieren. |
| **Nachteil** | Kein Echtzeit-Feedback ob der Import erfolgreich war. |
| **Eignung** | Empfohlen als **kurzfristige Lösung** und für EEGs mit geringem Antragsvolumen. |

---

## Empfehlung: Phasenweises Vorgehen

```
Phase 1 (jetzt):       Excel-Export als Fallback / manuelle Option implementieren
                       → Niedrigstes Risiko, sofort nutzbar

Phase 2 (mittelfristig): Direkten API-Aufruf implementieren, wenn eegFaktura eine
                          stabile interne Import-API bereitstellt
                          → Parallel zum Excel-Export betreiben
                          → Automatisiert für Standard-Fälle

Phase 3 (langfristig):   Excel-Import als Fallback beibehalten (Fehlerbehandlung,
                          manuelle Korrekturen)
```

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
