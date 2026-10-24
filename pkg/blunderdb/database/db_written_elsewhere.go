package database

import (
	"context"
	"database/sql"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// WrittenElsewhere reports whether another process (the CLI, a serve daemon)
// committed to d's library since the previous call; the writes made through d
// or any handle borrowed from it are left out (storage.WriteWatcher). False
// when nothing is open, d does not watch (WatchWrittenElsewhere) or the
// backend has no such watch. A function, not a method, so the binding does
// not carry it: the desktop asks for every group at once
// (App.LibrariesWrittenElsewhere). Reads only.
//
// It never waits: while d is held for writing (a compaction, the end of an
// import, which last minutes), it answers false at once and the change, if
// any, is reported by a later call, the watch keeping it until then.
func WrittenElsewhere(d *Database) (bool, error) {
	if !d.mu.TryRLock() {
		return false, nil
	}
	defer d.mu.RUnlock()
	w, ok := d.store.(storage.WriteWatcher)
	if !ok || d.db == nil {
		return false, nil
	}
	return w.WrittenElsewhere(context.Background())
}

// WatchWrittenElsewhere makes every library d opens from now on answer
// WrittenElsewhere; without it the pool is opened plainly and WrittenElsewhere
// answers false. The watch costs every write a few reads, so only the desktop,
// which polls, asks for it. A function, for the same reason as
// WrittenElsewhere.
func WatchWrittenElsewhere(d *Database) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.watchWrites = true
}

// openPool opens the pool of the library at path, watched when asked.
func (d *Database) openPool(path string) (*sql.DB, error) {
	if d.watchWrites {
		return sqlite.OpenDB(path)
	}
	return sql.Open("sqlite", sqlite.DSN(path))
}
