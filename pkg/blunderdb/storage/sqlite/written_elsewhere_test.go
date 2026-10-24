package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

// openPair opens one library twice: gui stands for the desktop, cli for
// another process. The first call of gui is made, so that the creation of the
// file (its switch to WAL, seen by the watch) is behind.
func openPair(t *testing.T) (gui, cli *Storage) {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "lib.db")
	gui, err := Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = gui.Close() })
	cli, err = Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cli.Close() })
	_, _ = gui.WrittenElsewhere(ctx)
	_, _ = cli.WrittenElsewhere(ctx)
	return gui, cli
}

var keys atomic.Int64

func writeMeta(t *testing.T, db *sql.DB) {
	t.Helper()
	key := fmt.Sprintf("k%d", keys.Add(1))
	if _, err := db.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, 'x')`, key); err != nil {
		t.Error(err)
	}
}

func elsewhere(t *testing.T, s *Storage) bool {
	t.Helper()
	got, err := s.WrittenElsewhere(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestWrittenElsewhere_SeesAnotherHandleNotItsOwn(t *testing.T) {
	gui, cli := openPair(t)
	if elsewhere(t, gui) {
		t.Fatal("nothing written: nothing reported")
	}
	writeMeta(t, cli.sqlDB)
	if !elsewhere(t, gui) {
		t.Fatal("a commit of the other handle must be reported")
	}
	if elsewhere(t, gui) {
		t.Fatal("reported once only")
	}

	// The handle's own writes, one after the other, in autocommit and in
	// transactions: none is reported.
	for i := range 6 {
		if i%2 == 0 {
			writeMeta(t, gui.sqlDB)
			continue
		}
		tx, err := gui.sqlDB.Begin()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('tx', ?)`, i); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
	if elsewhere(t, gui) {
		t.Fatal("the handle's own writes must not be reported")
	}
	if !elsewhere(t, cli) {
		t.Fatal("the second handle must see the first's writes")
	}
}

// A transaction that writes nothing still takes the write lock, and the
// commit hook still runs: it must not hide a commit of another process.
func TestWrittenElsewhere_EmptyOwnCommitHidesNothing(t *testing.T) {
	gui, cli := openPair(t)
	ctx := context.Background()

	tx, err := gui.BeginTx(ctx) // takes the write lock, writes nothing
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	writeMeta(t, cli.sqlDB)
	if !elsewhere(t, gui) {
		t.Fatal("an empty transaction of the handle hid the other's commit")
	}

	writeMeta(t, cli.sqlDB)
	if _, err := gui.sqlDB.Exec(`UPDATE metadata SET value = 'y' WHERE key = 'none'`); err != nil {
		t.Fatal(err)
	}
	if !elsewhere(t, gui) {
		t.Fatal("an update touching no row hid the other's commit")
	}
}

// Writes of the handle interleaved with another process's: the outside one
// is never lost, whichever comes first.
func TestWrittenElsewhere_InterleavedWritesLoseNothing(t *testing.T) {
	gui, cli := openPair(t)

	writeMeta(t, cli.sqlDB)
	writeMeta(t, gui.sqlDB)
	if !elsewhere(t, gui) {
		t.Fatal("outside commit then own commit: the outside one was lost")
	}

	writeMeta(t, gui.sqlDB)
	writeMeta(t, cli.sqlDB)
	if !elsewhere(t, gui) {
		t.Fatal("own commit then outside commit: the outside one was lost")
	}

	// The outside commit lands after the handle's commit and before its
	// connection is back in the pool: the window a mark taken after the
	// commit would cover.
	ctx := context.Background()
	pinned, err := gui.sqlDB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pinned.ExecContext(ctx, `INSERT OR REPLACE INTO metadata (key, value) VALUES ('pinned', 'x')`); err != nil {
		t.Fatal(err)
	}
	writeMeta(t, cli.sqlDB)
	_ = elsewhere(t, gui) // while unsettled: may wait, must not lose
	if err := pinned.Close(); err != nil {
		t.Fatal(err)
	}
	if !elsewhere(t, gui) {
		t.Fatal("an outside commit right after the handle's was lost")
	}

	// Both at once, many times: every round with an outside commit is seen.
	for round := range 40 {
		var wg sync.WaitGroup
		for range 3 {
			wg.Go(func() { writeMeta(t, gui.sqlDB) })
		}
		wg.Go(func() { writeMeta(t, cli.sqlDB) })
		wg.Wait()
		if !elsewhere(t, gui) {
			t.Fatalf("round %d: the outside commit among the handle's was lost", round)
		}
	}
}

// The mark is taken at open: what another process writes before the first
// call is reported by it.
func TestWrittenElsewhere_MarkTakenAtOpen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "lib.db")
	first, err := Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = first.Close()

	gui, err := Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer gui.Close()
	cli, err := Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()
	writeMeta(t, cli.sqlDB)
	if !elsewhere(t, gui) {
		t.Fatal("a write before the first call was lost")
	}
	if elsewhere(t, gui) {
		t.Fatal("reported once only")
	}
}

// A call that fails loses the mark: the next one that succeeds reports a
// change, since one may have come meanwhile.
func TestWrittenElsewhere_AfterAnErrorReportsAChange(t *testing.T) {
	gui, cli := openPair(t)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	w := watchOf(gui.sqlDB)
	w.mu.Lock()
	w.dropLocked() // the next read connects again, with the canceled context
	w.mu.Unlock()
	if _, err := gui.WrittenElsewhere(canceled); err == nil {
		t.Skip("the driver ignored the canceled context: no error to recover from")
	}
	writeMeta(t, cli.sqlDB)
	if !elsewhere(t, gui) {
		t.Fatal("after an error, the first call that succeeds must report a change")
	}
	if elsewhere(t, gui) {
		t.Fatal("then nothing more")
	}
}

// data_version on a connection of the pool moves at the commits of the other
// connections of the same pool: it alone cannot tell the handle's own writes
// from another process's.
func TestDataVersion_MovesAtOwnPoolCommits(t *testing.T) {
	gui, _ := openPair(t)
	ctx := context.Background()
	pinned, err := gui.sqlDB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer pinned.Close()
	read := func() int64 {
		var v int64
		if err := pinned.QueryRowContext(ctx, `PRAGMA data_version`).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	before := read()
	writeMeta(t, gui.sqlDB) // another pool connection: pinned is held
	if read() == before {
		t.Fatal("expected data_version to move at a commit of another connection of the same pool")
	}
}

func TestWrittenElsewhere_ClosedAndMemory(t *testing.T) {
	gui, _ := openPair(t)
	if err := gui.Close(); err != nil {
		t.Fatal(err)
	}
	if elsewhere(t, gui) {
		t.Fatal("a closed handle reports nothing")
	}
	mem, err := Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer mem.Close()
	writeMeta(t, mem.sqlDB)
	if first, second := elsewhere(t, mem), elsewhere(t, mem); first || second {
		t.Fatal("an in-memory database has no other writer")
	}
}
