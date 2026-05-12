# PROJ-29: IBAN-Eingabe mit visueller Gruppierung

## Status: Planned
**Created:** 2026-05-12
**Last Updated:** 2026-05-12

## Dependencies
- Requires: PROJ-1 (Public Registration) — `registration-form.tsx`
- Requires: PROJ-3 (Admin Frontend UI) — `admin-edit-form.tsx`
- Reuses: bestehende `MaskedInput`-Komponente (heute schon im Zählpunkt-Feld eingesetzt)

## Hintergrund

Das IBAN-Feld ist heute ein nackter `Input` (siehe [registration-form.tsx:760](src/components/registration-form.tsx#L760)). Banken und Behörden zeigen IBANs üblicherweise in Vierergruppen (`AT12 3456 7890 1234 5678`), was die Lesbarkeit erhöht und Tippfehler reduziert. Das Zählpunkt-Feld nutzt bereits `MaskedInput` mit derselben Logik — wir wollen denselben Stil auf IBAN anwenden.

## User Stories

- Als **Mitglied** möchte ich bei der IBAN-Eingabe Vierergruppen sehen, sodass ich Tippfehler beim Vergleich mit meinem Bankauszug schneller erkenne.
- Als **EEG-Admin** möchte ich im Bearbeitungsformular dieselbe Gruppierung sehen, sodass die UX zwischen Public- und Admin-Form konsistent ist.

## Acceptance Criteria

### Eingabe-Verhalten
- [ ] IBAN-Feld zeigt während der Eingabe Vierergruppen, getrennt durch ein Leerzeichen (z.B. `AT12 3456 7890 1234 5678`)
- [ ] Buchstaben (Ländercode) werden automatisch in Großbuchstaben umgewandelt (analog zur Zählpunkt-Maske)
- [ ] Max. Länge: 34 Zeichen netto (längste internationale IBAN) → 34 Zeichen + bis zu 8 Spaces = max. ~42 sichtbare Zeichen
- [ ] Paste aus Online-Banking funktioniert sowohl mit als auch ohne Spaces — Maske formatiert sofort um
- [ ] Tastatur-Navigation (Pfeiltasten, Backspace, Strg+A) verhält sich konsistent zum Zählpunkt-Feld (Standard-MaskedInput-Verhalten)

### Validierung
- [ ] `ibantools.isValidIBAN` validiert weiterhin den Inhalt — die Maske ist rein visuell. Der Validator akzeptiert IBANs sowohl mit als auch ohne Spaces; im Zweifel vor der Validierung Spaces strippen
- [ ] Server speichert IBAN **ohne** Spaces (kompakte Form)
- [ ] Server akzeptiert eingehende IBANs mit und ohne Spaces — Normalisierung im Handler/Service (`strings.ReplaceAll(iban, " ", "")` vor Persistenz)
- [ ] Error-Messages bleiben unverändert (`"Ungültige IBAN"`)

### Konsistenz und Wiederverwendung
- [ ] Selbe `MaskedInput`-Komponente wie das Zählpunkt-Feld — keine neue Komponente, keine neue Bibliothek
- [ ] Anwendung in `registration-form.tsx` **und** `admin-edit-form.tsx`
- [ ] PROJ-12 (SEPA-Mandat-PDF) und PROJ-14 (Firmen-SEPA): PDF-Generator nutzt weiterhin den kompakt gespeicherten Wert — keine Renderer-Anpassung nötig

### Tests
- [ ] Unit-Test (Vitest) für die IBAN-Eingabe: Tippen, Paste mit Spaces, Paste ohne Spaces, Großschreibung
- [ ] Backend-Test in `application_service_test.go`: eingehende IBAN mit Spaces wird beim Speichern normalisiert (in DB ohne Spaces)
- [ ] E2E-Test (Playwright): Registrierungsabschluss mit gepasteter IBAN inkl. Spaces

## Edge Cases

- IBANs unterschiedlicher Länder haben unterschiedliche Längen (15–34 Zeichen). Die Maske muss `lazy`-Mode unterstützen, sodass die Eingabe nach beliebiger Länge abbrechbar bleibt
- User markiert mit Shift+Pfeil über eine Gruppe und löscht → MaskedInput-Standardverhalten (wie beim Zählpunkt)
- Bestehende Anträge mit IBAN ohne Spaces in der DB → beim erneuten Öffnen im Admin-Edit-Formular wird die IBAN durch die Maske umformatiert dargestellt (rein optisch — Datenbank unverändert)
- Copy aus dem Feld: zu klären, ob copy mit oder ohne Spaces erfolgen soll. Default: mit Spaces (besser lesbar)

## Technical Requirements

- **Sicherheit:** keine Auswirkung — UX-Layer, Validation und Speicherung unverändert (parametrisierte Queries, ibantools-Check bleibt)
- **Performance:** vernachlässigbar — `imask` ist über MaskedInput bereits geladen, keine zusätzlichen Bytes im Bundle
- **Konsistenz:** identische Komponente und Konfigurationsmuster wie Zählpunkt-Feld
- **Rückwärtskompatibilität:** keine Datenmigration nötig, gespeichertes Format bleibt unverändert

## Notes

- Library: `imask` ist über `MaskedInput` bereits eingebunden (siehe Zählpunkt-Feld)
- Da IBAN-Länge variiert, kommt eine dynamische bzw. lazy Maske zum Einsatz (z.B. `"aa00 0000 0000 0000 0000 0000 0000 0000 00"` mit `lazy: true`); die genaue Maskendefinition wird im Architecture-Schritt fixiert
- Security-Review **nicht** erforderlich: keine neuen Endpoints, keine Auth-Änderung, kein Schema-Update
- Spec ist klein genug, dass `/architecture` und `/frontend` in einem Schritt zusammenfallen können

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
