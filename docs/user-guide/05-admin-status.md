# Statusverwaltung

## Statusübergänge

Der Status eines Antrags steuert den Bearbeitungsablauf. Folgende Übergänge sind möglich:

```
submitted ──→ under_review ──→ approved ──→ imported
                   │                 └──→ import_failed
                   ├──→ needs_info
                   │       └──→ submitted (nach Ergänzung durch Mitglied)
                   └──→ rejected
```

## Status ändern

In der Detailansicht eines Antrags finden Sie den Bereich **Status-Aktionen**.

![Status-Aktionen](images/admin-status-actions.png)

Klicken Sie auf die gewünschte Aktion. Je nach aktuellem Status stehen unterschiedliche Aktionen zur Verfügung:

| Aktueller Status | Mögliche Aktionen |
|-----------------|-------------------|
| `submitted` | In Prüfung nehmen, Rückfragen stellen, Ablehnen |
| `under_review` | Genehmigen, Rückfragen stellen, Ablehnen |
| `needs_info` | — (wartet auf Ergänzung durch das Mitglied) |
| `approved` | Import starten |
| `import_failed` | Import erneut starten |

## In Prüfung nehmen (`under_review`)

Nehmen Sie einen eingereichten Antrag in Prüfung, um anzuzeigen, dass Sie ihn aktiv bearbeiten. Dies ist optional, hilft aber wenn mehrere Admins auf dieselbe EEG arbeiten.

## Rückfragen stellen (`needs_info`)

Wenn Angaben fehlen oder unklar sind:

1. Klicken Sie auf **Rückfragen stellen**
2. Geben Sie den Grund / die Rückfrage ein
3. Das Mitglied erhält eine E-Mail und kann seinen Antrag ergänzen

Nach der Ergänzung durch das Mitglied wechselt der Status automatisch zurück auf `submitted`.

## Genehmigen (`approved`)

Wenn alle Angaben korrekt und vollständig sind:

1. Klicken Sie auf **Genehmigen**
2. Der Antrag wechselt auf `approved` und ist bereit für den Import in eegFaktura

## Ablehnen (`rejected`)

Wenn ein Antrag nicht genehmigt werden kann:

1. Klicken Sie auf **Ablehnen**
2. Geben Sie einen Ablehnungsgrund an (wird intern gespeichert)

## Import in eegFaktura

Nach der Genehmigung kann der Antrag in eegFaktura importiert werden:

1. Öffnen Sie den genehmigten Antrag
2. Klicken Sie auf **In eegFaktura importieren**
3. Bei Erfolg wechselt der Status auf `imported`
4. Bei Fehler wechselt der Status auf `import_failed` — der Import kann wiederholt werden

![Import-Aktion](images/admin-import-action.png)

> **Hinweis:** Der Import kann bei technischen Problemen mit eegFaktura fehlschlagen. In diesem Fall prüfen Sie den Fehlerhinweis und wiederholen Sie den Import, sobald das Problem behoben ist.

## Statusverlauf

Jede Statusänderung wird automatisch im **Statusverlauf** protokolliert:

- Zeitpunkt der Änderung
- Von-Status und An-Status
- Benutzer der die Änderung vorgenommen hat
- Optionaler Grund (bei Rückfragen und Ablehnung)

![Statusverlauf](images/admin-status-log.png)
