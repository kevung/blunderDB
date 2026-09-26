# Race win chances are read or convolved; cube verdicts are never estimated

Status: accepted.
See also: ADR-0012 (the three regimes), ADR-0027 (where the tables come from).

## Context
A gnubg two-sided (TS) bearoff table gives, per position pair, four money equities for the
player on roll (cubeless, cube owned, centred, against; `u/32767.5 − 1`, gammonless). The
one-sided (OS) table gives each side's roll distribution. Two measurements decide what may be
shown without a TS answer:
- convolving the two one-sided distributions, plus a 32-term correction calibrated on
  TS-06-11, estimates the win probability to σ 0.05 %, p99 0.15 %, max 0.42 %;
- no static model estimates cubeful equity (best: 45 dof, RMS 0.016, max 0.20), and the
  dead-cube `p → MET` chain agrees with gnubg on D/ND only 87.8 % of the time.

## Decision
1. **Money game is the referential** of the exact table: the panel shows money equities and
   the money cube verdict, all four planes kept. The table's verdict is not converted to a
   match score through the MET.
2. **One reader, widest domain wins.** A single gnubg `.bd` two-sided reader (`race.TwoSided`,
   `race.Resolve`) serves every source; invalid sources are skipped with a log warning.
   Lookups are 8-byte `ReadAt`s; the file is never loaded into memory. Sources: ADR-0027
   rules 3 and 4.
3. **Outside the table's domain, the win probability is estimated** (convolution + frozen
   calibrated correction) and shown with its error bound. EPC is always exact (the OS table
   covers all 15 checkers).
4. **A cube verdict is never estimated.** No summary statistic, calibrated Janowski model or
   MET chain produces a verdict. (A verdict from a search that plays the trajectory is a
   different method — ADR-0012.)
5. **Nothing is persisted.** No table data is written to user databases; `DatabaseVersion` is
   untouched. The 1.23 GB TS-06-11 oracle is never committed; calibration and tests read it
   via `BLUNDERDB_TS11_PATH` and skip when absent.
6. **Parity.** The logic lives in `pkg/blunderdb/engine/race`; the server EPC handler, the CLI
   `epc` command and `call` expose the same fields.

## Consequences
- The user documentation states every hypothesis behind a displayed number: independence of
  the two race processes, one-sided optimal play, correction calibrated on ≤ 11 checkers and
  extrapolated beyond, money gammonless referential, no verdict outside the exact domain.
- Rejected: compressing the TS data — lossless 1.1–1.3×, no smoothness between neighbouring
  indices.
- Rejected: a plane-0-only file — the cubeful planes are the product.
- Rejected: MET conversion of verdicts — wrong in 12 % of decisions, worse than none.
- Rejected: an estimated money verdict — σ 0.016 equity straddles the decision frontier.
- Deferred: extending the exact domain to no-contact races beyond the home board — no oracle
  certifies an error bound.

## Guard
`pkg/blunderdb/engine/race/convolve_test.go`, `convolve_beyond_test.go`, `epc_parity_test.go`.
