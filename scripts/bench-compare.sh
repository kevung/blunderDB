#!/usr/bin/env bash
#
# bench-compare.sh — benchstat a fresh benchmark run against the versioned
# baseline (E.9, #225) and fail on a sec/op regression past a threshold.
#
# Usage:
#   scripts/bench-compare.sh <baseline.txt> <new.txt> [threshold_pct] [summary_md]
#
#   threshold_pct   percent slowdown that counts as a regression (default 10,
#                   matching the fiche's "seuil ±10%"). Only sec/op is gated —
#                   B/op and allocs/op drift is reported but not blocking, the
#                   same way `-race`'s five-fold memory tax is tolerated
#                   elsewhere in this repo's CI (see build.yml's `gammonnet`
#                   shard comment): allocation counts are useful context, not
#                   the signal this gate exists for.
#   summary_md      optional path to append a Markdown table to (step summary
#                   or PR comment body). Always written to stdout too.
#
# Needs `benchstat` on PATH (golang.org/x/perf/cmd/benchstat).
#
# Design note: both files are almost always -count=1 (a single sample per
# benchmark — see tasks/bench/baseline.txt's own generation command), so
# benchstat can never reach the >=4 samples it wants to call an individual
# benchmark's delta significant — every per-benchmark row prints "~" and only
# the per-table `geomean` row still reports a plain percentage (benchstat
# computes that unconditionally, no significance test attached). Gating on
# geomean is therefore not a simplification of a richer per-benchmark gate,
# it is the only signal -count=1 produces; a real per-benchmark regression
# still moves its table's geomean, just diluted by however many other
# benchmarks share that table. This mirrors the project's tolerance for
# coarse-but-useful signals elsewhere (CLAUDE.md's rolling-stats non-total
# ORDER BY) over an unreachable, lab-grade one.
set -euo pipefail

baseline="${1:?usage: bench-compare.sh <baseline.txt> <new.txt> [threshold_pct] [summary_md]}"
new="${2:?usage: bench-compare.sh <baseline.txt> <new.txt> [threshold_pct] [summary_md]}"
threshold="${3:-10}"
summary_md="${4:-}"

if ! command -v benchstat >/dev/null 2>&1; then
  echo "::error::benchstat not found on PATH — install with: go install golang.org/x/perf/cmd/benchstat@<version>" >&2
  exit 2
fi

csv="$(mktemp)"
trap 'rm -f "$csv"' EXIT

# Warnings ("need >= 6 samples...") go to stderr with -format csv; only the
# table itself is wanted here.
benchstat -format csv "$baseline" "$new" 2>/dev/null > "$csv"

{
  echo '| Paquet | Mesure | geomean baseline → nouveau |'
  echo '|---|---|---|'
} > "${summary_md:-/dev/null}" 2>/dev/null || true
if [ -n "$summary_md" ]; then
  {
    echo '| Paquet | Mesure | geomean baseline → nouveau |'
    echo '|---|---|---|'
  } > "$summary_md"
fi

regressions=0
table_rows=""

# awk state machine: track the current package ("pkg: X" lines, only emitted
# when it changes) and the current unit (the header row ",<unit>,CI,<unit>,CI,vs base,P").
# Both persist across the blank-line-separated tables benchstat prints one per
# measurement (sec/op, B/op, allocs/op, and any per-benchmark custom unit like
# ns/valuation or ns/position).
while IFS= read -r line; do
  case "$line" in
    "pkg: "*)
      pkg="${line#pkg: }"
      ;;
    ,*,CI,*,CI,"vs base",P)
      unit="$(echo "$line" | cut -d, -f2)"
      ;;
    geomean,*)
      delta="$(echo "$line" | awk -F, '{print $NF}')"
      pkg_label="${pkg:-(racine)}"
      table_rows="${table_rows}| \`${pkg_label}\` | ${unit:-?} | ${delta} |
"
      if [ "${unit:-}" = "sec/op" ]; then
        case "$delta" in
          [+-]*%)
            pct="${delta%\%}"
            pct="${pct#+}"
            # Only a positive delta (new is slower) beyond the threshold is a
            # regression; a negative one is an improvement worth keeping quiet.
            is_regression=$(awk -v p="$pct" -v t="$threshold" 'BEGIN{print (p > t) ? 1 : 0}')
            if [ "$is_regression" = "1" ]; then
              echo "::error::régression perf: ${pkg_label} sec/op geomean ${delta} (seuil ${threshold}%)" >&2
              regressions=$((regressions + 1))
            fi
            ;;
        esac
      fi
      ;;
  esac
done < "$csv"

if [ -n "$summary_md" ]; then
  printf '%s' "$table_rows" >> "$summary_md"
  {
    echo
    if [ "$regressions" -gt 0 ]; then
      echo "**${regressions} régression(s) sec/op au-delà de ±${threshold}%.**"
    else
      echo "Aucune régression sec/op au-delà de ±${threshold}%."
    fi
  } >> "$summary_md"
fi

echo "--- geomean par paquet/mesure ---"
printf '%s' "$table_rows"
echo "seuil: ±${threshold}% — régressions détectées: ${regressions}"

if [ "$regressions" -gt 0 ]; then
  exit 1
fi
