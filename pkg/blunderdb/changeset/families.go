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
	PublicID string `json:"publicId,omitempty"`
	// Drawing is the whole drawing of the position. DrawingAt dates it: no date
	// says nothing, and a date with no marks is an erasure.
	Drawing   []domain.DrawingMark `json:"drawing,omitempty"`
	DrawingAt string               `json:"drawingAt,omitempty"`
	// DerivedFrom is the link to the origin, which it names by the hash of
	// its board like every cross-reference of a payload. DerivedAt dates it as
	// DrawingAt does the drawing.
	DerivedFrom *derivationPayload `json:"derivedFrom,omitempty"`
	DerivedAt   string             `json:"derivedAt,omitempty"`
}

type derivationPayload struct {
	Position string `json:"position"`
	Move     string `json:"move,omitempty"`
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
		if out.Drawing, out.DrawingAt, err = readDrawing(ctx, s, scope, id); err != nil {
			return nil, err
		}
		if out.DerivedFrom, out.DerivedAt, err = readLink(ctx, s, scope, id); err != nil {
			return nil, err
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
	// Each part keeps its latest fact. The rest is what the update could not
	// take back anyway, so the two sides reach one content.
	merge: func(winner, loser json.RawMessage) (json.RawMessage, error) {
		var w, l positionPayload
		if err := json.Unmarshal(winner, &w); err != nil {
			return nil, fmt.Errorf("position payload: %w", err)
		}
		if err := json.Unmarshal(loser, &l); err != nil {
			return nil, fmt.Errorf("position payload: %w", err)
		}
		w.Position.IndividuallyImported = w.Position.IndividuallyImported || l.Position.IndividuallyImported
		w.Position.Flagged = w.Position.Flagged || l.Position.Flagged
		if w.Analysis == nil {
			w.Analysis = l.Analysis
		}
		if w.PublicID == "" {
			w.PublicID = l.PublicID
		}
		if partBeats(l.DrawingAt, l.Drawing, w.DrawingAt, w.Drawing) {
			w.Drawing, w.DrawingAt = l.Drawing, l.DrawingAt
		}
		if partBeats(l.DerivedAt, l.DerivedFrom, w.DerivedAt, w.DerivedFrom) {
			w.DerivedFrom, w.DerivedAt = l.DerivedFrom, l.DerivedAt
		}
		return json.Marshal(w)
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
	// The peer may simply not have heard of it: an empty one says nothing.
	if ids, ok := s.(storage.PublicIDProvider); ok && p.PublicID != "" {
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
	if err := applyDrawing(ctx, s, scope, id, p); err != nil {
		return err
	}
	if err := applyLink(ctx, s, scope, id, p); err != nil {
		return err
	}
	if p.Analysis != nil {
		return s.Analyses().Save(ctx, scope, id, p.Analysis)
	}
	return nil
}

// readDrawing is a position's drawing and the date of its last fact. A drawing
// with no fact came in with the rows (a file import): it counts as old.
func readDrawing(ctx context.Context, s storage.Stores, scope string, id int64) ([]domain.DrawingMark, string, error) {
	dp, ok := s.(storage.DrawingProvider)
	if !ok {
		return nil, "", nil
	}
	marks, err := dp.PositionDrawings().Drawing(ctx, scope, id)
	if err != nil {
		return nil, "", err
	}
	journal, _, err := providers(s)
	if err != nil {
		return nil, "", err
	}
	f, known, err := journal.PartFact(ctx, scope, id, storage.PartDrawing)
	if err != nil {
		return nil, "", err
	}
	if !known && len(marks) > 0 {
		f.At = floor
	}
	if len(marks) == 0 {
		marks = nil
	}
	return marks, f.At, nil
}

// readLink is a position's link and the date of its last fact: the link waiting
// for its origin when there is one, as that is the last fact known.
func readLink(ctx context.Context, s storage.Stores, scope string, id int64) (*derivationPayload, string, error) {
	dp, ok := s.(storage.DerivationProvider)
	if !ok {
		return nil, "", nil
	}
	journal, _, err := providers(s)
	if err != nil {
		return nil, "", err
	}
	f, known, err := journal.PartFact(ctx, scope, id, storage.PartDerivation)
	if err != nil {
		return nil, "", err
	}
	if f.PendingOrigin != "" {
		return &derivationPayload{Position: f.PendingOrigin, Move: f.PendingMove}, f.At, nil
	}
	d, err := dp.PositionDerivations().DerivedFrom(ctx, scope, id)
	if err != nil || d == nil {
		return nil, f.At, err
	}
	ref, err := refOf(ctx, s, scope, d.FromID)
	if err != nil {
		return nil, "", err
	}
	if !known {
		f.At = floor
	}
	return &derivationPayload{Position: ref, Move: d.Move}, f.At, nil
}

// partBeats says whether a part, dated at, replaces another: the later fact,
// then content over none, then the greater content, the same way on both sides.
// No date and no content says nothing, so it never beats anything.
func partBeats(at string, content any, otherAt string, other any) bool {
	if at != otherAt {
		return at > otherAt
	}
	c, _ := json.Marshal(content)
	o, _ := json.Marshal(other)
	switch cNone, oNone := string(c) == "null", string(o) == "null"; {
	case cNone || oNone:
		return !cNone && oNone
	default:
		return string(c) > string(o)
	}
}

// partDate is the date of a part in a payload, as this library will hold it:
// content with no date is old, and a date ahead of the clock is taken as now.
func partDate(at string, hasContent bool) (string, error) {
	if at == "" {
		if hasContent {
			return floor, nil
		}
		return "", nil
	}
	at, err := normalize(at)
	if err != nil {
		return "", err
	}
	if now := time.Now().UTC().Format(storage.ChangedAtLayout); at > now {
		return now, nil
	}
	return at, nil
}

func applyDrawing(ctx context.Context, s storage.Stores, scope string, id int64, p positionPayload) error {
	dp, ok := s.(storage.DrawingProvider)
	if !ok {
		return nil
	}
	at, err := partDate(p.DrawingAt, len(p.Drawing) > 0)
	if err != nil || at == "" {
		return err
	}
	marks := p.Drawing
	if len(marks) == 0 {
		marks = nil
	}
	have, haveAt, err := readDrawing(ctx, s, scope, id)
	if err != nil || !partBeats(at, marks, haveAt, have) {
		return err
	}
	if err := dp.PositionDrawings().SetDrawing(ctx, scope, id, marks); err != nil {
		return err
	}
	journal, _, err := providers(s)
	if err != nil {
		return err
	}
	return journal.SetPartFact(ctx, scope, storage.PartFact{PositionID: id, Part: storage.PartDrawing, At: at})
}

// waitingKey carries the links of the record being applied whose origin is not
// here yet: Import tells apart, once every record is in, those still waiting.
type waitingKey struct{}

// applyLink makes position id derive from what the payload says. An origin this
// library does not hold leaves a fact waiting for it (see settleLinks); the link
// there was stays until then. A link that would close a loop here is left out
// and reported, and what there was stays.
func applyLink(ctx context.Context, s storage.Stores, scope string, id int64, p positionPayload) error {
	dp, ok := s.(storage.DerivationProvider)
	if !ok {
		return nil
	}
	at, err := partDate(p.DerivedAt, p.DerivedFrom != nil)
	if err != nil || at == "" {
		return err
	}
	have, haveAt, err := readLink(ctx, s, scope, id)
	if err != nil || !partBeats(at, p.DerivedFrom, haveAt, have) {
		return err
	}
	journal, _, err := providers(s)
	if err != nil {
		return err
	}
	fact := storage.PartFact{PositionID: id, Part: storage.PartDerivation, At: at}
	if p.DerivedFrom == nil {
		if err := dp.PositionDerivations().SetDerivedFrom(ctx, scope, id, nil); err != nil {
			return err
		}
		return journal.SetPartFact(ctx, scope, fact)
	}
	origin, found, err := findPosition(ctx, s, scope, p.DerivedFrom.Position)
	if err != nil {
		return err
	}
	if !found {
		fact.PendingOrigin, fact.PendingMove = p.DerivedFrom.Position, p.DerivedFrom.Move
		if list, ok := ctx.Value(waitingKey{}).(*[]waitingLink); ok {
			*list = append(*list, waitingLink{Variant: id, Ref: p.DerivedFrom.Position})
		}
		return journal.SetPartFact(ctx, scope, fact)
	}
	switch err := dp.PositionDerivations().SetDerivedFrom(ctx, scope, id, &domain.Derivation{FromID: origin, Move: p.DerivedFrom.Move}); {
	case errors.Is(err, storage.ErrInvalid):
		noteMissing(ctx, p.DerivedFrom.Position)
		return nil
	case err != nil:
		return err
	}
	return journal.SetPartFact(ctx, scope, fact)
}

type waitingLink struct {
	Family, UUID string
	Variant      int64
	Ref          string
}

// settleLinks makes the links whose origin has arrived, whichever import brought
// it or if the library made it itself. It gives the variants still waiting. A
// link that would close a loop is dropped and reported: the fact then dates what
// is left at the floor, so the link it stays with beats nothing elsewhere.
func settleLinks(ctx context.Context, s storage.Stores, scope string, rep *Report) (map[int64]bool, error) {
	journal, uuids, err := providers(s)
	if err != nil {
		return nil, err
	}
	pending, err := journal.PendingLinks(ctx, scope)
	if err != nil {
		return nil, err
	}
	waiting := map[int64]bool{}
	for _, f := range pending {
		var refused bool
		var settled bool
		err := trash.InTx(ctx, s, scope, func(s storage.Stores) error {
			refused, settled = false, false
			journal, _, err := providers(s)
			if err != nil {
				return err
			}
			// The list was read before this transaction: the link may have
			// been erased or made since, or its position be gone.
			cur, known, err := journal.PartFact(ctx, scope, f.PositionID, storage.PartDerivation)
			if err != nil {
				return err
			}
			if !known || cur.PendingOrigin == "" {
				settled = true
				return nil
			}
			f = cur
			origin, found, err := findPosition(ctx, s, scope, f.PendingOrigin)
			if err != nil || !found {
				return err
			}
			settled = true
			if err := journal.Applying(ctx, scope, f.At); err != nil {
				return err
			}
			kept := storage.PartFact{PositionID: f.PositionID, Part: storage.PartDerivation, At: f.At}
			err = s.(storage.DerivationProvider).PositionDerivations().SetDerivedFrom(ctx, scope, f.PositionID, &domain.Derivation{FromID: origin, Move: f.PendingMove})
			switch {
			case errors.Is(err, storage.ErrInvalid):
				refused, kept.At = true, floor
			case err != nil:
				return err
			}
			if err := journal.SetPartFact(ctx, scope, kept); err != nil {
				return err
			}
			return journal.Applying(ctx, scope, "")
		})
		if err != nil {
			return nil, err
		}
		if !settled {
			waiting[f.PositionID] = true
		}
		if refused {
			u, err := uuids.ObjectUUID(ctx, scope, "position", f.PositionID)
			if err != nil {
				return nil, err
			}
			rep.Missing = append(rep.Missing, MissingRef{Family: "position", UUID: u, Ref: f.PendingOrigin})
		}
	}
	return waiting, nil
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
		e, err := s.Comments().Get(ctx, scope, id)
		if err != nil {
			return err
		}
		if e.Text != p.Text {
			if err := trash.KeepCommentVersion(ctx, s, scope, e); err != nil {
				return err
			}
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
	// A match can be rewritten whole (a transcription saved again, the seats
	// swapped): what the games say and what the header says stay one thing. A
	// header whose hash another local match holds is refused, whole.
	update: func(ctx context.Context, s storage.Stores, scope string, id int64, raw json.RawMessage) error {
		var p matchPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("match payload: %w", err)
		}
		m := p.Match
		cur, err := trash.SnapshotMatch(ctx, s, scope, id)
		if err != nil {
			return err
		}
		portableMatch(cur)
		if err := hashesFree(ctx, s, scope, id, cur.Match, m); err != nil {
			return err
		}
		same, err := sameBody(cur, &p)
		if err != nil {
			return err
		}
		if !same {
			if err := trash.RewriteMatchBody(ctx, s, scope, id, &p); err != nil {
				return err
			}
		}
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

// hashesFree refuses a header whose new hash is another local match's.
func hashesFree(ctx context.Context, s storage.Stores, scope string, id int64, have, want domain.Match) error {
	check := func(hash, canonical string) error {
		other, found, err := s.Matches().FindByHash(ctx, scope, hash, canonical)
		if err != nil {
			return err
		}
		if found && other != id {
			return storage.ErrConflict
		}
		return nil
	}
	if want.MatchHash != "" && want.MatchHash != have.MatchHash {
		if err := check(want.MatchHash, ""); err != nil {
			return err
		}
	}
	if want.CanonicalHash != "" && want.CanonicalHash != have.CanonicalHash {
		return check("", want.CanonicalHash)
	}
	return nil
}

// sameBody says two portable snapshots hold the same games, moves, analyses and
// boards. A position is compared by hash only: the rest of its row is the
// position family's to carry.
func sameBody(a, b *matchPayload) (bool, error) {
	digest := func(p *matchPayload) (string, error) {
		boards := make([]uint64, len(p.Positions))
		for i := range p.Positions {
			boards[i] = storage.ZobristOf(&p.Positions[i].Position)
		}
		raw, err := json.Marshal([]any{boards, p.Games, p.Moves, p.MoveAnalyses})
		return string(raw), err
	}
	da, err := digest(a)
	if err != nil {
		return false, err
	}
	db, err := digest(b)
	return da == db, err
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
			} else {
				noteMissing(ctx, st.Collection)
			}
		}
		if st.Position != "" {
			pid, found, err := findPosition(ctx, s, scope, st.Position)
			if err != nil {
				return err
			}
			if found {
				step.PositionID = pid
			} else {
				noteMissing(ctx, st.Position)
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
