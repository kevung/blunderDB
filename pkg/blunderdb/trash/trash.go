// Package trash is the undo of a delete (issue #285, ADR-0036).
//
// A delete is still a delete. What changes is that a JSON snapshot of what is
// about to disappear is written first, so the gesture can be undone for thirty
// days. Nothing else in the schema knows the trash exists — no search filter,
// no statistic, no retention predicate, no uniqueness index — which is exactly
// why this could be added without auditing all of them.
//
// Restoring is deliberately NOT symmetric with deleting. Putting a position
// back means re-Saving it, so the Zobrist deduplication decides where it
// lands: onto the row that already holds the position if one came back
// meanwhile, onto a new row otherwise. It never creates a duplicate, which is
// the invariant doing its job — but it does not preserve the row id, since the
// old row is gone and AUTOINCREMENT does not reuse it. A restored position is
// the same position, at a new number.
//
// Everything here is written against storage.Stores, so the desktop wrapper,
// the CLI and the daemon share one implementation rather than three. The trash
// is entirely made of Storage calls — unlike the analysis sweeps next door,
// whose GATHERING genuinely differs by mode.
package trash

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Position deletes a position after snapshotting it, its analysis and its
// comments. It returns the trash entry's id.
//
// The snapshot carries what CASCADED off the position — a restore that gave
// back a bare board and lost what the user had written on it would be a
// restore in name only. It does not carry the position's collection or deck
// membership: those rows belong to the collection and the deck, and the
// cascade removed the position from them.
func Position(ctx context.Context, s storage.Stores, scope string, positionID int64) (int64, error) {
	pos, err := s.Positions().Load(ctx, scope, positionID)
	if err != nil {
		return 0, err
	}
	payload := domain.TrashPositionPayload{Position: *pos}
	switch a, err := s.Analyses().Load(ctx, scope, positionID); {
	case err == nil:
		payload.Analysis = a
	case !errors.Is(err, storage.ErrNotFound):
		return 0, err
	}
	for c, err := range s.Comments().ByPosition(ctx, scope, positionID) {
		if err != nil {
			return 0, err
		}
		payload.Comments = append(payload.Comments, *c)
	}

	id, err := put(ctx, s, scope, domain.TrashPosition, fmt.Sprintf("Position %d", positionID), payload)
	if err != nil {
		return 0, err
	}
	// ON DELETE CASCADE takes the analysis, the comments and the memberships.
	if err := s.Positions().Delete(ctx, scope, positionID); err != nil {
		// The snapshot outlived the delete it was for: discard it rather than
		// leave a trash entry for something still in the database, which would
		// restore into a duplicate on the next click.
		_ = s.Trash().Discard(ctx, scope, id)
		return 0, err
	}
	return id, nil
}

// Collection deletes a collection after snapshotting it and the ids of the
// positions it held, in order. The positions themselves are untouched: a
// collection is a view over them.
func Collection(ctx context.Context, s storage.Stores, scope string, collectionID int64) (int64, error) {
	cs := s.Collections()
	col, err := cs.Get(ctx, scope, collectionID)
	if err != nil {
		return 0, err
	}
	payload := domain.TrashCollectionPayload{
		Name:        col.Name,
		Description: col.Description,
		SortOrder:   col.SortOrder,
	}
	for p, err := range cs.Positions(ctx, scope, collectionID, storage.ListOpts{}) {
		if err != nil {
			return 0, err
		}
		payload.PositionIDs = append(payload.PositionIDs, p.ID)
	}

	id, err := put(ctx, s, scope, domain.TrashCollection, col.Name, payload)
	if err != nil {
		return 0, err
	}
	if err := cs.Delete(ctx, scope, collectionID); err != nil {
		_ = s.Trash().Discard(ctx, scope, id)
		return 0, err
	}
	return id, nil
}

// CommentEntry deletes one comment entry after snapshotting it.
func CommentEntry(ctx context.Context, s storage.Stores, scope string, commentID int64) (int64, error) {
	entry, err := findComment(ctx, s, scope, commentID)
	if err != nil {
		return 0, err
	}
	label := entry.Text
	if r := []rune(label); len(r) > 60 {
		label = string(r[:60]) + "…"
	}
	id, err := put(ctx, s, scope, domain.TrashComment, label, domain.TrashCommentPayload{Comment: *entry})
	if err != nil {
		return 0, err
	}
	if err := s.Comments().Delete(ctx, scope, commentID); err != nil {
		_ = s.Trash().Discard(ctx, scope, id)
		return 0, err
	}
	return id, nil
}

// findComment locates one comment entry by id. CommentStore has no by-id read
// — nothing needed one until a snapshot did — so this walks the database-wide
// listing, which is bounded by the number of comments and only ever runs on an
// explicit delete.
func findComment(ctx context.Context, s storage.Stores, scope string, commentID int64) (*domain.CommentEntry, error) {
	for c, err := range s.Comments().ListAll(ctx, scope, storage.ListOpts{}) {
		if err != nil {
			return nil, err
		}
		if c.ID == commentID {
			return c, nil
		}
	}
	return nil, fmt.Errorf("comment %d: %w", commentID, storage.ErrNotFound)
}

// Restore puts one entry back and removes it from the trash.
//
// It returns the id of what was restored, whose meaning depends on the kind: a
// position id, a collection id, a comment id. A restore that cannot happen —
// the position a comment belonged to is itself gone — fails and leaves the
// trash entry alone, so nothing is lost by trying.
func Restore(ctx context.Context, s storage.Stores, scope string, trashID int64) (int64, error) {
	entry, err := s.Trash().Load(ctx, scope, trashID)
	if err != nil {
		return 0, err
	}
	var restored int64
	switch entry.Kind {
	case domain.TrashPosition:
		restored, err = restorePosition(ctx, s, scope, entry)
	case domain.TrashCollection:
		restored, err = restoreCollection(ctx, s, scope, entry)
	case domain.TrashComment:
		restored, err = restoreComment(ctx, s, scope, entry)
	default:
		err = fmt.Errorf("trash entry %d: nothing knows how to restore a %q", trashID, entry.Kind)
	}
	if err != nil {
		return 0, err
	}
	return restored, s.Trash().Discard(ctx, scope, trashID)
}

// restorePosition re-Saves the position, then puts back what cascaded off it.
//
// Save deduplicates by Zobrist hash, so the position lands on whatever row
// already holds it and on a new one otherwise — never on a duplicate, and
// never on the id it had. The analysis is written only when the target carries none: a
// restore must not overwrite an analysis somebody has since produced for the
// same position (ADR-0013's spirit — a stored analysis is never overwritten by
// something that is not asking to correct it).
func restorePosition(ctx context.Context, s storage.Stores, scope string, entry *domain.TrashEntry) (int64, error) {
	var payload domain.TrashPositionPayload
	if err := json.Unmarshal(entry.Payload, &payload); err != nil {
		return 0, fmt.Errorf("trash entry %d: %w", entry.ID, err)
	}
	pos := payload.Position
	id, err := s.Positions().Save(ctx, scope, &pos)
	if err != nil {
		return 0, err
	}
	if payload.Analysis != nil {
		switch _, err := s.Analyses().Load(ctx, scope, id); {
		case errors.Is(err, storage.ErrNotFound):
			if err := s.Analyses().Save(ctx, scope, id, payload.Analysis); err != nil {
				return 0, err
			}
		case err != nil:
			return 0, err
		}
	}
	// Comments are appended, not replaced: a comment written on the position
	// since the delete is somebody's work too.
	existing := map[string]bool{}
	for c, err := range s.Comments().ByPosition(ctx, scope, id) {
		if err != nil {
			return 0, err
		}
		existing[c.Text] = true
	}
	for _, c := range payload.Comments {
		if c.Text == "" || existing[c.Text] {
			continue
		}
		if _, err := s.Comments().AddFrom(ctx, scope, id, c.Text, c.Origin); err != nil {
			return 0, err
		}
	}
	return id, nil
}

// restoreCollection recreates the collection and re-adds the positions that
// still exist. A member deleted since is simply absent — the alternative would
// be to refuse the whole restore over one missing row.
func restoreCollection(ctx context.Context, s storage.Stores, scope string, entry *domain.TrashEntry) (int64, error) {
	var payload domain.TrashCollectionPayload
	if err := json.Unmarshal(entry.Payload, &payload); err != nil {
		return 0, fmt.Errorf("trash entry %d: %w", entry.ID, err)
	}
	cs := s.Collections()
	id, err := cs.Create(ctx, scope, payload.Name, payload.Description)
	if err != nil {
		return 0, err
	}
	var alive []int64
	for _, pid := range payload.PositionIDs {
		if _, err := s.Positions().Load(ctx, scope, pid); err == nil {
			alive = append(alive, pid)
		}
	}
	if len(alive) > 0 {
		if err := cs.AddPositions(ctx, scope, id, alive); err != nil {
			return 0, err
		}
	}
	return id, nil
}

// restoreComment puts a comment back on its position, if the position is still
// there.
func restoreComment(ctx context.Context, s storage.Stores, scope string, entry *domain.TrashEntry) (int64, error) {
	var payload domain.TrashCommentPayload
	if err := json.Unmarshal(entry.Payload, &payload); err != nil {
		return 0, fmt.Errorf("trash entry %d: %w", entry.ID, err)
	}
	if _, err := s.Positions().Load(ctx, scope, payload.Comment.PositionID); err != nil {
		return 0, fmt.Errorf("restoring a comment on position %d: %w", payload.Comment.PositionID, err)
	}
	return s.Comments().AddFrom(ctx, scope, payload.Comment.PositionID,
		payload.Comment.Text, payload.Comment.Origin)
}

// put marshals a payload and writes the snapshot.
func put(ctx context.Context, s storage.Stores, scope string, kind domain.TrashKind, label string, payload any) (int64, error) {
	blob, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	return s.Trash().Put(ctx, scope, kind, label, blob)
}
