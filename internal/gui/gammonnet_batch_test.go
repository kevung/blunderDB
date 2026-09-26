package gui

import (
	"context"
	goruntime "runtime"
	"testing"
	"time"
)

// TestWaitForInteractiveEvaluationBlocksWhileBusy: the batch yields while a
// live search is in flight, driven through a.gnEvalCancel directly.
func TestWaitForInteractiveEvaluationBlocksWhileBusy(t *testing.T) {
	a := NewApp(nil)

	a.gnEvalMu.Lock()
	a.gnEvalCancel = func() {} // simulate an interactive search in flight
	a.gnEvalMu.Unlock()
	t.Cleanup(func() {
		a.gnEvalMu.Lock()
		a.gnEvalCancel = nil
		a.gnEvalMu.Unlock()
	})

	returned := make(chan struct{})
	go func() {
		a.waitForInteractiveEvaluation()
		close(returned)
	}()

	select {
	case <-returned:
		t.Fatal("waitForInteractiveEvaluation returned while the interactive search was still marked in flight")
	case <-time.After(3 * gnYieldPoll):
		// Still blocked after several poll intervals — as required.
	}

	a.gnEvalMu.Lock()
	a.gnEvalCancel = nil // the interactive search "finishes"
	a.gnEvalMu.Unlock()

	select {
	case <-returned:
		// Unblocked promptly once the interactive slot freed up.
	case <-time.After(2 * time.Second):
		t.Fatal("waitForInteractiveEvaluation did not return after the interactive search finished")
	}
}

// TestWaitForInteractiveEvaluationReturnsImmediatelyWhenIdle: the common
// case — no interactive search in flight — must never pay the poll delay.
func TestWaitForInteractiveEvaluationReturnsImmediatelyWhenIdle(t *testing.T) {
	a := NewApp(nil)

	done := make(chan struct{})
	go func() {
		a.waitForInteractiveEvaluation()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(gnYieldPoll):
		t.Fatal("waitForInteractiveEvaluation blocked despite no interactive search in flight")
	}
}

// TestEffectiveBatchJobsReducedWhileInteractiveBusy: a batch starting during
// a live search asks for fewer than NumCPU goroutines.
func TestEffectiveBatchJobsReducedWhileInteractiveBusy(t *testing.T) {
	if goruntime.NumCPU() < 2 {
		t.Skip("needs at least two cores to have any reduction to measure")
	}
	a := NewApp(nil)

	idle := a.effectiveBatchJobs()

	a.gnEvalMu.Lock()
	a.gnEvalCancel = func() {}
	a.gnEvalMu.Unlock()
	t.Cleanup(func() {
		a.gnEvalMu.Lock()
		a.gnEvalCancel = nil
		a.gnEvalMu.Unlock()
	})

	busy := a.effectiveBatchJobs()
	if busy >= idle {
		t.Fatalf("effectiveBatchJobs while busy = %d, want fewer than idle's %d", busy, idle)
	}
	if busy < 1 {
		t.Fatalf("effectiveBatchJobs = %d, want at least 1", busy)
	}
}

// TestStartGammonNetBatchNoopWithoutDatabase: NewApp(nil) must never panic.
func TestStartGammonNetBatchNoopWithoutDatabase(t *testing.T) {
	a := NewApp(nil)
	a.ctx = context.Background()
	a.StartGammonNetBatch(0, 0, 0) // must return without panicking, nothing to assert on
	a.CancelGammonNetBatch()       // likewise: no batch in flight, must be a quiet no-op
}
