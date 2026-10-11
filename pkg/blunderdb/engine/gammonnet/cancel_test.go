// SPDX-License-Identifier: MIT

package gammonnet

import (
	"context"
	"testing"
	"time"
)

// A bound context that never fires changes nothing, bit for bit, serial and
// with workers, plays and probabilities alike.
func TestBindContextLeavesResultUnchanged(t *testing.T) {
	pos := openingPosition(t)
	for _, workers := range []int{0, 4} {
		ref, err := NewSearcher(DefaultConfig(2))
		if err != nil {
			t.Fatal(err)
		}
		ref = ref.WithWorkers(workers)
		bound, err := NewSearcher(DefaultConfig(2))
		if err != nil {
			t.Fatal(err)
		}
		bound = bound.WithWorkers(workers)

		want := make([]Candidate, MaxPlays)
		nw, err := ref.Plays(&pos, 3, 1, want)
		if err != nil {
			t.Fatal(err)
		}
		wantProbs, ok := ref.Probs(&pos)
		if !ok {
			t.Fatal("reference probs failed")
		}

		release := bound.BindContext(context.Background())
		got := make([]Candidate, MaxPlays)
		ng, err := bound.Plays(&pos, 3, 1, got)
		gotProbs, okp := bound.Probs(&pos)
		release()
		if err != nil || !okp {
			t.Fatalf("workers=%d: bound search failed: %v %v", workers, err, okp)
		}
		if nw != ng {
			t.Fatalf("workers=%d: %d candidates, want %d", workers, ng, nw)
		}
		for i := 0; i < nw; i++ {
			if got[i].Equity != want[i].Equity || got[i].Play.Result != want[i].Play.Result {
				t.Fatalf("workers=%d: candidate %d differs", workers, i)
			}
		}
		if gotProbs != wantProbs {
			t.Fatalf("workers=%d: probs differ", workers)
		}
	}
}

// A context already done stops the search before its first roll.
func TestBindContextAlreadyCanceled(t *testing.T) {
	pos := openingPosition(t)
	s, err := NewSearcher(DefaultConfig(2))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	release := s.BindContext(ctx)
	_, err = s.Plays(&pos, 3, 1, make([]Candidate, MaxPlays))
	if !IsCanceled(err) {
		t.Fatalf("Plays: %v, want ErrCanceled", err)
	}
	if _, ok := s.Probs(&pos); ok {
		t.Fatal("Probs succeeded under a canceled context")
	}
	release()

	// Released, the searcher searches again, and a late cancel of the old
	// binding cannot reach the next one.
	if _, err := s.Plays(&pos, 3, 1, make([]Candidate, MaxPlays)); err != nil {
		t.Fatalf("after release: %v", err)
	}
}

// A deep search, serial or with workers, stops shortly after its context does.
func TestBindContextStopsDeepSearch(t *testing.T) {
	if testing.Short() {
		t.Skip("deep search")
	}
	pos := openingPosition(t)
	for _, workers := range []int{0, 4} {
		s, err := NewSearcher(DefaultConfig(4))
		if err != nil {
			t.Fatal(err)
		}
		s = s.WithWorkers(workers)
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		release := s.BindContext(ctx)
		start := time.Now()
		_, err = s.Plays(&pos, 3, 1, make([]Candidate, MaxPlays))
		elapsed := time.Since(start)
		release()
		cancel()
		if !IsCanceled(err) {
			t.Fatalf("workers=%d: %v, want ErrCanceled", workers, err)
		}
		if elapsed > 2*time.Second {
			t.Fatalf("workers=%d: stopped after %v", workers, elapsed)
		}
	}
}

// A searcher interrupted mid-search, at several depths into it, then reused
// answers bit for bit like a fresh one: cancelling leaves no state behind
// (scratch, match states, cache) that a later search could read.
func TestSearcherReusedAfterCancelMatchesFresh(t *testing.T) {
	if testing.Short() {
		t.Skip("several 2-ply searches")
	}
	pos := openingPosition(t)
	for _, workers := range []int{0, 4} {
		fresh, err := NewSearcher(DefaultConfig(2))
		if err != nil {
			t.Fatal(err)
		}
		fresh = fresh.WithWorkers(workers)
		want := make([]Candidate, MaxPlays)
		nw, err := fresh.Plays(&pos, 6, 4, want)
		if err != nil {
			t.Fatal(err)
		}
		wantProbs, ok := fresh.Probs(&pos)
		if !ok {
			t.Fatal("fresh probs failed")
		}

		for _, delay := range []time.Duration{0, time.Millisecond, 3 * time.Millisecond, 10 * time.Millisecond} {
			s, err := NewSearcher(DefaultConfig(2))
			if err != nil {
				t.Fatal(err)
			}
			s = s.WithWorkers(workers)

			ctx, cancel := context.WithTimeout(context.Background(), delay)
			release := s.BindContext(ctx)
			_, _ = s.Plays(&pos, 6, 4, make([]Candidate, MaxPlays))
			_, _ = s.Probs(&pos)
			release()
			cancel()

			got := make([]Candidate, MaxPlays)
			ng, err := s.Plays(&pos, 6, 4, got)
			if err != nil {
				t.Fatalf("workers=%d delay=%v: reuse: %v", workers, delay, err)
			}
			if ng != nw {
				t.Fatalf("workers=%d delay=%v: %d candidates, want %d", workers, delay, ng, nw)
			}
			for i := 0; i < nw; i++ {
				if got[i].Equity != want[i].Equity || got[i].Play.Result != want[i].Play.Result {
					t.Fatalf("workers=%d delay=%v: candidate %d differs from a fresh searcher", workers, delay, i)
				}
			}
			gotProbs, ok := s.Probs(&pos)
			if !ok || gotProbs != wantProbs {
				t.Fatalf("workers=%d delay=%v: probs differ from a fresh searcher", workers, delay)
			}
		}
	}
}
