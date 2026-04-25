# PROJ-16: Cloudflare Turnstile — Spam-Schutz für das Beitrittsformular

## Status: Approved
**Created:** 2026-04-24
**Last Updated:** 2026-04-24

## Dependencies
- Requires: PROJ-1 (Public Registration) — das zu schützende Formular

## User Stories

- Als **EEG-Administrator** möchte ich, dass das öffentliche Beitrittsformular gegen automatisierte Spam-Einreichungen geschützt ist, damit keine Bot-generierten Fake-Anträge in meiner Antragsliste erscheinen.
- Als **neues Mitglied** möchte ich das Formular ohne lästige Bildrätsels ausfüllen können, damit der Beitrittsprozess so einfach wie möglich bleibt.
- Als **neues Mitglied** möchte ich beim Absenden des Formulars klar sehen, wenn das CAPTCHA noch nicht gelöst wurde, damit ich weiß, was zu tun ist.

## Acceptance Criteria

### Frontend

- [x] Das Cloudflare Turnstile-Widget wird im Beitrittsformular unterhalb der Datenschutz-Checkbox und oberhalb des Absende-Buttons angezeigt, wenn `NEXT_PUBLIC_TURNSTILE_SITE_KEY` gesetzt ist.
- [x] Das Widget ist vom Typ `managed` (Cloudflare entscheidet, ob eine interaktive Challenge nötig ist — meist unsichtbar).
- [x] Der Absende-Button ist deaktiviert (`disabled`), solange kein gültiges Turnstile-Token vorliegt.
- [x] Nach erfolgreichem Widget-Lösen wird das Token im Formular-State gespeichert und der Button aktiviert.
- [x] Das Token wird beim Absenden als Feld `turnstileToken` im JSON-Body an das Backend gesendet.
- [x] Ist `NEXT_PUBLIC_TURNSTILE_SITE_KEY` **nicht** gesetzt, wird kein Widget angezeigt und das Formular funktioniert wie bisher (Dev-Modus ohne CAPTCHA).
- [x] Nach einem Fehler beim Absenden (Backend lehnt Token ab) wird das Widget zurückgesetzt und ein Hinweis angezeigt.

### Backend

- [x] Das Backend verifiziert das `turnstileToken`-Feld aus dem Request-Body gegen die Cloudflare Siteverify-API (`https://challenges.cloudflare.com/turnstile/v0/siteverify`).
- [x] Ist `TURNSTILE_SECRET_KEY` **nicht** konfiguriert, wird die Verifikation übersprungen (Dev-Modus — kein Fehler).
- [x] Ein fehlgeschlagener Turnstile-Check gibt HTTP 422 mit Fehlercode `turnstile_failed` zurück.
- [x] Ein fehlendes `turnstileToken` bei aktiviertem Secret Key gibt HTTP 422 mit `turnstile_missing` zurück.
- [x] Die externe Registrierungs-API (PROJ-13, `/api/external/registration`) ist vom Turnstile-Check ausgenommen — diese Schnittstelle ist bereits über API-Key gesichert.
- [x] Das Turnstile-Token wird **nicht** in der Datenbank gespeichert.

### Konfiguration

- [x] `NEXT_PUBLIC_TURNSTILE_SITE_KEY` ist in `.env.local.example` dokumentiert.
- [x] `TURNSTILE_SITE_KEY` (für das Frontend-SSR, wenn nötig) und `TURNSTILE_SECRET_KEY` (Backend) sind in den Helm-Values-Beispieldateien dokumentiert.
- [x] Der Secret Key wird als Kubernetes Secret verwaltet (analog zu anderen Secrets im Chart).

## Edge Cases

- **Turnstile-Dienst nicht erreichbar (Cloudflare down):** Verifikationsaufruf schlägt mit Netzwerkfehler fehl → Backend behandelt dies als `turnstile_failed` (fail-closed). Der Admin-Betrieb läuft weiter; nur neue Anträge sind blockiert bis Cloudflare wieder erreichbar ist.
- **Abgelaufenes Token:** Cloudflare-Token sind einmalig verwendbar und haben eine Gültigkeitsdauer (~5 Minuten). Wenn ein Mitglied das Formular sehr lange offen lässt und dann erst absendet, schlägt die Verifikation serverseitig fehl → Frontend zeigt Fehlermeldung + Widget-Reset.
- **Widget lädt nicht** (JS blockiert, langsame Verbindung): Der Button bleibt disabled — der Nutzer kann das Formular nicht absenden. Hinweis zur Fehlersuche wird nicht im Formular angezeigt (Scope: keine explizite Fallback-UI nötig für MVP).
- **Doppeltes Absenden:** Token ist nach Verifikation einmalig verbraucht. Ein zweiter Submit mit demselben Token schlägt fehl → Frontend muss Widget nach Fehler zurücksetzen.
- **Dev/Test-Modus:** Ist `NEXT_PUBLIC_TURNSTILE_SITE_KEY` nicht gesetzt, erscheint kein Widget und es wird kein Token gesendet. Backend ohne `TURNSTILE_SECRET_KEY` überspringt die Prüfung kommentarlos.
- **Cloudflare Test-Keys:** Für automatisierte Tests (E2E) können die offiziellen Cloudflare Test-Site-Keys verwendet werden, die immer ein gültiges Token liefern.

## Technische Hinweise

- Cloudflare stellt ein offizielles React-Paket bereit: `@marsidev/react-turnstile`
- Verifikation: POST an `https://challenges.cloudflare.com/turnstile/v0/siteverify` mit `secret` + `response` (Token) im Body
- Cloudflare Test-Keys für automatisierte Tests: Site Key `1x00000000000000000000AA`, Secret Key `1x0000000000000000000000000000000AA` (immer erfolgreich)
- Token-Verifikation erfolgt serverseitig im Go-Backend (nicht im Next.js-API-Route-Layer), da der Submit direkt an das Go-Backend geht

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results

**Datum:** 2026-04-25
**Ergebnis: APPROVED — produktionsbereit**

### Acceptance Criteria
| # | Kriterium | Ergebnis |
|---|-----------|---------|
| FE-1 | Widget unterhalb Datenschutz, oberhalb Button wenn SITE_KEY gesetzt | ✅ Pass |
| FE-2 | Widget-Typ managed | ✅ Pass |
| FE-3 | Button disabled ohne Token | ✅ Pass |
| FE-4 | Token nach Widget-Lösung im State | ✅ Pass |
| FE-5 | Token als turnstileToken im Submit-Body | ✅ Pass |
| FE-6 | Kein Widget wenn SITE_KEY fehlt (Dev-Modus) | ✅ Pass |
| FE-7 | Widget-Reset nach Backend-Fehler | ✅ Pass |
| BE-1 | Serverseitige Verifikation via Cloudflare Siteverify | ✅ Pass |
| BE-2 | Verifikation übersprungen wenn kein SECRET_KEY (Dev-Modus) | ✅ Pass |
| BE-3 | HTTP 422 turnstile_failed bei ungültigem Token | ✅ Pass |
| BE-4 | HTTP 422 turnstile_missing bei fehlendem Token mit aktiven Key | ✅ Pass |
| BE-5 | Externe API (/api/external) vom Check ausgenommen | ✅ Pass |
| BE-6 | Token nicht in DB gespeichert | ✅ Pass |
| CFG-1 | NEXT_PUBLIC_TURNSTILE_SITE_KEY in .env.local.example | ✅ Pass |
| CFG-2 | Helm-Values-Beispieldateien dokumentiert | ✅ Pass |
| CFG-3 | Secret Key als Kubernetes Secret | ✅ Pass |

**Gesamt: 16/16 Kriterien bestanden**

### E2E Tests
9 Playwright-Tests in `tests/PROJ-16-turnstile-spam-protection.spec.ts`:
- Dev-Modus (kein SITE_KEY): 3 Tests ✅
- Backend Dev-Modus (kein SECRET_KEY): 1 Test ✅
- Backend mit aktivem Key: 2 Tests ✅
- Externe API ausgenommen: 1 Test ✅
- Regression (Public + Admin API): 2 Tests ✅

### Security Audit
- Turnstile-Token wird nur für Verifizierung verwendet, nicht persistiert ✅
- Externe API korrekt ausgenommen (eigene API-Key-Absicherung) ✅
- Dev-Modus ist explizit konfigurationsgesteuert, kein Standard-Bypass ✅
- Fail-closed bei Cloudflare-Ausfall (→ 422, kein Durchlass) ✅

### Bugs
Keine.

## Deployment
_To be added by /deploy_
