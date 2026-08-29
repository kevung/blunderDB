package gui

import (
	"context"
	"testing"
	"time"
)

// TestWaitForInteractiveEvaluationBlocksWhileBusy is #129's non-negotiable
// preemption criterion made concrete: "un lot qui ne cède pas est exactement
// la panne qu'on prétend éviter". gammonNetEvalCancel is the exact package
// state StartEvaluationAtRest sets while a live 2-ply search is in flight
// (gammonnet_eval.go); this test drives it directly rather than through a
// real Wails app, since nothing here needs event emission.
func TestWaitForInteractiveEvaluationBlocksWhileBusy(t *testing.T) {
	gammonNetEvalMu.Lock()
	gammonNetEvalCancel = func() {} // simulate an interactive search in flight
	gammonNetEvalMu.Unlock()
	t.Cleanup(func() {
		gammonNetEvalMu.Lock()
		gammonNetEvalCancel = nil
		gammonNetEvalMu.Unlock()
	})

	returned := make(chan struct{})
	go func() {
		waitForInteractiveEvaluation()
		close(returned)
	}()

	select {
	case <-returned:
		t.Fatal("waitForInteractiveEvaluation returned while the interactive search was still marked in flight")
	case <-time.After(3 * gnYieldPoll):
		// Still blocked after several poll intervals — as required.
	}

	gammonNetEvalMu.Lock()
	gammonNetEvalCancel = nil // the interactive search "finishes"
	gammonNetEvalMu.Unlock()

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
	gammonNetEvalMu.Lock()
	gammonNetEvalCancel = nil
	gammonNetEvalMu.Unlock()

	done := make(chan struct{})
	go func() {
		waitForInteractiveEvaluation()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(gnYieldPoll):
		t.Fatal("waitForInteractiveEvaluation blocked despite no interactive search in flight")
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
