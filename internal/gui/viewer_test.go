package gui

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
)

// The viewer reads the group it names, whichever is active, and writes
// nothing: after a switch the same group number still reads the same library.
func TestViewer_ReadsTheNamedGroupAcrossASwitch(t *testing.T) {
	dir := t.TempDir()
	pathA := newLibraryFile(t, dir, "a.db", 3)
	pathB := newLibraryFile(t, dir, "b.db", 1)
	db := database.NewDatabase()
	a := NewApp(db)
	t.Cleanup(func() { a.closeParkedLibraries(); _ = db.Close() })

	if _, err := a.OpenLibraryGroup(pathA); err != nil {
		t.Fatal(err)
	}
	gB, err := a.OpenLibraryGroup(pathB)
	if err != nil {
		t.Fatal(err)
	}
	stepsA, err := a.ViewerSteps(1, ViewerSource{Kind: "library"})
	if err != nil || len(stepsA) != 3 {
		t.Fatalf("steps of group 1 = %v, %v; want 3", stepsA, err)
	}
	if err := a.ActivateLibraryGroup(1); err != nil {
		t.Fatal(err)
	}
	stepsB, err := a.ViewerSteps(gB.ID, ViewerSource{Kind: "library"})
	if err != nil || len(stepsB) != 1 {
		t.Fatalf("steps of group %d = %v, %v; want 1", gB.ID, stepsB, err)
	}
	item, err := a.ViewerShow(gB.ID, stepsB[0].PositionID)
	if err != nil || item == nil || item.Position.ID != stepsB[0].PositionID {
		t.Fatalf("show = %+v, %v", item, err)
	}
	if again, _ := a.ViewerSteps(1, ViewerSource{Kind: "library"}); len(again) != 3 {
		t.Errorf("the active group reads %d steps, want 3", len(again))
	}
	cat, err := a.ViewerCatalogOf(gB.ID)
	if err != nil || len(cat.Matches) != 0 {
		t.Errorf("catalog = %+v, %v", cat, err)
	}
	if _, err := a.ViewerSteps(42, ViewerSource{Kind: "library"}); err == nil {
		t.Error("a group that does not exist reads")
	}
	if n, _ := db.CountPositions(); n != 3 {
		t.Errorf("the active library holds %d positions, want 3", n)
	}
}
