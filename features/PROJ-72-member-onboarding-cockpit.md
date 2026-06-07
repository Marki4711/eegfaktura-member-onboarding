# PROJ-72: Member-Onboarding-Cockpit (Owner-EEG-Übersicht)

## Status: Planned
**Created:** 2026-06-06
**Last Updated:** 2026-06-06

## Hintergrund

Es gibt heute keine Admin-Oberfläche, die operativ zeigt, **welche EEGs**
die Member-Onboarding-Plattform aktiv nutzen und wie viel Arbeit dort
gerade läuft. Vorhandene Sichten haben jeweils einen anderen Fokus:

| Sicht | Zweck | Was fehlt |
|---|---|---|
| `/admin/customer-onboarding/` | Liste der Plattform-Buchungen (Approve/Reject) | Zeigt nur Vertrags-Lifecycle, nicht Nutzungs-Aktivität |
| `/admin/applications/` | Antragsliste der eigenen EEG (Tenant-gefiltert) | Eine RC zur Zeit; kein Cross-EEG-Überblick |
| `/admin/settings/` | EEG-spezifische Einstellungen | Eine RC zur Zeit; reines Setup, kein Status |

Owner-Bedarf (2026-06-06): eine **Superuser-Übersicht** über alle EEGs
in `registration_entrypoint` mit operativen Live-Zählern, von der aus
man direkt in die jeweilige Antragsliste oder Einstellungen springt.

## Dependencies

- Erfordert: PROJ-71 (Customer-Onboarding-Lifecycle) — liefert den State, der pro EEG angezeigt wird.
- Sieht: `registration_entrypoint` (RC-Stamm) und `application` (Pipeline-Counts).
- Keine Schema-Änderung — reine Aggregations-Query.

## User Stories

- Als **Owner** möchte ich auf einen Blick sehen, welche EEGs die Plattform
  aktiv nutzen, damit ich Aktivität und Stillstand schnell erkenne.
- Als **Owner** möchte ich pro EEG die Anzahl offener und erledigter
  Anträge sehen, damit ich Engpässe oder Backlog-Bildung früh bemerke.
- Als **Owner** möchte ich aus der Übersicht direkt in die Antragsliste
  oder die Einstellungen einer EEG springen können, ohne den RC-Wechsel
  über die Listbox.
- Als **Owner** möchte ich nach RC-Nummer oder EEG-Name suchen können,
  um auch bei 100+ EEGs gezielt eine bestimmte zu finden.
- Als **Owner** möchte ich nach offenen Anträgen sortieren können, um
  EEGs mit dem größten Backlog zuerst anzusehen.

## Akzeptanzkriterien

### Sichtbarkeit & Zugriff

Owner-Direktive 2026-06-06: Keycloak ist nicht unter Owner-Kontrolle —
neue Keycloak-Realm-Roles können nicht jederzeit vergeben werden. Das
Cockpit braucht deshalb einen zweiten Berechtigungs-Pfad neben der
bestehenden Keycloak-Realm-Role `superuser`, **ohne** andere Owner-only
Endpoints (Customer-Onboarding-Approve, Owner-BackOffice) mit zu öffnen.

- [ ] Cockpit erreichbar unter `/admin/cockpit` (neuer Top-Level-Eintrag in der Admin-Navigation).
- [ ] Zugriff erlaubt, wenn **eine** der beiden Bedingungen erfüllt ist:
  - JWT trägt Keycloak-Realm-Role `superuser` (bestehender `IsSuperuser()`-Pfad), **oder**
  - Die im JWT-Claim `email` mitgesendete E-Mail steht in der per ENV/Helm konfigurierten Cockpit-Allowlist `COCKPIT_ALLOWED_EMAILS`.
- [ ] Vergleich case-insensitive, beide Seiten vor dem Match getrimmt + lowercased.
- [ ] Leere Allowlist (Default in `values.yaml`) → nur Keycloak-Superuser dürfen — bestehendes Verhalten, kein Regression-Risiko.
- [ ] Fehlt der `email`-Claim im Token, ist die Allowlist wirkungslos; das ist keine 500, sondern fällt auf `IsSuperuser()` zurück. Ein einzelner Warn-Log pro Session reicht.
- [ ] **Andere Owner-only Endpoints (Customer-Onboarding-Approve/Reject, Owner-BackOffice-Liste/Detail) bleiben strikt an `IsSuperuser()`** — die Allowlist gilt ausschließlich für den Cockpit-Pfad. Keine globale `IsSuperuser()`-Änderung.
- [ ] Nav-Link "Cockpit" wird nur angezeigt, wenn der eingeloggte User Cockpit-berechtigt ist. Tenant-Admins ohne Allowlist-Eintrag sehen den Link nicht.
- [ ] Direkter URL-Aufruf durch nicht-Berechtigte liefert 403 mit Hinweis "Cockpit erfordert Superuser- oder Allowlist-Berechtigung".
- [ ] Unauthentifizierter Zugriff schlägt auf der bestehenden Auth-Middleware fehl (kein Sonder-Pfad).
- [ ] Audit-Log auf jeden Cockpit-Aufruf: `subject` + `email` + `path` + Anzahl zurückgegebener EEGs + `authPath` (`"superuser_role"` oder `"cockpit_allowlist"`).

### Tabellen-Spalten (MVP)

Pro Zeile (eine Zeile pro EEG in `registration_entrypoint`):

- [ ] **RC-Nummer** — monospace, links.
- [ ] **EEG-Name** — `eeg_name` aus `registration_entrypoint`, Platzhalter "—" wenn NULL.
- [ ] **Aktiv-Badge** — visuelle Anzeige von `is_active` (grün = aktiv, grau = inaktiv).
- [ ] **Customer-Onboarding-State-Badge** — fünf Zustände gemäß PROJ-71:
  `none` / `submitted` / `approved` / `suspended` / `owner_rejected`. Farbschema entspricht dem Header-Badge aus PROJ-71.
- [ ] **Offene Anträge** — Counter, summiert über Status `submitted` + `under_review` + `needs_info`.
- [ ] **Erledigte Anträge** — Counter, summiert über Status `approved` + `imported` + `awaiting_bank_confirmation` + `ready_for_activation` + `activated`.
- [ ] **Aktionen** — zwei separate Buttons "Anträge" (→ `/admin/applications?rcNumber=XXX`) und "Einstellungen" (→ `/admin/settings?rcNumber=XXX`) nebeneinander.

Nicht im MVP-Tabellen-Bild (Owner-Entscheidung 2026-06-06): Letzte
Aktivität als Spalte, Letzter Core-Sync, Pending-Owner-Action-Glocke
— alle drei können in einer späteren Iteration nachgezogen werden.

### Sortierung

- [ ] Default-Sortierung: nach letzter Member-Onboarding-Aktivität absteigend (neueste oben).
  Berechnet als `MAX(application.updated_at)` pro EEG; NULL (EEGs ohne Antrag) sortieren ans Ende.
- [ ] Alternative Sortierung 1: "Meiste offene Anträge oben" — sortiert nach `Offene-Anträge`-Counter absteigend.
- [ ] Alternative Sortierung 2: "RC-Nummer alphabetisch" — natürliche Strings-Sortierung.
- [ ] Sortier-Wahl per Dropdown/Buttons am Tabellenkopf; der aktive Modus ist visuell hervorgehoben.

### Volltextsuche

- [ ] Suchfeld am Seitenkopf, kein Placeholder-Text (Memory-Regel feedback_no_placeholders).
- [ ] Filter wirkt clientseitig auf der bereits geladenen Liste — keine zusätzliche Roundtrip-Query.
- [ ] Sucht in RC-Nummer **und** EEG-Name (case-insensitive Substring-Match).
- [ ] Leerer Suchstring zeigt die gesamte Liste.

### Daten-Aktualität

- [ ] Live-Berechnung bei jedem Seitenaufruf — eine Aggregations-Query liefert alle EEGs mit allen Countern.
- [ ] Kein Caching im Backend — die Query muss bei 500 EEGs + 5.000 Applications **unter 300 ms** im p95 bleiben.
- [ ] Bei Aufruf-Fehler wird im UI ein Retry-Knopf gezeigt; die Seite stürzt nicht ab.

### Empty State

- [ ] Wenn keine EEGs in `registration_entrypoint` existieren: Hinweis
  "Noch keine EEG hinterlegt — anlegen über das Faktura-Core."
- [ ] Wenn die Suche keine Treffer liefert: Hinweis "Keine Treffer für '…'."

### Audit / Logging

- [ ] Cockpit-Aufrufe loggen `subject` + `path` + `Anzahl zurückgegebener EEGs`, **keine** PII.

## Edge Cases

- **Allowlist mit Whitespace und Tippfehlern:** `COCKPIT_ALLOWED_EMAILS=" eegfaktura@vfeeg.org , Owner2@Example.at "` muss bei beiden Adressen matchen — Whitespace und Großschreibung sind kein Konfigurations-Stolperstein.
- **Allowlist mit Trailing-Komma:** `"a@b.com,"` bleibt valide, leere Einträge werden verworfen statt als „leerer String matcht jeden leeren Claim".
- **User in Keycloak umbenannt:** Ändert sich die E-Mail des Allowlist-Users in Keycloak, fällt er still aus der Berechtigung — gewollt. Owner muss `values-env.yaml` nachziehen.
- **User mit Allowlist-Eintrag UND `superuser`-Role:** Beide Bedingungen erfüllt → trotzdem ein einziger Audit-Eintrag, `authPath` wird auf `"superuser_role"` gesetzt (engere Berechtigung gewinnt im Log).
- **EEGs ohne Anträge:** Die Counter zeigen `0` / `0`. Die EEG erscheint trotzdem in der Liste (Owner muss sehen, dass sie inaktiv ist).
- **EEG mit `is_active=false`:** Bleibt sichtbar — der Owner muss erkennen können, warum sie still ist. Aktiv-Badge zeigt "inaktiv".
- **Customer-Onboarding-State=`none`:** Heißt: die EEG hat das Member-Onboarding-Formular, aber die Plattform-Buchung steht noch aus. State-Badge zeigt das explizit.
- **Customer-Onboarding-State=`suspended` (Cool-Down):** Member-Form läuft weiter (siehe Cool-Down-Modell PROJ-71). Counter zeigen reale Aktivität, State-Badge weist auf Suspended hin.
- **Application-Status `rejected` und `draft`:** Werden weder als "offen" noch als "erledigt" gezählt (= Null-Pipeline). Begründung: `draft` ist Vor-Submit, `rejected` ist eine Sackgasse — beide gehören in keinen Aktiv-Indikator.
- **Application-Status `import_failed`:** Wird als "offen" gezählt (= Owner-Action-Required, auch wenn Tenant-Admin näher dran ist als Owner).
- **500+ EEGs:** Tabelle muss virtualisiert oder paginiert werden, sobald die Live-Query oder das DOM-Rendering >300 ms / 60 fps unterschreitet. MVP: einfache Liste; Optimierung erst, wenn Bench-Zahlen sie nötig machen.
- **Race zwischen Core-Sync und Cockpit-Aufruf:** Cockpit liest snapshot; falls in der Mikrosekunde nach Query Stammdaten geändert werden, sieht der Owner den alten Stand. Nicht relevant für MVP — Reload zeigt aktuellen Stand.

## Technische Anforderungen

- **Performance:** Aggregations-Query unter 300 ms p95 bei 500 EEGs + 5.000 Applications.
- **Sicherheit (Berechtigung):** Endpoint `GET /api/admin/owner-cockpit/eegs` zugänglich über `IsSuperuser()` **oder** Cockpit-Allowlist; siehe Akzeptanzkriterium "Sichtbarkeit & Zugriff".
- **JWT-Claim:** `email` muss aus dem Keycloak-Token gelesen werden — `KeycloakClaims` ist um ein `Email`-Feld zu erweitern. Bei Keycloak-Standard-Setup steckt die Adresse im `email`-Claim des `profile`/`email`-Scopes; falls nicht vorhanden, fällt der Allowlist-Pfad still durch (kein 500).
- **Konfiguration:** Neue Helm/Env-Variable `COCKPIT_ALLOWED_EMAILS` (komma-getrennt). Default-Wert leer (`""`) — bestehendes Superuser-Verhalten bleibt erhalten. Wert wird beim Server-Start einmalig geparst, normalisiert (lowercase + trim) und im Memory gehalten; Änderungen erfordern Pod-Restart.
- **Frontend-Permission-Check:** Das Frontend muss den eigenen Cockpit-Status kennen, um den Nav-Link bedingt zu rendern. Mechanismus offen für `/architecture` (typische Optionen: neues `/api/admin/me`-Endpoint oder ein dezidierter HEAD/GET gegen den Cockpit-Endpoint vor dem Render).
- **API-Vertrag:** REST + JSON, ein einziger GET-Aufruf liefert komplettes Listen-Payload.
- **Browser-Support:** Chrome, Firefox, Safari (analog Rest der Admin-UI).
- **Responsive:** Tabelle scrollt horizontal auf Mobile (375 px) — die Sicht ist primär Desktop-orientiert.
- **Helm-Disziplin:** Memory `feedback_helm_values_split` — neue Werte gehören in `values.yaml` (Struktur + leerer Default) **und** in `values-env.yaml.example` (Vorlage mit realistischer Adresse) im selben Commit.
- **Memory-Regeln:**
  - `feedback_no_placeholders` — kein Placeholder-Text auf dem Suchfeld.
  - `feedback_anonymized_examples` — in Doku/Screenshots Max Mustermann / Musterbetrieb GmbH.
  - `feedback_no_proj_refs_in_user_doc` — `docs/user-guide/**` muss PROJ-frei sein.

## Nicht im Scope (MVP)

- Owner-bearbeitbare EEG-Settings vom Cockpit aus (verbleiben in der jeweiligen Tenant-Settings-Sicht).
- Reporting/Export (CSV/Excel) — eigenes späteres Feature.
- Trend-Charts (Aktivität über Zeit).
- Alerts/Benachrichtigungen bei Backlog-Schwellen.
- Drill-Down in einzelne Application-Details (öffnet die bestehende `/admin/applications`-Sicht).
- Customer-Onboarding-Approve/Reject aus dem Cockpit heraus (verbleibt unter `/admin/customer-onboarding/`).
- Anzeige von „Letzter Core-Sync", „Letzte Aktivität als Spalte", „Pending-Owner-Action-Glocke" — bewusst aus MVP entfernt.

---
<!-- Sections below are added by subsequent skills -->

## Tech Design (Solution Architect)
_To be added by /architecture_

## QA Test Results
_To be added by /qa_

## Deployment
_To be added by /deploy_
