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

// TestExplain_StaysSilent is the rule the fiche states and P18 turns into a
// principle: a rule speaks only when it is confident. A cheap decision, or one
// whose reason is none of the six, produces NO sentence — gnubg leaves its own
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
