package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ErrDatabaseBusy is a job that writes the library — a maintenance run, a
// gammonNet batch, a stored rollout — asked for while another one runs: one
// at a time, as the daemon holds one slot per tenant (ADR-0088).
var ErrDatabaseBusy = fmt.Errorf("another job is writing the database: %w", storage.ErrConflict)

// jobSlot is the job holding the library. ctx is the holder's: once it is
// done the holder is only finishing, and a newcomer waits for done rather
// than being refused — the GUI replaces a batch by cancelling it and
// starting the next at once.
type jobSlot struct {
	name   string
	ctx    context.Context
	cancel context.CancelFunc // set for a maintenance run, which CancelMaintenance ends
	done   chan struct{}
}

// acquireJob takes the slot for a job named name running under ctx, or
// returns ErrDatabaseBusy. release must be called once the job has ended.
func (d *Database) acquireJob(ctx context.Context, name string, cancel context.CancelFunc) (func(), error) {
	for {
		d.cancelMu.Lock()
		held := d.job
		if held == nil {
			j := &jobSlot{name: name, ctx: ctx, cancel: cancel, done: make(chan struct{})}
			d.job = j
			d.cancelMu.Unlock()
			return func() {
				d.cancelMu.Lock()
				if d.job == j {
					d.job = nil
				}
				d.cancelMu.Unlock()
				close(j.done)
			}, nil
		}
		d.cancelMu.Unlock()
		if held.ctx.Err() == nil {
			return nil, fmt.Errorf("%w (%s is running)", ErrDatabaseBusy, held.name)
		}
		select {
		case <-held.done:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
