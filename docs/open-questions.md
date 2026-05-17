# Open Questions

Lebendes Dokument für Fragen, die noch nicht final entschieden sind und Architektur-Implikationen haben. Aufgelöste Fragen bleiben mit Verweis auf die Feature-Spec stehen, damit der Diskussionsverlauf nachvollziehbar bleibt.

---

## Offene Fragen

### OQ-3: Mail-Strategie — durchgängig hard-fail vs. Mischform?

**Kontext:**
Aktuell sind nur die Status-Change-Mails an das Mitglied (Ablehnung PROJ-41, Rückfrage PROJ-43) synchron + hard-fail: scheitert SMTP, wird die Statusänderung zurückgerollt und der Admin sieht den Fehler sofort. Alle anderen Mail-Pfade sind best-effort async (Goroutine + Log + Prometheus, blockiert aber nichts):

- **Submit-Mails** (member-confirmation + EEG-notification) — best-effort
- **Post-Import-Mails** (PROJ-46 Stage B + PROJ-47: Beitrittsbestätigung + ggf. B2B-Mandat-Anhang an Member + EEG-Kopie) — best-effort
- **Activation-Mail** (PROJ-46, beim Übergang auf `activated`) — best-effort

**Offene Fragen:**
- Sollen alle Pfade auf hard-fail umgestellt werden?
- Falls ja: wie geht das System mit Submit-Mail-Fehlern um? Bei SMTP-Down kann sich dann kein Mitglied registrieren — schwerer Outage-Faktor für eine public-facing Form.
- Falls Mischform bleibt: dokumentieren als bewusste Entscheidung, oder ist „best-effort plus Mail-Outbox" die saubere Lösung für die nicht-Status-Mails?

**Impact:**
Mail-Outbox-Implementation (Retry-Queue) wäre Mittelweg — Outage-Resistenz + kein Mailverlust. Aufwand mittel (~1 Sprint), kein Blocker für aktuelle Funktionalität.

**Status:** Offen — dokumentiert auch in `docs/operations.md` (Section 2.2). Nächster Architektur-Review.

---

### OQ-4: Mehrere Mail-Anhänge — Single-Mail vs. zwei separate Mails?

**Kontext:**
Seit PROJ-47 bekommt ein B2B-Antragsteller beim Import zwei PDFs im selben Mail (Beitrittsbestätigung + Firmenlastschrift-Mandat mit Mandatsreferenz). Manche Mail-Clients zeigen Multi-Anhänge schlechter an als zwei separate Mails (z.B. mobile Outlook stapelt).

**Offene Frage:**
Bei besser separater Mails — sollte die Architektur perspektivisch auf „zwei einzelne Mails mit klarem Bezug" wechseln, oder reicht der aktuelle Single-Mail-Ansatz?

**Status:** Niedrige Dringlichkeit. Beobachten ob Member-Feedback kommt; entscheiden wenn Datenpunkte vorliegen.

---

### OQ-5: Aktivierung im Core — Lese-Konflikt bei sehr großen EEGs?

**Kontext:**
`POST /api/admin/applications/check-activation` (PROJ-46 Stage D) ruft pro Tenant einmal `GET /participant` im Core auf und cappt den Response-Body bei 4 MiB / ~2000 Teilnehmern. Bei EEGs jenseits dieser Größe schlägt der JSON-Decode fehl (silent truncation würde Daten verfälschen).

**Offene Frage:**
Wann erreicht eine produktive EEG > 2000 Teilnehmer? Wenn das absehbar ist, brauchen wir entweder ein „thinner" Core-Endpoint (id + status only) oder pagination.

**Impact:**
Aktuell kein Blocker — größte produktive EEG liegt deutlich unter 2000. Wenn die Zahl näher rückt: Core-Team einbinden für Pagination oder dedizierten Status-Endpoint.

**Status:** Beobachten. Nächste Eskalations-Schwelle: 1500 Teilnehmer pro EEG.

---

## Aufgelöste Fragen

### OQ-1: Documents in the Registration Form *(resolved)*

**Ursprüngliche Frage:** Welche Rechtsdokumente werden im Registrierungsformular gezeigt, EEG-spezifisch oder operator-weit, mit oder ohne Pflicht-Checkbox, mit oder ohne Audit-Trail?

**Auflösung (durch PROJ-9, PROJ-18, PROJ-36):**
- **PROJ-9** liefert pro-EEG konfigurierbare Rechtsdokumente (Satzung, Nutzungsbedingungen etc.), administrierbar im Admin-Web. Max 10 Dokumente pro EEG.
- **PROJ-18** trennt die zentrale Datenschutzerklärung des Operators (env-konfiguriert via `CENTRAL_POLICY_TITLE` / `CENTRAL_POLICY_URL`) von den EEG-spezifischen Dokumenten. Pro EEG via `show_central_policy` aktivierbar.
- **PROJ-36** ergänzt zwei Modi: Pflicht-Dokument mit Checkbox („explicit consent") und Info-Dokument ohne Checkbox („informational acknowledgement"). Beide werden in `document_consent` als unveränderlicher Snapshot je Antrag protokolliert (Titel + URL + Timestamp + ConsentType).

Mit diesen drei Features sind alle ursprünglichen Teilfragen abgedeckt.

---

### OQ-2: Formal Requirements for the SEPA Direct Debit Mandate *(resolved)*

**Ursprüngliche Frage:** Reicht eine Checkbox-Zustimmung als SEPA-Mandat? Was muss formal enthalten sein (Creditor-ID, Mandatsreferenz)? Muss das Mandat zugestellt werden?

**Auflösung (durch PROJ-12, PROJ-14, PROJ-46, PROJ-47):**
- **PROJ-12** liefert eine vollständige SEPA-Basislastschrift-PDF mit Creditor-ID, EEG-Adresse, Member-Adresse, IBAN-Eingabefeld — generiert beim Submit, als E-Mail-Anhang versendet.
- **PROJ-14** ergänzt die Firmenlastschrift-PDF-Variante (B2B) für Unternehmen/Vereine; per EEG via `useCompanySEPAMandate` aktivierbar.
- Per EEG steuerbar via `SEPAMandateEnabled`: TRUE = PDF wird generiert und versendet, FALSE = inline-Checkbox im Form reicht (für EEGs ohne formales SEPA-Erfordernis).
- **PROJ-46 Stage B** verschiebt die Beitrittsbestätigungs-PDF an den Import-Zeitpunkt, damit die später vergebene Mitgliedsnummer einbettbar ist.
- **PROJ-47** schließt die B2B-Lücke: beim Import wird die Firmenlastschrift-PDF erneut generiert, diesmal mit ausgefüllter Mandatsreferenz = Mitgliedsnummer, und an die Member-Mail (+ EEG-Kopie) angehängt. Der Member kann sie ausdrucken und seiner Hausbank vorlegen.
- Audit-Trail über `sepa_mandate_accepted` + `sepa_mandate_accepted_at` (Zustimmung) und `mandate_reference` + `mandate_date` (Mandats-Verwaltung).

Mandatsreferenz bei der Submission-Zeit-PDF bleibt absichtlich Platzhalter („wird von EEG ausgefüllt") — die Mitgliedsnummer existiert dort noch nicht; die finale Variante mit ausgefüllter Referenz kommt mit der Import-Zeit-Mail.

---

## Nicht-Architektur-Fragen

Spezifische Tagesfragen werden direkt im jeweiligen Feature-Spec gelöst (`features/PROJ-NN-*.md`). Dieses File trägt nur die Themen, deren Lösung mehrere Features oder die Gesamtarchitektur betrifft.
