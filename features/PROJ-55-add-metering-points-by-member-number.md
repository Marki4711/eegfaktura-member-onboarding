# PROJ-55 — Nachmelden von Zählpunkten anhand der Mitgliedsnummer

**Status:** Planned (Idea — wird in ein größeres Self-Service-Portal-Konzept aufgehen, siehe Hinweis unten)
**Erstellt:** 2026-05-21
**Letzte Klärung:** 2026-05-23 — Scope erweitert, separater Strang
**Quelle:** Tester-Feedback (ursprünglich), inzwischen Owner-Direction
**Abhängigkeiten:** PROJ-32 (EEG-Stammdaten + Mitgliederliste aus Core), evtl. PROJ-31 (E-Mail-Bestätigung als Auth-Hebel)

> **Scope-Hinweis (2026-05-23):** Diese Spec war ursprünglich eine punktuelle
> Erweiterung („Zählpunkt nachmelden mit Mitgliedsnummer"). Owner hat
> entschieden, dass das stattdessen Bestandteil eines **Self-Service-Portals
> für bestehende EEG-Mitglieder** werden soll — also ein eigener Bereich, in
> dem Mitglieder über die Onboarding-App hinaus ihre Daten einsehen und
> pflegen können (Zählpunkte nachmelden, Stammdaten ändern, Status verfolgen,
> evtl. weitere Aktionen). Das ist ein größerer Umbau / eine
> Weiterentwicklung der Lösung — nicht eine simple Feature-Erweiterung.
>
> Der genaue Umfang muss separat definiert werden (eigener `/requirements`-
> Lauf mit Scope-Diskussion: welche Mitglieder-Aktionen sind im Self-Service
> erlaubt, Auth-Modell, Abgrenzung zum Admin-Bereich, Verhältnis zum
> eegFaktura-Core-Mitgliederbereich). Diese Spec hier bleibt bis dahin als
> Idee-Ablage liegen — bei der späteren Self-Service-Portal-Spec wird sie
> als ein abgedeckter Use Case darin aufgehen oder geschlossen werden.

---

## Idee in einem Satz

Bestehendes EEG-Mitglied (hat bereits eine Mitgliedsnummer im Core) soll
einen **zusätzlichen Zählpunkt** zu seiner bestehenden Teilnahme melden
können — **ohne** den vollen Antragsprozess erneut zu durchlaufen.

## Hintergrund

Aktueller Stand: Wenn ein bestehendes Mitglied einen weiteren Zählpunkt
nachreichen will (z. B. neue PV-Anlage, weitere Verbrauchsstelle), muss
es entweder:
- den kompletten Public-Registrierungs-Flow nochmal durchlaufen (Doppelt-
  Antrag, Verein-Admin muss Duplikate mergen) **oder**
- den Verein-Admin manuell kontaktieren

Beide Pfade sind reibungsvoll und führen zu Inkonsistenzen.

## Skizze

Eine eigene Public-Strecke `/add-meter/<RC>` (oder vergleichbar), die:

1. **Mitgliedsnummer + Authentifizierungs-Hebel** abfragt (siehe offene
   Fragen unten)
2. Über Core-Sync das bestehende Mitglied identifiziert
3. Nur Zählpunkt-Daten erfasst (kein Wiederholen von Stammdaten, IBAN,
   Mandat etc. — die existieren ja schon)
4. Den neuen Zählpunkt als eigenständigen `metering_point`-Antrag
   anlegt, der dann vom Admin geprüft/genehmigt wird (oder leichtgewichtiger
   Auto-Approval-Pfad)

## Offene Fragen (zu klären in `/requirements`-Lauf)

- **Authentifizierung:** Mitgliedsnummer allein ist ratbar (vor allem bei
  Alphanumerik-Patterns). Was als zusätzlicher Hebel?
  - E-Mail-Bestätigung wie PROJ-31 (Token an die im Core hinterlegte
    Adresse)?
  - Geburtsdatum / letzte 4 Stellen IBAN?
  - Admin-mediated (Mitglied bekommt vom Admin einen Einladungs-Link)?
- **Auto-Import oder Admin-Review?** Zählpunkte sind heute reviewpflichtig
  (PROJ-2). Soll das für Nachmeldungen auch gelten oder direkt durchrutschen?
- **Adresse:** Erbt der neue Zählpunkt automatisch die Mitglieds-Adresse,
  oder muss eine abweichende Adresse eingegeben werden (PROJ-39)?
- **Tarif:** Wird der neue Zählpunkt automatisch dem Standard-Tarif der
  EEG zugewiesen, oder ist Tarif-Auswahl Teil des Flows (PROJ-27)?
- **SEPA-Mandat:** Bestehendes Mandat des Mitglieds wiederverwenden, oder
  ggf. neues Mandat erforderlich (B2B-Mandat-Konstellation)?
- **Direction-Wechsel:** Darf ein PRIVATE-Mitglied über diesen Flow eine
  GENERATION-Anlage nachmelden (Einkommens-Konsequenzen)? Oder nur
  gleichartige Zählpunkte?
- **EEG-Mitglieder ohne Onboarding-System-Eintrag:** Was, wenn das
  Mitglied nicht über Onboarding gekommen ist (manuell im Core erfasst)?
  Soll der Flow trotzdem funktionieren?
- **PROJ-59 (BgA/Hoheitsbereich-Vermerk im Anlagennamen):** Wenn der
  Nachmelde-Flow für ein Gemeinde-Mitglied (`member_type=municipality`)
  genutzt wird, sollte beim Anlagenname-Feld derselbe Hilfetext erscheinen
  wie im normalen Public-Form (Vermerk „BgA" bzw. „Hoheit" mit eintragen).
  Keine strukturelle Validierung erforderlich.

## Nächster Schritt

Bei tatsächlicher Aufnahme der Spec: `/requirements` mit dieser Datei
als Ausgangspunkt, um Auth-Modell + Scope sauber zu definieren.
