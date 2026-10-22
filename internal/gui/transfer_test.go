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

// A position carried from one pane onto the other lands in the destination
// alone, and can be taken back; the copy of a position the destination held
// already offers nothing to take back, and taking back leaves the source as
// it was.
func TestCopyPositionToGroup_UndoTakesBackOnlyWhatItAdded(t *testing.T) {
	a, _ := twoGroups(t)
	if s, err := a.ImportFilesIn(1, []string{filepath.Join("..", "..", "testdata", "test.mat")}, "test"); err != nil || s.Succeeded != 1 {
		t.Fatalf("import into A: %+v, %v", s, err)
	}
	ids := everything(t, a, 1).PositionIDs
	before := positionsOf(t, a, 1)

	cp, err := a.CopyPositionToGroup(1, ids[0], 2)
	if err != nil || cp.Receipt == 0 || cp.Positions != 1 {
		t.Fatalf("copy A → B: %+v, %v", cp, err)
	}
	if n := positionsOf(t, a, 2); n != 1 {
		t.Fatalf("B holds %d positions after one copy, want 1", n)
	}
	again, err := a.CopyPositionToGroup(1, ids[0], 2)
	if err != nil || again.Receipt != 0 {
		t.Fatalf("second copy: %+v, %v — want nothing to take back", again, err)
	}
	if err := a.UndoPositionCopy(cp.Receipt); err != nil {
		t.Fatal(err)
	}
	if n := positionsOf(t, a, 2); n != 0 {
		t.Fatalf("B holds %d positions after the undo, want 0", n)
	}
	if n := positionsOf(t, a, 1); n != before {
		t.Fatalf("A holds %d positions after the undo, want %d", n, before)
	}
	if err := a.SetLibraryGroupLocked(2, true); err != nil {
		t.Fatal(err)
	}
	if _, err := a.CopyPositionToGroup(1, ids[0], 2); !errors.Is(err, ErrLibraryGroupLocked) {
		t.Fatalf("copy into a locked group: %v, want ErrLibraryGroupLocked", err)
	}
}

// copiedIntoB imports test.mat into group A and carries its first position
// into group B, returning the copy and the position's id in B.
func copiedIntoB(t *testing.T, a *App) (TransferSummary, int64) {
	t.Helper()
	if s, err := a.ImportFilesIn(1, []string{filepath.Join("..", "..", "testdata", "test.mat")}, "test"); err != nil || s.Succeeded != 1 {
		t.Fatalf("import into A: %+v, %v", s, err)
	}
	cp, err := a.CopyPositionToGroup(1, everything(t, a, 1).PositionIDs[0], 2)
	if err != nil || cp.Receipt == 0 {
		t.Fatalf("copy A → B: %+v, %v", cp, err)
	}
	return cp, everything(t, a, 2).PositionIDs[0]
}

// A copy the destination has written on since — a comment, a collection —
// is not taken back: the delete would carry that work away with it.
func TestUndoPositionCopy_RefusesAPositionChangedSince(t *testing.T) {
	for name, change := range map[string]func(d *database.Database, id int64) error{
		"comment": func(d *database.Database, id int64) error { return d.AddComment(id, "noted after the copy") },
		"collection": func(d *database.Database, id int64) error {
			c, err := d.CreateCollection("kept", "")
			if err != nil {
				return err
			}
			return d.AddPositionToCollection(c, id)
		},
	} {
		t.Run(name, func(t *testing.T) {
			a, _ := twoGroups(t)
			cp, id := copiedIntoB(t, a)
			if err := a.readGroup(2, func(d *database.Database) error { return change(d, id) }); err != nil {
				t.Fatal(err)
			}
			if err := a.UndoPositionCopy(cp.Receipt); !errors.Is(err, database.ErrPositionChanged) {
				t.Fatalf("undo after a change: %v, want ErrPositionChanged", err)
			}
			if n := positionsOf(t, a, 2); n != 1 {
				t.Fatalf("B holds %d positions after the refused undo, want 1", n)
			}
		})
	}
}

// A group that opened another library since the copy names other rows by
// the same id: the copy is not taken back from it.
func TestUndoPositionCopy_RefusesAnotherLibraryInTheGroup(t *testing.T) {
	a, db := twoGroups(t)
	cp, _ := copiedIntoB(t, a)
	if err := a.ActivateLibraryGroup(2); err != nil {
		t.Fatal(err)
	}
	if err := db.OpenDatabase(newLibraryFile(t, t.TempDir(), "c.db", 1)); err != nil {
		t.Fatal(err)
	}
	if err := a.UndoPositionCopy(cp.Receipt); err == nil {
		t.Fatal("undo into a group that opened another library succeeded")
	}
	if n := positionsOf(t, a, 2); n != 1 {
		t.Fatalf("the other library holds %d positions after the refused undo, want 1", n)
	}
	if err := a.UndoPositionCopy(cp.Receipt); err == nil {
		t.Fatal("a receipt was taken back twice")
	}
}

// An undo refused for a reason that passes — the group locked meanwhile —
// keeps its receipt: once the group is unlocked, the same undo succeeds.
func TestUndoPositionCopy_KeepsTheReceiptAfterATransientRefusal(t *testing.T) {
	a, _ := twoGroups(t)
	cp, _ := copiedIntoB(t, a)
	if err := a.SetLibraryGroupLocked(2, true); err != nil {
		t.Fatal(err)
	}
	if err := a.UndoPositionCopy(cp.Receipt); !errors.Is(err, ErrLibraryGroupLocked) {
		t.Fatalf("undo into a locked group: %v, want ErrLibraryGroupLocked", err)
	}
	if err := a.SetLibraryGroupLocked(2, false); err != nil {
		t.Fatal(err)
	}
	if err := a.UndoPositionCopy(cp.Receipt); err != nil {
		t.Fatalf("undo once unlocked: %v", err)
	}
	if n := positionsOf(t, a, 2); n != 0 {
		t.Fatalf("B holds %d positions after the undo, want 0", n)
	}
}

// tournamentInA imports two matches into group A and puts both into one
// tournament, returning its id and the matches' ids.
func tournamentInA(t *testing.T, a *App) (int64, []int64) {
	t.Helper()
	for _, f := range []string{"test.mat", "gnubg_selfplay_drops_7p.mat"} {
		if s, err := a.ImportFilesIn(1, []string{filepath.Join("..", "..", "testdata", f)}, "test"); err != nil || s.Succeeded != 1 {
			t.Fatalf("import %s into A: %+v, %v", f, s, err)
		}
	}
	src := libraryOf(t, a, 1)
	tid, err := src.CreateTournament("Open", "2026-10-22", "Club")
	if err != nil {
		t.Fatal(err)
	}
	matches := everything(t, a, 1).MatchIDs
	if len(matches) != 2 {
		t.Fatalf("A holds %d matches, want 2", len(matches))
	}
	for _, m := range matches {
		if err := src.AddMatchToTournament(tid, m); err != nil {
			t.Fatal(err)
		}
	}
	return tid, matches
}

// A tournament copied leaves with all its matches; a match copied alone
// brings the tournament's record and none of its other matches, even when a
// tournament is named beside it in the same lot.
func TestTransferToGroup_TournamentCarriesItsMatches(t *testing.T) {
	t.Run("tournament", func(t *testing.T) {
		a, _ := twoGroups(t)
		tid, _ := tournamentInA(t, a)
		sum, err := a.TransferToGroup(2, []TransferBatch{{Group: 1, TournamentIDs: []int64{tid}}})
		if err != nil || len(sum.Errors) != 0 || sum.Tournaments != 1 || sum.Matches != 2 {
			t.Fatalf("tournament A → B: %+v, %v", sum, err)
		}
	})
	t.Run("match alone", func(t *testing.T) {
		a, _ := twoGroups(t)
		_, matches := tournamentInA(t, a)
		sum, err := a.TransferToGroup(2, []TransferBatch{{Group: 1, MatchIDs: matches[:1]}})
		if err != nil || len(sum.Errors) != 0 || sum.Tournaments != 1 || sum.Matches != 1 {
			t.Fatalf("match A → B: %+v, %v", sum, err)
		}
	})
	t.Run("match beside another tournament", func(t *testing.T) {
		a, _ := twoGroups(t)
		_, matches := tournamentInA(t, a)
		src := libraryOf(t, a, 1)
		other, err := src.CreateTournament("Empty", "", "")
		if err != nil {
			t.Fatal(err)
		}
		sum, err := a.TransferToGroup(2, []TransferBatch{{Group: 1, MatchIDs: matches[:1], TournamentIDs: []int64{other}}})
		if err != nil || len(sum.Errors) != 0 || sum.Tournaments != 2 || sum.Matches != 1 {
			t.Fatalf("match and tournament A → B: %+v, %v", sum, err)
		}
	})
}

// The session keeps every copy it can take back: they come back newest first
// into one library, and a refusal there does not spend the receipt.
func TestUndoPositionCopy_HistoryTakesBackInReverseOrder(t *testing.T) {
	a, _ := twoGroups(t)
	if s, err := a.ImportFilesIn(1, []string{filepath.Join("..", "..", "testdata", "test.mat")}, "test"); err != nil || s.Succeeded != 1 {
		t.Fatalf("import into A: %+v, %v", s, err)
	}
	ids := everything(t, a, 1).PositionIDs
	first, err := a.CopyPositionToGroup(1, ids[0], 2)
	if err != nil || first.Receipt == 0 {
		t.Fatalf("first copy: %+v, %v", first, err)
	}
	second, err := a.CopyPositionToGroup(1, ids[1], 2)
	if err != nil || second.Receipt == 0 {
		t.Fatalf("second copy: %+v, %v", second, err)
	}
	if err := a.UndoPositionCopy(first.Receipt); !errors.Is(err, ErrLaterCopyFirst) {
		t.Fatalf("undo of the older copy first: %v, want ErrLaterCopyFirst", err)
	}
	if err := a.UndoPositionCopy(second.Receipt); err != nil {
		t.Fatal(err)
	}
	if err := a.UndoPositionCopy(first.Receipt); err != nil {
		t.Fatal(err)
	}
	if n := positionsOf(t, a, 2); n != 0 {
		t.Fatalf("B holds %d positions after both undos, want 0", n)
	}
	if err := a.UndoPositionCopy(first.Receipt); err == nil {
		t.Fatal("a spent receipt was taken back twice")
	}
}
