package jobs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// within fails the test when ch is not closed soon: a waiter that never
// wakes is the bug these tests look for.
func within(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("%s: still blocked", what)
	}
}

// blocked fails the test when ch closes: the waiter must still wait.
func blocked(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
		t.Fatalf("%s: went on", what)
	case <-time.After(30 * time.Millisecond):
	}
}

func turnAsync(j *Job, ctx context.Context) (<-chan struct{}, *error) {
	done := make(chan struct{})
	var err error
	go func() { err = j.Turn(ctx); close(done) }()
	return done, &err
}

func TestTurn_SecondExclusiveTaskWaitsInsteadOfBeingRefused(t *testing.T) {
	r := New(nil)
	a := r.Start(Spec{Kind: GammonNetMissing, Library: 1, Priority: Background, Exclusive: true})
	if err := a.Turn(context.Background()); err != nil {
		t.Fatal(err)
	}
	b := r.Start(Spec{Kind: RolloutBatch, Library: 1, Priority: Requested, Exclusive: true})
	if got := r.List()[1]; got.State != Waiting || got.Position != 1 {
		t.Fatalf("second task: %+v, want waiting first in queue", got)
	}
	done, err := turnAsync(b, context.Background())
	blocked(t, done, "turn while the library is held")
	a.Finish(nil)
	within(t, done, "turn after the holder finished")
	if *err != nil {
		t.Fatal(*err)
	}
	if got := r.List()[1]; got.State != Running || got.Position != 0 {
		t.Fatalf("after its turn: %+v", got)
	}
}

func TestTurn_OtherLibraryDoesNotWait(t *testing.T) {
	r := New(nil)
	a := r.Start(Spec{Library: 1, Exclusive: true})
	_ = a.Turn(context.Background())
	b := r.Start(Spec{Library: 2, Exclusive: true})
	done, _ := turnAsync(b, context.Background())
	within(t, done, "turn on another library")
}

func TestStart_QueueOrdersByPriorityThenArrival(t *testing.T) {
	r := New(nil)
	hold := r.Start(Spec{Library: 1, Exclusive: true})
	_ = hold.Turn(context.Background())
	bg := r.Start(Spec{Library: 1, Priority: Background, Exclusive: true})
	req1 := r.Start(Spec{Library: 1, Priority: Requested, Exclusive: true})
	req2 := r.Start(Spec{Library: 1, Priority: Requested, Exclusive: true})
	want := map[int64]int{req1.ID(): 1, req2.ID(): 2, bg.ID(): 3}
	for _, in := range r.List() {
		if p, ok := want[in.ID]; ok && in.Position != p {
			t.Errorf("task %d at %d, want %d", in.ID, in.Position, p)
		}
	}
}

func TestMove_ReordersTheLibraryQueue(t *testing.T) {
	r := New(nil)
	hold := r.Start(Spec{Library: 1, Exclusive: true})
	_ = hold.Turn(context.Background())
	a := r.Start(Spec{Library: 1, Exclusive: true})
	other := r.Start(Spec{Library: 2, Exclusive: true}) // interleaved, never moved
	b := r.Start(Spec{Library: 1, Exclusive: true})
	if err := r.Move(b.ID(), 0); err != nil {
		t.Fatal(err)
	}
	doneB, _ := turnAsync(b, context.Background())
	doneA, _ := turnAsync(a, context.Background())
	hold.Finish(nil)
	within(t, doneB, "moved task")
	blocked(t, doneA, "task moved behind")
	b.Finish(nil)
	within(t, doneA, "task behind, once the first ended")
	if err := r.Move(hold.ID(), 0); !errors.Is(err, ErrNotQueued) {
		t.Errorf("moving a finished task: %v", err)
	}
	_ = other
}

func TestPause_WaitingTaskLetsTheNextOneGo(t *testing.T) {
	r := New(nil)
	a := r.Start(Spec{Library: 1, Exclusive: true})
	b := r.Start(Spec{Library: 1, Exclusive: true})
	if err := r.Pause(a.ID()); err != nil {
		t.Fatal(err)
	}
	doneB, _ := turnAsync(b, context.Background())
	within(t, doneB, "next task behind a paused one")
	b.Finish(nil)
	doneA, _ := turnAsync(a, context.Background())
	blocked(t, doneA, "paused task")
	_ = r.Resume(a.ID())
	within(t, doneA, "resumed task")
}

func TestCheckpoint_PauseAndResume(t *testing.T) {
	r := New(nil)
	j := r.Start(Spec{Pausable: true})
	if err := r.Pause(j.ID()); err != nil {
		t.Fatal(err)
	}
	if got := r.List()[0].State; got != Paused {
		t.Fatalf("state %s, want paused", got)
	}
	done := make(chan struct{})
	go func() { _ = j.Checkpoint(context.Background()); close(done) }()
	blocked(t, done, "paused checkpoint")
	_ = r.Resume(j.ID())
	within(t, done, "resumed checkpoint")
	if err := r.Pause(r.Start(Spec{}).ID()); !errors.Is(err, ErrNotPausable) {
		t.Errorf("pausing a running task that has no checkpoint: %v", err)
	}
}

func TestCheckpoint_BackgroundYieldsToVisibleAndRequested(t *testing.T) {
	r := New(nil)
	bg := r.Start(Spec{Priority: Background, Compute: true, Pausable: true})
	exit := r.EnterVisible()
	done := make(chan struct{})
	go func() { _ = bg.Checkpoint(context.Background()); close(done) }()
	blocked(t, done, "background under a visible computation")
	exit()
	exit() // idempotent
	within(t, done, "background once the visible computation ended")

	ro := r.Start(Spec{Priority: Requested, Compute: true})
	done = make(chan struct{})
	go func() { _ = bg.Checkpoint(context.Background()); close(done) }()
	blocked(t, done, "background under a requested rollout")
	ro.Finish(nil)
	within(t, done, "background once the rollout ended")

	// A requested computation yields to the visible one only.
	exit = r.EnterVisible()
	if err := r.Start(Spec{Priority: Requested}).Checkpoint(context.Background()); err != nil {
		t.Fatal(err) // not Compute: never outranked
	}
	exit()
}

func TestCheckpoint_YieldedWhileAVisibleComputationRuns(t *testing.T) {
	var mu sync.Mutex
	var seen []bool
	r := New(func(name string, payload any) {
		if in, ok := payload.(Info); ok && name == EventChanged && in.Kind == Rollout {
			mu.Lock()
			seen = append(seen, in.Yielded)
			mu.Unlock()
		}
	})
	ro := r.Start(Spec{Kind: Rollout, Priority: Requested, Compute: true, Pausable: true})
	exit := r.EnterVisible()
	done := make(chan struct{})
	go func() { _ = ro.Checkpoint(context.Background()); close(done) }()
	blocked(t, done, "requested rollout under a visible computation")
	if got := r.List()[0]; got.State != Running || !got.Yielded {
		t.Fatalf("while it yields: %+v, want running and yielded", got)
	}
	exit()
	within(t, done, "rollout once the visible computation ended")
	if got := r.List()[0]; got.Yielded {
		t.Fatalf("after its checkpoint: %+v, want not yielded", got)
	}
	mu.Lock()
	got := append([]bool(nil), seen...)
	mu.Unlock()
	if len(got) < 3 || !got[len(got)-2] || got[len(got)-1] {
		t.Fatalf("yielded on jobs:changed: %v, want ... true, false", got)
	}

	// A paused task is paused, not yielded.
	exit = r.EnterVisible()
	defer exit()
	_ = r.Pause(ro.ID())
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_ = ro.Checkpoint(ctx)
	if got := r.List()[0]; got.State != Paused || got.Yielded {
		t.Fatalf("paused under a visible computation: %+v", got)
	}
}

func TestCancel_ReleasesWaitersAndFinishesCancelled(t *testing.T) {
	r := New(nil)
	hold := r.Start(Spec{Library: 1, Exclusive: true})
	_ = hold.Turn(context.Background())
	var called bool
	w := r.Start(Spec{Library: 1, Exclusive: true, Cancel: func() { called = true }})
	done, err := turnAsync(w, context.Background())
	if e := r.Cancel(w.ID()); e != nil {
		t.Fatal(e)
	}
	within(t, done, "cancelled waiter")
	if !errors.Is(*err, context.Canceled) || !called {
		t.Fatalf("turn: %v, cancel called %v", *err, called)
	}
	w.Finish(*err)
	if got := r.List()[1].State; got != Cancelled {
		t.Fatalf("state %s", got)
	}
	if err := r.Cancel(w.ID()); !errors.Is(err, ErrFinished) {
		t.Errorf("cancelling a finished task: %v", err)
	}
	if err := r.Cancel(r.Start(Spec{}).ID()); !errors.Is(err, ErrNoCancel) {
		t.Errorf("cancelling a task without Cancel: %v", err)
	}
}

func TestRemove(t *testing.T) {
	r := New(nil)
	hold := r.Start(Spec{Library: 1, Exclusive: true})
	_ = hold.Turn(context.Background())
	if err := r.Remove(hold.ID()); !errors.Is(err, ErrRunning) {
		t.Fatalf("removing a running task: %v", err)
	}
	w := r.Start(Spec{Library: 1, Exclusive: true})
	if err := r.Remove(w.ID()); err != nil {
		t.Fatal(err)
	}
	if err := w.Turn(context.Background()); !errors.Is(err, context.Canceled) {
		t.Fatalf("turn of a removed task: %v", err)
	}
	w.Finish(context.Canceled)
	hold.Finish(errors.New("boom"))
	list := r.List()
	if len(list) != 1 || list[0].State != Failed || list[0].Error != "boom" {
		t.Fatalf("list %+v", list)
	}
	_ = r.Remove(hold.ID())
	if len(r.List()) != 0 {
		t.Fatal("finished task still listed")
	}
}

func TestStop_CancelsTheLibraryAndWaitsWithinGrace(t *testing.T) {
	r := New(nil)
	ctx1, cancel1 := context.WithCancel(context.Background())
	a := r.Start(Spec{Library: 1, Cancel: cancel1})
	go func() { <-ctx1.Done(); a.Finish(ctx1.Err()) }()
	stuck := r.Start(Spec{Library: 1, Cancel: func() {}})
	other := r.Start(Spec{Library: 2, Cancel: func() {}})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	late := r.Stop(ctx, 1)
	if len(late) != 1 || late[0].ID != stuck.ID() {
		t.Fatalf("late %+v, want the stuck task only", late)
	}
	if got := r.List()[0].State; got != Cancelled {
		t.Errorf("stopped task: %s", got)
	}
	r.mu.Lock()
	untouched := !other.stopping
	r.mu.Unlock()
	if !untouched {
		t.Error("Stop(1) cancelled a task of library 2")
	}
}

func TestEvents_InOrderOfTheChanges(t *testing.T) {
	var mu sync.Mutex
	var got []string
	r := New(func(name string, payload any) {
		mu.Lock()
		defer mu.Unlock()
		if in, ok := payload.(Info); ok {
			got = append(got, name+":"+string(in.State))
			return
		}
		got = append(got, name)
	})
	j := r.Start(Spec{})
	j.Progress(1, 2)
	j.Progress(1, 2) // unchanged: no event
	j.Finish(nil)
	_ = r.Remove(j.ID())
	want := []string{"jobs:changed:running", "jobs:changed:running", "jobs:changed:done", "jobs:removed"}
	mu.Lock()
	defer mu.Unlock()
	if len(got) != len(want) {
		t.Fatalf("events %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("events %v, want %v", got, want)
		}
	}
}

func TestFinished_ListIsBounded(t *testing.T) {
	r := New(nil)
	for range keepFinished + 5 {
		r.Start(Spec{}).Finish(nil)
	}
	if n := len(r.List()); n != keepFinished {
		t.Fatalf("%d finished tasks listed, want %d", n, keepFinished)
	}
}

func TestNilRegistry_RunsTasksAlone(t *testing.T) {
	var r *Registry
	j := r.Start(Spec{Library: 1, Exclusive: true})
	if err := j.Turn(context.Background()); err != nil {
		t.Fatal(err)
	}
	j.Progress(1, 1)
	j.Finish(nil)
	r.EnterVisible()()
	if r.List() != nil || !errors.Is(r.Cancel(j.ID()), ErrUnknown) || r.Stop(context.Background(), 0) != nil {
		t.Fatal("a nil registry lists something")
	}
}

func TestStop_LeavesTasksOfNoLibrary(t *testing.T) {
	r := New(nil)
	var called bool
	r.Start(Spec{Cancel: func() { called = true }})
	if late := r.Stop(context.Background(), 0); len(late) != 0 || called {
		t.Fatalf("Stop(0) touched a task bound to no library: late %v, cancel called %v", late, called)
	}
}

func TestCheckpoint_PausedTaskOutranksNothing(t *testing.T) {
	r := New(nil)
	bg := r.Start(Spec{Priority: Background, Compute: true, Pausable: true})
	ro := r.Start(Spec{Priority: Requested, Compute: true, Pausable: true})
	_ = r.Pause(ro.ID())
	done := make(chan struct{})
	go func() { _ = bg.Checkpoint(context.Background()); close(done) }()
	within(t, done, "background beside a paused rollout")
}

func TestPreemption_RequestedTaskTakesTheLibraryFromABackgroundBatch(t *testing.T) {
	r := New(nil)
	batch := r.Start(Spec{Library: 1, Priority: Background, Exclusive: true, Pausable: true, Preemptible: true})
	_ = batch.Turn(context.Background())
	if err := batch.Checkpoint(context.Background()); err != nil {
		t.Fatalf("alone: %v", err)
	}
	ro := r.Start(Spec{Library: 1, Priority: Requested, Exclusive: true})
	if err := batch.Checkpoint(context.Background()); !errors.Is(err, ErrPreempted) {
		t.Fatalf("checkpoint with a rollout waiting: %v, want ErrPreempted", err)
	}
	batch.Requeue()
	if got := r.List()[0]; got.State != Waiting || got.Position != 2 {
		t.Fatalf("requeued batch %+v, want waiting behind the rollout", got)
	}
	doneRo, _ := turnAsync(ro, context.Background())
	within(t, doneRo, "rollout once the batch gave the library back")
	doneB, _ := turnAsync(batch, context.Background())
	blocked(t, doneB, "batch while the rollout holds the library")
	ro.Finish(nil)
	within(t, doneB, "batch resuming after the rollout")

	// A paused waiting task, or a task that is not preemptible, takes nothing.
	ro2 := r.Start(Spec{Library: 1, Priority: Requested, Exclusive: true})
	_ = r.Pause(ro2.ID())
	if err := batch.Checkpoint(context.Background()); err != nil {
		t.Fatalf("checkpoint with a paused rollout waiting: %v", err)
	}
	m := r.Start(Spec{Library: 2, Priority: Background, Exclusive: true, Pausable: true})
	_ = m.Turn(context.Background())
	r.Start(Spec{Library: 2, Priority: Requested, Exclusive: true})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := m.Checkpoint(ctx); err != nil {
		t.Fatalf("a task that is not preemptible was preempted: %v", err)
	}
}
