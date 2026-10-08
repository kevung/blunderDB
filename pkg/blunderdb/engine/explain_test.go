package engine

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

func cubeAna(nd, dt, dp float64, best string) *domain.PositionAnalysis {
	return &domain.PositionAnalysis{DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
		CubefulNoDoubleError:   nd,
		CubefulDoubleTakeError: dt,
		CubefulDoublePassError: dp,
		BestCubeAction:         best,
	}}
}

// TestExplainCube_NamesTheDirection is what a player learns from: not that the
// cube decision was wrong, but which way.
func TestExplainCube_NamesTheDirection(t *testing.T) {
	late := ExplainCube(cubeAna(-0.120, 0, -0.300, "Double, Take"), CubeActionNoDouble)
	if late.Theme != "doubletoolate" {
		t.Errorf("missed double: got %q", late.Theme)
	}
	if late.CostMP != 120 {
		t.Errorf("cost: got %d, want 120", late.CostMP)
	}

	early := ExplainCube(cubeAna(0, -0.090, -0.400, "No double"), CubeActionDoubleTake)
	if early.Theme != "doubletooearly" {
		t.Errorf("premature double: got %q", early.Theme)
	}

	loose := ExplainCube(cubeAna(-0.500, -0.150, 0, "Double, Pass"), CubeActionDoubleTake)
	if loose.Theme != "taketooloose" {
		t.Errorf("loose take: got %q", loose.Theme)
	}

	tight := ExplainCube(cubeAna(-0.500, 0, -0.200, "Double, Take"), CubeActionDoublePass)
	if tight.Theme != "passtootight" {
		t.Errorf("tight pass: got %q", tight.Theme)
	}
}

// TestExplain_StaysSilent: a rule speaks only when it is confident. A cheap
// decision, or one whose reason is none of the six, produces NO sentence — gnubg leaves its own
// analysis menu empty in exactly that case.
func TestExplain_StaysSilent(t *testing.T) {
	// Below the speaking threshold: right answer, nothing to say.
	cheap := ExplainCube(cubeAna(-0.010, 0, -0.300, "Double, Take"), CubeActionNoDouble)
	if cheap.Theme != "" {
		t.Errorf("a 10-millipoint error is not worth a sentence: got %q", cheap.Theme)
	}
	// No analysis at all.
	if got := ExplainCube(nil, CubeActionNoDouble); got.Theme != "" {
		t.Errorf("no analysis, no sentence: got %q", got.Theme)
	}
	if got := ExplainChecker(nil, nil, "13/7"); got.Theme != "" {
		t.Errorf("no position, no sentence: got %q", got.Theme)
	}
	// A cube answer that is not one of the three.
	if got := ExplainCube(cubeAna(-0.500, 0, -0.200, "Double, Take"), "hésiter"); got.Theme != "" {
		t.Errorf("an answer outside the three has no direction: got %q", got.Theme)
	}
}

// TestExplainChecker_GammonBeforeBoard pins the order: the gammon swing is
// read straight off the analysis and cannot be wrong about what it saw, so it
// is tested before anything that has to reconstruct a board.
func TestExplainChecker_GammonBeforeBoard(t *testing.T) {
	pos := quizPosition()
	plays := domain.LegalMoves(&pos)
	if len(plays) < 2 {
		t.Fatalf("need two plays, got %d", len(plays))
	}
	ana := &domain.PositionAnalysis{CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
		{Move: plays[0].Notation, EquityError: errPtr(0), PlayerGammonChance: 30},
		{Move: plays[1].Notation, EquityError: errPtr(-0.150), PlayerGammonChance: 18},
	}}}
	got := ExplainChecker(&pos, ana, plays[1].Notation)
	if got.Theme != "gammon" {
		t.Fatalf("theme: got %q, want gammon", got.Theme)
	}
	if got.GammonPct != 12 {
		t.Errorf("gammon swing: got %d, want 12", got.GammonPct)
	}
	if got.Best != plays[0].Notation {
		t.Errorf("best: got %q, want %q", got.Best, plays[0].Notation)
	}
}

// TestExplainChecker_NoThemeStillMeansSilence: the cost can be real and the
// reason still be none of the six. Saying "you lost 120 millipoints, and we do
// not know why" would be worse than saying nothing — it teaches nothing and
// reads as a failure.
func TestExplainChecker_NoThemeStillMeansSilence(t *testing.T) {
	pos := quizPosition()
	plays := domain.LegalMoves(&pos)
	ana := &domain.PositionAnalysis{CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
		{Move: plays[0].Notation, EquityError: errPtr(0), PlayerGammonChance: 20, PlayerWinChance: 55},
		{Move: plays[1].Notation, EquityError: errPtr(-0.120), PlayerGammonChance: 20, PlayerWinChance: 54},
	}}}
	got := ExplainChecker(&pos, ana, plays[1].Notation)
	if got.Theme != "" && got.Theme != "blots" && got.Theme != "point" && got.Theme != "passive" {
		t.Errorf("unexpected theme %q", got.Theme)
	}
}

// TestBlotDeviation_SignsAgainstTheBest: the sign follows the blots the played
// move leaves against the best move's, the best move itself reads 0, and a move
// the generator does not produce is left out.
func TestBlotDeviation_SignsAgainstTheBest(t *testing.T) {
	pos := quizPosition()
	// Two White checkers still behind Black's: contact, so blots matter.
	pos.Board.Points[1] = domain.Point{Checkers: 2, Color: domain.White}
	plays := domain.LegalMoves(&pos)
	blots := func(i int) int { return countBlots(&plays[i].Result.Board, pos.PlayerOnRoll) }
	lo, hi := -1, -1
	for i := range plays {
		if lo < 0 || blots(i) < blots(lo) {
			lo = i
		}
		if hi < 0 || blots(i) > blots(hi) {
			hi = i
		}
	}
	if blots(lo) == blots(hi) {
		t.Fatalf("need two plays leaving different blots")
	}
	ana := func(best, other int) *domain.PositionAnalysis {
		return &domain.PositionAnalysis{CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: plays[best].Notation, EquityError: errPtr(0)},
			{Move: plays[other].Notation, EquityError: errPtr(-0.080)},
		}}}
	}
	if s, ok := BlotDeviation(&pos, ana(lo, hi), plays[hi].Notation); !ok || s != 1 {
		t.Errorf("bolder than the best: got %d, %v; want 1, true", s, ok)
	}
	if s, ok := BlotDeviation(&pos, ana(hi, lo), plays[lo].Notation); !ok || s != -1 {
		t.Errorf("more cautious than the best: got %d, %v; want -1, true", s, ok)
	}
	if s, ok := BlotDeviation(&pos, ana(lo, hi), plays[lo].Notation); !ok || s != 0 {
		t.Errorf("the best move: got %d, %v; want 0, true", s, ok)
	}
	if _, ok := BlotDeviation(&pos, ana(lo, hi), "25/1"); ok {
		t.Error("a move the analysis does not name has nothing to compare")
	}
	race := quizPosition()
	if _, ok := BlotDeviation(&race, ana(lo, hi), plays[hi].Notation); ok {
		t.Error("without contact a blot risks nothing: left out")
	}
}

// TestBlotDeviation_ReadsEveryNotationDialect: an analysis spells doubles and
// hits its own way ("13/7(2)", "8/5*/4") where the generator writes each step;
// the play is still replayed, matched by the board it leaves. Dropping them
// would leave doubles and hits — the bold plays — out of the blots bias.
func TestBlotDeviation_ReadsEveryNotationDialect(t *testing.T) {
	ana := func(best, played string) *domain.PositionAnalysis {
		return &domain.PositionAnalysis{CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: best, EquityError: errPtr(0)},
			{Move: played, EquityError: errPtr(-0.080)},
		}}}
	}
	doubles := domain.InitializePosition()
	doubles.Dice = [2]int{3, 3}
	if s, ok := BlotDeviation(&doubles, ana("8/5(2) 6/3(2)", "24/18 13/10 8/5"), "24/18 13/10 8/5"); !ok || s != 1 {
		t.Errorf("chained hop: got %d, %v; want 1, true", s, ok)
	}
	if s, ok := BlotDeviation(&doubles, ana("24/18 13/10 8/5", "13/7(2)"), "13/7(2)"); !ok || s != -1 {
		t.Errorf("chained doubles: got %d, %v; want -1, true", s, ok)
	}

	hit := domain.InitializePosition()
	hit.Dice = [2]int{3, 1}
	hit.Board.Points[12] = domain.Point{Checkers: 4, Color: domain.White}
	hit.Board.Points[5] = domain.Point{Checkers: 1, Color: domain.White}
	if s, ok := BlotDeviation(&hit, ana("8/5* 6/5", "8/5*/4"), "8/5*/4"); !ok || s != 1 {
		t.Errorf("hit on the way: got %d, %v; want 1, true", s, ok)
	}
	if s, ok := BlotDeviation(&hit, ana("8/5*/4", "6/5* 8/5"), "6/5* 8/5"); !ok || s != -1 {
		t.Errorf("hit in another order: got %d, %v; want -1, true", s, ok)
	}
}
