package transcription

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// Ranker evaluates a checker Position and returns EVERY legal play, best
// first: gammonnet.RankAllPlaysWithMET at 0-ply, pruning off, which the
// desktop (Database.TranscriptionCandidates) and the daemon
// (transcriptions.candidates) both wrap. An error, or no play recognised,
// leaves the candidates unranked.
type Ranker func(ctx context.Context, pos domain.Position) ([]domain.CheckerMove, error)

// Candidate is one legal play of the roll being entered.
type Candidate struct {
	// Index is the play's place in transcript.Candidates, the one
	// GestureSelectCandidate names.
	Index    int                  `json:"index"`
	Notation string               `json:"notation"`
	Steps    []domain.CheckerStep `json:"steps"`
	// Rank is 1 for the best play; 0 for a play the engine did not rank, which
	// comes after every ranked one.
	Rank int `json:"rank"`
	// Analysis is the play's 0-ply Evaluation (equity, equityError from the
	// best, chances), nil when unranked.
	Analysis *domain.CheckerMove `json:"analysis,omitempty"`
}

// Candidates is the roll being entered and its plays: best first when ranked,
// else in the generator's order, which is an order and not a ranking. An
// empty list is a dance.
type Candidates struct {
	Side       int         `json:"side"`
	Dice       [2]int      `json:"dice"`
	Candidates []Candidate `json:"candidates"`
	Unranked   bool        `json:"unranked"`
}

// Candidates lists the plays of the roll being entered in the draft's session,
// both dice typed, ranked by rank. The session is acquired like a gesture's
// (sessionID "" opens one) but nothing is written: the ranking runs outside
// the session's lock, so a slow engine never holds the draft's gestures.
func (s *Service) Candidates(ctx context.Context, scope string, id int64, sessionID string, rank Ranker) (*Candidates, error) {
	ss, err := s.acquire(ctx, scope, id, sessionID)
	if err != nil {
		return nil, err
	}
	pos := transcript.CandidatePosition(ss.ed.Doc)
	ss.mu.Unlock()
	if pos.Dice[0] == 0 || pos.Dice[1] == 0 {
		return nil, fmt.Errorf("%w: %w", storage.ErrInvalid, transcript.ErrNoDice)
	}
	return RankCandidates(ctx, pos, rank), nil
}

// RankCandidates lists the legal plays of pos and ranks them by rank. The
// join is exact: domain.LegalMoves dedups by resulting board and the engine
// names each play by that same list.
func RankCandidates(ctx context.Context, pos domain.Position, rank Ranker) *Candidates {
	plays := domain.LegalMoves(&pos)
	out := &Candidates{Side: pos.PlayerOnRoll, Dice: pos.Dice, Candidates: make([]Candidate, 0, len(plays))}
	if len(plays) == 0 {
		return out
	}
	if rank != nil {
		byNotation := make(map[string]int, len(plays))
		for i, p := range plays {
			if _, dup := byNotation[p.Notation]; !dup {
				byNotation[p.Notation] = i
			}
		}
		if moves, err := rank(ctx, pos); err == nil {
			for _, m := range moves {
				i, ok := byNotation[m.Move]
				if !ok {
					continue
				}
				out.Candidates = append(out.Candidates, Candidate{
					Index: i, Notation: plays[i].Notation, Steps: plays[i].Steps,
					Rank: len(out.Candidates) + 1, Analysis: &m,
				})
			}
			if len(out.Candidates) > 0 {
				// A play the engine left out still belongs to the roll: it
				// follows the ranked ones, unranked, so none is lost.
				ranked := make([]bool, len(plays))
				for _, c := range out.Candidates {
					ranked[c.Index] = true
				}
				for i, p := range plays {
					if !ranked[i] {
						out.Candidates = append(out.Candidates, Candidate{Index: i, Notation: p.Notation, Steps: p.Steps})
					}
				}
				return out
			}
		}
	}
	out.Candidates = out.Candidates[:0]
	for i, p := range plays {
		out.Candidates = append(out.Candidates, Candidate{Index: i, Notation: p.Notation, Steps: p.Steps})
	}
	out.Unranked = true
	return out
}
