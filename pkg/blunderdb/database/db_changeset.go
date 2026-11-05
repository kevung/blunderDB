package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Journal is the open library's change journal, for the desktop's
// synchronisation with a remote library. It is a handle apart from Database so
// that the window's bindings do not carry it.
// It belongs to the file open when it was taken: once the handle opens another
// one, every call fails with ErrDatabaseChanged rather than read or write the
// wrong library.
type Journal struct {
	d    *Database
	path string
	gen  uint64
}

// JournalOf returns the change journal of the library d has open now.
func JournalOf(d *Database) Journal {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return Journal{d: d, path: d.path, gen: d.generation}
}

// Path is the file the journal was taken on.
func (j Journal) Path() string { return j.path }

// current fails unless the handle still holds the library of the journal. The
// caller holds d.mu.
func (j Journal) current() error {
	if j.d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	if j.d.path != j.path || j.d.generation != j.gen {
		return ErrDatabaseChanged
	}
	return nil
}

// Export is one page of the journal after a cursor. It may adopt rows that
// predate the journal, so it takes the write lock.
func (j Journal) Export(ctx context.Context, after storage.Cursor, limit int) (changeset.Set, error) {
	j.d.mu.Lock()
	defer j.d.mu.Unlock()
	if err := j.current(); err != nil {
		return changeset.Set{}, err
	}
	return changeset.Export(ctx, j.d.store, "", after, limit)
}

// Import merges a page of records into the open library, the last write winning.
func (j Journal) Import(ctx context.Context, set changeset.Set) (changeset.Report, error) {
	if err := j.d.RefuseReadOnly(); err != nil {
		return changeset.Report{}, err
	}
	j.d.mu.Lock()
	defer j.d.mu.Unlock()
	if err := j.current(); err != nil {
		return changeset.Report{}, err
	}
	return changeset.Import(ctx, j.d.store, "", set)
}
