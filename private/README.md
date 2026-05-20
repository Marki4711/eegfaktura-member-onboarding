# `private/` — Staging-Verzeichnis für PROJ-54

Dieses Verzeichnis ist die Vorbereitung für die Aufteilung in
**öffentliches Schaufenster + privates Hauptrepo** (siehe
`features/PROJ-54-public-private-repo-split.md`).

## Wozu liegt das hier im noch-öffentlichen Repo?

Bis zum Cutover-Tag liegen die fertigen Mirror-Workflows, Hooks und
Filter-Skripte hier. Sie laufen **nicht** in diesem Repo, weil
`.github/workflows/` nur GitHub-Action-YAMLs in seinem direkten
Inhalt aktiviert — die Datei `private/workflows/mirror-to-public.yml`
ist GitHub egal.

So sind die Tools fertig, wenn der Cutover startet — und gleichzeitig
ist der Plan transparent für jeden, der das Repo gerade anschaut.

## Was passiert beim Cutover

Schritt-für-Schritt-Checkliste siehe `CUTOVER.md`.

Im Kern werden die Dateien beim Cutover an ihre Ziel-Positionen verschoben:

| Quelle (heute) | Ziel (im privaten Repo nach Cutover) |
|---|---|
| `private/workflows/mirror-to-public.yml` | `.github/workflows/mirror-to-public.yml` |
| `private/githooks/pre-commit` | `.githooks/pre-commit` |
| `private/githooks/pre-push` | `.githooks/pre-push` |
| `private/scripts/strip-private-frontmatter.sh` | `.github/scripts/strip-private-frontmatter.sh` |
| `private/scripts/apply-mirror-filter.sh` | `.github/scripts/apply-mirror-filter.sh` |
| `private/mirror-whitelist.txt` | `.github/mirror-whitelist.txt` |

Nach dem Cutover bleibt `private/` als Sammelplatz für die
**eigentlich** sensiblen Inhalte (Pricing, Verträge, Pen-Test, DPIA),
die der Mirror-Filter zuverlässig ausschließt.

## Struktur

```
private/
├── README.md                            (du bist hier)
├── CUTOVER.md                           Cutover-Checkliste
├── mirror-whitelist.txt                 Pfad-Whitelist für den Mirror-Filter
├── workflows/
│   └── mirror-to-public.yml             GitHub-Action für privates Repo
├── githooks/
│   ├── pre-commit                       defensive Schicht: blockt Sensibles
│   └── pre-push                         dito beim Push
└── scripts/
    ├── apply-mirror-filter.sh           Hauptskript: Whitelist + Frontmatter
    └── strip-private-frontmatter.sh     entfernt Dateien mit `visibility: private`
```

## Was kommt später noch hinzu (nach Cutover, NICHT in Public)

```
private/
├── pricing/      Tarif-Modelle, Kalkulationen
├── contracts/    Vertrags-Templates (AVV, EEG-Verträge)
├── dpia/         DSGVO-Folgenabschätzung + VVT
├── pentest/      Pen-Test-Reports + Fix-Trail
├── postmortems/  Incident-Berichte
├── runbooks/     Operationelle Runbooks mit Anbieter-Daten
├── vendor-setup/ PSP-Setup-Anleitungen mit echten Daten
└── billing/      eigenes Rechnungsmodul (sobald gebaut)
```
