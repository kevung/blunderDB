// SPDX-License-Identifier: MIT

package gammonnet

import (
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

// A claim parks a background goroutine at its next roll and lends its token:
// the claimant takes it without waiting, the background resumes once the
// claim ends, and a background Hold waits for the claim rather than take a
// lent token.
func TestCoresClaimLendsTheBackgroundToken(t *testing.T) {
	c := NewCores(1)
	release := c.HoldBackground()
	end := c.Claim()

	parked := make(chan struct{})
	go func() {
		defer close(parked)
		c.yield()
	}()
	waitCores(t, c, "the background to park", func() bool { return c.InUse() == 0 })

	got := make(chan struct{})
	go func() {
		defer close(got)
		c.HoldBackground()()
	}()
	if !c.tryTake() {
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

// A search parked mid-way behind a claim returns, once the claim ends, what
// it returns uninterrupted, bit for bit, on one core or on several.
func TestCoresClaimParksABatchSearchWithoutChangingIt(t *testing.T) {
	p := openingPosition(t)
	ref, err := NewSearcher(DefaultConfig(2))
	if err != nil {
		t.Fatal(err)
	}
	pos := p
	want, ok, err := ref.BestPlay(&pos, 3, 1)
	if err != nil || !ok {
		t.Fatalf("BestPlay: %v", err)
	}

	for _, width := range []int{1, 3} {
		c := NewCores(width)
		s, err := NewSearcher(DefaultConfig(2))
		if err != nil {
			t.Fatal(err)
		}
		s.WithCores(c).YieldsTo(c)

		release := c.HoldBackground()
		end := c.Claim()
		type answer struct {
			m   Candidate
			ok  bool
			err error
		}
		done := make(chan answer, 1)
		go func() {
			defer release()
			pos := p
			m, ok, err := s.BestPlay(&pos, 3, 1)
			done <- answer{m, ok, err}
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
						release = c.HoldBackground()
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
