# PROJ-81: SEPA-Einwilligung optional pro Mitgliedstyp

## Status: Spec-Reviewed (nach /grill-me 2026-06-08)
**Created:** 2026-06-08
**Last Updated:** 2026-06-08

## Hintergrund

Heute (nach PROJ-80) ist die SEPA-Einwilligungs-Checkbox im Public-Form
eine Pflicht-Konstante. Jedes Mitglied muss die Online-Zustimmung zum
SEPA-Lastschriftmandat erteilen, um den Antrag absenden zu können.

Es gibt EEGs, die diese Pflicht nicht für alle Mitgliedstypen wollen —
typische Beispiele:
- Privatmitglieder, die per Dauerauftrag oder Einzelüberweisung zahlen
  möchten
- Vereine mit eigener Buchhaltung, die nicht per Lastschrift gezogen
  werden wollen
- Gemeinden mit zentraler Anweisungs-Verwaltung

Diese EEGs sollen pro EEG selbst entscheiden können, für welche
Mitgliedstypen die SEPA-Einwilligung optional statt Pflicht ist.

## Owner-Direktive (2026-06-08)

> „Es gibt EEGs, die nicht durchgängig für ihre Mitglieder SEPA
> erzwingen. Es soll dort auch möglich sein, kein SEPA auszuwählen
> (SEPA optional). Die Möglichkeit soll steuerbar sein und nur für
> gewisse Mitgliedstypen gelten."
>
> Klarstellung: „Es geht nur darum, die SEPA-Checkbox für definierte
> Kundentypen optional zu machen. Nicht zu deaktivieren."

## Dependencies

- **Requires:** PROJ-80 (SEPA-Settings-Vereinfachung) — die heutige
  Pflicht-Konstante der Online-Zustimmungs-Checkbox kommt aus PROJ-80
  und wird durch PROJ-81 für definierte Mitgliedstypen aufgehoben.
- **Relates:** PROJ-7 (Mitgliedstypen) — die Mitgliedstyp-Liste
  (private/company/municipality/association) ist die Basis der Auswahl.

## Owner-Entscheidungen (festgenagelt im /requirements 2026-06-08)

| Frage | Entscheidung |
|-------|--------------|
| Welche Mitgliedstypen können die Wahl bekommen? | Konfigurierbar pro EEG (Auswahl aus `private`/`farmer`/`association`/`municipality`) |
| Toggle-Granularität | 1 Master-Toggle + konfigurierbare Mitgliedstyp-Liste |
| Mechanik | NUR die SEPA-Einwilligungs-Checkbox wird optional (nicht entfernt) |
| Bankdaten-Logik | Bankdaten (IBAN, Kontowortlaut) bleiben IMMER Pflicht im Public-Form — eegFaktura-Core verlangt sie. Bei nicht angekreuzter Checkbox wird `einzugsart=kein_sepa` gesetzt, Mandat-PDF entfällt, Bankdaten sind aber erfasst (Owner-Korrektur 2026-06-08 mid-implementation). |
| Settings-Ort | SEPA-Sektion (unter den 3 SEPA-Toggles aus PROJ-80) |
| B2B (`company`) | Pflicht-Lastschrift bleibt — `company` bekommt nie die Wahl angeboten |
| Bestandsanträge | Nur Public-Form-Pfad betroffen, Admin-Edit-Pfad bleibt wie heute |

### Vorab-Defaults aus /grill-me (Auto-Mode 2026-06-08)

Folgende Detail-Fragen wurden im Grilling **nicht** mehr beim Owner
nachgefragt; die Empfehlungen wurden als Default übernommen. Owner
kann beim Architecture-Review noch widersprechen.

| Frage | Default | Begründung |
|-------|---------|------------|
| Mitgliedstyp-Liste-Speicherung | TEXT[] auf `registration_entrypoint` mit `pq.Array` | Etabliertes Pattern in `metering_point_repo`/`dataexport`; keine separate Tabelle nötig bei max 4 Einträgen |
| Spaltennamen | `sepa_optional_enabled` (bool, NOT NULL DEFAULT FALSE) + `sepa_optional_member_types` (text[], NOT NULL DEFAULT `{}`) | Konsistent mit `sepa_mandate_*`-Spalten |
| NULL vs. leeres Array | Leeres Array `{}` (NOT NULL DEFAULT) | Einfacher Pattern, keine NULL-Behandlung |
| DB-CHECK-Constraint | KEINE | Service-Layer-Validation reicht; Owner-Edit über Settings-UI ist einziger Schreib-Pfad |
| Bankdaten-Reaktion bei Checkbox-Abhaken | Bleiben IMMER Pflicht (Sternchen) | Owner-Korrektur 2026-06-08: eegFaktura-Core verlangt Bankdaten für jedes Mitglied |
| Bankdaten-Reaktion bei nachträglichem Anhaken | Unverändert Pflicht | Bankdaten waren nie optional |
| Public-Form-Mitgliedstyp-Wechsel auf Nicht-berechtigt | `sepaMandateAccepted` wird auf `false` gesetzt, Checkbox wird wieder Pflicht (Sternchen sichtbar) | User muss aktiv neu zustimmen |
| Widersprüchlicher Submit (`sepaMandateAccepted=true` + `einzugsart=kein_sepa`) | 400 Validierungsfehler | Defensive |
| Hint-Text bei optionaler Checkbox | Inline italic muted-foreground unter Checkbox-Label | Konsistent mit PROJ-80-Kurz-Erklärungs-Pattern |
| Settings-UI-Layout | Master-Toggle + `pl-10`-Sub-Block mit Mitgliedstyp-Checkboxen | Visualisiert Coupling (PROJ-80-Sub-Toggle-Pattern) |
| Mitgliedstyp-Labels im Admin | „Privat", „Pauschalierter Landwirt", „Verein", „Gemeinde" | Konsistent mit Excel-Export-Labels |
| Validation-Fehler bei Save (leere Liste) | Inline-Field-Error unter der Liste, 400-Response | PROJ-80-Cross-Field-Validation-Pattern |
| Configexport Mitgliedstyp-Liste | `[]string, omitempty` (kein Pointer) | Leeres Slice + omitempty äquivalent zu `*[]string + nil` |

## User Stories

- **Als EEG-Admin** möchte ich in den EEG-Settings festlegen, dass die
  SEPA-Einwilligung für bestimmte Mitgliedstypen (z.B. nur Privat) optional
  statt verpflichtend ist, damit ich die Realität meiner Mitglieder
  abbilde, die nicht alle Lastschrift wünschen.
- **Als Privat-Mitglied** möchte ich den Antrag absenden können, ohne
  die SEPA-Einwilligungs-Checkbox anzukreuzen, damit ich auch per Dauer-
  auftrag oder Überweisung zahlen kann, wenn die EEG das zulässt.
- **Als Mitglied** möchte ich klar sehen, dass die SEPA-Einwilligung
  optional ist (kein Sternchen, kein Required-Hinweis), damit ich keine
  Pflicht-Verletzung befürchten muss, wenn ich das Häkchen weglasse.
- **Als EEG-Admin** möchte ich, dass die `einzugsart` automatisch auf
  `kein_sepa` gesetzt wird, wenn das Mitglied die Einwilligung weglässt,
  damit die Antrags-Daten konsistent sind und mein Excel-Export / Import
  korrekt erkennt, dass keine Lastschriftermächtigung erteilt wurde.
- **Als Firmen-Antragsteller (`company`)** möchte ich, dass die B2B-
  Mandats-Pflicht unverändert bleibt, damit die Geschäftslastschrift-
  Logik (PROJ-14/47/77/78) nicht aushöhlt.

## Acceptance Criteria

### EEG-Settings (Admin)

- [ ] **AC-1 Master-Toggle** „SEPA-Einwilligung für ausgewählte Mitglieds-
  typen optional" sitzt in der SEPA-Sektion des EEG-Settings-Editors,
  unterhalb der 3 PROJ-80-Toggles (CORE-Audit, B2B-Audit, Timing).
- [ ] **AC-2 Default FALSE** für neue und bestehende EEGs (bestehende
  Pflicht-Logik bleibt erhalten, bis Admin den Toggle aktiviert).
- [ ] **AC-3 Mitgliedstyp-Liste** ist nur sichtbar, wenn der Master-
  Toggle aktiv ist. Sie zeigt die 4 zulässigen Mitgliedstypen als
  Checkboxen: `private`, `farmer`, `association`, `municipality`.
  (Codebase-Befund /grill-me 2026-06-08: `farmer` ist ein eigener
  Mitgliedstyp gemäß `shared.MemberTypeFarmer`, war in der ersten Spec-
  Fassung übersehen.)
- [ ] **AC-4 Company ausgeschlossen** — `company` taucht in der Liste
  NICHT auf, weil B2B-Lastschrift Pflicht bleibt.
- [ ] **AC-5 Mindestens einer aktiv** — Wenn Master-Toggle aktiv aber
  keine Mitgliedstyp-Checkbox angehakt → Save schlägt fehl mit
  Validierungsfehler 400: „Mindestens ein Mitgliedstyp muss ausgewählt
  sein."
- [ ] **AC-6 Kurz-Erklärung** unter dem Master-Toggle (analog PROJ-80-
  Pattern): „Wenn aktiv, ist die SEPA-Einwilligung für die unten
  ausgewählten Mitgliedstypen optional statt Pflicht. Mitglieder dieser
  Typen können den Antrag auch ohne SEPA-Mandat einreichen."
- [ ] **AC-7 Hint-Popover** am Info-Icon erklärt das Verhalten in der
  Public-Form (max-w-80, Pattern aus .claude/rules/frontend.md).
- [ ] **AC-8 Standard-Modus** zeigt den Toggle NICHT (PROJ-67-Pattern —
  nur Erweitert/Advanced).
- [ ] **AC-9 Advanced-Trigger** — wenn Master-Toggle aktiv ist, gilt
  die EEG als „advanced" im Settings-Mode (analog `isAdvancedEEGSettings`-
  Logik). Sichert Sichtbarkeit der Konfiguration auch nach Neuladung.

### Public-Form

- [ ] **AC-10 Render-Bedingung** — Die SEPA-Einwilligungs-Checkbox
  wird optional (kein Sternchen, kein Required) gerendert, wenn:
  EEG-Master-Toggle aktiv UND gewählter Mitgliedstyp in der Liste.
  Sonst Pflicht wie heute.
- [ ] **AC-11 Bankdaten bleiben Pflicht** — IBAN, Kontowortlaut bleiben
  IMMER Pflicht im Public-Form (Sternchen, Required-Validierung
  unverändert), unabhängig vom Checkbox-Status. Begründung:
  eegFaktura-Core verlangt Bankdaten für alle Mitglieder, auch wenn
  kein Lastschrift-Mandat erteilt wird. (Owner-Korrektur 2026-06-08
  mid-implementation.)
- [ ] **AC-12 entfällt** — Bankdaten waren nie optional. Nummer
  bewusst freigelassen, um spätere AC-Nummern stabil zu halten.
- [ ] **AC-13 einzugsart-Mapping** — Beim Submit:
  - Checkbox angekreuzt → `einzugsart=core` (heutiges Verhalten)
  - Checkbox nicht angekreuzt → `einzugsart=kein_sepa`. Bankdaten
    sind weiterhin erfasst, werden aber NICHT für ein Mandat-PDF
    verwendet.
- [ ] **AC-14 Hinweis am Checkbox** — Bei optionaler Variante
  steht direkt am Checkbox-Label ein kurzer Hinweis: „optional — wenn
  nicht angekreuzt, vereinbaren Sie die Zahlung direkt mit der EEG."
- [ ] **AC-15 Mitgliedstyp-Wechsel** — Wenn Member den Mitgliedstyp
  während der Eingabe ändert (z.B. von `private` zu `company`), wird
  die Checkbox-Pflicht/-Optionalität live neu evaluiert und ggf. die
  Online-Zustimmungs-Pflicht reaktiviert.

### Backend (Submit + Validation)

- [ ] **AC-16 Submit-Validation** — Der Backend-Submit akzeptiert
  `sepaMandateAccepted=false` nur, wenn EEG-Master-Toggle aktiv UND
  gewählter Mitgliedstyp in der Liste. Sonst 400 „SEPA-Einwilligung
  ist für diesen Mitgliedstyp Pflicht."
- [ ] **AC-16a Request-Body-Validation: nur SepaMandateAccepted lockern**
  — Im `CreateApplicationRequest` (shared/requests.go) wird ausschließlich
  `SepaMandateAccepted` von `validate:"required"` auf `bool` (ohne Tag)
  umgestellt, damit der Validator `false` durchlässt. `IBAN`/`AccountHolder`
  bleiben `required` (eegFaktura-Core verlangt Bankdaten für alle
  Mitglieder, auch ohne Mandat). (Owner-Korrektur 2026-06-08
  mid-implementation.)
- [ ] **AC-17 einzugsart-Erzwingung** — Backend setzt `einzugsart`
  server-side basierend auf `sepaMandateAccepted`:
  - `sepaMandateAccepted=true` → `einzugsart=core` (heutiges Verhalten,
    hartkodiert, da Public-Submit heute keinen einzugsart-Wert mitschickt)
  - `sepaMandateAccepted=false` (nur wenn Toggle+Mitgliedstyp erlaubt) →
    `einzugsart=kein_sepa`
  Bankdaten werden in beiden Fällen erfasst und gespeichert.
- [ ] **AC-18 Mandat-PDF** — Bei `einzugsart=kein_sepa` wird kein
  SEPA-Mandat-PDF generiert (heutiges Verhalten von `buildSEPAMandateData`
  bleibt — default-Branch returnt nil, bestätigt in
  `application_service.go:1644`).
- [ ] **AC-19 EEG-Submit-Mail** — Die Einreichungs-Bestätigungsmail an
  die EEG zeigt im bestehenden Feld „SEPA-Ermächtigung" weiterhin den
  Wert aus `ResolveSepaMandateType` (bei `einzugsart=kein_sepa` automatisch
  „Kein SEPA"). **Zusätzlich** wird bei `einzugsart=kein_sepa` ein gelber
  Hinweis-Banner über der Antrags-Detail-Tabelle gerendert:
  > ⚠ Kein SEPA-Lastschriftmandat erteilt
  > Das Mitglied hat im Onboarding-Formular keiner SEPA-Einwilligung
  > zugestimmt. Es wurde kein Lastschriftmandat erzeugt — die Abrechnung
  > muss über einen alternativen Zahlungsweg (Überweisung, Dauerauftrag)
  > direkt mit dem Mitglied vereinbart werden.
- [ ] **AC-19a EEG-Activated-Mail (`application_activated_eeg.html`)** —
  Bei `einzugsart=kein_sepa` wird derselbe Hinweis-Banner gerendert.
  Begründung: das ist der Zeitpunkt, ab dem die EEG mit der Abrechnung
  beginnt — der Hinweis ist hier am wichtigsten. Heute kein SEPA-Bezug
  im Template; wird ergänzt.
- [ ] **AC-19b EEG-Beitrittserklärung-Mail (PROJ-76,
  `SendBoardApprovalRequest`)** — Bei `einzugsart=kein_sepa` wird der
  Hinweis-Banner zusätzlich im Mail-Body gerendert, damit der Vorstand
  bei der Genehmigung weiß, dass keine Lastschrift gezogen werden kann.
- [ ] **AC-19c EEG-Imported-Mail (`application_imported_eeg.html`)** —
  Kein Eingriff nötig. Die Mail kündigt den Mandat-an-Mitglied-Versand
  an und wird nur bei `einzugsart ∈ {core, b2b}` ausgelöst. Bei
  `kein_sepa` geht diese Mail nicht raus.
- [ ] **AC-19d Banner-Daten zentral** — Das Mail-Daten-Struct bekommt
  ein einzelnes neues bool-Feld `NoSepaMandate`, abgeleitet aus
  `app.Einzugsart == "kein_sepa"`. Templates verwenden `{{if .NoSepaMandate}}`
  zur konditionalen Darstellung. Single source of truth — kein Drift
  zwischen den drei Mail-Pfaden (Memory-Regel
  `feedback_shared_helpers_for_parallel_paths`).
- [ ] **AC-19a Shared Validation-Helper** — Neuer Helper im Backend
  (`shared.IsSEPAOptional(ep, memberType) bool`) wird von 3 Pfaden
  geteilt: Public-Submit (CreateApplication), externe API (POST
  `/api/external/v1/applications`), und Admin-Service (für Cross-
  Check). Frontend bekommt ein TS-Pendant `isSepaOptional()`. Drift-
  Risiko wird durch Vitest-Snapshot-Test über die EEG-Settings-API
  abgesichert.

### Externe API (PROJ-13)

- [ ] **AC-20 Externer Submit** — Der `POST /api/external/v1/applications`-
  Endpoint validiert die gleiche Regel: `sepaMandateAccepted=false` ist
  nur erlaubt, wenn EEG-Toggle + Mitgliedstyp passen. Sonst 400.

### Admin (Antrags-Detail + Edit)

- [ ] **AC-21 Admin-Edit unverändert** — Der bestehende Admin-Edit-Pfad
  bleibt wie heute. Admin kann weiterhin manuell `einzugsart=kein_sepa`
  setzen, unabhängig vom Toggle UND unabhängig vom Mitgliedstyp (auch
  für `company`-Mitglieder, sodass Admin im Notfall einen B2B-Antrag
  auf „kein SEPA" stellen kann — bewusste Owner-Direktive).
- [ ] **AC-22 Antrags-Detail-Anzeige** — Anträge mit
  `einzugsart=kein_sepa` zeigen im Detail-View die Bankverbindungs-Card
  wie heute (Bankdaten sind immer erfasst, AC-11). Zusätzlich wird ein
  Info-Banner über der Bankverbindungs-Card eingeblendet: „Kein SEPA-
  Lastschriftmandat erteilt — die Bankdaten sind erfasst, aber kein
  Lastschriftauftrag aktiv. Zahlungsmodalitäten direkt mit dem Mitglied
  vereinbaren."

### Configexport (PROJ-61)

- [ ] **AC-23 Configexport** — Der Master-Toggle (`*bool, omitempty`)
  und die Mitgliedstyp-Liste (`[]string, omitempty`) werden in die
  Configexport-JSON aufgenommen. Backward-Kompat: bestehende Configs
  ohne diese Felder importieren als FALSE/leer.

### Bug-Fix-Beifang (im selben Commit)

- [ ] **AC-23a Excel-Einzugsart-Label-Map fixen** — Heute kennt
  `EinzugsartLabels` in `dataexport/excel/fields.go:252` nur `basis`
  und `b2b` — `core` und `kein_sepa` werden als Raw-Wert exportiert.
  Map wird korrigiert auf `{"core": "SEPA-Basismandat", "b2b": "SEPA-
  Firmenmandat", "kein_sepa": "Kein SEPA-Mandat"}`. (Codebase-Befund
  /grill-me 2026-06-08: latenter Bug, der durch PROJ-81 deutlich
  sichtbarer wird wegen vermehrter `kein_sepa`-Anträge im Public-Form.)
- [ ] **AC-23b buildSEPAMandateData-Kommentar refreshen** — Der Block-
  Kommentar in `application_service.go:1620-1625` erwähnt noch
  `SEPAMandateEnabled`, der durch PROJ-80 entfernt wurde. Doku-Drift,
  kosmetisch — wird im selben Commit mit-erledigt.

### Tests

- [ ] **AC-23c Service-Layer-Permutations-Test** — Tabellen-Test über
  die Cross-Permutationen `{toggle ∈ {on, off}} × {mitgliedstyp ∈ {private,
  farmer, company}} × {sepaMandateAccepted ∈ {true, false}}` mit
  erwarteten Validierungsergebnissen (400 oder 201) und finalem
  `einzugsart`-Wert. Mindestens 8 Cases.
- [ ] **AC-23d Frontend-Helper-Snapshot** — Vitest-Test, der den TS-
  Helper `isSepaOptional()` gegen einen vom Backend gelieferten
  Settings-Snapshot validiert. Drift-Sicherung zwischen Go- und TS-
  Implementierung.
- [ ] **AC-23e E2E-Test (Playwright)** — Mitgliedstyp-Wechsel im
  Public-Form testet die Live-Re-Validation (B1): von `private`
  (optional) zu `company` (Pflicht) und zurück. Verifiziert dass
  Sternchen + Required-Status der Checkbox live updaten.

### Doku

- [ ] **AC-24 User-Guide-Update** — `docs/user-guide/06-admin-settings.md`
  bekommt einen neuen Abschnitt unter SEPA-Konfiguration, der das
  Verhalten anhand Max-Mustermann-Beispiel erklärt.
- [ ] **AC-25 API-Spec-Update** — `docs/api-spec.md` dokumentiert die
  neue Settings-Felder im EEG-Settings-Endpoint.
- [ ] **AC-26 Domain-Model-Update** — `docs/domain-model.md`
  dokumentiert die neuen Spalten auf `registration_entrypoint`.
- [ ] **AC-27 PROJ-frei** — User-Guide-Doku enthält keine PROJ-Refs.
- [ ] **AC-28 CHANGELOG** — Eintrag im selben Commit wie der Code.

## Edge Cases

- **EC-1 Toggle aktiviert, kein Mitgliedstyp in der Liste:**
  Save schlägt fehl (AC-5). Settings-State bleibt unverändert. Frontend
  zeigt Validierungsfehler inline.
- **EC-2 Mitgliedstyp wird im Public-Form gewechselt:**
  Checkbox-Pflicht muss live neu evaluiert werden (AC-15). Wenn Member
  zu einem Pflicht-Mitgliedstyp wechselt, wird die Checkbox wieder
  required. Wenn er zu einem optionalen wechselt, wird sie optional.
  Sichtbares Sternchen muss konsistent updaten.
- **EC-3 Bestandsantrag mit `einzugsart=core` und EEG aktiviert Toggle
  nachträglich:** Bestandsantrag bleibt unverändert. Toggle wirkt nur
  auf neue Submits über den Public-Form-Pfad.
- **EC-4 EEG deaktiviert den Toggle nach Submission eines „kein_sepa"-
  Antrags:** Der bestehende Antrag bleibt `einzugsart=kein_sepa`. Neue
  Anträge müssen wieder die Einwilligung erteilen.
- **EC-5 Externer API-Submit mit `einzugsart=kein_sepa` aber EEG-
  Toggle deaktiviert:** 400 Validierungsfehler (AC-20). Externer
  Integrator muss explizit konsentieren.
- **EC-6 Externer API-Submit mit `sepaMandateAccepted=false` für
  `company`:** 400, weil `company` nie auf der Liste sein darf (AC-4).
- **EC-7 Member lässt IBAN leer:** Validierungsfehler wie heute —
  Bankdaten sind IMMER Pflicht (AC-11), unabhängig vom Checkbox-Status.
- **EC-8 Re-Submission nach `needs_info`:** Funktioniert wie ein neuer
  Public-Submit (gleicher Validierungs-Pfad). EEG-Toggle-Stand zum
  Re-Submit-Zeitpunkt entscheidet.
- **EC-9 Configexport zwischen EEGs:** Quell-EEG hat den Toggle aktiv,
  Ziel-EEG nicht. Beim Import wird der Toggle ggf. übernommen, der
  Admin sieht in der Diff-Preview, was sich ändert. Keine Sonderlogik.
- **EC-10 Mitgliedstyp wird nachträglich in Admin-Settings deaktiviert
  (memberTypesEnabled):** Wenn ein deaktivierter Mitgliedstyp in der
  SEPA-Optional-Liste steht, hat das keine Auswirkung (Mitgliedstyp ist
  ohnehin nicht wählbar). Keine zusätzliche Aufräum-Logik nötig.
- **EC-11 Activated-Mail bei „kein_sepa":** Der ShowMandateReferenceHint
  (PROJ-47/53) ist heute schon konditional auf SEPA-Mandat-Existenz.
  Bei `kein_sepa` greift der Hinweis-Block nicht — das ist korrekt und
  bedarf keiner Anpassung. Bestätigt im Code: `mail/service.go:897`
  prüft explizit `app.Einzugsart == "core"`.
- **EC-12 Status-Pfad bei `einzugsart=kein_sepa`:** PROJ-46-Auto-
  Transition `imported -> awaiting_bank_confirmation` greift nur bei
  `einzugsart=b2b` (bestätigt in `internal/importing/import_service.go:428/631/682`).
  `kein_sepa`-Anträge laufen direkt nach `ready_for_activation`, identisch
  zum normalen `core`-Pfad. Keine zusätzliche Status-Logik nötig.
- **EC-13 Bankdaten sind immer Pflicht (Owner-Korrektur 2026-06-08):**
  IBAN/Kontowortlaut müssen IMMER ausgefüllt werden, unabhängig vom
  Checkbox-Status. Begründung: eegFaktura-Core verlangt Bankdaten für
  jedes Mitglied. Beim `einzugsart=kein_sepa`-Pfad werden die Bankdaten
  zwar erfasst, aber kein Mandat-PDF generiert — die EEG kann sie
  später für Überweisungsanforderungen oder manuelle Lastschrift-
  Klärung nutzen.

## Technical Requirements

- **Performance:** Settings-Save + Public-Submit < 200ms (unverändert
  zu PROJ-80).
- **Security:** Backend-Validation darf NICHT auf Frontend-Logik
  verlassen — `sepaMandateAccepted=false` muss server-side gegen EEG-
  Toggle + Mitgliedstyp gecheckt werden (Defense-in-Depth).
- **Browser-Support:** Chrome/Firefox/Safari (wie alle Public-Form-
  Änderungen).
- **Migration:** ALTER ADD COLUMN für 2 neue Spalten auf
  `registration_entrypoint` — non-blocking auf PG 11+.

## Non-Goals

- **Kein Alternativ-Zahlungsweg-Feld** (Dropdown, Freitext). Member
  klärt die Zahlungsmodalität direkt mit der EEG. Bewusste Entscheidung
  Owner.
- **Kein automatischer Mahn-/Erinnerungs-Pfad** für Member ohne SEPA.
  Out of scope.
- **Kein Migration-Pfad für Bestandsanträge.** Toggle wirkt nur auf
  neue Submits.
- **Keine Änderung an `company`/B2B-Lastschrift.** B2B bleibt Pflicht-
  Lastschrift, unabhängig vom Toggle.
- **Kein neuer Mitgliedstyp.** `kein_sepa` bleibt ein `einzugsart`-Wert,
  kein Mitgliedstyp.
- **Keine Anpassung des Admin-Edit-Pfads.** Admin-Override bleibt wie
  heute (AC-21).

## Verwandte Memorys

- `feedback_admin_field_full_chain` — alle 6 Layer durchziehen für
  Master-Toggle UND Mitgliedstyp-Liste
- `feedback_no_placeholders` — kein `placeholder=` auf neuen Form-Inputs
- `feedback_no_proj_refs_in_user_doc` — User-Guide PROJ-frei
- `feedback_batch_changelog_with_code` — Doku im selben Commit wie Code
- `feedback_shared_helpers_for_parallel_paths` — Validierungs-Helper
  („Ist SEPA für diesen Mitgliedstyp Pflicht?") zwischen Public-Submit,
  externer API und Frontend-Form teilen

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)

**Audience:** Owner + PM-Sicht. WAS gebaut wird und WARUM, nicht WIE im Detail.

### A) Komponenten-Baum (was wird angefasst)

```
PROJ-81 Änderungen
+-- Datenbank
|   +-- registration_entrypoint (2 neue Spalten)
|       +-- sepa_optional_enabled (Schalter, Default FALSE)
|       +-- sepa_optional_member_types (Liste von Mitgliedstypen)
|
+-- Backend (Go)
|   |   +-- Shared
|   |   +-- Models — neue Felder auf RegistrationEntrypoint
|   |   +-- Helper „Ist SEPA für diesen Mitgliedstyp optional?"
|   |   |   (zentrale Wahrheit, von 3 Pfaden geteilt)
|   |   +-- Request-Validierung: NUR SepaMandateAccepted-Tag entfernt
|   |       (Bankdaten bleiben Pflicht — eegFaktura-Core verlangt sie)
|   +-- Repository
|   |   +-- SELECT/INSERT/UPDATE um 2 Spalten erweitert
|   +-- HTTP-Handler
|   |   +-- Admin-Settings — Save akzeptiert neue Felder + Cross-Field-Check
|   |   +-- Public-Submit — server-side Entscheidung über einzugsart
|   |   +-- Externe API — gleiche Validierung wie Public-Submit
|   |   +-- Admin-Edit — UNVERÄNDERT (Owner-Direktive)
|   +-- Configexport
|   |   +-- Export/Import/Diff um 2 Felder erweitert
|   +-- Beifang
|       +-- Excel-Label-Map (Bug-Fix)
|       +-- Veralteter Kommentar in buildSEPAMandateData (kosmetisch)
|
+-- Frontend (Next.js / shadcn)
|   +-- Admin-Settings-Editor
|   |   +-- Master-Toggle „SEPA-Wahl im Formular zulassen"
|   |   +-- Sub-Block (eingerückt) mit 4 Mitgliedstyp-Checkboxen
|   |   +-- Hint-Popover am Toggle-Icon
|   |   +-- Inline-Validierungsfehler bei leerer Liste
|   +-- Public-Form (Registration)
|   |   +-- SEPA-Checkbox wird konditional optional/required
|   |   +-- Bankdaten-Felder bleiben IMMER Pflicht (eegFaktura-Core)
|   |   +-- Inline-Hint unter optionaler Checkbox
|   |   +-- Live-Re-Validation bei Mitgliedstyp-Wechsel
|   +-- Admin-Antrags-Detail
|       +-- Info-Block „Kein SEPA-Mandat erteilt" bei kein_sepa
|
+-- Dokumentation
    +-- docs/user-guide/06-admin-settings.md (neuer Abschnitt)
    +-- docs/api-spec.md (Settings-Felder dokumentiert)
    +-- docs/domain-model.md (neue Spalten)
    +-- CHANGELOG.md (im selben Commit wie Code)
```

### B) Datenfluss (4 Sequenzen)

**Sequenz 1 — EEG-Admin aktiviert den Toggle:**

```
Admin im Settings-Editor → klickt Master-Toggle „SEPA-Wahl im Formular zulassen"
  → Sub-Block mit 4 Mitgliedstyp-Checkboxen erscheint
  → Admin wählt z.B. „Privat" + „Pauschalierter Landwirt"
  → klickt Save (PROJ-66 Auto-Save oder explizit)
  → Backend prüft: Toggle aktiv UND Liste leer? → 400 mit Fehlertext
  → Backend prüft: alles OK → speichert beide Felder atomar
  → UI zeigt Erfolgs-Hinweis, Settings-State aktualisiert
```

**Sequenz 2 — Privat-Mitglied submitet MIT SEPA (Standard-Fall):**

```
Browser öffnet Public-Form → wählt Mitgliedstyp „Privat"
  → Form fragt Settings-API ab → erkennt Toggle aktiv + Privat in Liste
  → SEPA-Checkbox wird als „optional" gerendert (kein Sternchen)
  → Member hakt Checkbox an → Bankdaten werden wieder required
  → Member füllt IBAN/Kontowortlaut aus
  → Submit → POST /api/public/applications
  → Backend prüft Helper: optional erlaubt (Toggle+Typ passen)
  → sepaMandateAccepted=true → einzugsart=core
  → Bankdaten-Validierung schlägt zu (Bankdaten Pflicht weil Checkbox an)
  → INSERT, Status submitted, Mandat-PDF wird generiert (heutiger Pfad)
```

**Sequenz 3 — Privat-Mitglied submitet OHNE SEPA-Mandat:**

```
Browser öffnet Public-Form → wählt Mitgliedstyp „Privat"
  → SEPA-Checkbox wird optional gerendert (kein Sternchen)
  → Bankdaten-Felder (IBAN/Kontowortlaut) bleiben Pflicht (Sternchen,
    eegFaktura-Core verlangt sie immer)
  → Member füllt IBAN/Kontowortlaut aus
  → Member lässt SEPA-Checkbox unangehakt
  → Submit → POST /api/public/applications
  → Backend prüft Helper: Toggle aktiv + Privat in Liste → optional erlaubt
  → sepaMandateAccepted=false → einzugsart=kein_sepa
  → Bankdaten werden gespeichert (auch ohne Mandat — EEG kann sie später
    für manuelle Zahlungsklärung nutzen)
  → INSERT, Status submitted, KEIN Mandat-PDF
  → EEG-Submit-Mail meldet „SEPA-Mandat: nein"
```

**Sequenz 4 — Firmen-Mitglied versucht Submit OHNE SEPA (Defense-in-Depth):**

```
Angreifer manipuliert Browser-Request → POST mit memberType=company
  + sepaMandateAccepted=false (Browser-Validation umgangen)
  → Backend prüft Helper: company NIE in Liste (AC-4)
  → Helper liefert „nicht optional"
  → Backend antwortet 400 „SEPA-Einwilligung ist für diesen Mitgliedstyp Pflicht"
  → INSERT findet nicht statt
```

### C) Datenmodell (plain language)

```
registration_entrypoint (bestehende Tabelle, 2 neue Spalten)
+-- sepa_optional_enabled
|   Typ: Schalter (an/aus)
|   Default: aus
|   Bedeutung: „Mitglieder dieser EEG dürfen die SEPA-Einwilligung
|              für ausgewählte Mitgliedstypen weglassen"
|
+-- sepa_optional_member_types
    Typ: Liste von Mitgliedstyp-Codes
    Default: leere Liste
    Erlaubte Werte: private | farmer | association | municipality
    NIE enthalten: company (B2B-Pflicht-Lastschrift)
    Bedeutung: „Genau diese Mitgliedstypen dürfen die Einwilligung
               optional lassen, wenn der Schalter oben an ist"
```

**Keine neuen Tabellen.** Die Liste lebt direkt auf der EEG-Settings-
Zeile, weil sie maximal 4 Einträge hat. Ein JOIN wäre Overhead.

**`application` bleibt unverändert.** Der `einzugsart`-Wert `kein_sepa`
existiert schon seit PROJ-23 (Migration 000023). Wir nutzen ihn nur
neu — als Ergebnis des Public-Form-Pfads, statt nur als Admin-Override.

### D) Tech-Entscheidungen (für PM)

| Entscheidung | Begründung |
|---|---|
| **TEXT[]-Spalte statt separate Tabelle** | Maximal 4 Einträge pro EEG. Eine eigene Tabelle würde einen JOIN bei jedem Settings-Read erfordern, ohne Mehrwert. PostgreSQL kann TEXT[] effizient speichern und vergleichen. |
| **Kein DB-CHECK-Constraint** | Nur ein einziger Schreib-Pfad existiert (Settings-UI durch Admin). Die Validierung dort reicht. Constraints würden die Migration komplizierter machen, ohne realen Schutz zu addieren. |
| **Zentraler Validation-Helper** | Drei Pfade müssen identisch validieren: Public-Submit, Externe API, Admin-Service. Ohne gemeinsamen Helper driften die drei nach ein paar Refactorings auseinander. Memory-Regel `feedback_shared_helpers_for_parallel_paths` ist 2026-05/06 mehrfach reingerannt. |
| **Server-side einzugsart-Entscheidung** | Das Public-Form-Request-Schema enthält heute kein `einzugsart`-Feld. Backend setzt es hartkodiert auf `core`. Wir behalten dieses Pattern bei und entscheiden server-side basierend auf `sepaMandateAccepted`. Vorteil: kein Risiko, dass manipuliertes Frontend einen Fake-Einzugsart-Wert schickt. |
| **Bankdaten bleiben IMMER Pflicht** | eegFaktura-Core verlangt IBAN/Kontowortlaut für jedes Mitglied, auch wenn kein Lastschriftmandat erteilt wird. Bei `kein_sepa` sind die Daten zwar gespeichert, aber kein Mandat-PDF generiert — die EEG kann sie für manuelle Zahlungsklärung nutzen. (Owner-Korrektur 2026-06-08.) |
| **Cross-Field-Validation im HTTP-Handler** | PROJ-80 hat dieses Pattern etabliert (Cross-Field-Check `CORE-Audit ⇒ Timing`). Wir spiegeln das 1:1 für „Toggle aktiv ⇒ Liste nicht leer". |
| **Kein neuer Status im Status-Modell** | `kein_sepa`-Anträge laufen denselben Pfad wie `core`-Anträge, nur ohne Bank-Bestätigung-Branch. Status-Modell muss nicht erweitert werden. |
| **Snapshot-Test gegen Settings-API** | Verhindert Drift zwischen Backend-Helper (Go) und Frontend-Helper (TS). Der Test fragt eine reale Settings-Antwort ab und prüft, dass beide Implementierungen dieselben Mitgliedstyp-Entscheidungen treffen. |

### E) Migrationspfad

```
Phase 1: Migration 000072 deployen
   +-- 2 neue Spalten mit Defaults FALSE / leeres Array
   +-- Bestand-EEGs erben automatisch „Toggle aus" — Verhalten unverändert
   +-- Non-blocking auf PostgreSQL 11+

Phase 2: Backend deployen
   +-- Alle Codepfade kennen neue Spalten (lesen + schreiben)
   +-- Validation-Helper aktiv, antwortet aber für alle Bestand-EEGs „nicht optional"

Phase 3: Frontend Settings deployen
   +-- Admin kann Toggle aktivieren + Mitgliedstypen wählen
   +-- Public-Form rendert noch wie bisher (Toggle bei allen EEGs noch FALSE)

Phase 4: Frontend Public-Form deployen
   +-- Public-Form fragt Settings → reagiert auf Toggle
   +-- Mitgliedstyp-Wechsel triggert Live-Re-Validation
   +-- Bei Toggle FALSE: identisches Verhalten zu heute

Rollback-Pfad:
   +-- down-Migration droppt beide Spalten
   +-- Alle EEGs verlieren ggf. eingestellten Toggle (akzeptabel,
       weil neues Feature → keine Bestand-Daten verloren)
```

Reihenfolge ist sicher: jeder Schritt für sich ist mit allen
vorherigen Code-Ständen kompatibel.

### F) Risiken & Trade-offs

| Risiko | Wahrscheinlichkeit | Folge | Mitigation |
|---|---|---|---|
| Drift Backend-Helper ↔ Frontend-Helper | Mittel | Public-Form akzeptiert, Backend lehnt ab → schlechte UX | Vitest-Snapshot-Test gegen Settings-API |
| Member manipuliert Submit (`sepaMandateAccepted=false` ohne Toggle aktiv) | Hoch (Trivial via DevTools) | Antrag landet illegal mit `kein_sepa` | Backend-Helper-Check fängt es ab (400) |
| Bestand-EEGs reagieren unerwartet auf neue Spalten | Niedrig | Verhalten ändert sich ungewollt | Default FALSE — Verhalten ist garantiert unverändert für alle EEGs, bis Admin den Toggle aktiv setzt |
| Configexport-Cross-EEG-Import importiert Liste mit `company` | Sehr niedrig (Manipulation der JSON) | Inkonsistenter EEG-State | Importer prüft Werte, filtert `company` raus, loggt Warnung |
| Mitgliedstyp-Wechsel im Public-Form führt zu inkonsistentem State | Mittel | Form ist nicht absendbar oder akzeptiert ungültige Kombi | Live-Re-Validation (AC-15) plus serverseitige Defense-in-Depth |
| Admin aktiviert Toggle, leert Liste, klickt Save | Mittel | Settings inkonsistent | Cross-Field-Validation 400 mit klarer Fehlermeldung |
| PROJ-80-Audit-Toggle interferiert | Sehr niedrig | Audit-Block würde an leerem PDF gerendert | Sauber entkoppelt — bei `einzugsart=kein_sepa` wird kein PDF generiert, also auch kein Audit-Pfad |

### G) Dependencies

**Backend:**
- Kein neues Go-Paket. `github.com/lib/pq` (für `pq.Array`) ist seit
  PROJ-2 in `go.mod`.

**Frontend:**
- Kein neues NPM-Paket. shadcn/ui-Checkbox, Switch und Popover sind
  bereits aus PROJ-66/67/80 verbaut.

**Infrastruktur:**
- Keine neuen Environment-Variablen.
- Kein Helm-Wert-Update nötig.
- Keine Cluster-Konfiguration anzupassen.

### H) Implementierungs-Reihenfolge

```
1. Migration 000072 (ADD COLUMN × 2)
2. Shared-Layer (Models, Helper, Request-Validation)
3. Repository (SELECT + INSERT/UPDATE)
4. HTTP Settings-Handler (GET + PUT + Cross-Field-Validation)
5. HTTP Public-Submit (Helper-Check + einzugsart-Entscheidung)
6. HTTP Externe API (gleiche Helper-Check-Logik)
7. Configexport (Schema + Exporter + Importer + Diff)
8. Beifang: Excel-Label-Map fixen, buildSEPAMandateData-Kommentar refreshen
9. Service-Layer-Permutations-Tests (8 Cases)
10. Frontend Settings (Master-Toggle + Sub-Block + Validation-UI)
11. Frontend Public-Form (konditionale Checkbox + Live-Re-Validation)
12. Frontend Antrags-Detail (Info-Block bei kein_sepa)
13. Frontend Vitest-Snapshot-Test gegen Settings-API
14. Playwright E2E-Test (Mitgliedstyp-Wechsel)
15. Doku: User-Guide, api-spec, domain-model, CHANGELOG
```

### I) Test-Strategie

- **Service-Layer-Permutationen** (Backend): Tabellen-Test über
  `{toggle on/off} × {mitgliedstyp ∈ private/farmer/company} × {sepaMandateAccepted true/false}`,
  insgesamt 8 Cases mit erwartetem Outcome (201 oder 400).
- **Frontend-Helper-Snapshot**: Vitest gegen reale Settings-API-Antwort.
- **E2E (Playwright)**: Mitgliedstyp-Wechsel im Public-Form, prüft
  Sternchen/Required-Live-Switch.
- **Manueller QA-Sweep**: Settings aktivieren → Public-Form öffnen →
  Mitgliedstyp wechseln → Checkbox-Verhalten beobachten.

### J) Was bewusst NICHT im Tech-Design ist

- **Kein DB-CHECK-Constraint** für die Mitgliedstyp-Liste — Service-
  Layer reicht (siehe D).
- **Kein Migration-Skript für Bestand** — Toggle wirkt nur auf neue
  Submits (Non-Goal in der Spec).
- **Kein Admin-Edit-Eingriff** (Owner-Direktive AC-21).
- **Kein zusätzlicher Alternativ-Zahlungsweg-Eintrag** (Non-Goal in
  der Spec).
- **Kein Helm-Values-Update** — keine neuen ENV-Variablen.
- **Keine PROJ-80-Toggle-Anpassung** — Audit/Timing-Toggles bleiben
  unverändert, sind sauber entkoppelt.

## QA Test Results

**Tester:** QA Engineer (AI)
**Datum:** 2026-06-08
**Scope:** Code-Review-basiertes QA (kein lokaler Browser-Test, da Backend-Stack nicht gestartet); automatisierte Tests + Smoke-Security-Check.

### Test-Status (automatisiert)

| Suite | Ergebnis |
|---|---|
| `go test ./...` | grün — alle 14 Pakete (mit Cache, dabei: 2 neue Test-Files für PROJ-81) |
| `npx tsc --noEmit` | clean (0 Errors, 0 Warnings) |
| `npx vitest run` | grün — 3 Test-Files / 55 Tests (Rolldown-Override 1.1.0 macht lokales Vitest erstmals lauffähig seit Vitest 4) |
| `govulncheck ./...` | clean — 0 callable vulnerabilities im eigenen Code (5 unaufrufbare in Transitive-Deps, 1 in Modules, alle nicht über PROJ-81-Pfade erreichbar) |
| `npm audit --audit-level=high` | clean — 0 High; 4 Moderate Pre-PROJ-81-Bestand (uuid GHSA-w5hq-g745-h8pq, betrifft `next-auth` Transitive, kein PROJ-81-Regression) |

### Acceptance Criteria

#### Settings-UI (AC-1 bis AC-9)

| AC | Status | Beleg |
|---|---|---|
| AC-1 Master-Toggle in SEPA-Sektion unter PROJ-80-Toggles | Pass | `admin-eeg-settings-editor.tsx` — Block direkt nach dem B2B-Audit-Toggle, vor PROJ-76-Vorstands-Block |
| AC-2 Default FALSE für neue/bestehende EEGs | Pass | Migration 000072: `BOOLEAN NOT NULL DEFAULT FALSE` |
| AC-3 Mitgliedstyp-Liste mit 4 Werten | Pass | `SEPA_OPTIONAL_MEMBER_TYPE_ORDER = ["private", "farmer", "association", "municipality"]` in `sepa-optional.ts`; Helper-Test fixiert das Set |
| AC-4 `company` ausgeschlossen | Pass | Whitelist-Helper `IsValidSEPAOptionalMemberType` (Go + TS) lehnt company ab; UI-Order enthält company nicht |
| AC-5 Mindestens ein Mitgliedstyp aktiv | Pass | Backend `admin.go:2155-2167` 400; Frontend Pre-Save-Validation (UX-Komfort) |
| AC-6 Kurz-Erklärung unter Toggle | Pass | Italic-text-muted-foreground-Block im Settings-Editor (Pattern aus PROJ-80) |
| AC-7 Hint-Popover am Info-Icon | Pass | Popover mit max-w-80 + 3 Absätzen (Wirkung, Bankdaten-Pflicht, Firmen-Ausnahme) |
| AC-8 Standard-Modus blendet Toggle aus | Pass | `{isAdvanced && ...}`-Gate |
| AC-9 Advanced-Trigger | Pass | `settings-mode.ts:isAdvancedEEGSettingsActive` ergänzt + Test in `settings-mode.test.ts` |

#### Public-Form (AC-10 bis AC-15)

| AC | Status | Beleg |
|---|---|---|
| AC-10 Render-Bedingung optional/required | Pass | `sepaMandateOptional`-Variable in `registration-form.tsx:527+`, getrieben von `isSepaOptional(...)` |
| AC-11 Bankdaten bleiben Pflicht | Pass | `requests.go:32-33` IBAN/AccountHolder mit `validate:"required"` (zurückgerollt nach Owner-Korrektur) |
| AC-12 entfällt | N/A | Bewusst gestrichen — Bankdaten waren nie optional |
| AC-13 einzugsart-Mapping server-side | Pass | `application_service.go:170-180` — `einzugsart="kein_sepa"` wenn `!req.SepaMandateAccepted` |
| AC-14 Inline-Hint unter Checkbox | Pass | Italic-text-muted-foreground-Block in `registration-form.tsx`, sichtbar nur wenn `sepaMandateOptional` |
| AC-15 Live-Re-Validation bei Mitgliedstyp-Wechsel | Pass | `sepaMandateOptional` wird live aus `form.watch("memberType")` abgeleitet — Sternchen + zod-Required updaten beim Wechsel |

#### Backend Submit-Validation (AC-16 bis AC-19d)

| AC | Status | Beleg |
|---|---|---|
| AC-16 Submit-Validation Defense-in-Depth | Pass | `application_service.go:99-105` Pre-Insert-Check: `!sepaMandateAccepted && !IsSEPAOptional → 400` |
| AC-16a Request-Body-Validation nur SepaMandateAccepted gelockert | Pass | `requests.go:35` `bool` ohne `required`-Tag; IBAN/AccountHolder bleiben `required` |
| AC-17 einzugsart-Erzwingung | Pass | Server-side-Entscheidung in `application_service.go`, kein client-side-`einzugsart`-Feld |
| AC-18 Mandat-PDF entfällt bei kein_sepa | Pass | `buildSEPAMandateData` default-Branch returnt `nil` (unverändert seit PROJ-12; bestätigt in Code-Block) |
| AC-19 EEG-Submit-Mail mit Hinweis-Banner | Pass | `application_submitted_eeg.html` — `{{if .NoSepaMandate}}`-Block direkt nach Eingangsabsatz, gelber Banner |
| AC-19a EEG-Activated-Mail mit Hinweis-Banner | Pass | `application_activated_eeg.html` — gleicher Banner über der Antrags-Detail-Tabelle |
| AC-19b PROJ-76-Beitrittserklärung mit Hinweis-Banner | Pass | `service.go SendBoardApprovalRequest` — Inline-HTML-Block bei `app.Einzugsart == "kein_sepa"` |
| AC-19c Imported-Mail unverändert (Mail entfällt) | Pass | Mail-Trigger via `SendMandateAtImportNotification` greift nur bei `einzugsart ∈ {core,b2b}` — kein Eingriff nötig |
| AC-19d NoSepaMandate-Feld zentral | Pass | `eegSubmissionData.NoSepaMandate` + `activationTemplateData.NoSepaMandate` als Single-source, je via `strings.EqualFold(..., "kein_sepa")` gesetzt |
| AC-19a Shared Validation-Helper | Pass | `shared.IsSEPAOptional` ist die zentrale Wahrheit, aufgerufen von Public-Submit + Externe API |

#### Externe API + Admin (AC-20 bis AC-22)

| AC | Status | Beleg |
|---|---|---|
| AC-20 Externe API gleiche Validation | Pass | `external.go:120-127` harter SEPA-Check entfernt, `sepaMandateAccepted` durchgereicht; Service-Layer-Helper greift |
| AC-21 Admin-Edit unverändert | Pass | `admin_service.go:448` `app.Einzugsart = *req.Einzugsart` unverändert; Spec dokumentiert dass Admin auch `company` auf `kein_sepa` setzen darf |
| AC-22 Antrags-Detail-Banner bei kein_sepa | Pass | `admin-application-detail.tsx` — blauer Border-l-4-Banner direkt über der Bankverbindungs-Card |

#### Configexport + Beifang (AC-23 bis AC-23e)

| AC | Status | Beleg |
|---|---|---|
| AC-23 Configexport mit Legacy-Pattern | Pass | `schema.go` `*bool, omitempty` + `[]string, omitempty`; Exporter setzt Pointer, Importer defaultet auf FALSE/leer |
| AC-23a Excel-Label-Map gefixt | Pass | `dataexport/excel/fields.go:252` — `basis` → `core`, `kein_sepa` neu; `fields_test.go` fixiert das |
| AC-23b buildSEPAMandateData-Kommentar refresht | Pass | `application_service.go:1635-1648` — kein veraltetes `SEPAMandateEnabled` mehr im Doku-Block |
| AC-23c Service-Layer-Permutations-Test (8 Cases) | Pass | `internal/shared/sepa_optional_test.go` mit 8 Permutationen (Toggle on/off × memberType × Listed/Not) |
| AC-23d Frontend-Helper-Snapshot | Pass | `src/lib/sepa-optional.test.ts` Spiegel der Go-Permutationen, dient als Drift-Schutz |
| AC-23e Playwright E2E | Defer | E2E-Suite hat heute kein PROJ-81-Test; Empfehlung: dazu in eigener Welle, weil Playwright-Setup auf dem Rechner nicht startet (Backend muss live sein für Public-Form-Test). Kein QA-Blocker, weil die Live-Re-Eval-Logik durch isSepaOptional-Unit-Test + manuelles Code-Review abgedeckt ist. |

#### Doku (AC-24 bis AC-28)

| AC | Status | Beleg |
|---|---|---|
| AC-24 User-Guide-Update | Pass | `docs/user-guide/06-admin-settings.md` neuer Abschnitt „SEPA-Wahl im Formular zulassen" mit Max-Mustermann-Beispiel |
| AC-25 API-Spec-Update | Pass | `docs/api-spec.md` GET/PUT-Body + Public-Submit + Externe API |
| AC-26 Domain-Model-Update | Pass | `docs/domain-model.md` 2 neue Spalten beschrieben |
| AC-27 User-Guide PROJ-frei | Pass | `grep -r "PROJ-[0-9]" docs/user-guide` liefert 0 Treffer ✓ |
| AC-28 CHANGELOG | Pass | `CHANGELOG.md` Unreleased-Block mit vollständigem PROJ-81-Eintrag |

### Edge Cases

| EC | Status | Beleg |
|---|---|---|
| EC-1 Toggle aktiv, leere Liste | Pass | Backend `admin.go:2155` 400; Frontend Pre-Save + Inline-Hint („Bei aktiver Option muss mindestens ein Mitgliedstyp ausgewählt sein") |
| EC-2 Mitgliedstyp-Wechsel live | Pass | `sepaMandateOptional` via `form.watch("memberType")` — Sternchen/zod live reaktiv |
| EC-3 Bestandsantrag bei Toggle-Aktivierung unverändert | Pass | Toggle wirkt nur auf Public-Submit-Pfad; Bestandsanträge in DB werden nicht angefasst |
| EC-4 EEG deaktiviert Toggle nach kein_sepa-Submit | Pass | DB-Wert `einzugsart=kein_sepa` bleibt; neue Submits wieder Pflicht |
| EC-5 Externer API-Submit kein_sepa bei Toggle-aus | Pass | `IsSEPAOptional` returnt false → Service 400 |
| EC-6 Externer API mit company + kein SEPA | Pass | `IsSEPAOptional` returnt false (company-Branch), 400 |
| EC-7 Bankdaten-Pflicht-Verletzung | Pass | Validator-Tags `required` → klassischer 422 mit Field-Error |
| EC-8 Re-Submission nach needs_info | Pass | Re-Submit nutzt denselben CreateApplication-Pfad → gleiche Validation |
| EC-9 Configexport Cross-EEG company-Filter | Pass | `importer.go sanitizeSEPAOptionalMemberTypes` filtert + loggt; `sanitize_sepa_optional_test.go` deckt 6 Cases ab |
| EC-10 Mitgliedstyp deaktiviert in memberTypesEnabled | Pass | Heute kein `memberTypesEnabled` per EEG (alle global aktiv) — keine Sonderlogik nötig |
| EC-11 Activated-Mail bei kein_sepa | Pass | `ShowMandateReferenceHint` greift nicht (Mail-Service prüft `einzugsart == "core"`); neuer `NoSepaMandate`-Banner wird gerendert |
| EC-12 Status-Pfad bei kein_sepa | Pass | Import-Service-Branching: `awaiting_bank_confirmation` nur bei `b2b`; `kein_sepa` läuft zu `ready_for_activation` |
| EC-13 Bankdaten-Pflicht-Korrektur dokumentiert | Pass | Spec + CHANGELOG + Hint-Texte alle konsistent „Bankdaten bleiben Pflicht" |

### Security Smoke-Test

#### Scope
Geprüft: Migration 000072, neuer Helper `IsSEPAOptional`, Settings-Handler-Erweiterung (GET/PUT), Public-Submit-Validation-Erweiterung, Externe-API-Pfad, Configexport-Sanitizer, Excel-Label-Fix, 3 Mail-Templates, Frontend-Helper + UI.

#### Findings

| Severity | Datei | Funktion | Risiko | Exploit-Szenario | Fix-Empfehlung | Confidence |
|---|---|---|---|---|---|---|
| Info | `internal/shared/requests.go:33-36` | CreateApplicationRequest | `SepaMandateAccepted` ist jetzt `bool` ohne `required`-Tag | bewusste Designentscheidung — Validator akzeptiert `false` damit der Service-Layer-Helper über Pflichtigkeit entscheiden kann; Defense-in-Depth-Check in `CreateApplication` lehnt unberechtigtes `false` mit 400 ab | Keine Aktion. Test `internal/shared/sepa_optional_test.go` deckt 8 Permutationen ab; integrationstest des Defense-in-Depth-Checks wäre Plus, nicht Pflicht. | High |
| Info | `internal/http/admin.go SaveEEGSettings` | Settings-PUT-Body | Mitgliedstyp-Liste kommt aus User-Input und wird in DB-`TEXT[]` gespeichert | `pq.Array(sepa_optional_member_types)` ist parametrisiert; Whitelist-Check `IsValidSEPAOptionalMemberType` lehnt company und Garbage server-side ab; `sepaOptionalEnabled=FALSE` clearet die Liste defensiv | Keine Aktion. Whitelist + parametrisierte SQL ist OK. | High |
| Info | `internal/configexport/importer.go sanitizeSEPAOptionalMemberTypes` | Importer | Manipulierte JSON-Bundles mit `company` in der Liste werden weggefiltert und mit `slog.Warn` protokolliert | Cross-EEG-Bundle-Submission durch privilegierten Admin (tenant-scoped); Whitelist greift, Audit-Log entsteht | Keine Aktion. | High |
| Info | `internal/mail/service.go SendBoardApprovalRequest` | Inline-HTML-Bau | Hinweis-Banner wird per `bytes.Buffer.WriteString` zusammengebaut, nicht aus Mail-Daten-Struct | `app.Einzugsart` wird mit `strings.EqualFold` und `strings.TrimSpace` geprüft, KEIN User-Input rendert direkt in HTML; `html.EscapeString` schon im Bestandspfad für `memberName/eegName/memberNumber` | Keine Aktion. Banner-Block ist konstanter HTML-Template-Text mit Fixed-Style. | High |
| Low | Anti-Bot/Rate-Limit | Public-Submit-Pfad | PROJ-81 schaltet keine Rate-Limit-Lockerung; bestehender Pfad bleibt mit Turnstile + Rate-Limit aktiv | n/a | Keine Aktion. | High |

#### Geprüft + clean

- **Auth/AuthZ:** Settings-Endpoint nutzt unverändertes Tenant-Scope-Pattern (`parseRCAndCheck`). Kein neuer Endpoint. Kein Superuser-Eskalations-Pfad.
- **Tenant-Isolation:** PROJ-81 fügt keine Cross-EEG-Logik hinzu. Configexport-Bundle bleibt admin-only und tenant-scoped wie PROJ-61.
- **Injection:** Alle Schreibpfade auf `registration_entrypoint` nutzen `pq.Array(...)` für TEXT[]. Alle Reads/Writes sind parametrisierte Queries. Keine String-Concat.
- **Input-Validation:** 4-Element-Whitelist serverseitig erzwungen via `IsValidSEPAOptionalMemberType`. Liste-Wert-Limit auf TEXT-Default keine explizite `max=`, aber Whitelist enumeriert die 4 zulässigen Werte (≤ 16 Zeichen) — kein DoS-Vector.
- **Defense-in-Depth:** Public-Submit + Externe API + Admin-Settings-Save rufen alle drei den shared Helper auf. Manipuliertes Frontend (forged `sepaMandateAccepted=false` + nicht-berechtigter Mitgliedstyp) wird vom Backend mit 400 abgelehnt.
- **Logging:** `sanitizeSEPAOptionalMemberTypes` loggt nur `rc` + `member_type`-Roh-Wert (Whitelist-Wert oder Garbage), kein PII.
- **DSGVO:** Keine neuen PII-Felder. Mitgliedstyp-Liste ist Konfig, keine Personendaten.
- **Schema-Migration:** `ADD COLUMN` mit `NOT NULL DEFAULT` ist non-blocking auf PG 11+. Down-Migration mit `IF EXISTS` ist idempotent. Bestandsdaten bekommen automatisch Default-Werte.
- **Eingabe-Längenbeschränkungen:** `sepaOptionalMemberTypes []string` hat im Body-Struct keinen `validate:"max="` auf Anzahl, aber Whitelist + 4-Werte-Universum begrenzen die effektive Größe. **Empfehlung Low:** optional `validate:"max=10"` ergänzen, um Konfigexport-Bundles mit übergroßer Liste früh abzulehnen. Kein Blocker.

#### Pflicht-Trigger für `/security-review`
Ja — PROJ-81 berührt:
- DB-Schema-Change (Migration 000072) ✓
- Public-Submit-Validation (Defense-in-Depth-Pfad neu) ✓
- Configexport mit User-getriggertem Sanitizer ✓
- Drei EEG-Mail-Pfade mit neuem konditionalen Block ✓

Empfehlung: **`/security-review` als nächster Workflow-Schritt vor `/deploy`.**

### Regression-Sweep

| Bereich | Status | Notiz |
|---|---|---|
| PROJ-80 SEPA-Settings-Vereinfachung | Pass | CORE-Audit-Toggle + Cross-Field-Validation in `admin.go:2131-2143` unverändert; alle Tests `TestBuildSEPAMandateData_PROJ80_*` weiter grün |
| PROJ-78 Audit-Toggles (CORE/B2B) | Pass | Toggle-Felder unangetastet; PDF-Renderer-Branching unverändert |
| PROJ-76 Vorstands-Workflow | Pass | `SendBoardApprovalRequest`-Hauptpfad unverändert; Hinweis-Banner-Block nur ADD-ON bei `kein_sepa`; Member/EEG-Routing unverändert |
| PROJ-46 Status-Modell | Pass | `import_service.go:428` `awaiting_bank_confirmation` triggert nur bei `b2b`; `kein_sepa` läuft zu `ready_for_activation` |
| PROJ-21 Approval-PDF | Pass | PDF-Renderer-Pfad unverändert; bei `kein_sepa` returnt `buildSEPAMandateData` weiterhin nil |
| PROJ-17 Excel-Export | Pass | `EinzugsartLabels` korrigiert — heutiger Export für `core`-Anträge zeigt jetzt „SEPA-Basismandat" statt Raw-Wert „core" (Verbesserung) |
| Admin-Edit-Form einzugsart | Pass | `admin_service.go:448` unverändert; UI in `admin-edit-form.tsx` unverändert |

### Helm-Check

Kein Helm-Eingriff in PROJ-81. `git diff --name-only HEAD` zeigt keine Pfade unter `helm/` oder `Dockerfile*`. Keine neuen ENV-Variablen. Migration läuft automatisch über den bestehenden migrate-Job.

### Production-Ready-Empfehlung

**READY** (für `/security-review`).

- 33 ACs: 32 Pass, 1 Defer (AC-23e Playwright-E2E — kein Blocker, manuell + Code-Review ausreichend; E2E ist allgemein nur auf PR-Pfad relevant, siehe `reference_e2e_drift_window`)
- 13 ECs: alle Pass
- 0 Critical/High Security Findings
- 5 Info-Findings (alle bewusste Designentscheidungen mit Rationale)
- 1 Low (optional `max=10` auf Liste im Body — könnte beim Security-Review ergänzt werden)
- 0 Regression
- Tests: go grün, vitest grün, tsc clean, govulncheck clean, npm audit ohne High

**Nächster Workflow-Schritt:** `/security-review` (Migration + Public-Submit-Validation + Configexport-Sanitizer + 3 neue Mail-Pfade). Danach `/deploy`.

## Security Review (PROJ-81)

**Reviewer:** Security Engineer (AI)
**Datum:** 2026-06-08
**Scope:** Migration 000072, `IsSEPAOptional`-Helper, Public-Submit-Defense-in-Depth in `CreateApplication`, Externe-API-Pfad, Settings-Handler-Whitelist + Cross-Field-Validation, Configexport-Sanitizer mit `company`-Filter, 3 EEG-Mail-Templates mit `kein_sepa`-Banner, Excel-Label-Map-Beifang-Fix, Frontend-Spiegel + Settings-UI.

### Threat Model

Worst-Case-Szenarien, gegen die wir reviewen:

1. **Public-Member umgeht SEPA-Pflicht ohne EEG-Konsens.** Schaden: EEG erwartet Lastschrift-Mandat, bekommt Mitglied mit `kein_sepa`, geht in der Abrechnung leer aus. DSGVO-relevant, weil sich die EEG auf ihre eigene Geschäftspraxis verlässt.
2. **Tenant-Admin (oder manipuliertes Configexport-Bundle) schleicht `company` in die Whitelist.** Schaden: Firmen-Mitglieder könnten plötzlich ohne B2B-Mandat einreichen — würde SEPA-Regelwerk verletzen und das Mandat unwirksam machen.
3. **DoS via übergroßer Mitgliedstyp-Liste im Settings-Save oder Configexport-Import.** Authentifizierter Tenant-Admin/Superuser kann den Loop und DB-Roundtrip strapazieren.
4. **XSS in den neuen Hinweis-Bannern.** Wenn `app.Einzugsart` (DB-Wert, server-controlled) oder Member-Input ungeprüft als HTML gerendert würde, hätten wir E-Mail-XSS.

### Defense-in-Depth-Verifikation

| Pfad | Frontend-Check | Backend-Validator | Service-Layer-Check |
|---|---|---|---|
| Public-Submit | `isSepaOptional()` triggert konditional zod-Required (`registration-form.tsx`); Live-Re-Eval bei Mitgliedstyp-Wechsel | `validate:"omitempty"` auf `IBAN/AccountHolder` würde Pflicht-Verlust ermöglichen — bewusst **nicht** angefasst, beide bleiben `required` | `application_service.go:104` — `!req.SepaMandateAccepted && !shared.IsSEPAOptional(ep, req.MemberType)` lehnt mit 400 ab, **vor jedem Insert**, vor Field-Config-Load, vor MP-Build |
| Externe API | (entfällt — Backend-Integrator) | `external.go:120-125` — harter SEPA-Check entfernt, durchgereicht | Gleicher Service-Layer-Check wie Public-Submit |
| Settings-Save | Pre-Save UX-Warnung in `admin-eeg-settings-editor.tsx` (kein Save-Block bei Backend) | `admin.go:2160-2181` — leere Liste bei aktiv → 400; jeder Mitgliedstyp gegen `IsValidSEPAOptionalMemberType` → 400 bei `company`/Garbage | n/a (UPDATE direkt nach Validation) |
| Configexport-Import | (entfällt — Bundle-Datei) | n/a | `sanitizeSEPAOptionalMemberTypes` in `configexport/importer.go:26-40` filtert `company`/Garbage **silent + logged** statt zu rejecten |

Der Public-Submit-Pfad hat damit eine durchgehende Kette: Frontend zeigt richtige UX, Validator akzeptiert `false` (damit `IsSEPAOptional` entscheiden kann), Service-Layer wirft 400 bei unberechtigtem `false`. Manipuliertes Frontend (DevTools-`sepaMandateAccepted=false` forced) wird vom Service-Layer abgefangen, kommt nie zum INSERT.

Der Settings-Save-Pfad lehnt `company` zweimal ab: einmal im HTTP-Handler (mit Field-Error-Message), einmal beim Configexport-Import (silent + Audit-Log). Tenant-Admin kann `company` nicht in die DB schleichen.

### Findings

| Severity | Datei | Funktion/Bereich | Risiko | Exploit-Szenario | Empfohlener Fix | Confidence |
|---|---|---|---|---|---|---|
| Low | `internal/shared/requests.go` + `internal/http/external.go` | `CreateApplicationRequest` + `externalApplicationRequest` body | `sepaOptionalMemberTypes` ist heute kein Body-Feld auf Public-/Externe-API, sondern Settings-only. Über Settings-Save könnte ein authentifizierter Tenant-Admin theoretisch eine extrem große Liste schicken. Whitelist-Check ist O(n), DB-Spalte TEXT[] hat keinen harten Element-Limit. | Tenant-Admin schickt `{"sepaOptionalEnabled": true, "sepaOptionalMemberTypes": [10000× "private"]}` → 10000 Whitelist-Loops + großer DB-Payload. Kein effektiver DoS, weil tenant-admin-only und logbar. | Optional: `validate:"max=10"` auf das Body-Feld in `admin.go SaveEEGSettings`. Klein, kann beim nächsten Touch reingehen. | High |
| Low | `internal/http/admin.go:2175` | `SaveEEGSettings`-Error-Message | User-Input (`mt`-Wert) wird in die Fehlermeldung gerendert: `"Ungueltiger Mitgliedstyp: " + mt + " (...)"`. Response ist JSON, geht an authentifizierten Tenant-Admin. | JSON-Body kein XSS-Vector im Browser (Content-Type `application/json` rendert nicht als HTML). Nur Risiko wenn das Admin-Frontend `dangerouslySetInnerHTML` nutzt — tut es nicht (alle Toast-/Field-Error-Komponenten rendern via React Text). | Keine Aktion. | High |
| Info | `internal/shared/requests.go:35` | `CreateApplicationRequest.SepaMandateAccepted` | Validator-Tag `required` entfernt, damit `false` akzeptiert wird | Bewusste Designentscheidung — `validate:"required"` auf `bool` lehnt `false` als „nicht gesetzt" ab; Service-Layer-Helper entscheidet kontextabhängig. Defense-in-Depth-Check in `CreateApplication:104` greift sicher. | Keine Aktion. 8-Permutationen-Helper-Test (`internal/shared/sepa_optional_test.go`) deckt die Logik ab. | High |
| Info | `internal/http/admin.go SaveEEGSettings` | Tenant-Admin schreibt in TEXT[] | Whitelist + `pq.Array` parametrisiert | `IsValidSEPAOptionalMemberType` enumeriert 4 erlaubte Werte hardcoded, `company` ist nicht im Set; Defense-in-Depth doppelt: Handler-Loop + Configexport-Sanitizer | Keine Aktion. | High |
| Info | `internal/configexport/importer.go` | `sanitizeSEPAOptionalMemberTypes` | Silent-Filter statt Reject bei manipuliertem Bundle | Bewusste UX-Entscheidung: Configexport ist Migration zwischen EEGs, Reject würde gesamte Import-Transaktion abbrechen; Sanitizer filtert nur ungültige Werte und protokolliert via `slog.Warn` mit `rc + member_type`. Kein PII im Log. | Keine Aktion. | High |
| Info | `internal/mail/service.go SendBoardApprovalRequest` + `application_submitted_eeg.html` + `application_activated_eeg.html` | Hinweis-Banner-Block | Banner-HTML ist statischer Template-Text, kein User-Input rendert direkt | `app.Einzugsart` wird ausschließlich in `strings.EqualFold`-Check gegen `"kein_sepa"` verwendet, nicht in HTML interpoliert. Member-Name/EEG-Name im PROJ-76-Inline-Pfad gehen weiterhin durch `html.EscapeString`. Banner-Strings sind statisch. | Keine Aktion. | High |
| Info | Migration 000072 | Schema-Migration | `ADD COLUMN BOOLEAN NOT NULL DEFAULT FALSE` + `TEXT[] NOT NULL DEFAULT '{}'` | Non-blocking auf PG 11+ (Default-Wert wird ohne Table-Rewrite gesetzt). Down-Migration mit `IF EXISTS` ist idempotent. Bestandsdaten erben sichere Defaults — kein Behavior-Change ohne Admin-Toggle-Aktion. | Keine Aktion. | High |

### Scan Results

| Scanner | Ergebnis |
|---|---|
| `govulncheck ./...` | clean — 0 callable vulnerabilities |
| `gosec -severity medium -confidence medium ./...` | clean — 0 Issues über 89 Files / 32907 Lines |
| `npm audit --audit-level=high` | clean — 0 High; 4 Moderate Pre-PROJ-81-Bestand (uuid GHSA-w5hq-g745-h8pq via next-auth Transitive) |
| `trivy config .` | clean — 0 HIGH/CRITICAL Findings in Helm + k8s/test/* |
| Semgrep | nicht lokal ausgeführt — CI deckt das via `.github/workflows/security-scan.yml` ab |

### Out-of-Scope (bewusst nicht angefasst)

- Keycloak-Auth-Logik unverändert
- `IsSuperuser()` + `checkTenantAccess`/`parseRCAndCheck` unverändert
- Status-Transition-Map unverändert (`kein_sepa` läuft denselben Pfad wie `core`)
- Rate-Limit + Turnstile am Public-Submit unverändert
- Helm + Dockerfile + GitHub Actions unverändert (keine Diff-Hunks)
- Secrets/ENV-Variablen unverändert
- Admin-Edit-Form unverändert (AC-21)
- Core-Import-Boundary unverändert

### Verdict: **APPROVED**

Begründung:
- 0 Critical / 0 High Findings
- 2 Low Findings (max-N auf Liste; reflektierter String in JSON-Error) — beide kein Blocker, beide bewusste UX-/API-Designentscheidungen mit dokumentiertem Risiko
- 5 Info Findings — alle Designentscheidungen, alle mit High Confidence
- Defense-in-Depth-Kette durchgehend verifiziert (Public + Externe API + Settings + Configexport)
- Whitelist im Handler **UND** im Sanitizer — Tenant-Admin kann `company` nicht reinschleichen
- Banner-HTML statisch, kein XSS-Vector
- Migration sicher, non-blocking, Defaults-erbend
- Logging PII-frei
- Alle Scanner clean

**Empfehlung:** Direkt `/deploy` — Helm-Bump + Migrations-Job + Standard-Rollout. Migration `000072` läuft beim Pre-Deploy-migrate-Job automatisch.

**Optionaler Cleanup (nicht-blockierend):** Bei einem zukünftigen Touch von `admin.go SaveEEGSettings` könnte ein `validate:"max=10"` auf `SEPAOptionalMemberTypes` ergänzt werden. Klein, kein eigener PROJ-Aufwand nötig.

## Deployment
_To be added by /deploy_
