package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"
)

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
	return gui, cli
}

func writeMeta(t *testing.T, db *sql.DB, key string) {
	t.Helper()
	if _, err := db.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, 'x')`, key); err != nil {
		t.Fatal(err)
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
		t.Fatal("first call takes the mark, reports nothing")
	}
	if elsewhere(t, gui) {
		t.Fatal("nothing written: nothing reported")
	}

	writeMeta(t, cli.sqlDB, "from-cli")
	if !elsewhere(t, gui) {
		t.Fatal("a commit of the other handle must be reported")
	}
	if elsewhere(t, gui) {
		t.Fatal("reported once only")
	}

	// The handle's own writes, several at once so that they land on several
	// connections of its pool, in transactions and in autocommit.
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if i%2 == 0 {
				writeMeta(t, gui.sqlDB, "own-"+string(rune('a'+i)))
				return
			}
			tx, err := gui.sqlDB.Begin()
			if err != nil {
				t.Error(err)
				return
			}
			if _, err := tx.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, 'tx')`, "own-tx-"+string(rune('a'+i))); err != nil {
				t.Error(err)
			}
			if err := tx.Commit(); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if elsewhere(t, gui) {
		t.Fatal("the handle's own writes must not be reported")
	}

	// The other side sees the first handle's writes as outside ones.
	if elsewhere(t, cli) {
		t.Fatal("first call of the second handle takes the mark")
	}
	writeMeta(t, gui.sqlDB, "own-again")
	if elsewhere(t, gui) {
		t.Fatal("own write reported")
	}
	if !elsewhere(t, cli) {
		t.Fatal("the second handle must see the first's write")
	}
}

// data_version on a connection of the pool moves at the commits of the other
// connections of the same pool: it alone cannot tell the handle's own writes
// from another process's, hence the commit count.
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
	writeMeta(t, gui.sqlDB, "own") // another pool connection: pinned is held
	if read() == before {
		t.Fatal("expected data_version to move at a commit of another connection of the same pool")
	}
}

func TestWrittenElsewhere_ClosedAndMemory(t *testing.T) {
	gui, _ := openPair(t)
	_ = elsewhere(t, gui)
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
	writeMeta(t, mem.sqlDB, "k")
	if first, second := elsewhere(t, mem), elsewhere(t, mem); first || second {
		t.Fatal("an in-memory database has no other writer")
	}
}
