package service

import "testing"

// Two people with the same places, taken in another order, sum to totals that differ by float
// noise: they share their rank, and the order falls back on the best place and the name.
func TestRankRowsSharedPlacesInOtherOrders(t *testing.T) {
	a, b := 0.1, 0.2
	rows := []SeasonRow{
		{Name: "Zoe", Best: 1, Total: a + b},
		{Name: "Ada", Best: 1, Total: 0.3},
		{Name: "Bob", Best: 2, Total: 0.3},
	}
	rankRows(rows)
	if rows[0].Name != "Ada" || rows[1].Name != "Zoe" || rows[2].Name != "Bob" {
		t.Fatalf("order: %+v", rows)
	}
	if rows[0].Rank != 1 || rows[1].Rank != 1 || rows[2].Rank != 3 {
		t.Errorf("ranks: %+v", rows)
	}
}
