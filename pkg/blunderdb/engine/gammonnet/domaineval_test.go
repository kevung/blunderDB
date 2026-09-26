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

// TestEvaluateMovesSetsEquityError: evaluateMoves fills EquityError as
// ingest/merge.go does — nil for the best move, a non-negative
// bestEquity-equity for every other one.
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

// TestEvaluatePositionHonoursTheScore (ADR-0016): pos.Score must reach the
// checker-move search, so the gammonish 6-4 play (6/2 8/2) is valued
// differently at DMP, where a gammon is worth nothing extra, than at
// gammon-go.
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

	// Score is indexed by PLAYER and White is on roll, so gammon-go (the
	// mover 4-away chasing the gammon against a 2-away leader) is {2, 4}.
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
	// At gammon-go the gammon-chasing play should cost LESS than at DMP.
	// With cubeful leaves (ADR-0023) 8/2 6/2 is the best play at
	// 4-away/2-away, as gnubg plays it.
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

// TestEvaluatePositionDecodesPostCrawfordSentinel: a domain.Position at
// away=0 ("1-away, post-Crawford", CONTEXT.md) must evaluate, not be refused
// by MatchState.IsValid()'s "away >= 1".
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
// cube, money, cubeless equity below a point (unlike tooGoodXGID) — the
// case ADR-0022's plateau made unreportable.
//
// Both reference engines, on this position:
//
//	gnubg 0-ply   ND +1.160   DT +1.773   too good / pass (20.7 %)
//	gnubg 2-ply   ND +1.099   DT +1.707   too good / pass (14.0 %)
//	XG Roller++   ND +1.082   DT +1.678   too good / pass (12.1 %)
const tooGoodContactXGID = "XGID=bB-B--C-A---eE---c-caa--B-:0:0:1:00:0:0:0:0:0"

// TestCubeEquitiesAreNormalisedAtEveryScore (ADR-0019): every equity that
// leaves the package is normalised. The invariant needs no reference engine:
// conceding the cube's value is worth exactly −1 and cashing it +1, at every
// score.
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

	// The cubeless fact moves with the score only through the gammon prices
	// (a few hundredths): the sharpest check that its scale is right, since
	// 2×MWC−1 would be five times too small at an even score.
	for label, eq := range cubeless {
		if math.Abs(eq-moneyCubeless) > 0.15 {
			t.Errorf("%s: cubeless %+.4f is far from money's %+.4f — a scale, not a score effect",
				label, eq, moneyCubeless)
		}
	}
}

// TestTooGoodOnAContactPosition (ADR-0022): TooGood must be reachable on a
// position whose cubeless equity is BELOW a point. Money only, where the
// cash equivalent is exactly +1.
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

	// The fixture must straddle +1 (cubeless below, cubeful above) or it no
	// longer exercises the tail. The cubeless figure lives in the pre-roll
	// vector (evaluateCube leaves CubelessNoDoubleEquity zero).
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

// TestTooGoodIsReported (ADR-0019): a too-good position must not be labelled
// "No Double".
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

// notationForCandidateNaive is the linear rescan notationIndex replaces, kept
// for the A/B below and its equality check.
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

// BenchmarkNotationPhase compares the notation lookup's two forms, each
// including what it has to build.
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

// TestCrawfordAtDoubleMatchPoint pins the [1,1] score, read as the Crawford
// game: the decision must be evaluable, must not double, must price the
// double branch at exactly the no-double value, and its no-double equity
// must be the dead (cubeless) one.
func TestCrawfordAtDoubleMatchPoint(t *testing.T) {
	pos, err := domain.DecodeXGID(openingXGID)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	pos.PlayerOnRoll = domain.White
	pos.Dice = [2]int{0, 0}
	pos.Score = [2]int{1, 1}

	state, ok := MatchStateFromPosition(&pos)
	if !ok {
		t.Fatal("[1,1] refused by MatchStateFromPosition")
	}
	if want := (MatchState{AwayOnRoll: 1, AwayOpponent: 1, Cube: 1, Crawford: true}); state != want {
		t.Fatalf("[1,1] decoded as %+v, want %+v", state, want)
	}

	res, err := EvaluatePosition(pos, 0, 0, 0)
	if err != nil {
		t.Fatalf("[1,1]: %v", err)
	}
	if res.Cube == nil || res.PreRoll == nil {
		t.Fatal("[1,1]: no cube decision or no pre-roll facts")
	}
	if res.CubeAction != NoDouble {
		t.Errorf("[1,1]: action %v, want NoDouble — there is no cube in the Crawford game", res.CubeAction)
	}
	if res.Cube.BestCubeAction != "No Double" {
		t.Errorf("[1,1]: BestCubeAction %q, want %q", res.Cube.BestCubeAction, "No Double")
	}
	nd, dt := res.Cube.CubefulNoDoubleEquity, res.Cube.CubefulDoubleTakeEquity
	if nd != dt {
		t.Errorf("[1,1]: double/take %v priced away from no-double %v", dt, nd)
	}
	if res.Cube.CubefulNoDoubleError != 0 || res.Cube.CubefulDoubleTakeError != 0 {
		t.Errorf("[1,1]: errors %v/%v, want 0/0 — neither branch can be a mistake when they are the same number", res.Cube.CubefulNoDoubleError, res.Cube.CubefulDoubleTakeError)
	}
	// The dead value: at DMP the normalised no-double equity IS the cubeless
	// equity — same distribution, same 2×MWC−1, same scale.
	if math.Abs(nd-res.PreRoll.CubelessEquity) > 1e-9 {
		t.Errorf("[1,1]: no-double %v is not the cubeless equity %v — the Crawford game is valued at its dead value", nd, res.PreRoll.CubelessEquity)
	}
	if nd <= -1 || nd >= 1 {
		t.Errorf("[1,1]: equity %v outside (-1, 1)", nd)
	}

	// Directly on the model, without the search: same three facts.
	probs := [NumOutputs]float32{0.55, 0.15, 0.02, 0.1, 0.01}
	dec, ok := Decide(&probs, CubeCentred, &state, DefaultEfficiency(CubeCentred), false)
	if !ok {
		t.Fatal("Decide refused DMP")
	}
	if dec.Action != NoDouble || dec.EquityDouble != dec.EquityNoDouble || dec.EquityDoubleTake != dec.EquityNoDouble {
		t.Errorf("Decide at DMP: %+v", dec)
	}
	// Every stake is the match at DMP, so the MWC is the win probability
	// itself, on any gammon mix.
	if math.Abs(dec.EquityNoDouble-float64(probs[PWin])) > 1e-6 {
		t.Errorf("DMP no-double MWC %v, want the win probability %v", dec.EquityNoDouble, probs[PWin])
	}
}
