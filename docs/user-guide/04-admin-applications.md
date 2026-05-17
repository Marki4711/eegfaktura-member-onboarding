# Anträge verwalten

## Antragsübersicht

Nach der Anmeldung sehen Sie die **Antragsübersicht** mit allen eingereichten Anträgen Ihrer EEG(s).

![Antragsübersicht](images/admin-applications-list.png)

### Toolbar-Aktionen

Rechts oben in der Übersicht stehen Ihnen zwei Aktionen zur Verfügung:

- **„Aktivierung im Core prüfen"** *(PROJ-46 Stage D)* — fragt für alle Anträge im Status „Bereit zur Aktivierung" beim eegFaktura-Core nach, ob das Mitglied dort bereits als ACTIVE eingetragen ist. Falls ja, wird der Antrag automatisch auf **„Aktiviert"** gesetzt. Toast zeigt das Ergebnis (z. B. „3 von 5 auf Aktiviert gesetzt") und die Liste wird neu geladen.
- **„Alle Entwürfe löschen"** — erscheint nur, wenn Entwürfe existieren; löscht unwiderruflich alle nicht eingereichten Anträge Ihrer EEG(s).

Die Tabelle zeigt:
- **Referenznummer** — eindeutige Kennung des Antrags
- **Name** — Mitgliedsname oder Firmenname
- **E-Mail** — Kontaktadresse des Mitglieds
- **EEG** — RC-Nummer der zugehörigen EEG
- **Status** — aktueller Bearbeitungsstand
- **Eingereicht am** — Datum und Uhrzeit der Einreichung (Anzeige in Europe/Vienna)

Die Mitgliedsnummer ist nicht in der Liste, sondern erst in der Detailansicht eines Antrags sichtbar (sie wird erst beim Import vergeben und kann alphanumerisch sein, z. B. `A005`).

## Anträge filtern

Über das **Filterpanel** können Sie die Anträge gezielt einschränken:

| Filter | Beschreibung |
|--------|-------------|
| **Status** | Nur Anträge mit einem bestimmten Status anzeigen (Entwurf, Eingereicht, E-Mail bestätigt, In Prüfung, Info benötigt, Genehmigt, Abgelehnt, Importiert, Import fehlgeschlagen, **Warte auf Bank-Bestätigung**, **Bereit zur Aktivierung**, **Aktiviert**) |
| **Name** | Teilsuche über Vorname, Nachname und Firmenname (z. B. findet „Must" sowohl „Max Mustermann" als auch eine Firma „Musterbetrieb GmbH") |
| **E-Mail** | Teilsuche in der E-Mail-Adresse |
| **EEG** | Nur Anträge einer bestimmten EEG anzeigen (erscheint nur bei Admins mit mehreren EEGs) |
| **Eingereicht ab / bis** | Zeitraum der Einreichung (Datumsbereich) |

## Sortieren

Klicken Sie auf eine Spaltenüberschrift, um die Liste nach dieser Spalte zu sortieren:

* Erster Klick → aufsteigend (Pfeil ↑)
* Zweiter Klick → absteigend (Pfeil ↓)
* Dritter Klick → Standardsortierung (Pfeil ↕)

Die aktuelle Sortierung wird im Link in der Adressleiste mitgeführt, sodass Sie sortierte Ansichten teilen oder als Lesezeichen speichern können.

## Antrag öffnen

Klicken Sie auf eine Zeile in der Tabelle, um die **Detailansicht** des Antrags zu öffnen.

## Detailansicht

![Antragsdetail oben](images/admin-application-detail-1.png)

![Antragsdetail unten](images/admin-application-detail-2.png)

Die Detailansicht zeigt alle Angaben des Mitglieds:

- **Statusaktionen** — verfügbare Aktionen je nach aktuellem Status
- **Mitgliedsdaten** — Mitgliedstyp, Name, Geburtsdatum, Kontakt, Adresse
- **Bankverbindung** — IBAN, Kontoinhaber, SEPA-Mandat
- **Einwilligungen** — Datenschutz und Richtigkeitsbestätigung
- **Antragsdaten** — Referenznummer, RC-Nummer, Mitgliedsnummer (nach erfolgreichem Import), Zeitstempel
- **Zählpunkte** — alle angegebenen Zählpunkte mit Richtung und Teilnahmefaktor
- **Admin-Notiz** — interne Notizen (nur für Admins sichtbar)
- **Statusverlauf** — chronologische Historie aller Statusänderungen

## Antrag bearbeiten

Als Admin können Sie folgende Felder direkt korrigieren:

- Persönliche Daten und Adresse
- IBAN und Kontoinhaber
- Zählpunkte
- Admin-Notiz (interne Anmerkungen)

Klicken Sie auf **Bearbeiten**, nehmen Sie die Änderungen vor und speichern Sie.

> **Hinweis:** Änderungen an Antragsdaten werden im Statusverlauf nicht automatisch protokolliert. Nutzen Sie die Admin-Notiz für wichtige Vermerke.

> **Hinweis:** Das Speichern der Admin-Notiz aktualisiert ausschließlich das Notizfeld — andere Antragsdaten (Mitgliedstyp, Zählpunkte, Teilnahmefaktor, …) werden dabei nicht überschrieben.

## Entwürfe löschen

Wenn ein Mitglied einen Antrag begonnen, aber nie eingereicht hat (Status `draft`), können Sie ihn aus der Übersicht entfernen. Die Massen-Löschaktion respektiert dabei den aktiven **EEG-Filter**:

* Filter auf eine bestimmte EEG gesetzt → nur Entwürfe dieser EEG werden gelöscht
* Kein EEG-Filter gesetzt (Superuser) → Entwürfe aller EEGs werden gelöscht

## E-Mail erneut senden

Falls ein Mitglied die Bestätigungs-E-Mail nicht erhalten hat, können Sie diese über den Button **E-Mail erneut senden** in der Detailansicht nochmals versenden.

## EEG umzuordnen (PROJ-40)

Falls ein Antrag fälschlich der falschen EEG zugeordnet wurde (Mitglied hat den falschen RC-Link verwendet), kann er als Admin direkt umzuordnen werden — ohne dass das Mitglied neu einreichen muss. Verfügbar nur für Admins mit Zugriff auf mehrere EEGs und nur solange der Antrag in der Review-Phase ist (`submitted` / `email_confirmed` / `under_review` / `needs_info`). Detail-Beschreibung siehe [Statusverwaltung → EEG umzuordnen](05-admin-status.md#eeg-umzuordnen-proj-40).
