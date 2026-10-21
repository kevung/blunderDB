#!/usr/bin/env bash
# bench-daemon-load.sh — load test of `blunderdb serve` on a COPY of a real
# library: N concurrent clients (LEVELS) run the "library" scenario of
# cmd/blunderdb-loadtest (positions.load, analyses.load, search.find,
# search.count, matches.list, stats.compute, comments.add, analyses.save) and
# the script records, per level, the loadtest report (latency per operation,
# throughput, errors by code), the daemon's peak RSS, and a goroutine dump
# taken mid-run (where the requests wait: connection pool, SQLite lock, CPU).
#
#   scripts/bench-daemon-load.sh <base.db>
#
# - The base is copied first and never opened in place. A first daemon open
#   migrates the copy (timed); that migrated copy is the "cold" template, whose
#   match_stats rows may be missing. One stats.compute then fills them (timed)
#   and gives the "warm" template. Every level runs on its own fresh copy of a
#   template, so the writes of one level do not weigh on the next.
# - LEVELS (default "10 50 200"), DURATION (default 30s) per level.
# - COLD_N (default 50, 0 to skip): one more run of that many clients on the
#   cold template — concurrent readers filling match_stats at once.
# - PG_DSN (optional): a PostgreSQL DSN whose database the script may write;
#   the warm template is copied into it under tenant 1 with `blunderdb migrate`
#   (the full binary, built from the root package) and the levels run again
#   on the postgres backend.
# - Work files go under bench/ (gitignored), never /tmp: the copies are large.
#   Do not export GOTMPDIR for a measuring run: it changes SQLite's write path.
#
# Output: a Markdown summary on stdout; reports, goroutine dumps, allocation
# profiles (go tool pprof -sample_index=alloc_space) and daemon logs stay in
# the work directory printed at the end.
set -u

main() {
ROOT=$(cd "$(dirname "$0")/.." && pwd)
SRC=$(realpath "${1:?usage: scripts/bench-daemon-load.sh <base.db>}")
LEVELS=${LEVELS:-10 50 200}
DURATION=${DURATION:-30s}
COLD_N=${COLD_N:-50}
PORT=${PORT:-18093}
PPROF_PORT=$((PORT + 1))
URL=http://127.0.0.1:$PORT

mkdir -p "$ROOT/bench"
WORK=$(mktemp -d "$ROOT/bench/daemon-load.XXXXXX")
echo "work directory: $WORK" >&2
(cd "$ROOT" && go build -o "$WORK/serve" ./cmd/serve && go build -o "$WORK/loadtest" ./cmd/blunderdb-loadtest) || exit 1

copy_db() { # copy_db <from> <to>
	cp "$1" "$2"
	for s in -wal -shm; do [[ -f $1$s ]] && cp "$1$s" "$2$s"; done
	return 0
}

PID=
start_daemon() { # start_daemon <log> <serve args...> → sets PID, waits for /healthz
	local log=$1; shift
	"$WORK/serve" "$@" --addr "127.0.0.1:$PORT" --pprof-addr "127.0.0.1:$PPROF_PORT" \
		--rate-limit-rps 0 --log-level warn >"$log" 2>&1 &
	PID=$!
	local i
	for i in $(seq 1 3000); do
		curl -sf "$URL/healthz" >/dev/null 2>&1 && return 0
		kill -0 "$PID" 2>/dev/null || { echo "daemon died, see $log" >&2; return 1; }
		sleep 0.1
	done
	echo "daemon not ready after 300 s, see $log" >&2
	return 1
}

stop_daemon() {
	[[ -n $PID ]] || return 0
	kill -INT "$PID" 2>/dev/null
	wait "$PID" 2>/dev/null
	PID=
}

# sample_rss <pid> <out>: "VmRSS RssAnon RssFile" in kB every 0.5 s while the
# process lives. RssFile is the base's mmap (mmap_size), shared with the page
# cache; RssAnon is the Go heap plus SQLite's per-connection cache_size and
# temp_store=MEMORY b-trees.
sample_rss() {
	while kill -0 "$1" 2>/dev/null; do
		awk '/^VmRSS/ {v=$2} /^RssAnon/ {a=$2} /^RssFile/ {f=$2} END {if (v) print v, a, f}' "/proc/$1/status" 2>/dev/null
		sleep 0.5
	done >"$2"
}

secs() { python3 -I -c 'import sys,time; print(f"{time.time():.3f}")'; }

# run_level <label> <clients> <db args...>: one loadtest run against a fresh daemon.
SUMMARY=$WORK/summary.tsv
: >"$SUMMARY"
run_level() {
	local label=$1 n=$2; shift 2
	local dir=$WORK/$label
	mkdir -p "$dir"
	local t0; t0=$(secs)
	start_daemon "$dir/serve.log" "$@" || return 1
	local t1; t1=$(secs)
	sample_rss "$PID" "$dir/rss.txt" &
	local sampler=$!
	local half
	half=$(python3 -I -c "import sys; d=sys.argv[1]; print(float(d.rstrip('s'))/2 + 4)" "$DURATION")
	( sleep "$half"; curl -s "http://127.0.0.1:$PPROF_PORT/debug/pprof/goroutine?debug=1" >"$dir/goroutines.txt" ) &
	local dumper=$!
	"$WORK/loadtest" --target "$URL" --tenants 1 --scenario library --concurrency "$n" \
		--duration "$DURATION" --ramp-up 2s --output "$dir/report.json" >"$dir/loadtest.out" 2>&1
	wait "$dumper"
	curl -s "$URL/metrics" >"$dir/metrics.txt"
	curl -s "http://127.0.0.1:$PPROF_PORT/debug/pprof/allocs" >"$dir/allocs.pprof"
	stop_daemon
	wait "$sampler" 2>/dev/null
	local peak; peak=$(sort -n "$dir/rss.txt" | tail -1)
	read -r v a f <<<"${peak:-0 0 0}"
	printf '%s\t%s\t%.1f\t%s\n' "$label" "$n" "$(python3 -I -c "print($t1-$t0)")" "$((v / 1024)) ($((a / 1024)) anon, $((f / 1024)) file)" >>"$SUMMARY"
}

# --- templates ---------------------------------------------------------------
COLD=$WORK/cold.db
WARM=$WORK/warm.db
copy_db "$SRC" "$COLD"
t0=$(secs)
start_daemon "$WORK/open.log" --db "$COLD" || exit 1
t1=$(secs)
stop_daemon
copy_db "$COLD" "$WARM"
start_daemon "$WORK/fill.log" --db "$WARM" || exit 1
t2=$(secs)
curl -s -o /dev/null -X POST "$URL/v1/stats.compute" -H 'X-Tenant-ID: 1' \
	-H 'Content-Type: application/json' -d '{"filter":{"DecisionType":-1}}'
t3=$(secs)
stop_daemon
FIRST_OPEN=$(python3 -I -c "print(f'{$t1-$t0:.1f}')")
FIRST_FILL=$(python3 -I -c "print(f'{$t3-$t2:.1f}')")

# --- SQLite levels -----------------------------------------------------------
for n in $LEVELS; do
	copy_db "$WARM" "$WORK/run.db"
	run_level "sqlite-$n" "$n" --db "$WORK/run.db"
	rm -f "$WORK/run.db" "$WORK/run.db-wal" "$WORK/run.db-shm"
done
if [[ $COLD_N -gt 0 ]]; then
	copy_db "$COLD" "$WORK/run.db"
	run_level "sqlite-cold-$COLD_N" "$COLD_N" --db "$WORK/run.db"
	rm -f "$WORK/run.db" "$WORK/run.db-wal" "$WORK/run.db-shm"
fi

# --- PostgreSQL levels (optional) --------------------------------------------
if [[ -n ${PG_DSN:-} ]]; then
	t0=$(secs)
	if (cd "$ROOT" && go build -o "$WORK/blunderdb" .) &&
		"$WORK/blunderdb" migrate --from "sqlite://$WARM" --to "$PG_DSN" --tenant-id 1 >"$WORK/pg-migrate.log" 2>&1; then
		t1=$(secs)
		PG_MIGRATE=$(python3 -I -c "print(f'{$t1-$t0:.1f}')")
		for n in $LEVELS; do
			run_level "postgres-$n" "$n" --backend postgres --dsn "$PG_DSN"
		done
	else
		echo "migrate to PostgreSQL failed, see $WORK/pg-migrate.log; the postgres levels are skipped" >&2
	fi
fi

# --- summary -----------------------------------------------------------------
echo "## Conditions"
echo
echo "- base: \`$SRC\` (copied), duration per level $DURATION, loadavg $(cut -d' ' -f1-3 /proc/loadavg)"
echo "- first daemon open of the copy (migration): ${FIRST_OPEN} s; first stats.compute (fills match_stats): ${FIRST_FILL} s"
[[ -n ${PG_MIGRATE:-} ]] && echo "- migrate to PostgreSQL (tenant 1): ${PG_MIGRATE} s"
echo
echo "| Run | Clients | Daemon start (s) | Peak RSS, MiB (anon, file) | Requests | RPS | Errors | p50 (ms) | p95 (ms) | p99 (ms) |"
echo "|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|"
while IFS=$'\t' read -r label n start rss; do
	python3 -I - "$WORK/$label/report.json" "$label" "$n" "$start" "$rss" <<'PY'
import json, sys
p, label, n, start, rss = sys.argv[1:]
try:
    r = json.load(open(p))
except OSError:
    print(f"| {label} | {n} | {start} | {rss} | (no report) |||||| "); sys.exit()
l = r["latency_ms"]
print(f"| {label} | {n} | {start} | {rss} | {r['total_requests']} | {r['rps_achieved']:.0f} | {r['errors']} | {l['p50']:.1f} | {l['p95']:.1f} | {l['p99']:.1f} |")
PY
done <"$SUMMARY"
echo
while IFS=$'\t' read -r label _ _ _; do
	echo "### $label"
	echo
	sed -n '/^| Endpoint/,$p' "$WORK/$label/report.md"
	echo
	echo "Goroutines mid-run, by where they wait (top frames of blunderdb or database/sql):"
	echo
	python3 -I - "$WORK/$label/goroutines.txt" <<'PY'
import re, sys
txt = open(sys.argv[1]).read() if sys.argv[1] else ""
groups = re.split(r"\n\n", txt)
tally = {}
for g in groups:
    m = re.match(r"(\d+) @", g)
    if not m:
        continue
    frames = re.findall(r"#\s+0x[0-9a-f]+\s+(\S+)", g)
    if frames and "awaitDone" in frames[0]:
        continue  # database/sql's per-query watchers, not a request
    where = next((x.split("/")[-1] for x in frames if "/pkg/blunderdb/" in x), "")
    key = None
    for f in frames:
        if f.startswith("database/sql.(*DB).conn"):
            key = "database/sql: waiting for a pooled connection in " + where; break
        if "sqlite3_busy" in f or "busyHandler" in f or "sqlite3BtreeBeginTrans" in f:
            key = "sqlite: waiting for the write lock"; break
        if "sync.(*RWMutex)" in f or "sync.(*Mutex).Lock" in f:
            key = "sync mutex: " + next((x for x in frames if "blunderdb" in x), f); break
        if "modernc.org/sqlite" in f:
            key = "sqlite: running a statement in " + where; break
        if "net/http.(*conn).serve" in f or "net/http.(*persistConn)" in f:
            key = "net/http: idle connection"; break
    key = key or (frames[0] if frames else "?")
    tally[key] = tally.get(key, 0) + int(m.group(1))
for k, v in sorted(tally.items(), key=lambda kv: -kv[1])[:8]:
    print(f"- {v} × {k}")
PY
	echo
done <"$SUMMARY"
echo "work directory: $WORK"
}
main "$@"
