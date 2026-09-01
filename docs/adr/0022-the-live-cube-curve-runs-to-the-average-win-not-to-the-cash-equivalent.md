# The live cube curve runs to the average win, not to the cash equivalent

## Status

accepted — decided 2026-09-01. **Fixes a defect**, it does not revise a policy: the Janowski
model this codebase says it implements was implemented with two of its segments missing.

Touches nothing ADR-0019, ADR-0020 or ADR-0021 decided — the last of those is a layout decision
about the same panel, taken in parallel the same evening, and the two never meet. (This decision
was drafted as 0021 and renumbered when that one landed first; nothing in it depends on the
number.) ADR-0019 restored the *word* "too good" — it had
been folded into "No Double" by `cubeActionLabel` for want of a fourth label — and ADR-0020 gave
the verdict one shape wherever it is printed. Both were right, and neither could have been
noticed to be insufficient: the verdict they plumbed through was **unreachable**, so a panel that
never says "too good" and a panel that cannot say it look exactly alike from the outside. This
decision is what makes the label ADR-0019 restored appear on a real board.

**Also lands upstream.** `pkg/blunderdb/engine/gammonnet/cube.go` is a port of gammonNet's
`src/gn_cube.c` and its header says so — *"a discrepancy from it is a bug, never an
improvisation"*. The defect is in the C, and in `docs/specs/t34-videau-spec.md` §2 before that.
Fixing only the Go copy would have manufactured exactly the divergence that header forbids, so
the C, the spec, and the port move together, and the cube gold file is regenerated from the
fixed reference.

## Context

The complaint was a single position. Money play, centred cube, X on roll:

```
XGID=bB-B--C-A---eE---c-caa--B-:0:0:1:00:0:0:0:0:0
```

blunderDB said **Double / Pass**. Both reference engines say **too good to double, pass** — and
not narrowly:

| | cubeless | ND | DT | DP | verdict |
|---|---|---|---|---|---|
| gnubg 0-ply | +0.985 | **+1.160** | +1.773 | +1.000 | too good / pass (20.7 %) |
| gnubg 2-ply | +0.949 | **+1.099** | +1.707 | +1.000 | too good / pass (14.0 %) |
| XG Roller++ | +0.935 | **+1.082** | +1.678 | +1.000 | too good / pass (12.1 %) |
| blunderDB 2-ply, before | +0.983 | **+0.995** | +1.780 | +1.000 | double / pass |

The network is not the problem. Our probabilities sit on top of gnubg's (0.733/0.541/0.042
against 0.740/0.527/0.038) and our **cubeless** equity is, if anything, the more optimistic of
the two. Only the cubeful no-double equity decouples, and it decouples in one specific way: it
stops dead at +1.

`janowskiEquity` computed, for a centred cube above the live cash point,

```go
live = math.Max(dead, 1.0)   // "too good treated by the dead continuation"
```

and then blended it, `eND = (1−x)·e(p) + x·live`. That is a **plateau**: for every position
whose cubeless equity is below a point, it returns exactly `(1−x)·e(p) + x`, which is below a
point, which is below `eDP = +1`. Hand-checked against the numbers above: `0.312 × 0.983 +
0.688 × 1.000 = 0.9947`, the reported figure to the fourth decimal. `Verdict`'s first test is
`eND > eDP`. So

> **with the plateau in place, `TooGood` was reachable if and only if the CUBELESS equity
> already exceeded one point.**

Swept at 73 % wins, the verdict flipped at a 55 % gammon rate — where `e(p)` crosses 1.000, not
where playing on starts to beat cashing. The enum value, the `TOO_GOOD` row of the spec's own
verdict table, `race.VerdictTooGood`, `engine.BestCubeVerdict`'s parser, ADR-0019's fourth label
and its nine translations all described a state the model could not produce.

**What the plateau throws away.** Above the cash point the cube holder does not stop at the cash
equivalent: he plays the game on for the gammon, *still holding the cube*, and can still cash
later. Janowski's live curve prices that — it runs from `(CP_live, +1)` to `(1, +W)`, reaching
the average win when the game is certain. gnubg's `MoneyLive()` (`eval.c:3632`) writes that
segment out, and writes the plateau too, in exactly one place:

```c
return (pci->fJacoby) ? 1.0f : (+1.0f + (rW - 1.0f) * (p - rCP) / (1.0f - rCP));
```

The plateau is the **Jacoby** branch — gammons do not count before the cube is turned, so the
no-double branch really is capped at cashing. We were applying it unconditionally. And doubly:
`Decide` already implements Jacoby properly, by setting `W = L = 1` on the no-double branch,
which makes the correct segment flat at +1 all by itself.

The same flattening was in the match recursion, `levelLive`, as `math.Max(dead, lv.cash)` and
`math.Min(dead, lv.pass)`.

**Why this was never caught by the calibration.** `x` was fitted (gammonNet T34) against the
two-sided bearoff oracle, whose domain is gammonless — `W = L = 1`. There, `1 + (W−1)·(…) = 1`:
the corrected segment and the plateau are *the same number*. The fit could not see the defect,
and correcting it moves no measured `x`. The gammon component of the model was only ever
validated "by comparison with GNU Backgammon", which is the comparison that finally ran.

## Decision

1. **The live cube curve is piecewise linear across the whole of `[0, 1]`, and its endpoints are
   `(0, −L)` and `(1, +W)`.** It bends at whatever breakpoints the cube state puts in its way —
   `CP_live` when the holder may double, `TP_live` when he may be doubled, both when the cube is
   centred — and at no point is any piece of it constant. **A tail is never a plateau.**

2. **The pieces are named by their endpoints.** A `segment(p, x0, y0, x1, y1)` helper replaces
   every expanded slope in `janowskiEquity` and `levelLive`, so each branch reads as the row of
   the spec table it implements. A degenerate segment returns its endpoint rather than dividing
   by zero — bisected breakpoints can land at 0 or 1.

3. **The match recursion follows money's shape with its own anchors.** `levelLive`'s tails run
   to `loseAvg` and `winAvg` for the same reason money's run to `−L` and `+W`: past the cash
   point the game is played on, not conceded. `loseAvg ≤ pass ≤ cash ≤ winAvg` holds by
   construction, so every piece rises and `levelSolve`'s bisection keeps the monotonicity it
   stands on.

4. **Jacoby stays where it already is.** It is a property of the *no-double branch's payoffs*
   (`W = L = 1`), handled in `Decide`, never a clamp on the curve. The corrected segment is flat
   at +1 when `W = 1`, so the Jacoby case comes out right without a special case — which is the
   test that the two mechanisms were never the same mechanism.

5. **The port moves with the reference.** `gn_cube.c`, `t34-videau-spec.md` §2, and the Go copy
   carry the same shape, and `testdata/cube_gold.bin` is regenerated from the fixed C. The
   spec's provenance note is amended rather than left standing: it claimed no GNU Backgammon
   source had been read, and that is no longer true. **No gnubg line was copied** — the formula
   is Janowski 1993, gammonNet stays MIT — but the *diagnosis* came from reading `MoneyLive()`,
   and a provenance note that hides where a fix came from is worth nothing.

6. **`EngineVersion` becomes `gammonNet v1.3.0`.** Every stored cube analysis past the cash
   point changes value, so stored v1.2.0 rows are stale and `AnalyzeStaleGammonNet` re-runs
   them. Checker-move equities do not move — the search values plays, not cube states — but the
   version is per-analysis, so a position is stale as a whole and re-run as a whole.

## Considered options

**Add a cubeful recursion at the root of `Decide`.** This was the plan before the cause was
found, and it was aimed at a model believed to be structurally incapable of "too good". It is
not: gnubg reaches +1.160 at **0-ply**, from a closed form, with no recursion at all. `Value`
already carries antisymmetric cubeful values through the search, so the option remains open on
its merits — but it would have been built on top of a curve that was still wrong, and it would
have hidden the defect under an expensive layer rather than removing it. Rejected as a fix;
re-openable as an improvement, measured against a correct baseline.

**Fix the Go port only.** Rejected: `cube.go`'s header makes any divergence from `gn_cube.c` a
bug by definition, and the cube gold file exists precisely to catch it. The gate would have gone
red on the next run and the next reader would have "fixed" it by reverting this.

**Raise the plateau to some tuned constant above 1.** Rejected: it invents a parameter where the
model already has the answer, and it would have to be re-fitted per `W` — which is exactly what
the `(CP_live, +1) → (1, +W)` segment is.

**Leave it and document the limitation.** Rejected. The verdict is not decorative: "too good to
double" and "not good enough to double" are opposite errors, and a tool that reports the second
where the player made the first mis-teaches the exact skill it exists to train.

## Consequences

- Cube equities rise for every position past the cash point, in money and at a score, and only
  there. On the reference position, ND goes +0.995 → **+1.142** (gnubg 2-ply +1.099, XG +1.082).
- `TooGood` becomes a verdict users actually see, and a row the statistics count (ADR-0019
  already routed it into `storage/stats.go`).
- **Nothing gammonless moves.** Pinned by a test: at `W = L = 1` the tails are flat at ±1, the
  plateau's exact values. No measured cube efficiency is invalidated.
- The cube gold gate still measures `max|Δ| = 2.463e-06` over 2320 decisions with 4 tolerated
  ties — the same figures as before the fix, which is the port's own statement that it changed
  in step with the reference and not otherwise.
- Databases analysed with gammonNet v1.2.0 or earlier need a re-run for their cube decisions to
  be right. That is what the version bump is for.
