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

### Gemeinschafts-ID

Die interne ID Ihrer EEG in eegFaktura. Sie wird im Excel-Export für den Datenimport verwendet.

### EEG-Stammdaten — aus eegFaktura

Name, Adresse, Creditor-ID und Kontakt-E-Mail Ihrer Energiegemeinschaft werden direkt aus eegFaktura übernommen. Diese Felder sind in der Onboarding-Oberfläche **schreibgeschützt** (kleines Schloss-Symbol). Änderungen erfolgen ausschließlich in eegFaktura selbst.

**Stand-Anzeige am oberen Rand der Stammdaten-Card:**

- **Grün — „Synchron mit eegFaktura · Stand: DD.MM. HH:MM"**: die Daten stimmen mit eegFaktura überein, kein Handlungsbedarf.
- **Orange — „Stammdaten weichen ab"**: in eegFaktura wurden Daten geändert seit dem letzten Sync. Über **„Details anzeigen ▾"** sieht man eine Tabelle „Im Onboarding | In eegFaktura" je geändertem Feld. Mit **„Aus eegFaktura aktualisieren"** wird der lokale Stand überschrieben.
- **Grau — „eegFaktura nicht erreichbar"**: temporärer Ausfall des Core-Systems. Onboarding nutzt weiter den zuletzt gesyncten Stand.

**Erstmaliger Sync nach Inbetriebnahme:** klicken Sie einmal „Aus eegFaktura aktualisieren", damit die Stammdaten in die Onboarding-Datenbank kopiert werden. Bis dahin sind die Felder leer und die Hinweis-Box weist Sie darauf hin.

### SEPA-Lastschriftmandat

- **SEPA-Lastschriftmandat dem Willkommensmail anhängen**: Wenn aktiv, wird beim Einreichen eines Mitgliedsantrags automatisch ein PDF-Mandat generiert und als Anhang im Willkommensmail verschickt.
- **Firmenlastschrift (B2B)**: Erscheint nur wenn SEPA aktiv ist. Aktivieren Sie diese Option, wenn Unternehmen und Verbände das B2B-Mandat erhalten sollen.

> **Hinweis:** Wenn das SEPA-Mandat aktiviert ist, aber Stammdaten fehlen, erscheint eine Warnung. Solange Felder fehlen, wird kein PDF generiert.

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

Klicken Sie auf **Konfiguration speichern**, um die Änderungen zu übernehmen.

---

## Rechtsdokumente

![Rechtsdokumente](images/admin-settings-legal.png)

Hier verwalten Sie EEG-spezifische Dokumente (z.B. Satzung, Nutzungsbedingungen), denen Mitglieder bei der Registrierung zustimmen müssen.

### Dokument hinzufügen

1. Klicken Sie auf **Dokument hinzufügen**.
2. Geben Sie einen Titel und die URL des Dokuments ein.
3. Aktivieren Sie **Zustimmung erforderlich**, wenn das Mitglied dem Dokument aktiv zustimmen muss.
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
