package parser

import "testing"

// The clipboard text blunderDB writes for a stored checker analysis omits the
// "Equity Error:" line of the best move: its EquityError is nil by design (the
// best move has no error — see ingest.sortCheckerMovesByEquity) and the field is
// `omitempty`. Pasting such a block into another blunderDB must still yield the
// best move; the round-trip used to drop it, so the analysis panel of the pasted
// position started at the second move.
func TestParseInternalCheckerKeepsBestMoveWithoutEquityError(t *testing.T) {
	input := `XGID=-b----E-C---eE---c-e----B-:0:0:1:31:0:0:0:7:10

Position:
Board: {"points":[],"bearoff":[0,0]}
Cube: {"owner":-1,"value":0}
Dice: 3, 1
Score: -1, -1
Player on roll: 0
Decision type: 0

Analysis:
Checker Move Analysis:
Move 0: 24/23 13/10
Analysis Depth: "4-ply"
Equity: 0.123
Player Win Chance: 54.32%
Player Gammon Chance: 12.34%
Player Backgammon Chance: 0.56%
Opponent Win Chance: 45.68%
Opponent Gammon Chance: 10.12%
Opponent Backgammon Chance: 0.34%

Move 1: 24/21 13/12
Analysis Depth: "4-ply"
Equity: 0.1
Equity Error: 0.023
Player Win Chance: 53.1%
Player Gammon Chance: 11.2%
Player Backgammon Chance: 0.4%
Opponent Win Chance: 46.9%
Opponent Gammon Chance: 9.8%
Opponent Backgammon Chance: 0.3%

eXtreme Gammon Version: 2.19
`

	res, err := ParsePosition(input)
	if err != nil {
		t.Fatalf("ParsePosition: %v", err)
	}
	if res.Analysis.CheckerAnalysis == nil {
		t.Fatal("no checker analysis parsed")
	}
	moves := res.Analysis.CheckerAnalysis.Moves
	if len(moves) != 2 {
		t.Fatalf("got %d moves, want 2 (the best move must survive the round-trip)", len(moves))
	}
	if moves[0].Move != "24/23 13/10" {
		t.Errorf("first move = %q, want %q", moves[0].Move, "24/23 13/10")
	}
	if moves[0].EquityError != nil {
		t.Errorf("first move EquityError = %v, want nil (no line ⇒ no error)", *moves[0].EquityError)
	}
	if moves[1].EquityError == nil || *moves[1].EquityError != 0.023 {
		t.Errorf("second move EquityError = %v, want 0.023", moves[1].EquityError)
	}
}

// A stored analysis can carry an empty depth (some import routes leave it
// blank), and float formatting can hand us exponent notation. Neither may cost
// us the move.
func TestParseInternalCheckerToleratesEmptyDepthAndExponents(t *testing.T) {
	input := `XGID=-b----E-C---eE---c-e----B-:0:0:1:31:0:0:0:7:10

Analysis:
Checker Move Analysis:
Move 1: 24/21 13/12
Analysis Depth: ""
Equity: 0.1
Equity Error: 5.551115123125783e-17
Player Win Chance: 53.1%
Player Gammon Chance: 11.2%
Player Backgammon Chance: 0.4%
Opponent Win Chance: 46.9%
Opponent Gammon Chance: 9.8%
Opponent Backgammon Chance: 0.3%
`

	res, err := ParsePosition(input)
	if err != nil {
		t.Fatalf("ParsePosition: %v", err)
	}
	if res.Analysis.CheckerAnalysis == nil || len(res.Analysis.CheckerAnalysis.Moves) != 1 {
		t.Fatalf("got %+v, want exactly 1 move", res.Analysis.CheckerAnalysis)
	}
	m := res.Analysis.CheckerAnalysis.Moves[0]
	if m.AnalysisDepth != "" {
		t.Errorf("depth = %q, want empty", m.AnalysisDepth)
	}
	if m.EquityError == nil || *m.EquityError != 5.551115123125783e-17 {
		t.Errorf("EquityError = %v, want 5.551115123125783e-17", m.EquityError)
	}
}
