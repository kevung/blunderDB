//go:build postgres

package postgres

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func writeGraphPG(t *testing.T, s storage.Storage, g *ingest.MatchGraph) ingest.WriteResult {
	t.Helper()
	ctx := context.Background()
	tx, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	res, err := ingest.WriteMatch(ctx, tx, "", g, nil)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("WriteMatch: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return res
}

// WithFreshPosition lets Merge skip reading what a position created in the
// transaction cannot have. A position that already exists must not be treated
// so: importing the same match from SGF after XG has to merge into the stored
// analyses (both engines kept), not overwrite them. The invariant is
// savePositionWithAnalyses passing the hint only when SaveCreated says created.
func TestPostgresEnrichmentMergesIntoStoredAnalyses(t *testing.T) {
	ctx := context.Background()
	s, err := Open(ctx, purgeTestDB(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	const base = "../../../../testdata/charlot1-charlot2_7p_2025-11-08-2305"
	xg, err := ingest.MapXG(base + ".xg")
	if err != nil {
		t.Fatal(err)
	}
	sgf, err := ingest.MapGnuBG(base + ".sgf")
	if err != nil {
		t.Fatal(err)
	}
	res1 := writeGraphPG(t, s, xg)
	res2 := writeGraphPG(t, s, sgf)
	if !res2.Enriched || res2.MatchID != res1.MatchID {
		t.Fatalf("second import: Enriched=%v match %d, want enrichment of %d", res2.Enriched, res2.MatchID, res1.MatchID)
	}

	both := 0
	for mv, err := range s.Matches().MovesByMatch(ctx, "", res1.MatchID) {
		if err != nil {
			t.Fatal(err)
		}
		a, err := s.Analyses().Load(ctx, "", mv.PositionID)
		if err != nil || a == nil || len(a.AllCubeAnalyses) < 2 {
			continue
		}
		engines := map[string]bool{}
		for _, c := range a.AllCubeAnalyses {
			engines[c.AnalysisEngine] = true
		}
		if len(engines) >= 2 {
			both++
		}
	}
	if both == 0 {
		t.Fatal("no stored cube analysis carries both engines after enrichment: the SGF import overwrote the XG analyses")
	}
}
