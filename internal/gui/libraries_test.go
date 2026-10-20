package gui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
)

func newLibraryFile(t *testing.T, dir, name string, positions int) string {
	t.Helper()
	path := filepath.Join(dir, name)
	d := database.NewDatabase()
	if err := d.SetupDatabase(path); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < positions; i++ {
		p := database.Position{Board: database.Board{Points: [26]database.Point{{Checkers: i + 1, Color: 0}}}, Cube: database.Cube{Value: 0, Owner: -1}, PlayerOnRoll: 0, DecisionType: 0}
		if _, err := d.SavePosition(&p); err != nil {
			t.Fatal(err)
		}
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLibraryGroups_OpenSwitchAndClose(t *testing.T) {
	dir := t.TempDir()
	pathA := newLibraryFile(t, dir, "a.db", 2)
	pathB := newLibraryFile(t, dir, "b.db", 0)
	db := database.NewDatabase()
	a := NewApp(db)
	t.Cleanup(func() { a.closeParkedLibraries(); _ = db.Close() })

	// With nothing open, the active group takes the file.
	g, err := a.OpenLibraryGroup(pathA)
	if err != nil || g.ID != 1 {
		t.Fatalf("first open: %+v, %v", g, err)
	}
	g, err = a.OpenLibraryGroup(pathB)
	if err != nil || g.ID != 2 {
		t.Fatalf("second open: %+v, %v", g, err)
	}
	if n, _ := db.CountPositions(); n != 0 {
		t.Fatalf("active library has %d positions, want B's 0", n)
	}
	if groups := a.ListLibraryGroups(); len(groups) != 2 || !groups[1].Active || groups[0].Path != pathA {
		t.Fatalf("groups: %+v", groups)
	}

	if err := a.ActivateLibraryGroup(1); err != nil {
		t.Fatal(err)
	}
	if n, _ := db.CountPositions(); n != 2 {
		t.Fatalf("after switching back the bound handle reads %d positions, want A's 2", n)
	}

	// The same file, however spelled, is never opened twice.
	link := filepath.Join(dir, "link.db")
	if err := os.Symlink(pathB, link); err != nil {
		t.Fatal(err)
	}
	var already *ErrLibraryAlreadyOpen
	if _, err := a.OpenLibraryGroup(link); !errors.As(err, &already) || already.Group != 2 {
		t.Fatalf("opening B again: %v, want ErrLibraryAlreadyOpen{2}", err)
	}
	if got := a.LibraryGroupHolding(filepath.Join(dir, ".", "a.db")); got != 1 {
		t.Errorf("LibraryGroupHolding(a) = %d, want 1", got)
	}

	if err := a.CloseLibraryGroup(1); err == nil {
		t.Error("closing the active group while another waits must be refused")
	}
	if err := a.CloseLibraryGroup(2); err != nil {
		t.Fatal(err)
	}
	if groups := a.ListLibraryGroups(); len(groups) != 1 {
		t.Fatalf("after closing B: %+v", groups)
	}
	if a.LibraryGroupHolding(pathB) != 0 {
		t.Error("a closed group still holds its file")
	}
}

func TestLibraryGroups_SwitchRefusedWhileTheFolderWatchRuns(t *testing.T) {
	dir := t.TempDir()
	db := database.NewDatabase()
	a := NewApp(db)
	t.Cleanup(func() { a.StopFolderWatch(); a.closeParkedLibraries(); _ = db.Close() })
	if _, err := a.OpenLibraryGroup(newLibraryFile(t, dir, "a.db", 0)); err != nil {
		t.Fatal(err)
	}
	if _, err := a.OpenLibraryGroup(newLibraryFile(t, dir, "b.db", 0)); err != nil {
		t.Fatal(err)
	}
	if _, err := a.StartFolderWatch(t.TempDir(), 3600); err != nil {
		t.Skipf("folder watch unavailable: %v", err)
	}
	var busy *database.BusyError
	if err := a.ActivateLibraryGroup(1); !errors.As(err, &busy) {
		t.Fatalf("switch during a folder watch: %v, want *BusyError", err)
	}
}
