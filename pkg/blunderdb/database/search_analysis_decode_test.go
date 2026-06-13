package database

import (
	"path/filepath"
	"testing"
)

// TestSearchMovePatternDecodesAnalysis guards the fix for the analysis blob
// decode in the search path: analysis.data is stored zlib-compressed, so the
// search must decompress it (engine.DecodeAnalysisFromStorage) before the
// Go-side analysis filters can use it. A plain json.Unmarshal of the raw bytes
// silently failed, leaving the decoded analysis nil and making the move-pattern
// filter (and the mirror-search win/gammon fallbacks) match nothing.
func TestSearchMovePatternDecodesAnalysis(t *testing.T) {
	db := newTestDB(t)
	if _, err := db.ImportXGMatch(filepath.Join("testdata", "test.xg")); err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}

	all, err := db.LoadPositionsByFilters(SearchFilters{Filter: emptyFilter()})
	if err != nil {
		t.Fatalf("baseline search: %v", err)
	}
	if len(all) == 0 {
		t.Fatal("no positions imported")
	}

	// "/" appears in every checker-move notation, so a move-pattern search must
	// match the checker-move positions — which is only possible if the
	// compressed analysis blob was decoded.
	mp, err := db.LoadPositionsByFilters(SearchFilters{Filter: emptyFilter(), MovePatternFilter: `m"/"`})
	if err != nil {
		t.Fatalf("move-pattern search: %v", err)
	}
	if len(mp) == 0 {
		t.Fatal("move-pattern search returned 0 — analysis blob not decoded")
	}
	if len(mp) > len(all) {
		t.Fatalf("move-pattern matches (%d) exceed total positions (%d)", len(mp), len(all))
	}
}
