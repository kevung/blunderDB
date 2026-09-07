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

// TestAwayScoresWithCrawford: the sentinel is the whole point. Away 1 and away
// 0 describe the same distance to victory and different rules (CONTEXT.md,
// « Away score »), so the same score maps to two away scores depending on
// which game it is — and only the ambiguous value is ever rewritten.
func TestAwayScoresWithCrawford(t *testing.T) {
	cases := []struct {
		name                string
		matchLength, s0, s1 int
		crawford            bool
		want                [2]int
	}{
		{"the Crawford game of a 7-pointer", 7, 6, 2, true, [2]int{Crawford, 5}},
		{"the game after it, same score", 7, 6, 2, false, [2]int{PostCrawford, 5}},
		{"double match point is post-Crawford on both sides", 7, 6, 6, false, [2]int{PostCrawford, PostCrawford}},
		{"nowhere near match point: untouched", 7, 2, 4, false, [2]int{5, 3}},
		{"a 1-point match is its own Crawford game", 1, 0, 0, true, [2]int{Crawford, Crawford}},
		{"money play knows no Crawford", 0, 3, 1, false, [2]int{Unlimited, Unlimited}},
		{"money play, flag set anyway", 0, 3, 1, true, [2]int{Unlimited, Unlimited}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := AwayScoresWithCrawford(c.matchLength, c.s0, c.s1, c.crawford)
			if got != c.want {
				t.Errorf("AwayScoresWithCrawford(%d, %d, %d, %v) = %v, want %v",
					c.matchLength, c.s0, c.s1, c.crawford, got, c.want)
			}
		})
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
