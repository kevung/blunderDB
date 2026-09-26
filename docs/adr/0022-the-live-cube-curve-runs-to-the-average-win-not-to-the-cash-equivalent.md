# The live cube curve runs to the average win, not to the cash equivalent

Status: accepted.
See also: ADR-0011, ADR-0019

## Context

Janowski's live cube curve runs from `(CP_live, +1)` to `(1, +W)` above the cash point: the
holder plays on for the gammon, still holding the cube. Clamping that tail to the cash
equivalent (`max(dead, 1)`, and in the match recursion `max(dead, cash)` / `min(dead, pass)`)
makes `eND = (1−x)·e(p) + x` below a point, hence below `eDP = +1`: `TooGood` becomes reachable
only if the *cubeless* equity already exceeds a point. On a reference money position gnubg
2-ply gives ND +1.099 and XG +1.082 (too good / pass); the clamp gave +0.995 (double / pass).
The clamp is the Jacoby branch of gnubg's `MoneyLive()`, applied unconditionally. The
efficiency fit could not see it: its oracle is gammonless (`W = L = 1`), where tail and clamp
coincide.

## Decision

1. **The live cube curve is piecewise linear across `[0, 1]`, from `(0, −L)` to `(1, +W)`.** It
   bends only at the breakpoints the cube state imposes — `CP_live` when the holder may double,
   `TP_live` when he may be doubled, both when centred. **A tail is never a plateau.**
2. **Pieces are named by their endpoints**: `segment(p, x0, y0, x1, y1)` replaces every expanded
   slope in `janowskiEquity` and `levelLive`; a degenerate segment returns its endpoint.
3. **The match recursion follows money's shape with its own anchors**: `levelLive`'s tails run to
   `loseAvg` and `winAvg`. `loseAvg ≤ pass ≤ cash ≤ winAvg` holds by construction, so every
   piece rises and `levelSolve`'s inversion stays monotone.
4. **Jacoby lives in `Decide`'s no-double payoffs (`W = L = 1`), never as a clamp on the
   curve.** With `W = 1` the tail is flat at +1 by itself.
5. **The C, the spec and the port carry the same shape** (`gn_cube.c`, `t34-videau-spec.md` §2,
   `cube.go`), per ADR-0011 rule 7; `testdata/cube_gold.bin` comes from the C.
6. Version label — see ADR-0011 rule 8.

## Consequences

- Cube equities rise past the cash point, in money and at a score, and only there; `TooGood`
  is a verdict users see and the statistics count.
- Nothing gammonless moves: at `W = L = 1` the tails are flat at ±1; no measured cube
  efficiency is invalidated.
- Clamping either tail again is a bug even if all tests pass.
- Rejected: **a cubeful recursion at the root of `Decide`** (gnubg reaches too-good at 0-ply
  from a closed form; it would bury the defect under an expensive layer — re-openable only as
  an improvement on a correct baseline); **fixing the Go port only** (ADR-0011 rule 7);
  **raising the plateau to a tuned constant** (invents a parameter the segment already
  supplies); **documenting the limitation** ("too good" and "not good enough" are opposite
  errors).

## Guard

`pkg/blunderdb/engine/gammonnet/cube_gold_test.go` against `testdata/cube_gold.bin`;
`cube_test.go` (flat tails at `W = L = 1`).
