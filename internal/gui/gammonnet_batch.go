package gui

import (
	"context"
	"errors"
	goruntime "runtime"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/jobs"
)

// The gammonNet batch (ADR-0013): the loops live on *database.Database, shared
// with the CLI and daemon; this file is the GUI shell and its events:
//
//	gammonnet-batch:waiting   {job, ply, group}
//	gammonnet-batch:progress  {done, total, ply, group}
//	gammonnet-batch:done      {evaluated, refused, failed, group}
//	gammonnet-batch:cancelled {group}
//	gammonnet-batch:error     {message, group}
//
// group is the library group the batch analyses: it runs on a handle borrowed
// from that library (libraries.go), so it keeps analysing it when another
// group comes on screen.

// batchRun is the work of one batch, on the library handle it is given.
// ctx carries the batch's task (taskFrom), at whose checkpoint the batch
// yields before each position.
type batchRun func(ctx context.Context, db *database.Database, onProgress func(done, total int)) (database.GammonNetBatchSummary, error)

type taskKey struct{}

// batchTask is what a batch's context carries for its yield: the task, and
// how a preemption stops the current run (taskFrom).
type batchTask struct {
	job       *jobs.Job
	preempt   context.CancelFunc
	preempted *atomic.Bool
}

// taskFrom returns the registry task ctx carries, nil for none.
func taskFrom(ctx context.Context) *batchTask {
	t, _ := ctx.Value(taskKey{}).(*batchTask)
	return t
}

// batchYield is the yield a batch's loop calls before each position: the
// task's checkpoint — pause, every computation that outranks it (the visible
// ones on screen among them), a requested task waiting for the library,
// which stops the run.
func (a *App) batchYield(ctx context.Context) func() {
	t := taskFrom(ctx)
	return func() {
		if t != nil && errors.Is(t.job.Checkpoint(ctx), jobs.ErrPreempted) {
			t.preempted.Store(true)
			t.preempt()
		}
	}
}

// StartGammonNetBatch analyses every position without an analysis, in the
// background, cancelling any batch in flight. No-op when db is nil. While
// another task writes the library it waits its turn in the library's queue
// (ADR-0115).
func (a *App) StartGammonNetBatch(ply, pruneK, candidates int) {
	a.runGammonNetBatch(0, ply, jobs.GammonNetMissing, a.missingBatch(ply, pruneK, candidates))
}

// QueueGammonNetBatch is StartGammonNetBatch for the analysis an import
// starts on its own.
func (a *App) QueueGammonNetBatch(ply, pruneK, candidates int) {
	a.runGammonNetBatch(0, ply, jobs.GammonNetMissing, a.missingBatch(ply, pruneK, candidates))
}

// QueueGammonNetBatchIn is QueueGammonNetBatch for the library of group, on
// screen or not: an import that finished in a group the user left analyses
// that group's library, not the one shown.
func (a *App) QueueGammonNetBatchIn(group, ply, pruneK, candidates int) {
	a.runGammonNetBatch(group, ply, jobs.GammonNetMissing, a.missingBatch(ply, pruneK, candidates))
}

func (a *App) missingBatch(ply, pruneK, candidates int) batchRun {
	return func(ctx context.Context, db *database.Database, onProgress func(done, total int)) (database.GammonNetBatchSummary, error) {
		return db.AnalyzeMissingWithGammonNet(ctx, ply, pruneK, candidates, goruntime.NumCPU(),
			a.batchYield(ctx), onProgress)
	}
}

// StartGammonNetStaleBatch re-runs gammonNet on every position whose analysis
// is entirely its own but stale: another EngineVersion, or another depth.
func (a *App) StartGammonNetStaleBatch(ply, pruneK, candidates int) {
	a.runGammonNetBatch(0, ply, jobs.GammonNetStale, func(ctx context.Context, db *database.Database, onProgress func(done, total int)) (database.GammonNetBatchSummary, error) {
		return db.AnalyzeStaleGammonNet(ctx, ply, pruneK, candidates, goruntime.NumCPU(),
			a.batchYield(ctx), onProgress)
	})
}

// StartGammonNetMatchBatch is StartGammonNetBatch scoped to one match: the
// batch a transcription's or a duel's save starts (ADR-0045 §8). The caller
// sets depth.
func (a *App) StartGammonNetMatchBatch(matchID int64, ply, pruneK, candidates int) {
	a.runGammonNetBatch(0, ply, jobs.GammonNetMatch, func(ctx context.Context, db *database.Database, onProgress func(done, total int)) (database.GammonNetBatchSummary, error) {
		return db.AnalyzeMatchWithGammonNet(ctx, matchID, ply, pruneK, candidates, goruntime.NumCPU(),
			a.batchYield(ctx), onProgress)
	})
}

// runGammonNetBatch is the shell every batch shares: run does the work, this
// owns the single-in-flight bookkeeping, the task in the registry and the
// events. The batch waits its turn in its library's queue, then for any job
// still holding the library outside the registry. A requested task waiting
// for the library preempts it: the run stops at its next position, the batch
// goes back into the queue behind that task and, once its turn comes again,
// runs anew over what is still missing.
func (a *App) runGammonNetBatch(group, ply int, kind jobs.Kind, run batchRun) {
	if a.db == nil {
		return
	}
	// libs.mu until the batch is registered: a group closed meanwhile finds
	// it to stop.
	a.libs.mu.Lock()
	defer a.libs.mu.Unlock()
	group, db, err := a.borrowLibraryLocked(group)
	if err != nil {
		a.emitBatch("gammonnet-batch:error", map[string]any{"message": err.Error(), "group": group})
		return
	}

	a.gnBatchMu.Lock()
	if a.gnBatchCancel != nil {
		a.gnBatchCancel()
	}
	ctx, cancel := context.WithCancel(a.batchCtx())
	waiting := func(job string) {
		a.emitBatch("gammonnet-batch:waiting", map[string]any{"job": job, "ply": ply, "group": group})
	}
	ctx = database.WaitForDatabase(ctx, waiting)
	task := a.jobs.Start(jobs.Spec{
		Kind: kind, Library: group, Priority: jobs.Background,
		Exclusive: true, Pausable: true, Compute: true, Preemptible: true, Cancel: cancel,
	})
	stopped := make(chan struct{})
	a.gnBatchCancel = cancel
	a.gnBatchDone = stopped
	a.gnBatchGroup = group
	a.gnBatchMu.Unlock()

	go func() {
		// Registered first so it runs last, even after a panic: shutdown
		// waits on it, and the task ends, giving its library back.
		defer close(stopped)
		defer task.Finish(errTaskPanicked)
		defer recoverBackground(a.ctx, "gammonNet batch analysis")
		defer db.Close() // lets the borrowed library go; closes nothing
		var summary database.GammonNetBatchSummary
		var err error
		for {
			if holder, ok := task.Behind(); ok {
				waiting(jobCode(holder))
			}
			if err = task.Turn(ctx); err != nil {
				break
			}
			runCtx, stopRun := context.WithCancel(ctx)
			bt := &batchTask{job: task, preempt: stopRun, preempted: new(atomic.Bool)}
			var part database.GammonNetBatchSummary
			part, err = run(context.WithValue(runCtx, taskKey{}, bt), db, func(done, total int) {
				task.Progress(done, total)
				a.emitBatch("gammonnet-batch:progress", map[string]int{"done": done, "total": total, "ply": ply, "group": group})
			})
			stopRun()
			summary.Evaluated += part.Evaluated
			summary.Refused += part.Refused
			summary.Failed += part.Failed
			if !bt.preempted.Load() || ctx.Err() != nil {
				break
			}
			task.Requeue()
			err = nil
		}
		if ctx.Err() != nil && err != nil {
			err = context.Canceled
		}
		task.Finish(err)

		// Only if still the batch in flight: a newer one may have replaced it.
		a.gnBatchMu.Lock()
		if a.gnBatchDone == stopped {
			a.gnBatchCancel = nil
			a.gnBatchDone = nil
			a.gnBatchGroup = 0
		}
		a.gnBatchMu.Unlock()

		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) {
				a.emitBatch("gammonnet-batch:cancelled", map[string]int{"group": group})
				return
			}
			a.emitBatch("gammonnet-batch:error", map[string]any{"message": err.Error(), "group": group})
			return
		}
		a.emitBatch("gammonnet-batch:done", map[string]int{
			"evaluated": summary.Evaluated,
			"refused":   summary.Refused,
			"failed":    summary.Failed,
			"group":     group,
		})
	}()
}

// batchCtx is the Wails context, or a background one before startup
// (context.WithCancel(nil) panics).
func (a *App) batchCtx() context.Context {
	if a.ctx == nil {
		return context.Background()
	}
	return a.ctx
}

// emitBatch emits nothing without a window: EventsEmit on a non-Wails
// context ends the process (ADR-0004).
func (a *App) emitBatch(name string, payload ...any) {
	if hook := a.emitHook.Load(); hook != nil {
		(*hook)(name, payload...)
		return
	}
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, name, payload...)
}

// CancelGammonNetBatch aborts an in-flight batch and returns at once; to wait
// for it to stop, use cancelGammonNetBatch's channel.
func (a *App) CancelGammonNetBatch() {
	a.cancelGammonNetBatch()
}

// cancelGammonNetBatch also returns a channel closed once the batch has
// finished, nil when none was in flight.
func (a *App) cancelGammonNetBatch() <-chan struct{} {
	a.gnBatchMu.Lock()
	defer a.gnBatchMu.Unlock()
	if a.gnBatchCancel != nil {
		a.gnBatchCancel()
		a.gnBatchCancel = nil
	}
	return a.gnBatchDone
}

// stopBackgroundJobs, shutdown's first step, cancels every job and waits (up
// to grace, shared) only for the ones that write the database.
// The bearoff generation never touches it, and waiting for it could hang.
func (a *App) stopBackgroundJobs(grace time.Duration) {
	a.CancelBearoffGeneration()
	a.stopDatabaseJobs(grace)
}

// stopDatabaseJobs cancels every job that writes a library, in any group,
// and waits for them up to grace, shared: before shutdown.
func (a *App) stopDatabaseJobs(grace time.Duration) {
	a.stopGroupJobs(0, grace)
}

// stopGroupJobs is stopDatabaseJobs for the tasks of the registry bound to
// the library of group (0: every group), running or waiting their turn:
// before that library is closed or replaced. A task names the group it
// writes at its start; the rollout and the maintenance run write through the
// bound handle, so they name the group on screen then.
func (a *App) stopGroupJobs(group int, grace time.Duration) {
	// One context for the whole grace period, shared by every task waited on.
	deadline, release := context.WithTimeout(context.Background(), grace)
	defer release()
	a.stopRegisteredJobs(deadline, group, grace)
	// The database's own slot, held by a maintenance pass, is waited for
	// directly: the bound handle is about to change under it.
	if (group == 0 || group == a.shownGroup()) && a.db != nil {
		a.db.StopMaintenance(deadline)
	}
}

// shownGroup is the group on screen, read without libs.mu.
func (a *App) shownGroup() int {
	if id := int(a.libs.shown.Load()); id != 0 {
		return id
	}
	return 1
}
