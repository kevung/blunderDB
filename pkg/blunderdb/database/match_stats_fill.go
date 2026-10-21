package database

import (
	"context"
	"errors"
	"log/slog"
	"sync"
)

// A library whose match_stats table lacks rows (a migration that wiped it, a
// file written by an older blunderDB) would have its first tournament or
// player read fill the whole table, minutes on a large library, while the
// panel waits with nothing to show. The desktop instead fills it in the
// background from the open on, a lot at a time (fillMatchStatsLots), and
// reports the progress as the stats.matches maintenance line does. The CLI
// and the daemon keep the synchronous path: the CLI's reads fill before they
// answer (prefillMatchStats), and the daemon's storage readers fill what
// they read.

// matchStatsFill is the background fill in flight, guarded by its own mutex:
// bumpGeneration cancels it while holding Database.mu.
type matchStatsFill struct {
	mu      sync.Mutex
	running *matchStatsFillRun
	// report is told the fill's progress and its end; nil outside the
	// desktop, which then never fills in the background.
	report *MatchStatsFillReport
}

type matchStatsFillRun struct {
	gen    uint64
	cancel context.CancelFunc
	done   chan struct{}
	// byUser is raised when CancelMaintenance ended the run, which the
	// desktop reports; a run ended by the file changing is reported to no
	// one, since the screen already shows the next library.
	byUser bool
}

// MatchStatsFillReport receives a background fill's events. Progress gets the
// matches filled so far out of those missing at the start; End gets how many
// were filled and the error that stopped the run (nil once complete,
// context.Canceled when the user cancelled it).
type MatchStatsFillReport struct {
	Progress func(done, total int)
	End      func(filled int, err error)
}

// SetMatchStatsFillReport registers the desktop's report of the background
// fill and so enables it. A function, not a method, so the Wails binding of
// *Database does not grow.
func SetMatchStatsFillReport(d *Database, r *MatchStatsFillReport) {
	d.statsFill.mu.Lock()
	defer d.statsFill.mu.Unlock()
	d.statsFill.report = r
}

// FillMatchStatsInBackground starts filling the open library's missing
// match_stats rows in the background and returns whether a fill now runs for
// it. It starts nothing on a read-only library (ADR-0007), when no report is
// registered, or when no row is missing; a fill already running for this
// file goes on.
func (d *Database) FillMatchStatsInBackground() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil || d.store == nil || d.readOnly {
		return false
	}
	d.statsFill.mu.Lock()
	defer d.statsFill.mu.Unlock()
	if d.statsFill.report == nil {
		return false
	}
	if r := d.statsFill.running; r != nil && r.gen == d.generation {
		return true
	}
	// The count reads one index; the fill itself runs outside this hold.
	missing, err := d.store.Stats().MissingMatchStats(context.Background(), "")
	if err != nil || len(missing) == 0 {
		return false
	}
	ctx, cancel := context.WithCancel(context.Background())
	run := &matchStatsFillRun{gen: d.generation, cancel: cancel, done: make(chan struct{})}
	d.statsFill.running = run
	go d.runMatchStatsFill(ctx, run, len(missing), *d.statsFill.report)
	return true
}

func (d *Database) runMatchStatsFill(ctx context.Context, run *matchStatsFillRun, total int, report MatchStatsFillReport) {
	defer close(run.done)
	defer run.cancel()
	filled := 0
	// The caller holds mu for reading, and keep is asked under that hold.
	keep := func() bool { return d.generation == run.gen && !d.readOnly }
	err := d.fillMatchStatsLots(ctx, keep, func(missing int) {
		// A lot can drop rows filled earlier (shared positions): the count
		// shown never goes back.
		filled = max(filled, min(total-missing, total))
		if report.Progress != nil {
			report.Progress(filled, total)
		}
	})
	d.statsFill.mu.Lock()
	byUser := run.byUser
	if d.statsFill.running == run {
		d.statsFill.running = nil
	}
	d.statsFill.mu.Unlock()
	switch {
	case err == nil:
	case byUser:
		err = context.Canceled
	case errors.Is(err, ErrDatabaseChanged), ctx.Err() != nil:
		return
	default:
		slog.Warn("background match statistics fill failed", "err", err)
	}
	if report.End != nil {
		report.End(filled, err)
	}
}

// MatchStatsFilling reports whether the background fill runs for the open
// library.
func (d *Database) MatchStatsFilling() bool {
	d.statsFill.mu.Lock()
	defer d.statsFill.mu.Unlock()
	return d.statsFill.running != nil
}

// cancelMatchStatsFill stops the background fill; byUser says whether the
// user asked for it, which the desktop then reports.
func (d *Database) cancelMatchStatsFill(byUser bool) <-chan struct{} {
	d.statsFill.mu.Lock()
	defer d.statsFill.mu.Unlock()
	r := d.statsFill.running
	if r == nil {
		return nil
	}
	r.byUser = byUser
	r.cancel()
	if !byUser {
		d.statsFill.running = nil
	}
	return r.done
}
