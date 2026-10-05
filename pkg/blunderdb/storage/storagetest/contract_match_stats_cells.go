package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testMatchStatsSharedPositions: a position reached by several matches is
// one position of the totals, whichever order the matches were computed in.
// The second match reaches the first's positions after the first was
// computed with them private; the totals must not count them twice.
func testMatchStatsSharedPositions(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	totals := func(label string, f storage.StatsFilter, positions, decisions, matches int) {
		t.Helper()
		res, err := s.Stats().Compute(ctx, "", f)
		if err != nil {
			t.Fatalf("%s: Compute: %v", label, err)
		}
		got := res.Totals
		if got.NumPositions != positions || got.NumDecisions != decisions || got.NumMatches != matches {
			t.Errorf("%s: positions %d, decisions %d, matches %d; want %d, %d, %d",
				label, got.NumPositions, got.NumDecisions, got.NumMatches, positions, decisions, matches)
		}
	}
	all := storage.StatsFilter{DecisionType: -1}

	statsFixtureMatch(t, s, 0, "Ann", "Abe")
	totals("one match", all, 2, 2, 1)

	matchB, _ := statsFixtureMatch(t, s, 0, "Bea", "Bob")
	totals("a second match on the same positions", all, 2, 4, 2)
	totals("one seat of each", storage.StatsFilter{DecisionType: -1, PlayerName: "Ann", PlayerAliases: []string{"Bea"}}, 1, 2, 2)

	statsFixtureMatch(t, s, 10, "Cid", "Cal")
	totals("a third match on other positions", all, 4, 6, 3)

	if err := s.Matches().DeleteCascade(ctx, "", matchB); err != nil {
		t.Fatal(err)
	}
	totals("the second match deleted", all, 4, 4, 2)
	totals("its players gone", storage.StatsFilter{DecisionType: -1, PlayerName: "Bea"}, 0, 0, 0)
}
