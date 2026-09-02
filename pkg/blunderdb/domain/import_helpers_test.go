package domain

import (
	"errors"
	"testing"
)

func TestAwayScores(t *testing.T) {
	cases := []struct {
		matchLength, s0, s1 int
		want                [2]int
	}{
		{7, 0, 0, [2]int{7, 7}},
		{7, 6, 2, [2]int{1, 5}}, // the Crawford game of a 7-pointer
		{0, 0, 0, [2]int{Unlimited, Unlimited}},
		{0, 3, 1, [2]int{Unlimited, Unlimited}}, // money: the score is not an away score
		{-1, 0, 0, [2]int{Unlimited, Unlimited}},
	}
	for _, c := range cases {
		if got := AwayScores(c.matchLength, c.s0, c.s1); got != c.want {
			t.Errorf("AwayScores(%d, %d, %d) = %v, want %v", c.matchLength, c.s0, c.s1, got, c.want)
		}
	}
}

func TestCubeExponent(t *testing.T) {
	for value, want := range map[int]int{0: 0, 1: 0, 2: 1, 4: 2, 8: 3, 16: 4, 64: 6, 3: 1} {
		if got := CubeExponent(value); got != want {
			t.Errorf("CubeExponent(%d) = %d, want %d", value, got, want)
		}
	}
}

func TestRecomputeBearoff(t *testing.T) {
	var b Board
	for i := range b.Points {
		b.Points[i] = Point{Color: None}
	}
	b.Points[6] = Point{Checkers: 5, Color: Black}
	b.Points[BlackBar] = Point{Checkers: 2, Color: Black}
	b.Points[19] = Point{Checkers: 15, Color: White}
	b.Points[3] = Point{Checkers: 1, Color: ExcludeEmpty} // a search marker, not a checker
	b.Bearoff = [2]int{99, 99}

	if err := b.RecomputeBearoff(); err != nil {
		t.Fatalf("RecomputeBearoff: %v", err)
	}
	if b.Bearoff != [2]int{8, 0} {
		t.Errorf("Bearoff = %v, want [8 0] (bar counted, marker ignored)", b.Bearoff)
	}

	// One checker too many for White: the error names the player, the
	// bearoff is not overwritten with a negative count.
	b.Points[20] = Point{Checkers: 1, Color: White}
	err := b.RecomputeBearoff()
	var tooMany *TooManyCheckersError
	if !errors.As(err, &tooMany) {
		t.Fatalf("RecomputeBearoff with 16 white checkers: got %v, want *TooManyCheckersError", err)
	}
	if tooMany.Color != White || tooMany.OnBoard != 16 {
		t.Errorf("error = %+v, want {Color: White, OnBoard: 16}", *tooMany)
	}
	if b.Bearoff != [2]int{8, 0} {
		t.Errorf("Bearoff overwritten on error: %v", b.Bearoff)
	}
	if got, want := err.Error(), "player 2 has 16 checkers on the board (15 expected)"; got != want {
		t.Errorf("message = %q, want %q", got, want)
	}
}
