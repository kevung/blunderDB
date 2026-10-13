package database

import (
	"context"
	"testing"
)

// The desktop's two export options that reach no family of their own land on
// the plan the exporter reads: the collections' positions on a list export,
// and the decks' review history on both.
func TestExportPlanCarriesCollectionPositionsAndAnkiHistory(t *testing.T) {
	d := &Database{}
	list, err := d.exportPlan(context.Background(), ExportOptions{
		PositionIDs: []int64{1}, IncludeCollections: true, CollectionIDs: []int64{7},
		IncludeCollectionPositions: true, IncludeAnkiHistory: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !list.Selection.CollectionPositions || !list.AnkiHistory {
		t.Errorf("list plan: CollectionPositions %v, AnkiHistory %v; want both", list.Selection.CollectionPositions, list.AnkiHistory)
	}
	off, err := d.exportPlan(context.Background(), ExportOptions{PositionIDs: []int64{1}, IncludeCollectionPositions: true})
	if err != nil {
		t.Fatal(err)
	}
	if off.Selection.CollectionPositions {
		t.Error("collections left out still bring their positions")
	}
	if whole := wholeLibraryPlan(ExportOptions{WholeLibrary: true}); whole.AnkiHistory {
		t.Error("whole library carries the review history it was not asked for")
	}
}

// A tournament chosen leaves with its matches, whichever way the matches family is set.
func TestExportPlanTournamentCarriesItsMatches(t *testing.T) {
	d := &Database{}
	list, err := d.exportPlan(context.Background(), ExportOptions{
		PositionIDs: []int64{1}, IncludeTournaments: true, TournamentIDs: []int64{3},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !list.Selection.TournamentMatches {
		t.Error("list plan: a chosen tournament leaves without its matches")
	}
	none, err := d.exportPlan(context.Background(), ExportOptions{PositionIDs: []int64{1}})
	if err != nil {
		t.Fatal(err)
	}
	if none.Selection.TournamentMatches {
		t.Error("list plan: matches asked for with no tournament")
	}
	for name, opts := range map[string]ExportOptions{
		"picked":   {WholeLibrary: true, IncludeMatches: false, IncludeTournaments: true, TournamentIDs: []int64{3}},
		"restrict": {WholeLibrary: true, IncludeMatches: true, MatchIDs: []int64{9}, IncludeTournaments: true, TournamentIDs: []int64{3}},
		"all":      {WholeLibrary: true, IncludeTournaments: true},
	} {
		if plan := wholeLibraryPlan(opts); !plan.Selection.TournamentMatches {
			t.Errorf("whole library (%s): a tournament leaves without its matches", name)
		}
	}
	if plan := wholeLibraryPlan(ExportOptions{WholeLibrary: true}); plan.Selection.TournamentMatches {
		t.Error("whole library: matches asked for with no tournament")
	}
}
