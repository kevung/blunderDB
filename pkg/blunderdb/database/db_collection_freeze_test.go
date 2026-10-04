package database

import (
	"path/filepath"
	"slices"
	"testing"
)

// Freezing keeps what the query selected at that moment, drops what the old
// rows held, and stops obeying the query afterwards.
func TestFreezeCollection_KeepsTodaysResultAndStopsObeyingTheQuery(t *testing.T) {
	t.Parallel()
	db := NewDatabase()
	if err := db.SetupDatabase(filepath.Join(t.TempDir(), "freeze.db")); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer db.Close()

	var ids []int64
	for _, c := range []int{2, 3, 4} {
		p := InitializePosition()
		p.Board.Points[6].Checkers = c
		id, err := db.SavePosition(&p)
		if err != nil {
			t.Fatalf("SavePosition: %v", err)
		}
		ids = append(ids, id)
	}
	colID, err := db.CreateCollection("c", "")
	if err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	if _, err := db.FreezeCollection(colID); err == nil {
		t.Fatal("freezing a hand-made collection must be refused")
	}
	if err := db.AddPositionToCollection(colID, ids[0]); err != nil {
		t.Fatalf("AddPositionToCollection: %v", err)
	}
	// No move ever reached any position: n<1 selects all three.
	if err := db.SetCollectionFilter(colID, "s n<1"); err != nil {
		t.Fatalf("SetCollectionFilter: %v", err)
	}
	n, err := db.FreezeCollection(colID)
	if err != nil || n != 3 {
		t.Fatalf("FreezeCollection = %d, %v; want 3", n, err)
	}
	got, err := db.ListCollectionPositionIDs(colID, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	want := slices.Clone(ids)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("frozen membership = %v, want %v", got, want)
	}
	if q, _ := db.collectionFilterQuery(colID); q != "" {
		t.Fatalf("query still set after freeze: %q", q)
	}
}
