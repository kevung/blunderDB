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
type Journal struct{ d *Database }

// JournalOf returns the change journal of d.
func JournalOf(d *Database) Journal { return Journal{d} }

// Export is one page of the journal after a cursor. It may adopt rows that
// predate the journal, so it takes the write lock.
func (j Journal) Export(ctx context.Context, after storage.Cursor, limit int) (changeset.Set, error) {
	j.d.mu.Lock()
	defer j.d.mu.Unlock()
	if j.d.db == nil {
		return changeset.Set{}, fmt.Errorf("no database is currently open")
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
	if j.d.db == nil {
		return changeset.Report{}, fmt.Errorf("no database is currently open")
	}
	return changeset.Import(ctx, j.d.store, "", set)
}
