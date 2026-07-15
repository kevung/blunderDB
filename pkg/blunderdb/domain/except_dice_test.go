package domain

import (
	"reflect"
	"testing"
)

func TestParseExceptDice(t *testing.T) {
	cases := []struct {
		in   string
		want [][2]int
	}{
		{"", nil},
		{"65", [][2]int{{6, 5}}},
		{"65;54", [][2]int{{6, 5}, {5, 4}}},
		{"11", [][2]int{{1, 1}}},
		{" 65 ; 54 ", [][2]int{{6, 5}, {5, 4}}},
		{"70", nil},      // 7 and 0 out of range
		{"6", nil},       // single digit
		{"655", nil},     // too long
		{"6a", nil},      // non-digit
		{"65;99;54", [][2]int{{6, 5}, {5, 4}}}, // 99 skipped
	}
	for _, c := range cases {
		got := ParseExceptDice(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("ParseExceptDice(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestMatchesExceptDice(t *testing.T) {
	excl := ParseExceptDice("65;11")
	cases := []struct {
		dice [2]int
		keep bool
	}{
		{[2]int{6, 5}, false}, // excluded, as entered
		{[2]int{5, 6}, false}, // excluded, reversed order
		{[2]int{1, 1}, false}, // excluded double
		{[2]int{4, 3}, true},  // not excluded
		{[2]int{0, 0}, true},  // cube decision (no roll) survives
		{[2]int{6, 4}, true},  // shares a die but not the roll
	}
	for _, c := range cases {
		p := &Position{Dice: c.dice}
		if got := p.MatchesExceptDice(excl); got != c.keep {
			t.Errorf("dice %v: MatchesExceptDice = %v, want %v", c.dice, got, c.keep)
		}
	}
	// Empty exclusion keeps everything.
	p := &Position{Dice: [2]int{6, 5}}
	if !p.MatchesExceptDice(ParseExceptDice("")) {
		t.Errorf("empty exclusion should keep all positions")
	}
}
