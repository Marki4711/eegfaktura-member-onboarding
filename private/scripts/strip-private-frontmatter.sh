#!/usr/bin/env bash
#
# strip-private-frontmatter.sh — entfernt Markdown-Dateien, deren YAML-
# Frontmatter `visibility: private` enthält. Wird vom Mirror-Filter im
# zweiten Schritt aufgerufen.
#
# Aufruf:
#   strip-private-frontmatter.sh <root-dir>
#
# Erkennt das Frontmatter konservativ: nur YAML-Frontmatter am Datei-Anfang
# (erste Zeile = `---`, dann YAML, dann `---`).

set -euo pipefail

if [[ $# -ne 1 ]]; then
    echo "Usage: $0 <root-dir>" >&2
    exit 1
fi

ROOT_DIR="$1"
[[ -d "$ROOT_DIR" ]] || { echo "Root not found: $ROOT_DIR" >&2; exit 2; }

removed_count=0

# Alle .md-Dateien rekursiv durchgehen
while IFS= read -r -d '' file; do
    # Datei muss mit `---` beginnen
    first_line="$(head -n 1 "$file" 2>/dev/null || echo "")"
    [[ "$first_line" == "---" ]] || continue

    # Frontmatter-Block extrahieren (zwischen den beiden ---)
    # awk: print bis zur zweiten ---, dann stop
    frontmatter="$(awk '
        NR == 1 && /^---$/ { in_fm = 1; next }
        in_fm && /^---$/   { in_fm = 0; exit }
        in_fm              { print }
    ' "$file")"

    # `visibility:` mit Wert `private` (whitespace-tolerant) suchen
    if echo "$frontmatter" | grep -qE '^[[:space:]]*visibility:[[:space:]]*private[[:space:]]*$'; then
        echo "  removing (visibility: private): ${file#$ROOT_DIR/}"
        rm -f "$file"
        ((removed_count++))
    fi
done < <(find "$ROOT_DIR" -type f -name "*.md" -print0)

echo "  Frontmatter-stripped: $removed_count files"

# Leere Verzeichnisse aufräumen
find "$ROOT_DIR" -type d -empty -delete 2>/dev/null || true
