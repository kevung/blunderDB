// SPDX-License-Identifier: MIT

package gammonnet

import (
	"math"
	"testing"
)

func probs(win, winG, winBG, loseG, loseBG float32) *[NumOutputs]float32 {
	var p [NumOutputs]float32
	p[PWin] = win
	p[PWinGammon] = winG
	p[PWinBackgammon] = winBG
	p[PLoseGammon] = loseG
	p[PLoseBackgammon] = loseBG
	return &p
}

func closeEnough(t *testing.T, name string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s = %v, want %v (±%v)", name, got, want, tol)
	}
}

// The gammonless, fully-live centred cube take point is the textbook 20 %:
// TP = (L-0.5)/(W+L+0.5) = 0.5/2.5 = 0.2, and the cash point its mirror 0.8.
// A wrong sign or a swapped W/L anywhere upstream shows up here first.
func TestLivePointsTextbookGammonless(t *testing.T) {
	tp, cp := livePoints(1, 1)
	closeEnough(t, "tp_live", tp, 0.2, 1e-12)
	closeEnough(t, "cp_live", cp, 0.8, 1e-12)
}

// At the exact 50/50 gammonless point, a fully live centred cube is worth
// nothing to either side — the dead and live branches agree.
func TestJanowskiEquityEvenGameIsZero(t *testing.T) {
	got := janowskiEquity(0.5, 1, 1, CubeCentred, 1.0)
	closeEnough(t, "equity", got, 0.0, 1e-12)
}

func TestCubeOwnerMirror(t *testing.T) {
	cases := map[CubeOwner]CubeOwner{
		CubeOwned:    CubeOpponent,
		CubeOpponent: CubeOwned,
		CubeCentred:  CubeCentred,
	}
	for owner, want := range cases {
		if got := owner.Mirror(); got != want {
			t.Errorf("%v.Mirror() = %v, want %v", owner, got, want)
		}
	}
	// Mirroring twice is the identity.
	for _, o := range []CubeOwner{CubeOwned, CubeOpponent, CubeCentred} {
		if got := o.Mirror().Mirror(); got != o {
			t.Errorf("%v.Mirror().Mirror() = %v, want %v", o, got, o)
		}
	}
}

func TestCubeInputsFromProbsDegenerateWinIsOne(t *testing.T) {
	// P(win) = 0: every conditional expectation on the winning side is
	// averaged over zero mass and must fall back to 1, never NaN.
	in := CubeInputsFromProbs(probs(0, 0, 0, 0.1, 0.02))
	if in.Win != 0 {
		t.Fatalf("Win = %v, want 0", in.Win)
	}
	if in.WinPoints != 1.0 {
		t.Errorf("WinPoints = %v, want 1 (fallback, not NaN)", in.WinPoints)
	}
	if math.IsNaN(in.LosePoints) {
		t.Errorf("LosePoints is NaN")
	}
}

// A near-certain win, cube centred: the verdict must be to double, and the
// opponent must not have a take.
func TestDecideMoneyNearCertainWinIsDoublePass(t *testing.T) {
	p := probs(0.97, 0.3, 0.02, 0.0, 0.0)
	dec, ok := Decide(p, CubeCentred, nil, DefaultEfficiency(CubeCentred), false)
	if !ok {
		t.Fatal("Decide refused")
	}
	if dec.Action != DoublePass && dec.Action != TooGood {
		t.Errorf("action = %v, want DoublePass or TooGood", dec.Action)
	}
}

// A clear no-double: well below the take point.
func TestDecideMoneyBelowTakePointIsNoDouble(t *testing.T) {
	p := probs(0.55, 0.1, 0.0, 0.05, 0.0)
	dec, ok := Decide(p, CubeCentred, nil, DefaultEfficiency(CubeCentred), false)
	if !ok {
		t.Fatal("Decide refused")
	}
	if dec.Action != NoDouble {
		t.Errorf("action = %v, want NoDouble", dec.Action)
	}
}

// The opponent holding the cube can never be doubled by the player on roll,
// whatever the equities say.
func TestDecideMoneyOpponentOwnedNeverDoubles(t *testing.T) {
	p := probs(0.99, 0.5, 0.1, 0, 0)
	dec, ok := Decide(p, CubeOpponent, nil, DefaultEfficiency(CubeOpponent), false)
	if !ok {
		t.Fatal("Decide refused")
	}
	if dec.Action != NoDouble {
		t.Errorf("action = %v, want NoDouble (cube against)", dec.Action)
	}
}

// Jacoby resets W=L=1 for the "don't double" branch of a CENTRED money cube
// only. It must move that equity and leave an owned/opponent decision alone.
func TestDecideMoneyJacobyAffectsOnlyCentredNoDouble(t *testing.T) {
	p := probs(0.6, 0.25, 0.05, 0.05, 0.0)
	eff := DefaultEfficiency(CubeCentred)

	without, ok := Decide(p, CubeCentred, nil, eff, false)
	if !ok {
		t.Fatal("Decide refused")
	}
	with, ok := Decide(p, CubeCentred, nil, eff, true)
	if !ok {
		t.Fatal("Decide refused")
	}
	if without.EquityNoDouble == with.EquityNoDouble {
		t.Errorf("Jacoby did not move the centred no-double equity (gammons present in probs)")
	}

	effOwned := DefaultEfficiency(CubeOwned)
	ownedWithout, _ := Decide(p, CubeOwned, nil, effOwned, false)
	ownedWith, _ := Decide(p, CubeOwned, nil, effOwned, true)
	if ownedWithout.EquityNoDouble != ownedWith.EquityNoDouble {
		t.Errorf("Jacoby moved an OWNED cube's no-double equity; it must only affect a centred cube")
	}
}

func TestMatchStateIsValid(t *testing.T) {
	cases := []struct {
		name string
		s    MatchState
		want bool
	}{
		{"ordinary", MatchState{AwayOnRoll: 3, AwayOpponent: 5, Cube: 2}, true},
		{"zero away", MatchState{AwayOnRoll: 0, AwayOpponent: 5, Cube: 1}, false},
		{"negative away", MatchState{AwayOnRoll: 3, AwayOpponent: -1, Cube: 1}, false},
		{"non power of two cube", MatchState{AwayOnRoll: 3, AwayOpponent: 5, Cube: 3}, false},
		{"beyond horizon", MatchState{AwayOnRoll: matchMaxAway + 1, AwayOpponent: 5, Cube: 1}, false},
		{"at horizon", MatchState{AwayOnRoll: matchMaxAway, AwayOpponent: 5, Cube: 1}, true},
		{"crawford with nobody at match point", MatchState{AwayOnRoll: 2, AwayOpponent: 4, Cube: 1, Crawford: true}, false},
		{"crawford, on-roll at match point", MatchState{AwayOnRoll: 1, AwayOpponent: 5, Cube: 1, Crawford: true}, true},
		{"crawford, opponent at match point", MatchState{AwayOnRoll: 5, AwayOpponent: 1, Cube: 1, Crawford: true}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.s.IsValid(); got != c.want {
				t.Errorf("IsValid() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestMatchStateSwap(t *testing.T) {
	s := MatchState{AwayOnRoll: 3, AwayOpponent: 7, Cube: 4, Crawford: true}
	got := s.Swap()
	want := MatchState{AwayOnRoll: 7, AwayOpponent: 3, Cube: 4, Crawford: true}
	if got != want {
		t.Errorf("Swap() = %+v, want %+v", got, want)
	}
}

// Acceptance criterion (#122): a pair of decisions identical except for the
// score must be able to disagree. This is the one test that catches a score
// silently ignored anywhere between the codec, the recursion, and the MET.
func TestDecideMatchScoreChangesTheVerdict(t *testing.T) {
	p := probs(0.72, 0.28, 0.04, 0.03, 0.0)
	eff := DefaultEfficiency(CubeCentred)

	leaderTwoAway := MatchState{AwayOnRoll: 2, AwayOpponent: 4, Cube: 1}
	trailerFourAway := MatchState{AwayOnRoll: 4, AwayOpponent: 2, Cube: 1}

	a, ok := Decide(p, CubeCentred, &leaderTwoAway, eff, false)
	if !ok {
		t.Fatal("Decide refused (leader 2-away)")
	}
	b, ok := Decide(p, CubeCentred, &trailerFourAway, eff, false)
	if !ok {
		t.Fatal("Decide refused (trailer 4-away)")
	}

	if a.Action == b.Action && closeEqual(a.EquityNoDouble, b.EquityNoDouble) &&
		closeEqual(a.EquityDouble, b.EquityDouble) {
		t.Fatalf("2-away/4-away and 4-away/2-away produced the same decision "+
			"for the same distribution: %+v == %+v -- the score is being ignored", a, b)
	}
}

func closeEqual(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// The Crawford game itself has no cube in play, by rule -- regardless of what
// the equities say, and regardless of whether either away score happens to
// be 1. This is exactly the kind of fact that gets lost in silence if the
// flag is dropped anywhere on the way from a domain.Position.
func TestDecideMatchCrawfordForcesNoDouble(t *testing.T) {
	// The trailer (on roll, 5-away) against a leader at match point (1-away):
	// textbook post-Crawford theory says the trailer should virtually always
	// double. During the Crawford game itself, though, the cube is dead by
	// rule regardless of what the equities say — that is what the flag alone
	// must produce here, with every other input held fixed.
	p := probs(0.7, 0.2, 0.03, 0.05, 0.0)
	eff := DefaultEfficiency(CubeCentred)
	state := MatchState{AwayOnRoll: 5, AwayOpponent: 1, Cube: 1, Crawford: true}

	dec, ok := Decide(p, CubeCentred, &state, eff, false)
	if !ok {
		t.Fatal("Decide refused")
	}
	if dec.Action != NoDouble {
		t.Errorf("Crawford game action = %v, want NoDouble", dec.Action)
	}

	// Same away scores (still post-Crawford in the MET's eyes), but not
	// flagged as the Crawford game itself: the cube is live again, and the
	// trailer should double.
	post := state
	post.Crawford = false
	postDec, ok := Decide(p, CubeCentred, &post, eff, false)
	if !ok {
		t.Fatal("Decide refused")
	}
	if postDec.Action == NoDouble {
		t.Errorf("post-Crawford decision unexpectedly NoDouble for the trailer at 70%% "+
			"vs. a leader at match point; Crawford flag may be leaking")
	}
}

// The opponent can never be doubled in match play either.
func TestDecideMatchOpponentOwnedNeverDoubles(t *testing.T) {
	p := probs(0.95, 0.4, 0.05, 0, 0)
	state := MatchState{AwayOnRoll: 3, AwayOpponent: 5, Cube: 2}
	dec, ok := Decide(p, CubeOpponent, &state, DefaultEfficiency(CubeOpponent), false)
	if !ok {
		t.Fatal("Decide refused")
	}
	if dec.Action != NoDouble {
		t.Errorf("action = %v, want NoDouble (cube against)", dec.Action)
	}
}

// build_levels must terminate (reach a dead level) for ordinary match
// scores, at every cube value actually seen in play.
func TestBuildLevelsTerminatesForOrdinaryScores(t *testing.T) {
	p := probs(0.55, 0.15, 0.02, 0.1, 0.01)
	outcomes := probsExclusive(p)
	for _, cube := range []int{1, 2, 4, 8, 16} {
		for away := 1; away <= 15; away++ {
			state := MatchState{AwayOnRoll: away, AwayOpponent: away + 1, Cube: cube}
			_, count := buildLevels(state, outcomes)
			if count < 2 {
				t.Errorf("buildLevels(away=%d, cube=%d) refused", away, cube)
			}
		}
	}
}

func TestValueRefusesInvalidState(t *testing.T) {
	p := probs(0.6, 0.1, 0, 0.1, 0)
	bad := MatchState{AwayOnRoll: 0, AwayOpponent: 3, Cube: 1}
	if _, ok := Value(p, CubeCentred, &bad, 0.7); ok {
		t.Error("Value accepted an invalid MatchState")
	}
	if _, ok := Decide(p, CubeCentred, &bad, 0.7, false); ok {
		t.Error("Decide accepted an invalid MatchState")
	}
}

// Value must stay inside the [-1, 1] match-equity band it is documented to
// return (2*MWC-1), for any owner, at an ordinary score.
func TestValueMatchWithinEquityBand(t *testing.T) {
	p := probs(0.6, 0.2, 0.03, 0.08, 0.01)
	state := MatchState{AwayOnRoll: 4, AwayOpponent: 6, Cube: 2}
	for _, owner := range []CubeOwner{CubeCentred, CubeOwned, CubeOpponent} {
		v, ok := Value(p, owner, &state, DefaultEfficiency(owner))
		if !ok {
			t.Fatalf("Value refused for owner %v", owner)
		}
		if v < -1.0001 || v > 1.0001 {
			t.Errorf("Value(%v) = %v, outside [-1, 1]", owner, v)
		}
	}
}
