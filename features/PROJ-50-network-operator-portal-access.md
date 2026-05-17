# PROJ-50: Frage „Zugang Online-Portal Netzbetreiber vorhanden?" + bedingte Anleitungs-Mail

**Status:** Planned
**Created:** 2026-05-17
**Last Updated:** 2026-05-17

## Hintergrund

Praxis-Feedback einer EEG (2026-05-17): Beim Onboarding möchte die EEG erheben, ob das Mitglied bereits einen Zugang zum **Online-Portal des Netzbetreibers** hat (Achtung — explizit nicht das Portal des Energieversorgungsunternehmens). Mitglieder ohne Zugang sollen automatisch eine Mail mit einer **EEG-spezifischen Anleitung** zum Anlegen dieses Zugangs erhalten.

## Scope (Erstentwurf)

### Datenmodell

Neue boolean-Spalte auf `application` (Application-Scope, nicht pro Zählpunkt — der Portal-Zugang ist member-bezogen, nicht zählpunktspezifisch):

- `network_operator_portal_access` BOOLEAN NULL — Antwort des Mitglieds: `TRUE` = Zugang vorhanden, `FALSE` = nicht vorhanden, `NULL` = Mitglied hat die Frage nicht beantwortet (z. B. EEG hat das Feld nicht aktiviert).

Optional (siehe offene Fragen): begleitendes Audit-Feld `network_operator_portal_access_at` mit Zeitstempel der ersten Antwort.

### Sichtbarkeit + Konfigurierbarkeit

- PROJ-8-konfigurierbar via `field_config.network_operator_portal_access` mit Default `hidden`.
- Wenn `optional` oder `required` → erscheint im Mitglieds-Formular als Ja/Nein-Auswahl. Bei `required` muss eine der beiden Antworten gesetzt sein (kein null).
- Application-Scope (im allgemeinen Abschnitt „Weitere Angaben"), nicht pro Zählpunkt.

### Bedingte Mail an Mitglied

- Bei Antwort `FALSE` (kein Portal-Zugang) wird beim Submit automatisch eine **zusätzliche Mail** an das Mitglied gesendet — mit einer **per EEG konfigurierbaren Anleitung** (URL oder freier Text/HTML).
- Konfiguration in den EEG-Admin-Einstellungen, vergleichbar mit `intro_text`:
  - `network_operator_portal_guide_url` TEXT NULL — Link zur Anleitungs-Seite ODER
  - `network_operator_portal_guide_text` TEXT NULL — sanitisierter HTML-Body, der direkt in die Mail eingebettet wird.
- Wenn keine der beiden EEG-Optionen gesetzt ist → Frage bleibt anbietbar, aber **keine** zusätzliche Mail wird verschickt (sinnvoller Fallback, keine leere Mail).

## Offene Fragen / Diskussion

**Konflikt-Frage des Owners (2026-05-17):**
> „Sind wir nicht im Konflikt mit dem E-Mail, das aus eegFaktura kommt? Bei der Aktivierung kommt doch schon eine Mail mit Link auf die Seite mit Anleitungen."

**Rückmeldung Praxis-EEG:**
> „Das taugt uns nicht, weil nicht administrierbar — wir umgehen das im Workaround, indem wir unsere eigene Mail verwenden und erst später die des MG einsetzen."

**Implikation:** Die Core-Mail aus eegFaktura existiert, ist aber **nicht per EEG administrierbar** (zentraler Text). EEGs in der Praxis schicken bereits eigene Anleitungs-Mails außerhalb des Onboardings — dieses Feature würde das in den offiziellen Workflow heben und automatisieren.

**Noch zu klären vor Umsetzung:**
1. **Mail-Konfiguration:** URL-Verweis (kurz, immer aktuell) vs. inline-HTML (vollständige Kontrolle, aber Versions-Konflikt-Risiko)? Oder beides parallel?
2. **Versand-Modus:** Best-effort async (wie Submit-Mail) oder hard-fail (wie Status-Change-Mails PROJ-41/43)?
3. **Trigger-Zeitpunkt:** Genau beim Submit, beim Approve, oder am Übergang nach `ready_for_activation` (PROJ-46)? Aktivierungs-Mail aus Core kommt ja erst nach Import — frühere Zustellung könnte die EEG-eigene Anleitung „zu früh" wirken lassen, spätere Zustellung kollidiert mit Core-Mail.
4. **Abgrenzung zu PROJ-44 (Netzbetreiber-Vollmacht):** beide Felder betreffen den Netzbetreiber. Spec-Frage: Gehört das in dieselbe UI-Sektion „Netzbetreiber" mit Sammel-Label?
5. **Audit-Timestamp:** muss `*_at` mitgepflegt werden (für späteren Reset/Re-Submit-Pfad)?

## Acceptance Criteria (Skizze, vor Umsetzung verfeinern)

1. Migration legt `application.network_operator_portal_access` an + die zwei EEG-Settings-Felder.
2. Public-Form rendert die Frage je nach EEG-Konfiguration; required-Validierung greift bei `required`-State.
3. Admin-Settings-UI zeigt die EEG-Felder „Anleitungs-URL" + „Anleitungs-Text" mit klarer Erklärung des Verhaltens.
4. Beim Submit mit Antwort `FALSE`: zusätzliche Mail wird verschickt, wenn EEG eine Anleitung konfiguriert hat.
5. Antrag-PDF + Beitrittsbestätigungs-PDF zeigen die Antwort als eigene Zeile.
6. Admin-Detail-View zeigt die Antwort + ggf. „Anleitungs-Mail versendet am" Timestamp.

## Out of Scope

- Tracking, ob das Mitglied den Anleitungs-Link tatsächlich geöffnet hat (kein Pixel-Tracking, kein UTM).
- Eigener Versand-Mechanismus pro EEG (deckt PROJ-26 ab).
- Änderungen am eegFaktura-Core-Mail-System.

## Hinweis

Dieses Feature ergänzt **PROJ-44** (Netzbetreiber-Vollmacht) thematisch, ist aber inhaltlich unabhängig: PROJ-44 ist die rechtliche Bevollmächtigung, PROJ-50 nur die Status-Abfrage „Portal-Zugang vorhanden?" + automatisierte Hilfestellung.
