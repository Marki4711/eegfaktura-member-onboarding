#!/usr/bin/env bash
#
# apply-mirror-filter.sh — wendet die Mirror-Whitelist auf einen
# Verzeichnisbaum an. Wird vom Mirror-Workflow im privaten Repo aufgerufen.
#
# Aufruf:
#   apply-mirror-filter.sh <source-dir> <target-dir> <whitelist-file>
#
# Ablauf:
#   1. Source-Verzeichnis komplett nach Target kopieren
#   2. Alle Dateien löschen, die KEINEM Whitelist-Pattern entsprechen
#   3. Frontmatter-Filter aufrufen (entfernt visibility:private-Markierungen)
#
# Exit-Codes:
#   0 — Erfolg
#   1 — Argumente fehlen
#   2 — Source/Target/Whitelist nicht gefunden
#   3 — Filter-Operation fehlgeschlagen

set -euo pipefail

if [[ $# -ne 3 ]]; then
    echo "Usage: $0 <source-dir> <target-dir> <whitelist-file>" >&2
    exit 1
fi

SOURCE_DIR="$1"
TARGET_DIR="$2"
WHITELIST_FILE="$3"

[[ -d "$SOURCE_DIR" ]] || { echo "Source not found: $SOURCE_DIR" >&2; exit 2; }
[[ -f "$WHITELIST_FILE" ]] || { echo "Whitelist not found: $WHITELIST_FILE" >&2; exit 2; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "→ Filtering $SOURCE_DIR → $TARGET_DIR"
echo "  Whitelist: $WHITELIST_FILE"

# Schritt 1: Full-Copy (rsync mit --delete, damit Target sauber ist)
mkdir -p "$TARGET_DIR"
rsync -a --delete \
    --exclude='.git/' \
    "$SOURCE_DIR/" "$TARGET_DIR/"

# Schritt 2: Whitelist anwenden — alles löschen, was nicht matched
# Implementation: Liste aller aktuellen Dateien holen, gegen Patterns prüfen.
# Pattern-Matching via bash-extglob; das vermeidet eine externe Abhängigkeit.

# Whitelist-Patterns einlesen (ohne Kommentare/leere Zeilen)
PATTERNS=()
while IFS= read -r line; do
    # Trim, skip comments/empty
    line="${line#"${line%%[![:space:]]*}"}"  # ltrim
    line="${line%"${line##*[![:space:]]}"}"  # rtrim
    [[ -z "$line" || "$line" == \#* ]] && continue
    PATTERNS+=("$line")
done < "$WHITELIST_FILE"

if [[ ${#PATTERNS[@]} -eq 0 ]]; then
    echo "ERROR: Whitelist is empty — aborting (refuse to publish everything)" >&2
    exit 3
fi

echo "  ${#PATTERNS[@]} whitelist patterns loaded"

# Funktion: prüft ob $1 mindestens einem Pattern entspricht
shopt -s globstar nullglob extglob

path_matches_any() {
    local path="$1"
    local pattern
    for pattern in "${PATTERNS[@]}"; do
        # bash glob match (mit globstar für **)
        # shellcheck disable=SC2053
        [[ "$path" == $pattern ]] && return 0
    done
    return 1
}

# Alle Dateien im Target auflisten, gegen Whitelist prüfen, ggf. löschen
removed_count=0
kept_count=0

while IFS= read -r -d '' file; do
    # Relativen Pfad zum Target ermitteln
    rel="${file#$TARGET_DIR/}"
    if path_matches_any "$rel"; then
        ((kept_count++))
    else
        rm -f "$file"
        ((removed_count++))
    fi
done < <(find "$TARGET_DIR" -type f -print0)

# Leere Verzeichnisse aufräumen
find "$TARGET_DIR" -type d -empty -delete 2>/dev/null || true

echo "  Files kept:    $kept_count"
echo "  Files removed: $removed_count"

# Schritt 3: Frontmatter-Filter aufrufen
echo "→ Stripping files with 'visibility: private' frontmatter"
bash "$SCRIPT_DIR/strip-private-frontmatter.sh" "$TARGET_DIR"

# Schritt 4: Sanity-Check
if [[ ! -f "$TARGET_DIR/README.md" ]]; then
    echo "ERROR: README.md missing from filtered output — whitelist may be broken" >&2
    exit 3
fi
if [[ ! -d "$TARGET_DIR/internal" ]]; then
    echo "ERROR: internal/ missing from filtered output — whitelist may be broken" >&2
    exit 3
fi
if [[ -d "$TARGET_DIR/private" ]]; then
    echo "ERROR: private/ leaked into filtered output — whitelist broken" >&2
    exit 3
fi

echo "✓ Filter complete: $TARGET_DIR"
