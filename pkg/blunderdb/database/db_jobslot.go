package database

import (
	"context"
	"errors"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The jobs that write the library one at a time, as the daemon holds one
// slot per tenant (ADR-0088). The code travels in BusyError's message, which
// the frontend matches to name the job in the user's language.
const (
	JobMaintenance = "maintenance"
	JobGammonNet   = "gammonnet"
	JobRollout     = "rollout"
)

// ErrDatabaseBusy matches every BusyError under errors.Is.
var ErrDatabaseBusy = errors.New("database is busy")

// BusyError refuses a job asked for while Job holds the library. It is also
// storage.ErrConflict, so the daemon and the CLI report it as a conflict.
type BusyError struct{ Job string }

func (e *BusyError) Error() string { return "database is busy: " + e.Job }

func (e *BusyError) Is(target error) bool {
	return target == ErrDatabaseBusy || target == storage.ErrConflict
}

type waitKey struct{}

// WaitForDatabase marks ctx's job as one that queues for the library instead
// of being refused: an automatic batch the user did not ask for at that
// moment. onWait (nil for none) is told which job it waits behind, once.
// Only cancelling ctx ends the wait.
func WaitForDatabase(ctx context.Context, onWait func(job string)) context.Context {
	if onWait == nil {
		onWait = func(string) {}
	}
	return context.WithValue(ctx, waitKey{}, onWait)
}

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

// acquireJob takes the slot for job (one of the Job codes) running under
// ctx. While another job runs it returns a *BusyError, or waits for it when
// ctx was marked by WaitForDatabase. release must be called once the job
// has ended.
func (d *Database) acquireJob(ctx context.Context, job string, cancel context.CancelFunc) (func(), error) {
	onWait, queued := ctx.Value(waitKey{}).(func(string))
	told := false
	for {
		d.cancelMu.Lock()
		held := d.job
		if held == nil {
			j := &jobSlot{name: job, ctx: ctx, cancel: cancel, done: make(chan struct{})}
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
			if !queued {
				return nil, &BusyError{Job: held.name}
			}
			if !told {
				told = true
				onWait(held.name)
			}
		}
		select {
		case <-held.done:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
