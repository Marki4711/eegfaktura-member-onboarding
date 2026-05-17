# PROJ-44: Netzbetreiber-Vollmacht (per-EEG konfigurierbare Zustimmung)

**Status:** In Review
**Created:** 2026-05-17

## Hintergrund

Manche Netzbetreiber (z.B. Netz OÖ) verlangen eine separate Vollmacht
des Mitglieds, damit die EEG in dessen Namen Schritte rund um die
(De-)Aktivierung der Zählpunkte im Netzbetreiber-Portal durchführen
darf. Diese Vollmacht ist **nicht** Teil der EEG-Mitgliedschafts­
zustimmung — sie richtet sich an einen Dritten und ist nicht bei jeder
EEG nötig.

## Datenmodell

Zwei neue Spalten auf `application`:

- `network_operator_authorization` BOOLEAN NOT NULL DEFAULT FALSE —
  vom Mitglied erteilte Vollmacht (true) oder nicht (false). Default
  FALSE deckt Bestands-Anträge sauber ab.
- `network_operator_authorization_at` TIMESTAMPTZ NULL — Zeitstempel
  der Erteilung. Wird vom Service beim Zustimmen auf NOW() gesetzt.

Migration: `db/migrations/000039_network_operator_authorization.up.sql`.

## Feldkonfiguration (PROJ-8-Pattern)

Neues konfigurierbares Feld:

- `network_operator_authorization` — `defaultState: "hidden"`, Label
  „Netzbetreiber-Vollmacht erteilen"

EEGs ohne diese Anforderung lassen es auf `hidden` (Default —
Bestands-EEGs bleiben unverändert). Netz-OÖ-EEGs setzen es auf
`required`. `optional` ist erlaubt, aber für eine Vollmacht selten
sinnvoll — wir blockieren das nicht.

## Volltext der Vollmacht

Der rechtsverbindliche Wortlaut wird im Frontend neben der Checkbox
ausgegeben und in PDF/Excel mitgezeichnet. Die Version wird **nicht**
versioniert (analog `accuracy_confirmed`) — der Text liegt im Code,
Änderungen müssen über einen Code-Deployment laufen.

Wortlaut (verbindlich):

> Ich erteile der EEG für die Dauer der Mitgliedschaft zeitlich
> unbegrenzt die Vollmacht, in meinem Namen sämtliche Schritte und
> Abstimmungen mit dem zuständigen Netzbetreiber durchzuführen, die
> zur vollständigen (De-)Aktivierung der angeführten Zählpunkte in
> der EEG notwendig sind. Dies betrifft insbesondere auch die Nutzung
> des Online-Portals des Netzbetreibers.

## UI-Verhalten

Public-Form:
- Block wird **nur** gerendert, wenn EEG das Feld auf `optional` oder
  `required` konfiguriert hat
- Volltext wird über der Checkbox ausgegeben (nicht in einem Tooltip)
- Bei `required`: Häkchen muss gesetzt sein, sonst Validierungsfehler

Admin-Detail:
- „Netzbetreiber-Vollmacht: Ja (erteilt am 17.05.2026)" / „Nein"
- Nicht editierbar (Mitglied-Erteilung; admin-Override nicht V1)

## Export / Mail

- Mail (Member + EEG): erscheint im „Zusätzliche Informationen"-Block
  via `buildConfigurableFields` (Wert „Ja"). FALSE wird unterdrückt
  (Default für Bestandsanträge, soll nicht auftauchen).
- Approval-PDF: dito
- Excel-Export: **nicht** mit aufgenommen — der Excel-Export folgt einer
  fixen eegFaktura-Importer-Spalten­struktur und kennt das Feld nicht.
  Der Audit-Trail liegt in DB + PDF + Admin-Detail.

## Out of Scope

- Versionierung des Vollmachtstexts (YAGNI; bei Bedarf später analog
  PROJ-9 als legales Dokument modellieren)
- Per-EEG abweichender Wortlaut
- Admin-Override (Vollmacht im Nachhinein im Namen des Mitglieds
  setzen) — bewusst nicht V1
- Widerruf der Vollmacht über das Onboarding-Tool — out

## Tests

- Build muss grün bleiben
- Smoke-Test: EEG mit dem Feld auf `required` konfigurieren, Antrag
  ohne Häkchen → Validierungsfehler; Antrag mit Häkchen → submit
  erfolgreich, `_at` gesetzt, Wert erscheint in Mail/PDF/Excel
- Default-EEGs ohne Konfiguration: Feld wird nicht gerendert,
  Antragsfluss unverändert
