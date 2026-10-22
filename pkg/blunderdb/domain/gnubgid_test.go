package domain

import (
	"encoding/json"
	"os"
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
			// A match whose leader is at match point minus one, with no Crawford
			// flag, is a length the position cannot state (see pointsScored).
			if c.MatchTo > 0 && (c.Score[0] != 0 && c.Score[1] != 0) {
				return
			}
			matchChecked++
			if got := EncodeGNUbgMatchID(&pos); got != c.MatchID {
				t.Errorf("match id = %s, gnubg says %s", got, c.MatchID)
			}
		})
	}
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
