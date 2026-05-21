# PROJ-55 — Nachmelden von Zählpunkten anhand der Mitgliedsnummer

**Status:** Planned (Idea — Tester-Vorschlag, noch ohne ausformulierte Requirements)
**Erstellt:** 2026-05-21
**Quelle:** Tester-Feedback
**Abhängigkeiten:** PROJ-32 (EEG-Stammdaten + Mitgliederliste aus Core), evtl. PROJ-31 (E-Mail-Bestätigung als Auth-Hebel)

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

## Nächster Schritt

Bei tatsächlicher Aufnahme der Spec: `/requirements` mit dieser Datei
als Ausgangspunkt, um Auth-Modell + Scope sauber zu definieren.
