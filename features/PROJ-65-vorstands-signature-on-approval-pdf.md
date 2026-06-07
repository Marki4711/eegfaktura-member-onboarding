# PROJ-65 — Vorstands-Signaturblock im Beitrittsbestätigungs-PDF

**Status:** Superseded (durch PROJ-76, 2026-06-07)
**Created:** 2026-05-30
**Owner:** TBD
**Source:** Tester-Feedback (Dani Strasser, 2026-05-30)

> **Superseded-Hinweis:** PROJ-65 hätte nur einen Signaturblock am Ende des
> bestehenden Beitrittsbestätigungs-PDF ergänzt, ohne das Mail-Routing zu
> ändern (Mitglied bekommt das PDF weiter automatisch). Owner hat sich
> 2026-06-07 für die größere Workflow-Lösung entschieden: PROJ-76 trennt
> Auto-Modus und Vorstands-Genehmigungs-Modus per EEG-Toggle, generiert
> bei aktivem Toggle ein eigenes **Beitrittserklärungs-PDF** mit Signaturblock
> und routet die Mail an den EEG-Kontakt statt ans Mitglied. PROJ-65 wird
> nicht implementiert; sein Anwendungsfall (Statuten-konforme Vorstands-
> Signatur) ist von PROJ-76 vollständig abgedeckt.
>
> Siehe `features/PROJ-76-board-approval-workflow.md`.

## Hintergrund

Der Beitrittsbestätigungs-PDF (`internal/pdf/approval_pdf.go`) zeigt heute ausschließlich die Mitglieds-Daten, Zählpunkte, Zustimmungen und ggf. die Netzbetreiber-Info-Seite. **Es gibt keinen Signaturblock für den Vorstand des Vereins.**

Tester-Argument:

> „Ich gehe davon aus, dass die meisten EEGs Anträge lt. Statuten vom Vorstand signieren lassen und das dann die vertragliche Basis für den eigentlichen Stromhandelsvertrag EEG/MG ist (Prüfung durch FA möglich!!!). Von daher fehlt hier Platz/Passus für diese notwendige Formalität."

Tester hat ein Muster-PDF einer befreundeten EEG übermittelt, das als Vorlage für die Layout-Diskussion herangezogen werden soll.

## Rechtslage (Österreich, VerG 2002)

- Das VerG schreibt **keine Form** für den Mitgliedsbeitritt vor (§ 3 Abs 2 Z 5 VerG: die Statuten regeln das Wie).
- Mitgliedschaft entsteht durch Antrag des Mitglieds + Annahme-**Beschluss** des zuständigen Organs (laut Statuten meist Vorstand) + Zugang der Annahmeerklärung. Eine Unterschrift des Vorstands ist **kein** Wirksamkeitserfordernis.
- Eine beidseits signierte Beitrittsbestätigung dient als **Beweismittel** (FA-Prüfung, interne Dokumentation) — nicht als Vertragsgrundlage.
- Pflicht wird die Vorstands-Unterschrift nur, **wenn die Statuten der konkreten EEG das ausdrücklich verlangen** oder wenn EEG und Mitglied den Stromhandels-/Lieferanten-Vertrag im selben Dokument abwickeln.

→ Konsequenz: Die Anforderung gilt **EEG-spezifisch** und muss pro Tenant umschaltbar sein. Default = aus (heutiges Verhalten bleibt unverändert).

## User Stories

- **Als EEG-Admin** möchte ich pro EEG entscheiden können, ob das Beitrittsbestätigungs-PDF einen Vorstands-Signaturblock enthält, damit ich den Statuten und Praxis-Routinen meiner EEG entsprechen kann.
- **Als EEG-Vorstand** möchte ich auf dem ausgedruckten PDF eine vorbereitete Unterschriftsstelle finden, damit ich den Beitritt formal gegenzeichnen kann ohne handschriftlich Linien zu ergänzen.
- **Als Mitglied** möchte ich beim Erhalt der Beitrittsbestätigung sehen, dass die EEG den Beitritt bestätigt hat (entweder durch Beschluss-Vermerk im Text oder durch sichtbare Vorstandsunterschrift).

## Acceptance Criteria

### AC-1 — Toggle in den EEG-Einstellungen
- Neue Einstellung **„Vorstands-Signaturblock im Beitrittsbestätigungs-PDF"** unter EEG-Settings (Boolean, Default `false`).
- Toggle wird über `registration_entrypoint` persistiert (neue Spalte `approval_pdf_show_board_signature`, NULLable boolean mit Default `false`).
- Toggle ist in Configuration-Export (PROJ-61) enthalten.

### AC-2 — PDF-Rendering bei Toggle = aus
- Verhalten exakt wie heute. Keine sichtbare Änderung. Regression-Test verifiziert dies anhand des bestehenden Snapshot-Tests.

### AC-3 — PDF-Rendering bei Toggle = ein
- Am Ende des PDFs (nach „Weitere Angaben", vor optionaler Netzbetreiber-Info-Seite) wird ein Signaturblock gerendert:
  - Überschrift „Bestätigung durch den Vorstand"
  - Datumslinie + Ortlinie + Unterschriftslinie (analog zum SEPA-PDF in `internal/pdf/generator.go:211-231`)
  - Beschriftung „Datum, Ort" und „Unterschrift Vorstand"
  - Block ist eingerahmt, gleicher visueller Stil wie das bestehende Mitglieds-Unterschriftsfeld
- Der Block bleibt **leer** zum manuellen Ausfüllen — kein Pre-Fill von Name oder Datum.

### AC-4 — Layout-Robustheit
- Wenn die Netzbetreiber-Info-Seite (PROJ-56) vorhanden ist, kommt der Vorstands-Signaturblock **auf Seite 1/N (vor der Netzbetreiber-Info-Seite)** — er gehört zur Beitrittsbestätigung selbst, nicht zur Anhangseite.
- Bei sehr kurzem PDF (kein „Weitere Angaben", keine Genossenschaftsanteile) bleibt der Signaturblock am Seitenende; bei langem PDF kommt er auf einer eigenen Folgeseite.

### AC-5 — Doku
- `docs/user-guide/06-admin-settings.md` listet die neue Einstellung mit Beschreibung und Hinweis auf Vereinsrecht.
- `docs/user-guide/07-emails-and-pdfs.md` erwähnt den Signaturblock im PDF-Übersichts-Eintrag.
- `docs/user-guide/changelog.md` bekommt einen Eintrag.

## Non-Goals (für V1)

- **Digitale/elektronische Signatur** (qualifizierte Signatur, eingescannte Unterschrift einbetten) — explizit nicht in V1. Bei Bedarf eigenes Spec.
- **Mehrfach-Vorstands-Linien** (z. B. „Obmann + Schriftführer" gemeinsam) — V1 hat **eine** Unterschriftslinie. Bei Bedarf späterer Toggle „Anzahl Signaturlinien".
- **Counter-Sign-Workflow im UI** (Mitglied schickt PDF, EEG-Admin lädt signierte Version wieder hoch) — nicht in V1. PDF bleibt eine Print-Vorlage.
- **Statuten-Check** (Onboarding prüft die EEG-Statuten und schlägt Toggle-Wert vor) — Admin entscheidet selbst.

## Offene Punkte (vor `/architecture`)

1. **Position im PDF:** unmittelbar nach „Weitere Angaben", oder eigene Sektion „Bestätigung"? Tester-Muster heranziehen.
2. **Beschluss-Vermerk im Text:** soll zusätzlich zur Unterschriftslinie ein Satz wie „Der Vorstand hat die Aufnahme am … beschlossen." gerendert werden? Wenn ja: woher das Datum (= `approved_at`)?
3. **Wortlaut:** „Vorstand" vs. „Obmann/Obfrau" vs. „Vertretungsberechtigte/r" — eindeutig genug ohne EEG-spezifischen Text?
4. **Abstimmungsrunde:** Tester schlug vor, das Layout mit mehreren EEGs durchzugehen, bevor wir hart umsetzen. Sollten wir **vor** der Implementierung 2-3 Pilot-EEGs nach ihrem Muster fragen? Vermeidet, dass V1 ein Layout produziert, das nachher wieder geändert werden muss.

## Dependencies

- Keine harten Code-Dependencies. Nutzt bestehende `approval_pdf.go`-Render-Pipeline und das `registration_entrypoint`-EEG-Settings-Schema.
- PROJ-61 (Config-Export) muss das neue Feld in den Diff aufnehmen.

## Implementierungs-Skizze (vorläufig — nach `/architecture` konkretisieren)

- **DB:** Migration `000058_approval_pdf_board_signature.up.sql` + `.down.sql` — eine Spalte boolean default false.
- **Backend:** `shared.RegistrationEntrypoint` + DTO erweitern; `approval_pdf.go` bekommt einen `if data.ShowBoardSignature { renderBoardSignatureBlock(...) }`-Pfad.
- **Frontend:** Toggle im `admin-eeg-settings-editor.tsx`.
- **Tests:** Snapshot-Tests für PDF mit/ohne Toggle.
- **PROJ-61-Anpassung:** Diff-Plugin um das Feld erweitern.
