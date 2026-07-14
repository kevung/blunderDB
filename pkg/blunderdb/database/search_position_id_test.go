package database

// search_position_id_test.go — regression tests for the position-id filter
// (command-line token `id`), exposed via SearchFilters.PositionIDsFilter.
// It mirrors the match/tournament ID convention: "a,b" is the inclusive range
// a..b, while ";"-joined values are an explicit list.

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestSearchPositionIDFilter(t *testing.T) {
	dir := t.TempDir()
	db := NewDatabase()
	if err := db.SetupDatabase(filepath.Join(dir, "search_position_id.db")); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer db.Close()

	// Three distinct positions (dice differentiate them so dedup keeps all three).
	pos1 := InitializePosition()
	pos1.Dice = [2]int{2, 1}
	id1, err := db.SavePosition(&pos1)
	if err != nil {
		t.Fatalf("SavePosition 1: %v", err)
	}
	pos2 := InitializePosition()
	pos2.Dice = [2]int{4, 3}
	id2, err := db.SavePosition(&pos2)
	if err != nil {
		t.Fatalf("SavePosition 2: %v", err)
	}
	pos3 := InitializePosition()
	pos3.Dice = [2]int{6, 5}
	id3, err := db.SavePosition(&pos3)
	if err != nil {
		t.Fatalf("SavePosition 3: %v", err)
	}

	cases := []struct {
		name   string
		filter string
		want   []int64
	}{
		{"single", fmt.Sprintf("%d", id2), []int64{id2}},
		{"range", fmt.Sprintf("%d,%d", id1, id3), []int64{id1, id2, id3}},
		{"explicit list", fmt.Sprintf("%d;%d", id1, id3), []int64{id1, id3}},
		{"three-item comma list", fmt.Sprintf("%d,%d,%d", id1, id2, id3), []int64{id1, id2, id3}},
		{"empty matches all", "", []int64{id1, id2, id3}},
		{"unknown id", "99999", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			positions, err := db.LoadPositionsByFilters(SearchFilters{PositionIDsFilter: tc.filter})
			if err != nil {
				t.Fatalf("LoadPositionsByFilters(%q): %v", tc.filter, err)
			}
			var got []int64
			for _, p := range positions {
				got = append(got, p.ID)
			}
			if !sameIDSet(got, tc.want) {
				t.Errorf("PositionIDsFilter %q: got %v, want %v", tc.filter, got, tc.want)
			}
		})
	}
}
