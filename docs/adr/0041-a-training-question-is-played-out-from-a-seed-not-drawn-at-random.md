# A training question is played out from a seed, not drawn at random

## Status

accepted — decided 2026-09-07, with 0040. Refines 0037: the engine playing a few plies
against itself, out of sight, to *make* a position is not a play mode, and this record
says where the line is. The bearoff domain leans on 0027 (exact tables) and 0012 (the
regimes an Evaluation question may find itself in).

## Context

No generator of positions existed. The old `train epc` drew from the browsed list and
threw away every contact position — sixty draws, then a refusal — so a library without
races could not train the EPC at all, and the exercise depended on what the user had
happened to search for.

A uniform draw is not an answer either. « Fifteen checkers on the 24-point » or « twelve
on the ace point, three on the six » are *placements*, not positions: nothing plays
into them, and nobody learns to count what never comes up. What makes a position
realistic is that **a game leads there** — the gaps, the low stacks, the asymmetries of a
real bear-off appear with the frequency they have at the table only if something like a
game produced them.

The same generator has to serve the user who wants to work on *a kind* of position: a
bear-off they just met, a race with a checker sent back, a structure from their own
library. Finding a seed by hand every time is the cost the interview refused to pay.

## Decision

1. **A question is a seed plus `k` plies.** The engine's own legal-move generator and
   evaluator (`moves_gen.go`, gammonNet at one ply; for a bear-off, the exact table is
   enough to choose the move) play `k` random rolls from the seed, and the snapshot is
   the question. Everything runs in Go, in the Wails process, off the rendering thread;
   the tab awaits a promise as it does for `ComputeEPCFromPosition`.

2. **The seed comes from one of three sources, chosen at launch and remembered.**

   - **pool** — a few canonical shapes per exercise (« bear-in complete » for Bearoff,
     « contact just broken » for Evaluation, the opening for Pips, the 36 scores for
     Scores), then `k` drawn in 0–10 (6–40 for Pips, whose seed is the opening);
   - **board** — the position as it stands when the session starts, then `k` in 1–4: the
     neighbourhood of what the user is looking at, never the seed itself, which they
     have just seen;
   - **library** — a position of the open library that fits the domain, `k = 0`: it is
     already real, and simulating from it would add nothing. Greyed out when no library
     is open. For Pips this is the old draw; for Bearoff and Evaluation, a draw filtered
     on the domain.

   The third source is what makes « my own positions » free: load one, start from the
   board, and the exercise never learns what a library is.

3. **A seed outside the exercise's domain is refused by name.** A Bearoff session
   started on a contact position says « this exercise needs a bear-off (both sides in
   their home board) » and starts nothing. No silent adaptation — playing on until
   contact breaks would hand the user a position they did not choose, and the geometry
   would have changed without their deciding it. An empty board falls back to the pool,
   with the same sentence.

4. **Domains.** *Bearoff*: both sides entirely in their home board, 4 to 15 checkers a
   side, the rest borne off, cube centred, money, roller drawn — the domain where the
   EPC differs from the pip count (the wastage, which is what is trained), where the
   engine is exact (0027) and which the Eval panel's default board already is. The lower
   bound is not raised: nothing says a four-checker EPC is trivial. *Evaluation*: any
   position — a race in either regime, or contact — because gammonNet evaluates all of
   them; the EPC is shown after the answer only when the position has an exact one.
   Money play in v1; a match score is a later exercise, not a setting of this one.
   *Pips*: any position. *Scores*: the 36 unordered scores.

5. **The cost is bounded, prefetched, and measured — not assumed.** The next question is
   generated while the current one is being answered, so generation hides behind
   thinking time and the clock, which starts at display, never sees it. `k` is capped;
   past a deadline the generator falls back to `k = 0` on a pool seed rather than wait.
   A Go test measures the per-question cost on the reference machine against a stated
   threshold (100 ms proposed); crossing it is a red, not an impression. Realism is
   measured the same way: the pip-count and wastage histograms of generated positions
   are compared with those of the races of an imported library, rather than judged by
   eye.

## Considered options

- **Draw real positions from the library only** — the perfect realism, rejected as the
  *only* source: it puts the library back at the centre of an exercise that does not
  need one, and a library without races makes the exercise mute. Kept as the third
  source, where it costs nothing.
- **Weighted random placement** (favour low points, cap stacks) — rejected: realism
  asserted by a hand-tuned distribution nobody can check, where a simulated game is
  realistic by construction and checkable against real races.
- **Simulate whole games to reach a bear-off** — rejected: dozens of plies per question
  for nothing the « bear-in complete » seed does not give directly.
- **A long-race exercise of its own, next to Bearoff** — folded into Evaluation, whose
  domain is any position; a « long race » is a seed shape there, not an exercise.

## Consequences

- A `race.GenerateRace`-like entry point in `engine/race` with a Wails binding; `serve`
  may expose it later, nothing in v1 requires it.
- 0037 is unchanged in substance: no dice the user sees, no score that advances, no
  game to review. A reader who finds the engine « playing » in `engine/race` is meant to
  find this record next to it.
- The pool shapes are data of the exercise, not settings: adding one is a code change
  with its histogram check, never a user-facing option.
