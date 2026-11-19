package ingest

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

var updateMergedViewGolden = flag.Bool("update-merged-view", false, "rewrite testdata/merged_view.golden")

const mergedViewGolden = "../../../testdata/merged_view.golden"

// mergedViewDigests imports the XG and the GNUbg file of one match in the
// given order, each twice, and gives one line per stored analysis: its
// position, its XGID and a digest of every field a reader of the merged view
// reads — the analysis as JSON without the per-source evaluations and the
// dates.
func mergedViewDigests(t *testing.T, order []string) []string {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "merged.db")
	db, err := sql.Open("sqlite", sqlite.DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	sqlite.ConfigurePool(db, path)
	if err := sqlite.Bootstrap(ctx, db); err != nil {
		t.Fatal(err)
	}
	s := sqlite.New(db)
	for _, file := range append(append([]string(nil), order...), order...) {
		var g *MatchGraph
		var err error
		if strings.HasSuffix(file, ".xg") {
			g, err = MapXG(file)
		} else {
			g, err = MapGnuBG(file)
		}
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		writeGraph(t, s, g)
	}
	rows, err := db.QueryContext(ctx, `SELECT position_id, data FROM analysis ORDER BY position_id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var id int64
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			t.Fatal(err)
		}
		a, err := engine.DecodeAnalysisFromStorage(data)
		if err != nil {
			t.Fatalf("position %d: %v", id, err)
		}
		raw, err := json.Marshal(&a)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]any
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatal(err)
		}
		for _, k := range []string{"allCheckerAnalyses", "creationDate", "lastModifiedDate"} {
			delete(fields, k)
		}
		view, err := json.Marshal(fields)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(view)
		lines = append(lines, fmt.Sprintf("%d %s %s", id, a.XGID, hex.EncodeToString(sum[:8])))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return lines
}

// TestMergedViewUnchangedBySources: the merged view ("the deepest evaluation
// wins") of a match imported from XG and from GNUbg, in both orders and
// re-imported, does not depend on the sources: they ride beside the view and
// never feed it, so the golden digests leave out allCheckerAnalyses.
func TestMergedViewUnchangedBySources(t *testing.T) {
	xg := "../../../testdata/charlot1-charlot2_7p_2025-11-08-2305.xg"
	sgf := "../../../testdata/charlot1-charlot2_7p_2025-11-08-2305.sgf"
	var got []string
	for _, order := range [][]string{{xg, sgf}, {sgf, xg}} {
		got = append(got, "# "+filepath.Ext(order[0])+" first")
		got = append(got, mergedViewDigests(t, order)...)
	}
	text := strings.Join(got, "\n") + "\n"
	if *updateMergedViewGolden {
		if err := os.WriteFile(mergedViewGolden, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(mergedViewGolden)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) == text {
		return
	}
	wantLines := strings.Split(string(want), "\n")
	gotLines := strings.Split(text, "\n")
	for i := 0; i < len(wantLines) || i < len(gotLines); i++ {
		var w, g string
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if w != g {
			t.Fatalf("merged view differs at line %d:\n got  %q\n want %q", i+1, g, w)
		}
	}
}

// engineGraph is sampleGraph with one checker analysis of its single
// position by one engine, under a hash of its own: the same match exported
// by another program.
func engineGraph(hash string, moves []domain.CheckerMove) *MatchGraph {
	g := sampleGraph()
	g.Match.MatchHash = hash
	g.Games[0].Moves[0].Analyses = []*domain.PositionAnalysis{{
		AnalysisType:    "CheckerMove",
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: slices.Clone(moves)},
	}}
	return g
}

// A rollout and a 2-ply evaluation of the same candidates, in either order:
// the merged view is the one the merge rule gives without sources (the
// rollout wins every move it evaluated), and each source keeps its own moves.
func TestRolloutAgainstTwoPlyKeepsBothSources(t *testing.T) {
	ctx := context.Background()
	xg := []domain.CheckerMove{
		{Move: "8/5 6/5", AnalysisEngine: "XG", AnalysisDepth: "Rollout", Equity: 0.30},
		{Move: "24/23 13/10", AnalysisEngine: "XG", AnalysisDepth: "Rollout", Equity: 0.20},
	}
	gnu := []domain.CheckerMove{
		{Move: "24/23 13/10", AnalysisEngine: "GNUbg", AnalysisDepth: "2-ply", Equity: 0.35},
		{Move: "8/5 6/5", AnalysisEngine: "GNUbg", AnalysisDepth: "2-ply", Equity: 0.25},
		{Move: "13/10 6/5", AnalysisEngine: "GNUbg", AnalysisDepth: "2-ply", Equity: 0.10},
	}
	want := mergeCheckerMoves(xg, gnu)
	var views [][]domain.CheckerMove
	for _, order := range [][2][]domain.CheckerMove{{xg, gnu}, {gnu, xg}} {
		s, err := sqlite.Open(ctx, ":memory:", nil)
		if err != nil {
			t.Fatal(err)
		}
		writeGraph(t, s, engineGraph("first", order[0]))
		writeGraph(t, s, engineGraph("second", order[1]))
		a := storedAnalysis(t, s)
		s.Close()

		view := a.CheckerAnalysis.Moves
		views = append(views, view)
		if len(view) != len(want) {
			t.Fatalf("merged view = %+v, want %+v", view, want)
		}
		for i := range want {
			if view[i].Move != want[i].Move || view[i].AnalysisDepth != want[i].AnalysisDepth ||
				engine.RoundToMillipoint(view[i].Equity) != engine.RoundToMillipoint(want[i].Equity) {
				t.Errorf("merged view[%d] = %+v, want %+v", i, view[i], want[i])
			}
		}
		srcs := a.AllCheckerAnalyses
		if len(srcs) != 2 || srcs[0].AnalysisDepth != "Rollout" || len(srcs[0].Moves) != 2 ||
			srcs[1].AnalysisEngine != "GNUbg" || len(srcs[1].Moves) != 3 {
			t.Fatalf("sources = %+v, want the XG rollout and the GNUbg 2-ply", srcs)
		}
		for _, m := range srcs[1].Moves {
			if m.AnalysisDepth != "2-ply" {
				t.Errorf("GNUbg source holds %+v, want its own 2-ply moves", m)
			}
		}
	}
	if !reflect.DeepEqual(views[0], views[1]) {
		t.Errorf("merged view depends on the order:\n %+v\n %+v", views[0], views[1])
	}
}
