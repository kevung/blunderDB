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
