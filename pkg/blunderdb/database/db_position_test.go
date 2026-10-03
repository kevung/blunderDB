package database

import (
	"path/filepath"
	"slices"
	"testing"
)

// The GUI browses a library as an id list plus windows of positions fetched
// on demand. The two calls have to agree with LoadAllPositions on order, and
// the window loader must keep the caller's order and skip a missing id.
func TestListPositionIDsAndLoadPositionsByIDs(t *testing.T) {
	t.Parallel()
	db := NewDatabase()
	if err := db.SetupDatabase(filepath.Join(t.TempDir(), "test.db")); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer db.Close()

	if _, err := db.ImportXGMatch(filepath.Join("testdata", "test.xg")); err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}

	all, err := db.LoadAllPositions()
	if err != nil {
		t.Fatalf("LoadAllPositions: %v", err)
	}
	if len(all) < 3 {
		t.Fatalf("fixture too small: %d positions", len(all))
	}
	wantIDs := make([]int64, len(all))
	for i, p := range all {
		wantIDs[i] = p.ID
	}

	ids, err := db.ListPositionIDs(0, 0)
	if err != nil {
		t.Fatalf("ListPositionIDs: %v", err)
	}
	if !slices.Equal(ids, wantIDs) {
		t.Fatalf("ListPositionIDs order differs from LoadAllPositions:\n got %v\nwant %v", ids, wantIDs)
	}

	// The windowed face the GUI browses with: a window, the count, a rank.
	window, err := db.ListPositionIDs(1, 2)
	if err != nil {
		t.Fatalf("ListPositionIDs(1, 2): %v", err)
	}
	if !slices.Equal(window, wantIDs[1:3]) {
		t.Errorf("ListPositionIDs(1, 2): got %v, want %v", window, wantIDs[1:3])
	}
	if n, err := db.CountPositions(); err != nil || n != len(wantIDs) {
		t.Errorf("CountPositions: got %d, %v; want %d", n, err, len(wantIDs))
	}
	last := len(wantIDs) - 1
	if i, err := db.IndexOfPosition(wantIDs[last]); err != nil || i != last {
		t.Errorf("IndexOfPosition(last): got %d, %v; want %d", i, err, last)
	}
	if i, err := db.IndexOfPosition(987654321); err != nil || i != -1 {
		t.Errorf("IndexOfPosition(unknown): got %d, %v; want -1", i, err)
	}

	// A window in the caller's order (reversed here) with one id that no
	// longer exists in the middle.
	want := []int64{wantIDs[2], wantIDs[0], wantIDs[1]}
	got, err := db.LoadPositionsByIDs([]int64{wantIDs[2], 987654321, wantIDs[0], wantIDs[1]})
	if err != nil {
		t.Fatalf("LoadPositionsByIDs: %v", err)
	}
	gotIDs := make([]int64, len(got))
	for i, p := range got {
		gotIDs[i] = p.ID
	}
	if !slices.Equal(gotIDs, want) {
		t.Fatalf("LoadPositionsByIDs order: got %v, want %v", gotIDs, want)
	}
	if got[1].Board != all[0].Board {
		t.Errorf("LoadPositionsByIDs returned a different board than LoadAllPositions for id %d", wantIDs[0])
	}

	empty, err := db.LoadPositionsByIDs(nil)
	if err != nil {
		t.Fatalf("LoadPositionsByIDs(nil): %v", err)
	}
	if empty == nil || len(empty) != 0 {
		t.Errorf("LoadPositionsByIDs(nil) = %v, want an empty (non-nil) slice for the JSON bridge", empty)
	}
}
