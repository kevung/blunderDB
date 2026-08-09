# Race win chances are read or convolved; cube verdicts are never estimated

## Status

accepted — design decided 2026-08-09, implementation tracked in `tasks/ts-bearoff/`

## Context

The EPC panel computes Effective Pip Counts from the embedded one-sided bearoff
database (`engine/gnubg_os6.bd`, 1.4 MB, OS-06-15). It says nothing about win
chances or the cube. A two-sided database answers both — gnubg's TS format
stores, per position pair, four uint16 equities for the player on roll
(cubeless, cube owned, cube centered, cube against; scale `u/32767.5 − 1`,
money game, gammonless). The question was how to offer that in blunderDB when
a TS-06-11 file weighs 1.23 GB against a 30 MB binary.

Measurements taken before deciding (session of 2026-08-09, scripts and raw
numbers in the task sheet):

- **Lossless compression is dead.** xz over raw data: 1.13×. Plane splitting,
  pip-ordering, inter-plane deltas, varint: 1.26× at best. Neighbouring
  combinatorial indices are not neighbouring positions; there is no smoothness
  to exploit.
- **Coverage does not reward size.** Against a real user database (88 015
  positions), a 6.8 MB TS-06-06 covers 16.6 % of pure-bearoff positions; the
  1.23 GB TS-06-11 covers 55.4 %. 180× the bytes for 3.3× the coverage, and
  no embeddable point on the curve is complete.
- **Win probability estimates superbly without the TS data.** Convolving the
  two one-sided roll distributions (already embedded) gives p with raw error
  σ = 0.12 % of win probability; the residual bias is antisymmetric around
  p = 0.5 (the trailer plays for variance and recovers equity) and a 32-term
  polynomial correction calibrated against the TS-06-11 file brings it to
  σ = 0.05 %, p99 = 0.15 %, max 0.42 % (validated on 9–11 checkers; error
  *decreases* with checker count above 6).
- **Cubeful equity does not estimate.** The best static model tried (p plus
  both roll distributions' mean, σ, skew, kurtosis — 45 dof) leaves RMS 0.016
  equity, max 0.20. Cube equity is a timing problem along the trajectory; no
  snapshot summary carries it.
- **Match-score conversion via MET is not reliable for verdicts.** Benchmark
  of 5 280 gnubg evaluations (220 positions ≤ 6 checkers — the domain where
  gnubg's own TS base is authoritative — × 12 scores × 2 cube states,
  reference gnubg 2-ply cubeful): the dead-cube `p → MET` chain agrees on
  D/ND only 87.8 % of the time, with real blunders (doubling at p = 0.47 at
  2-away/4-away, cost 17 % MWC — it ignores recubes and the option value of
  waiting). Even gnubg's full 0-ply cube model only reaches 96.5 %.
- **Hosting is free.** GitHub release assets allow 2 GiB per file, unlimited
  total size and bandwidth. gnubg itself ships only a TS-06-06
  (`/usr/share/gnubg/gnubg_ts0.bd`, 6.83 MB) and classifies anything beyond
  6 checkers as BEAROFF_OS — i.e. it, too, falls back to one-sided methods.

## Decision

**Money game is the referential.** The panel shows exact money equities and
the money cube verdict, as the bearoff literature does. Match-score conversion
of *verdicts* is deliberately not offered (measured above). All four planes
are kept.

**Three sources, one reader, widest domain wins.** A single gnubg `.bd`
two-sided reader serves: (1) the embedded `gnubg_ts0.bd` (TS-06-06, 6.8 MB —
gnubg's own default); (2) an optional user-supplied `.bd` path from config;
(3) an optional download of the TS-06-11 published once as a GitHub release
asset under the dedicated tag `bearoff-data-1` (SHA-256 checked, stored under
`$XDG_DATA_HOME/blunderdb/`, deletable from the same UI). Invalid sources are
skipped with a log warning; the embedded socle guarantees an answer. Lookups
are 8-byte `ReadAt`s — the file is never loaded into memory.

**Two regimes, marked, permanent.** Inside the source's domain the panel is
*exact* (lookup). Outside it, win probability is *estimated* (convolution +
frozen calibrated correction) and displayed with its error bound; EPC is
always exact (the OS base covers all 15 checkers); **the cube verdict is
never estimated** — outside the exact domain the verdict line is absent and a
hint offers the download. There is no future state in which the panel is
always exact; the dual regime is the interface.

**No compression, no codec, no persistence.** Nothing is written to user
databases; `DatabaseVersion` is untouched. The 1.23 GB oracle file is never
committed; calibration and CI tests read it via `BLUNDERDB_TS11_PATH` and
skip when absent.

**Parity.** The engine logic lives in `pkg/blunderdb/engine/race` (which also
absorbs the previously duplicated EPC extraction in `database/db_epc.go` and
`internal/server/handlers_positions.go`). The server EPC handler gains the new
fields (the daemon never downloads; admins mount a volume), the CLI gains an
`epc` command, `call` follows the handlers.

## Considered options

- **Embed the 1.23 GB (or a compressed form).** Rejected: measured ratios
  1.1–1.3× lossless; a model+residual codec reaches ~2–3× at the cost of a
  proprietary lossy format, and still leaves 45 % of real bearoffs uncovered.
- **Plane-0-only file (307 MB, 4× smaller).** Rejected when money became the
  referential: the cubeful planes are the product.
- **MET conversion of verdicts to match scores.** Rejected by the benchmark
  above; a "recommendation" wrong in 12 % of decisions is worse than none.
- **Estimated money verdict (calibrated Janowski) outside the base.**
  Rejected: its uncertainty (σ 0.016 equity) straddles the decision frontier
  exactly where the answer is interesting.
- **Extending the domain to no-contact races beyond the home board.**
  Deferred: 58 % of race cube decisions live there, but no oracle exists to
  certify an error bound, and the project committed to displaying one.

## Consequences

- Binary grows ~6.8 MB, including the serve image (parity is worth more than
  the megabytes).
- The user documentation must state every hypothesis behind every displayed
  number (independence of the two race processes, one-sided optimal play,
  correction calibrated on ≤ 11 checkers and extrapolated beyond, money
  gammonless referential, cube-verdict absence outside the exact domain).
  The hypothesis catalogue is maintained in `tasks/ts-bearoff/hypotheses.md`
  and ships in `doc/source/` French text in the same branch as the feature.
- A "défi" training mode (mask results per zone, click to reveal, re-mask on
  edit) rides on the same panel state.
- Follow-up (not blocking): generate a TS-06-12 locally with `makebearoff` to
  validate the correction beyond 11 checkers; rollout bench before ever
  extending to open races.
