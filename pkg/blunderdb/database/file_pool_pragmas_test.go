package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"
)

// TestFilePoolEveryConnectionCarriesPragmas guards issue #157: a file-backed
// database is served by a pool of up to ten connections, and a PRAGMA run
// after sql.Open only configures the ONE connection it happens to run on.
// The other connections used to run with foreign_keys=OFF and busy_timeout=0,
// so a DeleteMatch landing on one of them skipped the ON DELETE CASCADE and
// left game/move/move_analysis rows orphaned. The PRAGMAs must therefore be
// carried by the DSN (sqlite.DSN), which the driver replays on every
// connection it opens.
//
// The test pins the pool to three connections, checks all three out at once
// so that they cannot be the same underlying connection, and reads the two
// PRAGMAs that matter on each. Both SetupDatabase and OpenDatabase open the
// file, so both are exercised.
func TestFilePoolEveryConnectionCarriesPragmas(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pool.db")

	seed := NewDatabase()
	if err := seed.SetupDatabase(path); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	assertPoolPragmas(t, "SetupDatabase", seed.conn())
	if err := seed.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	opened := NewDatabase()
	if err := opened.OpenDatabase(path); err != nil {
		t.Fatalf("OpenDatabase: %v", err)
	}
	t.Cleanup(func() { _ = opened.Close() })
	assertPoolPragmas(t, "OpenDatabase", opened.conn())
}

func assertPoolPragmas(t *testing.T, via string, db *sql.DB) {
	t.Helper()
	const conns = 3
	db.SetMaxOpenConns(conns)

	type reading struct {
		foreignKeys int
		busyTimeout int
		err         error
	}
	readings := make([]reading, conns)

	// Each goroutine holds its dedicated connection until every other
	// goroutine has obtained one, so the three readings come from three
	// distinct pooled connections rather than one connection reused thrice.
	var ready sync.WaitGroup
	ready.Add(conns)
	var done sync.WaitGroup
	for i := range conns {
		done.Go(func() {
			ctx := context.Background()
			conn, err := db.Conn(ctx)
			if err != nil {
				readings[i].err = err
				ready.Done()
				return
			}
			defer conn.Close()
			ready.Done()
			ready.Wait()
			r := &readings[i]
			if err := conn.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&r.foreignKeys); err != nil {
				r.err = err
				return
			}
			r.err = conn.QueryRowContext(ctx, `PRAGMA busy_timeout`).Scan(&r.busyTimeout)
		})
	}
	done.Wait()

	for i, r := range readings {
		if r.err != nil {
			t.Fatalf("%s: connection %d: %v", via, i, r.err)
		}
		if r.foreignKeys != 1 {
			t.Errorf("%s: connection %d: PRAGMA foreign_keys = %d, want 1", via, i, r.foreignKeys)
		}
		if r.busyTimeout != 10000 {
			t.Errorf("%s: connection %d: PRAGMA busy_timeout = %d, want 10000", via, i, r.busyTimeout)
		}
	}
}
