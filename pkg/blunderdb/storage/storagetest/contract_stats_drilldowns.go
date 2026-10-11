package storagetest

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testStatsCountDrilldowns holds the lists two counts of the Stats panel
// open to the figures they show: TournamentIDs to Totals.NumTournaments, the
// "blunders" selection to a Players row's Blunders.
func testStatsCountDrilldowns(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	matchA, posA := statsFixtureMatch(t, s, 0, "Alice", "Bob")
	matchB, posB := statsFixtureMatch(t, s, 2, "Alice", "Carol")
	// Played late on the last day of June: a filter ending that day keeps it.
	matchC, _ := statsFixtureMatchOn(t, s, 4, "Dave", "Erin", 0, time.Date(2025, 6, 30, 18, 0, 0, 0, time.UTC))
	_, posD := statsFixtureMatch(t, s, 6, "Alice", "Dave") // in no tournament

	cup, err := s.Tournaments().Create(ctx, "", "Cup", "2025-06-01", "Paris")
	if err != nil {
		t.Fatalf("Create Cup: %v", err)
	}
	open, err := s.Tournaments().Create(ctx, "", "Open", "2025-06-02", "Lyon")
	if err != nil {
		t.Fatalf("Create Open: %v", err)
	}
	// A tournament without a match holds no decision: never counted.
	if _, err := s.Tournaments().Create(ctx, "", "Empty", "2025-06-03", "Nice"); err != nil {
		t.Fatalf("Create Empty: %v", err)
	}
	for _, am := range [][2]int64{{cup, matchA}, {cup, matchB}, {open, matchC}} {
		if err := s.Tournaments().AddMatch(ctx, "", am[0], am[1]); err != nil {
			t.Fatalf("AddMatch %v: %v", am, err)
		}
	}

	for _, f := range []storage.StatsFilter{
		{DecisionType: -1},
		{DecisionType: -1, PlayerName: "Alice"},
		{DecisionType: -1, PlayerName: "Erin"},
		{DecisionType: -1, TournamentIDs: []int64{open}},
		{DecisionType: 1},
		{DecisionType: 0, PlayerName: "Alice"},
		{DecisionType: -1, DateFrom: "2025-06-30", DateTo: "2025-06-30"},
		{DecisionType: -1, DateTo: "2025-06-29"},
		{DecisionType: -1, AnalysisEngine: "XG"},
		{DecisionType: -1, MinAnalysisDepth: 1},
	} {
		res, err := s.Stats().Compute(ctx, "", f)
		if err != nil {
			t.Fatalf("Compute(%+v): %v", f, err)
		}
		ids, err := s.Stats().TournamentIDs(ctx, "", f)
		if err != nil || ids == nil || len(ids) != res.Totals.NumTournaments {
			t.Errorf("TournamentIDs(%+v) = %v, %v; want the %d tournaments of Totals", f, ids, err, res.Totals.NumTournaments)
		}
	}
	engines, err := s.Stats().AnalysisEngines(ctx, "")
	if err != nil {
		t.Fatalf("AnalysisEngines: %v", err)
	}
	for _, e := range engines {
		f := storage.StatsFilter{DecisionType: -1, AnalysisEngine: e}
		res, err := s.Stats().Compute(ctx, "", f)
		if err != nil {
			t.Fatalf("Compute(%q): %v", e, err)
		}
		if ids, err := s.Stats().TournamentIDs(ctx, "", f); err != nil || len(ids) != res.Totals.NumTournaments {
			t.Errorf("TournamentIDs(engine %q) = %v, %v; want %d", e, ids, err, res.Totals.NumTournaments)
		}
	}
	// The last day of a date filter is a whole day.
	lastDay, err := s.Stats().TournamentIDs(ctx, "", storage.StatsFilter{DecisionType: -1, DateFrom: "2025-06-30", DateTo: "2025-06-30"})
	if err != nil || len(lastDay) != 1 || lastDay[0] != open {
		t.Errorf("TournamentIDs(30 June) = %v, %v; want [Open], its match timed that evening", lastDay, err)
	}
	if res, err := s.Stats().Compute(ctx, "", storage.StatsFilter{DecisionType: -1, DateTo: "2025-06-30"}); err != nil || res.Totals.NumMatches != 4 {
		t.Errorf("Compute(to 30 June) matches = %v, %v; want all 4", res, err)
	}
	all, err := s.Stats().TournamentIDs(ctx, "", storage.StatsFilter{DecisionType: -1})
	if err != nil || len(all) != 2 || all[0] != min(cup, open) || all[1] != max(cup, open) {
		t.Errorf("TournamentIDs(all) = %v, %v; want [Cup Open] by id", all, err)
	}

	// The fixture's decisions cost 50 mp: none is a blunder at the default
	// threshold, every one is at 40. A second match at slot 0 has Alice and
	// Bob decide the same positions again: the column counts decisions, the
	// selection the distinct positions holding one.
	matchA2, _ := statsFixtureMatch(t, s, 0, "Alice", "Bob")
	if matchA2 == matchA {
		t.Fatal("the second slot-0 match is the first one")
	}
	want := map[string][]int64{"Alice": {posA[0], posB[0], posD[0]}, "Bob": {posA[1]}}
	checkBlunders := func(stage string, wantSome bool) {
		t.Helper()
		rows, err := s.Stats().PlayerTable(ctx, "", storage.StatsFilter{DecisionType: -1})
		if err != nil {
			t.Fatalf("%s: PlayerTable: %v", stage, err)
		}
		some := false
		for _, r := range rows {
			ids, err := s.Stats().PositionIDsBySelection(ctx, "",
				storage.StatsFilter{DecisionType: -1, PlayerName: r.Name}, storage.SelectionSpec{Kind: "blunders"})
			if err != nil || len(ids) > r.Blunders {
				t.Errorf("%s: blunders of %s = %v, %v; want at most the row's %d", stage, r.Name, ids, err, r.Blunders)
			}
			if exp, ok := want[r.Name]; ok {
				if !wantSome {
					exp = nil
				}
				if !sameIDs(ids, exp) {
					t.Errorf("%s: blunder positions of %s = %v, want %v", stage, r.Name, ids, exp)
				}
			}
			some = some || r.Blunders > 0
		}
		if some != wantSome {
			t.Errorf("%s: some row has blunders = %v, want %v", stage, some, wantSome)
		}
	}
	checkBlunders("default threshold", false)
	settings := storage.DefaultLibrarySettings()
	settings.ErrorThresholdMP, settings.BlunderThresholdMP = 20, 40
	if err := s.LibrarySettings().Save(ctx, "", settings); err != nil {
		t.Fatalf("Save library settings: %v", err)
	}
	checkBlunders("threshold at 40", true)
	rows, err := s.Stats().PlayerTable(ctx, "", storage.StatsFilter{DecisionType: -1})
	if err != nil {
		t.Fatalf("PlayerTable: %v", err)
	}
	for _, r := range rows {
		if r.Name == "Bob" && r.Blunders != 2 {
			t.Errorf("Bob's Blunders = %d, want his 2 decisions on one position", r.Blunders)
		}
	}
}

// sameIDs reports whether a and b hold the same ids, in any order.
func sameIDs(a, b []int64) bool {
	a, b = slices.Clone(a), slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}
