package gui

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

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
	db := a.db
	// The database cancels the pass holding its slot: a pass still waiting
	// in the queue is stopped by the registry alone, never one ahead of it.
	var started atomic.Bool
	task := a.jobs.Start(jobs.Spec{
		Kind: jobs.Maintenance, Label: kind, Library: a.shownGroup(), Priority: jobs.Background,
		Exclusive: true, Cancel: func() {
			if started.Load() {
				db.CancelMaintenance()
			}
		},
	})
	go func() {
		defer recoverBackground(a.ctx, "maintenance")
		if err := task.Turn(a.batchCtx()); err != nil {
			task.Finish(err)
			a.emitBatch("maintenance:cancelled", map[string]string{"kind": kind})
			return
		}
		started.Store(true)
		run, err := db.RunMaintenanceWithProgress(kind, func(done, total int) {
			task.Progress(done, total)
			a.emitBatch("maintenance:progress", map[string]any{"kind": kind, "done": done, "total": total})
		})
		switch {
		case err == nil:
			task.Finish(nil)
			a.emitBatch("maintenance:done", run)
		case run.Outcome == maintenance.OutcomeCancelled || errors.Is(err, context.Canceled):
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
	// The fill tells nothing at its start: its task is registered at its
	// first report and ended at End.
	var mu sync.Mutex
	var task *jobs.Job
	taskOf := func() *jobs.Job {
		mu.Lock()
		defer mu.Unlock()
		if task == nil {
			task = a.jobs.Start(jobs.Spec{
				Kind: jobs.MatchStatsFill, Label: kind, Library: a.shownGroup(), Priority: jobs.Background,
				Cancel: a.CancelMaintenance,
			})
		}
		return task
	}
	return &database.MatchStatsFillReport{
		Progress: func(done, total int) {
			taskOf().Progress(done, total)
			a.emitBatch("maintenance:progress", map[string]any{"kind": kind, "done": done, "total": total})
		},
		End: func(filled int, err error) {
			taskOf().Finish(err)
			mu.Lock()
			task = nil
			mu.Unlock()
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
