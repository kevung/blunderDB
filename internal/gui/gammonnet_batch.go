package gui

import (
	"context"
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
	ctx, cancel := context.WithCancel(a.ctx)
	a.gnBatchCancel = cancel
	a.gnBatchMu.Unlock()

	go func() {
		defer recoverBackground(a.ctx, "gammonNet batch analysis")
		summary, err := run(ctx, func(done, total int) {
			runtime.EventsEmit(a.ctx, "gammonnet-batch:progress", map[string]int{"done": done, "total": total})
		})

		a.gnBatchMu.Lock()
		a.gnBatchCancel = nil
		a.gnBatchMu.Unlock()

		if err != nil {
			if ctx.Err() != nil {
				runtime.EventsEmit(a.ctx, "gammonnet-batch:cancelled")
				return
			}
			runtime.EventsEmit(a.ctx, "gammonnet-batch:error", map[string]string{"message": err.Error()})
			return
		}
		// The three-way split (#191): the toast this event feeds tells the
		// user how many positions actually got a new analysis, how many
		// were legitimately out of gammonNet's reach (a dance, a match
		// score beyond the MET), and how many failed and will be retried
		// next time — never a bare "Done." that can't distinguish those.
		runtime.EventsEmit(a.ctx, "gammonnet-batch:done", map[string]int{
			"evaluated": summary.Evaluated,
			"refused":   summary.Refused,
			"failed":    summary.Failed,
		})
	}()
}

// CancelGammonNetBatch aborts an in-flight batch, if any.
func (a *App) CancelGammonNetBatch() {
	a.gnBatchMu.Lock()
	if a.gnBatchCancel != nil {
		a.gnBatchCancel()
		a.gnBatchCancel = nil
	}
	a.gnBatchMu.Unlock()
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
