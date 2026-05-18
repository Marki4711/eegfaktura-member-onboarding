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

| Typ | Beschreibung | USt.-Hinweis im Dropdown |
|-----|-------------|--------------------------|
| **Privatperson** | Natürliche Person | — |
| **Kleinunternehmer** | Einzelunternehmer mit Kleinunternehmer-Regelung | `(0 % USt.)` |
| **Pauschalierter Landwirt** | Land- und forstwirtschaftlicher Betrieb | `(13 % USt.)` |
| **Gemeinde / öffentliche Körperschaft** | — | `(variabel)` |
| **Unternehmen** | Juristische Person / GmbH, AG, etc. | `(20 % USt.)` |
| **Verein** | Eingetragener Verein | `(variabel)` |

Je nach Mitgliedstyp werden unterschiedliche Felder angezeigt (z. B. Firmenname statt Vorname/Nachname). Der USt.-Hinweis in Klammern dient der Orientierung — er zeigt, welchen Steuersatz Ihre Rechnungen aus der EEG voraussichtlich tragen werden.

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

Geben Sie mindestens einen Zählpunkt an. Pro Zählpunkt-Eintrag erscheinen die Felder in zwei Zeilen — zuerst **Richtung** und **Teilnahmefaktor** in einer Zeile, darunter die volle **Zählpunktnummer**. Das ist Absicht: die Richtung bestimmt die Eingabe-Mask der Zählpunktnummer (siehe unten).

- **Richtung** — Verbraucher (Strom wird bezogen) oder Erzeuger (Strom wird eingespeist)
- **Teilnahmefaktor** — prozentualer Anteil der Teilnahme an der EEG (Standard: 100 %)
- **Zählpunktnummer** — 33-stellige Nummer im Format `AT...` in der offiziellen E-Control-Gruppierung `2-6-5-20` (steht auf Ihrer Stromrechnung). Die letzten 20 Stellen können Großbuchstaben und Ziffern enthalten.
  - **Prefix-Vorbelegung (PROJ-52)**: Wenn Ihre EEG einen Zählpunkt-Prefix für die gewählte Richtung konfiguriert hat, ist dieser bereits eingetragen und kann nicht überschrieben werden — Sie tippen nur die individuellen letzten Stellen.
  - **Auto-Pad**: Wenn Sie das Eingabefeld verlassen und weniger Stellen als nötig eingetippt haben, werden fehlende Stellen automatisch mit führenden Nullen zwischen Prefix und Ihrer Eingabe ergänzt.
  - **Richtungs-Wechsel** löscht das Zählpunkt-Feld, damit der korrekte Prefix für die neue Richtung greifen kann.
- **Erzeugungsform** *(PROJ-45, nur bei Erzeuger-Zählpunkten)* — Auswahl PV / Wasser / Wind / Biomasse, Default PV
- **Batteriespeicher vorhanden** *(PROJ-49 follow-up, nur bei PV-Erzeugern)* — Master-Checkbox: nach dem Aktivieren erscheinen die drei Speicher-Felder gemeinsam:
  - **Größe Batterie (kWh)** *(PROJ-45, sofern die EEG das Feld konfiguriert hat)*
  - **Hersteller Wechselrichter** *(PROJ-45, sofern die EEG das Feld konfiguriert hat)*
  - **Speichersteuerung im Sinne der EEG vorstellbar?** *(PROJ-49 follow-up, sofern die EEG das Feld konfiguriert hat)* — Ja-/Nein-Häkchen: die EEG könnte Ihren Heimspeicher gemeinsam mit anderen Speichern der Mitglieder so steuern, dass die Erzeugung innerhalb der Gemeinschaft optimal genutzt wird. Eine konkrete Steuerung wird separat abgestimmt; das Häkchen ist nur Ihre grundsätzliche Bereitschaft.
- **Verbrauch Vorjahr / Verbrauch Prognose (kWh)** *(PROJ-49, nur bei Verbraucher-Zählpunkten)*
- **Einspeisung Prognose (kWh/Jahr)** *(PROJ-49, nur bei Erzeuger-Zählpunkten)*
- **PV-Leistung (kWp)** *(PROJ-49, nur bei Erzeuger-Zählpunkten mit Erzeugungsform PV)*
- **Einspeiselimit** *(PROJ-49, nur bei PV-Erzeugern)* — Checkbox „Einspeiselimit vorhanden". Bei Ja erscheint ein Eingabefeld für den maximalen Einspeisewert in kW. Hintergrund: manche Netzanschlüsse sind leistungstechnisch beschränkt, sodass nur ein Teil der erzeugten PV-Leistung tatsächlich ins Netz eingespeist werden darf.
- **Abweichende Adresse** *(PROJ-39, optional)* — Checkbox einblendet vier Adressfelder, wenn der Zählpunkt nicht an Ihrer Wohnadresse liegt. Alle vier Felder müssen ausgefüllt werden, sobald die Checkbox aktiviert ist.

Über **Zählpunkt hinzufügen** können Sie bis zu 10 Zählpunkte angeben.

### Schritt 5b: Weitere Angaben *(typabhängig, PROJ-45)*

Nach der Zählpunkt-Eingabe erscheint — sofern Ihre EEG die zugehörigen Felder konfiguriert hat — der Block „Weitere Angaben". Welche Felder dort sichtbar sind, hängt vom Typ Ihrer Zählpunkte ab:

- **Verbraucher-Zählpunkt vorhanden:** „Personen im Haushalt", „Wärmepumpe", „E-Auto" (+ optional Anzahl/Jahres-km, falls E-Auto = Ja), „Warmwasser elektrisch"
- Bei reinen Erzeuger-Anträgen werden diese Verbraucher-Felder ausgeblendet.

> **Hinweis (seit PROJ-49):** Die früheren Application-Level-Felder „Verbrauch Vorjahr/Prognose", „Einspeisung Prognose" und „PV-Leistung (kWp)" werden jetzt **pro Zählpunkt** abgefragt — direkt im jeweiligen Zählpunkt-Block des Formulars, nicht mehr hier im allgemeinen Abschnitt. Bei mehreren Verbraucher- oder Erzeuger-Zählpunkten gibt es entsprechend mehrere Eingaben.

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
- Falls Ihre EEG die **Netzbetreiber-Vollmacht** (PROJ-44) verlangt, lesen Sie den Volltext der Vollmacht und bestätigen Sie sie per Häkchen — damit ermächtigen Sie die EEG, in Ihrem Namen Abstimmungen mit dem Netzbetreiber durchzuführen
- Klicken Sie auf **Antrag einreichen**

Nach der Einreichung erhalten Sie eine **Bestätigungs-E-Mail** mit Ihrer Antragsnummer (Format `<RC>-<Jahr>-<NNNN>`, z. B. `RC123456-2026-0001`). Die E-Mail enthält zusätzlich:

* eine PDF-Zusammenfassung Ihrer Angaben,
* eine Identifikations-Fußzeile mit Ihrer EEG, damit Sie die Mail eindeutig zuordnen können,
* eine **Reply-To**-Adresse, über die Sie direkt mit Ihrer EEG in Kontakt treten können (Antworten gehen nicht an einen „noreply"-Postfach).

## Schritt 7: E-Mail-Adresse bestätigen (nur bei aktivierten EEGs)

Wenn Ihre EEG das Feature **„E-Mail-Bestätigung erforderlich"** (PROJ-31) aktiviert hat, enthält Ihre Bestätigungs-Mail zusätzlich einen gelben Hinweisblock mit einem Button **„E-Mail-Adresse bestätigen"**. Der Link ist 30 Tage gültig. Erst nach dem Klick wird Ihr Antrag von der EEG bearbeitet.

In diesem Fall zeigt die Erfolgsmeldung direkt nach dem Einreichen den Hinweis **„Bitte prüfen Sie jetzt Ihr E-Mail-Postfach und bestätigen Sie Ihre E-Mail-Adresse über den zugesandten Link."** statt der Standard-Meldung „wird nun von unserem Team geprüft".

Ist das Feature in Ihrer EEG deaktiviert, entfällt dieser Schritt — der Antrag geht direkt in die Bearbeitung.

## Was passiert nach der Einreichung?

Ihr Antrag wird nun vom EEG-Betreiber geprüft. Mögliche nächste Schritte:

- **Rückfragen:** Der EEG-Betreiber kann Sie um Ergänzungen bitten. Sie erhalten eine E-Mail mit den Rückfragen und können Ihren Antrag ergänzen.
- **Genehmigung:** Ihr Antrag wird genehmigt und in eegFaktura importiert.
- **Ablehnung:** In Ausnahmefällen kann ein Antrag abgelehnt werden.
