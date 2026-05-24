# Admin-Einstellungen

Die Einstellungsseite ist über **Einstellungen** im Admin-Bereich erreichbar. Sie enthält alle EEG-spezifischen Konfigurationen.

## EEG auswählen

Wenn dein Account für mehrere EEGs zuständig ist, erscheint oben rechts ein Auswahlfeld. Alle Einstellungen beziehen sich auf die gewählte EEG.

## EEG-Stammdaten & SEPA-Mandat

In diesem Abschnitt steuerst du die öffentliche Registrierung und hinterlegst die Stammdaten für das SEPA-Lastschriftmandat.

![EEG-Einstellungen](images/admin-settings-eeg.png)

### Mitgliederregistrierung aktiv

![Mitgliederregistrierung aktiv](images/admin-settings-eeg-registration.png)

Der Toggle ganz oben steuert, ob der öffentliche Registrierungslink für deine EEG aktiv ist.

- **Aktiv**: Interessenten können sich über den Registrierungslink anmelden.
- **Inaktiv**: Besucher des Registrierungslinks erhalten eine Fehlermeldung. Bestehende Anträge sind davon nicht betroffen.

Neue EEGs starten standardmäßig als inaktiv. Aktiviere die Registrierung erst, wenn alle Einstellungen konfiguriert sind.

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

**Erstmaliger Sync nach Inbetriebnahme:** klicke einmal „Aus eegFaktura aktualisieren", damit die Stammdaten in die Onboarding-Datenbank kopiert werden. Bis dahin sind die Felder leer und die Hinweis-Box weist dich darauf hin.

### SEPA-Lastschriftmandat

![SEPA-Lastschriftmandat](images/admin-settings-eeg-sepa.png)

- **SEPA-Mandat von der EEG bereitstellen**: Wenn aktiv, generiert das Onboarding automatisch ein SEPA-Mandats-PDF.
- **Firmenlastschrift (B2B) für Unternehmen und Gemeinden anbieten**: Erscheint nur wenn SEPA aktiv ist. Aktiviere diese Option, wenn Unternehmen und Gemeinden ein B2B-Mandat erhalten sollen. Welche Mandats-Variante (Basislastschrift CORE oder Firmenlastschrift B2B) ein konkreter Antrag bekommt, wird **nicht** automatisch aus dem Mitgliedstyp abgeleitet — die Wahl trifft die Admin pro Antrag über das Feld „Einzugsart" (`core` / `b2b` / `kein_sepa`, Default `core`).
- **SEPA-Mandat erst beim Import erzeugen**: Wenn aktiv, wird das Mandat **nicht** dem Willkommensmail beigelegt, sondern erst beim Import mit der zugewiesenen Mitgliedsnummer als Mandatsreferenz ausgegeben. Sinnvoll, wenn du digitale Signatur (z. B. ID Austria) einsetzt — ein signiertes PDF darf nicht mehr verändert werden, daher muss die Mandatsreferenz vor der Signatur eingedruckt sein. Im Registrierungsformular erscheint dann ein erklärender Hinweis, dass das Mandat später folgt.

> **Hinweis:** Wenn das SEPA-Mandat aktiviert ist, aber Stammdaten fehlen, erscheint eine Warnung. Solange Felder fehlen, wird kein PDF generiert.

### Genossenschaftsanteile

![Genossenschaftsanteile](images/admin-settings-eeg-cooperative.png)

Nur relevant für EEGs, deren Rechtsträger eine Genossenschaft ist:

- **Genossenschaftsanteile erfassen**: Wenn aktiv, sehen neue Mitglieder im Registrierungsformular einen eigenen Block „Genossenschaftsanteile" mit Eingabefeld für die Anzahl gezeichneter Anteile und Live-Berechnung des Gesamtbetrags.
- **Pflichtanteile je Standort**: Mindestanzahl, die ein Mitglied zeichnen muss (z.B. 1, 3). Das Eingabefeld im Formular ist mit diesem Wert vorbefüllt und akzeptiert keine kleineren Werte; das Mitglied kann freiwillig mehr zeichnen.
- **Genossenschaftsanteilswert**: Preis pro Anteil in Euro (z.B. 100,00). Wird im Formular als Live-Multiplikator verwendet und in der Beitrittsbestätigung als eigene Sektion „GENOSSENSCHAFTSANTEILE" mit Anzahl × Wert = Gesamtbetrag ausgewiesen.

Beide Wert-Felder sind nur sichtbar, wenn der Toggle aktiv ist. Änderungen wirken **prospektiv** — bestehende Anträge bleiben unverändert, auch wenn das Pflichtmaß später angehoben wird. Falls ein Antrag dadurch unter dem aktuellen Pflichtmaß liegt, zeigt das Antrags-Detail einen orangen Hinweis, der Antrag bleibt aber unverändert.

Die Anteilsinformation wird **nicht** an eegFaktura übertragen — sie ist reine Onboarding-Erfassung als Buchhaltungs-Beleg.

### Zählpunkt-Prefixes

![Zählpunkt-Prefixes](images/admin-settings-eeg-mp-prefix.png)

Mitglieder müssen heute eine 33-stellige Zählpunktnummer eintippen. Wenn die Zählpunkte deiner EEG mehrheitlich vom selben Netzbetreiber + Postleitzahl-Bereich kommen, kannst du hier den festen Anfang vorgeben — das Mitglied tippt dann nur noch die individuellen letzten Stellen.

- **Verbraucher-Prefix**: Vorbelegung für Verbraucher-Zählpunkte (CONSUMPTION).
- **Einspeisungs-Prefix**: Vorbelegung für Einspeise-Zählpunkte (PRODUCTION).
- Beide sind unabhängig. Wenn nur eine Richtung konfiguriert ist, fällt die andere automatisch auf das reine „AT"-Pattern zurück (Mitglied tippt alle 31 Stellen nach „AT").

**Format**: muss mit `AT` beginnen, max 33 Stellen, danach Ziffern und Großbuchstaben (offizielle E-Control-Spec: Stellen 3–13 numerisch für Netzbetreibernummer + PLZ, Stellen 14–33 alphanumerisch für die Zählpunkt-Kennung). Whitespace, Punkte und Bindestriche werden beim Speichern automatisch entfernt — du kannst den Prefix also bequem mit Leerzeichen eintippen.

**Live-Vorschau** unter jedem Input zeigt, wie viele Stellen das Mitglied im Formular noch selbst eintippen muss („AT + 31 Stellen frei" bei leerem Feld, sonst „[Prefix] + N Stelle(n) vom Mitglied").

**Effekt im Mitgliedsformular**:
- Beim Wechsel der Zählpunkt-Richtung wird der passende Prefix automatisch in das Zählpunkt-Feld eingetragen.
- Der Prefix-Teil ist gelockt — das Mitglied kann ihn weder überschreiben noch backspacen.
- Beim Verlassen des Eingabefelds werden fehlende Stellen zwischen Prefix und Mitglieds-Eingabe mit führenden Nullen aufgefüllt (z. B. tippt das Mitglied `12345` und bekommt nach dem Klick weg `[Prefix]000000000012345`).
- Backend prüft beim Submit zusätzlich, dass jeder Zählpunkt mit dem konfigurierten Prefix der jeweiligen Richtung beginnt (defense-in-depth).

### Aktivierungs-Kriterium

![Aktivierungs-Kriterium](images/admin-settings-eeg-activation.png)

Steuert, wann eine Anwendung von **„Bereit zur Aktivierung"** auf **„Aktiviert"** wechselt. Beim Übergang auf „Aktiviert" wird automatisch die volle Beitrittsbestätigungs-Mail mit PDF an das Mitglied versandt (und eine Kopie an den EEG-Contact).

Zwei Optionen:

- **Variante A — „Mitglied wurde laut eegFaktura in die EEG aufgenommen"** (Default, rückwärtskompatibel):
  Der Teilnehmer im eegFaktura-Core hat den Status `ACTIVE`. Klassisches Verhalten — empfohlen für EEGs, die die formale Aufnahme erst nach Abschluss der Netzbetreiber-Anmeldung sehen wollen.

- **Variante B — „Für die Mitgliedschaft ist die Online-Registrierung gestartet"**:
  Mindestens ein Zählpunkt im Core hat den `processState` in PENDING / APPROVED / ACTIVE — sprich der Netzbetreiber hat auf die EDA-Online-Registrierung mindestens geantwortet. Damit aktivierst du Mitglieder bereits, sobald die Anmeldung beim Netzbetreiber **läuft**, ohne den Abschluss abzuwarten.

Der Wechsel selbst wird in beiden Fällen entweder **per Antrag manuell** ausgelöst (Button „Als aktiv markieren") oder über den Batch-Button **„Aktivierung im Core prüfen"** in der Antragsübersicht — der nimmt das hier gewählte Kriterium dann automatisch für deine ganze EEG.

### E-Mail-Adresse bestätigen

![E-Mail-Adresse bestätigen](images/admin-settings-eeg-email-confirm.png)

- **E-Mail-Adresse bestätigen**: Wenn aktiv, erhält das neue Mitglied in der Bestätigungs-Mail einen Button „E-Mail-Adresse bestätigen". Erst nach dem Klick wechselt der Antrag in den Status **„E-Mail bestätigt"** und ist für deine Bearbeitung freigegeben. Solange die Bestätigung aussteht, siehst du den Antrag mit dem Status „Eingereicht" und einer Warnung in der Detail-Ansicht.

Empfehlung: aktivieren, wenn du regelmäßig Müll-Anträge oder Tippfehler bei der E-Mail-Adresse erlebst. Vor dem ersten Lauf prüfen, dass die SMTP-Konfiguration stabil ist — sonst können Mitglieder nicht klicken.

Falls eine Bestätigungs-Mail im Spam-Ordner landet: in der Antragsdetail-Seite über **„Bestätigungs-Link erneut senden"** kann der Link erneut versendet werden (mit neuem Token; alter Link wird ungültig). Anträge, die 30 Tage lang nicht bestätigt werden, werden automatisch abgelehnt.

Klicke auf **Speichern**, um alle Änderungen in diesem Abschnitt zu übernehmen.

---

## Einleitungstext

![Einleitungstext](images/admin-settings-intro.png)

Der Einleitungstext wird oberhalb des Registrierungsformulars angezeigt. Er kann genutzt werden, um Interessenten zu begrüßen oder Hinweise zur Registrierung zu geben.

Unterstützte Formatierungen: **Fett**, *Kursiv*, Listen und Links. Wenn das Feld leer bleibt, wird ein Standardtext angezeigt.

Klicke auf **Speichern**, um den Text zu übernehmen.

---

## Formular-Felder & Zählpunktfelder

![Formular-Felder](images/admin-settings-fields.png)

Hier legst du fest, welche optionalen Felder im Registrierungsformular angezeigt werden.

Für jedes Feld stehen vier Zustände zur Verfügung:

| Zustand | Beschreibung |
|---------|--------------|
| **Ausgeblendet** | Das Feld ist im Registrierungsformular nicht sichtbar. |
| **Optional** | Das Feld wird angezeigt, muss aber nicht ausgefüllt werden. |
| **Verpflichtend** | Das Feld muss vom Mitglied ausgefüllt werden. |
| **Admin-Vorbefüllung** | Das Feld wird nicht im Formular angezeigt. Stattdessen wird der hier eingetragene Standardwert automatisch auf neue Anträge angewendet. |

### Typabhängige Sichtbarkeit (Badges)

Neben einigen Feldern stehen farbige **Badges**, die dir sofort zeigen, **unter welcher Bedingung** das Feld im Formular wirklich greift — auch wenn du es hier auf **Verpflichtend** stellst:

- **`[Verbraucher]`** *(blau)* — wird nur angezeigt, wenn der Zählpunkt CONSUMPTION ist bzw. der Antrag mindestens einen Verbraucher-Zählpunkt enthält (Application-Scope). Felder: Wärmepumpe, E-Auto, Anzahl E-Fahrzeuge, Jahres-Kilometer, Warmwasser elektrisch, Personen im Haushalt, Verbrauch Vorjahr, Verbrauch Prognose.
- **`[Einspeisung]`** *(amber)* — wird nur bei Erzeuger-Zählpunkten angezeigt. Felder: Einspeisung Prognose (alle Erzeugungsformen).
- **`[PV]`** *(orange, zusätzlich)* — gilt zusätzlich zu `[Einspeisung]` für Felder, die nur bei Erzeugungsform „PV" sinnvoll sind. Felder: Größe Batterie (kWh), Hersteller Wechselrichter, PV-Leistung (kWp), Einspeiselimit (kW).
- **`[+E-Auto]`** *(lila, zusätzlich)* — gilt zusätzlich zu `[Verbraucher]` für Felder, die nur greifen, wenn das Mitglied „E-Auto vorhanden" mit Ja beantwortet hat. Felder: Anzahl E-Fahrzeuge, Jahres-Kilometer.
- **`[+Speicher]`** *(grün, zusätzlich)* — gilt zusätzlich zu `[Einspeisung] [PV]` für Felder, die im Mitgliedsformular hinter dem Master-Toggle „Batteriespeicher vorhanden" gruppiert sind. Felder: Größe Batterie (kWh), Hersteller Wechselrichter, Speichersteuerung im Sinne der EEG vorstellbar?. Hinweis: Die Pflicht-Validierung der Speichersteuerungs-Frage greift zusätzlich nur dann, wenn das Mitglied tatsächlich Batterie-Daten gesetzt hat.

Neben jedem Feld mit Badge steht ein kleines **Info-Icon** — Klick/Hover zeigt die exakte Bedingung in Worten. Die Badges sind Single Source of Truth: ändert sich die Bedingung im Code, ändert sich auch die Badge ohne separate Pflege.

### Spezielle konfigurierbare Felder

- **Netzbetreiber-Vollmacht** *(Application-Scope)* — das Mitglied erteilt der EEG die Vollmacht, in seinem Namen mit dem Netzbetreiber zu agieren (notwendig z. B. bei Netz OÖ). Der Volltext der Vollmacht ist **fest im Code** und kann hier nicht editiert werden — du steuerst lediglich, ob die Checkbox überhaupt erscheinen soll. Default: `Ausgeblendet`. Bei `Verpflichtend` muss das Mitglied das Häkchen aktiv setzen, sonst wird der Antrag nicht submitted.
- **Größe Batterie (kWh) / Hersteller Wechselrichter** *(Zählpunkt-Scope)* — sammeln Speicher- und WR-Daten für PV-Erzeuger-Zählpunkte, um die EEG-Bewirtschaftung zu optimieren. Im Mitgliedsformular gruppiert hinter dem Master-Toggle „Batteriespeicher vorhanden". Default: `Ausgeblendet`.
- **Speichersteuerung im Sinne der EEG vorstellbar?** *(Zählpunkt-Scope, nur PV)* — Mitglied-Einverständnis, dass die EEG den Heimspeicher gemeinsam mit anderen Speichern der Mitglieder steuern darf. Sichtbar im Mitgliedsformular nur, wenn das Mitglied den Master-Toggle „Batteriespeicher vorhanden" aktiviert hat. Default: `Ausgeblendet`. Auf `Verpflichtend` setzen, wenn ohne Einverständnis kein Antrag möglich sein soll (greift jedoch nur, wenn das Mitglied tatsächlich einen Speicher angegeben hat — sonst wird die Frage gar nicht erst gestellt).
- **Verbrauch Vorjahr / Verbrauch Prognose** *(Zählpunkt-Scope)* — Energiewerte pro Verbraucher-Zählpunkt. Default: `Ausgeblendet`.
- **Einspeisung Prognose** *(Zählpunkt-Scope)* — jährliche Einspeise-Prognose pro Erzeuger-Zählpunkt (alle Erzeugungsformen). Default: `Ausgeblendet`.
- **PV-Leistung (kWp)** *(Zählpunkt-Scope, nur PV)* — installierte Spitzenleistung pro PV-Zählpunkt. Default: `Ausgeblendet`.
- **Einspeiselimit (kW)** *(Zählpunkt-Scope, nur PV)* — maximal zulässige Einspeiseleistung, wenn der Netzanschluss begrenzt ist. Mitglied wählt zuerst Ja/Nein und gibt bei Ja den Wert in kW ein. Default: `Ausgeblendet`.
- **Bankname** *(Application-Scope, ab 2026-05-18 konfigurierbar)* — bisher fix im Bankverbindungsblock angezeigt. Default `Optional` (bewahrt heutiges Verhalten). Auf `Ausgeblendet` setzen, wenn IBAN+Kontoinhaber genügen sollen; auf `Verpflichtend`, wenn der Bankname explizit gefordert ist (z. B. weil die EEG bei Auslandsüberweisungen die Bank kennen will).
- **Teilnahmefaktor (%)** *(Zählpunkt-Scope, ab 2026-05-19 konfigurierbar)* — bisher fix sichtbar im Mitgliedsformular, vorbelegt mit 100 %. Default `Optional` (bewahrt heutiges Verhalten — Mitglied sieht das Feld und kann den Wert ändern). Bei `Ausgeblendet` oder `Admin-Vorbefüllung` ist das Feld im Formular weg und der Wert wird serverseitig automatisch auf **100 %** gesetzt. Bei `Verpflichtend` bleibt das Feld sichtbar und mit 100 % vorbelegt — der Default macht den Wert technisch nie leer, das Pflicht-Häkchen erinnert das Mitglied nur, hinzuschauen. **In allen Modi** zeigen Beitrittsbestätigungs-PDF, Mail und Excel-Export den Teilnahmefaktor unverändert — der Toggle steuert nur die Public-Form-Sichtbarkeit, nicht die Render-Pfade.

Klicke auf **Konfiguration speichern**, um die Änderungen zu übernehmen.

---

## Rechtsdokumente

![Rechtsdokumente](images/admin-settings-legal.png)

Hier verwaltest du EEG-spezifische Dokumente (z.B. Satzung, Nutzungsbedingungen). Jedes Dokument wird auf eine von zwei Arten behandelt:

| Modus | Anzeige im Formular | Was wird protokolliert |
|---|---|---|
| **Mitglied muss zustimmen** | Checkbox direkt im Formular. Ohne Häkchen kann der Antrag nicht abgesendet werden. | „Zugestimmt am …" mit Zeitstempel im Antrag und im Beitrittsbestätigungs-PDF. |
| **Nur zur Information** | Das Dokument erscheint als Link im Block „Zur Information", kein Häkchen. | „Kenntnis genommen am …" mit Zeitstempel — die Kenntnisnahme erfolgt implizit mit dem Absenden des Antrags. |

Die Auswahl ist binär — ein „optional anhakbar" gibt es nicht mehr.

### Dokument hinzufügen

1. Klicke auf **Dokument hinzufügen**.
2. Gib einen Titel und die URL des Dokuments ein.
3. Wähle über den Schalter **„Mitglied muss zustimmen"** vs **„Nur zur Information"** — der Hinweistext unter dem Schalter erklärt das Verhalten.
4. Klicke auf **Hinzufügen**.

### Dokument bearbeiten oder löschen

Über die Symbole in der Dokumentenliste kannst du bestehende Einträge bearbeiten oder entfernen.

> **Hinweis:** Die zentrale Datenschutzerklärung (für alle EEGs gemeinsam) wird über die Servereinstellungen konfiguriert, nicht hier.

---

## Externe API

![Externe API](images/admin-settings-api.png)

Dieser Abschnitt zeigt den API-Key für die externe Registrierungs-API. Der Key ermöglicht das Einreichen von Mitgliedsanträgen über eine eigene Integration (z.B. ein Formular auf deiner Website).

> **Sicherheitshinweis:** Der API-Key darf ausschließlich server-seitig verwendet werden — niemals direkt in Browser-seitigem Code. Behandle ihn wie ein Passwort.

Über **Neuen Key generieren** kannst du den bestehenden Key ungültig machen und einen neuen ausstellen.

---

## Datenweiterleitung

![Datenweiterleitung](images/admin-settings-datenweiterleitung.png)

Asynchrone Weitergabe von Antragsdaten an externe Systeme. Aktuell verfügbar:

- **Excel/CSV-Export** — generiert eine Datei mit konfigurierbarem Feldsatz; pro-EEG anpassbar (welche Felder enthalten sind, in welcher Reihenfolge, mit welcher Spaltenüberschrift).
- Weitere Plugins (Zoho, HubSpot, …) lassen sich später als zusätzliche Implementierungen ergänzen — der Mechanismus dahinter ist generisch.

### Daten weiterleiten

Aus der **Antragsliste**:
1. Mehrere Anträge per Checkbox auswählen
2. Bulk-Aktion **Datenweiterleitung** klicken → Plugin wählen → Job läuft im Hintergrund

Aus dem **Antragsdetail**:
- Schaltfläche **Datenweiterleitung** in der Aktionsleiste — leitet den einzelnen Antrag weiter.

### Job-Übersicht

Auf dieser Seite siehst du den Verlauf aller Jobs (Status, Anzahl Anträge, Zeitpunkt, Ergebnisdatei zum Download). Fehlerhafte Jobs erzeugen automatisch eine Benachrichtigungs-E-Mail an die EEG-Kontaktadresse.

> **DSGVO-Hinweis:** Beim Hinzufügen sensibler Felder (IBAN, Geburtsdatum) zu einer Exportkonfiguration zeigt die UI eine Warnung. Die Verantwortung für die rechtmäßige Weiterverarbeitung liegt beim Empfänger-System.

---

## Konfiguration Import / Export

![Konfiguration Import/Export](images/admin-settings-import-export.png)

Sicherung und Übertragung der per-EEG-Konfiguration als versionierte JSON-Datei. Nützlich um:

- mehrere EEGs auf eine gemeinsame Grund-Konfiguration zu bringen,
- vor einem riskanten Apply den Ist-Zustand zu sichern,
- Konfigurations-Stände nachvollziehbar in Git zu halten.

### Export

Vier Sektionen sind einzeln oder als **Komplett-Bundle** exportierbar:

| Sektion | Inhalt |
|---|---|
| EEG-Einstellungen | Stammdaten, SEPA-Mandat-Settings, Aktivierungsmodus, Mitgliedsnummern-Startwert, Einleitungstext |
| Formular-Felder | Sichtbarkeit/Pflicht/Admin-only-Status aller konfigurierbaren Felder |
| Rechtsdokumente | Liste aller hinterlegten Dokumente mit Titel, URL und Zustimmungsmodus |
| Datenweiterleitungs-Konfig | Plugin-Konfigurationen für die Datenweiterleitung |

Dateiname enthält RC-Nummer und Zeitstempel — manuelle Versionierung in Git oder einem Backup-System ist damit unproblematisch.

### Import mit Diff-Preview

1. **Datei hochladen** (Drag-and-Drop oder Auswahldialog) — max 1 MB, nur `.json`. Die Datei wird serverseitig schemavalidiert; bei Fehlern wird der Upload abgelehnt.
2. **Diff-Preview** zeigt pro Sektion was sich ändert: hinzugefügt, modifiziert, entfernt oder unverändert.
3. **Sektionen aus-/abwählen** — nur ausgewählte werden tatsächlich angewendet.
4. **Apply** schreibt die Änderungen atomar (pro Sektion eine Transaktion). Apply ist **nicht** automatisch reversibel — daher der Hinweis oben, vorher die aktuelle Konfig zu exportieren.

> **Tipp:** Apply läuft mit einer `pg_advisory_xact_lock` — parallele Konfig-Änderungen über mehrere Browser-Tabs werden serialisiert, niemand überschreibt sich gegenseitig.
