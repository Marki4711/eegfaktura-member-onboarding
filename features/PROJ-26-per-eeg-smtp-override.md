# PROJ-26: Eigener Mailserver pro EEG

## Status: Planned
**Created:** 2026-05-08
**Last Updated:** 2026-05-08

## Dependencies
- Requires: PROJ-6 (E-Mail-Benachrichtigungen) — bestehende Mail-Infrastruktur
- Requires: PROJ-3 (Admin Frontend UI) — Konfiguration in EEG-Einstellungsseite
- Requires: PROJ-5 (Keycloak-secured Admin Area) — nur authentifizierte Admins dürfen SMTP-Daten setzen

## Hintergrund

Aktuell laufen alle ausgehenden E-Mails (Mitgliederbestätigungen, EEG-Benachrichtigungen, Approval-Mails mit PDF) über den zentral konfigurierten SMTP-Server der vfeeg.

Einige EEGs möchten aus Compliance-, Branding- oder Datenschutzgründen ihre eigenen Mailserver verwenden. Zusätzlich besteht der Wunsch, dass die hinterlegten SMTP-Zugangsdaten **nicht durch vfeeg-Operatoren ausgelesen werden können** — das Plaintext-Passwort soll weder in der Datenbank noch in Logs einsehbar sein.

## User Stories

- Als **EEG-Admin** möchte ich für meine EEG einen eigenen SMTP-Server konfigurieren können, sodass alle Mails dieser EEG über meinen Mailserver versendet werden.
- Als **EEG-Admin** möchte ich eine Test-E-Mail an mich selbst verschicken können, bevor ich die Konfiguration speichere, sodass ich Fehlkonfigurationen vor dem Echtbetrieb erkenne.
- Als **EEG-Admin** möchte ich sicher sein, dass das hinterlegte Passwort **nicht von vfeeg-Operatoren** im Klartext gelesen werden kann, sodass meine Zugangsdaten geschützt sind.
- Als **EEG-Admin** möchte ich den Override jederzeit deaktivieren können (Rückfall auf vfeeg-Mailserver), sodass ich bei Problemen mit meinem eigenen Server zurückwechseln kann.
- Als **vfeeg-Betreiber** möchte ich, dass eine fehlgeschlagene Mail über einen EEG-eigenen Server **nicht** stillschweigend auf meinen Mailserver zurückfällt, sodass die Datenflussgrenze klar bleibt (DSGVO).
- Als **vfeeg-Betreiber** möchte ich, dass Fehler beim Mailversand pro EEG geloggt und der zuständige Admin benachrichtigt wird, sodass Probleme zeitnah erkannt werden.

## Acceptance Criteria

### Konfiguration
- [ ] Im Admin-UI in der bestehenden EEG-Einstellungsseite gibt es einen neuen Abschnitt "Eigener Mailserver"
- [ ] Felder: SMTP-Host, Port, Username, Passwort, From-Adresse, From-Name (alle optional)
- [ ] Ein Toggle "Eigenen Mailserver verwenden" aktiviert/deaktiviert den Override
- [ ] Wenn der Override deaktiviert ist, wird der vfeeg-Mailserver verwendet (heutiges Verhalten)
- [ ] Bei aktiviertem Override sind Host, Port, From-Adresse Pflichtfelder; User/Passwort optional (für offene Relays)
- [ ] Das Passwort-Feld zeigt nach dem Speichern niemals das Klartext-Passwort an — nur einen Platzhalter wie `••••••••` oder den Hinweis "gesetzt"
- [ ] Beim erneuten Speichern ohne Passwort-Änderung bleibt das bisher gespeicherte Passwort erhalten

### Test-E-Mail
- [ ] Button "Test-E-Mail senden" sendet eine Probemail an die E-Mail-Adresse des angemeldeten Admin-Users (aus Keycloak-JWT)
- [ ] Test-E-Mail nutzt die **aktuell im Formular eingegebenen Werte**, nicht die gespeicherten — so kann der Admin die Werte vor dem Speichern verifizieren
- [ ] Bei erfolgreichem Versand: grüne Bestätigung mit Hinweis "Posteingang prüfen"
- [ ] Bei Fehler: konkrete Fehlermeldung (z.B. "Authentifizierung fehlgeschlagen", "Verbindung verweigert", "Timeout")
- [ ] Test-E-Mail kann **nicht ohne** vorherigen erfolgreichen Test gespeichert werden? → **Open Question** (siehe unten)

### Mailversand-Verhalten
- [ ] Bei aktivem Override werden **alle** EEG-bezogenen Mails über den eigenen Server versendet:
  - Mitgliederbestätigung (Submission, Approval, Rejection, Needs-Info)
  - EEG-Benachrichtigung (neue Anträge, Status-Änderungen)
  - PDF-Anhänge (SEPA, Approval-PDF)
  - Admin-Resends
- [ ] Bei Fehler im EEG-eigenen Mailversand: **kein** automatischer Fallback auf vfeeg-Mailserver
- [ ] Fehler werden in Backend-Logs geschrieben (mit `slog.Error`, ohne Passwort)
- [ ] Fehler erscheinen optional im Admin-UI als Warning ("Letzter Mailversand fehlgeschlagen am ...") — **Open Question** (siehe unten)

### Sicherheit
- [ ] Das SMTP-Passwort wird **niemals** als Klartext in der Datenbank gespeichert
- [ ] Das SMTP-Passwort wird **niemals** in Logs ausgegeben (auch nicht in Debug-Logs, Stack Traces, Error-Messages)
- [ ] Das SMTP-Passwort erscheint **niemals** in API-Responses (GET-Endpoints geben nur ein "isSet"-Flag zurück, kein Wert)
- [ ] Das Verschlüsselungsverfahren ist im Security-Review dokumentiert
- [ ] Tenant-Isolation: ein Admin von EEG A kann SMTP-Daten von EEG B weder lesen noch schreiben
- [ ] Nur Tenant-Admins der jeweiligen EEG (oder Superuser) dürfen SMTP-Override konfigurieren

## Edge Cases

- Was passiert, wenn der Admin den Override aktiviert, aber kein Passwort hinterlegt? → Versuch ohne Auth (für offene Relays); bei Fehler klare Fehlermeldung
- Was passiert, wenn der EEG-eigene Server nach erfolgreicher Test-Mail später ausfällt? → Mail bleibt unversendet, Fehler wird geloggt, Admin sieht Warnung
- Was passiert bei Resend einer fehlgeschlagenen Mail? → erneuter Versuch über EEG-eigenen Server, kein Fallback
- Was passiert, wenn das Master-Encryption-Key (für die DB-Verschlüsselung) verloren geht? → alle hinterlegten SMTP-Passwörter werden unbrauchbar; Admins müssen sie neu eingeben
- Was passiert, wenn ein Admin die SMTP-Konfiguration löscht und gleich wieder neu setzt? → altes Passwort wird überschrieben, neues Passwort wird verschlüsselt gespeichert
- Was passiert, wenn From-Adresse nicht zur SMTP-Domain passt (SPF/DKIM)? → Mail wird vom Empfänger-Server abgelehnt; nicht das Problem von vfeeg, aber dokumentieren
- Was passiert bei Migration einer EEG vom vfeeg-Mailserver auf den eigenen? → bestehende Anträge werden ab Aktivierung über den neuen Server versendet, keine Backfill-Aktion
- Was passiert, wenn mehrere Mails gleichzeitig versendet werden? → bestehender Mail-Semaphore (`acquireMailSem`) greift weiterhin

## Technical Requirements

- **Performance:** Der per-EEG-Lookup der SMTP-Config darf den Mailversand nicht spürbar verlangsamen (Caching erlaubt, bei Update invalidieren)
- **Security:** SMTP-Passwort verschlüsselt gespeichert, niemals im Klartext in DB/Logs/API-Responses
- **Konsistenz:** Bestehende vfeeg-Mailserver-Konfiguration (`SMTP_HOST` etc.) bleibt als globaler Fallback erhalten
- **Rückwärtskompatibilität:** EEGs ohne Override-Konfiguration nutzen weiterhin den vfeeg-Mailserver (Default-Verhalten)

## Open Questions / Options zu evaluieren

### Q1: Passwort-Schutz-Strategie (Hauptthema)

Die Anforderung "vfeeg-Operator kann das Passwort nicht auslesen" hat eine **fundamentale Spannung**: Der vfeeg-Backend muss das Passwort zur Laufzeit verwenden, um Mails zu versenden. Echte kryptographische Zero-Knowledge ist daher nicht möglich. Was realistisch erreichbar ist: keine Plaintext-Speicherung, keine zufällige Sichtbarkeit, auditierbare Entschlüsselungen.

**Optionen zur Evaluierung:**

| # | Ansatz | Schutz-Niveau | Aufwand | Bemerkung |
|---|--------|---------------|---------|-----------|
| **A** | AES-GCM mit Master-Key in Env-Var (Kubernetes Secret) | Niedrig | Niedrig | vfeeg-Ops mit DB- und Cluster-Zugriff können entschlüsseln. Schutz nur gegen DB-Dumps und Backups |
| **B** | AES-GCM mit Master-Key aus externem KMS (HashiCorp Vault, Azure Key Vault, AWS KMS) | Mittel | Mittel | Entschlüsselungen auditierbar; vfeeg-Ops braucht KMS-Zugriff. Realistische Verbesserung |
| **C** | Customer-Managed Encryption Key (CMEK) — EEG hinterlegt eigenen Public Key, vfeeg verschlüsselt damit | Mittel-Hoch | Hoch | Backend braucht Private Key zur Laufzeit für Decrypt → Wer hostet ihn? Wenn vfeeg → wie A/B. Wenn EEG → Per-Request-Übergabe nicht praktikabel für autonome Mails |
| **D** | OAuth-based SMTP (Microsoft 365, Google Workspace) | Hoch | Hoch | Kein Passwort, sondern Refresh-Tokens (revocable, scoped). Funktioniert nur für Provider mit OAuth-SMTP |
| **E** | EEG-eigene SMTP-Relay/Gateway: vfeeg authentifiziert mit relayspezifischem Token, EEG-Server hat echte Credentials | Hoch | Sehr hoch | EEG muss eigenen Relay betreiben. vfeeg sieht nur Relay-Token, nie das echte SMTP-Passwort |
| **F** | Hybrid: Default = Option B; EEGs mit höchsten Anforderungen können auf Option D oder E migrieren | — | Mittel | Pragmatischer Mehrstufen-Ansatz |

**Empfehlung für Diskussion:** Option **B mit Roadmap zu D/E**. Damit haben wir baseline einen sauberen, nicht-trivialen Schutz, und EEGs mit M365/Google können auf OAuth migrieren, sobald Bedarf besteht.

### Q2: Wer darf SMTP-Override konfigurieren?

- (a) Nur Superuser (vfeeg)
- (b) Tenant-Admin der EEG (Standard für andere EEG-Settings)
- (c) Beide (Superuser kann im Auftrag des Admins setzen)

**Empfehlung:** (b) Tenant-Admin, konsistent mit anderen EEG-Settings (Intro-Text, Felder, Rechtsdokumente).

### Q3: Test-Mail vor Speichern verpflichtend?

- (a) Verpflichtend — keine Speicherung ohne erfolgreichen Test
- (b) Optional — Admin kann ohne Test speichern, akzeptiert das Risiko

**Empfehlung:** (a) Verpflichtend. Eine fehlerhafte SMTP-Konfiguration verschluckt sonst Mitglieder-Bestätigungen ohne Vorwarnung.

### Q4: Sichtbare Fehler-Anzeige im Admin-UI?

- (a) Nur Backend-Logs (Status-quo)
- (b) Banner im Admin-UI bei letzter fehlgeschlagener Mail
- (c) Banner + E-Mail-Benachrichtigung an Admin

**Empfehlung:** (b) Banner. (c) ist nett, aber wenn der Mailserver kaputt ist, kommt die Benachrichtigung sowieso nicht an.

### Q5: Wo wird die SMTP-Config gespeichert?

- (a) Erweiterung der bestehenden `registration_entrypoint`-Tabelle
- (b) Neue Tabelle `member_onboarding.eeg_smtp_config` (1:1 zu `registration_entrypoint`)

**Empfehlung:** (b) — separate Tabelle. Saubere Isolation, optional, einfache Migration ohne Risiko für bestehende EEG-Daten. Außerdem kann ein zukünftiger Schwenk zu OAuth (Option D) das Schema einfacher erweitern.

### Q6: Welche EEG-Daten dürfen im Footer der Mail erscheinen?

Wenn EEGs eigene Server nutzen, könnten sie eigene Footer-Texte (Impressum, Kontakt) wünschen. → **Out of scope für PROJ-26**, könnte ein Folge-Feature sein.

## Notes

- Die Konfiguration ist **opt-in** pro EEG. Default bleibt der vfeeg-Mailserver.
- Spec wurde nach Default-Empfehlung des Skills `/grill-me` erstellt — die Open Questions Q1–Q5 sind explizite Kandidaten für `/grill-me` vor `/architecture`.
- Security-Review (`/security-review`) ist verpflichtend, da neue Tabelle, Secret-Storage, externe Mail-Delivery-Pfade und DSGVO-relevante Daten betroffen sind.

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
