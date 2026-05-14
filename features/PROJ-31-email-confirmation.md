# PROJ-31: E-Mail-Adresse-Bestätigung (Anti-Abuse + Validitäts-Check)

## Status: Approved
**Created:** 2026-05-14
**Last Updated:** 2026-05-14 (Stages A–I implemented; security-review verdict YELLOW addressed via the M1/M2/L1/L2 follow-up fixes)

## Dependencies
- Requires: PROJ-1 (Public Registration) — der Submit-Pfad wird verzweigt
- Requires: PROJ-2 (Admin Review) — die neue Status-Transition wird in die Admin-Map eingewoben
- Requires: PROJ-6 (E-Mail-Benachrichtigungen) — die Bestätigungs-E-Mail wird erweitert
- Requires: PROJ-8 (Konfigurierbare Felder / EEG-Settings-Tabelle) — neue EEG-Setting-Spalte
- Berührt: PROJ-30 (Status-Transitions-Map und allowed-from/-to-Liste)

## Hintergrund

Heute kann jede Person mit der Kenntnis einer RC-Nummer einen Antrag mit beliebiger E-Mail-Adresse einreichen — auch mit fremder oder erfundener Adresse. Beobachtete und plausible Folgen:

1. **Müll-Anträge** durch Bots, die das öffentliche Formular abgrasen (Turnstile fängt nicht alles)
2. **Tippfehler** bei der E-Mail (z. B. `@gnail.com` statt `@gmail.com`) — Member bekommt keine Bestätigungsmail, EEG hat eine nicht funktionierende Adresse im Datensatz
3. **Anträge im Namen Dritter** — jemand registriert eine andere Person, ohne dass diese davon weiß

Eine E-Mail-Bestätigung mit einem klickbaren Token-Link in der Bestätigungs-E-Mail löst alle drei Fälle:
- Bots klicken die Links typischerweise nicht (anti-abuse)
- Tippfehler werden sofort erkannt (Member kriegt keine Mail → kontaktiert die EEG)
- Drittpersonen-Anträge laufen ins Leere, weil nur der echte Mailbox-Inhaber klicken kann

Das Feature ist **opt-in pro EEG** — manche EEGs (kleine Vereine mit persönlichen Kontakten) wollen die zusätzliche Hürde nicht.

## User Stories

- Als **EEG-Admin** möchte ich in den EEG-Einstellungen einen Schalter „E-Mail-Adresse bestätigen" aktivieren können, sodass neue Anträge erst nach Bestätigung der E-Mail-Adresse für mich sichtbar / aktionsfähig sind.
- Als **neues EEG-Mitglied** möchte ich in der Bestätigungs-E-Mail einen deutlich sichtbaren Button „E-Mail-Adresse bestätigen" sehen, sodass ich mit einem Klick meine Adresse verifizieren kann.
- Als **neues EEG-Mitglied** möchte ich nach dem Klick eine kurze Bestätigungsseite sehen („Vielen Dank, deine E-Mail-Adresse ist bestätigt") und dass mein Antrag nun beim EEG zur Prüfung liegt.
- Als **EEG-Admin** möchte ich in der Admin-Liste sehen, welche Anträge auf E-Mail-Bestätigung warten, sodass ich nicht versehentlich Müll-Anträge bearbeite.
- Als **EEG-Admin** möchte ich einen Antrag, dessen Bestätigungs-Mail evtl. im Spam-Ordner gelandet ist, manuell **noch einmal** verschicken können (Re-send).
- Als **EEG-Admin** möchte ich, dass Anträge, deren E-Mail-Bestätigung 30 Tage lang ausbleibt, automatisch abgelehnt werden, sodass meine Liste nicht mit toten Anträgen volläuft.
- Als **vfeeg-Betreiber** möchte ich, dass jede E-Mail-Bestätigung im `status_log` mit Zeitstempel + Quell-IP-Hash auftaucht, sodass die Aktion auditierbar ist.

## Acceptance Criteria

### EEG-Einstellung

- [ ] Neue Spalte `require_email_confirmation BOOLEAN NOT NULL DEFAULT FALSE` in `member_onboarding.registration_entrypoint`
- [ ] Neuer Admin-Settings-Toggle in der EEG-Einstellungs-Seite (`admin-settings-eeg.tsx`):
  - Label: „E-Mail-Adresse bestätigen"
  - Hilfetext: „Wenn aktiviert, müssen neue Mitglieder ihre E-Mail-Adresse über einen Bestätigungs-Link aus der Bestätigungs-Mail verifizieren, bevor du den Antrag prüfen kannst. Empfohlen als Schutz vor Müll-Anträgen und Tippfehlern."
  - Default: aus (rückwärtskompatibel)
- [ ] Setting ist über `PUT /api/admin/settings/eeg` setzbar (existierender Endpoint, neues Feld)
- [ ] Setting wird auf der **öffentlichen Config-API** (`GET /api/public/applications/config`) **NICHT** zurückgegeben — der Member braucht es nicht zu wissen; nur Backend nutzt es zur Verzweigung

### Datenbank-Modell

- [ ] Migration `000030_email_confirmation.up.sql` legt drei neue Spalten auf `member_onboarding.application` an:
  - `email_confirmed_at TIMESTAMPTZ NULL`
  - `email_confirmation_token TEXT NULL`
  - `email_confirmation_token_expires_at TIMESTAMPTZ NULL`
- [ ] Plus die `registration_entrypoint.require_email_confirmation`-Spalte (siehe oben)
- [ ] Partial UNIQUE index auf `email_confirmation_token WHERE email_confirmation_token IS NOT NULL` — verhindert Token-Kollisionen
- [ ] `.down.sql` entfernt alle vier Spalten und den Index sauber

### Status-Transition-Modell

- [ ] **Neuer Status:** `email_confirmed` — semantisch zwischen `submitted` und `under_review`
- [ ] Status-Werte-Liste (`CLAUDE.md`, `shared/`, Frontend-Translations): `draft, submitted, email_confirmed, under_review, needs_info, approved, rejected, imported, import_failed`
- [ ] Neue Transitions:
  - `submitted → email_confirmed` (ausschließlich via Member-Klick auf Bestätigungslink — kein generischer Status-Endpoint)
  - `email_confirmed → under_review` (Admin)
  - `email_confirmed → needs_info` (Admin)
  - `email_confirmed → approved` (Admin, falls Tenant Direct-Approval erlaubt)
  - `email_confirmed → rejected` (Admin)
  - `submitted → rejected` (Admin — manueller Reject auch ohne Bestätigung möglich, z. B. offensichtlicher Müll)
- [ ] Bestehende Transitions bleiben gültig (Tenants ohne Email-Confirmation-Setting flow draft → submitted → under_review → …)
- [ ] Tenant-Setting-abhängige Sperre: Wenn `require_email_confirmation = TRUE` und der Antrag im Status `submitted` ist, **lehnt** der generische Status-Endpoint die Transitions `submitted → under_review` / `submitted → needs_info` / `submitted → approved` mit **409 Conflict** und Begründung „E-Mail-Adresse nicht bestätigt" ab. `submitted → rejected` bleibt erlaubt (Admin-Override).
- [ ] Bei Tenants mit `require_email_confirmation = FALSE`: alle bestehenden Transitions funktionieren unverändert, `email_confirmed` wird nie betreten.

### Submit-Pfad

- [ ] Bei `POST /api/public/applications` (Submit):
  - Setting prüfen: `require_email_confirmation` am Tenant
  - Wenn TRUE:
    - Token generieren: `crypto/rand` 32 Byte, base64url-encoded
    - Spalten setzen: `email_confirmation_token`, `email_confirmation_token_expires_at = NOW() + 30 days`, `email_confirmed_at = NULL`
    - Status: `submitted` (wie bisher), aber `email_confirmed_at` ist NULL
    - Bestätigungs-Mail an Member enthält den Confirmation-Button (siehe Mail-Template unten)
    - EEG-Notification-Mail wird **nicht** versendet (oder versendet mit Marker „⚠ E-Mail-Adresse noch nicht bestätigt") — siehe Q3
  - Wenn FALSE (Default, bestehendes Verhalten):
    - Tokens bleiben NULL, `email_confirmed_at = NULL`
    - Verhalten unverändert (Bestätigungs-Mail ohne Button, EEG-Notification sofort)

### Bestätigungs-Mail-Template

- [ ] Datei `internal/mail/templates/application_submitted_member.html` (existierend) bekommt einen **conditional Block** `{{if .EmailConfirmationURL}}` mit einem prominent platzierten CTA:
  - Großer Button (table-based für Outlook-Kompatibilität, primäre Hintergrundfarbe der EEG)
  - Beschriftung: „E-Mail-Adresse bestätigen"
  - URL: `{{.EmailConfirmationURL}}` (vom Backend gerendert: `<NEXT_PUBLIC_BASE_URL>/confirm-email/<token>`)
  - Begleittext über dem Button: „Bitte bestätige deine E-Mail-Adresse mit einem Klick auf den Button unten. Dein Antrag wird dann von [EEG-Name] bearbeitet."
  - Hinweistext unter dem Button: „Der Link ist 30 Tage gültig. Falls du den Button nicht klicken kannst, kopiere folgende URL in deinen Browser: {{.EmailConfirmationURL}}"
  - Plain-Text-Variante (htmlToText) zeigt die URL als Text-Link
- [ ] Wenn `EmailConfirmationURL` leer ist (Setting deaktiviert), wird der Block weggelassen — kein leerer Bereich
- [ ] Identifikations-Fußzeile (PROJ-12) bleibt erhalten

### Public Confirmation-Endpoint

- [ ] Neuer Endpoint `POST /api/public/applications/confirm-email`
- [ ] Body: `{ "token": "string" }` (Token kommt aus dem URL-Pfad-Parameter und wird vom Frontend ans Backend gereicht)
- [ ] Server prüft:
  - Token existiert (matched eine `application.email_confirmation_token`)
  - Token nicht abgelaufen (`email_confirmation_token_expires_at > NOW()`)
  - Antrag im Status `submitted` (idempotent: wenn schon `email_confirmed`, gib 200 zurück, kein Fehler)
- [ ] Bei Erfolg:
  - Transaktion: Status `submitted → email_confirmed`, `email_confirmed_at = NOW()`, Token-Spalten auf NULL (One-Time-Use)
  - `status_log`-Eintrag: `from=submitted`, `to=email_confirmed`, `changed_by_user_id='member'`, `reason='E-Mail-Adresse über Bestätigungs-Link bestätigt'`
  - EEG-Notification-Mail jetzt erst versenden (oder mit „bestätigt"-Marker, je nach Q3)
- [ ] Bei Token nicht gefunden / abgelaufen: 400 mit Code `email_confirmation_invalid` und Message „Der Bestätigungs-Link ist ungültig oder abgelaufen."
- [ ] Rate-Limit: 5 Versuche pro IP pro 10 Minuten (gegen Token-Brute-Force)
- [ ] Body-Size-Limit: 1 KiB (Token ist klein)

### Member-facing Confirmation-Seite

- [ ] Neue Frontend-Route: `/confirm-email/[token]` (Next.js Server Component, kein Auth)
- [ ] Auf Mount: ruft `POST /api/public/applications/confirm-email` mit dem URL-Token
- [ ] Drei mögliche Zustände:
  - **Lädt** (kurz): Spinner + „Bestätige deine E-Mail-Adresse …"
  - **Erfolg**: ✓ „Vielen Dank — deine E-Mail-Adresse ist bestätigt. Dein Antrag liegt jetzt bei [EEG-Name] zur Prüfung." (EEG-Name kommt aus der Response, NICHT aus der URL)
  - **Fehler**: ✗ „Der Bestätigungs-Link ist ungültig oder abgelaufen. Bitte wende dich an deine EEG, falls du eine neue Bestätigungs-Mail benötigst." + EEG-Kontakt-Adresse (sofern aus Response verfügbar)
- [ ] Keine sensiblen Antragsdaten in der Response (kein Name, keine IBAN, keine Telefonnummer)
- [ ] Mobile-optimiert (Member klickt häufig vom Smartphone)

### Admin-UI: Sichtbarkeit unbestätigter Anträge

- [ ] Admin-Liste: Status-Badge für `email_confirmed` (neu, primärfarben „Bestätigt") und visueller Indikator für `submitted` mit aktivem `require_email_confirmation`-Setting (z. B. zusätzliches gelbes Badge „⏳ E-Mail unbestätigt")
- [ ] Filter „Status" listet `email_confirmed` als Option
- [ ] Admin-Detail-Seite zeigt im Status-Block:
  - Wenn `require_email_confirmation=TRUE` und `email_confirmed_at=NULL`: orangener Banner „E-Mail-Adresse noch nicht bestätigt. Link wurde am DD.MM.YYYY HH:MM versendet, ablaufend am DD.MM.YYYY."
  - Wenn `email_confirmed_at != NULL`: grünes Badge „✓ E-Mail bestätigt am DD.MM.YYYY HH:MM" im Header
- [ ] Status-Action-Buttons im `submitted`-Status: bei aktiver Setting nur „Ablehnen" und „Bestätigungs-Mail erneut senden" sichtbar — andere Aktionen disabled mit Tooltip „E-Mail-Adresse muss zuerst bestätigt werden"

### Admin-Aktion: Bestätigungs-Mail erneut senden

- [ ] Neuer Endpoint `POST /api/admin/applications/{id}/resend-confirmation`
- [ ] Vorbedingung: Antrag in Status `submitted`, EEG hat `require_email_confirmation=TRUE`, `email_confirmed_at IS NULL`
- [ ] Aktion:
  - Neuen Token generieren (alter Token wird ersetzt — alte Links werden ungültig)
  - `email_confirmation_token_expires_at` auf `NOW() + 30 days` zurücksetzen
  - Bestätigungs-Mail (Template wie beim Submit) erneut versenden
  - `status_log`-Eintrag: „Bestätigungs-Mail erneut versendet" (kein Status-Wechsel)
- [ ] Frontend: Button „Bestätigungs-Mail erneut senden" im Detail-Header eines `submitted`-Antrags mit aktiver Setting

### Auto-Reject nach 30 Tagen

- [ ] Backend-Hintergrundjob (cron-style oder beim Server-Start + täglich) sucht Anträge mit:
  - `email_confirmation_token_expires_at < NOW()`
  - `email_confirmed_at IS NULL`
  - Status `submitted`
- [ ] Pro Match: Transaktion
  - Status `submitted → rejected`
  - `rejected_at = NOW()`
  - `admin_note` (oder dediziertes Reason-Feld) → „E-Mail-Bestätigung ausgeblieben (Auto-Reject nach 30 Tagen)"
  - `status_log`-Eintrag mit `changed_by_user_id='system'`, `reason='E-Mail-Bestätigung ausgeblieben'`
- [ ] Keine E-Mail-Versendung beim Auto-Reject (Member hat ohnehin nicht reagiert)
- [ ] Job-Frequenz: 1× pro Tag. Implementierung: `time.Tick` in einer Goroutine in `cmd/server/main.go` oder als separater Cron-Container — siehe Q4

### Audit & Beobachtbarkeit

- [ ] `status_log`-Einträge wie oben spezifiziert
- [ ] Prometheus-Counter:
  - `email_confirmation_sent_total{result}` (`sent` / `resent`)
  - `email_confirmation_completed_total`
  - `email_confirmation_expired_total`
- [ ] `slog.Info` bei Confirmation mit `application_id`, `ip_hash` (SHA-256 vom realIP, nicht die volle IP — Privacy), `latency_ms`
- [ ] Confirmation-Klick wird **nicht** zur applikations-spezifischen Endpoint-Latenz-Histogramm gezählt (eigene Route)

### Dokumentation

- [ ] `CLAUDE.md` Status-Modell-Sektion: neuer Status `email_confirmed` + Transitions ergänzen
- [ ] `docs/api-spec.md`: zwei neue Endpoints (`POST /api/public/applications/confirm-email`, `POST /api/admin/applications/{id}/resend-confirmation`)
- [ ] `docs/domain-model.md`: vier neue Spalten dokumentieren
- [ ] `docs/swagger.yaml` (PROJ-24): Endpoints + Datentypen ergänzen
- [ ] `docs/user-guide/05-admin-status.md`: neuer Status `email_confirmed` in Diagramm + Tabelle; Resend-Aktion dokumentieren
- [ ] `docs/user-guide/06-admin-settings.md`: neuer Setting-Toggle „E-Mail-Adresse bestätigen"

## Edge Cases

- **Member klickt Link zweimal:** zweiter Klick ist No-Op (idempotent, 200 OK, kein Fehler). `email_confirmation_token` ist beim ersten Klick auf NULL gesetzt — zweiter Klick findet kein Match → eigentlich 400. **Spec-Entscheidung:** Server prüft zuerst, ob die `application` mit `email_confirmed_at IS NOT NULL` und einer Spur dieses Tokens (per `email_confirmed_at`-Existenz-Check ist ohne Token-Match-Trace nicht möglich — siehe Q5). Workaround: nach erstem Erfolg leitet das Frontend auf eine erfolgs-Seite, die nicht erneut die API ruft.
- **Member klickt nach Auto-Reject:** Token-Spalten sind beim Auto-Reject ebenfalls auf NULL gesetzt → Frontend zeigt Fehler-Zustand. Hilfetext erklärt, dass die EEG kontaktiert werden muss.
- **EEG aktiviert Setting nachträglich:** Bestehende Anträge im Status `submitted` werden **nicht** rückwirkend zu Pending-Confirmation. Sie laufen weiter im Alt-Flow. Setting wirkt nur auf neue Submits.
- **EEG deaktiviert Setting nachträglich:** Bestehende Anträge im Status `submitted` mit `email_confirmed_at=NULL` bleiben so — Admin kann sie nach Deaktivierung wieder normal weiterbearbeiten. Token-Spalten bleiben befüllt (kein Cleanup), das verfallene Token wird beim 30-Tage-Job aufgeräumt.
- **Token-Kollision:** UNIQUE-Index verhindert das. Bei `crypto/rand` 32 Byte ist das Risiko ohnehin astronomisch klein.
- **Token-Brute-Force:** 5 Versuche pro IP pro 10 min, plus die 32-Byte-Entropie (≈ 2²⁵⁶). Praktisch unangreifbar.
- **Mail-Client öffnet Links automatisch beim Scannen (Outlook Safe-Links, Microsoft Defender):** Diese Pre-Fetcher könnten den Token vorzeitig konsumieren. **Mitigation:** Confirmation-Endpoint nimmt nur `POST` an, nicht `GET`. Der Klick im Mail-Client läuft auf eine GET-Seite, die JS den POST auslöst. Pre-Fetcher führen i. d. R. kein JS aus.
- **Member-Endpoint ist öffentlich ohne Auth:** absichtlich. Sicherheit kommt vom 32-Byte-Token (nicht erratbar).
- **Race: Member klickt Link, gleichzeitig läuft der 30-Tage-Auto-Reject:** Beide bewegen den Antrag aus `submitted` heraus. Transaktion + Vorbedingung „Status == submitted" verhindert Doppel-Move. Wer zuerst kommt, gewinnt; der zweite scheitert mit 409 und bricht ab.

## Technical Requirements

- **Sicherheit:** Status-Transition-Map wird erweitert — fällt unter `.claude/rules/security.md` Code Review Trigger („Any changes to status transition rules" + „Any changes to public registration endpoints"). `/security-review` ist **verpflichtend** vor Produktiv-Use.
- **Privacy:** Confirmation-Page darf **keine** persönlichen Daten preisgeben (keine Antragsnummer, kein Name, keine E-Mail). Token soll keine Information leaken; reine 256-bit-Entropie.
- **Crypto:** `crypto/rand` für Token-Generierung, `subtle.ConstantTimeCompare` beim Token-Match (Timing-Attack-Schutz). Token wird **als Hash** (SHA-256) in der DB gespeichert, der Plaintext nur in der E-Mail-URL — verhindert DB-Dump-Risiko. (Siehe Q2.)
- **Rate-Limiting:** Existierende Rate-Limit-Middleware (10 Req / 10 min pro IP für `POST /api/public/applications`) bekommt einen zweiten Eintrag für `POST /api/public/applications/confirm-email` mit 5 Req / 10 min.
- **Job-Reliability:** Auto-Reject-Job läuft idempotent — Wiederholung mit identischem Result, falls Job zweimal startet (z. B. nach Crash). Lock via Postgres-Row-Lock auf dem `application`-Row während der Transaktion.
- **Mail-Deliverability:** keine Verschlechterung gegenüber heute. Der zusätzliche CTA-Button bricht keine SPF/DKIM/DMARC-Auth.
- **Rückwärtskompatibilität:** Tenants ohne Setting (= alle bestehenden EEGs nach Migration, Default FALSE) erleben **null** Verhaltensänderung. Bestehende Anträge im Status `submitted` werden nicht angefasst.

## Open Questions

### Q1: Status-Modell — neuer Status oder reine Flag-Lösung?

**Variante A (gewählt, in den ACs umgesetzt):** Neuer Status `email_confirmed` zwischen `submitted` und `under_review`. Macht die Verifikation im Status-Verlauf explizit sichtbar und passt zur User-Request-Formulierung.

**Variante B (alternativ):** Statt eines neuen Status nur ein Flag `email_confirmed_at TIMESTAMPTZ`. Status-Machine bleibt unverändert. Sichtbarkeit / Sperre via Tenant-Setting-Abfrage in jedem Transition-Check.

**Trade-off:** A ist semantisch klarer (Status zeigt explizit den Schritt), B ist code-leichter und vermeidet das „Tenants ohne Setting haben einen Status nie betreten"-Phänomen. Wenn das Feature später ausgeweitet werden soll (z. B. SMS-Bestätigung zusätzlich), ist B flexibler.

**Empfehlung:** A, weil die User-Anforderung explizit einen Status verlangt.

### Q2: Token im Klartext oder als Hash in der DB speichern?

**Variante A:** Plaintext-Token in `email_confirmation_token TEXT`. Vorteil: einfacher Lookup (`WHERE token = $1`). Nachteil: DB-Dump leakt aktive Tokens.

**Variante B:** Token-Hash (SHA-256) in der DB. Vorteil: DB-Dump enthüllt keine Tokens. Nachteil: Lookup-Pfad braucht erst Hash-Berechnung. Bei einem einzelnen Antrag (Member kennt seinen Token aus der Mail) trivial.

**Empfehlung:** B. Aufwand minimal, Sicherheitsgewinn deutlich. Token-Lookup: `SELECT id FROM application WHERE email_confirmation_token = encode(sha256($1), 'hex')`.

### Q3: EEG-Notification-Mail bei aktiver Confirmation — sofort, nach Bestätigung, oder beides?

- (a) **Sofort bei Submit**, mit Marker „⚠ E-Mail noch nicht bestätigt" im Betreff
- (b) **Erst nach Bestätigung**, normale Notification
- (c) **Beides**: Sofort als kurze Info („1 neuer Antrag eingelangt, wartet auf E-Mail-Bestätigung"), dann nochmal bei Bestätigung

**Empfehlung:** (b). Das ganze Feature gibt es, weil Müll-Anträge die EEG nicht erreichen sollen. Eine Sofort-Notification untergräbt das. (a) und (c) erzeugen Mail-Lärm.

### Q4: Auto-Reject-Job — In-Process-Goroutine oder externer Cron?

- (a) **In-Process-Goroutine** im Backend (`time.Ticker` mit 24h-Intervall, beim Server-Start). Vorteil: keine Extra-Infra. Nachteil: bei Multi-Replica-Deployment (S6 aus dem 2026-05-14-Memo) Race-Bedingungen — entweder Postgres-Lock oder Leader-Election nötig.
- (b) **Externer Cron** (Kubernetes-CronJob als Sidecar zum Helm-Chart). Vorteil: keine Replica-Probleme, kein State im App. Nachteil: zusätzliches Helm-Resource.
- (c) **In-Process mit Postgres-Advisory-Lock**: Beim Start-of-Job versucht jeder Replica `pg_try_advisory_lock`. Nur eine bekommt den Lock und führt aus. Lock wird in `pg_advisory_unlock` released.

**Empfehlung:** (a) jetzt + (c) sobald Multi-Replica kommt. Single-Replica-Deployment ist S6 (parked); für jetzt reicht die simple Variante. Im Code so faktorisieren, dass der Lock-Switch zu (c) eine Ein-Zeilen-Änderung wird.

### Q5: Idempotenz — was passiert, wenn Member den Link mehrfach klickt?

- (a) Confirmation-Endpoint löscht beim Erfolg den Token aus der DB. Zweiter Klick → 400 „Token nicht gefunden". Member ist verwirrt („wieso geht der Link aus meiner Bestätigungs-Mail nicht mehr?").
- (b) Confirmation-Endpoint behält den Token-Hash, markiert ihn als „verbraucht" (z. B. `email_confirmation_used_at TIMESTAMPTZ`). Zweiter Klick → 200 „Bereits bestätigt".
- (c) Frontend leitet nach erstem Erfolgs-Response auf eine andere URL um (z. B. `/confirm-email/success`), sodass ein zweiter Klick auf den Mail-Link zwar 400 liefert, aber das UX-Problem nicht auftaucht.

**Empfehlung:** (b). Zusätzliche Spalte ist billig, UX deutlich freundlicher (Member bekommt „bereits bestätigt"-Bestätigung). Spec oben ist mit (c) formuliert — bitte vor Implementation entscheiden, ob ich auf (b) umstelle.

### Q6: Resend-Limit pro Antrag?

Der Admin-Resend-Endpoint ist offen — ein Admin könnte 100× klicken und 100 Mails versenden. Soll es ein Limit geben?

- (a) Kein Limit (Admin-Vertrauen)
- (b) Max. 3 Resends pro Antrag, danach Sperre
- (c) Throttle: mind. 5 Minuten zwischen zwei Resends desselben Antrags

**Empfehlung:** (c). Verhindert Versehen ohne den Admin zu gängeln.

## Notes

- Migration `000030_email_confirmation` ist **strikt additiv** — keine Datenmigration alter Anträge nötig. Rückwärtskompatibel deploybar.
- Realistische Implementierungsdauer: 4–6 Stunden (Backend + Frontend + Mail-Template + Tests + Doku), plus Security-Review.
- Frontend-Bündel wird leicht größer (neue `/confirm-email/[token]`-Route — aber als Server-Component renderbar, minimaler JS-Footprint).
- Keine neuen Go- oder npm-Pakete erforderlich. `crypto/rand` + `crypto/sha256` + `encoding/base64` aus der Standardlibrary.

---
<!-- Sections below are added by subsequent skills -->
