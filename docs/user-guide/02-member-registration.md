# Mitglieder-Registrierung (öffentliches Formular)

Diese Anleitung richtet sich an **neue EEG-Mitglieder**, die sich über das Online-Formular anmelden möchten.

## Schritt 1: Registrierungslink öffnen

Jede EEG hat einen eigenen Registrierungslink in der Form:

```
https://<ihre-eeg-domain>/register/RC123456
```

Diesen Link erhalten Sie von Ihrem EEG-Betreiber (z.B. per E-Mail oder auf der Website der EEG).

![Startseite Registrierungsformular](images/register-form-start.png)

## Schritt 2: Mitgliedstyp auswählen

Wählen Sie den zutreffenden Mitgliedstyp:

| Typ | Beschreibung |
|-----|-------------|
| **Privatperson** | Natürliche Person |
| **Landwirt** | Land- und forstwirtschaftlicher Betrieb |
| **Gemeinde** | Öffentliche Körperschaft |
| **Unternehmen** | Juristische Person / GmbH, AG, etc. |
| **Verein** | Eingetragener Verein |

Je nach Mitgliedstyp werden unterschiedliche Felder angezeigt (z.B. Firmenname statt Vorname/Nachname).

## Schritt 3: Persönliche Daten eingeben

Füllen Sie alle Pflichtfelder aus (mit * markiert):

- **Vorname / Nachname** (bei Privatpersonen und Landwirten)
- **Firmenname** (bei Unternehmen, Gemeinden, Vereinen)
- **E-Mail-Adresse** — hieran erhalten Sie die Einreichungsbestätigung
- **Telefon** (optional, sofern von Ihrer EEG aktiviert)
- **Wohnadresse** (Straße, Hausnummer, PLZ, Ort)

## Schritt 4: Bankverbindung eingeben

Geben Sie Ihre IBAN und den Kontoinhaber an. Mit dem Setzen des Häkchens bei **SEPA-Lastschriftmandat** erteilen Sie der EEG die Erlaubnis, Beiträge einzuziehen.

> **Hinweis:** IBANs aus allen SEPA-Ländern werden akzeptiert (AT, DE, CH, etc.).

## Schritt 5: Zählpunkte angeben

![Zählpunkt-Eingabe](images/register-form-metering-points.png)

Geben Sie mindestens einen Zählpunkt an:

- **Zählpunktnummer** — 33-stellige Nummer im Format `AT...` (steht auf Ihrer Stromrechnung)
- **Richtung** — Verbraucher (Strom wird bezogen) oder Erzeuger (Strom wird eingespeist)
- **Teilnahmefaktor** — prozentualer Anteil der Teilnahme an der EEG (Standard: 100 %)

Über **Zählpunkt hinzufügen** können Sie bis zu 10 Zählpunkte angeben.

## Schritt 5a: Genossenschaftsanteile (nur bei aktivierten EEGs)

Wenn Ihre EEG als Genossenschaft organisiert ist und in den Einstellungen die Anteils-Erfassung aktiviert hat, erscheint ein zusätzlicher Block **„Genossenschaftsanteile"** im Formular:

- **Pflichtanteil je Standort** — der von der EEG festgelegte Mindestwert (z.B. „1 Anteil"). Reiner Hinweistext, kann nicht geändert werden.
- **Anzahl Anteile gesamt** — Eingabefeld, vorbefüllt mit dem Pflichtwert. Sie können den Wert nach oben überschreiben (mehr Anteile freiwillig zeichnen), aber nicht darunter.
- **Genossenschaftsanteilswert** und **Gesamtbetrag** werden live berechnet und unterhalb angezeigt (z.B. „€ 100,00 × 3 = € 300,00").

Wenn Ihre EEG dieses Feature nicht aktiviert hat, ist der Block ausgeblendet und Sie können diesen Schritt überspringen.

## Schritt 6: Datenschutz und Einreichung

- Stimmen Sie der **Datenschutzerklärung** zu
- Bestätigen Sie die **Richtigkeit Ihrer Angaben**
- Falls Ihre EEG zusätzliche Pflicht-Dokumente hinterlegt hat (z. B. Satzung), bestätigen Sie diese ebenfalls per Häkchen
- Falls Ihre EEG **Info-Dokumente** (PROJ-36) verlinkt hat (z. B. Mitgliederinfo, Hausordnung), werden diese nur zur Kenntnisnahme angezeigt — kein Häkchen, aber das Einreichen des Antrags gilt als Kenntnisnahme
- Klicken Sie auf **Antrag einreichen**

Nach der Einreichung erhalten Sie eine **Bestätigungs-E-Mail** mit Ihrer Antragsnummer (Format `<RC>-<Jahr>-<NNNN>`, z. B. `RC123456-2026-0001`). Die E-Mail enthält zusätzlich:

* eine PDF-Zusammenfassung Ihrer Angaben,
* eine Identifikations-Fußzeile mit Ihrer EEG, damit Sie die Mail eindeutig zuordnen können,
* eine **Reply-To**-Adresse, über die Sie direkt mit Ihrer EEG in Kontakt treten können (Antworten gehen nicht an einen „noreply"-Postfach).

## Schritt 7: E-Mail-Adresse bestätigen (nur bei aktivierten EEGs)

Wenn Ihre EEG das Feature **„E-Mail-Bestätigung erforderlich"** (PROJ-31) aktiviert hat, enthält Ihre Bestätigungs-Mail zusätzlich einen gelben Hinweisblock mit einem Button **„E-Mail-Adresse bestätigen"**. Der Link ist 30 Tage gültig. Erst nach dem Klick wird Ihr Antrag von der EEG bearbeitet.

Ist das Feature in Ihrer EEG deaktiviert, entfällt dieser Schritt — der Antrag geht direkt in die Prüfung.

## Was passiert nach der Einreichung?

Ihr Antrag wird nun vom EEG-Betreiber geprüft. Mögliche nächste Schritte:

- **Rückfragen:** Der EEG-Betreiber kann Sie um Ergänzungen bitten. Sie erhalten eine E-Mail mit den Rückfragen und können Ihren Antrag ergänzen.
- **Genehmigung:** Ihr Antrag wird genehmigt und in eegFaktura importiert.
- **Ablehnung:** In Ausnahmefällen kann ein Antrag abgelehnt werden.
