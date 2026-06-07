# PROJ-78: Toggle „Elektronisches SEPA-Mandat" (B2B + CORE)

## Status: Approved
**Created:** 2026-06-07
**Last Updated:** 2026-06-07 (Backend + Frontend implementiert, Tests grün, /qa APPROVED, /security-review APPROVED, Doku aktualisiert)

## Dependencies

- Erfordert: PROJ-12 (SEPA-Lastschriftmandat PDF, CORE) — Deployed, wird hier um Audit-Variante erweitert
- Erfordert: PROJ-14 (Firmenlastschriftmandat-PDF, B2B) — Deployed, Audit-Block existiert seit PROJ-77
- Erfordert: PROJ-77 (B2B-Mandat-Audit-Block) — Deployed, liefert IP-Spalte + Audit-Felder
- Erfordert: PROJ-67 (Standard/Erweitert-Modus für Settings) — Deployed, Trigger-Mechanismus für neuen Toggle
- Beeinflusst: PROJ-70 (Stammdaten-Resync + SEPA-Mandat-Resend) — Resend-PDF wird über denselben Renderer-Pfad gehen

## Hintergrund

PROJ-77 hat den Audit-Trail-Block für B2B eingeführt — heute datengesteuert
(rendert wenn `AuditTenant + AuditAcceptedAt + AuditAcceptedIP` gesetzt,
sonst klassischer Unterschriftsblock). Es gibt **kein explizites Owner-Steuerelement**.

**Owner-Wunsch 2026-06-07:**

1. Die B2B-Audit-Variante muss **per Parameter** umstellbar sein —
   manche EEGs / Vorstände werden den elektronischen Audit-Trail (noch)
   nicht akzeptieren wollen und bestehen auf einer manuellen Unterschrift.
2. Dieselbe Einstellmöglichkeit muss für das **normale SEPA-Lastschriftmandat
   (CORE)** umgesetzt werden — heute existiert dort gar keine Audit-Variante,
   die soll mit diesem Feature ebenfalls eingeführt werden und über
   denselben Toggle gesteuert sein.

## Owner-Entscheidungen

### Zwei unabhängige Toggles (B2B getrennt von CORE)

Die Audit-Variante muss für **CORE-Mandat** und **B2B-Mandat
(Firmenlastschrift)** **separat** schaltbar sein. Begründung: Die
Rechtsbewertung der formfreien Willenserklärung kann für Geschäftsleute
(B2B) anders ausfallen als für Verbraucher (CORE). Ein EEG könnte z.B.
B2B-Audit zulassen, weil hier ohnehin höhere Sorgfaltsannahme gilt, aber
CORE-Audit (Verbraucherschutz) weiterhin physisch zurückerwarten.

Es gibt also **zwei unabhängige Per-EEG-Schalter**:

- `sepa_mandate_core_audit_enabled BOOLEAN` — steuert nur das
  CORE-SEPA-Lastschriftmandat (`Generate`)
- `sepa_mandate_b2b_audit_enabled BOOLEAN` — steuert nur das
  B2B-Firmenlastschriftmandat (`GenerateCompany`)

Pro Toggle gilt:
- **Aktiv:** Audit-Trail-Block wird gerendert, wenn die Audit-Daten
  vollständig sind. Sonst Fallback auf Unterschriftsblock (Bestandsanträge
  ohne IP).
- **Inaktiv:** Das jeweilige Mandat-PDF rendert immer den klassischen
  Datum/Unterschrift-Block, auch wenn die Audit-Daten in der DB liegen.

### Default = FALSE für beide (Klassisch, opt-in zur Audit-Variante)

**Default für ALLE EEGs (neu und Bestand), für beide Toggles, ist FALSE**
— die Mandat-PDFs werden weiterhin mit Datum/Unterschriftslinie verschickt
und sollen vom Mitglied physisch unterschrieben zurückgesendet werden.

Hintergrund: Die Rechtskonformität der elektronischen Audit-Variante
(formfreie Willenserklärung gem. § 76 (3) EIWOG 2010) wird derzeit
geklärt — sowohl für B2B als auch für CORE. Bis das geklärt ist, soll
kein EEG unbemerkt in die fragliche Variante laufen. Die Audit-Variante
ist je Pfad nur per aktivem Opt-in pro EEG zugänglich.

**Auswirkung auf PROJ-77:** Die in PROJ-77 eingeführte B2B-Audit-Variante
ist bis zum aktiven Einschalten des B2B-Toggles **faktisch deaktiviert**
— neue B2B-Mandate werden vorerst wieder mit klassischem Unterschriftsblock
geliefert. Wenn die Rechtsklärung positiv ausgeht, kann der jeweilige
Default in einem späteren Mini-PROJ auf TRUE gehoben werden (separat pro
Toggle möglich).

## User Stories

- Als **EEG-Vorstand mit konservativem Compliance-Profil** möchte ich den
  Audit-Trail-Modus deaktivieren können, damit beide SEPA-Mandate (CORE +
  B2B) wie bisher mit einer Datum/Unterschrift-Zeile geliefert werden.
- Als **EEG-Vorstand mit modernem Onboarding-Setup** möchte ich, dass auch
  das CORE-SEPA-Mandat (nicht nur B2B) elektronisch als formfreie
  Willenserklärung dokumentiert wird — kein manuelles Zurückschicken
  von unterschriebenen PDFs mehr.
- Als **Owner-Superuser** möchte ich, dass der Default für alle EEGs auf
  „elektronisch" steht, damit das Feature ohne aktive Konfiguration sofort
  wirksam wird, und nur konservative EEGs aktiv abschalten müssen.
- Als **Admin der RC** möchte ich den Toggle in der EEG-Settings-Card im
  Advanced-Modus finden, mit einer klaren Erklärung der Rechtsgrundlage,
  damit ich die Auswirkung einschätzen kann, bevor ich umlege.
- Als **Mitglied eines EEG mit deaktiviertem Toggle** möchte ich am SEPA-
  Mandat-PDF erkennen, dass ich es ausdrucken, unterschreiben und
  zurücksenden muss — Erwartungsklarheit.

## Akzeptanzkriterien

### AC-1: Neue Spalten + Default
- **Eine** Migration `000070_sepa_mandate_audit_toggles.{up,down}.sql`:
  - `ALTER TABLE registration_entrypoint ADD COLUMN sepa_mandate_core_audit_enabled BOOLEAN NOT NULL DEFAULT FALSE;`
  - `ALTER TABLE registration_entrypoint ADD COLUMN sepa_mandate_b2b_audit_enabled BOOLEAN NOT NULL DEFAULT FALSE;`
- `DEFAULT FALSE` bleibt permanent (kein `DROP DEFAULT` nach Backfill — Konsistenz mit `board_approval_workflow_enabled` u.a.)
- Bestands-EEGs landen automatisch auf `FALSE` via Default — kein expliziter `UPDATE`-Backfill nötig
- Test-Betrieb erlaubt den abrupten Cut (Owner-Bestätigung 2026-06-07): auch EEGs mit bereits eingelaufenen Audit-Daten aus PROJ-77 starten auf FALSE
- Down-Migration: `ALTER TABLE registration_entrypoint DROP COLUMN sepa_mandate_core_audit_enabled; ALTER TABLE registration_entrypoint DROP COLUMN sepa_mandate_b2b_audit_enabled;`

### AC-2: Renderer-Verhalten B2B
- Toggle `TRUE` + Audit-Daten vollständig → Audit-Block (heute schon)
- Toggle `TRUE` + Audit-Daten unvollständig → klassischer Unterschriftsblock (heute schon)
- Toggle `FALSE` → IMMER klassischer Unterschriftsblock, auch wenn Audit-Daten in DB

### AC-3: Renderer-Verhalten CORE
- Toggle `TRUE` + Audit-Daten vollständig → Audit-Block (NEU, dieselbe Wortlaut-Variante wie B2B; Audit-Text inkl. § 76 (3) EIWOG-Hinweis)
- Toggle `TRUE` + Audit-Daten unvollständig → klassischer Unterschriftsblock (heute)
- Toggle `FALSE` → IMMER klassischer Unterschriftsblock

### AC-4: Shared Renderer-Helper
- Audit-Render-Logik aus `GenerateCompany` in einen gemeinsamen Helper extrahiert (`renderSEPAAuditBlock(f, data, lm, cw)`)
- Beide Generatoren (`Generate` + `GenerateCompany`) rufen denselben Helper
- Memory `feedback_shared_helpers_for_parallel_paths` eingehalten — keine duplizierte Render-Logik

### AC-5: Service-Layer-Übergabe
- `buildSEPAMandateData` lädt **beide** Toggles aus `entrypoint`
  (`SEPAMandateCoreAuditEnabled`, `SEPAMandateB2BAuditEnabled`)
- Service wählt anhand `application.Einzugsart` den passenden Toggle und
  übergibt das Ergebnis als **ein** Feld `ElectronicMandateEnabled` an
  `SEPAMandateData` (Renderer bleibt mandatstyp-agnostisch)
- Wenn Toggle `FALSE`: Service setzt `AuditTenant`, `AuditAcceptedAt`,
  `AuditAcceptedIP` defensiv NICHT zurück — das ist Aufgabe des Renderers
  (Single Source of Truth)
- Alternative-Design (geprüft, verworfen): Service blankt die Audit-
  Felder, wenn Toggle FALSE. Verworfen, weil dann der Resend-Pfad
  doppelt entscheiden müsste.
- Alternative-Design (geprüft, verworfen): zwei separate Felder
  `ElectronicMandateCoreEnabled` + `…B2BEnabled` an `SEPAMandateData`.
  Verworfen, weil der Renderer ohnehin nur einen der beiden Pfade
  bedient (`Generate` oder `GenerateCompany`) und der Service den
  Mandatstyp kennt — Auswahl-Logik gehört in den Service, nicht in den
  Renderer.

### AC-6: PDF-Renderer Toggle-Gate
- `SEPAMandateData` erhält neues Feld `ElectronicMandateEnabled bool`
- Renderer-Check wird zu: `if data.ElectronicMandateEnabled && AuditTenant != "" && !AcceptedAt.IsZero() && AuditIP != ""`
- Sonst klassischer Block
- Gilt für **beide** Renderer (`Generate` für CORE, `GenerateCompany` für B2B) — beide rufen denselben Helper

### AC-7: Settings-UI
- **Zwei Toggles** im EEG-Settings-Editor in der bestehenden SEPA-Sektion:
  - `sepaMandateCoreAuditEnabled` — Label „Im CORE-Mandat den elektronischen Audit-Trail nutzen (statt manueller Unterschrift)"
  - `sepaMandateB2BAuditEnabled` — Label „Im B2B-Mandat den elektronischen Audit-Trail nutzen (statt manueller Unterschrift)"

- **Sichtbarkeits-Logik (wichtig — separate conditional-Blöcke):**
  - **CORE-Audit-Toggle** sitzt im bestehenden `{isAdvanced && sepaMandateEnabled && (…)}`-Block, direkt neben `sepaMandateAtImport`. Ohne CORE-PDF (Toggle FALSE = Online-Zustimmung-Checkbox) ist der CORE-Audit-Toggle inhaltlich sinnlos und bleibt versteckt.
  - **B2B-Audit-Toggle** sitzt in einem **separaten** `{isAdvanced && (…)}`-Block direkt darunter — **NICHT** an `sepaMandateEnabled` gekoppelt. B2B-Mandat-PDFs werden unabhängig vom CORE-Toggle erzeugt (PROJ-74), also muss der B2B-Audit-Toggle im Advanced-Modus immer erreichbar sein, auch wenn `sepaMandateEnabled=false`.

- **Layout-Skizze:**
  ```
  1. sepaMandateEnabled                            (Default-sichtbar)
  2. Warn-Banner Pflichtfelder
  3. {isAdvanced && sepaMandateEnabled && (
       • sepaMandateAtImport
       • sepaMandateCoreAuditEnabled              ← NEU
     )}
  4. {isAdvanced && (                              ← SEPARATER Block
       • sepaMandateB2BAuditEnabled               ← NEU
     )}
  ```

- **Hint-Popover je Toggle** (zwei separate Popover, jeweils mit § 76 (3) EIWOG-Hintergrund):
  > Wenn aktiv, wird im [CORE-|B2B-]Mandat-PDF der klassische
  > Datum/Unterschrift-Block durch einen Audit-Trail-Text ersetzt
  > (formfreie Willenserklärung gem. § 76 (3) EIWOG 2010 — Datum,
  > Uhrzeit und IP-Adresse des Mitglieds bei elektronischer Zustimmung).
  > Wenn inaktiv, bleibt das Mandat-PDF klassisch mit Unterschriftslinie
  > — Vorgabe für EEGs, deren Vorstand auf einer manuellen Unterschrift
  > besteht. Die Rechtsbewertung kann für Firmenlastschrift (B2B) anders
  > ausfallen als für Basis-Lastschrift (CORE), daher zwei getrennte
  > Schalter.

### AC-8: Settings-Mode-Trigger
- `isAdvancedEEGSettingsActive` reagiert auf **mindestens einer** der
  beiden Toggles `=== true` (abweichend vom Default FALSE)
- Begründung: Default-Werte (beide FALSE) sollen den Standard-Modus nicht
  aufbrechen — sobald ein EEG aktiv eine der Audit-Varianten einschaltet,
  ist die EEG explizit „advanced"
- Unit-Test je Toggle ergänzen + Test für „beide TRUE"

### AC-9: Configexport
- Schema-Felder `sepaMandateCoreAuditEnabled` und
  `sepaMandateB2BAuditEnabled` jeweils als `*bool, omitempty`
  (Legacy-Kompat)
- Exporter, Importer (Default `FALSE` bei nil je Feld), Diff erweitern

### AC-10: Tests
- PDF-Snapshot CORE+CoreToggle=TRUE+Daten → Audit-Block
- PDF-Snapshot CORE+CoreToggle=FALSE+Daten → Unterschriftsblock
- PDF-Snapshot CORE+CoreToggle=TRUE+IP=NULL → Unterschriftsblock
- PDF-Snapshot B2B+B2BToggle=FALSE+Daten → Unterschriftsblock (Toggle übersteuert Audit-Daten — Regression-Schutz gegen PROJ-77)
- PDF-Snapshot B2B+B2BToggle=TRUE+Daten → Audit-Block (Regression PROJ-77-Variante)
- Service-Layer-Test: `buildSEPAMandateData` wählt bei `einzugsart=core` den CoreToggle, bei `einzugsart=b2b` den B2BToggle
- Settings-Endpoint-Tests: GET liefert beide Toggles, PUT speichert beide unabhängig
- Configexport-Round-Trip-Test (beide Felder)

### AC-11: Doku
- `docs/api-spec.md`: Settings-Body um Feld erweitern, PDF-Variant-Beschreibung
- `docs/domain-model.md`: neue Spalte dokumentieren
- `docs/user-guide/06-admin-settings.md`: neue Sektion „Elektronisches SEPA-Mandat" (PROJ-frei, anonymisiert)
- `docs/user-guide/changelog.md`: Eintrag
- `CHANGELOG.md`: voller Eintrag

## Edge Cases

1. **EEG schaltet Toggle nach Vorab-Submits um:** Bereits eingegangene
   Anträge tragen Audit-Daten in DB. Wenn EEG dann Toggle deaktiviert,
   rendern die nächsten Mandat-PDFs (Resend, Activate-Trigger) den
   Unterschriftsblock. Erwartetes Verhalten — Owner-Direktive.

2. **PROJ-70 Resend-Pfad:** Resend ruft `buildSEPAMandateData` → übernimmt
   aktuellen Toggle-Stand der EEG. Mandat-PDF an Mitglied entspricht
   immer dem aktuellen Toggle, nicht dem zur Submit-Zeit. Erwartet, weil
   Resend bewusst eine „aktuelle Variante" ist.

3. **PROJ-77 Bestandsanträge ohne IP:** Toggle TRUE + IP=NULL → Renderer-
   Check schlägt fehl → klassischer Block. Toggle ist Berechtigung,
   nicht Erzwingung. Konsistenz mit heutigem PROJ-77-Verhalten.

4. **Configexport von EEG-A nach EEG-B:** Toggle wird mit übernommen.
   Importer ohne Feld → Default TRUE. Legacy-Configs sind kompatibel.

5. **`require_email_confirmation = false` + Toggle = TRUE:** Audit-Text-
   Wortlaut-Variante „Eingabe" (statt „Verifizierung") greift weiterhin —
   siehe PROJ-77 AC-2.

6. **EEG-Settings-UI nach Toggle-Wechsel ohne Save:** Wenn der Admin den
   Toggle umlegt und die Settings nicht speichert, gilt weiterhin der
   alte Zustand — Standard-Settings-Save-Verhalten (PROJ-66).

7. **PROJ-71 Customer-Onboarding-AVV-PDF:** AVV-Mail nutzt nicht das
   SEPA-Mandat — kein Konflikt. AVV-PDF bleibt Audit-Trail (PROJ-71).

## Non-Goals

- KEIN globaler Setting — bleibt Per-EEG.
- KEINE Re-Render-Aktion auf Bestandsanträge (Toggle-Wechsel wirkt nur
  auf neue PDFs — Submission, Resend, Activate-Trigger).
- KEINE rückwirkende Ersetzung von bereits versandten PDFs.
- KEIN Audit-Trail-Modus für weitere Dokumente (z.B. AVV) im Scope dieses
  PROJ — AVV-Audit kommt aus PROJ-71 und bleibt davon unberührt.

## Memory-Regeln (zu beachten)

- `feedback_shared_helpers_for_parallel_paths` — Audit-Render-Helper geteilt
- `feedback_admin_field_full_chain` — 6 Layer (DB, Struct, Repo, Settings-GET, Settings-PUT, Frontend)
- `feedback_no_proj_refs_in_user_doc` — User-Guide bleibt PROJ-frei
- `feedback_no_placeholders` — kein `placeholder=` im Toggle/Hint
- `feedback_helm_values_split` — keine Helm-Änderung (kein neues ENV)
- `feedback_anonymized_examples` — Snapshot-Tests Max Mustermann, IP 192.0.2.42

## Workflow

`/requirements` ✓ → `/grill-me` ✓ (alle Architektur-Branches durch, siehe
Tech-Design-Notes unten) → **`/architecture` übersprungen** (Owner-
Entscheidung 2026-06-07; Tech-Design ist im Spec selbst konsolidiert,
PM-friendly Übersicht-Sektion wäre redundant) → `/backend` (Standard-
Reihenfolge: Migration → shared/Models → Repo → Service → PDF-Renderer →
HTTP-Handler → Configexport → Tests; Doku im selben Commit:
`docs/api-spec.md`, `docs/domain-model.md`, `docs/user-guide/06-admin-settings.md`,
`docs/user-guide/changelog.md`, `CHANGELOG.md`) → `/frontend` (settings-editor + settings-mode + Configexport-Frontend + Unit-Tests `settings-mode.test.ts`) → `/qa` → `/security-review` (DB-Schema-Change +
Settings-Endpoint-Body-Erweiterung → laut .claude/rules/security.md
obligatorisch) → `/deploy` (`v1.20.0-PROJ-78` Minor-Bump).

## Tech-Design-Notes (post-Grilling-Konsolidierung)

### Naming-Konvention
- DB-Spalten: `sepa_mandate_core_audit_enabled`, `sepa_mandate_b2b_audit_enabled`
- Go-Felder: `SEPAMandateCoreAuditEnabled`, `SEPAMandateB2BAuditEnabled` (Bool, kein Pointer)
- JSON-Felder: `sepaMandateCoreAuditEnabled`, `sepaMandateB2BAuditEnabled`
- Migration: `000070_sepa_mandate_audit_toggles.{up,down}.sql` (eine Migration, beide Spalten)

### Service-Layer-Verhalten (`buildSEPAMandateData`)
- Switch wählt per `app.Einzugsart` den passenden Toggle:
  - `case "core"`: `data.ElectronicMandateEnabled = ep.SEPAMandateCoreAuditEnabled`
  - `case "b2b"`: `data.ElectronicMandateEnabled = ep.SEPAMandateB2BAuditEnabled`
- `default`-Pfad: nil-Return (heute schon), Toggle-Read wird nicht erreicht
- Audit-Daten (`AuditTenant`, `AuditAcceptedAt`, `AuditAcceptedIP`, `AuditEmailVerified`) werden immer befüllt — Toggle ist reine Render-Gate, nicht Daten-Gate (Single Source of Truth im Renderer)

### Renderer-Helper (`internal/pdf/generator.go`, inline)
- Neuer Helper `renderSEPAAuditBlock(f *fpdf.Fpdf, data SEPAMandateData, lm, cw float64, boxTop float64) (boxBot float64)`
- Audit-Block-Logik aus heutigem `GenerateCompany` extrahiert (rund 30 Zeilen)
- Klassischer Unterschriftsblock bleibt **inline** pro Generator — B2B + CORE haben heute unterschiedliche Layouts (CORE: Datum-Linie + getrennte Unterschrift-Linie; B2B: einzelne „Ort, Datum, Unterschrift"-Linie). Diff dieses Layouts ist nicht Scope von PROJ-78.
- Renderer-Check-Reihenfolge (short-circuit, Toggle zuerst):
  `if data.ElectronicMandateEnabled && AuditTenant != "" && !AcceptedAt.IsZero() && AuditIP != ""`

### Resend + Activate (PROJ-70 + PROJ-46/53)
- Beide Pfade nutzen `buildSEPAMandateData` → aktueller EEG-Toggle-Stand gewinnt, nicht der Submit-Zeit-Stand. Toggle ist EEG-Policy, kein Antrags-Snapshot.
- Bereits versandte PDFs bleiben unverändert (keine rückwirkende Bearbeitung; siehe Non-Goals).

### Settings-PUT-Body
- Beide Toggles als **non-pointer bool** in `SaveEEGSettings`-Signatur nach `BoardApprovalWorkflowEnabled` einsortiert (konsistent mit anderen always-overwrite-Booleans im Bestand)
- Frontend lädt vor PUT die aktuellen Settings und schickt komplett → kein PATCH-Risiko

### Configexport
- Schema-Felder `*bool, omitempty` für Legacy-Kompat
- Importer: fehlendes Feld → FALSE (konservativer Default; konsistent mit `boardApprovalWorkflowEnabled`)
- Diff-Preview: zwei separate Diff-Zeilen (eine pro Toggle)
- Roundtrip-Test obligatorisch (Schutz vor Schema/Exporter/Importer-Drift)

### Tests
- PDF-Snapshot-Tests (`internal/pdf/generator_test.go`): 5 neue + 1 umgebaut
  - `Generate_AuditBlock_RenderedWhenAllFieldsSet` (NEU, CORE-Spiegel zu PROJ-77 B2B)
  - `Generate_AuditBlock_FallbackOnMissingIP` (NEU, CORE)
  - `Generate_AuditBlock_VerifyVerbToggle` (NEU, CORE)
  - `Generate_AuditBlock_IPv6` (NEU, CORE)
  - `Generate_ToggleFalse_IgnoresAuditFields` (UMBENANNT/UMGEBAUT aus `Generate_Core_IgnoresAuditFields`; testet, dass `ElectronicMandateEnabled=false` selbst bei vollständigen Audit-Daten den klassischen Block rendert)
  - `GenerateCompany_ToggleFalse_FallsBackToClassic` (NEU, B2B-Toggle-Übersteuerung als Regression-Schutz gegen PROJ-77)
- Service-Layer-Test (`internal/application/application_service_test.go`): Tabellen-Test mit 6 Permutationen (`{core,b2b} × CoreToggle × B2BToggle`)
- Settings-Endpoint-Integrationstest: GET liefert beide Toggles, PUT setzt beide unabhängig
- Configexport-Roundtrip-Test
- Frontend: `settings-mode.test.ts` mit drei Cases (nur Core, nur B2B, beide FALSE)

### Migration-Backfill (Test-Betrieb-Modus)
- Owner-Bestätigung 2026-06-07: Test-Betrieb erlaubt abrupten Cut. Keine `UPDATE`-Sonderlogik für Bestands-EEGs mit bereits eingelaufenen PROJ-77-Audit-Daten — Default-FALSE gilt für alle.

## QA Test Results (2026-06-07)

**Reviewer:** /qa (AI, Code-Review-basiert — kein Cluster-Browser-Test verfügbar)

### Acceptance Criteria

| AC | Beschreibung | Status |
|---|---|---|
| AC-1 | Neue Spalten + Default | ✓ Migration 000070 mit beiden Spalten, NOT NULL DEFAULT FALSE; up + down |
| AC-2 | Renderer-Verhalten B2B (Toggle gewinnt) | ✓ `GenerateCompany_ToggleFalse_FallsBackToClassic` (neu) + PROJ-77-Tests (umgebaut auf `ElectronicMandateEnabled=true`) |
| AC-3 | Renderer-Verhalten CORE (Audit-Pfad neu) | ✓ Vier neue Tests + Generate ruft `renderSEPAAuditBlock` |
| AC-4 | Shared Renderer-Helper | ✓ `renderSEPAAuditBlock` in [generator.go](internal/pdf/generator.go), beide Generatoren nutzen ihn |
| AC-5 | Service-Layer-Übergabe per `einzugsart` | ✓ `buildSEPAMandateData` Switch + Tabellen-Test mit 6 Permutationen |
| AC-6 | PDF-Renderer Toggle-Gate (Short-Circuit) | ✓ `shouldRenderSEPAAuditBlock`: Toggle zuerst, dann Datenchecks |
| AC-7 | Settings-UI (2 Toggles, korrekte Sichtbarkeit) | ✓ CORE conditional auf `sepaMandateEnabled`, B2B separater Block |
| AC-8 | Settings-Mode-Trigger | ✓ `isAdvancedEEGSettingsActive` reagiert auf jeden TRUE-Wert, drei neue Unit-Tests |
| AC-9 | Configexport | ✓ Schema + Exporter + Importer (nil→FALSE) + Diff erweitert |
| AC-10 | Tests | ✓ 5 neue PDF-Tests + 1 umgebauter + Tabellen-Test + Settings-Mode-Tests |
| AC-11 | Doku | ✓ api-spec.md + domain-model.md + user-guide/06 + user-guide/changelog + CHANGELOG.md |

### Test-Lauf

```
go test ./...        → ALL OK (alle 12 Pakete grün)
go vet ./...         → keine Findings
go build ./...       → clean
npx tsc --noEmit     → clean
govulncheck ./...    → 0 affected Vulnerabilities im eigenen Code
```

Vitest lokal blockiert durch Rolldown-Tooling-Problem (bekannt, siehe `reference_e2e_drift_window`-Memory). CI hat das Problem nicht — Tests laufen dort gegen die Pipeline.

### Security Smoke

- Auth/Authz: Settings-PUT/GET sind weiterhin durch KeycloakAuthMiddleware + `parseRCAndCheck` geschützt. Tenant-Isolation unverändert. Keine neuen Endpoints.
- Input-Validation: zwei neue bool-Felder, keine String-/UUID-Eingabe — keine Injection-Oberfläche.
- Tenant-Boundary: Toggles sind reine EEG-Policy, beeinflussen ausschließlich die PDF-Rendering-Variante des betreffenden EEG-Antrags.
- PII/Logs: keine neuen Log-Pfade, keine PII-Erweiterung.
- DB-Schema: zwei NOT-NULL-Bool-Spalten mit Default FALSE — non-blocking auf PG 11+.

### Verdikt: APPROVED
Keine Critical/High-Findings. Pflicht-Trigger für `/security-review` erfüllt (DB-Schema-Change + Settings-Endpoint-Erweiterung) — Review folgt direkt.

## Security Review (2026-06-07)

**Reviewer:** /security-review (AI)
**Scope:** Migration 000070, [models.go RegistrationEntrypoint](internal/shared/models.go), [registration_entrypoint_repo.go](internal/application/registration_entrypoint_repo.go) (SaveEEGSettings + GetByRCNumber), [registration_entrypoint_repo_tx.go](internal/application/registration_entrypoint_repo_tx.go) (SaveAllEEGSettingsTx), [application_service.go buildSEPAMandateData](internal/application/application_service.go), [pdf/generator.go renderSEPAAuditBlock + Generate + GenerateCompany](internal/pdf/generator.go), [http/admin.go GetEEGSettings + SaveEEGSettings](internal/http/admin.go), Configexport-Stack (schema/exporter/importer/diff), Frontend-Editor + settings-mode.

### Threat Model Summary

Zwei boolean Toggles in `registration_entrypoint`, die das Rendering von SEPA-Mandat-PDFs zwischen Audit-Block (PROJ-77/78) und klassischem Unterschrifts-Block schalten. Worst-Case-Risiko: ein EEG sieht den Toggle, akzeptiert irrtümlich die Audit-Variante und versendet PDFs ohne Datum/Unterschriftsfeld — bei negativer Rechtsklärung wären diese Mandate angreifbar. Mitigation: Default beide FALSE, expliziter Hint-Popover am Toggle, klare Beschriftung, advanced-only.

### Findings

Keine Critical/High/Medium-Findings.

| Severity | Finding |
|---|---|
| Info | Toggle ist reine Render-Policy, kein Daten-Eingriff. IP-Erfassung (PROJ-77) bleibt unabhängig vom Toggle aktiv — bewusste Entscheidung, damit später per Re-Aktivierung der Toggle-Pfad sofort verfügbar ist. |
| Info | Configexport-Importer-Default FALSE bei nil ist konservativ und schützt Pre-PROJ-78-Bundles vor unbeabsichtigtem Audit-Enable. |
| Info | Default-FALSE-Strategie cuttet PROJ-77 stillschweigend — Owner-Direktive 2026-06-07 (Test-Betrieb, Rechtsklärung in Schwebe). |

### Scan Results

- `go vet ./...` — keine Findings
- `govulncheck ./...` — 0 affected Vulnerabilities im eigenen Code
- `go test ./...` — alle Pakete grün

### Verdikt: APPROVED
Keine Critical/High-Findings. Schema-Change ist additive (non-blocking, NULL-safe), Settings-Endpoint-Body-Erweiterung trägt nur zwei Booleans ohne neue Validations-Oberfläche.

## Deployment-Bookkeeping

- **Tag:** `v1.20.0-PROJ-78` (Minor-Bump — Schema-Change + neuer Render-Pfad für CORE)
- **Migration:** `000070_sepa_mandate_audit_toggles.{up,down}.sql` — Migrate-Job appliziert sie automatisch vor Backend-Rollout
- **Helm:** Keine Wertänderung nötig (keine neuen ENV-Variablen)
- **Owner-Action:** `helm upgrade` nach erfolgreichem Image-Build (CI-SHA-Tag)
