package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/trash"
)

// The trash, as the desktop and the CLI reach it (ADR-0036): package trash
// does the work over storage.Stores; this layer only adds Database.mu.

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

// TrashMatch deletes a match after snapshotting it whole — games, moves,
// analyses, and the positions its moves reached — and returns the trash
// entry's id. The positions nothing else holds are purged as by DeleteMatch.
func (d *Database) TrashMatch(matchID int64) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}
	return trash.Match(context.Background(), d.store, "", matchID)
}

// RestoreFromTrash puts one entry back and removes it from the trash. The id
// it returns is a position, collection, comment or match id, depending on the kind.
//
// A restored Rencontre attaches its events again, and they come back on the room's tables as
// attaching aligns them: detached, each kept its own.
func (d *Database) RestoreFromTrash(trashID int64) (int64, error) {
	kind, err := d.trashKind(trashID)
	if err != nil {
		return 0, err
	}
	if kind == domain.TrashRencontre {
		// A gesture on the room's events, under their locks: the service's, as for the CLI
		// and the daemon.
		return d.directionService().RestoreFromTrash(context.Background(), trashID)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}
	return trash.Restore(context.Background(), d.store, "", trashID)
}

// trashKind is the kind of one trash entry.
func (d *Database) trashKind(trashID int64) (domain.TrashKind, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return "", fmt.Errorf("no database is currently open")
	}
	entry, err := d.store.Trash().Load(context.Background(), "", trashID)
	if err != nil {
		return "", err
	}
	return entry.Kind, nil
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
