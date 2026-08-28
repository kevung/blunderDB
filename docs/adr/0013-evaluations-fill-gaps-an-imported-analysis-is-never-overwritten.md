# Evaluations fill gaps; an imported analysis is never overwritten

## Status

accepted — decided 2026-08-28

## Context

An Evaluation is a computation over the position on the board; an Analysis is a record
attached to a saved Position (`CONTEXT.md`). Once blunderDB can evaluate, the question is
whether an Evaluation ever crosses into being an Analysis.

It usefully can. A match imported from a `.txt` or `.mat` file arrives with no analysis at
all, and a library full of unanalysed positions is a library the multi-criteria search cannot
reach: `PopulateAnalysisColumns` derives the indexed scalar columns — `best_move_equity_error`,
`cube_error`, `player1_win_rate` and the rest — from the stored Analysis, and the search
filters on those columns.

That same mechanism is the risk. Those columns carry no engine and no depth. A library half
measured by XG at 3-ply and half by a 2-ply evaluator answers "moves with error > 0.08" by
mixing two rulers, silently. The schema is not the problem — `SaveAnalysis` keeps one row per
Position and merges engines inside it, every entry tagged with its own `AnalysisEngine` and
`AnalysisDepth`, so the data is self-describing. The *search* is what cannot see it.

## Decision

**An Evaluation may be written down, and it only ever fills a gap.**

- gammonNet writes an Analysis for a Position that has none. A Position already carrying an
  imported analysis — XG, GNUbg, BGBlitz — is left alone, and the indexed scalar columns
  never move for it. A search that knew a corpus keeps knowing the same corpus.
- The consequence is accepted and named: at library scale the ruler is mixed, part XG, part
  gammonNet, and the search does not say so. Making it say so is a separate ticket (a
  provenance criterion), not a reason to block this one.

**Writing happens on a bounded, visible job — never as a resident background task.**

- A configuration option enables it: after an import that brought no analysis, one job runs
  over a known number of positions, with a progress bar, cancellable, exactly like an import.
- Closing the application mid-job costs nothing and needs no journal: each analysed Position
  is written as it is produced, so resuming means looking again for Positions without a
  gammonNet analysis. Idempotence comes free from the gap rule.
- An explicit catch-up action covers a library built before the feature existed — the same
  operation, reachable from the GUI, from the CLI, and from `serve`.
- One job at a time, N−1 cores, and interactive evaluation goes first: the batch yields
  within one position.

**What is displayed and what is written are two settings, named separately.**

- *Display depth* is comfort: a slow machine may stay at 0-ply while reading the board.
- *Analysis depth* is what the batch writes.
- Both default to the canonical parameters (2-ply, `k=12`). Conflating them is what turns a
  comfort knob into silent damage to data — lowering the depth to smooth the display would
  otherwise degrade everything written afterwards, with nothing to connect cause to effect.

## Considered options

- **Never persist.** No writes, no migration, no provenance question, no risk to the indexed
  columns. Rejected: an imported match without analysis stays unreachable to the search, and
  a studied position must be recomputed at every opening.
- **Persist only on an explicit gesture**, position by position. Keeps the concept clean but
  cannot address a library, which is the actual need.
- **A resident background task** filling gaps whenever the machine is idle. Rejected: no
  natural end, a database that grows without a gesture, and the hardest regime to make
  visible — the user never knows whether it is running.
- **One setting governing both display and what is written.** Rejected: a comfort knob that
  damages data.

## Consequences

- No schema change and no `DatabaseVersion` bump: the existing merge in `SaveAnalysis`
  already carries per-entry engine and depth.
- The batch is the first long-running computation in blunderDB other than an import; it
  follows the `DownloadBearoffDB` pattern — goroutine, `context` cancellation, `EventsEmit`
  for progress.
- A Svelte effect driving the panel must never read back state it has just written; the
  freeze fixed in `fcde0243` (`effect_update_depth_exceeded`, which stopped the DOM
  altogether) is the precedent.
- The configuration gains a gammonNet tab; two depth settings must be named well enough that
  the distinction is understood without the manual.
