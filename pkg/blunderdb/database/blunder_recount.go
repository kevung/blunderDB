package database

import (
	"context"
	"log/slog"
	"sync"
)

// The status bar reads the estimated counts only (GetDatabaseStatsEstimate),
// and on a large library the estimate does not count the blunders: it serves
// the cached count, which every write that can change it deletes. Without a
// recount the counter would show "?" from the first import on. So when the
// cache is missing, a recount starts in the background, filling the cache, and
// the desktop is told when the estimate knows the blunders again.

// blunderRecount is the background recount in flight, guarded by its own
// mutex: bumpGeneration cancels it while holding mu.
type blunderRecount struct {
	mu      sync.Mutex
	running *recountRun
	// ready is told once a recount has filled the cache; the desktop
	// re-reads its counter. nil outside the GUI.
	ready func()
}

type recountRun struct {
	gen    uint64
	cancel context.CancelFunc
}

// SetBlunderCountReady registers fn, called once a background recount has
// filled the blunder count cache. A function, not a method, so the Wails
// binding of *Database does not grow.
func SetBlunderCountReady(d *Database, fn func()) {
	d.recount.mu.Lock()
	defer d.recount.mu.Unlock()
	d.recount.ready = fn
}

// startBlunderRecount counts the blunders in the background unless a count of
// this file is already running. The caller holds mu (read or write). The
// count runs outside mu: it reads for seconds on a large library, and a
// writer waiting for mu would stall every reader behind it.
func (d *Database) startBlunderRecount() {
	if d.store == nil || d.readOnly {
		// Read-only, the count could not be cached: every refresh would
		// pay a full scan for nothing.
		return
	}
	d.recount.mu.Lock()
	defer d.recount.mu.Unlock()
	if r := d.recount.running; r != nil && r.gen == d.generation {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	run := &recountRun{gen: d.generation, cancel: cancel}
	d.recount.running = run
	store := d.store
	go func() {
		defer cancel()
		_, err := store.Metadata().Counts(ctx, "")
		var known bool
		if err == nil {
			// The count is cached unless a write landed while it ran; only
			// a cached count is worth telling the desktop about.
			est, estErr := store.Metadata().EstimatedCounts(ctx, "", EstimateExactBelow)
			known = estErr == nil && est.BlundersKnown
		} else if ctx.Err() == nil {
			slog.Warn("background blunder recount failed", "err", err)
		}
		d.recount.mu.Lock()
		if d.recount.running == run {
			d.recount.running = nil
		}
		ready := d.recount.ready
		d.recount.mu.Unlock()
		if known && ctx.Err() == nil && ready != nil {
			ready()
		}
	}()
}

// cancelBlunderRecount stops the recount of the file about to change.
func (d *Database) cancelBlunderRecount() {
	d.recount.mu.Lock()
	defer d.recount.mu.Unlock()
	if r := d.recount.running; r != nil {
		r.cancel()
		d.recount.running = nil
	}
}
