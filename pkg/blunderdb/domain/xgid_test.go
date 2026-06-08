package domain

import "testing"

// realXGIDBoards are board portions taken from real XG analysis exports. The
// codec must round-trip every one of them losslessly.
var realXGIDBoards = []string{
	"a--aB-BBA--acDa-Ab-db---BA",
	"-aa-B-D-B---dE---c-d---bB-",
	"-aa-B-D-C---bD---c-d--AbbA",
	"aBBABBB--A--A----b-cbBbcb-",
	"-aBBBBB-AA---Ba--bbg---bA-",
	"--a-BBB-B-B--BA---B----dj-",
	"-a--B-DbB---dE-A-c-e----A-",
	"-A--B-DCC----A------bbbcdA",
	"----BaC-B---aD--aa-bcbbBbB",
	"-bB-a-E-Ba-A-C---c-cbbaA-A",
	"----BBB-B---aC----B-B-gdc-",
	"--B-BcBBBB--bA-bBa-c-bb---",
	"--BCEb---BA-----b-bcbBbb--",
	"-b----E-C---eE---c-e----B-",
	"----C-E-A----C-AB------cb-",
}

func TestDecodeEncodeRoundTripBoard(t *testing.T) {
	for _, board := range realXGIDBoards {
		pos, err := DecodeXGID(board + ":0:0:1:00:0:0:0:0:10")
		if err != nil {
			t.Fatalf("DecodeXGID(%q): %v", board, err)
		}
		if got := EncodeXGIDBoard(&pos); got != board {
			t.Errorf("round-trip mismatch:\n  in:  %s\n  out: %s", board, got)
		}
		// Sanity: each player has at most 15 checkers on the board.
		var on [2]int
		for _, p := range pos.Board.Points {
			if p.Color == Black || p.Color == White {
				on[p.Color] += p.Checkers
			}
		}
		if on[Black] > 15 || on[White] > 15 {
			t.Errorf("%q: checker overflow black=%d white=%d", board, on[Black], on[White])
		}
		if pos.Board.Bearoff[Black] != 15-on[Black] || pos.Board.Bearoff[White] != 15-on[White] {
			t.Errorf("%q: wrong bearoff %v (on=%v)", board, pos.Board.Bearoff, on)
		}
	}
}

// TestDecodeXGIDFull checks the full metadata of one hand-verified vector
// (cross-checked against its XG board diagram):
//
//	XGID=a--aB-BBA--acDa-Ab-db---BA:0:0:1:64:2:0:0:13:10  (X to play 64, 13pt match, X:2 O:0)
func TestDecodeXGIDFull(t *testing.T) {
	pos, err := DecodeXGID("XGID=a--aB-BBA--acDa-Ab-db---BA:0:0:1:64:2:0:0:13:10")
	if err != nil {
		t.Fatalf("DecodeXGID: %v", err)
	}

	// X = Black: pts 13(4),16(1),24(2),8(1),7(2),6(2),4(2) + bar(25)=1 → 15.
	wantBlack := map[int]int{13: 4, 16: 1, 24: 2, 8: 1, 7: 2, 6: 2, 4: 2, BlackBar: 1}
	// O = White: pts 14(1),17(2),19(4),20(2),12(3),11(1),3(1) + bar(0)=1 → 15.
	wantWhite := map[int]int{14: 1, 17: 2, 19: 4, 20: 2, 12: 3, 11: 1, 3: 1, WhiteBar: 1}

	for i, p := range pos.Board.Points {
		if c, ok := wantBlack[i]; ok {
			if p.Color != Black || p.Checkers != c {
				t.Errorf("point %d: got {%d,%d}, want {%d,Black}", i, p.Checkers, p.Color, c)
			}
		} else if c, ok := wantWhite[i]; ok {
			if p.Color != White || p.Checkers != c {
				t.Errorf("point %d: got {%d,%d}, want {%d,White}", i, p.Checkers, p.Color, c)
			}
		} else if p.Checkers != 0 {
			t.Errorf("point %d: expected empty, got {%d,%d}", i, p.Checkers, p.Color)
		}
	}

	// Cube field 0 → exponent 0 (centred cube, displays as 2^0 = 1).
	if pos.Cube.Value != 0 || pos.Cube.Owner != None {
		t.Errorf("cube: got {value=%d owner=%d}, want centred exponent 0", pos.Cube.Value, pos.Cube.Owner)
	}
	if pos.PlayerOnRoll != Black {
		t.Errorf("PlayerOnRoll: got %d, want Black (X to play)", pos.PlayerOnRoll)
	}
	if pos.Dice != [2]int{6, 4} {
		t.Errorf("Dice: got %v, want [6 4]", pos.Dice)
	}
	if pos.DecisionType != CheckerAction {
		t.Errorf("DecisionType: got %d, want CheckerAction", pos.DecisionType)
	}
	// Away score: 13-2=11 (X/Black), 13-0=13 (O/White).
	if pos.Score != [2]int{11, 13} {
		t.Errorf("Score (away): got %v, want [11 13]", pos.Score)
	}
}

func TestDecodeXGIDErrors(t *testing.T) {
	cases := []string{
		"",                            // empty
		"XGID=",                       // empty after prefix
		"tooshort:0:0:1:00",           // board not 26 chars
		"a--aB-BBA--acDa-Ab-db---B1",  // bad board char '1'
	}
	for _, c := range cases {
		if _, err := DecodeXGID(c); err == nil {
			t.Errorf("DecodeXGID(%q): expected error, got nil", c)
		}
	}
}

func TestDecodeXGIDMoneyGame(t *testing.T) {
	// matchLen 0 → money game → away score sentinel [-1,-1].
	pos, err := DecodeXGID("-b----E-C---eE---c-e----B-:0:0:1:31:0:0:1:0:3")
	if err != nil {
		t.Fatalf("DecodeXGID: %v", err)
	}
	if pos.Score != [2]int{-1, -1} {
		t.Errorf("money game Score: got %v, want [-1 -1]", pos.Score)
	}
	// field 7 = 1 → Jacoby on, Beaver off.
	if pos.HasJacoby != 1 || pos.HasBeaver != 0 {
		t.Errorf("money game flags: got jacoby=%d beaver=%d, want jacoby=1 beaver=0", pos.HasJacoby, pos.HasBeaver)
	}
}

func TestDecodeXGIDJacobyBeaver(t *testing.T) {
	// Money game with field 7 = 3 (Jacoby + Beaver) — the "Pas de double /
	// Beaver" position from issue #13. Both flags must be set.
	pos, err := DecodeXGID("XGID=bA----D-C---dE---d-e----B-:0:0:1:00:0:0:3:0:10")
	if err != nil {
		t.Fatalf("DecodeXGID: %v", err)
	}
	if pos.HasJacoby != 1 || pos.HasBeaver != 1 {
		t.Errorf("flags: got jacoby=%d beaver=%d, want both 1", pos.HasJacoby, pos.HasBeaver)
	}

	// Match play: field 7 is the Crawford flag, not jacoby/beaver — both stay 0.
	pos, err = DecodeXGID("-b----E-C---eE---c-e----B-:0:0:1:31:0:0:1:7:3")
	if err != nil {
		t.Fatalf("DecodeXGID (match): %v", err)
	}
	if pos.HasJacoby != 0 || pos.HasBeaver != 0 {
		t.Errorf("match play flags: got jacoby=%d beaver=%d, want both 0", pos.HasJacoby, pos.HasBeaver)
	}
}
