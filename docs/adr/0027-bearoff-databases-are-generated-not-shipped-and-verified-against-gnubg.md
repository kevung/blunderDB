# Bearoff databases are generated, not shipped, and verified against gnubg

Status: accepted.
See also: ADR-0009, ADR-0012.

## Context
The two-sided table `gnubg_ts0.bd` (TS-06-06, 6.8 MB) and the one-sided `gnubg_os6.bd` (1.4 MB)
were the heaviest data in the binary. They are derived data: a Go port of gnubg's `makebearoff`
backward induction reproduces `gnubg_ts0.bd` byte for byte in a few seconds, and TS-06-11 in
minutes. gammonNet 2-ply cannot replace the table on its domain: it agrees on double/no-double
only 94.8 % of the time (worst decision 0.446), declining exact double/takes where the cube is
dead; and for win probability the convolution estimator (ADR-0009 rule 3) is thirty times closer.

## Decision
1. **Exact means identical to gnubg.** A generated table is correct iff it is byte-identical to
   what `makebearoff` produces for the same domain. The format is gnubg's `.bd`, unchanged and
   uncompressed; generated and external files are interchangeable and read by the same reader
   (`race.TwoSided`, `engine.BearoffDatabase`). gnubg's arithmetic quirks (the `~` complement,
   integer division by 36, one-sided distributions rounded to 65 535) are part of the answer.
2. **Verification has three levels**: identity tests against the two gnubg files kept as
   fixtures (`pkg/blunderdb/engine/bearoffgen/testdata/`); a table of SHA-256 fingerprints
   recorded from `makebearoff`, against which every generation checks itself on completion —
   a mismatching file is refused and deleted; the env-gated oracle test against a user-supplied
   gnubg file. A domain outside the fingerprint table can be generated and is then shown as
   **unverified** (glossary: *Verified*), never *exact*.
3. **Nothing is embedded; the defaults are generated silently on first need.** The binary
   carries no bearoff data. The application generates TS-06-06 and OS-06-15 into the data
   directory on first launch without asking; until they exist the panel runs on the estimated
   regime with no verdict. The CLI does the same at first need; the daemon generates them at
   start into its data directory if writable, otherwise in memory. The daemon never downloads
   and exposes no generation over HTTP.
4. **No download.** The code references no release asset. A user-supplied external `.bd` path
   remains a source.
5. **The user chooses the domain, bounded by memory.** The configuration panel offers the
   two-sided domain by checkers per side (6 … 15) and the one-sided domain by points, shows the
   exact size and an estimated time, and greys out any domain whose table does not fit in RAM
   (the whole table is resident during the sweep). `Resolve` picks the widest completed table,
   verified or unverified.
6. **The time estimate is calibrated on the machine, never a constant**: the first generation
   measures this machine's throughput (pairs per second per core), scaled per domain by its pair
   count and a machine-independent moves-per-position factor; once running, the remaining time
   comes from the measured rate. The core count is a setting, default all but one.
7. **The file under construction is the state.** Progress is a marker in the file (diagonal for
   two-sided, index for one-sided). Pause stops the workers and writes the marker; cancel
   deletes the file; an application exit is a pause. A partial file is never resolved as a
   source. On the next launch the panel offers *Resume* and does not restart on its own —
   except the first-launch defaults, which resume themselves.
8. **One pure generator, three fronts.** `pkg/blunderdb/engine/bearoffgen` has no UI (progress
   by callback, cancellation by context, checkpoints by file). The CLI exposes
   `bearoff generate|list|verify|delete`; the panel is a skin over the same calls; the daemon
   uses only the default-generation entry point.
9. **Two lots.** Lot 1: both generators written for *n* points, the two-sided domain picker,
   verification, progress, pause/resume, CLI. Lot 2 (second lot): the EPC panel reads positions
   with checkers beyond the six-point from a one-sided table of *n* points, and the picker gains
   that axis; at six points the domain label is not shown.

## Consequences
- First launch spends a few seconds of CPU and writes ~8 MB; a read-only data directory falls
  back to in-memory generation at each start.
- Fingerprints exist only for domains someone generated with `makebearoff`; anything beyond is
  unverified until then.
- The generator is integer-only: ADR-0024's floating-point rules do not apply.
- Rejected: keeping TS-06-06 embedded — the defaults generate in seconds, and one code path for
  all origins is simpler.
- Rejected: gammonNet 2-ply as the floor — structurally wrong where the cube is dead.
- Rejected: a compact proprietary encoding — −1.8 MB for a format nobody reads and a decode step
  on the "never estimated" path.
- Rejected: our own "mathematically exact" arithmetic — unverifiable, and files would differ from
  gnubg's.
- Rejected: generation over HTTP — a tenant-triggerable hour of CPU and gigabyte of RAM on a
  shared host; the operator uses the CLI on the volume.

## Guard
`pkg/blunderdb/engine/bearoffgen/twosided_test.go`, `onesided_test.go`, `verify_test.go`,
`checkpoint_test.go`; `pkg/blunderdb/engine/gammonnet/bearoff_floor_measure_test.go`.
