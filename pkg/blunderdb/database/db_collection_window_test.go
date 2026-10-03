package database

// db_collection_window_test.go — a collection is browsed by windows (ids, count, rank), hand-made
// or living, and each answer agrees with the whole list GetCollectionPositions returns.

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestCollectionWindowsAgreeWithTheWholeList(t *testing.T) {
	t.Parallel()
	db := NewDatabase()
	if err := db.SetupDatabase(filepath.Join(t.TempDir(), "windows.db")); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer db.Close()

	ids := make([]int64, 4)
	for i := range ids {
		p := InitializePosition()
		p.Board.Points[6].Checkers = i + 2
		id, err := db.SavePosition(&p)
		if err != nil {
			t.Fatalf("SavePosition: %v", err)
		}
		ids[i] = id
	}

	handMade, err := db.CreateCollection("hand-made", "")
	if err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	if err := db.AddPositionsToCollection(handMade, []int64{ids[2], ids[0], ids[3]}); err != nil {
		t.Fatalf("AddPositionsToCollection: %v", err)
	}
	living, err := db.CreateCollection("living", "")
	if err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	// Nothing was ever played: every position has been met fewer than once.
	if err := db.SetCollectionFilter(living, "s n<1"); err != nil {
		t.Fatalf("SetCollectionFilter: %v", err)
	}

	for name, id := range map[string]int64{"hand-made": handMade, "living": living} {
		whole, err := db.GetCollectionPositions(id)
		if err != nil {
			t.Fatalf("%s: GetCollectionPositions: %v", name, err)
		}
		want := make([]int64, len(whole))
		for i, p := range whole {
			want[i] = p.ID
		}
		if len(want) < 3 {
			t.Fatalf("%s: only %d positions", name, len(want))
		}

		if n, err := db.CountCollectionPositions(id); err != nil || n != len(want) {
			t.Errorf("%s: CountCollectionPositions = %d, %v; want %d", name, n, err, len(want))
		}
		if got, err := db.ListCollectionPositionIDs(id, 0, 0); err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("%s: ListCollectionPositionIDs(0, 0) = %v, %v; want %v", name, got, err, want)
		}
		if got, err := db.ListCollectionPositionIDs(id, 1, 2); err != nil || !reflect.DeepEqual(got, want[1:3]) {
			t.Errorf("%s: ListCollectionPositionIDs(1, 2) = %v, %v; want %v", name, got, err, want[1:3])
		}
		for rank, pid := range want {
			if got, err := db.IndexOfCollectionPosition(id, pid); err != nil || got != rank {
				t.Errorf("%s: IndexOfCollectionPosition(%d) = %d, %v; want %d", name, pid, got, err, rank)
			}
		}
		if got, err := db.IndexOfCollectionPosition(id, 987654); err != nil || got != -1 {
			t.Errorf("%s: IndexOfCollectionPosition of a stranger = %d, %v; want -1", name, got, err)
		}
	}
}
