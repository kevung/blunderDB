// SPDX-License-Identifier: MIT

package gammonnet

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestEvaluatePositionWithReusedSearcherIsBitIdentical is the licence the
// parallel batch (#147) rests on: one Searcher, re-aimed position after
// position and carrying its evaluation cache from one to the next, answers
// exactly what a brand-new Searcher answers for the same position.
//
// Two things could have broken that, and both are checked here at once: a
// leftover from the previous configuration (score, cube owner) surviving
// Reconfigure, and a warm cache changing an answer. It cannot — the cache
// key is the whole position and its value is the network's own output, so a
// hit is by construction what a miss would have computed (cache.go) — but
// "cannot" is what a test is for.
//
// Bit-identical, not "within a tolerance": reflect.DeepEqual over the whole
// EvalResult, every float64 included.
func TestEvaluatePositionWithReusedSearcherIsBitIdentical(t *testing.T) {
	rng := rand.New(rand.NewSource(20260902))

	// A deliberately mixed run: money and match scores, cube questions and
	// checker questions, so the reused searcher is re-aimed at a different
	// referential and a different cube state at nearly every step.
	scores := [][2]int{{-1, -1}, {3, 5}, {1, 1}, {7, 2}, {-1, -1}, {2, 4}}
	cubes := []domain.Cube{
		{Owner: domain.None, Value: 0},
		{Owner: domain.White, Value: 1},
		{Owner: domain.Black, Value: 2},
	}

	reused, err := NewBatchSearcher(0, 0)
	if err != nil {
		t.Fatalf("NewBatchSearcher: %v", err)
	}

	checked := 0
	for attempt := 0; checked < 24 && attempt < 400; attempt++ {
		onRoll := domain.White
		if attempt%2 == 1 {
			onRoll = domain.Black
		}
		pos := randomBoard(rng, onRoll)
		pos.Score = scores[attempt%len(scores)]
		pos.Cube = cubes[attempt%len(cubes)]
		if attempt%3 != 0 {
			pos.Dice = [2]int{1 + rng.Intn(6), 1 + rng.Intn(6)}
		} else {
			pos.Dice = [2]int{0, 0}
		}

		want, wantErr := EvaluatePosition(pos, 0, 0, 0)
		got, gotErr := EvaluatePositionWith(reused, pos, 0, 0, 0)

		if (wantErr == nil) != (gotErr == nil) {
			t.Fatalf("attempt %d: fresh searcher err = %v, reused searcher err = %v", attempt, wantErr, gotErr)
		}
		if wantErr != nil {
			continue
		}
		checked++
		if !reflect.DeepEqual(want, got) {
			t.Fatalf("attempt %d (score %v, cube %+v, dice %v): the reused searcher answered differently", attempt, pos.Score, pos.Cube, pos.Dice)
		}
	}
	if checked == 0 {
		t.Fatal("no position was evaluated: the test proved nothing")
	}
}

// TestReconfigureRefusesAnInvalidMatchState: Reconfigure keeps NewSearcher's
// posture — refused, never degraded to money (ADR-0016).
func TestReconfigureRefusesAnInvalidMatchState(t *testing.T) {
	s, err := NewBatchSearcher(0, 0)
	if err != nil {
		t.Fatalf("NewBatchSearcher: %v", err)
	}
	cfg := DefaultConfig(0)
	cfg.UseMatch = true
	cfg.Match = MatchState{AwayOnRoll: 0, AwayOpponent: 0}
	if err := s.Reconfigure(cfg); err == nil {
		t.Fatal("Reconfigure accepted an invalid match state, want a refusal")
	}
}
