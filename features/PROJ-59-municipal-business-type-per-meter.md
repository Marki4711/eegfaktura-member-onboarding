# PROJ-59: BgA / Hoheitsbereich-Vermerk im Anlagennamen bei Gemeinden

## Status: In Review
**Created:** 2026-05-23
**Last Updated:** 2026-05-23 (Scope auf Variante 2 reduziert, Hilfetext-Implementierung)

## Dependencies
- Berührt: PROJ-7 (Mitgliedstypen) — Hilfetext bei der Option „Gemeinde"
- Berührt: PROJ-8 (Konfigurierbare Felder) — Feld `installation_name` muss bei der EEG mindestens als optional konfiguriert sein, damit Gemeinden den Vermerk eintragen können

## Hintergrund

Bei österreichischen Gemeinden ist umsatzsteuerlich zwischen **Betrieb gewerblicher Art (BgA)** und **Hoheitsbereich** zu unterscheiden. Die Unterscheidung gilt zählpunktbezogen — eine Gemeinde kann z. B. den Bauhof-Bezugszählpunkt als BgA und den Rathaus-Bezug im Hoheitsbereich führen. Beide Klassifikationen beeinflussen, welcher Tarif (USt-Pflicht / Steuerbefreiung) in eegFaktura passend ist.

## Entscheidungs­historie

Während `/requirements` und `/grill-me` wurde zunächst eine **strukturierte Lösung** (eigene `municipal_business_type`-Spalte auf `metering_point` + Pflichtfeld im Public-Form + zwei Validierungs-Gates + Clearing-Helper + Badge im Tarif-Dialog + Excel/PDF-Spalte) erarbeitet. Beim Übergang in `/backend` wurde die Komplexität dieser Lösung an der erwarteten Mengenlage gespiegelt:

- Gemeinde-Anmeldungen sind im Bestand selten (einstellige Zahl pro EEG)
- Die Information wird **nur** zur Tarif-Entscheidung beim Import gebraucht — keine Filter, keine Reports, keine Downstream-Logik im Onboarding
- Die strukturelle Lösung hätte DB-Schema, Backend-Validierung an drei Stellen, Frontend-Discriminator-Union und Admin-UI an mehreren Komponenten bedeutet

Entscheidung 2026-05-23: **Variante 2 — Vermerk im Anlagennamen** ohne strukturelle Erfassung.

Die Spec ist dadurch von einem mehrschichtigen Feature auf einen **reinen Hilfetext-PR** geschrumpft.

## Umfang (umgesetzt)

### Frontend-Änderungen (zwei Stellen)

1. **`src/components/metering-point-fields.tsx`** — Der Info-Popover am Feld „Anlagenname" zeigt einen zusätzlichen Absatz, sobald `memberType=municipality` gewählt ist: Hinweis, dass „BgA" bzw. „Hoheit" mit in den Anlagennamen geschrieben werden soll (z. B. „Bauhof — BgA", „Rathaus — Hoheit").
2. **`src/components/registration-form.tsx`** — Der bestehende Gemeinde-Hilfetext im MemberTypeSelector wird um eine Notiz ergänzt, dass je Zählpunkt im Feld Anlagenname BgA/Hoheit vermerkt werden soll.

### Was **nicht** umgesetzt wird (bewusst)

- Keine DB-Migration
- Keine neue Spalte
- Kein Backend-Code (keine Validierung, keine Clearing-Logik, keine Gates)
- Keine Änderung an Public-Form-Validierung (Zod-Schema unverändert)
- Keine Änderung am Tarif-Auswahldialog (PROJ-27)
- Keine Änderung an Excel/PDF
- Keine Änderung an der externen API (PROJ-13)

## Bedingungen & Annahmen

- Das Feld `installation_name` ist konfigurierbar (PROJ-8). Damit Gemeinden den Vermerk eintragen können, muss die jeweilige EEG das Feld mindestens auf „optional" gestellt haben. Wenn die EEG das Feld auf „hidden" gesetzt hat, ist der Workflow nicht nutzbar — das ist eine bewusste Akzeptanz, da im aktuellen Bestand alle EEGs das Feld sichtbar haben.
- Der Admin liest den Anlagennamen beim Tarif-Auswahldialog (PROJ-27) und entscheidet auf dieser Basis manuell, welcher Tarif passt.
- Bei abweichenden Konventionen pro EEG (z. B. „BgA" vs. „gewerblich") muss die EEG ihre Mitglieder gesondert informieren — das System gibt nur einen Vorschlag im Popover.

## Edge Cases (akzeptiert)

- **Mitglied trägt den Vermerk nicht ein:** Admin sieht beim Tarif-Setzen keinen Hinweis, ruft Mitglied an oder vermutet basierend auf Zählpunkt-Adresse. Akzeptierter Workflow-Bruch — alternative Lösung wäre die ursprünglich verworfene strukturelle Erfassung.
- **Mitglied trägt unsinnigen Text ein:** unstrukturiertes Free-Text-Feld, keine Validierung. Admin liest und interpretiert.
- **Externe API liefert Antrag ohne Vermerk:** identisch zum Public-Form-Pfad ohne Eintrag — Admin klärt nach.
- **Bestandsdaten:** keine Migration, keine Aufarbeitung. Bestehende Gemeinde-Anträge bleiben unangetastet.

## Tests

Keine zusätzlichen Tests erforderlich:
- Keine Backend-Logik geändert
- Frontend-Änderung ist reines Popover-Text-Conditional — keine neue State-Machine
- Bestehende Tests für `MeteringPointFields` und `registration-form` bleiben grün, da kein neuer Pflichtfeld-Pfad eingeführt wird

## Notes

- **Optionale Erweiterung später**: Falls Gemeinde-Anmeldungen häufiger werden und der Tarif-Aufwand spürbar wird, kann der historische Variante-1-Entwurf (siehe Git-History dieser Datei) als Ausgangspunkt für eine strukturierte Lösung dienen. Bis dahin bleibt der Hilfetext-Vermerk der pragmatische Weg.
- **PROJ-55 (Nachmelden von Zählpunkten):** Die Vorgriffs-Notiz dort kann ebenfalls auf Variante 2 reduziert werden — der Nachmelde-Flow muss bei Gemeinde-Mitgliedern denselben Hilfetext am Anlagennamen anzeigen, mehr nicht.
- Security-Review nicht erforderlich: keine neuen Endpoints, keine Auth-Änderung, kein Schema-Change.

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)

_Verworfene strukturelle Lösung — historischer Zwischenstand in der Git-History dieser Datei. Aktuelle Lösung ist eine reine Hilfetext-Ergänzung an zwei Stellen im Frontend, kein eigenes Tech Design notwendig._

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
