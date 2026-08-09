# Two-sided bearoff integration — task sheet

Decided 2026-08-09 after a measurement-driven design session; the decision
record is ADR-0009. This sheet is the execution plan. The hypothesis catalogue
that must reach the user documentation is in `hypotheses.md` (same directory).

Scope in one line: the EPC panel gains exact win probability and money cube
verdicts from gnubg two-sided bearoff databases, an estimated-p fallback with
a calibrated error bound, a défi training mode, and CLI/server parity —
without embedding more than 6.8 MB.

## Key numbers (measured, session scripts in scratchpad — reproduce if needed)

| Fact | Value |
|---|---|
| TS-06-11 file (user's, `gnubg_ts6x11.bd`) | 1 225 323 048 B; header 40 B; nPos = C(17,6) = 12 376; entry = 4×uint16 |
| Equity scale (all 4 planes) | `u/32767.5 − 1`, player on roll; planes: 0 cubeless, 1 owned, 2 centered, 3 against |
| Plane order (empirical) | eq(3) < eq(0) < eq(2) < eq(1) |
| Lossless compression ceiling | 1.13× (xz raw) … 1.26× (planes+delta+varint+xz) |
| Convolution p error, raw | σ 0.12 %, antisymmetric bias ±0.11 % around p=0.5 |
| Convolution p error, corrected (32 coefs) | σ 0.048 %, p99 0.15 %, max 0.42 % (n = 9–11) |
| Cubeful static-model floor | RMS 0.016 equity, max 0.20 (45 dof) — do not estimate |
| Dead-cube p→MET vs gnubg 2-ply | 87.8 % D/ND agreement; worst blunder ≈ 17–19 % MWC |
| gnubg 0-ply vs 2-ply (same domain) | 96.5 % D/ND, 95.9 % T/P |
| Our reader vs gnubg on shared domain | p agrees to 1e-6 over 500 sampled pairs |
| gnubg's own default | ships TS-06-06 only (`gnubg_ts0.bd`, 6 830 248 B); >6 checkers → BEAROFF_OS |
| GitHub release asset limit | 2 GiB/file, unlimited total & bandwidth |

## Step 0 — data asset (no code)

- [ ] Publish `gnubg_ts6x11.bd` as the single asset of a release under the
      dedicated tag `bearoff-data-1`; put its SHA-256 in the release body.
      Never re-versioned (immutable mathematical table). The file stays out
      of git (1.23 GB ≫ 100 MiB limit).
- [ ] Keep the local copy; export `BLUNDERDB_TS11_PATH` for calibration/CI.

## PR 1 — EPC unification (zero behaviour change)

The extraction of home-board checkers exists twice with two contracts:
`database/db_epc.go: ComputeEPCFromPosition` (returns `map[string]interface{}`,
GUI) and `internal/server/handlers_positions.go: computeEPCSide` (typed,
daemon). This violates the parity invariant before we even start.

- [ ] Create `pkg/blunderdb/engine/race/`; move home-board extraction + EPC
      there with one typed result contract (server's `epcSide` shape wins).
- [ ] `db_epc.go` becomes a thin delegate; delete `computeEPCSide`.
- [ ] Align `epcStore.js` / `EPCPanel.svelte` with the typed JSON contract
      (`bottom.epc` etc.); restart `wails dev` for bindings.
- [ ] Parity test: old-path vs new-path outputs identical on a corpus of
      positions (all-home, partial, bar, empty).

## PR 2 — engine

- [ ] Two-sided `.bd` reader in `engine/race`: parse `gnubg-TS-06-NN`
      header, `Lookup(usIdx, themIdx)` via 8-byte `ReadAt` at
      `40 + (iu*nPos + it)*8`; no mmap, no full load.
- [ ] Embed `gnubg_ts0.bd` (copy from gnubg distribution, 6.8 MB) beside
      `gnubg_os6.bd`.
- [ ] Source resolution: embedded / config path / downloaded file; widest
      valid domain wins; invalid → log warning, never fatal; re-resolve on
      panel entry, config change, download completion.
- [ ] Convolution of the two OS roll distributions → p; frozen 32-coefficient
      correction in Go source with the regeneration command in a comment.
- [ ] Calibration tool (`cmd/` or `go run`-able) reading
      `BLUNDERDB_TS11_PATH`, emitting the coefficient table + error report.
- [ ] Money cube verdict from the 3 cubeful planes, porting gnubg's
      BEAROFF_TS money logic (reimplement the algorithm; do not copy GPL code
      — blunderDB is MIT).
- [ ] Server: extend the existing EPC handler response (p, equities, verdict,
      regime, error bound); daemon reads socle + configured path only, never
      downloads.
- [ ] CLI: `blunderdb epc <XGID>` (table + `--json`), same resolution minus
      download.
- [ ] Tests: committed small JSON fixtures generated with gnubg (positions +
      expected p/equities/verdicts); env-gated oracle test against TS-06-11
      (skip when `BLUNDERDB_TS11_PATH` unset); correction-error regression
      test asserting σ/p99/max bounds.

## PR 3 — panel + documentation (same branch, per repo rule)

- [ ] Three zones: bottom player (exact EPC block, as today), top player
      (idem), race zone (p, ND / DT / DP money equities, verdict).
- [ ] Regime badges on the race zone only: « exact » or « estimé ± borne » ;
      when the verdict is absent, a discreet line offering the download
      (« videau exact jusqu'à 11 dames — télécharger la base (1,2 Go) »).
- [ ] Download UX in ConfigModal: progress, SHA-256 verification, delete
      button, target `$XDG_DATA_HOME/blunderdb/`.
- [ ] Défi mode: checkbox in panel header, persisted in config; per-zone
      overlays (« cliquer pour révéler »); any board edit re-masks all —
      hook the re-mask on the same `$effect` that drives `updateEPC`
      (`App.svelte`), not on DOM events; badges masked too.
- [ ] i18n: new strings in the 9 frontend locale JSONs.
- [ ] Docs (French source only): `manuel.rst` (panel section rewritten, incl.
      défi), a dedicated methodology subsection transcribing
      `hypotheses.md` in user-facing French, `cmd_mode.rst`,
      `raccourcis.rst`, `CLI_USAGE.md`, `mode_headless.rst` (daemon config
      var). Translations at release time as usual.

## Follow-ups (out of scope, do not silently drop)

- Validate the correction beyond 11 checkers: generate TS-06-12 locally with
  `makebearoff` (≈2.8 GB, never distributed), extend the oracle test.
- Open races (no contact, checkers outside home): 58 % of race cube decisions
  live there; requires a rollout bench before any error bound can be shown.
  Blocked on purpose (ADR-0009, considered options).
- OS gammon distributions (`nzg`/`ioffg`) are still unread in `epc.go`;
  only relevant if 12–15-checker money equities are ever displayed.
