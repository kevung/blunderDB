package gui

import (
	"context"
	"log/slog"
	goruntime "runtime"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
)

// The gammonNet batch (ADR-0013): the loops live on *database.Database, shared
// with the CLI and daemon; this file is the GUI shell and its events:
//
//	gammonnet-batch:progress  {done, total}
//	gammonnet-batch:done      {evaluated, refused, failed}
//	gammonnet-batch:cancelled {}
//	gammonnet-batch:error     {message}

// gnYieldPoll is how often a yielding batch rechecks the live evaluation.
const gnYieldPoll = 50 * time.Millisecond

// StartGammonNetBatch analyses every position without an analysis, in the
// background, cancelling any batch in flight. No-op when db is nil.
func (a *App) StartGammonNetBatch(ply, pruneK, candidates int) {
	a.runGammonNetBatch(func(ctx context.Context, onProgress func(done, total int)) (database.GammonNetBatchSummary, error) {
		return a.db.AnalyzeMissingWithGammonNet(ctx, ply, pruneK, candidates, a.effectiveBatchJobs(),
			a.waitForInteractiveEvaluation, onProgress)
	})
}

// StartGammonNetStaleBatch re-runs gammonNet on every position whose analysis
// is entirely its own but stale: another EngineVersion, or another depth.
func (a *App) StartGammonNetStaleBatch(ply, pruneK, candidates int) {
	a.runGammonNetBatch(func(ctx context.Context, onProgress func(done, total int)) (database.GammonNetBatchSummary, error) {
		return a.db.AnalyzeStaleGammonNet(ctx, ply, pruneK, candidates, a.effectiveBatchJobs(),
			a.waitForInteractiveEvaluation, onProgress)
	})
}

// StartGammonNetMatchBatch is StartGammonNetBatch scoped to one match: the
// batch a transcription's save starts (ADR-0045 §8). The caller sets depth.
func (a *App) StartGammonNetMatchBatch(matchID int64, ply, pruneK, candidates int) {
	a.runGammonNetBatch(func(ctx context.Context, onProgress func(done, total int)) (database.GammonNetBatchSummary, error) {
		return a.db.AnalyzeMatchWithGammonNet(ctx, matchID, ply, pruneK, candidates, a.effectiveBatchJobs(),
			a.waitForInteractiveEvaluation, onProgress)
	})
}

// effectiveBatchJobs lowers the batch's goroutine count when a live search is
// already running, so the two never ask for 2×NumCPU; it only shortens the
// window before waitForInteractiveEvaluation takes over.
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

// runGammonNetBatch is the shell every batch shares: run does the work, this
// owns the single-in-flight bookkeeping and the events.
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
		// Registered first so it runs last, even after a panic: shutdown
		// waits on it.
		defer close(stopped)
		defer recoverBackground(a.ctx, "gammonNet batch analysis")
		summary, err := run(ctx, func(done, total int) {
			a.emitBatch("gammonnet-batch:progress", map[string]int{"done": done, "total": total})
		})

		// Only if still the batch in flight: a newer one may have replaced it.
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
		a.emitBatch("gammonnet-batch:done", map[string]int{
			"evaluated": summary.Evaluated,
			"refused":   summary.Refused,
			"failed":    summary.Failed,
		})
	}()
}

// batchCtx is the Wails context, or a background one before startup
// (context.WithCancel(nil) panics).
func (a *App) batchCtx() context.Context {
	if a.ctx == nil {
		return context.Background()
	}
	return a.ctx
}

// emitBatch emits nothing without a window: EventsEmit on a non-Wails
// context ends the process (ADR-0004).
func (a *App) emitBatch(name string, payload ...any) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, name, payload...)
}

// CancelGammonNetBatch aborts an in-flight batch and returns at once; to wait
// for it to stop, use cancelGammonNetBatch's channel.
func (a *App) CancelGammonNetBatch() {
	a.cancelGammonNetBatch()
}

// cancelGammonNetBatch also returns a channel closed once the batch has
// finished, nil when none was in flight.
func (a *App) cancelGammonNetBatch() <-chan struct{} {
	a.gnBatchMu.Lock()
	defer a.gnBatchMu.Unlock()
	if a.gnBatchCancel != nil {
		a.gnBatchCancel()
		a.gnBatchCancel = nil
	}
	return a.gnBatchDone
}

// waitForInteractiveEvaluation blocks each batch goroutine, before each
// position, while a live search runs: the batch gives way within `jobs`
// positions.
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

// stopBackgroundJobs, shutdown's first step, cancels every job and waits (up
// to grace) only for the gammonNet batch, the one that writes the database.
// The bearoff generation never touches it, and waiting for it could hang.
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
