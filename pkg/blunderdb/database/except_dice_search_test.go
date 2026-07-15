package database

import (
	"fmt"
	"testing"
)

// TestSearch_ExceptDice verifies the xD<d1><d2> "except dice" discriminator:
// positions rolled with the excluded roll (in either order) are removed and all
// others survive. Data-driven against the imported XG fixture so it doesn't hard
// code the fixture's roll distribution.
func TestSearch_ExceptDice(t *testing.T) {
	db := setupSearchTestDB(t)

	all, err := db.LoadPositionsByFilters(SearchFilters{Filter: Position{}})
	if err != nil {
		t.Fatalf("load all: %v", err)
	}

	// Pick a non-double roll actually present in the fixture.
	var d1, d2 int
	for _, p := range all {
		if p.Dice[0] >= 1 && p.Dice[0] <= 6 && p.Dice[1] >= 1 && p.Dice[1] <= 6 && p.Dice[0] != p.Dice[1] {
			d1, d2 = p.Dice[0], p.Dice[1]
			break
		}
	}
	if d1 == 0 {
		t.Skip("no rolled (non-double) positions in fixture")
	}

	excluded := 0
	for _, p := range all {
		if (p.Dice[0] == d1 && p.Dice[1] == d2) || (p.Dice[0] == d2 && p.Dice[1] == d1) {
			excluded++
		}
	}
	if excluded == 0 {
		t.Fatalf("chosen roll %d-%d not present after all", d1, d2)
	}

	got, err := db.LoadPositionsByFilters(SearchFilters{
		Filter:           Position{},
		ExceptDiceFilter: fmt.Sprintf("%d%d", d1, d2),
	})
	if err != nil {
		t.Fatalf("load except-dice: %v", err)
	}

	// No survivor carries the excluded roll (either order).
	for _, p := range got {
		if (p.Dice[0] == d1 && p.Dice[1] == d2) || (p.Dice[0] == d2 && p.Dice[1] == d1) {
			t.Errorf("position %d survived with excluded roll %d-%d", p.ID, d1, d2)
		}
	}
	if want := len(all) - excluded; len(got) != want {
		t.Errorf("count mismatch: got %d, want %d (all %d − excluded %d)", len(got), want, len(all), excluded)
	}

	// Reversed-order token excludes the same set (order-insensitive).
	gotRev, err := db.LoadPositionsByFilters(SearchFilters{
		Filter:           Position{},
		ExceptDiceFilter: fmt.Sprintf("%d%d", d2, d1),
	})
	if err != nil {
		t.Fatalf("load except-dice reversed: %v", err)
	}
	if len(gotRev) != len(got) {
		t.Errorf("order sensitivity: %d-%d gave %d, %d-%d gave %d", d1, d2, len(got), d2, d1, len(gotRev))
	}
}
