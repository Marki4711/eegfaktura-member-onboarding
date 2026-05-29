# Was ist neu?

Übersicht der Änderungen der letzten Tage, die für **EEG-Admins** und **Mitglieder** spürbar sind. Technische Details, Bug-Fixes ohne UI-Auswirkung und Infrastruktur-Themen sind hier bewusst weggelassen — die finden sich in der Commit-Historie.

---

## 2026-05-29

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

Sobald ein Mitglied in den Status „importiert" oder „aktiviert" wechselte, verschwand der Download-Button für die Beitrittsbestätigungs-PDF (und für den Excel-Export). Der PROJ-46-Statusausbau hat damals die Allow-List vergessen. Beide Downloads bleiben jetzt in allen Post-Approval-Status verfügbar — Admins ziehen den Excel-Export typischerweise erst nach dem Import zur Ablage.

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

Die USt.-Sätze stehen nicht mehr in den Optionen des Mitgliedstyp-Dropdowns — sie waren bei Misch-Typen (Gemeinde, Verein, Unternehmen mit/ohne Kleinunternehmerregelung) ohnehin nur ein grober Hinweis. Die tatsächliche umsatzsteuerliche Einordnung ergibt sich aus den Folgefeldern (z. B. UID-Nummer bei Unternehmen) und wird in der Abrechnung mit der EEG geklärt.

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
