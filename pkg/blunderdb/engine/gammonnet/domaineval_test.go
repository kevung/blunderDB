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

// TestEvaluatePositionHonoursTheScore is ADR-0016's own regression: before
// use_match, the opening 6-4's candidates were bit-identical whatever the
// score (measured 2026-08-31 across money, 7-away/7-away, gammon-go 4a/2a,
// gammon-save 2a/4a, 2a/2a and DMP 1a/1a) because pos.Score never reached the
// checker-move search. It must not go back to that: the gammonish play
// (fewer priming points, more gammon chances — 6/2 8/2 on this roll) must be
// valued differently at DMP, where a gammon is worth nothing extra, than at
// gammon-go, where it is worth most of the game.
func TestEvaluatePositionHonoursTheScore(t *testing.T) {
	base, err := domain.DecodeXGID(openingXGID)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	base.PlayerOnRoll = domain.White
	base.Dice = [2]int{6, 4}

	atScore := func(score [2]int) domain.Position {
		p := base
		p.Score = score
		return p
	}

	money := atScore([2]int{-1, -1})
	dmp := atScore([2]int{1, 1})        // both 1-away: a gammon is worth nothing extra
	gammonGo := atScore([2]int{4, 2})   // White 2-away: a gammon White wins is worth a lot
	gammonSave := atScore([2]int{2, 4}) // White 4-away, Black 2-away: mirror of gammonGo

	moneyRes, err := EvaluatePosition(money, 0, 0, 0)
	if err != nil {
		t.Fatalf("money: %v", err)
	}
	dmpRes, err := EvaluatePosition(dmp, 0, 0, 0)
	if err != nil {
		t.Fatalf("DMP: %v", err)
	}
	goRes, err := EvaluatePosition(gammonGo, 0, 0, 0)
	if err != nil {
		t.Fatalf("gammon-go: %v", err)
	}
	saveRes, err := EvaluatePosition(gammonSave, 0, 0, 0)
	if err != nil {
		t.Fatalf("gammon-save: %v", err)
	}
	for label, res := range map[string][]domain.CheckerMove{
		"money": moneyRes.Moves, "DMP": dmpRes.Moves,
		"gammon-go": goRes.Moves, "gammon-save": saveRes.Moves,
	} {
		if len(res) < 2 {
			t.Fatalf("%s: expected several candidates for the opening 6-4, got %d", label, len(res))
		}
	}

	// The candidate with the highest gammon chance — the play a score-blind
	// search chases identically everywhere.
	mostGammonish := func(moves []domain.CheckerMove) domain.CheckerMove {
		best := moves[0]
		for _, m := range moves[1:] {
			if m.PlayerGammonChance > best.PlayerGammonChance {
				best = m
			}
		}
		return best
	}
	lossOf := func(m domain.CheckerMove) float64 {
		if m.EquityError == nil {
			return 0
		}
		return *m.EquityError
	}

	moneyLoss := lossOf(mostGammonish(moneyRes.Moves))
	dmpLoss := lossOf(mostGammonish(dmpRes.Moves))
	goLoss := lossOf(mostGammonish(goRes.Moves))
	saveLoss := lossOf(mostGammonish(saveRes.Moves))

	// The bug this guards: all four were bit-identical. At minimum, DMP and
	// gammon-go must disagree — a gammon is worth the least at the first,
	// the most at the second, of any two scores this test tries.
	if dmpLoss == goLoss {
		t.Errorf("the gammonish play's loss is identical at DMP (%v) and gammon-go (%v) — the score is not reaching the checker-move search", dmpLoss, goLoss)
	}
	// At gammon-go the gammon-chasing play should cost LESS relative to the
	// field than at DMP, where its extra gammon chances buy nothing.
	if goLoss > dmpLoss {
		t.Errorf("gammon-go loss (%v) > DMP loss (%v) for the gammonish play — a gammon should be cheaper to chase at gammon-go, never more expensive", goLoss, dmpLoss)
	}
	// Mirroring the score should mirror the effect: gammon-save (the
	// opponent is the one 2-away) should value the gammonish play like DMP
	// does at best, never like gammon-go does.
	if math.Abs(saveLoss-goLoss) < 1e-9 && math.Abs(goLoss-dmpLoss) > 1e-9 {
		t.Errorf("gammon-save loss (%v) matches gammon-go's (%v) rather than DMP's (%v) — the score's SIDE is not being read correctly", saveLoss, goLoss, dmpLoss)
	}
	if moneyLoss == goLoss && moneyLoss != 0 {
		t.Errorf("money loss (%v) equals gammon-go's (%v) — money play looks like it is silently reusing a match state", moneyLoss, goLoss)
	}

	t.Logf("gammonish-play loss: money=%.4f DMP=%.4f gammon-go=%.4f gammon-save=%.4f", moneyLoss, dmpLoss, goLoss, saveLoss)
}

// TestEvaluatePositionDecodesPostCrawfordSentinel guards the bug the score
// sentinel decode fixes (ADR-0016, CONTEXT.md's Away score entry): a
// domain.Position at away=0 ("1-away, post-Crawford") must evaluate, not be
// silently refused because 0 fails MatchState.IsValid()'s "away >= 1". Before
// MatchStateFromPosition this position's cube decision failed with "not
// evaluable at this score" on every post-Crawford 1-away position.
func TestEvaluatePositionDecodesPostCrawfordSentinel(t *testing.T) {
	base, err := domain.DecodeXGID(openingXGID)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	base.PlayerOnRoll = domain.White
	base.Dice = [2]int{0, 0} // no dice: a cube decision

	postCrawford := base
	postCrawford.Score = [2]int{0, 7} // Black 1-away, post-Crawford; White 7-away

	res, err := EvaluatePosition(postCrawford, 0, 0, 0)
	if err != nil {
		t.Fatalf("post-Crawford (score [0,7]) refused: %v — the away=0 sentinel is not being decoded", err)
	}
	if res.Cube == nil {
		t.Fatal("expected a cube decision for a no-dice position")
	}
}
