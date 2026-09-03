package database

// search_ids_test.go — LoadPositionIDsByFilters (D.8, #208) must return
// exactly the ids of what LoadPositionsByFilters returns, in the same order:
// it is the GUI-facing shortcut that lets the frontend keep a search result
// as an id list (positionsStore / positionList.js) instead of shipping every
// matching position whole across the Wails bridge.

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadPositionIDsByFilters_MatchesLoadPositionsByFilters(t *testing.T) {
	dir := t.TempDir()
	db := NewDatabase()
	if err := db.SetupDatabase(filepath.Join(dir, "search_ids.db")); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer db.Close()

	var ids []int64
	for i := 0; i < 5; i++ {
		pos := InitializePosition()
		pos.Dice = [2]int{(i % 6) + 1, ((i + 1) % 6) + 1}
		id, err := db.SavePosition(&pos)
		if err != nil {
			t.Fatalf("SavePosition %d: %v", i, err)
		}
		ids = append(ids, id)
	}

	cases := []struct {
		name   string
		filter SearchFilters
	}{
		{"no filter (matches everything)", SearchFilters{}},
		{"position id range", SearchFilters{PositionIDsFilter: fmt.Sprintf("%d,%d", ids[0], ids[2])}},
		{"unknown id (empty result)", SearchFilters{PositionIDsFilter: "999999"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			positions, err := db.LoadPositionsByFilters(tc.filter)
			if err != nil {
				t.Fatalf("LoadPositionsByFilters: %v", err)
			}
			var wantIDs []int64
			for _, p := range positions {
				wantIDs = append(wantIDs, p.ID)
			}

			gotIDs, err := db.LoadPositionIDsByFilters(tc.filter)
			if err != nil {
				t.Fatalf("LoadPositionIDsByFilters: %v", err)
			}

			if !reflect.DeepEqual(gotIDs, wantIDs) {
				t.Errorf("LoadPositionIDsByFilters = %v, want %v (LoadPositionsByFilters's ids, same order)", gotIDs, wantIDs)
			}
		})
	}
}
