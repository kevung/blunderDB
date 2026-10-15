package ingest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// plan reads opts without a store: none of these options asks which matches
// a list of positions reaches.
func plan(t *testing.T, opts domain.ExportOptions) ExportOptions {
	t.Helper()
	p, err := OptionsFor(context.Background(), nil, "", opts)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// The desktop's two export options that reach no family of their own land on
// the plan the exporter reads: the collections' positions on a list export,
// and the decks' review history on both.
func TestExportPlanCarriesCollectionPositionsAndAnkiHistory(t *testing.T) {
	list := plan(t, domain.ExportOptions{
		PositionIDs: []int64{1}, IncludeCollections: true, CollectionIDs: []int64{7},
		IncludeCollectionPositions: true, IncludeAnkiHistory: true,
	})
	if !list.Selection.CollectionPositions || !list.AnkiHistory {
		t.Errorf("list plan: CollectionPositions %v, AnkiHistory %v; want both", list.Selection.CollectionPositions, list.AnkiHistory)
	}
	off := plan(t, domain.ExportOptions{PositionIDs: []int64{1}, IncludeCollectionPositions: true})
	if off.Selection.CollectionPositions {
		t.Error("collections left out still bring their positions")
	}
	if whole := plan(t, domain.ExportOptions{WholeLibrary: true}); whole.AnkiHistory {
		t.Error("whole library carries the review history it was not asked for")
	}
}

// A tournament chosen leaves with its matches, whichever way the matches family is set.
func TestExportPlanTournamentCarriesItsMatches(t *testing.T) {
	list := plan(t, domain.ExportOptions{
		PositionIDs: []int64{1}, IncludeTournaments: true, TournamentIDs: []int64{3},
	})
	if !list.Selection.TournamentMatches {
		t.Error("list plan: a chosen tournament leaves without its matches")
	}
	none := plan(t, domain.ExportOptions{PositionIDs: []int64{1}})
	if none.Selection.TournamentMatches {
		t.Error("list plan: matches asked for with no tournament")
	}
	for name, opts := range map[string]domain.ExportOptions{
		"picked":   {WholeLibrary: true, IncludeMatches: false, IncludeTournaments: true, TournamentIDs: []int64{3}},
		"restrict": {WholeLibrary: true, IncludeMatches: true, MatchIDs: []int64{9}, IncludeTournaments: true, TournamentIDs: []int64{3}},
		"all":      {WholeLibrary: true, IncludeTournaments: true},
	} {
		if p := plan(t, opts); !p.Selection.TournamentMatches {
			t.Errorf("whole library (%s): a tournament leaves without its matches", name)
		}
	}
	if p := plan(t, domain.ExportOptions{WholeLibrary: true}); p.Selection.TournamentMatches {
		t.Error("whole library: matches asked for with no tournament")
	}
}
