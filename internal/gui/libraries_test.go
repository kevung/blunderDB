package gui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/rollout"
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

// twoGroups opens a.db (with posA positions) in group 1 and b.db (empty) in
// group 2, and brings group 1 back on screen.
func twoGroups(t *testing.T) (*App, *database.Database) {
	t.Helper()
	dir := t.TempDir()
	db := database.NewDatabase()
	a := NewApp(db)
	t.Cleanup(func() { a.StopFolderWatch(); a.stopDatabaseJobs(time.Second); a.closeParkedLibraries(); _ = db.Close() })
	if _, err := a.OpenLibraryGroup(newLibraryFile(t, dir, "a.db", 1)); err != nil {
		t.Fatal(err)
	}
	if _, err := a.OpenLibraryGroup(newLibraryFile(t, dir, "b.db", 0)); err != nil {
		t.Fatal(err)
	}
	if err := a.ActivateLibraryGroup(1); err != nil {
		t.Fatal(err)
	}
	return a, db
}

// positionsOf counts the positions of group's library, on screen or not.
func positionsOf(t *testing.T, a *App, group int) int {
	t.Helper()
	_, lent, err := a.borrowLibrary(group)
	if err != nil {
		t.Fatal(err)
	}
	defer lent.Close()
	n, err := lent.CountPositions()
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// An import started in A keeps writing A when B comes on screen, and writes
// nothing into B. A write transaction held on A keeps the import from
// committing until after the switch, so the switch lands in its middle.
func TestLibraryGroups_ImportFinishesInItsGroupAcrossASwitch(t *testing.T) {
	a, _ := twoGroups(t)
	if _, err := a.db.BeginImportBatch("test.mat", "mixed"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	blocker, err := database.RawConn(a.db).Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Close()
	if _, err := blocker.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		t.Fatal(err)
	}
	type result struct {
		s   ImportFilesSummary
		err error
	}
	done := make(chan result, 1)
	go func() {
		s, err := a.ImportFiles([]string{filepath.Join("..", "..", "testdata", "test.mat")}, true)
		done <- result{s, err}
	}()
	waitFor(t, func() bool { return a.importsRunning() > 0 })

	if err := a.ActivateLibraryGroup(2); err != nil {
		t.Fatalf("switch during an import: %v", err)
	}
	select {
	case r := <-done:
		t.Fatalf("the import finished before the switch could land in it: %+v, %v", r.s, r.err)
	default:
	}
	if _, err := blocker.ExecContext(ctx, "ROLLBACK"); err != nil {
		t.Fatal(err)
	}
	r := <-done
	if r.err != nil || r.s.Succeeded != 1 || r.s.Group != 1 {
		t.Fatalf("import: %+v, %v; want one match into group 1", r.s, r.err)
	}
	if r.s.Report == nil {
		t.Error("the background import must close its batch and return the report")
	}
	if n, _ := a.db.CountPositions(); n != 0 {
		t.Errorf("B, on screen, has %d positions: the import wrote into it", n)
	}
	if n := positionsOf(t, a, 1); n <= 1 {
		t.Errorf("A has %d positions after the import, want more than its 1", n)
	}
	if database.OpenImportBatch(a.db) != 0 {
		t.Error("the batch was left open on the bound handle")
	}
}

// A batch analysis started in A writes A after the switch to B.
func TestLibraryGroups_BatchKeepsItsGroupAcrossASwitch(t *testing.T) {
	a, _ := twoGroups(t)
	release := make(chan struct{})
	wrote := make(chan error, 1)
	a.runGammonNetBatch(0, 2, false, func(ctx context.Context, db *database.Database, _ func(done, total int)) (database.GammonNetBatchSummary, error) {
		<-release
		p := database.Position{Board: database.Board{Points: [26]database.Point{{Checkers: 9, Color: 0}}}, Cube: database.Cube{Value: 0, Owner: -1}}
		_, err := db.SavePosition(&p)
		wrote <- err
		return database.GammonNetBatchSummary{}, err
	})
	a.gnBatchMu.Lock()
	group := a.gnBatchGroup
	a.gnBatchMu.Unlock()
	if group != 1 {
		t.Fatalf("batch group = %d, want 1", group)
	}
	if err := a.ActivateLibraryGroup(2); err != nil {
		t.Fatalf("switch during a batch: %v", err)
	}
	close(release)
	if err := <-wrote; err != nil {
		t.Fatal(err)
	}
	if n, _ := a.db.CountPositions(); n != 0 {
		t.Errorf("B, on screen, has %d positions: the batch wrote into it", n)
	}
	if n := positionsOf(t, a, 1); n != 2 {
		t.Errorf("A has %d positions, want 2", n)
	}
}

// The watched folder belongs to the group it was started in: the switch goes
// through, its files go to that group, and closing the group stops it.
func TestLibraryGroups_WatchKeepsItsGroup(t *testing.T) {
	a, _ := twoGroups(t)
	if _, err := a.StartFolderWatch(t.TempDir(), 3600); err != nil {
		t.Skipf("folder watch unavailable: %v", err)
	}
	if err := a.ActivateLibraryGroup(2); err != nil {
		t.Fatalf("switch during a folder watch: %v", err)
	}
	if st := a.FolderWatchStatus(); !st.Running || st.Group != 1 {
		t.Fatalf("watch after the switch: %+v, want running for group 1", st)
	}
	s, err := a.ImportFilesIn(1, []string{filepath.Join("..", "..", "testdata", "test.mat")}, "watched")
	if err != nil || s.Succeeded != 1 || s.Group != 1 {
		t.Fatalf("watched import into group 1: %+v, %v", s, err)
	}
	if n, _ := a.db.CountPositions(); n != 0 {
		t.Errorf("B, on screen, has %d positions: the watched import wrote into it", n)
	}
	if n := positionsOf(t, a, 1); n <= 1 {
		t.Errorf("A has %d positions, want the watched match in it", n)
	}
	if err := a.CloseLibraryGroup(1); err != nil {
		t.Fatal(err)
	}
	if a.FolderWatchStatus().Running {
		t.Error("closing the group left its watch running")
	}
}

// waitFor polls cond for up to two seconds.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met within two seconds")
		}
		time.Sleep(time.Millisecond)
	}
}

// Opening a file through the bound handle itself — not through a group — is
// refused when another group holds it: the file would sit in two groups.
func TestLibraryGroups_BoundOpenRefusesAFileAnotherGroupHolds(t *testing.T) {
	dir := t.TempDir()
	pathA := newLibraryFile(t, dir, "a.db", 1)
	pathB := newLibraryFile(t, dir, "b.db", 0)
	db := database.NewDatabase()
	a := NewApp(db)
	t.Cleanup(func() { a.closeParkedLibraries(); _ = db.Close() })

	first, err := a.OpenLibraryGroup(pathA)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.OpenLibraryGroup(pathB); err != nil {
		t.Fatal(err)
	}
	var held *ErrLibraryAlreadyOpen
	if err := db.OpenDatabase(pathA); !errors.As(err, &held) || held.Group != first.ID {
		t.Fatalf("OpenDatabase of a parked library = %v, want ErrLibraryAlreadyOpen{%d}", err, first.ID)
	}
	if got := database.OpenPath(db); got != pathB {
		t.Errorf("active library after the refusal = %q, want %q", got, pathB)
	}
	if err := db.OpenDatabase(pathB); err != nil {
		t.Errorf("reopening the active library: %v", err)
	}
}

// Ids read in one library name other positions in another: a rollout of a list
// read in A is refused once B is on screen, rather than run and stored in B.
func TestLibraryGroups_RolloutOfAListReadInAnotherGroupIsRefused(t *testing.T) {
	a, _ := twoGroups(t)
	if err := a.ActivateLibraryGroup(2); err != nil {
		t.Fatal(err)
	}
	if _, err := a.StartRolloutIDs([]int64{1}, rollout.Fast(), 1); !errors.Is(err, ErrListOfAnotherLibrary) {
		t.Fatalf("StartRolloutIDs of a list read in group 1, group 2 on screen: err = %v, want ErrListOfAnotherLibrary", err)
	}
	if st := a.RolloutStatus(); st.Running {
		t.Errorf("a rollout started: %+v", st)
	}
}
