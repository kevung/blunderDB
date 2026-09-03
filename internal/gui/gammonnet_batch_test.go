package gui

import (
	"context"
	goruntime "runtime"
	"testing"
	"time"
)

// TestWaitForInteractiveEvaluationBlocksWhileBusy is #129's non-negotiable
// preemption criterion made concrete: "un lot qui ne cède pas est exactement
// la panne qu'on prétend éviter". a.gnEvalCancel is the exact App field
// StartEvaluationAtRest sets while a live 2-ply search is in flight
// (gammonnet_eval.go); this test drives it directly rather than through a
// real Wails app, since nothing here needs event emission. gnEvalMu/
// gnEvalCancel moved from package vars onto App (#196/C.9), so each test
// now gets its own instance instead of sharing global state.
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

// TestEffectiveBatchJobsReducedWhileInteractiveBusy is #196/C.9's own
// criterion: a batch starting while the panel already has an "at rest"
// search in flight asks for fewer goroutines, not a full NumCPU on top of
// the panel's own WithWorkers(NumCPU) pool.
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

// TestStartGammonNetBatchNoopWithoutDatabase guards the nil-db escape hatch
// tests without a real Database rely on (NewApp(nil) elsewhere in this
// package) — it must never panic.
func TestStartGammonNetBatchNoopWithoutDatabase(t *testing.T) {
	a := NewApp(nil)
	a.ctx = context.Background()
	a.StartGammonNetBatch(0, 0, 0) // must return without panicking, nothing to assert on
	a.CancelGammonNetBatch()       // likewise: no batch in flight, must be a quiet no-op
}
