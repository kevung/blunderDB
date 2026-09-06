package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/trash"
)

// The trash, as the desktop and the CLI reach it (issue #285, ADR-0036).
//
// Everything these methods do lives in package trash, written once against
// storage.Stores: the trash is entirely made of Storage calls, so there is no
// half of it that is genuinely this mode's. What is this mode's is the lock —
// Database.mu, which the daemon does not have and does not need.

// TrashPosition deletes a position after snapshotting it, its analysis and its
// comments, and returns the trash entry's id.
func (d *Database) TrashPosition(positionID int64) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}
	return trash.Position(context.Background(), d.store, "", positionID)
}

// TrashCollection deletes a collection after snapshotting it and the positions
// it held. The positions themselves are untouched.
func (d *Database) TrashCollection(collectionID int64) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}
	return trash.Collection(context.Background(), d.store, "", collectionID)
}

// TrashCommentEntry deletes one comment entry after snapshotting it.
func (d *Database) TrashCommentEntry(commentID int64) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}
	return trash.CommentEntry(context.Background(), d.store, "", commentID)
}

// RestoreFromTrash puts one entry back and removes it from the trash. The id
// it returns is a position, collection or comment id, depending on the kind.
func (d *Database) RestoreFromTrash(trashID int64) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}
	return trash.Restore(context.Background(), d.store, "", trashID)
}

// ListTrash returns the trash, most recently deleted first. kind narrows to
// one kind of entry; empty lists them all.
func (d *Database) ListTrash(kind string, limit, offset int) ([]*domain.TrashEntry, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}
	return d.store.Trash().List(context.Background(), "", domain.TrashKind(kind),
		storage.ListOpts{Limit: limit, Offset: offset})
}

// CountTrash is how many entries the trash holds — what a panel needs before
// deciding whether to offer itself at all.
func (d *Database) CountTrash() (int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}
	return d.store.Trash().Count(context.Background(), "")
}

// DiscardFromTrash drops one entry without restoring it.
func (d *Database) DiscardFromTrash(trashID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Trash().Discard(context.Background(), "", trashID)
}

// EmptyTrash drops every entry older than olderThanDays, or all of them when
// olderThanDays is 0. `blunderdb vacuum` calls it with
// domain.TrashRetentionDays; nothing purges on open.
func (d *Database) EmptyTrash(olderThanDays int) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}
	return d.store.Trash().Purge(context.Background(), "", olderThanDays)
}
