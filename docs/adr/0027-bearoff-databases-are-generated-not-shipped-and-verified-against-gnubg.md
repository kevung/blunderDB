# Bearoff databases are generated, not shipped, and verified against gnubg

## Status

accepted — decided 2026-09-03 after a measurement-driven design session (the measurements are
in `tasks/taille-binaire-2026-09.md` and in `TestBearoffFloorMeasure`,
`pkg/blunderdb/engine/gammonnet/`). Extends ADR-0009 and ADR-0012, which it does not
contradict: the three regimes and the rule that a cube verdict is never estimated stand. It
**retires** the downloadable `bearoff-data-1` release asset introduced by ADR-0009 and removes
two embedded files from the binary.

Execution plan: `tasks/bearoff-generator/` (two lots).

## Context

### What started it

The question was the size of the binary (35 MB raw, 19 MB after UPX). The embedded Japanese
font, the suspect, weighs 178 kB. The two files that weigh are the two-sided bearoff table
`gnubg_ts0.bd` (6.8 MB, a third of the compressed delivery) and, to a lesser extent, the
one-sided table `gnubg_os6.bd` (1.4 MB). Lossless re-encoding of the two-sided table was
measured at −1.8 MB for a proprietary format: not worth it.

### The tempting answer, measured and rejected

Since gammonNet now evaluates races at 2 ply with the cube (ADR-0023), could the *evaluated*
regime replace the exact table on its own domain, the table becoming a download? Measured on
4 000 positions of the TS-06-06 domain with the searcher configured exactly as the live panel
configures it:

| | 0-ply | 1-ply | 2-ply |
|---|---|---|---|
| \|Δ win probability\| mean / p95 / max | 1.7 / 5.6 / 20.3 % | 1.4 / 5.1 / 17.2 % | 0.85 / 3.1 / 8.4 % |
| double / no-double agreement, centred cube | 94.0 % | 95.8 % | 94.8 % |
| mean cost of following gammonNet, in exact equity | 0.0095 | 0.0051 | 0.0065 |
| decisions costing ≥ 0.08 | 3.65 % | 2.50 % | 3.00 % |
| worst decision | 0.500 | 0.336 | 0.446 |

The win probability converges with the ply; the verdict does not. Almost every disagreement is
an exact *double/take* that gammonNet declines (2-ply centred: 189 of 206), on positions one or
two rolls from the end: the live-cube model (efficiency, recube vig) undervalues the double by
0.3–0.45 where the cube is in fact dead. The TS-06-06 domain is precisely that region. And for
the win probability alone, the convolution estimator of ADR-0009 (σ 0.05 %) is thirty times
closer than 2-ply. So the exact table stays the floor of the panel.

### The table is derived data

`makebearoff -t 6x6` regenerates `gnubg_ts0.bd` in 24 s. A Go port of its backward induction
(`generate_ts`/`BearOff2`/`CubeEquity`, short-int arithmetic reproduced, pairs swept by
diagonals of constant index sum) reproduced the file **byte for byte** in 2.3 s on 16 cores,
5.5 s on one, before any optimisation. The one-sided generator (`BearOff`, `generate_os`) is
the same shape, simpler. Shipping the generator instead of the data removes 8.3 MB from the
binary and, incidentally, the 1.2 GB download: TS-06-11 generates in about a quarter of an hour
on sixteen cores, the same order as downloading it.

## Decision

1. **Exact means identical to gnubg.** A generated database is correct if and only if it is
   byte-identical to what gnubg's `makebearoff` produces for the same domain. The file format is
   gnubg's `.bd`, unchanged, uncompressed; generated, external and (formerly) downloaded files
   are interchangeable and the reader (`race.TwoSided`, `engine.BearoffDatabase`) does not
   change. "Mathematically exact" is not the criterion: it is not checkable, and gnubg's
   arithmetic quirks (the `~` complement, integer division by 36, the rounding of one-sided
   distributions to 65 535) are part of what "the same answer as gnubg" means.

2. **Verification has three levels.** Identity tests in the repository against the two gnubg
   files kept as fixtures (`gnubg_ts0.bd`, `gnubg_os6.bd` — no longer embedded); a table of
   SHA-256 fingerprints obtained once from `makebearoff` for every domain the panel offers,
   against which every generation checks itself on completion, an unmatched file being refused
   and deleted; and the existing env-gated oracle test against a user-supplied gnubg file. A
   domain beyond the fingerprint table can still be generated, and the panel then says
   **unverified** (glossary: *Verified*), never *exact* with a wink.

3. **Nothing is embedded; the defaults are generated silently on first launch.** The binary
   carries no bearoff data. On first launch, without asking, the application generates TS-06-06
   and OS-06-15 into the data directory (a few seconds, once); until they exist the panel runs on
   the *estimated* regime for the win probability and shows no verdict, exactly as it does today
   outside the domain. The CLI does the same at first need; the daemon generates the defaults at
   start into its data directory if writable, otherwise in memory. The daemon still never
   downloads — computing is not downloading — and exposes no generation over HTTP.

4. **The download is retired.** The `bearoff-data-1` asset stays published for older versions
   but the code no longer references it; the resumable HTTP download, its checksum and its
   status fields go. The user-supplied external path stays.

5. **The domain is chosen by the user, bounded by memory.** The configuration panel offers the
   two-sided domain by checkers per side at six points (6 … 15) and, in lot 2, the one-sided
   domain by points; it shows the exact size, an estimated time, and greys out any domain whose
   table does not fit in the machine's RAM, since the whole table must be resident during the
   sweep. The widest completed and verified-or-unverified table is what `Resolve` picks, as
   today.

6. **The time estimate is calibrated on the machine, never a constant.** The first-launch
   generation of TS-06-06 measures this machine's throughput (pairs per second per core),
   which is stored; a domain's estimate is that throughput scaled by its pair count and by a
   machine-independent moves-per-position factor computed once from the move lists. Once a
   generation runs, the remaining time comes from the measured rate. The core count is a
   setting, default all but one.

7. **The file under construction is the state.** Progress is a marker in the file (diagonal
   reached for two-sided, index reached for one-sided). Pause stops the workers and writes the
   marker; cancel deletes the file; an application exit is a pause. A partial file is never
   resolved as a source. On the next launch the panel offers *Resume* and does not restart on
   its own — except the first-launch defaults, which finish in seconds and resume themselves.

8. **One pure generator, three fronts.** The generator is a package with no UI (progress by
   callback, cancellation by context, checkpoints by file). The CLI exposes it as `bearoff
   generate|list|verify|delete`; the panel is a skin over the same calls; the daemon uses only
   the default-generation entry point.

9. **Two lots.** Lot 1: generators for both tables (written for *n* points from the start, as
   gnubg's are), the two-sided domain picker, one-sided at six points only, verification,
   progress, pause/resume, CLI, removal of embed and download, documentation. Lot 2: the EPC
   panel learns positions with checkers beyond the six-point, on a one-sided table of *n*
   points, and the panel's picker gains that axis.

## Considered options

- **Keep TS-06-06 embedded, generate only larger domains.** Safer first launch; keeps the
  6.8 MB that started the question. Rejected: the defaults generate in seconds, and one code path
  for all origins is simpler than two.
- **gammonNet 2-ply as the floor, table on demand.** Rejected by the measurement above; the
  cube model is structurally wrong where the cube is dead.
- **A compact proprietary encoding of the shipped table.** −1.8 MB for a format nobody else
  reads and a decode step on the only "never estimated" path. Rejected.
- **Mathematical exactness with our own arithmetic.** Cleaner in principle, unverifiable in
  practice, and it would make our files differ from gnubg's for the same domain. Rejected.
- **Generation over HTTP for the daemon.** A tenant-triggerable hour of CPU and gigabyte of
  RAM on a shared host. Rejected; the operator uses the CLI on the volume.

## Consequences

- `DownloadedSHA256`, `DownloadedFileName`, `bearoffExpectedBytes`, the HTTP download and the
  `.part` resume logic disappear; `race.Resolve` gains "generated" origins and a partial-file
  rule. `EmbeddedTwoSided` and `loadBearoffDatabase` lose their embeds and gain a
  data-directory source with in-memory generation as the last resort.
- The two gnubg files move to `testdata/`; the identity tests are the regression gate of the
  generators and run in `go test ./...` for the defaults (seconds). Larger domains are an
  env-gated recipe step, as `TestEvalMeasure` already is.
- Fingerprints for TS-06-07 … TS-06-10 and OS-07 … OS-10 must be produced once with
  `makebearoff` and recorded; TS-06-11's is already known (`c52133cd…`). Anything beyond is
  *unverified* until someone runs `makebearoff` for it.
- The first launch spends a few seconds of CPU and writes ~8 MB to the data directory; a
  read-only data directory falls back to in-memory generation at each start.
- The performance target of the generator is measured, not hoped: TS-06-06 under one second on
  four cores, TS-06-11 under ten minutes on eight, with the pure-Go fallback rules of ADR-0024
  irrelevant here (integer arithmetic, no floating point anywhere in the sweep).
