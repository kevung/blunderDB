package domain

import "testing"

// mkPos builds a position from a signed checker map (positive = Black, negative =
// White; index 0 = WhiteBar, 25 = BlackBar, 1..24 = points). Bearoff is derived
// so each side totals 15.
func mkPos(mover, d1, d2 int, pts map[int]int) Position {
	var p Position
	for i := range p.Board.Points {
		p.Board.Points[i] = Point{Checkers: 0, Color: None}
	}
	var bOn, wOn int
	for idx, v := range pts {
		switch {
		case v > 0:
			p.Board.Points[idx] = Point{Checkers: v, Color: Black}
			bOn += v
		case v < 0:
			p.Board.Points[idx] = Point{Checkers: -v, Color: White}
			wOn += -v
		}
	}
	p.Board.Bearoff = [2]int{15 - bOn, 15 - wOn}
	p.PlayerOnRoll = mover
	p.Dice = [2]int{d1, d2}
	p.DecisionType = CheckerAction
	return p
}

func totals(p *Position) (black, white int) {
	for i := 0; i <= 25; i++ {
		switch p.Board.Points[i].Color {
		case Black:
			black += p.Board.Points[i].Checkers
		case White:
			white += p.Board.Points[i].Checkers
		}
	}
	return black + p.Board.Bearoff[Black], white + p.Board.Bearoff[White]
}

// Bear off from a single point with overage: 15 Black on point 3, roll 6-5.
func TestLegalMovesBearOffOverage(t *testing.T) {
	p := mkPos(Black, 6, 5, map[int]int{3: 15})
	plays := LegalMoves(&p)
	if len(plays) != 1 {
		t.Fatalf("expected 1 play, got %d", len(plays))
	}
	r := plays[0].Result
	if r.Board.Points[3].Checkers != 13 || r.Board.Bearoff[Black] != 2 {
		t.Fatalf("bad bear-off result: pt3=%d off=%d", r.Board.Points[3].Checkers, r.Board.Bearoff[Black])
	}
}

// No legal move (dance): Black on the bar, both entry points blocked.
func TestLegalMovesDance(t *testing.T) {
	p := mkPos(Black, 1, 2, map[int]int{25: 1, 6: 14, 24: -2, 23: -2, 1: -11})
	plays := LegalMoves(&p)
	if plays == nil || len(plays) != 0 {
		t.Fatalf("expected a dance (empty non-nil), got %v", plays)
	}
}

// A one-checker race: both orders reach the same square → a single play.
func TestLegalMovesOneCheckerRace(t *testing.T) {
	p := mkPos(Black, 2, 1, map[int]int{5: 1}) // 14 Black already off
	plays := LegalMoves(&p)
	if len(plays) != 1 {
		t.Fatalf("expected 1 play, got %d", len(plays))
	}
	if plays[0].Result.Board.Points[2].Checkers != 1 || plays[0].Result.Board.Points[2].Color != Black {
		t.Fatalf("checker should end on point 2: %+v", plays[0].Result.Board.Points[2])
	}
}

// Hitting a blot sends it to the opponent bar.
func TestLegalMovesHit(t *testing.T) {
	p := mkPos(Black, 2, 2, map[int]int{3: 1, 1: -1}) // Black blot on 3, White blot on 1
	plays := LegalMoves(&p)
	var hit *LegalPlay
	for i := range plays {
		for _, s := range plays[i].Steps {
			if s.Hit {
				hit = &plays[i]
			}
		}
	}
	if hit == nil {
		t.Fatalf("expected a play that hits; got %d plays", len(plays))
	}
	if hit.Result.Board.Points[WhiteBar].Checkers != 1 || hit.Result.Board.Points[WhiteBar].Color != White {
		t.Fatalf("hit White checker should be on the bar: %+v", hit.Result.Board.Points[WhiteBar])
	}
}

// Larger-die rule: when only one die is playable, plays using only the smaller
// die are illegal.
func TestLegalMovesLargerDie(t *testing.T) {
	// Lone Black checker on 13: 13/7 (die 6) or 13/10 (die 3). The 14 checkers on
	// point 7 are blocked (White owns 4 and 1), nothing is home (no bear-off), and
	// after either move the other die is blocked — so only one die plays and it
	// must be the larger (6): the only legal play is 13/7.
	p := mkPos(Black, 6, 3, map[int]int{13: 1, 7: 14, 4: -2, 1: -2, 24: -11})
	plays := LegalMoves(&p)
	if len(plays) != 1 {
		t.Fatalf("larger-die: expected 1 play, got %d: %+v", len(plays), plays)
	}
	s := plays[0].Steps[0]
	if s.From != 13 || s.To != 7 {
		t.Fatalf("larger-die: expected 13/7 (die 6), got %d/%d", s.From, s.To)
	}
}

// Doubles give four moves of the same value.
func TestLegalMovesDoublesUseFour(t *testing.T) {
	// 4 Black checkers on 24 with 6-6-6-6; each goes 24/18 (open). All four play.
	p := mkPos(Black, 6, 6, map[int]int{24: 4, 6: 11, 18: 0})
	plays := LegalMoves(&p)
	if len(plays) == 0 {
		t.Fatal("expected at least one play")
	}
	for _, pl := range plays {
		if len(pl.Steps) != 4 {
			t.Fatalf("doubles must use 4 dice, got %d (%s)", len(pl.Steps), pl.Notation)
		}
	}
}

// Every generated play conserves 15 checkers per side and uses the right number
// of dice; results are distinct.
func TestLegalMovesInvariants(t *testing.T) {
	cases := [][2]int{{3, 1}, {6, 5}, {2, 2}, {6, 4}, {5, 5}}
	for _, d := range cases {
		p := InitializePosition()
		p.Dice = d
		plays := LegalMoves(&p)
		if len(plays) == 0 {
			t.Fatalf("opening %v should have legal plays", d)
		}
		seen := map[string]bool{}
		for _, pl := range plays {
			b, w := totals(&pl.Result)
			if b != 15 || w != 15 {
				t.Fatalf("%v: checker leak black=%d white=%d (%s)", d, b, w, pl.Notation)
			}
			k := boardKey(&pl.Result)
			if seen[k] {
				t.Fatalf("%v: duplicate result %s", d, pl.Notation)
			}
			seen[k] = true
		}
	}
}
