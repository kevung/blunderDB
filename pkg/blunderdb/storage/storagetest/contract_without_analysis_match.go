// Contract case for AnalysisStore.WithoutAnalysisIDsInMatch. The table that
// runs it lives in contract.go.
package storagetest

import (
	"context"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testWithoutAnalysisIDsInMatch: the gaps of one match are the positions its
// moves reach that carry no analysis, each once even when two moves reach it,
// ascending; a position shared with another match counts, one only another
// match reaches does not, and an analysed one never does.
func testWithoutAnalysisIDsInMatch(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ids := make([]int64, 4)
	for i := range ids {
		p := provenancePos(i + 1)
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save: %v", err)
		}
		ids[i] = id
	}
	// m1 reaches ids[0] twice and ids[1]; m2 reaches ids[1] and ids[2];
	// ids[3] belongs to no match.
	m1, _ := matchWithMoves(t, s, [2]int64{ids[0], ids[0]})
	g := domain.Game{MatchID: m1, GameNumber: 2}
	gameID, err := s.Matches().CreateGame(ctx, "", &g)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	mv := domain.Move{GameID: gameID, MoveNumber: 1, MoveType: "checker", PositionID: ids[1],
		Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5"}
	if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
		t.Fatalf("CreateMove: %v", err)
	}
	m2, _ := matchWithMoves(t, s, [2]int64{ids[1], ids[2]})

	in := func(matchID int64) []int64 {
		t.Helper()
		got, err := s.Analyses().WithoutAnalysisIDsInMatch(ctx, "", matchID)
		if err != nil {
			t.Fatalf("WithoutAnalysisIDsInMatch(%d): %v", matchID, err)
		}
		return got
	}
	if got, want := in(m1), []int64{ids[0], ids[1]}; !slices.Equal(got, want) {
		t.Errorf("match 1: %v, want %v", got, want)
	}
	if got, want := in(m2), []int64{ids[1], ids[2]}; !slices.Equal(got, want) {
		t.Errorf("match 2: %v, want %v", got, want)
	}

	// Filling a gap closes it in every match that reaches it.
	if err := s.Analyses().Save(ctx, "", ids[1], evaluatedAnalysis()); err != nil {
		t.Fatal(err)
	}
	if got, want := in(m1), []int64{ids[0]}; !slices.Equal(got, want) {
		t.Errorf("match 1 after filling %d: %v, want %v", ids[1], got, want)
	}
	if got, want := in(m2), []int64{ids[2]}; !slices.Equal(got, want) {
		t.Errorf("match 2 after filling %d: %v, want %v", ids[1], got, want)
	}
	if got := in(999999); len(got) != 0 {
		t.Errorf("unknown match: %v, want none", got)
	}
}
