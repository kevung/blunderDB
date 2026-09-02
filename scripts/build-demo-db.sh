#!/usr/bin/env bash
# Rebuild the sample database the GUI embeds (internal/gui/demo.db.gz).
#
# Run it from anywhere in the repository after every schema bump, so the
# embedded copy carries the current database_version instead of being
# migrated at each `demo` command. The result is reproducible up to
# timestamps: same sources, same rankings, same seeded review journal.
#
#   scripts/build-demo-db.sh            # writes internal/gui/demo.db.gz
#   DEMO_PLY=1 scripts/build-demo-db.sh # quick smoke run (non-canonical analyses)
#
# Content (issue #162): three matches from testdata/ under fictional player,
# event and tournament names — an XG match (eXtreme Gammon analysis), a
# BGBlitz game and a .mat transcript with no analysis, which the gammonNet
# sweep below fills — plus three collections, ten tagged comments and an
# Anki deck with a simulated review journal (see scripts/demodb/main.go).
# The summary at the end reopens the file, so the compaction that leaves it
# in rollback-journal mode runs before it — keep that order.
# The analyses are written at the canonical 2-ply so the demo shows exactly
# what the GUI would compute; the sweep takes about half a minute.
#
# Requires: go, gzip. The sources named here and in scripts/demodb/main.go
# must match — the helper refuses a match it has no fictional identity for.
set -euo pipefail

ROOT=$(git -C "$(dirname "$0")" rev-parse --show-toplevel)
OUT="$ROOT/internal/gui/demo.db.gz"
MAX_GZ_BYTES=$((700 * 1024))
PLY=${DEMO_PLY:-2}

SOURCES=(
    "$ROOT/testdata/HsbtMarseille_main_ronde4_LamourDeCaslouGildas_UngerKevin_7p.xg"
    "$ROOT/testdata/test.mat"
    "$ROOT/testdata/TachiAI_V_player_Nov_2__2025__16_55.bgf"
)

# Names the embedded file must not contain: the people the fixtures name.
# internal/gui/demo_test.go carries the same list.
FORBIDDEN='unger|harmand|friebe|jacobi|huyck|larsen|caslou|maxence|tachi'

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT
DB="$WORK/demo.db"
CLI="$WORK/blunderdb"

# main.go embeds frontend/dist; a checkout that never built the frontend
# needs the placeholder CLAUDE.md describes (the directory is gitignored).
if [ ! -e "$ROOT/frontend/dist/index.html" ]; then
    mkdir -p "$ROOT/frontend/dist" && touch "$ROOT/frontend/dist/index.html"
fi

echo "== building the CLI"
(cd "$ROOT" && go build -o "$CLI" .)

echo "== creating $DB"
"$CLI" create --db "$DB" --user "blunderDB" \
    --description "Base de démonstration : joueurs, tournoi et lieu fictifs." >/dev/null

for src in "${SOURCES[@]}"; do
    echo "== importing $(basename "$src")"
    "$CLI" import --db "$DB" --type match --file "$src" >/dev/null
done

echo "== gammonNet sweep ($PLY-ply) over the positions without analysis"
"$CLI" analyze --db "$DB" --ply "$PLY" | tail -n 1

echo "== disguising, collections, comments, Anki deck and journal"
(cd "$ROOT" && go run ./scripts/demodb -db "$DB")

echo "== checking that no real name survived"
if { "$CLI" list --db "$DB" --type matches; "$CLI" list --db "$DB" --type tournaments; "$CLI" info --db "$DB"; } \
    | grep -v '^Connected to database\|^Path:' | grep -iE "$FORBIDDEN"; then
    echo "error: a real name survived in the demo database" >&2
    exit 1
fi

echo "== summary"
"$CLI" info --db "$DB" | sed -n '/^Statistics/,$p'
"$CLI" collection list --db "$DB"
"$CLI" anki decks --db "$DB"

echo "== compacting"
(cd "$ROOT" && go run ./scripts/demodb -db "$DB" -only-compact)

echo "== compressing"
gzip -9 -n -c "$DB" > "$OUT"
size=$(wc -c < "$OUT" | tr -d " ")
echo "   $OUT: $size bytes"
if [ "$size" -gt "$MAX_GZ_BYTES" ]; then
    echo "error: demo.db.gz exceeds $MAX_GZ_BYTES bytes" >&2
    exit 1
fi

