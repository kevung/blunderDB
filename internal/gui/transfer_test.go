package gui

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
)

// libraryOf lends group's library to the test, on screen or not.
func libraryOf(t *testing.T, a *App, group int) *database.Database {
	t.Helper()
	_, lent, err := a.borrowLibrary(group)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lent.Close() })
	return lent
}

// everything names every position, match and collection of group's library.
func everything(t *testing.T, a *App, group int) TransferBatch {
	t.Helper()
	d := libraryOf(t, a, group)
	b := TransferBatch{Group: group}
	positions, err := d.LoadAllPositions()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range positions {
		b.PositionIDs = append(b.PositionIDs, p.ID)
	}
	matches, err := d.GetAllMatches()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range matches {
		b.MatchIDs = append(b.MatchIDs, m.ID)
	}
	collections, err := d.GetAllCollections()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range collections {
		b.CollectionIDs = append(b.CollectionIDs, c.ID)
	}
	return b
}

type libraryCounts struct{ positions, matches, collections int }

func countsOf(t *testing.T, a *App, group int) libraryCounts {
	t.Helper()
	b := everything(t, a, group)
	return libraryCounts{len(b.PositionIDs), len(b.MatchIDs), len(b.CollectionIDs)}
}

// A → B, then B → A, then A → B again: each library ends with the union, and
// no round trip adds a row a library already held. The transfer writes the
// group off screen and leaves the one on screen as it was.
func TestTransferToGroup_RoundTripCreatesNoDuplicate(t *testing.T) {
	a, _ := twoGroups(t)
	if s, err := a.ImportFilesIn(1, []string{filepath.Join("..", "..", "testdata", "test.mat")}, "test"); err != nil || s.Succeeded != 1 {
		t.Fatalf("import into A: %+v, %v", s, err)
	}
	src := libraryOf(t, a, 1)
	coll, err := src.CreateCollection("club", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := src.AddPositionToCollection(coll, everything(t, a, 1).PositionIDs[0]); err != nil {
		t.Fatal(err)
	}
	before := countsOf(t, a, 1)

	sum, err := a.TransferToGroup(2, []TransferBatch{everything(t, a, 1)})
	if err != nil || len(sum.Errors) != 0 || sum.Group != 2 {
		t.Fatalf("A → B: %+v, %v", sum, err)
	}
	if got := countsOf(t, a, 2); got != before {
		t.Fatalf("B after A → B holds %+v, want A's %+v", got, before)
	}
	if got := countsOf(t, a, 1); got != before {
		t.Fatalf("A, on screen, changed: %+v, want %+v", got, before)
	}

	// B on screen, the transfer back reads the group on screen and writes A off screen.
	if err := a.ActivateLibraryGroup(2); err != nil {
		t.Fatal(err)
	}
	sum, err = a.TransferToGroup(1, []TransferBatch{everything(t, a, 2)})
	if err != nil || len(sum.Errors) != 0 {
		t.Fatalf("B → A: %+v, %v", sum, err)
	}
	if sum.Positions != 0 || sum.Matches != 0 {
		t.Errorf("B → A added %d positions and %d matches A already held", sum.Positions, sum.Matches)
	}
	if got := countsOf(t, a, 1); got != before {
		t.Fatalf("A after B → A holds %+v, want %+v", got, before)
	}
	if _, err := a.TransferToGroup(2, []TransferBatch{everything(t, a, 1)}); err != nil {
		t.Fatal(err)
	}
	if got := countsOf(t, a, 2); got != before {
		t.Fatalf("B after a second A → B holds %+v, want %+v", got, before)
	}
}

// A locked group takes nothing, and says so; unlocked, it takes the batch. A
// batch of the destination itself is skipped.
func TestTransferToGroup_LockRefuses(t *testing.T) {
	a, _ := twoGroups(t)
	if err := a.SetLibraryGroupLocked(2, true); err != nil {
		t.Fatal(err)
	}
	if g := a.ListLibraryGroups(); !g[1].Locked || g[0].Locked {
		t.Fatalf("groups: %+v, want group 2 locked", g)
	}
	_, err := a.TransferToGroup(2, []TransferBatch{everything(t, a, 1)})
	if !errors.Is(err, ErrLibraryGroupLocked) {
		t.Fatalf("transfer into a locked group: %v, want ErrLibraryGroupLocked", err)
	}
	if n := positionsOf(t, a, 2); n != 0 {
		t.Fatalf("the locked group took %d positions", n)
	}
	if err := a.SetLibraryGroupLocked(2, false); err != nil {
		t.Fatal(err)
	}
	if _, err := a.TransferToGroup(2, []TransferBatch{everything(t, a, 1), everything(t, a, 2)}); err != nil {
		t.Fatal(err)
	}
	if n := positionsOf(t, a, 2); n != 1 {
		t.Fatalf("B holds %d positions, want A's one", n)
	}
	if err := a.SetLibraryGroupLocked(9, true); err == nil {
		t.Error("locking a group that does not exist succeeded")
	}
}
