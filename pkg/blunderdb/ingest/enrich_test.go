package ingest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// writeGraph runs MapXxx → WriteMatch in its own transaction and returns the result.
func writeGraph(t *testing.T, s storage.Storage, g *MatchGraph) WriteResult {
	t.Helper()
	ctx := context.Background()
	tx, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	res, err := WriteMatch(ctx, tx, "", g, nil)
	if err != nil {
		tx.Rollback()
		t.Fatalf("WriteMatch: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	return res
}

// hasEngines reports whether a stored analysis carries cube analyses from both
// the named engines (via AllCubeAnalyses) — the signature of a cross-format
// enrichment merge.
func hasBothCubeEngines(a *domain.PositionAnalysis, e1, e2 string) bool {
	if a == nil || len(a.AllCubeAnalyses) < 2 {
		return false
	}
	var has1, has2 bool
	for _, c := range a.AllCubeAnalyses {
		if c.AnalysisEngine == e1 {
			has1 = true
		}
		if c.AnalysisEngine == e2 {
			has2 = true
		}
	}
	return has1 && has2
}

// TestCrossFormatEnrichment imports the same match first from XG and then from
// GnuBG (SGF). The second import must NOT create a second match (canonical
// duplicate); instead it enriches the existing positions, so a doubling
// position ends up with both the XG and the GnuBG cube analysis.
func TestCrossFormatEnrichment(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	defer s.Close()

	xg, err := MapXG("../../../testdata/charlot1-charlot2_7p_2025-11-08-2305.xg")
	if err != nil {
		t.Fatalf("MapXG: %v", err)
	}
	sgf, err := MapGnuBG("../../../testdata/charlot1-charlot2_7p_2025-11-08-2305.sgf")
	if err != nil {
		t.Fatalf("MapGnuBG: %v", err)
	}
	if xg.Match.CanonicalHash != sgf.Match.CanonicalHash {
		t.Fatalf("fixtures are not the same match (canonical hashes differ)")
	}

	res1 := writeGraph(t, s, xg)
	if res1.Skipped || res1.Enriched {
		t.Fatalf("first import: Skipped=%v Enriched=%v, want both false", res1.Skipped, res1.Enriched)
	}
	afterXG, _ := s.Metadata().Counts(ctx, "")

	res2 := writeGraph(t, s, sgf)
	if !res2.Enriched {
		t.Fatalf("second (cross-format) import should be Enriched")
	}
	if res2.MatchID != res1.MatchID {
		t.Fatalf("enrichment reused match %d, want %d", res2.MatchID, res1.MatchID)
	}

	afterSGF, _ := s.Metadata().Counts(ctx, "")
	if afterSGF.Matches != 1 {
		t.Fatalf("matches after cross-format import = %d, want 1", afterSGF.Matches)
	}
	if afterSGF.Games != afterXG.Games || afterSGF.Moves != afterXG.Moves {
		t.Fatalf("enrichment created game/move rows: games %d→%d moves %d→%d",
			afterXG.Games, afterSGF.Games, afterXG.Moves, afterSGF.Moves)
	}

	// At least one position must now carry both engines' cube analysis. Drain
	// the position list before loading analyses: a per-row Load nested inside
	// the List iterator would grab a second :memory: connection (empty DB).
	var ids []int64
	for p, err := range s.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		ids = append(ids, p.ID)
	}
	merged := 0
	for _, id := range ids {
		a, err := s.Analyses().Load(ctx, "", id)
		if err != nil {
			continue
		}
		if hasBothCubeEngines(a, "XG", "GNUbg") {
			merged++
		}
	}
	if merged == 0 {
		t.Fatal("no position carries both XG and GNUbg cube analysis after enrichment")
	}
	t.Logf("cross-format enrichment merged both engines on %d position(s)", merged)
}
