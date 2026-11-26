package gui

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
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

// parkedBatch starts, in the group on screen, a batch whose searches yield
// like the real one's (a token held with HoldBackground, a searcher bound to
// the batch's context that YieldsTo the process budget), opens a visible
// computation once it searches, and returns when the batch is parked behind
// it. ended closes once the batch's run has returned, its goroutine gone.
func parkedBatch(t *testing.T, a *App) (ended <-chan struct{}, endVisible func() bool) {
	t.Helper()
	pos, err := domain.DecodeXGID(mapsOpeningXGID)
	if err != nil {
		t.Fatal(err)
	}
	cores := gammonnet.ProcessCores()
	searching := make(chan struct{})
	done := make(chan struct{})
	a.runGammonNetBatch(0, 2, jobs.GammonNetMissing, func(ctx context.Context, _ *database.Database, _ func(int, int)) (database.GammonNetBatchSummary, error) {
		defer close(done)
		s, err := gammonnet.NewBatchSearcher(2, 0)
		if err != nil {
			return database.GammonNetBatchSummary{}, err
		}
		defer s.BindContext(ctx)()
		s.YieldsTo(cores)
		var once sync.Once
		for {
			release, err := cores.HoldBackground(ctx)
			if err != nil {
				return database.GammonNetBatchSummary{}, err
			}
			once.Do(func() { close(searching) })
			_, err = gammonnet.EvaluatePositionWith(s, pos, 2, 0, 0)
			release()
			if err != nil {
				return database.GammonNetBatchSummary{}, err
			}
		}
	})
	<-searching
	_, endVisible = a.beginInteractiveSearch(&a.gnRollMapSeq)
	deadline := time.After(10 * time.Second)
	for cores.InUse() != 0 {
		select {
		case <-cores.Changed():
		case <-deadline:
			t.Fatalf("the batch never parked behind the visible computation (in use %d)", cores.InUse())
		}
	}
	return done, endVisible
}

// Stopping a library's tasks while its batch is parked behind a visible
// computation stops the batch at once, the map still on screen: the park
// wakes on the batch's context, not only when the claim ends.
func TestStopGroupJobs_StopsABatchParkedBehindAVisibleComputation(t *testing.T) {
	for _, path := range []string{"switch", "close"} {
		t.Run(path, func(t *testing.T) {
			a, _ := twoGroups(t)
			ended, endVisible := parkedBatch(t, a)
			defer endVisible()
			start := time.Now()
			switch path {
			case "switch": // what the database's before-switch hook runs
				a.stopGroupJobs(a.shownGroup(), shutdownJobGrace)
			case "close":
				if err := a.ActivateLibraryGroup(2); err != nil {
					t.Fatal(err)
				}
				if err := a.CloseLibraryGroup(1); err != nil {
					t.Fatal(err)
				}
			}
			elapsed := time.Since(start)
			select {
			case <-ended:
			default:
				t.Fatalf("the batch survived the stop (after %s)", elapsed)
			}
			if elapsed > shutdownJobGrace/5 {
				t.Fatalf("stopping took %s, the grace is %s", elapsed, shutdownJobGrace)
			}
		})
	}
}
