# PROJ-35: Per-EEG-Referenznummern (`<RC>-<Jahr>-<NNNN>`)

## Status: In Review
**Created:** 2026-05-14
**Last Updated:** 2026-05-14 (Migration 000033 + Repo + Submit-Pfad implementiert; alte Refs bleiben unverändert)

## Dependencies
- Ersetzt: Migration 000025 (`reference_number_sequence` — globaler Sequenz-Zähler `MO-YYYY-NNNNNN`)
- Berührt: PROJ-1 (Public Registration, Submit-Pfad vergibt die Ref), Mails und PDFs die die Ref rendern

## Hintergrund

Die aktuelle Referenznummer hat Format `MO-2026-000043` — ein **global** hochzählender Sequenz-Wert. In einem Multi-EEG-Setup ist das verwirrend:

- EEG A bekommt sein erstes Mitglied mit Ref `MO-2026-000043`, weil davor 42 andere Anträge bei anderen EEGs eingegangen sind.
- Nicht selbsterklärend, welche EEG der Antrag betrifft.
- Beim Vergleich mit anderen Listen / Excel-Exports ist nicht direkt zuzuordnen.

**Neue Konvention:** `<RC>-<Jahr>-<NNNN>`, z.B. `RC105720-2026-0001`.

- Pro EEG und Jahr ein eigener Counter, startet bei 1.
- 4-stelliger Counter reicht für 9 999 Anträge pro EEG pro Jahr — selbst die größten EEGs liegen weit darunter.
- Counter resettet jeden 1. Januar auf 0001 (neuer Jahres-Bucket).
- Full RC-String im Prefix, nicht nur die Ziffern — Copy-Paste-Eindeutigkeit.

## Migration-Strategie

**Existierende Refs (alte `MO-YYYY-NNNNNN`) bleiben unverändert.** Das ist wichtig weil:
- Refs sind in bereits verschickten Bestätigungs-, Willkommens- und Genehmigungs-Mails referenziert.
- Refs werden teilweise in eegFaktura-Verknüpfungen / Excel-Exports / Buchhaltungen außerhalb unserer Hoheit verwendet.

Nur **neue Anträge** ab Migrations-Deploy bekommen das neue Format. Beide Formate koexistieren im Bestand.

## Architekturentscheidungen

1. **Per-(rc_number, year)-Counter via neue Tabelle.** `member_onboarding.reference_number_counter` mit PK `(rc_number, year)`, einer Spalte `last_value`, und Tx-sicherem Increment via `INSERT … ON CONFLICT DO UPDATE … RETURNING last_value`. Atomarität liegt bei Postgres.

2. **Sequenz aus Migration 000025 wird nicht gedroppt.** Sie ist obsolet, aber das Droppen würde Tests + Code-Pfade brechen, die noch nicht migriert sind. Migration 000025 bleibt als historisches Artefakt; neue Anträge nutzen sie einfach nicht mehr.

3. **Format-Bestandteile:**
   - **RC**: full string aus `application.rc_number` (z.B. `RC105720`), nicht nur die Ziffern.
   - **Jahr**: `EXTRACT(YEAR FROM NOW())` zum Zeitpunkt der Ref-Vergabe (= bei Submit, oder bei Application-Erzeugung — siehe Q1 unten).
   - **NNNN**: 4-stellig zero-padded, ab `0001`. Bei `last_value > 9999` (theoretisch) wirft die Generation einen Error, mit Hinweis „Counter-Überlauf, format manuell anpassen". Real wird das nie passieren.
4. **Nicht eindeutig im Schema-Constraint.** `application.reference_number` ist bereits UNIQUE (DB-Constraint aus 000001). Neue Refs sind innerhalb ihrer (RC, Jahr)-Gruppe pro Counter eindeutig + zwischen Gruppen disjunkt qua Prefix → global eindeutig. Constraint wird **nicht** gelockert.

## Acceptance Criteria

### Stage A: DB-Migration
- [ ] `db/migrations/000033_per_eeg_reference_counter.up.sql`:
  ```sql
  CREATE TABLE member_onboarding.reference_number_counter (
    rc_number  TEXT NOT NULL,
    year       INT  NOT NULL,
    last_value INT  NOT NULL DEFAULT 0,
    PRIMARY KEY (rc_number, year)
  );
  ```
- [ ] `.down.sql` droppt die Tabelle.

### Stage B: Backend-Logik
- [ ] Neue Repo-Methode `ApplicationRepository.NextReferenceNumberTx(tx *sql.Tx, rcNumber string, year int) (string, error)`:
  ```sql
  INSERT INTO member_onboarding.reference_number_counter (rc_number, year, last_value)
  VALUES ($1, $2, 1)
  ON CONFLICT (rc_number, year) DO UPDATE
     SET last_value = reference_number_counter.last_value + 1
   RETURNING last_value
  ```
  Wert wird zu `fmt.Sprintf("%s-%d-%04d", rcNumber, year, lastValue)` formatiert.
- [ ] Counter-Überlauf (`last_value > 9999`) gibt sprechenden Fehler — wird in der Praxis nie greifen.
- [ ] Bestehende Submit-Logik in `application_service.go` ruft jetzt `NextReferenceNumberTx` statt `nextval('reference_number_sequence')`. Year-Wert aus `time.Now().Year()`.
- [ ] **Public-Endpoint-Verträge bleiben unverändert** — Ref wird nur im Backend generiert und in `application.reference_number` persistiert; alle Mail/PDF-Render-Stellen lesen das Feld unverändert weiter.

### Stage C: Tests
- [ ] Unit-Test: zweimal `NextReferenceNumberTx` für (RC1, 2026) → 0001, 0002.
- [ ] Unit-Test: einmal RC1+2026, einmal RC2+2026 → beide 0001 (separate Buckets).
- [ ] Unit-Test: zweimal RC1+2026, einmal RC1+2027 → 0001, 0002, 0001 (Jahres-Reset).
- [ ] Test des bestehenden Submit-Flows: Ref hat neues Format `RC<digits>-YYYY-NNNN`.

### Stage D: Mail- und PDF-Rendering
- [ ] Keine Änderungen nötig — Ref wird nur aus `application.reference_number` gelesen, Format ist transparent.
- [ ] **Aber:** alle Mail/PDF-Tests die hardcoded `MO-2026-XXXXXX` matchen müssen ggf. angepasst werden. Vor Implementation: ein `grep -rE "MO-[0-9]{4}-[0-9]{6}"` über tests + templates.

### Stage E: Dokumentation
- [ ] `docs/domain-model.md`: `reference_number`-Beschreibung auf neues Format aktualisieren, Hinweis dass alte Refs (Format `MO-YYYY-NNNNNN`) im Bestand erhalten bleiben.
- [ ] `docs/api-spec.md`: bei jedem Endpoint der eine Ref zurückgibt, das neue Format kurz erwähnen.
- [ ] `docs/user-guide/04-admin-applications.md` (oder wo immer die Ref erwähnt wird): neues Format dokumentieren.
- [ ] `CHANGELOG.md`: Eintrag, dass alte Refs unverändert bleiben.

## Open Questions

### Q1: Year zum Submit-Zeitpunkt oder zum Application-Erzeugungs-Zeitpunkt?
Die Application durchläuft `draft → submitted`. Beim Erstellen (`draft`) hat sie noch keine Ref; die Ref wird beim Submit vergeben. Daher: **Submit-Zeitpunkt**. Bei einem Antrag der am 31.12. erstellt und am 02.01. abgesendet wird, bekommt er Year=neues Jahr — semantisch sinnvoll.

### Q2: Was wenn eine RC mehrmals umbenannt wird?
Outside-Scope. RC-Strings sind im aktuellen Modell konstant pro EEG. Falls je nötig: Counter-Tabelle wird mit dem alten RC-String gespeichert, manueller Eingriff bei Migration.

### Q3: Was ist mit dem CompanyName / Excel-Export?
Excel-Spalte „Reference Number" zeigt einfach `application.reference_number` — Format ist für sie egal. Kein Anpassungsbedarf.

## Out of Scope

- Migration der Bestands-Refs auf neues Format (siehe Migration-Strategie — bewusst nicht).
- Konfigurierbarer Format-String pro EEG (z.B. anderer Prefix). Wäre overengineering.
- Eindeutigkeit über mehrere Jahre hinweg innerhalb derselben RC garantieren (Format selbst ist eindeutig durch Year).

## Pointer-Files

- Spec: `features/PROJ-35-per-eeg-reference-numbers.md` (diese Datei)
- Bestehende Sequenz: `db/migrations/000025_reference_number_sequence.up.sql`
- Submit-Pfad: `internal/application/application_service.go` (Submit-Funktion, ruft nextval)
- Repo-Schicht: `internal/application/application_repo.go`
