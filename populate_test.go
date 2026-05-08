package main

import (
	"testing"
)

func TestPopulatePositionColumns_Initial(t *testing.T) {
	pos := initialPosition()
	c := populatePositionColumns(&pos)

	// ZobristHash must be non-zero.
	if c.ZobristHash == 0 {
		t.Error("ZobristHash is zero")
	}

	// DecisionType should mirror the position.
	if c.DecisionType != pos.DecisionType {
		t.Errorf("DecisionType: got %d, want %d", c.DecisionType, pos.DecisionType)
	}

	// Pip counts for initial position: each side starts with 167 pips.
	if c.Pip1 != 167 {
		t.Errorf("Pip1: got %d, want 167", c.Pip1)
	}
	if c.Pip2 != 167 {
		t.Errorf("Pip2: got %d, want 167", c.Pip2)
	}
	if c.PipDiff != 0 {
		t.Errorf("PipDiff: got %d, want 0", c.PipDiff)
	}

	// No bearoff at start.
	if c.Off1 != 0 || c.Off2 != 0 {
		t.Errorf("Off1=%d Off2=%d, both should be 0", c.Off1, c.Off2)
	}

	// No contact in initial position? Black is at 24 (high), White is at 1 (low) — they interleave, so contact exists.
	if c.NoContact {
		t.Error("initial position should have contact")
	}

	// Occupancy masks non-zero.
	if c.Occupancy1 == 0 || c.Occupancy2 == 0 {
		t.Error("occupancy masks should be non-zero for initial position")
	}

	// Cube mirrors
	if c.CubeValue != 0 {
		t.Errorf("CubeValue: got %d, want 0", c.CubeValue)
	}
	if c.CubeOwner != None {
		t.Errorf("CubeOwner: got %d, want None (%d)", c.CubeOwner, None)
	}
}

func TestPopulatePositionColumns_BackCheckers(t *testing.T) {
	// A position where Black has checkers in opponent's home (points 19-24).
	var b Board
	b.Points[20] = Point{Checkers: 2, Color: Black} // back checker
	b.Points[6] = Point{Checkers: 13, Color: Black}
	b.Points[1] = Point{Checkers: 2, Color: White}
	b.Points[18] = Point{Checkers: 13, Color: White}
	pos := Position{Board: b, Cube: Cube{Owner: None, Value: 0}, PlayerOnRoll: 0}

	c := populatePositionColumns(&pos)
	if c.BackCheckers1 != 2 {
		t.Errorf("BackCheckers1: got %d, want 2", c.BackCheckers1)
	}
	if c.BackCheckers2 != 2 {
		t.Errorf("BackCheckers2: got %d, want 2", c.BackCheckers2)
	}
}

func TestPopulatePositionColumns_NoContact(t *testing.T) {
	// Pure race: Black at points 1-3, White at points 20-22 — no contact.
	var b Board
	b.Points[1] = Point{Checkers: 5, Color: Black}
	b.Points[2] = Point{Checkers: 5, Color: Black}
	b.Points[3] = Point{Checkers: 5, Color: Black}
	b.Points[20] = Point{Checkers: 5, Color: White}
	b.Points[21] = Point{Checkers: 5, Color: White}
	b.Points[22] = Point{Checkers: 5, Color: White}
	pos := Position{Board: b, Cube: Cube{Owner: None, Value: 0}, PlayerOnRoll: 0}

	c := populatePositionColumns(&pos)
	if !c.NoContact {
		t.Error("pure race position should report NoContact=true")
	}
}

func TestPopulatePositionColumns_Normalization(t *testing.T) {
	// PlayerOnRoll=0 and its mirror (PlayerOnRoll=1) should produce the same ZobristHash.
	pos0 := initialPosition()
	pos1 := pos0.Mirror()

	c0 := populatePositionColumns(&pos0)
	c1 := populatePositionColumns(&pos1)

	if c0.ZobristHash != c1.ZobristHash {
		t.Errorf("PlayerOnRoll=0 hash 0x%016x != PlayerOnRoll=1 hash 0x%016x", c0.ZobristHash, c1.ZobristHash)
	}
}

// ---------------------------------------------------------------------------
// populateAnalysisColumns tests
// ---------------------------------------------------------------------------

func TestPopulateAnalysisColumns_Nil(t *testing.T) {
	c := populateAnalysisColumns(nil, "", "")
	if c.BestCubeAction != "" || c.CubeError != 0 || c.BestMoveEquityError != 0 {
		t.Error("nil analysis should produce zero-value columns")
	}
}

func TestPopulateAnalysisColumns_WinRates(t *testing.T) {
	errVal := 0.12
	a := &PositionAnalysis{
		DoublingCubeAnalysis: &DoublingCubeAnalysis{
			BestCubeAction:            "NoDouble",
			PlayerWinChances:          0.62,
			PlayerGammonChances:       0.18,
			PlayerBackgammonChances:   0.01,
			OpponentWinChances:        0.38,
			OpponentGammonChances:     0.08,
			OpponentBackgammonChances: 0.00,
			CubefulNoDoubleError:      0.0,
			CubefulDoubleTakeError:    0.05,
			CubefulDoublePassError:    errVal,
		},
		CheckerAnalysis: &CheckerAnalysis{
			Moves: []CheckerMove{
				{Move: "13/7 8/7", Equity: 0.42, EquityError: func() *float64 { v := 0.0; return &v }()},
				{Move: "13/7 6/5", Equity: 0.30, EquityError: func() *float64 { v := 0.12; return &v }()},
			},
		},
	}

	c := populateAnalysisColumns(a, "13/7 6/5", "NoDouble")

	if c.BestCubeAction != "NoDouble" {
		t.Errorf("BestCubeAction: got %q, want %q", c.BestCubeAction, "NoDouble")
	}
	// Values are integer-scaled: rates ×100, equities ×1000
	if c.Player1WinRate != 62 {
		t.Errorf("Player1WinRate: got %d, want 62", c.Player1WinRate)
	}
	if c.Player2WinRate != 38 {
		t.Errorf("Player2WinRate: got %d, want 38", c.Player2WinRate)
	}
	// NoDouble is the best action → error should be 0
	if c.CubeError != 0 {
		t.Errorf("CubeError for correct NoDouble: got %d, want 0", c.CubeError)
	}
	// Played move is Moves[1] with EquityError=0.12 → 120 millipoints
	if c.BestMoveEquityError != 120 {
		t.Errorf("BestMoveEquityError: got %d, want 120", c.BestMoveEquityError)
	}
}

func TestPopulateAnalysisColumns_CubeError(t *testing.T) {
	a := &PositionAnalysis{
		DoublingCubeAnalysis: &DoublingCubeAnalysis{
			BestCubeAction:         "Double/Take",
			CubefulNoDoubleError:   0.25,
			CubefulDoubleTakeError: 0.0,
			CubefulDoublePassError: 0.08,
		},
	}

	// Played "NoDouble" but best is "Double/Take" → error = |0.25| = 250 millipoints
	c := populateAnalysisColumns(a, "", "NoDouble")
	if c.CubeError != 250 {
		t.Errorf("CubeError for wrong NoDouble: got %d, want 250", c.CubeError)
	}

	// Same with space-separated form used by XG import
	c2 := populateAnalysisColumns(a, "", "No Double")
	if c2.CubeError != 250 {
		t.Errorf("CubeError for wrong 'No Double' (with space): got %d, want 250", c2.CubeError)
	}

	// Real XG data stores cube errors as negative (thisAction - bestEquity).
	// populateAnalysisColumns must take the absolute value.
	a2 := &PositionAnalysis{
		DoublingCubeAnalysis: &DoublingCubeAnalysis{
			BestCubeAction:         "Double, Take",
			CubefulNoDoubleError:   -0.120, // negative in raw XG convention
			CubefulDoubleTakeError: 0.0,
			CubefulDoublePassError: -0.05,
		},
	}
	c3 := populateAnalysisColumns(a2, "", "No Double")
	if c3.CubeError != 120 {
		t.Errorf("CubeError for negative raw error: got %d, want 120", c3.CubeError)
	}
}

func TestPopulateAnalysisColumns_NoPlayedMove(t *testing.T) {
	a := &PositionAnalysis{
		CheckerAnalysis: &CheckerAnalysis{
			Moves: []CheckerMove{
				{Move: "8/2", Equity: 0.3, EquityError: func() *float64 { v := 0.0; return &v }()},
			},
		},
	}
	c := populateAnalysisColumns(a, "", "")
	if c.BestMoveEquityError != 0 {
		t.Errorf("BestMoveEquityError should be 0 when no played move, got %d", c.BestMoveEquityError)
	}
}
