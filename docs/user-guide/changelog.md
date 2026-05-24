# Was ist neu?

Übersicht der Änderungen der letzten Tage, die für **EEG-Admins** und **Mitglieder** spürbar sind. Technische Details, Bug-Fixes ohne UI-Auswirkung und Infrastruktur-Themen sind hier bewusst weggelassen — die finden sich in der Commit-Historie.

---

## 2026-05-24

### Mitgliedstypen vereinfacht — Kleinunternehmer entfällt als eigener Typ

Der Mitgliedstyp **Kleinunternehmer** existiert nicht mehr als eigene Auswahl. Stattdessen wählt das Mitglied **Unternehmen** und lässt die UID-Nummer leer — das signalisiert dem System automatisch die Kleinunternehmerregelung (§ 6 Abs 1 Z 27 UStG, 0 % USt.). Mit ausgefüllter UID greift der reguläre Unternehmens-Pfad (20 % USt.). Bestehende Anträge mit altem Typ wurden automatisch auf `company` migriert.

→ [Mitglieder-Registrierung — Mitgliedstyp auswählen](02-member-registration.md#schritt-2-mitgliedstyp-auswahlen)

### USt.-Hinweise im Dropdown vereinheitlicht

Alle fünf Mitgliedstypen zeigen jetzt einen konkreten USt.-Satz im Auswahl-Dropdown statt einer Mischung aus konkreten Werten und „variabel". Gemeinde und Verein bekommen `(0 % oder 20 % USt.)`, das die typischen Fälle (Hoheits- vs. BgA-Bereich bzw. ideell vs. wirtschaftliche Tätigkeit) abdeckt.

### Bug-Fix: Antrag für Vereine/Unternehmen blockiert wegen fehlendem Geburtsdatum

Wenn eine EEG das Feld **Geburtsdatum** als Pflichtfeld konfiguriert hatte, scheiterten Vereins- und Unternehmens-Anträge mit „Geburtsdatum ist erforderlich" — obwohl das Feld im Formular für diese Typen gar nicht angezeigt wird. Behoben: der Pflicht-Check gilt jetzt nur noch für natürliche Personen (Privatperson, Landwirt). Analoge Korrektur für UID-Nummer (nur für Unternehmen, Gemeinde, Verein verlangt).

---

## 2026-05-23 — Neue Funktion: Datenweiterleitung an externe Systeme

Mit der **Datenweiterleitung** kannst du Antragsdaten an externe Systeme weitergeben — aktuell als **Excel/CSV-Export** mit konfigurierbarem Feldsatz. Pro EEG legst du fest, welche Felder enthalten sind, in welcher Reihenfolge sie stehen und mit welcher Spaltenüberschrift sie erscheinen.

Auslösen kannst du eine Weiterleitung entweder:
- **aus der Antragsliste** per Bulk-Aktion auf mehrere ausgewählte Anträge,
- **aus dem Antragsdetail** für einen einzelnen Antrag.

Jobs laufen asynchron im Hintergrund — eine Übersicht aller Läufe inkl. Download der Ergebnisdatei und Fehler-Diagnose findet sich im Job-Tab. Bei sensiblen Feldern (IBAN, Geburtsdatum) zeigt die UI eine DSGVO-Warnung. Weitere Plugins (Zoho, HubSpot, …) sind als Folge-Phasen geplant.

→ [Admin-Einstellungen — Datenweiterleitung](06-admin-settings.md#datenweiterleitung)
→ [Anträge verwalten — Massenaktionen](04-admin-applications.md#massenaktionen)

---

## 2026-05-24 — Neue Funktion: Konfiguration sichern und übertragen

Du kannst die komplette Konfiguration einer EEG (Stammdaten-Settings, Formular-Felder, Rechtsdokumente, Datenweiterleitungs-Configs) als **JSON-Datei** sichern und auf andere EEGs übertragen. Nützlich um:

- mehrere EEGs auf eine gemeinsame Grund-Konfiguration zu bringen,
- vor einem riskanten Änderungs-Schub den Ist-Zustand zu sichern (Apply ist nicht automatisch reversibel),
- Konfigurations-Stände nachvollziehbar in Git oder einem Backup-System zu halten.

Beim Import zeigt eine **Diff-Vorschau** pro Sektion was sich ändert (hinzugefügt / modifiziert / entfernt / unverändert). Du kannst einzelne Sektionen aus- oder abwählen.

→ [Admin-Einstellungen — Konfiguration Import / Export](06-admin-settings.md#konfiguration-import-export)

---

## 2026-05-21 — Ansprechperson + abweichende Rechnungs-E-Mail für Organisationen

Bei Unternehmen, Gemeinden und Vereinen kannst du jetzt zwei zusätzliche Datensätze erfassen:

- **Ansprechperson** — Name, E-Mail und Telefon der konkreten Kontaktperson der Organisation. Wird im Formular per Master-Checkbox ein-/ausgeblendet, damit der Block nur erscheint wenn relevant.
- **Abweichende Rechnungs-E-Mail** — separate E-Mail-Adresse für Rechnungs-Zustellung, unabhängig von der allgemeinen Kontakt-Adresse.

Beide Felder sind pro EEG konfigurierbar (Hidden / Optional / Pflicht / Admin-Vorbefüllung). Im Admin-Field-Editor zeigen Badges (`+Ansprechperson`, `+Rechnungs-E-Mail`), unter welcher Bedingung das jeweilige Feld im Formular wirklich greift.

→ [Mitglieder-Registrierung — Persönliche Daten](02-member-registration.md#schritt-3-personliche-daten-eingeben)
→ [Admin-Einstellungen — Formular-Felder](06-admin-settings.md#formular-felder-zahlpunktfelder)

---

## 2026-05-21 — Netzbetreiber-Info-Seite im Beitrittsbestätigungs-PDF

Wenn deine EEG die Netzbetreiber-Vollmacht aktiviert hat, kannst du jetzt Netzbetreiber-spezifische Hinweise (Anleitungs-Texte, Kontaktdaten, Portal-URLs) in den Einstellungen hinterlegen — diese erscheinen automatisch auf einer zusätzlichen Seite des Beitrittsbestätigungs-PDFs, das das Mitglied nach Annahme bekommt. Außerdem zwei neue konfigurierbare Felder im Mitgliedsformular:

- **Netzbetreiber Kundennummer** — die Vertragsnummer des Mitglieds beim Netzbetreiber
- **Inventarnummer eines Zählers** — eine eindeutige Kennung am Zähler

Beide nur sichtbar wenn die Netzbetreiber-Vollmacht aktiviert ist (Badge `+Vollmacht` im Admin-Editor).

→ [Admin-Einstellungen — Spezielle konfigurierbare Felder](06-admin-settings.md#spezielle-konfigurierbare-felder)
→ [E-Mails & PDFs — Beitrittsbestätigung](07-emails-and-pdfs.md)

---

## 2026-05-20

### Mitglieder-Registrierung: UID-Nummer für Vereine jetzt optional

Bisher wurde die UID-Nummer bei Vereinen vom Validator als Pflichtfeld behandelt — fachlich falsch, da nicht jeder Verein eine UID hat. Das Feld ist jetzt optional und kann leer gelassen werden.

### Hilfetext zur UID-Nummer

Im öffentlichen Registrierungsformular gibt es jetzt ein Info-Icon neben dem UID-Feld, das erklärt was die UID ist und wann sie nötig ist.

---

## 2026-05-19

### Aktivierungs-Modus pro EEG konfigurierbar

Steuert, wann ein Antrag automatisch von „Bereit zur Aktivierung" auf „Aktiviert" wechselt. Zwei Varianten:

- **Variante A (Default):** Mitglied im eegFaktura-Core hat Status `ACTIVE` — klassisches Verhalten.
- **Variante B:** Mindestens ein Zählpunkt im Core hat eine laufende Netzbetreiber-Anmeldung. Aktiviert das Mitglied bereits sobald die EDA-Meldung beim Netzbetreiber gestartet ist, ohne den Abschluss abzuwarten.

Beim Übergang auf „Aktiviert" wird die volle Beitrittsbestätigungs-Mail mit PDF versandt.

→ [Admin-Einstellungen — Aktivierungs-Kriterium](06-admin-settings.md#aktivierungs-kriterium)

### Manueller Skip-Import — Ausnahmefall `approved → activated`

Falls ein Mitglied im eegFaktura-Core bereits manuell angelegt/überschrieben wurde und der reguläre Import-Pfad übersprungen werden soll, gibt es jetzt im Detail-View den Button **„Manuell aktivieren …"** als Ausnahmefall.

→ [Statusverwaltung — Ausnahmefall: approved → activated](05-admin-status.md#ausnahmefall-approved-activated-manueller-skip-import)

### Teilnahmefaktor pro EEG konfigurierbar

Pro EEG steuerst du jetzt ob der **Teilnahmefaktor (%)** im Mitgliedsformular sichtbar ist, ob ihn das Mitglied ändern darf oder ob er fest auf 100 % steht. Bei Hidden/Admin-Vorbefüllung wird der Wert serverseitig automatisch auf 100 % gesetzt — Beitrittsbestätigung, Mail und Excel-Export zeigen ihn unverändert.

→ [Admin-Einstellungen — Formular-Felder](06-admin-settings.md#formular-felder-zahlpunktfelder)

---

## 2026-05-18

### SEPA-Mandat-Datum auf Übermittlungstag vorbefüllt

Beide SEPA-Mandate (Basislastschrift CORE und B2B-Firmenlastschrift) zeigen im Unterschriftsfeld jetzt das **Datum der Übermittlung** vorbefüllt. Das Mitglied trägt nur noch Ort + Unterschrift ein. Das Datum wird im Antrags-Detail unter „Mandatsdatum" angezeigt und beim Faktura-Import als Mandate-Date mitgeführt.

### Zählpunkt-Prefix pro EEG konfigurierbar

Wenn die Zählpunkte deiner EEG mehrheitlich vom selben Netzbetreiber + PLZ-Bereich kommen, kannst du in den Einstellungen einen **festen Prefix** pro Richtung (Verbraucher / Einspeisung) hinterlegen. Das Mitglied tippt dann nur noch die individuellen letzten Stellen — der Prefix ist gelockt und kann nicht überschrieben werden. Beim Verlassen des Eingabefelds werden fehlende Stellen automatisch mit führenden Nullen ergänzt.

→ [Admin-Einstellungen — Zählpunkt-Prefixes](06-admin-settings.md#zahlpunkt-prefixes)
→ [Mitglieder-Registrierung — Zählpunkte](02-member-registration.md#schritt-5-zahlpunkte-angeben)

### Zählpunkt-Format 2-6-5-20 in PDF + Mail

Die Zählpunkt-Nummer wird jetzt überall in der offiziellen E-Control-Gruppierung `AT 003100 00000 12345678901234567890` angezeigt (PDFs, Bestätigungs-E-Mails, Admin-Detail-View).

### Bankname als konfigurierbares Feld pro EEG

Bisher war der Bankname fest sichtbar im Bankverbindungs-Block. Jetzt steuerst du pro EEG ob er ausgeblendet, optional oder Pflicht ist. Default: Optional (bewahrt heutiges Verhalten).

### Firmenbuchnummer optional für Unternehmen

Bisher Pflichtfeld bei `memberType=company`, jetzt durchgehend optional (auch wenn EEG-Feld-Config sie als „Pflicht" markiert hat).

---

## 2026-05-17 — Größerer Rollout neuer Felder

### Erzeugungsform + Batterie-Felder

Erzeuger-Zählpunkte fragen jetzt die **Erzeugungsform** (PV / Wasser / Wind / Biomasse) ab und — bei PV — optional Batteriespeicher-Daten (Größe, Wechselrichter-Hersteller, Speichersteuerungs-Einverständnis). Alle Felder sind pro EEG konfigurierbar; die Sichtbarkeit ist typabhängig und wird im Admin-Editor mit farbigen Badges (`[Verbraucher]`, `[Einspeisung]`, `[PV]`, `[+Speicher]`, etc.) sofort sichtbar gemacht.

→ [Mitglieder-Registrierung — Zählpunkte](02-member-registration.md#schritt-5-zahlpunkte-angeben)
→ [Admin-Einstellungen — Typabhängige Sichtbarkeit](06-admin-settings.md#typabhangige-sichtbarkeit-badges)

### B2B-SEPA-Firmenlastschrift mit Mandatsreferenz beim Import

Für Unternehmens-Anträge mit B2B-SEPA-Mandat kommt das Mandat-PDF jetzt erst beim Import mit der zugewiesenen **Mitgliedsnummer als Mandatsreferenz** (notwendig damit das Mandat digital signiert werden kann — ein nachträglich modifiziertes Mandat hätte eine ungültige Signatur). Bis dahin wartet der Antrag im Status **„Warte auf Bank-Bestätigung"** auf die Rückmeldung des Mitglieds.

→ [Statusverwaltung — Post-Import-Stati](05-admin-status.md#post-import-stati)

### EEG-Umzuordnung im Admin

Falls ein Antrag fälschlich in der falschen EEG gelandet ist (Mitglied hat den falschen RC-Link verwendet), kannst du ihn jetzt — solange er noch in der Review-Phase ist — direkt in eine andere EEG umzuordnen, ohne dass das Mitglied neu einreichen muss.

→ [Statusverwaltung — EEG umzuordnen](05-admin-status.md#eeg-umzuordnen)

### E-Mail an Mitglied bei Ablehnung und Info-Anfrage

Bei den Status-Wechseln **Ablehnen** und **Info benötigt** wird die Begründung jetzt 1:1 im E-Mail-Body an das Mitglied übermittelt — kein generischer Text mehr, sondern dein konkreter Hinweis.

→ [Statusverwaltung — Ablehnen](05-admin-status.md#ablehnen-rejected)
→ [Statusverwaltung — Rückfragen stellen](05-admin-status.md#ruckfragen-stellen-needs_info)

### E-Fahrzeug-Detailerfassung

Bei „E-Auto vorhanden = Ja" werden jetzt zusätzlich **Anzahl E-Fahrzeuge** und **Jahres-Kilometer** abgefragt.

### Netzbetreiber-Vollmacht pro EEG konfigurierbar

Ob das Mitglied die EEG bevollmächtigt, beim Netzbetreiber für es zu handeln (z. B. bei Netz OÖ erforderlich), ist jetzt pro EEG ein- oder ausschaltbar.

### Titel-Nach + abweichende Adresse pro Zählpunkt

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

URL: **https://marki4711.github.io/eegfaktura-member-onboarding/**

Wenn dir Inhalts-Lücken auffallen oder ein Screenshot veraltet aussieht, gib Bescheid.
