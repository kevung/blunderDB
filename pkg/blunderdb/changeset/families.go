package changeset

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/trash"
)

// A position is named, in a payload, by the hash of its board: the one thing
// two libraries agree on whatever uuid each gave the row (the position family
// reconciles those). It is also what keeps a payload equal on both sides.

func uuidsOf(s storage.Stores) storage.ObjectUUIDStore {
	return s.(storage.ObjectUUIDProvider).ObjectUUIDs()
}

func refOf(ctx context.Context, s storage.Stores, scope string, id int64) (string, error) {
	p, err := s.Positions().Load(ctx, scope, id)
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(storage.ZobristOf(p), 10), nil
}

// findPosition returns the position a hash names here.
func findPosition(ctx context.Context, s storage.Stores, scope, ref string) (int64, bool, error) {
	z, err := strconv.ParseUint(ref, 10, 64)
	if err != nil {
		return 0, false, nil
	}
	return s.Positions().Exists(ctx, scope, z)
}

// --- position ---

type positionPayload struct {
	Position domain.Position          `json:"position"`
	Analysis *domain.PositionAnalysis `json:"analysis,omitempty"`
	// PublicID is opaque: carried and restored as given.
	PublicID string `json:"public_id,omitempty"`
}

var positionCodec = codec{
	rank:      0,
	trashKind: domain.TrashPosition,
	read: func(ctx context.Context, s storage.Stores, scope string, id int64) (any, error) {
		p, err := s.Positions().Load(ctx, scope, id)
		if err != nil {
			return nil, err
		}
		out := positionPayload{Position: *p}
		out.Position.ID = 0
		if ids, ok := s.(storage.PublicIDProvider); ok {
			if out.PublicID, err = ids.PublicIDs().PublicID(ctx, scope, id); err != nil {
				return nil, err
			}
		}
		switch a, err := s.Analyses().Load(ctx, scope, id); {
		case err == nil:
			out.Analysis = a
		case !errors.Is(err, storage.ErrNotFound):
			return nil, err
		}
		return out, nil
	},
	create: func(ctx context.Context, s storage.Stores, scope string, raw json.RawMessage) (int64, error) {
		var p positionPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return 0, fmt.Errorf("position payload: %w", err)
		}
		pos := p.Position
		pos.ID = 0
		id, _, err := s.Positions().SaveCreated(ctx, scope, &pos)
		if err != nil {
			return 0, err
		}
		return id, applyPosition(ctx, s, scope, id, p)
	},
	update: func(ctx context.Context, s storage.Stores, scope string, id int64, raw json.RawMessage) error {
		var p positionPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("position payload: %w", err)
		}
		return applyPosition(ctx, s, scope, id, p)
	},
	remove: func(ctx context.Context, s storage.Stores, scope string, id int64, toTrash bool) error {
		if toTrash {
			_, err := trash.Position(ctx, s, scope, id)
			return err
		}
		return s.Positions().Delete(ctx, scope, id)
	},
	// The same board under two uuids (two libraries named it before they met):
	// the older uuid, which sorts first, is kept on both sides. Whatever
	// points at the position does so by id here, so the notes and the
	// collection memberships follow the rename.
	locate: func(ctx context.Context, s storage.Stores, scope string, r Record) (int64, string, bool, error) {
		var p positionPayload
		if err := json.Unmarshal(r.Payload, &p); err != nil || len(r.Payload) == 0 {
			return 0, "", false, nil
		}
		pos := p.Position
		id, found, err := s.Positions().Exists(ctx, scope, storage.ZobristOf(&pos))
		if err != nil || !found {
			return 0, "", false, err
		}
		have, err := uuidsOf(s).ObjectUUID(ctx, scope, "position", id)
		if err != nil {
			return 0, "", false, err
		}
		if have == "" || r.UUID >= have {
			return id, have, have != "", nil
		}
		journal, _, err := providers(s)
		if err != nil {
			return 0, "", false, err
		}
		stamp, serr := journal.Entry(ctx, scope, have)
		if err := uuidsOf(s).SetObjectUUID(ctx, scope, "position", id, r.UUID); err != nil {
			return 0, "", false, err
		}
		// The rename rewrites the row: it is no edit, and must not look newer
		// than what the peer holds.
		op, at := storage.ChangeCreate, floor
		if serr == nil {
			op, at = stamp.Op, stamp.ChangedAt
		}
		if err := journal.Mark(ctx, scope, "position", r.UUID, op, at); err != nil {
			return 0, "", false, err
		}
		return id, r.UUID, true, nil
	},
}

func applyPosition(ctx context.Context, s storage.Stores, scope string, id int64, p positionPayload) error {
	if ids, ok := s.(storage.PublicIDProvider); ok {
		if err := ids.PublicIDs().SetPublicID(ctx, scope, id, p.PublicID); err != nil {
			return err
		}
	}
	if p.Position.IndividuallyImported {
		if err := s.Positions().SetIndividuallyImported(ctx, scope, id, true); err != nil {
			return err
		}
	}
	if p.Position.Flagged {
		pos := p.Position
		if _, err := s.Positions().RaiseFlag(ctx, scope, &pos); err != nil {
			return err
		}
	}
	if p.Analysis != nil {
		return s.Analyses().Save(ctx, scope, id, p.Analysis)
	}
	return nil
}

// --- comment ---

// commentPayload carries the board it is written on: a note must not wait for
// its position to arrive in a later set.
type commentPayload struct {
	Position  string          `json:"position"`
	Board     domain.Position `json:"board"`
	Text      string          `json:"text"`
	Author    string          `json:"author,omitempty"`
	Origin    string          `json:"origin,omitempty"`
	Signature string          `json:"signature,omitempty"`
}

var commentCodec = codec{
	rank:      3,
	trashKind: domain.TrashComment,
	read: func(ctx context.Context, s storage.Stores, scope string, id int64) (any, error) {
		e, err := s.Comments().Get(ctx, scope, id)
		if err != nil {
			return nil, err
		}
		ref, err := refOf(ctx, s, scope, e.PositionID)
		if err != nil {
			return nil, err
		}
		pos, err := s.Positions().Load(ctx, scope, e.PositionID)
		if err != nil {
			return nil, err
		}
		board := *pos
		board.ID = 0
		return commentPayload{Position: ref, Board: board, Text: e.Text, Author: e.Author,
			Origin: string(e.Origin), Signature: e.Signature}, nil
	},
	create: func(ctx context.Context, s storage.Stores, scope string, raw json.RawMessage) (int64, error) {
		var p commentPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return 0, fmt.Errorf("comment payload: %w", err)
		}
		pid, found, err := findPosition(ctx, s, scope, p.Position)
		if err != nil {
			return 0, err
		}
		if !found {
			board := p.Board
			board.ID = 0
			if pid, _, err = s.Positions().SaveCreated(ctx, scope, &board); err != nil {
				return 0, err
			}
		}
		return s.Comments().AddFrom(storage.WithCarriedComment(ctx, p.Author, p.Signature), scope, pid, p.Text, domain.ParseCommentOrigin(p.Origin))
	},
	update: func(ctx context.Context, s storage.Stores, scope string, id int64, raw json.RawMessage) error {
		var p commentPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("comment payload: %w", err)
		}
		return s.Comments().Update(storage.WithCarriedComment(ctx, p.Author, p.Signature), scope, id, p.Text)
	},
	remove: func(ctx context.Context, s storage.Stores, scope string, id int64, toTrash bool) error {
		if toTrash {
			_, err := trash.CommentEntry(ctx, s, scope, id)
			return err
		}
		return s.Comments().Delete(ctx, scope, id)
	},
}

// --- match ---

// matchPayload is a match whole, with every id replaced by its rank, so two
// libraries holding the same match read the same bytes. What the match is
// attached to here (tournament, import batch, direction slot, drafts, the
// duel that produced it) and where the user last looked belong to the library
// and stay out.
type matchPayload = domain.TrashMatchPayload

func portableMatch(p *domain.TrashMatchPayload) {
	p.UUID, p.TournamentIndex, p.Origin = "", 0, nil
	p.DirectionSlot, p.DirectionPair, p.TranscriptionIDs, p.ImportFileIDs = "", [2]string{}, nil, nil
	p.Match.ID, p.Match.ImportDate, p.Match.TournamentID, p.Match.ImportBatchID = 0, time.Time{}, nil, 0
	p.Match.LastVisitedPosition = -1
	pos := map[int64]int64{}
	for i := range p.Positions {
		pp := &p.Positions[i]
		pos[pp.Position.ID] = int64(i + 1)
		*pp = domain.TrashPositionPayload{Position: pp.Position}
		pp.Position.ID = int64(i + 1)
	}
	games := map[int64]int64{}
	for i := range p.Games {
		games[p.Games[i].ID] = int64(i + 1)
		p.Games[i].ID, p.Games[i].MatchID = int64(i+1), 0
	}
	moves := map[int64]int64{}
	for i := range p.Moves {
		mv := &p.Moves[i]
		moves[mv.ID] = int64(i + 1)
		mv.ID, mv.GameID = int64(i+1), games[mv.GameID]
		if mv.PositionID != 0 {
			mv.PositionID = pos[mv.PositionID]
		}
	}
	for i := range p.MoveAnalyses {
		ma := &p.MoveAnalyses[i]
		ma.ID, ma.MoveID = int64(i+1), moves[ma.MoveID]
	}
}

var matchCodec = codec{
	rank:      1,
	trashKind: domain.TrashMatch,
	read: func(ctx context.Context, s storage.Stores, scope string, id int64) (any, error) {
		p, err := trash.SnapshotMatch(ctx, s, scope, id)
		if err != nil {
			return nil, err
		}
		portableMatch(p)
		return p, nil
	},
	create: func(ctx context.Context, s storage.Stores, scope string, raw json.RawMessage) (int64, error) {
		var p matchPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return 0, fmt.Errorf("match payload: %w", err)
		}
		return trash.RebuildMatchAs(ctx, s, scope, &p)
	},
	// Games and moves of a match do not change once stored; the header, the
	// note and the video do.
	update: func(ctx context.Context, s storage.Stores, scope string, id int64, raw json.RawMessage) error {
		var p matchPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("match payload: %w", err)
		}
		m := p.Match
		if err := s.Matches().ReplaceHeader(ctx, scope, id, &m); err != nil {
			return err
		}
		if err := s.Matches().UpdateComment(ctx, scope, id, m.Comment); err != nil {
			return err
		}
		if m.VideoSource != nil {
			return s.Matches().SetVideoSource(ctx, scope, id, *m.VideoSource)
		}
		return nil
	},
	remove: func(ctx context.Context, s storage.Stores, scope string, id int64, toTrash bool) error {
		if toTrash {
			_, err := trash.Match(ctx, s, scope, id)
			return err
		}
		return s.Matches().DeleteCascade(ctx, scope, id)
	},
}

// --- lesson ---

type lessonStep struct {
	Title      string `json:"title"`
	Text       string `json:"text"`
	Collection string `json:"collection,omitempty"`
	Position   string `json:"position,omitempty"`
}

type lessonPayload struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Steps       []lessonStep `json:"steps,omitempty"`
}

func readLesson(ctx context.Context, s storage.Stores, scope string, id int64) (lessonPayload, error) {
	l, err := s.Lessons().Get(ctx, scope, id)
	if err != nil {
		return lessonPayload{}, err
	}
	out := lessonPayload{Name: l.Name, Description: l.Description}
	for _, st := range l.Steps {
		step := lessonStep{Title: st.Title, Text: st.Text}
		if st.CollectionID != 0 {
			if step.Collection, err = uuidsOf(s).ObjectUUID(ctx, scope, "collection", st.CollectionID); err != nil {
				return lessonPayload{}, err
			}
		}
		if st.PositionID != 0 {
			ref, err := refOf(ctx, s, scope, st.PositionID)
			if err != nil {
				return lessonPayload{}, err
			}
			step.Position = ref
		}
		out.Steps = append(out.Steps, step)
	}
	return out, nil
}

// fillSteps brings the lesson's steps to the payload's, leaving them (and the
// progress recorded on them) alone when they already read the same. A step
// names a collection or a position this library lacks as a text step.
func fillSteps(ctx context.Context, s storage.Stores, scope string, id int64, p lessonPayload) error {
	have, err := readLesson(ctx, s, scope, id)
	if err != nil {
		return err
	}
	if reflect.DeepEqual(have.Steps, p.Steps) {
		return nil
	}
	l, err := s.Lessons().Get(ctx, scope, id)
	if err != nil {
		return err
	}
	for _, st := range l.Steps {
		if err := s.Lessons().RemoveStep(ctx, scope, st.ID); err != nil {
			return err
		}
	}
	for _, st := range p.Steps {
		step := domain.LessonStep{Title: st.Title, Text: st.Text}
		if st.Collection != "" {
			if cid, err := uuidsOf(s).ObjectID(ctx, scope, "collection", st.Collection); err == nil {
				step.CollectionID = cid
			} else if !errors.Is(err, storage.ErrNotFound) {
				return err
			}
		}
		if st.Position != "" {
			pid, found, err := findPosition(ctx, s, scope, st.Position)
			if err != nil {
				return err
			}
			if found {
				step.PositionID = pid
			}
		}
		if _, err := s.Lessons().AddStep(ctx, scope, id, step); err != nil {
			return err
		}
	}
	return nil
}

var lessonCodec = codec{
	rank: 4,
	read: func(ctx context.Context, s storage.Stores, scope string, id int64) (any, error) {
		return readLesson(ctx, s, scope, id)
	},
	create: func(ctx context.Context, s storage.Stores, scope string, raw json.RawMessage) (int64, error) {
		var p lessonPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return 0, fmt.Errorf("lesson payload: %w", err)
		}
		id, err := s.Lessons().Create(ctx, scope, p.Name, p.Description)
		if err != nil {
			return 0, err
		}
		return id, fillSteps(ctx, s, scope, id, p)
	},
	update: func(ctx context.Context, s storage.Stores, scope string, id int64, raw json.RawMessage) error {
		var p lessonPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("lesson payload: %w", err)
		}
		if err := s.Lessons().Update(ctx, scope, id, p.Name, p.Description); err != nil {
			return err
		}
		return fillSteps(ctx, s, scope, id, p)
	},
	// A lesson has no trash: a trash from a peer is a delete here.
	remove: func(ctx context.Context, s storage.Stores, scope string, id int64, _ bool) error {
		return s.Lessons().Delete(ctx, scope, id)
	},
}
