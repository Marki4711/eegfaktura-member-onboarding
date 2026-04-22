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

## Abmeldung

Klicken Sie oben rechts auf **Abmelden**, um sich sicher aus der Admin-Oberfläche abzumelden.

![Abmelden-Button](images/admin-logout.png)
