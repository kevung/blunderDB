package trash

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	tournoi "github.com/kevung/nicomaque"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ErrMatchPresent refuses restoring a match the database holds again — it was
// re-imported since the delete, so a restore would make it a duplicate — or
// whose id another match now holds.
var ErrMatchPresent = errors.New("trash: the match is in the database again")

// Match deletes a match after snapshotting it whole: its header, games,
// moves, move analyses, and every position its moves reached with their
// analyses and comments. The delete is MatchStore.DeleteCascade, so the
// positions nothing else holds are purged by the same retention rule as a
// plain delete; the snapshot is what lets them come back.
func Match(ctx context.Context, s storage.Stores, scope string, matchID int64) (int64, error) {
	var id int64
	err := inTx(ctx, s, scope, func(s storage.Stores) error {
		payload, err := snapshotMatch(ctx, s, scope, matchID)
		if err != nil {
			return err
		}
		if id, err = put(ctx, s, scope, domain.TrashMatch, matchLabel(&payload.Match), payload); err != nil {
			return err
		}
		if err := s.Matches().DeleteCascade(ctx, scope, matchID); err != nil {
			// Outside a transaction nothing rolls the snapshot back: drop it
			// rather than leave an entry that would restore a duplicate.
			_ = s.Trash().Discard(ctx, scope, id)
			return err
		}
		return markMatchTrashed(ctx, s, scope, payload)
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

// markMatchTrashed tells the journal that the delete went through the trash:
// for the match, for each position the cascade purged and for their notes.
// A position another match or a collection still holds was not deleted.
func markMatchTrashed(ctx context.Context, s storage.Stores, scope string, p *domain.TrashMatchPayload) error {
	if err := markTrashed(ctx, s, scope, "match", p.UUID); err != nil {
		return err
	}
	uu, ok := s.(storage.ObjectUUIDProvider)
	if !ok {
		return nil
	}
	for _, pp := range p.Positions {
		if pp.UUID == "" {
			continue
		}
		if _, err := uu.ObjectUUIDs().ObjectID(ctx, scope, "position", pp.UUID); err == nil {
			continue
		} else if !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		if err := markTrashed(ctx, s, scope, "position", pp.UUID); err != nil {
			return err
		}
		for _, u := range pp.CommentUUIDs {
			if err := markTrashed(ctx, s, scope, "comment", u); err != nil {
				return err
			}
		}
	}
	return nil
}

func matchLabel(m *domain.Match) string {
	label := m.Player1Name + " – " + m.Player2Name
	if !m.MatchDate.IsZero() {
		label += ", " + m.MatchDate.Format("2006-01-02")
	}
	return label
}

func snapshotMatch(ctx context.Context, s storage.Stores, scope string, matchID int64) (*domain.TrashMatchPayload, error) {
	ms := s.Matches()
	m, err := ms.Get(ctx, scope, matchID)
	if err != nil {
		return nil, err
	}
	p := &domain.TrashMatchPayload{Match: *m}
	if p.UUID, err = uuidOf(ctx, s, scope, "match", matchID); err != nil {
		return nil, err
	}
	for g, err := range ms.Games(ctx, scope, matchID) {
		if err != nil {
			return nil, err
		}
		p.Games = append(p.Games, *g)
	}
	for mv, err := range ms.MovesByMatch(ctx, scope, matchID) {
		if err != nil {
			return nil, err
		}
		p.Moves = append(p.Moves, *mv)
	}
	// Outside the listing: in a transaction, or on a store pinned to one
	// connection, a query cannot run while another holds its rows.
	seen := map[int64]bool{}
	for _, mv := range p.Moves {
		if mv.PositionID != 0 && !seen[mv.PositionID] {
			seen[mv.PositionID] = true
			pos, err := snapshotPosition(ctx, s, scope, mv.PositionID)
			if err != nil {
				return nil, fmt.Errorf("position %d of match %d: %w", mv.PositionID, matchID, err)
			}
			p.Positions = append(p.Positions, pos)
		}
	}
	for ma, err := range ms.MoveAnalysesByMatch(ctx, scope, matchID) {
		if err != nil {
			return nil, err
		}
		p.MoveAnalyses = append(p.MoveAnalyses, *ma)
	}
	if m.TournamentID != nil {
		i := 0
		for tm, err := range s.Tournaments().Matches(ctx, scope, *m.TournamentID) {
			if err != nil {
				return nil, err
			}
			if tm.ID == matchID {
				p.TournamentIndex = i
				break
			}
			i++
		}
	}
	slotTournament, slot, err := s.Directions().SlotOf(ctx, scope, matchID)
	if err != nil {
		return nil, err
	}
	if slot != "" {
		p.DirectionSlot = slot
		if p.DirectionPair, _, err = slotPair(ctx, s, scope, slotTournament, slot); err != nil {
			return nil, err
		}
	}
	switch o, err := s.Duels().Origin(ctx, scope, matchID); {
	case err == nil:
		if p.Origin, err = json.Marshal(o); err != nil {
			return nil, err
		}
	case !errors.Is(err, storage.ErrNotFound):
		return nil, err
	}
	for t, err := range s.Transcriptions().List(ctx, scope) {
		if err != nil {
			return nil, err
		}
		if t.MatchID == matchID {
			p.TranscriptionIDs = append(p.TranscriptionIDs, t.ID)
		}
	}
	if p.ImportFileIDs, err = s.ImportBatches().FilesOfMatch(ctx, scope, matchID); err != nil {
		return nil, err
	}
	return p, nil
}

// restoreMatch rebuilds a match from its snapshot under its old id and import
// date, in one transaction when the store offers one: a half-restored match
// would be worse than none. Games, moves and analyses get new ids: nothing
// outside the match's own cascade names them.
//
// The positions are re-Saved first, so the moves point at whatever row now
// holds each one. Only a position the delete purged gets its analysis and
// comments back; one that stayed held kept them, edits included. What the match was attached to — its tournament,
// its import batch, the drafts that produced it — is attached again only
// where it still exists, and a Tournament gone since is warned of. Its
// Direction Slot is the director's: taken, gone or re-paired since, the match
// comes back without it, and the result warns.
func restoreMatch(ctx context.Context, s storage.Stores, scope string, entry *domain.TrashEntry) (domain.TrashRestore, error) {
	var p domain.TrashMatchPayload
	if err := json.Unmarshal(entry.Payload, &p); err != nil {
		return domain.TrashRestore{}, fmt.Errorf("trash entry %d: %w", entry.ID, err)
	}
	var res domain.TrashRestore
	err := inTx(ctx, s, scope, func(s storage.Stores) error {
		id, warnings, err := rebuildMatch(ctx, s, scope, &p)
		if err != nil {
			return err
		}
		res = domain.TrashRestore{ID: id, Warnings: warnings}
		return s.Trash().Discard(ctx, scope, entry.ID)
	})
	if err != nil {
		return domain.TrashRestore{}, err
	}
	return res, nil
}

func rebuildMatch(ctx context.Context, s storage.Stores, scope string, p *domain.TrashMatchPayload) (int64, []domain.TrashWarning, error) {
	ms := s.Matches()
	if p.Match.MatchHash != "" || p.Match.CanonicalHash != "" {
		other, found, err := ms.FindByHash(ctx, scope, p.Match.MatchHash, p.Match.CanonicalHash)
		if err != nil {
			return 0, nil, err
		}
		if found {
			return 0, nil, fmt.Errorf("%w (match %d)", ErrMatchPresent, other)
		}
	}

	positions := make(map[int64]int64, len(p.Positions))
	for _, pp := range p.Positions {
		newID, err := restorePositionPayload(ctx, s, scope, pp, true)
		if err != nil {
			return 0, nil, err
		}
		positions[pp.Position.ID] = newID
	}

	var warnings []domain.TrashWarning
	m := p.Match
	if m.TournamentID != nil {
		if _, err := s.Tournaments().Get(ctx, scope, *m.TournamentID); err != nil {
			if !errors.Is(err, storage.ErrNotFound) {
				return 0, nil, err
			}
			warnings = append(warnings, domain.TrashWarning{Code: domain.TrashWarnTournamentGone,
				Message: fmt.Sprintf("tournament %d was deleted; the match is restored outside any tournament", *m.TournamentID)})
			m.TournamentID = nil
		}
	}
	if m.ImportBatchID != 0 {
		if _, err := s.ImportBatches().Load(ctx, scope, m.ImportBatchID); err != nil {
			if !errors.Is(err, storage.ErrNotFound) {
				return 0, nil, err
			}
			m.ImportBatchID = 0
		}
	}
	// Under its own id and import date: whatever named the match by id — a
	// link, a review, a client's cache — finds it again.
	if err := ms.Reinstate(ctx, scope, &m); err != nil {
		if errors.Is(err, storage.ErrConflict) {
			return 0, nil, fmt.Errorf("%w: its id %d is taken", ErrMatchPresent, m.ID)
		}
		return 0, nil, err
	}
	matchID := m.ID
	if err := giveBackUUID(ctx, s, scope, "match", matchID, p.UUID); err != nil {
		return 0, nil, err
	}
	if m.VideoSource != nil {
		if err := ms.SetVideoSource(ctx, scope, matchID, *m.VideoSource); err != nil {
			return 0, nil, err
		}
	}
	// Always written: 0 is the first position, and the column's own default
	// (-1, never visited) is a value the snapshot carries as well.
	if err := ms.SetLastVisitedPosition(ctx, scope, matchID, m.LastVisitedPosition); err != nil {
		return 0, nil, err
	}

	games := make(map[int64]int64, len(p.Games))
	for _, g := range p.Games {
		old := g.ID
		g.ID, g.MatchID = 0, matchID
		newID, err := ms.CreateGame(ctx, scope, &g)
		if err != nil {
			return 0, nil, err
		}
		games[old] = newID
	}
	moves := make(map[int64]int64, len(p.Moves))
	for _, mv := range p.Moves {
		gameID, ok := games[mv.GameID]
		if !ok {
			return 0, nil, fmt.Errorf("trash: move %d names game %d, absent from the snapshot", mv.ID, mv.GameID)
		}
		old := mv.ID
		mv.ID, mv.GameID = 0, gameID
		if mv.PositionID != 0 {
			mv.PositionID = positions[mv.PositionID]
		}
		newID, err := ms.CreateMove(ctx, scope, &mv)
		if err != nil {
			return 0, nil, err
		}
		moves[old] = newID
	}
	for _, ma := range p.MoveAnalyses {
		moveID, ok := moves[ma.MoveID]
		if !ok {
			return 0, nil, fmt.Errorf("trash: analysis %d names move %d, absent from the snapshot", ma.ID, ma.MoveID)
		}
		ma.ID, ma.MoveID = 0, moveID
		if _, err := ms.CreateMoveAnalysis(ctx, scope, &ma); err != nil {
			return 0, nil, err
		}
	}

	if len(p.Origin) > 0 {
		var o storage.MatchOrigin
		if err := json.Unmarshal(p.Origin, &o); err != nil {
			return 0, nil, err
		}
		o.MatchID = matchID
		if err := s.Duels().SetOrigin(ctx, scope, &o); err != nil {
			return 0, nil, err
		}
	}
	if p.DirectionSlot != "" && m.TournamentID != nil {
		code, err := refillSlot(ctx, s, scope, *m.TournamentID, p.DirectionSlot, p.DirectionPair, matchID)
		if err != nil {
			return 0, nil, err
		}
		switch code {
		case domain.TrashWarnSlotTaken:
			warnings = append(warnings, domain.TrashWarning{Code: code,
				Message: fmt.Sprintf("slot %s of tournament %d is filled by another match; the match is restored without it",
					p.DirectionSlot, *m.TournamentID)})
		case domain.TrashWarnSlotGone:
			warnings = append(warnings, domain.TrashWarning{Code: code,
				Message: fmt.Sprintf("slot %s of tournament %d no longer pairs these players; the match is restored without it",
					p.DirectionSlot, *m.TournamentID)})
		}
	}
	if m.TournamentID != nil {
		if err := placeInTournament(ctx, s, scope, *m.TournamentID, matchID, p.TournamentIndex); err != nil {
			return 0, nil, err
		}
	}
	for _, tid := range p.TranscriptionIDs {
		t, err := s.Transcriptions().Get(ctx, scope, tid)
		if errors.Is(err, storage.ErrNotFound) {
			continue
		}
		if err != nil {
			return 0, nil, err
		}
		// A draft that has produced another match since keeps that one.
		if t.MatchID != 0 {
			continue
		}
		t.MatchID = matchID
		if _, err := s.Transcriptions().Save(ctx, scope, t); err != nil {
			return 0, nil, err
		}
	}
	if err := s.ImportBatches().RelinkFiles(ctx, scope, p.ImportFileIDs, matchID); err != nil {
		return 0, nil, err
	}
	// The per-seat tallies, in the same transaction, as an import writes
	// them: a reader never meets a restored match without its stats.
	if err := s.Stats().RefreshMatchStats(ctx, scope, []int64{matchID}); err != nil {
		return 0, nil, err
	}
	return matchID, warnings, nil
}

// refillSlot puts the match back in the Slot of a directed Tournament it
// filled, when the Slot is still in the Direction, pairs the same players
// (pair, when the snapshot recorded it) and no other match fills it: the Slot
// is the director's. It returns "" when the match has its Slot again, the
// warning code saying why not otherwise.
func refillSlot(ctx context.Context, s storage.Stores, scope string, tournamentID int64, slotID string, pair [2]string, matchID int64) (string, error) {
	current, ok, err := slotPair(ctx, s, scope, tournamentID, slotID)
	if err != nil {
		return "", err
	}
	if !ok || (pair != [2]string{} && current != pair) {
		return domain.TrashWarnSlotGone, nil
	}
	filled, err := s.Directions().FilledSlots(ctx, scope, tournamentID)
	if err != nil {
		return "", err
	}
	for _, f := range filled {
		if f.SlotID == slotID {
			return domain.TrashWarnSlotTaken, nil
		}
	}
	return "", s.Directions().AttachSlot(ctx, scope, tournamentID, slotID, matchID)
}

// slotPair reads the two players a Slot of a directed Tournament pairs, by
// their Direction ids, replaying the Direction as the direction service does.
// It reports false when the Tournament is no longer directed or its Direction
// has no such Slot.
func slotPair(ctx context.Context, s storage.Stores, scope string, tournamentID int64, slotID string) ([2]string, bool, error) {
	dir, err := direction.Open(ctx, storage.BindDirection(s.Directions(), scope), tournamentID)
	if errors.Is(err, direction.ErrNoDirection) {
		return [2]string{}, false, nil
	}
	if err != nil {
		return [2]string{}, false, err
	}
	st := dir.State()
	if st == nil {
		return [2]string{}, false, nil
	}
	m := st.Matches[tournoi.MatchID(slotID)]
	if m == nil {
		return [2]string{}, false, nil
	}
	return [2]string{string(m.A), string(m.B)}, true, nil
}

// placeInTournament puts matchID back at index among its tournament's matches.
func placeInTournament(ctx context.Context, s storage.Stores, scope string, tournamentID, matchID int64, index int) error {
	var ids []int64
	for tm, err := range s.Tournaments().Matches(ctx, scope, tournamentID) {
		if err != nil {
			return err
		}
		if tm.ID != matchID {
			ids = append(ids, tm.ID)
		}
	}
	ids = slices.Insert(ids, min(max(index, 0), len(ids)), matchID)
	return s.Tournaments().ReorderMatches(ctx, scope, tournamentID, ids)
}

// txBeginner is a store that can open a transaction; a storage.Tx cannot, and
// a function handed one runs inside the transaction it already is.
type txBeginner interface {
	BeginTx(ctx context.Context) (storage.Tx, error)
}

// inTx runs fn in a transaction when s can open one, directly on s otherwise.
// The transaction is guarded when the backend can guard it: fn reads, then
// writes, and a deferred SQLite transaction that read first would fail with
// SQLITE_BUSY at its first write when another connection wrote meanwhile.
func inTx(ctx context.Context, s storage.Stores, scope string, fn func(storage.Stores) error) error {
	b, ok := s.(txBeginner)
	if !ok {
		return fn(s)
	}
	var tx storage.Tx
	var err error
	if g, ok := s.(storage.GuardedBeginner); ok {
		tx, err = g.BeginGuardedTx(ctx, "trash:"+scope)
	} else {
		tx, err = b.BeginTx(ctx)
	}
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// SnapshotMatch reads a match whole, as Match does before deleting it, without
// deleting it.
func SnapshotMatch(ctx context.Context, s storage.Stores, scope string, matchID int64) (*domain.TrashMatchPayload, error) {
	return snapshotMatch(ctx, s, scope, matchID)
}

// RebuildMatchAs stores a snapshot as a new match under a fresh id, in the
// caller's transaction: what a library receives from another one, where the
// ids of the snapshot name nothing. Its positions are saved by hash (each
// becomes whichever row holds it) and keep neither analysis nor notes; its
// tournament, import batch, direction slot, drafts and origin are the
// sender's and are left out. ErrConflict when a match with the same hashes is
// held. The positions it creates are named by the library; the caller gives
// the match its uuid.
func RebuildMatchAs(ctx context.Context, s storage.Stores, scope string, p *domain.TrashMatchPayload) (int64, error) {
	ms := s.Matches()
	m := p.Match
	if m.MatchHash != "" || m.CanonicalHash != "" {
		if _, found, err := ms.FindByHash(ctx, scope, m.MatchHash, m.CanonicalHash); err != nil {
			return 0, err
		} else if found {
			return 0, storage.ErrConflict
		}
	}
	positions := make(map[int64]int64, len(p.Positions))
	for _, pp := range p.Positions {
		pos := pp.Position
		old := pos.ID
		pos.ID = 0
		id, _, err := s.Positions().SaveCreated(ctx, scope, &pos)
		if err != nil {
			return 0, err
		}
		positions[old] = id
	}
	m.ID, m.TournamentID, m.ImportBatchID = 0, nil, 0
	matchID, err := ms.Save(ctx, scope, &m)
	if err != nil {
		return 0, err
	}
	if m.VideoSource != nil {
		if err := ms.SetVideoSource(ctx, scope, matchID, *m.VideoSource); err != nil {
			return 0, err
		}
	}
	if m.Comment != "" {
		if err := ms.UpdateComment(ctx, scope, matchID, m.Comment); err != nil {
			return 0, err
		}
	}
	games := make(map[int64]int64, len(p.Games))
	for _, g := range p.Games {
		old := g.ID
		g.ID, g.MatchID = 0, matchID
		id, err := ms.CreateGame(ctx, scope, &g)
		if err != nil {
			return 0, err
		}
		games[old] = id
	}
	moves := make(map[int64]int64, len(p.Moves))
	for _, mv := range p.Moves {
		gameID, ok := games[mv.GameID]
		if !ok {
			return 0, fmt.Errorf("match snapshot: move %d names game %d, absent: %w", mv.ID, mv.GameID, storage.ErrInvalid)
		}
		old := mv.ID
		mv.ID, mv.GameID = 0, gameID
		if mv.PositionID != 0 {
			mv.PositionID = positions[mv.PositionID]
		}
		id, err := ms.CreateMove(ctx, scope, &mv)
		if err != nil {
			return 0, err
		}
		moves[old] = id
	}
	for _, ma := range p.MoveAnalyses {
		moveID, ok := moves[ma.MoveID]
		if !ok {
			return 0, fmt.Errorf("match snapshot: analysis %d names move %d, absent: %w", ma.ID, ma.MoveID, storage.ErrInvalid)
		}
		ma.ID, ma.MoveID = 0, moveID
		if _, err := ms.CreateMoveAnalysis(ctx, scope, &ma); err != nil {
			return 0, err
		}
	}
	if err := s.Stats().RefreshMatchStats(ctx, scope, []int64{matchID}); err != nil {
		return 0, err
	}
	return matchID, nil
}
