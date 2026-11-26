// SPDX-License-Identifier: MIT

package gammonnet

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// waitCores blocks until ok holds on c, woken by c's own change signal.
func waitCores(t *testing.T, c *Cores, what string, ok func() bool) {
	t.Helper()
	deadline := time.After(10 * time.Second)
	for !ok() {
		select {
		case <-c.Changed():
		case <-deadline:
			t.Fatalf("timed out waiting for %s (in use %d of %d)", what, c.InUse(), c.Cap())
		}
	}
}

func (c *Cores) parkedNow() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.parked
}

func (c *Cores) heldNow() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.held
}

func holdBackground(t *testing.T, c *Cores) func() {
	t.Helper()
	release, err := c.HoldBackground(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return release
}

// A claim parks a background goroutine at its next roll and lends its token:
// the claimant takes it without waiting, the background resumes once the
// claim ends, and a background Hold waits for the claim rather than take a
// lent token.
func TestCoresClaimLendsTheBackgroundToken(t *testing.T) {
	c := NewCores(1)
	release := holdBackground(t, c)
	end := c.Claim()

	parked := make(chan struct{})
	go func() {
		defer close(parked)
		c.yield(nil)
	}()
	waitCores(t, c, "the background to park", func() bool { return c.InUse() == 0 })

	got := make(chan struct{})
	go func() {
		defer close(got)
		holdBackground(t, c)()
	}()
	if c.tryTake(true) {
		t.Fatal("a background fork took a token lent to the claim")
	}
	if !c.tryTake(false) {
		t.Fatal("the claimant found no token while the background was parked")
	}
	select {
	case <-parked:
		t.Fatal("the background resumed under the claim")
	default:
	}
	c.give()
	select {
	case <-got:
		t.Fatal("a background Hold took a token lent to the claim")
	default:
	}

	end()
	end() // a second call releases nothing twice
	<-parked
	release()
	<-got
	if c.InUse() != 0 {
		t.Fatalf("in use %d after every release", c.InUse())
	}
}

// Claims stack: the background resumes only once the last is released.
func TestCoresStackedClaimsResumeAfterTheLast(t *testing.T) {
	c := NewCores(1)
	release := holdBackground(t, c)
	defer release()
	first, second := c.Claim(), c.Claim()
	parked := make(chan struct{})
	go func() {
		defer close(parked)
		c.yield(nil)
	}()
	waitCores(t, c, "the background to park", func() bool { return c.parkedNow() == 1 })
	first()
	select {
	case <-parked:
		t.Fatal("the background resumed while a claim was still open")
	default:
	}
	second()
	<-parked
}

// A claimant still holding a lent token once its claim ends keeps the
// background parked until it gives the token back: the budget is never
// exceeded.
func TestCoresResumeWaitsForTheClaimantsTokens(t *testing.T) {
	c := NewCores(1)
	release := holdBackground(t, c)
	defer release()
	end := c.Claim()
	parked := make(chan struct{})
	go func() {
		defer close(parked)
		c.yield(nil)
	}()
	waitCores(t, c, "the background to park", func() bool { return c.parkedNow() == 1 })
	lent := c.Hold()
	end()
	select {
	case <-parked:
		t.Fatal("the background resumed while the claimant still ran on its token")
	default:
	}
	if c.InUse() > c.Cap() {
		t.Fatalf("in use %d over a budget of %d", c.InUse(), c.Cap())
	}
	lent()
	<-parked
}

// A background goroutine parked or waiting under a claim ends as soon as its
// context does, not when the claim ends.
func TestCoresCancelReachesTheParkedBackground(t *testing.T) {
	p := openingPosition(t)
	c := NewCores(2)
	s, err := NewSearcher(DefaultConfig(2))
	if err != nil {
		t.Fatal(err)
	}
	s.WithCores(c).YieldsTo(c)
	ctx, cancel := context.WithCancel(context.Background())
	defer s.BindContext(ctx)()

	release := holdBackground(t, c)
	end := c.Claim()
	defer end()
	done := make(chan error, 1)
	go func() {
		defer release()
		pos := p
		_, _, err := s.BestPlay(&pos, 3, 1)
		done <- err
	}()
	waiting := make(chan error, 1)
	go func() {
		_, err := c.HoldBackground(ctx)
		waiting <- err
	}()
	waitCores(t, c, "the search to park", func() bool { return c.parkedNow() > 0 })

	cancel()
	if err := <-done; !IsCanceled(err) {
		t.Fatalf("parked search returned %v, want ErrCanceled", err)
	}
	if err := <-waiting; err == nil {
		t.Fatal("a background Hold under the claim outlived its context")
	}
	waitCores(t, c, "every token back", func() bool { return c.InUse() == 0 && c.parkedNow() == 0 })
}

type claimAnswer struct {
	m   Candidate
	ok  bool
	err error
}

// A search parked mid-way behind a claim returns, once the claim ends, what
// it returns uninterrupted, bit for bit, on one core or on several.
func TestCoresClaimParksABatchSearchWithoutChangingIt(t *testing.T) {
	p := openingPosition(t)
	want := referencePlay(t, p)
	for _, width := range []int{1, 3} {
		c := NewCores(width)
		s, err := NewSearcher(DefaultConfig(2))
		if err != nil {
			t.Fatal(err)
		}
		s.WithCores(c).YieldsTo(c)

		release := holdBackground(t, c)
		end := c.Claim()
		done := make(chan claimAnswer, 1)
		go func() {
			defer release()
			pos := p
			m, ok, err := s.BestPlay(&pos, 3, 1)
			done <- claimAnswer{m, ok, err}
		}()
		waitCores(t, c, "the batch search to park", func() bool { return c.InUse() == 0 })
		select {
		case <-done:
			t.Fatal("the batch search finished under the claim")
		default:
		}
		end()
		got := <-done
		if got.err != nil || !got.ok || got.m != want {
			t.Fatalf("width %d: parked search answered %+v (%v), want %+v", width, got.m, got.err, want)
		}
	}
}

// Claims opened and lifted again and again while a forked search runs park
// and resume its helpers mid-fork; the answer is the same bit for bit.
func TestCoresRepeatedClaimsMidForkLeaveTheAnswer(t *testing.T) {
	p := openingPosition(t)
	want := referencePlay(t, p)
	c := NewCores(3)
	s, err := NewSearcher(DefaultConfig(2))
	if err != nil {
		t.Fatal(err)
	}
	s.WithCores(c).YieldsTo(c)

	release := holdBackground(t, c)
	var finished atomic.Bool
	done := make(chan claimAnswer, 1)
	go func() {
		defer release()
		pos := p
		m, ok, err := s.BestPlay(&pos, 3, 1)
		finished.Store(true)
		done <- claimAnswer{m, ok, err}
	}()
	claims, parks := 0, 0
	for !finished.Load() {
		// Claim only once the search has helpers out, so the park lands
		// inside a fork.
		waitCores(t, c, "helpers out", func() bool { return c.heldNow() > 1 || finished.Load() })
		end := c.Claim()
		claims++
		waitCores(t, c, "a park or the end", func() bool { return c.parkedNow() > 0 || finished.Load() })
		if c.parkedNow() > 0 && c.heldNow() > 1 {
			parks++ // parked while helpers held tokens: mid-fork
		}
		end()
		waitCores(t, c, "the resumption", func() bool { return c.parkedNow() == 0 || finished.Load() })
	}
	got := <-done
	if got.err != nil || !got.ok || got.m != want {
		t.Fatalf("after %d claims: answered %+v (%v), want %+v", claims, got.m, got.err, want)
	}
	if parks == 0 {
		t.Fatalf("none of %d claims parked the search mid-fork", claims)
	}
}

func referencePlay(t *testing.T, p Position) Candidate {
	t.Helper()
	ref, err := NewSearcher(DefaultConfig(2))
	if err != nil {
		t.Fatal(err)
	}
	pos := p
	want, ok, err := ref.BestPlay(&pos, 3, 1)
	if err != nil || !ok {
		t.Fatalf("BestPlay: %v", err)
	}
	return want
}

// TestProbeVisibleLatencyUnderBatch mesure le délai entre la demande d'un
// calcul visible et son premier résultat pendant qu'un lot 2-ply occupe toute
// la machine : une décision 2-ply qui se répartit sur les cœurs libres
// (l'évaluation au repos) et une décision qui prend d'abord un jeton (la case
// d'une carte), sans puis avec la réquisition (Claim).
//
//	BLUNDERDB_PROBE=1 go test -run TestProbeVisibleLatencyUnderBatch -v ./pkg/blunderdb/engine/gammonnet/
func TestProbeVisibleLatencyUnderBatch(t *testing.T) {
	if os.Getenv("BLUNDERDB_PROBE") == "" {
		t.Skip("set BLUNDERDB_PROBE to measure; this test asserts nothing")
	}
	p := openingPosition(t)
	reps := 6
	width := runtime.GOMAXPROCS(0)
	fmt.Printf("noyau %s, GOMAXPROCS=%d, %d répétitions\n\n", KernelName(), width, reps)

	for _, claim := range []bool{false, true} {
		c := NewCores(width)
		var stop atomic.Bool
		var wg sync.WaitGroup
		for range width {
			s, err := NewSearcher(DefaultConfig(2))
			if err != nil {
				t.Fatal(err)
			}
			s.WithCores(c)
			if claim {
				s.YieldsTo(c)
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				for !stop.Load() {
					var release func()
					if claim {
						release, _ = c.HoldBackground(context.Background())
					} else {
						release = c.Hold()
					}
					// A batch position: the move and its two neighbours.
					for _, d := range [][2]int{{3, 1}, {6, 4}, {5, 2}} {
						coldCaches(s)
						pos := p
						_, _, _ = s.BestPlay(&pos, d[0], d[1])
					}
					release()
				}
			}()
		}
		waitCores(t, c, "the batch to fill the budget", func() bool { return c.InUse() == width })

		visible, err := NewSearcher(DefaultConfig(2))
		if err != nil {
			t.Fatal(err)
		}
		visible.WithCores(c)
		var forked, held []time.Duration
		for range reps {
			for _, hold := range []bool{false, true} {
				coldCaches(visible)
				start := time.Now()
				end := func() {}
				if claim {
					end = c.Claim()
				}
				release := func() {}
				if hold {
					release = c.Hold()
				}
				pos := p
				if _, ok, err := visible.BestPlay(&pos, 4, 2); err != nil || !ok {
					t.Fatalf("BestPlay: %v", err)
				}
				d := time.Since(start)
				release()
				end()
				if hold {
					held = append(held, d)
				} else {
					forked = append(forked, d)
				}
			}
		}
		stop.Store(true)
		wg.Wait()

		label := "sans réquisition"
		if claim {
			label = "avec réquisition"
		}
		for _, row := range []struct {
			name string
			d    []time.Duration
		}{{"décision répartie", forked}, {"case avec jeton", held}} {
			sort.Slice(row.d, func(i, j int) bool { return row.d[i] < row.d[j] })
			fmt.Printf("%-18s %-18s p50 %8s  p95 %8s  max %8s\n", label, row.name,
				quantile(row.d, 0.5).Round(time.Millisecond), quantile(row.d, 0.95).Round(time.Millisecond),
				row.d[len(row.d)-1].Round(time.Millisecond))
		}
	}
}
