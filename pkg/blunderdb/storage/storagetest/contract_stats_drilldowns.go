package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testStatsCountDrilldowns holds the lists two counts of the Stats panel
// open to the figures they show: TournamentIDs to Totals.NumTournaments, the
// "blunders" selection to a Players row's Blunders.
func testStatsCountDrilldowns(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	matchA, _ := statsFixtureMatch(t, s, 0, "Alice", "Bob")
	matchB, _ := statsFixtureMatch(t, s, 2, "Alice", "Carol")
	matchC, _ := statsFixtureMatch(t, s, 4, "Dave", "Erin")
	statsFixtureMatch(t, s, 6, "Alice", "Dave") // in no tournament

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
	all, err := s.Stats().TournamentIDs(ctx, "", storage.StatsFilter{DecisionType: -1})
	if err != nil || len(all) != 2 || all[0] != min(cup, open) || all[1] != max(cup, open) {
		t.Errorf("TournamentIDs(all) = %v, %v; want [Cup Open] by id", all, err)
	}

	// The fixture's decisions cost 50 mp: none is a blunder at the default
	// threshold, every one is at 40.
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
			if err != nil || len(ids) != r.Blunders {
				t.Errorf("%s: blunders of %s = %v, %v; want the row's %d", stage, r.Name, ids, err, r.Blunders)
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
}
