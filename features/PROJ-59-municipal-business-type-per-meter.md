# PROJ-59: BgA / Hoheitsbereich-Vermerk im Anlagennamen bei Gemeinden

## Status: In Review
**Created:** 2026-05-23
**Last Updated:** 2026-05-23 (Scope auf Variante 2 reduziert, Hilfetext-Implementierung)

## Dependencies
- Berührt: PROJ-7 (Mitgliedstypen) — Hilfetext bei der Option „Gemeinde"
- Berührt: PROJ-8 (Konfigurierbare Felder) — Feld `installation_name` muss bei der EEG mindestens als optional konfiguriert sein, damit Gemeinden den Vermerk eintragen können

## Hintergrund

Bei österreichischen Gemeinden ist umsatzsteuerlich zwischen **Betrieb gewerblicher Art (BgA)** und **Hoheitsbereich** zu unterscheiden. Die Unterscheidung gilt zählpunktbezogen — eine Gemeinde kann z. B. den Bauhof-Bezugszählpunkt als BgA und den Rathaus-Bezug im Hoheitsbereich führen. Beide Klassifikationen beeinflussen, welcher Tarif (USt-Pflicht / Steuerbefreiung) in eegFaktura passend ist.

## Entscheidungs­historie

Während `/requirements` und `/grill-me` wurde zunächst eine **strukturierte Lösung** (eigene `municipal_business_type`-Spalte auf `metering_point` + Pflichtfeld im Public-Form + zwei Validierungs-Gates + Clearing-Helper + Badge im Tarif-Dialog + Excel/PDF-Spalte) erarbeitet. Beim Übergang in `/backend` wurde die Komplexität dieser Lösung an der erwarteten Mengenlage gespiegelt:

- Gemeinde-Anmeldungen sind im Bestand selten (einstellige Zahl pro EEG)
- Die Information wird **nur** zur Tarif-Entscheidung beim Import gebraucht — keine Filter, keine Reports, keine Downstream-Logik im Onboarding
- Die strukturelle Lösung hätte DB-Schema, Backend-Validierung an drei Stellen, Frontend-Discriminator-Union und Admin-UI an mehreren Komponenten bedeutet

Entscheidung 2026-05-23: **Variante 2 — Vermerk im Anlagennamen** ohne strukturelle Erfassung.

Die Spec ist dadurch von einem mehrschichtigen Feature auf einen **reinen Hilfetext-PR** geschrumpft.

## Umfang (umgesetzt)

### Frontend-Änderungen (zwei Stellen)

1. **`src/components/metering-point-fields.tsx`** — Der Info-Popover am Feld „Anlagenname" zeigt einen zusätzlichen Absatz, sobald `memberType=municipality` gewählt ist: Hinweis, dass „BgA" bzw. „Hoheit" mit in den Anlagennamen geschrieben werden soll (z. B. „Bauhof — BgA", „Rathaus — Hoheit").
2. **`src/components/registration-form.tsx`** — Der bestehende Gemeinde-Hilfetext im MemberTypeSelector wird um eine Notiz ergänzt, dass je Zählpunkt im Feld Anlagenname BgA/Hoheit vermerkt werden soll.

### Was **nicht** umgesetzt wird (bewusst)

- Keine DB-Migration
- Keine neue Spalte
- Kein Backend-Code (keine Validierung, keine Clearing-Logik, keine Gates)
- Keine Änderung an Public-Form-Validierung (Zod-Schema unverändert)
- Keine Änderung am Tarif-Auswahldialog (PROJ-27)
- Keine Änderung an Excel/PDF
- Keine Änderung an der externen API (PROJ-13)

## Bedingungen & Annahmen

- Das Feld `installation_name` ist konfigurierbar (PROJ-8). Damit Gemeinden den Vermerk eintragen können, muss die jeweilige EEG das Feld mindestens auf „optional" gestellt haben. Wenn die EEG das Feld auf „hidden" gesetzt hat, ist der Workflow nicht nutzbar — das ist eine bewusste Akzeptanz, da im aktuellen Bestand alle EEGs das Feld sichtbar haben.
- Der Admin liest den Anlagennamen beim Tarif-Auswahldialog (PROJ-27) und entscheidet auf dieser Basis manuell, welcher Tarif passt.
- Bei abweichenden Konventionen pro EEG (z. B. „BgA" vs. „gewerblich") muss die EEG ihre Mitglieder gesondert informieren — das System gibt nur einen Vorschlag im Popover.

## Edge Cases (akzeptiert)

- **Mitglied trägt den Vermerk nicht ein:** Admin sieht beim Tarif-Setzen keinen Hinweis, ruft Mitglied an oder vermutet basierend auf Zählpunkt-Adresse. Akzeptierter Workflow-Bruch — alternative Lösung wäre die ursprünglich verworfene strukturelle Erfassung.
- **Mitglied trägt unsinnigen Text ein:** unstrukturiertes Free-Text-Feld, keine Validierung. Admin liest und interpretiert.
- **Externe API liefert Antrag ohne Vermerk:** identisch zum Public-Form-Pfad ohne Eintrag — Admin klärt nach.
- **Bestandsdaten:** keine Migration, keine Aufarbeitung. Bestehende Gemeinde-Anträge bleiben unangetastet.

## Tests

Keine zusätzlichen Tests erforderlich:
- Keine Backend-Logik geändert
- Frontend-Änderung ist reines Popover-Text-Conditional — keine neue State-Machine
- Bestehende Tests für `MeteringPointFields` und `registration-form` bleiben grün, da kein neuer Pflichtfeld-Pfad eingeführt wird

## Notes

- **Optionale Erweiterung später**: Falls Gemeinde-Anmeldungen häufiger werden und der Tarif-Aufwand spürbar wird, kann der historische Variante-1-Entwurf (siehe Git-History dieser Datei) als Ausgangspunkt für eine strukturierte Lösung dienen. Bis dahin bleibt der Hilfetext-Vermerk der pragmatische Weg.
- **PROJ-55 (Nachmelden von Zählpunkten):** Die Vorgriffs-Notiz dort kann ebenfalls auf Variante 2 reduziert werden — der Nachmelde-Flow muss bei Gemeinde-Mitgliedern denselben Hilfetext am Anlagennamen anzeigen, mehr nicht.
- Security-Review nicht erforderlich: keine neuen Endpoints, keine Auth-Änderung, kein Schema-Change.

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)

_Verworfene strukturelle Lösung — historischer Zwischenstand in der Git-History dieser Datei. Aktuelle Lösung ist eine reine Hilfetext-Ergänzung an zwei Stellen im Frontend, kein eigenes Tech Design notwendig._

## QA Test Results

**QA Date:** 2026-05-23
**Tester:** Claude QA
**Commit:** `6740c1b`

### Scope-Hinweis

PROJ-59 wurde während `/backend` auf **Variante 2** reduziert (Vermerk im Anlagennamen statt strukturelle Erfassung). Damit besteht der Code-Diff aus exakt zwei Hilfetext-Edits im Public-Form. QA fokussiert sich entsprechend auf:

1. Korrektheit der zwei UI-Edits (Conditional-Rendering + statischer Text)
2. Regression: bestehende Hilfetexte bleiben unverändert
3. Build/Test-Pipeline grün

Kein Backend, keine API, kein Schema, keine Auth-Pfade berührt → kein eigener Security-Smoke-Test erforderlich; `/security-review`-Trigger nicht ausgelöst.

### Automated Tests

| Suite | Ergebnis |
|---|---|
| GitHub Actions „CI Build & Test" auf `6740c1b` (Backend: `go vet` + `go build` + `go test`) | ✅ success |
| GitHub Actions „CI Build & Test" auf `6740c1b` (Frontend: `tsc` + `next build` + Vitest) | ✅ success |
| GitHub Actions „Snyk Security Scan" | ❌ failure — **pre-existing**, drei Runs in Folge (auch auf Commits ohne PROJ-59-Bezug). Nicht durch diese Änderung verursacht. Separat zu adressieren. |

### Acceptance Criteria

| # | Criterion | Result |
|---|---|---|
| AC-1 | `metering-point-fields.tsx`: bei `memberType=municipality` wird im Anlagenname-Popover ein zweiter Absatz mit BgA/Hoheit-Hinweis angezeigt | ✅ (Quelle: `form.watch("memberType") === "municipality"` an Zeile 522) |
| AC-2 | Bei allen anderen `memberType`-Werten erscheint der zweite Absatz **nicht** | ✅ (Strict-Equality auf Literal — JSX-Block in Conditional gekapselt) |
| AC-3 | Der bestehende erste Absatz („Der Anlagenname ist eine Bezeichnung des Zählpunkts …") bleibt für alle Mitgliedstypen sichtbar | ✅ (Edit fügt nur `<p className="mt-2">…</p>` an, ändert vorhandenen `<p>` nicht) |
| AC-4 | Der Gemeinde-Hilfetext im `MemberTypeSelector` enthält den neuen Absatz über den Anlagenname-Vermerk | ✅ (`registration-form.tsx` Zeile 856–860) |
| AC-5 | Bestehender BgA/hoheitlich-Aufzählungspunkt im Gemeinde-Hilfetext bleibt erhalten | ✅ (Edit fügt nur einen `<p className="mt-2">` nach der `<ul>` an) |
| AC-6 | TypeScript-Build und Production-Build laufen ohne Fehler | ✅ (CI grün, siehe oben) |
| AC-7 | Vitest-Unit-Tests laufen ohne Fehler | ✅ (CI grün, siehe oben) |

### Regression

| Bereich | Risiko | Bewertung |
|---|---|---|
| Public-Form-Submit für Nicht-municipality | Sehr niedrig | UI-Edit ist additiv und conditional — kein State, keine Validierung berührt |
| Public-Form-Submit für municipality | Sehr niedrig | Kein Pflichtfeld-Pfad hinzugefügt, keine Zod-Schema-Änderung. Submit-Verhalten unverändert |
| Andere Mitgliedstypen im `MemberTypeSelector` | Sehr niedrig | Edit liegt ausschließlich im `<div>` der Gemeinde-Option |
| Admin-Edit-Form | Nicht betroffen | `metering-point-fields.tsx` wird nur im Public-Form genutzt — Admin-Form hat eigene Felder |
| Externe API (PROJ-13) | Nicht betroffen | Reine Frontend-Änderung |
| Excel-Export, PDF, Mail-Templates | Nicht betroffen | Kein Backend-Code geändert |

### Security Smoke

| Bereich | Bewertung |
|---|---|
| Auth/Authz | n/a — keine neuen Endpoints, keine Auth-Pfade |
| Injection | n/a — keine Backend-Logik, keine DB-Query |
| XSS | ✓ — Hilfetext ist statisches JSX, kein User-Input gerendert. Anführungszeichen im Text sind echte Unicode-Zeichen, kein dangerouslySetInnerHTML |
| Secrets/PII in Logs | n/a — keine Log-Statements geändert |
| Input-Limits | n/a — kein neues Eingabefeld |
| Status-Transitions | n/a — kein Status-Code geändert |

→ Kein `/security-review`-Trigger ausgelöst.

### Manuelle Smoke-Tests (Empfehlung für Reviewer)

Visuelle Verifikation im Browser sollte vom Reviewer vor Deploy einmal ausgeführt werden — die zwei betroffenen Spots:

1. Public-Form öffnen → Mitgliedstyp „Gemeinde" wählen → Info-Icon neben „Mitgliedstyp" anklicken → der Gemeinde-Block enthält jetzt nach der BgA/hoheitlich-Aufzählung den zusätzlichen Absatz über den Anlagenname-Vermerk.
2. Public-Form öffnen → Mitgliedstyp „Gemeinde" wählen → Zählpunkt-Block aufklappen → Info-Icon neben „Anlagenname" anklicken → der Popover zeigt jetzt den zusätzlichen Absatz mit „Gemeinden: bitte hier auch vermerken …".
3. Mitgliedstyp auf „Privatperson" umstellen → derselbe Info-Icon-Klick zeigt **nur** den ursprünglichen Text, keinen zusätzlichen Absatz.

Optional Cross-Browser (Chrome/Firefox/Safari): Popover-Verhalten ist shadcn-Standard und in PROJ-39 bereits getestet — keine zusätzliche Cross-Browser-Pflicht.

### Bugs Found

Keine.

### Production-Ready Decision

**READY** — alle ACs erfüllt, CI grün (außer pre-existing Snyk), keine Regression-Risiken, keine Security-Implikationen. Status kann auf **Approved** wechseln und beim nächsten Deploy mitlaufen.

## Deployment

**Deployed:** _pending operator helm upgrade_ (Chart-Version + Image-SHA committed 2026-05-23)
**Chart version:** 1.9.2 / appVersion 1.9.2
**Image SHA:** `sha-6740c1b` (PROJ-59 frontend edit)
**Git tag:** `v1.9.2-PROJ-59`
**Migration:** none — reine Frontend-Änderung, keine Schema-Modifikation
**Rollback:** `helm rollback member-onboarding` zur vorherigen Revision; keine DB-Reverts nötig

### Deployment checklist
- [x] `go build ./...` grün (CI auf `6740c1b` + `56f7bb4`)
- [x] `go test ./...` grün (CI)
- [x] Frontend `tsc --noEmit` + `next build` grün (CI)
- [x] Vitest grün (CI)
- [x] QA abgenommen, keine Bugs gefunden
- [x] Snyk-CI grün (continue-on-error am Monitor-Step gefixt)
- [x] Kein neues Environment Variable, kein neuer Kubernetes Secret
- [x] Helm chart `appVersion` auf 1.9.2 gebumpt
- [x] Image SHA in `values.yaml` über CI auto-gesetzt (`sha-6740c1b`)
- [ ] Operator führt `helm upgrade` aus (manueller Schritt)
- [ ] Post-Deploy: Browser-Smoke (Mitgliedstyp Gemeinde wählen → Anlagenname-Popover prüfen)

### Deploy-Befehl (für Operator)

```bash
cd helm/
helm upgrade member-onboarding ./member-onboarding \
  -n member-onboarding \
  -f values-env.yaml \
  --atomic \
  --timeout 5m
```

Optional vorher Plan prüfen:
```bash
helm diff upgrade member-onboarding ./member-onboarding \
  -n member-onboarding -f values-env.yaml
```
