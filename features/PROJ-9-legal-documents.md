# PROJ-9: EEG-spezifische Rechtsdokumente mit granularer Zustimmung

## Status: Planned
**Created:** 2026-04-21
**Last Updated:** 2026-04-21

## Dependencies
- Requires: PROJ-1 (Public Registration) — Zustimmung erfolgt im Registrierungsformular
- Requires: PROJ-2 (Admin Review) — Admin verwaltet die Dokumentenliste der EEG
- Requires: PROJ-5 (Keycloak-secured Admin Area) — Verwaltung nur für authentifizierte Admins

## User Stories

- Als EEG-Administrator möchte ich eine Liste von Rechtsdokumenten (AGB, Datenschutzerklärung, Statuten) für meine EEG hinterlegen können, damit Neumitglieder beim Beitritt gezielt zustimmen.
- Als EEG-Administrator möchte ich festlegen können, ob die Zustimmung zu einem Dokument verpflichtend oder freiwillig ist, damit ich rechtliche Anforderungen abbilden kann.
- Als EEG-Administrator möchte ich die Reihenfolge der angezeigten Dokumente steuern können, damit wichtige Dokumente zuerst erscheinen.
- Als Mitglied möchte ich für jedes Rechtsdokument eine eigene Checkbox sehen und den Link direkt öffnen können, damit ich weiß, womit ich zustimme.
- Als Mitglied möchte ich zusätzlich zur EEG-spezifischen Liste immer die zentrale Datenschutzerklärung des Tool-Betreibers sehen und zustimmen, damit der Betrieb des Tools transparent ist.
- Als Admin möchte ich im Antrag nachvollziehen können, welchen Dokumenten das Mitglied zugestimmt hat (Titel, URL, Zeitstempel), damit die Zustimmung nachweisbar ist.

## Acceptance Criteria

- [ ] Pro EEG kann eine geordnete Liste von Dokumenten angelegt werden (Titel, URL, Pflicht ja/nein, Reihenfolge)
- [ ] Die Dokumentenliste wird über den `/api/public/registration/{rc_number}` Endpunkt mitgeliefert
- [ ] Im Registrierungsformular wird pro EEG-Dokument eine eigene Checkbox mit verlinktem Titel angezeigt
- [ ] Pflichtdokumente blockieren das Absenden des Formulars wenn nicht angehakt
- [ ] Die zentrale Datenschutzerklärung des Tool-Betreibers wird immer angezeigt und ist immer Pflicht
- [ ] Beim Speichern des Antrags wird pro zugestimmtem Dokument gespeichert: Titel, URL, Zeitstempel
- [ ] Die gespeicherten Zustimmungen sind in der Admin-Detailansicht eines Antrags sichtbar
- [ ] Ein Admin kann Dokumente seiner EEG(s) hinzufügen, bearbeiten, löschen und sortieren
- [ ] Das Löschen eines Dokuments beeinflusst keine bereits gespeicherten Zustimmungen

## Edge Cases

- Was passiert, wenn eine EEG keine eigenen Dokumente hinterlegt hat? → Nur die zentrale Datenschutzerklärung wird angezeigt, Formular bleibt funktionsfähig.
- Was passiert, wenn ein Dokument-Link nicht erreichbar ist? → Das Formular zeigt den Link trotzdem an; die Erreichbarkeit wird nicht geprüft.
- **Offener Punkt — Dokumentenversionen:** Da nur die URL gespeichert wird, kann nachträglich nicht nachgewiesen werden, welche Version des Dokuments zum Zeitpunkt der Zustimmung abrufbar war. Mögliche Lösungsansätze: (1) Inhalt des Dokuments zum Zeitpunkt der Einreichung archivieren, (2) EEGs verpflichten, versionierte URLs zu verwenden (z.B. `/datenschutz-v2.pdf`), (3) Hash des Dokumenteninhalts speichern. Dieser Aspekt muss vor der Implementierung entschieden werden.
- Was passiert, wenn ein Dokument nach Einreichung eines Antrags geändert oder gelöscht wird? → Bereits gespeicherte Zustimmungen bleiben unverändert (Snapshot zum Zeitpunkt der Einreichung).
- Was passiert, wenn ein optionales Dokument nicht angehakt wird? → Antrag kann trotzdem eingereicht werden; keine Zustimmung wird für dieses Dokument gespeichert.
- Was passiert, wenn die URL eines Dokuments sehr lang ist? → URL wird vollständig gespeichert, im Formular aber nur der Titel verlinkt angezeigt.

## Technical Requirements

- Zustimmungen werden als unveränderlicher Snapshot gespeichert (Titel + URL zum Zeitpunkt der Einreichung)
- Die zentrale Datenschutzerklärung ist nicht in der Datenbank konfiguriert, sondern im Code/Konfiguration hinterlegt
- Reihenfolge der Dokumente ist explizit steuerbar (z.B. über ein `sort_order` Feld)
- Maximale Anzahl Dokumente pro EEG: reasonable limit (z.B. 10)

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
