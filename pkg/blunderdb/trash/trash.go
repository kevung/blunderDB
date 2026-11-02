// Package trash is the undo of a delete (ADR-0036).
//
// A delete is still a delete, preceded by a JSON snapshot of what disappears,
// restorable for thirty days. Nothing else in the schema knows the trash exists
// (no search filter, statistic, retention predicate or uniqueness index).
//
// Restoring is NOT symmetric with deleting: a position is re-Saved, so Zobrist
// deduplication decides where it lands — never a duplicate. It takes back its
// old id when that id is free and the hash is not stored under another one.
//
// Everything here is written against storage.Stores, so the desktop wrapper,
// the CLI and the daemon share one implementation.
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
// uuidOf is the identifier of an object, "" when the backend keeps none.
func uuidOf(ctx context.Context, s storage.Stores, scope, family string, id int64) (string, error) {
	p, ok := s.(storage.ObjectUUIDProvider)
	if !ok {
		return "", nil
	}
	return p.ObjectUUIDs().ObjectUUID(ctx, scope, family, id)
}

// giveBackUUID restores the identifier a deleted object held. An empty one (a
// snapshot from before identifiers) leaves the object the one it was given.
func giveBackUUID(ctx context.Context, s storage.Stores, scope, family string, id int64, uuid string) error {
	p, ok := s.(storage.ObjectUUIDProvider)
	if !ok || uuid == "" {
		return nil
	}
	return p.ObjectUUIDs().SetObjectUUID(ctx, scope, family, id, uuid)
}

func Position(ctx context.Context, s storage.Stores, scope string, positionID int64) (int64, error) {
	var id int64
	err := inTx(ctx, s, scope, func(s storage.Stores) error {
		payload, err := snapshotPosition(ctx, s, scope, positionID)
		if err != nil {
			return err
		}
		if id, err = put(ctx, s, scope, domain.TrashPosition, fmt.Sprintf("Position %d", positionID), payload); err != nil {
			return err
		}
		// ON DELETE CASCADE takes the analysis, the comments and the memberships.
		if err := s.Positions().Delete(ctx, scope, positionID); err != nil {
			return err
		}
		// Deleted, but to the trash: the journal says so for the position and
		// for the notes that went with it.
		if err := markTrashed(ctx, s, scope, "position", payload.UUID); err != nil {
			return err
		}
		for _, u := range payload.CommentUUIDs {
			if err := markTrashed(ctx, s, scope, "comment", u); err != nil {
				return err
			}
		}
		return nil
	})
	return id, err
}

// markTrashed tells the change journal that a delete went through the trash.
func markTrashed(ctx context.Context, s storage.Stores, scope, family, uuid string) error {
	p, ok := s.(storage.ChangeJournalProvider)
	if !ok || uuid == "" {
		return nil
	}
	return p.ChangeJournal().Mark(ctx, scope, family, uuid, storage.ChangeTrash, "")
}

// snapshotPosition reads a position with what cascades off it: its analysis,
// the table that analysis was valued with, and its comments.
func snapshotPosition(ctx context.Context, s storage.Stores, scope string, positionID int64) (domain.TrashPositionPayload, error) {
	pos, err := s.Positions().Load(ctx, scope, positionID)
	if err != nil {
		return domain.TrashPositionPayload{}, err
	}
	payload := domain.TrashPositionPayload{Position: *pos}
	if payload.UUID, err = uuidOf(ctx, s, scope, "position", positionID); err != nil {
		return domain.TrashPositionPayload{}, err
	}
	switch a, err := s.Analyses().Load(ctx, scope, positionID); {
	case err == nil:
		payload.Analysis = a
		if payload.MET, err = s.MatchEquityTables().OfAnalysis(ctx, scope, positionID); err != nil {
			return domain.TrashPositionPayload{}, err
		}
	case !errors.Is(err, storage.ErrNotFound):
		return domain.TrashPositionPayload{}, err
	}
	for c, err := range s.Comments().ByPosition(ctx, scope, positionID) {
		if err != nil {
			return domain.TrashPositionPayload{}, err
		}
		payload.Comments = append(payload.Comments, *c)
	}
	for _, c := range payload.Comments {
		u, err := uuidOf(ctx, s, scope, "comment", c.ID)
		if err != nil {
			return domain.TrashPositionPayload{}, err
		}
		if u != "" {
			if payload.CommentUUIDs == nil {
				payload.CommentUUIDs = map[int64]string{}
			}
			payload.CommentUUIDs[c.ID] = u
		}
	}
	if payload.StudiedAt, err = s.ImportBatches().StudyMark(ctx, scope, positionID); err != nil {
		return domain.TrashPositionPayload{}, err
	}
	if payload.TrainingItemIDs, err = s.Training().ItemsOfPosition(ctx, scope, positionID); err != nil {
		return domain.TrashPositionPayload{}, err
	}
	if err := snapshotMarks(ctx, s, scope, positionID, &payload); err != nil {
		return domain.TrashPositionPayload{}, err
	}
	return payload, nil
}

// snapshotMarks reads what the deleted position's own tables carry: its
// public id, its drawing and its derivation links, in both directions.
func snapshotMarks(ctx context.Context, s storage.Stores, scope string, positionID int64, payload *domain.TrashPositionPayload) error {
	var err error
	if p, ok := s.(storage.PublicIDProvider); ok {
		if payload.PublicID, err = p.PublicIDs().PublicID(ctx, scope, positionID); err != nil {
			return err
		}
	}
	if p, ok := s.(storage.DrawingProvider); ok {
		if payload.Drawing, err = p.PositionDrawings().Drawing(ctx, scope, positionID); err != nil {
			return err
		}
	}
	if p, ok := s.(storage.DerivationProvider); ok {
		d := p.PositionDerivations()
		if payload.DerivedFrom, err = d.DerivedFrom(ctx, scope, positionID); err != nil {
			return err
		}
		ids, err := d.Variants(ctx, scope, positionID)
		if err != nil {
			return err
		}
		links, err := d.Derivations(ctx, scope, ids)
		if err != nil {
			return err
		}
		for _, id := range ids {
			payload.Variants = append(payload.Variants, domain.Derived{ID: id, Move: links[id].Move})
		}
	}
	if p, ok := s.(storage.ChangeJournalProvider); ok {
		for _, part := range []string{storage.PartDrawing, storage.PartDerivation} {
			f, known, err := p.ChangeJournal().PartFact(ctx, scope, positionID, part)
			if err != nil {
				return err
			}
			if known {
				payload.PartFacts = append(payload.PartFacts, domain.TrashPartFact{Part: part, At: f.At, PendingOrigin: f.PendingOrigin, PendingMove: f.PendingMove})
			}
		}
	}
	return nil
}

// restoreMarks puts back what snapshotMarks read, onto the position now
// called id. It fills what is missing and never overwrites: a drawing, a
// label or a link written since the delete is the newer work. A link whose
// other end is gone, or that would now close a loop, is left out.
func restoreMarks(ctx context.Context, s storage.Stores, scope string, id int64, payload domain.TrashPositionPayload) error {
	giveBackFacts, err := keptFacts(ctx, s, scope, id, payload)
	if err != nil {
		return err
	}
	if p, ok := s.(storage.PublicIDProvider); ok && payload.PublicID != "" {
		have, err := p.PublicIDs().PublicID(ctx, scope, id)
		if err != nil {
			return err
		}
		if have == "" {
			if err := p.PublicIDs().SetPublicID(ctx, scope, id, payload.PublicID); err != nil {
				return err
			}
		}
	}
	if p, ok := s.(storage.DrawingProvider); ok && len(payload.Drawing) > 0 {
		have, err := p.PositionDrawings().Drawing(ctx, scope, id)
		if err != nil {
			return err
		}
		if len(have) == 0 {
			if err := p.PositionDrawings().SetDrawing(ctx, scope, id, payload.Drawing); err != nil {
				return err
			}
		}
	}
	if err := restoreLinks(ctx, s, scope, id, payload, nil); err != nil {
		return err
	}
	return giveBackFacts()
}

// keptFacts picks the snapshot's part facts the position has no fact for yet,
// and gives what writes them. It runs before the marks go back, because the
// triggers date that write now: a restore is not a new drawing or a new link.
func keptFacts(ctx context.Context, s storage.Stores, scope string, id int64, payload domain.TrashPositionPayload) (func() error, error) {
	p, ok := s.(storage.ChangeJournalProvider)
	if !ok {
		return func() error { return nil }, nil
	}
	var missing []domain.TrashPartFact
	for _, f := range payload.PartFacts {
		_, known, err := p.ChangeJournal().PartFact(ctx, scope, id, f.Part)
		if err != nil {
			return nil, err
		}
		if !known {
			missing = append(missing, f)
		}
	}
	return func() error {
		for _, f := range missing {
			err := p.ChangeJournal().SetPartFact(ctx, scope, storage.PartFact{PositionID: id, Part: f.Part, At: f.At, PendingOrigin: f.PendingOrigin, PendingMove: f.PendingMove})
			if err != nil {
				return err
			}
		}
		return nil
	}, nil
}

// restoreLinks puts back the derivation links of a restored position. The
// ids in the payload are the ones the library had; remap says where a
// position restored under another id went (nil when none moved).
func restoreLinks(ctx context.Context, s storage.Stores, scope string, id int64, payload domain.TrashPositionPayload, remap map[int64]int64) error {
	p, ok := s.(storage.DerivationProvider)
	if !ok {
		return nil
	}
	moved := func(x int64) int64 {
		if n, ok := remap[x]; ok {
			return n
		}
		return x
	}
	d := p.PositionDerivations()
	link := func(variant int64, l domain.Derivation) error {
		have, err := d.DerivedFrom(ctx, scope, variant)
		if err != nil || have != nil {
			return err
		}
		switch err := d.SetDerivedFrom(ctx, scope, variant, &l); {
		case errors.Is(err, storage.ErrInvalid), errors.Is(err, storage.ErrNotFound):
			return nil
		default:
			return err
		}
	}
	if payload.DerivedFrom != nil {
		if err := link(id, domain.Derivation{FromID: moved(payload.DerivedFrom.FromID), Move: payload.DerivedFrom.Move}); err != nil {
			return err
		}
	}
	for _, v := range payload.Variants {
		if err := link(moved(v.ID), domain.Derivation{FromID: id, Move: v.Move}); err != nil {
			return err
		}
	}
	return nil
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
	colUUID, err := uuidOf(ctx, s, scope, "collection", collectionID)
	if err != nil {
		return 0, err
	}
	payload := domain.TrashCollectionPayload{
		UUID:        colUUID,
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

	var id int64
	err = inTx(ctx, s, scope, func(s storage.Stores) error {
		if id, err = put(ctx, s, scope, domain.TrashCollection, col.Name, payload); err != nil {
			return err
		}
		if err := s.Collections().Delete(ctx, scope, collectionID); err != nil {
			return err
		}
		return markTrashed(ctx, s, scope, "collection", colUUID)
	})
	return id, err
}

// CommentEntry deletes one comment entry after snapshotting it.
func CommentEntry(ctx context.Context, s storage.Stores, scope string, commentID int64) (int64, error) {
	entry, err := findComment(ctx, s, scope, commentID)
	if err != nil {
		return 0, err
	}
	label := commentLabel(entry)
	commentUUID, err := uuidOf(ctx, s, scope, "comment", commentID)
	if err != nil {
		return 0, err
	}
	var id int64
	err = inTx(ctx, s, scope, func(s storage.Stores) error {
		if id, err = put(ctx, s, scope, domain.TrashComment, label, domain.TrashCommentPayload{UUID: commentUUID, Comment: *entry}); err != nil {
			return err
		}
		if err := s.Comments().Delete(ctx, scope, commentID); err != nil {
			return err
		}
		return markTrashed(ctx, s, scope, "comment", commentUUID)
	})
	return id, err
}

func commentLabel(entry *domain.CommentEntry) string {
	if r := []rune(entry.Text); len(r) > 60 {
		return string(r[:60]) + "…"
	}
	return entry.Text
}

// KeepCommentVersion puts a copy of a note about to be overwritten in the
// trash, under the same purge window. It has no uuid: restoring it adds a note
// beside the current one, and nothing that discards the note's trash entries
// by uuid reaches it.
func KeepCommentVersion(ctx context.Context, s storage.Stores, scope string, entry *domain.CommentEntry) error {
	_, err := put(ctx, s, scope, domain.TrashComment, commentLabel(entry), domain.TrashCommentPayload{Comment: *entry})
	return err
}

// findComment locates one comment entry by id. CommentStore has no by-id read,
// so this walks the database-wide listing; it only runs on an explicit delete.
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
// position id, a collection id, a comment id, a match id; and the warnings of
// a restore that succeeded without putting everything back (a match's
// Direction Slot taken since). A restore that cannot happen —
// the position a comment belonged to is itself gone — fails and leaves the
// trash entry alone, so nothing is lost by trying. A Rencontre is not restored
// here (ErrRencontreByService).
func Restore(ctx context.Context, s storage.Stores, scope string, trashID int64) (domain.TrashRestore, error) {
	entry, err := s.Trash().Load(ctx, scope, trashID)
	if err != nil {
		return domain.TrashRestore{}, err
	}
	var restored int64
	switch entry.Kind {
	case domain.TrashPosition:
		restored, err = restorePosition(ctx, s, scope, entry)
	case domain.TrashCollection:
		restored, err = restoreCollection(ctx, s, scope, entry)
	case domain.TrashComment:
		restored, err = restoreComment(ctx, s, scope, entry)
	case domain.TrashMatch:
		// Discards the entry inside its own transaction.
		return restoreMatch(ctx, s, scope, entry)
	case domain.TrashRencontre:
		err = fmt.Errorf("trash entry %d: %w", trashID, ErrRencontreByService)
	default:
		err = fmt.Errorf("trash entry %d: nothing knows how to restore a %q", trashID, entry.Kind)
	}
	if err != nil {
		return domain.TrashRestore{}, err
	}
	if err := s.Trash().Discard(ctx, scope, trashID); err != nil {
		return domain.TrashRestore{}, err
	}
	return domain.TrashRestore{ID: restored}, nil
}

// restorePosition re-Saves the position, then puts back what cascaded off it.
//
// The analysis is written only when the target carries none: a restore must
// not overwrite an analysis produced since (ADR-0013).
func restorePosition(ctx context.Context, s storage.Stores, scope string, entry *domain.TrashEntry) (int64, error) {
	var payload domain.TrashPositionPayload
	if err := json.Unmarshal(entry.Payload, &payload); err != nil {
		return 0, fmt.Errorf("trash entry %d: %w", entry.ID, err)
	}
	id, _, err := restorePositionPayload(ctx, s, scope, payload, false)
	return id, err
}

// restorePositionPayload re-Saves a snapshotted position and puts back its
// analysis and comments; see restorePosition. The bool says the position was
// left as it was (onlyIfGone). With onlyIfGone, a position the
// database still holds under its own id is left as it is: it stayed in the
// library while the snapshot waited, so a note edited or deleted since must
// not come back. A position that was purged and stored again since under
// another id is a different case: it gets back what it lacks — its study mark,
// its training answers, an analysis and notes — and keeps what it has.
func restorePositionPayload(ctx context.Context, s storage.Stores, scope string, payload domain.TrashPositionPayload, onlyIfGone bool) (int64, bool, error) {
	pos := payload.Position
	id, created, err := reinstatePosition(ctx, s, scope, &pos)
	if err != nil {
		return 0, false, err
	}
	if onlyIfGone && !created && id == payload.Position.ID {
		return id, true, nil
	}
	if created {
		if err := giveBackUUID(ctx, s, scope, "position", id, payload.UUID); err != nil {
			return 0, false, err
		}
	}
	if err := restoreMarks(ctx, s, scope, id, payload); err != nil {
		return 0, false, err
	}
	if payload.StudiedAt != 0 {
		if err := s.ImportBatches().RestoreStudyMark(ctx, scope, id, payload.StudiedAt); err != nil {
			return 0, false, err
		}
	}
	if err := s.Training().RelinkItems(ctx, scope, payload.TrainingItemIDs, id); err != nil {
		return 0, false, err
	}
	if payload.Analysis != nil {
		switch _, err := s.Analyses().Load(ctx, scope, id); {
		case errors.Is(err, storage.ErrNotFound):
			if err := s.Analyses().Save(ctx, scope, id, payload.Analysis); err != nil {
				return 0, false, err
			}
			// Save leaves a new row on the built-in table; a verdict valued
			// with another one must say so again (ADR-0068).
			if payload.MET != 0 {
				if err := s.MatchEquityTables().TagAnalyses(ctx, scope, payload.MET, []int64{id}); err != nil {
					return 0, false, err
				}
			}
		case err != nil:
			return 0, false, err
		}
	}
	// Comments are appended, not replaced: a comment written on the position
	// since the delete is somebody's work too.
	existing := map[string]bool{}
	for c, err := range s.Comments().ByPosition(ctx, scope, id) {
		if err != nil {
			return 0, false, err
		}
		existing[c.Text] = true
	}
	for _, c := range payload.Comments {
		if c.Text == "" || existing[c.Text] {
			continue
		}
		cid, err := s.Comments().AddFrom(storage.WithCarriedComment(ctx, c.Author, c.Signature), scope, id, c.Text, c.Origin)
		if err != nil {
			return 0, false, err
		}
		if err := giveBackUUID(ctx, s, scope, "comment", cid, payload.CommentUUIDs[c.ID]); err != nil {
			return 0, false, err
		}
	}
	return id, false, nil
}

// reinstatePosition stores the position under the id it had, so what named it
// finds it again; a position stored meanwhile under another id is the one kept
// (one per hash). When its id is taken, or was never issued here (a snapshot
// from before ids were kept), it gets a new one.
func reinstatePosition(ctx context.Context, s storage.Stores, scope string, pos *domain.Position) (int64, bool, error) {
	if pos.ID > 0 {
		id, created, err := s.Positions().Reinstate(ctx, scope, pos)
		if !errors.Is(err, storage.ErrConflict) && !errors.Is(err, storage.ErrInvalid) {
			return id, created, err
		}
	}
	return s.Positions().SaveCreated(ctx, scope, pos)
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
	if err := giveBackUUID(ctx, s, scope, "collection", id, payload.UUID); err != nil {
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
	id, err := s.Comments().AddFrom(storage.WithCarriedComment(ctx, payload.Comment.Author, payload.Comment.Signature), scope, payload.Comment.PositionID,
		payload.Comment.Text, payload.Comment.Origin)
	if err != nil {
		return 0, err
	}
	return id, giveBackUUID(ctx, s, scope, "comment", id, payload.UUID)
}

// Rencontre deletes a Rencontre after snapshotting it and its members. Its
// Tournaments are detached, never deleted: each keeps its Direction, its log
// and its tables (ADR-0056).
func Rencontre(ctx context.Context, s storage.Stores, scope string, rencontreID int64) (int64, error) {
	r, err := s.Rencontres().Get(ctx, scope, rencontreID)
	if err != nil {
		return 0, err
	}
	id, err := put(ctx, s, scope, domain.TrashRencontre, r.Name, domain.TrashRencontrePayload{Rencontre: *r})
	if err != nil {
		return 0, err
	}
	if err := s.Rencontres().Delete(ctx, scope, rencontreID); err != nil {
		_ = s.Trash().Discard(ctx, scope, id)
		return 0, err
	}
	return id, nil
}

// ErrRencontreByService refuses restoring a Rencontre here: it moves its
// events into a room, a gesture the direction service makes under the events'
// locks, with the room's checks (service.RestoreFromTrash).
var ErrRencontreByService = errors.New("trash: a Rencontre is restored by the direction service")

// RestoreRencontre recreates the Rencontre of entry with its table settings
// and attaches again the Tournaments that still exist and have not joined
// another Rencontre meanwhile, each with the rooms it had. It leaves the entry
// in the trash: the caller discards it in the same transaction.
func RestoreRencontre(ctx context.Context, s storage.Stores, scope string, entry *domain.TrashEntry) (int64, error) {
	var payload domain.TrashRencontrePayload
	if err := json.Unmarshal(entry.Payload, &payload); err != nil {
		return 0, fmt.Errorf("trash entry %d: %w", entry.ID, err)
	}
	id, err := s.Rencontres().Create(ctx, scope, payload.Rencontre)
	if err != nil {
		return 0, err
	}
	if err := s.Rencontres().SetTableSettings(ctx, scope, id, payload.Rencontre.TableSettings); err != nil {
		return 0, err
	}
	for _, tid := range payload.Rencontre.TournamentIDs {
		if of, err := s.Rencontres().Of(ctx, scope, tid); err != nil || of != 0 {
			continue
		}
		if err := s.Rencontres().Attach(ctx, scope, tid, id); err != nil {
			return 0, err
		}
		if rooms := payload.Rencontre.EventRooms[tid]; len(rooms) > 0 {
			if err := s.Rencontres().SetEventRooms(ctx, scope, tid, rooms); err != nil {
				return 0, err
			}
		}
	}
	return id, nil
}

// put marshals a payload and writes the snapshot.
func put(ctx context.Context, s storage.Stores, scope string, kind domain.TrashKind, label string, payload any) (int64, error) {
	blob, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	return s.Trash().Put(ctx, scope, kind, label, blob)
}

// DiscardByUUID removes the trash entries of one kind whose snapshot holds the
// object uuid: the object is back (or gone for good) by another way, and
// restoring the entry would make a second one under a fresh identifier.
func DiscardByUUID(ctx context.Context, s storage.Stores, scope string, kind domain.TrashKind, uuid string) error {
	if uuid == "" {
		return nil
	}
	entries, err := s.Trash().List(ctx, scope, kind, storage.ListOpts{})
	if err != nil {
		return err
	}
	for _, e := range entries {
		var p struct {
			UUID string `json:"uuid"`
		}
		if json.Unmarshal(e.Payload, &p) == nil && p.UUID == uuid {
			if err := s.Trash().Discard(ctx, scope, e.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// InTx runs fn in one transaction when s can open one, directly on s otherwise.
func InTx(ctx context.Context, s storage.Stores, scope string, fn func(storage.Stores) error) error {
	return inTx(ctx, s, scope, fn)
}
