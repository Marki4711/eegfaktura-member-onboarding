# Admin-Einstellungen

Die Einstellungsseite ist über **Einstellungen** im Admin-Bereich erreichbar. Sie enthält alle EEG-spezifischen Konfigurationen.

## EEG auswählen

Wenn Ihr Account für mehrere EEGs zuständig ist, erscheint oben rechts ein Auswahlfeld. Alle Einstellungen beziehen sich auf die gewählte EEG.

## EEG-Stammdaten & SEPA-Mandat

In diesem Abschnitt steuern Sie die öffentliche Registrierung und hinterlegen die Stammdaten für das SEPA-Lastschriftmandat.

![EEG-Einstellungen](images/admin-settings-eeg.png)

### Mitgliederregistrierung aktiv

Der Toggle ganz oben steuert, ob der öffentliche Registrierungslink für Ihre EEG aktiv ist.

- **Aktiv**: Interessenten können sich über den Registrierungslink anmelden.
- **Inaktiv**: Besucher des Registrierungslinks erhalten eine Fehlermeldung. Bestehende Anträge sind davon nicht betroffen.

Neue EEGs starten standardmäßig als inaktiv. Aktivieren Sie die Registrierung erst, wenn alle Einstellungen konfiguriert sind.

### EEG-Stammdaten & Logo — aus eegFaktura

Neun Werte werden direkt aus eegFaktura übernommen und sind in der Onboarding-Oberfläche **schreibgeschützt** (kleines Schloss-Symbol). Änderungen erfolgen ausschließlich in eegFaktura selbst, danach hier per „Aus eegFaktura aktualisieren" synchronisieren.

| Feld | Verwendung im Onboarding |
|---|---|
| **Gemeinschafts-ID** | Excel-Export (Spalte B), eegFaktura-Import |
| **EEG-Name** | Antrags-PDF, Willkommens- und Bestätigungs-Mail, SEPA-Mandat |
| **Straße, Hausnummer, PLZ, Ort** | SEPA-Mandat, Adressblock im Anschreiben |
| **Creditor-ID** | SEPA-Mandat (Pflichtfeld für gültige Lastschrift) |
| **Kontakt-E-Mail** | Empfänger-Adresse für die Admin-Benachrichtigung bei jedem neuen Antrag |
| **Logo** | Erscheint oben rechts auf Beitrittsbestätigung und SEPA-Mandat. Max 256 KB, PNG/JPEG/GIF. Bei größeren Logos in eegFaktura erscheint nach dem Sync ein orange-Hinweis unter der Logo-Vorschau („Logo überschreitet 256 KB"). |

**Stand-Anzeige am oberen Rand der Stammdaten-Card:**

- **Grün — „Synchron mit eegFaktura · Stand: DD.MM. HH:MM"**: die Daten stimmen mit eegFaktura überein, kein Handlungsbedarf.
- **Orange — „Stammdaten weichen ab"**: in eegFaktura wurden Daten geändert seit dem letzten Sync. Über **„Details anzeigen ▾"** sieht man eine Tabelle „Im Onboarding | In eegFaktura" je geändertem Feld. Mit **„Aus eegFaktura aktualisieren"** wird der lokale Stand überschrieben.
- **Grau — „eegFaktura nicht erreichbar"**: temporärer Ausfall des Core-Systems. Onboarding nutzt weiter den zuletzt gesyncten Stand.

**Erstmaliger Sync nach Inbetriebnahme:** klicken Sie einmal „Aus eegFaktura aktualisieren", damit die Stammdaten in die Onboarding-Datenbank kopiert werden. Bis dahin sind die Felder leer und die Hinweis-Box weist Sie darauf hin.

### SEPA-Lastschriftmandat

- **SEPA-Lastschriftmandat dem Willkommensmail anhängen**: Wenn aktiv, wird beim Einreichen eines Mitgliedsantrags automatisch ein PDF-Mandat generiert und als Anhang im Willkommensmail verschickt.
- **Firmenlastschrift (B2B)**: Erscheint nur wenn SEPA aktiv ist. Aktivieren Sie diese Option, wenn Unternehmen und Vereine das B2B-Mandat erhalten sollen. (Privatpersonen, Landwirte, Kleinunternehmer und Gemeinden bekommen weiterhin das Standard-CORE-Mandat — der B2B-Toggle ändert daran nichts.)

> **Hinweis:** Wenn das SEPA-Mandat aktiviert ist, aber Stammdaten fehlen, erscheint eine Warnung. Solange Felder fehlen, wird kein PDF generiert.

### Genossenschaftsanteile (PROJ-37)

Nur relevant für EEGs, deren Rechtsträger eine Genossenschaft ist:

- **Genossenschaftsanteile erfassen**: Wenn aktiv, sehen neue Mitglieder im Registrierungsformular einen eigenen Block „Genossenschaftsanteile" mit Eingabefeld für die Anzahl gezeichneter Anteile und Live-Berechnung des Gesamtbetrags.
- **Pflichtanteile je Standort**: Mindestanzahl, die ein Mitglied zeichnen muss (z.B. 1, 3). Das Eingabefeld im Formular ist mit diesem Wert vorbefüllt und akzeptiert keine kleineren Werte; das Mitglied kann freiwillig mehr zeichnen.
- **Genossenschaftsanteilswert**: Preis pro Anteil in Euro (z.B. 100,00). Wird im Formular als Live-Multiplikator verwendet und in der Beitrittsbestätigung als eigene Sektion „GENOSSENSCHAFTSANTEILE" mit Anzahl × Wert = Gesamtbetrag ausgewiesen.

Beide Wert-Felder sind nur sichtbar, wenn der Toggle aktiv ist. Änderungen wirken **prospektiv** — bestehende Anträge bleiben unverändert, auch wenn das Pflichtmaß später angehoben wird. Falls ein Antrag dadurch unter dem aktuellen Pflichtmaß liegt, zeigt das Antrags-Detail einen orangen Hinweis, der Antrag bleibt aber unverändert.

Die Anteilsinformation wird **nicht** an eegFaktura übertragen — sie ist reine Onboarding-Erfassung als Buchhaltungs-Beleg.

### E-Mail-Adresse bestätigen

- **E-Mail-Adresse bestätigen**: Wenn aktiv, erhält das neue Mitglied in der Bestätigungs-Mail einen Button „E-Mail-Adresse bestätigen". Erst nach dem Klick wechselt der Antrag in den Status **„E-Mail bestätigt"** und ist für Ihre Prüfung freigegeben. Solange die Bestätigung aussteht, sehen Sie den Antrag mit dem Status „Eingereicht" und einer Warnung in der Detail-Ansicht.

Empfehlung: aktivieren, wenn Sie regelmäßig Müll-Anträge oder Tippfehler bei der E-Mail-Adresse erleben. Vor dem ersten Lauf prüfen, dass die SMTP-Konfiguration stabil ist — sonst können Mitglieder nicht klicken.

Falls eine Bestätigungs-Mail im Spam-Ordner landet: in der Antragsdetail-Seite über **„Bestätigungs-Link erneut senden"** kann der Link erneut versendet werden (mit neuem Token; alter Link wird ungültig). Anträge, die 30 Tage lang nicht bestätigt werden, werden automatisch abgelehnt.

Klicken Sie auf **Speichern**, um alle Änderungen in diesem Abschnitt zu übernehmen.

---

## Einleitungstext

![Einleitungstext](images/admin-settings-intro.png)

Der Einleitungstext wird oberhalb des Registrierungsformulars angezeigt. Er kann genutzt werden, um Interessenten zu begrüßen oder Hinweise zur Registrierung zu geben.

Unterstützte Formatierungen: **Fett**, *Kursiv*, Listen und Links. Wenn das Feld leer bleibt, wird ein Standardtext angezeigt.

Klicken Sie auf **Speichern**, um den Text zu übernehmen.

---

## Formular-Felder & Zählpunktfelder

![Formular-Felder](images/admin-settings-fields.png)

Hier legen Sie fest, welche optionalen Felder im Registrierungsformular angezeigt werden.

Für jedes Feld stehen vier Zustände zur Verfügung:

| Zustand | Beschreibung |
|---------|--------------|
| **Ausgeblendet** | Das Feld ist im Registrierungsformular nicht sichtbar. |
| **Optional** | Das Feld wird angezeigt, muss aber nicht ausgefüllt werden. |
| **Verpflichtend** | Das Feld muss vom Mitglied ausgefüllt werden. |
| **Admin-Vorbefüllung** | Das Feld wird nicht im Formular angezeigt. Stattdessen wird der hier eingetragene Standardwert automatisch auf neue Anträge angewendet. |

### Typabhängige Sichtbarkeit (PROJ-45, Badges)

Neben einigen Feldern stehen farbige **Badges**, die Ihnen sofort zeigen, **unter welcher Bedingung** das Feld im Formular wirklich greift — auch wenn Sie es hier auf **Verpflichtend** stellen:

- **`[Verbraucher]`** *(blau)* — wird nur angezeigt, wenn der Antrag mindestens einen Verbraucher-Zählpunkt enthält. Felder: Wärmepumpe, E-Auto, Anzahl E-Fahrzeuge, Jahres-Kilometer, Warmwasser elektrisch, Personen im Haushalt, Verbrauch Vorjahr/Prognose.
- **`[Einspeisung]`** *(amber)* — wird nur angezeigt, wenn der Antrag mindestens einen Einspeise-Zählpunkt enthält. Felder: PV-Leistung (kWp), Einspeisung Prognose.
- **`[PV]`** *(orange, zusätzlich)* — gilt zusätzlich zu `[Einspeisung]` für Felder, die nur bei Erzeugungsform „PV" sinnvoll sind. Felder: Größe Batterie (kWh), Hersteller Wechselrichter.
- **`[+E-Auto]`** *(lila, zusätzlich)* — gilt zusätzlich zu `[Verbraucher]` für Felder, die nur greifen, wenn das Mitglied „E-Auto vorhanden" mit Ja beantwortet hat. Felder: Anzahl E-Fahrzeuge, Jahres-Kilometer.

Neben jedem Feld mit Badge steht ein kleines **Info-Icon** — Klick/Hover zeigt die exakte Bedingung in Worten. Die Badges sind Single Source of Truth: ändert sich die Bedingung im Code, ändert sich auch die Badge ohne separate Pflege.

### Spezielle konfigurierbare Felder

- **Netzbetreiber-Vollmacht** *(PROJ-44, Application-Scope)* — das Mitglied erteilt der EEG die Vollmacht, in seinem Namen mit dem Netzbetreiber zu agieren (notwendig z. B. bei Netz OÖ). Der Volltext der Vollmacht ist **fest im Code** und kann hier nicht editiert werden — Sie steuern lediglich, ob die Checkbox überhaupt erscheinen soll. Default: `Ausgeblendet`. Bei `Verpflichtend` muss das Mitglied das Häkchen aktiv setzen, sonst wird der Antrag nicht submitted.
- **Größe Batterie (kWh) / Hersteller Wechselrichter** *(PROJ-45, Zählpunkt-Scope)* — sammeln Speicher- und WR-Daten für PV-Erzeuger-Zählpunkte, um die EEG-Bewirtschaftung zu optimieren. Default: `Ausgeblendet`.

Klicken Sie auf **Konfiguration speichern**, um die Änderungen zu übernehmen.

---

## Rechtsdokumente

![Rechtsdokumente](images/admin-settings-legal.png)

Hier verwalten Sie EEG-spezifische Dokumente (z.B. Satzung, Nutzungsbedingungen). Jedes Dokument wird auf eine von zwei Arten behandelt:

| Modus | Anzeige im Formular | Was wird protokolliert |
|---|---|---|
| **Mitglied muss zustimmen** | Checkbox direkt im Formular. Ohne Häkchen kann der Antrag nicht abgesendet werden. | „Zugestimmt am …" mit Zeitstempel im Antrag und im Beitrittsbestätigungs-PDF. |
| **Nur zur Information** | Das Dokument erscheint als Link im Block „Zur Information", kein Häkchen. | „Kenntnis genommen am …" mit Zeitstempel — die Kenntnisnahme erfolgt implizit mit dem Absenden des Antrags. |

Die Auswahl ist binär — ein „optional anhakbar" gibt es nicht mehr.

### Dokument hinzufügen

1. Klicken Sie auf **Dokument hinzufügen**.
2. Geben Sie einen Titel und die URL des Dokuments ein.
3. Wählen Sie über den Schalter **„Mitglied muss zustimmen"** vs **„Nur zur Information"** — der Hinweistext unter dem Schalter erklärt das Verhalten.
4. Klicken Sie auf **Hinzufügen**.

### Dokument bearbeiten oder löschen

Über die Symbole in der Dokumentenliste können Sie bestehende Einträge bearbeiten oder entfernen.

> **Hinweis:** Die zentrale Datenschutzerklärung (für alle EEGs gemeinsam) wird über die Servereinstellungen konfiguriert, nicht hier.

---

## Externe API

![Externe API](images/admin-settings-api.png)

Dieser Abschnitt zeigt den API-Key für die externe Registrierungs-API. Der Key ermöglicht das Einreichen von Mitgliedsanträgen über eine eigene Integration (z.B. ein Formular auf Ihrer Website).

> **Sicherheitshinweis:** Der API-Key darf ausschließlich server-seitig verwendet werden — niemals direkt in Browser-seitigem Code. Behandeln Sie ihn wie ein Passwort.

Über **Neuen Key generieren** können Sie den bestehenden Key ungültig machen und einen neuen ausstellen.
