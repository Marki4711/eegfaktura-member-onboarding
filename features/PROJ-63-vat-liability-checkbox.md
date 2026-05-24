# PROJ-63 — USt-Pflicht-Checkbox bei Unternehmen + Verein

**Status:** In Progress
**Created:** 2026-05-24
**Depends on:** PROJ-62 (Mitgliedstyp-Merge)

## Hintergrund

PROJ-62 hat den Mitgliedstyp `sole_proprietor` (Kleinunternehmer) in `company`
verschmolzen und „leere UID-Nummer" als implizites Signal für die
Kleinunternehmerregelung (§ 6 Abs 1 Z 27 UStG) genutzt. Owner-Feedback
nach dem Deploy: **es gibt Firmen, die eine UID haben und trotzdem
Kleinunternehmer sind** (z.B. wegen innergemeinschaftlicher Erwerbe). Die
implizite Ableitung scheitert in diesem Fall.

Fachliche Klärung: für die Gutschrift einer EEG an einen Kleinunternehmer
ist die UID umsatzsteuerlich **nicht erforderlich** (vgl. § 11 UStG). Wir
brauchen sie also gar nicht zu kennen — solange das System das Mitglied
nicht versehentlich als regelbesteuert behandelt, weil die UID ausgefüllt
wurde.

## Lösung

Eine **Checkbox** im öffentlichen Registrierungs- und im Admin-Edit-
Formular:

> ☐ Das Unternehmen ist umsatzsteuerpflichtig (Regelbesteuerung)
>   (bei Verein: „Der Verein ist umsatzsteuerpflichtig …")

- **Default:** unchecked = Kleinunternehmer.
- **Unchecked:** UID-Eingabefeld wird **nicht angezeigt** — der Bewerber
  kann gar nicht aus Reflex eine UID eintragen, auch wenn er eine hätte.
- **Checked:** UID-Eingabefeld erscheint und ist **Pflicht** (clientseitig).
- **Sichtbar bei:** `memberType ∈ { company, association }`.
- **Nicht sichtbar bei:** `municipality` — dort wird die USt-Differenzierung
  über die Zählpunkte (Hoheitsbereich vs. BgA, PROJ-59) abgewickelt; ein
  pauschaler Toggle am Application-Level wäre irreführend.
- **Beim Umschalten auf unchecked** wird ein zuvor eingetragener UID-Wert
  geclearted, damit kein Reststand mitgesendet wird.
- **Beim Wechsel** des Mitgliedstyps auf einen Nicht-Org-Typ oder auf
  Gemeinde wird der Toggle auf false zurückgesetzt.

## Bewusst NICHT umgesetzt

- **Keine DB-Spalte** `vat_liable`. Der Wahrheitswert „Kleinunternehmer"
  bleibt implizit aus `uid_number IS NULL` ableitbar (wie heute), was für
  die Abrechnungsanforderungen der EEG ausreicht.
- **Keine Backend-Validierung.** API-Direkt-Aufrufer (z.B. die externe
  Registrations-API) können weiterhin UID + nichts setzen — der UI-Gate
  ist reine Public-Form-Hygiene.
- **Kein per-EEG-Default.** Owner-Entscheidung: ein einheitlicher
  globaler Default reicht (siehe Edge-Case unten).

## User Stories

### US-1: Kleinunternehmer ohne UID

> Als Kleinunternehmer-GmbH ohne UID möchte ich das Registrierungs-
> formular einfach durchlaufen können, ohne mit einem irrelevanten
> UID-Feld konfrontiert zu werden.

### US-2: Kleinunternehmer mit UID

> Als Kleinunternehmer-GmbH, die zufällig eine UID besitzt (für
> innergemeinschaftliche Erwerbe), möchte ich nicht aus Versehen die
> UID eintragen und damit fälschlich als regelbesteuert eingestuft
> werden.

### US-3: Regelbesteuertes Unternehmen

> Als regelbesteuerte GmbH soll mir klar sein, dass ich die UID-Nummer
> hier eintragen muss — die Checkbox macht den Zusammenhang explizit.

### US-4: Admin bearbeitet einen Bestandsantrag

> Als Admin will ich bei bestehenden Anträgen sehen, ob das Mitglied
> als regelbesteuert oder Kleinunternehmer geführt wird, und das
> ändern können — auch wenn das Mitglied vor PROJ-63 eingereicht hat
> (Backwards-Compat: Checkbox-State wird aus `uid_number IS NOT NULL`
> abgeleitet beim Laden).

## Acceptance Criteria

### AC-1: Default-State (Public Form)
Bei Auswahl von `Unternehmen` oder `Verein` ist die USt-Pflicht-Checkbox
sichtbar und **unchecked**. Das UID-Eingabefeld ist **nicht** sichtbar.

### AC-2: Toggle aktiviert UID-Feld
Sobald die Checkbox angekreuzt wird, erscheint das UID-Feld mit dem
Label „UID-Nummer *". Bei Submit mit leerer UID erscheint die Fehler-
meldung „UID-Nummer ist erforderlich".

### AC-3: Toggle deaktiviert UID-Feld + cleart Eingabe
Wenn die Checkbox abgewählt wird, verschwindet das UID-Feld. Ein zuvor
eingetragener UID-Wert ist beim Wieder-Aktivieren leer (cleared).

### AC-4: Gemeinde unverändert
Bei Mitgliedstyp `Gemeinde` ist die Checkbox **nicht** sichtbar. Das
UID-Feld bleibt wie bisher **optional sichtbar**.

### AC-5: Mitgliedstyp-Wechsel cleart Toggle
Wenn von `Unternehmen`/`Verein` auf einen anderen Typ gewechselt wird,
springt die Checkbox zurück auf unchecked. Beim Zurück-Wechseln startet
sie wieder bei unchecked (kein versteckter Restzustand).

### AC-6: Admin-Edit-Form spiegelt Public Form
Im Admin-Edit-Formular existiert dieselbe Checkbox. Beim Laden eines
Bestandsantrags wird der Initial-State aus dem Vorhandensein einer
UID-Nummer abgeleitet (UID gesetzt ⇒ Toggle an).

### AC-7: API bleibt unverändert
Die Checkbox wird **nicht** an die Backend-API übertragen. Das Backend-
Schema (`application.uid_number`) und die Validierung bleiben unverändert.
Externe API-Aufrufer sind nicht betroffen.

## Edge Cases

- **API-Bypass:** Ein direkter `curl` gegen `/api/public/applications`
  mit `memberType=company` + `uidNumber="ATU..."` ohne Checkbox-Status
  funktioniert weiterhin und wird als regelbesteuert behandelt. Akzeptiert,
  weil die externe API ein anderer Use Case ist (typischerweise B2B-
  Integrationen mit vollständigen Stammdaten).
- **Bestandsanträge** vor PROJ-63: bleiben unverändert in der DB. Beim
  erneuten Admin-Edit wird der Checkbox-Default aus UID-Vorhandensein
  abgeleitet — das deckt sich mit dem PROJ-62-Behavior.
- **Kleinunternehmer wird regelbesteuert** (Überschreitung der €55.000-
  Grenze): Admin setzt die Checkbox und ergänzt die UID nach. Keine
  Migration nötig.

## Tech Design

- **Frontend-only Refactor.**
- Zod-Schema in `src/components/registration-form.tsx` bekommt ein
  optionales `vatLiable: z.boolean().optional()`. Default `false` in
  `defaultValues`.
- Render-Gating der UID-Sektion auf `(memberType === "company" ||
  memberType === "association") && form.watch("vatLiable")`. Bei
  `memberType === "municipality"` bleibt die UID-Sektion unverändert
  sichtbar.
- Required-Validation via `superRefine`: wenn `vatLiable===true` UND
  `memberType ∈ {company, association}` UND `!uidNumber?.trim()` →
  Fehler „UID-Nummer ist erforderlich".
- Checkbox-`onCheckedChange` cleart bei `false` das `uidNumber`-Feld
  und löscht etwaige Fehlermeldungen.
- `onMemberTypeChange` setzt `vatLiable` auf `false` bei Wechsel auf
  Nicht-Org-Typen sowie auf `municipality`.
- `src/components/admin-edit-form.tsx`: spiegelt das Verhalten mit
  einem lokalen `useState<boolean>(...)`, initialisiert aus
  `application.uidNumber` (truthy ⇒ true).
- **Kein Backend-Touchpoint.**
- **Kein DB-Touchpoint.**

## Implementation Status

- Frontend: Checkbox in `registration-form.tsx` + `admin-edit-form.tsx`
  hinzugefügt, `tsc --noEmit` grün, `go build ./...` grün.
- Tests: AC-13..AC-15 als Playwright-Cases in `tests/PROJ-7-member-types.spec.ts`
  ergänzt.
- Doku: `docs/user-guide/02-member-registration.md` + `changelog.md`
  + `CHANGELOG.md` nachgezogen.
- Deploy: Helm-Upgrade auf Test-Cluster am 2026-05-24 durchgeführt.

## Open Follow-ups

- [ ] **Screenshots regenerieren.** Der bestehende Screenshot
  `register-form-mitgliedstyp.png` zeigt noch den Stand vor PROJ-63
  (ohne die USt-Pflicht-Checkbox). Beim nächsten lokalen
  `npm run screenshots`-Lauf den Generator ggf. um einen
  Sub-Section-Shot für „USt-Pflicht bei Unternehmen und Verein"
  ergänzen (Pattern: `shootCard` oder `shootSettingsSection`
  unter dem Mitgliedstyp-Block — siehe Memory
  `project-user-guide-authoring`).
- [ ] Spec-Status auf **Deployed** setzen, sobald die Screenshots
  nachgezogen sind.
