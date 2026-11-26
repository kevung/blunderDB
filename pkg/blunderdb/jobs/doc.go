// Package jobs is the desktop's one scheduler of durable work (ADR-0115): a
// registry of every task that outlives a click — gammonNet re-analyses,
// rollouts, maintenance, the match_stats fill, imports, bearoff tables,
// synchronisation — and, per library, an in-memory queue of the tasks that
// write it one at a time.
//
// A task is registered with Start, which describes it (Spec) and returns its
// handle. An exclusive task then waits its turn in its library's queue
// (Job.Turn) instead of being refused; a pausable one calls Job.Checkpoint
// between units of work, where it stops while paused and while a computation
// of a higher priority runs. Neither waits by polling: every change to the
// registry wakes the waiters.
//
// Three priorities order the engine's work: Visible (what is on screen —
// evaluation, maps of rolls and scores), entered with EnterVisible and never
// listed; Requested (a rollout the user launched); Background (re-analysis,
// maintenance). The cores each search uses stay those of ADR-0093: priority
// decides who runs, not how wide.
//
// The queue is not persistent. Resuming after a close is what it was before
// the registry: a batch finds what is missing, an import its journal.
package jobs
