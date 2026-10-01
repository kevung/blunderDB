package transcription

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/events"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// SaveResult is what Finish reports: the Match written and the outline of
// what went into it.
type SaveResult struct {
	MatchID int64 `json:"matchId"`
	// Replaced: the draft was opened from this Match and finishing replaced
	// it in place, keeping its id.
	Replaced  bool `json:"replaced"`
	Games     int  `json:"games"`
	Moves     int  `json:"moves"`
	Positions int  `json:"positions"`
	// ToAnalyze counts the Match's positions without an analysis, what the
	// canonical analysis has to run on (ADR-0045 §8).
	ToAnalyze int `json:"toAnalyze"`
	// Inconsistent: the Match was written with Inconsistencies kept and
	// marked (ADR-0044).
	Inconsistent bool `json:"inconsistent"`
}

// Finish materialises the Match and releases the draft (ADR-0045 §2), in one
// transaction whose first statement checks the revision: the Match, its
// tournament link and the deletion of the draft land whole or not at all. A
// draft opened from a Match replaces it in place and keeps its id.
func (s *Service) Finish(ctx context.Context, scope string, id int64, exp Expect) (*SaveResult, error) {
	ss, err := s.acquire(ctx, scope, id, exp.Session)
	if err != nil {
		return nil, err
	}
	defer ss.mu.Unlock()
	if exp.Revision != 0 && exp.Revision != ss.rev {
		return nil, staleSession(id, ss)
	}

	parts := transcript.Build(ss.ed.Doc)
	if len(parts.Games) == 0 {
		return nil, fmt.Errorf("transcription %d: nothing to finish yet: %w", id, storage.ErrInvalid)
	}
	graph := transcriptGraph(parts)
	if m := ss.ed.Doc.Header.MatchID; m != nil && *m != 0 {
		graph.ReplaceMatchID = *m
	}
	header := *parts.Match
	header.MatchHash, header.CanonicalHash = transcriptMatchHashes(parts)

	res, err := s.writeMatch(ctx, scope, id, ss.rev, graph, header)
	if errors.Is(err, storage.ErrConflict) {
		// Another writer moved the row: the session reloads from it, or the
		// next Finish would compare against the revision that just lost.
		return nil, s.failed(ctx, scope, id, ss, err)
	}
	if err != nil {
		return nil, err
	}
	s.drop(scope, id, ss)
	s.published(scope, id, events.Event{Removed: true, MatchID: res.MatchID})

	out := &SaveResult{
		MatchID:      res.MatchID,
		Replaced:     res.Replaced,
		Games:        len(parts.Games),
		Positions:    res.SavedPositions,
		Inconsistent: parts.Inconsistent,
	}
	for _, moves := range parts.Moves {
		out.Moves += len(moves)
	}
	if n, err := s.toAnalyze(ctx, scope, res.MatchID); err == nil {
		out.ToAnalyze = n
	}
	return out, nil
}

// writeMatch is Finish's transaction: the revision checked, ingest.WriteMatch,
// the content hashes stated afterwards, and the draft row deleted.
//
// Afterwards on purpose: WriteMatch dedups on the hashes, which for a first
// write would enrich a match the library already holds from XG and hand it to
// the draft. So the graph travels hash-less and ReplaceHeader stamps them in
// the same transaction, for a future IMPORT to dedup against. A replacement
// is not snapshotted to the trash (ADR-0045 §3).
func (s *Service) writeMatch(ctx context.Context, scope string, id, rev int64, graph *ingest.MatchGraph, header domain.Match) (ingest.WriteResult, error) {
	var res ingest.WriteResult
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return res, err
	}
	fail := func(err error) (ingest.WriteResult, error) {
		_ = tx.Rollback()
		return res, err
	}
	if _, err := tx.Transcriptions().Touch(ctx, scope, id, rev); err != nil {
		return fail(err)
	}
	if graph.ReplaceMatchID != 0 {
		old, err := tx.Matches().Get(ctx, scope, graph.ReplaceMatchID)
		if err != nil {
			return fail(err)
		}
		// An imported Match keeps the hashes of its source file, as it keeps
		// its file_path and import batch: they are what makes the same file,
		// imported again, a duplicate of this Match rather than a second copy.
		if IsImported(old) {
			header.MatchHash, header.CanonicalHash = old.MatchHash, old.CanonicalHash
		}
	}
	res, err = ingest.WriteMatch(ctx, tx, scope, graph, nil)
	if err != nil {
		return fail(err)
	}
	header.ID = res.MatchID
	if err := tx.Matches().ReplaceHeader(ctx, scope, res.MatchID, &header); err != nil {
		return fail(err)
	}
	if err := attachTournament(ctx, tx, scope, res.MatchID, header.TournamentID); err != nil {
		return fail(err)
	}
	if err := tx.Transcriptions().Delete(ctx, scope, id); err != nil {
		return fail(err)
	}
	return res, tx.Commit()
}

// toAnalyze counts the Match's positions that carry no analysis.
func (s *Service) toAnalyze(ctx context.Context, scope string, matchID int64) (int, error) {
	seen := map[int64]bool{}
	var ids []int64
	for mp, err := range s.store.Matches().MovePositions(ctx, scope, matchID) {
		if err != nil {
			return 0, err
		}
		if id := mp.Position.ID; id != 0 && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return 0, nil
	}
	have, err := s.store.Analyses().LoadMany(ctx, scope, ids)
	if err != nil {
		return 0, err
	}
	return len(ids) - len(have), nil
}

// attachTournament is the header's tournament made true of the Match, in the
// same transaction as the write.
//
// Here, not in the graph: ReplaceHeader does not touch tournament_id, which
// the tournament store owns with its sort order. A save with no tournament
// DETACHES, so clearing the field is not silently ignored.
func attachTournament(ctx context.Context, tx storage.Tx, scope string, matchID int64, tournamentID *int64) error {
	if tournamentID != nil && *tournamentID != 0 {
		return tx.Tournaments().AddMatch(ctx, scope, *tournamentID, matchID)
	}
	return tx.Tournaments().RemoveMatch(ctx, scope, matchID)
}

// Losses is what opening a draft from a Match cannot carry and finishing it
// may drop, counted before anything is written (ADR-0045 §2).
type Losses struct {
	MatchID int64 `json:"matchId"`
	// Imported: the Match came from a file (XG, GnuBG, BGF…), not from a
	// transcription. A transcribed Match loses nothing and counts zero.
	Imported bool `json:"imported"`
	// Analyses and Comments are held by the Match's positions and moves; a
	// .mat carries neither.
	Analyses int `json:"analyses"`
	Comments int `json:"comments"`
	// DraftID is the draft already open on this Match, 0 when there is none:
	// reopening it loses nothing more, so no warning is due.
	DraftID int64 `json:"draftId"`
	// Lossy says whether opening a draft warrants the warning.
	Lossy bool `json:"lossy"`
}

// IsImported reports whether a Match came from a file: only an importer sets
// a source file or an import batch.
func IsImported(m *domain.Match) bool {
	return m.FilePath != "" || m.ImportBatchID != 0
}

// MatchLosses counts what EditMatch would leave behind for this Match. Writes
// nothing.
func (s *Service) MatchLosses(ctx context.Context, scope string, matchID int64) (*Losses, error) {
	m, err := s.store.Matches().Get(ctx, scope, matchID)
	if err != nil {
		return nil, fmt.Errorf("match %d: %w", matchID, err)
	}
	out := &Losses{MatchID: matchID, Imported: IsImported(m)}
	if out.DraftID, err = s.draftOf(ctx, scope, matchID); err != nil {
		return nil, err
	}
	if out.Imported {
		out.Analyses, out.Comments, err = s.store.Transcriptions().Annotations(ctx, scope, matchID)
		if err != nil {
			return nil, fmt.Errorf("match %d: counting what a transcription drops: %w", matchID, err)
		}
	}
	out.Lossy = out.DraftID == 0 && out.Imported && out.Analyses+out.Comments > 0
	return out, nil
}

// draftOf is the draft opened on matchID, 0 when none.
func (s *Service) draftOf(ctx context.Context, scope string, matchID int64) (int64, error) {
	for row, err := range s.store.Transcriptions().List(ctx, scope) {
		if err != nil {
			return 0, err
		}
		if row.MatchID == matchID {
			return row.ID, nil
		}
	}
	return 0, nil
}

// EditMatch opens a draft on an existing Match: the draft already open on it
// when there is one (one draft per Match), otherwise a new one replayed from
// the Match's .mat rendering, naming the Match so that Finish replaces it.
// The losses are counted before the draft is written; the caller shows them,
// this method does not refuse a lossy Match.
func (s *Service) EditMatch(ctx context.Context, scope string, matchID int64) (*State, *Losses, error) {
	s.editMu.Lock()
	defer s.editMu.Unlock()

	losses, err := s.MatchLosses(ctx, scope, matchID)
	if err != nil {
		return nil, nil, err
	}
	if losses.DraftID != 0 {
		st, err := s.Open(ctx, scope, losses.DraftID)
		return st, losses, err
	}

	m, games, moves, err := ingest.ReadMatchForMAT(ctx, s.store, scope, matchID)
	if err != nil {
		return nil, nil, fmt.Errorf("match %d: %w", matchID, err)
	}
	doc, err := transcript.FromMAT(ingest.RenderMAT(m, games, moves))
	if err != nil {
		return nil, nil, fmt.Errorf("match %d: %w", matchID, err)
	}
	id := matchID
	doc.Header.MatchID = &id
	// The .mat does not carry the tournament, and finishing with none would
	// detach the Match from it.
	doc.Header.TournamentID = m.TournamentID

	st, err := s.insert(ctx, scope, doc)
	return st, losses, err
}

// transcriptGraph turns what the transcript package returns into the graph
// ingest.WriteMatch writes. It carries no analysis and no comment: the
// analysis is the batch's job afterwards (ADR-0045 §8), and a transcription
// has no notes to attach. The transcript's own ids are not database ids and
// are dropped; so are the hashes (see writeMatch).
func transcriptGraph(parts transcript.Parts) *ingest.MatchGraph {
	g := &ingest.MatchGraph{Match: *parts.Match}
	g.Match.MatchHash, g.Match.CanonicalHash = "", ""

	for _, game := range parts.Games {
		moves := parts.Moves[game.ID]
		positions := parts.Positions[game.ID]

		gg := ingest.GameGraph{Game: *game}
		gg.Game.ID = 0
		gg.Game.MatchID = 0
		for i, mv := range moves {
			mg := ingest.MoveGraph{Move: *mv}
			mg.Move.GameID = 0
			if i < len(positions) {
				pos := positions[i]
				mg.Position = &pos
			}
			gg.Moves = append(gg.Moves, mg)
		}
		g.Games = append(g.Games, gg)
	}
	return g
}

// transcriptMatchHashes are the two content hashes of a transcribed match.
//
// The canonical one is the SAME scheme as the importers
// (computeCanonicalMatchHashFromXG and twins), so a later XG/GnuBG import of
// the same match enriches instead of duplicating. The format-specific one
// hashes the plays as typed and changes whenever the document does.
func transcriptMatchHashes(parts transcript.Parts) (matchHash, canonicalHash string) {
	m := parts.Match

	var b strings.Builder
	p1 := strings.TrimSpace(strings.ToLower(m.Player1Name))
	p2 := strings.TrimSpace(strings.ToLower(m.Player2Name))
	fmt.Fprintf(&b, "transcript1:%s|%s|%d|", p1, p2, m.MatchLength)
	for gi, game := range parts.Games {
		// The winner is hashed as a side (0/1/-1), like the document states it:
		// the hash follows the transcription, not the storage encoding.
		fmt.Fprintf(&b, "g%d:%d,%d,%d,%d|", gi,
			game.InitialScore[0], game.InitialScore[1], domain.WinnerSide(game.Winner), game.PointsWon)
		for mi, mv := range parts.Moves[game.ID] {
			fmt.Fprintf(&b, "m%d:%s,%d,d%d%d,p%s|", mi, mv.MoveType, mv.Player,
				mv.Dice[0], mv.Dice[1], mv.CheckerMove+mv.CubeAction)
		}
	}
	matchHash = sha256Hex(b.String())

	var c strings.Builder
	if p1 > p2 {
		p1, p2 = p2, p1
	}
	fmt.Fprintf(&c, "canonical2:%s|%s|%d|%d|", p1, p2, m.MatchLength, len(parts.Games))
	for gi, game := range parts.Games {
		fmt.Fprintf(&c, "g%d|", gi)
		dice := 0
		for _, mv := range parts.Moves[game.ID] {
			if dice >= canonicalDicePerGame {
				break
			}
			if mv.MoveType != "checker" {
				continue
			}
			d1, d2 := mv.Dice[0], mv.Dice[1]
			if d1 > d2 {
				d1, d2 = d2, d1
			}
			fmt.Fprintf(&c, "d%d%d|", d1, d2)
			dice++
		}
	}
	return matchHash, sha256Hex(c.String())
}

// canonicalDicePerGame mirrors ingest's maxCanonicalDicePerGame, which is
// unexported there. The two must state the same number or a transcribed match
// stops hashing like its imported twin — which is the whole point of the
// canonical hash.
const canonicalDicePerGame = 10

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
