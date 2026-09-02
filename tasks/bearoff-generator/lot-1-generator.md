# Lot 1 — the generators, nothing embedded, nothing downloaded

ADR-0027. Branch `feat/bearoff-generator`. Everything below is ordered; a step may only start
when the previous one is green.

## 1. Package `pkg/blunderdb/engine/bearoffgen` (pure, no UI)

Port of `gnubg/makebearoff.c` — `generate_ts` / `BearOff2` / `CubeEquity` for the two-sided
table, `generate_os` / `BearOff` / `WriteOS` / `WriteIndex` for the one-sided one. Reference
implementation to read first: the session prototype (scratchpad `ts_gen_prototype_test.go.txt`,
also described in ADR-0027) which is already byte-identical for TS-06-06.

- [ ] `Domain` type: `{Kind: TwoSided|OneSided, Points, Checkers int}`; `Size() int64` exact
      (two-sided: `40 + nPos²×8`; one-sided: header + index + data, data size only known after
      generation — expose the uncompressed bound); `FileName()` (`gnubg_ts<P>x<C>.bd`,
      `gnubg_os<P>.bd`, matching the existing `gnubg_ts6x11.bd` name); `String()` (`TS-06-11`).
- [ ] Move lists: `reach[i][roll]` distinct successor indices per position and roll, computed
      once per domain with the bearoff rules (both dice if possible, larger die alone otherwise,
      doubles ×4). Replace the prototype's map/recursion with a preallocated slice walk; this is
      the 1.1 s to bring under 50 ms for TS-06-06. Also yields `MovesFactor(domain)`: mean
      successors per (position, roll), the machine-independent term of the time estimate.
- [ ] Two-sided sweep: planes stored as one `[]uint16` (`(us*nPos+them)*4+k`, exactly the file
      body layout so writing is a copy); diagonals of constant `us+them`; a **persistent** worker
      pool fed one diagonal at a time (no goroutine spawn per diagonal — 1 847 diagonals ×
      16 spawns was measurable); `int32` arithmetic, `^` complement, `/36` truncation, as C.
- [ ] One-sided sweep: sequential by index (position *i* reads only `j < i`); per position, 21
      rolls × successors, `RollsOS` selection (minimum expected rolls, ties by first), distributions
      as 32 `uint16` (+32 gammon when all 15 checkers are on the board), rounding
      `(sum+18)/36` then the mode absorbs the residual to 65 535, compressed index exactly as
      `WriteIndex` (offset, nz, ioff, nzg, ioffg). Verify against `gnubg_os6.bd` byte for byte.
- [ ] Progress: `func(done, total int64)` callback, throttled by the caller; cancellation by
      `context.Context`; **checkpoint** = the diagonal (or index) reached, written into the
      `.part` file's trailer together with the domain and a CRC of the body so far. `Resume(path)`
      reloads the body and continues. The trailer is stripped on completion.
- [ ] Completion: rename `.part` → final name only after the fingerprint step (§2) passed or
      returned "unknown domain".
- [ ] Memory guard: `RAMNeeded(domain)` = table + reach lists; compared by the caller to
      `MemAvailable` (Linux `/proc/meminfo`, macOS `sysctl hw.memsize`, Windows
      `GlobalMemoryStatusEx` — one small file per OS, like the existing host-capability probes).
- [ ] Time estimate: `Estimate(domain, throughput, cores)` where `throughput` is pairs/s/core
      measured on this machine (§4) and the formula is `nPos² × MovesFactor / (throughput ×
      cores)`; live ETA from the measured rate once running.
- [ ] Tests (run in `go test ./...`, seconds): `TestTwoSided_6x6_IdenticalToGnubg`,
      `TestOneSided_6_IdenticalToGnubg` against `testdata/gnubg_ts0.bd`, `testdata/gnubg_os6.bd`
      (the two files leave `engine/` and `engine/race/` for `testdata/`); `TestResume_MidSweep`
      (cancel at a diagonal, resume, identical output); `TestCancel_LeavesPartOnly`.
      Env-gated: `BLUNDERDB_BEAROFF_ORACLE_DIR` with `makebearoff` outputs for 6x7…6x11 and
      os7…os10, identity test per file present.
- [ ] Benchmark gate (recipe, not CI): TS-06-06 < 1 s on 4 cores; TS-06-11 < 10 min on 8 cores,
      < 1.3 GB RSS. Record the numbers in this sheet when reached.

## 2. Fingerprints

- [ ] `fingerprints.go`: `map[Domain]string` SHA-256. Seed: TS-06-06 and OS-06 (hash the
      fixtures), TS-06-11 (`c52133cd59a7db478a71d18c8f2093ba343200fa72ede8004c32c6778c724f46`,
      the retired asset).
- [ ] Run `makebearoff -t 6x7 … 6x10` and `-o 7 … 10` once (hours for 6x10; background job,
      the outputs go to `BLUNDERDB_BEAROFF_ORACLE_DIR` for the oracle test) and record their
      hashes here and in the map. Anything not in the map is *unverified*.
- [ ] `Verify(path) (Verified|Unverified|Corrupt, error)`: header → domain → size check →
      hash → map lookup. `Corrupt` deletes nothing by itself; the caller decides.

## 3. Sources (`engine/race/source.go`, `engine/epc.go`)

- [ ] Remove `//go:embed gnubg_ts0.bd`, `EmbeddedTwoSided`, `DownloadedFileName`,
      `DownloadedSHA256`, `DownloadedPath`; remove `//go:embed gnubg_os6.bd` and the
      `bearoffDB` FS in `epc.go`.
- [ ] `Resolve()` candidates: external path, then every `gnubg_ts6x*.bd` in the data dir
      (widest valid wins; a `.part` is never a candidate). Same for the one-sided table
      (`gnubg_os6.bd` in the data dir; lot 2 widens).
- [ ] `EnsureDefaults(ctx, dataDir, progress)`: generate TS-06-06 and OS-06 if absent (resume a
      `.part` if present — the one auto-resume of ADR-0027 §7); if the dir is not writable,
      generate in memory and serve from `bytes.Reader` (today's embedded path, minus the embed).
      Measures and returns the throughput (§1 estimate) on the first real generation.
- [ ] Wire `EnsureDefaults` into GUI startup (background, Wails event on completion so the EPC
      panel refreshes), CLI (`epc`, `analyze` — at first need, with a one-line notice on
      stderr), daemon (`serve` start).

## 4. Configuration and persistence

- [ ] `config.go`: `BearoffThroughput float64` (pairs/s/core, measured), `BearoffCores int`
      (0 = all but one). Keep `BearoffTSPath` (external file).
- [ ] Remove `bearoffExpectedBytes`, `DownloadBearoffDB`, `CancelBearoffDownload`,
      `downloadBearoff`, `resumableDownload` from `internal/gui/bearoff.go`.

## 5. GUI — the Bearoff tab of `ConfigModal.svelte`

- [ ] Status block per table kind: active source (origin, domain, verified / unverified /
      external), list of generated files with size and a Delete button each.
- [ ] Generator block (two-sided only in this lot): checkers picker 6 … 15 at six points; for
      the selection, exact size, RAM needed vs available (greyed with the reason when it does
      not fit), estimated time on the chosen core count; core count picker; Generate.
- [ ] Progress: bar + measured ETA; Pause / Resume / Cancel; survives closing the modal (state in
      the store, events `bearoff:progress|paused|done|error` as today's download events).
- [ ] Interrupted-at-launch state: "TS-06-09 interrompue à 43 % — Reprendre / Supprimer".
- [ ] i18n: all nine languages in `frontend/src/i18n/*.json`; font tokens per ADR-0008.

## 6. CLI — `internal/cli/cli_bearoff.go`

- [ ] `blunderDB bearoff generate --ts 6x9 [--cores N] [--data-dir D]` with a text progress line
      and ETA, Ctrl-C = pause (the `.part` stays), rerun = resume; `bearoff list`;
      `bearoff verify <file>`; `bearoff delete --ts 6x9`. Register in `handlers()` only.
- [ ] `CLI_USAGE.md` entry.

## 7. Server

- [ ] `serve` start calls `EnsureDefaults` (in-memory fallback if the data dir is read-only);
      no route. `doc/source/mode_headless.rst`: the operator generates large domains with the
      CLI on the volume, as with an external file today.

## 8. Documentation and release notes

- [ ] `doc/source/manuel.rst`: Bearoff tab rewritten (generation replaces download; verified /
      unverified explained in one paragraph; memory and time guidance); + eight `.po`;
      `scripts/doc-i18n-check.sh` green.
- [ ] `CLAUDE.md`: architecture line for `engine/bearoffgen`; the "Nothing is recorded on the
      recipient's side" invariant is untouched (a generated table is not a registry).
- [ ] ADR index row for 0027 (done in the design commit) — check it still reads true.
- [ ] `tasks/taille-binaire-2026-09.md`: append the measured binary size after the embed
      removal (expected ≈ −8.3 MB raw, ≈ −6.5 MB compressed).

## Acceptance

- `go test ./...` green, identity tests included; `go test -race` on `bearoffgen`.
- Fresh data dir + GUI launch: EPC panel shows a verdict within ~5 s without any click.
- `wails build` binary smaller by ≥ 8 MB raw than `0.35.0`'s.
- `bearoff generate --ts 6x8` on this machine: identical hash to `makebearoff -t 6x8`.
