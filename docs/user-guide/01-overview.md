# eegFaktura Mitglieder-Onboarding — Überblick

## Was ist das Tool?

Das eegFaktura Mitglieder-Onboarding ermöglicht die **selbstständige Online-Registrierung neuer EEG-Mitglieder**. Neue Mitglieder füllen ein öffentlich zugängliches Webformular aus, das über einen individuellen Link der jeweiligen EEG erreichbar ist. Die Daten werden zunächst in einem Prüfbereich gesammelt und erst nach Freigabe durch den EEG-Betreiber in eegFaktura übernommen.

## Wie funktioniert der Prozess?

```
Mitglied                    EEG-Betreiber              eegFaktura
   |                              |                         |
   |-- Formular ausfüllen ------->|                         |
   |                              |                         |
   |<-- Bestätigung per E-Mail ---|                         |
   |                              |                         |
   |                    Antrag prüfen                       |
   |                    Rückfragen stellen (optional)       |
   |                    Antrag genehmigen                   |
   |                              |                         |
   |                              |-- Import starten ------>|
   |                              |   (inkl. Mitgliedsnr.)  |
   |                              |                         |
```

> **Hinweis:** Die **Mitgliedsnummer** wird nicht beim Einreichen, sondern erst beim Import in eegFaktura vergeben. Das eegFaktura-Core schlägt die nächste freie Nummer vor (numerisch oder alphanumerisch, z. B. `A006`), die der EEG-Betreiber im Import-Dialog übernehmen oder anpassen kann.

## Benutzerrollen

| Rolle | Zugang | Berechtigungen |
|-------|--------|----------------|
| **Mitglied** | Öffentlicher Registrierungslink | Antrag einreichen, Rückfragen beantworten |
| **EEG-Betreiber** | Admin-Oberfläche (Keycloak-Login) | Anträge prüfen, Status ändern, in eegFaktura importieren |

## Antragsstatus im Überblick

```
draft → submitted ─┬─ (Standard) ──→ under_review → approved → imported (Millisek.)
                   │                       ↕            ↑          │
                   │                  needs_info        │   ┌──────┴──────┐
                   │                       ↕            │   ▼             ▼
                   │                    rejected        │  b2b?         non-b2b
                   │                                    │   │             │
                   │                          import_failed │             │
                   │                                        ▼             │
                   │                          awaiting_bank_confirmation  │
                   │                                        ↕             ↓
                   │                                ready_for_activation ←┘
                   │                                        ↓
                   │                                    activated (Endzustand)
                   │
                   └─ (EEG mit E-Mail-Bestätigung) ─→ email_confirmed → under_review → …
```

* `submitted → email_confirmed`: nur wenn die EEG **E-Mail-Bestätigung erforderlich** aktiviert hat — Mitglied klickt den Bestätigungs-Link in der Willkommens-Mail.
* `import_failed → approved`: nach Fehlerbehebung kann der Import erneut versucht werden.
* `imported` ist **transient** (nur Millisekunden) — der Server transitioniert sofort weiter abhängig von `einzugsart=b2b`:
  - **b2b:** `imported → awaiting_bank_confirmation` (Admin wartet auf Mitglied-Rückmeldung zur Hausbank-Pre-Notification, dann manuell weiter)
  - **sonst:** `imported → ready_for_activation` (direkt nach Aktivierung im Core bereit)
* `ready_for_activation → activated`: Admin klickt manuell „Als aktiv markieren" ODER nutzt den Batch-Button „Aktivierung im Core prüfen" in der Antragsliste.
* `activated` ist **strikter Endzustand**: keine Übergänge raus, kein Reset. Deaktivierung erfolgt direkt im Core.
* `imported / awaiting_bank_confirmation / ready_for_activation → approved`: über die Aktion **Import zurücksetzen** in der Detailansicht. NICHT aus `activated`.

| Status | Bedeutung |
|--------|-----------|
| `draft` | Vom Mitglied begonnen, noch nicht eingereicht |
| `submitted` | Vom Mitglied eingereicht, wartet auf Prüfung (oder auf E-Mail-Bestätigung, wenn aktiviert) |
| `email_confirmed` | Mitglied hat den Bestätigungs-Link geklickt; Antrag wartet auf EEG-Prüfung |
| `under_review` | EEG-Betreiber prüft den Antrag |
| `needs_info` | EEG-Betreiber hat Rückfragen gestellt |
| `approved` | Antrag genehmigt, bereit für Import |
| `rejected` | Antrag abgelehnt |
| `imported` | Erfolgreich in eegFaktura importiert (transient, Auto-Routing direkt danach) |
| `import_failed` | Import fehlgeschlagen, kann wiederholt werden |
| `awaiting_bank_confirmation` | (B2B-SEPA) Mitglied muss sein B2B-Mandat bei der Hausbank hinterlegen, EEG-Admin wartet auf Bestätigung |
| `ready_for_activation` | Bereit zur finalen Aktivierung in der EEG |
| `activated` | Aktives Mitglied — Endzustand |
