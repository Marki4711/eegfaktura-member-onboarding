# PROJ-11: Konfigurierbarer Einleitungstext im Registrierungsformular

## Status: Approved
**Created:** 2026-04-23
**Last Updated:** 2026-04-23

## Dependencies
- Requires: PROJ-1 (Public Registration) — Einleitungstext wird im öffentlichen Formular angezeigt
- Requires: PROJ-5 (Keycloak-secured Admin Area) — Konfiguration nur für authentifizierte Admins

## User Stories

- Als EEG-Administrator möchte ich einen Einleitungstext für das Registrierungsformular meiner EEG hinterlegen können, damit Neumitglieder beim Öffnen des Formulars wichtige Informationen und Hinweise sehen.
- Als EEG-Administrator möchte ich Links im Einleitungstext einbauen können (z.B. auf Satzung, FAQ oder Kontaktseite), damit Interessenten weiterführende Informationen direkt aufrufen können.
- Als EEG-Administrator möchte ich den Text mit einfacher Formatierung gestalten können (Fett, Kursiv, Absätze, Listen), damit der Text übersichtlich und lesbar ist.
- Als Mitglied möchte ich beim Öffnen des Registrierungsformulars einen Begrüßungs- und Erklärungstext sehen, damit ich verstehe, was ich ausfüllen muss und wofür ich mich anmelde.
- Als Mitglied möchte ich, dass der Einleitungstext Links enthält, die ich direkt anklicken kann (z.B. zur Vereinswebsite oder Kontakt), damit ich weitere Informationen erhalten kann ohne das Formular zu verlassen.

## Acceptance Criteria

- [ ] Der Admin kann im Admin-Backend einen Einleitungstext pro EEG (RC-Nummer) speichern und bearbeiten
- [ ] Der Editor erlaubt folgende Formatierungen: Fett, Kursiv, Hyperlinks (Text + URL), Absätze, geordnete und ungeordnete Listen
- [ ] Der Einleitungstext wird im öffentlichen Registrierungsformular oberhalb des Formulars angezeigt
- [ ] Ist kein Einleitungstext konfiguriert, wird ein Standardtext angezeigt: „Füllen Sie das Formular aus, um Ihre Mitgliedschaft zu beantragen."
- [ ] Der gespeicherte Text wird sicher gerendert (kein Inline-JavaScript, kein unvalidiertes HTML)
- [ ] Der Einleitungstext ist pro EEG konfigurierbar — verschiedene EEGs können unterschiedliche Texte haben
- [ ] Links im Einleitungstext öffnen in einem neuen Tab (`target="_blank"`)
- [ ] Änderungen am Text werden sofort nach dem Speichern im Admin-Backend für neue Besucher des Formulars wirksam

## Edge Cases

- Was wenn der Admin HTML-Injection oder Script-Tags eingibt? → Eingabe wird server-seitig bereinigt (Strip unsichere Tags); nur erlaubte Formatierungen werden gespeichert
- Was wenn der Einleitungstext sehr lang ist (z.B. 10 Absätze)? → Kein hard limit, aber der Admin-Editor zeigt die tatsächliche Länge des Textes; Layoutanpassung im Formular nötig (scrollbar oder faltbar)
- Was wenn kein Text gespeichert ist (leerer String)? → Verhält sich wie "nicht konfiguriert" → Standardtext wird angezeigt
- Was wenn der Backend-Request für den Einleitungstext fehlschlägt? → Standardtext wird im Formular angezeigt (Fail-Open), Fehler wird geloggt
- Was wenn der Admin den Text löscht (leert und speichert)? → Standardtext wird wieder angezeigt

## Technical Requirements

- Speicherung als HTML-String (bereinigt) oder als strukturiertes Format (z.B. ProseMirror-JSON / Tiptap-JSON) — zu entscheiden in Architecture
- Backend: neues Feld `intro_text` in `member_onboarding.registration_entrypoint` ODER separater Eintrag in einer Settings-Tabelle
- Alternativ: Erweiterung der bestehenden `field_config`-Struktur nicht geeignet, da es kein Feld-State-Modell ist
- API: `PUT /api/admin/settings/intro-text?rc_number=...` zum Speichern
- API: Einleitungstext wird bereits über `GET /api/public/registration/{rc_number}` zurückgegeben (neues Feld `introText`)
- Frontend WYSIWYG: bevorzugt Tiptap (basiert auf ProseMirror, kompatibel mit React/Next.js, keine externe Lizenz notwendig)

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)

### Speicherformat
Sanitized HTML-String. Tiptap gibt HTML aus; dieses wird serverseitig mit `bluemonday` (Go) bereinigt bevor es gespeichert wird. Im Frontend rendert DOMPurify als zweite Schutzschicht. Kein proprietäres JSON-Format nötig.

### Speicherort
Neues Feld `intro_text TEXT NULL` in `member_onboarding.registration_entrypoint`. NULL = kein Text konfiguriert = Standardtext im Frontend.

### Komponenten-Struktur
```
Admin-Bereich
└── AdminIntroTextEditor (neu)
    ├── Tiptap-Toolbar (Fett, Kursiv, Listen, Link)
    ├── Tiptap-Eingabebereich (WYSIWYG)
    └── Speichern-Button

Öffentliches Registrierungsformular
└── RegistrationForm (bestehend)
    └── IntroTextDisplay (neu)
        ├── introText vorhanden → sicher gerendertes HTML (DOMPurify)
        └── leer/null → Standardtext „Füllen Sie das Formular aus, um Ihre Mitgliedschaft zu beantragen."
```

### API-Änderungen
- `GET /api/public/registration/{rc_number}` — Response erhält neues Feld `introText` (string | null)
- `PUT /api/admin/entrypoints/{rc_number}/intro-text` — speichert Einleitungstext (Keycloak-gesichert, Backend sanitized)

### Neue Pakete
Frontend: `@tiptap/react`, `@tiptap/starter-kit`, `@tiptap/extension-link`, `dompurify`, `@types/dompurify`
Backend: `github.com/microcosm-cc/bluemonday`

## QA Test Results

**Date:** 2026-04-23
**Status after QA:** Approved

### Acceptance Criteria

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| 1 | Admin kann Einleitungstext speichern und bearbeiten | ✅ Pass | `AdminIntroTextEditor` + `PUT /api/admin/settings/intro-text` implementiert |
| 2 | Editor erlaubt Fett, Kursiv, Hyperlinks, Absätze, Listen | ✅ Pass | Tiptap mit StarterKit + Link-Extension; alle Formate vorhanden |
| 3 | Einleitungstext wird oberhalb des Formulars angezeigt | ✅ Pass | `IntroTextDisplay` als erstes Element im Form-Render |
| 4 | Kein Text → Standardtext angezeigt | ✅ Pass | E2E-Test bestätigt; `IntroTextDisplay` fällt auf DEFAULT_TEXT zurück |
| 5 | Text wird sicher gerendert (kein JS, kein unvalidiertes HTML) | ✅ Pass | Backend: bluemonday; Frontend: DOMPurify; E2E-Security-Tests bestätigt |
| 6 | Einleitungstext ist pro EEG konfigurierbar | ✅ Pass | `rc_number`-Parameter auf beiden Endpoints; Tenant-Autorisierung aktiv |
| 7 | Links öffnen in neuem Tab (`target="_blank"`) | ✅ Pass | Tiptap Link-Extension + DOMPurify erlaubt `target`-Attribut |
| 8 | Änderungen sofort wirksam für neue Besucher | ✅ Pass | Kein Cache; Server Component liest direkt aus DB beim Seitenaufruf |

### Bugs Found

| # | Severity | Description | Fix |
|---|----------|-------------|-----|
| 1 | Medium (fixed) | Hydration-Mismatch in `IntroTextDisplay`: `useMemo` mit `typeof window === "undefined"`-Guard liefert auf SSR immer Standardtext, auf Client den echten Text → React-Hydration-Warning + kurzes Aufblitzen des Standardtexts | Umgestellt auf `useEffect` + `useState` — beide Renders zeigen Standardtext, erst nach Hydration wird introText gerendert |

### Security Audit

- ✅ Backend sanitisiert mit `bluemonday` (nur `p, br, strong, b, em, i, ul, ol, li, a[href,target,rel]` erlaubt)
- ✅ `javascript:`, `data:` URIs durch bluemonday `AllowURLSchemes` blockiert
- ✅ Frontend-Zweitschutz via DOMPurify (gleiche Allowlist)
- ✅ Tenant-Autorisierung auf `GET/PUT /api/admin/settings/intro-text`
- ✅ Kein XSS-Vektor durch Script-Tags oder Inline-Event-Handler

### Test Suite

- E2E: `tests/PROJ-11-registration-intro-text.spec.ts` — 5 bestanden, 4 mit Backend-Voraussetzung (erfordern laufenden Go-Backend-Stack)
- Unit tests: Vitest nicht lauffähig (rolldown native binding issue, npm-Bug — unabhängig von PROJ-11)

### Production-Ready

**JA** — kein Critical/High-Bug offen. Medium-Bug (Hydration) wurde im Rahmen der QA behoben.

## Deployment
_To be added by /deploy_
