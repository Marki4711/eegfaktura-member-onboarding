# Was ist neu?

Übersicht der Änderungen der letzten Tage, die für **EEG-Admins** und **Mitglieder** spürbar sind. Technische Details, Bug-Fixes ohne UI-Auswirkung und Infrastruktur-Themen sind hier bewusst weggelassen — die finden sich in der Commit-Historie.

---

## 2026-06-11

**EEG-Logo größer und rechts im Anmeldeformular-Kopf**

Auf der öffentlichen Mitglied-werden-Seite (`/register/<RC-Nummer>`) ist das EEG-Logo bisher klein (40 × 40 Pixel) links neben dem Schriftzug platziert gewesen — auf dem dunklen Header wirkte es verloren. Es sitzt jetzt prominent rechts ausgerückt und ist 64 × 64 Pixel groß. Der EEG-Name und „Mitglieder-Onboarding"-Untertitel bleiben links. EEGs ohne hinterlegtes Logo zeigen weiterhin das Standard-Blitz-Icon links beim Schriftzug.

**EEG-Kurzform in Listboxen und Antragsliste**

In allen drei Admin-Auswahllisten (EEG-Wechsel in den Einstellungen, EEG-Filter im Antragsbereich, Ziel-EEG-Auswahl beim Umordnen) sowie in der Spalte „EEG" der Antragsliste erscheint jetzt die in eegFaktura hinterlegte Kurzform der EEG (z. B. `EEG-Test`). In den Listboxen kombiniert mit der Referenznummer dahinter (Format `EEG-Test • RC0001`); in der Antragsliste nur die Kurzform, mit der Referenznummer als Tooltip beim Überfahren. In der Stammdaten-Card erscheint die Kurzform zusätzlich als eigene Read-Only-Zeile neben dem langen EEG-Namen.

Sortiert wird alphabetisch nach Kurzform — EEGs ohne Kurzform landen ans Listenende. Hat eine EEG in eegFaktura keine Kurzform hinterlegt, oder wurden die Stammdaten noch nicht synchronisiert, fällt die Anzeige automatisch auf die reine Referenznummer zurück. Der Sync läuft wie bisher manuell über den **Aktualisieren**-Knopf auf dem Stammdaten-Tab; eine Änderung der Kurzform in eegFaktura wird also erst nach dem nächsten Sync sichtbar.

**Erscheinungsbild der Online-Registrierung anpassbar**

Die öffentliche Mitglied-werden-Seite lässt sich jetzt pro EEG farblich an die eigene Marke anpassen. Auf dem Stammdaten-Tab (Modus „Alle Optionen") gibt es einen neuen Block „Erscheinungsbild der Online-Registrierung" mit drei Werkzeugen:

- Eine **Vorlagen-Listbox** mit **22 vorgefertigten Farb-Kombinationen** als Startwert aus drei etablierten Design-Systemen: zwölf **shadcn/ui**-Themes (Red/Orange/Yellow/Green/Blue/Violet jeweils Hell + Dunkel), sechs **IBM Carbon**-Themes (Blue/Teal/Magenta jeweils Hell + Dunkel) und die vier bisherigen **Dark-Mode-Klassiker** (Teal-Standard, Leaf, Sun, Slatey). Ein Klick lädt komplettes Farbschema + Schriftart in die Felder.
- **Acht einzelne Color-Picker** (Hauptfarbe, Text auf Hauptfarbe, Akzent, Text auf Akzent, Hintergrund, Text, Karten-Hintergrund, Text auf Karten) — entweder über die Picker-Quadrate oder per direkter HEX-Eingabe.
- Eine **Schriftart-Auswahl**: Sans-Serif (Inter), Serif (Georgia), Monospace (SF Mono) oder System-UI.

Eine **Live-Vorschau** unter den Pickern aktualisiert sich sofort beim Tippen. Ein **Kontrast-Panel** prüft die drei kritischen WCAG-AA-Paare (Hauptfarbe vs Text, Akzent vs Text, Text vs Hintergrund). Liegt ein Paar unter dem Mindest-Verhältnis 4,5:1, blockiert ein Sicherheitsfilter das Speichern — die Mitglied-werden-Seite würde sonst Texte zeigen, die schwer oder gar nicht lesbar sind. Die Vorschau rendert weiter, damit du auch im Fail-Zustand siehst, was du gerade kombinierst.

Zusätzlich werden auf der Mitglied-werden-Seite das **EEG-Logo** (aus eegFaktura synchronisiert) und der **EEG-Name** im Kopf eingebunden, der Footer zeigt dezent „Powered by eegFaktura". Wo das Branding bewusst nicht wirkt: die E-Mail-Bestätigungs-Seite (`/confirm-email`), der Admin-Bereich und die E-Mail-Templates.

Bestehende EEGs verändern sich nicht — sie behalten das Standard-Theme so lange, bis im Editor aktiv eine Vorlage gewählt oder eine Farbe gesetzt wird.

---

## 2026-06-10

**Antrag ablehnen ohne Mail an den Beitrittswerber**

Im „Antrag ablehnen"-Dialog gibt es jetzt eine Checkbox „Ablehnung per E-Mail an den Beitrittswerber übermitteln". Sie ist standardmäßig aktiv — das übliche Verhalten bleibt also gleich. Wenn ihr sie deaktiviert, wird die Ablehnungs-Mail nicht versendet (z. B. bei offensichtlichem Spam oder wenn das Mitglied telefonisch zurückgezogen hat). Die Begründung landet trotzdem im Statusverlauf, der Status springt normal auf „Abgelehnt". Bei „Info anfordern" gibt es bewusst keine Opt-out — ohne Mail an das Mitglied verliert dieser Status seinen Sinn.

**Reset-Dialoge: bessere Begründungs-Validierung**

Bei „Aktivierung zurücksetzen" und „Auf Prüfung zurücksetzen" blieb der Bestätigen-Button auch bei zu kurzer Begründung klickbar — beim Klick kam dann nur „Validation failed" als nichtssagende Meldung. Jetzt ist der Button erst aktiv, sobald die Mindestlänge erreicht ist (10 Zeichen für die zwei Reset-Pfade aus dem heutigen Vormittag). Falls doch ein Server-Validierungsfehler kommt, wird die genaue Feld-Meldung gezeigt statt eines generischen Texts.

**Datenweiterleitung: neue Spalte „Name"**

Im Spalten-Picker steht in der Kategorie „Stammdaten" jetzt ein einzelnes Feld „Name", das je nach Mitgliedstyp den passenden Wert liefert: bei Unternehmen, Vereinen und Gemeinden den Firmenname, bei Privatpersonen und Landwirten den Vornamen und Nachnamen kombiniert. Wer beides braucht — also Firmenname und Personennamen in getrennten Spalten — fügt die bestehenden Spalten „Vorname", „Nachname" und „Firmenname" zusätzlich hinzu wie bisher.

**Mitglied-werden-Formular: Reihenfolge der Titel-Felder**

Im „Persönliche Daten"-Block stehen Titel vor und Titel nach jetzt zusammen in einer Zeile ganz oben, darunter Vorname und Nachname. Vorher war Titel nach unter den Namen — die Reihenfolge ist jetzt logischer und auch im Admin-Edit-Dialog gleich.

**Mehr Platz für die Vorstands-Unterschrift im Beitrittserklärung-PDF**

Auf dem ausgedruckten PDF der Beitrittserklärung sitzt die Linie für Datum / Ort / Unterschrift Vorstand jetzt deutlich tiefer unter der Überschrift „Genehmigung durch den Vorstand" — genug Schreibhöhe für eine echte handschriftliche Unterschrift, ohne dass sich Tinte und Headline berühren.

**Layout-Politur: IBAN und Kontowortlaut auf einer Linie**

Im Bankverbindungs-Block des Public-Formulars saßen IBAN-Eingabefeld und Kontowortlaut-Eingabefeld minimal versetzt — der Hint-Popover beim Kontowortlaut-Label hat das verursacht. Beide Felder stehen jetzt pixelgenau auf gleicher Höhe.

**Neue Statusaktion „Aktivierung zurücksetzen"**

Im Status *Aktiviert* gibt es jetzt im Status-Block den Button „Aktivierung zurücksetzen". Damit kannst du einen Antrag zurück auf *Importiert* bringen — z. B. weil ein Mitglied versehentlich aktiviert wurde (manueller Klick oder Auto-Aktivierungs-Check trotz fehlender Core-Aktivität). Beim Reset wird ein gelber Warn-Banner eingeblendet, der dich daran erinnert, dass das Mitglied im eegFaktura-Core noch vorhanden sein kann und du dort separat den Status prüfen / korrigieren musst. Pflicht-Begründung von mindestens 10 Zeichen — landet im Statusverlauf mit dem Präfix `[reset-activation]`. Mitgliedsnummer und Core-Verknüpfung bleiben erhalten — es wird nur die Aktivierung selbst zurückgenommen.

**Neue Statusaktion „Auf Prüfung zurücksetzen"**

Im Status *Importiert* steht neben dem bestehenden Button „Import zurücksetzen" (der zurück auf *Genehmigt* führt) jetzt zusätzlich „Auf Prüfung zurücksetzen". Damit bringst du den Antrag in zwei Stufen zurück bis zu *In Prüfung* — und von dort über *Ablehnen* auf *Abgelehnt*. Geeignet wenn ein Antrag inhaltlich verworfen werden soll (Mitglied hat zurückgezogen, Daten-Qualität war ungenügend). Pflicht-Begründung mindestens 10 Zeichen, Präfix `[reset-to-review]` im Statusverlauf. Mitgliedsnummer und alle Import-Verknüpfungen werden genullt; das Mitglied bleibt aber im Core erhalten und muss dort gegebenenfalls separat gelöscht / deaktiviert werden — der Warn-Banner im Bestätigungs-Dialog erinnert daran.

> **Recovery-Pfad in voller Länge:** *Aktiviert* → „Aktivierung zurücksetzen" → *Importiert* → „Auf Prüfung zurücksetzen" → *In Prüfung* → „Ablehnen" → *Abgelehnt*. Jeder Schritt mit Pflicht-Begründung, jeder Schritt sichtbar im Statusverlauf der Detail-Ansicht.

> **Warntext bei „Import zurücksetzen" aktualisiert:** Der bisherige Hinweis „Anträge im Status 'Aktiviert' können nicht zurückgesetzt werden" passt nicht mehr — der Dialog verweist jetzt auf die neue Aktion „Aktivierung zurücksetzen" als ersten Schritt, falls du bei einem aktivierten Antrag gestartet bist.

---

## 2026-06-09

**Neues Toggle „Mitglied für Umstellung auf B2B vorbereiten"**

In der SEPA-Sektion des Bearbeiten-Dialogs gibt es bei Einzugsart „Core" einen neuen Toggle. Wenn aktiv, bekommt das Mitglied beim Import zwei Mandate: das übliche CORE-Mandat plus zusätzlich das B2B-Firmenlastschrift-Mandat zur Vorlage bei der Hausbank. Im eegFaktura-Core wird der Antrag weiterhin als CORE angelegt — sobald die Hausbank die B2B-Aktivierung bestätigt hat, stellst du den SEPA-Typ im Faktura-Core manuell um. Sichtbar in der Detail-Ansicht als Feld „B2B-Vorbereitung: Ja / Nein" (nicht in Listen/Übersichten).

> **Einstellungs-Tipp:** Der Toggle wirkt zusammen mit der EEG-Einstellung „SEPA-Mandat erst beim Import senden". Solange diese deaktiviert ist, wird das B2B-Mandat erst über „SEPA-Mandat erneut senden" zugestellt.

**Status „Warte auf Bank-Bestätigung" entfernt**

Der frühere Marker-Status für B2B-Anträge zwischen Import und Aktivierung entfällt. Anträge laufen jetzt direkt von „Importiert" auf „Bereit zur Aktivierung" — unabhängig vom Mandat-Typ. Bestand-Anträge im alten Status werden bei der Aktualisierung automatisch auf „Bereit zur Aktivierung" gesetzt; ein entsprechender Eintrag im Statusverlauf hält das Datum fest. Der „Bank-Bestätigung erhalten"-Button verschwindet aus der Detail-Ansicht.

**Bestand-Anträge mit Einzugsart „B2B" werden auf „Core" + Vorbereitungs-Toggle migriert**

Direkter Einzugsart-B2B-Pfad bleibt für Sonderfälle weiterhin im Bearbeiten-Dialog wählbar (z. B. wenn die Hausbank-Aktivierung schon erledigt ist und das Mitglied direkt B2B in den Core soll). Existierende Anträge werden mit dem Update automatisch auf das neue Modell (Core + Toggle=Ja) umgestellt.

---

## 2026-06-08

**Behoben: SEPA-Mandat-Mail bat trotz Audit-Trail um Unterschrift**

Wenn ihr den Toggle „Im CORE-Mandat den elektronischen Audit-Trail nutzen" oder „Im B2B-Mandat den elektronischen Audit-Trail nutzen" aktiviert hattet, wurde im PDF der Audit-Trail-Block korrekt gerendert (keine Unterschriftslinie). Die Mail an das Mitglied bat aber trotzdem darum, das PDF zu unterschreiben und zurückzusenden bzw. der Hausbank vorzulegen. Tester-Befund.

Ab jetzt spiegelt die Mail die PDF-Variante:

- Bei aktivem **CORE-Audit-Trail**: Mail sagt „Zustimmung wurde elektronisch dokumentiert, keine weitere Aktion nötig". Eine Ablage-Kopie geht an euch.
- Bei aktivem **B2B-Audit-Trail**: Mail sagt „elektronisch dokumentiert, keine Unterschrift nötig" — aber der Hinweis auf die Pre-Notification bei der Hausbank des Mitglieds bleibt erhalten (das ist SEPA-B2B-Regelwerk, unabhängig vom Mandat-Modus).
- Bei deaktiviertem Audit-Trail: heutige Klassik-Variante bleibt unverändert (Unterschrift + Rücksendung).

**USt-Pflicht-Status wird in der Antrags-Detail-Ansicht angezeigt**

Bei Anträgen vom Mitgliedstyp *Unternehmen* oder *Verein* siehst du jetzt in der Antrags-Detail-Ansicht direkt unter der UID-Nummer einen zusätzlichen Eintrag *„USt-pflichtig: Ja"* oder *„USt-pflichtig: Nein (Kleinunternehmerregelung)"*. Vorher musste der Status aus dem leeren UID-Feld abgeleitet werden — jetzt ist er sofort erkennbar, genau wie in der Bearbeiten-Maske.

**EEG-Stammdaten werden jetzt automatisch gespeichert**

Im Tab **Stammdaten & SEPA** gibt es keinen „Konfiguration speichern"-Button mehr. Jede Änderung wird nach einer halben Sekunde Tipp-Pause automatisch persistiert; oben in der Karte zeigt ein Status-Indikator „Speichert…" / „Gespeichert" — genau wie schon im Formular-Felder-Tab.

Wenn du einen Schalter aktivierst, der eine Folge-Eingabe verlangt (z. B. Genossenschaftsanteile aktivieren, ohne die Anteilswert-Eingabe), zeigt ein gelber Hinweis-Banner direkt unter dem Schalter, was noch fehlt: *„Änderungen werden gespeichert, sobald die folgenden Pflichtfelder ausgefüllt sind: …"*. Sobald du fertig bist, verschwindet der Banner und der Auto-Save speichert. Kein roter Fehler-Toast mehr.

Wenn du einen Schalter wieder ausschaltest, werden die Sub-Felder ausgeblendet und beim nächsten Auto-Save aus der DB geleert.

**EEG-Auswahl wird gemerkt**

Wenn ihr die Einstellungen für eine bestimmte EEG geöffnet habt und beim nächsten Besuch wieder auf `/admin/settings` springt, ist diese EEG direkt vorausgewählt — kein Sprung zurück auf die erste EEG der Liste. Funktioniert pro Browser. Wenn ihr für die zuletzt gewählte EEG keine Berechtigung mehr habt (Rollenwechsel, EEG entfernt), fällt die Auswahl still auf die erste verfügbare zurück.

**Behoben: Einstellungen im Formular-Tab waren nach Tab-Wechsel veraltet**

Wenn ihr in den EEG-Einstellungen ein Formular-Feld (Antragsteller oder Zählpunkt) umgestellt habt, wurde der neue Wert zwar sofort gespeichert. Wenn ihr aber auf einen anderen Tab gewechselt und wieder zurück zum Formular-Tab gesprungen seid, hat der Editor noch den alten Stand angezeigt. Erst ein Hard-Reload der Seite hat den aktuellen Stand gezeigt.

Ab jetzt bleibt die Anzeige nach Tab-Wechsel synchron mit dem gespeicherten Stand. Der Auto-Save selbst ist unverändert — der Bug betraf nur die Anzeige, nicht die Persistierung.

**Firmenlastschrift-Anträge werden im Faktura-Core zunächst als Basislastschrift angelegt**

Wenn ein Antrag mit Einzugsart *Firmenlastschrift (B2B)* importiert wird, legt das System ihn im eegFaktura-Core jetzt zunächst als *Basislastschrift (CORE)* an — nicht als B2B. Hintergrund: die B2B-Aktivierung verlangt eine separate Mandatsvereinbarung zwischen Mitglied und dessen Hausbank, die in der Praxis Tage bis Wochen dauert. Eine sofortige B2B-Abbuchung würde ohne Bank-Aktivierung abgelehnt; der CORE-Pfad überbrückt die Klärungs-Phase risikolos.

Die EEG-Kontaktperson bekommt in der Aktivierungs-Mail (Auto-Modus und Vorstands-Modus) einen gelben Hinweis-Block: *„Hinweis B2B-SEPA-Mandat — Der Antrag wurde mit Einzugsart Firmenlastschrift (B2B) angelegt, aber zur Sicherheit im eegFaktura-Core zunächst als Basislastschrift (CORE) importiert. Bitte vereinbaren Sie die Firmenlastschrift-Aktivierung eigenständig mit der Hausbank des Mitglieds. Sobald die Bank die B2B-Aktivierung bestätigt hat, ändern Sie den SEPA-Typ im eegFaktura-Core manuell auf B2B."*

Im Onboarding-Frontend bleibt die Einzugsart auf *Firmenlastschrift (B2B)* sichtbar. Der Antragsstatus läuft weiterhin über *„Auf Bank-Bestätigung warten"*. Anträge, die schon vorher als B2B in den Core importiert wurden, bleiben unangetastet — laufende B2B-Lastschriften mit aktivem Mandat werden nicht gestört.

**SEPA-Feld für ausgewählte Mitgliedstypen optional (Mitgliedstypen-Whitelist)**

Manche EEGs erzwingen SEPA-Lastschrift nicht für alle Mitgliedstypen. Im SEPA-Block der EEG-Einstellungen gibt es jetzt einen Schalter **„SEPA-Feld für ausgewählte Mitgliedstypen auf optional setzen"** mit darunterliegender Auswahl der berechtigten Mitgliedstypen (*Privat*, *Pauschalierter Landwirt*, *Verein*, *Gemeinde*, *Unternehmen*). Für die ausgewählten Mitgliedstypen wird die SEPA-Einwilligungs-Checkbox im Mitgliederformular optional — wenn das Mitglied sie weglässt, wird der Antrag mit „Kein SEPA" gespeichert, ohne Mandat-PDF.

Wichtig: **Bankdaten bleiben in jedem Fall Pflicht** — eegFaktura-Core verlangt sie für jedes Mitglied, unabhängig vom Mandat. Die EEG kann sie nach der Aktivierung für manuelle Zahlungsklärung nutzen.

Wenn ein Mitglied ohne SEPA-Mandat einreicht, bekommt die EEG in allen automatischen Info-Mails (Submit-Bestätigung, Aktivierungs-Mail, Beitrittserklärung an den Vorstand) einen gelben Hinweis-Banner: *„Kein SEPA-Lastschriftmandat erteilt — die Abrechnung muss über einen alternativen Zahlungsweg direkt mit dem Mitglied vereinbart werden."* Im Antrags-Detail siehst du über der Bankverbindungs-Karte denselben Hinweis als blauen Info-Streifen.

Auch Unternehmen sind wählbar — bei „Kein SEPA" gibt es ohnehin keine Lastschrift, also greift das SEPA-B2B-Regelwerk gar nicht. Die Abrechnung läuft dann manuell. Nur wenn das Unternehmen per Firmenlastschrift gezogen werden soll, bleibt das B2B-Mandat zwingend.

**SEPA-Einstellungen vereinfacht**

Der Schalter „SEPA-Mandat als Datei dem Antragsteller übermitteln" entfällt. Das System erzeugt jetzt für jedes Mitglied automatisch ein SEPA-Mandat-PDF. Zwei Schalter steuern weiterhin die Variante:

- **„Im CORE-Mandat den elektronischen Audit-Trail nutzen"** entscheidet, ob das PDF einen Audit-Trail-Block enthält (kein Rücksenden nötig) oder ein klassisches Unterschriftenfeld.
- **„SEPA-Mandat erst beim Import senden"** entscheidet, ob das PDF sofort mit der Bestätigungs-Mail rausgeht (mit Platzhalter „Mandatsreferenz wird von der EEG ausgefüllt" — die EEG trägt sie später händisch nach) oder erst beim Import (Mandatsreferenz = Mitgliedsnummer).

Wenn der Audit-Trail aktiv ist, wird das Mandat automatisch erst beim Import gesendet, weil das PDF beim Submit-Zeitpunkt noch keine Mitgliedsnummer hätte und damit unvollständig wäre.

Wenn der Audit-Trail aktiv ist, bekommt zusätzlich die EEG-Kontaktadresse eine Ablage-Kopie des Mandats — das Mitglied muss nichts zurücksenden, daher hat die EEG sonst kein Beleg-Exemplar.

EEGs, die bisher mit reiner Online-Zustimmung gearbeitet haben (ohne PDF-Versand), bekommen automatisch den Audit-Trail-Modus und das Import-Timing aktiviert. Damit bleibt das Mitglieder-Erlebnis gleich (keine Unterschrift nötig), und die EEG hat ein vollständig dokumentiertes Mandat.

**Bankverbindung: „Kontoinhaber:in" → „Kontowortlaut"**

Das Feld „Kontoinhaber:in" im Mitglieder-Formular heißt jetzt **„Kontowortlaut"** und hat einen erklärenden Hinweis-Text: bitte den exakten Wortlaut aus dem Konto-/Bankauszug eintragen — bei gemeinsamen Konten beide Namen. Hintergrund: bei gemeinsamen Haushaltskonten haben Mitglieder bisher oft nur den eigenen Namen eingetragen, was dazu führte, dass die Bank die SEPA-Lastschrift abgelehnt hat (Kontowortlaut stimmt nicht mit dem Konto überein).

---

## 2026-06-07

**Elektronisches SEPA-Mandat als Opt-in pro EEG (CORE + B2B getrennt)**

In den EEG-Einstellungen (Modus *Alle Optionen*) gibt es zwei neue Toggles:

- **Im CORE-Mandat den elektronischen Audit-Trail nutzen (statt manueller Unterschrift)** — sichtbar nur wenn die Basislastschrift im Onboarding aktiv ist.
- **Im B2B-Mandat den elektronischen Audit-Trail nutzen (statt manueller Unterschrift)** — immer sichtbar im Modus *Alle Optionen*, weil B2B-Mandate beim Import unabhängig erzeugt werden.

Mit diesen Toggles entscheidet die EEG selbst — und für CORE und B2B unabhängig voneinander —, ob das jeweilige Mandat-PDF mit dem elektronischen Audit-Trail-Block oder mit der klassischen Datum/Unterschrift-Zeile ausgestattet wird. **Standard ist „aus" für beide**: das PDF kommt klassisch mit Unterschriftslinie, das Mitglied unterschreibt physisch.

Hintergrund: Die Rechtsbewertung der elektronischen Willenserklärung (formfreie Willenserklärung gem. § 76 (3) EIWOG 2010) wird derzeit geklärt. Bis dahin soll keine EEG unbemerkt in die neue Variante laufen — wer den Audit-Trail bereits jetzt nutzen will, schaltet ihn pro Mandat-Typ bewusst ein. Die Trennung CORE vs. B2B berücksichtigt, dass für Geschäftsleute (Firmenlastschrift) eine andere Sorgfalt gilt als für Verbraucher.

**Was sich für Bestandsanträge ändert:** Beim erneuten Senden eines Mandats („SEPA-Mandat erneut senden" im Antragsdetail) oder bei der automatischen Mandat-Generierung beim Aktivieren übernimmt das System immer den aktuellen Stand der Toggles. Bereits an Mitglieder versandte PDFs bleiben unverändert.

Details und Erklärungen: siehe [Admin-Einstellungen → SEPA-Lastschriftmandat → Elektronisches SEPA-Mandat statt Datum/Unterschrift](06-admin-settings.md).

---

**SEPA-Firmenlastschrift-PDF mit elektronischem Audit-Trail**

Bei SEPA-Firmenlastschrift-Mandaten (`einzugsart=b2b`) erscheint im PDF nicht mehr ein leeres Datums- und Unterschriftsfeld, sondern ein Audit-Trail-Text, der die elektronische Zustimmung deines Mitglieds rechtskonform dokumentiert:

> Der Kunde hat der **\<EEG-Name\>** nach Verifizierung seiner E-Mail-Adresse am **\<Datum\>** **\<Uhrzeit\>** von der IP-Adresse **\<IP\>** auf elektronischem Weg (formfreie Willenserklärung gem. § 76 (3) EIWOG 2010) seine Zustimmung zum Vertrag im obigen Sinne sowie für das SEPA-Lastschriftmandat erteilt.

Das Mitglied muss das PDF dadurch **nicht mehr physisch unterschreiben** — die elektronische Zustimmung gilt als formfreie Willenserklärung gemäß EIWOG.

**Voraussetzungen für den Audit-Trail:**
- Der Antrag wurde nach dem 7. Juni 2026 eingereicht.
- Für Anträge aus deiner eigenen Website-Integration über die externe API: dein Backend muss die End-User-IP im neuen `submitterIp`-Body-Feld mitliefern (siehe API-Beschreibung im Settings-Bereich).

**Was passiert mit alten Anträgen?** Bestandsanträge (vor Juni 2026 eingereicht) erhalten weiterhin das klassische Mandat-PDF mit Datum/Unterschriftsfeld — sie wurden ohne IP-Erfassung gespeichert.

**Hinweis Core-Mandat:** Diese Änderung betrifft **nur** SEPA-Firmenlastschrift-Mandate (B2B). Das Basis-SEPA-Mandat für Privat-Mitglieder bleibt im klassischen Format mit Datum/Unterschriftsfeld.

**Vorstands-Genehmigungs-Workflow**

Neuer Toggle „Beitrittserklärung vom Vorstand genehmigen lassen, statt Beitrittsbestätigung automatisch zu versenden" in den EEG-Einstellungen (Alle Optionen). Wenn aktiv:

- Beim Wechsel auf „Aktiviert" geht eine Beitrittserklärung mit Vorstands-Signaturblock an den EEG-Kontakt statt einer Beitrittsbestätigung an das Mitglied.
- Der Vorstand unterschreibt das Dokument und leitet es per Hand an das Mitglied weiter.
- Das Mitglied bekommt die reguläre Aktivierungs-Mail aus eegFaktura und weiß so, dass sein Status auf „Aktiviert" steht — nur die zusätzliche Plattform-Mail entfällt.
- Im Antrags-Detail erscheint nach erfolgreicher Aktivierung ein blauer Hinweisblock plus ein Knopf „Beitrittserklärung herunterladen". Damit kannst du das PDF jederzeit neu erzeugen, z. B. wenn der Vorstand das Dokument verlegt hat.

**Hinweis:** Wenn der Toggle aktiv ist, muss die EEG-Kontakt-Mail gepflegt sein. Fehlt sie, bricht der Aktivierungs-Übergang ab — der Antrag bleibt im vorherigen Status, du siehst die Fehlermeldung direkt.

---

## 2026-06-01

**Stammdaten aktivierter Mitglieder mit eegFaktura abgleichen**

Im Antrags-Detail eines aktivierten Mitglieds stehen jetzt zwei neue Aktions-Knöpfe zur Verfügung:

- **„Stammdaten aus eegFaktura abgleichen"** pullt die aktuellen Mitglieder-Daten aus dem Kernsystem und überschreibt in der Onboarding-Kopie die Felder, in denen das Kernsystem inzwischen einen anderen Wert hält. Abgeglichen werden Name, Titel, UID-Nummer, Wohnort-Adresse, E-Mail, Telefon und Bankverbindung (IBAN, Kontoinhaber, Bankname). **Nicht** angefasst werden Mitgliedstyp, Geburtsdatum, Beitrittsdatum, Zählpunkte und Anteile-Anzahl. Ein Toast meldet die geänderten Felder, das Detail wird sofort neu geladen.
- **„SEPA-Mandat erneut senden"** erzeugt aus den aktuellen Onboarding-Werten ein frisches Mandat-PDF und schickt es an das Mitglied. Sinnvoll nach einem Abgleich, der eine geänderte IBAN oder einen geänderten Kontoinhaber gemeldet hat — das bestehende Mandat ist dann rechtlich nicht mehr gültig für die neue Bankverbindung. Mail-Subject und Wortlaut sind auf den Renewal-Kontext zugeschnitten („deine Bankverbindung wurde aktualisiert").

Beide Knöpfe sind unabhängig voneinander. Der Admin entscheidet selbst, ob nach einem Abgleich eine neue Mandat-Mail nötig ist.

**Bearbeiten-Knopf in nicht-editierbaren Status ausgeblendet**

Der „Bearbeiten"-Knopf erschien bislang in jedem Status — Klicks in z. B. `Aktiviert` führten dann beim Speichern zu einem 409-Fehler. Jetzt wird der Knopf nur noch in den Status angezeigt, in denen Bearbeiten technisch erlaubt ist (`Eingereicht`, `In Bearbeitung`, `Rückfrage`, `Genehmigt`, `Import fehlgeschlagen`). In `Aktiviert` übernehmen die beiden neuen Stammdaten-Abgleich-Knöpfe (siehe oben) die Datenaktualisierung.

→ [Statusverwaltung — Aktiviert: Stammdaten abgleichen + Mandat erneut senden](05-admin-status.md#activated--endzustand)

---

## 2026-05-31

**Automatischer Abgleich mit eegFaktura**

Sobald du dich als Admin anmeldest, prüft das Tool im Hintergrund einmal pro Tag und EEG, ob deine offenen Onboarding-Anträge bereits als Mitglieder im eegFaktura-Kernsystem existieren (verglichen werden nur IBAN UND E-Mail-Adresse — beide müssen übereinstimmen). Bei einem eindeutigen Treffer wird die Mitgliedsnummer aus eegFaktura mit dem Antrag verknüpft und im Antrags-Verlauf erscheint ein Hinweis „In eegFaktura erfasst (automatischer Abgleich)".

Damit schließt sich die Lücke, wenn du Onboarding-Daten nicht über den Import-Button überträgst, sondern manuell in eegFaktura eintippst — der Antrag im Onboarding zeigt trotzdem klar an, dass das Mitglied im Bestand existiert. Verarbeitungsdetails (welche Felder werden verglichen, was wird übernommen) stehen in der AVV.

**Excel-Download im Antragsdetail wieder ohne Bestätigungs-Dialog**

Der bisherige Bestätigungs-Dialog vor dem ersten „Excel herunterladen" und der automatische Vermerk „An eegFaktura übergeben am …" beim Excel-Download sind entfernt. Hintergrund: der automatische Abgleich (siehe oben) deckt zuverlässig auf, wenn Onboarding-Daten tatsächlich im Kernsystem landen. Der Vermerk beim Excel-Download produzierte zu oft False-Positives (Backup-Download, „mal sehen wie das aussieht"). Der Excel-Download ist jetzt wieder ein gewöhnlicher Download ohne Bestätigung; der „An eegFaktura übergeben am …"-Vermerk im Detail-Header wird nur noch beim echten Import oder beim automatischen Abgleich gesetzt.

→ [Admin-Einstellungen — Automatischer Abgleich mit eegFaktura](06-admin-settings.md#automatischer-abgleich-mit-eegfaktura)

---

**Neuer Umschalter „Einfache Ansicht" / „Alle Optionen" für die Einstellungsseite**

Pilot-Rückmeldung: für kleine Vereine ist die Vielzahl der Konfigurationsmöglichkeiten in den EEG-Einstellungen überwältigend. Daher gibt es jetzt einen **Ansichts-Umschalter** rechts oben in der Einstellungsseite:

- **Einfache Ansicht**: zeigt nur die ~5 wichtigsten Optionen, die alle EEGs brauchen (Registrierungs-Toggle, EEG-Stammdaten-Sync, SEPA-Master-Toggle, Einleitungstext, die vier wichtigsten Formular-Felder). Default für neu angelegte EEGs.
- **Alle Optionen**: zeigt die volle Konfiguration wie bisher. Default für bestehende EEGs (niemand wird überrascht). Hier liegen SEPA-B2B, Mandat-Timing, E-Mail-Bestätigung, Genossenschaftsanteile, Zählpunkt-Prefixes, Aktivierungs-Kriterium und alle Formular-Felder.

Der Umschalter speichert pro EEG. Hinterlegte Werte versteckter Sektionen bleiben in der Datenbank — sie wirken weiter im Mitgliederformular, sind nur nicht editierbar. Wenn in der Einfachen Ansicht eine erweiterte Option aktiv ist (z. B. SEPA-B2B), erscheint oberhalb der Tabs ein **gelber Hinweis-Banner** mit Button „Alle Optionen anzeigen" — damit keine versteckte Einstellung unbemerkt wirkt.

In der Einfachen Ansicht zeigt der Formular-Felder-Editor nur die drei im Catalog optional voreingestellten Felder (Telefon, Geburtsdatum, Teilnahmefaktor). Mit Alle Optionen die volle Liste (~27 Felder).

**Default-Änderung:** Bankname-Feld ist seit dieser Version `Ausgeblendet` per Default (statt `Optional`) — IBAN reicht für die meisten EEGs. Bestehende EEGs mit eigenem Eintrag bleiben unverändert. Wer den Bankname zeigen will, stellt das Feld auf `Optional` oder `Verpflichtend` (in **Alle Optionen** zu finden).

→ [Admin-Einstellungen — Einfache Ansicht oder Alle Optionen](06-admin-settings.md#einfache-ansicht-oder-alle-optionen-proj-67)

---

## 2026-05-30

**Vereinfachung: „Admin-Vorgabe"-Feldstatus bekommt kein Wert-Eingabefeld mehr**

Bisher zeigte der Feld-Konfigurations-Editor bei Auswahl von **„Admin-Vorgabe"** eine zusätzliche Eingabezeile, in die der Admin einen EEG-weiten Standardwert eintragen konnte; der Wert wurde dann beim Submit automatisch auf jeden neuen Antrag gesetzt. Das Feature wurde in der Praxis nicht genutzt und verwirrte mehr, als es half. Ab sofort ist der Status klar und reduziert:

- **Admin-Vorgabe** = Feld wird **nicht** im Mitglieder-Formular angezeigt, ist aber im **Admin-Edit-Dialog jedes Antrags** sichtbar und editierbar.
- Kein EEG-weiter Standardwert mehr — wenn ihr ein Feld systematisch befüllen wollt, müsst ihr es pro Antrag im Admin-Bereich eintragen.

Technisch entfernt: DB-Spalte `field_config.admin_value` (Migration 000058), Go-Funktion `applyAdminValues()`, das Eingabefeld in der Einstellungs-UI sowie das `adminValue`-JSON-Feld in `/api/admin/settings/fields`. Alte Konfigurations-Bundles aus der Konfig-Import/-Export-Funktion, die das Feld noch enthalten, werden weiterhin akzeptiert — der Wert wird beim Import still verworfen.

→ [Admin-Einstellungen — Formular-Felder](06-admin-settings.md#formular-felder-zahlpunktfelder)

**Feature: Automatisches Speichern in den Formular-Felder + Tab-Wechsel-Schutz**

In den Einstellungen waren bisher manche Editoren mit einem „Speichern"-Button versehen, andere nicht — wer den Button vergaß, riskierte Datenverlust beim Tab-Wechsel. Das ist jetzt aufgeräumt:

- **Formular-Felder** (Tab „Formular-Felder"): Speichern-Button entfernt. Jede Toggle-Änderung wird nach kurzer Verzögerung automatisch gespeichert. Ein Status-Indikator oben in der Karte zeigt „Speichert…" / „Gespeichert".
- **Einleitungstext** (Tiptap): Speichern-Button bleibt — der Admin entscheidet bewusst, wann der Text „fertig" ist. Zusätzlich läuft ein 30-Sekunden-Auto-Speichern im Hintergrund als Sicherheitsnetz gegen Browser-Crash oder versehentliches Schließen.
- **Stammdaten & SEPA**: Speichern-Button bleibt unverändert — die ~30 zusammenhängenden Felder sollen weiterhin als ein bewusster „Konfiguration speichern"-Klick abgesetzt werden (z. B. wegen gegenseitiger Validierung von SEPA-Toggle + Mandat-Timing).
- **Allgemein:** Wenn du einen Tab oder die EEG wechselst, **während es ungespeicherte Änderungen gibt**, erscheint jetzt ein Confirm-Dialog („Hier bleiben" / „Verwerfen und wechseln"). Tabs mit ungespeicherten Änderungen tragen ein orangenes Punkt-Symbol. Zusätzlich warnt der Browser beim Schließen des Tabs/Refreshs.

→ [Admin-Einstellungen](06-admin-settings.md)

**Doku-Schärfung: Was das Onboarding ist — und was nicht**

Auf Tester-Nachfrage „Wofür ist Member-Onboarding eigentlich gedacht?" steht jetzt in der Doku-Übersicht eine klare Abgrenzung: Das Tool ist für **Datenerfassung, Antragsprüfung und Übergabe** an ein Zielsystem (typischerweise eegFaktura) gedacht. Es ist explizit **keine** Mitgliederverwaltung, **kein** dauerhafter Datenspeicher und **kein** Reporting-Tool. Auswertungen und das laufende Mitglieder-Geschäft gehören ins Zielsystem; das Onboarding liefert dorthin via direktem Import oder Plugin-Datenweiterleitung (heute: Excel/CSV; künftig: CRM-Anbindungen). Zusätzlich aufgenommen: konkreter Praxis-Nutzen aus Pilot-Rückmeldungen — „neue Mitglieder über eine einfache Formularmaske sauber in eegFaktura bekommen, ohne die typischen Fehlerquellen der manuellen Aufnahme".

→ [Überblick](index.md)

**Geplant: Basic-/Advanced-Modus für Einstellungen**

Pilot-Rückmeldung: Die Menge an Konfigurations-Optionen überfordert kleine EEGs, die im Wesentlichen „Antragsdaten erfassen + an eegFaktura übergeben" wollen. Geplant ist daher ein **Toggle „Basic / Erweitert"** am Seitenkopf der Einstellungen: Basic zeigt nur die für den 80%-Use-Case relevanten Optionen (Registrierung aktiv, Stammdaten, einfaches SEPA, Einleitungstext, häufigste Formular-Felder, Rechtsdokumente, Datenweiterleitung), Erweitert blendet alles ein (heutiges Verhalten). Default für neue EEGs: Basic; bestehende EEGs behalten Advanced. Sobald implementiert, spiegelt sich die Klassifizierung auch in dieser Doku wider (z. B. Sektionen mit „(Erweitert)"-Marker, neuer Abschnitt „Welcher Modus passt zu mir?").

**Doku-Schärfung: SEPA-Toggle-Kombinationen klarer erklärt**

Im Stammdaten-&-SEPA-Tab gibt es zwei zusammenhängende Toggles („SEPA-Mandat von der EEG bereitstellen" + „SEPA-Mandat erst beim Import erzeugen"), die in Kombination vier verschiedene Verhaltensvarianten ergeben. Bisher musste man die Kombinationen herausfinden — jetzt steht in der Doku eine Übersichts-Tabelle: welche Toggle-Kombi → was sieht das Mitglied im Formular, wann kommt das Mandats-PDF, welche Mandatsreferenz wird verwendet. Außerdem explizit aufgenommen: Admin-eingetragene Mandatsreferenzen im Antrags-Edit haben Vorrang vor der Auto-Ableitung und werden beim Import an eegFaktura mit übergeben.

→ [Admin-Einstellungen — SEPA-Lastschriftmandat](06-admin-settings.md#welche-toggle-kombination-ergibt-was)

**Doku-Ergänzung: Praxis-Hinweise zur „Netzbetreiber-Vollmacht"**

Aus einer Tester-Runde kamen konkrete Erfahrungswerte, wann das Vollmachts-Feld sinnvoll ist und wann nicht: Bei **Netz OÖ** beschleunigt sie die Kommunikation, wenn das Mitglied beim Online-Portal-Onboarding klemmt — die Netz OÖ verlangt für ein Handeln im Mitglieds-Auftrag genau diese Vollmacht. Bei **Salzburg Netz** läuft es anders herum: dort braucht es **Kundennummer + Vertragskontonummer**, die das Mitglied selbst aus seinem Portal heraussucht — ein Vollmachts-Workflow ist nicht vorgesehen, der Toggle kann ausgeblendet bleiben. Faustregel ergänzt: vor dem Aktivieren beim Netzbetreiber kurz nachfragen.

→ [Admin-Einstellungen — Spezielle konfigurierbare Felder](06-admin-settings.md#spezielle-konfigurierbare-felder)

---

## 2026-05-29

**Feature: Excel-Download zählt jetzt auch als Übergabe an eegFaktura** *(am 2026-05-31 wieder zurückgenommen, siehe oben)*

Das xlsx aus „Excel herunterladen" entspricht 1:1 dem eegFaktura-Import-Template und kann von Admins direkt im Core hochgeladen werden. Der **erste** Klick auf „Excel herunterladen" markiert den Antrag jetzt als an eegFaktura übergeben; vor dem ersten Download erscheint ein Bestätigungs-Dialog. Weitere Downloads ändern den Vermerk nicht und lösen auch keinen Dialog mehr aus. Wer das xlsx nur als Backup oder für die Buchhaltungs-Ablage braucht, nutzt stattdessen den Button **„Datenweiterleitung"** mit frei konfigurierbaren Excel-Formaten. Im Detail-Header eines bereits übergebenen Antrags steht ab sofort eine Zeile „An eegFaktura übergeben am …".

**Feature: Genehmigte Anträge können jetzt abgelehnt werden**

Nach einem „Import zurücksetzen" landet ein Antrag wieder auf „Genehmigt". Bisher gab es in diesem Status keine Möglichkeit mehr, den Antrag final abzulehnen — der „Ablehnen"-Button fehlte. Jetzt erscheint er neben „In eegFaktura importieren" und „Manuell aktivieren …". Beim Klick muss eine Begründung eingegeben werden (wie bei jeder Ablehnung), und der Antrag landet auf „Abgelehnt". Die Mitgliedsnummer wurde bereits beim Reset-Import entfernt — keine separate Aktion nötig.

→ [Antrag ablehnen](05-admin-status.md)

---

## 2026-05-28

**Bug-Fix: SEPA-Mandatsreferenz + Mandatsdatum landen jetzt im Core**

Bei Anträgen, deren EEG das SEPA-Mandat **erst beim Import** versendet (Firmenlastschrift oder Privat-Mandat mit Option „Mandat bei Import"), hat das Onboarding bisher die Mandatsreferenz (= Mitgliedsnummer) und das Mandatsdatum (= Import-Tag) nur für das lokale PDF abgeleitet — im eegFaktura-Core blieben beide Felder leer und mussten von der Admin händisch nachgetragen werden. Behoben: die Werte werden jetzt VOR dem Core-POST persistiert und in das Payload-Feld `accountInfo` mitgesendet. Bestandsanträge VOR diesem Fix tragen die Werte zwar onboarding-seitig korrekt, im Core fehlen sie aber weiterhin — entweder via „Import zurücksetzen + neu importieren" überschreiben oder einmalig manuell im Core eintragen.

**Bug-Fix: „Zusatzangaben" lassen sich jetzt als Admin bearbeiten**

Im Admin-Edit-Form fehlten bisher die Eingabefelder für Beitrittsdatum, Personen im Haushalt, Wärmepumpe, Warmwasser elektrisch, E-Auto (inkl. Anzahl + Jahres-Kilometer), Genossenschaftsanteile und die Netzbetreiber-Vollmacht — sie waren nur in der Detail-Ansicht sichtbar. Jetzt gibt es eine eigene „Zusatzangaben"-Section zwischen Adresse und Zählpunkten. Welche Felder erscheinen, hängt von der EEG-Field-Config in den Einstellungen ab: Felder auf „Optional", „Pflicht" oder „Nur Admin" werden gerendert, Felder auf „Ausgeblendet" nicht. So bleibt das Verhalten konsistent mit dem Mitglieder-Formular. Wenn der E-Auto-Toggle deaktiviert wird, werden Anzahl und Jahres-Kilometer beim Speichern automatisch geleert. Der Zeitstempel der Netzbetreiber-Vollmacht wird beim ersten Setzen serverseitig vergeben und aus Audit-Gründen nicht entfernt.

→ [Antrag bearbeiten](04-admin-applications.md#antrag-bearbeiten)

**Bug-Fix: Aktivierungs-Mail erwähnte das Mandatsformular auch bei reiner Online-Zustimmung**

Mitglieder, die online der SEPA-Lastschrift zugestimmt hatten (ohne dass die EEG ein Papier-Mandat anbietet), bekamen mit der Beitrittsbestätigungs-Mail einen Hinweis-Block „SEPA-Lastschriftmandat — Mandatsreferenz: … bitte ergänze diese auf dem Mandatsformular". Es gibt aber gar kein Formular zum Ergänzen. Der Hinweis erscheint jetzt nur noch dann, wenn die EEG tatsächlich ein Mandat-PDF beim Submit verschickt (also bei `sepa_mandate_enabled = true` und vor dem Import).

→ [Beitrittsbestätigung](07-emails-and-pdfs.md)

---

## 2026-05-27

**Bug-Fix: Konfigurations-Import zeigt jetzt sofort die übernommenen Werte**

Beim Import einer Konfigurations-Datei (Stammdaten + Formular-Felder + Rechtsdokumente + Datenweiterleitung) zeigten die Tabs der Settings-Seite gelegentlich noch den Stand **vor** dem Import — vor allem im Tab „Formular-Felder", wenn dieser schon vorher geöffnet war. Tatsächlich war der Import erfolgreich in der Datenbank, nur das UI hatte den alten Stand zwischengespeichert. Behoben: nach einem erfolgreichen Apply werden alle Tabs der Settings-Seite automatisch neu geladen, ohne Browser-Refresh.

→ [Admin-Einstellungen — Konfiguration Import / Export](06-admin-settings.md#konfiguration-import-export)

**Bug-Fix: Antragsdetails verschwanden nach Admin-Bearbeitung**

Wenn ein Admin einen bestehenden Antrag im Edit-Form geöffnet und gespeichert hat, gingen die Zählpunkt-spezifischen Detailfelder (Verbrauch Vorjahr/Prognose, Einspeisung Prognose, PV-Leistung, Einspeisebegrenzung, Speichersteuerung-Frage, Wechselrichter-Hersteller + Leistung, Speichergröße) stillschweigend verloren — auch wenn der Admin diese Felder gar nicht angefasst hat. Behoben: die Felder bleiben jetzt bei jedem Save erhalten. Bestandsanträge, bei denen das Feld nach einer früheren Speicherung leer ist, müssen einmal manuell erneut eingegeben werden.

**Bug-Fix: Beitrittsbestätigungs-Mail fehlte bei automatischer Aktivierung**

Wenn ein Antrag über den **Aktivierungs-Check-Batch** (automatisch durch das Netzbetreiber-Routing) auf den Status „Aktiviert" wechselte, blieb die Beitrittsbestätigungs-Mail mit PDF aus. Nur der manuelle Admin-Klick auf „Aktivieren" hat sie verschickt. Behoben: beide Pfade verschicken jetzt die Mail. Wiederholungs-Check für die EEG zur Sicherheit: bereits aktivierte Mitglieder ohne empfangene Mail können über erneuten „Aktivieren"-Trigger einmalig nachgezogen werden — der Resend wird durch das Flag `activation_notification_sent_at` blockiert, sodass kein doppelter Versand passiert (das Flag muss in Ausnahmefällen manuell zurückgesetzt werden, sprich uns an).

**Bug-Fix: PDF zeigte „SEPA-Ermächtigung: Per E-Mail" trotz Online-Zustimmung**

Wenn die EEG kein zusätzliches SEPA-Mandat-PDF im Onboarding einbindet, der Antragsteller aber im Formular die SEPA-Lastschrift online angehakt hat, zeigte die Beitrittsbestätigung im Feld „SEPA-Ermächtigung" fälschlich „Per E-Mail". Steht jetzt korrekt **„Online-Zustimmung erteilt"**. Die Werte für die anderen SEPA-Varianten (Basislastschrift, Firmenlastschrift, Kein SEPA) sind unverändert.

→ [Anträge verwalten](04-admin-applications.md)

---

## 2026-05-26

**Bug-Fix: Admin-Detail zeigt jetzt auch die Zusatzangaben**

Im Antrags-Detail im Admin-Tool gab es bisher keine Anzeige für die konfigurierbaren Zusatzangaben (Beitrittsdatum, Personen im Haushalt, Wärmepumpe, E-Auto inkl. Anzahl + Jahres-km, Warmwasser elektrisch). Das Mitglied bekam diese Werte in seiner Submit-Bestätigungs-Mail aufgelistet, der Admin sah sie aber nirgends. Neue Karte „Zusatzangaben" rendert genau die Felder, die tatsächlich Werte enthalten — bleibt also unsichtbar für EEGs, die diese Felder nicht aktiviert haben.

→ [Anträge verwalten — Antrags-Detail](04-admin-applications.md)

**Bug-Fix: Resend-Bestätigungsmail enthält wieder alle Antragsdetails**

Der Admin-Button „Bestätigung erneut senden" verschickte eine stark reduzierte Mail: nur Name und EEG-Daten, ohne Zählpunkte, Adresse, Bankverbindung, Mitgliedstyp-Details. Behoben: die Resend-Mail ist jetzt inhaltsgleich zur initialen Submit-Mail (mit Ausnahme des SEPA-PDFs, das aus rechtlichen Gründen nicht neu erzeugt wird — die original-generierte Mandatsreferenz bleibt gültig).

**Bug-Fix: Beitrittsbestätigung-Download blieb nach Aktivierung verfügbar**

Sobald ein Mitglied in den Status „importiert" oder „aktiviert" wechselte, verschwand der Download-Button für die Beitrittsbestätigungs-PDF (und für den Excel-Export). Der Statusausbau für die Post-Import-Stati hat damals die Allow-List vergessen. Beide Downloads bleiben jetzt in allen Post-Approval-Status verfügbar — Admins ziehen den Excel-Export typischerweise erst nach dem Import zur Ablage.

→ [Anträge verwalten — Antrags-Detail](04-admin-applications.md)

---

## 2026-05-25

**Bug-Fix: EEG-Name in Stammdaten zeigt die Klar-Bezeichnung**

Im Admin-Bereich **Stammdaten** stand im Feld „EEG-Name" bisher der kurze interne Handle aus eegFaktura (z. B. `EEG-TEST`) statt der beschreibenden Bezeichnung (`Testenergiegemeinschaft EEG 1234`). Ursache: der Sync hat das falsche eegFaktura-Feld gelesen — den internen Handle statt der Beschreibung.

Behoben: Klick auf **„Aus eegFaktura aktualisieren"** zieht jetzt die Klar-Bezeichnung. Bestandsdaten werden beim nächsten Klick automatisch überschrieben.

→ [Admin-Einstellungen — Stammdaten](06-admin-settings.md)

**Bug-Fix: Tarif und Netzbetreiber fehlen nach Import im eegFaktura-Core**

Beim Import eines Mitglieds in den eegFaktura-Core wurden zwei Felder pro Zählpunkt nicht korrekt übergeben — mit dem Ergebnis, dass importierte Mitglieder im Core ohne den im Onboarding ausgewählten **Meter-Tarif** und ohne **Netzbetreiber-Zuordnung** angelegt wurden. Der Admin musste beides nachträglich im Core-UI manuell ergänzen.

Behoben:

- **Meter-Tarif** wird jetzt korrekt mitgesendet und pro Zählpunkt im Core gesetzt.
- **Netzbetreiber-ID** wird aus der Zählpunktnummer abgeleitet (E-Control-Standard: `AT` + 6-stelliger Netzbetreiber-Code, z. B. `AT003000` = Netz Oberösterreich). **Netzbetreiber-Name** wird zusätzlich aus dem Core-Stamm aufgelöst.

**Wichtig für BEGs (Bürgerenergiegemeinschaften):** jeder Zählpunkt wird unabhängig aufgelöst — Zählpunkte aus mehreren Netzgebieten in einer Mitgliedschaft funktionieren ohne Sonderkonfiguration.

→ Bereits importierte Mitglieder sind nicht betroffen; der Fix wirkt nur auf Neu-Importe. Bestehende Einträge ohne Tarif/Netzbetreiber im Core bleiben unverändert und müssen weiterhin manuell nachgezogen werden (oder per `/reset-import` → erneuter Import).

---

## 2026-05-24

**Saubere Erfassung: „Umsatzsteuerpflichtig?"-Checkbox bei Unternehmen + Vereinen**

Bei den Mitgliedstypen **Unternehmen** und **Verein** gibt es jetzt eine explizite Checkbox „… ist umsatzsteuerpflichtig (Regelbesteuerung)". Das UID-Eingabefeld erscheint nur, wenn das Häkchen gesetzt ist. So können auch Kleinunternehmer, die zufällig eine UID besitzen (z. B. für innergemeinschaftliche Erwerbe), nicht aus Reflex eine UID eintragen und damit fälschlich als regelbesteuert eingestuft werden. **Gemeinden** bekommen den Toggle bewusst nicht — dort wird die USt-Differenzierung über die Zählpunkte (Hoheitsbereich vs. Betrieb gewerblicher Art) abgewickelt.

→ [Mitglieder-Registrierung — Mitgliedstyp & USt-Pflicht](02-member-registration.md#schritt-2-mitgliedstyp-auswahlen)

**Mitgliedstypen vereinfacht — Kleinunternehmer entfällt als eigener Typ**

Der Mitgliedstyp **Kleinunternehmer** existiert nicht mehr als eigene Auswahl. Stattdessen wählt das Mitglied **Unternehmen** und lässt die UID-Nummer leer — das signalisiert dem System automatisch die Kleinunternehmerregelung (§ 6 Abs 1 Z 27 UStG, 0 % USt.). Mit ausgefüllter UID greift der reguläre Unternehmens-Pfad (20 % USt.). Bestehende Anträge mit altem Typ wurden automatisch auf `company` migriert.

→ [Mitglieder-Registrierung — Mitgliedstyp auswählen](02-member-registration.md#schritt-2-mitgliedstyp-auswahlen)

**USt.-Hinweise aus dem Dropdown entfernt**

Die USt.-Sätze stehen nicht mehr in den Optionen des Mitgliedstyp-Dropdowns — sie waren bei Misch-Typen (Gemeinde, Verein, Unternehmen mit/ohne Kleinunternehmerregelung) ohnehin nur ein grober Hinweis. Die tatsächliche umsatzsteuerliche Einordnung ergibt sich aus den Folgefeldern (z. B. UID-Nummer bei Unternehmen) und wird zwischen Mitglied und EEG geklärt.

**Neue Funktion: Konfiguration sichern und übertragen**

Du kannst die komplette Konfiguration einer EEG (Stammdaten-Settings, Formular-Felder, Rechtsdokumente, Datenweiterleitungs-Configs) als **JSON-Datei** sichern und auf andere EEGs übertragen. Nützlich um:

- mehrere EEGs auf eine gemeinsame Grund-Konfiguration zu bringen,
- vor einem riskanten Änderungs-Schub den Ist-Zustand zu sichern (Apply ist nicht automatisch reversibel),
- Konfigurations-Stände nachvollziehbar in Git oder einem Backup-System zu halten.

Beim Import zeigt eine **Diff-Vorschau** pro Sektion was sich ändert (hinzugefügt / modifiziert / entfernt / unverändert). Du kannst einzelne Sektionen aus- oder abwählen.

→ [Admin-Einstellungen — Konfiguration Import / Export](06-admin-settings.md#konfiguration-import-export)

**Bug-Fix: Antrag für Vereine/Unternehmen blockiert wegen fehlendem Geburtsdatum**

Wenn eine EEG das Feld **Geburtsdatum** als Pflichtfeld konfiguriert hatte, scheiterten Vereins- und Unternehmens-Anträge mit „Geburtsdatum ist erforderlich" — obwohl das Feld im Formular für diese Typen gar nicht angezeigt wird. Behoben: der Pflicht-Check gilt jetzt nur noch für natürliche Personen (Privatperson, Landwirt). Analoge Korrektur für UID-Nummer (nur für Unternehmen, Gemeinde, Verein verlangt).

---

## 2026-05-23

**Neue Funktion: Datenweiterleitung an externe Systeme**

Mit der **Datenweiterleitung** kannst du Antragsdaten an externe Systeme weitergeben — aktuell als **Excel/CSV-Export** mit konfigurierbarem Feldsatz. Pro EEG legst du fest, welche Felder enthalten sind, in welcher Reihenfolge sie stehen und mit welcher Spaltenüberschrift sie erscheinen.

Auslösen kannst du eine Weiterleitung entweder:

- **aus der Antragsliste** per Bulk-Aktion auf mehrere ausgewählte Anträge,
- **aus dem Antragsdetail** für einen einzelnen Antrag.

Jobs laufen asynchron im Hintergrund — eine Übersicht aller Läufe inkl. Download der Ergebnisdatei und Fehler-Diagnose findet sich im Job-Tab. Bei sensiblen Feldern (IBAN, Geburtsdatum) zeigt die UI eine DSGVO-Warnung. Weitere Plugins (Zoho, HubSpot, …) sind als Folge-Phasen geplant.

→ [Admin-Einstellungen — Datenweiterleitung](06-admin-settings.md#datenweiterleitung)
→ [Anträge verwalten — Massenaktionen](04-admin-applications.md#massenaktionen)

---

## 2026-05-21

**Ansprechperson für Organisationen**

Bei Unternehmen, Gemeinden und Vereinen kannst du jetzt eine **Ansprechperson** erfassen — Name, E-Mail und Telefon der konkreten Kontaktperson der Organisation. Wird im Formular per Master-Checkbox ein-/ausgeblendet, damit der Block nur erscheint wenn relevant.

→ [Mitglieder-Registrierung — Persönliche Daten](02-member-registration.md#schritt-3-personliche-daten-eingeben)

**Abweichende Rechnungs-E-Mail für Organisationen**

Bei Org-Mitgliedstypen kann jetzt zusätzlich eine separate **Rechnungs-E-Mail** angegeben werden — unabhängig von der allgemeinen Kontakt-Adresse. Im Admin-Field-Editor zeigen Badges (`+Ansprechperson`, `+Rechnungs-E-Mail`), unter welcher Bedingung das jeweilige Feld im Formular wirklich greift.

→ [Admin-Einstellungen — Formular-Felder](06-admin-settings.md#formular-felder-zahlpunktfelder)

**Netzbetreiber-Info-Seite im Beitrittsbestätigungs-PDF**

Wenn deine EEG die Netzbetreiber-Vollmacht aktiviert hat, kannst du jetzt Netzbetreiber-spezifische Hinweise (Anleitungs-Texte, Kontaktdaten, Portal-URLs) in den Einstellungen hinterlegen — diese erscheinen automatisch auf einer zusätzlichen Seite des Beitrittsbestätigungs-PDFs, das das Mitglied nach Annahme bekommt. Außerdem zwei neue konfigurierbare Felder im Mitgliedsformular:

- **Netzbetreiber Kundennummer** — die Vertragsnummer des Mitglieds beim Netzbetreiber
- **Inventarnummer eines Zählers** — eine eindeutige Kennung am Zähler

Beide nur sichtbar wenn die Netzbetreiber-Vollmacht aktiviert ist (Badge `+Vollmacht` im Admin-Editor).

→ [Admin-Einstellungen — Spezielle konfigurierbare Felder](06-admin-settings.md#spezielle-konfigurierbare-felder)
→ [E-Mails & PDFs — Beitrittsbestätigung](07-emails-and-pdfs.md)

**Hilfetexte am Mitgliedsformular**

- Neuer Hilfetext zur **UID-Nummer** im öffentlichen Registrierungsformular
- Hilfetext mit USt-Erklärung pro Mitgliedstyp
- „Titel" umbenannt in „Titel vor" (Symmetrie zu „Titel nach")
- Hilfetexte für „Titel vor" und „Titel nach"

---

## 2026-05-20

**Mitglieder-Registrierung: UID-Nummer für Vereine jetzt optional**

Bisher wurde die UID-Nummer bei Vereinen vom Validator als Pflichtfeld behandelt — fachlich falsch, da nicht jeder Verein eine UID hat. Das Feld ist jetzt optional und kann leer gelassen werden.

---

## 2026-05-19

**Aktivierungs-Modus pro EEG konfigurierbar**

Steuert, wann ein Antrag automatisch von „Bereit zur Aktivierung" auf „Aktiviert" wechselt. Zwei Varianten:

- **Variante A (Default):** Mitglied im eegFaktura-Core hat Status `ACTIVE` — klassisches Verhalten.
- **Variante B:** Mindestens ein Zählpunkt im Core hat eine laufende Netzbetreiber-Anmeldung. Aktiviert das Mitglied bereits sobald die EDA-Meldung beim Netzbetreiber gestartet ist, ohne den Abschluss abzuwarten.

Beim Übergang auf „Aktiviert" wird die volle Beitrittsbestätigungs-Mail mit PDF versandt.

→ [Admin-Einstellungen — Aktivierungs-Kriterium](06-admin-settings.md#aktivierungs-kriterium)

**Manueller Skip-Import — Ausnahmefall `approved → activated`**

Falls ein Mitglied im eegFaktura-Core bereits manuell angelegt/überschrieben wurde und der reguläre Import-Pfad übersprungen werden soll, gibt es jetzt im Detail-View den Button **„Manuell aktivieren …"** als Ausnahmefall.

→ [Statusverwaltung — Ausnahmefall: approved → activated](05-admin-status.md#ausnahmefall-approved-activated-manueller-skip-import)

**Teilnahmefaktor pro EEG konfigurierbar**

Pro EEG steuerst du jetzt ob der **Teilnahmefaktor (%)** im Mitgliedsformular sichtbar ist, ob ihn das Mitglied ändern darf oder ob er fest auf 100 % steht. Bei Hidden/Admin-Vorbefüllung wird der Wert serverseitig automatisch auf 100 % gesetzt — Beitrittsbestätigung, Mail und Excel-Export zeigen ihn unverändert.

→ [Admin-Einstellungen — Formular-Felder](06-admin-settings.md#formular-felder-zahlpunktfelder)

---

## 2026-05-18

**SEPA-Mandat-Datum auf Übermittlungstag vorbefüllt**

Beide SEPA-Mandate (Basislastschrift CORE und B2B-Firmenlastschrift) zeigen im Unterschriftsfeld jetzt das **Datum der Übermittlung** vorbefüllt. Das Mitglied trägt nur noch Ort + Unterschrift ein. Das Datum wird im Antrags-Detail unter „Mandatsdatum" angezeigt und beim Faktura-Import als Mandate-Date mitgeführt.

**Zählpunkt-Prefix pro EEG konfigurierbar**

Wenn die Zählpunkte deiner EEG mehrheitlich vom selben Netzbetreiber + PLZ-Bereich kommen, kannst du in den Einstellungen einen **festen Prefix** pro Richtung (Verbraucher / Einspeisung) hinterlegen. Das Mitglied tippt dann nur noch die individuellen letzten Stellen — der Prefix ist gelockt und kann nicht überschrieben werden. Beim Verlassen des Eingabefelds werden fehlende Stellen automatisch mit führenden Nullen ergänzt.

→ [Admin-Einstellungen — Zählpunkt-Prefixes](06-admin-settings.md#zahlpunkt-prefixes)
→ [Mitglieder-Registrierung — Zählpunkte](02-member-registration.md#schritt-5-zahlpunkte-angeben)

**Zählpunkt-Format 2-6-5-20 in PDF + Mail**

Die Zählpunkt-Nummer wird jetzt überall in der offiziellen E-Control-Gruppierung `AT 003100 00000 12345678901234567890` angezeigt (PDFs, Bestätigungs-E-Mails, Admin-Detail-View).

**Bankname als konfigurierbares Feld pro EEG**

Bisher war der Bankname fest sichtbar im Bankverbindungs-Block. Jetzt steuerst du pro EEG ob er ausgeblendet, optional oder Pflicht ist. Default: Optional (bewahrt heutiges Verhalten).

**Firmenbuchnummer optional für Unternehmen**

Bisher Pflichtfeld bei `memberType=company`, jetzt durchgehend optional (auch wenn EEG-Feld-Config sie als „Pflicht" markiert hat).

---

## 2026-05-17

**Erzeugungsform + Batterie-Felder**

Erzeuger-Zählpunkte fragen jetzt die **Erzeugungsform** (PV / Wasser / Wind / Biomasse) ab und — bei PV — optional Batteriespeicher-Daten (Größe, Wechselrichter-Hersteller, Speichersteuerungs-Einverständnis). Alle Felder sind pro EEG konfigurierbar; die Sichtbarkeit ist typabhängig und wird im Admin-Editor mit farbigen Badges (`[Verbraucher]`, `[Einspeisung]`, `[PV]`, `[+Speicher]`, etc.) sofort sichtbar gemacht.

→ [Mitglieder-Registrierung — Zählpunkte](02-member-registration.md#schritt-5-zahlpunkte-angeben)
→ [Admin-Einstellungen — Typabhängige Sichtbarkeit](06-admin-settings.md#typabhangige-sichtbarkeit-badges)

**Energiefelder pro Zählpunkt**

Die früheren Application-Level-Felder „Verbrauch Vorjahr/Prognose", „Einspeisung Prognose" und „PV-Leistung (kWp)" werden jetzt **pro Zählpunkt** abgefragt — direkt im jeweiligen Zählpunkt-Block des Formulars. Zusätzlich neu: **Einspeiselimit (kW)** für Anschlüsse mit begrenzter Einspeise-Leistung.

**B2B-SEPA-Firmenlastschrift mit Mandatsreferenz beim Import**

Für Unternehmens-Anträge mit B2B-SEPA-Mandat kommt das Mandat-PDF jetzt erst beim Import mit der zugewiesenen **Mitgliedsnummer als Mandatsreferenz** (notwendig damit das Mandat digital signiert werden kann — ein nachträglich modifiziertes Mandat hätte eine ungültige Signatur). Bis dahin wartet der Antrag im Status **„Warte auf Bank-Bestätigung"** auf die Rückmeldung des Mitglieds.

→ [Statusverwaltung — Post-Import-Stati](05-admin-status.md#post-import-stati)

**EEG-Umzuordnung im Admin**

Falls ein Antrag fälschlich in der falschen EEG gelandet ist (Mitglied hat den falschen RC-Link verwendet), kannst du ihn jetzt — solange er noch in der Review-Phase ist — direkt in eine andere EEG umzuordnen, ohne dass das Mitglied neu einreichen muss.

→ [Statusverwaltung — EEG umzuordnen](05-admin-status.md#eeg-umzuordnen)

**E-Mail an Mitglied bei Ablehnung und Info-Anfrage**

Bei den Status-Wechseln **Ablehnen** und **Info benötigt** wird die Begründung jetzt 1:1 im E-Mail-Body an das Mitglied übermittelt — kein generischer Text mehr, sondern dein konkreter Hinweis.

→ [Statusverwaltung — Ablehnen](05-admin-status.md#ablehnen-rejected)
→ [Statusverwaltung — Rückfragen stellen](05-admin-status.md#ruckfragen-stellen-needs_info)

**E-Fahrzeug-Detailerfassung**

Bei „E-Auto vorhanden = Ja" werden jetzt zusätzlich **Anzahl E-Fahrzeuge** und **Jahres-Kilometer** abgefragt.

**Netzbetreiber-Vollmacht pro EEG konfigurierbar**

Ob das Mitglied die EEG bevollmächtigt, beim Netzbetreiber für es zu handeln (z. B. bei Netz OÖ erforderlich), ist jetzt pro EEG ein- oder ausschaltbar.

**Titel-Nach + abweichende Adresse pro Zählpunkt**

Neue optionale Felder:

- **Titel nach** (z. B. „BSc", „MBA") als Pendant zum bisherigen Titel-Vor
- **Bankname** im öffentlichen Formular sichtbar
- **Abweichende Adresse pro Zählpunkt** — falls ein Zählpunkt nicht an der Wohnadresse liegt, kann pro Zählpunkt eine eigene Adresse angegeben werden

---

## Zur Doku selbst

Die **Doku-Site** ist jetzt eine echte Website mit:

- **Linke Sidebar** mit allen Seiten + Inhaltsverzeichnis je Seite
- **Such-Funktion** in der Header-Leiste
- **Light/Dark-Modus** (Toggle rechts oben — folgt standardmäßig dem Browser-Setting)
- **Klickbare „Edit"-Links** auf jeder Seite — führen direkt zum Markdown auf GitHub
- **Mermaid-Diagramme** statt ASCII-Skizzen für die Statusübergänge
- **Section-Screenshots** für die wichtigsten UI-Bereiche — neben der jeweiligen Beschreibung

URL: **<https://marki4711.github.io/eegfaktura-member-onboarding/>**

Wenn dir Inhalts-Lücken auffallen oder ein Screenshot veraltet aussieht, gib Bescheid.
