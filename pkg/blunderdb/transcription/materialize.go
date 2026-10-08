package transcription

import (
	"context"
	"errors"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// RefusedAction answers a document the rule machine does not accept: Rank is
// the index, from 0, of the first Action refused, or -1 when the header itself
// is. Nothing was written.
type RefusedAction struct {
	Rank    int
	Refusal *transcript.Refusal
}

func (e *RefusedAction) Error() string {
	if e.Rank < 0 {
		return fmt.Sprintf("transcription: header refused: %s", e.Refusal)
	}
	return fmt.Sprintf("transcription: action %d refused: %s", e.Rank, e.Refusal)
}

// Unwrap makes a refused document a storage.ErrInvalid for every caller that
// maps storage errors.
func (e *RefusedAction) Unwrap() error { return storage.ErrInvalid }

// ErrorDetails is what an API error envelope carries besides the message.
func (e *RefusedAction) ErrorDetails() map[string]any {
	return map[string]any{"rank": e.Rank, "kind": e.Refusal.Kind, "detail": e.Refusal.Detail}
}

// Materialize writes the Match a document of Actions plays, in one call and
// one transaction, or nothing: the rule machine is applied as the Arbiter
// applies it, so the first Action it refuses stops everything, where a draft
// signals and goes on (ADR-0044). The Match is a Transcription's, given the
// same hashes as Finish gives it: the dice came with the document, nothing was
// played here, so no origin and no seed. The decision durations the Actions
// carry go onto their Moves; absent, they stay unknown.
func (s *Service) Materialize(ctx context.Context, scope string, header transcript.Header, actions []transcript.Action) (*SaveResult, error) {
	header = s.draftHeader(ctx, scope, header)
	m, err := transcript.NewMachine(header, nil)
	if err != nil {
		var r *transcript.Refusal
		if errors.As(err, &r) {
			return nil, &RefusedAction{Rank: -1, Refusal: r}
		}
		return nil, err
	}
	infos := make([]transcript.ActionInfo, 0, len(actions))
	for i, a := range actions {
		var info transcript.ActionInfo
		if m, info, err = m.Apply(a); err != nil {
			var r *transcript.Refusal
			if errors.As(err, &r) {
				return nil, &RefusedAction{Rank: i, Refusal: r}
			}
			return nil, err
		}
		infos = append(infos, info)
	}

	// A game begun and not played in holds no Move: it is left out.
	games := m.Games()
	for len(games) > 0 {
		if last := games[len(games)-1]; last.Finished || last.First >= 0 {
			break
		}
		games = games[:len(games)-1]
	}
	// The Repères give the durations, as a Replay of the same draft would.
	transcript.TimeActions(actions, infos)
	parts := transcript.BuildPlayed(header, games, infos, false)
	if len(parts.Games) == 0 {
		return nil, fmt.Errorf("transcription: nothing to materialise, no Action opens a game: %w", storage.ErrInvalid)
	}
	matchHeader := *parts.Match
	matchHeader.MatchHash, matchHeader.CanonicalHash = MatchHashes(parts)

	res, err := s.writeNewMatch(ctx, scope, MatchGraph(parts), matchHeader)
	if err != nil {
		return nil, err
	}
	out := &SaveResult{MatchID: res.MatchID, Games: len(parts.Games), Positions: res.SavedPositions}
	for _, moves := range parts.Moves {
		out.Moves += len(moves)
	}
	if n, err := s.toAnalyze(ctx, scope, res.MatchID); err == nil {
		out.ToAnalyze = n
	}
	return out, nil
}

// writeNewMatch is Materialize's transaction: the graph hash-less and the
// hashes stated afterwards, as writeMatch does and for the same reason.
func (s *Service) writeNewMatch(ctx context.Context, scope string, graph *ingest.MatchGraph, header domain.Match) (ingest.WriteResult, error) {
	var res ingest.WriteResult
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return res, err
	}
	fail := func(err error) (ingest.WriteResult, error) {
		_ = tx.Rollback()
		return res, err
	}
	if res, err = ingest.WriteMatch(ctx, tx, scope, graph, nil); err != nil {
		return fail(err)
	}
	header.ID = res.MatchID
	if err := tx.Matches().ReplaceHeader(ctx, scope, res.MatchID, &header); err != nil {
		return fail(err)
	}
	if err := attachTournament(ctx, tx, scope, res.MatchID, header.TournamentID); err != nil {
		return fail(err)
	}
	return res, tx.Commit()
}
