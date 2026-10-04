package storagetest

import (
	"context"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testMETDifferentLeftOut: an analysis at a match score valued with another
// table than the current one is left out of the comparisons, decision by
// decision and match by match, and changing the current table changes what
// is retained without rewriting a row (ADR-0068).
func testMETDifferentLeftOut(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	_, posA := statsFixtureMatch(t, s, 0, "Ann", "Abe")
	_, posB := statsFixtureMatch(t, s, 10, "Bea", "Bob")
	mt := s.MatchEquityTables()

	retained := func(label string, wantMatches int, wantPlayers ...string) {
		t.Helper()
		res, err := s.Stats().Compute(ctx, "", storage.StatsFilter{DecisionType: -1})
		if err != nil {
			t.Fatalf("%s: Compute: %v", label, err)
		}
		if res.Totals.NumMatches != wantMatches {
			t.Errorf("%s: %d matches retained, want %d", label, res.Totals.NumMatches, wantMatches)
		}
		rows, err := s.Stats().PlayerTable(ctx, "", storage.StatsFilter{DecisionType: -1})
		if err != nil {
			t.Fatalf("%s: PlayerTable: %v", label, err)
		}
		var names []string
		for _, r := range rows {
			if r.Matches > 0 {
				names = append(names, r.Name)
			}
		}
		slices.Sort(names)
		slices.Sort(wantPlayers)
		if !slices.Equal(names, wantPlayers) {
			t.Errorf("%s: players %v, want %v", label, names, wantPlayers)
		}
	}

	retained("built-in current, nothing tagged", 2, "Abe", "Ann", "Bea", "Bob")

	id, err := mt.Save(ctx, "", domain.MatchEquityTable{Name: "Rockwell-Kazaross", Digest: "rk", Source: "<met/>"})
	if err != nil {
		t.Fatal(err)
	}
	if err := mt.SetCurrent(ctx, "", id); err != nil {
		t.Fatal(err)
	}
	retained("imported current, nothing tagged", 0)

	if err := mt.TagAnalyses(ctx, "", id, posA[:]); err != nil {
		t.Fatal(err)
	}
	retained("imported current, match A tagged", 1, "Abe", "Ann")
	if got, err := mt.OfAnalysis(ctx, "", posA[0]); err != nil || got != id {
		t.Errorf("OfAnalysis(tagged) = %d, %v; want %d", got, err, id)
	}
	if got, err := mt.OfAnalysis(ctx, "", posB[0]); err != nil || got != 0 {
		t.Errorf("OfAnalysis(untagged) = %d, %v; want 0", got, err)
	}

	if err := mt.SetCurrent(ctx, "", 0); err != nil {
		t.Fatal(err)
	}
	retained("built-in current, match A tagged", 1, "Bea", "Bob")

	if err := mt.TagAnalyses(ctx, "", 0, posA[:]); err != nil {
		t.Fatal(err)
	}
	retained("built-in current, tag cleared", 2, "Abe", "Ann", "Bea", "Bob")
}
