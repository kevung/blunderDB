package database

import (
	"context"
	"errors"
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The jobs that write the library one at a time, as the daemon holds one
// slot per tenant (ADR-0088). The code travels in BusyError's message, which
// the frontend matches to name the job in the user's language.
const (
	JobMaintenance = "maintenance"
	JobGammonNet   = "gammonnet"
	JobRollout     = "rollout"
	// JobExchange is the desktop swapping the open library between two
	// handles (Exchange): short, but no job may start across it.
	JobExchange = "exchange"
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
	// by is the handle the job runs on: an exchange is refused only for a
	// job on one of its two handles, not for one on a borrowed handle.
	by *Database
}

// jobGate holds the one job writing a library. Every handle on the library
// shares it, the one carrying the library and those Borrow lent to a job, so
// a job on a borrowed handle and one on the carrier exclude each other as two
// jobs on one handle do. Exchange moves it with the library.
type jobGate struct {
	mu  sync.Mutex
	job *jobSlot
}

// gateLocked returns d's job gate, made on first use; the caller holds cancelMu.
func (d *Database) gateLocked() *jobGate {
	if d.gate == nil {
		d.gate = &jobGate{}
	}
	return d.gate
}

// currentJob returns the job holding d's library, nil when idle.
func (d *Database) currentJob() *jobSlot {
	d.cancelMu.Lock()
	g := d.gateLocked()
	d.cancelMu.Unlock()
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.job
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
		if d.exchanging {
			d.cancelMu.Unlock()
			return nil, &BusyError{Job: JobExchange}
		}
		g := d.gateLocked()
		g.mu.Lock()
		held := g.job
		if held == nil {
			j := &jobSlot{name: job, ctx: ctx, cancel: cancel, done: make(chan struct{}), by: d}
			g.job = j
			g.mu.Unlock()
			d.cancelMu.Unlock()
			return func() {
				g.mu.Lock()
				if g.job == j {
					g.job = nil
				}
				g.mu.Unlock()
				close(j.done)
			}, nil
		}
		g.mu.Unlock()
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
