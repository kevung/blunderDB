package database

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// WrittenElsewhere reports whether another process (the CLI, a serve daemon)
// committed to d's library since the previous call; the writes made through d
// or any handle borrowed from it are left out (storage.WriteWatcher). False
// when nothing is open or the backend has no such watch. A function, not a
// method, so the binding does not carry it: the desktop asks for every group
// at once (App.LibrariesWrittenElsewhere). Reads only.
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
