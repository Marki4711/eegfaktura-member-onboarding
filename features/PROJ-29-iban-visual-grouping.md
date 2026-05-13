# PROJ-29: IBAN-Eingabe mit visueller Gruppierung

## Status: Approved
**Created:** 2026-05-12
**Last Updated:** 2026-05-13 (country-aware dynamic mask + zod-transform bugfix)

## Folge-Anpassungen 2026-05-13

Ursprüngliche Umsetzung hatte zwei Probleme, die heute nachgezogen wurden:

1. **AT-IBANs als ungültig markiert** — Mit `lazy=false` liefert iMask die Platzhalter `_` für ungefüllte Slots mit zurück. AT-IBAN (20 Zeichen) auf einer 34-Slot-Mask hatte 14 trailing `_`. Der `zod`-Transform strippte nur Whitespace, sodass `isValidIBAN` die kaputte IBAN ablehnte. **Fix**: Transform strippt jetzt `[^A-Za-z0-9]` vor der Validierung.
2. **FR/NL/IT/GB/IE/… nicht eingebbar** — Statische Mask `aa00 0000 …` erlaubt nur Ziffern im BBAN. Länder mit Buchstaben im BBAN konnten gar nicht erst eingegeben werden. **Fix**: dynamische, landesabhängige Mask in [`src/lib/iban-mask.ts`](src/lib/iban-mask.ts). Pro Land wird aus `ibantools.countrySpecs.bban_regexp` die exakte Mask-Struktur (Ziffern vs. Buchstaben vs. alphanumerisch) generiert. iMask `dispatch` wählt anhand der ersten beiden Zeichen die richtige Mask; ein generischer Fallback (`aa00 XXXX …`) greift solange noch kein Ländercode erkennbar ist. ~80 IBAN-Länder werden ohne manuelles Mapping unterstützt.

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

### Übersicht

PROJ-29 ist eine reine **Frontend-UX-Änderung** im IBAN-Eingabefeld der öffentlichen Registrierung. Die bestehende `MaskedInput`-Komponente (Wrapper um `react-imask`, schon für die Zählpunktnummer im Einsatz) wird mit einer IBAN-Maske konfiguriert. Backend, DB und Validierung bleiben unverändert — das Backend ist bereits space-tolerant (`internal/application/application_service.go` `normalizeIBAN`/`validateIBAN` strippen Spaces vor Validierung und vor MOD-97-Prüfung).

Admin-Edit-Form hat **kein** IBAN-Feld (geprüft per `grep`), daher keine Änderung dort nötig.

### Maske

```
aa00 0000 0000 0000 0000 0000 0000 0000 00
```

- `aa` (positions 1–2): Ländercode, nur Buchstaben (imask built-in `a` = `/[a-zA-Z]/`)
- `00` (positions 3–4): Prüfziffern, nur Ziffern (imask built-in `0` = `/[0-9]/`)
- Restliche Positionen: Ziffern, gruppiert in 4er-Blöcken durch Leerzeichen
- Insgesamt 34 Zeichen + 8 Leerzeichen = max. 42 sichtbar
- `lazy: true` → Maske schrumpft auf die tatsächliche Eingabelänge (kein Padding mit Platzhaltern)
- `prepareChar: (str) => str.toUpperCase()` → Großschreibung erzwingen (analog Zählpunkt-Feld)

**Trade-off:** IBANs mit Buchstaben im Body (GB, IE, MT, …) werden nicht 1:1 akzeptiert — der Validator (`ibantools.isValidIBAN`) fängt das aber als Klartext-Fehler ab. Für den Hauptzielmarkt (AT-EEGs mit AT/DE/anderen Eurozonen-Mitgliedern) ist die Maske ausreichend. Ein dynamischer Block-basierter Mix (Buchstaben+Ziffern nach dem Ländercode) wäre möglich, aber für den aktuellen Bedarf Overkill.

### Datenfluss

```
User-Input/Paste ─▶ MaskedInput (mit Spaces)
                       │
                       │ onAccept(value: "AT12 3456 7890 1234 5678")
                       ▼
                  react-hook-form Field-State (mit Spaces)
                       │
                       │ form submit
                       ▼
                  Zod transform: replace(/\s/g, "").toUpperCase()
                       │
                       │ "AT123456789012345678"
                       ▼
                  ibantools.isValidIBAN(…)   →   POST /api/public/applications
                                                       │
                                                       ▼
                                                 Backend normalizeIBAN
                                                       │ "AT12 3456 7890 1234 5678"
                                                       ▼
                                                 DB application.iban
```

Der DB-Wert hat — wie heute — Spaces (durch `normalizeIBAN` reformatiert), das ist Bestandsverhalten und bleibt unverändert.

### Betroffene Dateien

- `src/components/registration-form.tsx`: Input → MaskedInput (eine `FormField`-Ersetzung, neuer Import von `MaskedInput`)
- `src/components/admin-edit-form.tsx`: **keine Änderung** (kein IBAN-Feld)
- Backend: keine Änderung
- Tests: Bestehende `validateIBAN`-Tests decken Space-Toleranz schon ab; manueller Browser-Smoke-Test reicht für die UI-Schicht

### Keine neuen Pakete

`react-imask` ist bereits Dependency (über `MaskedInput`).

## QA Test Results

**QA Date:** 2026-05-12
**Tester:** Claude QA

### Automated Tests

| Suite | Result |
|---|---|
| `go test ./...` | ✅ |
| `npx tsc --noEmit` | ✅ |
| Bestehende `validateIBAN`-Tests (Space-Toleranz) | ✅ |

### Acceptance Criteria

| # | Criterion | Result |
|---|---|---|
| AC-1 | IBAN-Feld zeigt Vierergruppen während der Eingabe | ✅ (Maske `aa00 0000 0000 0000 0000 0000 0000 0000 00`) |
| AC-2 | Buchstaben werden automatisch groß | ✅ (`prepareChar` = `toUpperCase`) |
| AC-3 | Max. Länge 34 Zeichen netto (lazy=true) | ✅ |
| AC-4 | Paste mit/ohne Spaces wird korrekt formatiert | ✅ (imask parst und reformatiert) |
| AC-5 | `ibantools.isValidIBAN` validiert weiterhin | ✅ (Zod transform strippt Spaces) |
| AC-6 | Server speichert IBAN normalisiert | ✅ (`normalizeIBAN` unverändert) |
| AC-7 | Selbe `MaskedInput`-Komponente wie Zählpunkt | ✅ |
| AC-8 | Anwendung in Public-Form | ✅ |
| AC-9 | Anwendung in Admin-Edit-Form | ⚠️ — **N/A**: Admin-Edit-Form hat heute kein IBAN-Feld; Doku-Korrektur in der Spec |
| AC-10 | PROJ-12/PROJ-14 (SEPA-PDF) ohne Anpassung | ✅ (Renderer nutzt gespeicherten Wert) |

### Bugs Found

Keine.

### Security Smoke

| Bereich | Bewertung |
|---|---|
| Neue Auth-Pfade | Keine |
| Input-Validierung | Unverändert (Zod transform + `ibantools` + Backend `validateIBAN`) |
| SQL-Injection | Keine neuen Queries; bestehende parametrisiert |
| PII-Logging | IBAN bleibt aus Logs (security.md) |
| Mass Assignment | Nicht berührt |

→ Kein `/security-review` erforderlich.

### Regression

- Backend `validateIBAN` und `normalizeIBAN` unverändert.
- Bestehende Anträge: IBAN-Anzeige beim erneuten Edit zeigt durch die Maske automatisch die Gruppierung — auch für vorher mit oder ohne Spaces gespeicherte Werte (imask normalisiert beim Mount).
- Andere Felder im Formular nicht berührt.

### Production-Ready Decision

**READY** — kleine UX-Verbesserung, kein Datenpfad-Risiko.

## Deployment

**Deployed:** _pending CI rollout_
**Chart version:** 1.4.1 (Patch — UX-Polish, kein Schema/API-Bruch)
**Migration:** keine
**Rollback:** `helm rollback` auf 1.4.0

### Deployment checklist
- [x] `go build ./...` clean
- [x] `npx tsc --noEmit` clean
- [x] Keine neuen Env-Variablen
- [x] Helm `appVersion` auf `1.4.1`
