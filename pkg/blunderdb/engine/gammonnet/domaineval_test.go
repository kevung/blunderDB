// SPDX-License-Identifier: MIT

package gammonnet

import (
	"math"
	"math/rand"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestEvaluateMovesSetsEquityError guards the Eval panel bug (#132) where a
// live checker-move evaluation always showed a 0 equity loss, no matter how
// bad the candidate: evaluateMoves ranked moves but never filled in
// EquityError. The convention mirrors the three other places that already
// compute this figure at save time (ingest/merge.go's
// sortCheckerMovesByEquity, database/db_analysis.go's two copies): nil for
// the best move, bestEquity-equity (a non-negative loss, moves come back
// best-first per Searcher.Plays) for every other one.
func TestEvaluateMovesSetsEquityError(t *testing.T) {
	rng := rand.New(rand.NewSource(20260830))

	checked := 0
	for attempt := 0; checked < 15 && attempt < 500; attempt++ {
		onRoll := domain.White
		if attempt%2 == 1 {
			onRoll = domain.Black
		}
		pos := randomBoard(rng, onRoll)
		pos.Dice = [2]int{1 + rng.Intn(6), 1 + rng.Intn(6)}

		result, err := EvaluatePosition(pos, 0, 0, 0)
		if err != nil || len(result.Moves) < 2 {
			// Skip a dance or a position with only one legal play — nothing to
			// distinguish "best" from "the rest" there.
			continue
		}
		checked++

		best := result.Moves[0]
		if best.EquityError != nil {
			t.Fatalf("attempt %d: best move %q has EquityError = %v, want nil", attempt, best.Move, *best.EquityError)
		}
		for i, m := range result.Moves[1:] {
			if m.EquityError == nil {
				t.Fatalf("attempt %d: move %d (%q) has a nil EquityError, want bestEquity-equity", attempt, i+1, m.Move)
			}
			want := best.Equity - m.Equity
			if got := *m.EquityError; math.Abs(got-want) > 1e-9 {
				t.Fatalf("attempt %d: move %d (%q) EquityError = %v, want %v", attempt, i+1, m.Move, got, want)
			}
			if *m.EquityError < -1e-9 {
				t.Fatalf("attempt %d: move %d (%q) EquityError = %v, want a non-negative loss", attempt, i+1, m.Move, *m.EquityError)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no random position produced two or more candidate moves to compare — randomBoard/dice generation is broken")
	}
}
