package database

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestDB creates a file-backed database in t.TempDir with the current schema.
// Cleanup is registered automatically via t.Cleanup.
func newTestDB(t *testing.T) *Database {
	t.Helper()
	dbPath := filepath.Join(tempDir(t), "test.db")
	db := NewDatabase()
	if err := db.SetupDatabase(dbPath); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	closeOnCleanup(t, db)
	return db
}

// closeOnCleanup registers db.Close (idempotent — see Database.Close) via
// tb.Cleanup so the underlying SQLite handle is released before t.TempDir()
// tries to remove the directory it lives in. On Linux a still-open handle is
// unlinked silently; on Windows the removal itself fails ("The process cannot
// access the file because it is being used by another process"), which is
// where this class of bug was first noticed (windows-latest CI).
func closeOnCleanup(tb testing.TB, db *Database) {
	tb.Helper()
	tb.Cleanup(func() {
		if err := db.Close(); err != nil {
			tb.Logf("Close: %v", err)
		}
	})
}

// tempDir behaves like tb.TempDir, but also registers a tb.Cleanup guard
// (assertNoLeakedTempFiles) that fails the test if, once every other cleanup
// has run, a file under the directory is still open. It must be called before
// any closeOnCleanup for the same test so the guard — registered first —
// runs last (t.Cleanup is LIFO): otherwise it would see the handle still open
// and fail on the very Close it is meant to wait for.
func tempDir(tb testing.TB) string {
	tb.Helper()
	dir := tb.TempDir()
	assertNoLeakedTempFiles(tb, dir)
	return dir
}

// assertNoLeakedTempFiles registers the leak check tempDir relies on. Windows
// reports a leaked SQLite handle loudly, as a TempDir cleanup failure; Linux
// normally unlinks the file out from under the open descriptor without
// complaint, hiding the same bug. /proc/self/fd makes it visible here too: a
// symlink under it resolving inside dir after the test's own cleanups have
// run means something never called Close. Best-effort — silently a no-op
// where /proc is unavailable (non-Linux).
func assertNoLeakedTempFiles(tb testing.TB, dir string) {
	tb.Helper()
	tb.Cleanup(func() {
		for _, f := range leakedFilesUnder(dir) {
			tb.Errorf("leaked open file descriptor for %s (a Database/*sql.DB was not Close()d before the temp dir cleanup)", f)
		}
	})
}

// leakedFilesUnder lists the files under dir that this process still holds
// open, by resolving every /proc/self/fd symlink. Returns nil (no error) on
// any platform or sandbox where /proc/self/fd cannot be read.
func leakedFilesUnder(dir string) []string {
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		return nil
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil
	}
	prefix := absDir + string(filepath.Separator)
	var leaked []string
	for _, e := range entries {
		target, err := os.Readlink(filepath.Join("/proc/self/fd", e.Name()))
		if err != nil {
			continue
		}
		if strings.HasPrefix(target, prefix) {
			leaked = append(leaked, target)
		}
	}
	return leaked
}

// newTestDBWithXG creates a file-backed database and imports testdata/test.xg.
func newTestDBWithXG(t *testing.T) *Database {
	t.Helper()
	db := newTestDB(t)
	if _, err := db.ImportXGMatch(filepath.Join("testdata", "test.xg")); err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}
	return db
}

// importTestMatch imports testdata/test.sgf and returns the match ID.
func importTestMatch(t *testing.T, db *Database) int64 {
	t.Helper()
	matchID, err := db.ImportGnuBGMatch(filepath.Join("testdata", "test.sgf"))
	if err != nil {
		t.Fatalf("ImportGnuBGMatch: %v", err)
	}
	return matchID
}

// getPositionIDs returns position IDs from the database (up to limit).
func getPositionIDs(t *testing.T, db *Database, limit int) []int64 {
	t.Helper()
	rows, err := db.db.Query(`SELECT id FROM position ORDER BY id LIMIT ?`, limit)
	if err != nil {
		t.Fatalf("query positions: %v", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(ids) == 0 {
		t.Fatal("no positions in database")
	}
	return ids
}

// initialBoard returns the standard starting backgammon position.
// Board indices 0=WhiteBar, 1..24=board points, 25=BlackBar.
func initialBoard() Board {
	var b Board
	// Black checkers
	b.Points[24] = Point{Checkers: 2, Color: Black}
	b.Points[13] = Point{Checkers: 5, Color: Black}
	b.Points[8] = Point{Checkers: 3, Color: Black}
	b.Points[6] = Point{Checkers: 5, Color: Black}
	// White checkers
	b.Points[1] = Point{Checkers: 2, Color: White}
	b.Points[12] = Point{Checkers: 5, Color: White}
	b.Points[17] = Point{Checkers: 3, Color: White}
	b.Points[19] = Point{Checkers: 5, Color: White}
	return b
}

// initialPosition returns the standard starting position (checker decision).
func initialPosition() Position {
	return Position{
		Board:        initialBoard(),
		Cube:         Cube{Owner: None, Value: 0}, // 0 = exponent for cube-at-1
		PlayerOnRoll: 0,
		DecisionType: CheckerAction,
	}
}

// bearoffPosition returns a pure race where both players have only their
// home-board checkers left.
func bearoffPosition() Position {
	var b Board
	// Black racing home (points 1-6)
	b.Points[1] = Point{Checkers: 3, Color: Black}
	b.Points[2] = Point{Checkers: 3, Color: Black}
	b.Points[3] = Point{Checkers: 3, Color: Black}
	b.Points[4] = Point{Checkers: 3, Color: Black}
	b.Points[5] = Point{Checkers: 3, Color: Black}
	// White racing home (points 19-24)
	b.Points[20] = Point{Checkers: 3, Color: White}
	b.Points[21] = Point{Checkers: 3, Color: White}
	b.Points[22] = Point{Checkers: 3, Color: White}
	b.Points[23] = Point{Checkers: 3, Color: White}
	b.Points[24] = Point{Checkers: 3, Color: White}
	return Position{Board: b, Cube: Cube{Owner: None, Value: 0}, PlayerOnRoll: 0, DecisionType: CheckerAction}
}

// cubePosition creates a cube-decision position. cubeExp is the cube exponent:
// 0 = cube at 1 (initial), 1 = cube at 2, 2 = cube at 4, …
func cubePosition(cubeExp int, cubeOwner int) Position {
	return Position{
		Board:        initialBoard(),
		Cube:         Cube{Owner: cubeOwner, Value: cubeExp},
		PlayerOnRoll: 0,
		DecisionType: CubeAction,
	}
}

// loadPositionByIDUnlocked reads one position through the Storage backend
// without taking d.mu. It was the Anki family's private loader until that
// family became an adapter over storage/sqlite (which has its own); the tests
// that reach a position by id keep using it.
func (d *Database) loadPositionByIDUnlocked(positionID int64) (Position, error) {
	pos, err := d.store.Positions().Load(context.Background(), "", positionID)
	if err != nil {
		return Position{}, err
	}
	return *pos, nil
}
