package storagetest

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

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
			t.Errorf("%s: %d matches with a retained decision, want %d", label, res.Totals.NumMatches, wantMatches)
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

	// A match whose analyses are partly tagged keeps, decision by decision,
	// the ones valued with the current table, and is never whole under either
	// table for the per-match aggregates.
	if err := mt.TagAnalyses(ctx, "", id, posB[:1]); err != nil {
		t.Fatal(err)
	}
	retained("imported current, match B half tagged", 2, "Abe", "Ann")
	if err := mt.SetCurrent(ctx, "", 0); err != nil {
		t.Fatal(err)
	}
	retained("built-in current, match A tagged, match B half tagged", 1)
	if err := mt.TagAnalyses(ctx, "", 0, posB[:1]); err != nil {
		t.Fatal(err)
	}
	retained("built-in current, match A tagged", 1, "Bea", "Bob")

	if err := mt.TagAnalyses(ctx, "", 0, posA[:]); err != nil {
		t.Fatal(err)
	}
	retained("built-in current, tag cleared", 2, "Abe", "Ann", "Bea", "Bob")
}

// testMETMixedMatchLeavesThePlayersTable: the players table is a per-match
// aggregate, so a match only part of whose analyses carry the current table
// gives none of its decisions to PR or MWC7, while a match whole under it
// gives them all (ADR-0068 §4). The provenance filter takes the pass over the
// decisions instead of match_stats; selecting every analysis, it must answer
// the same table.
func testMETMixedMatchLeavesThePlayersTable(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	at := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	_, posA := statsFixtureMatchBy(t, s, 0, "Ann", "Abe", 0, at, "XG")
	_, posB := statsFixtureMatchBy(t, s, 10, "Bea", "Bob", 0, at, "XG")
	mt := s.MatchEquityTables()

	check := func(label string, whole, mixed []string) {
		t.Helper()
		var tables []string
		for _, f := range []storage.StatsFilter{{DecisionType: -1}, {DecisionType: -1, AnalysisEngine: "XG"}} {
			rows, err := s.Stats().PlayerTable(ctx, "", f)
			if err != nil {
				t.Fatalf("%s: PlayerTable(%+v): %v", label, f, err)
			}
			byName := map[string]storage.PlayerRow{}
			for _, r := range rows {
				byName[r.Name] = r
			}
			for _, n := range whole {
				r := byName[n]
				if r.Decisions != 1 || r.PR <= 0 || !r.MWC7.Available {
					t.Errorf("%s, engine %q: %s decisions=%d PR=%v MWC7=%+v, want its one decision counted",
						label, f.AnalysisEngine, n, r.Decisions, r.PR, r.MWC7)
				}
			}
			for _, n := range mixed {
				if r, ok := byName[n]; ok && (r.Decisions != 0 || r.PR != 0 || r.MWC7.Available) {
					t.Errorf("%s, engine %q: %s decisions=%d PR=%v MWC7=%+v, want its mixed match left out",
						label, f.AnalysisEngine, n, r.Decisions, r.PR, r.MWC7)
				}
			}
			b, err := json.Marshal(rows)
			if err != nil {
				t.Fatal(err)
			}
			tables = append(tables, string(b))
		}
		if tables[0] != tables[1] {
			t.Errorf("%s: players table over match_stats\n%s\nover the decisions\n%s", label, tables[0], tables[1])
		}
	}

	id, err := mt.Save(ctx, "", domain.MatchEquityTable{Name: "Rockwell-Kazaross", Digest: "rk", Source: "<met/>"})
	if err != nil {
		t.Fatal(err)
	}
	if err := mt.SetCurrent(ctx, "", id); err != nil {
		t.Fatal(err)
	}
	if err := mt.TagAnalyses(ctx, "", id, append(posA[:], posB[0])); err != nil {
		t.Fatal(err)
	}
	check("imported current, match B half tagged", []string{"Abe", "Ann"}, []string{"Bea", "Bob"})
	if err := mt.SetCurrent(ctx, "", 0); err != nil {
		t.Fatal(err)
	}
	check("built-in current, match B half tagged", nil, []string{"Abe", "Ann", "Bea", "Bob"})
	if err := mt.TagAnalyses(ctx, "", 0, posB[:1]); err != nil {
		t.Fatal(err)
	}
	check("built-in current, match A tagged", []string{"Bea", "Bob"}, []string{"Abe", "Ann"})
}
