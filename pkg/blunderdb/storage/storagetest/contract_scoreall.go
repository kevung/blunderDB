package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testScoreAllMoves: the loop over ScoreMoves scores the two scorable plays
// whatever the batching, reports the running total, and a second run finds
// nothing left to write.
func testScoreAllMoves(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	scoreFixture(t, s, "")
	var last int
	n, err := storage.ScoreAllMoves(ctx, s.Matches(), "", nil, func(total int) { last = total })
	if err != nil || n != 2 || last != 2 {
		t.Fatalf("ScoreAllMoves = %d (progress %d), %v; want 2", n, last, err)
	}
	if n, err := storage.ScoreAllMoves(ctx, s.Matches(), "", nil, nil); err != nil || n != 0 {
		t.Fatalf("second ScoreAllMoves = %d, %v; want 0", n, err)
	}
}
