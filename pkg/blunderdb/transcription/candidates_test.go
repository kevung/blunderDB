package transcription

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// rollPosition is the opening position with dice d1-d2 on roll.
func rollPosition(t *testing.T, d1, d2 int) domain.Position {
	t.Helper()
	doc := transcript.Document{Header: transcript.Header{MatchLength: 7}}
	doc.Entry = &transcript.Entry{Side: 0, Dice: [2]int{d1, d2}}
	pos := transcript.CandidatePosition(doc)
	if len(domain.LegalMoves(&pos)) < 3 {
		t.Fatalf("roll %d-%d: too few plays for the test", d1, d2)
	}
	return pos
}

func TestRankCandidatesJoinsByNotationAndKeepsEveryPlay(t *testing.T) {
	pos := rollPosition(t, 4, 2)
	plays := domain.LegalMoves(&pos)
	// The engine ranks the last two plays only, last first.
	n := len(plays)
	rank := func(context.Context, domain.Position) ([]domain.CheckerMove, error) {
		return []domain.CheckerMove{{Move: plays[n-1].Notation, Equity: 0.1}, {Move: "not a play"}, {Move: plays[n-2].Notation, Equity: 0.05}}, nil
	}
	got := RankCandidates(context.Background(), pos, rank)
	if got.Unranked || len(got.Candidates) != n {
		t.Fatalf("unranked=%v, %d candidates, want ranked and %d", got.Unranked, len(got.Candidates), n)
	}
	if c := got.Candidates[0]; c.Index != n-1 || c.Rank != 1 || c.Analysis == nil || c.Notation != plays[n-1].Notation {
		t.Fatalf("first = %+v, want play %d ranked 1", c, n-1)
	}
	if c := got.Candidates[1]; c.Index != n-2 || c.Rank != 2 {
		t.Fatalf("second = %+v, want play %d ranked 2", c, n-2)
	}
	for i, c := range got.Candidates[2:] {
		if c.Index != i || c.Rank != 0 || c.Analysis != nil {
			t.Fatalf("leftover %d = %+v, want play %d unranked in generator order", i, c, i)
		}
	}
}

func TestRankCandidatesUnrankedOnEngineFailure(t *testing.T) {
	pos := rollPosition(t, 6, 5)
	n := len(domain.LegalMoves(&pos))
	for name, rank := range map[string]Ranker{
		"error": func(context.Context, domain.Position) ([]domain.CheckerMove, error) {
			return nil, errors.New("beyond the MET")
		},
		"none": func(context.Context, domain.Position) ([]domain.CheckerMove, error) { return nil, nil },
		"nil":  nil,
	} {
		got := RankCandidates(context.Background(), pos, rank)
		if !got.Unranked || len(got.Candidates) != n {
			t.Fatalf("%s: unranked=%v, %d candidates, want unranked and %d", name, got.Unranked, len(got.Candidates), n)
		}
		for i, c := range got.Candidates {
			if c.Index != i || c.Rank != 0 {
				t.Fatalf("%s: candidate %d = %+v, want generator order, unranked", name, i, c)
			}
		}
	}
}
