package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestOpen_MaxConnsSizesThePool: Options.MaxConns bounds a file's pool, the
// default stays the desktop's, and ":memory:" keeps its one connection.
func TestOpen_MaxConnsSizesThePool(t *testing.T) {
	ctx := context.Background()
	for _, c := range []struct {
		name string
		dsn  string
		opts *storage.Options
		want int
	}{
		{"default", filepath.Join(t.TempDir(), "a.db"), nil, 10},
		{"bounded", filepath.Join(t.TempDir(), "b.db"), &storage.Options{MaxConns: 3}, 3},
		{"memory", ":memory:", &storage.Options{MaxConns: 16}, 1},
	} {
		st, err := Open(ctx, c.dsn, c.opts)
		if err != nil {
			t.Fatal(err)
		}
		if got := st.sqlDB.Stats().MaxOpenConnections; got != c.want {
			t.Errorf("%s: pool of %d, want %d", c.name, got, c.want)
		}
		st.Close()
	}
}

// TestRefusesWrites_AskedOncePerHandle: once read, the answer needs no
// pooled connection — with the pool's one connection held elsewhere, it
// still comes back at once.
func TestRefusesWrites_AskedOncePerHandle(t *testing.T) {
	ctx := context.Background()
	st, err := Open(ctx, filepath.Join(t.TempDir(), "a.db"), &storage.Options{MaxConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if shared(st.binder).RefusesWrites(ctx) {
		t.Fatal("a writable file refuses writes")
	}
	conn, err := st.sqlDB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	short, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	start := time.Now()
	if shared(st.binder).RefusesWrites(short) {
		t.Error("the cached answer flipped")
	}
	if d := time.Since(start); d > time.Second {
		t.Errorf("the second ask waited %v for a connection: not cached", d)
	}
}
