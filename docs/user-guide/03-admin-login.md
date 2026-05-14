# Anmeldung in der Admin-Oberfläche

## Voraussetzungen

Um die Admin-Oberfläche nutzen zu können, benötigen Sie:
- Einen Keycloak-Benutzeraccount (wird vom Systembetreiber eingerichtet)
- Das Attribut `tenant` mit den RC-Nummern Ihrer EEG(s) in Ihrem Benutzeraccount

Wenden Sie sich an Ihren Systembetreiber, falls Sie noch keinen Zugang haben.

## Anmeldung

1. Öffnen Sie die Admin-Oberfläche unter `https://<ihre-domain>/admin`
2. Sie werden automatisch zur Keycloak-Anmeldeseite weitergeleitet

![Keycloak-Anmeldeseite](images/admin-login-keycloak.png)

3. Geben Sie Ihren Benutzernamen und Ihr Passwort ein
4. Nach erfolgreicher Anmeldung werden Sie zur Antragsübersicht weitergeleitet

## Welche EEGs sehe ich?

Als EEG-Betreiber sehen Sie ausschließlich die Anträge jener EEGs, die in Ihrem Keycloak-Account hinterlegt sind. Es ist nicht möglich, Anträge anderer EEGs einzusehen.

## Sitzungsablauf

Aus Sicherheitsgründen läuft Ihre Anmelde-Sitzung nach einiger Zeit ab. Wenn Sie nach Ablauf eine Aktion ausführen (z. B. einen Antrag öffnen), werden Sie automatisch zur Keycloak-Anmeldeseite zurückgeleitet, um Ihre Sitzung zu erneuern. Nach erfolgreicher Re-Authentifizierung landen Sie wieder in der Admin-Oberfläche.

Sollte die automatische Re-Anmeldung selbst fehlschlagen (z. B. weil das Backend gerade einen Neustart durchführt), wird die Weiterleitung für 30 Sekunden ausgesetzt, um Endlos-Schleifen zu vermeiden. Versuchen Sie es danach erneut oder laden Sie die Seite neu.

## Abmeldung

Klicken Sie oben rechts auf **Abmelden**, um sich sicher aus der Admin-Oberfläche abzumelden.

![Abmelden-Button](images/admin-logout.png)
