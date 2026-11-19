package domain

import "testing"

func TestMatchResultOf(t *testing.T) {
	cases := []struct {
		name   string
		length int32
		games  []Game
		final  *[2]int32
		winner int32
	}{
		{"no games", 5, nil, nil, WinnerUnfinished},
		{"player 2 wins the last game", 5, []Game{
			{InitialScore: [2]int32{0, 0}, Winner: WinnerPlayer1, PointsWon: 2},
			{InitialScore: [2]int32{2, 3}, Winner: WinnerPlayer2, PointsWon: 4},
		}, &[2]int32{2, 7}, WinnerPlayer2},
		{"player 1 wins", 3, []Game{{InitialScore: [2]int32{2, 1}, Winner: WinnerPlayer1, PointsWon: 1}}, &[2]int32{3, 1}, WinnerPlayer1},
		{"abandoned", 7, []Game{{InitialScore: [2]int32{4, 1}, Winner: WinnerUnfinished, PointsWon: 0}}, &[2]int32{4, 1}, WinnerUnfinished},
		{"money", 0, []Game{{InitialScore: [2]int32{0, 0}, Winner: WinnerPlayer1, PointsWon: 4}}, &[2]int32{4, 0}, WinnerUnfinished},
	}
	for _, c := range cases {
		final, winner := MatchResultOf(c.length, c.games)
		if (final == nil) != (c.final == nil) || (final != nil && *final != *c.final) || winner != c.winner {
			t.Errorf("%s: got %v %d, want %v %d", c.name, final, winner, c.final, c.winner)
		}
	}
}
