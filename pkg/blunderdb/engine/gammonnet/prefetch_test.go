package gammonnet

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// prefetchParent is the opening's 1-ply map: the rows a prefetch descends.
func prefetchParent(t *testing.T) (domain.Position, []RollMapRow) {
	t.Helper()
	pos := decodeXGIDForTest(t, openingXGID)
	m, err := ComputeRollMap(context.Background(), pos, nil, 1, 0, LocalExec(0, 1, 0), nil)
	if err != nil {
		t.Fatal(err)
	}
	return pos, m.Rows
}

func childOf(t *testing.T, pos domain.Position, r RollMapRow) domain.Position {
	t.Helper()
	child, err := PositionAfterPlay(pos, r.Die1, r.Die2, r.Move)
	if err != nil {
		t.Fatal(err)
	}
	return child
}

// The oracle, on the desktop's executor: a descent after a prefetch is served
// without a search and is, bit for bit, the map computed fresh — every child
// at 0 and 1 ply, the child looked at at 2 ply, and no other child past 1 ply;
// a child not looked at, asked at 2 ply as the display asks it, replays its
// held phase and searches only the 2-ply one.
func TestPrefetcher_DescentIsTheFreshMapBitForBit(t *testing.T) {
	pos, rows := prefetchParent(t)
	const focus = 4
	// The 2-ply children are the slow part; the 1-ply prefetch exercises the
	// same paths.
	ply := 2
	if raceEnabled || testing.Short() {
		ply = 1
	}
	cache := NewMapCache(DefaultMapCacheBytes)
	p := NewPrefetcher(cache, "", BackgroundExec(ProcessCores(), 0), nil)
	var mu sync.Mutex
	held := map[[2]int]bool{}
	<-p.Start(context.Background(), PrefetchRequest{Pos: pos, Rows: rows, Ply: ply, Focus: focus}, func(i, ply int) {
		mu.Lock()
		held[[2]int{i, ply}] = true
		mu.Unlock()
	})
	want := 2 * NumRolls
	if ply == 2 {
		want++
	}
	if len(held) != want || !held[[2]int{focus, ply}] {
		t.Fatalf("progress %v, want every child at 0 and 1 ply and the focus at 2", held)
	}

	for i, r := range rows {
		child := childOf(t, pos, r)
		plies := []int{0, 1}
		if i == focus && ply == 2 {
			plies = append(plies, 2)
		}
		for _, ply := range plies {
			got, err := cache.RollMap(context.Background(), "", child, nil, ply, 0, refusingExec(t), nil)
			if err != nil {
				t.Fatal(err)
			}
			fresh, err := ComputeRollMap(context.Background(), child, nil, ply, 0, LocalExec(0, ply, 0), nil)
			if err != nil {
				t.Fatal(err)
			}
			if !sameRollMap(got, fresh) {
				t.Fatalf("row %d: the prefetched %d-ply map differs from the fresh one", i, ply)
			}
		}
		if i != focus {
			job, err := newRollMapJob(child, nil, 2, 0)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := cache.rows(rollMapKey("", job.pos, "", job.cfg)); ok {
				t.Fatalf("row %d is not looked at, yet its 2-ply map was prefetched", i)
			}
		}
	}
	if ply < 2 {
		return
	}

	other := childOf(t, pos, rows[(focus+1)%NumRolls])
	var phases atomic.Int32
	var plies []int
	got, err := cache.RollMap(context.Background(), "", other, nil, 2, 0, countingExec(2, 0, &phases), func(_ int, r RollMapRow) {
		plies = append(plies, r.Ply)
	})
	if err != nil {
		t.Fatal(err)
	}
	if phases.Load() != 1 {
		t.Fatalf("%d phases searched for a prefetched child, want only the 2-ply one", phases.Load())
	}
	if len(plies) != 2*NumRolls || plies[0] != 1 || plies[NumRolls-1] != 1 || plies[NumRolls] != 2 {
		t.Fatalf("rows arrived at plies %v, want the held 1-ply phase, then 2-ply", plies)
	}
	fresh, err := ComputeRollMap(context.Background(), other, nil, 2, 0, LocalExec(0, 2, 0), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !sameRollMap(got, fresh) {
		t.Fatal("a 2-ply descent on a prefetched child differs from the fresh map")
	}
}

// gatedExec holds every phase until gate is closed, and counts them.
func gatedExec(gate <-chan struct{}, phases *atomic.Int32) PrefetchExec {
	return func(_ context.Context, ply, pruneK int) CellExec {
		exec := LocalExec(0, ply, pruneK)
		return func(n int, cell func(int, *Searcher)) error {
			<-gate
			phases.Add(1)
			return exec(n, cell)
		}
	}
}

// The same request twice while it runs is one run.
func TestPrefetcher_IdenticalRequestsInFlightRunOnce(t *testing.T) {
	pos, rows := prefetchParent(t)
	gate := make(chan struct{})
	var phases atomic.Int32
	p := NewPrefetcher(NewMapCache(DefaultMapCacheBytes), "", gatedExec(gate, &phases), nil)
	req := PrefetchRequest{Pos: pos, Rows: rows, Ply: 1, Focus: -1}
	first := p.Start(context.Background(), req, nil)
	again := req
	again.Pos.Dice = [2]int{3, 1} // the dice on the board are not the map's
	second := p.Start(context.Background(), again, nil)
	if first != second {
		t.Fatal("an identical request in flight started a second run")
	}
	close(gate)
	<-first
	if n := phases.Load(); n != 2*NumRolls {
		t.Fatalf("%d phases searched, want %d", n, 2*NumRolls)
	}
}

// Another branch cancels the one running: it searches nothing more, and the
// new one runs to its end.
func TestPrefetcher_NewBranchCancelsTheRunningOne(t *testing.T) {
	pos, rows := prefetchParent(t)
	gate := make(chan struct{})
	var phases atomic.Int32
	cache := NewMapCache(DefaultMapCacheBytes)
	p := NewPrefetcher(cache, "", gatedExec(gate, &phases), nil)
	var old atomic.Int32
	first := p.Start(context.Background(), PrefetchRequest{Pos: pos, Rows: rows, Ply: 1, Focus: -1}, func(int, int) { old.Add(1) })

	child := childOf(t, pos, rows[0])
	sub, err := ComputeRollMap(context.Background(), child, nil, 1, 0, LocalExec(0, 1, 0), nil)
	if err != nil {
		t.Fatal(err)
	}
	var fresh atomic.Int32
	second := p.Start(context.Background(), PrefetchRequest{Pos: child, Rows: sub.Rows, Ply: 1, Focus: -1}, func(int, int) { fresh.Add(1) })
	close(gate)
	<-first
	<-second
	if old.Load() != 0 {
		t.Fatalf("the cancelled branch went on to %d children", old.Load())
	}
	if fresh.Load() == 0 {
		t.Fatal("the new branch computed nothing")
	}
	if cache.Len() > int(fresh.Load()) {
		t.Fatalf("%d phases held, want only the new branch's %d", cache.Len(), fresh.Load())
	}
}

// While a visible computation claims the cores, the background, started and
// waiting for its tokens, runs no cell; released, it runs them all. A phase
// cancelled while it waits runs nothing and says so.
func TestBackgroundExec_YieldsToAClaim(t *testing.T) {
	cores := NewCores(2)
	release := cores.Claim()
	var ran atomic.Int32
	done := make(chan error)
	go func() {
		done <- BackgroundExec(cores, 2)(context.Background(), 0, 0)(5, func(int, *Searcher) { ran.Add(1) })
	}()
	// Both workers are running and queued behind the claim, free tokens
	// notwithstanding.
	for queued := 0; queued < 2; runtime.Gosched() {
		cores.mu.Lock()
		queued = len(cores.waiters)
		cores.mu.Unlock()
	}
	if ran.Load() != 0 {
		t.Fatalf("%d cells ran under a claim", ran.Load())
	}
	release()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if ran.Load() != 5 {
		t.Fatalf("%d cells ran once the claim ended, want 5", ran.Load())
	}

	release = cores.Claim()
	defer release()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := BackgroundExec(cores, 2)(ctx, 0, 0)(5, func(int, *Searcher) { ran.Add(1) }); err == nil {
		t.Fatal("a phase cancelled while waiting for a token reported no error")
	}
	if ran.Load() != 5 {
		t.Fatal("a cancelled phase ran cells")
	}
}

// A panic in a worker reaches the caller of the phase, and the Prefetcher
// hands it to onPanic: the request ends, the process does not.
func TestPrefetcher_PanicGoesToOnPanic(t *testing.T) {
	pos, rows := prefetchParent(t)
	boom := func(ctx context.Context, ply, pruneK int) CellExec {
		exec := BackgroundExec(NewCores(2), 2)(ctx, ply, pruneK)
		return func(n int, cell func(int, *Searcher)) error {
			return exec(n, func(int, *Searcher) { panic("boom") })
		}
	}
	got := make(chan any, 1)
	p := NewPrefetcher(NewMapCache(DefaultMapCacheBytes), "", boom, func(r any) { got <- r })
	<-p.Start(context.Background(), PrefetchRequest{Pos: pos, Rows: rows, Ply: 1, Focus: -1}, nil)
	if r := <-got; r != "boom" {
		t.Fatalf("onPanic got %v", r)
	}
}
