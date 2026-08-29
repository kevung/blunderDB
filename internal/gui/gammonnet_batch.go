package gui

import (
	"context"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// The gammonNet batch job (#129, ADR-0013): a bounded, visible, cancellable,
// resumable sweep that writes an analysis for every position that has none.
// The actual query/evaluate/write loop lives on *database.Database
// (AnalyzeMissingWithGammonNet, database/db_gammonnet_batch.go) so the CLI
// (#130) can reuse it without a GUI dependency; this file is the same
// goroutine+context+mutex+EventsEmit shell DownloadBearoffDB (bearoff.go)
// already established, wired to the batch's own events:
//
//	gammonnet-batch:progress {done, total}
//	gammonnet-batch:done     {}
//	gammonnet-batch:cancelled {}
//	gammonnet-batch:error    {message}

var (
	gnBatchMu     sync.Mutex
	gnBatchCancel context.CancelFunc
)

// gnYieldPoll is how often the batch rechecks whether the interactive live
// evaluation (gammonnet_eval.go) has freed up, once it finds it busy. Small
// enough that the batch resumes promptly once the interactive search
// finishes (ADR-0011 bounds that at well under a second on the display
// depth), large enough not to spin.
const gnYieldPoll = 50 * time.Millisecond

// StartGammonNetBatch analyses every position without an analysis, in the
// background. A batch already in flight is cancelled first — one at a time,
// like DownloadBearoffDB. No-op (emits nothing) when db is nil, which only
// happens in tests that construct an App without one.
func (a *App) StartGammonNetBatch(ply, pruneK, candidates int) {
	if a.db == nil {
		return
	}

	gnBatchMu.Lock()
	if gnBatchCancel != nil {
		gnBatchCancel()
	}
	ctx, cancel := context.WithCancel(a.ctx)
	gnBatchCancel = cancel
	gnBatchMu.Unlock()

	go func() {
		err := a.db.AnalyzeMissingWithGammonNet(ctx, ply, pruneK, candidates,
			waitForInteractiveEvaluation,
			func(done, total int) {
				runtime.EventsEmit(a.ctx, "gammonnet-batch:progress", map[string]int{"done": done, "total": total})
			})

		gnBatchMu.Lock()
		gnBatchCancel = nil
		gnBatchMu.Unlock()

		if err != nil {
			if ctx.Err() != nil {
				runtime.EventsEmit(a.ctx, "gammonnet-batch:cancelled")
				return
			}
			runtime.EventsEmit(a.ctx, "gammonnet-batch:error", map[string]string{"message": err.Error()})
			return
		}
		runtime.EventsEmit(a.ctx, "gammonnet-batch:done")
	}()
}

// CancelGammonNetBatch aborts an in-flight batch, if any.
func (a *App) CancelGammonNetBatch() {
	gnBatchMu.Lock()
	if gnBatchCancel != nil {
		gnBatchCancel()
		gnBatchCancel = nil
	}
	gnBatchMu.Unlock()
}

// waitForInteractiveEvaluation is the batch's yield point ("le lot cède en
// une position", #129): called before every position, it blocks for as long
// as gammonnet_eval.go's live evaluation (#125) has a search in flight, so
// an editing user is never fighting the batch for cores. gammonNetEvalMu and
// gammonNetEvalCancel are package-level state this file shares with
// gammonnet_eval.go without an import — both live in package gui.
func waitForInteractiveEvaluation() {
	for {
		gammonNetEvalMu.Lock()
		busy := gammonNetEvalCancel != nil
		gammonNetEvalMu.Unlock()
		if !busy {
			return
		}
		time.Sleep(gnYieldPoll)
	}
}
