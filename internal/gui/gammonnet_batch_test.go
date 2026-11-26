package gui

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/jobs"
)

// batchStep runs one position's yield of a batch registered as task, and
// closes the returned channel once the batch may analyse it.
func batchStep(a *App, task *jobs.Job) <-chan struct{} {
	ctx := context.WithValue(context.Background(), taskKey{},
		&batchTask{job: task, preempt: func() {}, preempted: &atomic.Bool{}})
	passed := make(chan struct{})
	go func() { a.batchYield(ctx)(); close(passed) }()
	return passed
}

// Every search on screen — the panel's evaluation, the roll map, the score
// map — holds a background batch before its next position, and lets it go
// on once it ends; a superseded search no longer holds it, the newer does.
func TestInteractiveSearch_HoldsTheBackgroundBatchUntilItEnds(t *testing.T) {
	a := NewApp(nil)
	task := a.jobs.Start(jobs.Spec{Kind: jobs.GammonNetMissing, Library: 1, Priority: jobs.Background, Compute: true, Pausable: true})
	defer task.Finish(nil)

	held := func(passed <-chan struct{}) {
		t.Helper()
		select {
		case <-passed:
			t.Fatal("the batch went on under a visible computation")
		case <-time.After(30 * time.Millisecond):
		}
	}
	released := func(passed <-chan struct{}) {
		t.Helper()
		select {
		case <-passed:
		case <-time.After(5 * time.Second):
			t.Fatal("the batch stayed held once the visible computation ended")
		}
	}

	for _, owner := range []*uint64{nil, &a.gnRollMapSeq, &a.gnScoreMapSeq} {
		_, end := a.beginInteractiveSearch(owner)
		passed := batchStep(a, task)
		held(passed)
		end()
		end() // a deferred end after the explicit one releases nothing twice
		released(passed)
	}

	_, endOld := a.beginEvaluationAtRest()
	_, endNew := a.beginInteractiveSearch(&a.gnRollMapSeq)
	if !endOld() {
		t.Fatal("the superseded search was not cancelled")
	}
	passed := batchStep(a, task)
	held(passed)
	endNew()
	released(passed)
}

// A search whose goroutine panics still gives the batch back: end is
// deferred before the recovery.
func TestInteractiveSearch_PanicReleasesTheBatch(t *testing.T) {
	a := NewApp(nil)
	task := a.jobs.Start(jobs.Spec{Kind: jobs.GammonNetMissing, Library: 1, Priority: jobs.Background, Compute: true, Pausable: true})
	defer task.Finish(nil)

	_, end := a.beginEvaluationAtRest()
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = recover() }()
		defer end()
		panic("search broke")
	}()
	<-done
	select {
	case <-batchStep(a, task):
	case <-time.After(5 * time.Second):
		t.Fatal("a panicked search kept the batch held")
	}
}

// TestStartGammonNetBatchNoopWithoutDatabase: NewApp(nil) must never panic.
func TestStartGammonNetBatchNoopWithoutDatabase(t *testing.T) {
	a := NewApp(nil)
	a.ctx = context.Background()
	a.StartGammonNetBatch(0, 0, 0) // must return without panicking, nothing to assert on
	a.CancelGammonNetBatch()       // likewise: no batch in flight, must be a quiet no-op
}
