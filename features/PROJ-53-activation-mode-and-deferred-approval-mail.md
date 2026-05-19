# PROJ-53 — Aktivierungs-Modus pro EEG + Beitrittsbestätigung erst bei `activated` + manueller `approved → activated`-Skip

**Status:** Planned
**Erstellt:** 2026-05-19
**Abhängigkeiten:** PROJ-46 (Post-Import-Stati), PROJ-47 (B2B-Mandat-Import), PROJ-48 (SEPA-Timing)

---

## Ziel

Drei zusammenhängende Änderungen:

1. **Beitrittsbestätigung verschieben:** Die Beitrittsbestätigungs-Mail samt PDF
   geht nicht mehr beim Übergang in `imported` raus, sondern erst beim Übergang
   in `activated`. Damit bestätigt das Onboarding gegenüber dem Mitglied
   tatsächlich erst dann den Beitritt, wenn die Mitgliedschaft im
   eegFaktura-Core abgeschlossen / die Online-Registrierung gestartet ist.
2. **Aktivierungs-Kriterium pro EEG konfigurierbar:** Eine neue
   EEG-Einstellung legt fest, woran das Onboarding erkennt, dass eine
   Anwendung von `ready_for_activation` nach `activated` wechseln darf.
3. **Manueller `approved → activated`-Skip als Ausnahmefall:** Wenn das
   Mitglied im Core bereits existiert und vom Admin dort manuell mit den
   Onboarding-Daten überschrieben wurde, soll der Onboarding-Antrag direkt
   von `approved` auf `activated` gesetzt werden können — ohne Core-Import.
   Hintergrund: Mitglieder können in eegFaktura nicht gelöscht werden, also
   bleibt in diesem Fall nur das manuelle Überschreiben des bestehenden
   Datensatzes. Der Onboarding-Antrag muss trotzdem zu einem sauberen
   `activated`-Endzustand kommen.

## Hintergrund

Heute (`PROJ-46 Stage B`) versendet das Onboarding die Beitrittsbestätigung
direkt nach `import_failed → imported` (siehe
`internal/application/admin_service.go::SendPostImportNotification`).
Das Mitglied bekommt damit eine "Willkommen — du bist jetzt Mitglied"-Mail,
obwohl der Core-Status noch `PENDING` ist und die Aufnahme im EEG noch nicht
formal abgeschlossen wurde. Manche EEGs wünschen, dass die formale Bestätigung
erst dann beim Mitglied landet, wenn die EDA-Anmeldung der Zählpunkte beim
Netzbetreiber zumindest gestartet ist.

Die heutige `SendActivatedNotification` ist eine knappe Welcome-Mail ohne
PDF — die wird durch die volle Beitrittsbestätigungs-Mail abgelöst.

## Aktivierungs-Modi

| Modus | Wert in DB | Trigger |
|---|---|---|
| **Variante A** (Default, heute) | `participant_active` | `participant.status == ACTIVE` im Core |
| **Variante B** (neu) | `any_meter_registration_started` | mindestens ein `meters[].processState ∈ {PENDING, APPROVED, ACTIVE}` im Core |

**Variante B — Mapping (verifiziert am 2026-05-19 gegen RC101294):**

| EDA-Event | Wirkung im Core | `processState` |
|---|---|---|
| `ANFORDERUNG_ECON` | Online-Registrierung initiiert | `INVALID` (bleibt) |
| `ANTWORT_ECON` ("Meldung erhalten", Code 99) | Netzbetreiber bestätigt Empfang | `PENDING` |
| `ZUSTIMMUNG_ECON` ("Zustimmung erteilt") | Netzbetreiber stimmt zu | `APPROVED` |
| `ABSCHLUSS_ECON` | Aktivierung abgeschlossen | `ACTIVE` |

→ Variante B trifft genau die "Anmeldung gestartet"-Semantik
(Netzbetreiber hat zumindest geantwortet).

## Scope

### Backend

1. **Neue Spalte** `activation_mode VARCHAR(40) NOT NULL DEFAULT 'participant_active'`
   auf `member_onboarding.registration_entrypoint`, CHECK auf die zwei
   erlaubten Werte.
2. **CoreParticipantSummary erweitern** um `Meters []CoreMeterSummary{
   MeteringPoint, Status, ProcessState }`. Die nötigen Felder liefert das
   Core-Endpoint `GET /api/participant` heute bereits — wir verwerfen sie
   aktuell nur im JSON-Decode. Body-Cap der ListParticipants-Antwort ggf.
   anheben (4 MiB sollte für ~2000 Teilnehmer reichen, vor Implementierung
   im Live-Sample prüfen).
3. **`ImportService.CheckActivations` erweitern**: pro Anwendung den
   `activation_mode` aus dem Entrypoint ziehen und entsprechend
   evaluieren — Modus A wie heute (`participant.status == ACTIVE`),
   Modus B per `meters[].processState`-Filter.
4. **Beitrittsbestätigungs-Versand verschieben:** der Code-Pfad in
   `SendPostImportNotification` zieht um in einen neuen
   `SendActivationNotification`, der vom `→ activated`-Pfad getriggert wird
   (sowohl manueller Admin-Klick als auch Activation-Check-Batch).
5. **B2B-/SEPA-Mandat-PDF bleibt bei `imported`** in einer eigenen,
   schlanken "Anlage Mandat — Beitrittsbestätigung folgt"-Mail. Der
   Mandat-Trigger und die Mandatsreferenz-Generierung bleiben unverändert,
   nur die umgebende Mail wird neu.
6. **`SendActivatedNotification` (kurze Welcome-Mail) entfällt** — sie
   wird durch die volle Beitrittsbestätigungs-Mail abgelöst.
7. **Hartes Cut-off für Bestandsanträge:** Anträge, die zum Zeitpunkt des
   Deployments bereits in `imported` / `ready_for_activation` /
   `awaiting_bank_confirmation` / `activated` stehen, bekommen
   **gar keine Activation-Mail** mehr (sie haben die Beitrittsbestätigung
   schon beim alten `imported`-Pfad bekommen). Realisierung über
   `activation_notification_sent_at TIMESTAMPTZ NULL` auf application;
   Migration setzt für Bestandsanträge in den 4 Statussen das Flag auf
   `NOW()`. Der neue Send-Pfad prüft das Flag und sendet nicht doppelt.

8. **Neuer Übergang `approved → activated` (manueller Ausnahmefall):**
   - **Erlaubt nur über eine dedizierte Admin-Route**, nicht über das
     generische `/status`-Endpoint — analog zum `POST /reset-import`-Muster.
     Z. B. `POST /api/admin/applications/{id}/mark-activated`.
   - **Mitgliedsnummer ist Pflicht-Input** im Request-Body, weil sie sonst
     fehlen würde (kein Core-Import = kein Auto-Bezug). Das Feld
     `applications.member_number` wird gesetzt.
   - Der Status-Log-Eintrag enthält Reason `"manueller Skip (Core-Member
     bereits vorhanden)"` und Actor = eingeloggter Admin.
   - Beim Übergang wird **dieselbe** Beitrittsbestätigungs-Mail versandt
     wie beim regulären `ready_for_activation → activated`-Pfad
     (PDF + Mitgliedsnummer) — über denselben `SendActivationNotification`,
     der das `activation_notification_sent_at`-Flag prüft und setzt.
   - **Kein** Core-Import, **kein** Mandat-PDF beim Import (entfällt
     komplett, weil der Member-Datensatz im Core schon existiert).
     Wenn die EEG ein SEPA-Mandat benötigt, ist das Core-seitig schon
     manuell hinterlegt.
   - **Admin-UI:** Button "Manuell aktivieren" auf der Detailansicht
     einer `approved`-Anwendung, mit Pflichtfeld Mitgliedsnummer und
     deutlicher Warnung "Nur verwenden, wenn das Mitglied im eegFaktura
     bereits manuell überschrieben wurde — Import wird übersprungen".
   - Validierung: Mitgliedsnummer-Eingabe darf nicht leer sein und sollte
     bevorzugt rein numerisch sein (analog zu `MemberNumberTaken`-Logik
     beim normalen Import).

### Frontend (Admin)

8. **Settings-Editor:** neuer Block "Aktivierungs-Modus" in
   `admin-eeg-settings-editor.tsx`, Radio mit den zwei Optionen +
   Erklärtext, der den Unterschied auf den Punkt bringt:
   - "Variante A: Mitglied wurde laut eegFaktura in die EEG aufgenommen
     (Teilnehmer-Status `ACTIVE`). Empfohlen für klassische Workflows."
   - "Variante B: Für mindestens einen Zählpunkt ist die
     Online-Registrierung beim Netzbetreiber gestartet
     (`processState ∈ PENDING/APPROVED/ACTIVE`). Frühere
     Bestätigung beim Mitglied."
9. **GET/PUT `/api/admin/settings/eeg`** um das Feld erweitern (Patch-Semantik
   wie bei den anderen Settings).

### Migrations

- `000047_application_activation_notification_sent_at.up.sql` / `.down.sql`
- `000048_registration_entrypoint_activation_mode.up.sql` / `.down.sql`

(Nummerierung an aktuellem Stand orientieren — bei Konflikt nächste freie.)

### Docs

- `docs/architecture.md` — Modus-Logik in den Activation-Check-Abschnitt
- `docs/api-spec.md` — `activationMode` in EEG-Settings + Activation-Check-Beschreibung
- `docs/domain-model.md` — neue Spalte + Flag
- `docs/operations.md` — Modus-Auswahl-Hinweis beim Onboarding einer neuen EEG
- `docs/user-guide/03-admin-eeg-settings.md` (oder vergleichbar) —
  Anleitung "Wann wechselt das Mitglied auf aktiviert?"
- `CHANGELOG.md`
- `features/INDEX.md` — Eintrag PROJ-53

## Akzeptanzkriterien

- [ ] Eine neue EEG hat per Default `activation_mode = participant_active`
  → Verhalten exakt wie heute (rückwärts­kompatibel für alle Bestands-EEGs).
- [ ] Mit `participant_active` triggert die Activation nur dann, wenn der
  Core-Teilnehmer `status = ACTIVE` ist.
- [ ] Mit `any_meter_registration_started` triggert die Activation, sobald
  mindestens ein Zählpunkt `processState ∈ {PENDING, APPROVED, ACTIVE}` hat.
- [ ] Die Beitrittsbestätigungs-Mail (mit PDF, Mitgliedsnummer, etc.) wird
  bei neuen Anträgen **nur** beim `→ activated`-Übergang versandt — nicht
  mehr bei `→ imported`.
- [ ] Bei B2B / `sepa_mandate_at_import = true` geht beim `→ imported` eine
  schlanke Begleit-Mail mit Mandat-PDF (samt Mandatsreferenz) raus — kein
  Beitrittsbestätigungs-PDF.
- [ ] Bestandsanträge in `imported/ready_for_activation/awaiting_bank_confirmation/activated`
  bekommen beim nächsten `→ activated`-Übergang **keine** weitere Mail.
- [ ] Admin-Settings-Editor zeigt den Modus-Toggle, Speichern aktualisiert
  die DB, Reload zeigt den gespeicherten Wert.
- [ ] Activation-Check-Batch berücksichtigt den Modus pro EEG (nicht global).
- [ ] Status-Log-Eintrag beim Activation-Wechsel zeigt den Trigger
  (`system:activation-check (mode=...)`) — hilfreich beim Debugging.
- [ ] Status-Transitions-Tabelle in CLAUDE.md bleibt unverändert (nur die
  Mail-Logik hängt am Wechsel, nicht die erlaubten Transitions).
- [ ] Migration für `activation_notification_sent_at` setzt das Flag
  retrospektiv für die 4 Bestands-Status — verifiziert per SQL-Spotcheck.
- [ ] Manueller `approved → activated`-Skip:
  - Dedizierte Route `POST /api/admin/applications/{id}/mark-activated`
    akzeptiert nur Pflichtfeld `memberNumber`
  - Über generisches `/status` ist der Übergang **nicht** möglich (409)
  - Mitgliedsnummer wird persistiert, Status-Log-Reason erscheint im Detail
  - Beitrittsbestätigungs-Mail wird genau einmal versandt (Flag wird gesetzt)
  - Admin-UI zeigt Button nur bei `status = approved`, mit Warnhinweis
- [ ] Tests:
  - Unit: `CheckActivations` mit Variante A vs. B
  - Unit: `SendActivationNotification` skipt bei gesetztem Flag
  - Unit: `SendImportedNotification` enthält B2B-Mandat aber kein
    Beitrittsbestätigungs-PDF
  - Unit: `MarkActivated`-Handler verlangt Mitgliedsnummer, lehnt leeren
    String ab, persistiert Status + Mitgliedsnummer + Log in einer Tx
  - Integration: durchgehender Flow `imported → activated` mit beiden Modi
  - Integration: `approved → activated`-Skip-Pfad: Status + Mail + Flag korrekt

## Offene Fragen

Keine — alle Designentscheidungen sind oben festgehalten:
- B2B-Mandat-PDF bleibt beim Import in Begleit-Mail
- Bestandsanträge bekommen gar keine Activation-Mail (hartes Cut-off via Flag)
- Modus ist pro EEG, Default = Variante A

## Risiken

- **Core-Response-Größe:** Mit `meters[]` wächst die `ListParticipants`-
  Response signifikant (513 Meter × ~600 Bytes pro Meter ≈ 300 KiB Zusatz
  bei RC101294). Aktueller Body-Cap ist 4 MiB — sollte reichen, vor
  Implementierung mit dem grössten Tenant verifizieren.
- **Activation-Check-Performance:** Modus B muss alle Meter pro Anwendung
  durchlaufen — `O(n*m)`, bei großen EEGs in Maßen. Sollte sub-second
  bleiben; Prometheus-Histogramm beobachten.
- **Mail-Volumen-Verschiebung:** Beim Stichtag werden alle ab dann neu
  importierten Anträge die Mail erst Tage später beim Activation-Wechsel
  schicken. Für die EEG-Admins wahrnehmbar — im Changelog deutlich
  kommunizieren.
