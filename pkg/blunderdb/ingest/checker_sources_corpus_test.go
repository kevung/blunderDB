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
	"strings"
	"testing"

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
// re-imported, is what it was before the analyses kept their sources. The
// golden file was written by the code without the sources.
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
