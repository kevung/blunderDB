package engine

import (
	"math"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

func quizPosition() domain.Position {
	var p domain.Position
	for i := range p.Board.Points {
		p.Board.Points[i] = domain.Point{Checkers: 0, Color: domain.None}
	}
	// Black (moves high → low) has two checkers on 13 and one on 8; White has
	// two on 24. The roll is 6-5, so the plays are few and easy to name.
	p.Board.Points[13] = domain.Point{Checkers: 2, Color: domain.Black}
	p.Board.Points[8] = domain.Point{Checkers: 1, Color: domain.Black}
	p.Board.Points[24] = domain.Point{Checkers: 2, Color: domain.White}
	p.Dice = [2]int{6, 5}
	p.PlayerOnRoll = domain.Black
	p.Score = [2]int{-1, -1}
	return p
}

func errPtr(v float64) *float64 { return &v }

// TestGradeCheckerAnswer_ReadsTheBoardNotANotation pins the choice the file's
// doc states: the answer is the board the user produced. The generator dedups
// by resulting position, so a board names at most one play, and the interface
// never has to spell a move.
func TestGradeCheckerAnswer_ReadsTheBoardNotANotation(t *testing.T) {
	pos := quizPosition()
	plays := domain.LegalMoves(&pos)
	if len(plays) < 2 {
		t.Fatalf("need at least two legal plays, got %d", len(plays))
	}

	ana := &domain.PositionAnalysis{CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
		{Move: plays[0].Notation, EquityError: errPtr(0)},
		{Move: plays[1].Notation, EquityError: errPtr(-0.080)},
	}}}

	best := GradeCheckerAnswer(&pos, ana, plays[0].Result.Board)
	if !best.Legal || !best.Matched {
		t.Fatalf("the best play must be legal and matched: %+v", best)
	}
	if best.ErrorMP != 0 {
		t.Errorf("the best play costs nothing: got %d", best.ErrorMP)
	}
	if best.Best != plays[0].Notation {
		t.Errorf("best answer: got %q, want %q", best.Best, plays[0].Notation)
	}

	// The error is charged as an absolute value: an analysis writing −0.080
	// and one writing 0.080 describe the same 80 millipoints lost.
	blunder := GradeCheckerAnswer(&pos, ana, plays[1].Result.Board)
	if blunder.ErrorMP != 80 {
		t.Errorf("error in millipoints: got %d, want 80", blunder.ErrorMP)
	}
}

// TestGradeCheckerAnswer_IllegalIsNotUnanalysed keeps the two failures apart:
// a board no legal play produces is a rules mistake, a legal play the engine
// never ranked is a gap in the database. Collapsing them would tell a user
// their legal move was illegal.
func TestGradeCheckerAnswer_IllegalIsNotUnanalysed(t *testing.T) {
	pos := quizPosition()
	plays := domain.LegalMoves(&pos)

	var nonsense domain.Board
	for i := range nonsense.Points {
		nonsense.Points[i] = domain.Point{Checkers: 0, Color: domain.None}
	}
	nonsense.Points[3] = domain.Point{Checkers: 3, Color: domain.Black}

	illegal := GradeCheckerAnswer(&pos, nil, nonsense)
	if illegal.Legal {
		t.Error("a board no play produces must not be legal")
	}

	unranked := GradeCheckerAnswer(&pos, &domain.PositionAnalysis{
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{{Move: "24/18 24/19", EquityError: errPtr(0)}}},
	}, plays[0].Result.Board)
	if !unranked.Legal {
		t.Error("a legal play stays legal even when the analysis ignores it")
	}
	if unranked.Matched {
		t.Error("a play the analysis never ranked has no measured error")
	}
}

func TestGradeCubeAnswer(t *testing.T) {
	ana := &domain.PositionAnalysis{DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
		CubefulNoDoubleError:   -0.042,
		CubefulDoubleTakeError: 0,
		CubefulDoublePassError: -0.310,
		BestCubeAction:         "Double, Take",
	}}
	if got := GradeCubeAnswer(ana, CubeActionNoDouble); got.ErrorMP != 42 {
		t.Errorf("no double: got %d, want 42", got.ErrorMP)
	}
	if got := GradeCubeAnswer(ana, CubeActionDoubleTake); got.ErrorMP != 0 {
		t.Errorf("double/take: got %d, want 0", got.ErrorMP)
	}
	if got := GradeCubeAnswer(ana, "hésiter"); got.Legal {
		t.Error("an answer that is not one of the three must be refused")
	}
}

// TestQuizPR is the whole point of the module: the session's number must be on
// the SAME scale as the PR the statistics compute for real play, or comparing
// them — which is what a user will do — compares nothing.
func TestQuizPR(t *testing.T) {
	// One decision losing 0.080 normalised equity: 500 × 80 / 1000 / 1 = 40.
	if got := QuizPR(80, 1); math.Abs(got-40) > 1e-9 {
		t.Errorf("PR: got %v, want 40", got)
	}
	if got := QuizPR(0, 0); got != 0 {
		t.Errorf("no decision gives 0, read with the count: got %v", got)
	}
}
