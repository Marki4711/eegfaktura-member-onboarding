# Admin-Einstellungen

Die Einstellungsseite ist über **Einstellungen** im Admin-Bereich erreichbar. Sie enthält alle EEG-spezifischen Konfigurationen.

## EEG auswählen

Wenn dein Account für mehrere EEGs zuständig ist, erscheint oben rechts ein Auswahlfeld. Alle Einstellungen beziehen sich auf die gewählte EEG.

Die Liste zeigt — sobald die EEG-Stammdaten mindestens einmal über den **Aktualisieren**-Knopf in dieser Karte synchronisiert wurden — die Kurzform der EEG aus eegFaktura zusammen mit der Referenznummer im Format `Kurzform • RC-Nummer` (Beispiel: `EEG-Test • RC0001`). EEGs, für die noch keine Kurzform in eegFaktura hinterlegt ist, erscheinen am Ende der Liste mit reiner Referenznummer.

Dasselbe Format findest du auch im **Antrags-Filterpanel** (Filter „EEG") und im **Reassign-Dialog** (Ziel-EEG beim Umordnen eines Antrags). In der Antragsliste selbst zeigt die Spalte „EEG" nur die Kurzform, mit der Referenznummer als Tooltip beim Überfahren.

## Einfache Ansicht oder Alle Optionen

Direkt neben der EEG-Auswahl gibt es einen Umschalter **Einfache Ansicht / Alle Optionen**. Die Wahl wird pro EEG gespeichert.

- **Einfache Ansicht** blendet erweiterte Optionen aus, die nur ein Bruchteil der EEGs braucht (SEPA-B2B, Mandat-Timing, Genossenschaftsanteile, Zählpunkt-Prefixes, E-Mail-Bestätigung, Aktivierungs-Kriterium und alle nicht-standardmäßig sichtbaren Formular-Felder). Die hinterlegten Werte bleiben in der Datenbank — sie sind nur nicht editierbar, solange du in der Einfachen Ansicht bist.
- **Alle Optionen** zeigt die komplette Konfiguration wie bisher. Bestehende EEGs starten mit Alle Optionen (rückwärts-kompatibel), neu angelegte EEGs starten mit der Einfachen Ansicht.

In dieser Doku sind erweiterte Abschnitte mit **„(Alle Optionen)"** im Header markiert.

### Welche Ansicht passt zu uns?

- **Einfache Ansicht wählen, wenn:** ihr eine kleine EEG seid, hauptsächlich Privatpersonen registriert, SEPA-Basislastschrift nutzt und keine speziellen Zählpunkt-Konventionen habt. Die ausgeblendeten Optionen brauchst du in 95 % der Fälle ohnehin nicht.
- **Alle Optionen wählen, wenn:** ihr Genossenschaftsanteile verlangt, B2B-Mandate für Unternehmen braucht, mit einem einheitlichen Netzbetreiber-Prefix arbeitet, E-Mail-Bestätigung als Spam-Schutz aktivieren wollt oder das Aktivierungs-Kriterium feiner steuern müsst.

Wenn in der Einfachen Ansicht eine erweiterte Option **aktiv** ist (z. B. SEPA-B2B wurde früher mal eingeschaltet), erscheint ein **gelber Hinweis-Banner** über den Tabs mit Button „Alle Optionen anzeigen". Damit ist sichergestellt, dass keine versteckte Einstellung unbemerkt wirkt.

## Automatischer Abgleich mit eegFaktura

Sobald du dich als Admin anmeldest, prüft das Tool **einmal pro Tag und EEG** im Hintergrund, ob deine offenen Onboarding-Anträge bereits als Mitglieder im eegFaktura-Kernsystem existieren. Wird ein eindeutiger Treffer gefunden (gleiche **IBAN UND E-Mail-Adresse**), passiert Folgendes:

- Der Antrag wird im Onboarding mit der Mitgliedsnummer aus eegFaktura verknüpft (sofern diese im Onboarding noch leer ist).
- Im Antrags-Verlauf erscheint ein Eintrag: **„In eegFaktura erfasst (automatischer Abgleich)"**.

**Warum gibt es das?**
Auch wenn du Onboarding-Daten nicht über den Import-Button überträgst, sondern manuell in eegFaktura eintippst, soll der Status im Onboarding-Tool nicht „stehen bleiben". Der Abgleich schließt diese Lücke: jedes Mitglied, das du im Onboarding erfasst hast und das später (auf welchem Weg auch immer) in eegFaktura landet, wird automatisch als verknüpft markiert.

**Was wird abgefragt?**
Bei jedem Abgleich wird die Liste der aktiven Teilnehmer deiner EEG aus eegFaktura abgerufen. Verglichen werden nur **IBAN** und **E-Mail-Adresse**. Vor- und Nachname, Adresse oder weitere Daten werden nicht aus eegFaktura übernommen — nur die Mitgliedsnummer wird zur Verknüpfung gespeichert. Details zur Verarbeitung stehen in der Datenverarbeitungsvereinbarung (AVV) zwischen EEG und Plattform-Betreiber.

**Was wird nicht überschrieben?**
Wenn du einen Antrag bereits per Import-Button verarbeitet hast und die Mitgliedsnummer schon gesetzt ist, ändert der Abgleich daran nichts. Findet der Abgleich eine Übereinstimmung mit einem Mitglied, dessen Mitgliedsnummer im Onboarding aber bereits einem anderen Antrag gehört (z. B. weil Mitglieder in eurer Familie dieselbe IBAN und E-Mail teilen und einer davon schon per Import bearbeitet wurde), wird der zweite Antrag zwar als verknüpft markiert, behält aber **keine fremde** Mitgliedsnummer — keine Datenkollision möglich.

**Wenn dein Antrag im Verlauf den Hinweis „In eegFaktura erfasst" zeigt:** alles in Ordnung. Der Antrag wurde automatisch mit dem Bestand verknüpft, eure manuelle Bearbeitung in eegFaktura wird sichtbar.

## Speichern, Auto-Speichern, Tab-Wechsel-Schutz

Die Einstellungsseite besteht aus mehreren Tabs. Welcher Tab wie speichert, ist bewusst pro Tab passend zur Bedienlogik gewählt:

| Tab | Speichern-Verhalten |
|---|---|
| **Stammdaten & SEPA** | **Auto-Speichern** mit Wartetext-Hinweis. Jede Änderung wird nach ca. einer halben Sekunde Tipp-Pause automatisch persistiert; oben in der Karte zeigt ein Status-Indikator „Speichert…" / „Gespeichert". Wenn ein Schalter eine Folge-Eingabe verlangt (z. B. Genossenschaftsanteile aktivieren, ohne Anteilswert anzugeben), wartet der Auto-Save freundlich, bis du fertig bist — direkt unter dem Schalter erscheint ein gelber Hinweis-Banner, der zeigt, was noch fehlt. |
| **Einleitungstext** | Expliziter **„Speichern"**-Button. Im Hintergrund läuft zusätzlich alle 30 Sekunden ein **Auto-Speichern als Sicherheitsnetz**, damit ein Browser-Crash dich nicht den ganzen Text kostet. |
| **Formular-Felder** | **Auto-Speichern.** Jede Toggle-Änderung wird automatisch persistiert; oben in der Karte zeigt ein Status-Indikator „Speichert…" / „Gespeichert". Es gibt keinen Speichern-Button mehr. |
| **Rechtsdokumente, Externe API, Datenweiterleitung, Import/Export** | Jede Aktion (Hinzufügen, Bearbeiten, Löschen, Schlüssel-Generieren …) wird **sofort** persistiert. Kein Sammel-Save nötig. |

**Hinweis-Banner bei Stammdaten & SEPA — wann erscheint er?**

Drei Schalter haben Pflicht-Folge-Eingaben:

- **Genossenschaftsanteile erfassen** → benötigt „Pflichtanteile je Standort" + „Anteilswert in Euro"
- **Im CORE-Mandat den elektronischen Audit-Trail nutzen** → benötigt zusätzlich „SEPA-Mandat erst beim Import senden"
- **SEPA-Feld für ausgewählte Mitgliedstypen auf optional setzen** → benötigt mindestens einen ausgewählten Mitgliedstyp

Wenn du einen dieser Schalter einschaltest, ohne die Folge-Felder auszufüllen, erscheint direkt unter dem Schalter ein gelber Hinweis:

> **Änderungen werden gespeichert, sobald die folgenden Pflichtfelder ausgefüllt sind:**
> - Pflichtanteile je Standort (aktuell leer)
> - Anteilswert in Euro (aktuell leer)

Sobald du die fehlenden Werte einträgst, verschwindet der Banner und der Auto-Save speichert. Du bekommst keinen roten Fehler-Toast wie früher beim klassischen Save-Button — der Editor wartet einfach freundlich, bis dein Stand sauber konfigurierbar ist.

**Wenn du einen Schalter wieder ausschaltest**, blendet der Editor die Sub-Felder vollständig aus und setzt deren Werte zurück. Die DB-Spalten werden beim nächsten Auto-Save geleert. Wenn du den Schalter später erneut einschaltest, startest du sauber bei null.

**Schutz vor Datenverlust:** Wenn du den Tab oder die EEG wechselst, während es in **Stammdaten**, **Einleitungstext** oder **Formular-Felder** ungespeicherte Änderungen gibt, erscheint ein Confirm-Dialog („Hier bleiben" / „Verwerfen und wechseln"). Tabs mit ungespeicherten Änderungen tragen außerdem ein orangenes Punkt-Symbol im Tab-Header. Beim Schließen des Browser-Tabs oder beim Refresh warnt zusätzlich der Browser selbst.

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

Das System erzeugt für jedes SEPA-Mitglied automatisch ein Mandat-PDF. Die zwei Toggles unten entscheiden, wie das PDF aussieht und wann es verschickt wird.

**Was bleibt jetzt immer gleich:**

- Im Mitglieder-Formular ist die SEPA-Lastschriftmandats-Checkbox direkt unter den Konto-Eingabefeldern als Pflicht-Auswahl sichtbar — das Mitglied stimmt mit seiner Auswahl der Lastschrift zu.
- Das SEPA-Mandat-PDF wird automatisch erzeugt (sofern die EEG-Stammdaten — Name, Adresse, Creditor-ID — vollständig sind).
- Der Bankverbindungs-Block bleibt im Formular Pflicht, außer das Mitglied wird im Admin auf „Einzugsart = kein SEPA" gesetzt.

#### Welche Toggle-Kombination ergibt was?

Im Modus *Alle Optionen* findest du zwei Toggles unterhalb des SEPA-Lastschriftmandat-Abschnitts:

- **Im CORE-Mandat den elektronischen Audit-Trail nutzen (statt manueller Unterschrift)** — entscheidet, ob das CORE-Mandat-PDF (Basislastschrift, Privat) einen Audit-Trail-Block oder ein klassisches Unterschriftenfeld trägt.
- **SEPA-Mandat erst beim Import senden (Mandatsreferenz = Mitgliedsnummer)** — entscheidet, ob das PDF sofort mit der Bestätigungs-Mail rausgeht (mit Platzhalter „Mandatsreferenz wird von der EEG ausgefüllt" — die EEG trägt sie später händisch nach) oder erst beim Import (Mandatsreferenz = Mitgliedsnummer).

Drei gültige Kombinationen:

| CORE-Audit | Timing | Was passiert |
|---|---|---|
| **aus** *(Default)* | **aus** *(Default)* | PDF kommt sofort mit der Bestätigungs-Mail, hat ein Unterschriftenfeld. Mitglied unterschreibt und sendet zurück. Mandatsreferenz bleibt leer (Platzhalter „wird von der EEG ausgefüllt") — die EEG trägt sie händisch nach. |
| **aus** | **an** | PDF kommt erst beim Import, hat ein Unterschriftenfeld. Mitglied unterschreibt und sendet zurück. Mandatsreferenz = Mitgliedsnummer. Sinnvoll für digitale Signatur-Workflows. |
| **an** | **an** *(automatisch erzwungen)* | PDF kommt erst beim Import, mit Audit-Trail-Block. Mitglied muss nichts mehr zurücksenden. Mandatsreferenz = Mitgliedsnummer. EEG-Kontakt bekommt eine zusätzliche Ablage-Kopie per E-Mail. |

> **Auto-Kopplung:** Wenn du den CORE-Audit-Toggle einschaltest, wird der Timing-Toggle automatisch mit aktiviert und gesperrt — denn ein Audit-Trail-PDF, das beim Submit-Zeitpunkt rausgeht, hätte noch keine Mitgliedsnummer und wäre damit ein unvollständiges Mandat.

> **EEG-Kopie bei Audit-Trail:** Im Audit-Pfad muss das Mitglied nichts zurücksenden — der EEG würde sonst kein Beleg-Exemplar haben. Deshalb geht beim Import zusätzlich zur Mitglieder-Mail eine separate Mail mit derselben PDF-Kopie an die EEG-Kontakt-Adresse. Subject: „Ablage-Kopie: SEPA-Mandat — *Mitgliedsname*, Antrag *Referenznummer*". Diese Kopie ist nur ein Ablage-Beleg, kein Action-Item.

**Warn-Banner:** Wenn EEG-Stammdaten unvollständig sind (Name, Adresse, Creditor-ID), zeigt der Editor eine gelbe Warnung. Solange Pflichtfelder fehlen, wird kein PDF erzeugt.

**Mandatsreferenz manuell überschreiben:** Die Mitgliedsnummer-basierte Auto-Ableitung beim Import-Zeitpunkt ist nicht zwingend. Im Admin-Edit-Form jedes Antrags gibt es ein Eingabefeld **Mandatsreferenz** (z. B. für externe Kundennummern aus eurem Buchhaltungssystem). Ein dort eingetragener Wert hat **Vorrang** und wird beim Import in eegFaktura mit übernommen (analog zum Mandatsdatum). Bei Submit-Zeit-PDFs (Timing aus, Audit aus) bleibt das Feld im PDF leer mit Platzhalter — die EEG trägt die Referenz später beim Bearbeiten des Mandats händisch nach.

#### B2B-Firmenlastschrift

B2B-Mandate (`einzugsart=b2b`, für Unternehmen, Gemeinden, Vereine) werden **immer** erzeugt — unabhängig vom CORE-Toggle, weil das SEPA-Regelwerk eine schriftliche Mandats-Vereinbarung für Firmenlastschriften verlangt. Die Mandatsreferenz ist immer die Mitgliedsnummer; das PDF kommt erst beim Import.

Im Modus *Alle Optionen* findest du einen separaten Toggle:

- **Im B2B-Mandat den elektronischen Audit-Trail nutzen (statt manueller Unterschrift)** — wirkt unabhängig vom CORE-Toggle, weil die Rechtsbewertung für Geschäftsleute (B2B) anders ausfallen kann als für Verbraucher (CORE).

Auch beim B2B-Audit-Pfad bekommt der EEG-Kontakt die Ablage-Kopie per E-Mail.

> **Beispiel-Wortlaut Audit-Trail-Block:**
> *Der Kunde hat der Musterstadt EEG nach Verifizierung seiner E-Mail-Adresse am 21.05.2026 11:50 von der IP-Adresse 192.0.2.42 auf elektronischem Weg (formfreie Willenserklärung gem. § 76 (3) EIWOG 2010) seine Zustimmung zum Vertrag im obigen Sinne sowie für das SEPA-Lastschriftmandat erteilt.*

> **Hinweis:** Wenn der Audit-Trail aktiv ist, aber für einen einzelnen Antrag noch keine IP-Adresse gespeichert wurde (Altanträge aus der Zeit vor der elektronischen Erfassung), fällt das PDF automatisch auf die klassische Unterschriftslinie zurück. Es kommt also nie zu einem fehlerhaften Audit-Block ohne Datenbasis.

#### Firmenlastschrift (B2B) und Vorbereitungs-Toggle

Für die B2B-Firmenlastschrift gibt es zwei Wege, die der Admin im Antrags-Bearbeiten-Dialog wählen kann.

**Weg 1 — Einzugsart B2B direkt:**

Im Bearbeiten-Dialog wählt der Admin *Einzugsart = Firmenlastschrift (B2B)*. Der Antrag geht beim Import direkt mit SEPA-B2B in den eegFaktura-Core. Die erste Lastschrift kann nur ausgeführt werden, sobald das Mitglied das B2B-Mandat bei seiner Hausbank registriert hat — ohne diese Bank-Registrierung lehnt die Bank die Lastschrift ab. Vorteil: kein nachträglicher Eingriff im Core nötig.

**Weg 2 — Einzugsart CORE + Vorbereitungs-Toggle (PROJ-91):**

Im Bearbeiten-Dialog wählt der Admin *Einzugsart = Basislastschrift (CORE)* und aktiviert zusätzlich den Schalter **„Mitglied für Umstellung auf B2B vorbereiten"** (nur sichtbar bei Einzugsart=CORE). Beim Import:

- Im Faktura-Core wird der Eintrag mit SEPA-Typ CORE angelegt → Lastschriften können sofort starten, die Klärungs-Phase mit der Bank überbrückt der CORE-Schutz (CORE-Lastschriften sind 8 Wochen lang rückbuchbar).
- Das Mitglied erhält **zwei PDFs** in der Mandat-Mail: das CORE-Mandat (zur Unterschrift und Rücksendung) und zusätzlich das B2B-Firmenlastschrift-Mandat (zum späteren Vorlegen bei der eigenen Bank).
- Die EEG-Kontaktperson und der Vorstand bekommen einen gelben Hinweis-Block in der jeweiligen Mail.

Sobald die Hausbank des Mitglieds die B2B-Registrierung bestätigt hat, geht der EEG-Admin in den Faktura-Core und stellt den SEPA-Typ dort manuell von CORE auf B2B um. Ab diesem Moment greift der B2B-Schutz (kein Rückbuchungsrecht).

> ⚠ **Hinweis B2B-Vorbereitung in der Mail (Weg 2):**
> *Für dieses Mitglied liegt eine B2B-Vorbereitung an. Das Konto ist im eegFaktura-Core als SEPA-CORE angelegt. Sobald die Hausbank des Mitglieds das B2B-Mandat bestätigt hat, stellen Sie den SEPA-Typ im Core manuell auf B2B um.*

**Beispiel — Musterbetrieb GmbH (Weg 2):** Der Antrag wurde mit Einzugsart B2B eingereicht; der Admin entscheidet sich beim Bearbeiten für *Einzugsart=CORE + Vorbereitungs-Toggle* (um sofort abbuchen zu können). Beim Import landet im Faktura-Core ein Eintrag mit SEPA-Typ CORE. Die Musterbetrieb GmbH bekommt CORE-Mandat **und** B2B-Mandat in einer Mail. Drei Wochen später meldet die Hausbank der Musterbetrieb GmbH die B2B-Registrierung — der EEG-Admin wechselt den SEPA-Typ im Faktura-Core manuell auf B2B. Bis dahin liefen die Lastschriften als CORE-Lastschrift mit normaler 8-Wochen-Rückbuchungsfrist.

**Bestand:** Anträge, die vor PROJ-91 mit Einzugsart B2B erstellt wurden, wurden bei der Migration auf *Einzugsart=CORE + Vorbereitungs-Toggle=an* umgestellt — sie verhalten sich jetzt wie Weg 2. Laufende B2B-Lastschriften mit aktivem Mandat im Core wurden nicht angetastet.

### SEPA-Feld für ausgewählte Mitgliedstypen auf optional setzen *(Alle Optionen)*

Manche EEGs erzwingen SEPA-Lastschrift nicht für alle Mitglieder — typische Fälle sind Privatmitglieder, die per Dauerauftrag oder Überweisung zahlen möchten, oder Vereine mit eigener Buchhaltung, die nicht per Lastschrift gezogen werden wollen. Mit diesem Toggle kannst du das SEPA-Einwilligungs-Feld im Mitgliederformular für ausgewählte Mitgliedstypen **optional** statt verpflichtend machen.

**Toggle aktiv:** Im Modus *Alle Optionen* öffnet sich darunter eine Liste mit fünf Checkboxen — *Privat*, *Pauschalierter Landwirt*, *Verein*, *Gemeinde*, *Unternehmen*. Du wählst, für welche Mitgliedstypen die SEPA-Einwilligung im Formular optional sein soll. Mitglieder dieser Typen können den Antrag dann auch ohne Häkchen bei der SEPA-Lastschriftermächtigung einreichen — in dem Fall wird die *Einzugsart* auf „Kein SEPA" gesetzt und es wird kein Mandat-PDF erzeugt.

**Toggle aus (Standard):** Verhalten wie bisher — alle Mitglieder müssen der SEPA-Lastschrift im Formular zustimmen.

**Auch Unternehmen sind wählbar:** Wenn ein Unternehmen die SEPA-Einwilligung weglässt, wird gar keine Lastschrift gezogen — das SEPA-B2B-Regelwerk (Firmenlastschrift mit unterschriebenem Mandat) greift dann nicht, weil es gar keine Lastschrift gibt. Die Abrechnung läuft manuell (Überweisung, Dauerauftrag). Wenn das Unternehmen aber per Lastschrift zahlen soll, bleibt das B2B-Mandat zwingend — dann darf es die Checkbox nicht weglassen.

**Wichtig — Bankdaten bleiben Pflicht:** IBAN und Kontowortlaut müssen im Formular auch dann ausgefüllt werden, wenn das Mitglied die SEPA-Einwilligung weglässt. Hintergrund: eegFaktura-Core verlangt Bankdaten für jedes Mitglied. Die EEG kann sie nach der Aktivierung für manuelle Zahlungsklärung nutzen (Überweisungsanforderung, späteres Mandat-Setup per Hand).

**Was passiert in den Folge-Mails?** Wenn ein Mitglied ohne SEPA-Mandat einreicht, bekommt die EEG in allen automatischen Info-Mails (Submit-Bestätigung, Aktivierungs-Mail, Beitrittserklärung an den Vorstand) einen gelben Hinweis-Banner:

> ⚠ **Kein SEPA-Lastschriftmandat erteilt** — die Bankdaten sind erfasst, aber kein Lastschriftauftrag aktiv. Die Abrechnung muss über einen alternativen Zahlungsweg (Überweisung, Dauerauftrag) direkt mit dem Mitglied vereinbart werden.

So fällt es im Postfach sofort auf, dass für diesen Antrag ein abweichender Abrechnungs-Ablauf nötig ist.

**Beispiel — Max Mustermann hakt die SEPA-Checkbox nicht an:** Im Antrags-Detail erscheint über der Bankverbindungs-Karte ein blauer Info-Streifen mit der Mitteilung „Kein SEPA-Lastschriftmandat erteilt — die Bankdaten sind erfasst, aber kein Lastschriftauftrag aktiv. Zahlungsmodalitäten direkt mit dem Mitglied vereinbaren." Die Bank-Informationen siehst du wie sonst auch, nur das Mandat-PDF entfällt.

**Bestehende Mitglieder umstellen:** Der Admin-Edit-Form für einen einzelnen Antrag erlaubt weiterhin manuell die *Einzugsart* zu ändern — das ist unabhängig von diesem Toggle. Der Toggle wirkt nur auf neue Submits über das öffentliche Formular.

### Beitrittserklärung vom Vorstand genehmigen lassen *(Alle Optionen)*

Manche EEGs wollen die Beitrittsbestätigung nicht automatisch ans Mitglied versenden lassen, sondern stattdessen vom Vorstand formell genehmigen lassen — z. B. weil die Statuten einen Aufnahmebeschluss erfordern oder die EEG den Beitrittsprozess mit einer unterschriebenen Bestätigung dokumentieren will.

**Toggle aktiv:** Beim Wechsel auf Status „Aktiviert" entfällt die automatische Beitrittsbestätigungs-Mail an das Mitglied. Stattdessen geht eine Beitrittserklärung mit Vorstands-Signaturblock am Ende an die in den EEG-Stammdaten hinterlegte Kontakt-E-Mail. Der Vorstand unterschreibt das Dokument und leitet es per Hand an das Mitglied weiter. Das Mitglied wird zusätzlich über die reguläre eegFaktura-Aktivierungs-Mail über seinen Status informiert.

**Toggle aus (Standard):** Verhalten wie bisher — Mitglied bekommt automatisch die Beitrittsbestätigung als PDF-Anhang zur Aktivierungs-Mail.

**Vor der Aktivierung sicherstellen:** Wenn der Toggle aktiv ist, **muss** die EEG-Kontakt-Mail gepflegt sein. Fehlt sie, bricht der Aktivierungs-Übergang ab und der Antrag bleibt im vorherigen Status. Du siehst die Fehlermeldung direkt im Aktivierungs-Vorgang.

**Beitrittserklärung im Nachhinein herunterladen:** Im Antrags-Detail eines aktivierten Mitglieds findest du im Dropdown **Herunterladen** den Eintrag *Beitrittserklärung* (zusätzlich zu Excel + Beitrittsbestätigung). Damit kannst du das PDF jederzeit neu erzeugen — z. B. wenn der Vorstand das Dokument verlegt hat. Sichtbar nur, wenn die EEG den Vorstands-Genehmigungs-Workflow aktiv hat und der Antrag den Status *Aktiviert* erreicht hat.

### Genossenschaftsanteile *(Alle Optionen)*

> Diese Sektion ist nur in **Alle Optionen** sichtbar.

![Genossenschaftsanteile](images/admin-settings-eeg-cooperative.png)

Nur relevant für EEGs, deren Rechtsträger eine Genossenschaft ist:

- **Genossenschaftsanteile erfassen**: Wenn aktiv, sehen neue Mitglieder im Registrierungsformular einen eigenen Block „Genossenschaftsanteile" mit Eingabefeld für die Anzahl gezeichneter Anteile und Live-Berechnung des Gesamtbetrags.
- **Pflichtanteile je Standort**: Mindestanzahl, die ein Mitglied zeichnen muss (z.B. 1, 3). Das Eingabefeld im Formular ist mit diesem Wert vorbefüllt und akzeptiert keine kleineren Werte; das Mitglied kann freiwillig mehr zeichnen.
- **Genossenschaftsanteilswert**: Preis pro Anteil in Euro (z.B. 100,00). Wird im Formular als Live-Multiplikator verwendet und in der Beitrittsbestätigung als eigene Sektion „GENOSSENSCHAFTSANTEILE" mit Anzahl × Wert = Gesamtbetrag ausgewiesen.

Beide Wert-Felder sind nur sichtbar, wenn der Toggle aktiv ist. Änderungen wirken **prospektiv** — bestehende Anträge bleiben unverändert, auch wenn das Pflichtmaß später angehoben wird. Falls ein Antrag dadurch unter dem aktuellen Pflichtmaß liegt, zeigt das Antrags-Detail einen orangen Hinweis, der Antrag bleibt aber unverändert.

Die Anteilsinformation wird **nicht** an eegFaktura übertragen — sie ist reine Onboarding-Erfassung als Buchhaltungs-Beleg.

### Zählpunkt-Prefixes *(Alle Optionen)*

> Diese Sektion ist nur in **Alle Optionen** sichtbar.

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

### Aktivierungs-Kriterium *(Alle Optionen)*

> Diese Sektion ist nur in **Alle Optionen** sichtbar.

![Aktivierungs-Kriterium](images/admin-settings-eeg-activation.png)

Steuert, wann eine Anwendung von **„Bereit zur Aktivierung"** auf **„Aktiviert"** wechselt. Beim Übergang auf „Aktiviert" wird automatisch die volle Beitrittsbestätigungs-Mail mit PDF an das Mitglied versandt (und eine Kopie an den EEG-Contact).

Zwei Optionen:

- **Variante A — „Mitglied wurde laut eegFaktura in die EEG aufgenommen"** (Default, rückwärtskompatibel):
  Der Teilnehmer im eegFaktura-Core hat den Status `ACTIVE`. Klassisches Verhalten — empfohlen für EEGs, die die formale Aufnahme erst nach Abschluss der Netzbetreiber-Anmeldung sehen wollen.

- **Variante B — „Für die Mitgliedschaft ist die Online-Registrierung gestartet"**:
  Mindestens ein Zählpunkt im Core hat den `processState` in PENDING / APPROVED / ACTIVE — sprich der Netzbetreiber hat auf die EDA-Online-Registrierung mindestens geantwortet. Damit aktivierst du Mitglieder bereits, sobald die Anmeldung beim Netzbetreiber **läuft**, ohne den Abschluss abzuwarten.

Der Wechsel selbst wird in beiden Fällen entweder **per Antrag manuell** ausgelöst (Button „Als aktiv markieren") oder über den Batch-Button **„Aktivierung im Core prüfen"** in der Antragsübersicht — der nimmt das hier gewählte Kriterium dann automatisch für deine ganze EEG.

### E-Mail-Adresse bestätigen *(Alle Optionen)*

> Diese Sektion ist nur in **Alle Optionen** sichtbar.

![E-Mail-Adresse bestätigen](images/admin-settings-eeg-email-confirm.png)

- **E-Mail-Adresse bestätigen**: Wenn aktiv, erhält das neue Mitglied in der Bestätigungs-Mail einen Button „E-Mail-Adresse bestätigen". Erst nach dem Klick wechselt der Antrag in den Status **„E-Mail bestätigt"** und ist für deine Bearbeitung freigegeben. Solange die Bestätigung aussteht, siehst du den Antrag mit dem Status „Eingereicht" und einer Warnung in der Detail-Ansicht.

Empfehlung: aktivieren, wenn du regelmäßig Müll-Anträge oder Tippfehler bei der E-Mail-Adresse erlebst. Vor dem ersten Lauf prüfen, dass die SMTP-Konfiguration stabil ist — sonst können Mitglieder nicht klicken.

Falls eine Bestätigungs-Mail im Spam-Ordner landet: in der Antragsdetail-Seite über **„Bestätigungs-Link erneut senden"** kann der Link erneut versendet werden (mit neuem Token; alter Link wird ungültig). Anträge, die 30 Tage lang nicht bestätigt werden, werden automatisch abgelehnt.

### Erscheinungsbild der Online-Registrierung *(Alle Optionen)*

> Diese Sektion ist nur in **Alle Optionen** sichtbar.

Die öffentliche Mitglied-werden-Seite (`/register/<RC-Nummer>`) lässt sich pro EEG farblich an die eigene Marke anpassen. Der Editor hat zwei Modi mit eigenen Tabs:

- **Preset** — vier vordefinierte Farbwelten, durchgetestet und WCAG-konform
- **Eigene Farben** — freie HEX-Farbwahl plus optional Schriftart, mit Live-Vorschau und Kontrast-Check

Der Modus wird oben am Editor umgeschaltet. Die Einstellungen beider Modi bleiben gespeichert — wechselst du z. B. von „Eigene Farben" zurück auf „Preset" und speicherst, bleibt das individuelle Theme in der Datenbank erhalten und ist beim erneuten Wechsel auf „Eigene Farben" wieder da.

#### Tab „Preset"

| Preset | Wirkung |
|---|---|
| **Teal (Standard)** | Heutige dunkel-türkise Optik. Default für alle EEGs. |
| **Leaf (Grün)** | Dunkles Waldgrün mit hellem Grünton als Akzent. |
| **Sun (Orange)** | Warmes Anthrazit mit Sonnen-Orange-Akzent. |
| **Slatey (Neutral)** | Neutrales Anthrazit-Grau mit hellblauem Akzent. |

Vier kleine Vorschau-Karten direkt unter dem Auswahl-Feld zeigen, wie das jeweilige Schema aussieht — inklusive einer Beispiel-Schaltfläche und einer Beispiel-Karte in den Preset-Farben. Die aktuelle Auswahl bekommt einen Ring um die Karte.

Wenn ein anderer als „Teal" gewählt ist, wirken zusätzlich:

- **EEG-Logo**: das aus eegFaktura synchronisierte Logo wird oben links im Kopf der Mitglied-werden-Seite eingebunden — neben dem EEG-Namen aus der Stammdaten-Synchronisation. Ohne Logo bleibt das Standard-Blitz-Icon.
- **EEG-Name**: ersetzt im Kopf den Schriftzug „eegFaktura". Wenn der Name noch nicht synchronisiert ist, bleibt der Standard-Schriftzug.
- **Footer**: wechselt dezent auf „Powered by eegFaktura", damit klar bleibt, wer den Service technisch betreibt.

**Beispiel:** die Muster-EEG schaltet im Editor auf „Leaf", synchronisiert ihr Logo aus eegFaktura, und sieht beim nächsten Aufruf der Mitglied-werden-Seite einen grünen Kopf mit Muster-EEG-Logo und Namen statt der eegFaktura-Standard-Optik.

**Wo wirkt das Branding nicht?**

- Die separate E-Mail-Bestätigungs-Seite (`/confirm-email`) zeigt weiter das Standard-Theme — sie erreicht das Mitglied über einen Link mit Token statt über eine RC-Nummer und kennt deshalb die EEG nicht.
- Der Admin-Bereich (`/admin/…`) bleibt einheitlich türkis.
- E-Mail-Templates sind heute nicht gebrandet.

#### Tab „Eigene Farben"

Reicht dir keines der Presets, wechselst du auf den Tab „Eigene Farben". Hier kannst du jede Farbe einzeln über einen Color-Picker oder per HEX-Eingabe festlegen. Acht editierbare Felder:

| Feld | Wirkt auf |
|---|---|
| **Hauptfarbe** | Button-Hintergrund, Akzent-Strich im Kopf |
| **Text auf Hauptfarbe** | Schrift auf Buttons |
| **Akzent** | Highlights, Hover-Zustände |
| **Text auf Akzent** | Schrift auf Akzent-Flächen |
| **Hintergrund** | Seiten-Hintergrund |
| **Text** | Standard-Textfarbe |
| **Karten-Hintergrund** | Formularkarten und Boxen |
| **Text auf Karten** | Schrift in Formularen |

Plus eine Auswahl für die **Schriftart**: Sans-Serif (Standard Inter), Serif (Georgia), Monospace (SF Mono) oder System-UI (native Schrift des Endgeräts). Felder, die du nicht anpasst, werden vom oben gewählten Preset übernommen — weitere neun abgeleitete Werte (z. B. Rahmen, Focus-Ring, gedämpfte Töne) berechnet das System automatisch aus deinen acht Farben.

**Live-Vorschau** unter den Pickern zeigt sofort, wie die Mitglied-werden-Seite aussehen wird. Daneben prüft ein **Kontrast-Panel** die drei kritischen WCAG-AA-Paare:

- Hauptfarbe gegen Text auf Hauptfarbe
- Akzent gegen Text auf Akzent
- Text gegen Hintergrund

Liegt ein Paar unter dem Mindest-Verhältnis 4,5:1, blockiert ein Sicherheitsfilter das Speichern — die Mitglied-werden-Seite würde sonst Texte zeigen, die schwer oder gar nicht lesbar sind. Du siehst direkt im Panel welches Paar Probleme macht und welches Kontrast-Verhältnis aktuell erreicht wird.

**Beispiel:** Muster-EEG wechselt auf den Tab „Eigene Farben", lädt zunächst das Preset „Leaf" als Basis (links oben), nimmt für **Hauptfarbe** ein dunkles Markengrün `#1c3a28` und für **Text auf Hauptfarbe** Weiß `#ffffff` — das Kontrast-Panel zeigt 12,4:1 ✓. Background-Felder bleiben vom Preset. Speichern, fertig — die Mitglied-werden-Seite zeigt jetzt grüne Buttons auf dem grünen Hintergrund der Leaf-Welt.

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

> **Sichtbarkeit nach Ansicht:** In der **Einfachen Ansicht** siehst du nur die drei Felder, die im Catalog als „Optional" voreingestellt sind — *Telefon*, *Geburtsdatum* (Application-Scope) und *Teilnahmefaktor* (Zählpunkt-Scope). Alle übrigen Felder bleiben verborgen, ihre hinterlegten Werte (falls schon konfiguriert) bleiben aktiv. Wechsle auf **Alle Optionen**, um die volle Liste (~27 Felder) zu sehen und zu pflegen.

Für jedes Feld stehen vier Zustände zur Verfügung:

| Zustand | Beschreibung |
|---------|--------------|
| **Ausgeblendet** | Das Feld ist im Registrierungsformular nicht sichtbar. |
| **Optional** | Das Feld wird angezeigt, muss aber nicht ausgefüllt werden. |
| **Verpflichtend** | Das Feld muss vom Mitglied ausgefüllt werden. |
| **Admin-Vorgabe** | Das Feld wird **nicht** im Mitglieder-Formular angezeigt. Im Admin-Bereich kannst du es pro Antrag im **Bearbeiten**-Dialog eintragen — z. B. wenn du als EEG-Admin ein Feld führst, das das Mitglied nicht selbst pflegen soll, aber pro Antrag unterschiedlich sein kann. |

### Typabhängige Sichtbarkeit (Badges)

Neben einigen Feldern stehen farbige **Badges**, die dir sofort zeigen, **unter welcher Bedingung** das Feld im Formular wirklich greift — auch wenn du es hier auf **Verpflichtend** stellst:

- **`[Verbraucher]`** *(blau)* — wird nur angezeigt, wenn der Zählpunkt CONSUMPTION ist bzw. der Antrag mindestens einen Verbraucher-Zählpunkt enthält (Application-Scope). Felder: Wärmepumpe, E-Auto, Anzahl E-Fahrzeuge, Jahres-Kilometer, Warmwasser elektrisch, Personen im Haushalt, Verbrauch Vorjahr, Verbrauch Prognose.
- **`[Einspeisung]`** *(amber)* — wird nur bei Erzeuger-Zählpunkten angezeigt. Felder: Einspeisung Prognose (alle Erzeugungsformen).
- **`[PV]`** *(orange, zusätzlich)* — gilt zusätzlich zu `[Einspeisung]` für Felder, die nur bei Erzeugungsform „PV" sinnvoll sind. Felder: Größe Batterie (kWh), Hersteller Wechselrichter, PV-Leistung (kWp), Einspeiselimit (kW).
- **`[+E-Auto]`** *(lila, zusätzlich)* — gilt zusätzlich zu `[Verbraucher]` für Felder, die nur greifen, wenn das Mitglied „E-Auto vorhanden" mit Ja beantwortet hat. Felder: Anzahl E-Fahrzeuge, Jahres-Kilometer.
- **`[+Speicher]`** *(grün, zusätzlich)* — gilt zusätzlich zu `[Einspeisung] [PV]` für Felder, die im Mitgliedsformular hinter dem Master-Toggle „Batteriespeicher vorhanden" gruppiert sind. Felder: Größe Batterie (kWh), Hersteller Wechselrichter, Speichersteuerung im Sinne der EEG vorstellbar?. Hinweis: Die Pflicht-Validierung der Speichersteuerungs-Frage greift zusätzlich nur dann, wenn das Mitglied tatsächlich Batterie-Daten gesetzt hat.

Neben jedem Feld mit Badge steht ein kleines **Info-Icon** — Klick/Hover zeigt die exakte Bedingung in Worten. Die Badges sind Single Source of Truth: ändert sich die Bedingung im Code, ändert sich auch die Badge ohne separate Pflege.

### Antragsteller-Felder (Application-Scope)

![Antragsteller-Felder](images/admin-settings-fields-applicant.png)

Felder, die einmal pro Antrag erfasst werden. Badges zeigen typabhängige Sichtbarkeit (siehe oben).

### Zählpunkt-Felder (Zählpunkt-Scope)

![Zählpunkt-Felder](images/admin-settings-fields-metering.png)

Felder, die pro Zählpunkt im Mitgliedsformular erscheinen — bei mehreren Zählpunkten entsprechend mehrfach.

### Spezielle konfigurierbare Felder

- **Netzbetreiber-Vollmacht** *(Application-Scope)* — das Mitglied erteilt der EEG die Vollmacht, in seinem Namen mit dem Netzbetreiber zu agieren (notwendig z. B. bei Netz OÖ). Der Volltext der Vollmacht ist **fest im Code** und kann hier nicht editiert werden — du steuerst lediglich, ob die Checkbox überhaupt erscheinen soll. Default: `Ausgeblendet`. Bei `Verpflichtend` muss das Mitglied das Häkchen aktiv setzen, sonst wird der Antrag nicht submitted.
  - **Praxis-Hinweis Netz OÖ** — wenn beim Online-Portal der Netz OÖ ein Onboarding-Schritt klemmt, geht ein Kontakt der EEG mit dem Netzbetreiber meist schneller als der des Mitglieds. Die Netz OÖ verlangt für ein solches Handeln im Mitglieds-Auftrag die unterschriebene Vollmacht — Toggle daher auf `Verpflichtend` oder zumindest `Optional`.
  - **Praxis-Hinweis Salzburg Netz** — kein Vollmachts-Workflow; Salzburg Netz arbeitet stattdessen mit **Kundennummer + Vertragskontonummer**, die das Mitglied selbst aus seinem Portal-/Rechnungs-Dokument heraussucht. In diesen Fällen bringt die Vollmacht keinen Mehrwert — Toggle kann auf `Ausgeblendet` bleiben. Die beiden Nummern sind heute nicht eigenständig im Formular abgefragt; sie kommen üblicherweise über die normale Netzbetreiber-Kommunikation des Mitglieds.
  - **Faustregel** — vor dem Aktivieren beim jeweiligen Netzbetreiber kurz nachfragen, ob er die Vollmacht akzeptiert bzw. überhaupt benötigt. Manche Netzbetreiber fordern statt der Vollmacht spezifische Mitglieds-Daten (Kunden-/Vertragsnummer, Zähler-Inventarnummer); diese können als konfigurierbare Felder hinterlegt werden (siehe [Netzbetreiber-Info-PDF](07-emails-and-pdfs.md)).
- **Größe Batterie (kWh) / Hersteller Wechselrichter** *(Zählpunkt-Scope)* — sammeln Speicher- und WR-Daten für PV-Erzeuger-Zählpunkte, um die EEG-Bewirtschaftung zu optimieren. Im Mitgliedsformular gruppiert hinter dem Master-Toggle „Batteriespeicher vorhanden". Default: `Ausgeblendet`.
- **Speichersteuerung im Sinne der EEG vorstellbar?** *(Zählpunkt-Scope, nur PV)* — Mitglied-Einverständnis, dass die EEG den Heimspeicher gemeinsam mit anderen Speichern der Mitglieder steuern darf. Sichtbar im Mitgliedsformular nur, wenn das Mitglied den Master-Toggle „Batteriespeicher vorhanden" aktiviert hat. Default: `Ausgeblendet`. Auf `Verpflichtend` setzen, wenn ohne Einverständnis kein Antrag möglich sein soll (greift jedoch nur, wenn das Mitglied tatsächlich einen Speicher angegeben hat — sonst wird die Frage gar nicht erst gestellt).
- **Verbrauch Vorjahr / Verbrauch Prognose** *(Zählpunkt-Scope)* — Energiewerte pro Verbraucher-Zählpunkt. Default: `Ausgeblendet`.
- **Einspeisung Prognose** *(Zählpunkt-Scope)* — jährliche Einspeise-Prognose pro Erzeuger-Zählpunkt (alle Erzeugungsformen). Default: `Ausgeblendet`.
- **PV-Leistung (kWp)** *(Zählpunkt-Scope, nur PV)* — installierte Spitzenleistung pro PV-Zählpunkt. Default: `Ausgeblendet`.
- **Einspeiselimit (kW)** *(Zählpunkt-Scope, nur PV)* — maximal zulässige Einspeiseleistung, wenn der Netzanschluss begrenzt ist. Mitglied wählt zuerst Ja/Nein und gibt bei Ja den Wert in kW ein. Default: `Ausgeblendet`.
- **Bankname** *(Application-Scope, ab 2026-05-18 konfigurierbar)* — bisher fix im Bankverbindungsblock angezeigt. Seit 2026-05-31 Default `Ausgeblendet` (zuvor `Optional`) — Bankname ist für die meisten EEGs nicht zentral, IBAN reicht. Auf `Optional` setzen wenn du das Feld im Mitgliederformular anzeigen willst; auf `Verpflichtend`, wenn der Bankname explizit gefordert ist (z. B. weil die EEG bei Auslandsüberweisungen die Bank kennen will). Bestehende EEGs mit explizitem Eintrag bleiben unverändert.
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

![Dokument hinzufügen — Dialog](images/admin-settings-legal-add.png)

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

### Nutzung der API

Die externe API erlaubt einen einzigen Endpunkt: das **Einreichen eines Antrags** im Namen einer EEG. Der API-Key identifiziert die EEG; eine `rcNumber` im Request-Body ist nicht nötig.

**Authentifizierung** — der Key wird im `Authorization`-Header gesendet:

```
Authorization: Bearer moak_<dein-32-stelliger-key>
Content-Type: application/json
```

**Beispiel-Aufruf** mit `curl`:

```bash
curl -X POST https://member-onboarding.eegfaktura.at/api/external/v1/applications \
  -H "Authorization: Bearer moak_dein32stelligerkeyhier...." \
  -H "Content-Type: application/json" \
  -d '{
    "memberType": "private",
    "firstname": "Max",
    "lastname": "Mustermann",
    "email": "max.mustermann@example.org",
    "residentStreet": "Musterstraße",
    "residentStreetNumber": "1",
    "residentZip": "8010",
    "residentCity": "Graz",
    "residentCountry": "AT",
    "iban": "AT61 1904 3002 3457 3201",
    "accountHolder": "Max Mustermann",
    "privacyAccepted": true,
    "sepaMandateAccepted": true,
    "meteringPoints": [
      {
        "meteringPoint": "AT0010000000000000001000000000001",
        "direction": "CONSUMPTION",
        "participationFactor": 100
      }
    ]
  }'
```

Bei Erfolg antwortet die API mit Status `201` und der eindeutigen `id` plus `referenceNumber` des angelegten Antrags. Der Antrag landet direkt im Status „Eingereicht" und durchläuft danach denselben Prüfprozess wie ein Antrag aus dem öffentlichen Formular.

**Wichtige Limits:**

- Maximal **10 Anfragen pro 60 Sekunden** pro Key (Burst-Schutz).
- Maximal **200 Einreichungen pro Tag** pro Key (UTC-Reset um Mitternacht).
- Bei Überschreitung antwortet die API mit `429 Too Many Requests` und einem `Retry-After`-Header.

**Mitgliedstypen** und **Pflichtfelder** orientieren sich an der `field_config` der EEG — die gleichen Regeln wie im öffentlichen Formular. Für `private`/`farmer` sind `firstname` + `lastname` Pflicht, für `company`/`municipality`/`association` ist `companyName` Pflicht.

**Vollständige technische Referenz** (alle Felder, Validierungen, Fehler-Codes, Mitgliedstyp-Details): [API-Spezifikation, Sektion „External API"](https://github.com/Marki4711/eegfaktura-member-onboarding/blob/main/docs/api-spec.md#8-external-api).

**Neu seit Juni 2026:** Du kannst im Request-Body das optionale Feld `submitterIp` mitgeben (z.B. `"submitterIp": "203.0.113.42"`). Wenn dein Backend die End-User-IP aus dem ursprünglichen Browser-Request kennt, dokumentiert die Plattform sie als Audit-Trail im SEPA-Firmenlastschrift-Mandat-PDF (rechtskonform nach § 76 (3) EIWOG 2010). Ohne dieses Feld wird das PDF im klassischen Format mit Datum/Unterschriftsfeld erstellt.

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

### Welche Felder gibt es

Die Spalten-Auswahl im Editor enthält drei Arten von Feldern:

- **Stammdaten und Zählpunkt-Daten** — werden vom Mitglied im Public-Form eingegeben (z. B. Name, Adresse, IBAN, Zählpunkt-Bezeichnung).
- **Form-Felder mit Sichtbarkeits-Schalter** — werden nur befüllt, wenn das EEG das Feld unter **Einstellungen → Formular-Felder** auf *Optional*, *Pflicht* oder *Nur Admin* gestellt hat. Steht das Feld auf *Ausgeblendet*, sieht das Mitglied es nicht im Formular und der Wert bleibt leer. Beispiele: Beitrittsdatum, Personen im Haushalt, Wärmepumpe, E-Auto-Details, Genossenschaftsanteile.
- **System-Zeitstempel** — werden automatisch vom System gesetzt, wenn der Antrag den entsprechenden Status erreicht. Diese Werte sind immer befüllt, sobald der Status durchlaufen wurde.

Die wichtigsten zeitbezogenen Felder im Vergleich:

| Feld | Quelle | Wann befüllt |
|---|---|---|
| **Beitrittsdatum** | Mitglied im Public-Form | Nur wenn das Feld unter *Formular-Felder* sichtbar ist *und* das Mitglied einen Wert einträgt |
| **Importiert am** | System | Sobald der Antrag in das Zielsystem übergeben wurde |
| **Aktiviert am** | System | Sobald der Antrag den Status *Aktiviert* erreicht |
| **Eingereicht am** | System | Sobald das Mitglied den Antrag abschickt |

> Wenn die Spalte **Beitrittsdatum** im Export leer bleibt, prüfe unter *Einstellungen → Formular-Felder*, ob das Feld auf *Ausgeblendet* steht. In dem Fall hat das Mitglied keinen Wert eintragen können. Wer als „Beitritt" eher das Datum der formellen Aktivierung im System meint, nimmt stattdessen **Aktiviert am**.

### Job-Übersicht

![Datenweiterleitung Jobs](images/admin-settings-datenweiterleitung-jobs.png)

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

![Export-Bereich](images/admin-settings-import-export-export.png)

Vier Sektionen sind einzeln oder als **Komplett-Bundle** exportierbar:

| Sektion | Inhalt |
|---|---|
| EEG-Einstellungen | Stammdaten, SEPA-Mandat-Settings, Aktivierungsmodus, Mitgliedsnummern-Startwert, Einleitungstext |
| Formular-Felder | Sichtbarkeit/Pflicht/Admin-only-Status aller konfigurierbaren Felder |
| Rechtsdokumente | Liste aller hinterlegten Dokumente mit Titel, URL und Zustimmungsmodus |
| Datenweiterleitungs-Konfig | Plugin-Konfigurationen für die Datenweiterleitung |

Dateiname enthält RC-Nummer und Zeitstempel — manuelle Versionierung in Git oder einem Backup-System ist damit unproblematisch.

### Import mit Diff-Preview

![Import-Bereich](images/admin-settings-import-export-import.png)

1. **Datei hochladen** (Drag-and-Drop oder Auswahldialog) — max 1 MB, nur `.json`. Die Datei wird serverseitig schemavalidiert; bei Fehlern wird der Upload abgelehnt.
2. **Diff-Preview** zeigt pro Sektion was sich ändert: hinzugefügt, modifiziert, entfernt oder unverändert.
3. **Sektionen aus-/abwählen** — nur ausgewählte werden tatsächlich angewendet.
4. **Apply** schreibt die Änderungen atomar (pro Sektion eine Transaktion). Apply ist **nicht** automatisch reversibel — daher der Hinweis oben, vorher die aktuelle Konfig zu exportieren.

> **Tipp:** Apply läuft mit einer `pg_advisory_xact_lock` — parallele Konfig-Änderungen über mehrere Browser-Tabs werden serialisiert, niemand überschreibt sich gegenseitig.
