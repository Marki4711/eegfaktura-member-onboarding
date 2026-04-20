# Open Questions

Offene Punkte, die noch geklärt werden müssen, bevor sie als Feature spezifiziert werden können.

---

## OQ-1: Dokumente im Registrierungsformular

**Kontext:**
Im Einwilligungsbereich des Registrierungsformulars bestätigt das Mitglied, die Datenschutzerklärung gelesen zu haben. Daneben existieren weitere Dokumente, die einem neuen Mitglied vor oder bei der Registrierung zugänglich gemacht werden müssen — z.B.:

- Statuten der EEG
- Lieferantenverpflichtungen
- Datenschutzbestimmungen
- ggf. weitere rechtliche Dokumente

**Offene Fragen:**
- Welche Dokumente sind konkret erforderlich?
- Müssen alle EEGs dieselben Dokumente verwenden, oder gibt es EEG-spezifische Dokumente?
- Wie werden die Dokumente bereitgestellt? (Direktlink, Upload, statische URL, CMS?)
- Muss die Zustimmung zu jedem Dokument einzeln erfasst werden?
- Muss der Zeitpunkt der Zustimmung je Dokument gespeichert werden?

**Auswirkung auf bestehende Implementierung:**
Das Feld `privacy_version` und `privacy_accepted_at` deckt aktuell nur die Datenschutzerklärung ab. Bei Erweiterung auf mehrere Dokumente wäre ein eigenes Zustimmungsmodell notwendig.

**Status:** Ungeklärt — vor Implementierung mit Fachverantwortlichen abstimmen.
