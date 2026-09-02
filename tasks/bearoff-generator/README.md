# Bearoff generator — task sheets

Decided 2026-09-03; the decision record is ADR-0027. Two lots, one sheet each:

| Lot | Sheet | Scope in one line |
|---|---|---|
| 1 | [lot-1-generator.md](lot-1-generator.md) | Pure generators for both gnubg tables, byte-identical and fingerprint-verified; nothing embedded, nothing downloaded; two-sided domain picker with size/time/memory guard, pause/resume, CLI parity, docs |
| 2 | [lot-2-epc-beyond-home.md](lot-2-epc-beyond-home.md) | The EPC panel answers positions with checkers beyond the six-point on an *n*-point one-sided table; the picker gains the one-sided axis |

Lot 2 depends on lot 1 (the one-sided generator is written for *n* points in lot 1, exposed in
lot 2).

## Key numbers (measured 2026-09-03, prototype in the session scratchpad)

| Fact | Value |
|---|---|
| Prototype TS-06-06 generation, 16 cores / 1 core, unoptimised | 2.3 s / 5.5 s, of which 1.1 s naive move precompute |
| `makebearoff -t 6x6` (C, temp file + hash) | 24 s |
| Identity with `gnubg_ts0.bd` | byte for byte, 6 830 248 B |
| Two-sided size at 6 points, *n* checkers | `C(6+n,6)² × 8 + 40` B: 6→6.8 MB, 8→72 MB, 10→513 MB, 11→1.22 GB, 12→2.76 GB, 15→23.5 GB |
| One-sided size, 15 checkers, *p* points (compressed gnubg format) | 6→1.4 MB, 7→~11 MB, 10→~209 MB, 12→~1.1 GB (uncompressed bound `C(p+15,p)×64`) |
| Sweep order (two-sided) | diagonals of constant `us+them`; `(us,them)` reads row `them`, columns `j < us` — ordering by `us` alone is WRONG (first mismatch at pair (4,5)) |
| gammonNet 2-ply as floor | rejected: `TestBearoffFloorMeasure`, worst decision 0.45, DT→ND on last-roll positions |

## Working rules

- Each sheet ≤ 500 lines; one lot = one worktree branch = one merge.
- Exactness is identity with gnubg (ADR-0027 §1). No "close enough": a differing byte is a
  failing test, and the fingerprint check refuses the file at runtime.
- The generator never touches floating point. Everything is `int32` on `int16` storage, as
  `makebearoff.c` does with `short int`.
- A user-visible change ships with `doc/source/manuel.rst` (+ eight `.po`) in the same branch.
