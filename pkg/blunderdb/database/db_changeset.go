package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ChangesetExport is one page of the change journal after a cursor, for the
// desktop's synchronisation with a remote library. Export may adopt rows that
// predate the journal, so it takes the write lock.
func (d *Database) ChangesetExport(ctx context.Context, after storage.Cursor, limit int) (changeset.Set, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return changeset.Set{}, fmt.Errorf("no database is currently open")
	}
	return changeset.Export(ctx, d.store, "", after, limit)
}

// ChangesetImport merges a page of records into the open library, the last
// write winning.
func (d *Database) ChangesetImport(ctx context.Context, set changeset.Set) (changeset.Report, error) {
	if err := d.RefuseReadOnly(); err != nil {
		return changeset.Report{}, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return changeset.Report{}, fmt.Errorf("no database is currently open")
	}
	return changeset.Import(ctx, d.store, "", set)
}
