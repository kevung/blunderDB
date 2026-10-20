package database

import (
	"context"
	"log/slog"
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The status bar reads the estimated counts only (GetDatabaseStatsEstimate),
// and on a large library the estimate does not count the blunders: it serves
// the cached count, which every write that can change it deletes. Without a
// recount the counter would show "?" from the first import on. So when the
// cache is missing, a recount starts in the background, filling the cache, and
// the desktop is told when the estimate knows the blunders again.

// maxRecountRounds bounds the rounds of one recount: a library written to
// without pause would otherwise be scanned forever.
const maxRecountRounds = 5

// blunderRecount is the background recount in flight, guarded by its own
// mutex: bumpGeneration cancels it while holding mu.
type blunderRecount struct {
	mu      sync.Mutex
	running *recountRun
	// ready is told once a recount has filled the cache; the desktop
	// re-reads its counter. nil outside the GUI.
	ready func()
	// afterRound, nil outside tests, runs after each count, before the
	// recount checks that the cache holds it.
	afterRound func()
}

type recountRun struct {
	gen    uint64
	cancel context.CancelFunc
	// again is raised when the cache was found missing while this run
	// counted: a write landed during the count, whose result it may not have
	// been allowed to cache, so the run counts once more.
	again bool
}

// SetBlunderCountReady registers fn, called once a background recount has
// filled the blunder count cache. A function, not a method, so the Wails
// binding of *Database does not grow.
func SetBlunderCountReady(d *Database, fn func()) {
	d.recount.mu.Lock()
	defer d.recount.mu.Unlock()
	d.recount.ready = fn
}

// blunderCounter is the store's blunder count alone (sqlshared.MetadataStore);
// a store without it is recounted through Counts.
type blunderCounter interface {
	BlunderCount(ctx context.Context, scope string) (int, error)
}

// startBlunderRecount counts the blunders in the background, or asks the run
// already counting this file for one more round. The caller holds mu (read or
// write). The count runs outside mu: it reads for seconds on a large library,
// and a writer waiting for mu would stall every reader behind it.
func (d *Database) startBlunderRecount() {
	if d.store == nil || d.readOnly {
		// Read-only, the count could not be cached: every refresh would
		// pay a full scan for nothing.
		return
	}
	d.recount.mu.Lock()
	defer d.recount.mu.Unlock()
	if r := d.recount.running; r != nil && r.gen == d.generation {
		r.again = true
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	run := &recountRun{gen: d.generation, cancel: cancel}
	d.recount.running = run
	go d.recountBlunders(ctx, run, d.store.Metadata())
}

// recountBlunders is the run itself: it counts, and counts again while a
// write that landed during the count kept the cache empty.
func (d *Database) recountBlunders(ctx context.Context, run *recountRun, md storage.MetadataStore) {
	defer run.cancel()
	count := func() error {
		if bc, ok := md.(blunderCounter); ok {
			_, err := bc.BlunderCount(ctx, "")
			return err
		}
		_, err := md.Counts(ctx, "")
		return err
	}
	for round := 1; ; round++ {
		d.recount.mu.Lock()
		run.again = false
		hook := d.recount.afterRound
		d.recount.mu.Unlock()
		known := false
		if err := count(); err != nil {
			if ctx.Err() == nil {
				slog.Warn("background blunder recount failed", "err", err)
			}
		} else {
			if hook != nil {
				hook()
			}
			// The count is cached unless a write landed while it ran.
			est, err := md.EstimatedCounts(ctx, "", EstimateExactBelow)
			known = err == nil && est.BlundersKnown
		}
		// Deciding to stop and leaving the slot happen under one lock, so a
		// request that finds the run still there is never lost: it raised
		// again before this check, or it finds the slot empty and starts
		// its own run.
		d.recount.mu.Lock()
		if !known && run.again && ctx.Err() == nil && round < maxRecountRounds {
			d.recount.mu.Unlock()
			continue
		}
		if d.recount.running == run {
			d.recount.running = nil
		}
		ready := d.recount.ready
		d.recount.mu.Unlock()
		if known && ctx.Err() == nil && ready != nil {
			ready()
		}
		return
	}
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
