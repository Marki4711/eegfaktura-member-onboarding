# Operations Runbook — eegfaktura Member Onboarding

Operativer Leitfaden für den produktiven Betrieb. Adressaten: On-Call, Cluster-Admin, EEG-Fachadmin (für die fachlichen Auswirkungen).

> Gilt für den ATVB-Cluster mit Velero/Ceph-CSI/Wasabi-Backup-Setup (siehe Cluster-Doku `https://docs.eegfaktura.at/link/101`).

---

## 1. Backup & Restore

### 1.1 Was wird gesichert

Cluster-weit über Velero, ohne separates App-spezifisches Backup:

| Element | Mechanismus | Frequenz | TTL |
|---|---|---|---|
| Alle K8s-Ressourcen (Deployments, Services, ConfigMaps, Secrets …) | Velero → Wasabi S3 (`atsb-backup`) | daily 02:00, weekly So 03:00 | 7d / 4w |
| PostgreSQL-PVC (alle Member-Anträge, Status-Log, EEG-Settings) | Ceph CSI Snapshot + Kopia Data Movement | daily / weekly | 7d / 4w |
| **Postgres-Konsistenz** | Pre-Backup-Hook `psql -c CHECKPOINT` (Annotation am StatefulSet) | bei jedem Backup-Run | — |

**Was wird _nicht_ gesichert**:
- eegFaktura-Core (eigene Backup-Strategie, nicht in unserem Verantwortungsbereich)
- Keycloak (eigenes Backup im Cluster-Setup)
- Postal-Mail-Server-State

### 1.2 RPO und RTO

| Metrik | Wert | Begründung |
|---|---|---|
| **RPO** (max. Datenverlust) | bis 24h | daily Backup um 02:00; ein Incident um 23:00 verliert den ganzen Tag |
| **RTO** (max. Wiederherstellungszeit) | ~30-60 min | Velero Restore Namespace + PVC + Helm-Reconcile; abhängig von PVC-Größe |

> **Fachliche Konsequenz bei Restore aus dem letzten daily Backup**: Mitglieder, die zwischen letztem Backup (02:00 UTC) und dem Incident submitted/genehmigt wurden, müssen neu registrieren bzw. neu genehmigt werden. Die Mitgliedsnummern, die zwischenzeitlich vergeben wurden, sind weg → Lückenlosigkeit der Nummerierung ist nicht garantiert. Vorgehen: in den Velero-Logs nach `velero_backup_last_successful_timestamp` schauen, das ist die exakte Cut-off-Zeit.

### 1.3 Restore-Verfahren

#### a) Namespace-Restore (häufigster Fall: DB korrupt, Pod will nicht starten, jemand hat `kubectl delete ns` aus Versehen)

```bash
# 1. Sicherstellen, dass der Namespace nicht mehr da ist (oder leer):
kubectl get ns member-onboarding

# 2. Letztes erfolgreiches Daily Backup finden:
velero backup get | grep daily | head -5

# 3. Restore starten (ohne Service-Restart auf andere Namespaces):
velero restore create restore-$(date +%Y%m%d-%H%M) \
  --from-backup daily-backup-YYYY-MM-DD-HHMMSS \
  --include-namespaces member-onboarding \
  --restore-volumes=true

# 4. Status verfolgen:
velero restore describe restore-YYYYMMDD-HHMM --details

# 5. Nach Restore: Helm-Reconcile (Velero restored die Manifests stateless, aber
#    die Helm-Release-Metadaten müssen wieder mit dem Cluster synchronisiert sein):
helm list -n member-onboarding   # sollte den Release zeigen
```

#### b) Nur PostgreSQL-PVC wiederherstellen (App-Pods sind ok, DB ist defekt)

```bash
# 1. Backend skalieren auf 0 — sonst schreiben Pods in die noch-alte DB
kubectl scale -n member-onboarding deploy/member-onboarding-backend --replicas=0

# 2. PVC löschen (Postgres-StatefulSet hat ein einziges Volume `data`)
kubectl delete pvc -n member-onboarding data-member-onboarding-postgres-0

# 3. Velero Restore *nur* der PVC aus dem gewünschten Backup
velero restore create restore-pvc-$(date +%Y%m%d-%H%M) \
  --from-backup daily-backup-YYYY-MM-DD-HHMMSS \
  --include-namespaces member-onboarding \
  --include-resources persistentvolumeclaims,persistentvolumes

# 4. Postgres-Pod neu starten (StatefulSet rebindet automatisch an die restaurierte PVC)
kubectl delete pod -n member-onboarding member-onboarding-postgres-0

# 5. Warten bis pg_isready (Readiness-Probe im StatefulSet), dann Backend wieder hoch:
kubectl scale -n member-onboarding deploy/member-onboarding-backend --replicas=1
```

#### c) Vollständiger Cluster-Verlust

Siehe Cluster-DR-Doku — Velero auf neuem Cluster installieren, gleichen Wasabi-Bucket verbinden, Restore wie unter (a). Restore-Tests sind cluster-seitig durchgeführt.

### 1.4 Post-Restore-Checks (mandatory)

Nach jedem Restore in dieser Reihenfolge:

| # | Check | Erwartet |
|---|---|---|
| 1 | `kubectl get pods -n member-onboarding` | Alle Pods `Running`, `Ready 1/1` |
| 2 | `kubectl exec -n member-onboarding member-onboarding-postgres-0 -- pg_isready` | `accepting connections` |
| 3 | `kubectl exec ... psql -U postgres -d member_onboarding -c "SELECT COUNT(*) FROM member_onboarding.application;"` | Plausible Anzahl (mit Pre-Incident vergleichen) |
| 4 | Browser: Admin-Login auf `https://member-onboarding-test.eegfaktura.at` (oder Prod) | Login funktioniert, Antragsliste sichtbar |
| 5 | Browser: Detail eines beliebigen Antrags öffnen | Daten + Status-Log vollständig |
| 6 | Velero: `velero_backup_last_successful_timestamp` aus Grafana checken | Backup-Schedule läuft weiter |
| 7 | Mitgliedsnummer-Kollision prüfen | siehe 1.5 |

### 1.5 Mitgliedsnummer-Lückenlosigkeit nach Restore

Mitgliedsnummern (`application.member_number`) werden bei Submit vergeben und sind tenant-eindeutig (kein UNIQUE Constraint — siehe Open Issues). Nach einem Restore aus dem 24h-alten Backup:

- Nummern bis zur Backup-Zeit sind im Restore enthalten
- Zwischenzeitlich vergebene Nummern aus dem verlorenen Tag sind weg
- Neue Registrierungen ab Restore-Zeit nutzen wieder den höchsten gespeicherten Wert + 1 → keine Kollision, aber **Lücke in der Nummerierung** ist möglich

Wenn der EEG fachlich eine lückenlose Nummerierung erwartet: Fall mit EEG-Admin besprechen, ggf. die Sequenz manuell anpassen (`SELECT MAX(member_number) FROM member_onboarding.application WHERE rc_number = '…'`).

---

## 2. Häufige Incident-Szenarien

### 2.1 eegFaktura-Core nicht erreichbar (Import schlägt fehl)

**Symptom**: Admin klickt „In eegFaktura importieren", bekommt Fehler-Toast. Log zeigt 5xx/Timeout vom Core.

**Was passiert**: Status der Application bleibt `approved` oder wechselt zu `import_failed` (je nach Phase). Daten in Onboarding-DB sind intakt; nichts wurde verloren.

**Sofort-Maßnahme**: keine. Admins können später erneut auf „Import erneut versuchen" klicken (`import_failed → approved → imported`). PROJ-30 (Reset-Import) ist verfügbar, falls im Core ein halber Stand entstanden ist.

**Warten oder eskalieren**:
- &lt; 1h: warten, Core-Team informieren falls noch nicht bekannt
- &gt; 1h: EEG-Admins per Mail/Slack informieren, dass Imports verzögert sind
- Mehrere Stunden: Core-Team eskalieren, ggf. manuelle Anlage im Core erwägen

### 2.2 Postal-SMTP-Server unerreichbar

**Symptom**: Mitglieder erhalten keine Bestätigungsmails. Backend-Log zeigt `mail: failed to send member confirmation`.

**Was passiert**: Antrags-Submit ist trotzdem erfolgreich (Mails werden fire-and-forget abgesetzt). **Eingehende Mails ab Postal-Recovery werden NICHT nachgeholt** — sie sind verloren (siehe Open Issue „Mail-Outbox" im Architektur-Review).

**Sofort-Maßnahme**: 
- Postal-Status checken: `https://atvipostal.vfeeg.org`
- Falls Mails kritisch (laufende Beitritts-Werbung): EEG-Admin informieren, dass für betroffene Mitglieder manuell die „Bestätigung erneut senden"-Aktion im Admin-Web aufgerufen werden muss (PROJ-X — `resendMemberConfirmation`).

### 2.3 Hohe Last bei Antragsspitze (z.B. nach EEG-Marketing-Push)

**Symptom**: Public-Form zeigt 429 oder ist langsam. Admins beklagen Latenz.

**Was passiert**: 
- Rate-Limit auf Public-Endpoint: 10 req / 10 min / IP (in-process, per Pod)
- Backend ist single-replica → keine horizontale Skalierung möglich

**Sofort-Maßnahme**: Pod-Ressourcen prüfen (`kubectl top pod -n member-onboarding`); ggf. Limits anheben. Längerfristig: siehe Architektur-Review (Multi-Replica + DB-Backed Rate-Limit).

### 2.4 Velero-Alert „Daily Backup zu alt" feuert

**Symptom**: Grafana-Alert via Postal-Mail.

**Was tun**: 
1. `velero backup get | head` — wann lief der letzte erfolgreiche?
2. `velero backup describe daily-backup-LATEST` — Fehler-Details
3. `kubectl logs -n velero deploy/velero` — Velero-Server-Log
4. Häufige Ursachen: Wasabi-Bucket voll, Credentials abgelaufen, CSI-Snapshot-Class nicht verfügbar

**Während der Alert offen ist**: Keine destruktiven Operationen (DB-Migrationen, große Bulk-Imports) durchführen — kein aktuelles Backup zum Zurückrollen.

---

## 3. Deployment

Es gibt **keinen automatischen `helm upgrade`** in CI. Die GitHub-Actions-Pipeline:

1. Baut Docker-Images
2. Pusht sie in die Registry
3. Committet `helm/member-onboarding/values.yaml` mit der neuen Image-SHA als chore-Commit `chore: update Helm image tags to sha-XXXXX [skip ci]`

Der eigentliche Cluster-Sync erfolgt **manuell** durch den Operator:

```bash
# Update auf neueste Chart-Version (siehe Chart.yaml im Repo)
cd helm/
helm upgrade member-onboarding ./member-onboarding \
  -n member-onboarding \
  -f values-env.yaml \
  --atomic \
  --timeout 5m
```

**Migration**: läuft automatisch als pre-upgrade Hook-Job (siehe `templates/migrate-job.yaml`). Helm wartet bis zu 5 min auf den Job-Erfolg.

**Rollback**:
```bash
helm rollback member-onboarding <REVISION> -n member-onboarding
```

> **Wichtig**: `helm rollback` rollt **keine DB-Migrationen zurück**. Wenn die migrierte Schema-Version mit dem alten Image inkompatibel ist, muss die DB-Migration manuell zurückgerollt werden (`db/migrations/000NNN_*.down.sql`). Vor schema-relevanten Releases: aktuelles Backup verifizieren.

### Hängengebliebener Import (PROJ-34)

Wenn ein Import-Vorgang abbricht (Pod-Crash, DB-UNIQUE-Verletzung, …) und die Application im `approved`-Status mit gesetztem `import_started_at` ohne `import_finished_at` zurücklässt, bietet das Admin-UI seit PROJ-34 zwei Recovery-Aktionen direkt im Antrags-Detail:

- **„Als importiert markieren"** — Admin gibt die Teilnehmer-UUID + Mitgliedsnummer aus eegFaktura ein, Antrag wechselt sauber auf `imported`.
- **„Import-Lock räumen (Retry)"** — Lock weg, Status bleibt `approved`. Risiko: bei erneutem Import-Klick entsteht im Core ein Duplikat, falls der vorige Versuch dort schon eingefügt hat.

Die Buttons erscheinen automatisch, wenn der Server-side-Check `import_started_at > NOW() - 2 min AND import_finished_at IS NULL` zutrifft.

Für SQL-Diagnose (vor Eingriff via UI):
```sql
SELECT id, reference_number, status, import_started_at, import_finished_at,
       imported_at, target_participant_id, import_error_message
FROM member_onboarding.application
WHERE import_started_at IS NOT NULL
  AND import_finished_at IS NULL
  AND status = 'approved';
```

---

### Recovery: `Dirty database version N` (Migration abgebrochen)

Wenn der Migrate-Job mit `migrate up failed: Dirty database version N. Fix and force version.` abbricht, hat eine Migration mittendrin gescheitert (typisch: UNIQUE-Constraint, NOT-NULL-Backfill, Typ-Verengung) und golang-migrate hat `schema_migrations.dirty = true` gesetzt. `cmd/migrate` hat keinen `force`-Modus — das Flag wird per SQL zurückgesetzt.

**1. psql öffnen:**
```bash
NS=eegfaktura-member-onboarding         # bzw. -test
kubectl -n $NS exec -it member-onboarding-postgres-0 -c postgres -- \
  psql -U postgres -d member_onboarding
```

**2. Migration-Stand sichten** — gibt `version=N, dirty=t`:
```sql
SELECT * FROM schema_migrations;
```

**3. Prüfen, ob Migration N inhaltlich durchgegangen ist.** Die `.up.sql` der Migration anschauen, das wichtigste Artefakt finden (Spalte, Index, Constraint, Funktion …) und in der DB nachsehen, z.B. für einen Index:
```sql
SELECT indexname FROM pg_indexes
 WHERE schemaname='member_onboarding'
   AND indexname='<expected_index>';
```
- **Zeile zurück** → Migration ist durch, nur das Flag hängt → weiter mit Schritt 5a.
- **Leer** → Migration ist halb abgebrochen → Daten in Schritt 4 fixen.

**4. Daten aufräumen** (Migration-spezifisch). Beispiel UNIQUE-Verletzung — Duplikate finden:
```sql
SELECT <key_cols>, COUNT(*) FROM <schema>.<table>
 WHERE <new_unique_col> IS NOT NULL
 GROUP BY <key_cols>
HAVING COUNT(*) > 1;
```
Strategie zur Auflösung hängt vom Domänenmodell ab — Status-Felder berücksichtigen, ggf. ältere/neuere Duplikate nullen oder mergen. Bei produktiven Daten vorher mit dem Owner abklären.

**5. Dirty-Flag löschen** — eine der beiden Varianten:
```sql
-- 5a: Migration N war durch, nur Flag hängt:
UPDATE schema_migrations SET dirty = false WHERE version = N;

-- 5b: Migration N läuft beim nächsten Up-Run neu (nach Schritt 4):
UPDATE schema_migrations SET dirty = false, version = N-1;
```
*(`schema_migrations` liegt standardmäßig im `public`-Schema, nicht im `member_onboarding`. Vorher `\dt *.schema_migrations` prüfen.)*

**6. Migrate-Job neu erzeugen** — der pre-upgrade-Hook regeneriert ihn beim nächsten Helm-Upgrade:
```bash
kubectl -n $NS delete job <release>-migrate
helm upgrade <release> ./helm/member-onboarding -n $NS -f ...
kubectl -n $NS logs -l job-name=<release>-migrate -c migrate
```
Erwartet: `Migrations applied successfully` und `SELECT version, dirty FROM schema_migrations;` zeigt die neueste Version mit `dirty=f`.

---

## 4. Monitoring & Alerts

### Prometheus-Metrics (ab Chart 1.9.0)

Backend exponiert `/metrics` auf Port `9090` (separater HTTP-Server, **nicht** über den Public-Ingress). Scrape-fähig direkt aus dem Cluster (ClusterIP-Service `member-onboarding-backend-metrics`).

| Counter | Bedeutung | Beispiel-Alert |
|---|---|---|
| `eegfaktura_mo_applications_submitted_total` | Eingehende Public-Form-Submits | Plötzlicher Drop → Public-Form kaputt? |
| `eegfaktura_mo_imports_total{result}` | Imports zum Core, `success` vs `failed` | `rate(...{result="failed"}[5m]) > 0.1` |
| `eegfaktura_mo_mail_sent_total{kind,result}` | Mails pro Template+Result | `rate(...{result="failed"}[15m]) > 0` |
| `eegfaktura_mo_rate_limit_hits_total` | Public-Submit Rate-Limit-Denials | Plötzlicher Anstieg → Scraper |
| `eegfaktura_mo_member_number_lookups_total{result}` | next-member-number-Lookup-Result | `core_error`-Spikes → Core langsam |
| `eegfaktura_mo_http_request_duration_seconds` | HTTP-Latenz-Histogramm | P95/P99 nach Status-Klasse |

Plus die standard Prometheus-Collectors: `go_*` (GC, Goroutines, Memory) + `process_*` (CPU, RSS, FDs).

**Aktivierung der prometheus-operator-Integration**: in `values-env.yaml`:
```yaml
metrics:
  serviceMonitor:
    enabled: true
    labels:
      release: rancher-monitoring   # je nach Stack
```

### Sonstige Beobachtung

| Was | Wo | Aktion bei Alert |
|---|---|---|
| Velero-Backup-Alerts | Grafana-Folder `Velero` | siehe 2.4 |
| Pod-Restart-Loops | `kubectl get pods -n member-onboarding -w` | Logs prüfen, ggf. Helm-Rollback |
| HTTP 5xx-Rate | Prometheus + Backend-Logs | siehe Backend-Logs |
| Import-Failures | `eegfaktura_mo_imports_total{result="failed"}` | Alert ab >0/min für 5 min |

> Strukturierte Logs des Backends:
> ```bash
> kubectl logs -n member-onboarding deploy/member-onboarding-backend -f \
>   | jq 'select(.level == "ERROR" or .level == "WARN")'
> ```

---

## 5. Bekannte Einschränkungen (zur Erwartungssetzung)

Aus dem Architektur-Review 2026-05-13 — diese Punkte sind dem Team bekannt und in der Backlog-Pipeline:

- **`replicas: 1`** ist heute hart — kein HA während Rollouts. Multi-Replica-Vorarbeiten (DB-backed Rate-Limit, Mail-Outbox) sind PROJ-Items, noch nicht umgesetzt.
- **Mail-Versand** ist fire-and-forget; bei Pod-Restart während SMTP-Send kann eine Mail verloren gehen. Wird mit Mail-Outbox geschlossen.
- **Keine NetworkPolicies** — Frontend-Pod könnte direkt auf Postgres zugreifen (architektonisch unerwünscht, technisch möglich).
- **Keine Prometheus-Metrics** vom App-Service selbst — nur Logs. Counter wie `import_failed_total` sind heute nur durch Log-Aggregation sichtbar.

Vollständige Liste: siehe Architektur-Review-Bericht.
