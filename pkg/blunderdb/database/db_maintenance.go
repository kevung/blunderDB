package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/maintenance"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ErrMaintenanceBusy is a second maintenance run while one is in flight:
// one operation at a time (ADR-0088).
var ErrMaintenanceBusy = fmt.Errorf("a maintenance operation is already running: %w", storage.ErrConflict)

// openStore returns the backend under a read lock, or an error when no
// database is open. The maintenance calls then run without d.mu: the counts
// are reads of the backend's own pool, and a run takes the lock per batch
// through maintenance.Opts.Before.
func (d *Database) openStore(what string) (storage.Storage, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil || d.store == nil {
		return nil, fmt.Errorf("%s: no database open", what)
	}
	return d.store, nil
}

// MaintenanceStatus returns the library's state of health: every line, its
// free count and its last run (ADR-0088). It writes nothing.
func (d *Database) MaintenanceStatus() ([]maintenance.Line, error) {
	st, err := d.openStore("maintenance status")
	if err != nil {
		return nil, err
	}
	return maintenance.Status(context.Background(), st, "")
}

// CountMaintenance computes one line's count, whatever its cost.
func (d *Database) CountMaintenance(kind string) (int64, error) {
	st, err := d.openStore("maintenance count")
	if err != nil {
		return 0, err
	}
	return maintenance.Count(context.Background(), st, "", maintenance.Kind(kind))
}

// MaintenanceIDs lists the ids one line counts, in ascending order.
func (d *Database) MaintenanceIDs(kind string) ([]int64, error) {
	st, err := d.openStore("maintenance ids")
	if err != nil {
		return nil, err
	}
	return maintenance.IDs(context.Background(), st, "", maintenance.Kind(kind))
}

// RunMaintenance performs one line's action and records it. It refuses a
// second run while one is in flight; CancelMaintenance ends the current one
// between two batches.
func (d *Database) RunMaintenance(kind string) (maintenance.Run, error) {
	return d.RunMaintenanceWithProgress(kind, nil)
}

// RunMaintenanceWithProgress is RunMaintenance with progress reported to fn
// (nil for none); the CLI and the desktop's status bar print it.
func (d *Database) RunMaintenanceWithProgress(kind string, fn func(done, total int)) (maintenance.Run, error) {
	st, err := d.openStore("maintenance run")
	if err != nil {
		return maintenance.Run{}, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.cancelMu.Lock()
	if d.maintenanceCancel != nil {
		d.cancelMu.Unlock()
		return maintenance.Run{}, ErrMaintenanceBusy
	}
	d.maintenanceCancel = cancel
	d.cancelMu.Unlock()
	defer func() {
		d.cancelMu.Lock()
		d.maintenanceCancel = nil
		d.cancelMu.Unlock()
	}()
	return maintenance.RunKind(ctx, st, "", maintenance.Kind(kind), maintenance.Opts{
		Before: func() func() {
			d.mu.Lock()
			return d.mu.Unlock
		},
		Progress: fn,
	})
}

// CancelMaintenance ends the in-flight maintenance run; no-op when idle.
func (d *Database) CancelMaintenance() {
	d.cancelMu.Lock()
	cancel := d.maintenanceCancel
	d.cancelMu.Unlock()
	if cancel != nil {
		cancel()
	}
}
