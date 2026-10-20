package gui

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
)

// The Elsewhere view reads the waiting groups only, by hash, and writes
// nothing: the active library is not among its answers.
func TestElsewhereInGroups_ReadsTheWaitingGroupsByHash(t *testing.T) {
	dir := t.TempDir()
	pathA := newLibraryFile(t, dir, "a.db", 3)
	pathB := newLibraryFile(t, dir, "b.db", 0)
	db := database.NewDatabase()
	a := NewApp(db)
	t.Cleanup(func() { a.closeParkedLibraries(); _ = db.Close() })

	if _, err := a.OpenLibraryGroup(pathA); err != nil {
		t.Fatal(err)
	}
	// The second position of A, commented there.
	board := database.Position{Board: database.Board{Points: [26]database.Point{{Checkers: 2, Color: 0}}}, Cube: database.Cube{Value: 0, Owner: -1}}
	idA, err := db.SavePosition(&board)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AddComment(idA, "seen in A"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.OpenLibraryGroup(pathB); err != nil {
		t.Fatal(err)
	}

	// The board as B's screen holds it: an id that names nothing in A.
	asked := board
	asked.ID = 999
	view, err := a.ElsewhereInGroups(asked, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Groups) != 1 || view.Groups[0].Group != 1 || view.Groups[0].Path != database.OpenPath(a.libs.parked[1]) {
		t.Fatalf("groups: %+v", view.Groups)
	}
	got := view.Groups[0]
	if got.PositionID != idA {
		t.Errorf("positionId = %d, want A's %d", got.PositionID, idA)
	}
	if len(got.Comments) != 1 || got.Comments[0].Text != "seen in A" {
		t.Errorf("comments = %+v", got.Comments)
	}
	if len(got.Neighbours) == 0 || len(got.Neighbours) > 2 {
		t.Errorf("neighbours = %d, want 1 or 2", len(got.Neighbours))
	}
	if view.Zobrist == 0 {
		t.Error("no hash")
	}
	if n, _ := db.CountPositions(); n != 0 {
		t.Errorf("the active library holds %d positions after the read, want 0", n)
	}
}

// With one group open there is nowhere else to read.
func TestElsewhereInGroups_OneGroupHasNoOtherLibrary(t *testing.T) {
	dir := t.TempDir()
	db := database.NewDatabase()
	a := NewApp(db)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := a.OpenLibraryGroup(newLibraryFile(t, dir, "a.db", 1)); err != nil {
		t.Fatal(err)
	}
	view, err := a.ElsewhereInGroups(database.Position{Cube: database.Cube{Owner: -1}}, 0)
	if err != nil || len(view.Groups) != 0 {
		t.Fatalf("view %+v, err %v", view, err)
	}
}
