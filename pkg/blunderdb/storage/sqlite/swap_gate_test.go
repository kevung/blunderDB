package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite/filelock"
)

// TestOpen_WaitsAtTheSwapGate: the serve daemon and migrate open a library
// through Open; one that reaches the file while a vacuum replaces it waits
// for the rename and connects to the new file, never to the inode the rename
// unlinks, where whatever it wrote would be lost.
func TestOpen_WaitsAtTheSwapGate(t *testing.T) {
	spellings := map[string]func(path string) string{
		"path":        func(p string) string { return p },
		"file URI":    func(p string) string { return "file:" + uriPath(p) + "?mode=rwc" },
		"file:// URI": func(p string) string { return "file://" + uriPath(p) },
	}
	for name, spell := range spellings {
		t.Run(name, func(t *testing.T) { testOpenWaitsAtTheSwapGate(t, spell) })
	}
}

// uriPath spells path as the path of a file URI: forward slashes, and a
// leading one before a Windows drive.
func uriPath(path string) string {
	p := filepath.ToSlash(path)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

func testOpenWaitsAtTheSwapGate(t *testing.T, spell func(string) string) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "lib.db")
	replacement := filepath.Join(dir, "lib.db.vacuum")
	for _, p := range []string{path, replacement} {
		st, err := Open(ctx, p, nil)
		if err != nil {
			t.Fatal(err)
		}
		if p == replacement {
			if _, err := st.sqlDB.Exec(`INSERT INTO metadata (key, value) VALUES ('swap_probe', '1')`); err != nil {
				t.Fatal(err)
			}
		}
		if err := st.Close(); err != nil {
			t.Fatal(err)
		}
	}

	dsn := spell(path)
	if got := dsnFilePath(dsn); got != path {
		t.Fatalf("dsnFilePath(%q) = %q, want %q", dsn, got, path)
	}
	gate, ok, err := filelock.TryExclusive(filelock.SwapGatePath(path))
	if err != nil || !ok {
		t.Fatalf("taking the swap gate: %v, %v", ok, err)
	}
	waited, stop := filelock.ObserveWaits(filelock.SwapGatePath(path))
	defer stop()

	type result struct {
		st  *Storage
		err error
	}
	opened := make(chan result, 1)
	go func() {
		st, err := Open(ctx, dsn, nil)
		opened <- result{st, err}
	}()
	// The rename waits until the opener is seen blocked at the gate, or has
	// finished without it: no timing decides.
	select {
	case <-waited:
	case r := <-opened:
		t.Errorf("the open went past a gate held by a swap (%v)", r.err)
		opened <- r
	}
	if err := os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
	_ = gate.Release()

	r := <-opened
	if r.err != nil {
		t.Fatalf("the open after the swap: %v", r.err)
	}
	t.Cleanup(func() { _ = r.st.Close() })
	var v string
	if err := r.st.sqlDB.QueryRow(`SELECT value FROM metadata WHERE key = 'swap_probe'`).Scan(&v); err != nil {
		t.Fatalf("the opener does not read the replaced file: %v", err)
	}
}

// TestDSNFilePath_InMemoryNamesNoFile: an in-memory DSN takes no gate.
func TestDSNFilePath_InMemoryNamesNoFile(t *testing.T) {
	for _, dsn := range []string{":memory:", "file::memory:", "file:x?mode=memory&cache=shared"} {
		if got := dsnFilePath(dsn); got != "" && got != ":memory:" {
			t.Errorf("dsnFilePath(%q) = %q, want no file", dsn, got)
		}
	}
}
