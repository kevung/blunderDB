package domain

import (
	"encoding/json"
	"errors"
	"os"
	"testing"
)

func TestDecodeGNUbgIDAgainstGnubg(t *testing.T) {
	raw, err := os.ReadFile("testdata/gnubgid_reference.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []gnubgCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	exact := 0
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			id := c.PositionID + ":" + c.MatchID
			got, err := DecodeGNUbgID(id)
			if err != nil {
				t.Fatal(err)
			}
			want := gnubgPosition(c)
			want.Board = emptyPointsUncoloured(want.Board)
			if got.Board != want.Board {
				t.Errorf("board = %+v, want %+v", got.Board, want.Board)
			}
			if got.PlayerOnRoll != want.PlayerOnRoll || got.Cube != want.Cube || got.DecisionType != want.DecisionType {
				t.Errorf("turn/cube/decision = %d %+v %d, want %d %+v %d",
					got.PlayerOnRoll, got.Cube, got.DecisionType, want.PlayerOnRoll, want.Cube, want.DecisionType)
			}
			if got.Score != want.Score || got.HasJacoby != want.HasJacoby {
				t.Errorf("score/jacoby = %v %d, want %v %d", got.Score, got.HasJacoby, want.Score, want.HasJacoby)
			}
			if want.DecisionType == CheckerAction &&
				got.Dice != [2]int{max(c.Dice[0], c.Dice[1]), min(c.Dice[0], c.Dice[1])} {
				t.Errorf("dice = %v, gnubg says %v", got.Dice, c.Dice)
			}
			back, err := EncodeGNUbgID(&got)
			if err != nil {
				t.Fatal(err)
			}
			// Points scored by both sides come back as distances (see DecodeGNUbgID).
			if c.MatchTo == 0 || c.Score[0] == 0 || c.Score[1] == 0 {
				exact++
				if back != id {
					t.Errorf("Encode(Decode(%s)) = %s", id, back)
				}
			}
		})
	}
	if exact < 20 {
		t.Errorf("only %d identifiers round-tripped exactly", exact)
	}
}

func TestDecodeGNUbgPositionIDAlone(t *testing.T) {
	pos, err := DecodeGNUbgID("4HPwATDgc/ABMA")
	if err != nil {
		t.Fatal(err)
	}
	start := InitializePosition()
	start.Board = emptyPointsUncoloured(start.Board)
	if pos.Board != start.Board {
		t.Errorf("board = %+v, want the starting position", pos.Board)
	}
	if pos.PlayerOnRoll != Black || pos.DecisionType != CubeAction || pos.Score != [2]int{Unlimited, Unlimited} ||
		pos.Cube != (Cube{Owner: None, Value: 0}) {
		t.Errorf("defaults = %d %d %v %+v", pos.PlayerOnRoll, pos.DecisionType, pos.Score, pos.Cube)
	}
}

func TestDecodeGNUbgIDRefuses(t *testing.T) {
	for _, id := range []string{
		"",
		"4HPwATDgc/AB",                 // short
		"4HPwATDgc/AB!A",               // not base64
		"////////////////",             // 16 characters
		"//////////////",               // 15+ checkers for one side
		"4HPwATDgc/ABMA:cIkFAAAA",      // short match id
		"4HPwATDgc/ABMA:cIkFAAAAAAAEA", // 13-character match id
		"4HPwATDgc/ABMA:" + matchIDWith(map[[2]int]int{{21, 15}: 3, {36, 15}: 5}), // score past the length
		"4HPwATDgc/ABMA:" + matchIDWith(map[[2]int]int{{15, 3}: 7, {18, 3}: 1}),   // die of 7
		"4HPwATDgc/ABMA:" + matchIDWith(map[[2]int]int{{4, 2}: 2}),                // cube owner 2
	} {
		if _, err := DecodeGNUbgID(id); !errors.Is(err, ErrInvalidGNUbgID) {
			t.Errorf("DecodeGNUbgID(%q) = %v, want ErrInvalidGNUbgID", id, err)
		}
	}
}

func TestFindGNUbgID(t *testing.T) {
	for text, want := range map[string]string{
		"4HPwATDgc/ABMA:cIkFAAAAAAAE":                                                             "4HPwATDgc/ABMA:cIkFAAAAAAAE",
		"voici : 4HPwATDgc/ABMA:cIkFAAAAAAAE, à vous":                                             "4HPwATDgc/ABMA:cIkFAAAAAAAE",
		"GNU Backgammon  Position ID: 4HPwATDgc/ABMA\n                 Match ID   : cIkFAAAAAAAE": "4HPwATDgc/ABMA:cIkFAAAAAAAE",
		"Position ID: 4HPwATDgc/ABMA":                                                             "4HPwATDgc/ABMA",
		"  4HPwATDgc/ABMA \n":                                                                     "4HPwATDgc/ABMA",
	} {
		if got, ok := FindGNUbgID(text); !ok || got != want {
			t.Errorf("FindGNUbgID(%q) = %q %v, want %q", text, got, ok, want)
		}
	}
	for _, text := range []string{"Bonjour, quelle est la bonne case", "abcdefghijklmn est un mot", "XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:0:10"} {
		if got, ok := FindGNUbgID(text); ok {
			t.Errorf("FindGNUbgID(%q) = %q", text, got)
		}
	}
}

// emptyPointsUncoloured gives every empty point the colour None, as decoders write it.
func emptyPointsUncoloured(b Board) Board {
	for i, p := range b.Points {
		if p.Checkers == 0 {
			b.Points[i].Color = None
		}
	}
	return b
}

// matchIDWith writes a Match ID whose fields {at, width} hold the given values.
func matchIDWith(fields map[[2]int]int) string {
	var key [9]byte
	for f, v := range fields {
		for i := 0; i < f[1]; i++ {
			if v&(1<<i) != 0 {
				key[(f[0]+i)/8] |= 1 << ((f[0] + i) % 8)
			}
		}
	}
	return gnubgBase64Encode(key[:])
}
