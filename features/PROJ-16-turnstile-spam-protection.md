# PROJ-16: Cloudflare Turnstile — Spam-Schutz für das Beitrittsformular

## Status: Planned
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

- [ ] Das Cloudflare Turnstile-Widget wird im Beitrittsformular unterhalb der Datenschutz-Checkbox und oberhalb des Absende-Buttons angezeigt, wenn `NEXT_PUBLIC_TURNSTILE_SITE_KEY` gesetzt ist.
- [ ] Das Widget ist vom Typ `managed` (Cloudflare entscheidet, ob eine interaktive Challenge nötig ist — meist unsichtbar).
- [ ] Der Absende-Button ist deaktiviert (`disabled`), solange kein gültiges Turnstile-Token vorliegt.
- [ ] Nach erfolgreichem Widget-Lösen wird das Token im Formular-State gespeichert und der Button aktiviert.
- [ ] Das Token wird beim Absenden als Feld `turnstileToken` im JSON-Body an das Backend gesendet.
- [ ] Ist `NEXT_PUBLIC_TURNSTILE_SITE_KEY` **nicht** gesetzt, wird kein Widget angezeigt und das Formular funktioniert wie bisher (Dev-Modus ohne CAPTCHA).
- [ ] Nach einem Fehler beim Absenden (Backend lehnt Token ab) wird das Widget zurückgesetzt und ein Hinweis angezeigt.

### Backend

- [ ] Das Backend verifiziert das `turnstileToken`-Feld aus dem Request-Body gegen die Cloudflare Siteverify-API (`https://challenges.cloudflare.com/turnstile/v0/siteverify`).
- [ ] Ist `TURNSTILE_SECRET_KEY` **nicht** konfiguriert, wird die Verifikation übersprungen (Dev-Modus — kein Fehler).
- [ ] Ein fehlgeschlagener Turnstile-Check gibt HTTP 422 mit Fehlercode `turnstile_failed` zurück.
- [ ] Ein fehlendes `turnstileToken` bei aktiviertem Secret Key gibt HTTP 422 mit `turnstile_missing` zurück.
- [ ] Die externe Registrierungs-API (PROJ-13, `/api/external/registration`) ist vom Turnstile-Check ausgenommen — diese Schnittstelle ist bereits über API-Key gesichert.
- [ ] Das Turnstile-Token wird **nicht** in der Datenbank gespeichert.

### Konfiguration

- [ ] `NEXT_PUBLIC_TURNSTILE_SITE_KEY` ist in `.env.local.example` dokumentiert.
- [ ] `TURNSTILE_SITE_KEY` (für das Frontend-SSR, wenn nötig) und `TURNSTILE_SECRET_KEY` (Backend) sind in den Helm-Values-Beispieldateien dokumentiert.
- [ ] Der Secret Key wird als Kubernetes Secret verwaltet (analog zu anderen Secrets im Chart).

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
_To be added by /qa_

## Deployment
_To be added by /deploy_
