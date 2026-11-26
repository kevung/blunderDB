package gui

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/jobs"
	"github.com/kevung/blunderdb/pkg/blunderdb/maintenance"
)

// A background computation stops at its checkpoint while a visible one runs,
// and goes on as soon as it ends.
func TestVisibleComputation_HoldsTheBackgroundBatch(t *testing.T) {
	a := NewApp(nil)
	task := a.jobs.Start(jobs.Spec{Kind: jobs.GammonNetMissing, Library: 1, Priority: jobs.Background, Compute: true, Pausable: true})
	end := a.beginVisibleComputation()
	passed := make(chan struct{})
	go func() { task.Yield(context.Background()); close(passed) }()
	select {
	case <-passed:
		t.Fatal("the batch went on under a visible computation")
	case <-time.After(30 * time.Millisecond):
	}
	end()
	select {
	case <-passed:
	case <-time.After(5 * time.Second):
		t.Fatal("the batch stayed held once the visible computation ended")
	}
	task.Finish(nil)
	if got := a.ListJobs(); len(got) != 1 || got[0].Priority != "background" || got[0].State != jobs.Done {
		t.Fatalf("tasks %+v", got)
	}
}

// waitTask waits until the task listed under id reaches state.
func waitTask(t *testing.T, a *App, id int64, state jobs.State) jobs.Info {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for {
		for _, in := range a.ListJobs() {
			if in.ID == id && in.State == state {
				return in
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("task %d never reached %s: %+v", id, state, a.ListJobs())
		}
		time.Sleep(time.Millisecond)
	}
}

func memoryApp(t *testing.T) *App {
	t.Helper()
	d := database.NewDatabase()
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	a := NewApp(d)
	drop := func(string, ...any) {}
	a.emitHook.Store(&drop)
	return a
}

// A batch that panics ends in error and gives its library to the next one.
func TestPanickedTask_GivesItsLibraryToTheNext(t *testing.T) {
	a := memoryApp(t)
	a.runGammonNetBatch(0, 2, jobs.GammonNetMissing, func(context.Context, *database.Database, func(int, int)) (database.GammonNetBatchSummary, error) {
		panic("boom")
	})
	ran := make(chan struct{})
	a.runGammonNetBatch(0, 2, jobs.GammonNetStale, func(context.Context, *database.Database, func(int, int)) (database.GammonNetBatchSummary, error) {
		close(ran)
		return database.GammonNetBatchSummary{}, nil
	})
	select {
	case <-ran:
	case <-time.After(10 * time.Second):
		t.Fatalf("the next batch never ran: %+v", a.ListJobs())
	}
	waitTask(t, a, a.ListJobs()[0].ID, jobs.Failed)
}

// A requested task waiting for the library preempts the background batch
// holding it: the batch stops at its next position, waits behind the task
// and runs anew once it is done.
func TestRequestedTask_PreemptsTheBackgroundBatch(t *testing.T) {
	a := memoryApp(t)
	var runs atomic.Int32
	a.runGammonNetBatch(0, 2, jobs.GammonNetMissing, func(ctx context.Context, _ *database.Database, _ func(int, int)) (database.GammonNetBatchSummary, error) {
		if runs.Add(1) > 1 {
			return database.GammonNetBatchSummary{Evaluated: 1}, nil
		}
		yield := a.batchYield(ctx)
		for ctx.Err() == nil {
			yield()
			time.Sleep(time.Millisecond)
		}
		return database.GammonNetBatchSummary{Evaluated: 2}, ctx.Err()
	})
	batch := a.ListJobs()[0].ID
	waitTask(t, a, batch, jobs.Running)

	ro := a.jobs.Start(jobs.Spec{Kind: jobs.RolloutBatch, Library: a.shownGroup(), Priority: jobs.Requested, Exclusive: true})
	turned := make(chan error, 1)
	go func() { turned <- ro.Turn(context.Background()) }()
	select {
	case err := <-turned:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the rollout waited for the whole batch")
	}
	if in := waitTask(t, a, batch, jobs.Waiting); in.Position != 1 {
		t.Fatalf("preempted batch %+v, want first in the queue", in)
	}
	ro.Finish(nil)
	waitTask(t, a, batch, jobs.Done)
	if runs.Load() != 2 {
		t.Fatalf("batch ran %d times, want 2", runs.Load())
	}
}

// A pass waiting in the queue keeps the library it was asked for when
// another group comes on screen.
func TestStartMaintenance_WaitingPassKeepsItsLibraryAcrossASwitch(t *testing.T) {
	a, db := twoGroups(t)
	drop := func(string, ...any) {}
	a.emitHook.Store(&drop)
	hold := a.jobs.Start(jobs.Spec{Kind: jobs.GammonNetMissing, Library: 1, Exclusive: true})
	if err := hold.Turn(context.Background()); err != nil {
		t.Fatal(err)
	}
	a.StartMaintenance(string(maintenance.StatsMatches))
	pass := a.ListJobs()[1]
	if pass.Kind != jobs.Maintenance || pass.Library != 1 || pass.State != jobs.Waiting {
		t.Fatalf("pass %+v, want waiting in library 1", pass)
	}
	if err := a.ActivateLibraryGroup(2); err != nil {
		t.Fatal(err)
	}
	hold.Finish(nil)
	waitTask(t, a, pass.ID, jobs.Done)

	lastRun := func() *maintenance.Run {
		lines, err := db.MaintenanceStatus()
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range lines {
			if string(l.Kind) == string(maintenance.StatsMatches) {
				return l.Last
			}
		}
		return nil
	}
	if lastRun() != nil {
		t.Fatal("the pass ran on the library brought on screen")
	}
	if err := a.ActivateLibraryGroup(1); err != nil {
		t.Fatal(err)
	}
	if lastRun() == nil {
		t.Fatal("the pass did not run on the library it was asked for")
	}
}

// A rollout that yields to the evaluation on screen is listed as yielded,
// and no longer once it goes on.
func TestVisibleComputation_RolloutListedAsYielded(t *testing.T) {
	a := NewApp(nil)
	task := a.jobs.Start(jobs.Spec{Kind: jobs.Rollout, Priority: jobs.Requested, Compute: true, Pausable: true})
	end := a.beginVisibleComputation()
	passed := make(chan struct{})
	go func() { task.Yield(context.Background()); close(passed) }()
	deadline := time.Now().Add(5 * time.Second)
	for !a.ListJobs()[0].Yielded {
		if time.Now().After(deadline) {
			t.Fatalf("not listed as yielded: %+v", a.ListJobs()[0])
		}
		time.Sleep(time.Millisecond)
	}
	end()
	select {
	case <-passed:
	case <-time.After(5 * time.Second):
		t.Fatal("the rollout stayed held once the visible computation ended")
	}
	if got := a.ListJobs()[0]; got.Yielded || got.State != jobs.Running {
		t.Fatalf("after its checkpoint: %+v", got)
	}
	task.Finish(nil)
}

// The cores in use reach the listener after a change, and the last state is
// never lost however fast the tokens move.
func TestWatchCores_SendsTheLastCount(t *testing.T) {
	c := gammonnet.NewCores(3)
	got := make(chan JobCores, 16)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go watchCores(ctx, c, time.Millisecond, func(jc JobCores) { got <- jc })
	r1, r2 := c.Hold(), c.Hold()
	r1()
	want := JobCores{Used: 1, Total: 3}
	for {
		select {
		case jc := <-got:
			if jc == want {
				r2()
				return
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("never sent %+v", want)
		}
	}
}
