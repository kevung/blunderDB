package gui

import (
	"context"
	"errors"

	"github.com/kevung/blunderdb/pkg/blunderdb/maintenance"
)

// StartMaintenance runs one line of the Maintenance tab in the background and
// returns at once: a pass over a large library takes minutes, and a bound
// method blocks the Wails call that made it. It reports on
// maintenance:progress, then exactly one of maintenance:done (the Run),
// maintenance:cancelled or maintenance:error ({message}); a refusal (library
// busy, read-only, unknown line) is an error event too, so the frontend reads
// every outcome in one place. CancelMaintenance ends the pass between batches.
func (a *App) StartMaintenance(kind string) {
	if a.db == nil {
		return
	}
	if err := a.db.RefuseReadOnly(); err != nil {
		a.emitBatch("maintenance:error", map[string]string{"kind": kind, "message": err.Error()})
		return
	}
	go func() {
		defer recoverBackground(a.ctx, "maintenance")
		run, err := a.db.RunMaintenanceWithProgress(kind, func(done, total int) {
			a.emitBatch("maintenance:progress", map[string]any{"kind": kind, "done": done, "total": total})
		})
		switch {
		case err == nil:
			a.emitBatch("maintenance:done", run)
		case run.Outcome == maintenance.OutcomeCancelled || errors.Is(err, context.Canceled):
			a.emitBatch("maintenance:cancelled", map[string]string{"kind": kind})
		default:
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
