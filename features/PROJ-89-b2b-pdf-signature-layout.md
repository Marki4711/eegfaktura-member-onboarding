# PROJ-89: B2B-Klassik-PDF Signatur-Layout an CORE angleichen

## Status: Approved
**Created:** 2026-06-08
**Last Updated:** 2026-06-08 (Scope reduziert auf PDF-only)
**Typ:** UX-Polish (Tester-Befund)

## Hintergrund

Tester-Screenshot 2026-06-08:

Im B2B-Klassik-PDF (Audit-Toggle OFF) sass der Signatur-Block visuell
gequetscht:
- Datum (vorbefuellt) lag fast auf der Unterschriftslinie
- Caption „Ort, Datum, Unterschrift" auf einer schmalen 70mm-Linie
  zusammengedrueckt
- BIC-Fussnote klebte direkt am Box-Ende

Im **CORE-PDF** ist das identische Layout-Problem bereits sauber
geloest: zwei separate Linien (links Datum/Ort, rechts Unterschrift)
mit je eigener Caption.

## Owner-Direktive 2026-06-08

> „mach das so."

Direkter Hotfix, mirror den CORE-Layout-Code in der B2B-Variante.

## Scope-Reduktion (Owner-Klaerung 2026-06-08 spaet)

Owner hat erkannt, dass die **B2B-Klassik-Mail** im selben Pfad
inhaltliche Inkonsistenzen mit der PROJ-79-Praxis hat (Mail sagt
„schalten wir nach deiner Rueckmeldung frei", aber PROJ-79 legt b2b
sofort als CORE im Faktura an und aktiviert direkt). Dieser
Mail-Klartext-Fix wurde **vertagt** auf die Status-Klaerung
(`project_todo_awaiting_bank_confirmation_obsolet` Memory).

PROJ-89 beschraenkt sich daher auf den **PDF-Layout-Fix**, der
status-modell-unabhaengig ist.

## Scope

### Betroffen
- `internal/pdf/generator.go` — `GenerateCompany` Klassik-Signatur-Block

### Nicht betroffen
- Audit-Block-Pfad (PROJ-77/PROJ-78) — eigenes Render-Modul
- CORE-PDF — schon korrekt
- Mail-Templates — vertagt
- Migration, Helm, API

## Acceptance Criteria

- [x] **AC-1** B2B-Klassik-Signatur-Block hat zwei separate Linien:
  links 70mm „Datum, Ort", rechts ~95mm „Unterschrift"
- [x] **AC-2** Vorbefuelltes Datum erscheint 5mm ueber der linken Linie
  (unveraendert zum CORE-Pattern)
- [x] **AC-3** Mehr vertikales Atemraum: `f.Ln(18)` vor dem Block
  (vorher 15), `f.Ln(8)` nach dem Block vor der BIC-Fussnote
  (vorher 4)
- [x] **AC-4** Audit-Block-Pfad (`shouldRenderSEPAAuditBlock`) bleibt
  unangetastet
- [x] **AC-5** Code-Kommentar verweist auf PROJ-89 + Tester-Befund
- [x] **AC-6** `go build ./...` + `go test ./internal/pdf/...` clean

## Edge Cases

- **EC-1 `data.MandateDate.IsZero()`** (Legacy-Antrag ohne Datum):
  vorbefuelltes Datum entfaellt, restliches Layout bleibt — identisch
  zum CORE-Pattern
- **EC-2 Lange Mitglieds-Adressen am Anfang der PDF**: AutoPageBreak
  ist deaktiviert (`SetAutoPageBreak(false, 0)`). Bei extrem langen
  Texten koennte der Signatur-Block am unteren Rand kollidieren —
  Bestand-Verhalten, nicht durch PROJ-89 veraendert
- **EC-3 Owner-Mail-Klartext-Fix kommt morgen**: PDF und Mail-Text
  sind unabhaengig — der heutige PDF-Fix verbessert das visuelle
  Layout, der morgige Mail-Fix verbessert die Mitglieder-Anweisung

## Geaenderte Dateien

| Datei | Status |
|---|---|
| `internal/pdf/generator.go` | Modified — Klassik-B2B-Signatur-Block neu |
| `features/PROJ-89-b2b-pdf-signature-layout.md` | **New** — diese Spec |
| `features/INDEX.md` | Modified — Eintrag |
| `CHANGELOG.md` | Modified — Release-Notes |

## QA Test Results

```
$ go build ./...                 clean
$ go test ./internal/pdf/...     ok  0.687s
$ go test ./...                  alle 14 Pakete gruen
```

**Production-Ready: READY.**

## Deployment

**Datum:** 2026-06-08
**Versions-Tag:** v1.23.7-PROJ-89 (Patch)
**Status:** wartet auf `helm upgrade` (bundelt sich mit PROJ-86/87/88)
