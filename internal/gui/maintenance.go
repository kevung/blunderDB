package gui

import (
	"context"
	"errors"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/jobs"
	"github.com/kevung/blunderdb/pkg/blunderdb/maintenance"
)

// StartMaintenance runs one line of the Maintenance tab in the background and
// returns at once: a pass over a large library takes minutes, and a bound
// method blocks the Wails call that made it. It waits its turn in the
// library's queue (ADR-0115), then reports on maintenance:progress, then
// exactly one of maintenance:done (the Run), maintenance:cancelled or
// maintenance:error ({message}); a refusal (read-only, unknown line) is an
// error event too, so the frontend reads every outcome in one place.
// CancelMaintenance ends the pass between batches.
func (a *App) StartMaintenance(kind string) {
	if a.db == nil {
		return
	}
	if err := a.db.RefuseReadOnly(); err != nil {
		a.emitBatch("maintenance:error", map[string]string{"kind": kind, "message": err.Error()})
		return
	}
	// On a handle borrowed from the library on screen: a pass waiting in the
	// queue across a switch of groups still runs on the library it was asked
	// for (ADR-0101 §5).
	a.libs.mu.Lock()
	group, db, err := a.borrowLibraryLocked(0)
	if err != nil {
		a.libs.mu.Unlock()
		a.emitBatch("maintenance:error", map[string]string{"kind": kind, "message": err.Error()})
		return
	}
	ctx, cancel := context.WithCancel(a.batchCtx())
	task := a.jobs.Start(jobs.Spec{
		Kind: jobs.Maintenance, Label: kind, Library: group, Priority: jobs.Background,
		Exclusive: true, Cancel: cancel,
	})
	a.libs.mu.Unlock()
	go func() {
		defer task.Finish(errTaskPanicked) // after a panic: frees the library
		defer cancel()
		defer recoverBackground(a.ctx, "maintenance")
		defer db.Close() // lets the borrowed library go; closes nothing
		var run maintenance.Run
		err := task.Turn(ctx)
		if err == nil {
			// A cancel that came after the turn is in ctx: the pass does not
			// start, or stops between two batches.
			run, err = database.RunMaintenanceContext(db, database.WaitForDatabase(ctx, nil), kind, func(done, total int) {
				task.Progress(done, total)
				a.emitBatch("maintenance:progress", map[string]any{"kind": kind, "done": done, "total": total})
			})
		}
		switch {
		case err == nil:
			task.Finish(nil)
			a.emitBatch("maintenance:done", run)
		case run.Outcome == maintenance.OutcomeCancelled || errors.Is(err, context.Canceled) || ctx.Err() != nil:
			task.Finish(context.Canceled)
			a.emitBatch("maintenance:cancelled", map[string]string{"kind": kind})
		default:
			task.Finish(err)
			a.emitBatch("maintenance:error", map[string]string{"kind": kind, "message": err.Error()})
		}
	}()
}

// CancelMaintenance ends the background maintenance pass, if any; the pass
// reports maintenance:cancelled once it has stopped.
func (a *App) CancelMaintenance() {
	for _, t := range a.jobs.List() {
		if t.Kind == jobs.Maintenance && !t.State.Finished() {
			_ = a.jobs.Cancel(t.ID)
		}
	}
	if a.db != nil {
		a.db.CancelMaintenance()
	}
}

// matchStatsFillReport shows the background match_stats fill the open starts
// (Database.FillMatchStatsInBackground) as the stats.matches maintenance line
// running: the same chip, the same cancel, the same end events, which the
// panels reading match_stats wait for.
func (a *App) matchStatsFillReport() *database.MatchStatsFillReport {
	kind := string(maintenance.StatsMatches)
	return &database.MatchStatsFillReport{
		// Each fill is its own task, ended on every path; its cancel stops
		// that fill alone.
		Begin: func(stop func()) (func(done, total int), func(error)) {
			task := a.jobs.Start(jobs.Spec{
				Kind: jobs.MatchStatsFill, Label: kind, Library: a.shownGroup(), Priority: jobs.Background,
				Cancel: stop,
			})
			return task.Progress, task.Finish
		},
		Progress: func(done, total int) {
			a.emitBatch("maintenance:progress", map[string]any{"kind": kind, "done": done, "total": total})
		},
		End: func(filled int, err error) {
			switch {
			case err == nil:
				a.emitBatch("maintenance:done", maintenance.Run{Kind: maintenance.StatsMatches, Changed: filled, Outcome: maintenance.OutcomeDone})
			case errors.Is(err, context.Canceled):
				a.emitBatch("maintenance:cancelled", map[string]string{"kind": kind})
			default:
				a.emitBatch("maintenance:error", map[string]string{"kind": kind, "message": err.Error()})
			}
		},
	}
}
