# Evaluations fill gaps; an imported analysis is never overwritten

Status: accepted.

## Context
An Evaluation is a computation over the board; an Analysis is a record attached to a saved
Position (`CONTEXT.md`). A match imported from `.txt` or `.mat` arrives unanalysed and is out of
reach of the search, which filters on the scalar columns `PopulateAnalysisColumns` derives from
the stored Analysis. Those columns carry no engine and no depth: `SaveAnalysis` tags each entry
with `AnalysisEngine` and `AnalysisDepth`, but the search cannot see them.

## Decision
1. **An Evaluation may be written down, and only ever fills a gap.** gammonNet writes an
   Analysis for a Position that has none. A Position carrying an imported analysis (XG, GNUbg,
   BGBlitz) is left alone and its indexed columns never move.
2. **Writing happens on a bounded, visible job, never a resident background task.** A setting
   enables it after an import that brought no analysis: one job over a known number of
   positions, with progress, cancellable, like an import. An explicit catch-up action runs the
   same operation over an existing library, from the GUI, the CLI and `serve`.
3. **Each Position is written as it is produced.** Quitting mid-job costs nothing and needs no
   journal; resuming means looking again for Positions without a gammonNet analysis —
   idempotence follows from rule 1.
4. **One job at a time, N−1 cores, interactive evaluation first**: the batch yields within one
   position.
5. **Display depth and analysis depth are two settings.** Display depth is comfort; analysis
   depth is what the batch writes. Both default to the canonical parameters (2-ply, `k=12`).

## Consequences
- At library scale the ruler is mixed (part XG, part gammonNet) and the search does not say so;
  a provenance criterion is separate work.
- No schema change: `SaveAnalysis` already carries per-entry engine and depth.
- The batch follows the long-job pattern: goroutine, `context` cancellation, `EventsEmit`
  progress.
- Rejected: never persisting — unanalysed imports stay unreachable to search, and studied
  positions are recomputed at every opening.
- Rejected: persisting only position by position — cannot address a library.
- Rejected: a resident background task — no natural end, invisible, a database growing without
  a gesture.
- Rejected: one setting for display and writing — a comfort knob that silently degrades data.
