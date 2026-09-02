// SPDX-License-Identifier: MIT

package gammonnet

import (
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
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

	// Score is indexed by PLAYER, and White is the one on roll here, so
	// Score[1] is the mover's own away score. Gammon-go is the score where
	// the MOVER is the trailer chasing the gammon — 4-away against a 2-away
	// leader, i.e. {2, 4}. The mirror, {4, 2}, puts the mover 2-away: the
	// leader, who wants the plain win he is about to double for and has the
	// least use for a gammon of any score here.
	money := atScore([2]int{-1, -1})
	dmp := atScore([2]int{1, 1})        // both 1-away: a gammon is worth nothing extra
	gammonGo := atScore([2]int{2, 4})   // mover 4-away vs a 2-away leader
	gammonSave := atScore([2]int{4, 2}) // mover 2-away: the mirror of gammonGo

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
	// field than at DMP, where its extra gammon chances buy nothing. Since
	// ADR-0023 prices the leaves with the cube this is no longer a near-tie:
	// 8/2 6/2 IS the best play at 4-away/2-away (loss 0), and costs 0.038 at
	// the mirror score — which is what gnubg plays there too.
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

// tooGoodXGID: the on-roll player has a closed board and the opponent has
// two checkers on the bar. Cashing wins one point; playing on wins a gammon
// nearly every time — the textbook "too good to double".
const tooGoodXGID = "XGID=bBBBBBB-C----e-----e--c---:0:0:1:00:0:0:0:0:10"

// tooGoodContactXGID is the ordinary version of the same verdict: a live
// contact position, ~73 % wins with about half of them gammons, centred
// cube, money. No closed board, no checkers on the bar — the kind of
// position a player actually meets, and the kind ADR-0022's plateau made
// unreportable. tooGoodXGID above cleared the plateau only because its
// CUBELESS equity already exceeded a point, which is why it kept passing
// while the model was wrong everywhere else.
//
// Both reference engines, on this position:
//
//	gnubg 0-ply   ND +1.160   DT +1.773   too good / pass (20.7 %)
//	gnubg 2-ply   ND +1.099   DT +1.707   too good / pass (14.0 %)
//	XG Roller++   ND +1.082   DT +1.678   too good / pass (12.1 %)
const tooGoodContactXGID = "XGID=bB-B--C-A---eE---c-caa--B-:0:0:1:00:0:0:0:0:0"

// TestCubeEquitiesAreNormalisedAtEveryScore guards ADR-0019's bug: at a match
// score the panel showed Decide's raw match winning chances as equities
// (the opening position read "No double +0.767" at 3-away/7-away), while the
// cubeless fact beside them was on the search's 2×MWC−1 and money was in
// points — three scales, two of them wrong against the imported XG analyses
// sharing the same column.
//
// The invariant that pins it down needs no reference engine: conceding the
// cube's own value is worth exactly −1 and cashing it exactly +1, at every
// score, which is what "normalised" means.
func TestCubeEquitiesAreNormalisedAtEveryScore(t *testing.T) {
	base, err := domain.DecodeXGID(openingXGID)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	base.Dice = [2]int{0, 0} // no dice: a cube decision

	scores := map[string][2]int{
		"money":         {-1, -1},
		"5-away/5-away": {5, 5},
		"3-away/7-away": {3, 7},
		"2-away/4-away": {2, 4},
		"DMP":           {1, 1},
	}

	var moneyCubeless float64
	cubeless := make(map[string]float64, len(scores))
	for label, score := range scores {
		pos := base
		pos.Score = score
		res, err := EvaluatePosition(pos, 0, 0, 0)
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		if res.Cube == nil {
			t.Fatalf("%s: no cube decision", label)
		}
		if res.PreRoll == nil {
			t.Fatalf("%s: no pre-roll facts", label)
		}
		cubeless[label] = res.PreRoll.CubelessEquity
		if label == "money" {
			moneyCubeless = res.PreRoll.CubelessEquity
		}

		// Dropping is worth exactly one point of the current cube, on every
		// scale worth printing.
		if got := res.Cube.CubefulDoublePassEquity; math.Abs(got-1) > 1e-6 {
			t.Errorf("%s: double/pass = %+.4f, want +1.000 — the equity is not normalised", label, got)
		}
	}

	// The cubeless fact carries no cube, so the score moves it only through
	// the gammon prices — a few hundredths on the opening position. It is
	// the sharpest available check that the scale itself is right: the bug
	// put it on 2×MWC−1, five times too small at an even score, while the
	// three cube equities beside it were on a third scale again.
	for label, eq := range cubeless {
		if math.Abs(eq-moneyCubeless) > 0.15 {
			t.Errorf("%s: cubeless %+.4f is far from money's %+.4f — a scale, not a score effect",
				label, eq, moneyCubeless)
		}
	}
}

// TestTooGoodOnAContactPosition is ADR-0022's regression: the verdict must
// be reachable on a position whose cubeless equity is BELOW a point, which is
// where the flattened live curve made it impossible. Before the fix this
// position reported "Double, Pass" with a no-double equity of +0.995.
//
// Money only: the plateau's signature is that eND stops at the cash
// equivalent, and money is where that equivalent is exactly +1 and the
// comparison is legible.
func TestTooGoodOnAContactPosition(t *testing.T) {
	pos, err := domain.DecodeXGID(tooGoodContactXGID)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	res, err := EvaluatePosition(pos, 2, 0, 0)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	cube := res.Cube
	if cube == nil {
		t.Fatal("expected a cube decision for a no-dice position")
	}

	if res.CubeAction != TooGood {
		t.Errorf("action = %v, want TooGood (ND %+.4f, DT %+.4f, DP %+.4f) — "+
			"gnubg 2-ply and XG both say too good here",
			res.CubeAction, cube.CubefulNoDoubleEquity,
			cube.CubefulDoubleTakeEquity, cube.CubefulDoublePassEquity)
	}
	if !strings.HasPrefix(cube.BestCubeAction, "Too good to double") {
		t.Errorf("best action %q, want a too-good verdict", cube.BestCubeAction)
	}

	// The plateau's fingerprint: cubeless below a point, cubeful above it.
	// The flattened curve could only ever produce the second by way of the
	// first, so a position where they straddle +1 is exactly the one it got
	// wrong. If they ever land on the same side again, this position has
	// stopped exercising the tail and the guard is worth nothing —
	// hence the check, on the pre-roll vector, which is where the cubeless
	// figure actually lives (DoublingCubeAnalysis.CubelessNoDoubleEquity is
	// an importer's field; evaluateCube leaves it zero).
	if res.PreRoll == nil {
		t.Fatal("no pre-roll facts on a cube decision")
	}
	if res.PreRoll.CubelessEquity >= 1.0 {
		t.Errorf("cubeless %+.4f >= 1 — this position no longer exercises the tail",
			res.PreRoll.CubelessEquity)
	}
	if cube.CubefulNoDoubleEquity <= 1.0 {
		t.Errorf("cubeful no-double %+.4f does not beat cashing", cube.CubefulNoDoubleEquity)
	}
}

// TestTooGoodIsReported guards the second half of ADR-0019: the engine had
// been computing the TooGood verdict all along and cubeActionLabel threw it
// away, so a position that is too good to double reported "No Double" —
// indistinguishable from a position that is not good enough to double.
func TestTooGoodIsReported(t *testing.T) {
	base, err := domain.DecodeXGID(tooGoodXGID)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	for _, score := range [][2]int{{-1, -1}, {5, 5}, {3, 7}} {
		pos := base
		pos.Score = score
		res, err := EvaluatePosition(pos, 0, 0, 0)
		if err != nil {
			t.Fatalf("score %v: %v", score, err)
		}
		cube := res.Cube
		if !strings.HasPrefix(cube.BestCubeAction, "Too good to double") {
			t.Errorf("score %v: best action %q, want a too-good verdict (ND %+.4f, DP %+.4f)",
				score, cube.BestCubeAction, cube.CubefulNoDoubleEquity, cube.CubefulDoublePassEquity)
		}
		// Too good means exactly this: playing on beats cashing.
		if cube.CubefulNoDoubleEquity <= cube.CubefulDoublePassEquity {
			t.Errorf("score %v: no-double %+.4f does not beat cashing %+.4f, yet the verdict is %q",
				score, cube.CubefulNoDoubleEquity, cube.CubefulDoublePassEquity, cube.BestCubeAction)
		}
		// …and the label the whole application already reads must decode it.
		verdict, ok := engine.BestCubeVerdict(cube.BestCubeAction)
		if !ok || verdict.ShouldDouble {
			t.Errorf("score %v: %q decodes to %+v, ok=%v — a too-good label must rule against doubling",
				score, cube.BestCubeAction, verdict, ok)
		}
	}

	// Under Jacoby there is no such thing as too good: gammons do not count
	// until the cube has been turned, so the same position is a plain cash.
	jacoby := base
	jacoby.HasJacoby = 1
	res, err := EvaluatePosition(jacoby, 0, 0, 0)
	if err != nil {
		t.Fatalf("jacoby: %v", err)
	}
	if res.Cube.BestCubeAction != "Double, Pass" {
		t.Errorf("with Jacoby the too-good position should cash: got %q", res.Cube.BestCubeAction)
	}
}

// notationForCandidateNaive is the linear rescan notationForCandidate did
// before #150, kept for the A/B below and for the equality check that goes
// with it.
func notationForCandidateNaive(c *Candidate, legal []domain.LegalPlay, opponent int) string {
	for _, play := range legal {
		res := play.Result
		res.PlayerOnRoll = opponent
		gresult, err := FromDomain(&res)
		if err != nil {
			continue
		}
		if gresult == c.Play.Result {
			return play.Notation
		}
	}
	return ""
}

// notationBenchFixture ranks the plays of a position with many of them and
// returns everything the notation phase needs.
func notationBenchFixture(tb testing.TB) ([]Candidate, []domain.LegalPlay, map[Position]string, int) {
	dp, err := domain.DecodeXGID(openingXGID)
	if err != nil {
		tb.Fatal(err)
	}
	dp.PlayerOnRoll = domain.White
	dp.Dice = [2]int{3, 3}
	p, err := FromDomain(&dp)
	if err != nil {
		tb.Fatal(err)
	}
	s, err := NewSearcher(SearchConfig{Ply: 0})
	if err != nil {
		tb.Fatal(err)
	}
	out := make([]Candidate, MaxPlays)
	n, err := s.Plays(&p, 3, 3, out)
	if err != nil || n == 0 {
		tb.Fatalf("no plays: %v", err)
	}
	opponent := domain.Black
	return out[:n], domain.LegalMoves(&dp), notationIndex(domain.LegalMoves(&dp), dp.PlayerOnRoll), opponent
}

// TestNotationIndexMatchesTheLinearScan holds the index to the scan it
// replaces: the same notation for every candidate, empty ones included.
func TestNotationIndexMatchesTheLinearScan(t *testing.T) {
	cands, legal, index, opponent := notationBenchFixture(t)
	for i := range cands {
		got := notationForCandidate(&cands[i], index)
		want := notationForCandidateNaive(&cands[i], legal, opponent)
		if got != want {
			t.Fatalf("candidat %d: %q, balayage %q", i, got, want)
		}
	}
	t.Logf("%d candidats, %d coups légaux", len(cands), len(legal))
}

// BenchmarkNotationPhase is the per-poste figure for the notation lookup
// (#150). Both forms run in the same process, and both include what they
// each have to build: the index for one, nothing for the other.
func BenchmarkNotationPhase(b *testing.B) {
	cands, legal, _, opponent := notationBenchFixture(b)
	dp, _ := domain.DecodeXGID(openingXGID)
	dp.PlayerOnRoll = domain.White
	b.Run("index", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			index := notationIndex(legal, dp.PlayerOnRoll)
			for j := range cands {
				sinkNotation = notationForCandidate(&cands[j], index)
			}
		}
	})
	b.Run("balayage", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			for j := range cands {
				sinkNotation = notationForCandidateNaive(&cands[j], legal, opponent)
			}
		}
	})
}

var sinkNotation string
