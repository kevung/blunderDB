package database

// search_exclude_test.go — Tests for the "Sauf" (exclusion structure) search
// feature: positions that contain the include structure but NOT the exclude
// structure. Covers the SQL NOT pre-filter (non-tight) and the Go-side exact
// check (tight, >2 checkers on a point).

import (
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// boardWithBlackOn8 is initialBoard() (Black: 24/2, 13/5, 8/3, 6/5).
// boardWithoutBlackOn8 moves Black's 3 checkers from point 8 to point 7, so the
// race-relevant structure on point 6 is preserved but point 8 is empty for Black.
func boardWithoutBlackOn8() Board {
	b := initialBoard()
	b.Points[8] = Point{}
	b.Points[7] = Point{Checkers: 3, Color: Black}
	return b
}

// boardWithTwoBlackOn8 keeps only 2 Black checkers on point 8 (the third goes to
// point 7), so the position does NOT contain a "Black ≥3 on 8" structure but does
// set the occupancy/point-mask bits for point 8.
func boardWithTwoBlackOn8() Board {
	b := initialBoard()
	b.Points[8] = Point{Checkers: 2, Color: Black}
	b.Points[7] = Point{Checkers: 1, Color: Black}
	return b
}

// boardWithOneBlackOn8 keeps a single Black checker on point 8 (the other two go
// to point 7), so point 8 is occupied but not made for Black.
func boardWithOneBlackOn8() Board {
	b := initialBoard()
	b.Points[8] = Point{Checkers: 1, Color: Black}
	b.Points[7] = Point{Checkers: 2, Color: Black}
	return b
}

// whiteFiller places 15 White checkers in a fixed home arrangement so the boards
// used by the closeout test are distinct, well-formed positions.
func whiteFiller(b *Board) {
	b.Points[24] = Point{Checkers: 2, Color: White}
	b.Points[19] = Point{Checkers: 5, Color: White}
	b.Points[17] = Point{Checkers: 3, Color: White}
	b.Points[12] = Point{Checkers: 5, Color: White}
}

func savePos(t *testing.T, db *Database, b Board) int64 {
	t.Helper()
	p := Position{Board: b, Cube: Cube{Owner: None, Value: 0}, PlayerOnRoll: 0, DecisionType: CheckerAction}
	id, err := db.SavePosition(&p)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}
	return id
}

// TestSearch_Exclude_NonTight verifies that a non-tight exclude structure
// (Black ≥1 on point 8) removes positions that contain it while keeping
// positions that match the include structure (Black ≥1 on point 6) only.
func TestSearch_Exclude_NonTight(t *testing.T) {
	db := newTestDB(t)

	idWith := savePos(t, db, initialBoard())        // Black on 8 → must be excluded
	idWithout := savePos(t, db, boardWithoutBlackOn8()) // no Black on 8 → must be kept

	include := Position{}
	include.Board.Points[6] = Point{Color: Black, Checkers: 1} // both positions match

	exclude := Position{}
	exclude.Board.Points[8] = Point{Color: Black, Checkers: 1}

	if _, _, _, _, tight := engine.CheckerStructureMasks(exclude); tight {
		t.Fatalf("expected non-tight exclude template")
	}

	got, err := db.LoadPositionsByFilters(SearchFilters{Filter: include, ExcludeFilter: exclude})
	if err != nil {
		t.Fatalf("LoadPositionsByFilters: %v", err)
	}
	ids := sortedIDs(got)

	if slices.Contains(ids, idWith) {
		t.Errorf("position containing the exclude structure was not removed (id=%d)", idWith)
	}
	if !slices.Contains(ids, idWithout) {
		t.Errorf("position matching include but not exclude was missing (id=%d)", idWithout)
	}
}

// TestSearch_Exclude_AnyOf verifies OR semantics: a position is excluded if it
// contains ANY one of the excluded points, not only when it contains all of them.
func TestSearch_Exclude_AnyOf(t *testing.T) {
	db := newTestDB(t)

	// p1: Black single checker on point 5 (moved from point 6's stack) + point 4 kept.
	b1 := initialBoard()
	b1.Points[6] = Point{Checkers: 4, Color: Black}
	b1.Points[5] = Point{Checkers: 1, Color: Black}
	idHasOne := savePos(t, db, b1) // has Black on 5 (but nothing on 3 or 1) → excluded

	// p2: no Black on 1, 3 or 5 at all.
	idHasNone := savePos(t, db, boardWithoutBlackOn8()) // Black points 6,13,7,24 only

	include := Position{}
	include.Board.Points[6] = Point{Color: Black, Checkers: 1} // both match

	// Exclude Black on 5 AND 3 AND 1 (drawn as three single checkers).
	exclude := Position{}
	exclude.Board.Points[5] = Point{Color: Black, Checkers: 1}
	exclude.Board.Points[3] = Point{Color: Black, Checkers: 1}
	exclude.Board.Points[1] = Point{Color: Black, Checkers: 1}

	got, err := db.LoadPositionsByFilters(SearchFilters{Filter: include, ExcludeFilter: exclude})
	if err != nil {
		t.Fatalf("LoadPositionsByFilters: %v", err)
	}
	ids := sortedIDs(got)

	if slices.Contains(ids, idHasOne) {
		t.Errorf("position with Black on point 5 should be excluded (OR semantics) (id=%d)", idHasOne)
	}
	if !slices.Contains(ids, idHasNone) {
		t.Errorf("position with no Black on 1/3/5 should be kept (id=%d)", idHasNone)
	}
}

// TestSearch_Exclude_MadePoint verifies that excluding a made point (2 checkers)
// drops positions where that point is made (≥2) but keeps positions with a single
// checker there — exercising the point_mask SQL pre-filter path.
func TestSearch_Exclude_MadePoint(t *testing.T) {
	db := newTestDB(t)

	idMade := savePos(t, db, initialBoard())        // 3 Black on point 8 (made) → excluded
	idOne := savePos(t, db, boardWithOneBlackOn8())  // 1 Black on point 8 → kept

	include := Position{}
	include.Board.Points[6] = Point{Color: Black, Checkers: 1}

	exclude := Position{}
	exclude.Board.Points[8] = Point{Color: Black, Checkers: 2} // made point

	got, err := db.LoadPositionsByFilters(SearchFilters{Filter: include, ExcludeFilter: exclude})
	if err != nil {
		t.Fatalf("LoadPositionsByFilters: %v", err)
	}
	ids := sortedIDs(got)

	if slices.Contains(ids, idMade) {
		t.Errorf("position with a made point on 8 should be excluded (id=%d)", idMade)
	}
	if !slices.Contains(ids, idOne) {
		t.Errorf("position with a single checker on 8 should be kept (id=%d)", idOne)
	}
}

// TestSearch_Exclude_PrecedenceOverInclude reproduces the user's closeout example:
// "At least" a closed board on 1-6 (Black made points) and "Except" a checker on
// point 1. The exclusion must win on the shared point 1, so the search becomes
// "5 made points on 2-6 with point 1 free of Black" instead of returning nothing.
func TestSearch_Exclude_PrecedenceOverInclude(t *testing.T) {
	db := newTestDB(t)

	// Closeout 1-6 (point 1 made by Black).
	var bClosed Board
	for _, pt := range []int{1, 2, 3, 4, 5, 6} {
		bClosed.Points[pt] = Point{Checkers: 2, Color: Black}
	}
	bClosed.Points[13] = Point{Checkers: 3, Color: Black}
	whiteFiller(&bClosed)
	idClosed := savePos(t, db, bClosed) // has Black on 1 → excluded

	// 2-6 made, point 1 empty.
	var bFiveOpen Board
	for _, pt := range []int{2, 3, 4, 5, 6} {
		bFiveOpen.Points[pt] = Point{Checkers: 2, Color: Black}
	}
	bFiveOpen.Points[13] = Point{Checkers: 5, Color: Black}
	whiteFiller(&bFiveOpen)
	idFiveOpen := savePos(t, db, bFiveOpen) // no Black on 1 → kept

	// 2-6 made plus a single Black checker on point 1.
	var bFiveOn1 Board
	for _, pt := range []int{2, 3, 4, 5, 6} {
		bFiveOn1.Points[pt] = Point{Checkers: 2, Color: Black}
	}
	bFiveOn1.Points[1] = Point{Checkers: 1, Color: Black}
	bFiveOn1.Points[13] = Point{Checkers: 4, Color: Black}
	whiteFiller(&bFiveOn1)
	idFiveOn1 := savePos(t, db, bFiveOn1) // has Black on 1 → excluded

	// Include: closed board 1-6 (Black made points).
	include := Position{}
	for _, pt := range []int{1, 2, 3, 4, 5, 6} {
		include.Board.Points[pt] = Point{Color: Black, Checkers: 2}
	}
	// Except: a Black checker on point 1.
	exclude := Position{}
	exclude.Board.Points[1] = Point{Color: Black, Checkers: 1}

	got, err := db.LoadPositionsByFilters(SearchFilters{Filter: include, ExcludeFilter: exclude})
	if err != nil {
		t.Fatalf("LoadPositionsByFilters: %v", err)
	}
	ids := sortedIDs(got)

	if !slices.Contains(ids, idFiveOpen) {
		t.Errorf("the 5-point board (2-6 made, point 1 open) should match (id=%d), got %v", idFiveOpen, ids)
	}
	if slices.Contains(ids, idClosed) {
		t.Errorf("the full closeout (Black on point 1) should be excluded (id=%d)", idClosed)
	}
	if slices.Contains(ids, idFiveOn1) {
		t.Errorf("the 2-6 board with a Black checker on point 1 should be excluded (id=%d)", idFiveOn1)
	}
}

// TestSearch_Exclude_NoSpare reproduces the "closed board without a spare" example:
// At least = made points 1-6 (≥2) and Except = 3 checkers on each of 1-6 (≥3).
// Because the exclude count (3) is greater than the include count (2) the include
// is kept, so the search is "exactly 2 on each of 1-6" (no spare).
func TestSearch_Exclude_NoSpare(t *testing.T) {
	db := newTestDB(t)

	// Exactly 2 on each of 1-6 (12) + 3 elsewhere → 15.
	var bNoSpare Board
	for _, pt := range []int{1, 2, 3, 4, 5, 6} {
		bNoSpare.Points[pt] = Point{Checkers: 2, Color: Black}
	}
	bNoSpare.Points[13] = Point{Checkers: 3, Color: Black}
	whiteFiller(&bNoSpare)
	idNoSpare := savePos(t, db, bNoSpare) // kept

	// 3 on point 1 (a spare) + 2 on 2-6 (13) + 2 elsewhere → 15.
	var bSpare Board
	bSpare.Points[1] = Point{Checkers: 3, Color: Black}
	for _, pt := range []int{2, 3, 4, 5, 6} {
		bSpare.Points[pt] = Point{Checkers: 2, Color: Black}
	}
	bSpare.Points[13] = Point{Checkers: 2, Color: Black}
	whiteFiller(&bSpare)
	idSpare := savePos(t, db, bSpare) // has a spare on point 1 → excluded

	include := Position{}
	for _, pt := range []int{1, 2, 3, 4, 5, 6} {
		include.Board.Points[pt] = Point{Color: Black, Checkers: 2}
	}
	exclude := Position{}
	for _, pt := range []int{1, 2, 3, 4, 5, 6} {
		exclude.Board.Points[pt] = Point{Color: Black, Checkers: 3}
	}

	got, err := db.LoadPositionsByFilters(SearchFilters{Filter: include, ExcludeFilter: exclude})
	if err != nil {
		t.Fatalf("LoadPositionsByFilters: %v", err)
	}
	ids := sortedIDs(got)

	if !slices.Contains(ids, idNoSpare) {
		t.Errorf("the no-spare closed board (exactly 2 on 1-6) should match (id=%d), got %v", idNoSpare, ids)
	}
	if slices.Contains(ids, idSpare) {
		t.Errorf("the board with a spare on point 1 should be excluded (id=%d)", idSpare)
	}
}

// TestSearch_Exclude_EmptyMarker verifies the "must be empty" marker: a point
// flagged empty in Except drops positions with ANY checker (either colour) there.
func TestSearch_Exclude_EmptyMarker(t *testing.T) {
	db := newTestDB(t)

	idBlackOn8 := savePos(t, db, initialBoard())          // Black on point 8 → excluded
	idEmptyOn8 := savePos(t, db, boardWithoutBlackOn8())   // point 8 empty → kept

	// A position with White on point 8 must also be excluded by an empty marker.
	bWhite := boardWithoutBlackOn8()
	bWhite.Points[19] = Point{Checkers: 4, Color: White}
	bWhite.Points[8] = Point{Checkers: 1, Color: White}
	idWhiteOn8 := savePos(t, db, bWhite) // White on 8 → excluded

	include := Position{}
	include.Board.Points[6] = Point{Color: Black, Checkers: 1}

	exclude := Position{}
	exclude.Board.Points[8] = Point{Color: ExcludeEmpty, Checkers: 1} // point 8 must be empty

	got, err := db.LoadPositionsByFilters(SearchFilters{Filter: include, ExcludeFilter: exclude})
	if err != nil {
		t.Fatalf("LoadPositionsByFilters: %v", err)
	}
	ids := sortedIDs(got)

	if !slices.Contains(ids, idEmptyOn8) {
		t.Errorf("position with point 8 empty should be kept (id=%d)", idEmptyOn8)
	}
	if slices.Contains(ids, idBlackOn8) {
		t.Errorf("position with Black on point 8 should be excluded by the empty marker (id=%d)", idBlackOn8)
	}
	if slices.Contains(ids, idWhiteOn8) {
		t.Errorf("position with White on point 8 should be excluded by the empty marker (id=%d)", idWhiteOn8)
	}
}

// TestSearch_Exclude_Tight verifies the Go-side exact check for an exclude
// structure with >2 checkers on a point (Black ≥3 on point 8): a position with
// exactly 2 Black checkers on point 8 must be kept (it does not contain the
// excluded structure even though its point-8 bits are set), while the position
// with 3 must be removed.
func TestSearch_Exclude_Tight(t *testing.T) {
	db := newTestDB(t)

	idThree := savePos(t, db, initialBoard())       // 3 Black on 8 → excluded
	idTwo := savePos(t, db, boardWithTwoBlackOn8())  // 2 Black on 8 → kept

	include := Position{}
	include.Board.Points[6] = Point{Color: Black, Checkers: 1}

	exclude := Position{}
	exclude.Board.Points[8] = Point{Color: Black, Checkers: 3}

	if _, _, _, _, tight := engine.CheckerStructureMasks(exclude); !tight {
		t.Fatalf("expected tight exclude template for ≥3 checkers")
	}

	got, err := db.LoadPositionsByFilters(SearchFilters{Filter: include, ExcludeFilter: exclude})
	if err != nil {
		t.Fatalf("LoadPositionsByFilters: %v", err)
	}
	ids := sortedIDs(got)

	if slices.Contains(ids, idThree) {
		t.Errorf("position with 3 checkers on point 8 was not removed (id=%d)", idThree)
	}
	if !slices.Contains(ids, idTwo) {
		t.Errorf("position with only 2 checkers on point 8 was wrongly removed (id=%d)", idTwo)
	}
}
