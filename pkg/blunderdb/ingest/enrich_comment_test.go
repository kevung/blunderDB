package ingest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// commentedGraph is sampleGraph with a comment on its single position and the
// given format-specific match hash (canonical stays "canon-1" so a second import
// with a different match hash enriches rather than duplicates).
func commentedGraph(matchHash, comment string) *MatchGraph {
	g := sampleGraph()
	g.Match.MatchHash = matchHash
	g.Games[0].Moves[0].Comments = []string{comment}
	return g
}

// TestEnrichDoesNotDuplicateComments reproduces #108: importing the same match in
// two formats (canonical duplicate → enrich) must not add the shared position's
// comment twice.
func TestEnrichDoesNotDuplicateComments(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	defer s.Close()

	res1 := writeGraph(t, s, commentedGraph("xg-hash-1", "nice cube"))
	if res1.Enriched {
		t.Fatal("first import should not enrich")
	}
	res2 := writeGraph(t, s, commentedGraph("gnu-hash-2", "nice cube"))
	if !res2.Enriched {
		t.Fatalf("second (cross-format) import should enrich, got %+v", res2)
	}
	if res2.MatchID != res1.MatchID {
		t.Fatalf("enrich created a new match: %d vs %d", res2.MatchID, res1.MatchID)
	}

	// The position must carry the comment exactly once.
	var texts []string
	for c, err := range s.Comments().ByPosition(ctx, "", 1) {
		if err != nil {
			t.Fatalf("ByPosition: %v", err)
		}
		texts = append(texts, c.Text)
	}
	if len(texts) != 1 {
		t.Fatalf("position has %d comments %v, want 1 (no duplicate on enrich)", len(texts), texts)
	}
}
