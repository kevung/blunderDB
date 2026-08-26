package storage

import "testing"

// TestMatchOutcome pins the rule that turns per-game points into a win, since
// blunderDB stores no winner on a match. The case that matters is the
// unfinished match: counting it as a loss for whoever was behind would invent
// results out of truncated logs.
func TestMatchOutcome(t *testing.T) {
	cases := []struct {
		name        string
		matchLength int32
		pts1, pts2  int32
		want        int
	}{
		{"player 1 reaches the target", 7, 7, 3, 1},
		{"player 2 reaches the target", 7, 2, 7, -1},
		{"overshooting the target still wins", 7, 9, 4, 1},
		{"neither reaches it: unfinished, nobody wins", 7, 5, 4, 0},
		{"no game played at all", 7, 0, 0, 0},
		{"both at the target is corrupt, not a result", 7, 7, 7, 0},
		{"money session: more points wins", 0, 5, 2, 1},
		{"money session: the other side", 0, 1, 6, -1},
		{"money session: level is a draw", 0, 3, 3, 0},
		{"money session: nothing played", 0, 0, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := MatchOutcome(c.matchLength, c.pts1, c.pts2); got != c.want {
				t.Errorf("MatchOutcome(%d, %d, %d) = %d, want %d",
					c.matchLength, c.pts1, c.pts2, got, c.want)
			}
		})
	}
}

// TestBuildPlayerRowsWinsAndLosses checks the two halves that a per-match view
// cannot show: that a player's record adds up across matches, and that an
// unfinished match leaves wins+losses short of the match count rather than
// being quietly attributed to someone.
func TestBuildPlayerRowsWinsAndLosses(t *testing.T) {
	matches := []MatchOutcomeRow{
		{Player1: "Alice", Player2: "Bob", MatchLength: 7, Points1: 7, Points2: 3},
		{Player1: "Bob", Player2: "Alice", MatchLength: 7, Points1: 7, Points2: 5},
		{Player1: "Alice", Player2: "Carol", MatchLength: 7, Points1: 4, Points2: 5}, // unfinished
	}
	rows := BuildPlayerRows(nil, matches, nil, nil)

	by := map[string]PlayerRow{}
	for _, r := range rows {
		by[r.Name] = r
	}
	alice := by["Alice"]
	if alice.Matches != 3 || alice.Wins != 1 || alice.Losses != 1 {
		t.Errorf("Alice: got %d matches, %d wins, %d losses; want 3 / 1 / 1",
			alice.Matches, alice.Wins, alice.Losses)
	}
	if alice.Wins+alice.Losses >= alice.Matches {
		t.Error("the unfinished match must count for neither side")
	}
	if carol := by["Carol"]; carol.Wins != 0 || carol.Losses != 0 {
		t.Errorf("Carol only played the unfinished match: got %d wins, %d losses",
			carol.Wins, carol.Losses)
	}
}

// TestBuildPlayerRowsLuckIgnoresUnmeasuredRolls is the arithmetic behind
// ADR-0010's NULL: a luck average divides by the rolls actually measured. If it
// divided by rolls played, importing one analysed match into a database of
// unanalysed ones would drag every average towards zero.
func TestBuildPlayerRowsLuckIgnoresUnmeasuredRolls(t *testing.T) {
	rows := BuildPlayerRows(nil,
		[]MatchOutcomeRow{{Player1: "Alice", Player2: "Bob", MatchLength: 7}},
		nil,
		map[string]PlayerLuckAcc{"Alice": {SumMP: 300, Rolls: 10}},
	)
	by := map[string]PlayerRow{}
	for _, r := range rows {
		by[r.Name] = r
	}

	rate, ok := by["Alice"].LuckRateMP()
	if !ok {
		t.Fatal("Alice has measured rolls, want a luck rate")
	}
	if rate != 30 {
		t.Errorf("Alice's luck rate: got %v, want 30 millipoints per measured roll", rate)
	}
	if _, ok := by["Bob"].LuckRateMP(); ok {
		t.Error("Bob has no measured roll: want no rate at all, not an average of zero")
	}
}
