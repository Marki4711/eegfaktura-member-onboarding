# Statusverwaltung

## Statusübergänge

Der Status eines Antrags steuert den Bearbeitungsablauf. Folgende Übergänge sind möglich:

```
submitted ──→ under_review ──→ approved ──→ imported
                   │                 │           │
                   │                 └──→ import_failed
                   │                              │
                   │              ┌───────────────┘
                   │              ↓
                   │           approved  (Import zurücksetzen)
                   │
                   ├──→ needs_info
                   │       └──→ submitted (nach Ergänzung durch Mitglied)
                   └──→ rejected
```

* `import_failed → approved`: nach Fehlerbehebung kann der Import erneut versucht werden.
* `imported → approved`: über die Aktion **Import zurücksetzen** (siehe unten) — z. B. wenn der Teilnehmer im eegFaktura-Core manuell gelöscht und neu importiert werden soll.

## Status ändern

In der Detailansicht eines Antrags finden Sie den Bereich **Status-Aktionen**.

![Status-Aktionen](images/admin-status-actions.png)

Klicken Sie auf die gewünschte Aktion. Je nach aktuellem Status stehen unterschiedliche Aktionen zur Verfügung:

| Aktueller Status | Mögliche Aktionen |
|-----------------|-------------------|
| `submitted` | In Prüfung nehmen, Ablehnen *(bei aktiver E-Mail-Bestätigung nur „Ablehnen", bis das Mitglied bestätigt)* |
| `email_confirmed` | In Prüfung nehmen, Rückfragen stellen, Ablehnen |
| `under_review` | Genehmigen, Rückfragen stellen, Ablehnen |
| `needs_info` | — (wartet auf Ergänzung durch das Mitglied) |
| `approved` | Import starten |
| `import_failed` | Import erneut starten |
| `imported` | Import zurücksetzen |

Zusätzlich verfügbar in allen Review-Stati (`submitted` / `email_confirmed` / `under_review` / `needs_info`) für Admins mit Zugriff auf ≥ 2 EEGs:

| Aktion | Wirkung |
|--------|---------|
| **EEG umzuordnen** | Verschiebt den Antrag in eine andere EEG (siehe Abschnitt unten, PROJ-40) |

## E-Mail-Bestätigung (`email_confirmed`)

Wenn in den EEG-Einstellungen **„E-Mail-Adresse bestätigen"** aktiviert ist, erscheinen neue Anträge zunächst im Status `submitted` mit dem Hinweis **„E-Mail-Adresse noch nicht bestätigt"**. Solange der Bewerber den Link in der Bestätigungs-Mail nicht angeklickt hat, ist der einzig verfügbare Status-Schritt **„Ablehnen"** (für offensichtlichen Spam).

Sobald der Bewerber klickt:

- Status wechselt automatisch auf `email_confirmed`
- Sie erhalten die EEG-Benachrichtigungs-Mail mit den Antragsdaten
- Alle normalen Status-Aktionen (In Prüfung nehmen, Rückfragen, Genehmigen, Ablehnen) sind ab jetzt verfügbar

**Bestätigungs-Link erneut senden**: Sollte das Mitglied den Link nicht finden (z. B. Spam-Ordner), nutzen Sie in der Detail-Seite oben rechts **„Bestätigungs-Link erneut senden"**. Das generiert ein neues Token; der alte Link wird ungültig. Min. 5 Minuten Wartezeit zwischen zwei Sendungen.

**Automatische Ablehnung**: Anträge, deren Bestätigung 30 Tage lang ausbleibt, werden vom System automatisch auf `rejected` gesetzt mit dem Grund „E-Mail-Bestätigung ausgeblieben (Auto-Reject nach 30 Tagen)".

## In Prüfung nehmen (`under_review`)

Nehmen Sie einen eingereichten Antrag in Prüfung, um anzuzeigen, dass Sie ihn aktiv bearbeiten. Dies ist optional, hilft aber wenn mehrere Admins auf dieselbe EEG arbeiten.

## Rückfragen stellen (`needs_info`)

Wenn Angaben fehlen oder unklar sind:

1. Klicken Sie auf **Rückfragen stellen**
2. Geben Sie den Grund / die Rückfrage ein — der blaue Hinweis im Dialog erinnert daran: **„Der hier eingegebene Text wird per E-Mail an den Beitrittswerber übermittelt"**
3. Das Mitglied erhält eine E-Mail mit Ihrer Rückfrage 1:1 im Body und kann seinen Antrag ergänzen
4. **Hard-Fail (PROJ-43, ab 2026-05-17):** scheitert der SMTP-Versand, wird der Statuswechsel zurückgerollt und Sie sehen die Fehlermeldung direkt im Dialog. Status bleibt unverändert, Sie können nach SMTP-Recovery erneut klicken

Nach der Ergänzung durch das Mitglied wechselt der Status automatisch zurück auf `submitted`.

## Genehmigen (`approved`)

Wenn alle Angaben korrekt und vollständig sind:

1. Klicken Sie auf **Genehmigen**
2. Der Antrag wechselt auf `approved` und ist bereit für den Import in eegFaktura

## Ablehnen (`rejected`)

Wenn ein Antrag nicht genehmigt werden kann:

1. Klicken Sie auf **Ablehnen**
2. Geben Sie einen Ablehnungsgrund an — der blaue Hinweis im Dialog erinnert daran: **„Die hier eingegebene Begründung wird per E-Mail an den Beitrittswerber übermittelt"**
3. Das Mitglied erhält eine E-Mail mit Ihrer Begründung 1:1 im Body (PROJ-41)
4. **Hard-Fail:** gleiches Verhalten wie bei Rückfragen — Mail-Fehler rollt den Statuswechsel zurück

## Import in eegFaktura

Nach der Genehmigung kann der Antrag in eegFaktura importiert werden:

1. Öffnen Sie den genehmigten Antrag
2. Klicken Sie auf **In eegFaktura importieren**
3. Es öffnet sich der Dialog **Import-Konfiguration**:
   * **Tarif** — Auswahl aus den im eegFaktura-Core hinterlegten Tarifen der EEG
   * **Mitgliedsnummer** — Vorbelegt mit der nächsten freien Nummer (basierend auf dem dominanten Muster in eegFaktura, z. B. `A005 → A006` oder `12 → 13`). Maximal 50 Zeichen, alphanumerisch erlaubt. Sie können den Vorschlag übernehmen oder überschreiben.
4. Klicken Sie auf **Importieren**
5. Bei Erfolg wechselt der Status auf `imported`
6. Bei Fehler wechselt der Status auf `import_failed` — der Import kann wiederholt werden (Mitgliedsnummer und Tarif werden erneut abgefragt)

![Import-Aktion](images/admin-import-action.png)

> **Hinweis:** Die Mitgliedsnummer wird erst beim Import vergeben. Im Status `submitted` / `under_review` / `approved` ist sie noch leer — das ist beabsichtigt, weil nur eegFaktura die endgültige Nummern­vergabe steuert.

> **Hinweis:** Der Import kann bei technischen Problemen mit eegFaktura fehlschlagen. In diesem Fall prüfen Sie den Fehlerhinweis und wiederholen Sie den Import, sobald das Problem behoben ist.

## Hängengebliebener Import (PROJ-34)

Wenn ein Import-Versuch nicht sauber abschließt — z.B. weil eine Datenbank-Eindeutigkeitsverletzung den Bookkeeping-Schritt nach dem Core-Aufruf scheitern lässt, oder weil das Onboarding-Backend mitten im Import abstürzt — bleibt der Antrag im Status `approved`, kann aber nicht erneut importiert werden (das System meldet 409 „Import läuft bereits").

In diesem Fall erscheint im Antrags-Detail ein **oranger Banner** mit zwei Recovery-Aktionen:

- **„Als importiert markieren"** — wenn der Teilnehmer im Core trotzdem angelegt wurde (im eegFaktura-Core nachschauen, Teilnehmer-UUID und vergebene Mitgliedsnummer notieren). Tragen Sie beides im Dialog ein; der Antrag wechselt sauber auf `imported`.
- **„Import-Lock räumen (Retry)"** — wenn Sie sicher sind, dass im Core kein Teilnehmer angelegt wurde (oder Sie ihn vorher manuell gelöscht haben). Setzt den Lock zurück, der Antrag bleibt auf `approved`, ein erneuter Import wird möglich. **Achtung**: Bei vorhandenem Core-Teilnehmer entsteht beim Retry ein Duplikat.

Der Banner erscheint automatisch, sobald der Import-Versuch älter als 2 Minuten ist und nicht sauber abgeschlossen wurde — Sie müssen nicht raten, ob „nochmal probieren" sicher ist.

## Import zurücksetzen (`imported → approved`)

Wenn ein bereits importierter Teilnehmer im eegFaktura-Core gelöscht wurde (z. B. weil das Mitglied seine Teilnahme widerrufen hat oder der Import fehlerhaft war), kann der Antrag in den Status `approved` zurückgesetzt werden, um einen Neu-Import zu ermöglichen.

1. Öffnen Sie den importierten Antrag
2. Klicken Sie auf **Import zurücksetzen**
3. Geben Sie eine Begründung an (Pflichtfeld, wird im Statusverlauf protokolliert)
4. Der Antrag wechselt auf `approved`; die alte `target_participant_id` wird im Statusverlauf archiviert

> **Wichtig:** Diese Aktion kontaktiert den eegFaktura-Core *nicht*. Bevor Sie sie nutzen, müssen Sie den Teilnehmer im Core manuell gelöscht haben.

## EEG umzuordnen (PROJ-40)

Wenn ein Mitglied über den falschen RC-Link der EEG A registriert hat, aber eigentlich zur EEG B gehört (z. B. weil das Versorgungsgebiet woanders hingehört), kann der Antrag direkt umzuordnen werden — ohne dass das Mitglied neu einreichen muss.

**Verfügbarkeit**: der Button **„EEG umzuordnen"** erscheint im Status-Aktionen-Block nur, wenn:
- der Status `submitted`, `email_confirmed`, `under_review` oder `needs_info` ist (nicht bei `approved` / `imported` / `rejected`)
- Sie als Admin Zugriff auf mindestens 2 EEGs haben

**Ablauf**:

1. Klicken Sie auf **EEG umzuordnen**
2. Wählen Sie die Ziel-EEG aus dem Dropdown (Ihre Tenants außer der aktuellen)
3. Geben Sie eine Begründung an (Pflichtfeld, mindestens 5 Zeichen)
4. Bestätigen mit **Umzuordnen**

**Was passiert**:
- Eine neue **Referenznummer** wird vom per-EEG-Counter der Ziel-EEG vergeben (Format `<neue-RC>-<Jahr>-<NNNN>`)
- Die alte Referenznummer und alte RC werden im Statusverlauf als `[system] previous rc_number=…` und `[system] previous reference_number=…` archiviert
- Der Status bleibt **unverändert** (kein Status-Wechsel)
- Der Antrag erscheint ab sofort in der Liste der Ziel-EEG, nicht mehr unter der alten EEG
- Es wird **keine E-Mail** an das Mitglied verschickt — die nächste Status-Mail (Rückfrage / Ablehnung / Zustimmung) kommt automatisch von der neuen EEG

**Wichtig**: Cooperative-Shares, Konfigurierbare-Felder und die E-Mail-Bestätigungs-Pflicht werden **nicht** re-validiert. Wenn die Ziel-EEG andere Pflichtfelder hat, nutzen Sie nach dem Umzuordnen ggf. **Rückfragen stellen**, um fehlende Daten nachzufordern.

## Statusverlauf

Jede Statusänderung wird automatisch im **Statusverlauf** protokolliert:

- Zeitpunkt der Änderung (Anzeige in Europe/Vienna mit CET/CEST-Umstellung)
- Von-Status und An-Status
- Benutzer der die Änderung vorgenommen hat
- Optionaler Grund (bei Rückfragen, Ablehnung und Import-Reset)

![Statusverlauf](images/admin-status-log.png)
