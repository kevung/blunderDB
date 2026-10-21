#!/usr/bin/env bash
# bench-sweep.sh — times the read surface of blunderDB (CLI commands, and the
# handlers the GUI calls most, through `call`) plus three isolated writes, on a
# COPY of the given database, and prints a Markdown table sorted by duration.
#
#   scripts/bench-sweep.sh <base.db> [binary]
#
# - The base is copied first: opening a database writes derived caches, and the
#   original must stay untouched. The first open of the copy is timed apart.
# - Each operation runs once; under 20 s it runs twice more and the median is
#   kept (with the RSS of that median run). Peak RSS comes from wait4's
#   ru_maxrss (python3), since /usr/bin/time is not installed everywhere.
# - Writes (import, analyze, export) each run once on their own fresh copy.
# - IMPORT_DIR (default bench/import50) holds the match files of the batch import.
# - SWEEP_ONLY=<regex> runs only the operations whose label matches (the first
#   open and the first statistics read always run): to re-measure a few.
# - Work files go under bench/ (gitignored), never /tmp: the copies are large.
#
# Output: Markdown on stdout; the raw TSV and every command's stderr stay in the
# work directory printed at the end.
set -u

# The whole body is one function, parsed before it runs: editing the script
# during a sweep cannot derail the sweep in progress.
main() {
ROOT=$(cd "$(dirname "$0")/.." && pwd)
SRC=${1:?usage: scripts/bench-sweep.sh <base.db> [binary]}
BIN=${2:-$ROOT/bin/blunderdb}
IMPORT_DIR=${IMPORT_DIR:-$ROOT/bench/import50}
REPEAT_UNDER=${REPEAT_UNDER:-20}

if [[ ! -x $BIN ]]; then
	echo "building $BIN" >&2
	mkdir -p "$(dirname "$BIN")"
	(cd "$ROOT" && go build -o "$BIN" .) || exit 1
fi
SRC=$(realpath "$SRC")
BIN=$(realpath "$BIN")

mkdir -p "$ROOT/bench"
WORK=$(mktemp -d "$ROOT/bench/sweep.XXXXXX")
DB=$WORK/base.db
cp "$SRC" "$DB"
for s in -wal -shm; do [[ -f $SRC$s ]] && cp "$SRC$s" "$DB$s"; done
RAW=$WORK/raw.tsv
: >"$RAW"

# run1 <outfile> <cmd...> → "seconds maxrss_kb exit_code" on stdout.
run1() {
	python3 -I -c '
import os, sys, time, resource
out = open(sys.argv[1], "wb")
err = open(sys.argv[1] + ".err", "ab")
t0 = time.perf_counter()
pid = os.fork()
if pid == 0:
    os.dup2(out.fileno(), 1); os.dup2(err.fileno(), 2)
    os.execvp(sys.argv[2], sys.argv[2:])
_, status, ru = os.wait4(pid, 0)
dt = time.perf_counter() - t0
print("%.3f %d %d" % (dt, ru.ru_maxrss, os.waitstatus_to_exitcode(status)))
' "$@"
}

N=0
# measure <label> <cmd...>: once, then twice more when under REPEAT_UNDER s.
measure() {
	local label=$1
	shift
	if [[ -n ${SWEEP_ONLY:-} && ! $label =~ $SWEEP_ONLY && $label != *"(first"* ]]; then return 0; fi
	N=$((N + 1))
	local out=$WORK/out.$N
	local r1 r2 r3 runs=1
	r1=$(run1 "$out" "$@")
	local lines bytes
	lines=$(wc -l <"$out")
	bytes=$(wc -c <"$out")
	local t1=${r1%% *}
	if awk -v t="$t1" -v l="$REPEAT_UNDER" 'BEGIN{exit !(t < l)}'; then
		r2=$(run1 "$out.2" "$@")
		r3=$(run1 "$out.3" "$@")
		rm -f "$out.2" "$out.3"
		runs=3
		r1=$(printf '%s\n%s\n%s\n' "$r1" "$r2" "$r3" | sort -n | sed -n 2p)
	fi
	read -r secs rss rc <<<"$r1"
	printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$label" "$secs" "$rss" "$rc" "$lines" "$bytes" "$runs" >>"$RAW"
	printf '%-60s %8ss rc=%s lines=%s\n' "$label" "$secs" "$rc" "$lines" >&2
	# Keep a little of the output for inspection, not hundreds of megabytes.
	if ((bytes > 65536)); then head -c 65536 "$out" >"$out.head" && rm -f "$out"; fi
}

# measure_once: writes, a single run on a fresh copy of the base.
measure_once() {
	local label=$1
	shift
	if [[ -n ${SWEEP_ONLY:-} && ! $label =~ $SWEEP_ONLY && $label != *"(first"* ]]; then return 0; fi
	N=$((N + 1))
	local out=$WORK/out.$N r
	r=$(run1 "$out" "$@")
	read -r secs rss rc <<<"$r"
	printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$label" "$secs" "$rss" "$rc" "$(wc -l <"$out")" "$(wc -c <"$out")" 1 >>"$RAW"
	printf '%-60s %8ss rc=%s\n' "$label" "$secs" "$rc" >&2
}

sq() { sqlite3 "$DB" "$1"; }

LOAD_BEFORE=$(cat /proc/loadavg)
START=$(date +%s)

# --- first open: migrations and derived caches written on the copy ---------
measure_once "info (first open of the copy, derived caches)" "$BIN" info --db "$DB"

# The match statistics cache (match_stats, match_stats_cell) is filled lazily by
# the first statistics read; it is timed here, apart, so that every statistics
# operation below measures the steady state.
measure_once "call stats.compute (first, fills the match stats cache)" "$BIN" call stats.compute --db "$DB" --json '{"filter":{"DecisionType":-1}}'

# --- parameters drawn from the base ------------------------------------------
PLAYER=$(sq "SELECT n FROM (SELECT player1_name n FROM match UNION ALL SELECT player2_name FROM match) WHERE n <> '' GROUP BY n ORDER BY COUNT(*) DESC LIMIT 1")
OPPONENT=$(sq "SELECT CASE WHEN player1_name = '${PLAYER//\'/\'\'}' THEN player2_name ELSE player1_name END o FROM match WHERE '${PLAYER//\'/\'\'}' IN (player1_name, player2_name) GROUP BY o ORDER BY COUNT(*) DESC LIMIT 1")
TID=$(sq "SELECT tournament_id FROM match WHERE tournament_id IS NOT NULL GROUP BY 1 ORDER BY COUNT(*) DESC LIMIT 1")
MID=$(sq "SELECT g.match_id FROM game g JOIN move m ON m.game_id = g.id GROUP BY g.match_id ORDER BY COUNT(*) DESC LIMIT 1")
PID=$(sq "SELECT a.position_id FROM analysis a JOIN move m ON m.position_id = a.position_id ORDER BY a.position_id LIMIT 1 OFFSET 1000")
PAGE_IDS=$(sq "SELECT group_concat(id) FROM (SELECT id FROM match ORDER BY match_date DESC, id DESC LIMIT 500)")
XGID=$("$BIN" call positions.load --db "$DB" --json "{\"id\":$PID}" 2>/dev/null | jq -r '.xgid // empty' 2>/dev/null)
[[ -z $XGID ]] && XGID='XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:7:10'
BEAROFF_XGID='XGID=-BBBB----------------bbbb-:0:0:1:00:0:0:0:0:10'
echo "player=$PLAYER opponent=$OPPONENT tournament=$TID match=$MID position=$PID" >&2

# --- untimed setup on the copy: a collection, a deck, a lesson to read -------
"$BIN" collection create --db "$DB" --name "Sweep blunders" >/dev/null 2>&1
CID=$(sq "SELECT MAX(id) FROM collection")
BLUNDERS=$("$BIN" call search.ids --db "$DB" --json '{"filters":{"moveErrorFilter":"E>100"},"limit":500}' 2>/dev/null)
"$BIN" call collections.addPositions --db "$DB" --json "{\"collectionId\":$CID,\"positionIds\":$BLUNDERS}" >/dev/null 2>&1
"$BIN" call anki.createDeck --db "$DB" --json "{\"name\":\"Sweep deck\",\"sourceType\":\"collection\",\"sourceId\":$CID}" >/dev/null 2>&1
DECK=$(sq "SELECT MAX(id) FROM anki_deck")
"$BIN" anki sync --db "$DB" --deck "$DECK" >/dev/null 2>&1
"$BIN" lesson create --db "$DB" --name "Sweep lesson" >/dev/null 2>&1
LID=$(sq "SELECT MAX(id) FROM lesson")
"$BIN" lesson add-step --db "$DB" --lesson "$LID" --title Blunders --text x --collection "$CID" >/dev/null 2>&1
"$BIN" call history.save --db "$DB" --json '{"command":"s cube E>100"}' >/dev/null 2>&1

B=$BIN
# --- CLI: list ---------------------------------------------------------------
measure "list --type matches --limit 0" $B list --db "$DB" --type matches --limit 0
measure "list --type matches --query <player> --limit 0" $B list --db "$DB" --type matches --query "$PLAYER" --limit 0
measure "list --type tournaments --limit 0" $B list --db "$DB" --type tournaments --limit 0
measure "list --type positions --limit 100" $B list --db "$DB" --type positions --limit 100
measure "list --type positions --format csv --limit 0" $B list --db "$DB" --type positions --format csv --limit 0
measure "list --type moves --format csv --limit 0" $B list --db "$DB" --type moves --format csv --limit 0
measure "list --type analyses --format csv --limit 0" $B list --db "$DB" --type analyses --format csv --limit 0
measure "list --type imports" $B list --db "$DB" --type imports
measure "list --type stats" $B list --db "$DB" --type stats
measure "list --type stats --metric mwc --player <p>" $B list --db "$DB" --type stats --metric mwc --player "$PLAYER"
measure "list --type players --limit 0" $B list --db "$DB" --type players --limit 0
measure "list --type timeerrors" $B list --db "$DB" --type timeerrors
measure "list --type tags" $B list --db "$DB" --type tags
measure "list --type study --limit 0" $B list --db "$DB" --type study --limit 0

# --- CLI: search ---------------------------------------------------------------
measure "search --decision checker --limit 0 --format json (wide)" $B search --db "$DB" --decision checker --limit 0 --format json
measure "search --decision cube --limit 50" $B search --db "$DB" --decision cube --limit 50
measure "search --query 'E>100' --limit 0" $B search --db "$DB" --query 'E>100' --limit 0
measure "search --dice 6,5 --score1 2 --score2 4 --limit 0" $B search --db "$DB" --dice 6,5 --score1 2 --score2 4 --limit 0
measure "search --player <p> --error-min 0.05 --limit 0" $B search --db "$DB" --player "$PLAYER" --error-min 0.05 --limit 0
measure "search --decision cube --pip-min 80 --pip-max 120 --limit 0 --format xgid" $B search --db "$DB" --decision cube --pip-min 80 --pip-max 120 --limit 0 --format xgid
measure "search --game-type race --limit 0 --format xgid" $B search --db "$DB" --game-type race --limit 0 --format xgid

# --- CLI: stats -----------------------------------------------------------------
for s in ranking recurring plan effect biases training windows tournaments progression breakdown engines; do
	if [[ $s == ranking || $s == engines ]]; then
		measure "stats $s" $B stats $s --db "$DB"
	else
		measure "stats $s --player <p>" $B stats $s --db "$DB" --player "$PLAYER"
	fi
done
measure "stats contrast --player <p> --opponent <o>" $B stats contrast --db "$DB" --player "$PLAYER" --opponent "$OPPONENT"
measure "stats tournament --id <t> --player <p>" $B stats tournament --db "$DB" --id "$TID" --player "$PLAYER"
measure "stats report --html --player <p>" $B stats report --db "$DB" --html --player "$PLAYER" --output "$WORK/report.html"

# --- CLI: the rest of the read surface ----------------------------------------
measure "info" $B info --db "$DB"
measure "info --estimate" $B info --db "$DB" --estimate
measure "verify" $B verify --db "$DB"
measure "match --id <m> --format json" $B match --db "$DB" --id "$MID" --format json
measure "players alias list" $B players alias list --db "$DB"
measure "players alias suggest" $B players alias suggest --db "$DB"
measure "events alias list" $B events alias list --db "$DB"
measure "events alias suggest" $B events alias suggest --db "$DB"
measure "tournament list" $B tournament list --db "$DB"
measure "collection list" $B collection list --db "$DB"
measure "collection show --id <c> (500)" $B collection show --db "$DB" --id "$CID"
measure "collection suggest --player <p>" $B collection suggest --db "$DB" --player "$PLAYER" --size 10
measure "lesson list" $B lesson list --db "$DB"
measure "lesson show --id <l>" $B lesson show --db "$DB" --id "$LID"
measure "anki decks" $B anki decks --db "$DB"
measure "anki stats --deck <d>" $B anki stats --db "$DB" --deck "$DECK"
measure "anki forecast --deck <d> --days 30" $B anki forecast --db "$DB" --deck "$DECK" --days 30
measure "anki retention --deck <d>" $B anki retention --db "$DB" --deck "$DECK"
measure "anki export --deck <d> (.apkg)" $B anki export --db "$DB" --deck "$DECK" --out "$WORK/deck.apkg"
measure "training sessions" $B training sessions --db "$DB"
measure "training missed" $B training missed --db "$DB"
measure "study queue --limit 0" $B study queue --db "$DB" --limit 0
measure "openingbook status" $B openingbook status
measure "openingbook question --roll first" $B openingbook question --roll first
measure "cubematrix <xgid> (2-ply)" $B cubematrix "$XGID"
measure "epc <bearoff xgid>" $B epc "$BEAROFF_XGID"
measure "bearoff list" $B bearoff list
measure "maintenance status" $B maintenance status --db "$DB"

# --- call: the handlers the GUI calls most -------------------------------------
c() { local label=$1 m=$2 j=$3; measure "call $label" $B call "$m" --db "$DB" --json "$j"; }
c "metadata.countsEstimate (status bar)" metadata.countsEstimate '{}'
c "search.count E>100 (status bar blunders)" search.count '{"filters":{"moveErrorFilter":"E>100"}}'
c "matches.list (all, NDJSON)" matches.list '{}'
c "stats.matchBadges (page of 500)" stats.matchBadges "{\"matchIds\":[$PAGE_IDS]}"
c "tournaments.list (all)" tournaments.list '{}'
c "stats.tournamentBadges" stats.tournamentBadges '{}'
c "tournaments.matches <t>" tournaments.matches "{\"tournamentId\":$TID}"
c "positions.listIds (all)" positions.listIds '{"limit":0}'
c "positions.list page 100 @1000" positions.list '{"limit":100,"offset":1000}'
c "positions.load <p>" positions.load "{\"id\":$PID}"
c "analyses.load <p>" analyses.load "{\"positionId\":$PID}"
c "matches.byPosition <p> (provenance)" matches.byPosition "{\"positionId\":$PID}"
c "matches.origin <m>" matches.origin "{\"matchId\":$MID}"
c "matches.movesByMatch <m>" matches.movesByMatch "{\"matchId\":$MID}"
c "stats.compute (dashboard, no filter)" stats.compute '{"filter":{"DecisionType":-1}}'
c "stats.compute (dashboard, player)" stats.compute "{\"filter\":{\"DecisionType\":-1,\"PlayerName\":$(jq -Rn --arg p "$PLAYER" '$p')}}"
c "stats.playerTable" stats.playerTable '{"filter":{"DecisionType":-1}}'
c "stats.ranking" stats.ranking '{"filter":{"DecisionType":-1}}'
c "stats.recurringErrors" stats.recurringErrors '{"filter":{"DecisionType":-1}}'
c "stats.playerNames" stats.playerNames '{}'
c "search.ids E>100 (GUI search)" search.ids '{"filters":{"moveErrorFilter":"E>100"}}'
c "search.ids cube (GUI search)" search.ids '{"filters":{"decisionTypeFilter":true,"filter":{"decision_type":1}}}'
c "search.find cube limit 50" search.find '{"filters":{"decisionTypeFilter":true,"filter":{"decision_type":1}},"limit":50}'
c "search.query 's cube E>100'" search.query '{"query":"s cube E>100"}'
c "collections.list" collections.list '{}'
c "collections.positions <c> page 50" collections.positions "{\"collectionId\":$CID,\"limit\":50}"
c "anki.listDecks" anki.listDecks '{}'
c "anki.nextCard <d>" anki.nextCard "{\"deckId\":$DECK}"
c "lessons.list" lessons.list '{}'
c "lessons.get <l>" lessons.get "{\"id\":$LID}"
c "history.load" history.load '{}'

# --- writes, each on its own fresh copy -----------------------------------------
fresh() { rm -f "$WORK/w.db" "$WORK/w.db-wal" "$WORK/w.db-shm" && cp "$DB" "$WORK/w.db"; }
fresh
measure_once "export --type database (whole base)" $B export --db "$WORK/w.db" --type database --file "$WORK/export.db"
EXPORT_BYTES=$(stat -c %s "$WORK/export.db" 2>/dev/null || echo 0)
rm -f "$WORK/export.db"*

fresh
AMID=$(sqlite3 "$WORK/w.db" "SELECT g.match_id FROM game g JOIN move m ON m.game_id = g.id JOIN analysis a ON a.position_id = m.position_id GROUP BY g.match_id ORDER BY COUNT(DISTINCT m.position_id) DESC, g.match_id LIMIT 1")
# 20 positions of one match lose their analysis: `analyze --match` fills exactly that gap.
sqlite3 "$WORK/w.db" "DELETE FROM analysis WHERE position_id IN (SELECT DISTINCT m.position_id FROM move m JOIN game g ON g.id = m.game_id WHERE g.match_id = $AMID AND m.position_id IS NOT NULL ORDER BY m.position_id LIMIT 20)"
measure_once "analyze --match <m> (20 positions, 2-ply)" $B analyze --db "$WORK/w.db" --match "$AMID"

if [[ -d $IMPORT_DIR ]]; then
	fresh
	NFILES=$(find "$IMPORT_DIR" -type f | wc -l)
	# A sample drawn from the corpus the base was built from is all duplicates:
	# the matches it already holds under the same file name (an "NNNNN-" prefix
	# of the sample stripped) are deleted from the copy first, untimed, so the
	# import writes them anew.
	local f n del=0
	while IFS= read -r f; do
		n=${f##*/}
		[[ $n =~ ^[0-9]+-(.*)$ ]] && n=${BASH_REMATCH[1]}
		n=${n//\'/\'\'}
		for id in $(sqlite3 "$WORK/w.db" "SELECT id FROM match WHERE file_path LIKE '%/$n' OR file_path = '$n'"); do
			"$B" delete --db "$WORK/w.db" --type match --id "$id" --confirm >/dev/null 2>&1 && del=$((del + 1))
		done
	done < <(find "$IMPORT_DIR" -type f)
	echo "deleted $del matches of the import sample from the copy" >&2
	measure_once "import --type batch ($NFILES files)" $B import --db "$WORK/w.db" --type batch --dir "$IMPORT_DIR"
fi
rm -f "$WORK/w.db"*

# --- report ------------------------------------------------------------------------
LOAD_AFTER=$(cat /proc/loadavg)
END=$(date +%s)
{
	echo "Base: \`$SRC\` ($(du -h "$SRC" | cut -f1)), binary \`$BIN\`, $(nproc) CPUs"
	echo
	echo "Load (/proc/loadavg): before \`$LOAD_BEFORE\`, after \`$LOAD_AFTER\`; sweep $(((END - START) / 60)) min $(((END - START) % 60)) s"
	echo
	echo "Parameters: player \`$PLAYER\`, opponent \`$OPPONENT\`, tournament $TID, match $MID, position $PID; export file $((EXPORT_BYTES / 1048576)) MiB"
	echo
	echo "| # | Operation | Duration (s) | Output lines | Per line / per call | Peak RSS (MiB) | Runs | Exit |"
	echo "|---:|---|---:|---:|---:|---:|---:|---:|"
	sort -t$'\t' -k2,2 -g -r "$RAW" | awk -F'\t' '{
		per = ($5 > 1) ? sprintf("%.1f µs/line", $2 * 1e6 / $5) : sprintf("%.0f ms/call", $2 * 1000)
		printf "| %d | `%s` | %.2f | %d | %s | %.0f | %d | %s |\n", NR, $1, $2, $5, per, $3 / 1024, $7, $4
	}'
	echo
	echo "Work directory (raw TSV, stderr of each command): \`$WORK\`"
}
}

main "$@"; exit $?
