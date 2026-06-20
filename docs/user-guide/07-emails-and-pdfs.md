# E-Mails & PDF-Anhänge — Übersicht

Diese Übersicht zeigt für Tester:innen und EEG-Admins, **wann** das
Onboarding welche E-Mails versendet, **wer** sie bekommt und **welche
PDFs** angehängt sind. Sortiert nach typischer Reihenfolge im Lebenszyklus
eines Antrags.

> Die Inhalte und Auslöser-Zeitpunkte ändern sich je nach EEG-Einstellungen
> — die Spalten „Auslöser" und „Voraussetzungen" geben das jeweils an.

---

## Mails an das Mitglied

| # | Mail | Auslöser | PDF-Anhang | Voraussetzungen / Variante |
|---|------|----------|------------|----------------------------|
| 1 | **Eingangsbestätigung** | direkt nach Einreichung | SEPA-Mandat (Basis oder B2B-Firmenlastschrift) | nur wenn EEG SEPA-Mandat aktiviert hat UND in den Einstellungen NICHT „Mandat erst beim Import" gewählt ist |
| 1' | **Eingangsbestätigung** mit Hinweis „Mandat folgt beim Import" | direkt nach Einreichung | — kein Anhang | wenn EEG „Mandat erst beim Import" gewählt hat ODER kein SEPA-Mandat aktiviert ist |
| 2 | **E-Mail-Bestätigungs-Link** | als Bestandteil der Eingangsbestätigung (gleiche Mail wie #1) | — | nur wenn EEG „E-Mail-Adresse bestätigen" aktiviert hat. Bestätigungs-Button im Mailtext, Link 30 Tage gültig |
| 3 | **Rückfrage** („Wir brauchen noch Informationen") | Admin setzt Status auf `needs_info` | — | Reason des Admins wird 1:1 in den Mail-Body übernommen |
| 4 | **Ablehnung** | Admin setzt Status auf `rejected` | — | Reason des Admins wird 1:1 in den Mail-Body übernommen |
| 5 | **Anlage SEPA-Mandat** („Beitrittsbestätigung folgt") | nach erfolgreichem Import in eegFaktura | SEPA-Mandat mit eingedruckter Mitgliedsnummer als Mandatsreferenz | nur wenn einzugsart = B2B ODER EEG-Einstellung „Mandat erst beim Import" aktiv. Sonst geht beim Import **keine** Mail. |
| 6 | **Beitrittsbestätigung** (die formale Mitgliedschafts-Bestätigung) | beim Wechsel auf Status „Aktiviert" | Beitrittsbestätigungs-PDF (volles Antrags-Detail + Mitgliedsnummer) | sowohl beim regulären `ready_for_activation → activated` als auch beim manuellen Skip-Import `approved → activated`. Wird **genau einmal** verschickt (auch bei mehrfachen Status-Wechseln). **Entfällt, wenn die EEG den Vorstands-Genehmigungs-Workflow aktiviert hat** — dann gibt es stattdessen Mail-Nr. C′ an den EEG-Kontakt. **Entfällt ebenso, wenn die EEG „Beitrittsbestätigung an die EEG senden" aktiviert hat** — dann geht die Beitrittsbestätigung als Mail-Nr. C″ an den EEG-Kontakt zum Weiterleiten. In beiden Fällen wird das Mitglied über die reguläre eegFaktura-Aktivierungs-Mail vom Core informiert. |

---

## Mails an den EEG-Kontakt

| # | Mail | Auslöser | PDF-Anhang |
|---|------|----------|------------|
| A | **Neuer Antrag** | direkt nach Einreichung — oder, wenn das Mitglied seine E-Mail erst bestätigen muss, erst nach dem Klick auf den Bestätigungs-Link | — |
| B | **SEPA-Mandat versandt** | parallel zu Mitglieder-Mail #5 (Import + Mandat-Versand) | gleiche Mandat-PDF wie das Mitglied bekommen hat |
| C | **Mitglied aktiviert** | parallel zu Mitglieder-Mail #6 (Wechsel auf „Aktiviert") | gleiche Beitrittsbestätigungs-PDF wie das Mitglied |
| C′ | **Beitrittserklärung zur Unterzeichnung** | beim Wechsel auf „Aktiviert", wenn die EEG den Vorstands-Genehmigungs-Workflow aktiviert hat | Beitrittserklärungs-PDF mit Vorstands-Signaturblock am Ende. Vorstand unterschreibt und leitet manuell ans Mitglied weiter. Mail-Nr. 6 an das Mitglied entfällt; das Mitglied wird über die reguläre eegFaktura-Aktivierungs-Mail informiert. **Pflicht-Bedingung:** EEG-Kontakt-Mail muss gepflegt sein, sonst bricht der Aktivierungs-Übergang ab. |
| C″ | **Beitrittsbestätigung zum Weiterleiten** | beim Wechsel auf „Aktiviert", wenn die EEG „Beitrittsbestätigung an die EEG senden" aktiviert hat | Beitrittsbestätigungs-PDF (dasselbe, das sonst das Mitglied bekäme), mit Vorspann „Bitte an das Mitglied weiterleiten". Das Mitglied bekommt vom System **nichts** (auch keine Aktivierungs-Mail #6). **Pflicht-Bedingung:** EEG-Kontakt-Mail muss gepflegt sein. Unabhängig vom Vorstands-Workflow (C′) — beide Schalter wirken getrennt. |

> EEG-Kontakt = die E-Mail-Adresse, die in den EEG-Stammdaten als
> Kontaktadresse hinterlegt ist. Wenn keine Kontaktadresse gepflegt ist,
> entfällt diese Kopie still (das Mitglied bekommt seine Mail trotzdem).

---

## PDFs in der Übersicht

| PDF | Wann erzeugt | Was steht drin |
|-----|--------------|----------------|
| **SEPA-Basislastschrift-Mandat** | beim Einreichen (Default) ODER beim Import (EEG-Einstellung „Mandat erst beim Import") | Mitglieder-Daten, IBAN, Bank-Daten, Mandatsreferenz (Mitgliedsnummer, sofern beim Import erzeugt), Datum der Übermittlung als Unterschriftsdatum |
| **SEPA-Firmenlastschrift-Mandat (B2B)** | nur bei einzugsart = B2B, beim Import | wie Basis-Mandat, aber Firmen-Daten + Hinweise zur Hausbank-Pre-Notification, Mandatsreferenz = Mitgliedsnummer |
| **Beitrittsbestätigung** | beim Wechsel auf „Aktiviert" | volles Antrags-Detail (Stammdaten, Zählpunkte, Adresse, erteilte Zustimmungen, Netzbetreiber-Vollmacht), Mitgliedsnummer |

Alle PDFs tragen das EEG-Logo (sofern in den EEG-Einstellungen synchronisiert)
und werden mit Europe/Vienna-Zeitstempel versehen.

---

## Wann das Mitglied *nichts* bekommt

Bewusste Übergänge ohne Mail an das Mitglied:

- **`submitted → email_confirmed`** (Mitglied klickt Bestätigungs-Link) — keine zweite Mail, der Klick selbst ist die Bestätigung. Stattdessen wird _jetzt_ die aufgeschobene EEG-Notification verschickt.
- **`submitted → under_review`**, **`under_review → approved`** — interne Statuswechsel des Admins, ohne Mitglied-Information.
- **`approved → imported` ohne Mandat** (z. B. einzugsart = "kein SEPA" oder = "Bar"): kein Mandat zu versenden, also keine Import-Mail; die Beitrittsbestätigung folgt erst beim Wechsel auf „Aktiviert".
- **Reset-Import** (`imported → approved` über „Import zurücksetzen"): rein admin-seitig, keine Mitteilung ans Mitglied.

---

## Welche Mails blockieren den Statuswechsel bei SMTP-Fehler?

Zwei Mail-Typen sind **„hard-fail synchron"** — wenn der SMTP-Server zum
Zeitpunkt des Versands nicht erreichbar ist, wird der Statuswechsel
automatisch zurückgerollt und der Admin bekommt im Dialog eine Fehlermeldung
mit HTTP 500:

- **Ablehnung** (`→ rejected`)
- **Rückfrage** (`→ needs_info`)

Begründung: bei diesen Mails ist es kritisch, dass das Mitglied die Information
erhält — eine stille Mail-Verlust-Situation würde den Antrag in einem Status
hinterlassen, in dem das Mitglied passiv wartet.

Alle anderen Mails laufen **„best-effort async"** — der Statuswechsel wird
auch dann persistiert, wenn der SMTP-Server gerade nicht erreichbar ist. Bei
Fehlern landet eine Warnung im Backend-Log, der Admin kann ggf. eine Mail
nachträglich manuell verschicken (z. B. „Bestätigungs-Link erneut senden" in
der Antrags-Detailansicht).

---

## Im Mailprogramm wiederfinden

Damit Mitglieder die Mails leichter zuordnen können, sind die Betreffzeilen
einheitlich konstruiert:

- `Bestätigung deiner Anmeldung – [EEG-Name]`
- `Rückfragen zu deinem Beitrittsantrag ([Antragsnummer])`
- `Wir können deinen Antrag nicht annehmen – [EEG-Name]`
- `Dein SEPA-Mandat – Mitgliedsnummer [Nummer]`
- `Deine Beitrittsbestätigung – Mitgliedsnummer [Nummer]`

Die EEG-Kontaktadresse erhält Mails mit dem Präfix des Antragstellers
oder mit „Antrag …" / „SEPA-Mandat versandt …" / „Antrag aktiviert …",
gefolgt von Mitgliedsname und Referenznummer in Klammern.

---

## Technische Referenz

Die Entwickler-/Operations-Sicht (welche Funktion löst aus, welches Template
rendert, welche Metriken werden geschrieben) findet sich in
[`docs/architecture.md`](https://github.com/Marki4711/eegfaktura-member-onboarding/blob/main/docs/architecture.md#mail-flow-post-proj-46--proj-47--proj-48--proj-53)
(Mail-Flow-Tabelle) und in [`docs/operations.md`](https://github.com/Marki4711/eegfaktura-member-onboarding/blob/main/docs/operations.md)
(SMTP-Ausfall-Verhalten).
