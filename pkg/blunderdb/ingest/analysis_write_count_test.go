package ingest

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// openCountingStore opens a file-backed SQLite store whose analysis table
// counts its own inserts and updates in analysis_writes, so a test can tell
// "nothing was written" from "the same bytes were written again".
func openCountingStore(t *testing.T) (*sqlite.Storage, func() int) {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "count.db")
	db, err := sql.Open("sqlite", sqlite.DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	sqlite.ConfigurePool(db, path)
	if err := sqlite.Bootstrap(ctx, db); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`CREATE TABLE analysis_writes (n INTEGER NOT NULL)`,
		`INSERT INTO analysis_writes VALUES (0)`,
		`CREATE TRIGGER count_analysis_insert AFTER INSERT ON analysis BEGIN UPDATE analysis_writes SET n = n + 1; END`,
		`CREATE TRIGGER count_analysis_update AFTER UPDATE ON analysis BEGIN UPDATE analysis_writes SET n = n + 1; END`,
	} {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	count := func() int {
		var n int
		if err := db.QueryRowContext(ctx, `SELECT n FROM analysis_writes`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	return sqlite.New(db), count
}

// TestEnrichOfAnIdenticalMatchWritesNoAnalysis: re-importing through the
// cross-format path a match whose analyses are already all stored merges each
// position into an identical result, and writes no analysis row.
func TestEnrichOfAnIdenticalMatchWritesNoAnalysis(t *testing.T) {
	s, writes := openCountingStore(t)
	xg, err := MapXG("../../../testdata/charlot1-charlot2_7p_2025-11-08-2305.xg")
	if err != nil {
		t.Fatalf("MapXG: %v", err)
	}
	sgf := func() *MatchGraph {
		g, err := MapGnuBG("../../../testdata/charlot1-charlot2_7p_2025-11-08-2305.sgf")
		if err != nil {
			t.Fatalf("MapGnuBG: %v", err)
		}
		return g
	}
	writeGraph(t, s, xg)
	if res := writeGraph(t, s, sgf()); !res.Enriched {
		t.Fatal("first SGF import did not enrich")
	}
	before := writes()
	if before == 0 {
		t.Fatal("the imports wrote no analysis at all")
	}
	if res := writeGraph(t, s, sgf()); !res.Enriched {
		t.Fatal("second SGF import did not enrich")
	}
	if got := writes() - before; got != 0 {
		t.Errorf("re-enriching an identical match wrote %d analysis rows, want 0", got)
	}
}

// TestFragmentsOfOnePositionAreWrittenOnce: a checker fragment and a cube
// fragment for one position are merged in memory and stored in one write,
// both halves present.
func TestFragmentsOfOnePositionAreWrittenOnce(t *testing.T) {
	ctx := context.Background()
	s, writes := openCountingStore(t)
	g, err := MapXG("../../../testdata/charlot1-charlot2_7p_2025-11-08-2305.xg")
	if err != nil {
		t.Fatalf("MapXG: %v", err)
	}
	pos := g.Games[0].Moves[0].Position
	checker := &domain.PositionAnalysis{
		AnalysisType: "CheckerMove",
		PlayedMove:   "13/11 24/23",
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "13/11 24/23", Equity: -0.01},
			{Move: "24/21 13/11", Equity: -0.05},
		}},
	}
	cube := &domain.PositionAnalysis{
		AnalysisType:         "DoublingCube",
		PlayedCubeAction:     "No Double",
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisEngine: "XG", BestCubeAction: "No Double"},
	}

	tx, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	posID, err := savePositionWithAnalyses(ctx, tx, "", pos, nil, []*domain.PositionAnalysis{checker, nil, cube}, nil, domain.CommentOriginUnknown)
	if err != nil {
		t.Fatalf("savePositionWithAnalyses: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if got := writes(); got != 1 {
		t.Errorf("two fragments took %d analysis writes, want 1", got)
	}
	a, err := s.Analyses().Load(ctx, "", posID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if a.CheckerAnalysis == nil || len(a.CheckerAnalysis.Moves) != 2 {
		t.Errorf("checker half lost: %+v", a.CheckerAnalysis)
	} else if e := a.CheckerAnalysis.Moves[1].EquityError; e == nil || *e != 0.04 {
		t.Errorf("equity error of the second move = %v, want 0.04", e)
	}
	if a.DoublingCubeAnalysis == nil || a.DoublingCubeAnalysis.BestCubeAction != "No Double" {
		t.Errorf("cube half lost: %+v", a.DoublingCubeAnalysis)
	}
	if len(a.PlayedMoves) != 1 || len(a.PlayedCubeActions) != 1 {
		t.Errorf("played actions = %v / %v, want one of each", a.PlayedMoves, a.PlayedCubeActions)
	}
}

// BenchmarkReEnrichIdenticalMatch measures a cross-format re-import of a
// match whose analyses are all already stored: the path where every merge is
// a no-op.
func BenchmarkReEnrichIdenticalMatch(b *testing.B) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, filepath.Join(b.TempDir(), "bench.db"), nil)
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	write := func(g *MatchGraph) {
		tx, err := s.BeginTx(ctx)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := WriteMatch(ctx, tx, "", g, nil); err != nil {
			b.Fatal(err)
		}
		if err := tx.Commit(); err != nil {
			b.Fatal(err)
		}
	}
	xg, err := MapXG("../../../testdata/charlot1-charlot2_7p_2025-11-08-2305.xg")
	if err != nil {
		b.Fatal(err)
	}
	write(xg)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		g, err := MapGnuBG("../../../testdata/charlot1-charlot2_7p_2025-11-08-2305.sgf")
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		write(g)
	}
}
