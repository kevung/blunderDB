package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite/filelock"
)

// TestOpen_WaitsAtTheSwapGate: the serve daemon and migrate open a library
// through Open; one that reaches the file while a vacuum replaces it waits
// for the rename and connects to the new file, never to the inode the rename
// unlinks, where whatever it wrote would be lost.
func TestOpen_WaitsAtTheSwapGate(t *testing.T) {
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

	gate, ok, err := filelock.TryExclusive(filelock.SwapGatePath(path))
	if err != nil || !ok {
		t.Fatalf("taking the swap gate: %v, %v", ok, err)
	}
	atGate := make(chan struct{})
	hook := func(dsn string) {
		if dsn == path {
			close(atGate)
		}
	}
	beforeSwapGate.Store(&hook)
	t.Cleanup(func() { beforeSwapGate.Store(nil) })

	type result struct {
		st  *Storage
		err error
	}
	opened := make(chan result, 1)
	go func() {
		st, err := Open(ctx, path, nil)
		opened <- result{st, err}
	}()
	<-atGate
	select {
	case r := <-opened:
		t.Errorf("the open went past a gate held by a swap (%v)", r.err)
		opened <- r
	default:
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
