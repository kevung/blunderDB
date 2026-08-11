package domain

import "testing"

// posPts is mkPos without caring about mover/dice, for predicates that only
// look at the board. Reuses mkPos (moves_test.go) for the checker/bearoff
// bookkeeping.
func posPts(pts map[int]int) Position {
	return mkPos(Black, 0, 0, pts)
}

// --- MatchesScorePosition -------------------------------------------------

func TestMatchesScorePosition(t *testing.T) {
	cases := []struct {
		name        string
		score       [2]int
		filterScore [2]int
		want        bool
	}{
		{"exact match", [2]int{7, 5}, [2]int{7, 5}, true},
		{"player1 differs", [2]int{7, 5}, [2]int{6, 5}, false},
		{"player2 differs", [2]int{7, 5}, [2]int{7, 4}, false},
		{"swapped scores don't match", [2]int{7, 5}, [2]int{5, 7}, false},
		{"both unlimited (-1,-1)", [2]int{-1, -1}, [2]int{-1, -1}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := Position{Score: c.score}
			filter := Position{Score: c.filterScore}
			if got := p.MatchesScorePosition(filter); got != c.want {
				t.Errorf("MatchesScorePosition(score=%v, filter=%v) = %v, want %v", c.score, c.filterScore, got, c.want)
			}
		})
	}
}

// --- Mirror ----------------------------------------------------------------

func TestMirror(t *testing.T) {
	p := posPts(map[int]int{1: 2, 24: -5, 12: 3})
	p.Board.Bearoff = [2]int{4, 6}
	p.PlayerOnRoll = Black
	p.Score = [2]int{7, 3}
	p.Cube = Cube{Owner: Black, Value: 2}

	m := p.Mirror()

	// Point i's content moves to 25-i, with color flipped.
	if m.Board.Points[24].Checkers != 2 || m.Board.Points[24].Color != White {
		t.Errorf("point 1 (2 Black) should mirror to point 24 as White: got %+v", m.Board.Points[24])
	}
	if m.Board.Points[1].Checkers != 5 || m.Board.Points[1].Color != Black {
		t.Errorf("point 24 (5 White) should mirror to point 1 as Black: got %+v", m.Board.Points[1])
	}
	if m.Board.Points[13].Checkers != 3 || m.Board.Points[13].Color != White {
		t.Errorf("point 12 (3 Black) should mirror to point 13 as White: got %+v", m.Board.Points[13])
	}
	// An empty point stays empty and its color is left untouched (None).
	if m.Board.Points[2].Checkers != 0 {
		t.Errorf("empty point 23 (mirror of empty point 2) should stay empty: got %+v", m.Board.Points[23])
	}

	if m.Board.Bearoff[0] != 6 || m.Board.Bearoff[1] != 4 {
		t.Errorf("bearoff should swap: got %v, want [6 4]", m.Board.Bearoff)
	}
	if m.PlayerOnRoll != White {
		t.Errorf("player on roll should flip: got %d, want %d", m.PlayerOnRoll, White)
	}
	if m.Score[0] != 3 || m.Score[1] != 7 {
		t.Errorf("score should swap: got %v, want [3 7]", m.Score)
	}
	if m.Cube.Owner != White {
		t.Errorf("cube owner should flip: got %d, want %d", m.Cube.Owner, White)
	}
	if m.Cube.Value != 2 {
		t.Errorf("cube value should be untouched: got %d, want 2", m.Cube.Value)
	}
}

func TestMirrorCubeOwnerNone(t *testing.T) {
	p := posPts(nil)
	p.Cube = Cube{Owner: None, Value: 0}
	m := p.Mirror()
	if m.Cube.Owner != None {
		t.Errorf("cube owner None must stay None (centered cube has no owner to flip): got %d", m.Cube.Owner)
	}
}

func TestMirrorIsAnInvolution(t *testing.T) {
	// Mirroring twice must return to the original position.
	p := posPts(map[int]int{1: 2, 6: 5, 8: 3, 12: 5, 24: -2, 19: -5, 17: -3, 13: -5})
	p.Board.Bearoff = [2]int{0, 0}
	p.PlayerOnRoll = White
	p.Score = [2]int{7, 3}
	p.Cube = Cube{Owner: Black, Value: 1}

	once := p.Mirror()
	mm := once.Mirror()
	if mm.PlayerOnRoll != p.PlayerOnRoll || mm.Score != p.Score || mm.Cube != p.Cube || mm.Board.Bearoff != p.Board.Bearoff {
		t.Fatalf("Mirror(Mirror(p)) scalar fields != p: got %+v, want %+v", mm, p)
	}
	for i := range p.Board.Points {
		if mm.Board.Points[i] != p.Board.Points[i] {
			t.Errorf("point %d: Mirror(Mirror(p)) = %+v, want %+v", i, mm.Board.Points[i], p.Board.Points[i])
		}
	}
}

// --- NormalizeForStorage ----------------------------------------------------

func TestNormalizeForStorage(t *testing.T) {
	t.Run("player on roll 0 is unchanged", func(t *testing.T) {
		p := posPts(map[int]int{1: 2, 24: -2})
		p.PlayerOnRoll = Black
		got := p.NormalizeForStorage()
		if got.PlayerOnRoll != Black {
			t.Errorf("PlayerOnRoll = %d, want unchanged (%d)", got.PlayerOnRoll, Black)
		}
		if got.Board.Points[1] != p.Board.Points[1] {
			t.Errorf("board should be untouched: got %+v, want %+v", got.Board.Points[1], p.Board.Points[1])
		}
	})

	t.Run("player on roll 1 is mirrored to 0", func(t *testing.T) {
		p := posPts(map[int]int{1: 2, 24: -2})
		p.PlayerOnRoll = White
		got := p.NormalizeForStorage()
		if got.PlayerOnRoll != Black {
			t.Errorf("PlayerOnRoll = %d, want %d after normalization", got.PlayerOnRoll, Black)
		}
		want := p.Mirror()
		if got != want {
			t.Errorf("NormalizeForStorage(playerOnRoll=1) should equal Mirror(): got %+v, want %+v", got, want)
		}
	})

	t.Run("two mirror-equivalent positions from different matches normalize identically", func(t *testing.T) {
		// This is the mechanism that lets SavePosition dedup by Zobrist hash
		// across imports (ADR-0001): the same physical position seen from
		// each player's perspective must normalize to one canonical value.
		a := posPts(map[int]int{1: 2, 24: -5})
		a.PlayerOnRoll = Black
		a.Cube = Cube{Owner: None} // avoid the Owner==0 zero value, which Mirror flips like Black

		// b is exactly Mirror(a)'s board (point i's content moved to 25-i,
		// color flipped), stored as White's turn.
		b := posPts(map[int]int{1: 5, 24: -2})
		b.PlayerOnRoll = White
		b.Cube = Cube{Owner: None}

		normA := a.NormalizeForStorage()
		normB := b.NormalizeForStorage()
		if normA != normB {
			t.Errorf("mirror-equivalent positions should normalize identically:\n  a=%+v\n  b=%+v", normA, normB)
		}
	})
}

// --- MatchesMirrorPosition ---------------------------------------------------

func TestMatchesMirrorPosition(t *testing.T) {
	p := posPts(map[int]int{1: 2, 24: -5})

	t.Run("direct match", func(t *testing.T) {
		filter := posPts(map[int]int{1: 2})
		if !p.MatchesMirrorPosition(filter) {
			t.Error("expected direct checker-position match")
		}
	})

	t.Run("match only via mirror", func(t *testing.T) {
		// Mirror(p) has 5 Black checkers on point 1 (mirroring the 5 White
		// checkers on point 24 of p). p itself only has 2 Black on point 1,
		// so this filter can only be satisfied through the mirror.
		filter := posPts(map[int]int{1: 5})
		if p.MatchesCheckerPosition(filter) {
			t.Fatal("test setup error: filter should NOT match p directly")
		}
		if !p.MatchesMirrorPosition(filter) {
			t.Error("expected match via mirrored position")
		}
	})

	t.Run("no match either way", func(t *testing.T) {
		filter := posPts(map[int]int{12: 3})
		if p.MatchesMirrorPosition(filter) {
			t.Error("expected no match")
		}
	})
}

// --- ComputePipCounts / PipCountDifference ----------------------------------

func TestComputePipCounts(t *testing.T) {
	cases := []struct {
		name           string
		pts            map[int]int
		wantP1, wantP2 int
	}{
		{"empty board", nil, 0, 0},
		{"single black checker", map[int]int{6: 1}, 6, 0},
		{"single white checker", map[int]int{6: -1}, 0, 19}, // (25-6)*1
		{"standard opening position", map[int]int{
			24: 2, 13: 5, 8: 3, 6: 5, // Black (initializeBoard's Black layout)
			1: -2, 12: -5, 17: -3, 19: -5, // White (mirror layout)
		}, 167, 167}, // classic starting pip count for both sides
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := posPts(c.pts)
			p1, p2 := p.ComputePipCounts()
			if p1 != c.wantP1 || p2 != c.wantP2 {
				t.Errorf("ComputePipCounts() = (%d, %d), want (%d, %d)", p1, p2, c.wantP1, c.wantP2)
			}
			if diff := p.PipCountDifference(); diff != c.wantP1-c.wantP2 {
				t.Errorf("PipCountDifference() = %d, want %d", diff, c.wantP1-c.wantP2)
			}
		})
	}
}

// --- MatchesDiceRoll / MatchesDiceRollMode ----------------------------------

func TestMatchesDiceRollMode(t *testing.T) {
	base := Position{PlayerOnRoll: Black, DecisionType: CheckerAction}

	cases := []struct {
		name   string
		p      Position
		filter Position
		mode   string
		want   bool
	}{
		{"both: exact same order", withDice(base, 6, 5), withDice(base, 6, 5), "both", true},
		{"both: reversed order still matches", withDice(base, 6, 5), withDice(base, 5, 6), "both", true},
		{"both: different roll", withDice(base, 6, 5), withDice(base, 4, 3), "both", false},
		{"both: different player on roll", withDice(withRoll(base, White), 6, 5), withDice(base, 6, 5), "both", false},
		{"both: different decision type", withDice(withDecision(base, CubeAction), 0, 0), withDice(base, 0, 0), "both", false},
		{"first: filter die matches either die", withDice(base, 6, 5), withDice(base, 6, 0), "first", true},
		{"first: filter die matches the other die", withDice(base, 6, 5), withDice(base, 5, 0), "first", true},
		{"first: filter die matches neither", withDice(base, 6, 5), withDice(base, 4, 0), "first", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.p.MatchesDiceRollMode(c.filter, c.mode); got != c.want {
				t.Errorf("MatchesDiceRollMode(dice=%v, filterDice=%v, mode=%q) = %v, want %v",
					c.p.Dice, c.filter.Dice, c.mode, got, c.want)
			}
		})
	}
}

func TestMatchesDiceRoll(t *testing.T) {
	// MatchesDiceRoll is MatchesDiceRollMode(filter, "both").
	base := Position{PlayerOnRoll: Black, DecisionType: CheckerAction}
	p := withDice(base, 6, 5)
	filter := withDice(base, 5, 6)
	if !p.MatchesDiceRoll(filter) {
		t.Error("MatchesDiceRoll should accept reversed dice order, matching mode \"both\"")
	}
}

func withDice(p Position, d1, d2 int) Position {
	p.Dice = [2]int{d1, d2}
	return p
}

func withRoll(p Position, roll int) Position {
	p.PlayerOnRoll = roll
	return p
}

func withDecision(p Position, dt int) Position {
	p.DecisionType = dt
	return p
}

// --- MatchesNoContact --------------------------------------------------------

func TestMatchesNoContact(t *testing.T) {
	cases := []struct {
		name string
		pts  map[int]int
		want bool
	}{
		{"empty board: vacuously no contact", nil, true},
		{
			"classic no-contact race: Black's back checker (pt 6) already past White's back checker (pt 10)",
			map[int]int{6: 15, 10: -15},
			true,
		},
		{
			"contact: Black's back checker (pt 20) is behind White's back checker (pt 10)",
			map[int]int{20: 15, 10: -15},
			false,
		},
		{"only Black on the board: no opponent, so no contact", map[int]int{6: 15}, true},
		{"only White on the board: no opponent, so no contact", map[int]int{10: -15}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := posPts(c.pts)
			if got := p.MatchesNoContact(); got != c.want {
				t.Errorf("MatchesNoContact() = %v, want %v", got, c.want)
			}
		})
	}
}
