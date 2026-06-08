# PROJ-87: USt-Pflicht-Status in der Antrags-Detail-Ansicht sichtbar machen

## Status: Approved
**Created:** 2026-06-08
**Last Updated:** 2026-06-08
**Typ:** UX-Polish (Tester-Feedback)

## Hintergrund

Tester-Feedback 2026-06-08 (Abend, nach PROJ-86-Deploy):

> Bei den Daten des Antragstellers sieht man nicht, ob der Antragsteller
> ein Kleinunternehmer ist. Dafür gibt es aber eine Checkbox beim
> Registrierungsdialog.

Owner-Bestätigung:
> In Bearbeiten ist der Wert scheinbar ersichtlich.

Diagnose:
- Public-Form (`registration-form.tsx`) hat eine USt-Pflicht-Checkbox
  (`vatLiable`) als UI-Gate für die UID-Eingabe (PROJ-63)
- Admin-Edit-Form (`admin-edit-form.tsx:100-102`) zeigt denselben Status
  als Checkbox, abgeleitet aus `!!(uidNumber && uidNumber.trim() !== "")`
- Admin-Read-Only-Detail (`admin-application-detail.tsx`) zeigt nur die
  UID-Nummer (befüllt oder leer); der Admin musste den Schluss
  „UID leer ⇒ Kleinunternehmer" im Kopf nachvollziehen

PROJ-63-Direktive: `vatLiable` ist **kein DB-Feld**, sondern reines
UI-Gate. Die Ableitung muss client-seitig erfolgen.

## Owner-Direktive 2026-06-08

> „A" — direkt umsetzen wie PROJ-86 als Hotfix-PROJ, ohne /grill-me oder
> /architecture-Phase.

## Scope

### Betroffen
- `src/components/admin-application-detail.tsx` — neuer `<Field>`-Eintrag
  zwischen UID-Nummer und Firmenbuch-/Vereinsnummer

### Nicht betroffen
- Backend (PROJ-63-Direktive: kein DB-Feld)
- Public-Form (zeigt die Checkbox bereits)
- Edit-Form (zeigt die Checkbox bereits)
- API (kein neues Feld)
- Helm/Migration

## Acceptance Criteria

- [x] **AC-1** Bei Mitgliedstyp `company` oder `association` wird in der
  Antrags-Detail-Ansicht ein zusätzlicher `<Field>`-Eintrag „USt-pflichtig"
  angezeigt
- [x] **AC-2** Wert ist `Ja` wenn `application.uidNumber` befüllt
  (nicht leer, nicht nur Whitespace), sonst `Nein (Kleinunternehmerregelung)`
- [x] **AC-3** Bei Mitgliedstyp `municipality` wird das Field NICHT
  angezeigt (USt-Pflicht-Frage dort irrelevant)
- [x] **AC-4** Bei Mitgliedstyp `private` / `farmer` wird das Field NICHT
  angezeigt (Privatpersonen, kein UID-Konzept)
- [x] **AC-5** Position: zwischen UID-Nummer und Firmenbuch-/Vereinsnummer
  (logische Gruppierung der Steuer- und Register-Identifikationen)
- [x] **AC-6** Ableitungs-Logik identisch zur Edit-Form
  (`!!(application.uidNumber && application.uidNumber.trim() !== "")`) —
  Verhalten und Detail-Ansicht müssen denselben Wert zeigen
- [x] **AC-7** Code-Kommentar verweist auf Edit-Form als
  Single-source-of-truth-Anker
- [x] **AC-8** Build clean (tsc + vitest + Next-Production-Build)

## Edge Cases

- **EC-1 Altes Antrag mit `uidNumber=null` aus Pre-PROJ-63-Zeit:**
  Ableitung ergibt `false` → „Nein (Kleinunternehmerregelung)". Korrekt
  — Pre-PROJ-63-Antrag konnte UID nicht erfassen, also war das Mitglied
  faktisch Kleinunternehmer.
- **EC-2 `uidNumber` enthält nur Whitespace:** Trim-Check schließt das
  aus → „Nein". Identisch zum Edit-Form-Verhalten.
- **EC-3 `application.memberType` ungültig oder unbekannt:** Field wird
  nicht angezeigt (defensive Branching: nur company + association
  triggern das Rendering).
- **EC-4 Backward-Compatibility:** AC-3/AC-4 bewahren das heutige
  Verhalten für municipality/private/farmer (kein Mehraufwand für den
  Admin bei diesen Mitgliedstypen).

## Tech Design

Frontend-only, 5-Zeilen-Edit. Position im JSX:

```
{...company/association-Block...}
  Firmenname/Vereinsname/Organisationsname
  UID-Nummer
  USt-pflichtig                    ← PROJ-87 NEU
  Firmenbuch-/Vereinsnummer        (bereits da, conditional)
  E-Mail
  Telefon
```

Ableitung clientseitig:
```ts
application.uidNumber && application.uidNumber.trim() !== ""
  ? "Ja"
  : "Nein (Kleinunternehmerregelung)"
```

Spiegelt das Edit-Form-Pattern exakt (Memory-Regel
`feedback_shared_helpers_for_parallel_paths` — beide Pfade leiten
identisch ab).

## QA Test Results

**Datum:** 2026-06-08
**Reviewer:** QA Engineer (AI, Code-Review)
**Status:** Approved

### Test-Status

```
$ npx tsc --noEmit       →  clean
$ npx vitest run          →  88 / 88 grün (unverändert)
$ npm run build           →  Next-Production-Build clean
```

### Security-Smoke

- Pure Frontend-Änderung, kein Endpoint
- Kein User-Input rendert direkt (Field-Werte sind Constants oder aus
  DB-Daten)
- Kein neuer API-Pfad, kein neuer Endpoint
- Tenant-Isolation unverändert

**0 Findings.** `/security-review` nicht erforderlich.

### Regression

- PROJ-62 Mitgliedstyp-Verschmelzung: unverändert
- PROJ-63 USt-Pflicht-UI-Gate (Public-Form): unverändert
- PROJ-63 Admin-Edit-Form Checkbox: unverändert (Single source der
  Ableitungs-Logik)

**0 Regressionen.**

**Production-Ready: READY.**

## Deployment

**Datum:** 2026-06-08
**Versions-Tag:** wird beim Push gesetzt (vermutlich v1.23.5-PROJ-87)
**Image-SHA:** wird vom CI nach Push gesetzt
**Status:** wartet auf `helm upgrade` durch Owner

Owner führt `helm upgrade` manuell aus. Bundelt sich mit dem noch
offenen PROJ-86-Helm-Upgrade — ein einziger Apply bringt beide live.

---
<!-- Sections below are added by subsequent skills -->
