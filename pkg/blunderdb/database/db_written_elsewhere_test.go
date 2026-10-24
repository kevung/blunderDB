package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func writtenElsewhere(t *testing.T, d *Database) bool {
	t.Helper()
	got, err := WrittenElsewhere(d)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func putMeta(t *testing.T, db *sql.DB, key string) {
	t.Helper()
	if _, err := db.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, 'x')`, key); err != nil {
		t.Fatal(err)
	}
}

// The desktop's handle sees a serve daemon's (or the CLI's) commits to its
// file, never its own; a second desktop opened read-only on the same file sees
// the writer's.
func TestWrittenElsewhere_DesktopAndAnotherProcess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lib.db")
	gui := NewDatabase()
	if err := gui.SetupDatabase(path); err != nil {
		t.Fatal(err)
	}
	defer gui.Close()
	serve, err := sqlite.Open(context.Background(), path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serve.Close()
	reader := NewDatabase()
	if err := reader.OpenDatabase(path); err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if !reader.IsReadOnly() {
		t.Fatal("the second desktop handle should fall back to read-only")
	}

	// The creation of the file (its switch to WAL, the schema) may be
	// reported once: one read too many.
	for _, d := range []*Database{gui, reader} {
		_ = writtenElsewhere(t, d)
		if writtenElsewhere(t, d) {
			t.Fatal("nothing written since: nothing reported")
		}
	}

	if err := serve.Metadata().Save(context.Background(), "", map[string]string{"from-serve": "x"}); err != nil {
		t.Fatal(err)
	}
	if !writtenElsewhere(t, gui) || !writtenElsewhere(t, reader) {
		t.Fatal("both desktop handles must see the daemon's commit")
	}

	putMeta(t, RawConn(gui), "own")
	if writtenElsewhere(t, gui) {
		t.Fatal("the desktop's own write must not be reported")
	}
	if !writtenElsewhere(t, reader) {
		t.Fatal("the read-only handle must see the writer's commit")
	}

	if err := gui.Close(); err != nil {
		t.Fatal(err)
	}
	if writtenElsewhere(t, gui) {
		t.Fatal("a closed handle reports nothing")
	}
}

// A library held for writing (a compaction, the end of an import) is not
// waited for: the call answers at once, and the outside write is reported
// once the library is let go.
func TestWrittenElsewhere_DoesNotWaitForAWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lib.db")
	gui := NewDatabase()
	if err := gui.SetupDatabase(path); err != nil {
		t.Fatal(err)
	}
	defer gui.Close()
	serve, err := sqlite.Open(context.Background(), path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serve.Close()
	_ = writtenElsewhere(t, gui)

	if err := serve.Metadata().Save(context.Background(), "", map[string]string{"from-serve": "x"}); err != nil {
		t.Fatal(err)
	}
	gui.mu.Lock()
	answered := make(chan bool, 1)
	go func() {
		got, _ := WrittenElsewhere(gui)
		answered <- got
	}()
	select {
	case got := <-answered:
		if got {
			t.Error("a library held for writing is skipped, not read")
		}
	case <-time.After(5 * time.Second):
		t.Error("WrittenElsewhere waited for the writer")
	}
	gui.mu.Unlock()
	if !writtenElsewhere(t, gui) {
		t.Fatal("the outside write must be reported once the library is let go")
	}
}
