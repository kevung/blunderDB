# A training question is played out from a seed, not drawn at random

Status: accepted.
See also: ADR-0037, ADR-0040, ADR-0027

## Context

Drawing training positions from the browsed list makes an exercise mute on a library without
races. A uniform placement (fifteen checkers on the 24-point) is not a position: realism — the
gaps, low stacks and asymmetries of a real bear-off — comes from a game leading there. The same
generator must let the user drill a kind of position they just met without hunting for seeds.

## Decision

1. **A question is a seed plus `k` plies.** The engine's legal-move generator and evaluator
   (gammonNet at one ply; the exact table for a bear-off) play `k` random rolls from the seed;
   the snapshot is the question. It runs in Go, off the rendering thread; the tab awaits a
   promise.
2. **The seed has one of three sources, chosen at launch and remembered per exercise.**
   - **pool** — a few canonical shapes per exercise (« bear-in complete », « contact just
     broken », the opening for Pips, the 36 scores), then `k` in 0–10 (6–40 for Pips);
   - **board** — the position when the session starts, then `k` in 1–4, never the seed itself;
   - **library** — a position of the open library fitting the domain, `k = 0`; greyed without
     an open library.
3. **A seed outside the exercise's domain is refused by name** (« this exercise needs a
   bear-off (both sides in their home board) ») and nothing starts; no silent playing-on. An
   empty board falls back to the pool with the same sentence.
4. **Domains.** *Bearoff*: both sides entirely home, 4 to 15 checkers a side, the rest off,
   cube centred, money, roller drawn. *Evaluation*: any position, money play. *Pips*: any
   position. *Scores*: the 36 unordered scores.
5. **Cost is bounded, prefetched and measured.** The next question is generated while the
   current one is answered (the clock starts at display). `k` is capped; past a deadline the
   generator falls back to `k = 0` on a pool seed. A Go test holds the per-question cost under
   100 ms; generated pip-count and wastage histograms are compared with real races.

## Consequences

- This is not a play mode (ADR-0037): no dice the user sees, no score, no game to review.
- The generator lives in `pkg/blunderdb/engine/training/`; `serve` does not expose it.
- Pool shapes are exercise data, not settings: adding one is a code change with its histogram
  check.
- Rejected: the library as the only source — mute without races; kept as the third source.
- Rejected: weighted random placement — realism asserted by an uncheckable distribution.
- Rejected: simulating whole games to reach a bear-off — dozens of plies for what a seed gives.
- Rejected: a separate long-race exercise — a seed shape of Evaluation.

## Guard

`pkg/blunderdb/engine/training/cost_test.go`,
`pkg/blunderdb/engine/training/evaluation_histogram_test.go`,
`frontend/src/__tests__/trainingTabService.test.js`, `frontend/src/__tests__/TrainingPanel.test.js`.
