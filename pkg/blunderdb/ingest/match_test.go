package ingest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func sampleGraph() *MatchGraph {
	p := domain.InitializePosition()
	return &MatchGraph{
		Match: domain.Match{
			Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7,
			MatchHash: "xg-hash-1", CanonicalHash: "canon-1",
		},
		Games: []GameGraph{{
			Game: domain.Game{GameNumber: 1, Winner: 1, PointsWon: 2},
			Moves: []MoveGraph{{
				Move:     domain.Move{MoveNumber: 1, MoveType: "checker", Player: 1, Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5"},
				Position: &p,
			}},
		}},
	}
}

func TestWriteMatch(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	tx, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	res, err := WriteMatch(ctx, tx, "", sampleGraph(), nil)
	if err != nil {
		t.Fatalf("WriteMatch: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if res.Skipped {
		t.Fatal("first write should not be skipped")
	}
	if res.MatchID == 0 || res.SavedPositions != 1 {
		t.Fatalf("res = %+v, want MatchID!=0 SavedPositions=1", res)
	}

	counts, err := s.Metadata().Counts(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if counts.Matches != 1 || counts.Games != 1 || counts.Moves != 1 || counts.Positions != 1 {
		t.Fatalf("counts = %+v, want 1/1/1/1", counts)
	}
}

func TestWriteMatchDedup(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	write := func() (WriteResult, error) {
		tx, err := s.BeginTx(ctx)
		if err != nil {
			return WriteResult{}, err
		}
		res, err := WriteMatch(ctx, tx, "", sampleGraph(), nil)
		if err != nil {
			tx.Rollback()
			return res, err
		}
		return res, tx.Commit()
	}

	if _, err := write(); err != nil {
		t.Fatalf("first write: %v", err)
	}
	res2, err := write()
	if err != nil {
		t.Fatalf("second write: %v", err)
	}
	if !res2.Skipped {
		t.Fatal("second write of same match should be skipped (duplicate)")
	}

	counts, _ := s.Metadata().Counts(ctx, "")
	if counts.Matches != 1 {
		t.Fatalf("matches after dup import = %d, want 1", counts.Matches)
	}
}
