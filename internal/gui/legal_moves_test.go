package gui

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// mkPos builds a position from a signed checker map (positive = Black,
// negative = White; 1..24 are the points). Bearoff is derived so each side
// totals fifteen — the same shape domain's own move tests use, rewritten here
// because that helper is unexported.
func mkPos(mover, d1, d2 int, pts map[int]int) domain.Position {
	var p domain.Position
	for i := range p.Board.Points {
		p.Board.Points[i] = domain.Point{Checkers: 0, Color: domain.None}
	}
	var black, white int
	for idx, v := range pts {
		switch {
		case v > 0:
			p.Board.Points[idx] = domain.Point{Checkers: v, Color: domain.Black}
			black += v
		case v < 0:
			p.Board.Points[idx] = domain.Point{Checkers: -v, Color: domain.White}
			white += -v
		}
	}
	p.Board.Bearoff = [2]int{15 - black, 15 - white}
	p.PlayerOnRoll = mover
	p.Dice = [2]int{d1, d2}
	p.DecisionType = domain.CheckerAction
	return p
}

// The binding is a one-liner, so what is worth testing is the shape the
// frontend relies on: nil when there is no question to ask, an EMPTY slice
// when there is no answer to give, and plays that carry both their steps and
// their name.
func TestLegalMovesShapeForTheFrontend(t *testing.T) {
	a := &App{}

	t.Run("no dice asks no question, and that is nil", func(t *testing.T) {
		pos := mkPos(domain.Black, 0, 0, map[int]int{6: 5, 8: 3, 13: 5, 24: 2})
		if got := a.LegalMoves(pos); got != nil {
			t.Fatalf("got %d plays, want nil", len(got))
		}
	})

	t.Run("a dance has no answer, and that is empty, not nil", func(t *testing.T) {
		// Black on the bar, every entry point held by two White checkers.
		pts := map[int]int{25: 1}
		for pt := 19; pt <= 24; pt++ {
			pts[pt] = -2
		}
		got := a.LegalMoves(mkPos(domain.Black, 3, 1, pts))
		if got == nil {
			t.Fatal("a dance must be an empty slice: the interface tells it from a position with no dice")
		}
		if len(got) != 0 {
			t.Fatalf("got %d plays for a dance, want 0", len(got))
		}
	})

	t.Run("each play carries its steps and its name", func(t *testing.T) {
		plays := a.LegalMoves(mkPos(domain.Black, 3, 1, map[int]int{6: 5, 8: 3, 13: 5, 24: 2}))
		if len(plays) == 0 {
			t.Fatal("31 on the opening position has legal plays")
		}
		for _, p := range plays {
			if p.Notation == "" {
				t.Errorf("a play the interface offers must be nameable: %+v", p.Steps)
			}
			if len(p.Steps) == 0 {
				t.Errorf("a play must carry its steps — that is what the board replays: %q", p.Notation)
			}
		}
	})

	t.Run("the position asked about is not moved", func(t *testing.T) {
		pos := mkPos(domain.Black, 3, 1, map[int]int{6: 5, 8: 3, 13: 5, 24: 2})
		before := pos.Board
		a.LegalMoves(pos)
		if pos.Board != before {
			t.Fatal("LegalMoves must not move the checkers of the position it is asked about")
		}
	})
}
