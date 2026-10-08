package trash

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ErrMatchPresent refuses restoring a match the database holds again — it was
// re-imported since the delete: a restore would make it a duplicate.
var ErrMatchPresent = errors.New("trash: the match is in the database again")

// Match deletes a match after snapshotting it whole: its header, games,
// moves, move analyses, and every position its moves reached with their
// analyses and comments. The delete is MatchStore.DeleteCascade, so the
// positions nothing else holds are purged by the same retention rule as a
// plain delete; the snapshot is what lets them come back.
func Match(ctx context.Context, s storage.Stores, scope string, matchID int64) (int64, error) {
	var id int64
	err := inTx(ctx, s, func(s storage.Stores) error {
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
		return nil
	})
	if err != nil {
		return 0, err
	}
	return id, nil
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
	return p, nil
}

// restoreMatch rebuilds a match from its snapshot, in one transaction when the
// store offers one: a half-restored match would be worse than none.
//
// The positions are re-Saved first, so the moves point at whatever row now
// holds each one; a position's analysis and comments come back as
// restorePosition puts them. What the match was attached to — its tournament,
// its import batch, the drafts that produced it — is attached again only
// where it still exists.
func restoreMatch(ctx context.Context, s storage.Stores, scope string, entry *domain.TrashEntry) (int64, error) {
	var p domain.TrashMatchPayload
	if err := json.Unmarshal(entry.Payload, &p); err != nil {
		return 0, fmt.Errorf("trash entry %d: %w", entry.ID, err)
	}
	var id int64
	err := inTx(ctx, s, func(s storage.Stores) error {
		var err error
		id, err = rebuildMatch(ctx, s, scope, &p)
		return err
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

func rebuildMatch(ctx context.Context, s storage.Stores, scope string, p *domain.TrashMatchPayload) (int64, error) {
	ms := s.Matches()
	if p.Match.MatchHash != "" || p.Match.CanonicalHash != "" {
		other, found, err := ms.FindByHash(ctx, scope, p.Match.MatchHash, p.Match.CanonicalHash)
		if err != nil {
			return 0, err
		}
		if found {
			return 0, fmt.Errorf("%w (match %d)", ErrMatchPresent, other)
		}
	}

	positions := make(map[int64]int64, len(p.Positions))
	for _, pp := range p.Positions {
		newID, err := restorePositionPayload(ctx, s, scope, pp)
		if err != nil {
			return 0, err
		}
		positions[pp.Position.ID] = newID
	}

	m := p.Match
	m.ID = 0
	if m.TournamentID != nil {
		if _, err := s.Tournaments().Get(ctx, scope, *m.TournamentID); err != nil {
			if !errors.Is(err, storage.ErrNotFound) {
				return 0, err
			}
			m.TournamentID = nil
		}
	}
	if m.ImportBatchID != 0 {
		if _, err := s.ImportBatches().Load(ctx, scope, m.ImportBatchID); err != nil {
			if !errors.Is(err, storage.ErrNotFound) {
				return 0, err
			}
			m.ImportBatchID = 0
		}
	}
	matchID, err := ms.Save(ctx, scope, &m)
	if err != nil {
		return 0, err
	}
	if m.VideoSource != nil {
		if err := ms.SetVideoSource(ctx, scope, matchID, *m.VideoSource); err != nil {
			return 0, err
		}
	}
	if m.LastVisitedPosition != 0 {
		if err := ms.SetLastVisitedPosition(ctx, scope, matchID, m.LastVisitedPosition); err != nil {
			return 0, err
		}
	}

	games := make(map[int64]int64, len(p.Games))
	for _, g := range p.Games {
		old := g.ID
		g.ID, g.MatchID = 0, matchID
		newID, err := ms.CreateGame(ctx, scope, &g)
		if err != nil {
			return 0, err
		}
		games[old] = newID
	}
	moves := make(map[int64]int64, len(p.Moves))
	for _, mv := range p.Moves {
		gameID, ok := games[mv.GameID]
		if !ok {
			return 0, fmt.Errorf("trash: move %d names game %d, absent from the snapshot", mv.ID, mv.GameID)
		}
		old := mv.ID
		mv.ID, mv.GameID = 0, gameID
		if mv.PositionID != 0 {
			mv.PositionID = positions[mv.PositionID]
		}
		newID, err := ms.CreateMove(ctx, scope, &mv)
		if err != nil {
			return 0, err
		}
		moves[old] = newID
	}
	for _, ma := range p.MoveAnalyses {
		moveID, ok := moves[ma.MoveID]
		if !ok {
			return 0, fmt.Errorf("trash: analysis %d names move %d, absent from the snapshot", ma.ID, ma.MoveID)
		}
		ma.ID, ma.MoveID = 0, moveID
		if _, err := ms.CreateMoveAnalysis(ctx, scope, &ma); err != nil {
			return 0, err
		}
	}

	if len(p.Origin) > 0 {
		var o storage.MatchOrigin
		if err := json.Unmarshal(p.Origin, &o); err != nil {
			return 0, err
		}
		o.MatchID = matchID
		if err := s.Duels().SetOrigin(ctx, scope, &o); err != nil {
			return 0, err
		}
	}
	if m.TournamentID != nil {
		if err := placeInTournament(ctx, s, scope, *m.TournamentID, matchID, p.TournamentIndex); err != nil {
			return 0, err
		}
	}
	for _, tid := range p.TranscriptionIDs {
		t, err := s.Transcriptions().Get(ctx, scope, tid)
		if errors.Is(err, storage.ErrNotFound) {
			continue
		}
		if err != nil {
			return 0, err
		}
		// A draft that has produced another match since keeps that one.
		if t.MatchID != 0 {
			continue
		}
		t.MatchID = matchID
		if _, err := s.Transcriptions().Save(ctx, scope, t); err != nil {
			return 0, err
		}
	}
	return matchID, nil
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
func inTx(ctx context.Context, s storage.Stores, fn func(storage.Stores) error) error {
	b, ok := s.(txBeginner)
	if !ok {
		return fn(s)
	}
	tx, err := b.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
