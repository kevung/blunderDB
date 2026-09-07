package gui

import (
	"context"
	"log/slog"
	goruntime "runtime"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
)

// The gammonNet batch (#129, ADR-0013): a bounded, visible, cancellable,
// resumable sweep that writes an analysis for every position that has none —
// or, with StartGammonNetStaleBatch (#191), re-runs gammonNet on every
// position whose stored analysis is entirely its own but stale. The actual
// query/evaluate/write loops live on *database.Database
// (AnalyzeMissingWithGammonNet/AnalyzeStaleGammonNet,
// database/db_gammonnet_batch.go) so the CLI (#130) and the serve daemon
// (#191) can reuse them without a GUI dependency; this file is the same
// goroutine+context+mutex+EventsEmit shell DownloadBearoffDB (bearoff.go)
// already established, wired to the batch's own events:
//
//	gammonnet-batch:progress  {done, total}
//	gammonnet-batch:done      {evaluated, refused, failed}
//	gammonnet-batch:cancelled {}
//	gammonnet-batch:error     {message}
//
// gnBatchMu/gnBatchCancel live on App (#196/C.9) — see app.go's own comment.

// gnYieldPoll is how often the batch rechecks whether the interactive live
// evaluation (gammonnet_eval.go) has freed up, once it finds it busy. Small
// enough that the batch resumes promptly once the interactive search
// finishes (ADR-0011 bounds that at well under a second on the display
// depth), large enough not to spin.
const gnYieldPoll = 50 * time.Millisecond

// StartGammonNetBatch analyses every position without an analysis, in the
// background. A batch already in flight (of either kind — this one or
// StartGammonNetStaleBatch) is cancelled first — one at a time, like
// DownloadBearoffDB. No-op (emits nothing) when db is nil, which only
// happens in tests that construct an App without one.
func (a *App) StartGammonNetBatch(ply, pruneK, candidates int) {
	a.runGammonNetBatch(func(ctx context.Context, onProgress func(done, total int)) (database.GammonNetBatchSummary, error) {
		// a.effectiveBatchJobs(): every core (#147), UNLESS an interactive
		// evaluation is already in flight when this batch starts (#196/C.9)
		// — the yield below already keeps it that way position by position,
		// this only shortens the transient window before a goroutine
		// already mid-position notices.
		return a.db.AnalyzeMissingWithGammonNet(ctx, ply, pruneK, candidates, a.effectiveBatchJobs(),
			a.waitForInteractiveEvaluation, onProgress)
	})
}

// StartGammonNetStaleBatch re-runs gammonNet on every position whose stored
// analysis is entirely its own but stale at ply — a different EngineVersion
// than the running build's, or a different AnalysisDepth than ply now asks
// for (#191). The config modal's "re-analyse stale positions" button; before
// this there was no way to trigger AnalyzeStaleGammonNet at all — an
// EngineVersion bump (ADR-0016, ADR-0022, ADR-0023 each moved it) left every
// already-analysed position looking perfectly current with nothing to
// re-run it.
func (a *App) StartGammonNetStaleBatch(ply, pruneK, candidates int) {
	a.runGammonNetBatch(func(ctx context.Context, onProgress func(done, total int)) (database.GammonNetBatchSummary, error) {
		return a.db.AnalyzeStaleGammonNet(ctx, ply, pruneK, candidates, a.effectiveBatchJobs(),
			a.waitForInteractiveEvaluation, onProgress)
	})
}

// StartGammonNetMatchBatch analyses the positions of ONE match that have no
// analysis, in the background — the batch a transcription's save starts
// (ADR-0045 §8). Same shell as StartGammonNetBatch, same events, same
// single-batch-at-a-time rule; only the scope is narrower, so a correction
// re-saved does not sweep the whole library again, and — ADR-0013 filling
// gaps — analyses only the positions the correction actually created.
//
// The depth is the caller's: the transcription panel passes the canonical
// 2-ply, k=12, which is what an analysis written to the library must be
// (ADR-0013 names display depth and analysis depth as two settings). This
// function fixes neither.
func (a *App) StartGammonNetMatchBatch(matchID int64, ply, pruneK, candidates int) {
	a.runGammonNetBatch(func(ctx context.Context, onProgress func(done, total int)) (database.GammonNetBatchSummary, error) {
		return a.db.AnalyzeMatchWithGammonNet(ctx, matchID, ply, pruneK, candidates, a.effectiveBatchJobs(),
			a.waitForInteractiveEvaluation, onProgress)
	})
}

// effectiveBatchJobs is the batch's own #196/C.9 fix: goruntime.NumCPU()
// batch goroutines PLUS the panel's own WithWorkers(NumCPU) pool is
// 2×NumCPU worth of live goroutines competing for NumCPU cores whenever an
// interactive "at rest" evaluation (gammonnet_eval.go) is already running
// when a batch starts. waitForInteractiveEvaluation already makes every
// batch goroutine yield fully once it notices — this only shortens the
// transient window before it does: fewer goroutines means fewer stragglers
// still mid-position, still racing the panel's pool for cores, when the
// interactive search began.
func (a *App) effectiveBatchJobs() int {
	a.gnEvalMu.Lock()
	busy := a.gnEvalCancel != nil
	a.gnEvalMu.Unlock()
	if !busy {
		return goruntime.NumCPU()
	}
	if n := goruntime.NumCPU() / 4; n > 1 {
		return n
	}
	return 1
}

// runGammonNetBatch is the goroutine+context+mutex+EventsEmit shell both
// StartGammonNetBatch and StartGammonNetStaleBatch share: run does the
// query/evaluate/write work and returns the batch's evaluated/refused/failed
// summary (#191), this function only owns the single-in-flight bookkeeping
// and the events.
func (a *App) runGammonNetBatch(run func(ctx context.Context, onProgress func(done, total int)) (database.GammonNetBatchSummary, error)) {
	if a.db == nil {
		return
	}

	a.gnBatchMu.Lock()
	if a.gnBatchCancel != nil {
		a.gnBatchCancel()
	}
	ctx, cancel := context.WithCancel(a.batchCtx())
	stopped := make(chan struct{})
	a.gnBatchCancel = cancel
	a.gnBatchDone = stopped
	a.gnBatchMu.Unlock()

	go func() {
		// Registered first, so it runs LAST — after recoverBackground has
		// swallowed a panic, and whatever happens: a shutdown waiting on
		// this channel must never be the thing that hangs.
		defer close(stopped)
		defer recoverBackground(a.ctx, "gammonNet batch analysis")
		summary, err := run(ctx, func(done, total int) {
			a.emitBatch("gammonnet-batch:progress", map[string]int{"done": done, "total": total})
		})

		// Only if this goroutine is still the batch in flight: the cancel
		// above lets a NEW batch replace this one while it is finishing its
		// last position, and clearing unconditionally would leave that new
		// batch uncancellable.
		a.gnBatchMu.Lock()
		if a.gnBatchDone == stopped {
			a.gnBatchCancel = nil
			a.gnBatchDone = nil
		}
		a.gnBatchMu.Unlock()

		if err != nil {
			if ctx.Err() != nil {
				a.emitBatch("gammonnet-batch:cancelled")
				return
			}
			a.emitBatch("gammonnet-batch:error", map[string]string{"message": err.Error()})
			return
		}
		// The three-way split (#191): the toast this event feeds tells the
		// user how many positions actually got a new analysis, how many
		// were legitimately out of gammonNet's reach (a dance, a match
		// score beyond the MET), and how many failed and will be retried
		// next time — never a bare "Done." that can't distinguish those.
		a.emitBatch("gammonnet-batch:done", map[string]int{
			"evaluated": summary.Evaluated,
			"refused":   summary.Refused,
			"failed":    summary.Failed,
		})
	}()
}

// batchCtx is the context the batch's own is derived from: the Wails
// lifecycle context when a window exists, a background one when none does.
// Without this the shell could not run at all before startup has set a.ctx —
// context.WithCancel(nil) panics — which is the same "the generation starts
// before the window does" case bearoff.go already handles.
func (a *App) batchCtx() context.Context {
	if a.ctx == nil {
		return context.Background()
	}
	return a.ctx
}

// emitBatch is emitBearoff for the batch's events, and for the same reason
// plus a sharper one: given a context that is not the Wails lifecycle one,
// EventsEmit does not return an error, it ENDS THE PROCESS. So an App
// without a window emits nothing rather than taking the process down with
// it (ADR-0004: a host capability is detected, never assumed).
func (a *App) emitBatch(name string, payload ...any) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, name, payload...)
}

// CancelGammonNetBatch aborts an in-flight batch, if any. It returns at
// once: it is a bound method, and the frontend's "stop" button must not wait
// for the positions still mid-search. Whoever needs the batch to have
// actually STOPPED — shutdown does, it closes the database next — calls
// cancelGammonNetBatch below and waits on the channel it hands back.
func (a *App) CancelGammonNetBatch() {
	a.cancelGammonNetBatch()
}

// cancelGammonNetBatch is CancelGammonNetBatch with the batch's "stopped"
// channel — closed once the batch goroutine has really finished, nil when
// there was nothing in flight. Kept unexported so the bound method above
// stays a plain fire-and-forget one for the frontend (a bound method
// returning a channel is not bindable anyway).
func (a *App) cancelGammonNetBatch() <-chan struct{} {
	a.gnBatchMu.Lock()
	defer a.gnBatchMu.Unlock()
	if a.gnBatchCancel != nil {
		a.gnBatchCancel()
		a.gnBatchCancel = nil
	}
	return a.gnBatchDone
}

// waitForInteractiveEvaluation is the batch's yield point (#129): called by
// every batch goroutine before every position, it blocks for as long as
// gammonnet_eval.go's live evaluation (#125) has a search in flight, so an
// editing user is never fighting the batch for cores. With the batch spread
// over jobs goroutines (#147, effectiveBatchJobs #196/C.9) the promise is
// "the batch gives way within at most jobs positions" rather than one —
// every goroutine passes through here, so the whole batch stalls, just not
// on the same position.
func (a *App) waitForInteractiveEvaluation() {
	for {
		a.gnEvalMu.Lock()
		busy := a.gnEvalCancel != nil
		a.gnEvalMu.Unlock()
		if !busy {
			return
		}
		time.Sleep(gnYieldPoll)
	}
}

// stopBackgroundJobs cancels every long-running job this App owns and waits
// for the one that writes to the database to have actually stopped. It is
// shutdown's first step (run.go), and it is deliberately about ALL the jobs,
// not only the batch this file owns: whatever is still running when the user
// quits is running against a database that is about to close.
//
// Only the gammonNet batch is waited on. The bearoff generation writes files
// of its own, next to the tables, and never touches the database — cancelling
// it is enough, and its last block can take long enough that waiting for it
// would be indistinguishable from a hang.
//
// grace bounds the wait: exceeded, it logs and returns, because a window that
// will not close is a worse failure than the write it was avoiding.
func (a *App) stopBackgroundJobs(grace time.Duration) {
	a.CancelBearoffGeneration()

	stopped := a.cancelGammonNetBatch()
	if stopped == nil {
		return
	}
	select {
	case <-stopped:
	case <-time.After(grace):
		slog.Warn("shutdown: the gammonNet batch did not stop within the grace period; closing the database anyway", "grace", grace)
	}
}
