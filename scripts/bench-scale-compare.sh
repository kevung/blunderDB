#!/usr/bin/env bash
#
# bench-scale-compare.sh — compare two runs of the scale benchmarks
# (BenchmarkScale_*, pkg/blunderdb/database) benchmark by benchmark, and fail
# when any one got slower than <factor> times its previous time.
#
#   scripts/bench-scale-compare.sh <previous.txt> <new.txt> [factor] [summary_md] [floor_ms]
#
# A benchmark listed several times (-count N) counts by its best run: the
# first pass of a fresh process reads a cold base, and the minimum is the
# figure that noise moves least.
#
# Unlike bench-compare.sh this gates each benchmark, not a geomean: the scale
# benchmarks are few and measure distinct paths (one import, a search, the
# stats), so a regression on one must not hide behind the others. Samples
# are few and runners noisy, so the factor is wide (1.5 by default):
# it catches an index lost or a query turned into a full scan, not drift.
# A benchmark present on one side only is reported, never gated.
#
# A slowdown must also exceed an absolute floor (1 ms by default). Runner noise
# is additive, a preempted thread or a collection costing the same whatever
# the benchmark measures: on an operation of a few tenths of a millisecond,
# timed once per pass, it alone crosses the factor on unchanged code. Nightly
# best-of-three figures on one CPU model differ by up to 0.22 ms. The floor does
# not relax the factor elsewhere: the next shortest benchmark takes about
# 40 ms, where the factor already asks for 20 ms, and a full scan of the base,
# over a second on the wide searches, still fails. A ratio excused by the floor
# stays visible in the table.
#
# Both runs must come from the same CPU model (the `cpu:` line go test
# prints). Hosted runners are drawn from several processor generations whose
# speeds differ by more than the factor itself, so a pair of different CPUs
# measures the hardware, not the code: such a pair is reported and never
# gated. The nightly job keys its reference by CPU model, so this only fires
# when a caller hands it two unrelated files.
set -euo pipefail

prev="${1:?usage: bench-scale-compare.sh <previous.txt> <new.txt> [factor] [summary_md] [floor_ms]}"
new="${2:?usage: bench-scale-compare.sh <previous.txt> <new.txt> [factor] [summary_md] [floor_ms]}"
factor="${3:-1.5}"
summary_md="${4:-}"
floor_ms="${5:-1}"

cpu_of() { grep -m1 '^cpu:' "$1" | sed 's/^cpu:[[:space:]]*//; s/[[:space:]]*$//' || true; }
prev_cpu=$(cpu_of "$prev")
new_cpu=$(cpu_of "$new")
if [ "$prev_cpu" != "$new_cpu" ]; then
  msg="bench-scale-compare: previous run on '${prev_cpu:-unknown CPU}', new run on '${new_cpu:-unknown CPU}' — different hardware, not compared."
  echo "$msg"
  if [ -n "$summary_md" ]; then echo "$msg" >>"$summary_md"; fi
  exit 0
fi

table=$(awk -v factor="$factor" -v floor_ms="$floor_ms" '
  function name(s) { sub(/-[0-9]+$/, "", s); return s }
  FNR == 1 { file++ }
  /^BenchmarkScale_/ {
    for (i = 2; i <= NF; i++) if ($(i+1) == "ns/op") { t = $i; break }
    k = name($1)
    if (file == 1) { if (!(k in old) || t < old[k]) old[k] = t }
    else           { if (!(k in cur) || t < cur[k]) cur[k] = t }
  }
  END {
    print "| Benchmark | previous (ms) | new (ms) | ratio |"
    print "|---|---:|---:|---:|"
    bad = 0
    for (k in cur) {
      if (!(k in old)) { printf "| %s | — | %.1f | new |\n", k, cur[k]/1e6; continue }
      r = cur[k] / old[k]
      flag = ""
      if (r > factor) {
        if (cur[k] - old[k] > floor_ms * 1e6) { flag = " **regression**"; bad++ }
        else flag = " (under the " floor_ms " ms floor)"
      }
      printf "| %s | %.1f | %.1f | %.2f%s |\n", k, old[k]/1e6, cur[k]/1e6, r, flag
    }
    for (k in old) if (!(k in cur)) printf "| %s | %.1f | — | gone |\n", k, old[k]/1e6
    print ""
    printf "REGRESSIONS=%d\n", bad
  }' "$prev" "$new")

echo "$table"
if [ -n "$summary_md" ]; then echo "$table" >>"$summary_md"; fi
if ! echo "$table" | grep -q '^REGRESSIONS=0$'; then
  echo "bench-scale-compare: at least one benchmark slower than ×$factor and by more than $floor_ms ms" >&2
  exit 1
fi
