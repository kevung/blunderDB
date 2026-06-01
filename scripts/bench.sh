#!/usr/bin/env bash
# bench.sh — run the headless-engine benchmarks (P9).
#
#   scripts/bench.sh micro          # Go microbenchmarks, SQLite only (no Docker)
#   scripts/bench.sh micro-pg       # Go microbenchmarks, Postgres (needs Docker)
#   scripts/bench.sh loadtest       # end-to-end load test against a running daemon
#
# Microbenchmark results print to stdout; see tasks/headless/perf-baseline.md
# for the recorded baseline. The loadtest sub-command needs a daemon already
# listening (start one with `blunderdb serve ...`).
set -euo pipefail
cd "$(dirname "$0")/.."

BENCHTIME="${BENCHTIME:-1s}"

case "${1:-micro}" in
  micro)
    echo "# SQLite microbenchmarks (benchtime=$BENCHTIME)"
    go test -bench . -benchmem -benchtime "$BENCHTIME" -run '^$' \
      ./pkg/blunderdb/storage/sqlite/
    ;;
  micro-pg)
    echo "# Postgres microbenchmarks (benchtime=$BENCHTIME, needs Docker)"
    go test -tags postgres -bench . -benchmem -benchtime "$BENCHTIME" -run '^$' \
      ./pkg/blunderdb/storage/postgres/
    ;;
  loadtest)
    shift
    TARGET="${TARGET:-http://localhost:8080}"
    SCENARIO="${SCENARIO:-mixed}"
    TENANTS="${TENANTS:-200}"
    CONCURRENCY="${CONCURRENCY:-100}"
    DURATION="${DURATION:-30s}"
    OUTPUT="${OUTPUT:-report.json}"
    echo "# load test: $SCENARIO @ $TARGET (conc=$CONCURRENCY, ${DURATION})"
    go run ./cmd/blunderdb-loadtest \
      --target "$TARGET" --scenario "$SCENARIO" --tenants "$TENANTS" \
      --concurrency "$CONCURRENCY" --duration "$DURATION" --output "$OUTPUT" "$@"
    ;;
  *)
    echo "usage: $0 {micro|micro-pg|loadtest}" >&2
    exit 2
    ;;
esac
