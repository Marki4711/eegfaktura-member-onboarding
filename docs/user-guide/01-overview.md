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
draft → submitted → under_review → approved → imported
                         ↕             ↑          │
                    needs_info         │     (Import zurücksetzen)
                         ↕             │          │
                      rejected         └──────────┘
                                       │
                                  import_failed
```

* `import_failed → approved`: nach Fehlerbehebung kann der Import erneut versucht werden.
* `imported → approved`: über die Aktion **Import zurücksetzen** in der Detailansicht.

| Status | Bedeutung |
|--------|-----------|
| `draft` | Vom Mitglied begonnen, noch nicht eingereicht |
| `submitted` | Vom Mitglied eingereicht, wartet auf Prüfung |
| `under_review` | EEG-Betreiber prüft den Antrag |
| `needs_info` | EEG-Betreiber hat Rückfragen gestellt |
| `approved` | Antrag genehmigt, bereit für Import |
| `rejected` | Antrag abgelehnt |
| `imported` | Erfolgreich in eegFaktura importiert |
| `import_failed` | Import fehlgeschlagen, kann wiederholt werden |
