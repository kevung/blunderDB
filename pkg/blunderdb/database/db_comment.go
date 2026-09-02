package database

import (
	"context"
	"fmt"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The comment family is an adapter over storage.CommentStore: the SQL lives
// once, in storage/sqlite, under the contract suite both backends pass
// (storagetest, Comment/*). Every method takes d.mu the way it always did
// and passes the desktop's implicit tenant ("") as scope.

// commentStore returns the comment family of the open database, or the
// error every wrapper method reports when no database is open. The caller
// holds d.mu.
func (d *Database) commentStore() (storage.CommentStore, error) {
	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}
	return d.store.Comments(), nil
}

// collectComments drains a comment stream into the slice the callers expect
// (nil when the stream is empty, as the row scan it replaces produced).
func collectComments(seq iter.Seq2[*CommentEntry, error]) ([]CommentEntry, error) {
	var entries []CommentEntry
	for e, err := range seq {
		if err != nil {
			return nil, err
		}
		entries = append(entries, *e)
	}
	return entries, nil
}

// DeleteComment removes every comment entry of a position.
func (d *Database) DeleteComment(positionID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	cs, err := d.commentStore()
	if err != nil {
		return err
	}
	return cs.DeleteForPosition(context.Background(), "", positionID)
}

// AddComment inserts a new comment entry for a position (allows multiple per position)
func (d *Database) AddComment(positionID int64, text string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	cs, err := d.commentStore()
	if err != nil {
		return err
	}
	_, err = cs.Add(context.Background(), "", positionID, text)
	return err
}

// UpdateCommentEntry updates a specific comment by its ID
func (d *Database) UpdateCommentEntry(commentID int64, text string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	cs, err := d.commentStore()
	if err != nil {
		return err
	}
	return cs.Update(context.Background(), "", commentID, text)
}

// DeleteCommentEntry deletes a specific comment by its ID
func (d *Database) DeleteCommentEntry(commentID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	cs, err := d.commentStore()
	if err != nil {
		return err
	}
	return cs.Delete(context.Background(), "", commentID)
}

// SaveComment writes text as the position's primary comment: it rewrites the
// oldest entry when the position already has one and inserts a new entry
// otherwise. LoadComment reads that same entry back, so the pair the tag
// commands and the importers run (load, edit, save) touches one entry of the
// wall and never duplicates the others.
func (d *Database) SaveComment(positionID int64, text string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	cs, err := d.commentStore()
	if err != nil {
		return err
	}
	_, err = cs.Upsert(context.Background(), "", positionID, text)
	return err
}

// LoadComment returns the text of the position's primary comment — the entry
// SaveComment rewrites — or "" when the position has none.
func (d *Database) LoadComment(positionID int64) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	cs, err := d.commentStore()
	if err != nil {
		return "", err
	}
	// ByPosition streams most recent first: the last entry is the oldest one,
	// the primary comment Upsert edits.
	var text string
	for e, err := range cs.ByPosition(context.Background(), "", positionID) {
		if err != nil {
			return "", err
		}
		text = e.Text
	}
	return text, nil
}

// GetCommentsByPosition returns all non-empty comments for a given position, ordered by comment ID descending
func (d *Database) GetCommentsByPosition(positionID int64) ([]CommentEntry, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	cs, err := d.commentStore()
	if err != nil {
		return nil, err
	}
	return collectComments(cs.ByPosition(context.Background(), "", positionID))
}

// GetAllComments returns all non-empty comments, ordered by comment ID descending (most recent first)
func (d *Database) GetAllComments() ([]CommentEntry, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	cs, err := d.commentStore()
	if err != nil {
		return nil, err
	}
	return collectComments(cs.ListAll(context.Background(), ""))
}

// SearchComments searches for comments containing the given query string (case-insensitive)
func (d *Database) SearchComments(query string) ([]CommentEntry, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	cs, err := d.commentStore()
	if err != nil {
		return nil, err
	}
	return collectComments(cs.Search(context.Background(), "", query))
}
