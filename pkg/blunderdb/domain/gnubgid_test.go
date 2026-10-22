package domain

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type gnubgCase struct {
	Name         string   `json:"name"`
	Cube         int      `json:"cube"`
	CubeOwner    int      `json:"cube_owner"`
	PlayerOnRoll int      `json:"player_on_roll"`
	MatchTo      int      `json:"match_to"`
	Score        [2]int   `json:"score"`
	Crawford     int      `json:"crawford"`
	Jacoby       int      `json:"jacoby"`
	Dice         [2]int   `json:"dice"`
	PositionID   string   `json:"position_id"`
	MatchID      string   `json:"match_id"`
	Opp          [25]int  `json:"board_opponent"`
	Roll         [25]int  `json:"board_on_roll"`
	_            struct{} `json:"-"`
}

// gnubgPosition builds the position a gnubg case describes. gnubg numbers its
// players 0 and 1; White is 0 and Black is 1 (see gnubgid.go).
func gnubgPosition(c gnubgCase) Position {
	colorOf := func(player int) int {
		if player == 1 {
			return Black
		}
		return White
	}
	var pos Position
	onRoll := colorOf(c.PlayerOnRoll)
	opp := Black
	if onRoll == Black {
		opp = White
	}
	pos.PlayerOnRoll = onRoll
	fill := func(board [25]int, color int) {
		total := 0
		for slot, n := range board {
			if n == 0 {
				continue
			}
			total += n
			var i int
			switch {
			case slot == 24 && color == Black:
				i = 25
			case slot == 24:
				i = 0
			case color == Black:
				i = slot + 1
			default:
				i = 24 - slot
			}
			pos.Board.Points[i] = Point{Checkers: n, Color: color}
		}
		pos.Board.Bearoff[color] = 15 - total
	}
	fill(c.Roll, onRoll)
	fill(c.Opp, opp)

	pos.Cube.Value = c.Cube
	pos.Cube.Owner = None
	if c.CubeOwner == 0 || c.CubeOwner == 1 {
		pos.Cube.Owner = colorOf(c.CubeOwner)
	}
	if c.Dice[0] > 0 {
		pos.Dice = c.Dice
		pos.DecisionType = CheckerAction
	} else {
		pos.DecisionType = CubeAction
	}
	if c.MatchTo == 0 {
		pos.Score = [2]int{Unlimited, Unlimited}
		pos.HasJacoby = c.Jacoby
	} else {
		// Score is [Black, White], away; Crawford is the sentinel.
		pos.Score = AwayScoresWithCrawford(c.MatchTo, c.Score[1], c.Score[0], c.Crawford == 1)
	}
	return pos
}

func TestGNUbgIDAgainstGnubg(t *testing.T) {
	raw, err := os.ReadFile("testdata/gnubgid_reference.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []gnubgCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) < 40 {
		t.Fatalf("reference holds %d cases", len(cases))
	}
	matchChecked := 0
	defer func() {
		if matchChecked < 20 {
			t.Errorf("only %d match ids compared", matchChecked)
		}
	}()
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			pos := gnubgPosition(c)
			got, err := EncodeGNUbgPositionID(&pos)
			if err != nil {
				t.Fatal(err)
			}
			if got != c.PositionID {
				t.Errorf("position id = %s, gnubg says %s", got, c.PositionID)
			}
			got = EncodeGNUbgMatchID(&pos)
			// A position states only the distances to the match, so the length is
			// derived as the larger one (see pointsScored). When both players have
			// points, the length and scores differ from gnubg's, though not the
			// distances: everything else must still be gnubg's.
			if c.MatchTo > 0 && c.Score[0] != 0 && c.Score[1] != 0 {
				want, have := matchIDBits(t, c.MatchID), matchIDBits(t, got)
				for _, r := range [][2]int{{0, 21}, {66, 67}} {
					if g, w := bitsOf(have, r[0], r[1]-r[0]), bitsOf(want, r[0], r[1]-r[0]); g != w {
						t.Errorf("match id bits %d-%d = %d, gnubg says %d", r[0], r[1]-1, g, w)
					}
				}
				for i := range 2 {
					at := 36 + 15*i
					away := bitsOf(want, 21, 15) - bitsOf(want, at, 15)
					if g := bitsOf(have, 21, 15) - bitsOf(have, at, 15); g != away {
						t.Errorf("player %d is %d away, gnubg says %d", i, g, away)
					}
				}
				return
			}
			matchChecked++
			if got != c.MatchID {
				t.Errorf("match id = %s, gnubg says %s", got, c.MatchID)
			}
		})
	}
}

// matchIDBits unpacks a Match ID into its 67 bits, least significant first.
func matchIDBits(t *testing.T, id string) []int {
	t.Helper()
	var bits []int
	for _, r := range id {
		i := strings.IndexRune(gnubgBase64, r)
		if i < 0 {
			t.Fatalf("bad match id %q", id)
		}
		bits = append(bits, i>>5&1, i>>4&1, i>>3&1, i>>2&1, i>>1&1, i&1)
	}
	// base64 fills the high bits of each byte first; regroup into bytes, then
	// read each byte from its least significant end as gnubg writes it.
	var out []int
	for b := 0; b+8 <= len(bits); b += 8 {
		for k := 7; k >= 0; k-- {
			out = append(out, bits[b+k])
		}
	}
	return out
}

func bitsOf(bits []int, at, n int) int {
	v := 0
	for i := 0; i < n; i++ {
		v |= bits[at+i] << i
	}
	return v
}

func TestGNUbgPositionIDRejectsTooManyCheckers(t *testing.T) {
	var pos Position
	pos.PlayerOnRoll = Black
	pos.Board.Points[6] = Point{Checkers: 10, Color: Black}
	pos.Board.Points[8] = Point{Checkers: 6, Color: Black}
	if _, err := EncodeGNUbgPositionID(&pos); err == nil {
		t.Fatal("16 checkers accepted")
	}
}
